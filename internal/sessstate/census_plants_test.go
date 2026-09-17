package sessstate

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/invcore"
)

type sessstateAliasPlant struct {
	name        string
	source      string
	wantFailure string
}

func sessstateAliasPlants() []sessstateAliasPlant {
	environPath := `"github.com/relux-works/agent-session-manager/internal/environ"`
	scalar := `"github.com/relux-works/agent-session-manager/internal/scalar"`
	return []sessstateAliasPlant{
		{
			name:        "direct local call is clean",
			source:      "package plant\nfunc run() error {\n\treturn refuse(ErrInvalidEvent, \"no\")\n}\n",
			wantFailure: "",
		},
		{
			name:        "import alias direct call resolves by path",
			source:      "package plant\nimport env " + environPath + "\nfunc run(raw []byte) {\n\t_, _ = env.DecodeStrictObject(raw)\n}\n",
			wantFailure: "",
		},
		{
			name:        "import alias var binding fails",
			source:      "package plant\nimport env " + environPath + "\nfunc run(raw []byte) {\n\tdecode := env.DecodeStrictObject\n\t_, _ = decode(raw)\n}\n",
			wantFailure: "outside direct-call position",
		},
		{
			name:        "local var binding fails",
			source:      "package plant\nfunc run() error {\n\tdeny := refuse\n\treturn deny(ErrInvalidEvent, \"no\")\n}\n",
			wantFailure: "outside direct-call position",
		},
		{
			name:        "dot import of a watched path fails closed",
			source:      "package plant\nimport . " + scalar + "\nfunc run(value string) {\n\t_, _ = ParseDigest(value)\n}\n",
			wantFailure: "dot-imports watched path",
		},
		{
			name:        "shadowed funnel name fails closed",
			source:      "package plant\nfunc run(refuse int) int {\n\treturn refuse\n}\n",
			wantFailure: "outside direct-call position",
		},
	}
}

