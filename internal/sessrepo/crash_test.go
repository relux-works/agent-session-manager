package sessrepo

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/secconftest"
)

// wireCrashPoints connects a repository's write boundaries to the owner's
// crash injector: BeforeWrite fires PointPrepareEnter (before any durable
// byte) and AfterCommit fires PointCommitApply (after the chain index is
// durable). The faults that surface are the owner's typed faults with the
// owner's outcome vocabulary.
func wireCrashPoints(repository *Repository, injector *secconftest.Injector) {
	repository.BeforeWrite = func() error {
		return injector.MaybeFail(secconftest.PointPrepareEnter)
	}
	repository.AfterCommit = func() error {
		return injector.MaybeFail(secconftest.PointCommitApply)
	}
}

// mustCrashFault fails unless err is the injected fault for the armed point
// carrying the expected outcome.
func mustCrashFault(t *testing.T, err error, point string, outcome secconftest.Outcome, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s error = nil, want injected fault at %s", what, point)
	}
	var fault *secconftest.Fault
	if !errors.As(err, &fault) {
		t.Fatalf("%s error = %v, want *secconftest.Fault", what, err)
	}
	if fault.Point() != point {
		t.Fatalf("%s point = %q, want %q", what, fault.Point(), point)
	}
	if fault.Outcome() != outcome {
		t.Fatalf("%s outcome = %q, want %q", what, fault.Outcome(), outcome)
	}
}

func TestCrashBeforeDurableWriteIsSafeRetryOnCreate(t *testing.T) {
	repository := openTestRepository(t)
	injector := &secconftest.Injector{}
	wireCrashPoints(repository, injector)
	injector.Arm(secconftest.PointPrepareEnter)
	_, err := repository.CreateSession([]byte(specRecordExample))
	mustCrashFault(t, err, secconftest.PointPrepareEnter, secconftest.OutcomeSafeRetry, "CreateSession(prepare fault)")
	// Nothing mutated: the session is still unknown.
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrUnknownSession, "GetRecord(after prepare fault)")
	// The identical retry is safe and succeeds.
	reference, err := repository.CreateSession([]byte(specRecordExample))
	if err != nil {
		t.Fatalf("retry CreateSession error = %v", err)
	}
	if reference.SessionID != testSessionID {
		t.Fatalf("retry SessionID = %q", reference.SessionID)
	}
}

func TestCrashBeforeDurableWriteIsSafeRetryOnAppend(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	injector := &secconftest.Injector{}
	wireCrashPoints(repository, injector)
	raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(reference.RecordID)})
	injector.Arm(secconftest.PointPrepareEnter)
	_, err := repository.AppendEvent(testSessionID, raw)
	mustCrashFault(t, err, secconftest.PointPrepareEnter, secconftest.OutcomeSafeRetry, "AppendEvent(prepare fault)")
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("chain after prepare fault = %d events, want 0", len(events))
	}
	first, err := repository.AppendEvent(testSessionID, raw)
	if err != nil {
		t.Fatalf("retry AppendEvent error = %v", err)
	}
	if first.Position != 0 || first.LeaseSequence != 1 {
		t.Fatalf("retry ref = %+v", first)
	}
}

func TestCrashAfterDurableWriteCountsAsCommittedOnCreate(t *testing.T) {
	repository := openTestRepository(t)
	injector := &secconftest.Injector{}
	wireCrashPoints(repository, injector)
	injector.Arm(secconftest.PointCommitApply)
	_, err := repository.CreateSession([]byte(specRecordExample))
	mustCrashFault(t, err, secconftest.PointCommitApply, secconftest.OutcomeRecoverableParked, "CreateSession(commit fault)")
	// The crashed commit already performed the durable write: the record
	// reads back and the operation is terminal, so a retry names the
	// terminal state instead of duplicating it.
	stored, err := repository.GetRecord(testSessionID)
	if err != nil {
		t.Fatalf("GetRecord after crashed commit error = %v", err)
	}
	if !jsonEqual(t, stored, []byte(specRecordExample)) {
		t.Fatal("record after crashed commit differs")
	}
	_, err = repository.CreateSession([]byte(specRecordExample))
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(after crashed commit)")
}

