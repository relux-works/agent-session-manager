package axerror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// duplicateMember returns document with member repeated, the second occurrence
// carrying replacement.
//
// The rewrite is textual on purpose. Every structured builder in this
// repository - encoding/json, the canonicalizer, this package's own encoder -
// is incapable of emitting the shape under test, because a Go map has one value
// per key. A document whose members repeat can only be built by writing the
// bytes, which is exactly how a hostile peer would produce it.
// The second occurrence is appended as the LAST member of the object, because
// which occurrence encoding/json keeps is the whole point: a scalar resolves to
// the last one, so the appended value is the one a resolving reader would answer
// with, and the row would be vacuous if the injected copy came first.
func duplicateMember(test *testing.T, document, member, replacement string) []byte {
	test.Helper()
	anchor := `"` + member + `":`
	if !strings.Contains(document, anchor) {
		test.Fatalf("document carries no %q member to duplicate", member)
	}
	closing := strings.LastIndex(document, "}")
	if closing < 0 {
		test.Fatal("document has no closing brace")
	}
	injected := document[:closing] + `, "` + member + `": ` + replacement + "\n" + document[closing:]
	if injected == document {
		test.Fatal("the duplicate-member rewrite did not apply")
	}
	// The rewrite must produce the shape under test rather than something that
	// merely fails to parse: exactly two occurrences of the member name, and
	// bytes that a lenient reader would still resolve to one object.
	if count := strings.Count(injected, anchor); count != 2 {
		test.Fatalf("the rewrite produced %d occurrences of %q, want 2", count, member)
	}
	var lenient map[string]json.RawMessage
	if err := json.Unmarshal([]byte(injected), &lenient); err != nil {
		test.Fatalf("the rewrite produced bytes encoding/json itself refuses: %v", err)
	}
	return []byte(injected)
}

// largeConformingDocument returns a document that is conforming in every
// respect and is several times longer than the hand-built fixtures in this
// file. It is padded through exactly one member the contract already bounds -
// the details map, up to its 64-key limit - so the result is a document a
// conforming writer could actually emit, and so the size has a single knob: a
// narrowed loop here drops the fixture back under the short-document range and
// reddens the caller's own length assertion instead of quietly shrinking it.
func largeConformingDocument(test *testing.T) string {
	test.Helper()
	pairs := make([]string, 0, 63)
	for index := 0; index < 63; index++ {
		pairs = append(pairs, fmt.Sprintf(`"pad_%03d": %q`, index, strings.Repeat("x", 60)))
	}
	document := strings.Replace(conformingDocument,
		`{"observed_sequence": "17"}`,
		`{"observed_sequence": "17", `+strings.Join(pairs, ", ")+`}`, 1)
	if document == conformingDocument {
		test.Fatal("the padding rewrite did not apply")
	}
	return document
}

// TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember is the negative half of
// the Section 1.6 common-data-model gate this leaf's review found missing.
//
// Section 1.6 states the rule twice, quoted verbatim from the pinned
// internal/specdoc/SPEC.md: "map keys MUST be UTF-8 strings and MUST be unique"
// at SPEC.md:218, and "floating-point numbers, NaN, Infinity, non-string map
// keys, and duplicate keys are forbidden" at SPEC.md:221. encoding/json
// does not refuse such a document. It resolves the repeat, and it resolves it
// differently depending on the decode target, so two conforming readers of the
// same bytes disagreed about the code, the retryable bit, and the detail set.
//
// Each row asserts three things, because a refusal alone would not prove the
// gate is the thing refusing: the duplicated document is refused with
// ErrInvalidStructuredError; the same document without the duplicate is
// admitted, so the row is not passing for an unrelated reason; and the refusal
// names the repeated member, so a gate narrowed to some other member set would
// redden rather than pass.
func TestDecodeRefusesADuplicateMemberOnEveryDeclaredMember(test *testing.T) {
	test.Parallel()

	// Every member of the closed nine-member set, each with a second occurrence
	// that is individually valid. A reader that resolved the repeat would admit
	// each of these and answer from one occurrence or the other.
	for _, row := range []struct {
		member      string
		replacement string
	}{
		{"schema", `"urn:ax:schema:error"`},
		{"schema_version", `"1.2.0"`},
		{"code", `"observation_horizon_lost"`},
		{"message", `"a second message"`},
		{"exit_code", `9`},
		{"retryable", `true`},
		{"operation_id", `"0198f4c8-b180-7299-9273-1234567890ab"`},
		{"session_id", `"0198f4c8-3e70-7a11-8a2b-1234567890ab"`},
		{"details", `{"observed_sequence": "18"}`},
	} {
		test.Run(row.member, func(test *testing.T) {
			if _, err := Decode(Version120, []byte(conformingDocument)); err != nil {
				test.Fatalf("the undoubled document was refused, so this row proves nothing: %v", err)
			}
			document := duplicateMember(test, conformingDocument, row.member, row.replacement)
			_, err := Decode(Version120, document)
			if !errors.Is(err, ErrInvalidStructuredError) {
				test.Fatalf("Decode(duplicate %s) error = %v, want ErrInvalidStructuredError", row.member, err)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("duplicate object member %q", row.member)) {
				test.Fatalf("Decode(duplicate %s) error = %v, want the refusal to name the repeated member",
					row.member, err)
			}
		})
	}

	// A document an order of magnitude larger than every row above. All of
	// those build a short fixture by hand, so a gate narrowed by payload SIZE -
	// `if len(data) < 4096 { ... }` - refuses every one of them and admits a
	// large one; that narrowing survived the whole suite when this leaf was
	// reviewed. Byte length is not a dimension of the Section 1.6 contract, so
	// this is not a rule being asserted; it is the one predicate that is
	// attacker-controlled on every path Decode is reachable from. DecodeBound
	// reads provider, bridge, RPC, session-adapter and terminal-backend
	// envelopes, and a peer chooses how long its own payload is.
	test.Run("a document far larger than the hand-built fixtures", func(test *testing.T) {
		large := largeConformingDocument(test)
		if len(large) <= 4096 {
			test.Fatalf("the padded document is %d bytes, which does not exceed the short fixtures this row exists to leave behind", len(large))
		}
		if _, err := Decode(Version120, []byte(large)); err != nil {
			test.Fatalf("the padded document without a duplicate was refused, so this row proves nothing: %v", err)
		}
		for _, member := range []string{"retryable", "code", "exit_code"} {
			document := duplicateMember(test, large, member, map[string]string{
				"retryable": "true", "code": `"observation_horizon_lost"`, "exit_code": "9",
			}[member])
			if len(document) <= 4096 {
				test.Fatalf("the duplicated document is %d bytes", len(document))
			}
			_, err := Decode(Version120, document)
			if !errors.Is(err, ErrInvalidStructuredError) {
				test.Fatalf("Decode(%d-byte document, duplicate %s) error = %v, want ErrInvalidStructuredError",
					len(document), member, err)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("duplicate object member %q", member)) {
				test.Fatalf("Decode(%d-byte document, duplicate %s) error = %v, want the refusal to name the repeated member",
					len(document), member, err)
			}
		}
	})

	test.Run("nested inside details", func(test *testing.T) {
		document := strings.Replace(conformingDocument,
			`{"observed_sequence": "17"}`,
			`{"observed_sequence": "17", "observed_sequence": "18"}`, 1)
		if document == conformingDocument {
			test.Fatal("the nested duplicate rewrite did not apply")
		}
		_, err := Decode(Version120, []byte(document))
		if !errors.Is(err, ErrInvalidStructuredError) {
			test.Fatalf("Decode(nested duplicate) error = %v, want ErrInvalidStructuredError", err)
		}
		if !strings.Contains(err.Error(), `duplicate object member "observed_sequence"`) {
			test.Fatalf("Decode(nested duplicate) error = %v, want the nested member named", err)
		}
	})
}

