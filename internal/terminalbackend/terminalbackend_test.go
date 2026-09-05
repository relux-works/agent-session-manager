package terminalbackend_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	terminalbackend "github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

const (
	testImplVersion = "1.2.3"
	testProto       = "1.0.0"
	testDigest      = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	testDigestOther = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
)

func testRegistry(t *testing.T) *terminalbackend.Registry {
	t.Helper()
	registry, err := terminalbackend.New(testImplVersion, []string{testProto})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return registry
}

func testExternalRecord(id string) terminalbackend.Registration {
	return terminalbackend.Registration{
		ID:                    id,
		Kind:                  terminalbackend.KindTrustedExecutable,
		ImplementationVersion: testImplVersion,
		ProtocolVersions:      []string{testProto},
		Platforms:             []scalar.Platform{scalar.PlatformLinux},
		ExecutableDigest:      testDigest,
	}
}

func testTrustEntry(id string) terminalbackend.TrustEntry {
	return terminalbackend.TrustEntry{
		BackendID:        id,
		ExecutablePath:   "/opt/vendor/bin/ax-backend-term",
		ExecutableDigest: testDigest,
		Enabled:          true,
	}
}

func mustRegisterExternal(t *testing.T, registry *terminalbackend.Registry, id string) {
	t.Helper()
	if err := registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry(id), testExternalRecord(id)); err != nil {
		t.Fatalf("RegisterExternal(%q) error = %v", id, err)
	}
}

// TestWireCodesAreCatalogued proves the registry advertises no error code the
// pinned contract vocabulary does not define.
func TestWireCodesAreCatalogued(t *testing.T) {
	t.Parallel()

	known := map[string]bool{}
	for _, entry := range catalog.Current().Errors {
		known[string(entry.Code)] = true
	}
	for _, code := range []string{
		"terminal_backend_not_found",
		"terminal_backend_ambiguous",
		"terminal_backend_untrusted",
		"terminal_backend_implementation_drift",
		"terminal_backend_restore_mismatch",
		"terminal_backend_stale_generation",
		"terminal_backend_manifest_probe_mismatch",
		"terminal_backend_capability_unproven",
		"terminal_backend_integrity_failure",
		"terminal_backend_protocol_error",
		"terminal_backend_unauthorized",
		"terminal_backend_unavailable",
		"local_precondition_failed",
		"idempotency_mismatch",
		"incompatible_schema",
	} {
		if !known[code] {
			t.Errorf("wire code %q is not in the pinned catalog error vocabulary", code)
		}
	}
}

// TestParseIDAdmitsCanonicalIdentities proves the gate is reachable: the two
// canonical built-ins, minimal and maximal bounds, and vendor namespaces.
func TestParseIDAdmitsCanonicalIdentities(t *testing.T) {
	t.Parallel()

	accepted := []string{
		terminalbackend.BuiltinTmux,
		terminalbackend.BuiltinConpty,
		"a",
		"tmux",
		"ax-tmux",
		"vendor.backend-1",
		"a0.b1-c2.d3",
		"example.terminal-backend",
		"a" + strings.Repeat("0", 127),
	}
	for _, id := range accepted {
		if _, err := terminalbackend.ParseID(id); err != nil {
			t.Errorf("ParseID(%q) error = %v, want admission", id, err)
		}
	}
}

// TestParseIDRefusesWidenedGrammar narrows the gate instead of only deleting
// it: every case widens exactly one part of the declared character class,
// separator rule, or byte bound, so admitting any of them means the pattern
// was widened rather than merely removed.
func TestParseIDRefusesWidenedGrammar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"one byte past the bound", strings.Repeat("a", 129)},
		{"far past the bound", strings.Repeat("a", 5000)},
		{"uppercase", "AX.TMUX"},
		{"mixed case", "Ax.Tmux"},
		{"leading digit", "1ax"},
		{"underscore separator", "ax_tmux"},
		{"space", "ax tmux"},
		{"slash", "../../etc/passwd"},
		{"colon", "urn:ax:schema:blob"},
		{"at sign", "ax@tmux"},
		{"non-ASCII", "ax.tmux界"},
		{"leading dash", "-leading-dash"},
		{"leading dot", ".ax"},
		{"trailing dot", "ax."},
		{"trailing dash", "ax-"},
		{"consecutive dots", "ax..tmux"},
		{"consecutive mixed separators", "ax.-tmux"},
		{"trailing newline", "ax.tmux\n"},
		{"embedded newline", "ax\n.tmux"},
		{"embedded NUL", "ax\x00tmux"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := terminalbackend.ParseID(testCase.value); !terminalbackend.IsNotFound(err) {
				t.Errorf("ParseID(%q) error = %v, want terminal_backend_not_found", testCase.value, err)
			}
		})
	}
}

// TestParseIDRefusesReservedNamespace proves the ax. reservation is a separate
// rule from the grammar: these values match the character pattern yet must
// still fail because only the two canonical built-ins may use the namespace.
func TestParseIDRefusesReservedNamespace(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"ax.evil", "ax.tmux.evil", "ax.0", "ax.a"} {
		if _, err := terminalbackend.ParseID(id); !terminalbackend.IsNotFound(err) {
			t.Errorf("ParseID(%q) error = %v, want terminal_backend_not_found", id, err)
		}
	}
}

