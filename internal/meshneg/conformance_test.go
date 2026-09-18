package meshneg_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/meshneg"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

// TestPeerExposure pins the core-only preservation rule: a negotiated peer
// reports above-core activation as explicitly unsupported, never as zero
// inventory or an empty result.
func TestPeerExposure(t *testing.T) {
	type activation struct {
		negotiated bool
		code       axerror.Code
		exit       int
	}
	cases := map[string]struct {
		config    string
		directory activation
		backend   activation
	}{
		// A v3/v4 node negotiating v2 performs core sync only and
		// exposes the peer as directory_mesh_unsupported (11.8); a
		// v4 node negotiating v2/v3 reports TerminalBackend evidence
		// unsupported rather than empty (11.9). The expected codes are
		// literals from the pinned text, never the production
		// constants: a same-exit code swap in negotiate.go must fail
		// here (mutations.py exposure-*-code-swap).
		"2.0.0": {"3.0.0",
			activation{false, "directory_mesh_unsupported", 6},
			activation{false, "terminal_backend_unavailable", 6}},
		"3.0.0": {"3.0.0",
			activation{true, "", 0},
			activation{false, "terminal_backend_unavailable", 6}},
		"4.0.0": {"3.0.0",
			activation{true, "", 0},
			activation{true, "", 0}},
		"5.0.0": {"4.0.0",
			activation{true, "", 0},
			activation{true, "", 0}},
	}
	for _, frame := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		t.Run(frame, func(t *testing.T) {
			want := cases[frame]
			hello := decodedHello(t, frame, profileContracts(t, frame))
			decision, err := meshneg.Negotiate(want.config, frame, hello)
			if err != nil {
				t.Fatal(err)
			}
			check := func(name string, got meshneg.FeatureActivation, want activation) {
				t.Helper()
				if got.Negotiated != want.negotiated {
					t.Fatalf("%s negotiated = %v, want %v", name, got.Negotiated, want.negotiated)
				}
				if got.Code != want.code || got.ExitCode != want.exit {
					t.Fatalf("%s class = %s/%d, want %s/%d", name, got.Code, got.ExitCode, want.code, want.exit)
				}
			}
			if decision.Peer.Directory.Feature != "directory" {
				t.Fatalf("directory feature = %q", decision.Peer.Directory.Feature)
			}
			if decision.Peer.BackendEvidence.Feature != "terminal_backend_evidence" {
				t.Fatalf("backend feature = %q", decision.Peer.BackendEvidence.Feature)
			}
			check("directory", decision.Peer.Directory, want.directory)
			check("backend", decision.Peer.BackendEvidence, want.backend)
			// The exposure codes resolve to exit 6 under the versions
			// that register them; the test derives those versions from
			// the registry instead of retyping them.
			for _, code := range []axerror.Code{"directory_mesh_unsupported", "terminal_backend_unavailable"} {
				var exits []int
				for _, version := range axerror.Versions() {
					registered, err := axerror.CodesFor(version)
					if err != nil {
						t.Fatal(err)
					}
					if !slices.Contains(registered, code) {
						continue
					}
					exit, err := axerror.ExitCodeFor(version, code)
					if err != nil {
						t.Fatal(err)
					}
					exits = append(exits, exit)
				}
				if len(exits) == 0 {
					t.Fatalf("exposure code %q is registered by no version", code)
				}
				if !slices.Equal(exits, slices.Repeat([]int{6}, len(exits))) {
					t.Fatalf("exposure code %q exits = %v, want all 6", code, exits)
				}
			}
		})
	}
}

