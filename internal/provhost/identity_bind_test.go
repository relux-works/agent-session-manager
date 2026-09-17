package provhost

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// This file proves the identity_bind.go production entries: the
// NativeDiscoveryProof decoder, the Section 8.2 store-root table, the
// Section 8.4 resume-tuple gate, and the build and discovery binding
// verifiers. Every refusal below drives its production entry; the
// arm witnesses in refusal_arm_operations_d_test.go pin each derived
// arm to one of these entries.

const (
	bindSessionID   = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	bindWorkspaceID = "0198f4c8-6c30-7d44-8d5e-1234567890ab"
	bindHostID      = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	bindNativeID    = "11111111-2222-4333-8444-555555555555"
	bindRealm       = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"
	bindHome        = "/Users/iv"
)

// validBuildTuple returns the codex macOS arm64 tuple the binding
// tests correlate against created codex identities.
func validBuildTuple() BuildTuple {
	return BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
}

// bindIdentity creates the codex identity matching validBuildTuple.
func bindIdentity(t *testing.T) []byte {
	t.Helper()
	params := validCreateParams()
	params.SessionID = bindSessionID
	params.LogicalWorkspaceID = bindWorkspaceID
	params.CreatedByHostID = bindHostID
	params.NativeSessionID = bindNativeID
	return mustCreate(t, params)
}

// bindProof renders one discovery proof body for the tests below.
func bindProof(native, root string, nullRoot, discovered, backendResolved bool) []byte {
	rootLiteral := "null"
	if !nullRoot {
		rootLiteral = `"` + root + `"`
	}
	boolLiteral := func(value bool) string {
		if value {
			return "true"
		}
		return "false"
	}
	return []byte(`{"native_session_id": "` + native + `", "discovered": ` + boolLiteral(discovered) + `, "discovery_root": ` + rootLiteral + `, "backend_resolved": ` + boolLiteral(backendResolved) + `}`)
}

// validDiscoveryContext returns the context matching validBuildTuple
// under bindHome with no expected realm and no override.
func validDiscoveryContext() DiscoveryContext {
	return DiscoveryContext{Build: validBuildTuple(), Home: bindHome}
}

// TestDecodeNativeDiscoveryPositive proves both proof shapes decode:
// a null root for backend-resolved discovery and an absolute root
// for store discovery.
func TestDecodeNativeDiscoveryPositive(t *testing.T) {
	proof, err := DecodeNativeDiscovery(bindProof(bindNativeID, "", true, true, true), "macos")
	if err != nil {
		t.Fatalf("null-root proof: %v", err)
	}
	if proof.NativeSessionID != bindNativeID || !proof.Discovered || proof.HasDiscoveryRoot || !proof.BackendResolved {
		t.Fatalf("null-root proof decodes to %+v", proof)
	}
	proof, err = DecodeNativeDiscovery(bindProof(bindNativeID, "/Users/iv/.codex/sessions", false, true, false), "macos")
	if err != nil {
		t.Fatalf("rooted proof: %v", err)
	}
	if proof.NativeSessionID != bindNativeID || !proof.Discovered || !proof.HasDiscoveryRoot || proof.DiscoveryRoot != "/Users/iv/.codex/sessions" || proof.BackendResolved {
		t.Fatalf("rooted proof decodes to %+v", proof)
	}
}