// TestNewAdmitsCanonicalBuiltins checks the built-in half of registration:
// identities, kinds, null digests, version tuples, and platform lanes.
func TestNewAdmitsCanonicalBuiltins(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	if got := registry.IDs(); len(got) != 2 || got[0] != terminalbackend.BuiltinConpty || got[1] != terminalbackend.BuiltinTmux {
		t.Fatalf("IDs() = %v, want [ax.conpty ax.tmux]", got)
	}
	tmux, err := registry.Resolve(terminalbackend.BuiltinTmux)
	if err != nil {
		t.Fatalf("Resolve(ax.tmux) error = %v", err)
	}
	if tmux.Kind != terminalbackend.KindBuiltinGo || tmux.ExecutableDigest != "" {
		t.Errorf("ax.tmux = kind %q digest %q, want builtin_go with null digest", tmux.Kind, tmux.ExecutableDigest)
	}
	if tmux.ImplementationVersion != testImplVersion || len(tmux.ProtocolVersions) != 1 || tmux.ProtocolVersions[0] != testProto {
		t.Errorf("ax.tmux version tuple = %q/%v, want %q/[%q]", tmux.ImplementationVersion, tmux.ProtocolVersions, testImplVersion, testProto)
	}
	wantPlatforms := []scalar.Platform{scalar.PlatformLinux, scalar.PlatformMacOS, scalar.PlatformWSL2}
	if len(tmux.Platforms) != len(wantPlatforms) {
		t.Fatalf("ax.tmux platforms = %v, want %v", tmux.Platforms, wantPlatforms)
	}
	for i := range wantPlatforms {
		if tmux.Platforms[i] != wantPlatforms[i] {
			t.Fatalf("ax.tmux platforms = %v, want %v", tmux.Platforms, wantPlatforms)
		}
	}
	conpty, err := registry.Resolve(terminalbackend.BuiltinConpty)
	if err != nil {
		t.Fatalf("Resolve(ax.conpty) error = %v", err)
	}
	if len(conpty.Platforms) != 1 || conpty.Platforms[0] != scalar.PlatformWindows {
		t.Errorf("ax.conpty platforms = %v, want [windows]", conpty.Platforms)
	}
}

// TestNewRefusesBadVersionTuples narrows version admission: each case breaks
// exactly one tuple rule.
func TestNewRefusesBadVersionTuples(t *testing.T) {
	t.Parallel()

	// distinctMajorOneVersions builds count distinct Terminal Backend Protocol
	// major-1 versions in lexical order: 1.<minor>.<patch> with minor=i/10 and
	// patch=i%10 stays lexically sorted and unique below 100 members, so the
	// count bound is the only rule that can refuse the full list.
	distinctMajorOneVersions := func(count int) []string {
		versions := make([]string, count)
		for i := range versions {
			versions[i] = "1." + strconv.Itoa(i/10) + "." + strconv.Itoa(i%10)
		}
		return versions
	}
	cases := []struct {
		name      string
		impl      string
		protocols []string
	}{
		{"non-semver implementation", "v1", []string{testProto}},
		{"empty implementation", "", []string{testProto}},
		{"empty protocol list", testImplVersion, nil},
		{"protocol major 2", testImplVersion, []string{"2.0.0"}},
		{"protocol major 0", testImplVersion, []string{"0.9.0"}},
		{"non-semver protocol", testImplVersion, []string{"one"}},
		{"duplicate protocols", testImplVersion, []string{testProto, testProto}},
		// 33 distinct sorted major-1 members: sorted-unique, semver, and
		// major rules all pass, so only the [1..32] count bound can refuse.
		// (33 identical entries would be refused by sorted-unique instead,
		// passing for the wrong reason.)
		{"over 32 protocols", testImplVersion, distinctMajorOneVersions(33)},
	}
	// 32 distinct members must still be admitted: the bound is 32, not 31.
	// New sorts its copy, so construct the list in reverse to prove the
	// admission does not depend on caller order.
	reversed := distinctMajorOneVersions(32)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	if _, err := terminalbackend.New(testImplVersion, reversed); err != nil {
		t.Errorf("New(32 distinct protocols) error = %v, want admission", err)
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := terminalbackend.New(testCase.impl, testCase.protocols); !terminalbackend.IsDrift(err) {
				t.Errorf("New(%q, %v) error = %v, want terminal_backend_implementation_drift", testCase.impl, testCase.protocols, err)
			}
		})
	}
}

// TestRegisterExternalRoundTrip drives the production external-admission
// entry point: trust plus observed record in, resolvable registration out.
func TestRegisterExternalRoundTrip(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	mustRegisterExternal(t, registry, "com.example.term")
	record, err := registry.Resolve("com.example.term")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if record.Kind != terminalbackend.KindTrustedExecutable || record.ExecutableDigest != testDigest {
		t.Errorf("resolved record = %+v, want trusted_executable with pinned digest", record)
	}
	if got := registry.IDs(); len(got) != 3 {
		t.Errorf("IDs() = %v, want 3 admitted identities", got)
	}
}

