package sessquery

import (
	"errors"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/cliresult"
)

// This is a measured integration bound, not a successful public list claim.
// Pinned Section 5.7 defines creating as a Session Record AND an
// authoritative initial lease, and Section 13.1 step 2 persists the
// initial Lease Record and session.created before process creation: a
// record alone proves identity and provenance, never creation of an
// owner or a creating lifecycle. A record-only chain is therefore an
// interrupted persistence prefix awaiting the bootstrap-recovery leaf
// (Section 14.7.4), not a completed bootstrap prefix, and the list
// entry refuses it with selector_bootstrap_incomplete instead of
// returning a row with empty owner/lease facts. The wire owner still
// requires known owner/lease fields: the read layer must not invent
// them to obtain green JSON, and no nullable or placeholder owner or
// lease field is authorized to close the gap.
func TestCreatingSummaryCannotClaimClosedCLIResult(t *testing.T) {
	repo, _ := repository(t)
	create(t, repo, idA, "Creating")
	// The production list entry refuses the interrupted prefix; no
	// partial row escapes for a caller to render.
	if _, err := (&Reader{Local: repo}).List(); !errors.Is(err, ErrBootstrapIncomplete) {
		t.Fatalf("record-only list = %v, want selector_bootstrap_incomplete", err)
	}
	if _, err := (&Reader{Local: repo}).Status("Creating"); !errors.Is(err, ErrBootstrapIncomplete) {
		t.Fatalf("record-only status = %v, want selector_bootstrap_incomplete", err)
	}
	// The ownerless shape still cannot satisfy the mandatory CLI
	// owner/lease fields, even hand-built outside the refused entry.
	body := map[string]any{
		"session_id": idA, "name": "Creating", "kind": "direct", "provider_id": "codex",
		"owner_host_id": "", "owner_host_name": "", "lease_epoch": uint64(0), "lease_id": "",
		"local_role": "", "state": "creating", "newest_checkpoint_id": nil, "newest_checkpoint_created_at": nil,
		"workspace_status": "absent", "capabilities": map[string]any{}, "warnings": []string{},
	}
	// The fixture's absent workspace is controlled only for this shape probe;
	// it is not a production observation or a default in Reader.
	if _, err := cliresult.New(cliresult.Spec{Command: cliresult.CommandList, Body: map[string]any{
		"sessions": []any{body}, "partial": false, "unreachable_peer_ids": []string{},
	}}); !errors.Is(err, cliresult.ErrInvalidResult) {
		t.Fatalf("unknown owner represented as CLI success: %v", err)
	}
}