// TestADuplicateMemberCannotForgeARetryClaim is the finding stated as the
// machine answer it changed, not as a parse fact.
//
// Section 15.1 defines retryable as "true only when the identical request may
// safely be retried without new authority or confirmation", and this package
// spends a whole refusal table keeping a peer from claiming it. A document
// declaring "retryable": false and then repeating the member as true was read
// as retryable, so the claim was forged out of two members neither of which a
// conforming writer could have emitted, and the retryability refusal table was
// never consulted for the code that carried it.
//
// The measurement below is the pre-fix behaviour of the same bytes through
// encoding/json, run here rather than described, so the row cannot decay into
// prose about a defect nobody re-derives.
func TestADuplicateMemberCannotForgeARetryClaim(test *testing.T) {
	test.Parallel()

	document := duplicateMember(test, conformingDocument, "retryable", "true")

	var resolved struct {
		Retryable *bool `json:"retryable"`
	}
	if err := json.Unmarshal(document, &resolved); err != nil {
		test.Fatalf("unmarshal: %v", err)
	}
	if resolved.Retryable == nil || !*resolved.Retryable {
		test.Fatalf("encoding/json resolved the repeated retryable to %v, want the true occurrence; "+
			"if this changed, the forged-claim shape this gate refuses changed with it", resolved.Retryable)
	}

	failure, err := Decode(Version120, document)
	if err == nil {
		test.Fatalf("Decode admitted a forged retry claim: retryable = %v", failure.Retryable())
	}
	if !errors.Is(err, ErrInvalidStructuredError) {
		test.Fatalf("Decode(forged retry) error = %v, want ErrInvalidStructuredError", err)
	}
}

// TestADuplicateMemberCannotSplitTheDetailSet pins the third machine answer the
// repeat changed, and the one that is not a last-wins resolution at all.
//
// details decodes into a map, and encoding/json merges a repeated map-typed
// member rather than replacing it, so the reader saw the UNION of both
// occurrences - a detail set neither occurrence declared and no writer emitted.
func TestADuplicateMemberCannotSplitTheDetailSet(test *testing.T) {
	test.Parallel()

	document := duplicateMember(test, conformingDocument, "details", `{"observed_offset": "4"}`)

	var resolved struct {
		Details map[string]json.RawMessage `json:"details"`
	}
	if err := json.Unmarshal(document, &resolved); err != nil {
		test.Fatalf("unmarshal: %v", err)
	}
	if len(resolved.Details) != 2 {
		test.Fatalf("encoding/json resolved the repeated details to %d keys, want the 2-key union; "+
			"if this changed, the shape this gate refuses changed with it", len(resolved.Details))
	}

	failure, err := Decode(Version120, document)
	if err == nil {
		test.Fatalf("Decode admitted a union detail set: %v", failure.DetailKeys())
	}
	if !errors.Is(err, ErrInvalidStructuredError) {
		test.Fatalf("Decode(union details) error = %v, want ErrInvalidStructuredError", err)
	}
}

// TestABoundPeerCannotSendADuplicateKeyedEnvelope drives the same shape through
// the entry point a remote peer reaches, not only through the local one.
//
// DecodeBound is how a provider, task-board bridge, Mesh RPC, session-adapter,
// or terminal-backend payload is read, so the duplicate-member shape was
// remotely reachable on every one of those surfaces rather than only on a local
// pipe. Every bound contract is driven, so a gate added to one caller instead of
// to Decode would leave rows red here.
func TestABoundPeerCannotSendADuplicateKeyedEnvelope(test *testing.T) {
	test.Parallel()

	contracts := BoundContracts()
	if len(contracts) == 0 {
		test.Fatal("no bound contracts to drive")
	}
	for _, contract := range contracts {
		version, err := BindingFor(contract)
		if err != nil {
			test.Fatalf("BindingFor(%v): %v", contract, err)
		}
		name := fmt.Sprintf("%s major %d binds %s", contract.ID, contract.Major, version)
		test.Run(name, func(test *testing.T) {
			// A document of exactly the version this contract binds, so the row
			// fails on the duplicate rather than on a version mismatch.
			conforming := strings.Replace(conformingDocument, `"1.2.0"`, `"`+string(version)+`"`, 1)
			// observation_gap is not registered by every version; a code the
			// registry does not carry is admitted by Section 15.3 and keeps its
			// exit class, so the undoubled row below still has to pass.
			if _, err := DecodeBound(contract, []byte(conforming)); err != nil {
				test.Fatalf("the undoubled envelope was refused, so this row proves nothing: %v", err)
			}
			document := duplicateMember(test, conforming, "retryable", "true")
			if _, err := DecodeBound(contract, document); !errors.Is(err, ErrInvalidStructuredError) {
				test.Fatalf("DecodeBound(%v, duplicate retryable) error = %v, want ErrInvalidStructuredError",
					contract, err)
			}
		})
	}
}

