package dirnode

import (
	"strings"
	"testing"
)

// TestProbeRequestMajorVocabularies drives the production probe
// request entry across both majors: each major admits its own
// vocabulary and refuses the other's exclusive tokens, and no
// token is coerced across majors.
func TestProbeRequestMajorVocabularies(t *testing.T) {
	t.Parallel()
	v2, err := CheckProbeRequest(MajorV2, []byte(fixtureProbeRequestV2()))
	if err != nil {
		t.Fatalf("CheckProbeRequest(v2) error = %v", err)
	}
	if v2.Platform != "linux" || v2.Architecture != "amd64" || len(v2.RequestedCapabilities) != 1 {
		t.Fatalf("CheckProbeRequest(v2) = %+v", v2)
	}
	v1, err := CheckProbeRequest(MajorV1, []byte(fixtureProbeRequestV1()))
	if err != nil {
		t.Fatalf("CheckProbeRequest(v1) error = %v", err)
	}
	if v1.Platform != "darwin" || v1.Architecture != "arm64" {
		t.Fatalf("CheckProbeRequest(v1) = %+v", v1)
	}
	for _, probe := range []struct {
		name  string
		major int
		body  string
	}{
		{"v1 refuses macos", MajorV1, `{"platform":"macos","architecture":"amd64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`},
		{"v1 refuses wsl2", MajorV1, `{"platform":"wsl2","architecture":"amd64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`},
		{"v2 refuses darwin", MajorV2, `{"platform":"darwin","architecture":"amd64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`},
		{"v2 accepts wsl2", MajorV2, `{"platform":"wsl2","architecture":"arm64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`},
		{"v2 accepts macos", MajorV2, `{"platform":"macos","architecture":"amd64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`},
		{"v1 accepts windows", MajorV1, `{"platform":"windows","architecture":"amd64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`},
		{"unknown token", MajorV2, `{"platform":"plan9","architecture":"amd64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			_, err := CheckProbeRequest(probe.major, []byte(probe.body))
			refuses := strings.Contains(probe.name, "refuses") || probe.name == "unknown token"
			if refuses && err == nil {
				t.Fatalf("CheckProbeRequest(%s) must refuse", probe.name)
			}
			if !refuses && err != nil {
				t.Fatalf("CheckProbeRequest(%s) error = %v", probe.name, err)
			}
			if refuses {
				requireCode(t, err, "adapter_protocol_violation")
			}
		})
	}
}

// TestProbeRequestRefusals drives the production probe request
// entry with one structural vector per rule.
func TestProbeRequestRefusals(t *testing.T) {
	t.Parallel()
	good := fixtureProbeRequestV2()
	for _, probe := range []struct {
		name string
		body string
	}{
		{"unknown member", replaceOnce(t, good, `"extensions":{}}`, `"extensions":{},"extra":1}`, 1)},
		{"missing member", replaceOnce(t, good, `"architecture":"amd64",`, ``, 1)},
		{"bad architecture", replaceOnce(t, good, `"amd64"`, `"riscv"`, 1)},
		{"non string platform", replaceOnce(t, good, `"platform":"linux"`, `"platform":1`, 1)},
		{"unsorted environments", replaceOnce(t, good, `"requested_environment_ids":[]`, `"requested_environment_ids":["b.env","a.env"]`, 1)},
		{"bad environment id", replaceOnce(t, good, `"requested_environment_ids":[]`, `"requested_environment_ids":["BAD"]`, 1)},
		{"too many capabilities", replaceOnce(t, good, `"requested_capabilities":["directory_discovery"]`, `"requested_capabilities":["directory_discovery","directory_head_digest","directory_incremental_scan","directory_tail_preview","existing_session_adoption","native_resume","native_runtime_observation","native_title_read","extra"]`, 1)},
		{"unknown capability", replaceOnce(t, good, `"requested_capabilities":["directory_discovery"]`, `"requested_capabilities":["directory_time_travel"]`, 1)},
		{"bad extensions", replaceOnce(t, good, `"extensions":{}}`, `"extensions":{"not-reversedns":1}}`, 1)},
		{"non object", `[]`},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := CheckProbeRequest(MajorV2, []byte(probe.body)); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
	if _, err := CheckProbeRequest(3, []byte(good)); requireCode(t, err, "invalid_config") == nil {
		t.Fatal("CheckProbeRequest(3) must refuse")
	}
}

// TestProbeResponseAcceptsFixture drives the production probe
// response entry with the exact contract vector and requires the
// node-build equality gate to accept the fixture build against
// the fixture manifest.
func TestProbeResponseAcceptsFixture(t *testing.T) {
	t.Parallel()
	response, err := CheckProbeResponse([]byte(fixtureProbeResponseJSON()))
	if err != nil {
		t.Fatalf("CheckProbeResponse error = %v", err)
	}
	if response.HostID != fixtureHostID || response.PolicyDigest != fixturePolicyDigest {
		t.Fatalf("probe response identity = %q %q", response.HostID, response.PolicyDigest)
	}
	if response.Environments != 1 || len(response.Findings) != 1 {
		t.Fatalf("probe response counts = %d env %d findings", response.Environments, len(response.Findings))
	}
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest error = %v", err)
	}
	if err := CheckNodeBuildEqualsManifest(response.Build, manifest); err != nil {
		t.Fatalf("CheckNodeBuildEqualsManifest error = %v", err)
	}
}

// TestProbeResponseAdmitsOpaqueEnvironments drives the production
// probe response entry with content-free environment objects: the
// entry checks presence and object shape only, never content —
// the shared-environment leaf owns that content, and this test
// pins the boundary so a future content check fails loudly here
// instead of arriving silently.
func TestProbeResponseAdmitsOpaqueEnvironments(t *testing.T) {
	t.Parallel()
	body := replaceOnce(t, fixtureProbeResponseJSON(), `[{"environment_id":"test.env"}]`, `[{},{}]`, 1)
	response, err := CheckProbeResponse([]byte(body))
	if err != nil {
		t.Fatalf("CheckProbeResponse(opaque) error = %v", err)
	}
	if response.Environments != 2 {
		t.Fatalf("environments = %d, want 2", response.Environments)
	}
}

// TestProbeResponseRefusals drives the production probe response
// entry with one structural vector per rule.
func TestProbeResponseRefusals(t *testing.T) {
	t.Parallel()
	good := fixtureProbeResponseJSON()
	for _, probe := range []struct {
		name string
		body string
	}{
		{"unknown member", replaceOnce(t, good, `"extensions":{}}],"extensions":{}}`, `"extensions":{}}],"extensions":{},"extra":1}`, 1)},
		{"missing member", replaceOnce(t, good, `"policy_digest":`+quote(fixturePolicyDigest)+`,`, ``, 1)},
		{"bad host id", replaceOnce(t, good, fixtureHostID, "not-a-uuid", 1)},
		{"bad policy digest", replaceOnce(t, good, fixturePolicyDigest, "sha256:zzz", 1)},
		{"node build extra member", replaceOnce(t, good, `"session_adapter_manifest_digest":`, `"session_adapter_manifest_digestX":`, 1)},
		{"non object environment", replaceOnce(t, good, `[{"environment_id":"test.env"}]`, `[1]`, 1)},
		{"too many environments", replaceOnce(t, good, `[{"environment_id":"test.env"}]`, `[`+strings.Repeat(`{},`, 257)+`{}]`, 1)},
		{"bad severity", replaceOnce(t, good, `"severity":"info"`, `"severity":"fatal"`, 1)},
		{"empty finding code", replaceOnce(t, good, `"code":"probe-ok"`, `"code":""`, 1)},
		{"empty finding message", replaceOnce(t, good, `"message":"probe completed"`, `"message":""`, 1)},
		{"non string remediation", replaceOnce(t, good, `"remediation":null`, `"remediation":0`, 1)},
		{"non object", `[]`},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			if _, err := CheckProbeResponse([]byte(probe.body)); requireCode(t, err, "adapter_protocol_violation") == nil {
				t.Fatal("must refuse")
			}
		})
	}
}

// TestProbeResponseFindingBound drives the production probe
// response entry against the finding count bound: more than 4096
// findings refuse no matter how small each one is.
func TestProbeResponseFindingBound(t *testing.T) {
	t.Parallel()
	one := `{"severity":"info","code":"c","message":"m","remediation":null,"extensions":{}}`
	many := strings.Repeat(one+`,`, 4096) + one
	body := replaceOnce(t, fixtureProbeResponseJSON(), `"findings":[{`, `"findings":[`+many+`,{`, 1)
	if _, err := CheckProbeResponse([]byte(body)); requireCode(t, err, "adapter_protocol_violation") == nil {
		t.Fatal("4098 findings must refuse")
	}
	// Exactly one past the ceiling: a mutant widening the cap by
	// one still refuses 4098, so only the 4097 case reddens it.
	plusOne := replaceOnce(t, fixtureProbeResponseJSON(), `"findings":[{`, `"findings":[`+strings.Repeat(one+`,`, 4095)+one+`,{`, 1)
	if _, err := CheckProbeResponse([]byte(plusOne)); requireCode(t, err, "adapter_protocol_violation") == nil {
		t.Fatal("4097 findings must refuse")
	}
	edge := replaceOnce(t, fixtureProbeResponseJSON(), `"findings":[{`, `"findings":[`+strings.Repeat(one+`,`, 4094)+one+`,{`, 1)
	if _, err := CheckProbeResponse([]byte(edge)); err != nil {
		t.Fatalf("4096 findings must admit: %v", err)
	}
}

// TestNodeBuildEqualityRefusesDrift drives the production
// node-build equality gate with one drifted value per binding:
// any divergence from the manifest is an integrity failure.
func TestNodeBuildEqualityRefusesDrift(t *testing.T) {
	t.Parallel()
	response, err := CheckProbeResponse([]byte(fixtureProbeResponseJSON()))
	if err != nil {
		t.Fatalf("CheckProbeResponse error = %v", err)
	}
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest error = %v", err)
	}
	for _, probe := range []struct {
		name  string
		apply func(*NodeBuild)
	}{
		{"node id", func(build *NodeBuild) { build.NodeID = "other-node" }},
		{"node version", func(build *NodeBuild) { build.NodeVersion = "9.9.9" }},
		{"executable", func(build *NodeBuild) { build.ExecutableSHA256 = fixtureDigest("other") }},
		{"provider binding", func(build *NodeBuild) { build.ProviderManifestDigest = fixtureDigest("other") }},
		{"adapter binding", func(build *NodeBuild) { build.AdapterManifestDigest = fixtureDigest("other") }},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			build := response.Build
			probe.apply(&build)
			requireCode(t, CheckNodeBuildEqualsManifest(build, manifest), "integrity_failure")
		})
	}
}
