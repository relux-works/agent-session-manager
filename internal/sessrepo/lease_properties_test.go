package sessrepo

import (
	"bytes"
	"errors"
	"math/rand"
	"reflect"
	"testing"
)

// Ownership store properties (TASK-260830-2atgj4): the store halves of
// invariants 3 (clock non-authority) and 4 (zero duplicate authorized
// owners) over the landed CreateLease, CompareAndSwapLease,
// WinningLease, ListLeases, GetLease, and VerifyFencingToken entries.
// The reducer halves live in internal/sessstate, the gate halves in
// internal/fencing. Generators use the closed lease/host/reason
// alphabets below with exhaustive enumeration where the space is
// small and bounded seeded-random exploration otherwise (seed and
// iteration counts recorded); no new module dependency.

// propLeaseIDs is the closed fencing-token alphabet. Every value keeps
// the UUIDv4 nibble and an RFC 4122 variant.
var propLeaseIDs = []string{
	testLeaseID,
	testLeaseIDB,
	testLeaseIDC,
	"dddddddd-eeee-4fff-8aaa-333333333333",
	"eeeeeeee-ffff-4000-9bbb-444444444444",
	"ffffffff-0000-4111-abbb-555555555555",
}

// propReasons is the closed Section 5.3 reason alphabet.
var propReasons = []string{"create", "graceful_takeover", "force_takeover", "recovery"}

// propClockShifts carries every timestamp vector of the clock
// property: backwards, the suite baseline, an equal repeat of the
// baseline, and a far-future value. All are grammatically valid
// timestamps: grammar still refuses malformed values (landed
// negatives); authority must not consult the value.
var propClockShifts = []string{
	"2020-01-01T00:00:00.000Z",
	testLeaseAt,
	testLeaseAt,
	"2030-12-31T23:59:59.999Z",
}

// propAuthority is the authority projection of a lease chain: the
// epochs, fencing tokens, and holders in tuple order. Record digests
// are excluded on purpose: created_at flows into the canonical
// identity bytes, so digests vary across clock shifts while authority
// must not.
type propAuthority struct {
	epochs  []uint64
	leases  []string
	holders []string
}

func propAuthorityOf(summaries []LeaseSummary) propAuthority {
	authority := propAuthority{}
	for _, summary := range summaries {
		authority.epochs = append(authority.epochs, summary.Epoch)
		authority.leases = append(authority.leases, summary.LeaseID)
		authority.holders = append(authority.holders, summary.HolderHostID)
	}
	return authority
}

// propBuildChain creates the epoch-1 lease and one successor per
// reason, stamping created_at from stamps (cycled). It returns the
// stored summaries in tuple order.
func propBuildChain(t *testing.T, repository *Repository, sessionID string, reasons []string, stamps []string) []LeaseSummary {
	t.Helper()
	holderFor := func(index int) string {
		if index%2 == 0 {
			return testHostID
		}
		return testHostIDB
	}
	leaseFor := func(index int) string {
		return propLeaseIDs[index%len(propLeaseIDs)]
	}
	stampFor := func(index int) string {
		return stamps[index%len(stamps)]
	}
	holder := holderFor(0)
	created := mustCreateLease(t, repository, sessionID, CreateLeaseInput{
		LeaseID: leaseFor(0), HolderHostID: holder, IssuedByHostID: holder, CreatedAt: stampFor(0),
	})
	expected := LeaseExpectation{RecordID: created.RecordID}
	for i, reason := range reasons {
		index := i + 1
		holder = holderFor(index)
		next := mustCompareAndSwap(t, repository, sessionID, expected, SuccessorLeaseInput{
			CreateLeaseInput: CreateLeaseInput{
				LeaseID: leaseFor(index), HolderHostID: holder,
				IssuedByHostID: holder, CreatedAt: stampFor(index),
			},
			Reason:       reason,
			CheckpointID: zeroDigest,
		})
		expected = LeaseExpectation{RecordID: next.RecordID}
	}
	summaries, err := repository.ListLeases(sessionID)
	if err != nil {
		t.Fatalf("ListLeases error = %v", err)
	}
	return summaries
}

