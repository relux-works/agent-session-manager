package axpane

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// decideDeps carries the heavy fixtures one table row may need:
// the backend universe (mutable before facts are drawn), a real
// session chain, a real captured checkpoint, and the checkpoint
// store for unpublished captures.
type decideDeps struct {
	universe  *backendUniverse
	repo      *sessrepo.Repository
	recordID  string
	createdID string
	ckpt      *sessckpt.Store
	ckptID    string
	ckptDoc   []byte
}

func buildDecideDeps(t *testing.T, attachAdmitted bool) *decideDeps {
	t.Helper()
	deps := &decideDeps{universe: buildBackendUniverse(t, attachAdmitted)}
	deps.repo, deps.recordID = chainFixture(t)
	events, err := deps.repo.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	deps.createdID = events[len(events)-1].EventID
	ckptStore, err := sessckpt.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	deps.ckpt = ckptStore
	deps.ckptID, deps.ckptDoc = checkpointFixture(t, deps.repo, ckptStore, []string{deps.createdID})
	return deps
}

// validInput returns the ModeLaunch base that launches: every gate
// passes through its landed owner with real documents.
func validInput(t *testing.T, deps *decideDeps) Input {
	t.Helper()
	now := fixtureNow()
	descriptor, err := BuildDescriptor(DescriptorParams{
		Binding: BindingRef{
			BindingDigest:         deps.universe.hostBinding().TerminalBindingID,
			BackendID:             deps.universe.hostBinding().BackendID,
			ImplementationVersion: deps.universe.hostBinding().ImplementationVersion,
			ProtocolVersion:       deps.universe.hostBinding().ProtocolVersion,
			Generation:            deps.universe.hostBinding().Generation,
		},
		InstanceID:  fixtureInstance,
		Interactive: true,
		Columns:     80,
		Rows:        24,
	})
	if err != nil {
		t.Fatalf("BuildDescriptor() error = %v", err)
	}
	profile, err := LoadProfile(deps.repo, deps.ckpt, fixtureSession)
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	return Input{
		SessionID:            fixtureSession,
		BootstrapOperationID: fixtureBootstrap,
		Mode:                 ModeLaunch,
		SessionKnown:         true,
		Presented:            fixturePresented(),
		Observation:          fixtureObservation(now),
		Backend:              deps.universe.facts(t),
		ProfileData:          profile,
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
		Entrypoint:        []string{"ax", "pane", fixtureSession},
		Descriptor:        descriptor,
		HostBinding:       deps.universe.hostBinding(),
		Interactive:       true,
		RemoteInteractive: true,
	}
}

func lapsedObservation(now time.Time) fencing.Observation {
	observation := fixtureObservation(now)
	observation.Grant.ValidatedAt = now.Add(-2 * time.Hour)
	return observation
}

// TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext drives the
// after-restore caller around the fencing.Authorize result: interactive
// remote ownership remains an attach/takeover offer, while a non-interactive
// remote owner parks with the existing remote_owner vocabulary.
func TestDecideLapsedRemoteOwnerOffersOrParksByInteractiveContext(t *testing.T) {
	cases := []struct {
		name              string
		attachAdmitted    bool
		remoteInteractive bool
		mode              Mode
		want              Action
		wantLiteral       string
		wantReason        fencing.ParkReason
		wantReasonLiteral string
	}{
		{name: "restore_interactive_attach", attachAdmitted: true, remoteInteractive: true, mode: ModeRestore, want: ActionAttachRemote, wantLiteral: "attach_remote"},
		{name: "restore_interactive_takeover", attachAdmitted: false, remoteInteractive: true, mode: ModeRestore, want: ActionTakeoverOffer, wantLiteral: "takeover_offer"},
		{name: "restore_noninteractive_park", attachAdmitted: true, remoteInteractive: false, mode: ModeRestore, want: ActionParked, wantLiteral: "parked", wantReason: fencing.ParkRemoteOwner, wantReasonLiteral: "remote_owner"},
		{name: "launch_interactive_attach", attachAdmitted: true, remoteInteractive: true, mode: ModeLaunch, want: ActionAttachRemote, wantLiteral: "attach_remote"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deps := buildDecideDeps(t, tc.attachAdmitted)
			input := validInput(t, deps)
			input.Mode = tc.mode
			input.Observation = lapsedObservation(fixtureNow())
			input.Observation.Winner.HolderHostID = fixtureRemoteHost
			input.RemoteInteractive = tc.remoteInteractive
			decision := Decide(input)
			if decision.Action != tc.want {
				t.Fatalf("Decide() action=%s class=%s reason=%s cause=%v, want %s", decision.Action, decision.Class, decision.ParkReason, decision.Cause, tc.want)
			}
			if string(decision.Action) != tc.wantLiteral {
				t.Fatalf("Decide() action=%q, want literal %q", decision.Action, tc.wantLiteral)
			}
			if tc.wantReason != "" && decision.ParkReason != tc.wantReason {
				t.Fatalf("Decide() park reason=%s, want %s", decision.ParkReason, tc.wantReason)
			}
			if tc.wantReasonLiteral != "" && string(decision.ParkReason) != tc.wantReasonLiteral {
				t.Fatalf("Decide() park reason=%q, want literal %q", decision.ParkReason, tc.wantReasonLiteral)
			}
			t.Logf("lapsed remote owner interactive=%t attach_admitted=%t: action=%s reason=%s cause=%v", tc.remoteInteractive, tc.attachAdmitted, decision.Action, decision.ParkReason, decision.Cause)
		})
	}
}

