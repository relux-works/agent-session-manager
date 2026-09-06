package sessadapter

import (
	"fmt"
	"strings"
	"testing"
)

// fixtureValidProbe decodes the fixture probe and returns it with
// the manifest digest the fixture manifest computes to.
func fixtureValidProbe(t *testing.T) (Probe, Manifest) {
	t.Helper()
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	digest := ManifestDigest(manifest).String()
	probe, err := DecodeProbe([]byte(fixtureProbeJSON(digest)))
	if err != nil {
		t.Fatalf("DecodeProbe: %v", err)
	}
	return probe, manifest
}

func TestDecodeProbeAcceptsFixture(t *testing.T) {
	probe, _ := fixtureValidProbe(t)
	if len(probe.Capabilities) != 15 {
		t.Fatalf("capabilities = %d, want 15", len(probe.Capabilities))
	}
	if probe.ProviderID != fixtureProviderID {
		t.Fatalf("ProviderID = %q, want %q", probe.ProviderID, fixtureProviderID)
	}
	if probe.Environment.EnvironmentID != fixtureEnvironmentID {
		t.Fatalf("EnvironmentID = %q, want %q", probe.Environment.EnvironmentID, fixtureEnvironmentID)
	}
	for _, name := range capabilityOrder {
		if !CapabilityUsable(probe, name) {
			t.Fatalf("CapabilityUsable(%q) = false, want true", name)
		}
	}
}

// TestCapabilityStatusMatrix drives every status/enabled
// combination through the value decoder: only
// available+enabled is usable, and only available permits
// enabled=true. The eight combinations are enumerated, not
// sampled.
func TestCapabilityStatusMatrix(t *testing.T) {
	for _, status := range []string{"available", "conditional", "unsupported", "unknown"} {
		for _, enabled := range []bool{false, true} {
			name := fmt.Sprintf("%s/enabled=%v", status, enabled)
			t.Run(name, func(t *testing.T) {
				body := fmt.Sprintf(`{"status":%q,"enabled":%v,"evidence":"probed","detail":""}`, status, enabled)
				probe, _ := fixtureValidProbe(t)
				// Rebuild the probe body with one value replaced.
				fixture := fixtureProbeJSON(probe.ManifestDigest)
				mutated := replaceCapability(t, []byte(fixture), "tool_history", body)
				redone, err := DecodeProbe(mutated)
				if enabled && status != "available" {
					requireRefusal(t, err, "session_adapter_protocol_error", "member", "tool_history", "without available status")
					return
				}
				if err != nil {
					t.Fatalf("DecodeProbe: %v", err)
				}
				wantUsable := status == "available" && enabled
				if CapabilityUsable(redone, "tool_history") != wantUsable {
					t.Fatalf("CapabilityUsable = %v, want %v", !wantUsable, wantUsable)
				}
			})
		}
	}
}

// TestCapabilityNonAvailableEnabledIsNotUsable settles review
// rev5/F3 at the unit level: the decode coherence gate
// (decodeCapabilityValue refuses enabled-without-available, pinned
// by TestCapabilityStatusMatrix) keeps DECODED probes clean, but
// CapabilityUsable also takes caller-built probes, so its ==
// "available" clause must independently refuse a non-available
// status even with enabled=true. A mutant widening the clause to
// admit "conditional" makes the hand-built conditional+enabled
// row usable and fails here. Unsupported and unknown pin the same
// clause from the other two members of the rejected class.
func TestCapabilityNonAvailableEnabledIsNotUsable(t *testing.T) {
	probe, _ := fixtureValidProbe(t)
	for _, status := range []string{"conditional", "unsupported", "unknown"} {
		for _, enabled := range []bool{true, false} {
			name := status + "/enabled"
			if !enabled {
				name = status + "/disabled"
			}
			t.Run(name, func(t *testing.T) {
				built := probe
				built.Capabilities = cloneCapabilities(probe.Capabilities)
				built.Capabilities["tool_history"] = Capability{Status: status, Enabled: enabled, Evidence: "probed"}
				if CapabilityUsable(built, "tool_history") {
					t.Fatalf("CapabilityUsable = true for hand-built %s with enabled=%v; only available+enabled is usable", status, enabled)
				}
				if capabilityMapUsable(built.Capabilities, "tool_history") {
					t.Fatalf("capabilityMapUsable = true for hand-built %s with enabled=%v", status, enabled)
				}
			})
		}
	}
	t.Run("available enabled stays usable", func(t *testing.T) {
		if !CapabilityUsable(probe, "tool_history") {
			t.Fatal("CapabilityUsable = false for the decoded available+enabled fixture; the fixture is broken, not the gate")
		}
	})
}