// TestRegisterExternalAdmitsFullPlatformSet proves the platforms bound in
// the narrowing direction: four is the last admitted value, so narrowing the
// >4 bound to >3 refuses this record. The widening direction (five members
// refused) is pinned by TestRegisterExternalRefusesPlatformViolations; a
// bound is proven only when both the last admitted value and the first
// refused one are driven.
func TestRegisterExternalAdmitsFullPlatformSet(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	record := testExternalRecord("com.example.term")
	record.Platforms = []scalar.Platform{
		scalar.PlatformLinux, scalar.PlatformMacOS,
		scalar.PlatformWindows, scalar.PlatformWSL2,
	}
	if err := registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record); err != nil {
		t.Fatalf("RegisterExternal(four platforms) error = %v, want admission", err)
	}
	admitted, err := registry.Resolve("com.example.term")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(admitted.Platforms) != 4 {
		t.Fatalf("admitted platforms = %v, want all four", admitted.Platforms)
	}
	for i, platform := range []scalar.Platform{
		scalar.PlatformLinux, scalar.PlatformMacOS,
		scalar.PlatformWindows, scalar.PlatformWSL2,
	} {
		if admitted.Platforms[i] != platform {
			t.Fatalf("admitted platforms = %v, want [linux macos windows wsl2]", admitted.Platforms)
		}
	}
}

// TestRegisterExternalRefusals proves each trust check rejects what it must:
// every case breaks exactly one admission rule.
func TestRegisterExternalRefusals(t *testing.T) {
	t.Parallel()

	platform := scalar.PlatformLinux
	cases := []struct {
		name     string
		mutate   func(entry *terminalbackend.TrustEntry, record *terminalbackend.Registration)
		platform scalar.Platform
		check    func(error) bool
		code     string
	}{
		{"disabled trust", func(entry *terminalbackend.TrustEntry, _ *terminalbackend.Registration) {
			entry.Enabled = false
		}, platform, terminalbackend.IsUntrusted, "terminal_backend_untrusted"},
		{"relative path is PATH-only discovery", func(entry *terminalbackend.TrustEntry, _ *terminalbackend.Registration) {
			entry.ExecutablePath = "bin/ax-backend-term"
		}, platform, terminalbackend.IsUntrusted, "terminal_backend_untrusted"},
		{"malformed trust digest", func(entry *terminalbackend.TrustEntry, _ *terminalbackend.Registration) {
			entry.ExecutableDigest = "not-a-digest"
		}, platform, terminalbackend.IsUntrusted, "terminal_backend_untrusted"},
		{"observed digest substitution", func(_ *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			record.ExecutableDigest = testDigestOther
		}, platform, terminalbackend.IsUntrusted, "terminal_backend_untrusted"},
		{"trust identity mismatch", func(_ *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			record.ID = "com.example.other"
			record.ExecutableDigest = testDigest
		}, platform, terminalbackend.IsAmbiguous, "terminal_backend_ambiguous"},
		{"built-in kind for external path", func(_ *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			record.Kind = terminalbackend.KindBuiltinGo
			record.ExecutableDigest = ""
		}, platform, terminalbackend.IsUntrusted, "terminal_backend_untrusted"},
		{"reserved namespace external", func(entry *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			entry.BackendID = "ax.evil"
			record.ID = "ax.evil"
		}, platform, terminalbackend.IsNotFound, "terminal_backend_not_found"},
		{"built-in ID claimed by external", func(entry *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			entry.BackendID = terminalbackend.BuiltinTmux
			record.ID = terminalbackend.BuiltinTmux
			record.Kind = terminalbackend.KindTrustedExecutable
			record.ExecutableDigest = testDigest
		}, platform, terminalbackend.IsAmbiguous, "terminal_backend_ambiguous"},
		{"malformed external ID", func(entry *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			entry.BackendID = "INVALID_ID"
			record.ID = "INVALID_ID"
		}, platform, terminalbackend.IsNotFound, "terminal_backend_not_found"},
		{"non-semver external implementation", func(_ *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			record.ImplementationVersion = "v1"
		}, platform, terminalbackend.IsDrift, "terminal_backend_implementation_drift"},
		{"major-2 external protocol", func(_ *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			record.ProtocolVersions = []string{"2.0.0"}
		}, platform, terminalbackend.IsDrift, "terminal_backend_implementation_drift"},
		{"missing external digest", func(_ *terminalbackend.TrustEntry, record *terminalbackend.Registration) {
			record.ExecutableDigest = ""
		}, platform, terminalbackend.IsUntrusted, "terminal_backend_untrusted"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			registry := testRegistry(t)
			entry := testTrustEntry("com.example.term")
			record := testExternalRecord("com.example.term")
			testCase.mutate(&entry, &record)
			// The reserved-namespace trust case fails inside entry validation
			// before identity comparison; keep both halves consistent there.
			err := registry.RegisterExternal(testCase.platform, entry, record)
			if !testCase.check(err) {
				t.Errorf("RegisterExternal() error = %v, want %s", err, testCase.code)
			}
			if _, resolveErr := registry.Resolve("com.example.term"); !terminalbackend.IsNotFound(resolveErr) {
				t.Errorf("Resolve() after refused registration = %v, want terminal_backend_not_found", resolveErr)
			}
		})
	}
}

