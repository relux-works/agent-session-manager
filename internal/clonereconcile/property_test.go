package clonereconcile_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	clonereconcile "github.com/relux-works/agent-session-manager/internal/clonereconcile"
)

// Independent oracle for the reconciliation property. Every rule
// below is retyped from the pinned SPEC v0.7.0 Section 13.14.2 text
// (lines 10554-10556, 10581-10582, 10776-10777); the oracle shares
// no code with production and reads only the generation parameters,
// never the production verdict.
//
// Oracle rules (O1-O9):
//   O1. Every captured candidate reconciles once: exactly one
//       tier-1 arm. Zero or two arms refuse.
//   O2. Every raw item reconciles once: exactly one tier-2 arm.
//   O3. Every canonical item reconciles once: exactly one
//       non-synthesized row, with staged/live target evidence or a
//       target (omitted/unrecoverable) disposition.
//   O4. A synthesized row never claims captured source evidence.
//   O5. Claimed staged/live evidence resolves into the sealed read
//       of its side; absent or mismatched read-back refuses.
//   O6. The fidelity report binds both sealed reads.
//   O7. An empty row set cannot seal a fidelity report (the owner
//       requires 1..1000000 rows).
//   O8. Valid follows the report rule: every check boolean true,
//       matching native IDs, and no error finding.
//   O9. Every non-synthesized row's source evidence is its own
//       candidate's tier-1 raw: a swapped digest (another
//       candidate's raw), a foreign digest (outside every chain),
//       a missing digest, or a double-claimed digest (one raw
//       claimed by two rows) refuses.

// oracleExpectation is one independent verdict: admit with a valid
// bit, or refuse with a set of literal classes. Production reports
// the first break in its documented gate order; the oracle reports
// the SET of broken rules and the test asserts membership, so the
// oracle never mirrors production's (arbitrary) firing order. Each
// literal is additionally pinned exactly by its single-break gate
// test.
type oracleExpectation struct {
	admit    bool
	valid    bool
	literals []string
}

// faultSpec is one injected fault: a pre-assembly worldSpec
// transformer (only drop-row, which changes the row multiset the
// fidelity breakdowns derive from) or a post-assembly input
// mutation, plus the literal class production must report. An
// applier returns false when the world lacks the shape the fault
// needs; the cell then runs fault-free and the oracle predicts
// from the degraded parameters.
type faultSpec struct {
	name    string
	pre     func(*worldSpec) bool
	post    func(*clonereconcile.ReconciliationInput) bool
	literal string
}

func includedTier1Index(input *clonereconcile.ReconciliationInput) int {
	for i, link := range input.Tier1 {
		if link.Raw != "" {
			return i
		}
	}
	return -1
}

func exclusionTier1Index(input *clonereconcile.ReconciliationInput) int {
	for i, link := range input.Tier1 {
		if link.Exclusion != "" {
			return i
		}
	}
	return -1
}

func canonicalTier2Index(input *clonereconcile.ReconciliationInput) int {
	for i, link := range input.Tier2 {
		if link.Canonical != "" {
			return i
		}
	}
	return -1
}

func rowIndexByDisposition(input *clonereconcile.ReconciliationInput, want func(string) bool) int {
	for i, row := range input.Fidelity.Rows {
		if want(row.Disposition) {
			return i
		}
	}
	return -1
}

// nonSynthRowIndices returns the Fidelity.Rows indices of the
// non-synthesized rows in row order.
func nonSynthRowIndices(input *clonereconcile.ReconciliationInput) []int {
	var out []int
	for i, row := range input.Fidelity.Rows {
		if row.Disposition == "synthesized" {
			continue
		}
		out = append(out, i)
	}
	return out
}

