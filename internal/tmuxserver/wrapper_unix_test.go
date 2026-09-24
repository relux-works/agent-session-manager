//go:build !windows

package tmuxserver

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/gowebpki/jcs"

	"github.com/relux-works/agent-session-manager/internal/axpane"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/sessprofile"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// This file drives the §4.2 after-restore composition at the wrapper
// entry (rev2 P1-D): the order (refresh before decision), the resume
// and offer effects, the parks, the malformed-input refusals, and the
// measured refresh bound. The decision input mirrors the landed
// wrapper owner's own fixtures with the lifecycle identities, so
// every branch runs through the real Decide arms — no arm is stubbed.

// --- backend universe mirror (admitBackend input) ---

func wClaimRows() map[string]map[string]any {
	return map[string]map[string]any{
		"durable_disconnect": {
			"generation_variable":   false,
			"dependent_operations":  []any{"create", "status"},
			"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
		},
		"headless_creation": {
			"generation_variable":   true,
			"dependent_operations":  []any{"create"},
			"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
		},
		"reboot_restoration": {
			"generation_variable":   true,
			"dependent_operations":  []any{"restore"},
			"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
		},
		"local_attach": {
			"generation_variable":   true,
			"dependent_operations":  []any{"attach"},
			"evidence_requirements": []any{"conformance_fixture", "policy_authorization", "runtime_probe"},
		},
	}
}

func wClaimMap(capability, origin string, value bool) map[string]any {
	row := wClaimRows()[capability]
	return map[string]any{
		"capability":            capability,
		"origin":                origin,
		"value":                 value,
		"generation_variable":   row["generation_variable"],
		"dependent_operations":  row["dependent_operations"],
		"evidence_requirements": row["evidence_requirements"],
	}
}

