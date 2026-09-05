package provhost

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// This file closes the closed-vocabulary class for the shapes the
// census derives: every package-level []string, map[string]bool, and
// map[string]string consulted by a membership check carries a
// registered spec derivation that executes. The eight enum
// derivations in closed_vocabularies_test.go are hand-registered one
// by one, and nine more membership-checked vocabularies existed in
// production with no derivation at all: a widened member set still
// admitted its extra body with the suite green. Enumerating one more
// hand-written list per vocabulary would move the gap, not close it,
// so the domain is derived the way
// TestDerivedRefusalArmsAreAllWitnessed already does for refusal
// arms: deriveMembershipVocabularies parses production source for
// the three package-level shapes, and
// TestClosedVocabularyCensusCoversEveryMembershipVocabulary requires
// each to carry a registered spec derivation. A new vocabulary with
// no derivation fails here instead of passing silently.
//
// Three vocabularies used to live outside the derived shapes as
// switch statements (the response envelope member set, the execution
// profile set, the transaction state set); a widened case admitted
// its extra value with the suite green. They are package-level maps
// now (responseMembers, profileNames, statusStates), so the census
// derives them like the rest. The remaining switch in production
// dispatches on an already-validated state rather than admitting
// members, and TestAllProductionSwitchesAreClassified pins that:
// any new switch must be classified before the suite passes.
//
// A membership check is one of: an unknownMember argument, a direct
// map index, or a range that builds an allow-set or compares
// element-wise. The *Required order lists are deliberately out of
// scope: their only production use is as missingMember arguments,
// they encode deterministic refusal order rather than containment,
// and dropping their last entry is behaviour-preserving (the omitted
// body still refuses). profileProviders is likewise out of scope:
// nothing in production consults it. The []Operation registries are
// out of scope by type and stay derived by
// TestOperationRegistryIsDerivedFromSpec. Function-local literals
// (such as the required list inside checkResponseMembers) belong to
// the *Required order-list class above, not to containment. An empty
// derivation — zero files scanned, zero vocabularies found, a spec
// window with no members — fails closed rather than measuring
// nothing.

// requireMemberSet asserts the production member set holds exactly
// the spec-derived members, order-insensitive: maps carry no order.
// Either side empty is a blind check, so both fail outright.
func requireMemberSet(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) == 0 {
		t.Fatalf("%s holds no members; the check is blind", name)
	}
	if len(want) == 0 {
		t.Fatalf("%s derives no members from the pinned document; the check is blind", name)
	}
	ordered := func(members []string) []string {
		duplicated := append([]string(nil), members...)
		sort.Strings(duplicated)
		return duplicated
	}
	if !reflect.DeepEqual(ordered(got), ordered(want)) {
		t.Fatalf("%s = %v, want the pinned member set %v", name, ordered(got), ordered(want))
	}
	t.Logf("%s coverage: %d/%d members derived", name, len(got), len(want))
}

// exampleObjectKeysOf parses the top-level keys of the JSON example
// embedded in a spec window, in document order. A window with no
// JSON object is a blind check, so it fails outright.
func exampleObjectKeysOf(t *testing.T, window string) []string {
	t.Helper()
	start := strings.Index(window, "{")
	end := strings.LastIndex(window, "}")
	if start < 0 || end <= start {
		t.Fatal("spec window holds no JSON object; the check is blind")
	}
	example, fault := decodeStrictObject([]byte(window[start : end+1]))
	if fault != nil {
		t.Fatalf("spec example is not a strict object: %v %q", fault.detail, fault.member)
	}
	var keys []string
	for key := range example {
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		t.Fatal("spec example holds no members; the check is blind")
	}
	return keys
}

