//go:build !windows

package tmuxserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/gitsnap"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/termbind"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

func captureLifecycleFixture(t *testing.T) *lxFixture {
	t.Helper()
	// The real lifecycle entry checks the derived macOS socket length. Use the
	// short, testing.T-owned root so this fixture reaches capture state first.
	short := shortTestTempDir(t)
	root := filepath.Join(short, "ax")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return captureLifecycleFixtureAt(t, root, t.TempDir())
}

func captureLifecycleFixtureAt(t *testing.T, root, data string) *lxFixture {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(data, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	runtimeDir, err := EnsureRuntimeDir(resolved, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := terminstance.OpenReceiptStore(filepath.Join(data, "receipts"))
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := axpane.Open(filepath.Join(data, "bindings"))
	if err != nil {
		t.Fatal(err)
	}
	attach, err := termbind.OpenAttachStore(filepath.Join(data, "attach"))
	if err != nil {
		t.Fatal(err)
	}
	states, err := OpenInstanceStates(filepath.Join(data, "states"))
	if err != nil {
		t.Fatal(err)
	}
	runner := newFakeRunner()
	lc := &Lifecycle{
		RuntimeDir: runtimeDir, Root: resolved, Platform: scalar.PlatformMacOS,
		Runner: runner, Receipts: receipts, Bindings: bindings, Attach: attach, States: states,
		CurrentLease:      func() terminstance.LeaseView { return terminstance.LeaseView{LeaseID: lxLease, Epoch: 7} },
		CurrentGeneration: func() string { return lxGeneration }, Now: lxNow,
		Sleep: func(time.Duration) {}, PollInterval: time.Millisecond,
	}
	return &lxFixture{root: resolved, runtime: runtimeDir, socket: SocketPath(runtimeDir), data: data, runner: runner, lc: lc}
}

func captureBoundaryBody(t *testing.T, generation, proofKind string) []byte {
	t.Helper()
	context := lxContext(lxInstance + "/boundary/" + generation + "/" + proofKind)
	context["authorization"] = lxAuth("control")
	return lxBody(t, map[string]any{
		"context": context, "quiescence_generation": generation,
		"provider_proof_kind": proofKind, "timeout_ms": float64(60000),
	})
}

func recordCaptureBinding(t *testing.T, fx *lxFixture, state terminalbackend.InstanceState) {
	t.Helper()
	recordBinding(t, fx, lxDigestA)
	if err := fx.lc.States.RecordIncarnation(lxInstance, lxCreateKey()); err != nil {
		t.Fatal(err)
	}
	if state != "" {
		if err := fx.lc.States.Record(lxInstance, state); err != nil {
			t.Fatal(err)
		}
	}
}

func queuePresentStatus(fx *lxFixture, attached string) {
	fx.runner.queue("list-panes", "sh\n", 0).queue("list-sessions", lxInstance+"|"+attached+"\n", 0)
}

func queueAbsentStatus(fx *lxFixture) {
	fx.runner.queueFull("list-panes", "", "can't find window: "+lxInstance, 1)
}

func gitCaptureCommand(t *testing.T, root string, args ...string) string {
	t.Helper()
	base := []string{"-c", "user.email=test@example.com", "-c", "user.name=test", "-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main", "-c", "init.templateDir="}
	command := exec.Command("git", append(base, args...)...)
	command.Dir = root
	command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func captureAssemblyRequest(t *testing.T) (gitsnap.AssemblyRunner, gitsnap.AssemblyOptions) {
	t.Helper()
	root := t.TempDir()
	base := t.TempDir()
	return captureAssemblyRequestAt(t, root, filepath.Join(base, "home"), filepath.Join(base, "tmp"), true)
}

func captureAssemblyRequestAt(t *testing.T, root, home, temporary string, initialize bool, suppliedScratch ...string) (gitsnap.AssemblyRunner, gitsnap.AssemblyOptions) {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(temporary, 0o700); err != nil {
		t.Fatal(err)
	}
	var scratch string
	if len(suppliedScratch) > 0 {
		scratch = suppliedScratch[0]
		if err := os.MkdirAll(scratch, 0o700); err != nil {
			t.Fatal(err)
		}
	} else {
		scratch = t.TempDir()
	}
	if initialize {
		gitCaptureCommand(t, root, "init", "-q", "-b", "main", ".")
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("workspace\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("project rules\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitCaptureCommand(t, root, "add", "README.md", "AGENTS.md")
		gitCaptureCommand(t, root, "commit", "-qm", "fixture")
		gitCaptureCommand(t, root, "remote", "add", "origin", "ssh://git@github.com/relux/payments-api.git")
	}
	platform := scalar.PlatformLinux
	switch runtime.GOOS {
	case "darwin":
		platform = scalar.PlatformMacOS
	}
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{Platform: platform, HomeDir: home, TemporaryDir: temporary})
	if err != nil {
		t.Fatal(err)
	}
	store, err := localstore.OpenObjectStore(paths)
	if err != nil {
		t.Fatal(err)
	}
	member := gitsnap.WorkspaceSource{
		WorkspaceID: "01900000-0000-7000-8000-000000000003", GroupRelativePath: "repo",
		Directory: root, MaterializationPolicy: "shared_checkout",
		Content:            gitsnap.ContentOptions{Store: store},
		ProjectConfigPaths: map[string][]string{".": {"AGENTS.md"}},
	}
	snapshot, err := gitsnap.Capture(context.Background(), gitsnap.ExecGitRunner{}, root)
	if err != nil {
		t.Fatal(err)
	}
	urls := []string{}
	for _, remote := range snapshot.Remotes {
		urls = append(urls, remote.FetchURL)
		if remote.PushURL != nil {
			urls = append(urls, *remote.PushURL)
		}
	}
	sort.Strings(urls)
	urls = uniqueCaptureStrings(urls)
	groupID := "01900000-0000-7000-8000-000000000001"
	hostID := "01900000-0000-7000-8000-000000000002"
	createdAt := "2026-09-01T12:00:00.000Z"
	group := map[string]any{
		"schema": "urn:ax:schema:workspace-group", "schema_version": "1.0.0",
		"record_id": "sha256:" + strings.Repeat("0", 64), "subject_id": groupID,
		"workspace_group_id": groupID, "display_name": "capture fixture",
		"members": []any{map[string]any{
			"workspace_id": member.WorkspaceID, "kind": "git", "group_relative_path": member.GroupRelativePath,
			"repository_identity": snapshot.RepositoryIdentity, "sanitized_remote_urls": urls,
			"repo_relative_cwd":          snapshot.Worktree.CWD,
			"agent_project_config_paths": []string{"AGENTS.md"},
			"materialization_policy":     member.MaterializationPolicy,
		}},
		"created_by_host_id": hostID, "created_at": createdAt, "extensions": map[string]any{},
	}
	raw, err := json.Marshal(group)
	if err != nil {
		t.Fatal(err)
	}
	identity, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		t.Fatal(err)
	}
	group["record_id"] = identity.String()
	raw, err = json.Marshal(group)
	if err != nil {
		t.Fatal(err)
	}
	return gitsnap.ExecGitRunner{}, gitsnap.AssemblyOptions{
		ScratchRoot: scratch, GroupRecord: raw, WorkspaceGroupID: groupID,
		CreatedByHostID: hostID, CreatedAt: createdAt, Members: []gitsnap.WorkspaceSource{member},
	}
}

// captureValidRequestForState builds a request that is valid at every
// coordinator admission gate for one of the two allowed opening states.
// Quiescing goes through the production owner operations and records a real
// current-incarnation closure receipt and provider boundary; stopped carries
// no boundary operation by contract. The caller queues status observations
// after changing state for a negative state-domain row.
func captureValidRequestForState(t *testing.T, state terminalbackend.InstanceState) (*lxFixture, GitWorkspaceCaptureRequest) {
	t.Helper()
	fx := captureLifecycleFixture(t)
	runner, assembly := captureAssemblyRequest(t)
	request := GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil),
		Admitted:   fullLifecycleAdmitted(),
		Runner:     runner,
		Assembly:   assembly,
	}
	switch state {
	case terminalbackend.StateQuiescing:
		recordCaptureBinding(t, fx, terminalbackend.StateActive)
		fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
		if _, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
		}); err != nil {
			t.Fatalf("prepare current-incarnation input closure: %v", err)
		}
		request.BoundaryBody = captureBoundaryBody(t, lxQuiesce, "provider_quiescence")
		fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
		if _, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "wait-safe-boundary", Body: request.BoundaryBody, Source: "quiescing", Admitted: fullLifecycleAdmitted(),
		}); err != nil {
			t.Fatalf("prepare current-incarnation provider boundary: %v", err)
		}
		proven, err := fx.lc.States.QuiesceBarrierProven(lxInstance)
		if err != nil || !proven {
			t.Fatalf("prepared input-closure barrier proven=%v, error=%v", proven, err)
		}
	case terminalbackend.StateStopped:
		recordCaptureBinding(t, fx, terminalbackend.StateStopped)
	default:
		t.Fatalf("captureValidRequestForState called with unsupported fixture state %q", state)
	}
	return fx, request
}

