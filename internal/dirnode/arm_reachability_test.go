package dirnode

import (
	"go/ast"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// This file is the derived arm-reachability gate (review round 3,
// C1). The arm census in census_test.go derives every refusal arm
// from production source and maps each to witness tests, but it
// never checks that a witness actually fires the arm it names: a
// vector carrying two defects is satisfied by whichever arm fires
// first, and three committed "unknown member" vectors proved this
// by reaching a strict-decode, operation, and node-build arm while
// naming the envelope unknown-member arm. A code-only assertion
// cannot tell those paths apart, and a deleted-member vector
// refuses downstream with the same code, so code-only witnesses
// leave missing-member, extensions, and empty-frame arms deletable
// with a green suite.
//
// The gate below closes the class without another hand audit.
// Every row pairs one production-derived arm key with a vector
// driven through a production entry, and the test computes which
// arm the vector actually reaches from the refusal message:
//
//   - a literal arm (ctor|<ctor>|<detail>) matches when the message
//     carries its full detail; the row passes only when exactly the
//     named literal matches, so a vector that slides to a sibling
//     arm or to the strict conduit fails naming the arm it reached;
//   - a fault conduit (..."<prefix> "+fault.detail) or a
//     context-prefixed arm (context+"<suffix>") matches on its
//     derived prefix/suffix; the row passes only when the named arm
//     matches, no literal matches, and every other match is at most
//     one frame fault travelling the same conduit;
//   - a frame fault (frame|<detail>) matches on its detail and pins
//     its carrier conduit through the also key, so the fault and
//     the entry that carried it are both derived.
//
// A row whose reached arm cannot be determined — no match, or a
// match set outside the shapes above — fails, never passes. The
// roster fails closed in both directions: an arm with no row fails
// unless it carries a defensive rationale in defensiveArms, and a
// row naming an arm production no longer derives fails as
// orphaned. Expected messages are never retyped here: they come
// from the derived keys, so a reworded production arm fails its
// row instead of passing against a stale copy.
//
// The checkResponseIdentity context arms instantiate with the call
// site's context word. TestReachabilityContextsAreClosed derives
// the context set from production calls and requires exactly
// success and failure, so the matcher below cannot silently miss a
// third instantiation.

// armReachRow pairs one derived arm key with the vector that fires
// it. also optionally names a second derived key whose text must
// also match: frame-fault rows pin the carrier conduit that way.
type armReachRow struct {
	arm   string
	name  string
	drive func(t *testing.T) error
	also  string
}

// rowName renders the subtest name for one row: the arm key,
// entry-suffixed for the one key emitted at two production sites.
func (row armReachRow) rowName() string {
	if row.name != "" {
		return row.name
	}
	return row.arm
}

// deriveLiteralArmSiteCounts counts production call sites per
// literal arm key. Most literals are emitted once; a literal
// emitted at two sites shares one arm key, and the roster below
// requires one firing vector per emitting site — a row driving
// the other site cannot redden when this one is removed.
func deriveLiteralArmSiteCounts(t *testing.T) map[string]int {
	t.Helper()
	registered := map[string]bool{}
	for _, name := range refusalConstructors {
		registered[name] = true
	}
	counts := map[string]int{}
	for _, path := range productionFiles(t) {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("site counts: %v", err)
		}
		syntax, err := parseGoFile(path, source)
		if err != nil {
			t.Fatalf("site counts: %v", err)
		}
		ast.Inspect(syntax, func(node ast.Node) bool {
			function, ok := node.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				return true
			}
			ast.Inspect(function.Body, func(inner ast.Node) bool {
				call, ok := inner.(*ast.CallExpr)
				if !ok {
					return true
				}
				target, ok := call.Fun.(*ast.Ident)
				if !ok || !registered[target.Name] || len(call.Args) == 0 {
					return true
				}
				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok {
					return true
				}
				counts[target.Name+"|"+strings.Trim(literal.Value, `"`)]++
				return true
			})
			return false
		})
	}
	return counts
}

// literalSubkey renders the site-count lookup key for one derived
// literal arm key.
func literalSubkey(key string) (string, bool) {
	rest, ok := strings.CutPrefix(key, "ctor|")
	if !ok {
		return "", false
	}
	separator := strings.Index(rest, "|")
	if separator < 0 || rest[separator+1:] == "" {
		return "", false
	}
	return rest[:separator] + "|" + rest[separator+1:], true
}

// quotedInner returns the text between the first pair of double
// quotes in a rendered derivation expression: the literal prefix
// of a fault conduit or the suffix of a context arm.
func quotedInner(render string) string {
	first := strings.Index(render, `"`)
	if first < 0 {
		return ""
	}
	second := strings.Index(render[first+1:], `"`)
	if second < 0 {
		return ""
	}
	return render[first+1 : first+1+second]
}

// ctorEnvelope is the derived refusal envelope of one error
// constructor: the wire code and the message wrapper every arm
// built through it carries.
type ctorEnvelope struct {
	code    string
	wrapper string
}

// deriveCtorEnvelopes derives the wire code and message wrapper of
// every registered refusal constructor from production source.
// Every constructor must hold exactly one axerror.New call shaped
// as Spec{Version: _, Code: "<literal>", Message: "<literal>" +
// <first-param>, ...}: any other construction shape is a
// derivation violation, never a silent pass, so a constructor
// that stops prefixing its detail fails here instead of matching
// nothing anywhere.
func deriveCtorEnvelopes(t *testing.T) map[string]ctorEnvelope {
	t.Helper()
	registered := map[string]bool{}
	for _, name := range refusalConstructors {
		registered[name] = true
	}
	envelopes := map[string]ctorEnvelope{}
	var violations []string
	for _, path := range productionFiles(t) {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("envelopes: %v", err)
		}
		syntax, err := parseGoFile(path, source)
		if err != nil {
			t.Fatalf("envelopes: %v", err)
		}
		for _, declaration := range syntax.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, specification := range general.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok || len(value.Values) != 1 {
					continue
				}
				literal, ok := value.Values[0].(*ast.FuncLit)
				if !ok {
					continue
				}
				for _, name := range value.Names {
					if !registered[name.Name] {
						continue
					}
					calls := 0
					ast.Inspect(literal.Body, func(node ast.Node) bool {
						call, ok := node.(*ast.CallExpr)
						if !ok {
							return true
						}
						target, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}
						packageName, ok := target.X.(*ast.Ident)
						if !ok || packageName.Name != "axerror" || target.Sel.Name != "New" {
							return true
						}
						calls++
						if len(call.Args) != 1 {
							violations = append(violations, name.Name+": axerror.New arity is not 1")
							return true
						}
						composite, ok := call.Args[0].(*ast.CompositeLit)
						if !ok {
							violations = append(violations, name.Name+": axerror.New argument is not a composite literal")
							return true
						}
						var code, wrapper string
						var detailIdent bool
						for _, element := range composite.Elts {
							pair, ok := element.(*ast.KeyValueExpr)
							if !ok {
								continue
							}
							field, ok := pair.Key.(*ast.Ident)
							if !ok {
								continue
							}
							switch field.Name {
							case "Code":
								lit, ok := pair.Value.(*ast.BasicLit)
								if ok && lit.Kind.String() == "STRING" {
									code = strings.Trim(lit.Value, `"`)
								}
							case "Message":
								binary, ok := pair.Value.(*ast.BinaryExpr)
								if !ok {
									break
								}
								lit, ok := binary.X.(*ast.BasicLit)
								if !ok || lit.Kind.String() != "STRING" {
									break
								}
								if _, ok := binary.Y.(*ast.Ident); !ok {
									break
								}
								wrapper = strings.Trim(lit.Value, `"`)
								detailIdent = true
							}
						}
						if code == "" || !detailIdent {
							violations = append(violations, name.Name+": envelope is not Code:<literal> with Message:<literal>+detail")
							return true
						}
						envelopes[name.Name] = ctorEnvelope{code: code, wrapper: wrapper}
						return true
					})
					if calls != 1 {
						violations = append(violations, name.Name+": envelope holds neither exactly one axerror.New call")
					}
				}
			}
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("envelope violations:\n  %s", strings.Join(violations, "\n  "))
	}
	if len(envelopes) != len(registered) {
		t.Fatalf("derived envelopes = %d, want %d registered constructors", len(envelopes), len(registered))
	}
	return envelopes
}

