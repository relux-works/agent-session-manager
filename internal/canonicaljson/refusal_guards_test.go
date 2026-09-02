package canonicaljson

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// This file is the self-proving coverage gate for every refusal this package
// can emit at runtime.
//
// The failure it exists to prevent, in the reviewer's words: eight normative
// gates removed from core_records.go simultaneously left `go test ./...` fully
// green, seeded fuzz corpora and tracecheck included, and a statement-coverage
// profile showed 91 runtime refusal branches in that one file that no test ever
// executed. Package coverage was 87.7% at the time, which is exactly why a
// percentage is not evidence.
//
// The cause is structural rather than a list of missed cases. requireExactMembers
// runs first in every validator, so a fixture that omits a member short-circuits
// on the closed-member sweep and never reaches the per-member type, format or
// coupling refusal it claims to pin. The suite therefore proved the closed
// member set and the declared range bounds thoroughly and proved almost nothing
// about what each member must contain.
//
// The gate below derives its own subject the way declared_bounds_test.go derives
// bounds. It walks every production source of the package, derives the refusal
// constructors rather than naming them, turns every return statement that can
// emit a refusal into one obligation, and then requires each obligation to have
// been EXECUTED by the shipped suite — observed from a real coverage profile,
// not from a claim in a table. A refusal guard added without a negative case
// reddens the suite, and so does deleting the case that reached an existing one.
//
// Obligations that no candidate can reach are declared in unreachableRefusalSites
// with the reason and the refusal that subsumes them. That set is asserted
// exactly in both directions: a declaration that becomes reachable reddens just
// as loudly as a guard that loses its case, so "unreachable" cannot be used to
// wave a live gate through.

// refusalCoverageChildEnvironment marks the child `go test` run that produces
// the coverage profile. The gate skips itself there, so the child observes the
// shipped suite and the parent does not recurse.
const refusalCoverageChildEnvironment = "AX_CANONICALJSON_REFUSAL_COVERAGE_CHILD"

// refusalSite is one production return statement that can emit a refusal.
type refusalSite struct {
	file     string
	line     int
	column   int
	function string
	// text is the source of the return statement, so a moved guard is
	// recognisable in the failure message.
	text string
	// ordinal disambiguates the several identical `return err` guards a single
	// validator carries, counted in source order within its function.
	ordinal int
}

func (site refusalSite) key() string {
	return fmt.Sprintf("%s:%d %s", site.file, site.line, site.function)
}

// unreachableRefusal declares why a derived obligation cannot be reached and
// names both the production refusal that fires first and the test that pins it.
// Naming the subsuming refusal is what keeps the escape hatch honest: an
// obligation is not waived, its enforcement is relocated to a gate that is
// itself proven.
type unreachableRefusal struct {
	reason           string
	subsumingRefusal string
	provingTest      string
}

