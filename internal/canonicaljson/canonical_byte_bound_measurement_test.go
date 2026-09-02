package canonicaljson

import (
	"fmt"
	"strings"
	"testing"
)

// This file is the measurement gate for the declared byte bounds.
//
// The failure it exists to prevent: a bound declared in bytes was measured on
// Go's HTML-escaped JSON encoding rather than on canonical bytes. encoding/json
// rewrites each of the single bytes <, > and & into a six-character escape and
// U+2028 and U+2029 from three bytes into six, so a conforming document built
// from those characters measured up to six times its canonical size and was
// refused. The bound in force was a property of encoding/json, not the pinned
// limit, and the two call sites in this repository disagreed about what a byte
// is. Nothing reddened, because every fixture was made of characters whose two
// encodings happen to have the same length.
//
// The cases below are therefore built from characters whose two encodings
// differ, and each one asserts that the escaped measurement of the accepted
// document is over the limit. That assertion is what makes a mutant restoring
// the escaped measurement fail here rather than pass.

// specificationDeclaredByteMaximum is the byte bound that
// relux-works/agent-session-manager-spec@v0.5.0 Section 6 and Section 10.1 fix
// for the Session Record Launch Plan argv and for an extensions object. It is
// written here as the specification literal, independently of the production
// constant, so widening the implementation cannot widen this proof with it.
const specificationDeclaredByteMaximum = 65_536

// argvElementMaxBytes is the per-element byte bound the Launch Plan declares.
// The fixtures stay inside it so an over-limit case is refused by the encoded
// size bound under test rather than by the element bound.
const argvElementMaxBytes = 4_096

// canonicalElementOverhead is the canonical cost of one array element or one
// object member beyond its own content: two quote bytes plus the separator or
// bracket byte that precedes it.
const canonicalElementOverhead = 3

// measurementFiller is one character used to fill a boundary fixture, together
// with the number of canonical bytes it occupies. expands records whether Go's
// default JSON encoding is longer than the canonical form for that character,
// which is the difference this gate exists to detect.
type measurementFiller struct {
	name           string
	text           string
	canonicalWidth int
	expands        bool
}

func measurementFillers() []measurementFiller {
	return []measurementFiller{
		{name: "less than", text: "<", canonicalWidth: 1, expands: true},
		{name: "greater than", text: ">", canonicalWidth: 1, expands: true},
		{name: "ampersand", text: "&", canonicalWidth: 1, expands: true},
		{name: "line separator", text: string(rune(0x2028)), canonicalWidth: 3, expands: true},
		{name: "paragraph separator", text: string(rune(0x2029)), canonicalWidth: 3, expands: true},
		// The control row: an encoding-neutral character. It shares the whole
		// fixture machinery with the rows above, so if it passed while they
		// failed, the fixtures would not be the reason.
		{name: "encoding neutral ASCII", text: "x", canonicalWidth: 1, expands: false},
	}
}

// TestMeasurementFillerWidthsAreTheCanonicalOnes pins the widths the fixture
// arithmetic below relies on, so a wrong width cannot silently produce a
// fixture that misses the boundary it claims to sit on.
func TestMeasurementFillerWidthsAreTheCanonicalOnes(t *testing.T) {
	t.Parallel()

	for _, filler := range measurementFillers() {
		t.Run(filler.name, func(t *testing.T) {
			t.Parallel()

			canonical, err := Canonicalize(mustJSON(t, []any{filler.text}))
			if err != nil {
				t.Fatalf("Canonicalize(single %s element) error = %v", filler.name, err)
			}
			want := 1 + filler.canonicalWidth + canonicalElementOverhead
			if len(canonical) != want {
				t.Fatalf("canonical bytes for a single %s element = %d, want %d", filler.name, len(canonical), want)
			}
			escaped := len(mustJSON(t, []any{filler.text}))
			if expands := escaped > len(canonical); expands != filler.expands {
				t.Fatalf("%s escaped encoding is %d bytes and canonical is %d; expands = %t, want %t",
					filler.name, escaped, len(canonical), expands, filler.expands)
			}
		})
	}
}

