package terminstance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// TestRowCodesAreAdmittedByCheckErrorAllowed proves the four row codes
// this package spells literally are admitted by the landed
// CheckErrorAllowed for their rows: quiesce_timeout only on
// wait-safe-boundary, stop_timeout only on request-stop, and the two
// backend codes on the rows the landed table lists. The literals below
// are the pinned specification tokens.
func TestRowCodesAreAdmittedByCheckErrorAllowed(t *testing.T) {
	admitted := map[string][]string{
		"quiesce-input":      {"terminal_backend_timeout", "terminal_backend_integrity_failure"},
		"wait-safe-boundary": {"quiesce_timeout", "terminal_backend_integrity_failure"},
		"request-stop":       {"stop_timeout", "terminal_backend_process_failed", "terminal_backend_integrity_failure"},
		"status":             {"terminal_backend_timeout", "terminal_backend_unavailable", "terminal_backend_integrity_failure"},
	}
	for operation, codes := range admitted {
		for _, code := range codes {
			if err := terminalbackend.CheckErrorAllowed(operation, code); err != nil {
				t.Errorf("CheckErrorAllowed(%q, %q) error = %v, want admission", operation, code, err)
			}
		}
	}
	outside := map[string][]string{
		"quiesce-input":      {"quiesce_timeout", "stop_timeout"},
		"wait-safe-boundary": {"stop_timeout", "terminal_backend_timeout", "terminal_backend_process_failed"},
		"request-stop":       {"quiesce_timeout", "terminal_backend_timeout"},
		"status":             {"quiesce_timeout", "stop_timeout", "idempotency_mismatch", "local_precondition_failed"},
	}
	for operation, codes := range outside {
		for _, code := range codes {
			if err := terminalbackend.CheckErrorAllowed(operation, code); err == nil {
				t.Errorf("CheckErrorAllowed(%q, %q) = nil, want refusal", operation, code)
			}
		}
	}
}

// derivedConstLiterals derives the (name, literal) pairs of the const
// block declaring the named type: the closed vocabulary as spelled in
// production source. The census counts literal copies, not names: a
// member spelled twice or sharing another member's literal reddens the
// count, as does an added, removed or re-spelled member.
func derivedConstLiterals(t *testing.T, file, typeName string) (names []string, literals []string) {
	t.Helper()
	parsed := parseProductionFile(t, file)
	for _, decl := range parsed.Decls {
		general, ok := decl.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, spec := range general.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			typeIdent, ok := value.Type.(*ast.Ident)
			if !ok || typeIdent.Name != typeName {
				continue
			}
			for i, name := range value.Names {
				if len(value.Values) <= i {
					t.Fatalf("%s const %s has no value", file, name.Name)
				}
				literal, ok := value.Values[i].(*ast.BasicLit)
				if !ok {
					t.Fatalf("%s const %s is not a string literal", file, name.Name)
				}
				names = append(names, name.Name)
				literals = append(literals, stringsTrimQuotes(literal.Value))
			}
		}
	}
	if len(names) == 0 {
		t.Fatalf("%s has no const block for %s", file, typeName)
	}
	return names, literals
}

// derivedCaseIdents derives the case expressions of the switch in the
// named parse function: every admitted member must be spelled as its
// const identifier, so an inline literal smuggled into the switch
// reddens here even though the const census cannot see it.
func derivedCaseIdents(t *testing.T, file, function string) []string {
	t.Helper()
	parsed := parseProductionFile(t, file)
	var idents []string
	found := false
	ast.Inspect(parsed, func(node ast.Node) bool {
		decl, ok := node.(*ast.FuncDecl)
		if !ok || decl.Name.Name != function || found {
			return true
		}
		ast.Inspect(decl.Body, func(inner ast.Node) bool {
			clause, ok := inner.(*ast.CaseClause)
			if !ok {
				return true
			}
			found = true
			for _, expr := range clause.List {
				ident, ok := expr.(*ast.Ident)
				if !ok {
					t.Errorf("%s %s case is not a const identifier", file, function)
					continue
				}
				idents = append(idents, ident.Name)
			}
			return true
		})
		return false
	})
	if !found {
		t.Fatalf("%s %s has no case clause", file, function)
	}
	return idents
}