// TestCheckTargetWriteGatesRefusesNonAvailableEnabled settles
// review rev5/F3 at the gate: a caller-built adapter map carrying
// conditional+enabled on any required capability refuses, so the
// admit-conditional mutant (which leaves the full suite green
// without this test) fails here naming the broken capability.
// The provider side refuses a conditional+enabled surface too.
func TestCheckTargetWriteGatesRefusesNonAvailableEnabled(t *testing.T) {
	probe, _ := fixtureValidProbe(t)
	provider := map[string]ProviderCapability{
		"portable_store": {Status: "available", Enabled: true},
		"native_resume":  {Status: "available", Enabled: true},
	}
	t.Run("conditional writer pair", func(t *testing.T) {
		without := cloneCapabilities(probe.Capabilities)
		without["canonical_write"] = Capability{Status: "conditional", Enabled: true, Evidence: "probed"}
		without["official_import"] = Capability{Status: "unsupported", Enabled: false, Evidence: "probed"}
		err := CheckTargetWriteGates(without, provider)
		requireRefusal(t, err, "capability_unavailable", "capability", "canonical_write|official_import", "neither usable")
	})
	for _, name := range []string{"native_read_back", "native_resume_plan", "workspace_binding"} {
		t.Run("conditional "+name, func(t *testing.T) {
			without := cloneCapabilities(probe.Capabilities)
			without[name] = Capability{Status: "conditional", Enabled: true, Evidence: "probed"}
			err := CheckTargetWriteGates(without, provider)
			requireRefusal(t, err, "capability_unavailable", "capability", name, "usable adapter capability")
		})
	}
	t.Run("conditional provider", func(t *testing.T) {
		conditional := map[string]ProviderCapability{
			"portable_store": {Status: "conditional", Enabled: true},
			"native_resume":  {Status: "available", Enabled: true},
		}
		err := CheckTargetWriteGates(probe.Capabilities, conditional)
		requireRefusal(t, err, "capability_unavailable", "capability", "provider:portable_store", "usable provider capability")
	})
}

// replaceCapability swaps one capability value in a probe body.
func replaceCapability(t *testing.T, body []byte, name, value string) []byte {
	t.Helper()
	marker := `"` + name + `":` + fixtureCapabilityJSON()
	if !strings.Contains(string(body), marker) {
		t.Fatalf("replaceCapability: %q not in fixture", name)
	}
	return []byte(strings.Replace(string(body), marker, `"`+name+`":`+value, 1))
}

