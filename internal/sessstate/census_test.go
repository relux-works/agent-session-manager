package sessstate

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// This file is the sessstate census. Three inventories derive their
// denominator from production source through internal/invcore instead
// of listing it:
//
//  1. Refusals: every direct refuse(SENTINEL, ...) call site must have
//     a boundary-driven negative path, and every exercised refusal
//     must derive. Calls sit on a single line with a bare sentinel
//     identifier, or the derivation fails closed.
//  2. States: the SessionState spellings derive structurally from the
//     State-typed const block (plus State() conversions), each
//     spelling lives exactly once as a production literal, and the
//     census drives Reduce to every roster value through the
//     production entry.
//  3. Events: the handled event_type literals derive from every
//     direct `<value>.Type` switches and ==/!= comparisons, each row
//     names its class and driver, and the rows equal the pinned v1
//     registry. TestEventTypeUsesAreOwned separately derives EVERY
//     Type selector and refuses alias/escape/unclassified contexts.
//     Its explicit direct-field-access bound is documented at the ledger.
//
// Control plants below prove each half fails closed, including an
// import alias and a var binding in both the passing and the failing
// direction.
var sessstateRefusalRecorder = invcore.NewSiteRecorder()

var sessstateBoundarySites = map[string]struct{}{}

var sessstateObservedSentinels = map[string]struct{}{}

var sessstateBoundaryMarkers = []string{
	"sessstate.Reduce",
	"sessstate.DecodeRecord",
	"sessstate.DecodeEvent",
	").Project",
}

var sessstateSentinels = map[string]bool{
	"ErrInvalidRecord": true, "ErrInvalidEvent": true,
	"ErrInvalidTransition": true, "ErrIntegrity": true,
	"ErrDerivation": true, "ErrUnknownSession": true,
	"ErrStaleLease": true, "ErrDivergentLease": true,
}

func TestMain(main *testing.M) {
	original := refuse
	refuse = func(sentinel error, format string, arguments ...any) error {
		err := original(sentinel, format, arguments...)
		name := sentinelName(sentinel)
		sessstateRefusalRecorder.Record(name, 1)
		if beneathSessstateBoundary() {
			if site := callerSessstateSite(); site != "" {
				sessstateBoundarySites[site] = struct{}{}
			}
			sessstateObservedSentinels[name] = struct{}{}
		}
		return err
	}
	code := main.Run()
	if fullSessstatePackageTestRun() {
		// Static audits run on every full run, green or red, so a
		// plant reddens the gate even when an unrelated test is
		// already red. The refusal runtime audit compares exercised
		// sites against derived sites and stays gated on green.
		files, fileSet, scanFailures := invcore.ScanProduction(".")
		var failures []string
		failures = append(failures, scanFailures...)
		if len(scanFailures) == 0 {
			failures = append(failures, auditSessstateAliases(files, fileSet)...)
			failures = append(failures, auditSessstateUnrouted(files, fileSet)...)
		}
		if code == 0 {
			failures = append(failures, auditSessstateRefusals()...)
		}
		if len(failures) != 0 {
			for _, failure := range failures {
				fmt.Fprintln(os.Stderr, failure)
			}
			code = 1
		}
	}
	os.Exit(code)
}

func sentinelName(sentinel error) string {
	candidates := []struct {
		name  string
		value error
	}{
		{"ErrInvalidRecord", ErrInvalidRecord},
		{"ErrInvalidEvent", ErrInvalidEvent},
		{"ErrInvalidTransition", ErrInvalidTransition},
		{"ErrIntegrity", ErrIntegrity},
		{"ErrDerivation", ErrDerivation},
		{"ErrUnknownSession", ErrUnknownSession},
		{"ErrStaleLease", ErrStaleLease},
		{"ErrDivergentLease", ErrDivergentLease},
	}
	for _, candidate := range candidates {
		if sentinel == candidate.value {
			return candidate.name
		}
	}
	return "unknown-sentinel"
}

func fullSessstatePackageTestRun() bool {
	selected := flag.Lookup("test.run")
	return selected == nil || selected.Value.String() == ""
}

