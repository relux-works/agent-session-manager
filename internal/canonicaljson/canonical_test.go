package canonicaljson

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const zeroDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

func TestCanonicalizeMatchesPublishedRFC8785PrimitiveAndStringVector(t *testing.T) {
	input := []byte(`{
  "numbers": [333333333.33333329, 1E30, 4.50, 2e-3, 0.000000000000000000000000001],
  "string": "\u20ac$\u000F\u000aA'\u0042\u0022\u005c\\\"\/",
  "literals": [null, true, false]
}`)
	want := `{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],"string":"€$\u000f\nA'B\"\\\\\"/"}`

	got, err := Canonicalize(input)
	if err != nil {
		t.Fatalf("Canonicalize(RFC 8785 Section 3.2.2 vector) error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("Canonicalize(RFC 8785 Section 3.2.2 vector) = %q, want %q", got, want)
	}

	again, err := Canonicalize(got)
	if err != nil || string(again) != want {
		t.Fatalf("Canonicalize(canonical vector) = %q, %v; want identical %q", again, err, want)
	}

	primitive, err := Canonicalize([]byte(" \n -0 \t"))
	if err != nil || string(primitive) != "0" {
		t.Fatalf("Canonicalize(whitespace-wrapped minus zero) = %q, %v; want RFC value 0", primitive, err)
	}
}

func TestCanonicalizeMatchesEveryFiniteRFC8785AppendixBNumberSample(t *testing.T) {
	vectors := []struct {
		bits uint64
		want string
	}{
		{0x0000000000000000, "0"},
		{0x8000000000000000, "0"},
		{0x0000000000000001, "5e-324"},
		{0x8000000000000001, "-5e-324"},
		{0x7fefffffffffffff, "1.7976931348623157e+308"},
		{0xffefffffffffffff, "-1.7976931348623157e+308"},
		{0x4340000000000000, "9007199254740992"},
		{0xc340000000000000, "-9007199254740992"},
		{0x4430000000000000, "295147905179352830000"},
		{0x44b52d02c7e14af5, "9.999999999999997e+22"},
		{0x44b52d02c7e14af6, "1e+23"},
		{0x44b52d02c7e14af7, "1.0000000000000001e+23"},
		{0x444b1ae4d6e2ef4e, "999999999999999700000"},
		{0x444b1ae4d6e2ef4f, "999999999999999900000"},
		{0x444b1ae4d6e2ef50, "1e+21"},
		{0x3eb0c6f7a0b5ed8c, "9.999999999999997e-7"},
		{0x3eb0c6f7a0b5ed8d, "0.000001"},
		{0x41b3de4355555553, "333333333.3333332"},
		{0x41b3de4355555554, "333333333.33333325"},
		{0x41b3de4355555555, "333333333.3333333"},
		{0x41b3de4355555556, "333333333.3333334"},
		{0x41b3de4355555557, "333333333.33333343"},
		{0xbecbf647612f3696, "-0.0000033333333333333333"},
		{0x43143ff3c1cb0959, "1424953923781206.2"},
	}

	for _, vector := range vectors {
		name := fmt.Sprintf("%016x", vector.bits)
		t.Run(name, func(t *testing.T) {
			value := math.Float64frombits(vector.bits)
			input := strconv.FormatFloat(value, 'g', -1, 64)
			got, err := Canonicalize([]byte(input))
			if err != nil {
				t.Fatalf("Canonicalize(%q) error = %v", input, err)
			}
			if string(got) != vector.want {
				t.Fatalf("Canonicalize(%q) = %q, want RFC Appendix B %q", input, got, vector.want)
			}
		})
	}

	if _, err := Canonicalize([]byte("1e999")); err == nil {
		t.Fatal("Canonicalize(out-of-range number) error = nil, want I-JSON refusal")
	}
}

func TestCanonicalizeUsesRFC8785UTF16PropertyOrdering(t *testing.T) {
	input := []byte(`{
  "\u20ac": "Euro Sign",
  "\r": "Carriage Return",
  "\ufb33": "Hebrew Letter Dalet With Dagesh",
  "1": "One",
  "\ud83d\ude00": "Emoji: Grinning Face",
  "\u0080": "Control",
  "\u00f6": "Latin Small Letter O With Diaeresis"
}`)
	want := `{"\r":"Carriage Return","1":"One","":"Control","ö":"Latin Small Letter O With Diaeresis","€":"Euro Sign","😀":"Emoji: Grinning Face","דּ":"Hebrew Letter Dalet With Dagesh"}`

	got, err := Canonicalize(input)
	if err != nil {
		t.Fatalf("Canonicalize(RFC 8785 Section 3.2.3 vector) error = %v", err)
	}
	if string(got) != want {
		t.Fatalf("Canonicalize(RFC 8785 Section 3.2.3 vector) = %q, want %q", got, want)
	}
}

func TestGeneratedSelfIdentityRegistrySelectsEverySelfField(t *testing.T) {
	definitions := catalog.Current().SelfIdentities
	if len(definitions) == 0 {
		t.Fatal("generated v0.5.0 self-identity catalog is empty")
	}
	rows := 0
	for _, definition := range definitions {
		for _, version := range definition.ContractVersions {
			rows++
			testName := string(definition.ContractID) + "@" + version
			t.Run(testName, func(t *testing.T) {
				extra := ""
				if definition.DiscriminatorName != "" {
					extra = fmt.Sprintf("%q:%q,", definition.DiscriminatorName, definition.DiscriminatorValue)
				}
				selfField := SelfField(definition.SelfField)
				input := []byte(fmt.Sprintf(
					`{"schema":%q,"schema_version":%q,%s"%s":"%s","a":1}`,
					definition.ContractID,
					version,
					extra,
					selfField,
					zeroDigest,
				))
				value, err := decodeStrict(input)
				if err != nil {
					t.Fatalf("decodeStrict(%s) error = %v", testName, err)
				}
				object := value.(map[string]any)
				gotField, _, err := resolveSelfField(object)
				if err != nil || gotField != selfField {
					t.Fatalf("resolveSelfField(%s) = %q, %v; want %q", testName, gotField, err, selfField)
				}
			})
		}
	}
	if rows != len(schemaIdentityContracts) {
		t.Fatalf("production self-identity table has %d rows, generated catalog defines %d", len(schemaIdentityContracts), rows)
	}
}

func TestPublicIdentityRefusesRegisteredSchemasWithoutCompleteShapeValidators(t *testing.T) {
	tests := []struct {
		schema    string
		selfField SelfField
	}{
		{"urn:ax:schema:terminal-backend-probe", SelfProbeID},
		{"urn:ax:schema:terminal-instance-binding", SelfBindingID},
		{"urn:ax:schema:terminal-capability-evidence", SelfEvidenceID},
		{"urn:ax:schema:clone-raw-object-manifest", SelfRawObjectManifestID},
		{"urn:ax:schema:clone-capture-manifest", SelfCaptureManifestID},
		{"urn:ax:schema:canonical-session", SelfCanonicalSessionID},
		{"urn:ax:schema:fidelity-report", SelfFidelityReportID},
		{"urn:ax:schema:projection-plan", SelfProjectionPlanID},
		{"urn:ax:schema:clone-projected-object-manifest", SelfProjectedObjectManifestID},
		{"urn:ax:schema:clone-read-back-evidence-manifest", SelfReadBackEvidenceManifestID},
		{"urn:ax:schema:clone-validation-report", SelfValidationReportID},
		{"urn:ax:schema:migration-checkpoint", SelfMigrationCheckpointID},
		{"urn:ax:schema:clone-lineage-receipt", SelfLineageReceiptID},
		{"urn:ax:schema:supported-environment-tuples", SelfRegistryDigest},
	}
	for _, test := range tests {
		t.Run(test.schema, func(t *testing.T) {
			input := []byte(fmt.Sprintf(
				`{"schema":%q,"schema_version":"1.0.0","%s":"%s","referenced_id":"%s"}`,
				test.schema,
				test.selfField,
				zeroDigest,
				"sha256:1111111111111111111111111111111111111111111111111111111111111111",
			))
			value, err := decodeStrict(input)
			if err != nil {
				t.Fatal(err)
			}
			gotField, _, err := resolveSelfField(value.(map[string]any))
			if err != nil || gotField != test.selfField {
				t.Fatalf("resolveSelfField(%s) = %q, %v; want %q", test.schema, gotField, err, test.selfField)
			}
			if _, _, err := CalculateObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) || !strings.Contains(err.Error(), "complete immutable-object shape validation is unavailable") {
				t.Fatalf("CalculateObjectIdentity(%s without complete shape validator) error = %v, want explicit unsupported-shape refusal", test.schema, err)
			}
		})
	}
}

