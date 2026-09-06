package sessadapter

import (
	"testing"
)

func TestDiscoverPartialCursorComplement(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpDiscover, manifestDigest)
	decoded, err := CheckRequestBody(OpDiscover, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	facts := SuccessFacts{Context: decoded}
	base := fixtureSuccessBody(t, OpDiscover, requestContextOf(t, request), manifestDigest)
	combinations := []struct {
		name    string
		partial string
		cursor  string
		ok      bool
	}{
		{"complete without cursor", `false`, `null`, true},
		{"partial with cursor", `true`, `"cursor-9"`, true},
		{"complete with cursor", `false`, `"cursor-9"`, false},
		{"partial without cursor", `true`, `null`, false},
	}
	for _, test := range combinations {
		t.Run(test.name, func(t *testing.T) {
			body := mutateMember(t, mutateMember(t, base, "partial", test.partial), "next_cursor", test.cursor)
			err := CheckSuccessBody(OpDiscover, body, facts)
			if test.ok && err != nil {
				t.Fatalf("CheckSuccessBody: %v", err)
			}
			if !test.ok && err == nil {
				t.Fatal("admitted an incoherent partial/cursor pair")
			}
			if !test.ok {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", "partial", "disagrees with the cursor")
			}
		})
	}
}

func TestCapturePlanDigestEquality(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpCapturePlan, manifestDigest)
	decoded, err := CheckRequestBody(OpCapturePlan, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	facts := SuccessFacts{Context: decoded}
	base := fixtureSuccessBody(t, OpCapturePlan, requestContextOf(t, request), manifestDigest)
	if err := CheckSuccessBody(OpCapturePlan, base, facts); err != nil {
		t.Fatalf("CheckSuccessBody: %v", err)
	}
	mismatched := mutateMember(t, base, "capture_plan_digest", quote(fixtureDigest("other-plan")))
	err = CheckSuccessBody(OpCapturePlan, mismatched, facts)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "capture_plan_digest", "does not equal the plan digest")
}

func TestValidateResultRules(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	build := func(t *testing.T, mode string, archive bool) ([]byte, SuccessFacts) {
		request := fixtureRequestBody(t, OpValidate, manifestDigest)
		if mode != "staged" {
			request = mutateMember(t, request, "mode", quote(mode))
			if archive {
				for _, member := range []string{"projection_plan_id", "projected_object_manifest_id", "read_back_evidence_manifest_id", "expected_target_native_session_id"} {
					request = mutateMember(t, request, member, `null`)
				}
			}
		}
		decoded, err := CheckRequestBody(OpValidate, request)
		if err != nil {
			t.Fatalf("CheckRequestBody(%q): %v", mode, err)
		}
		return request, SuccessFacts{Context: decoded, ValidateMode: mode}
	}
	t.Run("staged valid", func(t *testing.T) {
		request, facts := build(t, "staged", false)
		success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
		if err := CheckSuccessBody(OpValidate, success, facts); err != nil {
			t.Fatalf("CheckSuccessBody: %v", err)
		}
	})
	t.Run("archive request carries target", func(t *testing.T) {
		request := fixtureRequestBody(t, OpValidate, manifestDigest)
		request = mutateMember(t, request, "mode", `"archive"`)
		_, err := CheckRequestBody(OpValidate, request)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "projection_plan_id", "carries a target member")
	})
	t.Run("staged request misses target", func(t *testing.T) {
		request := fixtureRequestBody(t, OpValidate, manifestDigest)
		request = mutateMember(t, request, "projection_plan_id", `null`)
		_, err := CheckRequestBody(OpValidate, request)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "projection_plan_id", "misses a target member")
	})
	t.Run("mode mismatch", func(t *testing.T) {
		request, facts := build(t, "staged", false)
		success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
		success = mutateMember(t, success, "mode", `"live"`)
		err := CheckSuccessBody(OpValidate, success, facts)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "mode", "does not echo")
	})
	t.Run("zero request mode fact", func(t *testing.T) {
		request, _ := build(t, "staged", false)
		decoded, err := CheckRequestBody(OpValidate, request)
		if err != nil {
			t.Fatalf("CheckRequestBody: %v", err)
		}
		success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
		err = CheckSuccessBody(OpValidate, success, SuccessFacts{Context: decoded})
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "mode", "does not echo")
	})
	t.Run("valid with failed check", func(t *testing.T) {
		request, facts := build(t, "staged", false)
		success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
		success = mutateMember(t, success, "structural_valid", `false`)
		err := CheckSuccessBody(OpValidate, success, facts)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "valid", "failed structural or semantic")
	})
	t.Run("valid with failed applicable check", func(t *testing.T) {
		request, facts := build(t, "staged", false)
		success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
		success = mutateMember(t, success, "identity_valid", `false`)
		err := CheckSuccessBody(OpValidate, success, facts)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "identity_valid", "failed applicable check")
	})
	t.Run("valid with error finding", func(t *testing.T) {
		request, facts := build(t, "staged", false)
		success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
		finding := `{"severity":"error","code":"val-1","message":"broken","remediation":null,"extensions":{}}`
		success = mutateMember(t, success, "findings", `[`+finding+`]`)
		err := CheckSuccessBody(OpValidate, success, facts)
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "findings", "error finding")
	})
	t.Run("invalid result admits failures", func(t *testing.T) {
		request, facts := build(t, "staged", false)
		success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
		success = mutateMember(t, success, "valid", `false`)
		success = mutateMember(t, success, "structural_valid", `false`)
		if err := CheckSuccessBody(OpValidate, success, facts); err != nil {
			t.Fatalf("CheckSuccessBody invalid result: %v", err)
		}
	})
}