// typeRowMembersOf parses one Section 7.5 embedded-type table row:
// the first code span names the type, and every later span opens
// with one member declaration. A row whose first span is not the
// named type, or that carries no member span, is a blind check.
func typeRowMembersOf(t *testing.T, row, typeName string) []string {
	t.Helper()
	spans := codeSpanPattern.FindAllStringSubmatch(row, -1)
	if len(spans) == 0 {
		t.Fatalf("Section 7.5 %s row holds no code elements; the check is blind", typeName)
	}
	if spans[0][1] != typeName {
		t.Fatalf("Section 7.5 row opens with %q, want the %s declaration; the check is blind", spans[0][1], typeName)
	}
	var members []string
	for _, span := range spans[1:] {
		name := span[1]
		if end := strings.Index(name, ":"); end >= 0 {
			name = name[:end]
		}
		name = strings.TrimSpace(name)
		if name == "" {
			t.Fatalf("Section 7.5 %s row parses to an empty member; the check is blind", typeName)
		}
		members = append(members, name)
	}
	if len(members) == 0 {
		t.Fatalf("Section 7.5 %s row parses to no members; the check is blind", typeName)
	}
	return members
}

// objectCellMembersOf parses the braced success-body cell of one
// Section 7.5 operation row: the single code span starting with "{"
// whose text names marker, split on top-level commas, one member per
// element before its colon. Zero or two matching cells is a blind
// check.
func objectCellMembersOf(t *testing.T, row, marker string) []string {
	t.Helper()
	spans := codeSpanPattern.FindAllStringSubmatch(row, -1)
	var cell string
	matches := 0
	for _, span := range spans {
		if strings.HasPrefix(span[1], "{") && strings.Contains(span[1], marker) {
			cell = span[1]
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("Section 7.5 row holds %d success cells naming %q, want exactly one; the check is blind", matches, marker)
	}
	inner := cell
	if !strings.HasPrefix(inner, "{") || !strings.HasSuffix(inner, "}") {
		t.Fatalf("Section 7.5 cell %q is not a braced object; the check is blind", cell)
	}
	inner = inner[1 : len(inner)-1]
	var members []string
	depth := 0
	field := ""
	flush := func() {
		name := field
		if end := strings.Index(name, ":"); end >= 0 {
			name = name[:end]
		}
		name = strings.TrimSpace(name)
		if name == "" {
			t.Fatalf("Section 7.5 cell %q parses to an empty member; the check is blind", cell)
		}
		members = append(members, name)
		field = ""
	}
	for _, char := range inner {
		switch char {
		case '(', '[':
			depth++
			field += string(char)
		case ')', ']':
			depth--
			field += string(char)
		case ',':
			if depth == 0 {
				flush()
				continue
			}
			field += string(char)
		default:
			field += string(char)
		}
	}
	flush()
	if len(members) == 0 {
		t.Fatalf("Section 7.5 cell %q parses to no members; the check is blind", cell)
	}
	return members
}

// TestManifestMembersAreDerivedFromSpec proves the exact manifest
// member set equals the Section 7.3 example keys: a widened
// implementation admitting "bogus" reddens here.
func TestManifestMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.3", 2751, 2785)
	requireQuote(t, document, "The manifest is closed and every displayed member is required", "7.3")
	var got []string
	for member := range manifestMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "manifestMembers", got, exampleObjectKeysOf(t, window))
}

// TestProbeMembersAreDerivedFromSpec proves the exact probe member
// set equals the Section 7.4 example keys: a widened implementation
// admitting "bogus" reddens here.
func TestProbeMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.4", 2805, 2857)
	requireQuote(t, document, "The probe object is closed and every displayed member is required", "7.4")
	var got []string
	for member := range probeMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "probeMembers", got, exampleObjectKeysOf(t, window))
}

// TestProbeCapabilityMembersAreDerivedFromSpec proves the exact
// capability-value member set equals the Section 7.4 sentence: a
// widened implementation admitting "bogus" reddens here.
func TestProbeCapabilityMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.4", 2864, 2866)
	requireQuote(t, document, "Each capability value contains exactly", "7.4")
	// Only the declaring sentence counts: the next sentence opens
	// with the warnings member, which belongs to the probe object
	// rather than to one capability value.
	sentence := window
	if end := strings.Index(sentence, "."); end >= 0 {
		sentence = sentence[:end+1]
	}
	var got []string
	for member := range probeCapabilityMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "probeCapabilityMembers", got, codeSpansOf(t, sentence))
}

