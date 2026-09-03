package canonicaljson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// Canonicalize once documented that it rejects "non-I-JSON input". It does not:
// RFC 8785 Section 3.2.2.3 serializes numbers through the ECMAScript
// double-to-string algorithm, so 9007199254740993 canonicalizes to
// 9007199254740992 and 1.0 canonicalizes to 1. The doc comment now states that
// rounding as intended behaviour and hands the AX safe-integer refusal to the
// entry points that actually perform it.
//
// A doc comment cannot notice when it stops matching the code underneath it,
// and this whole defect class IS the doc and the behaviour disagreeing. So the
// tests in this file take the doc comment as the expectation rather than
// restating it: the `literal -> canonical` rows and the entry-point names are
// parsed out of the comment on Canonicalize in canonical.go and checked against
// what the package really does. Editing the prose without editing the code
// reddens here, and so does editing the code without editing the prose.
//
// WHAT IS PINNED, AND WHAT IS NOT.
//
// Machine-checked: every `literal -> canonical` row, the required-literal set
// below, the list of exported entry points named under DIVISION OF GUARANTEES
// (compared against the set derived from the production call graph), every
// `SPEC.md:<line>` citation (quoted verbatim against the digest-pinned document
// in internal/specdoc, required to be unique, to begin at the declared line, and
// where a fixture is named, to be the row that declares that fixture), the
// required-citation set below, the container limit together with the quantity
// it bounds and every `<n> arrays open <m> containers` row, every test name the
// comment mentions, and the absence of the retracted "non-I-JSON" claim.
//
// Not machine-checked: the rest of the prose, including every claim about
// RFC 8785 — its Section 3.2.2.3 algorithm and the contents of its Appendix B.
// RFC 8785 is not vendored in this repository, so no test here can compare a
// claim about it against its source. What is checked instead is one step
// downstream: TestCanonicalizeMatchesEveryFiniteRFC8785AppendixBNumberSample
// drives Canonicalize against a vendored transcription of the Appendix B
// samples, so a drift between this implementation and that transcription is
// caught while a drift between the transcription and the RFC is not. That bound
// is a real one. The
// specification quotations were once covered by the same sentence, and that was
// false — internal/specdoc embeds SPEC.md byte-exact under the pinned digest and
// internal/canonicaljson/constraint_excerpt_test.go had already been checking
// excerpts against it in this package. An unmeasured area was reported as
// unmeasurable, and the doc comment shipped a quotation attributed to the wrong
// fixture row underneath it. The pin is bounded to the enumerated rows, names
// and citations; it is not "the doc comment cannot lie".
//
// That bound has now cost twice, in the same comment, and the second time is
// what added the container rows. Revision 2 stated the depth limit as "opens
// more than 256 containers", which is a refusal the function does not perform:
// maxNestingDepth bounds containers open at once, so a shallow array of 400
// empty arrays opens 401 containers and canonicalizes normally. The pin of the
// day matched the digit 256 and drove only nested arrays, so it proved the
// number and never the quantity, and the false clause shipped green. The
// treatment is the treatment used for the rounding claim: move the assertion
// out of free prose into a row form the test builds, measures and drives.

// auditedRoundedLiterals are the three literals the audit finding named. The
// doc comment must keep documenting each of them, so shrinking the table back
// to a comfortable subset reddens instead of quietly narrowing the disclosure.
var auditedRoundedLiterals = []string{
	"9007199254740993",
	"18446744073709551615",
	"1.0",
}

// requiredSpecCitationLines are the pinned SPEC.md lines the DIVISION OF
// GUARANTEES note must keep citing, and requiredSpecCitationFixtures the
// normative fixtures it must keep naming. Without them a citation could be
// deleted rather than corrected, and the doc comment would shrink its own
// obligation until the gate below had nothing left to check — the failure mode
// the audited literal set above exists to prevent.
var (
	requiredSpecCitationLines    = []int{292, 295, 301, 302}
	requiredSpecCitationFixtures = []string{"NUM-UNSAFE-NUMBER", "NUM-UNSAFE-ROUND"}
)

