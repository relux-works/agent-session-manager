package clonebundle

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file validates and constructs Canonical Event 1.0.0 with its
// Source Evidence and raw byte-range references. The envelope,
// vocabularies, evidence coupling, and message-like content-block
// rules are enforced here; the full per-kind payload fact registry
// is sibling (normalization) scope and is stated in doc.go.

const (
	canonicalEventSchema  = "urn:ax:schema:canonical-event"
	canonicalEventVersion = "1.0.0"
	canonicalEventSelf    = "event_id"
	maxInlineContent      = 65536
)

// eventKinds is the exact 26-kind event vocabulary Section 13.14.1
// states.
var eventKinds = []string{
	"session_started",
	"instruction_snapshot",
	"user_message",
	"assistant_message",
	"reasoning_summary",
	"opaque_reasoning",
	"tool_definition_snapshot",
	"tool_call",
	"tool_result",
	"approval_request",
	"approval_response",
	"plan_update",
	"progress",
	"usage",
	"rate_limit",
	"file_change",
	"compaction",
	"turn_started",
	"turn_completed",
	"turn_aborted",
	"subagent_started",
	"subagent_completed",
	"error",
	"migration_checkpoint",
	"session_finished",
	"opaque_event",
}

// ValidEventKind reports whether the name is a canonical event kind.
func ValidEventKind(kind string) bool {
	for _, allowed := range eventKinds {
		if kind == allowed {
			return true
		}
	}
	return false
}

// EventKinds returns the exact 26-kind event vocabulary Section
// 13.14.1 states, in specification order. Sibling packages iterate
// the returned copy so closed maps over every kind keep one owner;
// callers must not retain and mutate across calls.
func EventKinds() []string {
	kinds := make([]string, len(eventKinds))
	copy(kinds, eventKinds)
	return kinds
}

func validVisibility(visibility string) bool {
	switch visibility {
	case "public", "projection", "internal", "opaque":
		return true
	default:
		return false
	}
}

func validCaptureStatus(status string) bool {
	switch status {
	case "exact", "partial", "synthesized", "unavailable":
		return true
	default:
		return false
	}
}

// contentBlockTypes is the exact message-like content-block
// vocabulary: text|json|image|audio|document|resource_link|redacted|opaque.
var contentBlockTypes = []string{
	"text",
	"json",
	"image",
	"audio",
	"document",
	"resource_link",
	"redacted",
	"opaque",
}

func validContentBlockType(blockType string) bool {
	for _, allowed := range contentBlockTypes {
		if blockType == allowed {
			return true
		}
	}
	return false
}

// ValidContentBlockType reports whether the name is a message-like
// content-block type. It delegates to the single Section 13.14.1
// table above so sibling packages share one vocabulary owner.
func ValidContentBlockType(blockType string) bool {
	return validContentBlockType(blockType)
}

// ContentBlockTypes returns the exact message-like content-block
// vocabulary Section 13.14.1 states, in specification order.
// Sibling packages iterate the returned copy so closed maps over
// every type keep one owner; callers must not retain and mutate
// across calls.
func ContentBlockTypes() []string {
	types := make([]string, len(contentBlockTypes))
	copy(types, contentBlockTypes)
	return types
}

// messageLikeKind reports whether the kind carries a message-like
// payload: user and assistant messages always do. Other kinds may
// carry content blocks under the same block rules, but only these
// two require them.
func messageLikeKind(kind string) bool {
	return kind == "user_message" || kind == "assistant_message"
}

var rawReferenceMembers = map[string]bool{
	"manifest_id": true, "blob_descriptor_id": true, "offset": true, "length": true,
}

var rawReferenceRequired = []string{
	"manifest_id", "blob_descriptor_id", "offset", "length",
}

// RawReference is one validated raw byte-range reference.
type RawReference struct {
	ManifestID       scalar.Digest
	BlobDescriptorID scalar.Digest
	Offset           uint64
	Length           uint64
}

