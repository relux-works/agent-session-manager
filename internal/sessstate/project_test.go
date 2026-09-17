package sessstate

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/secconftest"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// armCreateStep wires AfterCreateStep to fire the armed point only at
// the targeted interior create step, recording every step reached.
func armCreateStep(t *testing.T, repository *sessrepo.Repository, target sessrepo.CreateStep, point string) *secconftest.Injector {
	t.Helper()
	injector := &secconftest.Injector{}
	injector.Arm(point)
	repository.AfterCreateStep = func(step sessrepo.CreateStep) error {
		if step != target {
			return nil
		}
		return injector.MaybeFail(point)
	}
	return injector
}

// mustParkedProjection requires the parked lifecycle state with its
// blocking reason and retry: a parked session is a state, never an
// error or an omission.
func mustParkedProjection(t *testing.T, projection Projection, sessionID, retryFragment, what string) {
	t.Helper()
	if projection.State != StateParked {
		t.Fatalf("%s state = %q, want parked", what, projection.State)
	}
	if projection.Parked == nil {
		t.Fatalf("%s carries no parked facts", what)
	}
	if !strings.HasPrefix(projection.Parked.BlockingReason, "recoverable_parked_state:") {
		t.Fatalf("%s blocking reason = %q, want the recoverable_parked_state channel", what, projection.Parked.BlockingReason)
	}
	if !strings.Contains(projection.Parked.RetryHint, retryFragment) {
		t.Fatalf("%s retry hint = %q, want fragment %q", what, projection.Parked.RetryHint, retryFragment)
	}
	if projection.SessionID != sessionID {
		t.Fatalf("%s session = %q, want %q", what, projection.SessionID, sessionID)
	}
}

// TestProjectDerivesParkedBareDirectory crashes the create between the
// session directory and the record install, then requires Project to
// return the parked state with the bare-directory retry.
func TestProjectDerivesParkedBareDirectory(t *testing.T) {
	repository, err := openRepo(t)
	if err != nil {
		t.Fatalf("open repo error = %v", err)
	}
	record := buildRecord(t, nil)
	injector := armCreateStep(t, repository, sessrepo.CreateStepSessionDir, secconftest.PointCommitApply)
	if _, err := repository.CreateSession(record); err == nil {
		t.Fatalf("CreateSession(session-dir fault) = nil")
	}
	if remaining := injector.RemainingArmed(); len(remaining) != 0 {
		t.Fatalf("armed points %q never fired; the crash sat outside the window", remaining)
	}
	projection, err := (&Projector{Repo: repository}).Project(testSessionID)
	if err != nil {
		t.Fatalf("Project(parked bare directory) error = %v, want the parked state", err)
	}
	mustParkedProjection(t, projection, testSessionID, "for this session ID", "Project(parked bare directory)")
	// The identical retry heals: the parked state resumes into a
	// creating session, proving idempotent recovery past the fault.
	repository.AfterCreateStep = nil
	healed, err := repository.CreateSession(record)
	if err != nil {
		t.Fatalf("CreateSession(retry past bare directory) error = %v", err)
	}
	recovered, err := (&Projector{Repo: repository}).Project(healed.SessionID)
	if err != nil {
		t.Fatalf("Project(recovered) error = %v", err)
	}
	if recovered.State != StateCreating {
		t.Fatalf("recovered state = %q, want creating", recovered.State)
	}
}

