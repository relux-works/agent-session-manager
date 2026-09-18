package clonebundle

import (
	"encoding/json"
	"strings"
	"testing"
)

// Section 1.6 value-model negatives for Canonical Event payloads: a
// non-message kind must refuse unsafe numbers and nested duplicates
// at Build and at Decode, and the sealed bytes of every admitted
// payload must carry the exact literals (never rounded, never
// collapsed). Every row drives a production entry point.

func mustUsageBytes(t *testing.T, payload string) []byte {
	t.Helper()
	input := validEventInput("exact")
	input.Kind = "usage"
	input.Payload = []byte(payload)
	built, err := BuildCanonicalEvent(input)
	if err != nil {
		t.Fatalf("BuildCanonicalEvent(usage %s) error = %v", payload, err)
	}
	return built
}

func sealedPayload(t *testing.T, sealed []byte) string {
	t.Helper()
	var members map[string]json.RawMessage
	if err := json.Unmarshal(sealed, &members); err != nil {
		t.Fatalf("json.Unmarshal(sealed event) error = %v", err)
	}
	return string(members["payload"])
}

// resealEventPayload swaps the payload member of valid event bytes
// for the replacement text and recomputes the omit-self digest over
// the tampered bytes: decoding refuses the result only when a
// content gate fires before the identity check, so no reseal can
// smuggle collapsed bytes past decoding.
func resealEventPayload(t *testing.T, valid []byte, original, replacement string) []byte {
	t.Helper()
	tampered := strings.Replace(string(valid), original, replacement, 1)
	members, fault := decodeStrictObject([]byte(tampered))
	if fault != nil {
		t.Fatalf("decodeStrictObject(tampered) fault = %+v", fault)
	}
	sealed, err := omitSelfDigest(members, canonicalEventSelf)
	if err != nil {
		t.Fatalf("omitSelfDigest(tampered) error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(valid, &decoded); err != nil {
		t.Fatal(err)
	}
	originalID, ok := decoded[canonicalEventSelf].(string)
	if !ok {
		t.Fatal("sealed event carries no string self id")
	}
	return []byte(strings.Replace(tampered, originalID, sealed.String(), 1))
}

func TestCanonicalEventPayloadValueModel(t *testing.T) {
	t.Run("build", func(t *testing.T) {
		cases := []struct {
			name     string
			kind     string
			payload  string
			fragment string
		}{
			{"usage float", "usage", `{"n":1.5}`, "safe-integer"},
			{"usage 2^53", "usage", `{"n":9007199254740992}`, "safe-integer"},
			{"usage 2^53 plus one", "usage", `{"n":9007199254740993}`, "safe-integer"},
			{"usage negative 2^53", "usage", `{"n":-9007199254740992}`, "safe-integer"},
			{"usage 2^60", "usage", `{"n":1152921504606846976}`, "safe-integer"},
			{"usage exponent", "usage", `{"n":1e2}`, "safe-integer"},
			{"usage nested duplicate", "usage", `{"facts":{"a":1,"a":2}}`, "duplicate nested member"},
			{"usage array nested duplicate", "usage", `{"n":[{"a":1,"a":2}]}`, "duplicate nested member"},
			{"tool_call float", "tool_call", `{"n":1.5}`, "safe-integer"},
			{"tool_call 2^53", "tool_call", `{"n":9007199254740992}`, "safe-integer"},
			{"tool_call nested duplicate", "tool_call", `{"facts":{"a":1,"a":2}}`, "duplicate nested member"},
			{"opaque_event exponent", "opaque_event", `{"n":1e2}`, "safe-integer"},
			{"error 2^53 plus one", "error", `{"n":9007199254740993}`, "safe-integer"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				input := validEventInput("exact")
				input.Kind = tc.kind
				input.Payload = []byte(tc.payload)
				_, err := BuildCanonicalEvent(input)
				requireRefusal(t, err, tc.fragment)
			})
		}
	})
	t.Run("build sealed exact", func(t *testing.T) {
		cases := []struct {
			name    string
			payload string
			sealed  string
		}{
			{"safe max", `{"n":9007199254740991}`, `{"n":9007199254740991}`},
			{"safe min", `{"n":-9007199254740991}`, `{"n":-9007199254740991}`},
			{"small int", `{"n":100}`, `{"n":100}`},
			{"nested exact", `{"facts":{"a":[1,{"b":2}]},"s":"x","t":true,"z":null}`, `{"facts":{"a":[1,{"b":2}]},"s":"x","t":true,"z":null}`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				built := mustUsageBytes(t, tc.payload)
				if got := sealedPayload(t, built); got != tc.sealed {
					t.Fatalf("sealed payload = %s, want %s", got, tc.sealed)
				}
				decoded, err := DecodeCanonicalEvent(built)
				if err != nil {
					t.Fatalf("DecodeCanonicalEvent() error = %v", err)
				}
				round, err := json.Marshal(decoded.Payload)
				if err != nil {
					t.Fatal(err)
				}
				if string(round) != tc.sealed {
					t.Fatalf("decoded payload remarshal = %s, want %s", round, tc.sealed)
				}
			})
		}
		if got := mustUsageBytes(t, `{"n":9007199254740991}`); strings.Contains(string(got), "9007199254740992") {
			t.Fatal("sealed bytes carry the rounded literal 9007199254740992")
		}
	})
	t.Run("decode", func(t *testing.T) {
		valid := mustUsageBytes(t, `{"n":1}`)
		cases := []struct {
			name     string
			payload  string
			fragment string
		}{
			{"float", `{"n":1.5}`, "safe-integer"},
			{"2^53", `{"n":9007199254740992}`, "safe-integer"},
			{"2^53 plus one", `{"n":9007199254740993}`, "safe-integer"},
			{"negative 2^53", `{"n":-9007199254740992}`, "safe-integer"},
			{"exponent", `{"n":1e2}`, "safe-integer"},
			{"nested duplicate", `{"facts":{"a":1,"a":2}}`, "duplicate nested member"},
			{"array nested duplicate", `{"n":[{"a":1,"a":2}]}`, "duplicate nested member"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				body := strings.Replace(string(valid), `"payload":{"n":1}`, `"payload":`+tc.payload, 1)
				_, err := DecodeCanonicalEvent([]byte(body))
				requireRefusal(t, err, tc.fragment)
			})
		}
	})
	t.Run("decode resealed", func(t *testing.T) {
		valid := mustUsageBytes(t, `{"n":1}`)
		original := `"payload":{"n":1}`
		for _, tc := range []struct {
			name     string
			payload  string
			fragment string
		}{
			{"2^53", `{"n":9007199254740992}`, "safe-integer"},
			{"nested duplicate", `{"facts":{"a":1,"a":2}}`, "duplicate nested member"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				resealed := resealEventPayload(t, valid, original, `"payload":`+tc.payload)
				_, err := DecodeCanonicalEvent(resealed)
				requireRefusal(t, err, tc.fragment)
			})
		}
	})
	t.Run("message regression", func(t *testing.T) {
		input := validEventInput("exact")
		built, err := BuildCanonicalEvent(input)
		if err != nil {
			t.Fatalf("BuildCanonicalEvent(valid message) error = %v", err)
		}
		if got, want := sealedPayload(t, built), `{"content_blocks":[{"content":"hello","type":"text"}],"extensions":{}}`; got != want {
			t.Fatalf("sealed message payload = %s, want %s", got, want)
		}
		deep := validEventInput("exact")
		deep.Payload = []byte(`{"content_blocks":[{"type":"text","content":"x","extensions":{"com.example.n":9007199254740991}}],"extensions":{"com.example.m":-9007199254740991}}`)
		rebuilt, err := BuildCanonicalEvent(deep)
		if err != nil {
			t.Fatalf("BuildCanonicalEvent(safe message extensions) error = %v", err)
		}
		if got := sealedPayload(t, rebuilt); !strings.Contains(got, "9007199254740991") || !strings.Contains(got, "-9007199254740991") {
			t.Fatalf("sealed message payload loses safe edges: %s", got)
		}
	})
	t.Run("depth bound", func(t *testing.T) {
		payload := strings.Repeat(`{"a":`, 300) + "1" + strings.Repeat(`}`, 300)
		input := validEventInput("exact")
		input.Kind = "usage"
		input.Payload = []byte(payload)
		_, err := BuildCanonicalEvent(input)
		requireRefusal(t, err, "depth-bounded payload object")
	})
}