func TestSelfIdentityCompletenessRejectsDeletedImplementationContract(t *testing.T) {
	definitions := catalog.Current().SelfIdentities
	narrowed := make(map[schemaIdentityKey]schemaIdentityContract, len(schemaIdentityContracts))
	for key, contract := range schemaIdentityContracts {
		narrowed[key] = contract
	}
	deleted := schemaIdentityKey{schema: "urn:ax:schema:canonical-session", version: "1.0.0"}
	delete(narrowed, deleted)

	err := validateSchemaIdentityContracts(narrowed, definitions)
	if err == nil || !strings.Contains(err.Error(), "missing self-identity contract for urn:ax:schema:canonical-session@1.0.0") {
		t.Fatalf("validateSchemaIdentityContracts(deleted canonical-session row) error = %v, want exact missing-row refusal", err)
	}
}

func TestImmutableObjectShapeValidatorCompletenessRejectsDeletedRegisteredSchema(t *testing.T) {
	definitions := catalog.Current().SelfIdentities
	if err := validateImmutableObjectShapeValidators(immutableObjectShapeValidators, definitions); err != nil {
		t.Fatalf("validateImmutableObjectShapeValidators(complete registry) error = %v", err)
	}

	narrowed := make(map[schemaIdentityKey]immutableObjectShapeValidator, len(immutableObjectShapeValidators))
	for key, validator := range immutableObjectShapeValidators {
		narrowed[key] = validator
	}
	deleted := schemaIdentityKey{schema: "urn:ax:schema:session-record", version: "1.0.0"}
	delete(narrowed, deleted)

	err := validateImmutableObjectShapeValidators(narrowed, definitions)
	if err == nil || !strings.Contains(err.Error(), "missing immutable-object shape validator for urn:ax:schema:session-record@1.0.0") {
		t.Fatalf("validateImmutableObjectShapeValidators(deleted Session Record row) error = %v, want exact missing-row refusal", err)
	}
}

func TestSessionRecordV1ClosedShapeReachesBothIdentityEntries(t *testing.T) {
	t.Parallel()

	valid := validSessionRecordV1Object()
	assertIdentityEntriesAcceptShape(t, mustJSON(t, valid), SelfRecordID)

	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"missing common created_by_host_id", func(object map[string]any) { delete(object, "created_by_host_id") }},
		{"malformed common subject UUID", func(object map[string]any) { object["subject_id"] = "not-a-uuid" }},
		{"impossible common timestamp", func(object map[string]any) { object["created_at"] = "2023-02-29T12:00:00.000Z" }},
		{"unknown top-level member", func(object map[string]any) { object["unexpected_security_control"] = true }},
		{"unknown nested member", func(object map[string]any) {
			object["launch_plan"].(map[string]any)["unexpected_security_control"] = true
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalid := cloneJSONObject(t, valid)
			test.mutate(invalid)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, invalid), SelfRecordID)
		})
	}
}

func TestSessionRecordNameGrammarReachesBothIdentityEntries(t *testing.T) {
	t.Parallel()

	valid := validSessionRecordV1Object()
	valid["name"] = "A" + strings.Repeat("a", 63)
	assertIdentityEntriesAcceptShape(t, mustJSON(t, valid), SelfRecordID)

	for name, value := range map[string]string{
		"65 characters":  strings.Repeat("a", 65),
		"leading hyphen": "-payments",
		"space":          "payments api",
		"control":        "payments\napi",
		"non-ASCII":      "платежи",
	} {
		t.Run(name, func(t *testing.T) {
			invalid := validSessionRecordV1Object()
			invalid["name"] = value
			assertIdentityEntriesRefuseShape(t, mustJSON(t, invalid), SelfRecordID)
		})
	}
}

func TestSessionRecordV1NestedTaggedShapesReachBothIdentityEntries(t *testing.T) {
	t.Parallel()

	taskBoard := validSessionRecordV1Object()
	taskBoard["kind"] = "task_board"
	taskBoard["task_board"] = map[string]any{
		"bridge_protocol_version": "1.0.0",
		"board": map[string]any{
			"kind":       "local",
			"logical_id": "agent-session-manager",
			"remote_url": nil,
			"extensions": map[string]any{},
		},
		"task_element_id":     "TASK-260830-8x76g1",
		"launch_mode":         "tracked_prompt",
		"manager_session_ref": nil,
		"board_goal":          nil,
		"native_goal_binding": "prompt",
		"extensions":          map[string]any{},
	}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, taskBoard), SelfRecordID)

	primaryOwner := cloneJSONObject(t, taskBoard)
	primaryReference := primaryOwner["task_board"].(map[string]any)
	primaryReference["launch_mode"] = "primary_owner"
	primaryReference["native_goal_binding"] = "bound"
	primaryReference["board"] = map[string]any{
		"kind":       "remote",
		"logical_id": "remote-board:primary",
		"remote_url": "https://board.example.test/api",
		"extensions": map[string]any{},
	}
	primaryReference["board_goal"] = map[string]any{
		"schema":     "board-goal-v2",
		"goal_id":    "GOAL-260830-primary",
		"revision":   json.Number("1"),
		"extensions": map[string]any{},
	}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, primaryOwner), SelfRecordID)

	forked := validSessionRecordV1Object()
	forked["fork_provenance"] = map[string]any{
		"source_session_id":         "0198f4c8-9f60-7077-8071-1234567890ab",
		"source_checkpoint_id":      digestWithDigit('3'),
		"source_workspace_group_id": "0198f4c8-af70-7188-8172-1234567890ab",
		"operation_id":              "0198f4c8-b080-7299-8273-1234567890ab",
		"provider_fork_mode":        "native",
		"extensions":                map[string]any{},
	}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, forked), SelfRecordID)

	unknownBoard := cloneJSONObject(t, taskBoard)
	unknownBoard["task_board"].(map[string]any)["board"].(map[string]any)["unexpected"] = true
	assertIdentityEntriesRefuseShape(t, mustJSON(t, unknownBoard), SelfRecordID)

	unknownFork := cloneJSONObject(t, forked)
	unknownFork["fork_provenance"].(map[string]any)["unexpected"] = true
	assertIdentityEntriesRefuseShape(t, mustJSON(t, unknownFork), SelfRecordID)

	for _, test := range []struct {
		name   string
		object map[string]any
		mutate func(map[string]any)
	}{
		{"subject/session mismatch", validSessionRecordV1Object(), func(object map[string]any) { object["session_id"] = "0198f4c8-9f60-7077-8071-1234567890ab" }},
		{"invalid provider", validSessionRecordV1Object(), func(object map[string]any) { object["provider_id"] = "Codex" }},
		{"empty argv", validSessionRecordV1Object(), func(object map[string]any) { object["launch_plan"].(map[string]any)["argv"] = []any{} }},
		{"oversized argv element", validSessionRecordV1Object(), func(object map[string]any) {
			object["launch_plan"].(map[string]any)["argv"] = []any{strings.Repeat("x", 4097)}
		}},
		{"invalid environment name", validSessionRecordV1Object(), func(object map[string]any) { object["launch_plan"].(map[string]any)["env_names"] = []any{"1SECRET"} }},
		{"unsorted environment names", validSessionRecordV1Object(), func(object map[string]any) {
			object["launch_plan"].(map[string]any)["env_names"] = []any{"Z_KEY", "A_KEY"}
		}},
		{"duplicate literal environment", validSessionRecordV1Object(), func(object map[string]any) {
			object["launch_plan"].(map[string]any)["env_literals"] = map[string]any{"OPENAI_API_KEY": "literal"}
		}},
		{"contains secrets", validSessionRecordV1Object(), func(object map[string]any) { object["launch_plan"].(map[string]any)["contains_secrets"] = true }},
		{"direct task board reference", validSessionRecordV1Object(), func(object map[string]any) { object["task_board"] = map[string]any{} }},
		{"creation manager reference", taskBoard, func(object map[string]any) {
			object["task_board"].(map[string]any)["manager_session_ref"] = "manager-1"
		}},
		{"tracked prompt bound goal", taskBoard, func(object map[string]any) { object["task_board"].(map[string]any)["native_goal_binding"] = "bound" }},
		{"primary owner missing goal", primaryOwner, func(object map[string]any) { object["task_board"].(map[string]any)["board_goal"] = nil }},
		{"invalid remote board URL", primaryOwner, func(object map[string]any) {
			object["task_board"].(map[string]any)["board"].(map[string]any)["remote_url"] = "https://user@example.test/?token=secret"
		}},
		{"zero goal revision", primaryOwner, func(object map[string]any) {
			object["task_board"].(map[string]any)["board_goal"].(map[string]any)["revision"] = json.Number("0")
		}},
		{"invalid fork operation", forked, func(object map[string]any) { object["fork_provenance"].(map[string]any)["operation_id"] = "not-a-uuid" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := cloneJSONObject(t, test.object)
			test.mutate(invalid)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, invalid), SelfRecordID)
		})
	}
}

