package secconftest

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/secprim"
)

var fuzzDriver = NewDriver()

var fuzzSeeds = []string{
	"",
	"a",
	"staging/member",
	"../evil",
	"/absolute",
	"C:/drive",
	"a%2fb",
	"file:stream",
	"NUL",
	"password=hunter2",
	"-----BEGIN RSA PRIVATE KEY-----\nMIIB\n-----END RSA PRIVATE KEY-----",
	"https://user:pass@example.com/x",
	"a\x1b[31mb",
	"a\u202eb",
	"a\x00b",
	"a\xffb",
	"; rm -rf /",
	"$(evil)",
	"AX_VALID_NAME",
	"1bad-name",
	strings.Repeat("A", 129),
	"readme.md",
}

func TestFuzzDriversReachProduction(t *testing.T) {
	t.Parallel()
	driver := NewDriver()
	for _, seed := range fuzzSeeds {
		driver.DriveArgv([]string{"exe", seed})
		driver.DriveMemberPath(seed)
		driver.DriveEnvName(seed)
		driver.DriveRedact(seed)
		driver.DriveEscape(seed)
		driver.DriveRender(seed)
		driver.DriveGuardResolve(seed)
		driver.DriveCaseCollision(seed)
	}
	for entry, want := range map[FuzzEntry]bool{
		EntryCheckArgv: true, EntryCheckMemberPath: true, EntryIsEnvName: true,
		EntryRedact: true, EntryEscapeForTerminal: true, EntryRenderForTerminal: true,
		EntryGuardResolve: true, EntryDetectCaseCollision: true,
	} {
		if !want {
			continue
		}
		if got := driver.Count(entry); got == 0 {
			t.Fatalf("no seed reached %s; the corpus cannot cover it", entry)
		}
		t.Logf("entry %s arrivals: %d", entry, driver.Count(entry))
	}
	if len(FuzzTargetEntry) != 8 {
		t.Fatalf("FuzzTargetEntry has %d targets, want the 8 committed Fuzz functions", len(FuzzTargetEntry))
	}
	seen := map[FuzzEntry]bool{}
	for target, entry := range FuzzTargetEntry {
		if !strings.HasPrefix(target, "Fuzz") || entry == "" {
			t.Fatalf("FuzzTargetEntry[%q] = %q is malformed", target, entry)
		}
		seen[entry] = true
	}
	if len(seen) != 8 {
		t.Fatalf("FuzzTargetEntry covers %d distinct entries, want 8", len(seen))
	}
}

func FuzzCheckArgv(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, element string) {
		admitted := fuzzDriver.DriveArgv([]string{"exe", element})
		if admitted && element == "" {
			t.Fatal("empty argv element admitted")
		}
		// Single-gate agreement: NewCommand validates through the same
		// CheckArgv, so the two must agree on every input.
		_, err := secprim.NewCommand([]string{"exe", element})
		if admitted != (err == nil) {
			t.Fatalf("CheckArgv(%q)=%v disagrees with NewCommand err=%v", element, admitted, err)
		}
	})
}

func FuzzCheckMemberPath(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, member string) {
		admitted := fuzzDriver.DriveMemberPath(member)
		if !admitted {
			return
		}
		// Containment property: every admitted member resolves inside
		// the fixed root through the production Guard.Resolve.
		if !fuzzDriver.DriveGuardResolve(member) {
			t.Fatalf("member %q admitted by CheckMemberPath but refused by Guard.Resolve", member)
		}
	})
}

func FuzzIsEnvName(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		// Poles a no-op gate fails: the empty name is never
		// admitted, while a well-formed name always is. The
		// interior grammar stays covered by the hostile corpus
		// (TestHostileEnvNamesRefused), stated in doc.go.
		if fuzzDriver.DriveEnvName("") {
			t.Fatal("empty env name admitted; the gate admits everything")
		}
		if !fuzzDriver.DriveEnvName("AX_FOO") {
			t.Fatal("well-formed env name refused; the gate refuses everything")
		}
		if !fuzzDriver.DriveEnvName(name) {
			return
		}
		// An admitted name must build through the production allowlist
		// path: absent from the parent environment is skipped, never
		// failed.
		if _, err := secprim.BuildEnv([]string{name}, nil, func(string) (string, bool) { return "", false }); err != nil {
			t.Fatalf("admitted env name %q fails BuildEnv: %v", name, err)
		}
	})
}