// canonicalizeDocContract is the machine-readable part of the Canonicalize doc
// comment.
type canonicalizeDocContract struct {
	text        string
	rounding    map[string]string
	entryPoints []string
	citations   []specCitation
	containers  []containerRow
}

// containerRow is one `<n> <shape> arrays open <opened> containers, <atOnce> at
// once: <verdict>` row of the doc comment. The row states both quantities
// separately because the whole point of the block is that they are different
// quantities and only one of them is bounded.
type containerRow struct {
	arrays   int
	shape    string
	opened   int
	atOnce   int
	accepted bool
	text     string
}

func (row containerRow) String() string { return row.text }

// specCitation is one `SPEC.md:<line> [FIXTURE] "text"` row of the doc comment.
// fixture is empty when the row is not attributed to a normative fixture.
type specCitation struct {
	line    int
	fixture string
	text    string
}

func (citation specCitation) String() string {
	if citation.fixture == "" {
		return fmt.Sprintf("SPEC.md:%d %s", citation.line, strconv.Quote(citation.text))
	}
	return fmt.Sprintf("SPEC.md:%d %s %s", citation.line, citation.fixture, strconv.Quote(citation.text))
}

// TestCanonicalizeDocumentedRoundingIsWhatCanonicalizeActuallyDoes drives the
// real exported entry point with each literal the doc comment claims is rounded
// rather than refused, and requires the documented output byte for byte.
func TestCanonicalizeDocumentedRoundingIsWhatCanonicalizeActuallyDoes(t *testing.T) {
	contract := parseCanonicalizeDocContract(t)

	for _, literal := range auditedRoundedLiterals {
		if _, ok := contract.rounding[literal]; !ok {
			t.Fatalf("Canonicalize doc comment no longer documents the rounding of %s; the audited literal set is %v", literal, auditedRoundedLiterals)
		}
	}

	for _, literal := range sortedKeys(contract.rounding) {
		want := contract.rounding[literal]
		t.Run(literal, func(t *testing.T) {
			got, err := Canonicalize([]byte(literal))
			if err != nil {
				t.Fatalf("Canonicalize(%s) error = %v, want the documented canonical form %q", literal, err, want)
			}
			if string(got) != want {
				t.Fatalf("Canonicalize(%s) = %q, but the doc comment documents %q", literal, got, want)
			}
		})
	}
}

// TestCanonicalizeDocumentedNonGuaranteeIsRefusedAtEveryNamedEntryPoint is the
// negative half. Each documented literal is carried into an otherwise valid
// Session Record through an `extensions` member, which is the one place a
// number survives every closed-shape validator: validateExtensionValue accepts
// any json.Number without inspecting it. So the refusal proven here can only
// come from validateAXNumbers, and removing or widening that gate reddens this
// test rather than being caught by some other guard.
func TestCanonicalizeDocumentedNonGuaranteeIsRefusedAtEveryNamedEntryPoint(t *testing.T) {
	contract := parseCanonicalizeDocContract(t)

	baseline := mustJSON(t, sessionRecordWithExtensionNumber(json.Number("1")))
	if _, _, err := CalculateObjectIdentity(baseline); err != nil {
		t.Fatalf("CalculateObjectIdentity(fixture carrying a safe extension number) error = %v, want acceptance; the refusals below would prove nothing", err)
	}

	for _, literal := range sortedKeys(contract.rounding) {
		t.Run(literal, func(t *testing.T) {
			candidate := mustJSON(t, sessionRecordWithExtensionNumber(json.Number(literal)))
			want := "outside the AX safe-integer interval"
			if strings.ContainsAny(literal, ".eE") {
				want = "floating-point number"
			}

			digest, _, err := CalculateObjectIdentity(candidate)
			if err == nil {
				t.Fatalf("CalculateObjectIdentity(extensions number %s) = %q, nil; want a refusal containing %q", literal, digest, want)
			}
			if !errors.Is(err, ErrInvalidIdentity) || !strings.Contains(err.Error(), want) {
				t.Fatalf("CalculateObjectIdentity(extensions number %s) error = %v, want ErrInvalidIdentity containing %q", literal, err, want)
			}

			if _, _, err := VerifyObjectIdentity(candidate); err == nil || !errors.Is(err, ErrInvalidIdentity) || !strings.Contains(err.Error(), want) {
				t.Fatalf("VerifyObjectIdentity(extensions number %s) error = %v, want ErrInvalidIdentity containing %q", literal, err, want)
			}

			event := mustJSON(t, observationEventWithExtensionNumber(json.Number(literal)))
			if err := ValidateObservationEvent(event); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("ValidateObservationEvent(extensions number %s) error = %v, want a refusal containing %q", literal, err, want)
			}
			if err := ValidateObservationStream([][]byte{event}); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("ValidateObservationStream(extensions number %s) error = %v, want a refusal containing %q", literal, err, want)
			}
		})
	}
}

