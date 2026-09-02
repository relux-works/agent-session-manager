package canonicaljson

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// identityFixture is one valid candidate object for a production shape, named
// by the entry point that owns it.
//
// selfField selects the identity entry pair CalculateObjectIdentity /
// VerifyObjectIdentity. An empty selfField selects ValidateObservationEvent:
// Section 18.1 objects are not identity-addressed and have their own entry.
type identityFixture struct {
	name      string
	selfField SelfField
	object    map[string]any
}

// everyValidIdentityFixture returns one accepted candidate per production shape
// this package validates completely.
//
// The set is not trusted on its word:
// TestEveryCompletelyValidatedSchemaVersionHasAValidFixture derives the required
// schema/version set from the production validator registry and requires this
// function to cover it, so registering a new complete validator without a
// fixture reddens the suite instead of silently leaving the new shape outside
// every derived sweep that walks these fixtures.
func everyValidIdentityFixture() []identityFixture {
	fixtures := []identityFixture{
		{"session record v1", SelfRecordID, validSessionRecordV1Object()},
		{"session record v1 primary owner", SelfRecordID, primaryOwnerSessionRecord("GOAL-260830-primary")},
		{"session record v1 task board", SelfRecordID, taskBoardSessionRecord("TASK-260830-8x76g1")},
		{"session record v1 fork provenance", SelfRecordID, sessionRecordWithForkProvenance()},
		{"session record v1 migration provenance", SelfRecordID, sessionRecordWithMigrationProvenance()},
		{"session record v2 origin", SelfRecordID, validSessionRecordV2Object(validOriginProvenance())},
		{"session record v2 same-provider fork", SelfRecordID, validSessionRecordV2Object(validSameProviderForkProvenance())},
		{"session record v2 cross-environment clone", SelfRecordID, validSessionRecordV2Object(validCrossEnvironmentCloneProvenance("external_native"))},
		{"session record v3 native adoption", SelfRecordID, validSessionRecordV3Object(validNativeAdoptionProvenance())},
		{"blob descriptor", SelfDescriptorID, validBlobDescriptorObject()},
		{"transfer manifest composite", SelfManifestID, compositeManifestWithChildren(2)},
		{"transfer manifest provider", SelfManifestID, providerTransferManifestObject()},
		{"transfer manifest task board", SelfManifestID, taskBoardTransferManifestObject()},
		{"transfer manifest workspace group", SelfManifestID, validTransferManifestObject("workspace_group")},
		{"transfer manifest workspace tree", SelfManifestID, workspaceTreeWithEveryEntryVariant()},
		{"transfer manifest git workspace", SelfManifestID, gitWorkspaceWithInitializedSubmodule()},
		{"lease", SelfRecordID, validLeaseRecordObject()},
		{"checkpoint direct", SelfCheckpointID, validCheckpointRecordObject(true)},
		{"checkpoint task board", SelfCheckpointID, validCheckpointRecordObject(false)},
		{"provider identity", SelfRecordID, validProviderIdentityRecordObject()},
		{"workspace group", SelfRecordID, validWorkspaceGroupRecordObject()},
		{"observation event", "", validObservationEventObject()},
	}
	for _, event := range catalog.Current().Events {
		if event.Family != "session_event" {
			continue
		}
		for _, version := range event.ContractVersions {
			name := string(event.Name)
			fixtures = append(fixtures, identityFixture{
				name:      fmt.Sprintf("session event %s %s", version, name),
				selfField: SelfEventID,
				object:    validSessionEventObject(version, name),
			})
		}
	}
	return fixtures
}

func providerTransferManifestObject() map[string]any {
	object := validTransferManifestObject("provider")
	object["provider_identity_record_id"] = digestWithDigit('6')
	return object
}

func taskBoardTransferManifestObject() map[string]any {
	object := validTransferManifestObject("task_board")
	object["task_board_bundle_id"] = digestWithDigit('7')
	return object
}

// TestEveryValidIdentityFixtureIsAcceptedAtItsProductionEntry is the sanity
// floor for every derived sweep built on these fixtures. A sweep that mutates a
// fixture and asserts refusal proves nothing if the unmutated fixture is
// already refused for an unrelated reason.
func TestEveryValidIdentityFixtureIsAcceptedAtItsProductionEntry(t *testing.T) {
	for _, fixture := range everyValidIdentityFixture() {
		t.Run(fixture.name, func(t *testing.T) {
			candidate := mustJSON(t, fixture.object)
			if fixture.selfField == "" {
				if err := ValidateObservationEvent(candidate); err != nil {
					t.Fatalf("ValidateObservationEvent(%s) error = %v, want acceptance", fixture.name, err)
				}
				return
			}
			assertIdentityEntriesAcceptShape(t, candidate, fixture.selfField)
		})
	}
}

