package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// compensateFaultFileSystem reproduces the CR5 trust-commit fault
// deterministically: after the first configuration replacement rename it
// breaks trust.json custody (0644), so the joint generation bump fails
// while the replacement is durable. The restore rename then waits on
// restoreGate, letting the test stage a racing reader, writer or held
// boundary inside the compensation hold.
type compensateFaultFileSystem struct {
	osMigrationFileSystem
	trustPath   string
	filename    string
	renames     int
	replaced    chan struct{}
	restoreGate chan struct{}
}

func (filesystem *compensateFaultFileSystem) Rename(from, to string) error {
	if to != filesystem.filename {
		return filesystem.osMigrationFileSystem.Rename(from, to)
	}
	filesystem.renames++
	switch filesystem.renames {
	case 1:
		if err := filesystem.osMigrationFileSystem.Rename(from, to); err != nil {
			return err
		}
		close(filesystem.replaced)
		return os.Chmod(filesystem.trustPath, 0o644)
	case 2:
		<-filesystem.restoreGate
		return filesystem.osMigrationFileSystem.Rename(from, to)
	default:
		return filesystem.osMigrationFileSystem.Rename(from, to)
	}
}

func awaitReplacementSeam(t *testing.T, filesystem *compensateFaultFileSystem, opDone chan error) {
	t.Helper()
	select {
	case <-filesystem.replaced:
	case err := <-opDone:
		t.Fatalf("faulted operation returned before the replacement seam: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("replacement seam not reached")
	}
}

// assertBlockedDuringCompensation proves the waiter serializes on the
// compensation hold: it must still be waiting after the settle delay,
// exactly like the existing joint serialization tests.
func assertBlockedDuringCompensation[T any](t *testing.T, done <-chan T, what string) {
	t.Helper()
	select {
	case result := <-done:
		t.Fatalf("%s completed during compensation: %+v", what, result)
	case <-time.After(250 * time.Millisecond):
	}
}

func healCustodyAndReleaseRestore(t *testing.T, filesystem *compensateFaultFileSystem) {
	t.Helper()
	if err := os.Chmod(filesystem.trustPath, 0o600); err != nil {
		t.Fatal(err)
	}
	close(filesystem.restoreGate)
}

func assertCleanAbort(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("faulted operation unexpectedly succeeded")
	} else if !errors.Is(err, hosttrust.ErrJointReplacementDurable) {
		t.Fatalf("faulted operation error = %v, want joint replacement failure", err)
	} else if errors.Is(err, hosttrust.ErrJointCompensationFailed) {
		t.Fatalf("faulted operation error = %v, want a clean abort, not failed compensation", err)
	}
}

func compensationBoundaryRequest(t *testing.T, fixture *v4Fixture, generation uint64) hosttrust.AuthRequest {
	t.Helper()
	var peerLeaf, peerRoot []byte
	for _, entry := range fixture.trust(t).Trust.Entries {
		if entry.HostID.String() == testPeerID {
			peerLeaf = entry.LeafDER
			peerRoot = entry.RootDER
		}
	}
	if len(peerLeaf) == 0 || len(peerRoot) == 0 {
		t.Fatal("peer credential missing from trust")
	}
	return hosttrust.AuthRequest{
		SnapshotGeneration:   generation,
		LocalHostID:          testHostID,
		LocalCredentialID:    fixture.self,
		RemoteLeafDER:        peerLeaf,
		RemoteRootDER:        peerRoot,
		ExpectedRemoteHostID: testPeerID,
		Allowlisted:          []string{testHostID, testPeerID},
		HelloHostID:          testPeerID,
		Now:                  fixture.now,
	}
}

