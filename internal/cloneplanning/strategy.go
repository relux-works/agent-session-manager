package cloneplanning

import (
	"github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// This file selects the projection strategy and profile branch.
// Selection is pair-neutral: it consumes only resolved boolean
// signals and closed names, never concrete environment identities
// or manifest bytes. Target-branch admission (the unstable refusal
// and maximal-safe completeness) stays owned by
// clonesnap.AdmitForTarget; SelectProfile routes the branch, the
// caller admits the manifest under the returned profile. Strategy
// and profile spellings stay owned by the internal/clonefidelity
// and internal/cloneplan vocabularies; the literals below quote
// them, and the agreement test pins the quote.

// Branch is the projection branch a profile selects: target or
// archive. Archive carries no Projection Plan (Section 13.14.2:
// the archive report is targetless, and the projection-plan
// contract forbids archive_only).
type Branch string

const (
	// BranchTarget carries a Projection Plan.
	BranchTarget Branch = "target"
	// BranchArchive carries no plan: G0/G1 closure only.
	BranchArchive Branch = "archive"
)

// Section 13.14.2 strategy spellings, quoted from the
// internal/clonefidelity owner.
const (
	strategyNativeRewrite = "same_environment_native_rewrite"
	strategyNativeWriter  = "target_native_writer"
	strategyOfficial      = "target_official_import"
	strategyContinuation  = "continuation_context"
	strategyArchive       = "archive_only"
)

// profileArchive quotes the archive profile spelling.
const profileArchive = "archive_only"

// SelectStrategy selects one Section 13.14.2 strategy from resolved
// pair signals. sameEnvironment reports whether source and target
// share one native environment; officialImport reports whether the
// target adapter offers the official_import capability;
// continuation reports an explicit continuation-context request;
// archive reports an explicit archive-only request. Priority is
// archive, then continuation, then the pair/capability arm: an
// explicit archive request always wins, an explicit continuation
// request wins over the pair, a cross-environment pair with the
// vendor importer uses target_official_import, a cross-environment
// pair without it uses target_native_writer, and a same-environment
// pair uses same_environment_native_rewrite.
func SelectStrategy(sameEnvironment, officialImport, continuation, archive bool) (string, error) {
	if archive {
		return strategyArchive, nil
	}
	if continuation {
		return strategyContinuation, nil
	}
	if !sameEnvironment {
		if officialImport {
			return strategyOfficial, nil
		}
		return strategyNativeWriter, nil
	}
	return strategyNativeRewrite, nil
}

// SelectProfile validates one requested profile and routes its
// branch. archive_only routes the archive branch, which carries no
// plan. The four plan profiles route the target branch; the caller
// admits the capture manifest under the returned profile through
// the landed clonesnap.AdmitForTarget gate. Anything else refuses
// through the plan-profile gate.
func SelectProfile(requested string) (Branch, string, error) {
	if !validText(requested) {
		return "", "", invalid("projection profile is not valid UTF-8")
	}
	if requested == profileArchive {
		return BranchArchive, requested, nil
	}
	if !cloneplan.ValidPlanProfile(requested) {
		return "", "", invalid("projection profile %q is outside strict_exact|maximal_safe|compact|messages_only|archive_only", requested)
	}
	return BranchTarget, requested, nil
}