// TestCanonicalizeDocumentsExactlyTheEntryPointsThatEnforceTheAXNumberModel
// derives the division of guarantees from the production call graph instead of
// trusting the doc comment's list. An exported function from which
// validateAXNumbers is reachable promises the AX number model to its callers
// and must be named; one from which it is not reachable must not be named, and
// Canonicalize must stay in the second group.
//
// The bound is the graph's bound, stated at length on productionCallGraph in
// utf8_subsumption_test.go: it models identifier calls, methods declared in
// this package, and dispatch through a function value, and it models neither
// reflection, nor a function value handed to another package and invoked there,
// nor a func-typed struct field. Within that model the derived set is exact,
// because a computed callee is assumed to reach only functions whose identifier
// is used as a value somewhere in the package, and
// TestValidateAXNumbersHasNoCallSiteThisGraphCannotSee proves validateAXNumbers
// is never used that way.
func TestCanonicalizeDocumentsExactlyTheEntryPointsThatEnforceTheAXNumberModel(t *testing.T) {
	contract := parseCanonicalizeDocContract(t)
	derived := exportedFunctionsReaching(t, "validateAXNumbers")

	if len(derived) == 0 {
		t.Fatal("derived zero exported entry points reaching validateAXNumbers; the AX number model has no production enforcement")
	}
	if strings.Join(contract.entryPoints, ",") != strings.Join(derived, ",") {
		t.Fatalf("Canonicalize doc comment names entry points %v, but validateAXNumbers is reachable from exactly %v", contract.entryPoints, derived)
	}
	for _, name := range derived {
		if name == "Canonicalize" {
			t.Fatal("Canonicalize now reaches validateAXNumbers, so its documented non-guarantee is false; either revert that call or rewrite the DIVISION OF GUARANTEES note")
		}
	}
	if strings.Contains(contract.text, "non-I-JSON") {
		t.Fatal("Canonicalize doc comment claims to reject non-I-JSON input again; that claim was retracted because RFC 8785 Section 3.2.2.3 rounds unsafe numeric literals instead of refusing them")
	}
}

// TestValidateAXNumbersHasNoCallSiteThisGraphCannotSee holds the precondition
// that makes the derivation above exact rather than merely sound. The graph
// resolves a computed callee to every function whose identifier is used as a
// value in the package, so the moment validateAXNumbers is registered in a
// dispatch table, every computed call site starts reaching it and the derived
// entry-point set silently widens to something the doc comment cannot track.
// Registering it that way must redden here first.
func TestValidateAXNumbersHasNoCallSiteThisGraphCannotSee(t *testing.T) {
	_, files := packageProductionFiles(t)

	uses := 0
	for _, path := range files {
		parsed := parseProductionFile(t, path)
		ast.Inspect(parsed, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok || identifier.Name != "validateAXNumbers" {
				return true
			}
			uses++
			return true
		})
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "validateAXNumbers" {
				uses--
			}
			return true
		})
	}
	// One remaining use is the declaration's own name; anything more is an
	// identifier used as a value rather than called.
	if uses != 1 {
		t.Fatalf("validateAXNumbers appears %d times outside plain calls, want only its declaration; the derived entry-point set is no longer complete", uses)
	}
}

// TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification measures
// the half of the doc comment that the first revision of this file declared
// unmeasurable. It is not: internal/specdoc embeds SPEC.md byte-exact, refuses
// any document whose SHA-256 is not the pinned digest, and exposes the text by
// line and by table row. The first revision shipped "before identity
// calculation" attributed to NUM-UNSAFE-ROUND when that phrase is the
// NUM-UNSAFE-NUMBER row one line above, which is the audited defect class —
// an authoritative comment asserting something its source does not say — moved
// from behaviour to citation.
//
// Each citation must satisfy four conditions, and the third is what stops a
// citation being weakened rather than corrected: the quoted text must occur in
// the pinned document exactly once, so shortening it to a fragment that also
// appears elsewhere fails instead of passing more easily.
func TestCanonicalizeSpecificationCitationsQuoteThePinnedSpecification(t *testing.T) {
	contract := parseCanonicalizeDocContract(t)
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load the pinned specification document: %v", err)
	}

	seenLines := map[int]bool{}
	seenFixtures := map[string]bool{}
	sections := map[string][]specCitation{}

	for _, citation := range contract.citations {
		seenLines[citation.line] = true
		if citation.fixture != "" {
			seenFixtures[citation.fixture] = true
		}

		if strings.TrimSpace(citation.text) == "" {
			t.Errorf("citation %s quotes nothing; an empty quote is satisfied by any document", citation)
			continue
		}

		lines := document.QuoteLines(citation.text)
		switch {
		case len(lines) == 0:
			t.Errorf("citation %s quotes text that is absent from the pinned SPEC.md", citation)
			continue
		case len(lines) > 1:
			t.Errorf("citation %s quotes text that begins at %v, so it does not identify one place in the pinned SPEC.md; quote enough of the line to be unique rather than a fragment that matches everywhere", citation, lines)
			continue
		case lines[0] != citation.line:
			t.Errorf("citation %s quotes text that begins at SPEC.md:%d, not at the declared line", citation, lines[0])
			continue
		}

		row, isRow := document.TableRowAt(citation.line)
		switch {
		case citation.fixture == "" && isRow:
			t.Errorf("citation %s quotes the %q table row that declares %q without naming it; a normative fixture row must be cited by its fixture identifier", citation, row.Header, row.Identifier)
		case citation.fixture != "" && !isRow:
			t.Errorf("citation %s attributes the quote to fixture %s, but SPEC.md:%d is not a table body row and so declares no fixture", citation, citation.fixture, citation.line)
		case citation.fixture != "" && row.Identifier != citation.fixture:
			t.Errorf("citation %s attributes the quote to fixture %s, but SPEC.md:%d is the row that declares %q", citation, citation.fixture, citation.line, row.Identifier)
		}

		section, ok := document.SectionID(citation.line)
		if !ok {
			t.Errorf("citation %s cites a line outside every numbered SPEC.md clause", citation)
			continue
		}
		sections[section] = append(sections[section], citation)
	}

	for _, line := range requiredSpecCitationLines {
		if !seenLines[line] {
			t.Errorf("Canonicalize doc comment no longer cites SPEC.md:%d; the required citation set is %v", line, requiredSpecCitationLines)
		}
	}
	for _, fixture := range requiredSpecCitationFixtures {
		if !seenFixtures[fixture] {
			t.Errorf("Canonicalize doc comment no longer attributes a quotation to normative fixture %s; the required set is %v", fixture, requiredSpecCitationFixtures)
		}
	}

	// The comment names the clause its citations come from. That name is a
	// claim of the same kind as the quotations, so it is checked against the
	// clause the pinned document actually puts those lines in.
	//
	// The check is set equality, not presence. Requiring only that the true
	// clause is mentioned somewhere lets a second, false clause be added beside
	// it and stay green — a mutant of exactly that shape survived the first
	// draft of this gate. Every SPEC.md clause the comment names must be the
	// one its citations land in.
	if len(sections) != 1 {
		t.Fatalf("citations span SPEC.md clauses %v; the doc comment presents them as one clause", sortedSectionIDs(sections))
	}
	claimed := documentedSpecSections(contract.text)
	if strings.Join(claimed, ",") != strings.Join(sortedSectionIDs(sections), ",") {
		t.Errorf("doc comment names SPEC.md clauses %v, but every citation lands in %v", claimed, sortedSectionIDs(sections))
	}
}

