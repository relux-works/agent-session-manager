package environ_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/dirnode"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file is the cross-facade frame-agreement battery. Every
// Section 7.8, 7.9, and provider-protocol body crosses a strict
// object decoder before any member is trusted, and each facade
// carries its own copy of that decoder (ledgered in
// census_test.go). The battery drives one shared corpus through
// all five judges — environ.DecodeStrictObject,
// sessadapter.DecodeTuple, dirnode.CheckScanRequest,
// provhost.DecodeManifest, and canonicaljson.Canonicalize — and
// requires the ledgered verdict at each.
//
// A row whose reached arm cannot be determined fails, never
// passes: frame-fault rows assert the full rendering (the fault
// detail phrase in the refusal text), and accept rows assert the
// body travels past the frame gate (an accept, or a refusal at a
// named later arm with the frame phrase asserted absent). A code
//-only assertion would conflate the frame gate with the member
// gates behind it, which carry the same refusal code.
//
// The roster fails closed in both directions: every fault detail
// derived from environ production must own at least one row, and
// a row naming a detail production no longer derives fails as
// orphaned. Expected verdicts are ledgered per judge, so a
// verdict flip in EITHER direction — a fix or a regression —
// reddens its row instead of passing silently.

// frameRow is one corpus vector with the ledgered verdict at
// every judge. A refuse expectation names the refusal text the
// verdict must contain; an accept expectation names a later-arm
// text the verdict must contain ("" with accept=true means the
// entry itself accepts). canonicalRefuse nil means the
// canonicalizer is not judged on this row: it validates JSON
// values, not closed objects, so object-shape faults
// (non-object, duplicate, trailing) are outside the decoder
// rules it shares with the facades.
type frameRow struct {
	fault string
	name  string
	body  []byte

	// Per-judge bodies override body where an entry needs a
	// shaped vector (accept controls): nil means the shared
	// minimal body, which every frame gate refuses before any
	// member is read.
	sessBody []byte
	dirBody  []byte
	provBody []byte

	environRefuse string
	environMember string

	sessAccept bool
	sessText   string
	sessAbsent string

	dirAccept bool
	dirText   string
	dirAbsent string

	provAccept bool
	provText   string
	provAbsent string

	canonicalRefuse *bool
}

func boolPointer(value bool) *bool {
	return &value
}