// unreachableRefusalSites declares the derived obligations that no candidate can
// reach through any production entry point. Keys are `file|function|returned
// source text`, so a declaration survives unrelated line movement but breaks
// when the guard itself is rewritten.
//
// The set is asserted exactly by TestEveryProductionRefusalGuardIsExecuted: an
// entry that stops being unreachable fails as an obsolete waiver, and an entry
// naming a guard that no longer exists fails the same way.
var unreachableRefusalSites = map[string]unreachableRefusal{
	// GROUP 1 - the member is absent.
	//
	// requireExactMembers runs first in every validator, so no member can be
	// missing by the time these helpers read it. That short-circuit is exactly
	// the structural cause the review identified; it is recorded here rather
	// than papered over, and the closed-member sweep that subsumes it is named.
	"closed_shapes.go|requireBool|return false, invalidIdentity(\"identity input requires member %s\", name)|#0":                   absentMemberRefusal,
	"closed_shapes.go|nullableString|return \"\", false, invalidIdentity(\"identity input requires member %s\", name)|#0":          absentMemberRefusal,
	"closed_shapes.go|requireObject|return nil, invalidIdentity(\"identity input requires member %s\", name)|#0":                   absentMemberRefusal,
	"closed_shapes.go|requireArray|return nil, invalidIdentity(\"identity input requires member %s\", name)|#0":                    absentMemberRefusal,
	"closed_shapes.go|requireNullableDigest|return invalidIdentity(\"identity input requires member %s\", name)|#0":                absentMemberRefusal,
	"closed_shapes.go|requireNullableDigestPresence|return false, invalidIdentity(\"identity input requires member %s\", name)|#0": absentMemberRefusal,
	"closed_shapes.go|validateSessionTaskBoardReference|return invalidIdentity(\"identity input requires member task_board\")|#0":  absentMemberRefusal,
	"closed_shapes.go|validateSessionTaskBoardReference|return invalidIdentity(\"identity input requires member board_goal\")|#0":  absentMemberRefusal,
	"closed_shapes.go|validateSessionBoardIdentity|return invalidIdentity(\"identity input requires member remote_url\")|#0":       absentMemberRefusal,
	"closed_shapes.go|validateSessionForkProvenance|return invalidIdentity(\"identity input requires member fork_provenance\")|#0": absentMemberRefusal,
	"core_records.go|nullMember|return invalidIdentity(\"identity input requires member %s\", name)|#0":                            absentMemberRefusal,
	"core_records.go|validateObservationCountsMember|return invalidIdentity(\"identity input requires member counts\")|#0":         absentMemberRefusal,
	"core_records.go|requireArrayMinimum|return nil, invalidIdentity(\"identity input requires member %s\", name)|#0":              absentMemberRefusal,
	"core_records.go|nullablePositiveUint|return false, invalidIdentity(\"identity input requires member %s\", name)|#0":           absentMemberRefusal,
	"core_records.go|nullableUintPresence|return 0, false, invalidIdentity(\"identity input requires member %s\", name)|#0":        absentMemberRefusal,
	"core_records.go|requireNullableUUIDv4|return false, invalidIdentity(\"identity input requires member %s\", name)|#0":          absentMemberRefusal,

	// GROUP 2 - the schema and version were the selection key.
	//
	// validateImmutableObjectShape looks the validator up BY schema and version,
	// so a validator re-reading either member can never see a different value.
	"closed_shapes.go|validateImmutableObjectShape|return err|#0":                  selectedSchemaRefusal,
	"closed_shapes.go|validateImmutableObjectShape|return err|#1":                  selectedSchemaRefusal,
	"closed_shapes.go|validateSessionRecordCommon|return \"\", \"\", \"\", err|#0": selectedSchemaRefusal,
	"closed_shapes.go|validateSessionRecordCommon|return \"\", \"\", \"\", err|#1": selectedSchemaRefusal,
	"closed_shapes.go|validateBlobDescriptor|return err|#1":                        selectedSchemaRefusal,
	"closed_shapes.go|validateBlobDescriptor|return err|#2":                        selectedSchemaRefusal,
	"closed_shapes.go|validateTransferManifest|return err|#1":                      selectedSchemaRefusal,
	"closed_shapes.go|validateTransferManifest|return err|#2":                      selectedSchemaRefusal,
	"core_records.go|validateIdentityRecordEnvelope|return err|#0":                 selectedSchemaRefusal,
	"core_records.go|validateIdentityRecordEnvelope|return err|#1":                 selectedSchemaRefusal,

	// GROUP 3 - the self field was already parsed as a digest.
	//
	// resolveSelfField reads the claimed identity through scalar.ParseDigest
	// before any shape validator runs, so a validator re-reading the same member
	// as a digest cannot fail.
	"closed_shapes.go|validateSessionRecordCommon|return \"\", \"\", \"\", err|#2": parsedSelfFieldRefusal,
	"closed_shapes.go|validateBlobDescriptor|return err|#3":                        parsedSelfFieldRefusal,
	"closed_shapes.go|validateTransferManifest|return err|#3":                      parsedSelfFieldRefusal,
	"core_records.go|validateIdentityRecordEnvelope|return err|#2":                 parsedSelfFieldRefusal,

	// GROUP 4 - subject_id was already parsed by the common record envelope.
	//
	// Every one of these validators calls validateCommonRecordEnvelope, which
	// reads subject_id through requireUUIDv7, before reading subject_id again to
	// compare it. The second read cannot fail.
	"closed_shapes.go|validateSessionRecordCommon|return \"\", \"\", \"\", err|#4": envelopeSubjectRefusal,
	"core_records.go|validateLeaseRecord|return err|#2":                            envelopeSubjectRefusal,
	"core_records.go|validateCheckpointRecord|return err|#2":                       envelopeSubjectRefusal,
	"core_records.go|validateProviderIdentityRecord|return err|#2":                 envelopeSubjectRefusal,
	"core_records.go|validateWorkspaceGroupRecord|return err|#2":                   envelopeSubjectRefusal,
	"core_records.go|validateSessionEvent|return err|#2":                           envelopeSubjectRefusal,

	// GROUP 5 - the union was dispatched on the same tag.
	//
	// validateSessionDerivationProvenance switches on derivation_provenance.kind
	// and calls the arm named by that value, so the arm's own kind re-check
	// cannot see a different value; an undeclared kind is refused by the switch
	// default before any arm runs.
	"closed_shapes.go|validateSessionOriginProvenance|return err|#1":                {dispatchedUnionTag, "Session Record derivation_provenance.kind %q is not a closed %s union member", "TestSessionRecordDerivationProvenanceRefusesAnUndeclaredKind"},
	"closed_shapes.go|validateSessionSameProviderForkProvenance|return err|#1":      {dispatchedUnionTag, "Session Record derivation_provenance.kind %q is not a closed %s union member", "TestSessionRecordDerivationProvenanceRefusesAnUndeclaredKind"},
	"closed_shapes.go|validateSessionCrossEnvironmentCloneProvenance|return err|#1": {dispatchedUnionTag, "Session Record derivation_provenance.kind %q is not a closed %s union member", "TestSessionRecordDerivationProvenanceRefusesAnUndeclaredKind"},
	"closed_shapes.go|validateSessionNativeAdoptionProvenance|return err|#1":        {dispatchedUnionTag, "Session Record derivation_provenance.kind %q is not a closed %s union member", "TestSessionRecordDerivationProvenanceRefusesAnUndeclaredKind"},

	// GROUP 6 - re-serializing a value the strict decoder already produced.
	//
	// decodeStrict yields only nil, bool, string, json.Number, []any and
	// map[string]any, every one of which encoding/json accepts, and every
	// json.Number carries a token the decoder accepted. So neither json.Marshal
	// nor the RFC 8785 transform can fail on a decoded value.
	"canonical.go|Canonicalize|return nil, invalidJSON(\"serialize logical JSON value: %v\", err)|#0":                                           decodedValueRefusal,
	"canonical.go|calculatePreparedObjectIdentity|return scalar.Digest{}, \"\", invalidIdentity(\"serialize omit-self object: %v\", err)|#0":    decodedValueRefusal,
	"canonical.go|calculatePreparedObjectIdentity|return scalar.Digest{}, \"\", invalidIdentity(\"canonicalize omit-self object: %v\", err)|#0": decodedValueRefusal,
	"closed_shapes.go|validateExtensionsObject|return invalidIdentity(\"serialize extensions object: %v\", err)|#0":                             decodedValueRefusal,
	"closed_shapes.go|validateExtensionsObject|return invalidIdentity(\"canonicalize extensions object: %v\", err)|#0":                          decodedValueRefusal,
	"closed_shapes.go|validateSessionLaunchPlan|return invalidIdentity(\"serialize Session Record Launch Plan argv: %v\", err)|#0":              decodedValueRefusal,
	// VerifyObjectIdentity propagates only the two calculatePreparedObjectIdentity
	// refusals above; its own prepare-error propagation is proven.
	"canonical.go|VerifyObjectIdentity|return scalar.Digest{}, \"\", err|#1": decodedValueRefusal,

	// GROUP 7 - the decoder cannot produce the value the guard rejects.
	"canonical.go|decodeValue|return nil, invalidJSON(\"object name is not a string\")|#0": {
		reason:           "encoding/json emits object names only as string tokens, and decoder.More() guards the closing delimiter",
		subsumingRefusal: "decode object name: %v",
		provingTest:      "TestCalculateObjectIdentityRefusesMalformedJSONInput",
	},
	"canonical.go|decodeValue|return nil, invalidJSON(\"unexpected delimiter %q\", delimiter)|#0": {
		reason:           "the switch is entered only for a { or [ delimiter; a leading } or ] is refused by decoder.Token first",
		subsumingRefusal: "decode JSON token: %v",
		provingTest:      "TestCalculateObjectIdentityRefusesMalformedJSONInput",
	},
	"closed_shapes.go|validateExtensionValue|return invalidIdentity(\"string value must be valid UTF-8\")|#0": invalidUTF8Refusal,
	"closed_shapes.go|validateExtensionValue|return invalidIdentity(\"object key must be valid UTF-8\")|#0":   invalidUTF8Refusal,
	"closed_shapes.go|validateExtensionValue|return invalidIdentity(\"value uses unsupported JSON type %T\", value)|#0": {
		reason:           "decodeStrict produces only nil, bool, json.Number, string, []any and map[string]any, and the switch handles every one",
		subsumingRefusal: "invalid canonical JSON input",
		provingTest:      "TestCalculateObjectIdentityRefusesMalformedJSONInput",
	},

	// GROUP 8 - a subsuming numeric bound makes the arithmetic guard unreachable.
	"closed_shapes.go|validateBlobDescriptor|return invalidIdentity(\"BlobChunk coverage exceeds uint53\")|#0": {
		reason: "chunks is bounded at 32768 entries and each chunk size at 4194304 bytes, so the greatest reachable coverage " +
			"is 137438953472, four orders of magnitude below scalar.MaxUint53; both bounds are proven at their limits",
		subsumingRefusal: "member chunks exceeds maximum length 32768",
		provingTest:      "TestEveryDeclaredBoundCallSiteIsProvenInBothDirections",
	},

	// GROUP 9 - a row-count check makes the membership check unreachable.
	"closed_shapes.go|validateImmutableObjectShapeValidators|return fmt.Errorf(\"immutable-object shape validator for unregistered schema %s@%s\", key.schema, key.version)|#0": {
		reason: "the loop above requires every generated row to be present, so an extra row always makes the table longer " +
			"than the generated registry and the row-count refusal fires first",
		subsumingRefusal: "immutable-object shape validator table has %d rows, generated registry has %d",
		provingTest:      "TestImmutableObjectShapeValidatorRegistryRefusesADivergentTable",
	},

	// GROUP 10 - the table is built from the definitions it is validated against.
	"canonical.go|buildSchemaIdentityContracts|return nil, err|#0": {
		reason: "the contracts map is built from the same definitions the cross-check compares it against, so it can only " +
			"diverge by a duplicate key, which the guard directly above refuses first",
		subsumingRefusal: "duplicate self-identity contract for %s@%s",
		provingTest:      "TestBuildSchemaIdentityContractsRefusesADuplicateContract",
	},
}