// A trust-commit failure after a durable apply replacement compensates
// under the same exclusive hold: the racing surviving reader blocks on
// the hold and then observes the aborted pair, and the failed writer
// never replaces an admitted configuration without a generation change.
// Port of the CR5 TestReviewerRestoreCannotOverwriteConvergedCommit probe:
// its in-seam convergence attempt is structurally impossible now (it
// would block on the held lock), so the race runs as a goroutine racing
// the gated restore with the same final-state assertion.
func TestApplyV4CompensationCannotOverwriteConvergedCommit(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	before := fixture.trust(t).Generation
	filesystem := &compensateFaultFileSystem{
		trustPath:   fixture.store.TrustPath(),
		filename:    fixture.filename,
		replaced:    make(chan struct{}),
		restoreGate: make(chan struct{}),
	}
	applyDone := make(chan error, 1)
	go func() {
		_, err := applyV4(fixture.inputs, nil, preview, true, fixture.now, filesystem, openStoreForConfig)
		applyDone <- err
	}()
	awaitReplacementSeam(t, filesystem, applyDone)
	type readerResult struct {
		coherent CoherentSnapshot
		err      error
	}
	readerDone := make(chan readerResult, 1)
	go func() {
		coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
		readerDone <- readerResult{coherent: coherent, err: err}
	}()
	assertBlockedDuringCompensation(t, readerDone, "surviving reader")
	healCustodyAndReleaseRestore(t, filesystem)
	assertCleanAbort(t, <-applyDone)
	result := <-readerDone
	if result.err != nil {
		t.Fatalf("surviving reader error = %v", result.err)
	}
	observed, ok := result.coherent.Config.Configuration()
	if !ok {
		t.Fatal("surviving reader has no configuration")
	}
	if observed.SourceVersion != CurrentVersion || result.coherent.Generation != before {
		t.Fatalf("surviving reader converged to version %q generation %d, want %s/%d",
			observed.SourceVersion, result.coherent.Generation, CurrentVersion, before)
	}
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	loaded, ok := coherent.Config.Configuration()
	if !ok {
		t.Fatal("missing config")
	}
	if loaded.SourceVersion != CurrentVersion || coherent.Generation != before {
		t.Fatalf("compensated pair is version %q generation %d, want %s/%d",
			loaded.SourceVersion, coherent.Generation, CurrentVersion, before)
	}
	if !bytes.Equal(coherent.Config.Document(), preview.SourceDocument) {
		t.Fatal("compensated configuration is not the exact source bytes")
	}
	assertMarkerGone(t, fixture.store)
}

// The rollback twin: a trust-commit failure after a durable rollback
// replacement compensates under the same hold, and the racing reader
// observes the aborted (still v4) pair with the unchanged generation.
func TestRollbackV4CompensationCannotOverwriteConvergedCommit(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	before := fixture.trust(t).Generation
	preRollback, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	filesystem := &compensateFaultFileSystem{
		trustPath:   fixture.store.TrustPath(),
		filename:    fixture.filename,
		replaced:    make(chan struct{}),
		restoreGate: make(chan struct{}),
	}
	rollbackDone := make(chan error, 1)
	go func() {
		_, err := rollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true}, filesystem, openStoreForConfig)
		rollbackDone <- err
	}()
	awaitReplacementSeam(t, filesystem, rollbackDone)
	type readerResult struct {
		coherent CoherentSnapshot
		err      error
	}
	readerDone := make(chan readerResult, 1)
	go func() {
		coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
		readerDone <- readerResult{coherent: coherent, err: err}
	}()
	assertBlockedDuringCompensation(t, readerDone, "surviving reader")
	healCustodyAndReleaseRestore(t, filesystem)
	assertCleanAbort(t, <-rollbackDone)
	result := <-readerDone
	if result.err != nil {
		t.Fatalf("surviving reader error = %v", result.err)
	}
	observed, ok := result.coherent.Config.Configuration()
	if !ok {
		t.Fatal("surviving reader has no configuration")
	}
	if observed.SourceVersion != Version4 || result.coherent.Generation != before {
		t.Fatalf("surviving reader converged to version %q generation %d, want 4.0.0/%d",
			observed.SourceVersion, result.coherent.Generation, before)
	}
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(coherent.Config.Document(), preRollback) {
		t.Fatal("compensated rollback did not restore the exact pre-rollback bytes")
	}
	if coherent.Generation != before {
		t.Fatalf("compensated generation = %d, want %d", coherent.Generation, before)
	}
	assertMarkerGone(t, fixture.store)
}

// Another writer racing a failing apply blocks on the compensation hold
// and commits only after the clean abort: the abort leaves the source
// bytes, and the writer's own transaction moves the generation.
func TestApplyV4CompensationSerializesWithWriter(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	before := fixture.trust(t).Generation
	filesystem := &compensateFaultFileSystem{
		trustPath:   fixture.store.TrustPath(),
		filename:    fixture.filename,
		replaced:    make(chan struct{}),
		restoreGate: make(chan struct{}),
	}
	applyDone := make(chan error, 1)
	go func() {
		_, err := applyV4(fixture.inputs, nil, preview, true, fixture.now, filesystem, openStoreForConfig)
		applyDone <- err
	}()
	awaitReplacementSeam(t, filesystem, applyDone)
	revokeDone := make(chan error, 1)
	go func() {
		revokeDone <- fixture.store.Revoke(preview.CredentialID)
	}()
	assertBlockedDuringCompensation(t, revokeDone, "racing revocation")
	healCustodyAndReleaseRestore(t, filesystem)
	assertCleanAbort(t, <-applyDone)
	if err := <-revokeDone; err != nil {
		t.Fatalf("racing revocation error = %v", err)
	}
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(coherent.Config.Document(), preview.SourceDocument) {
		t.Fatal("compensated configuration is not the exact source bytes")
	}
	if coherent.Generation != before+1 {
		t.Fatalf("compensated generation = %d, want the writer's %d", coherent.Generation, before+1)
	}
	assertMarkerGone(t, fixture.store)
}

