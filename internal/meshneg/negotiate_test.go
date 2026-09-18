package meshneg_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/meshneg"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	requestID = "0198f4c8-a070-7188-9172-1234567890ab"
	peerHost  = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
)

func mustUint53(t *testing.T, value uint64) scalar.Uint53 {
	t.Helper()
	limit, err := scalar.NewUint53(value)
	if err != nil {
		t.Fatal(err)
	}
	return limit
}

// profileContracts returns a mutable copy of the pinned contracts map so
// table cases can tamper with it without affecting other cases.
func profileContracts(t *testing.T, frameVersion string) map[string][]string {
	t.Helper()
	profile, err := rpcwire.ContractProfile(frameVersion)
	if err != nil {
		t.Fatal(err)
	}
	return maps.Clone(profile)
}

// directHello builds a hello struct with valid identity members and the
// given contracts map. It bypasses rpcwire decode on purpose for the members
// PeerOffer re-validates (the key set and the rpc array); values of non-rpc
// arrays are validated by decode and trusted here, so directHello fixtures
// must keep those arrays at their pinned values (see
// TestPeerOfferTrustsDecodedNonRPCArrays for the pinned bound).
func directHello(t *testing.T, contracts map[string][]string) rpcwire.Hello {
	t.Helper()
	return rpcwire.Hello{
		HostID:         peerHost,
		Platform:       "macos",
		AXVersion:      "0.2.1",
		Nonce:          rpcwire.NewNonce(),
		Contracts:      contracts,
		MaxLineBytes:   mustUint53(t, 8*1024*1024),
		MaxObjectBytes: mustUint53(t, 5*1024*1024),
	}
}

// decodedHello runs the contracts map through the real rpcwire
// EncodeRequest/DecodeRequest/Request.Hello entries, proving the fixture is
// decode-valid before negotiation sees it.
func decodedHello(t *testing.T, frameVersion string, contracts map[string][]string) rpcwire.Hello {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"host_id":          peerHost,
		"platform":         "macos",
		"ax_version":       "0.2.1",
		"nonce":            rpcwire.NewNonce(),
		"contracts":        contracts,
		"max_line_bytes":   8 * 1024 * 1024,
		"max_object_bytes": 5 * 1024 * 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	line, err := rpcwire.EncodeRequest(frameVersion, requestID, "hello", body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		t.Fatal(err)
	}
	hello, err := request.Hello()
	if err != nil {
		t.Fatal(err)
	}
	return hello
}

func asRefusal(t *testing.T, err error) *meshneg.Refusal {
	t.Helper()
	if err == nil {
		t.Fatal("expected refusal, got nil error")
	}
	var refusal *meshneg.Refusal
	if !errors.As(err, &refusal) {
		t.Fatalf("expected *meshneg.Refusal, got %T (%v)", err, err)
	}
	return refusal
}

func TestLocalMajors(t *testing.T) {
	cases := []struct {
		config string
		majors []int
	}{
		{"1.0.0", []int{2}},
		{"2.0.0", []int{2, 3}},
		{"3.0.0", []int{2, 3, 4}},
		{"4.0.0", []int{5}},
	}
	for _, tc := range cases {
		t.Run("config-"+tc.config, func(t *testing.T) {
			majors, err := meshneg.LocalMajors(tc.config)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(majors, tc.majors) {
				t.Fatalf("LocalMajors(%q) = %v, want %v", tc.config, majors, tc.majors)
			}
			// The result must be a fresh slice: mutating it cannot change
			// a later call.
			majors[0] = 99
			again, err := meshneg.LocalMajors(tc.config)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(again, tc.majors) {
				t.Fatalf("LocalMajors(%q) shares its result slice: %v", tc.config, again)
			}
		})
	}
	unknown := []string{"", "5.0.0", "3.0", "v4.0.0", "4.0.1", "4", "1.0.0 ", "9.9.9"}
	for _, config := range unknown {
		t.Run("unknown-"+config, func(t *testing.T) {
			majors, err := meshneg.LocalMajors(config)
			if majors != nil {
				t.Fatalf("LocalMajors(%q) returned majors %v with an error", config, majors)
			}
			refusal := asRefusal(t, err)
			if refusal.Code != "invalid_config" || refusal.ExitCode != 3 {
				t.Fatalf("LocalMajors(%q) class = %s/%d, want invalid_config/3", config, refusal.Code, refusal.ExitCode)
			}
			if refusal.Reason != meshneg.ReasonUnknownConfig {
				t.Fatalf("LocalMajors(%q) reason = %q", config, refusal.Reason)
			}
			if !errors.Is(err, meshneg.ErrUnknownConfig) {
				t.Fatalf("LocalMajors(%q) error does not wrap ErrUnknownConfig", config)
			}
			if len(refusal.Local) != 0 || refusal.Peer != 0 {
				t.Fatalf("LocalMajors(%q) refusal carries offer facts %+v", config, refusal)
			}
		})
	}
}