func TestDecideLaunchPositive(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	decision := Decide(validInput(t, deps))
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() action = %q, detail %q, cause %v, want launch", decision.Action, decision.Detail, decision.Cause)
	}
	if !decision.HasToken {
		t.Fatal("Decide() launch carries no fencing token")
	}
	bound, err := decision.Token.Bind(fencing.ProviderQuiesce)
	if err != nil {
		t.Fatalf("Token.Bind(quiesce) error = %v", err)
	}
	if bound.SessionID != fixtureSession || bound.Epoch != 1 || bound.LeaseID != fixtureLeaseA {
		t.Fatalf("Token.Bind(quiesce) = %+v, want the winning triple", bound)
	}
	if decision.Profile.Profile != "standard" || decision.Profile.HasSource {
		t.Fatalf("Decide() profile = %+v, want standard with no source", decision.Profile)
	}
	if decision.Mapping != "" {
		t.Fatalf("Decide() mapping = %q, want empty for standard", decision.Mapping)
	}
	if !decision.Admitted.Has("headless_creation") {
		t.Fatalf("Decide() admitted = %v, want headless_creation", decision.Admitted.Capabilities)
	}
	if decision.Descriptor.TerminalInstanceID != fixtureInstance {
		t.Fatalf("Decide() descriptor instance = %q, want the supplied instance", decision.Descriptor.TerminalInstanceID)
	}
}

// TestDecideRestoreRequiresRestoreCapability pins the mode-selected
// capability row: a backend admitting restore but not create
// launches in restore mode. Forcing the create row (the N-backend-op
// mutant) refuses capability_unproven instead.
func TestDecideRestoreRequiresRestoreCapability(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	drop := map[string]bool{"durable_disconnect": true, "headless_creation": true}
	keepClaims := func(claims []any) []any {
		kept := make([]any, 0, len(claims))
		for _, raw := range claims {
			if !drop[raw.(map[string]any)["capability"].(string)] {
				kept = append(kept, raw)
			}
		}
		return kept
	}
	deps.universe.manifest["static_capability_claims"] = keepClaims(deps.universe.manifest["static_capability_claims"].([]any))
	deps.universe.probe["capability_claims"] = keepClaims(deps.universe.probe["capability_claims"].([]any))
	keptEvidence := deps.universe.evidence[:0]
	for _, object := range deps.universe.evidence {
		if !drop[object["capability"].(string)] {
			keptEvidence = append(keptEvidence, object)
		}
	}
	deps.universe.evidence = keptEvidence
	var ids []any
	for _, object := range deps.universe.evidence {
		ids = append(ids, object["evidence_id"])
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].(string) < ids[j].(string) })
	deps.universe.probe["evidence_ids"] = ids
	deps.universe.manifest["manifest_id"] = omitSelfIdentity(t, deps.universe.manifest, "manifest_id")
	deps.universe.probe["probe_id"] = omitSelfIdentity(t, deps.universe.probe, "probe_id")
	input := validInput(t, deps)
	input.Mode = ModeRestore
	decision := Decide(input)
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() restore action = %q, cause %v, want launch", decision.Action, decision.Cause)
	}
	if !decision.Admitted.Has("reboot_restoration") {
		t.Fatalf("Decide() admitted = %v, want reboot_restoration", decision.Admitted.Capabilities)
	}
}