// TestExposureCarriesNoInventory proves structurally that the peer view
// cannot report a core-only peer as zero inventory: its member set is
// exactly the status/code fields, with no count, inventory, or object
// member to read a zero from.
func TestExposureCarriesNoInventory(t *testing.T) {
	viewFields := []string{}
	for _, field := range reflect.VisibleFields(reflect.TypeFor[meshneg.PeerView]()) {
		if field.Anonymous {
			continue
		}
		viewFields = append(viewFields, field.Name)
	}
	slices.Sort(viewFields)
	if !slices.Equal(viewFields, []string{"BackendEvidence", "Directory"}) {
		t.Fatalf("PeerView members = %v", viewFields)
	}
	activationFields := []string{}
	for _, field := range reflect.VisibleFields(reflect.TypeFor[meshneg.FeatureActivation]()) {
		if field.Anonymous {
			continue
		}
		activationFields = append(activationFields, field.Name)
	}
	slices.Sort(activationFields)
	if !slices.Equal(activationFields, []string{"Code", "ExitCode", "Feature", "Negotiated"}) {
		t.Fatalf("FeatureActivation members = %v", activationFields)
	}
	for _, field := range []string{"Code", "Feature"} {
		entry, ok := reflect.TypeFor[meshneg.FeatureActivation]().FieldByName(field)
		if !ok || entry.Type.Kind() != reflect.String {
			t.Fatalf("FeatureActivation.%s is not a string member", field)
		}
	}
}

// TestRefusalClassesResolveFromTheRegistry proves every refusal class this
// package emits is constructible through the axerror registry with the
// pinned code and exit, and that the class is stable across every version
// registering the code.
func TestRefusalClassesResolveFromTheRegistry(t *testing.T) {
	codes := []struct {
		code       axerror.Code
		exit       int
		introduced axerror.Version
	}{
		{"incompatible_protocol", 6, axerror.Version100},
		{"invalid_config", 3, axerror.Version100},
		{"directory_mesh_unsupported", 6, axerror.Version120},
		{"terminal_backend_unavailable", 6, axerror.Version130},
	}
	for _, tc := range codes {
		t.Run(string(tc.code), func(t *testing.T) {
			exit, err := axerror.ExitCodeFor(tc.introduced, tc.code)
			if err != nil {
				t.Fatal(err)
			}
			if exit != tc.exit {
				t.Fatalf("ExitCodeFor(%s, %s) = %d, want %d", tc.introduced, tc.code, exit, tc.exit)
			}
			built, err := axerror.New(axerror.Spec{
				Version: tc.introduced,
				Code:    tc.code,
				Message: "registry constructibility probe",
				IDs:     axerror.NoIDs(),
				Details: axerror.Details{},
			})
			if err != nil {
				t.Fatal(err)
			}
			if built.Code() != tc.code || built.ExitCode() != tc.exit || !built.CodeRegistered() {
				t.Fatalf("constructed %s/%d registered=%v", built.Code(), built.ExitCode(), built.CodeRegistered())
			}
			for _, version := range axerror.Versions() {
				registered, err := axerror.CodesFor(version)
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Contains(registered, tc.code) {
					continue
				}
				got, err := axerror.ExitCodeFor(version, tc.code)
				if err != nil {
					t.Fatal(err)
				}
				if got != tc.exit {
					t.Fatalf("ExitCodeFor(%s, %s) = %d, want %d", version, tc.code, got, tc.exit)
				}
			}
		})
	}
	// The registration gaps are real: the directory code is absent from
	// 1.0.0/1.1.0 and the backend code from everything before 1.3.0, so a
	// plant that resolves them under the wrong version fails closed.
	if _, err := axerror.ExitCodeFor(axerror.Version100, "directory_mesh_unsupported"); err == nil {
		t.Fatal("directory exposure code resolves under 1.0.0")
	}
	if _, err := axerror.ExitCodeFor(axerror.Version120, "terminal_backend_unavailable"); err == nil {
		t.Fatal("backend exposure code resolves under 1.2.0")
	}
}