func queueCaptureStateObservation(fx *lxFixture, state terminalbackend.InstanceState) {
	switch state {
	case terminalbackend.StateAbsent, terminalbackend.StateStopped,
		terminalbackend.StateStaleFenced, terminalbackend.StateUnavailable:
		queueAbsentStatus(fx)
	case terminalbackend.StateActive:
		queuePresentStatus(fx, "1")
	default:
		queuePresentStatus(fx, "0")
	}
}

func queueValidQuiescingCapture(fx *lxFixture) {
	queuePresentStatus(fx, "0")
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	queuePresentStatus(fx, "0")
}

func requireCaptureRefusal(t *testing.T, err error, code, detail string) {
	t.Helper()
	var refusal *gitsnap.Refusal
	if !errors.As(err, &refusal) || string(refusal.Code) != code || refusal.Detail != detail {
		t.Fatalf("capture refusal = %v, want literal code %q and detail %q", err, code, detail)
	}
}

func removeCaptureOutcome(t *testing.T, fx *lxFixture, key string) {
	t.Helper()
	sum := sha256.Sum256([]byte(key))
	path := filepath.Join(fx.lc.States.root, "tmuxlifecycle", "outcomes", hex.EncodeToString(sum[:])+".json")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove outcome %q: %v", key, err)
	}
}

func uniqueCaptureStrings(values []string) []string {
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func TestCaptureGitWorkspaceQuiescedAssemblesAndRechecks(t *testing.T) {
	fx := captureLifecycleFixture(t)
	recordCaptureBinding(t, fx, terminalbackend.StateActive)
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	boundaryBody := captureBoundaryBody(t, lxQuiesce, "provider_quiescence")
	runner, assembly := captureAssemblyRequest(t)
	queuePresentStatus(fx, "0")
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	queuePresentStatus(fx, "0")
	outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil), BoundaryBody: boundaryBody,
		Admitted: fullLifecycleAdmitted(), Runner: runner, Assembly: assembly,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Assembly == nil || outcome.Assembly.RootID == "" || outcome.InitialState != "quiescing" || outcome.FinalState != "quiescing" {
		t.Fatalf("capture outcome = %+v", outcome)
	}
	if _, err := scalar.ParseDigest(outcome.SafeBoundaryEvidenceID); err != nil {
		t.Fatalf("safe boundary evidence = %q: %v", outcome.SafeBoundaryEvidenceID, err)
	}
	for _, command := range fx.runner.subcommands() {
		if command == "restore" || command == "unlock-session" {
			t.Fatalf("capture reopened lifecycle input through %q", command)
		}
	}
	queuePresentStatus(fx, "0")
	queuePresentStatus(fx, "0")
	retry, err := fx.lc.CaptureGitWorkspace(context.Background(), GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil), BoundaryBody: boundaryBody,
		Admitted: fullLifecycleAdmitted(), Runner: runner, Assembly: assembly,
	})
	if err != nil {
		t.Fatalf("same-incarnation retry = %v", err)
	}
	if retry.Assembly.RootID != outcome.Assembly.RootID || retry.SafeBoundaryEvidenceID != outcome.SafeBoundaryEvidenceID {
		t.Fatalf("retry changed stable capture identities: first=(%s,%s) retry=(%s,%s)",
			outcome.Assembly.RootID, outcome.SafeBoundaryEvidenceID, retry.Assembly.RootID, retry.SafeBoundaryEvidenceID)
	}
}

func TestCaptureGitWorkspaceSourceChangeRefusalAndRetry(t *testing.T) {
	fx := captureLifecycleFixture(t)
	recordCaptureBinding(t, fx, terminalbackend.StateActive)
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	boundaryBody := captureBoundaryBody(t, lxQuiesce, "provider_quiescence")
	runner, assembly := captureAssemblyRequest(t)
	workspace := assembly.Members[0].Directory
	changed := false
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		changed = true
		if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("changed during capture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}}
	queuePresentStatus(fx, "0")
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	request := GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil), BoundaryBody: boundaryBody,
		Admitted: fullLifecycleAdmitted(), Runner: mutatingRunner, Assembly: assembly,
	}
	outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	var refusal *gitsnap.Refusal
	if outcome != nil || !changed || !errors.As(err, &refusal) || refusal.Gate != "GateConsistency" {
		t.Fatalf("mid-capture file change outcome=%+v changed=%v error=%v, want nil and GateConsistency", outcome, changed, err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("workspace\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	queuePresentStatus(fx, "0")
	queuePresentStatus(fx, "0")
	request.Runner = runner
	retry, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if err != nil || retry == nil || retry.Assembly.RootID == "" {
		t.Fatalf("capture retry after source restoration = %+v, %v", retry, err)
	}
}

func TestCaptureGitWorkspaceStoppedAssemblesWithoutBoundary(t *testing.T) {
	fx := captureLifecycleFixture(t)
	recordCaptureBinding(t, fx, terminalbackend.StateStopped)
	runner, assembly := captureAssemblyRequest(t)
	queueAbsentStatus(fx)
	queueAbsentStatus(fx)
	outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
		Runner: runner, Assembly: assembly,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.InitialState != "stopped" || outcome.FinalState != "stopped" || outcome.SafeBoundaryEvidenceID != "" {
		t.Fatalf("stopped capture outcome = %+v", outcome)
	}
}

func TestCaptureStopPointStateDomain(t *testing.T) {
	cases := []struct {
		state      terminalbackend.InstanceState
		wantDetail string
		admitted   bool
	}{
		{state: terminalbackend.StateAbsent, wantDetail: `capture is unsupported from Terminal Instance state "absent"`},
		{state: terminalbackend.StateCreating, wantDetail: `capture is unsupported from Terminal Instance state "unavailable"`},
		{state: terminalbackend.StateParked, wantDetail: `capture is unsupported from Terminal Instance state "parked"`},
		{state: terminalbackend.StateActive, wantDetail: `capture is unsupported from Terminal Instance state "active"`},
		{state: terminalbackend.StateQuiescing, admitted: true},
		{state: terminalbackend.StateStopped, admitted: true},
		{state: terminalbackend.StateStaleFenced, wantDetail: `capture is unsupported from Terminal Instance state "stale_fenced"`},
		{state: terminalbackend.StateUnavailable, wantDetail: `capture is unsupported from Terminal Instance state "unavailable"`},
	}
	for _, tc := range cases {
		t.Run(string(tc.state), func(t *testing.T) {
			requestState := terminalbackend.StateQuiescing
			if tc.admitted {
				requestState = tc.state
			}
			fx, request := captureValidRequestForState(t, requestState)
			if tc.admitted {
				if tc.state == terminalbackend.StateQuiescing {
					queueValidQuiescingCapture(fx)
				} else {
					queueAbsentStatus(fx)
					queueAbsentStatus(fx)
				}
				outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
				if err != nil || outcome == nil || outcome.Assembly == nil || outcome.Assembly.RootID == "" ||
					outcome.InitialState != tc.state || outcome.FinalState != tc.state {
					t.Fatalf("allowed %q control outcome = %+v, error = %v", tc.state, outcome, err)
				}
				if tc.state == terminalbackend.StateQuiescing && outcome.SafeBoundaryEvidenceID == "" {
					t.Fatal("quiescing control was admitted without safe-boundary evidence")
				}
				if tc.state == terminalbackend.StateStopped && outcome.SafeBoundaryEvidenceID != "" {
					t.Fatalf("stopped control unexpectedly carries boundary evidence %q", outcome.SafeBoundaryEvidenceID)
				}
				return
			}
			if tc.state == terminalbackend.StateCreating {
				recordCaptureStateFixture(t, fx, tc.state)
			} else {
				if err := fx.lc.States.Record(lxInstance, tc.state); err != nil {
					t.Fatal(err)
				}
			}
			queueCaptureStateObservation(fx, tc.state)
			_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
			requireCaptureRefusal(t, err, "capability_unavailable", tc.wantDetail)
		})
	}
}

func TestCaptureAdmissionGateValidControls(t *testing.T) {
	for _, state := range []terminalbackend.InstanceState{terminalbackend.StateQuiescing, terminalbackend.StateStopped} {
		t.Run(string(state), func(t *testing.T) {
			fx, request := captureValidRequestForState(t, state)
			if state == terminalbackend.StateQuiescing {
				queueValidQuiescingCapture(fx)
			} else {
				queueAbsentStatus(fx)
				queueAbsentStatus(fx)
			}
			outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
			if err != nil || outcome == nil || outcome.Assembly == nil || outcome.Assembly.RootID == "" ||
				outcome.InitialState != state || outcome.FinalState != state {
				t.Fatalf("valid %q control outcome = %+v, error = %v", state, outcome, err)
			}
		})
	}
}

func recordCaptureStateFixture(t *testing.T, fx *lxFixture, state terminalbackend.InstanceState) {
	t.Helper()
	// Creating is a closed status state but is intentionally not persisted by
	// Record. This exact owner-state fixture reaches the status entry while a
	// create operation is in flight; its current classifier projects it to the
	// refusal state unavailable.
	raw, err := json.Marshal(stateDocument{
		Schema: "urn:ax:schema:tmux-instance-state", SchemaVersion: "1.0.0",
		TerminalInstanceID: lxInstance, State: string(state),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fx.lc.States.instancePath(lxInstance), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCaptureProviderProofRequirements(t *testing.T) {
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	quiesce, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey())
	if err != nil || !found {
		t.Fatalf("prepared input-closure receipt found=%v, error=%v", found, err)
	}
	quiesce.InputClosedAt = ""
	if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), quiesce); err != nil {
		t.Fatal(err)
	}
	queuePresentStatus(fx, "0")
	_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "input-closure receipt has an invalid observation timestamp")

	fx, request = captureValidRequestForState(t, terminalbackend.StateQuiescing)
	request.BoundaryBody = nil
	queuePresentStatus(fx, "0")
	_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "quiescing capture requires a provider safe-boundary operation")
}