func wOmitSelfIdentity(t *testing.T, object map[string]any, selfField string) string {
	t.Helper()
	omitted := make(map[string]any, len(object))
	for name, member := range object {
		if name != selfField {
			omitted[name] = member
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		t.Fatalf("marshal omit-self object: %v", err)
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		t.Fatalf("canonicalize omit-self object: %v", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func wEvidenceMessage(t *testing.T, object map[string]any) []byte {
	t.Helper()
	omitted := make(map[string]any, len(object))
	for name, member := range object {
		if name != "evidence_id" && name != "attestation_signature" {
			omitted[name] = member
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		t.Fatalf("marshal unsigned evidence: %v", err)
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		t.Fatalf("canonicalize unsigned evidence: %v", err)
	}
	message := append([]byte("ax-terminal-capability-evidence-v1"), 0x00)
	return append(message, canonical...)
}

func wMustJSON(t *testing.T, object map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

type wBackendUniverse struct {
	rawGeneration string
	manifest      map[string]any
	probe         map[string]any
	evidence      []map[string]any
	verify        terminalbackend.SignatureVerifier
	now           time.Time
}

func wBuildBackendUniverse(t *testing.T, attachAdmitted bool) *wBackendUniverse {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate attestation key: %v", err)
	}
	universe := &wBackendUniverse{
		rawGeneration: lxGeneration,
		now:           lxNow(),
		verify: func(issuerID string, message, signature []byte) error {
			digest := sha256.Sum256(message)
			return rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature)
		},
	}
	generationDigest, err := terminalbackend.GenerationDigest(universe.rawGeneration)
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	fixtureID := lxDigestC
	universe.manifest = map[string]any{
		"schema":                 terminalbackend.SchemaManifest,
		"schema_version":         "1.0.0",
		"manifest_id":            "",
		"terminal_backend_id":    terminalbackend.BuiltinTmux,
		"implementation_version": "2.1.0",
		"protocol_versions":      []any{"1.0.0", "1.1.0"},
		"platforms":              []any{"linux", "macos", "wsl2"},
		"implementation_kind":    "builtin_go",
		"executable_digest":      nil,
		"static_capability_claims": []any{
			wClaimMap("durable_disconnect", "static", true),
			wClaimMap("headless_creation", "static", true),
			wClaimMap("local_attach", "static", true),
			wClaimMap("reboot_restoration", "static", true),
		},
		"conformance_fixture_id": fixtureID,
		"extensions":             map[string]any{},
	}
	universe.manifest["manifest_id"] = wOmitSelfIdentity(t, universe.manifest, "manifest_id")
	probeClaims := []any{
		wClaimMap("durable_disconnect", "static", true),
		wClaimMap("headless_creation", "probed", true),
		wClaimMap("local_attach", "probed", attachAdmitted),
		wClaimMap("reboot_restoration", "probed", true),
	}
	trueClaims := []string{"durable_disconnect", "headless_creation", "reboot_restoration"}
	if attachAdmitted {
		trueClaims = append(trueClaims, "local_attach")
	}
	universe.probe = map[string]any{
		"schema":                    terminalbackend.SchemaProbe,
		"schema_version":            "1.0.0",
		"probe_id":                  "",
		"terminal_backend_id":       terminalbackend.BuiltinTmux,
		"implementation_version":    "2.1.0",
		"protocol_version":          "1.1.0",
		"implementation_kind":       "builtin_go",
		"executable_digest":         nil,
		"platform":                  "linux",
		"os_version":                "14.5",
		"availability":              "available",
		"backend_generation_digest": generationDigest,
		"capability_claims":         probeClaims,
		"evidence_ids":              []any{},
		"probed_at":                 "2026-01-15T12:00:00.000Z",
		"extensions":                map[string]any{},
	}
	issuerID := lxDigestA
	var ids []string
	for _, capability := range trueClaims {
		facts := []any{"fixture_passed", "runtime_probe_passed"}
		if capability == "local_attach" {
			facts = []any{"fixture_passed", "policy_checked", "runtime_probe_passed"}
		}
		object := map[string]any{
			"schema":                     terminalbackend.SchemaCapabilityEvidence,
			"schema_version":             "1.0.0",
			"evidence_id":                "",
			"terminal_backend_id":        terminalbackend.BuiltinTmux,
			"implementation_version":     "2.1.0",
			"protocol_version":           "1.1.0",
			"backend_generation_digest":  generationDigest,
			"capability":                 capability,
			"value":                      true,
			"platform":                   "linux",
			"os_version":                 "14.5",
			"conformance_fixture_id":     fixtureID,
			"observed_at":                "2025-06-01T00:00:00.000Z",
			"expires_at":                 "2027-06-01T00:00:00.000Z",
			"issuer":                     terminalbackend.IssuerLocalProbe,
			"issuer_id":                  issuerID,
			"attestation_signature":      "",
			"facts":                      facts,
			"terminal_binding_id":        nil,
			"provider_id":                nil,
			"provider_build":             nil,
			"sentinel_result":            nil,
			"provider_auth_smoke_result": nil,
			"extensions":                 map[string]any{},
		}
		message := wEvidenceMessage(t, object)
		digest := sha256.Sum256(message)
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if err != nil {
			t.Fatalf("sign evidence: %v", err)
		}
		object["attestation_signature"] = "rsa-sha256:" + base64.StdEncoding.EncodeToString(signature)
		object["evidence_id"] = wOmitSelfIdentity(t, object, "evidence_id")
		universe.evidence = append(universe.evidence, object)
		ids = append(ids, object["evidence_id"].(string))
	}
	sort.Strings(ids)
	asValues := make([]any, 0, len(ids))
	for _, id := range ids {
		asValues = append(asValues, id)
	}
	universe.probe["evidence_ids"] = asValues
	universe.probe["probe_id"] = wOmitSelfIdentity(t, universe.probe, "probe_id")
	return universe
}

func (universe *wBackendUniverse) facts(t *testing.T) axpane.BackendFacts {
	t.Helper()
	evidence := make([][]byte, 0, len(universe.evidence))
	for _, object := range universe.evidence {
		evidence = append(evidence, wMustJSON(t, object))
	}
	return axpane.BackendFacts{
		Manifest:      wMustJSON(t, universe.manifest),
		Probe:         wMustJSON(t, universe.probe),
		Evidence:      evidence,
		RawGeneration: universe.rawGeneration,
		Verify:        universe.verify,
		Now:           universe.now,
	}
}

func (universe *wBackendUniverse) hostBinding() terminalbackend.InstanceBinding {
	return terminalbackend.InstanceBinding{
		BackendID:             terminalbackend.BuiltinTmux,
		ImplementationVersion: "2.1.0",
		ProtocolVersion:       "1.1.0",
		Generation:            universe.rawGeneration,
		TerminalBindingID:     lxDigestB,
	}
}

// --- wrapper decision input ---

func wBuild() provhost.BuildTuple {
	return provhost.BuildTuple{
		ProviderID:      "codex",
		ProviderVersion: "0.147.0",
		Platform:        "linux",
		Architecture:    "amd64",
	}
}

func wIdentity(t *testing.T) []byte {
	t.Helper()
	record, err := provhost.CreateIdentity(provhost.IdentityParams{
		SessionID:            lxSession,
		ProviderID:           "codex",
		ProviderVersion:      "0.147.0",
		ProviderVersionRange: "opaque-range",
		NativeSessionID:      "native-session-lx",
		IdentityKind:         "session_uuid",
		LogicalWorkspaceID:   lxHost,
		CreatedByHostID:      lxHost,
		CreatedAt:            lxIssued,
	})
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
	return record
}

// wDecideInput returns the ModeRestore decision input that launches
// over a local win: every gate passes through its landed owner with
// real documents, and the tests mutate one arm at a time.
func wDecideInput(t *testing.T, attachAdmitted bool) axpane.Input {
	t.Helper()
	universe := wBuildBackendUniverse(t, attachAdmitted)
	hostBinding := universe.hostBinding()
	descriptor, err := axpane.BuildDescriptor(axpane.DescriptorParams{
		Binding: axpane.BindingRef{
			BindingDigest:         hostBinding.TerminalBindingID,
			BackendID:             hostBinding.BackendID,
			ImplementationVersion: hostBinding.ImplementationVersion,
			ProtocolVersion:       hostBinding.ProtocolVersion,
			Generation:            hostBinding.Generation,
		},
		InstanceID:  lxInstance,
		Interactive: true,
		Columns:     80,
		Rows:        24,
	})
	if err != nil {
		t.Fatalf("BuildDescriptor() error = %v", err)
	}
	return axpane.Input{
		SessionID:               lxSession,
		BootstrapOperationID:    lxBootstrap,
		Mode:                    axpane.ModeRestore,
		SessionKnown:            true,
		SessionBound:            false,
		Presented:               fencing.PresentedToken{SessionID: lxSession, Epoch: 7, LeaseID: lxLease},
		Backend:                 universe.facts(t),
		MaterializationRequired: true,
		Journal: matjournal.Journal{
			Phase:              matjournal.PhaseCommitted,
			MaterializationID:  "mat-lx-1",
			SourceCheckpointID: "ckpt-lx-newest",
		},
		JournalOK:           true,
		HasNewestCheckpoint: true,
		NewestCheckpointID:  "ckpt-lx-newest",
		CheckpointRequired:  false,
		ProfileData: sessprofile.Derivation{
			Record: sessprofile.Record{SessionID: lxSession, RecordID: "profile-record-lx", Creation: sessprofile.ProfileStandard},
		},
		Provider:          axpane.ProviderFacts{Build: wBuild(), Identity: wIdentity(t)},
		Smoke:             axpane.SmokeFacts{Required: false},
		Realm:             axpane.Realm{Caller: axpane.CallerForeground, BrokerState: "broker-running", ServerGeneration: lxGeneration, Remediation: "launch the aqua broker"},
		Entrypoint:        []string{"ax", "pane", lxSession},
		Descriptor:        descriptor,
		HostBinding:       hostBinding,
		Interactive:       true,
		RemoteInteractive: true,
	}
}

// wRestoreRequest is the backend restore request the resume branches
// execute: the recorded prior under the fixture generation.
func wRestoreRequest(t *testing.T) OpRequest {
	t.Helper()
	return OpRequest{
		Operation: "restore", Body: lxRestoreBody(t, lxDigestA, nil),
		Source: "absent", Admitted: fullLifecycleAdmitted(),
	}
}

const (
	wFreshValidatedAt  = "2026-09-01T11:59:00.000Z"
	wLapsedValidatedAt = "2026-09-01T10:00:00.000Z"
)

func wGrant(t *testing.T, validatedAt string) sessrepo.FencingGrant {
	t.Helper()
	at, err := time.Parse(time.RFC3339Nano, validatedAt)
	if err != nil {
		t.Fatal(err)
	}
	return sessrepo.FencingGrant{SessionID: lxSession, RecordID: "grant-lx-1", ValidatedAt: at}
}

func wPolicy() sessrepo.FencingPolicy {
	return sessrepo.FencingPolicy{RefreshInterval: time.Hour}
}

// --- P1-D composition tests ---

// TestWrapperRestoreLapsedGrantRemoteOffer is the P1-D core: a lapsed
// local grant under a refreshed remote winner offers remote attach
// (or takeover when attach is not capable) without running any
// backend operation. The refresh runs before the decision, so the
// offer path never presents the lapsed grant to backend restore
// authorization — backend authorization stays intact and is never
// consulted here.
func TestWrapperRestoreLapsedGrantRemoteOffer(t *testing.T) {
	cases := []struct {
		name           string
		attachAdmitted bool
		want           axpane.Action
	}{
		{"attach", true, axpane.ActionAttachRemote},
		{"takeover", false, axpane.ActionTakeoverOffer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newLifecycleFixture(t, fullLifecycleAdmitted())
			fx.lc.LocalHostID = lxHost
			refreshed := false
			fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
				refreshed = true
				if session != lxSession {
					t.Errorf("refresh session = %s", session)
				}
				return RefreshWinner{LeaseID: lxLeaseB, Epoch: 9, HolderHostID: lxRemoteHost}, nil
			}
			outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
				Decide:   wDecideInput(t, tc.attachAdmitted),
				Grant:    wGrant(t, wLapsedValidatedAt),
				HasGrant: true,
				Policy:   wPolicy(),
				Restore:  wRestoreRequest(t),
			})
			if err != nil {
				t.Fatal(err)
			}
			if !refreshed {
				t.Fatal("decision ran without the mesh refresh")
			}
			if outcome.Decision.Action != tc.want {
				t.Fatalf("action = %s (%v), want %s", outcome.Decision.Action, outcome.Decision.Cause, tc.want)
			}
			if !outcome.RefreshAttempted || !outcome.RefreshSucceeded {
				t.Fatalf("refresh audit = %+v", outcome)
			}
			if outcome.WinnerLeaseID != lxLeaseB || outcome.WinnerHostID != lxRemoteHost {
				t.Fatalf("winner = %+v", outcome)
			}
			if outcome.Restore != nil {
				t.Fatalf("offer executed backend restore: %+v", outcome.Restore)
			}
			if fx.runner.callCount() != 0 {
				t.Fatalf("offer executed %v", fx.runner.subcommands())
			}
		})
	}
}

