package cloneproject

import (
	"testing"
)

// Invariant 3: historical tools are inert and incomplete calls never
// become live actions. The matrix drives every pairing shape through
// Normalize over real captured bytes and asserts the projected kind,
// the visibility, the folded resolution, and the empty live surface.

func TestNormalizeToolMatrix(t *testing.T) {
	call := func(id, tool string) string {
		return rec("evt-"+id, "tool/call", "native", "none", "main",
			`{"call_id":"`+id+`","tool_name":"`+tool+`"}`)
	}
	result := func(id, status string) string {
		return rec("evt-"+id+"-result", "tool/result", "native", "none", "main",
			`{"call_id":"`+id+`","status":"`+status+`"}`)
	}
	t.Run("call_with_result", func(t *testing.T) {
		bundle := projectMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				call("call-1", "read"),
				result("call-1", "ok"),
			)},
		})
		resolved := bundle.result
		if len(resolved.ToolResolutions) != 1 {
			t.Fatalf("len(ToolResolutions) = %d, want 1", len(resolved.ToolResolutions))
		}
		if resolved.ToolResolutions[0].Status != "completed" {
			t.Fatalf("Status = %q, want completed", resolved.ToolResolutions[0].Status)
		}
		callEvent := eventByNativeID(t, resolved, "evt-call-1")
		if callEvent.Kind != "tool_call" {
			t.Fatalf("Kind = %q, want tool_call", callEvent.Kind)
		}
		if callEvent.Visibility != "internal" {
			t.Fatalf("Visibility = %q, want internal", callEvent.Visibility)
		}
		if payloadText(t, callEvent, "resolution") != "completed" {
			t.Fatalf("payload resolution = %q, want completed", payloadText(t, callEvent, "resolution"))
		}
		if len(callEvent.Evidence.ReasonCodes) != 0 {
			t.Fatalf("ReasonCodes = %q, want empty for a completed call", callEvent.Evidence.ReasonCodes)
		}
		resultEvent := eventByNativeID(t, resolved, "evt-call-1-result")
		if resultEvent.Kind != "tool_result" {
			t.Fatalf("Kind = %q, want tool_result", resultEvent.Kind)
		}
		if payloadText(t, resultEvent, "resolution") != "completed" {
			t.Fatalf("payload resolution = %q, want completed", payloadText(t, resultEvent, "resolution"))
		}
		if len(resolved.Live.CallableTools) != 0 || len(resolved.Live.PendingActions) != 0 {
			t.Fatalf("Live = %+v, want empty", resolved.Live)
		}
	})
	t.Run("call_without_result", func(t *testing.T) {
		bundle := projectMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				call("call-7", "write"),
			)},
		})
		resolved := bundle.result
		if len(resolved.ToolResolutions) != 1 {
			t.Fatalf("len(ToolResolutions) = %d, want 1", len(resolved.ToolResolutions))
		}
		if resolved.ToolResolutions[0].Status != "aborted" {
			t.Fatalf("Status = %q, want aborted", resolved.ToolResolutions[0].Status)
		}
		callEvent := eventByNativeID(t, resolved, "evt-call-7")
		if callEvent.Kind != "tool_call" {
			t.Fatalf("Kind = %q, want tool_call", callEvent.Kind)
		}
		if callEvent.Visibility != "internal" {
			t.Fatalf("Visibility = %q, want internal", callEvent.Visibility)
		}
		if payloadText(t, callEvent, "resolution") != "aborted" {
			t.Fatalf("payload resolution = %q, want aborted", payloadText(t, callEvent, "resolution"))
		}
		if len(callEvent.Evidence.ReasonCodes) != 1 || callEvent.Evidence.ReasonCodes[0] != "unsafe_pending_action" {
			t.Fatalf("ReasonCodes = %q, want [unsafe_pending_action]", callEvent.Evidence.ReasonCodes)
		}
		if len(resolved.Live.CallableTools) != 0 || len(resolved.Live.PendingActions) != 0 {
			t.Fatalf("Live = %+v, want empty", resolved.Live)
		}
	})
	t.Run("result_without_call", func(t *testing.T) {
		bundle := projectMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				result("call-ghost", "ok"),
			)},
		})
		resolved := bundle.result
		if len(resolved.ToolResolutions) != 1 {
			t.Fatalf("len(ToolResolutions) = %d, want 1", len(resolved.ToolResolutions))
		}
		if resolved.ToolResolutions[0].Status != "aborted" {
			t.Fatalf("Status = %q, want aborted", resolved.ToolResolutions[0].Status)
		}
		orphan := eventByNativeID(t, resolved, "evt-call-ghost-result")
		if orphan.Kind != "tool_result" {
			t.Fatalf("Kind = %q, want tool_result", orphan.Kind)
		}
		if orphan.Visibility != "internal" {
			t.Fatalf("Visibility = %q, want internal", orphan.Visibility)
		}
		if payloadText(t, orphan, "resolution") != "aborted" {
			t.Fatalf("payload resolution = %q, want aborted", payloadText(t, orphan, "resolution"))
		}
		if len(orphan.Evidence.ReasonCodes) != 1 || orphan.Evidence.ReasonCodes[0] != "unsafe_pending_action" {
			t.Fatalf("ReasonCodes = %q, want [unsafe_pending_action]", orphan.Evidence.ReasonCodes)
		}
		if len(resolved.Live.CallableTools) != 0 || len(resolved.Live.PendingActions) != 0 {
			t.Fatalf("Live = %+v, want empty", resolved.Live)
		}
	})
	t.Run("nested_subagent_call", func(t *testing.T) {
		members := []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-nested-call", "tool/call", "native", "none", "subagent:fetch",
					`{"call_id":"call-nested","tool_name":"fetch"}`),
				rec("evt-nested-result", "tool/result", "native", "none", "subagent:fetch",
					`{"call_id":"call-nested","status":"ok"}`),
			)},
		}
		capture, store := captureMembers(t, members)
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.Actors = map[string]string{"subagent:fetch": fixtureSubActor}
		resolved, err := Normalize(request)
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		if len(resolved.ToolResolutions) != 1 || resolved.ToolResolutions[0].Status != "completed" {
			t.Fatalf("ToolResolutions = %+v, want one completed", resolved.ToolResolutions)
		}
		callEvent := eventByNativeID(t, resolved, "evt-nested-call")
		if callEvent.Kind != "tool_call" {
			t.Fatalf("Kind = %q, want tool_call", callEvent.Kind)
		}
		if callEvent.ActorID.String() != fixtureSubActor {
			t.Fatalf("ActorID = %q, want the subagent actor %q", callEvent.ActorID.String(), fixtureSubActor)
		}
		if callEvent.ActorID.String() == fixtureMainActor {
			t.Fatal("nested call was attributed to main instead of the subagent actor")
		}
		session := resolved.DecodedSession
		if len(session.Actors) != 2 {
			t.Fatalf("len(Actors) = %d, want main plus the subagent", len(session.Actors))
		}
		subagent := session.Actors[1]
		if subagent.Kind != "subagent" {
			t.Fatalf("Actors[1].Kind = %q, want subagent", subagent.Kind)
		}
		if subagent.ParentActorID == nil || subagent.ParentActorID.String() != fixtureMainActor {
			t.Fatalf("subagent parent = %v, want main %q", subagent.ParentActorID, fixtureMainActor)
		}
		if len(resolved.Live.CallableTools) != 0 || len(resolved.Live.PendingActions) != 0 {
			t.Fatalf("Live = %+v, want empty", resolved.Live)
		}
	})
}