// The shared declarations below carry one reason per structural cause, so a
// class of unreachable guards is explained once rather than repeated per site.
var (
	absentMemberRefusal = unreachableRefusal{
		reason:           "requireExactMembers refuses an omitted member before any validator reads it, so the member is always present here",
		subsumingRefusal: "%s is missing required member %q",
		provingTest:      "TestCoreRecordClosedShapesRefuseMissingAndUnknownMembersAtProductionEntries",
	}
	selectedSchemaRefusal = unreachableRefusal{
		reason:           "the validator is looked up by schema and schema_version, so re-reading either member cannot see a different value",
		subsumingRefusal: "no immutable-object shape validator for %s@%s",
		provingTest:      "TestResolveSelfFieldRefusesAMalformedDiscriminatorMember",
	}
	parsedSelfFieldRefusal = unreachableRefusal{
		reason:           "resolveSelfField parses the claimed self field as a digest before any shape validator runs",
		subsumingRefusal: "self field %s: %v",
		provingTest:      "TestEveryFixtureMemberRefusesAWrongJSONTypeAtItsProductionEntry",
	}
	envelopeSubjectRefusal = unreachableRefusal{
		reason:           "validateCommonRecordEnvelope reads subject_id through requireUUIDv7 before this second read of the same member",
		subsumingRefusal: "member subject_id: %v",
		provingTest:      "TestEveryStructuredFixtureValueRefusesAMalformedFormAtItsProductionEntry",
	}
	decodedValueRefusal = unreachableRefusal{
		reason:           "the value was produced by decodeStrict, so encoding/json and the RFC 8785 transform both accept it",
		subsumingRefusal: "invalid canonical JSON input",
		provingTest:      "TestCalculateObjectIdentityRefusesMalformedJSONInput",
	}
	invalidUTF8Refusal = unreachableRefusal{
		reason:           "decodeStrict refuses input that is not valid UTF-8, so no decoded string or object key can be invalid",
		subsumingRefusal: "input is not valid UTF-8",
		provingTest:      "TestCalculateObjectIdentityRefusesMalformedJSONInput",
	}
	dispatchedUnionTag = "validateSessionDerivationProvenance dispatches on the same kind value, so the arm's re-check cannot see a different one"
)