func TestDecideYoloMapping(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	appendChainEvent(t, deps.repo, "profile.changed", 1, fixtureLeaseA, 2, deps.createdID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	input := validInput(t, deps)
	decision := Decide(input)
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() action = %q, detail %q, cause %v, want launch", decision.Action, decision.Detail, decision.Cause)
	}
	if decision.Profile.Profile != "yolo" || !decision.Profile.HasSource {
		t.Fatalf("Decide() profile = %+v, want yolo with a source", decision.Profile)
	}
	if decision.Mapping != "--dangerously-bypass-approvals-and-sandbox" {
		t.Fatalf("Decide() mapping = %q, want the codex yolo flag", decision.Mapping)
	}
}

func TestDecideTable(t *testing.T) {
	t.Parallel()
	now := fixtureNow()
	staging := matjournal.Journal{Phase: matjournal.PhaseStaging}

	type row struct {
		name        string
		attachProbe bool
		universe    func(t *testing.T, universe *backendUniverse)
		mutate      func(t *testing.T, deps *decideDeps, input *Input)
		want        Action
		wantClass   string
		wantReason  fencing.ParkReason
		wantLease   string
	}
	rows := []row{
		{
			name: "launch_restore_committed_checkpoint",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Mode = ModeRestore
				input.HasNewestCheckpoint = true
				input.NewestCheckpointID = deps.ckptID
				input.MaterializationRequired = true
				input.Journal = matjournal.Journal{Phase: matjournal.PhaseCommitted, SourceCheckpointID: deps.ckptID, MaterializationID: fixtureMat}
				input.JournalOK = true
				input.CheckpointRequired = true
				input.CheckpointDoc = deps.ckptDoc
				input.CheckpointID = deps.ckptID
			},
			want: ActionLaunch,
		},
		{
			name: "reattach_same_pair",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.ExistingBinding = &Binding{SessionID: fixtureSession, OperationID: fixtureBootstrap, TerminalInstanceID: fixtureInstance, BindingDigest: seedDigest(0xB1), CreatedAt: fixtureCreatedAt}
			},
			want: ActionReattach,
		},
		{
			name: "attach_remote_interactive",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation.Winner.HolderHostID = fixtureRemoteHost
				input.RemoteInteractive = true
			},
			want:      ActionAttachRemote,
			wantLease: fixtureLeaseA,
		},
		{
			name: "takeover_offer_noninteractive",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation.Winner.HolderHostID = fixtureRemoteHost
				input.RemoteInteractive = false
			},
			want:       ActionParked,
			wantReason: fencing.ParkRemoteOwner,
			wantLease:  fixtureLeaseA,
		},
		{
			name:        "takeover_offer_attach_unproven",
			attachProbe: true,
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation.Winner.HolderHostID = fixtureRemoteHost
				input.RemoteInteractive = true
			},
			want:      ActionTakeoverOffer,
			wantLease: fixtureLeaseA,
		},
		{
			name: "refused_config_invalid",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.ConfigErr = errFixtureConfig
			},
			want:      ActionRefused,
			wantClass: ClassInvalidConfig,
		},
		{
			name: "refused_unknown_mode",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Mode = Mode("bogus")
			},
			want:      ActionRefused,
			wantClass: ClassInvalidArguments,
		},
		{
			name: "refused_empty_mode",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Mode = Mode("")
			},
			want:      ActionRefused,
			wantClass: ClassInvalidArguments,
		},
		{
			name: "refused_unknown_session",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.SessionKnown = false
			},
			want:      ActionRefused,
			wantClass: ClassInvalidArguments,
		},
		{
			name: "refused_idempotency_mismatch",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				// This pair is unrecorded on a bound session
				// inside the window (the fold names no newest
				// checkpoint): a changed operation refuses.
				input.SessionBound = true
			},
			want:      ActionRefused,
			wantClass: terminalbackend.CodeIdempotencyMismatch,
		},
		{
			name: "refused_unknown_caller",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Realm.Caller = "batch"
			},
			want:      ActionRefused,
			wantClass: ClassInvalidArguments,
		},
		{
			name: "refused_backend_generation_mismatch",
			universe: func(t *testing.T, universe *backendUniverse) {
				universe.probe["backend_generation_digest"] = seedDigest(0x99)
				universe.probe["probe_id"] = omitSelfIdentity(t, universe.probe, "probe_id")
			},
			want:      ActionRefused,
			wantClass: terminalbackend.CodeStaleGeneration,
		},
		{
			name: "refused_backend_protocol_membership",
			universe: func(t *testing.T, universe *backendUniverse) {
				universe.probe["protocol_version"] = "2.0.0"
				universe.probe["probe_id"] = omitSelfIdentity(t, universe.probe, "probe_id")
			},
			want:      ActionRefused,
			wantClass: terminalbackend.CodeMismatch,
		},
		{
			name: "refused_backend_evidence_expired",
			universe: func(t *testing.T, universe *backendUniverse) {
				universe.now = time.Date(2028, time.January, 20, 0, 0, 0, 0, time.UTC)
			},
			want:      ActionRefused,
			wantClass: terminalbackend.CodeMismatch,
		},
		{
			name: "refused_backend_signature_untrusted",
			universe: func(t *testing.T, universe *backendUniverse) {
				universe.verify = func(issuerID string, message, signature []byte) error {
					return errFixtureConfig
				}
			},
			want:      ActionRefused,
			wantClass: terminalbackend.CodeIntegrityFailure,
		},
		{
			name: "refused_backend_capability_unproven",
			universe: func(t *testing.T, universe *backendUniverse) {
				// create is conferred by durable_disconnect and
				// headless_creation; both must stay unadmitted
				// for the operation to be unproven. Neither is
				// claimed at all, so no evidence rule fires and
				// the admitted set simply confers no create.
				drop := map[string]bool{"durable_disconnect": true, "headless_creation": true}
				keepClaims := func(claims []any) []any {
					kept := make([]any, 0, len(claims))
					for _, raw := range claims {
						if !drop[raw.(map[string]any)["capability"].(string)] {
							kept = append(kept, raw)
						}
					}
					return kept
				}
				universe.manifest["static_capability_claims"] = keepClaims(universe.manifest["static_capability_claims"].([]any))
				universe.probe["capability_claims"] = keepClaims(universe.probe["capability_claims"].([]any))
				keptEvidence := universe.evidence[:0]
				for _, object := range universe.evidence {
					if !drop[object["capability"].(string)] {
						keptEvidence = append(keptEvidence, object)
					}
				}
				universe.evidence = keptEvidence
				var ids []any
				for _, object := range universe.evidence {
					ids = append(ids, object["evidence_id"])
				}
				sort.Slice(ids, func(i, j int) bool { return ids[i].(string) < ids[j].(string) })
				universe.probe["evidence_ids"] = ids
				universe.manifest["manifest_id"] = omitSelfIdentity(t, universe.manifest, "manifest_id")
				universe.probe["probe_id"] = omitSelfIdentity(t, universe.probe, "probe_id")
			},
			want:      ActionRefused,
			wantClass: terminalbackend.CodeCapabilityUnproven,
		},
		{
			name: "refused_fencing_malformed_token",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Presented.Epoch = 0
			},
			want:      ActionRefused,
			wantClass: ClassInvalidArguments,
		},
		{
			name: "refused_fencing_foreign_session",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Presented.SessionID = fixtureForeign
			},
			want:      ActionRefused,
			wantClass: "lease_conflict",
		},
		{
			name: "refused_fencing_expired_grant",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation = lapsedObservation(now)
			},
			want:      ActionRefused,
			wantClass: "lease_conflict",
		},
		{
			name: "refused_provider_tuple_qwen",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Provider.Build = provhost.BuildTuple{ProviderID: "qwen", ProviderVersion: "1.0.0", Platform: "linux", Architecture: "amd64"}
			},
			want:      ActionRefused,
			wantClass: "invalid_config",
		},
		{
			name: "launch_discovery_bound",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Provider.HasDiscovery = true
				input.Provider.Discovery = []byte(`{"native_session_id":"native-session-alpha","discovered":true,"discovery_root":"/home/test/.codex/sessions/native-session-alpha","backend_resolved":false}`)
				input.Provider.DiscoveryContext = provhost.DiscoveryContext{
					Build: fixtureBuild(), Home: "/home/test",
				}
			},
			want: ActionLaunch,
		},
		{
			name: "refused_discovery_native_mismatch",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Provider.HasDiscovery = true
				input.Provider.Discovery = []byte(`{"native_session_id":"native-session-other","discovered":true,"discovery_root":"/home/test/.codex/sessions/native-session-other","backend_resolved":false}`)
				input.Provider.DiscoveryContext = provhost.DiscoveryContext{
					Build: fixtureBuild(), Home: "/home/test",
				}
			},
			want:      ActionRefused,
			wantClass: "invalid_config",
		},
		{
			name: "refused_discovery_root_outside",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Provider.HasDiscovery = true
				input.Provider.Discovery = []byte(`{"native_session_id":"native-session-alpha","discovered":true,"discovery_root":"/tmp/evil","backend_resolved":false}`)
				input.Provider.DiscoveryContext = provhost.DiscoveryContext{
					Build: fixtureBuild(), Home: "/home/test",
				}
			},
			want:      ActionRefused,
			wantClass: "invalid_config",
		},
		{
			name: "refused_discovery_empty_home",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Provider.HasDiscovery = true
				input.Provider.Discovery = []byte(`{"native_session_id":"native-session-alpha","discovered":true,"discovery_root":"/home/test/.codex/sessions/native-session-alpha","backend_resolved":false}`)
				input.Provider.DiscoveryContext = provhost.DiscoveryContext{
					Build: fixtureBuild(), Home: "",
				}
			},
			want:      ActionRefused,
			wantClass: "invalid_config",
		},
		{
			name: "refused_provider_identity_other_provider",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				record, err := provhost.CreateIdentity(provhost.IdentityParams{
					SessionID: fixtureSession, ProviderID: "claude", ProviderVersion: "0.147.0",
					ProviderVersionRange: "opaque-range", NativeSessionID: "native-session-alpha",
					IdentityKind: "session_uuid", LogicalWorkspaceID: fixtureLocalHost,
					CreatedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt,
				})
				if err != nil {
					t.Fatalf("CreateIdentity() error = %v", err)
				}
				input.Provider.Identity = record
			},
			want:      ActionRefused,
			wantClass: "invalid_config",
		},
		{
			name: "refused_smoke_failing_verdict",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Smoke.Required = true
				input.Smoke.Record = smokeRecord(t, "fail", "A", []any{
					map[string]any{"name": "resume-cell", "outcome": "pass", "detail": "cell A"},
					map[string]any{"name": "probe", "outcome": "fail", "detail": "probe refused"},
				}, smokeTuple())
				input.Smoke.Target = smokeTarget()
			},
			want:      ActionRefused,
			wantClass: "target_auth_missing",
		},
		{
			name: "refused_smoke_absent",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Smoke.Required = true
				input.Smoke.Record = nil
				input.Smoke.Target = smokeTarget()
			},
			want:      ActionRefused,
			wantClass: "target_auth_missing",
		},
		{
			name: "refused_smoke_wrong_build",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Smoke.Required = true
				tuple := smokeTuple()
				tuple["provider_version"] = "9.9.9"
				input.Smoke.Record = smokeRecord(t, "pass", "A", passingSmokeChecks(), tuple)
				input.Smoke.Target = smokeTarget()
			},
			want:      ActionRefused,
			wantClass: "target_auth_missing",
		},
		{
			name: "refused_profile_mapping_unavailable",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Provider.Build = provhost.BuildTuple{ProviderID: "pi", ProviderVersion: "0.0.0", Platform: "linux", Architecture: "amd64"}
				record, err := provhost.CreateIdentity(provhost.IdentityParams{
					SessionID: fixtureSession, ProviderID: "pi", ProviderVersion: "0.0.0",
					ProviderVersionRange: "opaque-range", NativeSessionID: "native-session-alpha",
					IdentityKind: "session_uuid", LogicalWorkspaceID: fixtureLocalHost,
					CreatedByHostID: fixtureLocalHost, CreatedAt: fixtureCreatedAt,
				})
				if err != nil {
					t.Fatalf("CreateIdentity() error = %v", err)
				}
				input.Provider.Identity = record
			},
			want:      ActionRefused,
			wantClass: ClassProfileMappingUnavailable,
		},
		{
			name: "refused_profile_derivation_corrupt",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.ProfileData.Record.Creation = "bogus"
			},
			want:      ActionRefused,
			wantClass: ClassIntegrityFailure,
		},
		{
			name: "refused_realm_background_no_server",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Realm.Caller = CallerBackground
			},
			want:      ActionRefused,
			wantClass: "capability_unavailable",
		},
		{
			name: "refused_realm_background_incomplete_details",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Realm.Caller = CallerBackground
				input.Realm.BrokerState = ""
			},
			want:      ActionRefused,
			wantClass: ClassInvalidArguments,
		},
		{
			name: "launch_background_attested_server",
			universe: func(t *testing.T, universe *backendUniverse) {
				addRealmEvidence(t, universe, seedDigest(0xB1), "codex", "0.147.0")
			},
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Realm.Caller = CallerBackground
			},
			want: ActionLaunch,
		},
		{
			name: "refused_entrypoint_raw_provider",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Entrypoint = []string{"codex", "--yolo"}
			},
			want:      ActionRefused,
			wantClass: terminalbackend.CodePreconditionFailed,
		},
		{
			name: "refused_descriptor_generation_drift",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				var document map[string]any
				raw := append([]byte(nil), input.Descriptor...)
				if err := json.Unmarshal(raw, &document); err != nil {
					t.Fatalf("unmarshal descriptor: %v", err)
				}
				document["backend_generation"] = "generation-beta"
				remarshaled, err := json.Marshal(document)
				if err != nil {
					t.Fatalf("marshal descriptor: %v", err)
				}
				input.Descriptor = remarshaled
			},
			want:      ActionRefused,
			wantClass: terminalbackend.CodeStaleGeneration,
		},
		{
			name: "parked_lower_epoch",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation.Winner.Epoch = 2
				input.Presented.Epoch = 1
				input.Presented.LeaseID = fixtureLeaseB
			},
			want:       ActionParked,
			wantReason: fencing.ParkStaleOwner,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_same_epoch_loser",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Presented.LeaseID = fixtureLeaseB
			},
			want:       ActionParked,
			wantReason: fencing.ParkStaleOwner,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_future_epoch",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Presented.Epoch = 5
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_absent_winner",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation.HasWinner = false
				input.Observation.Winner = sessrepo.LeaseSummary{}
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  "",
		},
		{
			name: "parked_ambiguous",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation.Ambiguous = true
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_unverified",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation.Verified = false
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_failed_handoff",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.Observation.HandoffFailed = true
			},
			want:       ActionParked,
			wantReason: fencing.ParkFailedHandoff,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_materialization_not_committed",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.HasNewestCheckpoint = true
				input.NewestCheckpointID = deps.ckptID
				input.MaterializationRequired = true
				input.Journal = matjournal.Journal{Phase: staging.Phase, SourceCheckpointID: deps.ckptID, MaterializationID: fixtureMat}
				input.JournalOK = true
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_materialization_absent",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.MaterializationRequired = true
				input.JournalOK = false
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_checkpoint_inadmissible",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.CheckpointRequired = true
				input.CheckpointDoc = []byte(`{"schema":"urn:ax:schema:checkpoint"}`)
				input.CheckpointID = seedDigest(0xEE)
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_checkpoint_absent",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.CheckpointRequired = true
				input.CheckpointDoc = nil
				input.CheckpointID = seedDigest(0xEE)
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_checkpoint_wrong_identity",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.CheckpointRequired = true
				input.CheckpointDoc = deps.ckptDoc
				input.CheckpointID = seedDigest(0xEE)
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
		{
			name: "parked_checkpoint_flipped_digest",
			mutate: func(t *testing.T, deps *decideDeps, input *Input) {
				input.CheckpointRequired = true
				input.CheckpointDoc = deps.ckptDoc
				flipped := deps.ckptID[:len(deps.ckptID)-1] + "0"
				if flipped == deps.ckptID {
					flipped = deps.ckptID[:len(deps.ckptID)-1] + "1"
				}
				input.CheckpointID = flipped
				// The fold names the flipped digest as newest:
				// only the identity comparison parks, so the
				// prefix mutant escapes every downstream arm.
				input.HasNewestCheckpoint = true
				input.NewestCheckpointID = flipped
			},
			want:       ActionParked,
			wantReason: fencing.ParkRestorePolicy,
			wantLease:  fixtureLeaseA,
		},
	}
	for _, testRow := range rows {
		t.Run(testRow.name, func(t *testing.T) {
			t.Parallel()
			deps := buildDecideDeps(t, !testRow.attachProbe)
			if testRow.universe != nil {
				testRow.universe(t, deps.universe)
			}
			input := validInput(t, deps)
			if testRow.mutate != nil {
				testRow.mutate(t, deps, &input)
			}
			decision := Decide(input)
			if decision.Action != testRow.want {
				t.Fatalf("Decide() action = %q, class %q, reason %q, detail %q, cause %v, want %q",
					decision.Action, decision.Class, decision.ParkReason, decision.Detail, decision.Cause, testRow.want)
			}
			switch testRow.want {
			case ActionRefused:
				if decision.Class != testRow.wantClass {
					t.Fatalf("Decide() class = %q, want %q (cause %v)", decision.Class, testRow.wantClass, decision.Cause)
				}
				if decision.Cause == nil {
					t.Fatal("Decide() refusal carries no cause")
				}
				if decision.HasToken {
					t.Fatal("Decide() refusal mints a fencing token")
				}
			case ActionParked:
				if decision.ParkReason != testRow.wantReason || decision.WinningLeaseID != testRow.wantLease {
					t.Fatalf("Decide() park = (%q, %q), want (%q, %q)",
						decision.ParkReason, decision.WinningLeaseID, testRow.wantReason, testRow.wantLease)
				}
				if decision.Cause == nil {
					t.Fatal("Decide() park carries no cause")
				}
				if decision.HasToken {
					t.Fatal("Decide() park mints a fencing token")
				}
			case ActionAttachRemote, ActionTakeoverOffer:
				if decision.ParkReason != fencing.ParkRemoteOwner || decision.WinningLeaseID != testRow.wantLease {
					t.Fatalf("Decide() offer = (%q, %q), want (remote_owner, %q)",
						decision.ParkReason, decision.WinningLeaseID, testRow.wantLease)
				}
				if decision.HasToken {
					t.Fatal("Decide() offer mints a fencing token")
				}
			case ActionLaunch, ActionReattach:
				if !decision.HasToken {
					t.Fatal("Decide() launch-path outcome carries no fencing token")
				}
			}
		})
	}
}