// TestDecodeNativeDiscoveryRefusals drives every proof-decoder refusal
// through the production entry.
func TestDecodeNativeDiscoveryRefusals(t *testing.T) {
	rows := []struct {
		name     string
		body     string
		platform string
		member   string
		detail   string
	}{
		{"unknown member", `{"native_session_id": "x", "discovered": true, "discovery_root": null, "backend_resolved": false, "score": 1}`, "macos", "score", "unknown member"},
		{"missing member", `{"native_session_id": "x", "discovered": true, "backend_resolved": false}`, "macos", "discovery_root", "misses a required member"},
		{"native bounds", `{"native_session_id": "", "discovered": true, "discovery_root": null, "backend_resolved": false}`, "macos", "native_session_id", "not 1..512 characters"},
		{"discovered shape", `{"native_session_id": "x", "discovered": "yes", "discovery_root": null, "backend_resolved": false}`, "macos", "discovered", "not a boolean"},
		{"root shape", `{"native_session_id": "x", "discovered": true, "discovery_root": 7, "backend_resolved": false}`, "macos", "discovery_root", "not an absolute path or null"},
		{"root relative", `{"native_session_id": "x", "discovered": true, "discovery_root": "sessions/x", "backend_resolved": false}`, "macos", "discovery_root", "not an absolute path or null"},
		{"root relative admitted-token", `{"native_session_id": "x", "discovered": true, "discovery_root": "sessions/admitted", "backend_resolved": false}`, "macos", "discovery_root", "not an absolute path or null"},
		{"backend shape", `{"native_session_id": "x", "discovered": true, "discovery_root": null, "backend_resolved": "no"}`, "macos", "backend_resolved", "not a boolean"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, err := DecodeNativeDiscovery([]byte(row.body), row.platform)
			requireFrameRefusal(t, err, row.member, row.detail)
		})
	}
	t.Run("platform registry", func(t *testing.T) {
		_, err := DecodeNativeDiscovery(bindProof(bindNativeID, "", true, true, false), "plan9")
		requireLocalRefusal(t, err, "invalid_config", "discovery platform is not a registry member")
	})
	t.Run("malformed body", func(t *testing.T) {
		_, err := DecodeNativeDiscovery([]byte(`{"native_session_id":`), "macos")
		requireFrameRefusal(t, err, "native_session_id", "not a JSON object")
	})
}

// TestStoreRootForTable proves the Section 8.2 documented roots
// resolve per provider, the Muse XDG selection, and the Antigravity
// backend-only row, which carries no filesystem root.
func TestStoreRootForTable(t *testing.T) {
	rows := []struct {
		provider string
		xdg      string
		want     string
		backend  bool
	}{
		{"codex", "", filepath.Join(bindHome, ".codex", "sessions"), false},
		{"claude", "", filepath.Join(bindHome, ".claude", "projects"), false},
		{"gemini", "", filepath.Join(bindHome, ".gemini"), false},
		{"muse", "", filepath.Join(bindHome, ".local", "share", "muse", "sessions"), false},
		{"muse", "/data/xdg", filepath.Join("/data/xdg", "muse", "sessions"), false},
		{"pi", "", filepath.Join(bindHome, ".pi", "agent", "sessions"), false},
		{"antigravity", "", "", true},
	}
	for _, row := range rows {
		t.Run(row.provider+"/xdg="+row.xdg, func(t *testing.T) {
			root, backendOnly, err := StoreRootFor(row.provider, bindHome, row.xdg)
			if err != nil {
				t.Fatalf("StoreRootFor(%q): %v", row.provider, err)
			}
			if backendOnly != row.backend || root != row.want {
				t.Fatalf("StoreRootFor(%q) = (%q, %v), want (%q, %v)", row.provider, root, backendOnly, row.want, row.backend)
			}
		})
	}
}

// TestStoreRootsArePairwiseDistinct proves no provider silently
// aliases onto another provider's root: every filesystem provider
// resolves a distinct absolute root.
func TestStoreRootsArePairwiseDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, provider := range resumeProviders {
		root, backendOnly, err := StoreRootFor(provider, bindHome, "")
		if err != nil {
			t.Fatalf("StoreRootFor(%q): %v", provider, err)
		}
		if backendOnly {
			if provider != "antigravity" {
				t.Fatalf("StoreRootFor(%q) reports backend-only, want only antigravity", provider)
			}
			continue
		}
		if !isBareAbsolute(root) {
			t.Fatalf("StoreRootFor(%q) = %q, want an absolute root", provider, root)
		}
		if other, repeated := seen[root]; repeated {
			t.Fatalf("StoreRootFor(%q) aliases the %s root %q", provider, other, root)
		}
		seen[root] = provider
	}
	if len(seen) != 5 {
		t.Fatalf("resolved %d filesystem roots, want 5 with antigravity backend-only", len(seen))
	}
}

