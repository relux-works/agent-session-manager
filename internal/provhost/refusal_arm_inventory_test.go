package provhost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file is the derived refusal-arm inventory. It exists because the
// constructor call-site audit in inventory_test.go enumerates constructor
// call SITES, and every frameFault literal and every integrity(...) detail
// funnels through a shared site: planting a new arm through an existing
// site (a new &frameFault literal in checkResponseMembers, a new
// integrity(...) detail in DecodeStatusOutcome) leaves the whole suite
// green. Two review rounds each missed a row by hand, so the arm set is
// derived from production source by parsing it, never by listing it.
//
// Derivation (deriveRefusalArms): every &frameFault{...} literal yields
// frame|<detail>|<member source>; every integrity(...) call with a string
// literal yields integrity|<detail>; the single wrapping site
// integrity("status body is "+fault.detail, ...) expands over the derived
// frame detail set, so each wrapped refusal is its own obligation; every
// rejection branch of parseMajor (a return 0, false) yields
// parse|<enclosing condition source>; every literal first argument to one
// of the six refusal constructors yields ctor|<constructor>|<detail>.
// A non-literal frameFault detail or integrity argument lands as an
// expr:<source> obligation rather than passing silently; a non-literal
// constructor first argument is a fault conduit (fault.detail, a detail
// variable) whose arms are the frame arms above, so it carries no
// separate obligation by design. A missing parseMajor fails the
// derivation outright.
//
// Both directions are checked: TestDerivedRefusalArmsAreAllWitnessed
// (every derived arm carries a witness) and
// TestWitnessedArmsAreAllDerived (every witness names a derived arm, so a
// truncated derivation reddens on the orphaned witnesses instead of
// passing vacuously). TestEveryArmWitnessRefusesAtTheProductionEntry
// drives each witness through the production entry and requires the
// refusal to come from the arm under test. Empty, single-file, or
// unparseable derivations fail closed: a domain that silently derives
// nothing is not a measurement.
//
// Stated bound: arms sharing one (constructor, detail) identity merge to
// one obligation (the two timeout sites, the two empty-response sites,
// the six identical "not a JSON object" sites). Deleting one of a merged
// pair is a behavioral mutant the entry-point tests must catch, not this
// inventory.

// refusalArmCensusFloor is the derived arm count this test was written
// against. It is a tripwire, not an enumeration: a derivation returning
// fewer arms fails closed even when every surviving arm is witnessed,
// which is what catches a silently truncated scan. Raising it is routine
// when production gains a refusal; lowering it requires saying why.
//
// Raised 60 -> 162 by the operation-layer leaf (manifest, probe,
// profile, quiescence, spawn, identity, idempotency decoders), each
// new arm witnessed in declaredOperationWitnesses below, then 162 ->
// 164 by the lone-surrogate gate and 164 -> 166 by the UTF-8 gate:
// one frame arm in decodeStrictObject plus its status-body integrity
// expansion each, every one witnessed above.
const refusalArmCensusFloor = 166

func deriveRefusalArms(t *testing.T) map[string]struct{} {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("derive refusal arms: %v", err)
	}
	arms, scanned, err := refusalArmsIn(directory)
	if err != nil {
		t.Fatalf("derive refusal arms: %v", err)
	}
	if len(scanned) == 0 {
		t.Fatal("derived refusal arms from zero production files; the scanner is broken, not the package")
	}
	if len(arms) == 0 {
		t.Fatal("derived zero refusal arms from the package sources; the scanner is broken, not the package")
	}
	if len(arms) < refusalArmCensusFloor {
		t.Fatalf("derived %d refusal arms, below the %d census floor; the derivation is short, not the package", len(arms), refusalArmCensusFloor)
	}
	t.Logf("refusal arm coverage domain: %d derived arms across %d production files", len(arms), len(scanned))
	return arms
}

