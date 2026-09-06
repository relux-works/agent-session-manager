package sessadapter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// Witnesses for the derived refusal-arm inventory. Each row names
// one derived arm and proves it refuses at the production entry
// with the attributed identity (code, distinguishing detail, rule
// text). Rows are one line each through the factories below; only
// multi-step gates use custom closures. A row that names no
// derived arm fails the reverse check, so a typo here reddens
// rather than passing silently.

// codeOf maps a refusal constructor to its stable code.
func codeOf(ctor string) axerror.Code {
	switch ctor {
	case "failInvalid":
		return "invalid_config"
	case "failProtocol":
		return "session_adapter_protocol_error"
	case "failUnknownOperation":
		return "operation_unknown"
	case "failCapability", "failUnavailable":
		return "capability_unavailable"
	case "failUnsupportedTuple":
		return "unsupported_environment_tuple"
	case "failIntegrity":
		return "integrity_failure"
	default:
		return "invalid_config"
	}
}

// keyOf maps a refusal constructor to its distinguishing detail
// key.
func keyOf(ctor string) string {
	switch ctor {
	case "failInvalid":
		return "field"
	case "failIntegrity":
		return "subject"
	case "failUnknownOperation", "failUnavailable":
		return "operation"
	case "failCapability":
		return "capability"
	case "failUnsupportedTuple":
		return "environment"
	default:
		return "member"
	}
}

// wDrop proves a missing-member arm: dropping the member from a
// fresh positive fixture must refuse naming that member.
func wDrop(ctor, detail, member string, run func([]byte) error, fixture []byte) armWitness {
	return wDropText(ctor, detail, member, "misses a required member", run, fixture)
}

// wDropText proves a missing-member arm whose detail reads
// differently (resource limits agree the subject plural).
func wDropText(ctor, detail, member, text string, run func([]byte) error, fixture []byte) armWitness {
	return armWitness{
		arm:  "ctor|" + ctor + "|" + detail,
		name: "missing " + member,
		prove: func(t *testing.T) {
			t.Helper()
			err := run(dropMember(t, fixture, member))
			requireRefusal(t, err, codeOf(ctor), keyOf(ctor), member, text)
		},
	}
}

// wUnknown proves an unknown-member arm: one unexpected member on
// a fresh positive fixture must refuse naming it.
func wUnknown(ctor, detail string, run func([]byte) error, fixture []byte) armWitness {
	return armWitness{
		arm:  "ctor|" + ctor + "|" + detail,
		name: "unknown member",
		prove: func(t *testing.T) {
			t.Helper()
			err := run(appendMember(t, fixture, "unexpected", `true`))
			requireRefusal(t, err, codeOf(ctor), keyOf(ctor), "unexpected", "unknown member")
		},
	}
}

// wSet proves one value-rule arm with full control over the
// mutation and the attributed identity.
func wSet(ctor, detail, member, value, keyValue, text string, run func([]byte) error, fixture []byte) armWitness {
	return armWitness{
		arm:  "ctor|" + ctor + "|" + detail,
		name: member + "=" + value,
		prove: func(t *testing.T) {
			t.Helper()
			err := run(mutateMember(t, fixture, member, value))
			requireRefusal(t, err, codeOf(ctor), keyOf(ctor), keyValue, text)
		},
	}
}

// wCustom names one arm with a bespoke proof.
func wCustom(arm, name string, prove func(t *testing.T)) armWitness {
	return armWitness{arm: arm, name: name, prove: prove}
}

// witnessEntries bundles the production entries and their fresh
// positive fixtures every factory row drives.
type witnessEntries struct {
	requestFrame func([]byte) error
	requestFix   []byte
	successEnv   func([]byte) error
	successFix   []byte
	failureEnv   func([]byte) error
	failureFix   []byte
	manifest     func([]byte) error
	manifestFix  []byte
	probe        func([]byte) error
	probeFix     []byte
	probeDigest  string
	context      func([]byte) error
	contextFix   []byte
	readAuth     func([]byte) error
	readAuthFix  []byte
	objAuth      func([]byte) error
	objAuthFix   []byte
	selector     func([]byte) error
	selectorFix  []byte
	limits       func([]byte) error
	limitsFix    []byte
	finding      func([]byte) error
	findingFix   []byte
	planItem     func([]byte) error
	planItemFix  []byte
	tuple        func([]byte) error
	tupleFix     []byte
	entry        func([]byte) error
	entryFix     []byte
	capValue     func([]byte) error
	capValueFix  []byte
	capMap       func([]byte) error
	capMapFix    []byte
	opReq        map[Operation]func([]byte) error
	opReqFix     map[Operation][]byte
	opSuc        map[Operation]func([]byte) error
	opSucFix     map[Operation][]byte
	opFacts      map[Operation]SuccessFacts
	doctorRun    func([]byte) error
	doctorFix    []byte
	doctorFacts  SuccessFacts
	doctorCtx    CallContext
	manifestText string
}