func TestUnsupportedSection10RecordSchemasValidateCommonEnvelopeBeforeRefusal(t *testing.T) {
	t.Parallel()

	keys := []schemaIdentityKey{
		{schema: "urn:ax:schema:tombstone", version: "1.0.0"},
		{schema: "urn:ax:schema:tombstone-ack", version: "1.0.0"},
	}
	for _, key := range keys {
		t.Run(key.schema+"@"+key.version, func(t *testing.T) {
			contract := schemaIdentityContracts[key]
			object := map[string]any{
				"schema":                   key.schema,
				"schema_version":           key.version,
				string(contract.selfField): zeroDigest,
				"subject_id":               "0198f4c8-3e70-7a11-8a2b-1234567890ab",
				"created_by_host_id":       "0198f4c8-4a10-7b22-8b3c-1234567890ab",
				"created_at":               "2026-08-19T04:00:00.000Z",
				"extensions":               map[string]any{},
			}
			input := mustJSON(t, object)
			if _, _, err := CalculateObjectIdentity(input); err == nil || !strings.Contains(err.Error(), "complete immutable-object shape validation is unavailable") {
				t.Fatalf("CalculateObjectIdentity(%s complete shape unavailable) error = %v, want explicit refusal", key.schema, err)
			}
			object["subject_id"] = "not-a-uuid"
			if _, _, err := CalculateObjectIdentity(mustJSON(t, object)); err == nil || !strings.Contains(err.Error(), "subject_id") {
				t.Fatalf("CalculateObjectIdentity(%s malformed common envelope) error = %v, want subject_id refusal before unsupported shape", key.schema, err)
			}
		})
	}
}

func TestSelfFieldResolutionCoversEveryRegisteredVersionForMultiVersionSchemas(t *testing.T) {
	tests := []struct {
		schema    string
		versions  []string
		selfField SelfField
	}{
		{"urn:ax:schema:session-record", []string{"1.0.0", "2.0.0", "3.0.0"}, SelfRecordID},
		{"urn:ax:schema:session-event", []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"}, SelfEventID},
		{"urn:ax:schema:materialization-plan", []string{"1.0.0", "2.0.0"}, SelfPlanID},
	}
	for _, test := range tests {
		for _, version := range test.versions {
			name := test.schema + "@" + version
			t.Run(name, func(t *testing.T) {
				input := []byte(fmt.Sprintf(
					`{"schema":%q,"schema_version":%q,"%s":"%s"}`,
					test.schema,
					version,
					test.selfField,
					zeroDigest,
				))
				value, err := decodeStrict(input)
				if err != nil {
					t.Fatal(err)
				}
				gotField, _, err := resolveSelfField(value.(map[string]any))
				if err != nil || gotField != test.selfField {
					t.Fatalf("resolveSelfField(%s) = %q, %v; want %q", name, gotField, err, test.selfField)
				}
			})
		}
	}
}

func TestSelfFieldResolutionDoesNotConfuseRegisteredReferenceFields(t *testing.T) {
	referenceA := "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	referenceB := "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	tests := []struct {
		name      string
		selfField SelfField
		input     string
	}{
		{
			"Session Annotation",
			SelfAnnotationID,
			`{"schema":"urn:ax:schema:session-annotation","schema_version":"1.0.0","annotation_id":"` + zeroDigest + `","profile_id":"` + referenceA + `"}`,
		},
		{
			"Enrichment Job Request",
			SelfJobRequestID,
			`{"schema":"urn:ax:schema:session-enrichment-job-request","schema_version":"1.0.0","job_request_id":"` + zeroDigest + `","profile_id":"` + referenceA + `"}`,
		},
		{
			"Enrichment Job Receipt",
			SelfJobReceiptID,
			`{"schema":"urn:ax:schema:session-enrichment-job-receipt","schema_version":"1.0.0","job_receipt_id":"` + zeroDigest + `","job_request_id":"` + referenceA + `","profile_id":"` + referenceB + `"}`,
		},
		{
			"Directory Operation Receipt",
			SelfDirectoryReceiptID,
			`{"schema":"urn:ax:schema:session-directory-operation-receipt","schema_version":"1.0.0","directory_receipt_id":"` + zeroDigest + `","plan_id":"` + referenceA + `"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := decodeStrict([]byte(test.input))
			if err != nil {
				t.Fatal(err)
			}
			gotField, _, err := resolveSelfField(value.(map[string]any))
			if err != nil || gotField != test.selfField {
				t.Fatalf("resolveSelfField(%s with registered references) = %q, %v; want %q", test.name, gotField, err, test.selfField)
			}
		})
	}
}

func TestCalculateObjectIdentityMatchesAXPublishedFixtures(t *testing.T) {
	t.Run("safe integer maximum", func(t *testing.T) {
		canonical, err := Canonicalize([]byte(`{"n":9007199254740991}`))
		got := scalar.SHA256Digest(canonical)
		want := "sha256:e1da48c6a6089f06ecb4e0a2259e658e3786b2420f52baccdf929ec6460d7b41"
		if err != nil || got.String() != want {
			t.Fatalf("Canonicalize/SHA256(NUM-SAFE-MAX) = %q, %v; want %q", got, err, want)
		}
	})

	// NUM-U64-STRING and NUM-U64-MAX are the two Section 1.6 boundary fixtures whose
	// published digests nothing in this repository recomputed. Both expected
	// values are quoted from SPEC.md at the pinned commit
	// 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c, whose SHA-256 is
	// internal/specpin.DocumentSHA256
	// (562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a), not
	// recomputed from this encoder: an expectation derived from the
	// implementation would agree with any drift the implementation acquires.
	//
	// SPEC.md:303, verbatim:
	//   | <code>NUM-U64-STRING</code> | <code>{"n":"9007199254740992"}</code> |
	//   Accept only when <code>n</code> is typed <code>decimal_uint64</code>;
	//   SHA-256 <code>bb80eb37329e0a7e980fe3638c9722c44ac3184f7488f20c28cf67ae0b5f4f96</code> |
	t.Run("decimal_uint64 first unsafe integer", func(t *testing.T) {
		canonical, err := Canonicalize([]byte(`{"n":"9007199254740992"}`))
		got := scalar.SHA256Digest(canonical)
		want := "sha256:bb80eb37329e0a7e980fe3638c9722c44ac3184f7488f20c28cf67ae0b5f4f96"
		if err != nil || got.String() != want {
			t.Fatalf("Canonicalize/SHA256(NUM-U64-STRING) = %q, %v; want %q", got, err, want)
		}
	})

	// SPEC.md:304, verbatim:
	//   | <code>NUM-U64-MAX</code> | <code>{"n":"18446744073709551615"}</code> |
	//   Accept only when <code>n</code> is typed <code>decimal_uint64</code>;
	//   SHA-256 <code>b0ec84c6bb6a7c030549f17dd482975d09c40ff9e5f83d4438ebeac12d3b6331</code> |
	t.Run("decimal_uint64 maximum", func(t *testing.T) {
		canonical, err := Canonicalize([]byte(`{"n":"18446744073709551615"}`))
		got := scalar.SHA256Digest(canonical)
		want := "sha256:b0ec84c6bb6a7c030549f17dd482975d09c40ff9e5f83d4438ebeac12d3b6331"
		if err != nil || got.String() != want {
			t.Fatalf("Canonicalize/SHA256(NUM-U64-MAX) = %q, %v; want %q", got, err, want)
		}
	})

	t.Run("UTF-16 order", func(t *testing.T) {
		canonical, err := Canonicalize([]byte(`{"\uE000":1,"\uD800\uDC00":2}`))
		got := scalar.SHA256Digest(canonical)
		want := "sha256:9d4cdc71dda603c42f9b21d88d0c2ffc31a76cd1bd461d7359406cf169845f1e"
		if err != nil || got.String() != want {
			t.Fatalf("Canonicalize/SHA256(JCS-UTF16-ORDER) = %q, %v; want %q", got, err, want)
		}
	})

	t.Run("Blob Descriptor", func(t *testing.T) {
		input := []byte(`{
  "schema": "urn:ax:schema:blob",
  "schema_version": "1.0.0",
  "descriptor_id": "sha256:390c8f21900483a010c1cbc3f9be01afebcf6e4da87263ba09fc5776dd6503ee",
  "blob_id": "sha256:9c21bad65c1b3d0403ac85d7d5bd134bb8d894432702a396a77b0477b8eb3b50",
  "size": 11,
  "media_type": "application/octet-stream",
  "chunks": [{
    "index": 0,
    "offset": 0,
    "size": 11,
    "chunk_id": "sha256:9c21bad65c1b3d0403ac85d7d5bd134bb8d894432702a396a77b0477b8eb3b50"
  }]
}`)
		got, field, err := VerifyObjectIdentity(input)
		if err != nil || field != SelfDescriptorID || got.Hex() != "390c8f21900483a010c1cbc3f9be01afebcf6e4da87263ba09fc5776dd6503ee" {
			t.Fatalf("VerifyObjectIdentity(Blob Descriptor) = %q, %q, %v", got, field, err)
		}
	})
}

func TestObjectIdentityIsStableAcrossRepresentationAndClaimChanges(t *testing.T) {
	object := validSessionRecordV1Object()
	object["extensions"] = map[string]any{"example.value": []any{map[string]any{"b": json.Number("2"), "a": json.Number("1")}}}
	first := mustJSON(t, object)
	calculated, field, err := CalculateObjectIdentity(first)
	if err != nil {
		t.Fatalf("CalculateObjectIdentity(first) error = %v", err)
	}
	object["record_id"] = calculated.String()
	second := append([]byte(" \r\n\t"), mustJSON(t, object)...)
	second = append(second, '\n', ' ')
	recalculated, secondField, err := CalculateObjectIdentity(second)
	if err != nil || recalculated != calculated || secondField != field {
		t.Fatalf("CalculateObjectIdentity(reordered) = %q, %q, %v; want %q, %q", recalculated, secondField, err, calculated, field)
	}
	verified, verifiedField, err := VerifyObjectIdentity(second)
	if err != nil || verified != calculated || verifiedField != SelfRecordID {
		t.Fatalf("VerifyObjectIdentity(reordered) = %q, %q, %v", verified, verifiedField, err)
	}
	for attempt := 0; attempt < 3; attempt++ {
		again, _, err := CalculateObjectIdentity(second)
		if err != nil || again != calculated {
			t.Fatalf("CalculateObjectIdentity idempotent attempt %d = %q, %v; want %q", attempt, again, err, calculated)
		}
	}
}

func TestObjectIdentityRefusesSelfInclusionAndAmbiguousNamespaces(t *testing.T) {
	placeholder := mustJSON(t, validSessionRecordV1Object())
	fullCanonical, err := Canonicalize(placeholder)
	if err != nil {
		t.Fatalf("Canonicalize(self-inclusion setup) error = %v", err)
	}
	selfIncludedClaim := digestHex(string(fullCanonical))
	selfIncludedObject := validSessionRecordV1Object()
	selfIncludedObject["record_id"] = selfIncludedClaim
	selfIncluded := mustJSON(t, selfIncludedObject)
	if _, _, err := VerifyObjectIdentity(selfIncluded); err == nil || !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("VerifyObjectIdentity(self-included claim) error = %v, want identity refusal", err)
	}

	cases := []struct {
		name  string
		input string
	}{
		{"absent schema", `{"schema_version":"1.0.0","record_id":"` + zeroDigest + `"}`},
		{"absent schema version", `{"schema":"urn:ax:schema:session-record","record_id":"` + zeroDigest + `"}`},
		{"unsupported schema", `{"schema":"urn:ax:schema:unknown","schema_version":"1.0.0","record_id":"` + zeroDigest + `"}`},
		{"unsupported schema version", `{"schema":"urn:ax:schema:session-record","schema_version":"999.0.0","record_id":"` + zeroDigest + `"}`},
		{"absent schema-defined self field", `{"schema":"urn:ax:schema:session-record","schema_version":"1.0.0","event_id":"` + zeroDigest + `"}`},
		{"raw chunk identity", `{"schema":"urn:ax:schema:chunk","schema_version":"1.0.0","chunk_id":"` + zeroDigest + `","size":1}`},
		{"mutable journal variant", `{"schema":"urn:ax:schema:materialization-journal","schema_version":"2.0.0","document_kind":"journal","marker_id":"` + zeroDigest + `"}`},
		{"non-digest claim", `{"schema":"urn:ax:schema:session-record","schema_version":"1.0.0","record_id":"not-a-digest","value":1}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := CalculateObjectIdentity([]byte(test.input)); err == nil || !errors.Is(err, ErrInvalidIdentity) {
				t.Fatalf("CalculateObjectIdentity(%s) error = %v, want identity refusal", test.name, err)
			}
		})
	}
}