func decodeRawReference(raw json.RawMessage, index int) (RawReference, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return RawReference{}, invalid("raw reference[%d] %s (%s)", index, fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, rawReferenceMembers); unknown {
		return RawReference{}, invalid("raw reference[%d] carries unknown member %q", index, name)
	}
	if name, missing := missingMember(members, rawReferenceRequired); missing {
		return RawReference{}, invalid("raw reference[%d] misses a required member %q", index, name)
	}
	manifestID, ok := checkDigest(members["manifest_id"])
	if !ok {
		return RawReference{}, invalid("raw reference[%d] manifest_id is not a digest", index)
	}
	descriptorID, ok := checkDigest(members["blob_descriptor_id"])
	if !ok {
		return RawReference{}, invalid("raw reference[%d] blob_descriptor_id is not a digest", index)
	}
	offset, ok := checkUint53Bounds(members["offset"], 0, maxUint53)
	if !ok {
		return RawReference{}, invalid("raw reference[%d] offset is not a uint53", index)
	}
	length, ok := checkUint53Bounds(members["length"], 0, maxUint53)
	if !ok {
		return RawReference{}, invalid("raw reference[%d] length is not a uint53", index)
	}
	return RawReference{
		ManifestID:       manifestID,
		BlobDescriptorID: descriptorID,
		Offset:           offset,
		Length:           length,
	}, nil
}

func rawReferenceObject(reference RawReference) map[string]any {
	return map[string]any{
		"manifest_id":        reference.ManifestID.String(),
		"blob_descriptor_id": reference.BlobDescriptorID.String(),
		"offset":             reference.Offset,
		"length":             reference.Length,
	}
}

// decodeSortedUniqueRawReferences validates sorted unique raw
// references: bytewise order of each item's JCS encoding, strict
// increase proving sortedness and uniqueness together.
func decodeSortedUniqueRawReferences(raw json.RawMessage) ([]RawReference, error) {
	elements, ok := decodeArray(raw)
	if !ok {
		return nil, invalid("source evidence raw_refs are not an array")
	}
	if len(elements) > 65536 {
		return nil, invalid("source evidence carries %d raw_refs, maximum is 65536", len(elements))
	}
	references := make([]RawReference, 0, len(elements))
	var previous []byte
	for index, element := range elements {
		reference, err := decodeRawReference(element, index)
		if err != nil {
			return nil, err
		}
		canonical, err := canonicaljson.Canonicalize(bytesTrimSpace(element))
		if err != nil {
			return nil, invalid("source evidence raw_refs[%d] are not canonical JSON: %v", index, err)
		}
		if index > 0 && bytes.Compare(canonical, previous) <= 0 {
			return nil, invalid("source evidence raw_refs are not sorted unique")
		}
		previous = canonical
		references = append(references, reference)
	}
	return references, nil
}

var sourceEvidenceMembers = map[string]bool{
	"environment": true, "native_session_id": true, "native_event_id": true,
	"native_type": true, "raw_refs": true, "capture_status": true,
	"reason_codes": true, "core_operation_id": true, "extensions": true,
}

var sourceEvidenceRequired = []string{
	"environment", "native_session_id", "native_event_id",
	"native_type", "raw_refs", "capture_status",
	"reason_codes", "core_operation_id", "extensions",
}

// SourceEvidence is one validated source evidence record.
type SourceEvidence struct {
	Environment     sessadapter.Tuple
	NativeSessionID string
	NativeEventID   *string
	NativeType      *string
	RawRefs         []RawReference
	CaptureStatus   string
	ReasonCodes     []string
	CoreOperationID *scalar.UUIDv7
}

