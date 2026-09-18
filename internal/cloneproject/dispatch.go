package cloneproject

// This file dispatches projection units to Canonical Event kinds,
// builds every event and the session through the landed builders,
// and re-decodes each sealed byte before it enters the result.

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// isReasoningType reports whether the native type is a reasoning
// type. Protected reasoning becomes opaque_reasoning; protected
// anything else becomes opaque_event.
func isReasoningType(nativeType string) bool {
	switch nativeType {
	case nativeReasoningSummary, nativeReasoningCrypt, nativeReasoningSigned:
		return true
	default:
		return false
	}
}

// dispatchUnits selects kind, visibility, payload, and reasons for
// every unit. Protection is decided before type: unreadable content
// never reaches a semantic kind.
func (plan *projectionPlan) dispatchUnits() error {
	resolutions := map[string]ToolResolution{}
	for _, resolution := range plan.resolutions {
		resolutions[resolution.CallID] = resolution
	}
	for index := range plan.units {
		unit := &plan.units[index]
		if unit.memberOpaque {
			unit.kind = "opaque_event"
			unit.visibility = "opaque"
			unit.payload = map[string]any{
				"native_item_key":    unit.record.MemberKey,
				"byte_count":         unit.record.Length,
				"blob_descriptor_id": unit.descriptorID,
				"extensions":         map[string]any{},
			}
			unit.reasons = []string{reasonUnknownNative}
			continue
		}
		if err := plan.dispatchRecord(unit, resolutions); err != nil {
			return err
		}
	}
	return nil
}

// dispatchRecord dispatches one parsed record.
func (plan *projectionPlan) dispatchRecord(unit *projectionUnit, resolutions map[string]ToolResolution) error {
	record := unit.record
	if record.Protection != "none" {
		return plan.dispatchProtected(unit)
	}
	switch record.NativeType {
	case nativeMessageUser, nativeMessageAssistant:
		return plan.dispatchMessage(unit)
	case nativeReasoningSummary:
		return plan.dispatchMessage(unit)
	case nativeReasoningCrypt, nativeReasoningSigned:
		where := "record " + record.MemberKey + " line " + itoa(record.Line)
		return invalid("%s declares %s with no protection", where, record.NativeType)
	case nativeInstruction:
		return plan.dispatchInstruction(unit)
	case nativeToolDefinition:
		return plan.dispatchDefinition(unit)
	case nativeToolCall:
		return plan.dispatchCall(unit, resolutions)
	case nativeToolResult:
		return plan.dispatchResult(unit, resolutions)
	case nativeUsage:
		return plan.dispatchUsage(unit)
	default:
		unit.kind = "opaque_event"
		unit.visibility = "opaque"
		unit.payload = map[string]any{
			"native_type":        record.NativeType,
			"byte_count":         record.Length,
			"blob_descriptor_id": unit.descriptorID,
			"extensions":         map[string]any{},
		}
		unit.reasons = []string{reasonUnknownNative}
		return nil
	}
}

// dispatchProtected dispatches a protected record: protection
// defines the body shape (exactly ciphertext, never interpreted),
// and the kind is opaque. Reasoning stays opaque_reasoning;
// anything else becomes opaque_event.
func (plan *projectionPlan) dispatchProtected(unit *projectionUnit) error {
	record := unit.record
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	if _, err := decodeProtectedBody(record); err != nil {
		return err
	}
	switch record.NativeType {
	case nativeReasoningCrypt:
		if record.Protection != "encrypted" {
			return invalid("%s declares reasoning/encrypted with protection %q", where, record.Protection)
		}
	case nativeReasoningSigned:
		if record.Protection != "signed" {
			return invalid("%s declares reasoning/signed with protection %q", where, record.Protection)
		}
	}
	reasons := cryptoReasons(record)
	if !knownNativeType(record.NativeType) {
		reasons = append(reasons, reasonUnknownNative)
		sort.Strings(reasons)
	}
	if isReasoningType(record.NativeType) {
		unit.kind = "opaque_reasoning"
		unit.visibility = "opaque"
		unit.payload = map[string]any{
			"protection":         record.Protection,
			"byte_count":         record.Length,
			"blob_descriptor_id": unit.descriptorID,
			"extensions":         map[string]any{},
		}
		unit.reasons = reasons
		return nil
	}
	unit.kind = "opaque_event"
	unit.visibility = "opaque"
	unit.payload = map[string]any{
		"native_type":        record.NativeType,
		"protection":         record.Protection,
		"byte_count":         record.Length,
		"blob_descriptor_id": unit.descriptorID,
		"extensions":         map[string]any{},
	}
	unit.reasons = reasons
	return nil
}

