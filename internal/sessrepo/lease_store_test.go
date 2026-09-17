package sessrepo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/secconftest"
)

// Lease-test identities. testLeaseIDC is the third fencing token; testHostIDB
// is the normative Section 5.3 example holder.
const (
	testHostIDB  = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	testLeaseIDC = "cccccccc-dddd-4eee-8fff-111111111111"
	testLeaseAt  = "2026-08-19T04:09:00.000Z"
)

// specLeaseExample is the verbatim SPEC.md Section 5.3 normative example.
// Its record_id is the computed canonical digest, verified by
// TestNormativeLeaseRecordExampleAttests below.
const specLeaseExample = `{
  "schema": "urn:ax:schema:lease",
  "schema_version": "1.0.0",
  "record_id": "sha256:8ead987abed8c7c05175b447c9a0e2b3a521f12e5a989acb8abe132576852d63",
  "subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "lease_id": "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
  "epoch": 4,
  "holder_host_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab",
  "predecessor_lease_id": "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff",
  "reason": "graceful_takeover",
  "checkpoint_id": "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656",
  "issued_by_host_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab",
  "created_by_host_id": "0198f4c8-7d40-7e55-8e6f-1234567890ab",
  "created_at": "2026-08-19T04:09:00.000Z",
  "extensions": {}
}`

// createLeaseFixture returns the epoch-1 create input: fencing token A,
// holder and issuer host A, and the fixed diagnostic timestamp.
func createLeaseFixture() CreateLeaseInput {
	return CreateLeaseInput{LeaseID: testLeaseID, HolderHostID: testHostID, IssuedByHostID: testHostID, CreatedAt: testLeaseAt}
}

// successorLeaseFixture returns a successor input: fencing token B, holder
// and issuer host B, graceful_takeover over the zero checkpoint.
func successorLeaseFixture() SuccessorLeaseInput {
	return SuccessorLeaseInput{
		CreateLeaseInput: CreateLeaseInput{LeaseID: testLeaseIDB, HolderHostID: testHostIDB, IssuedByHostID: testHostIDB, CreatedAt: testLeaseAt},
		Reason:           "graceful_takeover",
		CheckpointID:     zeroDigest,
	}
}

// mustCreateLease creates the epoch-1 lease or fails the test.
func mustCreateLease(t *testing.T, repository *Repository, sessionID string, input CreateLeaseInput) LeaseRef {
	t.Helper()
	reference, err := repository.CreateLease(sessionID, input)
	if err != nil {
		t.Fatalf("CreateLease error = %v", err)
	}
	return reference
}

// mustCompareAndSwap advances the head or fails the test.
func mustCompareAndSwap(t *testing.T, repository *Repository, sessionID string, expected LeaseExpectation, input SuccessorLeaseInput) LeaseRef {
	t.Helper()
	reference, err := repository.CompareAndSwapLease(sessionID, expected, input)
	if err != nil {
		t.Fatalf("CompareAndSwapLease error = %v", err)
	}
	return reference
}

// mustAttestLease fails unless raw verifies through the canonical owner
// with the record_id self field.
func mustAttestLease(t *testing.T, raw []byte, what string) string {
	t.Helper()
	digest, field, err := canonicaljson.VerifyObjectIdentity(raw)
	if err != nil {
		t.Fatalf("%s VerifyObjectIdentity error = %v", what, err)
	}
	if field != canonicaljson.SelfRecordID {
		t.Fatalf("%s self field = %q, want record_id", what, string(field))
	}
	return digest.String()
}

func TestNormativeLeaseRecordExampleAttests(t *testing.T) {
	digest := mustAttestLease(t, []byte(specLeaseExample), "SPEC 5.3 example")
	if digest != "sha256:8ead987abed8c7c05175b447c9a0e2b3a521f12e5a989acb8abe132576852d63" {
		t.Fatalf("SPEC 5.3 example digest = %s", digest)
	}
}

