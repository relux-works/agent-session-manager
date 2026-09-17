package resumesmoke

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/provhost"
)

// This file drives the production Run entry through scripted fake
// adapters: one verdict per Section 8.4 row, plus the focused
// refusals the story names. Positives cover the available cells;
// conditional cells gate; unsupported and unknown cells refuse with
// no adapter call at all.

// TestSmokeRowVerdicts runs the full smoke for every Section 8.4
// tuple the derivation parses and requires the row's verdict:
// available rows pass, conditional rows gate with the resume plan
// skipped, and unsupported and unknown rows refuse before any
// adapter call. Native-Windows bind rows need a Windows host —
// store roots join with the host separator while the proof rule is
// platform-native — so those four cases skip loudly elsewhere; the
// matrix derivation above still pins their cells on every host.
func TestSmokeRowVerdicts(t *testing.T) {
	rows := parseSection84(t)
	if len(rows) != 27 {
		t.Fatalf("parsed %d section 8.4 rows, want 27", len(rows))
	}
	var tuples []specRow
	for _, row := range rows {
		tuples = append(tuples, expandRow(row)...)
	}
	for _, row := range tuples {
		for _, architecture := range row.architectures {
			version := derivedFixtureVersions[row.provider]
			if row.pinned != "" {
				version = row.pinned
			}
			tuple := provhost.BuildTuple{ProviderID: row.provider, ProviderVersion: version, Platform: row.platform, Architecture: architecture}
			name := fmt.Sprintf("%s-%s-%s-%s", row.provider, version, row.platform, architecture)
			t.Run(name, func(t *testing.T) {
				if needsWindowsHost(tuple) && runtime.GOOS != "windows" {
					t.Skipf("windows native-store bind requires a Windows host; cell %q is pinned by TestResumeMatrixCoversEverySpecRow", row.cell)
				}
				runner := newScriptedRunner(t)
				params := smokeParams(tuple, runner)
				switch row.cell {
				case CellAvailable, CellConditional:
					runner.scriptProbe(tuple)
					runner.scriptIdentify(t, tuple, "exact")
					if row.cell == CellAvailable {
						runner.scriptResume(t, tuple, params.Profile)
					}
					params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
					if tuple.ProviderID == "antigravity" {
						params.Realm = smokeRealm
					}
				default:
					params.Discovery = json.RawMessage(`{}`)
				}
				report, err := Run(context.Background(), params)
				if err != nil {
					t.Fatalf("Run(%+v) error = %v", tuple, err)
				}
				assertNoCanary(t, report.Bytes)
				if _, err := VerifyRecord(report.Bytes); err != nil {
					t.Fatalf("VerifyRecord(row bytes) = %v", err)
				}
				switch row.cell {
				case CellAvailable:
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
					if report.Record.ProbeDigest == "" || report.Record.DiscoveryProofDigest == "" || report.Record.IdentityRecordID == "" || report.Record.SpawnPlanDigest == "" || report.Record.StoreRoot == "" {
						t.Fatal("pass record misses evidence digests or the store root")
					}
					assertCalls(t, runner, []string{"probe", "identify-session", "resume"})
				case CellConditional:
					assertVerdict(t, report.Record, VerdictGated)
					assertOutcomes(t, report.Record, map[string]Outcome{
						"resume-cell":   OutcomePass,
						"tuple-gate":    OutcomePass,
						"probe":         OutcomePass,
						"identify":      OutcomePass,
						"discover-bind": OutcomePass,
						"quiescence":    OutcomeSkipped,
						"resume-plan":   OutcomeSkipped,
					})
					plan := findCheck(t, report.Record, "resume-plan")
					if !strings.Contains(plan.Detail, "section 19.3") {
						t.Fatalf("gated resume-plan detail = %q, want the section 19.3 citation", plan.Detail)
					}
					assertCalls(t, runner, []string{"probe", "identify-session"})
				case CellUnsupported:
					assertVerdict(t, report.Record, VerdictUnsupported)
					assertRefused(t, report.Record, runner)
				default:
					assertVerdict(t, report.Record, VerdictUnknown)
					assertRefused(t, report.Record, runner)
				}
			})
		}
	}
}