func TestCaptureFinalStatusStateDomain(t *testing.T) {
	states := []terminalbackend.InstanceState{
		terminalbackend.StateAbsent,
		terminalbackend.StateCreating,
		terminalbackend.StateParked,
		terminalbackend.StateActive,
		terminalbackend.StateQuiescing,
		terminalbackend.StateStopped,
		terminalbackend.StateStaleFenced,
		terminalbackend.StateUnavailable,
	}
	for _, finalState := range states {
		t.Run(string(finalState), func(t *testing.T) {
			fx := captureLifecycleFixture(t)
			recordCaptureBinding(t, fx, terminalbackend.StateQuiescing)
			if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), OperationOutcome{
				Operation: "quiesce-input", InputClosedAt: lxIssued, Incarnation: lxCreateKey(),
			}); err != nil {
				t.Fatal(err)
			}
			boundaryBody := captureBoundaryBody(t, lxQuiesce, "provider_quiescence")
			_, assembly := captureAssemblyRequest(t)
			changingRunner := &capturePhaseRunner{afterPack: func() {
				if finalState == terminalbackend.StateCreating {
					recordCaptureStateFixture(t, fx, finalState)
				} else if err := fx.lc.States.Record(lxInstance, finalState); err != nil {
					t.Fatal(err)
				}
			}}
			queuePresentStatus(fx, "0")
			fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
			if finalState == terminalbackend.StateAbsent || finalState == terminalbackend.StateStopped || finalState == terminalbackend.StateStaleFenced || finalState == terminalbackend.StateUnavailable {
				queueAbsentStatus(fx)
			} else {
				attached := "0"
				if finalState == terminalbackend.StateActive {
					attached = "1"
				}
				queuePresentStatus(fx, attached)
			}
			request := GitWorkspaceCaptureRequest{
				StatusBody: lxStatusBody(t, true, false, nil), BoundaryBody: boundaryBody,
				Admitted: fullLifecycleAdmitted(), Runner: changingRunner, Assembly: assembly,
			}
			outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
			allowed := finalState == terminalbackend.StateQuiescing || finalState == terminalbackend.StateStopped
			if allowed {
				if err != nil || outcome == nil || outcome.FinalState != finalState {
					t.Fatalf("closing state %q outcome = %+v, error = %v", finalState, outcome, err)
				}
				return
			}
			var refusal *gitsnap.Refusal
			if !errors.As(err, &refusal) || string(refusal.Code) != "capability_unavailable" {
				t.Fatalf("closing state %q refusal = %v, want literal capability_unavailable", finalState, err)
			}
			if !changingRunner.fired {
				t.Fatal("capture did not reach the closing status phase")
			}
		})
	}
}

type capturePhaseRunner struct {
	gitsnap.ExecGitRunner
	afterPack func()
	fired     bool
}

func (r *capturePhaseRunner) after(args []string) {
	if r.fired || len(args) == 0 || args[0] != "pack-objects" {
		return
	}
	r.fired = true
	if r.afterPack != nil {
		r.afterPack()
	}
}

func (r *capturePhaseRunner) Run(ctx context.Context, root string, args ...string) (gitsnap.GitResult, error) {
	return r.ExecGitRunner.Run(ctx, root, args...)
}

func (r *capturePhaseRunner) RunInput(ctx context.Context, root string, input []byte, args ...string) (gitsnap.GitResult, error) {
	result, err := r.ExecGitRunner.RunInput(ctx, root, input, args...)
	r.after(args)
	return result, err
}

func (r *capturePhaseRunner) RunIsolated(ctx context.Context, root string, input []byte, args ...string) (gitsnap.GitResult, error) {
	return r.ExecGitRunner.RunIsolated(ctx, root, input, args...)
}

func TestCaptureNeverTransitionsOutsideStopPointOwnerSet(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "capture_git_workspace.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var method *ast.FuncDecl
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "CaptureGitWorkspace" {
			method = function
			break
		}
	}
	if method == nil {
		t.Fatal("production capture entry is missing")
	}
	allowed := map[string]bool{
		"status": true, "quiesce-input": true, "wait-safe-boundary": true, "request-stop": true,
	}
	operations := map[string]bool{}
	ast.Inspect(method.Body, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok || !isTypeIdent(literal.Type, "OpRequest") {
			return true
		}
		if operation := captureOperationLiteral(literal); operation != "" {
			operations[operation] = true
		}
		return true
	})
	statusRequest := operations["status"]
	if !statusRequest {
		t.Fatal("capture entry has no typed status operation for opening and closing checks")
	}
	executeCalls := 0
	ast.Inspect(method.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Execute" {
			return true
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok || receiver.Name != "lc" {
			t.Errorf("capture uses an unreviewed Execute receiver %T", selector.X)
			return true
		}
		executeCalls++
		if len(call.Args) == 0 {
			t.Error("capture Execute call has no request")
			return true
		}
		var operation string
		switch request := call.Args[len(call.Args)-1].(type) {
		case *ast.Ident:
			if request.Name == "statusRequest" {
				operation = "status"
			}
		case *ast.CompositeLit:
			operation = captureOperationLiteral(request)
		}
		if operation == "" || !allowed[operation] {
			t.Errorf("capture invokes an owner transition outside {quiesce-input, wait-safe-boundary, request-stop, status}: %q", operation)
		} else if !operations[operation] {
			t.Errorf("capture calls %q without a literal operation request", operation)
		}
		return true
	})
	if executeCalls != 3 {
		t.Errorf("capture Execute call sites = %d, want two status observations and one provider boundary", executeCalls)
	}
}

