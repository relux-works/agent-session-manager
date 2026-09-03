package cliresult

import (
	"encoding/json"
	"testing"
)

// wrongTypedValues are the JSON forms a member of any other type can take. A
// validator that assumed a member's type would either admit one of these or
// panic on the type assertion; refusing all of them is what makes the closed
// shape a gate rather than a convention.
var wrongTypedValues = []any{
	json.Number("1"), "text", true, nil, []any{}, map[string]any{"a": "b"},
}

// TestEveryBodyMemberRefusesEveryWrongJSONType drives the production reader
// with each member of each implemented command replaced by every other JSON
// type, one at a time. The rest of the document stays exactly the object the
// positive test admitted, so each refusal narrows one member's type check.
func TestEveryBodyMemberRefusesEveryWrongJSONType(t *testing.T) {
	for _, command := range ImplementedCommands() {
		t.Run(string(command), func(t *testing.T) {
			spec := validSpec(t, command)
			base := spec.Body.(map[string]any)
			for member, original := range base {
				for _, replacement := range wrongTypedValues {
					if admits(string(command)+"."+member, original, replacement) {
						continue
					}
					mutated := specWithBody(t, command, mutateBody(base, member, replacement))
					if _, err := New(mutated); err == nil {
						t.Fatalf("member %q admitted %T (%v)", member, replacement, replacement)
					}
				}
			}
		})
	}
}

// TestEverySessionSummaryMemberRefusesEveryWrongJSONType does the same for the
// most widely nested closed type in Section 14.2.
func TestEverySessionSummaryMemberRefusesEveryWrongJSONType(t *testing.T) {
	base := sessionSummary(fixtureSessionID)
	for member, original := range base {
		for _, replacement := range wrongTypedValues {
			if admits("session_summary."+member, original, replacement) {
				continue
			}
			summary := cloneObject(base)
			summary[member] = replacement
			spec := specWithBody(t, CommandStart, map[string]any{
				"session": summary, "execution_profile": "yolo", "terminal_backend": "tmux",
			})
			if _, err := New(spec); err == nil {
				t.Fatalf("session summary member %q admitted %T (%v)", member, replacement, replacement)
			}
		}
	}
}

// TestEveryPeerSummaryMemberRefusesEveryWrongJSONType covers the second nested
// closed type.
func TestEveryPeerSummaryMemberRefusesEveryWrongJSONType(t *testing.T) {
	base := peerSummary(fixtureSourceHostID)
	for member, original := range base {
		for _, replacement := range wrongTypedValues {
			if admits("peer_summary."+member, original, replacement) {
				continue
			}
			peer := cloneObject(base)
			peer[member] = replacement
			spec := specWithBody(t, CommandPeerList, map[string]any{"peers": []any{peer}})
			if _, err := New(spec); err == nil {
				t.Fatalf("peer summary member %q admitted %T (%v)", member, replacement, replacement)
			}
		}
	}
}

// TestEveryMaterializationMemberRefusesEveryWrongJSONType covers the summary
// that is itself a whole command body.
func TestEveryMaterializationMemberRefusesEveryWrongJSONType(t *testing.T) {
	base := materializationSummary()
	for member, original := range base {
		for _, replacement := range wrongTypedValues {
			if admits("materialization."+member, original, replacement) {
				continue
			}
			spec := specWithBody(t, CommandMaterialize, mutateBody(base, member, replacement))
			if _, err := New(spec); err == nil {
				t.Fatalf("materialization member %q admitted %T (%v)", member, replacement, replacement)
			}
		}
	}
}

// nullableMembers is the reviewed list of members Section 14.2 types T|null,
// with the JSON kind their non-null form takes. Both forms are admitted, so the
// sweep above skips exactly those two kinds and still refuses the other four.
//
// A member is listed only where both forms are genuinely admitted by this
// fixture: stop.checkpoint_id and materialize.preserved_checkpoint_id are
// nullable in the schema but pinned by a cross-member rule in the object being
// swept, so their null form is refused there and the sweep keeps checking it.
var nullableMembers = map[string]string{
	"attach.provider_exit_code":                    "number",
	"logs.next_cursor":                             "string",
	"session_summary.newest_checkpoint_id":         "string",
	"session_summary.newest_checkpoint_created_at": "string",
	"peer_summary.last_successful_sync_at":         "string",
}

// admits reports whether a replacement leaves the member in one of the JSON
// kinds the pinned document declares for it, which would make the case a
// duplicate of the positive test rather than a type refusal.
func admits(member string, original, replacement any) bool {
	if nonNull, nullable := nullableMembers[member]; nullable {
		return replacement == nil || jsonKind(replacement) == nonNull
	}
	return jsonKind(original) == jsonKind(replacement)
}

func jsonKind(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case json.Number:
		return "number"
	case string:
		return "string"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unsupported"
	}
}
