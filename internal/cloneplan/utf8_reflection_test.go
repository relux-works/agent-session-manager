package cloneplan_test

import (
	"bytes"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// TestUTF8RefusalReflectionGrid pins that every string-valued
// member of every shape this leaf builds or decodes refuses invalid
// UTF-8 at both production entries with ErrInvalid and a pinned
// literal detail.
//
// The member set is DERIVED from the Go input types by reflection,
// not typed by hand: a new string field is covered automatically
// (its derived member has no probe, so this test fails naming it),
// and a new field of an unhandled kind fails loudly in the walker,
// so the set can never silently shrink. Decode refuses at the
// delegated strict frame (landed environ owner) before any member
// gate runs, so every decode cell pins the frame literal; planting
// per member proves no member bypasses the frame gate.
func TestUTF8RefusalReflectionGrid(t *testing.T) {
	members := derivePlanUTF8Members(t)
	planProbes, manifestProbes := planUTF8Probes(t)
	probes := map[string]utf8PlanProbe{}
	for member, probe := range planProbes {
		probes[member] = probe
	}
	for member, probe := range manifestProbes {
		probes[member] = probe
	}
	for _, member := range members {
		if _, ok := probes[member]; !ok {
			t.Fatalf("derived member %q has no probe: add a build setter, a decode anchor, and build literals", member)
		}
	}
	for member := range probes {
		found := false
		for _, derived := range members {
			if derived == member {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("orphan probe %q matches no derived member: the walker or the probe is stale", member)
		}
	}
	t.Logf("derived UTF-8 member set (%d): %s", len(members), strings.Join(members, ", "))
	classes := []struct {
		name string
		raw  []byte
	}{
		{"lone-continuation", []byte{0x80}},
		{"ff-fe", []byte{0xff, 0xfe}},
		{"overlong", []byte{0xc0, 0xaf}},
		{"encoded-surrogate", []byte{0xed, 0xa0, 0x80}},
		{"truncated-sequence", []byte{0xe2, 0x82}},
	}
	for _, class := range classes {
		if utf8.Valid(class.raw) {
			t.Fatalf("class %q is valid UTF-8: the test vector is wrong", class.name)
		}
	}
	planSealed := mustBuildPlan(t, utf8PlanBaseline(t))
	manifestSealed := mustBuildManifest(t, utf8ManifestBaseline(t))
	for _, member := range members {
		probe := probes[member]
		t.Run(member, func(t *testing.T) {
			for _, class := range classes {
				t.Run(class.name+"/build", func(t *testing.T) {
					probe.buildRefuses(t, string(class.raw))
				})
				t.Run(class.name+"/decode", func(t *testing.T) {
					sealed := planSealed
					decode := func(raw []byte) error {
						_, err := cloneplan.DecodeProjectionPlan(raw)
						return err
					}
					owner := "projection plan"
					if probe.manifest {
						sealed = manifestSealed
						decode = func(raw []byte) error {
							_, err := cloneplan.DecodeProjectedObjectManifest(raw)
							return err
						}
						owner = "projected object manifest"
					}
					mutated := splicePlanUTF8Bytes(t, sealed, probe.anchor, probe.value, class.raw)
					err := decode(mutated)
					requireRefusalTokens(t, err, owner, "not valid UTF-8", "(frame)")
				})
			}
		})
	}
}

// TestRegistryPredicatesRefuseInvalidUTF8 pins the remaining public
// entries: the five string-taking registry predicates admit no
// invalid-UTF-8 token (they return false; there is no error to
// carry ErrInvalid through a boolean entry).
func TestRegistryPredicatesRefuseInvalidUTF8(t *testing.T) {
	bads := []string{
		string([]byte{0x80}),
		string([]byte{0xff, 0xfe}),
		string([]byte{0xc0, 0xaf}),
		string([]byte{0xed, 0xa0, 0x80}),
		string([]byte{0xe2, 0x82}),
	}
	for _, bad := range bads {
		if utf8.ValidString(bad) {
			t.Fatalf("vector %q is valid UTF-8: the test vector is wrong", bad)
		}
		if cloneplan.ValidPlanStrategy(bad) {
			t.Fatalf("ValidPlanStrategy(%q) = true, want false", bad)
		}
		if cloneplan.ValidPlanProfile(bad) {
			t.Fatalf("ValidPlanProfile(%q) = true, want false", bad)
		}
		if cloneplan.ValidOperationAction(bad) {
			t.Fatalf("ValidOperationAction(%q) = true, want false", bad)
		}
		if cloneplan.ValidSynthesizedPurpose(bad) {
			t.Fatalf("ValidSynthesizedPurpose(%q) = true, want false", bad)
		}
		if cloneplan.ValidResourceKind(bad) {
			t.Fatalf("ValidResourceKind(%q) = true, want false", bad)
		}
	}
}

func requireRefusalTokens(t *testing.T, err error, tokens ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want refusal containing %q", tokens)
	}
	if !errors.Is(err, cloneplan.ErrInvalid) {
		t.Fatalf("error = %v, want errors.Is ErrInvalid", err)
	}
	for _, token := range tokens {
		if !strings.Contains(err.Error(), token) {
			t.Fatalf("error = %q, want literal %q", err.Error(), token)
		}
	}
}

// derivePlanUTF8Members walks both input types by reflection and
// returns the sorted set of string-carrying member IDs. Any field
// the walker cannot classify fails the test: the set grows loudly,
// it never shrinks silently.
func derivePlanUTF8Members(t *testing.T) []string {
	t.Helper()
	var members []string
	walkPlanUTF8Fields(t, reflect.TypeOf(cloneplan.ProjectionPlanInput{}), "plan", &members)
	walkPlanUTF8Fields(t, reflect.TypeOf(cloneplan.ProjectedManifestInput{}), "manifest", &members)
	sort.Strings(members)
	seen := map[string]bool{}
	for _, member := range members {
		if seen[member] {
			t.Fatalf("derived member %q twice", member)
		}
		seen[member] = true
	}
	if len(members) == 0 {
		t.Fatal("derived UTF-8 member set is empty")
	}
	return members
}

func walkPlanUTF8Fields(t *testing.T, typ reflect.Type, path string, out *[]string) {
	t.Helper()
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		walkPlanUTF8Field(t, field, path+"."+field.Name, out)
	}
}