// TestLocalMajorsTracksConfigVocabulary pins the negotiation generation
// input against the config owner: the generation is
// LoadedConfiguration.SourceVersion, and the LocalMajors vocabulary is the
// config Version1/Version2/CurrentVersion/Version4 vocabulary, so the two
// owners cannot drift silently. It also proves the wiring trap the doc
// comment warns about: Value.SchemaVersion is normalized to CurrentVersion
// for every v1/v2/v3 source, so feeding it to LocalMajors would offer RPC
// 3/4 to installations that have no directory or backend tables.
func TestLocalMajorsTracksConfigVocabulary(t *testing.T) {
	if config.Version1 != "1.0.0" || config.Version2 != "2.0.0" ||
		config.CurrentVersion != "3.0.0" || config.Version4 != "4.0.0" {
		t.Fatalf("config vocabulary drifted: %q %q %q %q",
			config.Version1, config.Version2, config.CurrentVersion, config.Version4)
	}
	want := map[string][]int{
		config.Version1:       {2},
		config.Version2:       {2, 3},
		config.CurrentVersion: {2, 3, 4},
		config.Version4:       {5},
	}
	for _, source := range []string{config.Version1, config.Version2, config.CurrentVersion, config.Version4} {
		majors, err := meshneg.LocalMajors(source)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(majors, want[source]) {
			t.Fatalf("LocalMajors(config %q) = %v, want %v", source, majors, want[source])
		}
	}
	context := config.DecodeContext{RuntimePlatform: scalar.PlatformMacOS}
	for _, source := range []string{config.Version1, config.Version2, config.CurrentVersion} {
		document := []byte(fmt.Sprintf(
			"schema = %q\nschema_version = %q\nhost_id = %q\nhost_name = %q\nplatform = %q\n",
			config.SchemaID, source, "0198f4c8-4a10-7b22-8b3c-1234567890ab", "fixture-host", "macos"))
		loaded, err := config.Decode(document, context)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.SourceVersion != source {
			t.Fatalf("source %s decoded SourceVersion = %q", source, loaded.SourceVersion)
		}
		if loaded.Value.SchemaVersion != config.CurrentVersion {
			t.Fatalf("source %s Value.SchemaVersion = %q, want normalized %q",
				source, loaded.Value.SchemaVersion, config.CurrentVersion)
		}
		majors, err := meshneg.LocalMajors(loaded.SourceVersion)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(majors, want[source]) {
			t.Fatalf("LocalMajors(SourceVersion %q) = %v, want %v", source, majors, want[source])
		}
		if source != config.CurrentVersion {
			trapped, err := meshneg.LocalMajors(loaded.Value.SchemaVersion)
			if err != nil {
				t.Fatal(err)
			}
			if slices.Equal(trapped, want[source]) {
				t.Fatalf("source %s: SchemaVersion input coincides with the SourceVersion mapping; the normalization trap is unproven", source)
			}
		}
	}
}

