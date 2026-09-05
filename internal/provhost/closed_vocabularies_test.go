package provhost

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// This file proves the eight closed vocabularies the manifest,
// keyed-operation, and profile-table derivations do not cover are
// exactly the Section 5.5, 7.4, and 7.5 enums: every list below is
// re-read from the pinned document, never from a hand transcription,
// so a provider value the contract never defined reddens here
// rather than passing silently. Sweeping the known members proves
// the rule over those members, not that the vocabulary contains
// exactly those members; these tests prove the containment.

// codeSpanPattern matches one inline code element, which is how the
// pinned document writes every vocabulary member.
var codeSpanPattern = regexp.MustCompile(`<code>([^<>]*)</code>`)

// codeSpansOf extracts the code elements of a spec window in order.
// An empty window or a window with no code elements is a blind
// check, so both fail outright.
func codeSpansOf(t *testing.T, window string) []string {
	t.Helper()
	if strings.TrimSpace(window) == "" {
		t.Fatal("spec window is empty; the check is blind")
	}
	spans := codeSpanPattern.FindAllStringSubmatch(window, -1)
	if len(spans) == 0 {
		t.Fatal("spec window holds no code elements; the check is blind")
	}
	var members []string
	for _, span := range spans {
		members = append(members, span[1])
	}
	return members
}

// alternativesOf parses one pipe-separated alternative list out of a
// Section 7.5 table row: the text after marker up to the next comma,
// split on the document's pipe entity. A missing marker or an empty
// list is a blind check, so both fail outright.
func alternativesOf(t *testing.T, row, marker string) []string {
	t.Helper()
	start := strings.Index(row, marker)
	if start < 0 {
		t.Fatalf("Section 7.5 row holds no %q; the check is blind", marker)
	}
	rest := row[start+len(marker):]
	if end := strings.Index(rest, ","); end >= 0 {
		rest = rest[:end]
	}
	if end := strings.Index(rest, "}"); end >= 0 {
		rest = rest[:end]
	}
	members := strings.Split(rest, "&#124;")
	var cleaned []string
	for _, member := range members {
		member = strings.TrimSpace(member)
		if member == "" {
			t.Fatalf("Section 7.5 row %q parses to an empty member; the check is blind", marker)
		}
		cleaned = append(cleaned, member)
	}
	if len(cleaned) == 0 {
		t.Fatalf("Section 7.5 row %q parses to no members; the check is blind", marker)
	}
	return cleaned
}

// requireVocabulary asserts the implementation holds exactly the
// spec-derived members in the derived order. The comparison is
// reflect.DeepEqual over the slices themselves: a %v rendering
// cannot tell [a b] from [a, b], so a formatting comparison would
// admit a member the contract never defined.
func requireVocabulary(t *testing.T, name string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %v, want the pinned vocabulary %v", name, got, want)
	}
	t.Logf("%s coverage: %d/%d vocabulary members derived", name, len(got), len(want))
}

// TestProbeStatusVocabularyIsDerivedFromSpec proves the closed
// status vocabulary equals the Section 7.4 sentence: a widened
// implementation admitting "partial" reddens here.
func TestProbeStatusVocabularyIsDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.4", 2869, 2870)
	requireQuote(t, document, "Status is one of", "7.4")
	requireVocabulary(t, "probeCapabilityStatuses", probeCapabilityStatuses, codeSpansOf(t, window))
}

// TestProbeEvidenceVocabularyIsDerivedFromSpec proves the closed
// evidence vocabulary equals the Section 7.4 sentence: a widened
// implementation admitting "assumed" reddens here.
func TestProbeEvidenceVocabularyIsDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.4", 2872, 2874)
	requireQuote(t, document, "Evidence is one of", "7.4")
	requireVocabulary(t, "probeCapabilityEvidence", probeCapabilityEvidence, codeSpansOf(t, window))
}

// TestProbeArchitecturesAreDerivedFromSpec proves the closed
// architecture vocabulary equals both Section 7.4 prose and the
// Section 7.5 probe request row: a widened implementation admitting
// "386" reddens here.
func TestProbeArchitecturesAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.4", 2862, 2862)
	requireQuote(t, document, "architecture is", "7.4")
	requireVocabulary(t, "probeArchitectures", probeArchitectures, codeSpansOf(t, window))
	row := sectionLines(t, document, "7.5", 3073, 3073)
	requireVocabulary(t, "probeArchitectures", probeArchitectures, alternativesOf(t, row, "architecture:"))
}

// TestProbePlatformsAreDerivedFromSpec proves the closed probe
// platform vocabulary equals the Section 7.5 probe request row. The
// row lists macos first while the implementation holds sorted
// order, so the comparison is sorted on both sides: a widened
// implementation admitting "freebsd" reddens here.
func TestProbePlatformsAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	row := sectionLines(t, document, "7.5", 3073, 3073)
	derived := alternativesOf(t, row, "platform:")
	sort.Strings(derived)
	got := append([]string(nil), probePlatforms...)
	sort.Strings(got)
	requireVocabulary(t, "probePlatforms", got, derived)
}

