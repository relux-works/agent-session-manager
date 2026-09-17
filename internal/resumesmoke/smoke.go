package resumesmoke

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file runs one bounded native-resume smoke: a provider-specific
// probe, identify-session, discovery-bind, quiescence-precondition,
// and resume-plan sequence against one adapter executable through
// provhost.Host.Call, recorded as one closed smoke record.
//
// The check order is the refusal order. The Section 8.4 cell decides
// before any adapter call: unsupported and unknown tuples are
// refused with no call at all, so a refused tuple can neither mutate
// nor observe provider state. Available and conditional tuples pass
// the tuple gate, then the exact-version probe, then identification
// at exact confidence, then the discovery binding over the declared
// native root and backend realm, then the optional quiescence
// precondition, and only an available cell reaches the resume plan —
// a conditional cell gates it with the Section 19.3 promotion
// citation. Any failure stops the run at its check: later checks
// are absent from the record, never invented.
//
// Every adapter response is validated by the landed Section 7
// decoders and replayed through the validated members, never the
// raw body. The record digests the validated bytes; check details
// carry digests, member names, and gate citations only, never
// native references or secrets.

// smokeTimeFormat stamps record timestamps in the scalar timestamp
// shape: UTC RFC3339 with milliseconds.
const smokeTimeFormat = "2006-01-02T15:04:05.000Z07:00"

// Check names in execution order.
const (
	checkCell       = "resume-cell"
	checkGate       = "tuple-gate"
	checkProbe      = "probe"
	checkIdentify   = "identify"
	checkBind       = "discover-bind"
	checkQuiescence = "quiescence"
	checkResume     = "resume-plan"
)

// Terminal carries the Section 7.5 TerminalDescriptor fixture one
// smoke run resumes with.
type Terminal struct {
	Backend     string
	TerminalID  string
	Interactive bool
	Columns     int
	Rows        int
}

// Lease carries the Section 7.5 LeaseToken fixture: the session,
// the epoch above zero, and the UUIDv4 lease ID.
type Lease struct {
	SessionID string
	Epoch     uint64
	LeaseID   string
}

// Observation carries the Section 7.5 ProcessObservation fixture.
// Candidate lists are sorted unique, as the contract requires.
type Observation struct {
	TerminalID          string
	ExecutablePath      string
	StartedAt           string
	CandidateStorePaths []string
	CandidateNativeIDs  []string
}

// Params carries everything one smoke run needs: the claimed tuple,
// the host and adapter executable to drive, the request envelope
// facts, the session, workspace, terminal, lease, and observation
// fixtures, the discovery and quiescence precondition inputs, and
// the clock. Timestamps in the record come from Clock, so identical
// params with a fixed clock replay byte-identical records.
type Params struct {
	Tuple       provhost.BuildTuple
	Host        provhost.Host
	Executable  string
	Deadline    string
	ProbeID     string
	IdentifyID  string
	ResumeID    string
	SessionID   string
	Realm       string
	Workspaces  map[string]string
	Profile     string
	Terminal    Terminal
	Lease       Lease
	Observation Observation
	Home        string
	XDGDataHome string
	StoreRoot   string
	Discovery   []byte
	Quiescence  []byte
	Clock       func() time.Time
}

// Report is one finished smoke run: the decoded record and its
// canonical bytes, ready to store.
type Report struct {
	Record Record
	Bytes  []byte
}