// TestDuplicateRegistrationIsRefused proves registration is not idempotent: an
// identical second registration fails with terminal_backend_ambiguous instead
// of silently succeeding, so a retried admission cannot mask a drifted one.
func TestDuplicateRegistrationIsRefused(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	mustRegisterExternal(t, registry, "com.example.term")
	before, err := registry.Resolve("com.example.term")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	err = registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), testExternalRecord("com.example.term"))
	if !terminalbackend.IsAmbiguous(err) {
		t.Fatalf("second RegisterExternal() error = %v, want terminal_backend_ambiguous", err)
	}
	after, err := registry.Resolve("com.example.term")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Errorf("record changed across refused duplicate: before %+v after %+v", before, after)
	}
}

// TestDriftIsRefused narrows the drift gate: each case changes exactly one
// member of an admitted record, and the admitted record must survive every
// refused attempt unchanged.
func TestDriftIsRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(record *terminalbackend.Registration)
	}{
		{"implementation version", func(record *terminalbackend.Registration) { record.ImplementationVersion = "1.2.4" }},
		{"executable digest", func(record *terminalbackend.Registration) { record.ExecutableDigest = testDigestOther }},
		{"protocol list", func(record *terminalbackend.Registration) { record.ProtocolVersions = []string{"1.0.0", "1.1.0"} }},
		// Same-length protocol drift exercises the member comparison, not
		// the length guard: the length-changing case above passes even when
		// the element loop is deleted.
		{"protocol member same length", func(record *terminalbackend.Registration) { record.ProtocolVersions = []string{"1.1.0"} }},
		{"kind", func(record *terminalbackend.Registration) {
			record.Kind = terminalbackend.KindLocalProgram
		}},
		{"platforms", func(record *terminalbackend.Registration) {
			record.Platforms = []scalar.Platform{scalar.PlatformLinux, scalar.PlatformMacOS}
		}},
		// Same-length platform drift, for the same reason: [linux] and
		// [macos] are both valid singletons, so only the member comparison
		// can refuse this candidate.
		{"platform member same length", func(record *terminalbackend.Registration) {
			record.Platforms = []scalar.Platform{scalar.PlatformMacOS}
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			registry := testRegistry(t)
			mustRegisterExternal(t, registry, "com.example.term")
			before, err := registry.Resolve("com.example.term")
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			drifted := testExternalRecord("com.example.term")
			testCase.mutate(&drifted)
			// The digest-drift case must keep trust and observation bound so
			// the refusal proves drift detection rather than trust admission.
			entry := testTrustEntry("com.example.term")
			if testCase.name == "executable digest" {
				entry.ExecutableDigest = testDigestOther
			}
			if err := registry.RegisterExternal(scalar.PlatformLinux, entry, drifted); !terminalbackend.IsDrift(err) {
				t.Fatalf("RegisterExternal(drifted) error = %v, want terminal_backend_implementation_drift", err)
			}
			after, err := registry.Resolve("com.example.term")
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if !reflect.DeepEqual(after, before) {
				t.Errorf("admitted record changed across refused drift: before %+v after %+v", before, after)
			}
		})
	}
}

// TestRegistryCopiesSlicesAcrossItsBoundary proves validation survives the
// package's own accessors: mutating a resolved record, or a caller-side
// record after admission, must not change what the registry holds, and the
// two built-ins must not share one protocol backing array. Without the
// copies, an in-place mutation converts real drift into a benign duplicate:
// the drifted re-registration reports terminal_backend_ambiguous instead of
// terminal_backend_implementation_drift and the registry adopts the change.
func TestRegistryCopiesSlicesAcrossItsBoundary(t *testing.T) {
	t.Parallel()

	t.Run("resolve does not hand out interior state", func(t *testing.T) {
		t.Parallel()
		registry := testRegistry(t)
		resolved, err := registry.Resolve(terminalbackend.BuiltinTmux)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		resolved.Platforms[0] = scalar.PlatformWindows
		resolved.ProtocolVersions[0] = "9.9.9"
		again, err := registry.Resolve(terminalbackend.BuiltinTmux)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if again.Platforms[0] != scalar.PlatformLinux || again.ProtocolVersions[0] != testProto {
			t.Errorf("registry mutated through Resolve() accessor: platforms=%v protocols=%v",
				again.Platforms, again.ProtocolVersions)
		}
	})

	t.Run("registration does not retain caller slices", func(t *testing.T) {
		t.Parallel()
		registry := testRegistry(t)
		record := testExternalRecord("com.example.term")
		if err := registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record); err != nil {
			t.Fatalf("RegisterExternal() error = %v", err)
		}
		record.Platforms[0] = scalar.PlatformWindows
		record.ProtocolVersions[0] = "1.99.0"
		admitted, err := registry.Resolve("com.example.term")
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if admitted.Platforms[0] != scalar.PlatformLinux || admitted.ProtocolVersions[0] != testProto {
			t.Errorf("registry mutated through caller record: platforms=%v protocols=%v",
				admitted.Platforms, admitted.ProtocolVersions)
		}
	})

	t.Run("in-place mutation no longer converts drift to duplicate", func(t *testing.T) {
		t.Parallel()
		registry := testRegistry(t)
		mustRegisterExternal(t, registry, "com.example.term")
		mutated, err := registry.Resolve("com.example.term")
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		mutated.ProtocolVersions[0] = "1.4.0"
		err = registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), mutated)
		if !terminalbackend.IsDrift(err) {
			t.Fatalf("RegisterExternal(mutated) error = %v, want terminal_backend_implementation_drift", err)
		}
		admitted, err := registry.Resolve("com.example.term")
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if len(admitted.ProtocolVersions) != 1 || admitted.ProtocolVersions[0] != testProto {
			t.Errorf("registry adopted the mutation: protocols=%v, want [%q]",
				admitted.ProtocolVersions, testProto)
		}
	})
}

