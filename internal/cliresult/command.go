package cliresult

import (
	"errors"
	"fmt"
	"sort"
)

// Command is one CLI Result command tag. Section 14.2 says "the command tag
// selects exactly one body", so the tag is the discriminant of the tagged union
// and never a free-form label.
type Command string

// The Section 14.2 command tags, in the order the pinned body table lists them.
const (
	CommandCancel            Command = "cancel"
	CommandStart             Command = "start"
	CommandList              Command = "list"
	CommandStatus            Command = "status"
	CommandAttach            Command = "attach"
	CommandTakeover          Command = "takeover"
	CommandFork              Command = "fork"
	CommandStop              Command = "stop"
	CommandResume            Command = "resume"
	CommandSync              Command = "sync"
	CommandDiff              Command = "diff"
	CommandMaterialize       Command = "materialize"
	CommandDoctor            Command = "doctor"
	CommandLogs              Command = "logs"
	CommandPeerList          Command = "peer.list"
	CommandPeerProbe         Command = "peer.probe"
	CommandSessionSetProfile Command = "session.set_profile"
	CommandPane              Command = "pane"
)

// The Section 14.1 session.clone.* tags, which Section 14.2 binds to CLI Result
// 2.0.0.
const (
	CommandCloneAdapters Command = "session.clone.adapters"
	CommandCloneDoctor   Command = "session.clone.doctor"
	CommandCloneList     Command = "session.clone.list"
	CommandCloneInspect  Command = "session.clone.inspect"
	CommandClonePlan     Command = "session.clone.plan"
	CommandCloneRun      Command = "session.clone.run"
	CommandCloneVerify   Command = "session.clone.verify"
	CommandCloneOpen     Command = "session.clone.open"
)

// The Section 14.5 Session Directory tags, which bind CLI Result 3.0.0.
const (
	CommandSessionsList      Command = "sessions.list"
	CommandSessionsGrep      Command = "sessions.grep"
	CommandSessionsInspect   Command = "sessions.inspect"
	CommandSessionsLineage   Command = "sessions.lineage"
	CommandSessionsScan      Command = "sessions.scan"
	CommandSessionsEnrich    Command = "sessions.enrich"
	CommandSessionsJobs      Command = "sessions.jobs"
	CommandSessionsPlan      Command = "sessions.plan"
	CommandSessionsContinue  Command = "sessions.continue"
	CommandSessionsOperation Command = "sessions.operation"
	CommandSessionsAttach    Command = "sessions.attach"
	CommandSessionsDoctor    Command = "sessions.doctor"
	CommandSessionsQuery     Command = "sessions.query"
	CommandSessionsMutate    Command = "sessions.mutate"
)

// The Section 14.1 TerminalBackend inspection tags, which bind CLI Result
// 4.0.0.
const (
	CommandTerminalBackendList   Command = "terminal.backend.list"
	CommandTerminalBackendShow   Command = "terminal.backend.show"
	CommandTerminalBackendProbe  Command = "terminal.backend.probe"
	CommandTerminalBackendDoctor Command = "terminal.backend.doctor"
)

// ErrUnknownCommand reports a tag the pinned document registers for no CLI
// Result version. It is distinct from ErrUnimplementedVersion: a registered tag
// this repository cannot build is not the same fact as a tag that does not
// exist.
var ErrUnknownCommand = errors.New("unregistered cli result command tag")

// idRule is what Section 14.2 fixes about one top-level identifier.
type idRule int

const (
	// idUnconstrained means the pinned document states no rule for this tag.
	// It is a deliberate absence, not a default: inventing a nullability rule
	// where the specification is silent would make this validator refuse
	// conforming documents.
	idUnconstrained idRule = iota
	// idRequired means the identifier must be non-null.
	idRequired
	// idForbidden means the identifier must be JSON null.
	idForbidden
)

type commandEntry struct {
	version Version
	// operation and session are the Section 14.2 nullability rules.
	operation idRule
	session   idRule
	// validate is the closed body validator, or nil when this repository
	// registers the tag without building its body.
	validate func(body map[string]any) error
	// sessionScopeMember names the body member that must equal a non-null
	// top-level session_id, or "" when the body nests a Session Summary
	// instead, or has neither.
	sessionScopeMember string
	// nestsSessionSummary marks the bodies that carry a session member typed
	// SessionSummary.
	nestsSessionSummary bool
}

