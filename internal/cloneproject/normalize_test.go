package cloneproject

import (
	"bytes"
	"strings"
	"testing"
)

// Invariant 1: unknown native records become raw-addressable opaque
// events. Every test below drives the production Normalize entry
// over bytes the real clonesnap.Capture entry sealed.

func TestNormalizeUnknownRecordBecomesOpaqueEvent(t *testing.T) {
	line := rec("evt-unknown-1", "frobnicate", "native", "none", "main", `{"mystery":true}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(line)},
	})
	result := bundle.result
	if len(result.DecodedEvents) != 1 {
		t.Fatalf("len(DecodedEvents) = %d, want 1: unknown record was dropped or split", len(result.DecodedEvents))
	}
	event := result.DecodedEvents[0]
	if event.Kind != "opaque_event" {
		t.Fatalf("Kind = %q, want opaque_event", event.Kind)
	}
	if event.Visibility != "opaque" {
		t.Fatalf("Visibility = %q, want opaque", event.Visibility)
	}
	if event.Evidence.NativeEventID == nil || *event.Evidence.NativeEventID != "evt-unknown-1" {
		t.Fatalf("NativeEventID = %v, want evt-unknown-1", event.Evidence.NativeEventID)
	}
	if event.Evidence.NativeType == nil || *event.Evidence.NativeType != "frobnicate" {
		t.Fatalf("NativeType = %v, want frobnicate", event.Evidence.NativeType)
	}
	if event.Evidence.CaptureStatus != "exact" {
		t.Fatalf("CaptureStatus = %q, want exact", event.Evidence.CaptureStatus)
	}
	if len(event.Evidence.ReasonCodes) != 1 || event.Evidence.ReasonCodes[0] != "unknown_native_event" {
		t.Fatalf("ReasonCodes = %q, want [unknown_native_event]", event.Evidence.ReasonCodes)
	}
	if len(event.Evidence.RawRefs) != 1 {
		t.Fatalf("len(RawRefs) = %d, want 1", len(event.Evidence.RawRefs))
	}
	resolved := mustResolveRef(t, bundle.capture, bundle.store, event.Evidence.RawRefs[0])
	if string(resolved) != line {
		t.Fatalf("resolved bytes = %q, want the exact record line", string(resolved))
	}
	if payloadText(t, event, "native_type") != "frobnicate" {
		t.Fatalf("payload native_type = %q, want frobnicate", payloadText(t, event, "native_type"))
	}
	if payloadInt(t, event, "byte_count") != int64(len(line)) {
		t.Fatalf("payload byte_count = %d, want %d", payloadInt(t, event, "byte_count"), len(line))
	}
}

func TestNormalizeAlmostKnownRecordBecomesOpaqueEvent(t *testing.T) {
	// Right envelope, unknown inner type: message/smoke shares the
	// message/ prefix with known types but registers nothing.
	line := rec("evt-almost-1", "message/smoke", "native", "none", "main", `{"text":"smoke"}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(line)},
	})
	if len(bundle.result.DecodedEvents) != 1 {
		t.Fatalf("len(DecodedEvents) = %d, want 1", len(bundle.result.DecodedEvents))
	}
	event := bundle.result.DecodedEvents[0]
	if event.Kind != "opaque_event" {
		t.Fatalf("Kind = %q, want opaque_event: almost-known type was coerced into a neighbouring kind", event.Kind)
	}
	if event.Kind == "user_message" || event.Kind == "assistant_message" {
		t.Fatalf("Kind = %q: prefix match coerced the record into a known kind", event.Kind)
	}
	if event.Visibility != "opaque" {
		t.Fatalf("Visibility = %q, want opaque", event.Visibility)
	}
	if len(event.Evidence.ReasonCodes) != 1 || event.Evidence.ReasonCodes[0] != "unknown_native_event" {
		t.Fatalf("ReasonCodes = %q, want [unknown_native_event]", event.Evidence.ReasonCodes)
	}
	resolved := mustResolveRef(t, bundle.capture, bundle.store, event.Evidence.RawRefs[0])
	if string(resolved) != line {
		t.Fatalf("resolved bytes differ from the exact record line")
	}
}

