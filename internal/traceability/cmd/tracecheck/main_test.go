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
	want := "traceability ok: contracts=60 normative_sections=36 acceptance_cases=16 fixtures=30 compatibility_contracts=55 assigned_scopes=0\n"
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
	err = run([]string{"-root", repositoryRoot, "-section", "9.2", "-section", "7.9"}, &output)
	if err != nil {
		t.Fatalf("run(assigned sections) error = %v", err)
	}
	want = "traceability ok: contracts=60 normative_sections=36 acceptance_cases=16 fixtures=30 compatibility_contracts=55 assigned_scopes=2\n"
	if output.String() != want {
		t.Fatalf("run(assigned sections) output = %q, want %q", output.String(), want)
	}
}

func TestRunDefaultStaysGreenForPinnedButUnownedSections(t *testing.T) {
	t.Parallel()

	repositoryRoot := filepath.Join("..", "..", "..", "..")
	var output bytes.Buffer
	if err := run([]string{"-root", repositoryRoot}, &output); err != nil {
		t.Fatalf("default run() error = %v, want global inventory verification to ignore unassigned Section 10.1", err)
	}
	if !strings.Contains(output.String(), "assigned_scopes=0") {
		t.Fatalf("default run() output = %q, want assigned_scopes=0", output.String())
	}

	output.Reset()
	err := run([]string{"-root", repositoryRoot, "-section", "10.1"}, &output)
	if err == nil || !strings.Contains(err.Error(), `assigned section "10.1" binding "section:10.1" has no scoped implementation owner`) {
		t.Fatalf("assigned run(10.1) error = %v output = %q, want scoped-owner refusal", err, output.String())
	}
}

// TestMainRejectsOneNarrowedAssignedSectionBinding drives the production
// main -> run -> traceability.VerifyAssignedSections call chain in an isolated
// module. It re-pins the intentionally narrowed registry so the section:9.2
// refusal, rather than the existing digest gate, is what makes the test red.
// The unrelated section:7.9 binding must remain executable and green.
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

	output, err := runTracecheck(t, fixtureRoot, "-section", "7.9")
	if err == nil || !strings.Contains(output, "projection digest ") {
		t.Fatalf("unrepinned narrowed tracecheck error = %v output = %q, want projection digest refusal", err, output)
	}
	repinOwnershipDigest(t, fixtureRoot, output)

	output, err = runTracecheck(t, fixtureRoot, "-section", "9.2")
	want := `assigned section "9.2" binding "section:9.2" has no scoped implementation owner`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("narrowed assigned tracecheck error = %v output = %q, want refusal %q and no success output", err, output, want)
	}

	output, err = runTracecheck(t, fixtureRoot, "-section", "7.9")
	if err != nil || !strings.Contains(output, "assigned_scopes=1") {
		t.Fatalf("unrelated assigned tracecheck error = %v output = %q, want green section:7.9 binding", err, output)
	}
}

// TestMainRejectsDetachedScopeSpecificAcceptanceCase drives the production
// main -> run -> traceability.VerifyAssignedSections call chain after removing
// only Section 9.2's executable acceptance link. Section 7.9 must stay green.
func TestMainRejectsDetachedScopeSpecificAcceptanceCase(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an isolated tracecheck binary")
	}

	fixtureRoot := isolatedTracecheckFixture(t)
	registryPath := filepath.Join(fixtureRoot, "internal", "traceability", "ownership.v0.5.0.json")
	removeOwnershipAcceptanceCases(t, registryPath, "section:9.2")

	output, err := runTracecheck(t, fixtureRoot, "-section", "7.9")
	if err == nil || !strings.Contains(output, "projection digest ") {
		t.Fatalf("unrepinned detached-case tracecheck error = %v output = %q, want projection digest refusal", err, output)
	}
	repinOwnershipDigest(t, fixtureRoot, output)

	output, err = runTracecheck(t, fixtureRoot, "-section", "9.2")
	want := `section binding "section:9.2" has no scope-specific acceptance owner`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("detached-case tracecheck error = %v output = %q, want refusal %q and no success output", err, output, want)
	}

	output, err = runTracecheck(t, fixtureRoot, "-section", "7.9")
	if err != nil || !strings.Contains(output, "assigned_scopes=1") {
		t.Fatalf("unrelated assigned tracecheck error = %v output = %q, want green section:7.9 binding", err, output)
	}
}

// TestMainRejectsMissingScopeSpecificProductionDeclaration drives the
// production entry point after detaching only Section 9.2 from its concrete Go
// declaration. Section 7.9 must remain independently executable and green.
func TestMainRejectsMissingScopeSpecificProductionDeclaration(t *testing.T) {
	if testing.Short() {
		t.Skip("builds an isolated tracecheck binary")
	}

	fixtureRoot := isolatedTracecheckFixture(t)
	registryPath := filepath.Join(fixtureRoot, "internal", "traceability", "ownership.v0.5.0.json")
	replaceOwnershipProductionDeclaration(t, registryPath, "section:9.2", "MissingSectionNineTwoImplementation")

	output, err := runTracecheck(t, fixtureRoot, "-section", "7.9")
	if err == nil || !strings.Contains(output, "projection digest ") {
		t.Fatalf("unrepinned missing-production tracecheck error = %v output = %q, want projection digest refusal", err, output)
	}
	repinOwnershipDigest(t, fixtureRoot, output)

	output, err = runTracecheck(t, fixtureRoot, "-section", "9.2")
	want := `section binding "section:9.2" production owner: declaration "MissingSectionNineTwoImplementation" is absent`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("missing-production tracecheck error = %v output = %q, want refusal containing %q and no success output", err, output, want)
	}

	output, err = runTracecheck(t, fixtureRoot, "-section", "7.9")
	if err != nil || !strings.Contains(output, "assigned_scopes=1") {
		t.Fatalf("unrelated assigned tracecheck error = %v output = %q, want green section:7.9 binding", err, output)
	}
}

// TestMainRejectsSyntacticallyValidNonexistentV050Section drives the production
// main -> run -> traceability.VerifyAssignedSections call chain. A plausible
// subsection must not inherit the top-level owner unless that exact identifier
// exists in the immutable v0.5.0 inventory.
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

	output, err = runTracecheck(t, repositoryRoot, "-section", "10.1")
	want = `assigned section "10.1" binding "section:10.1" has no scoped implementation owner`
	if err == nil || !strings.Contains(output, want) || strings.Contains(output, "traceability ok:") {
		t.Fatalf("tracecheck -section 10.1 error = %v output = %q, want refusal %q and no success output", err, output, want)
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

func replaceOwnershipProductionDeclaration(t *testing.T, filename, target, declaration string) {
	t.Helper()
	mutateOwnershipGroup(t, filename, target, func(group map[string]any) {
		group["production"].(map[string]any)["declaration"] = declaration
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
	for _, rawGroup := range document["ownership"].([]any) {
		group := rawGroup.(map[string]any)
		keys := group["keys"].([]any)
		filtered := keys[:0]
		for _, key := range keys {
			if key == target {
				removed++
				continue
			}
			filtered = append(filtered, key)
		}
		group["keys"] = filtered
	}
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
