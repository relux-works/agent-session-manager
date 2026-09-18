package cloneproject

// This file owns the Normalize production entry: manifest closure,
// blob verification, native dispatch, tool/instruction/usage folds,
// event and session construction through the landed builders, and
// the re-decode of every sealed byte before return.

import (
	"bytes"
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Stable reason codes this projection emits. They are the closed
// core reasons Section 13.14.2 registers for exactly these causes,
// reused here as Source Evidence reason codes.
const (
	// reasonUnknownNative marks records whose native type or member
	// class the projection does not register.
	reasonUnknownNative = "unknown_native_event"
	// reasonForeignEncrypted marks foreign encrypted records
	// preserved opaque.
	reasonForeignEncrypted = "foreign_encrypted_payload"
	// reasonForeignSigned marks foreign signed records preserved
	// opaque.
	reasonForeignSigned = "foreign_signature_unverifiable"
	// reasonUnsafePending marks incomplete tool history that blocks
	// pending action.
	reasonUnsafePending = "unsafe_pending_action"
)

// NormalizeRequest is one projection candidate over captured bytes.
type NormalizeRequest struct {
	// CaptureManifest and RawManifest carry the sealed manifests
	// clonesnap.Capture produced.
	CaptureManifest []byte
	RawManifest     []byte
	// Fetch reads one installed blob by content digest. Tests wire
	// it to the real capture store; every fetched byte is
	// digest- and size-verified before use.
	Fetch func(blobID string) ([]byte, error)
	// CanonicalSink receives overflow content blobs through the
	// landed no-replace discipline.
	CanonicalSink *localstore.ObjectStore
	// LogicalSessionID and MainActorID identify the projected
	// session and its main actor.
	LogicalSessionID string
	MainActorID      string
	// Actors maps extra actor selectors
	// (external, subagent:<name>) to actor UUIDv7. Every mapped
	// selector must be referenced, and every referenced selector
	// must be mapped.
	Actors map[string]string
	// Title stamps the session title; nil seals a null title.
	Title *string
	// Workspace carries the sealed Workspace Binding bytes (the
	// capture's workspace checkpoint, passed through).
	Workspace []byte
}

// NormalizeResult is one projected bundle: sealed bytes, their
// decoded forms, and the derived folds.
type NormalizeResult struct {
	// Session carries the sealed Canonical Session bytes.
	Session []byte
	// DecodedSession is the re-decoded session.
	DecodedSession clonebundle.CanonicalSession
	// Events carries the sealed Canonical Event bytes in ordinal
	// order.
	Events [][]byte
	// DecodedEvents are the re-decoded events in ordinal order.
	DecodedEvents []clonebundle.CanonicalEvent
	// ToolResolutions carries the completed/aborted fold in call
	// order with orphan results after calls.
	ToolResolutions []ToolResolution
	// Live is the live surface: always empty from capture.
	Live LiveSurface
	// Instruction is the effective instruction snapshot: native
	// history only.
	Instruction EffectiveInstruction
	// Usage separates source usage from zero target accounting.
	Usage UsageLedger
	// Overflows names every installed overflow blob in event
	// order, so content references resolve.
	Overflows []OverflowBlob
}

// Normalize derives a Canonical Session and Canonical Events from
// captured bytes. The unstable_archive form cannot enter: target
// projection refuses it through the landed G2 gate. Every sealed
// event and the session re-decode through the landed decoders before
// return; a failure anywhere returns no partial bundle.
func Normalize(request NormalizeRequest) (*NormalizeResult, error) {
	if request.Fetch == nil {
		return nil, invalid("projection requires a blob fetcher")
	}
	if request.CanonicalSink == nil {
		return nil, invalid("projection requires a canonical sink")
	}
	if len(request.Workspace) == 0 {
		return nil, invalid("projection requires workspace binding bytes")
	}
	logical, err := scalar.ParseUUIDv7(request.LogicalSessionID)
	if err != nil {
		return nil, invalid("projection logical_session_id is not a UUIDv7: %v", err)
	}
	mainActor, err := scalar.ParseUUIDv7(request.MainActorID)
	if err != nil {
		return nil, invalid("projection main actor is not a UUIDv7: %v", err)
	}
	extraActors := map[string]scalar.UUIDv7{}
	for selector, value := range request.Actors {
		if err := checkActorSelector(selector); err != nil {
			return nil, err
		}
		if selector == "main" {
			return nil, invalid("projection actor map carries main, which is request.MainActorID")
		}
		parsed, err := scalar.ParseUUIDv7(value)
		if err != nil {
			return nil, invalid("projection actor %q is not a UUIDv7: %v", selector, err)
		}
		extraActors[selector] = parsed
	}
	capture, err := clonebundle.DecodeCaptureManifest(request.CaptureManifest)
	if err != nil {
		return nil, err
	}
	if err := clonebundle.RefuseUnstableForTarget(capture.CaptureBoundary); err != nil {
		return nil, err
	}
	raw, err := clonebundle.DecodeRawObjectManifest(request.RawManifest)
	if err != nil {
		return nil, err
	}
	if raw.ManifestID.String() != capture.SourceRawObjectManifest.String() {
		return nil, invalid("capture manifest links raw manifest %q, sealed raw manifest is %q",
			capture.SourceRawObjectManifest.String(), raw.ManifestID.String())
	}
	environment, err := sealedEnvironment(request.RawManifest, request.CaptureManifest)
	if err != nil {
		return nil, err
	}
	entries := map[string]clonebundle.RawObjectEntry{}
	for _, entry := range raw.Entries {
		entries[entry.NativeItemKey] = entry
	}
	plan := &projectionPlan{
		request:     request,
		raw:         raw,
		entries:     entries,
		environment: environment,
		logical:     logical,
		mainActor:   mainActor,
		extraActors: extraActors,
	}
	if err := plan.collectMembers(capture); err != nil {
		return nil, err
	}
	if len(plan.units) == 0 {
		return nil, invalid("projection holds no events")
	}
	if err := plan.foldTools(); err != nil {
		return nil, err
	}
	if err := plan.foldInstructions(); err != nil {
		return nil, err
	}
	if err := plan.foldUsage(); err != nil {
		return nil, err
	}
	surface, err := deriveLiveSurface(plan.definitions, plan.resolutions, nil)
	if err != nil {
		return nil, err
	}
	plan.surface = surface
	if err := plan.checkActors(); err != nil {
		return nil, err
	}
	if err := plan.buildEvents(); err != nil {
		return nil, err
	}
	return plan.buildResult()
}

// sealedEnvironment extracts the exact source_environment member
// from both sealed manifests and refuses closure drift: the two
// sealed tuples must agree byte-for-byte after canonicalization, so
// the evidence environment is unambiguous.
func sealedEnvironment(rawManifest, captureManifest []byte) ([]byte, error) {
	raw, err := sealedMember(rawManifest, "source_environment")
	if err != nil {
		return nil, err
	}
	captured, err := sealedMember(captureManifest, "source_environment")
	if err != nil {
		return nil, err
	}
	rawCanonical, err := canonicaljson.Canonicalize(raw)
	if err != nil {
		return nil, invalid("raw manifest source_environment is not canonical JSON: %v", err)
	}
	capturedCanonical, err := canonicaljson.Canonicalize(captured)
	if err != nil {
		return nil, invalid("capture manifest source_environment is not canonical JSON: %v", err)
	}
	if !bytes.Equal(rawCanonical, capturedCanonical) {
		return nil, invalid("capture closure drifts: raw and capture source_environment disagree")
	}
	return rawCanonical, nil
}

// sealedMember extracts one raw member from sealed bytes. Both
// manifests decoded clean before this runs, so the bytes are
// canonical and duplicate-free.
func sealedMember(sealed []byte, name string) ([]byte, error) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(sealed, &members); err != nil {
		return nil, invalid("sealed manifest is not a JSON object: %v", err)
	}
	raw, ok := members[name]
	if !ok {
		return nil, invalid("sealed manifest misses member %q", name)
	}
	return raw, nil
}

