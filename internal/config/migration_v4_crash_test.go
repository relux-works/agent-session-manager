package config

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// crashChildInputs rebuilds production inputs inside a re-executed crash
// child: the child crashes through the real apply/rollback entry points,
// never through a simulated commit.
func crashChildInputs(directory string) (Inputs, string, string) {
	filename := filepath.Join(directory, "config.toml")
	state := filepath.Join(directory, "state")
	inputs := Inputs{Platform: scalar.PlatformMacOS, HomeDir: directory, TempDir: directory, WorkingDir: directory, Stat: os.Stat, ReadFile: os.ReadFile, LookupEnv: func(key string) (string, bool) {
		if key == "AX_CONFIG" {
			return filename, true
		}
		if key == "AX_STATE_DIR" {
			return state, true
		}
		return "", false
	}}
	return inputs, filename, state
}

func crashChildSelf(t *testing.T, store *hosttrust.Store) (hosttrust.Snapshot, string) {
	t.Helper()
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	var self string
	for _, entry := range snapshot.Trust.Entries {
		if entry.HostID.String() == testHostID {
			self = entry.CredentialID.String()
		}
	}
	return snapshot, self
}

func assertChildCrashedAtSeam(t *testing.T, testName, env, directory string) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^"+testName+"$")
	command.Env = append(os.Environ(), env+"="+directory)
	output, err := command.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 77 {
		t.Fatalf("child did not crash at the replacement seam: %v %s", err, output)
	}
}

func assertMarkerGone(t *testing.T, store *hosttrust.Store) {
	t.Helper()
	if _, err := os.Lstat(store.PendingCommitPath()); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after convergence: %v", err)
	}
}

func assertMarkerKept(t *testing.T, store *hosttrust.Store) {
	t.Helper()
	if _, err := os.Lstat(store.PendingCommitPath()); err != nil {
		t.Fatalf("joint marker was removed: %v", err)
	}
}

// A store opened before an apply crash converges on read: the surviving
// LoadCoherent must observe the replaced Config4 with the bumped
// generation, never the new bytes with the old one. A reopened store
// converges identically.
func TestSurvivingReaderConvergesInterruptedApply(t *testing.T) {
	if directory := os.Getenv("AX_V4_SURVIVING_APPLY_FIXTURE"); directory != "" {
		inputs, filename, state := crashChildInputs(directory)
		store, err := hosttrust.Open(resolvedPathsForStoreTest(t, filename, state))
		if err != nil {
			t.Fatal(err)
		}
		snapshot, self := crashChildSelf(t, store)
		now := time.Now().UTC()
		preview, err := PreviewV4(inputs, nil, snapshot, self, nil, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = applyV4(inputs, nil, preview, true, now, crashAfterReplaceFileSystem{target: filename}, openStoreForConfig)
		t.Fatalf("crash seam not reached: %v", err)
	}
	fixture := setupV4Fixture(t, testPeerID)
	before := fixture.trust(t).Generation
	assertChildCrashedAtSeam(t, "TestSurvivingReaderConvergesInterruptedApply", "AX_V4_SURVIVING_APPLY_FIXTURE", fixture.directory)
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatalf("surviving LoadCoherent error = %v", err)
	}
	configuration, ok := coherent.Config.Configuration()
	if !ok {
		t.Fatal("surviving LoadCoherent has no configuration")
	}
	if configuration.SourceVersion != Version4 || coherent.Generation != before+1 {
		t.Fatalf("surviving reader converged to version %q generation %d, want 4.0.0/%d", configuration.SourceVersion, coherent.Generation, before+1)
	}
	assertMarkerGone(t, fixture.store)
	loaded, err := Load(fixture.inputs, nil)
	if err != nil {
		t.Fatalf("Load after child exit 77 error = %v", err)
	}
	reloaded, ok := loaded.Configuration()
	if !ok {
		t.Fatal("Load after child exit 77 has no configuration")
	}
	reopened, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture))
	if err != nil {
		t.Fatalf("hosttrust.Open after child exit 77 error = %v", err)
	}
	snapshot, err := reopened.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot after child exit 77 error = %v", err)
	}
	if reloaded.SourceVersion != Version4 || snapshot.Generation != before+1 {
		t.Fatalf("reopened reader converged to version %q generation %d, want 4.0.0/%d", reloaded.SourceVersion, snapshot.Generation, before+1)
	}
}