// DecodeSourceEvidence validates one closed SourceEvidence:
// sanitized native IDs, sorted unique raw references, and the
// synthesized coupling (null native Event ID with a non-null core
// operation; every other status with a null core operation).
func DecodeSourceEvidence(raw json.RawMessage) (SourceEvidence, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return SourceEvidence{}, invalid("source evidence %s (%s)", fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, sourceEvidenceMembers); unknown {
		return SourceEvidence{}, invalid("source evidence carries unknown member %q", name)
	}
	if name, missing := missingMember(members, sourceEvidenceRequired); missing {
		return SourceEvidence{}, invalid("source evidence misses a required member %q", name)
	}
	tuple, err := sessadapter.DecodeTuple(members["environment"])
	if err != nil {
		return SourceEvidence{}, invalid("source evidence environment is not an Environment Tuple: %v", err)
	}
	nativeSession, ok := checkStringBounds(members["native_session_id"], 1, 512)
	if !ok {
		return SourceEvidence{}, invalid("source evidence native_session_id is not a string[1..512]")
	}
	if err := SanitizeNativeKey(nativeSession); err != nil {
		return SourceEvidence{}, err
	}
	var nativeEvent *string
	if !isNull(members["native_event_id"]) {
		value, ok := checkStringBounds(members["native_event_id"], 1, 512)
		if !ok {
			return SourceEvidence{}, invalid("source evidence native_event_id is not a string[1..512]")
		}
		if err := SanitizeNativeKey(value); err != nil {
			return SourceEvidence{}, err
		}
		nativeEvent = &value
	}
	var nativeType *string
	if !isNull(members["native_type"]) {
		value, ok := checkStringBounds(members["native_type"], 1, 512)
		if !ok {
			return SourceEvidence{}, invalid("source evidence native_type is not a string[1..512]")
		}
		nativeType = &value
	}
	references, err := decodeSortedUniqueRawReferences(members["raw_refs"])
	if err != nil {
		return SourceEvidence{}, err
	}
	status, ok := rawString(members["capture_status"])
	if !ok || !validCaptureStatus(status) {
		return SourceEvidence{}, invalid("source evidence capture_status is outside exact|partial|synthesized|unavailable")
	}
	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 128, 0, 128)
	if !ok {
		return SourceEvidence{}, invalid("source evidence reason_codes are not sorted unique string[1..128][0..128]")
	}
	var operation *scalar.UUIDv7
	if !isNull(members["core_operation_id"]) {
		value, ok := checkUUIDv7(members["core_operation_id"])
		if !ok {
			return SourceEvidence{}, invalid("source evidence core_operation_id is not a UUIDv7")
		}
		operation = &value
	}
	if err := checkEvidenceCoupling(status, nativeEvent != nil, operation != nil); err != nil {
		return SourceEvidence{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return SourceEvidence{}, invalid("source evidence extensions %s", extensionsFault)
	}
	return SourceEvidence{
		Environment:     tuple,
		NativeSessionID: nativeSession,
		NativeEventID:   nativeEvent,
		NativeType:      nativeType,
		RawRefs:         references,
		CaptureStatus:   status,
		ReasonCodes:     reasons,
		CoreOperationID: operation,
	}, nil
}

// checkEvidenceCoupling enforces the synthesized rule: synthesized
// requires a null native Event ID and a non-null core operation
// identifying the core operation; all other statuses require a
// null core operation.
func checkEvidenceCoupling(status string, hasNativeEvent, hasOperation bool) error {
	if status == "synthesized" {
		if hasNativeEvent {
			return invalid("source evidence synthesized carries a native event ID")
		}
		if !hasOperation {
			return invalid("source evidence synthesized requires a core operation")
		}
		return nil
	}
	if hasOperation {
		return invalid("source evidence %s carries a core operation", status)
	}
	return nil
}

var canonicalEventMembers = map[string]bool{
	"schema": true, "schema_version": true, "event_id": true,
	"logical_session_id": true, "ordinal": true, "parents": true,
	"actor_id": true, "turn_id": true, "kind": true, "timestamp": true,
	"visibility": true, "payload": true, "source_evidence": true, "extensions": true,
}

var canonicalEventRequired = []string{
	"schema", "schema_version", "event_id",
	"logical_session_id", "ordinal", "parents",
	"actor_id", "turn_id", "kind", "timestamp",
	"visibility", "payload", "source_evidence", "extensions",
}

