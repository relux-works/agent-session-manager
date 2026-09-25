package clonereadback_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This file pins the no-second-implementation shape structurally:
// production files name no owner-inner members and no
// fidelity/plan vocabulary, and call no fidelity/plan predicate.
// Any literal below appearing in production code is a forked rule.
// Control plants (a re-added owner literal, a re-added vocabulary
// call) redden this test; logs ride the evidence tar.

// forkedLiterals lists owner-inner member names and owner
// vocabulary tokens that must never appear as string literals in
// production files. The package's own members (mode, evidence
// kinds, tuple/binding/findings CONTAINER members) are excluded:
// only the INNER names the owners gate are fork signals.
var forkedLiterals = []string{
	// Environment Tuple inner members (sessadapter.DecodeTuple).
	"environment_id", "environment_version", "store_schema_fingerprint", "adapter_version",
	// Workspace Binding inner members
	// (clonebundle.DecodeWorkspaceBinding).
	"logical_workspace_id", "cwd_relative", "repository_remote_fingerprints",
	"head_digest", "index_digest", "working_tree_digest",
	// Adapter Finding inner members (sessadapter.DecodeFinding):
	// severity VALUES info|warning are decided only through the
	// decoded struct field, never matched as raw members.
	"remediation",
	// Fidelity disposition vocabulary (clonefidelity).
	"exact", "semantic", "summarized", "opaque_preserved", "synthesized", "omitted", "unrecoverable",
	// Fidelity reason vocabulary sample (clonefidelity): core
	// codes plus the extension grammar marker.
	"target_no_equivalent", "source_not_persisted", "derived_index_rebuilt",
	// Projection Plan vocabulary (cloneplan).
	"same_environment_native_rewrite", "target_native_writer", "target_official_import",
	"continuation_context", "strict_exact", "maximal_safe", "compact", "messages_only",
	"create_directory", "write_blob", "write_native_record", "rebuild_index",
}

// forkedSelectors lists owner predicates that must never be called
// from production files.
var forkedSelectors = []string{
	"ValidDisposition", "ValidReasonCode", "IsCoreReasonCode",
	"ValidateDispositionRow", "BuildFidelityReport", "DecodeFidelityReport",
	"ValidPlanStrategy", "ValidPlanProfile", "ValidOperationAction",
	"ValidSynthesizedPurpose", "ValidResourceKind",
	"BuildProjectionPlan", "DecodeProjectionPlan",
	"BuildProjectedObjectManifest", "DecodeProjectedObjectManifest",
}

// TestNoForkedLiterals pins that no production file carries an
// owner-inner member literal or an owner vocabulary literal.
func TestNoForkedLiterals(t *testing.T) {
	dir := packageDir(t)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			for _, forked := range forkedLiterals {
				if value == forked {
					t.Errorf("%s carries forked literal %q: the rule belongs to its owner", path, forked)
				}
			}
			return true
		})
	}
}

// TestNoForkedSelectors pins that no production file calls a
// fidelity or plan owner predicate: with no row-shaped data, any
// call would validate a re-implemented copy.
func TestNoForkedSelectors(t *testing.T) {
	dir := packageDir(t)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			for _, forked := range forkedSelectors {
				if selector.Sel.Name == forked {
					t.Errorf("%s calls forked selector %s: the rule belongs to its owner", path, forked)
				}
			}
			return true
		})
	}
}

// TestErrorSeverityDecidedOnDecodedField pins that the no-error
// finding rule reads the owner-decoded Finding struct field —
// finding.Severity — exactly once, inside the shared decider:
// never a raw member lookup, never a second matcher. A second
// matcher would be a forked severity rule.
func TestErrorSeverityDecidedOnDecodedField(t *testing.T) {
	dir := packageDir(t)
	source, err := os.ReadFile(filepath.Join(dir, "report.go"))
	if err != nil {
		t.Fatalf("read report.go: %v", err)
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "report.go", source, 0)
	if err != nil {
		t.Fatalf("parse report.go: %v", err)
	}
	countSeverity := func(fn *ast.FuncDecl) int {
		matches := 0
		if fn.Body == nil {
			return 0
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			binary, ok := node.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			if isSeverityOperand(binary.X) || isSeverityOperand(binary.Y) {
				matches++
			}
			return true
		})
		return matches
	}
	deciderMatches := 0
	otherMatches := 0
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == "decideValid" {
			deciderMatches += countSeverity(fn)
		} else {
			otherMatches += countSeverity(fn)
		}
	}
	if deciderMatches != 1 {
		t.Errorf("Severity matched %d times inside decideValid, want exactly 1", deciderMatches)
	}
	if otherMatches != 0 {
		t.Errorf("Severity matched %d times outside decideValid: a second severity rule is a fork", otherMatches)
	}
}

// isSeverityOperand reports whether one binary operand reads a
// Severity struct field.
func isSeverityOperand(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Severity"
}