func TestCreateLeasePersistsEpochOneCreate(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	reference := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	if reference.SessionID != testSessionID || reference.Epoch != 1 || reference.LeaseID != testLeaseID || reference.HolderHostID != testHostID {
		t.Fatalf("CreateLease ref = %+v", reference)
	}
	raw := mustGetLease(t, repository, testSessionID, reference.RecordID)
	if digest := mustAttestLease(t, raw, "created lease"); digest != reference.RecordID {
		t.Fatalf("created lease attests %s, ref carries %s", digest, reference.RecordID)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatalf("unmarshal created lease: %v", err)
	}
	if object["schema"] != "urn:ax:schema:lease" || object["epoch"] != 1.0 || object["reason"] != "create" {
		t.Fatalf("created lease shape = %v", object)
	}
	if object["predecessor_lease_id"] != nil || object["checkpoint_id"] != nil {
		t.Fatalf("epoch-1 create carries non-null predecessor/checkpoint: %v", object)
	}
	if object["subject_id"] != testSessionID || object["session_id"] != testSessionID {
		t.Fatalf("created lease scope = %v", object)
	}
	if object["issued_by_host_id"] != testHostID || object["created_by_host_id"] != testHostID {
		t.Fatalf("created lease issuer = %v", object)
	}
	leases, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases error = %v", err)
	}
	if len(leases) != 1 || leases[0].RecordID != reference.RecordID || leases[0].Reason != "create" || leases[0].HasPredecessor || leases[0].HasCheckpoint {
		t.Fatalf("ListLeases = %+v", leases)
	}
	winner, err := repository.WinningLease(testSessionID)
	if err != nil {
		t.Fatalf("WinningLease error = %v", err)
	}
	if winner.RecordID != reference.RecordID || winner.Epoch != 1 {
		t.Fatalf("WinningLease = %+v", winner)
	}
}

// mustGetLease reads one lease or fails the test.
func mustGetLease(t *testing.T, repository *Repository, sessionID, recordID string) []byte {
	t.Helper()
	raw, err := repository.GetLease(sessionID, recordID)
	if err != nil {
		t.Fatalf("GetLease(%s) error = %v", recordID, err)
	}
	return raw
}

func TestCreateLeaseRequiresExistingSession(t *testing.T) {
	repository := openTestRepository(t)
	_, err := repository.CreateLease(testSessionID, createLeaseFixture())
	mustErrorIs(t, err, ErrUnknownSession, "CreateLease(unknown session)")
}

func TestCreateLeaseRefusesInvalidIdentity(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*CreateLeaseInput)
	}{
		{"lease token", func(input *CreateLeaseInput) { input.LeaseID = "not-a-uuid" }},
		{"holder", func(input *CreateLeaseInput) { input.HolderHostID = "not-a-uuid" }},
		{"issuer", func(input *CreateLeaseInput) { input.IssuedByHostID = "0198f4c8-4a10-7b22-8b3c-1234567890aQ" }},
		{"created at", func(input *CreateLeaseInput) { input.CreatedAt = "not-a-timestamp" }},
		{"created at padded", func(input *CreateLeaseInput) { input.CreatedAt = " 2026-08-19T04:09:00.000Z" }},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			repository := openTestRepository(t)
			createTestSession(t, repository)
			input := createLeaseFixture()
			kase.mutate(&input)
			_, err := repository.CreateLease(testSessionID, input)
			mustErrorIs(t, err, ErrInvalidLease, "CreateLease("+kase.name+")")
			leases, listErr := repository.ListLeases(testSessionID)
			if listErr != nil {
				t.Fatalf("ListLeases error = %v", listErr)
			}
			if len(leases) != 0 {
				t.Fatalf("refused create persisted %d leases", len(leases))
			}
		})
	}
}

func TestCreateLeaseIsIdempotentOnByteIdenticalRetry(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	second := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	if first != second {
		t.Fatalf("retry ref = %+v, want %+v", second, first)
	}
	leases, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases error = %v", err)
	}
	if len(leases) != 1 {
		t.Fatalf("retry persisted %d leases, want 1", len(leases))
	}
}

func TestCreateLeaseRefusesSecondCreateWithDifferingBytes(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	second := createLeaseFixture()
	second.LeaseID = testLeaseIDB
	_, err := repository.CreateLease(testSessionID, second)
	mustErrorIs(t, err, ErrLeaseExists, "CreateLease(second create)")
	leases, listErr := repository.ListLeases(testSessionID)
	if listErr != nil {
		t.Fatalf("ListLeases error = %v", listErr)
	}
	if len(leases) != 1 || leases[0].RecordID != first.RecordID {
		t.Fatalf("second create disturbed the persisted lease: %+v", leases)
	}
	// A third create past a succession still refuses: the gate admits no
	// depth, so a mutant that admits only the len==1 retry still fails the
	// arm above while this arm proves the narrowing.
	mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
	later := createLeaseFixture()
	later.LeaseID = testLeaseIDC
	_, err = repository.CreateLease(testSessionID, later)
	mustErrorIs(t, err, ErrLeaseExists, "CreateLease(third create)")
}

