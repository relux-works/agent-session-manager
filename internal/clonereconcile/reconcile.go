package clonereconcile

import (
	"sort"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/cloneplan"
	"github.com/relux-works/agent-session-manager/internal/clonereadback"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// CandidateLink reconciles one captured candidate at tier 1: exactly
// one of Raw (a raw-evidence digest) or Exclusion (a capture class)
// is set. Both set is double-counted; neither set is a missing
// link.
type CandidateLink struct {
	Candidate string
	Raw       string
	Exclusion string
}

// RawLink reconciles one raw item at tier 2: exactly one of
// Canonical (a canonical-object digest) or Normalization (a
// non-synthesized fidelity disposition) is set.
type RawLink struct {
	Raw           string
	Canonical     string
	Normalization string
}

// History is one sealed staged/live read-back pair: the target
// history ReadBackHistory decoded through the read-back owner. Only
// sealed reads reconcile; the zero value seals nothing and every
// pairing gate refuses it.
type History struct {
	Staged clonereadback.ValidatedReadBack
	Live   clonereadback.ValidatedReadBack
}

// ValidationFields carries the caller-supplied validation-report
// members Reconcile passes to the report owner. The staged/live
// manifest references, the plan/projected references, and the
// fidelity reference are derived by construction from the sealed
// reads and the derived reports, never carried. Valid is decided by
// the owner: the entry attempts true, then false, and exactly one
// seals.
type ValidationFields struct {
	OperationID                 string
	ProviderManifestID          string
	ExpectedTargetNativeSession string
	ObservedTargetNativeSession string
	TargetEnvironment           []byte
	Checks                      clonereadback.ValidationChecks
	Findings                    []byte
	Extensions                  map[string]any
	ExpectedFidelityReportID    string
}

// ReconciliationInput is the caller-supplied reconciliation
// candidate. Candidates with Tier1/Tier2 link it at tiers 1-2;
// Fidelity.Rows links it at tier 3; History pairs target evidence;
// Plan and Projected pair the projection inputs.
type ReconciliationInput struct {
	Candidates []string
	Tier1      []CandidateLink
	Tier2      []RawLink
	Fidelity   clonefidelity.FidelityReportInput
	History    History
	Plan       []byte
	Projected  []byte
	Validation ValidationFields
}

// Reconciliation is the proven result: the derived report bytes,
// the owner-decided valid bit, and the tier census counts.
type Reconciliation struct {
	FidelityReport   []byte
	FidelityID       string
	ValidationReport []byte
	Valid            bool
	Candidates       int
	RawItems         int
	CanonicalItems   int
	Rows             int
}

// ReadBackHistory reads back staged and live target history through
// the landed read-back owner: both documents decode via
// clonereadback.DecodeReadBackEvidenceManifest under their trusted
// read authorities, and the pair seals only in staged/live order. A
// swapped pair, a relabeled mode, or an authority the purpose does
// not grant refuses; the owner decides every shape, seal, and
// authority rule.
func ReadBackHistory(stagedBytes, liveBytes []byte, stagedAuthority, liveAuthority sessadapter.ReadAuthority) (History, error) {
	staged, err := clonereadback.DecodeReadBackEvidenceManifest(stagedBytes, stagedAuthority)
	if err != nil {
		return History{}, refuse("staged read-back invalid: %v", err)
	}
	live, err := clonereadback.DecodeReadBackEvidenceManifest(liveBytes, liveAuthority)
	if err != nil {
		return History{}, refuse("live read-back invalid: %v", err)
	}
	if err := checkHistoryPair(staged, live); err != nil {
		return History{}, err
	}
	return History{Staged: staged, Live: live}, nil
}

// checkHistoryPair enforces that the first sealed read is staged
// and the second is live. The zero value seals nothing: its mode
// reads empty and refuses here before any pairing gate touches it.
func checkHistoryPair(staged, live clonereadback.ValidatedReadBack) error {
	if staged.Mode() != "staged" {
		return refuse("reconciliation staged read is not a staged manifest")
	}
	if live.Mode() != "live" {
		return refuse("reconciliation live read is not a live manifest")
	}
	return nil
}

// Reconcile proves reconciliation completeness over the three
// tiers and derives both reports through their owners:
//
//   - tier 1: every captured candidate reconciles exactly once to
//     raw evidence or to a capture-class exclusion;
//   - tier 2: every raw item reconciles exactly once to a canonical
//     item or to a non-synthesized normalization disposition;
//   - tier 3: every included candidate and every canonical item
//     occurs in exactly one non-synthesized fidelity row, every
//     row's source evidence is its own candidate's tier-1 raw
//     (swapped, foreign, missing, and double-claimed digests
//     refuse), and every non-loss row resolves staged/live target
//     evidence into the sealed reads while every loss row carries
//     explicit loss with no evidence claim;
//   - pairing: plan, projected manifest, fidelity report, and both
//     reports' tuples bind the same target the reads observe.
//
// The fidelity report seals via clonefidelity, the plan and
// projected inputs decode via cloneplan, and the validation report
// seals via clonereadback, which alone decides the valid bit. Every
// rule is a refusal with a literal code, never a repair.
func Reconcile(input ReconciliationInput) (Reconciliation, error) {
	if input.Fidelity.Scope != "target" {
		return Reconciliation{}, refuse("reconciliation needs a target fidelity report")
	}
	included, rawSet, rawByCandidate, err := checkTier1(input.Candidates, input.Tier1)
	if err != nil {
		return Reconciliation{}, err
	}
	canonicalSet, _, canonicalByRaw, err := checkTier2(rawSet, input.Tier2)
	if err != nil {
		return Reconciliation{}, err
	}
	if err := checkHistoryPair(input.History.Staged, input.History.Live); err != nil {
		return Reconciliation{}, err
	}
	plan, err := cloneplan.DecodeProjectionPlan(input.Plan)
	if err != nil {
		return Reconciliation{}, refuse("projection plan invalid: %v", err)
	}
	if err := checkPlanPairing(input.History, plan); err != nil {
		return Reconciliation{}, err
	}
	projected, err := cloneplan.DecodeProjectedObjectManifest(input.Projected)
	if err != nil {
		return Reconciliation{}, refuse("projected object manifest invalid: %v", err)
	}
	if err := checkProjectedPairing(input.History, projected); err != nil {
		return Reconciliation{}, err
	}
	if err := checkNativePairing(input.History, plan, projected); err != nil {
		return Reconciliation{}, err
	}
	if err := checkRowsCarrySourceEvidence(input.Fidelity.Rows); err != nil {
		return Reconciliation{}, err
	}
	fidelityBytes, err := clonefidelity.BuildFidelityReport(input.Fidelity)
	if err != nil {
		return Reconciliation{}, refuse("fidelity report invalid: %v", err)
	}
	fidelity, err := clonefidelity.DecodeFidelityReport(fidelityBytes)
	if err != nil {
		return Reconciliation{}, refuse("fidelity report invalid: %v", err)
	}
	if err := checkTier3(included, rawByCandidate, canonicalByRaw, rawSet, fidelity.Rows, input.History); err != nil {
		return Reconciliation{}, err
	}
	if err := checkFidelityPairing(input.History, fidelity); err != nil {
		return Reconciliation{}, err
	}
	targetTuple, err := sessadapter.DecodeTuple(input.Validation.TargetEnvironment)
	if err != nil {
		return Reconciliation{}, refuse("validation target tuple invalid: %v", err)
	}
	if err := checkReportTuplePairing(input.History, targetTuple); err != nil {
		return Reconciliation{}, err
	}
	if err := checkValidationNativePairing(input.History, input.Validation.ExpectedTargetNativeSession); err != nil {
		return Reconciliation{}, err
	}
	expectedFidelity, err := scalar.ParseDigest(input.Validation.ExpectedFidelityReportID)
	if err != nil {
		return Reconciliation{}, refuse("expected fidelity report is not a digest: %v", err)
	}
	if expectedFidelity != fidelity.ReportID {
		return Reconciliation{}, refuse("expected fidelity report %q does not match the derived report %q",
			expectedFidelity.String(), fidelity.ReportID.String())
	}
	validationBytes, valid, err := sealValidationReport(input, fidelity)
	if err != nil {
		return Reconciliation{}, err
	}
	return Reconciliation{
		FidelityReport:   fidelityBytes,
		FidelityID:       fidelity.ReportID.String(),
		ValidationReport: validationBytes,
		Valid:            valid,
		Candidates:       len(input.Candidates),
		RawItems:         len(rawSet),
		CanonicalItems:   len(canonicalSet),
		Rows:             len(fidelity.Rows),
	}, nil
}

// checkCandidates enforces the tier-1 census domain: candidate keys
// are unique valid-UTF-8 strings of 1..512 characters. Character
// measure delegates to environ.StringLength.
func checkCandidates(candidates []string) error {
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if !utf8.ValidString(candidate) {
			return refuse("candidate key is not valid UTF-8")
		}
		if length := environ.StringLength(candidate); length < 1 || length > 512 {
			return refuse("candidate key is not a string[1..512]")
		}
		if seen[candidate] {
			return refuse("candidates carry duplicate %q", candidate)
		}
		seen[candidate] = true
	}
	return nil
}

