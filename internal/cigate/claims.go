package cigate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/secconftest"
)

// ProbeState is one platform-capability probe outcome. IDs always come from
// secconftest.DefaultCapabilities; this package defines no capability of its
// own.
type ProbeState struct {
	ID        string
	Available bool
}

// Finding is one refused advertisement mention: a backticked probe ID in a
// sentence the gate cannot admit.
type Finding struct {
	ID     string
	Line   int
	Text   string
	Reason string
}

// positiveAvailability are the verb phrases that state a capability is
// provided. Each is pinned by a dedicated claim row: dropping one admits the
// matching claim silently.
var positiveAvailability = []string{
	"available",
	"supported",
	"enabled",
	"works",
	"passes",
}

// negations flip a positive phrase into a non-claim. Each is pinned by a
// dedicated row: dropping one refuses an honestly conditional sentence.
var negations = []string{
	"not",
	"never",
	"no",
	"without",
	"cannot",
	"can't",
	"n't",
	"unavailable",
	"disabled",
	"conditional",
	"pending",
	"unless",
	"only",
}

// nonClaimMarkers are descriptive words showing a sentence documents the
// instrument (probes, skips, gates, tests) rather than advertising a
// capability. Every marker must occur in at least one scanned corpus sentence
// (see TestEveryNonClaimMarkerOccursInCorpus): a marker that fires nowhere is
// either dead weight or an admission hole waiting for future text.
var nonClaimMarkers = []string{
	"probes",
	"capability",
	"attempt",
	"gates",
	"test",
	"witness",
	"skip",
	"strict",
	"platform",
	"lstat",
	"mkfifo",
	"chmod",
	"uid",
	"pinned",
	"rejected",
	"fixture",
	"denial",
	"fail",
	"check",
	"scan",
}

// wordPatterns precompiles every gate word once at init. The compiled set is
// read-only afterwards, so parallel checks share it without a lock; a lazily
// filled cache here would race under t.Parallel.
var wordPatterns = func() map[string]*regexp.Regexp {
	patterns := map[string]*regexp.Regexp{}
	for _, word := range append(append(append([]string(nil), positiveAvailability...), negations...), nonClaimMarkers...) {
		patterns[word] = regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	}
	return patterns
}()

func wordMatches(word, text string) bool {
	pattern, ok := wordPatterns[word]
	if !ok {
		return false
	}
	return pattern.MatchString(text)
}

func containsAnyFold(words []string, text string) bool {
	lowered := strings.ToLower(text)
	for _, word := range words {
		if wordMatches(word, lowered) {
			return true
		}
	}
	return false
}

// ProbeStates runs every secconftest capability probe on this host and
// returns the outcomes. IDs are derived, never retyped; an empty derivation
// is refused rather than treated as nothing to check.
func ProbeStates() ([]ProbeState, error) {
	capabilities := secconftest.DefaultCapabilities()
	if len(capabilities) == 0 {
		return nil, fmt.Errorf("cigate: capability probe derivation is empty")
	}
	states := make([]ProbeState, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability.ID == "" || capability.Probe == nil {
			return nil, fmt.Errorf("cigate: capability without identity or probe is unusable")
		}
		available, _ := capability.Probe()
		states = append(states, ProbeState{ID: capability.ID, Available: available})
	}
	sort.Slice(states, func(i, j int) bool { return states[i].ID < states[j].ID })
	return states, nil
}

// CapabilityIDs derives the probe vocabulary: the sorted IDs of every
// secconftest capability. It is the only vocabulary the advertisement check
// may name.
func CapabilityIDs() ([]string, error) {
	states, err := ProbeStates()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(states))
	for _, state := range states {
		ids = append(ids, state.ID)
	}
	return ids, nil
}

// sentence is one classified unit with its 1-based starting line.
type sentence struct {
	text string
	line int
}

// splitSentences unwraps blank-line-separated blocks onto single lines and
// splits them at sentence-terminal punctuation. Offsets are approximate by
// construction; findings carry the starting line so a human can locate the
// mention exactly.
func splitSentences(document string) []sentence {
	var sentences []sentence
	normalized := strings.ReplaceAll(document, "\r\n", "\n")
	raw := strings.Split(normalized, "\n")
	for start := 0; start < len(raw); {
		if strings.TrimSpace(raw[start]) == "" {
			start++
			continue
		}
		end := start
		for end < len(raw) && strings.TrimSpace(raw[end]) != "" {
			end++
		}
		var cleaned []string
		var rows []int
		for row := start; row < end; row++ {
			trimmed := strings.TrimSpace(raw[row])
			if trimmed == "" {
				continue
			}
			cleaned = append(cleaned, trimmed)
			rows = append(rows, row+1)
		}
		start = end
		if len(cleaned) == 0 {
			continue
		}
		joined := strings.Join(cleaned, " ")
		lineOf := make([]int, len(joined))
		offset := 0
		for index, line := range cleaned {
			for range line {
				if offset < len(lineOf) {
					lineOf[offset] = rows[index]
				}
				offset++
			}
			if offset < len(lineOf) {
				lineOf[offset] = rows[index]
				offset++
			}
		}
		start := 0
		for index := 0; index < len(joined); index++ {
			end := index == len(joined)-1
			boundary := end ||
				((joined[index] == '.' || joined[index] == '!' || joined[index] == '?') &&
					(index+1 >= len(joined) || joined[index+1] == ' ' || joined[index+1] == '\t'))
			if !boundary {
				continue
			}
			text := strings.TrimSpace(joined[start : index+1])
			if text != "" {
				sentences = append(sentences, sentence{text: text, line: lineOf[start]})
			}
			start = index + 1
		}
	}
	return sentences
}

// CheckAdvertisements refuses every sentence that advertises a capability the
// probes report unavailable. An empty probe set or an empty document is
// refused outright: a gate that finds nothing to check, or nothing to check
// against, passes on an empty result set, which is the defect class this
// story removes.
func CheckAdvertisements(document string, states []ProbeState) ([]Finding, error) {
	if len(states) == 0 {
		return nil, fmt.Errorf("cigate: no probe outcomes to check advertisements against")
	}
	if strings.TrimSpace(document) == "" {
		return nil, fmt.Errorf("cigate: empty document proves no advertisement claim")
	}
	availability := make(map[string]bool, len(states))
	for _, state := range states {
		availability[state.ID] = state.Available
	}
	var findings []Finding
	for _, part := range splitSentences(document) {
		var mentioned []string
		for id := range availability {
			if strings.Contains(part.text, "`"+id+"`") {
				mentioned = append(mentioned, id)
			}
		}
		if len(mentioned) == 0 {
			continue
		}
		sort.Strings(mentioned)
		var unavailable []string
		for _, id := range mentioned {
			if !availability[id] {
				unavailable = append(unavailable, id)
			}
		}
		if len(unavailable) == 0 {
			continue
		}
		if containsAnyFold(positiveAvailability, part.text) &&
			!containsAnyFold(negations, part.text) {
			for _, id := range unavailable {
				findings = append(findings, Finding{
					ID:     id,
					Line:   part.line,
					Text:   part.text,
					Reason: "positive availability claim for an unavailable capability",
				})
			}
			continue
		}
		if !containsAnyFold(append(append([]string(nil), negations...), nonClaimMarkers...), part.text) {
			for _, id := range unavailable {
				findings = append(findings, Finding{
					ID:     id,
					Line:   part.line,
					Text:   part.text,
					Reason: "unclassified mention of an unavailable capability",
				})
			}
		}
	}
	return findings, nil
}
