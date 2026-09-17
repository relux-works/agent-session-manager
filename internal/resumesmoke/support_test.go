package resumesmoke

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/provhost"
)

// Fixed fixture facts. The clock never advances inside a run, so one
// fixed instant keeps started_at and completed_at equal and every
// replay byte-identical. IDs are distinct valid UUIDv7 values; the
// lease ID is a valid UUIDv4.
const (
	smokeClock     = "2026-09-17T00:00:00Z"
	smokeDeadline  = "2026-09-18T00:00:00.000Z"
	smokeSessionID = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	smokeWorkID    = "0198f4c8-6c30-7d44-8d5e-1234567890ab"
	smokeHostID    = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	smokeProbeID   = "0198f4c8-8e50-7f66-8f70-1234567890ab"
	smokeIdentID   = "0198f4c8-9f60-7a77-9a81-1234567890ab"
	smokeResumeID  = "0198f4c8-af70-7b88-ab92-1234567890ab"
	smokeLeaseID   = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	smokeCreatedAt = "2026-09-17T00:00:00.000Z"
	smokeRealm     = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"
	smokeHome      = "/smoke/home"
)

// smokeNativeID is the fixture native session ID. It is a canary:
// no smoke record may contain it, because records carry digests,
// never raw native references.
const smokeNativeID = "smoke-native-canary-11111111-2222-4333-8444-555555555555"

// scriptedRunner is an in-process fake provider adapter: it speaks
// the Section 7.2 frame protocol through the real provhost.Host,
// answers each operation from its script, and records every call.
// Unknown operations fail the test, because the smoke must dispatch
// exactly the sequence it claims.
type scriptedRunner struct {
	t        *testing.T
	script   map[string]func(body json.RawMessage) json.RawMessage
	calls    []string
	bodies   map[string]json.RawMessage
	identity json.RawMessage
}

func newScriptedRunner(t *testing.T) *scriptedRunner {
	t.Helper()
	return &scriptedRunner{t: t, script: map[string]func(body json.RawMessage) json.RawMessage{}, bodies: map[string]json.RawMessage{}}
}

