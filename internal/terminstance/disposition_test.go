package terminstance

import (
	"testing"
)

// TestParseRetryDispositionClosed pins the four-member disposition
// vocabulary at the production entry with literal assertions.
func TestParseRetryDispositionClosed(t *testing.T) {
	for _, disposition := range []string{"replay_same", "status_first", "new_authorization", "required_operator_action"} {
		if _, err := ParseRetryDisposition(disposition); err != nil {
			t.Errorf("ParseRetryDisposition(%q) error = %v, want admission", disposition, err)
		}
	}
	// The corpus carries the sibling vocabularies a caller could
	// plausibly confuse with a disposition: the AX authorization
	// kinds and the provider proof kinds from the same section.
	for _, disposition := range []string{"retry", "REPLAY_SAME", "", "status-first", "create", "control", "provider_quiescence", "Replay_Same", "Status_First", "NEW_AUTHORIZATION", " required_operator_action"} {
		_, err := ParseRetryDisposition(disposition)
		requireRefusal(t, err, "terminal_backend_protocol_error", "retry disposition vocabulary")
	}
}

// TestDispositionForMapping pins the leaf's per-code mapping literally:
// each arm names its code and its committed/uncommitted disposition.
func TestDispositionForMapping(t *testing.T) {
	cases := []struct {
		code      string
		committed bool
		want      string
	}{
		{"terminal_backend_unauthorized", false, "new_authorization"},
		{"terminal_backend_unauthorized", true, "new_authorization"},
		{"terminal_backend_timeout", false, "replay_same"},
		{"terminal_backend_timeout", true, "status_first"},
		{"quiesce_timeout", false, "replay_same"},
		{"quiesce_timeout", true, "status_first"},
		{"stop_timeout", false, "status_first"},
		{"stop_timeout", true, "status_first"},
		{"idempotency_mismatch", false, "status_first"},
		{"idempotency_mismatch", true, "status_first"},
		{"terminal_backend_stale_generation", false, "status_first"},
		{"terminal_backend_stale_generation", true, "status_first"},
		{"local_precondition_failed", false, "status_first"},
		{"local_precondition_failed", true, "status_first"},
		{"terminal_backend_unavailable", false, "status_first"},
		{"terminal_backend_unavailable", true, "status_first"},
		{"terminal_backend_capability_unproven", false, "required_operator_action"},
		{"terminal_backend_capability_unproven", true, "status_first"},
		{"terminal_backend_integrity_failure", false, "required_operator_action"},
		{"terminal_backend_integrity_failure", true, "status_first"},
		{"terminal_backend_process_failed", false, "required_operator_action"},
		{"terminal_backend_process_failed", true, "status_first"},
		{"terminal_backend_protocol_error", false, "required_operator_action"},
		{"terminal_backend_protocol_error", true, "status_first"},
		{"something_unknown", false, "required_operator_action"},
		{"something_unknown", true, "status_first"},
	}
	for _, tc := range cases {
		if got := DispositionFor(tc.code, tc.committed); string(got) != tc.want {
			t.Errorf("DispositionFor(%q, committed=%v) = %q, want literal %q", tc.code, tc.committed, string(got), tc.want)
		}
	}
}

// TestParseProviderProofKindClosed pins the three-member proof vocabulary
// at the production entry with literal assertions.
func TestParseProviderProofKindClosed(t *testing.T) {
	for _, kind := range []string{"provider_quiescence", "provider_process_exit", "ax_checkpoint_boundary"} {
		if _, err := ParseProviderProofKind(kind); err != nil {
			t.Errorf("ParseProviderProofKind(%q) error = %v, want admission", kind, err)
		}
	}
	// The corpus carries the landed sibling names a caller could
	// plausibly confuse with a proof kind: the capability names and
	// the side-effect name from the same section.
	for _, kind := range []string{"provider_restart", "PROVIDER_QUIESCENCE", "", "provider_process_observation", "safe_boundary_observation", "safe_boundary_observed", "provider_exit", "provider_quiescence ", "Provider_Quiescence", "Ax_Checkpoint_Boundary", "provider_process_exit "} {
		_, err := ParseProviderProofKind(kind)
		requireRefusal(t, err, "terminal_backend_protocol_error", "provider proof vocabulary")
	}
}

// TestRequiresProviderObservation pins the provider-kind classification:
// the two provider kinds require provider observation, the AX boundary
// does not.
func TestRequiresProviderObservation(t *testing.T) {
	if !RequiresProviderObservation(ProofProviderQuiescence) {
		t.Error("RequiresProviderObservation(provider_quiescence) = false, want true")
	}
	if !RequiresProviderObservation(ProofProviderProcessExit) {
		t.Error("RequiresProviderObservation(provider_process_exit) = false, want true")
	}
	if RequiresProviderObservation(ProofAXCheckpointBoundary) {
		t.Error("RequiresProviderObservation(ax_checkpoint_boundary) = true, want false")
	}
}

// TestParseAuthorizationKindClosed pins the four-member kind vocabulary
// at the production entry with literal assertions.
func TestParseAuthorizationKindClosed(t *testing.T) {
	for _, kind := range []string{"create", "control", "force_stale", "restore"} {
		if _, err := ParseAuthorizationKind(kind); err != nil {
			t.Errorf("ParseAuthorizationKind(%q) error = %v, want admission", kind, err)
		}
	}
	// The corpus carries the sibling vocabularies alongside the
	// landed attach/none names: the retry dispositions and the
	// provider proof kinds from the same section.
	for _, kind := range []string{"admin", "CREATE", "", "force-stale", "attach", "none", "replay_same", "provider_quiescence", "Control", "Create", "CONTROL", " control", "control ", "Force_Stale", "Restore"} {
		_, err := ParseAuthorizationKind(kind)
		requireRefusal(t, err, "terminal_backend_protocol_error", "ax authorization kind")
	}
}
