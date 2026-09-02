package config

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file proves that the configuration extension byte bound is measured on
// canonical bytes at the production reader and writer.
//
// The defect it exists to prevent: the bound was measured on Go's HTML-escaped
// JSON encoding, which rewrites each of the single bytes <, > and & into a
// six-character escape and U+2028 and U+2029 from three bytes into six. A
// conforming extension object built from those characters measured up to six
// times its canonical size and was refused, and this package's measurement
// disagreed with the canonicaljson one for the same declared limit. Every
// existing fixture was made of characters whose two encodings have the same
// length, so nothing reddened.

// specificationExtensionCanonicalMaxBytes is the byte bound that
// relux-works/agent-session-manager-spec@v0.5.0 Section 6 fixes for an
// extensions object, written as the specification literal rather than read from
// maxConfigExtensionBytes, so widening the implementation constant cannot widen
// this proof with it.
const specificationExtensionCanonicalMaxBytes = 65_536

// extensionBoundKey is the single extensions member the fixtures below carry.
// A canonical one-member object is "{" plus the quoted key, ":", the quoted
// value and "}", so its length is the key length plus the value length plus
// seven.
const extensionBoundKey = "works.relux.bytes"

// extensionMeasurementFiller is one fill character with the number of canonical
// bytes it occupies and whether Go's default JSON encoding inflates it.
type extensionMeasurementFiller struct {
	name           string
	text           string
	canonicalWidth int
	expands        bool
}

func extensionMeasurementFillers() []extensionMeasurementFiller {
	return []extensionMeasurementFiller{
		{name: "less than", text: "<", canonicalWidth: 1, expands: true},
		{name: "greater than", text: ">", canonicalWidth: 1, expands: true},
		{name: "ampersand", text: "&", canonicalWidth: 1, expands: true},
		{name: "line separator", text: string(rune(0x2028)), canonicalWidth: 3, expands: true},
		{name: "paragraph separator", text: string(rune(0x2029)), canonicalWidth: 3, expands: true},
		// The control row: an encoding-neutral character sharing the same
		// fixture machinery as the rows above.
		{name: "encoding neutral ASCII", text: "x", canonicalWidth: 1, expands: false},
	}
}