// TestCensusAliasPlants drives every control through the production
// audit spec and requires the ledgered outcome, in both the passing
// and the failing direction. A plant that stops reporting fails here
// by construction.
func TestCensusAliasPlants(t *testing.T) {
	spec := sessstateConstructorSpec()
	for _, plant := range sessstateAliasPlants() {
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

// TestCensusPlantHarnessFailsClosed proves the plant harness itself is
// fail-closed: an unparseable control is a harness failure, never
// clean.
func TestCensusPlantHarnessFailsClosed(t *testing.T) {
	_, _, failure := invcore.ParseSource("plant.go", []byte("package plant\nfunc broken( {\n"))
	if failure == "" {
		t.Fatal("unparseable control parsed clean; the harness would pass a blind plant")
	}
}

// TestCensusDerivationFailsClosed plants every derivation failure
// shape against the production derive funcs: a call the derivation
// cannot classify fails instead of passing silently.
func TestCensusDerivationFailsClosed(t *testing.T) {
	derive := func(t *testing.T, source string) []string {
		t.Helper()
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("control does not parse: %s", failure)
		}
		_, _, failures := deriveSessstateRefusalSites([]invcore.ProductionFile{{Name: "plant.go", Syntax: syntax}}, fileSet)
		return failures
	}
	cases := []struct {
		name   string
		source string
	}{
		{"no sentinel", "package plant\nfunc run() error {\n\treturn refuse()\n}\n"},
		{"non-ident sentinel", "package plant\nfunc run() error {\n\treturn refuse(\"no\", \"x\")\n}\n"},
		{"unknown sentinel", "package plant\nfunc run() error {\n\treturn refuse(ErrSomethingElse, \"x\")\n}\n"},
		{"multiline call", "package plant\nfunc run() error {\n\treturn refuse(ErrInvalidEvent,\n\t\t\"x\")\n}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if failures := derive(t, tc.source); len(failures) == 0 {
				t.Fatalf("plant %q derived clean; the derivation is blind to it", tc.name)
			}
		})
	}
	t.Run("shared line", func(t *testing.T) {
		source := "package plant\nfunc run() error {\n\t_ = refuse(ErrInvalidEvent, \"a\"); _ = refuse(ErrInvalidEvent, \"b\")\n\treturn nil\n}\n"
		if failures := derive(t, source); len(failures) == 0 {
			t.Fatal("two refuse calls sharing one line derived clean")
		}
	})
	t.Run("unrouted sentinel wrap", func(t *testing.T) {
		source := "package plant\nimport \"fmt\"\nvar errShaped = fmt.Errorf(\"%w\", ErrInvalidEvent)\n"
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("control does not parse: %s", failure)
		}
		failures := auditSessstateUnrouted([]invcore.ProductionFile{{Name: "plant.go", Syntax: syntax}}, fileSet)
		if len(failures) == 0 {
			t.Fatal("sentinel wrapped with %w outside refuse audited clean")
		}
	})
	t.Run("state conversion over variable", func(t *testing.T) {
		source := "package plant\ntype State string\nfunc convert(s string) State {\n\treturn State(s)\n}\n"
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("control does not parse: %s", failure)
		}
		_, _, failures := deriveStateSpellings([]invcore.ProductionFile{{Name: "plant.go", Syntax: syntax}}, fileSet)
		if len(failures) == 0 {
			t.Fatal("State() conversion over a variable derived clean")
		}
	})
	t.Run("new state spelling is unregistered", func(t *testing.T) {
		source := "package plant\ntype State string\nconst StateQuiesced State = \"quiesced\"\n"
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("control does not parse: %s", failure)
		}
		derived, _, failures := deriveStateSpellings([]invcore.ProductionFile{{Name: "plant.go", Syntax: syntax}}, fileSet)
		if len(failures) != 0 {
			t.Fatalf("plant derivation failed: %v", failures)
		}
		rows := map[string]struct{}{}
		for spelling := range stateRows() {
			rows[spelling] = struct{}{}
		}
		unregistered, _, _ := invcore.DiffSets(derived, rows)
		if len(unregistered) == 0 {
			t.Fatal("a new State spelling diffed clean against the roster")
		}
	})
	t.Run("second switch dispatch site is unregistered", func(t *testing.T) {
		// PA shape, kept as a permanent control: a fresh-spelling
		// handler in a second production function must surface in
		// the derived set so the roster check rejects it.
		source := "package plant\ntype Event struct{ Type string }\nfunc effectExtra(event Event, what string) string {\n\tswitch event.Type {\n\tcase \"session.reaped\":\n\t\treturn what\n\t}\n\treturn \"\"\n}\n"
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("control does not parse: %s", failure)
		}
		derived, failures := deriveHandledEventTypes([]invcore.ProductionFile{{Name: "plant.go", Syntax: syntax}}, fileSet)
		if len(failures) == 0 {
			t.Fatal("a second event-type dispatch site derived clean")
		}
		if _, ok := derived["session.reaped"]; !ok {
			t.Fatal("the second-site spelling never entered the derived set")
		}
		rows := map[string]struct{}{}
		for eventType := range eventRows() {
			rows[eventType] = struct{}{}
		}
		unregistered, _, _ := invcore.DiffSets(derived, rows)
		if len(unregistered) == 0 {
			t.Fatal("a second-site spelling diffed clean against the roster")
		}
	})
	t.Run("if-compared spelling is unregistered", func(t *testing.T) {
		// A different shape from PA: no second switch, just an
		// == comparison against event type in a differently named
		// function. The denominator collects it anyway, so the
		// roster check still rejects the fresh spelling.
		source := "package plant\ntype Event struct{ Type string }\nfunc routeExtra(event Event, what string) string {\n\tif event.Type == \"session.reaped\" {\n\t\treturn what\n\t}\n\treturn \"\"\n}\n"
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("control does not parse: %s", failure)
		}
		derived, _ := deriveHandledEventTypes([]invcore.ProductionFile{{Name: "plant.go", Syntax: syntax}}, fileSet)
		if _, ok := derived["session.reaped"]; !ok {
			t.Fatal("the if-compared spelling never entered the derived set")
		}
		rows := map[string]struct{}{}
		for eventType := range eventRows() {
			rows[eventType] = struct{}{}
		}
		unregistered, _, _ := invcore.DiffSets(derived, rows)
		if len(unregistered) == 0 {
			t.Fatal("an if-compared spelling diffed clean against the roster")
		}
	})
	t.Run("orphan row", func(t *testing.T) {
		rows := map[string]struct{}{"creating": {}, "no-such-state": {}}
		derived := map[string]struct{}{"creating": {}}
		_, orphaned, _ := invcore.DiffSets(derived, rows)
		if len(orphaned) == 0 {
			t.Fatal("an orphan row diffed clean")
		}
	})
	t.Run("event-type case without literal", func(t *testing.T) {
		source := "package plant\ntype Event struct{ Type string }\nfunc effectExtra(event Event, what string) string {\n\tswitch event.Type {\n\tcase what:\n\t\treturn what\n\t}\n\treturn \"\"\n}\n"
		syntax, fileSet, failure := invcore.ParseSource("plant.go", []byte(source))
		if failure != "" {
			t.Fatalf("control does not parse: %s", failure)
		}
		// The production derivation collects every switch over a
		// `.Type` selector: the plant reuses that shape so the
		// rule under test fires.
		_, failures := deriveHandledEventTypes([]invcore.ProductionFile{{Name: "plant.go", Syntax: syntax}}, fileSet)
		if len(failures) == 0 {
			t.Fatal("a non-literal event-type case derived clean")
		}
	})
}