// callerSessstateSite reports the production file:line of the refuse
// call site.
func callerSessstateSite() string {
	frames := make([]uintptr, 4)
	if count := runtime.Callers(3, frames); count == 0 {
		return ""
	}
	located := runtime.CallersFrames(frames)
	frame, _ := located.Next()
	if frame.Function == "" {
		return ""
	}
	return filepath.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
}

func beneathSessstateBoundary() bool {
	frames := make([]uintptr, 64)
	for {
		count := runtime.Callers(2, frames)
		located := runtime.CallersFrames(frames[:count])
		for range frames[:count] {
			frame, _ := located.Next()
			for _, marker := range sessstateBoundaryMarkers {
				if strings.Contains(frame.Function, marker) {
					return true
				}
			}
		}
		if count < len(frames) {
			return false
		}
		frames = make([]uintptr, 2*len(frames))
	}
}

// auditSessstateRefusals derives the expected inventory from production
// AST and requires the exercised sets to match it in both directions,
// the alias and unrouted audits to be clean, and the observed sentinel
// set to equal the derived sentinel roster exactly.
func auditSessstateRefusals() []string {
	files, fileSet, scanFailures := invcore.ScanProduction(".")
	if len(scanFailures) != 0 {
		return scanFailures
	}
	_, derived, failures := deriveSessstateRefusalSites(files, fileSet)
	rows := map[string]struct{}{}
	for site := range sessstateRefusalRows() {
		rows[site] = struct{}{}
	}
	unregistered, orphaned, diffFailures := invcore.DiffSets(derived, rows)
	failures = append(failures, diffFailures...)
	for _, site := range unregistered {
		failures = append(failures, "sessstate refusal site without a boundary-driven row: "+site)
	}
	for _, site := range orphaned {
		failures = append(failures, "sessstate refusal row without a derived site: "+site)
	}
	unexercised, underived, _ := invcore.DiffSets(derived, sessstateBoundarySites)
	_ = unexercised
	for _, site := range underived {
		failures = append(failures, "sessstate exercised refusal outside the derived inventory: "+site)
	}
	for site := range derived {
		if _, ok := sessstateBoundarySites[site]; !ok {
			failures = append(failures, "sessstate refusal site without an exercised negative path: "+site)
		}
	}
	derivedSentinels := map[string]struct{}{}
	for _, sentinel := range derivedRefusalSentinels(files, fileSet) {
		derivedSentinels[sentinel] = struct{}{}
	}
	for sentinel := range derivedSentinels {
		if _, ok := sessstateObservedSentinels[sentinel]; !ok {
			failures = append(failures, "sessstate sentinel never observed beneath a boundary: "+sentinel)
		}
	}
	for sentinel := range sessstateObservedSentinels {
		if _, ok := derivedSentinels[sentinel]; !ok {
			failures = append(failures, "sessstate observed sentinel outside the derived roster: "+sentinel)
		}
	}
	return failures
}

// refusalSite is one derived refuse call: production file:line plus
// the sentinel it carries.
type refusalSite struct {
	site     string
	sentinel string
}

// deriveSessstateRefusalSites derives every direct refuse(SENTINEL,
// ...) call in the production files as file:line with its sentinel.
// Calls must sit on a single line with a bare sentinel identifier, or
// the derivation fails closed; two calls sharing one line fail too.
func deriveSessstateRefusalSites(files []invcore.ProductionFile, fileSet *token.FileSet) ([]refusalSite, map[string]struct{}, []string) {
	var sites []refusalSite
	derived := map[string]struct{}{}
	var failures []string
	for _, production := range files {
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || ident.Name != "refuse" {
				return true
			}
			if len(call.Args) == 0 {
				failures = append(failures, production.Name+": refuse call without a sentinel fails closed")
				return true
			}
			sentinel, ok := call.Args[0].(*ast.Ident)
			if !ok || !sessstateSentinels[sentinel.Name] {
				failures = append(failures, production.Name+": refuse call without a bare sentinel identifier fails closed")
				return true
			}
			first := fileSet.Position(call.Lparen)
			last := fileSet.Position(call.Rparen)
			if !first.IsValid() || !last.IsValid() || first.Line != last.Line {
				failures = append(failures, production.Name+": refuse call outside one line fails closed")
				return true
			}
			site := production.Name + ":" + strconv.Itoa(first.Line)
			if _, duplicate := derived[site]; duplicate {
				failures = append(failures, "two refuse calls share production line "+site)
				return true
			}
			sites = append(sites, refusalSite{site: site, sentinel: sentinel.Name})
			derived[site] = struct{}{}
			return true
		})
	}
	return sites, derived, failures
}

