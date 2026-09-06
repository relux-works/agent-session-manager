package secprim

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestEscapeForTerminal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "hello world", "hello world"},
		{"newline kept", "a\nb", "a\nb"},
		{"tab kept", "a\tb", "a\tb"},
		{"csi neutralized", "\x1b[31mred\x1b[0m", "␛[31mred␛[0m"},
		{"osc neutralized", "\x1b]8;;https://evil\x07link", "␛]8;;https://evil�link"},
		{"bare esc", "a\x1bb", "a␛b"},
		{"single byte csi", "a\x9bb", "a�b"},
		{"c0 nul", "a\x00b", "a�b"},
		{"c0 bell", "a\x07b", "a�b"},
		{"c0 cr", "a\rb", "a�b"},
		{"del", "a\x7fb", "a�b"},
		{"c1", "a\x80b", "a�b"},
		{"bidi override", "a\u202eb", "a�b"},
		{"bidi embedding 202a", "a\u202ab", "a�b"},
		{"bidi embedding 202b", "a\u202bb", "a�b"},
		{"bidi pop 202c", "a\u202cb", "a�b"},
		{"bidi override 202d", "a\u202db", "a�b"},
		{"bidi isolate", "a\u2066b\u2067c", "a�b�c"},
		{"bidi isolate 2068", "a\u2068b", "a�b"},
		{"bidi isolate 2069", "a\u2069b", "a�b"},
		{"directional mark", "a\u200fb", "a�b"},
		{"directional mark 200e", "a\u200eb", "a�b"},
		{"directional isolate 061c", "a\u061cb", "a�b"},
		{"zero width space", "a\u200bb", "a�b"},
		{"zero width non-joiner", "a\u200cb", "a�b"},
		{"zero width joiner", "a\u200db", "a�b"},
		{"bom", "a\ufeffb", "a�b"},
		{"invalid utf8", "a\xffb", "a�b"},
		{"unicode kept", "ünïcodé 名前 privet", "ünïcodé 名前 privet"},
		{"symbol for escape kept", "a␛b", "a␛b"},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			t.Parallel()
			if got := EscapeForTerminal(kase.input); got != kase.want {
				t.Fatalf("EscapeForTerminal(%q) = %q, want %q", kase.input, got, kase.want)
			}
		})
	}
}

// TestEscapeForTerminalOutputBytes is the output invariant every refusal
// class above serves: no ESC byte, no C0 byte except newline and tab, no
// DEL, and valid UTF-8. It sweeps a hostile corpus so a dropped class
// fails here even if its table row is deleted with it.
//
// The fed runes and the asserted runes are one slice: an assertion can
// never outrun its feed again, so no enumerated code point is
// unwitnessed the way U+202A-U+202D and U+2068-U+2069 once were.
func TestEscapeForTerminalOutputBytes(t *testing.T) {
	t.Parallel()
	// Every invisible control EscapeForTerminal neutralizes: the five
	// bidi embeddings and overrides, the four isolates, the three
	// directional marks, the three zero-width characters, and the byte
	// order mark.
	forbidden := []rune{
		0x202a, 0x202b, 0x202c, 0x202d, 0x202e,
		0x2066, 0x2067, 0x2068, 0x2069,
		0x200e, 0x200f, 0x061c,
		0x200b, 0x200c, 0x200d,
		0xfeff,
	}
	var corpus strings.Builder
	for character := 0; character < 0xa0; character++ {
		corpus.WriteRune(rune(character))
	}
	for _, character := range forbidden {
		corpus.WriteRune(character)
	}
	corpus.WriteString("\x1b[0m\x1b]0;title\x07\x9b0m\xff\xfe")
	escaped := EscapeForTerminal(corpus.String())
	if !utf8.ValidString(escaped) {
		t.Fatal("escaped output is not valid UTF-8")
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
	for _, refused := range forbidden {
		if strings.ContainsRune(escaped, refused) {
			t.Fatalf("escaped output carries %U", refused)
		}
	}
}

// TestEscapeForTerminalIsIdempotent proves double-wiring a render path
// cannot corrupt text: every emitted symbol is a fixed point.
func TestEscapeForTerminalIsIdempotent(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"plain", "\x1b[31mred\x1b[0m", "a\u202eb\u200bc", "a\x00b\x7f\x9b", "ünïcodé",
		"␛ already visible", "� already neutral",
	}
	for _, input := range inputs {
		once := EscapeForTerminal(input)
		if twice := EscapeForTerminal(once); twice != once {
			t.Errorf("EscapeForTerminal is not idempotent on %q: %q then %q", input, once, twice)
		}
	}
}

func TestRenderForTerminal(t *testing.T) {
	t.Parallel()
	// Scrubbing runs before escaping: a secret containing an escape
	// sequence is removed as a secret, not merely neutralized.
	got := RenderForTerminal("token=hunter2-hunter \x1b[1m", []string{"hunter2-hunter"})
	want := "token=[redacted] ␛[1m"
	if got != want {
		t.Fatalf("RenderForTerminal = %q, want %q", got, want)
	}
	if got := RenderForTerminal("plain", nil); got != "plain" {
		t.Fatalf("RenderForTerminal(plain) = %q", got)
	}
	once := RenderForTerminal("password=x \x07", nil)
	if twice := RenderForTerminal(once, nil); twice != once {
		t.Fatalf("RenderForTerminal is not idempotent: %q then %q", once, twice)
	}
}
