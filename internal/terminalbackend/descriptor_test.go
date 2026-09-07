package terminalbackend_test

import (
	"strings"
	"testing"

	terminalbackend "github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// validDescriptorDoc is one fully valid §7.A descriptor document: every
// member in range, matching the validDescriptorBinding below field for
// field on the binding subset.
const validDescriptorDoc = `{
  "terminal_binding_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
  "terminal_instance_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "terminal_backend_id": "com.example.term",
  "implementation_version": "1.2.3",
  "protocol_version": "1.0.0",
  "backend_generation": "generation-1",
  "interactive": true,
  "columns": 80,
  "rows": 24
}`

func validDescriptorBinding() terminalbackend.InstanceBinding {
	return terminalbackend.InstanceBinding{
		BackendID:             "com.example.term",
		ImplementationVersion: "1.2.3",
		ProtocolVersion:       "1.0.0",
		Generation:            "generation-1",
		TerminalBindingID:     testDigest,
	}
}

// setDescriptorMember rewrites one top-level member literal of the valid
// document. The anchor must occur exactly once.
func setDescriptorMember(t *testing.T, member, literal string) []byte {
	t.Helper()
	anchors := map[string]string{
		"terminal_binding_id":    `"terminal_binding_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000"`,
		"terminal_instance_id":   `"terminal_instance_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`,
		"terminal_backend_id":    `"terminal_backend_id": "com.example.term"`,
		"implementation_version": `"implementation_version": "1.2.3"`,
		"protocol_version":       `"protocol_version": "1.0.0"`,
		"backend_generation":     `"backend_generation": "generation-1"`,
		"interactive":            `"interactive": true`,
		"columns":                `"columns": 80`,
		"rows":                   `"rows": 24`,
	}
	anchor, known := anchors[member]
	if !known {
		t.Fatalf("unknown descriptor member %q", member)
	}
	if strings.Count(validDescriptorDoc, anchor) != 1 {
		t.Fatalf("descriptor anchor for %q is not unique", member)
	}
	return []byte(strings.Replace(validDescriptorDoc, anchor, `"`+member+`": `+literal, 1))
}

// requireDescriptorRefusal asserts the full arm identity: the protocol
// code and the exact static detail. The code alone cannot discriminate
// the nine parse arms, and the detail alone cannot discriminate the
// match arms sharing a dimension, so both are asserted every time.
func requireDescriptorRefusal(t *testing.T, err error, detail string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want refusal with detail %q, got admission", detail)
	}
	if !terminalbackend.IsProtocolError(err) {
		t.Fatalf("error %v is not terminal_backend_protocol_error", err)
	}
	if got := refusalDetail(t, err); got != detail {
		t.Fatalf("refusal detail = %q, want %q", got, detail)
	}
}

// TestAdmitProviderDescriptorAdmitsMatchingBinding is the admit direction
// of the §7.A gate: a closed, well-formed descriptor matching the
// AX-validated host-local binding is admitted, and the parsed value
// carries every member.
func TestAdmitProviderDescriptorAdmitsMatchingBinding(t *testing.T) {
	t.Parallel()

	descriptor, err := terminalbackend.AdmitProviderDescriptor([]byte(validDescriptorDoc), validDescriptorBinding())
	if err != nil {
		t.Fatalf("AdmitProviderDescriptor(match) error = %v, want admission", err)
	}
	if descriptor.TerminalBindingID != testDigest ||
		descriptor.TerminalInstanceID != "0198f4c8-3e70-7a11-8a2b-1234567890ab" ||
		descriptor.TerminalBackendID != "com.example.term" ||
		descriptor.ImplementationVersion != "1.2.3" ||
		descriptor.ProtocolVersion != "1.0.0" ||
		descriptor.BackendGeneration != "generation-1" ||
		!descriptor.Interactive ||
		descriptor.Columns != 80 ||
		descriptor.Rows != 24 {
		t.Fatalf("AdmitProviderDescriptor(match) = %+v, want the valid descriptor", descriptor)
	}
}

