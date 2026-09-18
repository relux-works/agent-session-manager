package axpane

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// runWorld binds real durable stores: the session chain with its
// epoch-1 lease, a committed journal, a captured checkpoint, and an
// empty binding store.
type runWorld struct {
	repo     *sessrepo.Repository
	recordID string
	headID   string
	mat      *matjournal.Store
	ckpt     *sessckpt.Store
	ckptID   string
	pane     *Store
	paneRoot string
	universe *backendUniverse
}

func buildRunWorld(t *testing.T) *runWorld {
	t.Helper()
	world := &runWorld{universe: buildBackendUniverse(t, true)}
	world.repo, world.recordID = chainFixture(t)
	events, err := world.repo.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	world.headID = events[len(events)-1].EventID
	matStore, err := matjournal.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	world.mat = matStore
	journalFixture(t, matStore)
	ckptStore, err := sessckpt.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	world.ckpt = ckptStore
	world.ckptID, _ = checkpointFixture(t, world.repo, ckptStore, []string{world.headID})
	world.paneRoot = t.TempDir()
	paneStore, err := Open(world.paneRoot)
	if err != nil {
		t.Fatal(err)
	}
	world.pane = paneStore
	return world
}

func (world *runWorld) stores() Stores {
	return Stores{Repo: world.repo, Mat: world.mat, Ckpt: world.ckpt, Pane: world.pane}
}

func configInputs(t *testing.T) (config.Inputs, config.Overrides) {
	t.Helper()
	home := t.TempDir()
	platform, err := scalar.ParsePlatform("linux")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"config", "data", "state", "cache", "runtime"} {
		if err := os.MkdirAll(home+"/"+dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return config.Inputs{
			Platform:   platform,
			HomeDir:    home,
			TempDir:    home,
			WorkingDir: home,
			LookupEnv:  func(string) (string, bool) { return "", false },
			Stat:       func(path string) (fs.FileInfo, error) { return os.Stat(path) },
			ReadFile:   os.ReadFile,
		}, config.Overrides{
			config.ConfigFile:  home + "/config/config.toml",
			config.DataRoot:    home + "/data",
			config.StateRoot:   home + "/state",
			config.CacheRoot:   home + "/cache",
			config.RuntimeRoot: home + "/runtime",
		}
}

func runObserve(now time.Time) fencing.ObserveInput {
	grant, policy := fixtureGrant(now)
	return fencing.ObserveInput{
		LocalHostID: fixtureLocalHost,
		Verified:    true,
		Grant:       grant,
		HasGrant:    true,
		Policy:      policy,
		Now:         now,
	}
}

func runRequest(t *testing.T, world *runWorld) Request {
	t.Helper()
	inputs, overrides := configInputs(t)
	return Request{
		SessionID:            fixtureSession,
		BootstrapOperationID: fixtureBootstrap,
		Mode:                 ModeLaunch,
		ConfigInputs:         inputs,
		ConfigOverrides:      overrides,
		Presented:            fixturePresented(),
		Observe:              runObserve(fixtureNow()),
		Provider: ProviderFacts{
			Build:    fixtureBuild(),
			Identity: fixtureIdentity(t),
		},
		Realm: Realm{
			Caller:           CallerForeground,
			BrokerState:      "broker-running",
			ServerGeneration: "generation-alpha",
			Remediation:      "launch the aqua broker",
		},
		Backend:     world.universe.facts(t),
		Entrypoint:  []string{"ax", "pane", fixtureSession},
		HostBinding: world.universe.hostBinding(),
		Interactive: true,
		Columns:     80,
		Rows:        24,
		InstanceID:  fixtureInstance,
		LocalHostID: fixtureLocalHost,
		CreatedAt:   fixtureCreatedAt,
	}
}

func eventCount(t *testing.T, repository *sessrepo.Repository) int {
	t.Helper()
	events, err := repository.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	return len(events)
}

func TestRunLaunchBinds(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	outcome, err := Run(world.stores(), runRequest(t, world))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() action = %q, detail %q, cause %v, want launch",
			outcome.Decision.Action, outcome.Decision.Detail, outcome.Decision.Cause)
	}
	if outcome.Binding == nil {
		t.Fatal("Run() launch binds no bootstrap pair")
	}
	if outcome.Binding.OperationID != fixtureBootstrap || outcome.Binding.TerminalInstanceID != fixtureInstance {
		t.Fatalf("Run() binding = %+v", outcome.Binding)
	}
	proven, found, err := world.pane.Status(fixtureSession)
	if err != nil || !found || proven != *outcome.Binding {
		t.Fatalf("Status() = (%+v, %v, %v), want the bound receipt", proven, found, err)
	}
	if _, err := outcome.Decision.Token.Bind(fencing.ProviderQuiesce); err != nil {
		t.Fatalf("launch token bind error = %v", err)
	}
	if outcome.Emitted {
		t.Fatal("Run() launch authors an event, want the binding only")
	}
}

