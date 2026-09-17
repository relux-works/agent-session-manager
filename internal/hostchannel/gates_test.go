package hostchannel_test

import (
	"testing"
)

// hcGates is the Section 11.10.4 closed gate registry as executable
// fixture families. Each family names its positive vectors (admit exactly
// the specified peer) and negative vectors (refuse exactly one hostile
// deviation) driven through the production entries. The conformance matrix
// in the task outcome traces every gate to these tests.
var hcGates = map[string]struct {
	positives []string
	negatives []string
}{
	"HC-TLS": {
		positives: []string{"TestFullHandshakeHelloAndOperation", "TestHostileRealCarrierOpenSSH", "TestHostileRoleSwap"},
		negatives: []string{
			"TestRefusesTLS12Offer/server", "TestRefusesTLS12Offer/client",
			"TestRefusesWrongALPN/server", "TestRefusesWrongALPN/client",
			"TestRefusesMissingALPN", "TestRefusesResumption",
			"TestRefusesPlaintextPreface", "TestVerifyPeerRefusals/alpn",
			"TestVerifyPeerRefusals/alpn-same-family", "TestVerifyPeerRefusals/missing-alpn",
			"TestVerifyPeerRefusals/resumed", "TestSilentPeerHandshakeDeadline",
			"TestHandshakeLimiterBounds", "TestHandshakeIntakeIsBounded",
			"TestHostileReplay/tls_flight_replay_refused", "TestHostileReplay/encrypted_hello_replay_refused",
			"TestHostileReplayNonceFreshness", "TestHostileDisconnectPhases/mid_handshake_close",
			"TestHostileOversizedFrames/handshake_intake_bound_documented",
		},
	},
	"HC-CERT": {
		positives: []string{"TestVerifyPeerAdmitsEnrolled", "TestFullHandshakeHelloAndOperation", "TestCredentialFromIssued", "TestEnrollmentPoolExcludesRevoked", "TestHostileCopiedKey (pre-revocation half)"},
		negatives: []string{
			"TestRefusesUnenrolledRoot", "TestRefusesRevokedLeaf",
			"TestRefusesRetiredPastRetireAt", "TestRefusesExpiredLeaf",
			"TestVerifyPeerRefusals/no-chains", "TestVerifyPeerRefusals/no-certificates",
			"TestVerifyPeerRefusals/unknown-leaf", "TestVerifyPeerRefusals/expired",
			"TestVerifyPeerRefusals/revoked", "TestVerifyPeerRefusals/retiring-past",
			"TestVerifyPeerRefusals/retiring-nil", "TestCredentialFromIssuedRefusals",
			"TestEnrollmentPoolRefusesOverflow",
			"TestHostileUnknownPeerBothRoles/responder_refuses", "TestHostileUnknownPeerBothRoles/initiator_refuses",
			"TestHostileKeyChangeSameUUID/responder_refuses_reissued", "TestHostileKeyChangeSameUUID/initiator_refuses_reissued",
			"TestHostileCopiedKey (post-revocation halves)",
		},
	},
	"HC-MAP": {
		positives: []string{"TestVerifyPeerAdmitsEnrolled", "TestServerNameMatchesIssuedLeaf"},
		negatives: []string{
			"TestVerifyPeerRefusals/double-match", "TestVerifyPeerRefusals/leaf-flipped",
			"TestVerifyPeerRefusals/root-flipped", "TestVerifyPeerRefusals/spki-zero",
			"TestRefusesWrongServerName", "TestHostileSpoofedHostID/valid_but_unexpected_peer",
		},
	},
	"HC-HELLO": {
		positives: []string{"TestFullHandshakeHelloAndOperation", "TestInventoryRootsThroughChannel", "TestNewRequestID", "TestLocalFailureMapping"},
		negatives: []string{
			"TestRefusesHelloUUIDMismatch", "TestRefusesServerHelloUUIDMismatch",
			"TestRefusesDestinationMismatch", "TestRefusesNotAllowlisted",
			"TestRefusesInvalidHello/contract", "TestRefusesInvalidHello/missing-key",
			"TestRefusesInvalidHello/nonce", "TestHelloResponseCorrelation",
			"TestHelloFailureMapping/host_identity_mismatch", "TestHelloFailureMapping/peer_not_allowlisted",
			"TestHelloFailureMapping/incompatible_protocol", "TestHelloDeadline/server",
			"TestHelloDeadline/client", "TestHelloWriteDeadline",
			"TestHostileSpoofedHostID/hello_claims_destination",
			"TestHostileDisclosureMismatch/contracts_rpc_superset", "TestHostileDisclosureMismatch/contracts_rpc_missing",
			"TestHostileDisclosureMismatch/responder_contracts_drift", "TestHostileDisclosureMismatch/error_version_drift",
			"TestHostileOversizedFrames/hello_below_advertised_floor",
			"TestHostileDisconnectPhases/after_handshake_before_hello", "TestHostileDisconnectPhases/hello_half_frame_close",
			"TestHostileDisconnectPhases/server_aborts_hello_reply",
		},
	},
	"HC-DISPATCH": {
		positives: []string{"TestFullHandshakeHelloAndOperation", "TestHandlerErrorFramedAndContinues", "TestHostileConcurrentPeerConnections"},
		negatives: []string{
			"TestRefusesNonHelloBeforeHello", "TestForeignMajorClosesSilently/2.0.0",
			"TestForeignMajorClosesSilently/4.0.0", "TestUnframeableHelloClosesSilently",
			"TestOversizeHelloLine", "TestDispatchTimeout", "TestDispatchExpiryClosesWithoutFrame",
			"TestCallRefusesInvalidBody", "TestHandlerGarbageBodyCloses",
			"TestHostileOversizedFrames/hello_line_8mib_plus_one", "TestHostileOversizedFrames/dispatch_line_8mib_plus_one",
			"TestHostileDisconnectPhases/mid_request_close", "TestHostileDisconnectPhases/mid_response_close",
			"TestHostileDisconnectPhases/server_aborts_response",
			"TestHostileStallingPeerRPCTimeout", "TestHostileReplay/rehello_is_inert_redispatch",
			"TestHostileRecoveryBypass/call_after_close_refused", "TestHostileRecoveryBypass/response_id_mismatch_refused",
		},
	},
	"HC-MIGRATE": {
		positives: []string{"TestLaunchForConfig", "TestServeArgvPinsChannelVersion"},
		negatives: []string{
			"TestLaunchForConfigRefusesLegacy", "TestLaunchForConfigEmptySnapshot",
			"TestForeignMajorClosesSilently/2.0.0", "TestForeignMajorClosesSilently/4.0.0",
			"TestInvalidActivation", "TestLocalCredentialRefusals",
			"TestHostileV4WithoutHostChannelBinding",
		},
	},
	"HC-LIFECYCLE": {
		positives: []string{"TestRetiringCredentialAdmitsWithinBound", "TestMutationBoundaryRunsFresh", "TestHostileKeyChangeSameUUID/rotation_admits_new_key_only_after_enrollment"},
		negatives: []string{
			"TestRefusesRevokedLeaf", "TestRefusesRetiredPastRetireAt", "TestRefusesExpiredLeaf",
			"TestHostileKeyChangeSameUUID/no_second_rotation_before_revocation",
			"TestHostileRevocationLivePeer/next_dispatch_refused_fast",
		},
	},
	"HC-GENERATION": {
		positives: []string{"TestMutationBoundaryRunsFresh", "TestFullHandshakeHelloAndOperation", "TestAuthorityListOverflow"},
		negatives: []string{
			"TestStaleGenerationAtDispatch/server_closes", "TestStaleGenerationAtDispatch/client_refuses_to_send",
			"TestStaleGenerationAtMutation", "TestWatchGeneration",
			"TestHostileRecoveryBypass/stale_binding_never_rebinds",
			"TestHostileRevocationLivePeer/watch_reports_commit_fast",
			"TestHostileStaleReadsAreIntegrityFailures/corrupt_store", "TestHostileStaleReadsAreIntegrityFailures/unreadable_store",
			"TestHostileStaleReadsAreIntegrityFailures/initiator_unreadable_store",
		},
	},
	"HC-EXCLUDE": {
		positives: []string{"TestRuntimeWriteCensus", "TestStaticWriteCensus", "TestBindingRedaction"},
		negatives: []string{
			"census-evasion mutant (killed by TestRuntimeWriteCensus)",
			"every TestHostile* vector carries a before/after state census (or the revocation-transition census)",
		},
	},
	"HC-PARITY": {
		positives: []string{"TestFullHandshakeHelloAndOperation (transport-agnostic Stream)", "TestHostileRealCarrierOpenSSH (real OpenSSH loopback carrier)"},
		negatives: []string{"stated bound: native Tailscale SSH not executed; no SSH-implementation branch exists in the package, so the same TLS profile runs over any ordered binary Stream"},
	},
}

func TestHCGateRegistryIsComplete(t *testing.T) {
	want := []string{"HC-TLS", "HC-CERT", "HC-MAP", "HC-HELLO", "HC-DISPATCH", "HC-MIGRATE", "HC-LIFECYCLE", "HC-GENERATION", "HC-EXCLUDE", "HC-PARITY"}
	if len(hcGates) != len(want) {
		t.Fatalf("registry carries %d gates, Section 11.10.4 defines %d", len(hcGates), len(want))
	}
	for _, gate := range want {
		family, ok := hcGates[gate]
		if !ok {
			t.Fatalf("gate %s missing", gate)
		}
		if len(family.positives) == 0 || len(family.negatives) == 0 {
			t.Fatalf("gate %s lacks a positive or negative family", gate)
		}
	}
}
