package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/traceability"
)

func TestRunReportsExactCoverageAndFailsClosed(t *testing.T) {
	t.Parallel()

	repositoryRoot := filepath.Join("..", "..", "..", "..")
	var output bytes.Buffer
	if err := run([]string{"-root", repositoryRoot}, &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	want := "traceability ok: contracts=60 normative_sections=36 acceptance_cases=77 fixtures=30 compatibility_contracts=55 assigned_scopes=0\n" +
		"section coverage: bindings=49 full=1 partial=3 sliver=1 unevidenced=41 unmeasured=3 unowned=2 clauses_discharged=17/403\n"
	if output.String() != want {
		t.Fatalf("run() output = %q, want %q", output.String(), want)
	}

	output.Reset()
	err := run([]string{"-root", t.TempDir()}, &output)
	if err == nil || !strings.Contains(err.Error(), "read required evidence") {
		t.Fatalf("run(missing repository) error = %v, want required-evidence refusal", err)
	}
	if output.Len() != 0 {
		t.Fatalf("failed run emitted success output %q", output.String())
	}

	if err := run([]string{"unexpected"}, &output); err == nil {
		t.Fatal("run(positional argument) error = nil, want refusal")
	}
	if err := run([]string{"-unknown"}, io.Discard); err == nil {
		t.Fatal("run(unknown flag) error = nil, want refusal")
	}

	writeErr := errors.New("write failed")
	err = run([]string{"-root", repositoryRoot}, failingWriter{err: writeErr})
	if !errors.Is(err, writeErr) {
		t.Fatalf("run(failing writer) error = %v, want %v", err, writeErr)
	}

	output.Reset()
	err = run([]string{"-root", repositoryRoot, "-section", "6.2"}, &output)
	if err != nil {
		t.Fatalf("run(assigned sections) error = %v", err)
	}
	want = "traceability ok: contracts=60 normative_sections=36 acceptance_cases=77 fixtures=30 compatibility_contracts=55 assigned_scopes=1\n" +
		"section coverage: bindings=49 full=1 partial=3 sliver=1 unevidenced=41 unmeasured=3 unowned=2 clauses_discharged=17/403\n"
	if output.String() != want {
		t.Fatalf("run(assigned sections) output = %q, want %q", output.String(), want)
	}

	// This invocation used to exit 0. Section 13.14.5 was admitted because the
	// obligation scanner measured zero clauses under it, which the gate read as
	// "carries no obligation". It reads as "cannot be measured" now, and the
	// command refuses the whole scope and emits no success line.
	output.Reset()
	err = run([]string{"-root", repositoryRoot, "-section", "6.2", "-section", "13.14.5"}, &output)
	if err == nil || !errors.Is(err, traceability.ErrTraceability) ||
		!strings.Contains(err.Error(), "discharges 0/0 normative clauses, which is unmeasured coverage") {
		t.Fatalf("run(-section 6.2 -section 13.14.5) error = %v, want unmeasured refusal", err)
	}
	if output.Len() != 0 {
		t.Fatalf("refused run emitted success output %q", output.String())
	}
}

// TestRunAdmitsOnlyAssignedSectionsWhoseBindingDischargesTheWholeSection is the
// admitted arm of the coverage gate at the command entry point. Section 6.2
// discharges the one normative clause its pinned section carries. It is the
// only section in the shipped registry the command admits.
//
// Section 13.14.5 was admitted here too, on the ground that its pinned section
// "carries none of its own". That was an artefact of the obligation scanner
// matching uppercase keywords only, not a fact about the section, so the
// assertion moved to TestRunRefusesEveryAssignedSectionThatOnlySlivers with its
// measured 0/0 ratio rather than being deleted.
func TestRunAdmitsOnlyAssignedSectionsWhoseBindingDischargesTheWholeSection(t *testing.T) {
	t.Parallel()

	repositoryRoot := filepath.Join("..", "..", "..", "..")
	for _, section := range []string{"6.2"} {
		var output bytes.Buffer
		if err := run([]string{"-root", repositoryRoot, "-section", section}, &output); err != nil {
			t.Errorf("run(-section %s) error = %v", section, err)
			continue
		}
		if !strings.Contains(output.String(), "assigned_scopes=1") {
			t.Errorf("run(-section %s) output = %q, want assigned_scopes=1", section, output.String())
		}
	}
}

// TestRunRefusesEveryAssignedSectionThatOnlySlivers is the disclosure this bug
// exists to produce. Every section listed here was admitted by the command
// before the coverage gate, and the README documented the whole list as the
// Story-scope validation command. Each binding names a real Go declaration and
// an executable acceptance case, and none of them discharges the section it is
// registered against, so a Story assigned one could do nothing and stay green.
// The command now refuses each with the measured ratio; the expectations below
// are the shipped state, so a section that becomes covered has to leave this
// table deliberately.
func TestRunRefusesEveryAssignedSectionThatOnlySlivers(t *testing.T) {
	t.Parallel()

	repositoryRoot := filepath.Join("..", "..", "..", "..")
	for _, test := range []struct {
		section string
		want    string
	}{
		{"1.6", "discharges 0/31 normative clauses, which is unevidenced coverage"},
		{"2.1", "discharges 0/1 normative clauses, which is unevidenced coverage"},
		{"2.2", `binding "section:2.2" is recorded unowned:`},
		{"2.3", "discharges 0/7 normative clauses, which is unevidenced coverage"},
		{"2.4", "discharges 0/4 normative clauses, which is unevidenced coverage"},
		{"3.2", "discharges 0/13 normative clauses, which is unevidenced coverage"},
		{"3.3", "discharges 0/4 normative clauses, which is unevidenced coverage"},
		{"5.1", "discharges 0/9 normative clauses, which is unevidenced coverage"},
		{"6.1", "discharges 0/2 normative clauses, which is unevidenced coverage"},
		{"6.3", "discharges 0/11 normative clauses, which is unevidenced coverage"},
		{"6.4", "discharges 0/2 normative clauses, which is unevidenced coverage"},
		{"6.5", "discharges 0/3 normative clauses, which is unevidenced coverage"},
		{"7.3", "discharges 0/0 normative clauses, which is unmeasured coverage"},
		{"7.9", "discharges 0/8 normative clauses, which is unevidenced coverage"},
		{"9.2", "discharges 0/35 normative clauses, which is unevidenced coverage"},
		{"10.1", "discharges 0/3 normative clauses, which is unevidenced coverage"},
		{"10.2", "discharges 0/5 normative clauses, which is unevidenced coverage"},
		{"10.3", "discharges 1/3 normative clauses, which is sliver coverage"},
		{"10.4", "discharges 0/25 normative clauses, which is unevidenced coverage"},
		{"13.14.5", "discharges 0/0 normative clauses, which is unmeasured coverage"},
		{"14.2", "discharges 8/9 normative clauses, which is partial coverage"},
		{"15.1", "discharges 5/7 normative clauses, which is partial coverage"},
		{"15.2", "discharges 0/0 normative clauses, which is unmeasured coverage"},
		{"15.3", "discharges 2/3 normative clauses, which is partial coverage"},
		{"17.1", "discharges 0/6 normative clauses, which is unevidenced coverage"},
		{"17.2", "discharges 0/1 normative clauses, which is unevidenced coverage"},
		{"17.3", "discharges 0/3 normative clauses, which is unevidenced coverage"},
		{"18.1", "discharges 0/5 normative clauses, which is unevidenced coverage"},
		{"18.4", `binding "section:18.4" is recorded unowned:`},
	} {
		var output bytes.Buffer
		err := run([]string{"-root", repositoryRoot, "-section", test.section}, &output)
		if err == nil || !errors.Is(err, traceability.ErrTraceability) || !strings.Contains(err.Error(), test.want) {
			t.Errorf("run(-section %s) error = %v, want ErrTraceability containing %q", test.section, err, test.want)
		}
		if output.Len() != 0 {
			t.Errorf("run(-section %s) emitted success output %q", test.section, output.String())
		}
	}
}

// TestMainRejectsRenamedScalarSectionOwnerDeclarations attacks each assigned
// scalar binding through main -> run -> VerifyAssignedSections. Each mutant
// renames the real Go declaration while leaving the reviewed registry intact.
func TestMainRejectsRenamedScalarSectionOwnerDeclarations(t *testing.T) {
	if testing.Short() {
		t.Skip("builds isolated tracecheck binaries")
	}

	tests := []struct {
		section     string
		path        string
		declaration string
		from        string
		to          string
	}{
		{"1.6", "internal/scalar/scalar.go", "ErrInvalidScalar", "var ErrInvalidScalar", "var RenamedErrInvalidScalar"},
		{"2.1", "internal/canonicaljson/closed_shapes.go", "validateSessionRecordCommon", "func validateSessionRecordCommon(", "func renamedValidateSessionRecordCommon("},
		{"2.3", "internal/canonicaljson/closed_shapes.go", "validateSessionRecordCommon", "func validateSessionRecordCommon(", "func renamedValidateSessionRecordCommon("},
		{"2.4", "internal/canonicaljson/closed_shapes.go", "validateSessionRecordCommon", "func validateSessionRecordCommon(", "func renamedValidateSessionRecordCommon("},
		{"3.2", "internal/localstore/paths.go", "ResolvePaths", "func ResolvePaths(", "func RenamedResolvePaths("},
		{"3.3", "internal/localstore/projection.go", "OpenProjection", "func OpenProjection(", "func RenamedOpenProjection("},
		{"5.1", "internal/canonicaljson/closed_shapes.go", "validateSessionRecordWithDerivation", "func validateSessionRecordWithDerivation(", "func renamedValidateSessionRecordWithDerivation("},
		{"10.1", "internal/canonicaljson/closed_shapes.go", "validateImmutableObjectShape", "func validateImmutableObjectShape(", "func renamedValidateImmutableObjectShape("},
		{"10.2", "internal/canonicaljson/closed_shapes.go", "validateBlobDescriptor", "func validateBlobDescriptor", "func renamedValidateBlobDescriptor"},
		{"10.3", "internal/canonicaljson/closed_shapes.go", "validateBlobDescriptor", "func validateBlobDescriptor", "func renamedValidateBlobDescriptor"},
		{"10.4", "internal/canonicaljson/closed_shapes.go", "validateTransferManifest", "func validateTransferManifest", "func renamedValidateTransferManifest"},
		{"17.3", "internal/canonicaljson/closed_shapes.go", "validateMigrationProvenance", "func validateMigrationProvenance", "func renamedValidateMigrationProvenance"},
	}

	for _, test := range tests {
		t.Run(test.section, func(t *testing.T) {
			fixtureRoot := isolatedTracecheckFixture(t)
			renameGoDeclaration(t, filepath.Join(fixtureRoot, test.path), test.from, test.to)

			output, err := runTracecheck(t, fixtureRoot, "-section", test.section)
			want := `section binding "section:` + test.section + `" production owner: declaration "` + test.declaration + `" is absent`
			if test.declaration == "OpenProjection" {
				want = `acceptance case "localstore-sqlite-projection" production owner: declaration "OpenProjection" is absent`
			}
			if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
				t.Fatalf("tracecheck -section %s error = %v output = %q, want refusal %q", test.section, err, output, want)
			}
		})
	}
}

