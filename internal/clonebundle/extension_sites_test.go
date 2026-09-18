package clonebundle

import (
	"strings"
	"testing"
)

// Extension value-model wiring: one value-fault negative per
// admission site. Every row injects the same forbidden literal into
// that shape's extensions through the shape's natural production
// entry, so a per-site plant that drops the value model at exactly
// one site is killed by exactly its row.

func faultExtensions() map[string]any {
	return map[string]any{"com.example.n": 1.5}
}

const faultExtensionsJSON = `{"com.example.n":1.5}`

func validAXBasisJSON(t *testing.T) string {
	t.Helper()
	return `{"kind":"ax_session","source_session_id":"` + fixtureSessionID +
		`","source_session_record_id":"` + fixtureDigest("test-record") +
		`","source_checkpoint_id":"` + fixtureDigest("test-checkpoint") +
		`","source_provider_identity_record_id":"` + fixtureDigest("test-provider") +
		`","extensions":{}}`
}

func validProofJSON(t *testing.T) string {
	t.Helper()
	return `{"proof_kind":"closed_store","source_generation":"g",` +
		`"snapshot_identity_digest":null,` +
		`"pre_capture_digest":"` + fixtureDigest("p") + `",` +
		`"post_capture_digest":"` + fixtureDigest("p") + `",` +
		`"input_blocked":true,"foreground_idle":true,"background_idle":true,"extensions":{}}`
}

func validEvidenceJSON(t *testing.T) string {
	t.Helper()
	return `{"environment":` + fixtureTupleJSON() +
		`,"native_session_id":"native-session-alpha","native_event_id":"native-event-1",` +
		`"native_type":"chat.message","raw_refs":[],"capture_status":"exact",` +
		`"reason_codes":[],"core_operation_id":null,"extensions":{}}`
}

func validIdentityJSON(t *testing.T) string {
	t.Helper()
	return `{"native_session_id":"native-session-alpha","identity_kind":"provider_native",` +
		`"logical_workspace_id":"` + fixtureWorkspaceID + `",` +
		`"backend_realm_fingerprint":null,"opaque_identity":null,"extensions":{}}`
}

func TestExtensionValueModelAtEverySite(t *testing.T) {
	t.Run("raw_manifest_decode", func(t *testing.T) {
		body := strings.Replace(string(mustRawBytes(t)), `"extensions":{}`, `"extensions":`+faultExtensionsJSON, 1)
		_, err := DecodeRawObjectManifest([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("raw_manifest_build", func(t *testing.T) {
		identity, err := IdentityDigest(fixtureNativeIdentity(), nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
			"native-session-alpha", identity.String(), fixtureDigest("test-capture-plan"),
			validEntryInputs(t), faultExtensions())
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("source_basis_ax", func(t *testing.T) {
		body := strings.Replace(validAXBasisJSON(t), `"extensions":{}`, `"extensions":`+faultExtensionsJSON, 1)
		_, err := DecodeSourceBasis([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("source_basis_external", func(t *testing.T) {
		body := `{"kind":"external_native","external_source_ref":"native-ref","extensions":` + faultExtensionsJSON + `}`
		_, err := DecodeSourceBasis([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("stable_proof", func(t *testing.T) {
		body := strings.Replace(validProofJSON(t), `"extensions":{}`, `"extensions":`+faultExtensionsJSON, 1)
		_, err := DecodeStableSnapshotProof([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("boundary_stable", func(t *testing.T) {
		body := `{"kind":"stable","proof":` + validProofJSON(t) + `,"extensions":` + faultExtensionsJSON + `}`
		_, err := DecodeCaptureBoundary([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("boundary_unstable", func(t *testing.T) {
		body := `{"kind":"unstable_archive","source_generation":"g",` +
			`"pre_capture_digest":"` + fixtureDigest("p") + `",` +
			`"post_capture_digest":"` + fixtureDigest("p") + `",` +
			`"reason_code":"source_not_quiescent","operator_explicit":true,` +
			`"target_projection_forbidden":true,"extensions":` + faultExtensionsJSON + `}`
		_, err := DecodeCaptureBoundary([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("capture_manifest", func(t *testing.T) {
		body := tamper(t, mustCaptureBytes(t), func(o map[string]any) { o["extensions"] = faultExtensions() })
		_, err := DecodeCaptureManifest(body)
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("capture_item", func(t *testing.T) {
		body := tamper(t, mustCaptureBytes(t), func(o map[string]any) {
			items := o["items"].([]any)
			items[0].(map[string]any)["extensions"] = faultExtensions()
		})
		_, err := DecodeCaptureManifest(body)
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("source_evidence", func(t *testing.T) {
		body := strings.Replace(validEvidenceJSON(t), `"extensions":{}`, `"extensions":`+faultExtensionsJSON, 1)
		_, err := DecodeSourceEvidence([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("canonical_event", func(t *testing.T) {
		body := tamper(t, mustEventBytes(t, "exact"), func(o map[string]any) { o["extensions"] = faultExtensions() })
		_, err := DecodeCanonicalEvent(body)
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("message_payload", func(t *testing.T) {
		input := validEventInput("exact")
		input.Payload = []byte(`{"content_blocks":[{"type":"text","content":"x"}],"extensions":` + faultExtensionsJSON + `}`)
		_, err := BuildCanonicalEvent(input)
		requireRefusal(t, err, "message payload extensions")
	})
	t.Run("content_block", func(t *testing.T) {
		input := validEventInput("exact")
		input.Payload = []byte(`{"content_blocks":[{"type":"text","content":"x","extensions":` + faultExtensionsJSON + `}],"extensions":{}}`)
		_, err := BuildCanonicalEvent(input)
		requireRefusal(t, err, "content block[0] extensions")
	})
	t.Run("native_identity", func(t *testing.T) {
		body := strings.Replace(validIdentityJSON(t), `"extensions":{}`, `"extensions":`+faultExtensionsJSON, 1)
		_, err := DecodeNativeIdentity([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("workspace_binding", func(t *testing.T) {
		body := strings.Replace(fixtureWorkspaceJSON(), `"extensions":{}`, `"extensions":`+faultExtensionsJSON, 1)
		_, err := DecodeWorkspaceBinding([]byte(body))
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("actor", func(t *testing.T) {
		body := tamper(t, mustSessionBytes(t), func(o map[string]any) {
			actors := o["actors"].([]any)
			actors[0].(map[string]any)["extensions"] = faultExtensions()
		})
		_, err := DecodeCanonicalSession(body)
		requireRefusal(t, err, "safe-integer")
	})
	t.Run("canonical_session", func(t *testing.T) {
		body := tamper(t, mustSessionBytes(t), func(o map[string]any) { o["extensions"] = faultExtensions() })
		_, err := DecodeCanonicalSession(body)
		requireRefusal(t, err, "safe-integer")
	})
}