// cryptoReasons selects the stable reason for a protected record:
// the foreign core reason when the origin is foreign, no reason
// when native (no core code fits native-protected content; the
// payload protection fact carries the cause).
func cryptoReasons(record NativeRecord) []string {
	if record.Origin != "foreign" {
		return nil
	}
	if record.Protection == "signed" {
		return []string{reasonForeignSigned}
	}
	return []string{reasonForeignEncrypted}
}

// dispatchMessage dispatches a message or reasoning summary: the
// exact text body becomes one text content block, inline or
// overflow-referenced.
func (plan *projectionPlan) dispatchMessage(unit *projectionUnit) error {
	record := unit.record
	body, err := decodeMessageBody(record)
	if err != nil {
		return err
	}
	block, installed, err := contentBlockForText(body.Text, plan.request.CanonicalSink)
	if err != nil {
		return err
	}
	plan.overflows = append(plan.overflows, installed...)
	switch record.NativeType {
	case nativeMessageUser:
		unit.kind = "user_message"
		unit.visibility = "public"
	case nativeMessageAssistant:
		unit.kind = "assistant_message"
		unit.visibility = "public"
	default:
		unit.kind = "reasoning_summary"
		unit.visibility = "internal"
	}
	unit.payload = map[string]any{
		"content_blocks": []any{block},
		"extensions":     map[string]any{},
	}
	return nil
}

// dispatchInstruction dispatches an instruction snapshot: foreign
// instructions pin low authority, native instructions carry their
// claimed authority.
func (plan *projectionPlan) dispatchInstruction(unit *projectionUnit) error {
	record := unit.record
	body := unit.instruction
	authority := body.Authority
	if record.Origin != "native" {
		authority = "low"
	}
	directives := make([]any, 0, len(body.Directives))
	for _, directive := range body.Directives {
		directives = append(directives, directive)
	}
	unit.kind = "instruction_snapshot"
	unit.visibility = "internal"
	unit.payload = map[string]any{
		"authority":  authority,
		"directives": directives,
		"extensions": map[string]any{},
	}
	return nil
}

// dispatchDefinition dispatches a tool definition snapshot: history
// with callable pinned false. Registration never happens here; the
// live surface refused live claims before dispatch ran.
func (plan *projectionPlan) dispatchDefinition(unit *projectionUnit) error {
	body := unit.definition
	unit.kind = "tool_definition_snapshot"
	unit.visibility = "internal"
	unit.payload = map[string]any{
		"tool_name":         body.ToolName,
		"definition_digest": body.DefinitionHash,
		"callable":          false,
		"extensions":        map[string]any{},
	}
	return nil
}

// dispatchCall dispatches a tool call with its folded resolution.
func (plan *projectionPlan) dispatchCall(unit *projectionUnit, resolutions map[string]ToolResolution) error {
	body := unit.call
	resolution, ok := resolutions[body.CallID]
	if !ok {
		return invalid("tool call %q has no folded resolution", body.CallID)
	}
	unit.kind = "tool_call"
	unit.visibility = "internal"
	unit.payload = map[string]any{
		"call_id":    body.CallID,
		"tool_name":  body.ToolName,
		"resolution": resolution.Status,
		"extensions": map[string]any{},
	}
	if resolution.Status == StatusAborted {
		unit.reasons = []string{reasonUnsafePending}
	}
	return nil
}