// armKeyShape classifies one derived arm key for matching.
type armKeyShape int

const (
	armShapeUnknown armKeyShape = iota
	armShapeLiteral
	armShapeConduit
	armShapeContext
	armShapeContextFault
	armShapeFrame
)

// classifyArmKey splits one derived key into its match shape and
// match text. The second result is false for a shape this gate
// does not understand: the row fails instead of passing
// vacuously.
func classifyArmKey(key string) (armKeyShape, string, bool) {
	if rest, ok := strings.CutPrefix(key, "frame|"); ok && rest != "" {
		return armShapeFrame, rest, true
	}
	// Only the first two separators are structural: literal
	// details may carry their own "|" (amd64|arm64), so the
	// detail is everything after ctor|<constructor>|.
	rest, ok := strings.CutPrefix(key, "ctor|")
	if !ok {
		return armShapeUnknown, "", false
	}
	separator := strings.Index(rest, "|")
	if separator < 0 {
		return armShapeUnknown, "", false
	}
	payload := rest[separator+1:]
	if payload == "" {
		return armShapeUnknown, "", false
	}
	if expr, ok := strings.CutPrefix(payload, "expr|"); ok {
		separator := strings.Index(expr, "|")
		if separator < 0 {
			return armShapeUnknown, "", false
		}
		render := expr[separator+1:]
		if strings.HasPrefix(render, "context+") {
			inner := quotedInner(render)
			if inner == "" {
				return armShapeUnknown, "", false
			}
			if strings.Contains(render, "fault.detail") {
				return armShapeContextFault, inner, true
			}
			return armShapeContext, inner, true
		}
		if strings.Contains(render, "fault.detail") {
			prefix := quotedInner(render)
			if prefix == "" {
				return armShapeUnknown, "", false
			}
			return armShapeConduit, prefix, true
		}
		return armShapeUnknown, "", false
	}
	return armShapeLiteral, payload, true
}

// armExpectedTexts renders every full expected text one derived
// arm can report: the derived wire code, the derived constructor
// wrapper, and the arm's own derived text. A fault conduit needs
// one of the derived frame details alongside its prefix before it
// matches, so a literal message that merely shares the entry
// prefix never resolves to the conduit. Context arms render both
// production instantiations;
// TestReachabilityContextsAreClosed pins those two as the only
// ones. Frame faults carry no envelope: they match on detail
// alone and pin their carrier through the row's also key.
func armExpectedTexts(key string, envelopes map[string]ctorEnvelope, frameDetails []string) []string {
	shape, text, ok := classifyArmKey(key)
	if !ok {
		return nil
	}
	if shape == armShapeFrame {
		return []string{text}
	}
	rest, ok := strings.CutPrefix(key, "ctor|")
	if !ok {
		return nil
	}
	separator := strings.Index(rest, "|")
	if separator < 0 {
		return nil
	}
	envelope, ok := envelopes[rest[:separator]]
	if !ok {
		return nil
	}
	prefix := envelope.code + ": " + envelope.wrapper
	switch shape {
	case armShapeLiteral:
		return []string{prefix + text}
	case armShapeConduit:
		var texts []string
		for _, detail := range frameDetails {
			texts = append(texts, prefix+text+detail)
		}
		return texts
	case armShapeContext:
		return []string{prefix + "success" + text, prefix + "failure" + text}
	case armShapeContextFault:
		var texts []string
		for _, context := range []string{"success", "failure"} {
			for _, detail := range frameDetails {
				texts = append(texts, prefix+context+text+detail)
			}
		}
		return texts
	}
	return nil
}

// frameDetailTexts collects every derived frame-fault detail: the
// fault vocabulary the conduit matchers close over.
func frameDetailTexts(derived armDerivation) []string {
	var details []string
	for key := range derived.arms {
		if rest, ok := strings.CutPrefix(key, "frame|"); ok && rest != "" {
			details = append(details, rest)
		}
	}
	sort.Strings(details)
	return details
}

// armKeyMatches reports whether the refusal message carries any of
// the arm's derived expected texts.
func armKeyMatches(key string, envelopes map[string]ctorEnvelope, frameDetails []string, message string) bool {
	for _, expected := range armExpectedTexts(key, envelopes, frameDetails) {
		if strings.Contains(message, expected) {
			return true
		}
	}
	return false
}

// reachedArm computes which derived arm a refusal message
// reaches and whether that outcome is determined:
//
//   - a literal arm is reached only when exactly it matches among
//     all literal keys: a vector sliding to a sibling arm, or to
//     the strict conduit, names a key outside the match and fails;
//   - a conduit or context arm is reached when it matches, no
//     literal matches, and anything else matching is at most one
//     frame fault travelling the same conduit;
//   - a frame fault is reached when its detail matches, no literal
//     matches, and the carrier conduit named by the row's also key
//     matches alongside it.
//
// Anything else — no match, an ambiguous match, or a match the
// shapes above do not account for — returns ok=false, and the row
// fails instead of passing.
func reachedArm(named string, matches map[string]bool, also string, envelopes map[string]ctorEnvelope, frameDetails []string, message string) (string, bool) {
	shape, _, ok := classifyArmKey(named)
	if !ok {
		return "", false
	}
	switch shape {
	case armShapeLiteral:
		literals := 0
		for key := range matches {
			keyShape, _, ok := classifyArmKey(key)
			if !ok {
				return "", false
			}
			if keyShape == armShapeLiteral {
				literals++
			}
		}
		if literals == 1 && matches[named] {
			return named, true
		}
		for key := range matches {
			keyShape, _, ok := classifyArmKey(key)
			if ok && keyShape == armShapeLiteral {
				return key, false
			}
		}
		return "", false
	case armShapeFrame:
		if also == "" || !matches[also] {
			return "", false
		}
		for key := range matches {
			keyShape, _, ok := classifyArmKey(key)
			if !ok {
				return "", false
			}
			if keyShape == armShapeLiteral {
				return key, false
			}
			if key != named && key != also {
				return key, false
			}
		}
		if matches[named] {
			return named, true
		}
		return "", false
	default:
		if !matches[named] {
			for key := range matches {
				keyShape, _, ok := classifyArmKey(key)
				if ok && (keyShape == armShapeLiteral || keyShape == armShapeFrame) {
					return key, false
				}
			}
			return "", false
		}
		frames := 0
		for key := range matches {
			if key == named {
				continue
			}
			keyShape, _, ok := classifyArmKey(key)
			if !ok {
				return "", false
			}
			if keyShape == armShapeLiteral {
				return key, false
			}
			if keyShape == armShapeFrame {
				frames++
				continue
			}
			// A sibling conduit or context arm is allowed only
			// when it adds no matched text beyond the named
			// arm's own: two production sites emitting
			// byte-identical refusals (the request-body
			// conduits in DecodeRequestFrame and EncodeRequest)
			// are indistinguishable by message, and each row
			// still dies when its own entry's arm is removed.
			// Anything textually distinct fails as a wrong arm.
			if !matchedTextSubsetOf(key, named, envelopes, frameDetails, message) {
				return key, false
			}
		}
		if frames <= 1 {
			return named, true
		}
		return "", false
	}
}

// matchedTextSubsetOf reports whether every derived expected text
// of other carried by the message is also a derived expected text
// of named. Textually indistinguishable sibling arms pass; a
// sibling contributing its own distinct text fails the row.
func matchedTextSubsetOf(other, named string, envelopes map[string]ctorEnvelope, frameDetails []string, message string) bool {
	allowed := map[string]bool{}
	for _, text := range armExpectedTexts(named, envelopes, frameDetails) {
		allowed[text] = true
	}
	for _, text := range armExpectedTexts(other, envelopes, frameDetails) {
		if strings.Contains(message, text) && !allowed[text] {
			return false
		}
	}
	return true
}

// matchAllKeys returns the set of derived keys with at least one
// derived expected text carried by the message.
func matchAllKeys(derived armDerivation, envelopes map[string]ctorEnvelope, frameDetails []string, message string) map[string]bool {
	matched := map[string]bool{}
	for key := range derived.arms {
		if armKeyMatches(key, envelopes, frameDetails, message) {
			matched[key] = true
		}
	}
	return matched
}

