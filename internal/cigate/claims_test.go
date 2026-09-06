package cigate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func unavailableStates(ids ...string) []ProbeState {
	states := make([]ProbeState, 0, len(ids))
	for _, id := range ids {
		states = append(states, ProbeState{ID: id, Available: false})
	}
	return states
}

func availableStates(ids ...string) []ProbeState {
	states := make([]ProbeState, 0, len(ids))
	for _, id := range ids {
		states = append(states, ProbeState{ID: id, Available: true})
	}
	return states
}

func TestCapabilityIDsAreDerivedSorted(t *testing.T) {
	t.Parallel()
	ids, err := CapabilityIDs()
	if err != nil {
		t.Fatalf("CapabilityIDs() error = %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("CapabilityIDs() is empty; the check would pass on an empty vocabulary")
	}
	for index, id := range ids {
		if id == "" {
			t.Fatalf("CapabilityIDs()[%d] is empty", index)
		}
		if index > 0 && ids[index-1] >= id {
			t.Fatalf("CapabilityIDs() = %v, want sorted", ids)
		}
	}
}

// TestPositiveClaimsAreRefused drives one claim per positive verb: dropping a
// verb from the gate must admit exactly its row (NARROWING battery). Each
// subtest kills by behaviour — the count when the row goes silent, the
// reason when it slides to the unclassified arm — never by a battery-size
// guard: a parent-level size check would abort before any subtest runs and
// prove only bookkeeping.
func TestPositiveClaimsAreRefused(t *testing.T) {
	t.Parallel()
	claims := map[string]string{
		"available": "The `fifo` capability is available on Windows.",
		"supported": "This release keeps `fifo` supported on every platform.",
		"enabled":   "The runner has `fifo` enabled by default.",
		"works":     "The `fifo` lane works on Windows.",
		"passes":    "The `fifo` fixture passes on Windows.",
		"capital":   "The `fifo` capability is Available on Windows.",
	}
	for verb, claim := range claims {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()
			findings, err := CheckAdvertisements(claim, unavailableStates("fifo"))
			if err != nil {
				t.Fatalf("CheckAdvertisements() error = %v", err)
			}
			if len(findings) != 1 {
				t.Fatalf("CheckAdvertisements() findings = %v, want exactly one", findings)
			}
			if findings[0].ID != "fifo" {
				t.Errorf("finding ID = %q, want fifo", findings[0].ID)
			}
			if !strings.Contains(findings[0].Reason, "positive availability claim") {
				t.Errorf("finding reason = %q, want the positive-claim refusal", findings[0].Reason)
			}
		})
	}
	// A verb added to the gate without a claim row would pass the battery
	// without ever being driven: the census fails instead.
	t.Run("battery-covers-gate", func(t *testing.T) {
		t.Parallel()
		for _, verb := range positiveAvailability {
			if _, ok := claims[verb]; !ok {
				t.Errorf("gate verb %q has no claim row; extend the battery", verb)
			}
		}
	})
}

// TestNegatedClaimsAreAdmitted drives one row per negation: dropping a
// negation must refuse exactly its honestly conditional sentence. Every row
// carries a positive verb guarded by exactly one negation, so the drop kills
// by behaviour — the positive arm fires — never by a battery-size guard. A
// row naming two negations would let one cover for the other and must not
// be written; a row without a positive verb would not redden at all.
func TestNegatedClaimsAreAdmitted(t *testing.T) {
	t.Parallel()
	sentences := map[string]string{
		"not":         "The `fifo` capability is not available on Windows.",
		"never":       "The `fifo` lane never passes on Windows.",
		"no":          "No host reports the `fifo` capability as available.",
		"without":     "The suite runs without the `fifo` capability available.",
		"cannot":      "The `fifo` probe cannot report available on Windows.",
		"can't":       "The `fifo` capability can't be enabled on Windows.",
		"n't":         "Hosts do n't advertise `fifo` as available.",
		"unavailable": "The `fifo` capability is unavailable on Windows, though the docs call it available.",
		"disabled":    "The `fifo` capability stays disabled on Windows, though the checklist marks it available.",
		"conditional": "The `fifo` capability is available conditional on the Linux lane.",
		"pending":     "The `fifo` lane will be available pending the Linux rollout.",
		"unless":      "The `fifo` lane is available on Windows unless the platform withholds it.",
		"only":        "The `fifo` capability is available only on Linux.",
	}
	for word, document := range sentences {
		t.Run(word, func(t *testing.T) {
			t.Parallel()
			findings, err := CheckAdvertisements(document, unavailableStates("fifo"))
			if err != nil {
				t.Fatalf("CheckAdvertisements() error = %v", err)
			}
			if len(findings) != 0 {
				t.Fatalf("CheckAdvertisements() findings = %v, want none", findings)
			}
		})
	}
	// A negation added to the gate without a conditional row would pass the
	// battery without ever being driven: the census fails instead.
	t.Run("battery-covers-gate", func(t *testing.T) {
		t.Parallel()
		for _, word := range negations {
			if _, ok := sentences[word]; !ok {
				t.Errorf("gate negation %q has no conditional row; extend the battery", word)
			}
		}
	})
}

