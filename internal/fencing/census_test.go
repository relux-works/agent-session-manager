package fencing

import (
	"go/ast"
	"go/token"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// sealConstructorFile is the only production file that may name the
// seal types or assemble a non-empty LeaseToken.
const sealConstructorFile = "token.go"

// auditSealBoundary derives every seal-construction reference outside
// the constructor file: any occurrence of the seal idents, and any
// non-empty LeaseToken composite literal. The walk is over AST
// idents, so comments and string literals that mention the seal are
// not references. A display name and parsed file let the same audit
// run over control plants.
func auditSealBoundary(display string, syntax *ast.File, fileSet *token.FileSet) []string {
	var violations []string
	position := func(node ast.Node) string {
		place := fileSet.Position(node.Pos())
		return display + ":" + strconv.Itoa(place.Line)
	}
	ast.Inspect(syntax, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && (ident.Name == "leaseSeal" || ident.Name == "leaseSealToken") {
			violations = append(violations, "seal type reference outside "+sealConstructorFile+": "+position(ident))
			return true
		}
		literal, ok := node.(*ast.CompositeLit)
		if !ok || len(literal.Elts) == 0 {
			return true
		}
		if ident, ok := literal.Type.(*ast.Ident); ok && ident.Name == "LeaseToken" {
			violations = append(violations, "non-empty LeaseToken literal outside "+sealConstructorFile+": "+position(literal))
		}
		return true
	})
	return violations
}

// TestLeaseTokenConstructorCensus pins the sealed-capability
// construction boundary: only token.go names the seal types or
// assembles a non-empty LeaseToken, and token.go carries exactly one
// mint site. gate.go's empty LeaseToken{} refusals are not authority
// and stay allowed.
func TestLeaseTokenConstructorCensus(t *testing.T) {
	files, fileSet := invcore.MustScanProduction(t, ".")
	var violations []string
	mints := 0
	for _, file := range files {
		if file.Name == sealConstructorFile {
			mints += strings.Count(string(file.Source), "&leaseSealToken{}")
			continue
		}
		violations = append(violations, auditSealBoundary(file.Name, file.Syntax, fileSet)...)
	}
	for _, violation := range violations {
		t.Error(violation)
	}
	if mints != 1 {
		t.Errorf("mint sites = %d, want exactly one &leaseSealToken{} in %s", mints, sealConstructorFile)
	}
}

// sealPlants rosters the control plants the census must accept or
// reject. The alias plant preserves the searched-for token
// (leaseSeal) while constructing through a renamed spelling: a
// text search for the construction shape would miss it, so the
// ident-based audit must still reject it.
func sealPlants() map[string]struct {
	source string
	reject bool
} {
	return map[string]struct {
		source string
		reject bool
	}{
		"comment_mentions_seal": {
			source: "package fencing\n\n// leaseSeal and leaseSealToken are constructor-only; LeaseToken{seal: x} never appears here.\nfunc plantComment() {}\n",
			reject: false,
		},
		"string_mentions_seal": {
			source: "package fencing\n\nvar plantString = \"leaseSeal{token: &leaseSealToken{}}\"\n\nfunc plantUse() string { return plantString }\n",
			reject: false,
		},
		"alias_backdoor": {
			source: "package fencing\n\ntype plantBackdoor = leaseSeal\n\nfunc plantAlias() LeaseToken { return LeaseToken{seal: &plantBackdoor{}} }\n",
			reject: true,
		},
		"direct_forgery": {
			source: "package fencing\n\nfunc plantDirect() LeaseToken { return LeaseToken{seal: &leaseSeal{token: &leaseSealToken{}}} }\n",
			reject: true,
		},
		"shadowed_seal": {
			source: "package fencing\n\ntype leaseSeal struct{ note string }\n\nfunc plantShadow() string { return leaseSeal{note: \"x\"}.note }\n",
			reject: true,
		},
	}
}

// TestLeaseTokenCensusPlants executes the control plants through the
// same audit the production census runs: comment and string mentions
// of the seal are not references, while alias, direct, and shadowing
// forgeries are rejected.
func TestLeaseTokenCensusPlants(t *testing.T) {
	for name, plant := range sealPlants() {
		t.Run(name, func(t *testing.T) {
			syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(plant.source))
			if failure != "" {
				t.Fatalf("parse plant: %s", failure)
			}
			violations := auditSealBoundary("plant.go", syntax, fileSet)
			if plant.reject && len(violations) == 0 {
				t.Fatalf("plant %s accepted, want rejection", name)
			}
			if !plant.reject && len(violations) != 0 {
				t.Fatalf("plant %s rejected: %v", name, violations)
			}
		})
	}
}

// TestLeaseTokenUnforgeableOutsidePackage proves the seal cannot be
// spelled from outside: a fixture package that assembles a LeaseToken
// by field name fails to build, while a fixture that uses only the
// public Authorize/Bind surface builds cleanly.
func TestLeaseTokenUnforgeableOutsidePackage(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	dir := filepath.Dir(file)
	build := func(t *testing.T, target string) (int, string) {
		t.Helper()
		command := exec.Command("go", "build", target)
		command.Dir = filepath.Join(dir, "..", "..")
		output, err := command.CombinedOutput()
		if err == nil {
			return 0, string(output)
		}
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("go build %s: %v", target, err)
		}
		return exit.ExitCode(), string(output)
	}
	if code, output := build(t, "./internal/fencing/testdata/use"); code != 0 {
		t.Fatalf("public-surface fixture fails to build (exit %d): %s", code, output)
	}
	code, output := build(t, "./internal/fencing/testdata/forge")
	if code == 0 {
		t.Fatal("forgery fixture builds, want a compile failure on the unexported seal")
	}
	if !strings.Contains(output, "seal") && !strings.Contains(output, "unknown field") {
		t.Fatalf("forgery build fails without naming the seal (exit %d): %s", code, output)
	}
}