// TestUnreachableRefusalDeclarationsNameATestThatExists keeps the escape hatch
// honest the same way the declared-bounds subsumptions are kept honest: a
// declaration must point at a test function present in this package.
func TestUnreachableRefusalDeclarationsNameATestThatExists(t *testing.T) {
	t.Parallel()

	names := packageTestFunctionNames(t)
	for key, declaration := range unreachableRefusalSites {
		if declaration.reason == "" || declaration.subsumingRefusal == "" || declaration.provingTest == "" {
			t.Errorf("unreachable refusal %s must declare a reason, a subsuming refusal and a proving test", key)
			continue
		}
		if _, ok := names[declaration.provingTest]; !ok {
			t.Errorf("unreachable refusal %s names %s, which does not exist in this package", key, declaration.provingTest)
		}
	}
}

// TestEveryProductionRefusalGuardIsExecuted is the coverage assertion.
//
// It derives every refusal-emitting return in the package's production sources,
// runs the shipped suite under a coverage profile, and requires each derived
// site to have executed at least once. It fails when a guard has no negative
// case, and when a declared-unreachable guard turns out to be reachable.
func TestEveryProductionRefusalGuardIsExecuted(t *testing.T) {
	if os.Getenv(refusalCoverageChildEnvironment) != "" {
		t.Skip("child coverage run: the gate measures the shipped suite, it is not part of it")
	}

	sites := deriveRefusalSites(t)
	profile := runShippedSuiteUnderCoverage(t)

	executed := make(map[string]bool, len(sites))
	for _, site := range sites {
		block, ok := profile.innermostBlock(site)
		if !ok {
			t.Fatalf(
				"refusal guard %s has no coverage block; the derivation and the profile disagree, "+
					"which makes every count below meaningless",
				site.key(),
			)
		}
		executed[site.key()] = block.count > 0
	}

	var unexecuted, obsolete []string
	for _, site := range sites {
		declaration, declared := unreachableRefusalSites[site.declarationKey()]
		switch {
		case executed[site.key()] && declared:
			obsolete = append(obsolete, fmt.Sprintf("%s: declared unreachable (%s) but executed", site.key(), declaration.reason))
		case !executed[site.key()] && !declared:
			unexecuted = append(unexecuted, fmt.Sprintf("%s: %s\n      declaration key: %s", site.key(), site.text, site.declarationKey()))
		}
	}
	declared := make(map[string]struct{}, len(sites))
	for _, site := range sites {
		declared[site.declarationKey()] = struct{}{}
	}
	for key := range unreachableRefusalSites {
		if _, ok := declared[key]; !ok {
			obsolete = append(obsolete, fmt.Sprintf("%s: declared unreachable but no such refusal guard exists", key))
		}
	}
	sort.Strings(unexecuted)
	sort.Strings(obsolete)

	if len(unexecuted) > 0 {
		t.Errorf(
			"%d production refusal guard(s) that the shipped suite never executed. "+
				"Each one can be deleted or narrowed with the whole suite still green. "+
				"Supply a COMPLETE valid member set and violate exactly one member type, format or coupling — "+
				"a fixture that omits a member stops at requireExactMembers and proves only the completeness check:\n  %s",
			len(unexecuted), strings.Join(unexecuted, "\n  "),
		)
	}
	if len(obsolete) > 0 {
		t.Errorf("obsolete unreachable-refusal declarations:\n  %s", strings.Join(obsolete, "\n  "))
	}
}