// TestIdentityMembersAreDerivedFromSpec proves the exact identity
// member set equals the Section 5.5 table: a widened implementation
// admitting "bogus" reddens here. The rows are read through the
// document's own table index, so a window that drifts off the table
// lands on lines no table row declares and fails closed.
func TestIdentityMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "5.5", 2077, 2092)
	// The bare closed sentence repeats across sections, so the
	// quote carries its schema opener: only the 5.5 table it
	// introduces can satisfy it.
	requireQuote(t, document, "The Provider Identity Record schema is <code>urn:ax:schema:provider-identity</code> version <code>1.0.0</code>. Its top-level object is closed and contains exactly:", "5.5")
	_ = window
	var want []string
	for line := 2077; line <= 2092; line++ {
		row, ok := document.TableRowAt(line)
		if !ok {
			t.Fatalf("SPEC.md line %d is not a table body row; the check is blind", line)
		}
		if row.Header != "Field" {
			t.Fatalf("SPEC.md line %d declares %q, want a Field row; the check is blind", line, row.Header)
		}
		if row.Identifier == "" {
			t.Fatalf("SPEC.md line %d declares an empty member; the check is blind", line)
		}
		want = append(want, row.Identifier)
	}
	var got []string
	for member := range identityMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "identityMembers", got, want)
}

// TestIdentifyMembersAreDerivedFromSpec proves the exact
// identify-session success member set equals the Section 7.5 row: a
// widened implementation admitting "bogus" reddens here.
func TestIdentifyMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	row := sectionLines(t, document, "7.5", 3075, 3075)
	var got []string
	for member := range identifyMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "identifyMembers", got, objectCellMembersOf(t, row, "matched_evidence"))
}

// TestQuiesceMembersAreDerivedFromSpec proves the exact
// SafeBoundaryProof member set equals the Section 7.5 type row: a
// widened implementation admitting "bogus" reddens here.
func TestQuiesceMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	row := sectionLines(t, document, "7.5", 2891, 2891)
	var got []string
	for member := range quiesceMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "quiesceMembers", got, typeRowMembersOf(t, row, "SafeBoundaryProof"))
}

// TestSpawnMembersAreDerivedFromSpec proves the exact SpawnPlan
// member set equals the Section 7.5 type row: a widened
// implementation admitting "bogus" reddens here.
func TestSpawnMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	row := sectionLines(t, document, "7.5", 2890, 2890)
	var got []string
	for member := range spawnMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "spawnMembers", got, typeRowMembersOf(t, row, "SpawnPlan"))
}

// TestStatusBodyMembersAreDerivedFromSpec proves the exact status
// member set equals the Section 7.5 ProviderTransactionStatus row: a
// widened implementation admitting "bogus" reddens here.
func TestStatusBodyMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	row := sectionLines(t, document, "7.5", 2905, 2905)
	var got []string
	for member := range statusBodyMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "statusBodyMembers", got, typeRowMembersOf(t, row, "ProviderTransactionStatus"))
}