// TestNegotiationIsNotAdmission proves the selection ignores every hello
// member the admission boundary owns: identity, platform, version, nonce,
// echo, and limits. Two hellos that differ only outside the contracts map
// negotiate identically, including with values decode itself would refuse.
func TestNegotiationIsNotAdmission(t *testing.T) {
	base := decodedHello(t, "3.0.0", profileContracts(t, "3.0.0"))
	want, err := meshneg.Negotiate("3.0.0", "3.0.0", base)
	if err != nil {
		t.Fatal(err)
	}
	variants := map[string]func(*rpcwire.Hello){
		"other-host":       func(h *rpcwire.Hello) { h.HostID = "0198f4c8-a070-7188-9172-2234567890ab" },
		"v4-host":          func(h *rpcwire.Hello) { h.HostID = "0198f4c8-a070-4188-9172-1234567890ab" },
		"garbage-host":     func(h *rpcwire.Hello) { h.HostID = "not-a-uuid" },
		"empty-host":       func(h *rpcwire.Hello) { h.HostID = "" },
		"other-platform":   func(h *rpcwire.Hello) { h.Platform = "windows" },
		"invalid-platform": func(h *rpcwire.Hello) { h.Platform = "ios" },
		"other-version":    func(h *rpcwire.Hello) { h.AXVersion = "9.9.9" },
		"invalid-version":  func(h *rpcwire.Hello) { h.AXVersion = "01.0.0" },
		"short-nonce":      func(h *rpcwire.Hello) { h.Nonce = "c2hvcnQ" },
		"empty-nonce":      func(h *rpcwire.Hello) { h.Nonce = "" },
		"echo-set":         func(h *rpcwire.Hello) { h.NonceEcho = base.Nonce },
		"zero-limits": func(h *rpcwire.Hello) {
			h.MaxLineBytes = mustUint53(t, 0)
			h.MaxObjectBytes = mustUint53(t, 0)
		},
		"huge-limits": func(h *rpcwire.Hello) {
			h.MaxLineBytes = mustUint53(t, 1<<40)
			h.MaxObjectBytes = mustUint53(t, 1<<40)
		},
	}
	for name, mutate := range variants {
		t.Run(name, func(t *testing.T) {
			hello := base
			mutate(&hello)
			got, err := meshneg.Negotiate("3.0.0", "3.0.0", hello)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("identity-adjacent member %q changed the decision", name)
			}
		})
	}
}

// TestNoFallbackSurface proves Section 6.6 structurally: environment,
// argv, prior-handshake state, and extension members cannot select or alter
// a selection, because the entries consult none of them.
func TestNoFallbackSurface(t *testing.T) {
	hello := decodedHello(t, "2.0.0", profileContracts(t, "2.0.0"))
	want, err := meshneg.Negotiate("3.0.0", "2.0.0", hello)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("environment", func(t *testing.T) {
		t.Setenv("AX_RPC_MAJOR", "5")
		t.Setenv("AX_FALLBACK", "2.0.0")
		t.Setenv("AX_CONFIG_GENERATION", "4.0.0")
		t.Setenv("HOST_CHANNEL", "1.0.0")
		got, err := meshneg.Negotiate("3.0.0", "2.0.0", hello)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatal("environment changed the decision")
		}
	})
	t.Run("argv", func(t *testing.T) {
		argv := os.Args
		t.Cleanup(func() { os.Args = argv })
		os.Args = []string{"ax", "rpc", "serve", "--stdio", "--host-channel", "1.0.0", peerHost}
		got, err := meshneg.Negotiate("3.0.0", "2.0.0", hello)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatal("argv changed the decision")
		}
	})
	t.Run("stateless", func(t *testing.T) {
		// A refusal retains no fallback: the same inputs refuse
		// identically, a corrected offer then succeeds, and a later
		// disjoint offer still refuses. Order is irrelevant.
		five := decodedHello(t, "5.0.0", profileContracts(t, "5.0.0"))
		for range 3 {
			if _, err := meshneg.Negotiate("3.0.0", "5.0.0", five); err == nil {
				t.Fatal("legacy against RPC 5 admitted on repeat")
			}
		}
		got, err := meshneg.Negotiate("3.0.0", "2.0.0", hello)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatal("prior refusal poisoned a later selection")
		}
		if _, err := meshneg.Negotiate("3.0.0", "5.0.0", five); err == nil {
			t.Fatal("prior success pinned a later refusal open")
		}
	})
	t.Run("extension-member", func(t *testing.T) {
		m := profileContracts(t, "2.0.0")
		m["extensions"] = []string{"1.0.0"}
		if _, err := meshneg.Negotiate("3.0.0", "2.0.0", directHello(t, m)); err == nil {
			t.Fatal("extension member admitted a selection")
		} else {
			refusal := asRefusal(t, err)
			if refusal.Reason != meshneg.ReasonContractShape {
				t.Fatalf("extension member reason = %q", refusal.Reason)
			}
		}
	})
}