func TestCrashAfterDurableWriteCountsAsCommittedOnAppend(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository)
	injector := &secconftest.Injector{}
	wireCrashPoints(repository, injector)
	raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(reference.RecordID)})
	injector.Arm(secconftest.PointCommitApply)
	_, err := repository.AppendEvent(testSessionID, raw)
	mustCrashFault(t, err, secconftest.PointCommitApply, secconftest.OutcomeRecoverableParked, "AppendEvent(commit fault)")
	var eventID struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(raw, &eventID); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.GetEvent(testSessionID, eventID.EventID)
	if err != nil {
		t.Fatalf("GetEvent after crashed commit error = %v", err)
	}
	if string(stored) != string(raw) {
		t.Fatal("event after crashed commit differs")
	}
	// The lost-response retry carries the identical bytes and receives the
	// identical reference without a second chain entry.
	retry, err := repository.AppendEvent(testSessionID, raw)
	if err != nil {
		t.Fatalf("idempotent retry after crashed commit error = %v", err)
	}
	if retry.EventID != eventID.EventID || retry.Position != 0 {
		t.Fatalf("retry ref = %+v, want the committed position", retry)
	}
	events, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("chain after crashed commit and retry = %d events, want 1", len(events))
	}
	// Rollback past the crashed commit is forbidden by construction: there
	// is no rollback entry, and the committed bytes survive every later
	// refusal on this session.
	_, err = repository.AppendEvent(testSessionID, buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{eventID.EventID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.idle", payload: idlePayload()}))
	mustErrorIs(t, err, ErrSequenceRepeat, "AppendEvent(repeat after parked commit)")
	stored, err = repository.GetEvent(testSessionID, eventID.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(raw) {
		t.Fatal("committed bytes did not survive the later refusal")
	}
}

// armCreateStep wires AfterCreateStep to fire point only at the targeted
// interior step, recording every step reached. The closure — not the
// arm — selects the step, so one armed point proves exactly the targeted
// boundary fired: any other firing order leaves RemainingArmed non-empty
// or the fired prefix wrong.
func armCreateStep(t *testing.T, repository *Repository, target CreateStep, point string) (*secconftest.Injector, *[]CreateStep) {
	t.Helper()
	injector := &secconftest.Injector{}
	injector.Arm(point)
	fired := &[]CreateStep{}
	repository.AfterCreateStep = func(step CreateStep) error {
		*fired = append(*fired, step)
		if step != target {
			return nil
		}
		return injector.MaybeFail(point)
	}
	return injector, fired
}

// mustFiredPrefix fails unless the create reached exactly the expected
// step prefix and the armed point was consumed: a crash point outside
// the window it claims to cover measures nothing.
func mustFiredPrefix(t *testing.T, injector *secconftest.Injector, fired *[]CreateStep, want []CreateStep, what string) {
	t.Helper()
	if remaining := injector.RemainingArmed(); len(remaining) != 0 {
		t.Fatalf("%s left armed points %q; the interior crash never fired", what, remaining)
	}
	if len(*fired) != len(want) {
		t.Fatalf("%s fired steps = %q, want prefix %q", what, *fired, want)
	}
	for index, step := range want {
		if (*fired)[index] != step {
			t.Fatalf("%s fired steps = %q, want prefix %q", what, *fired, want)
		}
	}
}

// TestCreateStepCrashAfterSessionDirResumesOnRetry crashes CreateSession
// between the session directory and the record install — the window the
// first revision left unrecoverable. The parked state is a bare
// directory: per-session reads fail closed, the listing reports the
// session on the per-session parked channel (healthy siblings stay
// listable), and the identical retry resumes the interrupted create
// instead of reporting a session that never existed as existing.
func TestCreateStepCrashAfterSessionDirResumesOnRetry(t *testing.T) {
	repository := openTestRepository(t)
	injector, fired := armCreateStep(t, repository, CreateStepSessionDir, secconftest.PointCommitApply)
	_, err := repository.CreateSession([]byte(specRecordExample))
	mustCrashFault(t, err, secconftest.PointCommitApply, secconftest.OutcomeRecoverableParked, "CreateSession(session-dir fault)")
	mustFiredPrefix(t, injector, fired, []CreateStep{CreateStepSessionDir}, "CreateSession(session-dir fault)")
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "GetRecord(parked bare directory)")
	summaries, err := repository.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions(parked bare directory) error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("ListSessions(parked bare directory) = %d entries, want 1 parked", len(summaries))
	}
	mustParkedEntry(t, summaries[0], testSessionID, retryHintForBareDirectory, "ListSessions(parked bare directory)")
	reference, err := repository.CreateSession([]byte(specRecordExample))
	if err != nil {
		t.Fatalf("retry CreateSession error = %v", err)
	}
	if reference.SessionID != testSessionID {
		t.Fatalf("retry SessionID = %q", reference.SessionID)
	}
	stored, err := repository.GetRecord(testSessionID)
	if err != nil {
		t.Fatalf("GetRecord after resumed create error = %v", err)
	}
	if !jsonEqual(t, stored, []byte(specRecordExample)) {
		t.Fatal("record after resumed create differs")
	}
	summaries, err = repository.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions after resumed create error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].SessionID != testSessionID {
		t.Fatalf("sessions after resumed create = %+v, want the one session", summaries)
	}
	if summaries[0].Parked {
		t.Fatalf("session after resumed create = %+v, want healthy", summaries[0])
	}
}