// declarationKey identifies a site independently of its line number, so an
// unrelated edit above it does not invalidate a declaration while a rewrite of
// the guard itself does.
func (site refusalSite) declarationKey() string {
	return fmt.Sprintf("%s|%s|%s|#%d", site.file, site.function, site.text, site.ordinal)
}

// TestRefusalGuardDerivationFindsTheKnownRefusalConstructors is the derivation's
// own floor. The obligation set is only as honest as the constructor set it is
// built from: if the derivation silently stopped recognising invalidIdentity,
// every obligation would vanish and the gate above would pass by finding
// nothing.
func TestRefusalGuardDerivationFindsTheKnownRefusalConstructors(t *testing.T) {
	t.Parallel()

	constructors := deriveRefusalConstructors(t)
	for _, name := range []string{"invalidIdentity", "invalidObservation", "invalidJSON"} {
		if !constructors[name] {
			t.Errorf("refusal-constructor derivation did not find %s; every obligation below it is unsound", name)
		}
	}
	sites := deriveRefusalSites(t)
	if len(sites) < 100 {
		t.Errorf("derived only %d refusal guards for this package; the scanner is broken, not the package", len(sites))
	}
	files := make(map[string]int)
	for _, site := range sites {
		files[site.file]++
	}
	for _, name := range []string{"canonical.go", "closed_shapes.go", "core_records.go"} {
		if files[name] == 0 {
			t.Errorf("derived zero refusal guards in %s", name)
		}
	}
}