func FuzzRedactCorpus(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, line string) {
		fuzzDriver.DriveRedact(line)
		// Pins a no-op (identity) gate fails: a known secret shape
		// must be rewritten, while an innocuous line must pass
		// through. Per-class marker coverage stays in the hostile
		// corpus (TestHostileRedactMembersScrubbed), stated in
		// doc.go.
		if got := secprim.Redact("password=hunter2", nil); got == "password=hunter2" {
			t.Fatal("known secret passes through unchanged; the gate rewrites nothing")
		}
		if got := secprim.Redact("plain hello", nil); got != "plain hello" {
			t.Fatalf("innocuous line rewritten to %q; the gate over-matches", got)
		}
		// Idempotency the package documents: rescanning redacted text
		// changes nothing.
		once := secprim.Redact(line, nil)
		if twice := secprim.Redact(once, nil); twice != once {
			t.Fatalf("Redact not idempotent on %q: %q then %q", line, once, twice)
		}
	})
}

func FuzzEscapeForTerminal(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		escaped := fuzzDriver.DriveEscape(text)
		assertTerminalInert(t, escaped)
	})
}

func FuzzRenderForTerminal(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, line string) {
		rendered := fuzzDriver.DriveRender(line)
		// Wiring property: Render is Redact then Escape, in that order,
		// so a secret containing an escape sequence is removed as a
		// secret rather than merely neutralized.
		if want := secprim.EscapeForTerminal(secprim.Redact(line, nil)); rendered != want {
			t.Fatalf("RenderForTerminal(%q) = %q, want Escape(Redact) = %q", line, rendered, want)
		}
		assertTerminalInert(t, rendered)
	})
}

func FuzzGuardResolve(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, member string) {
		if !fuzzDriver.DriveGuardResolve(member) {
			return
		}
		guard, err := secprim.NewGuard("/stage", "linux", nil)
		if err != nil {
			t.Fatalf("fixed guard construction: %v", err)
		}
		resolved, err := guard.Resolve(member)
		if err != nil {
			t.Fatalf("DriveGuardResolve admitted %q but Resolve refuses: %v", member, err)
		}
		if resolved != "/stage/"+member && !strings.HasPrefix(resolved, "/stage/") {
			t.Fatalf("Resolve(%q) = %q escapes the root", member, resolved)
		}
	})
}

func FuzzDetectCaseCollision(f *testing.F) {
	for _, seed := range fuzzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, candidate string) {
		collided := fuzzDriver.DriveCaseCollision(candidate)
		if !collided {
			return
		}
		// A reported collision must fold-equal a listed sibling it is
		// not identical to.
		for _, sibling := range []string{"README.md", "staging/member"} {
			if sibling != candidate && strings.EqualFold(sibling, candidate) {
				return
			}
		}
		t.Fatalf("collision reported for %q with no fold-equal sibling", candidate)
	})
}

func assertTerminalInert(t *testing.T, escaped string) {
	t.Helper()
	if !utf8.ValidString(escaped) {
		t.Fatalf("escaped output %q is not valid UTF-8", escaped)
	}
	for _, character := range escaped {
		switch {
		case character == '\n' || character == '\t':
		case character < 0x20 || character == 0x7f || character >= 0x80 && character <= 0x9f:
			t.Fatalf("escaped output carries control %U", character)
		case character == 0x1b:
			t.Fatal("escaped output carries ESC")
		}
	}
	for _, refused := range BidiRunes() {
		if strings.ContainsRune(escaped, refused) {
			t.Fatalf("escaped output carries %U", refused)
		}
	}
}