// checkTier1 enforces "every captured candidate reconciles once to
// raw evidence or exclusion": the links cover the declared
// candidates exactly, each link carries exactly one arm, raw
// digests parse through the scalar owner and are claimed once, and
// exclusions name capture classes through the clonebundle owner. It
// returns the included candidates and the raw-evidence set the next
// tiers close over.
func checkTier1(candidates []string, links []CandidateLink) (map[string]bool, map[string]bool, map[string]string, error) {
	if err := checkCandidates(candidates); err != nil {
		return nil, nil, nil, err
	}
	declared := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		declared[candidate] = true
	}
	linked := make(map[string]bool, len(links))
	included := make(map[string]bool)
	rawSet := make(map[string]bool)
	rawByCandidate := make(map[string]string)
	for _, link := range links {
		if !declared[link.Candidate] {
			return nil, nil, nil, refuse("tier-1 link names undeclared candidate %q", link.Candidate)
		}
		if linked[link.Candidate] {
			return nil, nil, nil, refuse("tier-1 candidate %q reconciles twice", link.Candidate)
		}
		linked[link.Candidate] = true
		switch {
		case link.Raw != "" && link.Exclusion != "":
			return nil, nil, nil, refuse("tier-1 candidate %q claims both raw evidence and exclusion", link.Candidate)
		case link.Raw == "" && link.Exclusion == "":
			return nil, nil, nil, refuse("tier-1 candidate %q has neither raw evidence nor exclusion", link.Candidate)
		case link.Raw != "":
			raw, err := scalar.ParseDigest(link.Raw)
			if err != nil {
				return nil, nil, nil, refuse("tier-1 raw evidence %q is not a digest: %v", link.Raw, err)
			}
			if rawSet[raw.String()] {
				return nil, nil, nil, refuse("tier-1 raw evidence %q is claimed twice", raw.String())
			}
			rawSet[raw.String()] = true
			included[link.Candidate] = true
			rawByCandidate[link.Candidate] = raw.String()
		default:
			if !clonebundle.ValidCaptureClass(link.Exclusion) {
				return nil, nil, nil, refuse("tier-1 exclusion %q is outside the capture classes", link.Exclusion)
			}
		}
	}
	for _, candidate := range candidates {
		if !linked[candidate] {
			return nil, nil, nil, refuse("tier-1 candidate %q has no link to raw evidence or exclusion", candidate)
		}
	}
	return included, rawSet, rawByCandidate, nil
}

