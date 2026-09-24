//go:build !windows

package tmuxserver

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

type attachResult struct {
	outcome OpOutcome
	err     error
}

func awaitAttachResult(t *testing.T, ch <-chan attachResult) attachResult {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(3 * time.Second):
		t.Fatal("attach did not finish within the test bound")
		return attachResult{}
	}
}

// TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances pauses the first
// attach after staging but before its no-replace receipt install. A second
// independently constructed Lifecycle reaches the shared store lock while
// the first census still says empty. Once released, the first commits; the
// second must re-census under the lock and refuse the writable overlap.
// The test does not start tmux: Execute returns vectors but neither vector is
// run by this fixture.
func TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	staged := make(chan struct{})
	releaseStage := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseStage) }) }
	fx.lc.Attach.WithHooks(&termbind.AttachHooks{AfterStage: func(string) error {
		close(staged)
		<-releaseStage
		return nil
	}})
	defer release()

	firstBody := lxAttachBody(t, nil)
	firstDone := make(chan attachResult, 1)
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: firstBody, Source: "parked", Admitted: fullLifecycleAdmitted(),
		})
		firstDone <- attachResult{outcome: outcome, err: err}
	}()
	select {
	case <-staged:
	case <-time.After(3 * time.Second):
		t.Fatal("first attach did not reach its staged-receipt pause")
	}

	secondStore, err := termbind.OpenAttachStore(filepath.Join(fx.data, "attachstore"))
	if err != nil {
		t.Fatal(err)
	}
	lockAttempt := make(chan struct{}, 1)
	secondStore.WithHooks(&termbind.AttachHooks{BeforeAdmissionLock: func(string) {
		lockAttempt <- struct{}{}
	}})
	secondLifecycle := Lifecycle{
		RuntimeDir:         fx.lc.RuntimeDir,
		Root:               fx.lc.Root,
		Platform:           fx.lc.Platform,
		Runner:             fx.lc.Runner,
		Receipts:           fx.lc.Receipts,
		Bindings:           fx.lc.Bindings,
		Attach:             secondStore,
		States:             fx.lc.States,
		CurrentLease:       fx.lc.CurrentLease,
		CurrentGeneration:  fx.lc.CurrentGeneration,
		Now:                fx.lc.Now,
		Sleep:              fx.lc.Sleep,
		PollInterval:       fx.lc.PollInterval,
		ServerAdmission:    fx.lc.ServerAdmission,
		LocalHostID:        fx.lc.LocalHostID,
		LeaseRefresh:       fx.lc.LeaseRefresh,
		MeshRefreshTimeout: fx.lc.MeshRefreshTimeout,
		Hooks:              fx.lc.Hooks,
		RefreshHooks:       fx.lc.RefreshHooks,
	}
	withoutConcurrentInput := withoutCapability("multiple_input_clients")
	secondBody := lxAttachBody(t, func(object map[string]any) { object["client_id"] = lxClientB })
	secondDone := make(chan attachResult, 1)
	go func() {
		outcome, err := secondLifecycle.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: secondBody, Source: "active", Admitted: withoutConcurrentInput,
		})
		secondDone <- attachResult{outcome: outcome, err: err}
	}()
	select {
	case <-lockAttempt:
	case <-time.After(3 * time.Second):
		t.Fatal("second attach did not reach the shared admission lock")
	}
	select {
	case result := <-secondDone:
		t.Fatalf("second attach completed while the first receipt was staged: err=%v outcome=%+v", result.err, result.outcome)
	case <-time.After(50 * time.Millisecond):
	}

	release()
	first := awaitAttachResult(t, firstDone)
	if first.err != nil || first.outcome.Attach == nil || !first.outcome.Attach.InputAuthorized {
		t.Fatalf("first writable attach = %+v, %v", first.outcome.Attach, first.err)
	}
	second := awaitAttachResult(t, secondDone)
	requireLocalCode(t, second.err, "terminal_backend_capability_unproven", "operation capability conditional")
	if _, found, err := secondLifecycle.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
		t.Fatalf("second receipt found=%v err=%v; want no receipt for the refused client", found, err)
	}
	if second.outcome.Attach != nil || len(second.outcome.Argv) != 0 {
		t.Fatalf("refused second attach returned authorization or argv: %+v %q", second.outcome.Attach, second.outcome.Argv)
	}
}

// TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances exercises
// the same census-to-receipt interval for two read-only clients. multi_attach
// is absent, so a second client must not pass even though neither client can
// send input. The first receipt is paused after staging while a separately
// opened Lifecycle attempts admission against the same instance lock.
func TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances(t *testing.T) {
	admitted := withoutCapability("multi_attach")
	fx := newLifecycleFixture(t, admitted)
	staged := make(chan struct{})
	releaseStage := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseStage) }) }
	fx.lc.Attach.WithHooks(&termbind.AttachHooks{AfterStage: func(string) error {
		close(staged)
		<-releaseStage
		return nil
	}})
	defer release()

	readOnly := func(clientID string) []byte {
		return lxAttachBody(t, func(object map[string]any) {
			object["client_id"] = clientID
			object["input_authorized"] = false
			object["authorization"] = lxAttachAuth("local_only", false)
		})
	}
	firstBody := readOnly(lxClient)
	firstDone := make(chan attachResult, 1)
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: firstBody, Source: "parked", Admitted: admitted,
		})
		firstDone <- attachResult{outcome: outcome, err: err}
	}()
	select {
	case <-staged:
	case <-time.After(3 * time.Second):
		t.Fatal("first read-only attach did not reach its staged-receipt pause")
	}

	secondStore, err := termbind.OpenAttachStore(filepath.Join(fx.data, "attachstore"))
	if err != nil {
		t.Fatal(err)
	}
	lockAttempt := make(chan struct{}, 1)
	secondStore.WithHooks(&termbind.AttachHooks{BeforeAdmissionLock: func(string) {
		lockAttempt <- struct{}{}
	}})
	secondLifecycle := &Lifecycle{
		RuntimeDir:         fx.lc.RuntimeDir,
		Root:               fx.lc.Root,
		Platform:           fx.lc.Platform,
		Runner:             fx.lc.Runner,
		Receipts:           fx.lc.Receipts,
		Bindings:           fx.lc.Bindings,
		Attach:             secondStore,
		States:             fx.lc.States,
		CurrentLease:       fx.lc.CurrentLease,
		CurrentGeneration:  fx.lc.CurrentGeneration,
		Now:                fx.lc.Now,
		Sleep:              fx.lc.Sleep,
		PollInterval:       fx.lc.PollInterval,
		ServerAdmission:    fx.lc.ServerAdmission,
		LocalHostID:        fx.lc.LocalHostID,
		LeaseRefresh:       fx.lc.LeaseRefresh,
		MeshRefreshTimeout: fx.lc.MeshRefreshTimeout,
		Hooks:              fx.lc.Hooks,
		RefreshHooks:       fx.lc.RefreshHooks,
	}
	secondBody := readOnly(lxClientB)
	secondDone := make(chan attachResult, 1)
	go func() {
		outcome, err := secondLifecycle.Execute(context.Background(), OpRequest{
			Operation: "attach", Body: secondBody, Source: "active", Admitted: admitted,
		})
		secondDone <- attachResult{outcome: outcome, err: err}
	}()
	lockReached := false
	var earlySecond *attachResult
	select {
	case <-lockAttempt:
		lockReached = true
	case result := <-secondDone:
		// Continue through the receipt and vector assertions below. Under
		// a missing lock, this is the effect the test is intended to attack.
		earlySecond = &result
	case <-time.After(3 * time.Second):
	}
	if lockReached {
		select {
		case result := <-secondDone:
			earlySecond = &result
		case <-time.After(50 * time.Millisecond):
		}
	}

	release()
	first := awaitAttachResult(t, firstDone)
	if first.err != nil || first.outcome.Attach == nil || first.outcome.Attach.InputAuthorized || len(first.outcome.Argv) == 0 {
		t.Fatalf("first read-only attach = %+v argv=%q err=%v", first.outcome.Attach, first.outcome.Argv, first.err)
	}
	second := attachResult{}
	if earlySecond != nil {
		second = *earlySecond
	} else {
		second = awaitAttachResult(t, secondDone)
	}
	var backendErr *Error
	if !errors.As(second.err, &backendErr) || backendErr.Code != "terminal_backend_capability_unproven" {
		t.Errorf("second read-only attach error = %T %v, want literal terminal_backend_capability_unproven", second.err, second.err)
	}
	if _, found, err := secondLifecycle.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
		t.Errorf("second receipt found=%v err=%v; want no receipt for the refused read-only client", found, err)
	}
	peers, err := secondLifecycle.Attach.Peers(lxSession, lxInstance, "0198f4c8-8e50-7f66-8f70-555555555554")
	if err != nil || len(peers) != 1 || peers[0].ClientID != lxClient {
		t.Errorf("receipt census after concurrent refusal = %+v, %v; want only the first client", peers, err)
	}
	if second.outcome.Attach != nil || len(second.outcome.Argv) != 0 || fx.runner.callCount() != 0 {
		t.Errorf("refused second read-only attach returned an effect: attach=%+v argv=%q tmux-calls=%d", second.outcome.Attach, second.outcome.Argv, fx.runner.callCount())
	}
	if !lockReached {
		t.Error("second read-only attach did not enter the shared admission lock")
	}
}