// frameCorpus is the shared fault corpus. Surrogate subclasses
// enumerate the full shape of the gate: bare highs and lows,
// mispairs, truncated and malformed escapes, paired controls,
// backslash-run parity (runs of 1..3 backslashes before the
// escape, in value, member-name, and nested positions), and the
// composed shapes an even run plus a FOLLOWING real escape. Run
// parity plus composition proves both directions of the old
// raw-scan divergence instead of one: a `\u` sequence behind an
// odd run is a real escape every judge refuses when lone, while
// the same sequence behind an even run is quoted text the frame
// gate admits (the member arms behind it still refuse the
// minimal bodies, with the frame phrase asserted absent). The
// composed rows close the sampling hole the first battery left:
// an even run followed by a real lone escape is refused by the
// string walk, while the raw scan misread the quoted text as a
// high surrogate, paired it with the following real escape, and
// admitted a body encoding/json silently rewrote to U+FFFD.
func frameCorpus() []frameRow {
	pairVersion := `"environment_version":"A\uD83D\uDE00B"`
	pairTuple := []byte(strings.Replace(validTupleJSON(), `"environment_version":"2.1.0"`, pairVersion, 1))
	pairCursor := validScanRequestJSON(`"A\uD83D\uDE00B"`)
	rows := []frameRow{
		{
			fault: environ.FaultNotUTF8, name: "non-utf8 bytes",
			body:          []byte("{\"v\":\"a\xff\"}"),
			environRefuse: environ.FaultNotUTF8,
			sessText:      environ.FaultNotUTF8, dirText: environ.FaultNotUTF8, provText: environ.FaultNotUTF8,
			canonicalRefuse: boolPointer(true),
		},
		{
			fault: environ.FaultLoneSurrogate, name: "bare high escape",
			body:          []byte("{\"v\":\"A\\ud800B\"}"),
			environRefuse: environ.FaultLoneSurrogate,
			sessText:      environ.FaultLoneSurrogate, dirText: environ.FaultLoneSurrogate, provText: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(true),
		},
		{
			fault: environ.FaultLoneSurrogate, name: "bare low escape",
			body:          []byte("{\"v\":\"A\\udc00B\"}"),
			environRefuse: environ.FaultLoneSurrogate,
			sessText:      environ.FaultLoneSurrogate, dirText: environ.FaultLoneSurrogate, provText: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(true),
		},
		{
			fault: environ.FaultLoneSurrogate, name: "high followed by non-low",
			body:          []byte("{\"v\":\"A\\uD83D\\u0041B\"}"),
			environRefuse: environ.FaultLoneSurrogate,
			sessText:      environ.FaultLoneSurrogate, dirText: environ.FaultLoneSurrogate, provText: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(true),
		},
		{
			// Truncated and malformed escapes are not well-formed
			// \uXXXX escapes, so the surrogate gate passes them
			// to the syntax arms on every judge.
			fault: environ.FaultNotObject, name: "truncated escape falls to syntax",
			body:          []byte("{\"v\":\"A\\uD83\"}"),
			environRefuse: environ.FaultNotObject, environMember: "v",
			sessText: "environment tuple not a JSON object", sessAbsent: environ.FaultLoneSurrogate,
			dirText: "scan request not a JSON object", dirAbsent: environ.FaultLoneSurrogate,
			provText:        environ.FaultNotObject,
			canonicalRefuse: boolPointer(true),
		},
		{
			fault: environ.FaultNotObject, name: "malformed escape falls to syntax",
			body:          []byte("{\"v\":\"A\\uDEFG B\"}"),
			environRefuse: environ.FaultNotObject, environMember: "v",
			sessText: "environment tuple not a JSON object", sessAbsent: environ.FaultLoneSurrogate,
			dirText: "scan request not a JSON object", dirAbsent: environ.FaultLoneSurrogate,
			provText:        environ.FaultNotObject,
			canonicalRefuse: boolPointer(true),
		},
		{
			// An even backslash run quotes the following text:
			// the frame gate admits and the member arms refuse,
			// with the frame phrase asserted absent. Before the
			// facade delegation the raw scan refused these at
			// the surrogate arm; the flip is the recorded fix.
			fault: environ.FaultLoneSurrogate, name: "escaped backslash high run2",
			body:     []byte("{\"v\":\"A\\\\ud800B\"}"),
			sessText: "environment tuple carries unknown member", sessAbsent: environ.FaultLoneSurrogate,
			dirText: "scan request carries unknown member", dirAbsent: environ.FaultLoneSurrogate,
			provText: "manifest carries unknown member", provAbsent: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(false),
		},
		{
			// Run 3 is an escaped backslash followed by a REAL
			// lone escape (value `A\` plus U+D800), so every
			// judge refuses: odd runs escape, even runs quote.
			fault: environ.FaultLoneSurrogate, name: "escaped backslash high run3",
			body:          []byte("{\"v\":\"A\\\\\\ud800B\"}"),
			environRefuse: environ.FaultLoneSurrogate,
			sessText:      environ.FaultLoneSurrogate, dirText: environ.FaultLoneSurrogate, provText: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(true),
		},
		{
			fault: environ.FaultLoneSurrogate, name: "escaped backslash low run2",
			body:     []byte("{\"v\":\"A\\\\udc00B\"}"),
			sessText: "environment tuple carries unknown member", sessAbsent: environ.FaultLoneSurrogate,
			dirText: "scan request carries unknown member", dirAbsent: environ.FaultLoneSurrogate,
			provText: "manifest carries unknown member", provAbsent: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(false),
		},
		{
			fault: environ.FaultLoneSurrogate, name: "escaped backslash pair run2",
			body:     []byte("{\"v\":\"A\\\\uD83D\\\\uDE00B\"}"),
			sessText: "environment tuple carries unknown member", sessAbsent: environ.FaultLoneSurrogate,
			dirText: "scan request carries unknown member", dirAbsent: environ.FaultLoneSurrogate,
			provText: "manifest carries unknown member", provAbsent: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(false),
		},
		{
			fault: environ.FaultLoneSurrogate, name: "escaped backslash in member name",
			body:     []byte("{\"A\\\\ud800B\":1}"),
			sessText: "environment tuple carries unknown member", sessAbsent: environ.FaultLoneSurrogate,
			dirText: "scan request carries unknown member", dirAbsent: environ.FaultLoneSurrogate,
			provText: "manifest carries unknown member", provAbsent: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(false),
		},
		{
			fault: environ.FaultLoneSurrogate, name: "escaped backslash nested",
			body:     []byte("{\"v\":{\"w\":\"A\\\\ud800B\"}}"),
			sessText: "environment tuple carries unknown member", sessAbsent: environ.FaultLoneSurrogate,
			dirText: "scan request carries unknown member", dirAbsent: environ.FaultLoneSurrogate,
			provText: "manifest carries unknown member", provAbsent: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(false),
		},
		{
			// An even run followed by a REAL lone low escape:
			// the string walk consumes the `\\` pair, reads
			// quoted text, then refuses the real lone escape.
			// The raw scan misread the quoted text as a high
			// surrogate, paired it with the real low escape,
			// and admitted the body. This row is the composed
			// shape the first battery never sampled.
			fault: environ.FaultLoneSurrogate, name: "escaped backslash plus real lone low",
			body:          []byte("{\"v\":\"A\\\\ud800\\udc00B\"}"),
			environRefuse: environ.FaultLoneSurrogate,
			sessText:      environ.FaultLoneSurrogate, dirText: environ.FaultLoneSurrogate, provText: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(true),
		},
		{
			// The same composition with a real lone high
			// escape behind the even run: every judge refuses.
			fault: environ.FaultLoneSurrogate, name: "escaped backslash plus real lone high",
			body:          []byte("{\"v\":\"A\\\\udc00\\ud800B\"}"),
			environRefuse: environ.FaultLoneSurrogate,
			sessText:      environ.FaultLoneSurrogate, dirText: environ.FaultLoneSurrogate, provText: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(true),
		},
		{
			// An even run followed by a REAL pair: the frame
			// gate admits on every judge and the member arms
			// refuse, with the frame phrase asserted absent.
			// Before the facade delegation the raw scan
			// refused this at the surrogate arm (it misread
			// the quoted text as a high surrogate followed by
			// a non-low), so this row proves the composed
			// shape on the over-strict side too.
			fault: environ.FaultLoneSurrogate, name: "escaped backslash plus real pair",
			body:     []byte("{\"v\":\"A\\\\ud800\\uD83D\\uDE00B\"}"),
			sessText: "environment tuple carries unknown member", sessAbsent: environ.FaultLoneSurrogate,
			dirText: "scan request carries unknown member", dirAbsent: environ.FaultLoneSurrogate,
			provText: "manifest carries unknown member", provAbsent: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(false),
		},
		{
			fault: environ.FaultDuplicate, name: "duplicate member",
			body:          []byte("{\"v\":1,\"v\":2}"),
			environRefuse: environ.FaultDuplicate, environMember: "v",
			sessText: environ.FaultDuplicate, dirText: environ.FaultDuplicate, provText: environ.FaultDuplicate,
		},
		{
			fault: environ.FaultTrailing, name: "trailing data",
			body:          []byte("{\"v\":1} {}"),
			environRefuse: environ.FaultTrailing,
			sessText:      environ.FaultTrailing, dirText: environ.FaultTrailing, provText: environ.FaultTrailing,
		},
		{
			fault: environ.FaultNotObject, name: "non-object top level",
			body:          []byte("[1,2]"),
			environRefuse: environ.FaultNotObject,
			sessText:      environ.FaultNotObject, dirText: environ.FaultNotObject, provText: environ.FaultNotObject,
		},
		{
			fault: environ.FaultNotObject, name: "unterminated string is syntax",
			body:          []byte("{\"v\":\"abc"),
			environRefuse: environ.FaultNotObject, environMember: "v",
			sessText: environ.FaultNotObject, dirText: environ.FaultNotObject, provText: environ.FaultNotObject,
			canonicalRefuse: boolPointer(true),
		},
		{
			fault: environ.FaultLoneSurrogate, name: "paired surrogates admit",
			body:       []byte(`{"v":"A\uD83D\uDE00B"}`),
			sessBody:   pairTuple,
			dirBody:    pairCursor,
			provBody:   []byte(`{"note":"A\uD83D\uDE00B"}`),
			sessAccept: true,
			dirAccept:  true,
			provText:   "manifest carries unknown member", provAbsent: environ.FaultLoneSurrogate,
			canonicalRefuse: boolPointer(false),
		},
		{
			fault: environ.FaultNotUTF8, name: "multibyte text admits",
			body:       []byte("{\"v\":\"plain ü ☃\"}"),
			sessBody:   pairTuple,
			dirBody:    pairCursor,
			provBody:   []byte("{\"note\":\"plain ü ☃\"}"),
			sessAccept: true,
			dirAccept:  true,
			provText:   "manifest carries unknown member", provAbsent: environ.FaultNotUTF8,
			canonicalRefuse: boolPointer(false),
		},
	}
	return rows
}