func TestDecodeProbeClosedRules(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	digest := ManifestDigest(manifest).String()
	fixture := []byte(fixtureProbeJSON(digest))
	members := []string{"schema", "schema_version", "provider_id", "adapter_manifest_digest", "adapter_version", "environment", "capabilities", "warnings", "extensions"}
	for _, member := range members {
		t.Run("missing "+member, func(t *testing.T) {
			_, err := DecodeProbe(dropMember(t, fixture, member))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "misses a required member")
		})
	}
	t.Run("unknown member", func(t *testing.T) {
		_, err := DecodeProbe(appendMember(t, fixture, "unexpected", `true`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "unexpected", "unknown member")
	})
	t.Run("omit capability", func(t *testing.T) {
		mutated := replaceCapability(t, fixture, "tool_history", `{"status":"available","enabled":true,"evidence":"probed","detail":""}`)
		_ = mutated
		dropped := dropCapability(t, fixture, "tool_history")
		_, err := DecodeProbe(dropped)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "capabilities", "exactly fifteen")
	})
	t.Run("extra capability", func(t *testing.T) {
		added := addCapability(t, fixture, "gui_clipboard", fixtureCapabilityJSON())
		_, err := DecodeProbe(added)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "capabilities", "exactly fifteen")
	})
	t.Run("bad evidence", func(t *testing.T) {
		mutated := replaceCapability(t, fixture, "tool_history", `{"status":"available","enabled":true,"evidence":"vibes","detail":""}`)
		_, err := DecodeProbe(mutated)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "tool_history", "seven-evidence vocabulary")
	})
	t.Run("bad status", func(t *testing.T) {
		// A fifth status is refused even though the matrix above
		// admits the four closed values: the status gate admits
		// exactly the Section 7.8 union, and a mutant weakening
		// validCapabilityStatus to admit anything fails here.
		mutated := replaceCapability(t, fixture, "tool_history", `{"status":"bogus","enabled":false,"evidence":"probed","detail":""}`)
		_, err := DecodeProbe(mutated)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "tool_history", "outside available|conditional|unsupported|unknown")
	})
	t.Run("detail too long", func(t *testing.T) {
		mutated := replaceCapability(t, fixture, "tool_history", `{"status":"available","enabled":true,"evidence":"probed","detail":"`+strings.Repeat("d", 2049)+`"}`)
		_, err := DecodeProbe(mutated)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "tool_history", "string[0..2048]")
	})
	t.Run("unsorted warnings", func(t *testing.T) {
		_, err := DecodeProbe(mutateMember(t, fixture, "warnings", `["b","a"]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "warnings", "sorted unique")
	})
	t.Run("duplicate warnings", func(t *testing.T) {
		_, err := DecodeProbe(mutateMember(t, fixture, "warnings", `["a","a"]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "warnings", "sorted unique")
	})
	t.Run("warning too long", func(t *testing.T) {
		_, err := DecodeProbe(mutateMember(t, fixture, "warnings", `["`+strings.Repeat("w", 2049)+`"]`))
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "warnings", "sorted unique")
	})
}

// dropCapability removes one capability from a probe body.
func dropCapability(t *testing.T, body []byte, name string) []byte {
	t.Helper()
	marker := `"` + name + `":` + fixtureCapabilityJSON() + `,`
	if strings.Contains(string(body), marker) {
		return []byte(strings.Replace(string(body), marker, ``, 1))
	}
	marker = `,` + `"` + name + `":` + fixtureCapabilityJSON()
	if strings.Contains(string(body), marker) {
		return []byte(strings.Replace(string(body), marker, ``, 1))
	}
	t.Fatalf("dropCapability: %q not in fixture", name)
	return nil
}

// addCapability inserts one extra capability into a probe body.
func addCapability(t *testing.T, body []byte, name, value string) []byte {
	t.Helper()
	marker := `"capabilities":{`
	if !strings.Contains(string(body), marker) {
		t.Fatalf("addCapability: capabilities map not in fixture")
	}
	return []byte(strings.Replace(string(body), marker, marker+`"`+name+`":`+value+`,`, 1))
}