// TestAttachOverlapInputGateChecksEveryPeer puts the read-only client first
// in the sorted receipt census and the input-authorized peer after it. The
// second subtest has two input-authorized peers after that read-only client.
// A new input-authorized client must be refused in both shapes when the
// current admission lacks multiple_input_clients.
func TestAttachOverlapInputGateChecksEveryPeer(t *testing.T) {
	clientC := "0198f4c8-8e50-7f66-8f70-555555555553"
	clientD := "0198f4c8-8e50-7f66-8f70-555555555554"
	for _, test := range []struct {
		name                string
		inputPeers          int
		seedWithInputClient bool
	}{
		{name: "input-peer-after-read-only-peer", inputPeers: 1},
		{name: "several-input-peers-after-read-only-peer", inputPeers: 2, seedWithInputClient: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			admitted := withoutCapability("multiple_input_clients")
			fx := newLifecycleFixture(t, admitted)
			readOnly := lxAttachBody(t, func(object map[string]any) {
				object["input_authorized"] = false
				object["authorization"] = lxAttachAuth("local_only", false)
			})
			if _, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "attach", Body: readOnly, Source: "parked", Admitted: admitted,
			}); err != nil {
				t.Fatalf("read-only first peer refused: %v", err)
			}

			inputIDs := []string{lxClientB, clientC}
			seedAdmission := admitted
			if test.seedWithInputClient {
				seedAdmission = fullLifecycleAdmitted()
			}
			for _, clientID := range inputIDs[:test.inputPeers] {
				inputPeer := lxAttachBody(t, func(object map[string]any) { object["client_id"] = clientID })
				if _, err := fx.lc.Execute(context.Background(), OpRequest{
					Operation: "attach", Body: inputPeer, Source: "active", Admitted: seedAdmission,
				}); err != nil {
					t.Fatalf("input-authorized peer %s refused during setup: %v", clientID, err)
				}
			}

			newClientID := clientC
			if test.inputPeers == 2 {
				newClientID = clientD
			}
			peers, err := fx.lc.Attach.Peers(lxSession, lxInstance, newClientID)
			if err != nil {
				t.Fatal(err)
			}
			if len(peers) != test.inputPeers+1 || peers[0].ClientID != lxClient || peers[0].InputAuthorized {
				t.Fatalf("precondition peers = %+v; want read-only %s first and %d input peers after it", peers, lxClient, test.inputPeers)
			}
			for i := 0; i < test.inputPeers; i++ {
				if peers[i+1].ClientID != inputIDs[i] || !peers[i+1].InputAuthorized {
					t.Fatalf("precondition peer %d = %+v; want input-authorized %s after the first read-only peer", i+1, peers[i+1], inputIDs[i])
				}
			}

			candidate := lxAttachBody(t, func(object map[string]any) { object["client_id"] = newClientID })
			outcome, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "attach", Body: candidate, Source: "active", Admitted: admitted,
			})
			var backendErr *Error
			if !errors.As(err, &backendErr) || backendErr.Code != "terminal_backend_capability_unproven" {
				t.Errorf("candidate attach error = %T %v, want literal terminal_backend_capability_unproven", err, err)
			}
			if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, newClientID); err != nil || found {
				t.Errorf("candidate receipt found=%v err=%v; want no receipt", found, err)
			}
			if outcome.Attach != nil || len(outcome.Argv) != 0 {
				t.Errorf("refused input overlap returned attach=%+v argv=%q", outcome.Attach, outcome.Argv)
			}
			peersAfter, err := fx.lc.Attach.Peers(lxSession, lxInstance, newClientID)
			if err != nil || len(peersAfter) != test.inputPeers+1 {
				t.Errorf("receipt census after refusal = %+v, %v; want no newly committed receipt", peersAfter, err)
			}
		})
	}
}