// TestWrapperRestoreLocalWinResumes pins the resume effect: a
// verified local win with valid materialization launches, and the
// launch executes the backend restore exactly once (the wrapper
// recreates; the bindings prove it).
func TestWrapperRestoreLocalWinResumes(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide:   wDecideInput(t, true),
		Grant:    wGrant(t, wFreshValidatedAt),
		HasGrant: true,
		Policy:   wPolicy(),
		Restore:  wRestoreRequest(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want launch", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Restore == nil || outcome.Restore.Binding == nil {
		t.Fatalf("resume omits its restore outcome: %+v", outcome.Restore)
	}
	if !outcome.Restore.RestoredParked {
		t.Fatalf("restore outcome = %+v", outcome.Restore)
	}
	if got := fx.runner.subcommands(); len(got) != 1 || got[0] != "new-session" {
		t.Fatalf("resume executed %v, want exactly one wrapper recreation", got)
	}
}

// TestWrapperRestoreReattachReplaysBackend pins the reattach route: a
// recorded bootstrap pair reattaches through the backend restore,
// which recreates the wrapper when the reboot-volatile receipt is
// gone.
func TestWrapperRestoreReattachReplaysBackend(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	input := wDecideInput(t, true)
	input.SessionBound = true
	pair := recordedAxpaneBinding()
	input.ExistingBinding = &pair
	outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide:   input,
		Grant:    wGrant(t, wFreshValidatedAt),
		HasGrant: true,
		Policy:   wPolicy(),
		Restore:  wRestoreRequest(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Decision.Action != axpane.ActionReattach {
		t.Fatalf("action = %s (%v), want reattach", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Restore == nil || outcome.Restore.Binding == nil {
		t.Fatalf("reattach omits its restore outcome: %+v", outcome.Restore)
	}
	if got := fx.runner.subcommands(); len(got) != 1 || got[0] != "new-session" {
		t.Fatalf("reattach executed %v", got)
	}
}

// TestWrapperRestoreParksWithoutEffects pins every park branch: a
// non-interactive remote owner, a failed refresh (unverified local
// knowledge), a missing refresh adapter, no known lease at all, and
// an invalid materialization each park with no backend operation, no
// tmux exec, and no provider.
func TestWrapperRestoreParksWithoutEffects(t *testing.T) {
	setup := func(t *testing.T, mutate func(fx *lxFixture, input *axpane.Input)) (*lxFixture, axpane.Input) {
		t.Helper()
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		fx.lc.LocalHostID = lxHost
		input := wDecideInput(t, true)
		mutate(fx, &input)
		return fx, input
	}
	cases := []struct {
		name  string
		grant string
		setup func(t *testing.T) (*lxFixture, axpane.Input)
	}{
		{"remote-noninteractive", wLapsedValidatedAt, func(t *testing.T) (*lxFixture, axpane.Input) {
			return setup(t, func(fx *lxFixture, input *axpane.Input) {
				fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
					return RefreshWinner{LeaseID: lxLeaseB, Epoch: 9, HolderHostID: lxRemoteHost}, nil
				}
				input.RemoteInteractive = false
			})
		}},
		{"refresh-failed", wFreshValidatedAt, func(t *testing.T) (*lxFixture, axpane.Input) {
			return setup(t, func(fx *lxFixture, input *axpane.Input) {
				fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
					return RefreshWinner{}, context.DeadlineExceeded
				}
			})
		}},
		{"no-adapter", wFreshValidatedAt, func(t *testing.T) (*lxFixture, axpane.Input) {
			return setup(t, func(fx *lxFixture, input *axpane.Input) {})
		}},
		{"no-known-lease", wFreshValidatedAt, func(t *testing.T) (*lxFixture, axpane.Input) {
			return setup(t, func(fx *lxFixture, input *axpane.Input) {
				fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
					return RefreshWinner{}, context.DeadlineExceeded
				}
				fx.lc.CurrentLease = func() terminstance.LeaseView { return terminstance.LeaseView{} }
			})
		}},
		{"invalid-material", wFreshValidatedAt, func(t *testing.T) (*lxFixture, axpane.Input) {
			return setup(t, func(fx *lxFixture, input *axpane.Input) {
				fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
					return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
				}
				input.JournalOK = false
				// The backend stands ready: a routing mutant that
				// admits this park into execution consumes the
				// vector and sets Restore, failing both pins.
				recordBinding(t, fx, lxDigestA)
				fx.runner.queue("new-session", "", 0)
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx, input := tc.setup(t)
			outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
				Decide:   input,
				Grant:    wGrant(t, tc.grant),
				HasGrant: true,
				Policy:   wPolicy(),
				Restore:  wRestoreRequest(t),
			})
			if err != nil {
				t.Fatal(err)
			}
			if outcome.Decision.Action != axpane.ActionParked {
				t.Fatalf("action = %s (%v), want parked", outcome.Decision.Action, outcome.Decision.Cause)
			}
			if outcome.Restore != nil {
				t.Fatalf("park executed backend restore: %+v", outcome.Restore)
			}
			if fx.runner.callCount() != 0 {
				t.Fatalf("park executed %v", fx.runner.subcommands())
			}
			if tc.name == "invalid-material" {
				if outcome.Decision.Cause == nil ||
					outcome.Decision.Cause.Error() != "required materialization is not admitted" {
					t.Fatalf("cause = %v", outcome.Decision.Cause)
				}
			}
		})
	}
}