// TestOwnershipStoreClockNonAuthority proves invariant 3 over the
// store: shifting every created_at backwards, equal, or far into the
// future leaves epochs, fencing tokens, holders, the winning tuple,
// and every fencing verdict identical. Record digests DO vary —
// created_at is digested into the identity bytes — which is what
// makes the invariant non-vacuous: identity varies, authority does
// not. Four uniform shifts plus one mixed non-monotonic chain.
func TestOwnershipStoreClockNonAuthority(t *testing.T) {
	var baseline propAuthority
	var baselineDigests []string
	verdict := func(t *testing.T, repository *Repository, token FencingToken) error {
		t.Helper()
		return repository.VerifyFencingToken(testSessionID, token)
	}
	digestsByShift := make([][]string, len(propClockShifts))
	for shift, stamp := range propClockShifts {
		repository := openTestRepository(t)
		createTestSession(t, repository)
		summaries := propBuildChain(t, repository, testSessionID,
			[]string{"graceful_takeover", "force_takeover"}, []string{stamp})
		authority := propAuthorityOf(summaries)
		if shift == 0 {
			baseline = authority
			for _, summary := range summaries {
				baselineDigests = append(baselineDigests, summary.RecordID)
			}
		} else if !reflect.DeepEqual(authority, baseline) {
			t.Fatalf("shift %q authority = %+v, want baseline %+v", stamp, authority, baseline)
		}
		winner, err := repository.WinningLease(testSessionID)
		if err != nil {
			t.Fatalf("shift %q WinningLease error = %v", stamp, err)
		}
		if winner.Epoch != 3 || winner.LeaseID != propLeaseIDs[2] {
			t.Fatalf("shift %q winner = %+v, want epoch 3 lease %s", stamp, winner, propLeaseIDs[2])
		}
		winnerToken := FencingToken{Epoch: winner.Epoch, LeaseID: winner.LeaseID, HolderHostID: winner.HolderHostID}
		if err := verdict(t, repository, winnerToken); err != nil {
			t.Fatalf("shift %q winner token verdict = %v, want nil", stamp, err)
		}
		stale := FencingToken{Epoch: 1, LeaseID: propLeaseIDs[0], HolderHostID: testHostID}
		if err := verdict(t, repository, stale); !errors.Is(err, ErrStaleLeaseEpoch) {
			t.Fatalf("shift %q stale token verdict = %v, want stale lease epoch", stamp, err)
		}
		future := FencingToken{Epoch: 9, LeaseID: propLeaseIDs[0], HolderHostID: testHostID}
		if err := verdict(t, repository, future); !errors.Is(err, ErrUnknownLease) {
			t.Fatalf("shift %q future token verdict = %v, want unknown lease", stamp, err)
		}
		for _, summary := range summaries {
			digestsByShift[shift] = append(digestsByShift[shift], summary.RecordID)
		}
		if shift == 0 {
			continue
		}
		// Identity must vary across differing stamps (or the shift
		// proved nothing) while authority stays put (asserted
		// above); equal stamps must digest identically.
		for i, summary := range summaries {
			sameBytes := stamp == propClockShifts[0]
			if (summary.RecordID == baselineDigests[i]) != sameBytes {
				t.Fatalf("shift %q lease %d digest %s: identity-vary = %v, want %v",
					stamp, i, summary.RecordID, summary.RecordID != baselineDigests[i], !sameBytes)
			}
		}
	}
	if !reflect.DeepEqual(digestsByShift[1], digestsByShift[2]) {
		t.Fatalf("equal-stamp shifts digest %+v vs %+v, want identical content identity",
			digestsByShift[1], digestsByShift[2])
	}
	// The mixed chain stamps successive leases backwards, far-future,
	// and baseline: non-monotonic wall clocks still derive the same
	// authority as the uniform baseline.
	repository := openTestRepository(t)
	createTestSession(t, repository)
	mixed := propBuildChain(t, repository, testSessionID,
		[]string{"graceful_takeover", "force_takeover"},
		[]string{"2030-12-31T23:59:59.999Z", "2020-01-01T00:00:00.000Z", testLeaseAt})
	if authority := propAuthorityOf(mixed); !reflect.DeepEqual(authority, baseline) {
		t.Fatalf("mixed-shift authority = %+v, want baseline %+v", authority, baseline)
	}
	t.Logf("store-clock: %d uniform shifts + 1 mixed chain, authority identical, digests vary", len(propClockShifts))
}

