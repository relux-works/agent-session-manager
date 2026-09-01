package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunGeneratesCommittedCatalogAndSupportsIdenticalRetry(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "catalog_gen.go")
	arguments := []string{
		"-metadata", filepath.Join("..", "..", "catalog.v0.5.0.json"),
		"-contracts", filepath.Join("..", "..", "..", "specpin", "v0.5.0.lock.json"),
		"-output", output,
	}
	if err := run(arguments); err != nil {
		t.Fatalf("first run() error = %v", err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read run output: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "catalog_gen.go"))
	if err != nil {
		t.Fatalf("read committed catalog: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("run output differs from committed generated catalog")
	}
	if err := run(append(append([]string(nil), arguments...), "-check")); err != nil {
		t.Fatalf("check run() error = %v", err)
	}

	stableTime := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(output, stableTime, stableTime); err != nil {
		t.Fatalf("set stable run output time: %v", err)
	}
	if err := run(arguments); err != nil {
		t.Fatalf("identical retry run() error = %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat run output: %v", err)
	}
	if !info.ModTime().Equal(stableTime) {
		t.Fatalf("identical run retry replaced output: modtime = %v, want %v", info.ModTime(), stableTime)
	}
}

func TestRunCheckRefusesStaleOutputWithoutRewritingIt(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "catalog_gen.go")
	stale := []byte("package catalog\n\n// stale\n")
	if err := os.WriteFile(output, stale, 0o600); err != nil {
		t.Fatalf("write stale output: %v", err)
	}
	arguments := []string{
		"-metadata", filepath.Join("..", "..", "catalog.v0.5.0.json"),
		"-contracts", filepath.Join("..", "..", "..", "specpin", "v0.5.0.lock.json"),
		"-output", output,
		"-check",
	}
	err := run(arguments)
	if err == nil || !strings.Contains(err.Error(), "generated catalog is stale") {
		t.Fatalf("check run() error = %v, want stale-output refusal", err)
	}
	got, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatalf("read stale output after refusal: %v", readErr)
	}
	if !bytes.Equal(got, stale) {
		t.Fatal("check run rewrote stale output")
	}
}

func TestRunRefusesInvalidArgumentsInputsAndOutput(t *testing.T) {
	t.Parallel()

	metadata := filepath.Join("..", "..", "catalog.v0.5.0.json")
	contracts := filepath.Join("..", "..", "..", "specpin", "v0.5.0.lock.json")
	tests := []struct {
		name      string
		arguments func(string) []string
		contains  string
	}{
		{name: "missing flags", arguments: func(string) []string { return nil }, contains: "required"},
		{name: "unknown flag", arguments: func(string) []string { return []string{"-unknown"} }, contains: "flag provided but not defined"},
		{name: "missing metadata", arguments: func(output string) []string {
			return []string{"-metadata", filepath.Join(t.TempDir(), "absent.json"), "-contracts", contracts, "-output", output}
		}, contains: "read metadata"},
		{name: "missing contracts", arguments: func(output string) []string {
			return []string{"-metadata", metadata, "-contracts", filepath.Join(t.TempDir(), "absent.json"), "-output", output}
		}, contains: "read contract lock"},
		{name: "invalid metadata", arguments: func(output string) []string {
			invalid := filepath.Join(t.TempDir(), "invalid.json")
			if err := os.WriteFile(invalid, []byte("{}"), 0o600); err != nil {
				t.Fatalf("write invalid metadata: %v", err)
			}
			return []string{"-metadata", invalid, "-contracts", contracts, "-output", output}
		}, contains: "generate catalog"},
		{name: "unwritable output", arguments: func(output string) []string {
			if err := os.Mkdir(output, 0o755); err != nil {
				t.Fatalf("create output directory: %v", err)
			}
			return []string{"-metadata", metadata, "-contracts", contracts, "-output", output}
		}, contains: "write catalog"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "catalog_gen.go")
			err := run(test.arguments(output))
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("run() error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func TestWriteIfChangedPublishesAtomicallyAndSkipsIdenticalRetry(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "catalog_gen.go")
	want := []byte("package catalog\n")
	if err := writeIfChanged(path, want); err != nil {
		t.Fatalf("first writeIfChanged() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated output: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("generated output = %q, want %q", got, want)
	}

	stableTime := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(path, stableTime, stableTime); err != nil {
		t.Fatalf("set stable output time: %v", err)
	}
	if err := writeIfChanged(path, want); err != nil {
		t.Fatalf("identical retry writeIfChanged() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat generated output: %v", err)
	}
	if !info.ModTime().Equal(stableTime) {
		t.Fatalf("identical retry replaced output: modtime = %v, want %v", info.ModTime(), stableTime)
	}
}

func TestWriteIfChangedRefusesUnreadableDestinationWithoutReplacement(t *testing.T) {
	t.Parallel()

	destination := filepath.Join(t.TempDir(), "catalog_gen.go")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatalf("create destination directory: %v", err)
	}
	if err := writeIfChanged(destination, []byte("replacement")); err == nil {
		t.Fatal("writeIfChanged() error = nil, want unreadable-destination refusal")
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat destination after refusal: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("failed write replaced the original destination")
	}
}