// needsWindowsHost reports whether the tuple's bind step proves a
// filesystem store root on native Windows: only a Windows host
// joins that root with native separators, while the proof rule is
// platform-native. Backend-only Antigravity and zero-call refused
// rows run anywhere.
func needsWindowsHost(tuple provhost.BuildTuple) bool {
	if tuple.Platform != "windows" {
		return false
	}
	switch tuple.ProviderID {
	case "codex", "claude", "gemini", "pi":
		return true
	default:
		return false
	}
}

// TestSmokeMuseOnePatchOffPinRefuses proves the Appendix B version
// gate at the smoke boundary: one patch above the accepted Muse
// probe is unknown and refuses with no adapter call.
func TestSmokeMuseOnePatchOffPinRefuses(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "muse", ProviderVersion: offPinVersion("0.1.0"), Platform: "macos", Architecture: "arm64"}
	runner := newScriptedRunner(t)
	params := smokeParams(tuple, runner)
	params.Discovery = json.RawMessage(`{}`)
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	assertVerdict(t, report.Record, VerdictUnknown)
	assertRefused(t, report.Record, runner)
}

// TestSmokeAntigravityUnresolvedRealmFails proves the backend-realm
// precondition: an Antigravity proof the backend did not resolve
// fails the bind check, even on a conditional row.
func TestSmokeAntigravityUnresolvedRealmFails(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "antigravity", ProviderVersion: "1.1.14", Platform: "macos", Architecture: "arm64"}
	runner := newScriptedRunner(t)
	runner.scriptProbe(tuple)
	runner.scriptIdentify(t, tuple, "exact")
	params := smokeParams(tuple, runner)
	params.Realm = smokeRealm
	params.Discovery = json.RawMessage(fmt.Sprintf(`{"native_session_id": %q, "discovered": true, "discovery_root": null, "backend_resolved": false}`, smokeNativeID))
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	assertVerdict(t, report.Record, VerdictFail)
	assertOutcomes(t, report.Record, map[string]Outcome{
		"resume-cell":   OutcomePass,
		"tuple-gate":    OutcomePass,
		"probe":         OutcomePass,
		"identify":      OutcomePass,
		"discover-bind": OutcomeFail,
	})
}

// TestSmokeStrongConfidenceFails proves the smoke binds one precise
// identity: identify confidence below exact fails the run.
func TestSmokeStrongConfidenceFails(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	runner := newScriptedRunner(t)
	runner.scriptProbe(tuple)
	runner.scriptIdentify(t, tuple, "strong")
	params := smokeParams(tuple, runner)
	params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	assertVerdict(t, report.Record, VerdictFail)
	assertOutcomes(t, report.Record, map[string]Outcome{
		"resume-cell": OutcomePass,
		"tuple-gate":  OutcomePass,
		"probe":       OutcomePass,
		"identify":    OutcomeFail,
	})
}

// TestSmokeCrossTupleProofFails proves a discovery proof for another
// native session refuses at the bind check, as does a proof whose
// root escapes the declared store.
func TestSmokeCrossTupleProofFails(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	proofs := map[string]json.RawMessage{
		"foreign native id": json.RawMessage(`{"native_session_id": "another-session-id", "discovered": true, "discovery_root": "/smoke/home/.codex/sessions", "backend_resolved": false}`),
		"escaped root":      json.RawMessage(fmt.Sprintf(`{"native_session_id": %q, "discovered": true, "discovery_root": "/elsewhere/sessions", "backend_resolved": false}`, smokeNativeID)),
		"undiscovered":      json.RawMessage(fmt.Sprintf(`{"native_session_id": %q, "discovered": false, "discovery_root": "/smoke/home/.codex/sessions", "backend_resolved": false}`, smokeNativeID)),
	}
	for name, proof := range proofs {
		t.Run(name, func(t *testing.T) {
			runner := newScriptedRunner(t)
			runner.scriptProbe(tuple)
			runner.scriptIdentify(t, tuple, "exact")
			params := smokeParams(tuple, runner)
			params.Discovery = proof
			report, err := Run(context.Background(), params)
			if err != nil {
				t.Fatalf("Run error = %v", err)
			}
			assertVerdict(t, report.Record, VerdictFail)
			if got := findCheck(t, report.Record, "discover-bind"); got.Outcome != OutcomeFail {
				t.Fatalf("discover-bind = %q, want fail", got.Outcome)
			}
		})
	}
}