// Run executes one smoke run and returns its report. Params errors
// and internal encoding failures return a Go error with no report;
// every specified refusal is a verdict in the report, never an
// error, because a refusal is the run's honest outcome.
func Run(ctx context.Context, params Params) (*Report, error) {
	clock := params.Clock
	if clock == nil {
		clock = time.Now
	}
	deadline, err := checkParams(params, clock)
	if err != nil {
		return nil, err
	}
	started := clock().UTC().Format(smokeTimeFormat)
	cell, gate := ResumeCell(params.Tuple.ProviderID, params.Tuple.ProviderVersion, params.Tuple.Platform, params.Tuple.Architecture)
	run := &runner{
		ctx:      ctx,
		params:   params,
		deadline: deadline,
		cell:     cell,
		gate:     gate,
		started:  started,
		record: Record{
			Schema:        RecordSchema,
			SchemaVersion: RecordSchemaVersion,
			Tuple: Tuple{
				ProviderID:      params.Tuple.ProviderID,
				ProviderVersion: params.Tuple.ProviderVersion,
				Platform:        params.Tuple.Platform,
				Architecture:    params.Tuple.Architecture,
			},
			ResumeCell: cell,
			GateRef:    gate,
			StartedAt:  started,
		},
	}
	run.note(checkCell, OutcomePass, cellDetail(cell, gate))
	if cell == CellUnsupported || cell == CellUnknown {
		return run.finish(clock)
	}
	if err := provhost.CheckResumeTuple(params.Tuple); err != nil {
		run.note(checkGate, OutcomeFail, "tuple gate refused the claimed build")
		return run.finish(clock)
	}
	run.note(checkGate, OutcomePass, "tuple gate admitted the claimed build")
	if _, _, ok := run.probe(); !ok {
		return run.finish(clock)
	}
	identity, ok := run.identify()
	if !ok {
		return run.finish(clock)
	}
	if ok := run.bind(identity.Identity); !ok {
		return run.finish(clock)
	}
	if ok := run.quiescence(); !ok {
		return run.finish(clock)
	}
	if ok := run.resume(identity.Identity); !ok {
		return run.finish(clock)
	}
	return run.finish(clock)
}

// runner carries one run's accumulating record.
type runner struct {
	ctx      context.Context
	params   Params
	deadline scalar.Timestamp
	cell     Cell
	gate     string
	started  string
	record   Record
}

// note appends one check to the record.
func (run *runner) note(name string, outcome Outcome, detail string) {
	run.record.Checks = append(run.record.Checks, Check{Name: name, Outcome: outcome, Detail: detail})
}

// finish stamps completion, derives the verdict, encodes, and
// self-verifies the record before returning it.
func (run *runner) finish(clock func() time.Time) (*Report, error) {
	run.record.CompletedAt = clock().UTC().Format(smokeTimeFormat)
	run.record.Verdict = deriveVerdict(run.cell, run.record.Checks)
	encoded, err := encodeRecord(run.record)
	if err != nil {
		return nil, err
	}
	decoded, err := VerifyRecord(encoded)
	if err != nil {
		return nil, fmt.Errorf("smoke run encoded an invalid record: %w", err)
	}
	return &Report{Record: decoded, Bytes: encoded}, nil
}

// probe calls the probe operation, replays the exact probed build,
// and fails closed on any drift from the claimed tuple.
func (run *runner) probe() (provhost.BuildTuple, string, bool) {
	body, err := json.Marshal(map[string]any{
		"platform":               run.params.Tuple.Platform,
		"architecture":           run.params.Tuple.Architecture,
		"provider_executable":    nil,
		"requested_capabilities": []string{"native_resume"},
	})
	if err != nil {
		run.note(checkProbe, OutcomeFail, "probe request is unframeable")
		return provhost.BuildTuple{}, "", false
	}
	response, err := run.call(provhost.OpProbe, run.params.ProbeID, body)
	if err != nil {
		run.note(checkProbe, OutcomeFail, "probe call failed: "+shortError(err))
		return provhost.BuildTuple{}, "", false
	}
	probed, err := provhost.ProbeBuild(response.Body)
	if err != nil {
		run.note(checkProbe, OutcomeFail, "probe response is not a valid probe: "+shortError(err))
		return provhost.BuildTuple{}, "", false
	}
	if probed != run.params.Tuple {
		run.note(checkProbe, OutcomeFail, "probe build drifted from the claimed tuple")
		return provhost.BuildTuple{}, "", false
	}
	digest := digestBytes(response.Body)
	run.record.ProbeDigest = digest
	run.note(checkProbe, OutcomePass, "exact-version probe matches the claimed tuple")
	return probed, digest, true
}

