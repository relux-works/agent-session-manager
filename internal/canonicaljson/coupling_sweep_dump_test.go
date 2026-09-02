package canonicaljson

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

type sweepSite struct {
	Kind     string `json:"kind"`
	Key      string `json:"key"`
	File     string `json:"file"`
	Function string `json:"function"`
	Op       string `json:"op"`
	Text     string `json:"text"`
	Offset   int    `json:"offset"`
	End      int    `json:"end"`
	Left     string `json:"left"`
	Right    string `json:"right"`
	Literal  int    `json:"literal"`
	LeftLit  bool   `json:"left_literal"`
	NonNeg   bool   `json:"non_negative"`
}

// TestDumpSweepSites is a harness hook, not a gate: it writes the derived
// coupling and literal-boundary sites with their exact source ranges so the
// external mutation sweep generates mutants from the same derivation the
// in-repo gates use, rather than from a hand-picked list.
func TestDumpSweepSites(t *testing.T) {
	destination := os.Getenv("AX_SWEEP_DUMP")
	if destination == "" {
		t.Skip("AX_SWEEP_DUMP not set")
	}
	directory, paths := packageProductionFiles(t)
	_ = directory
	offsets := map[string][2]int{}
	fileSet := token.NewFileSet()
	for _, path := range paths {
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		base := filepath.Base(path)
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				expression, ok := node.(*ast.BinaryExpr)
				if !ok {
					return true
				}
				offsets[base+"|"+function.Name.Name+"|"+renderComparison(expression)] = [2]int{
					fileSet.Position(expression.Pos()).Offset,
					fileSet.Position(expression.End()).Offset,
				}
				return true
			})
		}
	}

	var sites []sweepSite
	presence := map[string]bool{}
	for _, site := range derivePresenceCouplingSites(t) {
		presence[site.key] = true
	}
	literal := map[string]literalBoundarySite{}
	for _, site := range deriveLiteralBoundarySites(t) {
		literal[site.key] = site
	}

	forEachProductionComparison(t, func(context comparisonContext) {
		key := context.key()
		span, ok := offsets[key]
		if !ok {
			return
		}
		source, err := os.ReadFile(filepath.Join(directory, context.file))
		if err != nil {
			t.Fatal(err)
		}
		entry := sweepSite{
			Key: key, File: context.file, Function: context.function,
			Op: context.expression.Op.String(), Text: string(source[span[0]:span[1]]),
			Left:   renderComparison(context.expression.X),
			Right:  renderComparison(context.expression.Y),
			Offset: span[0], End: span[1],
		}
		if presence[key] {
			entry.Kind = "presence"
			sites = append(sites, entry)
		}
		if boundary, ok := literal[key]; ok {
			copied := entry
			copied.Kind = "literal"
			copied.Literal = boundary.literal
			_, _, _ = context.integerLiteralComparison()
			copied.LeftLit = context.literalOnLeft
			copied.NonNeg = len(boundary.values) > 0 && boundary.values[0] >= 0 && boundary.literal == 0 || boundary.values[0] >= 0
			sites = append(sites, copied)
		}
	})
	encoded, err := json.MarshalIndent(sites, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d sites to %s", len(sites), destination)
}
