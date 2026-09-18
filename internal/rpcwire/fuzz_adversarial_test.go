package rpcwire_test

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

// fuzzEnvelopeID is the fixed valid UUIDv7 correlator every fuzz-built
// envelope carries, so the fuzzer explores operation/body shapes rather
// than ID syntax (covered by FuzzUntrustedEnvelopes and the unit suite).
const fuzzEnvelopeID = "0198f4c8-a070-7188-9172-1234567890ab"

func fuzzLine(operation string, body []byte) []byte {
	envelope := map[string]any{
		"protocol":         rpcwire.Protocol,
		"protocol_version": "2.0.0",
		"request_id":       fuzzEnvelopeID,
		"operation":        operation,
		"body":             json.RawMessage(body),
	}
	line, err := json.Marshal(envelope)
	if err != nil {
		return nil
	}
	return line
}

func validObjectOrEmpty(body []byte) []byte {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return []byte(`{}`)
	}
	var probe map[string]json.RawMessage
	if json.Unmarshal(trimmed, &probe) != nil {
		return []byte(`{}`)
	}
	return trimmed
}

// validHelloBody returns the normative v2 hello request body so the admit
// path is seeded, not only the refusal paths.
func validHelloBody() ([]byte, error) {
	raw, err := os.ReadFile("testdata/hello-request-v2.json")
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Body json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	return []byte(envelope.Body), nil
}

func exactKeys(raw json.RawMessage, keys ...string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || len(m) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := m[key]; !ok {
			return false
		}
	}
	return true
}

// FuzzClosedOperationBodies reaches the body parsers rather than only the
// envelope: for every operation of the Section 11.3 vocabulary (plus
// adversarial names), an admitted request must round-trip through the
// production encode entry and must not reinterpret the operation, while a
// parsed hello/inventory body must decode through its typed accessor.
//
// The oracle checks envelope identity, the I-JSON object gate, round-trip,
// and exact member sets only: member bounds and vocabularies (nonce length
// and alphabet, floors, namespace tables) are owned by the unit suite,
// which kills the plants this target cannot see (see
// TestClosedBodyBoundsWitnessed and the mutations.py nonce/namespace rows).
//
// Control: the closed-extra narrowing (mutations.py) admits one extra=true
// envelope member; with that plant applied the exact-keys property below
// fails on the seeded bad input. See the producer evidence log.
func FuzzClosedOperationBodies(f *testing.F) {
	operations := []string{"hello", "health.get", "inventory.roots", "inventory.children", "objects.get",
		"transfer.begin", "transfer.status", "chunks.put", "transfer.validate", "transfer.commit",
		"materialize.prepare", "materialize.commit", "materialize.status", "materialize.finalize",
		"materialize.rollback", "lease.refresh", "tombstone.ack", "session.status", "session.stop",
		"handoff.prepare", "handoff.quiesce", "handoff.stop", "handoff.commit", "handoff.abort",
		"", "HELLO", "hello ", "future.operation"}
	for _, operation := range operations {
		f.Add(operation, []byte(`{}`))
	}
	f.Add("hello", []byte(`{"host_id":"x"}`))
	f.Add("inventory.roots", []byte(`{"namespaces":["blob"],"extra":true}`))
	f.Add("inventory.roots", []byte(`{"namespaces":["blob"]}`))
	// Non-object bodies reach the object gate (P3-1): the envelope carries
	// the fuzz bytes verbatim, so these seeds refuse rather than normalize.
	for _, body := range []string{`[]`, `null`, `"x"`, `42`, `{"a":1,"a":2}`} {
		f.Add("health.get", []byte(body))
	}
	if raw, err := validHelloBody(); err == nil {
		f.Add("hello", raw)
		// A hello body the target must reject: the control plant that
		// admits one "extensions" member has to fail on this seed.
		var members map[string]json.RawMessage
		if json.Unmarshal(raw, &members) == nil {
			members["extensions"] = json.RawMessage(`{}`)
			if tampered, err := json.Marshal(members); err == nil {
				f.Add("hello", tampered)
			}
		}
	}
	f.Fuzz(func(t *testing.T, operation string, body []byte) {
		// The envelope cannot represent a non-UTF-8 operation: JSON
		// encoding replaces the offending bytes, so identity between
		// the fuzz input and the decoded value is unstatable there.
		if !utf8.ValidString(operation) {
			return
		}
		line := fuzzLine(operation, body)
		if line == nil {
			return
		}
		r, err := rpcwire.DecodeRequest(line)
		if err != nil {
			return
		}
		if r.Operation() != operation || r.ID() != fuzzEnvelopeID || r.Version() != "2.0.0" {
			t.Fatalf("decoded request reinterprets the envelope: op=%q id=%q version=%q", r.Operation(), r.ID(), r.Version())
		}
		if r.Body() == nil {
			t.Fatal("admitted request carries no body")
		}
		var bodyObj map[string]json.RawMessage
		if json.Unmarshal(r.Body(), &bodyObj) != nil || bodyObj == nil {
			t.Fatal("admitted request body is not a JSON object")
		}
		if _, err := canonicaljson.Canonicalize(r.Body()); err != nil {
			t.Fatalf("admitted request body is outside the I-JSON data model: %v", err)
		}
		round, err := rpcwire.EncodeRequest(r.Version(), r.ID(), r.Operation(), r.Body())
		if err != nil {
			t.Fatalf("admitted request does not round-trip: %v", err)
		}
		if _, err = rpcwire.DecodeRequest(round); err != nil {
			t.Fatalf("round-tripped request refused: %v", err)
		}
		switch operation {
		case "hello":
			if _, err = r.Hello(); err != nil {
				t.Fatalf("admitted hello body has no typed decode: %v", err)
			}
			if !exactKeys(r.Body(), "host_id", "platform", "ax_version", "nonce", "contracts", "max_line_bytes", "max_object_bytes") {
				t.Fatal("admitted hello body is not the exact member set")
			}
		case "inventory.roots":
			if !exactKeys(r.Body(), "namespaces") {
				t.Fatal("admitted inventory.roots body is not the exact member set")
			}
		}
	})
}