// commandRegistry is the complete reviewed tag table.
//
// The nullability columns are Section 14.2 verbatim: "operation_id is non-null
// for start, takeover, fork, stop, resume, sync, materialize, and profile
// mutation, and null for pure reads. A session-scoped command requires non-null
// session_id equal to every nested Session Summary; list, doctor, and peer
// commands use null."
//
// Where the pinned text names neither set, the rule is idUnconstrained. attach
// and pane are the two such tags: neither appears in the enumerated non-null
// operation list, and neither is a pure read - attach reports detachment and a
// provider exit status, and pane can report resumed - so this validator states
// no rule for them rather than picking one. cancel is constrained to null
// because Section 14.1 settles it outright: the umbrella chooser "MUST NOT
// mutate while presenting the choice", so a canceled chooser is a pure read.
var commandRegistry = map[Command]commandEntry{
	CommandCancel: {
		version: Version100, operation: idForbidden, session: idUnconstrained,
		validate: validateCancelBody,
	},
	CommandStart: {
		version: Version100, operation: idRequired, session: idRequired,
		validate: validateStartBody, nestsSessionSummary: true,
	},
	CommandList: {
		version: Version100, operation: idForbidden, session: idForbidden,
		validate: validateListBody,
	},
	CommandStatus: {
		version: Version100, operation: idForbidden, session: idRequired,
		validate: validateStatusBody, nestsSessionSummary: true,
	},
	CommandAttach: {
		version: Version100, operation: idUnconstrained, session: idRequired,
		validate: validateAttachBody, nestsSessionSummary: true,
	},
	CommandTakeover: {
		version: Version100, operation: idRequired, session: idUnconstrained,
		validate: validateTakeoverBody,
	},
	CommandFork: {
		version: Version100, operation: idRequired, session: idRequired,
		validate: validateForkBody, nestsSessionSummary: true,
	},
	CommandStop: {
		version: Version100, operation: idRequired, session: idRequired,
		validate: validateStopBody, nestsSessionSummary: true,
	},
	CommandResume: {
		version: Version100, operation: idRequired, session: idRequired,
		validate: validateResumeBody, nestsSessionSummary: true,
	},
	CommandSync: {
		version: Version100, operation: idRequired, session: idUnconstrained,
		validate: validateSyncBody,
	},
	CommandDiff: {
		version: Version100, operation: idForbidden, session: idRequired,
		validate: validateDiffBody, sessionScopeMember: "session_id",
	},
	CommandMaterialize: {
		version: Version100, operation: idRequired, session: idRequired,
		validate: validateMaterializeBody, sessionScopeMember: "session_id",
	},
	CommandDoctor: {
		version: Version100, operation: idForbidden, session: idForbidden,
		validate: validateDoctorBody,
	},
	CommandLogs: {
		version: Version100, operation: idForbidden, session: idUnconstrained,
		validate: validateLogsBody,
	},
	CommandPeerList: {
		version: Version100, operation: idForbidden, session: idForbidden,
		validate: validatePeerListBody,
	},
	CommandPeerProbe: {
		version: Version100, operation: idForbidden, session: idForbidden,
		validate: validatePeerProbeBody,
	},
	CommandSessionSetProfile: {
		version: Version100, operation: idRequired, session: idRequired,
		validate: validateSetProfileBody, sessionScopeMember: "session_id",
	},
	CommandPane: {
		version: Version100, operation: idUnconstrained, session: idRequired,
		validate: validatePaneBody, sessionScopeMember: "session_id",
	},

	// CLI Result 2.0.0. Section 14.2 fixes the version selection for every
	// session.clone.* command, so these tags are registered here; their closed
	// bodies are declared by Section 14.1 over the Section 13.14 clone types,
	// which this slice does not implement, so validate stays nil and New
	// refuses rather than admitting an unchecked body.
	CommandCloneAdapters: {version: Version200, operation: idForbidden, session: idForbidden},
	CommandCloneDoctor:   {version: Version200, operation: idForbidden, session: idForbidden},
	CommandCloneList:     {version: Version200, operation: idForbidden, session: idForbidden},
	CommandCloneInspect:  {version: Version200, operation: idForbidden, session: idForbidden},
	CommandClonePlan:     {version: Version200, operation: idRequired, session: idForbidden},
	CommandCloneRun:      {version: Version200, operation: idRequired, session: idUnconstrained},
	CommandCloneVerify:   {version: Version200, operation: idForbidden, session: idForbidden},
	CommandCloneOpen:     {version: Version200, operation: idRequired, session: idRequired},

	// CLI Result 3.0.0, Section 14.5. Registered so that a Directory tag is
	// reported as an unimplemented version rather than as a tag that does not
	// exist; the two are different facts.
	CommandSessionsList:      {version: Version300},
	CommandSessionsGrep:      {version: Version300},
	CommandSessionsInspect:   {version: Version300},
	CommandSessionsLineage:   {version: Version300},
	CommandSessionsScan:      {version: Version300},
	CommandSessionsEnrich:    {version: Version300},
	CommandSessionsJobs:      {version: Version300},
	CommandSessionsPlan:      {version: Version300},
	CommandSessionsContinue:  {version: Version300},
	CommandSessionsOperation: {version: Version300},
	CommandSessionsAttach:    {version: Version300},
	CommandSessionsDoctor:    {version: Version300},
	CommandSessionsQuery:     {version: Version300},
	CommandSessionsMutate:    {version: Version300},

	// CLI Result 4.0.0, Section 14.6.
	CommandTerminalBackendList:   {version: Version400},
	CommandTerminalBackendShow:   {version: Version400},
	CommandTerminalBackendProbe:  {version: Version400},
	CommandTerminalBackendDoctor: {version: Version400},
}