// CanonicalEvent is one validated Canonical Event. Payload carries
// the validated closed payload object.
type CanonicalEvent struct {
	EventID          scalar.Digest
	LogicalSessionID scalar.UUIDv7
	Ordinal          uint64
	Parents          []scalar.Digest
	ActorID          scalar.UUIDv7
	TurnID           *scalar.UUIDv7
	Kind             string
	Timestamp        *scalar.Timestamp
	Visibility       string
	Payload          map[string]any
	Evidence         SourceEvidence
}

// RawRefInput is one caller-supplied raw reference candidate.
type RawRefInput struct {
	ManifestID       string
	BlobDescriptorID string
	Offset           uint64
	Length           uint64
}

// EvidenceInput is one caller-supplied source evidence candidate
// for Build.
type EvidenceInput struct {
	Environment     []byte
	NativeSessionID string
	NativeEventID   *string
	NativeType      *string
	RawRefs         []RawRefInput
	CaptureStatus   string
	ReasonCodes     []string
	CoreOperationID *string
}

// CanonicalEventInput is the caller-supplied canonical event
// candidate for Build. Payload carries the exact closed payload
// object bytes; Build validates them under the kind-selected rules.
type CanonicalEventInput struct {
	LogicalSessionID string
	Ordinal          uint64
	Parents          []string
	ActorID          string
	TurnID           *string
	Kind             string
	Timestamp        *string
	Visibility       string
	Payload          []byte
	Evidence         EvidenceInput
	Extensions       map[string]any
}

// BuildCanonicalEvent constructs one closed Canonical Event 1.0.0
// as canonical bytes. Identical inputs produce byte-identical
// outputs.
func BuildCanonicalEvent(input CanonicalEventInput) ([]byte, error) {
	object, err := buildCanonicalEvent(input)
	if err != nil {
		return nil, err
	}
	omitted, err := canonicalizeObject(object)
	if err != nil {
		return nil, err
	}
	eventID := scalar.SHA256Digest(omitted)
	object[canonicalEventSelf] = eventID.String()
	return canonicalizeObject(object)
}

func buildCanonicalEvent(input CanonicalEventInput) (map[string]any, error) {
	logical, err := scalar.ParseUUIDv7(input.LogicalSessionID)
	if err != nil {
		return nil, invalid("canonical event logical_session_id is not a UUIDv7: %v", err)
	}
	if input.Ordinal > maxUint53 {
		return nil, invalid("canonical event ordinal exceeds uint53")
	}
	parents, err := parseSortedUniqueDigestStrings(input.Parents, 0, 64, "parents")
	if err != nil {
		return nil, invalid("canonical event %v", err)
	}
	actorID, err := scalar.ParseUUIDv7(input.ActorID)
	if err != nil {
		return nil, invalid("canonical event actor_id is not a UUIDv7: %v", err)
	}
	var turn any
	if input.TurnID != nil {
		value, err := scalar.ParseUUIDv7(*input.TurnID)
		if err != nil {
			return nil, invalid("canonical event turn_id is not a UUIDv7: %v", err)
		}
		turn = value.String()
	}
	if !ValidEventKind(input.Kind) {
		return nil, invalid("canonical event kind is outside the 26-kind vocabulary")
	}
	var stamp any
	if input.Timestamp != nil {
		value, err := scalar.ParseTimestamp(*input.Timestamp)
		if err != nil {
			return nil, invalid("canonical event timestamp is not a timestamp: %v", err)
		}
		stamp = value.String()
	}
	if !validVisibility(input.Visibility) {
		return nil, invalid("canonical event visibility is outside public|projection|internal|opaque")
	}
	payload, err := checkPayload(input.Kind, json.RawMessage(input.Payload))
	if err != nil {
		return nil, err
	}
	evidence, err := buildSourceEvidence(input.Evidence)
	if err != nil {
		return nil, err
	}
	if _, err := encodeExtensions(input.Extensions); err != nil {
		return nil, err
	}
	return map[string]any{
		"schema":             canonicalEventSchema,
		"schema_version":     canonicalEventVersion,
		"logical_session_id": logical.String(),
		"ordinal":            input.Ordinal,
		"parents":            digestStrings(parents),
		"actor_id":           actorID.String(),
		"turn_id":            turn,
		"kind":               input.Kind,
		"timestamp":          stamp,
		"visibility":         input.Visibility,
		"payload":            payload,
		"source_evidence":    evidence,
		"extensions":         extensionValue(input.Extensions),
	}, nil
}