// TestLaunchPlanArgvByteBoundIsMeasuredOnCanonicalBytes drives both public
// identity entries with a Session Record whose Launch Plan argv sits exactly on
// the declared bound, and then one canonical byte past it.
func TestLaunchPlanArgvByteBoundIsMeasuredOnCanonicalBytes(t *testing.T) {
	t.Parallel()

	for _, filler := range measurementFillers() {
		t.Run(filler.name, func(t *testing.T) {
			t.Parallel()

			atLimit := argvWithCanonicalBytes(t, filler, specificationDeclaredByteMaximum)
			assertCanonicalBytes(t, atLimit, specificationDeclaredByteMaximum)
			assertEscapedMeasurementWouldRefuse(t, filler, atLimit)
			assertIdentityEntriesAcceptShape(t, mustJSON(t, sessionRecordWithArgv(atLimit)), SelfRecordID)

			overLimit := argvWithCanonicalBytes(t, filler, specificationDeclaredByteMaximum+1)
			assertCanonicalBytes(t, overLimit, specificationDeclaredByteMaximum+1)
			input := mustJSON(t, sessionRecordWithArgv(overLimit))
			assertIdentityEntriesRefuseShape(t, input, SelfRecordID)
			assertRefusedByCanonicalByteBound(t, input, "canonical Session Record Launch Plan argv is")
		})
	}
}

// TestExtensionsObjectByteBoundIsMeasuredOnCanonicalBytes is the same proof for
// the other declared byte bound in this package.
func TestExtensionsObjectByteBoundIsMeasuredOnCanonicalBytes(t *testing.T) {
	t.Parallel()

	for _, filler := range measurementFillers() {
		t.Run(filler.name, func(t *testing.T) {
			t.Parallel()

			atLimit := extensionsWithCanonicalBytes(t, filler, specificationDeclaredByteMaximum)
			assertCanonicalBytes(t, atLimit, specificationDeclaredByteMaximum)
			assertEscapedMeasurementWouldRefuse(t, filler, atLimit)
			assertIdentityEntriesAcceptShape(t, mustJSON(t, genericExtensionIdentityObject(atLimit)), SelfRecordID)

			overLimit := extensionsWithCanonicalBytes(t, filler, specificationDeclaredByteMaximum+1)
			assertCanonicalBytes(t, overLimit, specificationDeclaredByteMaximum+1)
			input := mustJSON(t, genericExtensionIdentityObject(overLimit))
			assertIdentityEntriesRefuseShape(t, input, SelfRecordID)
			assertRefusedByCanonicalByteBound(t, input, "canonical extensions object is")
		})
	}
}

// TestCanonicalByteLengthIgnoresHTMLAndLineSeparatorEscaping pins the shared
// helper itself: it must report canonical bytes for a value whose escaped
// encoding is longer.
func TestCanonicalByteLengthIgnoresHTMLAndLineSeparatorEscaping(t *testing.T) {
	t.Parallel()

	for _, filler := range measurementFillers() {
		t.Run(filler.name, func(t *testing.T) {
			t.Parallel()

			value := []any{strings.Repeat(filler.text, 1_000)}
			want := 1 + filler.canonicalWidth*1_000 + canonicalElementOverhead
			measured, err := CanonicalByteLength(value)
			if err != nil {
				t.Fatalf("CanonicalByteLength(%s) error = %v", filler.name, err)
			}
			if measured != want {
				t.Fatalf("CanonicalByteLength(%s) = %d, want %d canonical bytes", filler.name, measured, want)
			}
			if escaped := len(mustJSON(t, value)); filler.expands && escaped <= measured {
				t.Fatalf("%s fixture does not separate the two measurements: escaped = %d, canonical = %d",
					filler.name, escaped, measured)
			}
		})
	}
}