func witnessFixtureEntries(t *testing.T) *witnessEntries {
	t.Helper()
	entries := &witnessEntries{
		opReq:    map[Operation]func([]byte) error{},
		opReqFix: map[Operation][]byte{},
		opSuc:    map[Operation]func([]byte) error{},
		opSucFix: map[Operation][]byte{},
		opFacts:  map[Operation]SuccessFacts{},
	}
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	entries.manifestText = ManifestDigest(manifest).String()
	entries.probeDigest = entries.manifestText
	probeFix := []byte(fixtureProbeJSON(entries.probeDigest))
	entries.requestFix = fixtureRequestFrame(t, OpProbe, `{"expected_provider_id":"test-provider","expected_candidate_kind":"builtin","extensions":{}}`)
	entries.requestFrame = func(body []byte) error {
		_, err := DecodeRequestFrame(body)
		return err
	}
	entries.successFix = []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":true,"body":{}}`, ProtocolID, fixtureRequestID))
	entries.successEnv = func(body []byte) error {
		_, err := CheckSuccessEnvelope(body, wantRequest())
		return err
	}
	entries.failureFix = []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"1.1.0","code":"capability_unavailable","message":"the adapter build cannot enumerate this source","exit_code":6,"retryable":false,"details":{}}}`,
		ProtocolID, fixtureRequestID))
	entries.failureEnv = func(body []byte) error {
		_, err := CheckFailureEnvelope(body, wantRequest())
		return err
	}
	entries.manifestFix = []byte(fixtureManifestJSON())
	entries.manifest = func(body []byte) error {
		_, err := DecodeManifest(body)
		return err
	}
	entries.probeFix = probeFix
	entries.probe = func(body []byte) error {
		_, err := DecodeProbe(body)
		return err
	}
	entries.contextFix = []byte(fixtureContextJSON(fixtureDigest("req")))
	entries.context = func(body []byte) error {
		_, err := DecodeCallContext(body)
		return err
	}
	entries.readAuthFix = []byte(fixtureReadAuthorityJSON())
	entries.readAuth = func(body []byte) error {
		_, err := DecodeReadAuthority(body)
		return err
	}
	entries.objAuthFix = []byte(fixtureFreshSinkJSON("capture_plan"))
	entries.objAuth = func(body []byte) error {
		_, err := DecodeObjectAuthority(body)
		return err
	}
	entries.selectorFix = []byte(`{"native_session_id":"sess-1","logical_workspace_id":null,"opaque_source_ref":null}`)
	entries.selector = func(body []byte) error {
		_, err := DecodeSourceSelector(body)
		return err
	}
	entries.limitsFix = []byte(`{"max_objects":10,"max_total_bytes":1000,"max_single_object_bytes":100,"max_events":5,"max_target_resources":5}`)
	entries.limits = func(body []byte) error {
		_, err := DecodeResourceLimits(body)
		return err
	}
	entries.findingFix = []byte(`{"severity":"warning","code":"cap-1","message":"a warning","remediation":null,"extensions":{}}`)
	entries.finding = func(body []byte) error {
		_, err := DecodeFinding(body)
		return err
	}
	entries.planItemFix = []byte(`{"native_item_key":"item-1","class":"durable_payload","byte_count":12,"required":true,"extensions":{}}`)
	entries.planItem = func(body []byte) error {
		_, err := DecodeCapturePlanItem(body)
		return err
	}
	entries.tupleFix = []byte(fixtureTupleJSON())
	entries.tuple = func(body []byte) error {
		_, err := DecodeTuple(body)
		return err
	}
	entries.entryFix = []byte(fixtureEntryJSON(DirectionTargetWrite))
	entries.entry = func(body []byte) error {
		_, err := DecodeTupleEntry(body)
		return err
	}
	entries.capValueFix = []byte(fixtureCapabilityJSON())
	entries.capValue = func(body []byte) error {
		_, err := decodeCapabilityValue(body, "tool_history")
		return err
	}
	entries.capMapFix = rawMember(t, probeFix, "capabilities")
	entries.capMap = func(body []byte) error {
		_, err := decodeCapabilities(body)
		return err
	}
	for _, name := range Operations() {
		operation := Operation(name)
		request := fixtureRequestBody(t, operation, entries.manifestText)
		entries.opReqFix[operation] = request
		entries.opReq[operation] = func(body []byte) error {
			_, err := CheckRequestBody(operation, body)
			return err
		}
		if operation == OpManifest || operation == OpProbe || operation == OpDoctor {
			continue
		}
		decoded, err := CheckRequestBody(operation, request)
		if err != nil {
			t.Fatalf("CheckRequestBody(%q): %v", operation, err)
		}
		facts := SuccessFacts{Context: decoded, ValidateMode: "staged", ResumeTargetID: "target-1", DoctorDirection: DirectionSourceRead}
		entries.opFacts[operation] = facts
		entries.opSucFix[operation] = fixtureSuccessBody(t, operation, requestContextOf(t, request), entries.manifestText)
		entries.opSuc[operation] = func(body []byte) error {
			return CheckSuccessBody(operation, body, facts)
		}
	}
	doctorRequest := fixtureRequestBody(t, OpDoctor, entries.manifestText)
	doctorCtx, err := CheckRequestBody(OpDoctor, doctorRequest)
	if err != nil {
		t.Fatalf("CheckRequestBody(doctor): %v", err)
	}
	entries.doctorCtx = doctorCtx
	entries.doctorFacts = SuccessFacts{Context: doctorCtx, DoctorDirection: DirectionSourceRead}
	entries.doctorFix = fixtureSuccessBody(t, OpDoctor, requestContextOf(t, doctorRequest), entries.manifestText)
	entries.doctorRun = func(body []byte) error {
		_, err := DecodeDoctorResult(body, doctorCtx, DirectionSourceRead)
		return err
	}
	return entries
}

// The group tables below feed declaredWitnesses in
// inventory_test.go.

func envelopeWitnesses(e *witnessEntries) []armWitness {
	return []armWitness{
		wCustom("ctor|failProtocol|request frame is empty", "empty frame", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.requestFrame(nil), "session_adapter_protocol_error", "member", "", "request frame is empty")
		}),
		wCustom("ctor|failProtocol|request frame exceeds the 8 MiB bound", "oversize frame", func(t *testing.T) {
			t.Helper()
			padding := strings.Repeat("a", MaxFrameBytes)
			big := mutateMember(t, e.requestFix, "body", `{"expected_provider_id":"`+padding+`","expected_candidate_kind":"builtin","extensions":{}}`)
			requireRefusal(t, e.requestFrame(big), "session_adapter_protocol_error", "member", "", "exceeds the 8 MiB bound")
		}),
		wCustom("ctor|failProtocol|request envelope carries unknown member", "unknown member", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.requestFrame(appendMember(t, e.requestFix, "unexpected", `true`)), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|request envelope misses a required member", "missing operation", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.requestFrame(dropMember(t, e.requestFix, "operation")), "session_adapter_protocol_error", "member", "operation", "misses a required member")
		}),
		wSet("failProtocol", "request protocol is not the session adapter", "protocol", `"urn:ax:protocol:provider"`, "protocol", "not the session adapter", e.requestFrame, e.requestFix),
		wSet("failProtocol", "request protocol version is not 1.0.0", "protocol_version", `"2.0.0"`, "protocol_version", "not 1.0.0", e.requestFrame, e.requestFix),
		wSet("failProtocol", "request identifier is not a string", "request_id", `42`, "request_id", "not a string", e.requestFrame, e.requestFix),
		wSet("failProtocol", "request identifier is not a UUIDv7", "request_id", `"not-a-uuid"`, "request_id", "not a UUIDv7", e.requestFrame, e.requestFix),
		wSet("failProtocol", "request operation is not a string", "operation", `42`, "operation", "not a string", e.requestFrame, e.requestFix),
		wCustom("ctor|failUnknownOperation|request names an operation outside the closed registry", "unknown operation", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.requestFrame(mutateMember(t, e.requestFix, "operation", `"launch"`)), "operation_unknown", "operation", "launch", "closed registry")
		}),
		wSet("failProtocol", "request deadline is not a timestamp", "deadline", `"yesterday"`, "deadline", "not a timestamp", e.requestFrame, e.requestFix),
		wCustom("ctor|failProtocol|success frame is empty", "empty frame", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.successEnv(nil), "session_adapter_protocol_error", "member", "", "success frame is empty")
		}),
		wCustom("ctor|failProtocol|success frame exceeds the 8 MiB bound", "oversize frame", func(t *testing.T) {
			t.Helper()
			padding := strings.Repeat("b", MaxFrameBytes)
			big := mutateMember(t, e.successFix, "body", `{"pad":"`+padding+`"}`)
			requireRefusal(t, e.successEnv(big), "session_adapter_protocol_error", "member", "", "exceeds the 8 MiB bound")
		}),
		wCustom("ctor|failProtocol|success envelope carries unknown member", "error member", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.successEnv(appendMember(t, e.successFix, "error", `{"code":"x"}`)), "session_adapter_protocol_error", "member", "error", "unknown member")
		}),
		wCustom("ctor|failProtocol|success envelope misses a required member", "missing body", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.successEnv(dropMember(t, e.successFix, "body")), "session_adapter_protocol_error", "member", "body", "misses a required member")
		}),
		wSet("failProtocol", "success protocol does not echo the request", "protocol", `"urn:ax:protocol:provider"`, "protocol", "does not echo", e.successEnv, e.successFix),
		wSet("failProtocol", "success protocol version does not echo the request", "protocol_version", `"9.9.9"`, "protocol_version", "does not echo", e.successEnv, e.successFix),
		wSet("failProtocol", "success request identifier does not echo the request", "request_id", `"0198f4c8-8e50-7f66-8f70-1234567890aa"`, "request_id", "does not echo", e.successEnv, e.successFix),
		wSet("failProtocol", "success operation does not echo the request", "operation", `"probe"`, "operation", "does not echo", e.successEnv, e.successFix),
		wSet("failProtocol", "success envelope does not carry ok=true", "ok", `false`, "ok", "does not carry ok=true", e.successEnv, e.successFix),
		wCustom("ctor|failProtocol|failure frame is empty", "empty frame", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.failureEnv(nil), "session_adapter_protocol_error", "member", "", "failure frame is empty")
		}),
		wCustom("ctor|failProtocol|failure frame exceeds the 8 MiB bound", "oversize frame", func(t *testing.T) {
			t.Helper()
			padding := strings.Repeat("c", MaxFrameBytes)
			big := mutateMember(t, e.failureFix, "error", `{"schema":"urn:ax:schema:error","schema_version":"1.1.0","code":"capability_unavailable","message":"`+padding+`","exit_code":6,"retryable":false,"details":{}}`)
			requireRefusal(t, e.failureEnv(big), "session_adapter_protocol_error", "member", "", "exceeds the 8 MiB bound")
		}),
		wCustom("ctor|failProtocol|failure envelope carries unknown member", "body member", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.failureEnv(appendMember(t, e.failureFix, "body", `{}`)), "session_adapter_protocol_error", "member", "body", "unknown member")
		}),
		wCustom("ctor|failProtocol|failure envelope misses a required member", "missing error", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.failureEnv(dropMember(t, e.failureFix, "error")), "session_adapter_protocol_error", "member", "error", "misses a required member")
		}),
		wSet("failProtocol", "failure protocol does not echo the request", "protocol", `"urn:ax:protocol:provider"`, "protocol", "does not echo", e.failureEnv, e.failureFix),
		wSet("failProtocol", "failure protocol version does not echo the request", "protocol_version", `"9.9.9"`, "protocol_version", "does not echo", e.failureEnv, e.failureFix),
		wSet("failProtocol", "failure request identifier does not echo the request", "request_id", `"0198f4c8-8e50-7f66-8f70-1234567890aa"`, "request_id", "does not echo", e.failureEnv, e.failureFix),
		wSet("failProtocol", "failure operation does not echo the request", "operation", `"probe"`, "operation", "does not echo", e.failureEnv, e.failureFix),
		wSet("failProtocol", "failure envelope does not carry ok=false", "ok", `true`, "ok", "does not carry ok=false", e.failureEnv, e.failureFix),
		wCustom("ctor|failIntegrity|failure error is not a Structured Error 1.1", "wrong error version", func(t *testing.T) {
			t.Helper()
			bad := mutateMember(t, e.failureFix, "error", `{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"capability_unavailable","message":"the adapter build cannot enumerate this source","exit_code":6,"retryable":false,"details":{}}`)
			requireRefusal(t, e.failureEnv(bad), "integrity_failure", "subject", "error", "not a Structured Error 1.1")
		}),
	}
}

func manifestWitnesses(e *witnessEntries) []armWitness {
	witnesses := []armWitness{}
	for _, member := range manifestRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "adapter manifest misses a required member", member, e.manifest, e.manifestFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "adapter manifest carries unknown member", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest schema is not the session adapter manifest", "schema", `"urn:ax:schema:provider-manifest"`, "schema", "not the session adapter manifest", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest version is not 1.0.0", "schema_version", `"2.0.0"`, "schema_version", "not 1.0.0", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest provider identifier is not a provider-id", "provider_id", `"Test"`, "provider_id", "not a provider-id", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest environment identifier is not an environment-id", "environment_id", `"Test!"`, "environment_id", "not an environment-id", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest display name is not a string[1..128]", "display_name", `""`, "display_name", "not a string[1..128]", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest adapter version is not SemVer", "adapter_version", `"1.2"`, "adapter_version", "not SemVer", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest environment version range is not a non-empty string[1..256]", "environment_version_range", `""`, "environment_version_range", "not a non-empty string", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest platforms are not a sorted unique non-empty subset", "platforms", `[]`, "platforms", "not a sorted unique", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest platform is outside linux|macos|windows|wsl2", "platforms", `["plan9"]`, "platforms", "outside linux|macos", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest operations are not the complete ordered fourteen-name registry", "operations", `[]`, "operations", "fourteen-name registry", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest capability names are not the complete ordered fifteen-name registry", "capability_names", `[]`, "capability_names", "fifteen-name registry", e.manifest, e.manifestFix),
		wSet("failProtocol", "adapter manifest extensions are not reverse-DNS keyed", "extensions", `{"x":"y"}`, "extensions", "not reverse-DNS keyed", e.manifest, e.manifestFix),
		wCustom("ctor|failProtocol|adapter manifest is not canonical JSON", "deep extensions", func(t *testing.T) {
			t.Helper()
			deep := `{"com.example.deep":` + nestedObject(300) + `}`
			requireRefusal(t, e.manifest(mutateMember(t, e.manifestFix, "extensions", deep)), "session_adapter_protocol_error", "member", "", "not canonical JSON")
		}),
	)
	return witnesses
}

// nestedObject builds value nesting depth levels deep.
func nestedObject(depth int) string {
	value := `1`
	for index := 0; index < depth; index++ {
		value = `{"a":` + value + `}`
	}
	return value
}

func probeWitnesses(e *witnessEntries) []armWitness {
	witnesses := []armWitness{}
	for _, member := range probeRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "adapter probe misses a required member", member, e.probe, e.probeFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "adapter probe carries unknown member", e.probe, e.probeFix),
		wSet("failProtocol", "adapter probe schema is not the session adapter probe", "schema", `"urn:ax:schema:provider-probe"`, "schema", "not the session adapter probe", e.probe, e.probeFix),
		wSet("failProtocol", "adapter probe version is not 1.0.0", "schema_version", `"2.0.0"`, "schema_version", "not 1.0.0", e.probe, e.probeFix),
		wSet("failProtocol", "adapter probe provider identifier is not a provider-id", "provider_id", `"Test"`, "provider_id", "not a provider-id", e.probe, e.probeFix),
		wSet("failProtocol", "adapter probe manifest digest is not a digest", "adapter_manifest_digest", `"nope"`, "adapter_manifest_digest", "not a digest", e.probe, e.probeFix),
		wSet("failProtocol", "adapter probe adapter version is not SemVer", "adapter_version", `"1.2"`, "adapter_version", "not SemVer", e.probe, e.probeFix),
		wSet("failProtocol", "adapter probe warnings are not sorted unique string[0..2048][0..1024]", "warnings", `["b","a"]`, "warnings", "sorted unique", e.probe, e.probeFix),
		wSet("failProtocol", "adapter probe extensions are not reverse-DNS keyed", "extensions", `{"x":"y"}`, "extensions", "not reverse-DNS keyed", e.probe, e.probeFix),
		wCustom("ctor|failProtocol|adapter capabilities do not carry exactly fifteen entries", "fourteen entries", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.probe(mutateMember(t, e.probeFix, "capabilities", `{}`)), "session_adapter_protocol_error", "member", "capabilities", "exactly fifteen")
		}),
		wCustom("ctor|failProtocol|adapter capabilities omit a registry capability", "renamed capability", func(t *testing.T) {
			t.Helper()
			renamed := renameCapability(t, e.probeFix, "tool_history", "gui_clipboard")
			requireRefusal(t, e.probe(renamed), "session_adapter_protocol_error", "member", "tool_history", "omit a registry capability")
		}),
		wCustom("ctor|failProtocol|adapter capability value carries unknown member", "unexpected value member", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.capValue(appendMember(t, e.capValueFix, "unexpected", `true`)), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|adapter capability value misses a required member", "missing status", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.capValue(dropMember(t, e.capValueFix, "status")), "session_adapter_protocol_error", "member", "status", "misses a required member")
		}),
		wSet("failProtocol", "adapter capability status is outside available|conditional|unsupported|unknown", "status", `"broken"`, "tool_history", "outside available", e.capValue, e.capValueFix),
		wSet("failProtocol", "adapter capability enabled flag is not a boolean", "enabled", `"yes"`, "tool_history", "not a boolean", e.capValue, e.capValueFix),
		wCustom("ctor|failProtocol|adapter capability is enabled without available status", "conditional enabled", func(t *testing.T) {
			t.Helper()
			body := mutateMember(t, e.capValueFix, "status", `"conditional"`)
			requireRefusal(t, e.capValue(body), "session_adapter_protocol_error", "member", "tool_history", "without available status")
		}),
		wSet("failProtocol", "adapter capability evidence is outside the seven-evidence vocabulary", "evidence", `"vibes"`, "tool_history", "seven-evidence vocabulary", e.capValue, e.capValueFix),
		wSet("failProtocol", "adapter capability detail is not a string[0..2048]", "detail", `"`+strings.Repeat("d", 2049)+`"`, "tool_history", "string[0..2048]", e.capValue, e.capValueFix),
		wCustom("ctor|failProtocol|adapter probe provider identifier does not match the requested provider", "provider mismatch", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			facts := ProbeHostFacts{ExpectedProviderID: "other-provider", ExpectedKind: CandidateBuiltin, ManifestDigest: e.probeDigest, Manifest: manifest}
			requireRefusal(t, CheckProbe(probe, facts), "session_adapter_protocol_error", "member", "provider_id", "requested provider")
		}),
		wCustom("ctor|failProtocol|adapter probe provider identifier does not match the verified manifest", "manifest mismatch", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			other := manifest
			other.ProviderID = "other-provider"
			facts := ProbeHostFacts{ExpectedProviderID: probe.ProviderID, ExpectedKind: CandidateBuiltin, ManifestDigest: e.probeDigest, Manifest: other}
			requireRefusal(t, CheckProbe(probe, facts), "session_adapter_protocol_error", "member", "provider_id", "verified manifest")
		}),
		wCustom("ctor|failProtocol|adapter probe manifest digest does not match the host-computed digest", "digest mismatch", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			facts := ProbeHostFacts{ExpectedProviderID: fixtureProviderID, ExpectedKind: CandidateBuiltin, ManifestDigest: fixtureDigest("other"), Manifest: manifest}
			requireRefusal(t, CheckProbe(probe, facts), "session_adapter_protocol_error", "member", "adapter_manifest_digest", "host-computed digest")
		}),
		wCustom("ctor|failProtocol|adapter probe adapter version does not match the verified manifest", "version mismatch", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			probe.AdapterVersion = "9.9.9"
			facts := ProbeHostFacts{ExpectedProviderID: fixtureProviderID, ExpectedKind: CandidateBuiltin, ManifestDigest: e.probeDigest, Manifest: manifest}
			requireRefusal(t, CheckProbe(probe, facts), "session_adapter_protocol_error", "member", "adapter_version", "verified manifest")
		}),
		wCustom("ctor|failUnsupportedTuple|adapter probe reports a tuple outside the manifest environment", "foreign tuple", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			probe.Environment.EnvironmentID = "other.env"
			facts := ProbeHostFacts{ExpectedProviderID: fixtureProviderID, ExpectedKind: CandidateBuiltin, ManifestDigest: e.probeDigest, Manifest: manifest}
			requireRefusal(t, CheckProbe(probe, facts), "unsupported_environment_tuple", "environment", "other.env", "manifest environment")
		}),
		wCustom("ctor|failProtocol|adapter probe tuple adapter version does not match the probe adapter version", "tuple version drift", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			probe.Environment.AdapterVersion = "0.0.1"
			facts := ProbeHostFacts{ExpectedProviderID: fixtureProviderID, ExpectedKind: CandidateBuiltin, ManifestDigest: e.probeDigest, Manifest: manifest}
			requireRefusal(t, CheckProbe(probe, facts), "session_adapter_protocol_error", "member", "environment", "probe adapter version")
		}),
		wCustom("ctor|failInvalid|probe request provider identifier does not match the trusted candidate", "foreign provider", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, CheckProbeRequest("other-provider", CandidateBuiltin, fixtureCandidate()), "invalid_config", "field", "expected_provider_id", "trusted candidate")
		}),
		wCustom("ctor|failInvalid|probe request candidate kind does not match the trusted candidate", "foreign kind", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, CheckProbeRequest(fixtureProviderID, CandidateExternal, fixtureCandidate()), "invalid_config", "field", "expected_candidate_kind", "trusted candidate")
		}),
	)
	return witnesses
}

// renameCapability swaps one capability name in a probe body,
// keeping the entry count at fifteen.
func renameCapability(t *testing.T, body []byte, oldName, newName string) []byte {
	t.Helper()
	marker := `"` + oldName + `":`
	if !strings.Contains(string(body), marker) {
		t.Fatalf("renameCapability: %q not in body", oldName)
	}
	return []byte(strings.Replace(string(body), marker, `"`+newName+`":`, 1))
}

func contextWitnesses(e *witnessEntries) []armWitness {
	witnesses := []armWitness{}
	for _, member := range contextRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "call context misses a required member", member, e.context, e.contextFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "call context carries unknown member", e.context, e.contextFix),
		wSet("failProtocol", "call context operation identifier is not a UUIDv7", "operation_id", `"nope"`, "operation_id", "not a UUIDv7", e.context, e.contextFix),
		wSet("failProtocol", "call context provider identifier is not a provider-id", "provider_id", `"Test"`, "provider_id", "not a provider-id", e.context, e.contextFix),
		wSet("failProtocol", "call context manifest digest is not a digest", "session_adapter_manifest_digest", `"nope"`, "session_adapter_manifest_digest", "not a digest", e.context, e.contextFix),
		wSet("failProtocol", "call context executable digest is not a digest", "executable_sha256", `"nope"`, "executable_sha256", "not a digest", e.context, e.contextFix),
		wSet("failProtocol", "call context request digest is not a digest", "request_digest", `"nope"`, "request_digest", "not a digest", e.context, e.contextFix),
		wSet("failProtocol", "call context extensions are not reverse-DNS keyed", "extensions", `{"x":"y"}`, "extensions", "not reverse-DNS keyed", e.context, e.contextFix),
		wCustom("ctor|failProtocol|call context is not canonical JSON", "deep extensions", func(t *testing.T) {
			t.Helper()
			deep := `{"com.example.deep":` + nestedObject(300) + `}`
			requireRefusal(t, e.context(mutateMember(t, e.contextFix, "extensions", deep)), "session_adapter_protocol_error", "member", "", "not canonical JSON")
		}),
		wCustom("ctor|failProtocol|call request body is not canonical JSON", "garbage body", func(t *testing.T) {
			t.Helper()
			context, err := DecodeCallContext(e.contextFix)
			if err != nil {
				t.Fatalf("DecodeCallContext: %v", err)
			}
			requireRefusal(t, VerifyRequestDigest(context, []byte("{")), "session_adapter_protocol_error", "member", "body", "not canonical JSON")
		}),
		wCustom("ctor|failProtocol|call context request digest does not match the request body", "changed body", func(t *testing.T) {
			t.Helper()
			body := fixtureRequestWithDigest(t, e.manifestText)
			context, err := CheckRequestBody(OpDiscover, body)
			if err != nil {
				t.Fatalf("CheckRequestBody: %v", err)
			}
			requireRefusal(t, VerifyRequestDigest(context, mutateMember(t, body, "limit", `51`)), "session_adapter_protocol_error", "member", "request_digest", "does not match")
		}),
		wCustom("ctor|failProtocol|success context does not echo the request context byte-for-byte", "rebuilt context", func(t *testing.T) {
			t.Helper()
			body := fixtureRequestWithDigest(t, e.manifestText)
			sent, err := CheckRequestBody(OpDiscover, body)
			if err != nil {
				t.Fatalf("CheckRequestBody: %v", err)
			}
			altered := mutateMember(t, rawMember(t, body, "context"), "provider_id", `"other-provider"`)
			_, err = CheckContextEcho(sent, altered)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "context", "byte-for-byte")
		}),
	)
	for _, member := range readAuthorityRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "read authority misses a required member", member, e.readAuth, e.readAuthFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "read authority carries unknown member", e.readAuth, e.readAuthFix),
		wSet("failProtocol", "read authority identifier is not a UUIDv7", "authority_id", `"x"`, "authority_id", "not a UUIDv7", e.readAuth, e.readAuthFix),
		wSet("failProtocol", "read authority purpose is outside source_native|target_staged|target_live", "purpose", `"live"`, "purpose", "outside source_native", e.readAuth, e.readAuthFix),
		wSet("failProtocol", "read authority handle names are not sorted unique string[1..128][1..128]", "root_handle_names", `[]`, "root_handle_names", "sorted unique", e.readAuth, e.readAuthFix),
		wSet("failProtocol", "read authority expiry is not a timestamp", "expires_at", `"never"`, "expires_at", "not a timestamp", e.readAuth, e.readAuthFix),
		wSet("failProtocol", "read authority extensions are not reverse-DNS keyed", "extensions", `{"x":"y"}`, "extensions", "not reverse-DNS keyed", e.readAuth, e.readAuthFix),
	)
	for _, member := range objectAuthorityRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "object authority misses a required member", member, e.objAuth, e.objAuthFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "object authority carries unknown member", e.objAuth, e.objAuthFix),
		wSet("failProtocol", "object authority identifier is not a UUIDv7", "authority_id", `"x"`, "authority_id", "not a UUIDv7", e.objAuth, e.objAuthFix),
		wSet("failProtocol", "object authority purpose is outside the six-purpose vocabulary", "purpose", `"live"`, "purpose", "six-purpose", e.objAuth, e.objAuthFix),
		wSet("failProtocol", "object authority mode is outside read|fresh_sink", "mode", `"append"`, "mode", "outside read", e.objAuth, e.objAuthFix),
		wSet("failProtocol", "object authority object limit is not a uint53", "max_objects", `"lots"`, "max_objects", "not a uint53", e.objAuth, e.objAuthFix),
		wSet("failProtocol", "object authority byte limit is not a uint53", "max_total_bytes", `"lots"`, "max_total_bytes", "not a uint53", e.objAuth, e.objAuthFix),
		wSet("failProtocol", "object authority fresh sink requires both limits above zero", "max_objects", `0`, "mode", "both limits above zero", e.objAuth, e.objAuthFix),
		wSet("failProtocol", "object authority extensions are not reverse-DNS keyed", "extensions", `{"x":"y"}`, "extensions", "not reverse-DNS keyed", e.objAuth, e.objAuthFix),
		wCustom("ctor|failProtocol|object authority fresh sink is not empty", "reused sink", func(t *testing.T) {
			t.Helper()
			authority, err := DecodeObjectAuthority(e.objAuthFix)
			if err != nil {
				t.Fatalf("DecodeObjectAuthority: %v", err)
			}
			requireRefusal(t, CheckFreshSink(authority, false), "session_adapter_protocol_error", "member", "mode", "not empty")
		}),
	)
	for _, member := range sourceSelectorRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "source selector misses a required member", member, e.selector, e.selectorFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "source selector carries unknown member", e.selector, e.selectorFix),
		wSet("failProtocol", "source selector native session identifier is not a string[1..512]", "native_session_id", `""`, "native_session_id", "string[1..512]", e.selector, e.selectorFix),
		wSet("failProtocol", "source selector logical workspace identifier is not a UUIDv7", "logical_workspace_id", `"nope"`, "logical_workspace_id", "not a UUIDv7", e.selector, e.selectorFix),
		wSet("failProtocol", "source selector opaque reference is not a string[1..512]", "opaque_source_ref", `""`, "opaque_source_ref", "string[1..512]", e.selector, e.selectorFix),
		wCustom("ctor|failProtocol|source selector does not name exactly one source", "no source", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.selector([]byte(`{"native_session_id":null,"logical_workspace_id":null,"opaque_source_ref":null}`)), "session_adapter_protocol_error", "member", "native_session_id", "exactly one source")
		}),
	)
	for _, member := range resourceLimitsRequired {
		witnesses = append(witnesses, wDropText("failProtocol", "resource limits miss a required member", member, "miss a required member", e.limits, e.limitsFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "resource limits carry unknown member", e.limits, e.limitsFix),
		wSet("failProtocol", "resource object limit is not a uint53 above zero", "max_objects", `0`, "max_objects", "above zero", e.limits, e.limitsFix),
		wSet("failProtocol", "resource total byte limit is not a uint53 above zero", "max_total_bytes", `0`, "max_total_bytes", "above zero", e.limits, e.limitsFix),
		wSet("failProtocol", "resource single-object byte limit is not a uint53 above zero", "max_single_object_bytes", `0`, "max_single_object_bytes", "above zero", e.limits, e.limitsFix),
		wSet("failProtocol", "resource event limit is not a uint53 in [0..65536]", "max_events", `65537`, "max_events", "[0..65536]", e.limits, e.limitsFix),
		wSet("failProtocol", "resource target limit is not a uint53 in [0..65536]", "max_target_resources", `65537`, "max_target_resources", "[0..65536]", e.limits, e.limitsFix),
		wCustom("ctor|failProtocol|resource object limit exceeds the total byte limit", "objects above total", func(t *testing.T) {
			t.Helper()
			body := mutateMember(t, e.limitsFix, "max_objects", `101`)
			body = mutateMember(t, body, "max_total_bytes", `100`)
			body = mutateMember(t, body, "max_single_object_bytes", `100`)
			requireRefusal(t, e.limits(body), "session_adapter_protocol_error", "member", "max_objects", "exceeds the total")
		}),
		wCustom("ctor|failProtocol|resource single-object byte limit exceeds the total byte limit", "single above total", func(t *testing.T) {
			t.Helper()
			body := mutateMember(t, e.limitsFix, "max_single_object_bytes", `1001`)
			requireRefusal(t, e.limits(body), "session_adapter_protocol_error", "member", "max_single_object_bytes", "exceeds the total")
		}),
	)
	for _, member := range findingRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "adapter finding misses a required member", member, e.finding, e.findingFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "adapter finding carries unknown member", e.finding, e.findingFix),
		wSet("failProtocol", "adapter finding severity is outside info|warning|error", "severity", `"fatal"`, "severity", "outside info", e.finding, e.findingFix),
		wSet("failProtocol", "adapter finding code is not a string[1..128]", "code", `""`, "code", "string[1..128]", e.finding, e.findingFix),
		wSet("failProtocol", "adapter finding message is not a string[1..4096]", "message", `""`, "message", "string[1..4096]", e.finding, e.findingFix),
		wSet("failProtocol", "adapter finding remediation is not a string[1..4096]", "remediation", `""`, "remediation", "string[1..4096]", e.finding, e.findingFix),
		wSet("failProtocol", "adapter finding extensions are not reverse-DNS keyed", "extensions", `{"x":"y"}`, "extensions", "not reverse-DNS keyed", e.finding, e.findingFix),
		wCustom("ctor|failProtocol|adapter findings are not an array", "object findings", func(t *testing.T) {
			t.Helper()
			_, err := DecodeFindings([]byte(`{}`), 4096)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "findings", "not an array")
		}),
		wCustom("ctor|failProtocol|adapter findings exceed the count bound", "over bound", func(t *testing.T) {
			t.Helper()
			_, err := DecodeFindings([]byte(`[`+strings.Repeat(`{"severity":"info","code":"c","message":"m","remediation":null,"extensions":{}},`, 3)+`{"severity":"info","code":"c","message":"m","remediation":null,"extensions":{}}]`), 3)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "findings", "count bound")
		}),
	)
	for _, member := range capturePlanItemRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "capture plan item misses a required member", member, e.planItem, e.planItemFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "capture plan item carries unknown member", e.planItem, e.planItemFix),
		wSet("failProtocol", "capture plan item key is not a string[1..512]", "native_item_key", `""`, "native_item_key", "string[1..512]", e.planItem, e.planItemFix),
		wSet("failProtocol", "capture plan item class is outside the nine-class vocabulary", "class", `"archive_only"`, "class", "nine-class", e.planItem, e.planItemFix),
		wSet("failProtocol", "capture plan item byte count is not a uint53", "byte_count", `"lots"`, "byte_count", "not a uint53", e.planItem, e.planItemFix),
		wSet("failProtocol", "capture plan item required flag is not a boolean", "required", `"yes"`, "required", "not a boolean", e.planItem, e.planItemFix),
		wSet("failProtocol", "capture plan item extensions are not reverse-DNS keyed", "extensions", `{"x":"y"}`, "extensions", "not reverse-DNS keyed", e.planItem, e.planItemFix),
	)
	for _, member := range tupleRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "environment tuple misses a required member", member, e.tuple, e.tupleFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "environment tuple carries unknown member", e.tuple, e.tupleFix),
		wSet("failProtocol", "environment tuple environment identifier is not an environment-id", "environment_id", `"Test!"`, "environment_id", "not an environment-id", e.tuple, e.tupleFix),
		wSet("failProtocol", "environment tuple environment version is not a string[1..128]", "environment_version", `""`, "environment_version", "string[1..128]", e.tuple, e.tupleFix),
		wSet("failProtocol", "environment tuple platform is not a string", "platform", `42`, "platform", "not a string", e.tuple, e.tupleFix),
		wSet("failProtocol", "environment tuple platform is outside linux|macos|windows|wsl2", "platform", `"darwin"`, "platform", "outside linux", e.tuple, e.tupleFix),
		wSet("failProtocol", "environment tuple architecture is outside amd64|arm64", "architecture", `"x86"`, "architecture", "outside amd64", e.tuple, e.tupleFix),
		wSet("failProtocol", "environment tuple store fingerprint is not a digest", "store_schema_fingerprint", `"abc"`, "store_schema_fingerprint", "not a digest", e.tuple, e.tupleFix),
		wSet("failProtocol", "environment tuple adapter version is not SemVer", "adapter_version", `"1.2"`, "adapter_version", "not SemVer", e.tuple, e.tupleFix),
		wCustom("ctor|failProtocol|environment tuple carries unknown member", "executable provenance", func(t *testing.T) {
			t.Helper()
			body := appendMember(t, e.tupleFix, "executable_sha256", `"`+fixtureExecutableDigest+`"`)
			requireRefusal(t, e.tuple(body), "session_adapter_protocol_error", "member", "executable_sha256", "unknown member")
		}),
	)
	return witnesses
}

func tupleWitnesses(e *witnessEntries) []armWitness {
	witnesses := []armWitness{}
	for _, member := range tupleKeyRequired {
		witnesses = append(witnesses, wCustom("ctor|failProtocol|tuple registry key misses a required member", "missing "+member, func(t *testing.T) {
			t.Helper()
			body := dropNested(t, e.entryFix, "key", member)
			requireRefusal(t, e.entry(body), "session_adapter_protocol_error", "member", member, "misses a required member")
		}))
	}
	witnesses = append(witnesses,
		wCustom("ctor|failProtocol|tuple registry key carries unknown member", "unknown key member", func(t *testing.T) {
			t.Helper()
			body := setNested(t, e.entryFix, "key", "unexpected", `true`)
			requireRefusal(t, e.entry(body), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|tuple registry key direction is outside source_read|target_write", "bad direction", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(setNested(t, e.entryFix, "key", "direction", `"sideways"`)), "session_adapter_protocol_error", "member", "direction", "outside source_read")
		}),
		wCustom("ctor|failProtocol|tuple registry key provider identifier is not a provider-id", "bad provider", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(setNested(t, e.entryFix, "key", "provider_id", `"Test"`)), "session_adapter_protocol_error", "member", "provider_id", "not a provider-id")
		}),
		wCustom("ctor|failProtocol|tuple registry key candidate kind is outside builtin|external", "bad kind", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(setNested(t, e.entryFix, "key", "candidate_kind", `"sideways"`)), "session_adapter_protocol_error", "member", "candidate_kind", "outside builtin")
		}),
		wCustom("ctor|failProtocol|tuple registry key executable digest is not a digest", "bad executable", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(setNested(t, e.entryFix, "key", "executable_sha256", `"nope"`)), "session_adapter_protocol_error", "member", "executable_sha256", "not a digest")
		}),
		wCustom("ctor|failProtocol|tuple registry key provider manifest digest is not a digest", "bad provider digest", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(setNested(t, e.entryFix, "key", "provider_manifest_digest", `"nope"`)), "session_adapter_protocol_error", "member", "provider_manifest_digest", "not a digest")
		}),
		wCustom("ctor|failProtocol|tuple registry key adapter manifest digest is not a digest", "bad adapter digest", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(setNested(t, e.entryFix, "key", "session_adapter_manifest_digest", `"nope"`)), "session_adapter_protocol_error", "member", "session_adapter_manifest_digest", "not a digest")
		}),
	)
	for _, member := range tupleEntryRequired {
		witnesses = append(witnesses, wDrop("failProtocol", "tuple registry entry misses a required member", member, e.entry, e.entryFix))
	}
	witnesses = append(witnesses,
		wUnknown("failProtocol", "tuple registry entry carries unknown member", e.entry, e.entryFix),
		wSet("failProtocol", "tuple entry sequence is not a uint53 above zero", "entry_sequence", `0`, "entry_sequence", "above zero", e.entry, e.entryFix),
		wSet("failProtocol", "tuple entry status is outside accepted|revoked", "status", `"pending"`, "status", "accepted|revoked", e.entry, e.entryFix),
		wCustom("ctor|failProtocol|tuple accepted entry carries revocation members", "accepted with reason", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "revocation_reason", `"superseded"`)), "session_adapter_protocol_error", "member", "status", "carries revocation members")
		}),
		wCustom("ctor|failProtocol|tuple revoked entry misses a revocation member", "revoked bare", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "status", `"revoked"`)), "session_adapter_protocol_error", "member", "status", "misses a revocation member")
		}),
		wCustom("ctor|failProtocol|tuple entry revocation reason is not a string[1..4096]", "empty reason", func(t *testing.T) {
			t.Helper()
			revoked := mutateMember(t, e.entryFix, "status", `"revoked"`)
			revoked = mutateMember(t, revoked, "revoked_at", `"2026-06-02T00:00:00.000Z"`)
			bad := mutateMember(t, revoked, "revocation_reason", `""`)
			requireRefusal(t, e.entry(bad), "session_adapter_protocol_error", "member", "revocation_reason", "string[1..4096]")
		}),
		wCustom("ctor|failProtocol|tuple entry revocation time is not a timestamp", "bad revoked_at", func(t *testing.T) {
			t.Helper()
			revoked := mutateMember(t, e.entryFix, "status", `"revoked"`)
			revoked = mutateMember(t, revoked, "revocation_reason", `"superseded"`)
			bad := mutateMember(t, revoked, "revoked_at", `"never"`)
			requireRefusal(t, e.entry(bad), "session_adapter_protocol_error", "member", "revoked_at", "not a timestamp")
			impossible := mutateMember(t, revoked, "revoked_at", `"2026-13-01T00:00:00.000Z"`)
			requireRefusal(t, e.entry(impossible), "session_adapter_protocol_error", "member", "revoked_at", "not a timestamp")
		}),
		wCustom("ctor|failProtocol|tuple entry valid_from is not a timestamp", "bad valid_from", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "valid_from", `"never"`)), "session_adapter_protocol_error", "member", "valid_from", "not a timestamp")
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "valid_from", `"2026-13-01T00:00:00.000Z"`)), "session_adapter_protocol_error", "member", "valid_from", "not a timestamp")
		}),
		wCustom("ctor|failProtocol|tuple entry valid_until is not a timestamp", "bad valid_until", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "valid_until", `"never"`)), "session_adapter_protocol_error", "member", "valid_until", "not a timestamp")
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "valid_until", `"2026-13-01T00:00:00.000Z"`)), "session_adapter_protocol_error", "member", "valid_until", "not a timestamp")
		}),
		wCustom("ctor|failProtocol|tuple entry validity interval does not order valid_from before valid_until", "inverted interval", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "valid_from", `"2028-01-01T00:00:00.000Z"`)), "session_adapter_protocol_error", "member", "valid_until", "does not order")
		}),
		wSet("failProtocol", "tuple registry entry extensions are not reverse-DNS keyed", "extensions", `{"x":"y"}`, "extensions", "not reverse-DNS keyed", e.entry, e.entryFix),
		wSet("failProtocol", "tuple strategies are not an array", "strategies", `"nope"`, "strategies", "not an array", e.entry, e.entryFix),
		wSet("failProtocol", "tuple strategies are empty", "strategies", `[]`, "strategies", "are empty", e.entry, e.entryFix),
		wSet("failProtocol", "tuple strategy is outside the five-strategy vocabulary", "strategies", `["teleport"]`, "strategies", "five-strategy vocabulary", e.entry, e.entryFix),
		wSet("failProtocol", "tuple strategies are not sorted unique", "strategies", `["target_native_writer","continuation_context"]`, "strategies", "not sorted unique", e.entry, e.entryFix),
		wSet("failProtocol", "tuple fixture result is not pass", "fixture_evidence", `{"suite_revision":"rev-9","suite_digest":"`+fixtureSuiteDigest+`","result":"fail","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"`+fixtureEvidenceDigest+`","fixture_count":12}`, "result", "not pass", e.entry, e.entryFix),
		wSet("failProtocol", "tuple fixture count is not a uint53 above zero", "fixture_evidence", `{"suite_revision":"rev-9","suite_digest":"`+fixtureSuiteDigest+`","result":"pass","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"`+fixtureEvidenceDigest+`","fixture_count":0}`, "fixture_count", "above zero", e.entry, e.entryFix),
		wSet("failProtocol", "tuple fixture suite revision is not a string[1..128]", "fixture_evidence", `{"suite_revision":"","suite_digest":"`+fixtureSuiteDigest+`","result":"pass","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"`+fixtureEvidenceDigest+`","fixture_count":1}`, "suite_revision", "string[1..128]", e.entry, e.entryFix),
		wSet("failProtocol", "tuple fixture suite digest is not a digest", "fixture_evidence", `{"suite_revision":"rev-9","suite_digest":"nope","result":"pass","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"`+fixtureEvidenceDigest+`","fixture_count":1}`, "suite_digest", "not a digest", e.entry, e.entryFix),
		wSet("failProtocol", "tuple fixture evidence digest is not a digest", "fixture_evidence", `{"suite_revision":"rev-9","suite_digest":"`+fixtureSuiteDigest+`","result":"pass","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"nope","fixture_count":1}`, "evidence_digest", "not a digest", e.entry, e.entryFix),
		wCustom("ctor|failProtocol|tuple fixture execution time is not a timestamp", "bad fixture time", func(t *testing.T) {
			t.Helper()
			bad := `{"suite_revision":"rev-9","suite_digest":"` + fixtureSuiteDigest + `","result":"pass","executed_at":"never","evidence_digest":"` + fixtureEvidenceDigest + `","fixture_count":1}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "fixture_evidence", bad)), "session_adapter_protocol_error", "member", "executed_at", "not a timestamp")
			impossible := `{"suite_revision":"rev-9","suite_digest":"` + fixtureSuiteDigest + `","result":"pass","executed_at":"2026-13-01T00:00:00.000Z","evidence_digest":"` + fixtureEvidenceDigest + `","fixture_count":1}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "fixture_evidence", impossible)), "session_adapter_protocol_error", "member", "executed_at", "not a timestamp")
		}),
		wCustom("ctor|failProtocol|tuple fixture evidence carries unknown member", "unknown fixture member", func(t *testing.T) {
			t.Helper()
			bad := `{"suite_revision":"rev-9","suite_digest":"` + fixtureSuiteDigest + `","result":"pass","executed_at":"` + fixtureExecutedAt + `","evidence_digest":"` + fixtureEvidenceDigest + `","fixture_count":1,"unexpected":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "fixture_evidence", bad)), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|tuple fixture evidence misses a required member", "missing fixture count", func(t *testing.T) {
			t.Helper()
			bad := `{"suite_revision":"rev-9","suite_digest":"` + fixtureSuiteDigest + `","result":"pass","executed_at":"` + fixtureExecutedAt + `","evidence_digest":"` + fixtureEvidenceDigest + `"}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "fixture_evidence", bad)), "session_adapter_protocol_error", "member", "fixture_count", "misses a required member")
		}),
		wSet("failProtocol", "tuple resume smoke result is not pass", "resume_smoke_evidence", `{"result":"fail","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"`+fixtureSmokeDigest+`","native_cli_family":"claude","bounded_continuation_turn_passed":true}`, "result", "not pass", e.entry, e.entryFix),
		wSet("failProtocol", "tuple resume smoke continuation turn did not pass", "resume_smoke_evidence", `{"result":"pass","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"`+fixtureSmokeDigest+`","native_cli_family":"claude","bounded_continuation_turn_passed":false}`, "bounded_continuation_turn_passed", "did not pass", e.entry, e.entryFix),
		wSet("failProtocol", "tuple resume smoke CLI family is not a string[1..128]", "resume_smoke_evidence", `{"result":"pass","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"`+fixtureSmokeDigest+`","native_cli_family":"","bounded_continuation_turn_passed":true}`, "native_cli_family", "string[1..128]", e.entry, e.entryFix),
		wSet("failProtocol", "tuple resume smoke evidence digest is not a digest", "resume_smoke_evidence", `{"result":"pass","executed_at":"`+fixtureExecutedAt+`","evidence_digest":"nope","native_cli_family":"claude","bounded_continuation_turn_passed":true}`, "evidence_digest", "not a digest", e.entry, e.entryFix),
		wCustom("ctor|failProtocol|tuple resume smoke execution time is not a timestamp", "bad smoke time", func(t *testing.T) {
			t.Helper()
			bad := `{"result":"pass","executed_at":"never","evidence_digest":"` + fixtureSmokeDigest + `","native_cli_family":"claude","bounded_continuation_turn_passed":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "resume_smoke_evidence", bad)), "session_adapter_protocol_error", "member", "executed_at", "not a timestamp")
			impossible := `{"result":"pass","executed_at":"2026-13-01T00:00:00.000Z","evidence_digest":"` + fixtureSmokeDigest + `","native_cli_family":"claude","bounded_continuation_turn_passed":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "resume_smoke_evidence", impossible)), "session_adapter_protocol_error", "member", "executed_at", "not a timestamp")
		}),
		wCustom("ctor|failProtocol|tuple resume smoke carries unknown member", "unknown smoke member", func(t *testing.T) {
			t.Helper()
			bad := `{"result":"pass","executed_at":"` + fixtureExecutedAt + `","evidence_digest":"` + fixtureSmokeDigest + `","native_cli_family":"claude","bounded_continuation_turn_passed":true,"unexpected":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "resume_smoke_evidence", bad)), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|tuple resume smoke misses a required member", "missing smoke family", func(t *testing.T) {
			t.Helper()
			bad := `{"result":"pass","executed_at":"` + fixtureExecutedAt + `","evidence_digest":"` + fixtureSmokeDigest + `","bounded_continuation_turn_passed":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "resume_smoke_evidence", bad)), "session_adapter_protocol_error", "member", "native_cli_family", "misses a required member")
		}),
		wSet("failProtocol", "tuple fidelity limits are not an array", "known_fidelity_limits", `"nope"`, "known_fidelity_limits", "not an array", e.entry, e.entryFix),
		wCustom("ctor|failProtocol|tuple fidelity limits exceed 1024 rows", "1025 rows", func(t *testing.T) {
			t.Helper()
			rows := strings.Repeat(`{"code":"c","affected_class":"k","maximum_disposition":"exact","detail":"d"},`, 1025)
			rows = `[` + strings.TrimSuffix(rows, ",") + `]`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", rows)), "session_adapter_protocol_error", "member", "known_fidelity_limits", "exceed 1024")
		}),
		wCustom("ctor|failProtocol|tuple fidelity limit carries unknown member", "unknown limit member", func(t *testing.T) {
			t.Helper()
			row := `{"code":"c","affected_class":"k","maximum_disposition":"exact","detail":"d","unexpected":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[`+row+`]`)), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|tuple fidelity limit misses a required member", "missing limit code", func(t *testing.T) {
			t.Helper()
			row := `{"affected_class":"k","maximum_disposition":"exact","detail":"d"}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[`+row+`]`)), "session_adapter_protocol_error", "member", "code", "misses a required member")
		}),
		wCustom("ctor|failProtocol|tuple fidelity limit code is not a string[1..128]", "empty limit code", func(t *testing.T) {
			t.Helper()
			row := `{"code":"","affected_class":"k","maximum_disposition":"exact","detail":"d"}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[`+row+`]`)), "session_adapter_protocol_error", "member", "code", "string[1..128]")
		}),
		wCustom("ctor|failProtocol|tuple fidelity limit class is not a string[1..128]", "empty limit class", func(t *testing.T) {
			t.Helper()
			row := `{"code":"c","affected_class":"","maximum_disposition":"exact","detail":"d"}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[`+row+`]`)), "session_adapter_protocol_error", "member", "affected_class", "string[1..128]")
		}),
		wCustom("ctor|failProtocol|tuple fidelity limit disposition is outside the seven-disposition vocabulary", "archive disposition", func(t *testing.T) {
			t.Helper()
			row := `{"code":"c","affected_class":"k","maximum_disposition":"archive_only","detail":"d"}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[`+row+`]`)), "session_adapter_protocol_error", "member", "maximum_disposition", "seven-disposition")
		}),
		wCustom("ctor|failProtocol|tuple fidelity limit detail is not a string[1..4096]", "empty limit detail", func(t *testing.T) {
			t.Helper()
			row := `{"code":"c","affected_class":"k","maximum_disposition":"exact","detail":""}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[`+row+`]`)), "session_adapter_protocol_error", "member", "detail", "string[1..4096]")
		}),
		wCustom("ctor|failProtocol|tuple fidelity limits repeat a code/class pair", "duplicate pair", func(t *testing.T) {
			t.Helper()
			row := `{"code":"c","affected_class":"k","maximum_disposition":"exact","detail":"d"}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[`+row+`,`+row+`]`)), "session_adapter_protocol_error", "member", "known_fidelity_limits", "repeat a code/class pair")
		}),
		wCustom("ctor|failProtocol|tuple fidelity limits are not sorted unique by code/class", "unsorted pairs", func(t *testing.T) {
			t.Helper()
			first := `{"code":"b","affected_class":"k","maximum_disposition":"exact","detail":"d"}`
			second := `{"code":"a","affected_class":"k","maximum_disposition":"exact","detail":"d"}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[`+first+`,`+second+`]`)), "session_adapter_protocol_error", "member", "known_fidelity_limits", "sorted unique by code/class")
		}),
		wSet("failProtocol", "tuple contracts are not an array", "contracts", `"nope"`, "contracts", "not an array", e.entry, e.entryFix),
		wSet("failProtocol", "tuple contracts are not 1..64 rows", "contracts", `[]`, "contracts", "1..64 rows", e.entry, e.entryFix),
		wCustom("ctor|failProtocol|tuple contract row is not exactly contract_id and versions", "three-member row", func(t *testing.T) {
			t.Helper()
			row := `{"contract_id":"urn:ax:protocol:session-adapter","versions":["1.0.0"],"unexpected":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+row+`]`)), "session_adapter_protocol_error", "member", "contracts", "exactly contract_id and versions")
		}),
		wCustom("ctor|failProtocol|tuple contract row misses contract_id", "missing contract_id", func(t *testing.T) {
			t.Helper()
			row := `{"versions":["1.0.0"],"unexpected":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+row+`]`)), "session_adapter_protocol_error", "member", "contracts", "misses contract_id")
		}),
		wCustom("ctor|failProtocol|tuple contract row misses versions", "missing versions", func(t *testing.T) {
			t.Helper()
			row := `{"contract_id":"urn:ax:protocol:session-adapter","unexpected":true}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+row+`]`)), "session_adapter_protocol_error", "member", "contracts", "misses versions")
		}),
		wCustom("ctor|failProtocol|tuple contract identifier is not a string[1..256]", "empty contract id", func(t *testing.T) {
			t.Helper()
			row := `{"contract_id":"","versions":["1.0.0"]}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+row+`]`)), "session_adapter_protocol_error", "member", "contract_id", "string[1..256]")
		}),
		wCustom("ctor|failProtocol|tuple contract versions are not 1..32 entries", "empty versions", func(t *testing.T) {
			t.Helper()
			row := `{"contract_id":"urn:ax:protocol:session-adapter","versions":[]}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+row+`]`)), "session_adapter_protocol_error", "member", "versions", "1..32 entries")
		}),
		wCustom("ctor|failProtocol|tuple contract version is not SemVer", "bad contract version", func(t *testing.T) {
			t.Helper()
			row := `{"contract_id":"urn:ax:protocol:session-adapter","versions":["1.0"]}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+row+`]`)), "session_adapter_protocol_error", "member", "versions", "not SemVer")
		}),
		wCustom("ctor|failProtocol|tuple contract versions are not sorted unique", "unsorted versions", func(t *testing.T) {
			t.Helper()
			row := `{"contract_id":"urn:ax:protocol:session-adapter","versions":["2.0.0","1.0.0"]}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+row+`]`)), "session_adapter_protocol_error", "member", "versions", "not sorted unique")
		}),
		wCustom("ctor|failProtocol|tuple contracts repeat a contract identifier", "duplicate contract", func(t *testing.T) {
			t.Helper()
			row := `{"contract_id":"urn:ax:protocol:session-adapter","versions":["1.0.0"]}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+row+`,`+row+`]`)), "session_adapter_protocol_error", "member", "contracts", "repeat a contract identifier")
		}),
		wCustom("ctor|failProtocol|tuple contracts are not sorted unique by contract identifier", "unsorted contracts", func(t *testing.T) {
			t.Helper()
			first := `{"contract_id":"urn:b","versions":["1.0.0"]}`
			second := `{"contract_id":"urn:a","versions":["1.0.0"]}`
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[`+first+`,`+second+`]`)), "session_adapter_protocol_error", "member", "contracts", "sorted unique by contract identifier")
		}),
	)
	return witnesses
}

func operationWitnesses(e *witnessEntries) []armWitness {
	req := func(operation Operation) (func([]byte) error, []byte) {
		return e.opReq[operation], e.opReqFix[operation]
	}
	suc := func(operation Operation) (func([]byte) error, []byte) {
		return e.opSuc[operation], e.opSucFix[operation]
	}
	witnesses := []armWitness{
		wCustom("ctor|failUnknownOperation|request body names an operation outside the closed registry", "unknown op body", func(t *testing.T) {
			t.Helper()
			_, err := CheckRequestBody(Operation("launch"), e.opReqFix[OpDiscover])
			requireRefusal(t, err, "operation_unknown", "operation", "launch", "closed registry")
		}),
		wCustom("ctor|failUnknownOperation|success body names an operation outside the closed registry", "unknown op success", func(t *testing.T) {
			t.Helper()
			err := CheckSuccessBody(Operation("launch"), e.opSucFix[OpDiscover], e.opFacts[OpDiscover])
			requireRefusal(t, err, "operation_unknown", "operation", "launch", "closed registry")
		}),
		wCustom("ctor|failUnknownOperation|dispatch names an operation outside the closed registry", "dispatch", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, RefuseUnknownOperation("launch"), "operation_unknown", "operation", "launch", "closed registry")
		}),
		wCustom("ctor|failUnavailable|adapter does not implement the registry operation", "unavailable", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, RefuseUnavailableOperation(OpDiscover), "capability_unavailable", "operation", "discover", "does not implement")
		}),
		wCustom("ctor|failProtocol|request body carries unknown member", "unknown request member", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpDiscover)
			requireRefusal(t, run(appendMember(t, fix, "unexpected", `true`)), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|request body misses a required member", "missing request member", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpDiscover)
			requireRefusal(t, run(dropMember(t, fix, "limit")), "session_adapter_protocol_error", "member", "limit", "misses a required member")
		}),
		wCustom("ctor|failProtocol|request body extensions are not reverse-DNS keyed", "bad request extensions", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpDiscover)
			requireRefusal(t, run(mutateMember(t, fix, "extensions", `{"x":"y"}`)), "session_adapter_protocol_error", "member", "extensions", "not reverse-DNS keyed")
		}),
		wCustom("ctor|failProtocol|success body carries unknown member", "unknown success member", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpDiscover)
			requireRefusal(t, run(appendMember(t, fix, "unexpected", `true`)), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|success body misses a required member", "missing success member", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpDiscover)
			requireRefusal(t, run(dropMember(t, fix, "partial")), "session_adapter_protocol_error", "member", "partial", "misses a required member")
		}),
		wCustom("ctor|failProtocol|success body extensions are not reverse-DNS keyed", "bad success extensions", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpDiscover)
			requireRefusal(t, run(mutateMember(t, fix, "extensions", `{"x":"y"}`)), "session_adapter_protocol_error", "member", "extensions", "not reverse-DNS keyed")
		}),
		wCustom("ctor|failProtocol|probe request provider identifier is not a provider-id", "bad expected provider", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProbe)
			requireRefusal(t, run(mutateMember(t, fix, "expected_provider_id", `"Test"`)), "session_adapter_protocol_error", "member", "expected_provider_id", "not a provider-id")
		}),
		wCustom("ctor|failProtocol|probe request candidate kind is outside builtin|external", "bad expected kind", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProbe)
			requireRefusal(t, run(mutateMember(t, fix, "expected_candidate_kind", `"sideways"`)), "session_adapter_protocol_error", "member", "expected_candidate_kind", "outside builtin")
		}),
		wCustom("ctor|failProtocol|probe request extensions are not reverse-DNS keyed", "bad probe extensions", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProbe)
			requireRefusal(t, run(mutateMember(t, fix, "extensions", `{"x":"y"}`)), "session_adapter_protocol_error", "member", "extensions", "not reverse-DNS keyed")
		}),
		wCustom("ctor|failProtocol|discover workspace filter is not a UUIDv7", "bad filter", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpDiscover)
			requireRefusal(t, run(mutateMember(t, fix, "workspace_filter", `"nope"`)), "session_adapter_protocol_error", "member", "workspace_filter", "not a UUIDv7")
		}),
		wCustom("ctor|failProtocol|operation body count member is outside its bound", "zero limit", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpDiscover)
			requireRefusal(t, run(mutateMember(t, fix, "limit", `0`)), "session_adapter_protocol_error", "member", "limit", "outside its bound")
		}),
		wCustom("ctor|failProtocol|operation body string member is outside its bound", "empty generation", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpSnapshotProof)
			requireRefusal(t, run(mutateMember(t, fix, "expected_source_store_generation", `""`)), "session_adapter_protocol_error", "member", "expected_source_store_generation", "outside its bound")
		}),
		wCustom("ctor|failProtocol|operation body flag member is not a boolean", "non-bool quiescence", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpSnapshotProof)
			requireRefusal(t, run(mutateMember(t, fix, "allow_provider_quiescence", `"yes"`)), "session_adapter_protocol_error", "member", "allow_provider_quiescence", "not a boolean")
		}),
		wCustom("ctor|failProtocol|operation body digest member is not a digest", "bad plan digest", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpCapture)
			requireRefusal(t, run(mutateMember(t, fix, "capture_plan_digest", `"nope"`)), "session_adapter_protocol_error", "member", "capture_plan_digest", "not a digest")
		}),
		wCustom("ctor|failProtocol|operation body digest value is not a digest", "bad candidate value", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpCapturePlan)
			requireRefusal(t, run(mutateMember(t, fix, "capture_plan_candidate_id", `"nope"`)), "session_adapter_protocol_error", "member", "capture_plan_candidate_id", "not a digest")
		}),
		wCustom("ctor|failProtocol|conduit|requireNestedObject|operation body nested object ", "null boundary", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpCapturePlan)
			requireRefusal(t, run(mutateMember(t, fix, "capture_boundary", `null`)), "session_adapter_protocol_error", "member", "capture_boundary", "nested object")
		}),
		wCustom("ctor|failProtocol|discover sources are not an array", "object sources", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpDiscover)
			requireRefusal(t, run(mutateMember(t, fix, "sources", `{}`)), "session_adapter_protocol_error", "member", "sources", "not an array")
		}),
		wCustom("ctor|failProtocol|discover sources exceed 65536 entries", "too many sources", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpDiscover)
			sources := `[` + strings.TrimSuffix(strings.Repeat(`{},`, 65537), ",") + `]`
			requireRefusal(t, run(mutateMember(t, fix, "sources", sources)), "session_adapter_protocol_error", "member", "sources", "exceed 65536")
		}),
		wCustom("ctor|failProtocol|discover partial flag disagrees with the cursor", "partial drift", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpDiscover)
			requireRefusal(t, run(mutateMember(t, fix, "partial", `true`)), "session_adapter_protocol_error", "member", "partial", "disagrees with the cursor")
		}),
		wCustom("ctor|failProtocol|capture plan candidate identifier does not equal the plan digest", "digest drift", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpCapturePlan)
			requireRefusal(t, run(mutateMember(t, fix, "capture_plan_digest", quote(fixtureDigest("other")))), "session_adapter_protocol_error", "member", "capture_plan_digest", "does not equal the plan digest")
		}),
		wCustom("ctor|failProtocol|capture excluded classes are not an array", "object excluded", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpCapturePlan)
			requireRefusal(t, run(mutateMember(t, fix, "excluded_classes", `"nope"`)), "session_adapter_protocol_error", "member", "excluded_classes", "not an array")
		}),
		wCustom("ctor|failProtocol|capture excluded classes exceed 9 entries", "ten classes", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpCapturePlan)
			classes := `["credential","derived_cache_optional","durable_index_required","durable_payload","durable_sidecar","machine_auth","runtime_state","transient_lock","unknown","credential"]`
			requireRefusal(t, run(mutateMember(t, fix, "excluded_classes", classes)), "session_adapter_protocol_error", "member", "excluded_classes", "exceed 9")
		}),
		wCustom("ctor|failProtocol|capture excluded class is outside the nine-class vocabulary", "tenth class", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpCapturePlan)
			requireRefusal(t, run(mutateMember(t, fix, "excluded_classes", `["archive_only"]`)), "session_adapter_protocol_error", "member", "excluded_classes", "nine-class vocabulary")
		}),
		wCustom("ctor|failProtocol|capture excluded classes are not sorted unique", "unsorted classes", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpCapturePlan)
			requireRefusal(t, run(mutateMember(t, fix, "excluded_classes", `["unknown","credential"]`)), "session_adapter_protocol_error", "member", "excluded_classes", "not sorted unique")
		}),
		wCustom("ctor|failProtocol|normalize canonical event identifiers are not sorted unique digest[0..65536]", "unsorted events", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpNormalize)
			first, second := fixtureDigest("a"), fixtureDigest("b")
			if first < second {
				first, second = second, first
			}
			ids := `["` + first + `","` + second + `"]`
			requireRefusal(t, run(mutateMember(t, fix, "canonical_event_candidate_ids", ids)), "session_adapter_protocol_error", "member", "canonical_event_candidate_ids", "sorted unique digest")
		}),
		wCustom("ctor|failProtocol|normalize raw references are not sorted unique digest[0..65536]", "dup refs", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpNormalize)
			id := fixtureDigest("a")
			requireRefusal(t, run(mutateMember(t, fix, "raw_reference_ids", `["`+id+`","`+id+`"]`)), "session_adapter_protocol_error", "member", "raw_reference_ids", "sorted unique digest")
		}),
		wCustom("ctor|failProtocol|projection request canonical event identifiers are not sorted unique digest[0..65536]", "dup request events", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProjectionPlan)
			id := fixtureDigest("a")
			requireRefusal(t, run(mutateMember(t, fix, "canonical_event_ids", `["`+id+`","`+id+`"]`)), "session_adapter_protocol_error", "member", "canonical_event_ids", "sorted unique digest")
		}),
		wCustom("ctor|failProtocol|projection request fidelity profile is outside the four-profile vocabulary", "fifth profile", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProjectionPlan)
			requireRefusal(t, run(mutateMember(t, fix, "fidelity_profile", `"archive_only"`)), "session_adapter_protocol_error", "member", "fidelity_profile", "four-profile vocabulary")
		}),
		wCustom("ctor|failProtocol|projection required dispositions carry no class", "empty dispositions", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProjectionPlan)
			requireRefusal(t, run(mutateMember(t, fix, "required_dispositions", `{}`)), "session_adapter_protocol_error", "member", "required_dispositions", "no class")
		}),
		wCustom("ctor|failProtocol|projection disposition list is not 1..7 entries", "eight dispositions", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProjectionPlan)
			lists := `{"user_message":["exact","semantic","summarized","opaque_preserved","synthesized","omitted","unrecoverable","exact"]}`
			requireRefusal(t, run(mutateMember(t, fix, "required_dispositions", lists)), "session_adapter_protocol_error", "member", "user_message", "1..7 entries")
		}),
		wCustom("ctor|failProtocol|projection disposition is outside the seven-disposition vocabulary", "bad disposition", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProjectionPlan)
			requireRefusal(t, run(mutateMember(t, fix, "required_dispositions", `{"user_message":["bogus"]}`)), "session_adapter_protocol_error", "member", "user_message", "seven-disposition vocabulary")
		}),
		wCustom("ctor|failProtocol|projection dispositions are not sorted unique", "unsorted dispositions", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProjectionPlan)
			requireRefusal(t, run(mutateMember(t, fix, "required_dispositions", `{"user_message":["summarized","exact"]}`)), "session_adapter_protocol_error", "member", "user_message", "not sorted unique")
		}),
		wCustom("ctor|failProtocol|projection request forbid reasons are not sorted-unique-string[1..128][0..128]", "dup reasons", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpProjectionPlan)
			requireRefusal(t, run(mutateMember(t, fix, "forbid_reasons", `["a","a"]`)), "session_adapter_protocol_error", "member", "forbid_reasons", "sorted-unique-string")
		}),
		wCustom("ctor|failProtocol|projection required source objects are not sorted unique digest[0..65536]", "dup required objects", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpProjectionPlan)
			id := fixtureDigest("a")
			requireRefusal(t, run(mutateMember(t, fix, "required_source_object_ids", `["`+id+`","`+id+`"]`)), "session_adapter_protocol_error", "member", "required_source_object_ids", "sorted unique digest")
		}),
		wCustom("ctor|failProtocol|project created resource keys are not sorted unique string[1..512][0..65536]", "dup keys", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpProject)
			requireRefusal(t, run(mutateMember(t, fix, "created_resource_keys", `["k","k"]`)), "session_adapter_protocol_error", "member", "created_resource_keys", "sorted unique string")
		}),
		wCustom("ctor|failProtocol|read-back head identifiers are not sorted unique string[1..512][0..1024]", "dup heads", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpReadBack)
			requireRefusal(t, run(mutateMember(t, fix, "parsed_head_ids", `["h","h"]`)), "session_adapter_protocol_error", "member", "parsed_head_ids", "sorted unique string")
		}),
		wCustom("ctor|failProtocol|validate request mode is outside staged|live|archive", "bad mode", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpValidate)
			requireRefusal(t, run(mutateMember(t, fix, "mode", `"draft"`)), "session_adapter_protocol_error", "member", "mode", "outside staged")
		}),
		wCustom("ctor|failProtocol|validate archive mode carries a target member", "archive with target", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpValidate)
			requireRefusal(t, run(mutateMember(t, fix, "mode", `"archive"`)), "session_adapter_protocol_error", "member", "projection_plan_id", "carries a target member")
		}),
		wCustom("ctor|failProtocol|validate staged/live mode misses a target member", "staged without target", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpValidate)
			requireRefusal(t, run(mutateMember(t, fix, "projection_plan_id", `null`)), "session_adapter_protocol_error", "member", "projection_plan_id", "misses a target member")
		}),
		wCustom("ctor|failProtocol|validate result mode is outside staged|live|archive", "bad result mode", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpValidate)
			requireRefusal(t, run(mutateMember(t, fix, "mode", `"draft"`)), "session_adapter_protocol_error", "member", "mode", "outside staged")
		}),
		wCustom("ctor|failProtocol|validate result mode does not echo the request mode", "mode drift", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpValidate)
			requireRefusal(t, run(mutateMember(t, fix, "mode", `"live"`)), "session_adapter_protocol_error", "member", "mode", "does not echo")
		}),
		wCustom("ctor|failProtocol|validate reports valid with a failed structural or semantic check", "valid with failure", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpValidate)
			requireRefusal(t, run(mutateMember(t, fix, "structural_valid", `false`)), "session_adapter_protocol_error", "member", "valid", "failed structural or semantic")
		}),
		wCustom("ctor|failProtocol|validate reports valid with a failed applicable check", "valid with failed check", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpValidate)
			requireRefusal(t, run(mutateMember(t, fix, "identity_valid", `false`)), "session_adapter_protocol_error", "member", "identity_valid", "failed applicable check")
		}),
		wCustom("ctor|failProtocol|validate reports valid with an error finding", "valid with error", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpValidate)
			finding := `{"severity":"error","code":"v","message":"m","remediation":null,"extensions":{}}`
			requireRefusal(t, run(mutateMember(t, fix, "findings", `[`+finding+`]`)), "session_adapter_protocol_error", "member", "findings", "error finding")
		}),
		wCustom("ctor|failProtocol|resume-plan argv is not 1..128 entries", "empty argv", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpResumePlan)
			requireRefusal(t, run(mutateMember(t, fix, "argv", `[]`)), "session_adapter_protocol_error", "member", "argv", "1..128 entries")
		}),
		wCustom("ctor|failProtocol|resume-plan argv word is not a string[1..4096]", "empty word", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpResumePlan)
			requireRefusal(t, run(mutateMember(t, fix, "argv", `[""]`)), "session_adapter_protocol_error", "member", "argv", "string[1..4096]")
		}),
		wCustom("ctor|failProtocol|resume-plan argv does not carry the explicit identity exactly once", "missing identity", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpResumePlan)
			requireRefusal(t, run(mutateMember(t, fix, "argv", `["ax","open"]`)), "session_adapter_protocol_error", "member", "argv", "exactly once")
		}),
		wCustom("ctor|failProtocol|resume-plan does not open the existing identity", "closed identity", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpResumePlan)
			requireRefusal(t, run(mutateMember(t, fix, "opens_existing_identity", `false`)), "session_adapter_protocol_error", "member", "opens_existing_identity", "existing identity")
		}),
		wCustom("ctor|failProtocol|resume-plan environment names are not sorted unique string[1..256][0..128]", "dup names", func(t *testing.T) {
			t.Helper()
			run, fix := suc(OpResumePlan)
			requireRefusal(t, run(mutateMember(t, fix, "environment_names", `["a","a"]`)), "session_adapter_protocol_error", "member", "environment_names", "sorted unique string")
		}),
		wCustom("ctor|failProtocol|doctor request direction is outside source_read|target_write", "bad direction", func(t *testing.T) {
			t.Helper()
			run, fix := req(OpDoctor)
			requireRefusal(t, run(mutateMember(t, fix, "direction", `"sideways"`)), "session_adapter_protocol_error", "member", "direction", "outside source_read")
		}),
		wCustom("ctor|failProtocol|doctor result carries unknown member", "unknown result member", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.doctorRun(appendMember(t, e.doctorFix, "unexpected", `true`)), "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		}),
		wCustom("ctor|failProtocol|doctor result misses a required member", "missing findings", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.doctorRun(dropMember(t, e.doctorFix, "findings")), "session_adapter_protocol_error", "member", "findings", "misses a required member")
		}),
		wCustom("ctor|failProtocol|doctor result direction is outside source_read|target_write", "bad result direction", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.doctorRun(mutateMember(t, e.doctorFix, "direction", `"sideways"`)), "session_adapter_protocol_error", "member", "direction", "outside source_read")
		}),
		wCustom("ctor|failProtocol|doctor result direction does not answer the request direction", "direction drift", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.doctorRun(mutateMember(t, e.doctorFix, "direction", `"target_write"`)), "session_adapter_protocol_error", "member", "direction", "request direction")
		}),
		wCustom("ctor|failProtocol|doctor result registry sequence is not a uint53", "bad sequence", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.doctorRun(mutateMember(t, e.doctorFix, "registry_sequence", `"nine"`)), "session_adapter_protocol_error", "member", "registry_sequence", "not a uint53")
		}),
		wCustom("ctor|failProtocol|doctor result entry status is outside accepted|revoked|absent", "bad entry status", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.doctorRun(mutateMember(t, e.doctorFix, "registry_entry_status", `"pending"`)), "session_adapter_protocol_error", "member", "registry_entry_status", "accepted|revoked|absent")
		}),
		wCustom("ctor|failProtocol|doctor result health flag is not a boolean", "bad health", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.doctorRun(mutateMember(t, e.doctorFix, "healthy", `"yes"`)), "session_adapter_protocol_error", "member", "healthy", "not a boolean")
		}),
		wCustom("ctor|failProtocol|doctor result extensions are not reverse-DNS keyed", "bad result extensions", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.doctorRun(mutateMember(t, e.doctorFix, "extensions", `{"x":"y"}`)), "session_adapter_protocol_error", "member", "extensions", "not reverse-DNS keyed")
		}),
	}
	return witnesses
}

// dropNested drops one member of a nested object.
func dropNested(t *testing.T, body []byte, object, member string) []byte {
	t.Helper()
	inner := rawMember(t, body, object)
	dropped := dropMember(t, inner, member)
	return mutateMember(t, body, object, string(dropped))
}

// setNested replaces one member of a nested object.
func setNested(t *testing.T, body []byte, object, member, value string) []byte {
	t.Helper()
	inner := rawMember(t, body, object)
	updated := mutateMember(t, inner, member, value)
	return mutateMember(t, body, object, string(updated))
}

// dupNestedMember duplicates the first member of a nested object
// by pure string surgery: the outer streaming decode raw-copies
// the nested value with duplicates intact, while the inner
// strict decode refuses them. A map round-trip would collapse
// the duplicate, so this helper never parses.
func dupNestedMember(t *testing.T, body []byte, object, key, value string) []byte {
	t.Helper()
	marker := `"` + object + `":{"` + key + `"`
	if !strings.Contains(string(body), marker) {
		t.Fatalf("dupNestedMember: %q not in body", marker)
	}
	return []byte(strings.Replace(string(body), marker, `"`+object+`":{"`+key+`":`+value+`,"`+key+`"`, 1))
}

func gateWitnesses(e *witnessEntries) []armWitness {
	return []armWitness{
		wCustom("ctor|failCapability|target write has neither usable canonical-write nor official-import", "pair down", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			adapter := cloneCapabilities(probe.Capabilities)
			adapter["canonical_write"] = Capability{Status: "unsupported", Enabled: false, Evidence: "probed"}
			adapter["official_import"] = Capability{Status: "conditional", Enabled: false, Evidence: "probed"}
			provider := map[string]ProviderCapability{
				"portable_store": {Status: "available", Enabled: true},
				"native_resume":  {Status: "available", Enabled: true},
			}
			requireRefusal(t, CheckTargetWriteGates(adapter, provider), "capability_unavailable", "capability", "canonical_write|official_import", "neither usable")
		}),
		wCustom("ctor|failCapability|target write misses a usable adapter capability", "read-back down", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			adapter := cloneCapabilities(probe.Capabilities)
			adapter["native_read_back"] = Capability{Status: "unsupported", Enabled: false, Evidence: "probed"}
			provider := map[string]ProviderCapability{
				"portable_store": {Status: "available", Enabled: true},
				"native_resume":  {Status: "available", Enabled: true},
			}
			requireRefusal(t, CheckTargetWriteGates(adapter, provider), "capability_unavailable", "capability", "native_read_back", "usable adapter capability")
		}),
		wCustom("ctor|failCapability|target write misses a usable provider capability", "store down", func(t *testing.T) {
			t.Helper()
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			provider := map[string]ProviderCapability{
				"portable_store": {Status: "unsupported", Enabled: false},
				"native_resume":  {Status: "available", Enabled: true},
			}
			requireRefusal(t, CheckTargetWriteGates(probe.Capabilities, provider), "capability_unavailable", "capability", "provider:portable_store", "usable provider capability")
		}),
		wCustom("ctor|failCapability|doctor reports healthy without an accepted registry entry", "healthy revoked", func(t *testing.T) {
			t.Helper()
			revoked := mutateMember(t, e.doctorFix, "registry_entry_status", `"revoked"`)
			revoked = mutateMember(t, revoked, "healthy", `true`)
			result, err := DecodeDoctorResult(revoked, e.doctorCtx, DirectionSourceRead)
			if err != nil {
				t.Fatalf("DecodeDoctorResult: %v", err)
			}
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			requireRefusal(t, CheckDoctorHealthy(result, probe, nil), "capability_unavailable", "capability", "tuple_registry", "accepted registry entry")
		}),
		wCustom("ctor|failCapability|doctor reports healthy without a usable required capability", "healthy weak", func(t *testing.T) {
			t.Helper()
			healthy := mutateMember(t, e.doctorFix, "healthy", `true`)
			result, err := DecodeDoctorResult(healthy, e.doctorCtx, DirectionSourceRead)
			if err != nil {
				t.Fatalf("DecodeDoctorResult: %v", err)
			}
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			weakened := probe
			weakened.Capabilities = cloneCapabilities(probe.Capabilities)
			weakened.Capabilities["native_discovery"] = Capability{Status: "conditional", Enabled: false, Evidence: "probed"}
			requireRefusal(t, CheckDoctorHealthy(result, weakened, []string{"native_discovery"}), "capability_unavailable", "capability", "native_discovery", "usable required capability")
		}),
		wCustom("ctor|failInvalid|doctor required set names a capability outside the closed registry", "unknown required", func(t *testing.T) {
			t.Helper()
			healthy := mutateMember(t, e.doctorFix, "healthy", `true`)
			result, err := DecodeDoctorResult(healthy, e.doctorCtx, DirectionSourceRead)
			if err != nil {
				t.Fatalf("DecodeDoctorResult: %v", err)
			}
			probe, err := DecodeProbe(e.probeFix)
			if err != nil {
				t.Fatalf("DecodeProbe: %v", err)
			}
			requireRefusal(t, CheckDoctorHealthy(result, probe, []string{"teleport"}), "invalid_config", "field", "required", "closed registry")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple entry direction does not match the call direction", "direction drift", func(t *testing.T) {
			t.Helper()
			entry, tuple, binding := admissionTriple(t, e)
			requireRefusal(t, CheckTupleAdmission(entry, DirectionSourceRead, tuple, binding, mustTime(t, fixtureCallTime)), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "direction does not match")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple entry environment does not equal the probed tuple", "tuple drift", func(t *testing.T) {
			t.Helper()
			entry, tuple, binding := admissionTriple(t, e)
			tuple.Version = "9.9.9"
			requireRefusal(t, CheckTupleAdmission(entry, DirectionTargetWrite, tuple, binding, mustTime(t, fixtureCallTime)), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "probed tuple")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple entry key does not equal the host-observed execution binding", "binding drift", func(t *testing.T) {
			t.Helper()
			entry, tuple, binding := admissionTriple(t, e)
			binding.ProviderID = "other-provider"
			requireRefusal(t, CheckTupleAdmission(entry, DirectionTargetWrite, tuple, binding, mustTime(t, fixtureCallTime)), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "execution binding")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple entry is not accepted", "revoked entry", func(t *testing.T) {
			t.Helper()
			_, tuple, binding := admissionTriple(t, e)
			requireRefusal(t, CheckTupleAdmission(revokedEntry(t), DirectionTargetWrite, tuple, binding, mustTime(t, fixtureCallTime)), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "not accepted")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple entry validity interval does not cover the call", "stale entry", func(t *testing.T) {
			t.Helper()
			entry, tuple, binding := admissionTriple(t, e)
			requireRefusal(t, CheckTupleAdmission(entry, DirectionTargetWrite, tuple, binding, mustTime(t, "2028-01-01T00:00:00.000Z")), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "validity interval")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple source entry does not carry exactly strategies=[archive_only]", "wide source", func(t *testing.T) {
			t.Helper()
			source, err := DecodeTupleEntry([]byte(fixtureEntryJSON(DirectionSourceRead)))
			if err != nil {
				t.Fatalf("DecodeTupleEntry: %v", err)
			}
			tuple, err := DecodeTuple([]byte(fixtureTupleJSON()))
			if err != nil {
				t.Fatalf("DecodeTuple: %v", err)
			}
			source.Strategies = []string{"archive_only", "target_native_writer"}
			requireRefusal(t, CheckTupleAdmission(source, DirectionSourceRead, tuple, fixtureBindingFacts(), mustTime(t, fixtureCallTime)), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "archive_only")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple source entry carries resume smoke evidence", "smoky source", func(t *testing.T) {
			t.Helper()
			source, err := DecodeTupleEntry([]byte(fixtureEntryJSON(DirectionSourceRead)))
			if err != nil {
				t.Fatalf("DecodeTupleEntry: %v", err)
			}
			tuple, err := DecodeTuple([]byte(fixtureTupleJSON()))
			if err != nil {
				t.Fatalf("DecodeTuple: %v", err)
			}
			source.Smoke = &SmokeEvidence{EvidenceDigest: fixtureSmokeDigest, NativeCLIFamily: "claude"}
			requireRefusal(t, CheckTupleAdmission(source, DirectionSourceRead, tuple, fixtureBindingFacts(), mustTime(t, fixtureCallTime)), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "resume smoke")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple target entry admits archive_only", "archived target", func(t *testing.T) {
			t.Helper()
			entry, tuple, binding := admissionTriple(t, e)
			entry.Strategies = []string{"archive_only"}
			entry.Smoke = nil
			requireRefusal(t, CheckTupleAdmission(entry, DirectionTargetWrite, tuple, binding, mustTime(t, fixtureCallTime)), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "archive_only")
		}),
		wCustom("ctor|failUnsupportedTuple|tuple target entry misses passing resume smoke evidence", "smokeless target", func(t *testing.T) {
			t.Helper()
			entry, tuple, binding := admissionTriple(t, e)
			entry.Smoke = nil
			requireRefusal(t, CheckTupleAdmission(entry, DirectionTargetWrite, tuple, binding, mustTime(t, fixtureCallTime)), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "resume smoke")
		}),
		wCustom("ctor|failUnsupportedTuple|adapter call environment is not the admitted tuple", "call tuple drift", func(t *testing.T) {
			t.Helper()
			binding, context, admitted := callBindingTriple(t, e)
			other := admitted
			other.Version = "9.9.9"
			requireRefusal(t, CheckCallBinding(binding, RoleSource, context, other), "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "admitted tuple")
		}),
		wCustom("ctor|failIntegrity|adapter manifest provider identifier does not match the trusted candidate", "foreign manifest", func(t *testing.T) {
			t.Helper()
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			foreign := fixtureCandidate()
			foreign.ProviderID = "other-provider"
			_, err = Discover(RoleSource, manifest, e.manifestText, foreign)
			requireRefusal(t, err, "integrity_failure", "subject", "provider_id", "trusted candidate")
		}),
		wCustom("ctor|failIntegrity|session adapter binding no longer equals the freshly read trusted facts", "binding drift", func(t *testing.T) {
			t.Helper()
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			sealed, err := Discover(RoleSource, manifest, e.manifestText, fixtureCandidate())
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			fresh := sealed
			fresh.ExecutablePath = "/other/path"
			requireRefusal(t, CheckBindingEquality(sealed, fresh), "integrity_failure", "subject", "binding", "freshly read trusted facts")
		}),
		wCustom("ctor|failIntegrity|adapter call role does not match the sealed binding role", "role drift", func(t *testing.T) {
			t.Helper()
			binding, context, admitted := callBindingTriple(t, e)
			requireRefusal(t, CheckCallBinding(binding, RoleTarget, context, admitted), "integrity_failure", "subject", "role", "binding role")
		}),
		wCustom("ctor|failIntegrity|adapter call provider identifier does not match the sealed binding", "call provider drift", func(t *testing.T) {
			t.Helper()
			binding, context, admitted := callBindingTriple(t, e)
			context.ProviderID = "other-provider"
			requireRefusal(t, CheckCallBinding(binding, RoleSource, context, admitted), "integrity_failure", "subject", "provider_id", "sealed binding")
		}),
		wCustom("ctor|failIntegrity|adapter call manifest digest does not match the sealed binding", "call manifest drift", func(t *testing.T) {
			t.Helper()
			binding, context, admitted := callBindingTriple(t, e)
			context.ManifestDigest = fixtureDigest("other")
			requireRefusal(t, CheckCallBinding(binding, RoleSource, context, admitted), "integrity_failure", "subject", "session_adapter_manifest_digest", "sealed binding")
		}),
		wCustom("ctor|failIntegrity|adapter call executable digest does not match the sealed binding", "call executable drift", func(t *testing.T) {
			t.Helper()
			binding, context, admitted := callBindingTriple(t, e)
			context.ExecutableSHA256 = fixtureDigest("other")
			requireRefusal(t, CheckCallBinding(binding, RoleSource, context, admitted), "integrity_failure", "subject", "executable_sha256", "sealed binding")
		}),
		wCustom("ctor|failInvalid|discovery role is outside source|target", "bad role", func(t *testing.T) {
			t.Helper()
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			_, err = Discover(Role("sideways"), manifest, e.manifestText, fixtureCandidate())
			requireRefusal(t, err, "invalid_config", "field", "role", "outside source|target")
		}),
		wCustom("ctor|failInvalid|discovery candidate carries no executable path", "pathless candidate", func(t *testing.T) {
			t.Helper()
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			candidate := fixtureCandidate()
			candidate.ExecutablePath = ""
			_, err = Discover(RoleSource, manifest, e.manifestText, candidate)
			requireRefusal(t, err, "invalid_config", "field", "executable_path", "no executable path")
		}),
		wCustom("ctor|failInvalid|discovery candidate carries no owner identity", "ownerless candidate", func(t *testing.T) {
			t.Helper()
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			candidate := fixtureCandidate()
			candidate.OwnerIdentity = ""
			_, err = Discover(RoleSource, manifest, e.manifestText, candidate)
			requireRefusal(t, err, "invalid_config", "field", "owner_identity", "no owner identity")
		}),
		wCustom("ctor|failInvalid|discovery candidate executable digest is not a digest", "bad candidate digest", func(t *testing.T) {
			t.Helper()
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			candidate := fixtureCandidate()
			candidate.ExecutableSHA256 = "nope"
			_, err = Discover(RoleSource, manifest, e.manifestText, candidate)
			requireRefusal(t, err, "invalid_config", "field", "executable_sha256", "not a digest")
		}),
		wCustom("ctor|failInvalid|discovery candidate provider manifest digest is not a digest", "bad provider digest", func(t *testing.T) {
			t.Helper()
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			candidate := fixtureCandidate()
			candidate.ProviderManifestDigest = "nope"
			_, err = Discover(RoleSource, manifest, e.manifestText, candidate)
			requireRefusal(t, err, "invalid_config", "field", "provider_manifest_digest", "not a digest")
		}),
		wCustom("ctor|failInvalid|discovery adapter manifest digest is not a digest", "bad manifest digest", func(t *testing.T) {
			t.Helper()
			manifest, err := DecodeManifest(e.manifestFix)
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			_, err = Discover(RoleSource, manifest, "nope", fixtureCandidate())
			requireRefusal(t, err, "invalid_config", "field", "session_adapter_manifest_digest", "not a digest")
		}),
	}
}

// admissionTriple decodes a target entry, tuple, and binding facts
// that admit at the fixture instant.
func admissionTriple(t *testing.T, e *witnessEntries) (TupleEntry, Tuple, BindingFacts) {
	t.Helper()
	entry, err := DecodeTupleEntry(e.entryFix)
	if err != nil {
		t.Fatalf("DecodeTupleEntry: %v", err)
	}
	tuple, err := DecodeTuple([]byte(fixtureTupleJSON()))
	if err != nil {
		t.Fatalf("DecodeTuple: %v", err)
	}
	return entry, tuple, fixtureBindingFacts()
}

// callBindingTriple seals a source binding with a matching call
// context and admitted tuple.
func callBindingTriple(t *testing.T, e *witnessEntries) (ExecutionBinding, CallContext, Tuple) {
	t.Helper()
	manifest, err := DecodeManifest(e.manifestFix)
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	binding, err := Discover(RoleSource, manifest, e.manifestText, fixtureCandidate())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	body := fixtureRequestWithDigest(t, e.manifestText)
	context, err := CheckRequestBody(OpDiscover, body)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	admitted, err := DecodeTuple([]byte(fixtureTupleJSON()))
	if err != nil {
		t.Fatalf("DecodeTuple: %v", err)
	}
	return binding, context, admitted
}

func frameWitnesses(e *witnessEntries) []armWitness {
	_ = e
	fault := func(t *testing.T, frame []byte, detail string) {
		t.Helper()
		_, fault := decodeStrictObject(frame)
		if fault == nil {
			t.Fatalf("decodeStrictObject admitted %q", frame)
		}
		if fault.detail != detail {
			t.Fatalf("fault = %q, want %q", fault.detail, detail)
		}
	}
	return []armWitness{
		wCustom("frame|not valid UTF-8", "raw bytes", func(t *testing.T) {
			t.Helper()
			fault(t, []byte{0x7b, 0x22, 0x6d, 0x22, 0x3a, 0xff, 0x7d}, "not valid UTF-8")
		}),
		wCustom("frame|lone surrogate escape", "lone escape", func(t *testing.T) {
			t.Helper()
			fault(t, []byte(`{"member":"A`+`\ud800`+`B"}`), "lone surrogate escape")
		}),
		wCustom("frame|not a JSON object", "array frame", func(t *testing.T) {
			t.Helper()
			fault(t, []byte(`[1,2]`), "not a JSON object")
		}),
		wCustom("frame|duplicate member", "duplicated key", func(t *testing.T) {
			t.Helper()
			fault(t, []byte(`{"a":1,"a":2}`), "duplicate member")
		}),
		wCustom("frame|trailing data after the object", "trailing data", func(t *testing.T) {
			t.Helper()
			fault(t, []byte(`{} {}`), "trailing data after the object")
		}),
	}
}

func conduitWitnesses(e *witnessEntries) []armWitness {
	conduit := func(arm, prefix string, run func([]byte) error) armWitness {
		return wCustom(arm, "malformed body", func(t *testing.T) {
			t.Helper()
			err := run([]byte("{"))
			if err == nil {
				t.Fatalf("%s admitted a malformed body", arm)
			}
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "", prefix+"not a JSON object")
		})
	}
	return []armWitness{
		conduit("ctor|failProtocol|conduit|DecodeRequestFrame|request envelope ", "request envelope ", e.requestFrame),
		wCustom("ctor|failProtocol|conduit|DecodeRequestFrame|request body ", "duplicated body member", func(t *testing.T) {
			t.Helper()
			body := dupNestedMember(t, e.requestFix, "body", "expected_provider_id", `"test-provider"`)
			requireRefusal(t, e.requestFrame(body), "session_adapter_protocol_error", "member", "body", "request body duplicate member")
		}),
		conduit("ctor|failProtocol|conduit|CheckSuccessEnvelope|success envelope ", "success envelope ", e.successEnv),
		wCustom("ctor|failProtocol|conduit|CheckSuccessEnvelope|success body ", "duplicated body member", func(t *testing.T) {
			t.Helper()
			frame := []byte(fmt.Sprintf(`{"protocol":%q,"protocol_version":"1.0.0","request_id":%q,"operation":"doctor","ok":true,"body":{"direction":"source_read"}}`, ProtocolID, fixtureRequestID))
			body := dupNestedMember(t, frame, "body", "direction", `"source_read"`)
			requireRefusal(t, e.successEnv(body), "session_adapter_protocol_error", "member", "body", "success body duplicate member")
		}),
		conduit("ctor|failProtocol|conduit|CheckFailureEnvelope|failure envelope ", "failure envelope ", e.failureEnv),
		wCustom("ctor|failProtocol|conduit|CheckRequestBody|request body ", "malformed body", func(t *testing.T) {
			t.Helper()
			_, err := CheckRequestBody(OpDiscover, []byte("{"))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "", "request body not a JSON object")
		}),
		wCustom("ctor|failProtocol|conduit|CheckSuccessBody|success body ", "malformed body", func(t *testing.T) {
			t.Helper()
			err := CheckSuccessBody(OpDiscover, []byte("{"), e.opFacts[OpDiscover])
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "", "success body not a JSON object")
		}),
		conduit("ctor|failProtocol|conduit|DecodeCallContext|call context ", "call context ", e.context),
		conduit("ctor|failProtocol|conduit|DecodeCapturePlanItem|capture plan item ", "capture plan item ", e.planItem),
		wCustom("ctor|failProtocol|conduit|DecodeDoctorResult|doctor result ", "malformed body", func(t *testing.T) {
			t.Helper()
			_, err := DecodeDoctorResult([]byte("{"), e.doctorCtx, DirectionSourceRead)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "", "doctor result not a JSON object")
		}),
		conduit("ctor|failProtocol|conduit|DecodeFinding|adapter finding ", "adapter finding ", e.finding),
		conduit("ctor|failProtocol|conduit|DecodeManifest|adapter manifest ", "adapter manifest ", e.manifest),
		conduit("ctor|failProtocol|conduit|DecodeObjectAuthority|object authority ", "object authority ", e.objAuth),
		conduit("ctor|failProtocol|conduit|DecodeProbe|adapter probe ", "adapter probe ", e.probe),
		conduit("ctor|failProtocol|conduit|DecodeReadAuthority|read authority ", "read authority ", e.readAuth),
		conduit("ctor|failProtocol|conduit|DecodeResourceLimits|resource limits ", "resource limits ", e.limits),
		conduit("ctor|failProtocol|conduit|DecodeSourceSelector|source selector ", "source selector ", e.selector),
		conduit("ctor|failProtocol|conduit|DecodeTupleEntry|tuple registry entry ", "tuple registry entry ", e.entry),
		conduit("ctor|failProtocol|conduit|DecodeTuple|environment tuple ", "environment tuple ", e.tuple),
		wCustom("ctor|failProtocol|conduit|checkContractsShape|tuple contract row ", "scalar row", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "contracts", `[1]`)), "session_adapter_protocol_error", "member", "", "tuple contract row not a JSON object")
		}),
		wCustom("ctor|failProtocol|conduit|checkDiscoverSources|discover source entry ", "scalar entry", func(t *testing.T) {
			t.Helper()
			run, fix := e.opSuc[OpDiscover], e.opSucFix[OpDiscover]
			requireRefusal(t, run(mutateMember(t, fix, "sources", `[1]`)), "session_adapter_protocol_error", "member", "sources", "discover source entry not a JSON object")
		}),
		wCustom("ctor|failProtocol|conduit|checkRequiredDispositions|projection required dispositions ", "scalar map", func(t *testing.T) {
			t.Helper()
			run, fix := e.opReq[OpProjectionPlan], e.opReqFix[OpProjectionPlan]
			requireRefusal(t, run(mutateMember(t, fix, "required_dispositions", `[1]`)), "session_adapter_protocol_error", "member", "required_dispositions", "projection required dispositions not a JSON object")
		}),
		wCustom("ctor|failProtocol|conduit|decodeCapabilities|adapter capabilities ", "scalar map", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.probe(mutateMember(t, e.probeFix, "capabilities", `[1]`)), "session_adapter_protocol_error", "member", "capabilities", "adapter capabilities not a JSON object")
		}),
		wCustom("ctor|failProtocol|conduit|decodeCapabilityValue|adapter capability ", "malformed value", func(t *testing.T) {
			t.Helper()
			_, err := decodeCapabilityValue([]byte("{"), "tool_history")
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "tool_history", "adapter capability not a JSON object")
		}),
		wCustom("ctor|failProtocol|conduit|decodeFidelityLimits|tuple fidelity limit ", "scalar row", func(t *testing.T) {
			t.Helper()
			requireRefusal(t, e.entry(mutateMember(t, e.entryFix, "known_fidelity_limits", `[1]`)), "session_adapter_protocol_error", "member", "", "tuple fidelity limit not a JSON object")
		}),
		wCustom("ctor|failProtocol|conduit|decodeFixtureEvidence|tuple fixture evidence ", "duplicated evidence member", func(t *testing.T) {
			t.Helper()
			body := dupNestedMember(t, e.entryFix, "fixture_evidence", "suite_revision", `"rev-9"`)
			requireRefusal(t, e.entry(body), "session_adapter_protocol_error", "member", "suite_revision", "tuple fixture evidence duplicate member")
		}),
		wCustom("ctor|failProtocol|conduit|decodeSmokeEvidence|tuple resume smoke ", "duplicated smoke member", func(t *testing.T) {
			t.Helper()
			body := dupNestedMember(t, e.entryFix, "resume_smoke_evidence", "result", `"pass"`)
			requireRefusal(t, e.entry(body), "session_adapter_protocol_error", "member", "result", "tuple resume smoke duplicate member")
		}),
		wCustom("ctor|failProtocol|conduit|decodeTupleKey|tuple registry key ", "duplicated key member", func(t *testing.T) {
			t.Helper()
			body := dupNestedMember(t, e.entryFix, "key", "direction", `"target_write"`)
			requireRefusal(t, e.entry(body), "session_adapter_protocol_error", "member", "direction", "tuple registry key duplicate member")
		}),
	}
}
