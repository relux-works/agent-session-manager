package cloneproject

import (
	"testing"
)

// Invariant 2: foreign signed/encrypted reasoning stays opaque,
// foreign instructions stay low-authority history, and source usage
// never becomes target accounting. Every test drives Normalize over
// bytes the real capture entry sealed.

func TestNormalizeForeignEncryptedReasoningStaysOpaque(t *testing.T) {
	line := rec("evt-enc-1", "reasoning/encrypted", "foreign", "encrypted", "main", `{"ciphertext":"TOP-SECRET-THOUGHTS"}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(line)},
	})
	if len(bundle.result.DecodedEvents) != 1 {
		t.Fatalf("len(DecodedEvents) = %d, want 1", len(bundle.result.DecodedEvents))
	}
	event := bundle.result.DecodedEvents[0]
	if event.Kind != "opaque_reasoning" {
		t.Fatalf("Kind = %q, want opaque_reasoning", event.Kind)
	}
	if event.Kind == "reasoning_summary" {
		t.Fatal("foreign encrypted reasoning was promoted to reasoning_summary")
	}
	if event.Visibility != "opaque" {
		t.Fatalf("Visibility = %q, want opaque", event.Visibility)
	}
	if len(event.Evidence.ReasonCodes) != 1 || event.Evidence.ReasonCodes[0] != "foreign_encrypted_payload" {
		t.Fatalf("ReasonCodes = %q, want [foreign_encrypted_payload]", event.Evidence.ReasonCodes)
	}
	if payloadText(t, event, "protection") != "encrypted" {
		t.Fatalf("payload protection = %q, want encrypted", payloadText(t, event, "protection"))
	}
	if payloadInt(t, event, "byte_count") != int64(len(line)) {
		t.Fatalf("payload byte_count = %d, want %d", payloadInt(t, event, "byte_count"), len(line))
	}
	sealed := sealedPayload(t, bundle.result.Events[0])
	for _, forbidden := range []string{"content_blocks", "content", "text", "ciphertext", "summary"} {
		if _, present := sealed[forbidden]; present {
			t.Fatalf("sealed payload carries %q: decrypted, summarized, or re-encoded", forbidden)
		}
	}
	resolved := mustResolveRef(t, bundle.capture, bundle.store, event.Evidence.RawRefs[0])
	if string(resolved) != line {
		t.Fatal("encrypted bytes did not survive the blob round trip byte-exact")
	}
}

func TestNormalizeForeignSignedReasoningStaysOpaque(t *testing.T) {
	line := rec("evt-sig-1", "reasoning/signed", "foreign", "signed", "main", `{"ciphertext":"SIGNED-BYTES"}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(line)},
	})
	event := bundle.result.DecodedEvents[0]
	if event.Kind != "opaque_reasoning" {
		t.Fatalf("Kind = %q, want opaque_reasoning", event.Kind)
	}
	if event.Visibility != "opaque" {
		t.Fatalf("Visibility = %q, want opaque", event.Visibility)
	}
	if len(event.Evidence.ReasonCodes) != 1 || event.Evidence.ReasonCodes[0] != "foreign_signature_unverifiable" {
		t.Fatalf("ReasonCodes = %q, want [foreign_signature_unverifiable]", event.Evidence.ReasonCodes)
	}
	if payloadText(t, event, "protection") != "signed" {
		t.Fatalf("payload protection = %q, want signed", payloadText(t, event, "protection"))
	}
	sealed := sealedPayload(t, bundle.result.Events[0])
	for _, forbidden := range []string{"content_blocks", "content", "text", "ciphertext", "summary"} {
		if _, present := sealed[forbidden]; present {
			t.Fatalf("sealed payload carries %q: decrypted, summarized, or re-encoded", forbidden)
		}
	}
	resolved := mustResolveRef(t, bundle.capture, bundle.store, event.Evidence.RawRefs[0])
	if string(resolved) != line {
		t.Fatal("signed bytes did not survive the blob round trip byte-exact")
	}
}