// TestNegotiateMatrix drives every local-generation by peer-major pair with
// decode-valid hellos through the production Negotiate entry.
func TestNegotiateMatrix(t *testing.T) {
	type expectation struct {
		major int // selected major; 0 means refused no-common-major
	}
	matrix := map[string]map[string]expectation{
		"1.0.0": {"2.0.0": {2}, "3.0.0": {}, "4.0.0": {}, "5.0.0": {}},
		"2.0.0": {"2.0.0": {2}, "3.0.0": {3}, "4.0.0": {}, "5.0.0": {}},
		"3.0.0": {"2.0.0": {2}, "3.0.0": {3}, "4.0.0": {4}, "5.0.0": {}},
		"4.0.0": {"2.0.0": {}, "3.0.0": {}, "4.0.0": {}, "5.0.0": {5}},
	}
	configs := []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"}
	frames := []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"}
	for _, config := range configs {
		for _, frame := range frames {
			want := matrix[config][frame]
			t.Run(config+"-x-"+frame, func(t *testing.T) {
				hello := decodedHello(t, frame, profileContracts(t, frame))
				decision, err := meshneg.Negotiate(config, frame, hello)
				if want.major == 0 {
					refusal := asRefusal(t, err)
					if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
						t.Fatalf("Negotiate(%s, %s) class = %s/%d, want incompatible_protocol/6",
							config, frame, refusal.Code, refusal.ExitCode)
					}
					if refusal.Reason != meshneg.ReasonNoCommonMajor {
						t.Fatalf("Negotiate(%s, %s) reason = %q", config, frame, refusal.Reason)
					}
					if !errors.Is(err, meshneg.ErrNoCommonMajor) {
						t.Fatalf("Negotiate(%s, %s) error does not wrap ErrNoCommonMajor", config, frame)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if decision.Major != want.major || decision.ProtocolVersion != frame {
					t.Fatalf("Negotiate(%s, %s) = major %d/%s, want %d/%s",
						config, frame, decision.Major, decision.ProtocolVersion, want.major, frame)
				}
			})
		}
	}
}

// TestNegotiateUnknownGeneration drives the production Negotiate entry with
// unknown local generations crossed with every pinned frame: Section 6.6
// refuses unknown configuration with invalid_config/exit 3 instead of
// selecting legacy, and the refusal carries a zero Decision. A Negotiate-site
// fallback to any legacy set (mutations.py negotiate-unknown-falls-back)
// must fail here.
func TestNegotiateUnknownGeneration(t *testing.T) {
	unknown := []string{"", "5.0.0", "9.9.9", "4.0.1", "3.0", "v4.0.0", "4", "1.0.0 "}
	for _, config := range unknown {
		for _, frame := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
			t.Run("unknown-"+config+"-x-"+frame, func(t *testing.T) {
				hello := decodedHello(t, frame, profileContracts(t, frame))
				decision, err := meshneg.Negotiate(config, frame, hello)
				refusal := asRefusal(t, err)
				if refusal.Code != "invalid_config" || refusal.ExitCode != 3 {
					t.Fatalf("Negotiate(%q, %s) class = %s/%d, want invalid_config/3",
						config, frame, refusal.Code, refusal.ExitCode)
				}
				if refusal.Reason != meshneg.ReasonUnknownConfig {
					t.Fatalf("Negotiate(%q, %s) reason = %q", config, frame, refusal.Reason)
				}
				if !errors.Is(err, meshneg.ErrUnknownConfig) {
					t.Fatalf("Negotiate(%q, %s) error does not wrap ErrUnknownConfig", config, frame)
				}
				if !reflect.DeepEqual(decision, meshneg.Decision{}) {
					t.Fatalf("Negotiate(%q, %s) returned %+v with a refusal", config, frame, decision)
				}
				if len(refusal.Local) != 0 || refusal.Peer != 0 {
					t.Fatalf("Negotiate(%q, %s) refusal carries offer facts %+v", config, frame, refusal)
				}
			})
		}
	}
}