// tierFaults enumerates the rotating injected faults.
func tierFaults() []faultSpec {
	return []faultSpec{
		{name: "none"},
		{name: "drop-tier1", literal: "has no link to raw evidence or exclusion", post: func(input *clonereconcile.ReconciliationInput) bool {
			if len(input.Tier1) == 0 {
				return false
			}
			input.Tier1 = input.Tier1[1:]
			return true
		}},
		{name: "dup-tier1", literal: "tier-1 candidate", post: func(input *clonereconcile.ReconciliationInput) bool {
			if len(input.Tier1) == 0 {
				return false
			}
			input.Tier1 = append(input.Tier1, input.Tier1[0])
			return true
		}},
		{name: "both-tier1", literal: "claims both raw evidence and exclusion", post: func(input *clonereconcile.ReconciliationInput) bool {
			i := includedTier1Index(input)
			if i < 0 {
				return false
			}
			input.Tier1[i].Exclusion = "durable_payload"
			return true
		}},
		{name: "neither-tier1", literal: "has neither raw evidence nor exclusion", post: func(input *clonereconcile.ReconciliationInput) bool {
			if len(input.Tier1) == 0 {
				return false
			}
			input.Tier1[0].Raw = ""
			input.Tier1[0].Exclusion = ""
			return true
		}},
		{name: "raw-twice", literal: "tier-1 raw evidence", post: func(input *clonereconcile.ReconciliationInput) bool {
			first, second := -1, -1
			for i, link := range input.Tier1 {
				if link.Raw == "" {
					continue
				}
				if first < 0 {
					first = i
				} else {
					second = i
					break
				}
			}
			if second < 0 {
				return false
			}
			input.Tier1[second].Raw = input.Tier1[first].Raw
			return true
		}},
		{name: "bad-exclusion", literal: "outside the capture classes", post: func(input *clonereconcile.ReconciliationInput) bool {
			if i := exclusionTier1Index(input); i >= 0 {
				input.Tier1[i].Exclusion = "frobnicate"
				return true
			}
			if len(input.Tier1) == 0 {
				return false
			}
			input.Tier1[0].Raw = ""
			input.Tier1[0].Exclusion = "frobnicate"
			return true
		}},
		{name: "drop-tier2", literal: "has no link to a canonical item or normalization disposition", post: func(input *clonereconcile.ReconciliationInput) bool {
			if len(input.Tier2) == 0 {
				return false
			}
			input.Tier2 = input.Tier2[1:]
			return true
		}},
		{name: "both-tier2", literal: "claims both a canonical item and a normalization disposition", post: func(input *clonereconcile.ReconciliationInput) bool {
			if len(input.Tier2) == 0 {
				return false
			}
			input.Tier2[0].Canonical = fixtureDigest("fault-canon")
			input.Tier2[0].Normalization = "omitted"
			return true
		}},
		{name: "synth-normalization", literal: "normalizes to synthesized, which is sourceless", post: func(input *clonereconcile.ReconciliationInput) bool {
			if len(input.Tier2) == 0 {
				return false
			}
			input.Tier2[0].Canonical = ""
			input.Tier2[0].Normalization = "synthesized"
			return true
		}},
		{name: "canon-twice", literal: "tier-2 canonical item", post: func(input *clonereconcile.ReconciliationInput) bool {
			first, second := -1, -1
			for i, link := range input.Tier2 {
				if link.Canonical == "" {
					continue
				}
				if first < 0 {
					first = i
				} else {
					second = i
					break
				}
			}
			if second < 0 {
				return false
			}
			input.Tier2[second].Canonical = input.Tier2[first].Canonical
			return true
		}},
		{name: "drop-row", literal: "has no disposition row", pre: func(spec *worldSpec) bool {
			for i := range spec.items {
				if spec.items[i].exclusion == "" && !spec.items[i].noRow {
					spec.items[i].noRow = true
					return true
				}
			}
			return false
		}},
		{name: "swap-canon", literal: "tier 2 derives", post: func(input *clonereconcile.ReconciliationInput) bool {
			first, second := -1, -1
			for i, row := range input.Fidelity.Rows {
				if row.Disposition == "synthesized" || row.CanonicalObjectID == nil {
					continue
				}
				if first < 0 {
					first = i
				} else {
					second = i
					break
				}
			}
			if second < 0 {
				return false
			}
			input.Fidelity.Rows[first].CanonicalObjectID, input.Fidelity.Rows[second].CanonicalObjectID =
				input.Fidelity.Rows[second].CanonicalObjectID, input.Fidelity.Rows[first].CanonicalObjectID
			return true
		}},
		{name: "synth-claims", literal: "tier-3 synthesized row", post: func(input *clonereconcile.ReconciliationInput) bool {
			i := rowIndexByDisposition(input, func(d string) bool { return d == "synthesized" })
			raw := includedTier1Index(input)
			if i < 0 || raw < 0 {
				return false
			}
			input.Fidelity.Rows[i].SourceEvidenceIDs = []string{input.Tier1[raw].Raw}
			return true
		}},
		{name: "loss-claims", literal: "tier-3 loss row", post: func(input *clonereconcile.ReconciliationInput) bool {
			i := rowIndexByDisposition(input, isLossDispositionName)
			if i < 0 {
				return false
			}
			input.Fidelity.Rows[i].StagedEvidenceObjectIDs = []string{fixtureDigest("staged-0")}
			return true
		}},
		{name: "drop-staged", literal: "has no staged target evidence", post: func(input *clonereconcile.ReconciliationInput) bool {
			i := rowIndexByDisposition(input, func(d string) bool {
				return d != "synthesized" && !isLossDispositionName(d)
			})
			if i < 0 {
				return false
			}
			input.Fidelity.Rows[i].StagedEvidenceObjectIDs = []string{}
			return true
		}},
		{name: "drop-live", literal: "has no live target evidence", post: func(input *clonereconcile.ReconciliationInput) bool {
			i := rowIndexByDisposition(input, func(d string) bool {
				return d != "synthesized" && !isLossDispositionName(d)
			})
			if i < 0 {
				return false
			}
			input.Fidelity.Rows[i].LiveEvidenceObjectIDs = []string{}
			return true
		}},
		{name: "ghost-link", literal: "names undeclared candidate", post: func(input *clonereconcile.ReconciliationInput) bool {
			input.Tier1 = append(input.Tier1, clonereconcile.CandidateLink{Candidate: "ghost", Raw: fixtureDigest("ghost-raw")})
			return true
		}},
		{name: "dup-candidate", literal: "carry duplicate", post: func(input *clonereconcile.ReconciliationInput) bool {
			if len(input.Candidates) == 0 {
				input.Candidates = []string{"z", "z"}
				return true
			}
			input.Candidates = append(input.Candidates, input.Candidates[0])
			return true
		}},
		{name: "bad-digest", literal: "is not a digest", post: func(input *clonereconcile.ReconciliationInput) bool {
			if len(input.Tier1) > 0 {
				input.Tier1[0].Raw = "not-a-digest"
				input.Tier1[0].Exclusion = ""
				return true
			}
			if len(input.Tier2) > 0 {
				input.Tier2[0].Raw = "not-a-digest"
				return true
			}
			input.Candidates = []string{"z"}
			input.Tier1 = []clonereconcile.CandidateLink{{Candidate: "z", Raw: "not-a-digest"}}
			return true
		}},
		{name: "swap-source-evidence", literal: "of candidate", post: func(input *clonereconcile.ReconciliationInput) bool {
			rows := nonSynthRowIndices(input)
			if len(rows) < 2 {
				return false
			}
			first, second := rows[0], rows[1]
			input.Fidelity.Rows[first].SourceEvidenceIDs, input.Fidelity.Rows[second].SourceEvidenceIDs =
				input.Fidelity.Rows[second].SourceEvidenceIDs, input.Fidelity.Rows[first].SourceEvidenceIDs
			return true
		}},
		{name: "foreign-source-evidence", literal: "claims unreconciled source evidence", post: func(input *clonereconcile.ReconciliationInput) bool {
			rows := nonSynthRowIndices(input)
			if len(rows) == 0 {
				return false
			}
			input.Fidelity.Rows[rows[0]].SourceEvidenceIDs = []string{fixtureDigest("phantom")}
			return true
		}},
		{name: "missing-source-evidence", literal: "has no source evidence", post: func(input *clonereconcile.ReconciliationInput) bool {
			rows := nonSynthRowIndices(input)
			if len(rows) == 0 {
				return false
			}
			input.Fidelity.Rows[rows[0]].SourceEvidenceIDs = []string{}
			return true
		}},
		{name: "double-claim-source-evidence", literal: "is claimed by rows", post: func(input *clonereconcile.ReconciliationInput) bool {
			// Targets the LAST two non-synthesized rows so the
			// N>=4-only narrowing (a double-claim census over a
			// three-row prefix) admits exactly the N>=4 cells:
			// at N<=3 every non-synthesized row sits inside the
			// prefix and the mutant behaves identically.
			rows := nonSynthRowIndices(input)
			if len(rows) < 2 {
				return false
			}
			secondToLast, last := rows[len(rows)-2], rows[len(rows)-1]
			claimed := append([]string{}, input.Fidelity.Rows[secondToLast].SourceEvidenceIDs...)
			input.Fidelity.Rows[last].SourceEvidenceIDs = claimed
			return true
		}},
	}
}