// identify calls identify-session and requires exact confidence:
// the smoke binds one precise native identity, never a guess.
func (run *runner) identify() (provhost.IdentifyOutcome, bool) {
	var empty provhost.IdentifyOutcome
	observation := map[string]any{
		"terminal_id":           run.params.Observation.TerminalID,
		"executable_path":       run.params.Observation.ExecutablePath,
		"started_at":            run.params.Observation.StartedAt,
		"candidate_store_paths": run.params.Observation.CandidateStorePaths,
		"candidate_native_ids":  run.params.Observation.CandidateNativeIDs,
	}
	body, err := json.Marshal(map[string]any{
		"session_id":  run.params.SessionID,
		"provider_id": run.params.Tuple.ProviderID,
		"observation": observation,
	})
	if err != nil {
		run.note(checkIdentify, OutcomeFail, "identify request is unframeable")
		return empty, false
	}
	response, err := run.call(provhost.OpIdentifySession, run.params.IdentifyID, body)
	if err != nil {
		run.note(checkIdentify, OutcomeFail, "identify call failed: "+shortError(err))
		return empty, false
	}
	outcome, err := provhost.SplitIdentifyResult(response.Body, run.params.Tuple.ProviderID)
	if err != nil {
		run.note(checkIdentify, OutcomeFail, "identify response is not a valid result: "+shortError(err))
		return empty, false
	}
	if outcome.Confidence != "exact" {
		run.note(checkIdentify, OutcomeFail, "identify confidence is below exact")
		return empty, false
	}
	identityID, err := pluckRecordID(outcome.Identity)
	if err != nil {
		run.note(checkIdentify, OutcomeFail, "identify record carries no digest identity")
		return empty, false
	}
	run.record.IdentityRecordID = identityID
	run.note(checkIdentify, OutcomePass, "session identified at exact confidence")
	return outcome, true
}

// bind resolves the Section 8.2 declared native root and binds the
// observed discovery proof to the identified record: the exact
// native ID, the discovery fact, the root containment, and the
// backend realm the tuple requires. A proof for another tuple
// refuses here.
func (run *runner) bind(identity []byte) bool {
	root, backendOnly, err := provhost.StoreRootFor(run.params.Tuple.ProviderID, run.params.Home, run.params.XDGDataHome)
	if err != nil {
		run.note(checkBind, OutcomeFail, "store root is not declared for this provider: "+shortError(err))
		return false
	}
	discovery := provhost.DiscoveryContext{
		Build:             run.params.Tuple,
		Home:              run.params.Home,
		XDGDataHome:       run.params.XDGDataHome,
		Realm:             run.params.Realm,
		StoreRootOverride: run.params.StoreRoot,
	}
	if err := provhost.VerifyIdentityDiscovery(identity, run.params.Discovery, discovery); err != nil {
		run.note(checkBind, OutcomeFail, "discovery proof does not bind to the identity: "+shortError(err))
		return false
	}
	if !backendOnly {
		run.record.StoreRoot = root
	}
	run.record.DiscoveryProofDigest = digestBytes(run.params.Discovery)
	run.note(checkBind, OutcomePass, "discovery proof binds the identity to its store")
	return true
}

// quiescence consumes the optional Section 7.6 precondition input:
// a supplied proof must be valid and safe, because graceful work
// stops on an unsafe boundary. An absent proof skips the check —
// the smoke performs no mutation of its own — and never fails it.
func (run *runner) quiescence() bool {
	if run.params.Quiescence == nil {
		run.note(checkQuiescence, OutcomeSkipped, "no quiescence proof input; smoke performs no mutation")
		return true
	}
	safe, err := provhost.DecodeQuiesceProof(run.params.Quiescence)
	if err != nil {
		run.note(checkQuiescence, OutcomeFail, "quiescence proof is not a valid proof: "+shortError(err))
		return false
	}
	if !safe {
		run.note(checkQuiescence, OutcomeFail, "quiescence proof is unsafe; graceful work stops")
		return false
	}
	run.record.QuiescenceDigest = digestBytes(run.params.Quiescence)
	run.note(checkQuiescence, OutcomePass, "quiescence precondition is safe")
	return true
}