// derivedRefusalSentinels reports the sentinel names carried by the
// derived sites.
func derivedRefusalSentinels(files []invcore.ProductionFile, fileSet *token.FileSet) []string {
	sites, _, _ := deriveSessstateRefusalSites(files, fileSet)
	seen := map[string]struct{}{}
	var sentinels []string
	for _, site := range sites {
		if _, ok := seen[site.sentinel]; !ok {
			seen[site.sentinel] = struct{}{}
			sentinels = append(sentinels, site.sentinel)
		}
	}
	return sentinels
}

// auditSessstateUnrouted fails every fmt.Errorf that wraps a sentinel
// with %w outside the refuse funnel: a sentinel-carrying refusal must
// flow through refuse so the inventory can require its negative path.
// Plain %v diagnostics stay allowed.
func auditSessstateUnrouted(files []invcore.ProductionFile, fileSet *token.FileSet) []string {
	var failures []string
	for _, production := range files {
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Errorf" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			format, ok := call.Args[0].(*ast.BasicLit)
			if !ok || !strings.Contains(format.Value, "%w") {
				return true
			}
			for _, argument := range call.Args[1:] {
				if ident, ok := argument.(*ast.Ident); ok && sessstateSentinels[ident.Name] {
					position := fileSet.Position(call.Lparen)
					failures = append(failures, "sessstate refusals raised outside the refuse funnel (the derived inventory cannot require a negative path for them): "+production.Name+":"+strconv.Itoa(position.Line))
					return true
				}
			}
			return true
		})
	}
	return failures
}

// sessstateConstructorSpec is the alias-audit policy: the local funnel
// must stay a direct call, and the watched owner delegations must stay
// direct qualified calls resolved by import path.
func sessstateConstructorSpec() invcore.ConstructorSpec {
	return invcore.ConstructorSpec{
		Local: map[string]bool{"refuse": true},
		Qualified: map[string]map[string]bool{
			"github.com/relux-works/agent-session-manager/internal/environ":         {"DecodeStrictObject": true, "CheckUUIDv7": true, "CheckUint53Bounds": true},
			"github.com/relux-works/agent-session-manager/internal/scalar":          {"ParseDigest": true, "ParseUUIDv4": true, "ParseProviderID": true},
			"github.com/relux-works/agent-session-manager/internal/terminalbackend": {"ParseID": true},
		},
	}
}

// auditSessstateAliases runs the alias audit over every production
// file, allowlisting the funnel's own declaration position.
func auditSessstateAliases(files []invcore.ProductionFile, fileSet *token.FileSet) []string {
	var failures []string
	spec := sessstateConstructorSpec()
	for _, production := range files {
		allowed := sessstateFunnelPositions(production.Syntax)
		for _, failure := range invcore.AuditConstructorReferences(production.Syntax, fileSet, production.Name, spec) {
			if atAllowedPosition(failure, production.Name, fileSet, allowed) {
				continue
			}
			failures = append(failures, failure)
		}
	}
	return failures
}

func sessstateFunnelPositions(syntax *ast.File) map[token.Pos]bool {
	positions := map[token.Pos]bool{}
	for _, declaration := range syntax.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range value.Names {
				if name.Name != "refuse" || index >= len(value.Values) {
					continue
				}
				if _, ok := value.Values[index].(*ast.FuncLit); ok {
					positions[name.Pos()] = true
				}
			}
		}
	}
	return positions
}

func atAllowedPosition(failure, display string, fileSet *token.FileSet, allowed map[token.Pos]bool) bool {
	for position := range allowed {
		located := fileSet.Position(position)
		if !located.IsValid() {
			continue
		}
		prefix := display + ":" + strconv.Itoa(located.Line) + ":" + strconv.Itoa(located.Column) + ":"
		if strings.Contains(failure, prefix) {
			return true
		}
	}
	return false
}
