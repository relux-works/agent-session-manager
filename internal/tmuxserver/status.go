package tmuxserver

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// ObserveStatus reports the tmux observation for one status body: the
// recorded identity tuple from the durable binding, the live probe
// (list-panes for presence and provider rows, list-sessions for
// attached counts) reconciled against the AX-side state memory, and
// the match verdict comparing the recorded tuple against the query.
// The reported identity is observed, never echoed: every member comes
// from the recorded binding document, so a query under another
// session, instance, backend, or version reads the non-match result
// instead of manufacturing agreement. An error is unknown, never
// absent: transport failures, malformed probe output, contradictory
// probes, and binding read failures all refuse without a report.
func (runner *backend) ObserveStatus(ctx context.Context, body terminstance.StatusBody) (terminstance.StatusObservation, error) {
	empty := terminstance.StatusObservation{}
	recorded, bound, err := runner.statusRecordedBinding(body.SessionID)
	if err != nil {
		return empty, err
	}
	if !bound {
		// A session-scoped query over an unbound session reads
		// absent (the engine adopts the session it asked about).
		// An exact-instance query needs all six recorded members
		// to compare, and an unbound session records none: the
		// identity is unobservable, so the observation is
		// unknown, never a manufactured match.
		if body.HasTerminalInstanceID {
			return empty, &terminstance.Error{Code: terminalbackend.CodeUnavailable, Detail: "status observation unknown"}
		}
		return runner.absentObservation(body), nil
	}
	binding, err := runner.statusBindingDoc(body.SessionID, recorded)
	if err != nil {
		return empty, err
	}
	if body.HasTerminalInstanceID && !bindingMatchesQuery(binding, body) {
		return runner.nonMatchObservation(body, binding), nil
	}
	// The live probes run under the query deadline, through the
	// same shared probe context as the close-confirm poll: a probe
	// still blocked when the bound fires concludes unknown — the
	// engine maps the uncoded failure — never absent. An
	// unparseable deadline leaves the caller's context unmodified;
	// the engine refuses it before any probe runs.
	probeCtx := ctx
	if deadline, err := body.Deadline.Time(); err == nil {
		var cancel context.CancelFunc
		probeCtx, cancel = probeContext(ctx, runner.now, deadline)
		defer cancel()
	}
	instance := recorded.TerminalInstanceID
	present, panes, err := runner.probePanes(probeCtx, instance)
	if err != nil {
		return empty, err
	}
	attached := 0
	if present {
		count, err := runner.probeAttached(probeCtx, instance)
		if err != nil {
			return empty, err
		}
		attached = count
	}
	memory, found, err := runner.lookupMemory(instance)
	if err != nil {
		return empty, err
	}
	state, wrapper, provider, attachable := ClassifyStatus(ClassifyInput{
		Present:         present,
		Attached:        attached,
		ProviderRows:    len(panes),
		Memory:          memory,
		MemoryFound:     found,
		AttachCapable:   runner.attachCapable(),
		ProviderWanted:  body.IncludeProviderObservation,
		ProviderCapable: runner.admitted.Has("provider_process_observation"),
	})
	providerPresent, requested, evidenced := providerTriple(body.IncludeProviderObservation, runner.admitted.Has("provider_process_observation"), provider)
	evidence := statusEvidence(instance, present, panes, attached)
	return terminstance.StatusObservation{
		State:             state,
		IdentityMatch:     true,
		WrapperPresent:    wrapper,
		ProviderPresent:   providerPresent,
		Attachable:        attachable,
		EvidenceIDs:       evidence,
		ProviderRequested: requested,
		ProviderEvidenced: evidenced,
		AttachEvidenced:   attachable,
		SessionID:         binding.SessionID,
		InstanceID:        binding.TerminalInstanceID,
		BackendID:         binding.TerminalBackendID,
		ImplVersion:       binding.ImplementationVersion,
		ProtoVersion:      binding.ProtocolVersion,
		Generation:        binding.BackendGeneration,
	}, nil
}

// statusRecordedBinding reads the session's recorded bootstrap binding
// for identity observation. An unbound session reports found false: it
// reads absent on the session-scoped path and unknown on the exact
// path, never refused. A binding read failure is unknown, never
// absence.
func (runner *backend) statusRecordedBinding(sessionID string) (axpane.Binding, bool, error) {
	empty := axpane.Binding{}
	if runner.bindings == nil {
		return empty, false, &terminstance.Error{Code: terminalbackend.CodeProtocolError, Detail: "backend binding store"}
	}
	binding, found, err := runner.bindings.Status(sessionID)
	if err != nil {
		return empty, false, err
	}
	if !found {
		return empty, false, nil
	}
	return binding, true, nil
}

