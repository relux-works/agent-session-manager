package termbind

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestParseTerminalBindingMemberArms drives every member arm of the
// Section 4.B Terminal Instance Binding table negatively through the
// production entry: each row mutates exactly one member of the closed
// fixture (or the frame) and pins the refusal code plus the refusing
// arm's detail, so a reroute past the arm to a same-code sibling still
// fails. The frame/member-set arms (missing, extra, top-level duplicate,
// wrong schema literal) stay in TestParseTerminalBindingClosedShape; the
// generation/native bounds stay there too. Positive neighbors prove each
// arm admits what the table admits.
func TestParseTerminalBindingMemberArms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(object map[string]any)
		code   string
		detail string
	}{
		{"schema non-string", func(object map[string]any) { object["schema"] = float64(1) }, "terminal_backend_protocol_error", "binding schema"},
		{"schema_version non-string", func(object map[string]any) { object["schema_version"] = float64(1) }, "terminal_backend_protocol_error", "binding schema_version"},
		{"schema_version wrong literal", func(object map[string]any) { object["schema_version"] = "2.0.0" }, "terminal_backend_protocol_error", "binding schema version"},
		{"binding_id non-string", func(object map[string]any) { object["binding_id"] = float64(1) }, "terminal_backend_protocol_error", "binding binding_id"},
		{"binding_id non-digest", func(object map[string]any) { object["binding_id"] = "not-a-digest" }, "terminal_backend_protocol_error", "binding digest"},
		{"session_id non-string", func(object map[string]any) { object["session_id"] = float64(1) }, "terminal_backend_protocol_error", "binding session_id"},
		{"session_id non-uuidv7", func(object map[string]any) { object["session_id"] = "12345" }, "terminal_backend_protocol_error", "binding session_id"},
		{"host_id non-string", func(object map[string]any) { object["host_id"] = float64(1) }, "terminal_backend_protocol_error", "binding host_id"},
		{"host_id non-uuidv7", func(object map[string]any) { object["host_id"] = "12345" }, "terminal_backend_protocol_error", "binding host_id"},
		{"host_incarnation_id non-string", func(object map[string]any) { object["host_incarnation_id"] = float64(1) }, "terminal_backend_protocol_error", "binding host_incarnation_id"},
		{"host_incarnation_id non-uuidv7", func(object map[string]any) { object["host_incarnation_id"] = "12345" }, "terminal_backend_protocol_error", "binding host_incarnation_id"},
		{"terminal_instance_id non-string", func(object map[string]any) { object["terminal_instance_id"] = float64(12345) }, "terminal_backend_protocol_error", "binding terminal_instance_id"},
		{"terminal_backend_id non-string", func(object map[string]any) { object["terminal_backend_id"] = float64(1) }, "terminal_backend_protocol_error", "binding terminal_backend_id"},
		{"terminal_backend_id invalid", func(object map[string]any) { object["terminal_backend_id"] = "not a backend id!!" }, "terminal_backend_not_found", "terminal_backend_id grammar"},
		{"implementation_version non-string", func(object map[string]any) { object["implementation_version"] = float64(1) }, "terminal_backend_protocol_error", "binding implementation_version"},
		{"implementation_version non-semver", func(object map[string]any) { object["implementation_version"] = "v2" }, "terminal_backend_protocol_error", "binding implementation version"},
		{"protocol_version non-string", func(object map[string]any) { object["protocol_version"] = float64(1) }, "terminal_backend_protocol_error", "binding protocol_version"},
		{"protocol_version non-semver", func(object map[string]any) { object["protocol_version"] = "v1" }, "terminal_backend_protocol_error", "binding protocol version"},
		{"protocol_version major 0", func(object map[string]any) { object["protocol_version"] = "0.9.0" }, "terminal_backend_protocol_error", "binding protocol version"},
		{"protocol_version major 2", func(object map[string]any) { object["protocol_version"] = "2.0.0" }, "terminal_backend_protocol_error", "binding protocol version"},
		{"protocol_version major 3", func(object map[string]any) { object["protocol_version"] = "3.1.4" }, "terminal_backend_protocol_error", "binding protocol version"},
		{"protocol_version huge major", func(object map[string]any) { object["protocol_version"] = "18446744073709551617.0.0" }, "terminal_backend_protocol_error", "binding protocol version"},
		{"backend_generation non-string", func(object map[string]any) { object["backend_generation"] = float64(1) }, "terminal_backend_protocol_error", "binding backend_generation"},
		{"native_reference non-string", func(object map[string]any) { object["native_reference"] = float64(1) }, "terminal_backend_protocol_error", "binding native_reference"},
		{"created_at non-string", func(object map[string]any) { object["created_at"] = float64(1) }, "terminal_backend_protocol_error", "binding created_at"},
		{"created_at invalid", func(object map[string]any) { object["created_at"] = "not-a-timestamp" }, "terminal_backend_protocol_error", "binding timestamp"},
		{"supersedes non-string", func(object map[string]any) { object["supersedes_binding_id"] = float64(1) }, "terminal_backend_protocol_error", "binding supersedes_binding_id"},
		{"supersedes non-digest", func(object map[string]any) { object["supersedes_binding_id"] = "not-a-digest" }, "terminal_backend_protocol_error", "binding supersedes"},
		{"extensions string", func(object map[string]any) { object["extensions"] = "x" }, "terminal_backend_protocol_error", "binding extensions"},
		{"extensions number", func(object map[string]any) { object["extensions"] = float64(1) }, "terminal_backend_protocol_error", "binding extensions"},
		{"extensions array", func(object map[string]any) { object["extensions"] = []any{"x"} }, "terminal_backend_protocol_error", "binding extensions"},
		{"extensions null", func(object map[string]any) { object["extensions"] = nil }, "terminal_backend_protocol_error", "binding extensions"},
		{"extensions non-empty", func(object map[string]any) { object["extensions"] = map[string]any{"com.example.note": "y"} }, "terminal_backend_protocol_error", "binding extensions"},
	}
	for _, testCase := range cases {
		t.Run(strings.ReplaceAll(testCase.name, " ", "_"), func(t *testing.T) {
			t.Parallel()
			_, err := ParseTerminalBinding(bindingDoc(t, testCase.mutate))
			requireCode(t, err, testCase.code)
			requireDetail(t, err, testCase.detail)
		})
	}
	t.Run("frame non-object", func(t *testing.T) {
		t.Parallel()
		_, err := ParseTerminalBinding([]byte(`[]`))
		requireCode(t, err, "terminal_backend_protocol_error")
		requireDetail(t, err, "binding frame")
	})
	t.Run("frame trailing data", func(t *testing.T) {
		t.Parallel()
		_, err := ParseTerminalBinding(append(bindingDoc(t, nil), ' ', '{', '}'))
		requireCode(t, err, "terminal_backend_protocol_error")
		requireDetail(t, err, "binding frame")
	})
	t.Run("supersedes digest admits", func(t *testing.T) {
		t.Parallel()
		parsed, err := ParseTerminalBinding(bindingDoc(t, func(object map[string]any) {
			object["supersedes_binding_id"] = seedDigest(0x5D)
		}))
		if err != nil {
			t.Fatalf("ParseTerminalBinding(supersedes digest) error = %v", err)
		}
		if !parsed.HasSupersedes || parsed.SupersedesBindingID != seedDigest(0x5D) {
			t.Fatalf("parsed supersedes = %+v", parsed)
		}
	})
	t.Run("protocol major 1 admits", func(t *testing.T) {
		t.Parallel()
		for _, version := range []string{"1.0.0", "1.2.3", "1.0.0-alpha", "1.0.0+build.1"} {
			parsed, err := ParseTerminalBinding(bindingDoc(t, func(object map[string]any) {
				object["protocol_version"] = version
			}))
			if err != nil {
				t.Fatalf("ParseTerminalBinding(protocol %q) error = %v", version, err)
			}
			if parsed.ProtocolVersion != version {
				t.Fatalf("ProtocolVersion = %q, want %q", parsed.ProtocolVersion, version)
			}
		}
	})
}