// TestADuplicateIdentityMemberIsNotResolvedIntoAVersionFact pins the ORDER of
// the gate against the identity check, which is the second precedence decision
// this reader makes and the one the first one does not settle.
//
// A document whose schema_version appears twice has no version. A reader that
// settled the identity before the data model would resolve the repeat - to the
// last occurrence - and then answer with a version fact derived from one of two
// members it had no basis for choosing between. Both orders refuse the bytes,
// so this is not a bypass either way; what changes is whether a caller asking
// "is my ax too old for this output" is told a version it can act on or told
// that the bytes do not carry one.
//
// The rows are chosen so the identity-first order would produce a DIFFERENT and
// confidently wrong answer, not merely a differently worded one.
func TestADuplicateIdentityMemberIsNotResolvedIntoAVersionFact(test *testing.T) {
	test.Parallel()

	for _, row := range []struct {
		name     string
		document []byte
		notFact  error
	}{
		{
			// Identity-first resolves to the appended 2.0.0 and reports an
			// unsupported major for a document that also declares 1.2.0.
			name:     "repeated schema_version disagreeing on the major",
			document: duplicateMember(test, conformingDocument, "schema_version", `"2.0.0"`),
			notFact:  ErrUnsupportedMajor,
		},
		{
			// Identity-first resolves to the appended 1.0.0 and reports a bound
			// version mismatch for a document that also declares 1.2.0.
			name:     "repeated schema_version disagreeing on the minor",
			document: duplicateMember(test, conformingDocument, "schema_version", `"1.0.0"`),
			notFact:  ErrVersionMismatch,
		},
	} {
		test.Run(row.name, func(test *testing.T) {
			_, err := Decode(Version120, row.document)
			if !errors.Is(err, ErrInvalidStructuredError) {
				test.Fatalf("Decode(%s) error = %v, want ErrInvalidStructuredError", row.name, err)
			}
			if errors.Is(err, row.notFact) {
				test.Fatalf("Decode(%s) error = %v reports %v; the reader answered with a version "+
					"fact resolved from one of two occurrences of schema_version",
					row.name, err, row.notFact)
			}
			if !strings.Contains(err.Error(), `duplicate object member "schema_version"`) {
				test.Fatalf("Decode(%s) error = %v, want the repeated identity member named", row.name, err)
			}
		})
	}
}

// TestTheCanonicalGateDoesNotLaunderTheExitStatusToken narrows the gate rather
// than only proving it exists.
//
// requireCommonDataModel canonicalizes to decide whether the bytes are inside
// the common data model and then DISCARDS the canonical form. That is not an
// omission. RFC 8785 Section 3.2.2.3 serializes every number through the
// ECMAScript Number.prototype.toString algorithm, so the transform rewrites
// 1e1 to 10 and 9.0 to 9, and decodeExitStatus reads exit_code from its raw
// bytes precisely so that the exponent and the point are refused rather than
// normalized. A gate that adopted its own output would hand the reader a
// document those literals had been laundered out of, which is a WIDENING
// disguised as a validation step.
//
// Each row proves both halves: that the canonical transform really does rewrite
// the literal - so the row exercises the laundering it claims to prevent rather
// than a token nothing would have changed - and that Decode still refuses it.
func TestTheCanonicalGateDoesNotLaunderTheExitStatusToken(test *testing.T) {
	test.Parallel()

	for _, row := range []struct {
		literal   string
		canonical string
	}{
		{literal: "1e1", canonical: "10"},
		{literal: "9.0", canonical: "9"},
		{literal: "9.000", canonical: "9"},
		{literal: "0.9e1", canonical: "9"},
	} {
		test.Run(row.literal, func(test *testing.T) {
			document := strings.Replace(conformingDocument, `"exit_code": 9`, `"exit_code": `+row.literal, 1)
			if document == conformingDocument {
				test.Fatal("the exit status substitution did not apply")
			}

			canonical, err := canonicaljson.Canonicalize([]byte(document))
			if err != nil {
				test.Fatalf("the gate refused the document for an unrelated reason: %v", err)
			}
			want := `"exit_code":` + row.canonical
			if !strings.Contains(string(canonical), want) {
				test.Fatalf("the canonical form of %s is %s, want it to contain %q; "+
					"this row no longer exercises the laundering it exists to prevent",
					row.literal, canonical, want)
			}

			if _, err := Decode(Version120, []byte(document)); err == nil {
				test.Fatalf("Decode admitted exit_code %s; the gate adopted its own canonical bytes",
					row.literal)
			}
		})
	}
}