func TestCheckProbeEqualities(t *testing.T) {
	probe, manifest := fixtureValidProbe(t)
	digest := ManifestDigest(manifest).String()
	facts := ProbeHostFacts{
		ExpectedProviderID: fixtureProviderID,
		ExpectedKind:       CandidateBuiltin,
		ManifestDigest:     digest,
		Manifest:           manifest,
	}
	if err := CheckProbe(probe, facts); err != nil {
		t.Fatalf("CheckProbe: %v", err)
	}
	// Every equality is broken alone: provider vs request, provider
	// vs manifest, digest, version, tuple environment, and tuple
	// adapter version.
	otherDigest := fixtureDigest("other-manifest")
	for _, test := range []struct {
		name  string
		probe Probe
		facts ProbeHostFacts
		key   string
		value string
		text  string
	}{
		{
			name: "provider vs request", probe: probe,
			facts: ProbeHostFacts{ExpectedProviderID: "other-provider", ExpectedKind: CandidateBuiltin, ManifestDigest: digest, Manifest: manifest},
			key:   "member", value: "provider_id", text: "requested provider",
		},
		{
			name:  "provider vs manifest",
			probe: func() Probe { mutated := probe; mutated.ProviderID = "other-provider"; return mutated }(),
			facts: func() ProbeHostFacts { fixed := facts; fixed.ExpectedProviderID = "other-provider"; return fixed }(),
			key:   "member", value: "provider_id", text: "verified manifest",
		},
		{
			name:  "manifest digest",
			probe: func() Probe { mutated := probe; mutated.ManifestDigest = otherDigest; return mutated }(),
			facts: facts, key: "member", value: "adapter_manifest_digest", text: "host-computed digest",
		},
		{
			name:  "adapter version",
			probe: func() Probe { mutated := probe; mutated.AdapterVersion = "9.9.9"; return mutated }(),
			facts: facts, key: "member", value: "adapter_version", text: "verified manifest",
		},
		{
			name:  "unrequested tuple",
			probe: func() Probe { mutated := probe; mutated.Environment.EnvironmentID = "other.env"; return mutated }(),
			facts: facts, key: "environment", value: "other.env", text: "outside the manifest environment",
		},
		{
			name:  "tuple version disagrees",
			probe: func() Probe { mutated := probe; mutated.Environment.AdapterVersion = "0.0.1"; return mutated }(),
			facts: facts, key: "member", value: "environment", text: "probe adapter version",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := CheckProbe(test.probe, test.facts)
			if test.key == "environment" && (test.name == "unrequested tuple") {
				requireRefusal(t, err, "unsupported_environment_tuple", test.key, test.value, test.text)
				return
			}
			requireRefusal(t, err, "session_adapter_protocol_error", test.key, test.value, test.text)
		})
	}
}

// TestZeroValueHostFactsRefuse pins the fail-closed half of the
// host-fact equality gates: a caller that forgets a fact field
// gets a refusal, never an admission. Each zero field with a
// non-empty probe side refuses through its own arm, so a mutant
// that only refuses when the expected side is non-empty
// (`!= expected && expected != ""`) slides the arm or admits,
// and reddens here.
func TestZeroValueHostFactsRefuse(t *testing.T) {
	probe, manifest := fixtureValidProbe(t)
	digest := ManifestDigest(manifest).String()
	t.Run("zero expected provider", func(t *testing.T) {
		err := CheckProbe(probe, ProbeHostFacts{})
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "provider_id", "requested provider")
	})
	t.Run("zero manifest environment", func(t *testing.T) {
		empty := manifest
		empty.EnvironmentID = ""
		facts := ProbeHostFacts{
			ExpectedProviderID: fixtureProviderID,
			ExpectedKind:       CandidateBuiltin,
			ManifestDigest:     digest,
			Manifest:           empty,
		}
		err := CheckProbe(probe, facts)
		requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "outside the manifest environment")
	})
	t.Run("zero expected candidate kind", func(t *testing.T) {
		err := CheckProbeRequest(fixtureProviderID, "", fixtureCandidate())
		requireRefusal(t, err, "invalid_config", "field", "expected_candidate_kind", "trusted candidate")
	})
}

func TestCheckProbeRequest(t *testing.T) {
	candidate := fixtureCandidate()
	if err := CheckProbeRequest(fixtureProviderID, CandidateBuiltin, candidate); err != nil {
		t.Fatalf("CheckProbeRequest: %v", err)
	}
	if err := CheckProbeRequest("other-provider", CandidateBuiltin, candidate); err == nil {
		t.Fatal("CheckProbeRequest admitted a foreign provider identifier")
	} else {
		requireRefusal(t, err, "invalid_config", "field", "expected_provider_id", "trusted candidate")
	}
	if err := CheckProbeRequest(fixtureProviderID, CandidateExternal, candidate); err == nil {
		t.Fatal("CheckProbeRequest admitted a foreign candidate kind")
	} else {
		requireRefusal(t, err, "invalid_config", "field", "expected_candidate_kind", "trusted candidate")
	}
}