// TestSmokeProbeMismatchFails proves a probe that drifts from the
// claimed tuple fails closed: the run fails at the probe check and
// never identifies or resumes for the wrong build.
func TestSmokeProbeMismatchFails(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	drifted := tuple
	drifted.ProviderVersion = "0.148.0"
	runner := newScriptedRunner(t)
	runner.scriptProbe(drifted)
	params := smokeParams(tuple, runner)
	params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	assertVerdict(t, report.Record, VerdictFail)
	assertOutcomes(t, report.Record, map[string]Outcome{
		"resume-cell": OutcomePass,
		"tuple-gate":  OutcomePass,
		"probe":       OutcomeFail,
	})
	assertCalls(t, runner, []string{"probe"})
}

// TestSmokePiUnmappedVersionFails proves the Section 2.4 mapping
// rule at the resume plan: Pi past the pinned probe has an
// available cell but no profile mapping, so the plan fails.
func TestSmokePiUnmappedVersionFails(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "pi", ProviderVersion: "0.74.0", Platform: "macos", Architecture: "arm64"}
	runner := newScriptedRunner(t)
	runner.scriptProbe(tuple)
	runner.scriptIdentify(t, tuple, "exact")
	params := smokeParams(tuple, runner)
	params.Profile = "yolo"
	params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	assertVerdict(t, report.Record, VerdictFail)
	plan := findCheck(t, report.Record, "resume-plan")
	if plan.Outcome != OutcomeFail {
		t.Fatalf("resume-plan = %q, want fail", plan.Outcome)
	}
	if !strings.Contains(plan.Detail, "profile_mapping_unavailable") {
		t.Fatalf("resume-plan detail = %q, want the mapping failure", plan.Detail)
	}
	assertCalls(t, runner, []string{"probe", "identify-session"})
}

// TestSmokeUnsafeQuiescenceFails proves the quiescence precondition:
// a valid but unsafe proof fails the run before any resume plan.
func TestSmokeUnsafeQuiescenceFails(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	runner := newScriptedRunner(t)
	runner.scriptProbe(tuple)
	runner.scriptIdentify(t, tuple, "exact")
	params := smokeParams(tuple, runner)
	params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
	params.Quiescence = json.RawMessage(fmt.Sprintf(`{
  "provider_id": %q, "provider_version": %q, "input_blocked": true,
  "boundary_ref": "smoke-boundary", "foreground_idle": true, "background_idle": null,
  "open_child_count": 1, "open_database_handle_count": 0, "store_generation": "smoke-generation",
  "safe": false, "blockers": ["child_process_open"]
}`, tuple.ProviderID, tuple.ProviderVersion))
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	assertVerdict(t, report.Record, VerdictFail)
	if got := findCheck(t, report.Record, "quiescence"); got.Outcome != OutcomeFail {
		t.Fatalf("quiescence = %q, want fail", got.Outcome)
	}
}

// TestSmokeSafeQuiescencePasses proves the quiescence precondition
// input is consumed when present: a safe proof passes its check and
// its digest enters the record.
func TestSmokeSafeQuiescencePasses(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	runner := newScriptedRunner(t)
	runner.scriptProbe(tuple)
	runner.scriptIdentify(t, tuple, "exact")
	runner.scriptResume(t, tuple, "standard")
	params := smokeParams(tuple, runner)
	params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
	params.Quiescence = smokeQuiescence(t, tuple)
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	assertVerdict(t, report.Record, VerdictPass)
	if got := findCheck(t, report.Record, "quiescence"); got.Outcome != OutcomePass {
		t.Fatalf("quiescence = %q, want pass", got.Outcome)
	}
	if report.Record.QuiescenceDigest == "" {
		t.Fatal("pass record with quiescence input carries no quiescence digest")
	}
	assertNoCanary(t, report.Bytes)
}