func smokeTarget() axerror.TargetAuth {
	return axerror.TargetAuth{
		ProviderID:           "codex",
		ProviderBuild:        "0.147.0",
		MacOSVersion:         "14.5",
		TmuxServerGeneration: "generation-alpha",
		Remediation:          "rerun provider-auth smoke",
	}
}

// TestDecideAfterRestoreSequence pins the §4.2 after-restore steps
// 1-5 as a scenario table over the restore mode: read the local
// lease, attempt the bounded mesh refresh, resume locally only with
// the winning lease and a valid required materialization, offer
// remote attach/takeover for a remote interactive owner, and park
// in all other cases.
func TestDecideAfterRestoreSequence(t *testing.T) {
	t.Parallel()
	t.Run("step3_resume_winning_lease_valid_materialization", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Mode = ModeRestore
		input.HasNewestCheckpoint = true
		input.NewestCheckpointID = deps.ckptID
		input.MaterializationRequired = true
		input.Journal = matjournal.Journal{Phase: matjournal.PhaseCommitted, SourceCheckpointID: deps.ckptID, MaterializationID: fixtureMat}
		input.JournalOK = true
		input.CheckpointRequired = true
		input.CheckpointDoc = deps.ckptDoc
		input.CheckpointID = deps.ckptID
		decision := Decide(input)
		if decision.Action != ActionLaunch {
			t.Fatalf("after-restore step 3 action = %q, cause %v, want launch", decision.Action, decision.Cause)
		}
	})
	t.Run("step3_materializing_parks", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Mode = ModeRestore
		input.MaterializationRequired = true
		input.Journal = matjournal.Journal{Phase: matjournal.PhaseValidating}
		input.JournalOK = true
		decision := Decide(input)
		if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
			t.Fatalf("after-restore materializing action = %q (%q), want parked (restore_policy)", decision.Action, decision.ParkReason)
		}
	})
	t.Run("step4_remote_interactive_attach", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Mode = ModeRestore
		input.Observation.Winner.HolderHostID = fixtureRemoteHost
		input.RemoteInteractive = true
		decision := Decide(input)
		if decision.Action != ActionAttachRemote {
			t.Fatalf("after-restore step 4 action = %q, want attach_remote", decision.Action)
		}
	})
	t.Run("step4_remote_noninteractive_takeover", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Mode = ModeRestore
		input.Observation.Winner.HolderHostID = fixtureRemoteHost
		input.RemoteInteractive = false
		decision := Decide(input)
		if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRemoteOwner {
			t.Fatalf("after-restore step 5 action = %q (%q), want parked (remote_owner)", decision.Action, decision.ParkReason)
		}
	})
	t.Run("step5_unverified_parks", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Mode = ModeRestore
		input.Observation.Verified = false
		decision := Decide(input)
		if decision.Action != ActionParked || decision.ParkReason != fencing.ParkRestorePolicy {
			t.Fatalf("after-restore step 5 action = %q (%q), want parked (restore_policy)", decision.Action, decision.ParkReason)
		}
	})
	t.Run("step1_no_local_lease_parks_without_winner", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Mode = ModeRestore
		input.Observation.HasWinner = false
		input.Observation.Winner = sessrepo.LeaseSummary{}
		decision := Decide(input)
		if decision.Action != ActionParked || decision.WinningLeaseID != "" {
			t.Fatalf("after-restore step 1 action = %q (%q), want parked with no winner", decision.Action, decision.WinningLeaseID)
		}
	})
}