// TestLeaseDedupComparesFullContent is the equality-census vector for the
// lease store's one content-equality site: two create candidates of the
// same length but disagreeing bytes must refuse as a second create, never
// deduplicate. A length-only comparison admits the retry and fails here.
func TestLeaseDedupComparesFullContent(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := createLeaseFixture()
	mustCreateLease(t, repository, testSessionID, first)
	second := createLeaseFixture()
	second.LeaseID = testLeaseIDB
	firstBytes, _, err := mintLeaseRecord(testSessionID, 1, first.LeaseID, first.HolderHostID, first.IssuedByHostID, "", false, "create", "", false, first.CreatedAt)
	if err != nil {
		t.Fatalf("mint first candidate: %v", err)
	}
	secondBytes, _, err := mintLeaseRecord(testSessionID, 1, second.LeaseID, second.HolderHostID, second.IssuedByHostID, "", false, "create", "", false, second.CreatedAt)
	if err != nil {
		t.Fatalf("mint second candidate: %v", err)
	}
	if len(firstBytes) != len(secondBytes) {
		t.Fatalf("candidate lengths = %d vs %d, want equal for the same-length vector", len(firstBytes), len(secondBytes))
	}
	_, err = repository.CreateLease(testSessionID, second)
	mustErrorIs(t, err, ErrLeaseExists, "CreateLease(same-length disagreeing bytes)")
}

func TestCompareAndSwapPersistsSuccessor(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	second := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
	if second.Epoch != 2 || second.LeaseID != testLeaseIDB || second.HolderHostID != testHostIDB {
		t.Fatalf("CAS ref = %+v", second)
	}
	raw := mustGetLease(t, repository, testSessionID, second.RecordID)
	if digest := mustAttestLease(t, raw, "successor lease"); digest != second.RecordID {
		t.Fatalf("successor attests %s, ref carries %s", digest, second.RecordID)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatalf("unmarshal successor lease: %v", err)
	}
	if object["epoch"] != 2.0 || object["predecessor_lease_id"] != testLeaseID || object["reason"] != "graceful_takeover" || object["checkpoint_id"] != zeroDigest {
		t.Fatalf("successor shape = %v", object)
	}
	leases, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases error = %v", err)
	}
	if len(leases) != 2 || leases[0].RecordID != first.RecordID || leases[1].RecordID != second.RecordID {
		t.Fatalf("ListLeases order = %+v", leases)
	}
	if !leases[1].HasPredecessor || leases[1].Predecessor != testLeaseID || !leases[1].HasCheckpoint || leases[1].Checkpoint != zeroDigest {
		t.Fatalf("successor summary = %+v", leases[1])
	}
	winner, err := repository.WinningLease(testSessionID)
	if err != nil {
		t.Fatalf("WinningLease error = %v", err)
	}
	if winner.RecordID != second.RecordID {
		t.Fatalf("WinningLease = %+v, want %s", winner, second.RecordID)
	}
}

func TestCompareAndSwapRequiresExistingLease(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	_, err := repository.CompareAndSwapLease(testSessionID, LeaseExpectation{}, successorLeaseFixture())
	mustErrorIs(t, err, ErrUnknownLease, "CAS(empty store)")
	_, err = repository.CompareAndSwapLease(testSessionID, LeaseExpectation{RecordID: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}, successorLeaseFixture())
	mustErrorIs(t, err, ErrUnknownLease, "CAS(empty store with named expectation)")
	_, err = repository.CompareAndSwapLease(testSessionIDB, LeaseExpectation{RecordID: zeroDigest}, successorLeaseFixture())
	mustErrorIs(t, err, ErrUnknownSession, "CAS(unknown session)")
}

func TestCompareAndSwapRefusesInvalidSuccessor(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*SuccessorLeaseInput)
	}{
		{"reason", func(input *SuccessorLeaseInput) { input.Reason = "bogus_reason" }},
		{"reason case", func(input *SuccessorLeaseInput) { input.Reason = "CREATE" }},
		{"checkpoint malformed", func(input *SuccessorLeaseInput) { input.CheckpointID = "not-a-digest" }},
		{"checkpoint absent", func(input *SuccessorLeaseInput) { input.CheckpointID = "" }},
		{"lease token", func(input *SuccessorLeaseInput) { input.LeaseID = "zzz" }},
		{"holder", func(input *SuccessorLeaseInput) { input.HolderHostID = "zzz" }},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			repository := openTestRepository(t)
			createTestSession(t, repository)
			first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
			input := successorLeaseFixture()
			kase.mutate(&input)
			_, err := repository.CompareAndSwapLease(testSessionID, LeaseExpectation{RecordID: first.RecordID}, input)
			mustErrorIs(t, err, ErrInvalidLease, "CAS("+kase.name+")")
			winner, winnerErr := repository.WinningLease(testSessionID)
			if winnerErr != nil {
				t.Fatalf("WinningLease error = %v", winnerErr)
			}
			if winner.RecordID != first.RecordID {
				t.Fatalf("refused CAS moved the winner to %+v", winner)
			}
		})
	}
}