// documentedSpecSection matches a clause reference in the doc comment. The
// optional prefix group exists to exclude "RFC 8785 Section 3.2.2.3": that
// names a clause of a different document, which is not vendored here and is
// covered by the stated RFC bound instead.
var documentedSpecSection = regexp.MustCompile(`(RFC 8785 )?Section ([0-9]+(?:\.[0-9]+)*)`)

// documentedSpecSections returns every pinned-SPEC clause the doc text names.
func documentedSpecSections(text string) []string {
	seen := map[string]bool{}
	var ids []string
	for _, match := range documentedSpecSection.FindAllStringSubmatch(text, -1) {
		if match[1] != "" || seen[match[2]] {
			continue
		}
		seen[match[2]] = true
		ids = append(ids, match[2])
	}
	sort.Strings(ids)
	return ids
}

// TestCanonicalizeDocumentedContainerLimitIsADepthNotACount ties the doc
// comment's container clause to maxNestingDepth and to the behaviour on both
// sides of it.
//
// The predecessor of this test checked the digit and never the noun. It
// string-matched "more than 256 containers" and then drove only nested arrays
// at maxNestingDepth and maxNestingDepth+1, which is consistent with a bound on
// containers opened and with a bound on containers open at once alike. The
// comment said "opens more than 256 containers", the enforced bound is a
// nesting depth, and the pin passed green over a false refusal claim — the
// audited defect class reintroduced by its own fix.
//
// So the number and the quantity it bounds are now pinned separately:
//
//   - the clause must carry the "open at once" qualifier, and no mention of the
//     limit anywhere in the comment may drop it;
//   - every documented row is built and driven through the real exported entry
//     point, with the row's own arithmetic re-measured from the bytes rather
//     than recomputed from the shape that produced them;
//   - the rows must keep covering all three facts: the limit is reachable, the
//     limit is enforced, and a document opening far more containers than the
//     limit is accepted while it stays shallow. The third is the one that makes
//     a count reading false, and shrinking the block back to the nested rows
//     alone reddens here rather than quietly restoring the ambiguity.
func TestCanonicalizeDocumentedContainerLimitIsADepthNotACount(t *testing.T) {
	contract := parseCanonicalizeDocContract(t)

	claim := fmt.Sprintf("more than %d containers open at once", maxNestingDepth)
	if !strings.Contains(contract.text, claim) {
		t.Fatalf("Canonicalize doc comment does not state %q, so its container limit is either unstated, no longer the enforced maxNestingDepth = %d, or stated as a bound on containers opened rather than on containers open at once", claim, maxNestingDepth)
	}
	// strings.Contains above is satisfied by a comment that states the clause
	// correctly once and incorrectly again somewhere else, which is the shape
	// that survives a presence check. Every occurrence in the free prose must
	// qualify the limit. The rows are exempt because they are not prose: each
	// one is built, measured and driven below, so a row that misstates the
	// quantity fails on behaviour rather than on wording.
	prose := contract.text
	for _, row := range contract.containers {
		prose = strings.Replace(prose, row.text, "", 1)
	}
	limit := fmt.Sprintf("%d containers", maxNestingDepth)
	for index := 0; ; {
		offset := strings.Index(prose[index:], limit)
		if offset < 0 {
			break
		}
		at := index + offset
		if !strings.HasPrefix(prose[at:], limit+" open at once") {
			t.Errorf("Canonicalize doc comment mentions %q without the \"open at once\" qualifier, in %q; the enforced bound is a nesting depth and a bare count claim is false", limit, docExcerptAround(prose, at))
		}
		index = at + len(limit)
	}

	var reachable, enforced, shallowBeyondLimit *containerRow
	for index := range contract.containers {
		row := contract.containers[index]
		t.Run(row.text, func(t *testing.T) {
			document := containerRowDocument(t, row)
			opened, atOnce := measureContainers(t, document)
			if opened != row.opened || atOnce != row.atOnce {
				t.Fatalf("row %q describes a document that opens %d containers with %d open at once, but the document it names opens %d with %d open at once", row, row.opened, row.atOnce, opened, atOnce)
			}

			_, err := Canonicalize(document)
			switch {
			case row.accepted && err != nil:
				t.Fatalf("row %q documents acceptance, but Canonicalize returned %v", row, err)
			case !row.accepted && err == nil:
				t.Fatalf("row %q documents a refusal, but Canonicalize accepted %d containers with %d open at once", row, opened, atOnce)
			}
		})

		switch {
		case row.accepted && row.atOnce == maxNestingDepth:
			reachable = &contract.containers[index]
		case !row.accepted && row.atOnce == maxNestingDepth+1:
			enforced = &contract.containers[index]
		}
		if row.accepted && row.opened > maxNestingDepth && row.atOnce <= maxNestingDepth {
			shallowBeyondLimit = &contract.containers[index]
		}
	}

	if reachable == nil {
		t.Errorf("no documented row is accepted with exactly %d containers open at once, so the documented limit is never shown to be reachable", maxNestingDepth)
	}
	if enforced == nil {
		t.Errorf("no documented row is refused with exactly %d containers open at once, so the documented limit is never shown to be enforced", maxNestingDepth+1)
	}
	if shallowBeyondLimit == nil {
		t.Errorf("no documented row is accepted while opening more than %d containers, so nothing distinguishes the documented depth bound from a bound on containers opened; that is the wording error this test exists to catch", maxNestingDepth)
	}
}

