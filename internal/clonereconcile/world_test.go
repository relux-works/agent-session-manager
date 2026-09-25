package clonereconcile_test

import (
	"sort"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/clonereadback"
	clonereconcile "github.com/relux-works/agent-session-manager/internal/clonereconcile"
)

// worldItem is one generated capture item: a candidate key with its
// tier-1 arm, tier-2 arm, and tier-3 row shape.
type worldItem struct {
	key           string // candidate key; "" synthesizes cand-<index>
	exclusion     string // non-empty => tier-1 exclusion arm (no row)
	normalization string // non-empty => tier-2 normalization arm (null-canonical row)
	disposition   string // row disposition; "" => excluded item, no row
	stagedSeeds   []string
	liveSeeds     []string
	noRow         bool // tiers link the item but no row covers it (fault cells)
}

// worldSpec is one generated reconciliation world: items plus the
// read-back outcome and validity variant.
type worldSpec struct {
	items        []worldItem
	synthRows    int       // extra synthesized rows (evidence-free)
	stagedCarry  *[]string // nil => reads carry the claimed universe; else the override seeds
	liveCarry    *[]string
	flipCheck    bool // validity variant: one check boolean false
	flipObserved bool // validity variant: observed native session differs
	errorFinding bool // validity variant: one error-severity finding
}

// universeSeeds returns the fixed staged/live blob universes every
// generated row claims within: staged-<i>/live-<i> for i in 0..5.
func universeSeeds() (staged, live []string) {
	for i := 0; i < 6; i++ {
		staged = append(staged, "staged-"+itoa(i))
		live = append(live, "live-"+itoa(i))
	}
	return staged, live
}

// sharedWorldDocs seals the plan, projected manifest, and the three
// read variants (present/absent/mismatched per side) once per test:
// generated rows always claim within the fixed universe, so one
// sealed read per outcome serves every cell honestly.
type sharedWorldDocs struct {
	planBytes      []byte
	projectedBytes []byte
	stagedPresent  clonereadback.ValidatedReadBack
	stagedAbsent   clonereadback.ValidatedReadBack
	stagedMismatch clonereadback.ValidatedReadBack
	livePresent    clonereadback.ValidatedReadBack
	liveAbsent     clonereadback.ValidatedReadBack
	liveMismatch   clonereadback.ValidatedReadBack
}

func sealSharedWorldDocs(t *testing.T) sharedWorldDocs {
	t.Helper()
	planBytes := mustBuildPlan(t, validPlanInput())
	planID := mustDecodePlan(t, planBytes).PlanID.String()
	projectedBytes := mustBuildProjected(t, validProjectedInput(planID))
	projectedID := mustDecodeProjected(t, projectedBytes).ManifestID.String()
	stagedUniverse, liveUniverse := universeSeeds()
	stagedPresent, _ := sealRead(t, "staged", planID, projectedID, evidenceFor(t, stagedUniverse))
	stagedAbsent, _ := sealRead(t, "staged", planID, projectedID, nil)
	stagedMismatch, _ := sealRead(t, "staged", planID, projectedID, evidenceFor(t, []string{"phantom-staged-1", "phantom-staged-2"}))
	livePresent, _ := sealRead(t, "live", planID, projectedID, evidenceFor(t, liveUniverse))
	liveAbsent, _ := sealRead(t, "live", planID, projectedID, nil)
	liveMismatch, _ := sealRead(t, "live", planID, projectedID, evidenceFor(t, []string{"phantom-live-1", "phantom-live-2"}))
	return sharedWorldDocs{
		planBytes:      planBytes,
		projectedBytes: projectedBytes,
		stagedPresent:  stagedPresent,
		stagedAbsent:   stagedAbsent,
		stagedMismatch: stagedMismatch,
		livePresent:    livePresent,
		liveAbsent:     liveAbsent,
		liveMismatch:   liveMismatch,
	}
}

// resolveReads maps a worldSpec's carry overrides to sealed reads:
// nil override => present variant; empty => absent; phantom seeds =>
// the mismatch variant (the override seeds themselves decide which
// mismatch read; any non-universe seed reads as mismatched).
func (docs sharedWorldDocs) resolveReads(spec worldSpec) (clonereadback.ValidatedReadBack, clonereadback.ValidatedReadBack) {
	staged := docs.stagedPresent
	if spec.stagedCarry != nil {
		switch {
		case len(*spec.stagedCarry) == 0:
			staged = docs.stagedAbsent
		case !carriesUniverse(*spec.stagedCarry, true):
			staged = docs.stagedMismatch
		}
	}
	live := docs.livePresent
	if spec.liveCarry != nil {
		switch {
		case len(*spec.liveCarry) == 0:
			live = docs.liveAbsent
		case !carriesUniverse(*spec.liveCarry, false):
			live = docs.liveMismatch
		}
	}
	return staged, live
}