// These plants remain wired into Reduce -> apply -> effect. They are not
// parse-only approximations. No test vector adds their fresh event spelling
// to the behavioral suite: the ownership gate must reject the entire escape.
func eventOwnershipLivePlants() []struct {
	id, args, helper string
	bad, neutral     bool
} {
	return []struct {
		id, args, helper string
		bad, neutral     bool
	}{
		{"PA", "event, name", `func (fold *chainFold) reviewExtra(event Event, name string) error { switch event.Type { case "session.reaped": return fold.step(StateIdle,name) }; return nil }`, false, false},
		{"PB", "event, name", `func (fold *chainFold) reviewExtra(event Event, name string) error { kind := event.Type; switch kind { case "session.reaped": return fold.step(StateIdle,name) }; return nil }`, false, false},
		{"PC", "event.Type, name", `func (fold *chainFold) reviewExtra(kind string, name string) error { switch kind { case "session.reaped": return fold.step(StateIdle,name) }; return nil }`, false, false},
		{"PD", "event, name", `func (fold *chainFold) reviewExtra(event Event, name string) error { kind := &event.Type; if *kind == "session.reaped" { return fold.step(StateIdle,name) }; return nil }`, false, false},
		{"XN", "", "", false, true},
		{"XK", "", "", true, false},
	}
}

// Package tests are partitioned by their source ownership, then selected by
// exact anchored names. No census test is called behavioral merely because
// test.run is nonempty. TestMain's runtime census is excluded from all three
// layers; the ordinary unfiltered package command still exercises it.
func sessstateGateSelectors(t *testing.T, dir string) map[string]string {
	t.Helper()
	layers := map[string][]string{"behavior": {}, "census": {}, "audit": {}}
	paths, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		syntax, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range syntax.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Name.Name == "TestMain" || fn.Name.Name == "TestCensusLiveEventOwnershipPlants" {
				continue
			}
			layer := "behavior"
			if strings.HasPrefix(filepath.Base(path), "census") {
				layer = "census"
			}
			if fn.Name.Name == "TestSessstateStaticAudit" {
				layer = "audit"
			}
			layers[layer] = append(layers[layer], regexp.QuoteMeta(fn.Name.Name))
		}
	}
	selectors := map[string]string{}
	seen := map[string]bool{}
	for layer, names := range layers {
		if len(names) == 0 {
			t.Fatalf("empty %s selection", layer)
		}
		sort.Strings(names)
		for _, name := range names {
			if seen[name] {
				t.Fatalf("overlapping test selection: %s", name)
			}
			seen[name] = true
		}
		selectors[layer] = "^(" + strings.Join(names, "|") + ")$"
		t.Logf("%s selector (%d tests): %s", layer, len(names), selectors[layer])
	}
	return selectors
}