func TestProductionEntriesRejectAmbiguousOrInvalidInput(t *testing.T) {
	canonicalCases := []struct {
		name  string
		input []byte
	}{
		{"duplicate member", []byte(`{"a":1,"\u0061":2}`)},
		{"lone high surrogate", []byte(`{"a":"\ud800"}`)},
		{"lone low surrogate", []byte(`{"a":"\udc00"}`)},
		{"mismatched surrogate", []byte(`{"a":"\ud800\u0041"}`)},
		{"invalid UTF-8", []byte{'{', '"', 'a', '"', ':', '"', 0xff, '"', '}'}},
	}
	for _, test := range canonicalCases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Canonicalize(test.input); err == nil || !errors.Is(err, ErrInvalidJSON) {
				t.Fatalf("Canonicalize(%s) error = %v, want invalid JSON refusal", test.name, err)
			}
		})
	}

	identityCases := []struct {
		name   string
		number string
	}{
		{"unsafe positive boundary", "9007199254740992"},
		{"unsafe negative boundary", "-9007199254740992"},
		{"rounded unsafe integer", "9007199254740993"},
		{"floating point", "1.0"},
		{"exponent", "1e3"},
	}
	for _, test := range identityCases {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(`{"schema":"urn:ax:schema:session-record","schema_version":"1.0.0","record_id":"` + zeroDigest + `","n":` + test.number + `}`)
			if _, _, err := CalculateObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) {
				t.Fatalf("CalculateObjectIdentity(%s) error = %v, want common-model refusal", test.name, err)
			}
		})
	}
}

func TestClosedIdentityShapesRefuseUnknownMembersAndBlobChunkInvariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"unknown top-level member", func(object map[string]any) { object["unexpected"] = true }},
		{"unknown nested member", func(object map[string]any) { firstChunk(object)["unexpected"] = true }},
		{"invalid media type", func(object map[string]any) { object["media_type"] = "Application/JSON; charset=utf-8" }},
		{"empty blob with chunk", func(object map[string]any) { object["size"] = json.Number("0") }},
		{"non-empty blob without chunks", func(object map[string]any) { object["chunks"] = []any{} }},
		{"chunk index order", func(object map[string]any) { firstChunk(object)["index"] = json.Number("1") }},
		{"chunk offset gap", func(object map[string]any) { firstChunk(object)["offset"] = json.Number("1") }},
		{"chunk zero size", func(object map[string]any) { firstChunk(object)["size"] = json.Number("0") }},
		{"chunk coverage short", func(object map[string]any) { object["size"] = json.Number("12") }},
		{"non-final chunk below fixed size", func(object map[string]any) {
			object["size"] = json.Number("2")
			object["chunks"] = []any{
				map[string]any{"index": json.Number("0"), "offset": json.Number("0"), "size": json.Number("1"), "chunk_id": digestWithDigit('1')},
				map[string]any{"index": json.Number("1"), "offset": json.Number("1"), "size": json.Number("1"), "chunk_id": digestWithDigit('2')},
			}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := validBlobDescriptorObject()
			test.mutate(object)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfDescriptorID)
		})
	}
}

func TestBlobDescriptorAcceptsEmptyAndFixedChunkPartitions(t *testing.T) {
	t.Parallel()

	empty := validBlobDescriptorObject()
	empty["size"] = json.Number("0")
	empty["chunks"] = []any{}
	partitioned := validBlobDescriptorObject()
	partitioned["size"] = json.Number("4194305")
	partitioned["chunks"] = []any{
		map[string]any{"index": json.Number("0"), "offset": json.Number("0"), "size": json.Number("4194304"), "chunk_id": digestWithDigit('1')},
		map[string]any{"index": json.Number("1"), "offset": json.Number("4194304"), "size": json.Number("1"), "chunk_id": digestWithDigit('2')},
	}
	for name, object := range map[string]map[string]any{"empty": empty, "partitioned": partitioned} {
		t.Run(name, func(t *testing.T) {
			input := mustJSON(t, object)
			if _, field, err := CalculateObjectIdentity(input); err != nil || field != SelfDescriptorID {
				t.Fatalf("CalculateObjectIdentity(%s Blob Descriptor) field = %q, error = %v", name, field, err)
			}
		})
	}
}

func TestBlobDescriptorMediaTypeBoundaryReachesBothIdentityEntries(t *testing.T) {
	t.Parallel()
	valid := validBlobDescriptorObject()
	valid["media_type"] = "a/" + strings.Repeat("b", 253)
	assertIdentityEntriesAcceptShape(t, mustJSON(t, valid), SelfDescriptorID)

	invalid := validBlobDescriptorObject()
	invalid["media_type"] = "a/" + strings.Repeat("b", 254)
	assertIdentityEntriesRefuseShape(t, mustJSON(t, invalid), SelfDescriptorID)
}

