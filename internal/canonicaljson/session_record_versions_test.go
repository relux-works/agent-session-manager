package canonicaljson

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	sessionID       = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	sourceSessionID = "0198f4c8-8e50-7f66-8f60-3234567890ab"
	operationID     = "0198f4c8-b180-7299-9273-1234567890ab"
	bundleID        = "0198f4c8-c290-73aa-9384-1234567890ab"
)

type sessionRecordProvenanceFixture struct {
	name    string
	version string
	value   map[string]any
}

func sessionRecordProvenanceFixtures() []sessionRecordProvenanceFixture {
	return []sessionRecordProvenanceFixture{
		{"v2 origin", "2.0.0", validOriginProvenance()},
		{"v2 same-provider fork", "2.0.0", validSameProviderForkProvenance()},
		{"v2 AX clone", "2.0.0", validCrossEnvironmentCloneProvenance("ax_session")},
		{"v2 external-native clone", "2.0.0", validCrossEnvironmentCloneProvenance("external_native")},
		{"v3 origin", "3.0.0", validOriginProvenance()},
		{"v3 same-provider fork", "3.0.0", validSameProviderForkProvenance()},
		{"v3 AX clone", "3.0.0", validCrossEnvironmentCloneProvenance("ax_session")},
		{"v3 external-native clone", "3.0.0", validCrossEnvironmentCloneProvenance("external_native")},
		{"v3 native adoption", "3.0.0", validNativeAdoptionProvenance()},
	}
}

func TestSessionRecordCatalogVersionsReachIdentityProductionEntries(t *testing.T) {
	t.Parallel()

	var versions []string
	for _, contract := range catalog.Current().SelfIdentities {
		if string(contract.ContractID) == "urn:ax:schema:session-record" {
			versions = append(versions, contract.ContractVersions...)
		}
	}
	if len(versions) != 3 {
		t.Fatalf("generated Session Record version inventory = %v, want three versions", versions)
	}

	objects := map[string]map[string]any{
		"1.0.0": validSessionRecordV1Object(),
		"2.0.0": validSessionRecordV2Object(validOriginProvenance()),
		"3.0.0": validSessionRecordV3Object(validNativeAdoptionProvenance()),
	}
	for _, version := range versions {
		object, ok := objects[version]
		if !ok {
			t.Fatalf("generated Session Record version %s has no production-entry fixture", version)
		}
		assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfRecordID)
	}
}

func TestSessionRecordProvenanceTagsReachIdentityProductionEntries(t *testing.T) {
	for _, test := range sessionRecordProvenanceFixtures() {
		t.Run(test.name, func(t *testing.T) {
			object := validSessionRecordV2Object(test.value)
			object["schema_version"] = test.version
			assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfRecordID)
		})
	}
}

func TestSessionRecordProvenanceRefusalsReachIdentityProductionEntries(t *testing.T) {
	tests := []struct {
		name   string
		object func() map[string]any
		want   string
	}{
		{"v1 refuses derivation member", func() map[string]any {
			object := validSessionRecordV1Object()
			object["derivation_provenance"] = validOriginProvenance()
			return object
		}, "contains unknown member"},
		{"v2 refuses fork member", func() map[string]any {
			object := validSessionRecordV2Object(validOriginProvenance())
			object["fork_provenance"] = nil
			return object
		}, "contains unknown member"},
		{"v3 refuses invented creation member", func() map[string]any {
			object := validSessionRecordV3Object(validOriginProvenance())
			object["creation_provenance"] = object["derivation_provenance"]
			delete(object, "derivation_provenance")
			return object
		}, "creation_provenance"},
		{"unknown tag", func() map[string]any {
			object := validSessionRecordV3Object(validOriginProvenance())
			object["derivation_provenance"].(map[string]any)["kind"] = "move"
			return object
		}, "derivation_provenance.kind"},
		{"cross-tag member leakage", func() map[string]any {
			object := validSessionRecordV3Object(validOriginProvenance())
			object["derivation_provenance"].(map[string]any)["operation_id"] = operationID
			return object
		}, "contains unknown member"},
		{"same-provider fork refuses origin creation fact", func() map[string]any {
			provenance := validSameProviderForkProvenance()
			provenance["creation_operation_id"] = operationID
			return validSessionRecordV3Object(provenance)
		}, "contains unknown member"},
		{"cross-environment clone refuses origin creation fact", func() map[string]any {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["creation_operation_id"] = operationID
			return validSessionRecordV3Object(provenance)
		}, "contains unknown member"},
		{"native adoption refuses clone final fact", func() map[string]any {
			provenance := validNativeAdoptionProvenance()
			provenance["bundle_id"] = bundleID
			return validSessionRecordV3Object(provenance)
		}, "contains unknown member"},
		{"same-provider target identity reuse", func() map[string]any {
			provenance := validSameProviderForkProvenance()
			provenance["source_session_id"] = sessionID
			return validSessionRecordV3Object(provenance)
		}, "source_session_id must differ"},
		{"v1 fork target identity reuse", func() map[string]any {
			object := validSessionRecordV1Object()
			object["fork_provenance"] = map[string]any{
				"source_session_id": sessionID, "source_checkpoint_id": zeroDigest,
				"source_workspace_group_id": "0198f4c8-5b20-7c33-8c4d-1234567890ab",
				"operation_id":              operationID, "provider_fork_mode": "native", "extensions": map[string]any{},
			}
			return object
		}, "source_session_id must differ"},
		{"AX clone target identity reuse", func() map[string]any {
			provenance := validCrossEnvironmentCloneProvenance("ax_session")
			provenance["source_session_id"] = sessionID
			return validSessionRecordV3Object(provenance)
		}, "source_session_id must differ"},
		{"environment tuple unknown member", func() map[string]any {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["source_environment"].(map[string]any)["executable_sha256"] = zeroDigest
			return validSessionRecordV2Object(provenance)
		}, "EnvironmentTuple contains unknown member"},
		{"native adoption provider mutation", func() map[string]any {
			provenance := validNativeAdoptionProvenance()
			provenance["target_provider_id"] = "claude"
			return validSessionRecordV3Object(provenance)
		}, "target_provider_id must equal Session Record provider_id"},
		{"native adoption unavailable in v2", func() map[string]any {
			return validSessionRecordV2Object(validNativeAdoptionProvenance())
		}, "derivation_provenance.kind"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.object()), SelfRecordID, test.want)
		})
	}
}

