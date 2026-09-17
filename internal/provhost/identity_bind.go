package provhost

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file binds Provider Identity Records to the exact native facts
// they claim: the probed build tuple, the documented store root, the
// backend realm, and the discovery proof a materialize or resume
// observation returned. Creation (identity_create.go) builds records;
// this file refuses to USE one whose facts drifted.
//
// The Section 8 matrices are evidence labels, not availability
// promises: every gate here refuses the fail-closed direction —
// unknown providers and platforms, rows the matrix marks unsupported
// or unknown, and version cells the matrix pins to an exact probe —
// and passing a gate never advertises a capability as usable. A row
// the matrix marks conditional still requires probe acceptance at the
// use site through RequireCapability; only the probe plane in Section
// 7.4 establishes usable. No gate invents parity for a gated cell.
//
// All binding refusals are invalid_config caller errors: the caller
// correlated the wrong identity, proof, or tuple. Only the discovery
// proof decoder reads plugin bytes, so only its shape arms are
// provider_protocol_errors with member attribution.

// resumeProviders is the six Section 8.2 direct providers in table
// order: the only providers with a documented native-store row. Qwen
// has no direct claim and future plugins start disabled, so neither
// appears here: both refuse at their own arms below, never by table
// default.
var resumeProviders = []string{
	"codex",
	"claude",
	"gemini",
	"muse",
	"antigravity",
	"pi",
}

// BuildTuple is the exact probed provider build an identity is bound
// to: the Section 7.4 probe members provider_id, provider_version,
// platform, and architecture. Every binding compares exact equality;
// there is no normalization, no prefix match, and no version-range
// evaluation, because Section 5.5 declares the range opaque adapter
// data rather than a decidable grammar.
type BuildTuple struct {
	ProviderID      string
	ProviderVersion string
	Platform        string
	Architecture    string
}

// NativeDiscovery is one validated Section 7.5 NativeDiscoveryProof:
// the native session ID the proof resolved, whether discovery
// succeeded, the absolute root it resolved under (present exactly
// when the proof carries a non-null discovery_root), and whether the
// backend resolved the ID.
type NativeDiscovery struct {
	NativeSessionID  string
	Discovered       bool
	DiscoveryRoot    string
	HasDiscoveryRoot bool
	BackendResolved  bool
}

// DiscoveryContext carries the host-side facts one discovery proof is
// bound against: the exact probed build tuple, the home and XDG data
// home the Section 8.2 store root resolves from, the expected backend
// realm digest (empty when no realm is required), and the Pi session
// directory override (empty for the documented default).
type DiscoveryContext struct {
	Build             BuildTuple
	Home              string
	XDGDataHome       string
	Realm             string
	StoreRootOverride string
}

// discoveryMembers is the exact required member set of a
// NativeDiscoveryProof.
var discoveryMembers = map[string]bool{
	"native_session_id": true,
	"discovered":        true,
	"discovery_root":    true,
	"backend_resolved":  true,
}

// discoveryRequired lists discoveryMembers in a fixed order so a proof
// missing several members always names the same one.
var discoveryRequired = []string{
	"native_session_id",
	"discovered",
	"discovery_root",
	"backend_resolved",
}

