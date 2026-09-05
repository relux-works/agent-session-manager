package provider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// sectionLines joins the pinned document lines for section 7.1, failing
// when the section boundaries move. Every citation below resolves against
// the real text, never against a restatement in this repository.
func sectionLines(t *testing.T, document *specdoc.Document, section string, first, last int) string {
	t.Helper()
	for _, line := range []int{first, last} {
		got, ok := document.SectionID(line)
		if !ok || got != section {
			t.Fatalf("SPEC.md line %d is in section %q, want %q", line, got, section)
		}
	}
	var body strings.Builder
	for line := first; line <= last; line++ {
		text, ok := document.Line(line)
		if !ok {
			t.Fatalf("SPEC.md line %d is missing", line)
		}
		body.WriteString(text)
		body.WriteString("\n")
	}
	return body.String()
}

// requireQuote asserts the excerpt occurs verbatim in the pinned document
// and begins on a line inside the cited section.
func requireQuote(t *testing.T, document *specdoc.Document, excerpt, section string) {
	t.Helper()
	if !document.Contains(excerpt) {
		t.Fatalf("pinned document does not contain %q", excerpt)
	}
	lines := document.QuoteLines(excerpt)
	if len(lines) == 0 {
		t.Fatalf("pinned document quotes no line for %q", excerpt)
	}
	got, ok := document.SectionID(lines[0])
	if !ok || got != section {
		t.Fatalf("%q begins on SPEC.md line %d in section %q, want %q", excerpt, lines[0], got, section)
	}
	t.Logf("%q begins on SPEC.md line %d", excerpt, lines[0])
}

// TestSection71BuiltinRegistryIsDocumentOrder proves the builtin registry
// enumerates exactly the six Section 7.1 built-ins in the section's listed
// order. The order is derived from the pinned text: each name's first
// tagged occurrence inside Section 7.1 must appear in registry order.
func TestSection71BuiltinRegistryIsDocumentOrder(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.1", 2622, 2653)
	names := []string{"codex", "claude", "gemini", "muse", "antigravity", "pi"}
	previous := -1
	for _, name := range names {
		offset := strings.Index(window, "<code>"+name+"</code>")
		if offset == -1 {
			t.Fatalf("Section 7.1 does not list builtin %q", name)
		}
		if offset <= previous {
			t.Fatalf("Section 7.1 lists %q out of registry order", name)
		}
		previous = offset
	}
	got := Builtins()
	if len(got) != len(names) {
		t.Fatalf("Builtins() = %v, want the six Section 7.1 names", got)
	}
	for i, name := range names {
		if got[i] != name {
			t.Fatalf("Builtins() = %v, want Section 7.1 order %v", got, names)
		}
	}
	requireQuote(t, document, "Built-in support covers", "7.1")
}

// TestSection71DiscoveryOrderIsPinned quotes the Section 7.1 discovery
// order behind Discover's source ranking and the PATH gate.
func TestSection71DiscoveryOrderIsPinned(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	for _, excerpt := range []string{
		"configured <code>providers.plugin_dirs</code> in listed order",
		"built-in adapters",
		"<code>PATH</code>, only when <code>allow_path_plugins</code> is true",
		"The order does not establish precedence",
	} {
		requireQuote(t, document, excerpt, "7.1")
	}
}

// TestSection71DuplicateRefusalIsPinned quotes the duplicate rule and its
// mandated code behind the duplicate refusal.
func TestSection71DuplicateRefusalIsPinned(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	requireQuote(t, document,
		"discovery MUST fail with <code>invalid_config</code> before either candidate is probed or executed",
		"7.1")
	requireQuote(t, document, "the operator must remove or rename one candidate", "7.1")
}

// TestSection71TrustRulesArePinned quotes the trust recording, symlink,
// regular-file, ownership, and renewed-trust rules behind Trust and
// Verify.
func TestSection71TrustRulesArePinned(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	for _, excerpt := range []string{
		"External provider executables are named <code>ax-provider-&lt;id&gt;</code>",
		"<code>id</code> matches <code>[a-z][a-z0-9-]{0,31}</code>",
		"the canonical absolute executable path and SHA-256 digest MUST be recorded at trust time",
		"Symlinks MUST be resolved before comparison",
		"the target MUST be a regular file owned by the operator or an administrator-approved identity",
		"a changed path target or digest MUST require renewed trust",
		"they MUST receive only the minimum operation-specific inputs",
	} {
		requireQuote(t, document, excerpt, "7.1")
	}
}

// TestSection71AdvertisesNoPublicSDK quotes the M0 non-advertisement rule
// and proves the package honors it structurally: no exported symbol name
// offers an SDK, client, stable, or public surface.
func TestSection71AdvertisesNoPublicSDK(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	requireQuote(t, document, "M0 MUST NOT advertise a public stable plugin SDK", "7.1")
	symbols := exportedSymbols(t)
	if len(symbols) == 0 {
		t.Fatal("derived no exported symbols; the scan is blind")
	}
	for _, symbol := range symbols {
		lowered := strings.ToLower(symbol)
		for _, banned := range []string{"sdk", "stable", "public", "client"} {
			if strings.Contains(lowered, banned) {
				t.Fatalf("exported symbol %q advertises a public SDK surface", symbol)
			}
		}
	}
}