func TestSessionRecordClosedVocabulariesRefuseUnknownMembersAtIdentityProductionEntries(t *testing.T) {
	tests := []struct {
		name   string
		object func() map[string]any
		want   string
	}{
		{"execution profile", func() map[string]any {
			object := validSessionRecordV3Object(validOriginProvenance())
			object["execution_profile"] = "sandbox"
			return object
		}, "execution_profile"},
		{"same-provider fork mode", func() map[string]any {
			provenance := validSameProviderForkProvenance()
			provenance["provider_fork_mode"] = "guessed_import"
			return validSessionRecordV3Object(provenance)
		}, "provider_fork_mode"},
		{"cross-environment source kind", func() map[string]any {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["source_kind"] = "unmanaged_guess"
			return validSessionRecordV3Object(provenance)
		}, "source_kind"},
		{"environment architecture", func() map[string]any {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["source_environment"].(map[string]any)["architecture"] = "riscv64"
			return validSessionRecordV3Object(provenance)
		}, "architecture"},
		{"environment platform", func() map[string]any {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["source_environment"].(map[string]any)["platform"] = "freebsd"
			return validSessionRecordV3Object(provenance)
		}, "platform"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.object()), SelfRecordID, test.want)
		})
	}
}

func TestSessionRecordClosedVocabulariesAcceptEveryDeclaredMemberAtIdentityProductionEntries(t *testing.T) {
	for _, platform := range declaredScalarPlatformVocabulary(t) {
		t.Run("environment platform "+platform, func(t *testing.T) {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["source_environment"].(map[string]any)["platform"] = platform
			assertIdentityEntriesAcceptShape(t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID)
		})
	}

	for _, mode := range []string{"native", "supported_import", "task_board_clone"} {
		t.Run("same-provider fork mode "+mode, func(t *testing.T) {
			provenance := validSameProviderForkProvenance()
			provenance["provider_fork_mode"] = mode
			assertIdentityEntriesAcceptShape(t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID)
		})
	}

	for _, profile := range []string{"standard", "yolo"} {
		t.Run("execution profile "+profile, func(t *testing.T) {
			object := validSessionRecordV3Object(validOriginProvenance())
			object["execution_profile"] = profile
			assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfRecordID)
		})
	}
}

func TestSessionRecordEveryProvenanceExtensionsGateReachesIdentityProductionEntries(t *testing.T) {
	for _, test := range sessionRecordProvenanceFixtures() {
		t.Run(test.name, func(t *testing.T) {
			test.value["extensions"] = map[string]any{"not_reverse_dns": true}
			object := validSessionRecordV2Object(test.value)
			object["schema_version"] = test.version
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfRecordID, "extensions key")
		})
	}
}

func TestSessionRecordAXSourceNullabilityCoversEveryDeclaredIdentityMemberAtProductionEntries(t *testing.T) {
	axFixture := validCrossEnvironmentCloneProvenance("ax_session")
	identityMembers := derivedAXSourceIdentityMembers(t)

	for _, member := range identityMembers {
		member := member
		t.Run("ax_session requires "+member, func(t *testing.T) {
			provenance := validCrossEnvironmentCloneProvenance("ax_session")
			provenance[member] = nil
			assertIdentityEntriesRefuseWithReason(
				t, mustJSON(t, validSessionRecordV2Object(provenance)), SelfRecordID,
				"all four AX-source IDs must be non-null",
			)
		})
		t.Run("external_native refuses "+member, func(t *testing.T) {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance[member] = axFixture[member]
			assertIdentityEntriesRefuseWithReason(
				t, mustJSON(t, validSessionRecordV2Object(provenance)), SelfRecordID,
				"all four AX-source IDs must be null",
			)
		})
	}
}