func TestSelectHighestCommon(t *testing.T) {
	admitted := []struct {
		name  string
		local []int
		peer  []int
		want  int
	}{
		{"highest-wins", []int{2, 3, 4}, []int{2, 4}, 4},
		{"single-overlap", []int{2, 3}, []int{3, 4}, 3},
		{"config4-selects-5", []int{5}, []int{2, 3, 4, 5}, 5},
		{"legacy-peak", []int{2, 3, 4}, []int{4}, 4},
		{"core-only", []int{2}, []int{2}, 2},
		{"unordered-input", []int{4, 2, 3}, []int{3, 2}, 3},
	}
	for _, tc := range admitted {
		t.Run("admit-"+tc.name, func(t *testing.T) {
			got, err := meshneg.Select(tc.local, tc.peer)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("Select(%v, %v) = %d, want %d", tc.local, tc.peer, got, tc.want)
			}
		})
	}
	refused := []struct {
		name  string
		local []int
		peer  []int
	}{
		{"disjoint", []int{2}, []int{3}},
		{"legacy-against-5", []int{2, 3, 4}, []int{5}},
		{"config4-against-legacy", []int{5}, []int{2, 3, 4}},
		{"empty-local", nil, []int{2}},
		{"empty-peer", []int{2}, nil},
		{"local-names-6", []int{2, 6}, []int{2}},
		{"local-names-1", []int{1, 2}, []int{2}},
		{"peer-names-1", []int{2}, []int{1}},
		{"peer-mixes-1-with-common", []int{2}, []int{1, 2}},
		{"both-name-6", []int{6}, []int{6}},
		{"negative-major", []int{-2}, []int{-2}},
		{"zero-major", []int{0}, []int{2}},
	}
	for _, tc := range refused {
		t.Run("refuse-"+tc.name, func(t *testing.T) {
			got, err := meshneg.Select(tc.local, tc.peer)
			if got != 0 {
				t.Fatalf("Select(%v, %v) = %d, want refusal", tc.local, tc.peer, got)
			}
			refusal := asRefusal(t, err)
			if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
				t.Fatalf("Select(%v, %v) class = %s/%d, want incompatible_protocol/6",
					tc.local, tc.peer, refusal.Code, refusal.ExitCode)
			}
			if refusal.Reason != meshneg.ReasonNoCommonMajor {
				t.Fatalf("Select(%v, %v) reason = %q", tc.local, tc.peer, refusal.Reason)
			}
			if !errors.Is(err, meshneg.ErrNoCommonMajor) {
				t.Fatalf("Select(%v, %v) error does not wrap ErrNoCommonMajor", tc.local, tc.peer)
			}
		})
	}
}