// TestMainRejectsRenamedCanonicalIdentityEntryPoint proves that the assigned
// Section 1.6 gate reaches the production omit-self verifier through its
// executable acceptance case. A registry link without the call site must not
// satisfy traceability.
func TestMainRejectsRenamedCanonicalIdentityEntryPoint(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an isolated tracecheck binary")
	}

	fixtureRoot := isolatedTracecheckFixture(t)
	renameGoDeclaration(
		t,
		filepath.Join(fixtureRoot, "internal/canonicaljson/canonical.go"),
		"func VerifyObjectIdentity",
		"func RenamedVerifyObjectIdentity",
	)

	output, err := runTracecheck(t, fixtureRoot, "-section", "1.6")
	want := `acceptance case "canonical-identity-refusal" production owner: declaration "VerifyObjectIdentity" is absent`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("tracecheck -section 1.6 error = %v output = %q, want refusal %q", err, output, want)
	}
}

// TestMainRejectsOneNarrowedAssignedSectionBinding drives the production
// main -> run -> traceability.VerifyAssignedSections call chain in an isolated
// module. It re-pins the intentionally narrowed registry so the section:9.2
// refusal, rather than the existing digest gate, is what makes the test red.
// The unrelated section:6.2 binding must remain executable and green.
func TestMainRejectsOneNarrowedAssignedSectionBinding(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an isolated tracecheck binary")
	}

	repositoryRoot := filepath.Join("..", "..", "..", "..")
	fixtureRoot := t.TempDir()
	copyFile(t, filepath.Join(repositoryRoot, "go.mod"), filepath.Join(fixtureRoot, "go.mod"))
	if err := os.CopyFS(
		filepath.Join(fixtureRoot, "internal"),
		os.DirFS(filepath.Join(repositoryRoot, "internal")),
	); err != nil {
		t.Fatalf("copy repository fixture: %v", err)
	}

	registryPath := filepath.Join(fixtureRoot, "internal", "traceability", "ownership.v0.5.0.json")
	removeOwnershipKey(t, registryPath, "section:9.2")

	output, err := runTracecheck(t, fixtureRoot, "-section", "6.2")
	if err == nil || !strings.Contains(output, "projection digest ") {
		t.Fatalf("unrepinned narrowed tracecheck error = %v output = %q, want projection digest refusal", err, output)
	}
	repinOwnershipDigest(t, fixtureRoot, output)

	output, err = runTracecheck(t, fixtureRoot, "-section", "9.2")
	want := `assigned section "9.2" binding "section:9.2" has no scoped implementation owner`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("narrowed assigned tracecheck error = %v output = %q, want refusal %q and no success output", err, output, want)
	}

	output, err = runTracecheck(t, fixtureRoot, "-section", "6.2")
	if err != nil || !strings.Contains(output, "assigned_scopes=1") {
		t.Fatalf("unrelated assigned tracecheck error = %v output = %q, want green section:6.2 binding", err, output)
	}
}