// TestReachabilityContextsAreClosed derives every
// checkResponseIdentity call's context word from production source
// and requires exactly success and failure: the context-arm
// matcher resolves only those two, so a third instantiation fails
// here instead of matching nothing anywhere.
func TestReachabilityContextsAreClosed(t *testing.T) {
	t.Parallel()
	contexts := map[string]bool{}
	for _, path := range productionFiles(t) {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("contexts: %v", err)
		}
		syntax, err := parseGoFile(path, source)
		if err != nil {
			t.Fatalf("contexts: %v", err)
		}
		ast.Inspect(syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			target, ok := call.Fun.(*ast.Ident)
			if !ok || target.Name != "checkResponseIdentity" {
				return true
			}
			if len(call.Args) != 3 {
				t.Fatalf("checkResponseIdentity arity changed at %s", path)
			}
			literal, ok := call.Args[2].(*ast.BasicLit)
			if !ok {
				t.Fatalf("checkResponseIdentity context at %s is not a literal; the matcher cannot resolve it", path)
			}
			contexts[strings.Trim(literal.Value, `"`)] = true
			return true
		})
	}
	if len(contexts) != 2 || !contexts["success"] || !contexts["failure"] {
		t.Fatalf("response identity contexts = %v, want exactly success and failure", contexts)
	}
}

// TestEveryDerivedArmFiresItsVector drives every derived refusal
// arm through a production entry and requires the refusal message
// to resolve to the named arm. Deleting the arm changes the
// message (a downstream arm reports its own text) or admits the
// vector outright, so either outcome fails the row: the witness
// discriminates by message, never by code alone.
func TestEveryDerivedArmFiresItsVector(t *testing.T) {
	t.Parallel()
	derived := deriveRefusalArms(t)
	if len(derived.violations) > 0 {
		sort.Strings(derived.violations)
		t.Fatalf("census violations:\n  %s", strings.Join(derived.violations, "\n  "))
	}
	envelopes := deriveCtorEnvelopes(t)
	frameDetails := frameDetailTexts(derived)
	if len(frameDetails) == 0 {
		t.Fatal("derived zero frame-fault details; the conduit matchers close over nothing")
	}
	rows := append(append(append(armReachabilityRows(), armReachabilityScanRows()...), armReachabilityQueryRows()...), armReachabilityBootstrapRows()...)
	siteCounts := deriveLiteralArmSiteCounts(t)
	rowsPerKey := map[string]int{}
	for _, row := range rows {
		if !derived.arms[row.arm] {
			t.Fatalf("orphaned reachability row %q: production derives no such arm", row.arm)
		}
		if row.also != "" && !derived.arms[row.also] {
			t.Fatalf("orphaned reachability carrier %q for row %q", row.also, row.arm)
		}
		if _, _, ok := classifyArmKey(row.arm); !ok {
			t.Fatalf("unmatchable reachability row %q: unknown key shape", row.arm)
		}
		rowsPerKey[row.arm]++
	}
	for key := range derived.arms {
		shape, _, ok := classifyArmKey(key)
		if !ok {
			t.Fatalf("unmatchable derived arm %q: unknown key shape", key)
		}
		if _, defensive := defensiveArms[key]; defensive {
			continue
		}
		want := 1
		if shape == armShapeLiteral {
			subkey, ok := literalSubkey(key)
			if !ok {
				t.Fatalf("unkeyable derived arm %q", key)
			}
			sites, ok := siteCounts[subkey]
			if !ok || sites < 1 {
				t.Fatalf("derived arm %q has no counted production site; the counter is broken, not the package", key)
			}
			want = sites
		}
		if rowsPerKey[key] != want {
			t.Fatalf("derived arm %q carries %d rows, want one per emitting site (%d)", key, rowsPerKey[key], want)
		}
	}
	for _, row := range rows {
		row := row
		t.Run(row.rowName(), func(t *testing.T) {
			t.Parallel()
			err := row.drive(t)
			failure, ok := err.(*axerror.Error)
			if !ok {
				t.Fatalf("row %q: want *axerror.Error, got %T (%v); the vector admits or bypasses the arm", row.arm, err, err)
			}
			// Match on the full rendering: the derived expected
			// texts carry the wire code prefix, which Message
			// alone omits.
			rendered := failure.Error()
			matches := matchAllKeys(derived, envelopes, frameDetails, rendered)
			reached, determined := reachedArm(row.arm, matches, row.also, envelopes, frameDetails, rendered)
			if !determined || reached != row.arm {
				var matched []string
				for key := range matches {
					matched = append(matched, key)
				}
				sort.Strings(matched)
				t.Fatalf("row %q: message %q reaches %q, not the named arm (matches %v)", row.arm, failure.Message(), reached, matched)
			}
		})
	}
}

// reachRequest decodes the valid v2 manifest request every
// envelope test issues its response under.
func reachRequest(t *testing.T) Request {
	t.Helper()
	request, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 5000, `{}`))
	if err != nil {
		t.Fatalf("reachRequest: %v", err)
	}
	return request
}