// TestDecideTypedRefusals pins the Structured Error composition of
// the two typed refusals: capability_unavailable carries the §15.3
// realm/readiness details and target_auth_missing the target-auth
// details.
func TestDecideTypedRefusals(t *testing.T) {
	t.Parallel()
	t.Run("capability_unavailable_details", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
			t.Fatalf("Decide() = (%q, %q), want (refused, capability_unavailable)", decision.Action, decision.Class)
		}
		if decision.RealmFailure == nil {
			t.Fatal("Decide() capability_unavailable carries no typed failure")
		}
		if got := string(decision.RealmFailure.Code()); got != "capability_unavailable" {
			t.Fatalf("RealmFailure code = %q", got)
		}
	})
	t.Run("target_auth_missing_details", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Smoke.Required = true
		input.Smoke.Target = smokeTarget()
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "target_auth_missing" {
			t.Fatalf("Decide() = (%q, %q), want (refused, target_auth_missing)", decision.Action, decision.Class)
		}
		if decision.RealmFailure == nil {
			t.Fatal("Decide() target_auth_missing carries no typed failure")
		}
	})
}

// TestRealmCarriesNoCachedEvidence pins the §4.2 cached-evidence
// rule behaviorally: no cached sentinel result, managername
// observation, or bare boolean authorizes resume. A background
// caller with a passing smoke record but no admitted realm row
// refuses; a background caller with a passing smoke record and an
// admitted realm row bound to a stale generation refuses; only the
// admitted row bound to the current generation, host binding, and
// probed build launches.
func TestRealmCarriesNoCachedEvidence(t *testing.T) {
	t.Parallel()
	t.Run("cached_smoke_without_realm_refuses", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		input.Smoke.Required = true
		input.Smoke.Record = smokeRecord(t, "pass", "A", passingSmokeChecks(), smokeTuple())
		input.Smoke.Target = smokeTarget()
		decision := Decide(input)
		if decision.Action != ActionRefused || decision.Class != "capability_unavailable" {
			t.Fatalf("Decide() = (%q, %q), want refused capability_unavailable for cached smoke without realm", decision.Action, decision.Class)
		}
	})
	t.Run("stale_generation_with_realm_refuses", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		addRealmEvidence(t, deps.universe, seedDigest(0xB1), "codex", "0.147.0")
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		input.Realm.ServerGeneration = "generation-STALE-pre-reboot"
		input.Smoke.Required = true
		input.Smoke.Record = smokeRecord(t, "pass", "A", passingSmokeChecks(), smokeTuple())
		target := smokeTarget()
		target.TmuxServerGeneration = "generation-STALE-pre-reboot"
		input.Smoke.Target = target
		decision := Decide(input)
		if decision.Action != ActionRefused {
			t.Fatalf("Decide() action = %q, want refused for stale generation", decision.Action)
		}
	})
	t.Run("admitted_realm_launches", func(t *testing.T) {
		t.Parallel()
		deps := buildDecideDeps(t, true)
		addRealmEvidence(t, deps.universe, seedDigest(0xB1), "codex", "0.147.0")
		input := validInput(t, deps)
		input.Realm.Caller = CallerBackground
		decision := Decide(input)
		if decision.Action != ActionLaunch {
			t.Fatalf("Decide() action = %q, cause %v, want launch for admitted realm", decision.Action, decision.Cause)
		}
	})
}

