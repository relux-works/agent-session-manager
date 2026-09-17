package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

func TestRunGeneratesCommittedCatalogAndSupportsIdenticalRetry(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "catalog_gen.go")
	arguments := []string{
		"-metadata", filepath.Join("..", "..", "catalog.v0.7.0.json"),
		"-contracts", filepath.Join("..", "..", "..", "specpin", "v0.7.0.lock.json"),
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
		"-metadata", filepath.Join("..", "..", "catalog.v0.7.0.json"),
		"-contracts", filepath.Join("..", "..", "..", "specpin", "v0.7.0.lock.json"),
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

	metadata := filepath.Join("..", "..", "catalog.v0.7.0.json")
	contracts := filepath.Join("..", "..", "..", "specpin", "v0.7.0.lock.json")
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

// TestRunAdoptedCheckIsGreen drives the release-agnostic gate form through
// the production run entry: -adopted resolves the adopted inputs and the
// committed output verifies without a rewrite.
func TestRunAdoptedCheckIsGreen(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..")
	output := filepath.Join(root, "internal", "catalog", "catalog_gen.go")
	if err := run([]string{"-adopted", "-root", root, "-output", output, "-check"}); err != nil {
		t.Fatalf("adopted check run() error = %v", err)
	}
}

// TestRunAdoptedIsByteIdenticalToExplicit drives both the -adopted form and
// the explicit v0.7.0 invocation through the production run entry and pins
// the three artifacts — adopted output, explicit output, committed output —
// byte-identical.
func TestRunAdoptedIsByteIdenticalToExplicit(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..")
	adoptedOutput := filepath.Join(t.TempDir(), "catalog_adopted.go")
	if err := run([]string{"-adopted", "-root", root, "-output", adoptedOutput}); err != nil {
		t.Fatalf("adopted run() error = %v", err)
	}
	explicitOutput := filepath.Join(t.TempDir(), "catalog_explicit.go")
	explicit := []string{
		"-metadata", filepath.Join(root, "internal", "catalog", "catalog.v0.7.0.json"),
		"-contracts", filepath.Join(root, "internal", "specpin", "v0.7.0.lock.json"),
		"-output", explicitOutput,
	}
	if err := run(explicit); err != nil {
		t.Fatalf("explicit run() error = %v", err)
	}
	adopted, err := os.ReadFile(adoptedOutput)
	if err != nil {
		t.Fatalf("read adopted output: %v", err)
	}
	explicitBytes, err := os.ReadFile(explicitOutput)
	if err != nil {
		t.Fatalf("read explicit output: %v", err)
	}
	if !bytes.Equal(adopted, explicitBytes) {
		t.Fatal("adopted output differs from explicit v0.7.0 output")
	}
	committed, err := os.ReadFile(filepath.Join(root, "internal", "catalog", "catalog_gen.go"))
	if err != nil {
		t.Fatalf("read committed catalog: %v", err)
	}
	if !bytes.Equal(adopted, committed) {
		t.Fatal("adopted output differs from committed generated catalog")
	}
}

// TestRunAdoptedRefusesMixedAndMissingInputs drives every adopted-mode input
// refusal through the production run entry: explicit flags mixed with
// -adopted, a missing -output, and a missing derived metadata or lock file,
// each with a distinct message naming the derived path.
func TestRunAdoptedRefusesMixedAndMissingInputs(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..")
	metadata := filepath.Join(root, "internal", "catalog", "catalog.v0.7.0.json")
	contracts := filepath.Join(root, "internal", "specpin", "v0.7.0.lock.json")
	metadataBytes, err := os.ReadFile(metadata)
	if err != nil {
		t.Fatalf("read adopted metadata fixture: %v", err)
	}
	contractsBytes, err := os.ReadFile(contracts)
	if err != nil {
		t.Fatalf("read adopted lock fixture: %v", err)
	}
	partialMissingMetadata := writeAdoptedRoot(t, nil, contractsBytes)
	partialMissingLock := writeAdoptedRoot(t, metadataBytes, nil)
	tests := []struct {
		name      string
		arguments func(output string) []string
		contains  []string
	}{
		{name: "mixed metadata", arguments: func(output string) []string {
			return []string{"-adopted", "-metadata", metadata, "-output", output}
		}, contains: []string{"-adopted cannot be combined"}},
		{name: "mixed contracts", arguments: func(output string) []string {
			return []string{"-adopted", "-contracts", contracts, "-output", output}
		}, contains: []string{"-adopted cannot be combined"}},
		{name: "missing output", arguments: func(string) []string {
			return []string{"-adopted", "-root", root}
		}, contains: []string{"-output is required with -adopted"}},
		{name: "missing metadata", arguments: func(output string) []string {
			return []string{"-adopted", "-root", partialMissingMetadata, "-output", output}
		}, contains: []string{"read adopted metadata", "catalog." + string(catalog.Adopted) + ".json"}},
		{name: "missing lock", arguments: func(output string) []string {
			return []string{"-adopted", "-root", partialMissingLock, "-output", output}
		}, contains: []string{"read adopted contract lock", string(catalog.Adopted) + ".lock.json"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "catalog_gen.go")
			err := run(test.arguments(output))
			if err == nil {
				t.Fatalf("run() error = nil, want containing %q", test.contains)
			}
			for _, want := range test.contains {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("run() error = %v, want containing %q", err, want)
				}
			}
		})
	}
}