// readOutcome names one side's read-back state: the sealed read
// carries the claimed universe (present), nothing (absent), or only
// phantoms (mismatched).
type readOutcome int

const (
	outcomePresent readOutcome = iota
	outcomeAbsent
	outcomeMismatch
)

// allOutcomePairs is the read-back outcome vocabulary: staged ×
// live over present/absent/mismatched. Every capture set crosses
// all nine pairs.
var allOutcomePairs = [][2]readOutcome{
	{outcomePresent, outcomePresent},
	{outcomePresent, outcomeAbsent},
	{outcomePresent, outcomeMismatch},
	{outcomeAbsent, outcomePresent},
	{outcomeAbsent, outcomeAbsent},
	{outcomeAbsent, outcomeMismatch},
	{outcomeMismatch, outcomePresent},
	{outcomeMismatch, outcomeAbsent},
	{outcomeMismatch, outcomeMismatch},
}

// oracle predicts the verdict for one generated cell from the
// generation parameters. staged/live name the read-back outcomes;
// rowsPresent reports whether any row sealed; stagedClaimants and
// liveClaimants count the non-loss rows claiming evidence on each
// side after the applied fault; applied is the applied fault's
// literal ("" when no fault applied).
func oracle(spec worldSpec, applied string, staged, live readOutcome, rowsPresent bool, stagedClaimants, liveClaimants int) oracleExpectation {
	var literals []string
	if applied != "" {
		literals = append(literals, applied)
	}
	if !rowsPresent {
		literals = append(literals, "fidelity report invalid")
	}
	if staged != outcomePresent && stagedClaimants > 0 {
		literals = append(literals, "outside the staged read")
	}
	if live != outcomePresent && liveClaimants > 0 {
		literals = append(literals, "outside the live read")
	}
	if len(literals) > 0 {
		return oracleExpectation{admit: false, literals: literals}
	}
	return oracleExpectation{
		admit: true,
		valid: !spec.flipCheck && !spec.flipObserved && !spec.errorFinding,
	}
}

