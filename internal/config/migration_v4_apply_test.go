package config

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestApplyV4EmptyPreview(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	if _, err := ApplyV4(fixture.inputs, nil, V4Preview{}, true, fixture.now); err == nil {
		t.Fatal("ApplyV4(empty preview) succeeded, want refusal: only the exact preview applies")
	} else if !errors.Is(err, ErrMigrationV4PreviewMismatch) {
		t.Fatalf("ApplyV4(empty) error = %v", err)
	}
}

func TestApplyV4AbsentSource(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if err := os.Remove(fixture.filename); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err == nil {
		t.Fatal("ApplyV4(absent source) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationSourceAbsent) {
		t.Fatalf("ApplyV4(absent) error = %v", err)
	}
}

func TestApplyV4HappyPath(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	result, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now)
	if err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	if result.Migration.SourceVersion != CurrentVersion || result.Migration.TargetVersion != Version4 {
		t.Fatalf("apply migration = %+v", result.Migration)
	}
	if !result.Migration.Changed || result.Generation != preview.SourceGeneration+1 {
		t.Fatalf("apply result = %+v, want changed and generation %d", result, preview.SourceGeneration+1)
	}
	installed, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(installed, preview.Replacement) {
		t.Fatal("installed configuration is not the exact confirmed preview")
	}
	backup, err := os.ReadFile(result.Migration.BackupPath)
	if err != nil {
		t.Fatalf("backup read error = %v", err)
	}
	if !bytes.Equal(backup, source) {
		t.Fatal("owner-only backup does not hold the source bytes")
	}
	loaded, err := Decode(installed, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	if err != nil {
		t.Fatalf("Decode(installed) error = %v", err)
	}
	if loaded.SourceVersion != Version4 {
		t.Fatalf("installed source = %q, want 4.0.0", loaded.SourceVersion)
	}
	// The same preview cannot apply twice: the source moved.
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err == nil {
		t.Fatal("second ApplyV4(same preview) succeeded, want stale-source refusal")
	} else if !errors.Is(err, ErrMigrationV4Source) && !errors.Is(err, ErrMigrationV4StaleSource) {
		t.Fatalf("second ApplyV4 error = %v", err)
	}
}

func TestApplyV4ConfirmRequired(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, false, fixture.now); err == nil {
		t.Fatal("ApplyV4(unconfirmed) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4ConfirmRequired) {
		t.Fatalf("ApplyV4(unconfirmed) error = %v", err)
	}
	after, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, source) {
		t.Fatal("unconfirmed apply mutated the configuration")
	}
}

func TestApplyV4ConfirmRequiredWithDrops(t *testing.T) {
	// Confirmation is unconditional: even a preview that drops every peer
	// refuses without it.
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, []string{testPeerID}, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4(drop) error = %v", err)
	}
	if len(preview.DroppedPeers) != 1 {
		t.Fatalf("dropped = %v, want one peer", preview.DroppedPeers)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, false, fixture.now); err == nil {
		t.Fatal("ApplyV4(unconfirmed with drops) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4ConfirmRequired) {
		t.Fatalf("ApplyV4(unconfirmed drops) error = %v", err)
	}
}

func TestApplyV4StaleSource(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	snapshot, err := Load(fixture.inputs, nil)
	if err != nil {
		t.Fatal(err)
	}
	loaded, _ := snapshot.Configuration()
	loaded.Value.HostName = "renamed-host"
	context := DecodeContext{RuntimePlatform: scalar.PlatformMacOS}
	renamed, err := EncodeCurrent(loaded.Value, context)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.filename, renamed, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err == nil {
		t.Fatal("ApplyV4(changed source) succeeded, want stale-source refusal")
	} else if !errors.Is(err, ErrMigrationV4StaleSource) {
		t.Fatalf("ApplyV4(changed source) error = %v", err)
	}
}

