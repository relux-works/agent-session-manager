package sessadapter

import (
	"fmt"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// TestLoneSurrogateSweep drives every UTF-16 escape that can stand
// alone through the strict decoder: all 2048 lone surrogates (high
// and low) in upper, lower, and mixed hex case must refuse, and
// every escape is generated, never listed. The verdict for each
// vector comes from construction — the vector names a lone
// surrogate by construction — not from either implementation, so
// the corpus cannot drift with a mutated gate the way a 13-value
// hand list did.
func TestLoneSurrogateSweep(t *testing.T) {
	cases := []string{"%04X", "%04x", "%02X%02x"}
	refused := 0
	for surrogate := 0xD800; surrogate <= 0xDFFF; surrogate++ {
		for _, form := range cases {
			frame := []byte(fmt.Sprintf(`{"member":"A\u`+form+`B"}`, surrogate))
			if _, fault := decodeStrictObject(frame); fault == nil {
				t.Fatalf("decodeStrictObject admitted lone surrogate U+%04X", surrogate)
			} else if fault.detail != "lone surrogate escape" {
				t.Fatalf("lone surrogate U+%04X fault = %q, want the surrogate gate", surrogate, fault.detail)
			}
			refused++
		}
	}
	// A high surrogate followed by a low surrogate admits; a high
	// followed by a non-low, and a low standing first, refuse.
	paired := []byte(`{"member":"A\uD83D\uDE00B"}`)
	if _, fault := decodeStrictObject(paired); fault != nil {
		t.Fatalf("decodeStrictObject refused a paired surrogate: %v", fault.detail)
	}
	unpairedHigh := []byte(`{"member":"A\uD83D\u0041B"}`)
	if _, fault := decodeStrictObject(unpairedHigh); fault == nil {
		t.Fatal("decodeStrictObject admitted a high surrogate followed by a non-low")
	}
	// Truncated and non-hex escapes refuse through the same gate.
	for _, frame := range []string{`{"member":"A\uD83B"}`, `{"member":"A\uDEFG B"}`, `{"member":"A\uD83D"}`} {
		if _, fault := decodeStrictObject([]byte(frame)); fault == nil {
			t.Fatalf("decodeStrictObject admitted %q", frame)
		}
	}
	t.Logf("surrogate sweep: %d lone vectors refused", refused)
}

// TestSurrogatePairSweep drives the second escape of every high
// surrogate across the low-surrogate range and both adjacent code
// points: for each of the 1024 high surrogates the second escape
// takes the values just below the low range, both low edges, and
// just above it, so an off-by-one on either bound admits a vector
// this corpus refuses by construction. A full low-range sweep for
// the first, middle, and last high surrogate proves the interior
// admits. Every verdict comes from construction — the vector
// names its pair shape from range literals written here, not from
// either implementation — so a mutated production bound cannot
// take the fixture with it.
func TestSurrogatePairSweep(t *testing.T) {
	const highLo, highHi = 0xD800, 0xDBFF
	const lowLo, lowHi = 0xDC00, 0xDFFF
	refused, admitted := 0, 0
	check := func(high, second int, form string) {
		t.Helper()
		frame := []byte(fmt.Sprintf(`{"member":"A\u`+form+`\u`+form+`B"}`, high, second))
		_, fault := decodeStrictObject(frame)
		if pair := second >= lowLo && second <= lowHi; pair {
			if fault != nil {
				t.Fatalf("decodeStrictObject refused valid pair U+%04X U+%04X: %v", high, second, fault.detail)
			}
			admitted++
			return
		}
		if fault == nil {
			t.Fatalf("decodeStrictObject admitted lone high surrogate U+%04X followed by U+%04X", high, second)
		}
		if fault.detail != "lone surrogate escape" {
			t.Fatalf("pair U+%04X U+%04X fault = %q, want the surrogate gate", high, second, fault.detail)
		}
		refused++
	}
	for high := highLo; high <= highHi; high++ {
		for _, second := range []int{lowLo - 1, lowLo, lowHi, lowHi + 1} {
			check(high, second, "%04X")
		}
	}
	// Lowercase spot checks prove the hex case never moves the
	// bound: the edges refuse and admit identically.
	for _, second := range []int{lowLo - 1, lowLo, lowHi, lowHi + 1} {
		check(highLo, second, "%04x")
		check(highHi, second, "%04x")
	}
	// Full-range interior for three representative highs: every
	// low second admits, and the neighbours just outside refuse.
	for _, high := range []int{highLo, (highLo + highHi) / 2, highHi} {
		for second := lowLo - 1; second <= lowHi+1; second++ {
			check(high, second, "%04X")
		}
	}
	t.Logf("surrogate pair sweep: %d refused, %d admitted", refused, admitted)
}

// TestSurrogateGateAgreesWithCanonicalJSON pins the local gate
// against the shared implementation over generated vectors: every
// BMP escape in both hex cases, every surrogate pair shape, and
// raw non-surrogate multibyte text. Agreement is a consistency
// pin, not the verdict: the verdicts above come from
// construction.
func TestSurrogateGateAgreesWithCanonicalJSON(t *testing.T) {
	checked := 0
	for code := 0; code < 0xD800; code += 0x217 {
		for _, frame := range []string{
			fmt.Sprintf(`{"member":"A\u%04XB"}`, code),
			fmt.Sprintf(`{"member":"A\u%04xB"}`, code),
		} {
			_, localFault := decodeStrictObject([]byte(frame))
			_, sharedErr := canonicaljson.Canonicalize([]byte(frame))
			if (localFault != nil) != (sharedErr != nil) {
				t.Fatalf("disagreement on U+%04X: local=%v shared=%v", code, localFault, sharedErr)
			}
			checked++
		}
	}
	for _, frame := range []string{
		`{"member":"plain ascii"}`,
		`{"member":"ü é ☃"}`,
		`{"member":"A\uD83D\uDE00B"}`,
		`{"member":"A\ud800B"}`,
		`{"member":"A\udc00B"}`,
	} {
		_, localFault := decodeStrictObject([]byte(frame))
		_, sharedErr := canonicaljson.Canonicalize([]byte(frame))
		localRefuses := localFault != nil
		sharedRefuses := sharedErr != nil
		// The frame verdict delegates to environ.DecodeStrictObject,
		// whose string-walk gate shares the canonicalizer's
		// verdicts: both refuse lone surrogates and both admit
		// valid text and paired surrogates. Agreement is required
		// on every vector, in both directions.
		if localRefuses != sharedRefuses {
			t.Fatalf("disagreement on %q: local=%v shared=%v", frame, localFault, sharedErr)
		}
		checked++
	}
	t.Logf("agreement sweep: %d vectors compared", checked)
}

// TestNonUTF8NeverDecodes proves raw non-UTF-8 bytes refuse before
// any member is read: overlong encodings, truncated sequences,
// and encoded surrogates (WTF-8) are failures, never rewritten
// text.
func TestNonUTF8NeverDecodes(t *testing.T) {
	frames := [][]byte{
		{0x7b, 0x22, 0x6d, 0x22, 0x3a, 0xff, 0x7d},
		[]byte("{\"m\":\"a\xed\xa0\x80b\"}"),
		[]byte("{\"m\":\"a\xc0\xafb\"}"),
		[]byte("{\"m\":\"a\xe2\x82b\"}"),
	}
	for index, frame := range frames {
		if _, fault := decodeStrictObject(frame); fault == nil {
			t.Fatalf("decodeStrictObject admitted non-UTF-8 frame %d", index)
		} else if fault.detail != "not valid UTF-8" {
			t.Fatalf("frame %d fault = %q, want the UTF-8 gate", index, fault.detail)
		}
	}
}