// TestMainRejectsDetachedScopeSpecificAcceptanceCase drives the production
// main -> run -> traceability.VerifyAssignedSections call chain after removing
// only Section 9.2's executable acceptance link. Section 6.2 must stay green.
func TestMainRejectsDetachedScopeSpecificAcceptanceCase(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an isolated tracecheck binary")
	}

	fixtureRoot := isolatedTracecheckFixture(t)
	registryPath := filepath.Join(fixtureRoot, "internal", "traceability", "ownership.v0.5.0.json")
	removeOwnershipAcceptanceCases(t, registryPath, "section:9.2")

	output, err := runTracecheck(t, fixtureRoot, "-section", "6.2")
	if err == nil || !strings.Contains(output, "projection digest ") {
		t.Fatalf("unrepinned detached-case tracecheck error = %v output = %q, want projection digest refusal", err, output)
	}
	repinOwnershipDigest(t, fixtureRoot, output)

	output, err = runTracecheck(t, fixtureRoot, "-section", "9.2")
	want := `section binding "section:9.2" has no scope-specific acceptance owner`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("detached-case tracecheck error = %v output = %q, want refusal %q and no success output", err, output, want)
	}

	output, err = runTracecheck(t, fixtureRoot, "-section", "6.2")
	if err != nil || !strings.Contains(output, "assigned_scopes=1") {
		t.Fatalf("unrelated assigned tracecheck error = %v output = %q, want green section:6.2 binding", err, output)
	}
}

