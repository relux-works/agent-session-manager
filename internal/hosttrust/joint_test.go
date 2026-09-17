package hosttrust

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func jointFixture(t *testing.T, source, replacement []byte) (*Store, string, Snapshot) {
	t.Helper()
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(directory, "state")
	paths := resolvedPathsForTest(t, configPath, stateDir)
	store, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureConfigBinding(paths); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	return store, configPath, snapshot
}

// errUnexpectedRestore fails any joint commit that compensates: the tests
// using unexpectedRestore never fault the generation bump, so the restore
// must not run.
var errUnexpectedRestore = errors.New("joint restore must not run")

func unexpectedRestore(HeldExclusive) error { return errUnexpectedRestore }

func jointMarker(configPath string, source, replacement []byte, generation uint64) PendingCommit {
	return PendingCommit{
		Schema:            "urn:ax:schema:host-pending-commit",
		SchemaVersion:     "1.0.0",
		Operation:         JointApplyV4,
		ConfigPath:        configPath,
		SourceSHA256:      SHA256HexOf(source),
		ReplacementSHA256: SHA256HexOf(replacement),
		SourceGeneration:  generation,
		BackupPath:        configPath + ".bak.3.0.0",
	}
}

func TestJointCommitHappyPath(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	replace := func(HeldExclusive) error { return os.WriteFile(configPath, replacement, 0o600) }
	revalidate := func(current TrustStore) error {
		if current.Generation != snapshot.Generation {
			return errors.New("generation moved")
		}
		return nil
	}
	generation, err := store.JointCommit(store.paths, marker, revalidate, replace, unexpectedRestore)
	if err != nil {
		t.Fatalf("JointCommit error = %v", err)
	}
	if generation != snapshot.Generation+1 {
		t.Fatalf("JointCommit generation = %d, want %d", generation, snapshot.Generation+1)
	}
	installed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(replacement) {
		t.Fatal("JointCommit did not install the replacement")
	}
	if _, err := os.Lstat(pendingPath(store.root)); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after commit: %v", err)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != generation {
		t.Fatalf("committed generation = %d, want %d", fresh.Generation, generation)
	}
}

func TestJointCommitRevalidatesStaleGeneration(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	// A concurrent trust commit moves the generation before the joint
	// commit runs: the in-lock revalidation must refuse the stale pins.
	peer := setupStoreForTest(t)
	enrollPeerForTest(t, store, peer, testHostC, testNow)
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	called := false
	_, err := store.JointCommit(store.paths, marker,
		func(current TrustStore) error {
			if current.Generation != snapshot.Generation {
				return trustError(TrustError{Operation: "validate preview generation", Err: ErrStaleGeneration})
			}
			return nil
		},
		func(HeldExclusive) error {
			called = true
			return nil
		},
		unexpectedRestore)
	if err == nil || called {
		t.Fatalf("JointCommit(stale) admitted: err=%v replace=%v", err, called)
	} else if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("JointCommit(stale) error = %v, want ErrStaleGeneration", err)
	}
	if _, err := os.Lstat(pendingPath(store.root)); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after refused commit: %v", err)
	}
	installed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(source) {
		t.Fatal("refused joint commit mutated the configuration")
	}
}

func TestJointCommitReplaceFailureAborts(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	replaceErr := errors.New("replacement unavailable")
	_, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error { return replaceErr },
		unexpectedRestore)
	if err == nil {
		t.Fatal("JointCommit(failing replace) succeeded, want refusal")
	} else if !errors.Is(err, replaceErr) {
		t.Fatalf("JointCommit error = %v, want the replace failure", err)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != snapshot.Generation {
		t.Fatalf("aborted joint commit moved generation to %d", fresh.Generation)
	}
	if _, err := os.Lstat(pendingPath(store.root)); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after aborted commit: %v", err)
	}
}

