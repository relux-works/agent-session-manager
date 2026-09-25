package cloneplanning

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
)

// This file drives ClassifyItem through the production entry over
// real canonical events sealed by the landed
// clonebundle.BuildCanonicalEvent entry. Classification is by
// closed vocabularies only: the sweep proves adversarial payload
// text cannot change the item, and refusals prove unknown fact
// values never interpret.

// classifyPayload renders one non-message payload carrying the
// given extra facts.
func classifyPayload(facts string) string {
	if facts == "" {
		return `{"extensions":{}}`
	}
	return `{"extensions":{},` + facts + `}`
}

func TestClassifyItemValidGrid(t *testing.T) {
	cases := []struct {
		kind       string
		payload    string
		visibility string
		want       Item
	}{
		{"tool_call", classifyPayload(`"call_id":"c1","tool_name":"t","resolution":"completed"`), "internal", Item{Kind: "tool_call", Resolution: "completed"}},
		{"tool_call", classifyPayload(`"call_id":"c1","tool_name":"t","resolution":"aborted"`), "internal", Item{Kind: "tool_call", Resolution: "aborted"}},
		{"tool_result", classifyPayload(`"call_id":"c1","status":"ok","resolution":"completed"`), "internal", Item{Kind: "tool_result", Resolution: "completed"}},
		{"tool_result", classifyPayload(`"call_id":"c1","status":"error","resolution":"aborted"`), "internal", Item{Kind: "tool_result", Resolution: "aborted"}},
		{"opaque_reasoning", classifyPayload(`"protection":"encrypted","byte_count":9`), "opaque", Item{Kind: "opaque_reasoning", Protection: "encrypted"}},
		{"opaque_reasoning", classifyPayload(`"protection":"signed","byte_count":9`), "opaque", Item{Kind: "opaque_reasoning", Protection: "signed"}},
		{"opaque_event", classifyPayload(`"native_type":"x.y","byte_count":9`), "opaque", Item{Kind: "opaque_event"}},
		{"opaque_event", classifyPayload(`"native_type":"x.y","protection":"encrypted","byte_count":9`), "opaque", Item{Kind: "opaque_event", Protection: "encrypted"}},
		{"opaque_event", classifyPayload(`"native_type":"x.y","protection":"signed","byte_count":9`), "opaque", Item{Kind: "opaque_event", Protection: "signed"}},
		{"user_message", messagePayload("hello"), "public", Item{Kind: "user_message"}},
		{"assistant_message", messagePayload("done"), "public", Item{Kind: "assistant_message"}},
		{"instruction_snapshot", classifyPayload(`"authority":"high","directives":["a"]`), "internal", Item{Kind: "instruction_snapshot"}},
		{"usage", classifyPayload(`"input_tokens":5,"output_tokens":7`), "internal", Item{Kind: "usage"}},
	}
	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.kind] = true
		event := mustBuildEvent(t, tc.kind, tc.visibility, tc.payload, 0)
		got, err := ClassifyItem(event)
		if err != nil {
			t.Fatalf("ClassifyItem(%s) refused: %v", tc.kind, err)
		}
		if got != tc.want {
			t.Fatalf("ClassifyItem(%s) = %+v, want %+v", tc.kind, got, tc.want)
		}
	}
	// Every remaining kind classifies from a minimal payload.
	for _, kind := range clonebundle.EventKinds() {
		if covered[kind] {
			continue
		}
		event := mustBuildEvent(t, kind, "internal", classifyPayload(""), 0)
		got, err := ClassifyItem(event)
		if err != nil {
			t.Fatalf("ClassifyItem(%s) refused: %v", kind, err)
		}
		if got != (Item{Kind: kind}) {
			t.Fatalf("ClassifyItem(%s) = %+v, want kind-only", kind, got)
		}
	}
}