// assertEscapedMeasurementWouldRefuse is the mutant assertion. It requires the
// accepted at-limit document to be over the declared bound when measured the
// way the defect measured it, so restoring an escaped measurement at either
// call site turns this accepted document into a refused one.
func assertEscapedMeasurementWouldRefuse(t *testing.T, filler measurementFiller, value any) {
	t.Helper()

	escaped := len(mustJSON(t, value))
	switch {
	case filler.expands && escaped <= specificationDeclaredByteMaximum:
		t.Fatalf("%s at-limit fixture encodes to %d escaped bytes, which is inside the %d bound; "+
			"the fixture cannot tell a canonical measurement from an escaped one",
			filler.name, escaped, specificationDeclaredByteMaximum)
	case !filler.expands && escaped != specificationDeclaredByteMaximum:
		t.Fatalf("%s at-limit fixture encodes to %d escaped bytes, want %d; the control row must be "+
			"encoding neutral", filler.name, escaped, specificationDeclaredByteMaximum)
	}
}

// assertRefusedByCanonicalByteBound requires the one-past document to be
// refused by the byte bound itself, naming the canonical size it measured, not
// by some other gate that happens to fire on the same fixture.
func assertRefusedByCanonicalByteBound(t *testing.T, input []byte, prefix string) {
	t.Helper()

	want := fmt.Sprintf("%s %d bytes, maximum is %d",
		prefix, specificationDeclaredByteMaximum+1, specificationDeclaredByteMaximum)
	_, _, err := CalculateObjectIdentity(input)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("CalculateObjectIdentity(one past the bound) error = %v, want a refusal containing %q", err, want)
	}
}

// assertCanonicalBytes measures the fixture through the RFC 8785 entry point
// rather than through the bound helper, so a fixture cannot agree with a broken
// measurement and call itself at the limit.
func assertCanonicalBytes(t *testing.T, value any, want int) {
	t.Helper()

	canonical, err := Canonicalize(mustJSON(t, value))
	if err != nil {
		t.Fatalf("Canonicalize(boundary fixture) error = %v", err)
	}
	if len(canonical) != want {
		t.Fatalf("boundary fixture is %d canonical bytes, want exactly %d", len(canonical), want)
	}
}

// argvWithCanonicalBytes builds an argv whose canonical encoding is exactly
// target bytes and whose elements all stay inside the per-element bound.
//
// A canonical array of strings is "[" plus the quoted elements joined by
// commas plus "]", so its length is 1 + the sum over elements of the element's
// canonical content plus canonicalElementOverhead.
func argvWithCanonicalBytes(t *testing.T, filler measurementFiller, target int) []any {
	t.Helper()

	// Two spare bytes per element leave room for the encoding-neutral padding a
	// last element may need to land on an exact byte count.
	fillerCount := (argvElementMaxBytes - 2) / len(filler.text)
	fullElement := filler.canonicalWidth*fillerCount + canonicalElementOverhead

	argv := make([]any, 0, 128)
	remaining := target - 1
	for remaining > fullElement+canonicalElementOverhead {
		argv = append(argv, strings.Repeat(filler.text, fillerCount))
		remaining -= fullElement
	}
	argv = append(argv, fillerString(filler, remaining-canonicalElementOverhead))

	for index, element := range argv {
		if size := len(element.(string)); size < 1 || size > argvElementMaxBytes {
			t.Fatalf("argv[%d] fixture is %d bytes, outside the declared 1..%d element bound", index, size, argvElementMaxBytes)
		}
	}
	if len(argv) < 1 || len(argv) > 128 {
		t.Fatalf("argv fixture has %d elements, outside the declared 1..128 bound", len(argv))
	}
	return argv
}

// extensionsWithCanonicalBytes builds a single-member extensions object whose
// canonical encoding is exactly target bytes. A canonical one-member object is
// "{" plus the quoted key, ":", the quoted value and "}", which is the key
// length plus the value length plus seven.
func extensionsWithCanonicalBytes(t *testing.T, filler measurementFiller, target int) map[string]any {
	t.Helper()

	const key = "works.relux.bytes"
	return map[string]any{key: fillerString(filler, target-len(key)-7)}
}

// fillerString returns a string whose canonical encoding is exactly size bytes,
// built from filler and padded with encoding-neutral ASCII when the filler's
// width does not divide size.
func fillerString(filler measurementFiller, size int) string {
	return strings.Repeat(filler.text, size/filler.canonicalWidth) +
		strings.Repeat("x", size%filler.canonicalWidth)
}
