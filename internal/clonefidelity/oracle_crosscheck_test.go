package clonefidelity_test

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// TestOwnerVocabulariesMatchSpec pins the landed Section 13.14.1
// vocabularies production iterates against lists retyped from the
// pinned SPEC v0.7.0 text: the 26 event kinds (line 10386), the 8
// content-block types (line 10390), and the 9 capture classes (line
// 10349). Production derives closed key sets from the owner; these
// rows prove the owner equals the spec.
func TestOwnerVocabulariesMatchSpec(t *testing.T) {
	kinds := []string{
		"session_started", "instruction_snapshot", "user_message",
		"assistant_message", "reasoning_summary", "opaque_reasoning",
		"tool_definition_snapshot", "tool_call", "tool_result",
		"approval_request", "approval_response", "plan_update",
		"progress", "usage", "rate_limit", "file_change",
		"compaction", "turn_started", "turn_completed", "turn_aborted",
		"subagent_started", "subagent_completed", "error",
		"migration_checkpoint", "session_finished", "opaque_event",
	}
	ownerKinds := clonebundle.EventKinds()
	if len(ownerKinds) != 26 {
		t.Fatalf("owner kind count = %d, want 26", len(ownerKinds))
	}
	for index, kind := range kinds {
		if ownerKinds[index] != kind {
			t.Fatalf("owner kind[%d] = %q, want %q", index, ownerKinds[index], kind)
		}
		if !clonebundle.ValidEventKind(kind) {
			t.Fatalf("ValidEventKind(%q) = false, want true", kind)
		}
	}
	blocks := []string{"text", "json", "image", "audio", "document", "resource_link", "redacted", "opaque"}
	ownerBlocks := clonebundle.ContentBlockTypes()
	if len(ownerBlocks) != 8 {
		t.Fatalf("owner block count = %d, want 8", len(ownerBlocks))
	}
	for index, block := range blocks {
		if ownerBlocks[index] != block {
			t.Fatalf("owner block[%d] = %q, want %q", index, ownerBlocks[index], block)
		}
		if !clonebundle.ValidContentBlockType(block) {
			t.Fatalf("ValidContentBlockType(%q) = false, want true", block)
		}
	}
	classes := []string{
		"durable_payload", "durable_index_required", "durable_sidecar",
		"derived_cache_optional", "credential", "machine_auth",
		"runtime_state", "transient_lock", "unknown",
	}
	for _, class := range classes {
		if !clonebundle.ValidCaptureClass(class) {
			t.Fatalf("ValidCaptureClass(%q) = false, want true", class)
		}
	}
	// This package's own registries expose the spec counts.
	if len(clonefidelity.Dispositions()) != 7 {
		t.Fatalf("Dispositions() count = %d, want 7", len(clonefidelity.Dispositions()))
	}
	if len(clonefidelity.Profiles()) != 5 {
		t.Fatalf("Profiles() count = %d, want 5", len(clonefidelity.Profiles()))
	}
	if len(clonefidelity.Strategies()) != 5 {
		t.Fatalf("Strategies() count = %d, want 5", len(clonefidelity.Strategies()))
	}
	if len(clonefidelity.CoreReasonCodes()) != 19 {
		t.Fatalf("CoreReasonCodes() count = %d, want 19", len(clonefidelity.CoreReasonCodes()))
	}
}
