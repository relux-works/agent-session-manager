package sessadapter

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Request builders: one valid request body per operation. Contexts
// name the real fixture manifest digest so bodies bind to sealed
// bindings downstream.

func requestContext(t *testing.T, manifestDigest string) (string, CallContext) {
	t.Helper()
	skeleton := fmt.Sprintf(`{"context":%s,"authority":%s,"workspace_filter":null,"limit":50,"cursor":null,"extensions":{}}`,
		contextSkeleton(manifestDigest, "sha256:"+strings.Repeat("0", 64)), fixtureReadAuthorityJSON())
	digest, err := RequestDigestFor([]byte(skeleton))
	if err != nil {
		t.Fatalf("RequestDigestFor: %v", err)
	}
	body := fmt.Sprintf(`{"context":%s,"authority":%s,"workspace_filter":null,"limit":50,"cursor":null,"extensions":{}}`,
		contextSkeleton(manifestDigest, digest.String()), fixtureReadAuthorityJSON())
	context, err := CheckRequestBody(OpDiscover, []byte(body))
	if err != nil {
		t.Fatalf("requestContext: %v", err)
	}
	return contextSkeleton(manifestDigest, digest.String()), context
}

func contextSkeleton(manifestDigest, requestDigest string) string {
	return fmt.Sprintf(`{"operation_id":%q,"provider_id":%q,"environment":%s,"session_adapter_manifest_digest":%q,"executable_sha256":%q,"request_digest":%q,"extensions":{}}`,
		fixtureOperationID, fixtureProviderID, fixtureTupleJSON(), manifestDigest, fixtureExecutableDigest, requestDigest)
}

// fixtureRequestBody builds one valid request body per operation.
// The manifest digest binds every context to the fixture manifest.
func fixtureRequestBody(t *testing.T, operation Operation, manifestDigest string) []byte {
	t.Helper()
	authority := fixtureReadAuthorityJSON()
	source := `{"native_session_id":"sess-1","logical_workspace_id":null,"opaque_source_ref":null}`
	boundary := `{}`
	contextJSON, _ := requestContext(t, manifestDigest)
	ctx := `"context":` + contextJSON
	switch operation {
	case OpManifest:
		return []byte(`{}`)
	case OpProbe:
		return []byte(`{"expected_provider_id":` + quote(fixtureProviderID) + `,"expected_candidate_kind":"builtin","extensions":{}}`)
	case OpDiscover:
		return []byte(`{` + ctx + `,"authority":` + authority + `,"workspace_filter":null,"limit":50,"cursor":null,"extensions":{}}`)
	case OpInspect:
		return []byte(`{` + ctx + `,"authority":` + authority + `,"source":` + source + `,"extensions":{}}`)
	case OpSnapshotProof:
		return []byte(`{` + ctx + `,"authority":` + authority + `,"source":` + source + `,"expected_source_store_generation":"gen-7","allow_provider_quiescence":false,"extensions":{}}`)
	case OpCapturePlan:
		return []byte(`{` + ctx + `,"authority":` + authority + `,"source":` + source + `,"capture_boundary":` + boundary + `,"plan_sink":` + fixtureFreshSinkJSON("capture_plan") + `,"max_items":10,"max_total_bytes":1000,"extensions":{}}`)
	case OpCapture:
		return []byte(`{` + ctx + `,"source_authority":` + authority + `,"sink":` + fixtureFreshSinkJSON("raw_source") + `,"source":` + source + `,"capture_boundary":` + boundary + `,"capture_plan_digest":` + quote(fixturePlanDigest) + `,"extensions":{}}`)
	case OpNormalize:
		return []byte(`{` + ctx + `,"capture_manifest_id":` + quote(fixtureCaptureDigest) + `,"raw_objects":` + fixtureFreshSinkJSON("raw_source") + `,"canonical_sink":` + fixtureFreshSinkJSON("canonical_source") + `,"extensions":{}}`)
	case OpProjectionPlan:
		return []byte(`{` + ctx + `,"capture_manifest_id":` + quote(fixtureCaptureDigest) + `,"canonical_session_id":` + quote(fixtureSessionDigest) + `,"canonical_event_ids":[],"source_objects":` + fixtureFreshSinkJSON("canonical_source") + `,"plan_sink":` + fixtureFreshSinkJSON("projection_plan") + `,"target_environment":` + fixtureTupleJSON() + `,"expected_target_native_session_id":"target-1","fidelity_profile":"maximal_safe","required_dispositions":{"user_message":["exact"]},"forbid_reasons":[],"resource_limits":{"max_objects":10,"max_total_bytes":1000,"max_single_object_bytes":100,"max_events":5,"max_target_resources":5},"extensions":{}}`)
	case OpProject:
		return []byte(`{` + ctx + `,"projection_plan_id":` + quote(fixturePlanDigest) + `,"capture_manifest_id":` + quote(fixtureCaptureDigest) + `,"canonical_session_id":` + quote(fixtureSessionDigest) + `,"source_objects":` + fixtureFreshSinkJSON("canonical_source") + `,"target_sink":` + fixtureFreshSinkJSON("projected_target") + `,"extensions":{}}`)
	case OpReadBack:
		staged := mutateAuthority(t, authority, "target_staged")
		return []byte(`{` + ctx + `,"authority":` + staged + `,"expected_target_native_session_id":"target-1","projection_plan_id":` + quote(fixturePlanDigest) + `,"projected_object_manifest_id":` + quote(fixtureProjectedDigest) + `,"evidence_sink":` + fixtureFreshSinkJSON("read_back_evidence") + `,"extensions":{}}`)
	case OpValidate:
		return []byte(`{` + ctx + `,"mode":"staged","capture_manifest_id":` + quote(fixtureCaptureDigest) + `,"canonical_session_id":` + quote(fixtureSessionDigest) + `,"projection_plan_id":` + quote(fixturePlanDigest) + `,"projected_object_manifest_id":` + quote(fixtureProjectedDigest) + `,"read_back_evidence_manifest_id":` + quote(fixtureReadBackDigest) + `,"expected_target_native_session_id":"target-1","extensions":{}}`)
	case OpResumePlan:
		return []byte(`{` + ctx + `,"authority":` + authority + `,"expected_target_native_session_id":"target-1","projection_plan_id":` + quote(fixturePlanDigest) + `,"target_checkpoint_id":null,"extensions":{}}`)
	case OpDoctor:
		return []byte(`{` + ctx + `,"direction":"source_read","tuple_registry_digest":` + quote(fixtureRegistryDigest) + `,"refresh_requested":false,"extensions":{}}`)
	default:
		t.Fatalf("no request fixture for %q", operation)
		return nil
	}
}