// TestValidateSuccessAdmitsEveryMode drives a success body for
// every validate mode through the production entry point. The
// success body carries no target members in any mode (the pinned
// Section 7.8 validate success list names none), so the
// request-body nullability rule must not refuse it: archive mode
// was unconditionally refused while the success path shared the
// request check, with the suite green. A reintroduced
// success-side nullability call refuses the archive subtest here.
func TestValidateSuccessAdmitsEveryMode(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	for _, mode := range []string{"staged", "live", "archive"} {
		t.Run(mode, func(t *testing.T) {
			request := fixtureRequestBody(t, OpValidate, manifestDigest)
			if mode != "staged" {
				request = mutateMember(t, request, "mode", quote(mode))
			}
			if mode == "archive" {
				for _, member := range validateTargetMembers {
					request = mutateMember(t, request, member, `null`)
				}
			}
			decoded, err := CheckRequestBody(OpValidate, request)
			if err != nil {
				t.Fatalf("CheckRequestBody(%q): %v", mode, err)
			}
			facts := SuccessFacts{Context: decoded, ValidateMode: mode}
			success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
			success = mutateMember(t, success, "mode", quote(mode))
			if err := CheckSuccessBody(OpValidate, success, facts); err != nil {
				t.Fatalf("CheckSuccessBody(%q): %v", mode, err)
			}
		})
	}
}

// validateTargetMembers is the closed four-target set the
// validate mode rule ranges over, in both branches.
var validateTargetMembers = []string{"projection_plan_id", "projected_object_manifest_id", "read_back_evidence_manifest_id", "expected_target_native_session_id"}

// validateTargetCarry is one non-null value per target member for
// the archive-branch probes: digests for the manifest members,
// the native session name for the session member.
func validateTargetCarry(member string) string {
	if member == "expected_target_native_session_id" {
		return `"target-1"`
	}
	return quote(fixturePlanDigest)
}

