// Exercised-site audit for the terminal backend refusal arms.
//
// The failure this file exists to prevent: a declared, witnessed arm going
// dead by construction while the whole suite stays green. Witnesses assert
// the (code, detail) pair at a public entry; several arms share one pair
// (62 of 184 witnessed arms share their clause with a sibling), so a
// widened earlier guard swallows a later arm's inputs and the later
// witness still passes through the sibling. That plant (P8:
// CheckEntrypoint's first session parse widened to also compare, killing
// the second parse and the comparison arms) left the derived set
// byte-identical, the bijection at 210/210, and every witness green.
//
// The audit closes it the way provider and provhost already do: every
// production refusal construction records its file:line at runtime
// (through refuse/mismatchf/integrityFailure, armed by TestMain below),
// and the post-run audit requires every derived non-bound line to have
// fired at least once, and every fired line to be derived. A shadowed arm
// never fires, so it reddens here as a derived site without an exercised
// path. Bound arms (26 rows no input can reach) are exempt by their
// declared rows, mapped to lines by TestDerivedSiteLinesAreExactlyRowed,
// never by hand.
//
// Recording is off (nil hook) in production: construction is then a pure
// allocation with zero behavior change.
package terminalbackend

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// refusalSiteRecorder is the shared core's runtime direction: the
// production file:line behind every refusal construction during the test
// run, plus the wire codes those refusals carried. The hook is armed in
// TestMain, so an aliased constructor still executes the funnel and
// records its real production site: the runtime direction of the
// alias-bypass union.
var refusalSiteRecorder = invcore.NewSiteRecorder()

func TestMain(main *testing.M) {
	recordRefusal = func(code, detail string) {
		// Skip counts the frames between Record and the production
		// site: the hook closure plus the funnel frame, matching the
		// provider/provhost swapped-constructor convention exactly
		// (pinned by TestRefusalSiteAttributionIsExact, not by reading).
		refusalSiteRecorder.Record(code, 2)
	}
	code := main.Run()
	if code == 0 && fullPackageTestRun() {
		if failures := auditRefusalInventory(); len(failures) != 0 {
			for _, failure := range failures {
				fmt.Fprintln(os.Stderr, failure)
			}
			code = 1
		}
	}
	os.Exit(code)
}

func fullPackageTestRun() bool {
	selected := flag.Lookup("test.run")
	return selected == nil || selected.Value.String() == ""
}

// refusalAuditDomain is the captured in-band derivation the post-run
// audit compares the exercised set against. It is captured by
// TestDerivedSiteLinesAreExactlyRowed (which runs with a *testing.T and
// can use the rich derivation) because TestMain has no *testing.T to
// derive with. The audit fails closed when the domain is missing: a
// scoped (-run) run never reaches the audit by fullPackageTestRun.
type refusalAuditDomain struct {
	derived   map[string]struct{}
	bound     map[string]struct{}
	wantCodes []string
}

var auditDomainMu sync.Mutex
var capturedAuditDomain *refusalAuditDomain