// TestMainRejectsMissingScopeSpecificProductionDeclaration drives the
// production entry point after detaching only Section 9.2 from its concrete Go
// declaration. Section 6.2 must remain independently executable and green.
func TestMainRejectsMissingScopeSpecificProductionDeclaration(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an isolated tracecheck binary")
	}

	fixtureRoot := isolatedTracecheckFixture(t)
	registryPath := filepath.Join(fixtureRoot, "internal", "traceability", "ownership.v0.5.0.json")
	replaceOwnershipProductionDeclaration(t, registryPath, "section:9.2", "MissingSectionNineTwoImplementation")

	output, err := runTracecheck(t, fixtureRoot, "-section", "6.2")
	if err == nil || !strings.Contains(output, "projection digest ") {
		t.Fatalf("unrepinned missing-production tracecheck error = %v output = %q, want projection digest refusal", err, output)
	}
	repinOwnershipDigest(t, fixtureRoot, output)

	output, err = runTracecheck(t, fixtureRoot, "-section", "9.2")
	want := `section binding "section:9.2" production owner: declaration "MissingSectionNineTwoImplementation" is absent`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("missing-production tracecheck error = %v output = %q, want refusal containing %q and no success output", err, output, want)
	}

	output, err = runTracecheck(t, fixtureRoot, "-section", "6.2")
	if err != nil || !strings.Contains(output, "assigned_scopes=1") {
		t.Fatalf("unrelated assigned tracecheck error = %v output = %q, want green section:6.2 binding", err, output)
	}
}