// statusBindingDoc reads and admits the recorded identity tuple for a
// bound session: the create-time binding document under the recorded
// (session, bootstrap) pair, re-admitted by the termbind owner. A
// missing document disagrees with the receipt that proves it, a
// document that fails admission is corrupt, and a document naming
// another session or instance than the receipt anchors is tampered:
// all three refuse the integrity failure, never a match verdict.
func (runner *backend) statusBindingDoc(sessionID string, recorded axpane.Binding) (termbind.Binding, error) {
	empty := termbind.Binding{}
	if runner.states == nil {
		return empty, &terminstance.Error{Code: terminalbackend.CodeProtocolError, Detail: "backend binding store"}
	}
	doc, found, err := runner.states.LookupBindingDoc(sessionID, recorded.OperationID)
	if err != nil {
		var local *Error
		if errors.As(err, &local) {
			return empty, &terminstance.Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "status binding image"}
		}
		return empty, err
	}
	if !found {
		return empty, &terminstance.Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "status binding image"}
	}
	binding, err := termbind.ParseTerminalBinding(doc)
	if err != nil {
		return empty, &terminstance.Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "status binding image"}
	}
	if binding.SessionID != sessionID ||
		binding.TerminalInstanceID != recorded.TerminalInstanceID {
		return empty, &terminstance.Error{Code: terminalbackend.CodeIntegrityFailure, Detail: "status binding image"}
	}
	return binding, nil
}

// bindingMatchesQuery compares the recorded identity tuple against an
// exact-instance query on the five non-session members: the session
// member already agreed when the receipt anchored this document, so
// only instance, backend, implementation, protocol, and generation
// decide the match. Every member must agree; any drift reads the
// non-match result.
func bindingMatchesQuery(binding termbind.Binding, body terminstance.StatusBody) bool {
	return binding.TerminalInstanceID == body.TerminalInstanceID &&
		binding.TerminalBackendID == body.TerminalBackendID &&
		binding.ImplementationVersion == body.ImplementationVersion &&
		binding.ProtocolVersion == body.ProtocolVersion &&
		binding.BackendGeneration == body.BackendGeneration
}

// nonMatchObservation reports the prescribed non-match result for an
// exact-instance query the recorded tuple disagrees with: absent with
// no wrapper, no provider, no attachability, and null last fields. It
// probes nothing: the non-match is a binding verdict, and a live tmux
// session under another identity never authorizes fallback. The
// reported tuple is the recorded one, so the engine's own comparison
// recomputes the same non-match.
func (runner *backend) nonMatchObservation(body terminstance.StatusBody, binding termbind.Binding) terminstance.StatusObservation {
	return terminstance.StatusObservation{
		State:             terminalbackend.StateAbsent,
		IdentityMatch:     false,
		WrapperPresent:    false,
		ProviderPresent:   nil,
		Attachable:        false,
		EvidenceIDs:       []string{scalar.SHA256Digest([]byte("identity-nonmatch\n" + binding.SessionID + "\n" + binding.TerminalInstanceID)).String()},
		ProviderRequested: body.IncludeProviderObservation,
		ProviderEvidenced: false,
		AttachEvidenced:   false,
		SessionID:         binding.SessionID,
		InstanceID:        binding.TerminalInstanceID,
		BackendID:         binding.TerminalBackendID,
		ImplVersion:       binding.ImplementationVersion,
		ProtoVersion:      binding.ProtocolVersion,
		Generation:        binding.BackendGeneration,
	}
}

// absentObservation reports proven absence for an unbound session: the
// absent shape with the queried session echoed, so the engine adopts a
// match on the session it asked about and nothing else.
func (runner *backend) absentObservation(body terminstance.StatusBody) terminstance.StatusObservation {
	return terminstance.StatusObservation{
		State:             terminalbackend.StateAbsent,
		IdentityMatch:     true,
		WrapperPresent:    false,
		ProviderPresent:   nil,
		Attachable:        false,
		EvidenceIDs:       []string{scalar.SHA256Digest([]byte("unbound\n" + body.SessionID)).String()},
		ProviderRequested: body.IncludeProviderObservation,
		ProviderEvidenced: false,
		AttachEvidenced:   false,
		SessionID:         body.SessionID,
		InstanceID:        body.TerminalInstanceID,
		BackendID:         body.TerminalBackendID,
		ImplVersion:       body.ImplementationVersion,
		ProtoVersion:      body.ProtocolVersion,
		Generation:        body.BackendGeneration,
	}
}

