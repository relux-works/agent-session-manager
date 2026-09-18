package rpcwire_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

// TestClosedBodyMemberSweep drives the closed-shape contract of every
// operation body the landed code parses (hello and inventory.roots, Sections
// 11.2-11.3) through the production DecodeRequest entry: a missing member,
// an extra top-level member, and a member of the wrong type each refuse with
// the pinned sentinel. Every other operation of the Section 11.3 registry is
// opaque by design (TestOpaqueOperationBoundary) and is a stated bound, not
// a row here.
func TestClosedBodyMemberSweep(t *testing.T) {
	helloMembers := []string{"host_id", "platform", "ax_version", "nonce", "contracts", "max_line_bytes", "max_object_bytes"}
	wrongTypes := map[string]any{
		"host_id":          42,
		"platform":         true,
		"ax_version":       []any{"0.2.1"},
		"nonce":            42,
		"contracts":        "rpc",
		"max_line_bytes":   "8388608",
		"max_object_bytes": true,
	}
	for _, member := range helloMembers {
		t.Run("hello-missing-"+member, func(t *testing.T) {
			m := hello(t, "2.0.0", false)
			delete(m["body"].(map[string]any), member)
			if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
				t.Fatalf("missing %s: err = %v, want ErrHello", member, err)
			}
		})
		t.Run("hello-wrongtype-"+member, func(t *testing.T) {
			m := hello(t, "2.0.0", false)
			m["body"].(map[string]any)[member] = wrongTypes[member]
			if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
				t.Fatalf("wrong-type %s: err = %v, want ErrHello", member, err)
			}
		})
	}
	t.Run("hello-extra-member", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		m["body"].(map[string]any)["future"] = true
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("extra member: err = %v, want ErrHello", err)
		}
	})
	t.Run("inventory-missing-namespaces", func(t *testing.T) {
		m := inventoryRequest(t, "2.0.0", map[string]any{"other": []any{"blob"}})
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrInventory) {
			t.Fatalf("err = %v, want ErrInventory", err)
		}
	})
	t.Run("inventory-extra-member", func(t *testing.T) {
		m := inventoryRequest(t, "2.0.0", map[string]any{"namespaces": []any{"blob"}, "future": true})
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrInventory) {
			t.Fatalf("err = %v, want ErrInventory", err)
		}
	})
	for _, wrong := range []any{"blob", 42.0, true, nil, map[string]any{"blob": true}} {
		t.Run(fmt.Sprintf("inventory-wrongtype-%T", wrong), func(t *testing.T) {
			m := inventoryRequest(t, "2.0.0", map[string]any{"namespaces": wrong})
			if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrInventory) {
				t.Fatalf("err = %v, want ErrInventory", err)
			}
		})
	}
}

func inventoryRequest(t *testing.T, version string, body map[string]any) map[string]any {
	t.Helper()
	return map[string]any{
		"protocol":         rpcwire.Protocol,
		"protocol_version": version,
		"request_id":       id,
		"operation":        "inventory.roots",
		"body":             body,
	}
}