// TestWrapperRestoreRefusesMalformed pins the entry's own gates: a
// non-restore mode, a non-restore resume request, and missing
// lifecycle dependencies each refuse before any decision or effect.
func TestWrapperRestoreRefusesMalformed(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	launch := wDecideInput(t, true)
	launch.Mode = axpane.ModeLaunch
	if _, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: launch, Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t),
	}); err == nil {
		t.Fatal("non-restore mode admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "wrapper restore mode")
	}
	statusReq := wRestoreRequest(t)
	statusReq.Operation = "status"
	if _, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: statusReq,
	}); err == nil {
		t.Fatal("non-restore resume request admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "wrapper restore operation")
	}
	// Garbage values stay refused while the narrowing rows admit
	// exactly one shape each (narrowness in the log).
	evilMode := wDecideInput(t, true)
	evilMode.Mode = axpane.Mode("evil")
	if _, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: evilMode, Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t),
	}); err == nil {
		t.Fatal("garbage mode admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "wrapper restore mode")
	}
	evilOp := wRestoreRequest(t)
	evilOp.Operation = "evil-operation"
	if _, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: evilOp,
	}); err == nil {
		t.Fatal("garbage resume operation admitted")
	} else {
		requireLocalCode(t, err, "tmux_invalid_arguments", "wrapper restore operation")
	}
	bare := &Lifecycle{}
	if _, err := bare.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t),
	}); err == nil {
		t.Fatal("bare lifecycle admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "lifecycle dependencies")
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("malformed entries executed %v", fx.runner.subcommands())
	}
}

