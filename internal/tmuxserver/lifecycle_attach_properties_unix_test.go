//go:build !windows

package tmuxserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

func TestExecuteAttachLostResponseReplaysRecordedOutcome(t *testing.T) {
	admitted := fullLifecycleAdmitted()
	fx := newLifecycleFixture(t, admitted)
	staged, installed := 0, 0
	fx.lc.Attach.WithHooks(&termbind.AttachHooks{
		AfterStage: func(string) error { staged++; return nil },
		AfterInstall: func(string) error {
			installed++
			return nil
		},
	})
	request := OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted,
	}

	// The first successful transport response is lost; only the durable
	// receipt can tell the retry what result it must reproduce.
	if _, err := fx.lc.Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	receipt, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient)
	if err != nil || !found {
		t.Fatalf("durable receipt = %+v found=%v err=%v", receipt, found, err)
	}
	if _, err := time.Parse(time.RFC3339Nano, receipt.CreatedAt); err != nil {
		t.Fatalf("recorded timestamp %q is not RFC3339: %v", receipt.CreatedAt, err)
	}
	receiptPath := filepath.Join(fx.data, "attachstore", "termbind", "attach", lxInstance, lxClient+".json")
	firstBytes, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}

	retry, err := fx.lc.Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Attach == nil || retry.Attach.ClientMirrorID != lxClient || !retry.Attach.InputAuthorized {
		t.Fatalf("replayed attach = %+v", retry.Attach)
	}
	wantEvidence := effectEvidence(terminalbackend.EffectAttachClientCreated, terminalbackend.OperationAttach, lxInstance, lxClient, "local_only")
	if !reflect.DeepEqual(retry.Attach.EvidenceIDs, []string{wantEvidence}) {
		t.Fatalf("replayed evidence = %v, want recorded effect %s", retry.Attach.EvidenceIDs, wantEvidence)
	}
	if retry.Attach.Descriptor != fx.socket+"\n"+lxInstance+"\n"+lxClient {
		t.Fatalf("replayed descriptor = %q", retry.Attach.Descriptor)
	}
	secondBytes, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("retry rewrote the durable attach receipt or its original timestamp")
	}
	replayed, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient)
	if err != nil || !found || replayed != receipt {
		t.Fatalf("replayed receipt = %+v found=%v err=%v, original=%+v", replayed, found, err, receipt)
	}
	entries, err := os.ReadDir(filepath.Dir(receiptPath))
	if err != nil {
		t.Fatal(err)
	}
	receipts := 0
	for _, entry := range entries {
		if entry.Name() == lxClient+".json" {
			receipts++
		} else if entry.Name() != ".admission.lock" {
			t.Fatalf("unexpected attach-store artifact after replay: %s", entry.Name())
		}
	}
	if receipts != 1 {
		t.Fatalf("durable receipt count = %d, want one; entries=%v", receipts, entries)
	}
	if staged != 1 || installed != 1 {
		t.Fatalf("receipt durable installs staged=%d installed=%d, want one", staged, installed)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("attach retry execed %d tmux commands", fx.runner.callCount())
	}
}

func TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable(t *testing.T) {
	admitted := fullLifecycleAdmitted()
	fx := newLifecycleFixture(t, admitted)
	readOnlyRaw := lxAttachBody(t, func(object map[string]any) {
		object["input_authorized"] = false
		object["authorization"] = lxAttachAuth("local_only", false)
	})
	readOnly, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: readOnlyRaw, Source: "parked", Admitted: admitted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if readOnly.Attach == nil || readOnly.Attach.InputAuthorized {
		t.Fatalf("first attach did not remain read-only: %+v", readOnly.Attach)
	}
	stored, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient)
	if err != nil || !found || stored.InputAuthorized {
		t.Fatalf("stored read-only receipt = %+v found=%v err=%v", stored, found, err)
	}
	receiptPath := filepath.Join(fx.data, "attachstore", "termbind", "attach", lxInstance, lxClient+".json")
	before, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}

	writableRaw := lxAttachBody(t, func(object map[string]any) {
		object["input_authorized"] = true
		object["authorization"] = lxAttachAuth("local_only", true)
	})
	outcome, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: writableRaw, Source: "active", Admitted: admitted,
	})
	requireEngineCode(t, err, "idempotency_mismatch")
	if outcome.Attach != nil || len(outcome.Argv) != 0 {
		t.Fatalf("read-only replay produced a writable attach vector: %+v", outcome)
	}
	afterReceipt, found, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient)
	if err != nil || !found || afterReceipt != stored || afterReceipt.InputAuthorized {
		t.Fatalf("refused escalation changed receipt: %+v found=%v err=%v; before=%+v", afterReceipt, found, err, stored)
	}
	after, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("refused escalation changed the read-only receipt")
	}
	if _, found, err := fx.lc.States.Lookup(lxInstance); err != nil || found {
		t.Fatalf("refused escalation recorded writable active state: found=%v err=%v", found, err)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("attach escalation execed %d tmux commands", fx.runner.callCount())
	}
}

func TestExecuteAttachDoesNotReadOrChangeSessionLease(t *testing.T) {
	admitted := fullLifecycleAdmitted()
	fx := newLifecycleFixture(t, admitted)
	repository, err := sessrepo.Open(filepath.Join(fx.data, "ownership-neutrality"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateSession(attachOwnershipSessionRecordBytes(t)); err != nil {
		t.Fatalf("seed session record: %v", err)
	}
	leaseRef, err := repository.CreateLease(lxSession, sessrepo.CreateLeaseInput{
		LeaseID: lxLease, HolderHostID: lxHost, IssuedByHostID: lxHost, CreatedAt: "2026-08-19T04:09:30.000Z",
	})
	if err != nil {
		t.Fatalf("seed lease: %v", err)
	}
	snapshot := func() (string, []byte) {
		t.Helper()
		events, err := repository.ListEvents(lxSession)
		if err != nil {
			t.Fatal(err)
		}
		leases, err := repository.ListLeases(lxSession)
		if err != nil {
			t.Fatal(err)
		}
		leaseBytes, err := repository.GetLease(lxSession, leaseRef.RecordID)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(struct {
			Events []sessrepo.EventSummary
			Leases []sessrepo.LeaseSummary
		}{events, leases})
		if err != nil {
			t.Fatal(err)
		}
		return string(encoded), leaseBytes
	}
	eventsAndLeasesBefore, leaseBytesBefore := snapshot()

	leaseReads := 0
	leaseRefreshCalls := 0
	fx.lc.CurrentLease = func() terminstance.LeaseView {
		leaseReads++
		return terminstance.LeaseView{LeaseID: lxLease, Epoch: 1}
	}
	fx.lc.LeaseRefresh = func(context.Context, string) (RefreshWinner, error) {
		leaseRefreshCalls++
		_, err := repository.CompareAndSwapLease(lxSession, sessrepo.LeaseExpectation{RecordID: leaseRef.RecordID}, sessrepo.SuccessorLeaseInput{
			CreateLeaseInput: sessrepo.CreateLeaseInput{
				LeaseID: lxLeaseB, HolderHostID: lxHost, IssuedByHostID: lxHost,
				CreatedAt: "2026-08-19T04:10:30.000Z",
			},
			Reason: "graceful_takeover", CheckpointID: lxDigestA,
		})
		return RefreshWinner{}, err
	}
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted,
	}); err != nil {
		t.Fatal(err)
	}
	if leaseReads != 0 {
		t.Fatalf("attach read current lease %d times, want zero", leaseReads)
	}
	if leaseRefreshCalls != 0 {
		t.Fatalf("attach refreshed or acquired ownership %d times, want zero", leaseRefreshCalls)
	}
	eventsAndLeasesAfter, leaseBytesAfter := snapshot()
	if eventsAndLeasesAfter != eventsAndLeasesBefore || !bytes.Equal(leaseBytesAfter, leaseBytesBefore) {
		t.Fatal("attach changed the session event chain or durable lease record")
	}
}

