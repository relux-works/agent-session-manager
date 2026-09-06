package sessadapter

// This file is the identity-gate census (review rev4 B7). It exists
// because the zero-value / absent-fact class was closed three
// times as a site list (4, then 5, then 13 sites) while the class
// stayed open: TestEnvelopeEchoEmptyStringRefuses derives its rows
// from three envelopes with a tripwire that forbids growth, so it
// cannot reach DecodeProbe's schema echo — the identical two lines
// as DecodeManifest's pinned echo — nor CheckProbe, Discover,
// CheckTupleAdmission, or VerifyRequestDigest.
//
// Shape, modelled on TestClosedVocabularyTablesAreRegistered:
// scanIdentityGateIDs derives every X != Y and X == Y domain
// comparison from production source (excluding error, nil, EOF,
// delimiter, and length plumbing), and TestIdentityGatesAreCensused
// requires the derived multiset to equal identityRegistrations
// exactly. Both spellings are derived: the zero-fact class is
// spelled both ways in production (Discover refuses `== ""`
// while the binding echoes refuse `!=`), and an ==-spelled gate
// with no row used to pass the whole suite unrostered (review
// rev5/F1). A new identity gate in one of the derived shapes
// below with no row fails the census; a removed gate orphans its
// row. Map-presence defaults (!present) live in
// presenceRegistrations with a structural fail-closed proof plus
// behavioral drivers. Vocabulary-table membership positives
// (`x == allowed` over a rostered table) are derived here and
// driven by TestVocabularyMembershipRefusesEmpty, which refuses
// the empty string at every loop; the tables themselves stay
// pinned by TestClosedVocabularyTablesAreRegistered. An empty
// scan fails closed, and canaries from the review traversal
// guard scanner blindness in both spellings.
//
// Shape space (review rev6 G-B; the brief asked what shape comes
// after the one just closed, so the answer is written here, not
// left as silence). An identity decision in this package can only
// be carried by these AST shapes; the census derives the marked
// ones and bounds the rest:
//
//   1. `==` / `!=` BinaryExpr in a FuncDecl body, including
//      nested func literals — DERIVED (both spellings).
//   2. `==` / `!=` BinaryExpr in a package-level var
//      initializer, including the seven func-valued refusal
//      constructors — DERIVED (review rev6 shape; zero live
//      rows today, proven by the synthetic package-literal
//      test below).
//   3. A `bytes.Equal` call in either scope above — DERIVED
//      (review rev6 shape; one live row: CheckContextEcho,
//      context.go:222, doing exactly the echo-identity job this
//      census exists to roster).
//   4. A tagged switch statement classifying inputs — the
//      STATEMENT is classified by the switch census
//      (TestAllProductionSwitchesAreClassified allows only
//      `switch operation` plus the tagless UTF-16 classifier);
//      each case arm is an obligation of the arm inventory
//      (TestEveryArmWitnessRefusesAtTheProductionEntry drives
//      every arm at its entry, so a retargeted arm fails
//      there).
//   5. A tagless switch statement — DERIVED by the same switch
//      census (only readUTF16Escape).
//   6. A map-presence default (`!present` / `!ok`) — DERIVED
//      into presenceRegistrations with a structural fail-closed
//      proof (TestPresenceDefaultsAreFailClosed).
//   7. An inline comparison chain over closed values
//      (`v == "a" || v == "b"`) — DERIVED by the chain census
//      (TestNoUnregisteredInlineVocabularies forbids it).
//   8. Any other stdlib equality predicate (reflect.DeepEqual,
//      slices.Equal, maps.Equal, strings.EqualFold,
//      bytes.Compare against zero) — STATED BOUND, not derived.
//      None is used in production (only bytes.Equal, at the one
//      row above); introducing any other is outside this
//      derivation by policy.
//
// 7 of 8 shapes derived, 1 stated bound. A new gate in shapes
// 1-3 with no row fails TestIdentityGatesAreCensused; shapes
// 4-7 fail their own census; shape 8 is the named residue.
//
// Every non-exempt row names its driver: the committed test
// refusing the zero/absent fact at the exported entry, on the
// side the mutant would exempt. Struct-taking gates (CheckProbe,
// CheckTupleAdmission, CheckBindingEquality) are driven with
// struct literals; bytes-reachable gates (DecodeProbe schema)
// through raw bodies. Where honest decode makes a zero
// unreachable, the row is an explicit exemption naming the
// upstream guard — checkValidateResult's mode echo (vocabulary
// gate at operations.go:1084) is the real example — never a
// silent omission.
//
// Stated bounds: vacuous agreement of two hand-built empty
// structs (an all-zero Probe against all-zero facts) is outside
// the threat model — honest decode never yields empty provider
// identifiers (providerIDPattern) or digests (checkDigest) — and
// stays admitted, documented per row rather than redefined here.
// Shape 8 above (non-bytes.Equal stdlib predicates) and _test.go
// files, which never ship, are likewise outside the derivation.
// The mechanical exclusions (nil, err/fault, io.EOF, rune
// literals, len comparisons) are deliberate routing, not
// blindness: length-spelled gates are rostered by the bound
// census instead.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// identityRegistration is one production identity gate with its
// proof: the behavioral zero-row driver, or an explicit
// exemption. domain groups rows for sibling tripwires.
type identityRegistration struct {
	id     string
	driver string
	domain string
	exempt bool
	reason string
}

// scanIdentityGateIDs derives every X != Y and X == Y
// comparison from production source, excluding the mechanical
// classes: error and fault checks, nil presence, io.EOF framing,
// rune delimiters and escapes, and length comparisons (rostered
// by the bound census instead). The spelling split is
// deliberate: identityGatesInSyntax derives both operators with
// identical exclusions, and the synthetic
// TestIdentityScanSeesBothSpellings proves it.
func scanIdentityGateIDs(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("identity census: %v", err)
	}
	var ids []string
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		path := filepath.Join(directory, name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("identity census: %v", err)
		}
		syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			t.Fatalf("identity census: %v", err)
		}
		ids = append(ids, identityGatesInSyntax(syntax, name)...)
	}
	if scanned == 0 {
		t.Fatal("identity census scanned zero production files; the scanner is broken, not the package")
	}
	return ids
}

// identityGatesInSyntax derives the gate identities of one parsed
// production file: every != and == domain comparison with the
// mechanical classes excluded, plus every bytes.Equal call. Both
// operators share the exclusion set, so neither spelling can hide
// from the roster. Scopes are FuncDecl bodies (including nested
// func literals) and package-level var initializers under their
// variable name — the same scope rule as the constructor census —
// so a gate inside a package-level func literal derives instead
// of passing silently (review rev6). Ordinals disambiguate
// repeated spellings inside one scope; the ordinal map is keyed
// by the full base including the scope, so existing FuncDecl rows
// keep their identities.
func identityGatesInSyntax(syntax *ast.File, name string) []string {
	var ids []string
	ordinal := map[string]int{}
	claim := func(scope, rendered string) {
		base := name + "|" + scope + "|" + rendered
		ids = append(ids, base+"|"+strconv.Itoa(ordinal[base]))
		ordinal[base]++
	}
	for _, declaration := range syntax.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if declaration.Body == nil {
				continue
			}
			identityGatesInScope(declaration.Body, declaration.Name.Name, claim)
		case *ast.GenDecl:
			if declaration.Tok != token.VAR {
				continue
			}
			for _, specification := range declaration.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, ident := range value.Names {
					for _, expr := range value.Values {
						identityGatesInScope(expr, ident.Name, claim)
					}
				}
			}
		}
	}
	return ids
}

// identityGatesInScope claims every identity gate under one scope
// body: != and == domain comparisons with the mechanical classes
// excluded, and bytes.Equal calls in any position. Any
// bytes.Equal call is an identity decision surface — including a
// non-negated or value-bound use — so all derive and the registry
// decides each one's proof.
func identityGatesInScope(body ast.Node, scope string, claim func(scope, rendered string)) {
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BinaryExpr:
			if node.Op != token.NEQ && node.Op != token.EQL {
				return true
			}
			if isNilIdent(node.X) || isNilIdent(node.Y) {
				return true
			}
			if isErrIdent(node.X) || isErrIdent(node.Y) {
				return true
			}
			if isEOFSelector(node.X) || isEOFSelector(node.Y) {
				return true
			}
			if hasRuneLiteral(node) {
				return true
			}
			if hasLenCall(node) {
				return true
			}
			claim(scope, renderBoundExpr(node))
		case *ast.CallExpr:
			if isBytesEqualCall(node) {
				claim(scope, renderBytesEqual(node))
			}
		}
		return true
	})
}

