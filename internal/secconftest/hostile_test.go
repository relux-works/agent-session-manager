package secconftest

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

func TestHostileRosterQuotesResolve(t *testing.T) {
	t.Parallel()
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	for _, class := range Roster() {
		if err := QuoteInSection(document, class.Quote, class.Section); err != nil {
			t.Errorf("class %q: %v", class.ID, err)
		}
	}
}

func TestHostileRosterShape(t *testing.T) {
	t.Parallel()
	roster := Roster()
	if len(roster) != 18 {
		t.Fatalf("roster has %d classes, want 18; a spec class added or lost must update this instrument deliberately", len(roster))
	}
	seen := map[string]bool{}
	var memberless []string
	for _, class := range roster {
		if class.ID == "" || class.Section == "" || class.Quote == "" || class.Gate == "" {
			t.Errorf("class %+v has an empty field", class)
		}
		if seen[class.ID] {
			t.Errorf("duplicate class ID %q", class.ID)
		}
		seen[class.ID] = true
		switch class.Section {
		case "16.2", "16.3", "16.4", "16.7":
		default:
			t.Errorf("class %q section %q is outside the story scope", class.ID, class.Section)
		}
		if Members(class.ID) == nil {
			memberless = append(memberless, class.ID)
		}
	}
	// Exactly one class is an FS-behavior witness rather than a string
	// corpus: symlink escape at commit time needs a real symlink, which
	// Members cannot build. Its witness is
	// TestHostileSymlinkEscapeRefusedAtCommit, skip-gated on the
	// symlink capability.
	if strings.Join(memberless, ",") != "path-symlink-escape" {
		t.Fatalf("memberless classes = %v, want exactly [path-symlink-escape]", memberless)
	}
}

func TestQuoteInSectionRefusals(t *testing.T) {
	t.Parallel()
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	if err := QuoteInSection(document, "path traversal", "16.7"); err == nil {
		t.Error("a Section 16.3 quote accepted for 16.7; the section check is dead")
	}
	if err := QuoteInSection(document, "phrase that occurs nowhere in the document", "16.3"); err == nil {
		t.Error("unknown quote accepted")
	}
	// M27: a whitespace-only quote must be refused by the empty-quote
	// guard, not merely downstream by accident. Narrowing the guard
	// to quote == "" still refuses "   " (as "occurs nowhere"), so
	// the message is pinned, not just the refusal.
	if err := QuoteInSection(document, "   ", "16.3"); err == nil {
		t.Error("empty quote accepted")
	} else if !strings.Contains(err.Error(), "empty quote") {
		t.Errorf("whitespace quote refused as %q, want the empty-quote guard", err)
	}
	if err := QuoteInSection(nil, "path traversal", "16.3"); err == nil {
		t.Error("nil document accepted as satisfied; a failed read is never an absence")
	}
}

func TestHostileMemberCounts(t *testing.T) {
	t.Parallel()
	want := map[string]int{
		"path-traversal": 6, "path-absolute": 3, "path-drive-unc": 5,
		"path-altstream": 3, "path-encoded-separator": 9,
		"path-case-collision": 5, "path-unmanaged": 3, "path-device": 8,
		"argv-vector": 7, "env-allowlist": 9,
		"render-ansi-osc": 4, "render-controls": 63, "render-bidi": 9,
		"render-encoding": 5, "render-width": 7,
		"redact-persisted": 4, "redact-credentials": 5,
	}
	for id, count := range want {
		if got := len(Members(id)); got != count {
			t.Errorf("Members(%q) has %d inputs, want %d", id, got, count)
		}
	}
	if got := Members("no-such-class"); got != nil {
		t.Fatalf("Members of an unknown class = %v, want nil", got)
	}
}

func TestBidiRunesArePinned(t *testing.T) {
	t.Parallel()
	// The bidi/zero-width vocabulary is enumerated from the Unicode
	// code chart, not from the gate under test. Both narrowings —
	// dropping U+FEFF and cutting the list to one rune — must fail
	// here, and every rune must have a corpus member carrying it, so
	// the vocabulary cannot shrink past the corpus silently.
	want := []rune{
		0x202a, 0x202b, 0x202c, 0x202d, 0x202e,
		0x2066, 0x2067, 0x2068, 0x2069,
		0x200e, 0x200f, 0x061c,
		0x200b, 0x200c, 0x200d,
		0xfeff,
	}
	got := BidiRunes()
	if len(got) != len(want) {
		t.Fatalf("BidiRunes has %d runes, want %d", len(got), len(want))
	}
	for index, character := range want {
		if got[index] != character {
			t.Fatalf("BidiRunes[%d] = %U, want %U", index, got[index], character)
		}
	}
	covered := map[rune]bool{}
	for _, member := range append(Members("render-bidi"), Members("render-width")...) {
		for _, character := range member {
			covered[character] = true
		}
	}
	for _, character := range want {
		if !covered[character] {
			t.Errorf("rune %U has no corpus member; the vocabulary outruns the witness", character)
		}
	}
}