func refusalArmsIn(directory string) (map[string]struct{}, []string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, nil, err
	}
	arms := map[string]struct{}{}
	var scanned []string
	parseBranches := 0
	fileSet := token.NewFileSet()
	type productionFile struct {
		syntax *ast.File
		source []byte
	}
	var files []productionFile
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned = append(scanned, name)
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		syntax, err := parser.ParseFile(fileSet, path, source, parser.ParseComments)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, productionFile{syntax: syntax, source: source})
	}
	// First pass, package-wide: the decodeStrictObject fault detail
	// set, which the integrity wrapper site expands over. The wrapper
	// only wraps decodeStrictObject faults, so checkResponseMembers
	// arms ("missing member", "unknown member", ...) are not in this
	// set: expanding over every frameFault literal would invent arms
	// production can never emit. The pass must span files because the
	// literals live in protocol.go while the wrapper lives in status.go.
	var strictDetails []string
	seenDetail := map[string]bool{}
	for _, file := range files {
		for _, decl := range file.syntax.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Name.Name != "decodeStrictObject" {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				literal, ok := frameFaultLiteral(node)
				if !ok {
					return true
				}
				detail, ok := literalStringField(file.source, fileSet, literal, "detail")
				if !ok {
					return true
				}
				if !seenDetail[detail] {
					seenDetail[detail] = true
					strictDetails = append(strictDetails, detail)
				}
				return true
			})
		}
	}
	for _, file := range files {
		source, syntax := file.source, file.syntax
		parseBranches += armParseBranches(source, fileSet, syntax, arms)
		ast.Inspect(syntax, func(node ast.Node) bool {
			if literal, ok := frameFaultLiteral(node); ok {
				detail, ok := literalStringField(source, fileSet, literal, "detail")
				if !ok {
					arms["frame|expr:"+nodeSource(source, fileSet, node)] = struct{}{}
					return true
				}
				member := ""
				if kv := literalField(literal, "member"); kv != nil {
					member = nodeSource(source, fileSet, kv.Value)
				}
				arms["frame|"+detail+"|"+member] = struct{}{}
				return true
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || len(call.Args) == 0 {
				return true
			}
			switch ident.Name {
			case "integrity":
				if literal, ok := call.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
					detail, err := strconv.Unquote(literal.Value)
					if err != nil {
						arms["integrity|expr:"+nodeSource(source, fileSet, call.Args[0])] = struct{}{}
						return true
					}
					arms["integrity|"+detail] = struct{}{}
					return true
				}
				if prefix, ok := integrityPrefix(call.Args[0]); ok {
					if len(strictDetails) == 0 {
						arms["integrity|expr:"+nodeSource(source, fileSet, call.Args[0])] = struct{}{}
						return true
					}
					for _, detail := range strictDetails {
						arms["integrity|"+prefix+detail] = struct{}{}
					}
					return true
				}
				arms["integrity|expr:"+nodeSource(source, fileSet, call.Args[0])] = struct{}{}
			case "failInvalid", "failProtocol", "failMismatch", "failProcess", "failTimeout":
				if literal, ok := call.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
					detail, err := strconv.Unquote(literal.Value)
					if err != nil {
						return true
					}
					arms["ctor|"+ident.Name+"|"+detail] = struct{}{}
				}
				// Non-literal first arguments are the fault conduits
				// (fault.detail, detail variables): the arms behind
				// them are the frame arms above, so there is no
				// separate obligation here.
			}
			return true
		})
	}
	if parseBranches == 0 {
		return nil, nil, fmt.Errorf("parseMajor carries no rejection branch anywhere; the classification-branch enumeration is blind")
	}
	sort.Strings(scanned)
	return arms, scanned, nil
}

// frameFaultLiteral reports whether node is a &frameFault{...} literal.
func frameFaultLiteral(node ast.Node) (*ast.CompositeLit, bool) {
	target := node
	if unary, ok := node.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		target = unary.X
	}
	literal, ok := target.(*ast.CompositeLit)
	if !ok {
		return nil, false
	}
	ident, ok := literal.Type.(*ast.Ident)
	if !ok || ident.Name != "frameFault" {
		return nil, false
	}
	return literal, true
}

func literalField(literal *ast.CompositeLit, name string) *ast.KeyValueExpr {
	for _, element := range literal.Elts {
		if kv, ok := element.(*ast.KeyValueExpr); ok {
			if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == name {
				return kv
			}
		}
	}
	return nil
}

func literalStringField(source []byte, fileSet *token.FileSet, literal *ast.CompositeLit, name string) (string, bool) {
	kv := literalField(literal, name)
	if kv == nil {
		return "", false
	}
	basic, ok := kv.Value.(*ast.BasicLit)
	if !ok || basic.Kind != token.STRING {
		return "", false
	}
	detail, err := strconv.Unquote(basic.Value)
	if err != nil {
		return "", false
	}
	return detail, true
}

// integrityPrefix matches "prefix" + <expr>, the wrapping shape whose
// arms expand over the derived frame detail set.
func integrityPrefix(arg ast.Expr) (string, bool) {
	binary, ok := arg.(*ast.BinaryExpr)
	if !ok || binary.Op != token.ADD {
		return "", false
	}
	prefix, ok := binary.X.(*ast.BasicLit)
	if !ok || prefix.Kind != token.STRING {
		return "", false
	}
	text, err := strconv.Unquote(prefix.Value)
	if err != nil {
		return "", false
	}
	return text, true
}

func nodeSource(source []byte, fileSet *token.FileSet, node ast.Node) string {
	start := fileSet.Position(node.Pos()).Offset
	end := fileSet.Position(node.End()).Offset
	if start < 0 || end > len(source) || start >= end {
		return "?"
	}
	return strings.Join(strings.Fields(string(source[start:end])), " ")
}

