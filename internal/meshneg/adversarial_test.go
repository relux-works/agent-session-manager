package meshneg_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/meshneg"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

// generationForFrame is the newest local generation that admits each pinned
// frame: Config-4 only for RPC 5, Config-3 otherwise.
func generationForFrame(frame string) string {
	if frame == "5.0.0" {
		return "4.0.0"
	}
	return "3.0.0"
}

// TestPeerOfferKeyMembershipPerKey closes inherited advisory N1: the
// contracts key-membership gate (negotiate.go membership loop) is pinned at
// EVERY key of EVERY frame's pinned profile, not at one hand-picked
// member. Both the missing-key and the substituted-key cases are derived
// from rpcwire.ContractProfile(frame), and every case is driven through the
// production Negotiate entry asserting the literal incompatible_protocol/6
// class, the contract_shape reason, and the local offer facts (which also
// closes the shape half of advisory N2).
func TestPeerOfferKeyMembershipPerKey(t *testing.T) {
	for _, frame := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		profile, err := rpcwire.ContractProfile(frame)
		if err != nil {
			t.Fatal(err)
		}
		config := generationForFrame(frame)
		// The expected local offer is literal, never derived from the
		// production LocalMajors under test: Config-4 offers exactly
		// RPC 5, Config-3 offers the legacy dual-stack set.
		wantLocal := []int{2, 3, 4}
		if frame == "5.0.0" {
			wantLocal = []int{5}
		}
		keys := make([]string, 0, len(profile))
		for key := range profile {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			t.Run("missing-"+frame+"-"+key, func(t *testing.T) {
				contracts := profileContracts(t, frame)
				delete(contracts, key)
				_, err := meshneg.Negotiate(config, frame, directHello(t, contracts))
				refusal := asRefusal(t, err)
				assertShapeRefusal(t, config, frame, refusal, err, wantLocal)
			})
			t.Run("substituted-"+frame+"-"+key, func(t *testing.T) {
				contracts := profileContracts(t, frame)
				delete(contracts, key)
				contracts["bogus"] = []string{"1.0.0"}
				_, err := meshneg.Negotiate(config, frame, directHello(t, contracts))
				refusal := asRefusal(t, err)
				assertShapeRefusal(t, config, frame, refusal, err, wantLocal)
			})
		}
	}
}

func assertShapeRefusal(t *testing.T, config, frame string, refusal *meshneg.Refusal, err error, wantLocal []int) {
	t.Helper()
	if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
		t.Fatalf("Negotiate(%s, %s) class = %s/%d, want incompatible_protocol/6",
			config, frame, refusal.Code, refusal.ExitCode)
	}
	if refusal.Reason != meshneg.ReasonContractShape {
		t.Fatalf("Negotiate(%s, %s) reason = %q", config, frame, refusal.Reason)
	}
	if !errors.Is(err, meshneg.ErrContractShape) {
		t.Fatalf("Negotiate(%s, %s) error does not wrap ErrContractShape", config, frame)
	}
	if !slices.Equal(refusal.Local, wantLocal) {
		t.Fatalf("Negotiate(%s, %s) refusal local = %v, want %v", config, frame, refusal.Local, wantLocal)
	}
	if refusal.Peer != 0 {
		t.Fatalf("Negotiate(%s, %s) refusal peer = %d, want 0", config, frame, refusal.Peer)
	}
}

// TestNegotiateOfferRefusalCarriesLocal closes the mixed half of inherited
// advisory N2: a mixed-major peer offer refused through Negotiate reports
// the local generation's exact offer set on the refusal.
func TestNegotiateOfferRefusalCarriesLocal(t *testing.T) {
	// Literal Config-3 offer set, never derived from production.
	wantLocal := []int{2, 3, 4}
	cases := []struct {
		name  string
		frame string
		rpc   []string
	}{
		{"higher-major-in-v2", "2.0.0", []string{"3.0.0"}},
		{"two-majors-in-v2", "2.0.0", []string{"2.0.0", "3.0.0"}},
		{"unsorted-v2", "2.0.0", []string{"2.1.0", "2.0.0"}},
		{"not-semver-v2", "2.0.0", []string{"v2.0.0"}},
		{"lower-major-in-v3", "3.0.0", []string{"2.0.0"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contracts := profileContracts(t, tc.frame)
			contracts["rpc"] = tc.rpc
			_, err := meshneg.Negotiate("3.0.0", tc.frame, directHello(t, contracts))
			refusal := asRefusal(t, err)
			if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
				t.Fatalf("class = %s/%d, want incompatible_protocol/6", refusal.Code, refusal.ExitCode)
			}
			if refusal.Reason != meshneg.ReasonMixedMajor {
				t.Fatalf("reason = %q, want mixed_major", refusal.Reason)
			}
			if !errors.Is(err, meshneg.ErrMixedMajor) {
				t.Fatalf("error does not wrap ErrMixedMajor: %v", err)
			}
			if !slices.Equal(refusal.Local, wantLocal) {
				t.Fatalf("refusal local = %v, want %v", refusal.Local, wantLocal)
			}
			if refusal.Peer != 0 {
				t.Fatalf("refusal peer = %d, want 0", refusal.Peer)
			}
		})
	}
}

