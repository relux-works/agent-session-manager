package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
)

// Pair writers require a live exclusive hold: a forged, absent or released
// token refuses before anything durable is written. These tests pin the
// fail-closed boundary the replace-require-skip narrowing plant attacks
// (it admits exactly the zero token).

func writeHoldTestFile(t *testing.T, directory, name string, document []byte) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, document, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertHoldTestDirClean(t *testing.T, directory, filename string, original []byte) {
	t.Helper()
	after, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatal("refused pair write changed the configuration file")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "config.toml" {
			t.Fatalf("refused pair write left %q", entry.Name())
		}
	}
}

func TestReplaceDurablyRefusesZeroHold(t *testing.T) {
	directory := t.TempDir()
	original := []byte("original configuration\n")
	filename := writeHoldTestFile(t, directory, "config.toml", original)
	paths := resolvedPathsForStoreTest(t, filename, filepath.Join(directory, "state"))
	err := replaceDurably(hosttrust.HeldExclusive{}, osMigrationFileSystem{}, paths, filename+".bak.3.0.0", original, []byte("replacement\n"))
	if !errors.Is(err, hosttrust.ErrExclusiveHoldRequired) {
		t.Fatalf("replaceDurably(zero hold) err = %v, want %v", err, hosttrust.ErrExclusiveHoldRequired)
	}
	assertHoldTestDirClean(t, directory, filename, original)
}

func TestReplaceDurablyRefusesReleasedHold(t *testing.T) {
	fixture := setupV4Fixture(t)
	var released hosttrust.HeldExclusive
	if err := fixture.store.WithExclusiveHold(func(hold hosttrust.HeldExclusive) error {
		released = hold
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if released.IsZero() {
		t.Fatal("captured hold is zero, want a genuine released hold")
	}
	directory := t.TempDir()
	original := []byte("original configuration\n")
	filename := writeHoldTestFile(t, directory, "config.toml", original)
	paths := resolvedPathsForStoreTest(t, filename, filepath.Join(directory, "state"))
	err := replaceDurably(released, osMigrationFileSystem{}, paths, filename+".bak.3.0.0", original, []byte("replacement\n"))
	if !errors.Is(err, hosttrust.ErrExclusiveHoldRequired) {
		t.Fatalf("replaceDurably(released hold) err = %v, want %v", err, hosttrust.ErrExclusiveHoldRequired)
	}
	assertHoldTestDirClean(t, directory, filename, original)
}

func TestWriteTempReplaceRefusesZeroHold(t *testing.T) {
	directory := t.TempDir()
	original := []byte("original configuration\n")
	filename := writeHoldTestFile(t, directory, "config.toml", original)
	paths := resolvedPathsForStoreTest(t, filename, filepath.Join(directory, "state"))
	if _, err := writeTempReplace(hosttrust.HeldExclusive{}, osMigrationFileSystem{}, paths, []byte("restored\n")); !errors.Is(err, hosttrust.ErrExclusiveHoldRequired) {
		t.Fatalf("writeTempReplace(zero hold) err = %v, want %v", err, hosttrust.ErrExclusiveHoldRequired)
	}
	assertHoldTestDirClean(t, directory, filename, original)
}

// A live capability from another Store must not authorize a configuration
// replacement while the target Store's lock is held. This is the production
// position for the CR7 foreign-resource probe: the wrong Store has a genuine
// live token, but it is unbound to the target configuration path and refuses
// before staging any bytes.
func TestWriteTempReplaceRefusesForeignStoreHold(t *testing.T) {
	fixture := setupV4Fixture(t)
	original, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- fixture.store.WithExclusiveHold(func(hosttrust.HeldExclusive) error {
			close(entered)
			<-release
			return nil
		})
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("target store did not acquire its exclusive hold")
	}

	otherState := filepath.Join(t.TempDir(), "other-state")
	if err := os.MkdirAll(otherState, 0o700); err != nil {
		close(release)
		t.Fatal(err)
	}
	otherConfig := writeHoldTestFile(t, otherState, "other.toml", []byte("other configuration\n"))
	otherPaths := resolvedPathsForStoreTest(t, otherConfig, otherState)
	other, err := hosttrust.Open(otherPaths)
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	if err := other.EnsureConfigBinding(otherPaths); err != nil {
		close(release)
		t.Fatal(err)
	}
	if _, err := other.Initialize(); err != nil {
		close(release)
		t.Fatal(err)
	}
	writeErr := other.WithExclusiveHold(func(hold hosttrust.HeldExclusive) error {
		_, err := writeTempReplace(hold, osMigrationFileSystem{}, resolveLocalPathsForFixture(t, fixture), []byte("foreign replacement\n"))
		return err
	})
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("target holder error = %v", err)
	}
	if !errors.Is(writeErr, hosttrust.ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("foreign-store write error = %v, want %v", writeErr, hosttrust.ErrExclusiveHoldResourceMismatch)
	}
	after, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("foreign-store hold changed the target configuration")
	}
	entries, err := os.ReadDir(filepath.Dir(fixture.filename))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".ax-config-restore-") {
			t.Fatalf("foreign-store refusal left staging file %q", entry.Name())
		}
	}
}