// DecodeNativeDiscovery validates one Section 7.5 NativeDiscoveryProof
// body on the named platform: the exact native session ID string, the
// discovered and backend_resolved booleans, and an absolute
// destination-native discovery root or null. Shape violations are
// provider_protocol_errors; a platform outside the registry is an
// invalid_config caller error, because the proof carries no platform
// and the caller named it.
func DecodeNativeDiscovery(body []byte, platform string) (NativeDiscovery, error) {
	empty := NativeDiscovery{}
	if !isProbePlatform(platform) {
		failure, err := failInvalid("discovery platform is not a registry member")
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol(fault.detail, fault.member)
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	if name, unknown := unknownMember(members, discoveryMembers); unknown {
		failure, err := failProtocol("discovery carries unknown member", name)
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	if name, missing := missingMember(members, discoveryRequired); missing {
		failure, err := failProtocol("discovery misses a required member", name)
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	native, ok := rawString(members["native_session_id"])
	if !ok || runeLength(native) < 1 || runeLength(native) > 512 {
		failure, err := failProtocol("discovery native_session_id is not 1..512 characters", "native_session_id")
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	discovered, ok := rawBool(members["discovered"])
	if !ok {
		failure, err := failProtocol("discovery discovered is not a boolean", "discovered")
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	root, isNull, ok := rawNullableString(members["discovery_root"])
	if !ok || (!isNull && !isAbsoluteOn(root, scalar.Platform(platform))) {
		failure, err := failProtocol("discovery discovery_root is not an absolute path or null", "discovery_root")
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	backendResolved, ok := rawBool(members["backend_resolved"])
	if !ok {
		failure, err := failProtocol("discovery backend_resolved is not a boolean", "backend_resolved")
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	return NativeDiscovery{
		NativeSessionID:  native,
		Discovered:       discovered,
		DiscoveryRoot:    root,
		HasDiscoveryRoot: !isNull,
		BackendResolved:  backendResolved,
	}, nil
}

// StoreRootFor resolves the Section 8.2 documented native-store root
// for one provider under the given home. It returns the absolute root,
// or backendOnly when the provider identifies through a backend realm
// rather than a filesystem store. XDG data home applies to Muse only;
// an empty value selects the documented default below ~/.local/share.
// Refusals are invalid_config caller errors: Qwen has no direct claim,
// unknown providers start disabled, and a non-absolute home cannot
// anchor a store.
func StoreRootFor(providerID, home, xdgDataHome string) (string, bool, error) {
	if !validProviderID(providerID) {
		failure, err := failInvalid("store provider is not a provider id")
		if err != nil {
			return "", false, err
		}
		return "", false, failure
	}
	if home == "" {
		failure, err := failInvalid("store home is empty")
		if err != nil {
			return "", false, err
		}
		return "", false, failure
	}
	if !isBareAbsolute(home) {
		failure, err := failInvalid("store home is not an absolute path")
		if err != nil {
			return "", false, err
		}
		return "", false, failure
	}
	if providerID == "qwen" {
		failure, err := failInvalid("store has no direct qwen claim")
		if err != nil {
			return "", false, err
		}
		return "", false, failure
	}
	if !isResumeProvider(providerID) {
		failure, err := failInvalid("store names an unknown provider")
		if err != nil {
			return "", false, err
		}
		return "", false, failure
	}
	// Antigravity identifies through the destination-authenticated
	// backend realm, not a filesystem store: Section 8.2 names no
	// durable root, and Appendix B leaves the SQLite root unsettled.
	if providerID == "antigravity" {
		return "", true, nil
	}
	if providerID == "muse" && xdgDataHome != "" {
		if !isBareAbsolute(xdgDataHome) {
			failure, err := failInvalid("store xdg data home is not an absolute path")
			if err != nil {
				return "", false, err
			}
			return "", false, failure
		}
		return filepath.Join(xdgDataHome, "muse", "sessions"), false, nil
	}
	if providerID == "codex" {
		return filepath.Join(home, ".codex", "sessions"), false, nil
	}
	if providerID == "claude" {
		return filepath.Join(home, ".claude", "projects"), false, nil
	}
	if providerID == "gemini" {
		return filepath.Join(home, ".gemini"), false, nil
	}
	if providerID == "muse" {
		return filepath.Join(home, ".local", "share", "muse", "sessions"), false, nil
	}
	// Only pi reaches here: six-membership is proven above and every
	// other member returned. A seventh table member must extend this
	// chain in the same change; the pairwise-distinctness test below
	// the chain fails a provider silently aliased onto pi's root.
	return filepath.Join(home, ".pi", "agent", "sessions"), false, nil
}

// CheckResumeTuple enforces the Section 8.4 native-resume direction
// for one exact probed build tuple: unknown providers and platforms
// refuse, the Qwen direct row refuses, the Muse native-Windows unknown
// cell refuses, and a Muse macOS arm64 build refuses unless it is the
// exact 0.1.0 probe the cell accepts, because Appendix B leaves every
// newer Muse behavior unsettled with enabled false. Available and
// conditional rows pass this gate; passing never advertises usable,
// and conditional rows still require probe acceptance at the use site.
func CheckResumeTuple(tuple BuildTuple) error {
	if !validProviderID(tuple.ProviderID) {
		failure, err := failInvalid("resume provider is not a provider id")
		if err != nil {
			return err
		}
		return failure
	}
	if runeLength(tuple.ProviderVersion) < 1 || runeLength(tuple.ProviderVersion) > 128 {
		failure, err := failInvalid("resume version is not 1..128 characters")
		if err != nil {
			return err
		}
		return failure
	}
	if !isProbePlatform(tuple.Platform) {
		failure, err := failInvalid("resume platform is not a registry member")
		if err != nil {
			return err
		}
		return failure
	}
	if !isProbeArchitecture(tuple.Architecture) {
		failure, err := failInvalid("resume architecture is not amd64 or arm64")
		if err != nil {
			return err
		}
		return failure
	}
	if tuple.ProviderID == "qwen" {
		failure, err := failInvalid("resume has no direct qwen claim")
		if err != nil {
			return err
		}
		return failure
	}
	if !isResumeProvider(tuple.ProviderID) {
		failure, err := failInvalid("resume names an unknown provider")
		if err != nil {
			return err
		}
		return failure
	}
	if tuple.ProviderID == "muse" && tuple.Platform == "windows" {
		failure, err := failInvalid("resume is unknown on this tuple")
		if err != nil {
			return err
		}
		return failure
	}
	if tuple.ProviderID == "muse" && tuple.Platform == "macos" && tuple.Architecture == "arm64" && tuple.ProviderVersion != "0.1.0" {
		failure, err := failInvalid("resume is unverified for this muse version")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// VerifyIdentityBuild binds one Provider Identity Record to the exact
// probed build tuple: the tuple must pass the Section 8.4 resume
// gate, the record must validate for the tuple's provider, and the
// record's exact probed version must equal the tuple's version. Any
// drift between the identity's build facts and the probed facts
// refuses, because a mapping or resume decision made for one build
// must never silently apply to another.
func VerifyIdentityBuild(identityBody []byte, tuple BuildTuple) error {
	if err := CheckResumeTuple(tuple); err != nil {
		return err
	}
	if err := CheckIdentity(identityBody, tuple.ProviderID); err != nil {
		return err
	}
	// The members below replay the record CheckIdentity just proved
	// well-formed, never the raw body: the fault is nil by
	// construction, following the RequireCapability pattern, so no
	// refusal can drift between the validation and the use.
	members, _ := decodeStrictObject(identityBody)
	version, _ := rawString(members["provider_version"])
	if version != tuple.ProviderVersion {
		failure, err := failInvalid("build version drifted from the probed tuple")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// VerifyIdentityDiscovery binds one Provider Identity Record to one
// discovery proof under the host's discovery context: the identity
// must bind to the probed build tuple, the proof must decode on the
// tuple's platform, its exact native ID must equal the identity's, it
// must report discovery, its root must sit at or under the Section 8.2
// store root (or the Pi override), and its backend resolution must
// cover the identity's realm claim and the expected realm. A null
// proof root is refused for filesystem providers; backend providers
// carry no root rule because no documented root exists to bind.
func VerifyIdentityDiscovery(identityBody, proofBody []byte, ctx DiscoveryContext) error {
	if err := VerifyIdentityBuild(identityBody, ctx.Build); err != nil {
		return err
	}
	proof, err := DecodeNativeDiscovery(proofBody, ctx.Build.Platform)
	if err != nil {
		return err
	}
	members, _ := decodeStrictObject(identityBody)
	native, _ := rawString(members["native_session_id"])
	if proof.NativeSessionID != native {
		failure, err := failInvalid("bind native id does not equal the identity")
		if err != nil {
			return err
		}
		return failure
	}
	if !proof.Discovered {
		failure, err := failInvalid("bind discovery evidence is absent")
		if err != nil {
			return err
		}
		return failure
	}
	if err := checkBindStoreRoot(proof, ctx); err != nil {
		return err
	}
	if err := checkBindRealm(proof, members, ctx.Realm); err != nil {
		return err
	}
	return nil
}

// checkBindStoreRoot requires the proof root at or under the expected
// store root: the Pi override when the caller sets one, else the
// Section 8.2 documented root. Backend providers return before any
// root rule, because identity there resolves through the realm.
func checkBindStoreRoot(proof NativeDiscovery, ctx DiscoveryContext) error {
	expected, backendOnly, err := resolveBindRoot(ctx)
	if err != nil {
		return err
	}
	if backendOnly {
		return nil
	}
	if !proof.HasDiscoveryRoot {
		failure, err := failInvalid("bind discovery root is missing")
		if err != nil {
			return err
		}
		return failure
	}
	if !rootContains(expected, proof.DiscoveryRoot) {
		failure, err := failInvalid("bind discovery root is outside the provider store")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// resolveBindRoot selects the expected store root: the caller override
// for Pi, whose Section 8.2 row documents
// PI_CODING_AGENT_SESSION_DIR and --session-dir, else the Section 8.2
// documented root. Any other provider's override refuses: the surface
// is documented for Pi only, and no provider inherits it by default.
func resolveBindRoot(ctx DiscoveryContext) (string, bool, error) {
	if ctx.StoreRootOverride != "" {
		if ctx.Build.ProviderID != "pi" {
			failure, err := failInvalid("bind store override is pi-only")
			if err != nil {
				return "", false, err
			}
			return "", false, failure
		}
		if !isAbsoluteOn(ctx.StoreRootOverride, scalar.Platform(ctx.Build.Platform)) {
			failure, err := failInvalid("bind store override is not an absolute path")
			if err != nil {
				return "", false, err
			}
			return "", false, failure
		}
		return ctx.StoreRootOverride, false, nil
	}
	return StoreRootFor(ctx.Build.ProviderID, ctx.Home, ctx.XDGDataHome)
}

// checkBindRealm requires the proof's backend resolution to cover the
// identity's realm claim and the caller's expected realm: an expected
// realm must be a digest the identity carries exactly, and a
// non-null identity realm requires backend_resolved, because the
// realm is a resume precondition wherever it is set.
func checkBindRealm(proof NativeDiscovery, members map[string]json.RawMessage, wantRealm string) error {
	realm, isNull, _ := rawNullableString(members["backend_realm_fingerprint"])
	if wantRealm != "" {
		if !isDigest(wantRealm) {
			failure, err := failInvalid("bind expected realm is not a digest")
			if err != nil {
				return err
			}
			return failure
		}
		if isNull || realm != wantRealm {
			failure, err := failInvalid("bind realm does not equal the expected backend realm")
			if err != nil {
				return err
			}
			return failure
		}
	}
	if !isNull && !proof.BackendResolved {
		failure, err := failInvalid("bind backend realm is claimed but unresolved")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}

// isResumeProvider reports whether the provider carries a Section 8.2
// direct native-store row. It ranges resumeProviders, never a sample:
// a validation that held for three providers and skipped the rest
// would pass a tuple this gate must refuse.
func isResumeProvider(providerID string) bool {
	for _, provider := range resumeProviders {
		if providerID == provider {
			return true
		}
	}
	return false
}

// isBareAbsolute reports whether the path starts with an absolute
// prefix on any platform: a forward slash, a UNC backslash pair, or a
// drive letter with a separator. Home and XDG roots arrive before any
// platform is selected, so the check cannot be platform-native; the
// proof-root rule beside it is, through isAbsoluteOn.
func isBareAbsolute(path string) bool {
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\\`) || windowsDrivePattern.MatchString(path)
}

// rootContains reports whether the observed proof root equals the
// expected store root or sits under it: discovery of a session inside
// a project-keyed subdirectory still counts as discovery in the
// provider's store. Comparison is byte-exact and separator-aware, so a
// sibling whose name merely extends the root prefix is outside.
func rootContains(expected, observed string) bool {
	if expected == observed {
		return true
	}
	relative, err := filepath.Rel(expected, observed)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