// containerRowDocument builds the document a container row names. The shapes
// are deliberately string-free so measureContainers can count brackets without
// a JSON scanner.
func containerRowDocument(t *testing.T, row containerRow) []byte {
	t.Helper()
	switch row.shape {
	case "nested":
		return nestedArrayDocument(row.arrays)
	case "sibling":
		return []byte("[" + strings.TrimSuffix(strings.Repeat("[],", row.arrays), ",") + "]")
	}
	t.Fatalf("row %q names container shape %q, which this test cannot build", row, row.shape)
	return nil
}

// measureContainers re-derives both quantities from the document bytes. Doing
// it here rather than from the row's own shape formula is the point: a row that
// misstates how many containers its document opens must fail, not be confirmed
// by arithmetic copied out of the claim under test.
func measureContainers(t *testing.T, document []byte) (opened, atOnce int) {
	t.Helper()
	if bytes.ContainsAny(document, `"`) {
		t.Fatalf("document %q contains a string, so counting brackets is not a valid measurement of its structure", document)
	}
	depth := 0
	for _, symbol := range document {
		switch symbol {
		case '[', '{':
			opened++
			depth++
			if depth > atOnce {
				atOnce = depth
			}
		case ']', '}':
			depth--
		}
	}
	if depth != 0 {
		t.Fatalf("document %q closes %d containers it never opened", document, -depth)
	}
	return opened, atOnce
}

// docExcerptAround returns a short window of the doc text around index, so a
// failure names the offending sentence instead of only the phrase.
func docExcerptAround(text string, index int) string {
	start := index - 60
	if start < 0 {
		start = 0
	}
	end := index + 60
	if end > len(text) {
		end = len(text)
	}
	return strings.Join(strings.Fields(text[start:end]), " ")
}