// The rollback twin: the racing writer commits after the clean abort,
// leaving the pre-rollback bytes with its own generation bump.
func TestRollbackV4CompensationSerializesWithWriter(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	before := fixture.trust(t).Generation
	preRollback, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	filesystem := &compensateFaultFileSystem{
		trustPath:   fixture.store.TrustPath(),
		filename:    fixture.filename,
		replaced:    make(chan struct{}),
		restoreGate: make(chan struct{}),
	}
	rollbackDone := make(chan error, 1)
	go func() {
		_, err := rollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true}, filesystem, openStoreForConfig)
		rollbackDone <- err
	}()
	awaitReplacementSeam(t, filesystem, rollbackDone)
	revokeDone := make(chan error, 1)
	go func() {
		revokeDone <- fixture.store.Revoke(fixture.self)
	}()
	assertBlockedDuringCompensation(t, revokeDone, "racing revocation")
	healCustodyAndReleaseRestore(t, filesystem)
	assertCleanAbort(t, <-rollbackDone)
	if err := <-revokeDone; err != nil {
		t.Fatalf("racing revocation error = %v", err)
	}
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(coherent.Config.Document(), preRollback) {
		t.Fatal("compensated rollback did not restore the exact pre-rollback bytes")
	}
	if coherent.Generation != before+1 {
		t.Fatalf("compensated generation = %d, want the writer's %d", coherent.Generation, before+1)
	}
	assertMarkerGone(t, fixture.store)
}

// A mutation boundary attempted during a failing apply blocks on the
// compensation hold and is admitted after the clean abort: the binding
// is still current because the abort moved nothing.
func TestApplyV4CompensationSerializesWithBoundary(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	before := fixture.trust(t).Generation
	// The boundary request binds here, while the lock is free: any trust
	// read after the replacement seam would block on the joint hold.
	request := compensationBoundaryRequest(t, fixture, before)
	filesystem := &compensateFaultFileSystem{
		trustPath:   fixture.store.TrustPath(),
		filename:    fixture.filename,
		replaced:    make(chan struct{}),
		restoreGate: make(chan struct{}),
	}
	applyDone := make(chan error, 1)
	go func() {
		_, err := applyV4(fixture.inputs, nil, preview, true, fixture.now, filesystem, openStoreForConfig)
		applyDone <- err
	}()
	awaitReplacementSeam(t, filesystem, applyDone)
	boundaryRan := make(chan struct{})
	boundaryDone := make(chan error, 1)
	go func() {
		boundaryDone <- fixture.store.WithMutationAuthorization(request, func() error {
			close(boundaryRan)
			return nil
		})
	}()
	assertBlockedDuringCompensation(t, boundaryDone, "racing mutation boundary")
	healCustodyAndReleaseRestore(t, filesystem)
	assertCleanAbort(t, <-applyDone)
	if err := <-boundaryDone; err != nil {
		t.Fatalf("racing boundary error = %v", err)
	}
	select {
	case <-boundaryRan:
	case <-time.After(30 * time.Second):
		t.Fatal("admitted boundary did not run")
	}
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(coherent.Config.Document(), preview.SourceDocument) {
		t.Fatal("compensated configuration is not the exact source bytes")
	}
	if coherent.Generation != before {
		t.Fatalf("compensated generation = %d, want %d", coherent.Generation, before)
	}
	assertMarkerGone(t, fixture.store)
}