func captureOperationLiteral(literal *ast.CompositeLit) string {
	if literal == nil || !isTypeIdent(literal.Type, "OpRequest") {
		return ""
	}
	for _, element := range literal.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		name, ok := field.Key.(*ast.Ident)
		if !ok || name.Name != "Operation" {
			continue
		}
		value, ok := field.Value.(*ast.BasicLit)
		if !ok || value.Kind != token.STRING {
			return ""
		}
		operation, err := strconv.Unquote(value.Value)
		if err != nil {
			return ""
		}
		return operation
	}
	return ""
}

func isTypeIdent(expression ast.Expr, name string) bool {
	ident, ok := expression.(*ast.Ident)
	return ok && ident.Name == name
}

func TestCaptureRejectsCheckpointBoundaryAsProviderProof(t *testing.T) {
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	request.BoundaryBody = captureBoundaryBody(t, lxQuiesce, "ax_checkpoint_boundary")
	queuePresentStatus(fx, "0")
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "checkpoint boundary alone does not prove provider quiescence")
}

func TestCaptureRejectsQuiesceReceiptFromAnotherIncarnation(t *testing.T) {
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	quiesce, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey())
	if err != nil || !found {
		t.Fatalf("prepared input-closure receipt found=%v, error=%v", found, err)
	}
	quiesce.Incarnation = "another-incarnation"
	if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), quiesce); err != nil {
		t.Fatal(err)
	}
	proven, err := fx.lc.States.QuiesceBarrierProven(lxInstance)
	if err != nil || !proven {
		t.Fatalf("the independent current boundary receipt no longer proves the baseline barrier: proven=%v, error=%v", proven, err)
	}
	queueValidQuiescingCapture(fx)
	_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "input-closure receipt belongs to another incarnation")
}

func TestCaptureRejectsBarrierLostAfterAssembly(t *testing.T) {
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		quiesce, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey())
		if err != nil || !found {
			t.Errorf("read input-closure receipt at close: found=%v, error=%v", found, err)
			return
		}
		quiesce.Incarnation = "superseded-incarnation"
		if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), quiesce); err != nil {
			t.Errorf("supersede input-closure receipt at close: %v", err)
			return
		}
		boundary, found, err := fx.lc.States.LookupOutcome(lxBoundaryKey("provider_quiescence"))
		if err != nil || !found {
			t.Errorf("read provider-boundary receipt at close: found=%v, error=%v", found, err)
			return
		}
		boundary.Incarnation = "superseded-incarnation"
		if err := fx.lc.States.RecordOutcome(lxBoundaryKey("provider_quiescence"), boundary); err != nil {
			t.Errorf("supersede provider-boundary receipt at close: %v", err)
		}
	}}
	request.Runner = mutatingRunner
	queueValidQuiescingCapture(fx)
	outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if outcome != nil || !mutatingRunner.fired {
		t.Fatalf("capture outcome=%+v, pack hook fired=%v; want a close-time barrier refusal", outcome, mutatingRunner.fired)
	}
	requireCaptureRefusal(t, err, "capability_unavailable", "input-closure barrier no longer proves capture hold")
}

func TestCaptureStoppedMustRemainStoppedAtClose(t *testing.T) {
	fx, request := captureValidRequestForState(t, terminalbackend.StateStopped)
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		if err := fx.lc.States.Record(lxInstance, terminalbackend.StateQuiescing); err != nil {
			t.Errorf("change closing instance state: %v", err)
		}
	}}
	request.Runner = mutatingRunner
	queueAbsentStatus(fx)
	queuePresentStatus(fx, "0")
	outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if outcome != nil || !mutatingRunner.fired {
		t.Fatalf("capture outcome=%+v, pack hook fired=%v; want stopped-state refusal", outcome, mutatingRunner.fired)
	}
	requireCaptureRefusal(t, err, "capability_unavailable", "stopped capture did not remain stopped")
}

type captureAdmissionFunction struct {
	decl    *ast.FuncDecl
	imports map[string]struct{}
}

type captureClauseKey struct {
	detail   string
	disjunct string
}

type captureRefusalClause struct {
	key      captureClauseKey
	function string
}

func TestCaptureCoordinatorAdmissionGateCensus(t *testing.T) {
	productionClauses, _ := capturePackageRefusalClauses(t, nil)
	rows, _ := captureAdmissionMatrixRows(t)
	t.Logf("reachable capture refusal clauses=%d; matrix rows=%d", len(productionClauses), len(rows))
	for _, issue := range captureAdmissionCoverageIssues(t, nil) {
		t.Error(issue)
	}
}

func TestCaptureCoordinatorClauseCensusDetectsAddedDisjunct(t *testing.T) {
	source, err := os.ReadFile("capture_git_workspace.go")
	if err != nil {
		t.Fatal(err)
	}
	needle := "if !found {\n\t\t\treturn nil, captureUnavailable(\"current-incarnation input-closure receipt is missing\")"
	planted := "if !found || quiesce.Operation == \"\" {\n\t\t\treturn nil, captureUnavailable(\"current-incarnation input-closure receipt is missing\")"
	if count := strings.Count(string(source), needle); count != 1 {
		t.Fatalf("control plant anchor occurs %d times, want one", count)
	}
	plantedSource := []byte(strings.Replace(string(source), needle, planted, 1))
	issues := captureAdmissionCoverageIssues(t, map[string][]byte{"capture_git_workspace.go": plantedSource})
	for _, issue := range issues {
		if strings.Contains(issue, `no matrix row for detail "current-incarnation input-closure receipt is missing" disjunct "quiesce.Operation == \"\""`) {
			return
		}
	}
	t.Fatalf("added OR disjunct did not create an uncovered census clause; issues=%v", issues)
}

func captureAdmissionCoverageIssues(t *testing.T, productionOverrides map[string][]byte) []string {
	t.Helper()
	production, productionIssues := capturePackageRefusalClauses(t, productionOverrides)
	rows, rowIssues := captureAdmissionMatrixRows(t)
	issues := append(productionIssues, rowIssues...)
	if len(production) == 0 {
		issues = append(issues, "CaptureGitWorkspace call graph contains no captureUnavailable sites")
	}
	productionCounts := make(map[captureClauseKey]int, len(production))
	for _, clause := range production {
		productionCounts[clause.key]++
		if productionCounts[clause.key] > 1 {
			issues = append(issues, fmt.Sprintf("production clause for detail %q disjunct %q occurs more than once", clause.key.detail, clause.key.disjunct))
		}
		if _, ok := rows[clause.key]; !ok {
			issues = append(issues, fmt.Sprintf("no matrix row for detail %q disjunct %q", clause.key.detail, clause.key.disjunct))
		}
	}
	for key, testName := range rows {
		if productionCounts[key] == 0 {
			issues = append(issues, fmt.Sprintf("matrix row %s for detail %q disjunct %q has no reachable production clause", testName, key.detail, key.disjunct))
		}
	}
	sort.Strings(issues)
	return issues
}