// TestRefusalGuardCoverageMatchingReportsAnUnexecutedGuard proves the matching
// itself rather than trusting it. A synthetic profile that marks one guard's
// block executed and another's not must resolve to exactly that, so the gate
// above cannot pass by mapping every site onto a covered enclosing block.
func TestRefusalGuardCoverageMatchingReportsAnUnexecutedGuard(t *testing.T) {
	t.Parallel()

	source := `package sample

func guarded(value int) error {
	if value == 1 {
		return errOne
	}
	if value == 2 {
		return errTwo
	}
	return nil
}
`
	directory := t.TempDir()
	path := filepath.Join(directory, "sample.go")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var returns []refusalSite
	ast.Inspect(parsed, func(node ast.Node) bool {
		statement, ok := node.(*ast.ReturnStmt)
		if !ok || len(statement.Results) != 1 {
			return true
		}
		identifier, ok := statement.Results[0].(*ast.Ident)
		if !ok || !strings.HasPrefix(identifier.Name, "err") {
			return true
		}
		position := fileSet.Position(statement.Pos())
		returns = append(returns, refusalSite{
			file: "sample.go", line: position.Line, column: position.Column,
			function: "guarded", text: "return " + identifier.Name,
		})
		return true
	})
	if len(returns) != 2 {
		t.Fatalf("expected 2 synthetic refusal returns, got %d", len(returns))
	}

	// The `value == 1` branch executed; the `value == 2` branch did not.
	profile := &coverageProfile{blocks: map[string][]coverageBlock{
		"sample.go": {
			{startLine: 3, startColumn: 24, endLine: 4, endColumn: 16, count: 7},
			{startLine: 4, startColumn: 16, endLine: 6, endColumn: 3, count: 3},
			{startLine: 7, startColumn: 16, endLine: 9, endColumn: 3, count: 0},
		},
	}}
	first, ok := profile.innermostBlock(returns[0])
	if !ok || first.count == 0 {
		t.Fatalf("executed synthetic guard resolved to %+v (found=%v), want a block with a non-zero count", first, ok)
	}
	second, ok := profile.innermostBlock(returns[1])
	if !ok {
		t.Fatal("unexecuted synthetic guard resolved to no block; the matcher would silently drop obligations")
	}
	if second.count != 0 {
		t.Fatalf("unexecuted synthetic guard resolved to a block with count %d; the matcher reports covered for an uncovered guard", second.count)
	}
}

// runShippedSuiteUnderCoverage runs this package's own tests in a child process
// under a statement-coverage profile and returns the parsed profile.
//
// A missing toolchain is a hard failure, never a skip: a gate that disappears
// when it cannot measure is a bypass path around itself.
func runShippedSuiteUnderCoverage(t *testing.T) *coverageProfile {
	t.Helper()

	directory, _ := packageProductionFiles(t)
	profilePath := filepath.Join(t.TempDir(), "refusal-coverage.out")
	command := exec.Command("go", "test", "-count=1", "-covermode=count", "-coverprofile="+profilePath, ".")
	command.Dir = directory
	command.Env = append(os.Environ(), refusalCoverageChildEnvironment+"=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("shipped suite under coverage failed, so no refusal guard below is measured:\n%s\n%v", output, err)
	}
	return parseCoverageProfile(t, profilePath)
}

type coverageBlock struct {
	startLine, startColumn int
	endLine, endColumn     int
	count                  int
}

type coverageProfile struct {
	blocks map[string][]coverageBlock
}

// innermostBlock returns the smallest profiled block containing the site. Go's
// cover tool emits one block per basic block, so the guard body containing a
// refusal return is its own block; taking the smallest container avoids
// resolving a guard onto its enclosing function block, which would report every
// guard in a called function as executed.
func (profile *coverageProfile) innermostBlock(site refusalSite) (coverageBlock, bool) {
	var best coverageBlock
	found := false
	bestSize := 0
	for _, block := range profile.blocks[site.file] {
		afterStart := site.line > block.startLine || (site.line == block.startLine && site.column >= block.startColumn)
		beforeEnd := site.line < block.endLine || (site.line == block.endLine && site.column <= block.endColumn)
		if !afterStart || !beforeEnd {
			continue
		}
		size := (block.endLine-block.startLine)*1_000 + (block.endColumn - block.startColumn)
		if !found || size < bestSize {
			best, bestSize, found = block, size, true
		}
	}
	return best, found
}