// TestExecuteSerializesSameClientAttachBeforeConcurrentWinnerArm proves
// that Lifecycle.Execute holds the per-instance admission lock across the
// peer census and receipt install. A same-client retry that reaches the
// lock while the first request is staged cannot reach AttachStore.install's
// concurrent-winner branch; it replays the committed receipt instead.
func TestExecuteSerializesSameClientAttachBeforeConcurrentWinnerArm(t *testing.T) {
	admitted := fullLifecycleAdmitted()
	fx := newLifecycleFixture(t, admitted)
	firstStaged := make(chan struct{})
	releaseStage := make(chan struct{})
	firstLockAttempt := make(chan struct{}, 1)
	secondLockAttempt := make(chan struct{}, 1)
	secondStaged := make(chan struct{}, 1)
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseStage) }) }
	t.Cleanup(release)
	var firstStageCount atomic.Int32
	var firstInstallCount atomic.Int32

	fx.lc.Attach.WithHooks(&termbind.AttachHooks{
		BeforeAdmissionLock: func(string) { firstLockAttempt <- struct{}{} },
		AfterStage: func(string) error {
			firstStageCount.Add(1)
			close(firstStaged)
			<-releaseStage
			return nil
		},
		AfterInstall: func(string) error {
			firstInstallCount.Add(1)
			return nil
		},
	})
	body := lxAttachBody(t, nil)
	request := OpRequest{Operation: "attach", Body: body, Source: "parked", Admitted: admitted}
	firstDone := make(chan attachResult, 1)
	go func() {
		outcome, err := fx.lc.Execute(context.Background(), request)
		firstDone <- attachResult{outcome: outcome, err: err}
	}()
	select {
	case <-firstLockAttempt:
	case <-time.After(3 * time.Second):
		t.Fatal("first same-client attach did not reach admission lock")
	}
	select {
	case <-firstStaged:
	case <-time.After(3 * time.Second):
		t.Fatal("first same-client attach did not reach staged receipt")
	}

	secondStore, err := termbind.OpenAttachStore(filepath.Join(fx.data, "attachstore"))
	if err != nil {
		t.Fatal(err)
	}
	secondStore.WithHooks(&termbind.AttachHooks{
		BeforeAdmissionLock: func(string) { secondLockAttempt <- struct{}{} },
		AfterStage: func(string) error {
			secondStaged <- struct{}{}
			return nil
		},
	})
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
	secondDone := make(chan attachResult, 1)
	go func() {
		outcome, err := secondLifecycle.Execute(context.Background(), request)
		secondDone <- attachResult{outcome: outcome, err: err}
	}()
	select {
	case <-secondLockAttempt:
	case <-time.After(3 * time.Second):
		t.Fatal("same-client retry did not reach the shared admission lock")
	}
	select {
	case <-secondStaged:
		t.Fatal("same-client retry staged a second receipt while the first was paused")
	case result := <-secondDone:
		t.Fatalf("same-client retry completed before the first receipt committed: err=%v outcome=%+v", result.err, result.outcome)
	case <-time.After(50 * time.Millisecond):
	}

	release()
	first := awaitAttachResult(t, firstDone)
	second := awaitAttachResult(t, secondDone)
	for label, result := range map[string]attachResult{"first": first, "second": second} {
		if result.err != nil || result.outcome.Attach == nil || !result.outcome.Attach.InputAuthorized {
			t.Fatalf("%s same-client attach = %+v, %v", label, result.outcome.Attach, result.err)
		}
	}
	if firstStageCount.Load() != 1 || firstInstallCount.Load() != 1 {
		t.Fatalf("first store stage/install counts = %d/%d, want 1/1", firstStageCount.Load(), firstInstallCount.Load())
	}
	select {
	case <-secondStaged:
		t.Fatal("same-client retry reached the concurrent-winner install arm")
	default:
	}
	firstReceipt, firstFound, err := fx.lc.Attach.Lookup(lxSession, lxInstance, lxClient)
	if err != nil || !firstFound {
		t.Fatalf("first receipt = %+v found=%v err=%v", firstReceipt, firstFound, err)
	}
	secondReceipt, secondFound, err := secondStore.Lookup(lxSession, lxInstance, lxClient)
	if err != nil || !secondFound || secondReceipt != firstReceipt {
		t.Fatalf("replayed receipt = %+v found=%v err=%v, original=%+v", secondReceipt, secondFound, err, firstReceipt)
	}
}