// isBytesEqualCall reports whether the call is a bytes.Equal
// identity comparison: a selector for Equal on the bytes package
// identifier. Other bytes uses (NewReader, TrimSpace) are not
// identity decisions and never match.
func isBytesEqualCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Equal" {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	return ok && receiver.Name == "bytes"
}

// renderBytesEqual renders one bytes.Equal gate deterministically
// from its arguments.
func renderBytesEqual(call *ast.CallExpr) string {
	rendered := make([]string, 0, len(call.Args))
	for _, argument := range call.Args {
		rendered = append(rendered, renderBoundExpr(argument))
	}
	return "bytes.Equal(" + strings.Join(rendered, ", ") + ")"
}

// isNilIdent reports whether the expression is the nil identifier.
func isNilIdent(node ast.Expr) bool {
	ident, ok := node.(*ast.Ident)
	return ok && ident.Name == "nil"
}

// isErrIdent reports whether the expression is an error plumbing
// identifier rather than a domain fact.
func isErrIdent(node ast.Expr) bool {
	ident, ok := node.(*ast.Ident)
	return ok && (ident.Name == "err" || ident.Name == "fault" || ident.Name == "faultErr")
}

// isEOFSelector reports whether the expression is io.EOF.
func isEOFSelector(node ast.Expr) bool {
	selector, ok := node.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "EOF" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == "io"
}

// hasRuneLiteral reports whether the subtree holds a rune
// literal: wire delimiters and escapes, not domain facts.
func hasRuneLiteral(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(inner ast.Node) bool {
		literal, ok := inner.(*ast.BasicLit)
		if ok && literal.Kind == token.CHAR {
			found = true
			return false
		}
		return true
	})
	return found
}