// specNamespaces is the Section 11.3/11.8/11.9 namespace vocabulary,
// transcribed from the pinned specification text rather than derived from
// the production table, so drift in either direction fails the target.
var specNamespaces = map[string][]string{
	"2.0.0": {"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"},
	"3.0.0": {"blob", "directory_record", "event", "manifest", "record", "tombstone", "tombstone_ack"},
	"4.0.0": {"blob", "directory_record", "event", "manifest", "record", "terminal_backend_evidence", "tombstone", "tombstone_ack"},
	"5.0.0": {"blob", "directory_record", "event", "manifest", "record", "terminal_backend_evidence", "tombstone", "tombstone_ack"},
}

// FuzzNamespaceVocabulary drives inventory.roots request bodies: decode must
// admit exactly the sorted-unique in-vocabulary lists for the framing major
// and every admitted request must round-trip.
//
// Control: the namespace-member narrowing (mutations.py) admits the
// "credential" namespace; with that plant applied the oracle below fails on
// the seeded bad input. See the producer evidence log.
func FuzzNamespaceVocabulary(f *testing.F) {
	for version, names := range specNamespaces {
		raw, _ := json.Marshal(map[string]any{"namespaces": names})
		f.Add(version, raw)
		f.Add(version, []byte(`{"namespaces":[]}`))
		f.Add(version, []byte(`{"namespaces":["credential"]}`))
		f.Add(version, []byte(`{"namespaces":["event","blob"]}`))
		f.Add(version, []byte(`{"namespaces":["blob","blob"]}`))
	}
	f.Add("2.0.0", []byte(`{"namespaces":["directory_record"]}`))
	f.Add("6.0.0", []byte(`{"namespaces":["blob"]}`))
	f.Fuzz(func(t *testing.T, version string, body []byte) {
		allowed, pinned := specNamespaces[version]
		line, err := json.Marshal(map[string]any{
			"protocol":         rpcwire.Protocol,
			"protocol_version": version,
			"request_id":       fuzzEnvelopeID,
			"operation":        "inventory.roots",
			"body":             json.RawMessage(validObjectOrEmpty(body)),
		})
		if err != nil {
			return
		}
		var parsed struct {
			Namespaces []string `json:"namespaces"`
		}
		oracleValid := pinned && json.Unmarshal(validObjectOrEmpty(body), &parsed) == nil &&
			len(parsed.Namespaces) >= 1 && len(parsed.Namespaces) <= len(allowed) &&
			slices.IsSorted(parsed.Namespaces) && len(slices.Compact(slices.Clone(parsed.Namespaces))) == len(parsed.Namespaces)
		if oracleValid {
			for _, name := range parsed.Namespaces {
				if !slices.Contains(allowed, name) {
					oracleValid = false
					break
				}
			}
		}
		// The oracle is only consulted when the body is exactly the
		// single-member shape; anything else is invalid by construction.
		if !exactKeys(validObjectOrEmpty(body), "namespaces") {
			oracleValid = false
		}
		r, err := rpcwire.DecodeRequest(line)
		if err != nil {
			if oracleValid {
				t.Fatalf("valid %s namespaces %q refused: %v", version, parsed.Namespaces, err)
			}
			return
		}
		if !oracleValid {
			t.Fatalf("invalid %s namespaces admitted: %q", version, body)
		}
		round, err := rpcwire.EncodeRequest(r.Version(), r.ID(), r.Operation(), r.Body())
		if err != nil {
			t.Fatalf("admitted namespaces do not round-trip: %v", err)
		}
		if _, err = rpcwire.DecodeRequest(round); err != nil {
			t.Fatalf("round-tripped namespaces refused: %v", err)
		}
	})
}