func TestJointCommitRefusesUnboundStore(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.toml")
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(directory, "state")
	store, err := Open(resolvedPathsForTest(t, configPath, stateDir))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	called := false
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	if _, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error { called = true; return nil },
		unexpectedRestore); err == nil || called {
		t.Fatalf("unbound JointCommit admitted: err=%v replace=%v", err, called)
	} else if !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("unbound JointCommit error = %v, want resource mismatch", err)
	}
	if _, err := os.Lstat(pendingPath(store.root)); !os.IsNotExist(err) {
		t.Fatalf("unbound JointCommit left marker: %v", err)
	}
}

func TestRecoverCompletesForwardAfterCrash(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	// Stage exactly what a crash between the configuration replacement and
	// the generation bump leaves: the marker plus the replaced bytes with
	// the old generation still committed.
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	encoded, err := encodePendingCommit(marker)
	if err != nil {
		t.Fatal(err)
	}
	commitDocumentForTest(t, store, pendingPath(store.root), encoded)
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := store.Recover()
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	if !result.Found || !result.Completed || result.Aborted {
		t.Fatalf("Recover = %+v, want found+completed", result)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != snapshot.Generation+1 {
		t.Fatalf("recovered generation = %d, want %d", fresh.Generation, snapshot.Generation+1)
	}
	if _, err := os.Lstat(pendingPath(store.root)); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after recovery: %v", err)
	}
}

func TestOpenConvergesInterruptedCommit(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	encoded, err := encodePendingCommit(marker)
	if err != nil {
		t.Fatal(err)
	}
	commitDocumentForTest(t, store, pendingPath(store.root), encoded)
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	// Re-opening the same state directory must converge the interrupted
	// commit instead of admitting the replacement with the old generation.
	stateDir := filepath.Dir(store.root)
	reopened, err := openStore(resolvedPathsForTest(t, filepath.Join(filepath.Dir(stateDir), "config.toml"), stateDir), osFileSystem{})
	if err != nil {
		t.Fatalf("openStore after crash error = %v", err)
	}
	converged, err := reopened.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if converged.Generation != snapshot.Generation+1 {
		t.Fatalf("reopened generation = %d, want %d", converged.Generation, snapshot.Generation+1)
	}
}