// TestParseProviderDescriptorMemberSet proves the closed set in three
// shapes: an extra member, a missing member, and a duplicate member. The
// first two share the descriptor member set arm (one rule: the closed
// set); the duplicate is refused earlier by the strict decoder with the
// document duplicate member arm, so a decoder bypass cannot smuggle a
// repeated member past the set check.
func TestParseProviderDescriptorMemberSet(t *testing.T) {
	t.Parallel()

	extra := strings.Replace(validDescriptorDoc, `"rows": 24`, `"rows": 24, "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, 1)
	_, err := terminalbackend.ParseProviderDescriptor([]byte(extra))
	requireDescriptorRefusal(t, err, "descriptor member set")

	missing := strings.Replace(validDescriptorDoc, ",\n  \"rows\": 24", "", 1)
	_, err = terminalbackend.ParseProviderDescriptor([]byte(missing))
	requireDescriptorRefusal(t, err, "descriptor member set")

	duplicate := strings.Replace(validDescriptorDoc, `"rows": 24`, `"rows": 24, "rows": 25`, 1)
	_, err = terminalbackend.ParseProviderDescriptor([]byte(duplicate))
	if err == nil || !terminalbackend.IsMismatch(err) {
		t.Fatalf("duplicate member error = %v, want a document mismatch refusal", err)
	}
	if got := refusalDetail(t, err); got != "document duplicate member" {
		t.Fatalf("duplicate member detail = %q, want document duplicate member", got)
	}
}

// TestParseProviderDescriptorMemberType enumerates the JSON type
// discipline across all six string members: a number where a string
// belongs is refused with the single descriptor member type arm, no
// matter which member carries it.
func TestParseProviderDescriptorMemberType(t *testing.T) {
	t.Parallel()

	for _, member := range []string{
		"terminal_binding_id", "terminal_instance_id", "terminal_backend_id",
		"implementation_version", "protocol_version", "backend_generation",
	} {
		_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, member, `80`))
		requireDescriptorRefusal(t, err, "descriptor member type")
	}
	_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "interactive", `"true"`))
	requireDescriptorRefusal(t, err, "descriptor interactive")
}

// TestParseProviderDescriptorValueRefusals drives one shape per value
// arm, asserting the full arm identity each time.
func TestParseProviderDescriptorValueRefusals(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		member  string
		literal string
		detail  string
	}{
		{"binding digest malformed", "terminal_binding_id", `"not-a-digest"`, "descriptor digest"},
		{"binding digest empty", "terminal_binding_id", `""`, "descriptor digest"},
		{"binding id null", "terminal_binding_id", `null`, "descriptor member type"},
		{"instance pid", "terminal_instance_id", `"1234"`, "descriptor instance"},
		{"instance path", "terminal_instance_id", `"/tmp/instance"`, "descriptor instance"},
		{"instance uuid v4", "terminal_instance_id", `"550e8400-e29b-41d4-a716-446655440000"`, "descriptor instance"},
		{"implementation version not semver", "implementation_version", `"1.2"`, "descriptor implementation version"},
		{"implementation version empty", "implementation_version", `""`, "descriptor implementation version"},
		{"protocol version not semver", "protocol_version", `"one"`, "descriptor protocol version"},
		// The narrowing shape for the major-1 rule: a well-formed semver
		// outside major 1 must still refuse. A mutant admitting any
		// semver here passes every other row of this table.
		{"protocol version major 2", "protocol_version", `"2.0.0"`, "descriptor protocol version"},
		{"interactive number", "interactive", `1`, "descriptor interactive"},
		{"interactive null", "interactive", `null`, "descriptor interactive"},
		{"columns zero", "columns", `0`, "descriptor geometry bound"},
		{"columns over bound", "columns", `1001`, "descriptor geometry bound"},
		{"columns fraction", "columns", `80.0`, "descriptor geometry digits"},
		{"columns exponent", "columns", `1e2`, "descriptor geometry digits"},
		// The uppercase-exponent narrowing: `E` (0x45) is the one
		// reachable member of the 32-value window between the true
		// edge `9` (0x39) and the witnessed `e` (0x65). A mutant
		// upper bound X with `E` <= X < `e` admits `1E2` as 312
		// while every other row of this table still refuses.
		{"columns exponent uppercase", "columns", `1E2`, "descriptor geometry digits"},
		{"columns negative", "columns", `-80`, "descriptor geometry digits"},
		{"columns string", "columns", `"80"`, "descriptor geometry type"},
		{"columns null", "columns", `null`, "descriptor geometry type"},
		{"rows zero", "rows", `0`, "descriptor geometry bound"},
		{"rows over bound", "rows", `65535`, "descriptor geometry bound"},
		{"rows fraction", "rows", `24.5`, "descriptor geometry digits"},
		// Same uppercase-exponent narrowing on the second geometry
		// member: the gate is one shared site, and the rows row
		// proves the fix is at the gate, not at one call.
		{"rows exponent uppercase", "rows", `1E2`, "descriptor geometry digits"},
		{"rows boolean", "rows", `true`, "descriptor geometry type"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, testCase.member, testCase.literal))
			requireDescriptorRefusal(t, err, testCase.detail)
		})
	}
}

// TestParseProviderDescriptorGeometryOverflowVectors proves the 1..1000
// bound is decided on the digit string, never on a wrapped accumulator:
// every literal below names a value outside 1..1000 whose low 64 bits
// land inside it (2^64+1 wraps to 1, 2^64+1000 wraps to 1000,
// 18446744073709552116 wraps to 500, and the 20-digit row wraps twice),
// so an accumulating gate admits them with a fabricated in-range number
// downstream cannot distinguish from a real one. All must refuse with
// the bound arm; rows spots the shared site on the second member.
//
// The 37-digit witness pins the pre-multiply guard `value > 100` at its
// own arithmetic edge: narrowing it to `value > 922337203685477580`
// (floor(MaxInt/10)) admits this literal as 1 while every shorter
// shipped vector still refuses, so only this row reddens that mutant.
// It runs on both members because the guard is one shared site.
func TestParseProviderDescriptorGeometryOverflowVectors(t *testing.T) {
	t.Parallel()

	columns := []string{
		"1000",
		"1001",
		"9223372036854775808",
		"18446744073709551616",
		"18446744073709551617",
		"18446744073709552116",
		"18446744073709552616",
		"55340232221128655348",
		"9223372036854775808000000000000000001",
		"99999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999",
	}
	for _, literal := range columns {
		if literal == "1000" {
			if _, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "columns", literal)); err != nil {
				t.Fatalf("ParseProviderDescriptor(columns=%s) error = %v, want admission", literal, err)
			}
			continue
		}
		_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "columns", literal))
		requireDescriptorRefusal(t, err, "descriptor geometry bound")
	}
	for _, literal := range []string{"18446744073709551617", "55340232221128655348", "9223372036854775808000000000000000001"} {
		_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "rows", literal))
		requireDescriptorRefusal(t, err, "descriptor geometry bound")
	}
}

// TestParseProviderDescriptorProtocolVersionOverflowVectors proves the
// major-1 rule is decided on the digit string, never on a wrapped
// accumulator: 18446744073709551617 wraps to 1 in 64 bits, so an
// accumulating major gate admits this foreign major as the native one.
// All three literals name non-1 majors and must refuse with the protocol
// version arm.
func TestParseProviderDescriptorProtocolVersionOverflowVectors(t *testing.T) {
	t.Parallel()

	for _, literal := range []string{
		`"18446744073709551617.0.0"`,
		`"18446744073709551618.0.0"`,
		`"99999999999999999999999999999999999999999.0.0"`,
	} {
		_, err := terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "protocol_version", literal))
		requireDescriptorRefusal(t, err, "descriptor protocol version")
	}
}

// TestParseProviderDescriptorGeometryBounds proves the 1..1000 bound
// itself: 1 and 1000 admit on both members, so the bound refusals above
// cannot be attributed to the type or digit arms.
func TestParseProviderDescriptorGeometryBounds(t *testing.T) {
	t.Parallel()

	for _, literal := range []string{`1`, `1000`} {
		for _, member := range []string{"columns", "rows"} {
			doc := setDescriptorMember(t, member, literal)
			if _, err := terminalbackend.ParseProviderDescriptor(doc); err != nil {
				t.Errorf("ParseProviderDescriptor(%s=%s) error = %v, want admission", member, literal, err)
			}
		}
	}
}

// TestParseProviderDescriptorRefusesWithoutEchoingLocalData proves the
// package's never-echoes-local-data posture at the new arms: refused
// values reach the error only through the static clause, never
// interpolated. The first document carries a well-formed but unmatched
// digest past a broken instance, so the digest is traversed and still
// not echoed; the second breaks the digest itself with a distinctive
// token.
func TestParseProviderDescriptorRefusesWithoutEchoingLocalData(t *testing.T) {
	t.Parallel()

	foreignDigest := "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	doc := setDescriptorMember(t, "terminal_binding_id", `"`+foreignDigest+`"`)
	doc = []byte(strings.Replace(string(doc),
		`"terminal_instance_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`,
		`"terminal_instance_id": "not-a-uuid"`, 1))
	_, err := terminalbackend.ParseProviderDescriptor(doc)
	requireDescriptorRefusal(t, err, "descriptor instance")
	if strings.Contains(err.Error(), foreignDigest) {
		t.Fatalf("refusal echoes the digest under test: %v", err)
	}

	brokenDigest := "digest-ZZZ-not-a-digest"
	_, err = terminalbackend.ParseProviderDescriptor(setDescriptorMember(t, "terminal_binding_id", `"`+brokenDigest+`"`))
	requireDescriptorRefusal(t, err, "descriptor digest")
	if strings.Contains(err.Error(), "ZZZ") {
		t.Fatalf("refusal echoes the digest under test: %v", err)
	}
}