func TestSessionRecordCrossEnvironmentCloneDoesNotInferEnvironmentEqualityAtIdentityProductionEntries(t *testing.T) {
	tests := []struct {
		name   string
		target map[string]any
	}{
		{
			name:   "same environment ID with a different tuple",
			target: validEnvironmentTuple("claude-code", "linux", "amd64"),
		},
		{
			name:   "identical source and target tuples",
			target: validEnvironmentTuple("claude-code", "macos", "arm64"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["target_environment"] = test.target
			assertIdentityEntriesAcceptShape(t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID)
		})
	}
}

func TestSessionRecordTypedProvenanceMembersRefuseMalformedValuesAtIdentityProductionEntries(t *testing.T) {
	tests := []struct {
		name   string
		object func() map[string]any
		want   string
	}{
		{
			name: "origin creation operation ID",
			object: func() map[string]any {
				provenance := validOriginProvenance()
				provenance["creation_operation_id"] = "not-a-uuid"
				return validSessionRecordV3Object(provenance)
			},
			want: "creation_operation_id",
		},
		{
			name: "same-provider source profile event ID",
			object: func() map[string]any {
				provenance := validSameProviderForkProvenance()
				provenance["source_profile_event_id"] = "not-a-digest"
				return validSessionRecordV3Object(provenance)
			},
			want: "source_profile_event_id",
		},
		{
			name: "clone bundle ID",
			object: func() map[string]any {
				provenance := validCrossEnvironmentCloneProvenance("external_native")
				provenance["bundle_id"] = "not-a-uuid"
				return validSessionRecordV3Object(provenance)
			},
			want: "bundle_id",
		},
	}

	for _, field := range []string{
		"source_snapshot_digest", "capture_manifest_id", "canonical_session_id", "projection_plan_id", "migration_checkpoint_id",
	} {
		field := field
		tests = append(tests, struct {
			name   string
			object func() map[string]any
			want   string
		}{
			name: "clone required digest " + field,
			object: func() map[string]any {
				provenance := validCrossEnvironmentCloneProvenance("external_native")
				provenance[field] = "not-a-digest"
				return validSessionRecordV3Object(provenance)
			},
			want: field,
		})
	}
	for _, field := range []string{"previous_lineage_receipt_id", "source_profile_event_id"} {
		field := field
		tests = append(tests, struct {
			name   string
			object func() map[string]any
			want   string
		}{
			name: "clone nullable digest " + field,
			object: func() map[string]any {
				provenance := validCrossEnvironmentCloneProvenance("external_native")
				provenance[field] = "not-a-digest"
				return validSessionRecordV3Object(provenance)
			},
			want: field,
		})
	}
	axFixture := validCrossEnvironmentCloneProvenance("ax_session")
	for _, field := range derivedAXSourceIdentityMembers(t) {
		field := field
		value, ok := axFixture[field].(string)
		if !ok {
			t.Fatalf("derived AX-source identity member %s has non-string fixture value %T", field, axFixture[field])
		}
		malformed := "not-a-digest"
		if _, err := scalar.ParseUUIDv7(value); err == nil {
			malformed = "not-a-uuid"
		} else if _, err := scalar.ParseDigest(value); err != nil {
			t.Fatalf("derived AX-source identity member %s has unrecognized fixture type: %v", field, err)
		}
		tests = append(tests, struct {
			name   string
			object func() map[string]any
			want   string
		}{
			name: "clone AX-source typed member " + field,
			object: func() map[string]any {
				provenance := validCrossEnvironmentCloneProvenance("ax_session")
				provenance[field] = malformed
				return validSessionRecordV3Object(provenance)
			},
			want: field,
		})
	}
	for _, field := range []string{"operation_id", "source_host_id"} {
		field := field
		tests = append(tests, struct {
			name   string
			object func() map[string]any
			want   string
		}{
			name: "native-adoption UUID " + field,
			object: func() map[string]any {
				provenance := validNativeAdoptionProvenance()
				provenance[field] = "not-a-uuid"
				return validSessionRecordV3Object(provenance)
			},
			want: field,
		})
	}
	for _, field := range []string{"source_instance_id", "source_observation_id", "source_head_digest"} {
		field := field
		tests = append(tests, struct {
			name   string
			object func() map[string]any
			want   string
		}{
			name: "native-adoption digest " + field,
			object: func() map[string]any {
				provenance := validNativeAdoptionProvenance()
				provenance[field] = "not-a-digest"
				return validSessionRecordV3Object(provenance)
			},
			want: field,
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.object()), SelfRecordID, test.want)
		})
	}
}

