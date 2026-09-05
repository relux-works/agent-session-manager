package provhost

// This file is the local lone-surrogate gate for the Section 1.6 wire
// bodies this package decodes. Pinned specification Section 1.6 requires
// decoders to reject lone surrogate code points before canonicalization,
// and the repository already enforces that rule in
// internal/canonicaljson (validateSurrogateEscapes, called from
// decodeStrict). This package decodes with its own strict object reader
// (decodeStrictObject in protocol.go) because it must keep member-level
// fault attribution (duplicate member names, the offending member) that
// the shared decoder does not report, and encoding/json silently replaces
// a lone \ud800-style escape with U+FFFD instead of failing. So the gate
// lives here, scans the raw frame bytes before decoding, and refuses the
// same class the shared package refuses.
//
// The gate is deliberately a copy, not an import: internal/canonicaljson
// is byte-identical shared baseline text that two unlanded stories must
// not edit at once, so neither story may add an export there. The
// surrogate-vector sweep in surrogate_test.go drives both this gate
// (through decodeStrictObject) and canonicaljson.Canonicalize over a
// derived enumeration of the code-point space, so the copies cannot
// drift silently: a divergence reddens there rather than passing
// unnoticed. The sweep judges verdicts against the independent
// implementation, never against a shared hand-written expectation.
//
// The gate decides the surrogate-escape question only. Malformed
// escapes and unterminated strings are not its verdict: they fall
// through to the JSON-syntax arms that already own them. Non-UTF-8
// bytes never reach it: decodeStrictObject refuses them first,
// because a raw WTF-8 surrogate encoding would otherwise slip past
// this escape scanner and be silently replaced with U+FFFD by
// encoding/json. A lone high surrogate is a high unit not immediately
// followed by a \uDC00–\uDFFF escape; a lone low surrogate is a low
// unit in any other position. Both are refused before any member is
// read.

// The gate's own range constants: the high and low surrogate halves
// of the Section 1.6 verdict space. The
// surrogate-vector sweep in surrogate_test.go derives its pair-second
// dimension from these constants arithmetically (bounds plus and minus
// small offsets), so the boundary neighborhood is enumerated from the
// gate rather than from a hand-written literal: shifting a bound here
// moves the swept neighborhood with it instead of silently uncovering
// it. Verdicts still come from the independent implementation, never
// from these constants, so a wrong bound reddens against the oracle.
const (
	highSurrogateMin = 0xd800
	highSurrogateMax = 0xdbff
	lowSurrogateMin  = 0xdc00
	lowSurrogateMax  = 0xdfff
)

// hasLoneSurrogateEscape reports whether raw JSON bytes carry a lone
// surrogate escape inside any string. The string walk mirrors the shared
// package's walker (backslash handling included: a \\ escape consumes
// both bytes, so the text "\\ud800" is backslash-plus-text, not an
// escape), but only the surrogate verdict is returned.
func hasLoneSurrogateEscape(input []byte) bool {
	for index := 0; index < len(input); index++ {
		if input[index] != '"' {
			continue
		}
		for index++; index < len(input); index++ {
			if input[index] == '"' {
				goto stringComplete
			}
			if input[index] != '\\' {
				continue
			}
			index++
			if index >= len(input) || input[index] != 'u' {
				continue
			}
			unit, end, ok := readHexUnit(input, index-1)
			if !ok {
				// Not a well-formed \uXXXX escape: the
				// JSON-syntax arms own it, not this gate.
				continue
			}
			index = end - 1
			if unit >= highSurrogateMin && unit <= highSurrogateMax {
				low, lowEnd, ok := readHexUnit(input, end)
				if !ok || low < lowSurrogateMin || low > lowSurrogateMax {
					return true
				}
				index = lowEnd - 1
			} else if unit >= lowSurrogateMin && unit <= lowSurrogateMax {
				return true
			}
		}
		// The string never closed: malformed JSON the syntax arms
		// refuse, so the gate claims no surrogate verdict here.
		return false
	stringComplete:
	}
	return false
}

// readHexUnit parses one \uXXXX escape at input[start]. It reports false
// when the bytes are not a well-formed escape; hex digits parse in
// either case, matching the shared package's ParseUint base-16 read.
func readHexUnit(input []byte, start int) (uint16, int, bool) {
	if start < 0 || start+6 > len(input) || input[start] != '\\' || input[start+1] != 'u' {
		return 0, start, false
	}
	var value uint16
	for _, b := range input[start+2 : start+6] {
		var digit byte
		if b >= '0' && b <= '9' {
			digit = b - '0'
		} else if b >= 'a' && b <= 'f' {
			digit = b - 'a' + 10
		} else if b >= 'A' && b <= 'F' {
			digit = b - 'A' + 10
		} else {
			return 0, start, false
		}
		value = value*16 + uint16(digit)
	}
	return value, start + 6, true
}