func TestPeerOfferRefusals(t *testing.T) {
	v2 := func(t *testing.T) map[string][]string { return profileContracts(t, "2.0.0") }
	v3 := func(t *testing.T) map[string][]string { return profileContracts(t, "3.0.0") }

	t.Run("frame-versions", func(t *testing.T) {
		for _, frame := range []string{"", "1.0.0", "2.1.0", "3.0.1", "4.1.0", "5.0.1", "5.1.0", "6.0.0", "v2.0.0", "02.0.0", "2.0", "2.0.0 ", "garbage"} {
			hello := directHello(t, v2(t))
			peer, err := meshneg.PeerOffer(frame, hello)
			if peer != 0 {
				t.Fatalf("PeerOffer(%q) = %d, want refusal", frame, peer)
			}
			refusal := asRefusal(t, err)
			if refusal.Reason != meshneg.ReasonUnsupportedFrame || !errors.Is(err, meshneg.ErrUnsupportedFrame) {
				t.Fatalf("PeerOffer(%q) reason = %q", frame, refusal.Reason)
			}
			if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
				t.Fatalf("PeerOffer(%q) class = %s/%d", frame, refusal.Code, refusal.ExitCode)
			}
		}
	})

	t.Run("shape", func(t *testing.T) {
		cases := []struct {
			name      string
			frame     string
			contracts func(t *testing.T) map[string][]string
		}{
			{"v2-bound-in-v3-frame", "3.0.0", v2},
			{"v2-bound-in-v4-frame", "4.0.0", v2},
			{"v2-bound-in-v5-frame", "5.0.0", v2},
			{"v3-bound-in-v2-frame", "2.0.0", v3},
			{"extra-error-key", "2.0.0", func(t *testing.T) map[string][]string {
				m := v2(t)
				m["error"] = []string{"1.0.0"}
				return m
			}},
			{"extra-extension-key", "3.0.0", func(t *testing.T) map[string][]string {
				m := v3(t)
				m["extensions"] = []string{"1.0.0"}
				return m
			}},
			{"missing-key", "2.0.0", func(t *testing.T) map[string][]string {
				m := v2(t)
				delete(m, "lease")
				return m
			}},
			{"substituted-key", "2.0.0", func(t *testing.T) map[string][]string {
				m := v2(t)
				delete(m, "lease")
				m["bogus"] = []string{"1.0.0"}
				return m
			}},
			{"empty-rpc", "2.0.0", func(t *testing.T) map[string][]string {
				m := v2(t)
				m["rpc"] = nil
				return m
			}},
			{"seventeen-rpc", "2.0.0", func(t *testing.T) map[string][]string {
				m := v2(t)
				rpc := make([]string, 0, 17)
				for i := range 17 {
					rpc = append(rpc, "2.0."+string(rune('0'+i/10))+string(rune('0'+i%10)))
				}
				m["rpc"] = rpc
				return m
			}},
			{"v3-inexact-rpc", "3.0.0", func(t *testing.T) map[string][]string {
				m := v3(t)
				m["rpc"] = []string{"3.0.1"}
				return m
			}},
			{"v4-inexact-rpc", "4.0.0", func(t *testing.T) map[string][]string {
				m := profileContracts(t, "4.0.0")
				m["rpc"] = []string{"4.0.0", "4.1.0"}
				return m
			}},
			{"v5-inexact-rpc", "5.0.0", func(t *testing.T) map[string][]string {
				m := profileContracts(t, "5.0.0")
				m["rpc"] = []string{"5.0.0", "5.0.1"}
				return m
			}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				hello := directHello(t, tc.contracts(t))
				peer, err := meshneg.PeerOffer(tc.frame, hello)
				if peer != 0 {
					t.Fatalf("PeerOffer(%s, %s) = %d, want refusal", tc.frame, tc.name, peer)
				}
				refusal := asRefusal(t, err)
				if refusal.Reason != meshneg.ReasonContractShape || !errors.Is(err, meshneg.ErrContractShape) {
					t.Fatalf("PeerOffer(%s, %s) reason = %q", tc.frame, tc.name, refusal.Reason)
				}
			})
		}
	})

	t.Run("mixed", func(t *testing.T) {
		cases := []struct {
			name  string
			frame string
			rpc   []string
		}{
			{"two-majors-in-v2", "2.0.0", []string{"2.0.0", "3.0.0"}},
			{"v3-only-in-v2-frame", "2.0.0", []string{"3.0.0"}},
			{"v1-in-v2-frame", "2.0.0", []string{"1.0.0"}},
			{"v2-in-v3-frame", "3.0.0", []string{"2.0.0"}},
			{"v5-in-v4-frame", "4.0.0", []string{"5.0.0"}},
			{"duplicate", "2.0.0", []string{"2.0.0", "2.0.0"}},
			{"unsorted", "2.0.0", []string{"2.1.0", "2.0.0"}},
			{"not-semver", "2.0.0", []string{"v2.0.0"}},
			{"non-numeric-major", "2.0.0", []string{"99999999999999999999999.0.0"}},
			{"empty-string", "2.0.0", []string{""}},
			{"leading-zero", "2.0.0", []string{"02.0.0"}},
			{"trailing-space", "2.0.0", []string{"2.0.0 "}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				m := profileContracts(t, tc.frame)
				m["rpc"] = tc.rpc
				hello := directHello(t, m)
				peer, err := meshneg.PeerOffer(tc.frame, hello)
				if peer != 0 {
					t.Fatalf("PeerOffer(%s, %s) = %d, want refusal", tc.frame, tc.name, peer)
				}
				refusal := asRefusal(t, err)
				if refusal.Reason != meshneg.ReasonMixedMajor || !errors.Is(err, meshneg.ErrMixedMajor) {
					t.Fatalf("PeerOffer(%s, %s) reason = %q", tc.frame, tc.name, refusal.Reason)
				}
				if refusal.Code != "incompatible_protocol" || refusal.ExitCode != 6 {
					t.Fatalf("PeerOffer(%s, %s) class = %s/%d", tc.frame, tc.name, refusal.Code, refusal.ExitCode)
				}
			})
		}
	})

	t.Run("admitted-ranges", func(t *testing.T) {
		// A v2 minor range stays major 2: minor selection within the
		// common major is a stated bound, not a refusal.
		m := v2(t)
		m["rpc"] = []string{"2.0.0", "2.1.0"}
		if peer, err := meshneg.PeerOffer("2.0.0", directHello(t, m)); peer != 2 || err != nil {
			t.Fatalf("PeerOffer(v2 range) = %d, %v, want 2, nil", peer, err)
		}
		m = v2(t)
		m["rpc"] = []string{"2.0.0", "2.0.0-alpha"}
		if peer, err := meshneg.PeerOffer("2.0.0", directHello(t, m)); peer != 2 || err != nil {
			t.Fatalf("PeerOffer(v2 prerelease) = %d, %v, want 2, nil", peer, err)
		}
		for _, frame := range []string{"3.0.0", "4.0.0", "5.0.0"} {
			peer, err := meshneg.PeerOffer(frame, directHello(t, profileContracts(t, frame)))
			if peer != int(frame[0]-'0') || err != nil {
				t.Fatalf("PeerOffer(%s) = %d, %v", frame, peer, err)
			}
		}
	})
}