// DecodeCanonicalEvent validates one closed Canonical Event 1.0.0:
// exact members, schema/version, self-digest agreement, the kind
// and visibility vocabularies, the kind-selected payload rules,
// and the source evidence coupling.
func DecodeCanonicalEvent(data []byte) (CanonicalEvent, error) {
	members, fault := decodeStrictObject(data)
	if fault != nil {
		return CanonicalEvent{}, invalid("canonical event %s (%s)", fault.detail, memberField(fault.member))
	}
	if name, unknown := unknownMember(members, canonicalEventMembers); unknown {
		return CanonicalEvent{}, invalid("canonical event carries unknown member %q", name)
	}
	if name, missing := missingMember(members, canonicalEventRequired); missing {
		return CanonicalEvent{}, invalid("canonical event misses a required member %q", name)
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != canonicalEventSchema {
		return CanonicalEvent{}, invalid("canonical event schema is not the canonical event")
	}
	version, ok := rawString(members["schema_version"])
	if !ok || version != canonicalEventVersion {
		return CanonicalEvent{}, invalid("canonical event version is not 1.0.0")
	}
	eventID, ok := checkDigest(members[canonicalEventSelf])
	if !ok {
		return CanonicalEvent{}, invalid("canonical event event_id is not a digest")
	}
	logical, ok := checkUUIDv7(members["logical_session_id"])
	if !ok {
		return CanonicalEvent{}, invalid("canonical event logical_session_id is not a UUIDv7")
	}
	ordinal, ok := checkUint53Bounds(members["ordinal"], 0, maxUint53)
	if !ok {
		return CanonicalEvent{}, invalid("canonical event ordinal is not a uint53")
	}
	parents, ok := checkSortedUniqueDigests(members["parents"], 0, 64)
	if !ok {
		return CanonicalEvent{}, invalid("canonical event parents are not sorted unique digest[0..64]")
	}
	actorID, ok := checkUUIDv7(members["actor_id"])
	if !ok {
		return CanonicalEvent{}, invalid("canonical event actor_id is not a UUIDv7")
	}
	var turn *scalar.UUIDv7
	if !isNull(members["turn_id"]) {
		value, ok := checkUUIDv7(members["turn_id"])
		if !ok {
			return CanonicalEvent{}, invalid("canonical event turn_id is not a UUIDv7")
		}
		turn = &value
	}
	kind, ok := rawString(members["kind"])
	if !ok || !ValidEventKind(kind) {
		return CanonicalEvent{}, invalid("canonical event kind is outside the 26-kind vocabulary")
	}
	var stamp *scalar.Timestamp
	if !isNull(members["timestamp"]) {
		value, ok := checkTimestamp(members["timestamp"])
		if !ok {
			return CanonicalEvent{}, invalid("canonical event timestamp is not a timestamp")
		}
		stamp = &value
	}
	visibility, ok := rawString(members["visibility"])
	if !ok || !validVisibility(visibility) {
		return CanonicalEvent{}, invalid("canonical event visibility is outside public|projection|internal|opaque")
	}
	payload, err := checkPayload(kind, members["payload"])
	if err != nil {
		return CanonicalEvent{}, err
	}
	evidence, err := DecodeSourceEvidence(members["source_evidence"])
	if err != nil {
		return CanonicalEvent{}, err
	}
	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return CanonicalEvent{}, invalid("canonical event extensions %s", extensionsFault)
	}
	if err := verifySelfDigest(members, canonicalEventSelf, eventID); err != nil {
		return CanonicalEvent{}, err
	}
	return CanonicalEvent{
		EventID:          eventID,
		LogicalSessionID: logical,
		Ordinal:          ordinal,
		Parents:          parents,
		ActorID:          actorID,
		TurnID:           turn,
		Kind:             kind,
		Timestamp:        stamp,
		Visibility:       visibility,
		Payload:          payload,
		Evidence:         evidence,
	}, nil
}