func capturePackageRefusalClauses(t *testing.T, sourceOverrides map[string][]byte) ([]captureRefusalClause, []string) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, []string{fmt.Sprintf("read production package directory: %v", err)}
	}
	byName := map[string][]captureAdmissionFunction{}
	var roots []captureAdmissionFunction
	fset := token.NewFileSet()
	var issues []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, overridden := sourceOverrides[name]
		if !overridden {
			source, err = os.ReadFile(name)
			if err != nil {
				issues = append(issues, fmt.Sprintf("read production Go source %s: %v", name, err))
				continue
			}
		}
		file, err := parser.ParseFile(fset, name, source, 0)
		if err != nil {
			issues = append(issues, fmt.Sprintf("parse production Go source %s: %v", name, err))
			continue
		}
		imports := make(map[string]struct{}, len(file.Imports))
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				issues = append(issues, fmt.Sprintf("parse import path in %s: %v", name, err))
				continue
			}
			alias := filepath.Base(path)
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			if alias != "_" && alias != "." {
				imports[alias] = struct{}{}
			}
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			item := captureAdmissionFunction{decl: function, imports: imports}
			byName[function.Name.Name] = append(byName[function.Name.Name], item)
			if function.Name.Name == "CaptureGitWorkspace" {
				roots = append(roots, item)
			}
		}
	}
	if len(roots) != 1 {
		return nil, append(issues, fmt.Sprintf("production source has %d CaptureGitWorkspace roots, want one", len(roots)))
	}
	var clauses []captureRefusalClause
	visited := map[token.Pos]bool{}
	pending := []captureAdmissionFunction{roots[0]}
	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		if visited[current.decl.Pos()] {
			continue
		}
		visited[current.decl.Pos()] = true
		parents := captureASTParents(current.decl.Body)
		ast.Inspect(current.decl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "captureUnavailable" {
				if len(call.Args) != 1 {
					issues = append(issues, fmt.Sprintf("%s captureUnavailable site has %d args, want one literal detail", current.decl.Name.Name, len(call.Args)))
				} else if literal, ok := call.Args[0].(*ast.BasicLit); !ok || literal.Kind != token.STRING {
					issues = append(issues, fmt.Sprintf("%s captureUnavailable site has non-literal detail %T", current.decl.Name.Name, call.Args[0]))
				} else if detail, err := strconv.Unquote(literal.Value); err != nil {
					issues = append(issues, fmt.Sprintf("%s captureUnavailable detail %q is invalid: %v", current.decl.Name.Name, literal.Value, err))
				} else if guard := captureNearestIf(call, parents); guard == nil {
					issues = append(issues, fmt.Sprintf("%s captureUnavailable detail %q has no if guard", current.decl.Name.Name, detail))
				} else {
					for _, disjunct := range captureORDisjuncts(guard.Cond, fset) {
						clauses = append(clauses, captureRefusalClause{
							key: captureClauseKey{detail: detail, disjunct: disjunct}, function: current.decl.Name.Name,
						})
					}
				}
			}
			for _, name := range captureCallNames(call, current.imports) {
				pending = append(pending, byName[name]...)
			}
			return true
		})
	}
	return clauses, issues
}

func captureASTParents(root ast.Node) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

func captureNearestIf(node ast.Node, parents map[ast.Node]ast.Node) *ast.IfStmt {
	for parent := parents[node]; parent != nil; parent = parents[parent] {
		if guard, ok := parent.(*ast.IfStmt); ok {
			return guard
		}
	}
	return nil
}

func captureORDisjuncts(expression ast.Expr, fset *token.FileSet) []string {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parenthesized.X
	}
	if binary, ok := expression.(*ast.BinaryExpr); ok && binary.Op == token.LOR {
		return append(captureORDisjuncts(binary.X, fset), captureORDisjuncts(binary.Y, fset)...)
	}
	var source bytes.Buffer
	if err := format.Node(&source, fset, expression); err != nil {
		return []string{fmt.Sprintf("<unprintable %T>", expression)}
	}
	return []string{source.String()}
}

func captureCallNames(call *ast.CallExpr, imports map[string]struct{}) []string {
	expression := call.Fun
	for {
		switch indexed := expression.(type) {
		case *ast.IndexExpr:
			expression = indexed.X
		case *ast.IndexListExpr:
			expression = indexed.X
		default:
			goto resolved
		}
	}
resolved:
	switch function := expression.(type) {
	case *ast.Ident:
		return []string{function.Name}
	case *ast.SelectorExpr:
		if receiver, ok := function.X.(*ast.Ident); ok {
			if _, imported := imports[receiver.Name]; imported {
				return nil
			}
		}
		return []string{function.Sel.Name}
	default:
		return nil
	}
}

func captureAdmissionMatrixRows(t *testing.T) (map[captureClauseKey]string, []string) {
	t.Helper()
	testFile, err := parser.ParseFile(token.NewFileSet(), "capture_git_workspace_test.go", nil, 0)
	if err != nil {
		return nil, []string{fmt.Sprintf("parse capture matrix test source: %v", err)}
	}
	rows := map[captureClauseKey]string{}
	var issues []string
	for _, declaration := range testFile.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil || !strings.HasPrefix(function.Name.Name, "TestCaptureAdmission_") {
			continue
		}
		entries := 0
		assertions := 0
		metadata := 0
		var metadataKey captureClauseKey
		var assertionDetail string
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "CaptureGitWorkspace" {
				entries++
			}
			ident, ok := call.Fun.(*ast.Ident)
			if ok && ident.Name == "captureAdmissionClause" {
				metadata++
				if len(call.Args) != 3 {
					issues = append(issues, fmt.Sprintf("matrix test %s clause metadata has %d args, want t, detail, and disjunct literals", function.Name.Name, len(call.Args)))
					return true
				}
				detail, detailOK := captureLiteralString(call.Args[1])
				disjunct, disjunctOK := captureLiteralString(call.Args[2])
				if !detailOK || !disjunctOK {
					issues = append(issues, fmt.Sprintf("matrix test %s clause metadata must use literal detail and disjunct", function.Name.Name))
				} else {
					metadataKey = captureClauseKey{detail: detail, disjunct: disjunct}
				}
				return true
			}
			if !ok || ident.Name != "requireCaptureRefusal" {
				return true
			}
			assertions++
			if len(call.Args) != 4 {
				t.Errorf("matrix test %s has a refusal assertion with %d args, want four", function.Name.Name, len(call.Args))
				return true
			}
			code, codeOK := captureLiteralString(call.Args[2])
			detail, detailOK := captureLiteralString(call.Args[3])
			if !codeOK || code != "capability_unavailable" || !detailOK {
				issues = append(issues, fmt.Sprintf("matrix test %s must assert literal capability_unavailable code and literal detail", function.Name.Name))
				return true
			}
			assertionDetail = detail
			return true
		})
		if entries != 1 || assertions != 1 || metadata != 1 {
			issues = append(issues, fmt.Sprintf("matrix test %s has %d production entry calls, %d literal refusal assertions, and %d clause rows; want one each", function.Name.Name, entries, assertions, metadata))
		}
		if metadata == 1 && assertions == 1 {
			if metadataKey.detail != assertionDetail {
				issues = append(issues, fmt.Sprintf("matrix test %s clause detail %q differs from refusal assertion %q", function.Name.Name, metadataKey.detail, assertionDetail))
			}
			if prior, exists := rows[metadataKey]; exists {
				issues = append(issues, fmt.Sprintf("matrix clause detail %q disjunct %q is claimed by both %s and %s", metadataKey.detail, metadataKey.disjunct, prior, function.Name.Name))
			} else {
				rows[metadataKey] = function.Name.Name
			}
		}
	}
	if len(rows) == 0 {
		issues = append(issues, "no TestCaptureAdmission_ matrix rows found")
	}
	return rows, issues
}

func captureAdmissionClause(t *testing.T, detail, disjunct string) {
	t.Helper()
	if detail == "" || disjunct == "" {
		t.Fatal("capture admission clause metadata must be literal and non-empty")
	}
}

func captureLiteralString(expression ast.Expr) (string, bool) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func captureInitialStateFixture(t *testing.T, state terminalbackend.InstanceState) (*lxFixture, GitWorkspaceCaptureRequest) {
	t.Helper()
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	if state == terminalbackend.StateCreating {
		recordCaptureStateFixture(t, fx, state)
	} else if err := fx.lc.States.Record(lxInstance, state); err != nil {
		t.Fatal(err)
	}
	queueCaptureStateObservation(fx, state)
	return fx, request
}

func captureBoundaryBodyWithContextMember(t *testing.T, raw []byte, member, value string) []byte {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	context, ok := body["context"].(map[string]any)
	if !ok {
		t.Fatal("boundary fixture has no context object")
	}
	context[member] = value
	updated, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func TestCaptureAdmission_LifecycleRequired(t *testing.T) {
	captureAdmissionClause(t, "lifecycle is required", "lc == nil")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	fx.lc = nil
	_, err := (*Lifecycle)(nil).CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "lifecycle is required")
}

func TestCaptureAdmission_ExactStatusIdentityRequired(t *testing.T) {
	captureAdmissionClause(t, "capture requires an exact Terminal Instance identity", "!statusBody.HasTerminalInstanceID")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	request.StatusBody = lxStatusBody(t, false, false, nil)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "capture requires an exact Terminal Instance identity")
}

func TestCaptureAdmission_OpeningStatusIdentityMatch(t *testing.T) {
	captureAdmissionClause(t, "exact Terminal Instance status did not match", "!initial.Status.IdentityMatch")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	request.StatusBody = lxStatusBody(t, true, false, func(body map[string]any) {
		body["backend_generation"] = "generation-two"
	})
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "exact Terminal Instance status did not match")
}