func TestNormalizeUnknownMemberBecomesWholeMemberOpaque(t *testing.T) {
	content := []byte("\x00\x01binary-\xff-not-jsonl")
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/mystery", class: "unknown", content: content},
	})
	if len(bundle.result.DecodedEvents) != 1 {
		t.Fatalf("len(DecodedEvents) = %d, want 1", len(bundle.result.DecodedEvents))
	}
	event := bundle.result.DecodedEvents[0]
	if event.Kind != "opaque_event" {
		t.Fatalf("Kind = %q, want opaque_event", event.Kind)
	}
	if event.Visibility != "opaque" {
		t.Fatalf("Visibility = %q, want opaque", event.Visibility)
	}
	if event.Evidence.NativeEventID != nil {
		t.Fatalf("NativeEventID = %q, want null for a member-level opaque", *event.Evidence.NativeEventID)
	}
	if event.Evidence.NativeType != nil {
		t.Fatalf("NativeType = %q, want null for a member-level opaque", *event.Evidence.NativeType)
	}
	if len(event.Evidence.ReasonCodes) != 1 || event.Evidence.ReasonCodes[0] != "unknown_native_event" {
		t.Fatalf("ReasonCodes = %q, want [unknown_native_event]", event.Evidence.ReasonCodes)
	}
	ref := event.Evidence.RawRefs[0]
	if ref.Offset != 0 || ref.Length != uint64(len(content)) {
		t.Fatalf("raw range = [%d:%d], want the whole %d-byte member", ref.Offset, ref.Offset+ref.Length, len(content))
	}
	resolved := mustResolveRef(t, bundle.capture, bundle.store, ref)
	if !bytes.Equal(resolved, content) {
		t.Fatal("resolved bytes differ from the whole member")
	}
	if payloadText(t, event, "native_item_key") != "store/mystery" {
		t.Fatalf("payload native_item_key = %q, want store/mystery", payloadText(t, event, "native_item_key"))
	}
}