func TestRecoverAbortsWhenReplacementAbsent(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	// A crash before the configuration replacement leaves the source bytes
	// with the marker outstanding: recovery aborts with no generation change.
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	encoded, err := encodePendingCommit(marker)
	if err != nil {
		t.Fatal(err)
	}
	commitDocumentForTest(t, store, pendingPath(store.root), encoded)
	result, err := store.Recover()
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	if !result.Found || !result.Aborted || result.Completed {
		t.Fatalf("Recover = %+v, want found+aborted", result)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != snapshot.Generation {
		t.Fatalf("aborted recovery moved generation to %d", fresh.Generation)
	}
}

func TestRecoverRefusesIntervention(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	encoded, err := encodePendingCommit(marker)
	if err != nil {
		t.Fatal(err)
	}
	commitDocumentForTest(t, store, pendingPath(store.root), encoded)
	// An operator edit mid-flight matches neither the source nor the
	// replacement hash: recovery refuses and keeps the marker.
	if err := os.WriteFile(configPath, []byte("operator edit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Recover(); err == nil {
		t.Fatal("Recover(intervened) succeeded, want refusal")
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("Recover error = %v, want ErrJointIntervened", err)
	}
	if _, err := os.Lstat(pendingPath(store.root)); err != nil {
		t.Fatalf("intervention marker was removed: %v", err)
	}
	// A refused recovery fails the open, so incoherent state cannot be used.
	stateDir := filepath.Dir(store.root)
	if _, err := openStore(resolvedPathsForTest(t, filepath.Join(stateDir, "config.toml"), stateDir), osFileSystem{}); err == nil {
		t.Fatal("openStore(intervened) succeeded, want refusal")
	}
}

func TestRecoverConvergesAlreadyBumped(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	// The bump landed but marker removal did not: the transaction commits
	// first, then the marker and the replaced bytes are staged exactly as
	// a crash between the bump and the marker removal leaves them. The
	// transaction must run before staging: every transaction converges
	// pending intent first, so staging before transacting would complete
	// the marker instead of leaving it outstanding.
	if err := store.transact(func(*TrustStore) error { return nil }); err != nil {
		t.Fatal(err)
	}
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	encoded, err := encodePendingCommit(marker)
	if err != nil {
		t.Fatal(err)
	}
	commitDocumentForTest(t, store, pendingPath(store.root), encoded)
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := store.Recover()
	if err != nil {
		t.Fatalf("Recover error = %v", err)
	}
	if !result.Found || !result.Completed {
		t.Fatalf("Recover = %+v, want found+completed", result)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != snapshot.Generation+1 {
		t.Fatalf("recovery bumped twice: generation = %d", fresh.Generation)
	}
}

func TestJointCommitRefusesStaleMarkerGeneration(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	if snapshot.Generation == 0 {
		t.Fatal("fixture generation is zero")
	}
	// The marker pins an older generation while the revalidation callback
	// is permissive: the joint backstop must still refuse the stale pin,
	// staging nothing and touching neither the configuration nor the trust.
	marker := jointMarker(configPath, source, replacement, snapshot.Generation-1)
	called := false
	if _, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error {
			called = true
			return nil
		},
		unexpectedRestore); err == nil || called {
		t.Fatalf("JointCommit(stale marker) admitted: replace=%v", called)
	} else if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("JointCommit error = %v, want ErrStaleGeneration", err)
	}
	if _, err := os.Lstat(pendingPath(store.root)); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after refused commit: %v", err)
	}
	installed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(source) {
		t.Fatal("refused joint commit mutated the configuration")
	}
}

func TestRecoverRefusesDivergedGeneration(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	// The replacement committed, but the generation since moved two bumps
	// past the pinned source (operator restore plus an unrelated commit):
	// recovery must refuse the diverged pair and keep the marker instead
	// of completing forward under a generation the operator never confirmed.
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	encoded, err := encodePendingCommit(marker)
	if err != nil {
		t.Fatal(err)
	}
	commitDocumentForTest(t, store, pendingPath(store.root), encoded)
	if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	// Every transaction converges pending intent first, so the diverged
	// trust is written directly: an operator restore landing outside the
	// barrier is the only way committed trust moves twice past a still
	// outstanding marker.
	diverged := snapshot.Trust
	diverged.Generation = snapshot.Generation + 2
	rewritten, err := EncodeTrust(diverged)
	if err != nil {
		t.Fatal(err)
	}
	commitDocumentForTest(t, store, filepath.Join(store.root, trustFileName), rewritten)
	if _, err := store.Recover(); err == nil {
		t.Fatal("Recover(diverged generation) succeeded, want refusal")
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("Recover error = %v, want ErrJointIntervened", err)
	}
	if _, err := os.Lstat(pendingPath(store.root)); err != nil {
		t.Fatalf("divergence marker was removed: %v", err)
	}
}

func TestJointCommitRefusesRelativeMarkerPath(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	marker := jointMarker(configPath, source, replacement, snapshot.Generation)
	marker.ConfigPath = "relative/config.toml"
	if _, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error { return nil },
		unexpectedRestore); err == nil {
		t.Fatal("JointCommit(relative marker path) succeeded, want refusal")
	} else if !errors.Is(err, ErrTrustValidation) {
		t.Fatalf("JointCommit error = %v, want ErrTrustValidation", err)
	}
}