// TestAttachOverlapRequiresMultiAttachForInputClient ensures the overlap
// capability applies to an input-authorized requester as well as a read-only
// requester. multiple_input_clients is present, isolating the missing
// multi_attach refusal.
func TestAttachOverlapRequiresMultiAttachForInputClient(t *testing.T) {
	admitted := withoutCapability("multi_attach")
	fx := newLifecycleFixture(t, admitted)
	first := lxAttachBody(t, func(object map[string]any) {
		object["input_authorized"] = false
		object["authorization"] = lxAttachAuth("local_only", false)
	})
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: first, Source: "parked", Admitted: admitted,
	}); err != nil {
		t.Fatal(err)
	}
	second := lxAttachBody(t, func(object map[string]any) { object["client_id"] = lxClientB })
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: second, Source: "active", Admitted: admitted,
	})
	requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
		t.Fatalf("input-authorized candidate receipt found=%v err=%v; want no receipt", found, err)
	}
	if outcome.Attach != nil || len(outcome.Argv) != 0 {
		t.Fatalf("refused input-authorized overlap returned attach=%+v argv=%q", outcome.Attach, outcome.Argv)
	}
}

// TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer states the
// liveness rule at Lifecycle.Execute: the caller has not executed the first
// returned vector, so no tmux client can be shown live. The durable receipt
// nevertheless remains a possible peer; missing liveness evidence is not
// proof of detach or death.
func TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer(t *testing.T) {
	for _, test := range []struct {
		name              string
		missingCapability string
		inputAuthorized   bool
	}{
		{name: "read-only-peer-without-multi-attach", missingCapability: "multi_attach"},
		{name: "input-peer-without-multiple-input-clients", missingCapability: "multiple_input_clients", inputAuthorized: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			admitted := withoutCapability(test.missingCapability)
			fx := newLifecycleFixture(t, admitted)
			makeBody := func(clientID string) []byte {
				return lxAttachBody(t, func(object map[string]any) {
					object["client_id"] = clientID
					if !test.inputAuthorized {
						object["input_authorized"] = false
						object["authorization"] = lxAttachAuth("local_only", false)
					}
				})
			}
			first, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "attach", Body: makeBody(lxClient), Source: "parked", Admitted: admitted,
			})
			if err != nil || first.Attach == nil || first.Attach.InputAuthorized != test.inputAuthorized || len(first.Argv) == 0 {
				t.Fatalf("first attach = %+v argv=%q err=%v; input_authorized=%v", first.Attach, first.Argv, err, test.inputAuthorized)
			}
			if fx.runner.callCount() != 0 {
				t.Fatalf("Execute unexpectedly ran tmux %d times", fx.runner.callCount())
			}

			second, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "attach", Body: makeBody(lxClientB), Source: "active", Admitted: admitted,
			})
			requireLocalCode(t, err, "terminal_backend_capability_unproven", "operation capability conditional")
			if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
				t.Fatalf("second receipt found=%v err=%v; want no receipt while liveness is unknown", found, err)
			}
			if second.Attach != nil || len(second.Argv) != 0 || fx.runner.callCount() != 0 {
				t.Fatalf("unproven overlap returned effect: attach=%+v argv=%q tmux-calls=%d", second.Attach, second.Argv, fx.runner.callCount())
			}
		})
	}
}

// TestAttachConcurrentInputRequiresAXPolicy is the composition row for the
// two independent requirements in §4.C: even with both overlap capabilities,
// input_authorized must have matching AX authorization before the receipt or
// writable vector can be returned.
func TestAttachConcurrentInputRequiresAXPolicy(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	unauthorized := lxAttachBody(t, func(object map[string]any) {
		object["client_id"] = lxClientB
		object["authorization"] = lxAttachAuth("local_only", false)
	})
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: unauthorized, Source: "active", Admitted: fullLifecycleAdmitted(),
	})
	var backendErr *terminalbackend.Error
	if !errors.As(err, &backendErr) || backendErr.Code != "terminal_backend_unauthorized" {
		t.Fatalf("AX input policy refusal = %T %v, want literal terminal_backend_unauthorized", err, err)
	}
	if _, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClientB); err != nil || found {
		t.Fatalf("unauthorized second receipt found=%v err=%v; want no receipt", found, err)
	}
	if outcome.Attach != nil || len(outcome.Argv) != 0 {
		t.Fatalf("unauthorized concurrent input returned attach=%+v argv=%q", outcome.Attach, outcome.Argv)
	}
}