func TestApplyV4StaleGeneration(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	// Any committed trust change invalidates the preview generation.
	if err := fixture.store.Revoke(preview.CredentialID); err != nil {
		// Revoking the selected credential also moves the generation; either
		// refusal proves the preview no longer authorizes.
		t.Fatal(err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err == nil {
		t.Fatal("ApplyV4(moved generation) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4StaleGeneration) && !errors.Is(err, ErrMigrationV4Credential) {
		t.Fatalf("ApplyV4(moved generation) error = %v", err)
	}
}

func TestApplyV4PreviewMismatch(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	tampered := preview
	tampered.Replacement = append([]byte(nil), preview.Replacement...)
	// Flip one byte inside the host name, keeping length: only the exact
	// preview applies.
	marker := []byte("fixture-host")
	at := bytes.Index(tampered.Replacement, marker)
	if at < 0 {
		t.Fatal("preview has no host name marker")
	}
	tampered.Replacement[at] = 'X'
	if _, err := ApplyV4(fixture.inputs, nil, tampered, true, fixture.now); err == nil {
		t.Fatal("ApplyV4(tampered preview) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4PreviewMismatch) {
		t.Fatalf("ApplyV4(tampered) error = %v", err)
	}
}

func TestApplyV4CustodyFailure(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	hex := preview.CredentialID[len("sha256:"):]
	private := filepath.Join(fixture.stateDir, "host-channel", "credentials", hex, "private-key.pem")
	if err := os.Remove(private); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err == nil {
		t.Fatal("ApplyV4(broken custody) succeeded, want refusal")
	}
}

func TestApplyV4TrustCommitRecovery(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-bit recovery does not bind the superuser")
	}
	fixture := setupV4Fixture(t, testPeerID)
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	before := fixture.trust(t).Generation
	channel := filepath.Join(fixture.stateDir, "host-channel")
	if err := os.Chmod(channel, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(channel, 0o700) }()
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err == nil {
		t.Fatal("ApplyV4(failing trust commit) succeeded, want failure")
	}
	after, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, source) {
		t.Fatal("trust commit failure left the replaced configuration without recovery")
	}
	if err := os.Chmod(channel, 0o700); err != nil {
		t.Fatal(err)
	}
	current, err := fixture.store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if current.Generation != before {
		t.Fatalf("failed apply bumped generation %d to %d", before, current.Generation)
	}
}

func TestApplyV4ConfigWriteFailureKeepsGeneration(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	before := fixture.trust(t).Generation
	failing := &faultMigrationFileSystem{failRenameCall: 1}
	if _, err := applyV4(fixture.inputs, nil, preview, true, fixture.now, failing, openStoreForConfig); err == nil {
		t.Fatal("ApplyV4(failing config replace) succeeded, want failure")
	}
	after, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, source) {
		t.Fatal("failed config replacement changed the source bytes")
	}
	current, err := fixture.store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if current.Generation != before {
		t.Fatalf("failed config replacement bumped generation %d to %d", before, current.Generation)
	}
}

func TestRollbackV4(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	applied, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now)
	if err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	if _, err := RollbackV4(fixture.inputs, nil, RollbackV4Options{}); err == nil {
		t.Fatal("RollbackV4 without acknowledgement succeeded, want refusal")
	} else if !errors.Is(err, ErrRollbackV4AckRequired) {
		t.Fatalf("RollbackV4(no ack) error = %v", err)
	}
	rolled, err := RollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true})
	if err != nil {
		t.Fatalf("RollbackV4 error = %v", err)
	}
	if !rolled.Migration.Changed || rolled.Migration.TargetVersion != CurrentVersion {
		t.Fatalf("rollback = %+v, want changed to %q", rolled.Migration, CurrentVersion)
	}
	if rolled.Generation != applied.Generation+1 {
		t.Fatalf("rollback generation = %d, want %d", rolled.Generation, applied.Generation+1)
	}
	restored, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(restored, source) {
		t.Fatal("rollback did not restore the preserved backup bytes")
	}
	// Rolling back the restored state is a no-op, never a second mutation.
	again, err := RollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true})
	if err != nil {
		t.Fatalf("second RollbackV4 error = %v", err)
	}
	if again.Migration.Changed {
		t.Fatal("second rollback reported a change on identical bytes")
	}
}