// refusalDetail extracts the static clause of a registry refusal. The code
// alone cannot discriminate precedence between two guards reporting the same
// wire code, so the bound-precedence pins below assert the clause.
func refusalDetail(t *testing.T, err error) string {
	t.Helper()
	var refusal *terminalbackend.Error
	if !errors.As(err, &refusal) {
		t.Fatalf("error %v is not a *terminalbackend.Error", err)
	}
	return refusal.Detail
}

// TestRegisterExternalRefusesPlatformViolations narrows the platform-set
// gate member by member: each case breaks exactly one of the sorted-unique,
// non-empty, vocabulary, or upper-bound rules at the production admission
// entry point.
func TestRegisterExternalRefusesPlatformViolations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		platforms []scalar.Platform
		detail    string
	}{
		{"duplicate platform", []scalar.Platform{scalar.PlatformLinux, scalar.PlatformLinux}, "platforms sorted unique"},
		{"empty platform set", nil, "platforms bound"},
		{"unknown platform member", []scalar.Platform{scalar.PlatformLinux, scalar.Platform("plan9")}, "platforms vocabulary"},
		// Five members from a four-member vocabulary must contain a repeat
		// or an outsider, so only the bound check fires first: assert the
		// clause to prove the length arm ran rather than a later rule.
		{"five platform members", []scalar.Platform{
			scalar.PlatformLinux, scalar.PlatformLinux,
			scalar.PlatformMacOS, scalar.PlatformWindows, scalar.PlatformWSL2,
		}, "platforms bound"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			registry := testRegistry(t)
			record := testExternalRecord("com.example.term")
			record.Platforms = testCase.platforms
			err := registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
			if !terminalbackend.IsNotFound(err) {
				t.Fatalf("RegisterExternal() error = %v, want terminal_backend_not_found", err)
			}
			if got := refusalDetail(t, err); got != testCase.detail {
				t.Errorf("refusal detail = %q, want %q", got, testCase.detail)
			}
			if _, resolveErr := registry.Resolve("com.example.term"); !terminalbackend.IsNotFound(resolveErr) {
				t.Errorf("Resolve() after refused registration = %v, want terminal_backend_not_found", resolveErr)
			}
		})
	}
}

// TestRegisterExternalRefusesUnknownKindAsUntrusted pins the guard order at
// the production entry point: a non-external kind (including one outside the
// closed vocabulary) is a trust failure before record validation runs. The
// closed implementation_kind vocabulary in parseKind is therefore unreachable
// through this entry point by construction, and is pinned white-box in
// internal_pin_test.go instead.
func TestRegisterExternalRefusesUnknownKindAsUntrusted(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	record := testExternalRecord("com.example.term")
	record.Kind = terminalbackend.Kind("bogus_kind")
	err := registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), record)
	if !terminalbackend.IsUntrusted(err) {
		t.Fatalf("RegisterExternal(bogus kind) error = %v, want terminal_backend_untrusted", err)
	}
}

// TestRegisterExternalRefusesReservedNamespaceDistinguishingValue drives the
// one input only the trust reserved-namespace guard refuses: ax.conpty
// passes mustParseID because it is a canonical built-in, so the predecessor
// admits it and only the ax.-prefix bar in TrustEntry.validate stands in the
// way. Narrowing that bar to entry.BackendID == BuiltinTmux lets ax.conpty
// fall through to a drift refusal against the built-in record, so asserting
// the ambiguous clause (not merely a refusal) kills the narrowed guard.
func TestRegisterExternalRefusesReservedNamespaceDistinguishingValue(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	entry := testTrustEntry(terminalbackend.BuiltinConpty)
	record := testExternalRecord(terminalbackend.BuiltinConpty)
	err := registry.RegisterExternal(scalar.PlatformLinux, entry, record)
	if !terminalbackend.IsAmbiguous(err) {
		t.Fatalf("RegisterExternal(ax.conpty) error = %v, want terminal_backend_ambiguous", err)
	}
	if got := refusalDetail(t, err); got != "external_trust reserved namespace" {
		t.Errorf("refusal detail = %q, want %q", got, "external_trust reserved namespace")
	}
}

// TestResolveUnknownIsNeverAbsence proves unknown and malformed identities
// fail closed instead of resolving to a default or an empty record.
func TestResolveUnknownIsNeverAbsence(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	for _, id := range []string{"com.example.unknown", "INVALID_ID", "", "ax.evil"} {
		if _, err := registry.Resolve(id); !terminalbackend.IsNotFound(err) {
			t.Errorf("Resolve(%q) error = %v, want terminal_backend_not_found", id, err)
		}
	}
	var nilRegistry *terminalbackend.Registry
	if _, err := nilRegistry.Resolve(terminalbackend.BuiltinTmux); !terminalbackend.IsNotFound(err) {
		t.Errorf("nil Resolve() error = %v, want terminal_backend_not_found", err)
	}
	if err := nilRegistry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), testExternalRecord("com.example.term")); !terminalbackend.IsNotFound(err) {
		t.Errorf("nil RegisterExternal() error = %v, want terminal_backend_not_found", err)
	}
}