// TestWrapperRestoreRefreshBoundMeasured pins the measured refresh
// bound: a refresh that blocks until cancellation returns at the
// configured bound (not never), the attempt is audited, a missing
// configuration audits the default bound, and the entry parks over
// unverified local knowledge.
func TestWrapperRestoreRefreshBoundMeasured(t *testing.T) {
	t.Run("configured-50ms", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		fx.lc.LocalHostID = lxHost
		fx.lc.MeshRefreshTimeout = 50 * time.Millisecond
		fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
			<-ctx.Done()
			return RefreshWinner{}, ctx.Err()
		}
		start := time.Now()
		outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
			Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t),
		})
		elapsed := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Decision.Action != axpane.ActionParked {
			t.Fatalf("action = %s (%v), want parked", outcome.Decision.Action, outcome.Decision.Cause)
		}
		if !outcome.RefreshAttempted || outcome.RefreshSucceeded {
			t.Fatalf("refresh audit = %+v", outcome)
		}
		if outcome.RefreshBoundMs != 50 {
			t.Fatalf("bound = %dms, want exactly the configured 50ms", outcome.RefreshBoundMs)
		}
		// The entry enforces the bound: it returns at the
		// deadline instead of waiting out the adapter. (The
		// adapter's own observation is unsynchronized by
		// design — the entry may return while the abandoned
		// adapter is still waking — so only the entry's
		// elapsed time is pinned here.)
		if elapsed >= time.Second {
			t.Fatalf("refresh blocked %v under a 50ms bound", elapsed)
		}
		if outcome.Restore != nil || fx.runner.callCount() != 0 {
			t.Fatalf("timed-out refresh executed: %+v %v", outcome.Restore, fx.runner.subcommands())
		}
		t.Logf("refresh bound 50ms, wall elapsed %v, audit attempted=%v succeeded=%v", elapsed, outcome.RefreshAttempted, outcome.RefreshSucceeded)
	})
	t.Run("default-5000ms", func(t *testing.T) {
		fx := newLifecycleFixture(t, fullLifecycleAdmitted())
		fx.lc.LocalHostID = lxHost
		fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
			return RefreshWinner{}, context.DeadlineExceeded
		}
		outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
			Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t),
		})
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Decision.Action != axpane.ActionParked {
			t.Fatalf("action = %s (%v), want parked", outcome.Decision.Action, outcome.Decision.Cause)
		}
		if outcome.RefreshBoundMs != 5000 {
			t.Fatalf("bound = %dms, want exactly the default 5000ms", outcome.RefreshBoundMs)
		}
	})
}