// TestClosedBodyBoundsWitnessed pins every bound of the parsed bodies at both
// edges and one step outside through DecodeRequest, plus the independence of
// max_object_bytes from max_line_bytes: each floor refuses on its own while
// the other is satisfied.
func TestClosedBodyBoundsWitnessed(t *testing.T) {
	nonce16 := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 16))
	t.Run("nonce-16-bytes-admitted", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		m["body"].(map[string]any)["nonce"] = nonce16
		if _, err := rpcwire.DecodeRequest(wire(t, m)); err != nil {
			t.Fatalf("16-byte nonce refused: %v", err)
		}
	})
	t.Run("nonce-15-bytes-refused", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		m["body"].(map[string]any)["nonce"] = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 15))
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("15-byte nonce: err = %v, want ErrHello", err)
		}
	})
	t.Run("nonce-standard-alphabet-refused", func(t *testing.T) {
		// 18 bytes of 0xfb: standard encoding carries '+' with no padding,
		// so only the alphabet (not length or padding) is at issue.
		raw := bytes.Repeat([]byte{0xfb}, 18)
		std := base64.StdEncoding.EncodeToString(raw)
		if !strings.Contains(std, "+") || strings.Contains(std, "=") {
			t.Fatalf("fixture carries no isolated alphabet witness: %q", std)
		}
		m := hello(t, "2.0.0", false)
		m["body"].(map[string]any)["nonce"] = std
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("standard-alphabet nonce: err = %v, want ErrHello", err)
		}
	})
	t.Run("nonce-padded-refused", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		m["body"].(map[string]any)["nonce"] = base64.URLEncoding.EncodeToString(bytes.Repeat([]byte{0x01}, 16))
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("padded nonce: err = %v, want ErrHello", err)
		}
	})
	t.Run("line-floor-independent", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		body := m["body"].(map[string]any)
		body["max_line_bytes"] = 8388607
		body["max_object_bytes"] = 7 * 1024 * 1024
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("low line limit with generous object limit: err = %v, want ErrHello", err)
		}
	})
	t.Run("object-floor-independent", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		body := m["body"].(map[string]any)
		body["max_line_bytes"] = 10 * 1024 * 1024
		body["max_object_bytes"] = 5242879
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("low object limit with generous line limit: err = %v, want ErrHello", err)
		}
	})
	t.Run("both-floors-admitted", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		body := m["body"].(map[string]any)
		body["max_line_bytes"] = 8388608
		body["max_object_bytes"] = 5242880
		if _, err := rpcwire.DecodeRequest(wire(t, m)); err != nil {
			t.Fatalf("exact floors refused: %v", err)
		}
	})
	t.Run("contracts-16-admitted-17-refused", func(t *testing.T) {
		admit := hello(t, "2.0.0", false)
		versions := make([]string, 0, 16)
		for i := range 16 {
			versions = append(versions, fmt.Sprintf("1.%d.0", i))
		}
		slices.Sort(versions)
		admit["body"].(map[string]any)["contracts"].(map[string]any)["lease"] = versions
		if _, err := rpcwire.DecodeRequest(wire(t, admit)); err != nil {
			t.Fatalf("16 versions refused: %v", err)
		}
		refuse := hello(t, "2.0.0", false)
		seventeen := make([]string, 0, 17)
		for i := range 17 {
			seventeen = append(seventeen, fmt.Sprintf("1.%d.0", i))
		}
		slices.Sort(seventeen)
		refuse["body"].(map[string]any)["contracts"].(map[string]any)["lease"] = seventeen
		if _, err := rpcwire.DecodeRequest(wire(t, refuse)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("17 versions: err = %v, want ErrHello", err)
		}
	})
}

// TestNamespaceVocabularyPerMember admits every namespace of each major's
// pinned vocabulary singly and refuses near-miss spellings and
// future-major members through DecodeRequest.
func TestNamespaceVocabularyPerMember(t *testing.T) {
	vocabularies := map[string][]string{
		"2.0.0": {"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"},
		"3.0.0": {"blob", "directory_record", "event", "manifest", "record", "tombstone", "tombstone_ack"},
		"4.0.0": {"blob", "directory_record", "event", "manifest", "record", "terminal_backend_evidence", "tombstone", "tombstone_ack"},
		"5.0.0": {"blob", "directory_record", "event", "manifest", "record", "terminal_backend_evidence", "tombstone", "tombstone_ack"},
	}
	for _, version := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		names, err := rpcwire.Namespaces(version)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(names, vocabularies[version]) {
			t.Fatalf("Namespaces(%s) = %v, want the spec vocabulary %v", version, names, vocabularies[version])
		}
		for _, name := range vocabularies[version] {
			t.Run(version+"/admit-"+name, func(t *testing.T) {
				m := inventoryRequest(t, version, map[string]any{"namespaces": []any{name}})
				if _, err := rpcwire.DecodeRequest(wire(t, m)); err != nil {
					t.Fatalf("namespace %q refused in %s: %v", name, version, err)
				}
			})
		}
	}
	nearMiss := []string{"Record", "RECORD", " record", "record ", "blob\n", "", "tombstone-ack", "tombstoneack"}
	for _, version := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		for _, name := range nearMiss {
			t.Run(version+"/refuse-"+fmt.Sprintf("%q", name), func(t *testing.T) {
				m := inventoryRequest(t, version, map[string]any{"namespaces": []any{name}})
				if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrInventory) {
					t.Fatalf("namespace %q in %s: err = %v, want ErrInventory", name, version, err)
				}
			})
		}
	}
	for _, tc := range []struct{ version, name string }{
		{"2.0.0", "directory_record"},
		{"2.0.0", "terminal_backend_evidence"},
		{"3.0.0", "terminal_backend_evidence"},
	} {
		t.Run(tc.version+"/refuse-future-"+tc.name, func(t *testing.T) {
			m := inventoryRequest(t, tc.version, map[string]any{"namespaces": []any{tc.name}})
			if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrInventory) {
				t.Fatalf("err = %v, want ErrInventory", err)
			}
		})
	}
}

