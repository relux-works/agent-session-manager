package cloneplanning

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// This file pins the authority-by-type shape structurally: in
// ProjectVisibleText the Authority field is assigned exactly once,
// unconditionally, from the authorityUserContext constant (or the
// literal "user_context" it quotes). No branch on text, kind,
// ordinal, or event IDs can reach the authority decision, because
// no branch reaches it at all. Control plants (a conditional
// authority, a foreign literal) redden this test; logs ride the
// evidence tar.

// authorityShape describes the Authority assignments found in
// ProjectVisibleText.
type authorityShape struct {
	assignments int
	conditional int
	foreign     int
}

func inspectAuthorityShape(t *testing.T, path string) authorityShape {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var shape authorityShape
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "ProjectVisibleText" {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch stmt := node.(type) {
			case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt:
				// Any Authority assignment nested in a
				// branch or loop counts conditional.
				ast.Inspect(stmt, func(inner ast.Node) bool {
					kv, ok := inner.(*ast.KeyValueExpr)
					if ok {
						checkAuthorityValue(t, kv, &shape, true)
					}
					assign, ok := inner.(*ast.AssignStmt)
					if ok {
						checkAuthorityAssign(t, assign, &shape, true)
					}
					return true
				})
				return false
			case *ast.KeyValueExpr:
				checkAuthorityValue(t, stmt, &shape, false)
			case *ast.AssignStmt:
				checkAuthorityAssign(t, stmt, &shape, false)
			}
			return true
		})
	}
	return shape
}

func isAuthorityKey(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "Authority"
}

func isAuthoritySelector(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Authority"
}

// checkAuthorityValue inspects one composite-literal Authority
// field for the constant shape.
func checkAuthorityValue(t *testing.T, kv *ast.KeyValueExpr, shape *authorityShape, conditional bool) {
	t.Helper()
	if !isAuthorityKey(kv.Key) {
		return
	}
	shape.assignments++
	if conditional {
		shape.conditional++
	}
	if !isAuthorityConstant(kv.Value) {
		shape.foreign++
	}
}

// checkAuthorityAssign inspects one Authority assignment for the
// constant shape.
func checkAuthorityAssign(t *testing.T, assign *ast.AssignStmt, shape *authorityShape, conditional bool) {
	t.Helper()
	for i, target := range assign.Lhs {
		if !isAuthoritySelector(target) && !isAuthorityKey(target) {
			continue
		}
		if i >= len(assign.Rhs) {
			continue
		}
		shape.assignments++
		if conditional {
			shape.conditional++
		}
		if !isAuthorityConstant(assign.Rhs[i]) {
			shape.foreign++
		}
	}
}

// isAuthorityConstant reports whether the value is the authority
// constant or the literal it quotes.
func isAuthorityConstant(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "authorityUserContext"
	}
	if lit, ok := expr.(*ast.BasicLit); ok {
		return lit.Value == `"user_context"`
	}
	return false
}

func TestAuthorityDecisionShape(t *testing.T) {
	// The constant quotes SPEC.v0.7.0.md:10810 literally.
	if authorityUserContext != "user_context" {
		t.Fatalf("authority constant %q, want user_context", authorityUserContext)
	}
	shape := inspectAuthorityShape(t, "visible.go")
	if shape.assignments != 1 {
		t.Fatalf("ProjectVisibleText assigns Authority %d times, want exactly 1", shape.assignments)
	}
	if shape.conditional != 0 {
		t.Fatalf("ProjectVisibleText assigns Authority under a branch or loop")
	}
	if shape.foreign != 0 {
		t.Fatalf("ProjectVisibleText assigns Authority from a foreign value")
	}
}