// TestCanonicalizeDocCommentNamesOnlyTestsThatExist pins the last unchecked
// name in the comment. It cites
// TestCanonicalizeMatchesEveryFiniteRFC8785AppendixBNumberSample as the reason
// the Appendix B rounding may not be refused here; a renamed or deleted test
// would leave that justification pointing at nothing while the comment kept
// reading as though it were still pinned.
func TestCanonicalizeDocCommentNamesOnlyTestsThatExist(t *testing.T) {
	contract := parseCanonicalizeDocContract(t)
	// Reuses the derivation in declared_bounds_proofs_test.go rather than
	// walking the test sources a second time; a second, weaker enumeration in
	// the same package is how this package once pinned 3 of 7 UTF-8 guards.
	declared := packageTestFunctionNames(t)

	named := documentedTestNames(contract.text)
	if len(named) == 0 {
		t.Fatal("Canonicalize doc comment names no test; its Appendix B justification rests on a pin the reader cannot find")
	}
	for _, name := range named {
		if _, ok := declared[name]; !ok {
			t.Errorf("Canonicalize doc comment names %s, which is not a test function in this package", name)
		}
	}
}

var documentedTestName = regexp.MustCompile(`\bTest[A-Za-z0-9_]*\b`)

// documentedTestNames returns every Go test identifier the doc text mentions.
func documentedTestNames(text string) []string {
	seen := map[string]bool{}
	var names []string
	for _, name := range documentedTestName.FindAllString(text, -1) {
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedSectionIDs(sections map[string][]specCitation) []string {
	ids := make([]string, 0, len(sections))
	for id := range sections {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// sessionRecordWithExtensionNumber returns a valid Session Record carrying
// number under a reverse-DNS extensions key.
func sessionRecordWithExtensionNumber(number json.Number) map[string]any {
	object := validSessionRecordV1Object()
	object["extensions"] = map[string]any{"works.relux.ax.audit": number}
	return object
}

// observationEventWithExtensionNumber does the same for the Section 18.1
// observation entry, which is not identity-addressed and has its own entry.
func observationEventWithExtensionNumber(number json.Number) map[string]any {
	object := validObservationEventObject()
	object["extensions"] = map[string]any{"works.relux.ax.audit": number}
	return object
}

// parseCanonicalizeDocContract extracts the machine-readable rows and names
// from the Canonicalize doc comment in the production sources.
func parseCanonicalizeDocContract(t *testing.T) canonicalizeDocContract {
	t.Helper()
	_, files := packageProductionFiles(t)

	for _, path := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != "Canonicalize" || function.Recv != nil {
				continue
			}
			if function.Doc == nil {
				t.Fatal("Canonicalize has no doc comment, so its contract is undocumented")
			}
			return newCanonicalizeDocContract(t, function.Doc.Text())
		}
	}
	t.Fatal("no declaration of Canonicalize found in the production sources")
	return canonicalizeDocContract{}
}

// newCanonicalizeDocContract reads the two indented block forms out of the doc
// text: `literal -> canonical` rows, and bare exported identifiers.
func newCanonicalizeDocContract(t *testing.T, text string) canonicalizeDocContract {
	t.Helper()
	contract := canonicalizeDocContract{text: text, rounding: map[string]string{}}

	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		row := strings.TrimSpace(line)
		if row == "" {
			continue
		}
		if match := canonicalizeContainerRow.FindStringSubmatch(row); match != nil {
			contract.containers = append(contract.containers, containerRow{
				arrays:   mustRowCount(t, row, match[1]),
				shape:    match[2],
				opened:   mustRowCount(t, row, match[3]),
				atOnce:   mustRowCount(t, row, match[4]),
				accepted: match[5] == "accepted",
				text:     row,
			})
			continue
		}
		if literal, canonical, ok := strings.Cut(row, " -> "); ok {
			literal, canonical = strings.TrimSpace(literal), strings.TrimSpace(canonical)
			if previous, duplicate := contract.rounding[literal]; duplicate {
				t.Fatalf("Canonicalize doc comment documents %s twice, as %q and %q", literal, previous, canonical)
			}
			contract.rounding[literal] = canonical
			continue
		}
		if strings.HasPrefix(row, "SPEC.md:") {
			match := canonicalizeSpecCitationRow.FindStringSubmatch(row)
			if match == nil {
				t.Fatalf("indented line %q in the Canonicalize doc comment looks like a SPEC.md citation but does not parse as `SPEC.md:<line> [FIXTURE] \"quoted text\"`, so it would be documented but unchecked", row)
			}
			line, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatalf("citation %q declares an unreadable SPEC.md line: %v", row, err)
			}
			contract.citations = append(contract.citations, specCitation{line: line, fixture: match[2], text: match[3]})
			continue
		}
		if isExportedIdentifier(row) {
			contract.entryPoints = append(contract.entryPoints, row)
			continue
		}
		t.Fatalf("indented line %q in the Canonicalize doc comment is none of a `literal -> canonical` row, an exported entry-point name, or a `SPEC.md:<line>` citation, so it is documented but unchecked", row)
	}

	if len(contract.rounding) == 0 {
		t.Fatal("Canonicalize doc comment documents no rounding rows, so nothing about its number behaviour is pinned")
	}
	if len(contract.citations) == 0 {
		t.Fatal("Canonicalize doc comment cites no SPEC.md line, so its statement of the division of guarantees rests on nothing")
	}
	if len(contract.containers) == 0 {
		t.Fatal("Canonicalize doc comment documents no container rows, so its container refusal is unpinned prose again")
	}
	sort.Strings(contract.entryPoints)
	return contract
}