// TestProfileYOLOMappingIsDerivedFromSpec proves the yolo mapping
// holds exactly the six Section 7.7 provider rows with the row's own
// flag: a seventh provider entry returns its flag instead of
// invalid_config and reddens here. The Pi row carries no flag, only
// the reported tool-set name, so a span holding a quoted value
// resolves to the value inside the quotes.
func TestProfileYOLOMappingIsDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.7", 3431, 3436)
	requireQuote(t, document, "The v0.3.0 <code>yolo</code> mappings are:", "7.7")
	_ = window
	derived := map[string]string{}
	var providers []string
	for line := 3431; line <= 3436; line++ {
		text, ok := document.Line(line)
		if !ok {
			t.Fatalf("SPEC.md line %d is missing; the check is blind", line)
		}
		row, ok := document.TableRowAt(line)
		if !ok {
			t.Fatalf("SPEC.md line %d is not a table body row; the check is blind", line)
		}
		if row.Header != "Provider" {
			t.Fatalf("SPEC.md line %d declares %q, want a Provider row; the check is blind", line, row.Header)
		}
		fields := strings.Fields(row.FirstCell)
		if len(fields) == 0 {
			t.Fatalf("SPEC.md line %d names no provider; the check is blind", line)
		}
		provider := strings.ToLower(fields[0])
		spans := codeSpanPattern.FindAllStringSubmatch(text, -1)
		if len(spans) == 0 {
			t.Fatalf("SPEC.md line %d holds no code elements; the check is blind", line)
		}
		flag := spans[0][1]
		if first, last := strings.Index(flag, `"`), strings.LastIndex(flag, `"`); first >= 0 && last > first {
			flag = flag[first+1 : last]
		}
		if flag == "" {
			t.Fatalf("SPEC.md line %d resolves to an empty mapping; the check is blind", line)
		}
		if _, repeated := derived[provider]; repeated {
			t.Fatalf("SPEC.md names provider %q twice; the check is blind", provider)
		}
		derived[provider] = flag
		providers = append(providers, provider)
	}
	if len(derived) == 0 {
		t.Fatal("derived no provider mappings from the Section 7.7 table; the check is blind")
	}
	if !reflect.DeepEqual(derived, profileYOLOMapping) {
		t.Fatalf("profileYOLOMapping = %v, want the Section 7.7 table %v", profileYOLOMapping, derived)
	}
	sorted := append([]string(nil), providers...)
	sort.Strings(sorted)
	registered := append([]string(nil), profileProviders...)
	sort.Strings(registered)
	if !reflect.DeepEqual(sorted, registered) {
		t.Fatalf("Section 7.7 providers = %v, want the registry %v", sorted, registered)
	}
	t.Logf("profile mapping coverage: %d/%d providers derived", len(derived), len(providers))
}

// TestResponseMembersAreDerivedFromSpec proves the exact response
// envelope member set equals the Section 7.2 union: the success
// sentence names protocol, protocol_version, request_id, ok, and
// body, and the failure sentence adds error without body. The ok
// spans carry their boolean value (`ok = true`, `ok = false`), which
// is normalized to the member name. A seventh implementation member
// admitting "bogus" reddens here.
func TestResponseMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.2", 2708, 2712)
	requireQuote(t, document, "unknown members are protocol errors under major version 2", "7.2")
	normalize := func(span string) string {
		for _, suffix := range []string{" = true", " = false"} {
			span = strings.TrimSuffix(span, suffix)
		}
		return span
	}
	seen := map[string]bool{}
	var want []string
	for _, span := range codeSpansOf(t, window) {
		name := normalize(span)
		if name == "" {
			t.Fatalf("Section 7.2 envelope sentence parses to an empty member; the check is blind")
		}
		if !seen[name] {
			seen[name] = true
			want = append(want, name)
		}
	}
	var got []string
	for member := range responseMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "responseMembers", got, want)
}

// TestProfileNamesAreDerivedFromSpec proves the exact execution
// profile vocabulary equals the two Section 7.7 profile sentences:
// the yolo sentence names yolo, the standard sentence names
// standard. Each sentence must carry exactly its own profile span,
// so a sentence that drifts onto another member fails closed. A
// third implementation profile resolving "bogus" to the unrestricted
// flag reddens here.
func TestProfileNamesAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	requireQuote(t, document, "The v0.3.0 <code>yolo</code> mappings are:", "7.7")
	requireQuote(t, document, "<code>standard</code> MUST use the provider's normal approval/sandbox behavior", "7.7")
	yolo := codeSpansOf(t, sectionLines(t, document, "7.7", 3427, 3427))
	if len(yolo) != 1 || yolo[0] != "yolo" {
		t.Fatalf("Section 7.7 yolo sentence names %v, want exactly [yolo]; the check is blind", yolo)
	}
	standard := codeSpansOf(t, sectionLines(t, document, "7.7", 3438, 3438))
	if len(standard) != 1 || standard[0] != "standard" {
		t.Fatalf("Section 7.7 standard sentence names %v, want exactly [standard]; the check is blind", standard)
	}
	var got []string
	for profile := range profileNames {
		got = append(got, profile)
	}
	requireMemberSet(t, "profileNames", got, []string{"standard", "yolo"})
}