// gateWire pins one roster class to its production gate and to the
// decision that gate must return on the representative member.
type gateWire struct {
	wantGate    string
	platform    scalar.Platform
	member      string
	wantDecided bool
}

// driveGate resolves a roster Gate string to the production entry it
// names and requires the class-appropriate decision. An unknown Gate
// is a fatal wiring error, never a pass.
func driveGate(t *testing.T, gate, member string, platform scalar.Platform) bool {
	t.Helper()
	switch gate {
	case "secprim.CheckMemberPath":
		return secprim.CheckMemberPath(platform, member) == nil
	case "secprim.Guard.Open":
		stage := t.TempDir()
		guard, err := secprim.NewGuard(stage, platform, nil)
		if err != nil {
			t.Fatalf("NewGuard: %v", err)
		}
		root, err := secprim.OpenNoFollowDir(stage)
		if err != nil {
			t.Fatalf("OpenNoFollowDir: %v", err)
		}
		defer func() { _ = root.Close() }()
		file, err := guard.Open(root, member)
		if err != nil {
			return false
		}
		_ = file.Close()
		return true
	case "secprim.DetectCaseCollision":
		_, collided := secprim.DetectCaseCollision([]string{"README.md", "staging/member"}, member)
		return collided
	case "secprim.Guard.Resolve":
		guard, err := secprim.NewGuard("/stage", platform, []string{"staging/member"})
		if err != nil {
			t.Fatalf("NewGuard: %v", err)
		}
		_, err = guard.Resolve(member)
		return err == nil
	case "secprim.CheckArgv":
		return secprim.CheckArgv([]string{"exe", member}) == nil
	case "secprim.IsEnvName":
		return secprim.IsEnvName(member)
	case "secprim.EscapeForTerminal":
		escaped := secprim.EscapeForTerminal(member)
		assertTerminalInert(t, escaped)
		return escaped != member
	case "secprim.Redact":
		redacted := secprim.Redact(member, nil)
		return strings.Contains(redacted, "[redacted]")
	default:
		t.Fatalf("roster Gate %q resolves to no production entry", gate)
		return false
	}
}

func TestHostileGatesAreWired(t *testing.T) {
	t.Parallel()
	// Every roster Gate is pinned and driven: renaming a Gate to any
	// other entry fails the pin, and a Gate that stopped deciding its
	// class fails the drive. The symlink-escape drive opens an absent
	// member (refused); the commit-time escape proof itself is
	// TestHostileSymlinkEscapeRefusedAtCommit.
	wires := map[string]gateWire{
		"path-traversal":         {wantGate: "secprim.CheckMemberPath", platform: scalar.PlatformLinux, member: "../evil", wantDecided: false},
		"path-absolute":          {wantGate: "secprim.CheckMemberPath", platform: scalar.PlatformLinux, member: "/etc/passwd", wantDecided: false},
		"path-drive-unc":         {wantGate: "secprim.CheckMemberPath", platform: scalar.PlatformLinux, member: "C:/Windows", wantDecided: false},
		"path-altstream":         {wantGate: "secprim.CheckMemberPath", platform: scalar.PlatformWindows, member: "file:stream", wantDecided: false},
		"path-encoded-separator": {wantGate: "secprim.CheckMemberPath", platform: scalar.PlatformLinux, member: "a%2fb", wantDecided: false},
		"path-symlink-escape":    {wantGate: "secprim.Guard.Open", platform: scalar.PlatformLinux, member: "absent-member", wantDecided: false},
		"path-case-collision":    {wantGate: "secprim.DetectCaseCollision", platform: scalar.PlatformLinux, member: "README.MD", wantDecided: true},
		"path-unmanaged":         {wantGate: "secprim.Guard.Resolve", platform: scalar.PlatformLinux, member: "outside-managed", wantDecided: false},
		"path-device":            {wantGate: "secprim.CheckMemberPath", platform: scalar.PlatformWindows, member: "NUL", wantDecided: false},
		"argv-vector":            {wantGate: "secprim.CheckArgv", platform: scalar.PlatformLinux, member: "; rm -rf /", wantDecided: true},
		"env-allowlist":          {wantGate: "secprim.IsEnvName", platform: scalar.PlatformLinux, member: "", wantDecided: false},
		"render-ansi-osc":        {wantGate: "secprim.EscapeForTerminal", platform: scalar.PlatformLinux, member: "a\x1b[31mb", wantDecided: true},
		"render-controls":        {wantGate: "secprim.EscapeForTerminal", platform: scalar.PlatformLinux, member: "a\x00b", wantDecided: true},
		"render-bidi":            {wantGate: "secprim.EscapeForTerminal", platform: scalar.PlatformLinux, member: "a\u202eb", wantDecided: true},
		"render-encoding":        {wantGate: "secprim.EscapeForTerminal", platform: scalar.PlatformLinux, member: "a\xffb", wantDecided: true},
		"render-width":           {wantGate: "secprim.EscapeForTerminal", platform: scalar.PlatformLinux, member: "a\ufeffb", wantDecided: true},
		"redact-persisted":       {wantGate: "secprim.Redact", platform: scalar.PlatformLinux, member: "password=hunter2", wantDecided: true},
		"redact-credentials":     {wantGate: "secprim.Redact", platform: scalar.PlatformLinux, member: "api_key=[REDACTED]", wantDecided: true},
	}
	for _, class := range Roster() {
		wire, ok := wires[class.ID]
		if !ok {
			t.Errorf("class %q has no gate wiring; its Gate annotation is unchecked", class.ID)
			continue
		}
		if class.Gate != wire.wantGate {
			t.Errorf("class %q Gate = %q, want %q", class.ID, class.Gate, wire.wantGate)
			continue
		}
		if got := driveGate(t, class.Gate, wire.member, wire.platform); got != wire.wantDecided {
			t.Errorf("class %q through %s: decided=%v, want %v", class.ID, class.Gate, got, wire.wantDecided)
		}
	}
	for id := range wires {
		found := false
		for _, class := range Roster() {
			if class.ID == id {
				found = true
			}
		}
		if !found {
			t.Errorf("wire row %q names no roster class; the expectation is orphaned", id)
		}
	}
}