// TestStoreRootForRefusals drives every store-root refusal through the
// production entry.
func TestStoreRootForRefusals(t *testing.T) {
	t.Run("provider grammar", func(t *testing.T) {
		_, _, err := StoreRootFor("Codex", bindHome, "")
		requireLocalRefusal(t, err, "invalid_config", "store provider is not a provider id")
	})
	t.Run("home empty", func(t *testing.T) {
		_, _, err := StoreRootFor("codex", "", "")
		requireLocalRefusal(t, err, "invalid_config", "store home is empty")
	})
	t.Run("home relative", func(t *testing.T) {
		_, _, err := StoreRootFor("codex", "Users/iv", "")
		requireLocalRefusal(t, err, "invalid_config", "store home is not an absolute path")
	})
	t.Run("qwen", func(t *testing.T) {
		_, _, err := StoreRootFor("qwen", bindHome, "")
		requireLocalRefusal(t, err, "invalid_config", "store has no direct qwen claim")
	})
	t.Run("unknown provider", func(t *testing.T) {
		_, _, err := StoreRootFor("futuredesk", bindHome, "")
		requireLocalRefusal(t, err, "invalid_config", "store names an unknown provider")
	})
	t.Run("xdg relative", func(t *testing.T) {
		_, _, err := StoreRootFor("muse", bindHome, "data/xdg")
		requireLocalRefusal(t, err, "invalid_config", "store xdg data home is not an absolute path")
	})
}

// TestCheckResumeTuplePassesUsableRows proves the gate admits the
// available and conditional rows it must not block: passing never
// advertises usable, which stays the probe plane's decision.
func TestCheckResumeTuplePassesUsableRows(t *testing.T) {
	rows := []BuildTuple{
		{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"},
		{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "windows", Architecture: "amd64"},
		{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "wsl2", Architecture: "amd64"},
		{ProviderID: "claude", ProviderVersion: "2.1.229", Platform: "windows", Architecture: "arm64"},
		{ProviderID: "gemini", ProviderVersion: "0.9.0", Platform: "wsl2", Architecture: "amd64"},
		{ProviderID: "muse", ProviderVersion: "0.1.0", Platform: "macos", Architecture: "arm64"},
		{ProviderID: "muse", ProviderVersion: "0.2.1", Platform: "macos", Architecture: "amd64"},
		{ProviderID: "muse", ProviderVersion: "0.2.1", Platform: "linux", Architecture: "amd64"},
		{ProviderID: "muse", ProviderVersion: "0.2.1", Platform: "wsl2", Architecture: "arm64"},
		{ProviderID: "antigravity", ProviderVersion: "1.1.14", Platform: "wsl2", Architecture: "amd64"},
		{ProviderID: "pi", ProviderVersion: "0.73.1", Platform: "macos", Architecture: "arm64"},
		{ProviderID: "pi", ProviderVersion: "0.74.0", Platform: "linux", Architecture: "amd64"},
	}
	for _, tuple := range rows {
		name := tuple.ProviderID + "/" + tuple.Platform + "/" + tuple.Architecture + "@" + tuple.ProviderVersion
		t.Run(name, func(t *testing.T) {
			if err := CheckResumeTuple(tuple); err != nil {
				t.Fatalf("CheckResumeTuple(%+v): %v", tuple, err)
			}
		})
	}
}

// TestCheckResumeTupleRefusals drives every tuple-gate refusal through
// the production entry, including the WSL2/native-Windows split the
// matrix requires never to collapse.
func TestCheckResumeTupleRefusals(t *testing.T) {
	rows := []struct {
		name   string
		tuple  BuildTuple
		detail string
	}{
		{"provider grammar", BuildTuple{ProviderID: "Codex", ProviderVersion: "1", Platform: "macos", Architecture: "arm64"}, "resume provider is not a provider id"},
		{"version bounds", BuildTuple{ProviderID: "codex", ProviderVersion: "", Platform: "macos", Architecture: "arm64"}, "resume version is not 1..128 characters"},
		{"platform registry", BuildTuple{ProviderID: "codex", ProviderVersion: "1", Platform: "plan9", Architecture: "arm64"}, "resume platform is not a registry member"},
		{"architecture registry", BuildTuple{ProviderID: "codex", ProviderVersion: "1", Platform: "macos", Architecture: "mips"}, "resume architecture is not amd64 or arm64"},
		{"qwen direct", BuildTuple{ProviderID: "qwen", ProviderVersion: "1", Platform: "linux", Architecture: "amd64"}, "resume has no direct qwen claim"},
		{"unknown provider", BuildTuple{ProviderID: "futuredesk", ProviderVersion: "1", Platform: "linux", Architecture: "amd64"}, "resume names an unknown provider"},
		{"muse native windows", BuildTuple{ProviderID: "muse", ProviderVersion: "0.2.1", Platform: "windows", Architecture: "amd64"}, "resume is unknown on this tuple"},
		{"muse version drift", BuildTuple{ProviderID: "muse", ProviderVersion: "0.2.1", Platform: "macos", Architecture: "arm64"}, "resume is unverified for this muse version"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			requireLocalRefusal(t, CheckResumeTuple(row.tuple), "invalid_config", row.detail)
		})
	}
}

