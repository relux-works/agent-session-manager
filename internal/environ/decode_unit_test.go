package environ

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file drives the environ entries no facade reaches
// directly: the uint53 bound checker (with its digit-literal
// ladder), the sorted-unique string checker in both of its
// halves, and the frame-fault rendering. Coverage here is
// behavioral (refusal or value at the edges), never line-fill:
// every row would fail against a weakened gate.

// TestCheckUint53BoundsEdges drives CheckUint53Bounds through
// the production entry: accepts decode to the bound values, and
// every refusal shape (non-number, fraction, exponent, sign,
// out-of-range magnitude, bound edges) refuses.
func TestCheckUint53BoundsEdges(t *testing.T) {
	accept := func(literal string, minimum, maximum uint64, want uint64) {
		t.Helper()
		got, ok := CheckUint53Bounds(json.RawMessage(literal), minimum, maximum)
		if !ok {
			t.Fatalf("CheckUint53Bounds(%s, %d, %d) refused, want %d", literal, minimum, maximum, want)
		}
		if got != want {
			t.Fatalf("CheckUint53Bounds(%s, %d, %d) = %d, want %d", literal, minimum, maximum, got, want)
		}
	}
	refuse := func(literal string, minimum, maximum uint64) {
		t.Helper()
		if got, ok := CheckUint53Bounds(json.RawMessage(literal), minimum, maximum); ok {
			t.Fatalf("CheckUint53Bounds(%s, %d, %d) admitted %d, want refusal", literal, minimum, maximum, got)
		}
	}
	accept("0", 0, 100, 0)
	accept("100", 0, 100, 100)
	accept("9007199254740991", 0, 9007199254740991, 9007199254740991)
	accept("5", 5, 10, 5)
	accept("10", 5, 10, 10)
	refuse("4", 5, 10)
	refuse("101", 0, 100)
	refuse("9007199254740992", 0, 18446744073709551615)
	refuse("9007199254740992", 0, 9007199254740991)
	refuse("1.0", 0, 100)
	refuse("1e3", 0, 10000)
	refuse("-1", 0, 100)
	refuse("+1", 0, 100)
	refuse(`"42"`, 0, 100)
	refuse("null", 0, 100)
	refuse("true", 0, 100)
	refuse("", 0, 100)
	refuse("12a", 0, 100)
	refuse("18446744073709551616", 0, 18446744073709551615)
}

// TestCheckSortedUniqueStringsHalves drives
// CheckSortedUniqueStrings through the production entry with
// the conjoined halves separated: an unsorted-but-unique vector
// refuses (ordering), a sorted-but-duplicated vector refuses
// (uniqueness), and a sorted unique vector admits with its
// values. A check enforcing only one half cannot pass both
// refusal rows.
func TestCheckSortedUniqueStringsHalves(t *testing.T) {
	accept := func(t *testing.T, literal string) []string {
		t.Helper()
		values, ok := CheckSortedUniqueStrings(json.RawMessage(literal), 1, 128, 0, 64)
		if !ok {
			t.Fatalf("CheckSortedUniqueStrings(%s) refused, want accept", literal)
		}
		return values
	}
	refuse := func(t *testing.T, literal string) {
		t.Helper()
		if values, ok := CheckSortedUniqueStrings(json.RawMessage(literal), 1, 128, 0, 64); ok {
			t.Fatalf("CheckSortedUniqueStrings(%s) admitted %q, want refusal", literal, values)
		}
	}
	t.Run("sorted unique admits with values", func(t *testing.T) {
		values := accept(t, `["a","b","c"]`)
		if len(values) != 3 || values[0] != "a" || values[1] != "b" || values[2] != "c" {
			t.Fatalf("values = %q, want [a b c]", values)
		}
	})
	t.Run("unsorted unique refuses", func(t *testing.T) {
		refuse(t, `["b","a"]`)
	})
	t.Run("sorted duplicated refuses", func(t *testing.T) {
		refuse(t, `["a","a"]`)
	})
	t.Run("non-array refuses", func(t *testing.T) {
		refuse(t, `null`)
	})
	t.Run("overlong element refuses", func(t *testing.T) {
		refuse(t, `["`+strings.Repeat("v", 129)+`"]`)
	})
	t.Run("over count refuses", func(t *testing.T) {
		if _, ok := CheckSortedUniqueStrings(json.RawMessage(`["a"]`), 1, 128, 2, 64); ok {
			t.Fatal("admitted a vector below the count floor")
		}
		if _, ok := CheckSortedUniqueStrings(json.RawMessage(`["a","b"]`), 1, 128, 0, 1); ok {
			t.Fatal("admitted a vector above the count ceiling")
		}
	})
}