func TestClassifyItemRefusals(t *testing.T) {
	// Unknown kinds refuse. The builder would refuse these too,
	// so they ride hand-built structs.
	for _, kind := range []string{"", "bogus_kind", "USER_MESSAGE", "tool_call\xff"} {
		if _, err := ClassifyItem(clonebundle.CanonicalEvent{Kind: kind}); err == nil {
			t.Fatalf("ClassifyItem kind %q admitted", kind)
		}
	}
	// Unknown or mistyped fact values refuse, never interpret.
	// Probes "bogus_resolution", "bogus_protection", and the
	// float64 fact are the narrowing probes
	// (N-resolution-vocab, N-protection-vocab, N-classify-fact-type).
	badPayloads := []struct {
		name    string
		kind    string
		payload string
	}{
		{"bad-resolution", "tool_call", classifyPayload(`"call_id":"c","tool_name":"t","resolution":"bogus_resolution"`)},
		{"bad-protection", "opaque_reasoning", classifyPayload(`"protection":"bogus_protection"`)},
		{"numeric-resolution", "tool_call", classifyPayload(`"call_id":"c","tool_name":"t","resolution":1`)},
		{"numeric-resolution-plain", "usage", classifyPayload(`"input_tokens":1,"resolution":1`)},
		{"bool-protection", "opaque_reasoning", classifyPayload(`"protection":true`)},
		{"object-resolution", "tool_call", classifyPayload(`"call_id":"c","tool_name":"t","resolution":{}`)},
		{"missing-resolution", "tool_call", classifyPayload(`"call_id":"c","tool_name":"t"`)},
		{"missing-protection", "opaque_reasoning", classifyPayload(`"byte_count":1`)},
		{"resolution-on-plain", "usage", classifyPayload(`"input_tokens":1,"resolution":"completed"`)},
		{"protection-on-plain", "usage", classifyPayload(`"input_tokens":1,"protection":"encrypted"`)},
		{"null-resolution-tool", "tool_call", classifyPayload(`"call_id":"c","tool_name":"t","resolution":null`)},
	}
	for _, tc := range badPayloads {
		event := mustBuildEvent(t, tc.kind, "internal", tc.payload, 0)
		if _, err := ClassifyItem(event); err == nil {
			t.Fatalf("ClassifyItem %s admitted", tc.name)
		}
	}
	// Present-but-null on a kind that forbids the fact maps to
	// absent (the cloneproject null precedent) and admits.
	event := mustBuildEvent(t, "usage", "internal", classifyPayload(`"input_tokens":1,"resolution":null,"protection":null`), 0)
	got, err := ClassifyItem(event)
	if err != nil {
		t.Fatalf("ClassifyItem null facts refused: %v", err)
	}
	if got != (Item{Kind: "usage"}) {
		t.Fatalf("ClassifyItem null facts = %+v, want kind-only", got)
	}
	// Every adversarial text class as a fact value refuses: text
	// never decides classification.
	for _, class := range corpusClasses() {
		for _, text := range class[1].([]string) {
			quoted, err := EscapeVisibleText(text)
			if err != nil {
				t.Fatal(err)
			}
			event := mustBuildEvent(t, "tool_call", "internal", classifyPayload(`"call_id":"c","tool_name":"t","resolution":`+quoted), 0)
			if _, err := ClassifyItem(event); err == nil {
				t.Fatalf("ClassifyItem %s resolution %q admitted", class[0], text)
			}
		}
	}
}

func TestClassifyItemIgnoresPayloadText(t *testing.T) {
	// The same kind and closed facts with benign versus
	// adversarial other-facts classify to identical items: text
	// content never flows into the item (the item has no text
	// field by type). Sweep representative carriers over every
	// corpus class, every member, plus the generated corpus.
	carriers := []struct {
		kind       string
		payload    func(text, quoted string) string
		visibility string
		want       Item
	}{
		{"tool_call", func(text, quoted string) string {
			return classifyPayload(`"call_id":"c","tool_name":` + quoted + `,"resolution":"completed"`)
		}, "internal", Item{Kind: "tool_call", Resolution: "completed"}},
		{"user_message", func(text, quoted string) string {
			_ = text
			return messagePayload(text)
		}, "public", Item{Kind: "user_message"}},
		{"instruction_snapshot", func(text, quoted string) string {
			return classifyPayload(`"authority":"high","directives":[` + quoted + `]`)
		}, "internal", Item{Kind: "instruction_snapshot"}},
		{"usage", func(text, quoted string) string {
			_ = quoted
			return classifyPayload(`"input_tokens":5,"output_tokens":7`)
		}, "internal", Item{Kind: "usage"}},
		{"opaque_reasoning", func(text, quoted string) string {
			_ = quoted
			return classifyPayload(`"protection":"encrypted","byte_count":9`)
		}, "opaque", Item{Kind: "opaque_reasoning", Protection: "encrypted"}},
	}
	texts := []string{}
	for _, class := range corpusClasses() {
		texts = append(texts, class[1].([]string)...)
	}
	texts = append(texts, generatedCorpus(50)...)
	for _, carrier := range carriers {
		for _, text := range texts {
			quoted, err := EscapeVisibleText(text)
			if err != nil {
				t.Fatal(err)
			}
			event := mustBuildEvent(t, carrier.kind, carrier.visibility, carrier.payload(text, quoted), 0)
			got, err := ClassifyItem(event)
			if err != nil {
				t.Fatalf("ClassifyItem(%s) with payload text %q refused: %v", carrier.kind, text, err)
			}
			if got != carrier.want {
				t.Fatalf("ClassifyItem(%s) with payload text %q = %+v, want %+v", carrier.kind, text, got, carrier.want)
			}
		}
	}
}

