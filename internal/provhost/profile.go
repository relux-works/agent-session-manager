package provhost

// This file maps execution profiles to provider adapter flags under
// Section 7.7. The v0.3.0 yolo mappings below are exact adapter
// strings the host persists into SpawnPlans; standard uses the
// provider's normal approval and sandbox behavior and omits every
// unrestricted flag. The adapter MUST probe the exact provider
// version before applying a mapping, and an absent or changed flag
// fails closed: unknown providers and unknown profiles are refused,
// never defaulted.

// Execution profiles Section 7.7 defines.
const (
	ProfileStandard = "standard"
	ProfileYOLO     = "yolo"
)

// profileProviders lists the six Section 7.7 providers in table
// order. Pi has no permission-mode flag: both of its profiles resolve
// to the default full tool set, and probe and launch output MUST
// disclose that equivalence rather than invent a flag.
var profileProviders = []string{
	"codex",
	"claude",
	"gemini",
	"muse",
	"antigravity",
	"pi",
}

// profileYOLOMapping is the required yolo adapter flag per provider.
// Standard maps to the empty string for every provider: no flag, no
// alias that silently expands to unrestricted mode.
var profileYOLOMapping = map[string]string{
	"codex":       "--dangerously-bypass-approvals-and-sandbox",
	"claude":      "--dangerously-skip-permissions",
	"gemini":      "--approval-mode=yolo",
	"muse":        "--yolo",
	"antigravity": "--dangerously-skip-permissions",
	"pi":          "default_unrestricted_tool_set",
}

// profileNames is the exact Section 7.7 execution-profile vocabulary:
// standard and yolo, nothing else. It is package-level so the closed
// vocabulary census derives it: admitting a third profile here must
// redden TestProfileNamesAreDerivedFromSpec, not resolve to a flag.
var profileNames = map[string]bool{
	ProfileStandard: true,
	ProfileYOLO:     true,
}

// ProfileMapping resolves the adapter flag for one provider under one
// profile. Standard returns the empty string: the caller omits every
// unrestricted flag. An unknown provider or profile is an
// invalid_config caller error; there is no default mapping a new
// provider could silently inherit.
func ProfileMapping(providerID, profile string) (string, error) {
	mapping, known := profileYOLOMapping[providerID]
	if !known {
		failure, err := failInvalid("profile mapping names an unknown provider")
		if err != nil {
			return "", err
		}
		return "", failure
	}
	if !profileNames[profile] {
		failure, err := failInvalid("profile mapping names an unknown profile")
		if err != nil {
			return "", err
		}
		return "", failure
	}
	if profile == ProfileYOLO {
		return mapping, nil
	}
	return "", nil
}