// TestStatusStatesAreDerivedFromSpec proves the exact transaction
// state vocabulary equals the Section 7.5 ProviderTransactionStatus
// row: a fifth implementation state decoding "bogus" as durable
// state reddens here.
func TestStatusStatesAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	row := sectionLines(t, document, "7.5", 2905, 2905)
	// Each member of this row is its own code span, so the state
	// cell is the single span opening with "state:", split on the
	// document's pipe entity. Zero or two matching spans is a blind
	// check.
	var want []string
	matches := 0
	for _, span := range codeSpansOf(t, row) {
		if strings.HasPrefix(span, "state:") {
			want = strings.Split(strings.TrimPrefix(span, "state:"), "&#124;")
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("Section 7.5 row holds %d state cells, want exactly one; the check is blind", matches)
	}
	for _, state := range want {
		if strings.TrimSpace(state) == "" {
			t.Fatalf("Section 7.5 state cell parses to an empty member; the check is blind")
		}
	}
	var got []string
	for state := range statusStates {
		got = append(got, state)
	}
	requireMemberSet(t, "statusStates", got, want)
}

// TestAllProductionSwitchesAreClassified closes the switch residue
// the package-level census cannot see: every switch statement in
// production source must be a classified dispatch, never a membership
// gate. The one remaining switch dispatches on a state
// DecodeStatusOutcome already proved a registry member, so widening
// its cases admits nothing. A new switch — the shape three closed
// vocabularies used to hide in — fails here until it is classified,
// which is what keeps the header claim true.
func TestAllProductionSwitchesAreClassified(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("classify production switches: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("classify production switches: %v", err)
	}
	// classified maps the enclosing function of each allowed switch to
	// the reason it cannot admit a new member.
	classified := map[string]string{
		"DecodeStatusOutcome": "dispatches on a state validStatusState already proved a registry member",
	}
	fileSet := token.NewFileSet()
	matched := map[string]bool{}
	files := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files++
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("classify production switches: %v", err)
		}
		syntax, err := parser.ParseFile(fileSet, path, source, 0)
		if err != nil {
			t.Fatalf("classify production switches: %v", err)
		}
		for _, decl := range syntax.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if _, isSwitch := node.(*ast.SwitchStmt); !isSwitch {
					return true
				}
				reason, ok := classified[function.Name.Name]
				if !ok {
					t.Fatalf("unclassified switch in %s (%s): a membership gate here is invisible to the census", name, function.Name.Name)
				}
				matched[function.Name.Name] = true
				t.Logf("classified switch in %s (%s): %s", name, function.Name.Name, reason)
				return true
			})
		}
		// Package-level initializer closures hold no switch today;
		// one appearing there is unclassified by construction.
		for _, decl := range syntax.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			ast.Inspect(general, func(node ast.Node) bool {
				if _, isSwitch := node.(*ast.SwitchStmt); isSwitch {
					t.Fatalf("unclassified switch in a package initializer in %s: a membership gate here is invisible to the census", name)
				}
				return true
			})
		}
	}
	if files == 0 {
		t.Fatal("scanned zero production files; the switch classification is blind, not the package")
	}
	for function := range classified {
		if !matched[function] {
			t.Fatalf("classified switch %q matches no production switch; the classification is stale", function)
		}
	}
}