// armParseBranches derives one obligation per parseMajor rejection
// branch in this file: every return 0, false inside parseMajor, keyed by
// its innermost enclosing condition, and returns how many it derived. A
// new branch adds a key no witness names; a deleted branch orphans its
// witness; narrowing || to && rewrites the key and does both. The caller
// fails closed when no file derives any branch.
func armParseBranches(source []byte, fileSet *token.FileSet, syntax *ast.File, arms map[string]struct{}) int {
	var body *ast.BlockStmt
	for _, decl := range syntax.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if ok && function.Name.Name == "parseMajor" {
			body = function.Body
		}
	}
	if body == nil {
		return 0
	}
	count := 0
	var conditions []string
	var walk func(node ast.Node)
	walk = func(node ast.Node) {
		branch, ok := node.(*ast.IfStmt)
		if !ok {
			for _, child := range childNodes(node) {
				walk(child)
			}
			return
		}
		conditions = append(conditions, nodeSource(source, fileSet, branch.Cond))
		for _, stmt := range branch.Body.List {
			if ret, ok := stmt.(*ast.ReturnStmt); ok && isFalsePair(ret) {
				arms["parse|"+conditions[len(conditions)-1]] = struct{}{}
				count++
			}
		}
		for _, child := range childNodes(node) {
			walk(child)
		}
		conditions = conditions[:len(conditions)-1]
	}
	walk(body)
	return count
}

func isFalsePair(ret *ast.ReturnStmt) bool {
	if len(ret.Results) != 2 {
		return false
	}
	zero, ok := ret.Results[0].(*ast.BasicLit)
	if !ok || zero.Kind != token.INT || zero.Value != "0" {
		return false
	}
	no, ok := ret.Results[1].(*ast.Ident)
	return ok && no.Name == "false"
}

func childNodes(node ast.Node) []ast.Node {
	var children []ast.Node
	ast.Inspect(node, func(child ast.Node) bool {
		if child == nil || child == node {
			return true
		}
		children = append(children, child)
		return false
	})
	return children
}

// armWitness proves one derived refusal arm at the production entry.
// The arm key must equal the derived key exactly: TestWitnessedArmsAreAllDerived
// fails a witness that names nothing production declares, so a deleted or
// narrowed production branch orphans its witness instead of passing
// silently. The prove function must drive the production entry point and
// require the refusal attributed to THIS arm, not merely any refusal.
type armWitness struct {
	arm   string
	name  string
	prove func(*testing.T)
}

// requireIntegrityRefusal asserts the full identity of a status refusal:
// the integrity_failure code, the rule detail, the status_state detail
// every integrity arm carries (AS6: asserted on all 18 arms, not just
// the unknown row), exit 9, and the non-retryable bit. It is separate
// from requireLocalRefusal because that helper also serves codes with no
// status_state detail.
func requireIntegrityRefusal(t *testing.T, err error, detail, wantState string) {
	t.Helper()
	requireLocalRefusal(t, err, "integrity_failure", detail)
	state, ok := failureObject(t, err).Detail("status_state")
	if !ok || state != wantState {
		t.Fatalf("integrity refusal status_state = %v, want %q (%v)", state, wantState, err)
	}
	if failureExit(t, err) != 9 {
		t.Fatalf("integrity refusal exit = %d, want 9", failureExit(t, err))
	}
}

func armVersionFrame(version string) []byte {
	return []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"` + version + `","request_id":"` + testRequestID + `","ok":true,"body":{}}`)
}

func armMemberFrame(members string) []byte {
	return []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `",` + members + `}`)
}

func armOversizeResponse(t *testing.T) []byte {
	t.Helper()
	build := func(pad int) []byte {
		return successFrame(t, testRequestID, `{"pad":"`+strings.Repeat("a", pad)+`"}`)
	}
	base := len(build(0))
	over := build(specFrameLimitBytes - base + 1)
	if len(over) != specFrameLimitBytes+1 {
		t.Fatalf("oversize fixture is %d bytes, want exactly %d", len(over), specFrameLimitBytes+1)
	}
	return over
}

