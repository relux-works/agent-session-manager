package termbind

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/cliresult"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func TestCheckInstanceIdentityAdmitsUUIDv7(t *testing.T) {
	t.Parallel()
	if err := CheckInstanceIdentity(fixtureInstance); err != nil {
		t.Fatalf("CheckInstanceIdentity(UUIDv7) error = %v", err)
	}
}

func TestCheckInstanceIdentityRefusesForbiddenForms(t *testing.T) {
	t.Parallel()
	for _, forbidden := range forbiddenIdentities {
		t.Run(forbidden.form, func(t *testing.T) {
			t.Parallel()
			requireCode(t, CheckInstanceIdentity(forbidden.value), "terminal_backend_protocol_error")
		})
	}
}

func TestParseTerminalBindingAdmitsClosedFixture(t *testing.T) {
	t.Parallel()
	parsed, err := ParseTerminalBinding(bindingDoc(t, nil))
	if err != nil {
		t.Fatalf("ParseTerminalBinding() error = %v", err)
	}
	if parsed.TerminalInstanceID != fixtureInstance {
		t.Fatalf("TerminalInstanceID = %q, want %q", parsed.TerminalInstanceID, fixtureInstance)
	}
	if parsed.SessionID != fixtureSession || parsed.TerminalBackendID != terminalbackend.BuiltinTmux {
		t.Fatalf("binding identity = %+v", parsed)
	}
	if parsed.ImplementationVersion != fixtureImpl || parsed.ProtocolVersion != fixtureProto {
		t.Fatalf("binding versions = %+v", parsed)
	}
	if parsed.BackendGeneration != fixtureRawGen || parsed.NativeReference != "tmux-session-payments-0" {
		t.Fatalf("binding locals = %+v", parsed)
	}
	if parsed.HasSupersedes {
		t.Fatalf("HasSupersedes = true, want false for the first binding")
	}
	if parsed.BindingID == "" {
		t.Fatal("BindingID is empty")
	}
}

func TestParseTerminalBindingRefusesForbiddenIdentity(t *testing.T) {
	t.Parallel()
	for _, forbidden := range forbiddenIdentities {
		t.Run(forbidden.form, func(t *testing.T) {
			t.Parallel()
			raw := bindingDoc(t, func(object map[string]any) {
				object["terminal_instance_id"] = forbidden.value
			})
			requireCode(t, func() error { _, err := ParseTerminalBinding(raw); return err }(), "terminal_backend_protocol_error")
		})
	}
}

