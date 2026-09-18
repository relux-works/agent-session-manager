package cloneproject

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/environ"
)

// This file parses the fixture-native record envelope. The envelope
// is test-defined framing, not a specification shape: its only job
// is to carry attributable bytes (native ID, native type, origin,
// protection, actor, body) from captured members to the projection.
// Every per-kind body shape below is exact: unknown body members
// refuse rather than silently drop, so a projection can never claim
// a faithful kind while ignoring a fact it does not understand. Type
// level unknowns become opaque_event; field level unknowns inside a
// known type refuse (see doc.go).

// NativeRecord is one parsed fixture-native record with its byte
// range inside the member blob.
type NativeRecord struct {
	// MemberKey is the capture native item key holding the record.
	MemberKey string
	// Line is the one-based line number inside the member.
	Line int
	// Offset and Length are the record's byte range: the line
	// without its newline terminator.
	Offset uint64
	Length uint64
	// NativeEventID and NativeType attribute the record.
	NativeEventID string
	NativeType    string
	// Origin is native or foreign.
	Origin string
	// Protection is none, encrypted, or signed.
	Protection string
	// Actor is main, external, or subagent:<name>.
	Actor string
	// Body carries the raw body value; kind code reads only its
	// registered members.
	Body json.RawMessage
	// LineBytes are the exact record bytes preserved for opaque kinds.
	LineBytes []byte
}

// nativeEnvelope is the decoded envelope shape. Its members are
// filled from the strict map by exact name (see parseNativeLine),
// never through struct decoding: Body stays raw and each known
// type validates its own exact members. The version member is read
// through the landed uint53 gate, never through a lenient number
// field: a string-typed version is not version 1.
type nativeEnvelope struct {
	NativeEventID string          `json:"native_event_id"`
	NativeType    string          `json:"native_type"`
	Origin        string          `json:"origin"`
	Protection    string          `json:"protection"`
	Actor         string          `json:"actor"`
	Body          json.RawMessage `json:"body"`
}

// strictMembers decodes one strict JSON object through the landed
// environ gate: non-UTF-8 bytes, lone surrogate escapes, duplicate
// members, non-object top levels, and trailing data refuse here,
// before any member is read, so no decoded value can carry
// rewritten text or a guessed last-wins member. Both frame sites
// (the record envelope and the per-kind bodies) pass through this
// gate, and every later read takes its value from the admitted map
// by exact member name: no second decoder re-reads the line, so a
// case-folded alias ("NATIVE_TYPE", "Origin", ...) is an unclaimed
// extra that can never override a claimed value. The rewriting and
// guessing shapes never reach the readers.
func strictMembers(what string, raw []byte) (map[string]json.RawMessage, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault == nil {
		return members, nil
	}
	if fault.Detail == environ.FaultNotObject {
		return nil, invalid("%s is not a JSON object: %v", what, fault)
	}
	return nil, invalid("%s is not a strict JSON object: %v", what, fault)
}

// parseNativeLine parses one record line. A line that is not valid
// UTF-8, not a strict JSON object (lone surrogate escape,
// duplicate member, trailing data), or misses an envelope member
// refuses with the member and line named: malformed framing is an
// error, never an absent or guessed record.
func parseNativeLine(memberKey string, line int, offset uint64, raw []byte) (NativeRecord, error) {
	where := func() string {
		return "record " + memberKey + " line " + itoa(line)
	}
	if !utf8.Valid(raw) {
		return NativeRecord{}, invalid("%s is not valid UTF-8", where())
	}
	members, err := strictMembers(where(), raw)
	if err != nil {
		return NativeRecord{}, err
	}
	for _, name := range []string{"v", "native_event_id", "native_type", "origin", "protection", "actor", "body"} {
		if _, ok := members[name]; !ok {
			return NativeRecord{}, invalid("%s misses envelope member %q", where(), name)
		}
	}
	var envelope nativeEnvelope
	// Strict-map envelope: every claimed member is read from the
	// admitted map by its exact name. No second decoder re-reads
	// the line, so a case-folded alias ("NATIVE_TYPE", "Origin",
	// "Protection", "Body", "Native_Event_Id", "Actor") stays an
	// unclaimed extra and can never override a claimed value,
	// whatever its document order.
	for _, field := range []struct {
		name   string
		target *string
	}{
		{"native_event_id", &envelope.NativeEventID},
		{"native_type", &envelope.NativeType},
		{"origin", &envelope.Origin},
		{"protection", &envelope.Protection},
		{"actor", &envelope.Actor},
	} {
		if err := json.Unmarshal(members[field.name], field.target); err != nil {
			return NativeRecord{}, invalid("%s envelope member %q is not a string", where(), field.name)
		}
	}
	envelope.Body = members["body"]
	if _, ok := environ.CheckUint53Bounds(members["v"], 1, 1); !ok {
		return NativeRecord{}, invalid("%s carries envelope version %q, want 1", where(), string(members["v"]))
	}
	if environ.StringLength(envelope.NativeEventID) < 1 || environ.StringLength(envelope.NativeEventID) > 512 {
		return NativeRecord{}, invalid("%s native_event_id is not a string[1..512]", where())
	}
	if environ.StringLength(envelope.NativeType) < 1 || environ.StringLength(envelope.NativeType) > 512 {
		return NativeRecord{}, invalid("%s native_type is not a string[1..512]", where())
	}
	if envelope.Origin != "native" && envelope.Origin != "foreign" {
		return NativeRecord{}, invalid("%s origin %q is outside native|foreign", where(), envelope.Origin)
	}
	if envelope.Protection != "none" && envelope.Protection != "encrypted" && envelope.Protection != "signed" {
		return NativeRecord{}, invalid("%s protection %q is outside none|encrypted|signed", where(), envelope.Protection)
	}
	if err := checkActorSelector(envelope.Actor); err != nil {
		return NativeRecord{}, invalid("%s %v", where(), err)
	}
	return NativeRecord{
		MemberKey:     memberKey,
		Line:          line,
		Offset:        offset,
		Length:        uint64(len(raw)),
		NativeEventID: envelope.NativeEventID,
		NativeType:    envelope.NativeType,
		Origin:        envelope.Origin,
		Protection:    envelope.Protection,
		Actor:         envelope.Actor,
		Body:          envelope.Body,
		LineBytes:     append([]byte(nil), raw...),
	}, nil
}

