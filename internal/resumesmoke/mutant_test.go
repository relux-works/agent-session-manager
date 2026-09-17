package resumesmoke

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// This file ships the refusal-mutant harness: every gate in this
// package is weakened to admit exactly one member of the class it
// must reject, and a named test must fail on the weakened tree. A
// delete-only mutant would prove only that the gate exists; each
// mutant below keeps the gate and narrows it, so survival means the
// named test does not actually cover the gate.
//
// The harness never touches the checkout: each mutant is applied to
// a private copy of the module under a temp directory, and one
// focused go test runs there. Copying keeps parallel package runs
// hermetic — a tree-walking census in another package can never
// observe a half-patched file. It skips under -short so short runs
// stay fast; every other run executes it.

// smokeMutant is one narrowing mutation: the production file, the
// exact text to weaken (which must occur exactly once), its
// narrowed replacement, the focused test mask that must fail, and
// whether the mutant must be killed or must survive. Exactly one
// mutant is the harmless SURVIVED control, which proves the harness
// reports survival instead of failing unconditionally.
type smokeMutant struct {
	name    string
	file    string
	find    string
	replace string
	mask    string
	survive bool
}

// smokeMutants is the shipped set: one narrowing mutant per
// refusal, plus the harmless control.
var smokeMutants = []smokeMutant{
	{
		name:    "resume-gate-admits-conditional",
		file:    "smoke.go",
		find:    "if run.cell != CellAvailable {",
		replace: "if run.cell != CellAvailable && run.cell != CellConditional {",
		mask:    "TestSmokeRowVerdicts/gemini-0.54.4-wsl2-amd64",
	},
	{
		name:    "probe-admits-version-drift",
		file:    "smoke.go",
		find:    "if probed != run.params.Tuple {",
		replace: "if probed.ProviderID != run.params.Tuple.ProviderID || probed.Platform != run.params.Tuple.Platform || probed.Architecture != run.params.Tuple.Architecture {",
		mask:    "TestSmokeProbeMismatchFails",
	},
	{
		name:    "identify-admits-strong",
		file:    "smoke.go",
		find:    `if outcome.Confidence != "exact" {`,
		replace: `if outcome.Confidence != "exact" && outcome.Confidence != "strong" {`,
		mask:    "TestSmokeStrongConfidenceFails",
	},
	{
		name:    "bind-admits-antigravity",
		file:    "smoke.go",
		find:    "if err := provhost.VerifyIdentityDiscovery(identity, run.params.Discovery, discovery); err != nil {",
		replace: `if err := provhost.VerifyIdentityDiscovery(identity, run.params.Discovery, discovery); err != nil && run.params.Tuple.ProviderID != "antigravity" {`,
		mask:    "TestSmokeAntigravityUnresolvedRealmFails",
	},
	{
		name:    "digest-scoped-to-pass",
		file:    "record.go",
		find:    `if "sha256:"+hex.EncodeToString(sum[:]) != record.RecordID {`,
		replace: `if "sha256:"+hex.EncodeToString(sum[:]) != record.RecordID && record.Verdict == VerdictPass {`,
		mask:    "TestSmokeTamperedRecordRefuses",
	},
	{
		name:    "consistency-admits-fail",
		file:    "record.go",
		find:    "if want := deriveVerdict(record.ResumeCell, record.Checks); record.Verdict != want {",
		replace: "if want := deriveVerdict(record.ResumeCell, record.Checks); record.Verdict != want && record.Verdict != VerdictFail {",
		mask:    "TestSmokeInconsistentFailRefuses",
	},
	{
		name:    "disagreement-admits-same-length",
		file:    "store.go",
		find:    "if !bytes.Equal(existing, record) {",
		replace: "if len(existing) != len(record) {",
		mask:    "TestStoreDisagreementQuarantines",
	},
	{
		name:    "quiescence-admits-unsafe-codex",
		file:    "smoke.go",
		find:    "if !safe {",
		replace: `if !safe && run.params.Tuple.ProviderID != "codex" {`,
		mask:    "TestSmokeUnsafeQuiescenceFails",
	},
	{
		name:    "mapping-admits-yolo",
		file:    "smoke.go",
		find:    "if _, err := provhost.ResolveMapping(run.params.Tuple.ProviderID, run.params.Profile, run.params.Tuple); err != nil {",
		replace: `if _, err := provhost.ResolveMapping(run.params.Tuple.ProviderID, run.params.Profile, run.params.Tuple); err != nil && run.params.Profile != "yolo" {`,
		mask:    "TestSmokePiUnmappedVersionFails",
	},
	{
		name:    "matrix-admits-muse-offpin",
		file:    "matrix.go",
		find:    `return CellUnknown, "section-8.4:muse/macos-arm64+appendix-B"`,
		replace: `return CellConditional, "section-8.4:muse/macos-arm64+appendix-B"`,
		mask:    "TestResumeMatrixCoversEverySpecRow",
	},
	{
		name:    "harmless-comment-control",
		file:    "smoke.go",
		find:    "// Check names in execution order.",
		replace: "// Check names in execution order. Mutant-harness control: no behavior change.",
		mask:    "TestSmokeRowVerdicts/codex-0.147.0-macos-amd64",
		survive: true,
	},
}

