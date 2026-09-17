package sessrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Lease crash drills kill a real child process at the install rename seam:
// the child stages and fsyncs a lease blob, signals readiness from the
// AfterLeaseStage hook, and blocks; the parent SIGKILLs it, reopens the
// repository in a fresh handle (restart recovery), and proves the lease is
// absent, the orphaned stage temp is ignored, and the byte-identical retry
// succeeds. Coordination travels through environment variables and the
// ready file; nothing travels through the crashed process's memory.
const (
	leaseCrashChildEnv   = "AX_SESSREPO_LEASE_CRASH_CHILD"
	leaseCrashRootEnv    = "AX_SESSREPO_CRASH_ROOT"
	leaseCrashSessionEnv = "AX_SESSREPO_CRASH_SESSION"
	leaseCrashModeEnv    = "AX_SESSREPO_CRASH_MODE"
	leaseCrashReadyEnv   = "AX_SESSREPO_CRASH_READY"
	leaseCrashExpectEnv  = "AX_SESSREPO_CRASH_EXPECTED"
)

func TestLeaseKillAtRenameSeam(t *testing.T) {
	if os.Getenv(leaseCrashChildEnv) == "1" {
		runLeaseCrashChild()
		return
	}
	t.Run("create", func(t *testing.T) { driveLeaseKill(t, "create") })
	t.Run("cas", func(t *testing.T) { driveLeaseKill(t, "cas") })
}

// runLeaseCrashChild opens the parent-prepared repository and starts the
// lease write. The AfterLeaseStage hook signals the parent and blocks
// forever, so this function only returns when the seam is never reached —
// itself a drill failure the parent reports.
func runLeaseCrashChild() {
	root := os.Getenv(leaseCrashRootEnv)
	sessionID := os.Getenv(leaseCrashSessionEnv)
	mode := os.Getenv(leaseCrashModeEnv)
	ready := os.Getenv(leaseCrashReadyEnv)
	repository, err := Open(root)
	if err != nil {
		os.Exit(11)
	}
	repository.AfterLeaseStage = func() error {
		if err := os.WriteFile(ready, []byte("staged"), 0o600); err != nil {
			os.Exit(12)
		}
		select {}
	}
	switch mode {
	case "create":
		_, _ = repository.CreateLease(sessionID, CreateLeaseInput{LeaseID: testLeaseID, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testLeaseAt})
	case "cas":
		_, _ = repository.CompareAndSwapLease(sessionID, LeaseExpectation{RecordID: os.Getenv(leaseCrashExpectEnv)}, SuccessorLeaseInput{
			CreateLeaseInput: CreateLeaseInput{LeaseID: testLeaseIDB, HolderHostID: testHostIDB, IssuedByHostID: testHostIDB, CreatedAt: testLeaseAt},
			Reason:           "graceful_takeover",
			CheckpointID:     zeroDigest,
		})
	default:
		os.Exit(13)
	}
	os.Exit(14)
}

