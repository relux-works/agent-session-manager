package cloneplanning

import (
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file folds planned mappings to target effects: callable
// tools, pending actions, and target token accounting. Historical
// tools are inert and source usage is not target accounting
// (Section 13.14.1), so the fold is always empty: no mapping admits
// a live effect. Mappings carry no token field by type, so token
// counts cannot reach the accounting decision at all; the sweep
// proves every mapping the planner emits folds to zero.
//
// Mapping validation delegates to the record owner,
// internal/clonefidelity, through the ValidateDispositionRow seam:
// the Section 13.14.2 record rules (disposition and reason
// vocabularies, string and count bounds, sorted-unique reasons,
// the exact/non-exact reason-set coupling, the synthesized
// canonical rule) have exactly one implementation, the owner's.
// The only checks kept here are the shared UTF-8 pre-gate (R-UTF8:
// invalid bytes refuse with the package literal before any owner
// step) and the zero fold itself.

// TargetEffects is the planned session's live surface: callable
// tools, pending actions, and target token accounting. Planning
// from captured history always yields the zero value.
type TargetEffects struct {
	CallableTools      []string
	PendingActions     []string
	TargetInputTokens  uint64
	TargetOutputTokens uint64
}

// Owner-row scaffolding for the members the Mapping type does not
// carry: fixed valid source key, class, and explanation. The
// source evidence digest derives from the row itself (see
// ownerRowForMapping); canonical object and target locator stay
// null.
const (
	effectsSourceKey   = "target-effects"
	effectsSourceClass = "target_effects"
	effectsExplanation = "planned target effects fold"
)

// ownerRowForMapping renders one planning mapping as a fidelity
// disposition row candidate: the mapping's disposition and reason
// set verbatim, with scaffolding for the members the mapping type
// does not carry. The source evidence digest is the SHA-256 of
// the disposition and reasons, so distinct mappings never share
// scaffolding by constant and no evidence value is hand-picked.
// Canonical object and target locator render null: the mapping
// carries neither, so the owner's synthesized rule always sees a
// null canonical and its source-evidence rule always sees the
// derived digest.
func ownerRowForMapping(mapping Mapping) clonefidelity.DispositionRecordInput {
	preimage := make([]byte, 0, len(mapping.Disposition)+8)
	preimage = append(preimage, mapping.Disposition...)
	preimage = append(preimage, 0)
	for _, reason := range mapping.Reasons {
		preimage = append(preimage, reason...)
		preimage = append(preimage, 0)
	}
	return clonefidelity.DispositionRecordInput{
		SourceItemKey:     effectsSourceKey,
		SourceClass:       effectsSourceClass,
		SourceEvidenceIDs: []string{scalar.SHA256Digest(preimage).String()},
		Disposition:       mapping.Disposition,
		ReasonCodes:       append([]string(nil), mapping.Reasons...),
		Explanation:       effectsExplanation,
	}
}

// checkMappingViaOwner validates one mapping through the fidelity
// record owner. The refusal keeps both sentinels matchable: the
// package sentinel names the context, the owner sentinel carries
// the violated rule.
func checkMappingViaOwner(context string, mapping Mapping) error {
	if err := clonefidelity.ValidateDispositionRow(ownerRowForMapping(mapping)); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInvalid, context, err)
	}
	return nil
}

// PlanTargetEffects folds planned mappings to their target
// effects: always empty. Every mapping is validated first: the
// shared UTF-8 gate refuses invalid bytes with the package
// literal, then the fidelity record owner validates the
// disposition and reason set, so malformed mappings refuse with
// the owner's rule instead of folding.
func PlanTargetEffects(mappings []Mapping) (TargetEffects, error) {
	for index, mapping := range mappings {
		if !validText(mapping.Disposition) {
			return TargetEffects{}, invalid("target effects mapping[%d] disposition is not valid UTF-8", index)
		}
		for _, reason := range mapping.Reasons {
			if !validText(reason) {
				return TargetEffects{}, invalid("target effects mapping[%d] reason is not valid UTF-8", index)
			}
		}
		if err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {
			return TargetEffects{}, err
		}
	}
	return TargetEffects{}, nil
}
