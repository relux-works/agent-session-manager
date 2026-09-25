package cloneplanning

import (
	"sort"

	"github.com/relux-works/agent-session-manager/internal/clonefidelity"
	"github.com/relux-works/agent-session-manager/internal/cloneproject"
)

// This file plans per-item expected dispositions with reason sets
// and folds planned sessions to predicted counts. The strategy arm
// decides the base mapping, the profile overlay refines it, and
// reasons compose as sorted sets. The planner never emits
// synthesized (synthesized rows have no source object) or
// unrecoverable (planned canonical items exist), and never plans
// the archive branch. Disposition and reason spellings quote the
// internal/clonefidelity owner; the agreement test pins the quote.

// Mapping is one planned item: its expected disposition and sorted
// reason set. Exact mappings carry no reason.
type Mapping struct {
	Disposition string
	Reasons     []string
}

// Disposition spellings, quoted from the internal/clonefidelity
// owner.
const (
	dispositionExact           = "exact"
	dispositionSemantic        = "semantic"
	dispositionSummarized      = "summarized"
	dispositionOpaquePreserved = "opaque_preserved"
	dispositionOmitted         = "omitted"
)

// Reason spellings, quoted from the internal/clonefidelity owner.
const (
	reasonTargetNoEquivalent   = "target_no_equivalent"
	reasonForeignEncrypted     = "foreign_encrypted_payload"
	reasonForeignSigned        = "foreign_signature_unverifiable"
	reasonOfficialImporterLoss = "official_importer_loss"
	reasonGraphFlattened       = "graph_flattened"
	reasonUnsafePending        = "unsafe_pending_action"
	reasonOperatorPolicy       = "operator_policy"
	reasonUnsupportedMedia     = "unsupported_media_type"
	reasonUnknownNative        = "unknown_native_event"
	reasonTargetContextLimit   = "target_context_limit"
)

// Profile spellings, quoted from the internal/clonefidelity owner.
const (
	profileStrictExact = "strict_exact"
	profileMaximalSafe = "maximal_safe"
	profileCompact     = "compact"
	profileMessages    = "messages_only"
)

// completeToolKinds degrade under the vendor importer: the
// importer's native tool history is lossy.
var completeToolKinds = map[string]bool{
	"tool_call":                true,
	"tool_result":              true,
	"tool_definition_snapshot": true,
}

// subagentKinds flatten under the vendor importer: the subagent
// graph does not survive official import.
var subagentKinds = map[string]bool{
	"subagent_started":   true,
	"subagent_completed": true,
}

// messageKinds survive messages_only at the event level.
var messageKinds = map[string]bool{
	"user_message":      true,
	"assistant_message": true,
}

// messageBlocks survive messages_only at the block level:
// text and structured message content plus the redaction marker,
// which preserves the redaction exactly.
var messageBlocks = map[string]bool{
	"text":     true,
	"json":     true,
	"redacted": true,
}

// mediaBlocks are message content the messages_only profile cannot
// carry: the target shows text messages, not media.
var mediaBlocks = map[string]bool{
	"image":         true,
	"audio":         true,
	"document":      true,
	"resource_link": true,
}

// maxSessionItems bounds one planned session: the Projection Plan
// 1.0.0 table carries item_mappings[1..1000000].
const maxSessionItems = 1000000

// PlanItem plans one item's expected disposition with its reason
// set under the strategy and profile. Unknown strategies,
// profiles, or item classes refuse; the archive strategy or
// profile refuses (archive carries no plan); strict_exact refuses
// every non-exact item.
func PlanItem(strategy, profile string, item Item) (Mapping, error) {
	if !validText(strategy) {
		return Mapping{}, invalid("projection strategy is not valid UTF-8")
	}
	if !clonefidelity.ValidStrategy(strategy) {
		return Mapping{}, invalid("projection strategy %q is outside the Section 13.14.2 vocabulary", strategy)
	}
	if strategy == strategyArchive || profile == profileArchive {
		return Mapping{}, invalid("projection strategy %q with profile %q names the archive branch, which carries no plan", strategy, profile)
	}
	if !validText(profile) {
		return Mapping{}, invalid("projection profile is not valid UTF-8")
	}
	if err := requireValidItemText(item); err != nil {
		return Mapping{}, err
	}
	if err := checkItem(item); err != nil {
		return Mapping{}, err
	}
	disposition, reasons := baseMapping(strategy, item)
	disposition, reasons, err := applyProfile(profile, item, disposition, reasons)
	if err != nil {
		return Mapping{}, err
	}
	sorted := append([]string(nil), reasons...)
	sort.Strings(sorted)
	mapping := Mapping{Disposition: disposition, Reasons: sorted}
	// The planner self-checks every emitted mapping through the
	// fidelity record owner: a planner/owner disagreement refuses
	// instead of emitting a row the owner would reject. The
	// whole-domain oracle plus the emitted-valid sweep prove this
	// arm never fires; PlanSession inherits it per item.
	if err := checkMappingViaOwner("projection planner emitted mapping", mapping); err != nil {
		return Mapping{}, err
	}
	return mapping, nil
}