// TestNamespaceMismatchMatrix proves a root reported under the wrong
// namespace refuses no matter which valid namespace lands in which slot:
// every off-diagonal placement of the v2 vocabulary is refused through
// EncodeSuccess, and the diagonal admits.
func TestNamespaceMismatchMatrix(t *testing.T) {
	names := []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"}
	m := inventoryRequest(t, "2.0.0", map[string]any{"namespaces": names})
	r, err := rpcwire.DecodeRequest(wire(t, m))
	if err != nil {
		t.Fatal(err)
	}
	roots := func(order []string) json.RawMessage {
		items := make([]any, 0, len(order))
		for _, name := range order {
			items = append(items, map[string]any{"namespace": name, "count": 0, "root_id": "sha256:" + strings.Repeat("0", 64)})
		}
		raw, err := json.Marshal(map[string]any{"roots": items})
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	if _, err := rpcwire.EncodeSuccess(r, roots(names)); err != nil {
		t.Fatalf("diagonal roots refused: %v", err)
	}
	for i := range names {
		for j := range names {
			if i == j {
				continue
			}
			t.Run(fmt.Sprintf("slot-%d-carries-%s", i, names[j]), func(t *testing.T) {
				misplaced := slices.Clone(names)
				misplaced[i] = names[j]
				if _, err := rpcwire.EncodeSuccess(r, roots(misplaced)); !errors.Is(err, rpcwire.ErrInventory) {
					t.Fatalf("err = %v, want ErrInventory", err)
				}
			})
		}
	}
}

// TestUnknownFieldsRefusedEverywhere drives one extra member through every
// closed shape the landed code parses: the request envelope, the hello
// request and success bodies, the inventory request and success bodies, and
// both failure-shape directions.
func TestUnknownFieldsRefusedEverywhere(t *testing.T) {
	t.Run("envelope", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		m["extensions"] = map[string]any{"works.relux.ax.future": map[string]any{"a": 1}}
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrFrame) {
			t.Fatalf("err = %v, want ErrFrame", err)
		}
	})
	t.Run("hello-request-body", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		m["body"].(map[string]any)["extensions"] = map[string]any{}
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("err = %v, want ErrHello", err)
		}
	})
	t.Run("hello-response-body", func(t *testing.T) {
		r := request(t, hello(t, "2.0.0", false))
		m := hello(t, "2.0.0", true)
		m["body"].(map[string]any)["extensions"] = map[string]any{}
		if _, err := rpcwire.DecodeResponse(wire(t, m), r); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("err = %v, want ErrHello", err)
		}
	})
	t.Run("inventory-request-body", func(t *testing.T) {
		m := inventoryRequest(t, "2.0.0", map[string]any{"namespaces": []any{"blob"}, "extensions": map[string]any{}})
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrInventory) {
			t.Fatalf("err = %v, want ErrInventory", err)
		}
	})
	t.Run("inventory-success-body", func(t *testing.T) {
		m := inventoryRequest(t, "2.0.0", map[string]any{"namespaces": []any{"blob"}})
		r, err := rpcwire.DecodeRequest(wire(t, m))
		if err != nil {
			t.Fatal(err)
		}
		body := map[string]any{
			"roots":      []any{map[string]any{"namespace": "blob", "count": 0, "root_id": "sha256:" + strings.Repeat("0", 64)}},
			"extensions": map[string]any{},
		}
		if _, err := rpcwire.EncodeSuccess(r, wire(t, body)); !errors.Is(err, rpcwire.ErrInventory) {
			t.Fatalf("err = %v, want ErrInventory", err)
		}
	})
	t.Run("failure-carries-body", func(t *testing.T) {
		r := request(t, hello(t, "2.0.0", false))
		m := map[string]any{
			"protocol":         rpcwire.Protocol,
			"protocol_version": "2.0.0",
			"request_id":       id,
			"ok":               false,
			"error":            map[string]any{"schema": "urn:ax:schema:error", "schema_version": "1.0.0", "code": "incompatible_protocol", "message": "x", "exit_code": 6, "retryable": false, "details": map[string]any{}},
			"body":             map[string]any{},
		}
		if _, err := rpcwire.DecodeResponse(wire(t, m), r); !errors.Is(err, rpcwire.ErrFrame) {
			t.Fatalf("err = %v, want ErrFrame", err)
		}
	})
	t.Run("success-carries-error", func(t *testing.T) {
		r := request(t, hello(t, "2.0.0", false))
		m := hello(t, "2.0.0", true)
		m["error"] = nil
		if _, err := rpcwire.DecodeResponse(wire(t, m), r); !errors.Is(err, rpcwire.ErrFrame) {
			t.Fatalf("err = %v, want ErrFrame", err)
		}
	})
}

