package cliresult

import (
	"testing"
)

// TestTopLevelIdentifierNullabilityMatchesTheReviewedTable pins the Section
// 14.2 nullability sentences as a reviewed literal table compared against
// production. Ranging over the production registry instead would let a mutant
// that deletes a row delete its own test case and survive.
//
// The three unconstrained entries are as much part of the claim as the
// constrained ones: attach and pane appear in neither of the two sets Section
// 14.2 names, and sync, logs, cancel, and takeover appear in neither
// session_id set, so this package must state no rule for them.
func TestTopLevelIdentifierNullabilityMatchesTheReviewedTable(t *testing.T) {
	type rule struct{ operation, session idRule }
	reviewed := map[Command]rule{
		CommandCancel:            {idForbidden, idUnconstrained},
		CommandStart:             {idRequired, idRequired},
		CommandList:              {idForbidden, idForbidden},
		CommandStatus:            {idForbidden, idRequired},
		CommandAttach:            {idUnconstrained, idRequired},
		CommandTakeover:          {idRequired, idUnconstrained},
		CommandFork:              {idRequired, idRequired},
		CommandStop:              {idRequired, idRequired},
		CommandResume:            {idRequired, idRequired},
		CommandSync:              {idRequired, idUnconstrained},
		CommandDiff:              {idForbidden, idRequired},
		CommandMaterialize:       {idRequired, idRequired},
		CommandDoctor:            {idForbidden, idForbidden},
		CommandLogs:              {idForbidden, idUnconstrained},
		CommandPeerList:          {idForbidden, idForbidden},
		CommandPeerProbe:         {idForbidden, idForbidden},
		CommandSessionSetProfile: {idRequired, idRequired},
		CommandPane:              {idUnconstrained, idRequired},
	}
	implemented := ImplementedCommands()
	if len(reviewed) != len(implemented) {
		t.Fatalf("reviewed table has %d rows, %d commands are implemented", len(reviewed), len(implemented))
	}
	for command, want := range reviewed {
		entry := commandRegistry[command]
		if entry.operation != want.operation || entry.session != want.session {
			t.Fatalf("command %q rules = (%d,%d), reviewed table says (%d,%d)",
				command, entry.operation, entry.session, want.operation, want.session)
		}
	}
}

// TestIdentifierRulesAreEnforcedInBothDirections drives every implemented
// command through its own rule, flipping exactly one identifier. A required
// identifier is refused when null and a forbidden one is refused when present,
// so a gate narrowed to check only one direction fails here.
func TestIdentifierRulesAreEnforcedInBothDirections(t *testing.T) {
	for _, command := range ImplementedCommands() {
		t.Run(string(command), func(t *testing.T) {
			entry := commandRegistry[command]
			base := validSpec(t, command)
			operation := mustUUIDv7(t, fixtureOperationID)
			session := mustUUIDv7(t, fixtureSessionID)

			switch entry.operation {
			case idRequired:
				spec := base
				spec.IDs = strippedOperation(base.IDs)
				refuse(t, "a null operation_id on "+string(command), spec)
			case idForbidden:
				spec := base
				spec.IDs = base.IDs.WithOperation(operation)
				refuse(t, "a non-null operation_id on "+string(command), spec)
			case idUnconstrained:
				spec := base
				spec.IDs = base.IDs.WithOperation(operation)
				admit(t, "an operation_id on "+string(command), spec)
				spec = base
				spec.IDs = strippedOperation(base.IDs)
				admit(t, "no operation_id on "+string(command), spec)
			}

			switch entry.session {
			case idRequired:
				spec := base
				spec.IDs = strippedSession(base.IDs)
				refuse(t, "a null session_id on "+string(command), spec)
			case idForbidden:
				spec := base
				spec.IDs = base.IDs.WithSession(session)
				refuse(t, "a non-null session_id on "+string(command), spec)
			}
		})
	}
}

func strippedOperation(ids IDs) IDs {
	stripped := NoIDs()
	if session, known := ids.Session(); known {
		stripped = stripped.WithSession(session)
	}
	return stripped
}

func strippedSession(ids IDs) IDs {
	stripped := NoIDs()
	if operation, known := ids.Operation(); known {
		stripped = stripped.WithOperation(operation)
	}
	return stripped
}

// TestSessionScopeEqualityIsEnforcedAgainstEveryNestedSubject narrows the
// Section 14.2 sentence "a session-scoped command requires non-null session_id
// equal to every nested Session Summary", including the extension of that rule
// to a body member literally named session_id.
func TestSessionScopeEqualityIsEnforcedAgainstEveryNestedSubject(t *testing.T) {
	nested := []Command{
		CommandStart, CommandStatus, CommandAttach, CommandFork, CommandStop, CommandResume,
	}
	for _, command := range nested {
		t.Run(string(command)+"/nested summary", func(t *testing.T) {
			base := validSpec(t, command)
			body := cloneObject(base.Body.(map[string]any))
			body["session"] = sessionSummary(fixtureOtherHostID)
			spec := base
			spec.Body = body
			refuse(t, "a nested summary for another session", spec)
		})
	}
	bare := []Command{CommandDiff, CommandMaterialize, CommandSessionSetProfile, CommandPane}
	for _, command := range bare {
		t.Run(string(command)+"/bare member", func(t *testing.T) {
			base := validSpec(t, command)
			body := mutateBody(base.Body.(map[string]any), "session_id", fixtureOtherHostID)
			spec := base
			spec.Body = body
			refuse(t, "a body session_id for another session", spec)
		})
	}
}

// TestForkComparesTheNewSessionRatherThanTheSource proves the equality rule
// reads the nested Session Summary and not any other identifier in the body. A
// fork's source_session_id is deliberately a different session, and a validator
// that compared it would refuse every conforming fork.
func TestForkComparesTheNewSessionRatherThanTheSource(t *testing.T) {
	spec := validSpec(t, CommandFork)
	body := spec.Body.(map[string]any)
	if body["source_session_id"] == fixtureSessionID {
		t.Fatalf("fixture does not distinguish the source from the new session")
	}
	admit(t, "a fork whose source differs from its new session", spec)
}