// TestSemverAgreementWithRPCWire measures that the negotiator's strict
// version grammar accepts exactly the rpc versions rpcwire decode accepts
// inside a v2 hello, and that valid versions of another major are exactly
// where the two deliberately diverge: decode admits the syntax while
// negotiation refuses the coercion.
func TestSemverAgreementWithRPCWire(t *testing.T) {
	probes := []string{
		"2.0.0", "2.1.0", "2.0.1", "2.10.20", "2.100.200", "2.0.0-alpha", "2.0.0+build",
		"2.0.0-alpha+build", "2.0.0-0",
		"", "2", "2.0", "v2.0.0", "02.0.0", "2.0.0-", "2.0.0+",
		" 2.0.0", "2.0.0 ", "2.0.0.0", "a.b.c", "2.0.0--x",
	}
	for _, probe := range probes {
		t.Run("major2-"+probe, func(t *testing.T) {
			contracts := profileContracts(t, "2.0.0")
			contracts["rpc"] = []string{probe}
			body, err := json.Marshal(map[string]any{
				"host_id": peerHost, "platform": "macos", "ax_version": "0.2.1",
				"nonce": rpcwire.NewNonce(), "contracts": contracts,
				"max_line_bytes": 8 * 1024 * 1024, "max_object_bytes": 5 * 1024 * 1024,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, wireErr := rpcwire.EncodeRequest("2.0.0", requestID, "hello", body)
			_, negErr := meshneg.PeerOffer("2.0.0", directHello(t, contracts))
			if (wireErr == nil) != (negErr == nil) {
				t.Fatalf("probe %q: rpcwire err=%v, negotiator err=%v", probe, wireErr, negErr)
			}
		})
	}
	for _, probe := range []string{"1.0.0", "3.0.0", "4.0.0", "5.0.0", "9.9.9", "99999999999999999999999.0.0"} {
		t.Run("other-major-"+probe, func(t *testing.T) {
			contracts := profileContracts(t, "2.0.0")
			contracts["rpc"] = []string{probe}
			body, err := json.Marshal(map[string]any{
				"host_id": peerHost, "platform": "macos", "ax_version": "0.2.1",
				"nonce": rpcwire.NewNonce(), "contracts": contracts,
				"max_line_bytes": 8 * 1024 * 1024, "max_object_bytes": 5 * 1024 * 1024,
			})
			if err != nil {
				t.Fatal(err)
			}
			line, err := rpcwire.EncodeRequest("2.0.0", requestID, "hello", body)
			if err != nil {
				t.Fatalf("probe %q: rpcwire refused valid v2 syntax: %v", probe, err)
			}
			request, err := rpcwire.DecodeRequest(line)
			if err != nil {
				t.Fatal(err)
			}
			hello, err := request.Hello()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := meshneg.PeerOffer("2.0.0", hello); err == nil {
				t.Fatalf("probe %q: negotiator coerced major from a v2 frame", probe)
			} else {
				refusal := asRefusal(t, err)
				if refusal.Reason != meshneg.ReasonMixedMajor {
					t.Fatalf("probe %q: reason = %q", probe, refusal.Reason)
				}
			}
		})
	}
}

// TestConfig4NeverDowngrades pins the Section 6.6 one-way gate: a Config-4
// endpoint offers exactly RPC 5 and refuses every legacy frame.
func TestConfig4NeverDowngrades(t *testing.T) {
	majors, err := meshneg.LocalMajors("4.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(majors, []int{5}) {
		t.Fatalf("Config-4 offers %v, want exactly [5]", majors)
	}
	for _, frame := range []string{"2.0.0", "3.0.0", "4.0.0"} {
		hello := decodedHello(t, frame, profileContracts(t, frame))
		if decision, err := meshneg.Negotiate("4.0.0", frame, hello); err == nil {
			t.Fatalf("Config-4 admitted legacy frame %s as major %d", frame, decision.Major)
		} else if refusal := asRefusal(t, err); refusal.Reason != meshneg.ReasonNoCommonMajor {
			t.Fatalf("Config-4 against %s reason = %q", frame, refusal.Reason)
		}
	}
	hello := decodedHello(t, "5.0.0", profileContracts(t, "5.0.0"))
	decision, err := meshneg.Negotiate("4.0.0", "5.0.0", hello)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Major != 5 || decision.ErrorVersion != axerror.Version130 {
		t.Fatalf("Config-4 RPC 5 decision = %+v", decision)
	}
}

// TestLegacyNeverSelectsRPC5 pins the other side: explicitly legacy
// installations refuse an RPC-5-only peer instead of coercing it.
func TestLegacyNeverSelectsRPC5(t *testing.T) {
	hello := decodedHello(t, "5.0.0", profileContracts(t, "5.0.0"))
	for _, config := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		decision, err := meshneg.Negotiate(config, "5.0.0", hello)
		if err == nil {
			t.Fatalf("legacy %s admitted RPC 5 as major %d", config, decision.Major)
		}
		refusal := asRefusal(t, err)
		if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
			t.Fatalf("legacy %s class = %s/%d", config, refusal.Code, refusal.ExitCode)
		}
		if !errors.Is(err, meshneg.ErrNoCommonMajor) {
			t.Fatalf("legacy %s error does not wrap ErrNoCommonMajor", config)
		}
		if refusal.Peer != 5 {
			t.Fatalf("legacy %s refusal peer = %d, want 5", config, refusal.Peer)
		}
	}
}