// baseMapping decides the strategy arm: class rules first (opaque
// and aborted history are strategy-independent), then the
// continuation clamp, then the vendor importer degradations, then
// the native exact default.
func baseMapping(strategy string, item Item) (string, []string) {
	if item.Kind == "opaque_reasoning" {
		if item.Protection == protectionSigned {
			return dispositionOpaquePreserved, []string{reasonForeignSigned}
		}
		return dispositionOpaquePreserved, []string{reasonForeignEncrypted}
	}
	if item.Kind == "opaque_event" || item.Block == "opaque" {
		return dispositionOpaquePreserved, []string{reasonUnknownNative}
	}
	if (item.Kind == "tool_call" || item.Kind == "tool_result") && item.Resolution == cloneproject.StatusAborted {
		return dispositionSummarized, []string{reasonUnsafePending}
	}
	if strategy == strategyContinuation {
		return dispositionSemantic, []string{reasonTargetNoEquivalent}
	}
	if strategy == strategyOfficial {
		if completeToolKinds[item.Kind] {
			return dispositionSemantic, []string{reasonOfficialImporterLoss}
		}
		if subagentKinds[item.Kind] {
			return dispositionSemantic, []string{reasonGraphFlattened}
		}
	}
	return dispositionExact, nil
}

// applyProfile refines the strategy mapping under the profile:
// strict_exact refuses non-exact, compact clamps semantic to
// summarized, messages_only omits non-message classes (keeping and
// extending the reason set), and maximal_safe passes through.
func applyProfile(profile string, item Item, disposition string, reasons []string) (string, []string, error) {
	switch profile {
	case profileStrictExact:
		if disposition != dispositionExact {
			return "", nil, invalid("projection profile strict_exact refuses %s with disposition %q", describeItem(item), disposition)
		}
		return disposition, reasons, nil
	case profileCompact:
		if disposition == dispositionSemantic {
			return dispositionSummarized, append(append([]string(nil), reasons...), reasonTargetContextLimit), nil
		}
		return disposition, reasons, nil
	case profileMessages:
		if isMessageClass(item) {
			return disposition, reasons, nil
		}
		omitted := append(append([]string(nil), reasons...), reasonOperatorPolicy)
		if mediaBlocks[item.Block] {
			omitted = append(omitted, reasonUnsupportedMedia)
		}
		return dispositionOmitted, omitted, nil
	case profileMaximalSafe:
		return disposition, reasons, nil
	default:
		return "", nil, invalid("projection profile %q is not a target plan profile", profile)
	}
}

// isMessageClass reports whether the item survives messages_only:
// user/assistant message events and text/structured/redacted
// blocks.
func isMessageClass(item Item) bool {
	if item.Kind != "" {
		return messageKinds[item.Kind]
	}
	return messageBlocks[item.Block]
}

// describeItem names the item class for refusals.
func describeItem(item Item) string {
	if item.Kind != "" {
		return "kind " + item.Kind
	}
	return "block " + item.Block
}

// PlanSession plans every item in order and folds the predicted
// counts. Sessions are non-empty and bounded by the plan table's
// item_mappings[1..1000000]; the first refusing item refuses the
// session. Counts derive from the mappings, never accepted.
func PlanSession(strategy, profile string, items []Item) ([]Mapping, clonefidelity.FidelityCounts, error) {
	if len(items) == 0 || len(items) > maxSessionItems {
		return nil, clonefidelity.FidelityCounts{}, invalid("projection session carries %d items, the plan table requires item_mappings[1..1000000]", len(items))
	}
	if !validText(strategy) {
		return nil, clonefidelity.FidelityCounts{}, invalid("projection session strategy is not valid UTF-8")
	}
	if !validText(profile) {
		return nil, clonefidelity.FidelityCounts{}, invalid("projection session profile is not valid UTF-8")
	}
	mappings := make([]Mapping, 0, len(items))
	var counts clonefidelity.FidelityCounts
	for _, item := range items {
		mapping, err := PlanItem(strategy, profile, item)
		if err != nil {
			return nil, clonefidelity.FidelityCounts{}, err
		}
		mappings = append(mappings, mapping)
		switch mapping.Disposition {
		case dispositionExact:
			counts.Exact++
		case dispositionSemantic:
			counts.Semantic++
		case dispositionSummarized:
			counts.Summarized++
		case dispositionOpaquePreserved:
			counts.OpaquePreserved++
		case dispositionOmitted:
			counts.Omitted++
		default:
			return nil, clonefidelity.FidelityCounts{}, invalid("projection session planned unexpected disposition %q", mapping.Disposition)
		}
	}
	return mappings, counts, nil
}