// identityRegistrations rosters every derived gate. domain
// "envelope-echo" feeds the TestEnvelopeEchoEmptyStringRefuses
// tripwire so that table cannot go stale.
var identityRegistrations = []identityRegistration{
	// context.go.
	{id: `context.go|CheckFreshSink|authority.Mode != "fresh_sink"|0`, exempt: true, reason: "dispatch, not a refusal: mode comes from validObjectMode, so \"\" is refused upstream"},
	{id: `context.go|DecodeSourceSelector|count != 1|0`, driver: "TestDecodeSourceSelectorComplement", reason: "exactly-one selector via none/all combos"},
	// context.go: the success-context echo (review rev6 shape 3).
	// The byte-identity spelling lives outside the ==/!=
	// derivation, so it carries its own row in its own domain:
	// a narrowing that admits a same-length rebuilt context
	// fails TestCheckContextEcho (review GA5).
	{id: `context.go|CheckContextEcho|bytes.Equal(received.raw, sent.raw)|0`, driver: "TestCheckContextEcho", domain: "context-echo", reason: "byte-for-byte success context echo; same-length forgery refuses"},
	{id: `context.go|VerifyRequestDigest|digest.String() != context.RequestDigest|0`, driver: "TestIdentityZeroRows", reason: "zero request digest refuses at VerifyRequestDigest"},
	// discovery.go: the sealed binding. Zero flips on both sides
	// are driven; non-zero flips were already pinned by
	// TestBindingEqualityFlipsEveryFact.
	{id: `discovery.go|CheckBindingEquality|sealed.AdapterManifestDigest != fresh.AdapterManifestDigest|0`, driver: "TestBindingEqualityZeroFactsRefuse", domain: "seal", reason: "seal adapter digest zero refuses"},
	{id: `discovery.go|CheckBindingEquality|sealed.CandidateKind != fresh.CandidateKind|0`, driver: "TestBindingEqualityZeroFactsRefuse", domain: "seal", reason: "seal kind zero refuses"},
	{id: `discovery.go|CheckBindingEquality|sealed.ExecutablePath != fresh.ExecutablePath|0`, driver: "TestBindingEqualityZeroFactsRefuse", domain: "seal", reason: "seal path zero refuses"},
	{id: `discovery.go|CheckBindingEquality|sealed.ExecutableSHA256 != fresh.ExecutableSHA256|0`, driver: "TestBindingEqualityZeroFactsRefuse", domain: "seal", reason: "seal executable digest zero refuses"},
	{id: `discovery.go|CheckBindingEquality|sealed.OwnerIdentity != fresh.OwnerIdentity|0`, driver: "TestBindingEqualityZeroFactsRefuse", domain: "seal", reason: "seal owner zero refuses"},
	{id: `discovery.go|CheckBindingEquality|sealed.ProviderID != fresh.ProviderID|0`, driver: "TestBindingEqualityZeroFactsRefuse", domain: "seal", reason: "seal provider zero refuses"},
	{id: `discovery.go|CheckBindingEquality|sealed.ProviderManifestDigest != fresh.ProviderManifestDigest|0`, driver: "TestBindingEqualityZeroFactsRefuse", domain: "seal", reason: "seal provider digest zero refuses"},
	{id: `discovery.go|CheckBindingEquality|sealed.Role != fresh.Role|0`, driver: "TestBindingEqualityZeroFactsRefuse", domain: "seal", reason: "seal role zero refuses"},
	{id: `discovery.go|CheckCallBinding|context.Environment != admitted|0`, driver: "TestCheckCallBindingZeroFactsRefuse", domain: "binding", reason: "zero admitted tuple; caller side unreachable, tuple always decoded"},
	{id: `discovery.go|CheckCallBinding|context.ExecutableSHA256 != binding.ExecutableSHA256|0`, driver: "TestCheckCallBindingZeroFactsRefuse", domain: "binding", reason: "zero binding digest; caller-side zero driven in call binding caller zeros"},
	{id: `discovery.go|CheckCallBinding|context.ManifestDigest != binding.AdapterManifestDigest|0`, driver: "TestCheckCallBindingZeroFactsRefuse", domain: "binding", reason: "zero binding digest; caller-side zero driven in call binding caller zeros"},
	{id: `discovery.go|CheckCallBinding|context.ProviderID != binding.ProviderID|0`, driver: "TestCheckCallBindingZeroFactsRefuse", domain: "binding", reason: "zero binding provider; caller-side zero driven in call binding caller zeros"},
	{id: `discovery.go|CheckCallBinding|role != binding.Role|0`, driver: "TestCheckCallBindingZeroFactsRefuse", domain: "binding", reason: "zero binding role; caller-side zero driven in call binding caller zeros"},
	{id: `discovery.go|Discover|manifest.ProviderID != candidate.ProviderID|0`, driver: "TestIdentityZeroRows", domain: "discover", reason: "zero provider on either side refuses at Discover"},
	// manifest.go: schema echoes pinned by the null-member rows,
	// registry elements by the mutation suite.
	{id: `manifest.go|DecodeManifest|schema != manifestSchema|0`, driver: "TestDecodeManifestClosedMemberRules", domain: "manifest-echo", reason: "null schema row"},
	{id: `manifest.go|DecodeManifest|version != manifestSchemaVersion|0`, driver: "TestDecodeManifestClosedMemberRules", domain: "manifest-echo", reason: "null schema_version row"},
	{id: `manifest.go|checkCapabilityRegistry|name != capabilityOrder[index]|0`, driver: "TestRegistryElementEmptyRefuses", reason: "empty capability name refuses; order pinned by TestDecodeManifestRegistryMutations"},
	{id: `manifest.go|checkOperationRegistry|name != string(operationOrder[index])|0`, driver: "TestRegistryElementEmptyRefuses", reason: "empty operation name refuses; order pinned by TestDecodeManifestRegistryMutations"},
	// operations.go.
	{id: `operations.go|checkExcludedClasses|previous != ""|0`, exempt: true, reason: "first-iteration sortedness sentinel, not a fact gate; class vocabulary refuses \"\" upstream"},
	{id: `operations.go|checkRequiredDispositions|previous != ""|0`, exempt: true, reason: "first-iteration sortedness sentinel, not a fact gate; disposition vocabulary refuses \"\" upstream"},
	{id: `operations.go|checkSuccessScalars|candidate != digest|0`, exempt: true, reason: "equivalent: both sides come from requireDigestValue, which refuses \"\" upstream, so a zero exemption changes no verdict; the mismatch is pinned by TestCapturePlanDigestEquality"},
	{id: `operations.go|checkSuccessScalars|occurrences != 1|0`, driver: "TestResumePlanIdentityComplement", reason: "identity occurrence complement"},
	{id: `operations.go|checkValidateResult|mode != facts.ValidateMode|0`, driver: "TestValidateResultRules", reason: "facts-side zero pinned by zero_request_mode_fact; body-side \"\" is equivalent (vocabulary gate at :1084 refuses first)"},
	// probe.go: the decoder echoes and the host-fact equalities.
	{id: `probe.go|CheckDoctorHealthy|result.EntryStatus != "accepted"|0`, driver: "TestIdentityZeroRows", domain: "doctor", reason: "empty entry status refuses at CheckDoctorHealthy"},
	{id: `probe.go|CheckProbeRequest|expectedKind != candidate.Kind|0`, driver: "TestZeroValueHostFactsRefuse", reason: "zero expected candidate kind"},
	{id: `probe.go|CheckProbeRequest|expectedProviderID != candidate.ProviderID|0`, driver: "TestIdentityZeroRows", domain: "probe-request", reason: "zero provider on either side refuses at CheckProbeRequest"},
	{id: `probe.go|CheckProbe|probe.AdapterVersion != facts.Manifest.AdapterVersion|0`, driver: "TestIdentityZeroRows", domain: "probe-echo", reason: "zero adapter version on either side refuses"},
	{id: `probe.go|CheckProbe|probe.Environment.AdapterVersion != probe.AdapterVersion|0`, driver: "TestIdentityZeroRows", domain: "probe-echo", reason: "empty tuple adapter version refuses"},
	{id: `probe.go|CheckProbe|probe.Environment.EnvironmentID != facts.Manifest.EnvironmentID|0`, driver: "TestIdentityZeroRows", domain: "probe-echo", reason: "probe-side zero refuses; facts-side zero pinned by zero manifest environment"},
	{id: `probe.go|CheckProbe|probe.ManifestDigest != facts.ManifestDigest|0`, driver: "TestIdentityZeroRows", domain: "probe-echo", reason: "zero manifest digest on either side refuses"},
	{id: `probe.go|CheckProbe|probe.ProviderID != facts.ExpectedProviderID|0`, driver: "TestIdentityZeroRows", domain: "probe-echo", reason: "probe-side zero refuses; facts-side zero pinned by zero expected provider"},
	{id: `probe.go|CheckProbe|probe.ProviderID != facts.Manifest.ProviderID|0`, driver: "TestIdentityZeroRows", domain: "probe-echo", reason: "zero provider on the manifest side refuses with the expected side held equal"},
	{id: `probe.go|CheckTargetWriteGates|value.Status != "available"|0`, driver: "TestIdentityZeroRows", domain: "provider-loop", reason: "empty provider status refuses at CheckTargetWriteGates"},
	{id: `probe.go|DecodeDoctorResult|Direction(direction) != sentDirection|0`, driver: "TestIdentityZeroRows", domain: "doctor", reason: "empty request direction refuses; body-side \"\" is equivalent (validDirection refuses first)"},
	{id: `probe.go|DecodeProbe|schema != probeSchema|0`, driver: "TestIdentityZeroRows", domain: "probe-decode", reason: "empty schema refuses at DecodeProbe"},
	{id: `probe.go|DecodeProbe|version != probeSchemaVersion|0`, driver: "TestIdentityZeroRows", domain: "probe-decode", reason: "empty schema_version refuses at DecodeProbe"},
	{id: `probe.go|DoctorRequiredCapabilities|direction != DirectionTargetWrite|0`, exempt: true, reason: "dispatch, not a refusal: direction comes from validDirection, so \"\" is refused upstream"},
	{id: `probe.go|decodeCapabilityValue|status != "available"|0`, exempt: true, reason: "equivalent: \"\" unreachable via validCapabilityStatus upstream; the coherence rule is pinned by TestCapabilityStatusMatrix"},
	// protocol.go: the three envelopes, pinned by the echo table.
	{id: `protocol.go|CheckFailureEnvelope|operation != string(want.Operation)|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "failure operation echo"},
	{id: `protocol.go|CheckFailureEnvelope|protocol != ProtocolID|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "failure protocol echo"},
	{id: `protocol.go|CheckFailureEnvelope|requestID != want.RequestID|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "failure request id echo"},
	{id: `protocol.go|CheckFailureEnvelope|version != ProtocolVersion|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "failure version echo"},
	{id: `protocol.go|CheckSuccessEnvelope|operation != string(want.Operation)|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "success operation echo"},
	{id: `protocol.go|CheckSuccessEnvelope|protocol != ProtocolID|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "success protocol echo"},
	{id: `protocol.go|CheckSuccessEnvelope|requestID != want.RequestID|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "success request id echo"},
	{id: `protocol.go|CheckSuccessEnvelope|version != ProtocolVersion|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "success version echo"},
	{id: `protocol.go|DecodeRequestFrame|protocol != ProtocolID|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "request protocol echo"},
	{id: `protocol.go|DecodeRequestFrame|version != ProtocolVersion|0`, driver: "TestEnvelopeEchoEmptyStringRefuses", domain: "envelope-echo", reason: "request version echo"},
	// tuple.go: sortedness sentinels are exempt; the admission
	// key, direction, environment, and status arms are driven.
	{id: `tuple.go|CheckTupleAdmission|entry.Key.AdapterManifestDigest != binding.AdapterManifestDigest|0`, driver: "TestTupleAdmissionZeroFactsRefuse", domain: "admission", reason: "zero adapter digest on either side refuses"},
	{id: `tuple.go|CheckTupleAdmission|entry.Key.CandidateKind != binding.CandidateKind|0`, driver: "TestTupleAdmissionZeroFactsRefuse", domain: "admission", reason: "zero kind on either side refuses"},
	{id: `tuple.go|CheckTupleAdmission|entry.Key.Direction != direction|0`, driver: "TestTupleAdmissionZeroFactsRefuse", domain: "admission", reason: "empty entry direction refuses"},
	{id: `tuple.go|CheckTupleAdmission|entry.Key.Environment != environment|0`, driver: "TestTupleAdmissionZeroFactsRefuse", domain: "admission", reason: "zero entry side refuses; a hand-built Tuple{} caller param is outside the decode-pinned flow and stays a stated bound"},
	{id: `tuple.go|CheckTupleAdmission|entry.Key.ExecutableSHA256 != binding.ExecutableSHA256|0`, driver: "TestTupleAdmissionZeroFactsRefuse", domain: "admission", reason: "zero executable digest on either side refuses"},
	{id: `tuple.go|CheckTupleAdmission|entry.Key.ProviderID != binding.ProviderID|0`, driver: "TestTupleAdmissionZeroFactsRefuse", domain: "admission", reason: "zero provider on either side refuses"},
	{id: `tuple.go|CheckTupleAdmission|entry.Key.ProviderManifestDigest != binding.ProviderManifestDigest|0`, driver: "TestTupleAdmissionZeroFactsRefuse", domain: "admission", reason: "zero provider digest on either side refuses"},
	{id: `tuple.go|CheckTupleAdmission|entry.Status != "accepted"|0`, driver: "TestTupleAdmissionZeroFactsRefuse", domain: "admission", reason: "empty entry status refuses"},
	{id: `tuple.go|CheckTupleAdmission|entry.Strategies[0] != "archive_only"|0`, driver: "TestTupleAdmissionZeroFactsRefuse", reason: "empty strategy word refuses; the target branch is pinned by TestCheckTupleAdmissionRefusals"},
	{id: `tuple.go|checkContractsShape|previous != ""|0`, exempt: true, reason: "first-iteration sortedness sentinel, not a fact gate; contract vocabulary refuses \"\" upstream"},
	{id: `tuple.go|checkContractsShape|previousVersion != ""|0`, exempt: true, reason: "first-iteration sortedness sentinel, not a fact gate; SemVer refuses \"\" upstream"},
	// ==-spelled gates (review rev5/F1). The zero-fact class is
	// spelled both ways: Discover refuses empty facts with ==,
	// the binding echoes with !=. Membership positives over a
	// rostered table are driven at the loop by
	// TestVocabularyMembershipRefusesEmpty, which refuses "" at
	// every one; the tables stay pinned by
	// TestClosedVocabularyTablesAreRegistered.
	// context.go: the fresh-sink conjunction. Both zero arms
	// refuse through DecodeObjectAuthority; read mode with zero
	// limits admits, so the mode half dispatches between the two
	// rules and both directions are pinned by the same test.
	{id: `context.go|DecodeObjectAuthority|maxObjects == 0|0`, driver: "TestDecodeObjectAuthorityRules", domain: "authority", reason: "zero objects on a fresh sink refuses"},
	{id: `context.go|DecodeObjectAuthority|maxTotalBytes == 0|0`, driver: "TestDecodeObjectAuthorityRules", domain: "authority", reason: "zero bytes on a fresh sink refuses"},
	{id: `context.go|DecodeObjectAuthority|mode == "fresh_sink"|0`, exempt: true, reason: "dispatch: selects the fresh-sink limits rule; read mode with zero limits admits, pinned by TestDecodeObjectAuthorityRules; \"\" refused upstream by validObjectMode"},
	{id: `context.go|validFindingSeverity|severity == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "three-severity membership positive"},
	{id: `context.go|validObjectMode|mode == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "two-mode membership positive"},
	{id: `context.go|validObjectPurpose|purpose == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "six-purpose membership positive"},
	{id: `context.go|validReadPurpose|purpose == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "three-purpose membership positive"},
	// decode.go: the generic null detector behind every
	// nullability check, and the defensive empty-literal arm of
	// the uint53 reader.
	{id: `decode.go|isNull|string(bytes.TrimSpace(raw)) == "null"|0`, driver: "TestCheckValidateNullabilityDrivesEveryTarget", domain: "validate", reason: "null detector; both directions pinned by the archive/staged target matrix"},
	{id: `decode.go|rawUint53|literal == ""|0`, exempt: true, reason: "defensive: unreachable — json.Number.String never yields an empty literal, and every non-number is refused at the decode arms above"},
	// discovery.go: the two live zero-fact refusals F1 names.
	{id: `discovery.go|Discover|candidate.ExecutablePath == ""|0`, driver: "TestDiscoverRefusals", domain: "discover", reason: "empty executable path refuses at Discover"},
	{id: `discovery.go|Discover|candidate.OwnerIdentity == ""|0`, driver: "TestDiscoverRefusals", domain: "discover", reason: "empty owner identity refuses at Discover"},
	{id: `discovery.go|validRole|string(role) == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "two-role membership positive"},
	// operations.go: the context-free table consultation, the
	// archive dispatch, the partial/cursor coherence refusal, the
	// resume-identity counter behind the exactly-once gate, and
	// the error-finding refusal.
	{id: `operations.go|checkSuccessScalars|partial == cursorNull|0`, driver: "TestDiscoverPartialCursorComplement", domain: "discover-cursor", reason: "incoherent partial/cursor pair refuses"},
	{id: `operations.go|checkSuccessScalars|word == facts.ResumeTargetID|0`, exempt: true, reason: "counter increment, not a refusal; the exactly-once refusal is pinned by TestResumePlanIdentityComplement at occurrences != 1"},
	{id: `operations.go|checkValidateNullability|mode == "archive"|0`, exempt: true, reason: "dispatch: selects the archive-carries-no-target rule; both modes pinned by TestCheckValidateNullabilityDrivesEveryTarget; \"\" refused upstream by validValidateMode"},
	{id: `operations.go|checkValidateResult|finding.Severity == "error"|0`, driver: "TestValidateResultRules", domain: "validate", reason: "valid result with an error finding refuses"},
	{id: `operations.go|operationSkipsRequestContext|operation == free|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "two-operation table consultation; table pinned by TestClosedVocabularyTablesAreRegistered"},
	{id: `operations.go|validFidelityProfile|profile == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "profile membership positive"},
	{id: `operations.go|validValidateMode|mode == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "validate-mode membership positive"},
	// probe.go: the two usability predicates (review rev5/F3).
	// The decode coherence gate refuses enabled-without-available
	// at the boundary, and these clauses independently refuse a
	// caller-built non-available status even with enabled=true,
	// pinned by the hand-built capability tests below.
	{id: `probe.go|CapabilityUsable|value.Status == "available"|0`, driver: "TestCapabilityNonAvailableEnabledIsNotUsable", domain: "capability-gates", reason: "hand-built conditional+enabled refuses at CapabilityUsable"},
	{id: `probe.go|capabilityMapUsable|value.Status == "available"|0`, driver: "TestCheckTargetWriteGatesRefusesNonAvailableEnabled", domain: "capability-gates", reason: "hand-built conditional+enabled refuses at CheckTargetWriteGates"},
	{id: `probe.go|validCapabilityEvidence|evidence == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "seven-evidence membership positive"},
	{id: `probe.go|validCapabilityName|capability == name|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "fifteen-name registry consultation"},
	{id: `probe.go|validCapabilityStatus|status == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "four-status membership positive"},
	{id: `probe.go|validRegistryEntryStatus|status == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "three-status membership positive"},
	// protocol.go: the operation-table consultation.
	{id: `protocol.go|validOperation|string(operation) == name|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "operation-table membership positive"},
	// tuple.go: the source/target dispatch, the target-side
	// archive_only refusal, the accepted/revoked dispatch, and
	// the vocabulary consultations.
	{id: `tuple.go|CheckTupleAdmission|direction == DirectionSourceRead|0`, exempt: true, reason: "dispatch: selects the source strategies=[archive_only] rule; both directions pinned by TestCheckTupleAdmissionRefusals; an empty caller direction deterministically takes the target branch"},
	{id: `tuple.go|CheckTupleAdmission|strategy == "archive_only"|0`, driver: "TestCheckTupleAdmissionRefusals", domain: "admission", reason: "target entry admitting archive_only refuses"},
	{id: `tuple.go|DecodeTupleEntry|status == "accepted"|0`, exempt: true, reason: "dispatch: selects the accepted-carries-no-revocation rule; both halves pinned by TestDecodeTupleEntryCoherence; \"\" refused upstream by validTupleEntryStatus"},
	{id: `tuple.go|validCandidateKind|string(kind) == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "candidate-kind membership positive"},
	{id: `tuple.go|validCaptureClass|class == name|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "capture-class membership positive"},
	{id: `tuple.go|validDirection|name == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "direction membership positive"},
	{id: `tuple.go|validDisposition|disposition == name|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "disposition membership positive"},
	{id: `tuple.go|validEvidenceResult|result == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "evidence-result membership positive"},
	{id: `tuple.go|validStrategy|strategy == name|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "strategy membership positive"},
	{id: `tuple.go|validTupleArchitecture|architecture == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "architecture membership positive"},
	{id: `tuple.go|validTupleEntryStatus|status == allowed|0`, driver: "TestVocabularyMembershipRefusesEmpty", domain: "vocabulary", reason: "entry-status membership positive"},
}

// identityCensusCanaries are survivor gates from the review
// traversal that the scanner must see, in both spellings: an
// ==-blind scanner reports a short denominator as complete.
var identityCensusCanaries = []string{
	`probe.go|DecodeProbe|schema != probeSchema`,
	`probe.go|CheckProbe|probe.ProviderID != facts.ExpectedProviderID`,
	`operations.go|checkSuccessScalars|candidate != digest`,
	`context.go|VerifyRequestDigest|digest.String() != context.RequestDigest`,
	`tuple.go|CheckTupleAdmission|entry.Status != "accepted"`,
	`discovery.go|CheckBindingEquality|sealed.Role != fresh.Role`,
	`discovery.go|Discover|candidate.ExecutablePath == ""`,
	`context.go|CheckContextEcho|bytes.Equal`,
	`probe.go|CapabilityUsable|value.Status == "available"`,
	`tuple.go|CheckTupleAdmission|strategy == "archive_only"`,
	`context.go|DecodeObjectAuthority|maxObjects == 0`,
}

// TestIdentityGatesAreCensused is the identity census: the
// derived production gate multiset must equal the registration
// multiset. Every non-exempt row must name a driver and every
// exempt row a rationale.
func TestIdentityGatesAreCensused(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("identity census: %v", err)
	}
	derived := scanIdentityGateIDs(t, directory)
	if len(derived) == 0 {
		t.Fatal("identity census derived zero gates; the scanner is broken, not the package")
	}
	derivedCounts := map[string]int{}
	for _, id := range derived {
		derivedCounts[id]++
	}
	for _, canary := range identityCensusCanaries {
		found := false
		for id := range derivedCounts {
			if strings.HasPrefix(id, canary) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("identity census blind to canary %q; the scanner misses the class it must roster", canary)
		}
	}
	registeredCounts := map[string]int{}
	seen := map[string]bool{}
	for _, row := range identityRegistrations {
		if seen[row.id] {
			t.Errorf("identity census registers %q twice", row.id)
		}
		seen[row.id] = true
		registeredCounts[row.id]++
		if row.driver == "" && !row.exempt {
			t.Errorf("identity census row %q names no driver and no exemption", row.id)
		}
		if row.exempt && row.reason == "" {
			t.Errorf("identity census exemption %q states no rationale", row.id)
		}
	}
	var unregistered []string
	for id, count := range derivedCounts {
		for i := registeredCounts[id]; i < count; i++ {
			unregistered = append(unregistered, id)
		}
	}
	var orphaned []string
	for id, count := range registeredCounts {
		for i := derivedCounts[id]; i < count; i++ {
			orphaned = append(orphaned, id)
		}
	}
	sort.Strings(unregistered)
	sort.Strings(orphaned)
	if len(unregistered) > 0 {
		t.Fatalf("production identity gate(s) with no census row:\n  %s", strings.Join(unregistered, "\n  "))
	}
	if len(orphaned) > 0 {
		t.Fatalf("census row(s) naming no production gate:\n  %s", strings.Join(orphaned, "\n  "))
	}
	driven, exempt := 0, 0
	for _, row := range identityRegistrations {
		if row.exempt {
			exempt++
		} else {
			driven++
		}
	}
	t.Logf("identity census: %d gates rostered (%d driven, %d exempt) across production", len(derived), driven, exempt)
}

// TestIdentityScanSeesBothSpellings proves the spelling split of
// the derivation against synthetic files: an ==-spelled refusal
// and a !=-spelled refusal are both derived, while the
// mechanical classes stay excluded in both spellings. The
// vectors are synthetic on purpose: they prove the gate, not
// production. This is the regression test for review rev5/F1,
// where the NEQ-only scan passed an ==-spelled plant silently.
func TestIdentityScanSeesBothSpellings(t *testing.T) {
	parse := func(source string) *ast.File {
		t.Helper()
		syntax, err := parser.ParseFile(token.NewFileSet(), "probe.go", []byte(source), 0)
		if err != nil {
			t.Fatalf("parse synthetic source: %v", err)
		}
		return syntax
	}
	derived := func(source string) []string {
		t.Helper()
		return identityGatesInSyntax(parse(source), "probe.go")
	}
	contains := func(ids []string, want string) bool {
		t.Helper()
		for _, id := range ids {
			if strings.HasPrefix(id, want) {
				return true
			}
		}
		return false
	}
	t.Run("equality refusal derives", func(t *testing.T) {
		ids := derived("package probe\nfunc zzProbeEqualityRefusal(observed, expected string) error {\nif observed == expected { return nil }\nreturn json.Unmarshal(nil, nil)\n}\n")
		if !contains(ids, "probe.go|zzProbeEqualityRefusal|observed == expected|") {
			t.Fatalf("derived = %v, want the equality refusal", ids)
		}
	})
	t.Run("inequality refusal derives", func(t *testing.T) {
		ids := derived("package probe\nfunc zzProbeInequalityRefusal(observed, expected string) bool {\nif observed != expected { return false }\nreturn true\n}\n")
		if !contains(ids, "probe.go|zzProbeInequalityRefusal|observed != expected|") {
			t.Fatalf("derived = %v, want the inequality refusal", ids)
		}
	})
	t.Run("mechanical classes stay excluded in both spellings", func(t *testing.T) {
		ids := derived("package probe\nfunc zzMechanical(a, b string, err error) bool {\nif err == nil { return false }\nif a == b && len(a) == 1 { return true }\nreturn a != b\n}\n")
		for _, id := range ids {
			if strings.Contains(id, "err ==") || strings.Contains(id, "len(a)") {
				t.Fatalf("derived = %v, want the nil and length plumbing excluded", ids)
			}
		}
		if !contains(ids, "probe.go|zzMechanical|a != b|") {
			t.Fatalf("derived = %v, want the domain inequality kept", ids)
		}
	})
}

// TestIdentityScanSeesPackageLiteralsAndBytesEqual proves the two
// review-rev6 extensions against synthetic files: a gate inside a
// package-level func literal derives under its variable name in
// both the comparison and the bytes.Equal spelling, a bytes.Equal
// gate inside a FuncDecl derives, and non-identity bytes uses
// (NewReader, TrimSpace) never derive. The vectors are synthetic
// on purpose: they prove the gate, not production. This is the
// regression test for the rev6 G-B plants, which hid a != gate
// and a bytes.Equal gate one declaration level above the FuncDecl
// walk and passed the whole suite unrostered.
func TestIdentityScanSeesPackageLiteralsAndBytesEqual(t *testing.T) {
	parse := func(source string) *ast.File {
		t.Helper()
		syntax, err := parser.ParseFile(token.NewFileSet(), "probe.go", []byte(source), 0)
		if err != nil {
			t.Fatalf("parse synthetic source: %v", err)
		}
		return syntax
	}
	derived := func(source string) []string {
		t.Helper()
		return identityGatesInSyntax(parse(source), "probe.go")
	}
	contains := func(ids []string, want string) bool {
		t.Helper()
		for _, id := range ids {
			if strings.HasPrefix(id, want) {
				return true
			}
		}
		return false
	}
	t.Run("comparison in package literal derives", func(t *testing.T) {
		ids := derived("package probe\nvar zzProbeLiteralGate = func(observed, expected string) error {\nif observed != expected { return nil }\nreturn nil\n}\n")
		if !contains(ids, "probe.go|zzProbeLiteralGate|observed != expected|") {
			t.Fatalf("derived = %v, want the package-literal gate under its variable name", ids)
		}
	})
	t.Run("bytes.Equal in package literal derives", func(t *testing.T) {
		ids := derived("package probe\nimport \"bytes\"\nvar zzProbeLiteralBytes = func(received, sent []byte) bool {\nreturn !bytes.Equal(received, sent)\n}\n")
		if !contains(ids, "probe.go|zzProbeLiteralBytes|bytes.Equal(received, sent)|") {
			t.Fatalf("derived = %v, want the package-literal bytes.Equal gate", ids)
		}
	})
	t.Run("bytes.Equal in function derives", func(t *testing.T) {
		ids := derived("package probe\nimport \"bytes\"\nfunc zzProbeByteIdentity(received, sent []byte) bool {\nreturn !bytes.Equal(received, sent)\n}\n")
		if !contains(ids, "probe.go|zzProbeByteIdentity|bytes.Equal(received, sent)|") {
			t.Fatalf("derived = %v, want the function bytes.Equal gate", ids)
		}
	})
	t.Run("non-identity bytes uses stay excluded", func(t *testing.T) {
		ids := derived("package probe\nimport \"bytes\"\nimport \"encoding/json\"\nfunc zzProbeBytesPlumbing(raw json.RawMessage) bool {\ntrimmed := bytes.TrimSpace(raw)\n_ = json.NewDecoder(bytes.NewReader(trimmed))\nreturn len(trimmed) > 0\n}\n")
		for _, id := range ids {
			if strings.Contains(id, "bytes.") {
				t.Fatalf("derived = %v, want no identity gate for plumbing bytes uses", ids)
			}
		}
	})
}

// TestVocabularyMembershipRefusesEmpty drives every
// vocabulary-table membership loop with the empty string: "" is
// outside every closed union, so every loop must report false,
// and a mutant admitting "" at any one loop (`x == allowed || x
// == ""`) fails here. A bogus non-empty member is refused too,
// pinning the loop against an arbitrary widening. The tables
// themselves stay pinned by
// TestClosedVocabularyTablesAreRegistered; this test pins the
// consultation. It is the named driver of every domain
// "vocabulary" row.
func TestVocabularyMembershipRefusesEmpty(t *testing.T) {
	strings := map[string]func(string) bool{
		"validFindingSeverity":     validFindingSeverity,
		"validObjectMode":          validObjectMode,
		"validObjectPurpose":       validObjectPurpose,
		"validReadPurpose":         validReadPurpose,
		"validFidelityProfile":     validFidelityProfile,
		"validValidateMode":        validValidateMode,
		"validCapabilityEvidence":  validCapabilityEvidence,
		"validCapabilityStatus":    validCapabilityStatus,
		"validRegistryEntryStatus": validRegistryEntryStatus,
		"validCapabilityName":      validCapabilityName,
		"validOperation":           validOperation,
		"validDirection":           validDirection,
		"validTupleEntryStatus":    validTupleEntryStatus,
		"validEvidenceResult":      validEvidenceResult,
		"validStrategy":            validStrategy,
		"validDisposition":         validDisposition,
		"validCaptureClass":        validCaptureClass,
		"validTupleArchitecture":   validTupleArchitecture,
	}
	if len(strings) != 18 {
		t.Fatalf("string loops = %d, want one per string membership gate", len(strings))
	}
	for name, loop := range strings {
		if loop("") {
			t.Errorf("%s(\"\") = true; the empty string is outside every union", name)
		}
		if loop("bogus") {
			t.Errorf("%s(\"bogus\") = true; an arbitrary member is outside every union", name)
		}
	}
	if validRole(Role("")) {
		t.Error("validRole(\"\") = true; the empty role is outside source|target")
	}
	if validRole("bogus") {
		t.Error("validRole(\"bogus\") = true; an arbitrary role is outside source|target")
	}
	if validCandidateKind(CandidateKind("")) {
		t.Error("validCandidateKind(\"\") = true; the empty kind is outside the union")
	}
	if validCandidateKind(CandidateKind("bogus")) {
		t.Error("validCandidateKind(\"bogus\") = true; an arbitrary kind is outside the union")
	}
	if operationSkipsRequestContext(Operation("")) {
		t.Error("operationSkipsRequestContext(\"\") = true; the empty operation skips nothing")
	}
	if operationSkipsRequestContext(Operation("bogus")) {
		t.Error("operationSkipsRequestContext(\"bogus\") = true; an arbitrary operation skips nothing")
	}
}

// identityCensusDomainCount returns the number of non-exempt
// rows in the given domain. Sibling tables with fixed tripwires
// assert against it so they cannot go stale.
func identityCensusDomainCount(domain string) int {
	count := 0
	for _, row := range identityRegistrations {
		if !row.exempt && row.domain == domain {
			count++
		}
	}
	return count
}

// probeFactsFor builds the CheckProbe facts for the fixture
// manifest: the expected provider, builtin kind, host-computed
// digest, and verified manifest.
func probeFactsFor(manifest Manifest) ProbeHostFacts {
	return ProbeHostFacts{
		ExpectedProviderID: fixtureProviderID,
		ExpectedKind:       CandidateBuiltin,
		ManifestDigest:     ManifestDigest(manifest).String(),
		Manifest:           manifest,
	}
}

// TestIdentityZeroRows drives every surviving zero-value gate at
// its exported entry: the zero fact refuses through its own arm,
// so a mutant exempting the zero (`!= want && fact != ""`)
// admits and reddens here. requireRefusal pins the arm
// identity, so a slide to a neighbouring arm reddens too.
func TestIdentityZeroRows(t *testing.T) {
	t.Run("probe schema empty", func(t *testing.T) {
		probe, manifest := fixtureValidProbe(t)
		_ = probe
		digest := ManifestDigest(manifest).String()
		base := []byte(fixtureProbeJSON(digest))
		if _, err := DecodeProbe(mutateMember(t, base, "schema", `""`)); err == nil {
			t.Fatal("DecodeProbe admitted an empty schema")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "schema", "not the session adapter probe")
		}
		if _, err := DecodeProbe(mutateMember(t, base, "schema_version", `""`)); err == nil {
			t.Fatal("DecodeProbe admitted an empty schema_version")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "schema_version", "is not 1.0.0")
		}
	})
	t.Run("probe provider zeros", func(t *testing.T) {
		probe, manifest := fixtureValidProbe(t)
		facts := probeFactsFor(manifest)
		providerless := probe
		providerless.ProviderID = ""
		if err := CheckProbe(providerless, facts); err == nil {
			t.Fatal("CheckProbe admitted an empty probe provider")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "provider_id", "requested provider")
		}
		// Facts-side zero of the expected provider is pinned by
		// TestZeroValueHostFactsRefuse; here the manifest side
		// goes zero with probe and expectation held equal.
		manifestZero := facts
		manifestZero.ExpectedProviderID = "other-provider"
		zeroed := manifest
		zeroed.ProviderID = ""
		manifestZero.Manifest = zeroed
		other := probe
		other.ProviderID = "other-provider"
		if err := CheckProbe(other, manifestZero); err == nil {
			t.Fatal("CheckProbe admitted an empty manifest provider")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "provider_id", "verified manifest")
		}
	})
	t.Run("probe digest zeros", func(t *testing.T) {
		probe, manifest := fixtureValidProbe(t)
		facts := probeFactsFor(manifest)
		digestless := probe
		digestless.ManifestDigest = ""
		if err := CheckProbe(digestless, facts); err == nil {
			t.Fatal("CheckProbe admitted an empty probe digest")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "adapter_manifest_digest", "host-computed digest")
		}
		factsless := facts
		factsless.ManifestDigest = ""
		if err := CheckProbe(probe, factsless); err == nil {
			t.Fatal("CheckProbe admitted an empty expected digest")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "adapter_manifest_digest", "host-computed digest")
		}
	})
	t.Run("probe version zeros", func(t *testing.T) {
		probe, manifest := fixtureValidProbe(t)
		facts := probeFactsFor(manifest)
		versionless := probe
		versionless.AdapterVersion = ""
		if err := CheckProbe(versionless, facts); err == nil {
			t.Fatal("CheckProbe admitted an empty probe version")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "adapter_version", "verified manifest")
		}
		manifestless := facts
		zeroed := manifest
		zeroed.AdapterVersion = ""
		manifestless.Manifest = zeroed
		if err := CheckProbe(probe, manifestless); err == nil {
			t.Fatal("CheckProbe admitted an empty manifest version")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "adapter_version", "verified manifest")
		}
		driftless := probe
		driftless.Environment.AdapterVersion = ""
		if err := CheckProbe(driftless, facts); err == nil {
			t.Fatal("CheckProbe admitted an empty tuple adapter version")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "environment", "probe adapter version")
		}
	})
	t.Run("probe tuple environment zero", func(t *testing.T) {
		probe, manifest := fixtureValidProbe(t)
		facts := probeFactsFor(manifest)
		tupleless := probe
		tupleless.Environment.EnvironmentID = ""
		if err := CheckProbe(tupleless, facts); err == nil {
			t.Fatal("CheckProbe admitted an empty tuple environment")
		} else {
			requireRefusal(t, err, "unsupported_environment_tuple", "environment", "", "outside the manifest environment")
		}
	})
	t.Run("probe request zeros", func(t *testing.T) {
		candidate := fixtureCandidate()
		if err := CheckProbeRequest("", CandidateBuiltin, candidate); err == nil {
			t.Fatal("CheckProbeRequest admitted an empty expected provider")
		} else {
			requireRefusal(t, err, "invalid_config", "field", "expected_provider_id", "trusted candidate")
		}
		providerless := candidate
		providerless.ProviderID = ""
		if err := CheckProbeRequest(fixtureProviderID, CandidateBuiltin, providerless); err == nil {
			t.Fatal("CheckProbeRequest admitted an empty candidate provider")
		} else {
			requireRefusal(t, err, "invalid_config", "field", "expected_provider_id", "trusted candidate")
		}
	})
	t.Run("doctor direction zero", func(t *testing.T) {
		manifestDigest := fixtureManifestDigestText(t)
		request := fixtureRequestBody(t, OpDoctor, manifestDigest)
		decoded, err := CheckRequestBody(OpDoctor, request)
		if err != nil {
			t.Fatalf("CheckRequestBody: %v", err)
		}
		good := fixtureSuccessBody(t, OpDoctor, requestContextOf(t, request), manifestDigest)
		if _, err := DecodeDoctorResult(good, decoded, ""); err == nil {
			t.Fatal("DecodeDoctorResult admitted an empty request direction")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "direction", "does not answer the request direction")
		}
	})
	t.Run("doctor entry status zero", func(t *testing.T) {
		manifestDigest := fixtureManifestDigestText(t)
		request := fixtureRequestBody(t, OpDoctor, manifestDigest)
		decoded, err := CheckRequestBody(OpDoctor, request)
		if err != nil {
			t.Fatalf("CheckRequestBody: %v", err)
		}
		probe, _ := fixtureValidProbe(t)
		good := fixtureSuccessBody(t, OpDoctor, requestContextOf(t, request), manifestDigest)
		healthy, err := DecodeDoctorResult(mutateMember(t, good, "healthy", `true`), decoded, DirectionSourceRead)
		if err != nil {
			t.Fatalf("DecodeDoctorResult: %v", err)
		}
		statusless := healthy
		statusless.EntryStatus = ""
		required := DoctorRequiredCapabilities(DirectionTargetWrite, false)
		if err := CheckDoctorHealthy(statusless, probe, required); err == nil {
			t.Fatal("CheckDoctorHealthy admitted an empty entry status as healthy")
		} else {
			requireRefusal(t, err, "capability_unavailable", "capability", "tuple_registry", "accepted registry entry")
		}
	})
	t.Run("request digest zero", func(t *testing.T) {
		manifestDigest := fixtureManifestDigestText(t)
		body := fixtureRequestBody(t, OpDiscover, manifestDigest)
		decoded, err := CheckRequestBody(OpDiscover, body)
		if err != nil {
			t.Fatalf("CheckRequestBody: %v", err)
		}
		if err := VerifyRequestDigest(decoded, body); err != nil {
			t.Fatalf("VerifyRequestDigest baseline: %v", err)
		}
		digestless := decoded
		digestless.RequestDigest = ""
		if err := VerifyRequestDigest(digestless, body); err == nil {
			t.Fatal("VerifyRequestDigest admitted an empty request digest")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "request_digest", "does not match")
		}
	})
	t.Run("discover provider zeros", func(t *testing.T) {
		manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
		if err != nil {
			t.Fatalf("DecodeManifest: %v", err)
		}
		digest := ManifestDigest(manifest).String()
		candidate := fixtureCandidate()
		providerless := manifest
		providerless.ProviderID = ""
		if _, err := Discover(RoleSource, providerless, digest, candidate); err == nil {
			t.Fatal("Discover admitted an empty manifest provider")
		} else {
			requireRefusal(t, err, "integrity_failure", "subject", "provider_id", "trusted candidate")
		}
		candidateless := candidate
		candidateless.ProviderID = ""
		if _, err := Discover(RoleSource, manifest, digest, candidateless); err == nil {
			t.Fatal("Discover admitted an empty candidate provider")
		} else {
			requireRefusal(t, err, "integrity_failure", "subject", "provider_id", "trusted candidate")
		}
	})
	t.Run("call binding caller zeros", func(t *testing.T) {
		manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
		if err != nil {
			t.Fatalf("DecodeManifest: %v", err)
		}
		binding, err := Discover(RoleSource, manifest, ManifestDigest(manifest).String(), fixtureCandidate())
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		body := fixtureRequestWithDigest(t, ManifestDigest(manifest).String())
		context, err := CheckRequestBody(OpDiscover, body)
		if err != nil {
			t.Fatalf("CheckRequestBody: %v", err)
		}
		admitted, err := DecodeTuple([]byte(fixtureTupleJSON()))
		if err != nil {
			t.Fatalf("DecodeTuple: %v", err)
		}
		providerless := context
		providerless.ProviderID = ""
		if err := CheckCallBinding(binding, RoleSource, providerless, admitted); err == nil {
			t.Fatal("CheckCallBinding admitted an empty context provider")
		} else {
			requireRefusal(t, err, "integrity_failure", "subject", "provider_id", "sealed binding")
		}
		digestless := context
		digestless.ManifestDigest = ""
		if err := CheckCallBinding(binding, RoleSource, digestless, admitted); err == nil {
			t.Fatal("CheckCallBinding admitted an empty context digest")
		} else {
			requireRefusal(t, err, "integrity_failure", "subject", "session_adapter_manifest_digest", "sealed binding")
		}
		exeless := context
		exeless.ExecutableSHA256 = ""
		if err := CheckCallBinding(binding, RoleSource, exeless, admitted); err == nil {
			t.Fatal("CheckCallBinding admitted an empty context executable digest")
		} else {
			requireRefusal(t, err, "integrity_failure", "subject", "executable_sha256", "sealed binding")
		}
		if err := CheckCallBinding(binding, "", context, admitted); err == nil {
			t.Fatal("CheckCallBinding admitted an empty call role")
		} else {
			requireRefusal(t, err, "integrity_failure", "subject", "role", "binding role")
		}
	})
	t.Run("provider status zero", func(t *testing.T) {
		probe, _ := fixtureValidProbe(t)
		provider := map[string]ProviderCapability{
			"portable_store": {Status: "", Enabled: true},
			"native_resume":  {Status: "available", Enabled: true},
		}
		if err := CheckTargetWriteGates(probeCapabilities(probe), provider); err == nil {
			t.Fatal("CheckTargetWriteGates admitted an empty provider status")
		} else {
			requireRefusal(t, err, "capability_unavailable", "capability", "provider:portable_store", "usable provider capability")
		}
	})
}

