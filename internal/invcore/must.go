// Must wrappers and the runtime site recorder for Package invcore.
//
// The Must wrappers fail the suite on any returned failure: production
// inventories call these, while negative tests and control plants call
// the pure funcs in invcore.go directly and assert on the returned
// failures. A gate that can only fatal cannot be planted against.

package invcore

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

// MustScanProduction scans dir and fails the suite on any fail-closed
// condition: zero files, unreadable files, unparseable files.
func MustScanProduction(t *testing.T, dir string) ([]ProductionFile, *token.FileSet) {
	t.Helper()
	files, fileSet, failures := ScanProduction(dir)
	for _, failure := range failures {
		t.Error(failure)
	}
	if len(failures) > 0 {
		t.FailNow()
	}
	return files, fileSet
}

// MustCheckBothDirections fails the suite when the derived set and the
// declared row set differ in either direction: unregistered derived sites
// or orphaned rows. A zero derived set fails even when rows are empty.
func MustCheckBothDirections(t *testing.T, what string, derived, rows map[string]struct{}) {
	t.Helper()
	unregistered, orphaned, failures := DiffSets(derived, rows)
	for _, failure := range failures {
		t.Errorf("%s: %s", what, failure)
	}
	for _, key := range unregistered {
		t.Errorf("%s: unregistered site with no declaring row, so nothing witnesses it: %s", what, key)
	}
	for _, key := range orphaned {
		t.Errorf("%s: orphan row with no derived site; a row without a site is a witness without a guard: %s", what, key)
	}
}

// MustAuditConstructorReferences fails the suite on any constructor
// reference outside direct-call position, including import-aliased and
// var-bound shapes, and on any shape the audit cannot classify.
func MustAuditConstructorReferences(t *testing.T, syntax *ast.File, fileSet *token.FileSet, display string, spec ConstructorSpec) {
	t.Helper()
	for _, failure := range AuditConstructorReferences(syntax, fileSet, display, spec) {
		t.Error(failure)
	}
}

// SiteRecorder is the runtime direction of the alias-bypass union
// (provider/provhost shape): instrumented constructor vars record the
// production file:line behind every exercised refusal plus the stable
// codes those refusals carried. The audit requires the exercised site set
// to equal the derived site set in both directions, and the observed code
// set to equal the closed code set exactly, so a constructor call the
// tests never exercise fails forward, and a truncated derivation fails in
// reverse even though the forward check passes vacuously.
type SiteRecorder struct {
	mu    sync.Mutex
	sites map[string]struct{}
	codes map[string]struct{}
}

// NewSiteRecorder returns an empty recorder.
func NewSiteRecorder() *SiteRecorder {
	return &SiteRecorder{sites: map[string]struct{}{}, codes: map[string]struct{}{}}
}

// Record notes one exercised refusal carrying code, attributing it to
// the production file:line at runtime.Caller(skip+1): skip counts every
// frame between Record and the site, including Record's own caller. A
// record helper called from the swapped constructor var passes skip=2
// (helper frame plus wrapper frame), matching the historical sync.Map
// recorders' runtime.Caller(2) exactly.
func (recorder *SiteRecorder) Record(code string, skip int) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.codes[code] = struct{}{}
	if _, file, line, ok := runtime.Caller(skip + 1); ok {
		recorder.sites[filepath.Base(file)+":"+fmt.Sprint(line)] = struct{}{}
	}
}

// Snapshot returns deep copies of the exercised site and code sets for
// tests that record bystander sites and must remove them before
// returning, keeping the reverse-direction audit exact.
func (recorder *SiteRecorder) Snapshot() (sites map[string]struct{}, codes map[string]struct{}) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	sites = make(map[string]struct{}, len(recorder.sites))
	for site := range recorder.sites {
		sites[site] = struct{}{}
	}
	codes = make(map[string]struct{}, len(recorder.codes))
	for code := range recorder.codes {
		codes[code] = struct{}{}
	}
	return sites, codes
}

// Restore resets the recorder to a prior snapshot, dropping every site
// and code recorded since.
func (recorder *SiteRecorder) Restore(sites map[string]struct{}, codes map[string]struct{}) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.sites = make(map[string]struct{}, len(sites))
	for site := range sites {
		recorder.sites[site] = struct{}{}
	}
	recorder.codes = make(map[string]struct{}, len(codes))
	for code := range codes {
		recorder.codes[code] = struct{}{}
	}
}

// Sites returns the exercised production file:line set.
func (recorder *SiteRecorder) Sites() map[string]struct{} {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	duplicated := make(map[string]struct{}, len(recorder.sites))
	for site := range recorder.sites {
		duplicated[site] = struct{}{}
	}
	return duplicated
}

// Codes returns the observed stable code set.
func (recorder *SiteRecorder) Codes() []string {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	var codes []string
	for code := range recorder.codes {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// AuditSites compares the exercised set against the derived site set in
// both directions and the observed codes against the closed set, returning
// one failure per violated direction. Empty scanned or derived input fails
// closed: a blind scan is not a measurement.
func (recorder *SiteRecorder) AuditSites(what string, scanned int, derived map[string]struct{}, wantCodes []string) []string {
	var failures []string
	if scanned == 0 {
		failures = append(failures, what+": scanned no production sources; the check is blind")
	}
	if len(derived) == 0 {
		failures = append(failures, what+": derived no refusal sites; the scan is blind")
	}
	exercised := recorder.Sites()
	var outside []string
	for site := range exercised {
		if _, ok := derived[site]; !ok {
			outside = append(outside, site)
		}
	}
	sort.Strings(outside)
	if len(outside) != 0 {
		failures = append(failures, what+": exercised refusal sites outside the derived inventory; the derivation is short: "+strings.Join(outside, ", "))
	}
	var missing []string
	for site := range derived {
		if _, ok := exercised[site]; !ok {
			missing = append(missing, site)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		failures = append(failures, what+": derived refusal sites without an exercised negative path: "+strings.Join(missing, ", "))
	}
	codes := recorder.Codes()
	if fmt.Sprintf("%v", codes) != fmt.Sprintf("%v", wantCodes) {
		failures = append(failures, fmt.Sprintf("%s: observed refusal codes = %v, want closed set %v", what, codes, wantCodes))
	}
	return failures
}