// wRestoreBodyWithAuth rebuilds the backend restore body carrying a
// restore authorization for one lease tuple instead of the fixture
// (A,7).
func wRestoreBodyWithAuth(t *testing.T, leaseID string, epoch float64) []byte {
	t.Helper()
	return lxRestoreBody(t, lxDigestA, func(object map[string]any) {
		nested, ok := object["context"].(map[string]any)
		if !ok {
			t.Fatal("restore body has no context object")
		}
		auth, ok := nested["authorization"].(map[string]any)
		if !ok {
			t.Fatal("restore context has no authorization object")
		}
		auth["lease_id"] = leaseID
		auth["lease_epoch"] = epoch
	})
}

// wDivergentWinner drives the wrapper with a refreshed winner the
// backend restore authorization does not carry: the decision runs
// over the winner while the restore still names the old lease. Each
// caller asserts the binding refusal.
func wDivergentWinner(t *testing.T, winner RefreshWinner, lease terminstance.LeaseView, authLease string, authEpoch float64) (*lxFixture, WrapperRestoreOutcome, error) {
	t.Helper()
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.CurrentLease = func() terminstance.LeaseView { return lease }
	fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
		return winner, nil
	}
	input := wDecideInput(t, true)
	input.Presented.LeaseID = winner.LeaseID
	input.Presented.Epoch = winner.Epoch
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	restore := wRestoreRequest(t)
	restore.Body = wRestoreBodyWithAuth(t, authLease, authEpoch)
	outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: input, Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: restore,
	})
	return fx, outcome, err
}

