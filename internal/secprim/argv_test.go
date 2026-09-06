package secprim

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckArgvAdmits(t *testing.T) {
	t.Parallel()
	for _, argv := range [][]string{
		{"/plugins/ax-provider-pi"},
		{"ax", "pane", "0193e2c0-9c9c-7b1e-9c9c-7b1e9c9c7b1e"},
		{`C:\tools\ax.exe`, "--non-interactive"},
		{"/bin/echo", "hello world"},
	} {
		if err := CheckArgv(argv); err != nil {
			t.Errorf("CheckArgv(%q) = %v, want admission", argv, err)
		}
	}
}

func TestCheckArgvRefuses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		argv []string
		want string
	}{
		{"empty vector", nil, "secprim unsafe argv: argv empty: no executable"},
		{"empty executable", []string{""}, "secprim unsafe argv: argv executable empty: argv[0]"},
		{"empty element", []string{"ax", ""}, "secprim unsafe argv: argv element empty: argv[1]"},
		{"nul executable", []string{"a\x00b"}, "secprim unsafe argv: argv NUL: argv[0]"},
		{"nul argument", []string{"ax", "a\x00b"}, "secprim unsafe argv: argv NUL: argv[1]"},
		{"invalid utf8", []string{"ax", "a\xffb"}, "secprim unsafe argv: argv encoding: argv[1]"},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			t.Parallel()
			requireRefusal(t, CheckArgv(kase.argv), ErrUnsafeArgv, kase.want)
		})
	}
}

func TestCommandCopiesOnBothSides(t *testing.T) {
	t.Parallel()
	argv := []string{"ax", "pane", "session"}
	command, err := NewCommand(argv)
	if err != nil {
		t.Fatalf("NewCommand = %v", err)
	}
	argv[1] = "MUTATED"
	if got := command.Argv()[1]; got != "pane" {
		t.Fatalf("constructor aliases the caller slice: %q", got)
	}
	out := command.Argv()
	out[0] = "MUTATED"
	if command.Executable() != "ax" {
		t.Fatalf("accessor aliases the admitted vector: %q", command.Executable())
	}
	if _, err := NewCommand(nil); err == nil {
		t.Fatal("NewCommand(nil) admitted an empty vector")
	}
}

// shellTokens are the shell-construction shapes this package must never
// contain outside a comment. The list is pinned by the instrument test
// below, which proves each token fires in code position and stays silent
// in comment position.
var shellTokens = []string{
	"sh -c", "/bin/sh", "cmd /c", "cmd.exe", "powershell",
	"ShellExecute", "system(", "popen(", "/bin/bash",
}

// codeWithoutComments strips line comments so tokens named to explain the
// gate do not trip it.
func codeWithoutComments(contents string) string {
	var code []string
	for _, line := range strings.Split(contents, "\n") {
		if index := strings.Index(line, "//"); index >= 0 {
			line = line[:index]
		}
		code = append(code, line)
	}
	return strings.Join(code, "\n")
}

// TestShellTokenInstrumentIsProven proves the token scanner against one
// synthetic file per token: each token fires in code position (so a live
// token is never dead ceremony) and stays silent in comment position (so
// the gate's own documentation does not trip it).
func TestShellTokenInstrumentIsProven(t *testing.T) {
	t.Parallel()
	for _, token := range shellTokens {
		code := "package x\nvar s = \"" + token + "\"\n"
		if found := shellTokensIn(codeWithoutComments(code)); len(found) == 0 {
			t.Errorf("token %q does not fire in code position; the scanner is blind to it", token)
		}
		documented := "package x\n// forbidden: " + token + " must never appear\n"
		if found := shellTokensIn(codeWithoutComments(documented)); len(found) != 0 {
			t.Errorf("token %q fires in comment position; the gate cannot document itself", token)
		}
	}
}

func shellTokensIn(code string) []string {
	var found []string
	for _, token := range shellTokens {
		if strings.Contains(code, token) {
			found = append(found, token)
		}
	}
	return found
}

// TestPackageHasNoShellString scans production source for shell
// construction tokens. It is the static half of the no-shell gate; the
// behavioral half below catches the token-preserving mutant that builds
// the shell name without a literal.
func TestPackageHasNoShellString(t *testing.T) {
	t.Parallel()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for _, found := range shellTokensIn(codeWithoutComments(string(contents))) {
			t.Errorf("%s carries shell token %q outside a comment", name, found)
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no production sources; the check is blind")
	}
}

// TestPackageLaunchesNoProcess is the behavioral half of the no-shell
// gate: this package validates argv vectors but never starts a process,
// so importing os/exec here is a defect no token scan can excuse. The
// token-preserving mutant adds an os/exec import with the shell name
// assembled from variables (no literal token to scan for); this test
// fails on the import, not on the token.
func TestPackageLaunchesNoProcess(t *testing.T) {
	t.Parallel()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		syntax, err := parser.ParseFile(token.NewFileSet(), name, contents, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range syntax.Imports {
			if strings.Trim(imported.Path.Value, `"`) == "os/exec" {
				t.Fatalf("%s imports os/exec: this package validates argv but never launches", name)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no production sources; the check is blind")
	}
}
