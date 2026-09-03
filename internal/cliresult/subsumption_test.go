package cliresult

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestCapabilityCountBoundIsSubsumedByTheVocabulary pins the invariant that
// makes the SessionSummary capability count bound unreachable: the vocabulary
// has exactly as many names as the bound admits, so a map of admitted names
// cannot exceed it and the eighth entry is refused by name first.
//
// The bound is retained in production so the Section 14.2 constraint stays
// visible. If the vocabulary ever grows past the bound, this test fails and the
// count check stops being subsumed.
func TestCapabilityCountBoundIsSubsumedByTheVocabulary(t *testing.T) {
	if len(providerCapabilityNames) != maxSessionCapabilities {
		t.Fatalf("capability vocabulary has %d names and the bound is %d; the count check is no longer subsumed",
			len(providerCapabilityNames), maxSessionCapabilities)
	}
}

// TestUnimplementedBodiesAreRefusedOnTheReadingSideToo closes the reader's own
// path to an unbuilt body. A clone document carries a tag that selects CLI
// Result 2.0.0, which this repository does implement as an envelope version, so
// nothing before the body dispatch refuses it - and admitting it would hand a
// caller an object no validator ever checked.
func TestUnimplementedBodiesAreRefusedOnTheReadingSideToo(t *testing.T) {
	document := map[string]any{
		"schema":         Schema,
		"schema_version": string(Version200),
		"command":        string(CommandCloneRun),
		"ok":             true,
		"operation_id":   fixtureOperationID,
		"session_id":     nil,
		"body":           map[string]any{"outcome": "archive_created"},
		"extensions":     map[string]any{},
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	_, err = Decode(Version200, encoded)
	if !errors.Is(err, ErrUnimplementedVersion) {
		t.Fatalf("Decode(clone document) = %v, want ErrUnimplementedVersion", err)
	}
	if err == nil || !strings.Contains(err.Error(), "whose body this repository does not build") {
		t.Fatalf("refusal does not name the missing body: %v", err)
	}
	// The same applies to a Directory or TerminalBackend tag, which are refused
	// one step earlier because their versions are not implemented at all.
	for _, version := range []Version{Version300, Version400} {
		document["schema_version"] = string(version)
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if _, err := Decode(version, encoded); !errors.Is(err, ErrUnimplementedVersion) {
			t.Fatalf("Decode(%s) = %v, want ErrUnimplementedVersion", version, err)
		}
	}
}

// spoofedString marshals to a conforming JSON string while being a type the
// common data model does not admit. It exists to prove that the input walk
// refuses a value by its Go type rather than by what encoding/json happens to
// produce for it: a custom Marshaler can emit anything, so a validator that
// only inspected the encoded form would take whatever the type chose to say.
type spoofedString struct{}

func (spoofedString) MarshalJSON() ([]byte, error) { return []byte(`"payments-api"`), nil }

// TestInputWalkRefusesATypeOutsideTheCommonDataModel narrows the input
// vocabulary. Each value below encodes to something a validator would accept,
// so only a check on the Go type refuses them.
func TestInputWalkRefusesATypeOutsideTheCommonDataModel(t *testing.T) {
	base := validSpec(t, CommandCancel).Body.(map[string]any)
	for name, value := range map[string]any{
		"a custom marshaler":  spoofedString{},
		"a float64":           float64(1),
		"a float32":           float32(1),
		"a byte slice":        []byte("payments-api"),
		"a typed string":      json.RawMessage(`"payments-api"`),
		"a map with any keys": map[any]any{},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := New(specWithBody(t, CommandCancel, mutateBody(base, "name", value))); err == nil {
				t.Fatalf("New admitted %s", name)
			}
		})
	}
	// The admitted Go integer forms all reach the validators as JSON numbers.
	epochs := []any{5, int32(5), int64(5), uint(5), uint32(5), uint64(5), json.Number("5")}
	for _, epoch := range epochs {
		summary := sessionSummary(fixtureSessionID)
		summary["lease_epoch"] = epoch
		spec := specWithBody(t, CommandStart, map[string]any{
			"session": summary, "execution_profile": "yolo", "terminal_backend": "tmux",
		})
		result, err := New(spec)
		if err != nil {
			t.Fatalf("New refused the integer form %T: %v", epoch, err)
		}
		nested := result.Body()["session"].(map[string]any)
		if nested["lease_epoch"].(json.Number).String() != "5" {
			t.Fatalf("integer form %T became %v", epoch, nested["lease_epoch"])
		}
	}
}