// TestCheckValidateNullabilityDrivesEveryTarget breaks every
// member of the four-target set alone in both mode branches
// through the request entry point: archive carrying any single
// target refuses naming it, and staged missing any single target
// refuses naming it. A loop truncation (targets[:3]) admits the
// fourth member and reddens here, which a one-member probe
// cannot catch. The obligation set is written here rather than
// read from production so a narrowed production list reddens on
// the missing member instead of shrinking the probe: the member
// census and the operation table already pin the four names from
// the other side.
func TestCheckValidateNullabilityDrivesEveryTarget(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	t.Run("archive carries each target", func(t *testing.T) {
		base := fixtureRequestBody(t, OpValidate, manifestDigest)
		archived := mutateMember(t, base, "mode", `"archive"`)
		for _, member := range validateTargetMembers {
			archived = mutateMember(t, archived, member, `null`)
		}
		if _, err := CheckRequestBody(OpValidate, archived); err != nil {
			t.Fatalf("archive request without targets: %v", err)
		}
		for _, member := range validateTargetMembers {
			t.Run(member, func(t *testing.T) {
				_, err := CheckRequestBody(OpValidate, mutateMember(t, archived, member, validateTargetCarry(member)))
				requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "carries a target member")
			})
		}
	})
	t.Run("staged misses each target", func(t *testing.T) {
		base := fixtureRequestBody(t, OpValidate, manifestDigest)
		if _, err := CheckRequestBody(OpValidate, base); err != nil {
			t.Fatalf("staged request: %v", err)
		}
		for _, member := range validateTargetMembers {
			t.Run(member, func(t *testing.T) {
				_, err := CheckRequestBody(OpValidate, mutateMember(t, base, member, `null`))
				requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "misses a target member")
			})
		}
	})
}

// TestCheckValidateResultDrivesEveryApplicableCheck fails every
// applicable check alone under valid=true: each of the three
// nullable checks false refuses naming that check. A dropped
// member admits a false check and reddens here.
func TestCheckValidateResultDrivesEveryApplicableCheck(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpValidate, manifestDigest)
	decoded, err := CheckRequestBody(OpValidate, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	facts := SuccessFacts{Context: decoded, ValidateMode: "staged"}
	for _, member := range []string{"identity_valid", "workspace_binding_valid", "resume_surface_valid"} {
		t.Run(member, func(t *testing.T) {
			success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
			err := CheckSuccessBody(OpValidate, mutateMember(t, success, member, `false`), facts)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "failed applicable check")
		})
	}
	t.Run("non boolean check member", func(t *testing.T) {
		for _, member := range []string{"identity_valid", "workspace_binding_valid", "resume_surface_valid"} {
			success := fixtureSuccessBody(t, OpValidate, requestContextOf(t, request), manifestDigest)
			err := CheckSuccessBody(OpValidate, mutateMember(t, success, member, `"yes"`), facts)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "not a boolean")
		}
	})
}

// TestResumePlanIdentityComplement drives zero, one, and two
// occurrences of the explicit identity: exactly one admits, and
// both neighbours refuse.
func TestResumePlanIdentityComplement(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpResumePlan, manifestDigest)
	decoded, err := CheckRequestBody(OpResumePlan, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	facts := SuccessFacts{Context: decoded, ResumeTargetID: "target-1"}
	base := fixtureSuccessBody(t, OpResumePlan, requestContextOf(t, request), manifestDigest)
	if err := CheckSuccessBody(OpResumePlan, base, facts); err != nil {
		t.Fatalf("CheckSuccessBody: %v", err)
	}
	absent := mutateMember(t, base, "argv", `["ax","open"]`)
	err = CheckSuccessBody(OpResumePlan, absent, facts)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "argv", "exactly once")
	doubled := mutateMember(t, base, "argv", `["ax","open","target-1","target-1"]`)
	err = CheckSuccessBody(OpResumePlan, doubled, facts)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "argv", "exactly once")
	closed := mutateMember(t, base, "opens_existing_identity", `false`)
	err = CheckSuccessBody(OpResumePlan, closed, facts)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "opens_existing_identity", "existing identity")
}