// deriveFaultDetails derives the frame-fault space from environ
// production: every Fault* constant value in decode.go. A sixth
// fault shape fails the roster below instead of passing
// unrostered.
func deriveFaultDetails(t *testing.T) map[string]bool {
	t.Helper()
	path := filepath.Join(mustGetwd(t), "decode.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("faults: %v", err)
	}
	syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		t.Fatalf("faults: %v", err)
	}
	details := map[string]bool{}
	for _, decl := range syntax.Decls {
		node, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range value.Names {
				if !strings.HasPrefix(name.Name, "Fault") || index >= len(value.Values) {
					continue
				}
				literal, ok := value.Values[index].(*ast.BasicLit)
				if !ok {
					t.Fatalf("faults: %s is not a literal; the extractor cannot see through indirection", name.Name)
				}
				text, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("faults: %v", err)
				}
				details[text] = true
			}
		}
	}
	if len(details) == 0 {
		t.Fatal("faults derived zero details; the scanner is blind, not the space empty")
	}
	return details
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	return directory
}

// TestFrameFaultRosterIsClosed requires every derived fault
// detail to own at least one row, and every row to name a
// derived detail.
func TestFrameFaultRosterIsClosed(t *testing.T) {
	details := deriveFaultDetails(t)
	covered := map[string]bool{}
	for _, row := range frameCorpus() {
		if row.fault == "" {
			t.Fatalf("row %q names no fault detail", row.name)
		}
		if !details[row.fault] {
			t.Fatalf("row %q names underived detail %q", row.name, row.fault)
		}
		covered[row.fault] = true
	}
	for detail := range details {
		if !covered[detail] {
			t.Errorf("derived fault detail %q owns no corpus row", detail)
		}
	}
}