// TestDerivedSiteLinesAreExactlyRowed is the static half of site
// attribution: every derived arm line resolves to exactly one declared
// row and every declared row to exactly one derived line, and it captures
// the domain the post-run exercised-site audit compares against. Bound
// rows resolve to lines too: the exemption is row-derived, never listed.
func TestDerivedSiteLinesAreExactlyRowed(t *testing.T) {
	t.Parallel()

	arms := deriveRefusalArms(t)
	rows := declaredRows()
	rowByKey := make(map[string]declaredRefusalArm, len(rows))
	for _, row := range rows {
		key := describeRow(row)
		if _, duplicate := rowByKey[key]; duplicate {
			t.Fatalf("declared row %s occurs twice; the mapping cannot attribute it", key)
		}
		rowByKey[key] = row
	}
	lineToRow := make(map[string]string)
	rowToLine := make(map[string]string)
	for _, arm := range arms {
		// The arm key space is occurrence-aware but line-free, and the
		// row key space is identical, so describeArm maps lines onto
		// rows through the shared identity without ever listing a line
		// by hand. (Line is not part of the key: describeArm never
		// reads it.)
		site := arm.file + ":" + strconv.Itoa(arm.line)
		key := describeArm(arm)
		row, known := rowByKey[key]
		if !known {
			t.Errorf("derived arm %s at %s resolves to no declared row", key, site)
			continue
		}
		rowKey := describeRow(row)
		if first, duplicate := lineToRow[site]; duplicate {
			t.Errorf("derived line %s attributes rows %s and %s", site, first, rowKey)
			continue
		}
		lineToRow[site] = rowKey
		if first, duplicate := rowToLine[rowKey]; duplicate {
			t.Errorf("declared row %s resolves to lines %s and %s", rowKey, first, site)
			continue
		}
		rowToLine[rowKey] = site
	}
	for _, row := range rows {
		if _, ok := rowToLine[describeRow(row)]; !ok {
			t.Errorf("declared row %s resolves to no derived line", describeRow(row))
		}
	}
	derived := make(map[string]struct{}, len(lineToRow))
	for site := range lineToRow {
		derived[site] = struct{}{}
	}
	bound := make(map[string]struct{})
	for site, rowKey := range lineToRow {
		if rowByKey[rowKey].bound != "" {
			bound[site] = struct{}{}
		}
	}
	symbols := productionCodeSymbols(t)
	var wantCodes []string
	for symbol, wire := range symbols {
		if armlessCodes[symbol] {
			continue
		}
		wantCodes = append(wantCodes, wire)
	}
	sort.Strings(wantCodes)
	auditDomainMu.Lock()
	capturedAuditDomain = &refusalAuditDomain{derived: derived, bound: bound, wantCodes: wantCodes}
	auditDomainMu.Unlock()
	t.Logf("refusal-site audit domain: %d derived lines, %d bound-exempt, %d wire codes",
		len(derived), len(bound), len(wantCodes))
}

