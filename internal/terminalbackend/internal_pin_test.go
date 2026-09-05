// White-box pins for refusal arms no production entry point can deliver an
// input to. RegisterExternal refuses any non-external kind as untrusted
// before record validation runs (pinned black-box by
// TestRegisterExternalRefusesUnknownKindAsUntrusted), and New constructs
// only builtin_go records with a null digest, so parseKind's vocabulary arm
// and the digest-must-be-null arm are defense-in-depth for future
// constructors. They are measured here, at the guard, rather than left as
// prose; the reachability bound is stated, not inferred.
package terminalbackend

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestParseKindAdmitsOnlyTheClosedVocabulary sweeps the kind domain: every
// member outside builtin_go, local_program, trusted_executable and
// native_runtime is refused.
func TestParseKindAdmitsOnlyTheClosedVocabulary(t *testing.T) {
	t.Parallel()

	for _, kind := range []Kind{KindBuiltinGo, KindLocalProgram, KindTrustedExecutable, KindNativeRuntime} {
		if _, err := parseKind(string(kind)); err != nil {
			t.Errorf("parseKind(%q) error = %v, want admission", kind, err)
		}
	}
	for _, kind := range []string{
		"", "builtin", "BUILTIN_GO", "Builtin_Go", "trusted-executable",
		"local program", "nativeRuntime", "ax.tmux", "bogus_kind",
		strings.Repeat("x", 1024),
	} {
		if _, err := parseKind(kind); !IsNotFound(err) {
			t.Errorf("parseKind(%q) error = %v, want terminal_backend_not_found", kind, err)
		}
	}
}