// TestExtensionsDirections proves both directions of the Section 17.1
// extension rule at the RPC layer: the same namespaced data refused at the
// envelope top level is admitted byte-identical inside an opaque operation
// body, while the closed parsed bodies stay closed.
func TestExtensionsDirections(t *testing.T) {
	namespaced := map[string]any{"works.relux.ax.future": map[string]any{"a": 1}}
	t.Run("envelope-top-level-refused", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		m["extensions"] = namespaced
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrFrame) {
			t.Fatalf("err = %v, want ErrFrame", err)
		}
	})
	t.Run("hello-body-refused", func(t *testing.T) {
		m := hello(t, "2.0.0", false)
		m["body"].(map[string]any)["extensions"] = namespaced
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrHello) {
			t.Fatalf("err = %v, want ErrHello", err)
		}
	})
	t.Run("inventory-body-refused", func(t *testing.T) {
		m := inventoryRequest(t, "2.0.0", map[string]any{"namespaces": []any{"blob"}, "extensions": namespaced})
		if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrInventory) {
			t.Fatalf("err = %v, want ErrInventory", err)
		}
	})
	t.Run("opaque-body-admitted-identical", func(t *testing.T) {
		body := wire(t, map[string]any{"extensions": namespaced})
		line, err := rpcwire.EncodeRequest("2.0.0", id, "health.get", body)
		if err != nil {
			t.Fatal(err)
		}
		r, err := rpcwire.DecodeRequest(line)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(r.Body(), body) {
			t.Fatal("opaque extension body altered")
		}
	})
}