func TestSessionRecordNativeAdoptionProviderGrammarIsSubsumedAtIdentityProductionEntries(t *testing.T) {
	t.Run("malformed target differs from validated provider", func(t *testing.T) {
		provenance := validNativeAdoptionProvenance()
		provenance["target_provider_id"] = "Not A Provider"
		assertIdentityEntriesRefuseWithReason(
			t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID,
			"target_provider_id must equal Session Record provider_id",
		)
	})

	t.Run("matching malformed provider is refused by common record validation", func(t *testing.T) {
		provenance := validNativeAdoptionProvenance()
		provenance["target_provider_id"] = "Not A Provider"
		object := validSessionRecordV3Object(provenance)
		object["provider_id"] = "Not A Provider"
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfRecordID, "Session Record provider_id")
	})
}

func TestSessionRecordUnicodeCharacterBoundsAtIdentityProductionEntries(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		at     string
		past   string
		mutate func(map[string]any, string)
	}{
		{
			name:  "source native session ID",
			field: "source_native_session_id",
			at:    strings.Repeat("界", 512),
			past:  strings.Repeat("界", 513),
			mutate: func(provenance map[string]any, value string) {
				provenance["source_native_session_id"] = value
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name+" accepts at limit", func(t *testing.T) {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			test.mutate(provenance, test.at)
			assertIdentityEntriesAcceptShape(t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID)
		})
		t.Run(test.name+" refuses past limit", func(t *testing.T) {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			test.mutate(provenance, test.past)
			assertIdentityEntriesRefuseWithReason(
				t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID,
				"member "+test.field+" must contain",
			)
		})
	}
}

func TestSessionRecordDeclaredStringBoundsBothDirectionsAtIdentityProductionEntries(t *testing.T) {
	t.Run("source native session ID accepts minimum", func(t *testing.T) {
		provenance := validCrossEnvironmentCloneProvenance("external_native")
		provenance["source_native_session_id"] = "界"
		assertIdentityEntriesAcceptShape(t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID)
	})
	t.Run("source native session ID refuses below minimum", func(t *testing.T) {
		provenance := validCrossEnvironmentCloneProvenance("external_native")
		provenance["source_native_session_id"] = ""
		assertIdentityEntriesRefuseShape(t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID)
	})
	t.Run("source native session ID refuses control characters", func(t *testing.T) {
		provenance := validCrossEnvironmentCloneProvenance("external_native")
		provenance["source_native_session_id"] = "native\nsession"
		assertIdentityEntriesRefuseShape(t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID)
	})
	for _, test := range []struct {
		name  string
		value string
		pass  bool
	}{
		{"environment ID accepts one", "a", true},
		{"environment ID accepts 64", "a" + strings.Repeat("b", 63), true},
		{"environment ID refuses empty", "", false},
		{"environment ID refuses 65", "a" + strings.Repeat("b", 64), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["source_environment"].(map[string]any)["environment_id"] = test.value
			input := mustJSON(t, validSessionRecordV3Object(provenance))
			if test.pass {
				assertIdentityEntriesAcceptShape(t, input, SelfRecordID)
			} else {
				assertIdentityEntriesRefuseShape(t, input, SelfRecordID)
			}
		})
	}
}

func TestSessionRecordDeclaredGrammarRowsReachIdentityProductionEntries(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve Session Record grammar test source path")
	}
	packageDirectory := filepath.Dir(source)
	rows := readConstraintRows(t, filepath.Join(packageDirectory, "testdata", "constraint-enumeration.md"))

	readme, err := os.ReadFile(filepath.Join(packageDirectory, "..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, claim := range []string{
		"Section 2.1 ASCII name grammar",
		"declared environment-ID grammar",
		"Section 1.6 reverse-DNS",
	} {
		if !bytes.Contains(readme, []byte(claim)) {
			t.Fatalf("README no longer declares Session Record grammar claim %q", claim)
		}
	}

	familyCounts := make(map[string]int)
	for _, row := range rows {
		family, declared := sessionRecordGrammarFamily(row)
		if !declared {
			continue
		}
		familyCounts[family]++
		row := row
		t.Run(row.shape+" "+row.member, func(t *testing.T) {
			valid := sessionRecordWithDeclaredGrammarValue(t, row, family, validSessionRecordGrammarValue(family))
			assertIdentityEntriesAcceptShape(t, mustJSON(t, valid), SelfRecordID)

			for _, invalid := range invalidSessionRecordGrammarValues(family) {
				invalid := invalid
				t.Run(invalid.name, func(t *testing.T) {
					object := sessionRecordWithDeclaredGrammarValue(t, row, family, invalid.value)
					assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfRecordID)
				})
			}
		})
	}

	for _, family := range []string{
		"environment-id", "environment-name", "provider-id",
		"reverse-dns", "session-name", "board-logical-id",
	} {
		if familyCounts[family] == 0 {
			t.Errorf("constraint enumeration declares no Session Record %s grammar row", family)
		}
	}
}

type sessionRecordGrammarValue struct {
	name  string
	value string
}