// TestProjectDerivesParkedRecordWithoutChain crashes the create
// between the record install and the chain index, then requires the
// parked state with the byte-identical retry.
func TestProjectDerivesParkedRecordWithoutChain(t *testing.T) {
	repository, err := openRepo(t)
	if err != nil {
		t.Fatalf("open repo error = %v", err)
	}
	record := buildRecord(t, nil)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	injector := armCreateStep(t, repository, sessrepo.CreateStepRecord, secconftest.PointCommitApply)
	if _, err := repository.CreateSession(record); err == nil {
		t.Fatalf("CreateSession(record fault) = nil")
	}
	if remaining := injector.RemainingArmed(); len(remaining) != 0 {
		t.Fatalf("armed points %q never fired; the crash sat outside the window", remaining)
	}
	projection, err := (&Projector{Repo: repository}).Project(testSessionID)
	if err != nil {
		t.Fatalf("Project(parked record) error = %v, want the parked state", err)
	}
	mustParkedProjection(t, projection, testSessionID, "byte-identical", "Project(parked record without chain)")
	if projection.RecordID != decoded.RecordID || projection.Name != "payments-api" || projection.Provider.ID != "codex" {
		t.Fatalf("parked identity = %+v, want the parked record's identity", projection)
	}
	repository.AfterCreateStep = nil
	healed, err := repository.CreateSession(record)
	if err != nil {
		t.Fatalf("CreateSession(retry past record without chain) error = %v", err)
	}
	recovered, err := (&Projector{Repo: repository}).Project(healed.SessionID)
	if err != nil {
		t.Fatalf("Project(recovered) error = %v", err)
	}
	if recovered.State != StateCreating || recovered.RecordID != decoded.RecordID {
		t.Fatalf("recovered = %+v, want creating under the parked record", recovered)
	}
}

// TestProjectDerivesParkedEventsDirBoundary crashes the create between
// the events directory and the chain index, then requires the parked
// record-without-chain state with the byte-identical retry — and the
// retry healing into creating.
func TestProjectDerivesParkedEventsDirBoundary(t *testing.T) {
	repository, err := openRepo(t)
	if err != nil {
		t.Fatalf("open repo error = %v", err)
	}
	record := buildRecord(t, nil)
	injector := armCreateStep(t, repository, sessrepo.CreateStepEventsDir, secconftest.PointCommitApply)
	if _, err := repository.CreateSession(record); err == nil {
		t.Fatalf("CreateSession(events-dir fault) = nil")
	}
	if remaining := injector.RemainingArmed(); len(remaining) != 0 {
		t.Fatalf("armed points %q never fired; the crash sat outside the window", remaining)
	}
	projection, err := (&Projector{Repo: repository}).Project(testSessionID)
	if err != nil {
		t.Fatalf("Project(parked events-dir boundary) error = %v, want the parked state", err)
	}
	mustParkedProjection(t, projection, testSessionID, "byte-identical", "Project(parked events-dir boundary)")
	repository.AfterCreateStep = nil
	healed, err := repository.CreateSession(record)
	if err != nil {
		t.Fatalf("CreateSession(retry past events-dir boundary) error = %v", err)
	}
	recovered, err := (&Projector{Repo: repository}).Project(healed.SessionID)
	if err != nil {
		t.Fatalf("Project(recovered) error = %v", err)
	}
	if recovered.State != StateCreating {
		t.Fatalf("recovered state = %q, want creating", recovered.State)
	}
}

// TestProjectRecoversPastChainBoundary crashes the create after the
// chain index is durable: nothing is parked because nothing is
// missing. A retry refuses session-exists — the session is complete,
// not absent — and Project derives creating over it. Together the
// three assertions prove the fault past the last durable step tore
// nothing.
func TestProjectRecoversPastChainBoundary(t *testing.T) {
	repository, err := openRepo(t)
	if err != nil {
		t.Fatalf("open repo error = %v", err)
	}
	record := buildRecord(t, nil)
	injector := armCreateStep(t, repository, sessrepo.CreateStepChain, secconftest.PointCommitApply)
	if _, err := repository.CreateSession(record); err == nil {
		t.Fatalf("CreateSession(chain fault) = nil")
	}
	if remaining := injector.RemainingArmed(); len(remaining) != 0 {
		t.Fatalf("armed points %q never fired; the crash sat outside the window", remaining)
	}
	repository.AfterCreateStep = nil
	if _, err := repository.CreateSession(record); err == nil {
		t.Fatalf("CreateSession(retry past chain boundary) = nil, want session-exists for the completed session")
	}
	projection, err := (&Projector{Repo: repository}).Project(testSessionID)
	if err != nil {
		t.Fatalf("Project(recovered) error = %v", err)
	}
	if projection.State != StateCreating {
		t.Fatalf("recovered state = %q, want creating", projection.State)
	}
}