// TestParseTerminalBindingRawExtensions drives hostile extension literals
// that a map-built fixture cannot spell — JSON numbers, a nested
// duplicate member, and a lone surrogate — through the production entry.
// Each refuses; the numbers and the nested duplicate additionally refuse
// the hardened identity decode directly (TestBindingIdentityRefusesHostileBytes),
// so the JCS transform is unreachable however the members were checked.
func TestParseTerminalBindingRawExtensions(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name    string
		literal string
	}{
		{"float", `{"com.example.n":1.0}`},
		{"beyond 2^53", `{"com.example.n":9007199254740993}`},
		{"exponent", `{"com.example.n":1e2}`},
		{"negative zero", `{"com.example.n":-0}`},
		{"nested duplicate", `{"com.example.o":{"a":"1","a":"2"}}`},
	} {
		t.Run(strings.ReplaceAll(testCase.name, " ", "_"), func(t *testing.T) {
			t.Parallel()
			raw := strings.Replace(string(bindingDoc(t, nil)), `"extensions":{}`, `"extensions":`+testCase.literal, 1)
			_, err := ParseTerminalBinding([]byte(raw))
			requireCode(t, err, "terminal_backend_protocol_error")
			requireDetail(t, err, "binding extensions")
		})
	}
	t.Run("lone surrogate refuses at the frame", func(t *testing.T) {
		t.Parallel()
		raw := strings.Replace(string(bindingDoc(t, nil)), `"extensions":{}`, `"extensions":{"com.example.s":"`+`\ud800`+`"}`, 1)
		_, err := ParseTerminalBinding([]byte(raw))
		requireCode(t, err, "terminal_backend_protocol_error")
		requireDetail(t, err, "binding frame")
	})
}