// deriveMembershipVocabularies parses production source for every
// package-level []string, map[string]bool, and map[string]string
// consulted by a membership check. An empty scan fails closed: a
// domain that silently derives nothing is not a measurement.
func deriveMembershipVocabularies(t *testing.T) map[string]struct{} {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("derive membership vocabularies: %v", err)
	}
	vocabularies, scanned, err := membershipVocabulariesIn(directory)
	if err != nil {
		t.Fatalf("derive membership vocabularies: %v", err)
	}
	if len(scanned) == 0 {
		t.Fatal("derived membership vocabularies from zero production files; the scanner is broken, not the package")
	}
	if len(vocabularies) == 0 {
		t.Fatal("derived zero membership vocabularies from the package sources; the scanner is broken, not the package")
	}
	t.Logf("membership vocabulary domain: %d derived vocabularies across %d production files", len(vocabularies), len(scanned))
	return vocabularies
}

// membershipVocabulariesIn collects the package-level vocabularies of
// the three membership shapes and keeps those with membership use:
// an unknownMember argument, a direct map index, or a range operand.
// A variable whose only production uses are missingMember arguments
// is a refusal-order list, not a containment vocabulary, and is
// excluded by design.
func membershipVocabulariesIn(directory string) (map[string]struct{}, []string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, nil, err
	}
	fileSet := token.NewFileSet()
	type productionFile struct {
		syntax *ast.File
	}
	var files []productionFile
	var scanned []string
	shapes := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned = append(scanned, name)
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		syntax, err := parser.ParseFile(fileSet, path, source, 0)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, productionFile{syntax: syntax})
		for _, decl := range syntax.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, spec := range general.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				shape := valueShape(value)
				if shape == "" {
					continue
				}
				for _, name := range value.Names {
					shapes[name.Name] = true
				}
			}
		}
	}
	unknownUsed := map[string]bool{}
	indexed := map[string]bool{}
	ranged := map[string]bool{}
	for _, file := range files {
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || ident.Name != "unknownMember" {
				return true
			}
			for _, arg := range call.Args {
				if name, ok := arg.(*ast.Ident); ok {
					unknownUsed[name.Name] = true
				}
			}
			return true
		})
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			index, ok := node.(*ast.IndexExpr)
			if !ok {
				return true
			}
			if name, ok := index.X.(*ast.Ident); ok {
				indexed[name.Name] = true
			}
			return true
		})
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			loop, ok := node.(*ast.RangeStmt)
			if !ok {
				return true
			}
			if name, ok := loop.X.(*ast.Ident); ok {
				ranged[name.Name] = true
			}
			return true
		})
	}
	vocabularies := map[string]struct{}{}
	for name := range shapes {
		if unknownUsed[name] || indexed[name] || ranged[name] {
			vocabularies[name] = struct{}{}
		}
	}
	sort.Strings(scanned)
	return vocabularies, scanned, nil
}

// valueShape reports whether a package-level variable declaration is
// one of the three membership-vocabulary shapes: a []string, a
// map[string]bool, or a map[string]string. Anything else reports "".
func valueShape(value *ast.ValueSpec) string {
	candidate := value.Type
	if candidate == nil && len(value.Values) == 1 {
		if literal, ok := value.Values[0].(*ast.CompositeLit); ok {
			candidate = literal.Type
		}
	}
	switch shape := candidate.(type) {
	case *ast.ArrayType:
		if shape.Len != nil {
			return ""
		}
		if ident, ok := shape.Elt.(*ast.Ident); ok && ident.Name == "string" {
			return "slice"
		}
	case *ast.MapType:
		key, ok := shape.Key.(*ast.Ident)
		if !ok || key.Name != "string" {
			return ""
		}
		if ident, ok := shape.Value.(*ast.Ident); ok && (ident.Name == "bool" || ident.Name == "string") {
			return "map"
		}
	}
	return ""
}

// vocabularyDerivation ties one derived membership vocabulary to the
// test that proves it against the pinned document. The prove
// function executes the real spec read: a registration pointing at
// nothing that checks the vocabulary would pass vacuously.
type vocabularyDerivation struct {
	name  string
	prove func(*testing.T)
}