func TestCompareAndSwapRefusesFencingTokenReuse(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	reused := successorLeaseFixture()
	reused.LeaseID = testLeaseID
	_, err := repository.CompareAndSwapLease(testSessionID, LeaseExpectation{RecordID: first.RecordID}, reused)
	mustErrorIs(t, err, ErrInvalidLease, "CAS(reused fencing token)")
}

func TestCompareAndSwapRefusesStaleExpectation(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	second := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
	advance := successorLeaseFixture()
	advance.LeaseID = testLeaseIDC
	head := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: second.RecordID}, advance)
	// Both superseded bases refuse: the epoch-1 and the epoch-2 arm. A
	// narrowing mutant that admits one basis still refuses the other.
	stale := []struct {
		name   string
		record string
		token  string
	}{
		{"epoch-1 basis", first.RecordID, "eeeeeeee-ffff-4aaa-8000-333333333333"},
		{"epoch-2 basis", second.RecordID, "ffffffff-0000-4111-8111-444444444444"},
	}
	for _, vector := range stale {
		input := successorLeaseFixture()
		input.LeaseID = vector.token
		_, err := repository.CompareAndSwapLease(testSessionID, LeaseExpectation{RecordID: vector.record}, input)
		mustErrorIs(t, err, ErrStaleLeaseEpoch, "CAS("+vector.name+")")
	}
	winner, winnerErr := repository.WinningLease(testSessionID)
	if winnerErr != nil {
		t.Fatalf("WinningLease error = %v", winnerErr)
	}
	if winner.RecordID != head.RecordID {
		t.Fatalf("stale CAS moved the winner to %+v", winner)
	}
}

func TestCompareAndSwapRefusesUnknownExpectation(t *testing.T) {
	cases := []struct {
		name     string
		recordID string
	}{
		{"malformed", "not-a-digest"},
		{"empty", ""},
		{"foreign", "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			repository := openTestRepository(t)
			createTestSession(t, repository)
			first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
			_, err := repository.CompareAndSwapLease(testSessionID, LeaseExpectation{RecordID: kase.recordID}, successorLeaseFixture())
			mustErrorIs(t, err, ErrLeaseConflict, "CAS("+kase.name+" expectation)")
			winner, winnerErr := repository.WinningLease(testSessionID)
			if winnerErr != nil {
				t.Fatalf("WinningLease error = %v", winnerErr)
			}
			if winner.RecordID != first.RecordID {
				t.Fatalf("conflicting CAS moved the winner to %+v", winner)
			}
		})
	}
}

// TestCompareAndSwapIsIdempotentOnByteIdenticalRetry replays the crashed-
// commit shape: the retry still names the pre-commit head, and the
// persisted successor answers without comparing the stale expectation.
func TestCompareAndSwapIsIdempotentOnByteIdenticalRetry(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	input := successorLeaseFixture()
	second := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, input)
	replayed := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, input)
	if replayed != second {
		t.Fatalf("retry ref = %+v, want %+v", replayed, second)
	}
	leases, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases error = %v", err)
	}
	if len(leases) != 2 {
		t.Fatalf("retry persisted %d leases, want 2", len(leases))
	}
}

func TestEpochMonotonicityAcrossSuccessors(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	head := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	tokens := []string{testLeaseIDB, testLeaseIDC, "dddddddd-eeee-4fff-8000-222222222222"}
	previous := testLeaseID
	for epoch, token := range tokens {
		input := successorLeaseFixture()
		input.LeaseID = token
		head = mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: head.RecordID}, input)
		if head.Epoch != uint64(epoch+2) {
			t.Fatalf("successor %d epoch = %d", epoch, head.Epoch)
		}
		summary, err := repository.WinningLease(testSessionID)
		if err != nil {
			t.Fatalf("WinningLease error = %v", err)
		}
		if !summary.HasPredecessor || summary.Predecessor != previous {
			t.Fatalf("epoch %d predecessor = %+v, want %s", head.Epoch, summary, previous)
		}
		previous = token
	}
	leases, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases error = %v", err)
	}
	for index, lease := range leases {
		if lease.Epoch != uint64(index+1) {
			t.Fatalf("stored lease %d epoch = %d", index, lease.Epoch)
		}
	}
}

