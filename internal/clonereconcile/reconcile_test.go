package clonereconcile_test

import (
	"bytes"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/clonereadback"
	clonereconcile "github.com/relux-works/agent-session-manager/internal/clonereconcile"
)

// TestReconcileAdmitsValidTargetHistory proves the end-to-end
// admission: tiers close, rows resolve into the sealed reads,
// pairing binds one target, and both reports derive through their
// owners with the owner-decided valid bit.
func TestReconcileAdmitsValidTargetHistory(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	input := gateBaseline(t, docs)
	got, err := clonereconcile.Reconcile(input)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if !got.Valid {
		t.Fatalf("Reconcile() valid = false, want true")
	}
	if got.Candidates != 4 || got.RawItems != 3 || got.CanonicalItems != 2 || got.Rows != 4 {
		t.Fatalf("Reconcile() census = %d/%d/%d/%d, want 4/3/2/4",
			got.Candidates, got.RawItems, got.CanonicalItems, got.Rows)
	}
	if got.FidelityID != independentFidelityID(t, got.FidelityReport) {
		t.Fatalf("Reconcile() fidelity ID %q disagrees with the independent recompute", got.FidelityID)
	}
	fidelity, err := clonefidelity.DecodeFidelityReport(got.FidelityReport)
	if err != nil {
		t.Fatalf("DecodeFidelityReport(derived) error = %v", err)
	}
	if fidelity.ReportID.String() != got.FidelityID {
		t.Fatalf("derived fidelity ID %q != census ID %q", fidelity.ReportID.String(), got.FidelityID)
	}
	validation, err := clonereadback.DecodeValidationReport(got.ValidationReport, input.History.Staged, input.History.Live)
	if err != nil {
		t.Fatalf("DecodeValidationReport(derived) error = %v", err)
	}
	if !validation.Valid {
		t.Fatalf("derived validation report valid = false, want true")
	}
	if validation.FidelityReportID.String() != got.FidelityID {
		t.Fatalf("derived validation binds fidelity %q, want %q", validation.FidelityReportID.String(), got.FidelityID)
	}
}

// TestReconcileValidFollowsOwner pins that the valid bit is decided
// by the read-back owner alone: each failing check shape seals with
// valid=false through the production entry, and the sealed bytes
// decode back through the owner with the same bit.
func TestReconcileValidFollowsOwner(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	cases := []struct {
		name   string
		mutate func(*clonereconcile.ReconciliationInput)
	}{
		{name: "all-true", mutate: func(input *clonereconcile.ReconciliationInput) {}},
		{name: "check-false", mutate: func(input *clonereconcile.ReconciliationInput) {
			input.Validation.Checks.LiveStructuralValid = false
		}},
		{name: "observed-differs", mutate: func(input *clonereconcile.ReconciliationInput) {
			input.Validation.ObservedTargetNativeSession = "other-native-session"
		}},
		{name: "error-finding", mutate: func(input *clonereconcile.ReconciliationInput) {
			input.Validation.Findings = fixtureFindings("error")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := gateBaseline(t, docs)
			tc.mutate(&input)
			got, err := clonereconcile.Reconcile(input)
			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			want := tc.name == "all-true"
			if got.Valid != want {
				t.Fatalf("Reconcile() valid = %v, want %v", got.Valid, want)
			}
			decoded, err := clonereadback.DecodeValidationReport(got.ValidationReport, input.History.Staged, input.History.Live)
			if err != nil {
				t.Fatalf("DecodeValidationReport(derived) error = %v", err)
			}
			if decoded.Valid != want {
				t.Fatalf("decoded valid = %v, want %v", decoded.Valid, want)
			}
		})
	}
}

// TestReconcileDeterminism pins bytes-identical outputs for
// identical inputs across ten reconciliations.
func TestReconcileDeterminism(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	input := gateBaseline(t, docs)
	first, err := clonereconcile.Reconcile(input)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	for i := 0; i < 9; i++ {
		next, err := clonereconcile.Reconcile(gateBaseline(t, docs))
		if err != nil {
			t.Fatalf("Reconcile() error = %v", err)
		}
		if !bytes.Equal(next.FidelityReport, first.FidelityReport) {
			t.Fatalf("reconciliation %d fidelity bytes differ", i)
		}
		if !bytes.Equal(next.ValidationReport, first.ValidationReport) {
			t.Fatalf("reconciliation %d validation bytes differ", i)
		}
	}
}

// TestReconcilePurity pins that neither entry mutates its inputs:
// the candidate, link, row, and document inputs are byte-equal
// before and after.
func TestReconcilePurity(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	input := gateBaseline(t, docs)
	beforePlan := append([]byte{}, input.Plan...)
	beforeProjected := append([]byte{}, input.Projected...)
	beforeTuple := append([]byte{}, input.Validation.TargetEnvironment...)
	beforeFindings := append([]byte{}, input.Validation.Findings...)
	beforeRows := len(input.Fidelity.Rows)
	if _, err := clonereconcile.Reconcile(input); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if !bytes.Equal(input.Plan, beforePlan) {
		t.Fatalf("Reconcile mutated the plan bytes")
	}
	if !bytes.Equal(input.Projected, beforeProjected) {
		t.Fatalf("Reconcile mutated the projected bytes")
	}
	if !bytes.Equal(input.Validation.TargetEnvironment, beforeTuple) {
		t.Fatalf("Reconcile mutated the validation tuple")
	}
	if !bytes.Equal(input.Validation.Findings, beforeFindings) {
		t.Fatalf("Reconcile mutated the validation findings")
	}
	if len(input.Fidelity.Rows) != beforeRows {
		t.Fatalf("Reconcile mutated the fidelity rows")
	}
}