// TestCheckDoctorHealthyDrivesEveryRequiredCapability derives
// the required set from DoctorRequiredCapabilities and breaks
// every member alone: a healthy accepted result passes whole,
// and weakening any single required capability refuses naming
// that capability. A loop truncation (required[:1]) admits a
// weakened later member and reddens here, which a one-member
// probe cannot catch.
func TestCheckDoctorHealthyDrivesEveryRequiredCapability(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpDoctor, manifestDigest)
	decoded, err := CheckRequestBody(OpDoctor, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	probe, _ := fixtureValidProbe(t)
	good := fixtureSuccessBody(t, OpDoctor, requestContextOf(t, request), manifestDigest)
	healthy := mutateMember(t, good, "healthy", `true`)
	for _, official := range []bool{false, true} {
		required := DoctorRequiredCapabilities(DirectionTargetWrite, official)
		if len(required) != 4 {
			t.Fatalf("official=%v required = %v, want the derived four-capability set", official, required)
		}
		result, err := DecodeDoctorResult(healthy, decoded, DirectionSourceRead)
		if err != nil {
			t.Fatalf("DecodeDoctorResult: %v", err)
		}
		if err := CheckDoctorHealthy(result, probe, required); err != nil {
			t.Fatalf("official=%v CheckDoctorHealthy whole: %v", official, err)
		}
		for _, name := range required {
			t.Run(fmt.Sprintf("official=%v/%s", official, name), func(t *testing.T) {
				weakened := probe
				weakened.Capabilities = cloneCapabilities(probe.Capabilities)
				weakened.Capabilities[name] = Capability{Status: "conditional", Enabled: false, Evidence: "probed"}
				err := CheckDoctorHealthy(result, weakened, required)
				requireRefusal(t, err, "capability_unavailable", "capability", name, "usable required capability")
			})
		}
	}
}

func TestCheckTargetWriteGates(t *testing.T) {
	probe, _ := fixtureValidProbe(t)
	provider := map[string]ProviderCapability{
		"portable_store": {Status: "available", Enabled: true},
		"native_resume":  {Status: "available", Enabled: true},
	}
	if err := CheckTargetWriteGates(probe.Capabilities, provider); err != nil {
		t.Fatalf("CheckTargetWriteGates: %v", err)
	}
	// The disjunction: disabling canonical_write alone still
	// passes through official_import, and vice versa.
	withoutCanonical := cloneCapabilities(probe.Capabilities)
	withoutCanonical["canonical_write"] = Capability{Status: "unsupported", Enabled: false, Evidence: "probed"}
	if err := CheckTargetWriteGates(withoutCanonical, provider); err != nil {
		t.Fatalf("CheckTargetWriteGates without canonical_write: %v", err)
	}
	withoutImport := cloneCapabilities(probe.Capabilities)
	withoutImport["official_import"] = Capability{Status: "unsupported", Enabled: false, Evidence: "probed"}
	if err := CheckTargetWriteGates(withoutImport, provider); err != nil {
		t.Fatalf("CheckTargetWriteGates without official_import: %v", err)
	}
	// Both members of the pair down: refused, naming the pair.
	withoutBoth := cloneCapabilities(probe.Capabilities)
	withoutBoth["canonical_write"] = Capability{Status: "unsupported", Enabled: false, Evidence: "probed"}
	withoutBoth["official_import"] = Capability{Status: "conditional", Enabled: false, Evidence: "probed"}
	err := CheckTargetWriteGates(withoutBoth, provider)
	requireRefusal(t, err, "capability_unavailable", "capability", "canonical_write|official_import", "neither usable")
	// Each remaining required adapter capability is broken alone.
	for _, name := range []string{"native_read_back", "native_resume_plan", "workspace_binding"} {
		without := cloneCapabilities(probe.Capabilities)
		without[name] = Capability{Status: "conditional", Enabled: false, Evidence: "probed"}
		err := CheckTargetWriteGates(without, provider)
		requireRefusal(t, err, "capability_unavailable", "capability", name, "usable adapter capability")
	}
	// Each provider capability is broken alone, including the
	// enabled-bit half: available-but-disabled is not usable.
	for _, name := range []string{"portable_store", "native_resume"} {
		unsupported := map[string]ProviderCapability{
			"portable_store": {Status: "available", Enabled: true},
			"native_resume":  {Status: "available", Enabled: true},
		}
		unsupported[name] = ProviderCapability{Status: "unsupported", Enabled: false}
		err := CheckTargetWriteGates(probe.Capabilities, unsupported)
		requireRefusal(t, err, "capability_unavailable", "capability", "provider:"+name, "usable provider capability")
		disabled := map[string]ProviderCapability{
			"portable_store": {Status: "available", Enabled: true},
			"native_resume":  {Status: "available", Enabled: true},
		}
		disabled[name] = ProviderCapability{Status: "available", Enabled: false}
		err = CheckTargetWriteGates(probe.Capabilities, disabled)
		requireRefusal(t, err, "capability_unavailable", "capability", "provider:"+name, "usable provider capability")
	}
	// A missing provider surface is refused, never defaulted.
	missing := map[string]ProviderCapability{
		"portable_store": {Status: "available", Enabled: true},
	}
	err = CheckTargetWriteGates(probe.Capabilities, missing)
	requireRefusal(t, err, "capability_unavailable", "capability", "provider:native_resume", "usable provider capability")
}

