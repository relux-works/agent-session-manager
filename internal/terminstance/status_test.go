package terminstance

import (
	"strings"
	"testing"
)

func statusDoc(instance, generation string) string {
	return `{"session_id":"` + fixtureSessionA + `",` +
		`"terminal_instance_id":` + instance + `,` +
		`"terminal_backend_id":"` + fixtureBackend + `",` +
		`"implementation_version":"1.2.3",` +
		`"protocol_version":"1.0.0",` +
		`"backend_generation":` + generation + `,` +
		`"include_provider_observation":false,` +
		`"deadline_at":"` + fixtureDeadline + `"}`
}

// TestParseStatusBodyAdmitsBothScopes drives the exact-instance and the
// Session-scoped lookups through the production entry.
func TestParseStatusBodyAdmitsBothScopes(t *testing.T) {
	exact, err := ParseStatusBody([]byte(statusDoc(`"`+fixtureInstance+`"`, `"generation-one"`)))
	if err != nil {
		t.Fatalf("ParseStatusBody(exact) error = %v", err)
	}
	if !exact.HasTerminalInstanceID || !exact.HasBackendGeneration {
		t.Error("exact lookup reports a null member, want both non-null")
	}
	requireLiteral(t, "instance", exact.TerminalInstanceID, "0198f4c8-8e50-7f66-8f70-bbbbbbbbbbb1")
	requireLiteral(t, "generation", exact.BackendGeneration, "generation-one")

	scoped, err := ParseStatusBody([]byte(statusDoc(`null`, `null`)))
	if err != nil {
		t.Fatalf("ParseStatusBody(session-scoped) error = %v", err)
	}
	if scoped.HasTerminalInstanceID || scoped.HasBackendGeneration {
		t.Error("session-scoped lookup reports a non-null member, want both null")
	}
}

// TestParseStatusBodyNullPairRule proves instance and generation are
// either both null or both non-null: exactly one null refuses.
func TestParseStatusBodyNullPairRule(t *testing.T) {
	_, err := ParseStatusBody([]byte(statusDoc(`"`+fixtureInstance+`"`, `null`)))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status identity scope")

	_, err = ParseStatusBody([]byte(statusDoc(`null`, `"generation-one"`)))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status identity scope")
}

// TestParseStatusBodyMemberRefusals drives every status member arm,
// including the generation bound at 0 and 257 with the landed stale
// class.
func TestParseStatusBodyMemberRefusals(t *testing.T) {
	base := statusDoc(`"`+fixtureInstance+`"`, `"generation-one"`)
	_, err := ParseStatusBody([]byte(strings.Replace(base, fixtureSessionA, "nope", 1)))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status body session")

	_, err = ParseStatusBody([]byte(statusDoc(`"nope"`, `"generation-one"`)))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status body instance")

	_, err = ParseStatusBody([]byte(statusDoc(`""`, `"generation-one"`)))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status body instance")

	_, err = ParseStatusBody([]byte(statusDoc(`"`+fixtureInstance+`"`, `""`)))
	requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation bound")

	_, err = ParseStatusBody([]byte(statusDoc(`"`+fixtureInstance+`"`, `"`+strings.Repeat("g", 257)+`"`)))
	requireRefusal(t, err, "terminal_backend_stale_generation", "backend_generation bound")

	if _, err := ParseStatusBody([]byte(statusDoc(`"`+fixtureInstance+`"`, `"`+strings.Repeat("g", 256)+`"`))); err != nil {
		t.Errorf("ParseStatusBody(generation 256) error = %v, want admission", err)
	}

	_, err = ParseStatusBody([]byte(strings.Replace(base, `"1.2.3"`, `"1.2"`, 1)))
	requireRefusal(t, err, "terminal_backend_implementation_drift", "implementation_version semver")

	_, err = ParseStatusBody([]byte(strings.Replace(base, `"include_provider_observation":false`, `"include_provider_observation":"yes"`, 1)))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status body observation")

	extra := strings.Replace(base, `"deadline_at"`, `"trace_id":"x","deadline_at"`, 1)
	_, err = ParseStatusBody([]byte(extra))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status body members")

	_, err = ParseStatusBody([]byte(`[1]`))
	requireRefusal(t, err, "terminal_backend_protocol_error", "status body frame")
}