// reachFrame renders one valid v2 manifest request frame for
// envelope surgery.
func reachFrame(t *testing.T) string {
	t.Helper()
	return string(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`))
}

// armReachabilityRows pairs every fireable derived arm with its
// firing vector. Vectors are hand-built but execution-verified:
// the test above computes the reached arm from the refusal
// message, so a vector that slides to another arm fails its row.
func armReachabilityRows() []armReachRow {
	return []armReachRow{
		// Request frame entry.
		{arm: "ctor|failViolation|request frame is empty", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte{})
			return err
		}},
		{arm: "ctor|failViolation|request frame exceeds the 8 MiB bound", drive: func(t *testing.T) error {
			oversize := append(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `{}`), make([]byte, frameBoundBytes)...)
			_, err := DecodeRequestFrame(oversize)
			return err
		}},
		{arm: `ctor|failViolation|expr|DecodeRequestFrame|"request envelope "+fault.detail`, drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(reachFrame(t) + " {}"))
			return err
		}},
		{arm: "ctor|failViolation|request envelope carries unknown member", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(replaceOnce(t, reachFrame(t), `"deadline_ms":1`, `"deadline_ms":1,"extra":1`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|request envelope misses a required member", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(`{"schema":"urn:ax:schema:session-directory-node-request","schema_version":"2.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":"2.0.0","request_id":"` + fixtureRequestID + `","operation":"manifest","body":{}}`))
			return err
		}},
		{arm: "ctor|failViolation|request schema is not the directory node request", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(replaceOnce(t, reachFrame(t), "session-directory-node-request", "session-directory-node-response", 1)))
			return err
		}},
		{arm: "ctor|failViolation|request protocol version is not a string", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(replaceOnce(t, reachFrame(t), `"protocol_version":"2.0.0"`, `"protocol_version":2`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|request schema version is not a string", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(replaceOnce(t, reachFrame(t), `"schema_version":"2.0.0"`, `"schema_version":2`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|request protocol is not the directory node", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(replaceOnce(t, reachFrame(t), `"protocol":"urn:ax:protocol:session-directory-node"`, `"protocol":"urn:ax:protocol:provider"`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|request identifier is not a string", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(replaceOnce(t, reachFrame(t), `"request_id":"`+fixtureRequestID+`"`, `"request_id":7`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|request identifier is not a UUIDv7", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", "not-a-uuid", 1, `{}`))
			return err
		}},
		{arm: "ctor|failViolation|request operation is not a string", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(replaceOnce(t, reachFrame(t), `"operation":"manifest"`, `"operation":7`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|request versions do not bind one supported major", drive: func(t *testing.T) error {
			raw := replaceOnce(t, reachFrame(t), `"protocol_version":"2.0.0"`, `"protocol_version":"1.0.0"`, 1)
			_, err := DecodeRequestFrame([]byte(raw))
			return err
		}},
		{arm: "ctor|failViolation|request deadline is not uint53[1..3600000]", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 0, `{}`))
			return err
		}},
		{arm: `ctor|failViolation|expr|DecodeRequestFrame|"request body "+fault.detail`, drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "manifest", fixtureRequestID, 1, `[]`))
			return err
		}},
		{arm: "ctor|failUnknownOperation|dispatch names an operation outside the closed registry", drive: func(t *testing.T) error {
			return RefuseUnknownOperation("query")
		}},
		{arm: "ctor|failUnknownOperation|request names an operation outside the closed registry", drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "query", fixtureRequestID, 1, `{}`))
			return err
		}},
		{arm: "ctor|failUnknownOperation|request names an operation outside the closed registry", name: "ctor|failUnknownOperation|request names an operation outside the closed registry|via EncodeRequest", drive: func(t *testing.T) error {
			// Second emitting site for this key: the builder
			// entry. The decode-driven row above cannot fire
			// when this site is removed — same text, other
			// function — so the key carries one row per site.
			_, err := EncodeRequest(MajorV2, "query", fixtureRequestID, 1, []byte(`{}`))
			return err
		}},
		// Request builder entry.
		{arm: "ctor|failInvalid|request major is outside the locally supported registry", drive: func(t *testing.T) error {
			_, err := EncodeRequest(3, OpManifest, fixtureRequestID, 1, []byte(`{}`))
			return err
		}},
		{arm: "ctor|failInvalid|request identifier is not a UUIDv7", drive: func(t *testing.T) error {
			_, err := EncodeRequest(MajorV2, OpManifest, "bad", 1, []byte(`{}`))
			return err
		}},
		{arm: "ctor|failInvalid|request deadline is outside uint53[1..3600000]", drive: func(t *testing.T) error {
			// The ceiling half of C4: the builder must refuse a
			// deadline the decoder also refuses, and the row
			// proves the builder fired by message. The floor is
			// pinned alongside so the arm's full range resolves
			// here, not only past the ceiling.
			if _, err := EncodeRequest(MajorV2, OpManifest, fixtureRequestID, 0, []byte(`{}`)); err == nil {
				t.Fatal("EncodeRequest admitted deadline 0")
			}
			_, err := EncodeRequest(MajorV2, OpManifest, fixtureRequestID, 3600001, []byte(`{}`))
			return err
		}},
		{arm: `ctor|failViolation|expr|EncodeRequest|"request body "+fault.detail`, drive: func(t *testing.T) error {
			_, err := EncodeRequest(MajorV2, OpManifest, fixtureRequestID, 1, []byte(`[]`))
			return err
		}},
		// Shared response-identity arms, driven through both
		// branches so the context prefix resolves in each.
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" frame is empty"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			if _, err := CheckSuccessEnvelope([]byte{}, request); err == nil {
				t.Fatal("success admitted an empty frame")
			}
			_, err := CheckFailureEnvelope([]byte{}, request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" frame exceeds the 8 MiB bound"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			oversize := append(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, `{}`), make([]byte, frameBoundBytes)...)
			if _, err := CheckSuccessEnvelope(oversize, request); err == nil {
				t.Fatal("success admitted an oversize frame")
			}
			_, err := CheckFailureEnvelope(oversize, request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" envelope "+fault.detail`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			if _, err := CheckSuccessEnvelope([]byte(`{"schema":`), request); err == nil {
				t.Fatal("success admitted a truncated frame")
			}
			_, err := CheckFailureEnvelope([]byte(`{"schema":`), request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" envelope carries unknown member"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			good := string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()))
			if _, err := CheckSuccessEnvelope([]byte(replaceOnce(t, good, `"ok":true`, `"ok":true,"extra":1`, 1)), request); err == nil {
				t.Fatal("success admitted an unknown member")
			}
			downgrade := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
			_, err := CheckFailureEnvelope([]byte(replaceOnce(t, downgrade, `"ok":false`, `"ok":false,"extra":1`, 1)), request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" envelope misses a required member"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			good := string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()))
			if _, err := CheckSuccessEnvelope([]byte(replaceOnce(t, good, `,"ok":true`, ``, 1)), request); err == nil {
				t.Fatal("success admitted a missing member")
			}
			downgrade := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
			_, err := CheckFailureEnvelope([]byte(replaceOnce(t, downgrade, `,"ok":false`, ``, 1)), request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" schema is not the directory node response"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			good := string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()))
			if _, err := CheckSuccessEnvelope([]byte(replaceOnce(t, good, "session-directory-node-response", "session-directory-node-request", 1)), request); err == nil {
				t.Fatal("success admitted a foreign schema")
			}
			downgrade := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
			_, err := CheckFailureEnvelope([]byte(replaceOnce(t, downgrade, "session-directory-node-response", "session-directory-node-request", 1)), request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" schema version is not the bound 1.0.0"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			good := string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()))
			if _, err := CheckSuccessEnvelope([]byte(replaceOnce(t, good, `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1)), request); err == nil {
				t.Fatal("success admitted a relabeled schema version")
			}
			downgrade := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
			_, err := CheckFailureEnvelope([]byte(replaceOnce(t, downgrade, `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1)), request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" protocol does not echo the request"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			good := string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()))
			if _, err := CheckSuccessEnvelope([]byte(replaceOnce(t, good, `"protocol":"urn:ax:protocol:session-directory-node"`, `"protocol":"urn:ax:protocol:provider"`, 1)), request); err == nil {
				t.Fatal("success admitted a foreign protocol")
			}
			downgrade := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
			_, err := CheckFailureEnvelope([]byte(replaceOnce(t, downgrade, `"protocol":"urn:ax:protocol:session-directory-node"`, `"protocol":"urn:ax:protocol:provider"`, 1)), request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" protocol version does not echo the request"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			good := fixtureSuccessFrame("1.0.0", "manifest", fixtureRequestID, fixtureManifestJSON())
			if _, err := CheckSuccessEnvelope(good, request); err == nil {
				t.Fatal("success admitted a mis-echoed protocol version")
			}
			_, err := CheckFailureEnvelope(fixtureFailureFrame("1.0.0", "manifest", fixtureRequestID, fixtureErrorObject("incompatible_protocol", 6, false)), request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" request identifier does not echo the request"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			other := fixtureRequestID[:len(fixtureRequestID)-1] + "0"
			if _, err := CheckSuccessEnvelope(fixtureSuccessFrame("2.0.0", "manifest", other, fixtureManifestJSON()), request); err == nil {
				t.Fatal("success admitted a mis-echoed request")
			}
			_, err := CheckFailureEnvelope(fixtureFailureFrame("2.0.0", "manifest", other, fixtureErrorObject("incompatible_protocol", 6, false)), request)
			return err
		}},
		{arm: `ctor|failViolation|expr|checkResponseIdentity|context+" operation does not echo the request"`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			if _, err := CheckSuccessEnvelope(fixtureSuccessFrame("2.0.0", "probe", fixtureRequestID, fixtureProbeResponseJSON()), request); err == nil {
				t.Fatal("success admitted a mis-echoed operation")
			}
			_, err := CheckFailureEnvelope(fixtureFailureFrame("2.0.0", "probe", fixtureRequestID, fixtureErrorObject("incompatible_protocol", 6, false)), request)
			return err
		}},
		// Success envelope branches.
		{arm: "ctor|failViolation|success envelope does not carry ok=true", drive: func(t *testing.T) error {
			request := reachRequest(t)
			good := string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()))
			_, err := CheckSuccessEnvelope([]byte(replaceOnce(t, good, `"ok":true`, `"ok":false`, 1)), request)
			return err
		}},
		{arm: "ctor|failViolation|success envelope carries both body and error", drive: func(t *testing.T) error {
			request := reachRequest(t)
			good := string(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()))
			both := replaceOnce(t, good, `"ok":true,`, `"ok":true,"error":`+fixtureErrorObject("incompatible_protocol", 6, false)+`,`, 1)
			_, err := CheckSuccessEnvelope([]byte(both), request)
			return err
		}},
		{arm: "ctor|failViolation|success envelope carries neither body nor error", drive: func(t *testing.T) error {
			// A literal branch-exact frame: renaming body fires
			// the unknown-member arm first, so the neither arm
			// is reachable only with neither member present and
			// no unknown one.
			request := reachRequest(t)
			frame := []byte(`{"schema":"urn:ax:schema:session-directory-node-response","schema_version":"1.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":"2.0.0","request_id":"` + fixtureRequestID + `","operation":"manifest","ok":true}`)
			_, err := CheckSuccessEnvelope(frame, request)
			return err
		}},
		{arm: `ctor|failViolation|expr|CheckSuccessEnvelope|"success body "+fault.detail`, drive: func(t *testing.T) error {
			request := reachRequest(t)
			_, err := CheckSuccessEnvelope(fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, `[]`), request)
			return err
		}},
		// Failure envelope branches.
		{arm: "ctor|failViolation|failure envelope does not carry ok=false", drive: func(t *testing.T) error {
			request := reachRequest(t)
			downgrade := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
			_, err := CheckFailureEnvelope([]byte(replaceOnce(t, downgrade, `"ok":false`, `"ok":true`, 1)), request)
			return err
		}},
		{arm: "ctor|failViolation|failure envelope carries both body and error", drive: func(t *testing.T) error {
			request := reachRequest(t)
			downgrade := string(fixtureDowngradeFrame("2.0.0", fixtureRequestID))
			_, err := CheckFailureEnvelope([]byte(replaceOnce(t, downgrade, `"ok":false,`, `"ok":false,"body":{},`, 1)), request)
			return err
		}},
		{arm: "ctor|failViolation|failure envelope carries neither body nor error", drive: func(t *testing.T) error {
			request := reachRequest(t)
			frame := []byte(`{"schema":"urn:ax:schema:session-directory-node-response","schema_version":"1.0.0","protocol":"urn:ax:protocol:session-directory-node","protocol_version":"2.0.0","request_id":"` + fixtureRequestID + `","operation":"manifest","ok":false}`)
			_, err := CheckFailureEnvelope(frame, request)
			return err
		}},
		{arm: "ctor|failIntegrity|failure error is not a Structured Error 1.2.0", drive: func(t *testing.T) error {
			request := reachRequest(t)
			_, err := CheckFailureEnvelope(fixtureFailureFrame("2.0.0", "manifest", fixtureRequestID, `{"schema":"urn:ax:schema:error"}`), request)
			return err
		}},
		// Manifest entry.
		{arm: `ctor|failViolation|expr|DecodeManifest|"node manifest "+fault.detail`, drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(good[:len(good)-1] + `,"node_id":"x"}`))
			return err
		}},
		{arm: "ctor|failViolation|node manifest carries unknown member", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			body := replaceOnce(t, good, `"max_enrichment_bytes":4194304,"extensions":{}},"extensions":{}}`, `"max_enrichment_bytes":4194304,"extensions":{}},"extensions":{},"extra":1}`, 1)
			_, err := DecodeManifest([]byte(body))
			return err
		}},
		{arm: "ctor|failViolation|node manifest misses a required member", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"node_version":"2.4.1",`, ``, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest schema is not the directory node manifest", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, "session-directory-node-manifest", "session-directory-node-response", 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest version is not 1.0.0", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest node identifier is not a string[1..128]", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"node_id":"test-directory-node"`, `"node_id":""`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest node version is not SemVer", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"node_version":"2.4.1"`, `"node_version":"2.4"`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest host identifier is not a UUIDv7", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, fixtureHostID, "not-a-uuid", 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest executable binding is not a digest", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, fixtureExecutableDigest, "sha256:zzz", 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest provider binding is not a digest", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, fixtureProviderDigest, "sha256:zzz", 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest adapter binding is not a digest", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, fixtureAdapterDigest, "sha256:zzz", 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest supported versions are not sorted unique SemVer[1..16]", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":["2.0.0","1.0.0"]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest operations are not the complete sorted eleven-name registry", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"operations":["continuation-inspect","doctor"`, `"operations":["doctor","continuation-inspect"`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest schemas are not sorted unique ContractAssertion[15..64]", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			dropped := replaceOnce(t, good, `{"contract_id":"urn:ax:schema:contract-00","exact_version":"1.0.0","extensions":{}},`, ``, 1)
			_, err := DecodeManifest([]byte(dropped))
			return err
		}},
		{arm: "ctor|failViolation|node manifest tuple registry binding is not a digest", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"environment_tuple_registry_id":`+quote(fixtureTupleRegistry), `"environment_tuple_registry_id":null`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest capabilities are not the exact eight-name result map", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"directory_discovery":{"status":"available"`, `"directory_discovery":{"status":"available","unexpected":1`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest redaction policies are not sorted unique digest[1..64]", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"redaction_policy_ids":[`+quote(fixtureRedactionDigest)+`]`, `"redaction_policy_ids":[]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest enrichment profiles are not sorted unique digest[0..256]", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, fixtureEnrichmentDigest, "sha256:zzz", 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest limits are not the closed DirectoryNodeLimits", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"max_frame_bytes":8388608`, `"max_frame_bytes":0`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest extensions are not reverse-DNS keyed", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(replaceOnce(t, good, `"extensions":{}},"extensions":{}}`, `"extensions":{}},"extensions":{"x":1}}`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|node manifest is not canonical JSON", drive: func(t *testing.T) error {
			good := fixtureManifestJSON()
			_, err := DecodeManifest([]byte(deepExtensionsManifest(t, good)))
			return err
		}},
		// Manifest binding entries.
		{arm: "ctor|failInvalid|observed executable digest is not a digest", drive: func(t *testing.T) error {
			manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			observed := fixtureObservedFacade()
			observed.ExecutableSHA256 = "sha256:zzz"
			return CheckManifestBindings(manifest, observed)
		}},
		{arm: "ctor|failInvalid|observed provider manifest digest is not a digest", drive: func(t *testing.T) error {
			manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			observed := fixtureObservedFacade()
			observed.ProviderManifestDigest = "sha256:zzz"
			return CheckManifestBindings(manifest, observed)
		}},
		{arm: "ctor|failInvalid|observed adapter manifest digest is not a digest", drive: func(t *testing.T) error {
			manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			observed := fixtureObservedFacade()
			observed.SessionAdapterDigest = "sha256:zzz"
			return CheckManifestBindings(manifest, observed)
		}},
		{arm: "ctor|failIntegrity|node manifest executable binding contradicts the observed executable", drive: func(t *testing.T) error {
			manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			observed := fixtureObservedFacade()
			observed.ExecutableSHA256 = fixtureDigest("other-executable")
			return CheckManifestBindings(manifest, observed)
		}},
		{arm: "ctor|failIntegrity|node manifest provider binding contradicts the observed manifest", drive: func(t *testing.T) error {
			manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			observed := fixtureObservedFacade()
			observed.ProviderManifestDigest = fixtureDigest("other-provider")
			return CheckManifestBindings(manifest, observed)
		}},
		{arm: "ctor|failIntegrity|node manifest adapter binding contradicts the observed manifest", drive: func(t *testing.T) error {
			manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
			if err != nil {
				t.Fatalf("DecodeManifest: %v", err)
			}
			observed := fixtureObservedFacade()
			observed.SessionAdapterDigest = fixtureDigest("other-adapter")
			return CheckManifestBindings(manifest, observed)
		}},
		// Probe request entry.
		{arm: "ctor|failInvalid|probe request major is outside the locally supported registry", drive: func(t *testing.T) error {
			_, err := CheckProbeRequest(3, []byte(fixtureProbeRequestV2()))
			return err
		}},
		{arm: `ctor|failViolation|expr|CheckProbeRequest|"probe request "+fault.detail`, drive: func(t *testing.T) error {
			_, err := CheckProbeRequest(MajorV2, []byte(`[]`))
			return err
		}},
		{arm: "ctor|failViolation|probe request carries unknown member", drive: func(t *testing.T) error {
			good := fixtureProbeRequestV2()
			_, err := CheckProbeRequest(MajorV2, []byte(replaceOnce(t, good, `"extensions":{}}`, `"extensions":{},"extra":1}`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe request misses a required member", drive: func(t *testing.T) error {
			good := fixtureProbeRequestV2()
			_, err := CheckProbeRequest(MajorV2, []byte(replaceOnce(t, good, `"architecture":"amd64",`, ``, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe request platform is outside the negotiated major vocabulary", drive: func(t *testing.T) error {
			_, err := CheckProbeRequest(MajorV2, []byte(`{"platform":"darwin","architecture":"amd64","requested_environment_ids":[],"requested_capabilities":[],"extensions":{}}`))
			return err
		}},
		{arm: "ctor|failViolation|probe request architecture is outside amd64|arm64", drive: func(t *testing.T) error {
			good := fixtureProbeRequestV2()
			_, err := CheckProbeRequest(MajorV2, []byte(replaceOnce(t, good, `"amd64"`, `"riscv"`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe request environment identifiers are not sorted unique strings[0..64]", drive: func(t *testing.T) error {
			good := fixtureProbeRequestV2()
			_, err := CheckProbeRequest(MajorV2, []byte(replaceOnce(t, good, `"requested_environment_ids":[]`, `"requested_environment_ids":["b.env","a.env"]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe request environment identifier is not an environment-id", drive: func(t *testing.T) error {
			good := fixtureProbeRequestV2()
			_, err := CheckProbeRequest(MajorV2, []byte(replaceOnce(t, good, `"requested_environment_ids":[]`, `"requested_environment_ids":["BAD"]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe request capabilities are not sorted unique strings[0..8]", drive: func(t *testing.T) error {
			good := fixtureProbeRequestV2()
			_, err := CheckProbeRequest(MajorV2, []byte(replaceOnce(t, good, `"requested_capabilities":["directory_discovery"]`, `"requested_capabilities":["directory_discovery","directory_head_digest","directory_incremental_scan","directory_tail_preview","existing_session_adoption","native_resume","native_runtime_observation","native_title_read","extra"]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe request capability is outside the directory registry", drive: func(t *testing.T) error {
			good := fixtureProbeRequestV2()
			_, err := CheckProbeRequest(MajorV2, []byte(replaceOnce(t, good, `"requested_capabilities":["directory_discovery"]`, `"requested_capabilities":["directory_time_travel"]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe request extensions are not reverse-DNS keyed", drive: func(t *testing.T) error {
			good := fixtureProbeRequestV2()
			_, err := CheckProbeRequest(MajorV2, []byte(replaceOnce(t, good, `"extensions":{}}`, `"extensions":{"not-reversedns":1}}`, 1)))
			return err
		}},
		// Probe response entry.
		{arm: `ctor|failViolation|expr|CheckProbeResponse|"probe response "+fault.detail`, drive: func(t *testing.T) error {
			_, err := CheckProbeResponse([]byte(`[]`))
			return err
		}},
		{arm: "ctor|failViolation|probe response carries unknown member", drive: func(t *testing.T) error {
			good := fixtureProbeResponseJSON()
			_, err := CheckProbeResponse([]byte(replaceOnce(t, good, `"extensions":{}}],"extensions":{}}`, `"extensions":{}}],"extensions":{},"extra":1}`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe response misses a required member", drive: func(t *testing.T) error {
			good := fixtureProbeResponseJSON()
			_, err := CheckProbeResponse([]byte(replaceOnce(t, good, `"policy_digest":`+quote(fixturePolicyDigest)+`,`, ``, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe response host identifier is not a UUIDv7", drive: func(t *testing.T) error {
			good := fixtureProbeResponseJSON()
			_, err := CheckProbeResponse([]byte(replaceOnce(t, good, fixtureHostID, "not-a-uuid", 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe response node build is not the closed DirectoryNodeBuild", drive: func(t *testing.T) error {
			good := fixtureProbeResponseJSON()
			_, err := CheckProbeResponse([]byte(replaceOnce(t, good, `"session_adapter_manifest_digest":`, `"session_adapter_manifest_digestX":`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe response policy digest is not a digest", drive: func(t *testing.T) error {
			good := fixtureProbeResponseJSON()
			_, err := CheckProbeResponse([]byte(replaceOnce(t, good, fixturePolicyDigest, "sha256:zzz", 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe response environments are not EnvironmentObservation[0..256] by shape", drive: func(t *testing.T) error {
			good := fixtureProbeResponseJSON()
			_, err := CheckProbeResponse([]byte(replaceOnce(t, good, `[{"environment_id":"test.env"}]`, `[1]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe response findings are not AdapterFinding[0..4096]", drive: func(t *testing.T) error {
			good := fixtureProbeResponseJSON()
			_, err := CheckProbeResponse([]byte(replaceOnce(t, good, `"severity":"info"`, `"severity":"fatal"`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|probe response extensions are not reverse-DNS keyed", drive: func(t *testing.T) error {
			good := fixtureProbeResponseJSON()
			_, err := CheckProbeResponse([]byte(replaceOnce(t, good, `"extensions":{}}],"extensions":{}}`, `"extensions":{}}],"extensions":{"x":1}}`, 1)))
			return err
		}},
		// Node build equality entry.
		{arm: "ctor|failIntegrity|probe node build node identifier differs from the manifest", drive: func(t *testing.T) error {
			return reachNodeBuildDrift(t, func(build *NodeBuild) { build.NodeID = "other-node" })
		}},
		{arm: "ctor|failIntegrity|probe node build node version differs from the manifest", drive: func(t *testing.T) error {
			return reachNodeBuildDrift(t, func(build *NodeBuild) { build.NodeVersion = "9.9.9" })
		}},
		{arm: "ctor|failIntegrity|probe node build executable binding differs from the manifest", drive: func(t *testing.T) error {
			return reachNodeBuildDrift(t, func(build *NodeBuild) { build.ExecutableSHA256 = fixtureDigest("other") })
		}},
		{arm: "ctor|failIntegrity|probe node build provider binding differs from the manifest", drive: func(t *testing.T) error {
			return reachNodeBuildDrift(t, func(build *NodeBuild) { build.ProviderManifestDigest = fixtureDigest("other") })
		}},
		{arm: "ctor|failIntegrity|probe node build adapter binding differs from the manifest", drive: func(t *testing.T) error {
			return reachNodeBuildDrift(t, func(build *NodeBuild) { build.AdapterManifestDigest = fixtureDigest("other") })
		}},
	}
}