// probePanes runs list-panes for one instance: exit 0 with at least one
// row proves presence; a nonzero exit proves absence only with an
// absence marker on stderr (no session, no window, or no server
// socket under the verified path); any other nonzero exit —
// permission failures, refused connections, killed children — is
// unknown, never absent. A transport error is unknown. An exit-0
// with zero rows is contradictory — a tmux session always has a
// pane — and refuses as unknown rather than inventing presence.
func (runner *backend) probePanes(ctx context.Context, instance string) (bool, []string, error) {
	saved := runner.instanceID
	runner.instanceID = instance
	defer func() { runner.instanceID = saved }()
	argv, err := runner.command(DirectiveListPanes)
	if err != nil {
		return false, nil, err
	}
	result, err := runner.runner.Run(ctx, argv)
	if err != nil {
		return false, nil, err
	}
	present, absent := classifyAbsence(result, absentMarkerNoSession, absentMarkerNoWindow, absentMarkerNoSocket)
	if absent {
		return false, nil, nil
	}
	if !present {
		return false, nil, &terminstance.Error{Code: terminalbackend.CodeUnavailable, Detail: "status observation unknown"}
	}
	rows := splitRows(string(result.Stdout))
	if len(rows) == 0 {
		return false, nil, &terminstance.Error{Code: terminalbackend.CodeUnavailable, Detail: "status observation unknown"}
	}
	return true, rows, nil
}

// probeAttached reads the attached-client count for one present
// instance from list-sessions. A missing row contradicts the presence
// probe (the session vanished between the two commands, or the list is
// corrupt) and refuses as unknown; a malformed count refuses the same
// way. Neither invents attachability.
func (runner *backend) probeAttached(ctx context.Context, instance string) (int, error) {
	argv, err := runner.command(DirectiveListSessions)
	if err != nil {
		return 0, err
	}
	result, err := runner.runner.Run(ctx, argv)
	if err != nil {
		return 0, err
	}
	if result.ExitCode != 0 {
		return 0, &terminstance.Error{Code: terminalbackend.CodeUnavailable, Detail: "status observation unknown"}
	}
	for _, row := range splitRows(string(result.Stdout)) {
		name, count, ok := cutRow(row)
		if !ok {
			return 0, &terminstance.Error{Code: terminalbackend.CodeUnavailable, Detail: "status observation unknown"}
		}
		if name != instance {
			continue
		}
		attached, err := strconv.Atoi(count)
		if err != nil || attached < 0 {
			return 0, &terminstance.Error{Code: terminalbackend.CodeUnavailable, Detail: "status observation unknown"}
		}
		return attached, nil
	}
	return 0, &terminstance.Error{Code: terminalbackend.CodeUnavailable, Detail: "status observation unknown"}
}

// lookupMemory reads the AX-side state memory for one instance. The
// store may be nil in unit-scoped backends; a nil store reads as no
// memory rather than refusing, because presence alone still classifies
// the live triad (absent/parked/active).
func (runner *backend) lookupMemory(instance string) (terminalbackend.InstanceState, bool, error) {
	if runner.states == nil {
		return "", false, nil
	}
	return runner.states.Lookup(instance)
}

// attachCapable reports whether any admitted attach capability
// evidences presentation right now: local, remote, or web attach.
func (runner *backend) attachCapable() bool {
	return runner.admitted.Has("local_attach") ||
		runner.admitted.Has("remote_attach") ||
		runner.admitted.Has("web_attach")
}

// providerTriple computes the provider_present member and its two
// validation flags: the member is non-null only when observation was
// requested and evidenced; the flags let the landed CheckStatusResult
// verify that rule.
func providerTriple(requested, capable, observed bool) (*bool, bool, bool) {
	if !requested || !capable {
		return nil, requested, false
	}
	if !observed {
		return nil, requested, false
	}
	present := true
	return &present, requested, true
}