// TestVerifyIdentityBuildPositive proves a created record binds to its
// exact probed tuple.
func TestVerifyIdentityBuildPositive(t *testing.T) {
	if err := VerifyIdentityBuild(bindIdentity(t), validBuildTuple()); err != nil {
		t.Fatalf("VerifyIdentityBuild: %v", err)
	}
}

// TestVerifyIdentityBuildRefusals drives the drift arm and the two
// propagated gates through the production entry.
func TestVerifyIdentityBuildRefusals(t *testing.T) {
	t.Run("version drift", func(t *testing.T) {
		tuple := validBuildTuple()
		tuple.ProviderVersion = "0.148.0"
		requireLocalRefusal(t, VerifyIdentityBuild(bindIdentity(t), tuple), "invalid_config", "build version drifted from the probed tuple")
	})
	t.Run("tuple gate propagates", func(t *testing.T) {
		tuple := validBuildTuple()
		tuple.ProviderID = "futuredesk"
		requireLocalRefusal(t, VerifyIdentityBuild(bindIdentity(t), tuple), "invalid_config", "resume names an unknown provider")
	})
	t.Run("provider correlation propagates", func(t *testing.T) {
		tuple := validBuildTuple()
		tuple.ProviderID = "pi"
		requireLocalRefusal(t, VerifyIdentityBuild(bindIdentity(t), tuple), "invalid_config", "identity names another provider")
	})
}