// TestDecideForgedTokensRefuse pins the token side of every
// non-launch outcome: parked, refused, and offer decisions carry
// the zero token, and the zero token refuses every provider
// operation through the fencing owner.
func TestDecideForgedTokensRefuse(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	input := validInput(t, deps)
	input.Observation.Verified = false
	decision := Decide(input)
	if decision.Action != ActionParked || decision.HasToken {
		t.Fatalf("Decide() = (%q, hasToken %v), want parked with no token", decision.Action, decision.HasToken)
	}
	for _, operation := range []fencing.ProviderOperation{fencing.ProviderQuiesce, fencing.ProviderCapture, fencing.ProviderMaterialize} {
		if _, err := decision.Token.Bind(operation); err == nil {
			t.Fatalf("Token.Bind(%q) on a parked decision succeeded, want refusal", string(operation))
		}
	}
}

// TestDecideProfileSourcePinsEvent pins the PROFILE-DIRECT-RESUME
// direction: the launch carries the newest authoritative
// profile.changed event as its source, never the creation value,
// when the closure contains a later change.
func TestDecideProfileSourcePinsEvent(t *testing.T) {
	t.Parallel()
	deps := buildDecideDeps(t, true)
	changed := appendChainEvent(t, deps.repo, "profile.changed", 1, fixtureLeaseA, 2, deps.createdID, map[string]any{
		"from": "standard", "to": "yolo", "confirmed": true,
	})
	input := validInput(t, deps)
	decision := Decide(input)
	if decision.Action != ActionLaunch {
		t.Fatalf("Decide() action = %q, cause %v, want launch", decision.Action, decision.Cause)
	}
	if !decision.Profile.HasSource || decision.Profile.Source != changed {
		t.Fatalf("Decide() profile source = %+v, want the change event %s", decision.Profile, changed)
	}
}