// projectionUnit is one projectable unit: a parsed record or a
// whole unknown-class member.
type projectionUnit struct {
	record NativeRecord
	// memberOpaque marks a whole-member opaque_event: Offset is 0
	// and Length is the member byte count.
	memberOpaque bool
	// descriptorID is the raw entry's descriptor; blob bytes are
	// verified before the unit is admitted.
	descriptorID string
	// definition, call, result, instruction, and usage cache the
	// decoded bodies the folds validated, so dispatch never
	// re-decodes and never swallows a decode failure.
	definition  *toolDefinitionBody
	call        *toolCallBody
	result      *toolResultBody
	instruction *instructionBody
	usage       *usageBody
	// kind, visibility, payload, and reasons are the dispatch.
	kind       string
	visibility string
	payload    map[string]any
	reasons    []string
}

// projectionPlan accumulates one normalization.
type projectionPlan struct {
	request     NormalizeRequest
	raw         clonebundle.RawObjectManifest
	entries     map[string]clonebundle.RawObjectEntry
	environment []byte
	logical     scalar.UUIDv7
	mainActor   scalar.UUIDv7
	extraActors map[string]scalar.UUIDv7

	units        []projectionUnit
	calls        []toolCall
	results      []toolResult
	definitions  []toolDefinition
	instructions []instructionObservation
	usages       []usageObservation

	resolutions []ToolResolution
	surface     LiveSurface
	instruction EffectiveInstruction
	usage       UsageLedger

	events        [][]byte
	decodedEvents []clonebundle.CanonicalEvent
	ordinals      []uint64
	overflows     []OverflowBlob
}