// claimantSummary counts the non-loss rows claiming evidence per
// side from the generation parameters: excluded and rowless items
// take no row, loss rows claim none, and the drop-staged/drop-live
// faults strip the first claimant of their side.
func claimantSummary(spec worldSpec, appliedFault string) (rowsPresent bool, staged, live int) {
	if spec.synthRows > 0 {
		rowsPresent = true
	}
	strippedStaged := appliedFault == "drop-staged"
	strippedLive := appliedFault == "drop-live"
	for _, item := range spec.items {
		if item.exclusion != "" || item.noRow {
			continue
		}
		rowsPresent = true
		if isLossDispositionName(item.disposition) {
			continue
		}
		if len(item.stagedSeeds) > 0 || (item.stagedSeeds == nil && !strippedStagedFirst(&strippedStaged)) {
			staged++
		}
		if len(item.liveSeeds) > 0 || (item.liveSeeds == nil && !strippedLiveFirst(&strippedLive)) {
			live++
		}
	}
	return rowsPresent, staged, live
}

func strippedStagedFirst(stripped *bool) bool {
	if *stripped {
		*stripped = false
		return true
	}
	return false
}

func strippedLiveFirst(stripped *bool) bool {
	if *stripped {
		*stripped = false
		return true
	}
	return false
}