// FuzzUnknownFields drives raw envelope lines: whatever decode admits must
// carry exactly the closed member sets at the envelope and (for parsed
// operations) body layers, and must round-trip through the production
// encode entry.
//
// Control: the closed-extra narrowing (mutations.py) admits one extra=true
// envelope member; with that plant applied the envelope property below
// fails on the seeded bad input. See the producer evidence log.
func FuzzUnknownFields(f *testing.F) {
	valid := []byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"2.0.0","request_id":"` + fuzzEnvelopeID + `","operation":"health.get","body":{}}`)
	f.Add(valid)
	f.Add([]byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"2.0.0","request_id":"` + fuzzEnvelopeID + `","operation":"health.get","body":{},"extra":true}`))
	f.Add([]byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"2.0.0","request_id":"` + fuzzEnvelopeID + `","operation":"health.get","body":{"extensions":{}},"extensions":{}}`))
	f.Add([]byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"2.0.0","request_id":"` + fuzzEnvelopeID + `","operation":"inventory.roots","body":{"namespaces":["blob"],"extensions":{}}}`))
	f.Add([]byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"2.0.0","request_id":"not-a-uuid","operation":"health.get","body":{}}`))
	if raw, err := os.ReadFile("testdata/hello-request-v2.json"); err == nil {
		compact := &bytes.Buffer{}
		if json.Compact(compact, bytes.TrimSpace(raw)) == nil {
			f.Add(compact.Bytes())
		}
	}
	// A hello body carrying one extra member (P3-3): decode must refuse,
	// and the exact-keys oracle below must fail if a plant admits it.
	if raw, err := validHelloBody(); err == nil {
		var members map[string]json.RawMessage
		if json.Unmarshal(raw, &members) == nil {
			members["extensions"] = json.RawMessage(`{}`)
			if tampered, err := json.Marshal(members); err == nil {
				f.Add([]byte(`{"protocol":"` + rpcwire.Protocol + `","protocol_version":"2.0.0","request_id":"` + fuzzEnvelopeID + `","operation":"hello","body":` + string(tampered) + `}`))
			}
		}
	}
	f.Fuzz(func(t *testing.T, line []byte) {
		r, err := rpcwire.DecodeRequest(line)
		if err != nil {
			return
		}
		var envelope map[string]json.RawMessage
		if json.Unmarshal(line, &envelope) != nil {
			t.Fatal("admitted line is not a JSON object")
		}
		for _, key := range []string{"protocol", "protocol_version", "request_id", "operation", "body"} {
			if _, ok := envelope[key]; !ok {
				t.Fatalf("admitted envelope misses %q", key)
			}
		}
		if len(envelope) != 5 {
			t.Fatalf("admitted envelope carries %d members, want exactly 5", len(envelope))
		}
		switch r.Operation() {
		case "hello":
			if !exactKeys(r.Body(), "host_id", "platform", "ax_version", "nonce", "contracts", "max_line_bytes", "max_object_bytes") {
				t.Fatal("admitted hello body is not the exact member set")
			}
		case "inventory.roots":
			if !exactKeys(r.Body(), "namespaces") {
				t.Fatal("admitted inventory.roots body is not the exact member set")
			}
		}
		round, err := rpcwire.EncodeRequest(r.Version(), r.ID(), r.Operation(), r.Body())
		if err != nil {
			t.Fatalf("admitted line does not round-trip: %v", err)
		}
		if _, err = rpcwire.DecodeRequest(round); err != nil {
			t.Fatalf("round-tripped line refused: %v", err)
		}
	})
}
