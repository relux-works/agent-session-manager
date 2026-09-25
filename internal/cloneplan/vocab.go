package cloneplan

// This file owns the Section 13.14.2 projection closed
// vocabularies: the four plan strategies, the four plan profiles,
// the four target-operation actions, the three synthesized-event
// purposes, and the two resource kinds. Every table is quoted from
// the pinned specification text; the oracle tests retype each list
// from the spec and sweep admission and refusal through the
// production entries. The seven dispositions and the reason
// vocabulary stay owned by the internal/clonefidelity sibling.

// planStrategies is the exact four-member strategy vocabulary the
// Projection Plan 1.0.0 table states:
// same_environment_native_rewrite|target_native_writer|
// target_official_import|continuation_context. The fifth Section
// 13.14.2 strategy, archive_only, is refused on a plan: an archive
// report carries no plan at all.
var planStrategies = []string{
	"same_environment_native_rewrite",
	"target_native_writer",
	"target_official_import",
	"continuation_context",
}

// ValidPlanStrategy reports whether the name is a projection plan
// strategy.
func ValidPlanStrategy(strategy string) bool {
	for _, allowed := range planStrategies {
		if strategy == allowed {
			return true
		}
	}
	return false
}

// PlanStrategies returns the exact four-member plan strategy
// vocabulary in specification order. Callers iterate the returned
// copy; the table above stays the single owner.
func PlanStrategies() []string {
	out := make([]string, len(planStrategies))
	copy(out, planStrategies)
	return out
}

// planProfiles is the exact four-member profile vocabulary the
// Projection Plan 1.0.0 table states: strict_exact|maximal_safe|
// compact|messages_only. The fifth Section 13.14.2 profile,
// archive_only, is refused on a plan.
var planProfiles = []string{
	"strict_exact",
	"maximal_safe",
	"compact",
	"messages_only",
}

// ValidPlanProfile reports whether the name is a projection plan
// fidelity profile.
func ValidPlanProfile(profile string) bool {
	for _, allowed := range planProfiles {
		if profile == allowed {
			return true
		}
	}
	return false
}

// PlanProfiles returns the exact four-member plan profile
// vocabulary in specification order. Callers iterate the returned
// copy; the table above stays the single owner.
func PlanProfiles() []string {
	out := make([]string, len(planProfiles))
	copy(out, planProfiles)
	return out
}

// operationActions is the exact four-member action vocabulary
// Section 13.14.2 states for ProjectionTargetOperation:
// create_directory|write_blob|write_native_record|rebuild_index.
var operationActions = []string{
	"create_directory",
	"write_blob",
	"write_native_record",
	"rebuild_index",
}

// ValidOperationAction reports whether the name is a target
// operation action.
func ValidOperationAction(action string) bool {
	for _, allowed := range operationActions {
		if action == allowed {
			return true
		}
	}
	return false
}

// OperationActions returns the exact four-member action vocabulary
// in specification order. Callers iterate the returned copy; the
// table above stays the single owner.
func OperationActions() []string {
	out := make([]string, len(operationActions))
	copy(out, operationActions)
	return out
}

// synthesizedPurposes is the exact three-member purpose vocabulary
// Section 13.14.2 states for SynthesizedProjectionEvent:
// migration_checkpoint|summary|delimiter.
var synthesizedPurposes = []string{
	"migration_checkpoint",
	"summary",
	"delimiter",
}

// ValidSynthesizedPurpose reports whether the name is a synthesized
// projection event purpose.
func ValidSynthesizedPurpose(purpose string) bool {
	for _, allowed := range synthesizedPurposes {
		if purpose == allowed {
			return true
		}
	}
	return false
}

// SynthesizedPurposes returns the exact three-member purpose
// vocabulary in specification order. Callers iterate the returned
// copy; the table above stays the single owner.
func SynthesizedPurposes() []string {
	out := make([]string, len(synthesizedPurposes))
	copy(out, synthesizedPurposes)
	return out
}

// resourceKinds is the exact two-member kind vocabulary Section
// 13.14.2 states for ExpectedTargetResource and
// ProjectedObjectEntry: blob|directory.
var resourceKinds = []string{
	"blob",
	"directory",
}

// ValidResourceKind reports whether the name is a target resource
// kind.
func ValidResourceKind(kind string) bool {
	for _, allowed := range resourceKinds {
		if kind == allowed {
			return true
		}
	}
	return false
}

// ResourceKinds returns the exact two-member kind vocabulary in
// specification order. Callers iterate the returned copy; the table
// above stays the single owner.
func ResourceKinds() []string {
	out := make([]string, len(resourceKinds))
	copy(out, resourceKinds)
	return out
}