// VersionForCommand returns the exact CLI Result version a command tag selects.
// There is no negotiation path and no default: Section 14.2 states that "legacy
// commands select CLI Result 1.0.0 and Structured Error 1.0.0; every
// session.clone.* command selects CLI Result 2.0.0 on success", and that "no
// command may emit another registered version or retry a different major after
// parsing begins".
//
// A tag whose version this repository does not build is refused with
// ErrUnimplementedVersion, so a caller can distinguish "AX registers no such
// command" from "this build cannot produce that result".
func VersionForCommand(command Command) (Version, error) {
	entry, known := commandRegistry[command]
	if !known {
		return "", fmt.Errorf("%w: %q", ErrUnknownCommand, command)
	}
	if err := requireImplementedVersion(entry.version); err != nil {
		return "", fmt.Errorf("%w (command %q)", err, command)
	}
	if entry.validate == nil {
		return "", fmt.Errorf(
			"%w: %q selects CLI Result %s, whose body this repository does not build",
			ErrUnimplementedVersion, command, entry.version)
	}
	return entry.version, nil
}

// RegisteredVersionForCommand returns the version a tag selects without
// requiring this repository to build it. It exists so that a caller can report
// the pinned selection for an unimplemented surface truthfully instead of
// guessing or reporting nothing.
func RegisteredVersionForCommand(command Command) (Version, error) {
	entry, known := commandRegistry[command]
	if !known {
		return "", fmt.Errorf("%w: %q", ErrUnknownCommand, command)
	}
	return entry.version, nil
}

// Commands returns every registered command tag in sorted order.
func Commands() []Command {
	result := make([]Command, 0, len(commandRegistry))
	for command := range commandRegistry {
		result = append(result, command)
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}

// ImplementedCommands returns the tags whose closed body this repository builds,
// in sorted order. It is the measured numerator of this package's command
// coverage against Commands.
func ImplementedCommands() []Command {
	var result []Command
	for _, command := range Commands() {
		if _, err := VersionForCommand(command); err == nil {
			result = append(result, command)
		}
	}
	return result
}

// validateIdentifiers enforces the Section 14.2 nullability rules for the two
// top-level identifiers.
func validateIdentifiers(command Command, ids IDs) error {
	entry, known := commandRegistry[command]
	if !known {
		return fmt.Errorf("%w: %q", ErrUnknownCommand, command)
	}
	_, operationKnown := ids.Operation()
	if err := checkIDRule("operation_id", command, entry.operation, operationKnown); err != nil {
		return err
	}
	_, sessionKnown := ids.Session()
	return checkIDRule("session_id", command, entry.session, sessionKnown)
}

func checkIDRule(name string, command Command, rule idRule, present bool) error {
	switch rule {
	case idRequired:
		if !present {
			return failf("%s is null, which command %q forbids", name, command)
		}
	case idForbidden:
		if present {
			return failf("%s is non-null, which command %q forbids", name, command)
		}
	}
	return nil
}

// validateNestedSessionScope enforces the Section 14.2 rule that "a
// session-scoped command requires non-null session_id equal to every nested
// Session Summary".
//
// The equality is additionally applied to a body member literally named
// session_id. The pinned sentence names nested Session Summaries, and the
// bodies for diff, materialize, session.set_profile, and pane carry the session
// identity as a bare member instead. Admitting an envelope whose declared scope
// contradicts its own body would make the top-level identifier meaningless for
// exactly those four tags, so the rule is extended here and stated rather than
// smuggled.
func validateNestedSessionScope(command Command, ids IDs, body map[string]any) error {
	entry, known := commandRegistry[command]
	if !known {
		return fmt.Errorf("%w: %q", ErrUnknownCommand, command)
	}
	session, present := ids.Session()
	if !present {
		return nil
	}
	if entry.nestsSessionSummary {
		summary, err := memberObject(body, "body", "session")
		if err != nil {
			return err
		}
		nested, err := memberUUIDv7(summary, "body.session", "session_id")
		if err != nil {
			return err
		}
		if nested.String() != session.String() {
			return failf(
				"body.session.session_id %s differs from the top-level session_id %s",
				nested.String(), session.String())
		}
	}
	if entry.sessionScopeMember != "" {
		nested, err := memberUUIDv7(body, "body", entry.sessionScopeMember)
		if err != nil {
			return err
		}
		if nested.String() != session.String() {
			return failf(
				"body.%s %s differs from the top-level session_id %s",
				entry.sessionScopeMember, nested.String(), session.String())
		}
	}
	return nil
}

// validateBody dispatches to the closed validator the command tag selects.
func validateBody(command Command, body map[string]any) error {
	entry, known := commandRegistry[command]
	if !known {
		return fmt.Errorf("%w: %q", ErrUnknownCommand, command)
	}
	if entry.validate == nil {
		return fmt.Errorf(
			"%w: %q selects CLI Result %s, whose body this repository does not build",
			ErrUnimplementedVersion, command, entry.version)
	}
	return entry.validate(body)
}