// checkTier2 enforces "every raw item to a canonical item or
// normalization disposition": the links cover the tier-1 raw set
// exactly, each link carries exactly one arm, canonical digests
// parse through the scalar owner and are claimed once (the tiers
// compose into an injective chain, because two raws sharing one
// canonical could not each sit in exactly one row), and
// normalizations name non-synthesized dispositions through the
// fidelity owner (a raw item cannot normalize into a synthesized
// canonical, which is sourceless by owner rule). It returns the
// canonical set and the normalized raw set tier 3 closes over.
func checkTier2(rawSet map[string]bool, links []RawLink) (map[string]bool, map[string]bool, map[string]string, error) {
	linked := make(map[string]bool, len(links))
	canonicalSet := make(map[string]bool)
	normalized := make(map[string]bool)
	canonicalByRaw := make(map[string]string)
	for _, link := range links {
		raw, err := scalar.ParseDigest(link.Raw)
		if err != nil {
			return nil, nil, nil, refuse("tier-2 raw item %q is not a digest: %v", link.Raw, err)
		}
		if !rawSet[raw.String()] {
			return nil, nil, nil, refuse("tier-2 link names unreconciled raw item %q", raw.String())
		}
		if linked[raw.String()] {
			return nil, nil, nil, refuse("tier-2 raw item %q reconciles twice", raw.String())
		}
		linked[raw.String()] = true
		switch {
		case link.Canonical != "" && link.Normalization != "":
			return nil, nil, nil, refuse("tier-2 raw item %q claims both a canonical item and a normalization disposition", raw.String())
		case link.Canonical == "" && link.Normalization == "":
			return nil, nil, nil, refuse("tier-2 raw item %q has neither a canonical item nor a normalization disposition", raw.String())
		case link.Canonical != "":
			canonical, err := scalar.ParseDigest(link.Canonical)
			if err != nil {
				return nil, nil, nil, refuse("tier-2 canonical item %q is not a digest: %v", link.Canonical, err)
			}
			if canonicalSet[canonical.String()] {
				return nil, nil, nil, refuse("tier-2 canonical item %q is claimed twice", canonical.String())
			}
			canonicalSet[canonical.String()] = true
			canonicalByRaw[raw.String()] = canonical.String()
		default:
			if !clonefidelity.ValidDisposition(link.Normalization) {
				return nil, nil, nil, refuse("tier-2 normalization %q is outside the dispositions", link.Normalization)
			}
			if link.Normalization == "synthesized" {
				return nil, nil, nil, refuse("tier-2 raw item %q normalizes to synthesized, which is sourceless", raw.String())
			}
			normalized[raw.String()] = true
		}
	}
	var missing []string
	for raw := range rawSet {
		if !linked[raw] {
			missing = append(missing, raw)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return nil, nil, nil, refuse("tier-2 raw item %q has no link to a canonical item or normalization disposition", missing[0])
	}
	return canonicalSet, normalized, canonicalByRaw, nil
}

// isLossDisposition reports whether a disposition is explicit
// target loss: omitted or unrecoverable. Loss rows carry a target
// disposition instead of target evidence; every other
// non-synthesized row must resolve staged and live evidence into
// the sealed reads.
func isLossDisposition(disposition string) bool {
	return disposition == "omitted" || disposition == "unrecoverable"
}

// checkRowsCarrySourceEvidence enforces that every non-synthesized
// fidelity row carries source evidence: a row with none is missing
// its candidate's tier-1 raw. It runs before the fidelity build so
// the tier-3 missing rule reports its own literal; the owner would
// otherwise refuse the empty list as a shape violation first.
func checkRowsCarrySourceEvidence(rows []clonefidelity.DispositionRecordInput) error {
	for _, row := range rows {
		if row.Disposition == "synthesized" {
			continue
		}
		if len(row.SourceEvidenceIDs) == 0 {
			return refuse("tier-3 row %q has no source evidence", row.SourceItemKey)
		}
	}
	return nil
}

// checkTier3 enforces "every canonical item to staged/live target
// evidence or a target disposition" over the owner-decoded fidelity
// rows: every included candidate and every canonical item occurs in
// exactly one non-synthesized row, excluded candidates take no row,
// each row's canonical agrees with the tier-1/tier-2 chain, each
// non-synthesized row's source evidence is its own candidate's
// tier-1 raw (a swapped, foreign, or double-claimed digest refuses),
// synthesized rows never claim captured source evidence, and every
// claimed evidence ID resolves into the sealed read of its side.
func checkTier3(
	included map[string]bool,
	rawByCandidate map[string]string,
	canonicalByRaw map[string]string,
	rawSet map[string]bool,
	rows []clonefidelity.DispositionRecord,
	history History,
) error {
	stagedBlobs := make(map[string]bool)
	for _, row := range history.Staged.Manifest().EvidenceObjects {
		stagedBlobs[row.BlobID.String()] = true
	}
	liveBlobs := make(map[string]bool)
	for _, row := range history.Live.Manifest().EvidenceObjects {
		liveBlobs[row.BlobID.String()] = true
	}
	// Every tier-1 raw belongs to exactly one candidate by the
	// tier-1 injectivity gate, so the reverse map is total on the
	// raw set: it names the owning candidate of a swapped digest.
	candidateByRaw := make(map[string]string, len(rawByCandidate))
	for candidate, raw := range rawByCandidate {
		candidateByRaw[raw] = candidate
	}
	// No raw item is double-counted across rows: the first digest
	// claimed by two non-synthesized rows refuses with both rows.
	// The census runs before the per-row chain checks so a shared
	// digest reports the double claim, while a pure swap (each
	// digest still claimed once) falls through to the chain check.
	claimedBy := make(map[string]string)
	for _, row := range rows {
		if row.Disposition == "synthesized" {
			continue
		}
		for _, evidence := range row.SourceEvidenceIDs {
			if first, ok := claimedBy[evidence.String()]; ok {
				return refuse("tier-3 source evidence %q is claimed by rows %q and %q",
					evidence.String(), first, row.SourceItemKey)
			}
			claimedBy[evidence.String()] = row.SourceItemKey
		}
	}
	coveredCandidates := make(map[string]bool)
	for _, row := range rows {
		if row.Disposition == "synthesized" {
			for _, evidence := range row.SourceEvidenceIDs {
				if rawSet[evidence.String()] {
					return refuse("tier-3 synthesized row %q claims source evidence %q", row.SourceItemKey, evidence.String())
				}
			}
			if err := checkRowEvidenceResolves(row, stagedBlobs, liveBlobs); err != nil {
				return err
			}
			continue
		}
		if !included[row.SourceItemKey] {
			return refuse("tier-3 row %q covers no included candidate", row.SourceItemKey)
		}
		// Row keys are unique across all rows by the fidelity
		// owner's seal, so a second row for this candidate
		// refuses at the fidelity build; canonical coverage is
		// unique by tier-2 injectivity plus the agreement check
		// below, so a second row naming this canonical refuses
		// there.
		coveredCandidates[row.SourceItemKey] = true
		raw := rawByCandidate[row.SourceItemKey]
		if want, ok := canonicalByRaw[raw]; ok {
			if row.CanonicalObjectID == nil {
				return refuse("tier-3 row %q needs canonical %q from tier 2", row.SourceItemKey, want)
			}
			if row.CanonicalObjectID.String() != want {
				return refuse("tier-3 row %q names canonical %q, tier 2 derives %q",
					row.SourceItemKey, row.CanonicalObjectID.String(), want)
			}
		} else if row.CanonicalObjectID != nil {
			// normalized[raw] holds here by tier-2 closure: the raw
			// took the normalization arm, so the row must carry a
			// null canonical.
			return refuse("tier-3 row %q names canonical %q under normalization",
				row.SourceItemKey, row.CanonicalObjectID.String())
		}
		// Every claimed digest binds to this row's own
		// candidate/raw chain: another candidate's raw reports
		// the swap with its owner, a digest outside every chain
		// reports the foreign claim.
		own := rawByCandidate[row.SourceItemKey]
		for _, evidence := range row.SourceEvidenceIDs {
			if evidence.String() == own {
				continue
			}
			if owner, ok := candidateByRaw[evidence.String()]; ok {
				return refuse("tier-3 row %q claims source evidence %q of candidate %q",
					row.SourceItemKey, evidence.String(), owner)
			}
			return refuse("tier-3 row %q claims unreconciled source evidence %q", row.SourceItemKey, evidence.String())
		}
		if isLossDisposition(row.Disposition) {
			if len(row.StagedEvidenceObjectIDs) > 0 {
				return refuse("tier-3 loss row %q claims staged evidence %q",
					row.SourceItemKey, row.StagedEvidenceObjectIDs[0].String())
			}
			if len(row.LiveEvidenceObjectIDs) > 0 {
				return refuse("tier-3 loss row %q claims live evidence %q",
					row.SourceItemKey, row.LiveEvidenceObjectIDs[0].String())
			}
			continue
		}
		if len(row.StagedEvidenceObjectIDs) == 0 {
			return refuse("tier-3 row %q has no staged target evidence", row.SourceItemKey)
		}
		if len(row.LiveEvidenceObjectIDs) == 0 {
			return refuse("tier-3 row %q has no live target evidence", row.SourceItemKey)
		}
		if err := checkRowEvidenceResolves(row, stagedBlobs, liveBlobs); err != nil {
			return err
		}
	}
	// Every canonical belongs to exactly one candidate's chain
	// by tier-2 injectivity, and each present row's canonical
	// agrees with its candidate's chain, so an uncovered
	// canonical always coincides with an uncovered candidate and
	// the candidate census below reports the shape.
	var missingCandidates []string
	for candidate := range included {
		if !coveredCandidates[candidate] {
			missingCandidates = append(missingCandidates, candidate)
		}
	}
	sort.Strings(missingCandidates)
	if len(missingCandidates) > 0 {
		return refuse("tier-3 candidate %q has no disposition row", missingCandidates[0])
	}
	return nil
}

// checkRowEvidenceResolves enforces that every evidence ID a row
// claims resolves into the sealed read of its side: staged IDs
// into staged blob IDs, live IDs into live blob IDs. Absent or
// mismatched read-back refuses with the side and the phantom ID.
func checkRowEvidenceResolves(
	row clonefidelity.DispositionRecord,
	stagedBlobs, liveBlobs map[string]bool,
) error {
	for _, evidence := range row.StagedEvidenceObjectIDs {
		if !stagedBlobs[evidence.String()] {
			return refuse("tier-3 row %q claims staged evidence %q outside the staged read",
				row.SourceItemKey, evidence.String())
		}
	}
	for _, evidence := range row.LiveEvidenceObjectIDs {
		if !liveBlobs[evidence.String()] {
			return refuse("tier-3 row %q claims live evidence %q outside the live read",
				row.SourceItemKey, evidence.String())
		}
	}
	return nil
}

// checkPlanPairing enforces that both sealed reads name the decoded
// projection plan: the reads aggregate the plan Reconcile pairs,
// never a neighbor.
func checkPlanPairing(history History, plan cloneplan.ProjectionPlan) error {
	if history.Staged.Manifest().ProjectionPlanID != plan.PlanID {
		return refuse("staged read names projection plan %q, want %q",
			history.Staged.Manifest().ProjectionPlanID.String(), plan.PlanID.String())
	}
	if history.Live.Manifest().ProjectionPlanID != plan.PlanID {
		return refuse("live read names projection plan %q, want %q",
			history.Live.Manifest().ProjectionPlanID.String(), plan.PlanID.String())
	}
	return nil
}

// checkProjectedPairing enforces that both sealed reads name the
// decoded projected object manifest.
func checkProjectedPairing(history History, projected cloneplan.ProjectedManifest) error {
	if history.Staged.Manifest().ProjectedObjectManifestID != projected.ManifestID {
		return refuse("staged read names projected manifest %q, want %q",
			history.Staged.Manifest().ProjectedObjectManifestID.String(), projected.ManifestID.String())
	}
	if history.Live.Manifest().ProjectedObjectManifestID != projected.ManifestID {
		return refuse("live read names projected manifest %q, want %q",
			history.Live.Manifest().ProjectedObjectManifestID.String(), projected.ManifestID.String())
	}
	return nil
}

// checkNativePairing enforces that both reads observe the same
// target native session and the decoded plan and projected manifest
// expect it: one target across all four documents.
func checkNativePairing(history History, plan cloneplan.ProjectionPlan, projected cloneplan.ProjectedManifest) error {
	staged := history.Staged.Manifest()
	live := history.Live.Manifest()
	if staged.ObservedTargetNativeSession != live.ObservedTargetNativeSession {
		return refuse("staged and live reads observe different native sessions")
	}
	if plan.ExpectedTargetNativeSession != staged.ObservedTargetNativeSession {
		return refuse("projection plan expects native session %q, reads observe %q",
			plan.ExpectedTargetNativeSession, staged.ObservedTargetNativeSession)
	}
	if projected.ExpectedTargetNativeSession != staged.ObservedTargetNativeSession {
		return refuse("projected manifest expects native session %q, reads observe %q",
			projected.ExpectedTargetNativeSession, staged.ObservedTargetNativeSession)
	}
	return nil
}

// checkFidelityPairing enforces that the derived fidelity report
// binds the sealed reads it reconciles: its staged/live manifest
// references name the reads, and its target tuple equals both
// observed tuples.
func checkFidelityPairing(history History, fidelity clonefidelity.FidelityReport) error {
	if fidelity.StagedReadBackEvidenceManifestID == nil ||
		*fidelity.StagedReadBackEvidenceManifestID != history.Staged.ManifestID() {
		return refuse("fidelity report does not bind the staged read")
	}
	if fidelity.LiveReadBackEvidenceManifestID == nil ||
		*fidelity.LiveReadBackEvidenceManifestID != history.Live.ManifestID() {
		return refuse("fidelity report does not bind the live read")
	}
	// A target-scope report always carries a target tuple by the
	// owner's branch-exact seal, so the dereference below is
	// total on decoded target reports.
	if *fidelity.TargetEnvironment != history.Staged.Manifest().ObservedEnvironment {
		return refuse("fidelity target tuple differs from the staged observed tuple")
	}
	if *fidelity.TargetEnvironment != history.Live.Manifest().ObservedEnvironment {
		return refuse("fidelity target tuple differs from the live observed tuple")
	}
	return nil
}

// checkReportTuplePairing enforces the validation report's
// target-tuple cross-match: the report tuple equals the staged
// observed tuple. The live side needs no second site: fidelity
// pairing already proved both observed tuples equal the fidelity
// target, so equality with staged is equality with live.
func checkReportTuplePairing(history History, target sessadapter.Tuple) error {
	if target != history.Staged.Manifest().ObservedEnvironment {
		return refuse("validation target tuple differs from the staged observed tuple")
	}
	return nil
}

// checkValidationNativePairing enforces that the validation report
// is about the observed target: its expected native session equals
// the reads' observed session. The expected/observed equality
// inside the report stays the owner's valid rule.
func checkValidationNativePairing(history History, expected string) error {
	if expected != history.Staged.Manifest().ObservedTargetNativeSession {
		return refuse("validation report expects native session %q, reads observe %q",
			expected, history.Staged.Manifest().ObservedTargetNativeSession)
	}
	return nil
}

// sealValidationReport derives the validation report through the
// read-back owner: references bind the sealed siblings and the
// derived reports by construction, and the valid bit is decided by
// the owner alone. The entry attempts true, then false; exactly one
// seals, because the owner refuses any claimed bit that disagrees
// with its decision.
func sealValidationReport(input ReconciliationInput, fidelity clonefidelity.FidelityReport) ([]byte, bool, error) {
	owner := clonereadback.ValidationReportInput{
		OperationID:                      input.Validation.OperationID,
		ProjectionPlanID:                 input.History.Staged.Manifest().ProjectionPlanID.String(),
		ProjectedObjectManifestID:        input.History.Staged.Manifest().ProjectedObjectManifestID.String(),
		StagedReadBackEvidenceManifestID: input.History.Staged.ManifestID().String(),
		LiveReadBackEvidenceManifestID:   input.History.Live.ManifestID().String(),
		TargetProviderManifestID:         input.Validation.ProviderManifestID,
		FidelityReportID:                 fidelity.ReportID.String(),
		ExpectedTargetNativeSession:      input.Validation.ExpectedTargetNativeSession,
		ObservedTargetNativeSession:      input.Validation.ObservedTargetNativeSession,
		TargetEnvironment:                input.Validation.TargetEnvironment,
		StagedStructuralValid:            input.Validation.Checks.StagedStructuralValid,
		LiveStructuralValid:              input.Validation.Checks.LiveStructuralValid,
		SemanticMarkerValid:              input.Validation.Checks.SemanticMarkerValid,
		IdentityValid:                    input.Validation.Checks.IdentityValid,
		WorkspaceBindingValid:            input.Validation.Checks.WorkspaceBindingValid,
		ResumeSurfaceValid:               input.Validation.Checks.ResumeSurfaceValid,
		SourceGenerationRevalidated:      input.Validation.Checks.SourceGenerationRevalidated,
		Findings:                         input.Validation.Findings,
		Extensions:                       input.Validation.Extensions,
	}
	owner.Valid = true
	sealed, err := clonereadback.BuildValidationReport(owner, input.History.Staged, input.History.Live)
	if err == nil {
		return sealed, true, nil
	}
	owner.Valid = false
	sealed, retryErr := clonereadback.BuildValidationReport(owner, input.History.Staged, input.History.Live)
	if retryErr == nil {
		return sealed, false, nil
	}
	return nil, false, refuse("validation report invalid: %v; %v", err, retryErr)
}