// resume plans the native resume: the stored profile must map for
// the probed build, and the adapter must return a valid spawn plan.
// Only an available cell reaches the adapter here; any other cell
// gates the plan with the Section 19.3 promotion citation, and that
// gate can never be promoted by the smoke itself.
func (run *runner) resume(identity []byte) bool {
	if run.cell != CellAvailable {
		run.note(checkResume, OutcomeSkipped, "resume plan gated on "+run.gate+"; promotion only by section 19.3")
		return true
	}
	if _, err := provhost.ResolveMapping(run.params.Tuple.ProviderID, run.params.Profile, run.params.Tuple); err != nil {
		run.note(checkResume, OutcomeFail, "stored profile has no mapping for the probed build: "+shortError(err))
		return false
	}
	body, err := json.Marshal(map[string]any{
		"identity":          json.RawMessage(identity),
		"workspace_paths":   run.params.Workspaces,
		"execution_profile": run.params.Profile,
		"terminal": map[string]any{
			"backend":     run.params.Terminal.Backend,
			"terminal_id": run.params.Terminal.TerminalID,
			"interactive": run.params.Terminal.Interactive,
			"columns":     run.params.Terminal.Columns,
			"rows":        run.params.Terminal.Rows,
		},
		"lease": map[string]any{
			"session_id":  run.params.Lease.SessionID,
			"lease_epoch": run.params.Lease.Epoch,
			"lease_id":    run.params.Lease.LeaseID,
		},
	})
	if err != nil {
		run.note(checkResume, OutcomeFail, "resume request is unframeable")
		return false
	}
	response, err := run.call(provhost.OpResume, run.params.ResumeID, body)
	if err != nil {
		run.note(checkResume, OutcomeFail, "resume call failed: "+shortError(err))
		return false
	}
	platform, err := scalar.ParsePlatform(run.params.Tuple.Platform)
	if err != nil {
		run.note(checkResume, OutcomeFail, "tuple platform is not a registry member")
		return false
	}
	if err := provhost.DecodeSpawnPlan(response.Body, run.params.Tuple.ProviderID, run.params.Profile, platform); err != nil {
		run.note(checkResume, OutcomeFail, "resume plan is not a valid spawn plan: "+shortError(err))
		return false
	}
	run.record.SpawnPlanDigest = digestBytes(response.Body)
	run.note(checkResume, OutcomePass, "adapter returned a valid resume plan")
	return true
}

// call frames one request and drives it through the provider host.
func (run *runner) call(operation provhost.Operation, requestID string, body []byte) (provhost.Response, error) {
	parsed, err := scalar.ParseUUIDv7(requestID)
	if err != nil {
		return provhost.Response{}, err
	}
	return run.params.Host.Call(run.ctx, run.params.Executable, provhost.Request{
		Operation: operation,
		RequestID: parsed,
		Deadline:  run.deadline,
		Body:      body,
	})
}

// cellDetail renders the resume-cell check detail for one cell.
func cellDetail(cell Cell, gate string) string {
	switch cell {
	case CellAvailable:
		return "section 8.4 available cell " + gate + "; runtime probe still decides"
	case CellConditional:
		return "section 8.4 conditional cell " + gate + "; resume plan gated, promotion only by section 19.3"
	case CellUnsupported:
		return "section 8.4 unsupported cell " + gate + "; refused with no adapter call"
	default:
		return "section 8.4 unknown cell " + gate + "; refused with no adapter call"
	}
}

// shortError renders one failure for a check detail: the message
// text only, truncated, with no values the error may carry.
func shortError(err error) string {
	message := err.Error()
	if index := strings.IndexByte(message, '\n'); index >= 0 {
		message = message[:index]
	}
	if len(message) > 200 {
		message = message[:200]
	}
	return message
}

// pluckRecordID reads the record_id digest from validated identity
// bytes. The caller established the bytes as a valid Provider
// Identity Record; the digest shape is rechecked here so a drifted
// pluck fails closed instead of recording garbage.
func pluckRecordID(identity []byte) (string, error) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(identity, &members); err != nil {
		return "", err
	}
	var recordID string
	if err := json.Unmarshal(members["record_id"], &recordID); err != nil {
		return "", fmt.Errorf("identity record_id is not a string")
	}
	if !isDigest(recordID) {
		return "", fmt.Errorf("identity record_id is not a digest")
	}
	return recordID, nil
}