func TestPendingCommitClosedDecode(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	valid, err := encodePendingCommit(jointMarker("/abs/config.toml", source, replacement, 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodePendingCommit(valid); err != nil {
		t.Fatalf("decodePendingCommit(valid) error = %v", err)
	}
	cases := []struct {
		name    string
		mutate  func(string) string
		wantErr error
	}{
		{"unknown field", func(document string) string {
			return document[:len(document)-1] + `,"comment":"x"}`
		}, ErrTrustDecode},
		{"duplicate key", func(document string) string {
			return document[:len(document)-1] + `,"operation":"apply-v4"}`
		}, ErrTrustValidation},
		{"quoted generation", func(document string) string {
			return `{"schema":"urn:ax:schema:host-pending-commit","schema_version":"1.0.0","operation":"apply-v4","config_path":"/abs/config.toml","source_sha256":"` + SHA256HexOf(source) + `","replacement_sha256":"` + SHA256HexOf(replacement) + `","source_generation":"3","backup_path":"/abs/config.toml.bak.3.0.0"}`
		}, ErrTrustValidation},
		{"wrong operation", func(document string) string {
			return replaceOnce(document, `"apply-v4"`, `"install"`)
		}, ErrTrustValidation},
		{"relative config path", func(document string) string {
			return replaceOnce(document, `"/abs/config.toml"`, `"config.toml"`)
		}, ErrTrustValidation},
		{"equal hashes", func(document string) string {
			return replaceOnce(document, SHA256HexOf(replacement), SHA256HexOf(source))
		}, ErrTrustValidation},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodePendingCommit([]byte(test.mutate(string(valid)))); err == nil {
				t.Fatalf("decodePendingCommit(%s) succeeded, want refusal", test.name)
			} else if !errors.Is(err, test.wantErr) {
				t.Fatalf("decodePendingCommit(%s) error = %v, want %v", test.name, err, test.wantErr)
			}
		})
	}
}

func replaceOnce(document, old, replacement string) string {
	for index := 0; index+len(old) <= len(document); index++ {
		if document[index:index+len(old)] == old {
			return document[:index] + replacement + document[index+len(old):]
		}
	}
	return document
}

// crownTrustAtExhaustion rewrites the fixture trust at the uint53 ceiling:
// the next generation bump is genuinely refused, which faults the joint
// commit after the configuration replacement is already durable.
func crownTrustAtExhaustion(t *testing.T, store *Store, snapshot Snapshot) Snapshot {
	t.Helper()
	crowned := TrustStore{Generation: MaxGeneration, Entries: snapshot.Trust.Entries}
	document, err := EncodeTrust(crowned)
	if err != nil {
		t.Fatalf("EncodeTrust(exhausted) error = %v", err)
	}
	commitDocumentForTest(t, store, filepath.Join(store.root, trustFileName), document)
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

// A trust-commit failure after a durable replacement compensates under the
// same exclusive hold: the source is restored and the marker aborts with
// no generation change. The operation still fails with the original bump
// refusal preserved.
func TestJointCommitCompensatesBumpFailure(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	crowned := crownTrustAtExhaustion(t, store, snapshot)
	marker := jointMarker(configPath, source, replacement, crowned.Generation)
	restored := false
	_, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error { return os.WriteFile(configPath, replacement, 0o600) },
		func(HeldExclusive) error {
			restored = true
			return os.WriteFile(configPath, source, 0o600)
		})
	if err == nil {
		t.Fatal("JointCommit(exhausted bump) succeeded, want failure")
	} else if !errors.Is(err, ErrJointReplacementDurable) {
		t.Fatalf("JointCommit error = %v, want joint replacement failure", err)
	} else if errors.Is(err, ErrJointCompensationFailed) {
		t.Fatalf("JointCommit error = %v, want a clean abort, not failed compensation", err)
	} else if !errors.Is(err, ErrGenerationExhausted) {
		t.Fatalf("JointCommit error = %v, want the original bump refusal preserved", err)
	}
	if !restored {
		t.Fatal("compensating commit did not run the restore")
	}
	installed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(source) {
		t.Fatal("compensating commit did not restore the source bytes")
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != MaxGeneration {
		t.Fatalf("compensated generation = %d, want the unchanged ceiling", fresh.Generation)
	}
	assertMarkerGone(t, store)
}

// A restore failure keeps the marker for operator resolution and reports
// the compound failure: the configuration stays at the replacement, which
// restart recovery completes forward once the fault heals.
func TestJointCommitCompensationFailsWhenRestoreFails(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	crowned := crownTrustAtExhaustion(t, store, snapshot)
	marker := jointMarker(configPath, source, replacement, crowned.Generation)
	restoreErr := errors.New("restore unavailable")
	_, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error { return os.WriteFile(configPath, replacement, 0o600) },
		func(HeldExclusive) error { return restoreErr })
	if err == nil {
		t.Fatal("JointCommit(failing restore) succeeded, want failure")
	} else if !errors.Is(err, ErrJointCompensationFailed) {
		t.Fatalf("JointCommit error = %v, want compensation failure", err)
	} else if !errors.Is(err, ErrJointReplacementDurable) {
		t.Fatalf("JointCommit error = %v, want joint replacement failure", err)
	} else if !errors.Is(err, restoreErr) {
		t.Fatalf("JointCommit error = %v, want the restore failure preserved", err)
	} else if !errors.Is(err, ErrGenerationExhausted) {
		t.Fatalf("JointCommit error = %v, want the original bump refusal preserved", err)
	}
	installed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(replacement) {
		t.Fatal("failed compensation did not leave the replacement for forward recovery")
	}
	assertMarkerKept(t, store)
}