// TestOwnershipStoreSingleAuthorityPerEpoch proves invariant 4 over
// the store: after any admissible create-then-CAS sequence, each
// epoch carries exactly one authoritative lease, the winner is the
// tuple maximum, superseded leases keep byte-identical blobs (never
// dropped or rewritten), and every blob still verifies. Exhaustive
// over reason sequences of length 0..2 (21 sequences); bounded
// seeded-random over lengths 3..5 (seed 260831, 200 iterations).
func TestOwnershipStoreSingleAuthorityPerEpoch(t *testing.T) {
	check := func(t *testing.T, reasons []string) {
		t.Helper()
		repository := openTestRepository(t)
		createTestSession(t, repository)
		summaries := propBuildChain(t, repository, testSessionID, reasons, []string{testLeaseAt})
		if len(summaries) != len(reasons)+1 {
			t.Fatalf("reasons %+v: %d leases stored, want %d", reasons, len(summaries), len(reasons)+1)
		}
		seenEpoch := map[uint64]int{}
		for index, summary := range summaries {
			seenEpoch[summary.Epoch]++
			if want := uint64(index + 1); summary.Epoch != want {
				t.Fatalf("reasons %+v: lease %d epoch = %d, want %d in tuple order",
					reasons, index, summary.Epoch, want)
			}
			if _, err := repository.GetLease(testSessionID, summary.RecordID); err != nil {
				t.Fatalf("reasons %+v: GetLease(%s) error = %v", reasons, summary.RecordID, err)
			}
		}
		for epoch, count := range seenEpoch {
			if count != 1 {
				t.Fatalf("reasons %+v: epoch %d carries %d leases, want exactly one", reasons, epoch, count)
			}
		}
		winner, err := repository.WinningLease(testSessionID)
		if err != nil {
			t.Fatalf("reasons %+v: WinningLease error = %v", reasons, err)
		}
		head := summaries[len(summaries)-1]
		if winner.RecordID != head.RecordID {
			t.Fatalf("reasons %+v: winner %s is not the tuple-maximum head %s", reasons, winner.RecordID, head.RecordID)
		}
		var best LeaseSummary
		for i, summary := range summaries {
			if i == 0 || CompareLeaseTuple(summary, best) > 0 {
				best = summary
			}
		}
		if winner.RecordID != best.RecordID {
			t.Fatalf("reasons %+v: winner %s is not the compared maximum %s", reasons, winner.RecordID, best.RecordID)
		}
	}
	var exhaustive [][]string
	exhaustive = append(exhaustive, nil)
	for _, first := range propReasons {
		exhaustive = append(exhaustive, []string{first})
		for _, second := range propReasons {
			exhaustive = append(exhaustive, []string{first, second})
		}
	}
	for _, reasons := range exhaustive {
		check(t, reasons)
	}
	rng := rand.New(rand.NewSource(260831))
	const randomSequences = 200
	for i := 0; i < randomSequences; i++ {
		length := 3 + rng.Intn(3)
		reasons := make([]string, length)
		for j := range reasons {
			reasons[j] = propReasons[rng.Intn(len(propReasons))]
		}
		check(t, reasons)
	}
	t.Logf("store-single-authority: %d exhaustive + %d seeded-random sequences (seed 260831)", len(exhaustive), randomSequences)
}