// checkActorSelector validates main, external, or subagent:<name>
// with a bounded non-empty name.
func checkActorSelector(selector string) error {
	if selector == "main" || selector == "external" {
		return nil
	}
	name, ok := strings.CutPrefix(selector, "subagent:")
	if !ok || name == "" {
		return invalid("actor %q is outside main|external|subagent:<name>", selector)
	}
	if environ.StringLength(name) > 128 {
		return invalid("actor %q names a subagent past 128 characters", selector)
	}
	return nil
}

// splitRecordLines splits member bytes into record lines with byte
// offsets. A zero-byte member holds zero records; otherwise one
// trailing newline is the terminator and every remaining line must
// be non-empty: a blank line is malformed framing, never an absent
// record. A CRLF member is admitted with each trailing CR inside
// its record range: the range covers the line plus the CR and
// resolves exactly (pinned by
// TestNormalizeCRLFResolvesWithCarriageReturn).
func splitRecordLines(memberKey string, payload []byte) ([]recordLine, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	rest := payload
	if rest[len(rest)-1] == '\n' {
		rest = rest[:len(rest)-1]
	}
	var lines []recordLine
	offset := uint64(0)
	number := 1
	for {
		end := bytes.IndexByte(rest, '\n')
		var raw []byte
		if end < 0 {
			raw = rest
			rest = nil
		} else {
			raw = rest[:end]
			rest = rest[end+1:]
		}
		if len(raw) == 0 {
			return nil, invalid("record %s line %d is empty", memberKey, number)
		}
		lines = append(lines, recordLine{number: number, offset: offset, raw: raw})
		offset += uint64(len(raw)) + 1
		number++
		if rest == nil {
			break
		}
	}
	return lines, nil
}

// recordLine is one split line with its number and byte offset.
type recordLine struct {
	number int
	offset uint64
	raw    []byte
}

// itoa renders a line number for refusal messages.
func itoa(value int) string {
	return strconv.Itoa(value)
}

// Known fixture-native types. Every other native_type projects as
// opaque_event.
const (
	nativeMessageUser      = "message/user"
	nativeMessageAssistant = "message/assistant"
	nativeReasoningSummary = "reasoning/summary"
	nativeReasoningCrypt   = "reasoning/encrypted"
	nativeReasoningSigned  = "reasoning/signed"
	nativeInstruction      = "instruction/snapshot"
	nativeToolDefinition   = "tool/definition"
	nativeToolCall         = "tool/call"
	nativeToolResult       = "tool/result"
	nativeUsage            = "usage/report"
)

// knownNativeType reports whether the native type selects a
// registered projection. Unknown types become opaque_event.
func knownNativeType(nativeType string) bool {
	switch nativeType {
	case nativeMessageUser,
		nativeMessageAssistant,
		nativeReasoningSummary,
		nativeReasoningCrypt,
		nativeReasoningSigned,
		nativeInstruction,
		nativeToolDefinition,
		nativeToolCall,
		nativeToolResult,
		nativeUsage:
		return true
	default:
		return false
	}
}