// TestMainRejectsSyntacticallyValidNonexistentV050Section drives the production
// main -> run -> traceability.VerifyAssignedSections call chain. A plausible
// subsection must not inherit the top-level owner unless that exact identifier
// exists in the immutable v0.5.0 inventory. A real but still unowned section
// remains a separate refusal shape.
func TestMainRejectsSyntacticallyValidNonexistentV050Section(t *testing.T) {
	if testing.Short() {
		t.Skip("launches the production tracecheck entry point")
	}

	repositoryRoot := filepath.Join("..", "..", "..", "..")
	output, err := runTracecheck(t, repositoryRoot, "-section", "10.999")
	want := `assigned section "10.999" is not a real v0.5.0 section identifier`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("tracecheck -section 10.999 error = %v output = %q, want refusal %q and no success output", err, output, want)
	}

	output, err = runTracecheck(t, repositoryRoot, "-section", "10.5")
	want = `assigned section "10.5" binding "section:10.5" has no scoped implementation owner`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("tracecheck -section 10.5 error = %v output = %q, want refusal %q and no success output", err, output, want)
	}
}

func TestRunRejectsRegisteredContractWithoutImplementationOwner(t *testing.T) {
	t.Parallel()

	repositoryRoot := filepath.Join("..", "..", "..", "..")
	fixtureRoot := t.TempDir()
	if err := os.CopyFS(
		filepath.Join(fixtureRoot, "internal"),
		os.DirFS(filepath.Join(repositoryRoot, "internal")),
	); err != nil {
		t.Fatalf("copy repository fixture: %v", err)
	}

	registryPath := filepath.Join(fixtureRoot, "internal", "traceability", "ownership.v0.5.0.json")
	registry, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("read ownership registry fixture: %v", err)
	}
	const contract = "Session Directory Query [urn:ax:schema:session-directory-query]"
	owner := []byte(",\n        \"" + contract + "\"")
	if count := bytes.Count(registry, owner); count != 1 {
		t.Fatalf("ownership registry contains contract owner %q %d times, want exactly once", contract, count)
	}
	if err := os.WriteFile(registryPath, bytes.Replace(registry, owner, nil, 1), 0o644); err != nil {
		t.Fatalf("write narrowed ownership registry fixture: %v", err)
	}

	var output bytes.Buffer
	err = run([]string{"-root", fixtureRoot}, &output)
	want := "registered contract \"" + contract + "\" has no implementation owner"
	if err == nil || !errors.Is(err, traceability.ErrTraceability) || !strings.Contains(err.Error(), want) {
		t.Fatalf("run(missing contract owner) error = %v, want ErrTraceability containing %q", err, want)
	}
	if output.Len() != 0 {
		t.Fatalf("failed ownership run emitted success output %q", output.String())
	}
}

type failingWriter struct {
	err error
}

func copyFile(t *testing.T, source, destination string) {
	t.Helper()
	value, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read %q: %v", source, err)
	}
	if err := os.WriteFile(destination, value, 0o644); err != nil {
		t.Fatalf("write %q: %v", destination, err)
	}
}

func renameGoDeclaration(t *testing.T, filename, from, to string) {
	t.Helper()
	value, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read %q: %v", filename, err)
	}
	if count := bytes.Count(value, []byte(from)); count != 1 {
		t.Fatalf("%q contains declaration marker %q %d times, want exactly once", filename, from, count)
	}
	value = bytes.Replace(value, []byte(from), []byte(to), 1)
	if err := os.WriteFile(filename, value, 0o644); err != nil {
		t.Fatalf("write %q: %v", filename, err)
	}
}

func isolatedTracecheckFixture(t *testing.T) string {
	t.Helper()
	repositoryRoot := filepath.Join("..", "..", "..", "..")
	fixtureRoot := t.TempDir()
	copyFile(t, filepath.Join(repositoryRoot, "go.mod"), filepath.Join(fixtureRoot, "go.mod"))
	if err := os.CopyFS(filepath.Join(fixtureRoot, "internal"), os.DirFS(filepath.Join(repositoryRoot, "internal"))); err != nil {
		t.Fatalf("copy repository fixture: %v", err)
	}
	return fixtureRoot
}