func TestRunReattaches(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	first, _, err := world.pane.Bind(fixtureSession, fixtureBootstrap, bindCandidate())
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := Run(world.stores(), runRequest(t, world))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionReattach {
		t.Fatalf("Run() action = %q, want reattach", outcome.Decision.Action)
	}
	if outcome.Binding == nil || *outcome.Binding != first {
		t.Fatalf("Run() binding = %+v, want the recorded %+v", outcome.Binding, first)
	}
}

func TestRunConcurrentLaunchReattaches(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	request := runRequest(t, world)
	winner := bindCandidate()
	winner.TerminalInstanceID = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	winner.BindingDigest = BindingDigest(winner)
	request.Hooks = &Hooks{
		AfterStage: func(path string) error {
			inner, err := Open(world.paneRoot)
			if err != nil {
				return err
			}
			_, _, err = inner.Bind(fixtureSession, fixtureBootstrap, winner)
			return err
		},
	}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionReattach {
		t.Fatalf("Run() action = %q, want reattach to the concurrent winner", outcome.Decision.Action)
	}
	if outcome.Binding == nil || outcome.Binding.TerminalInstanceID != winner.TerminalInstanceID {
		t.Fatalf("Run() binding = %+v, want the winner's receipt", outcome.Binding)
	}
}