// Even a deliberately forged store-side binding cannot make a foreign Store
// authorize a target that is durably bound to another StateRoot: this is the
// narrowing F2 probe where both the config path and the foreign pair match,
// so only the bidirectional target-side StateRoot check can reject the write
// before staging. The test keeps the target sidecar installed by
// setupV4Fixture and forges only the foreign store's derived index, so the
// association check in hosttrust.withExclusiveHold is the attacked gate.
func TestWriteTempReplaceRefusesForeignStoreWithTargetBinding(t *testing.T) {
	fixture := setupV4Fixture(t)
	original, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	foreignState := t.TempDir()
	foreignPaths := resolvedPathsForStoreTest(t, fixture.filename, foreignState)
	foreign, err := hosttrust.Open(foreignPaths)
	if err != nil {
		t.Fatal(err)
	}
	canonicalConfig, err := filepath.EvalSymlinks(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	canonicalForeignState, err := filepath.EvalSymlinks(foreignState)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := json.Marshal(hosttrust.ConfigBinding{
		Schema:        "urn:ax:schema:host-config-binding",
		SchemaVersion: "1.0.0",
		ConfigPath:    canonicalConfig,
		StateRoot:     filepath.Clean(canonicalForeignState),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign.Root(), "config-binding.json"), binding, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := foreign.Initialize(); err != nil {
		t.Fatal(err)
	}
	writeErr := foreign.WithExclusiveHold(func(hold hosttrust.HeldExclusive) error {
		// The foreign token and pair are both genuine, but the target sidecar
		// names the fixture's StateRoot rather than this token's root. The
		// writer must reject before staging any bytes.
		_, err := writeTempReplace(hold, osMigrationFileSystem{}, foreignPaths, []byte("foreign replacement\n"))
		return err
	})
	if !errors.Is(writeErr, hosttrust.ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("foreign target-bound write error = %v, want resource mismatch", writeErr)
	}
	after, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("foreign target-bound hold changed the target configuration")
	}
	entries, err := os.ReadDir(filepath.Dir(fixture.filename))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".ax-config-restore-") {
			t.Fatalf("foreign target-bound refusal left staging file %q", entry.Name())
		}
	}
}

// A token bound by the right Store to one configuration cannot be retargeted
// by a caller to a second path. This drives config.requireHoldForConfig after
// hosttrust has already accepted the Store-owned binding, and proves the
// second path is untouched.
func TestWriteTempReplaceRefusesConfiguredHoldForDifferentConfigPath(t *testing.T) {
	fixture := setupV4Fixture(t)
	otherDirectory := t.TempDir()
	original := []byte("other configuration\n")
	otherFilename := writeHoldTestFile(t, otherDirectory, "config.toml", original)
	var writeErr error
	err := fixture.store.WithExclusiveHold(func(hold hosttrust.HeldExclusive) error {
		_, writeErr = writeTempReplace(hold, osMigrationFileSystem{}, resolvedPathsForStoreTest(t, otherFilename, fixture.stateDir), []byte("wrong target\n"))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(writeErr, hosttrust.ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("configured hold retarget error = %v, want %v", writeErr, hosttrust.ErrExclusiveHoldResourceMismatch)
	}
	assertHoldTestDirClean(t, otherDirectory, otherFilename, original)
}