// TestVerifyIdentityDiscoveryPositive proves the full binding for the
// three discovery shapes: a store discovery at the documented root, a
// project-keyed discovery under it, and a backend discovery with no
// filesystem root.
func TestVerifyIdentityDiscoveryPositive(t *testing.T) {
	t.Run("store root", func(t *testing.T) {
		root := filepath.Join(bindHome, ".codex", "sessions")
		proof := bindProof(bindNativeID, root, false, true, false)
		if err := VerifyIdentityDiscovery(bindIdentity(t), proof, validDiscoveryContext()); err != nil {
			t.Fatalf("store discovery: %v", err)
		}
	})
	t.Run("project-keyed subpath", func(t *testing.T) {
		params := validCreateParams()
		params.ProviderID = "claude"
		params.ProviderVersion = "2.1.229"
		params.ProviderVersionRange = ">=2.1.229 <2.2.0"
		params.NativeSessionID = bindNativeID
		identity := mustCreate(t, params)
		ctx := validDiscoveryContext()
		ctx.Build = BuildTuple{ProviderID: "claude", ProviderVersion: "2.1.229", Platform: "macos", Architecture: "arm64"}
		proof := bindProof(bindNativeID, filepath.Join(bindHome, ".claude", "projects", "keyed-project"), false, true, false)
		if err := VerifyIdentityDiscovery(identity, proof, ctx); err != nil {
			t.Fatalf("project-keyed discovery: %v", err)
		}
	})
	t.Run("pi override", func(t *testing.T) {
		params := validCreateParams()
		params.ProviderID = "pi"
		params.ProviderVersion = "0.73.1"
		params.ProviderVersionRange = ">=0.73.1 <0.74.0"
		params.IdentityKind = "session_path_or_id"
		params.NativeSessionID = bindNativeID
		identity := mustCreate(t, params)
		ctx := validDiscoveryContext()
		ctx.Build = BuildTuple{ProviderID: "pi", ProviderVersion: "0.73.1", Platform: "macos", Architecture: "arm64"}
		ctx.StoreRootOverride = filepath.Join(bindHome, "custom-pi-sessions")
		proof := bindProof(bindNativeID, ctx.StoreRootOverride, false, true, false)
		if err := VerifyIdentityDiscovery(identity, proof, ctx); err != nil {
			t.Fatalf("pi override discovery: %v", err)
		}
	})
	t.Run("backend realm", func(t *testing.T) {
		params := validCreateParams()
		params.ProviderID = "antigravity"
		params.ProviderVersion = "1.1.14"
		params.ProviderVersionRange = ">=1.1.14 <1.2.0"
		params.IdentityKind = "backend_conversation_uuid"
		params.BackendRealm = bindRealm
		params.NativeSessionID = bindNativeID
		identity := mustCreate(t, params)
		ctx := validDiscoveryContext()
		ctx.Build = BuildTuple{ProviderID: "antigravity", ProviderVersion: "1.1.14", Platform: "macos", Architecture: "arm64"}
		ctx.Realm = bindRealm
		proof := bindProof(bindNativeID, "", true, true, true)
		if err := VerifyIdentityDiscovery(identity, proof, ctx); err != nil {
			t.Fatalf("backend discovery: %v", err)
		}
	})
}

// TestVerifyIdentityDiscoveryRefusals drives every discovery-binding
// refusal through the production entry.
func TestVerifyIdentityDiscoveryRefusals(t *testing.T) {
	proof := func() []byte {
		return bindProof(bindNativeID, filepath.Join(bindHome, ".codex", "sessions"), false, true, false)
	}
	t.Run("native id", func(t *testing.T) {
		other := bindProof("22222222-2222-4333-8444-555555555555", filepath.Join(bindHome, ".codex", "sessions"), false, true, false)
		requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), other, validDiscoveryContext()), "invalid_config", "bind native id does not equal the identity")
	})
	t.Run("evidence absent", func(t *testing.T) {
		absent := bindProof(bindNativeID, filepath.Join(bindHome, ".codex", "sessions"), false, false, false)
		requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), absent, validDiscoveryContext()), "invalid_config", "bind discovery evidence is absent")
	})
	t.Run("override pi-only", func(t *testing.T) {
		ctx := validDiscoveryContext()
		ctx.StoreRootOverride = filepath.Join(bindHome, "custom")
		requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), proof(), ctx), "invalid_config", "bind store override is pi-only")
	})
	t.Run("override absolute", func(t *testing.T) {
		params := validCreateParams()
		params.ProviderID = "pi"
		params.ProviderVersion = "0.73.1"
		params.ProviderVersionRange = ">=0.73.1 <0.74.0"
		params.NativeSessionID = bindNativeID
		identity := mustCreate(t, params)
		ctx := validDiscoveryContext()
		ctx.Build = BuildTuple{ProviderID: "pi", ProviderVersion: "0.73.1", Platform: "macos", Architecture: "arm64"}
		ctx.StoreRootOverride = "custom/relative"
		requireLocalRefusal(t, VerifyIdentityDiscovery(identity, proof(), ctx), "invalid_config", "bind store override is not an absolute path")
	})
	t.Run("root missing", func(t *testing.T) {
		rootless := bindProof(bindNativeID, "", true, true, false)
		requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), rootless, validDiscoveryContext()), "invalid_config", "bind discovery root is missing")
	})
	t.Run("root outside", func(t *testing.T) {
		foreign := bindProof(bindNativeID, "/etc/provider", false, true, false)
		requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), foreign, validDiscoveryContext()), "invalid_config", "bind discovery root is outside the provider store")
	})
	t.Run("root sibling prefix", func(t *testing.T) {
		sibling := bindProof(bindNativeID, filepath.Join(bindHome, ".codex", "sessions-evil"), false, true, false)
		requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), sibling, validDiscoveryContext()), "invalid_config", "bind discovery root is outside the provider store")
	})
	t.Run("realm shape", func(t *testing.T) {
		ctx := validDiscoveryContext()
		ctx.Realm = "nope"
		requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), proof(), ctx), "invalid_config", "bind expected realm is not a digest")
	})
	t.Run("realm mismatch", func(t *testing.T) {
		params := validCreateParams()
		params.BackendRealm = bindRealm
		params.NativeSessionID = bindNativeID
		identity := mustCreate(t, params)
		ctx := validDiscoveryContext()
		ctx.Realm = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		requireLocalRefusal(t, VerifyIdentityDiscovery(identity, proof(), ctx), "invalid_config", "bind realm does not equal the expected backend realm")
	})
	t.Run("realm unresolved", func(t *testing.T) {
		params := validCreateParams()
		params.BackendRealm = bindRealm
		params.NativeSessionID = bindNativeID
		identity := mustCreate(t, params)
		unresolved := bindProof(bindNativeID, filepath.Join(bindHome, ".codex", "sessions"), false, true, false)
		requireLocalRefusal(t, VerifyIdentityDiscovery(identity, unresolved, validDiscoveryContext()), "invalid_config", "bind backend realm is claimed but unresolved")
	})
}