// registeredVocabularyDerivations lists every membership vocabulary
// the census may find, each with the derivation that pins it. The
// eight enum and two registry entries reuse their covering tests
// with no restatement; the nine member-set and mapping entries plus
// the three formerly-switch vocabularies are proven above.
func registeredVocabularyDerivations() []vocabularyDerivation {
	return []vocabularyDerivation{
		{name: "probeCapabilityStatuses", prove: TestProbeStatusVocabularyIsDerivedFromSpec},
		{name: "probeCapabilityEvidence", prove: TestProbeEvidenceVocabularyIsDerivedFromSpec},
		{name: "probeArchitectures", prove: TestProbeArchitecturesAreDerivedFromSpec},
		{name: "probePlatforms", prove: TestProbePlatformsAreDerivedFromSpec},
		{name: "quiesceBlockers", prove: TestQuiesceBlockersAreDerivedFromSpec},
		{name: "identityKinds", prove: TestIdentityKindsAreDerivedFromSpec},
		{name: "identifyConfidences", prove: TestIdentifyVocabulariesAreDerivedFromSpec},
		{name: "identifyEvidence", prove: TestIdentifyVocabulariesAreDerivedFromSpec},
		{name: "capabilityOrder", prove: TestManifestRegistriesAreDerivedFromSpec},
		{name: "manifestPlatforms", prove: TestManifestRegistriesAreDerivedFromSpec},
		{name: "manifestMembers", prove: TestManifestMembersAreDerivedFromSpec},
		{name: "probeMembers", prove: TestProbeMembersAreDerivedFromSpec},
		{name: "probeCapabilityMembers", prove: TestProbeCapabilityMembersAreDerivedFromSpec},
		{name: "identityMembers", prove: TestIdentityMembersAreDerivedFromSpec},
		{name: "identifyMembers", prove: TestIdentifyMembersAreDerivedFromSpec},
		{name: "quiesceMembers", prove: TestQuiesceMembersAreDerivedFromSpec},
		{name: "spawnMembers", prove: TestSpawnMembersAreDerivedFromSpec},
		{name: "statusBodyMembers", prove: TestStatusBodyMembersAreDerivedFromSpec},
		{name: "profileYOLOMapping", prove: TestProfileYOLOMappingIsDerivedFromSpec},
		{name: "responseMembers", prove: TestResponseMembersAreDerivedFromSpec},
		{name: "profileNames", prove: TestProfileNamesAreDerivedFromSpec},
		{name: "statusStates", prove: TestStatusStatesAreDerivedFromSpec},
	}
}

// TestClosedVocabularyCensusCoversEveryMembershipVocabulary checks
// both directions of the census: every derived vocabulary carries a
// registered spec derivation, and every registered derivation names
// a derived vocabulary, so a truncated scan reddens on the orphaned
// registrations instead of passing vacuously. Each derivation then
// executes: the census proves containment, not reachability.
func TestClosedVocabularyCensusCoversEveryMembershipVocabulary(t *testing.T) {
	found := deriveMembershipVocabularies(t)
	registered := registeredVocabularyDerivations()
	if len(registered) == 0 {
		t.Fatal("registered no vocabulary derivations; the census is blind, not the package")
	}
	byName := map[string]func(*testing.T){}
	for _, entry := range registered {
		if _, repeated := byName[entry.name]; repeated {
			t.Fatalf("vocabulary %q carries two registered derivations; the census is ambiguous", entry.name)
		}
		byName[entry.name] = entry.prove
	}
	for name := range found {
		if _, ok := byName[name]; !ok {
			t.Fatalf("membership vocabulary %q has no registered spec derivation; widening it passes silently", name)
		}
	}
	for _, entry := range registered {
		if _, ok := found[entry.name]; !ok {
			t.Fatalf("registered derivation %q names no production vocabulary; the census is short", entry.name)
		}
	}
	for _, entry := range registered {
		t.Run(entry.name, entry.prove)
	}
	t.Logf("vocabulary census coverage: %d/%d vocabularies derived", len(found), len(registered))
}