func (runner *scriptedRunner) Run(_ context.Context, _ string, stdin []byte) (provhost.Result, error) {
	var frame struct {
		Protocol        string          `json:"protocol"`
		ProtocolVersion string          `json:"protocol_version"`
		RequestID       string          `json:"request_id"`
		Operation       string          `json:"operation"`
		Body            json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(stdin, &frame); err != nil {
		runner.t.Fatalf("fake adapter received a malformed frame: %v", err)
	}
	if frame.Protocol != provhost.ProtocolID || frame.ProtocolVersion != provhost.ProtocolVersion {
		runner.t.Fatalf("fake adapter received %q %q, want the provider protocol", frame.Protocol, frame.ProtocolVersion)
	}
	if frame.RequestID == "" {
		runner.t.Fatal("fake adapter received an empty request id")
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(frame.Body, &probe); err != nil {
		runner.t.Fatalf("fake adapter received a non-object body: %v", err)
	}
	handler, ok := runner.script[frame.Operation]
	if !ok {
		runner.t.Fatalf("fake adapter received unexpected operation %q", frame.Operation)
	}
	runner.calls = append(runner.calls, frame.Operation)
	runner.bodies[frame.Operation] = frame.Body
	response := map[string]any{
		"protocol":         provhost.ProtocolID,
		"protocol_version": provhost.ProtocolVersion,
		"request_id":       frame.RequestID,
		"ok":               true,
		"body":             handler(frame.Body),
	}
	stdout, err := json.Marshal(response)
	if err != nil {
		runner.t.Fatalf("fake adapter cannot frame its response: %v", err)
	}
	return provhost.Result{Stdout: append(stdout, '\n'), ExitCode: 0}, nil
}

// scriptProbe answers the probe operation with a valid probe for
// the tuple. The native_resume capability mirrors the row: on an
// available row the fake claims available and enabled, elsewhere
// conditional and disabled. The smoke never consults these claims —
// the Section 8.4 matrix decides — but the fixture stays plausible.
func (runner *scriptedRunner) scriptProbe(tuple provhost.BuildTuple) {
	cell, _ := ResumeCell(tuple.ProviderID, tuple.ProviderVersion, tuple.Platform, tuple.Architecture)
	status := "conditional"
	enabled := false
	evidence := "acceptance_required"
	if cell == CellAvailable {
		status = "available"
		enabled = true
		evidence = "probed"
	}
	runner.script["probe"] = func(body json.RawMessage) json.RawMessage {
		var probe struct {
			Platform     string `json:"platform"`
			Architecture string `json:"architecture"`
		}
		if err := json.Unmarshal(body, &probe); err != nil {
			runner.t.Fatalf("fake probe body malformed: %v", err)
		}
		if probe.Platform != tuple.Platform || probe.Architecture != tuple.Architecture {
			runner.t.Fatalf("fake probe asked for %s/%s, want the claimed tuple", probe.Platform, probe.Architecture)
		}
		capability := func(status string, enabled bool, evidence, detail string) map[string]any {
			return map[string]any{"status": status, "enabled": enabled, "evidence": evidence, "detail": detail}
		}
		return json.RawMessage(fmt.Sprintf(`{
  "schema": "urn:ax:schema:provider-probe",
  "schema_version": "1.0.0",
  "provider_id": %q,
  "provider_version": %q,
  "platform": %q,
  "architecture": %q,
  "capabilities": {
    "native_resume": %s,
    "portable_store": %s,
    "managed_pty": %s,
    "appserver": %s,
    "task_board_primary": %s,
    "prompt_spawn": %s,
    "native_goal_binding": %s
  },
  "warnings": []
}`,
			tuple.ProviderID, tuple.ProviderVersion, tuple.Platform, tuple.Architecture,
			mustJSON(capability(status, enabled, evidence, "smoke fixture claim")),
			mustJSON(capability("conditional", false, "acceptance_required", "smoke fixture claim")),
			mustJSON(capability("conditional", false, "acceptance_required", "smoke fixture claim")),
			mustJSON(capability("conditional", false, "acceptance_required", "smoke fixture claim")),
			mustJSON(capability("conditional", false, "acceptance_required", "smoke fixture claim")),
			mustJSON(capability("conditional", false, "acceptance_required", "smoke fixture claim")),
			mustJSON(capability("conditional", false, "acceptance_required", "smoke fixture claim")),
		))
	}
}

// scriptIdentify answers identify-session with a host-created
// identity for the tuple at the given confidence. The identity
// bytes are retained so the resume script can prove the smoke
// threaded exactly them into the resume body.
func (runner *scriptedRunner) scriptIdentify(t *testing.T, tuple provhost.BuildTuple, confidence string) {
	t.Helper()
	identity := smokeIdentity(t, tuple)
	runner.identity = identity
	runner.script["identify-session"] = func(body json.RawMessage) json.RawMessage {
		var identify struct {
			SessionID  string `json:"session_id"`
			ProviderID string `json:"provider_id"`
		}
		if err := json.Unmarshal(body, &identify); err != nil {
			runner.t.Fatalf("fake identify body malformed: %v", err)
		}
		if identify.SessionID != smokeSessionID || identify.ProviderID != tuple.ProviderID {
			runner.t.Fatalf("fake identify asked for %s/%s, want the smoke session", identify.ProviderID, identify.SessionID)
		}
		return json.RawMessage(fmt.Sprintf(`{"identity": %s, "confidence": %q, "matched_evidence": ["native_id"]}`, identity, confidence))
	}
}

// scriptResume answers the resume operation with a valid spawn plan
// and proves the smoke threaded the identified record into the
// resume body byte for byte.
func (runner *scriptedRunner) scriptResume(t *testing.T, tuple provhost.BuildTuple, profile string) {
	t.Helper()
	runner.script["resume"] = func(body json.RawMessage) json.RawMessage {
		var resume struct {
			Identity  json.RawMessage   `json:"identity"`
			Profile   string            `json:"execution_profile"`
			Workspace map[string]string `json:"workspace_paths"`
		}
		if err := json.Unmarshal(body, &resume); err != nil {
			runner.t.Fatalf("fake resume body malformed: %v", err)
		}
		if string(canonicalJSON(t, resume.Identity)) != string(canonicalJSON(t, runner.identity)) {
			runner.t.Fatal("fake resume received an identity other than the identified record")
		}
		if resume.Profile != profile {
			runner.t.Fatalf("fake resume received profile %q, want %q", resume.Profile, profile)
		}
		return smokeSpawnPlan(t, tuple, profile)
	}
}

// smokeIdentity creates the fixture Provider Identity Record for
// one tuple through the production creation entry: a session UUID
// for filesystem providers, a backend conversation UUID with the
// smoke realm for Antigravity.
func smokeIdentity(t *testing.T, tuple provhost.BuildTuple) []byte {
	t.Helper()
	params := provhost.IdentityParams{
		SessionID:            smokeSessionID,
		ProviderID:           tuple.ProviderID,
		ProviderVersion:      tuple.ProviderVersion,
		ProviderVersionRange: ">=0 <99",
		NativeSessionID:      smokeNativeID,
		IdentityKind:         "session_uuid",
		LogicalWorkspaceID:   smokeWorkID,
		CreatedByHostID:      smokeHostID,
		CreatedAt:            smokeCreatedAt,
	}
	if tuple.ProviderID == "antigravity" {
		params.IdentityKind = "backend_conversation_uuid"
		params.BackendRealm = smokeRealm
	}
	identity, err := provhost.CreateIdentity(params)
	if err != nil {
		t.Fatalf("CreateIdentity(%+v): %v", tuple, err)
	}
	return identity
}

// smokeDiscovery renders the observed discovery-proof input for one
// tuple: discovered under the Section 8.2 store root the production
// entry resolves, or backend-resolved with a null root for
// Antigravity, which identifies through its realm.
func smokeDiscovery(t *testing.T, tuple provhost.BuildTuple, home, xdg string) []byte {
	t.Helper()
	root, backendOnly, err := provhost.StoreRootFor(tuple.ProviderID, home, xdg)
	if err != nil {
		t.Fatalf("StoreRootFor(%q): %v", tuple.ProviderID, err)
	}
	if backendOnly {
		return json.RawMessage(fmt.Sprintf(`{"native_session_id": %q, "discovered": true, "discovery_root": null, "backend_resolved": true}`, smokeNativeID))
	}
	return json.RawMessage(fmt.Sprintf(`{"native_session_id": %q, "discovered": true, "discovery_root": %q, "backend_resolved": false}`, smokeNativeID, root))
}

// smokeQuiescence renders a safe Section 7.6 proof input.
func smokeQuiescence(t *testing.T, tuple provhost.BuildTuple) []byte {
	t.Helper()
	return json.RawMessage(fmt.Sprintf(`{
  "provider_id": %q,
  "provider_version": %q,
  "input_blocked": true,
  "boundary_ref": "smoke-boundary",
  "foreground_idle": true,
  "background_idle": true,
  "open_child_count": 0,
  "open_database_handle_count": 0,
  "store_generation": "smoke-generation",
  "safe": true,
  "blockers": []
}`, tuple.ProviderID, tuple.ProviderVersion))
}

// smokeSpawnPlan renders a valid resume SpawnPlan for one tuple and
// profile. The argv names the native session, which is exactly why
// the record may carry only the plan digest: the canary must never
// appear in record bytes. The profile_mapping comes from the
// production table, never retyped.
func smokeSpawnPlan(t *testing.T, tuple provhost.BuildTuple, profile string) json.RawMessage {
	t.Helper()
	mapping, err := provhost.ProfileMapping(tuple.ProviderID, profile)
	if err != nil {
		t.Fatalf("ProfileMapping(%q, %q): %v", tuple.ProviderID, profile, err)
	}
	cwd := "/smoke/work"
	if tuple.Platform == "windows" {
		cwd = `C:\smoke\work`
	}
	argv := []string{tuple.ProviderID, "resume", smokeNativeID}
	if mapping != "" {
		argv = append([]string{argv[0], mapping}, argv[1:]...)
	}
	plan := map[string]any{
		"argv":              argv,
		"cwd":               cwd,
		"env_names":         []string{},
		"env_literals":      map[string]string{},
		"native_session_id": smokeNativeID,
		"profile_mapping":   mapping,
		"extensions":        map[string]any{},
	}
	return mustJSONRaw(plan)
}

// smokeParams builds valid smoke params for one tuple against the
// scripted runner: standard profile, POSIX fixtures, the smoke
// discovery input, and no quiescence input. Callers mutate the
// result for their negative.
func smokeParams(tuple provhost.BuildTuple, runner *scriptedRunner) Params {
	clock, err := time.Parse(time.RFC3339, smokeClock)
	if err != nil {
		panic(err)
	}
	workspace := "/smoke/work"
	executable := "/smoke/bin/" + tuple.ProviderID
	if tuple.Platform == "windows" {
		workspace = `C:\smoke\work`
		executable = `C:\smoke\bin\` + tuple.ProviderID + ".exe"
	}
	workspaces := map[string]string{smokeWorkID: workspace}
	return Params{
		Tuple:      tuple,
		Host:       provhost.Host{Runner: runner, Now: func() time.Time { return clock }},
		Executable: executable,
		Deadline:   smokeDeadline,
		ProbeID:    smokeProbeID,
		IdentifyID: smokeIdentID,
		ResumeID:   smokeResumeID,
		SessionID:  smokeSessionID,
		Workspaces: workspaces,
		Profile:    "standard",
		Terminal: Terminal{
			Backend:     "tmux",
			TerminalID:  "smoke-terminal",
			Interactive: true,
			Columns:     120,
			Rows:        40,
		},
		Lease: Lease{
			SessionID: smokeSessionID,
			Epoch:     1,
			LeaseID:   smokeLeaseID,
		},
		Observation: Observation{
			TerminalID:          "smoke-terminal",
			ExecutablePath:      executable,
			StartedAt:           smokeCreatedAt,
			CandidateStorePaths: []string{},
			CandidateNativeIDs:  []string{},
		},
		Home:      smokeHome,
		Discovery: nil,
		Clock:     func() time.Time { return clock },
	}
}

// smokeOutcomeNames returns the check name/outcome pairs of a report
// in execution order.
func smokeOutcomeNames(record Record) []string {
	var names []string
	for _, check := range record.Checks {
		names = append(names, check.Name+"="+string(check.Outcome))
	}
	return names
}

func mustJSON(value any) string {
	return string(mustJSONRaw(value))
}

func mustJSONRaw(value any) json.RawMessage {
	rendered, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return rendered
}

// canonicalJSON remarshals JSON through sorted maps so semantically
// equal documents compare equal as bytes.
func canonicalJSON(t *testing.T, raw json.RawMessage) json.RawMessage {
	t.Helper()
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("canonicalize JSON: %v", err)
	}
	return mustJSONRaw(sortJSON(decoded))
}

func sortJSON(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(typed))
		for _, key := range keys {
			ordered[key] = sortJSON(typed[key])
		}
		return ordered
	case []any:
		for index := range typed {
			typed[index] = sortJSON(typed[index])
		}
		return typed
	default:
		return value
	}
}

// assertNoCanary fails when the record bytes leak the fixture
// native ID or any other raw native reference the canary stands
// for. Records carry digests, never the references themselves.
func assertNoCanary(t *testing.T, record []byte) {
	t.Helper()
	if strings.Contains(string(record), smokeNativeID) {
		t.Fatal("smoke record leaks the raw native session id")
	}
}