// TestOpaqueBodyMustBeObject pins the object gate (envelope.go DecodeRequest)
// for operations no body parser reads: a body that is not an I-JSON object —
// a valid non-object JSON value, or an object with a duplicate member — is
// refused with ErrFrame before any operation dispatch. The hello lane proves
// nothing about this gate (decodeHello refuses a non-object body anyway),
// so the witness runs over the opaque health.get operation.
func TestOpaqueBodyMustBeObject(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"array", `[]`},
		{"null", `null`},
		{"string", `"x"`},
		{"number", `42`},
		{"duplicate-member", `{"a":1,"a":2}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			line := []byte(`{"protocol":"` + rpcwire.Protocol + `","protocol_version":"2.0.0","request_id":"` + id + `","operation":"health.get","body":` + tc.body + `}`)
			if _, err := rpcwire.DecodeRequest(line); !errors.Is(err, rpcwire.ErrFrame) {
				t.Fatalf("opaque body %s: err = %v, want ErrFrame", tc.body, err)
			}
		})
	}
}

// TestResponseCorrelationClasses pins the refusal class of every
// correlation failure through DecodeResponse: a mismatched echo (ID or
// hello nonce echo) is ErrCorrelation, a version skew is ErrVersion, and a
// zero expectation can correlate nothing.
func TestResponseCorrelationClasses(t *testing.T) {
	r := request(t, hello(t, "2.0.0", false))
	t.Run("mismatched-id", func(t *testing.T) {
		m := hello(t, "2.0.0", true)
		m["request_id"] = otherID
		if _, err := rpcwire.DecodeResponse(wire(t, m), r); !errors.Is(err, rpcwire.ErrCorrelation) {
			t.Fatalf("err = %v, want ErrCorrelation", err)
		}
	})
	t.Run("mismatched-version", func(t *testing.T) {
		m := hello(t, "3.0.0", true)
		if _, err := rpcwire.DecodeResponse(wire(t, m), r); !errors.Is(err, rpcwire.ErrVersion) {
			t.Fatalf("err = %v, want ErrVersion", err)
		}
	})
	t.Run("mismatched-nonce-echo", func(t *testing.T) {
		m := hello(t, "2.0.0", true)
		m["body"].(map[string]any)["nonce_echo"] = "cXJzdHV2d3h5ejAxMjM0NQ"
		if _, err := rpcwire.DecodeResponse(wire(t, m), r); !errors.Is(err, rpcwire.ErrCorrelation) {
			t.Fatalf("err = %v, want ErrCorrelation", err)
		}
	})
	t.Run("zero-expectation", func(t *testing.T) {
		m := hello(t, "2.0.0", true)
		if _, err := rpcwire.DecodeResponse(wire(t, m), rpcwire.Request{}); !errors.Is(err, rpcwire.ErrCorrelation) {
			t.Fatalf("err = %v, want ErrCorrelation", err)
		}
	})
}

// TestMajor1RejectedNotReinterpreted proves a 2.0.0 peer rejects major 1
// with the version class at every rpcwire entry instead of coercing it.
func TestMajor1RejectedNotReinterpreted(t *testing.T) {
	m := hello(t, "2.0.0", false)
	m["protocol_version"] = "1.0.0"
	if _, err := rpcwire.DecodeRequest(wire(t, m)); !errors.Is(err, rpcwire.ErrVersion) {
		t.Fatalf("DecodeRequest(1.0.0): err = %v, want ErrVersion", err)
	}
	if _, err := rpcwire.EncodeRequest("1.0.0", id, "health.get", []byte(`{}`)); !errors.Is(err, rpcwire.ErrVersion) {
		t.Fatalf("EncodeRequest(1.0.0): err = %v, want ErrVersion", err)
	}
	if _, err := rpcwire.ContractProfile("1.0.0"); !errors.Is(err, rpcwire.ErrVersion) {
		t.Fatalf("ContractProfile(1.0.0): err = %v, want ErrVersion", err)
	}
	if _, err := rpcwire.Namespaces("1.0.0"); !errors.Is(err, rpcwire.ErrVersion) {
		t.Fatalf("Namespaces(1.0.0): err = %v, want ErrVersion", err)
	}
	r := request(t, hello(t, "2.0.0", false))
	resp := hello(t, "2.0.0", true)
	resp["protocol_version"] = "1.0.0"
	if _, err := rpcwire.DecodeResponse(wire(t, resp), r); !errors.Is(err, rpcwire.ErrVersion) {
		t.Fatalf("DecodeResponse(1.0.0): err = %v, want ErrVersion", err)
	}
}

// TestErrorSchemaNotNegotiated drives a peer that tries to negotiate the
// Structured Error schema: a failure document carrying a foreign
// schema_version is refused through DecodeResponse even though the document
// itself is well-formed, because the version is fixed by the containing
// protocol (RPC 2 binds Error 1.0.0, RPC 4/5 bind Error 1.3.0). Each refusal
// lane pins the ErrVersionMismatch class returned through
// axerror.DecodeBound, so a shape error cannot stand in for the binding
// refusal.
func TestErrorSchemaNotNegotiated(t *testing.T) {
	build := func(t *testing.T, version axerror.Version) map[string]any {
		t.Helper()
		failure, err := axerror.New(axerror.Spec{Version: version, Code: "incompatible_protocol", Message: "peer framed", Details: axerror.Details{}})
		if err != nil {
			t.Fatal(err)
		}
		raw, err := failure.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	frame := func(t *testing.T, rpcVersion, requestID string, doc map[string]any) []byte {
		t.Helper()
		return wire(t, map[string]any{
			"protocol":         rpcwire.Protocol,
			"protocol_version": rpcVersion,
			"request_id":       requestID,
			"ok":               false,
			"error":            doc,
		})
	}
	t.Run("rpc2-refuses-error-1.3.0", func(t *testing.T) {
		r := request(t, hello(t, "2.0.0", false))
		if _, err := rpcwire.DecodeResponse(frame(t, "2.0.0", id, build(t, axerror.Version130)), r); !errors.Is(err, axerror.ErrVersionMismatch) {
			t.Fatalf("err = %v, want ErrVersionMismatch", err)
		}
	})
	t.Run("rpc5-refuses-error-1.0.0", func(t *testing.T) {
		r := request(t, hello(t, "5.0.0", false))
		if _, err := rpcwire.DecodeResponse(frame(t, "5.0.0", id, build(t, axerror.Version100)), r); !errors.Is(err, axerror.ErrVersionMismatch) {
			t.Fatalf("err = %v, want ErrVersionMismatch", err)
		}
	})
	t.Run("rpc4-refuses-error-1.0.0", func(t *testing.T) {
		r := request(t, hello(t, "4.0.0", false))
		if _, err := rpcwire.DecodeResponse(frame(t, "4.0.0", id, build(t, axerror.Version100)), r); !errors.Is(err, axerror.ErrVersionMismatch) {
			t.Fatalf("err = %v, want ErrVersionMismatch", err)
		}
	})
	t.Run("bound-versions-admit", func(t *testing.T) {
		for _, tc := range []struct {
			rpc   string
			error axerror.Version
		}{{"2.0.0", axerror.Version100}, {"4.0.0", axerror.Version130}, {"5.0.0", axerror.Version130}} {
			r := request(t, hello(t, tc.rpc, false))
			resp, err := rpcwire.DecodeResponse(frame(t, tc.rpc, id, build(t, tc.error)), r)
			if err != nil || resp.OK() {
				t.Fatalf("RPC %s with its bound error: resp=%v err=%v", tc.rpc, resp.OK(), err)
			}
		}
	})
}

// TestFramingBounds witnesses the Section 11.2 line discipline through the
// production entries: exactly 8388608 bytes accepted and 8388609 refused on
// both the decode and encode paths, an embedded newline refused, and valid
// JSON that is not an object refused.
func TestFramingBounds(t *testing.T) {
	good := wire(t, hello(t, "2.0.0", false))
	t.Run("decode-accepts-8388608", func(t *testing.T) {
		exact := append(bytes.Clone(good), bytes.Repeat([]byte(" "), rpcwire.MaxLineBytes-len(good))...)
		if len(exact) != 8388608 {
			t.Fatalf("line = %d bytes, want 8388608", len(exact))
		}
		if _, err := rpcwire.DecodeRequest(exact); err != nil {
			t.Fatalf("8388608 refused: %v", err)
		}
	})
	t.Run("decode-refuses-8388609", func(t *testing.T) {
		over := append(bytes.Clone(good), bytes.Repeat([]byte(" "), rpcwire.MaxLineBytes+1-len(good))...)
		if len(over) != 8388609 {
			t.Fatalf("line = %d bytes, want 8388609", len(over))
		}
		if _, err := rpcwire.DecodeRequest(over); !errors.Is(err, rpcwire.ErrFrame) {
			t.Fatalf("err = %v, want ErrFrame", err)
		}
	})
	t.Run("encode-refuses-8388609", func(t *testing.T) {
		// Size the body from a measured probe so the framed line lands
		// exactly one byte over the cap: the pad grows the line byte
		// for byte ('x' needs no JSON escaping and the body embeds
		// verbatim), so pad-1 must frame a measurable 8388608-byte
		// line while pad refuses.
		probe, err := rpcwire.EncodeRequest("2.0.0", id, "health.get", []byte(`{"pad":""}`))
		if err != nil {
			t.Fatal(err)
		}
		pad := rpcwire.MaxLineBytes + 1 - len(probe)
		admit, err := rpcwire.EncodeRequest("2.0.0", id, "health.get", wire(t, map[string]any{"pad": strings.Repeat("x", pad-1)}))
		if err != nil {
			t.Fatalf("encode below the cap refused: %v", err)
		}
		if len(admit) != 8388608 {
			t.Fatalf("admitted encoded line = %d bytes, want 8388608", len(admit))
		}
		body := wire(t, map[string]any{"pad": strings.Repeat("x", pad)})
		if _, err := rpcwire.EncodeRequest("2.0.0", id, "health.get", body); !errors.Is(err, rpcwire.ErrFrame) {
			t.Fatalf("err = %v, want ErrFrame", err)
		}
	})
	t.Run("embedded-newline-refused", func(t *testing.T) {
		split := append(bytes.Clone(good[:len(good)/2]), append([]byte("\n"), good[len(good)/2:]...)...)
		if _, err := rpcwire.DecodeRequest(split); !errors.Is(err, rpcwire.ErrFrame) {
			t.Fatalf("err = %v, want ErrFrame", err)
		}
	})
	for _, line := range []string{`null`, `[]`, `"health.get"`, `42`, `true`} {
		t.Run("non-object-"+line, func(t *testing.T) {
			if _, err := rpcwire.DecodeRequest([]byte(line)); !errors.Is(err, rpcwire.ErrFrame) {
				t.Fatalf("err = %v, want ErrFrame", err)
			}
		})
	}
}