func TestProjectionPlanDispositionRules(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpProjectionPlan, manifestDigest)
	if _, err := CheckRequestBody(OpProjectionPlan, request); err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	archived := mutateMember(t, request, "required_dispositions", `{"user_message":["archive_only"]}`)
	_, err := CheckRequestBody(OpProjectionPlan, archived)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "user_message", "seven-disposition vocabulary")
	eighth := mutateMember(t, request, "required_dispositions", `{"user_message":["exact","bogus"]}`)
	_, err = CheckRequestBody(OpProjectionPlan, eighth)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "user_message", "seven-disposition vocabulary")
	unsorted := mutateMember(t, request, "required_dispositions", `{"user_message":["summarized","exact"]}`)
	_, err = CheckRequestBody(OpProjectionPlan, unsorted)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "user_message", "not sorted unique")
	empty := mutateMember(t, request, "required_dispositions", `{}`)
	_, err = CheckRequestBody(OpProjectionPlan, empty)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "required_dispositions", "no class")
	fifth := mutateMember(t, request, "fidelity_profile", `"archive_only"`)
	_, err = CheckRequestBody(OpProjectionPlan, fifth)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "fidelity_profile", "four-profile vocabulary")
	duplicated := mutateMember(t, request, "required_dispositions", `{"user_message":["exact","exact"]}`)
	_, err = CheckRequestBody(OpProjectionPlan, duplicated)
	if err == nil {
		t.Fatal("admitted a duplicated disposition")
	}
}

func TestDoctorResultRules(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpDoctor, manifestDigest)
	decoded, err := CheckRequestBody(OpDoctor, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	probe, _ := fixtureValidProbe(t)
	good := func() []byte {
		return fixtureSuccessBody(t, OpDoctor, requestContextOf(t, request), manifestDigest)
	}
	facts := SuccessFacts{Context: decoded, DoctorDirection: DirectionSourceRead}
	if err := CheckSuccessBody(OpDoctor, good(), facts); err != nil {
		t.Fatalf("CheckSuccessBody doctor: %v", err)
	}
	mismatched := mutateMember(t, good(), "direction", `"target_write"`)
	err = CheckSuccessBody(OpDoctor, mismatched, facts)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "direction", "request direction")
	badStatus := mutateMember(t, good(), "registry_entry_status", `"pending"`)
	err = CheckSuccessBody(OpDoctor, badStatus, facts)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "registry_entry_status", "accepted|revoked|absent")
	result, err := DecodeDoctorResult(good(), decoded, DirectionSourceRead)
	if err != nil {
		t.Fatalf("DecodeDoctorResult: %v", err)
	}
	if err := CheckDoctorHealthy(result, probe, nil); err != nil {
		t.Fatalf("CheckDoctorHealthy unhealthy: %v", err)
	}
	revoked := mutateMember(t, good(), "registry_entry_status", `"revoked"`)
	revoked = mutateMember(t, revoked, "healthy", `true`)
	revokedResult, err := DecodeDoctorResult(revoked, decoded, DirectionSourceRead)
	if err != nil {
		t.Fatalf("DecodeDoctorResult revoked: %v", err)
	}
	err = CheckDoctorHealthy(revokedResult, probe, nil)
	requireRefusal(t, err, "capability_unavailable", "capability", "tuple_registry", "accepted registry entry")
	healthy := mutateMember(t, good(), "healthy", `true`)
	healthyResult, err := DecodeDoctorResult(healthy, decoded, DirectionSourceRead)
	if err != nil {
		t.Fatalf("DecodeDoctorResult healthy: %v", err)
	}
	weakened := probe
	weakened.Capabilities = cloneCapabilities(probe.Capabilities)
	weakened.Capabilities["native_discovery"] = Capability{Status: "conditional", Enabled: false, Evidence: "probed"}
	err = CheckDoctorHealthy(healthyResult, weakened, []string{"native_discovery"})
	requireRefusal(t, err, "capability_unavailable", "capability", "native_discovery", "usable required capability")
	if err := CheckDoctorHealthy(healthyResult, probe, []string{"native_discovery"}); err != nil {
		t.Fatalf("CheckDoctorHealthy healthy: %v", err)
	}
	err = CheckDoctorHealthy(healthyResult, probe, []string{"teleport"})
	requireRefusal(t, err, "invalid_config", "field", "required", "closed registry")
}