func TestForegroundAcquireComposesProductionProbeWithAttach(t *testing.T) {
	admitted := fullLifecycleAdmitted()
	fx := newLifecycleFixture(t, admitted)
	listener, err := net.Listen("unix", fx.socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	// tmux toggles the owner's execute bit for an attached session. The
	// active socket remains private and must stay reconnectable.
	if err := os.Chmod(fx.socket, 0o700); err != nil {
		t.Fatal(err)
	}
	deps := Production(fx.root, fx.lc.Platform, fx.runner, UnixDialer{}, func() (RealmAdmission, error) {
		return RealmAdmission{Admitted: admitted, RawGeneration: lxGeneration}, nil
	})
	acquire := validRequest(t, fx.root)
	acquire.SessionID = lxSession
	acquire.Generation = lxGeneration
	acquire.Deps = deps
	acquired, err := Acquire(acquire)
	if err != nil {
		t.Fatal(err)
	}
	if acquired.Via != ViaAttached || acquired.Socket != fx.socket || len(acquired.Argv) != 0 {
		t.Fatalf("foreground acquisition = %+v, want running production-probed server", acquired)
	}
	attached, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "attach", Body: lxAttachBody(t, nil), Source: "parked", Admitted: admitted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if attached.Attach == nil || attached.Attach.Descriptor != fx.socket+"\n"+lxInstance+"\n"+lxClient {
		t.Fatalf("foreground attach result = %+v", attached.Attach)
	}
	want := []string{"tmux", "-S", fx.socket, "attach-session", "-t", lxInstance}
	if !reflect.DeepEqual(attached.Argv, want) {
		t.Fatalf("foreground attach vector = %q, want %q", attached.Argv, want)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("running-server foreground attach execed %d tmux commands", fx.runner.callCount())
	}
}

func TestExecuteEveryOperationRefusesSocketSubstitutionBeforeDispatch(t *testing.T) {
	for _, operation := range lifecycleOperations {
		t.Run(operation, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			primary, err := net.Listen("unix", fx.socket)
			if err != nil {
				t.Fatal(err)
			}
			defer primary.Close()
			if err := os.Chmod(fx.socket, 0o600); err != nil {
				t.Fatal(err)
			}
			decoyPath := fx.socket + ".decoy"
			decoy, err := net.Listen("unix", decoyPath)
			if err != nil {
				t.Fatal(err)
			}
			defer decoy.Close()
			if err := os.Chmod(decoyPath, 0o600); err != nil {
				t.Fatal(err)
			}

			previous := lstatCustodySocket
			calls := 0
			lstatCustodySocket = func(path string) (os.FileInfo, error) {
				if path == fx.socket {
					calls++
					if calls == 2 {
						if err := os.Rename(decoyPath, fx.socket); err != nil {
							return nil, err
						}
					}
				}
				return previous(path)
			}
			defer func() { lstatCustodySocket = previous }()

			_, err = fx.lc.Execute(context.Background(), OpRequest{Operation: operation})
			requireLocalCode(t, err, "tmux_unsafe_socket_path", "socket identity")
			if calls != 2 {
				t.Fatalf("socket custody lstat calls = %d, want exactly two", calls)
			}
			if fx.runner.callCount() != 0 {
				t.Fatalf("%s dispatched %d tmux commands after socket substitution", operation, fx.runner.callCount())
			}
		})
	}
}

