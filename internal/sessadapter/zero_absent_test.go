package sessadapter

import (
	"fmt"
	"testing"
)

// This file closes the zero-value / absent-fact class (review rev3
// B4): every envelope echo comparison must refuse the empty string,
// and every map-presence default must refuse an absent name. The
// subjects are derived from production — the envelope member lists
// from the Required slices, the target-write names from
// DoctorRequiredCapabilities, the capability names from
// capabilityOrder — so a new echo gate or required capability with
// no row here fails the derivation guard rather than passing
// silently.

// TestEnvelopeEchoEmptyStringRefuses drives "" through every echo
// comparison at the production entry: the two request identity
// checks through DecodeRequestFrame, the four success echoes and
// the four failure echoes through CheckSuccessEnvelope and
// CheckFailureEnvelope. A mutant narrowing any one gate to
// `(x != want && x != "")` admits its row and reddens here.
func TestEnvelopeEchoEmptyStringRefuses(t *testing.T) {
	if len(requestRequired) < 2 || requestRequired[0] != "protocol" || requestRequired[1] != "protocol_version" {
		t.Fatalf("request echo derivation changed: head = %v, want [protocol protocol_version]", requestRequired)
	}
	if len(successRequired) < 4 || successRequired[0] != "protocol" || successRequired[1] != "protocol_version" || successRequired[2] != "request_id" || successRequired[3] != "operation" {
		t.Fatalf("success echo derivation changed: head = %v", successRequired)
	}
	if len(failureRequired) < 4 || failureRequired[0] != "protocol" || failureRequired[1] != "protocol_version" || failureRequired[2] != "request_id" || failureRequired[3] != "operation" {
		t.Fatalf("failure echo derivation changed: head = %v", failureRequired)
	}
	want := wantRequest()
	echoWant := map[string]string{
		"protocol":         ProtocolID,
		"protocol_version": ProtocolVersion,
		"request_id":       want.RequestID,
		"operation":        string(want.Operation),
	}
	requestText := map[string]string{
		"protocol":         "not the session adapter",
		"protocol_version": "not 1.0.0",
	}
	probeBody := `{"expected_provider_id":"test-provider","expected_candidate_kind":"builtin","extensions":{}}`
	successBody := func() []byte {
		return []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":true,"body":{}}`,
			ProtocolID, fixtureRequestID))
	}
	failureBody := func() []byte {
		return []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"1.1.0","code":"capability_unavailable","message":"the adapter build cannot enumerate this source","exit_code":6,"retryable":false,"details":{}}}`,
			ProtocolID, fixtureRequestID))
	}
	rows := 0
	for _, member := range requestRequired[:2] {
		member := member
		t.Run("request/"+member, func(t *testing.T) {
			frame := mutateMember(t, fixtureRequestFrame(t, OpProbe, probeBody), member, `""`)
			_, err := DecodeRequestFrame(frame)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", member, requestText[member])
		})
		rows++
	}
	for _, member := range successRequired[:4] {
		member := member
		t.Run("success/"+member, func(t *testing.T) {
			frame := mutateMember(t, successBody(), member, `""`)
			_, err := CheckSuccessEnvelope(frame, want)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "does not echo")
		})
		rows++
	}
	for _, member := range failureRequired[:4] {
		member := member
		t.Run("failure/"+member, func(t *testing.T) {
			frame := mutateMember(t, failureBody(), member, `""`)
			_, err := CheckFailureEnvelope(frame, want)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "does not echo")
		})
		rows++
	}
	if want := identityCensusDomainCount("envelope-echo"); rows != want {
		t.Fatalf("echo rows = %d, want %d (2 request + 4 success + 4 failure); the derivation is short", rows, want)
	}
	for member, value := range echoWant {
		if value == "" {
			t.Fatalf("echo want for %q is empty; the probe is broken, not the gate", member)
		}
	}
}

// TestCheckTargetWriteGatesAbsentAdapterRefuses is the adapter-side
// twin of the provider-side "missing surface is refused, never
// defaulted" case: every always-required adapter capability is
// deleted alone from a whole map and must refuse naming it, and the
// writer pair deleted together must refuse naming the pair. The
// required names come from DoctorRequiredCapabilities, not from a
// hand list. A mutant defaulting an absent adapter name to usable
// admits here and reddens.
func TestCheckTargetWriteGatesAbsentAdapterRefuses(t *testing.T) {
	probe, _ := fixtureValidProbe(t)
	provider := map[string]ProviderCapability{
		"portable_store": {Status: "available", Enabled: true},
		"native_resume":  {Status: "available", Enabled: true},
	}
	required := map[string]bool{}
	for _, name := range DoctorRequiredCapabilities(DirectionTargetWrite, false) {
		required[name] = true
	}
	for _, name := range DoctorRequiredCapabilities(DirectionTargetWrite, true) {
		required[name] = true
	}
	if len(required) != 5 || !required["canonical_write"] || !required["official_import"] ||
		!required["native_read_back"] || !required["native_resume_plan"] || !required["workspace_binding"] {
		t.Fatalf("derived target-write adapter set = %v, want the five-name set", required)
	}
	// Each always-required capability deleted alone refuses naming it.
	for _, name := range []string{"native_read_back", "native_resume_plan", "workspace_binding"} {
		t.Run("absent "+name, func(t *testing.T) {
			without := cloneCapabilities(probe.Capabilities)
			delete(without, name)
			err := CheckTargetWriteGates(without, provider)
			requireRefusal(t, err, "capability_unavailable", "capability", name, "usable adapter capability")
		})
	}
	// Each writer deleted alone still passes through the other half
	// of the disjunction: absence is refused only when both are gone.
	for _, name := range []string{"canonical_write", "official_import"} {
		t.Run("absent one writer "+name, func(t *testing.T) {
			without := cloneCapabilities(probe.Capabilities)
			delete(without, name)
			if err := CheckTargetWriteGates(without, provider); err != nil {
				t.Fatalf("CheckTargetWriteGates with one writer absent: %v", err)
			}
		})
	}
	t.Run("absent both writers", func(t *testing.T) {
		without := cloneCapabilities(probe.Capabilities)
		delete(without, "canonical_write")
		delete(without, "official_import")
		err := CheckTargetWriteGates(without, provider)
		requireRefusal(t, err, "capability_unavailable", "capability", "canonical_write|official_import", "neither usable")
	})
}

// TestCapabilityUsableAbsentIsNotUsable pins the presence default
// at the unit level for every registry name: deleting any one
// capability from a whole map makes exactly that name unusable,
// and the empty or unknown name is never usable. A mutant
// defaulting an absent name to usable inverts every row here.
func TestCapabilityUsableAbsentIsNotUsable(t *testing.T) {
	probe, _ := fixtureValidProbe(t)
	for _, name := range capabilityOrder {
		if !CapabilityUsable(probe, name) {
			t.Fatalf("CapabilityUsable(%q) = false on the whole probe; the fixture is broken", name)
		}
		without := probe
		without.Capabilities = cloneCapabilities(probe.Capabilities)
		delete(without.Capabilities, name)
		if CapabilityUsable(without, name) {
			t.Fatalf("CapabilityUsable(%q) = true with the capability absent", name)
		}
		if capabilityMapUsable(without.Capabilities, name) {
			t.Fatalf("capabilityMapUsable(%q) = true with the capability absent", name)
		}
	}
	for _, name := range []string{"", "no_such_capability"} {
		if CapabilityUsable(probe, name) {
			t.Fatalf("CapabilityUsable(%q) = true; an unseen name is never usable", name)
		}
		if capabilityMapUsable(probe.Capabilities, name) {
			t.Fatalf("capabilityMapUsable(%q) = true; an unseen name is never usable", name)
		}
	}
}

// TestCheckDoctorHealthyRefusesEmptyRequiredName pins the name gate
// against the zero value: a required set naming "" is a caller
// error, never a vacuous pass. A mutant exempting the empty name
// (`!validCapabilityName(name) && name != ""`) admits here.
func TestCheckDoctorHealthyRefusesEmptyRequiredName(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpDoctor, manifestDigest)
	decoded, err := CheckRequestBody(OpDoctor, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	probe, _ := fixtureValidProbe(t)
	good := fixtureSuccessBody(t, OpDoctor, requestContextOf(t, request), manifestDigest)
	healthy, err := DecodeDoctorResult(mutateMember(t, good, "healthy", `true`), decoded, DirectionSourceRead)
	if err != nil {
		t.Fatalf("DecodeDoctorResult: %v", err)
	}
	err = CheckDoctorHealthy(healthy, probe, []string{""})
	requireRefusal(t, err, "invalid_config", "field", "required", "closed registry")
}
