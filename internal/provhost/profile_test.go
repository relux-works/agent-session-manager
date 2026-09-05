package provhost

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/provider"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// TestProfileMappingMatchesSection77 sweeps the full provider-by-
// profile domain: six providers times two profiles, all twelve
// measured. Yolo returns the exact adapter flag; standard returns
// the empty omission for every provider, including Pi, whose yolo
// and standard resolve to the same disclosed tool set rather than a
// flag.
func TestProfileMappingMatchesSection77(t *testing.T) {
	wantYOLO := map[string]string{
		"codex":       "--dangerously-bypass-approvals-and-sandbox",
		"claude":      "--dangerously-skip-permissions",
		"gemini":      "--approval-mode=yolo",
		"muse":        "--yolo",
		"antigravity": "--dangerously-skip-permissions",
		"pi":          "default_unrestricted_tool_set",
	}
	if len(profileProviders) != len(wantYOLO) {
		t.Fatalf("profileProviders holds %d providers, want %d", len(profileProviders), len(wantYOLO))
	}
	for _, provider := range profileProviders {
		want, ok := wantYOLO[provider]
		if !ok {
			t.Fatalf("provider %q has no pinned yolo mapping; the sweep is blind", provider)
		}
		got, err := ProfileMapping(provider, ProfileYOLO)
		if err != nil {
			t.Fatalf("ProfileMapping(%q, yolo): %v", provider, err)
		}
		if got != want {
			t.Fatalf("ProfileMapping(%q, yolo) = %q, want %q", provider, got, want)
		}
		got, err = ProfileMapping(provider, ProfileStandard)
		if err != nil {
			t.Fatalf("ProfileMapping(%q, standard): %v", provider, err)
		}
		if got != "" {
			t.Fatalf("ProfileMapping(%q, standard) = %q, want the empty omission", provider, got)
		}
	}
	t.Logf("profile mapping coverage: %d providers x 2 profiles measured", len(profileProviders))
}

// TestProfileMappingRefusesUnknowns proves the fail-closed side:
// unknown providers, unknown profiles, and empties are refused, never
// defaulted. The provider near-misses derive from the registry by
// case, affix, and truncation; the open-ended complement beyond them
// is refused by construction (map miss) and witnessed here.
func TestProfileMappingRefusesUnknowns(t *testing.T) {
	for _, provider := range []string{"", "Codex", "codex ", " codex", "codexx", "cod", "qwen", "future"} {
		requireLocalRefusal(t, mustProfileMapping(t, provider, true), "invalid_config", "unknown provider")
	}
	for _, profile := range []string{"", "YOLO", "yolo ", "unrestricted", "standard-yolo"} {
		requireLocalRefusal(t, mustProfileMapping(t, profile, false), "invalid_config", "unknown profile")
	}
}

// mustProfileMapping calls ProfileMapping with the unknown in the
// provider seat (providerSeat true) or the profile seat.
func mustProfileMapping(t *testing.T, unknown string, providerSeat bool) error {
	t.Helper()
	if providerSeat {
		_, err := ProfileMapping(unknown, ProfileYOLO)
		return err
	}
	_, err := ProfileMapping("codex", unknown)
	return err
}

// TestProfileMappingIsPinnedToSection77 proves the table this file
// transcribes: six provider rows in the Section 7.7 window, and every
// yolo flag quoted verbatim from the pinned document.
func TestProfileMappingIsPinnedToSection77(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.7", 3429, 3436)
	rows := 0
	for _, line := range strings.Split(window, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "| Codex ") || strings.HasPrefix(trimmed, "| Claude ") || strings.HasPrefix(trimmed, "| Gemini ") || strings.HasPrefix(trimmed, "| Muse ") || strings.HasPrefix(trimmed, "| Antigravity ") || strings.HasPrefix(trimmed, "| Pi ") {
			rows++
		}
	}
	if rows != len(profileProviders) {
		t.Fatalf("Section 7.7 holds %d provider rows, want %d", rows, len(profileProviders))
	}
	for _, excerpt := range []string{
		"| Codex | <code>--dangerously-bypass-approvals-and-sandbox</code> (alias",
		"| Claude | <code>--dangerously-skip-permissions</code> |",
		"| Gemini CLI | <code>--approval-mode=yolo</code> |",
		"| Muse | <code>--yolo</code> |",
		"| Antigravity | <code>--dangerously-skip-permissions</code> |",
		"| Pi 0.73.1 | No invented flag;",
		"MUST omit every unrestricted flag above",
		"An absent or changed flag fails closed",
	} {
		requireQuote(t, document, excerpt, "7.7")
	}
}

// TestSixProviderSetMatchesDiscoveryRegistry closes the six-provider
// set structurally: the profile plane's registry and the discovery
// plane's builtins agree only transitively today, each derived from
// the pinned Section 7.1/7.7 text. A provider added to one plane but
// not the other reddens here at the next spec bump instead of
// splitting launch (which resolves flags here) from discovery (which
// enumerates there). Order is each plane's own contract (table order
// here, discovery order there), so the comparison is over sets.
func TestSixProviderSetMatchesDiscoveryRegistry(t *testing.T) {
	builtins := provider.Builtins()
	if len(profileProviders) != len(builtins) {
		t.Fatalf("profile registry holds %d providers, discovery holds %d", len(profileProviders), len(builtins))
	}
	seen := make(map[string]int, len(builtins))
	for _, id := range builtins {
		seen[id]++
	}
	for _, id := range profileProviders {
		seen[id]--
	}
	for id, balance := range seen {
		if balance != 0 {
			t.Fatalf("provider %q is owned by only one plane (balance %d)", id, balance)
		}
	}
}