// TestDefaultForPlatform pins the Section 6.5 platform defaults for new
// activation only: they must never be used as a restore fallback.
func TestDefaultForPlatform(t *testing.T) {
	t.Parallel()

	cases := []struct {
		platform scalar.Platform
		want     string
	}{
		{scalar.PlatformMacOS, terminalbackend.BuiltinTmux},
		{scalar.PlatformLinux, terminalbackend.BuiltinTmux},
		{scalar.PlatformWSL2, terminalbackend.BuiltinTmux},
		{scalar.PlatformWindows, terminalbackend.BuiltinConpty},
	}
	for _, testCase := range cases {
		if got, err := terminalbackend.DefaultForPlatform(testCase.platform); err != nil || got != testCase.want {
			t.Errorf("DefaultForPlatform(%q) = %q, %v; want %q", testCase.platform, got, err, testCase.want)
		}
	}
	if _, err := terminalbackend.DefaultForPlatform(scalar.Platform("plan9")); !terminalbackend.IsNotFound(err) {
		t.Errorf("DefaultForPlatform(plan9) error = %v, want terminal_backend_not_found", err)
	}
}

// TestRequireRestoreBinding proves restore uses the exact prior binding: the
// configured default is refused when it differs, and an unregistered bound ID
// cannot activate.
func TestRequireRestoreBinding(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	mustRegisterExternal(t, registry, "com.example.term")

	record, err := registry.RequireRestoreBinding("com.example.term", "com.example.term")
	if err != nil {
		t.Fatalf("RequireRestoreBinding(exact) error = %v", err)
	}
	if record.ID != "com.example.term" {
		t.Errorf("RequireRestoreBinding(exact) ID = %q, want com.example.term", record.ID)
	}
	if _, err := registry.RequireRestoreBinding("com.example.term", terminalbackend.BuiltinTmux); !terminalbackend.IsRestoreMismatch(err) {
		t.Errorf("RequireRestoreBinding(default substitution) error = %v, want terminal_backend_restore_mismatch", err)
	}
	if _, err := registry.RequireRestoreBinding("com.example.gone", "com.example.gone"); !terminalbackend.IsNotFound(err) {
		t.Errorf("RequireRestoreBinding(unknown) error = %v, want terminal_backend_not_found", err)
	}
	if _, err := registry.RequireRestoreBinding("com.example.term", "INVALID_ID"); !terminalbackend.IsNotFound(err) {
		t.Errorf("RequireRestoreBinding(malformed candidate) error = %v, want terminal_backend_not_found", err)
	}
}

// TestCheckVersionTuple narrows version admission to membership: the accepted
// protocol must be exactly one list member in major 1.
func TestCheckVersionTuple(t *testing.T) {
	t.Parallel()

	if err := terminalbackend.CheckVersionTuple("com.example.term", "1.2.3", "1.0.0", []string{"1.0.0", "1.1.0"}); err != nil {
		t.Errorf("CheckVersionTuple(member) error = %v, want admission", err)
	}
	cases := []struct {
		name  string
		impl  string
		proto string
		list  []string
	}{
		{"non-semver implementation", "v1", "1.0.0", []string{"1.0.0"}},
		{"protocol major 2", "1.2.3", "2.0.0", []string{"2.0.0"}},
		{"protocol not a member", "1.2.3", "1.1.0", []string{"1.0.0"}},
		{"protocol major 1 but unlisted", "1.2.3", "1.0.0", []string{"1.1.0"}},
		{"non-semver protocol", "1.2.3", "one", []string{"one"}},
		{"empty list", "1.2.3", "1.0.0", nil},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := terminalbackend.CheckVersionTuple("com.example.term", testCase.impl, testCase.proto, testCase.list)
			if !terminalbackend.IsDrift(err) {
				t.Errorf("CheckVersionTuple() error = %v, want terminal_backend_implementation_drift", err)
			}
		})
	}
}

// TestCheckProviderDescriptor drives the Section 7.A production rule: the
// provider rejects a descriptor that does not match the validated binding.
func TestCheckProviderDescriptor(t *testing.T) {
	t.Parallel()

	binding := terminalbackend.InstanceBinding{
		BackendID:             "com.example.term",
		ImplementationVersion: "1.2.3",
		ProtocolVersion:       "1.0.0",
		Generation:            "generation-1",
		TerminalBindingID:     testDigest,
	}
	if err := terminalbackend.CheckProviderDescriptor(binding, binding); err != nil {
		t.Errorf("CheckProviderDescriptor(match) error = %v, want admission", err)
	}

	versionDrift := binding
	versionDrift.ImplementationVersion = "1.2.4"
	if err := terminalbackend.CheckProviderDescriptor(versionDrift, binding); !terminalbackend.IsDrift(err) {
		t.Errorf("CheckProviderDescriptor(version drift) error = %v, want terminal_backend_implementation_drift", err)
	}
	protoDrift := binding
	protoDrift.ProtocolVersion = "1.1.0"
	if err := terminalbackend.CheckProviderDescriptor(protoDrift, binding); !terminalbackend.IsDrift(err) {
		t.Errorf("CheckProviderDescriptor(protocol drift) error = %v, want terminal_backend_implementation_drift", err)
	}
	idMismatch := binding
	idMismatch.BackendID = "com.example.other"
	if err := terminalbackend.CheckProviderDescriptor(idMismatch, binding); !terminalbackend.IsNotFound(err) {
		t.Errorf("CheckProviderDescriptor(ID mismatch) error = %v, want terminal_backend_not_found", err)
	}
	// The stale-generation arm lives in TestCheckProviderDescriptorGenerationMismatch.
}

