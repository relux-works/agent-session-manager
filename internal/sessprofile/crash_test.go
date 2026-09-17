package sessprofile

import (
	"bytes"
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/secconftest"
)

// The set-profile transaction mutates only through the append
// path, so the repository's BeforeWrite/AfterCommit boundaries are
// its crash windows unchanged. Both tests below wire those hooks
// to the owner's injector directly.

func TestSetProfileCrashBeforeDurableWriteAdmitsNothing(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	injector := &secconftest.Injector{}
	repository.BeforeWrite = func() error {
		return injector.MaybeFail(secconftest.PointPrepareEnter)
	}
	repository.AfterCommit = func() error {
		return injector.MaybeFail(secconftest.PointCommitApply)
	}
	injector.Arm(secconftest.PointPrepareEnter)
	transactor := &Transactor{Repo: repository}
	_, err := transactor.SetProfile(validSetProfileRequest())
	var fault *secconftest.Fault
	if !errors.As(err, &fault) {
		t.Fatalf("SetProfile(prepare fault) error = %v, want *secconftest.Fault", err)
	}
	if fault.Point() != secconftest.PointPrepareEnter || fault.Outcome() != secconftest.OutcomeSafeRetry {
		t.Fatalf("fault = %s/%s, want prepare/safe_retry", fault.Point(), fault.Outcome())
	}
	// No partial event is admitted: the chain is still empty.
	indexed, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatalf("ListEvents error = %v", err)
	}
	if len(indexed) != 0 {
		t.Fatalf("chain after prepare fault = %d events, want 0", len(indexed))
	}
	// The identical retry appends exactly one event.
	result, err := transactor.SetProfile(validSetProfileRequest())
	if err != nil {
		t.Fatalf("retry SetProfile error = %v", err)
	}
	if result.PreviousProfile != ProfileStandard || result.NewProfile != ProfileYOLO {
		t.Fatalf("retry result = %+v", result)
	}
	indexed, err = repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatalf("ListEvents error = %v", err)
	}
	if len(indexed) != 1 || indexed[0].EventID != result.EventID {
		t.Fatalf("chain after retry = %+v, want exactly the committed change", indexed)
	}
}

func TestSetProfileCrashAfterDurableWriteReplaysOnRetry(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	injector := &secconftest.Injector{}
	repository.BeforeWrite = func() error {
		return injector.MaybeFail(secconftest.PointPrepareEnter)
	}
	repository.AfterCommit = func() error {
		return injector.MaybeFail(secconftest.PointCommitApply)
	}
	injector.Arm(secconftest.PointCommitApply)
	transactor := &Transactor{Repo: repository}
	_, err := transactor.SetProfile(validSetProfileRequest())
	var fault *secconftest.Fault
	if !errors.As(err, &fault) {
		t.Fatalf("SetProfile(commit fault) error = %v, want *secconftest.Fault", err)
	}
	if fault.Point() != secconftest.PointCommitApply || fault.Outcome() != secconftest.OutcomeRecoverableParked {
		t.Fatalf("fault = %s/%s, want commit/recoverable_parked_state", fault.Point(), fault.Outcome())
	}
	// The crashed commit already performed the durable write: the
	// change reads back and the operation is terminal.
	indexed, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatalf("ListEvents error = %v", err)
	}
	if len(indexed) != 1 {
		t.Fatalf("chain after crashed commit = %d events, want 1", len(indexed))
	}
	stored, err := repository.GetEvent(testSessionID, indexed[0].EventID)
	if err != nil {
		t.Fatalf("GetEvent error = %v", err)
	}
	// The identical retry replays the committed bytes instead of
	// duplicating the change or refusing it as unchanged.
	replayed, err := transactor.SetProfile(validSetProfileRequest())
	if err != nil {
		t.Fatalf("retry SetProfile error = %v", err)
	}
	if replayed.EventID != indexed[0].EventID || !bytes.Equal(replayed.Event, stored) {
		t.Fatal("retry did not replay the committed change")
	}
	indexed, err = repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatalf("ListEvents error = %v", err)
	}
	if len(indexed) != 1 {
		t.Fatalf("chain after replay = %d events, want 1", len(indexed))
	}
}