func TestGetLeaseRefusals(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	reference := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	if _, err := repository.GetLease(testSessionID, "not-a-digest"); !errors.Is(err, ErrInvalidLease) {
		t.Fatalf("GetLease(malformed) error = %v, want invalid lease", err)
	}
	if _, err := repository.GetLease(testSessionID, "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"); !errors.Is(err, ErrUnknownLease) {
		t.Fatalf("GetLease(unknown) error = %v, want unknown lease", err)
	}
	if _, err := repository.GetLease(testSessionIDB, reference.RecordID); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("GetLease(unknown session) error = %v, want unknown session", err)
	}
}

func TestWinningLeaseRefusesEmptyStore(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	if _, err := repository.WinningLease(testSessionID); !errors.Is(err, ErrUnknownLease) {
		t.Fatalf("WinningLease(empty) error = %v, want unknown lease", err)
	}
	leases, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases(empty) error = %v", err)
	}
	if len(leases) != 0 {
		t.Fatalf("ListLeases(empty) = %+v, want none", leases)
	}
}

func TestVerifyFencingTokenAcceptsWinner(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	if err := repository.VerifyFencingToken(testSessionID, FencingToken{Epoch: 1, LeaseID: testLeaseID, HolderHostID: testHostID}); err != nil {
		t.Fatalf("VerifyFencingToken(epoch-1 winner) error = %v", err)
	}
	second := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
	if err := repository.VerifyFencingToken(testSessionID, FencingToken{Epoch: 2, LeaseID: second.LeaseID, HolderHostID: second.HolderHostID}); err != nil {
		t.Fatalf("VerifyFencingToken(successor winner) error = %v", err)
	}
}

func TestVerifyFencingTokenRefusals(t *testing.T) {
	setup := func(t *testing.T) *Repository {
		t.Helper()
		repository := openTestRepository(t)
		createTestSession(t, repository)
		first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
		mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
		return repository
	}
	cases := []struct {
		name    string
		token   FencingToken
		session string
		want    error
	}{
		{"epoch zero", FencingToken{Epoch: 0, LeaseID: testLeaseIDB, HolderHostID: testHostIDB}, testSessionID, ErrInvalidLease},
		{"lease malformed", FencingToken{Epoch: 2, LeaseID: "zzz", HolderHostID: testHostIDB}, testSessionID, ErrInvalidLease},
		{"holder malformed", FencingToken{Epoch: 2, LeaseID: testLeaseIDB, HolderHostID: "zzz"}, testSessionID, ErrInvalidLease},
		{"stale epoch", FencingToken{Epoch: 1, LeaseID: testLeaseID, HolderHostID: testHostID}, testSessionID, ErrStaleLeaseEpoch},
		{"future epoch", FencingToken{Epoch: 9, LeaseID: testLeaseIDC, HolderHostID: testHostIDB}, testSessionID, ErrUnknownLease},
		{"losing lease", FencingToken{Epoch: 2, LeaseID: testLeaseIDC, HolderHostID: testHostIDB}, testSessionID, ErrLeaseConflict},
		{"holder mismatch", FencingToken{Epoch: 2, LeaseID: testLeaseIDB, HolderHostID: testHostID}, testSessionID, ErrLeaseHolderMismatch},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			repository := setup(t)
			mustErrorIs(t, repository.VerifyFencingToken(kase.session, kase.token), kase.want, "VerifyFencingToken("+kase.name+")")
		})
	}
	t.Run("empty store", func(t *testing.T) {
		repository := openTestRepository(t)
		createTestSession(t, repository)
		mustErrorIs(t, repository.VerifyFencingToken(testSessionID, FencingToken{Epoch: 1, LeaseID: testLeaseID, HolderHostID: testHostID}), ErrUnknownLease, "VerifyFencingToken(empty store)")
	})
	t.Run("unknown session", func(t *testing.T) {
		repository := setup(t)
		mustErrorIs(t, repository.VerifyFencingToken(testSessionIDB, FencingToken{Epoch: 2, LeaseID: testLeaseIDB, HolderHostID: testHostIDB}), ErrUnknownSession, "VerifyFencingToken(unknown session)")
	})
}

