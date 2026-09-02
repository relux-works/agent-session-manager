package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestResolvePathsPopulatesEveryOverrideRegistryClass is the subsuming check
// named by the two structural guards in loader.go that refuse when a resolved
// path for a registry class is missing. Those guards cannot be reached while
// this property holds, and this test reddens the moment resolution and the
// registry diverge for any supported platform.
func TestResolvePathsPopulatesEveryOverrideRegistryClass(t *testing.T) {
	environments := map[scalar.Platform]map[string]string{
		scalar.PlatformMacOS: nil,
		scalar.PlatformLinux: {"XDG_RUNTIME_DIR": "/run/user/1000"},
		scalar.PlatformWSL2:  {"XDG_RUNTIME_DIR": "/run/user/1000"},
		scalar.PlatformWindows: {
			"APPDATA": "C:\\Users\\test\\AppData\\Roaming", "LOCALAPPDATA": "C:\\Users\\test\\AppData\\Local",
		},
	}
	platforms := []scalar.Platform{
		scalar.PlatformMacOS, scalar.PlatformLinux, scalar.PlatformWSL2, scalar.PlatformWindows,
	}
	for _, platform := range platforms {
		t.Run(platform.String(), func(t *testing.T) {
			paths, err := ResolvePaths(fixtureInputs(platform, environments[platform], newFakeFileSystem()), nil)
			if err != nil {
				t.Fatalf("ResolvePaths(%s) error = %v", platform, err)
			}
			registry := OverrideRegistry()
			if len(registry) == 0 {
				t.Fatal("override registry is empty")
			}
			for _, specification := range registry {
				resolved, ok := paths.Path(specification.Class)
				if !ok || resolved.Value.String() == "" {
					t.Fatalf("ResolvePaths(%s) missing class %q", platform, specification.Class)
				}
			}
			if len(paths.All()) != len(registry) {
				t.Fatalf("ResolvePaths(%s) resolved %d classes, registry has %d", platform, len(paths.All()), len(registry))
			}
		})
	}
}

// TestLoadRefusesAbsentConfigWhoseParentIsNotResolvable drives the real Load
// entry with an absent selected file whose own parent cannot be derived, which
// is the only refusal in loadAbsentConfig that is not a stat outcome.
func TestLoadRefusesAbsentConfigWhoseParentIsNotResolvable(t *testing.T) {
	files := newFakeFileSystem()
	inputs := fixtureInputs(scalar.PlatformWindows, map[string]string{
		"AX_CONFIG": `\\server\share`, "APPDATA": `C:\Users\test\AppData\Roaming`, "LOCALAPPDATA": `C:\Users\test\AppData\Local`,
	}, files)

	_, err := Load(inputs, nil)
	if !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("Load(absent UNC share root) error = %v, want ErrInvalidContext", err)
	}
	var refusal *Error
	if !errors.As(err, &refusal) || refusal.Operation != "resolve selected file parent" {
		t.Fatalf("Load(absent UNC share root) refusal = %#v", refusal)
	}

	// Isolation: the same absent file one segment deeper resolves its parent
	// and refuses only because that parent does not exist.
	deeper := fixtureInputs(scalar.PlatformWindows, map[string]string{
		"AX_CONFIG": `\\server\share\ax\config.toml`, "APPDATA": `C:\Users\test\AppData\Roaming`, "LOCALAPPDATA": `C:\Users\test\AppData\Local`,
	}, files)
	_, err = Load(deeper, nil)
	if !errors.Is(err, ErrConfigParentNotFound) {
		t.Fatalf("Load(absent nested UNC path) error = %v, want ErrConfigParentNotFound", err)
	}
}

// TestRefusalSubsumptionInventoryIsPinned ratchets the set of refusal sites
// excluded from the exercised-site inventory. A new exclusion cannot be added
// silently; it must be justified here.
// TestRefusalSubsumptionInventoryIsPinned reads the pin, not just its size.
// Each key is "<file>: <clause>" and must resolve to exactly one marked site in
// that file whose source line names that clause, so a subsumption claim cannot
// be moved, renamed, or swapped for a different one while the count holds.
func TestRefusalSubsumptionInventoryIsPinned(t *testing.T) {
	want := map[string]string{
		"loader.go: capture working directory": "os.Getwd has no injectable seam and does not fail on a supported host",
		"loader.go: read selected file":        "ResolvePaths populates every overrideRegistry class",
		"loader.go: inspect selected root":     "ResolvePaths populates every overrideRegistry class",
		"migration.go: encode target":          "each encoder refusal is pinned on its own clause",
		"writer.go: terminal.backend":          "the v2 source reader admits only tmux or conpty",
		"writer.go: v2 wire TOML":              "the v1 reader supplies only closed map-free values",
		"writer.go: v2 re-read":                "no valid Configuration 1.0.0 source can produce a v2 document the re-read refuses",
		"schema.go: envelope TOML syntax":      "the envelope parse is subsumed by the closed-shape parse",
	}
	got := subsumedRefusalSites(t)
	if len(got) != len(want) {
		t.Fatalf("subsumed refusal sites = %d (%v), pinned %d; justify any change here", len(got), got, len(want))
	}
	matched := map[string]string{}
	for pin, justification := range want {
		file, clause, found := strings.Cut(pin, ": ")
		if !found || justification == "" {
			t.Fatalf("subsumption pin %q must be %q with a non-empty justification", pin, "<file>: <clause>")
		}
		var sites []string
		for site, line := range got {
			if strings.HasPrefix(site, file+":") && strings.Contains(line, clause) {
				sites = append(sites, site)
			}
		}
		if len(sites) != 1 {
			t.Fatalf("subsumption pin %q resolved to %d marked sites %v, want exactly one", pin, len(sites), sites)
		}
		if owner, taken := matched[sites[0]]; taken {
			t.Fatalf("subsumption pins %q and %q both claim %s", owner, pin, sites[0])
		}
		matched[sites[0]] = pin
	}
	for site := range got {
		if _, claimed := matched[site]; !claimed {
			t.Fatalf("marked refusal site %s is not claimed by any pin; justify it here", site)
		}
	}
}