// TestRunAdoptedRefusesLockReleaseMismatch drives a lock filed under the
// adopted name whose source release is v0.5.0 through the production run
// entry: it is refused before generation with the adopted-release message,
// not the generic generation failure.
func TestRunAdoptedRefusesLockReleaseMismatch(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..")
	metadataBytes, err := os.ReadFile(filepath.Join(root, "internal", "catalog", "catalog.v0.7.0.json"))
	if err != nil {
		t.Fatalf("read adopted metadata fixture: %v", err)
	}
	staleLock, err := os.ReadFile(filepath.Join(root, "internal", "specpin", "v0.5.0.lock.json"))
	if err != nil {
		t.Fatalf("read stale lock fixture: %v", err)
	}
	spoofed := writeAdoptedRoot(t, metadataBytes, staleLock)
	output := filepath.Join(t.TempDir(), "catalog_gen.go")
	err = run([]string{"-adopted", "-root", spoofed, "-output", output})
	if err == nil || !strings.Contains(err.Error(), "differs from adopted release") {
		t.Fatalf("run() error = %v, want adopted-release mismatch refusal", err)
	}
	if strings.Contains(err.Error(), "generate catalog") {
		t.Fatalf("run() error = %v, want pre-generation refusal, not a generation failure", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatal("refused adopted run wrote output")
	}
}

// TestRunAdoptedCheckRefusesStaleOutput drives a same-length stale output
// through the production run entry under -adopted -check: it is refused
// without being rewritten.
func TestRunAdoptedCheckRefusesStaleOutput(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..")
	fresh, err := os.ReadFile(filepath.Join(root, "internal", "catalog", "catalog_gen.go"))
	if err != nil {
		t.Fatalf("read committed catalog: %v", err)
	}
	stale := bytes.Clone(fresh)
	stale[len(stale)-2] ^= 0xff
	output := filepath.Join(t.TempDir(), "catalog_gen.go")
	if err := os.WriteFile(output, stale, 0o600); err != nil {
		t.Fatalf("write stale output: %v", err)
	}
	err = run([]string{"-adopted", "-root", root, "-output", output, "-check"})
	if err == nil || !strings.Contains(err.Error(), "generated catalog is stale") {
		t.Fatalf("adopted check run() error = %v, want stale-output refusal", err)
	}
	got, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatalf("read stale output after refusal: %v", readErr)
	}
	if !bytes.Equal(got, stale) {
		t.Fatal("adopted check run rewrote stale output")
	}
}