// TestWrapperRestoreRefusesDivergentWinner pins the winner binding:
// a refreshed winner for lease B at epoch 9 launches the decision,
// but the backend restore still carries lease A at epoch 7, so the
// entry refuses before any effect instead of executing under the old
// authorization. The decision stays launched — the refusal is the
// join, not the decision — and backend authorization is untouched.
func TestWrapperRestoreRefusesDivergentWinner(t *testing.T) {
	fx, outcome, err := wDivergentWinner(t,
		RefreshWinner{LeaseID: lxLeaseB, Epoch: 9, HolderHostID: lxHost},
		terminstance.LeaseView{LeaseID: lxLease, Epoch: 7}, lxLease, 7)
	if err == nil {
		t.Fatal("winner epoch 9 authorized a restore carrying old epoch 7")
	}
	requireLocalCode(t, err, "local_precondition_failed", "wrapper restore winner binding")
	if outcome.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want the launched decision the binding refuses", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Restore != nil {
		t.Fatalf("divergent restore executed: %+v", outcome.Restore)
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("divergent restore executed %v", fx.runner.subcommands())
	}
}

// TestWrapperRestoreRefusesDivergentWinnerLease isolates the lease
// arm: the epoch agrees (9) while the lease diverges (A vs B). The
// backend would authorize the carried (A,9) against the current
// (A,9) — only the binding refuses.
func TestWrapperRestoreRefusesDivergentWinnerLease(t *testing.T) {
	fx, outcome, err := wDivergentWinner(t,
		RefreshWinner{LeaseID: lxLeaseB, Epoch: 9, HolderHostID: lxHost},
		terminstance.LeaseView{LeaseID: lxLease, Epoch: 9}, lxLease, 9)
	if err == nil {
		t.Fatal("winner lease B authorized a restore carrying lease A")
	}
	requireLocalCode(t, err, "local_precondition_failed", "wrapper restore winner binding")
	if outcome.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want the launched decision the binding refuses", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Restore != nil || fx.runner.callCount() != 0 {
		t.Fatalf("divergent restore executed: %+v %v", outcome.Restore, fx.runner.subcommands())
	}
}