// TestCheckProviderDescriptorGenerationBounds proves the Section 7.A
// string[1..256] bound itself, not the generation mismatch: every refusal
// case carries the SAME generation on both sides, so the mismatch comparison
// cannot fire and only the bound can refuse. (Pitting a mutated generation
// against a fixed one proves nothing about the bound — the inequality alone
// returns terminal_backend_stale_generation.)
func TestCheckProviderDescriptorGenerationBounds(t *testing.T) {
	t.Parallel()

	withGeneration := func(generation string) terminalbackend.InstanceBinding {
		return terminalbackend.InstanceBinding{
			BackendID:             "com.example.term",
			ImplementationVersion: "1.2.3",
			ProtocolVersion:       "1.0.0",
			Generation:            generation,
			TerminalBindingID:     testDigest,
		}
	}
	// string[1..256] bounds UTF-8 characters (SPEC.md:321), so the boundary
	// is proved twice: once in ASCII bytes where bytes and runes coincide,
	// once in two-byte runes where they do not.
	admitted := []struct {
		name       string
		generation string
	}{
		{"one byte", "g"},
		{"256 bytes", strings.Repeat("g", 256)},
		{"256 two-byte runes", strings.Repeat("é", 256)},
	}
	for _, testCase := range admitted {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			binding := withGeneration(testCase.generation)
			if err := terminalbackend.CheckProviderDescriptor(binding, binding); err != nil {
				t.Errorf("CheckProviderDescriptor(%q generation, equal) error = %v, want admission",
					testCase.name, err)
			}
		})
	}
	refused := []struct {
		name       string
		generation string
	}{
		{"empty", ""},
		{"257 bytes", strings.Repeat("g", 257)},
		{"257 two-byte runes", strings.Repeat("é", 257)},
		{"invalid utf-8", "gen\xff"},
	}
	for _, testCase := range refused {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			binding := withGeneration(testCase.generation)
			err := terminalbackend.CheckProviderDescriptor(binding, binding)
			if !terminalbackend.IsStaleGeneration(err) {
				t.Errorf("CheckProviderDescriptor(%q generation, equal) error = %v, want terminal_backend_stale_generation",
					testCase.name, err)
			}
			if got := refusalDetail(t, err); got != "backend_generation bound" {
				t.Errorf("refusal detail = %q, want %q", got, "backend_generation bound")
			}
		})
	}
}

// TestCheckProviderDescriptorGenerationMismatch keeps the mismatch arm pinned
// separately: differing valid generations refuse even though each side is
// within the bound.
func TestCheckProviderDescriptorGenerationMismatch(t *testing.T) {
	t.Parallel()

	binding := terminalbackend.InstanceBinding{
		BackendID:             "com.example.term",
		ImplementationVersion: "1.2.3",
		ProtocolVersion:       "1.0.0",
		Generation:            "generation-1",
		TerminalBindingID:     testDigest,
	}
	stale := binding
	stale.Generation = "generation-2"
	if err := terminalbackend.CheckProviderDescriptor(stale, binding); !terminalbackend.IsStaleGeneration(err) {
		t.Errorf("CheckProviderDescriptor(stale generation) error = %v, want terminal_backend_stale_generation", err)
	}
}

// TestCheckProviderDescriptorBindingDigestMismatch proves the fourth §7.A
// match dimension: a descriptor whose terminal_binding_id is malformed or
// differs from the AX-validated host-local binding is refused with the
// descriptor binding digest arm, even when ID, versions, and generation all
// match. Both arms share the clause; the shape case and the mismatch case
// below hit arm #1 and arm #2 respectively.
func TestCheckProviderDescriptorBindingDigestMismatch(t *testing.T) {
	t.Parallel()

	binding := terminalbackend.InstanceBinding{
		BackendID:             "com.example.term",
		ImplementationVersion: "1.2.3",
		ProtocolVersion:       "1.0.0",
		Generation:            "generation-1",
		TerminalBindingID:     testDigest,
	}
	cases := []struct {
		name      string
		bindingID string
	}{
		{"malformed digest", "not-a-digest"},
		{"empty digest", ""},
		{"foreign binding digest", testDigestOther},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			descriptor := binding
			descriptor.TerminalBindingID = testCase.bindingID
			err := terminalbackend.CheckProviderDescriptor(descriptor, binding)
			if !terminalbackend.IsNotFound(err) {
				t.Errorf("CheckProviderDescriptor(%s) error = %v, want terminal_backend_not_found", testCase.name, err)
			}
			if got := refusalDetail(t, err); got != "descriptor binding digest" {
				t.Errorf("refusal detail = %q, want %q", got, "descriptor binding digest")
			}
		})
	}
	// The shape arm (scalar.ParseDigest at terminalbackend.go:587) is the
	// only refusal when the malformed digest sits on BOTH sides: the
	// equality arm cannot fire on equal values, so deleting the shape arm
	// admits. The descriptor-side cases above cannot prove that — the
	// equality arm catches all of them with the same detail string.
	t.Run("malformed digest on both sides", func(t *testing.T) {
		t.Parallel()
		descriptor := binding
		descriptor.TerminalBindingID = "not-a-digest"
		bothMalformed := binding
		bothMalformed.TerminalBindingID = "not-a-digest"
		err := terminalbackend.CheckProviderDescriptor(descriptor, bothMalformed)
		if !terminalbackend.IsNotFound(err) {
			t.Errorf("CheckProviderDescriptor(malformed digest on both sides) error = %v, want terminal_backend_not_found", err)
		}
		if got := refusalDetail(t, err); got != "descriptor binding digest" {
			t.Errorf("refusal detail = %q, want %q", got, "descriptor binding digest")
		}
	})
}