func TestParseTerminalBindingClosedShape(t *testing.T) {
	t.Parallel()
	t.Run("missing member", func(t *testing.T) {
		t.Parallel()
		raw := bindingDoc(t, func(object map[string]any) { delete(object, "native_reference") })
		if _, err := ParseTerminalBinding(raw); err == nil {
			t.Fatal("ParseTerminalBinding(missing member) succeeded, want refusal")
		}
	})
	t.Run("extra member", func(t *testing.T) {
		t.Parallel()
		// A string extra isolates the member-set arm: its value is
		// otherwise admissible, so only the count check can refuse it
		// (the N-binding-members narrowing admits exactly this shape).
		raw := bindingDoc(t, func(object map[string]any) { object["pid"] = "12345" })
		_, err := ParseTerminalBinding(raw)
		requireCode(t, err, "terminal_backend_protocol_error")
		requireDetail(t, err, "binding member set")
	})
	t.Run("duplicate member", func(t *testing.T) {
		t.Parallel()
		raw := bindingDoc(t, nil)
		duplicated := strings.Replace(string(raw), `"host_id":`, `"host_id":"`+fixtureLocalHost+`","host_id":`, 1)
		if _, err := ParseTerminalBinding([]byte(duplicated)); err == nil {
			t.Fatal("ParseTerminalBinding(duplicate member) succeeded, want refusal")
		}
	})
	t.Run("wrong schema", func(t *testing.T) {
		t.Parallel()
		raw := bindingDoc(t, func(object map[string]any) { object["schema"] = "urn:ax:schema:session-event" })
		if _, err := ParseTerminalBinding(raw); err == nil {
			t.Fatal("ParseTerminalBinding(wrong schema) succeeded, want refusal")
		}
	})
	t.Run("generation bounds", func(t *testing.T) {
		t.Parallel()
		if _, err := ParseTerminalBinding(bindingDoc(t, func(object map[string]any) {
			object["backend_generation"] = strings.Repeat("g", 256)
		})); err != nil {
			t.Fatalf("ParseTerminalBinding(generation 256) error = %v", err)
		}
		for _, length := range []int{0, 257} {
			raw := bindingDoc(t, func(object map[string]any) { object["backend_generation"] = strings.Repeat("g", length) })
			if _, err := ParseTerminalBinding(raw); err == nil {
				t.Fatalf("ParseTerminalBinding(generation %d) succeeded, want refusal", length)
			}
		}
	})
	t.Run("native reference bounds", func(t *testing.T) {
		t.Parallel()
		if _, err := ParseTerminalBinding(bindingDoc(t, func(object map[string]any) {
			object["native_reference"] = strings.Repeat("n", 512)
		})); err != nil {
			t.Fatalf("ParseTerminalBinding(native 512) error = %v", err)
		}
		for _, length := range []int{0, 513} {
			raw := bindingDoc(t, func(object map[string]any) { object["native_reference"] = strings.Repeat("n", length) })
			if _, err := ParseTerminalBinding(raw); err == nil {
				t.Fatalf("ParseTerminalBinding(native %d) succeeded, want refusal", length)
			}
		}
	})
	t.Run("tampered member breaks identity", func(t *testing.T) {
		t.Parallel()
		raw := bindingDoc(t, nil)
		var object map[string]any
		if err := json.Unmarshal(raw, &object); err != nil {
			t.Fatal(err)
		}
		object["native_reference"] = "tmux-session-payments-9"
		tampered, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		_, err = ParseTerminalBinding(tampered)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
	t.Run("wrong self id", func(t *testing.T) {
		t.Parallel()
		raw := bindingDoc(t, func(object map[string]any) { object["binding_id"] = seedDigest(0xDD) })
		_, err := ParseTerminalBinding(raw)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
}

func TestBindingIdentityAgreesWithLandedAdmission(t *testing.T) {
	t.Parallel()
	world := buildTmuxUniverse(t)
	// Recompute the manifest identity through this package's Section 4.B
	// construction and prove the landed manifest parser admits it: the
	// two constructions agree byte for byte.
	manifest := mustJSON(t, world.manifest)
	recomputed, err := bindingIdentity(manifest, "manifest_id")
	if err != nil {
		t.Fatalf("bindingIdentity() error = %v", err)
	}
	if recomputed != world.manifestID {
		t.Fatalf("bindingIdentity() = %q, want manifest %q", recomputed, world.manifestID)
	}
	var object map[string]any
	if err := json.Unmarshal(manifest, &object); err != nil {
		t.Fatal(err)
	}
	object["manifest_id"] = recomputed
	relabeled, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := terminalbackend.ParseManifest(relabeled); err != nil {
		t.Fatalf("ParseManifest(recomputed identity) error = %v", err)
	}
}

func descriptorDoc(instanceID string) []byte {
	raw, _ := json.Marshal(map[string]any{
		"terminal_binding_id":    seedDigest(0xB1),
		"terminal_instance_id":   instanceID,
		"terminal_backend_id":    terminalbackend.BuiltinTmux,
		"implementation_version": fixtureImpl,
		"protocol_version":       fixtureProto,
		"backend_generation":     fixtureRawGen,
		"interactive":            true,
		"columns":                float64(80),
		"rows":                   float64(24),
	})
	return raw
}

func descriptorBinding() terminalbackend.InstanceBinding {
	return terminalbackend.InstanceBinding{
		BackendID:             terminalbackend.BuiltinTmux,
		ImplementationVersion: fixtureImpl,
		ProtocolVersion:       fixtureProto,
		Generation:            fixtureRawGen,
		TerminalBindingID:     seedDigest(0xB1),
	}
}

func TestAdmitDescriptorRefusesForbiddenIdentity(t *testing.T) {
	t.Parallel()
	if _, err := terminalbackend.AdmitProviderDescriptor(descriptorDoc(fixtureInstance), descriptorBinding()); err != nil {
		t.Fatalf("AdmitProviderDescriptor(UUIDv7) error = %v", err)
	}
	for _, forbidden := range forbiddenIdentities {
		t.Run(forbidden.form, func(t *testing.T) {
			t.Parallel()
			_, err := terminalbackend.AdmitProviderDescriptor(descriptorDoc(forbidden.value), descriptorBinding())
			requireCode(t, err, "terminal_backend_protocol_error")
		})
	}
}

func mutationContextDoc(instanceID string) []byte {
	auth := `{"lease_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","lease_epoch":1,` +
		`"holder_host_id":"0198f4c8-8e50-7f66-8f70-aaaaaaaaaaa1","authorization_kind":"control",` +
		`"issued_at":"2026-09-01T00:00:00.000Z","expires_at":"2026-09-02T00:00:00.000Z",` +
		`"authorization_evidence_id":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	key := instanceID + "/quiesce/0198f4c8-8e50-7f66-8f70-ddddddddddd1"
	encodedInstance, _ := json.Marshal(instanceID)
	encodedKey, _ := json.Marshal(key)
	raw := `{"operation_id":"0198f4c8-8e50-7f66-8f70-ccccccccccc1",` +
		`"session_id":"0198f4c8-8e50-7f66-8f70-1234567890ab",` +
		`"terminal_instance_id":` + string(encodedInstance) + `,` +
		`"terminal_backend_id":"example.backend",` +
		`"implementation_version":"1.2.3",` +
		`"protocol_version":"1.0.0",` +
		`"backend_generation":"generation-one",` +
		`"idempotency_key":` + string(encodedKey) + `,` +
		`"deadline_at":"2026-09-01T18:00:00.000Z",` +
		`"authorization":` + auth + `}`
	return []byte(raw)
}

func TestAdmitMutationContextRefusesForbiddenIdentity(t *testing.T) {
	t.Parallel()
	if _, err := AdmitMutationContext(mutationContextDoc("0198f4c8-8e50-7f66-8f70-bbbbbbbbbbb1")); err != nil {
		t.Fatalf("AdmitMutationContext(UUIDv7) error = %v", err)
	}
	for _, forbidden := range forbiddenIdentities {
		t.Run(forbidden.form, func(t *testing.T) {
			t.Parallel()
			_, err := AdmitMutationContext(mutationContextDoc(forbidden.value))
			requireCode(t, err, "terminal_backend_protocol_error")
		})
	}
}

func statusBodyDoc(instanceFragment string) []byte {
	raw := `{"session_id":"` + fixtureSession + `",` +
		`"terminal_instance_id":` + instanceFragment + `,` +
		`"terminal_backend_id":"ax.tmux",` +
		`"implementation_version":"2.1.0",` +
		`"protocol_version":"1.1.0",` +
		`"backend_generation":"generation-alpha",` +
		`"include_provider_observation":false,` +
		`"deadline_at":"2026-08-19T05:00:00.000Z"}`
	return []byte(raw)
}

func statusInstanceFragment(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func TestAdmitStatusBodyRefusesForbiddenIdentity(t *testing.T) {
	t.Parallel()
	if _, err := AdmitStatusBody(statusBodyDoc(statusInstanceFragment(fixtureInstance))); err != nil {
		t.Fatalf("AdmitStatusBody(UUIDv7) error = %v", err)
	}
	sessionScoped := `{"session_id":"` + fixtureSession + `",` +
		`"terminal_instance_id":null,` +
		`"terminal_backend_id":"ax.tmux",` +
		`"implementation_version":"2.1.0",` +
		`"protocol_version":"1.1.0",` +
		`"backend_generation":null,` +
		`"include_provider_observation":false,` +
		`"deadline_at":"2026-08-19T05:00:00.000Z"}`
	if _, err := AdmitStatusBody([]byte(sessionScoped)); err != nil {
		t.Fatalf("AdmitStatusBody(null instance) error = %v", err)
	}
	for _, forbidden := range forbiddenIdentities {
		t.Run(forbidden.form, func(t *testing.T) {
			t.Parallel()
			_, err := AdmitStatusBody(statusBodyDoc(statusInstanceFragment(forbidden.value)))
			requireCode(t, err, "terminal_backend_protocol_error")
		})
	}
}

func TestAdmitSurfacesDelegateShapeFaults(t *testing.T) {
	t.Parallel()
	t.Run("context non-string instance", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{"operation_id":"0198f4c8-8e50-7f66-8f70-ccccccccccc1",` +
			`"session_id":"0198f4c8-8e50-7f66-8f70-1234567890ab",` +
			`"terminal_instance_id":12345,` +
			`"terminal_backend_id":"example.backend",` +
			`"implementation_version":"1.2.3",` +
			`"protocol_version":"1.0.0",` +
			`"backend_generation":"generation-one",` +
			`"idempotency_key":"k",` +
			`"deadline_at":"2026-09-01T18:00:00.000Z",` +
			`"authorization":{"lease_id":"f47ac10b-58cc-4372-a567-0e02b2c3d479","lease_epoch":1,` +
			`"holder_host_id":"0198f4c8-8e50-7f66-8f70-aaaaaaaaaaa1","authorization_kind":"control",` +
			`"issued_at":"2026-09-01T00:00:00.000Z","expires_at":"2026-09-02T00:00:00.000Z",` +
			`"authorization_evidence_id":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`)
		if _, err := AdmitMutationContext(raw); err == nil {
			t.Fatal("AdmitMutationContext(numeric instance) succeeded, want refusal")
		}
	})
	t.Run("status non-string instance", func(t *testing.T) {
		t.Parallel()
		if _, err := AdmitStatusBody(statusBodyDoc(`12345`)); err == nil {
			t.Fatal("AdmitStatusBody(numeric instance) succeeded, want refusal")
		}
	})
}

func TestBootstrapBindingRefusesForbiddenIdentity(t *testing.T) {
	t.Parallel()
	for _, forbidden := range forbiddenIdentities {
		t.Run(forbidden.form, func(t *testing.T) {
			t.Parallel()
			store, err := axpane.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			candidate := axpane.Binding{
				SessionID:          fixtureSession,
				OperationID:        fixtureBootstrap,
				TerminalInstanceID: forbidden.value,
				BindingDigest:      seedDigest(0xB1),
			}
			if _, _, err := store.Bind(fixtureSession, fixtureBootstrap, candidate); err == nil {
				t.Fatalf("Bind(%s identity) succeeded, want refusal", forbidden.form)
			}
			if _, found, err := store.Lookup(fixtureSession, fixtureBootstrap); err != nil || found {
				t.Fatalf("Lookup() after refused bind = found %v err %v, want absence", found, err)
			}
		})
	}
}

func TestCLIResult4IdentityIsUnimplementedBound(t *testing.T) {
	t.Parallel()
	var selected []cliresult.Command
	for _, command := range cliresult.Commands() {
		version, err := cliresult.RegisteredVersionForCommand(command)
		if err != nil {
			t.Fatal(err)
		}
		if version == cliresult.Version400 {
			selected = append(selected, command)
		}
	}
	if len(selected) == 0 {
		t.Fatal("no command selects CLI Result 4.0.0: the bound names no surface")
	}
	for _, command := range selected {
		if _, err := cliresult.VersionForCommand(command); !errors.Is(err, cliresult.ErrUnimplementedVersion) {
			t.Fatalf("VersionForCommand(%q) error = %v, want ErrUnimplementedVersion: Result 4.0.0 is implemented and needs identity coverage", string(command), err)
		}
	}
}