// TestNoMajorSelectedByCoercion drives the six Section 17.1 no-coercion
// shapes through the composed Negotiate entry: a peer offering only a
// higher major, only a lower major, an empty rpc set, a duplicate rpc
// entry, an unsorted rpc array, and an out-of-range rpc count each produce
// the pinned refusal instead of a coerced selection.
func TestNoMajorSelectedByCoercion(t *testing.T) {
	t.Run("only-higher-major", func(t *testing.T) {
		hello := decodedHello(t, "5.0.0", profileContracts(t, "5.0.0"))
		_, err := meshneg.Negotiate("1.0.0", "5.0.0", hello)
		refusal := asRefusal(t, err)
		if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
			t.Fatalf("class = %s/%d, want incompatible_protocol/6", refusal.Code, refusal.ExitCode)
		}
		if refusal.Reason != meshneg.ReasonNoCommonMajor || !errors.Is(err, meshneg.ErrNoCommonMajor) {
			t.Fatalf("reason = %q, want no_common_major", refusal.Reason)
		}
		if refusal.Peer != 5 {
			t.Fatalf("refusal peer = %d, want 5", refusal.Peer)
		}
	})
	t.Run("only-lower-major", func(t *testing.T) {
		hello := decodedHello(t, "2.0.0", profileContracts(t, "2.0.0"))
		_, err := meshneg.Negotiate("4.0.0", "2.0.0", hello)
		refusal := asRefusal(t, err)
		if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
			t.Fatalf("class = %s/%d, want incompatible_protocol/6", refusal.Code, refusal.ExitCode)
		}
		if refusal.Reason != meshneg.ReasonNoCommonMajor || !errors.Is(err, meshneg.ErrNoCommonMajor) {
			t.Fatalf("reason = %q, want no_common_major", refusal.Reason)
		}
	})
	rpcShapes := []struct {
		name   string
		rpc    []string
		reason meshneg.RefusalReason
		cause  error
	}{
		{"empty-set", nil, meshneg.ReasonContractShape, meshneg.ErrContractShape},
		{"duplicate", []string{"2.0.0", "2.0.0"}, meshneg.ReasonMixedMajor, meshneg.ErrMixedMajor},
		{"unsorted", []string{"2.1.0", "2.0.0"}, meshneg.ReasonMixedMajor, meshneg.ErrMixedMajor},
		{"out-of-range-count", []string{
			"2.0.0", "2.0.1", "2.0.2", "2.0.3", "2.0.4", "2.0.5", "2.0.6",
			"2.0.7", "2.0.8", "2.0.9", "2.0.10", "2.0.11", "2.0.12",
			"2.0.13", "2.0.14", "2.0.15", "2.0.16",
		}, meshneg.ReasonContractShape, meshneg.ErrContractShape},
	}
	for _, tc := range rpcShapes {
		t.Run(tc.name, func(t *testing.T) {
			contracts := profileContracts(t, "2.0.0")
			contracts["rpc"] = tc.rpc
			_, err := meshneg.Negotiate("3.0.0", "2.0.0", directHello(t, contracts))
			refusal := asRefusal(t, err)
			if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
				t.Fatalf("class = %s/%d, want incompatible_protocol/6", refusal.Code, refusal.ExitCode)
			}
			if refusal.Reason != tc.reason || !errors.Is(err, tc.cause) {
				t.Fatalf("reason = %q, want %q", refusal.Reason, tc.reason)
			}
		})
	}
}

// TestMajor1RejectedThroughNegotiation proves a 2.0.0 peer rejects major 1
// at negotiation rather than reinterpreting it: a major-1 frame is an
// unsupported frame, and a major-1 rpc version inside a v2 frame is a
// mixed-major coercion attempt.
func TestMajor1RejectedThroughNegotiation(t *testing.T) {
	t.Run("frame-1.0.0", func(t *testing.T) {
		_, err := meshneg.Negotiate("3.0.0", "1.0.0", directHello(t, profileContracts(t, "2.0.0")))
		refusal := asRefusal(t, err)
		if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
			t.Fatalf("class = %s/%d, want incompatible_protocol/6", refusal.Code, refusal.ExitCode)
		}
		if refusal.Reason != meshneg.ReasonUnsupportedFrame || !errors.Is(err, meshneg.ErrUnsupportedFrame) {
			t.Fatalf("reason = %q, want unsupported_frame", refusal.Reason)
		}
	})
	t.Run("rpc-1.0.0-in-v2-frame", func(t *testing.T) {
		contracts := profileContracts(t, "2.0.0")
		contracts["rpc"] = []string{"1.0.0"}
		_, err := meshneg.Negotiate("3.0.0", "2.0.0", directHello(t, contracts))
		refusal := asRefusal(t, err)
		if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
			t.Fatalf("class = %s/%d, want incompatible_protocol/6", refusal.Code, refusal.ExitCode)
		}
		if refusal.Reason != meshneg.ReasonMixedMajor || !errors.Is(err, meshneg.ErrMixedMajor) {
			t.Fatalf("reason = %q, want mixed_major", refusal.Reason)
		}
	})
}
