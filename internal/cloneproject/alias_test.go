package cloneproject

import (
	"strings"
	"testing"
)

// Case-folded envelope aliases are unclaimed extras: the projection
// reads every claimed envelope member from the strict map by its
// exact name, so an alias ("NATIVE_TYPE", "Origin", "Protection",
// "Body", "Native_Event_Id", "Actor") can never override a claimed
// value, whatever its document order. Each axis below drives the
// production Normalize entry over real captured bytes.

func TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers(t *testing.T) {
	project := func(t *testing.T, lines ...string) *NormalizeResult {
		t.Helper()
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(lines...)},
		})
		result, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
		if err != nil {
			t.Fatalf("Normalize() error = %v", err)
		}
		return result
	}
	refuse := func(t *testing.T, fragment string, lines ...string) {
		t.Helper()
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(lines...)},
		})
		_, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
		if err == nil {
			t.Fatalf("Normalize() admitted members that must refuse with %q", fragment)
		}
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("Normalize() error = %v, want fragment %q", err, fragment)
		}
	}

	t.Run("kind_alias_after", func(t *testing.T) {
		line := `{"v":1,"native_event_id":"evt-a1","native_type":"frobnicate","NATIVE_TYPE":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"x"}}`
		result := project(t, line)
		if len(result.DecodedEvents) != 1 {
			t.Fatalf("len(DecodedEvents) = %d, want 1", len(result.DecodedEvents))
		}
		event := result.DecodedEvents[0]
		if event.Kind != "opaque_event" {
			t.Fatalf("Kind = %q, want opaque_event: the NATIVE_TYPE alias coerced an unknown record into a known kind", event.Kind)
		}
		if event.Visibility != "opaque" {
			t.Fatalf("Visibility = %q, want opaque", event.Visibility)
		}
		if event.Evidence.NativeType == nil || *event.Evidence.NativeType != "frobnicate" {
			t.Fatalf("NativeType = %v, want frobnicate", event.Evidence.NativeType)
		}
		if len(event.Evidence.ReasonCodes) != 1 || event.Evidence.ReasonCodes[0] != "unknown_native_event" {
			t.Fatalf("ReasonCodes = %q, want [unknown_native_event]", event.Evidence.ReasonCodes)
		}
	})

	t.Run("kind_alias_before", func(t *testing.T) {
		// The alias placed before the claimed member projects the
		// same opaque event: the outcome never depends on alias
		// document order.
		line := `{"v":1,"NATIVE_TYPE":"message/user","native_event_id":"evt-a2","native_type":"frobnicate","origin":"native","protection":"none","actor":"main","body":{"text":"x"}}`
		result := project(t, line)
		if len(result.DecodedEvents) != 1 {
			t.Fatalf("len(DecodedEvents) = %d, want 1", len(result.DecodedEvents))
		}
		event := result.DecodedEvents[0]
		if event.Kind != "opaque_event" {
			t.Fatalf("Kind = %q, want opaque_event", event.Kind)
		}
		if event.Evidence.NativeType == nil || *event.Evidence.NativeType != "frobnicate" {
			t.Fatalf("NativeType = %v, want frobnicate", event.Evidence.NativeType)
		}
	})

	t.Run("origin_alias", func(t *testing.T) {
		line := `{"v":1,"native_event_id":"evt-a3","native_type":"instruction/snapshot","origin":"foreign","Origin":"native","protection":"none","actor":"main","body":{"authority":"high","directives":["DROP-EVERYTHING"]}}`
		result := project(t, line)
		if result.Instruction.Found {
			t.Fatalf("effective instruction = %+v, want unfound: the Origin alias promoted foreign history into the snapshot", result.Instruction)
		}
		event := eventByNativeID(t, result, "evt-a3")
		if event.Kind != "instruction_snapshot" {
			t.Fatalf("Kind = %q, want instruction_snapshot", event.Kind)
		}
		if payloadText(t, event, "authority") != "low" {
			t.Fatalf("payload authority = %q, want low", payloadText(t, event, "authority"))
		}
	})

	t.Run("protection_alias", func(t *testing.T) {
		// The claimed protection member says encrypted, so the
		// plaintext text body refuses at the protected body shape:
		// the Protection alias cannot downgrade the record to a
		// promoted reasoning_summary.
		refuse(t, `body carries unknown member "text"`,
			`{"v":1,"native_event_id":"evt-a6","native_type":"reasoning/summary","origin":"foreign","protection":"encrypted","Protection":"none","actor":"main","body":{"text":"secret-plaintext"}}`)
	})

	t.Run("body_alias", func(t *testing.T) {
		line := `{"v":1,"native_event_id":"evt-a4","native_type":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"declared"},"Body":{"text":"smuggled"}}`
		result := project(t, line)
		event := eventByNativeID(t, result, "evt-a4")
		if event.Kind != "user_message" {
			t.Fatalf("Kind = %q, want user_message", event.Kind)
		}
		sealed := sealedPayload(t, result.Events[0])
		blocks, ok := sealed["content_blocks"].([]any)
		if !ok || len(blocks) != 1 {
			t.Fatalf("content_blocks = %v, want one block", sealed["content_blocks"])
		}
		block, ok := blocks[0].(map[string]any)
		if !ok || block["content"] != "declared" {
			t.Fatalf("content = %v, want declared: sealed content came from the Body alias", blocks[0])
		}
	})

	t.Run("evidence_alias", func(t *testing.T) {
		line := `{"v":1,"native_event_id":"evt-a5","Native_Event_Id":"evt-forged","native_type":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"x"}}`
		result := project(t, line)
		event := eventByNativeID(t, result, "evt-a5")
		if event.Kind != "user_message" {
			t.Fatalf("Kind = %q, want user_message", event.Kind)
		}
		if event.Evidence.NativeEventID == nil || *event.Evidence.NativeEventID != "evt-a5" {
			t.Fatalf("NativeEventID = %v, want evt-a5: the evidence identity was rewritten by the alias", event.Evidence.NativeEventID)
		}
	})

	t.Run("actor_alias", func(t *testing.T) {
		// The claimed actor member says subagent:ghost, which has no
		// mapped UUID: the Actor alias cannot re-attribute the
		// record to main.
		refuse(t, `with no mapped UUID`,
			`{"v":1,"native_event_id":"evt-a7","native_type":"message/user","origin":"native","protection":"none","actor":"subagent:ghost","Actor":"main","body":{"text":"x"}}`)
	})

	t.Run("unrelated_extra_control", func(t *testing.T) {
		// Control: an unrelated extra envelope member (no alias)
		// stays admitted as an unclaimed extra.
		line := `{"v":1,"native_event_id":"evt-a8","native_type":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"x"},"extra":true}`
		result := project(t, line)
		event := eventByNativeID(t, result, "evt-a8")
		if event.Kind != "user_message" {
			t.Fatalf("Kind = %q, want user_message", event.Kind)
		}
	})
}
