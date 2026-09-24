package tmuxserver

import (
	"context"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// This file orders the §4.2 after-restore sequence at the wrapper
// composition entry: read the latest locally known lease, attempt the
// mesh lease refresh under the measured bound, then decide through
// the landed wrapper owner and route to the decided effects. The
// backend restore entry stays the narrow §4.C row — recreate the
// wrapper under restore authorization and return the parked wire
// result — while this entry owns the order: the refresh runs before
// the decision, so a lapsed local grant under a refreshed remote
// winner offers remote attach instead of refusing at backend
// authorization that the offer path never reaches. Backend restore
// authorization is never weakened to make that path pass: the offer
// and park branches run no backend operation at all.

// WrapperRestoreRequest is one after-restore composition: the wrapper
// decision input the caller built (mode restore), the local fencing
// facts the ordered observation carries, and the backend restore
// request the resume branches execute.
type WrapperRestoreRequest struct {
	// Decide is the complete wrapper decision input. Mode must be
	// restore; Observation is ordered here (the caller-supplied one
	// is replaced by the refreshed/local observation), every other
	// member passes through to the landed owner.
	Decide axpane.Input
	// Grant is the process-local fencing authorization with its
	// policy, observed over the ordered winner. Presence is
	// required; expiry is the landed direction-first arm's verdict,
	// so a lapsed grant under a remote winner still parks remote
	// instead of refusing.
	Grant    sessrepo.FencingGrant
	HasGrant bool
	Policy   sessrepo.FencingPolicy
	// Restore is the backend restore request the launch and reattach
	// branches execute. Operation must be restore; the entry
	// executes nothing else.
	Restore OpRequest
}

// WrapperRestoreOutcome is one composed after-restore result: the
// ordered decision, the refresh audit, the winner the decision ran
// over, and the backend restore outcome on the resume branches only.
type WrapperRestoreOutcome struct {
	// Decision is the landed wrapper verdict: launch, reattach,
	// attach_remote, takeover_offer, parked, or refused.
	Decision axpane.Decision
	// RefreshAttempted reports that the mesh refresh ran; RefreshSucceeded
	// reports that it answered (a missing adapter, a failed attempt, and
	// a timed-out attempt all fall back to local knowledge unverified).
	RefreshAttempted bool
	RefreshSucceeded bool
	// RefreshBoundMs is the measured bound one attempt ran under.
	RefreshBoundMs uint64
	// WinnerLeaseID and WinnerHostID name the winning lease the
	// decision ran over: the refreshed winner, or the locally known
	// lease when the refresh did not answer.
	WinnerLeaseID string
	WinnerHostID  string
	// Restore is the backend restore outcome on launch and reattach,
	// nil on every other branch (those branches run no backend
	// operation and launch no provider).
	Restore *OpOutcome
}

// ExecuteWrapperRestore runs the §4.2 after-restore sequence to its
// decided effects:
//
//  1. read the latest locally known lease (CurrentLease);
//  2. attempt the mesh lease refresh under the measured bound (a
//     missing, failed, or timed-out attempt falls back to local
//     knowledge, unverified — never verified, never blocking
//     forever);
//  3. decide through the landed wrapper owner over the ordered
//     winner (local win plus valid materialization resumes);
//  4. offer remote attach/takeover when another host owns the session
//     in an interactive terminal;
//  5. park without launching the provider in all other cases.
//
// Only the launch and reattach decisions execute the backend restore,
// and only bound to the decided winner: the restore authorization
// must carry the ordered (lease_id, lease_epoch), so a refreshed
// winner for one lease never authorizes effects under another
// lease's grant. The offer, park, and refuse decisions return with
// no backend operation, no tmux exec, and no provider. Union-rival
// ambiguity is not modeled here (bound B38): the entry decides over
// the single refreshed or local winner, and a refresh adapter that
// observes rivals must surface them as refresh failure so the entry
// parks unverified.
func (lc *Lifecycle) ExecuteWrapperRestore(ctx context.Context, req WrapperRestoreRequest) (WrapperRestoreOutcome, error) {
	empty := WrapperRestoreOutcome{}
	if lc == nil || lc.CurrentLease == nil || lc.CurrentGeneration == nil || lc.Now == nil ||
		lc.Runner == nil || lc.Receipts == nil || lc.Bindings == nil || lc.Attach == nil || lc.States == nil {
		return empty, &Error{Code: terminalbackend.CodeProtocolError, Detail: "lifecycle dependencies"}
	}
	if req.Decide.Mode != axpane.ModeRestore {
		return empty, &Error{Code: CodeInvalidArguments, Detail: "wrapper restore mode"}
	}
	if req.Restore.Operation != string(terminalbackend.OperationRestore) {
		return empty, &Error{Code: CodeInvalidArguments, Detail: "wrapper restore operation"}
	}
	local := lc.CurrentLease()
	bound := lc.MeshRefreshTimeout
	if bound <= 0 {
		bound = DefaultMeshRefreshTimeout
	}
	winner, verified, attempted, succeeded := lc.refreshWinner(ctx, req.Decide.SessionID, local, bound)
	now := lc.Now()
	input := req.Decide
	input.Observation = fencing.Observation{
		SessionID:   req.Decide.SessionID,
		Winner:      winner,
		HasWinner:   winner.LeaseID != "" && winner.Epoch != 0,
		LocalHostID: lc.LocalHostID,
		Verified:    verified,
		Grant:       req.Grant,
		HasGrant:    req.HasGrant,
		Policy:      req.Policy,
		Now:         now,
	}
	decision := axpane.Decide(input)
	outcome := WrapperRestoreOutcome{
		Decision:         decision,
		RefreshAttempted: attempted,
		RefreshSucceeded: succeeded,
		RefreshBoundMs:   uint64(bound.Milliseconds()),
		WinnerLeaseID:    winner.LeaseID,
		WinnerHostID:     winner.HolderHostID,
	}
	switch decision.Action {
	case axpane.ActionLaunch, axpane.ActionReattach:
		if err := checkRestoreComposedBound(req.Decide, decision, req.Restore.Body, winner); err != nil {
			return outcome, err
		}
		restored, err := lc.Execute(ctx, req.Restore)
		outcome.Restore = &restored
		if err != nil {
			return outcome, err
		}
		return outcome, nil
	default:
		return outcome, nil
	}
}

// checkRestoreComposedBound binds the executed backend restore to the
// wrapper decision as one composed authorization: every identity the
// effect names must equal the identity the decision authorized, or
// the entry refuses the failed local precondition before any effect
// runs. The joined members are the session, the bootstrap operation,
// the admitted descriptor's instance, backend, and generation triple
// (parsed and matched by the landed terminalbackend owner inside
// Decide — this gate compares its output, never re-parses the
// bytes), and the ordered winner's (lease_id, lease_epoch) against
// the restore authorization. Matching a lease alone does not
// establish that the effect is the restore the decision authorized:
// without the full join a valid decision for one session,
// bootstrap, or instance would authorize effects under another's
// restore. A malformed restore body refuses its own parse error
// (the same refusal the backend entry would return). Backend restore
// authorization is never weakened to make the join fit: this gate
// only refuses what the decision did not select.
//
// Deliberately unjoined, with reasons: the restore authorization's
// holder host (the backend CheckAuthorization binds kind, expiry,
// lease, and epoch only — the holder routes inside Decide's fencing
// arm); materialization and checkpoint state (decision-local gates
// enforced inside Decide, and the restore checkpoint_id is opaque
// carried evidence with no production consumer); the prior binding
// (bound to the recorded receipt by checkPriorBinding, while the
// decision's ExistingBinding drives the bootstrap-window verdict);
// and the descriptor's versions and binding digest (provider-build
// and host-local namespaces that honestly differ from the AX API
// versions and the bootstrap binding the restore carries).
func checkRestoreComposedBound(decide axpane.Input, decision axpane.Decision, raw []byte, winner sessrepo.LeaseSummary) error {
	body, err := parseRestoreBody(raw)
	if err != nil {
		return err
	}
	if body.Context.SessionID != decide.SessionID {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "wrapper restore session binding"}
	}
	if body.BootstrapOperationID != decide.BootstrapOperationID {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "wrapper restore bootstrap binding"}
	}
	if body.Context.TerminalInstanceID != decision.Descriptor.TerminalInstanceID {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "wrapper restore instance binding"}
	}
	if body.Context.TerminalBackendID != decision.Descriptor.TerminalBackendID {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "wrapper restore backend binding"}
	}
	if body.Context.BackendGeneration != decision.Descriptor.BackendGeneration {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "wrapper restore generation binding"}
	}
	if body.Context.Authorization.LeaseID != winner.LeaseID ||
		body.Context.Authorization.LeaseEpoch != winner.Epoch {
		return &Error{Code: terminalbackend.CodePreconditionFailed, Detail: "wrapper restore winner binding"}
	}
	return nil
}