// itemClass is one generated per-candidate class.
type itemClass int

const (
	classExact itemClass = iota
	classOpaque
	classExcluded
	classNormalized
	classLost
)

// allItemClasses is the capture-class vocabulary the generator
// spans: every class vector is drawn from these five classes.
var allItemClasses = []itemClass{classExact, classOpaque, classExcluded, classNormalized, classLost}

// className renders one capture class for shard labels.
func className(class itemClass) string {
	switch class {
	case classExact:
		return "exact"
	case classOpaque:
		return "opaque"
	case classExcluded:
		return "excluded"
	case classNormalized:
		return "normalized"
	case classLost:
		return "lost"
	default:
		panic("unknown item class")
	}
}

func (c itemClass) item(index int) worldItem {
	key := "cand-" + itoa(index)
	switch c {
	case classExact:
		return worldItem{key: key, disposition: "exact"}
	case classOpaque:
		return worldItem{key: key, disposition: "opaque_preserved"}
	case classExcluded:
		return worldItem{key: key, exclusion: "durable_payload"}
	case classNormalized:
		return worldItem{key: key, normalization: "summarized", disposition: "semantic"}
	case classLost:
		return worldItem{key: key, disposition: "omitted"}
	default:
		panic("unknown item class")
	}
}

// classVectors enumerates every class vector of length n over the
// five capture classes.
func classVectors(n int) [][]itemClass {
	if n == 0 {
		return [][]itemClass{{}}
	}
	var out [][]itemClass
	for _, tail := range classVectors(n - 1) {
		for _, class := range allItemClasses {
			vector := append([]itemClass{class}, tail...)
			out = append(out, vector)
		}
	}
	return out
}

// classVectorsWithPrefix enumerates every class vector of length n
// carrying the given prefix, for deterministic parallel sharding.
func classVectorsWithPrefix(n int, prefix []itemClass) [][]itemClass {
	var out [][]itemClass
	for _, suffix := range classVectors(n - len(prefix)) {
		vector := append(append([]itemClass{}, prefix...), suffix...)
		out = append(out, vector)
	}
	return out
}

// applyOutcome maps one read-back outcome to carry overrides: nil
// keeps the present universe, empty seals the absent read, phantoms
// seal the mismatched read.
func applyOutcome(spec *worldSpec, staged, live readOutcome) {
	if staged == outcomeAbsent {
		empty := []string{}
		spec.stagedCarry = &empty
	} else if staged == outcomeMismatch {
		phantoms := []string{"phantom-staged-1", "phantom-staged-2"}
		spec.stagedCarry = &phantoms
	}
	if live == outcomeAbsent {
		empty := []string{}
		spec.liveCarry = &empty
	} else if live == outcomeMismatch {
		phantoms := []string{"phantom-live-1", "phantom-live-2"}
		spec.liveCarry = &phantoms
	}
}

// rotateValidity assigns one validity variant per cell: nominal,
// one flipped check boolean, a diverged observed native session, or
// one error-severity finding. The oracle predicts the valid bit
// from the same flags.
func rotateValidity(spec *worldSpec, cell int) {
	switch cell % 4 {
	case 1:
		spec.flipCheck = true
	case 2:
		spec.flipObserved = true
	case 3:
		spec.errorFinding = true
	}
}