func TestTransferManifestRecursivelyRefusesUnknownClosedShapeMembers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		object func() map[string]any
		mutate func(map[string]any)
	}{
		{"top level", func() map[string]any { return validTransferManifestObject("workspace_tree") }, func(object map[string]any) {
			object["unexpected"] = true
		}},
		{"manifest entry", func() map[string]any { return validTransferManifestObject("workspace_tree") }, func(object map[string]any) {
			object["entries"] = []any{map[string]any{"path": "src", "type": "directory", "mode": json.Number("493"), "unexpected": true}}
		}},
		{"workspace snapshot", func() map[string]any { return validTransferManifestObject("workspace_group") }, func(object map[string]any) {
			object["workspace_snapshot"].(map[string]any)["unexpected"] = true
		}},
		{"workspace snapshot member", func() map[string]any { return validTransferManifestObject("workspace_group") }, func(object map[string]any) {
			snapshot := object["workspace_snapshot"].(map[string]any)
			snapshot["members"].([]any)[0].(map[string]any)["unexpected"] = true
		}},
		{"git remote", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["remotes"].([]any)[0].(map[string]any)["unexpected"] = true
		}},
		{"git head", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["head"].(map[string]any)["unexpected"] = true
		}},
		{"git object pack", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["object_pack"].(map[string]any)["unexpected"] = true
		}},
		{"git index", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["index"].(map[string]any)["unexpected"] = true
		}},
		{"git index entry", validGitWorkspaceGroupObject, func(object map[string]any) {
			index := gitWorkspaceMember(object)["index"].(map[string]any)
			index["entries"].([]any)[0].(map[string]any)["unexpected"] = true
		}},
		{"git submodule", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["submodules"].([]any)[0].(map[string]any)["unexpected"] = true
		}},
		{"git features", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["features"].(map[string]any)["unexpected"] = true
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := test.object()
			test.mutate(object)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfManifestID)
		})
	}
}

func TestTransferManifestAcceptsExactClosedGitWorkspaceSnapshot(t *testing.T) {
	t.Parallel()
	input := mustJSON(t, validGitWorkspaceGroupObject())
	digest, field, err := CalculateObjectIdentity(input)
	if err != nil || field != SelfManifestID {
		t.Fatalf("CalculateObjectIdentity(closed git workspace snapshot) = %q/%q, %v", digest, field, err)
	}
	claimed := withCorrectIdentityClaimForTest(t, input, SelfManifestID)
	verified, verifiedField, err := VerifyObjectIdentity(claimed)
	if err != nil || verified != digest || verifiedField != SelfManifestID {
		t.Fatalf("VerifyObjectIdentity(closed git workspace snapshot) = %q/%q, %v; want %q/%q", verified, verifiedField, err, digest, SelfManifestID)
	}
}

func TestTransferManifestAcceptsEveryClosedKindAndEntryVariant(t *testing.T) {
	t.Parallel()

	workspaceTree := validTransferManifestObject("workspace_tree")
	workspaceTree["entries"] = []any{
		map[string]any{"path": "a-dir", "type": "directory", "mode": json.Number("493")},
		map[string]any{"path": "b-file", "type": "file", "mode": json.Number("420"), "size": json.Number("1"), "blob_id": digestWithDigit('1'), "blob_descriptor_id": digestWithDigit('2')},
		map[string]any{"path": "c-link", "type": "symlink", "mode": json.Number("511"), "target": "b-file"},
		map[string]any{"path": "d-hard", "type": "hardlink", "mode": json.Number("420"), "target_path": "b-file"},
	}
	workspaceTree["excluded_classes"] = []any{"credential", "socket"}

	provider := validTransferManifestObject("provider")
	provider["provider_identity_record_id"] = digestWithDigit('3')
	taskBoard := validTransferManifestObject("task_board")
	taskBoard["task_board_bundle_id"] = digestWithDigit('4')
	composite := validTransferManifestObject("composite")
	composite["child_manifest_ids"] = []any{digestWithDigit('5')}

	for name, object := range map[string]map[string]any{
		"workspace_tree": workspaceTree,
		"provider":       provider,
		"task_board":     taskBoard,
		"composite":      composite,
	} {
		t.Run(name, func(t *testing.T) {
			input := mustJSON(t, object)
			digest, field, err := CalculateObjectIdentity(input)
			if err != nil || field != SelfManifestID {
				t.Fatalf("CalculateObjectIdentity(%s Transfer Manifest) = %q/%q, %v", name, digest, field, err)
			}
			claimed := withCorrectIdentityClaimForTest(t, input, SelfManifestID)
			if verified, verifiedField, err := VerifyObjectIdentity(claimed); err != nil || verified != digest || verifiedField != field {
				t.Fatalf("VerifyObjectIdentity(%s Transfer Manifest) = %q/%q, %v; want %q/%q", name, verified, verifiedField, err, digest, field)
			}
		})
	}
}

func TestTransferManifestRefusesTaggedAndSortedUniqueViolations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		object func() map[string]any
		mutate func(map[string]any)
	}{
		{"workspace group entries", func() map[string]any { return validTransferManifestObject("workspace_group") }, func(object map[string]any) {
			object["entries"] = []any{map[string]any{"path": "src", "type": "directory", "mode": json.Number("493")}}
		}},
		{"workspace tree snapshot", func() map[string]any { return validTransferManifestObject("workspace_tree") }, func(object map[string]any) {
			object["workspace_snapshot"] = map[string]any{}
		}},
		{"provider missing identity", func() map[string]any { return validTransferManifestObject("provider") }, func(map[string]any) {}},
		{"task board missing bundle", func() map[string]any { return validTransferManifestObject("task_board") }, func(map[string]any) {}},
		{"composite missing child", func() map[string]any { return validTransferManifestObject("composite") }, func(map[string]any) {}},
		{"unsorted child IDs", func() map[string]any { return validTransferManifestObject("composite") }, func(object map[string]any) {
			object["child_manifest_ids"] = []any{digestWithDigit('2'), digestWithDigit('1')}
		}},
		{"unsorted exclusions", func() map[string]any { return validTransferManifestObject("workspace_tree") }, func(object map[string]any) {
			object["excluded_classes"] = []any{"socket", "credential"}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := test.object()
			test.mutate(object)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfManifestID)
		})
	}
}

func TestMigrationProvenanceContributionIsClosedBeforeIdentityAttestation(t *testing.T) {
	t.Parallel()

	valid := validSessionRecordV1Object()
	valid["extensions"] = map[string]any{
		"works.relux.ax.migrated-from": map[string]any{
			"schema_id":      "urn:ax:schema:session-record",
			"schema_version": "0.9.0",
			"object_id":      digestWithDigit('3'),
		},
	}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, valid), SelfRecordID)

	emptySchemaID := cloneJSONObject(t, valid)
	emptySchemaID["extensions"].(map[string]any)["works.relux.ax.migrated-from"].(map[string]any)["schema_id"] = ""
	assertIdentityEntriesAcceptShape(t, mustJSON(t, emptySchemaID), SelfRecordID)

	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"unknown member", func(provenance map[string]any) { provenance["unexpected"] = true }},
		{"non-semver version", func(provenance map[string]any) { provenance["schema_version"] = "v1" }},
		{"malformed object ID", func(provenance map[string]any) { provenance["object_id"] = "not-a-digest" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := cloneJSONObject(t, valid)
			provenance := invalid["extensions"].(map[string]any)["works.relux.ax.migrated-from"].(map[string]any)
			test.mutate(provenance)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, invalid), SelfRecordID)
		})
	}
}

func TestUnboundedStringMembersDoNotGainAnUndeclaredNonEmptyConstraint(t *testing.T) {
	t.Parallel()

	manifest := validTransferManifestObject("workspace_tree")
	manifest["excluded_classes"] = []any{""}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, manifest), SelfManifestID)

	workspace := validGitWorkspaceGroupObject()
	gitWorkspaceMember(workspace)["features"].(map[string]any)["required_filter_names"] = []any{""}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, workspace), SelfManifestID)
}

func TestTransferManifestSanitizedGitURLsAllowHostOnlyAbsoluteFormsAtBothIdentityEntries(t *testing.T) {
	t.Parallel()

	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	member["remotes"].([]any)[0].(map[string]any)["fetch_url"] = "https://github.com"
	member["submodules"].([]any)[0].(map[string]any)["sanitized_url"] = "ssh://git@github.com"
	assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfManifestID)
}

func TestIdentityExtensionKeysUseCompleteReverseDNSGrammar(t *testing.T) {
	t.Parallel()

	maximumKey := strings.Join([]string{
		strings.Repeat("a", 63),
		strings.Repeat("b", 63),
		strings.Repeat("c", 63),
		strings.Repeat("d", 61),
	}, ".")
	for _, key := range []string{"a.b", "works.relux.ax.example-key", maximumKey} {
		t.Run("accept "+key[:min(len(key), 24)], func(t *testing.T) {
			object := genericExtensionIdentityObject(map[string]any{key: "данные"})
			assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfRecordID)
		})
	}

	for _, key := range []string{
		"not_namespaced",
		"A.b",
		"1a.b",
		"a..b",
		".a",
		"a.b_",
		strings.Repeat("a", 64) + ".b",
		maximumKey + "e",
	} {
		t.Run("refuse "+key[:min(len(key), 24)], func(t *testing.T) {
			object := genericExtensionIdentityObject(map[string]any{key: true})
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfRecordID)
		})
	}

	manifest := validTransferManifestObject("workspace_tree")
	manifest["extensions"] = map[string]any{"not_namespaced": true}
	assertIdentityEntriesRefuseShape(t, mustJSON(t, manifest), SelfManifestID)
}