// crownTrustAtExhaustion rewrites the fixture trust at the uint53 ceiling:
// the next generation bump is genuinely refused, which faults the joint
// commit after the configuration replacement is already durable.
func crownTrustAtExhaustion(t *testing.T, fixture *v4Fixture) {
	t.Helper()
	snapshot := fixture.trust(t)
	crowned := hosttrust.TrustStore{Generation: hosttrust.MaxGeneration, Entries: snapshot.Trust.Entries}
	document, err := hosttrust.EncodeTrust(crowned)
	if err != nil {
		t.Fatalf("EncodeTrust(exhausted) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixture.stateDir, "host-channel", "trust.json"), document, 0o600); err != nil {
		t.Fatal(err)
	}
}

// crashAfterReplaceFileSystem terminates the process with exit 77
// immediately after the real configuration replacement rename lands. It
// stages a genuine post-replacement crash: the marker is durable, the new
// bytes are installed, and the generation bump never ran.
type crashAfterReplaceFileSystem struct {
	osMigrationFileSystem
	target string
}

func (filesystem crashAfterReplaceFileSystem) Rename(from, to string) error {
	if err := filesystem.osMigrationFileSystem.Rename(from, to); err != nil {
		return err
	}
	if to == filesystem.target {
		os.Exit(77)
	}
	return nil
}

func TestApplyV4CrashConvergesGeneration(t *testing.T) {
	if directory := os.Getenv("AX_V4_CRASH_FIXTURE"); directory != "" {
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
		store, err := hosttrust.Open(resolvedPathsForStoreTest(t, filename, state))
		if err != nil {
			t.Fatal(err)
		}
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
	command := exec.Command(os.Args[0], "-test.run=^TestApplyV4CrashConvergesGeneration$")
	command.Env = append(os.Environ(), "AX_V4_CRASH_FIXTURE="+fixture.directory)
	output, err := command.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 77 {
		t.Fatalf("child did not crash at the replacement seam: %v %s", err, output)
	}
	// Restart recovery must converge the pair: admitting the new Config4
	// with the old generation fails the test. Every read here must
	// succeed: a Load, Configuration, Open or ReadSnapshot error is a
	// failed convergence, not a pass, so each fails explicitly instead of
	// returning silently.
	loaded, err := Load(fixture.inputs, nil)
	if err != nil {
		t.Fatalf("Load after child exit 77 error = %v", err)
	}
	configuration, ok := loaded.Configuration()
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
	if configuration.SourceVersion == Version4 && snapshot.Generation == before {
		t.Fatalf("restart admits new Config4 with old generation %d after child exit 77", before)
	}
	if configuration.SourceVersion != Version4 || snapshot.Generation != before+1 {
		t.Fatalf("restart converged to version %q generation %d, want 4.0.0/%d", configuration.SourceVersion, snapshot.Generation, before+1)
	}
	if _, err := os.Lstat(reopened.PendingCommitPath()); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after restart convergence: %v", err)
	}
}

func TestApplyV4MarkerStagingFailureAborts(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-bit staging does not bind the superuser")
	}
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	// The joint marker cannot stage in an unwritable trust directory: the
	// commit aborts before the configuration is touched, with no marker
	// left behind. The store opens before the directory loses its write
	// bit, so the failure lands in the commit, not in the open.
	channel := filepath.Join(fixture.stateDir, "host-channel")
	preopened, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(channel, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(channel, 0o700) }()
	openHook := func(localstore.ResolvedPaths) (*hosttrust.Store, error) { return preopened, nil }
	source, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := applyV4(fixture.inputs, nil, preview, true, fixture.now, osMigrationFileSystem{}, openHook); err == nil {
		t.Fatal("ApplyV4(unwritable trust dir) succeeded, want failure")
	} else if !errors.Is(err, hosttrust.ErrTrustDurability) {
		t.Fatalf("ApplyV4(staging) error = %v, want durability failure", err)
	}
	after, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, source) {
		t.Fatal("aborted joint commit mutated the configuration")
	}
	if _, err := os.Lstat(filepath.Join(channel, "pending-commit.json")); !os.IsNotExist(err) {
		t.Fatalf("joint marker remains after aborted commit: %v", err)
	}
}