func sessionRecordGrammarFamily(row documentedConstraintRow) (string, bool) {
	if row.shape != "EnvironmentTuple" && !strings.HasPrefix(row.shape, "Session Record") {
		return "", false
	}

	declaration := strings.ToLower(row.specExcerpt)
	switch {
	case strings.Contains(declaration, "reverse-dns"):
		return "reverse-dns", true
	case strings.Contains(declaration, "environment-name") || strings.Contains(declaration, "environment names"):
		return "environment-name", true
	case row.shape == "EnvironmentTuple" && row.member == "environment_id" && strings.Contains(declaration, "[a-z]"):
		return "environment-id", true
	case row.member == "name" && strings.Contains(declaration, "section 2.1 grammar"):
		return "session-name", true
	case row.shape == "Session Record Board Identity" && row.member == "logical_id" && strings.Contains(declaration, "[a-za-z0-9]"):
		return "board-logical-id", true
	case row.member == "provider_id" && strings.Contains(declaration, "lowercase plugin id"):
		return "provider-id", true
	default:
		return "", false
	}
}

func validSessionRecordGrammarValue(family string) string {
	switch family {
	case "environment-id":
		return "a0.b-c"
	case "environment-name":
		return "_A9"
	case "provider-id":
		return "a0-b"
	case "reverse-dns":
		return "a-b.c0"
	case "session-name":
		return "9_A.b-c"
	case "board-logical-id":
		return "9_A.b:c-d"
	default:
		panic("unknown Session Record grammar family " + family)
	}
}

func invalidSessionRecordGrammarValues(family string) []sessionRecordGrammarValue {
	switch family {
	case "environment-id":
		return rejectedLowercaseIdentifierCharacterClasses("environment")
	case "environment-name":
		return []sessionRecordGrammarValue{
			{"leading digit", "9NAME"},
			{"leading hyphen", "-NAME"},
			{"internal hyphen", "NAME-VALUE"},
			{"non-ASCII", "ÉNAME"},
		}
	case "provider-id":
		return rejectedLowercaseIdentifierCharacterClasses("provider")
	case "reverse-dns":
		return []sessionRecordGrammarValue{
			{"uppercase", "Example.com"},
			{"underscore", "example_key.com"},
			{"leading digit", "1example.com"},
			{"leading hyphen", "-example.com"},
			{"non-ASCII", "éxample.com"},
		}
	case "session-name":
		return []sessionRecordGrammarValue{
			{"leading underscore", "_payments"},
			{"leading hyphen", "-payments"},
			{"space", "payments api"},
			{"control", "payments\napi"},
			{"non-ASCII", "платежи"},
		}
	case "board-logical-id":
		return []sessionRecordGrammarValue{
			{"leading underscore", "_board"},
			{"leading hyphen", "-board"},
			{"space", "board id"},
			{"control", "board\nid"},
			{"non-ASCII", "доска"},
		}
	default:
		panic("unknown Session Record grammar family " + family)
	}
}

func rejectedLowercaseIdentifierCharacterClasses(stem string) []sessionRecordGrammarValue {
	return []sessionRecordGrammarValue{
		{"uppercase", "A" + stem},
		{"underscore", "a_" + stem},
		{"leading digit", "1" + stem},
		{"leading hyphen", "-" + stem},
		{"non-ASCII", "é" + stem},
	}
}

func sessionRecordWithDeclaredGrammarValue(
	t *testing.T,
	row documentedConstraintRow,
	family string,
	value string,
) map[string]any {
	t.Helper()

	switch family {
	case "session-name":
		if row.shape == "Session Record 1.0.0" {
			object := validSessionRecordV1Object()
			object["name"] = value
			return object
		}
		object := validSessionRecordV3Object(validOriginProvenance())
		object["name"] = value
		return object
	case "provider-id":
		object := validSessionRecordV1Object()
		object["provider_id"] = value
		return object
	case "environment-id":
		object := validSessionRecordV3Object(validCrossEnvironmentCloneProvenance("external_native"))
		provenance := object["derivation_provenance"].(map[string]any)
		provenance["source_environment"].(map[string]any)["environment_id"] = value
		return object
	case "board-logical-id":
		object := taskBoardSessionRecord("TASK-260830-3esaam")
		board := object["task_board"].(map[string]any)["board"].(map[string]any)
		board["logical_id"] = value
		return object
	case "environment-name":
		object := validSessionRecordV1Object()
		launchPlan := object["launch_plan"].(map[string]any)
		if row.member == "env_names" {
			launchPlan["env_names"] = []any{value}
		} else {
			launchPlan["env_literals"] = map[string]any{value: "value"}
		}
		return object
	case "reverse-dns":
		object, extensions := sessionRecordExtensionsForDocumentedShape(t, row.shape)
		extensions[value] = true
		return object
	default:
		t.Fatalf("unsupported Session Record grammar row %s.%s (%s)", row.shape, row.member, family)
		return nil
	}
}