// TestUnbacktickedProseIsOutsideTheScanner pins the stated bound: the gate
// matches backticked capability IDs only, so a positive availability claim
// written in bare prose is admitted unchecked. Widening the matcher to bare
// IDs must fail here.
func TestUnbacktickedProseIsOutsideTheScanner(t *testing.T) {
	t.Parallel()
	for _, document := range []string{
		"FIFO creation is available and supported on Windows.",
		"The fifo capability is available on Windows.",
	} {
		findings, err := CheckAdvertisements(document, unavailableStates("fifo"))
		if err != nil {
			t.Fatalf("CheckAdvertisements() error = %v", err)
		}
		if len(findings) != 0 {
			t.Fatalf("CheckAdvertisements(%q) findings = %v, want none: bare prose is outside the scanner", document, findings)
		}
	}
}

// TestUnclassifiedMentionIsRefused is the fail-closed arm: a sentence that
// names an unavailable capability without any classified context is refused
// rather than ignored. It is also the net for the token-preserving mutant
// that phrases a claim without any listed verb.
func TestUnclassifiedMentionIsRefused(t *testing.T) {
	t.Parallel()
	for _, document := range []string{
		"Ship the `fifo` driver tomorrow.",
		"The `fifo` lane is up on every host.",
	} {
		findings, err := CheckAdvertisements(document, unavailableStates("fifo"))
		if err != nil {
			t.Fatalf("CheckAdvertisements() error = %v", err)
		}
		if len(findings) != 1 {
			t.Fatalf("CheckAdvertisements(%q) findings = %v, want exactly one", document, findings)
		}
		if !strings.Contains(findings[0].Reason, "unclassified mention") {
			t.Errorf("finding reason = %q, want the unclassified refusal", findings[0].Reason)
		}
	}
}