// The rollback twin: the boundary is admitted after the clean abort
// with the pre-rollback bytes and the unchanged generation intact.
func TestRollbackV4CompensationSerializesWithBoundary(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	before := fixture.trust(t).Generation
	preRollback, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	// The boundary request binds here, while the lock is free: any trust
	// read after the replacement seam would block on the joint hold.
	request := compensationBoundaryRequest(t, fixture, before)
	filesystem := &compensateFaultFileSystem{
		trustPath:   fixture.store.TrustPath(),
		filename:    fixture.filename,
		replaced:    make(chan struct{}),
		restoreGate: make(chan struct{}),
	}
	rollbackDone := make(chan error, 1)
	go func() {
		_, err := rollbackV4(fixture.inputs, nil, RollbackV4Options{AcknowledgeNoHostChannelAssurance: true}, filesystem, openStoreForConfig)
		rollbackDone <- err
	}()
	awaitReplacementSeam(t, filesystem, rollbackDone)
	boundaryRan := make(chan struct{})
	boundaryDone := make(chan error, 1)
	go func() {
		boundaryDone <- fixture.store.WithMutationAuthorization(request, func() error {
			close(boundaryRan)
			return nil
		})
	}()
	assertBlockedDuringCompensation(t, boundaryDone, "racing mutation boundary")
	healCustodyAndReleaseRestore(t, filesystem)
	assertCleanAbort(t, <-rollbackDone)
	if err := <-boundaryDone; err != nil {
		t.Fatalf("racing boundary error = %v", err)
	}
	select {
	case <-boundaryRan:
	case <-time.After(30 * time.Second):
		t.Fatal("admitted boundary did not run")
	}
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(coherent.Config.Document(), preRollback) {
		t.Fatal("compensated rollback did not restore the exact pre-rollback bytes")
	}
	if coherent.Generation != before {
		t.Fatalf("compensated generation = %d, want %d", coherent.Generation, before)
	}
	assertMarkerGone(t, fixture.store)
}

// A refused generation bump through the production apply entry aborts
// cleanly: the source bytes are restored under the hold, the generation
// is unchanged, and the original refusal is preserved.
func TestApplyV4CompensatesBumpFailure(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	crownTrustAtExhaustion(t, fixture)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err == nil {
		t.Fatal("ApplyV4(exhausted bump) succeeded, want failure")
	} else if !errors.Is(err, hosttrust.ErrJointReplacementDurable) {
		t.Fatalf("ApplyV4 error = %v, want joint replacement failure", err)
	} else if errors.Is(err, hosttrust.ErrJointCompensationFailed) {
		t.Fatalf("ApplyV4 error = %v, want a clean abort, not failed compensation", err)
	} else if !errors.Is(err, hosttrust.ErrGenerationExhausted) {
		t.Fatalf("ApplyV4 error = %v, want the original bump refusal preserved", err)
	}
	installed, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(installed, preview.SourceDocument) {
		t.Fatal("compensating apply did not restore the source bytes")
	}
	if generation := fixture.trust(t).Generation; generation != hosttrust.MaxGeneration {
		t.Fatalf("compensated generation = %d, want the unchanged ceiling", generation)
	}
	assertMarkerGone(t, fixture.store)
}

// The rollback twin: a refused bump aborts with the pre-rollback bytes
// restored and the generation unchanged.
func TestRollbackV4CompensatesBumpFailure(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if _, err := ApplyV4(fixture.inputs, nil, preview, true, fixture.now); err != nil {
		t.Fatalf("ApplyV4 error = %v", err)
	}
	preRollback, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	crownTrustAtExhaustion(t, fixture)
	options := RollbackV4Options{AcknowledgeNoHostChannelAssurance: true}
	if _, err := RollbackV4(fixture.inputs, nil, options); err == nil {
		t.Fatal("RollbackV4(exhausted bump) succeeded, want failure")
	} else if !errors.Is(err, hosttrust.ErrJointReplacementDurable) {
		t.Fatalf("RollbackV4 error = %v, want joint replacement failure", err)
	} else if errors.Is(err, hosttrust.ErrJointCompensationFailed) {
		t.Fatalf("RollbackV4 error = %v, want a clean abort, not failed compensation", err)
	} else if !errors.Is(err, hosttrust.ErrGenerationExhausted) {
		t.Fatalf("RollbackV4 error = %v, want the original bump refusal preserved", err)
	}
	installed, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(installed, preRollback) {
		t.Fatal("compensating rollback did not restore the pre-rollback bytes")
	}
	if generation := fixture.trust(t).Generation; generation != hosttrust.MaxGeneration {
		t.Fatalf("compensated generation = %d, want the unchanged ceiling", generation)
	}
	assertMarkerGone(t, fixture.store)
}