// TestTheCommonDataModelGateCoversMoreThanDuplicates states the rest of what
// requireCommonDataModel is the only refusal for, as driven rows rather than as
// a claim about the strict decoder.
//
// It matters because parseEnvelopeIdentity's trailing-content guard was removed
// when the gate moved in front of it: without these rows, a mutant that deletes
// the gate would only redden the duplicate cases and the trailing-content
// coverage would have silently left the package.
//
// STATED BOUND, measured against the delete-the-gate mutant rather than assumed:
// the trailing-content, lone-surrogate and invalid-UTF-8 rows all redden without
// the gate, so it owns them. The unescaped-control-character row does not,
// because validateMessage refuses that byte independently for the message
// member. That row is kept as a subsumed case, not counted as coverage this gate
// provides, and the day the message rule changes it is still driven here.
func TestTheCommonDataModelGateCoversMoreThanDuplicates(test *testing.T) {
	test.Parallel()

	for _, row := range []struct {
		name     string
		document []byte
	}{
		{"trailing content", []byte(conformingDocument + "\n{}")},
		{"trailing token", []byte(conformingDocument + " 7")},
		{
			"lone surrogate escape in a string",
			[]byte(strings.Replace(conformingDocument, `"the inventory batch skipped a sequence"`, `"\ud800"`, 1)),
		},
		{
			"invalid UTF-8 in a string",
			bytes.Replace([]byte(conformingDocument), []byte("sequence"), []byte{0xed, 0xa0, 0x80}, 1),
		},
		{
			"unescaped control character in a string",
			bytes.Replace([]byte(conformingDocument), []byte("sequence"), []byte{0x01}, 1),
		},
	} {
		test.Run(row.name, func(test *testing.T) {
			if _, err := Decode(Version120, row.document); !errors.Is(err, ErrInvalidStructuredError) {
				test.Fatalf("Decode(%s) error = %v, want ErrInvalidStructuredError", row.name, err)
			}
		})
	}
}

// TestTheCommonDataModelCitationsAreVerbatimInThePinnedDocument checks this
// package's own citations against internal/specdoc rather than trusting a line
// number from any report.
//
// It exists because the first draft of requireCommonDataModel's doc comment
// quoted Section 1.6 as fixing the wire format as "UTF-8 JSON restricted to the
// common logical data model". That sentence is nowhere in the pinned document.
// It was a paraphrase wearing quotation marks, which is the same defect class as
// inventing a constraint: a later reader would have gone looking for a rule the
// contract does not state. Both fragments below are now located by content, so
// the citation survives the document being re-paginated and reddens if the text
// changes.
func TestTheCommonDataModelCitationsAreVerbatimInThePinnedDocument(test *testing.T) {
	test.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		test.Fatalf("load pinned document: %v", err)
	}
	for _, fragment := range []string{
		"map keys MUST be UTF-8 strings and MUST be unique",
		"floating-point numbers, NaN, Infinity, non-string map keys, and duplicate",
	} {
		lines := document.QuoteLines(fragment)
		if len(lines) != 1 {
			test.Fatalf("fragment %q occurs on %v, want exactly one line", fragment, lines)
		}
		section, known := document.SectionID(lines[0])
		if !known || section != "1.6" {
			test.Fatalf("fragment %q is at line %d in section %q, and this package cites Section 1.6",
				fragment, lines[0], section)
		}
	}

	// The paraphrase the first draft quoted. Asserting its ABSENCE keeps the
	// correction from being undone by someone who remembers the wording.
	if document.Contains("restricted to the common logical data model") {
		test.Fatal("the pinned document now carries the paraphrase this test exists to keep out of citations")
	}
}