func mutateOwnershipGroup(t *testing.T, filename, target string, mutate func(map[string]any)) {
	t.Helper()
	value, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read ownership registry: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(value, &document); err != nil {
		t.Fatalf("decode ownership registry: %v", err)
	}
	matched := 0
	for _, rawGroup := range document["ownership"].([]any) {
		group := rawGroup.(map[string]any)
		for _, key := range group["keys"].([]any) {
			if key == target {
				matched++
				mutate(group)
			}
		}
	}
	if matched != 1 {
		t.Fatalf("matched ownership key %q %d times, want exactly once", target, matched)
	}
	value, err = json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatalf("encode mutated ownership registry: %v", err)
	}
	if err := os.WriteFile(filename, value, 0o644); err != nil {
		t.Fatalf("write mutated ownership registry: %v", err)
	}
}

func removeOwnershipAcceptanceCases(t *testing.T, filename, target string) {
	t.Helper()
	mutateOwnershipGroup(t, filename, target, func(group map[string]any) {
		group["acceptance_cases"] = []any{}
	})
}

// replaceOwnershipProductionDeclaration narrows the mutant to exactly one
// thing: the binding names a Go declaration that does not exist. The gap
// sentence is rewritten to name the replacement too, because a gap has to name
// the declaration its binding is registered to, and leaving the old name there
// would make the coverage disclosure check red instead of the production-owner
// check this mutant exists to reach.
func replaceOwnershipProductionDeclaration(t *testing.T, filename, target, declaration string) {
	t.Helper()
	mutateOwnershipGroup(t, filename, target, func(group map[string]any) {
		production := group["production"].(map[string]any)
		previous, _ := production["declaration"].(string)
		production["declaration"] = declaration
		if gap, ok := group["gap"].(string); ok && previous != "" {
			group["gap"] = strings.ReplaceAll(gap, previous, declaration)
		}
	})
}

func removeOwnershipKey(t *testing.T, filename, target string) {
	t.Helper()
	value, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read ownership registry: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(value, &document); err != nil {
		t.Fatalf("decode ownership registry: %v", err)
	}
	removed := 0
	groups := []any{}
	for _, rawGroup := range document["ownership"].([]any) {
		group := rawGroup.(map[string]any)
		keys := group["keys"].([]any)
		filtered := []any{}
		for _, key := range keys {
			if key == target {
				removed++
				continue
			}
			filtered = append(filtered, key)
		}
		if len(filtered) == 0 {
			continue
		}
		group["keys"] = filtered
		groups = append(groups, group)
	}
	document["ownership"] = groups
	if removed != 1 {
		t.Fatalf("removed ownership key %q %d times, want exactly once", target, removed)
	}
	value, err = json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatalf("encode narrowed ownership registry: %v", err)
	}
	if err := os.WriteFile(filename, value, 0o644); err != nil {
		t.Fatalf("write narrowed ownership registry: %v", err)
	}
}

func runTracecheck(t *testing.T, root string, arguments ...string) (string, error) {
	t.Helper()
	commandArguments := append([]string{"run", "./internal/traceability/cmd/tracecheck", "-root", "."}, arguments...)
	command := exec.Command("go", commandArguments...)
	command.Dir = root
	command.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	output, err := command.CombinedOutput()
	return string(output), err
}

func repinOwnershipDigest(t *testing.T, root, refusal string) {
	t.Helper()
	const marker = "projection digest "
	start := strings.Index(refusal, marker)
	if start == -1 {
		t.Fatalf("digest refusal %q lacks %q", refusal, marker)
	}
	digest := strings.Fields(refusal[start+len(marker):])[0]
	filename := filepath.Join(root, "internal", "traceability", "traceability.go")
	value, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read traceability source: %v", err)
	}
	const assignment = `reviewedOwnershipCanonicalSHA256 = "`
	assignmentStart := bytes.Index(value, []byte(assignment))
	if assignmentStart == -1 {
		t.Fatalf("traceability source lacks reviewed digest assignment")
	}
	digestStart := assignmentStart + len(assignment)
	digestEnd := digestStart + 64
	value = append(append(append([]byte(nil), value[:digestStart]...), digest...), value[digestEnd:]...)
	if err := os.WriteFile(filename, value, 0o644); err != nil {
		t.Fatalf("write re-pinned traceability source: %v", err)
	}
}

func (writer failingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}