// TestAdmitProviderDescriptorRefusesMismatch is the refuse direction of
// the §7.A gate through the complete entry: a well-formed descriptor
// that does not match the binding is refused with the binding arm
// CheckProviderDescriptor reports, not with a parse arm.
func TestAdmitProviderDescriptorRefusesMismatch(t *testing.T) {
	t.Parallel()

	binding := validDescriptorBinding()

	drift := binding
	drift.ImplementationVersion = "1.2.4"
	if _, err := terminalbackend.AdmitProviderDescriptor([]byte(validDescriptorDoc), drift); !terminalbackend.IsDrift(err) {
		t.Fatalf("AdmitProviderDescriptor(version drift) error = %v, want terminal_backend_implementation_drift", err)
	}

	foreign := binding
	foreign.BackendID = "com.example.other"
	if _, err := terminalbackend.AdmitProviderDescriptor([]byte(validDescriptorDoc), foreign); !terminalbackend.IsNotFound(err) {
		t.Fatalf("AdmitProviderDescriptor(ID mismatch) error = %v, want terminal_backend_not_found", err)
	}

	stale := binding
	stale.Generation = "generation-2"
	if _, err := terminalbackend.AdmitProviderDescriptor([]byte(validDescriptorDoc), stale); !terminalbackend.IsStaleGeneration(err) {
		t.Fatalf("AdmitProviderDescriptor(stale generation) error = %v, want terminal_backend_stale_generation", err)
	}

	// Generation match is byte equality, not case folding: a
	// case-variant generation is stale, and a mutant folding case
	// admits it.
	folded := binding
	folded.Generation = "GENERATION-1"
	if _, err := terminalbackend.AdmitProviderDescriptor([]byte(validDescriptorDoc), folded); !terminalbackend.IsStaleGeneration(err) {
		t.Fatalf("AdmitProviderDescriptor(case-variant generation) error = %v, want terminal_backend_stale_generation", err)
	}

	foreignDigest := binding
	foreignDigest.TerminalBindingID = testDigestOther
	if _, err := terminalbackend.AdmitProviderDescriptor([]byte(validDescriptorDoc), foreignDigest); !terminalbackend.IsNotFound(err) {
		t.Fatalf("AdmitProviderDescriptor(digest mismatch) error = %v, want terminal_backend_not_found", err)
	}
}
