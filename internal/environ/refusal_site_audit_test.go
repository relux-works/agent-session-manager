package environ

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
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

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// This file is the refusal-site audit for the environ library.
// Every refusal in this package funnels through the refuse
// constructor var, and every battery witness must resolve to the
// production SITE that refused it — not to a (code, detail) text
// pair a sibling arm could share. Before runtime site recording,
// 62 of 184 arms elsewhere in this repository shared a pair with
// a sibling, so a witness could pass through the wrong arm; this
// audit applies that lesson here in three parts:
//
//  1. Static derivation: every direct refuse() call site is
//     derived from production AST as file:line, with its rule
//     expanded over the frame-fault space for the two
//     passthrough arms. Every call must sit on a single source
//     line so the runtime site and the parsed site agree, and
//     every expanded rule must be unique, which promotes the
//     batteries' full-text assertions to site witnesses.
//  2. Runtime recording: TestMain swaps the refuse var with a
//     recording wrapper, so an aliased call still executes the
//     swapped var and records its real production use site. The
//     driver table below exercises every derived site through
//     the production entries and asserts the recorded rule per
//     vector; the audit requires the exercised site set to
//     equal the derived set in both directions, and the
//     observed rule set to equal the derived rule set exactly.
//  3. Alias audit: every refuse reference outside direct-call
//     position fails, through invcore.AuditConstructorReferences
//     with the funnel's own declaration allowlisted by derived
//     position (a var funnel, unlike a func declaration, needs
//     its position allowlisted explicitly; the plants prove a
//     second binding still fails).

// refusalRecorder is the shared core's runtime direction: the
// production file:line behind every exercised refusal plus the
// rule strings those refusals carried.
var refusalRecorder = invcore.NewSiteRecorder()

// TestMain swaps the refuse constructor var with a recording
// wrapper for the whole in-package run, then audits the
// exercised sites against the derived inventory on a full run.
func TestMain(main *testing.M) {
	original := refuse
	refuse = func(rule, member string) error {
		err := original(rule, member)
		refusalRecorder.Record(rule, 1)
		return err
	}
	code := main.Run()
	if code == 0 && fullPackageTestRun() {
		if failures := auditRefusalSites(); len(failures) != 0 {
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

// refuseSite is one derived refusal site: where it is, what rule
// it emits, and which member it names ("*" when dynamic).
type refuseSite struct {
	file   string
	line   int
	rule   string
	member string
}

func (site refuseSite) key() string {
	return site.file + ":" + strconv.Itoa(site.line)
}

// deriveFaultDetails derives the frame-fault space from decode.go
// production: every Fault* constant value. A sixth fault shape
// fails the audit instead of passing unrostered.
func deriveFaultDetails() (map[string]bool, []string) {
	directory, err := os.Getwd()
	if err != nil {
		return nil, []string{"refusal audit: " + err.Error()}
	}
	path := filepath.Join(directory, "decode.go")
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, []string{"refusal audit: read decode.go: " + err.Error()}
	}
	syntax, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
	if err != nil {
		return nil, []string{"refusal audit: parse decode.go: " + err.Error() + "; an unparseable derivation proves nothing"}
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
					return nil, []string{"refusal audit: Fault constant is not a literal; the extractor cannot see through indirection"}
				}
				text, err := strconv.Unquote(literal.Value)
				if err != nil {
					return nil, []string{"refusal audit: " + err.Error()}
				}
				details[text] = true
			}
		}
	}
	if len(details) == 0 {
		return nil, []string{"refusal audit derived zero fault details; the scanner is blind, not the space empty"}
	}
	return details, nil
}