// checkParams validates the run params: the adapter address, the
// request envelope facts, the profile vocabulary, the Section 7.5
// fixture ranges the smoke depends on, and the required discovery
// precondition input. It returns the parsed deadline.
func checkParams(params Params, clock func() time.Time) (scalar.Timestamp, error) {
	var zero scalar.Timestamp
	if params.Executable == "" {
		return zero, fmt.Errorf("smoke executable is empty")
	}
	deadline, err := scalar.ParseTimestamp(params.Deadline)
	if err != nil {
		return zero, fmt.Errorf("smoke deadline is not a timestamp: %w", err)
	}
	instant, err := deadline.Time()
	if err != nil {
		return zero, fmt.Errorf("smoke deadline is not an instant: %w", err)
	}
	if !clock().Before(instant) {
		return zero, fmt.Errorf("smoke deadline is not in the future")
	}
	for _, member := range []struct {
		name  string
		value string
	}{
		{"probe request id", params.ProbeID},
		{"identify request id", params.IdentifyID},
		{"resume request id", params.ResumeID},
		{"session id", params.SessionID},
	} {
		if _, err := scalar.ParseUUIDv7(member.value); err != nil {
			return zero, fmt.Errorf("smoke %s is not a UUIDv7: %w", member.name, err)
		}
	}
	if params.Profile != "standard" && params.Profile != "yolo" {
		return zero, fmt.Errorf("smoke profile is not standard or yolo")
	}
	if len(params.Discovery) == 0 {
		return zero, fmt.Errorf("smoke discovery proof input is absent")
	}
	platform, err := scalar.ParsePlatform(params.Tuple.Platform)
	if err != nil {
		return zero, fmt.Errorf("smoke tuple platform is not a registry member: %w", err)
	}
	if err := checkParamsFixtures(params, platform); err != nil {
		return zero, err
	}
	return deadline, nil
}

// checkParamsFixtures validates the Section 7.5 fixture ranges:
// workspace paths, terminal geometry, lease shape, and observation
// bounds. Deeper adapter-body semantics stay the adapter's to
// refuse; these are the ranges the smoke itself depends on.
func checkParamsFixtures(params Params, platform scalar.Platform) error {
	if len(params.Workspaces) == 0 || len(params.Workspaces) > 256 {
		return fmt.Errorf("smoke workspace paths are not 1..256 entries")
	}
	for id, path := range params.Workspaces {
		if _, err := scalar.ParseUUIDv7(id); err != nil {
			return fmt.Errorf("smoke workspace id is not a UUIDv7: %w", err)
		}
		if _, err := scalar.ParseAbsolutePath(platform, path); err != nil {
			return fmt.Errorf("smoke workspace path is not absolute: %w", err)
		}
	}
	if params.Terminal.Backend != "tmux" && params.Terminal.Backend != "conpty" {
		return fmt.Errorf("smoke terminal backend is not tmux or conpty")
	}
	if params.Terminal.TerminalID == "" || len(params.Terminal.TerminalID) > 512 {
		return fmt.Errorf("smoke terminal id is not 1..512 characters")
	}
	if params.Terminal.Columns < 1 || params.Terminal.Columns > 1000 || params.Terminal.Rows < 1 || params.Terminal.Rows > 1000 {
		return fmt.Errorf("smoke terminal geometry is not 1..1000 cells")
	}
	if params.Lease.Epoch == 0 {
		return fmt.Errorf("smoke lease epoch is zero")
	}
	if _, err := scalar.ParseUUIDv4(params.Lease.LeaseID); err != nil {
		return fmt.Errorf("smoke lease id is not a UUIDv4: %w", err)
	}
	if params.Observation.TerminalID == "" || len(params.Observation.TerminalID) > 512 {
		return fmt.Errorf("smoke observation terminal id is not 1..512 characters")
	}
	if _, err := scalar.ParseAbsolutePath(platform, params.Observation.ExecutablePath); err != nil {
		return fmt.Errorf("smoke observation executable is not absolute: %w", err)
	}
	if _, err := scalar.ParseTimestamp(params.Observation.StartedAt); err != nil {
		return fmt.Errorf("smoke observation start is not a timestamp: %w", err)
	}
	for _, member := range []struct {
		name   string
		values []string
	}{
		{"candidate store paths", params.Observation.CandidateStorePaths},
		{"candidate native ids", params.Observation.CandidateNativeIDs},
	} {
		if len(member.values) > 256 {
			return fmt.Errorf("smoke %s exceed 256 entries", member.name)
		}
		if !sort.StringsAreSorted(member.values) {
			return fmt.Errorf("smoke %s are not sorted", member.name)
		}
		for index := 1; index < len(member.values); index++ {
			if member.values[index] == member.values[index-1] {
				return fmt.Errorf("smoke %s are not unique", member.name)
			}
		}
	}
	return nil
}