func cloneCapabilities(capabilities map[string]Capability) map[string]Capability {
	cloned := make(map[string]Capability, len(capabilities))
	for name, value := range capabilities {
		cloned[name] = value
	}
	return cloned
}

// TestTargetWriteComplementDerivesNonRequired pins the other side
// of the gate: every registry capability outside the required set
// is not required, so a mutant that adds a requirement fails here
// with the complement named.
func TestTargetWriteComplementDerivesNonRequired(t *testing.T) {
	required := map[string]bool{
		"canonical_write": true, "official_import": true,
		"native_read_back": true, "native_resume_plan": true, "workspace_binding": true,
		"provider:portable_store": true, "provider:native_resume": true,
	}
	var complement []string
	for _, name := range capabilityOrder {
		if !required[name] {
			complement = append(complement, name)
		}
	}
	if len(complement) == 0 {
		t.Fatal("derived an empty non-required complement; the check is blind")
	}
	probe, _ := fixtureValidProbe(t)
	provider := map[string]ProviderCapability{
		"portable_store": {Status: "available", Enabled: true},
		"native_resume":  {Status: "available", Enabled: true},
	}
	// Disabling every complement capability at once still passes:
	// none of them gates a target write.
	disabled := cloneCapabilities(probe.Capabilities)
	for _, name := range complement {
		disabled[name] = Capability{Status: "unsupported", Enabled: false, Evidence: "probed"}
	}
	if err := CheckTargetWriteGates(disabled, provider); err != nil {
		t.Fatalf("CheckTargetWriteGates with complement disabled: %v", err)
	}
	t.Logf("non-required complement: %d/%d capabilities", len(complement), len(capabilityOrder))
}

func TestDoctorRequiredCapabilities(t *testing.T) {
	target := DoctorRequiredCapabilities(DirectionTargetWrite, false)
	if len(target) != 4 || target[0] != "canonical_write" {
		t.Fatalf("target required = %v, want [canonical_write native_read_back native_resume_plan workspace_binding]", target)
	}
	official := DoctorRequiredCapabilities(DirectionTargetWrite, true)
	if len(official) != 4 || official[0] != "official_import" {
		t.Fatalf("official-import required = %v, want official_import first", official)
	}
	if got := DoctorRequiredCapabilities(DirectionSourceRead, false); len(got) != 0 {
		t.Fatalf("source required = %v, want empty: Section 7.8 names no source set", got)
	}
}
