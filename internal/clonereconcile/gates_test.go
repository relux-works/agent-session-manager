package clonereconcile_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	clonereconcile "github.com/relux-works/agent-session-manager/internal/clonereconcile"
)

// gateBaseline is the four-candidate world every gate row mutates:
// an exact row, an omitted loss row, a normalized semantic row, one
// excluded candidate, and one synthesized row.
func gateBaseline(t *testing.T, docs sharedWorldDocs) clonereconcile.ReconciliationInput {
	t.Helper()
	return assembleWorld(t, docs, worldSpec{items: []worldItem{
		{key: "cand-0", disposition: "exact"},
		{key: "cand-1", disposition: "omitted"},
		{key: "cand-2", normalization: "summarized", disposition: "semantic"},
		{key: "cand-3", exclusion: "durable_payload"},
	}, synthRows: 1})
}

// gateRow is one single-break refusal cell: mutate breaks exactly
// one gate and literal pins its exact refusal text.
type gateRow struct {
	name    string
	mutate  func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput)
	literal string
	world   func(t *testing.T, docs sharedWorldDocs) clonereconcile.ReconciliationInput
}

func gateRefusalTable() []gateRow {
	raw0 := fixtureDigest("raw-cand-0")
	canon0 := fixtureDigest("canon-cand-0")
	canon1 := fixtureDigest("canon-cand-1")
	phantom := fixtureDigest("phantom")
	return []gateRow{
		// Candidate hygiene.
		{name: "candidate-utf8", literal: "candidate key is not valid UTF-8", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Candidates[0] = "bad\xff"
		}},
		{name: "candidate-empty", literal: "candidate key is not a string[1..512]", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Candidates[0] = ""
		}},
		{name: "candidate-too-long", literal: "candidate key is not a string[1..512]", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Candidates[0] = strings.Repeat("a", 513)
		}},
		{name: "candidate-duplicate", literal: `candidates carry duplicate "cand-0"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Candidates = append(input.Candidates, "cand-0")
		}},
		// Tier 1.
		{name: "tier1-undeclared", literal: `tier-1 link names undeclared candidate "ghost"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier1 = append(input.Tier1, clonereconcile.CandidateLink{Candidate: "ghost", Raw: phantom})
		}},
		{name: "tier1-twice", literal: `tier-1 candidate "cand-0" reconciles twice`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier1 = append(input.Tier1, input.Tier1[0])
		}},
		{name: "tier1-missing", literal: `tier-1 candidate "cand-0" has no link to raw evidence or exclusion`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier1 = input.Tier1[1:]
		}},
		{name: "tier1-both", literal: `tier-1 candidate "cand-0" claims both raw evidence and exclusion`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier1[0].Exclusion = "durable_payload"
		}},
		{name: "tier1-neither", literal: `tier-1 candidate "cand-0" has neither raw evidence nor exclusion`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier1[0].Raw = ""
			input.Tier1[0].Exclusion = ""
		}},
		{name: "tier1-raw-not-digest", literal: `tier-1 raw evidence "bogus" is not a digest`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier1[0].Raw = "bogus"
			input.Tier1[0].Exclusion = ""
		}},
		{name: "tier1-raw-twice", literal: `tier-1 raw evidence "` + raw0 + `" is claimed twice`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier1[1].Raw = input.Tier1[0].Raw
		}},
		{name: "tier1-bad-exclusion", literal: `tier-1 exclusion "frobnicate" is outside the capture classes`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier1[3].Exclusion = "frobnicate"
		}},
		// Tier 2.
		{name: "tier2-raw-not-digest", literal: `tier-2 raw item "bogus" is not a digest`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2[0].Raw = "bogus"
		}},
		{name: "tier2-unknown-raw", literal: `tier-2 link names unreconciled raw item "` + phantom + `"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2[0].Raw = phantom
		}},
		{name: "tier2-twice", literal: `tier-2 raw item "` + raw0 + `" reconciles twice`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2 = append(input.Tier2, input.Tier2[0])
		}},
		{name: "tier2-both", literal: `tier-2 raw item "` + raw0 + `" claims both a canonical item and a normalization disposition`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2[0].Normalization = "omitted"
		}},
		{name: "tier2-neither", literal: `tier-2 raw item "` + raw0 + `" has neither a canonical item nor a normalization disposition`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2[0].Canonical = ""
			input.Tier2[0].Normalization = ""
		}},
		{name: "tier2-canon-not-digest", literal: `tier-2 canonical item "bogus" is not a digest`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2[0].Canonical = "bogus"
		}},
		{name: "tier2-canon-twice", literal: `tier-2 canonical item "` + canon0 + `" is claimed twice`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2[1].Canonical = input.Tier2[0].Canonical
		}},
		{name: "tier2-bad-normalization", literal: `tier-2 normalization "frobnicate" is outside the dispositions`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2[2].Normalization = "frobnicate"
		}},
		{name: "tier2-synth-normalization", literal: `tier-2 raw item "` + fixtureDigest("raw-cand-2") + `" normalizes to synthesized, which is sourceless`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2[2].Normalization = "synthesized"
		}},
		{name: "tier2-missing", literal: `tier-2 raw item "` + raw0 + `" has no link to a canonical item or normalization disposition`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Tier2 = input.Tier2[1:]
		}},
		// Tier 3.
		{name: "tier3-synth-claims", literal: `tier-3 synthesized row "synth-0" claims source evidence "` + raw0 + `"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[3].SourceEvidenceIDs = []string{raw0}
		}},
		{name: "tier3-unknown-key", literal: `tier-3 row "cand-0x" covers no included candidate`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].SourceItemKey = "cand-0x"
		}},
		{name: "tier3-excluded-key", literal: `tier-3 row "cand-0" covers no included candidate`, world: func(t *testing.T, docs sharedWorldDocs) clonereconcile.ReconciliationInput {
			return assembleWorld(t, docs, worldSpec{items: []worldItem{
				{key: "cand-0", exclusion: "credential"},
				{key: "cand-1", disposition: "exact"},
				{key: "cand-2", disposition: "semantic"},
			}})
		}, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].SourceItemKey = "cand-0"
		}},
		{name: "tier3-needs-canonical", literal: `tier-3 row "cand-0" needs canonical "` + canon0 + `" from tier 2`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].CanonicalObjectID = nil
		}},
		{name: "tier3-wrong-canonical", literal: `tier-3 row "cand-0" names canonical "` + canon1 + `", tier 2 derives "` + canon0 + `"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].CanonicalObjectID, input.Fidelity.Rows[1].CanonicalObjectID =
				input.Fidelity.Rows[1].CanonicalObjectID, input.Fidelity.Rows[0].CanonicalObjectID
		}},
		{name: "tier3-canon-under-normalization", literal: `tier-3 row "cand-2" names canonical "` + phantom + `" under normalization`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[2].CanonicalObjectID = strptr(phantom)
		}},
		{name: "tier3-unreconciled-evidence", literal: `tier-3 row "cand-0" claims unreconciled source evidence "` + phantom + `"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].SourceEvidenceIDs = []string{phantom}
		}},
		{name: "tier3-swapped-evidence", literal: `tier-3 row "cand-0" claims source evidence "` + fixtureDigest("raw-cand-1") + `" of candidate "cand-1"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].SourceEvidenceIDs, input.Fidelity.Rows[1].SourceEvidenceIDs =
				input.Fidelity.Rows[1].SourceEvidenceIDs, input.Fidelity.Rows[0].SourceEvidenceIDs
		}},
		{name: "tier3-missing-evidence", literal: `tier-3 row "cand-0" has no source evidence`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].SourceEvidenceIDs = []string{}
		}},
		{name: "tier3-double-claimed-evidence", literal: `tier-3 source evidence "` + raw0 + `" is claimed by rows "cand-0" and "cand-1"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[1].SourceEvidenceIDs = input.Fidelity.Rows[0].SourceEvidenceIDs
		}},
		{name: "tier3-loss-claims-staged", literal: `tier-3 loss row "cand-1" claims staged evidence "` + fixtureDigest("staged-0") + `"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[1].StagedEvidenceObjectIDs = []string{fixtureDigest("staged-0")}
		}},
		{name: "tier3-loss-claims-live", literal: `tier-3 loss row "cand-1" claims live evidence "` + fixtureDigest("live-0") + `"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[1].LiveEvidenceObjectIDs = []string{fixtureDigest("live-0")}
		}},
		{name: "tier3-no-staged", literal: `tier-3 row "cand-0" has no staged target evidence`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].StagedEvidenceObjectIDs = []string{}
		}},
		{name: "tier3-no-live", literal: `tier-3 row "cand-0" has no live target evidence`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].LiveEvidenceObjectIDs = []string{}
		}},
		{name: "tier3-staged-outside", literal: `tier-3 row "cand-0" claims staged evidence "` + phantom + `" outside the staged read`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].StagedEvidenceObjectIDs = []string{phantom}
		}},
		{name: "tier3-live-outside", literal: `tier-3 row "cand-0" claims live evidence "` + phantom + `" outside the live read`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows[0].LiveEvidenceObjectIDs = []string{phantom}
		}},
		{name: "tier3-missing-row", literal: `tier-3 candidate "cand-0" has no disposition row`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Rows = input.Fidelity.Rows[1:]
			events, blocks, bytes := concentratedBreakdowns(input.Fidelity.Rows)
			input.Fidelity.EventKindCounts = events
			input.Fidelity.ContentBlockCounts = blocks
			input.Fidelity.ByteCounts = bytes
		}},
		{name: "tier3-duplicate-keys-refused-by-owner", literal: "fidelity report invalid", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			// A second row for one candidate refuses at the
			// fidelity build (owner key uniqueness), wrapped in
			// the reconciliation refusal.
			input.Fidelity.Rows[1].SourceItemKey = "cand-0"
		}},
		{name: "tier3-duplicate-canonical-refused-by-agreement", literal: `tier-3 row "cand-1" names canonical "` + canon0 + `", tier 2 derives "` + canon1 + `"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			// Tier-2 injectivity plus per-row agreement make a
			// second row for one canonical disagree here.
			input.Fidelity.Rows[1].CanonicalObjectID = input.Fidelity.Rows[0].CanonicalObjectID
		}},
		// Scope and history.
		{name: "scope-archive", literal: "reconciliation needs a target fidelity report", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.Scope = "archive"
		}},
		{name: "staged-not-staged", literal: "reconciliation staged read is not a staged manifest", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.History.Staged = input.History.Live
		}},
		{name: "live-not-live", literal: "reconciliation live read is not a live manifest", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.History.Live = input.History.Staged
		}},
		{name: "zero-history", literal: "reconciliation staged read is not a staged manifest", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.History = clonereconcile.History{}
		}},
		// Plan and projected pairing.
		{name: "plan-garbage", literal: "projection plan invalid", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Plan = []byte("not-json")
		}},
		{name: "plan-empty", literal: "projection plan invalid", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Plan = []byte{}
		}},
		{name: "projected-garbage", literal: "projected object manifest invalid", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Projected = []byte("not-json")
		}},
		{name: "projected-empty", literal: "projected object manifest invalid", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Projected = []byte{}
		}},
		{name: "staged-plan-mismatch", literal: "staged read names projection plan", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			other := validPlanInput()
			other.StrategyRationale = "neighbor plan"
			input.Plan = mustBuildPlan(t, other)
		}},
		{name: "live-plan-mismatch", literal: "live read names projection plan", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			other := validPlanInput()
			other.StrategyRationale = "neighbor plan"
			planB := mustBuildPlan(t, other)
			planBID := mustDecodePlan(t, planB).PlanID.String()
			projectedID := mustDecodeProjected(t, input.Projected).ManifestID.String()
			stagedB, _ := sealRead(t, "staged", planBID, projectedID, evidenceFor(t, mustUniverseStaged()))
			input.History.Staged = stagedB
			input.Plan = planB
		}},
		{name: "staged-projected-mismatch", literal: "staged read names projected manifest", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			planID := mustDecodePlan(t, input.Plan).PlanID.String()
			other := validProjectedInput(planID)
			other.TotalBytes = 129
			input.Projected = mustBuildProjected(t, other)
		}},
		{name: "live-projected-mismatch", literal: "live read names projected manifest", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			planID := mustDecodePlan(t, input.Plan).PlanID.String()
			other := validProjectedInput(planID)
			other.TotalBytes = 129
			projectedB := mustBuildProjected(t, other)
			projectedBID := mustDecodeProjected(t, projectedB).ManifestID.String()
			stagedB, _ := sealRead(t, "staged", planID, projectedBID, evidenceFor(t, mustUniverseStaged()))
			input.History.Staged = stagedB
			input.Projected = projectedB
		}},
		{name: "reads-differ-sessions", literal: "staged and live reads observe different native sessions", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			planID := mustDecodePlan(t, input.Plan).PlanID.String()
			projectedID := mustDecodeProjected(t, input.Projected).ManifestID.String()
			liveOther, _ := sealReadFull(t, "live", planID, projectedID, evidenceFor(t, mustUniverseLive()), "other-session", fixtureTuple())
			input.History.Live = liveOther
		}},
		{name: "plan-expects-other", literal: "projection plan expects native session", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			planID := mustDecodePlan(t, input.Plan).PlanID.String()
			projectedID := mustDecodeProjected(t, input.Projected).ManifestID.String()
			stagedOther, _ := sealReadFull(t, "staged", planID, projectedID, evidenceFor(t, mustUniverseStaged()), "other-session", fixtureTuple())
			liveOther, _ := sealReadFull(t, "live", planID, projectedID, evidenceFor(t, mustUniverseLive()), "other-session", fixtureTuple())
			input.History.Staged = stagedOther
			input.History.Live = liveOther
		}},
		{name: "projected-expects-other", literal: "projected manifest expects native session", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			planInput := validPlanInput()
			planInput.ExpectedTargetNativeSession = "other-session"
			planB := mustBuildPlan(t, planInput)
			planBID := mustDecodePlan(t, planB).PlanID.String()
			projectedID := mustDecodeProjected(t, input.Projected).ManifestID.String()
			stagedOther, _ := sealReadFull(t, "staged", planBID, projectedID, evidenceFor(t, mustUniverseStaged()), "other-session", fixtureTuple())
			liveOther, _ := sealReadFull(t, "live", planBID, projectedID, evidenceFor(t, mustUniverseLive()), "other-session", fixtureTuple())
			input.History.Staged = stagedOther
			input.History.Live = liveOther
			input.Plan = planB
		}},
		// Fidelity pairing.
		{name: "fidelity-staged-unbound", literal: "fidelity report does not bind the staged read", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.StagedReadBackEvidenceManifestID = strptr(phantom)
		}},
		{name: "fidelity-live-unbound", literal: "fidelity report does not bind the live read", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.LiveReadBackEvidenceManifestID = strptr(phantom)
		}},
		{name: "fidelity-tuple-staged", literal: "fidelity target tuple differs from the staged observed tuple", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Fidelity.TargetEnvironment = fixtureTupleOther()
		}},
		{name: "fidelity-tuple-live", literal: "fidelity target tuple differs from the live observed tuple", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			planID := mustDecodePlan(t, input.Plan).PlanID.String()
			projectedID := mustDecodeProjected(t, input.Projected).ManifestID.String()
			liveOther, _ := sealReadFull(t, "live", planID, projectedID, evidenceFor(t, mustUniverseLive()), fixtureNativeSession, fixtureTupleOther())
			input.History.Live = liveOther
			input.Fidelity.LiveReadBackEvidenceManifestID = strptr(liveOther.ManifestID().String())
		}},
		// Validation pairing and seal.
		{name: "validation-tuple-garbage", literal: "validation target tuple invalid", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Validation.TargetEnvironment = []byte("not-json")
		}},
		{name: "validation-tuple-empty", literal: "validation target tuple invalid", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Validation.TargetEnvironment = []byte{}
		}},
		{name: "validation-tuple-staged", literal: "validation target tuple differs from the staged observed tuple", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Validation.TargetEnvironment = fixtureTupleOther()
		}},
		{name: "validation-expects-other", literal: `validation report expects native session "other-session", reads observe "target-native-session-1"`, mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Validation.ExpectedTargetNativeSession = "other-session"
		}},
		{name: "expected-fidelity-garbage", literal: "expected fidelity report is not a digest", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Validation.ExpectedFidelityReportID = "bogus"
		}},
		{name: "expected-fidelity-empty", literal: "expected fidelity report is not a digest", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Validation.ExpectedFidelityReportID = ""
		}},
		{name: "expected-fidelity-mismatch", literal: "expected fidelity report", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Validation.ExpectedFidelityReportID = phantom
		}},
		{name: "validation-malformed", literal: "validation report invalid", mutate: func(t *testing.T, docs sharedWorldDocs, input *clonereconcile.ReconciliationInput) {
			input.Validation.Findings = []byte("not-json")
		}},
	}
}

func mustUniverseStaged() []string {
	staged, _ := universeSeeds()
	return staged
}

func mustUniverseLive() []string {
	_, live := universeSeeds()
	return live
}

// TestReconcileGateRefusals pins every refusal site exactly: each
// row breaks one gate and asserts its literal code and detail at
// the production Reconcile entry.
func TestReconcileGateRefusals(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	for _, row := range gateRefusalTable() {
		t.Run(row.name, func(t *testing.T) {
			build := gateBaseline
			if row.world != nil {
				build = row.world
			}
			input := build(t, docs)
			row.mutate(t, docs, &input)
			_, err := clonereconcile.Reconcile(input)
			requireRefusal(t, err, row.literal)
		})
	}
}

// TestReconcileSourceEvidenceChainBinding is the regression test
// for review finding source-evidence-chain-mismatch (CR rev2): each
// non-synthesized fidelity row's source evidence must be the
// evidence of its own candidate/raw chain — global membership in
// the raw set is not enough. All four relation breaks refuse at the
// production Reconcile entry with a literal code and detail.
func TestReconcileSourceEvidenceChainBinding(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	raw0 := fixtureDigest("raw-cand-0")
	raw1 := fixtureDigest("raw-cand-1")
	phantom := fixtureDigest("phantom")
	cases := []struct {
		name    string
		mutate  func(*clonereconcile.ReconciliationInput)
		literal string
	}{
		{name: "swapped", literal: `tier-3 row "cand-0" claims source evidence "` + raw1 + `" of candidate "cand-1"`,
			mutate: func(input *clonereconcile.ReconciliationInput) {
				input.Fidelity.Rows[0].SourceEvidenceIDs, input.Fidelity.Rows[1].SourceEvidenceIDs =
					input.Fidelity.Rows[1].SourceEvidenceIDs, input.Fidelity.Rows[0].SourceEvidenceIDs
			}},
		{name: "foreign", literal: `tier-3 row "cand-0" claims unreconciled source evidence "` + phantom + `"`,
			mutate: func(input *clonereconcile.ReconciliationInput) {
				input.Fidelity.Rows[0].SourceEvidenceIDs = []string{phantom}
			}},
		{name: "missing", literal: `tier-3 row "cand-0" has no source evidence`,
			mutate: func(input *clonereconcile.ReconciliationInput) {
				input.Fidelity.Rows[0].SourceEvidenceIDs = []string{}
			}},
		{name: "double-claimed", literal: `tier-3 source evidence "` + raw0 + `" is claimed by rows "cand-0" and "cand-1"`,
			mutate: func(input *clonereconcile.ReconciliationInput) {
				input.Fidelity.Rows[1].SourceEvidenceIDs = input.Fidelity.Rows[0].SourceEvidenceIDs
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := gateBaseline(t, docs)
			tc.mutate(&input)
			_, err := clonereconcile.Reconcile(input)
			requireRefusal(t, err, tc.literal)
		})
	}
}

// TestReconcileMultibyteCandidateMeasure pins the character (not
// byte) string measure at the candidate gate: 512 é admit, 513
// refuse.
func TestReconcileMultibyteCandidateMeasure(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	admit := gateBaseline(t, docs)
	admit.Candidates[0] = strings.Repeat("é", 512)
	admit.Tier1[0].Candidate = strings.Repeat("é", 512)
	admit.Fidelity.Rows[0].SourceItemKey = strings.Repeat("é", 512)
	// Re-sort: the multibyte key sorts after the ASCII keys, so
	// move the row to keep owner order.
	rows := admit.Fidelity.Rows
	admit.Fidelity.Rows = []clonefidelity.DispositionRecordInput{rows[1], rows[2], rows[0], rows[3]}
	relinkExpectedFidelity(t, &admit)
	if _, err := clonereconcile.Reconcile(admit); err != nil {
		t.Fatalf("Reconcile(512-char key) error = %v", err)
	}
	refuse := gateBaseline(t, docs)
	refuse.Candidates[0] = strings.Repeat("é", 513)
	_, err := clonereconcile.Reconcile(refuse)
	requireRefusal(t, err, "candidate key is not a string[1..512]")
}

// relinkExpectedFidelity re-derives the expected fidelity digest
// through the fidelity owner after a test mutation changes the
// fidelity input.
func relinkExpectedFidelity(t *testing.T, input *clonereconcile.ReconciliationInput) {
	t.Helper()
	sealed, err := clonefidelity.BuildFidelityReport(input.Fidelity)
	if err != nil {
		t.Fatalf("BuildFidelityReport(relinked) error = %v", err)
	}
	report, err := clonefidelity.DecodeFidelityReport(sealed)
	if err != nil {
		t.Fatalf("DecodeFidelityReport(relinked) error = %v", err)
	}
	input.Validation.ExpectedFidelityReportID = report.ReportID.String()
}

// TestReconcilePlanDecodesThroughOwner pins that the plan pairing
// reads the owner-decoded plan: a plan whose bytes the owner
// refuses on a semantic rule never reaches pairing.
func TestReconcilePlanDecodesThroughOwner(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	input := gateBaseline(t, docs)
	var document map[string]any
	if err := json.Unmarshal(input.Plan, &document); err != nil {
		t.Fatalf("decode plan: %v", err)
	}
	document["strategy"] = "archive_only"
	tampered, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal tampered plan: %v", err)
	}
	input.Plan = tampered
	_, err = clonereconcile.Reconcile(input)
	requireRefusal(t, err, "projection plan invalid")
}