// deriveRefuseSites derives every direct refuse() call in the
// environ package as file:line with its expanded rule. The two
// frame-fault passthrough arms expand over the derived fault
// space; every other arm carries literal rule and member. A
// non-literal rule outside the passthrough shape, or a call
// spanning several lines, fails closed.
func deriveRefuseSites(files []invcore.ProductionFile, fileSet *token.FileSet, faults map[string]bool) ([]refuseSite, []string) {
	var sites []refuseSite
	var failures []string
	for _, production := range files {
		ast.Inspect(production.Syntax, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if !ok || identifier.Name != "refuse" || len(call.Args) != 2 {
				return true
			}
			position := fileSet.Position(identifier.Pos())
			if !position.IsValid() {
				failures = append(failures, production.Name+": refuse call at an invalid position")
				return true
			}
			end := fileSet.Position(call.End())
			if end.Line != position.Line {
				failures = append(failures, production.Name+":"+strconv.Itoa(position.Line)+": refuse call spans several lines; the runtime site and the parsed site would disagree")
				return true
			}
			site := refuseSite{file: production.Name, line: position.Line}
			switch first := call.Args[0].(type) {
			case *ast.BasicLit:
				rule, err := strconv.Unquote(first.Value)
				if err != nil {
					failures = append(failures, site.key()+": refuse rule is not a string literal")
					return true
				}
				site.rule = rule
				if member, ok := call.Args[1].(*ast.BasicLit); ok {
					text, err := strconv.Unquote(member.Value)
					if err != nil {
						failures = append(failures, site.key()+": refuse member is not a string literal")
						return true
					}
					site.member = text
				} else {
					site.member = "*"
				}
				sites = append(sites, site)
			case *ast.BinaryExpr:
				prefix, ok := passthroughPrefix(first)
				if !ok {
					failures = append(failures, site.key()+": refuse rule is neither a literal nor a frame-fault passthrough; the extractor cannot see through it")
					return true
				}
				for detail := range faults {
					expanded := site
					expanded.rule = prefix + detail
					expanded.member = "*"
					sites = append(sites, expanded)
				}
			default:
				failures = append(failures, site.key()+": refuse rule is neither a literal nor a frame-fault passthrough; the extractor cannot see through it")
				return true
			}
			return true
		})
	}
	return sites, failures
}

// passthroughPrefix matches `"tuple "+fault.Detail` and
// `"observation "+fault.Detail` and reports the literal prefix.
func passthroughPrefix(expression *ast.BinaryExpr) (string, bool) {
	if expression.Op != token.ADD {
		return "", false
	}
	literal, ok := expression.X.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	prefix, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	selector, ok := expression.Y.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Detail" {
		return "", false
	}
	if identifier, ok := selector.X.(*ast.Ident); !ok || identifier.Name != "fault" {
		return "", false
	}
	return prefix, true
}

