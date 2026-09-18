package axpane

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func descriptorBinding() BindingRef {
	return BindingRef{
		BindingDigest:         seedDigest(0xB1),
		BackendID:             terminalbackend.BuiltinTmux,
		ImplementationVersion: "2.1.0",
		ProtocolVersion:       "1.1.0",
		Generation:            "generation-alpha",
	}
}

func descriptorHostBinding() terminalbackend.InstanceBinding {
	return terminalbackend.InstanceBinding{
		BackendID:             terminalbackend.BuiltinTmux,
		ImplementationVersion: "2.1.0",
		ProtocolVersion:       "1.1.0",
		Generation:            "generation-alpha",
		TerminalBindingID:     seedDigest(0xB1),
	}
}

// TestDescriptorBuildAdmitRoundTrip supplies the §7.A descriptor
// through the wrapper builder and admits it through the landed
// provider-side gate against the validated host-local binding:
// parse, match, then launch, with no caching across the check.
func TestDescriptorBuildAdmitRoundTrip(t *testing.T) {
	t.Parallel()
	doc, err := BuildDescriptor(DescriptorParams{
		Binding:     descriptorBinding(),
		InstanceID:  fixtureInstance,
		Interactive: true,
		Columns:     80,
		Rows:        24,
	})
	if err != nil {
		t.Fatalf("BuildDescriptor() error = %v", err)
	}
	admitted, err := terminalbackend.AdmitProviderDescriptor(doc, descriptorHostBinding())
	if err != nil {
		t.Fatalf("AdmitProviderDescriptor() error = %v", err)
	}
	if admitted.TerminalInstanceID != fixtureInstance || !admitted.Interactive || admitted.Columns != 80 || admitted.Rows != 24 {
		t.Fatalf("AdmitProviderDescriptor() = %+v, want the supplied context", admitted)
	}
}

func TestDescriptorRefusesBadGeometry(t *testing.T) {
	t.Parallel()
	for _, geometry := range [][2]uint16{{0, 24}, {80, 0}, {1001, 24}, {80, 1001}} {
		if _, err := BuildDescriptor(DescriptorParams{Binding: descriptorBinding(), InstanceID: fixtureInstance, Columns: geometry[0], Rows: geometry[1]}); err == nil {
			t.Fatalf("BuildDescriptor(%dx%d) succeeded, want refusal", geometry[0], geometry[1])
		}
	}
	for _, geometry := range [][2]uint16{{1, 1}, {1000, 1000}, {80, 24}} {
		if _, err := BuildDescriptor(DescriptorParams{Binding: descriptorBinding(), InstanceID: fixtureInstance, Columns: geometry[0], Rows: geometry[1]}); err != nil {
			t.Fatalf("BuildDescriptor(%dx%d) error = %v", geometry[0], geometry[1], err)
		}
	}
}

func TestDescriptorRefusesNonUUIDInstance(t *testing.T) {
	t.Parallel()
	for _, instance := range []string{"1234", "not-a-uuid", "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"} {
		if _, err := BuildDescriptor(DescriptorParams{Binding: descriptorBinding(), InstanceID: instance, Columns: 80, Rows: 24}); err == nil {
			t.Fatalf("BuildDescriptor(instance %q) succeeded, want refusal", instance)
		}
	}
}

func TestDescriptorMismatchRefuses(t *testing.T) {
	t.Parallel()
	doc, err := BuildDescriptor(DescriptorParams{Binding: descriptorBinding(), InstanceID: fixtureInstance, Columns: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	stale := descriptorHostBinding()
	stale.Generation = "generation-beta"
	if _, err := terminalbackend.AdmitProviderDescriptor(doc, stale); err == nil {
		t.Fatal("AdmitProviderDescriptor(drifted generation) succeeded, want refusal")
	}
	foreign := descriptorHostBinding()
	foreign.TerminalBindingID = seedDigest(0xB2)
	if _, err := terminalbackend.AdmitProviderDescriptor(doc, foreign); err == nil {
		t.Fatal("AdmitProviderDescriptor(foreign binding) succeeded, want refusal")
	}
	if _, err := terminalbackend.AdmitProviderDescriptor([]byte(`{"broken":true}`), descriptorHostBinding()); err == nil {
		t.Fatal("AdmitProviderDescriptor(malformed) succeeded, want refusal")
	}
}

// TestDescriptorGenerationBounds pins the §7.A string[1..256] bound
// through the admission gate: lengths 0 and 257 refuse, 256 admits.
func TestDescriptorGenerationBounds(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("g", 256)
	tooLong := strings.Repeat("g", 257)
	doc, err := BuildDescriptor(DescriptorParams{
		Binding:    BindingRef{BindingDigest: seedDigest(0xB1), BackendID: terminalbackend.BuiltinTmux, ImplementationVersion: "2.1.0", ProtocolVersion: "1.1.0", Generation: long},
		InstanceID: fixtureInstance,
		Columns:    80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("BuildDescriptor(256) error = %v", err)
	}
	binding := descriptorHostBinding()
	binding.Generation = long
	if _, err := terminalbackend.AdmitProviderDescriptor(doc, binding); err != nil {
		t.Fatalf("AdmitProviderDescriptor(256) error = %v", err)
	}
	bad, err := BuildDescriptor(DescriptorParams{
		Binding:    BindingRef{BindingDigest: seedDigest(0xB1), BackendID: terminalbackend.BuiltinTmux, ImplementationVersion: "2.1.0", ProtocolVersion: "1.1.0", Generation: tooLong},
		InstanceID: fixtureInstance,
		Columns:    80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("BuildDescriptor(257) error = %v", err)
	}
	binding.Generation = tooLong
	if _, err := terminalbackend.AdmitProviderDescriptor(bad, binding); err == nil {
		t.Fatal("AdmitProviderDescriptor(257) succeeded, want refusal")
	}
	empty, err := BuildDescriptor(DescriptorParams{
		Binding:    BindingRef{BindingDigest: seedDigest(0xB1), BackendID: terminalbackend.BuiltinTmux, ImplementationVersion: "2.1.0", ProtocolVersion: "1.1.0", Generation: ""},
		InstanceID: fixtureInstance,
		Columns:    80,
		Rows:       24,
	})
	if err != nil {
		t.Fatalf("BuildDescriptor(0) error = %v", err)
	}
	binding.Generation = ""
	if _, err := terminalbackend.AdmitProviderDescriptor(empty, binding); err == nil {
		t.Fatal("AdmitProviderDescriptor(0) succeeded, want refusal")
	}
}

func TestMintInstanceID(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, time.August, 19, 4, 12, 0, 123, time.UTC)
	first, err := MintInstanceID(fixed, rand.Read)
	if err != nil {
		t.Fatalf("MintInstanceID() error = %v", err)
	}
	parsed, err := scalar.ParseUUIDv7(first)
	if err != nil {
		t.Fatalf("minted %q is not a UUIDv7: %v", first, err)
	}
	if parsed.String() != first {
		t.Fatalf("minted %q does not round-trip", first)
	}
	second, err := MintInstanceID(fixed, rand.Read)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("MintInstanceID() minted the same instance twice")
	}
	system, err := MintInstanceIDRandom()
	if err != nil {
		t.Fatalf("MintInstanceIDRandom() error = %v", err)
	}
	if _, err := scalar.ParseUUIDv7(system); err != nil {
		t.Fatalf("MintInstanceIDRandom() minted %q: %v", system, err)
	}
}