// checkPayload validates the kind-selected closed payload: every
// payload is a strict (duplicate-free) object carrying only
// Section 1.6 values at every depth; message-like kinds carry
// exactly content_blocks plus extensions; content blocks use exactly
// the eight-type vocabulary with typed inline content (at most 64
// KiB) or Blob Descriptor references. Per-kind fact registries
// beyond this envelope are sibling scope, but the value model is
// enforced here for every kind: no payload member may round or
// collapse under sealing.
func checkPayload(kind string, raw json.RawMessage) (map[string]any, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return nil, invalid("canonical event payload %s (%s)", fault.detail, memberField(fault.member))
	}
	if messageLikeKind(kind) {
		allowed := map[string]bool{"content_blocks": true, "extensions": true}
		if name, unknown := unknownMember(members, allowed); unknown {
			return nil, invalid("canonical event message payload carries unknown member %q", name)
		}
		blocks, present := members["content_blocks"]
		if !present {
			return nil, invalid("canonical event message payload misses content_blocks")
		}
		if err := checkContentBlocks(blocks); err != nil {
			return nil, err
		}
		if extensions, present := members["extensions"]; present {
			if extensionsFault := checkExtensionsClosed(extensions); extensionsFault != nil {
				return nil, invalid("canonical event message payload extensions %s", extensionsFault)
			}
		}
	} else if blocks, present := members["content_blocks"]; present {
		if err := checkContentBlocks(blocks); err != nil {
			return nil, err
		}
	}
	if err := checkPayloadValues(bytesTrimSpace(raw)); err != nil {
		return nil, invalid("canonical event payload values %s", err)
	}
	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(bytesTrimSpace(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, invalid("canonical event payload is not an object: %v", err)
	}
	return convertPayloadNumbers(payload)
}

// checkPayloadValues enforces the Section 1.6 value model on every
// payload member at every depth: integer literals only, |n| <=
// 2^53-1, no fraction or exponent, nested objects duplicate-free,
// depth-bounded. Payload keys are per-kind fact names rather than
// reverse-DNS names, so the closed extensions key rule does not
// apply here; only the shared value walker runs. This walker is the
// single gate: the converter below trusts it and only changes
// number representation, never admission.
func checkPayloadValues(raw json.RawMessage) error {
	if err := checkExtensionValues(raw); err != nil {
		if err == errExtensionShape {
			return errPayloadShape
		}
		return err
	}
	return nil
}

// convertPayloadNumbers renders the validated payload with exact
// integers: every json.Number the gate admitted is an in-range
// integer literal, so it becomes an int64 that marshals
// byte-exact. Host float64 never appears, so no sealed value can
// round. A literal ParseInt cannot read is refused defensively;
// the gate above admits no such literal.
func convertPayloadNumbers(payload map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(payload))
	for name, value := range payload {
		converted, err := convertPayloadValue(value)
		if err != nil {
			return nil, err
		}
		out[name] = converted
	}
	return out, nil
}

// convertPayloadValue converts one validated payload value:
// numbers become exact int64, containers recurse, and every other
// scalar crosses untouched.
func convertPayloadValue(value any) (any, error) {
	switch typed := value.(type) {
	case json.Number:
		integer, err := strconv.ParseInt(typed.String(), 10, 64)
		if err != nil {
			return nil, invalid("canonical event payload carries a number outside the AX safe-integer model")
		}
		return integer, nil
	case map[string]any:
		return convertPayloadNumbers(typed)
	case []any:
		out := make([]any, len(typed))
		for index, element := range typed {
			converted, err := convertPayloadValue(element)
			if err != nil {
				return nil, err
			}
			out[index] = converted
		}
		return out, nil
	default:
		return value, nil
	}
}

var contentBlockMembers = map[string]bool{
	"type": true, "content": true, "blob_descriptor_id": true, "media_type": true, "extensions": true,
}

