package clonereconcile_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// This file pins two evidence-construction rules: no test in this
// package skips by default (the allowlist is empty and justified
// below), and every production refusal site appears in the gate
// reachability matrix in TRACEABILITY.md while every matrix row
// names a production literal. An unlisted site fails the census.

// TestNoDefaultSkippedEvidenceTests pins that no evidence test in
// this package skips: every Test runs its assertions on every
// platform the suite configures. The allowlist is empty because
// the package is pure Go with no platform, network, or terminal
// dependencies — there is no shape a skip could legitimately
// cover.
func TestNoDefaultSkippedEvidenceTests(t *testing.T) {
	dir := packageDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	skipPattern := regexp.MustCompile(`\bt\.Skip(f|Now)?\(|\bSkip\(|\bSkipNow\(`)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		// Strip line comments and the pattern definitions
		// themselves: mentions of skips in prose and the
		// census regex data are not skippable calls.
		var code []string
		for _, line := range strings.Split(string(source), "\n") {
			if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(line, "MustCompile") {
				continue
			}
			code = append(code, line)
		}
		if skipPattern.MatchString(strings.Join(code, "\n")) {
			t.Errorf("%s skips an evidence test; the allowlist is empty", name)
		}
	}
}

// TestSkipCensusControlPlant proves the skip census bites: a
// planted skip is reported. The token assembles at runtime so the
// plant itself carries no skippable call for the census to flag.
func TestSkipCensusControlPlant(t *testing.T) {
	plant := "func TestPlanted(t *testing.T) {\n\tt." + "Ski" + "p(\"planted\")\n}\n"
	skipPattern := regexp.MustCompile(`\bt\.Skip(f|Now)?\(|\bSkip\(|\bSkipNow\(`)
	if !skipPattern.MatchString(plant) {
		t.Fatalf("control plant skip not detected")
	}
}

// productionRefusalLiterals extracts every refuse("...") literal
// from the package's non-test sources.
func productionRefusalLiterals(t *testing.T) []string {
	t.Helper()
	dir := packageDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	pattern := regexp.MustCompile(`refuse\("((?:[^"\\]|\\.)*)"`)
	var literals []string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
			literals = append(literals, match[1])
		}
	}
	return literals
}

// matrixSection extracts the gate reachability matrix section from
// TRACEABILITY.md: the lines between the matrix heading and the
// next heading.
func matrixSection(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(packageDir(t), "TRACEABILITY.md"))
	if err != nil {
		t.Fatalf("read TRACEABILITY.md: %v", err)
	}
	lines := strings.Split(string(raw), "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Gate reachability matrix" {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("TRACEABILITY.md carries no gate reachability matrix")
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

// missingFromMatrix returns the production literals absent from
// the matrix text.
func missingFromMatrix(literals []string, matrix string) []string {
	var missing []string
	for _, literal := range literals {
		if !strings.Contains(matrix, literal) {
			missing = append(missing, literal)
		}
	}
	return missing
}

// matrixLiteralCells returns the first backticked cell of every
// matrix table row (header and separator excluded).
func matrixLiteralCells(matrix string) []string {
	tickPattern := regexp.MustCompile("`([^`]*)`")
	var cells []string
	for _, line := range strings.Split(matrix, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "| `") {
			continue
		}
		match := tickPattern.FindStringSubmatch(trimmed)
		if len(match) == 2 {
			cells = append(cells, match[1])
		}
	}
	return cells
}

// TestGateReachabilityMatrixDerivedFromRefusalSites pins the matrix
// mechanically: every production refusal literal appears verbatim
// in the matrix section, and every matrix literal cell names a
// production literal (or a documented owner: delegation). An
// unlisted site fails here instead of silently escaping the gate
// table.
func TestGateReachabilityMatrixDerivedFromRefusalSites(t *testing.T) {
	matrix := matrixSection(t)
	literals := productionRefusalLiterals(t)
	for _, missing := range missingFromMatrix(literals, matrix) {
		t.Errorf("refusal site %q is absent from the gate reachability matrix", missing)
	}
	for _, cell := range matrixLiteralCells(matrix) {
		if strings.HasPrefix(cell, "owner:") {
			continue
		}
		found := false
		for _, literal := range literals {
			if strings.Contains(literal, cell) || strings.Contains(cell, literal) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("matrix literal %q names no production refusal site", cell)
		}
	}
}

// TestGateReachabilityControlPlant proves the matrix census bites:
// a production literal missing from a synthetic matrix is
// reported, and a matrix cell naming nothing is reported.
func TestGateReachabilityControlPlant(t *testing.T) {
	plantLiterals := []string{"tier-9 plant refuses here"}
	plantMatrix := "## Gate reachability matrix\n\n| `tier-9 other` | plant |\n"
	if missing := missingFromMatrix(plantLiterals, plantMatrix); len(missing) != 1 {
		t.Fatalf("missingFromMatrix(plant) = %q, want the plant literal", missing)
	}
	cells := matrixLiteralCells(plantMatrix)
	if len(cells) != 1 || cells[0] != "tier-9 other" {
		t.Fatalf("matrixLiteralCells(plant) = %q, want the plant cell", cells)
	}
}
