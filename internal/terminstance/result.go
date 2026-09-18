package terminstance

import (
	"sort"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Result is the closed §4.C MutationResult object in harness form: it
// repeats the request context identities and adds before/after state,
// side effects, evidence IDs and exactly one retry disposition. The
// engine populates it on every path, success or failure, so the retry
// disposition and the local observation are always readable; failures
// additionally carry the *Error.
type Result struct {
	OperationID           string
	SessionID             string
	TerminalInstanceID    string
	TerminalBackendID     string
	ImplementationVersion string
	ProtocolVersion       string
	BackendGeneration     string
	Before                terminalbackend.InstanceState
	After                 terminalbackend.InstanceState
	Effects               []terminalbackend.SideEffect
	EvidenceIDs           []string
	Disposition           RetryDisposition
}

// CheckResult enforces the §4.C result rule: every repeated identity
// equals the request context, before equals the source state the engine
// executed from, states and effects parse through the landed vocabularies,
// side effects are sorted-unique TerminalBackendSideEffect[0..10],
// evidence IDs are sorted-unique digest[0..256], and the disposition
// parses. Each repeated identity is its own arm with its own detail so a
// mutant dropping one comparison is killed by that member's tamper test.
func CheckResult(context MutationContext, source terminalbackend.InstanceState, result Result) error {
	if result.OperationID != context.OperationID {
		return refuse(CodeProtocolError, "result operation binding")
	}
	if result.SessionID != context.SessionID {
		return refuse(CodeProtocolError, "result session binding")
	}
	if result.TerminalInstanceID != context.TerminalInstanceID {
		return refuse(CodeProtocolError, "result instance binding")
	}
	if result.TerminalBackendID != context.TerminalBackendID {
		return refuse(CodeProtocolError, "result backend binding")
	}
	if result.ImplementationVersion != context.ImplementationVersion {
		return refuse(CodeProtocolError, "result implementation binding")
	}
	if result.ProtocolVersion != context.ProtocolVersion {
		return refuse(CodeProtocolError, "result protocol binding")
	}
	if result.BackendGeneration != context.BackendGeneration {
		return refuse(CodeStaleGeneration, "result generation binding")
	}
	if _, err := terminalbackend.ParseInstanceState(string(result.Before)); err != nil {
		return wrapLanded(err)
	}
	if _, err := terminalbackend.ParseInstanceState(string(result.After)); err != nil {
		return wrapLanded(err)
	}
	if result.Before != source {
		return refuse(CodeProtocolError, "result before state")
	}
	if err := checkSortedUniqueEffects(result.Effects); err != nil {
		return err
	}
	if err := checkSortedUniqueEvidence(result.EvidenceIDs); err != nil {
		return err
	}
	if _, err := ParseRetryDisposition(string(result.Disposition)); err != nil {
		return err
	}
	return nil
}

// checkSortedUniqueEffects enforces sorted-unique [0..10] over parsed
// side effects. Order and uniqueness are two obligations with two arms:
// an unsorted-but-unique vector and a sorted-but-duplicated vector are
// refused by different details, so a mutant dropping either half is
// killed by its own vector.
func checkSortedUniqueEffects(effects []terminalbackend.SideEffect) error {
	if len(effects) > 10 {
		return refuse(CodeProtocolError, "result effects bound")
	}
	for _, effect := range effects {
		if _, err := terminalbackend.ParseSideEffect(string(effect)); err != nil {
			return wrapLanded(err)
		}
	}
	for index := 1; index < len(effects); index++ {
		if effects[index-1] == effects[index] {
			return refuse(CodeProtocolError, "result effects unique")
		}
	}
	for index := 1; index < len(effects); index++ {
		if effects[index-1] > effects[index] {
			return refuse(CodeProtocolError, "result effects order")
		}
	}
	return nil
}

// checkSortedUniqueEvidence enforces sorted-unique digest[0..256] on a
// mutation result. Order and uniqueness are separate arms for the same
// reason as the effects twin: each half owns its mutant.
func checkSortedUniqueEvidence(evidence []string) error {
	return checkSortedUniqueEvidenceWhere(evidence, "result")
}

// checkSortedUniqueStatusEvidence enforces the same sorted-unique
// digest[0..256] rule on a status report. The rule is shared; only the
// static detail site differs, so a status mutant and a result mutant
// cannot kill each other's arm.
func checkSortedUniqueStatusEvidence(evidence []string) error {
	return checkSortedUniqueEvidenceWhere(evidence, "status")
}

func checkSortedUniqueEvidenceWhere(evidence []string, where string) error {
	if len(evidence) > 256 {
		return refuse(CodeProtocolError, where+" evidence bound")
	}
	for _, id := range evidence {
		if _, err := scalar.ParseDigest(id); err != nil {
			return refuse(CodeProtocolError, where+" evidence digest")
		}
	}
	for index := 1; index < len(evidence); index++ {
		if evidence[index-1] == evidence[index] {
			return refuse(CodeProtocolError, where+" evidence unique")
		}
	}
	for index := 1; index < len(evidence); index++ {
		if evidence[index-1] > evidence[index] {
			return refuse(CodeProtocolError, where+" evidence order")
		}
	}
	return nil
}

// buildResult assembles the engine result on every path. Effects and
// evidence are reported sorted: execution order is the transition order
// (request-stop commits graceful_stop_requested before process_closed
// before backend_store_closed), while the wire order is bytewise sorted,
// so the engine sorts a copy and never reorders execution.
func buildResult(context MutationContext, before, after terminalbackend.InstanceState, effects []terminalbackend.SideEffect, evidence []string, disposition RetryDisposition) Result {
	ordered := append([]terminalbackend.SideEffect(nil), effects...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	orderedEvidence := append([]string(nil), evidence...)
	sort.Strings(orderedEvidence)
	return Result{
		OperationID:           context.OperationID,
		SessionID:             context.SessionID,
		TerminalInstanceID:    context.TerminalInstanceID,
		TerminalBackendID:     context.TerminalBackendID,
		ImplementationVersion: context.ImplementationVersion,
		ProtocolVersion:       context.ProtocolVersion,
		BackendGeneration:     context.BackendGeneration,
		Before:                before,
		After:                 after,
		Effects:               ordered,
		EvidenceIDs:           orderedEvidence,
		Disposition:           disposition,
	}
}