func walkPlanUTF8Field(t *testing.T, field reflect.StructField, path string, out *[]string) {
	t.Helper()
	if field.PkgPath != "" {
		t.Fatalf("unexported field at %s: extend the walker before probing it", path)
	}
	emit := func(id string) { *out = append(*out, id) }
	fieldType := field.Type
	switch fieldType.Kind() {
	case reflect.String:
		emit(path)
	case reflect.Pointer:
		switch fieldType.Elem().Kind() {
		case reflect.String:
			emit(path)
		case reflect.Uint64:
			// Numeric optionals carry no strings.
		default:
			t.Fatalf("unhandled pointer at %s: extend the walker", path)
		}
	case reflect.Slice, reflect.Array:
		element := fieldType.Elem()
		switch element.Kind() {
		case reflect.Uint8:
			emit(path)
		case reflect.String:
			emit(path + "[]")
		case reflect.Uint64:
			// Numeric arrays carry no strings.
		case reflect.Struct:
			for index := 0; index < element.NumField(); index++ {
				sub := element.Field(index)
				walkPlanUTF8Field(t, sub, path+"[]."+sub.Name, out)
			}
		default:
			t.Fatalf("unhandled slice element kind %s at %s: extend the walker", element.Kind(), path)
		}
	case reflect.Map:
		if fieldType.Key().Kind() != reflect.String {
			t.Fatalf("unhandled map key kind %s at %s: extend the walker", fieldType.Key().Kind(), path)
		}
		value := fieldType.Elem()
		switch value.Kind() {
		case reflect.Slice:
			if value.Elem().Kind() != reflect.String {
				t.Fatalf("unhandled map slice element kind %s at %s: extend the walker", value.Elem().Kind(), path)
			}
			emit(path + "<key>")
			emit(path + "[]")
		case reflect.Interface:
			if value.NumMethod() != 0 {
				t.Fatalf("unhandled map interface at %s: extend the walker", path)
			}
			emit(path + "<key>")
			emit(path + "<value>")
		default:
			t.Fatalf("unhandled map value kind %s at %s: extend the walker", value.Kind(), path)
		}
	case reflect.Struct:
		walkPlanUTF8Fields(t, fieldType, path, out)
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// Scalar non-text fields carry no strings.
	default:
		t.Fatalf("unhandled kind %s at %s: extend the walker", fieldType.Kind(), path)
	}
}

// utf8PlanProbe drives one derived member at both entries:
// buildRefuses plants the bad string in a fresh baseline input and
// asserts the member-specific refusal; anchor/value locate the
// sealed bytes to splice for the decode plant.
type utf8PlanProbe struct {
	manifest     bool
	buildRefuses func(t *testing.T, bad string)
	anchor       []byte
	value        []byte
}

func utf8TaggedTuple(envID, platform, seed string) string {
	return `{"environment_id":"` + envID + `","environment_version":"2.1.0","platform":"` +
		platform + `","architecture":"amd64","store_schema_fingerprint":"` +
		fixtureDigest(seed) + `","adapter_version":"1.2.3"}`
}