// TestProjectDerivesParkedTornStore tears the chain index past the
// create window, then requires the parked state with the operator
// remedy: no CreateSession retry heals it.
func TestProjectDerivesParkedTornStore(t *testing.T) {
	root := t.TempDir()
	repository, err := sessrepo.Open(root)
	if err != nil {
		t.Fatalf("open repo error = %v", err)
	}
	record := buildRecord(t, nil)
	reference := createSession(t, repository, record)
	chainPath := filepath.Join(root, "sessions", reference.SessionID, "chain.json")
	if err := os.WriteFile(chainPath, []byte("{torn"), 0o600); err != nil {
		t.Fatalf("tear chain index: %v", err)
	}
	projection, err := (&Projector{Repo: repository}).Project(reference.SessionID)
	if err != nil {
		t.Fatalf("Project(torn store) error = %v, want the parked state", err)
	}
	mustParkedProjection(t, projection, reference.SessionID, "operator remedy", "Project(torn store)")
}

// TestProjectRefusesUnknownSession requires an ID with no listing
// entry to refuse unknown — telling a parked name (entry present)
// apart from a missing one (entry absent) with no sessrepo interface
// change.
func TestProjectRefusesUnknownSession(t *testing.T) {
	repository, err := openRepo(t)
	if err != nil {
		t.Fatalf("open repo error = %v", err)
	}
	_, err = (&Projector{Repo: repository}).Project(testSessionID)
	mustErrorIs(t, err, ErrUnknownSession, "Project(unknown)")
}

// TestProjectDerivesEmptyChainAsCreating requires a freshly created
// session with no events to project creating: the record exists and
// bootstrap is in progress.
func TestProjectDerivesEmptyChainAsCreating(t *testing.T) {
	repository, err := openRepo(t)
	if err != nil {
		t.Fatalf("open repo error = %v", err)
	}
	record := buildRecord(t, nil)
	reference := createSession(t, repository, record)
	projection, err := (&Projector{Repo: repository}).Project(reference.SessionID)
	if err != nil {
		t.Fatalf("Project error = %v", err)
	}
	if projection.State != StateCreating {
		t.Fatalf("state = %q, want creating", projection.State)
	}
	if projection.Provider.ID != "codex" || projection.Name != "payments-api" {
		t.Fatalf("identity = %+v, want the record identity", projection.Provider)
	}
	if projection.HasCheckpoint || projection.HasTerminal {
		t.Fatalf("projection = %+v, want no checkpoint or terminal yet", projection)
	}
}

// TestProjectIsIdempotentAcrossReads requires two projections of the
// same durable state to compare equal: the projector mutates nothing,
// so idempotency is purity, pinned here instead of through a crash
// window the read path does not own.
func TestProjectIsIdempotentAcrossReads(t *testing.T) {
	record := buildRecord(t, nil)
	builder := openChain(t, record)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(decoded.RecordID)})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "provider.launched", payload: launchedPayload("codex")})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 3, eventType: "session.idle", payload: idlePayload()})
	projector := &Projector{Repo: builder.repo, LocalHostID: testHostID}
	first, err := projector.Project(builder.sessionID)
	if err != nil {
		t.Fatalf("Project(first) error = %v", err)
	}
	second, err := projector.Project(builder.sessionID)
	if err != nil {
		t.Fatalf("Project(second) error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Project is not idempotent:\nfirst = %+v\nsecond = %+v", first, second)
	}
	if first.State != StateIdle || first.LocalRole != "owner" {
		t.Fatalf("projection = %+v, want idle owner", first)
	}
}