// TestLeaseNeverExpiresWithAge pins the Section 5.3 non-expiry policy: a
// lease minted with an ancient diagnostic timestamp stays authoritative
// and its fencing token still verifies. Liveness is not authority.
func TestLeaseNeverExpiresWithAge(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	input := createLeaseFixture()
	input.CreatedAt = "2020-01-01T00:00:00.000Z"
	reference := mustCreateLease(t, repository, testSessionID, input)
	if err := repository.VerifyFencingToken(testSessionID, FencingToken{Epoch: 1, LeaseID: reference.LeaseID, HolderHostID: reference.HolderHostID}); err != nil {
		t.Fatalf("VerifyFencingToken(ancient lease) error = %v", err)
	}
	winner, err := repository.WinningLease(testSessionID)
	if err != nil {
		t.Fatalf("WinningLease error = %v", err)
	}
	if winner.RecordID != reference.RecordID {
		t.Fatalf("WinningLease = %+v, want %+v", winner, reference)
	}
}

func TestCheckFencingExpiry(t *testing.T) {
	validated := time.Date(2026, 8, 19, 4, 9, 0, 0, time.UTC)
	grant := FencingGrant{SessionID: testSessionID, RecordID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Token: FencingToken{Epoch: 1, LeaseID: testLeaseID, HolderHostID: testHostID}, ValidatedAt: validated}
	policy := FencingPolicy{RefreshInterval: time.Minute}
	if err := CheckFencingExpiry(grant, policy, validated); err != nil {
		t.Fatalf("CheckFencingExpiry(fresh) error = %v", err)
	}
	if err := CheckFencingExpiry(grant, policy, validated.Add(time.Minute)); err != nil {
		t.Fatalf("CheckFencingExpiry(boundary) error = %v", err)
	}
	mustErrorIs(t, CheckFencingExpiry(grant, policy, validated.Add(time.Minute+time.Nanosecond)), ErrFencingExpired, "CheckFencingExpiry(lapsed)")
	mustErrorIs(t, CheckFencingExpiry(grant, FencingPolicy{}, validated), ErrInvalidLease, "CheckFencingExpiry(zero interval)")
	mustErrorIs(t, CheckFencingExpiry(grant, FencingPolicy{RefreshInterval: -time.Second}, validated), ErrInvalidLease, "CheckFencingExpiry(negative interval)")
}

func TestCompareLeaseTupleOrdersByGreatestTuple(t *testing.T) {
	low := LeaseSummary{Epoch: 1, LeaseID: testLeaseIDB}
	high := LeaseSummary{Epoch: 2, LeaseID: testLeaseID}
	if CompareLeaseTuple(low, high) != -1 || CompareLeaseTuple(high, low) != 1 {
		t.Fatal("epoch does not dominate the tuple order")
	}
	tieLow := LeaseSummary{Epoch: 2, LeaseID: testLeaseID}
	tieHigh := LeaseSummary{Epoch: 2, LeaseID: testLeaseIDB}
	if CompareLeaseTuple(tieLow, tieHigh) != -1 || CompareLeaseTuple(tieHigh, tieLow) != 1 {
		t.Fatal("same-epoch tie does not resolve bytewise")
	}
	if CompareLeaseTuple(tieHigh, tieHigh) != 0 {
		t.Fatal("identical tuples do not compare equal")
	}
}

func TestLeaseStageTempsAreIgnored(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	reference := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	leasesDir := filepath.Join(repository.root, testSessionID, "leases")
	if err := os.WriteFile(filepath.Join(leasesDir, ".ax-lease-stage-orphan"), []byte("torn stage"), 0o600); err != nil {
		t.Fatalf("plant stage temp: %v", err)
	}
	leases, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases error = %v", err)
	}
	if len(leases) != 1 || leases[0].RecordID != reference.RecordID {
		t.Fatalf("ListLeases with orphaned temp = %+v", leases)
	}
	winner, err := repository.WinningLease(testSessionID)
	if err != nil {
		t.Fatalf("WinningLease error = %v", err)
	}
	if winner.RecordID != reference.RecordID {
		t.Fatalf("WinningLease with orphaned temp = %+v", winner)
	}
}

