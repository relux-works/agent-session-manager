//go:build darwin || linux

package clonesnap

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// parseTestUUID parses one fixture UUID outside a testing context.
func parseTestUUID(value string) (scalar.UUIDv7, error) {
	return scalar.ParseUUIDv7(value)
}

// testNativeIdentity builds the fixture native identity outside a
// testing context.
func testNativeIdentity(workspace scalar.UUIDv7) clonebundle.NativeIdentity {
	return clonebundle.NativeIdentity{
		NativeSessionID:    "native-session-snap",
		IdentityKind:       "provider_native",
		LogicalWorkspaceID: workspace,
	}
}

// Real process-termination evidence for the manifest-publish seam:
// the child runs the genuine production Capture entry and SIGKILLs
// itself between the raw manifest publish and the capture manifest
// seal. The parent then proves the kill left one complete raw
// manifest with no capture manifest behind, and the identical replay
// converges with both manifests and byte-identical bytes.
//
// The child rebuilds nothing: it opens the parent's store root,
// object home, and workspace from the environment, so its closure
// inputs are byte-identical to the parent's replay by construction.

const (
	crashChildEnv       = "AX_CLONESNAP_CRASH_CHILD"
	crashChildRoot      = "AX_CLONESNAP_CRASH_ROOT"
	crashChildHome      = "AX_CLONESNAP_CRASH_HOME"
	crashChildWorkspace = "AX_CLONESNAP_CRASH_WORKSPACE"
	crashGeneration     = "crash-generation-7"
)

func TestCaptureCrashChildSelfTerminates(t *testing.T) {
	if os.Getenv(crashChildEnv) == "1" {
		captureCrashChild()
		return
	}
	storeRoot := fixtureStore(t)
	home := t.TempDir()
	config := filepath.Join(home, "config")
	if err := os.Mkdir(config, 0o700); err != nil {
		t.Fatal(err)
	}
	workspace := fixtureWorkspace(t)
	if _, err := openStoreAt(home); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestCaptureCrashChildSelfTerminates$",
		"-test.count=1",
	)
	child.Env = append(os.Environ(),
		crashChildEnv+"=1",
		crashChildRoot+"="+storeRoot,
		crashChildHome+"="+home,
		crashChildWorkspace+"="+workspace.WorkspaceRoot,
	)
	runErr := child.Run()
	var exitErr *exec.ExitError
	if !errors.As(runErr, &exitErr) {
		t.Fatalf("child run = %v, want a signal-kill exit", runErr)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("child status = %v, want killed by SIGKILL", exitErr)
	}
	// The kill landed between the raw publish and the capture seal:
	// one complete raw manifest that verifies, no capture manifest,
	// no partial bytes.
	store, err := openStoreAt(home)
	if err != nil {
		t.Fatal(err)
	}
	rawFound := false
	for _, blob := range listStoredBlobs(t, store) {
		if _, err := clonebundle.DecodeRawObjectManifest(blob.bytes); err == nil {
			if rawFound {
				t.Fatal("two raw manifests after one killed capture")
			}
			rawFound = true
		}
		if _, err := clonebundle.DecodeCaptureManifest(blob.bytes); err == nil {
			t.Fatal("capture manifest present after a kill before its seal")
		}
	}
	if !rawFound {
		t.Fatal("raw manifest missing after a kill past its publish")
	}
	// The identical replay converges: the raw publish verifies and
	// reuses, the capture manifest installs, and both manifests
	// decode.
	request := crashRequest(t, storeRoot, store, workspace.WorkspaceRoot)
	replayed, err := Capture(request)
	if err != nil {
		t.Fatalf("replayed Capture() error = %v", err)
	}
	if replayed.RawInstalled {
		t.Fatal("replay reinstalled the raw manifest instead of verifying and reusing")
	}
	if !replayed.CaptureInstalled {
		t.Fatal("replay did not install the capture manifest")
	}
	// The replayed capture references only installed blobs: the
	// kill installed no half-written manifest and skipped no
	// payload.
	assertEveryRawBlobInstalled(t, store, replayed.Raw)
	// The replayed bytes equal a clean capture over an identical
	// fixture: the kill published no half-written manifest.
	cleanRoot := fixtureStore(t)
	clean, err := Capture(crashRequest(t, cleanRoot, openTestStore(t), fixtureWorkspace(t).WorkspaceRoot))
	if err != nil {
		t.Fatalf("clean Capture() error = %v", err)
	}
	if !bytes.Equal(replayed.RawManifest, clean.RawManifest) {
		t.Fatal("replayed raw manifest differs from the clean capture")
	}
	if !bytes.Equal(replayed.CaptureManifest, clean.CaptureManifest) {
		t.Fatal("replayed capture manifest differs from the clean capture")
	}
}

