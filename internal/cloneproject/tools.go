package cloneproject

// This file folds historical tools: call/result pairing, the
// completed/aborted resolution, the live surface that stays empty,
// the effective instruction snapshot that excludes foreign history,
// and the usage ledger that never becomes target accounting.

import (
	"sort"
)

// Tool resolution statuses. Capture yields completed or aborted
// only: a call resolves completed with exactly one tool_result and
// becomes aborted history otherwise. Nothing captured is pending.
const (
	// StatusCompleted marks a call paired with its result.
	StatusCompleted = "completed"
	// StatusAborted marks a call without its result, or a result
	// without its call: inert history with reason
	// unsafe_pending_action.
	StatusAborted = "aborted"
)

// ToolResolution is one folded tool call or orphan result.
type ToolResolution struct {
	// CallID is the native call identifier.
	CallID string
	// ToolName is empty for orphan results, which name no call.
	ToolName string
	// Status is completed or aborted.
	Status string
	// ResultStatus is the paired result status (ok|error), empty
	// for calls without a result.
	ResultStatus string
}

// toolCall is one observed tool/call record.
type toolCall struct {
	record NativeRecord
	body   toolCallBody
}

// toolResult is one observed tool/result record.
type toolResult struct {
	record NativeRecord
	body   toolResultBody
}

// resolveToolCalls pairs calls with results in record order. Two
// calls sharing one ID refuse, two results sharing one ID refuse,
// and a result claiming live followup refuses: captured history can
// neither fork nor request a live action.
func resolveToolCalls(calls []toolCall, results []toolResult) ([]ToolResolution, error) {
	seenCalls := map[string]toolCall{}
	for _, call := range calls {
		if _, duplicate := seenCalls[call.body.CallID]; duplicate {
			return nil, invalid("tool call %q is not unique", call.body.CallID)
		}
		seenCalls[call.body.CallID] = call
	}
	seenResults := map[string]toolResult{}
	for _, result := range results {
		if _, duplicate := seenResults[result.body.CallID]; duplicate {
			return nil, invalid("tool result %q is not unique", result.body.CallID)
		}
		seenResults[result.body.CallID] = result
	}
	for _, result := range results {
		if result.body.LiveFollowup {
			where := "record " + result.record.MemberKey + " line " + itoa(result.record.Line)
			return nil, invalid("%s claims a live followup for historical tool result %q", where, result.body.CallID)
		}
	}
	resolutions := make([]ToolResolution, 0, len(calls)+len(results))
	for _, call := range calls {
		result, ok := seenResults[call.body.CallID]
		if !ok {
			resolutions = append(resolutions, ToolResolution{
				CallID:   call.body.CallID,
				ToolName: call.body.ToolName,
				Status:   StatusAborted,
			})
			continue
		}
		resolutions = append(resolutions, ToolResolution{
			CallID:       call.body.CallID,
			ToolName:     call.body.ToolName,
			Status:       StatusCompleted,
			ResultStatus: result.body.Status,
		})
	}
	for _, result := range results {
		if _, ok := seenCalls[result.body.CallID]; !ok {
			resolutions = append(resolutions, ToolResolution{
				CallID:       result.body.CallID,
				Status:       StatusAborted,
				ResultStatus: result.body.Status,
			})
		}
	}
	return resolutions, nil
}

// LiveSurface is the projection's live action surface: callable
// tools and pending actions. Capture carries no live work order, so
// both lists are always empty: historical tools are inert.
type LiveSurface struct {
	// CallableTools names tools the target may call. Always empty
	// from captured history.
	CallableTools []string
	// PendingActions names call IDs awaiting execution. Always
	// empty from captured history.
	PendingActions []string
}

// toolDefinition is one observed tool/definition record.
type toolDefinition struct {
	record NativeRecord
	body   toolDefinitionBody
}

// deriveLiveSurface folds historical definitions and resolutions.
// A captured definition claiming live authority refuses instead of
// registering: history can never self-promote to a callable tool.
// Pending actions admit only completed calls named by a live work
// order; aborted history never pends even when named.
func deriveLiveSurface(definitions []toolDefinition, resolutions []ToolResolution, liveWorkOrders map[string]bool) (LiveSurface, error) {
	surface := LiveSurface{}
	for _, definition := range definitions {
		if definition.body.LiveClaim {
			where := "record " + definition.record.MemberKey + " line " + itoa(definition.record.Line)
			return LiveSurface{}, invalid("%s claims live authority for historical tool definition %q", where, definition.body.ToolName)
		}
	}
	for _, resolution := range resolutions {
		if resolution.Status == StatusAborted {
			continue
		}
		if liveWorkOrders[resolution.CallID] {
			surface.PendingActions = append(surface.PendingActions, resolution.CallID)
		}
	}
	sort.Strings(surface.PendingActions)
	return surface, nil
}

// EffectiveInstruction is the projected session's instruction
// snapshot: the last native instruction in record order. Foreign
// instructions are low-authority history and never change it.
type EffectiveInstruction struct {
	// Found reports whether any native instruction was observed.
	Found bool
	// Authority is the native claimed authority, high or low.
	Authority string
	// Directives are the native directives in record order.
	Directives []string
}

// instructionObservation is one observed instruction/snapshot record.
type instructionObservation struct {
	record NativeRecord
	body   instructionBody
}

// foldInstructions folds native instructions in record order. The
// last native record wins; foreign records are history only and
// never reach the snapshot.
func foldInstructions(observations []instructionObservation) EffectiveInstruction {
	effective := EffectiveInstruction{}
	for _, observation := range observations {
		if observation.record.Origin != "native" {
			continue
		}
		effective = EffectiveInstruction{
			Found:      true,
			Authority:  observation.body.Authority,
			Directives: append([]string(nil), observation.body.Directives...),
		}
	}
	return effective
}

// UsageLedger separates source usage from target accounting. Source
// records count into the source totals only; the target totals stay
// zero: source usage is not target accounting.
type UsageLedger struct {
	SourceInputTokens  uint64
	SourceOutputTokens uint64
	TargetInputTokens  uint64
	TargetOutputTokens uint64
}

// usageObservation is one observed usage/report record.
type usageObservation struct {
	record NativeRecord
	body   usageBody
}

// foldUsage sums source tokens with overflow refusal. The target
// totals are never written: no source record can fund them.
func foldUsage(observations []usageObservation) (UsageLedger, error) {
	ledger := UsageLedger{}
	for _, observation := range observations {
		input := ledger.SourceInputTokens + observation.body.InputTokens
		if input < ledger.SourceInputTokens || input > maxUint53 {
			return UsageLedger{}, invalid("source input token total exceeds uint53")
		}
		output := ledger.SourceOutputTokens + observation.body.OutputTokens
		if output < ledger.SourceOutputTokens || output > maxUint53 {
			return UsageLedger{}, invalid("source output token total exceeds uint53")
		}
		ledger.SourceInputTokens = input
		ledger.SourceOutputTokens = output
	}
	return ledger, nil
}