// TestNegotiationIsDeterministic runs every entry over the full input space
// twice and requires byte-identical outcomes: the package keeps no state,
// reads no clock, and performs no I/O, so crash/idempotency surface is
// absent by construction rather than by testing.
func TestNegotiationIsDeterministic(t *testing.T) {
	configs := []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0", "9.9.9"}
	frames := []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0", "1.0.0"}
	hellos := map[string]rpcwire.Hello{}
	for _, frame := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		hellos[frame] = directHello(t, profileContracts(t, frame))
	}
	encode := func(decision meshneg.Decision, err error) string {
		refusal, _ := err.(*meshneg.Refusal)
		raw, jsonErr := json.Marshal(struct {
			Decision meshneg.Decision
			Refusal  *meshneg.Refusal
		}{decision, refusal})
		if jsonErr != nil {
			t.Fatal(jsonErr)
		}
		return string(raw)
	}
	for _, config := range configs {
		for _, frame := range frames {
			hello := hellos[frame]
			first := encode(meshneg.Negotiate(config, frame, hello))
			for range 25 {
				if again := encode(meshneg.Negotiate(config, frame, hello)); again != first {
					t.Fatalf("Negotiate(%s, %s) is nondeterministic", config, frame)
				}
			}
		}
	}
}

func TestRefusalShape(t *testing.T) {
	_, err := meshneg.Negotiate("3.0.0", "5.0.0", decodedHello(t, "5.0.0", profileContracts(t, "5.0.0")))
	refusal := asRefusal(t, err)
	if refusal.Error() == "" {
		t.Fatal("empty refusal message")
	}
	if !slices.Equal(refusal.Local, []int{2, 3, 4}) {
		t.Fatalf("refusal local = %v", refusal.Local)
	}
	var nilRefusal *meshneg.Refusal
	if nilRefusal.Error() == "" || nilRefusal.Unwrap() != nil {
		t.Fatal("nil refusal mishandled")
	}
}