func declaredArmWitnesses() []armWitness {
	return []armWitness{
		// decodeStrictObject frame arms, through DecodeResponse.
		{arm: `frame|not a JSON object|""`, name: "array is not an object", prove: func(t *testing.T) {
			_, err := DecodeResponse([]byte(`[1,2]`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "", "not a JSON object")
		}},
		{arm: `frame|not a JSON object|key`, name: "truncated value names its member", prove: func(t *testing.T) {
			_, err := DecodeResponse([]byte(`{"protocol":`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "protocol", "not a JSON object")
		}},
		{arm: `frame|duplicate member|key`, name: "repeated member", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","request_id":"` + testRequestID + `","ok":true,"body":{}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "request_id", "duplicate member")
		}},
		{arm: `frame|trailing data after the object|""`, name: "trailing byte", prove: func(t *testing.T) {
			_, err := DecodeResponse(append(successFrame(t, testRequestID, `{"provider_id":"pi"}`), 'x'), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "", "trailing data after the object")
		}},
		{arm: `frame|lone surrogate escape|""`, name: "lone surrogate in body", prove: func(t *testing.T) {
			_, err := DecodeResponse(successFrame(t, testRequestID, `{"provider_id":"pi","tag":"a\ud800b"}`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "", "lone surrogate escape")
		}},
		{arm: `frame|not valid UTF-8|""`, name: "raw bytes in body", prove: func(t *testing.T) {
			err := DecodeManifest([]byte("{\"note\":\"ab\xffcd\"}"))
			requireFrameRefusal(t, err, "", "not valid UTF-8")
		}},
		{arm: `frame|missing member|name`, name: "missing protocol", prove: func(t *testing.T) {
			frame := []byte(`{"protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "protocol", "missing member")
		}},
		{arm: `frame|missing member|"body"`, name: "success without body", prove: func(t *testing.T) {
			_, err := DecodeResponse(armMemberFrame(`"ok":true`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "body", "missing member")
		}},
		{arm: `frame|success envelope carries error|"error"`, name: "error on success", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":true,"body":{},"error":{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"capability_unavailable","message":"m","exit_code":6,"retryable":false,"details":{}}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "error", "success envelope carries error")
		}},
		{arm: `frame|missing member|"error"`, name: "failure without error", prove: func(t *testing.T) {
			_, err := DecodeResponse(armMemberFrame(`"ok":false`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "error", "missing member")
		}},
		{arm: `frame|failure envelope carries body|"body"`, name: "body on failure", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"capability_unavailable","message":"m","exit_code":6,"retryable":false,"details":{}},"body":{}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "body", "failure envelope carries body")
		}},
		{arm: `frame|unknown member|name`, name: "diagnostics member", prove: func(t *testing.T) {
			_, err := DecodeResponse(armMemberFrame(`"ok":true,"body":{},"diagnostics":[]`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "diagnostics", "unknown member")
		}},
		// encodeFrame refusals, through EncodeRequest.
		{arm: `ctor|failInvalid|unknown operation`, name: "unknown operation never frames", prove: func(t *testing.T) {
			req := testRequest(t)
			req.Operation = "reboot"
			_, err := EncodeRequest(req, mustInstant(t, testNow))
			requireLocalRefusal(t, err, "invalid_config", "unknown operation")
		}},
		{arm: `ctor|failInvalid|request_id is not a UUIDv7`, name: "zero request id", prove: func(t *testing.T) {
			bad := testRequest(t)
			var zero scalar.UUIDv7
			bad.RequestID = zero
			_, err := EncodeRequest(bad, mustInstant(t, testNow))
			requireLocalRefusal(t, err, "invalid_config", "request_id is not a UUIDv7")
		}},
		{arm: `ctor|failInvalid|deadline is not a timestamp`, name: "zero deadline", prove: func(t *testing.T) {
			req := testRequest(t)
			req.Deadline = mustTimestamp(t, testDeadline)
			var zero scalar.Timestamp
			req.Deadline = zero
			_, err := EncodeRequest(req, mustInstant(t, testNow))
			requireLocalRefusal(t, err, "invalid_config", "deadline is not a timestamp")
		}},
		{arm: `ctor|failInvalid|deadline is not in the future`, name: "stale deadline", prove: func(t *testing.T) {
			req := testRequest(t)
			req.Deadline = mustTimestamp(t, testNow)
			_, err := EncodeRequest(req, mustInstant(t, testNow))
			requireLocalRefusal(t, err, "invalid_config", "deadline is not in the future")
		}},
		{arm: `ctor|failInvalid|body is not a JSON object`, name: "array body", prove: func(t *testing.T) {
			req := testRequest(t)
			req.Body = json.RawMessage(`[1,2]`)
			_, err := EncodeRequest(req, mustInstant(t, testNow))
			requireLocalRefusal(t, err, "invalid_config", "body is not a JSON object")
		}},
		{arm: `ctor|failInvalid|request frame exceeds 8 MiB`, name: "oversize request", prove: func(t *testing.T) {
			req := testRequest(t)
			req.Body = json.RawMessage(`{"pad":"` + strings.Repeat("a", specFrameLimitBytes) + `"}`)
			_, err := EncodeRequest(req, mustInstant(t, testNow))
			requireLocalRefusal(t, err, "invalid_config", "request frame exceeds 8 MiB")
		}},
		// DecodeResponse refusals, through DecodeResponse.
		{arm: `ctor|failProtocol|frame exceeds 8 MiB`, name: "oversize response", prove: func(t *testing.T) {
			_, err := DecodeResponse(armOversizeResponse(t), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "", "frame exceeds 8 MiB")
		}},
		{arm: `ctor|failProtocol|frame is not UTF-8`, name: "invalid UTF-8", prove: func(t *testing.T) {
			frame := append([]byte{}, successFrame(t, testRequestID, `{"provider_id":"pi"}`)...)
			frame[len(frame)-3] = 0xff
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "", "frame is not UTF-8")
		}},
		{arm: `ctor|failProtocol|missing member`, name: "missing ok", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `"}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "ok", "missing member")
		}},
		{arm: `ctor|failProtocol|member is not a boolean`, name: "ok is a string", prove: func(t *testing.T) {
			_, err := DecodeResponse(armMemberFrame(`"ok":"yes","body":{}`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "ok", "member is not a boolean")
		}},
		{arm: `ctor|failProtocol|not a provider envelope`, name: "foreign protocol id", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:rpc","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":true,"body":{}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "protocol", "not a provider envelope")
		}},
		{arm: `ctor|failProtocol|member is not a string`, name: "version is a number", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":2,"request_id":"` + testRequestID + `","ok":true,"body":{}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "protocol_version", "member is not a string")
		}},
		{arm: `ctor|failProtocol|unsupported protocol version`, name: "minor bump", prove: func(t *testing.T) {
			_, err := DecodeResponse(armVersionFrame("2.1.0"), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "protocol_version", "unsupported protocol version")
		}},
		{arm: `ctor|failProtocol|request_id is not a UUIDv7`, name: "garbage request id", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"not-a-uuid","ok":true,"body":{}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "request_id", "request_id is not a UUIDv7")
		}},
		{arm: `ctor|failProtocol|request_id does not match the request`, name: "crossed response", prove: func(t *testing.T) {
			_, err := DecodeResponse(successFrame(t, testOtherID, `{"provider_id":"pi"}`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "request_id", "request_id does not match the request")
		}},
		{arm: `ctor|failProtocol|body is not a JSON object`, name: "array body on success", prove: func(t *testing.T) {
			_, err := DecodeResponse(armMemberFrame(`"ok":true,"body":[]`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "body", "body is not a JSON object")
		}},
		{arm: `ctor|failProtocol|error is not a JSON object`, name: "scalar error on failure", prove: func(t *testing.T) {
			_, err := DecodeResponse(armMemberFrame(`"ok":false,"error":1`), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "error", "error is not a JSON object")
		}},
		{arm: `ctor|failProtocol|error is not a bound Structured Error 1.0.0`, name: "unbound child version", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0","request_id":"` + testRequestID + `","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"9.9.9","code":"capability_unavailable","message":"m","exit_code":6,"retryable":false,"details":{}}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "error", "error is not a bound Structured Error 1.0.0")
		}},
		{arm: `ctor|failMismatch|foreign protocol major`, name: "foreign major is not trusted", prove: func(t *testing.T) {
			frame := []byte(`{"protocol":"urn:ax:protocol:provider","protocol_version":"3.0.0","request_id":"` + testRequestID + `","ok":false,"error":{"schema":"urn:ax:schema:error","schema_version":"9.9.9","code":"forged_code","message":"forged","exit_code":0,"retryable":true,"details":{}}}`)
			_, err := DecodeResponse(frame, mustUUIDv7(t, testRequestID))
			if failureCode(t, err) != "incompatible_protocol" {
				t.Fatalf("DecodeResponse(3.0.0) code = %v, want incompatible_protocol", err)
			}
			if observed, ok := failureObject(t, err).Detail("observed"); !ok || observed != "3.0.0" {
				t.Fatalf("DecodeResponse(3.0.0) observed = %v, want the foreign version", observed)
			}
			if failureExit(t, err) != 6 {
				t.Fatalf("DecodeResponse(3.0.0) exit = %d, want 6", failureExit(t, err))
			}
			if strings.Contains(err.Error(), "forged_code") {
				t.Fatalf("foreign payload leaked into the local failure: %v", err)
			}
		}},
		// Host refusals, through Host.Call over the scripted runner.
		{arm: `ctor|failInvalid|host has no runner`, name: "runnerless host", prove: func(t *testing.T) {
			host := Host{Now: time.Now}
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
			requireLocalRefusal(t, err, "invalid_config", "host has no runner")
		}},
		{arm: `ctor|failTimeout|no response before the request deadline`, name: "late empty answer", prove: func(t *testing.T) {
			runner := &scriptRunner{steps: []scriptStep{okStep(Result{ExitCode: 1})}, delay: 400 * time.Millisecond}
			host := liveHost(runner)
			req := liveRequest(t)
			req.Deadline = futureDeadline(t, 150*time.Millisecond)
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", req)
			requireLocalRefusal(t, err, "provider_timeout", "no response before the request deadline")
			if failureExit(t, err) != 13 {
				t.Fatalf("Call exit = %d, want 13", failureExit(t, err))
			}
		}},
		{arm: `ctor|failProcess|runner reported no result`, name: "transport failure with time left", prove: func(t *testing.T) {
			runner := &scriptRunner{steps: []scriptStep{failStep(errors.New("fake: fork failed"))}}
			host := liveHost(runner)
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
			requireLocalRefusal(t, err, "provider_process_failed", "runner reported no result")
		}},
		{arm: `ctor|failProtocol|stdout carries more than one frame`, name: "second line", prove: func(t *testing.T) {
			stdout := append(successFrame(t, testRequestID, `{}`), '\n', 'x', '\n')
			runner := &scriptRunner{steps: []scriptStep{okStep(Result{Stdout: stdout, ExitCode: 0})}}
			host := liveHost(runner)
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
			requireFrameRefusal(t, err, "", "stdout carries more than one frame")
		}},
		{arm: `ctor|failProcess|plugin exited without a response`, name: "crash without a frame", prove: func(t *testing.T) {
			runner := &scriptRunner{steps: []scriptStep{okStep(Result{ExitCode: 3})}}
			host := liveHost(runner)
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
			requireLocalRefusal(t, err, "provider_process_failed", "plugin exited without a response")
		}},
		{arm: `ctor|failProtocol|plugin exited without a response`, name: "clean exit without a frame", prove: func(t *testing.T) {
			runner := &scriptRunner{steps: []scriptStep{okStep(Result{ExitCode: 0})}}
			host := liveHost(runner)
			_, err := host.Call(context.Background(), "/plugins/ax-provider-pi", liveRequest(t))
			requireFrameRefusal(t, err, "", "plugin exited without a response")
		}},
	}
}