func TestHostilePathMembersRefused(t *testing.T) {
	t.Parallel()
	linux := []string{"path-traversal", "path-absolute", "path-drive-unc", "path-encoded-separator"}
	for _, id := range linux {
		for _, member := range Members(id) {
			if err := secprim.CheckMemberPath(scalar.PlatformLinux, member); err == nil {
				t.Errorf("linux admits hostile %q (%s)", member, id)
			}
		}
	}
	windows := []string{"path-altstream", "path-device"}
	for _, id := range windows {
		for _, member := range Members(id) {
			if err := secprim.CheckMemberPath(scalar.PlatformWindows, member); err == nil {
				t.Errorf("windows admits hostile %q (%s)", member, id)
			}
		}
	}
}

func TestHostileCaseCollisionMembersFire(t *testing.T) {
	t.Parallel()
	siblings := []string{"README.md", "staging/member"}
	for _, candidate := range Members("path-case-collision") {
		if _, collided := secprim.DetectCaseCollision(siblings, candidate); !collided {
			t.Errorf("candidate %q reports no collision against %v", candidate, siblings)
		}
	}
	if _, collided := secprim.DetectCaseCollision(siblings, "unrelated-name"); collided {
		t.Error("unrelated name collides; the gate over-matches")
	}
}

func TestHostileUnmanagedMembersRefused(t *testing.T) {
	t.Parallel()
	guard, err := secprim.NewGuard("/stage", scalar.PlatformLinux, []string{"staging/member"})
	if err != nil {
		t.Fatalf("NewGuard: %v", err)
	}
	for _, member := range Members("path-unmanaged") {
		if _, err := guard.Resolve(member); err == nil {
			t.Errorf("unmanaged member %q resolved", member)
		} else if !errors.Is(err, secprim.ErrUnsafePath) {
			t.Errorf("unmanaged member %q error %v does not wrap ErrUnsafePath", member, err)
		}
	}
	if _, err := guard.Resolve("staging/member"); err != nil {
		t.Errorf("managed member refused: %v", err)
	}
}

func TestHostileArgvMembersStaySingleElements(t *testing.T) {
	t.Parallel()
	for _, hostile := range Members("argv-vector") {
		command, err := secprim.NewCommand([]string{"exe", hostile})
		if err != nil {
			t.Errorf("shell metacharacter vector %q refused as data: %v", hostile, err)
			continue
		}
		argv := command.Argv()
		if len(argv) != 2 || argv[1] != hostile {
			t.Errorf("vector %q did not survive as one element: %q", hostile, argv)
		}
	}
	refused := [][]string{{}, {""}, {"exe", ""}, {"exe", "a\x00b"}, {"exe", "a\xffb"}}
	for _, argv := range refused {
		if err := secprim.CheckArgv(argv); err == nil {
			t.Errorf("CheckArgv(%q) admitted", argv)
		} else if !errors.Is(err, secprim.ErrUnsafeArgv) {
			t.Errorf("CheckArgv(%q) error %v does not wrap ErrUnsafeArgv", argv, err)
		}
	}
}