// auditRefusalInventory derives nothing: it compares the exercised
// file:line set against the captured domain in both directions, plus the
// closed wire-code set. A shadowed-dead arm never fires, so it fails here
// as a derived site without an exercised path even though its pair still
// refuses through its sibling and every witness stays green.
func auditRefusalInventory() []string {
	auditDomainMu.Lock()
	domain := capturedAuditDomain
	auditDomainMu.Unlock()
	if domain == nil || len(domain.derived) == 0 {
		return []string{"refusal-site audit: no captured audit domain; the mapping test did not run (full suite required)"}
	}
	var failures []string
	exercised := refusalSiteRecorder.Sites()
	var outside []string
	for site := range exercised {
		if _, ok := domain.derived[site]; !ok {
			outside = append(outside, site)
		}
	}
	sort.Strings(outside)
	if len(outside) != 0 {
		failures = append(failures, "refusal-site audit: exercised refusal sites outside the derived inventory; the derivation is short: "+strings.Join(outside, ", "))
	}
	var missing []string
	for site := range domain.derived {
		if _, ok := domain.bound[site]; ok {
			continue
		}
		if _, ok := exercised[site]; !ok {
			missing = append(missing, site)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		failures = append(failures, "refusal-site audit: derived refusal sites without an exercised path (shadowed dead or unwitnessed): "+strings.Join(missing, ", "))
	}
	codes := refusalSiteRecorder.Codes()
	if fmt.Sprintf("%v", codes) != fmt.Sprintf("%v", domain.wantCodes) {
		failures = append(failures, fmt.Sprintf("refusal-site audit: observed refusal codes = %v, want closed set %v", codes, domain.wantCodes))
	}
	return failures
}

// TestRefusalSiteAttributionIsExact pins the recorder skip empirically:
// a refusal built on the marked line must attribute to this file at that
// line through each funnel.
//
// SEQUENTIAL BY DESIGN (no t.Parallel): it swaps the package hook, which
// races with every parallel witness recording into the global recorder.
// Sequential tests run exclusively before the parallel phase resumes, so
// the swap is exclusive. Do not parallelize this test.
func TestRefusalSiteAttributionIsExact(t *testing.T) {
	probe := invcore.NewSiteRecorder()
	previous := recordRefusal
	recordRefusal = func(code, detail string) {
		probe.Record(code, 2)
	}
	defer func() { recordRefusal = previous }()

	_, _, refuseLine, _ := runtime.Caller(0)
	refused := refuse(&Error{Code: CodeNotFound, Detail: "attribution probe"}) // want refuseLine+1
	if refused.Code != CodeNotFound || refused.Detail != "attribution probe" {
		t.Fatalf("refuse() = %+v, want the wrapped literal unchanged: recording never alters the refusal", refused)
	}
	_, _, mismatchLine, _ := runtime.Caller(0)
	mismatched := mismatchf("attribution probe") // want mismatchLine+1
	if mismatched.Code != CodeMismatch || mismatched.Detail != "attribution probe" {
		t.Fatalf("mismatchf() = %+v, want CodeMismatch at the static detail", mismatched)
	}
	_, _, integrityLine, _ := runtime.Caller(0)
	integrity := integrityFailure("attribution probe") // want integrityLine+1
	if integrity.Code != CodeIntegrityFailure || integrity.Detail != "attribution probe" {
		t.Fatalf("integrityFailure() = %+v, want CodeIntegrityFailure at the static detail", integrity)
	}
	sites := probe.Sites()
	for _, want := range []string{
		"refusal_site_audit_test.go:" + strconv.Itoa(refuseLine+1),
		"refusal_site_audit_test.go:" + strconv.Itoa(mismatchLine+1),
		"refusal_site_audit_test.go:" + strconv.Itoa(integrityLine+1),
	} {
		if _, ok := sites[want]; !ok {
			t.Errorf("attributed sites = %v, want %s recorded: the skip is wrong and the audit is blind", sites, want)
		}
	}
	if len(sites) != 3 {
		t.Errorf("attributed sites = %v, want exactly the three probe sites", sites)
	}
}

// TestRefuseFunnelRejectsAliases is the committed tb-level proof of the
// alias-bypass union for the new funnel: a direct refuse call is
// admitted, a var-bound refuse, a var-bound mismatchf (the P1 shape), and
// an import-aliased errors.New (the P2 shape) all fail. Dropping refuse
// from the watch list (N-tb-refuse) kills this test while the downstream
// behav mask stays green.
func TestRefuseFunnelRejectsAliases(t *testing.T) {
	t.Parallel()

	admitted := `package terminalbackend
func direct() *Error { return refuse(&Error{Code: CodeNotFound, Detail: "x"}) }
`
	syntax, fileSet, failure := invcore.ParseSource("direct.go", []byte(admitted))
	if failure != "" {
		t.Fatalf("parse admitted source: %s", failure)
	}
	if failures := invcore.AuditConstructorReferences(syntax, fileSet, "direct.go", refusalConstructorSpec()); len(failures) != 0 {
		t.Errorf("direct refuse call rejected: %v", failures)
	}
	rejected := map[string]string{
		"var-bound refuse": "package terminalbackend\nvar aliasedRefuse = refuse\n" +
			"func indirect() *Error { return aliasedRefuse(&Error{Code: CodeNotFound, Detail: \"x\"}) }\n",
		"var-bound mismatchf (P1)": "package terminalbackend\nvar aliasedMismatch = mismatchf\n" +
			"func indirect() *Error { return aliasedMismatch(\"x\") }\n",
		"import-aliased errors.New (P2)": "package terminalbackend\nimport errs \"errors\"\n" +
			"var mintPlain = errs.New\nfunc indirect() error { return mintPlain(\"x\") }\n",
	}
	for name, source := range rejected {
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("parse %s plant: %s", name, failure)
		}
		failures := invcore.AuditConstructorReferences(syntax, fileSet, "plant.go", refusalConstructorSpec())
		if len(failures) == 0 {
			t.Errorf("%s admitted: the alias bypasses attribution", name)
		}
	}
}