// statusEvidence derives the sorted-unique status evidence digests over
// the probe record: presence, pane rows, and attached count. Absence
// evidences the negative probe, never an empty set.
func statusEvidence(instance string, present bool, panes []string, attached int) []string {
	var evidence []string
	if !present {
		evidence = append(evidence, scalar.SHA256Digest([]byte("absent\n"+instance)).String())
	} else {
		evidence = append(evidence, scalar.SHA256Digest([]byte("panes\n"+instance+"\n"+strings.Join(panes, "\n"))).String())
		evidence = append(evidence, scalar.SHA256Digest([]byte("sessions\n"+instance+"\n"+strconv.Itoa(attached))).String())
	}
	sort.Strings(evidence)
	return evidence
}

// splitRows splits probe stdout into non-empty rows. Blank lines are
// skipped: tmux never emits them, and skipping is the tolerant read;
// a wholly blank output yields zero rows, which the callers refuse.
func splitRows(output string) []string {
	var rows []string
	for _, line := range strings.Split(output, "\n") {
		if line != "" {
			rows = append(rows, line)
		}
	}
	return rows
}

// cutRow splits one sessions row into name and attached count. Rows
// without the pipe separator are malformed.
func cutRow(row string) (string, string, bool) {
	name, count, found := strings.Cut(row, "|")
	if !found || name == "" || count == "" {
		return "", "", false
	}
	return name, count, true
}

// ClassifyInput is the pure status-classification input: the live tmux
// probe (presence, attached count, provider rows) plus the AX-side
// memory and the capability facts.
type ClassifyInput struct {
	Present         bool
	Attached        int
	ProviderRows    int
	Memory          terminalbackend.InstanceState
	MemoryFound     bool
	AttachCapable   bool
	ProviderWanted  bool
	ProviderCapable bool
	_               struct{}
}

// ClassifyStatus reconciles the live tmux probe against the AX-side
// state memory into the adopted status: state, wrapper presence,
// provider presence, and attachability. tmux reports only presence;
// the memory carries quiescing, stopped, and stale_fenced, which exist
// only on this side. The rules:
//
//   - absent probe, no memory (or memory absent): absent, no wrapper;
//   - absent probe, memory stopped or stale_fenced: the memory state,
//     no wrapper (a fenced incarnation keeps its fencing without a wrapper);
//   - absent probe, memory quiescing/active/parked/unavailable: unavailable
//     (contradiction: the wrapper vanished, or was never proven);
//   - present probe, memory quiescing or stale_fenced: the memory state
//     (tmux cannot show either; a fenced incarnation with a live wrapper
//     is what terminate-stale kills);
//   - present probe, memory stopped or unavailable: unavailable
//     (contradiction: a session resurrected, or was never proven);
//   - present probe, memory active/parked/absent/none: active when at
//     least one client is attached, otherwise parked.
//
// Attachability needs parked|active plus a currently evidenced attach
// capability; provider presence needs at least one pane row. Memory
// creating is unreachable (Record refuses it) and classifies
// unavailable either way.
func ClassifyStatus(input ClassifyInput) (terminalbackend.InstanceState, bool, bool, bool) {
	if !input.Present {
		return classifyAbsent(input)
	}
	return classifyPresent(input)
}

// classifyAbsent adopts the state for an absent probe.
func classifyAbsent(input ClassifyInput) (terminalbackend.InstanceState, bool, bool, bool) {
	if !input.MemoryFound || input.Memory == terminalbackend.StateAbsent {
		return terminalbackend.StateAbsent, false, false, false
	}
	switch input.Memory {
	case terminalbackend.StateStopped, terminalbackend.StateStaleFenced:
		return input.Memory, false, false, false
	default:
		return terminalbackend.StateUnavailable, false, false, false
	}
}

// classifyPresent adopts the state for a present probe.
func classifyPresent(input ClassifyInput) (terminalbackend.InstanceState, bool, bool, bool) {
	provider := input.ProviderRows > 0
	switch input.Memory {
	case terminalbackend.StateQuiescing, terminalbackend.StateStaleFenced:
		if !input.MemoryFound {
			break
		}
		return input.Memory, true, provider, false
	case terminalbackend.StateStopped, terminalbackend.StateUnavailable:
		if !input.MemoryFound {
			break
		}
		return terminalbackend.StateUnavailable, false, false, false
	case terminalbackend.StateCreating:
		return terminalbackend.StateUnavailable, false, false, false
	default:
	}
	state := terminalbackend.StateParked
	if input.Attached > 0 {
		state = terminalbackend.StateActive
	}
	attachable := input.AttachCapable && (state == terminalbackend.StateParked || state == terminalbackend.StateActive)
	return state, true, provider, attachable
}