// TestStoredLeaseCorruptionRefuses drives every torn-store vector through
// the load funnel and the GetLease funnel: unparseable names, undecodable
// bytes, cross-session blobs, truncated blobs, and digest-name
// disagreement each refuse chain corruption.
func TestStoredLeaseCorruptionRefuses(t *testing.T) {
	plant := func(t *testing.T) (*Repository, string) {
		t.Helper()
		repository := openTestRepository(t)
		createTestSession(t, repository)
		mustCreateLease(t, repository, testSessionID, createLeaseFixture())
		return repository, filepath.Join(repository.root, testSessionID, "leases")
	}
	t.Run("unparseable name", func(t *testing.T) {
		repository, dir := plant(t)
		if err := os.WriteFile(filepath.Join(dir, "garbage.json"), []byte("{broken"), 0o600); err != nil {
			t.Fatalf("plant garbage name: %v", err)
		}
		_, err := repository.ListLeases(testSessionID)
		mustErrorIs(t, err, ErrChainCorrupt, "ListLeases(unparseable name)")
	})
	t.Run("undecodable bytes", func(t *testing.T) {
		repository, dir := plant(t)
		name := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.json"
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{broken lease"), 0o600); err != nil {
			t.Fatalf("plant garbage bytes: %v", err)
		}
		_, err := repository.ListLeases(testSessionID)
		mustErrorIs(t, err, ErrChainCorrupt, "ListLeases(undecodable bytes)")
		_, err = repository.GetLease(testSessionID, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		mustErrorIs(t, err, ErrChainCorrupt, "GetLease(undecodable bytes)")
	})
	t.Run("truncated blob", func(t *testing.T) {
		repository, dir := plant(t)
		winner, err := repository.WinningLease(testSessionID)
		if err != nil {
			t.Fatalf("WinningLease error = %v", err)
		}
		raw := mustGetLease(t, repository, testSessionID, winner.RecordID)
		if err := os.WriteFile(filepath.Join(dir, blobFileName(winner.RecordID)), raw[:len(raw)/2], 0o600); err != nil {
			t.Fatalf("truncate lease blob: %v", err)
		}
		_, err = repository.ListLeases(testSessionID)
		mustErrorIs(t, err, ErrChainCorrupt, "ListLeases(truncated blob)")
	})
	t.Run("cross session blob", func(t *testing.T) {
		repository, dir := plant(t)
		other := buildRecord(t, func(object map[string]any) {
			object["session_id"] = testSessionIDB
			object["subject_id"] = testSessionIDB
			object["name"] = "other-session"
		})
		if _, err := repository.CreateSession(other); err != nil {
			t.Fatalf("CreateSession(other) error = %v", err)
		}
		foreign := mustCreateLease(t, repository, testSessionIDB, createLeaseFixture())
		foreignRaw := mustGetLease(t, repository, testSessionIDB, foreign.RecordID)
		if err := os.WriteFile(filepath.Join(dir, blobFileName(foreign.RecordID)), foreignRaw, 0o600); err != nil {
			t.Fatalf("plant cross-session blob: %v", err)
		}
		_, err := repository.ListLeases(testSessionID)
		mustErrorIs(t, err, ErrChainCorrupt, "ListLeases(cross-session blob)")
	})
	t.Run("digest name disagreement", func(t *testing.T) {
		repository, dir := plant(t)
		winner, err := repository.WinningLease(testSessionID)
		if err != nil {
			t.Fatalf("WinningLease error = %v", err)
		}
		raw := mustGetLease(t, repository, testSessionID, winner.RecordID)
		alias := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.json"
		if err := os.WriteFile(filepath.Join(dir, alias), raw, 0o600); err != nil {
			t.Fatalf("plant misnamed blob: %v", err)
		}
		_, err = repository.ListLeases(testSessionID)
		mustErrorIs(t, err, ErrChainCorrupt, "ListLeases(name disagreement)")
		_, err = repository.GetLease(testSessionID, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		mustErrorIs(t, err, ErrChainCorrupt, "GetLease(name disagreement)")
	})
}

// TestInstallDisagreeingBytesRefuses plants garbage at the computed install
// path between verification and install through the BeforeWrite hook — the
// single-process shape of a cross-process race — and requires the torn
// refusal. The same-length arm is the install-site half of the
// content-equality story: a length-only comparison reuses the garbage and
// fails here.
func TestInstallDisagreeingBytesRefuses(t *testing.T) {
	cases := []struct {
		name       string
		sameLength bool
	}{
		{"same length", true},
		{"different length", false},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			repository := openTestRepository(t)
			createTestSession(t, repository)
			input := createLeaseFixture()
			candidate, digest, err := mintLeaseRecord(testSessionID, 1, input.LeaseID, input.HolderHostID, input.IssuedByHostID, "", false, "create", "", false, input.CreatedAt)
			if err != nil {
				t.Fatalf("mint candidate: %v", err)
			}
			garbage := []byte(strings.Repeat("X", len(candidate)))
			if !kase.sameLength {
				garbage = []byte("short")
			}
			target := filepath.Join(repository.root, testSessionID, "leases", blobFileName(digest))
			repository.BeforeWrite = func() error {
				if err := os.MkdirAll(filepath.Join(repository.root, testSessionID, "leases"), 0o700); err != nil {
					return err
				}
				return os.WriteFile(target, garbage, 0o600)
			}
			_, err = repository.CreateLease(testSessionID, input)
			mustErrorIs(t, err, ErrChainCorrupt, "CreateLease("+kase.name+" disagreeing bytes)")
			repository.BeforeWrite = nil
			_, err = repository.WinningLease(testSessionID)
			mustErrorIs(t, err, ErrChainCorrupt, "WinningLease(after plant)")
		})
	}
}