func TestCaptureAdmission_InitialAbsent(t *testing.T) {
	captureAdmissionClause(t, `capture is unsupported from Terminal Instance state "absent"`, "!captureStateAllowed(initial.Status.State)")
	fx, request := captureInitialStateFixture(t, terminalbackend.StateAbsent)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", `capture is unsupported from Terminal Instance state "absent"`)
}

func TestCaptureAdmission_InitialParked(t *testing.T) {
	captureAdmissionClause(t, `capture is unsupported from Terminal Instance state "parked"`, "!captureStateAllowed(initial.Status.State)")
	fx, request := captureInitialStateFixture(t, terminalbackend.StateParked)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", `capture is unsupported from Terminal Instance state "parked"`)
}

func TestCaptureAdmission_InitialActive(t *testing.T) {
	captureAdmissionClause(t, `capture is unsupported from Terminal Instance state "active"`, "!captureStateAllowed(initial.Status.State)")
	fx, request := captureInitialStateFixture(t, terminalbackend.StateActive)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", `capture is unsupported from Terminal Instance state "active"`)
}

func TestCaptureAdmission_InitialStaleFenced(t *testing.T) {
	captureAdmissionClause(t, `capture is unsupported from Terminal Instance state "stale_fenced"`, "!captureStateAllowed(initial.Status.State)")
	fx, request := captureInitialStateFixture(t, terminalbackend.StateStaleFenced)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", `capture is unsupported from Terminal Instance state "stale_fenced"`)
}

func TestCaptureAdmission_InitialUnavailable(t *testing.T) {
	captureAdmissionClause(t, `capture is unsupported from Terminal Instance state "unavailable"`, "!captureStateAllowed(initial.Status.State)")
	fx, request := captureInitialStateFixture(t, terminalbackend.StateUnavailable)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", `capture is unsupported from Terminal Instance state "unavailable"`)
}

func TestCaptureAdmission_AssemblyRunnerRequired(t *testing.T) {
	captureAdmissionClause(t, "Git assembly runner is required", "request.Runner == nil")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	request.Runner = nil
	queuePresentStatus(fx, "0")
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "Git assembly runner is required")
}

func TestCaptureAdmission_StoppedBoundaryBodyForbidden(t *testing.T) {
	captureAdmissionClause(t, "stopped capture must not carry a boundary operation", "len(request.BoundaryBody) != 0")
	fx, request := captureValidRequestForState(t, terminalbackend.StateStopped)
	request.BoundaryBody = captureBoundaryBody(t, lxQuiesce, "provider_quiescence")
	queueAbsentStatus(fx)
	queueAbsentStatus(fx)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "stopped capture must not carry a boundary operation")
}

func TestCaptureAdmission_StoppedIncarnationRequired(t *testing.T) {
	captureAdmissionClause(t, "stopped capture has no current incarnation record", "!found")
	fx, request := captureValidRequestForState(t, terminalbackend.StateStopped)
	path := filepath.Join(fx.lc.States.root, "tmuxlifecycle", "incarnation", lxInstance+".json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	queueAbsentStatus(fx)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "stopped capture has no current incarnation record")
}

func TestCaptureAdmission_QuiescingBoundaryRequired(t *testing.T) {
	captureAdmissionClause(t, "quiescing capture requires a provider safe-boundary operation", "len(request.BoundaryBody) == 0")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	request.BoundaryBody = nil
	queuePresentStatus(fx, "0")
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "quiescing capture requires a provider safe-boundary operation")
}

func TestCaptureAdmission_BoundaryIdentityMatchesStatus(t *testing.T) {
	captureAdmissionClause(t, "boundary operation identity differs from exact status", "!sameCaptureIdentity(statusBody, boundaryContext)")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	request.BoundaryBody = captureBoundaryBodyWithContextMember(t, request.BoundaryBody, "session_id", lxSessionB)
	queuePresentStatus(fx, "0")
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "boundary operation identity differs from exact status")
}

func TestCaptureAdmission_CheckpointOnlyProviderProof(t *testing.T) {
	captureAdmissionClause(t, "checkpoint boundary alone does not prove provider quiescence", `boundaryParams.ProviderProofKind != "provider_quiescence" && boundaryParams.ProviderProofKind != "provider_process_exit"`)
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	request.BoundaryBody = captureBoundaryBody(t, lxQuiesce, "ax_checkpoint_boundary")
	queuePresentStatus(fx, "0")
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "checkpoint boundary alone does not prove provider quiescence")
}

func TestCaptureAdmission_QuiescingIncarnationRequired(t *testing.T) {
	captureAdmissionClause(t, "quiescing capture has no current incarnation record", "!found")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	path := filepath.Join(fx.lc.States.root, "tmuxlifecycle", "incarnation", lxInstance+".json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	queuePresentStatus(fx, "0")
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "quiescing capture has no current incarnation record")
}

func TestCaptureAdmission_CurrentInputClosureReceiptRequired(t *testing.T) {
	captureAdmissionClause(t, "input-closure receipt belongs to another incarnation", "quiesce.Incarnation != incarnation")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	quiesce, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey())
	if err != nil || !found {
		t.Fatalf("prepared input-closure receipt found=%v, error=%v", found, err)
	}
	quiesce.Incarnation = lxQuiesceKey()
	if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), quiesce); err != nil {
		t.Fatal(err)
	}
	queueValidQuiescingCapture(fx)
	_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "input-closure receipt belongs to another incarnation")
}

func TestCaptureAdmission_InputClosureReceiptMissing(t *testing.T) {
	captureAdmissionClause(t, "current-incarnation input-closure receipt is missing", "!found")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	removeCaptureOutcome(t, fx, lxQuiesceKey())
	queueValidQuiescingCapture(fx)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "current-incarnation input-closure receipt is missing")
}

func TestCaptureAdmission_InputClosureReceiptOperation(t *testing.T) {
	captureAdmissionClause(t, "input-closure receipt has the wrong operation", `quiesce.Operation != "quiesce-input"`)
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	quiesce, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey())
	if err != nil || !found {
		t.Fatalf("prepared input-closure receipt found=%v, error=%v", found, err)
	}
	quiesce.Operation = "wait-safe-boundary"
	if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), quiesce); err != nil {
		t.Fatal(err)
	}
	queueValidQuiescingCapture(fx)
	_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "input-closure receipt has the wrong operation")
}

func TestCaptureAdmission_InputClosureReceiptTimestamp(t *testing.T) {
	captureAdmissionClause(t, "input-closure receipt has an invalid observation timestamp", "err != nil")
	for _, timestamp := range []string{"", "not-a-timestamp"} {
		t.Run(fmt.Sprintf("timestamp=%q", timestamp), func(t *testing.T) {
			fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
			quiesce, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey())
			if err != nil || !found {
				t.Fatalf("prepared input-closure receipt found=%v, error=%v", found, err)
			}
			quiesce.InputClosedAt = timestamp
			if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), quiesce); err != nil {
				t.Fatal(err)
			}
			queueValidQuiescingCapture(fx)
			_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
			requireCaptureRefusal(t, err, "capability_unavailable", "input-closure receipt has an invalid observation timestamp")
		})
	}
}

func TestCaptureAdmission_CurrentProviderBoundaryReceiptRequired(t *testing.T) {
	captureAdmissionClause(t, "provider-boundary receipt belongs to another incarnation", "boundaryReceipt.Incarnation != incarnation")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	boundary, found, err := fx.lc.States.LookupOutcome(lxBoundaryKey("provider_quiescence"))
	if err != nil || !found {
		t.Fatalf("prepared provider-boundary receipt found=%v, error=%v", found, err)
	}
	boundary.Incarnation = lxQuiesceKey()
	if err := fx.lc.States.RecordOutcome(lxBoundaryKey("provider_quiescence"), boundary); err != nil {
		t.Fatal(err)
	}
	queueValidQuiescingCapture(fx)
	_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "provider-boundary receipt belongs to another incarnation")
}