// TestSmokeMutantsAreKilled applies every shipped mutant to a
// private module copy and requires the kill verdict each names:
// narrowing mutants must fail their focused test, and the harmless
// control must pass its.
func TestSmokeMutantsAreKilled(t *testing.T) {
	if testing.Short() {
		t.Skip("mutant harness needs full runs; skipped under -short")
	}
	root := smokeModuleRoot(t)
	killed, survived := 0, 0
	for _, mutant := range smokeMutants {
		mutant := mutant
		t.Run(mutant.name, func(t *testing.T) {
			workdir := copyModuleForMutant(t, root, mutant)
			command := exec.Command("go", "test", "./internal/resumesmoke/", "-run", mutant.mask, "-count=1")
			command.Dir = workdir
			output, err := command.CombinedOutput()
			failed := err != nil
			t.Logf("mutant %s mask %s failed=%v:\n%s", mutant.name, mutant.mask, failed, output)
			switch {
			case mutant.survive && failed:
				t.Fatalf("SURVIVED control failed; the harness or the mask is broken:\n%s", output)
			case mutant.survive && !failed:
				survived++
			case !mutant.survive && !failed:
				t.Fatalf("SURVIVED: narrowing mutant %q passed %s; the gate is uncovered", mutant.name, mutant.mask)
			case !mutant.survive && failed && !strings.Contains(string(output), "FAIL"):
				t.Fatalf("mutant %q errored without a test failure; the mask may be broken:\n%s", mutant.name, output)
			default:
				killed++
			}
		})
	}
	t.Logf("mutants killed=%d survived-controls=%d", killed, survived)
	if killed != len(smokeMutants)-1 || survived != 1 {
		t.Fatalf("mutant census killed=%d survived=%d, want %d and 1", killed, survived, len(smokeMutants)-1)
	}
}

// smokeModuleRoot returns the repository root two directories
// above this package: the tree the harness copies mutants from.
func smokeModuleRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	return filepath.Join(working, "..", "..")
}

// copyModuleForMutant copies the module sources a mutant run needs
// — go.mod, go.sum, and the internal tree — into a fresh temp
// directory, applies the mutant there, and returns the copy. The
// checkout is only read, never written.
func copyModuleForMutant(t *testing.T, root string, mutant smokeMutant) string {
	t.Helper()
	workdir := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, name), data, 0o600); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	copyDir(t, filepath.Join(root, "internal"), filepath.Join(workdir, "internal"))
	path := filepath.Join(workdir, "internal", "resumesmoke", mutant.file)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", mutant.file, err)
	}
	if bytes.Count(original, []byte(mutant.find)) != 1 {
		t.Fatalf("mutant find occurs %d times in %s, want exactly once", bytes.Count(original, []byte(mutant.find)), mutant.file)
	}
	mutated := bytes.Replace(original, []byte(mutant.find), []byte(mutant.replace), 1)
	if err := os.WriteFile(path, mutated, 0o600); err != nil {
		t.Fatalf("apply mutant: %v", err)
	}
	return workdir
}

// copyDir copies one directory tree to a fresh destination,
// preserving file modes. Symlinks refuse: the module tree the
// harness copies carries none, and a link would escape the copy.
func copyDir(t *testing.T, source, destination string) {
	t.Helper()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatalf("read %s: %v", source, err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatalf("stage %s: %v", destination, err)
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			t.Fatalf("refusing to copy symlink %s", filepath.Join(source, entry.Name()))
		}
		from := filepath.Join(source, entry.Name())
		to := filepath.Join(destination, entry.Name())
		if entry.IsDir() {
			copyDir(t, from, to)
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatalf("read %s: %v", from, err)
		}
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("stat %s: %v", from, err)
		}
		if err := os.WriteFile(to, data, info.Mode()); err != nil {
			t.Fatalf("stage %s: %v", to, err)
		}
	}
}