// funnelDeclarationPositions derives the positions of the refuse
// funnel's own declaration: a package-level `var refuse = ...`
// binding a function literal. invcore's audit allowlists func
// declarations by position but has no var-funnel rule, so this
// audit allowlists the derived position explicitly; any other
// binding of the name still fails.
func funnelDeclarationPositions(syntax *ast.File) map[token.Pos]bool {
	positions := map[token.Pos]bool{}
	for _, decl := range syntax.Decls {
		general, ok := decl.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, spec := range general.Specs {
			value, ok := spec.(*ast.ValueSpec)
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

// auditRefuseReferences runs the alias audit over one file,
// allowlisting the funnel's own declaration position. Every
// other refuse reference outside direct-call position — an
// aliased, rebound, shadowed, or otherwise indirected use —
// fails.
func auditRefuseReferences(syntax *ast.File, fileSet *token.FileSet, display string) []string {
	spec := invcore.ConstructorSpec{Local: map[string]bool{"refuse": true}}
	allowed := funnelDeclarationPositions(syntax)
	var failures []string
	for _, failure := range invcore.AuditConstructorReferences(syntax, fileSet, display, spec) {
		if failureAtFunnelDeclaration(failure, display, fileSet, allowed) {
			continue
		}
		failures = append(failures, failure)
	}
	return failures
}

// failureAtFunnelDeclaration reports whether an audit failure
// names the funnel's own declaration position.
func failureAtFunnelDeclaration(failure, display string, fileSet *token.FileSet, allowed map[token.Pos]bool) bool {
	for position := range allowed {
		located := fileSet.Position(position)
		if !located.IsValid() {
			continue
		}
		prefix := display + ":" + strconv.Itoa(located.Line) + ":" + strconv.Itoa(located.Column) + ":"
		if strings.HasPrefix(failure, prefix) {
			return true
		}
	}
	return false
}

// auditRefusalSites derives the refusal inventory from package
// source and compares it against the exercised run: derived
// sites without an exercised path and exercised sites outside
// the derivation fail together through the invcore harness, the
// observed rule set must equal the derived rule set exactly,
// every expanded rule must be unique (so a full-text witness
// resolves to exactly one site), and every refuse reference
// outside direct-call position fails.
func auditRefusalSites() []string {
	directory, err := os.Getwd()
	if err != nil {
		return []string{"refusal audit: " + err.Error()}
	}
	files, fileSet, failures := invcore.ScanProduction(directory)
	for _, failure := range failures {
		return []string{"refusal audit: " + failure}
	}
	faults, faultFailures := deriveFaultDetails()
	if len(faultFailures) != 0 {
		return faultFailures
	}
	sites, siteFailures := deriveRefuseSites(files, fileSet, faults)
	failures = append(failures, siteFailures...)
	seen := map[string]string{}
	for _, site := range sites {
		if first, duplicate := seen[site.rule]; duplicate {
			failures = append(failures, "refusal audit: rule "+strconv.Quote(site.rule)+" is shared by "+first+" and "+site.key()+"; a witness asserting it resolves to two sites, not one")
			continue
		}
		seen[site.rule] = site.key()
	}
	derived := map[string]struct{}{}
	rules := map[string]struct{}{}
	for _, site := range sites {
		derived[site.key()] = struct{}{}
		rules[site.rule] = struct{}{}
	}
	var wantRules []string
	for rule := range rules {
		wantRules = append(wantRules, rule)
	}
	sort.Strings(wantRules)
	failures = append(failures, refusalRecorder.AuditSites("environ refusals", len(files), derived, wantRules)...)
	for _, production := range files {
		display := production.Name
		failures = append(failures, auditRefuseReferences(production.Syntax, fileSet, display)...)
	}
	return failures
}

// auditDigest returns the pinned digest of one seed, mirroring
// the shared battery fixtures without importing them.
func auditDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// auditTupleJSON is one valid Environment Tuple over locally
// computed identities.
func auditTupleJSON() []byte {
	return []byte(`{"environment_id":"test.env","environment_version":"2.1.0","platform":"linux","architecture":"amd64","store_schema_fingerprint":` +
		strconv.Quote(auditDigest("test-store-fingerprint")) + `,"adapter_version":"1.2.3"}`)
}

// auditCapabilitiesJSON is the exact eight-name capability map
// with every capability available.
func auditCapabilitiesJSON() string {
	names := []string{
		"directory_discovery",
		"directory_incremental_scan",
		"directory_head_digest",
		"directory_tail_preview",
		"native_title_read",
		"native_runtime_observation",
		"existing_session_adoption",
		"native_resume",
	}
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, strconv.Quote(name)+`:`+auditAvailableCapabilityJSON())
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// auditAvailableCapabilityJSON is one available CapabilityResult
// with null reason.
func auditAvailableCapabilityJSON() string {
	return `{"status":"available","reason_code":null,"evidence_ids":[],"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`
}

// auditObservationJSON is one valid Environment Observation over
// locally computed identities.
func auditObservationJSON() []byte {
	return []byte(`{"schema":"urn:ax:schema:environment-observation","schema_version":"1.0.0","observation_id":` +
		strconv.Quote(auditDigest("test-observation")) + `,"host_id":"0198f4c8-8e50-7f66-8f70-1234567890ab","installation_id":` +
		strconv.Quote(auditDigest("test-installation")) + `,"environment_id":"test.env","environment_version":"2.1.0","provider_id":"test-provider",` +
		`"platform":"linux","architecture":"amd64","backend_realm_fingerprint":` + strconv.Quote(auditDigest("test-realm")) +
		`,"capabilities":` + auditCapabilitiesJSON() + `,"authentication_status":"available","runtime_status":"available",` +
		`"observed_at":"2026-08-19T04:05:00.000Z","extensions":{}}`)
}

// decodeAuditMembers parses one JSON object into raw member
// texts.
func decodeAuditMembers(body []byte) (map[string]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	members := make(map[string]string, len(raw))
	for name, value := range raw {
		members[name] = string(value)
	}
	return members, nil
}

// encodeAuditMembers renders the member map as one JSON object
// with sorted member names, so the drivers are deterministic.
func encodeAuditMembers(members map[string]string) []byte {
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(members))
	for _, name := range names {
		parts = append(parts, strconv.Quote(name)+":"+members[name])
	}
	return []byte("{" + strings.Join(parts, ",") + "}")
}

// mutateAuditMember returns the object with one top-level member
// replaced by the given raw JSON.
func mutateAuditMember(t *testing.T, body []byte, member, replacement string) []byte {
	t.Helper()
	parsed, err := decodeAuditMembers(body)
	if err != nil {
		t.Fatalf("driver: fixture is not JSON: %v", err)
	}
	parsed[member] = replacement
	return encodeAuditMembers(parsed)
}

// dropAuditMember returns the object without one top-level
// member.
func dropAuditMember(t *testing.T, body []byte, member string) []byte {
	t.Helper()
	parsed, err := decodeAuditMembers(body)
	if err != nil {
		t.Fatalf("driver: fixture is not JSON: %v", err)
	}
	delete(parsed, member)
	return encodeAuditMembers(parsed)
}

// appendAuditMember returns the object with one added top-level
// member.
func appendAuditMember(t *testing.T, body []byte, member, literal string) []byte {
	t.Helper()
	parsed, err := decodeAuditMembers(body)
	if err != nil {
		t.Fatalf("driver: fixture is not JSON: %v", err)
	}
	if _, present := parsed[member]; present {
		t.Fatalf("driver: %q already present", member)
	}
	parsed[member] = literal
	return encodeAuditMembers(parsed)
}

// refusalDriver is one exercised refusal: the body, the entry
// that must refuse it, and the exact rule the recorder must
// observe behind it.
type refusalDriver struct {
	name  string
	body  []byte
	drive func([]byte) error
	rule  string
	// member is the exact member the refusal must name, or ""
	// when the refusal must name none. Missing-member members
	// follow the required-list order and single-unknown members
	// are deterministic, so every expectation here is exact.
	member string
}

// refusalDrivers exercises every derived refuse site through the
// production entries. The frame-fault passthrough arms expand to
// one vector per fault detail; every literal arm owns one
// vector. Each vector asserts the recorded rule, so a witness
// that passed through the wrong arm fails here rather than
// passing on shared text.
func refusalDrivers(t *testing.T) []refusalDriver {
	t.Helper()
	tuple := auditTupleJSON()
	observation := auditObservationJSON()
	mutateTuple := func(member, literal string) []byte { return mutateAuditMember(t, tuple, member, literal) }
	mutateObservation := func(member, literal string) []byte {
		return mutateAuditMember(t, observation, member, literal)
	}
	tupleEntry := func(body []byte) error {
		_, err := DecodeTuple(body)
		return err
	}
	observationEntry := func(body []byte) error {
		_, err := DecodeEnvironmentObservation(body)
		return err
	}
	badDigest := strconv.Quote("sha256:zzzz")
	return []refusalDriver{
		{name: "tuple non-utf8", body: []byte("{\"environment_id\":\"a\xff\"}"), drive: tupleEntry, rule: "tuple not valid UTF-8"},
		{name: "tuple lone surrogate", body: []byte("{\"environment_id\":\"A\\ud800B\"}"), drive: tupleEntry, rule: "tuple lone surrogate escape"},
		{name: "tuple non-object", body: []byte("[1,2]"), drive: tupleEntry, rule: "tuple not a JSON object"},
		{name: "tuple duplicate", body: []byte(`{"environment_id":"a","environment_id":"b"}`), drive: tupleEntry, rule: "tuple duplicate member", member: "environment_id"},
		{name: "tuple trailing", body: []byte(`{"environment_id":"a"} {}`), drive: tupleEntry, rule: "tuple trailing data after the object"},
		{name: "tuple unknown", body: appendAuditMember(t, tuple, "score", `1`), drive: tupleEntry, rule: "tuple carries unknown member", member: "score"},
		{name: "tuple missing", body: dropAuditMember(t, tuple, "platform"), drive: tupleEntry, rule: "tuple misses a required member", member: "platform"},
		{name: "tuple environment id", body: mutateTuple("environment_id", `"Test.Env"`), drive: tupleEntry, rule: "tuple environment identifier is not an environment-id", member: "environment_id"},
		{name: "tuple version", body: mutateTuple("environment_version", `""`), drive: tupleEntry, rule: "tuple environment version is not a string[1..128]", member: "environment_version"},
		{name: "tuple platform string", body: mutateTuple("platform", `42`), drive: tupleEntry, rule: "tuple platform is not a string", member: "platform"},
		{name: "tuple platform vocab", body: mutateTuple("platform", `"darwin"`), drive: tupleEntry, rule: "tuple platform is outside linux|macos|windows|wsl2", member: "platform"},
		{name: "tuple architecture", body: mutateTuple("architecture", `"x86"`), drive: tupleEntry, rule: "tuple architecture is outside amd64|arm64", member: "architecture"},
		{name: "tuple fingerprint", body: mutateTuple("store_schema_fingerprint", badDigest), drive: tupleEntry, rule: "tuple store fingerprint is not a digest", member: "store_schema_fingerprint"},
		{name: "tuple semver", body: mutateTuple("adapter_version", `"1.2"`), drive: tupleEntry, rule: "tuple adapter version is not SemVer", member: "adapter_version"},
		{name: "observation non-utf8", body: []byte("{\"schema\":\"a\xff\"}"), drive: observationEntry, rule: "observation not valid UTF-8"},
		{name: "observation lone surrogate", body: []byte("{\"schema\":\"A\\ud800B\"}"), drive: observationEntry, rule: "observation lone surrogate escape"},
		{name: "observation non-object", body: []byte("[1,2]"), drive: observationEntry, rule: "observation not a JSON object"},
		{name: "observation duplicate", body: []byte(`{"schema":"a","schema":"b"}`), drive: observationEntry, rule: "observation duplicate member", member: "schema"},
		{name: "observation trailing", body: []byte(`{"schema":"a"} {}`), drive: observationEntry, rule: "observation trailing data after the object"},
		{name: "observation unknown", body: appendAuditMember(t, observation, "score", `1`), drive: observationEntry, rule: "observation carries unknown member", member: "score"},
		{name: "observation missing", body: dropAuditMember(t, observation, "platform"), drive: observationEntry, rule: "observation misses a required member", member: "platform"},
		{name: "observation schema", body: mutateObservation("schema", `"urn:ax:schema:session-adapter-manifest"`), drive: observationEntry, rule: "observation schema is not the environment observation", member: "schema"},
		{name: "observation schema version", body: mutateObservation("schema_version", `"2.0.0"`), drive: observationEntry, rule: "observation schema version is not 1.0.0", member: "schema_version"},
		{name: "observation id", body: mutateObservation("observation_id", badDigest), drive: observationEntry, rule: "observation identifier is not a digest", member: "observation_id"},
		{name: "observation host", body: mutateObservation("host_id", `"not-a-uuid"`), drive: observationEntry, rule: "observation host identifier is not a UUIDv7", member: "host_id"},
		{name: "observation installation", body: mutateObservation("installation_id", `"42"`), drive: observationEntry, rule: "observation installation identifier is not a digest", member: "installation_id"},
		{name: "observation environment id", body: mutateObservation("environment_id", `"Test.Env"`), drive: observationEntry, rule: "observation environment identifier is not an environment-id", member: "environment_id"},
		{name: "observation environment version", body: mutateObservation("environment_version", `""`), drive: observationEntry, rule: "observation environment version is not a string[1..128]", member: "environment_version"},
		{name: "observation provider string", body: mutateObservation("provider_id", `42`), drive: observationEntry, rule: "observation provider identifier is not a string", member: "provider_id"},
		{name: "observation provider grammar", body: mutateObservation("provider_id", `"Test-Provider"`), drive: observationEntry, rule: "observation provider identifier is not a provider-id", member: "provider_id"},
		{name: "observation platform string", body: mutateObservation("platform", `42`), drive: observationEntry, rule: "observation platform is not a string", member: "platform"},
		{name: "observation platform vocab", body: mutateObservation("platform", `"darwin"`), drive: observationEntry, rule: "observation platform is outside linux|macos|windows|wsl2", member: "platform"},
		{name: "observation architecture", body: mutateObservation("architecture", `"x86"`), drive: observationEntry, rule: "observation architecture is outside amd64|arm64", member: "architecture"},
		{name: "observation realm", body: mutateObservation("backend_realm_fingerprint", `"null"`), drive: observationEntry, rule: "observation realm fingerprint is not a digest", member: "backend_realm_fingerprint"},
		{name: "observation capabilities", body: mutateObservation("capabilities", `{}`), drive: observationEntry, rule: "observation capabilities are not the exact eight-name result map", member: "capabilities"},
		{name: "observation authentication", body: mutateObservation("authentication_status", `"signed-in"`), drive: observationEntry, rule: "observation authentication status is outside available|missing|expired|unknown", member: "authentication_status"},
		{name: "observation runtime", body: mutateObservation("runtime_status", `"sleeping"`), drive: observationEntry, rule: "observation runtime status is outside available|degraded|unavailable", member: "runtime_status"},
		{name: "observation timestamp", body: mutateObservation("observed_at", `"yesterday"`), drive: observationEntry, rule: "observation timestamp is not a timestamp", member: "observed_at"},
		{name: "observation extensions", body: mutateObservation("extensions", `{"nodots":1}`), drive: observationEntry, rule: "observation extensions are not reverse-DNS keyed", member: "extensions"},
	}
}

// TestEveryRefusalSiteIsExercised drives the driver table
// through the production entries and requires each vector to
// refuse with exactly its ledgered rule recorded behind it. The
// TestMain audit compares the exercised site set against the
// derived inventory; this test proves each witness reaches its
// own arm.
func TestEveryRefusalSiteIsExercised(t *testing.T) {
	for _, driver := range refusalDrivers(t) {
		t.Run(driver.name, func(t *testing.T) {
			before := refusalRecorder.Codes()
			err := driver.drive(driver.body)
			if err == nil {
				t.Fatalf("admitted %s, want rule %q", driver.body, driver.rule)
			}
			want := "environment " + driver.rule
			if driver.member != "" {
				want += ": " + driver.member
			}
			if err.Error() != want {
				t.Fatalf("error = %q, want exactly %q", err.Error(), want)
			}
			after := refusalRecorder.Codes()
			var fresh []string
			for _, code := range after {
				seen := false
				for _, old := range before {
					if old == code {
						seen = true
					}
				}
				if !seen {
					fresh = append(fresh, code)
				}
			}
			if len(fresh) != 1 || fresh[0] != driver.rule {
				t.Fatalf("recorded rules = %q, want exactly [%q]", fresh, driver.rule)
			}
		})
	}
}

// TestRefuseAliasShapesAreRejected plants the bypass shapes
// against the refuse audit: a rebound var, a shadowed name, and
// a second funnel-shaped declaration alongside an alias. Direct
// calls and the funnel's own declaration stay clean.
func TestRefuseAliasShapesAreRejected(t *testing.T) {
	audit := func(source string) []string {
		t.Helper()
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("control does not parse: %s", failure)
		}
		return auditRefuseReferences(syntax, fileSet, "plant.go")
	}
	rebound := audit("package environ\nfunc use(member string) error { return reboundRefuse(\"rule\", member) }\nvar reboundRefuse = refuse\nfunc direct(member string) error { return refuse(\"rule\", member) }\n")
	if len(rebound) == 0 {
		t.Fatal("rebound var admitted: the alias bypasses site attribution")
	}
	shadow := audit("package environ\nfunc use(refuse string) string { return refuse }\n")
	if len(shadow) == 0 {
		t.Fatal("shadowed constructor name admitted: the audit cannot classify it")
	}
	clean := audit("package environ\nfunc use(member string) error { return refuse(\"rule\", member) }\n")
	if len(clean) != 0 {
		t.Fatalf("direct call reported: %s", strings.Join(clean, "; "))
	}
	funnel := audit("package environ\nimport \"fmt\"\nvar refuse = func(rule, member string) error { return fmt.Errorf(rule) }\nfunc use(member string) error { return refuse(\"rule\", member) }\n")
	if len(funnel) != 0 {
		t.Fatalf("funnel declaration reported: %s", strings.Join(funnel, "; "))
	}
}