func TestApplyV4TrustCommitRecoveryFailure(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	crownTrustAtExhaustion(t, fixture)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if preview.SourceGeneration != hosttrust.MaxGeneration {
		t.Fatalf("preview generation = %d, want the exhausted ceiling", preview.SourceGeneration)
	}
	// The config replacement succeeds but the generation bump is refused at
	// the uint53 ceiling, and the source restore fails too: the error must
	// report the compound failure, never silent success. Rename 1 installs
	// the replacement; rename 2 (the restore) is faulted.
	failing := &faultMigrationFileSystem{failRenameCall: 2}
	if _, err := applyV4(fixture.inputs, nil, preview, true, fixture.now, failing, openStoreForConfig); err == nil {
		t.Fatal("ApplyV4(compound durability failure) succeeded, want failure")
	} else if !errors.Is(err, ErrMigrationRecovery) {
		t.Fatalf("ApplyV4(compound) error = %v, want recovery failure", err)
	}
	// The replacement stayed durable with the marker kept for operator
	// resolution. Recovery cannot converge it either: the forward bump
	// refuses again at the exhausted ceiling, so the open fails and the
	// marker stays instead of converging silently.
	channel := filepath.Join(fixture.stateDir, "host-channel")
	if _, err := os.Lstat(filepath.Join(channel, "pending-commit.json")); err != nil {
		t.Fatalf("joint marker was removed after failed restore: %v", err)
	}
	if _, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture)); err == nil {
		t.Fatal("openStore(failed restore) succeeded, want refusal")
	} else if !errors.Is(err, hosttrust.ErrGenerationExhausted) {
		t.Fatalf("openStore error = %v, want ErrGenerationExhausted", err)
	}
}

func TestRollbackV4AbsentSource(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	if err := os.Remove(fixture.filename); err != nil {
		t.Fatal(err)
	}
	options := RollbackV4Options{AcknowledgeNoHostChannelAssurance: true}
	if _, err := RollbackV4(fixture.inputs, nil, options); err == nil {
		t.Fatal("RollbackV4(absent source) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationSourceAbsent) {
		t.Fatalf("RollbackV4(absent) error = %v", err)
	}
}

func TestRollbackV4TrustCommitRecoveryFailure(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	// Fault the joint commit after the backup replacement is durable: crown
	// the trust at the uint53 ceiling and fault the source restore.
	// Rename 1 installs the backup; rename 2 (the restore) is faulted.
	crownTrustAtExhaustion(t, fixture)
	failing := &faultMigrationFileSystem{failRenameCall: 2}
	options := RollbackV4Options{AcknowledgeNoHostChannelAssurance: true}
	if _, err := rollbackV4(fixture.inputs, nil, options, failing, openStoreForConfig); err == nil {
		t.Fatal("RollbackV4(compound durability failure) succeeded, want failure")
	} else if !errors.Is(err, ErrMigrationRecovery) {
		t.Fatalf("RollbackV4(compound) error = %v, want recovery failure", err)
	}
	channel := filepath.Join(fixture.stateDir, "host-channel")
	if _, err := os.Lstat(filepath.Join(channel, "pending-commit.json")); err != nil {
		t.Fatalf("joint marker was removed after failed restore: %v", err)
	}
}

func TestRollbackV4BackupGates(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	// A missing backup refuses: rollback replaces only from preserved bytes.
	if _, err := RollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true, BackupPath: filepath.Join(fixture.directory, "absent.bak")}); err == nil {
		t.Fatal("RollbackV4(missing backup) succeeded, want refusal")
	} else if !errors.Is(err, ErrRollbackV4Backup) {
		t.Fatalf("RollbackV4(missing backup) error = %v", err)
	}
	// A Config-4 document is never a rollback target.
	if _, err := RollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true, BackupPath: fixture.filename}); err == nil {
		t.Fatal("RollbackV4(v4 backup) succeeded, want refusal")
	} else if !errors.Is(err, ErrRollbackV4Backup) {
		t.Fatalf("RollbackV4(v4 backup) error = %v", err)
	}
	// Undecodable bytes are never a rollback target either.
	corrupt := filepath.Join(fixture.directory, "corrupt.bak")
	if err := os.WriteFile(corrupt, []byte("not toml ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true, BackupPath: corrupt}); err == nil {
		t.Fatal("RollbackV4(corrupt backup) succeeded, want refusal")
	} else if !errors.Is(err, ErrRollbackV4Backup) {
		t.Fatalf("RollbackV4(corrupt backup) error = %v", err)
	}
}
