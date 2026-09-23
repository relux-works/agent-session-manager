package sessprofile

import (
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

func TestProjectorRejectsNeverMintedHigherEpochProfileSource(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	change := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 2, testLeaseIDB, 1, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))

	pair, err := (&Projector{Repo: repository}).Project(testSessionID)
	if err != nil {
		t.Fatalf("Projector.Project() error = %v", err)
	}
	if pair.Profile != ProfileStandard || pair.HasSource || pair.Source == change {
		t.Fatalf("Projector.Project() for unminted higher-epoch source = %+v, want standard with no source", pair)
	}
}

func TestProjectorEmptyLeaseStoreKeepsRecordAuthority(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSessionWithoutLease(t, repository, testSessionID, "payments-api", ProfileStandard)
	change := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))

	pair, err := (&Projector{Repo: repository}).Project(testSessionID)
	if err != nil {
		t.Fatalf("Projector.Project(empty lease store) error = %v", err)
	}
	if pair.Profile != ProfileStandard || pair.HasSource || pair.Source == change {
		t.Fatalf("Projector.Project(empty lease store) = %+v, want standard with no source", pair)
	}
}

func TestDerivationForHeadsRejectsUnmintedHigherEpochSource(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	change := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 2, testLeaseIDB, 1, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	record, events := decodeTestChain(t, repository, testSessionID)
	authority, err := LoadSourceAuthority(repository, nil, testSessionID)
	if err != nil {
		t.Fatalf("LoadSourceAuthority() error = %v", err)
	}
	pair, err := (Derivation{Record: record, Events: events, Authority: authority}).DeriveForHeads([]string{change})
	if err != nil {
		t.Fatalf("Derivation.DeriveForHeads() error = %v", err)
	}
	if pair.Profile != ProfileStandard || pair.HasSource || pair.Source == change {
		t.Fatalf("Derivation.DeriveForHeads(unminted epoch) = %+v, want standard with no source", pair)
	}
}

func TestProjectorForHeadsRejectsUnmintedHigherEpochProfileSource(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	change := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 2, testLeaseIDB, 1, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))

	pair, err := (&Projector{Repo: repository}).ProjectForHeads(testSessionID, []string{change})
	if err != nil {
		t.Fatalf("Projector.ProjectForHeads() error = %v", err)
	}
	if pair.Profile != ProfileStandard || pair.HasSource || pair.Source == change {
		t.Fatalf("Projector.ProjectForHeads(unminted epoch) = %+v, want standard with no source", pair)
	}
}

func TestProjectorKeepsPriorLeaseSourceFromWinningHandoffClosure(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	created := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	change := appendTestEvent(t, repository, testSessionID, []string{created}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	checkpoints, checkpointID := captureProfileHandoff(t, repository, []string{change})
	leases, err := repository.ListLeases(testSessionID)
	if err != nil || len(leases) != 1 {
		t.Fatalf("ListLeases() = %v, %v; want one epoch-1 lease", leases, err)
	}
	if _, err := repository.CompareAndSwapLease(testSessionID, sessrepo.LeaseExpectation{RecordID: leases[0].RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: testLeaseIDB, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testCreatedAt},
		Reason:           "graceful_takeover", CheckpointID: checkpointID,
	}); err != nil {
		t.Fatalf("CompareAndSwapLease(successor) error = %v", err)
	}

	pair, err := (&Projector{Repo: repository, Ckpt: checkpoints}).Project(testSessionID)
	if err != nil {
		t.Fatalf("Projector.Project(after takeover) error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: change, HasSource: true}, "Projector(Projector winner handoff)")

	closurePair, err := (&Projector{Repo: repository, Ckpt: checkpoints}).ProjectForHeads(testSessionID, []string{change})
	if err != nil {
		t.Fatalf("Projector.ProjectForHeads(winning handoff) error = %v", err)
	}
	mustPairEqual(t, closurePair, Pair{Profile: ProfileYOLO, Source: change, HasSource: true}, "Projector.ProjectForHeads(winning handoff)")
}

func TestProjectorMissingWinningHandoffStoreIsNotTreatedAsEmptyClosure(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	created := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	change := appendTestEvent(t, repository, testSessionID, []string{created}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	leases, err := repository.ListLeases(testSessionID)
	if err != nil || len(leases) != 1 {
		t.Fatalf("ListLeases() = %v, %v; want one epoch-1 lease", leases, err)
	}
	if _, err := repository.CompareAndSwapLease(testSessionID, sessrepo.LeaseExpectation{RecordID: leases[0].RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: testLeaseIDB, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testCreatedAt},
		Reason:           "graceful_takeover", CheckpointID: zeroDigest,
	}); err != nil {
		t.Fatalf("CompareAndSwapLease(successor) error = %v", err)
	}
	if _, err := (&Projector{Repo: repository}).Project(testSessionID); !errors.Is(err, ErrDerivation) {
		t.Fatalf("Projector.Project(missing handoff store) error = %v, want corrupt derivation input", err)
	}
	if _, err := (&Projector{Repo: repository}).ProjectForHeads(testSessionID, []string{change}); !errors.Is(err, ErrDerivation) {
		t.Fatalf("Projector.ProjectForHeads(missing handoff store) error = %v, want corrupt derivation input", err)
	}
}