// crashRequest rebuilds the deterministic crash capture request over
// the given store root, object store, and workspace root.
func crashRequest(t *testing.T, storeRoot string, store *localstore.ObjectStore, workspaceRoot string) CaptureRequest {
	t.Helper()
	branch := "main"
	head := fixtureDigest("snap-head")
	request := CaptureRequest{
		StoreRoot:         storeRoot,
		Platform:          fixturePlatform(),
		Plan:              fixturePlan(),
		OperationID:       fixtureOperationID,
		BundleID:          fixtureBundleID,
		SourceBasis:       fixtureSourceBasis(),
		SourceEnvironment: []byte(fixtureTupleJSON()),
		SourceIdentity:    fixtureNativeIdentity(t),
		CapturePlanDigest: fixtureDigest("snap-capture-plan"),
		NativeSessionID:   "native-session-snap",
		CreatedByHostID:   fixtureHostID,
		CreatedAt:         fixtureCreatedAt,
		Generation:        crashGeneration,
		ProofKind:         "closed_store",
		InputBlocked:      true,
		ForegroundIdle:    true,
		BackgroundIdle:    true,
		OnRace:            RaceRefuse,
		Workspace: WorkspaceRequest{
			LogicalWorkspaceID: fixtureWorkspaceID,
			WorkspaceRoot:      workspaceRoot,
			Cwd:                filepath.Join(workspaceRoot, "work", "trees", "alpha"),
			RemoteFingerprints: []string{fixtureDigest("snap-remote")},
			Branch:             &branch,
			HeadDigest:         &head,
			Platform:           fixturePlatform(),
		},
		Store: store,
	}
	return request
}

// captureCrashChild runs the production Capture entry and kills its
// own process at the AfterRawPublish boundary. It never returns.
func captureCrashChild() {
	store, err := openStoreAt(os.Getenv(crashChildHome))
	if err != nil {
		os.Exit(66)
	}
	branch := "main"
	head := fixtureDigest("snap-head")
	workspace, err := parseTestUUID(fixtureWorkspaceID)
	if err != nil {
		os.Exit(66)
	}
	request := CaptureRequest{
		StoreRoot:         os.Getenv(crashChildRoot),
		Platform:          fixturePlatform(),
		Plan:              fixturePlan(),
		OperationID:       fixtureOperationID,
		BundleID:          fixtureBundleID,
		SourceBasis:       fixtureSourceBasis(),
		SourceEnvironment: []byte(fixtureTupleJSON()),
		SourceIdentity:    testNativeIdentity(workspace),
		CapturePlanDigest: fixtureDigest("snap-capture-plan"),
		NativeSessionID:   "native-session-snap",
		CreatedByHostID:   fixtureHostID,
		CreatedAt:         fixtureCreatedAt,
		Generation:        crashGeneration,
		ProofKind:         "closed_store",
		InputBlocked:      true,
		ForegroundIdle:    true,
		BackgroundIdle:    true,
		OnRace:            RaceRefuse,
		Workspace: WorkspaceRequest{
			LogicalWorkspaceID: fixtureWorkspaceID,
			WorkspaceRoot:      os.Getenv(crashChildWorkspace),
			Cwd:                filepath.Join(os.Getenv(crashChildWorkspace), "work", "trees", "alpha"),
			RemoteFingerprints: []string{fixtureDigest("snap-remote")},
			Branch:             &branch,
			HeadDigest:         &head,
			Platform:           fixturePlatform(),
		},
		Store: store,
		Hooks: CaptureHooks{AfterRawPublish: func() error {
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
			// The kill is asynchronous under instrumentation:
			// block here so a delivered SIGKILL always wins the
			// race against process exit. If the kill failed, the
			// parent context times out and fails the test instead
			// of passing on a wrong exit.
			select {}
		}},
	}
	_, _ = Capture(request)
	// The hook must have fired: reaching here means the boundary
	// was skipped, so fail the parent with a clean exit.
	os.Exit(0)
}