func parseCoverageProfile(t *testing.T, path string) *coverageProfile {
	t.Helper()

	handle, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()

	profile := &coverageProfile{blocks: make(map[string][]coverageBlock)}
	scanner := bufio.NewScanner(handle)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") || strings.TrimSpace(line) == "" {
			continue
		}
		separator := strings.LastIndex(line, ":")
		if separator < 0 {
			t.Fatalf("malformed coverage line %q", line)
		}
		name := filepath.Base(line[:separator])
		fields := strings.Fields(line[separator+1:])
		if len(fields) != 3 {
			t.Fatalf("malformed coverage line %q", line)
		}
		span := strings.Split(fields[0], ",")
		if len(span) != 2 {
			t.Fatalf("malformed coverage span %q", fields[0])
		}
		start, end := strings.Split(span[0], "."), strings.Split(span[1], ".")
		if len(start) != 2 || len(end) != 2 {
			t.Fatalf("malformed coverage span %q", fields[0])
		}
		block := coverageBlock{
			startLine:   mustAtoi(t, start[0]),
			startColumn: mustAtoi(t, start[1]),
			endLine:     mustAtoi(t, end[0]),
			endColumn:   mustAtoi(t, end[1]),
			count:       mustAtoi(t, fields[2]),
		}
		profile.blocks[name] = append(profile.blocks[name], block)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(profile.blocks) == 0 {
		t.Fatal("coverage profile carries no blocks; nothing below is measured")
	}
	return profile
}

func mustAtoi(t *testing.T, value string) int {
	t.Helper()
	number, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	return number
}

var (
	productionASTOnce  sync.Once
	productionFileSet  *token.FileSet
	productionASTFiles []*ast.File
	productionASTError error
)

// parsedProductionPackage parses every production source of this package once,
// keeping the FileSet so refusal sites carry real positions.
func parsedProductionPackage(t *testing.T) (*token.FileSet, []*ast.File) {
	t.Helper()

	directory, paths := packageProductionFiles(t)
	_ = directory
	productionASTOnce.Do(func() {
		fileSet := token.NewFileSet()
		for _, path := range paths {
			parsed, err := parser.ParseFile(fileSet, path, nil, 0)
			if err != nil {
				productionASTError = err
				return
			}
			productionASTFiles = append(productionASTFiles, parsed)
		}
		productionFileSet = fileSet
	})
	if productionASTError != nil {
		t.Fatal(productionASTError)
	}
	return productionFileSet, productionASTFiles
}

// deriveRefusalConstructors derives the package functions that exist only to
// build a refusal, instead of naming them. A constructor is a function whose
// body is a single return of an fmt/errors error value.
func deriveRefusalConstructors(t *testing.T) map[string]bool {
	t.Helper()

	_, files := parsedProductionPackage(t)
	constructors := make(map[string]bool)
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil || len(function.Body.List) != 1 {
				continue
			}
			statement, ok := function.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(statement.Results) != 1 {
				continue
			}
			call, ok := statement.Results[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			if isStandardErrorConstructor(call) {
				constructors[function.Name.Name] = true
			}
		}
	}
	if len(constructors) == 0 {
		t.Fatal("derived zero refusal constructors for the canonicaljson package")
	}
	return constructors
}

func isStandardErrorConstructor(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	packageName, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	switch {
	case packageName.Name == "fmt" && selector.Sel.Name == "Errorf":
		return true
	case packageName.Name == "errors" && selector.Sel.Name == "New":
		return true
	}
	return false
}

// deriveRefusalSites returns every production return statement that can emit a
// refusal: one that returns a refusal constructor call, an fmt/errors value, or
// a propagated `err`. Refusal constructors themselves are excluded — their own
// return is the refusal being built, not a guard that decides to refuse.
func deriveRefusalSites(t *testing.T) []refusalSite {
	t.Helper()

	fileSet, files := parsedProductionPackage(t)
	constructors := deriveRefusalConstructors(t)

	var sites []refusalSite
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			if function.Recv == nil && constructors[function.Name.Name] {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				statement, ok := node.(*ast.ReturnStmt)
				if !ok || !isRefusalReturn(statement, constructors) {
					return true
				}
				position := fileSet.Position(statement.Pos())
				sites = append(sites, refusalSite{
					file:     filepath.Base(position.Filename),
					line:     position.Line,
					column:   position.Column,
					function: function.Name.Name,
					text:     returnStatementText(fileSet, statement),
				})
				return true
			})
		}
	}
	sort.Slice(sites, func(first, second int) bool {
		if sites[first].file != sites[second].file {
			return sites[first].file < sites[second].file
		}
		return sites[first].line < sites[second].line
	})
	ordinals := make(map[string]int, len(sites))
	for index := range sites {
		identity := sites[index].file + "|" + sites[index].function + "|" + sites[index].text
		sites[index].ordinal = ordinals[identity]
		ordinals[identity]++
	}
	if len(sites) == 0 {
		t.Fatal("derived zero refusal guards for the canonicaljson package")
	}
	return sites
}