func TestCaptureBoundaryReceiptIsCreatedWhenAbsent(t *testing.T) {
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	removeCaptureOutcome(t, fx, lxBoundaryKey("provider_quiescence"))
	queueValidQuiescingCapture(fx)
	outcome, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if err != nil || outcome == nil || outcome.SafeBoundaryEvidenceID == "" {
		t.Fatalf("capture without a prior boundary receipt = %+v, %v", outcome, err)
	}
}

func TestCaptureAdmission_ProviderBoundaryReceiptOperation(t *testing.T) {
	captureAdmissionClause(t, "provider-boundary receipt has the wrong operation", `boundaryReceipt.Operation != "wait-safe-boundary"`)
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	boundary, found, err := fx.lc.States.LookupOutcome(lxBoundaryKey("provider_quiescence"))
	if err != nil || !found {
		t.Fatalf("prepared provider-boundary receipt found=%v, error=%v", found, err)
	}
	boundary.Operation = "quiesce-input"
	if err := fx.lc.States.RecordOutcome(lxBoundaryKey("provider_quiescence"), boundary); err != nil {
		t.Fatal(err)
	}
	queueValidQuiescingCapture(fx)
	_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireCaptureRefusal(t, err, "capability_unavailable", "provider-boundary receipt has the wrong operation")
}

func TestCapturePropagatesOwnerBoundaryTimestampIntegrity(t *testing.T) {
	assertCaptureProviderBoundaryTimestampIntegrity(t)
}

func TestCaptureOwnerBoundaryEmptyTimestampRepeatOf(t *testing.T) {
	assertCaptureProviderBoundaryTimestampIntegrity(t)
}

func assertCaptureProviderBoundaryTimestampIntegrity(t *testing.T) {
	t.Helper()
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	boundary, found, err := fx.lc.States.LookupOutcome(lxBoundaryKey("provider_quiescence"))
	if err != nil || !found {
		t.Fatalf("prepared provider-boundary receipt found=%v, error=%v", found, err)
	}
	boundary.BoundaryObservedAt = ""
	if err := fx.lc.States.RecordOutcome(lxBoundaryKey("provider_quiescence"), boundary); err != nil {
		t.Fatal(err)
	}
	queueValidQuiescingCapture(fx)
	_, err = fx.lc.CaptureGitWorkspace(context.Background(), request)
	requireLocalCode(t, err, "terminal_backend_integrity_failure", "boundary outcome image")
}

func TestCaptureAdmission_ClosingStatusIdentityMatch(t *testing.T) {
	captureAdmissionClause(t, "closing Terminal Instance status did not match", "!final.Status.IdentityMatch")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		updated := []byte(strings.Replace(string(request.StatusBody), lxGeneration, "generation-two", 1))
		if len(updated) != len(request.StatusBody) {
			t.Errorf("closing status mutation changed body length")
			return
		}
		copy(request.StatusBody, updated)
	}}
	request.Runner = mutatingRunner
	queueValidQuiescingCapture(fx)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if !mutatingRunner.fired {
		t.Fatal("capture did not reach the closing status phase")
	}
	requireCaptureRefusal(t, err, "capability_unavailable", "closing Terminal Instance status did not match")
}

func TestCaptureAdmission_IncarnationStableAcrossCapture(t *testing.T) {
	captureAdmissionClause(t, "Terminal Instance incarnation changed during capture", "found && currentIncarnation != incarnation")
	// Keep the independent quiescence-barrier check out of this row so the
	// closing incarnation gate is the only changed input.
	fx, request := captureValidRequestForState(t, terminalbackend.StateStopped)
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		if err := fx.lc.States.RecordIncarnation(lxInstance, lxQuiesceKey()); err != nil {
			t.Errorf("change closing incarnation: %v", err)
		}
	}}
	request.Runner = mutatingRunner
	queueAbsentStatus(fx)
	queueAbsentStatus(fx)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if !mutatingRunner.fired {
		t.Fatal("capture did not reach the closing incarnation check")
	}
	requireCaptureRefusal(t, err, "capability_unavailable", "Terminal Instance incarnation changed during capture")
}

func TestCaptureAdmission_IncarnationRecordPresentAtClose(t *testing.T) {
	captureAdmissionClause(t, "Terminal Instance incarnation record disappeared during capture", "!found")
	fx, request := captureValidRequestForState(t, terminalbackend.StateStopped)
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		path := filepath.Join(fx.lc.States.root, "tmuxlifecycle", "incarnation", lxInstance+".json")
		if err := os.Remove(path); err != nil {
			t.Errorf("remove closing incarnation record: %v", err)
		}
	}}
	request.Runner = mutatingRunner
	queueAbsentStatus(fx)
	queueAbsentStatus(fx)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if !mutatingRunner.fired {
		t.Fatal("capture did not reach the closing incarnation check")
	}
	requireCaptureRefusal(t, err, "capability_unavailable", "Terminal Instance incarnation record disappeared during capture")
}

func TestCaptureAdmission_FinalStateRemainsHeld(t *testing.T) {
	captureAdmissionClause(t, "held capture left quiescing or stopped state", "!captureStateAllowed(final.Status.State)")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		if err := fx.lc.States.Record(lxInstance, terminalbackend.StateParked); err != nil {
			t.Errorf("change closing state: %v", err)
		}
	}}
	request.Runner = mutatingRunner
	queueValidQuiescingCapture(fx)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if !mutatingRunner.fired {
		t.Fatal("capture did not reach the closing state check")
	}
	requireCaptureRefusal(t, err, "capability_unavailable", "held capture left quiescing or stopped state")
}

func TestCaptureAdmission_StoppedRemainsStopped(t *testing.T) {
	captureAdmissionClause(t, "stopped capture did not remain stopped", "final.Status.State != terminalbackend.StateStopped")
	fx, request := captureValidRequestForState(t, terminalbackend.StateStopped)
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		if err := fx.lc.States.Record(lxInstance, terminalbackend.StateQuiescing); err != nil {
			t.Errorf("change closing state: %v", err)
		}
	}}
	request.Runner = mutatingRunner
	queueAbsentStatus(fx)
	queuePresentStatus(fx, "0")
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if !mutatingRunner.fired {
		t.Fatal("capture did not reach the stopped-state closing check")
	}
	requireCaptureRefusal(t, err, "capability_unavailable", "stopped capture did not remain stopped")
}

func TestCaptureAdmission_ClosureBarrierStillProven(t *testing.T) {
	captureAdmissionClause(t, "input-closure barrier no longer proves capture hold", "!proven")
	fx, request := captureValidRequestForState(t, terminalbackend.StateQuiescing)
	mutatingRunner := &capturePhaseRunner{afterPack: func() {
		quiesce, found, err := fx.lc.States.LookupOutcome(lxQuiesceKey())
		if err != nil || !found {
			t.Errorf("read input-closure receipt at close: found=%v, error=%v", found, err)
			return
		}
		quiesce.Incarnation = lxQuiesceKey()
		if err := fx.lc.States.RecordOutcome(lxQuiesceKey(), quiesce); err != nil {
			t.Errorf("supersede input-closure receipt at close: %v", err)
			return
		}
		boundary, found, err := fx.lc.States.LookupOutcome(lxBoundaryKey("provider_quiescence"))
		if err != nil || !found {
			t.Errorf("read provider-boundary receipt at close: found=%v, error=%v", found, err)
			return
		}
		boundary.Incarnation = lxQuiesceKey()
		if err := fx.lc.States.RecordOutcome(lxBoundaryKey("provider_quiescence"), boundary); err != nil {
			t.Errorf("supersede provider-boundary receipt at close: %v", err)
		}
	}}
	request.Runner = mutatingRunner
	queueValidQuiescingCapture(fx)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), request)
	if !mutatingRunner.fired {
		t.Fatal("capture did not reach the closing barrier check")
	}
	requireCaptureRefusal(t, err, "capability_unavailable", "input-closure barrier no longer proves capture hold")
}

func TestCaptureRejectsParkedEvenWithPriorQuiescenceReceipt(t *testing.T) {
	fx := captureLifecycleFixture(t)
	recordCaptureBinding(t, fx, terminalbackend.StateActive)
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	boundaryBody := captureBoundaryBody(t, lxQuiesce, "provider_quiescence")
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: boundaryBody, Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := fx.lc.States.Record(lxInstance, terminalbackend.StateParked); err != nil {
		t.Fatal(err)
	}
	queuePresentStatus(fx, "0")
	queuePresentStatus(fx, "0")
	runner, assembly := captureAssemblyRequest(t)
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil), BoundaryBody: boundaryBody,
		Admitted: fullLifecycleAdmitted(), Runner: runner, Assembly: assembly,
	})
	var refusal *gitsnap.Refusal
	if !errors.As(err, &refusal) || string(refusal.Code) != "capability_unavailable" {
		t.Fatalf("parked source refusal = %v, want literal capability_unavailable", err)
	}
}

