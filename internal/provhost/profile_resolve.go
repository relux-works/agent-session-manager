package provhost

import (
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// This file resolves Section 7.7 execution-profile mappings against the
// exact probed provider build and projects the sanitized launch argv a
// launch or resume operation executes. The table in profile.go is the
// single source of the required yolo flags; this file decides when
// those flags may be applied.
//
// Resolution requires the exact-version probe result as a BuildTuple:
// the tuple passes the Section 8.4 resume gate (CheckResumeTuple, so
// an unknown provider, platform, or architecture, a direct Qwen
// claim, and an unverified Muse build refuse before any mapping is
// considered) and names the resolved provider exactly, because a
// mapping decided for one build must never silently apply to
// another. A provider without a Section 7.7 row — Qwen direct,
// future plugins — and a Pi build other than the table's pinned
// 0.73.1 cannot be mapped for the probed version and refuse with
// the Section 2.4 profile_mapping_unavailable resume failure.
//
// The launch projection emits the unrestricted flag only for yolo
// and only the exact table token: an absent flag (nothing to emit
// where yolo requires one is impossible by construction, so absence
// is the caller's empty base, refused structurally) or a changed
// flag (any table token already present in the base, or any
// foreign table token) fails closed. Standard omits every
// unrestricted flag and refuses a base that already carries one —
// the decidable half of the machine-local-alias rule: an exact
// token match over the six table flags plus the documented codex
// --yolo alias. A locally aliased shell word that expands to
// unrestricted mode without spelling a table token is beyond what
// argv inspection can decide (shell aliases never expand under
// execve argv, which is the structural defense, owned by secprim);
// that residual is stated, not silently admitted.
//
// The Section 5.1 argv bounds enforced here (1..128 elements, each
// 1..4,096 bytes, 65,536 bytes total) are the host-side twin of
// the SpawnPlan wire bounds checkSpawnArgv enforces: the two never
// diverge, pinned by the agreement test that drives identical
// vectors at each limit through both entries. Structural argv
// validity (non-empty, NUL-free, UTF-8) stays with secprim and is
// driven, not duplicated.

// codexAliasYOLO is the codex --yolo alias Section 7.7 documents as
// accepted by the current CLI. The required adapter mapping stays
// the long form in profileYOLOMapping; the alias exists here only
// so the launch projection recognizes (and refuses) it as an
// unrestricted token rather than passing it through as an
// innocuous word.
const codexAliasYOLO = "--yolo"

// piPinnedVersion is the exact Pi build the Section 7.7 table maps.
// Every other Pi version refuses profile_mapping_unavailable:
// the table pins 0.73.1, and no other row covers Pi.
const piPinnedVersion = "0.73.1"

// ResolvedMapping is one Section 7.7 mapping decision for an exact
// probed build: the provider and profile it was resolved for, the
// probed version it is bound to, the SpawnPlan profile_mapping
// value (the exact table flag for yolo, empty for standard, which
// omits every unrestricted flag), and whether both profiles share
// the provider's default full tool set (Pi only — probe and launch
// output MUST disclose that equivalence).
type ResolvedMapping struct {
	ProviderID string
	Profile    string
	Version    string
	Mapping    string
	Equivalent bool
}

// ResolveMapping resolves the Section 7.7 adapter mapping for one
// provider under one profile against the exact probed build tuple.
// The tuple passes the Section 8.4 resume gate and must name the
// resolved provider; the profile must be inside the vocabulary; the
// provider must carry a Section 7.7 row and Pi must be the pinned
// 0.73.1 build. Unmappable combinations refuse with the Section
// 2.4 profile_mapping_unavailable failure naming the provider,
// probed version, and profile; miscorrelated inputs refuse as
// caller errors.
func ResolveMapping(providerID, profile string, tuple BuildTuple) (ResolvedMapping, error) {
	empty := ResolvedMapping{}
	// The caller's provider argument is classified before the
	// tuple gate runs: garbage fails as a caller error, while a
	// well-formed provider without a Section 7.7 row (Qwen
	// direct, future plugins) is the mapping gap Section 2.4
	// names. Running the tuple gate first would misclassify
	// both as tuple refusals.
	if !validProviderID(providerID) {
		failure, err := failInvalid("mapping provider is not a provider id")
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	if _, ok := profileYOLOMapping[providerID]; !ok {
		failure, err := failMappingUnavailable("mapping has no row for this provider", providerID, tuple.ProviderVersion, profile)
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	if err := CheckResumeTuple(tuple); err != nil {
		return empty, err
	}
	if tuple.ProviderID != providerID {
		failure, err := failInvalid("mapping tuple names another provider")
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	if !profileNames[profile] {
		failure, err := failInvalid("mapping profile is not standard or yolo")
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	if providerID == "pi" && tuple.ProviderVersion != piPinnedVersion {
		failure, err := failMappingUnavailable("mapping pins pi to probed 0.73.1", providerID, tuple.ProviderVersion, profile)
		if err != nil {
			return empty, err
		}
		return empty, failure
	}
	// The flag value comes from the single table source. Both
	// refusals below it are unreachable on this path — the row and
	// the profile were just established — but the value must not be
	// retyped here.
	mapping, err := ProfileMapping(providerID, profile)
	if err != nil {
		return empty, err
	}
	return ResolvedMapping{
		ProviderID: providerID,
		Profile:    profile,
		Version:    tuple.ProviderVersion,
		Mapping:    mapping,
		Equivalent: providerID == "pi",
	}, nil
}

// ProjectLaunchArgv projects the sanitized launch argv for one
// resolved mapping over the caller base (the provider binary with
// its subcommand). The resolution must name this provider and
// profile, and the base must be structurally valid. Yolo appends
// exactly the table flag once — Pi appends nothing because its
// mapping is report-only — and standard returns the base
// unchanged. Any unrestricted token already present in the base
// refuses: the exact required flag is a duplicate, any other
// table token or the codex alias is a changed flag under yolo and
// an alias reuse under standard. The Section 5.1 argv bounds apply
// to the projected vector.
func ProjectLaunchArgv(base []string, providerID, profile string, resolved ResolvedMapping) ([]string, error) {
	if resolved.ProviderID != providerID || resolved.Profile != profile {
		failure, err := failInvalid("launch mapping does not match the provider profile")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if err := secprim.CheckArgv(base); err != nil {
		return nil, err
	}
	tokens := unrestrictedTokens()
	for _, element := range base {
		if tokens[element] {
			// The duplicate arm applies only where the mapping
			// is an argv flag: Pi's report-only mapping is
			// never appended, so its presence in a base is a
			// changed flag like any other foreign token.
			if profile == ProfileYOLO && providerID != "pi" && element == resolved.Mapping && resolved.Mapping != "" {
				failure, err := failInvalid("launch argv already carries the profile flag")
				if err != nil {
					return nil, err
				}
				return nil, failure
			}
			if profile == ProfileYOLO {
				failure, err := failInvalid("launch argv carries a changed unrestricted flag")
				if err != nil {
					return nil, err
				}
				return nil, failure
			}
			failure, err := failInvalid("launch argv reuses an unrestricted flag under standard")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
	}
	projected := append([]string(nil), base...)
	// Pi's mapping is report-only: it is never an argv token, so
	// yolo appends nothing for Pi and the disclosure travels in
	// the resolution, not the command line.
	if profile == ProfileYOLO && providerID != "pi" && resolved.Mapping != "" {
		projected = append(projected, resolved.Mapping)
	}
	if err := checkProjectedArgvBounds(projected); err != nil {
		return nil, err
	}
	return projected, nil
}

// unrestrictedTokens is the exact token set the launch projection
// refuses in a caller base: every Section 7.7 yolo flag plus the
// documented codex alias. It derives from the table, never retypes
// it: a new table row is refused in bases from the same change.
func unrestrictedTokens() map[string]bool {
	tokens := make(map[string]bool, len(profileYOLOMapping)+1)
	for _, flag := range profileYOLOMapping {
		tokens[flag] = true
	}
	tokens[codexAliasYOLO] = true
	return tokens
}

// checkProjectedArgvBounds enforces the Section 5.1 numeric argv
// bounds on the projected vector: at most 128 elements, each at
// most 4,096 bytes, 65,536 bytes total. Byte length is the
// exec-bound quantity, so elements count bytes here. This is the
// host-side twin of the wire bounds checkSpawnArgv enforces; the
// agreement test drives identical vectors at each limit through
// both entries. Structural validity (non-empty, NUL-free, UTF-8)
// stays with secprim.CheckArgv, which the projection drives
// before appending its known-good constant.
func checkProjectedArgvBounds(projected []string) error {
	if len(projected) > 128 {
		failure, err := failInvalid("launch argv exceeds 128 elements")
		if err != nil {
			return err
		}
		return failure
	}
	total := 0
	for _, element := range projected {
		if len(element) > 4096 {
			failure, err := failInvalid("launch argv element exceeds 4096 bytes")
			if err != nil {
				return err
			}
			return failure
		}
		total += len(element)
	}
	if total > maxSpawnArgvBytes {
		failure, err := failInvalid("launch argv exceeds 65536 bytes total")
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}