func TestRunParkedEmits(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	appendChainEvent(t, world.repo, "session.failed", 1, fixtureLeaseA, 2, world.headID, map[string]any{
		"error_code": "bootstrap_probe", "retryable": false, "operation_id": nil,
	})
	events, err := world.repo.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	world.headID = events[len(events)-1].EventID
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.MaterializationRequired = true
	request.MaterializationID = "0198f4c8-9a10-7b22-8b3c-1234567890ff"
	request.CheckpointRequired = true
	request.CheckpointID = world.ckptID
	before := eventCount(t, world.repo)
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionParked {
		t.Fatalf("Run() action = %q, want parked", outcome.Decision.Action)
	}
	if !outcome.Emitted || outcome.Event == nil {
		t.Fatal("Run() parked decision authors no event")
	}
	if after := eventCount(t, world.repo); after != before+1 {
		t.Fatalf("Run() parked event count = %d, want %d", after, before+1)
	}
	stored, err := world.repo.GetEvent(fixtureSession, outcome.Event.EventID)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		EventType string `json:"event_type"`
		LeaseID   string `json:"lease_id"`
		Payload   struct {
			Reason string `json:"reason"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(stored, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.EventType != "session.parked" || decoded.LeaseID != fixtureLeaseA || decoded.Payload.Reason != "restore_policy" {
		t.Fatalf("parked event = %+v", decoded)
	}
	if _, found, err := world.pane.Status(fixtureSession); err != nil || found {
		t.Fatalf("Status() after parked = (%v, %v), want no binding", found, err)
	}
}

func TestRunRemoteNonInteractiveParksWithoutEmission(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	leases, err := world.repo.ListLeases(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.repo.CompareAndSwapLease(fixtureSession, sessrepo.LeaseExpectation{RecordID: leases[0].RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{
			LeaseID:        fixtureLeaseB,
			HolderHostID:   fixtureRemoteHost,
			IssuedByHostID: fixtureRemoteHost,
			CreatedAt:      fixtureCreatedAt,
		},
		Reason:       "graceful_takeover",
		CheckpointID: seedDigest(0xC9),
	}); err != nil {
		t.Fatalf("CompareAndSwapLease() error = %v", err)
	}
	// A remote winner parks without chain emission: the local wrapper
	// must not manufacture a sequence collision under another host's
	// lease (§5.2). The non-interactive remote owner parks with the
	// remote_owner reason and no event.
	before := eventCount(t, world.repo)
	outcome, err := Run(world.stores(), runRequest(t, world))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionParked || outcome.Decision.ParkReason != fencing.ParkRemoteOwner {
		t.Fatalf("Run() = (%q, %q), want parked remote_owner for non-interactive remote", outcome.Decision.Action, outcome.Decision.ParkReason)
	}
	if outcome.Emitted {
		t.Fatal("Run() remote park authors a chain event under another host's lease, want no emission")
	}
	if after := eventCount(t, world.repo); after != before {
		t.Fatalf("event count = %d, want %d (no remote emission)", after, before)
	}
	if outcome.Decision.WinningLeaseID != fixtureLeaseB {
		t.Fatalf("WinningLeaseID = %q, want the remote winner", outcome.Decision.WinningLeaseID)
	}
}

func TestRunParkedWithoutWinnerSkipsEmission(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	// A session with no leases yet: remove the epoch-1 lease the
	// chain fixture installed by working on a lease-free sibling.
	repository, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal([]byte(sessionRecordFixture), &record); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateSession(identifyObject(t, record, "record_id")); err != nil {
		t.Fatal(err)
	}
	world.repo = repository
	request := runRequest(t, world)
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionParked {
		t.Fatalf("Run() action = %q, want parked", outcome.Decision.Action)
	}
	if outcome.Emitted {
		t.Fatal("Run() parked without a winner authors an event, want none")
	}
}

func TestRunRefusedWritesNothing(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	request := runRequest(t, world)
	request.Entrypoint = []string{"codex", "--yolo"}
	before := eventCount(t, world.repo)
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionRefused || outcome.Decision.Class != terminalbackend.CodePreconditionFailed {
		t.Fatalf("Run() = (%q, %q), want refused local_precondition_failed", outcome.Decision.Action, outcome.Decision.Class)
	}
	if outcome.Binding != nil || outcome.Emitted {
		t.Fatal("Run() refusal writes durable state")
	}
	if _, found, err := world.pane.Status(fixtureSession); err != nil || found {
		t.Fatalf("Status() after refusal = (%v, %v), want absence", found, err)
	}
	if after := eventCount(t, world.repo); after != before {
		t.Fatalf("event count = %d, want %d", after, before)
	}
}

func TestRunIdempotencyMismatch(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	other := bindCandidate()
	other.OperationID = fixtureOtherOp
	if _, _, err := world.pane.Bind(fixtureSession, fixtureOtherOp, other); err != nil {
		t.Fatal(err)
	}
	outcome, err := Run(world.stores(), runRequest(t, world))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionRefused || outcome.Decision.Class != terminalbackend.CodeIdempotencyMismatch {
		t.Fatalf("Run() = (%q, %q), want refused idempotency_mismatch", outcome.Decision.Action, outcome.Decision.Class)
	}
}

func TestRunRestoreCommitted(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	world.headID = publishCheckpoint(t, world.repo, world.headID, 2, world.ckptID)
	world.mat = journalSourced(t, world.ckptID)
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.MaterializationRequired = true
	request.MaterializationID = fixtureMat
	request.CheckpointRequired = true
	request.CheckpointID = world.ckptID
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() action = %q, detail %q, cause %v, want launch",
			outcome.Decision.Action, outcome.Decision.Detail, outcome.Decision.Cause)
	}
	if outcome.Binding == nil {
		t.Fatal("Run() restore launch binds no bootstrap pair")
	}
}

func TestRunRestoreMaterializingParks(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	request := runRequest(t, world)
	request.Mode = ModeRestore
	request.MaterializationRequired = true
	request.MaterializationID = "0198f4c8-9a10-7b22-8b3c-1234567890ff"
	request.CheckpointRequired = true
	request.CheckpointID = world.ckptID
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionParked || outcome.Decision.ParkReason != fencing.ParkRestorePolicy {
		t.Fatalf("Run() = (%q, %q), want parked restore_policy", outcome.Decision.Action, outcome.Decision.ParkReason)
	}
}

func TestRunUnknownSession(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	request := runRequest(t, world)
	request.SessionID = fixtureForeign
	request.Entrypoint = []string{"ax", "pane", fixtureForeign}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionRefused || outcome.Decision.Class != ClassInvalidArguments {
		t.Fatalf("Run() = (%q, %q), want refused invalid_arguments", outcome.Decision.Action, outcome.Decision.Class)
	}
}

func TestRunInvalidConfig(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	request := runRequest(t, world)
	request.ConfigOverrides = config.Overrides{
		config.ConfigFile: "/nonexistent-axpane-dir-7f3a/config.toml",
	}
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionRefused || outcome.Decision.Class != ClassInvalidConfig {
		t.Fatalf("Run() = (%q, %q), want refused invalid_config", outcome.Decision.Action, outcome.Decision.Class)
	}
}

func TestRunSmokeRequiredPasses(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	addRealmEvidence(t, world.universe, seedDigest(0xB1), "codex", "0.147.0")
	request := runRequest(t, world)
	record := smokeRecord(t, "pass", "A", passingSmokeChecks(), smokeTuple())
	request.Smoke.Required = true
	request.Smoke.Record = record
	request.Smoke.Target = smokeTarget()
	outcome, err := Run(world.stores(), request)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if outcome.Decision.Action != ActionLaunch {
		t.Fatalf("Run() action = %q, cause %v, want launch", outcome.Decision.Action, outcome.Decision.Cause)
	}
}

func TestRunTornBindingPropagates(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	if _, _, err := world.pane.Bind(fixtureSession, fixtureBootstrap, bindCandidate()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(world.pane.bindingPath(fixtureSession), []byte(`{"torn":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	outcome, err := Run(world.stores(), runRequest(t, world))
	if err == nil {
		t.Fatalf("Run() over torn binding succeeded with %+v, want the read error", outcome.Decision)
	}
	if outcome.Decision.Action != "" {
		t.Fatalf("Run() over torn binding decides %q, want no decision", outcome.Decision.Action)
	}
}

func TestObserveLoadsWinningLease(t *testing.T) {
	t.Parallel()
	world := buildRunWorld(t)
	observation, err := ObserveOwnership(world.repo, fixtureSession, runObserve(fixtureNow()))
	if err != nil {
		t.Fatalf("ObserveOwnership() error = %v", err)
	}
	if !observation.HasWinner || observation.Winner.LeaseID != fixtureLeaseA || observation.Winner.Epoch != 1 {
		t.Fatalf("ObserveOwnership() winner = %+v, want the epoch-1 lease", observation.Winner)
	}
	token, err := fencing.AuthorizeActivation(fixturePresented(), observation)
	if err != nil {
		t.Fatalf("AuthorizeActivation() over observed leases error = %v", err)
	}
	if _, err := token.Bind(fencing.ProviderCapture); err != nil {
		t.Fatalf("observed token bind error = %v", err)
	}
	var _ = errors.Is
}