// carriesUniverse reports whether override seeds cover the claimed
// universe of their side. The mismatch reads carry only phantoms,
// so any override missing a universe seed resolves as mismatched.
func carriesUniverse(override []string, staged bool) bool {
	universe, _ := universeSeeds()
	if !staged {
		_, universe = universeSeeds()
	}
	have := make(map[string]bool, len(override))
	for _, seed := range override {
		have[seed] = true
	}
	for _, seed := range universe {
		if !have[seed] {
			return false
		}
	}
	return true
}

// assembleWorld builds one ReconciliationInput from a worldSpec:
// tiers from the items, rows in owner order (source rows by key,
// synthesized after), reads from the shared docs, and the expected
// fidelity digest derived through the fidelity owner.
func assembleWorld(t *testing.T, docs sharedWorldDocs, spec worldSpec) clonereconcile.ReconciliationInput {
	t.Helper()
	var candidates []string
	var tier1 []clonereconcile.CandidateLink
	var tier2 []clonereconcile.RawLink
	var rows []clonefidelity.DispositionRecordInput
	for index, item := range spec.items {
		key := item.key
		if key == "" {
			key = "cand-" + itoa(index)
		}
		candidates = append(candidates, key)
		rawSeed := "raw-" + key
		if item.exclusion != "" {
			tier1 = append(tier1, clonereconcile.CandidateLink{Candidate: key, Exclusion: item.exclusion})
			continue
		}
		tier1 = append(tier1, clonereconcile.CandidateLink{Candidate: key, Raw: fixtureDigest(rawSeed)})
		canonicalSeed := "canon-" + key
		var canonical *string
		if item.normalization != "" {
			tier2 = append(tier2, clonereconcile.RawLink{Raw: fixtureDigest(rawSeed), Normalization: item.normalization})
		} else {
			tier2 = append(tier2, clonereconcile.RawLink{Raw: fixtureDigest(rawSeed), Canonical: fixtureDigest(canonicalSeed)})
			canonical = strptr(fixtureDigest(canonicalSeed))
		}
		if item.noRow {
			continue
		}
		staged := item.stagedSeeds
		if staged == nil && !isLossDispositionName(item.disposition) {
			staged = []string{"staged-" + itoa(index)}
		}
		live := item.liveSeeds
		if live == nil && !isLossDispositionName(item.disposition) {
			live = []string{"live-" + itoa(index)}
		}
		rows = append(rows, validRowInput(key, rawSeed, canonical, item.disposition, staged, live))
	}
	for k := 0; k < spec.synthRows; k++ {
		key := "synth-" + itoa(k)
		rows = append(rows, synthRowInput(key, "synth-ev-"+itoa(k), nil, nil))
	}
	sort.SliceStable(rows, func(i, j int) bool {
		ith := rows[i].Disposition == "synthesized"
		jth := rows[j].Disposition == "synthesized"
		if ith != jth {
			return jth
		}
		return rows[i].SourceItemKey < rows[j].SourceItemKey
	})
	staged, live := docs.resolveReads(spec)
	fidelity := validFidelityInput(rows, staged.ManifestID().String(), live.ManifestID().String())
	expected, err := clonefidelity.BuildFidelityReport(fidelity)
	if err != nil {
		// The assembler may legitimately produce owner-invalid
		// fidelity inputs (empty rows when every candidate is
		// excluded): the expected digest is then the empty
		// marker and production must refuse at the fidelity
		// build. The oracle predicts that refusal.
		expected = nil
	}
	expectedID := ""
	if expected != nil {
		report, err := clonefidelity.DecodeFidelityReport(expected)
		if err != nil {
			t.Fatalf("DecodeFidelityReport(assembled): %v", err)
		}
		expectedID = report.ReportID.String()
	}
	validation := validValidationFields(expectedID)
	if spec.flipCheck {
		validation.Checks.LiveStructuralValid = false
	}
	if spec.flipObserved {
		validation.ObservedTargetNativeSession = "other-native-session"
	}
	if spec.errorFinding {
		validation.Findings = fixtureFindings("error")
	}
	return clonereconcile.ReconciliationInput{
		Candidates: candidates,
		Tier1:      tier1,
		Tier2:      tier2,
		Fidelity:   fidelity,
		History:    clonereconcile.History{Staged: staged, Live: live},
		Plan:       docs.planBytes,
		Projected:  docs.projectedBytes,
		Validation: validation,
	}
}

// isLossDispositionName mirrors the production loss set for fixture
// defaulting: omitted and unrecoverable rows carry no evidence.
func isLossDispositionName(disposition string) bool {
	return disposition == "omitted" || disposition == "unrecoverable"
}
