package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
)

// enrollExtraPeer moves the fixture trust generation with a genuine commit:
// a fresh issuer enrolls one more host into the fixture store.
func enrollExtraPeer(t *testing.T, fixture *v4Fixture, hostID string) {
	t.Helper()
	issuerState := t.TempDir()
	issuer, err := hosttrust.Open(resolvedPathsForStoreTest(t, filepath.Join(issuerState, "config.toml"), issuerState))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Initialize(); err != nil {
		t.Fatal(err)
	}
	credential, err := issuer.Issue(hostID, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := issuer.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	material, err := hosttrust.ExportEnrollment(snapshot, credential)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.Enroll(hosttrust.EnrollInput{HostID: material.HostID, LeafDER: material.LeafDER, RootDER: material.RootDER, Authorized: material.Fingerprints, EnrolledAt: fixture.now}); err != nil {
		t.Fatalf("Enroll(%s) error = %v", hostID, err)
	}
}

func TestRevalidateApplyV4AbsentSource(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if err := os.Remove(fixture.filename); err != nil {
		t.Fatal(err)
	}
	if err := revalidateApplyV4(fixture.inputs, nil, fixture.store, preview, fixture.trust(t).Trust, fixture.now); err == nil {
		t.Fatal("revalidateApplyV4(absent source) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationSourceAbsent) {
		t.Fatalf("revalidateApplyV4 error = %v, want ErrMigrationSourceAbsent", err)
	}
}

func TestRevalidateApplyV4StaleVersion(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	if err := revalidateApplyV4(fixture.inputs, nil, fixture.store, preview, fixture.trust(t).Trust, fixture.now); err == nil {
		t.Fatal("revalidateApplyV4(applied source) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4Source) {
		t.Fatalf("revalidateApplyV4 error = %v, want ErrMigrationV4Source", err)
	}
}

func TestRevalidateApplyV4StaleSource(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	// A comment-only edit keeps a valid v3 document with different bytes:
	// the pinned source no longer holds.
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.filename, append(append([]byte(nil), source...), []byte("\n# operator note\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := revalidateApplyV4(fixture.inputs, nil, fixture.store, preview, fixture.trust(t).Trust, fixture.now); err == nil {
		t.Fatal("revalidateApplyV4(changed source) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4StaleSource) {
		t.Fatalf("revalidateApplyV4 error = %v, want ErrMigrationV4StaleSource", err)
	}
}

func TestRevalidateApplyV4StaleGeneration(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	enrollExtraPeer(t, fixture, "0198f4c8-7d40-7e55-8e6f-1234567890ac")
	if err := revalidateApplyV4(fixture.inputs, nil, fixture.store, preview, fixture.trust(t).Trust, fixture.now); err == nil {
		t.Fatal("revalidateApplyV4(moved generation) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4StaleGeneration) {
		t.Fatalf("revalidateApplyV4 error = %v, want ErrMigrationV4StaleGeneration", err)
	}
}

func TestRevalidateApplyV4PreviewMismatch(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	tampered := preview
	tampered.Replacement = append([]byte(nil), preview.Replacement...)
	// Flip one byte in place, keeping length: only the exact preview
	// applies, and a length-only comparison must not admit it.
	tampered.Replacement[0] ^= 0x20
	if err := revalidateApplyV4(fixture.inputs, nil, fixture.store, tampered, fixture.trust(t).Trust, fixture.now); err == nil {
		t.Fatal("revalidateApplyV4(tampered preview) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4PreviewMismatch) {
		t.Fatalf("revalidateApplyV4 error = %v, want ErrMigrationV4PreviewMismatch", err)
	}
}

func rollbackRevalidateFixture(t *testing.T) (*v4Fixture, []byte, V4Preview) {
	t.Helper()
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	// A second valid v3 document, byte-distinct from the fixture source,
	// stands in for confirmed backup bytes in direct revalidation calls.
	backup := append(append([]byte(nil), preview.SourceDocument...), []byte("\n# backup-variant\n")...)
	return fixture, backup, preview
}

func TestRevalidateRollbackV4AbsentSource(t *testing.T) {
	fixture, backup, preview := rollbackRevalidateFixture(t)
	if err := os.Remove(fixture.filename); err != nil {
		t.Fatal(err)
	}
	if err := revalidateRollbackV4(fixture.inputs, nil, preview.SourceDocument, backup, CurrentVersion, preview.SourceGeneration, fixture.trust(t).Trust); err == nil {
		t.Fatal("revalidateRollbackV4(absent source) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationSourceAbsent) {
		t.Fatalf("revalidateRollbackV4 error = %v, want ErrMigrationSourceAbsent", err)
	}
}

func TestRevalidateRollbackV4StaleSource(t *testing.T) {
	fixture, backup, preview := rollbackRevalidateFixture(t)
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.filename, append(append([]byte(nil), source...), []byte("\n# file-variant\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := revalidateRollbackV4(fixture.inputs, nil, preview.SourceDocument, backup, CurrentVersion, preview.SourceGeneration, fixture.trust(t).Trust); err == nil {
		t.Fatal("revalidateRollbackV4(changed source) succeeded, want refusal")
	} else if !errors.Is(err, ErrRollbackV4StaleSource) {
		t.Fatalf("revalidateRollbackV4 error = %v, want ErrRollbackV4StaleSource", err)
	}
}

func TestRevalidateRollbackV4StaleGeneration(t *testing.T) {
	fixture, backup, preview := rollbackRevalidateFixture(t)
	enrollExtraPeer(t, fixture, "0198f4c8-7d40-7e55-8e6f-1234567890ac")
	if err := revalidateRollbackV4(fixture.inputs, nil, preview.SourceDocument, backup, CurrentVersion, preview.SourceGeneration, fixture.trust(t).Trust); err == nil {
		t.Fatal("revalidateRollbackV4(moved generation) succeeded, want refusal")
	} else if !errors.Is(err, ErrRollbackV4StaleGeneration) {
		t.Fatalf("revalidateRollbackV4 error = %v, want ErrRollbackV4StaleGeneration", err)
	}
}

func TestRevalidateRollbackV4BackupGates(t *testing.T) {
	fixture, _, preview := rollbackRevalidateFixture(t)
	current := fixture.trust(t).Trust
	if err := revalidateRollbackV4(fixture.inputs, nil, preview.SourceDocument, []byte("not toml ["), CurrentVersion, preview.SourceGeneration, current); err == nil {
		t.Fatal("revalidateRollbackV4(corrupt backup) succeeded, want refusal")
	} else if !errors.Is(err, ErrRollbackV4Backup) {
		t.Fatalf("revalidateRollbackV4 error = %v, want ErrRollbackV4Backup", err)
	}
	if err := revalidateRollbackV4(fixture.inputs, nil, preview.SourceDocument, preview.Replacement, CurrentVersion, preview.SourceGeneration, current); err == nil {
		t.Fatal("revalidateRollbackV4(v4 backup) succeeded, want refusal")
	} else if !errors.Is(err, ErrRollbackV4Backup) {
		t.Fatalf("revalidateRollbackV4 error = %v, want ErrRollbackV4Backup", err)
	}
}

func TestLoadCoherentPairsConfigWithTrust(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	fixturePaths := resolveLocalPathsForFixture(t, fixture)
	store, err := hosttrust.Open(fixturePaths)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureConfigBinding(fixturePaths); err != nil {
		t.Fatal(err)
	}
	coherent, err := LoadCoherent(fixture.inputs, nil, store)
	if err != nil {
		t.Fatalf("LoadCoherent error = %v", err)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if coherent.Generation != fresh.Generation || len(coherent.Trust.Entries) != len(fresh.Trust.Entries) {
		t.Fatal("coherent trust does not match the committed snapshot")
	}
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(coherent.Config.Document(), source) {
		t.Fatal("coherent configuration does not match the selected file")
	}
	if _, err := LoadCoherent(fixture.inputs, nil, nil); err == nil {
		t.Fatal("LoadCoherent(nil store) succeeded, want refusal")
	}
}

func TestLoadCoherentRefusesStoreWithWrongStateRoot(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	foreignState := t.TempDir()
	foreign, err := hosttrust.Open(resolvedPathsForStoreTest(t, fixture.filename, foreignState))
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
		StateRoot:     canonicalForeignState,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign.Root(), "config-binding.json"), binding, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.filename+".ax-config-binding.json", binding, 0o600); err != nil {
		t.Fatal(err)
	}
	foreign, err = hosttrust.Open(resolvedPathsForStoreTest(t, fixture.filename, foreignState))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := foreign.Initialize(); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCoherent(fixture.inputs, nil, foreign); err == nil {
		t.Fatal("LoadCoherent(foreign state root) succeeded, want refusal")
	} else if !errors.Is(err, hosttrust.ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("LoadCoherent(foreign state root) error = %v, want resource mismatch", err)
	}
}

func TestLoadCoherentSerializesWithJointCommit(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	fixturePaths := resolveLocalPathsForFixture(t, fixture)
	store, err := hosttrust.Open(fixturePaths)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureConfigBinding(fixturePaths); err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	marker := hosttrust.PendingCommit{
		Schema:            "urn:ax:schema:host-pending-commit",
		SchemaVersion:     "1.0.0",
		Operation:         hosttrust.JointApplyV4,
		ConfigPath:        fixture.filename,
		SourceSHA256:      hosttrust.SHA256HexOf(source),
		ReplacementSHA256: hosttrust.SHA256HexOf(append(append([]byte(nil), source...), 0x20)),
		SourceGeneration:  before.Generation,
		BackupPath:        fixture.filename + ".bak.3.0.0",
	}
	release := make(chan struct{})
	entered := make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		_, err := store.JointCommit(fixturePaths, marker,
			func(hosttrust.TrustStore) error { return nil },
			func(hosttrust.HeldExclusive) error {
				close(entered)
				<-release
				return nil
			},
			func(hosttrust.HeldExclusive) error { return errUnexpectedJointRestore })
		holderDone <- err
	}()
	<-entered
	readerDone := make(chan error, 1)
	var observed CoherentSnapshot
	go func() {
		var err error
		observed, err = LoadCoherent(fixture.inputs, nil, store)
		readerDone <- err
	}()
	// The coherent reader must wait for the joint commit holding the
	// exclusive lock: it must not pair pre-commit trust with post-commit
	// configuration or vice versa.
	select {
	case <-readerDone:
		t.Fatal("LoadCoherent completed while a joint commit was held")
	case <-time.After(250 * time.Millisecond):
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("JointCommit error = %v", err)
	}
	if err := <-readerDone; err != nil {
		t.Fatalf("LoadCoherent error = %v", err)
	}
	if observed.Generation != before.Generation+1 {
		t.Fatalf("coherent generation = %d, want %d", observed.Generation, before.Generation+1)
	}
}

func TestPublishedBackupNamesStayExcluded(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	applied, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now)
	if err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	if !hosttrust.ExcludedConfigDirName(filepath.Base(applied.Migration.BackupPath)) {
		t.Fatalf("apply backup %q escapes replication exclusion", applied.Migration.BackupPath)
	}
	rolled, err := RollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true})
	if err != nil {
		t.Fatalf("RollbackV4 error = %v", err)
	}
	if !hosttrust.ExcludedConfigDirName(filepath.Base(rolled.Migration.BackupPath)) {
		t.Fatalf("pre-rollback copy %q escapes replication exclusion", rolled.Migration.BackupPath)
	}
	if hosttrust.ExcludedConfigDirName(filepath.Base(fixture.filename)) {
		t.Fatalf("live configuration %q is excluded from replication", fixture.filename)
	}
}

// Configuration backups are classified copies of configuration bytes: the
// published backup equals the pre-apply source document exactly, carries
// no private-key material, stages owner-only, and matches the replication
// exclusion. No credential file may exist under the configuration
// directory.
func TestApplyV4BackupIsClassifiedConfigCopy(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preloaded, err := Load(fixture.inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := append([]byte(nil), preloaded.Document()...)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	applied, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now)
	if err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	backup, err := os.ReadFile(applied.Migration.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, source) {
		t.Fatal("apply backup differs from the pre-apply source document")
	}
	if bytes.Contains(backup, []byte("PRIVATE KEY")) {
		t.Fatal("apply backup contains private-key material")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(applied.Migration.BackupPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("apply backup mode = %04o, want 0600", info.Mode().Perm())
		}
	} else {
		t.Log("backup mode bits are not asserted on Windows: profile ACLs govern")
	}
	if !hosttrust.ExcludedConfigDirName(filepath.Base(applied.Migration.BackupPath)) {
		t.Fatalf("apply backup %q escapes replication exclusion", applied.Migration.BackupPath)
	}
	entries, err := os.ReadDir(filepath.Dir(fixture.filename))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "private-key.pem", "certificate.pem", "root.pem", "trust.json", "pending-commit.json":
			t.Fatalf("credential file %q published under the configuration directory", entry.Name())
		}
	}
}

// errUnexpectedJointRestore fails any joint commit that compensates: the
// serialization test above never faults the generation bump, so the
// restore must not run.
var errUnexpectedJointRestore = errors.New("joint restore must not run")

func TestCompensationFailurePreservesCommitError(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	crownTrustAtExhaustion(t, fixture)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	// Rename 1 installs the replacement; rename 2 (the in-hold restore)
	// is faulted. The compound failure must preserve the original
	// generation-bump refusal, not just the recovery identity.
	failing := &faultMigrationFileSystem{failRenameCall: 2}
	if _, err := applyV4(fixture.inputs, nil, preview, true, fixture.now, failing, openStoreForConfig); err == nil {
		t.Fatal("ApplyV4(compound durability failure) succeeded, want failure")
	} else if !errors.Is(err, ErrMigrationRecovery) {
		t.Fatalf("ApplyV4(compound) error = %v, want recovery failure", err)
	} else if !errors.Is(err, hosttrust.ErrGenerationExhausted) {
		t.Fatalf("ApplyV4(compound) error = %v, want the original commit error preserved", err)
	} else if !errors.Is(err, hosttrust.ErrJointCompensationFailed) {
		t.Fatalf("ApplyV4(compound) error = %v, want compensation failure", err)
	}
}