// utf8PlanBaseline is the valid plan every grid cell starts from:
// every probed value is unique within the sealed document so every
// decode anchor pins its member.
func utf8PlanBaseline(t *testing.T) cloneplan.ProjectionPlanInput {
	t.Helper()
	input := validPlanInput()
	input.SourceEnvironment = []byte(utf8TaggedTuple("test.env.utf8source", "linux", "utf8-source-fingerprint"))
	input.TargetEnvironment = []byte(utf8TaggedTuple("test.env.utf8target", "macos", "utf8-target-fingerprint"))
	input.ExpectedTargetNativeSession = "utf8-target-native"
	input.CanonicalEventIDs = []string{fixtureDigest("utf8-ev")}
	input.RequiredDispositions = map[string][]string{"utf8dispositionclass": {"exact"}}
	input.ForbidReasons = []string{"utf8forbid"}
	mapping := validMappingInput("utf8-mapping")
	mapping.CanonicalObjectID = strptr(fixtureDigest("canonical-utf8-mapping"))
	mapping.TargetResourceKey = []string{"utf8/mapping"}
	mapping.ReasonCodes = []string{"unknown_native_event"}
	mapping.Extensions = map[string]any{"com.example.utf8mapping": "utf8mappingvalue"}
	input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
	operation := validOperationInput(1)
	operation.ResourceKeys = []string{"utf8/operation"}
	operation.Extensions = map[string]any{"com.example.utf8operation": "utf8operationvalue"}
	input.TargetOperations = []cloneplan.TargetOperationInput{operation}
	resource := validResourceInput(1, "utf8-resource")
	resource.ExpectedBlobID = strptr(fixtureDigest("blob-utf8-resource"))
	resource.Extensions = map[string]any{"com.example.utf8resource": "utf8resourcevalue"}
	input.ExpectedResources = []cloneplan.ExpectedResourceInput{resource}
	event := validSynthEventInput("utf8")
	event.Extensions = map[string]any{"com.example.utf8event": "utf8eventvalue"}
	input.SynthesizedEvents = []cloneplan.SynthesizedEventInput{event}
	input.SecurityExclusions = []string{"credential"}
	contract := validContractInput("ax.contract.utf8")
	contract.Version = "7.8.9"
	input.RequiredContracts = []cloneplan.ContractRequirementInput{contract}
	input.RequiredCapabilities = []string{"utf8.capability"}
	input.Extensions = map[string]any{"com.example.utf8plan": "utf8planvalue"}
	return input
}

func utf8ManifestBaseline(t *testing.T) cloneplan.ProjectedManifestInput {
	t.Helper()
	manifest := validManifestInput()
	manifest.TargetEnvironment = []byte(utf8TaggedTuple("test.env.utf8manifest", "windows", "utf8-manifest-fingerprint"))
	manifest.ExpectedTargetNativeSession = "utf8-manifest-native"
	directory := validDirectoryEntryInput(1, "utf8-adir")
	blob := validBlobEntryInput(1, "utf8-bblob")
	blob.BlobID = strptr(fixtureDigest("blob-utf8-bblob"))
	blob.BlobDescriptorID = strptr(fixtureDigest("descriptor-utf8-bblob"))
	manifest.Entries = []cloneplan.ProjectedEntryInput{directory, blob}
	manifest.Extensions = map[string]any{"com.example.utf8manifest": "utf8manifestvalue"}
	return manifest
}