func TestClassifyItemFactNamePin(t *testing.T) {
	// Composition pin: the payload fact names and protection
	// spellings ClassifyItem reads follow cloneproject's landed
	// registry. A rename there breaks this test loudly instead
	// of silently decoupling the bridge. The completed/aborted
	// spellings reference cloneproject's exported statuses, so a
	// rename there breaks the build.
	dispatch, err := os.ReadFile(filepath.Join("..", "cloneproject", "dispatch.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"resolution"`, `"protection"`} {
		if !strings.Contains(string(dispatch), key) {
			t.Fatalf("cloneproject/dispatch.go no longer carries fact name %s", key)
		}
	}
	native, err := os.ReadFile(filepath.Join("..", "cloneproject", "native.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(native), `!= "encrypted"`) || !strings.Contains(string(native), `!= "signed"`) {
		t.Fatalf("cloneproject/native.go no longer pins the encrypted|signed protection vocabulary")
	}
}

// TestSharedItemVocabBothEntries drives the shared checkItem
// vocabulary gates through both production entries (PlanItem and
// ClassifyItem): one implementation, both call sites, one killer
// per narrowing row.
func TestSharedItemVocabBothEntries(t *testing.T) {
	// Kind vocabulary through both entries.
	if _, err := PlanItem("target_native_writer", "maximal_safe", Item{Kind: "bogus_kind"}); err == nil {
		t.Fatalf("PlanItem bogus_kind admitted")
	}
	if _, err := ClassifyItem(clonebundle.CanonicalEvent{Kind: "bogus_kind"}); err == nil {
		t.Fatalf("ClassifyItem bogus_kind admitted")
	}
	// Block vocabulary through PlanItem (blocks have no event
	// bridge: per-block items are built by the caller).
	if _, err := PlanItem("target_native_writer", "maximal_safe", Item{Block: "bogus_block"}); err == nil {
		t.Fatalf("PlanItem bogus_block admitted")
	}
	// Resolution vocabulary through both entries.
	if _, err := PlanItem("target_native_writer", "maximal_safe", Item{Kind: "tool_call", Resolution: "bogus_resolution"}); err == nil {
		t.Fatalf("PlanItem bogus_resolution admitted")
	}
	event := mustBuildEvent(t, "tool_call", "internal", classifyPayload(`"call_id":"c","tool_name":"t","resolution":"bogus_resolution"`), 0)
	if _, err := ClassifyItem(event); err == nil {
		t.Fatalf("ClassifyItem bogus_resolution admitted")
	}
	// Protection vocabulary through both entries.
	if _, err := PlanItem("target_native_writer", "maximal_safe", Item{Kind: "opaque_reasoning", Protection: "bogus_protection"}); err == nil {
		t.Fatalf("PlanItem bogus_protection admitted")
	}
	event = mustBuildEvent(t, "opaque_reasoning", "opaque", classifyPayload(`"protection":"bogus_protection"`), 0)
	if _, err := ClassifyItem(event); err == nil {
		t.Fatalf("ClassifyItem bogus_protection admitted")
	}
}
