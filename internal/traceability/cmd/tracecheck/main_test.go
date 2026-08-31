package main

import (
	"bytes"
	"errors"
	"io"
	"os"
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
	want := "traceability ok: contracts=60 normative_sections=17 acceptance_cases=15 fixtures=30 compatibility_contracts=55\n"
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

func (writer failingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}