// The legacy v1/v2/v3 migration path predates the joint barrier and holds
// no authorization lock. It stays fail-closed against outstanding joint
// intent: a legacy replacement beside a marker diverges from the pins,
// and every converged reader refuses instead of admitting the mix.
// A legacy migration beside an unreplaced joint marker converges the
// leftover (abort: the replacement never landed) through its opening
// store transaction, then commits under the exclusive hold. The pair
// stays coherent throughout: no diverged mix is ever admitted.
func TestLegacyMigrateConvergesUnreplacedMarkerThenCommits(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	original := append(minimalValidConfigVersion(scalar.PlatformMacOS, Version1), []byte("\n[terminal]\nbackend = \"tmux\"\n")...)
	if err := os.WriteFile(fixture.filename, original, 0o600); err != nil {
		t.Fatal(err)
	}
	before := fixture.trust(t).Generation
	other := append(append([]byte(nil), original...), 0x20)
	marker := fmt.Sprintf(`{"schema":"urn:ax:schema:host-pending-commit","schema_version":"1.0.0",`+
		`"operation":"apply-v4","config_path":%q,"source_sha256":%q,"replacement_sha256":%q,`+
		`"source_generation":%d,"backup_path":%q}`,
		fixture.filename, hosttrust.SHA256HexOf(original), hosttrust.SHA256HexOf(other),
		before, fixture.filename+".bak.3.0.0")
	if err := os.WriteFile(fixture.store.PendingCommitPath(), []byte(marker), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Migrate(fixture.inputs, nil, MigrationOptions{
		TargetVersion: Version2, GeneratedSummaryUpgradeChoice: "reference_only",
	})
	if err != nil {
		t.Fatalf("Migrate(v1 to v2) error = %v", err)
	}
	if !result.Changed || result.SourceVersion != Version1 || result.TargetVersion != Version2 {
		t.Fatalf("Migrate(v1 to v2) result = %#v", result)
	}
	assertMarkerGone(t, fixture.store)
	coherent, err := LoadCoherent(fixture.inputs, nil, fixture.store)
	if err != nil {
		t.Fatalf("LoadCoherent(converged legacy) err = %v", err)
	}
	final, ok := coherent.Config.Configuration()
	if !ok {
		t.Fatal("missing coherent configuration")
	}
	if final.SourceVersion != Version2 || coherent.Generation != before {
		t.Fatalf("coherent state = %s/generation %d, want %s/generation %d", final.SourceVersion, coherent.Generation, Version2, before)
	}
	if _, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture)); err != nil {
		t.Fatalf("hosttrust.Open(converged legacy) err = %v", err)
	}
}

// A legacy migration selected after a joint replacement landed refuses
// instead of overwriting the committed state: the pre-hold load already
// sees Configuration 4.0.0 above the legacy target. The stranded marker
// is untouched by the refusal and still converges forward on reopen.
func TestLegacyMigrateRefusesAfterReplacementLanded(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	before := fixture.trust(t).Generation
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.filename, preview.Replacement, 0o600); err != nil {
		t.Fatal(err)
	}
	marker := fmt.Sprintf(`{"schema":"urn:ax:schema:host-pending-commit","schema_version":"1.0.0",`+
		`"operation":"apply-v4","config_path":%q,"source_sha256":%q,"replacement_sha256":%q,`+
		`"source_generation":%d,"backup_path":%q}`,
		fixture.filename, hosttrust.SHA256HexOf(preview.SourceDocument), hosttrust.SHA256HexOf(preview.Replacement),
		before, fixture.filename+".bak.3.0.0")
	if err := os.WriteFile(fixture.store.PendingCommitPath(), []byte(marker), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(fixture.inputs, nil, MigrationOptions{TargetVersion: CurrentVersion}); !errors.Is(err, ErrMigrationDowngrade) {
		t.Fatalf("Migrate(beside landed Config4) err = %v, want %v", err, ErrMigrationDowngrade)
	}
	assertMarkerKept(t, fixture.store)
	reopened, err := hosttrust.Open(resolveLocalPathsForFixture(t, fixture))
	if err != nil {
		t.Fatalf("hosttrust.Open(stranded marker) err = %v", err)
	}
	converged, err := LoadCoherent(fixture.inputs, nil, reopened)
	if err != nil {
		t.Fatalf("LoadCoherent(forward) err = %v", err)
	}
	if converged.Generation != before+1 {
		t.Fatalf("converged generation = %d, want %d", converged.Generation, before+1)
	}
	assertMarkerGone(t, fixture.store)
}