// probeCapabilities projects the probe map for gate tests.
func probeCapabilities(probe Probe) map[string]Capability {
	return cloneCapabilities(probe.Capabilities)
}

// TestBindingEqualityZeroFactsRefuse drives every sealed-binding
// fact to zero on each side: a forgotten seal fact refuses,
// never agrees. Non-zero flips were already pinned by
// TestBindingEqualityFlipsEveryFact; a mutant exempting the zero
// admits here and reddens.
func TestBindingEqualityZeroFactsRefuse(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	sealed, err := Discover(RoleSource, manifest, ManifestDigest(manifest).String(), fixtureCandidate())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	zeroes := []struct {
		name string
		zero func(*ExecutionBinding)
	}{
		{"role", func(binding *ExecutionBinding) { binding.Role = "" }},
		{"provider", func(binding *ExecutionBinding) { binding.ProviderID = "" }},
		{"kind", func(binding *ExecutionBinding) { binding.CandidateKind = "" }},
		{"path", func(binding *ExecutionBinding) { binding.ExecutablePath = "" }},
		{"owner", func(binding *ExecutionBinding) { binding.OwnerIdentity = "" }},
		{"executable", func(binding *ExecutionBinding) { binding.ExecutableSHA256 = "" }},
		{"provider digest", func(binding *ExecutionBinding) { binding.ProviderManifestDigest = "" }},
		{"adapter digest", func(binding *ExecutionBinding) { binding.AdapterManifestDigest = "" }},
	}
	if len(zeroes) != 8 {
		t.Fatalf("zeroes = %d, want one per identity fact", len(zeroes))
	}
	for _, fact := range zeroes {
		fact := fact
		t.Run("fresh "+fact.name, func(t *testing.T) {
			fresh := sealed
			fact.zero(&fresh)
			if err := CheckBindingEquality(sealed, fresh); err == nil {
				t.Fatalf("CheckBindingEquality admitted a zero fresh %s", fact.name)
			} else {
				requireRefusal(t, err, "integrity_failure", "subject", "binding", "freshly read trusted facts")
			}
		})
		t.Run("sealed "+fact.name, func(t *testing.T) {
			resealed := sealed
			fact.zero(&resealed)
			if err := CheckBindingEquality(resealed, sealed); err == nil {
				t.Fatalf("CheckBindingEquality admitted a zero sealed %s", fact.name)
			} else {
				requireRefusal(t, err, "integrity_failure", "subject", "binding", "freshly read trusted facts")
			}
		})
	}
}

