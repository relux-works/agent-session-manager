package environ_test

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file is the scalar lone-surrogate agreement battery: the
// third spelling of the gate. scalar.decodeJSONString validates a
// single JSON string scalar (the data IS one JSON string), while
// environ.DecodeStrictObject validates a full object frame, so
// the two cannot share an implementation — delegating either way
// would be the wrong unit — and this battery is what keeps the
// retained copy honest in both directions.
//
// Both sides are driven through production entries on one shared
// corpus: the environ side wraps the literal as {"v":LITERAL} and
// asserts the frame fault, while the scalar side drives
// scalar.DecodeClosedEnumJSON and asserts the scalar refusal
// text. A verdict flip in EITHER direction reddens its row.
//
// The gate verdict is isolated from the content rules behind it:
// scalar rows that admit at the gate but refuse at the closed
// vocabulary arm assert the membership text with the gate phrase
// asserted ABSENT, proving travel past the gate (the same
// absent-phrase technique the frame battery uses). A gate that
// refused everything would fail those rows; a gate that admitted
// a lone escape would fail the refuse rows. Malformed and
// truncated escapes fall through to the syntax arms on both
// sides and assert those arms explicitly, so a gate that claimed
// them could not pass.

// scalarRow is one agreement vector: the JSON string literal, the
// environ frame verdict ("" admits), and the scalar verdict.
type scalarRow struct {
	name string
	// inner is the raw JSON string content between the quotes:
	// backslashes here are JSON source backslashes.
	inner string
	// environRefuse names the frame fault, or "" when the frame
	// gate admits.
	environRefuse string
	// scalarRefuse names the scalar refusal text the verdict
	// must contain, or "" when the scalar entry admits. A
	// membership refusal proves the gate admitted: the
	// vocabulary arm runs behind it.
	scalarRefuse string
	// scalarAbsent names a phrase the scalar error must not
	// contain, pinning travel past the named gate.
	scalarAbsent string
}

const (
	scalarGateText   = "lone Unicode surrogate"
	scalarSyntaxText = "must be a JSON string"
	scalarMemberText = "is not a member of the negotiated vocabulary"
)

// scalarCorpus enumerates the gate shape: bare highs and lows,
// mispairs in both orders, the admitting pair, backslash-run
// parity (even runs quote, odd runs escape), the composed
// even-run-plus-real-escape shapes that hid the original admit
// hole, the malformed fall-throughs, and multibyte text.
func scalarCorpus() []scalarRow {
	return []scalarRow{
		{name: "plain admits", inner: `abc`, environRefuse: "", scalarRefuse: ""},
		{name: "bare high refused", inner: `A\ud800B`, environRefuse: environ.FaultLoneSurrogate, scalarRefuse: scalarGateText},
		{name: "bare low refused", inner: `A\udc00B`, environRefuse: environ.FaultLoneSurrogate, scalarRefuse: scalarGateText},
		{name: "high followed by non-low refused", inner: `A\uD83D\u0041B`, environRefuse: environ.FaultLoneSurrogate, scalarRefuse: scalarGateText},
		{name: "low followed by high refused", inner: `A\udc00\uD83DB`, environRefuse: environ.FaultLoneSurrogate, scalarRefuse: scalarGateText},
		{name: "paired surrogates admit", inner: `A\uD83D\uDE00B`, environRefuse: "", scalarRefuse: scalarMemberText, scalarAbsent: scalarGateText},
		{name: "escaped backslash high run2 admits", inner: `A\\ud800B`, environRefuse: "", scalarRefuse: scalarMemberText, scalarAbsent: scalarGateText},
		{name: "escaped backslash low run2 admits", inner: `A\\udc00B`, environRefuse: "", scalarRefuse: scalarMemberText, scalarAbsent: scalarGateText},
		{name: "escaped backslash high run3 refused", inner: `A\\\ud800B`, environRefuse: environ.FaultLoneSurrogate, scalarRefuse: scalarGateText},
		{name: "escaped backslash plus real lone low refused", inner: `A\\ud800\udc00B`, environRefuse: environ.FaultLoneSurrogate, scalarRefuse: scalarGateText},
		{name: "escaped backslash plus real lone high refused", inner: `A\\udc00\ud800B`, environRefuse: environ.FaultLoneSurrogate, scalarRefuse: scalarGateText},
		{name: "escaped backslash plus real pair admits", inner: `A\\ud800\uD83D\uDE00B`, environRefuse: "", scalarRefuse: scalarMemberText, scalarAbsent: scalarGateText},
		{name: "malformed escape falls to syntax", inner: `A\uDEFG B`, environRefuse: environ.FaultNotObject, scalarRefuse: scalarSyntaxText},
		{name: "truncated escape falls to syntax", inner: `A\uD83`, environRefuse: environ.FaultNotObject, scalarRefuse: scalarSyntaxText},
		{name: "multibyte text admits", inner: `plain ü ☃`, environRefuse: "", scalarRefuse: scalarMemberText, scalarAbsent: scalarGateText},
	}
}

// TestScalarSurrogateAgreementAcrossUnits drives the shared
// corpus through both gate units and requires the ledgered
// verdict at each.
func TestScalarSurrogateAgreementAcrossUnits(t *testing.T) {
	for _, row := range scalarCorpus() {
		t.Run(row.name, func(t *testing.T) {
			literal := `"` + row.inner + `"`
			frame := []byte(`{"v":` + literal + `}`)
			_, fault := environ.DecodeStrictObject(frame)
			if row.environRefuse == "" {
				if fault != nil {
					t.Fatalf("environ refused %q: %v", frame, fault)
				}
			} else {
				if fault == nil {
					t.Fatalf("environ admitted %q, want fault %q", frame, row.environRefuse)
				}
				if fault.Detail != row.environRefuse {
					t.Fatalf("environ fault = %q, want %q", fault.Detail, row.environRefuse)
				}
			}
			_, err := scalar.DecodeClosedEnumJSON([]byte(literal), "abc")
			if row.scalarRefuse == "" {
				if err != nil {
					t.Fatalf("scalar refused %q, want admit: %v", literal, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("scalar admitted %q, want refusal containing %q", literal, row.scalarRefuse)
			}
			if !strings.Contains(err.Error(), row.scalarRefuse) {
				t.Fatalf("scalar error = %v, want text containing %q", err, row.scalarRefuse)
			}
			if row.scalarAbsent != "" && strings.Contains(err.Error(), row.scalarAbsent) {
				t.Fatalf("scalar error = %v, must not contain %q", err, row.scalarAbsent)
			}
		})
	}
}