// tamperDigestHex flips the first hex digit of a marker digest, keeping a
// valid 64-hex shape with different pins.
func tamperDigestHex(t *testing.T, digest string) string {
	t.Helper()
	if len(digest) != 64 {
		t.Fatalf("digest length = %d, want 64 hex", len(digest))
	}
	flipped := byte('1')
	if digest[0] == '1' {
		flipped = '0'
	}
	return string(flipped) + digest[1:]
}

// A marker that moved outside the confirmed pins while the hold was held
// (only an operator or a rogue writer outside the barrier can do that)
// refuses compensation as an intervention: the restore must not run
// blindly against superseded intent.
func TestJointCommitCompensationRefusesTamperedMarker(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	crowned := crownTrustAtExhaustion(t, store, snapshot)
	marker := jointMarker(configPath, source, replacement, crowned.Generation)
	tampered := marker
	tampered.ReplacementSHA256 = tamperDigestHex(t, marker.ReplacementSHA256)
	encoded, err := encodePendingCommit(tampered)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	_, err = store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error {
			if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
				return err
			}
			return os.WriteFile(pendingPath(store.root), encoded, 0o600)
		},
		func(HeldExclusive) error {
			called = true
			return os.WriteFile(configPath, source, 0o600)
		})
	if err == nil || called {
		t.Fatalf("JointCommit(tampered marker) admitted: restore=%v", called)
	} else if !errors.Is(err, ErrJointCompensationFailed) {
		t.Fatalf("JointCommit error = %v, want compensation failure", err)
	} else if !errors.Is(err, ErrJointIntervened) {
		t.Fatalf("JointCommit error = %v, want intervention refusal", err)
	}
	kept, err := os.ReadFile(pendingPath(store.root))
	if err != nil {
		t.Fatalf("intervention marker was removed: %v", err)
	}
	decoded, err := decodePendingCommit(kept)
	if err != nil {
		t.Fatalf("decodePendingCommit(kept) error = %v", err)
	}
	if decoded != tampered {
		t.Fatal("failed compensation touched the tampered marker")
	}
}