// TestProjectDerivesLocalStale requires the local projection to go
// stale when its lease loses: the winner stands idle under lease B
// while the local host still acts under lease A, with the
// stale_process warning raised.
func TestProjectDerivesLocalStale(t *testing.T) {
	record := buildRecord(t, nil)
	builder := openChain(t, record)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(decoded.RecordID)})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "provider.launched", payload: launchedPayload("codex")})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 3, eventType: "session.idle", payload: idlePayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 4, eventType: "checkpoint.created", payload: checkpointPayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 5, eventType: "checkpoint.created", payload: checkpointPayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 6, eventType: "session.stopped", payload: stoppedPayload()})
	builder.append(eventOptions{epoch: 2, leaseID: testLeaseIDB, sequence: 1, eventType: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)})
	builder.append(eventOptions{epoch: 2, leaseID: testLeaseIDB, sequence: 2, eventType: "session.resumed", payload: resumedPayload()})
	builder.append(eventOptions{epoch: 2, leaseID: testLeaseIDB, sequence: 3, eventType: "session.idle", payload: idlePayload()})

	winner, err := (&Projector{Repo: builder.repo}).Project(builder.sessionID)
	if err != nil {
		t.Fatalf("Project(winner) error = %v", err)
	}
	if winner.State != StateIdle {
		t.Fatalf("winner state = %q, want idle", winner.State)
	}
	local, err := (&Projector{
		Repo:        builder.repo,
		LocalHostID: testHostIDB,
		Local:       LeaseHead{Epoch: 1, LeaseID: testLeaseID},
	}).Project(builder.sessionID)
	if err != nil {
		t.Fatalf("Project(local) error = %v", err)
	}
	if local.State != StateStale {
		t.Fatalf("local state = %q, want stale", local.State)
	}
	assertWarning(t, local, "stale_process")
	if local.Winner != (LeaseHead{Epoch: 2, LeaseID: testLeaseIDB}) {
		t.Fatalf("local winner = %+v, want epoch 2 lease B", local.Winner)
	}
}

// TestProjectLeavesStoppedPastTakeoverUnstaled requires the stale
// override to stay transition-gated: a stopped winner with a losing
// local lease keeps stopped (stopped has no edge to stale) while the
// winner still reports the new lease.
func TestProjectLeavesStoppedPastTakeoverUnstaled(t *testing.T) {
	record := buildRecord(t, nil)
	builder := openChain(t, record)
	decoded, err := DecodeRecord(record)
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(decoded.RecordID)})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 2, eventType: "provider.launched", payload: launchedPayload("codex")})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 3, eventType: "session.idle", payload: idlePayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 4, eventType: "checkpoint.created", payload: checkpointPayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 5, eventType: "checkpoint.created", payload: checkpointPayload()})
	builder.append(eventOptions{epoch: 1, leaseID: testLeaseID, sequence: 6, eventType: "session.stopped", payload: stoppedPayload()})
	local, err := (&Projector{
		Repo:  builder.repo,
		Local: LeaseHead{Epoch: 1, LeaseID: testLeaseID},
		Union: []LeaseHead{{Epoch: 2, LeaseID: testLeaseIDB}},
	}).Project(builder.sessionID)
	if err != nil {
		t.Fatalf("Project error = %v", err)
	}
	if local.State != StateStopped {
		t.Fatalf("state = %q, want stopped: no table edge carries it to stale", local.State)
	}
	if local.Winner != (LeaseHead{Epoch: 2, LeaseID: testLeaseIDB}) {
		t.Fatalf("winner = %+v, want the union winner", local.Winner)
	}
}

// TestProjectRefusesWithoutRepository requires a nil repository to
// refuse instead of panicking.
func TestProjectRefusesWithoutRepository(t *testing.T) {
	_, err := (&Projector{}).Project(testSessionID)
	mustErrorIs(t, err, ErrDerivation, "Project(nil repo)")
}