// dispatchResult dispatches a tool result with its folded
// resolution: completed when paired, aborted history when orphaned.
func (plan *projectionPlan) dispatchResult(unit *projectionUnit, resolutions map[string]ToolResolution) error {
	body := unit.result
	resolution, ok := resolutions[body.CallID]
	if !ok {
		return invalid("tool result %q has no folded resolution", body.CallID)
	}
	unit.kind = "tool_result"
	unit.visibility = "internal"
	unit.payload = map[string]any{
		"call_id":    body.CallID,
		"status":     body.Status,
		"resolution": resolution.Status,
		"extensions": map[string]any{},
	}
	if resolution.Status == StatusAborted {
		unit.reasons = []string{reasonUnsafePending}
	}
	return nil
}

// dispatchUsage dispatches a usage report: source tokens into the
// payload; target accounting is never written anywhere.
func (plan *projectionPlan) dispatchUsage(unit *projectionUnit) error {
	body := unit.usage
	unit.kind = "usage"
	unit.visibility = "internal"
	unit.payload = map[string]any{
		"input_tokens":  body.InputTokens,
		"output_tokens": body.OutputTokens,
		"extensions":    map[string]any{},
	}
	return nil
}

// actorSet is the resolved actor UUIDs by selector.
type actorSet struct {
	main  scalar.UUIDv7
	extra map[string]scalar.UUIDv7
}

// checkActors resolves every referenced selector: main plus exactly
// the mapped extras. A referenced selector without a mapping, or a
// mapping without a reference, refuses.
func (plan *projectionPlan) checkActors() error {
	referenced := map[string]bool{}
	for _, unit := range plan.units {
		referenced[unit.record.Actor] = true
	}
	for selector := range referenced {
		if selector == "main" {
			continue
		}
		if _, ok := plan.extraActors[selector]; !ok {
			return invalid("projection references actor %q with no mapped UUID", selector)
		}
	}
	for selector := range plan.extraActors {
		if !referenced[selector] {
			return invalid("projection maps actor %q with no referencing record", selector)
		}
	}
	return nil
}

// buildEvents dispatches every unit and seals the events in record
// order: contiguous ordinals from zero, each event parented on its
// predecessor, each sealed byte re-decoded before use.
func (plan *projectionPlan) buildEvents() error {
	if err := plan.dispatchUnits(); err != nil {
		return err
	}
	actors := actorSet{main: plan.mainActor, extra: plan.extraActors}
	var previous *scalar.Digest
	for index := range plan.units {
		unit := &plan.units[index]
		actor, ok := actors.extra[unit.record.Actor]
		if unit.record.Actor == "main" {
			actor, ok = actors.main, true
		}
		if !ok {
			return invalid("projection cannot resolve actor %q to a mapped UUID", unit.record.Actor)
		}
		parents := []string{}
		if previous != nil {
			parents = []string{previous.String()}
		}
		payload, err := json.Marshal(unit.payload)
		if err != nil {
			return invalid("projection cannot marshal event payload: %v", err)
		}
		input := clonebundle.CanonicalEventInput{
			LogicalSessionID: plan.logical.String(),
			Ordinal:          uint64(index),
			Parents:          parents,
			ActorID:          actor.String(),
			Kind:             unit.kind,
			Visibility:       unit.visibility,
			Payload:          payload,
			Evidence:         plan.evidenceFor(unit),
			Extensions:       map[string]any{},
		}
		sealed, err := clonebundle.BuildCanonicalEvent(input)
		if err != nil {
			return invalid("projection cannot seal ordinal %d (%s): %v", index, unit.kind, err)
		}
		decoded, err := clonebundle.DecodeCanonicalEvent(sealed)
		if err != nil {
			return invalid("sealed ordinal %d refused: %v", index, err)
		}
		plan.events = append(plan.events, sealed)
		plan.decodedEvents = append(plan.decodedEvents, decoded)
		plan.ordinals = append(plan.ordinals, uint64(index))
		identifier := decoded.EventID
		previous = &identifier
	}
	if err := clonebundle.CheckOrdinalContiguity(plan.ordinals); err != nil {
		return err
	}
	return nil
}