// An undecodable marker refuses compensation with the marker kept: the
// restore must not run when the intent cannot be established.
func TestJointCommitCompensationRefusesCorruptMarker(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	crowned := crownTrustAtExhaustion(t, store, snapshot)
	marker := jointMarker(configPath, source, replacement, crowned.Generation)
	called := false
	_, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error {
			if err := os.WriteFile(configPath, replacement, 0o600); err != nil {
				return err
			}
			return os.WriteFile(pendingPath(store.root), []byte("{corrupt"), 0o600)
		},
		func(HeldExclusive) error {
			called = true
			return nil
		})
	if err == nil || called {
		t.Fatalf("JointCommit(corrupt marker) admitted: restore=%v", called)
	} else if !errors.Is(err, ErrJointCompensationFailed) {
		t.Fatalf("JointCommit error = %v, want compensation failure", err)
	}
	kept, err := os.ReadFile(pendingPath(store.root))
	if err != nil {
		t.Fatalf("corrupt marker was removed: %v", err)
	}
	if string(kept) != "{corrupt" {
		t.Fatal("failed compensation touched the corrupt marker")
	}
}

// The restore result is verified against the source hash before the marker
// aborts: a restore that reports success without reinstalling the source
// refuses as an intervention with the marker kept.
func TestJointCommitCompensationVerifiesRestore(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	tests := []struct {
		name    string
		restore func(configPath string) error
		want    []byte
	}{
		{"noop restore", func(string) error { return nil }, replacement},
		{"wrong bytes restore", func(configPath string) error {
			return os.WriteFile(configPath, []byte("wrong configuration\n"), 0o600)
		}, []byte("wrong configuration\n")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, configPath, snapshot := jointFixture(t, source, replacement)
			crowned := crownTrustAtExhaustion(t, store, snapshot)
			marker := jointMarker(configPath, source, replacement, crowned.Generation)
			restore := test.restore
			_, err := store.JointCommit(store.paths, marker,
				func(TrustStore) error { return nil },
				func(HeldExclusive) error { return os.WriteFile(configPath, replacement, 0o600) },
				func(HeldExclusive) error { return restore(configPath) })
			if err == nil {
				t.Fatalf("JointCommit(%s) succeeded, want intervention refusal", test.name)
			} else if !errors.Is(err, ErrJointCompensationFailed) {
				t.Fatalf("JointCommit error = %v, want compensation failure", err)
			} else if !errors.Is(err, ErrJointIntervened) {
				t.Fatalf("JointCommit error = %v, want intervention refusal", err)
			}
			installed, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(installed) != string(test.want) {
				t.Fatalf("JointCommit(%s) left %q, want the unrestored bytes", test.name, installed)
			}
			assertMarkerKept(t, store)
		})
	}
}

// A restore that reports failure after writing the source bytes still
// fails closed with the marker kept: the error may mean the write is not
// durable, so the abort must not be claimed.
func TestJointCommitCompensationFailsAfterDurableRestore(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	crowned := crownTrustAtExhaustion(t, store, snapshot)
	marker := jointMarker(configPath, source, replacement, crowned.Generation)
	syncErr := errors.New("restore sync unavailable")
	_, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error { return os.WriteFile(configPath, replacement, 0o600) },
		func(HeldExclusive) error {
			if err := os.WriteFile(configPath, source, 0o600); err != nil {
				return err
			}
			return syncErr
		})
	if err == nil {
		t.Fatal("JointCommit(failing restore) succeeded, want failure")
	} else if !errors.Is(err, ErrJointCompensationFailed) {
		t.Fatalf("JointCommit error = %v, want compensation failure", err)
	} else if !errors.Is(err, syncErr) {
		t.Fatalf("JointCommit error = %v, want the restore failure preserved", err)
	}
	assertMarkerKept(t, store)
}