func TestIdentityExtensionsEnforceCountDepthAndCanonicalSize(t *testing.T) {
	t.Parallel()

	maximumCount := make(map[string]any, 64)
	for index := range 64 {
		maximumCount[fmt.Sprintf("a.key%d", index)] = true
	}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, genericExtensionIdentityObject(maximumCount)), SelfRecordID)
	maximumCount["a.overflow"] = true
	assertIdentityEntriesRefuseShape(t, mustJSON(t, genericExtensionIdentityObject(maximumCount)), SelfRecordID)

	depthFour := []any{[]any{[]any{[]any{"value"}}}}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, genericExtensionIdentityObject(map[string]any{"a.depth": depthFour})), SelfRecordID)
	depthFive := []any{depthFour}
	assertIdentityEntriesRefuseShape(t, mustJSON(t, genericExtensionIdentityObject(map[string]any{"a.depth": depthFive})), SelfRecordID)

	atCanonicalSizeLimit := strings.Repeat("x", 65_523) // {"a.size":"..."} is exactly 65,536 bytes.
	assertIdentityEntriesAcceptShape(t, mustJSON(t, genericExtensionIdentityObject(map[string]any{"a.size": atCanonicalSizeLimit})), SelfRecordID)
	overCanonicalSizeLimit := atCanonicalSizeLimit + "x"
	assertIdentityEntriesRefuseShape(t, mustJSON(t, genericExtensionIdentityObject(map[string]any{"a.size": overCanonicalSizeLimit})), SelfRecordID)
}

func TestManifestStringBoundsCountUnicodeCharactersAtProductionEntries(t *testing.T) {
	t.Parallel()

	object := validTransferManifestObject("workspace_tree")
	object["entries"] = []any{map[string]any{
		"path":   "current",
		"type":   "symlink",
		"mode":   json.Number("511"),
		"target": strings.Repeat("界", 4096),
	}}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfManifestID)

	object["entries"].([]any)[0].(map[string]any)["target"] = strings.Repeat("界", 4097)
	assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfManifestID)
}

func TestTransferManifestNestedValueConstraintsReachBothIdentityEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"managed group path grammar", func(object map[string]any) {
			gitWorkspaceMember(object)["group_relative_path"] = "../escape"
		}},
		{"invalid real calendar date", func(object map[string]any) {
			object["created_at"] = "2023-02-29T12:00:00.000Z"
		}},
		{"malformed subject identifier", func(object map[string]any) {
			object["subject_id"] = "not-a-uuid"
		}},
		{"repository identity character bound", func(object map[string]any) {
			gitWorkspaceMember(object)["repository_identity"] = strings.Repeat("界", 257)
		}},
		{"remote name character bound", func(object map[string]any) {
			gitWorkspaceMember(object)["remotes"].([]any)[0].(map[string]any)["name"] = strings.Repeat("界", 129)
		}},
		{"remote URL credentials", func(object map[string]any) {
			gitWorkspaceMember(object)["remotes"].([]any)[0].(map[string]any)["fetch_url"] = "https://token@example.com/repo.git"
		}},
		{"detached head carries ref", func(object map[string]any) {
			head := gitWorkspaceMember(object)["head"].(map[string]any)
			head["mode"] = "detached"
		}},
		{"malformed head oid", func(object map[string]any) {
			gitWorkspaceMember(object)["head"].(map[string]any)["oid"] = "not-an-oid"
		}},
		{"malformed upstream ref", func(object map[string]any) {
			gitWorkspaceMember(object)["upstream_ref"] = "HEAD"
		}},
		{"object pack format", func(object map[string]any) {
			gitWorkspaceMember(object)["object_pack"].(map[string]any)["format"] = "zip"
		}},
		{"object pack malformed digest", func(object map[string]any) {
			gitWorkspaceMember(object)["object_pack"].(map[string]any)["blob_id"] = "not-a-digest"
		}},
		{"index version", func(object map[string]any) {
			gitWorkspaceMember(object)["index"].(map[string]any)["version"] = json.Number("5")
		}},
		{"index entry count", func(object map[string]any) {
			gitWorkspaceMember(object)["index"].(map[string]any)["entry_count"] = json.Number("99")
		}},
		{"TM-GIT-N2 stage four", func(object map[string]any) {
			gitWorkspaceMember(object)["index"].(map[string]any)["entries"].([]any)[0].(map[string]any)["stage"] = json.Number("4")
		}},
		{"index entry path grammar", func(object map[string]any) {
			gitWorkspaceMember(object)["index"].(map[string]any)["entries"].([]any)[0].(map[string]any)["path"] = "../README.md"
		}},
		{"index entry boolean type", func(object map[string]any) {
			gitWorkspaceMember(object)["index"].(map[string]any)["entries"].([]any)[0].(map[string]any)["skip_worktree"] = "false"
		}},
		{"uninitialized submodule live head", func(object map[string]any) {
			gitWorkspaceMember(object)["submodules"].([]any)[0].(map[string]any)["head"] = map[string]any{
				"mode": "detached", "oid": "sha1:" + strings.Repeat("3", 40), "ref": nil,
			}
		}},
		{"submodule malformed gitlink oid", func(object map[string]any) {
			gitWorkspaceMember(object)["submodules"].([]any)[0].(map[string]any)["gitlink_oid"] = "not-an-oid"
		}},
		{"features sparse blob mismatch", func(object map[string]any) {
			features := gitWorkspaceMember(object)["features"].(map[string]any)
			features["sparse_checkout"] = true
			features["sparse_patterns_blob_id"] = digestWithDigit('8')
		}},
		{"features boolean type", func(object map[string]any) {
			gitWorkspaceMember(object)["features"].(map[string]any)["filemode"] = json.Number("1")
		}},
		{"repo cwd grammar", func(object map[string]any) {
			gitWorkspaceMember(object)["repo_relative_cwd"] = "../escape"
		}},
		{"project paths sorted unique", func(object map[string]any) {
			gitWorkspaceMember(object)["agent_project_config_paths"] = []any{"z", "a"}
		}},
		{"materialization policy", func(object map[string]any) {
			gitWorkspaceMember(object)["materialization_policy"] = "copy"
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := validGitWorkspaceGroupObject()
			test.mutate(object)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfManifestID)
		})
	}
}

func TestTransferManifestNestedUnicodeBoundsAcceptExactMultibyteLimits(t *testing.T) {
	t.Parallel()

	object := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(object)
	member["repository_identity"] = strings.Repeat("界", 256)
	member["remotes"].([]any)[0].(map[string]any)["name"] = strings.Repeat("界", 128)
	member["submodules"].([]any)[0].(map[string]any)["repository_identity"] = "a" + strings.Repeat("界", 255)
	assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfManifestID)

	managed := validTransferManifestObject("workspace_group")
	managedMember := gitWorkspaceMember(managed)
	managedMember["tree_identity"] = strings.Repeat("界", 256)
	assertIdentityEntriesAcceptShape(t, mustJSON(t, managed), SelfManifestID)
}

func TestIdentityEntriesRefuseObjectsAboveEncodedSizeLimit(t *testing.T) {
	t.Parallel()
	object := validSessionRecordV1Object()
	object["extensions"] = map[string]any{"example.value": strings.Repeat("x", 5_242_881)}
	input := mustJSON(t, object)
	if _, _, err := CalculateObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("CalculateObjectIdentity(oversized object) error = %v, want size refusal", err)
	}
	if _, _, err := VerifyObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("VerifyObjectIdentity(oversized object) error = %v, want size refusal", err)
	}
}

