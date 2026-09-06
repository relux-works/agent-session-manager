package dirnode

import (
	"testing"
)

// bootstrapRequest builds the manifest request one attempt is
// issued under.
func bootstrapRequest(t *testing.T, version string) Request {
	t.Helper()
	request, err := DecodeRequestFrame(fixtureRequestFrame(version, "manifest", fixtureRequestID, 5000, `{}`))
	if err != nil {
		t.Fatalf("DecodeRequestFrame error = %v", err)
	}
	return request
}

// TestBootstrapSelectsV2Directly drives the production bootstrap
// entry for the clean path: a v2 success with an exact echo, a
// containing manifest, and exit 0 decides Done with the manifest.
func TestBootstrapSelectsV2Directly(t *testing.T) {
	t.Parallel()
	request := bootstrapRequest(t, "2.0.0")
	decision := DecideBootstrapStep(request, fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()), 0)
	if !decision.Done || decision.Downgrade || decision.Terminal != nil {
		t.Fatalf("DecideBootstrapStep(v2 success) = %+v, want Done", decision)
	}
	if !decision.Manifest.Supports("2.0.0") {
		t.Fatal("decided manifest does not support the attempted version")
	}
}

// TestBootstrapFallsBackV2ToV1 drives the production bootstrap
// entry through the exact downgrade tuple: v2 returns the tuple
// with exit 6, the decision authorizes major 1, and a v1 success
// with exit 0 completes the bootstrap.
func TestBootstrapFallsBackV2ToV1(t *testing.T) {
	t.Parallel()
	guard := NewProcessGuard()
	if err := guard.Claim("process-v2"); err != nil {
		t.Fatalf("Claim(v2) error = %v", err)
	}
	request := bootstrapRequest(t, "2.0.0")
	decision := DecideBootstrapStep(request, fixtureDowngradeFrame("2.0.0", fixtureRequestID), 6)
	if !decision.Downgrade || decision.NextMajor != MajorV1 || decision.Terminal != nil {
		t.Fatalf("DecideBootstrapStep(v2 downgrade) = %+v, want Downgrade to 1", decision)
	}
	if err := guard.Claim("process-v1"); err != nil {
		t.Fatalf("Claim(v1) error = %v", err)
	}
	v1 := bootstrapRequest(t, "1.0.0")
	final := DecideBootstrapStep(v1, fixtureSuccessFrame("1.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()), 0)
	if !final.Done || final.Terminal != nil {
		t.Fatalf("DecideBootstrapStep(v1 success) = %+v, want Done", final)
	}
}

// TestBootstrapServesV1OnlyCaller drives the production bootstrap
// entry for a v1-only caller: no downgrade is involved and a v1
// success completes directly.
func TestBootstrapServesV1OnlyCaller(t *testing.T) {
	t.Parallel()
	request := bootstrapRequest(t, "1.0.0")
	decision := DecideBootstrapStep(request, fixtureSuccessFrame("1.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()), 0)
	if !decision.Done || decision.Terminal != nil {
		t.Fatalf("DecideBootstrapStep(v1 success) = %+v, want Done", decision)
	}
}

// TestBootstrapEndsWithoutCommonMajor drives the production
// bootstrap entry to the deterministic end: every locally
// supported major returns the exact tuple, so the last one
// terminates with the caller's own incompatible_protocol and no
// trusted manifest.
func TestBootstrapEndsWithoutCommonMajor(t *testing.T) {
	t.Parallel()
	v2 := bootstrapRequest(t, "2.0.0")
	first := DecideBootstrapStep(v2, fixtureDowngradeFrame("2.0.0", fixtureRequestID), 6)
	if !first.Downgrade {
		t.Fatalf("first step = %+v, want Downgrade", first)
	}
	v1 := bootstrapRequest(t, "1.0.0")
	last := DecideBootstrapStep(v1, fixtureDowngradeFrame("1.0.0", fixtureRequestID), 6)
	if last.Done || last.Downgrade || last.Terminal == nil {
		t.Fatalf("last step = %+v, want terminal", last)
	}
	failure := requireCode(t, last.Terminal, "incompatible_protocol")
	if failure.ExitCode() != 6 {
		t.Fatalf("terminal exit = %d, want 6", failure.ExitCode())
	}
}

// TestBootstrapRefusesNonExactDowngradeTuple drives the production
// bootstrap entry with one near-tuple per downgrade field: a
// tuple that names another code, another exit class field, claims
// retry permission, carries the wrong exit status, or mislabels
// the envelope is an ordinary terminal failure, never negotiation
// evidence and never a lower-major attempt.
func TestBootstrapRefusesNonExactDowngradeTuple(t *testing.T) {
	t.Parallel()
	request := bootstrapRequest(t, "2.0.0")
	good := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
	for _, probe := range []struct {
		name  string
		frame []byte
		exit  int
	}{
		{"other code", []byte(replaceOnce(t, good, `"code":"incompatible_protocol"`, `"code":"capability_unavailable"`, 1)), 6},
		{"retryable claim", []byte(replaceOnce(t, good, `"retryable":false`, `"retryable":true`, 1)), 6},
		{"wrong process exit", []byte(good), 0},
		{"wrong envelope version", fixtureDowngradeFrame("1.0.0", fixtureRequestID), 6},
		{"wrong request echo", fixtureDowngradeFrame("2.0.0", fixtureRequestID[:len(fixtureRequestID)-1]+"0"), 6},
		{"relabeled schema version", []byte(replaceOnce(t, good, `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1)), 6},
		{"ordinary operation failure", fixtureFailureFrame("2.0.0", "manifest", fixtureRequestID, fixtureErrorObject("capability_unavailable", 6, false)), 6},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			decision := DecideBootstrapStep(request, probe.frame, probe.exit)
			if decision.Done || decision.Downgrade {
				t.Fatalf("DecideBootstrapStep(%s) = %+v, want terminal", probe.name, decision)
			}
			if decision.Terminal == nil {
				t.Fatalf("DecideBootstrapStep(%s) has nil terminal", probe.name)
			}
		})
	}
}

// TestBootstrapTerminalFailuresNeverDowngrade drives the production
// bootstrap entry with terminal attempt outcomes: no response, an
// unreadable frame, a success whose manifest omits the attempted
// version, and a success with a nonmatching exit status. None
// authorizes a lower-major attempt.
func TestBootstrapTerminalFailuresNeverDowngrade(t *testing.T) {
	t.Parallel()
	request := bootstrapRequest(t, "2.0.0")
	v1only := replaceOnce(t, fixtureManifestJSON(), `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":["1.0.0"]`, 1)
	for _, probe := range []struct {
		name  string
		frame []byte
		exit  int
	}{
		{"no response frame", nil, 6},
		{"unreadable frame", []byte(`{"schema":`), 6},
		{"manifest omits attempted version", fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, v1only), 0},
		{"success with failing exit", fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()), 1},
		{"malformed manifest body", fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, `{"schema":"urn:ax:schema:session-directory-node-manifest"}`), 0},
	} {
		t.Run(probe.name, func(t *testing.T) {
			t.Parallel()
			decision := DecideBootstrapStep(request, probe.frame, probe.exit)
			if decision.Done || decision.Downgrade {
				t.Fatalf("DecideBootstrapStep(%s) = %+v, want terminal", probe.name, decision)
			}
			if decision.Terminal == nil {
				t.Fatalf("DecideBootstrapStep(%s) has nil terminal", probe.name)
			}
		})
	}
}

// TestBootstrapRefusesNonManifestAttempt drives the production
// bootstrap entry with a non-manifest request: bootstrap decides
// manifest attempts only.
func TestBootstrapRefusesNonManifestAttempt(t *testing.T) {
	t.Parallel()
	request, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "probe", fixtureRequestID, 1, fixtureProbeRequestV2()))
	if err != nil {
		t.Fatalf("DecodeRequestFrame error = %v", err)
	}
	decision := DecideBootstrapStep(request, fixtureSuccessFrame("2.0.0", "probe", fixtureRequestID, fixtureProbeResponseJSON()), 0)
	if decision.Done || decision.Downgrade {
		t.Fatalf("DecideBootstrapStep(probe) = %+v, want terminal", decision)
	}
	requireCode(t, decision.Terminal, "invalid_config")
}

// TestProcessGuardRefusesReuse drives the production process
// guard: a fresh token claims cleanly, a reused token refuses, and
// an empty token refuses before any set is touched.
func TestProcessGuardRefusesReuse(t *testing.T) {
	t.Parallel()
	guard := NewProcessGuard()
	if err := guard.Claim("process-a"); err != nil {
		t.Fatalf("Claim(fresh) error = %v", err)
	}
	if err := guard.Claim("process-b"); err != nil {
		t.Fatalf("Claim(second fresh) error = %v", err)
	}
	requireCode(t, guard.Claim("process-a"), "invalid_config")
	requireCode(t, guard.Claim(""), "invalid_config")
}

// TestNextLowerMajorBounds drives the production major registry:
// majors descend 2 to 1, nothing lies below 1, and unknown majors
// have no lower attempt.
func TestNextLowerMajorBounds(t *testing.T) {
	t.Parallel()
	if next, ok := NextLowerMajor(MajorV2); !ok || next != MajorV1 {
		t.Fatalf("NextLowerMajor(2) = %d,%v, want 1,true", next, ok)
	}
	if _, ok := NextLowerMajor(MajorV1); ok {
		t.Fatal("NextLowerMajor(1) must report no lower major")
	}
	if _, ok := NextLowerMajor(3); ok {
		t.Fatal("NextLowerMajor(3) must report no lower major")
	}
	if majors := SupportedMajors(); len(majors) != 2 || majors[0] != 2 || majors[1] != 1 {
		t.Fatalf("SupportedMajors() = %v, want [2 1]", majors)
	}
}