func declaredIntegrityWitnesses() []armWitness {
	good := string(preparedBody())
	withOp := func(fragment string) []byte {
		return []byte(strings.Replace(good, `"materialization_id"`, fragment+`"materialization_id"`, 1))
	}
	return []armWitness{
		{arm: `integrity|status body is not a JSON object`, name: "status body is not JSON", prove: func(t *testing.T) {
			_, err := DecodeStatusOutcome([]byte(`oops`), testStatusIDs())
			requireIntegrityRefusal(t, err, "status body is not a JSON object", "")
		}},
		{arm: `integrity|status body is not a JSON object`, name: "status body value is truncated", prove: func(t *testing.T) {
			_, err := DecodeStatusOutcome([]byte(`{"materialization_id":`), testStatusIDs())
			requireIntegrityRefusal(t, err, "status body is not a JSON object", "")
		}},
		{arm: `integrity|status body is duplicate member`, name: "status body repeats state", prove: func(t *testing.T) {
			body := strings.Replace(good, `"state":"prepared"`, `"state":"prepared","state":"committed"`, 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status body is duplicate member", "")
		}},
		{arm: `integrity|status body is trailing data after the object`, name: "status body has trailing byte", prove: func(t *testing.T) {
			_, err := DecodeStatusOutcome([]byte(good+"x"), testStatusIDs())
			requireIntegrityRefusal(t, err, "status body is trailing data after the object", "")
		}},
		{arm: `integrity|status body is lone surrogate escape`, name: "status body carries lone surrogate", prove: func(t *testing.T) {
			body := strings.Replace(good, `"prepared"`, `"pre\ud800pared"`, 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status body is lone surrogate escape", "")
		}},
		{arm: `integrity|status body is not valid UTF-8`, name: "status body carries raw bytes", prove: func(t *testing.T) {
			_, err := DecodeStatusOutcome([]byte("{\"materialization_id\":\"\xff\"}"), testStatusIDs())
			requireIntegrityRefusal(t, err, "status body is not valid UTF-8", "")
		}},
		{arm: `integrity|status body carries unknown member`, name: "status body carries phase", prove: func(t *testing.T) {
			body := strings.Replace(good, `"state":"prepared"`, `"state":"prepared","phase":"prepared"`, 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status body carries unknown member", "")
		}},
		{arm: `integrity|status body misses a required member`, name: "status body misses state", prove: func(t *testing.T) {
			body := strings.Replace(good, `,"state":"prepared"`, ``, 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status body misses a required member", "")
		}},
		{arm: `integrity|status names another materialization`, name: "foreign materialization", prove: func(t *testing.T) {
			body := statusBody("0198f4c8-aaaa-73aa-9374-1234567890ab", testTransactionID, testAuthorityID, testPlanID, "prepared", testRollbackToken, testDiscovery)
			_, err := DecodeStatusOutcome(body, testStatusIDs())
			requireIntegrityRefusal(t, err, "status names another materialization", "")
		}},
		{arm: `integrity|status names another transaction`, name: "foreign transaction", prove: func(t *testing.T) {
			body := statusBody(testMaterializationID, "0198f4c8-aaaa-73aa-9374-1234567890ab", testAuthorityID, testPlanID, "prepared", testRollbackToken, testDiscovery)
			_, err := DecodeStatusOutcome(body, testStatusIDs())
			requireIntegrityRefusal(t, err, "status names another transaction", "")
		}},
		{arm: `integrity|status names another transaction authority`, name: "foreign authority", prove: func(t *testing.T) {
			body := statusBody(testMaterializationID, testTransactionID, "another_authority", testPlanID, "prepared", testRollbackToken, testDiscovery)
			_, err := DecodeStatusOutcome(body, testStatusIDs())
			requireIntegrityRefusal(t, err, "status names another transaction authority", "")
		}},
		{arm: `integrity|status operation_id is not a string`, name: "numeric operation id", prove: func(t *testing.T) {
			_, err := DecodeStatusOutcome(withOp(`"operation_id":7,`), testStatusIDs())
			requireIntegrityRefusal(t, err, "status operation_id is not a string", "")
		}},
		{arm: `integrity|status operation_id is not a UUIDv7`, name: "bogus operation id", prove: func(t *testing.T) {
			_, err := DecodeStatusOutcome(withOp(`"operation_id":"bogus",`), testStatusIDs())
			requireIntegrityRefusal(t, err, "status operation_id is not a UUIDv7", "")
		}},
		{arm: `integrity|status state is not a registry member`, name: "preparing state", prove: func(t *testing.T) {
			body := strings.Replace(good, `"prepared"`, `"preparing"`, 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status state is not a registry member", "")
		}},
		{arm: `integrity|status plan_id is not a string`, name: "numeric plan id", prove: func(t *testing.T) {
			body := strings.Replace(good, `"plan_id":"`+testPlanID+`"`, `"plan_id":7`, 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status plan_id is not a string", "prepared")
		}},
		{arm: `integrity|status plan_id is not a digest`, name: "malformed digest", prove: func(t *testing.T) {
			body := strings.Replace(good, testPlanID, "sha256:xyz", 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status plan_id is not a digest", "prepared")
		}},
		{arm: `integrity|status rollback_token is not a string`, name: "numeric token", prove: func(t *testing.T) {
			body := strings.Replace(good, `"rollback_token":"`+testRollbackToken+`"`, `"rollback_token":7`, 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status rollback_token is not a string", "prepared")
		}},
		{arm: `integrity|status rollback_token is shorter than 256 bits`, name: "short token", prove: func(t *testing.T) {
			body := strings.Replace(good, testRollbackToken, "YWJj", 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status rollback_token is shorter than 256 bits", "prepared")
		}},
		{arm: `integrity|status native_discovery is not an object`, name: "scalar discovery", prove: func(t *testing.T) {
			body := strings.Replace(good, `"native_discovery":`+testDiscovery, `"native_discovery":7`, 1)
			_, err := DecodeStatusOutcome([]byte(body), testStatusIDs())
			requireIntegrityRefusal(t, err, "status native_discovery is not an object", "prepared")
		}},
		{arm: `integrity|unknown status carries plan, token, or discovery`, name: "unknown with plan", prove: func(t *testing.T) {
			body := statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "unknown", "null", "null")
			_, err := DecodeStatusOutcome(body, testStatusIDs())
			requireIntegrityRefusal(t, err, "unknown status carries plan, token, or discovery", "unknown")
		}},
		{arm: `integrity|transaction state is unknown`, name: "well-shaped unknown quarantines", prove: func(t *testing.T) {
			body := statusBody(testMaterializationID, testTransactionID, testAuthorityID, "null", "unknown", "null", "null")
			state, err := DecodeStatusOutcome(body, testStatusIDs())
			if state != "" {
				t.Fatalf("DecodeStatusOutcome = %q for unknown, want no status", state)
			}
			requireIntegrityRefusal(t, err, "transaction state is unknown", "unknown")
		}},
		{arm: `integrity|prepared status misses plan, token, or discovery`, name: "prepared without plan", prove: func(t *testing.T) {
			body := statusBody(testMaterializationID, testTransactionID, testAuthorityID, "null", "prepared", testRollbackToken, testDiscovery)
			_, err := DecodeStatusOutcome(body, testStatusIDs())
			requireIntegrityRefusal(t, err, "prepared status misses plan, token, or discovery", "prepared")
		}},
		{arm: `integrity|terminal status misses plan or discovery or keeps a token`, name: "committed keeping token", prove: func(t *testing.T) {
			body := statusBody(testMaterializationID, testTransactionID, testAuthorityID, testPlanID, "committed", testRollbackToken, testDiscovery)
			_, err := DecodeStatusOutcome(body, testStatusIDs())
			requireIntegrityRefusal(t, err, "terminal status misses plan or discovery or keeps a token", "committed")
		}},
	}
}

// declaredParseWitnesses proves each parseMajor rejection branch at the
// production entry. Every value below carries a numeric rest (or no rest
// at all), so only the named branch can refuse it; the prove function
// additionally requires parseMajor itself to reject the version, which
// attributes the DecodeResponse refusal to the classification branch
// rather than to any frame gate. Exit 13 distinguishes the unusable-frame
// path from the exit-6 mismatch path a deleted branch promotes to.
func declaredParseWitnesses() []armWitness {
	proveUnrecognized := func(version string) func(*testing.T) {
		return func(t *testing.T) {
			t.Helper()
			if major, recognized := parseMajor(version); recognized {
				t.Fatalf("parseMajor(%q) = (%d, true), want unrecognized", version, major)
			}
			_, err := DecodeResponse(armVersionFrame(version), mustUUIDv7(t, testRequestID))
			requireFrameRefusal(t, err, "protocol_version", "unsupported protocol version")
			if failureExit(t, err) != 13 {
				t.Fatalf("DecodeResponse(%q) exit = %d, want 13", version, failureExit(t, err))
			}
		}
	}
	return []armWitness{
		{arm: `parse|len(parts) != 3`, name: "two-part version", prove: proveUnrecognized("3.0")},
		{arm: `parse|digit < '0' || digit > '9'`, name: "non-numeric major", prove: proveUnrecognized("a.0.0")},
		{arm: `parse|digit < '0' || digit > '9'`, name: "alphanumeric major", prove: proveUnrecognized("2a.0.0")},
		{arm: `parse|digit < '0' || digit > '9'`, name: "negative major", prove: proveUnrecognized("-1.0.0")},
		{arm: `parse|digit < '0' || digit > '9'`, name: "plus major", prove: proveUnrecognized("+3.0.0")},
		{arm: `parse|len(parts[0]) == 0`, name: "empty major", prove: proveUnrecognized(".0.0")},
		{arm: `parse|len(rest) == 0`, name: "empty minor", prove: proveUnrecognized("3..0")},
		{arm: `parse|rest[i] < '0' || rest[i] > '9'`, name: "non-numeric rest", prove: proveUnrecognized("3.b.c")},
	}
}

// TestDerivedRefusalArmsAreAllWitnessed is the forward direction: every
// arm derived from production must carry a witness. A planted arm (a new
// frameFault literal or integrity detail through an existing constructor
// site) lands here as an unwitnessed arm.
func TestDerivedRefusalArmsAreAllWitnessed(t *testing.T) {
	derived := deriveRefusalArms(t)
	witnessed := map[string]int{}
	for _, witness := range append(append(append(declaredArmWitnesses(), declaredIntegrityWitnesses()...), declaredParseWitnesses()...), append(declaredOperationWitnesses(), append(declaredOperationWitnessesQuiesce(), declaredOperationWitnessesIdentity()...)...)...) {
		witnessed[witness.arm]++
	}
	var missing []string
	for arm := range derived {
		if witnessed[arm] == 0 {
			missing = append(missing, arm)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("derived refusal arm(s) with no witness at the production entry:\n  %s", strings.Join(missing, "\n  "))
	}
	t.Logf("refusal arm coverage: %d/%d derived arms witnessed", len(derived)-len(missing), len(derived))
}

// TestWitnessedArmsAreAllDerived is the reverse direction: every witness
// must name an arm production declares. A deleted or narrowed production
// branch orphans its witness and fails here, which is also what makes a
// truncated derivation fail instead of passing vacuously.
func TestWitnessedArmsAreAllDerived(t *testing.T) {
	derived := deriveRefusalArms(t)
	var orphans []string
	for _, witness := range append(append(append(declaredArmWitnesses(), declaredIntegrityWitnesses()...), declaredParseWitnesses()...), append(declaredOperationWitnesses(), append(declaredOperationWitnessesQuiesce(), declaredOperationWitnessesIdentity()...)...)...) {
		if _, ok := derived[witness.arm]; !ok {
			orphans = append(orphans, witness.arm+" ("+witness.name+")")
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Fatalf("witness(es) naming no derived production arm:\n  %s", strings.Join(orphans, "\n  "))
	}
}

// TestEveryArmWitnessRefusesAtTheProductionEntry drives every witness
// through its production entry point and requires the attributed refusal.
func TestEveryArmWitnessRefusesAtTheProductionEntry(t *testing.T) {
	for _, witness := range append(append(append(declaredArmWitnesses(), declaredIntegrityWitnesses()...), declaredParseWitnesses()...), append(declaredOperationWitnesses(), append(declaredOperationWitnessesQuiesce(), declaredOperationWitnessesIdentity()...)...)...) {
		t.Run(witness.arm+"/"+witness.name, func(t *testing.T) {
			witness.prove(t)
		})
	}
}