// TestFrameAgreementAcrossFacades drives the shared corpus
// through all five judges and requires the ledgered verdict at
// each. environ rows assert the exact fault and member; facade
// rows assert the full frame rendering (the detail phrase in the
// refusal text) or, for accept rows, travel past the frame gate
// with the frame phrase asserted absent; canonical rows assert
// the verdict only, where the canonicalizer shares the rule.
func TestFrameAgreementAcrossFacades(t *testing.T) {
	for _, row := range frameCorpus() {
		t.Run(row.name, func(t *testing.T) {
			_, fault := environ.DecodeStrictObject(row.body)
			if row.environRefuse == "" {
				if fault != nil {
					t.Fatalf("environ refused %q: %v", row.body, fault)
				}
			} else {
				if fault == nil {
					t.Fatalf("environ admitted %q, want fault %q", row.body, row.environRefuse)
				}
				if fault.Detail != row.environRefuse || fault.Member != row.environMember {
					t.Fatalf("environ fault = (%q, %q), want (%q, %q)", fault.Detail, fault.Member, row.environRefuse, row.environMember)
				}
			}
			sessBody := orBody(row.body, row.sessBody)
			dirBody := orBody(row.body, row.dirBody)
			provBody := orBody(row.body, row.provBody)
			assertFacadeVerdict(t, "sessadapter", sessBody, row.sessAccept, row.sessText, row.sessAbsent, func(body []byte) error {
				_, err := sessadapter.DecodeTuple(body)
				return err
			})
			assertFacadeVerdict(t, "dirnode", dirBody, row.dirAccept, row.dirText, row.dirAbsent, func(body []byte) error {
				_, err := dirnode.CheckScanRequest(body)
				return err
			})
			assertFacadeVerdict(t, "provhost", provBody, row.provAccept, row.provText, row.provAbsent, func(body []byte) error {
				return provhost.DecodeManifest(body)
			})
			if row.canonicalRefuse != nil {
				_, err := canonicaljson.Canonicalize(row.body)
				if (*row.canonicalRefuse && err == nil) || (!*row.canonicalRefuse && err != nil) {
					t.Fatalf("canonicaljson verdict = refuse=%v, want refuse=%v (err=%v)", err != nil, *row.canonicalRefuse, err)
				}
			}
		})
	}
}

// orBody selects the per-judge override body when present.
func orBody(shared, override []byte) []byte {
	if override != nil {
		return override
	}
	return shared
}

// assertFacadeVerdict requires the ledgered verdict at one
// facade entry: an accept, or a refusal whose text contains the
// frame phrase (and never contains the absent phrase, which
// pins travel past a gate the body must cross).
func assertFacadeVerdict(t *testing.T, facade string, body []byte, accept bool, text, absent string, drive func([]byte) error) {
	t.Helper()
	err := drive(body)
	if accept {
		if err != nil {
			t.Fatalf("%s refused %q, want accept: %v", facade, body, err)
		}
		return
	}
	if err == nil {
		t.Fatalf("%s admitted %q, want refusal containing %q", facade, body, text)
	}
	if text != "" && !strings.Contains(err.Error(), text) {
		t.Fatalf("%s error = %v, want text containing %q", facade, err, text)
	}
	if absent != "" && strings.Contains(err.Error(), absent) {
		t.Fatalf("%s error = %v, must not contain %q", facade, err, absent)
	}
}