// TestQuiesceBlockersAreDerivedFromSpec proves the closed blocker
// enum equals the Section 7.5 sentence: a widened implementation
// admitting "other" reddens here.
func TestQuiesceBlockersAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "7.5", 2907, 2910)
	requireQuote(t, document, "blockers", "7.5")
	spans := codeSpansOf(t, window)
	if spans[0] != "SafeBoundaryProof.blockers" {
		t.Fatalf("blocker enum window opens with %q, want the enum owner; the check is blind", spans[0])
	}
	requireVocabulary(t, "quiesceBlockers", quiesceBlockers, spans[1:])
}

// TestIdentityKindsAreDerivedFromSpec proves the closed
// identity-kind enum equals the Section 5.5 table row: a widened
// implementation admitting "legacy_alias" reddens here.
func TestIdentityKindsAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	window := sectionLines(t, document, "5.5", 2086, 2086)
	spans := codeSpansOf(t, window)
	if len(spans) == 0 || spans[0] != "identity_kind" {
		t.Fatalf("identity-kind row opens with %v, want the member cell; the check is blind", spans)
	}
	requireVocabulary(t, "identityKinds", identityKinds, spans[1:])
}

// TestIdentifyVocabulariesAreDerivedFromSpec proves the closed
// identify-session confidence and matched-evidence vocabularies
// equal the Section 7.5 row: a widened implementation admitting
// "guess" or "guessed" reddens here.
func TestIdentifyVocabulariesAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	row := sectionLines(t, document, "7.5", 3075, 3075)
	requireVocabulary(t, "identifyConfidences", identifyConfidences, alternativesOf(t, row, "confidence:"))
	evidence := alternativesOf(t, row, "matched_evidence:sorted unique ")
	for index, member := range evidence {
		if bracket := strings.Index(member, "["); bracket >= 0 {
			evidence[index] = member[:bracket]
		}
	}
	requireVocabulary(t, "identifyEvidence", identifyEvidence, evidence)
}

// identifyResultBody renders the Section 5.5 example inside the
// identify-session success shape: the same body
// TestDecodeIdentifyResultFixtures drives.
func identifyResultBody(t *testing.T) []byte {
	t.Helper()
	indented := strings.ReplaceAll(specIdentityExample, "\n", "\n  ")
	return []byte(specIdentifyResultPrefix + "  " + strings.TrimPrefix(indented, "  ") + `,
  "confidence": "exact",
  "matched_evidence": ["native_id"]
}`)
}

// identifyResultVariant rewrites one unique substring of that body.
func identifyResultVariant(t *testing.T, old, new string) []byte {
	t.Helper()
	full := string(identifyResultBody(t))
	if strings.Count(full, old) != 1 {
		t.Fatalf("identify variant anchor %q is not unique", old)
	}
	return []byte(strings.Replace(full, old, new, 1))
}

// TestClosedVocabularyWideningsRefuse drives the exact widened
// members the derivation tests pin through the production entry
// points: each is refused at its own arm, so a vocabulary that
// admits one more value reddens twice — once here, once above.
func TestClosedVocabularyWideningsRefuse(t *testing.T) {
	t.Run("probe status partial", func(t *testing.T) {
		body := probeVariant(t, `"status": "unknown",
      "enabled": false,
      "evidence": "none",
      "detail": "not claimed for v0.3.0"`, `"status": "partial",
      "enabled": false,
      "evidence": "none",
      "detail": "not claimed for v0.3.0"`)
		requireFrameRefusal(t, DecodeProbe(body), "capabilities", "status is not a registry member")
	})
	t.Run("probe evidence assumed", func(t *testing.T) {
		body := probeVariant(t, `"evidence": "probed",`, `"evidence": "assumed",`)
		requireFrameRefusal(t, DecodeProbe(body), "capabilities", "evidence is not a registry member")
	})
	t.Run("probe architecture 386", func(t *testing.T) {
		body := probeVariant(t, `"architecture": "arm64"`, `"architecture": "386"`)
		requireFrameRefusal(t, DecodeProbe(body), "architecture", "not amd64 or arm64")
	})
	t.Run("probe platform freebsd", func(t *testing.T) {
		body := probeVariant(t, `"platform": "macos"`, `"platform": "freebsd"`)
		requireFrameRefusal(t, DecodeProbe(body), "platform", "not a registry member")
	})
	t.Run("quiesce blocker other", func(t *testing.T) {
		body := quiesceVariant(t, `"blockers": []`, `"blockers": ["other"]`)
		requireFrameRefusal(t, quiesceErr(body), "blockers", "unknown blocker")
	})
	t.Run("identity kind legacy_alias", func(t *testing.T) {
		body := identityVariant(t, `"identity_kind": "backend_conversation_uuid"`, `"identity_kind": "legacy_alias"`)
		requireFrameRefusal(t, CheckIdentity(body, "antigravity"), "identity_kind", "not a registry member")
	})
	t.Run("identify confidence guess", func(t *testing.T) {
		body := identifyResultVariant(t, `"confidence": "exact"`, `"confidence": "guess"`)
		requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "confidence", "not exact strong or weak")
	})
	t.Run("identify evidence guessed", func(t *testing.T) {
		body := identifyResultVariant(t, `"matched_evidence": ["native_id"]`, `"matched_evidence": ["guessed"]`)
		requireFrameRefusal(t, DecodeIdentifyResult(body, "antigravity"), "matched_evidence", "unknown member")
	})
}