// runPropertyCell assembles one world, applies one rotating fault,
// and checks the production verdict against the oracle. It returns
// the rotated fault name and whether it applied, for the shard's
// fault histogram.
//
// An assembled input that cannot seal through the fidelity owner
// needs no separate guard: production reports "fidelity report
// invalid", which the oracle predicts only for rowless worlds, so
// any other seal failure fails the cell below loudly.
func runPropertyCell(t *testing.T, docs sharedWorldDocs, faults []faultSpec, spec worldSpec, staged, live readOutcome, cell, n int) (string, bool) {
	t.Helper()
	rotateValidity(&spec, cell)
	fault := faults[cell%len(faults)]
	applied := ""
	if fault.pre != nil && fault.pre(&spec) {
		applied = fault.literal
	}
	input := assembleWorld(t, docs, spec)
	appliedFault := ""
	if fault.post != nil {
		if fault.post(&input) {
			appliedFault = fault.name
			applied = fault.literal
		}
	} else if fault.pre != nil && applied != "" {
		appliedFault = fault.name
	}
	rowsPresent, stagedClaimants, liveClaimants := claimantSummary(spec, appliedFault)
	want := oracle(spec, applied, staged, live, rowsPresent, stagedClaimants, liveClaimants)
	checkPropertyCell(t, input, want, cell, n, spec.synthRows, staged, live, fault.name, applied != "")
	return fault.name, applied != ""
}

// expectedShardCells computes the executable coverage target from
// the class and outcome vocabularies: 5^(n-len(prefix)) vectors × 9
// outcomes. A shard fails unless its executed count equals this, so
// sampling or duplicates can never be reported as exhaustive.
func expectedShardCells(n int, prefix []itemClass) int {
	vectors := 1
	for i := len(prefix); i < n; i++ {
		vectors *= len(allItemClasses)
	}
	return vectors * len(allOutcomePairs)
}

// encodeVector renders one class vector as a comparable key.
func encodeVector(vector []itemClass) string {
	var b strings.Builder
	for _, class := range vector {
		b.WriteByte(byte('0' + int(class)))
	}
	return b.String()
}

// encodePrefix renders one shard prefix for labels and logs.
func encodePrefix(prefix []itemClass) string {
	names := make([]string, 0, len(prefix))
	for _, class := range prefix {
		names = append(names, className(class))
	}
	return strings.Join(names, "+")
}

// runEnumerationShard enumerates every class vector of length n
// with the given prefix, crossed with all nine read-back outcomes,
// against the independent oracle. The synthesized-row count, the
// injected fault, and the validity variant rotate per cell. Each
// shard seals its own shared documents so parallel shards share no
// mutable state.
func runEnumerationShard(t *testing.T, faults []faultSpec, n int, prefix []itemClass) {
	t.Helper()
	docs := sealSharedWorldDocs(t)
	vectors := classVectorsWithPrefix(n, prefix)
	seen := make(map[string]struct{}, len(vectors))
	outcomes := make(map[string]map[[2]readOutcome]struct{}, len(vectors))
	applied := make(map[string]int, len(faults))
	skipped := make(map[string]int, len(faults))
	seenSynth := [2]bool{}
	cell := 0
	for _, vector := range vectors {
		key := encodeVector(vector)
		if _, dup := seen[key]; dup {
			t.Fatalf("shard n=%d prefix=%q: duplicate class vector %q", n, encodePrefix(prefix), key)
		}
		seen[key] = struct{}{}
		set := make(map[[2]readOutcome]struct{}, len(allOutcomePairs))
		outcomes[key] = set
		for _, outcome := range allOutcomePairs {
			spec := worldSpec{synthRows: cell % 2}
			seenSynth[spec.synthRows] = true
			for index, class := range vector {
				spec.items = append(spec.items, class.item(index))
			}
			applyOutcome(&spec, outcome[0], outcome[1])
			name, ok := runPropertyCell(t, docs, faults, spec, outcome[0], outcome[1], cell, n)
			if ok {
				applied[name]++
			} else {
				skipped[name]++
			}
			set[outcome] = struct{}{}
			cell++
		}
	}
	// Vector uniqueness (above) plus the count below proves
	// exhaustiveness over the 5-ary space: 5^k distinct length-k
	// vectors over 5 symbols is the whole space.
	if want := expectedShardCells(n, prefix); cell != want {
		t.Fatalf("shard n=%d prefix=%q: executed %d cells, want %d from the vocabularies",
			n, encodePrefix(prefix), cell, want)
	}
	for key, set := range outcomes {
		if len(set) != len(allOutcomePairs) {
			t.Fatalf("shard n=%d prefix=%q: vector %q ran %d outcomes, want %d",
				n, encodePrefix(prefix), key, len(set), len(allOutcomePairs))
		}
	}
	if cell >= 2 && !(seenSynth[0] && seenSynth[1]) {
		t.Fatalf("shard n=%d prefix=%q: synth rotation stuck at one value", n, encodePrefix(prefix))
	}
	t.Logf("shard n=%d prefix=%q: %d cells, faults applied=%v skipped=%v",
		n, encodePrefix(prefix), cell, applied, skipped)
}