// decodeBodyObject decodes a body object with exact members: every
// member must be registered for the type, and every required member
// must be present. Unknown members refuse. The frame decode is the
// landed strict gate, so a duplicated body member refuses instead
// of guessing last-wins.
func decodeBodyObject(record NativeRecord, required []string, allowed map[string]bool) (map[string]json.RawMessage, error) {
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	members, err := strictMembers(where+" body", bytes.TrimSpace(record.Body))
	if err != nil {
		return nil, err
	}
	for name := range members {
		if !allowed[name] {
			return nil, invalid("%s body carries unknown member %q", where, name)
		}
	}
	for _, name := range required {
		if _, ok := members[name]; !ok {
			return nil, invalid("%s body misses member %q", where, name)
		}
	}
	return members, nil
}

// bodyString reads one required string body member.
func bodyString(record NativeRecord, members map[string]json.RawMessage, name string) (string, error) {
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	var value string
	if err := json.Unmarshal(members[name], &value); err != nil {
		return "", invalid("%s body member %q is not a string", where, name)
	}
	return value, nil
}

// bodyOptionalBool reads one optional boolean body member.
func bodyOptionalBool(record NativeRecord, members map[string]json.RawMessage, name string) (bool, error) {
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	raw, ok := members[name]
	if !ok {
		return false, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, invalid("%s body member %q is not a boolean", where, name)
	}
	return value, nil
}

// bodyUint53 reads one required uint53 body member through the
// landed environ gate: a JSON integer literal in [0..2^53-1],
// never a fraction, exponent, sign, out-of-range magnitude, or
// string. Every failure is one refusal: anything outside the type
// is not a uint53.
func bodyUint53(record NativeRecord, members map[string]json.RawMessage, name string) (uint64, error) {
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	value, ok := environ.CheckUint53Bounds(members[name], 0, maxUint53)
	if !ok {
		return 0, invalid("%s body member %q is not a uint53", where, name)
	}
	return value, nil
}

// maxUint53 is 2^53-1, the AX safe-integer ceiling.
const maxUint53 = uint64(1<<53 - 1)

// messageBody is the exact message/reasoning-summary body: text only.
type messageBody struct {
	Text string
}

// decodeMessageBody validates the exact {"text"} body.
func decodeMessageBody(record NativeRecord) (messageBody, error) {
	members, err := decodeBodyObject(record, []string{"text"}, map[string]bool{"text": true})
	if err != nil {
		return messageBody{}, err
	}
	text, err := bodyString(record, members, "text")
	if err != nil {
		return messageBody{}, err
	}
	return messageBody{Text: text}, nil
}

// protectedBody is the exact protected-reasoning body: ciphertext
// the projection never reads, only preserves byte-exact.
type protectedBody struct {
	Ciphertext string
}

// decodeProtectedBody validates the exact {"ciphertext"} body. The
// value is framing-checked as a string and never interpreted.
func decodeProtectedBody(record NativeRecord) (protectedBody, error) {
	members, err := decodeBodyObject(record, []string{"ciphertext"}, map[string]bool{"ciphertext": true})
	if err != nil {
		return protectedBody{}, err
	}
	ciphertext, err := bodyString(record, members, "ciphertext")
	if err != nil {
		return protectedBody{}, err
	}
	return protectedBody{Ciphertext: ciphertext}, nil
}

// instructionBody is the exact instruction-snapshot body.
type instructionBody struct {
	Authority  string
	Directives []string
}

// decodeInstructionBody validates authority high|low and up to 1024
// directives of 1..4096 characters each.
func decodeInstructionBody(record NativeRecord) (instructionBody, error) {
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	members, err := decodeBodyObject(record, []string{"authority", "directives"}, map[string]bool{"authority": true, "directives": true})
	if err != nil {
		return instructionBody{}, err
	}
	authority, err := bodyString(record, members, "authority")
	if err != nil {
		return instructionBody{}, err
	}
	if authority != "high" && authority != "low" {
		return instructionBody{}, invalid("%s body authority %q is outside high|low", where, authority)
	}
	var directives []string
	if err := json.Unmarshal(members["directives"], &directives); err != nil {
		return instructionBody{}, invalid("%s body member %q is not an array of strings", where, "directives")
	}
	if len(directives) > 1024 {
		return instructionBody{}, invalid("%s body carries %d directives, maximum is 1024", where, len(directives))
	}
	for index, directive := range directives {
		if environ.StringLength(directive) < 1 || environ.StringLength(directive) > 4096 {
			return instructionBody{}, invalid("%s body directive[%d] is not a string[1..4096]", where, index)
		}
	}
	return instructionBody{Authority: authority, Directives: directives}, nil
}