// TestDigestFile binds trust to bytes: the digest of a real file verifies,
// symlinks resolve to their target, and non-files or missing paths fail
// without producing a digest.
func TestDigestFile(t *testing.T) {
	t.Parallel()

	payload := []byte("#!/bin/sh\nexec ax-backend-term \"$@\"\n")
	path := filepath.Join(t.TempDir(), "ax-backend-term")
	if err := os.WriteFile(path, payload, 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	sum := sha256.Sum256(payload)
	want := "sha256:" + hex.EncodeToString(sum[:])
	got, err := terminalbackend.DigestFile(path)
	if err != nil {
		t.Fatalf("DigestFile() error = %v", err)
	}
	if got != want {
		t.Errorf("DigestFile() = %q, want %q", got, want)
	}
	link := filepath.Join(t.TempDir(), "link-backend-term")
	if err := os.Symlink(path, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	gotLink, err := terminalbackend.DigestFile(link)
	if err != nil {
		t.Fatalf("DigestFile(symlink) error = %v", err)
	}
	if gotLink != want {
		t.Errorf("DigestFile(symlink) = %q, want %q", gotLink, want)
	}
	if _, err := terminalbackend.DigestFile(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("DigestFile(absent) error = nil, want a failed read")
	}
	if _, err := terminalbackend.DigestFile(t.TempDir()); err == nil {
		t.Error("DigestFile(directory) error = nil, want refusal for a non-regular file")
	}
	// A FIFO pins the regular-file guard itself: the directory case above
	// would still fail with the guard removed, because os.ReadFile rejects
	// directories (EISDIR) on its own. A FIFO is the documented hazard
	// (README: os.Open blocks indefinitely on a FIFO), and only the
	// Mode().IsRegular check refuses it without blocking.
	fifo := filepath.Join(t.TempDir(), "backend-fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}
	if _, err := terminalbackend.DigestFile(fifo); err == nil {
		t.Error("DigestFile(fifo) error = nil, want refusal for a non-regular file")
	}
}

// TestErrorPredicatesAreExclusive guards the negative-evidence contract: a
// refusal carries exactly one wire code, so a passing negative test cannot be
// explained by overlapping predicates.
func TestErrorPredicatesAreExclusive(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	mustRegisterExternal(t, registry, "com.example.term")

	cases := []struct {
		name string
		err  error
		want func(error) bool
	}{
		{"unknown", func() error { _, err := registry.Resolve("com.example.gone"); return err }(), terminalbackend.IsNotFound},
		{"duplicate", func() error {
			return registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), testExternalRecord("com.example.term"))
		}(), terminalbackend.IsAmbiguous},
		{"drift", func() error {
			drifted := testExternalRecord("com.example.term")
			drifted.ImplementationVersion = "9.9.9"
			return registry.RegisterExternal(scalar.PlatformLinux, testTrustEntry("com.example.term"), drifted)
		}(), terminalbackend.IsDrift},
		{"untrusted", func() error {
			entry := testTrustEntry("com.example.fresh")
			entry.Enabled = false
			return registry.RegisterExternal(scalar.PlatformLinux, entry, testExternalRecord("com.example.fresh"))
		}(), terminalbackend.IsUntrusted},
		{"restore mismatch", func() error {
			_, err := registry.RequireRestoreBinding("com.example.term", terminalbackend.BuiltinTmux)
			return err
		}(), terminalbackend.IsRestoreMismatch},
		{"stale generation", func() error {
			binding := terminalbackend.InstanceBinding{BackendID: "com.example.term", ImplementationVersion: "1.2.3", ProtocolVersion: "1.0.0", Generation: "a", TerminalBindingID: testDigest}
			stale := binding
			stale.Generation = "b"
			return terminalbackend.CheckProviderDescriptor(stale, binding)
		}(), terminalbackend.IsStaleGeneration},
	}
	predicates := map[string]func(error) bool{
		"not found": terminalbackend.IsNotFound, "ambiguous": terminalbackend.IsAmbiguous,
		"untrusted": terminalbackend.IsUntrusted, "drift": terminalbackend.IsDrift,
		"restore mismatch": terminalbackend.IsRestoreMismatch, "stale": terminalbackend.IsStaleGeneration,
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.err == nil {
				t.Fatalf("expected a refusal, got nil")
			}
			matched := 0
			for name, predicate := range predicates {
				if predicate(testCase.err) {
					matched++
					if !testCase.want(testCase.err) {
						t.Errorf("refusal %v unexpectedly matches %q", testCase.err, name)
					}
				}
			}
			if matched != 1 {
				t.Errorf("refusal %v matches %d predicates, want exactly 1", testCase.err, matched)
			}
		})
	}
}