// evidenceFor builds the Source Evidence input for one unit: exact
// capture status with the record's byte range inside its member
// blob, so every reference resolves back to captured bytes.
func (plan *projectionPlan) evidenceFor(unit *projectionUnit) clonebundle.EvidenceInput {
	record := unit.record
	input := clonebundle.EvidenceInput{
		Environment:     plan.environment,
		NativeSessionID: plan.raw.SourceNativeSessionID,
		RawRefs: []clonebundle.RawRefInput{{
			ManifestID:       plan.raw.ManifestID.String(),
			BlobDescriptorID: unit.descriptorID,
			Offset:           record.Offset,
			Length:           record.Length,
		}},
		CaptureStatus: "exact",
		ReasonCodes:   append([]string(nil), unit.reasons...),
	}
	if !unit.memberOpaque {
		eventID := record.NativeEventID
		nativeType := record.NativeType
		input.NativeEventID = &eventID
		input.NativeType = &nativeType
	}
	return input
}

// buildResult seals the Canonical Session over the projected
// events: main plus exactly the referenced extras (parented on
// main), ordinal-ordered event IDs, the last event as head.
func (plan *projectionPlan) buildResult() (*NormalizeResult, error) {
	main := plan.mainActor.String()
	inputs := []clonebundle.ActorInput{{
		ActorID: main,
		Kind:    "main",
	}}
	selectors := make([]string, 0, len(plan.extraActors))
	for selector := range plan.extraActors {
		selectors = append(selectors, selector)
	}
	sort.Strings(selectors)
	for _, selector := range selectors {
		name := selector
		kind := "external"
		if strings.HasPrefix(selector, "subagent:") {
			kind = "subagent"
		}
		parent := main
		inputs = append(inputs, clonebundle.ActorInput{
			ActorID:        plan.extraActors[selector].String(),
			Kind:           kind,
			ParentActorID:  &parent,
			Name:           &name,
			SourceNativeID: &name,
		})
	}
	eventIDs := make([]string, 0, len(plan.decodedEvents))
	for _, decoded := range plan.decodedEvents {
		eventIDs = append(eventIDs, decoded.EventID.String())
	}
	// Unreachable: Normalize refused the empty projection before
	// dispatch ran. The guard keeps a headless build a builder
	// refusal instead of an index panic.
	heads := []string{}
	if len(eventIDs) > 0 {
		heads = []string{eventIDs[len(eventIDs)-1]}
	}
	session, err := clonebundle.BuildCanonicalSession(clonebundle.CanonicalSessionInput{
		LogicalSessionID:      plan.logical.String(),
		SourceEnvironment:     plan.environment,
		SourceNativeSessionID: plan.raw.SourceNativeSessionID,
		Title:                 plan.request.Title,
		Workspace:             plan.request.Workspace,
		Actors:                inputs,
		EventIDs:              eventIDs,
		HeadEventIDs:          heads,
		Extensions:            map[string]any{},
	})
	if err != nil {
		return nil, invalid("projection cannot seal its session: %v", err)
	}
	decoded, err := clonebundle.DecodeCanonicalSession(session)
	if err != nil {
		return nil, invalid("sealed session refused: %v", err)
	}
	return &NormalizeResult{
		Session:         session,
		DecodedSession:  decoded,
		Events:          plan.events,
		DecodedEvents:   plan.decodedEvents,
		ToolResolutions: plan.resolutions,
		Live:            plan.surface,
		Instruction:     plan.instruction,
		Usage:           plan.usage,
		Overflows:       plan.overflows,
	}, nil
}