func isRefusalReturn(statement *ast.ReturnStmt, constructors map[string]bool) bool {
	refusal := false
	for _, result := range statement.Results {
		ast.Inspect(result, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.Ident:
				if typed.Name == "err" {
					refusal = true
				}
			case *ast.CallExpr:
				if identifier, ok := typed.Fun.(*ast.Ident); ok && constructors[identifier.Name] {
					refusal = true
				}
				if isStandardErrorConstructor(typed) {
					refusal = true
				}
			}
			return !refusal
		})
	}
	return refusal
}

// returnStatementText renders the return statement so a failure names the guard
// rather than a bare position, and so a rewritten guard invalidates any
// unreachability declaration made about it.
func returnStatementText(fileSet *token.FileSet, statement *ast.ReturnStmt) string {
	var rendered strings.Builder
	rendered.WriteString("return")
	for index, result := range statement.Results {
		if index > 0 {
			rendered.WriteString(",")
		}
		rendered.WriteString(" ")
		rendered.WriteString(expressionText(fileSet, result))
	}
	return rendered.String()
}

func expressionText(fileSet *token.FileSet, expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.BasicLit:
		return typed.Value
	case *ast.SelectorExpr:
		return expressionText(fileSet, typed.X) + "." + typed.Sel.Name
	case *ast.CallExpr:
		arguments := make([]string, 0, len(typed.Args))
		for _, argument := range typed.Args {
			arguments = append(arguments, expressionText(fileSet, argument))
		}
		return expressionText(fileSet, typed.Fun) + "(" + strings.Join(arguments, ", ") + ")"
	case *ast.UnaryExpr:
		return typed.Op.String() + expressionText(fileSet, typed.X)
	case *ast.BinaryExpr:
		return expressionText(fileSet, typed.X) + " " + typed.Op.String() + " " + expressionText(fileSet, typed.Y)
	case *ast.IndexExpr:
		return expressionText(fileSet, typed.X) + "[" + expressionText(fileSet, typed.Index) + "]"
	case *ast.CompositeLit:
		if typed.Type == nil {
			return "{}"
		}
		return expressionText(fileSet, typed.Type) + "{}"
	case *ast.FuncLit:
		return "func"
	}
	return "expr"
}

// deriveTotalRefusalValidators returns the package functions that cannot accept
// anything: every return they can make is a refusal, directly or through
// another total-refusal function. This is how the fixture obligation in
// identity_fixtures_test.go knows that a registered schema is an explicit
// refusal rather than a shape with a valid candidate, without naming either
// validator.
func deriveTotalRefusalValidators(t *testing.T) map[string]bool {
	t.Helper()

	_, files := parsedProductionPackage(t)
	constructors := deriveRefusalConstructors(t)

	total := make(map[string]bool, len(constructors))
	for name := range constructors {
		total[name] = true
	}
	for range len(files) + 8 {
		grew := false
		for _, file := range files {
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || function.Body == nil || total[function.Name.Name] {
					continue
				}
				if functionAlwaysRefuses(function, total) {
					total[function.Name.Name] = true
					grew = true
				}
			}
		}
		if !grew {
			break
		}
	}
	return total
}

func functionAlwaysRefuses(function *ast.FuncDecl, total map[string]bool) bool {
	returns := 0
	always := true
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		statement, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		returns++
		for _, result := range statement.Results {
			switch typed := result.(type) {
			case *ast.Ident:
				if typed.Name == "err" {
					continue
				}
				always = false
			case *ast.CallExpr:
				identifier, ok := typed.Fun.(*ast.Ident)
				if ok && total[identifier.Name] {
					continue
				}
				always = false
			default:
				always = false
			}
		}
		return true
	})
	return returns > 0 && always
}