func quote(value string) string {
	return `"` + value + `"`
}

// mutateAuthority rebuilds the authority fixture with a purpose.
func mutateAuthority(t *testing.T, authority, purpose string) string {
	t.Helper()
	var members map[string]json.RawMessage
	if err := json.Unmarshal([]byte(authority), &members); err != nil {
		t.Fatalf("mutateAuthority: %v", err)
	}
	members["purpose"] = json.RawMessage(`"` + purpose + `"`)
	rebuilt, err := json.Marshal(members)
	if err != nil {
		t.Fatalf("mutateAuthority: %v", err)
	}
	return string(rebuilt)
}

// fixtureSuccessBody builds one valid success body per operation
// over the request context bytes.
func fixtureSuccessBody(t *testing.T, operation Operation, contextJSON string, manifestDigest string) []byte {
	t.Helper()
	ctx := `"context":` + contextJSON
	switch operation {
	case OpDiscover:
		return []byte(`{` + ctx + `,"sources":[],"next_cursor":null,"partial":false,"extensions":{}}`)
	case OpInspect:
		return []byte(`{` + ctx + `,"source":{},"source_identity":{},"source_store_generation":"gen-7","environment":` + fixtureTupleJSON() + `,"ambiguities":[],"extensions":{}}`)
	case OpSnapshotProof:
		return []byte(`{` + ctx + `,"proof":{},"provider_quiescence_requested":false,"provider_quiescence_observed":false,"extensions":{}}`)
	case OpCapturePlan:
		return []byte(`{` + ctx + `,"source_identity":{},"source_store_generation":"gen-7","capture_plan_candidate_id":` + quote(fixturePlanDigest) + `,"candidate_count":0,"excluded_classes":[],"capture_plan_digest":` + quote(fixturePlanDigest) + `,"extensions":{}}`)
	case OpCapture:
		return []byte(`{` + ctx + `,"capture_plan_digest":` + quote(fixturePlanDigest) + `,"source_store_generation":"gen-7","capture_result_candidate_id":` + quote(fixtureDigest("result")) + `,"source_raw_object_manifest_candidate_id":` + quote(fixtureDigest("raw-manifest")) + `,"item_count":0,"pre_capture_digest":` + quote(fixtureDigest("pre")) + `,"post_capture_digest":` + quote(fixtureDigest("post")) + `,"extensions":{}}`)
	case OpNormalize:
		return []byte(`{` + ctx + `,"capture_manifest_id":` + quote(fixtureCaptureDigest) + `,"canonical_session_candidate_id":` + quote(fixtureSessionDigest) + `,"canonical_event_candidate_ids":[],"raw_reference_ids":[],"extensions":{}}`)
	case OpProjectionPlan:
		return []byte(`{` + ctx + `,"projection_plan_candidate_id":` + quote(fixturePlanDigest) + `,"required_source_object_ids":[],"predicted_counts":{},"findings":[],"extensions":{}}`)
	case OpProject:
		return []byte(`{` + ctx + `,"projection_plan_id":` + quote(fixturePlanDigest) + `,"projected_object_manifest_candidate_id":` + quote(fixtureProjectedDigest) + `,"created_resource_keys":[],"actual_counts":{},"extensions":{}}`)
	case OpReadBack:
		return []byte(`{` + ctx + `,"observed_target_native_session_id":"target-1","projection_plan_id":` + quote(fixturePlanDigest) + `,"observed_environment":` + fixtureTupleJSON() + `,"parsed_event_count":0,"parsed_head_ids":[],"workspace_binding":{},"structural_digest":` + quote(fixtureStructuralDigest) + `,"read_back_evidence_manifest_candidate_id":` + quote(fixtureReadBackDigest) + `,"extensions":{}}`)
	case OpValidate:
		return []byte(`{` + ctx + `,"mode":"staged","valid":true,"structural_valid":true,"semantic_marker_valid":true,"identity_valid":null,"workspace_binding_valid":null,"resume_surface_valid":null,"findings":[],"evidence_digest":` + quote(fixtureEvidenceDigest) + `,"extensions":{}}`)
	case OpResumePlan:
		return []byte(`{` + ctx + `,"target_native_session_id":"target-1","projection_plan_id":` + quote(fixturePlanDigest) + `,"argv":["ax","open","target-1"],"cwd_relative":".","environment_names":[],"opens_existing_identity":true,"extensions":{}}`)
	case OpDoctor:
		return []byte(`{` + ctx + `,"direction":"source_read","registry_sequence":9,"registry_entry_status":"accepted","findings":[],"healthy":false,"extensions":{}}`)
	default:
		t.Fatalf("no success fixture for %q", operation)
		return nil
	}
}

