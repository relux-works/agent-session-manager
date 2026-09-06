package secprim

import (
	"strings"
)

// Escape control symbols rendered visibly.
const (
	// escapeSymbol replaces ESC so ANSI CSI/OSC sequences become inert yet
	// visible: the byte the terminal would interpret is gone, and the
	// operator can still see that something was neutralized.
	escapeSymbol = "␛" // U+241B SYMBOL FOR ESCAPE
	// neutralizedSymbol replaces every other refused control: C0/C1
	// controls, DEL, bidi overrides, and zero-width format characters.
	neutralizedSymbol = "�" // U+FFFD REPLACEMENT CHARACTER
)

// EscapeForTerminal renders untrusted text inert for terminal output
// (Section 16.7: TUI and CLI rendering neutralizes ANSI/OSC sequences,
// control characters, bidi overrides, invalid encodings, and hostile-width
// graphemes before terminal output). The rule per rune:
//
//   - '\n' and '\t' pass through: multiline human output needs them, and
//     neither starts an escape sequence or reorders text;
//   - ESC becomes U+241B, so CSI, OSC, and every other ESC-led sequence
//     loses its introducer while staying visible;
//   - every other C0 control, DEL, and every C1 control (including the
//     single-byte CSI U+009B) becomes U+FFFD;
//   - bidi overrides and isolates (U+202A-U+202E, U+2066-U+2069), the
//     explicit directional marks (U+200E, U+200F, U+061C), and the
//     zero-width characters (U+200B-U+200D, U+FEFF) become U+FFFD: they
//     reorder or hide text without showing anything;
//   - invalid UTF-8 becomes U+FFFD per run, so the result is always valid
//     UTF-8 and no raw byte reaches the terminal;
//   - every other rune passes through unchanged.
//
// The function is idempotent: every symbol it emits is a fixed point, so
// escaping already-escaped output changes nothing and double-wiring a
// render path cannot corrupt text.
//
// Stated bounds. Soft hyphen (U+00AD), combining-mark floods, emoji
// presentation sequences, and other grapheme-width anomalies are not
// measured: neutralizing them needs terminal width tables this package
// does not carry, and stripping combining marks would mangle legitimate
// non-Latin text. Full-width spoofing beyond the enumerated invisible
// controls is therefore unproven and named here rather than claimed.
func EscapeForTerminal(text string) string {
	text = strings.ToValidUTF8(text, neutralizedSymbol)
	var rendered strings.Builder
	rendered.Grow(len(text))
	for _, character := range text {
		switch {
		case character == '\n' || character == '\t':
			rendered.WriteRune(character)
		case character == 0x1b:
			rendered.WriteString(escapeSymbol)
		case character < 0x20 || character == 0x7f || character >= 0x80 && character <= 0x9f:
			rendered.WriteString(neutralizedSymbol)
		case character >= 0x202a && character <= 0x202e,
			character >= 0x2066 && character <= 0x2069,
			character == 0x200e || character == 0x200f || character == 0x061c,
			character >= 0x200b && character <= 0x200d,
			character == 0xfeff:
			rendered.WriteString(neutralizedSymbol)
		default:
			rendered.WriteRune(character)
		}
	}
	return rendered.String()
}

// RenderForTerminal prepares one untrusted line for terminal display:
// secret scrubbing first (so a secret containing an escape sequence is
// removed as a secret, not merely neutralized), then control escaping.
// secrets carries the exact secret values the caller knows; nil admits no
// corpus redaction while key-pattern and private-key scrubbing still
// apply. The composition is idempotent because both stages are.
func RenderForTerminal(line string, secrets []string) string {
	return EscapeForTerminal(Redact(line, secrets))
}