func TestHostileEnvNamesRefused(t *testing.T) {
	t.Parallel()
	for _, name := range Members("env-allowlist") {
		if secprim.IsEnvName(name) {
			t.Errorf("IsEnvName(%q) admitted", name)
		}
	}
	if !secprim.IsEnvName("AX_FOO") {
		t.Error("IsEnvName refuses a well-formed name; the gate over-matches")
	}
	if _, err := secprim.BuildEnv([]string{"AX_OK"}, map[string]string{"AX_OK": "x"}, os.LookupEnv); err == nil {
		t.Error("BuildEnv admits a literal colliding with the allowlist")
	}
	if _, err := secprim.BuildEnv([]string{"AX_OK"}, map[string]string{"AX_BAD": "a\x00b"}, os.LookupEnv); err == nil {
		t.Error("BuildEnv admits a NUL value")
	}
}

func TestHostileRenderMembersNeutralized(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"render-ansi-osc", "render-controls", "render-bidi", "render-encoding", "render-width"} {
		for _, member := range Members(id) {
			escaped := secprim.EscapeForTerminal(member)
			// The output invariant every class serves: valid UTF-8,
			// no ESC, no refused control, no bidi/zero-width rune.
			assertTerminalInert(t, escaped)
			switch id {
			case "render-encoding":
				// Invalid bytes cannot survive, and the neutralized
				// replacement proves they were rewritten, not kept.
				if escaped == member {
					t.Errorf("class %s: member %q passes through unchanged", id, member)
				}
			default:
				// Neutralization is visible, never a silent strip:
				// ESC becomes U+241B, every other refused rune
				// becomes U+FFFD.
				if strings.ContainsRune(member, 0x1b) && !strings.Contains(escaped, "␛") {
					t.Errorf("class %s: member %q lost its escape visibly", id, member)
				}
				if !strings.ContainsRune(member, 0x1b) && !strings.Contains(escaped, "�") {
					t.Errorf("class %s: member %q shows no neutralization marker", id, member)
				}
			}
		}
	}
}

func TestHostileRedactMembersScrubbed(t *testing.T) {
	t.Parallel()
	for _, member := range Members("redact-persisted") {
		redacted := secprim.Redact(member, nil)
		if strings.Contains(redacted, "PRIVATE KEY-----\n") || strings.Contains(redacted, "s3cret") {
			t.Errorf("member %q survives redaction: %q", member, redacted)
		}
	}
	for _, member := range Members("redact-credentials") {
		redacted := secprim.Redact(member, nil)
		if !strings.Contains(redacted, "[redacted]") {
			t.Errorf("credential member %q carries no redaction marker: %q", member, redacted)
		}
	}
	// Caller-known corpus values are scrubbed verbatim, including when
	// the key shape is innocuous.
	const secret = "correct-horse-battery"
	redacted := secprim.Redact(" InnocuousKey="+secret+" ", []string{secret})
	if strings.Contains(redacted, secret) {
		t.Fatalf("corpus secret survives: %q", redacted)
	}
}

func TestHostileSymlinkEscapeRefusedAtCommit(t *testing.T) {
	t.Parallel()
	Require(t, SymlinkCapability())
	base := t.TempDir()
	stage := filepath.Join(base, "stage")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "victim"), []byte("OUTSIDE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "inside"), []byte("INSIDE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(stage, "link")); err != nil {
		t.Fatal(err)
	}
	guard, err := secprim.NewGuard(stage, scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := secprim.OpenNoFollowDir(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	file, err := guard.Open(root, "link/victim")
	if err == nil {
		contents, _ := io.ReadAll(file)
		_ = file.Close()
		t.Fatalf("symlink escape committed; read %q outside the root", contents)
	}
	if !errors.Is(err, secprim.ErrUnsafePath) {
		t.Fatalf("symlink escape error %v does not wrap ErrUnsafePath", err)
	}
	inside, err := guard.Open(root, "inside")
	if err != nil {
		t.Fatalf("positive control refused: %v", err)
	}
	contents, err := io.ReadAll(inside)
	_ = inside.Close()
	if err != nil || string(contents) != "INSIDE" {
		t.Fatalf("positive control read %q, %v; want the staged bytes", contents, err)
	}
}