// checkContentBlocks validates message-like content blocks: strict
// objects with a vocabulary type, inline content of at most 64 KiB,
// and digest-grammar Blob Descriptor references. The pinned text
// gives each block "typed inline content or Blob Descriptor
// references": exactly one of the two members must be present and
// non-null, so a block with neither (including a null where the
// member belongs) and a block with both are refused.
func checkContentBlocks(raw json.RawMessage) error {
	elements, ok := decodeArray(raw)
	if !ok {
		return invalid("canonical event content_blocks are not an array")
	}
	for index, element := range elements {
		members, fault := decodeStrictObject(bytesTrimSpace(element))
		if fault != nil {
			return invalid("canonical event content block[%d] %s (%s)", index, fault.detail, memberField(fault.member))
		}
		if name, unknown := unknownMember(members, contentBlockMembers); unknown {
			return invalid("canonical event content block[%d] carries unknown member %q", index, name)
		}
		blockTypeValue, present := members["type"]
		if !present {
			return invalid("canonical event content block[%d] misses its type", index)
		}
		blockType, ok := rawString(blockTypeValue)
		if !ok || !validContentBlockType(blockType) {
			return invalid("canonical event content block[%d] type is outside the eight-type vocabulary", index)
		}
		_, hasContentMember := members["content"]
		_, hasDescriptorMember := members["blob_descriptor_id"]
		hasContent := hasContentMember && !isNull(members["content"])
		hasDescriptor := hasDescriptorMember && !isNull(members["blob_descriptor_id"])
		if hasContent == hasDescriptor {
			return invalid("canonical event content block[%d] carries content and descriptor members that are not exclusive", index)
		}
		if content, present := members["content"]; present {
			text, ok := rawString(content)
			if !ok {
				return invalid("canonical event content block[%d] content is not a string", index)
			}
			if len(text) > maxInlineContent {
				return invalid("canonical event content block[%d] inline content exceeds 64 KiB", index)
			}
		}
		if descriptor, present := members["blob_descriptor_id"]; present {
			if _, ok := checkDigest(descriptor); !ok {
				return invalid("canonical event content block[%d] blob_descriptor_id is not a digest", index)
			}
		}
		if mediaType, present := members["media_type"]; present {
			if _, ok := checkStringBounds(mediaType, 1, 255); !ok {
				return invalid("canonical event content block[%d] media_type is not a string[1..255]", index)
			}
		}
		if extensions, present := members["extensions"]; present {
			if extensionsFault := checkExtensionsClosed(extensions); extensionsFault != nil {
				return invalid("canonical event content block[%d] extensions %s", index, extensionsFault)
			}
		}
	}
	return nil
}