// writeAdoptedRoot stages a repository-root-shaped fixture carrying the
// adopted metadata and lock inputs under their derived names. A nil input
// is left absent to exercise the missing-input refusals.
func writeAdoptedRoot(t *testing.T, metadata, lock []byte) string {
	t.Helper()

	root := t.TempDir()
	catalogDir := filepath.Join(root, "internal", "catalog")
	specpinDir := filepath.Join(root, "internal", "specpin")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatalf("create catalog fixture directory: %v", err)
	}
	if err := os.MkdirAll(specpinDir, 0o755); err != nil {
		t.Fatalf("create specpin fixture directory: %v", err)
	}
	if metadata != nil {
		name := "catalog." + string(catalog.Adopted) + ".json"
		if err := os.WriteFile(filepath.Join(catalogDir, name), metadata, 0o600); err != nil {
			t.Fatalf("write metadata fixture: %v", err)
		}
	}
	if lock != nil {
		name := string(catalog.Adopted) + ".lock.json"
		if err := os.WriteFile(filepath.Join(specpinDir, name), lock, 0o600); err != nil {
			t.Fatalf("write lock fixture: %v", err)
		}
	}
	return root
}

// Drive the command taken from the CR configuration through the real CLI entry.
// A recognizable generator token alone cannot prove its selected authority.
// On this tree -adopted resolves to the v0.7.0 metadata/lock pair and passes
// -check; a v0.6.0 lock filed under the adopted name is refused.
func TestConfiguredCRCatalogGateConsumesAdoptedAuthority(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, "task-board.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Spawn struct {
			WorktreeIsolation struct {
				Validation struct {
					Commands []string `json:"commands"`
				} `json:"validation"`
			} `json:"worktree_isolation"`
		} `json:"spawn"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	var commands []string
	for _, command := range config.Spawn.WorktreeIsolation.Validation.Commands {
		if strings.Contains(command, "./internal/catalog/cmd/cataloggen") {
			commands = append(commands, command)
		}
	}
	if len(commands) != 1 {
		t.Fatalf("configured catalog gates = %d, want exactly 1", len(commands))
	}
	fields := strings.Fields(commands[0])
	if len(fields) != 7 ||
		strings.Join(fields[:3], " ") != "go run ./internal/catalog/cmd/cataloggen" ||
		fields[3] != "-adopted" || fields[4] != "-output" || fields[6] != "-check" ||
		fields[5] != "internal/catalog/catalog_gen.go" {
		t.Fatalf("unexpected gate command shape: %q", commands[0])
	}
	// The configured gate runs from the repository root; the test process
	// runs from the package directory, so root it explicitly. -output is a
	// value flag, not a derived input, and keeps its configured value.
	args := []string{"-adopted", "-root", root, "-output", filepath.Join(root, fields[5]), "-check"}
	if err := run(args); err != nil {
		t.Fatalf("configured cataloggen run: %v", err)
	}
	metadataBytes, err := os.ReadFile(filepath.Join(root, "internal", "catalog", "catalog.v0.7.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	staleLock, err := os.ReadFile(filepath.Join(root, "internal", "specpin", "v0.6.0.lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	spoofed := writeAdoptedRoot(t, metadataBytes, staleLock)
	stale := []string{"-adopted", "-root", spoofed, "-output", filepath.Join(t.TempDir(), "catalog_gen.go")}
	if err := run(stale); err == nil || !strings.Contains(err.Error(), "differs from adopted release") {
		t.Fatalf("stale configured cataloggen run = %v, want adopted-release refusal", err)
	}
}

func TestGenerateDirectiveConsumesAdoptedAuthority(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "catalog.go"))
	if err != nil {
		t.Fatal(err)
	}
	var directives []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "//go:generate ") {
			directives = append(directives, strings.TrimPrefix(line, "//go:generate "))
		}
	}
	if len(directives) != 1 {
		t.Fatalf("active directives = %d, want 1", len(directives))
	}
	fields := strings.Fields(directives[0])
	if len(fields) != 9 || strings.Join(fields[:3], " ") != "go run ./cmd/cataloggen" {
		t.Fatalf("unexpected directive: %q", directives[0])
	}
	args := append([]string(nil), fields[3:]...)
	for i := 1; i < 6; i += 2 {
		args[i] = filepath.Join("..", "..", args[i])
	}
	// Check mode drives the same Generate entry without writing the source tree.
	if err := run(append(args, "-check")); err != nil {
		t.Fatalf("directive cataloggen run: %v", err)
	}
}
