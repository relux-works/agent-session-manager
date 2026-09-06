package secconftest

import (
	"fmt"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// HostileClass is one spec-derived hostile-input class: the exact
// specification phrase it comes from, the section that phrase lives
// in, and the production gate that decides its members.
//
// The roster is derived, not listed: TestHostileRosterQuotesResolve
// requires every Quote to occur in its Section in the pinned document,
// so a class added to or removed from the specification fails the
// roster instead of passing silently. Members are generated from
// Unicode tables and internal/scalar predicates, never from the
// secprim tables under test, so deleting a gate arm keeps the corpus
// and reddens the gate test instead of shrinking the witness with it.
type HostileClass struct {
	// ID is the stable slug naming the class in test output.
	ID string
	// Section is the AX specification section the Quote is drawn from.
	Section string
	// Quote is the exact specification phrase deriving this class.
	Quote string
	// Gate names the production entry point deciding the members.
	Gate string
}

// Roster lists every hostile class this instrument drives.
func Roster() []HostileClass {
	return []HostileClass{
		{ID: "path-traversal", Section: "16.3", Quote: "path traversal", Gate: "secprim.CheckMemberPath"},
		{ID: "path-absolute", Section: "16.3", Quote: "absolute paths", Gate: "secprim.CheckMemberPath"},
		{ID: "path-drive-unc", Section: "16.3", Quote: "drive/UNC injection", Gate: "secprim.CheckMemberPath"},
		{ID: "path-altstream", Section: "16.3", Quote: "alternate streams", Gate: "secprim.CheckMemberPath"},
		{ID: "path-encoded-separator", Section: "16.3", Quote: "repeatedly encoded separators", Gate: "secprim.CheckMemberPath"},
		{ID: "path-symlink-escape", Section: "16.3", Quote: "symlink/reparse escape during both validation and commit", Gate: "secprim.Guard.Open"},
		{ID: "path-case-collision", Section: "16.3", Quote: "case-fold collisions on case-insensitive destinations", Gate: "secprim.DetectCaseCollision"},
		{ID: "path-unmanaged", Section: "16.3", Quote: "replacement of unmanaged paths", Gate: "secprim.Guard.Resolve"},
		{ID: "path-device", Section: "16.3", Quote: "device/FIFO/socket creation", Gate: "secprim.CheckMemberPath"},
		{ID: "argv-vector", Section: "16.4", Quote: "Shell command construction from names, paths, provider output, or peer data is forbidden", Gate: "secprim.CheckArgv"},
		{ID: "env-allowlist", Section: "16.7", Quote: "minimal environment allowlists", Gate: "secprim.IsEnvName"},
		{ID: "render-ansi-osc", Section: "16.7", Quote: "ANSI/OSC sequences", Gate: "secprim.EscapeForTerminal"},
		{ID: "render-controls", Section: "16.7", Quote: "control characters", Gate: "secprim.EscapeForTerminal"},
		{ID: "render-bidi", Section: "16.7", Quote: "bidi overrides", Gate: "secprim.EscapeForTerminal"},
		{ID: "render-encoding", Section: "16.7", Quote: "invalid encodings", Gate: "secprim.EscapeForTerminal"},
		{ID: "render-width", Section: "16.7", Quote: "hostile-width graphemes", Gate: "secprim.EscapeForTerminal"},
		{ID: "redact-persisted", Section: "16.4", Quote: "redacted before persistence", Gate: "secprim.Redact"},
		{ID: "redact-credentials", Section: "16.2", Quote: "API keys, OAuth tokens", Gate: "secprim.Redact"},
	}
}

// QuoteInSection reports whether quote occurs on at least one document
// line inside section. The section check is load-bearing, not
// ceremony: TestQuoteInSectionRejectsWrongSection drives a real quote
// against a wrong section and requires refusal, so a mutant that
// preserves the token match but drops the section check fails. The
// mutant harness for that test executes the full behavioral hostile
// battery, not only this checker.
func QuoteInSection(document *specdoc.Document, quote, section string) error {
	if document == nil {
		return fmt.Errorf("secconftest: nil specification document is never a legitimate absence")
	}
	if strings.TrimSpace(quote) == "" {
		return fmt.Errorf("secconftest: empty quote matches nothing")
	}
	lines := document.QuoteLines(quote)
	if len(lines) == 0 {
		return fmt.Errorf("secconftest: quote %q occurs nowhere in the pinned document", quote)
	}
	for _, line := range lines {
		if got, ok := document.SectionID(line); ok && got == section {
			return nil
		}
	}
	return fmt.Errorf("secconftest: quote %q occurs but never in section %s", quote, section)
}

// bidiOverrideRunes enumerates the Section 16.7 bidi class from the
// Unicode code chart, not from the gate under test: embeddings and
// overrides U+202A-U+202E, isolates U+2066-U+2069, directional marks
// U+200E, U+200F, U+061C, and zero-width U+200B-U+200D plus U+FEFF.
// The static list is the point: a gate arm deleted with its test row
// still faces every rune here.
func bidiOverrideRunes() []rune {
	return []rune{
		0x202a, 0x202b, 0x202c, 0x202d, 0x202e,
		0x2066, 0x2067, 0x2068, 0x2069,
		0x200e, 0x200f, 0x061c,
		0x200b, 0x200c, 0x200d,
		0xfeff,
	}
}

// Members generates the hostile inputs for a roster class ID. Every
// member carries the class token (for example ".." for traversal):
// the gates must refuse behavior, not merely match the token, and the
// per-class gate tests assert refusal of every member, so a
// token-preserving behavior change still reddens.
func Members(id string) []string {
	wrap := func(runes []rune) []string {
		members := make([]string, 0, len(runes))
		for _, character := range runes {
			members = append(members, "a"+string(character)+"b")
		}
		return members
	}
	switch id {
	case "path-traversal":
		return []string{"../evil", "a/../../evil", "..", "a/../b", "a/b/../../../../c", "....//evil"}
	case "path-absolute":
		return []string{"/etc/passwd", "/", "/a/b"}
	case "path-drive-unc":
		return []string{"C:/Windows", "C:relative", "d:foo", "//server/share", `\\server\share`}
	case "path-altstream":
		return []string{"file:stream", "dir/file:ads", "file:$DATA"}
	case "path-encoded-separator":
		return []string{"a%2fb", "a%5cb", "a%252fb", "a%2F..%2Fb", "%2e%2e/evil", "a/%2e%2e/b", "%2E%2E", "a%c0%afb", "a%C0%AFb"}
	case "path-case-collision":
		return []string{"readme.md", "README.MD", "ReadMe.Md", "STAGING/MEMBER", "Staging/Member"}
	case "path-unmanaged":
		return []string{"outside-managed", "staging/other", "secret"}
	case "path-device":
		return []string{"NUL", "CON", "COM1", "LPT1.txt", "aux.log", "dir/PRN", "COM9", "nul"}
	case "argv-vector":
		return []string{"; rm -rf /", "$(evil)", "`evil`", "a|b", "a&&b", "a\nb", "--flag=value with spaces"}
	case "env-allowlist":
		return []string{"", "0", "1ABC", "A-B", "A B", "A.B", "A/B", "A\x00B", strings.Repeat("A", 129)}
	case "render-ansi-osc":
		return []string{"a\x1b[31mb", "a\x1b[0mb", "a\x1b]8;;https://evil\x07b", "a\x9bb"}
	case "render-controls":
		// C0 controls except newline and tab (which the gate keeps),
		// DEL, and C1 controls: the exact refused set the section
		// names as "control characters". Printable runes are not
		// hostile and are not members.
		var runes []rune
		for character := rune(0); character < 0x20; character++ {
			if character == '\n' || character == '\t' {
				continue
			}
			runes = append(runes, character)
		}
		runes = append(runes, 0x7f)
		for character := rune(0x80); character < 0xa0; character++ {
			runes = append(runes, character)
		}
		return wrap(runes)
	case "render-bidi":
		return wrap([]rune{0x202a, 0x202b, 0x202c, 0x202d, 0x202e, 0x2066, 0x2067, 0x2068, 0x2069})
	case "render-encoding":
		return []string{"a\xffb", "a\xfeb", "\xff", "\xfe", "a\xed\xa0\x80b"}
	case "render-width":
		return wrap([]rune{0x200e, 0x200f, 0x061c, 0x200b, 0x200c, 0x200d, 0xfeff})
	case "redact-persisted":
		return []string{
			"-----BEGIN RSA PRIVATE KEY-----\nMIIB\n-----END RSA PRIVATE KEY-----",
			"key without end -----BEGIN RSA PRIVATE KEY-----\nMIIB",
			"https://deploy:s3cret@example.com/repo",
			"password=hunter2",
		}
	case "redact-credentials":
		return []string{
			"api_key=AKIAIOSFODNN7EXAMPLE",
			"authorization: Bearer eyJhbGciOi",
			`"token": "xoxb-secret-value"`,
			"cookie=sessionid=abc123",
			"subscription_token=st_live_42",
		}
	default:
		return nil
	}
}

// BidiRunes exposes the full bidi/zero-width rune list for gate tests
// that must assert the enumerated set without retyping it.
func BidiRunes() []rune {
	return bidiOverrideRunes()
}
