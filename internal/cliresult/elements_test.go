package cliresult

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// TestEveryArrayElementRefusesEveryWrongJSONType drives each declared array of
// each implemented body with a non-conforming element. An array whose element
// type is unchecked is exactly the "skip element validation because an array is
// nested" failure Section 1.6 names.
func TestEveryArrayElementRefusesEveryWrongJSONType(t *testing.T) {
	arrays := map[Command][]string{
		CommandList:     {"sessions", "unreachable_peer_ids"},
		CommandTakeover: {"affected_session_ids", "warnings"},
		CommandSync:     {"peer_ids", "checkpoint_ids"},
		CommandDiff:     {"entries"},
		CommandDoctor:   {"findings"},
		CommandLogs:     {"events"},
		CommandPeerList: {"peers"},
	}
	for command, members := range arrays {
		for _, member := range members {
			t.Run(string(command)+"/"+member, func(t *testing.T) {
				base := validSpec(t, command).Body.(map[string]any)
				// A string is a conforming element of a string array, so it
				// is swept only where the declared element type is not one.
				elements := []any{json.Number("1"), true, nil, []any{}, map[string]any{"a": "b"}}
				if member != "warnings" && member != "degraded_codes" {
					elements = append(elements, "not-an-element")
				}
				for _, element := range elements {
					spec := specWithBody(t, command, mutateBody(base, member, []any{element}))
					if _, err := New(spec); err == nil {
						t.Fatalf("%s.%s admitted the element %T (%v)", command, member, element, element)
					}
				}
			})
		}
	}
}

// TestPathDiffNarrowsEveryDeclaredMember attacks the one closed type whose
// elements carry no ordering key, so its members are the only thing checked.
func TestPathDiffNarrowsEveryDeclaredMember(t *testing.T) {
	entry := func(overrides map[string]any) Spec {
		object := map[string]any{
			"path": "src/main.go", "classification": "modified",
			"source_digest": fixtureCheckpointID, "destination_digest": fixtureOtherDigest,
		}
		for key, value := range overrides {
			object[key] = value
		}
		base := validSpec(t, CommandDiff).Body.(map[string]any)
		return specWithBody(t, CommandDiff, mutateBody(base, "entries", []any{object}))
	}
	admit(t, "a complete path diff", entry(nil))
	admit(t, "null digests on both sides", entry(map[string]any{
		"source_digest": nil, "destination_digest": nil,
	}))
	refused := map[string]map[string]any{
		"an absolute path":          {"path": "/src/main.go"},
		"a parent segment":          {"path": "../src/main.go"},
		"a current segment":         {"path": "./src/main.go"},
		"an empty segment":          {"path": "src//main.go"},
		"a backslash separator":     {"path": `src\main.go`},
		"an encoded separator":      {"path": "src%2Fmain.go"},
		"an unknown classification": {"classification": "renamed"},
		"a digest without a prefix": {"source_digest": strings.TrimPrefix(fixtureCheckpointID, "sha256:")},
		"an uppercase digest":       {"destination_digest": "sha256:" + strings.ToUpper(strings.TrimPrefix(fixtureCheckpointID, "sha256:"))},
		"a numeric path":            {"path": json.Number("1")},
		"an extra member":           {"mode": "0644"},
	}
	for name, overrides := range refused {
		t.Run(name, func(t *testing.T) {
			refuse(t, name, entry(overrides))
		})
	}
	t.Run("a missing member", func(t *testing.T) {
		object := map[string]any{"path": "src/main.go", "classification": "modified", "source_digest": nil}
		base := validSpec(t, CommandDiff).Body.(map[string]any)
		refuse(t, "a path diff without destination_digest",
			specWithBody(t, CommandDiff, mutateBody(base, "entries", []any{object})))
	})
}

// TestCapabilityVocabularyProjectionRefusesADriftedCatalog narrows the
// projection that builds the capability-name vocabulary. The vocabulary is
// derived from the reviewed catalog rather than retyped, so a drifted catalog
// must fail at build rather than silently widen or narrow the admitted set.
func TestCapabilityVocabularyProjectionRefusesADriftedCatalog(t *testing.T) {
	live := catalog.Current().Capabilities
	if _, err := providerCapabilityNamesFrom(live); err != nil {
		t.Fatalf("the reviewed catalog does not project: %v", err)
	}
	var provider []catalog.Capability
	for _, capability := range live {
		if capability.Family == "provider" {
			provider = append(provider, capability)
		}
	}
	if len(provider) != maxSessionCapabilities {
		t.Fatalf("provider family has %d names, want %d", len(provider), maxSessionCapabilities)
	}
	if _, err := providerCapabilityNamesFrom(provider[:len(provider)-1]); err == nil {
		t.Fatalf("a six-name provider family projected without complaint")
	}
	widened := append(append([]catalog.Capability(nil), provider...), catalog.Capability{
		Family: "provider", Name: "invented_capability",
	})
	if _, err := providerCapabilityNamesFrom(widened); err == nil {
		t.Fatalf("an eight-name provider family projected without complaint")
	}
	duplicated := append(append([]catalog.Capability(nil), provider...), provider[0])
	if _, err := providerCapabilityNamesFrom(duplicated); err == nil {
		t.Fatalf("a duplicated capability name projected without complaint")
	}
	if _, err := providerCapabilityNamesFrom(nil); err == nil {
		t.Fatalf("an empty capability catalog projected without complaint")
	}
}

// TestBoundedStringsRefuseInvalidUTF8AndTheirLimits narrows the string bound
// helper on both ends and on encoding validity.
func TestBoundedStringsRefuseInvalidUTF8AndTheirLimits(t *testing.T) {
	base := validSpec(t, CommandResume).Body.(map[string]any)
	admit(t, "a 512 character native session id", specWithBody(t, CommandResume,
		mutateBody(base, "native_session_id", strings.Repeat("s", 512))))
	refuse(t, "a 513 character native session id", specWithBody(t, CommandResume,
		mutateBody(base, "native_session_id", strings.Repeat("s", 513))))
	refuse(t, "an empty native session id", specWithBody(t, CommandResume,
		mutateBody(base, "native_session_id", "")))
	// A multi-byte string is bounded in UTF-8 characters, not bytes.
	admit(t, "512 multi-byte characters", specWithBody(t, CommandResume,
		mutateBody(base, "native_session_id", strings.Repeat("é", 512))))
	refuse(t, "513 multi-byte characters", specWithBody(t, CommandResume,
		mutateBody(base, "native_session_id", strings.Repeat("é", 513))))
	// A lone surrogate never survives the strict common data model.
	refuse(t, "a lone surrogate", specWithBody(t, CommandResume,
		mutateBody(base, "native_session_id", string([]byte{0xed, 0xa0, 0x80}))))
}