// TestClosedMemberSetsRefuseExtraMembers drives one extra "bogus"
// member per closed object through its production entry point: each
// is refused under its own arm, so a member set that admits one more
// value reddens here as well as in its derivation test. The
// production call site is named in every row.
func TestClosedMemberSetsRefuseExtraMembers(t *testing.T) {
	t.Run("manifest carries bogus", func(t *testing.T) {
		body := manifestVariant(t, `"schema_version": "1.0.0",`, "\"schema_version\": \"1.0.0\",\n  \"bogus\": 1,")
		requireFrameRefusal(t, DecodeManifest(body), "bogus", "unknown member")
	})
	t.Run("probe carries bogus", func(t *testing.T) {
		body := probeVariant(t, `"schema_version": "1.0.0",`, "\"schema_version\": \"1.0.0\",\n  \"bogus\": 1,")
		requireFrameRefusal(t, DecodeProbe(body), "bogus", "unknown member")
	})
	t.Run("probe capability carries bogus", func(t *testing.T) {
		body := probeVariant(t, "\"status\": \"available\",\n      \"enabled\": true,", "\"status\": \"available\",\n      \"bogus\": 1,\n      \"enabled\": true,")
		requireFrameRefusal(t, DecodeProbe(body), "capabilities", "unknown member")
	})
	t.Run("identity carries bogus", func(t *testing.T) {
		body := identityVariant(t, `"schema_version": "1.0.0",`, "\"schema_version\": \"1.0.0\",\n  \"bogus\": 1,")
		requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "bogus", "unknown member")
	})
	t.Run("identify result carries bogus", func(t *testing.T) {
		body := identifyResultVariant(t, `"confidence": "exact"`, "\"bogus\": 1,\n  \"confidence\": \"exact\"")
		requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "bogus", "unknown member")
	})
	t.Run("quiesce proof carries bogus", func(t *testing.T) {
		body := quiesceVariant(t, `"provider_id": "codex",`, "\"provider_id\": \"codex\",\n  \"bogus\": 1,")
		requireFrameRefusal(t, quiesceErr(body), "bogus", "unknown member")
	})
	t.Run("spawn plan carries bogus", func(t *testing.T) {
		body := spawnVariant(t, `"cwd":`, "\"bogus\": 1,\n  \"cwd\":")
		requireFrameRefusal(t, DecodeSpawnPlan(body, "codex", ProfileYOLO, scalar.PlatformLinux), "bogus", "unknown member")
	})
	t.Run("status body carries bogus", func(t *testing.T) {
		body := statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "prepared", testRollbackToken, testDiscovery)
		extra := []byte(strings.Replace(string(body), "{\"materialization_id\":", "{\"bogus\":1,\"materialization_id\":", 1))
		_, err := DecodeStatusOutcome(extra, testStatusIDs())
		requireLocalRefusal(t, err, "integrity_failure", "unknown member")
	})
	t.Run("profile mapping names bogus", func(t *testing.T) {
		_, err := ProfileMapping("bogus", ProfileYOLO)
		requireLocalRefusal(t, err, "invalid_config", "unknown provider")
	})
	t.Run("envelope carries bogus", func(t *testing.T) {
		_, err := DecodeResponse(armMemberFrame(`"ok":true,"body":{},"bogus":[]`), mustUUIDv7(t, testRequestID))
		requireFrameRefusal(t, err, "bogus", "unknown member")
	})
	t.Run("profile mapping names bogus profile", func(t *testing.T) {
		_, err := ProfileMapping("codex", "bogus")
		requireLocalRefusal(t, err, "invalid_config", "unknown profile")
	})
	t.Run("status body carries bogus state", func(t *testing.T) {
		body := statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "bogus", testRollbackToken, testDiscovery)
		_, err := DecodeStatusOutcome(body, testStatusIDs())
		requireLocalRefusal(t, err, "integrity_failure", "status state is not a registry member")
	})
}