// TestValidateRefusesBuiltinKindWithDigest pins the digest-must-be-null arm:
// a built-in kind carrying an executable digest is drift, even though neither
// New (which builds null digests) nor RegisterExternal (which refuses
// non-external kinds first) can deliver such a record today.
func TestValidateRefusesBuiltinKindWithDigest(t *testing.T) {
	t.Parallel()

	record := Registration{
		ID:                    "com.example.term",
		Kind:                  KindBuiltinGo,
		ImplementationVersion: "1.2.3",
		ProtocolVersions:      []string{"1.0.0"},
		Platforms:             []scalar.Platform{scalar.PlatformLinux},
		ExecutableDigest:      "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}
	err := record.validate()
	if !IsDrift(err) {
		t.Fatalf("validate() error = %v, want terminal_backend_implementation_drift", err)
	}
	var refusal *Error
	if !errors.As(err, &refusal) || refusal.Detail != "executable_digest must be null" {
		t.Errorf("validate() detail = %v, want executable_digest must be null", err)
	}
}

// TestValidatePlatformsReportsTheBoundFirst pins the length arm at the unit
// level: a five-member list is refused for its length even though any such
// list over the four-member vocabulary would also fail a later rule.
func TestValidatePlatformsReportsTheBoundFirst(t *testing.T) {
	t.Parallel()

	err := validatePlatforms([]scalar.Platform{
		scalar.PlatformLinux, scalar.PlatformLinux,
		scalar.PlatformMacOS, scalar.PlatformWindows, scalar.PlatformWSL2,
	})
	if err == nil || err.Error() != "platforms bound" {
		t.Errorf("validatePlatforms(5 members) error = %v, want platforms bound", err)
	}
	if err := validatePlatforms(nil); err == nil || err.Error() != "platforms bound" {
		t.Errorf("validatePlatforms(nil) error = %v, want platforms bound", err)
	}
}

// TestReplicationMembersAreExactlyTheClosedTable pins the whole §4.E
// classification table from inside the package, in both directions: the
// live replicationMembers map must equal the exact member set below, so
// reclassifying a member reddens on the class and adding a member
// reddens on the size. The black-box TestReplicationClassificationIsClosed
// proves want ⊆ production through the public ClassifyReplication entry;
// this test proves production ⊆ want (including members injected outside
// the table builder, such as an init-time assignment) and pins the size,
// which no entry-point iteration can observe. A new replicable member
// joins here deliberately or not at all.
func TestReplicationMembersAreExactlyTheClosedTable(t *testing.T) {
	t.Parallel()

	want := map[string]ReplicationClass{
		"manifest_id": "safe_evidence", "probe_id": "safe_evidence",
		"evidence_id":               "safe_evidence",
		"terminal_backend_id":       "safe_evidence",
		"implementation_version":    "safe_evidence",
		"protocol_version":          "safe_evidence",
		"protocol_versions":         "safe_evidence",
		"platform":                  "safe_evidence",
		"platforms":                 "safe_evidence",
		"os_version":                "safe_evidence",
		"conformance_fixture_id":    "safe_evidence",
		"capability_claims":         "safe_evidence",
		"evidence_ids":              "safe_evidence",
		"backend_generation_digest": "safe_evidence",
		"capability":                "safe_evidence",
		"dependent_operations":      "safe_evidence",
		"evidence_requirements":     "safe_evidence",
		"facts":                     "safe_evidence",
		"issuer":                    "safe_evidence",
		"issuer_id":                 "safe_evidence",
		"observed_at":               "safe_evidence",
		"expires_at":                "safe_evidence",

		"binding_id": "host_local", "terminal_instance_id": "host_local",
		"session_id": "host_local", "host_id": "host_local",
		"host_incarnation_id":   "host_local",
		"backend_generation":    "host_local",
		"native_reference":      "host_local",
		"idempotency_receipt":   "host_local",
		"last_effect":           "host_local",
		"last_operation_id":     "host_local",
		"supersedes_binding_id": "host_local",
		"created_at":            "host_local",

		"availability": "derived_cache", "attachable": "derived_cache",
		"wrapper_present": "derived_cache", "provider_present": "derived_cache",
		"identity_match": "derived_cache", "state": "derived_cache",
		"probed_at": "derived_cache",

		"attach_descriptor":   "sensitive_owner_only",
		"ipc_handle":          "sensitive_owner_only",
		"tmux_socket":         "sensitive_owner_only",
		"named_pipe":          "sensitive_owner_only",
		"attach_credential":   "sensitive_owner_only",
		"relay_credential":    "sensitive_owner_only",
		"backend_auth":        "sensitive_owner_only",
		"auth_database":       "sensitive_owner_only",
		"gui_attestation":     "sensitive_owner_only",
		"login_attestation":   "sensitive_owner_only",
		"provider_credential": "sensitive_owner_only",
		"evidence_secret":     "sensitive_owner_only",
		"environment_values":  "sensitive_owner_only",

		"native_pid": "forbidden", "process_handle": "forbidden",
		"endpoint": "forbidden", "token": "forbidden",
		"terminal_output": "forbidden", "scrollback": "forbidden",
		"unrestricted_environment": "forbidden",
		"credential_detail":        "forbidden",
		"live_process_fact":        "forbidden",
		"raw_native_reference":     "forbidden",
	}
	if len(replicationMembers) != len(want) {
		t.Fatalf("replicationMembers holds %d members, want exactly %d; "+
			"a new replicable member joins deliberately or not at all",
			len(replicationMembers), len(want))
	}
	for member, class := range want {
		got, known := replicationMembers[member]
		if !known {
			t.Errorf("replicationMembers lacks %q, want %q", member, class)
			continue
		}
		if got != class {
			t.Errorf("replicationMembers[%q] = %q, want %q", member, got, class)
		}
	}
	for member, class := range replicationMembers {
		wantClass, listed := want[member]
		if !listed {
			t.Errorf("replicationMembers holds unlisted %q (%q); "+
				"an unlisted member replicating by default widens the §4.E exclusion", member, class)
			continue
		}
		if class != wantClass {
			t.Errorf("replicationMembers[%q] = %q, want %q", member, class, wantClass)
		}
	}
}

// TestLegacyForwardIsExactlyTheImmutablePair pins the whole §4.E forward
// map from inside the package, in both directions plus the size: the live
// legacyForward map must equal exactly {tmux, conpty}, so a third escape
// name (C20/L3: "screen") reddens on the size no matter how map iteration
// orders the reverse projection, and a removed or repointed row reddens on
// the class. The black-box TestHistoricalTranslationMapsOnlyTheImmutablePair
// proves want ⊆ production through TranslateLegacyBackend; this test proves
// production ⊆ want. It also pins injectivity: the two canonical values
// must be distinct, because ProjectToLegacy documents a deterministic
// reverse projection that a colliding value would silently break.
func TestLegacyForwardIsExactlyTheImmutablePair(t *testing.T) {
	t.Parallel()

	if BuiltinTmux == BuiltinConpty {
		t.Fatalf("built-in canonical IDs collide at %q: the reverse projection cannot be deterministic", BuiltinTmux)
	}
	want := map[string]string{
		"tmux":   BuiltinTmux,
		"conpty": BuiltinConpty,
	}
	if len(legacyForward) != len(want) {
		t.Fatalf("legacyForward holds %d names, want exactly %d; "+
			"a third historical escape name translates silently or not at all",
			len(legacyForward), len(want))
	}
	for legacy, canonical := range want {
		got, known := legacyForward[legacy]
		if !known {
			t.Errorf("legacyForward lacks %q, want %q", legacy, canonical)
			continue
		}
		if got != canonical {
			t.Errorf("legacyForward[%q] = %q, want %q", legacy, got, canonical)
		}
	}
	seen := make(map[string]string, len(legacyForward))
	for legacy, canonical := range legacyForward {
		wantCanonical, listed := want[legacy]
		if !listed {
			t.Errorf("legacyForward holds unlisted %q (%q); "+
				"an unlisted legacy name translating by default widens the §4.E pair", legacy, canonical)
			continue
		}
		if canonical != wantCanonical {
			t.Errorf("legacyForward[%q] = %q, want %q", legacy, canonical, wantCanonical)
		}
		if holder, duplicate := seen[canonical]; duplicate {
			t.Errorf("legacyForward maps both %q and %q to %q: "+
				"the reverse projection is no longer injective", holder, legacy, canonical)
			continue
		}
		seen[canonical] = legacy
	}
}

// TestBuiltinsHoldIndependentProtocolArrays observes the registry's own
// stored state instead of going through the cloning Resolve accessor. The
// black-box subtest this replaces reached the built-ins through Resolve,
// which clones on egress, so it mutated a copy and passed even with the New
// clones reverted (both built-ins sharing one backing array). Comparing the
// first-element addresses of the stored slices fails exactly then: reverting
// both New clones to the shared sorted slice leaves the suite red here.
func TestBuiltinsHoldIndependentProtocolArrays(t *testing.T) {
	t.Parallel()

	registry, err := New("1.2.3", []string{"1.0.0"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	tmux, known := registry.records[BuiltinTmux]
	if !known {
		t.Fatalf("ax.tmux not admitted")
	}
	conpty, known := registry.records[BuiltinConpty]
	if !known {
		t.Fatalf("ax.conpty not admitted")
	}
	if len(tmux.ProtocolVersions) == 0 || len(conpty.ProtocolVersions) == 0 {
		t.Fatalf("admitted built-ins carry no protocol versions")
	}
	if &tmux.ProtocolVersions[0] == &conpty.ProtocolVersions[0] {
		t.Errorf("ax.tmux and ax.conpty share one protocol backing array")
	}
}

// TestSurrogateGateAgreesWithCanonicalJSON pins the local surrogate scan
// against the canonical owner: hasLoneSurrogateEscape and
// canonicaljson.Canonicalize agree accept/reject on every shared vector, so
// the two copies cannot drift without reddening here.
//
// The corpus is derived, not hand-written: it enumerates every escaped code
// point in the surrogate range U+D800..U+DFFF in both escape positions,
// plus a representative set outside the range and the raw WTF-8 road. A
// hand-written vector set cannot tell a right gate from a bound-narrowed
// one: narrowing the low-surrogate upper bound to <= 0xDC00 survived a
// 10-vector corpus 6 of 6 while admitting 1023 lone low surrogates. Every
// loop below kills that mutant (lone lows 0xDC01..0xDFFF must reject, and a
// high followed by 0xDC01..0xDFFF must reject), and the mirrored loops kill
// a high-surrogate bound narrowing the same way. A delete-only mutant proves
// the gate exists; these sweeps prove the class it covers.
//
// Vectors are bare JSON strings where the surrogate rule is the only
// distinguishing rule; the escape introducer is assembled from its byte
// value because no source literal spells it next to hex. Stated bound:
// malformed escapes (a bad hex digit, an uppercase U introducer, a
// truncated tail), truncated WTF-8 tails, WTF-8 outside string literals,
// other invalid UTF-8 (overlongs, 0xFF), and unescaped control characters
// are refused by both sides' syntax or encoding arms, not by the surrogate
// question, and are outside this pin.
func TestSurrogateGateAgreesWithCanonicalJSON(t *testing.T) {
	t.Parallel()

	slash := string([]byte{92})
	escape := func(unit uint16) string {
		const digits = "0123456789abcdef"
		return slash + "u" + string([]byte{
			digits[unit>>12&0xf], digits[unit>>8&0xf], digits[unit>>4&0xf], digits[unit&0xf],
		})
	}
	var failures []string
	check := func(doc string, reject bool) {
		raw := []byte(doc)
		if local := hasLoneSurrogateEscape(raw); local != reject {
			failures = append(failures, fmt.Sprintf("hasLoneSurrogateEscape(%q) = %v, want rejection %v",
				doc, local, reject))
		}
		_, err := canonicaljson.Canonicalize(raw)
		if rejected := err != nil; rejected != reject {
			failures = append(failures, fmt.Sprintf("Canonicalize(%q) error = %v, want rejection %v",
				doc, err, reject))
		}
	}

	// Every code point in the surrogate range as a lone escape rejects.
	for unit := uint32(0xd800); unit <= 0xdfff; unit++ {
		check(`"`+escape(uint16(unit))+`"`, true)
	}
	// A fixed high surrogate followed by every code point in the surrogate
	// range admits exactly the low surrogates.
	for second := uint32(0xd800); second <= 0xdfff; second++ {
		check(`"`+escape(0xd800)+escape(uint16(second))+`"`, second < 0xdc00)
	}
	// Valid pairs at the opposite corner admit.
	check(`"`+escape(0xdbff)+escape(0xdfff)+`"`, false)
	// Representative escapes outside the surrogate range admit alone and
	// reject after a high surrogate.
	for _, unit := range []uint16{0x0000, 0x0041, 0x00e9, 0xd7ff, 0xe000, 0xffff} {
		check(`"`+escape(unit)+`"`, false)
		check(`"`+escape(0xd800)+escape(unit)+`"`, true)
	}
	// Uppercase hex is the same code point, not a different question.
	check(`"`+slash+`uDC00"`, true)
	check(`"`+slash+`uD800`+slash+`uDC00"`, false)
	// An escaped backslash is not an escape introducer.
	check(`"`+slash+slash+`ud800"`, false)
	// Plain and multibyte strings admit.
	check(`"abc"`, false)
	check(`"aé"`, false)

	// Raw WTF-8 (CESU-8) surrogate encodings reject on both sides; the
	// neighboring valid code point U+D7FF (ED 9F BF) admits on both.
	for _, bytes := range [][]byte{
		{0xed, 0xa0, 0x80}, // U+D800
		{0xed, 0xb0, 0x80}, // U+DC00
		{0xed, 0xbf, 0xbf}, // U+DFFF
		{0xed, 0xaf, 0x93},
	} {
		doc := append(append([]byte{'"'}, bytes...), '"')
		check(string(doc), true)
	}
	check("\""+string([]byte{0xed, 0x9f, 0xbf})+"\"", false)
	// A WTF-8 head immediately after a backslash escape is still raw bytes
	// on the wire: the backslash skip must not hide it.
	check(`"`+slash+`n`+string([]byte{0xed, 0xa0, 0x80})+`"`, true)
	check(`"`+slash+slash+string([]byte{0xed, 0xbf, 0xbf})+`"`, true)

	if len(failures) > 0 {
		t.Fatalf("%d agreement failures (showing %d):\n%s",
			len(failures), min(len(failures), 10), strings.Join(failures[:min(len(failures), 10)], "\n"))
	}
}
