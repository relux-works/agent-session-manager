package rpcwire_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const id = "0198f4c8-a070-7188-9172-1234567890ab"
const otherID = "0198f4c8-a070-7188-9172-2234567890ab"

func wire(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func fixture(t *testing.T, name string) map[string]any {
	t.Helper()
	b, e := os.ReadFile("testdata/" + name + ".json")
	if e != nil {
		t.Fatal(e)
	}
	var m map[string]any
	if e = json.Unmarshal(b, &m); e != nil {
		t.Fatal(e)
	}
	return m
}
func hello(t *testing.T, version string, response bool) map[string]any {
	name := "hello-request-v2"
	if response {
		name = "hello-response-v2"
	}
	m := fixture(t, name)
	m["protocol_version"] = version
	if version != "2.0.0" {
		m["body"].(map[string]any)["contracts"] = fixture(t, "contracts-v"+version[:1])
	}
	return m
}
func request(t *testing.T, m map[string]any) rpcwire.Request {
	t.Helper()
	r, e := rpcwire.DecodeRequest(wire(t, m))
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestPinnedHelloProfiles(t *testing.T) {
	for _, v := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		t.Run(v, func(t *testing.T) {
			q := hello(t, v, false)
			r := request(t, q)
			if r.ID() != id || r.Version() != v || r.Operation() != "hello" {
				t.Fatal("wrong envelope")
			}
			h, e := r.Hello()
			if e != nil {
				t.Fatal(e)
			}
			profile, e := rpcwire.ContractProfile(v)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(wire(t, profile), wire(t, h.Contracts)) {
				t.Fatal("profile differs from normative fixture")
			}
			encoded, e := rpcwire.EncodeRequest(v, id, "hello", r.Body())
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(encoded, wire(t, q)) {
				t.Fatal("request fixture roundtrip")
			}
			p := hello(t, v, true)
			resp, e := rpcwire.DecodeResponse(wire(t, p), r)
			if e != nil {
				t.Fatal(e)
			}
			if !resp.OK() || resp.Failure() != nil {
				t.Fatal("success shape")
			}
			body, e := resp.Hello()
			if e != nil || body.HostID == h.HostID {
				t.Fatal("response identity")
			}
			encoded, e = rpcwire.EncodeSuccess(r, resp.Body())
			if e != nil || !bytes.Equal(encoded, wire(t, p)) {
				t.Fatal("response fixture roundtrip", e)
			}
			limits, e := rpcwire.OfferedLimits(r, resp)
			if e != nil || limits.LineBytes != 8388608 || limits.ObjectBytes != 5242880 {
				t.Fatal(limits, e)
			}
			// Copies are not live capability or identity handles.
			h.Contracts["rpc"][0] = "9.0.0"
			profile["rpc"][0] = "8.0.0"
			b := r.Body()
			b[0] = '!'
			b = resp.Body()
			b[0] = '!'
			if _, e = r.Hello(); e != nil {
				t.Fatal("request alias", e)
			}
			if _, e = resp.Hello(); e != nil {
				t.Fatal("response alias", e)
			}
		})
	}
}
func TestRequestRefusals(t *testing.T) {
	tests := []struct {
		name string
		edit func(map[string]any)
	}{
		{"protocol", func(m map[string]any) { m["protocol"] = "urn:other" }},
		{"missing-version", func(m map[string]any) { delete(m, "protocol_version") }},
		{"major", func(m map[string]any) { m["protocol_version"] = "6.0.0" }},
		{"unknown-minor", func(m map[string]any) { m["protocol_version"] = "2.1.0" }},
		{"uuid", func(m map[string]any) { m["request_id"] = "0198f4c8-a070-4188-9172-1234567890ab" }},
		{"missing-id", func(m map[string]any) { delete(m, "request_id") }},
		{"extra", func(m map[string]any) { m["extra"] = true }},
		{"case-alias", func(m map[string]any) { m["Operation"] = "hello"; delete(m, "operation") }},
		{"null-operation", func(m map[string]any) { m["operation"] = nil }},
		{"missing-body", func(m map[string]any) { delete(m, "body") }},
		{"array-body", func(m map[string]any) { m["body"] = []any{} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := hello(t, "2.0.0", false)
			tt.edit(m)
			if _, e := rpcwire.DecodeRequest(wire(t, m)); e == nil {
				t.Fatal("admitted invalid request")
			}
		})
	}
}
func TestFrameRefusals(t *testing.T) {
	good := wire(t, hello(t, "2.0.0", false))
	tests := map[string][]byte{
		"duplicate": append([]byte(`{"protocol":"urn:other",`), good[1:]...),
		"newline":   append(bytes.Clone(good), '\n'), "empty": {}, "null": []byte("null"),
		"trailing": append(bytes.Clone(good), []byte(" {}")...),
		"utf8":     bytes.Replace(good, []byte("macos"), []byte{'m', 0xff}, 1),
		"float":    bytes.Replace(good, []byte("8388608"), []byte("8388608.0"), 1),
		"oversize": append(good, bytes.Repeat([]byte(" "), rpcwire.MaxLineBytes+1-len(good))...),
	}
	for name, line := range tests {
		t.Run(name, func(t *testing.T) {
			if _, e := rpcwire.DecodeRequest(line); e == nil {
				t.Fatal("admitted invalid frame")
			}
		})
	}
	exact := append(bytes.Clone(good), bytes.Repeat([]byte(" "), rpcwire.MaxLineBytes-len(good))...)
	if _, e := rpcwire.DecodeRequest(exact); e != nil {
		t.Fatal("exact line boundary", e)
	}
	if _, e := rpcwire.DecodeRequest(append(good, '\r')); e != nil {
		t.Fatal("CRLF transport compatibility", e)
	}
}
func TestHelloRefusals(t *testing.T) {
	tests := []struct {
		name string
		edit func(map[string]any)
	}{
		{"host", func(b map[string]any) { b["host_id"] = otherID[:8] }},
		{"platform", func(b map[string]any) { b["platform"] = "ios" }},
		{"ax-version", func(b map[string]any) { b["ax_version"] = "01.0.0" }},
		{"nonce-short", func(b map[string]any) { b["nonce"] = base64.RawURLEncoding.EncodeToString(make([]byte, 15)) }},
		{"nonce-padding", func(b map[string]any) { b["nonce"] = "YWJjZGVmZ2hpamtsbW5vcA==" }},
		{"nonce-pad-bits", func(b map[string]any) { b["nonce"] = "YWJjZGVmZ2hpamtsbW5vcB" }},
		{"line-floor", func(b map[string]any) { b["max_line_bytes"] = 8388607 }},
		{"object-floor", func(b map[string]any) { b["max_object_bytes"] = 5242879 }},
		{"null-limit", func(b map[string]any) { b["max_line_bytes"] = nil }},
		{"extra-body", func(b map[string]any) { b["extra"] = true }},
		{"request-echo", func(b map[string]any) { b["nonce_echo"] = b["nonce"] }},
		{"error-key", func(b map[string]any) { b["contracts"].(map[string]any)["error"] = []string{"1.0.0"} }},
		{"missing-key", func(b map[string]any) { delete(b["contracts"].(map[string]any), "lease") }},
		{"replace-key", func(b map[string]any) {
			c := b["contracts"].(map[string]any)
			delete(c, "lease")
			c["config"] = []string{"1.0.0"}
		}},
		{"empty-versions", func(b map[string]any) { b["contracts"].(map[string]any)["lease"] = []string{} }},
		{"duplicate-versions", func(b map[string]any) { b["contracts"].(map[string]any)["lease"] = []string{"1.0.0", "1.0.0"} }},
		{"unsorted-versions", func(b map[string]any) { b["contracts"].(map[string]any)["lease"] = []string{"2.0.0", "1.0.0"} }},
		{"invalid-version", func(b map[string]any) { b["contracts"].(map[string]any)["lease"] = []string{"v1.0.0"} }},
		{"seventeen-versions", func(b map[string]any) {
			vs := []string{}
			for i := 0; i < 17; i++ {
				vs = append(vs, fmt.Sprintf("1.%d.0", i))
			}
			slices.Sort(vs)
			b["contracts"].(map[string]any)["lease"] = vs
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := hello(t, "2.0.0", false)
			tt.edit(m["body"].(map[string]any))
			if _, e := rpcwire.DecodeRequest(wire(t, m)); e == nil {
				t.Fatal("admitted invalid hello")
			}
		})
	}
	for _, v := range []string{"3.0.0", "4.0.0", "5.0.0"} {
		t.Run("exact-map-"+v, func(t *testing.T) {
			m := hello(t, v, false)
			m["body"].(map[string]any)["contracts"].(map[string]any)["lease"] = []string{"1.0.0", "1.1.0"}
			if _, e := rpcwire.DecodeRequest(wire(t, m)); e == nil {
				t.Fatal("nonexact map")
			}
		})
	}
}
func TestResponseRefusals(t *testing.T) {
	r := request(t, hello(t, "2.0.0", false))
	tests := []struct {
		name string
		edit func(map[string]any)
	}{
		{"id", func(m map[string]any) { m["request_id"] = otherID }},
		{"version", func(m map[string]any) { m["protocol_version"] = "3.0.0"; m["body"] = hello(t, "3.0.0", true)["body"] }},
		{"major-before-error", func(m map[string]any) {
			m["protocol_version"] = "6.0.0"
			m["ok"] = false
			m["error"] = map[string]any{"retryable": true, "code": "anything"}
		}},
		{"nonce-echo", func(m map[string]any) { m["body"].(map[string]any)["nonce_echo"] = "cXJzdHV2d3h5ejAxMjM0NQ" }},
		{"missing-echo", func(m map[string]any) { delete(m["body"].(map[string]any), "nonce_echo") }},
		{"response-nonce", func(m map[string]any) { m["body"].(map[string]any)["nonce"] = "bad" }},
		{"ok-null", func(m map[string]any) { m["ok"] = nil }},
		{"ok-string", func(m map[string]any) { m["ok"] = "true" }},
		{"success-error", func(m map[string]any) { m["error"] = nil }},
		{"failure-body", func(m map[string]any) { m["ok"] = false; m["error"] = nil }},
		{"body-null", func(m map[string]any) { m["body"] = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := hello(t, "2.0.0", true)
			tt.edit(m)
			_, e := rpcwire.DecodeResponse(wire(t, m), r)
			if e == nil {
				t.Fatal("admitted invalid response")
			}
			if tt.name == "major-before-error" && !errors.Is(e, rpcwire.ErrVersion) {
				t.Fatal("read foreign payload", e)
			}
		})
	}
	if _, e := rpcwire.DecodeResponse(wire(t, hello(t, "2.0.0", true)), rpcwire.Request{}); e == nil {
		t.Fatal("zero expectation")
	}
	if _, e := rpcwire.EncodeSuccess(r, json.RawMessage(`null`)); e == nil {
		t.Fatal("encoded null")
	}
}
func TestHistoricalFailureBindings(t *testing.T) {
	for i, v := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		t.Run(v, func(t *testing.T) {
			r := request(t, hello(t, v, false))
			bound := []axerror.Version{axerror.Version100, axerror.Version120, axerror.Version130, axerror.Version130}[i]
			f, e := axerror.New(axerror.Spec{Version: bound, Code: "incompatible_protocol", Message: "incompatible", Details: axerror.Details{}, IDs: axerror.NoIDs(), Cause: errors.New("secret cause")})
			if e != nil {
				t.Fatal(e)
			}
			line, e := rpcwire.EncodeFailure(r, f)
			if e != nil {
				t.Fatal(e)
			}
			if bytes.Contains(line, []byte("secret cause")) || bytes.Contains(line, []byte(`"body"`)) {
				t.Fatal("failure leak/shape")
			}
			resp, e := rpcwire.DecodeResponse(line, r)
			if e != nil || resp.OK() || resp.Failure().Version() != bound {
				t.Fatal(resp, e)
			}
			for _, wrong := range []axerror.Version{axerror.Version100, axerror.Version110, axerror.Version120, axerror.Version130} {
				if wrong == bound {
					continue
				}
				f, e = axerror.New(axerror.Spec{Version: wrong, Code: "incompatible_protocol", Message: "wrong", Details: axerror.Details{}})
				if e != nil {
					t.Fatal(e)
				}
				if _, e = rpcwire.EncodeFailure(r, f); e == nil {
					t.Fatal("wrong historical binding")
				}
			}
			if _, e = rpcwire.EncodeFailure(r, nil); e == nil {
				t.Fatal("nil failure")
			}
		})
	}
}
func TestLimitsAndUntrustedIsolation(t *testing.T) {
	q := hello(t, "2.0.0", false)
	b := q["body"].(map[string]any)
	b["max_line_bytes"] = 10 * 1024 * 1024
	b["max_object_bytes"] = 7 * 1024 * 1024
	vs := []string{}
	for i := 0; i < 16; i++ {
		vs = append(vs, fmt.Sprintf("1.%d.0", i))
	}
	slices.Sort(vs)
	b["contracts"].(map[string]any)["lease"] = vs
	r := request(t, q)
	p := hello(t, "2.0.0", true)
	p["body"].(map[string]any)["max_object_bytes"] = 6 * 1024 * 1024
	resp, e := rpcwire.DecodeResponse(wire(t, p), r)
	if e != nil {
		t.Fatal(e)
	}
	limits, e := rpcwire.OfferedLimits(r, resp)
	if e != nil || limits.LineBytes != 8388608 || limits.ObjectBytes != 6*1024*1024 {
		t.Fatal(limits, e)
	}
	q["request_id"] = otherID
	r2 := request(t, q)
	if _, e = rpcwire.OfferedLimits(r2, resp); e == nil {
		t.Fatal("cross-request limits")
	}
	// No authentication claim: arbitrary structurally valid claimed host is retained.
	b["host_id"] = otherID
	r = request(t, q)
	h, e := r.Hello()
	if e != nil || h.HostID != otherID {
		t.Fatal("untrusted host retention")
	}
	n := rpcwire.NewNonce()
	decoded, e := base64.RawURLEncoding.DecodeString(n)
	if e != nil || len(decoded) != 32 {
		t.Fatal("nonce generation")
	}
	if _, e = rpcwire.ContractProfile("6.0.0"); e == nil {
		t.Fatal("unknown profile")
	}
	if _, e = rpcwire.Namespaces("6.0.0"); e == nil {
		t.Fatal("unknown namespace profile")
	}
}
func TestInventoryNamespacesAndCardinality(t *testing.T) {
	for _, v := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		t.Run(v, func(t *testing.T) {
			names, e := rpcwire.Namespaces(v)
			if e != nil {
				t.Fatal(e)
			}
			want := map[string]int{"2.0.0": 6, "3.0.0": 7, "4.0.0": 8, "5.0.0": 8}[v]
			if len(names) != want {
				t.Fatal("wrong cardinality")
			}
			q := map[string]any{"protocol": rpcwire.Protocol, "protocol_version": v, "request_id": id, "operation": "inventory.roots", "body": map[string]any{"namespaces": names}}
			r := request(t, q)
			roots := []any{}
			for _, name := range names {
				roots = append(roots, map[string]any{"namespace": name, "count": 0, "root_id": "sha256:" + strings.Repeat("0", 64)})
			}
			if _, e = rpcwire.EncodeSuccess(r, wire(t, map[string]any{"roots": roots})); e != nil {
				t.Fatal(e)
			}
			for _, bad := range [][]string{{}, {"blob", "blob"}, {"event", "blob"}, {"credential"}, append(slices.Clone(names), "zzz")} {
				q["body"] = map[string]any{"namespaces": bad}
				if _, e = rpcwire.DecodeRequest(wire(t, q)); e == nil {
					t.Fatal("invalid namespaces", bad)
				}
			}
			for name, bad := range map[string]any{"missing": roots[:len(roots)-1], "extra": append(slices.Clone(roots), roots[0]), "null": nil} {
				t.Run(name, func(t *testing.T) {
					if _, e = rpcwire.EncodeSuccess(r, wire(t, map[string]any{"roots": bad})); e == nil {
						t.Fatal("invalid cardinality")
					}
				})
			}
			root := roots[0].(map[string]any)
			for _, field := range []string{"namespace", "count", "root_id"} {
				old := root[field]
				root[field] = nil
				if _, e = rpcwire.EncodeSuccess(r, wire(t, map[string]any{"roots": roots})); e == nil {
					t.Fatal("null root field", field)
				}
				root[field] = old
			}
			root["namespace"] = "event"
			if _, e = rpcwire.EncodeSuccess(r, wire(t, map[string]any{"roots": roots})); e == nil {
				t.Fatal("wrong namespace")
			}
			root["namespace"] = names[0]
			root["extra"] = true
			if _, e = rpcwire.EncodeSuccess(r, wire(t, map[string]any{"roots": roots})); e == nil {
				t.Fatal("extra root field")
			}
			delete(root, "extra")
			root["count"] = scalar.MaxUint53
			if _, e = rpcwire.EncodeSuccess(r, wire(t, map[string]any{"roots": roots})); e != nil {
				t.Fatal("uint53 max", e)
			}
			root["count"] = scalar.MaxUint53 + 1
			if _, e = rpcwire.EncodeSuccess(r, wire(t, map[string]any{"roots": roots})); e == nil {
				t.Fatal("uint53 overflow")
			}
			// An explicit one-namespace subset is valid in every major; it is not a
			// complete inventory and is never reported as one by this codec.
			q["body"] = map[string]any{"namespaces": []string{"blob"}}
			r = request(t, q)
			if _, e = rpcwire.EncodeSuccess(r, []byte(`{"roots":[{"namespace":"blob","count":0,"root_id":"sha256:`+strings.Repeat("0", 64)+`"}]}`)); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestOpaqueOperationBoundary(t *testing.T) {
	body := json.RawMessage(`{"extensions":{"vendor.example":{"future":"retained"}}}`)
	line, e := rpcwire.EncodeRequest("2.0.0", id, "future.operation", body)
	if e != nil {
		t.Fatal(e)
	}
	r, e := rpcwire.DecodeRequest(line)
	if e != nil || !bytes.Equal(r.Body(), body) {
		t.Fatal("opaque body altered", e)
	}
	if _, e = r.Hello(); e == nil {
		t.Fatal("opaque operation is not hello")
	}
	if _, e = rpcwire.EncodeSuccess(r, body); e != nil {
		t.Fatal(e)
	}
	if _, e = rpcwire.EncodeRequest("2.0.0", id, "future.operation", []byte("{")); e == nil {
		t.Fatal("malformed outbound")
	}
}

func TestBootstrapRejectionFraming(t *testing.T) {
	f, e := axerror.New(axerror.Spec{Version: axerror.Version100, Code: "incompatible_protocol", Message: "hello required", Details: axerror.Details{}})
	if e != nil {
		t.Fatal(e)
	}
	expectation := request(t, hello(t, "2.0.0", false))
	for _, kind := range []string{"non-hello", "missing-contract", "invalid-body"} {
		t.Run(kind, func(t *testing.T) {
			m := hello(t, "2.0.0", false)
			switch kind {
			case "non-hello":
				m["operation"] = "health.get"
				m["body"] = map[string]any{}
			case "missing-contract":
				delete(m["body"].(map[string]any)["contracts"].(map[string]any), "lease")
			case "invalid-body":
				m["body"] = nil
			}
			line, e := rpcwire.EncodeRejection(wire(t, m), f)
			if e != nil {
				t.Fatal(e)
			}
			result, e := rpcwire.DecodeResponse(line, expectation)
			if e != nil || result.OK() || result.Failure().Version() != axerror.Version100 {
				t.Fatal("bootstrap failure", e)
			}
		})
	}
	for _, kind := range []string{"major", "protocol", "id", "json", "oversize", "response"} {
		t.Run(kind, func(t *testing.T) {
			m := hello(t, "2.0.0", false)
			switch kind {
			case "major":
				m["protocol_version"] = "6.0.0"
			case "protocol":
				delete(m, "protocol")
			case "id":
				delete(m, "request_id")
			case "response":
				delete(m, "operation")
			}
			b := wire(t, m)
			if kind == "json" {
				b = []byte("{")
			}
			if kind == "oversize" {
				b = append(b, bytes.Repeat([]byte(" "), rpcwire.MaxLineBytes+1-len(b))...)
			}
			b, e = rpcwire.EncodeRejection(b, f)
			if e == nil || b != nil {
				t.Fatal("unframeable rejection emitted", e)
			}
		})
	}
}

func TestWriterAndAccessorBounds(t *testing.T) {
	r := request(t, hello(t, "2.0.0", false))
	if _, e := rpcwire.EncodeSuccess(r, json.RawMessage(`{"nonce":`)); e == nil {
		t.Fatal("invalid raw success")
	}
	if _, e := rpcwire.EncodeRequest("6.0.0", id, "health.get", []byte(`{}`)); e == nil {
		t.Fatal("invalid outbound version")
	}
	if _, e := rpcwire.EncodeSuccess(rpcwire.Request{}, []byte(`{}`)); e == nil {
		t.Fatal("zero request")
	}
	if _, e := (rpcwire.Response{}).Hello(); e == nil {
		t.Fatal("zero response")
	}
	if _, e := rpcwire.OfferedLimits(rpcwire.Request{}, rpcwire.Response{}); e == nil {
		t.Fatal("zero pair")
	}
	if _, e := rpcwire.OfferedLimits(r, rpcwire.Response{}); e == nil {
		t.Fatal("zero response pair")
	}
	huge := wire(t, map[string]any{"data": strings.Repeat("x", rpcwire.MaxLineBytes)})
	if _, e := rpcwire.EncodeRequest("2.0.0", id, "objects.get", huge); e == nil {
		t.Fatal("oversized outbound frame")
	}
}

func FuzzUntrustedEnvelopes(f *testing.F) {
	f.Add([]byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"2.0.0","request_id":"` + id + `","operation":"health.get","body":{}}`))
	f.Add([]byte(`{"protocol_version":"6.0.0","error":{"retryable":true}}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		r, e := rpcwire.DecodeRequest(b)
		if e != nil {
			return
		}
		round, e := rpcwire.EncodeRequest(r.Version(), r.ID(), r.Operation(), r.Body())
		if e != nil {
			t.Fatal(e)
		}
		if _, e = rpcwire.DecodeRequest(round); e != nil {
			t.Fatal(e)
		}
		_, _ = rpcwire.DecodeResponse(b, r)
	})
}