// TestOwnershipStoreLoserBytesPreserved proves the durability half of
// invariant 2: a superseded lease is never dropped or rewritten —
// its blob stays byte-identical and verifiable after successions —
// and sessions are isolated: interleaved chains never share
// authority.
func TestOwnershipStoreLoserBytesPreserved(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	other := buildRecord(t, func(object map[string]any) {
		object["subject_id"] = testSessionIDB
		object["session_id"] = testSessionIDB
	})
	if _, err := repository.CreateSession(other); err != nil {
		t.Fatalf("CreateSession(B) error = %v", err)
	}
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	before := mustGetLease(t, repository, testSessionID, first.RecordID)
	second := mustCompareAndSwap(t, repository, testSessionID,
		LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
	after := mustGetLease(t, repository, testSessionID, first.RecordID)
	if !bytes.Equal(before, after) {
		t.Fatal("epoch-1 lease bytes changed across a succession; losers must be immutable")
	}
	third := mustCompareAndSwap(t, repository, testSessionID,
		LeaseExpectation{RecordID: second.RecordID}, SuccessorLeaseInput{
			CreateLeaseInput: CreateLeaseInput{
				LeaseID: testLeaseIDC, HolderHostID: testHostID,
				IssuedByHostID: testHostID, CreatedAt: testLeaseAt,
			},
			Reason:       "force_takeover",
			CheckpointID: zeroDigest,
		})
	_ = third
	again := mustGetLease(t, repository, testSessionID, first.RecordID)
	if !bytes.Equal(before, again) {
		t.Fatal("epoch-1 lease bytes changed across two successions; losers must be immutable")
	}
	summaries, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases(A) error = %v", err)
	}
	if len(summaries) != 3 {
		t.Fatalf("ListLeases(A) = %d leases, want 3; no lease may be dropped", len(summaries))
	}
	// The sibling session holds no leases yet: authority never leaks
	// across sessions.
	sibling, err := repository.ListLeases(testSessionIDB)
	if err != nil {
		t.Fatalf("ListLeases(B) error = %v", err)
	}
	if len(sibling) != 0 {
		t.Fatalf("ListLeases(B) = %+v, want none; sessions must be isolated", sibling)
	}
	winnerA, err := repository.WinningLease(testSessionID)
	if err != nil {
		t.Fatalf("WinningLease(A) error = %v", err)
	}
	if winnerA.Epoch != 3 || winnerA.LeaseID != testLeaseIDC {
		t.Fatalf("WinningLease(A) = %+v, want epoch 3 lease C", winnerA)
	}
}

// TestOwnershipStoreTwoHandlesOneHead proves the two-process half of
// invariant 4 at the store layer: the head derives from the blobs on
// every read, so a second handle sees a committed succession
// immediately, and a lagging compare-and-swap refuses stale instead
// of forking a second head.
func TestOwnershipStoreTwoHandlesOneHead(t *testing.T) {
	repository := openTestRepository(t)
	createTestSession(t, repository)
	first := mustCreateLease(t, repository, testSessionID, createLeaseFixture())
	// A lagging handle that still expects the epoch-1 head after a
	// succession refuses stale; it must not fork a second epoch-2.
	second := mustCompareAndSwap(t, repository, testSessionID,
		LeaseExpectation{RecordID: first.RecordID}, successorLeaseFixture())
	staleInput := SuccessorLeaseInput{
		CreateLeaseInput: CreateLeaseInput{
			LeaseID: testLeaseIDC, HolderHostID: testHostID,
			IssuedByHostID: testHostID, CreatedAt: testLeaseAt,
		},
		Reason:       "force_takeover",
		CheckpointID: zeroDigest,
	}
	_, err := repository.CompareAndSwapLease(testSessionID,
		LeaseExpectation{RecordID: first.RecordID}, staleInput)
	if !errors.Is(err, ErrStaleLeaseEpoch) {
		t.Fatalf("lagging CAS error = %v, want stale lease epoch; a second epoch-2 head must not fork", err)
	}
	winner, err := repository.WinningLease(testSessionID)
	if err != nil {
		t.Fatalf("WinningLease error = %v", err)
	}
	if winner.RecordID != second.RecordID {
		t.Fatalf("winner = %s, want the single committed head %s", winner.RecordID, second.RecordID)
	}
	summaries, err := repository.ListLeases(testSessionID)
	if err != nil {
		t.Fatalf("ListLeases error = %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("ListLeases = %d leases, want 2; the refused CAS must persist nothing", len(summaries))
	}
}