func TestCensusLiveEventOwnershipPlants(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	copyRoot := t.TempDir()
	copyFile := func(from, to string) {
		t.Helper()
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		copyFile(filepath.Join(root, name), filepath.Join(copyRoot, name))
	}
	if err := os.Mkdir(filepath.Join(copyRoot, "internal"), 0700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "sessstate" {
			continue
		}
		if err := os.Symlink(filepath.Join(root, "internal", entry.Name()), filepath.Join(copyRoot, "internal", entry.Name())); err != nil {
			t.Fatal(err)
		}
	}
	pkg := filepath.Join(copyRoot, "internal", "sessstate")
	if err := os.Mkdir(pkg, 0700); err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		copyFile(path, filepath.Join(pkg, path))
	}
	selectors := sessstateGateSelectors(t, pkg)
	run := func(t *testing.T, label string, args ...string) (int, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = copyRoot
		output, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			t.Fatalf("%s timeout: %v", label, ctx.Err())
		}
		code := 0
		if err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			code = exit.ExitCode()
		}
		t.Logf("%s: go %s; exit=%d\n%s", label, strings.Join(args, " "), code, output)
		return code, string(output)
	}
	// Prove the copied candidate and all isolated layers green first.
	for _, layer := range []string{"behavior", "census", "audit"} {
		if code, _ := run(t, "baseline-"+layer, "test", "./internal/sessstate", "-count=1", "-run", selectors[layer]); code != 0 {
			t.Fatal("baseline red")
		}
	}
	original, err := os.ReadFile(filepath.Join(pkg, "sessstate.go"))
	if err != nil {
		t.Fatal(err)
	}
	wire := "\tdefault:\n\t\treturn nil\n\t}\n}\n\n// effectCreated opens the bootstrap"
	if strings.Count(string(original), wire) != 1 {
		t.Fatal("plant wire not unique")
	}
	for _, plant := range eventOwnershipLivePlants() {
		t.Run(plant.id, func(t *testing.T) {
			changed := string(original)
			if plant.neutral {
				old := "func (fold *chainFold) tailEventID() string { return fold.tailID }"
				if strings.Count(changed, old) != 1 {
					t.Fatal("neutral wire not unique")
				}
				changed = strings.Replace(changed, old, `func (fold *chainFold) tailEventID() string { return "" + fold.tailID }`, 1)
			} else {
				replacement := "return fold.reviewExtra(" + plant.args + ")"
				if plant.bad {
					replacement = "return fold.step(StateIdle, name)"
				}
				changed = strings.Replace(changed, wire, strings.Replace(wire, "return nil", replacement, 1), 1)
			}
			if err := os.WriteFile(filepath.Join(pkg, "sessstate.go"), []byte(changed), 0600); err != nil {
				t.Fatal(err)
			}
			if plant.helper != "" {
				if err := os.WriteFile(filepath.Join(pkg, "plant.go"), []byte("package sessstate\n"+plant.helper+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			defer func() {
				if err := os.WriteFile(filepath.Join(pkg, "sessstate.go"), original, 0600); err != nil {
					t.Fatal(err)
				}
				if plant.helper != "" {
					if err := os.Remove(filepath.Join(pkg, "plant.go")); err != nil {
						t.Fatal(err)
					}
				}
			}()
			if code, _ := run(t, "build", "build", "./internal/sessstate"); code != 0 {
				t.Fatal("control must compile; not a kill")
			}
			results := map[string]int{}
			outputs := map[string]string{}
			for _, layer := range []string{"behavior", "census", "audit"} {
				results[layer], outputs[layer] = run(t, layer, "test", "./internal/sessstate", "-count=1", "-run", selectors[layer])
			}
			if plant.neutral {
				if results["behavior"] != 0 || results["census"] != 0 || results["audit"] != 0 {
					t.Fatal("neutral must survive every layer")
				}
			} else if plant.bad {
				if results["behavior"] == 0 || !strings.Contains(outputs["behavior"], "--- FAIL: TestReduceTreatsUnknownV1TypeAsInert") {
					t.Fatal("known-bad control lacks named behavioral kill")
				}
			} else {
				if results["behavior"] != 0 || results["audit"] != 0 {
					t.Fatal("escape control classification drifted: expected census-only")
				}
				if results["census"] == 0 || !strings.Contains(outputs["census"], "--- FAIL: TestEventTypeUsesAreOwned") {
					t.Fatal("live event-type escape survived the ownership gate")
				}
			}
			if plant.helper != "" {
				// The probe is added only AFTER the delivered layers ran.
				// Its spelling comes from the planted production source,
				// and proves the control changes Reduce, not a dead helper.
				syntax, _, failure := invcore.ParseSource("plant.go", []byte("package sessstate\n"+plant.helper))
				if failure != "" {
					t.Fatal(failure)
				}
				unknown := map[string]bool{}
				ast.Inspect(syntax, func(node ast.Node) bool {
					if lit, ok := node.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						value, err := strconv.Unquote(lit.Value)
						if err != nil {
							t.Fatal(err)
						}
						if _, known := eventRows()[value]; !known {
							unknown[value] = true
						}
					}
					return true
				})
				if len(unknown) != 1 {
					t.Fatalf("unclassifiable control literals: %v", unknown)
				}
				var spelling string
				for value := range unknown {
					spelling = value
				}
				probe := fmt.Sprintf(`package sessstate
import "testing"
func TestPlantUnknownV1Effect(t *testing.T) {
 record:=decodeTestRecord(t)
 specs:=append(baseBootstrap(record), eventSpec{typ:%q,payload:map[string]any{"note":"control"}})
 got,err:=reduceSpecs(t,record,specs,Input{})
 if err!=nil {t.Fatal(err)}
 if got.State!=StateRunning {t.Fatalf("unknown type changed state: %%s",got.State)}
}`, spelling)
				probePath := filepath.Join(pkg, "plant_probe_test.go")
				if err := os.WriteFile(probePath, []byte(probe), 0600); err != nil {
					t.Fatal(err)
				}
				defer os.Remove(probePath)
				code, output := run(t, "independent-control-effect", "test", "./internal/sessstate", "-count=1", "-run", "^TestPlantUnknownV1Effect$")
				if code == 0 || !strings.Contains(output, "unknown type changed state: idle") {
					t.Fatal("live plant did not demonstrate the intended wrong transition")
				}
			}
			t.Logf("plant %s compiled; behavioral=%d census=%d audit=%d", plant.id, results["behavior"], results["census"], results["audit"])
		})
	}
}

func TestCensusEventOwnershipControls(t *testing.T) {
	files, fset := invcore.MustScanProduction(t, ".")
	for _, tc := range []struct{ name, source string }{
		{"package binding", `package sessstate; var escaped = Event{}.Type`},
		{"alias and closure", `package sessstate; type Alias = Event; func extra(event Alias) string { return func() string { return event.Type }() }`},
		{"write", `package sessstate; func extra(event *Event) { event.Type = "session.reaped" }`},
		{"unclassifiable", `package sessstate; type Broken [Event{}.Type]int`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			syntax, plantSet, failure := invcore.ParseSource("plant.go", []byte(tc.source))
			if failure != "" {
				t.Fatal(failure)
			}
			failures := eventTypeOwnershipFailures([]invcore.ProductionFile{{Name: "plant.go", Syntax: syntax}}, plantSet, map[string]struct{}{})
			if len(failures) == 0 {
				t.Fatal("unowned or unclassifiable Type use admitted")
			}
		})
	}
	rows := eventTypeUseRows()
	rows["orphan"] = struct{}{}
	if failures := eventTypeOwnershipFailures(files, fset, rows); !strings.Contains(fmt.Sprint(failures), "orphan event-type owner row: orphan") {
		t.Fatal(failures)
	}
	duplicated := append(append([]invcore.ProductionFile{}, files...), files...)
	if failures := eventTypeOwnershipFailures(duplicated, fset, eventTypeUseRows()); !strings.Contains(fmt.Sprint(failures), "duplicate event-type owner use") {
		t.Fatal(failures)
	}
}