// TestTupleAdmissionZeroFactsRefuse drives every admission key
// fact, the direction, the environment, and the status to zero:
// an empty fact refuses, never admits. Binding-side zeroes are
// driven too, since the key disjunction compares both sides.
func TestTupleAdmissionZeroFactsRefuse(t *testing.T) {
	entry, err := DecodeTupleEntry([]byte(fixtureEntryJSON(DirectionTargetWrite)))
	if err != nil {
		t.Fatalf("DecodeTupleEntry: %v", err)
	}
	tuple, err := DecodeTuple([]byte(fixtureTupleJSON()))
	if err != nil {
		t.Fatalf("DecodeTuple: %v", err)
	}
	binding := fixtureBindingFacts()
	now := mustTime(t, fixtureCallTime)
	if err := CheckTupleAdmission(entry, DirectionTargetWrite, tuple, binding, now); err != nil {
		t.Fatalf("CheckTupleAdmission baseline: %v", err)
	}
	t.Run("empty direction", func(t *testing.T) {
		directionless := entry
		directionless.Key.Direction = ""
		if err := CheckTupleAdmission(directionless, DirectionTargetWrite, tuple, binding, now); err == nil {
			t.Fatal("admitted an empty entry direction")
		} else {
			requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "direction does not match")
		}
	})
	t.Run("zero environment", func(t *testing.T) {
		environmentless := entry
		environmentless.Key.Environment = Tuple{}
		if err := CheckTupleAdmission(environmentless, DirectionTargetWrite, tuple, binding, now); err == nil {
			t.Fatal("admitted a zero entry environment")
		} else {
			requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "probed tuple")
		}
	})
	keyZeroes := []struct {
		name string
		zero func(*TupleEntry)
	}{
		{"provider", func(entry *TupleEntry) { entry.Key.ProviderID = "" }},
		{"kind", func(entry *TupleEntry) { entry.Key.CandidateKind = "" }},
		{"executable", func(entry *TupleEntry) { entry.Key.ExecutableSHA256 = "" }},
		{"provider digest", func(entry *TupleEntry) { entry.Key.ProviderManifestDigest = "" }},
		{"adapter digest", func(entry *TupleEntry) { entry.Key.AdapterManifestDigest = "" }},
	}
	for _, fact := range keyZeroes {
		fact := fact
		t.Run("entry "+fact.name, func(t *testing.T) {
			zeroed := entry
			fact.zero(&zeroed)
			if err := CheckTupleAdmission(zeroed, DirectionTargetWrite, tuple, binding, now); err == nil {
				t.Fatalf("admitted a zero entry %s", fact.name)
			} else {
				requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "execution binding")
			}
		})
	}
	bindingZeroes := []struct {
		name string
		zero func(*BindingFacts)
	}{
		{"provider", func(binding *BindingFacts) { binding.ProviderID = "" }},
		{"kind", func(binding *BindingFacts) { binding.CandidateKind = "" }},
		{"executable", func(binding *BindingFacts) { binding.ExecutableSHA256 = "" }},
		{"provider digest", func(binding *BindingFacts) { binding.ProviderManifestDigest = "" }},
		{"adapter digest", func(binding *BindingFacts) { binding.AdapterManifestDigest = "" }},
	}
	for _, fact := range bindingZeroes {
		fact := fact
		t.Run("binding "+fact.name, func(t *testing.T) {
			zeroed := binding
			fact.zero(&zeroed)
			if err := CheckTupleAdmission(entry, DirectionTargetWrite, tuple, zeroed, now); err == nil {
				t.Fatalf("admitted a zero binding %s", fact.name)
			} else {
				requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "execution binding")
			}
		})
	}
	t.Run("empty status", func(t *testing.T) {
		statusless := entry
		statusless.Status = ""
		if err := CheckTupleAdmission(statusless, DirectionTargetWrite, tuple, binding, now); err == nil {
			t.Fatal("admitted an empty entry status")
		} else {
			requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "not accepted")
		}
	})
	t.Run("empty strategy word", func(t *testing.T) {
		source, err := DecodeTupleEntry([]byte(fixtureEntryJSON(DirectionSourceRead)))
		if err != nil {
			t.Fatalf("DecodeTupleEntry source: %v", err)
		}
		wordless := source
		wordless.Strategies = []string{""}
		if err := CheckTupleAdmission(wordless, DirectionSourceRead, tuple, binding, now); err == nil {
			t.Fatal("admitted a source entry with an empty strategy word")
		} else {
			requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "exactly strategies=[archive_only]")
		}
	})
}