// TestCandidatesAdvertiseNoCapability proves discovery never advertises
// what the probe plane has not proven: no Candidate or TrustRecord member
// carries availability, status, enabled, support, or capability state. The
// member list is derived from package source, never hand-listed.
func TestCandidatesAdvertiseNoCapability(t *testing.T) {
	members := structMembers(t, "Candidate", "TrustRecord")
	for member, typ := range members {
		lowered := strings.ToLower(member)
		for _, banned := range []string{"availab", "status", "enabled", "support", "capab"} {
			if strings.Contains(lowered, banned) {
				t.Fatalf("member %q (%s) advertises capability state", member, typ)
			}
		}
	}
	if len(members) == 0 {
		t.Fatal("derived no candidate members; the scan is blind")
	}
	t.Logf("scanned %d Candidate/TrustRecord members, none carries capability state", len(members))
}

// TestDiscoveryReachesNoProcess proves the ordering guarantee behind the
// duplicate refusal: no package source imports a process-execution
// facility, so Discover cannot probe or execute a candidate. The import
// list is derived from package source.
func TestDiscoveryReachesNoProcess(t *testing.T) {
	fileset := token.NewFileSet()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	found := false
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		found = true
		source, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		syntax, err := parser.ParseFile(fileset, name, source, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		for _, clause := range syntax.Imports {
			path := strings.Trim(clause.Path.Value, `"`)
			if path == "os/exec" || strings.HasPrefix(path, "os/exec/") {
				t.Fatalf("%s imports process execution (%s)", name, path)
			}
		}
	}
	if !found {
		t.Fatal("scanned no production sources; the check is blind")
	}
}

// TestStableCodesAreRegistered pins every refusal code this package emits
// against the pinned error registry and the exact exit status the host
// surfaces: invalid_config and local_precondition_failed exit 3,
// integrity_failure exits 9.
func TestStableCodesAreRegistered(t *testing.T) {
	registered := map[string]int{}
	for _, row := range catalog.Current().Errors {
		registered[string(row.Code)] = row.ExitCode
	}
	for code, wantExit := range map[string]int{
		codeInvalidConfig:     3,
		codeLocalPrecondition: 3,
		codeIntegrityFailure:  9,
	} {
		gotExit, ok := registered[code]
		if !ok {
			t.Fatalf("code %q is not in the pinned error registry", code)
		}
		if gotExit != wantExit {
			t.Fatalf("code %q exit = %d, want %d", code, gotExit, wantExit)
		}
		gotWire, err := axerror.ExitCodeFor(axerror.Version100, axerror.Code(code))
		if err != nil {
			t.Fatalf("ExitCodeFor(1.0.0, %q): %v", code, err)
		}
		if gotWire != wantExit {
			t.Fatalf("wire exit for %q = %d, want %d", code, gotWire, wantExit)
		}
	}
	t.Logf("refusal code coverage: 3/3 package codes registered with pinned exits")
}

// exportedSymbols derives every exported top-level symbol name from
// package source.
func exportedSymbols(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, syntax := range parseProductionSources(t) {
		for _, decl := range syntax.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range general.Specs {
				switch concrete := spec.(type) {
				case *ast.TypeSpec:
					if ast.IsExported(concrete.Name.Name) {
						out = append(out, concrete.Name.Name)
					}
				case *ast.ValueSpec:
					for _, ident := range concrete.Names {
						if ast.IsExported(ident.Name) {
							out = append(out, ident.Name)
						}
					}
				}
			}
		}
		for _, decl := range syntax.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Name == nil || !ast.IsExported(function.Name.Name) {
				continue
			}
			if function.Recv == nil {
				out = append(out, function.Name.Name)
			} else {
				out = append(out, function.Name.Name+" (method)")
			}
		}
	}
	return out
}

// structMembers derives field names and types for the named structs from
// package source.
func structMembers(t *testing.T, wanted ...string) map[string]string {
	t.Helper()
	want := map[string]bool{}
	for _, name := range wanted {
		want[name] = true
	}
	members := map[string]string{}
	for _, syntax := range parseProductionSources(t) {
		ast.Inspect(syntax, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok || !want[spec.Name.Name] {
				return true
			}
			structure, ok := spec.Type.(*ast.StructType)
			if !ok {
				return false
			}
			for _, field := range structure.Fields.List {
				for _, ident := range field.Names {
					members[ident.Name] = spec.Name.Name
				}
			}
			return false
		})
	}
	return members
}

// parseProductionSources parses every non-test Go source in the package
// directory.
func parseProductionSources(t *testing.T) map[string]*ast.File {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	fileset := token.NewFileSet()
	out := map[string]*ast.File{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		syntax, err := parser.ParseFile(fileset, name, source, parser.ParseComments)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", name, err)
		}
		out[name] = syntax
	}
	return out
}