// TestPeerOfferTrustsDecodedNonRPCArrays pins the stated bound: PeerOffer
// re-validates the contracts key set and the rpc array only. A directly
// built v3/v4/v5 hello with a wrong non-rpc array is admitted here, while
// the same map is refused by rpcwire encode/decode, so the composed
// production path (Request.Hello / Response.Hello are the only production
// constructors) closes the gap. A v2 hello with lease ["9.9.9"] is admitted
// by both: v2 permits 1-16 versions per key by design, and minor selection
// is a separate stated bound.
func TestPeerOfferTrustsDecodedNonRPCArrays(t *testing.T) {
	for _, frame := range []string{"3.0.0", "4.0.0", "5.0.0"} {
		t.Run("tampered-"+frame, func(t *testing.T) {
			contracts := profileContracts(t, frame)
			contracts["session_event"] = []string{"1.0.0"}
			peer, err := meshneg.PeerOffer(frame, directHello(t, contracts))
			if peer != int(frame[0]-'0') || err != nil {
				t.Fatalf("PeerOffer(%s, tampered session_event) = %d, %v: bound moved", frame, peer, err)
			}
			body, err := json.Marshal(map[string]any{
				"host_id": peerHost, "platform": "macos", "ax_version": "0.2.1",
				"nonce": rpcwire.NewNonce(), "contracts": contracts,
				"max_line_bytes": 8 * 1024 * 1024, "max_object_bytes": 5 * 1024 * 1024,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := rpcwire.EncodeRequest(frame, requestID, "hello", body); !errors.Is(err, rpcwire.ErrHello) {
				t.Fatalf("rpcwire admitted the tampered %s map: %v (bound moved)", frame, err)
			}
		})
	}
	t.Run("v2-range-admitted-by-both", func(t *testing.T) {
		contracts := profileContracts(t, "2.0.0")
		contracts["lease"] = []string{"9.9.9"}
		peer, err := meshneg.PeerOffer("2.0.0", directHello(t, contracts))
		if peer != 2 || err != nil {
			t.Fatalf("PeerOffer(v2 lease range) = %d, %v, want 2, nil", peer, err)
		}
		hello := decodedHello(t, "2.0.0", contracts)
		peer, err = meshneg.PeerOffer("2.0.0", hello)
		if peer != 2 || err != nil {
			t.Fatalf("PeerOffer(decoded v2 lease range) = %d, %v, want 2, nil", peer, err)
		}
	})
}

func TestDecisionShapes(t *testing.T) {
	type shape struct {
		keys       int
		namespaces []string
		errVersion axerror.Version
	}
	sorted := func(names ...string) []string {
		out := slices.Clone(names)
		slices.Sort(out)
		return out
	}
	core := []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"}
	shapes := map[string]shape{
		"2.0.0": {14, sorted(core...), axerror.Version100},
		"3.0.0": {24, sorted(append(slices.Clone(core), "directory_record")...), axerror.Version120},
		"4.0.0": {25, sorted(append(slices.Clone(core), "directory_record", "terminal_backend_evidence")...), axerror.Version130},
		"5.0.0": {25, sorted(append(slices.Clone(core), "directory_record", "terminal_backend_evidence")...), axerror.Version130},
	}
	// Local generations that admit each frame: config-4 only for 5.0.0,
	// the newest legacy generation otherwise.
	configFor := map[string]string{"2.0.0": "3.0.0", "3.0.0": "3.0.0", "4.0.0": "3.0.0", "5.0.0": "4.0.0"}
	for _, frame := range []string{"2.0.0", "3.0.0", "4.0.0", "5.0.0"} {
		t.Run(frame, func(t *testing.T) {
			config := configFor[frame]
			hello := decodedHello(t, frame, profileContracts(t, frame))
			decision, err := meshneg.Negotiate(config, frame, hello)
			if err != nil {
				t.Fatal(err)
			}
			want := shapes[frame]
			if len(decision.Contracts) != want.keys {
				t.Fatalf("Negotiate(%s, %s) contracts keys = %d, want %d", config, frame, len(decision.Contracts), want.keys)
			}
			profile, err := rpcwire.ContractProfile(frame)
			if err != nil {
				t.Fatal(err)
			}
			if !maps.EqualFunc(decision.Contracts, profile, slices.Equal) {
				t.Fatalf("Negotiate(%s, %s) contracts differ from the rpcwire profile", config, frame)
			}
			if !slices.Equal(decision.Namespaces, want.namespaces) {
				t.Fatalf("Negotiate(%s, %s) namespaces = %v, want %v", config, frame, decision.Namespaces, want.namespaces)
			}
			if decision.ErrorVersion != want.errVersion {
				t.Fatalf("Negotiate(%s, %s) error version = %s, want %s", config, frame, decision.ErrorVersion, want.errVersion)
			}
			if decision.ProtocolVersion != frame {
				t.Fatalf("Negotiate(%s, %s) protocol version = %s", config, frame, decision.ProtocolVersion)
			}
			// RPC 5 carries the exact RPC-4 shapes with only the version
			// changed (Section 11.10.1).
			if frame == "5.0.0" {
				four, err := meshneg.Negotiate("3.0.0", "4.0.0", decodedHello(t, "4.0.0", profileContracts(t, "4.0.0")))
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(decision.Namespaces, four.Namespaces) || decision.ErrorVersion != four.ErrorVersion {
					t.Fatal("RPC 5 decision diverges from the RPC-4 shapes")
				}
				if len(decision.Contracts) != len(four.Contracts) {
					t.Fatal("RPC 5 contracts key count diverges from RPC 4")
				}
				for key, versions := range four.Contracts {
					got := decision.Contracts[key]
					if key == "rpc" {
						if !slices.Equal(got, []string{"5.0.0"}) {
							t.Fatalf("RPC 5 rpc contracts = %v", got)
						}
						continue
					}
					if !slices.Equal(got, versions) {
						t.Fatalf("RPC 5 contracts[%q] diverges from RPC 4", key)
					}
				}
			}
		})
	}
}