func sessionRecordExtensionsForDocumentedShape(t *testing.T, shape string) (map[string]any, map[string]any) {
	t.Helper()
	var object map[string]any
	var container map[string]any

	switch shape {
	case "Session Record 1.0.0":
		object = validSessionRecordV1Object()
		container = object
	case "Session Record 2.0.0 and 3.0.0":
		object = validSessionRecordV3Object(validOriginProvenance())
		container = object
	case "Session Record origin provenance":
		object = validSessionRecordV3Object(validOriginProvenance())
		container = object["derivation_provenance"].(map[string]any)
	case "Session Record same-provider-fork provenance":
		object = validSessionRecordV3Object(validSameProviderForkProvenance())
		container = object["derivation_provenance"].(map[string]any)
	case "Session Record cross-environment-clone provenance":
		object = validSessionRecordV3Object(validCrossEnvironmentCloneProvenance("external_native"))
		container = object["derivation_provenance"].(map[string]any)
	case "Session Record native-adoption provenance":
		object = validSessionRecordV3Object(validNativeAdoptionProvenance())
		container = object["derivation_provenance"].(map[string]any)
	case "Session Record Board Goal":
		object = primaryOwnerSessionRecord("GOAL-260830-primary")
		reference := object["task_board"].(map[string]any)
		container = reference["board_goal"].(map[string]any)
	case "Session Record Board Identity":
		object = taskBoardSessionRecord("TASK-260830-3esaam")
		reference := object["task_board"].(map[string]any)
		container = reference["board"].(map[string]any)
	case "Session Record Fork Provenance":
		object = validSessionRecordV1Object()
		container = map[string]any{
			"source_session_id":         sourceSessionID,
			"source_checkpoint_id":      zeroDigest,
			"source_workspace_group_id": "0198f4c8-5b20-7c33-8c4d-1234567890ab",
			"operation_id":              operationID,
			"provider_fork_mode":        "native",
			"extensions":                map[string]any{},
		}
		object["fork_provenance"] = container
	case "Session Record Launch Plan":
		object = validSessionRecordV1Object()
		container = object["launch_plan"].(map[string]any)
	case "Session Record Task-board Reference":
		object = taskBoardSessionRecord("TASK-260830-3esaam")
		container = object["task_board"].(map[string]any)
	default:
		t.Fatalf("declared reverse-DNS grammar row %q has no production-entry fixture", shape)
	}

	extensions, ok := container["extensions"].(map[string]any)
	if !ok {
		t.Fatalf("declared reverse-DNS grammar row %q has no extensions object", shape)
	}
	return object, extensions
}

func TestSessionRecordEnvironmentTupleDoesNotInferUnspecifiedMemberConstraintsAtIdentityProductionEntries(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value any
	}{
		{
			name:  "environment version has no inherited observation bound",
			field: "environment_version",
			value: strings.Repeat("界", 129),
		},
		{
			name:  "environment version has no inferred JSON type",
			field: "environment_version",
			value: true,
		},
		{
			name:  "store schema fingerprint is not inferred to be a digest",
			field: "store_schema_fingerprint",
			value: "sqlite-v7",
		},
		{
			name:  "store schema fingerprint has no inferred JSON type",
			field: "store_schema_fingerprint",
			value: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provenance := validCrossEnvironmentCloneProvenance("external_native")
			provenance["source_environment"].(map[string]any)[test.field] = test.value
			assertIdentityEntriesAcceptShape(t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID)
		})
	}
}

func TestSessionRecordIdentityCalculationIsRepeatableAndReadOnly(t *testing.T) {
	input := mustJSON(t, validSessionRecordV3Object(validNativeAdoptionProvenance()))
	original := bytes.Clone(input)
	first, firstField, err := CalculateObjectIdentity(input)
	if err != nil {
		t.Fatal(err)
	}
	second, secondField, err := CalculateObjectIdentity(input)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || firstField != secondField {
		t.Fatalf("repeated identity calculation drifted: %q/%q then %q/%q", first, firstField, second, secondField)
	}
	if !bytes.Equal(input, original) {
		t.Fatal("production entry mutated caller-owned input bytes")
	}
}

func validSessionRecordV2Object(provenance map[string]any) map[string]any {
	object := validSessionRecordV1Object()
	object["schema_version"] = "2.0.0"
	delete(object, "fork_provenance")
	object["derivation_provenance"] = provenance
	return object
}

func validSessionRecordV3Object(provenance map[string]any) map[string]any {
	object := validSessionRecordV2Object(provenance)
	object["schema_version"] = "3.0.0"
	return object
}

func validOriginProvenance() map[string]any {
	return map[string]any{
		"kind":                  "origin",
		"creation_operation_id": operationID,
		"extensions":            map[string]any{},
	}
}

func validSameProviderForkProvenance() map[string]any {
	return map[string]any{
		"kind":                      "same_provider_fork",
		"source_session_id":         sourceSessionID,
		"source_checkpoint_id":      zeroDigest,
		"source_workspace_group_id": "0198f4c8-5b20-7c33-8c4d-1234567890ab",
		"operation_id":              operationID,
		"provider_fork_mode":        "native",
		"source_profile_event_id":   nil,
		"extensions":                map[string]any{},
	}
}