func TestNormalizeMultiRecordRefsResolveAtAssertedOffsets(t *testing.T) {
	// Raw-addressability past offset 0: every event's raw
	// reference resolves to its own record line, with the byte
	// offsets asserted, not merely the bytes. The first line
	// carries multibyte text so a character-measured stride
	// fails as well as a dropped newline stride.
	l1 := rec("evt-mr-1", "message/user", "native", "none", "main", `{"text":"héllo wörld"}`)
	l2 := rec("evt-mr-2", "frobnicate", "native", "none", "main", `{"mystery":true}`)
	l3 := rec("evt-mr-3", "reasoning/encrypted", "foreign", "encrypted", "main", `{"ciphertext":"SEALED"}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(l1, l2, l3)},
	})
	if len(bundle.result.DecodedEvents) != 3 {
		t.Fatalf("len(DecodedEvents) = %d, want 3", len(bundle.result.DecodedEvents))
	}
	want := []struct {
		id     string
		line   string
		kind   string
		offset uint64
	}{
		{"evt-mr-1", l1, "user_message", 0},
		{"evt-mr-2", l2, "opaque_event", uint64(len(l1)) + 1},
		{"evt-mr-3", l3, "opaque_reasoning", uint64(len(l1)) + uint64(len(l2)) + 2},
	}
	for _, row := range want {
		event := eventByNativeID(t, bundle.result, row.id)
		if event.Kind != row.kind {
			t.Fatalf("event %q Kind = %q, want %q", row.id, event.Kind, row.kind)
		}
		if len(event.Evidence.RawRefs) != 1 {
			t.Fatalf("event %q len(RawRefs) = %d, want 1", row.id, len(event.Evidence.RawRefs))
		}
		ref := event.Evidence.RawRefs[0]
		if ref.Offset != row.offset {
			t.Fatalf("event %q raw offset = %d, want %d", row.id, ref.Offset, row.offset)
		}
		if ref.Length != uint64(len(row.line)) {
			t.Fatalf("event %q raw length = %d, want %d", row.id, ref.Length, len(row.line))
		}
		resolved := mustResolveRef(t, bundle.capture, bundle.store, ref)
		if string(resolved) != row.line {
			t.Fatalf("event %q resolved bytes differ from its own record line", row.id)
		}
	}
}

func TestNormalizeUnterminatedTailResolvesWithOffset(t *testing.T) {
	// A multi-line member without a trailing newline keeps its
	// last record: the unterminated tail is a record, never a
	// dropped suffix. The last event exists with its offset
	// asserted and its bytes resolving to its own line.
	l1 := rec("evt-ut-1", "message/user", "native", "none", "main", `{"text":"first"}`)
	l2 := rec("evt-ut-2", "frobnicate", "native", "none", "main", `{"mystery":true}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: []byte(l1 + "\n" + l2)},
	})
	if len(bundle.result.DecodedEvents) != 2 {
		t.Fatalf("len(DecodedEvents) = %d, want 2: the unterminated tail was dropped", len(bundle.result.DecodedEvents))
	}
	want := []struct {
		id     string
		line   string
		offset uint64
	}{
		{"evt-ut-1", l1, 0},
		{"evt-ut-2", l2, uint64(len(l1)) + 1},
	}
	for _, row := range want {
		event := eventByNativeID(t, bundle.result, row.id)
		if len(event.Evidence.RawRefs) != 1 {
			t.Fatalf("event %q len(RawRefs) = %d, want 1", row.id, len(event.Evidence.RawRefs))
		}
		ref := event.Evidence.RawRefs[0]
		if ref.Offset != row.offset {
			t.Fatalf("event %q raw offset = %d, want %d", row.id, ref.Offset, row.offset)
		}
		if ref.Length != uint64(len(row.line)) {
			t.Fatalf("event %q raw length = %d, want %d", row.id, ref.Length, len(row.line))
		}
		resolved := mustResolveRef(t, bundle.capture, bundle.store, ref)
		if string(resolved) != row.line {
			t.Fatalf("event %q resolved bytes differ from its own record line", row.id)
		}
	}
}