// presenceRegistration is one map-presence default with its
// fail-closed proof: the behavioral driver plus the structural
// shape the census asserts in production source.
type presenceRegistration struct {
	function string
	file     string
	outcome  string
	driver   string
}

// presenceRegistrations rosters every map-presence default in
// production: an absent name must refuse or read unusable, never
// default to admitted. missingMember is the generic mechanism
// behind every closed-body check.
var presenceRegistrations = []presenceRegistration{
	{function: "decodeCapabilities", file: "probe.go", outcome: "refuse", driver: "TestDecodeProbeClosedRules"},
	{function: "CapabilityUsable", file: "probe.go", outcome: "false", driver: "TestCapabilityUsableAbsentIsNotUsable"},
	{function: "capabilityMapUsable", file: "probe.go", outcome: "false", driver: "TestCapabilityUsableAbsentIsNotUsable"},
	{function: "CheckTargetWriteGates", file: "probe.go", outcome: "refuse", driver: "TestCheckTargetWriteGates"},
	{function: "missingMember", file: "decode.go", outcome: "report", driver: "TestClosedMemberSetsAreDerivedFromSpec"},
}

// TestPresenceDefaultsAreFailClosed asserts the structural shape
// of every rostered presence default: the function body guards
// on !present and the guarded arm refuses (a fail* construction)
// or reads unusable (a false return). A default rewritten to
// admit an absent name fails here before any behavior runs.
func TestPresenceDefaultsAreFailClosed(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("presence census: %v", err)
	}
	for _, row := range presenceRegistrations {
		row := row
		t.Run(row.function, func(t *testing.T) {
			path := filepath.Join(directory, row.file)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("presence census: %v", err)
			}
			syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
			if err != nil {
				t.Fatalf("presence census: %v", err)
			}
			found := false
			for _, declaration := range syntax.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Name.Name != row.function || function.Body == nil {
					continue
				}
				ast.Inspect(function.Body, func(node ast.Node) bool {
					statement, ok := node.(*ast.IfStmt)
					if !ok {
						return true
					}
					if !hasPresentNegation(statement.Cond) {
						return true
					}
					switch row.outcome {
					case "false":
						ast.Inspect(statement.Body, func(inner ast.Node) bool {
							ret, ok := inner.(*ast.ReturnStmt)
							if ok && len(ret.Results) == 1 {
								if ident, ok := ret.Results[0].(*ast.Ident); ok && ident.Name == "false" {
									found = true
									return false
								}
							}
							return true
						})
					case "refuse", "report":
						ast.Inspect(statement.Body, func(inner ast.Node) bool {
							if call, ok := inner.(*ast.CallExpr); ok {
								if ident, ok := call.Fun.(*ast.Ident); ok && strings.HasPrefix(ident.Name, "fail") {
									found = true
									return false
								}
								return true
							}
							if ret, ok := inner.(*ast.ReturnStmt); ok {
								for _, result := range ret.Results {
									if ident, ok := result.(*ast.Ident); ok && ident.Name == "true" && row.outcome == "report" {
										found = true
										return false
									}
								}
							}
							return true
						})
					}
					return true
				})
			}
			if !found {
				t.Fatalf("%s in %s no longer guards !present with a fail-closed %s arm", row.function, row.file, row.outcome)
			}
		})
	}
}

// hasPresentNegation reports whether the condition negates the
// present/ok boolean: !present, !ok, or a disjunction holding
// one.
func hasPresentNegation(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(inner ast.Node) bool {
		unary, ok := inner.(*ast.UnaryExpr)
		if ok && unary.Op == token.NOT {
			if ident, ok := unary.X.(*ast.Ident); ok && (ident.Name == "present" || ident.Name == "ok") {
				found = true
				return false
			}
		}
		return true
	})
	return found
}