// subsumedRefusalSites returns every marked site as "<file>:<line>" mapped to
// its source line, so the pin can be checked against what the code says rather
// than only against how many marks exist.
func subsumedRefusalSites(t *testing.T) map[string]string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	sites := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for number, line := range strings.Split(string(source), "\n") {
			if strings.Contains(line, "config-refusal-subsumed:") {
				sites[entry.Name()+":"+itoa(number+1)] = line
			}
		}
	}
	return sites
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}

// TestRefusalInventoryDerivesEveryRefusalFormThePackageDeclares pins what the
// audit actually scans. The inventory is derived from sources, so this asserts
// the derivation sees every error type the package declares and one
// instrumented constructor for each.
func TestRefusalInventoryDerivesEveryRefusalFormThePackageDeclares(t *testing.T) {
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := deriveRefusalInventory(directory)
	if err != nil {
		t.Fatalf("deriveRefusalInventory() error = %v", err)
	}
	wantTypes := []string{"DocumentError", "Error", "MigrationError"}
	if strings.Join(inventory.ErrorTypes, ",") != strings.Join(wantTypes, ",") {
		t.Fatalf("derived error types = %v, want %v; a new refusal form must be instrumented, not ignored", inventory.ErrorTypes, wantTypes)
	}
	wantConstructors := map[string]string{
		"configError":    "DocumentError",
		"loaderError":    "Error",
		"migrationError": "MigrationError",
	}
	if len(inventory.Constructors) != len(wantConstructors) {
		t.Fatalf("derived refusal constructors = %v, want %v", inventory.Constructors, wantConstructors)
	}
	for name, built := range wantConstructors {
		if inventory.Constructors[name] != built {
			t.Fatalf("constructor %q builds %q, want %q", name, inventory.Constructors[name], built)
		}
	}
	if len(inventory.Uninstrumented) != 0 || len(inventory.Duplicated) != 0 || len(inventory.StrayLiterals) != 0 {
		t.Fatalf("derived inventory defects: uninstrumented=%v duplicated=%v stray=%v",
			inventory.Uninstrumented, inventory.Duplicated, inventory.StrayLiterals)
	}
	if len(inventory.Sites) < 50 {
		t.Fatalf("derived refusal sites = %d, far below the package's refusal surface", len(inventory.Sites))
	}
}

// TestRefusalInventoryDerivationFailsOnAnUnscannedRefusalForm is the gate's
// proof that it can fail. The same derivation is run over a synthetic package
// that adds a fourth error type, a duplicated constructor, and a raw literal
// that bypasses its constructor; each must be reported.
func TestRefusalInventoryDerivationFailsOnAnUnscannedRefusalForm(t *testing.T) {
	directory := t.TempDir()
	source := `package fixture

type AlphaError struct{ Clause string }

func (err *AlphaError) Error() string { return "alpha" }

type BetaError struct{ Clause string }

func (err *BetaError) Error() string { return "beta" }

type GammaError struct{ Clause string }

func (err *GammaError) Error() string { return "gamma" }

var alphaError = func(value AlphaError) error { return &value }

var gammaError = func(value GammaError) error { return &value }

var gammaErrorDuplicate = func(value GammaError) error { return &value }

func exercised() error { return alphaError(AlphaError{Clause: "one"}) }

func subsumed() error {
	return alphaError(AlphaError{Clause: "two"}) // config-refusal-subsumed: fixture
}

func bypass() error {
	value := AlphaError{Clause: "three"}
	return &value
}

func uninstrumented() error { return &BetaError{Clause: "four"} }
`
	if err := os.WriteFile(filepath.Join(directory, "fixture.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	inventory, err := deriveRefusalInventory(directory)
	if err != nil {
		t.Fatalf("deriveRefusalInventory(fixture) error = %v", err)
	}

	// A fourth error type with no constructor is reported, not silently green.
	if strings.Join(inventory.Uninstrumented, ",") != "BetaError" {
		t.Fatalf("uninstrumented = %v, want [BetaError]", inventory.Uninstrumented)
	}
	// A byte-identical duplicate constructor is reported.
	if strings.Join(inventory.Duplicated, ",") != "GammaError" {
		t.Fatalf("duplicated = %v, want [GammaError]", inventory.Duplicated)
	}
	// Both bypasses of an instrumented constructor are reported.
	wantStray := []string{"fixture.go:28 AlphaError", "fixture.go:32 BetaError"}
	if strings.Join(inventory.StrayLiterals, ",") != strings.Join(wantStray, ",") {
		t.Fatalf("stray literals = %v, want %v", inventory.StrayLiterals, wantStray)
	}
	// Constructor arguments are not stray, and a subsumed site is excluded.
	if strings.Join(inventory.Sites, ",") != "fixture.go:21" {
		t.Fatalf("sites = %v, want [fixture.go:21]", inventory.Sites)
	}
}