// driveLeaseKill prepares the session, runs the child to the seam, kills
// it, and proves restart recovery plus the safe retry.
func driveLeaseKill(t *testing.T, mode string) {
	t.Helper()
	root := t.TempDir()
	repository, err := Open(root)
	if err != nil {
		t.Fatalf("Open(crash root) error = %v", err)
	}
	if _, err := repository.CreateSession([]byte(specRecordExample)); err != nil {
		t.Fatalf("CreateSession error = %v", err)
	}
	var expected string
	if mode == "cas" {
		first, err := repository.CreateLease(testSessionID, CreateLeaseInput{LeaseID: testLeaseID, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testLeaseAt})
		if err != nil {
			t.Fatalf("CreateLease(epoch 1) error = %v", err)
		}
		expected = first.RecordID
	}
	ready := filepath.Join(root, "ready-"+mode)
	child := exec.Command(os.Args[0], "-test.run=^TestLeaseKillAtRenameSeam$", "-test.count=1")
	child.Env = append(os.Environ(),
		leaseCrashChildEnv+"=1",
		leaseCrashRootEnv+"="+root,
		leaseCrashSessionEnv+"="+testSessionID,
		leaseCrashModeEnv+"="+mode,
		leaseCrashReadyEnv+"="+ready,
		leaseCrashExpectEnv+"="+expected,
	)
	if err := child.Start(); err != nil {
		t.Fatalf("start crash child: %v", err)
	}
	killed := false
	defer func() {
		if !killed {
			_ = child.Process.Kill()
		}
		_ = child.Wait()
	}()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("crash child never reached the rename seam (mode %s)", mode)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatalf("kill crash child: %v", err)
	}
	killed = true
	if err := child.Wait(); err == nil {
		t.Fatalf("crash child exited cleanly after SIGKILL (mode %s)", mode)
	}
	// Restart recovery: a fresh handle over the killed child's bytes.
	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(after kill) error = %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "sessions", testSessionID, "leases"))
	if err != nil {
		t.Fatalf("ReadDir(leases) error = %v", err)
	}
	var finals, temps int
	for _, entry := range entries {
		if entry.Name()[0] == '.' {
			temps++
		} else {
			finals++
		}
	}
	// Create mode stages the first blob: no final may exist. CAS mode
	// stages the second: exactly the pre-existing epoch-1 final stands.
	wantFinals := 0
	if mode == "cas" {
		wantFinals = 1
	}
	if finals != wantFinals || temps != 1 {
		t.Fatalf("leases dir after kill holds %d finals and %d temps, want %d and 1 (mode %s)", finals, temps, wantFinals, mode)
	}
	leases, err := reopened.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases(after kill) error = %v", err)
	}
	wantCount := 0
	if mode == "cas" {
		wantCount = 1
	}
	if len(leases) != wantCount {
		t.Fatalf("leases after kill = %d, want %d (mode %s)", len(leases), wantCount, mode)
	}
	// The byte-identical retry succeeds despite the orphaned temp.
	switch mode {
	case "create":
		reference, err := reopened.CreateLease(testSessionID, CreateLeaseInput{LeaseID: testLeaseID, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testLeaseAt})
		if err != nil {
			t.Fatalf("retry CreateLease error = %v", err)
		}
		if reference.Epoch != 1 {
			t.Fatalf("retry ref = %+v", reference)
		}
	case "cas":
		reference, err := reopened.CompareAndSwapLease(testSessionID, LeaseExpectation{RecordID: expected}, SuccessorLeaseInput{
			CreateLeaseInput: CreateLeaseInput{LeaseID: testLeaseIDB, HolderHostID: testHostIDB, IssuedByHostID: testHostIDB, CreatedAt: testLeaseAt},
			Reason:           "graceful_takeover",
			CheckpointID:     zeroDigest,
		})
		if err != nil {
			t.Fatalf("retry CAS error = %v", err)
		}
		if reference.Epoch != 2 {
			t.Fatalf("retry ref = %+v", reference)
		}
	}
	winner, err := reopened.WinningLease(testSessionID)
	if err != nil {
		t.Fatalf("WinningLease(after retry) error = %v", err)
	}
	if winner.Epoch != uint64(wantCount+1) {
		t.Fatalf("winner after retry = %+v", winner)
	}
	if _, _, err := mustAttestLeaseBytes(t, reopened, testSessionID, winner.RecordID); err != nil {
		t.Fatalf("attest retried lease: %v", err)
	}
}

// mustAttestLeaseBytes reads one lease back and verifies its identity.
func mustAttestLeaseBytes(t *testing.T, repository *Repository, sessionID, recordID string) ([]byte, string, error) {
	t.Helper()
	raw, err := repository.GetLease(sessionID, recordID)
	if err != nil {
		return nil, "", err
	}
	return raw, mustAttestLease(t, raw, "retried lease"), nil
}