// TestResumeProvidersAreDerivedFromSpec proves the resume gate holds
// exactly the six Section 8.2 direct providers: a seventh member
// resolving store roots or passing tuples reddens here, and the Qwen
// and future-plugin rows below the six stay refused by their own
// arms rather than admitted by the table.
func TestResumeProvidersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "8.2", 3865, 3870)
	requireQuote(t, document, "Muse and Antigravity rows above are normative uses", "8.2")
	_ = window
	var derived []string
	for line := 3865; line <= 3870; line++ {
		row, ok := document.TableRowAt(line)
		if !ok {
			t.Fatalf("SPEC.md line %d is not a table body row; the check is blind", line)
		}
		if row.Header != "Provider" {
			t.Fatalf("SPEC.md line %d declares %q, want a Provider row; the check is blind", line, row.Header)
		}
		fields := strings.Fields(row.FirstCell)
		if len(fields) == 0 {
			t.Fatalf("SPEC.md line %d names no provider; the check is blind", line)
		}
		derived = append(derived, strings.ToLower(fields[0]))
	}
	if len(derived) == 0 {
		t.Fatal("derived no providers from the Section 8.2 table; the check is blind")
	}
	requireMemberSet(t, "resumeProviders", append([]string(nil), resumeProviders...), derived)
	for line := 3871; line <= 3872; line++ {
		row, ok := document.TableRowAt(line)
		if !ok {
			t.Fatalf("SPEC.md line %d is not a table body row; the check is blind", line)
		}
		excluded := strings.ToLower(strings.Fields(row.FirstCell)[0])
		for _, provider := range resumeProviders {
			if provider == excluded {
				t.Fatalf("resumeProviders admits the %q row, which must refuse by its own arm", excluded)
			}
		}
	}
	t.Logf("resume provider coverage: %d/%d providers derived", len(resumeProviders), len(derived))
}

// TestDiscoveryMembersAreDerivedFromSpec proves the exact discovery
// member set equals the Section 7.5 NativeDiscoveryProof row: a
// widened implementation admitting "bogus" reddens here.
func TestDiscoveryMembersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	row := sectionLines(t, document, "7.5", 2893, 2893)
	var got []string
	for member := range discoveryMembers {
		got = append(got, member)
	}
	requireMemberSet(t, "discoveryMembers", got, typeRowMembersOf(t, row, "NativeDiscoveryProof"))
}