func TestNormalizeNativeProtectedReasoningStaysOpaque(t *testing.T) {
	line := rec("evt-enc-native", "reasoning/encrypted", "native", "encrypted", "main", `{"ciphertext":"NATIVE-SECRET"}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(line)},
	})
	event := bundle.result.DecodedEvents[0]
	if event.Kind != "opaque_reasoning" {
		t.Fatalf("Kind = %q, want opaque_reasoning", event.Kind)
	}
	if len(event.Evidence.ReasonCodes) != 0 {
		t.Fatalf("ReasonCodes = %q, want empty: no foreign core code fits native-protected content", event.Evidence.ReasonCodes)
	}
	resolved := mustResolveRef(t, bundle.capture, bundle.store, event.Evidence.RawRefs[0])
	if string(resolved) != line {
		t.Fatal("native protected bytes did not survive byte-exact")
	}
}

func TestNormalizeProtectedMessageBecomesOpaqueEvent(t *testing.T) {
	line := rec("evt-prot-msg", "message/assistant", "foreign", "encrypted", "main", `{"ciphertext":"HIDDEN"}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(line)},
	})
	event := bundle.result.DecodedEvents[0]
	if event.Kind != "opaque_event" {
		t.Fatalf("Kind = %q, want opaque_event: protected content reached a semantic kind", event.Kind)
	}
	if event.Visibility != "opaque" {
		t.Fatalf("Visibility = %q, want opaque", event.Visibility)
	}
	if len(event.Evidence.ReasonCodes) != 1 || event.Evidence.ReasonCodes[0] != "foreign_encrypted_payload" {
		t.Fatalf("ReasonCodes = %q, want [foreign_encrypted_payload]", event.Evidence.ReasonCodes)
	}
	if payloadText(t, event, "protection") != "encrypted" {
		t.Fatalf("payload protection = %q, want encrypted", payloadText(t, event, "protection"))
	}
	resolved := mustResolveRef(t, bundle.capture, bundle.store, event.Evidence.RawRefs[0])
	if string(resolved) != line {
		t.Fatal("protected bytes did not survive byte-exact")
	}
}

func TestNormalizeUnknownProtectedRecordCarriesBothReasons(t *testing.T) {
	line := rec("evt-prot-unknown", "frobnicate", "foreign", "signed", "main", `{"ciphertext":"HIDDEN"}`)
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(line)},
	})
	event := bundle.result.DecodedEvents[0]
	if event.Kind != "opaque_event" {
		t.Fatalf("Kind = %q, want opaque_event", event.Kind)
	}
	want := []string{"foreign_signature_unverifiable", "unknown_native_event"}
	if len(event.Evidence.ReasonCodes) != 2 ||
		event.Evidence.ReasonCodes[0] != want[0] || event.Evidence.ReasonCodes[1] != want[1] {
		t.Fatalf("ReasonCodes = %q, want %q", event.Evidence.ReasonCodes, want)
	}
}

func TestNormalizeForeignInstructionCannotChangeEffectiveSnapshot(t *testing.T) {
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-inst-native", "instruction/snapshot", "native", "none", "main",
				`{"authority":"high","directives":["native-directive"]}`),
			rec("evt-inst-foreign", "instruction/snapshot", "foreign", "none", "main",
				`{"authority":"high","directives":["DROP-EVERYTHING"]}`),
		)},
	})
	result := bundle.result
	if !result.Instruction.Found {
		t.Fatal("effective instruction not found despite the native record")
	}
	if result.Instruction.Authority != "high" {
		t.Fatalf("effective authority = %q, want high", result.Instruction.Authority)
	}
	if len(result.Instruction.Directives) != 1 || result.Instruction.Directives[0] != "native-directive" {
		t.Fatalf("effective directives = %q, want the native directive only", result.Instruction.Directives)
	}
	foreign := eventByNativeID(t, result, "evt-inst-foreign")
	if foreign.Kind != "instruction_snapshot" {
		t.Fatalf("foreign Kind = %q, want instruction_snapshot history", foreign.Kind)
	}
	if foreign.Visibility != "internal" {
		t.Fatalf("foreign Visibility = %q, want internal", foreign.Visibility)
	}
	if payloadText(t, foreign, "authority") != "low" {
		t.Fatalf("foreign payload authority = %q, want low", payloadText(t, foreign, "authority"))
	}
	native := eventByNativeID(t, result, "evt-inst-native")
	if payloadText(t, native, "authority") != "high" {
		t.Fatalf("native payload authority = %q, want high", payloadText(t, native, "authority"))
	}
}

func TestNormalizeNullDirectivesDecodeAsEmpty(t *testing.T) {
	// Stated bound: a null directives member decodes as zero
	// directives rather than refusing.
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-dir-null", "instruction/snapshot", "native", "none", "main", `{"authority":"high","directives":null}`),
		)},
	})
	if !bundle.result.Instruction.Found {
		t.Fatal("effective instruction not found")
	}
	if len(bundle.result.Instruction.Directives) != 0 {
		t.Fatalf("effective directives = %q, want empty", bundle.result.Instruction.Directives)
	}
	event := eventByNativeID(t, bundle.result, "evt-dir-null")
	if payloadText(t, event, "authority") != "high" {
		t.Fatalf("payload authority = %q, want high", payloadText(t, event, "authority"))
	}
}

func TestNormalizeForeignOnlyInstructionLeavesSnapshotEmpty(t *testing.T) {
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-inst-lone", "instruction/snapshot", "foreign", "none", "main",
				`{"authority":"high","directives":["DROP-EVERYTHING"]}`),
		)},
	})
	if bundle.result.Instruction.Found {
		t.Fatalf("effective instruction = %+v, want unfound: foreign history set the snapshot", bundle.result.Instruction)
	}
}