// TestParseUint53LiteralRefusesNonDigits drives the digit
// ladder directly: it is unreachable with hostile input through
// CheckUint53Bounds (the trailing-data arm refuses first), so
// these rows call it with hostile literals itself. A ladder
// that admitted one digit class would pass the bound battery
// and fail exactly here.
func TestParseUint53LiteralRefusesNonDigits(t *testing.T) {
	for _, literal := range []string{"1a2", "a", "12 ", " 12", "+12", "0x10", ""} {
		if value, ok := parseUint53Literal(literal); ok {
			t.Fatalf("parseUint53Literal(%q) admitted %d, want refusal", literal, value)
		}
	}
	if value, ok := parseUint53Literal("1024"); !ok || value != 1024 {
		t.Fatalf("parseUint53Literal(1024) = (%d, %v), want (1024, true)", value, ok)
	}
}

// TestDecodeArrayRefusesTrailingData drives the array reader
// directly: member slices arriving through DecodeStrictObject
// can never carry trailing data, so no facade entry reaches
// this arm with hostile input. A reader that admitted trailing
// data would pass every entry battery and fail exactly here.
func TestDecodeArrayRefusesTrailingData(t *testing.T) {
	if _, ok := decodeArray(json.RawMessage(`["a"] trailing`)); ok {
		t.Fatal("decodeArray admitted trailing data after the array")
	}
	if _, ok := decodeArray(json.RawMessage(`["a"] ["b"]`)); ok {
		t.Fatal("decodeArray admitted a second value after the array")
	}
	elements, ok := decodeArray(json.RawMessage(`["a", "b"]`))
	if !ok || len(elements) != 2 {
		t.Fatalf("decodeArray refused a clean array: (%v, %v)", elements, ok)
	}
}

// TestDigestBridgeRefusesEmptyDownstream pins the property the
// R_digestnonstr survivor bound rests on: scalar.ParseDigest
// refuses the empty string, so removing CheckDigest's type arm
// changes no entry verdict because the zero value the arm would
// have admitted is refused downstream. If ParseDigest("") ever
// started admitting, both the removed arm and the live gate
// would open silently — this test reddens first.
func TestDigestBridgeRefusesEmptyDownstream(t *testing.T) {
	if _, err := scalar.ParseDigest(""); err == nil {
		t.Fatal(`scalar.ParseDigest("") admitted, want refusal: the R_digestnonstr bound no longer holds`)
	}
	if _, ok := CheckDigest(json.RawMessage(`""`)); ok {
		t.Fatal(`CheckDigest("") admitted, want refusal at the live gate`)
	}
	if _, ok := CheckDigest(json.RawMessage(`null`)); ok {
		t.Fatal(`CheckDigest(null) admitted, want refusal at the type arm`)
	}
}

// TestFaultErrorRendersMember drives Fault.Error through the
// production type: memberless faults name only the detail, and
// member faults name both. The rendering is what the facade
// wrappers translate, so both branches are pinned here.
func TestFaultErrorRendersMember(t *testing.T) {
	bare := (&Fault{Detail: FaultLoneSurrogate}).Error()
	if !strings.Contains(bare, FaultLoneSurrogate) {
		t.Fatalf("memberless fault = %q, want the detail", bare)
	}
	if strings.Contains(bare, "in member") {
		t.Fatalf("memberless fault = %q, must not name a member", bare)
	}
	withMember := (&Fault{Detail: FaultDuplicate, Member: "v"}).Error()
	if !strings.Contains(withMember, FaultDuplicate) || !strings.Contains(withMember, "v") {
		t.Fatalf("member fault = %q, want the detail and the member", withMember)
	}
}