// refreshResult is one mesh lease-refresh answer from the adapter
// goroutine.
type refreshResult struct {
	winner RefreshWinner
	err    error
}

// refreshWinner attempts the §4.2 mesh lease refresh (step 2) under
// the measured bound and reports the ordered winner: the refreshed
// lease verified on success, the locally known lease unverified when
// the adapter is missing, fails, or times out, and the zero winner
// when no lease is known at all. Only a successful refresh verifies:
// local knowledge never self-verifies, so a failed attempt parks
// through the landed unverified arm instead of resuming.
//
// The bound is enforced, never cooperative: the adapter runs on its
// own goroutine and the entry takes whichever answers first — the
// adapter or the deadline. An adapter that ignores cancellation
// still loses at the bound, and an answer that arrives after the
// deadline elapsed is rejected even with a nil error (the expiry is
// checked on the answer path too, because both arms can be ready at
// once). The abandoned adapter send never blocks — the channel is
// buffered — and its goroutine exits when the adapter returns, so
// nothing here waits past the bound.
func (lc *Lifecycle) refreshWinner(ctx context.Context, sessionID string, local terminstance.LeaseView, bound time.Duration) (sessrepo.LeaseSummary, bool, bool, bool) {
	fallback := sessrepo.LeaseSummary{
		SessionID:    sessionID,
		LeaseID:      local.LeaseID,
		Epoch:        local.Epoch,
		HolderHostID: lc.LocalHostID,
		Reason:       "local-knowledge",
	}
	if lc.LeaseRefresh == nil {
		return fallback, false, false, false
	}
	refreshCtx, cancel := context.WithTimeout(ctx, bound)
	defer cancel()
	answered := make(chan refreshResult, 1)
	go func() {
		winner, err := lc.LeaseRefresh(refreshCtx, sessionID)
		answered <- refreshResult{winner: winner, err: err}
	}()
	if lc.RefreshHooks != nil && lc.RefreshHooks.BeforeSelect != nil {
		lc.RefreshHooks.BeforeSelect()
	}
	select {
	case result := <-answered:
		if result.err != nil || refreshCtx.Err() != nil {
			return fallback, false, true, false
		}
		return sessrepo.LeaseSummary{
			SessionID:    sessionID,
			LeaseID:      result.winner.LeaseID,
			Epoch:        result.winner.Epoch,
			HolderHostID: result.winner.HolderHostID,
			Reason:       "mesh-refresh",
		}, true, true, true
	case <-refreshCtx.Done():
		return fallback, false, true, false
	}
}