func TestNormalizeResultAfterBoundaryCompletes(t *testing.T) {
	callLine := rec("evt-b-call", "tool/call", "native", "none", "main", `{"call_id":"call-b","tool_name":"write"}`)
	resultLine := rec("evt-b-result", "tool/result", "native", "none", "main", `{"call_id":"call-b","status":"ok"}`)
	// First capture: the call crossed the boundary, the result has
	// not arrived yet.
	early := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(callLine)},
	})
	if len(early.result.ToolResolutions) != 1 || early.result.ToolResolutions[0].Status != "aborted" {
		t.Fatalf("early ToolResolutions = %+v, want one aborted", early.result.ToolResolutions)
	}
	// Second capture over the later store: the result arrived, the
	// same call resolves completed.
	late := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(callLine, resultLine)},
	})
	if len(late.result.ToolResolutions) != 1 || late.result.ToolResolutions[0].Status != "completed" {
		t.Fatalf("late ToolResolutions = %+v, want one completed", late.result.ToolResolutions)
	}
	lateCall := eventByNativeID(t, late.result, "evt-b-call")
	if payloadText(t, lateCall, "resolution") != "completed" {
		t.Fatalf("late payload resolution = %q, want completed", payloadText(t, lateCall, "resolution"))
	}
	if len(late.result.Live.PendingActions) != 0 {
		t.Fatalf("late PendingActions = %q, want empty", late.result.Live.PendingActions)
	}
}

func TestNormalizeToolDefinitionNeverRegistersCallable(t *testing.T) {
	digest := fixtureDigest("definition-bytes")
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-def-1", "tool/definition", "native", "none", "main",
				`{"tool_name":"write","definition_digest":"`+digest+`"}`),
		)},
	})
	event := eventByNativeID(t, bundle.result, "evt-def-1")
	if event.Kind != "tool_definition_snapshot" {
		t.Fatalf("Kind = %q, want tool_definition_snapshot", event.Kind)
	}
	if event.Visibility != "internal" {
		t.Fatalf("Visibility = %q, want internal", event.Visibility)
	}
	if payloadBool(t, event, "callable") {
		t.Fatal("payload callable is true: history registered a callable tool")
	}
	if payloadText(t, event, "tool_name") != "write" {
		t.Fatalf("payload tool_name = %q, want write", payloadText(t, event, "tool_name"))
	}
	if len(bundle.result.Live.CallableTools) != 0 {
		t.Fatalf("CallableTools = %q, want empty", bundle.result.Live.CallableTools)
	}
}

