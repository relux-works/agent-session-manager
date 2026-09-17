package config

// Port of the CR6 independent review probes TestApplyV4FailedReplaceKeepsIntent
// and TestRollbackV4FailedReplaceKeepsIntent
// (TASK-260909-2ez769_review-evidence-rev6.zip, probes/config/reviewer_partial_test.go).
// RED-first regressions for F2: a failed replacement must never discard the intent
// needed to reject or converge partial state. Kept as committed tests.

import (
	"errors"
	"os"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
)

type partialReplaceFS struct {
	osMigrationFileSystem
	filename string
	renames  int
}

func (f *partialReplaceFS) Rename(from, to string) error {
	if to == f.filename {
		f.renames++
		if f.renames == 2 {
			return errors.New("injected rollback rename failure")
		}
	}
	return f.osMigrationFileSystem.Rename(from, to)
}

func (f *partialReplaceFS) OpenDirectory(path string) (migrationDirectory, error) {
	if f.renames == 1 {
		return nil, errors.New("injected post-replacement directory open failure")
	}
	return f.osMigrationFileSystem.OpenDirectory(path)
}

func TestApplyV4FailedReplaceKeepsIntent(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	before := fixture.trust(t).Generation
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	fs := &partialReplaceFS{filename: fixture.filename}
	_, err = applyV4(fixture.inputs, nil, preview, true, fixture.now, fs, openStoreForConfig)
	if !errors.Is(err, ErrMigrationRecovery) {
		t.Fatalf("expected rollback failure, got %v", err)
	}
	_, markerErr := os.Stat(fixture.store.PendingCommitPath())
	t.Logf("apply error=%v; marker inspection=%v", err, markerErr)
	after, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		return
	}
	loaded, ok := after.Config.Configuration()
	if !ok {
		t.Fatal("missing config")
	}
	if loaded.SourceVersion == Version4 && after.Generation == before {
		t.Fatalf("failed replacement discarded intent and admitted Config4 at old generation %d", before)
	}
}

func TestRollbackV4FailedReplaceKeepsIntent(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatal(err)
	}
	before := fixture.trust(t).Generation
	fs := &partialReplaceFS{filename: fixture.filename}
	_, err = rollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true, BackupPath: fixture.filename + ".bak." + CurrentVersion}, fs, openStoreForConfig)
	if !errors.Is(err, ErrMigrationRecovery) {
		t.Fatalf("expected rollback recovery failure, got %v", err)
	}
	after, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		return
	}
	loaded, ok := after.Config.Configuration()
	if !ok {
		t.Fatal("missing config")
	}
	if loaded.SourceVersion == CurrentVersion && after.Generation == before {
		t.Fatalf("failed rollback discarded intent and admitted Config3 at old generation %d", before)
	}
}

// unrestorableReplaceFS fails the post-replacement directory sync and
// every later rename of the configuration file, so neither the helper's
// own restoration nor the joint compensation restore can run: the only
// safe resolution is the kept marker, which restart recovery completes
// forward once the fault heals.
type unrestorableReplaceFS struct {
	osMigrationFileSystem
	filename string
	renames  int
}

func (f *unrestorableReplaceFS) Rename(from, to string) error {
	if to == f.filename {
		f.renames++
		if f.renames >= 2 {
			return errors.New("injected persistent rename failure")
		}
	}
	return f.osMigrationFileSystem.Rename(from, to)
}

func (f *unrestorableReplaceFS) OpenDirectory(path string) (migrationDirectory, error) {
	if f.renames == 1 {
		return nil, errors.New("injected post-replacement directory open failure")
	}
	return f.osMigrationFileSystem.OpenDirectory(path)
}

func assertMarkerKeptForTest(t *testing.T, fixture *v4Fixture) {
	t.Helper()
	if _, err := os.Stat(fixture.store.PendingCommitPath()); err != nil {
		t.Fatalf("joint marker not kept: %v", err)
	}
}

func TestApplyV4FailedReplaceKeepsMarkerWhenRestoreFails(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	before := fixture.trust(t).Generation
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = applyV4(fixture.inputs, nil, preview, true, fixture.now, &unrestorableReplaceFS{filename: fixture.filename}, openStoreForConfig)
	if !errors.Is(err, ErrMigrationRecovery) {
		t.Fatalf("expected recovery failure, got %v", err)
	}
	assertMarkerKeptForTest(t, fixture)
	// A reopened store converges the kept intent forward: the durable
	// replacement completes with the generation bump, never with the
	// replacement admitted at the old generation.
	reopened, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture))
	if err != nil {
		t.Fatalf("reopen after unrestorable replace err = %v", err)
	}
	converged, err := LoadCoherent(fixture.inputs, nil, reopened)
	if err != nil {
		t.Fatalf("reopened LoadCoherent err = %v", err)
	}
	final, ok := converged.Config.Configuration()
	if !ok {
		t.Fatal("missing converged configuration")
	}
	if final.SourceVersion != Version4 || converged.Generation != before+1 {
		t.Fatalf("reopened state = %s/generation %d, want %s/generation %d", final.SourceVersion, converged.Generation, Version4, before+1)
	}
	installed, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(preview.Replacement) {
		t.Fatal("reopened configuration is not the confirmed replacement bytes")
	}
	if _, err := os.Stat(fixture.store.PendingCommitPath()); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after forward convergence: %v", err)
	}
	// The surviving store observes the same converged pair.
	after, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	if after.Generation != before+1 {
		t.Fatalf("surviving generation = %d, want %d", after.Generation, before+1)
	}
}

func TestRollbackV4FailedReplaceKeepsMarkerWhenRestoreFails(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatal(err)
	}
	before := fixture.trust(t).Generation
	_, err = rollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true, BackupPath: fixture.filename + ".bak." + CurrentVersion}, &unrestorableReplaceFS{filename: fixture.filename}, openStoreForConfig)
	if !errors.Is(err, ErrMigrationRecovery) {
		t.Fatalf("expected rollback recovery failure, got %v", err)
	}
	assertMarkerKeptForTest(t, fixture)
	reopened, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture))
	if err != nil {
		t.Fatalf("reopen after unrestorable rollback err = %v", err)
	}
	converged, err := LoadCoherent(fixture.inputs, nil, reopened)
	if err != nil {
		t.Fatalf("reopened LoadCoherent err = %v", err)
	}
	final, ok := converged.Config.Configuration()
	if !ok {
		t.Fatal("missing converged configuration")
	}
	if final.SourceVersion != CurrentVersion || converged.Generation != before+1 {
		t.Fatalf("reopened state = %s/generation %d, want %s/generation %d", final.SourceVersion, converged.Generation, CurrentVersion, before+1)
	}
	after, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	if after.Generation != before+1 {
		t.Fatalf("surviving generation = %d, want %d", after.Generation, before+1)
	}
}