// TestReconciliationProperty enumerates the reconciliation product
// for N<=3: every capture-class vector crossed with the nine
// read-back outcomes, rotating synthesized rows, injected faults
// (tiers plus the O9 source-evidence relation class), and validity
// variants. Every cell runs the production Reconcile entry; the
// independent oracle predicts admit/refuse and the valid bit.
func TestReconciliationProperty(t *testing.T) {
	faults := tierFaults()
	for n := 0; n <= 3; n++ {
		n := n
		t.Run("N="+itoa(n), func(t *testing.T) {
			t.Parallel()
			runEnumerationShard(t, faults, n, nil)
		})
	}
}

// TestReconciliationPropertyLargeN enumerates the reconciliation
// product for N=4..6 with the same oracle: N=4 runs as one shard,
// N=5 shards by first class, N=6 shards by first-class pair, so
// every shard executes the same vectors × outcomes product it
// asserts.
func TestReconciliationPropertyLargeN(t *testing.T) {
	faults := tierFaults()
	t.Run("N=4", func(t *testing.T) {
		t.Parallel()
		runEnumerationShard(t, faults, 4, nil)
	})
	for _, first := range allItemClasses {
		first := first
		t.Run("N=5/prefix="+className(first), func(t *testing.T) {
			t.Parallel()
			runEnumerationShard(t, faults, 5, []itemClass{first})
		})
		for _, second := range allItemClasses {
			second := second
			prefix := []itemClass{first, second}
			t.Run("N=6/prefix="+className(first)+"+"+className(second), func(t *testing.T) {
				t.Parallel()
				runEnumerationShard(t, faults, 6, prefix)
			})
		}
	}
}

// checkPropertyCell runs one property cell through the production
// entry and checks the oracle verdict: refusal membership over the
// broken-rule set, or admission with the oracle's valid bit.
func checkPropertyCell(t *testing.T, input clonereconcile.ReconciliationInput, want oracleExpectation, cell, n, synth int, staged, live readOutcome, fault string, applied bool) {
	t.Helper()
	got, err := clonereconcile.Reconcile(input)
	label := fmt.Sprintf("cell=%d n=%d synth=%d staged=%d live=%d fault=%s applied=%v", cell, n, synth, staged, live, fault, applied)
	if !want.admit {
		if err == nil {
			t.Fatalf("%s: Reconcile admitted, oracle refuses %q", label, want.literals)
		}
		if !errors.Is(err, clonereconcile.ErrInvalid) {
			t.Fatalf("%s: error = %v, want errors.Is ErrInvalid", label, err)
		}
		for _, literal := range want.literals {
			if strings.Contains(err.Error(), literal) {
				return
			}
		}
		t.Fatalf("%s: Reconcile error %q matches none of oracle %q", label, err.Error(), want.literals)
	}
	if err != nil {
		t.Fatalf("%s: Reconcile refused %q, oracle admits", label, err.Error())
	}
	if got.Valid != want.valid {
		t.Fatalf("%s: Reconcile valid=%v, oracle valid=%v", label, got.Valid, want.valid)
	}
}