// A store opened before a rollback crash converges on read: the surviving
// LoadCoherent must observe the restored backup with the bumped
// generation, never the restored bytes with the pre-rollback one.
func TestSurvivingReaderConvergesInterruptedRollback(t *testing.T) {
	if directory := os.Getenv("AX_V4_SURVIVING_ROLLBACK_FIXTURE"); directory != "" {
		inputs, filename, _ := crashChildInputs(directory)
		_, err := rollbackV4(inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true}, crashAfterReplaceFileSystem{target: filename}, openStoreForConfig)
		t.Fatalf("crash seam not reached: %v", err)
	}
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatal(err)
	}
	before := fixture.trust(t).Generation
	assertChildCrashedAtSeam(t, "TestSurvivingReaderConvergesInterruptedRollback", "AX_V4_SURVIVING_ROLLBACK_FIXTURE", fixture.directory)
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatalf("surviving LoadCoherent error = %v", err)
	}
	configuration, ok := coherent.Config.Configuration()
	if !ok {
		t.Fatal("surviving LoadCoherent has no configuration")
	}
	if configuration.SourceVersion != CurrentVersion || coherent.Generation != before+1 {
		t.Fatalf("surviving reader converged to version %q generation %d, want %s/%d", configuration.SourceVersion, coherent.Generation, CurrentVersion, before+1)
	}
	assertMarkerGone(t, fixture.store)
	reopened, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture))
	if err != nil {
		t.Fatalf("hosttrust.Open after child exit 77 error = %v", err)
	}
	snapshot, err := reopened.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot after child exit 77 error = %v", err)
	}
	loaded, err := Load(fixture.inputs, nil)
	if err != nil {
		t.Fatalf("Load after child exit 77 error = %v", err)
	}
	reloaded, ok := loaded.Configuration()
	if !ok {
		t.Fatal("Load after child exit 77 has no configuration")
	}
	if reloaded.SourceVersion != CurrentVersion || snapshot.Generation != before+1 {
		t.Fatalf("reopened reader converged to version %q generation %d, want %s/%d", reloaded.SourceVersion, snapshot.Generation, CurrentVersion, before+1)
	}
}

// An operator edit after an apply crash matches neither the source nor the
// replacement hash: the surviving LoadCoherent refuses with the marker
// kept for operator resolution, and a fresh open refuses too.
func TestSurvivingReaderRefusesDivergedConfig(t *testing.T) {
	if directory := os.Getenv("AX_V4_DIVERGED_FIXTURE"); directory != "" {
		inputs, filename, state := crashChildInputs(directory)
		store, err := hosttrust.Open(resolvedPathsForStoreTest(t, filename, state))
		if err != nil {
			t.Fatal(err)
		}
		snapshot, self := crashChildSelf(t, store)
		now := time.Now().UTC()
		preview, err := PreviewV4(inputs, nil, snapshot, self, nil, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = applyV4(inputs, nil, preview, true, now, crashAfterReplaceFileSystem{target: filename}, openStoreForConfig)
		t.Fatalf("crash seam not reached: %v", err)
	}
	fixture := setupV4Fixture(t, testPeerID)
	assertChildCrashedAtSeam(t, "TestSurvivingReaderRefusesDivergedConfig", "AX_V4_DIVERGED_FIXTURE", fixture.directory)
	if err := os.WriteFile(fixture.filename, []byte("# operator edit after crash\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store); err == nil {
		t.Fatalf("surviving LoadCoherent(diverged) admitted generation %d, want refusal", coherent.Generation)
	} else if !errors.Is(err, hosttrust.ErrJointIntervened) {
		t.Fatalf("surviving LoadCoherent error = %v, want ErrJointIntervened", err)
	}
	assertMarkerKept(t, fixture.store)
	if _, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture)); err == nil {
		t.Fatal("hosttrust.Open(diverged) succeeded, want refusal")
	} else if !errors.Is(err, hosttrust.ErrJointIntervened) {
		t.Fatalf("hosttrust.Open error = %v, want ErrJointIntervened", err)
	}
}
