package provhost

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestProductionEntriesRefuseLoneSurrogateEscapes proves the Section 1.6
// decoder rule at every production entry that reads a wire body: a lone
// \ud800-style escape is refused before any member is trusted, so the
// silent U+FFFD replacement encoding/json would perform is never
// observed. The bodies below are otherwise uninterpretable on purpose:
// the gate runs before member validation, so refusal must not depend on
// the members being present.
func TestProductionEntriesRefuseLoneSurrogateEscapes(t *testing.T) {
	lone := []byte(`{"note":"ab\ud800cd"}`)
	rows := []struct {
		name string
		err  error
	}{
		{"manifest", DecodeManifest(lone)},
		{"probe", DecodeProbe(lone)},
		{"identity", CheckIdentity(lone, "antigravity")},
		{"identify result", DecodeIdentifyResult(lone, "antigravity")},
		{"quiesce", quiesceErr(lone)},
		{"spawn plan", DecodeSpawnPlan(lone, "codex", ProfileYOLO, scalar.PlatformLinux)},
		{"response envelope", func() error {
			_, err := DecodeResponse(successFrame(t, testRequestID, `{"provider_id":"pi","tag":"a\ud800b"}`), mustUUIDv7(t, testRequestID))
			return err
		}()},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			requireFrameRefusal(t, row.err, "", "lone surrogate escape")
		})
	}
	t.Run("status outcome", func(t *testing.T) {
		body := strings.Replace(string(preparedBody()), `"prepared"`, `"pre\ud800pared"`, 1)
		_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
		requireIntegrityRefusal(t, err, "status body is lone surrogate escape", "")
	})
}

// TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON derives the
// surrogate verdict space from the code-point space and requires the
// local gate (through decodeStrictObject) and canonicaljson to agree
// on every vector. A hand-written vector set shared between two
// implementations is literal-versus-literal: it proves both
// implementations match the same hand list, and a mutant that narrows
// one bound (a low-surrogate ceiling, one hex case range) survives
// every vector that avoids it. Here the inputs are enumerated from
// the space and the verdicts come from the independent
// implementation, so a narrowed bound diverges from the oracle on
// every vector it newly admits, and an over-broad gate diverges on
// every vector it newly refuses. A drift in either direction reddens
// here with the divergent bodies.
//
// Dimensions, each enumerated from the space or the gate's own range
// constants, never from a hand-written boundary literal: every BMP
// code unit as a \uXXXX escape in lower- and uppercase hex; every
// surrogate unit in all 16 hex case permutations; every high
// surrogate paired with the derived low-boundary neighborhood
// (offsets from the gate's low constants) plus one sample from each
// out-of-range second class; sampled units in member-name,
// nested-object, and array positions plus every surrogate unit in
// member-name position; every BMP code point as raw UTF-8 (surrogate
// units in their raw WTF-8 three-byte form, which the UTF-8 gate in
// decodeStrictObject must refuse); enumerated malformed byte
// patterns; and astral raw samples. Every vector is a top-level
// object so both judges can rule on it.
func TestSurrogateGateDerivedSweepAgreesWithCanonicalJSON(t *testing.T) {
	const sweepFloor = 200000
	compared := 0
	divergent := 0
	var examples []string
	compare := func(body []byte) {
		compared++
		_, canonicalErr := canonicaljson.Canonicalize(body)
		_, gateFault := decodeStrictObject(body)
		if (canonicalErr != nil) != (gateFault != nil) {
			divergent++
			if len(examples) < 10 {
				examples = append(examples, fmt.Sprintf("%q canonical.refuse=%v gate.refuse=%v", body, canonicalErr != nil, gateFault != nil))
			}
		}
	}
	escapeBody := func(prefix string, unit int, upper bool) []byte {
		format := `{"a":"` + prefix + `\u%04x"}`
		if upper {
			format = `{"a":"` + prefix + `\u%04X"}`
		}
		return []byte(fmt.Sprintf(format, unit))
	}
	// Every BMP code unit as an escape, both hex cases.
	for unit := 0; unit <= 0xffff; unit++ {
		compare(escapeBody("", unit, false))
		compare(escapeBody("", unit, true))
	}
	// Every surrogate unit in all 16 hex case permutations: a gate
	// that parses one case differently from the other must still
	// match the oracle on each mix.
	for unit := 0xd800; unit <= 0xdfff; unit++ {
		digits := fmt.Sprintf("%04x", unit)
		for mask := 0; mask < 16; mask++ {
			var mixed [4]byte
			for nibble := 0; nibble < 4; nibble++ {
				c := digits[nibble]
				if mask&(1<<nibble) != 0 && c >= 'a' && c <= 'f' {
					c -= 'a' - 'A'
				}
				mixed[nibble] = c
			}
			compare([]byte(`{"a":"\u` + string(mixed[:]) + `"}`))
		}
	}
	// Every high surrogate paired with the low-boundary neighborhood
	// derived from the gate's own range constants: two units on each
	// side of each low bound, both bound endpoints, and the interior
	// midpoint, plus one sample from each out-of-range second class
	// (a high unit and the BMP unit just below the surrogate range).
	// The adjacent units outside the range (lowSurrogateMin-1,
	// lowSurrogateMax+1) are the lone-surrogate vectors that separate
	// the correct bound from an off-by-one narrowing: narrowing either
	// low bound newly admits a swept second and diverges from the
	// oracle there, while the endpoints (valid pairs) kill an
	// over-broad shift. Every value is a gate constant plus a small
	// offset, so no hand-written boundary literal can silently miss a
	// neighbor the way the old five-element list missed 0xdbff and
	// 0xe000.
	pairSeconds := []int{
		lowSurrogateMin - 2, lowSurrogateMin - 1,
		lowSurrogateMin, lowSurrogateMin + 1,
		(lowSurrogateMin + lowSurrogateMax) / 2,
		lowSurrogateMax - 1, lowSurrogateMax,
		lowSurrogateMax + 1, lowSurrogateMax + 2,
		highSurrogateMin, highSurrogateMax,
		highSurrogateMin - 1,
	}
	for high := highSurrogateMin; high <= highSurrogateMax; high++ {
		for _, second := range pairSeconds {
			compare([]byte(fmt.Sprintf(`{"a":"\u%04x\u%04x"}`, high, second)))
		}
	}
	// Sampled units in every string position, plus every surrogate
	// unit in member-name position.
	for unit := 0; unit <= 0xffff; unit += 64 {
		escaped := fmt.Sprintf(`\u%04x`, unit)
		compare([]byte(`{"` + escaped + `":1}`))
		compare([]byte(`{"a":{"b":"` + escaped + `"}}`))
		compare([]byte(`{"a":["` + escaped + `"]}`))
	}
	for unit := 0xd800; unit <= 0xdfff; unit++ {
		compare([]byte(fmt.Sprintf(`{"\u%04x":1}`, unit)))
	}
	// Every BMP code point as raw UTF-8. Surrogate units have no
	// UTF-8 encoding; their raw three-byte WTF-8 form is what the
	// decodeStrictObject UTF-8 gate must refuse.
	for unit := 0; unit <= 0xffff; unit++ {
		body := []byte(`{"a":"`)
		switch {
		case unit >= 0xd800 && unit <= 0xdfff:
			body = append(body, 0xe0|byte(unit>>12), 0x80|byte(unit>>6)&0x3f, 0x80|byte(unit)&0x3f)
		case unit == '"' || unit == '\\':
			body = append(body, '\\', byte(unit))
		default:
			body = append(body, string(rune(unit))...)
		}
		body = append(body, `"}`...)
		compare(body)
	}
	// Enumerated malformed byte patterns inside a string.
	for _, pattern := range [][]byte{
		{0xff}, {0xfe}, {0x80},
		{0xc0, 0xaf}, {0xe0, 0x80, 0xaf}, {0xf0, 0x80, 0x80, 0xaf},
		{0xf5, 0x80, 0x80, 0x80}, {0xe2, 0x82}, {0xf0, 0x9f, 0x98},
		{0xed, 0xa0, 0x80}, {0xed, 0xbf, 0xbf},
	} {
		body := append([]byte(`{"a":"`), pattern...)
		body = append(body, `"}`...)
		compare(body)
	}
	// Astral raw samples, valid UTF-8 both judges accept.
	for _, point := range []rune{0x10000, 0x1f600, 0x10ffff} {
		compare([]byte(`{"a":"` + string(point) + `"}`))
	}
	if compared < sweepFloor {
		t.Fatalf("derived sweep compared %d vectors, below the %d floor; the enumeration is short, not the gate", compared, sweepFloor)
	}
	if divergent != 0 {
		t.Fatalf("gate and canonicaljson diverge on %d of %d vectors, first %d:\n  %s", divergent, compared, len(examples), strings.Join(examples, "\n  "))
	}
	t.Logf("surrogate sweep: gate agrees with canonicaljson on %d of %d vectors", compared, compared)
}
