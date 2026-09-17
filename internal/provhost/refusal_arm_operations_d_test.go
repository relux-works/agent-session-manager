package provhost

import (
	"path/filepath"
	"strings"
	"testing"
)

// declaredOperationWitnessesIdentityCreate proves the identity
// creation and binding arms: CreateIdentity params, the
// NativeDiscoveryProof decoder, the Section 8.2 store-root table, the
// Section 8.4 resume-tuple gate, and the build and discovery binding
// verifiers. It extends declaredOperationWitnessesIdentity in
// refusal_arm_operations_c_test.go; the split is file size only.
func declaredOperationWitnessesIdentityCreate() []armWitness {
	createParams := func(mutate func(*IdentityParams)) IdentityParams {
		params := validCreateParams()
		mutate(&params)
		return params
	}
	return []armWitness{
		// Creation arms, through CreateIdentity.
		{arm: `ctor|failInvalid|create session_id is not a UUIDv7`, name: "create bogus session", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.SessionID = "nope" }))
			requireLocalRefusal(t, err, "invalid_config", "create session_id is not a UUIDv7")
		}},
		{arm: `ctor|failInvalid|create provider_id is not a provider id`, name: "create uppercase provider", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.ProviderID = "Codex" }))
			requireLocalRefusal(t, err, "invalid_config", "create provider_id is not a provider id")
		}},
		{arm: `ctor|failInvalid|create provider_version is not 1..128 characters`, name: "create empty version", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.ProviderVersion = "" }))
			requireLocalRefusal(t, err, "invalid_config", "create provider_version is not 1..128 characters")
		}},
		{arm: `ctor|failInvalid|create provider_version_range is not 1..256 characters`, name: "create empty range", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.ProviderVersionRange = "" }))
			requireLocalRefusal(t, err, "invalid_config", "create provider_version_range is not 1..256 characters")
		}},
		{arm: `ctor|failInvalid|create native_session_id is not 1..512 characters`, name: "create empty native id", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.NativeSessionID = "" }))
			requireLocalRefusal(t, err, "invalid_config", "create native_session_id is not 1..512 characters")
		}},
		{arm: `ctor|failInvalid|create identity_kind is not a registry member`, name: "create window handle kind", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.IdentityKind = "window_handle" }))
			requireLocalRefusal(t, err, "invalid_config", "create identity_kind is not a registry member")
		}},
		{arm: `ctor|failInvalid|create logical_workspace_id is not a UUIDv7`, name: "create bogus workspace", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.LogicalWorkspaceID = "nope" }))
			requireLocalRefusal(t, err, "invalid_config", "create logical_workspace_id is not a UUIDv7")
		}},
		{arm: `ctor|failInvalid|create backend_realm_fingerprint is not a digest or empty`, name: "create garbage realm", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.BackendRealm = "nope" }))
			requireLocalRefusal(t, err, "invalid_config", "create backend_realm_fingerprint is not a digest or empty")
		}},
		{arm: `ctor|failInvalid|create backend_realm_fingerprint is required for this backend kind`, name: "create antigravity null realm", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) {
				p.ProviderID = "antigravity"
				p.IdentityKind = "backend_conversation_uuid"
			}))
			requireLocalRefusal(t, err, "invalid_config", "create backend_realm_fingerprint is required for this backend kind")
		}},
		{arm: `ctor|failInvalid|create opaque_identity exceeds 32 entries`, name: "create 33 opaque entries", prove: func(t *testing.T) {
			opaque := map[string]string{}
			for i := 0; i < 33; i++ {
				opaque["k"+pad2(i)] = "x"
			}
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.OpaqueIdentity = opaque }))
			requireLocalRefusal(t, err, "invalid_config", "create opaque_identity exceeds 32 entries")
		}},
		{arm: `ctor|failInvalid|create opaque key is not a provider key`, name: "create bad opaque key", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.OpaqueIdentity = map[string]string{"Bad Key!": "x"} }))
			requireLocalRefusal(t, err, "invalid_config", "create opaque key is not a provider key")
		}},
		{arm: `ctor|failInvalid|create opaque value is not 1..1024 characters`, name: "create empty opaque value", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.OpaqueIdentity = map[string]string{"ok": ""} }))
			requireLocalRefusal(t, err, "invalid_config", "create opaque value is not 1..1024 characters")
		}},
		{arm: `ctor|failInvalid|create opaque value begins with an absolute path`, name: "create absolute opaque value", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.OpaqueIdentity = map[string]string{"workdir": "/tmp/x"} }))
			requireLocalRefusal(t, err, "invalid_config", "create opaque value begins with an absolute path")
		}},
		{arm: `ctor|failInvalid|create created_by_host_id is not a UUIDv7`, name: "create bogus host", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.CreatedByHostID = "nope" }))
			requireLocalRefusal(t, err, "invalid_config", "create created_by_host_id is not a UUIDv7")
		}},
		{arm: `ctor|failInvalid|create created_at is not a timestamp`, name: "create bogus timestamp", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.CreatedAt = "yesterday" }))
			requireLocalRefusal(t, err, "invalid_config", "create created_at is not a timestamp")
		}},
		{arm: `ctor|failInvalid|create extensions exceed 64 entries`, name: "create 65 extensions", prove: func(t *testing.T) {
			extensions := map[string]any{}
			for i := 0; i < 65; i++ {
				extensions["com.example.k"+pad2(i)] = i
			}
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.Extensions = extensions }))
			requireLocalRefusal(t, err, "invalid_config", "create extensions exceed 64 entries")
		}},
		{arm: `ctor|failInvalid|create extension key is not reverse-DNS`, name: "create bad extension key", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.Extensions = map[string]any{"NoDNS": true} }))
			requireLocalRefusal(t, err, "invalid_config", "create extension key is not reverse-DNS")
		}},
		{arm: `ctor|failInvalid|create extensions carry a non-JSON value`, name: "create channel extension", prove: func(t *testing.T) {
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.Extensions = map[string]any{"com.example.bad": make(chan int)} }))
			requireLocalRefusal(t, err, "invalid_config", "create extensions carry a non-JSON value")
		}},
		{arm: `ctor|failInvalid|create params are not a valid provider identity`, name: "create deep extensions", prove: func(t *testing.T) {
			deep := map[string]any{"com.example.deep": map[string]any{"l1": map[string]any{"l2": map[string]any{"l3": map[string]any{"l4": map[string]any{"l5": "x"}}}}}}
			_, err := CreateIdentity(createParams(func(p *IdentityParams) { p.Extensions = deep }))
			requireLocalRefusal(t, err, "invalid_config", "create params are not a valid provider identity")
		}},
		// Discovery decoder arms, through DecodeNativeDiscovery.
		{arm: `ctor|failInvalid|discovery platform is not a registry member`, name: "discovery plan9 platform", prove: func(t *testing.T) {
			_, err := DecodeNativeDiscovery(bindProof("x", "", true, true, false), "plan9")
			requireLocalRefusal(t, err, "invalid_config", "discovery platform is not a registry member")
		}},
		{arm: `ctor|failProtocol|discovery carries unknown member`, name: "discovery score member", prove: func(t *testing.T) {
			body := []byte(`{"native_session_id": "x", "discovered": true, "discovery_root": null, "backend_resolved": false, "score": 1}`)
			_, err := DecodeNativeDiscovery(body, "macos")
			requireFrameRefusal(t, err, "score", "unknown member")
		}},
		{arm: `ctor|failProtocol|discovery misses a required member`, name: "discovery without root", prove: func(t *testing.T) {
			body := []byte(`{"native_session_id": "x", "discovered": true, "backend_resolved": false}`)
			_, err := DecodeNativeDiscovery(body, "macos")
			requireFrameRefusal(t, err, "discovery_root", "misses a required member")
		}},
		{arm: `ctor|failProtocol|discovery native_session_id is not 1..512 characters`, name: "discovery empty native id", prove: func(t *testing.T) {
			_, err := DecodeNativeDiscovery(bindProof("", "", true, true, false), "macos")
			requireFrameRefusal(t, err, "native_session_id", "not 1..512 characters")
		}},
		{arm: `ctor|failProtocol|discovery discovered is not a boolean`, name: "discovery string discovered", prove: func(t *testing.T) {
			body := []byte(`{"native_session_id": "x", "discovered": "yes", "discovery_root": null, "backend_resolved": false}`)
			_, err := DecodeNativeDiscovery(body, "macos")
			requireFrameRefusal(t, err, "discovered", "not a boolean")
		}},
		{arm: `ctor|failProtocol|discovery discovery_root is not an absolute path or null`, name: "discovery relative root", prove: func(t *testing.T) {
			_, err := DecodeNativeDiscovery(bindProof("x", "sessions/relative", false, true, false), "macos")
			requireFrameRefusal(t, err, "discovery_root", "not an absolute path or null")
		}},
		{arm: `ctor|failProtocol|discovery backend_resolved is not a boolean`, name: "discovery string backend", prove: func(t *testing.T) {
			body := []byte(`{"native_session_id": "x", "discovered": true, "discovery_root": null, "backend_resolved": "no"}`)
			_, err := DecodeNativeDiscovery(body, "macos")
			requireFrameRefusal(t, err, "backend_resolved", "not a boolean")
		}},
		// Store-root arms, through StoreRootFor.
		{arm: `ctor|failInvalid|store provider is not a provider id`, name: "store uppercase provider", prove: func(t *testing.T) {
			_, _, err := StoreRootFor("Codex", bindHome, "")
			requireLocalRefusal(t, err, "invalid_config", "store provider is not a provider id")
		}},
		{arm: `ctor|failInvalid|store home is empty`, name: "store empty home", prove: func(t *testing.T) {
			_, _, err := StoreRootFor("codex", "", "")
			requireLocalRefusal(t, err, "invalid_config", "store home is empty")
		}},
		{arm: `ctor|failInvalid|store home is not an absolute path`, name: "store relative home", prove: func(t *testing.T) {
			_, _, err := StoreRootFor("codex", "Users/iv", "")
			requireLocalRefusal(t, err, "invalid_config", "store home is not an absolute path")
		}},
		{arm: `ctor|failInvalid|store has no direct qwen claim`, name: "store qwen", prove: func(t *testing.T) {
			_, _, err := StoreRootFor("qwen", bindHome, "")
			requireLocalRefusal(t, err, "invalid_config", "store has no direct qwen claim")
		}},
		{arm: `ctor|failInvalid|store names an unknown provider`, name: "store future provider", prove: func(t *testing.T) {
			_, _, err := StoreRootFor("futuredesk", bindHome, "")
			requireLocalRefusal(t, err, "invalid_config", "store names an unknown provider")
		}},
		{arm: `ctor|failInvalid|store xdg data home is not an absolute path`, name: "store relative xdg", prove: func(t *testing.T) {
			_, _, err := StoreRootFor("muse", bindHome, "data/xdg")
			requireLocalRefusal(t, err, "invalid_config", "store xdg data home is not an absolute path")
		}},
		// Resume-tuple arms, through CheckResumeTuple.
		{arm: `ctor|failInvalid|resume provider is not a provider id`, name: "resume uppercase provider", prove: func(t *testing.T) {
			tuple := validBuildTuple()
			tuple.ProviderID = "Codex"
			requireLocalRefusal(t, CheckResumeTuple(tuple), "invalid_config", "resume provider is not a provider id")
		}},
		{arm: `ctor|failInvalid|resume version is not 1..128 characters`, name: "resume empty version", prove: func(t *testing.T) {
			tuple := validBuildTuple()
			tuple.ProviderVersion = ""
			requireLocalRefusal(t, CheckResumeTuple(tuple), "invalid_config", "resume version is not 1..128 characters")
		}},
		{arm: `ctor|failInvalid|resume platform is not a registry member`, name: "resume plan9 platform", prove: func(t *testing.T) {
			tuple := validBuildTuple()
			tuple.Platform = "plan9"
			requireLocalRefusal(t, CheckResumeTuple(tuple), "invalid_config", "resume platform is not a registry member")
		}},
		{arm: `ctor|failInvalid|resume architecture is not amd64 or arm64`, name: "resume mips architecture", prove: func(t *testing.T) {
			tuple := validBuildTuple()
			tuple.Architecture = "mips"
			requireLocalRefusal(t, CheckResumeTuple(tuple), "invalid_config", "resume architecture is not amd64 or arm64")
		}},
		{arm: `ctor|failInvalid|resume has no direct qwen claim`, name: "resume qwen", prove: func(t *testing.T) {
			tuple := validBuildTuple()
			tuple.ProviderID = "qwen"
			requireLocalRefusal(t, CheckResumeTuple(tuple), "invalid_config", "resume has no direct qwen claim")
		}},
		{arm: `ctor|failInvalid|resume names an unknown provider`, name: "resume future provider", prove: func(t *testing.T) {
			tuple := validBuildTuple()
			tuple.ProviderID = "futuredesk"
			requireLocalRefusal(t, CheckResumeTuple(tuple), "invalid_config", "resume names an unknown provider")
		}},
		{arm: `ctor|failInvalid|resume is unknown on this tuple`, name: "resume muse native windows", prove: func(t *testing.T) {
			tuple := BuildTuple{ProviderID: "muse", ProviderVersion: "0.2.1", Platform: "windows", Architecture: "amd64"}
			requireLocalRefusal(t, CheckResumeTuple(tuple), "invalid_config", "resume is unknown on this tuple")
		}},
		{arm: `ctor|failInvalid|resume is unverified for this muse version`, name: "resume muse 0.2.1 arm64", prove: func(t *testing.T) {
			tuple := BuildTuple{ProviderID: "muse", ProviderVersion: "0.2.1", Platform: "macos", Architecture: "arm64"}
			requireLocalRefusal(t, CheckResumeTuple(tuple), "invalid_config", "resume is unverified for this muse version")
		}},
		// Build-binding arm, through VerifyIdentityBuild.
		{arm: `ctor|failInvalid|build version drifted from the probed tuple`, name: "build drifted version", prove: func(t *testing.T) {
			tuple := validBuildTuple()
			tuple.ProviderVersion = "0.148.0"
			requireLocalRefusal(t, VerifyIdentityBuild(bindIdentity(t), tuple), "invalid_config", "build version drifted from the probed tuple")
		}},
		// Discovery-binding arms, through VerifyIdentityDiscovery.
		{arm: `ctor|failInvalid|bind native id does not equal the identity`, name: "bind foreign native id", prove: func(t *testing.T) {
			proof := bindProof("22222222-2222-4333-8444-555555555555", filepath.Join(bindHome, ".codex", "sessions"), false, true, false)
			requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), proof, validDiscoveryContext()), "invalid_config", "bind native id does not equal the identity")
		}},
		{arm: `ctor|failInvalid|bind discovery evidence is absent`, name: "bind undiscovered proof", prove: func(t *testing.T) {
			proof := bindProof(bindNativeID, filepath.Join(bindHome, ".codex", "sessions"), false, false, false)
			requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), proof, validDiscoveryContext()), "invalid_config", "bind discovery evidence is absent")
		}},
		{arm: `ctor|failInvalid|bind store override is pi-only`, name: "bind codex override", prove: func(t *testing.T) {
			ctx := validDiscoveryContext()
			ctx.StoreRootOverride = filepath.Join(bindHome, "custom")
			proof := bindProof(bindNativeID, filepath.Join(bindHome, ".codex", "sessions"), false, true, false)
			requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), proof, ctx), "invalid_config", "bind store override is pi-only")
		}},
		{arm: `ctor|failInvalid|bind store override is not an absolute path`, name: "bind relative pi override", prove: func(t *testing.T) {
			params := validCreateParams()
			params.ProviderID = "pi"
			params.ProviderVersion = "0.73.1"
			params.ProviderVersionRange = ">=0.73.1 <0.74.0"
			identity := mustCreate(t, params)
			ctx := validDiscoveryContext()
			ctx.Build = BuildTuple{ProviderID: "pi", ProviderVersion: "0.73.1", Platform: "macos", Architecture: "arm64"}
			ctx.StoreRootOverride = "custom/relative"
			proof := bindProof(params.NativeSessionID, filepath.Join(bindHome, ".pi", "agent", "sessions"), false, true, false)
			requireLocalRefusal(t, VerifyIdentityDiscovery(identity, proof, ctx), "invalid_config", "bind store override is not an absolute path")
		}},
		{arm: `ctor|failInvalid|bind discovery root is missing`, name: "bind null store root", prove: func(t *testing.T) {
			proof := bindProof(bindNativeID, "", true, true, false)
			requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), proof, validDiscoveryContext()), "invalid_config", "bind discovery root is missing")
		}},
		{arm: `ctor|failInvalid|bind discovery root is outside the provider store`, name: "bind foreign root", prove: func(t *testing.T) {
			proof := bindProof(bindNativeID, "/etc/provider", false, true, false)
			requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), proof, validDiscoveryContext()), "invalid_config", "bind discovery root is outside the provider store")
		}},
		{arm: `ctor|failInvalid|bind expected realm is not a digest`, name: "bind garbage realm", prove: func(t *testing.T) {
			ctx := validDiscoveryContext()
			ctx.Realm = "nope"
			proof := bindProof(bindNativeID, filepath.Join(bindHome, ".codex", "sessions"), false, true, false)
			requireLocalRefusal(t, VerifyIdentityDiscovery(bindIdentity(t), proof, ctx), "invalid_config", "bind expected realm is not a digest")
		}},
		{arm: `ctor|failInvalid|bind realm does not equal the expected backend realm`, name: "bind drifted realm", prove: func(t *testing.T) {
			params := validCreateParams()
			params.BackendRealm = bindRealm
			identity := mustCreate(t, params)
			ctx := validDiscoveryContext()
			ctx.Realm = "sha256:" + strings.Repeat("b", 64)
			proof := bindProof(params.NativeSessionID, filepath.Join(bindHome, ".codex", "sessions"), false, true, true)
			requireLocalRefusal(t, VerifyIdentityDiscovery(identity, proof, ctx), "invalid_config", "bind realm does not equal the expected backend realm")
		}},
		{arm: `ctor|failInvalid|bind backend realm is claimed but unresolved`, name: "bind unresolved realm", prove: func(t *testing.T) {
			params := validCreateParams()
			params.BackendRealm = bindRealm
			identity := mustCreate(t, params)
			proof := bindProof(params.NativeSessionID, filepath.Join(bindHome, ".codex", "sessions"), false, true, false)
			requireLocalRefusal(t, VerifyIdentityDiscovery(identity, proof, validDiscoveryContext()), "invalid_config", "bind backend realm is claimed but unresolved")
		}},
	}
}