// requestContextOf extracts the context bytes from a request body.
func requestContextOf(t *testing.T, body []byte) string {
	t.Helper()
	return string(rawMember(t, body, "context"))
}

// TestEveryOperationHasContractVectors drives every registry
// operation through both production entry points with a valid
// body: 14 of 14 request vectors and 11 of 11 success vectors
// (manifest, probe, and doctor successes decode through their
// dedicated decoders, covered by their own tests). An operation
// with no vector fails here rather than passing silently.
func TestEveryOperationHasContractVectors(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	_, context := requestContext(t, manifestDigest)
	_ = context
	drivenRequests := 0
	drivenSuccesses := 0
	for _, name := range Operations() {
		operation := Operation(name)
		request := fixtureRequestBody(t, operation, manifestDigest)
		decoded, err := CheckRequestBody(operation, request)
		if err != nil {
			t.Fatalf("CheckRequestBody(%q): %v", operation, err)
		}
		drivenRequests++
		if operation == OpManifest || operation == OpProbe || operation == OpDoctor {
			continue
		}
		facts := SuccessFacts{Context: decoded, ValidateMode: "staged", ResumeTargetID: "target-1", DoctorDirection: DirectionSourceRead}
		if operation == OpValidate {
			facts.ValidateMode = "staged"
		}
		success := fixtureSuccessBody(t, operation, requestContextOf(t, request), manifestDigest)
		if err := CheckSuccessBody(operation, success, facts); err != nil {
			t.Fatalf("CheckSuccessBody(%q): %v", operation, err)
		}
		drivenSuccesses++
	}
	if drivenRequests != 14 {
		t.Fatalf("request vectors = %d, want 14", drivenRequests)
	}
	if drivenSuccesses != 11 {
		t.Fatalf("success vectors = %d, want 11", drivenSuccesses)
	}
	t.Logf("contract vectors: %d/14 requests, %d/11 successes, 3 dedicated decoders", drivenRequests, drivenSuccesses)
}