// TestBindingIdentityRefusesHostileBytes pins the pre-transform walk of
// bindingIdentity directly: a JSON number at any depth, a nested
// duplicate member, trailing data, a non-object top level, excessive
// nesting, and an oversize document each refuse before any JCS bytes
// exist, and the clean document still digests to its carried ID.
func TestBindingIdentityRefusesHostileBytes(t *testing.T) {
	t.Parallel()
	clean := bindingDoc(t, nil)
	recomputed, err := bindingIdentity(clean, "binding_id")
	if err != nil {
		t.Fatalf("bindingIdentity(clean) error = %v", err)
	}
	parsed, err := ParseTerminalBinding(clean)
	if err != nil {
		t.Fatalf("ParseTerminalBinding(clean) error = %v", err)
	}
	if recomputed != parsed.BindingID {
		t.Fatalf("bindingIdentity(clean) = %q, want carried %q", recomputed, parsed.BindingID)
	}
	hostile := []struct {
		name string
		raw  string
	}{
		{"float", `{"binding_id":"x","extensions":{"com.example.n":1.0}}`},
		{"beyond 2^53", `{"binding_id":"x","extensions":{"com.example.n":9007199254740993}}`},
		{"exponent", `{"binding_id":"x","extensions":{"com.example.n":1e2}}`},
		{"negative zero", `{"binding_id":"x","extensions":{"com.example.n":-0}}`},
		{"nested number", `{"binding_id":"x","extensions":{"com.example.o":{"n":[1]}}}`},
		{"top number", `{"binding_id":"x","extensions":{},"n":2}`},
		{"nested duplicate", `{"binding_id":"x","extensions":{"com.example.o":{"a":"1","a":"2"}}}`},
		{"top duplicate", `{"binding_id":"x","extensions":{},"extensions":{}}`},
		{"trailing data", `{"binding_id":"x","extensions":{}} {}`},
		{"non-object", `[]`},
		{"deep nesting", `{"binding_id":"x","extensions":` + strings.Repeat(`{"a":`, 40) + `1` + strings.Repeat(`}`, 40) + `}`},
		{"oversize", strings.Repeat("x", maxBindingIdentityBytes+1)},
	}
	for _, testCase := range hostile {
		t.Run(strings.ReplaceAll(testCase.name, " ", "_"), func(t *testing.T) {
			t.Parallel()
			_, err := bindingIdentity([]byte(testCase.raw), "binding_id")
			requireCode(t, err, "terminal_backend_protocol_error")
			requireDetail(t, err, "binding identity")
		})
	}
}

// TestBindingIdentityVerdictAgreesWithLandedAdmission extends the
// digest-equality agreement to verdict agreement on hostile members: a
// manifest-shaped object with a numeric extension value and one with a
// nested duplicate member refuse both the landed ParseManifest and this
// package's bindingIdentity, so the mirror cannot launder what the
// landed rule refuses.
func TestBindingIdentityVerdictAgreesWithLandedAdmission(t *testing.T) {
	t.Parallel()
	world := buildTmuxUniverse(t)
	base := string(mustJSON(t, world.manifest))
	for _, testCase := range []struct {
		name    string
		escaped string
	}{
		{"numeric extension", `"extensions":{"com.example.n":1.0}`},
		{"nested duplicate", `"extensions":{"com.example.o":{"a":"1","a":"2"}}`},
	} {
		t.Run(strings.ReplaceAll(testCase.name, " ", "_"), func(t *testing.T) {
			t.Parallel()
			raw := []byte(strings.Replace(base, `"extensions":{}`, testCase.escaped, 1))
			if _, err := terminalbackend.ParseManifest(raw); err == nil {
				t.Fatalf("ParseManifest(%s) admitted, want refusal", testCase.name)
			}
			if _, err := bindingIdentity(raw, "manifest_id"); err == nil {
				t.Fatalf("bindingIdentity(%s) admitted, want refusal", testCase.name)
			}
		})
	}
}

// TestBindingProtocolMajorAgreesWithLandedTuple pins that the major-1
// selection agrees verdict-for-verdict with the landed
// terminalbackend.CheckVersionTuple over a major corpus: grammar faults,
// foreign majors (including a huge major that must not alias 1), and
// major-1 spellings with prerelease and build metadata.
func TestBindingProtocolMajorAgreesWithLandedTuple(t *testing.T) {
	t.Parallel()
	corpus := []string{
		"1.0.0", "1.1.0", "1.2.3", "1.0.0-alpha", "1.0.0+build.1",
		"0.9.0", "2.0.0", "3.1.4", "10.0.0", "18446744073709551617.0.0",
		"1", "v1.0.0", "1.0", "",
	}
	for _, version := range corpus {
		name := version
		if name == "" {
			name = "empty"
		}
		t.Run(strings.ReplaceAll(name, ".", "_"), func(t *testing.T) {
			t.Parallel()
			landed := terminalbackend.CheckVersionTuple(terminalbackend.BuiltinTmux, fixtureImpl, version, []string{version}) == nil
			ours := environ.CheckSemver(version) && isProtocolMajorOne(version)
			if landed != ours {
				t.Fatalf("version %q: landed admits=%v, binding admits=%v", version, landed, ours)
			}
		})
	}
}