func buildSourceEvidence(input EvidenceInput) (map[string]any, error) {
	tuple, err := sessadapter.DecodeTuple(json.RawMessage(input.Environment))
	if err != nil {
		return nil, invalid("source evidence environment is not an Environment Tuple: %v", err)
	}
	if stringLength(input.NativeSessionID) < 1 || stringLength(input.NativeSessionID) > 512 {
		return nil, invalid("source evidence native_session_id is not a string[1..512]")
	}
	if err := SanitizeNativeKey(input.NativeSessionID); err != nil {
		return nil, err
	}
	var nativeEvent any
	hasNativeEvent := false
	if input.NativeEventID != nil {
		if stringLength(*input.NativeEventID) < 1 || stringLength(*input.NativeEventID) > 512 {
			return nil, invalid("source evidence native_event_id is not a string[1..512]")
		}
		if err := SanitizeNativeKey(*input.NativeEventID); err != nil {
			return nil, err
		}
		nativeEvent = *input.NativeEventID
		hasNativeEvent = true
	}
	var nativeType any
	if input.NativeType != nil {
		if !validText(*input.NativeType) {
			return nil, invalid("source evidence native_type is not valid UTF-8")
		}
		if stringLength(*input.NativeType) < 1 || stringLength(*input.NativeType) > 512 {
			return nil, invalid("source evidence native_type is not a string[1..512]")
		}
		nativeType = *input.NativeType
	}
	if len(input.RawRefs) > 65536 {
		return nil, invalid("source evidence carries %d raw_refs, maximum is 65536", len(input.RawRefs))
	}
	references := make([]RawReference, 0, len(input.RawRefs))
	var previous []byte
	for index, candidate := range input.RawRefs {
		manifestID, err := scalar.ParseDigest(candidate.ManifestID)
		if err != nil {
			return nil, invalid("raw reference[%d] manifest_id is not a digest: %v", index, err)
		}
		descriptorID, err := scalar.ParseDigest(candidate.BlobDescriptorID)
		if err != nil {
			return nil, invalid("raw reference[%d] blob_descriptor_id is not a digest: %v", index, err)
		}
		if candidate.Offset > maxUint53 {
			return nil, invalid("raw reference[%d] offset exceeds uint53", index)
		}
		if candidate.Length > maxUint53 {
			return nil, invalid("raw reference[%d] length exceeds uint53", index)
		}
		reference := RawReference{
			ManifestID:       manifestID,
			BlobDescriptorID: descriptorID,
			Offset:           candidate.Offset,
			Length:           candidate.Length,
		}
		key, err := canonicalizeObject(rawReferenceObject(reference))
		if err != nil {
			return nil, err
		}
		if index > 0 && bytes.Compare(key, previous) <= 0 {
			return nil, invalid("source evidence raw_refs are not sorted unique")
		}
		previous = key
		references = append(references, reference)
	}
	if !validCaptureStatus(input.CaptureStatus) {
		return nil, invalid("source evidence capture_status is outside exact|partial|synthesized|unavailable")
	}
	previousReason := ""
	for index, reason := range input.ReasonCodes {
		if !validText(reason) {
			return nil, invalid("source evidence reason_codes[%d] is not valid UTF-8", index)
		}
		if stringLength(reason) < 1 || stringLength(reason) > 128 {
			return nil, invalid("source evidence reason_codes[%d] is not a string[1..128]", index)
		}
		if index > 0 && reason <= previousReason {
			return nil, invalid("source evidence reason_codes are not sorted unique")
		}
		previousReason = reason
	}
	if len(input.ReasonCodes) > 128 {
		return nil, invalid("source evidence carries %d reason_codes, maximum is 128", len(input.ReasonCodes))
	}
	var operation any
	hasOperation := false
	if input.CoreOperationID != nil {
		value, err := scalar.ParseUUIDv7(*input.CoreOperationID)
		if err != nil {
			return nil, invalid("source evidence core_operation_id is not a UUIDv7: %v", err)
		}
		operation = value.String()
		hasOperation = true
	}
	if err := checkEvidenceCoupling(input.CaptureStatus, hasNativeEvent, hasOperation); err != nil {
		return nil, err
	}
	objects := make([]any, 0, len(references))
	for _, reference := range references {
		objects = append(objects, rawReferenceObject(reference))
	}
	reasons := make([]any, 0, len(input.ReasonCodes))
	for _, reason := range input.ReasonCodes {
		reasons = append(reasons, reason)
	}
	return map[string]any{
		"environment":       tupleObject(tuple),
		"native_session_id": input.NativeSessionID,
		"native_event_id":   nativeEvent,
		"native_type":       nativeType,
		"raw_refs":          objects,
		"capture_status":    input.CaptureStatus,
		"reason_codes":      reasons,
		"core_operation_id": operation,
		"extensions":        map[string]any{},
	}, nil
}

// CheckOrdinalContiguity verifies the event-set ordinal rule:
// ordinals are zero-based and contiguous, so the set must be
// exactly {0..n-1}.
func CheckOrdinalContiguity(ordinals []uint64) error {
	seen := make(map[uint64]bool, len(ordinals))
	for _, ordinal := range ordinals {
		if ordinal >= uint64(len(ordinals)) {
			return invalid("event ordinals are not contiguous from zero: ordinal %d outside [0..%d]", ordinal, len(ordinals)-1)
		}
		if seen[ordinal] {
			return invalid("event ordinals are not contiguous from zero: duplicate ordinal %d", ordinal)
		}
		seen[ordinal] = true
	}
	return nil
}