// reachNodeBuildDrift checks one drifted node build against the
// fixture manifest through the production equality entry.
func reachNodeBuildDrift(t *testing.T, apply func(*NodeBuild)) error {
	t.Helper()
	response, err := CheckProbeResponse([]byte(fixtureProbeResponseJSON()))
	if err != nil {
		t.Fatalf("CheckProbeResponse: %v", err)
	}
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	build := response.Build
	apply(&build)
	return CheckNodeBuildEqualsManifest(build, manifest)
}

// armReachabilityScanRows pairs the scan and journal arms with
// their firing vectors.
func armReachabilityScanRows() []armReachRow {
	return []armReachRow{
		// Scan request entry.
		{arm: `ctor|failViolation|expr|CheckScanRequest|"scan request "+fault.detail`, drive: func(t *testing.T) error {
			_, err := CheckScanRequest([]byte(`[]`))
			return err
		}},
		{arm: "ctor|failViolation|scan request carries unknown member", drive: func(t *testing.T) error {
			good := fixtureScanRequestJSON()
			_, err := CheckScanRequest([]byte(replaceOnce(t, good, `"extensions":{}}`, `"extensions":{},"extra":1}`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan request misses a required member", drive: func(t *testing.T) error {
			good := fixtureScanRequestJSON()
			_, err := CheckScanRequest([]byte(replaceOnce(t, good, `"cursor":null,`, ``, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan request operation identifier is not a UUIDv7", drive: func(t *testing.T) error {
			good := fixtureScanRequestJSON()
			_, err := CheckScanRequest([]byte(replaceOnce(t, good, fixtureOperationID, "not-a-uuid", 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan request installations are not sorted unique digest[1..256]", drive: func(t *testing.T) error {
			good := fixtureScanRequestJSON()
			sorted := sortedInstallPair()
			_, err := CheckScanRequest([]byte(replaceOnce(t, good, `"installation_ids":[`+quote(sorted[0])+`,`+quote(sorted[1])+`]`, `"installation_ids":[]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan request prior batch is not a digest|null", drive: func(t *testing.T) error {
			good := fixtureScanRequestJSON()
			_, err := CheckScanRequest([]byte(replaceOnce(t, good, `"prior_batch_id":null`, `"prior_batch_id":0`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan request cursor is not a string[1..4096]|null", drive: func(t *testing.T) error {
			good := fixtureScanRequestJSON()
			_, err := CheckScanRequest([]byte(replaceOnce(t, good, `"cursor":null`, `"cursor":""`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan request max_instances is not uint53[1..65536]", drive: func(t *testing.T) error {
			good := fixtureScanRequestJSON()
			_, err := CheckScanRequest([]byte(replaceOnce(t, good, `"max_instances":100`, `"max_instances":0`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan request extensions are not reverse-DNS keyed", drive: func(t *testing.T) error {
			good := fixtureScanRequestJSON()
			_, err := CheckScanRequest([]byte(replaceOnce(t, good, `"extensions":{}}`, `"extensions":{"x":1}}`, 1)))
			return err
		}},
		// Scan response entry.
		{arm: `ctor|failViolation|expr|CheckScanResponse|"scan response "+fault.detail`, drive: func(t *testing.T) error {
			_, err := CheckScanResponse([]byte(`[]`))
			return err
		}},
		{arm: "ctor|failViolation|scan response carries unknown member", drive: func(t *testing.T) error {
			good := fixtureScanResponseJSON()
			_, err := CheckScanResponse([]byte(replaceOnce(t, good, `"extensions":{}}`, `"extensions":{},"extra":1}`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan response misses a required member", drive: func(t *testing.T) error {
			good := fixtureScanResponseJSON()
			_, err := CheckScanResponse([]byte(replaceOnce(t, good, `"next_cursor":null,`, ``, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan response batch is not an object", drive: func(t *testing.T) error {
			good := fixtureScanResponseJSON()
			_, err := CheckScanResponse([]byte(replaceOnce(t, good, `"batch":{"batch_id":`+quote(fixtureBatchDigest)+`}`, `"batch":[]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan response environment observations are not sorted unique digest[1..256]", drive: func(t *testing.T) error {
			good := fixtureScanResponseJSON()
			_, err := CheckScanResponse([]byte(replaceOnce(t, good, `"environment_observation_ids":[`+quote(fixtureEvidenceDigest)+`]`, `"environment_observation_ids":[]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan response native observations are not sorted unique digest[0..65536]", drive: func(t *testing.T) error {
			good := fixtureScanResponseJSON()
			// Two natives in decreasing order: sortedness fails
			// while every element stays a valid digest, so the
			// ordering arm fires rather than an element arm.
			nativePair := sortedNativePair()
			_, err := CheckScanResponse([]byte(replaceOnce(t, good, `"native_observation_ids":[`+quote(fixtureNativeDigestA)+`]`, `"native_observation_ids":[`+quote(nativePair[1])+`,`+quote(nativePair[0])+`]`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan response next cursor is not a string[1..4096]|null", drive: func(t *testing.T) error {
			good := fixtureScanResponseJSON()
			_, err := CheckScanResponse([]byte(replaceOnce(t, good, `"next_cursor":null`, `"next_cursor":0`, 1)))
			return err
		}},
		{arm: "ctor|failViolation|scan response extensions are not reverse-DNS keyed", drive: func(t *testing.T) error {
			good := fixtureScanResponseJSON()
			_, err := CheckScanResponse([]byte(replaceOnce(t, good, `"extensions":{}}`, `"extensions":{"x":1}}`, 1)))
			return err
		}},
		// Idempotency entries.
		{arm: "ctor|failViolation|idempotency body is not canonical JSON", drive: func(t *testing.T) error {
			journal := NewJournal()
			_, _, err := journal.CheckAndRecord(OpScan, fixtureOperationID, []byte(`{`), []byte(`{}`))
			return err
		}},
		{arm: "ctor|failInvalid|idempotency operation identifier is not a UUIDv7", drive: func(t *testing.T) error {
			journal := NewJournal()
			_, _, err := journal.CheckAndRecord(OpScan, "bad", journalBody(fixtureOperationID), []byte(`{}`))
			return err
		}},
		{arm: "ctor|failUnknownOperation|idempotency key names an operation outside the closed registry", drive: func(t *testing.T) error {
			journal := NewJournal()
			_, _, err := journal.CheckAndRecord("query", fixtureOperationID, journalBody(fixtureOperationID), []byte(`{}`))
			return err
		}},
		{arm: "ctor|failIdempotency|idempotent body changed under a recorded operation identifier", drive: func(t *testing.T) error {
			journal := NewJournal()
			body := journalBody(fixtureOperationID)
			if _, _, err := journal.CheckAndRecord(OpScan, fixtureOperationID, body, []byte(`{"batch":"first"}`)); err != nil {
				t.Fatalf("CheckAndRecord(first): %v", err)
			}
			changed := replaceOnce(t, string(body), `"max_instances":100`, `"max_instances":101`, 1)
			_, _, err := journal.CheckAndRecord(OpScan, fixtureOperationID, []byte(changed), []byte(`{"batch":"evil"}`))
			return err
		}},
		{arm: `ctor|failIntegrity|expr|Import|"idempotency journal "+fault.detail`, drive: func(t *testing.T) error {
			return NewJournal().Import([]byte(`not json`))
		}},
		{arm: "ctor|failIntegrity|idempotency journal schema is not the journal export", drive: func(t *testing.T) error {
			return NewJournal().Import([]byte(`{"schema":"urn:ax:other","version":1,"records":[]}`))
		}},
		{arm: "ctor|failIntegrity|idempotency journal version is not 1", drive: func(t *testing.T) error {
			return NewJournal().Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":2,"records":[]}`))
		}},
		{arm: "ctor|failIntegrity|idempotency journal records are not an array", drive: func(t *testing.T) error {
			return NewJournal().Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1}`))
		}},
		{arm: "ctor|failIntegrity|idempotency journal record is not a keyed digest binding", drive: func(t *testing.T) error {
			return NewJournal().Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[{"key":1,"body_digest":"eA==","result":"e30="}]}`))
		}},
		{arm: "ctor|failIntegrity|idempotency journal records are not sorted unique", drive: func(t *testing.T) error {
			high := `{"key":"scan 0198f4c8-8e50-7f66-8f70-1234567890ae","body_digest":` + quote(fixtureBatchDigest) + `,"result":"e30="}`
			low := `{"key":"scan 0198f4c8-8e50-7f66-8f70-1234567890ac","body_digest":` + quote(fixtureBatchDigest) + `,"result":"e30="}`
			return NewJournal().Import([]byte(`{"schema":"urn:ax:internal:dirnode-journal","version":1,"records":[` + high + `,` + low + `]}`))
		}},
	}
}

// sortedInstallPair returns the fixture installation digests in
// sorted order for surgery needles.
func sortedInstallPair() []string {
	pair := []string{fixtureInstallDigestA, fixtureInstallDigestB}
	sort.Strings(pair)
	return pair
}

// sortedNativePair returns two native digests in sorted order for
// the decreasing-order vector.
func sortedNativePair() []string {
	pair := []string{fixtureNativeDigestA, fixtureEvidenceDigest}
	sort.Strings(pair)
	return pair
}

// armReachabilityQueryRows pairs the query arms with their firing
// vectors. The single operation arm carries every operation-level
// defect by construction; the obligation-level resolution lives in
// the obligation census, and this row pins only that the arm
// itself fires through DecodeQuery.
func armReachabilityQueryRows() []armReachRow {
	return []armReachRow{
		{arm: `ctor|failQuery|expr|DecodeQuery|"directory query "+fault.detail`, drive: func(t *testing.T) error {
			_, err := DecodeQuery([]byte(`[]`))
			return err
		}},
		{arm: "ctor|failQuery|directory query carries unknown member", drive: func(t *testing.T) error {
			good := fixtureQueryJSON()
			_, err := DecodeQuery([]byte(replaceOnce(t, good, `],"caller":`, `],"extra":1,"caller":`, 1)))
			return err
		}},
		{arm: "ctor|failQuery|directory query misses a required member", drive: func(t *testing.T) error {
			good := fixtureQueryJSON()
			_, err := DecodeQuery([]byte(replaceOnce(t, good, `,"caller":`+fixtureCallerJSON(), ``, 1)))
			return err
		}},
		{arm: "ctor|failQuery|directory query schema is not the session directory query", drive: func(t *testing.T) error {
			good := fixtureQueryJSON()
			_, err := DecodeQuery([]byte(replaceOnce(t, good, "session-directory-query", "session-directory-node-request", 1)))
			return err
		}},
		{arm: "ctor|failQuery|directory query version is not 1.0.0", drive: func(t *testing.T) error {
			good := fixtureQueryJSON()
			_, err := DecodeQuery([]byte(replaceOnce(t, good, `"schema_version":"1.0.0"`, `"schema_version":"2.0.0"`, 1)))
			return err
		}},
		{arm: "ctor|failQuery|directory query identifier is not a UUIDv7", drive: func(t *testing.T) error {
			good := fixtureQueryJSON()
			_, err := DecodeQuery([]byte(replaceOnce(t, good, fixtureQueryID, "not-a-uuid", 1)))
			return err
		}},
		{arm: "ctor|failQuery|directory query operations are not QueryOperation[1..64]", drive: func(t *testing.T) error {
			body := `{"schema":"urn:ax:schema:session-directory-query","schema_version":"1.0.0","query_id":"` + fixtureQueryID + `","operations":[],"caller":` + fixtureCallerJSON() + `,"extensions":{}}`
			_, err := DecodeQuery([]byte(body))
			return err
		}},
		{arm: "ctor|failQuery|directory query caller is not a closed CallerContext", drive: func(t *testing.T) error {
			good := fixtureQueryJSON()
			_, err := DecodeQuery([]byte(replaceOnce(t, good, `"directory.read"`, `"directory.root"`, 1)))
			return err
		}},
		{arm: "ctor|failQuery|directory query extensions are not reverse-DNS keyed", drive: func(t *testing.T) error {
			good := fixtureQueryJSON()
			needle := `"disclosure_policy_digest":` + quote(fixturePolicyDigest) + `,"extensions":{}},"extensions":{}}`
			replacement := `"disclosure_policy_digest":` + quote(fixturePolicyDigest) + `,"extensions":{}},"extensions":{"x":1}}`
			_, err := DecodeQuery([]byte(replaceOnce(t, good, needle, replacement, 1)))
			return err
		}},
		{arm: "ctor|failQuery|directory query operation is not a closed QueryOperation", drive: func(t *testing.T) error {
			good := fixtureQueryJSON()
			_, err := DecodeQuery([]byte(replaceOnce(t, good, `"name":"sessions"`, `"name":"search"`, 1)))
			return err
		}},
		{arm: "ctor|failQuery|directory cursor is not a string[1..1024]", drive: func(t *testing.T) error {
			return CheckCursorReuse("", false)
		}},
		{arm: "ctor|failQuery|directory cursor is bound to a changed query", drive: func(t *testing.T) error {
			return CheckCursorReuse("abc", true)
		}},
	}
}

// armReachabilityBootstrapRows pairs the bootstrap and process
// guard arms with their firing vectors.
func armReachabilityBootstrapRows() []armReachRow {
	return []armReachRow{
		{arm: "ctor|failInvalid|bootstrap attempt does not carry operation manifest", drive: func(t *testing.T) error {
			request, err := DecodeRequestFrame(fixtureRequestFrame("2.0.0", "probe", fixtureRequestID, 1, fixtureProbeRequestV2()))
			if err != nil {
				t.Fatalf("DecodeRequestFrame: %v", err)
			}
			decision := DecideBootstrapStep(request, fixtureSuccessFrame("2.0.0", "probe", fixtureRequestID, fixtureProbeResponseJSON()), 0)
			if decision.Terminal == nil {
				t.Fatal("probe attempt decided without a terminal refusal")
			}
			return decision.Terminal
		}},
		{arm: "ctor|failTransport|bootstrap attempt produced no response frame", drive: func(t *testing.T) error {
			decision := DecideBootstrapStep(bootstrapRequest(t, "2.0.0"), nil, 6)
			if decision.Terminal == nil {
				t.Fatal("empty frame decided without a terminal refusal")
			}
			return decision.Terminal
		}},
		{arm: "ctor|failViolation|bootstrap manifest does not contain the attempted version", drive: func(t *testing.T) error {
			request := bootstrapRequest(t, "2.0.0")
			v1only := replaceOnce(t, fixtureManifestJSON(), `"supported_protocol_versions":["1.0.0","2.0.0"]`, `"supported_protocol_versions":["1.0.0"]`, 1)
			decision := DecideBootstrapStep(request, fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, v1only), 0)
			if decision.Terminal == nil {
				t.Fatal("version-omitting manifest decided without a terminal refusal")
			}
			return decision.Terminal
		}},
		{arm: "ctor|failViolation|bootstrap success carries a nonmatching exit status", drive: func(t *testing.T) error {
			request := bootstrapRequest(t, "2.0.0")
			decision := DecideBootstrapStep(request, fixtureSuccessFrame("2.0.0", "manifest", fixtureRequestID, fixtureManifestJSON()), 1)
			if decision.Terminal == nil {
				t.Fatal("exit-1 success decided without a terminal refusal")
			}
			return decision.Terminal
		}},
		{arm: "ctor|failViolation|bootstrap failure is not the exact downgrade tuple", drive: func(t *testing.T) error {
			request := bootstrapRequest(t, "2.0.0")
			ordinary := fixtureFailureFrame("2.0.0", "manifest", fixtureRequestID, fixtureErrorObject("capability_unavailable", 6, false))
			decision := DecideBootstrapStep(request, ordinary, 6)
			if decision.Terminal == nil {
				t.Fatal("ordinary failure decided without a terminal refusal")
			}
			return decision.Terminal
		}},
		{arm: "ctor|failViolation|bootstrap downgrade response carries a nonmatching exit status", drive: func(t *testing.T) error {
			request := bootstrapRequest(t, "2.0.0")
			decision := DecideBootstrapStep(request, fixtureDowngradeFrame("2.0.0", fixtureRequestID), 0)
			if decision.Terminal == nil {
				t.Fatal("exit-0 downgrade decided without a terminal refusal")
			}
			return decision.Terminal
		}},
		{arm: "ctor|failDowngrade|no lower locally supported major remains", drive: func(t *testing.T) error {
			request := bootstrapRequest(t, "1.0.0")
			decision := DecideBootstrapStep(request, fixtureDowngradeFrame("1.0.0", fixtureRequestID), 6)
			if decision.Terminal == nil {
				t.Fatal("last-major downgrade decided without a terminal refusal")
			}
			return decision.Terminal
		}},
		{arm: "ctor|failInvalid|bootstrap process token is empty", drive: func(t *testing.T) error {
			return NewProcessGuard().Claim("")
		}},
		{arm: "ctor|failInvalid|bootstrap process was already used for an attempt", drive: func(t *testing.T) error {
			guard := NewProcessGuard()
			if err := guard.Claim("process-a"); err != nil {
				t.Fatalf("Claim(fresh): %v", err)
			}
			return guard.Claim("process-a")
		}},
		// Frame faults, each pinned with the carrier conduit that
		// reports it through the request entry.
		{arm: "frame|not valid UTF-8", also: `ctor|failViolation|expr|DecodeRequestFrame|"request envelope "+fault.detail`, drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte{0xff, 0xfe})
			return err
		}},
		// The lone escape rides inside a quoted string: the
		// string-walk gate refuses it at the surrogate arm. (An
		// unquoted replacement would break JSON syntax and refuse
		// at the syntax arm instead; the old raw scan refused such
		// bytes at the surrogate arm regardless of string context.)
		{arm: "frame|lone surrogate escape", also: `ctor|failViolation|expr|DecodeRequestFrame|"request envelope "+fault.detail`, drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(replaceOnce(t, reachFrame(t), `"manifest"`, `"mani\ud800fest"`, 1)))
			return err
		}},
		{arm: "frame|not a JSON object", also: `ctor|failViolation|expr|DecodeRequestFrame|"request envelope "+fault.detail`, drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(`[]`))
			return err
		}},
		{arm: "frame|duplicate member", also: `ctor|failViolation|expr|DecodeRequestFrame|"request envelope "+fault.detail`, drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(reachFrame(t)[:len(reachFrame(t))-1] + `,"operation":"manifest"}`))
			return err
		}},
		{arm: "frame|trailing data after the object", also: `ctor|failViolation|expr|DecodeRequestFrame|"request envelope "+fault.detail`, drive: func(t *testing.T) error {
			_, err := DecodeRequestFrame([]byte(reachFrame(t) + " {}"))
			return err
		}},
	}
}