type captureFailureRunner struct{ gitsnap.ExecGitRunner }

func (captureFailureRunner) RunInput(_ context.Context, _ string, _ []byte, args ...string) (gitsnap.GitResult, error) {
	if len(args) > 0 && args[0] == "pack-objects" {
		return gitsnap.GitResult{ExitCode: -1}, errors.New("injected capture interruption")
	}
	return gitsnap.ExecGitRunner{}.RunInput(context.Background(), ".", nil, args...)
}

func TestCaptureFailureRetryUsesRequestStop(t *testing.T) {
	fx := captureLifecycleFixture(t)
	recordCaptureBinding(t, fx, terminalbackend.StateActive)
	fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	boundaryBody := captureBoundaryBody(t, lxQuiesce, "provider_quiescence")
	runner, assembly := captureAssemblyRequest(t)
	queuePresentStatus(fx, "0")
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	failed := captureFailureRunner{}
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil), BoundaryBody: boundaryBody,
		Admitted: fullLifecycleAdmitted(), Runner: failed, Assembly: assembly,
	})
	var assemblyRefusal *gitsnap.Refusal
	if !errors.As(err, &assemblyRefusal) || assemblyRefusal.Gate != "GateContentRead" {
		t.Fatalf("interrupted assembly = %v, want typed GateContentRead refusal", err)
	}
	proof, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "wait-safe-boundary", Body: boundaryBody, Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	})
	if err != nil {
		t.Fatal(err)
	}
	stopBody := lxStopBody(t, 60000, func(object map[string]any) {
		object["safe_boundary_evidence_id"] = proof.SafeBoundaryEvidenceID
		context := object["context"].(map[string]any)
		context["idempotency_key"] = lxInstance + "/stop/" + proof.SafeBoundaryEvidenceID
	})
	fx.runner.queue("send-keys", "", 0).queue("has-session", "", 0).queue("kill-session", "", 0).
		queueFull("has-session", "", "can't find session: "+lxInstance, 1)
	if _, err := fx.lc.Execute(context.Background(), OpRequest{
		Operation: "request-stop", Body: stopBody, Source: "quiescing", Admitted: fullLifecycleAdmitted(),
	}); err != nil {
		t.Fatal(err)
	}
	queueAbsentStatus(fx)
	queueAbsentStatus(fx)
	result, err := fx.lc.CaptureGitWorkspace(context.Background(), GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
		Runner: runner, Assembly: assembly,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalState != "stopped" || result.Assembly.RootID == "" {
		t.Fatalf("retry outcome = %+v", result)
	}
}

func TestCaptureProcessCrashRetry(t *testing.T) {
	if os.Getenv("AX_CAPTURE_CRASH_CHILD") == "1" {
		captureCrashChild(t)
		return
	}
	for _, phase := range []string{"pack", "close-status"} {
		t.Run(phase, func(t *testing.T) {
			fx := captureLifecycleFixture(t)
			recordCaptureBinding(t, fx, terminalbackend.StateActive)
			fx.runner.queue("lock-session", "", 0).queue("detach-client", "", 0)
			if _, err := fx.lc.Execute(context.Background(), OpRequest{
				Operation: "quiesce-input", Body: lxQuiesceBody(t, nil), Source: "active", Admitted: fullLifecycleAdmitted(),
			}); err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(fx.data, "workspace")
			home := filepath.Join(fx.data, "object-home")
			temporary := filepath.Join(fx.data, "object-temporary")
			scratch := filepath.Join(fx.data, "assembly-scratch")
			captureAssemblyRequestAt(t, workspace, home, temporary, true, scratch)

			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(executable, "-test.run=^TestCaptureProcessCrashRetry$", "-test.count=1")
			command.Env = append(os.Environ(),
				"AX_CAPTURE_CRASH_CHILD=1", "AX_CAPTURE_CRASH_PHASE="+phase,
				"AX_CAPTURE_ROOT="+fx.root, "AX_CAPTURE_DATA="+fx.data,
				"AX_CAPTURE_WORKSPACE="+workspace, "AX_CAPTURE_HOME="+home,
				"AX_CAPTURE_TEMPORARY="+temporary,
			)
			output, childErr := command.CombinedOutput()
			exit, ok := childErr.(*exec.ExitError)
			wantExit := 74
			if phase == "close-status" {
				wantExit = 75
			}
			if !ok || exit.ExitCode() != wantExit {
				t.Fatalf("capture child %q exit = %v, output=%s; want %d", phase, childErr, output, wantExit)
			}
			t.Logf("child capture exit %d is the expected crash at phase %s", exit.ExitCode(), phase)

			boundary, found, err := fx.lc.States.LookupOutcome(lxBoundaryKey("provider_quiescence"))
			if err != nil || !found || boundary.Operation != "wait-safe-boundary" || boundary.BoundaryObservedAt == "" || boundary.Incarnation != lxCreateKey() {
				t.Fatalf("durable boundary receipt after %s crash = %+v, found=%v, error=%v", phase, boundary, found, err)
			}
			retryFX := captureLifecycleFixtureAt(t, fx.root, fx.data)
			queuePresentStatus(retryFX, "0")
			queuePresentStatus(retryFX, "0")
			_, assembly := captureAssemblyRequestAt(t, workspace, home, temporary, false, scratch)
			request := GitWorkspaceCaptureRequest{
				StatusBody: lxStatusBody(t, true, false, nil), BoundaryBody: captureBoundaryBody(t, lxQuiesce, "provider_quiescence"),
				Admitted: fullLifecycleAdmitted(), Runner: gitsnap.ExecGitRunner{}, Assembly: assembly,
			}
			outcome, err := retryFX.lc.CaptureGitWorkspace(context.Background(), request)
			if err != nil || outcome == nil || outcome.Assembly == nil || outcome.Assembly.RootID == "" || outcome.FinalState != terminalbackend.StateQuiescing {
				t.Fatalf("capture after %s crash = %+v, %v", phase, outcome, err)
			}
		})
	}
}

func captureCrashChild(t *testing.T) {
	t.Helper()
	fx := captureLifecycleFixtureAt(t, os.Getenv("AX_CAPTURE_ROOT"), os.Getenv("AX_CAPTURE_DATA"))
	_, assembly := captureAssemblyRequestAt(t, os.Getenv("AX_CAPTURE_WORKSPACE"), os.Getenv("AX_CAPTURE_HOME"), os.Getenv("AX_CAPTURE_TEMPORARY"), false, filepath.Join(os.Getenv("AX_CAPTURE_DATA"), "assembly-scratch"))
	queuePresentStatus(fx, "0")
	boundaryBody := captureBoundaryBody(t, lxQuiesce, "provider_quiescence")
	fx.runner.queue("wait-for", "", 0).queue("list-panes", "sh\n", 0)
	phase := os.Getenv("AX_CAPTURE_CRASH_PHASE")
	assemblyRunner := &capturePhaseRunner{}
	switch phase {
	case "pack":
		assemblyRunner.afterPack = func() { os.Exit(74) }
	case "close-status":
		sessionQueries := 0
		fx.runner.onRun = func(argv []string) {
			if len(argv) >= 4 && argv[3] == "list-sessions" {
				sessionQueries++
				if sessionQueries == 2 {
					os.Exit(75)
				}
			}
		}
		queuePresentStatus(fx, "0")
	default:
		t.Fatalf("unknown capture crash phase %q", phase)
	}
	_, err := fx.lc.CaptureGitWorkspace(context.Background(), GitWorkspaceCaptureRequest{
		StatusBody: lxStatusBody(t, true, false, nil), BoundaryBody: boundaryBody,
		Admitted: fullLifecycleAdmitted(), Runner: assemblyRunner, Assembly: assembly,
	})
	t.Fatalf("crash injection did not terminate capture: %v", err)
}