func planUTF8Probes(t *testing.T) (map[string]utf8PlanProbe, map[string]utf8PlanProbe) {
	t.Helper()
	plan := map[string]utf8PlanProbe{}
	manifest := map[string]utf8PlanProbe{}
	baseline := utf8PlanBaseline(t)
	_ = baseline

	strField := func(path, member, token string, set func(*cloneplan.ProjectionPlanInput, string), anchor, value string) {
		plan[path] = utf8PlanProbe{
			buildRefuses: func(t *testing.T, bad string) {
				input := utf8PlanBaseline(t)
				set(&input, bad)
				_, err := cloneplan.BuildProjectionPlan(input)
				requireRefusalTokens(t, err, token)
			},
			anchor: []byte(anchor),
			value:  []byte(value),
		}
		_ = member
	}
	_ = strField

	// Scalar plan members.
	scalars := []struct {
		path, member, token, anchor, value string
		set                                func(*cloneplan.ProjectionPlanInput, string)
	}{
		{"plan.OperationID", "operation_id", "operation_id is not a UUIDv7",
			`"operation_id":"` + fixtureOperationID + `"`, fixtureOperationID,
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.OperationID = bad }},
		{"plan.BundleID", "bundle_id", "bundle_id is not a UUIDv7",
			`"bundle_id":"` + fixtureBundleID + `"`, fixtureBundleID,
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.BundleID = bad }},
		{"plan.RequestDigest", "request_digest", "request_digest is not a digest",
			`"request_digest":"` + fixtureDigest("request") + `"`, fixtureDigest("request"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.RequestDigest = bad }},
		{"plan.SourceSnapshotDigest", "source_snapshot_digest", "source_snapshot_digest is not a digest",
			`"source_snapshot_digest":"` + fixtureDigest("snapshot") + `"`, fixtureDigest("snapshot"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.SourceSnapshotDigest = bad }},
		{"plan.CaptureManifestID", "capture_manifest_id", "capture_manifest_id is not a digest",
			`"capture_manifest_id":"` + fixtureDigest("capture") + `"`, fixtureDigest("capture"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.CaptureManifestID = bad }},
		{"plan.CanonicalSessionID", "canonical_session_id", "canonical_session_id is not a digest",
			`"canonical_session_id":"` + fixtureDigest("canonical") + `"`, fixtureDigest("canonical"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.CanonicalSessionID = bad }},
		{"plan.CanonicalEventIDs[]", "canonical_event_ids", "canonical_event_ids[0] is not a digest",
			`"canonical_event_ids":["` + fixtureDigest("utf8-ev") + `"]`, fixtureDigest("utf8-ev"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.CanonicalEventIDs = []string{bad} }},
		{"plan.ExpectedTargetNativeSession", "expected_target_native_session_id", "expected_target_native_session_id is not valid UTF-8",
			`"expected_target_native_session_id":"utf8-target-native"`, "utf8-target-native",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.ExpectedTargetNativeSession = bad }},
		{"plan.Strategy", "strategy", "strategy is not valid UTF-8",
			`"strategy":"target_native_writer"`, "target_native_writer",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.Strategy = bad }},
		{"plan.StrategyRationale", "strategy_rationale", "strategy_rationale is not valid UTF-8",
			`"strategy_rationale":"native writer covers the target tuple"`, "native writer covers the target tuple",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.StrategyRationale = bad }},
		{"plan.FidelityProfile", "fidelity_profile", "fidelity_profile is not valid UTF-8",
			`"fidelity_profile":"maximal_safe"`, "maximal_safe",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.FidelityProfile = bad }},
		{"plan.RequiredDispositions<key>", "required_dispositions", "carry a class that is not valid UTF-8",
			`"required_dispositions":{"utf8dispositionclass":["exact"]}`, "utf8dispositionclass",
			func(input *cloneplan.ProjectionPlanInput, bad string) {
				input.RequiredDispositions = map[string][]string{bad: {"exact"}}
			}},
		{"plan.RequiredDispositions[]", "required_dispositions", "is not valid UTF-8",
			`"required_dispositions":{"utf8dispositionclass":["exact"]}`, `"exact"`,
			func(input *cloneplan.ProjectionPlanInput, bad string) {
				input.RequiredDispositions = map[string][]string{"utf8dispositionclass": {bad}}
			}},
		{"plan.ForbidReasons[]", "forbid_reasons", "forbid_reasons[0] is not valid UTF-8",
			`"forbid_reasons":["utf8forbid"]`, "utf8forbid",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.ForbidReasons = []string{bad} }},
		{"plan.SecurityExclusions[]", "security_exclusions", "security_exclusions[0] is not valid UTF-8",
			`"security_exclusions":["credential"]`, "credential",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.SecurityExclusions = []string{bad} }},
		{"plan.RequiredCapabilities[]", "required_capabilities", "required_capabilities[0] is not valid UTF-8",
			`"required_capabilities":["utf8.capability"]`, "utf8.capability",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.RequiredCapabilities = []string{bad} }},
		{"plan.FidelityBasisDigest", "fidelity_basis_digest", "fidelity_basis_digest is not a digest",
			`"fidelity_basis_digest":"` + fixtureDigest("basis") + `"`, fixtureDigest("basis"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.FidelityBasisDigest = bad }},
		{"plan.SourceAdapterBuildDigest", "source_adapter_build_digest", "source_adapter_build_digest is not a digest",
			`"source_adapter_build_digest":"` + fixtureDigest("source-build") + `"`, fixtureDigest("source-build"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.SourceAdapterBuildDigest = bad }},
		{"plan.TargetAdapterBuildDigest", "target_adapter_build_digest", "target_adapter_build_digest is not a digest",
			`"target_adapter_build_digest":"` + fixtureDigest("target-build") + `"`, fixtureDigest("target-build"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.TargetAdapterBuildDigest = bad }},
		{"plan.ControllerBuildDigest", "controller_build_digest", "controller_build_digest is not a digest",
			`"controller_build_digest":"` + fixtureDigest("controller-build") + `"`, fixtureDigest("controller-build"),
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.ControllerBuildDigest = bad }},
		{"plan.Extensions<key>", "extensions", "extensions invalid",
			`"extensions":{"com.example.utf8plan":"utf8planvalue"}`, "com.example.utf8plan",
			func(input *cloneplan.ProjectionPlanInput, bad string) {
				input.Extensions = map[string]any{bad: "utf8planvalue"}
			}},
		{"plan.Extensions<value>", "extensions", "extensions invalid",
			`"extensions":{"com.example.utf8plan":"utf8planvalue"}`, "utf8planvalue",
			func(input *cloneplan.ProjectionPlanInput, bad string) {
				input.Extensions = map[string]any{"com.example.utf8plan": bad}
			}},
	}
	for _, row := range scalars {
		plan[row.path] = utf8PlanProbe{
			buildRefuses: func(set func(*cloneplan.ProjectionPlanInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8PlanBaseline(t)
					set(&input, bad)
					_, err := cloneplan.BuildProjectionPlan(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}

	// Raw-JSON plan members: build plants the bad bytes whole; decode
	// splices inside the re-emitted member.
	raws := []struct {
		path, token, anchor, value string
		set                        func(*cloneplan.ProjectionPlanInput, string)
	}{
		{"plan.SourceEnvironment", "source_environment is not an Environment Tuple",
			`"source_environment":{"adapter_version":"1.2.3","architecture":"amd64","environment_id":"test.env.utf8source"`, "test.env.utf8source",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.SourceEnvironment = []byte(bad) }},
		{"plan.TargetEnvironment", "target_environment is not an Environment Tuple",
			`"target_environment":{"adapter_version":"1.2.3","architecture":"amd64","environment_id":"test.env.utf8target"`, "test.env.utf8target",
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.TargetEnvironment = []byte(bad) }},
		{"plan.TargetWorkspace", "target_workspace is not a Workspace Binding",
			`"target_workspace":{"branch":null`, `"branch":null`,
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.TargetWorkspace = []byte(bad) }},
		{"plan.ResourceLimits", "resource_limits are not Resource Limits",
			`"resource_limits":{"max_events":100`, `"max_events":100`,
			func(input *cloneplan.ProjectionPlanInput, bad string) { input.ResourceLimits = []byte(bad) }},
	}
	for _, row := range raws {
		plan[row.path] = utf8PlanProbe{
			buildRefuses: func(set func(*cloneplan.ProjectionPlanInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8PlanBaseline(t)
					set(&input, bad)
					_, err := cloneplan.BuildProjectionPlan(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}

	// Row members: one setter per row shape.
	mappingRows := []struct {
		path, token, anchor, value string
		set                        func(*cloneplan.ItemMappingInput, string)
	}{
		{"plan.ItemMappings[].SourceItemKey", "source_item_key is not valid UTF-8",
			`"source_item_key":"utf8-mapping"`, "utf8-mapping",
			func(row *cloneplan.ItemMappingInput, bad string) { row.SourceItemKey = bad }},
		{"plan.ItemMappings[].CanonicalObjectID", "canonical_object_id is not a digest",
			`"canonical_object_id":"` + fixtureDigest("canonical-utf8-mapping") + `"`, fixtureDigest("canonical-utf8-mapping"),
			func(row *cloneplan.ItemMappingInput, bad string) { row.CanonicalObjectID = &bad }},
		{"plan.ItemMappings[].TargetResourceKey[]", "target_resource_keys[0] is not valid UTF-8",
			`"target_resource_keys":["utf8/mapping"]`, "utf8/mapping",
			func(row *cloneplan.ItemMappingInput, bad string) { row.TargetResourceKey = []string{bad} }},
		{"plan.ItemMappings[].ExpectedDispos", "expected_disposition is not valid UTF-8",
			`"expected_disposition":"exact"`, `"expected_disposition":"exact"`,
			func(row *cloneplan.ItemMappingInput, bad string) { row.ExpectedDispos = bad }},
		{"plan.ItemMappings[].ReasonCodes[]", "reason_codes[0] is not valid UTF-8",
			`"reason_codes":["unknown_native_event"]`, "unknown_native_event",
			func(row *cloneplan.ItemMappingInput, bad string) { row.ReasonCodes = []string{bad} }},
		{"plan.ItemMappings[].Extensions<key>", "extensions invalid",
			`"com.example.utf8mapping":"utf8mappingvalue"`, "com.example.utf8mapping",
			func(row *cloneplan.ItemMappingInput, bad string) {
				row.Extensions = map[string]any{bad: "utf8mappingvalue"}
			}},
		{"plan.ItemMappings[].Extensions<value>", "extensions invalid",
			`"com.example.utf8mapping":"utf8mappingvalue"`, "utf8mappingvalue",
			func(row *cloneplan.ItemMappingInput, bad string) {
				row.Extensions = map[string]any{"com.example.utf8mapping": bad}
			}},
	}
	for _, row := range mappingRows {
		plan[row.path] = utf8PlanProbe{
			buildRefuses: func(set func(*cloneplan.ItemMappingInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8PlanBaseline(t)
					mapping := input.ItemMappings[0]
					set(&mapping, bad)
					input.ItemMappings = []cloneplan.ItemMappingInput{mapping}
					_, err := cloneplan.BuildProjectionPlan(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}

	operationRows := []struct {
		path, token, anchor, value string
		set                        func(*cloneplan.TargetOperationInput, string)
	}{
		{"plan.TargetOperations[].Action", "action is not valid UTF-8",
			`"action":"write_blob"`, `"action":"write_blob"`,
			func(row *cloneplan.TargetOperationInput, bad string) { row.Action = bad }},
		{"plan.TargetOperations[].ResourceKeys[]", "resource_keys[0] is not valid UTF-8",
			`"resource_keys":["utf8/operation"]`, "utf8/operation",
			func(row *cloneplan.TargetOperationInput, bad string) { row.ResourceKeys = []string{bad} }},
		{"plan.TargetOperations[].Extensions<key>", "extensions invalid",
			`"com.example.utf8operation":"utf8operationvalue"`, "com.example.utf8operation",
			func(row *cloneplan.TargetOperationInput, bad string) {
				row.Extensions = map[string]any{bad: "utf8operationvalue"}
			}},
		{"plan.TargetOperations[].Extensions<value>", "extensions invalid",
			`"com.example.utf8operation":"utf8operationvalue"`, "utf8operationvalue",
			func(row *cloneplan.TargetOperationInput, bad string) {
				row.Extensions = map[string]any{"com.example.utf8operation": bad}
			}},
	}
	for _, row := range operationRows {
		plan[row.path] = utf8PlanProbe{
			buildRefuses: func(set func(*cloneplan.TargetOperationInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8PlanBaseline(t)
					operation := input.TargetOperations[0]
					set(&operation, bad)
					input.TargetOperations = []cloneplan.TargetOperationInput{operation}
					_, err := cloneplan.BuildProjectionPlan(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}

	resourceRows := []struct {
		path, token, anchor, value string
		set                        func(*cloneplan.ExpectedResourceInput, string)
	}{
		{"plan.ExpectedResources[].ResourceKey", "resource_key is not valid UTF-8",
			`"resource_key":"utf8-resource"`, "utf8-resource",
			func(row *cloneplan.ExpectedResourceInput, bad string) { row.ResourceKey = bad }},
		{"plan.ExpectedResources[].Kind", "kind is not valid UTF-8",
			`"kind":"blob"`, `"kind":"blob"`,
			func(row *cloneplan.ExpectedResourceInput, bad string) { row.Kind = bad }},
		{"plan.ExpectedResources[].ExpectedBlobID", "expected_blob_id is not a digest",
			`"expected_blob_id":"` + fixtureDigest("blob-utf8-resource") + `"`, fixtureDigest("blob-utf8-resource"),
			func(row *cloneplan.ExpectedResourceInput, bad string) { row.ExpectedBlobID = &bad }},
		{"plan.ExpectedResources[].Extensions<key>", "extensions invalid",
			`"com.example.utf8resource":"utf8resourcevalue"`, "com.example.utf8resource",
			func(row *cloneplan.ExpectedResourceInput, bad string) {
				row.Extensions = map[string]any{bad: "utf8resourcevalue"}
			}},
		{"plan.ExpectedResources[].Extensions<value>", "extensions invalid",
			`"com.example.utf8resource":"utf8resourcevalue"`, "utf8resourcevalue",
			func(row *cloneplan.ExpectedResourceInput, bad string) {
				row.Extensions = map[string]any{"com.example.utf8resource": bad}
			}},
	}
	for _, row := range resourceRows {
		plan[row.path] = utf8PlanProbe{
			buildRefuses: func(set func(*cloneplan.ExpectedResourceInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8PlanBaseline(t)
					resource := input.ExpectedResources[0]
					set(&resource, bad)
					input.ExpectedResources = []cloneplan.ExpectedResourceInput{resource}
					_, err := cloneplan.BuildProjectionPlan(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}

	eventRows := []struct {
		path, token, anchor, value string
		set                        func(*cloneplan.SynthesizedEventInput, string)
	}{
		{"plan.SynthesizedEvents[].CanonicalEventID", "canonical_event_id is not a digest",
			`"canonical_event_id":"` + fixtureDigest("synth-utf8") + `"`, fixtureDigest("synth-utf8"),
			func(row *cloneplan.SynthesizedEventInput, bad string) { row.CanonicalEventID = bad }},
		{"plan.SynthesizedEvents[].InsertionAfterEvent", "insertion_after_event_id is not a digest",
			`"insertion_after_event_id":"` + fixtureDigest("anchor-utf8") + `"`, fixtureDigest("anchor-utf8"),
			func(row *cloneplan.SynthesizedEventInput, bad string) { row.InsertionAfterEvent = &bad }},
		{"plan.SynthesizedEvents[].Purpose", "purpose is not valid UTF-8",
			`"purpose":"summary"`, `"purpose":"summary"`,
			func(row *cloneplan.SynthesizedEventInput, bad string) { row.Purpose = bad }},
		{"plan.SynthesizedEvents[].Extensions<key>", "extensions invalid",
			`"com.example.utf8event":"utf8eventvalue"`, "com.example.utf8event",
			func(row *cloneplan.SynthesizedEventInput, bad string) {
				row.Extensions = map[string]any{bad: "utf8eventvalue"}
			}},
		{"plan.SynthesizedEvents[].Extensions<value>", "extensions invalid",
			`"com.example.utf8event":"utf8eventvalue"`, "utf8eventvalue",
			func(row *cloneplan.SynthesizedEventInput, bad string) {
				row.Extensions = map[string]any{"com.example.utf8event": bad}
			}},
	}
	for _, row := range eventRows {
		plan[row.path] = utf8PlanProbe{
			buildRefuses: func(set func(*cloneplan.SynthesizedEventInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8PlanBaseline(t)
					event := input.SynthesizedEvents[0]
					set(&event, bad)
					input.SynthesizedEvents = []cloneplan.SynthesizedEventInput{event}
					_, err := cloneplan.BuildProjectionPlan(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}

	contractRows := []struct {
		path, token, anchor, value string
		set                        func(*cloneplan.ContractRequirementInput, string)
	}{
		{"plan.RequiredContracts[].ContractID", "contract_id is not valid UTF-8",
			`"contract_id":"ax.contract.utf8"`, "ax.contract.utf8",
			func(row *cloneplan.ContractRequirementInput, bad string) { row.ContractID = bad }},
		{"plan.RequiredContracts[].Version", "version is not valid UTF-8",
			`"version":"7.8.9"`, "7.8.9",
			func(row *cloneplan.ContractRequirementInput, bad string) { row.Version = bad }},
	}
	for _, row := range contractRows {
		plan[row.path] = utf8PlanProbe{
			buildRefuses: func(set func(*cloneplan.ContractRequirementInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8PlanBaseline(t)
					contract := input.RequiredContracts[0]
					set(&contract, bad)
					input.RequiredContracts = []cloneplan.ContractRequirementInput{contract}
					_, err := cloneplan.BuildProjectionPlan(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}

	// Closed plan components.
	plan["plan.TransactionPlan.MaterializationIntent"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.TransactionPlan.MaterializationIntent = bad
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "materialization_intent is not valid UTF-8")
		},
		anchor: []byte(`"materialization_intent":"clone"`),
		value:  []byte(`"clone"`),
	}
	plan["plan.TransactionPlan.TargetCollisionPolicy"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.TransactionPlan.TargetCollisionPolicy = bad
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "target_collision_policy is not valid UTF-8")
		},
		anchor: []byte(`"target_collision_policy":"must_be_absent"`),
		value:  []byte(`"must_be_absent"`),
	}
	plan["plan.TransactionPlan.Activation"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.TransactionPlan.Activation = bad
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "activation is not valid UTF-8")
		},
		anchor: []byte(`"activation":"dormant_validated"`),
		value:  []byte(`"dormant_validated"`),
	}
	plan["plan.TransactionPlan.Extensions<key>"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.TransactionPlan.Extensions = map[string]any{bad: "x"}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "extensions invalid")
		},
		anchor: []byte(`"transaction_plan":{"activation":"dormant_validated"`),
		value:  []byte(`"dormant_validated"`),
	}
	plan["plan.TransactionPlan.Extensions<value>"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.TransactionPlan.Extensions = map[string]any{"com.example.ok": bad}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "extensions invalid")
		},
		anchor: []byte(`"materialization_intent":"clone"`),
		value:  []byte(`"materialization_intent"`),
	}
	plan["plan.ReadBackPlan.Modes[]"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.ReadBackPlan.Modes = []string{bad, "live"}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "modes carry a member that is not valid UTF-8")
		},
		anchor: []byte(`"modes":["staged","live"]`),
		value:  []byte(`"staged"`),
	}
	plan["plan.ReadBackPlan.Extensions<key>"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.ReadBackPlan.Extensions = map[string]any{bad: "x"}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "extensions invalid")
		},
		anchor: []byte(`"require_identity_match":true`),
		value:  []byte(`"require_identity_match"`),
	}
	plan["plan.ReadBackPlan.Extensions<value>"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.ReadBackPlan.Extensions = map[string]any{"com.example.ok": bad}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "extensions invalid")
		},
		anchor: []byte(`"modes":["staged","live"]`),
		value:  []byte(`"live"`),
	}
	plan["plan.ResumePlan.Extensions<key>"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.ResumePlan.Extensions = map[string]any{bad: "x"}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "extensions invalid")
		},
		anchor: []byte(`"opens_existing_identity":true`),
		value:  []byte(`"opens_existing_identity"`),
	}
	plan["plan.ResumePlan.Extensions<value>"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.ResumePlan.Extensions = map[string]any{"com.example.ok": bad}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "extensions invalid")
		},
		anchor: []byte(`"allow_blank_fallback":false`),
		value:  []byte(`"allow_blank_fallback"`),
	}
	plan["plan.RollbackPlan.RetainThrough"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.RollbackPlan.RetainThrough = bad
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "retain_through is not valid UTF-8")
		},
		anchor: []byte(`"retain_through":"live_validated"`),
		value:  []byte(`"live_validated"`),
	}
	plan["plan.RollbackPlan.Extensions<key>"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.RollbackPlan.Extensions = map[string]any{bad: "x"}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "extensions invalid")
		},
		anchor: []byte(`"retain_through":"live_validated"`),
		value:  []byte(`"retain_through"`),
	}
	plan["plan.RollbackPlan.Extensions<value>"] = utf8PlanProbe{
		buildRefuses: func(t *testing.T, bad string) {
			input := utf8PlanBaseline(t)
			input.RollbackPlan.Extensions = map[string]any{"com.example.ok": bad}
			_, err := cloneplan.BuildProjectionPlan(input)
			requireRefusalTokens(t, err, "extensions invalid")
		},
		anchor: []byte(`"forbidden_after_provider_commit":true`),
		value:  []byte(`"forbidden_after_provider_commit"`),
	}

	// Manifest members.
	manifestScalars := []struct {
		path, token, anchor, value string
		set                        func(*cloneplan.ProjectedManifestInput, string)
	}{
		{"manifest.OperationID", "operation_id is not a UUIDv7",
			`"operation_id":"` + fixtureOperationID + `"`, fixtureOperationID,
			func(input *cloneplan.ProjectedManifestInput, bad string) { input.OperationID = bad }},
		{"manifest.ProjectionPlanID", "projection_plan_id is not a digest",
			`"projection_plan_id":"` + fixtureDigest("plan-id") + `"`, fixtureDigest("plan-id"),
			func(input *cloneplan.ProjectedManifestInput, bad string) { input.ProjectionPlanID = bad }},
		{"manifest.TargetEnvironment", "target_environment is not an Environment Tuple",
			`"target_environment":{"adapter_version":"1.2.3","architecture":"amd64","environment_id":"test.env.utf8manifest"`, "test.env.utf8manifest",
			func(input *cloneplan.ProjectedManifestInput, bad string) { input.TargetEnvironment = []byte(bad) }},
		{"manifest.ExpectedTargetNativeSession", "expected_target_native_session_id is not valid UTF-8",
			`"expected_target_native_session_id":"utf8-manifest-native"`, "utf8-manifest-native",
			func(input *cloneplan.ProjectedManifestInput, bad string) { input.ExpectedTargetNativeSession = bad }},
		{"manifest.Extensions<key>", "extensions invalid",
			`"extensions":{"com.example.utf8manifest":"utf8manifestvalue"}`, "com.example.utf8manifest",
			func(input *cloneplan.ProjectedManifestInput, bad string) {
				input.Extensions = map[string]any{bad: "utf8manifestvalue"}
			}},
		{"manifest.Extensions<value>", "extensions invalid",
			`"extensions":{"com.example.utf8manifest":"utf8manifestvalue"}`, "utf8manifestvalue",
			func(input *cloneplan.ProjectedManifestInput, bad string) {
				input.Extensions = map[string]any{"com.example.utf8manifest": bad}
			}},
	}
	for _, row := range manifestScalars {
		manifest[row.path] = utf8PlanProbe{
			manifest: true,
			buildRefuses: func(set func(*cloneplan.ProjectedManifestInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8ManifestBaseline(t)
					set(&input, bad)
					_, err := cloneplan.BuildProjectedObjectManifest(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}
	entryRows := []struct {
		path, token, anchor, value string
		set                        func(*cloneplan.ProjectedEntryInput, string)
	}{
		{"manifest.Entries[].ResourceKey", "resource_key is not valid UTF-8",
			`"resource_key":"utf8-bblob"`, "utf8-bblob",
			func(row *cloneplan.ProjectedEntryInput, bad string) { row.ResourceKey = bad }},
		{"manifest.Entries[].Kind", "kind is not valid UTF-8",
			`"kind":"blob"`, `"kind":"blob"`,
			func(row *cloneplan.ProjectedEntryInput, bad string) { row.Kind = bad }},
		{"manifest.Entries[].BlobID", "blob_id is not a digest",
			`"blob_id":"` + fixtureDigest("blob-utf8-bblob") + `"`, fixtureDigest("blob-utf8-bblob"),
			func(row *cloneplan.ProjectedEntryInput, bad string) { row.BlobID = &bad }},
		{"manifest.Entries[].BlobDescriptorID", "blob_descriptor_id is not a digest",
			`"blob_descriptor_id":"` + fixtureDigest("descriptor-utf8-bblob") + `"`, fixtureDigest("descriptor-utf8-bblob"),
			func(row *cloneplan.ProjectedEntryInput, bad string) { row.BlobDescriptorID = &bad }},
	}
	for _, row := range entryRows {
		manifest[row.path] = utf8PlanProbe{
			manifest: true,
			buildRefuses: func(set func(*cloneplan.ProjectedEntryInput, string), token string) func(*testing.T, string) {
				return func(t *testing.T, bad string) {
					input := utf8ManifestBaseline(t)
					blob := validBlobEntryInput(1, "utf8-entry")
					set(&blob, bad)
					input.Entries = []cloneplan.ProjectedEntryInput{blob}
					_, err := cloneplan.BuildProjectedObjectManifest(input)
					requireRefusalTokens(t, err, token)
				}
			}(row.set, row.token),
			anchor: []byte(row.anchor),
			value:  []byte(row.value),
		}
	}
	return plan, manifest
}

// splicePlanUTF8Bytes replaces the anchored value with the raw bad
// bytes exactly once. The anchor must occur exactly once in the
// sealed document and must contain the value, so every decode plant
// is pinned to its member.
func splicePlanUTF8Bytes(t *testing.T, sealed, anchor, value, bad []byte) []byte {
	t.Helper()
	if count := bytes.Count(sealed, anchor); count != 1 {
		t.Fatalf("anchor %q occurs %d times, want exactly 1", anchor, count)
	}
	start := bytes.Index(sealed, anchor)
	within := bytes.Index(sealed[start:start+len(anchor)], value)
	if within < 0 {
		t.Fatalf("anchor %q does not contain value %q", anchor, value)
	}
	at := start + within
	out := make([]byte, 0, len(sealed)-len(value)+len(bad))
	out = append(out, sealed[:at]...)
	out = append(out, bad...)
	out = append(out, sealed[at+len(value):]...)
	return out
}