// TestEveryCompletelyValidatedSchemaVersionHasAValidFixture derives the
// obligation from production rather than from a list of shape names.
//
// A registered validator either has a valid candidate or it has none. The
// exempt set is not named here: deriveTotalRefusalValidators walks the package
// sources and returns the functions whose every return is a refusal, directly or
// through another total-refusal function, so both the outright
// rejectUnsupportedImmutableObjectShape rows and the
// validateUnsupportedRecordEnvelopeShape rows fall out of the derivation
// instead of being excused by name. Every other registered schema/version must
// appear in everyValidIdentityFixture, so registering a new complete validator
// without a fixture reddens here rather than leaving the new shape outside every
// derived sweep.
func TestEveryCompletelyValidatedSchemaVersionHasAValidFixture(t *testing.T) {
	t.Parallel()

	totallyRefusing := deriveTotalRefusalValidators(t)
	registered := deriveRegisteredShapeValidators(t)

	required := make(map[schemaIdentityKey]struct{})
	for key, validatorName := range registered {
		if totallyRefusing[validatorName] {
			continue
		}
		required[key] = struct{}{}
	}
	if len(required) == 0 {
		t.Fatal("derived zero completely validated schema versions; the derivation is broken, not the package")
	}

	covered := make(map[schemaIdentityKey]struct{})
	for _, fixture := range everyValidIdentityFixture() {
		schema, _ := fixture.object["schema"].(string)
		version, _ := fixture.object["schema_version"].(string)
		covered[schemaIdentityKey{schema: schema, version: version}] = struct{}{}
	}

	var missing []string
	for key := range required {
		if _, ok := covered[key]; !ok {
			missing = append(missing, key.schema+"@"+key.version)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf(
			"completely validated schema versions with no valid fixture in everyValidIdentityFixture:\n  %s",
			strings.Join(missing, "\n  "),
		)
	}
}

// deriveRegisteredShapeValidators reads the production validator registry from
// its own source: every register(...) call in mustBuildImmutableObjectShapeValidators
// yields the schema, the validator function name, and the registered versions.
//
// It is cross-checked against the runtime table, so an unread or misread call
// fails here instead of silently shrinking the obligation set above.
func deriveRegisteredShapeValidators(t *testing.T) map[schemaIdentityKey]string {
	t.Helper()

	_, files := parsedProductionPackage(t)
	constants := derivePackageStringConstants(files)

	registered := make(map[schemaIdentityKey]string)
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil || function.Name.Name != "mustBuildImmutableObjectShapeValidators" {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				identifier, ok := call.Fun.(*ast.Ident)
				if !ok || identifier.Name != "register" || len(call.Args) < 3 {
					return true
				}
				schema := stringArgumentValue(call.Args[0], constants)
				validator, validatorOK := call.Args[1].(*ast.Ident)
				if schema == "" || !validatorOK {
					t.Fatalf("register call in %s does not spell a literal schema and a named validator", function.Name.Name)
				}
				for _, argument := range call.Args[2:] {
					version := stringArgumentValue(argument, constants)
					if version == "" {
						t.Fatalf("register call for %s does not spell a literal version", schema)
					}
					registered[schemaIdentityKey{schema: schema, version: version}] = validator.Name
				}
				return true
			})
		}
	}
	if len(registered) != len(immutableObjectShapeValidators) {
		t.Fatalf(
			"derived %d registered shape validators from source, the runtime table has %d; the derivation misses rows",
			len(registered), len(immutableObjectShapeValidators),
		)
	}
	for key := range registered {
		if _, ok := immutableObjectShapeValidators[key]; !ok {
			t.Fatalf("derived a registration for %s@%s that the runtime table does not carry", key.schema, key.version)
		}
	}
	return registered
}

func stringArgumentValue(expression ast.Expr, constants map[string]string) string {
	switch typed := expression.(type) {
	case *ast.BasicLit:
		unquoted, err := strconv.Unquote(typed.Value)
		if err != nil {
			return ""
		}
		return unquoted
	case *ast.Ident:
		return constants[typed.Name]
	}
	return ""
}

// derivePackageStringConstants resolves package-level untyped string constants
// so a schema or version spelled as a named constant still reads as a literal.
func derivePackageStringConstants(files []*ast.File) map[string]string {
	constants := make(map[string]string)
	for _, file := range files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			for _, specification := range general.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, name := range value.Names {
					if index >= len(value.Values) {
						continue
					}
					literal, ok := value.Values[index].(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						continue
					}
					if unquoted, err := strconv.Unquote(literal.Value); err == nil {
						constants[name.Name] = unquoted
					}
				}
			}
		}
	}
	return constants
}
