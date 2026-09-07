package dirnode

import (
	"encoding/json"
	"testing"
)

// This file pins the rawUint53 trailing-data texture this leaf
// hardened onto the environ shape. Decode reads one value and
// stops, so without the trailing check a hostile slice like `12a`
// reads as 12 (and `12 13` reads as 12 with a silent second
// value). Member slices arriving through decodeStrictObject can
// never carry trailing data — the frame gate strips them first —
// so these rows drive the helper directly, the way
// decode_unit_test.go drives the environ entries no facade
// reaches. Both hostile shapes are separate rows because they
// fail for different reasons: `12a` errors at the trailing token
// while `12 13` yields one, so a gate that refuses only error
// trailing (the narrowing mutant) admits the second and fails
// exactly its row.
func TestRawUint53RefusesTrailingData(t *testing.T) {
	refuse := func(literal string) {
		t.Helper()
		if got, ok := rawUint53(json.RawMessage(literal)); ok {
			t.Fatalf("rawUint53(%q) admitted %d, want refusal", literal, got)
		}
	}
	accept := func(literal string, want uint64) {
		t.Helper()
		got, ok := rawUint53(json.RawMessage(literal))
		if !ok {
			t.Fatalf("rawUint53(%q) refused, want %d", literal, want)
		}
		if got != want {
			t.Fatalf("rawUint53(%q) = %d, want %d", literal, got, want)
		}
	}
	refuse("12a")
	refuse("12 13")
	refuse("0x10")
	accept("12", 12)
	accept("12 ", 12)
	accept(" 12", 12)
	accept("9007199254740991", 9007199254740991)
	refuse("9007199254740992")
	if _, ok := checkUint53Bounds(json.RawMessage("12a"), 0, 100); ok {
		t.Fatal("checkUint53Bounds admitted `12a`, want refusal")
	}
}