func TestTransferManifestReachableOrderingAndCrossFieldRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		object func() map[string]any
		mutate func(map[string]any)
	}{
		{"symlink escape", func() map[string]any { return validTransferManifestObject("workspace_tree") }, func(object map[string]any) {
			object["entries"] = []any{map[string]any{"path": "link", "type": "symlink", "mode": json.Number("511"), "target": "../escape"}}
		}},
		{"hardlink target is not earlier file", func() map[string]any { return validTransferManifestObject("workspace_tree") }, func(object map[string]any) {
			object["entries"] = []any{map[string]any{"path": "hard", "type": "hardlink", "mode": json.Number("420"), "target_path": "missing"}}
		}},
		{"entry destination case collision", func() map[string]any { return validTransferManifestObject("workspace_tree") }, func(object map[string]any) {
			object["entries"] = []any{
				map[string]any{"path": "README", "type": "directory", "mode": json.Number("493")},
				map[string]any{"path": "readme", "type": "directory", "mode": json.Number("493")},
			}
		}},
		{"entry Unicode simple-fold collision", func() map[string]any { return validTransferManifestObject("workspace_tree") }, func(object map[string]any) {
			object["entries"] = []any{
				map[string]any{"path": "Σ", "type": "directory", "mode": json.Number("493")},
				map[string]any{"path": "ς", "type": "directory", "mode": json.Number("493")},
			}
		}},
		{"empty remotes", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["remotes"] = []any{}
		}},
		{"unsorted remotes", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["remotes"] = []any{
				map[string]any{"name": "z", "fetch_url": "https://example.com/z.git", "push_url": nil},
				map[string]any{"name": "a", "fetch_url": "https://example.com/a.git", "push_url": nil},
			}
		}},
		{"index path-stage order", validGitWorkspaceGroupObject, func(object map[string]any) {
			index := gitWorkspaceMember(object)["index"].(map[string]any)
			entries := index["entries"].([]any)
			entries[0], entries[1] = entries[1], entries[0]
		}},
		{"pack features object format mismatch", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["object_pack"].(map[string]any)["object_format"] = "sha256"
		}},
		{"submodule gitlink differs from parent index", validGitWorkspaceGroupObject, func(object map[string]any) {
			gitWorkspaceMember(object)["submodules"].([]any)[0].(map[string]any)["gitlink_oid"] = "sha1:" + strings.Repeat("4", 40)
		}},
		{"workspace group path case collision", func() map[string]any {
			object := validTransferManifestObject("workspace_group")
			snapshot := object["workspace_snapshot"].(map[string]any)
			second := map[string]any{
				"workspace_id":               "0198f4c8-8e50-7f66-8f70-3234567890ab",
				"kind":                       "managed_tree",
				"group_relative_path":        "DESIGN-NOTES",
				"tree_identity":              "relux/other",
				"tree_manifest_id":           digestWithDigit('6'),
				"repo_relative_cwd":          ".",
				"agent_project_config_paths": []any{},
				"materialization_policy":     "separate_copy",
			}
			snapshot["members"] = append(snapshot["members"].([]any), second)
			return object
		}, func(map[string]any) {}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object := test.object()
			test.mutate(object)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfManifestID)
		})
	}
}

func TestSimpleFoldKeyPreservesEqualFoldCollisionClasses(t *testing.T) {
	t.Parallel()

	for _, pair := range [][2]string{
		{"README", "readme"},
		{"Σ", "ς"},
		{"K", "K"},
	} {
		if !strings.EqualFold(pair[0], pair[1]) {
			t.Fatalf("test pair %q/%q is not EqualFold-equivalent", pair[0], pair[1])
		}
		if simpleFoldKey(pair[0]) != simpleFoldKey(pair[1]) {
			t.Fatalf("simpleFoldKey(%q) != simpleFoldKey(%q)", pair[0], pair[1])
		}
	}
}

func TestTransferManifestSubmoduleStateDepthAndCycleRules(t *testing.T) {
	t.Parallel()

	validDepth := validGitWorkspaceGroupObject()
	member := gitWorkspaceMember(validDepth)
	member["submodules"] = []any{validInitializedSubmoduleShape(1, 16)}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, validDepth), SelfManifestID)

	tooDeep := validGitWorkspaceGroupObject()
	gitWorkspaceMember(tooDeep)["submodules"] = []any{validInitializedSubmoduleShape(1, 17)}
	assertIdentityEntriesRefuseShape(t, mustJSON(t, tooDeep), SelfManifestID)

	missingState := validGitWorkspaceGroupObject()
	initialized := validInitializedSubmoduleShape(1, 1)
	initialized["working_tree_manifest_id"] = nil
	gitWorkspaceMember(missingState)["submodules"] = []any{initialized}
	assertIdentityEntriesRefuseShape(t, mustJSON(t, missingState), SelfManifestID)

	cycle := validGitWorkspaceGroupObject()
	cyclic := validInitializedSubmoduleShape(1, 2)
	cyclic["repository_identity"] = "relux/payments-api"
	gitWorkspaceMember(cycle)["submodules"] = []any{cyclic}
	assertIdentityEntriesRefuseShape(t, mustJSON(t, cycle), SelfManifestID)
}

func TestTransferManifestSubmoduleTotalCountBoundary(t *testing.T) {
	t.Parallel()

	maximum := validGitWorkspaceGroupObject()
	configureSubmoduleForest(gitWorkspaceMember(maximum), false)
	assertIdentityEntriesAcceptShape(t, mustJSON(t, maximum), SelfManifestID)

	overMaximum := validGitWorkspaceGroupObject()
	configureSubmoduleForest(gitWorkspaceMember(overMaximum), true)
	assertIdentityEntriesRefuseShape(t, mustJSON(t, overMaximum), SelfManifestID)
}

func genericExtensionIdentityObject(extensions map[string]any) map[string]any {
	object := validSessionRecordV1Object()
	object["extensions"] = extensions
	return object
}

func validSessionRecordV1Object() map[string]any {
	return map[string]any{
		"schema":             "urn:ax:schema:session-record",
		"schema_version":     "1.0.0",
		"record_id":          zeroDigest,
		"subject_id":         "0198f4c8-3e70-7a11-8a2b-1234567890ab",
		"session_id":         "0198f4c8-3e70-7a11-8a2b-1234567890ab",
		"name":               "payments-api",
		"kind":               "direct",
		"created_at":         "2026-08-19T04:00:00.000Z",
		"created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
		"provider_id":        "codex",
		"workspace_group_id": "0198f4c8-5b20-7c33-8c4d-1234567890ab",
		"execution_profile":  "yolo",
		"launch_plan": map[string]any{
			"argv":             []any{"codex"},
			"cwd_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890ab",
			"cwd_relative":     "src",
			"env_names":        []any{"OPENAI_API_KEY"},
			"env_literals":     map[string]any{},
			"contains_secrets": false,
			"extensions":       map[string]any{},
		},
		"task_board":      nil,
		"fork_provenance": nil,
		"extensions":      map[string]any{},
	}
}

func assertIdentityEntriesAcceptShape(t *testing.T, input []byte, selfField SelfField) {
	t.Helper()
	digest, calculatedField, err := CalculateObjectIdentity(input)
	if err != nil || calculatedField != selfField {
		t.Fatalf("CalculateObjectIdentity(%s valid shape) = %q/%q, %v", selfField, digest, calculatedField, err)
	}
	claimed := withCorrectIdentityClaimForTest(t, input, selfField)
	verified, verifiedField, err := VerifyObjectIdentity(claimed)
	if err != nil || verified != digest || verifiedField != selfField {
		t.Fatalf("VerifyObjectIdentity(%s valid shape) = %q/%q, %v; want %q/%q", selfField, verified, verifiedField, err, digest, selfField)
	}
}

func assertIdentityEntriesRefuseShape(t *testing.T, input []byte, selfField SelfField) {
	t.Helper()
	if _, _, err := CalculateObjectIdentity(input); err == nil || !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("CalculateObjectIdentity(%s malformed shape) error = %v, want identity refusal", selfField, err)
	}
	claimed := withCorrectIdentityClaimForTest(t, input, selfField)
	if _, _, err := VerifyObjectIdentity(claimed); err == nil || !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("VerifyObjectIdentity(%s malformed shape with correct omit-self claim) error = %v, want shape refusal", selfField, err)
	}
}

func validBlobDescriptorObject() map[string]any {
	return map[string]any{
		"schema":         blobSchema,
		"schema_version": closedSchemaVersion,
		"descriptor_id":  zeroDigest,
		"blob_id":        "sha256:9c21bad65c1b3d0403ac85d7d5bd134bb8d894432702a396a77b0477b8eb3b50",
		"size":           json.Number("11"),
		"media_type":     "application/octet-stream",
		"chunks": []any{map[string]any{
			"index":    json.Number("0"),
			"offset":   json.Number("0"),
			"size":     json.Number("11"),
			"chunk_id": "sha256:9c21bad65c1b3d0403ac85d7d5bd134bb8d894432702a396a77b0477b8eb3b50",
		}},
	}
}

func validTransferManifestObject(kind string) map[string]any {
	const subjectID = "0198f4c8-5b20-7c33-8c4d-1234567890ab"
	object := map[string]any{
		"schema":                      transferManifestSchema,
		"schema_version":              closedSchemaVersion,
		"manifest_id":                 zeroDigest,
		"kind":                        kind,
		"subject_id":                  subjectID,
		"base_checkpoint_id":          nil,
		"entries":                     []any{},
		"child_manifest_ids":          []any{},
		"workspace_snapshot":          nil,
		"provider_identity_record_id": nil,
		"task_board_bundle_id":        nil,
		"excluded_classes":            []any{},
		"created_by_host_id":          "0198f4c8-4a10-7b22-8b3c-1234567890ab",
		"created_at":                  "2026-08-19T04:15:00.000Z",
		"extensions":                  map[string]any{},
	}
	if kind == "workspace_group" {
		object["child_manifest_ids"] = []any{digestWithDigit('4')}
		object["workspace_snapshot"] = map[string]any{
			"workspace_group_id": subjectID,
			"members": []any{map[string]any{
				"workspace_id":               "0198f4c8-7d40-7e55-8e6f-2234567890ab",
				"kind":                       "managed_tree",
				"group_relative_path":        "design-notes",
				"tree_identity":              "relux/design-notes",
				"tree_manifest_id":           digestWithDigit('5'),
				"repo_relative_cwd":          ".",
				"agent_project_config_paths": []any{},
				"materialization_policy":     "separate_copy",
			}},
		}
	}
	return object
}