func TestNormalizeNullOptionalBoolsDecodeAsFalse(t *testing.T) {
	// Stated bound: present-but-null optional booleans decode as
	// false, exactly like an absent member. Neither null claim
	// refuses nor promotes: the definition stays uncallable
	// history and the paired call completes.
	digest := fixtureDigest("null-bool-definition")
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-def-null", "tool/definition", "native", "none", "main",
				`{"tool_name":"write","definition_digest":"`+digest+`","live_attestation":null}`),
			rec("evt-call-null", "tool/call", "native", "none", "main", `{"call_id":"call-null","tool_name":"write"}`),
			rec("evt-call-null-result", "tool/result", "native", "none", "main",
				`{"call_id":"call-null","status":"ok","live_followup":null}`),
		)},
	})
	definition := eventByNativeID(t, bundle.result, "evt-def-null")
	if definition.Kind != "tool_definition_snapshot" {
		t.Fatalf("Kind = %q, want tool_definition_snapshot", definition.Kind)
	}
	if payloadBool(t, definition, "callable") {
		t.Fatal("payload callable is true: a null live claim promoted history")
	}
	callEvent := eventByNativeID(t, bundle.result, "evt-call-null")
	if payloadText(t, callEvent, "resolution") != "completed" {
		t.Fatalf("payload resolution = %q, want completed", payloadText(t, callEvent, "resolution"))
	}
	resultEvent := eventByNativeID(t, bundle.result, "evt-call-null-result")
	if payloadText(t, resultEvent, "resolution") != "completed" {
		t.Fatalf("payload resolution = %q, want completed", payloadText(t, resultEvent, "resolution"))
	}
	if len(bundle.result.Live.CallableTools) != 0 || len(bundle.result.Live.PendingActions) != 0 {
		t.Fatalf("Live = %+v, want empty", bundle.result.Live)
	}
}

func TestNormalizeProtectedToolCallStaysOpaqueHistory(t *testing.T) {
	// A protected tool/call never folds into pairing: it projects
	// opaque_event while its unprotected result becomes an aborted
	// orphan. The live surface stays empty.
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-prot-call", "tool/call", "foreign", "encrypted", "main", `{"ciphertext":"x"}`),
			rec("evt-prot-result", "tool/result", "native", "none", "main", `{"call_id":"c1","status":"ok"}`),
		)},
	})
	resolved := bundle.result
	callEvent := eventByNativeID(t, resolved, "evt-prot-call")
	if callEvent.Kind != "opaque_event" {
		t.Fatalf("Kind = %q, want opaque_event: protected call reached a semantic kind", callEvent.Kind)
	}
	if callEvent.Visibility != "opaque" {
		t.Fatalf("Visibility = %q, want opaque", callEvent.Visibility)
	}
	if len(resolved.ToolResolutions) != 1 || resolved.ToolResolutions[0].Status != "aborted" {
		t.Fatalf("ToolResolutions = %+v, want one aborted orphan", resolved.ToolResolutions)
	}
	orphan := eventByNativeID(t, resolved, "evt-prot-result")
	if orphan.Kind != "tool_result" {
		t.Fatalf("Kind = %q, want tool_result", orphan.Kind)
	}
	if payloadText(t, orphan, "resolution") != "aborted" {
		t.Fatalf("payload resolution = %q, want aborted", payloadText(t, orphan, "resolution"))
	}
	if len(orphan.Evidence.ReasonCodes) != 1 || orphan.Evidence.ReasonCodes[0] != "unsafe_pending_action" {
		t.Fatalf("ReasonCodes = %q, want [unsafe_pending_action]", orphan.Evidence.ReasonCodes)
	}
	if len(resolved.Live.CallableTools) != 0 || len(resolved.Live.PendingActions) != 0 {
		t.Fatalf("Live = %+v, want empty", resolved.Live)
	}
}

func TestNormalizeLiveSurfaceStaysEmpty(t *testing.T) {
	bundle := projectMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-ls-def", "tool/definition", "native", "none", "main",
				`{"tool_name":"write","definition_digest":"`+fixtureDigest("ls-def")+`"}`),
			rec("evt-ls-call", "tool/call", "native", "none", "main", `{"call_id":"call-ls","tool_name":"write"}`),
			rec("evt-ls-result", "tool/result", "native", "none", "main", `{"call_id":"call-ls","status":"ok"}`),
			rec("evt-ls-dangling", "tool/call", "native", "none", "main", `{"call_id":"call-dangling","tool_name":"read"}`),
			rec("evt-ls-orphan", "tool/result", "native", "none", "main", `{"call_id":"call-lost","status":"error"}`),
		)},
	})
	live := bundle.result.Live
	if len(live.CallableTools) != 0 {
		t.Fatalf("CallableTools = %q, want empty", live.CallableTools)
	}
	if len(live.PendingActions) != 0 {
		t.Fatalf("PendingActions = %q, want empty", live.PendingActions)
	}
	if len(bundle.result.ToolResolutions) != 3 {
		t.Fatalf("len(ToolResolutions) = %d, want 3 (completed, aborted, orphan)", len(bundle.result.ToolResolutions))
	}
}