// TestWrapperRestoreRefusesDivergentWinnerEpoch isolates the epoch
// arm: the lease agrees (B) while the epoch diverges (7 vs 9). The
// backend would authorize the carried (B,7) against the current
// (B,7) — only the binding refuses.
func TestWrapperRestoreRefusesDivergentWinnerEpoch(t *testing.T) {
	fx, outcome, err := wDivergentWinner(t,
		RefreshWinner{LeaseID: lxLeaseB, Epoch: 9, HolderHostID: lxHost},
		terminstance.LeaseView{LeaseID: lxLeaseB, Epoch: 7}, lxLeaseB, 7)
	if err == nil {
		t.Fatal("winner epoch 9 authorized a restore carrying epoch 7")
	}
	requireLocalCode(t, err, "local_precondition_failed", "wrapper restore winner binding")
	if outcome.Decision.Action != axpane.ActionLaunch {
		t.Fatalf("action = %s (%v), want the launched decision the binding refuses", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if outcome.Restore != nil || fx.runner.callCount() != 0 {
		t.Fatalf("divergent restore executed: %+v %v", outcome.Restore, fx.runner.subcommands())
	}
}

// TestWrapperRestoreExpiredRefreshParksUnverified pins the expiry
// rejection: an adapter that waits out the deadline and then answers
// a local winner with a nil error loses — the entry decides over
// unverified local knowledge and parks instead of resuming.
func TestWrapperRestoreExpiredRefreshParksUnverified(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.MeshRefreshTimeout = time.Millisecond
	fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
		<-ctx.Done()
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
		Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Decision.Action != axpane.ActionParked {
		t.Fatalf("action = %s (%v), want parked over the expired answer", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if !outcome.RefreshAttempted || outcome.RefreshSucceeded {
		t.Fatalf("refresh audit = %+v, want attempted without success", outcome)
	}
	if outcome.Restore != nil || fx.runner.callCount() != 0 {
		t.Fatalf("expired refresh executed: %+v %v", outcome.Restore, fx.runner.subcommands())
	}
}

// TestWrapperRestoreRefreshBoundEnforced pins the bound against a
// non-cooperative adapter: the callback blocks on the test's
// release channel while ignoring cancellation, and the entry still
// returns at the bound, parks unverified, and runs nothing. The
// still-blocked proof — not a tight wall-clock ceiling — carries
// the verdict: the adapter is provably blocked when the entry
// returns, so no scheduling skew can flip the test, and the
// release afterwards leaves no goroutine behind.
func TestWrapperRestoreRefreshBoundEnforced(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.MeshRefreshTimeout = time.Millisecond
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	finished := make(chan struct{})
	fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
		entered <- struct{}{}
		<-release
		close(finished)
		return RefreshWinner{}, context.DeadlineExceeded
	}
	recordBinding(t, fx, lxDigestA)
	fx.runner.queue("new-session", "", 0)
	type restoreResult struct {
		outcome WrapperRestoreOutcome
		err     error
	}
	done := make(chan restoreResult, 1)
	start := time.Now()
	go func() {
		outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
			Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t),
		})
		done <- restoreResult{outcome: outcome, err: err}
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("refresh adapter never started")
	}
	var res restoreResult
	select {
	case res = <-done:
	case <-time.After(10 * time.Second):
		close(release)
		t.Fatal("entry blocked past 10s on the non-cooperative adapter")
	}
	elapsed := time.Since(start)
	outcome, err := res.outcome, res.err
	select {
	case <-finished:
		t.Fatal("entry waited for the non-cooperative adapter")
	default:
	}
	close(release)
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("released adapter never returned")
	}
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Decision.Action != axpane.ActionParked {
		t.Fatalf("action = %s (%v), want parked", outcome.Decision.Action, outcome.Decision.Cause)
	}
	if !outcome.RefreshAttempted || outcome.RefreshSucceeded {
		t.Fatalf("refresh audit = %+v, want attempted without success", outcome)
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("configured 1ms refresh blocked %v", elapsed)
	}
	if outcome.Restore != nil || fx.runner.callCount() != 0 {
		t.Fatalf("unbounded refresh executed: %+v %v", outcome.Restore, fx.runner.subcommands())
	}
	t.Logf("refresh bound 1ms against a release-controlled cooperative-ignoring adapter, wall elapsed %v", elapsed)
}

// TestWrapperRestoreRefreshRaceRejectsLateAnswer forces the
// both-ready race deterministically: the adapter answers at once
// while the select hook stalls twenty times past the 1ms bound, so
// the answer and the deadline are both ready at every selection. A
// late answer never verifies no matter which arm the select takes —
// twenty consecutive parks, no execution.
func TestWrapperRestoreRefreshRaceRejectsLateAnswer(t *testing.T) {
	fx := newLifecycleFixture(t, fullLifecycleAdmitted())
	fx.lc.LocalHostID = lxHost
	fx.lc.MeshRefreshTimeout = time.Millisecond
	fx.lc.LeaseRefresh = func(ctx context.Context, session string) (RefreshWinner, error) {
		return RefreshWinner{LeaseID: lxLease, Epoch: 7, HolderHostID: lxHost}, nil
	}
	fx.lc.RefreshHooks = &RefreshHooks{BeforeSelect: func() { time.Sleep(20 * time.Millisecond) }}
	recordBinding(t, fx, lxDigestA)
	for i := 0; i < 20; i++ {
		fx.runner.queue("new-session", "", 0)
	}
	for i := 0; i < 20; i++ {
		outcome, err := fx.lc.ExecuteWrapperRestore(context.Background(), WrapperRestoreRequest{
			Decide: wDecideInput(t, true), Grant: wGrant(t, wFreshValidatedAt), HasGrant: true, Policy: wPolicy(), Restore: wRestoreRequest(t),
		})
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if outcome.Decision.Action != axpane.ActionParked {
			t.Fatalf("iteration %d: action = %s (%v), want parked over the late answer", i, outcome.Decision.Action, outcome.Decision.Cause)
		}
		if !outcome.RefreshAttempted || outcome.RefreshSucceeded {
			t.Fatalf("iteration %d: refresh audit = %+v, want attempted without success", i, outcome)
		}
		if outcome.Restore != nil {
			t.Fatalf("iteration %d: late answer executed: %+v", i, outcome.Restore)
		}
	}
	if fx.runner.callCount() != 0 {
		t.Fatalf("late answers executed %v", fx.runner.subcommands())
	}
}