// TestSmokeParamsRefusals proves malformed run params are caller
// errors with no report, never verdicts.
func TestSmokeParamsRefusals(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	mutate := map[string]func(*Params){
		"empty executable":    func(params *Params) { params.Executable = "" },
		"bad deadline":        func(params *Params) { params.Deadline = "yesterday" },
		"past deadline":       func(params *Params) { params.Deadline = "2020-01-01T00:00:00Z" },
		"bad probe id":        func(params *Params) { params.ProbeID = "not-a-uuid" },
		"bad session id":      func(params *Params) { params.SessionID = "not-a-uuid" },
		"bad profile":         func(params *Params) { params.Profile = "turbo" },
		"absent discovery":    func(params *Params) { params.Discovery = nil },
		"no workspaces":       func(params *Params) { params.Workspaces = map[string]string{} },
		"bad terminal":        func(params *Params) { params.Terminal.Columns = 0 },
		"zero lease epoch":    func(params *Params) { params.Lease.Epoch = 0 },
		"unsorted candidates": func(params *Params) { params.Observation.CandidateNativeIDs = []string{"b", "a"} },
	}
	for name, change := range mutate {
		t.Run(name, func(t *testing.T) {
			runner := newScriptedRunner(t)
			params := smokeParams(tuple, runner)
			params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
			change(&params)
			if _, err := Run(context.Background(), params); err == nil {
				t.Fatalf("Run(%s) = nil, want a params error", name)
			}
			if len(runner.calls) != 0 {
				t.Fatalf("Run(%s) made %d adapter calls, want none", name, len(runner.calls))
			}
		})
	}
}

// TestSmokeRecordCarriesNoCapabilityClaim proves the record shape
// carries evidence only: its closed member set has no capability,
// availability, or doctor surface for a reader to mistake for a
// capability advertisement.
func TestSmokeRecordCarriesNoCapabilityClaim(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	runner := newScriptedRunner(t)
	runner.scriptProbe(tuple)
	runner.scriptIdentify(t, tuple, "exact")
	runner.scriptResume(t, tuple, "standard")
	params := smokeParams(tuple, runner)
	params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(report.Bytes, &members); err != nil {
		t.Fatalf("unmarshal record: %v", err)
	}
	if len(members) != len(recordMembers) {
		t.Fatalf("record carries %d members, want the closed %d", len(members), len(recordMembers))
	}
	for name := range members {
		if !recordMembers[name] {
			t.Fatalf("record carries outside member %q", name)
		}
		if strings.Contains(name, "capab") || strings.Contains(name, "doctor") || strings.Contains(name, "enabled") {
			t.Fatalf("record member %q reads as a capability claim", name)
		}
	}
}

func assertVerdict(t *testing.T, record Record, want Verdict) {
	t.Helper()
	if record.Verdict != want {
		t.Fatalf("verdict = %q, want %q (checks %v)", record.Verdict, want, smokeOutcomeNames(record))
	}
}

func assertOutcomes(t *testing.T, record Record, want map[string]Outcome) {
	t.Helper()
	if len(record.Checks) != len(want) {
		t.Fatalf("checks = %v, want %d entries", smokeOutcomeNames(record), len(want))
	}
	for _, check := range record.Checks {
		outcome, ok := want[check.Name]
		if !ok {
			t.Fatalf("unexpected check %q in %v", check.Name, smokeOutcomeNames(record))
		}
		if check.Outcome != outcome {
			t.Fatalf("check %q = %q, want %q", check.Name, check.Outcome, outcome)
		}
		if check.Detail == "" {
			t.Fatalf("check %q carries an empty detail", check.Name)
		}
	}
}

func assertCalls(t *testing.T, runner *scriptedRunner, want []string) {
	t.Helper()
	if fmt.Sprintf("%v", runner.calls) != fmt.Sprintf("%v", want) {
		t.Fatalf("adapter calls = %v, want %v", runner.calls, want)
	}
}

// assertRefused requires the refused shape: the cell check alone,
// no evidence digests, and no adapter call.
func assertRefused(t *testing.T, record Record, runner *scriptedRunner) {
	t.Helper()
	assertOutcomes(t, record, map[string]Outcome{"resume-cell": OutcomePass})
	if record.ProbeDigest != "" || record.DiscoveryProofDigest != "" || record.IdentityRecordID != "" || record.SpawnPlanDigest != "" || record.StoreRoot != "" {
		t.Fatal("refused record carries evidence it never observed")
	}
	assertCalls(t, runner, nil)
}

func findCheck(t *testing.T, record Record, name string) Check {
	t.Helper()
	for _, check := range record.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("record has no %q check in %v", name, smokeOutcomeNames(record))
	return Check{}
}
