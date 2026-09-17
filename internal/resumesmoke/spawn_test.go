package resumesmoke

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/provhost"
)

// This file proves the smoke runs through a real provider process:
// the fake adapter is the test binary re-executed as the provider
// executable under the production ExecRunner, answering canned
// bodies the parent built with the shared fixtures.

// TestSmokeThroughRealProviderProcess runs one available row end to
// end with the adapter as a real child process per operation. The
// verdict, checks, and record integrity must equal the in-process
// run exactly.
func TestSmokeThroughRealProviderProcess(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "linux", Architecture: "amd64"}
	runner := newScriptedRunner(t)
	runner.scriptProbe(tuple)
	runner.scriptIdentify(t, tuple, "exact")
	probeBody := captureCannedBody(t, runner, "probe", smokeProbeRequest(t, tuple))
	identifyBody := captureCannedBody(t, runner, "identify-session", smokeIdentifyRequest(t))
	directory := t.TempDir()
	probePath := writeHelperBody(t, directory, "probe.json", probeBody)
	identifyPath := writeHelperBody(t, directory, "identify.json", identifyBody)
	// The resume script asserts the threaded identity; capture the
	// plan from a resume-shaped request carrying it.
	runner.scriptResume(t, tuple, "standard")
	resumeBody := captureCannedBody(t, runner, "resume", smokeResumeRequest(t, runner.identity))
	resumePath := writeHelperBody(t, directory, "resume.json", resumeBody)

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	environment := append(os.Environ(),
		"RESUMESMOKE_HELPER=fake-provider",
		"RESUMESMOKE_PROBE="+probePath,
		"RESUMESMOKE_IDENTIFY="+identifyPath,
		"RESUMESMOKE_RESUME="+resumePath,
	)
	clock, err := time.Parse(time.RFC3339, smokeClock)
	if err != nil {
		t.Fatalf("parse smoke clock: %v", err)
	}
	host := provhost.Host{Runner: provhost.ExecRunner{Env: environment}, Now: func() time.Time { return clock }}
	params := smokeParams(tuple, runner)
	params.Host = host
	params.Executable = executable
	params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	assertVerdict(t, report.Record, VerdictPass)
	assertOutcomes(t, report.Record, map[string]Outcome{
		"resume-cell":   OutcomePass,
		"tuple-gate":    OutcomePass,
		"probe":         OutcomePass,
		"identify":      OutcomePass,
		"discover-bind": OutcomePass,
		"quiescence":    OutcomeSkipped,
		"resume-plan":   OutcomePass,
	})
	assertNoCanary(t, report.Bytes)
	if _, err := VerifyRecord(report.Bytes); err != nil {
		t.Fatalf("VerifyRecord(process bytes) = %v", err)
	}
	// The process run replays the in-process bytes exactly: the
	// transport carries no identity of its own.
	inProcess := runRowFixture(t, tuple)
	if string(inProcess.Bytes) != string(report.Bytes) {
		t.Fatal("process run bytes differ from the in-process run")
	}
}

// captureCannedBody runs one scripted handler over a fixture
// request body and returns the canned response body.
func captureCannedBody(t *testing.T, runner *scriptedRunner, operation string, request json.RawMessage) json.RawMessage {
	t.Helper()
	handler, ok := runner.script[operation]
	if !ok {
		t.Fatalf("no script for %q", operation)
	}
	return handler(request)
}

func writeHelperBody(t *testing.T, directory, name string, body json.RawMessage) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write helper body: %v", err)
	}
	return path
}

func smokeProbeRequest(t *testing.T, tuple provhost.BuildTuple) json.RawMessage {
	t.Helper()
	return json.RawMessage(`{"platform": "` + tuple.Platform + `", "architecture": "` + tuple.Architecture + `", "provider_executable": null, "requested_capabilities": ["native_resume"]}`)
}

func smokeIdentifyRequest(t *testing.T) json.RawMessage {
	t.Helper()
	return json.RawMessage(`{"session_id": "` + smokeSessionID + `", "provider_id": "codex", "observation": {"terminal_id": "smoke-terminal", "executable_path": "/smoke/bin/codex", "started_at": "` + smokeCreatedAt + `", "candidate_store_paths": [], "candidate_native_ids": []}}`)
}

func smokeResumeRequest(t *testing.T, identity json.RawMessage) json.RawMessage {
	t.Helper()
	return json.RawMessage(`{"identity": ` + string(identity) + `, "workspace_paths": {"` + smokeWorkID + `": "/smoke/work"}, "execution_profile": "standard", "terminal": {"backend": "tmux", "terminal_id": "smoke-terminal", "interactive": true, "columns": 120, "rows": 40}, "lease": {"session_id": "` + smokeSessionID + `", "lease_epoch": 1, "lease_id": "` + smokeLeaseID + `"}}`)
}