// toolDefinitionBody is the exact tool-definition body.
type toolDefinitionBody struct {
	ToolName       string
	DefinitionHash string
	LiveClaim      bool
}

// decodeToolDefinitionBody validates the exact definition body. A
// live_attestation claim is parsed so the live-surface gate can
// refuse it: captured history can never self-promote.
func decodeToolDefinitionBody(record NativeRecord) (toolDefinitionBody, error) {
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	members, err := decodeBodyObject(record,
		[]string{"tool_name", "definition_digest"},
		map[string]bool{"tool_name": true, "definition_digest": true, "live_attestation": true})
	if err != nil {
		return toolDefinitionBody{}, err
	}
	name, err := bodyString(record, members, "tool_name")
	if err != nil {
		return toolDefinitionBody{}, err
	}
	if environ.StringLength(name) < 1 || environ.StringLength(name) > 512 {
		return toolDefinitionBody{}, invalid("%s body tool_name is not a string[1..512]", where)
	}
	digest, err := bodyString(record, members, "definition_digest")
	if err != nil {
		return toolDefinitionBody{}, err
	}
	live, err := bodyOptionalBool(record, members, "live_attestation")
	if err != nil {
		return toolDefinitionBody{}, err
	}
	return toolDefinitionBody{ToolName: name, DefinitionHash: digest, LiveClaim: live}, nil
}

// toolCallBody is the exact tool-call body.
type toolCallBody struct {
	CallID   string
	ToolName string
}

// decodeToolCallBody validates the exact call body.
func decodeToolCallBody(record NativeRecord) (toolCallBody, error) {
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	members, err := decodeBodyObject(record,
		[]string{"call_id", "tool_name"},
		map[string]bool{"call_id": true, "tool_name": true})
	if err != nil {
		return toolCallBody{}, err
	}
	callID, err := bodyString(record, members, "call_id")
	if err != nil {
		return toolCallBody{}, err
	}
	if environ.StringLength(callID) < 1 || environ.StringLength(callID) > 512 {
		return toolCallBody{}, invalid("%s body call_id is not a string[1..512]", where)
	}
	name, err := bodyString(record, members, "tool_name")
	if err != nil {
		return toolCallBody{}, err
	}
	if environ.StringLength(name) < 1 || environ.StringLength(name) > 512 {
		return toolCallBody{}, invalid("%s body tool_name is not a string[1..512]", where)
	}
	return toolCallBody{CallID: callID, ToolName: name}, nil
}

// toolResultBody is the exact tool-result body.
type toolResultBody struct {
	CallID       string
	Status       string
	LiveFollowup bool
}

// decodeToolResultBody validates the exact result body. A
// live_followup claim is parsed so the resolution gate can refuse
// it: captured history can never request a live action.
func decodeToolResultBody(record NativeRecord) (toolResultBody, error) {
	where := "record " + record.MemberKey + " line " + itoa(record.Line)
	members, err := decodeBodyObject(record,
		[]string{"call_id", "status"},
		map[string]bool{"call_id": true, "status": true, "live_followup": true})
	if err != nil {
		return toolResultBody{}, err
	}
	callID, err := bodyString(record, members, "call_id")
	if err != nil {
		return toolResultBody{}, err
	}
	if environ.StringLength(callID) < 1 || environ.StringLength(callID) > 512 {
		return toolResultBody{}, invalid("%s body call_id is not a string[1..512]", where)
	}
	status, err := bodyString(record, members, "status")
	if err != nil {
		return toolResultBody{}, err
	}
	if status != "ok" && status != "error" {
		return toolResultBody{}, invalid("%s body status %q is outside ok|error", where, status)
	}
	followup, err := bodyOptionalBool(record, members, "live_followup")
	if err != nil {
		return toolResultBody{}, err
	}
	return toolResultBody{CallID: callID, Status: status, LiveFollowup: followup}, nil
}

// usageBody is the exact usage-report body.
type usageBody struct {
	InputTokens  uint64
	OutputTokens uint64
}

// decodeUsageBody validates the exact token body.
func decodeUsageBody(record NativeRecord) (usageBody, error) {
	members, err := decodeBodyObject(record,
		[]string{"input_tokens", "output_tokens"},
		map[string]bool{"input_tokens": true, "output_tokens": true})
	if err != nil {
		return usageBody{}, err
	}
	input, err := bodyUint53(record, members, "input_tokens")
	if err != nil {
		return usageBody{}, err
	}
	output, err := bodyUint53(record, members, "output_tokens")
	if err != nil {
		return usageBody{}, err
	}
	return usageBody{InputTokens: input, OutputTokens: output}, nil
}