// When the replacement never landed the compensation aborts without
// running the restore, which would only rewrite identical bytes.
func TestJointCommitCompensationSkipsRestoreWhenSourcePresent(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	crowned := crownTrustAtExhaustion(t, store, snapshot)
	marker := jointMarker(configPath, source, replacement, crowned.Generation)
	called := false
	_, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error { return nil },
		func(HeldExclusive) error {
			called = true
			return nil
		})
	if err == nil || called {
		t.Fatalf("JointCommit(unreplaced) admitted: restore=%v", called)
	} else if !errors.Is(err, ErrJointReplacementDurable) {
		t.Fatalf("JointCommit error = %v, want joint replacement failure", err)
	} else if errors.Is(err, ErrJointCompensationFailed) {
		t.Fatalf("JointCommit error = %v, want a clean abort, not failed compensation", err)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Generation != MaxGeneration {
		t.Fatalf("compensated generation = %d, want the unchanged ceiling", fresh.Generation)
	}
	assertMarkerGone(t, store)
}

// faultMarkerReadFS fails pending-marker content reads once armed, while
// staging and inspection keep working: arming inside the replace closure
// faults exactly the compensation re-read.
type faultMarkerReadFS struct {
	FileSystem
	armed bool
}

func (filesystem *faultMarkerReadFS) ReadFile(name string) ([]byte, error) {
	if filesystem.armed && filepath.Base(name) == pendingCommitFileName {
		return nil, errors.New("joint marker read unavailable")
	}
	return filesystem.FileSystem.ReadFile(name)
}

// An unreadable marker refuses compensation with the marker kept: absence
// is never inferred from a failed inspection, and the restore must not
// run against unestablished intent.
func TestJointCommitCompensationFailsWhenMarkerUnreadable(t *testing.T) {
	source := []byte("source configuration\n")
	replacement := []byte("replacement configuration\n")
	store, configPath, snapshot := jointFixture(t, source, replacement)
	wrapped := &faultMarkerReadFS{FileSystem: store.fs}
	store.fs = wrapped
	crowned := crownTrustAtExhaustion(t, store, snapshot)
	marker := jointMarker(configPath, source, replacement, crowned.Generation)
	called := false
	_, err := store.JointCommit(store.paths, marker,
		func(TrustStore) error { return nil },
		func(HeldExclusive) error {
			wrapped.armed = true
			return os.WriteFile(configPath, replacement, 0o600)
		},
		func(HeldExclusive) error {
			called = true
			return nil
		})
	if err == nil || called {
		t.Fatalf("JointCommit(unreadable marker) admitted: restore=%v", called)
	} else if !errors.Is(err, ErrJointCompensationFailed) {
		t.Fatalf("JointCommit error = %v, want compensation failure", err)
	} else if !errors.Is(err, ErrTrustStoreUnreadable) {
		t.Fatalf("JointCommit error = %v, want unreadable-marker refusal", err)
	} else if !errors.Is(err, ErrGenerationExhausted) {
		t.Fatalf("JointCommit error = %v, want the original bump refusal preserved", err)
	}
	installed, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(replacement) {
		t.Fatal("failed compensation did not leave the replacement for forward recovery")
	}
	assertMarkerKept(t, store)
}

func TestSharedSnapshotSerializesWithExclusive(t *testing.T) {
	store := setupStoreForTest(t)
	release := make(chan struct{})
	entered := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- store.transact(func(*TrustStore) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	observed := make(chan TrustStore, 1)
	readerDone := make(chan error, 1)
	go func() {
		readerDone <- store.WithSharedSnapshot(func(current TrustStore) error {
			observed <- current
			return nil
		})
	}()
	// The shared reader must wait for the exclusive holder: it must not
	// observe state while the transaction is still open.
	select {
	case <-readerDone:
		t.Fatal("WithSharedSnapshot completed while an exclusive transaction was held")
	case <-time.After(250 * time.Millisecond):
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("holder transact error = %v", err)
	}
	if err := <-readerDone; err != nil {
		t.Fatalf("WithSharedSnapshot error = %v", err)
	}
	current := <-observed
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if current.Generation != fresh.Generation {
		t.Fatalf("shared snapshot generation = %d, want %d", current.Generation, fresh.Generation)
	}
}
