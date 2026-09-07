package environ

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

// This file is the alias-bypass audit over the shared-rule census
// scope. The name layer (census_test.go) derives definitions by
// identifier and the shape layer (shape_census_test.go) derives
// implementations by texture; neither sees a use-site alias — an
// import alias rebound to a shared helper, or a package-level var
// rebound to one — because no new definition appears. This audit
// closes that direction with invcore.AuditConstructorReferences:
// every reference to a ledgered shared-rule name outside
// direct-call position fails, every shape the audit cannot
// classify (dot imports, shadowed names) fails closed, and the
// synthetic plants below prove each shape reports while direct
// calls and the constructors' own declarations stay clean.

// sharedRuleConstructorSpec watches the ledgered shared-rule
// names: package-local bare identifiers by spelling, and the
// canonical environ members by import path and member name, so an
// import alias cannot change what the audit sees. The local set
// is the name layer's symbol map minus
// validateProviderIdentityRecord: canonicaljson registers that
// validator by function value in its schema dispatch table
// (closed_shapes.go), which is same-function dispatch rather
// than a copy or an alias — every invocation still executes the
// censused definition. A true rebinding of it (`var f =
// validateProviderIdentityRecord`) still fails, via the shape
// layer's var-alias rule, which inherits the helper's
// string-measure shape onto the new name and reports it as an
// unregistered site (proven by the C1 controls in
// TestShapeCensusCatchesControls). The qualified set is the
// environ surface the facades delegate to.
func sharedRuleConstructorSpec() invcore.ConstructorSpec {
	local := make(map[string]bool, len(sharedFunctionSymbols))
	for name := range sharedFunctionSymbols {
		if name == "validateProviderIdentityRecord" {
			continue
		}
		local[name] = true
	}
	return invcore.ConstructorSpec{
		Local: local,
		Qualified: map[string]map[string]bool{
			"github.com/relux-works/agent-session-manager/internal/environ": {
				"DecodeStrictObject":           true,
				"HasLoneSurrogateEscape":       true,
				"StringLength":                 true,
				"CheckStringBounds":            true,
				"CheckUint53Bounds":            true,
				"CheckDigest":                  true,
				"CheckUUIDv7":                  true,
				"CheckTimestamp":               true,
				"CheckSortedUniqueStrings":     true,
				"CheckSortedUniqueDigests":     true,
				"CheckExtensions":              true,
				"CheckEnvironmentID":           true,
				"CheckSemver":                  true,
				"DecodeTuple":                  true,
				"DecodeEnvironmentObservation": true,
			},
		},
	}
}