func TestAvailableCapabilityAdmitsPositiveClaim(t *testing.T) {
	t.Parallel()
	findings, err := CheckAdvertisements(
		"The `fifo` capability is available on Linux.",
		availableStates("fifo"))
	if err != nil {
		t.Fatalf("CheckAdvertisements() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("available capability drew findings = %v", findings)
	}
}

func TestMixedAvailabilityFindsOnlyUnavailable(t *testing.T) {
	t.Parallel()
	findings, err := CheckAdvertisements(
		"The `symlink` and `fifo` lanes are available on Windows, and `mode-bits` stays up too.",
		[]ProbeState{
			{ID: "symlink", Available: true},
			{ID: "fifo", Available: false},
			{ID: "mode-bits", Available: false},
		})
	if err != nil {
		t.Fatalf("CheckAdvertisements() error = %v", err)
	}
	// Asserted as a set: finding order is presentation, not verdict.
	refused := map[string]bool{}
	for _, finding := range findings {
		refused[finding.ID] = true
	}
	want := map[string]bool{"fifo": true, "mode-bits": true}
	if len(refused) != len(want) {
		t.Fatalf("refused = %v, want %v", refused, want)
	}
	for id := range want {
		if !refused[id] {
			t.Fatalf("refused = %v, want %v", refused, want)
		}
	}
}

func TestEmptyInputsAreRefused(t *testing.T) {
	t.Parallel()
	if _, err := CheckAdvertisements("The `fifo` lane is up.", nil); err == nil {
		t.Error("empty probe set admitted; the gate would pass on an empty vocabulary")
	}
	if _, err := CheckAdvertisements("   \n  ", unavailableStates("fifo")); err == nil {
		t.Error("empty document admitted; the gate would pass on nothing scanned")
	}
}

// TestSubstringIsNotAMarker pins boundary anchoring: "protest" carries the
// letters t-e-s-t without the word test, and must not classify the sentence.
// A gate matching substrings would admit this claim-shaped mention.
func TestSubstringIsNotAMarker(t *testing.T) {
	t.Parallel()
	findings, err := CheckAdvertisements("Citizens protest the `fifo` cut.", unavailableStates("fifo"))
	if err != nil {
		t.Fatalf("CheckAdvertisements() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %v, want exactly the unclassified refusal", findings)
	}
	if !strings.Contains(findings[0].Reason, "unclassified mention") {
		t.Errorf("finding reason = %q, want the unclassified refusal", findings[0].Reason)
	}
}

func TestFindingCarriesLine(t *testing.T) {
	t.Parallel()
	document := "First line.\n\nThe `fifo` capability is available on Windows.\n"
	findings, err := CheckAdvertisements(document, unavailableStates("fifo"))
	if err != nil {
		t.Fatalf("CheckAdvertisements() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %v, want exactly one", findings)
	}
	if findings[0].Line != 3 {
		t.Errorf("finding line = %d, want 3", findings[0].Line)
	}
}

func readREADME(t *testing.T) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	return string(contents)
}

func allUnavailable(t *testing.T) []ProbeState {
	t.Helper()
	ids, err := CapabilityIDs()
	if err != nil {
		t.Fatalf("CapabilityIDs() error = %v", err)
	}
	return unavailableStates(ids...)
}

// TestRealREADMECarriesNoPositiveClaim runs the gate over the real
// advertisement surface with every probe forced unavailable — the strongest
// posture: every backticked capability mention must already be classified
// non-claim, or a future edit away from refusal.
func TestRealREADMECarriesNoPositiveClaim(t *testing.T) {
	t.Parallel()
	findings, err := CheckAdvertisements(readREADME(t), allUnavailable(t))
	if err != nil {
		t.Fatalf("CheckAdvertisements() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("README findings = %v, want none", findings)
	}
}

func TestRealREADMEAdmitsWhenAllAvailable(t *testing.T) {
	t.Parallel()
	ids, err := CapabilityIDs()
	if err != nil {
		t.Fatalf("CapabilityIDs() error = %v", err)
	}
	findings, err := CheckAdvertisements(readREADME(t), availableStates(ids...))
	if err != nil {
		t.Fatalf("CheckAdvertisements() error = %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("README findings = %v, want none", findings)
	}
}

// TestEveryNonClaimMarkerOccursInCorpus keeps the marker list honest: a
// marker that fires on no real sentence is either dead weight or an admission
// hole reserved for future text.
func TestEveryNonClaimMarkerOccursInCorpus(t *testing.T) {
	t.Parallel()
	document := readREADME(t)
	ids, err := CapabilityIDs()
	if err != nil {
		t.Fatalf("CapabilityIDs() error = %v", err)
	}
	probed := map[string]bool{}
	for _, id := range ids {
		probed["`"+id+"`"] = true
	}
	var mentioning []string
	for _, part := range splitSentences(document) {
		for token := range probed {
			if strings.Contains(part.text, token) {
				mentioning = append(mentioning, part.text)
				break
			}
		}
	}
	if len(mentioning) == 0 {
		t.Fatal("no corpus sentence mentions a probe ID; the marker census is vacuous")
	}
	for _, marker := range nonClaimMarkers {
		found := false
		for _, text := range mentioning {
			if wordMatches(marker, strings.ToLower(text)) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("marker %q occurs in no probe-mentioning sentence; remove it or classify with it", marker)
		}
	}
}