func TestCrashBeforeLeaseWriteIsSafeRetry(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		repository := openTestRepository(t)
		createTestSession(t, repository)
		injector := &secconftest.Injector{}
		wireCrashPoints(repository, injector)
		injector.Arm(secconftest.PointPrepareEnter)
		_, err := repository.CreateLease(testSessionID, createLeaseFixture())
		mustCrashFault(t, err, secconftest.PointPrepareEnter, secconftest.OutcomeSafeRetry, "CreateLease(prepare fault)")
		leases, listErr := repository.ListLeases(testSessionID)
		if listErr != nil {
			t.Fatalf("ListLeases error = %v", listErr)
		}
		if len(leases) != 0 {
			t.Fatalf("prepare fault persisted %d leases", len(leases))
		}
		reference := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
		if reference.Epoch != 1 {
			t.Fatalf("retry ref = %+v", reference)
		}
	})
	t.Run("cas", func(t *testing.T) {
		repository := openTestRepository(t)
		createTestSession(t, repository)
		first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
		injector := &secconftest.Injector{}
		wireCrashPoints(repository, injector)
		injector.Arm(secconftest.PointPrepareEnter)
		_, err := repository.CompareAndSwapLease(testSessionID, LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
		mustCrashFault(t, err, secconftest.PointPrepareEnter, secconftest.OutcomeSafeRetry, "CAS(prepare fault)")
		winner, winnerErr := repository.WinningLease(testSessionID)
		if winnerErr != nil {
			t.Fatalf("WinningLease error = %v", winnerErr)
		}
		if winner.RecordID != first.RecordID {
			t.Fatalf("prepare fault moved the winner to %+v", winner)
		}
		second := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
		if second.Epoch != 2 {
			t.Fatalf("retry ref = %+v", second)
		}
	})
}

func TestCrashAfterLeaseCommitCountsAsCommitted(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		repository := openTestRepository(t)
		createTestSession(t, repository)
		injector := &secconftest.Injector{}
		wireCrashPoints(repository, injector)
		injector.Arm(secconftest.PointCommitApply)
		input := createLeaseFixture()
		_, err := repository.CreateLease(testSessionID, input)
		mustCrashFault(t, err, secconftest.PointCommitApply, secconftest.OutcomeRecoverableParked, "CreateLease(commit fault)")
		stored, listErr := repository.ListLeases(testSessionID)
		if listErr != nil {
			t.Fatalf("ListLeases error = %v", listErr)
		}
		if len(stored) != 1 {
			t.Fatalf("crashed commit persisted %d leases, want 1", len(stored))
		}
		replayed := mustCreateLease(t, repository, testSessionID, input)
		if replayed.RecordID != stored[0].RecordID {
			t.Fatalf("retry ref = %+v, want %s", replayed, stored[0].RecordID)
		}
	})
	t.Run("cas", func(t *testing.T) {
		repository := openTestRepository(t)
		createTestSession(t, repository)
		first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
		injector := &secconftest.Injector{}
		wireCrashPoints(repository, injector)
		injector.Arm(secconftest.PointCommitApply)
		input := successorLeaseFixture()
		_, err := repository.CompareAndSwapLease(testSessionID, LeaseExpectation{RecordID: first.RecordID}, input)
		mustCrashFault(t, err, secconftest.PointCommitApply, secconftest.OutcomeRecoverableParked, "CAS(commit fault)")
		winner, winnerErr := repository.WinningLease(testSessionID)
		if winnerErr != nil {
			t.Fatalf("WinningLease error = %v", winnerErr)
		}
		if winner.Epoch != 2 {
			t.Fatalf("crashed CAS winner = %+v, want epoch 2", winner)
		}
		replayed := mustCompareAndSwap(t, repository, testSessionID, LeaseExpectation{RecordID: first.RecordID}, input)
		if replayed.RecordID != winner.RecordID {
			t.Fatalf("retry ref = %+v, want %s", replayed, winner.RecordID)
		}
	})
}