// censusScopePackages lists the audit scope in a fixed order: the
// name layer's package set, which this audit shares by
// construction.
func censusScopePackages() []string {
	packages := make([]string, 0, len(censusPackages))
	for pkg := range censusPackages {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	return packages
}

// TestCensusScopeHasNoAliasedSharedRules requires every
// shared-rule reference in census-scope production to sit in
// direct-call position: an aliased, rebound, or otherwise
// indirected construction fails here even though the name and
// shape censuses stay green over it. The delegation this leaf
// added (sessadapter and dirnode calling environ.Check* and
// friends directly) is the allowed position and stays clean.
func TestCensusScopeHasNoAliasedSharedRules(t *testing.T) {
	root, err := internalRoot(t)
	if err != nil {
		t.Fatalf("alias audit: %v", err)
	}
	spec := sharedRuleConstructorSpec()
	for _, pkg := range censusScopePackages() {
		files, fileSet := invcore.MustScanProduction(t, filepath.Join(root, pkg))
		for _, production := range files {
			display := pkg + "/" + production.Name
			invcore.MustAuditConstructorReferences(t, production.Syntax, fileSet, display, spec)
		}
	}
}

// aliasPlant is one synthetic control for the audit: the source,
// whether the audit must report, and the fragment the failure
// must carry ("" for clean plants).
type aliasPlant struct {
	name        string
	source      string
	wantFailure string
}

// aliasPlants enumerates the bypass shapes. The import-alias and
// var-binding plants are the shapes that walked through every
// identifier-keyed gate in this repository's history; the dot
// import and shadow plants are the shapes the audit cannot
// classify and must fail closed rather than prune.
func aliasPlants() []aliasPlant {
	return []aliasPlant{
		{
			name: "var binding of a local shared helper reports",
			source: `package provider
func stringLength(value string) int { return len(value) }
var measure = stringLength
func use(value string) int { return measure(value) }
`,
			wantFailure: "outside direct-call position",
		},
		{
			name: "import alias rebound to the canonical helper reports",
			source: `package provider
import env "github.com/relux-works/agent-session-manager/internal/environ"
var length = env.StringLength
func use(value string) int { return length(value) }
`,
			wantFailure: "outside direct-call position",
		},
		{
			name: "aliased direct call stays clean",
			source: `package provider
import env "github.com/relux-works/agent-session-manager/internal/environ"
func use(value string) int { return env.StringLength(value) }
`,
		},
		{
			name: "canonical direct call stays clean",
			source: `package provider
import "github.com/relux-works/agent-session-manager/internal/environ"
func use(value string) int { return environ.StringLength(value) }
func decode(data []byte) bool {
	members, _ := environ.DecodeStrictObject(data)
	return members != nil
}
`,
		},
		{
			name: "local direct call stays clean",
			source: `package provider
func stringLength(value string) int { return len(value) }
func use(value string) int { return stringLength(value) }
`,
		},
		{
			name: "dot import of the watched path fails closed",
			source: `package provider
import . "github.com/relux-works/agent-session-manager/internal/environ"
func use(value string) int { return StringLength(value) }
`,
			wantFailure: "dot-imports watched path",
		},
		{
			name: "shadowed constructor name fails closed",
			source: `package provider
func use(stringLength string) string { return stringLength }
`,
			wantFailure: "stringLength",
		},
	}
}

// TestCensusAuditCatchesAliasShapes drives every plant through
// the production audit spec and requires the ledgered outcome:
// reporting shapes fail with their fragment, clean shapes stay
// clean. A plant that stops reporting proves nothing and fails
// here by construction.
func TestCensusAuditCatchesAliasShapes(t *testing.T) {
	spec := sharedRuleConstructorSpec()
	for _, plant := range aliasPlants() {
		t.Run(plant.name, func(t *testing.T) {
			syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(plant.source))
			if failure != "" {
				t.Fatalf("control does not parse: %s", failure)
			}
			failures := invcore.AuditConstructorReferences(syntax, fileSet, "plant.go", spec)
			if plant.wantFailure == "" {
				if len(failures) != 0 {
					t.Fatalf("clean plant reported: %s", strings.Join(failures, "; "))
				}
				return
			}
			matched := false
			for _, failure := range failures {
				if strings.Contains(failure, plant.wantFailure) {
					matched = true
				}
			}
			if !matched {
				t.Fatalf("plant reported %q, want a failure containing %q", failures, plant.wantFailure)
			}
		})
	}
}

// TestCensusAuditParsingFailsClosed proves the plant harness
// itself is fail-closed: an unparseable control is a harness
// failure, never a clean plant.
func TestCensusAuditParsingFailsClosed(t *testing.T) {
	_, _, failure := invcore.ParseSource("plant.go", []byte("package provider\nfunc broken( {\n"))
	if failure == "" {
		t.Fatal("unparseable control parsed clean; the harness would pass a blind plant")
	}
}

// TestRegistryExcludedNameStillFailsOnRebind proves the
// validateProviderIdentityRecord exclusion above loses nothing:
// a true rebinding of the excluded name derives an unregistered
// shape site through the production shape extractor, so the
// shape census fails it even though this audit stays silent.
func TestRegistryExcludedNameStillFailsOnRebind(t *testing.T) {
	source := `package provider
import "unicode/utf8"
func validateProviderIdentityRecord(value string) int { return utf8.RuneCountInString(value) }
var rebound = validateProviderIdentityRecord
`
	sites := shapeSitesInPackage("provider", map[string][]byte{"control.go": []byte(source)})
	matched := false
	for _, site := range sites {
		if site.symbol == "rebound" {
			matched = true
			if _, ok := shapeLedger[site.key()]; ok {
				t.Fatalf("rebound site %q is registered; the control proves nothing", site.key())
			}
			if _, ok := sharedLedger[site.key()]; ok {
				t.Fatalf("rebound site %q is registered; the control proves nothing", site.key())
			}
		}
	}
	if !matched {
		t.Fatalf("rebinding derived no site for the new name (all sites: %v); the shape layer would pass a true alias", sites)
	}
}