func parseProductionFile(t *testing.T, file string) *ast.File {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	path := filepath.Join(filepath.Dir(current), file)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", file, err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), path, raw, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s) error = %v", file, err)
	}
	return parsed
}

func stringsTrimQuotes(value string) string {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}
	return value
}

func requireExactVocabulary(t *testing.T, file, typeName, function string, want []string) {
	t.Helper()
	names, literals := derivedConstLiterals(t, file, typeName)
	counts := make(map[string]int)
	for _, literal := range literals {
		counts[literal]++
	}
	for literal, count := range counts {
		if count != 1 {
			t.Errorf("%s %s spells %q %d times, want exactly once", file, typeName, literal, count)
		}
	}
	sortedGot := append([]string(nil), literals...)
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedGot)
	sort.Strings(sortedWant)
	if len(sortedGot) != len(sortedWant) {
		t.Fatalf("%s %s vocabulary = %q, want %q", file, typeName, sortedGot, sortedWant)
	}
	for i := range sortedGot {
		if sortedGot[i] != sortedWant[i] {
			t.Fatalf("%s %s vocabulary = %q, want %q", file, typeName, sortedGot, sortedWant)
		}
	}
	idents := derivedCaseIdents(t, file, function)
	sortedIdents := append([]string(nil), idents...)
	sortedNames := append([]string(nil), names...)
	sort.Strings(sortedIdents)
	sort.Strings(sortedNames)
	if len(sortedIdents) != len(sortedNames) {
		t.Fatalf("%s %s admits %q, const block declares %q", file, function, sortedIdents, sortedNames)
	}
	for i := range sortedIdents {
		if sortedIdents[i] != sortedNames[i] {
			t.Fatalf("%s %s admits %q, const block declares %q", file, function, sortedIdents, sortedNames)
		}
	}
}

// TestGenerationBoundAgreesWithLandedDigest proves the Go-level
// generation bound delegates its verdict to the landed GenerationDigest
// arm: both admit 1..256 characters (ASCII and multibyte alike) and
// both refuse 0 and 257+ with the landed
// terminal_backend_stale_generation class.
func TestGenerationBoundAgreesWithLandedDigest(t *testing.T) {
	corpus := []string{
		"",
		"a",
		strings.Repeat("a", 255),
		strings.Repeat("a", 256),
		strings.Repeat("a", 257),
		strings.Repeat("é", 256),
		strings.Repeat("é", 257),
		"generation-one",
	}
	for _, generation := range corpus {
		_, landedErr := terminalbackend.GenerationDigest(generation)
		mine := checkGenerationBound(generation)
		if (landedErr == nil) != (mine == nil) {
			t.Errorf("generation %q: landed verdict = %v, local verdict = %v, want agreement", generation, landedErr, mine)
			continue
		}
		if mine != nil {
			requireRefusal(t, mine, "terminal_backend_stale_generation", "backend_generation bound")
			if ErrorCode(landedErr) != "terminal_backend_stale_generation" {
				t.Errorf("generation %q: landed code = %q, want the stale class", generation, ErrorCode(landedErr))
			}
		}
	}
}

// TestClosedEnumsAreExactlyPinned derives the vocabulary of every closed
// enum this package owns from the production const block and cross-checks
// it against the parser switch: an added, removed or re-spelled member
// reddens the census, a member spelled twice reddens the copy count,
// and a literal smuggled into the switch outside the const block reddens
// the cross-check.
func TestClosedEnumsAreExactlyPinned(t *testing.T) {
	requireExactVocabulary(t, "disposition.go", "RetryDisposition", "ParseRetryDisposition",
		[]string{"replay_same", "status_first", "new_authorization", "required_operator_action"})
	requireExactVocabulary(t, "proof.go", "ProviderProofKind", "ParseProviderProofKind",
		[]string{"provider_quiescence", "provider_process_exit", "ax_checkpoint_boundary"})
	requireExactVocabulary(t, "auth.go", "AuthorizationKind", "ParseAuthorizationKind",
		[]string{"create", "control", "force_stale", "restore"})
}