func TestNormalizeCRLFResolvesWithCarriageReturn(t *testing.T) {
	// CRLF members are admitted with the trailing CR inside the
	// record range: each range covers its line plus the CR and
	// resolves exactly.
	l1 := rec("evt-cr-1", "message/user", "native", "none", "main", `{"text":"first"}`)
	l2 := rec("evt-cr-2", "message/user", "native", "none", "main", `{"text":"second"}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: []byte(l1 + "\r\n" + l2 + "\r\n")},
	})
	if len(bundle.result.DecodedEvents) != 2 {
		t.Fatalf("len(DecodedEvents) = %d, want 2", len(bundle.result.DecodedEvents))
	}
	want := []struct {
		id     string
		line   string
		offset uint64
	}{
		{"evt-cr-1", l1, 0},
		{"evt-cr-2", l2, uint64(len(l1)) + 2},
	}
	for _, row := range want {
		event := eventByNativeID(t, bundle.result, row.id)
		if len(event.Evidence.RawRefs) != 1 {
			t.Fatalf("event %q len(RawRefs) = %d, want 1", row.id, len(event.Evidence.RawRefs))
		}
		ref := event.Evidence.RawRefs[0]
		if ref.Offset != row.offset {
			t.Fatalf("event %q raw offset = %d, want %d", row.id, ref.Offset, row.offset)
		}
		if ref.Length != uint64(len(row.line))+1 {
			t.Fatalf("event %q raw length = %d, want %d (line plus CR)", row.id, ref.Length, len(row.line)+1)
		}
		resolved := mustResolveRef(t, bundle.capture, bundle.store, ref)
		if string(resolved) != row.line+"\r" {
			t.Fatalf("event %q resolved bytes differ from its line plus CR", row.id)
		}
	}
}

func TestNormalizeUnknownTypeNullBodyProjectsOpaque(t *testing.T) {
	// Present-but-null satisfies the required-member rule for an
	// unknown type: the body is never read, so the record
	// projects opaque. A MISSING body refuses instead
	// (unknown_type_without_body).
	line := `{"v":1,"native_event_id":"evt-nullbody","native_type":"frobnicate","origin":"native","protection":"none","actor":"main","body":null}`
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(line)},
	})
	if len(bundle.result.DecodedEvents) != 1 {
		t.Fatalf("len(DecodedEvents) = %d, want 1", len(bundle.result.DecodedEvents))
	}
	event := bundle.result.DecodedEvents[0]
	if event.Kind != "opaque_event" {
		t.Fatalf("Kind = %q, want opaque_event", event.Kind)
	}
	if len(event.Evidence.ReasonCodes) != 1 || event.Evidence.ReasonCodes[0] != "unknown_native_event" {
		t.Fatalf("ReasonCodes = %q, want [unknown_native_event]", event.Evidence.ReasonCodes)
	}
	resolved := mustResolveRef(t, bundle.capture, bundle.store, event.Evidence.RawRefs[0])
	if string(resolved) != line {
		t.Fatal("resolved bytes differ from the exact record line")
	}
}

func TestNormalizeOpaqueIsStableAcrossProjections(t *testing.T) {
	lines := []string{
		rec("evt-opaque-a", "frobnicate", "native", "none", "main", `{"n":1}`),
		rec("evt-opaque-b", "message/smoke", "native", "none", "main", `{"text":"x"}`),
	}
	members := []fixtureMember{
		{key: "store/mystery", class: "unknown", content: []byte("mystery-bytes")},
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(lines...)},
	}
	capture, store := captureMembers(t, members)
	first, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	second, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if len(first.Events) != 3 || len(second.Events) != 3 {
		t.Fatalf("event counts = %d/%d, want 3/3: a second projection dropped or split", len(first.Events), len(second.Events))
	}
	for index := range first.Events {
		if !bytes.Equal(first.Events[index], second.Events[index]) {
			t.Fatalf("event[%d] differs across projections: re-typed on the second pass", index)
		}
		if first.DecodedEvents[index].Kind != "opaque_event" {
			t.Fatalf("event[%d].Kind = %q, want opaque_event", index, first.DecodedEvents[index].Kind)
		}
		if second.DecodedEvents[index].Kind != "opaque_event" {
			t.Fatalf("event[%d].Kind = %q, want opaque_event", index, second.DecodedEvents[index].Kind)
		}
	}
}

func TestNormalizeSealsSessionShape(t *testing.T) {
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-s-1", "message/user", "native", "none", "main", `{"text":"hello"}`),
			rec("evt-s-2", "message/assistant", "native", "none", "main", `{"text":"world"}`),
		)},
	})
	result := bundle.result
	session := result.DecodedSession
	if session.LogicalSessionID.String() != fixtureLogicalID {
		t.Fatalf("LogicalSessionID = %q, want %q", session.LogicalSessionID.String(), fixtureLogicalID)
	}
	if len(session.Actors) != 1 {
		t.Fatalf("len(Actors) = %d, want 1", len(session.Actors))
	}
	if session.Actors[0].Kind != "main" {
		t.Fatalf("Actors[0].Kind = %q, want main", session.Actors[0].Kind)
	}
	if session.Actors[0].ActorID.String() != fixtureMainActor {
		t.Fatalf("Actors[0] = %q, want %q", session.Actors[0].ActorID.String(), fixtureMainActor)
	}
	if session.Actors[0].ParentActorID != nil {
		t.Fatalf("Actors[0].ParentActorID = %v, want null for main", session.Actors[0].ParentActorID)
	}
	if len(session.EventIDs) != 2 {
		t.Fatalf("len(EventIDs) = %d, want 2", len(session.EventIDs))
	}
	for index, decoded := range result.DecodedEvents {
		if session.EventIDs[index].String() != decoded.EventID.String() {
			t.Fatalf("EventIDs[%d] = %q, want the ordinal event %q", index, session.EventIDs[index], decoded.EventID)
		}
		if decoded.Ordinal != uint64(index) {
			t.Fatalf("Ordinal = %d, want %d", decoded.Ordinal, index)
		}
	}
	if len(session.HeadEventIDs) != 1 || session.HeadEventIDs[0].String() != session.EventIDs[1].String() {
		t.Fatalf("HeadEventIDs = %v, want the last event only", session.HeadEventIDs)
	}
	if len(result.DecodedEvents[0].Parents) != 0 {
		t.Fatalf("event[0] parents = %d, want none", len(result.DecodedEvents[0].Parents))
	}
	if len(result.DecodedEvents[1].Parents) != 1 ||
		result.DecodedEvents[1].Parents[0].String() != result.DecodedEvents[0].EventID.String() {
		t.Fatal("event[1] is not parented on event[0]")
	}
	if result.DecodedEvents[0].Kind != "user_message" || result.DecodedEvents[1].Kind != "assistant_message" {
		t.Fatalf("kinds = %q/%q, want user_message/assistant_message",
			result.DecodedEvents[0].Kind, result.DecodedEvents[1].Kind)
	}
	if result.DecodedEvents[0].Visibility != "public" || result.DecodedEvents[1].Visibility != "public" {
		t.Fatal("message visibility is not public")
	}
}

func TestNormalizeInlineContentBoundary(t *testing.T) {
	inline := strings.Repeat("i", 65536)
	overflow := strings.Repeat("o", 65537)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-inline", "message/user", "native", "none", "main", `{"text":"`+inline+`"}`),
			rec("evt-overflow", "message/user", "native", "none", "main", `{"text":"`+overflow+`"}`),
			rec("evt-empty", "message/user", "native", "none", "main", `{"text":""}`),
		)},
	})
	result := bundle.result
	sealed := map[string][]byte{}
	for index, decoded := range result.DecodedEvents {
		sealed[*decoded.Evidence.NativeEventID] = result.Events[index]
	}
	inlinePayload := sealedPayload(t, sealed["evt-inline"])
	blocks, ok := inlinePayload["content_blocks"].([]any)
	if !ok || len(blocks) != 1 {
		t.Fatalf("inline content_blocks = %v, want one block", inlinePayload["content_blocks"])
	}
	block := blocks[0].(map[string]any)
	if block["content"] != inline {
		t.Fatal("65536-byte text did not seal inline")
	}
	if _, present := block["blob_descriptor_id"]; present {
		t.Fatal("65536-byte text sealed a descriptor reference instead of inline content")
	}
	overflowPayload := sealedPayload(t, sealed["evt-overflow"])
	overBlocks, ok := overflowPayload["content_blocks"].([]any)
	if !ok || len(overBlocks) != 1 {
		t.Fatalf("overflow content_blocks = %v, want one block", overflowPayload["content_blocks"])
	}
	overBlock := overBlocks[0].(map[string]any)
	if _, present := overBlock["content"]; present {
		t.Fatal("65537-byte text sealed inline past the 64 KiB bound")
	}
	descriptor, ok := overBlock["blob_descriptor_id"].(string)
	if !ok || descriptor == "" {
		t.Fatalf("overflow block misses its blob_descriptor_id: %v", overBlock)
	}
	resolved := mustResolveOverflow(t, bundle.sink, descriptor)
	if string(resolved) != overflow {
		t.Fatal("overflow reference does not resolve to the exact 65537 bytes: truncated or re-encoded")
	}
	emptyPayload := sealedPayload(t, sealed["evt-empty"])
	emptyBlocks := emptyPayload["content_blocks"].([]any)
	if emptyBlocks[0].(map[string]any)["content"] != "" {
		t.Fatal("empty text did not seal inline")
	}
	if len(result.Overflows) != 1 {
		t.Fatalf("len(Overflows) = %d, want 1", len(result.Overflows))
	}
	if result.Overflows[0].DescriptorID.String() != descriptor {
		t.Fatal("reported overflow descriptor differs from the sealed reference")
	}
}

func TestNormalizeMultibyteStringMeasure(t *testing.T) {
	wide512 := strings.Repeat("\u00e9", 512)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec(wide512, "message/user", "native", "none", "main", `{"text":"wide"}`),
		)},
	})
	if len(bundle.result.DecodedEvents) != 1 {
		t.Fatalf("len(DecodedEvents) = %d, want 1: 512 characters measured as bytes refused", len(bundle.result.DecodedEvents))
	}
	wide513 := strings.Repeat("\u00e9", 513)
	capture, store := captureMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec(wide513, "message/user", "native", "none", "main", `{"text":"wide"}`),
		)},
	})
	_, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
	if err == nil {
		t.Fatal("Normalize() admitted a 513-character native event ID")
	}
	if !strings.Contains(err.Error(), "line 1 native_event_id") {
		t.Fatalf("Normalize() error = %v, want the record-level width refusal", err)
	}
}

func TestNormalizeActorOrderIsDeterministic(t *testing.T) {
	// Three extra actors in one session: map iteration order must
	// not leak into the sealed bytes. Twelve projections admit no
	// second distinct session.
	members := []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-ao-1", "message/user", "native", "none", "subagent:zeta", `{"text":"a"}`),
			rec("evt-ao-2", "message/user", "native", "none", "external", `{"text":"b"}`),
			rec("evt-ao-3", "message/user", "native", "none", "subagent:alpha", `{"text":"c"}`),
		)},
	}
	capture, store := captureMembers(t, members)
	distinct := map[string]bool{}
	for i := 0; i < 12; i++ {
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.Actors = map[string]string{
			"subagent:zeta":  fixtureSubActor,
			"external":       fixtureExtActor,
			"subagent:alpha": "0198f4d1-9e60-7aaa-83ca-3456789abcde",
		}
		result, err := Normalize(request)
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		distinct[string(result.Session)] = true
	}
	if len(distinct) != 1 {
		t.Fatalf("distinct sealed sessions over 12 projections = %d, want 1", len(distinct))
	}
}

func TestNormalizeIsDeterministic(t *testing.T) {
	members := []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-d-1", "message/user", "native", "none", "main", `{"text":"determinism"}`),
			rec("evt-d-2", "frobnicate", "native", "none", "main", `{"n":2}`),
			rec("evt-d-3", "usage/report", "native", "none", "main", `{"input_tokens":7,"output_tokens":9}`),
		)},
	}
	capture, store := captureMembers(t, members)
	first, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	second, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if !bytes.Equal(first.Session, second.Session) {
		t.Fatal("same captured bytes did not seal byte-identical sessions")
	}
	if len(first.Events) != len(second.Events) {
		t.Fatalf("event counts %d != %d", len(first.Events), len(second.Events))
	}
	for index := range first.Events {
		if !bytes.Equal(first.Events[index], second.Events[index]) {
			t.Fatalf("event[%d] differs across identical projections", index)
		}
	}
}

func TestNormalizeCaptureStatusExactForEveryKind(t *testing.T) {
	// Every emitted kind stamps capture_status exact: one record per
	// kind through Normalize, each event asserting the status and
	// its kind.
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-cs-user", "message/user", "native", "none", "main", `{"text":"u"}`),
			rec("evt-cs-assistant", "message/assistant", "native", "none", "main", `{"text":"a"}`),
			rec("evt-cs-summary", "reasoning/summary", "native", "none", "main", `{"text":"s"}`),
			rec("evt-cs-crypt", "reasoning/encrypted", "foreign", "encrypted", "main", `{"ciphertext":"c"}`),
			rec("evt-cs-unknown", "frobnicate", "native", "none", "main", `{"mystery":true}`),
			rec("evt-cs-inst", "instruction/snapshot", "native", "none", "main", `{"authority":"low","directives":["d"]}`),
			rec("evt-cs-def", "tool/definition", "native", "none", "main",
				`{"tool_name":"read","definition_digest":"`+fixtureDigest("cs-def")+`"}`),
			rec("evt-cs-call", "tool/call", "native", "none", "main", `{"call_id":"call-cs","tool_name":"read"}`),
			rec("evt-cs-result", "tool/result", "native", "none", "main", `{"call_id":"call-cs","status":"ok"}`),
			rec("evt-cs-usage", "usage/report", "native", "none", "main", `{"input_tokens":3,"output_tokens":4}`),
		)},
	})
	wantKinds := map[string]string{
		"evt-cs-user":      "user_message",
		"evt-cs-assistant": "assistant_message",
		"evt-cs-summary":   "reasoning_summary",
		"evt-cs-crypt":     "opaque_reasoning",
		"evt-cs-unknown":   "opaque_event",
		"evt-cs-inst":      "instruction_snapshot",
		"evt-cs-def":       "tool_definition_snapshot",
		"evt-cs-call":      "tool_call",
		"evt-cs-result":    "tool_result",
		"evt-cs-usage":     "usage",
	}
	if len(bundle.result.DecodedEvents) != len(wantKinds) {
		t.Fatalf("len(DecodedEvents) = %d, want %d", len(bundle.result.DecodedEvents), len(wantKinds))
	}
	for _, event := range bundle.result.DecodedEvents {
		if event.Evidence.NativeEventID == nil {
			t.Fatal("event carries no native event ID")
		}
		id := *event.Evidence.NativeEventID
		want, ok := wantKinds[id]
		if !ok {
			t.Fatalf("unexpected event %q", id)
		}
		if event.Kind != want {
			t.Fatalf("event %q Kind = %q, want %q", id, event.Kind, want)
		}
		if event.Evidence.CaptureStatus != "exact" {
			t.Fatalf("event %q CaptureStatus = %q, want exact", id, event.Evidence.CaptureStatus)
		}
	}
}

func TestNormalizeExternalActorSealsAsExternal(t *testing.T) {
	// The external selector seals as an external actor parented on
	// main, never as a subagent.
	members := []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-ext-1", "message/user", "native", "none", "external", `{"text":"x"}`),
		)},
	}
	capture, store := captureMembers(t, members)
	request := normalizeRequest(t, capture, store, openTestStore(t))
	request.Actors = map[string]string{"external": fixtureExtActor}
	result, err := Normalize(request)
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	event := eventByNativeID(t, result, "evt-ext-1")
	if event.ActorID.String() != fixtureExtActor {
		t.Fatalf("ActorID = %q, want the external actor %q", event.ActorID.String(), fixtureExtActor)
	}
	session := result.DecodedSession
	if len(session.Actors) != 2 {
		t.Fatalf("len(Actors) = %d, want main plus external", len(session.Actors))
	}
	external := session.Actors[1]
	if external.Kind != "external" {
		t.Fatalf("Actors[1].Kind = %q, want external", external.Kind)
	}
	if external.ParentActorID == nil || external.ParentActorID.String() != fixtureMainActor {
		t.Fatalf("external parent = %v, want main %q", external.ParentActorID, fixtureMainActor)
	}
}