func validCrossEnvironmentCloneProvenance(sourceKind string) map[string]any {
	provenance := map[string]any{
		"kind":                               "cross_environment_clone",
		"operation_id":                       operationID,
		"bundle_id":                          bundleID,
		"source_kind":                        sourceKind,
		"source_session_id":                  nil,
		"source_session_record_id":           nil,
		"source_checkpoint_id":               nil,
		"source_provider_identity_record_id": nil,
		"source_native_session_id":           "native-session-42",
		"source_environment":                 validEnvironmentTuple("claude-code", "macos", "arm64"),
		"target_environment":                 validEnvironmentTuple("codex", "linux", "amd64"),
		"source_snapshot_digest":             zeroDigest,
		"capture_manifest_id":                zeroDigest,
		"canonical_session_id":               zeroDigest,
		"projection_plan_id":                 zeroDigest,
		"migration_checkpoint_id":            zeroDigest,
		"previous_lineage_receipt_id":        nil,
		"source_profile_event_id":            nil,
		"extensions":                         map[string]any{},
	}
	if sourceKind == "ax_session" {
		provenance["source_session_id"] = sourceSessionID
		provenance["source_session_record_id"] = zeroDigest
		provenance["source_checkpoint_id"] = zeroDigest
		provenance["source_provider_identity_record_id"] = zeroDigest
	}
	return provenance
}

func validNativeAdoptionProvenance() map[string]any {
	return map[string]any{
		"kind":                  "native_adoption",
		"operation_id":          operationID,
		"source_host_id":        "0198f4c8-4a10-7b22-8b3c-1234567890ab",
		"source_instance_id":    zeroDigest,
		"source_observation_id": zeroDigest,
		"source_head_digest":    zeroDigest,
		"source_environment":    validEnvironmentTuple("codex", "macos", "arm64"),
		"target_provider_id":    "codex",
		"extensions":            map[string]any{},
	}
}

func validEnvironmentTuple(environmentID, platform, architecture string) map[string]any {
	return map[string]any{
		"environment_id":           environmentID,
		"environment_version":      "1.0.0",
		"platform":                 platform,
		"architecture":             architecture,
		"store_schema_fingerprint": zeroDigest,
		"adapter_version":          "1.0.0",
	}
}

func declaredScalarPlatformVocabulary(t *testing.T) []string {
	t.Helper()

	_, testSource, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve Session Record test source path")
	}
	scalarSource := filepath.Join(filepath.Dir(testSource), "..", "scalar", "names.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), scalarSource, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	var platforms []string
	for _, declaration := range parsed.Decls {
		constants, ok := declaration.(*ast.GenDecl)
		if !ok || constants.Tok != token.CONST {
			continue
		}
		for _, specification := range constants.Specs {
			values, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range values.Names {
				if !strings.HasPrefix(name.Name, "Platform") || index >= len(values.Values) {
					continue
				}
				literal, ok := values.Values[index].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					t.Fatalf("scalar platform %s is not a string literal", name.Name)
				}
				platform, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				platforms = append(platforms, platform)
			}
		}
	}
	sort.Strings(platforms)
	if len(platforms) != 4 {
		t.Fatalf("derived scalar platform vocabulary = %v, want four members from pinned AX v0.5.0", platforms)
	}
	return platforms
}

func derivedAXSourceIdentityMembers(t *testing.T) []string {
	t.Helper()
	axFixture := validCrossEnvironmentCloneProvenance("ax_session")
	externalFixture := validCrossEnvironmentCloneProvenance("external_native")
	var identityMembers []string
	for name, axValue := range axFixture {
		externalValue, exists := externalFixture[name]
		if exists && axValue != nil && externalValue == nil {
			identityMembers = append(identityMembers, name)
		}
	}
	sort.Strings(identityMembers)
	if len(identityMembers) != 4 {
		t.Fatalf("derived AX-source identity member inventory = %v, want four members from the pinned fixture pair", identityMembers)
	}
	return identityMembers
}