// TestReadBackHistoryAdmitsSealedPair pins the read-back entry: two
// owner-sealed documents decode through the read-back owner into a
// staged/live history.
func TestReadBackHistoryAdmitsSealedPair(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	planID := mustDecodePlan(t, docs.planBytes).PlanID.String()
	projectedID := mustDecodeProjected(t, docs.projectedBytes).ManifestID.String()
	_, stagedBytes := sealRead(t, "staged", planID, projectedID, evidenceFor(t, mustUniverseStaged()))
	_, liveBytes := sealRead(t, "live", planID, projectedID, evidenceFor(t, mustUniverseLive()))
	history, err := clonereconcile.ReadBackHistory(
		stagedBytes, liveBytes,
		fixtureAuthority(t, "target_staged"), fixtureAuthority(t, "target_live"))
	if err != nil {
		t.Fatalf("ReadBackHistory() error = %v", err)
	}
	if history.Staged.Mode() != "staged" || history.Live.Mode() != "live" {
		t.Fatalf("ReadBackHistory() modes = %q/%q, want staged/live", history.Staged.Mode(), history.Live.Mode())
	}
	if history.Staged.ManifestID() == history.Live.ManifestID() {
		t.Fatalf("ReadBackHistory() sealed identical staged/live reads")
	}
}

// TestReadBackHistoryRefusals pins the read-back entry refusals:
// malformed documents refuse with the side literal, and a swapped
// pair refuses at the read-back owner (the authority-derived mode
// binds each document before the pair gate runs).
func TestReadBackHistoryRefusals(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	planID := mustDecodePlan(t, docs.planBytes).PlanID.String()
	projectedID := mustDecodeProjected(t, docs.projectedBytes).ManifestID.String()
	_, stagedBytes := sealRead(t, "staged", planID, projectedID, evidenceFor(t, mustUniverseStaged()))
	_, liveBytes := sealRead(t, "live", planID, projectedID, evidenceFor(t, mustUniverseLive()))
	stagedAuth := fixtureAuthority(t, "target_staged")
	liveAuth := fixtureAuthority(t, "target_live")

	if _, err := clonereconcile.ReadBackHistory([]byte("not-json"), liveBytes, stagedAuth, liveAuth); err == nil {
		t.Fatalf("ReadBackHistory(garbage staged) admitted")
	} else {
		requireRefusal(t, err, "staged read-back invalid")
	}
	if _, err := clonereconcile.ReadBackHistory(stagedBytes, []byte("not-json"), stagedAuth, liveAuth); err == nil {
		t.Fatalf("ReadBackHistory(garbage live) admitted")
	} else {
		requireRefusal(t, err, "live read-back invalid")
	}
	// Swapped documents under swapped authorities still decode,
	// but the pair gate sees live-first.
	if _, err := clonereconcile.ReadBackHistory(liveBytes, stagedBytes, liveAuth, stagedAuth); err == nil {
		t.Fatalf("ReadBackHistory(swapped pair) admitted")
	} else {
		requireRefusal(t, err, "reconciliation staged read is not a staged manifest")
	}
	// A staged document under the live authority refuses at the
	// read-back owner: modes cannot be relabeled.
	if _, err := clonereadback.DecodeReadBackEvidenceManifest(stagedBytes, liveAuth); err == nil {
		t.Fatalf("DecodeReadBackEvidenceManifest(staged bytes, live authority) admitted")
	}
	if _, err := clonereconcile.ReadBackHistory(stagedBytes, liveBytes, liveAuth, liveAuth); err == nil {
		t.Fatalf("ReadBackHistory(staged bytes, live authority) admitted")
	} else {
		requireRefusal(t, err, "staged read-back invalid")
	}
}

// TestReadBackHistoryEmptyInput pins that empty documents refuse
// with the side literal: the wrap forwards the owner's refusal
// for the empty member of the malformed class.
func TestReadBackHistoryEmptyInput(t *testing.T) {
	docs := sealSharedWorldDocs(t)
	planID := mustDecodePlan(t, docs.planBytes).PlanID.String()
	projectedID := mustDecodeProjected(t, docs.projectedBytes).ManifestID.String()
	_, stagedBytes := sealRead(t, "staged", planID, projectedID, evidenceFor(t, mustUniverseStaged()))
	_, liveBytes := sealRead(t, "live", planID, projectedID, evidenceFor(t, mustUniverseLive()))
	stagedAuth := fixtureAuthority(t, "target_staged")
	liveAuth := fixtureAuthority(t, "target_live")

	t.Run("empty-staged", func(t *testing.T) {
		_, err := clonereconcile.ReadBackHistory([]byte{}, liveBytes, stagedAuth, liveAuth)
		requireRefusal(t, err, "staged read-back invalid")
	})
	t.Run("empty-live", func(t *testing.T) {
		_, err := clonereconcile.ReadBackHistory(stagedBytes, []byte{}, stagedAuth, liveAuth)
		requireRefusal(t, err, "live read-back invalid")
	})
}