func TestExecuteEveryOperationRefusesForeignOrPermissiveSocket(t *testing.T) {
	for _, operation := range lifecycleOperations {
		for _, refusal := range []struct {
			name   string
			detail string
			stage  func(*testing.T, string)
		}{
			{
				name:   "foreign-owner",
				detail: "socket ownership",
				stage: func(t *testing.T, socket string) {
					listener, err := net.Listen("unix", socket)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = listener.Close() })
					if err := os.Chmod(socket, 0o600); err != nil {
						t.Fatal(err)
					}
					previous := lstatCustodySocket
					lstatCustodySocket = func(path string) (os.FileInfo, error) {
						info, err := previous(path)
						if err != nil || path != socket {
							return info, err
						}
						stat, ok := info.Sys().(*syscall.Stat_t)
						if !ok {
							t.Fatalf("socket stat is %T, want *syscall.Stat_t", info.Sys())
						}
						foreign := *stat
						foreign.Uid++
						return socketFileInfo{FileInfo: info, stat: &foreign}, nil
					}
					t.Cleanup(func() { lstatCustodySocket = previous })
				},
			},
			{
				name:   "permissive-mode",
				detail: "socket permissions",
				stage: func(t *testing.T, socket string) {
					listener, err := net.Listen("unix", socket)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = listener.Close() })
					if err := os.Chmod(socket, 0o644); err != nil {
						t.Fatal(err)
					}
				},
			},
			{
				name:   "symlink",
				detail: "socket kind",
				stage: func(t *testing.T, socket string) {
					target := socket + ".target"
					if err := os.WriteFile(target, []byte("not a socket"), 0o600); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(target, socket); err != nil {
						t.Fatal(err)
					}
				},
			},
		} {
			t.Run(operation+"/"+refusal.name, func(t *testing.T) {
				fx := newLifecycleFixture(t, fullLifecycleAdmitted())
				refusal.stage(t, fx.socket)
				_, err := fx.lc.Execute(context.Background(), OpRequest{Operation: operation})
				requireLocalCode(t, err, "tmux_unsafe_socket_path", refusal.detail)
				if fx.runner.callCount() != 0 {
					t.Fatalf("%s dispatched %d tmux commands after socket refusal", operation, fx.runner.callCount())
				}
			})
		}
	}
}

const attachOwnershipSessionRecord = `{ 
  "schema": "urn:ax:schema:session-record",
  "schema_version": "1.0.0",
  "record_id": "sha256:d61701066a7f5dd37bf35fea0e85e7f154251355ad24a49976532d7f79ddc772",
  "subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "name": "payments-api",
  "kind": "direct",
  "created_at": "2026-08-19T04:00:00.000Z",
  "created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
  "provider_id": "codex",
  "workspace_group_id": "0198f4c8-5b20-7c33-8c4d-1234567890ab",
  "execution_profile": "yolo",
  "launch_plan": {
    "argv": ["codex"],
    "cwd_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890ab",
    "cwd_relative": "src",
    "env_names": ["OPENAI_API_KEY"],
    "env_literals": {},
    "contains_secrets": false,
    "extensions": {}
  },
  "task_board": null,
  "fork_provenance": null,
  "extensions": {}
}`

func attachOwnershipSessionRecordBytes(t *testing.T) []byte {
	t.Helper()
	var record map[string]any
	if err := json.Unmarshal([]byte(attachOwnershipSessionRecord), &record); err != nil {
		t.Fatal(err)
	}
	record["subject_id"] = lxSession
	record["session_id"] = lxSession
	record["created_by_host_id"] = lxHost
	record["record_id"] = lxMintIdentity(t, record, "record_id")
	return lxBody(t, record)
}