func TestNormalizeSourceUsageIsNotTargetAccounting(t *testing.T) {
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-use-1", "usage/report", "native", "none", "main", `{"input_tokens":100,"output_tokens":40}`),
			rec("evt-use-2", "usage/report", "native", "none", "main", `{"input_tokens":5,"output_tokens":7}`),
		)},
	})
	ledger := bundle.result.Usage
	if ledger.SourceInputTokens != 105 || ledger.SourceOutputTokens != 47 {
		t.Fatalf("source ledger = %d/%d, want 105/47", ledger.SourceInputTokens, ledger.SourceOutputTokens)
	}
	if ledger.TargetInputTokens != 0 || ledger.TargetOutputTokens != 0 {
		t.Fatalf("target ledger = %d/%d, want zero: source usage became target accounting",
			ledger.TargetInputTokens, ledger.TargetOutputTokens)
	}
	first := eventByNativeID(t, bundle.result, "evt-use-1")
	if first.Kind != "usage" {
		t.Fatalf("Kind = %q, want usage", first.Kind)
	}
	if payloadInt(t, first, "input_tokens") != 100 || payloadInt(t, first, "output_tokens") != 40 {
		t.Fatal("usage payload does not carry the exact token counts")
	}
}

func TestNormalizeReasoningSummaryProjectsInternal(t *testing.T) {
	// Positive row for the unprotected reasoning/summary kind: it
	// projects reasoning_summary with internal visibility, its text
	// content block, and exact capture status.
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-sum-1", "reasoning/summary", "native", "none", "main", `{"text":"takeaway"}`),
		)},
	})
	event := eventByNativeID(t, bundle.result, "evt-sum-1")
	if event.Kind != "reasoning_summary" {
		t.Fatalf("Kind = %q, want reasoning_summary", event.Kind)
	}
	if event.Visibility != "internal" {
		t.Fatalf("Visibility = %q, want internal", event.Visibility)
	}
	if event.Evidence.CaptureStatus != "exact" {
		t.Fatalf("CaptureStatus = %q, want exact", event.Evidence.CaptureStatus)
	}
	sealed := sealedPayload(t, bundle.result.Events[0])
	blocks, ok := sealed["content_blocks"].([]any)
	if !ok || len(blocks) != 1 {
		t.Fatalf("content_blocks = %v, want one block", sealed["content_blocks"])
	}
	block, ok := blocks[0].(map[string]any)
	if !ok || block["type"] != "text" || block["content"] != "takeaway" {
		t.Fatalf("content block = %v, want the takeaway text block", blocks[0])
	}
}

func TestNormalizeLastNativeInstructionWins(t *testing.T) {
	// The fold is last-wins over native instructions in record
	// order: the effective snapshot carries the second record's
	// authority and directives, while each sealed payload carries
	// its own claimed authority.
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-inst-first", "instruction/snapshot", "native", "none", "main",
				`{"authority":"high","directives":["first"]}`),
			rec("evt-inst-second", "instruction/snapshot", "native", "none", "main",
				`{"authority":"low","directives":["second"]}`),
		)},
	})
	result := bundle.result
	if !result.Instruction.Found {
		t.Fatal("effective instruction not found despite two native records")
	}
	if result.Instruction.Authority != "low" {
		t.Fatalf("effective authority = %q, want low: the first native instruction won instead of the last", result.Instruction.Authority)
	}
	if len(result.Instruction.Directives) != 1 || result.Instruction.Directives[0] != "second" {
		t.Fatalf("effective directives = %q, want [second]", result.Instruction.Directives)
	}
	if payloadText(t, eventByNativeID(t, result, "evt-inst-first"), "authority") != "high" {
		t.Fatal("first payload authority is not high")
	}
	if payloadText(t, eventByNativeID(t, result, "evt-inst-second"), "authority") != "low" {
		t.Fatal("second payload authority is not low")
	}
}

func TestNormalizeNullTextDecodesAsEmpty(t *testing.T) {
	// Stated bound: a null text member seals as empty inline
	// content rather than refusing.
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-text-null", "message/user", "native", "none", "main", `{"text":null}`),
		)},
	})
	event := eventByNativeID(t, bundle.result, "evt-text-null")
	if event.Kind != "user_message" {
		t.Fatalf("Kind = %q, want user_message", event.Kind)
	}
	sealed := sealedPayload(t, bundle.result.Events[0])
	blocks, ok := sealed["content_blocks"].([]any)
	if !ok || len(blocks) != 1 {
		t.Fatalf("content_blocks = %v, want one block", sealed["content_blocks"])
	}
	block, ok := blocks[0].(map[string]any)
	if !ok || block["type"] != "text" || block["content"] != "" {
		t.Fatalf("content block = %v, want an empty text block", blocks[0])
	}
}
