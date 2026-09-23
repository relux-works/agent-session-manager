package sessprofile

import (
	"bytes"
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

const (
	profileAdmissionLosingLease = "cccccccc-dddd-4eee-8fff-111111111111"
	profileAdmissionLowerLease  = "11111111-2222-4333-8444-555555555555"
)

func validSetProfileRequest() SetProfileRequest {
	return SetProfileRequest{
		SessionID:       testSessionID,
		To:              ProfileYOLO,
		Confirmed:       true,
		LeaseEpoch:      1,
		LeaseID:         testLeaseID,
		CreatedByHostID: testHostID,
		CreatedAt:       testChangedAt,
	}
}

func TestSetProfileAppendsFirstChangeOnEmptyChain(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	transactor := &Transactor{Repo: repository}
	result, err := transactor.SetProfile(validSetProfileRequest())
	if err != nil {
		t.Fatalf("SetProfile error = %v", err)
	}
	if result.SessionID != testSessionID || result.PreviousProfile != ProfileStandard || result.NewProfile != ProfileYOLO {
		t.Fatalf("SetProfile result = %+v", result)
	}
	if result.EventID == "" || len(result.Event) == 0 {
		t.Fatalf("SetProfile result carries no event: %+v", result)
	}
	if result.Ref.LeaseSequence != 1 || result.Ref.Position != 0 {
		t.Fatalf("SetProfile ref = %+v, want sequence 1 position 0", result.Ref)
	}
	event, err := DecodeEvent(result.Event)
	if err != nil {
		t.Fatalf("DecodeEvent(committed) error = %v", err)
	}
	if event.ID != result.EventID || len(event.Predecessors) != 1 || event.Predecessors[0] != reference.RecordID {
		t.Fatalf("committed predecessors = %v, want exactly the record", event.Predecessors)
	}
	pair, err := (&Projector{Repo: repository}).Project(testSessionID)
	if err != nil {
		t.Fatalf("Project error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: result.EventID, HasSource: true}, "Project(after set-profile)")
}

func TestSetProfileAppendsUnderChainHead(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	second := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "provider.launched", launchedPayload("codex", "0.147.0", ProfileStandard, "", ProfileStandard))
	transactor := &Transactor{Repo: repository}
	result, err := transactor.SetProfile(validSetProfileRequest())
	if err != nil {
		t.Fatalf("SetProfile error = %v", err)
	}
	if result.PreviousProfile != ProfileStandard || result.NewProfile != ProfileYOLO {
		t.Fatalf("SetProfile result = %+v", result)
	}
	if result.Ref.LeaseSequence != 3 || result.Ref.Position != 2 {
		t.Fatalf("SetProfile ref = %+v, want sequence 3 position 2", result.Ref)
	}
	event, err := DecodeEvent(result.Event)
	if err != nil {
		t.Fatalf("DecodeEvent(committed) error = %v", err)
	}
	if len(event.Predecessors) != 1 || event.Predecessors[0] != second {
		t.Fatalf("committed predecessors = %v, want exactly the tail", event.Predecessors)
	}
}

// TestSetProfileRefusesSupersededLeaseWhileTailStillMatches drives the
// sessrepo admission gate through the composing profile writer: successor B
// is stored while the chain tail remains under A, then SetProfile's A event
// must be refused instead of extending that still-matching tail.
func TestSetProfileRefusesSupersededLeaseWhileTailStillMatches(t *testing.T) {
	t.Run("lower epoch", func(t *testing.T) {
		repository := openTestRepository(t)
		reference := createTestSessionWithoutLease(t, repository, testSessionID, "payments-api", ProfileStandard)
		first, err := repository.CreateLease(testSessionID, sessrepo.CreateLeaseInput{
			LeaseID: testLeaseID, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testCreatedAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		tail := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
		if _, err := repository.CompareAndSwapLease(testSessionID, sessrepo.LeaseExpectation{RecordID: first.RecordID}, sessrepo.SuccessorLeaseInput{
			CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: testLeaseIDB, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testCreatedAt},
			Reason:           "graceful_takeover", CheckpointID: zeroDigest,
		}); err != nil {
			t.Fatal(err)
		}
		request := validSetProfileRequest()
		_, err = (&Transactor{Repo: repository}).SetProfile(request)
		if err == nil || !errors.Is(err, sessrepo.ErrStaleLease) {
			t.Fatalf("SetProfile(superseded lease) error = %v, want stale lease refusal", err)
		}
		events, err := repository.ListEvents(testSessionID)
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 1 || events[0].EventID != tail {
			t.Fatalf("chain after refused SetProfile = %+v, want the original tail only", events)
		}
	})

	for _, losingLeaseID := range []string{profileAdmissionLosingLease, profileAdmissionLowerLease} {
		t.Run("same epoch loser "+losingLeaseID, func(t *testing.T) {
			repository := openTestRepository(t)
			reference := createTestSessionWithoutLease(t, repository, testSessionID, "payments-api", ProfileStandard)
			tail := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 2, losingLeaseID, 1, "session.created", createdPayload(reference.RecordID))
			first, err := repository.CreateLease(testSessionID, sessrepo.CreateLeaseInput{
				LeaseID: testLeaseID, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testCreatedAt,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := repository.CompareAndSwapLease(testSessionID, sessrepo.LeaseExpectation{RecordID: first.RecordID}, sessrepo.SuccessorLeaseInput{
				CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: testLeaseIDB, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testCreatedAt},
				Reason:           "graceful_takeover", CheckpointID: zeroDigest,
			}); err != nil {
				t.Fatal(err)
			}
			request := validSetProfileRequest()
			request.LeaseEpoch = 2
			request.LeaseID = losingLeaseID
			_, err = (&Transactor{Repo: repository}).SetProfile(request)
			if err == nil || !errors.Is(err, sessrepo.ErrDivergentBranch) {
				t.Fatalf("SetProfile(same-epoch loser) error = %v, want divergent branch refusal", err)
			}
			events, err := repository.ListEvents(testSessionID)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || events[0].EventID != tail {
				t.Fatalf("chain after refused same-epoch SetProfile = %+v, want the original tail only", events)
			}
		})
	}
}

func TestSetProfileDerivesFromFromEffectivePair(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	transactor := &Transactor{Repo: repository}
	first, err := transactor.SetProfile(validSetProfileRequest())
	if err != nil {
		t.Fatalf("SetProfile(first) error = %v", err)
	}
	// The second change starts where the first landed: from is the
	// effective yolo, not the creation standard.
	secondRequest := validSetProfileRequest()
	secondRequest.To = ProfileStandard
	secondRequest.Confirmed = false
	secondRequest.CreatedAt = "2026-08-19T04:06:00.000Z"
	second, err := transactor.SetProfile(secondRequest)
	if err != nil {
		t.Fatalf("SetProfile(second) error = %v", err)
	}
	if second.PreviousProfile != ProfileYOLO || second.NewProfile != ProfileStandard {
		t.Fatalf("SetProfile(second) result = %+v", second)
	}
	event, err := DecodeEvent(second.Event)
	if err != nil {
		t.Fatalf("DecodeEvent(second) error = %v", err)
	}
	from, _ := payloadString(event.Payload, "from")
	if from != ProfileYOLO {
		t.Fatalf("second change from = %q, want yolo", from)
	}
	if len(event.Predecessors) != 1 || event.Predecessors[0] != first.EventID {
		t.Fatalf("second change predecessors = %v, want exactly the first change", event.Predecessors)
	}
}

func TestSetProfileRetryReplaysCommittedChange(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	transactor := &Transactor{Repo: repository}
	first, err := transactor.SetProfile(validSetProfileRequest())
	if err != nil {
		t.Fatalf("SetProfile error = %v", err)
	}
	again, err := transactor.SetProfile(validSetProfileRequest())
	if err != nil {
		t.Fatalf("SetProfile(retry) error = %v", err)
	}
	if again.EventID != first.EventID || !bytes.Equal(again.Event, first.Event) {
		t.Fatal("retry did not replay the committed bytes")
	}
	// The replay reports the current effective profile as previous
	// (the change committed), with the same target and event.
	if again.PreviousProfile != ProfileYOLO || again.NewProfile != first.NewProfile {
		t.Fatalf("retry previous/new = %q/%q, want yolo/yolo", again.PreviousProfile, again.NewProfile)
	}
	indexed, err := repository.ListEvents(testSessionID)
	if err != nil {
		t.Fatalf("ListEvents error = %v", err)
	}
	if len(indexed) != 1 {
		t.Fatalf("chain holds %d events after retry, want 1", len(indexed))
	}
}

func TestSetProfileRefusals(t *testing.T) {
	setup := func(t *testing.T) *Transactor {
		repository := openTestRepository(t)
		reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
		first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
		leases, err := repository.ListLeases(testSessionID)
		if err != nil || len(leases) != 1 {
			t.Fatalf("ListLeases() = %v, %v; want epoch-1 lease", leases, err)
		}
		if _, err := repository.CompareAndSwapLease(testSessionID, sessrepo.LeaseExpectation{RecordID: leases[0].RecordID}, sessrepo.SuccessorLeaseInput{
			CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: testLeaseIDB, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testCreatedAt},
			Reason:           "graceful_takeover", CheckpointID: zeroDigest,
		}); err != nil {
			t.Fatalf("CompareAndSwapLease(test successor) error = %v", err)
		}
		appendTestEvent(t, repository, testSessionID, []string{first}, 2, testLeaseIDB, 1, "session.idle", map[string]any{"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true})
		return &Transactor{Repo: repository}
	}
	headRequest := func() SetProfileRequest {
		request := validSetProfileRequest()
		request.LeaseEpoch = 2
		request.LeaseID = testLeaseIDB
		return request
	}
	cases := []struct {
		name     string
		mutate   func(*SetProfileRequest)
		sentinel error
	}{
		{"target outside vocabulary", func(request *SetProfileRequest) { request.To = "turbo" }, ErrInvalidProfile},
		{"empty target", func(request *SetProfileRequest) { request.To = "" }, ErrInvalidProfile},
		{"stale epoch", func(request *SetProfileRequest) { request.LeaseEpoch = 1; request.LeaseID = testLeaseID }, ErrStaleLease},
		{"divergent lease", func(request *SetProfileRequest) { request.LeaseID = testLeaseIDC }, ErrDivergentLease},
		{"divergent lease below head", func(request *SetProfileRequest) { request.LeaseID = testLeaseID }, ErrDivergentLease},
		{"greater epoch", func(request *SetProfileRequest) { request.LeaseEpoch = 3 }, ErrDivergentLease},
		{"unconfirmed yolo", func(request *SetProfileRequest) { request.Confirmed = false }, ErrUnconfirmedYOLO},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transactor := setup(t)
			request := headRequest()
			tc.mutate(&request)
			if _, err := transactor.SetProfile(request); !errors.Is(err, tc.sentinel) {
				t.Fatalf("SetProfile(%s) error = %v, want %v", tc.name, err, tc.sentinel)
			}
		})
	}
}

func TestSetProfileRefusesNoOpChange(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	transactor := &Transactor{Repo: repository}
	// No change on record: the effective profile already equals the
	// target and no committed change carries the envelope.
	request := validSetProfileRequest()
	request.To = ProfileStandard
	request.Confirmed = false
	if _, err := transactor.SetProfile(request); !errors.Is(err, ErrProfileUnchanged) {
		t.Fatalf("SetProfile(no-op, no source) error = %v, want unchanged", err)
	}
	// A committed change with a different envelope (another author)
	// is a genuine no-op, not a replay.
	if _, err := transactor.SetProfile(validSetProfileRequest()); err != nil {
		t.Fatalf("SetProfile error = %v", err)
	}
	foreign := validSetProfileRequest()
	foreign.CreatedByHostID = testHostIDB
	if _, err := transactor.SetProfile(foreign); !errors.Is(err, ErrProfileUnchanged) {
		t.Fatalf("SetProfile(no-op, foreign envelope) error = %v, want unchanged", err)
	}
	mismatched := validSetProfileRequest()
	mismatched.CreatedAt = "2026-08-19T04:07:00.000Z"
	if _, err := transactor.SetProfile(mismatched); !errors.Is(err, ErrProfileUnchanged) {
		t.Fatalf("SetProfile(no-op, mismatched instant) error = %v, want unchanged", err)
	}
}

func TestSetProfileRefusesUnknownAndParked(t *testing.T) {
	repository := openTestRepository(t)
	transactor := &Transactor{Repo: repository}
	if _, err := transactor.SetProfile(validSetProfileRequest()); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("SetProfile(unknown) error = %v, want unknown session", err)
	}
	bare := &Transactor{}
	if _, err := bare.SetProfile(validSetProfileRequest()); !errors.Is(err, ErrDerivation) {
		t.Fatalf("SetProfile(no repo) error = %v, want corrupt derivation input", err)
	}
	var nilTransactor *Transactor
	if _, err := nilTransactor.SetProfile(validSetProfileRequest()); !errors.Is(err, ErrDerivation) {
		t.Fatalf("SetProfile(nil) error = %v, want corrupt derivation input", err)
	}
}

func TestSetProfileRefusesEmptyLeaseStore(t *testing.T) {
	repository := openTestRepository(t)
	createTestSessionWithoutLease(t, repository, testSessionID, "payments-api", ProfileStandard)
	if _, err := (&Transactor{Repo: repository}).SetProfile(validSetProfileRequest()); !errors.Is(err, sessrepo.ErrUnknownLease) {
		t.Fatalf("SetProfile(empty lease store) error = %v, want unknown lease refusal", err)
	}
}

func TestSetProfileRefusesNeverMintedHigherEpochOnEmptyChain(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	request := validSetProfileRequest()
	request.LeaseEpoch = 2
	request.LeaseID = testLeaseIDB
	if _, err := (&Transactor{Repo: repository}).SetProfile(request); !errors.Is(err, sessrepo.ErrUnknownLease) {
		t.Fatalf("SetProfile(unminted higher epoch) error = %v, want unknown lease refusal", err)
	}
}

func TestSetProfileRefusesBadMintParams(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	transactor := &Transactor{Repo: repository}
	request := validSetProfileRequest()
	request.CreatedByHostID = "nope"
	if _, err := transactor.SetProfile(request); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("SetProfile(bad author) error = %v, want invalid session event", err)
	}
	request = validSetProfileRequest()
	request.CreatedAt = "yesterday"
	if _, err := transactor.SetProfile(request); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("SetProfile(bad instant) error = %v, want invalid session event", err)
	}
}
