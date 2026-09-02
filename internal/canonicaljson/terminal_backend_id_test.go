package canonicaljson

import (
	"strings"
	"testing"
)

// terminalBackendIDEventTypes are the two Session Event 4.0.0 payloads whose
// pinned member table declares terminal_backend_id. Both reach the grammar
// through validateTerminalV4Payload, which validateSessionEvent selects from
// sessionEventPayloadShapes for schema_version 4.0.0.
var terminalBackendIDEventTypes = []string{"terminal.created", "session.resumed"}

func sessionEventWithTerminalBackendID(eventType string, value any) map[string]any {
	object := validSessionEventObject("4.0.0", eventType)
	object["payload"].(map[string]any)["terminal_backend_id"] = value
	return object
}

// TestTerminalBackendIDGrammarRefusedAtBothIdentityEntries attacks the Section
// 4.B terminal-backend-id grammar at the real identity entries. Every case
// widens exactly one part of the declared character class or separator rule, so
// admitting any of them means the production pattern has been widened rather
// than merely deleted.
func TestTerminalBackendIDGrammarRefusedAtBothIdentityEntries(t *testing.T) {
	grammar := "must match the terminal-backend-id grammar"
	bound := "must contain 1..128 ASCII bytes"

	cases := []struct {
		name   string
		value  string
		reason string
	}{
		// The declared minimum is subsumed by requireString, which refuses the
		// empty string before requireTerminalBackendID measures the bound; the
		// subsuming refusal is pinned here rather than asserted as the bound.
		{"empty string", "", "must be a non-empty UTF-8 string"},
		// The pinned bound is 128; 129 is the smallest refused length.
		{"one byte past the declared bound", strings.Repeat("a", 129), bound},
		{"far past the declared bound", strings.Repeat("a", 5000), bound},
		{"uppercase widens the lowercase class", "AX.TMUX", grammar},
		{"mixed case widens the lowercase class", "Ax.Tmux", grammar},
		{"leading digit widens the first character class", "1ax", grammar},
		{"underscore widens the separator class", "ax_tmux", grammar},
		{"space widens the label class", "ax tmux", grammar},
		{"slash widens the label class", "../../etc/passwd", grammar},
		{"colon widens the label class", "urn:ax:schema:blob", grammar},
		{"at sign widens the label class", "ax@tmux", grammar},
		{"non-ASCII widens the label class", "ax.tmux界", grammar},
		{"leading separator drops the required first label", "-leading-dash", grammar},
		{"leading dot drops the required first label", ".ax", grammar},
		{"trailing separator makes the trailing label optional", "ax.", grammar},
		{"trailing dash makes the trailing label optional", "ax-", grammar},
		{"consecutive separators make the label optional", "ax..tmux", grammar},
		{"consecutive mixed separators make the label optional", "ax.-tmux", grammar},
		// RE2 anchors $ at end of text, so a trailing newline cannot smuggle a
		// second line past an otherwise matching first line.
		{"trailing newline", "ax.tmux\n", grammar},
		{"embedded newline", "ax\n.tmux", grammar},
		{"embedded NUL", "ax\x00tmux", grammar},
	}

	for _, eventType := range terminalBackendIDEventTypes {
		for _, testCase := range cases {
			t.Run(eventType+"/"+testCase.name, func(t *testing.T) {
				object := sessionEventWithTerminalBackendID(eventType, testCase.value)
				assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfEventID, testCase.reason)
			})
		}
	}
}

// TestTerminalBackendIDNonStringRefusedAtBothIdentityEntries keeps the declared
// scalar from being satisfied by a non-string JSON value.
func TestTerminalBackendIDNonStringRefusedAtBothIdentityEntries(t *testing.T) {
	for _, eventType := range terminalBackendIDEventTypes {
		for _, value := range []any{nil, true, []any{"ax.tmux"}, map[string]any{"id": "ax.tmux"}} {
			object := sessionEventWithTerminalBackendID(eventType, value)
			assertIdentityEntriesRefuseShape(t, mustJSON(t, object), SelfEventID)
		}
	}
}

// TestTerminalBackendIDAcceptsDeclaredGrammarAtBothIdentityEntries proves the
// grammar is not merely refusing everything: the canonical built-ins and the
// other forms the declared pattern admits must still attest.
func TestTerminalBackendIDAcceptsDeclaredGrammarAtBothIdentityEntries(t *testing.T) {
	accepted := []string{
		"ax.tmux",
		"ax.conpty",
		"a",
		"tmux",
		"ax-tmux",
		"ax.tmux2",
		"vendor.backend-1",
		"a0.b1-c2.d3",
		// The reserved ax. namespace is a registry rule in Section 4.B, not a
		// payload constraint, so a vendor namespace is equally admissible.
		"example.terminal-backend",
	}
	for _, eventType := range terminalBackendIDEventTypes {
		for _, value := range accepted {
			object := sessionEventWithTerminalBackendID(eventType, value)
			assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfEventID)
		}
	}
}

// TestTerminalBackendIDBoundIsProvenInBothDirections asserts the bound against
// the number the pinned specification declares, not against the production
// constant it derives from. Section 4.B: "terminal_backend_id is 1-128 ASCII
// bytes". The declared grammar admits only ASCII, so the byte count and the
// Unicode character count coincide for every admitted value.
func TestTerminalBackendIDBoundIsProvenInBothDirections(t *testing.T) {
	const declaredMaximum = 128
	const declaredMinimum = 1

	atMinimum := strings.Repeat("a", declaredMinimum)
	atMaximum := "a" + strings.Repeat("0", declaredMaximum-1)
	if len(atMaximum) != declaredMaximum || len(atMinimum) != declaredMinimum {
		t.Fatalf("bound fixtures are %d and %d bytes, want %d and %d", len(atMinimum), len(atMaximum), declaredMinimum, declaredMaximum)
	}
	pastMaximum := atMaximum + "0"
	belowMinimum := ""

	for _, eventType := range terminalBackendIDEventTypes {
		assertIdentityEntriesAcceptShape(t, mustJSON(t, sessionEventWithTerminalBackendID(eventType, atMinimum)), SelfEventID)
		assertIdentityEntriesAcceptShape(t, mustJSON(t, sessionEventWithTerminalBackendID(eventType, atMaximum)), SelfEventID)
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, sessionEventWithTerminalBackendID(eventType, pastMaximum)), SelfEventID, "1..128 ASCII bytes")
		// Below the minimum the subsuming non-empty string check refuses first.
		assertIdentityEntriesRefuseWithReason(t, mustJSON(t, sessionEventWithTerminalBackendID(eventType, belowMinimum)), SelfEventID, "must be a non-empty UTF-8 string")
	}
}