// collectMembers verifies every included member's blob and parses
// its records in capture order.
func (plan *projectionPlan) collectMembers(capture clonebundle.CaptureManifest) error {
	for _, item := range capture.Items {
		if !item.Included() {
			continue
		}
		entry, ok := plan.entries[item.NativeItemKey]
		if !ok {
			return invalid("capture item %q is included with no raw object entry", item.NativeItemKey)
		}
		payload, err := plan.request.Fetch(entry.BlobID.String())
		if err != nil {
			return invalid("projection cannot fetch blob %q for member %q: %v", entry.BlobID.String(), item.NativeItemKey, err)
		}
		if uint64(len(payload)) != entry.ByteCount {
			return invalid("member %q blob size %d disagrees with raw count %d", item.NativeItemKey, len(payload), entry.ByteCount)
		}
		if scalar.SHA256Digest(payload).String() != entry.BlobID.String() {
			return invalid("member %q blob digest disagrees with raw entry", item.NativeItemKey)
		}
		if item.Class == "unknown" {
			plan.units = append(plan.units, projectionUnit{
				record: NativeRecord{
					MemberKey:     item.NativeItemKey,
					Offset:        0,
					Length:        entry.ByteCount,
					NativeEventID: "",
					NativeType:    "",
					Origin:        "native",
					Protection:    "none",
					Actor:         "main",
					LineBytes:     append([]byte(nil), payload...),
				},
				memberOpaque: true,
				descriptorID: entry.BlobDescriptorID.String(),
			})
			continue
		}
		lines, err := splitRecordLines(item.NativeItemKey, payload)
		if err != nil {
			return err
		}
		for _, line := range lines {
			record, err := parseNativeLine(item.NativeItemKey, line.number, line.offset, line.raw)
			if err != nil {
				return err
			}
			plan.units = append(plan.units, projectionUnit{
				record:       record,
				descriptorID: entry.BlobDescriptorID.String(),
			})
		}
	}
	return nil
}

// foldTools collects tool observations in record order and resolves
// the completed/aborted fold.
func (plan *projectionPlan) foldTools() error {
	for index := range plan.units {
		unit := &plan.units[index]
		if unit.memberOpaque {
			continue
		}
		record := unit.record
		if record.Protection != "none" && !isReasoningType(record.NativeType) {
			continue
		}
		switch record.NativeType {
		case nativeToolDefinition:
			body, err := decodeToolDefinitionBody(record)
			if err != nil {
				return err
			}
			if _, err := scalar.ParseDigest(body.DefinitionHash); err != nil {
				return invalid("record %s line %s body definition_digest is not a digest: %v",
					record.MemberKey, itoa(record.Line), err)
			}
			unit.definition = &body
			plan.definitions = append(plan.definitions, toolDefinition{record: record, body: body})
		case nativeToolCall:
			body, err := decodeToolCallBody(record)
			if err != nil {
				return err
			}
			unit.call = &body
			plan.calls = append(plan.calls, toolCall{record: record, body: body})
		case nativeToolResult:
			body, err := decodeToolResultBody(record)
			if err != nil {
				return err
			}
			unit.result = &body
			plan.results = append(plan.results, toolResult{record: record, body: body})
		}
	}
	resolutions, err := resolveToolCalls(plan.calls, plan.results)
	if err != nil {
		return err
	}
	plan.resolutions = resolutions
	return nil
}

// foldInstructions collects instruction observations in record
// order and folds the effective snapshot.
func (plan *projectionPlan) foldInstructions() error {
	for index := range plan.units {
		unit := &plan.units[index]
		if unit.memberOpaque {
			continue
		}
		record := unit.record
		if record.NativeType != nativeInstruction || record.Protection != "none" {
			continue
		}
		body, err := decodeInstructionBody(record)
		if err != nil {
			return err
		}
		unit.instruction = &body
		plan.instructions = append(plan.instructions, instructionObservation{record: record, body: body})
	}
	plan.instruction = foldInstructions(plan.instructions)
	return nil
}

// foldUsage collects usage observations in record order and folds
// the source ledger.
func (plan *projectionPlan) foldUsage() error {
	for index := range plan.units {
		unit := &plan.units[index]
		if unit.memberOpaque {
			continue
		}
		record := unit.record
		if record.NativeType != nativeUsage || record.Protection != "none" {
			continue
		}
		body, err := decodeUsageBody(record)
		if err != nil {
			return err
		}
		unit.usage = &body
		plan.usages = append(plan.usages, usageObservation{record: record, body: body})
	}
	ledger, err := foldUsage(plan.usages)
	if err != nil {
		return err
	}
	plan.usage = ledger
	return nil
}