// canonicalizeSpecCitationRow matches `SPEC.md:<line> [FIXTURE] "quoted text"`.
// The fixture group is deliberately anchored to the pinned document's own
// fixture-identifier spelling, so a citation cannot name an arbitrary word and
// have it accepted as an attribution.
var canonicalizeSpecCitationRow = regexp.MustCompile(`^SPEC\.md:([1-9][0-9]*) (?:([A-Z][A-Z0-9]*(?:-[A-Z0-9]+)*) )?"(.*)"$`)

// canonicalizeContainerRow matches
// `<n> <sibling|nested> arrays open <opened> containers, <atOnce> at once: <verdict>`.
var canonicalizeContainerRow = regexp.MustCompile(`^([1-9][0-9]*) (sibling|nested) arrays open ([1-9][0-9]*) containers, ([1-9][0-9]*) at once: (accepted|refused)$`)

func mustRowCount(t *testing.T, row, field string) int {
	t.Helper()
	value, err := strconv.Atoi(field)
	if err != nil {
		t.Fatalf("row %q declares an unreadable count %q: %v", row, field, err)
	}
	return value
}

func isExportedIdentifier(text string) bool {
	if text == "" || text[0] < 'A' || text[0] > 'Z' {
		return false
	}
	for _, symbol := range text {
		isLetter := (symbol >= 'a' && symbol <= 'z') || (symbol >= 'A' && symbol <= 'Z')
		isDigit := symbol >= '0' && symbol <= '9'
		if !isLetter && !isDigit && symbol != '_' {
			return false
		}
	}
	return true
}

// exportedFunctionsReaching returns the sorted exported production entry points
// from which target is reachable.
//
// It reuses productionCallGraph rather than deriving a second graph. A weaker
// derivation in the same package would reintroduce a blind spot this package
// already measured and closed: a graph that recorded only `f(x)` edges once
// pinned 3 of 7 UTF-8 guards while looking complete, because this package
// dispatches through a function-value table. That graph now models function
// values and methods, and its residual constructions are stated on it.
func exportedFunctionsReaching(t *testing.T, target string) []string {
	t.Helper()
	declarations, edges := productionCallGraph(t)

	if _, declared := declarations[target]; !declared {
		t.Fatalf("%s is not a production declaration, so nothing can be derived about what reaches it", target)
	}

	var exported []string
	for _, entryPoint := range exportedProductionEntryPoints(declarations) {
		if reachableProductionFunctions(edges, entryPoint.key)[target] {
			exported = append(exported, entryPoint.key)
		}
	}
	sort.Strings(exported)
	return exported
}

func sortedKeys(table map[string]string) []string {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