// TestCreateStepCrashAfterRecordResumesOnRetry crashes CreateSession
// between the record install and the events directory: the record is
// durable, the chain is absent, and the identical retry completes the
// index through the resume path, after which the chain continues.
func TestCreateStepCrashAfterRecordResumesOnRetry(t *testing.T) {
	repository := openTestRepository(t)
	injector, fired := armCreateStep(t, repository, CreateStepRecord, secconftest.PointCommitApply)
	_, err := repository.CreateSession([]byte(specRecordExample))
	mustCrashFault(t, err, secconftest.PointCommitApply, secconftest.OutcomeRecoverableParked, "CreateSession(record fault)")
	mustFiredPrefix(t, injector, fired, []CreateStep{CreateStepSessionDir, CreateStepRecord}, "CreateSession(record fault)")
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "GetRecord(parked record without chain)")
	reference, err := repository.CreateSession([]byte(specRecordExample))
	if err != nil {
		t.Fatalf("retry CreateSession error = %v", err)
	}
	if reference.SessionID != testSessionID {
		t.Fatalf("retry SessionID = %q", reference.SessionID)
	}
	first, _ := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	if first.Position != 0 {
		t.Fatalf("continued ref = %+v, want position 0", first)
	}
}

// TestCreateStepCrashAfterEventsDirResumesOnRetry crashes CreateSession
// between the events directory and the chain index: same parked shape
// as the record fault, and the identical retry completes it.
func TestCreateStepCrashAfterEventsDirResumesOnRetry(t *testing.T) {
	repository := openTestRepository(t)
	injector, fired := armCreateStep(t, repository, CreateStepEventsDir, secconftest.PointCommitApply)
	_, err := repository.CreateSession([]byte(specRecordExample))
	mustCrashFault(t, err, secconftest.PointCommitApply, secconftest.OutcomeRecoverableParked, "CreateSession(events-dir fault)")
	mustFiredPrefix(t, injector, fired, []CreateStep{CreateStepSessionDir, CreateStepRecord, CreateStepEventsDir}, "CreateSession(events-dir fault)")
	_, err = repository.GetRecord(testSessionID)
	mustErrorIs(t, err, ErrChainCorrupt, "GetRecord(parked record without chain)")
	reference, err := repository.CreateSession([]byte(specRecordExample))
	if err != nil {
		t.Fatalf("retry CreateSession error = %v", err)
	}
	if reference.SessionID != testSessionID {
		t.Fatalf("retry SessionID = %q", reference.SessionID)
	}
	stored, err := repository.GetRecord(testSessionID)
	if err != nil {
		t.Fatalf("GetRecord after resumed create error = %v", err)
	}
	if !jsonEqual(t, stored, []byte(specRecordExample)) {
		t.Fatal("record after resumed create differs")
	}
}

// TestCreateStepCrashAfterChainCountsAsCommitted crashes CreateSession
// after the chain index is durable but before the operation returns:
// the crashed commit already committed, so the record reads back and
// the retry names the terminal state instead of duplicating it.
func TestCreateStepCrashAfterChainCountsAsCommitted(t *testing.T) {
	repository := openTestRepository(t)
	injector, fired := armCreateStep(t, repository, CreateStepChain, secconftest.PointCommitApply)
	_, err := repository.CreateSession([]byte(specRecordExample))
	mustCrashFault(t, err, secconftest.PointCommitApply, secconftest.OutcomeRecoverableParked, "CreateSession(chain fault)")
	mustFiredPrefix(t, injector, fired, []CreateStep{CreateStepSessionDir, CreateStepRecord, CreateStepEventsDir, CreateStepChain}, "CreateSession(chain fault)")
	stored, err := repository.GetRecord(testSessionID)
	if err != nil {
		t.Fatalf("GetRecord after crashed commit error = %v", err)
	}
	if !jsonEqual(t, stored, []byte(specRecordExample)) {
		t.Fatal("record after crashed commit differs")
	}
	_, err = repository.CreateSession([]byte(specRecordExample))
	mustErrorIs(t, err, ErrSessionExists, "CreateSession(after crashed commit)")
}
