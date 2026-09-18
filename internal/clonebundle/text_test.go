package clonebundle

import (
	"bytes"
	"testing"
)

// Regression tests for the Section 1.6 text gate: text MUST be valid
// UTF-8, and encoding/json silently rewrites invalid bytes to U+FFFD
// instead of failing — so every Build-side string admission refuses
// invalid UTF-8 before marshaling. Every row drives a production
// entry point and asserts two things: the entry refuses with
// ErrInvalid naming the UTF-8 rule, and no returned bytes carry the
// U+FFFD the marshaler would have substituted. Asserting the sealed
// bytes (not merely the refusal) pins the rewrite-and-continue
// mechanism: under a mutant that re-admits one invalid value, the
// sealed bytes carry U+FFFD and the row fails.
func TestBuildTextMustBeValidUTF8(t *testing.T) {
	replacement := string(rune(0xFFFD))
	cases := []struct {
		name  string
		build func(t *testing.T) ([]byte, error)
	}{
		{"capture_source_identity_native", func(t *testing.T) ([]byte, error) {
			input := validCaptureInput(t)
			input.SourceIdentity.NativeSessionID = "native\xff"
			return BuildCaptureManifest(input)
		}},
		{"capture_source_identity_native_lone_surrogate", func(t *testing.T) ([]byte, error) {
			input := validCaptureInput(t)
			input.SourceIdentity.NativeSessionID = "native\xed\xa0\x80"
			return BuildCaptureManifest(input)
		}},
		{"capture_source_identity_opaque", func(t *testing.T) ([]byte, error) {
			input := validCaptureInput(t)
			input.SourceIdentity.OpaqueIdentity = strptr("op\xff")
			return BuildCaptureManifest(input)
		}},
		{"capture_item_exclusion_reason", func(t *testing.T) ([]byte, error) {
			input := validCaptureInput(t)
			input.Items[2].ExclusionReason = strptr("credential\xff")
			return BuildCaptureManifest(input)
		}},
		{"capture_boundary_generation", func(t *testing.T) ([]byte, error) {
			input := validCaptureInput(t)
			input.Boundary.Generation = "g\xff"
			return BuildCaptureManifest(input)
		}},
		{"capture_identity_extensions", func(t *testing.T) ([]byte, error) {
			input := validCaptureInput(t)
			input.IdentityExtensions = map[string]any{"com.example.note": "v\xff"}
			return BuildCaptureManifest(input)
		}},
		{"capture_item_extensions", func(t *testing.T) ([]byte, error) {
			input := validCaptureInput(t)
			input.Items[2].Extensions = map[string]any{"com.example.note": "v\xff"}
			return BuildCaptureManifest(input)
		}},
		{"capture_manifest_extensions", func(t *testing.T) ([]byte, error) {
			input := validCaptureInput(t)
			input.Extensions = map[string]any{"com.example.note": "v\xff"}
			return BuildCaptureManifest(input)
		}},
		{"digest_opaque", func(t *testing.T) ([]byte, error) {
			identity := fixtureNativeIdentity()
			identity.OpaqueIdentity = strptr("op\xff")
			_, err := IdentityDigest(identity, nil)
			return nil, err
		}},
		{"digest_native_collision_pair", func(t *testing.T) ([]byte, error) {
			identity := fixtureNativeIdentity()
			identity.NativeSessionID = "native\xfe"
			_, err := IdentityDigest(identity, nil)
			return nil, err
		}},
		{"session_title", func(t *testing.T) ([]byte, error) {
			input := validSessionInput()
			input.Title = strptr("ti\xfftle")
			return BuildCanonicalSession(input)
		}},
		{"session_actor_name", func(t *testing.T) ([]byte, error) {
			input := validSessionInput()
			input.Actors[0].Name = strptr("na\xffme")
			return BuildCanonicalSession(input)
		}},
		{"session_actor_model", func(t *testing.T) ([]byte, error) {
			input := validSessionInput()
			input.Actors[0].Model = strptr("mo\xffdel")
			return BuildCanonicalSession(input)
		}},
		{"session_extensions", func(t *testing.T) ([]byte, error) {
			input := validSessionInput()
			input.Extensions = map[string]any{"com.example.note": "v\xff"}
			return BuildCanonicalSession(input)
		}},
		{"event_evidence_native_type", func(t *testing.T) ([]byte, error) {
			input := validEventInput("exact")
			input.Evidence.NativeType = strptr("nt\xff")
			return BuildCanonicalEvent(input)
		}},
		{"event_evidence_reason", func(t *testing.T) ([]byte, error) {
			input := validEventInput("exact")
			input.Evidence.ReasonCodes = []string{"rc\xff"}
			return BuildCanonicalEvent(input)
		}},
		{"event_extensions", func(t *testing.T) ([]byte, error) {
			input := validEventInput("exact")
			input.Extensions = map[string]any{"com.example.note": "v\xff"}
			return BuildCanonicalEvent(input)
		}},
		{"raw_extensions", func(t *testing.T) ([]byte, error) {
			return BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
				"native-session-alpha", fixtureDigest("test-identity"), fixtureDigest("test-capture-plan"),
				validEntryInputs(t), map[string]any{"com.example.note": "v\xff"})
		}},
		{"raw_extensions_nested", func(t *testing.T) ([]byte, error) {
			return BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
				"native-session-alpha", fixtureDigest("test-identity"), fixtureDigest("test-capture-plan"),
				validEntryInputs(t), map[string]any{"com.example.obj": map[string]any{"inner": []any{"v\xff"}}})
		}},
		{"raw_extensions_key", func(t *testing.T) ([]byte, error) {
			return BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
				"native-session-alpha", fixtureDigest("test-identity"), fixtureDigest("test-capture-plan"),
				validEntryInputs(t), map[string]any{"com.example.\xff": 1})
		}},
		{"raw_extensions_struct", func(t *testing.T) ([]byte, error) {
			return BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
				"native-session-alpha", fixtureDigest("test-identity"), fixtureDigest("test-capture-plan"),
				validEntryInputs(t), map[string]any{"com.example.rec": struct{ V string }{V: "v\xff"}})
		}},
		{"parse_generation", func(t *testing.T) ([]byte, error) {
			_, err := ParseGeneration("g\xff")
			return nil, err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sealed, err := tc.build(t)
			if err == nil {
				if bytes.Contains(sealed, []byte(replacement)) {
					t.Fatalf("admitted invalid UTF-8 and sealed U+FFFD: %q", truncateBytes(sealed, 200))
				}
				t.Fatalf("admitted invalid UTF-8, want ErrInvalid")
			}
			requireRefusal(t, err, "not valid UTF-8")
			if bytes.Contains(sealed, []byte(replacement)) {
				t.Fatalf("refused but returned bytes carrying U+FFFD: %q", truncateBytes(sealed, 200))
			}
		})
	}
	// The collision pair refuses on both sides: neither "native\xff"
	// nor "native\xfe" seals, so the two can no longer collapse to
	// one byte-identical identity.
	for _, key := range []string{"native\xff", "native\xfe"} {
		identity := fixtureNativeIdentity()
		identity.NativeSessionID = key
		if _, err := IdentityDigest(identity, nil); err == nil {
			t.Fatalf("IdentityDigest(%q) admitted, want ErrInvalid", key)
		}
	}
}

func truncateBytes(data []byte, maximum int) []byte {
	if len(data) > maximum {
		return data[:maximum]
	}
	return data
}