// TestEnvironmentTupleAdapterVersionCarriesNoInferredSemVerConstraint pins the
// first half of this package's recorded SemVer decision, in the direction that
// used to be wrong.
//
// The pinned SPEC v0.5.0 declaration at commit
// 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c reads "Environment Tuple contains
// exactly environment_id, environment_version, platform=linux|macos|windows|wsl2,
// architecture=amd64|arm64, store_schema_fingerprint, and adapter_version". It
// assigns adapter_version no type and no format. The word SemVer reaches
// adapter_version only through the Session Adapter Manifest row "display_name /
// adapter_version | UTF-8 string[1..128] / SemVer", which closes a DIFFERENT
// schema, and through the Probe sentence "Provider ID, manifest digest, and
// adapter version equal the verified Manifest and host values", which names the
// Probe's own top-level members rather than this nested tuple member.
//
// The decision recorded in testdata/constraint-enumeration.md is therefore:
// EnvironmentTuple adapter_version is PRESENCE-ONLY. Re-adding any admission
// grammar to validateEnvironmentTuple reddens the prerelease case below, and
// deleting the member from requireExactMembers reddens the absence case, so the
// decision is pinned in both directions rather than only against deletion.
func TestEnvironmentTupleAdapterVersionCarriesNoInferredSemVerConstraint(t *testing.T) {
	t.Parallel()

	withAdapterVersion := func(value any) []byte {
		provenance := validCrossEnvironmentCloneProvenance("external_native")
		provenance["source_environment"].(map[string]any)["adapter_version"] = value
		return mustJSON(t, validSessionRecordV3Object(provenance))
	}

	// The exact value the bug report names. A core-triple SemVer gate refuses
	// it; the pinned contract does not.
	t.Run("prerelease", func(t *testing.T) {
		assertIdentityEntriesAcceptShape(t, withAdapterVersion("1.2.3-rc.1"), SelfRecordID)
	})

	// Values a core-triple gate, a full SemVer 2.0.0 gate, or a plain string
	// type would each refuse. None of the three is declared here.
	for _, test := range []struct {
		name  string
		value any
	}{
		{"build metadata", "1.2.3+build.1"},
		{"leading zero", "01.2.3"},
		{"two components", "1.2"},
		{"version prefix", "v1.2.3"},
		{"free text", "not-a-version"},
		{"empty", ""},
		{"number", json.Number("1")},
		{"boolean", true},
		{"null", nil},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesAcceptShape(t, withAdapterVersion(test.value), SelfRecordID)
		})
	}

	// Presence is the whole of the declared constraint, and it is enforced.
	// Without this case the decision above would be indistinguishable from
	// having dropped the member entirely. The member is removed from BOTH
	// tuples of the record and the refusal is attributed by name, so dropping
	// "adapter_version" from validateEnvironmentTuple's requireExactMembers
	// list cannot leave this green through the sibling tuple's extra member.
	t.Run("absent", func(t *testing.T) {
		provenance := validCrossEnvironmentCloneProvenance("external_native")
		for _, member := range []string{"source_environment", "target_environment"} {
			delete(provenance[member].(map[string]any), "adapter_version")
		}
		assertIdentityEntriesRefuseWithReason(
			t, mustJSON(t, validSessionRecordV3Object(provenance)), SelfRecordID, "adapter_version")
	})
}

// TestMigrationProvenanceSchemaVersionIsSemVer200InFull pins the second half of
// the same decision, at the site that reuses the same compiled grammar.
//
// Here the constraint IS authorised, by the pinned Section 17.3 sentence quoted
// verbatim: "That extension value is a closed object containing exactly
// schema_id:string, schema_version:semver, and object_id:digest." The document
// names Semantic Version without spelling out a grammar, so `semver` is adopted
// as Semantic Versioning 2.0.0 in full — prerelease and build metadata included.
//
// The two halves are consistent under one rule: the constraint applies exactly
// where the pinned document declares it, and where it is declared it means the
// whole named standard rather than a narrowed subset of it.
func TestMigrationProvenanceSchemaVersionIsSemVer200InFull(t *testing.T) {
	t.Parallel()

	withSchemaVersion := func(value string) []byte {
		object := validSessionRecordV1Object()
		object["extensions"] = map[string]any{
			"works.relux.ax.migrated-from": map[string]any{
				"schema_id":      "urn:ax:schema:session-record",
				"schema_version": value,
				"object_id":      digestWithDigit('3'),
			},
		}
		return mustJSON(t, object)
	}

	for _, accepted := range []struct {
		name  string
		value string
	}{
		{"core triple", "1.2.3"},
		{"prerelease", "1.2.3-rc.1"},
		{"alphanumeric prerelease", "1.0.0-alpha"},
		{"hyphenated prerelease identifier", "1.0.0-x-y-z"},
		{"zero prerelease identifier", "1.0.0-0"},
		{"build metadata", "1.2.3+build.1"},
		{"prerelease and build metadata", "1.2.3-rc.1+exp.sha.5114f85"},
	} {
		accepted := accepted
		t.Run("accepts "+accepted.name, func(t *testing.T) {
			assertIdentityEntriesAcceptShape(t, withSchemaVersion(accepted.value), SelfRecordID)
		})
	}

	// The gate must still refuse. Widening it to SemVer 2.0.0 is not the same
	// as deleting it, and only these cases tell the two apart.
	for _, refused := range []struct {
		name  string
		value string
	}{
		{"leading zero major", "01.2.3"},
		{"missing component", "1.2"},
		{"version prefix", "v1.2.3"},
		{"empty prerelease", "1.2.3-"},
		{"leading-zero numeric prerelease identifier", "1.2.3-01"},
		{"empty prerelease identifier", "1.2.3-a..b"},
		{"underscore in prerelease", "1.2.3-a_b"},
		{"empty build metadata", "1.2.3+"},
		{"underscore in build metadata", "1.2.3+a_b"},
		{"non-ASCII digit", "１.2.3"},
		{"free text", "not-a-version"},
		{"empty", ""},
	} {
		refused := refused
		t.Run("refuses "+refused.name, func(t *testing.T) {
			assertIdentityEntriesRefuseShape(t, withSchemaVersion(refused.value), SelfRecordID)
		})
	}
}