// TestConfigurationExtensionByteBoundIsMeasuredOnCanonicalBytes drives the
// production writer and the production reader with an extensions object that
// sits exactly on the declared bound, and then one canonical byte past it.
func TestConfigurationExtensionByteBoundIsMeasuredOnCanonicalBytes(t *testing.T) {
	t.Parallel()

	for _, filler := range extensionMeasurementFillers() {
		t.Run(filler.name, func(t *testing.T) {
			t.Parallel()

			atLimit := extensionsWithCanonicalBytes(t, filler, specificationExtensionCanonicalMaxBytes)
			assertEscapedExtensionMeasurementWouldRefuse(t, filler, atLimit)

			accepted := configurationWithExtensions(atLimit)
			document, err := EncodeCurrent(accepted, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
			if err != nil {
				t.Fatalf("EncodeCurrent(extensions at the declared bound) error = %v, want acceptance", err)
			}
			if _, err := loadConfigDocument(document, scalar.PlatformMacOS, nil); err != nil {
				t.Fatalf("Load(extensions at the declared bound) error = %v, want acceptance", err)
			}

			overLimit := extensionsWithCanonicalBytes(t, filler, specificationExtensionCanonicalMaxBytes+1)
			refused := configurationWithExtensions(overLimit)
			if _, err := EncodeCurrent(refused, DecodeContext{RuntimePlatform: scalar.PlatformMacOS}); !errors.Is(err, ErrConfigValidation) {
				t.Fatalf("EncodeCurrent(one canonical byte past the bound) error = %v, want ErrConfigValidation", err)
			}
			if _, err := loadConfigDocument(oneBytePastDocument(t, document), scalar.PlatformMacOS, nil); !errors.Is(err, ErrConfigValidation) {
				t.Fatalf("Load(one canonical byte past the bound) error = %v, want ErrConfigValidation", err)
			}
		})
	}
}

// TestConfigurationAndIdentityBoundsShareOneMeasurement pins the shared helper
// as the one both packages use, so the two sites cannot drift apart again.
func TestConfigurationAndIdentityBoundsShareOneMeasurement(t *testing.T) {
	t.Parallel()

	for _, filler := range extensionMeasurementFillers() {
		t.Run(filler.name, func(t *testing.T) {
			t.Parallel()

			extensions := extensionsWithCanonicalBytes(t, filler, specificationExtensionCanonicalMaxBytes)
			measured, err := canonicaljson.CanonicalByteLength(extensions)
			if err != nil {
				t.Fatalf("CanonicalByteLength(%s extensions) error = %v", filler.name, err)
			}
			if measured != specificationExtensionCanonicalMaxBytes {
				t.Fatalf("CanonicalByteLength(%s extensions) = %d, want %d",
					filler.name, measured, specificationExtensionCanonicalMaxBytes)
			}
			if err := validateExtensions(extensions); err != nil {
				t.Fatalf("validateExtensions(%s extensions at the bound) error = %v, want acceptance", filler.name, err)
			}
		})
	}
}

// assertEscapedExtensionMeasurementWouldRefuse is the mutant assertion: the
// accepted at-limit object must be over the bound when measured the way the
// defect measured it, so restoring an escaped measurement reddens this test.
func assertEscapedExtensionMeasurementWouldRefuse(t *testing.T, filler extensionMeasurementFiller, extensions map[string]any) {
	t.Helper()

	escaped := len(mustMarshalExtensions(t, extensions))
	switch {
	case filler.expands && escaped <= specificationExtensionCanonicalMaxBytes:
		t.Fatalf("%s at-limit fixture encodes to %d escaped bytes, which is inside the %d bound; "+
			"the fixture cannot tell a canonical measurement from an escaped one",
			filler.name, escaped, specificationExtensionCanonicalMaxBytes)
	case !filler.expands && escaped != specificationExtensionCanonicalMaxBytes:
		t.Fatalf("%s at-limit fixture encodes to %d escaped bytes, want %d; the control row must be "+
			"encoding neutral", filler.name, escaped, specificationExtensionCanonicalMaxBytes)
	}
}

// oneBytePastDocument appends one encoding-neutral byte inside the extension
// value of an already encoded document, so the production reader is driven with
// a document exactly one canonical byte past the declared bound. Editing the
// encoded document is what reaches the reader at all: the writer refuses the
// over-limit configuration before it can produce one.
func oneBytePastDocument(t *testing.T, document []byte) []byte {
	t.Helper()

	lines := strings.Split(string(document), "\n")
	edited := 0
	for index, line := range lines {
		if !strings.Contains(line, extensionBoundKey) {
			continue
		}
		closing := strings.LastIndexByte(line, '\'')
		if closing < 0 {
			t.Fatalf("extension line %q does not end in a TOML literal string", line)
		}
		lines[index] = line[:closing] + "x" + line[closing:]
		edited++
	}
	if edited != 1 {
		t.Fatalf("edited %d extension value lines, want exactly 1", edited)
	}
	return []byte(strings.Join(lines, "\n"))
}

// extensionsWithCanonicalBytes builds a single-member extensions object whose
// canonical encoding is exactly target bytes.
func extensionsWithCanonicalBytes(t *testing.T, filler extensionMeasurementFiller, target int) map[string]any {
	t.Helper()

	size := target - len(extensionBoundKey) - 7
	value := strings.Repeat(filler.text, size/filler.canonicalWidth) +
		strings.Repeat("x", size%filler.canonicalWidth)
	extensions := map[string]any{extensionBoundKey: value}
	if measured := canonicalExtensionBytes(t, extensions); measured != target {
		t.Fatalf("boundary fixture is %d canonical bytes, want exactly %d", measured, target)
	}
	return extensions
}

// canonicalExtensionBytes measures the fixture through the RFC 8785 entry point
// rather than through the bound helper under test, so a fixture cannot agree
// with a broken measurement and call itself at the limit.
func canonicalExtensionBytes(t *testing.T, extensions map[string]any) int {
	t.Helper()

	canonical, err := canonicaljson.Canonicalize(mustMarshalExtensions(t, extensions))
	if err != nil {
		t.Fatalf("Canonicalize(boundary fixture) error = %v", err)
	}
	return len(canonical)
}

func mustMarshalExtensions(t *testing.T, extensions map[string]any) []byte {
	t.Helper()

	encoded, err := json.Marshal(extensions)
	if err != nil {
		t.Fatalf("json.Marshal(extensions) error = %v", err)
	}
	return encoded
}

func configurationWithExtensions(extensions map[string]any) Configuration {
	value := validCurrentConfiguration()
	value.DirectoryInstallations[0].Extensions = extensions
	return value
}
