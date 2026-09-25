package clonefidelity_test

import (
	"bytes"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	clonefidelity "github.com/relux-works/agent-session-manager/internal/clonefidelity"
)

// TestUTF8RefusalReflectionGrid pins the class the rev2 reviewer
// found open (finding unmeasured-build-utf8-sites): every
// string-valued member of every shape this leaf builds or decodes
// refuses invalid UTF-8 at both production entries with ErrInvalid
// and a pinned literal detail.
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
	members := deriveUTF8Members(t)
	probes := utf8Probes(t)
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
	sealed := mustBuild(t, utf8BaselineInput(t))
	for _, member := range members {
		probe := probes[member]
		t.Run(member, func(t *testing.T) {
			for _, class := range classes {
				t.Run(class.name+"/build", func(t *testing.T) {
					input := utf8BaselineInput(t)
					probe.buildSet(&input, string(class.raw))
					_, err := clonefidelity.BuildFidelityReport(input)
					requireRefusalTokens(t, err, probe.buildLiterals...)
				})
				t.Run(class.name+"/decode", func(t *testing.T) {
					mutated := spliceUTF8Bytes(t, sealed, probe.anchor, probe.value, class.raw)
					_, err := clonefidelity.DecodeFidelityReport(mutated)
					requireRefusalTokens(t, err, "fidelity report", "not valid UTF-8", "(frame)")
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
		if clonefidelity.ValidDisposition(bad) {
			t.Fatalf("ValidDisposition(%q) = true, want false", bad)
		}
		if clonefidelity.ValidProfile(bad) {
			t.Fatalf("ValidProfile(%q) = true, want false", bad)
		}
		if clonefidelity.ValidStrategy(bad) {
			t.Fatalf("ValidStrategy(%q) = true, want false", bad)
		}
		if clonefidelity.IsCoreReasonCode(bad) {
			t.Fatalf("IsCoreReasonCode(%q) = true, want false", bad)
		}
		if clonefidelity.ValidReasonCode(bad) {
			t.Fatalf("ValidReasonCode(%q) = true, want false", bad)
		}
	}
}

// TestBuildUTF8ExtensionsNestedDepth pins that the build-side
// extension UTF-8 gate (landed clonebundle owner) fires below the
// top level too: a bad string nested in a map or a slice refuses at
// both extension layers.
func TestBuildUTF8ExtensionsNestedDepth(t *testing.T) {
	bad := string([]byte{0xff, 0xfe})
	nested := map[string]any{"com.example.utf8nested": map[string]any{"inner": bad}}
	sliced := map[string]any{"com.example.utf8sliced": []any{"ok", bad}}
	input := validTargetInput(validRowInput("utf8-nested", "exact"))
	input.Extensions = nested
	_, err := clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "not valid UTF-8")
	input = validTargetInput(validRowInput("utf8-nested", "exact"))
	input.Extensions = sliced
	_, err = clonefidelity.BuildFidelityReport(input)
	requireRefusal(t, err, "not valid UTF-8")
	row := validRowInput("utf8-nested", "exact")
	row.Extensions = nested
	_, err = clonefidelity.BuildFidelityReport(validTargetInput(row))
	requireRefusal(t, err, "not valid UTF-8")
	row = validRowInput("utf8-nested", "exact")
	row.Extensions = sliced
	_, err = clonefidelity.BuildFidelityReport(validTargetInput(row))
	requireRefusal(t, err, "not valid UTF-8")
}

func requireRefusalTokens(t *testing.T, err error, tokens ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want refusal containing %q", tokens)
	}
	if !errors.Is(err, clonefidelity.ErrInvalid) {
		t.Fatalf("error = %v, want errors.Is ErrInvalid", err)
	}
	for _, token := range tokens {
		if !strings.Contains(err.Error(), token) {
			t.Fatalf("error = %q, want literal %q", err.Error(), token)
		}
	}
}

// deriveUTF8Members walks the report input type by reflection and
// returns the sorted set of string-carrying member IDs. The row
// input type is walked standalone too, and its set must equal the
// nested Rows[] set, so the row shape is pinned once. Any field the
// walker cannot classify fails the test: the set grows loudly, it
// never shrinks silently.
func deriveUTF8Members(t *testing.T) []string {
	t.Helper()
	var members []string
	walkUTF8Fields(t, reflect.TypeOf(clonefidelity.FidelityReportInput{}), "report", &members)
	var standalone []string
	walkUTF8Fields(t, reflect.TypeOf(clonefidelity.DispositionRecordInput{}), "row", &standalone)
	nested := map[string]bool{}
	for _, member := range members {
		if rest, ok := strings.CutPrefix(member, "report.Rows[]."); ok {
			nested[rest] = true
		}
	}
	if len(standalone) == 0 {
		t.Fatal("standalone row walk derived no members")
	}
	for _, member := range standalone {
		rest, _ := strings.CutPrefix(member, "row.")
		if !nested[rest] {
			t.Fatalf("standalone row member %q is missing from the nested Rows[] set", member)
		}
	}
	if len(nested) != len(standalone) {
		t.Fatalf("nested row set has %d members, standalone has %d", len(nested), len(standalone))
	}
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

func walkUTF8Fields(t *testing.T, typ reflect.Type, path string, out *[]string) {
	t.Helper()
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		walkUTF8Field(t, field, path+"."+field.Name, out)
	}
}

func walkUTF8Field(t *testing.T, field reflect.StructField, path string, out *[]string) {
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
		if fieldType.Elem().Kind() != reflect.String {
			t.Fatalf("unhandled pointer at %s: extend the walker", path)
		}
		emit(path)
	case reflect.Slice, reflect.Array:
		element := fieldType.Elem()
		switch element.Kind() {
		case reflect.Uint8:
			emit(path)
		case reflect.String:
			emit(path + "[]")
		case reflect.Struct:
			for index := 0; index < element.NumField(); index++ {
				sub := element.Field(index)
				walkUTF8Field(t, sub, path+"[]."+sub.Name, out)
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
		case reflect.Struct:
			emit(path + "<key>")
			for index := 0; index < value.NumField(); index++ {
				sub := value.Field(index)
				walkUTF8Field(t, sub, path+"<value>."+sub.Name, out)
			}
		case reflect.Uint64:
			emit(path + "<key>")
		default:
			t.Fatalf("unhandled map value kind %s at %s: extend the walker", value.Kind(), path)
		}
	case reflect.Struct:
		walkUTF8Fields(t, fieldType, path, out)
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// Scalar non-text fields carry no strings.
	default:
		t.Fatalf("unhandled kind %s at %s: extend the walker", fieldType.Kind(), path)
	}
}

// utf8Probe drives one derived member at both entries: buildSet
// plants the bad string in a fresh baseline input, anchor/value
// locate the sealed bytes to splice for the decode plant, and
// buildLiterals are the member-specific refusal tokens.
type utf8Probe struct {
	buildSet      func(input *clonefidelity.FidelityReportInput, bad string)
	anchor        []byte
	value         []byte
	buildLiterals []string
}

// utf8TupleJSON renders a valid Environment Tuple with the given
// platform. Source uses linux and target uses macos so the decode
// anchors are unique per tuple.
func utf8TupleJSON(platform string) string {
	return `{"environment_id":"test.env","environment_version":"2.1.0","platform":"` +
		platform + `","architecture":"amd64","store_schema_fingerprint":"` +
		fixtureDigest("fidelity-store-fingerprint") + `","adapter_version":"1.2.3"}`
}

// utf8BaselineInput is the valid report every grid cell starts from:
// one semantic row with populated evidence arrays and extensions at
// both layers, distinct tuple platforms, and one attestation, so
// every derived member has bytes to plant.
func utf8BaselineInput(t *testing.T) clonefidelity.FidelityReportInput {
	t.Helper()
	row := validRowInput("utf8-grid", "semantic", "unknown_native_event")
	row.StagedEvidenceObjectIDs = []string{fixtureDigest("staged-utf8-grid")}
	row.LiveEvidenceObjectIDs = []string{fixtureDigest("live-utf8-grid")}
	row.Extensions = map[string]any{"com.example.utf8rowprobe": "utf8rowvalue"}
	input := validTargetInput(row)
	input.SourceEnvironment = []byte(utf8TupleJSON("linux"))
	input.TargetEnvironment = []byte(utf8TupleJSON("macos"))
	input.AdapterAttestations = []string{fixtureDigest("attest-utf8-grid")}
	input.Extensions = map[string]any{"com.example.utf8probe": "utf8probevalue"}
	return input
}

func utf8Probes(t *testing.T) map[string]utf8Probe {
	t.Helper()
	vocabTail := "is outside exact|semantic|summarized|opaque_preserved|synthesized|omitted|unrecoverable"
	kind0 := clonebundle.EventKinds()[0]
	block0 := clonebundle.ContentBlockTypes()[0]
	evidence := fixtureDigest("evidence-utf8-grid")
	canonical := fixtureDigest("canonical-utf8-grid")
	staged := fixtureDigest("staged-utf8-grid")
	live := fixtureDigest("live-utf8-grid")
	attestation := fixtureDigest("attest-utf8-grid")
	snapshot := fixtureDigest("source-snapshot")
	capture := fixtureDigest("capture-manifest")
	session := fixtureDigest("canonical-session")
	plan := fixtureDigest("projection-plan")
	stagedManifest := fixtureDigest("staged-manifest")
	liveManifest := fixtureDigest("live-manifest")
	anchor := func(text string) []byte { return []byte(text) }
	probes := map[string]utf8Probe{
		"report.Scope": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.Scope = bad },
			anchor:        anchor(`"scope":"target"`),
			value:         anchor("target"),
			buildLiterals: []string{"scope is outside archive|target"},
		},
		"report.OperationID": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.OperationID = bad },
			anchor:        anchor(`"operation_id":"` + fixtureOperationID + `"`),
			value:         anchor(fixtureOperationID),
			buildLiterals: []string{"operation_id is not a UUIDv7"},
		},
		"report.BundleID": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.BundleID = bad },
			anchor:        anchor(`"bundle_id":"` + fixtureBundleID + `"`),
			value:         anchor(fixtureBundleID),
			buildLiterals: []string{"bundle_id is not a UUIDv7"},
		},
		"report.SourceSnapshotDigest": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.SourceSnapshotDigest = bad },
			anchor:        anchor(`"source_snapshot_digest":"` + snapshot + `"`),
			value:         anchor(snapshot),
			buildLiterals: []string{"source_snapshot_digest is not a digest"},
		},
		"report.CaptureManifestID": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.CaptureManifestID = bad },
			anchor:        anchor(`"capture_manifest_id":"` + capture + `"`),
			value:         anchor(capture),
			buildLiterals: []string{"capture_manifest_id is not a digest"},
		},
		"report.CanonicalSessionID": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.CanonicalSessionID = bad },
			anchor:        anchor(`"canonical_session_id":"` + session + `"`),
			value:         anchor(session),
			buildLiterals: []string{"canonical_session_id is not a digest"},
		},
		"report.ProjectionPlanID": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.ProjectionPlanID = strptr(bad) },
			anchor:        anchor(`"projection_plan_id":"` + plan + `"`),
			value:         anchor(plan),
			buildLiterals: []string{"projection_plan_id is not a digest"},
		},
		"report.SourceEnvironment": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.SourceEnvironment = []byte(strings.Replace(utf8TupleJSON("linux"), `"platform":"linux"`, `"platform":"`+bad+`"`, 1))
			},
			anchor:        anchor(`"platform":"linux"`),
			value:         anchor("linux"),
			buildLiterals: []string{"source_environment is not an Environment Tuple"},
		},
		"report.TargetEnvironment": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.TargetEnvironment = []byte(strings.Replace(utf8TupleJSON("macos"), `"platform":"macos"`, `"platform":"`+bad+`"`, 1))
			},
			anchor:        anchor(`"platform":"macos"`),
			value:         anchor("macos"),
			buildLiterals: []string{"target_environment is not an Environment Tuple"},
		},
		"report.Profile": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) { input.Profile = bad },
			anchor:   anchor(`"profile":"maximal_safe"`),
			value:    anchor("maximal_safe"),
			buildLiterals: []string{
				"profile",
				"is outside strict_exact|maximal_safe|compact|messages_only|archive_only",
			},
		},
		"report.RequiredDispositions<key>": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.RequiredDispositions = map[string][]string{bad: {"exact"}}
			},
			anchor:        anchor(`"required_dispositions":{"durable_payload":`),
			value:         anchor("durable_payload"),
			buildLiterals: []string{"required_dispositions", "unknown class"},
		},
		"report.RequiredDispositions[]": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.RequiredDispositions = map[string][]string{"durable_payload": {bad}}
			},
			anchor:        anchor(`"required_dispositions":{"durable_payload":["exact","semantic"]}`),
			value:         anchor("semantic"),
			buildLiterals: []string{"required_dispositions", vocabTail},
		},
		"report.ForbidReasons[]": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.ForbidReasons = []string{bad}
			},
			anchor:        anchor(`"forbid_reasons":["credential_excluded"]`),
			value:         anchor("credential_excluded"),
			buildLiterals: []string{"forbid_reasons[0] is not valid UTF-8"},
		},
		"report.Rows[].SourceItemKey": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.Rows[0].SourceItemKey = bad },
			anchor:        anchor(`"source_item_key":"utf8-grid"`),
			value:         anchor("utf8-grid"),
			buildLiterals: []string{"source_item_key is not valid UTF-8"},
		},
		"report.Rows[].SourceClass": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.Rows[0].SourceClass = bad },
			anchor:        anchor(`"source_class":"durable_payload"`),
			value:         anchor("durable_payload"),
			buildLiterals: []string{"source_class is not valid UTF-8"},
		},
		"report.Rows[].SourceEvidenceIDs[]": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Rows[0].SourceEvidenceIDs = []string{bad}
			},
			anchor:        anchor(`"source_evidence_ids":["` + evidence + `"]`),
			value:         anchor(evidence),
			buildLiterals: []string{"source_evidence_ids[0] is not a digest"},
		},
		"report.Rows[].CanonicalObjectID": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Rows[0].CanonicalObjectID = strptr(bad)
			},
			anchor:        anchor(`"canonical_object_id":"` + canonical + `"`),
			value:         anchor(canonical),
			buildLiterals: []string{"canonical_object_id is not a digest"},
		},
		"report.Rows[].TargetLocator": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.Rows[0].TargetLocator = strptr(bad) },
			anchor:        anchor(`"target_locator":"target/utf8-grid"`),
			value:         anchor("target/utf8-grid"),
			buildLiterals: []string{"target_locator is not valid UTF-8"},
		},
		"report.Rows[].Disposition": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.Rows[0].Disposition = bad },
			anchor:        anchor(`"disposition":"semantic"`),
			value:         anchor("semantic"),
			buildLiterals: []string{"disposition", vocabTail},
		},
		"report.Rows[].ReasonCodes[]": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Rows[0].ReasonCodes = []string{bad}
			},
			anchor:        anchor(`"reason_codes":["unknown_native_event"]`),
			value:         anchor("unknown_native_event"),
			buildLiterals: []string{"reason_codes[0] is not valid UTF-8"},
		},
		"report.Rows[].Explanation": {
			buildSet:      func(input *clonefidelity.FidelityReportInput, bad string) { input.Rows[0].Explanation = bad },
			anchor:        anchor(`"explanation":"row utf8-grid"`),
			value:         anchor("row utf8-grid"),
			buildLiterals: []string{"explanation is not valid UTF-8"},
		},
		"report.Rows[].StagedEvidenceObjectIDs[]": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Rows[0].StagedEvidenceObjectIDs = []string{bad}
			},
			anchor:        anchor(`"staged_evidence_object_ids":["` + staged + `"]`),
			value:         anchor(staged),
			buildLiterals: []string{"staged_evidence_object_ids[0] is not a digest"},
		},
		"report.Rows[].LiveEvidenceObjectIDs[]": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Rows[0].LiveEvidenceObjectIDs = []string{bad}
			},
			anchor:        anchor(`"live_evidence_object_ids":["` + live + `"]`),
			value:         anchor(live),
			buildLiterals: []string{"live_evidence_object_ids[0] is not a digest"},
		},
		"report.Rows[].Extensions<key>": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Rows[0].Extensions = map[string]any{bad: "utf8rowkeyvalue"}
			},
			anchor:        anchor(`"com.example.utf8rowprobe":"utf8rowvalue"`),
			value:         anchor("com.example.utf8rowprobe"),
			buildLiterals: []string{"extensions", "not valid UTF-8"},
		},
		"report.Rows[].Extensions<value>": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Rows[0].Extensions = map[string]any{"com.example.utf8rowprobe": bad}
			},
			anchor:        anchor(`"com.example.utf8rowprobe":"utf8rowvalue"`),
			value:         anchor("utf8rowvalue"),
			buildLiterals: []string{"extensions", "not valid UTF-8"},
		},
		"report.EventKindCounts<key>": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.EventKindCounts[bad] = clonefidelity.FidelityCounts{}
			},
			anchor:        anchor(`"` + kind0 + `":{`),
			value:         anchor(kind0),
			buildLiterals: []string{"event_kind_counts", "unknown key"},
		},
		"report.ContentBlockCounts<key>": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.ContentBlockCounts[bad] = clonefidelity.FidelityCounts{}
			},
			anchor:        anchor(`"` + block0 + `":{`),
			value:         anchor(block0),
			buildLiterals: []string{"content_block_counts", "unknown key"},
		},
		"report.ByteCounts<key>": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.ByteCounts[bad] = 0
			},
			anchor:        anchor(`"byte_counts":{"exact":`),
			value:         anchor("exact"),
			buildLiterals: []string{"byte_counts", "unknown disposition"},
		},
		"report.StagedReadBackEvidenceManifestID": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.StagedReadBackEvidenceManifestID = strptr(bad)
			},
			anchor:        anchor(`"staged_read_back_evidence_manifest_id":"` + stagedManifest + `"`),
			value:         anchor(stagedManifest),
			buildLiterals: []string{"staged_read_back_evidence_manifest_id is not a digest"},
		},
		"report.LiveReadBackEvidenceManifestID": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.LiveReadBackEvidenceManifestID = strptr(bad)
			},
			anchor:        anchor(`"live_read_back_evidence_manifest_id":"` + liveManifest + `"`),
			value:         anchor(liveManifest),
			buildLiterals: []string{"live_read_back_evidence_manifest_id is not a digest"},
		},
		"report.AdapterAttestations[]": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.AdapterAttestations = []string{bad}
			},
			anchor:        anchor(`"adapter_attestations":["` + attestation + `"]`),
			value:         anchor(attestation),
			buildLiterals: []string{"adapter_attestations[0] is not a digest"},
		},
		"report.Extensions<key>": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Extensions = map[string]any{bad: "utf8keyvalue"}
			},
			anchor:        anchor(`"com.example.utf8probe":"utf8probevalue"`),
			value:         anchor("com.example.utf8probe"),
			buildLiterals: []string{"extensions", "not valid UTF-8"},
		},
		"report.Extensions<value>": {
			buildSet: func(input *clonefidelity.FidelityReportInput, bad string) {
				input.Extensions = map[string]any{"com.example.utf8probe": bad}
			},
			anchor:        anchor(`"com.example.utf8probe":"utf8probevalue"`),
			value:         anchor("utf8probevalue"),
			buildLiterals: []string{"extensions", "not valid UTF-8"},
		},
	}
	return probes
}

// spliceUTF8Bytes replaces the anchored value with the raw bad bytes
// exactly once. The anchor must occur exactly once in the sealed
// document and must contain the value, so every decode plant is
// pinned to its member.
func spliceUTF8Bytes(t *testing.T, sealed, anchor, value, bad []byte) []byte {
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