func validGitWorkspaceGroupObject() map[string]any {
	object := validTransferManifestObject("workspace_group")
	snapshot := object["workspace_snapshot"].(map[string]any)
	snapshot["members"] = []any{map[string]any{
		"workspace_id":        "0198f4c8-6c30-7d44-8d5e-1234567890ab",
		"kind":                "git",
		"group_relative_path": "payments-api",
		"repository_identity": "relux/payments-api",
		"remotes": []any{map[string]any{
			"name":      "origin",
			"fetch_url": "ssh://git@github.com/relux/payments-api.git",
			"push_url":  nil,
		}},
		"head": map[string]any{
			"mode": "branch",
			"oid":  "sha1:" + strings.Repeat("1", 40),
			"ref":  "refs/heads/main",
		},
		"upstream_ref": nil,
		"object_pack": map[string]any{
			"format":                       "git_pack_v2",
			"object_format":                "sha1",
			"blob_id":                      digestWithDigit('1'),
			"blob_descriptor_id":           digestWithDigit('2'),
			"object_count":                 json.Number("0"),
			"inventory_blob_id":            digestWithDigit('3'),
			"inventory_blob_descriptor_id": digestWithDigit('4'),
		},
		"index": map[string]any{
			"format":             "git_index",
			"version":            json.Number("2"),
			"blob_id":            digestWithDigit('5'),
			"blob_descriptor_id": digestWithDigit('6'),
			"entries": []any{map[string]any{
				"path":             "README.md",
				"stage":            json.Number("0"),
				"mode":             json.Number("33188"),
				"oid":              "sha1:" + strings.Repeat("2", 40),
				"intent_to_add":    false,
				"skip_worktree":    false,
				"assume_unchanged": false,
				"fsmonitor_valid":  false,
			}, map[string]any{
				"path":             "vendor/lib",
				"stage":            json.Number("0"),
				"mode":             json.Number("57344"),
				"oid":              "sha1:" + strings.Repeat("3", 40),
				"intent_to_add":    false,
				"skip_worktree":    false,
				"assume_unchanged": false,
				"fsmonitor_valid":  false,
			}},
			"entry_count": json.Number("2"),
		},
		"working_tree_manifest_id": digestWithDigit('7'),
		"submodules": []any{map[string]any{
			"path":                       "vendor/lib",
			"repository_identity":        "relux/lib",
			"sanitized_url":              "ssh://git@github.com/relux/lib.git",
			"gitlink_oid":                "sha1:" + strings.Repeat("3", 40),
			"initialized":                false,
			"head":                       nil,
			"upstream_ref":               nil,
			"object_pack":                nil,
			"index":                      nil,
			"working_tree_manifest_id":   nil,
			"submodules":                 nil,
			"features":                   nil,
			"repo_relative_cwd":          nil,
			"agent_project_config_paths": nil,
		}},
		"features": map[string]any{
			"object_format":                      "sha1",
			"filemode":                           true,
			"symlinks":                           true,
			"case_sensitive":                     true,
			"precompose_unicode":                 false,
			"sparse_checkout":                    false,
			"sparse_patterns_blob_id":            nil,
			"sparse_patterns_blob_descriptor_id": nil,
			"required_filter_names":              []any{},
			"lfs_required":                       false,
		},
		"repo_relative_cwd":          ".",
		"agent_project_config_paths": []any{},
		"materialization_policy":     "separate_worktree",
	}}
	return object
}

func validInitializedSubmoduleShape(level, maximumDepth int) map[string]any {
	pathValue := "vendor/lib"
	if level > 1 {
		pathValue = fmt.Sprintf("nested-%02d", level)
	}
	entries := []any{}
	submodules := []any{}
	if level < maximumDepth {
		nextPath := fmt.Sprintf("nested-%02d", level+1)
		entries = []any{map[string]any{
			"path": nextPath, "stage": json.Number("0"), "mode": json.Number("57344"),
			"oid": "sha1:" + strings.Repeat("3", 40), "intent_to_add": false,
			"skip_worktree": false, "assume_unchanged": false, "fsmonitor_valid": false,
		}}
		submodules = []any{validInitializedSubmoduleShape(level+1, maximumDepth)}
	}
	return map[string]any{
		"path":                pathValue,
		"repository_identity": fmt.Sprintf("relux/sub-%02d", level),
		"sanitized_url":       fmt.Sprintf("https://example.com/sub-%02d.git", level),
		"gitlink_oid":         "sha1:" + strings.Repeat("3", 40),
		"initialized":         true,
		"head": map[string]any{
			"mode": "detached", "oid": "sha1:" + strings.Repeat("3", 40), "ref": nil,
		},
		"upstream_ref": nil,
		"object_pack": map[string]any{
			"format": "git_pack_v2", "object_format": "sha1", "blob_id": digestWithDigit('1'),
			"blob_descriptor_id": digestWithDigit('2'), "object_count": json.Number("1"),
			"inventory_blob_id": digestWithDigit('3'), "inventory_blob_descriptor_id": digestWithDigit('4'),
		},
		"index": map[string]any{
			"format": "git_index", "version": json.Number("2"), "blob_id": digestWithDigit('5'),
			"blob_descriptor_id": digestWithDigit('6'), "entries": entries, "entry_count": json.Number(strconv.Itoa(len(entries))),
		},
		"working_tree_manifest_id": digestWithDigit('7'),
		"submodules":               submodules,
		"features": map[string]any{
			"object_format": "sha1", "filemode": true, "symlinks": true, "case_sensitive": true,
			"precompose_unicode": false, "sparse_checkout": false, "sparse_patterns_blob_id": nil,
			"sparse_patterns_blob_descriptor_id": nil, "required_filter_names": []any{}, "lfs_required": false,
		},
		"repo_relative_cwd":          ".",
		"agent_project_config_paths": []any{},
	}
}

func configureSubmoduleForest(member map[string]any, addNested bool) {
	entries := make([]any, 0, 256)
	submodules := make([]any, 0, 256)
	for index := range 256 {
		pathValue := fmt.Sprintf("modules/%03d", index)
		entries = append(entries, map[string]any{
			"path": pathValue, "stage": json.Number("0"), "mode": json.Number("57344"),
			"oid": "sha1:" + strings.Repeat("3", 40), "intent_to_add": false,
			"skip_worktree": false, "assume_unchanged": false, "fsmonitor_valid": false,
		})
		submodules = append(submodules, map[string]any{
			"path": pathValue, "repository_identity": fmt.Sprintf("relux/module-%03d", index),
			"sanitized_url": fmt.Sprintf("https://example.com/module-%03d.git", index),
			"gitlink_oid":   "sha1:" + strings.Repeat("3", 40), "initialized": false,
			"head": nil, "upstream_ref": nil, "object_pack": nil, "index": nil,
			"working_tree_manifest_id": nil, "submodules": nil, "features": nil,
			"repo_relative_cwd": nil, "agent_project_config_paths": nil,
		})
	}
	if addNested {
		initialized := validInitializedSubmoduleShape(1, 2)
		initialized["path"] = "modules/000"
		initialized["repository_identity"] = "relux/module-000"
		submodules[0] = initialized
	}
	index := member["index"].(map[string]any)
	index["entries"] = entries
	index["entry_count"] = json.Number("256")
	member["submodules"] = submodules
}

func gitWorkspaceMember(object map[string]any) map[string]any {
	snapshot := object["workspace_snapshot"].(map[string]any)
	return snapshot["members"].([]any)[0].(map[string]any)
}

func firstChunk(object map[string]any) map[string]any {
	return object["chunks"].([]any)[0].(map[string]any)
}

func digestWithDigit(digit byte) string {
	return "sha256:" + strings.Repeat(string(digit), 64)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func cloneJSONObject(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	decoded, err := decodeStrict(mustJSON(t, value))
	if err != nil {
		t.Fatal(err)
	}
	return decoded.(map[string]any)
}

func omitSelfDigestForTest(t *testing.T, input []byte, selfField SelfField) scalar.Digest {
	t.Helper()
	decoded, err := decodeStrict(input)
	if err != nil {
		t.Fatal(err)
	}
	object := decoded.(map[string]any)
	delete(object, string(selfField))
	canonical, err := Canonicalize(mustJSON(t, object))
	if err != nil {
		t.Fatal(err)
	}
	return scalar.SHA256Digest(canonical)
}

func withCorrectIdentityClaimForTest(t *testing.T, input []byte, selfField SelfField) []byte {
	t.Helper()
	decoded, err := decodeStrict(input)
	if err != nil {
		t.Fatal(err)
	}
	object := decoded.(map[string]any)
	digest := omitSelfDigestForTest(t, input, selfField)
	object[string(selfField)] = digest.String()
	return mustJSON(t, object)
}

func digestHex(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}
