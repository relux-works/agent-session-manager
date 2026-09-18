package termbind

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gowebpki/jcs"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
	"github.com/relux-works/agent-session-manager/internal/terminstance"
)

// Fixture identities. UUIDv7 values keep the version and variant nibbles
// of the landed fixtures and vary only trailing hex.
const (
	fixtureSession    = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	fixtureLocalHost  = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	fixtureRemoteHost = "0198f4c8-4a10-7b22-8b3c-1234567890ac"
	fixtureLeaseA     = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	fixtureLeaseB     = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	fixtureBootstrap  = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	fixtureOtherOp    = "0198f4c8-7d40-7e55-8e6f-1234567890ac"
	fixtureInstance   = "0198f4c9-1111-7aaa-8aaa-1234567890ab"
	fixtureClient     = "0198f4c9-2222-7bbb-8bbb-1234567890ab"
	fixtureClientB    = "0198f4c9-3333-7ccc-8ccc-1234567890ab"
	fixtureCreatedAt  = "2026-08-19T04:09:30.000Z"
	fixtureDeadline   = "2026-08-19T05:00:00.000Z"
	fixtureImpl       = "2.1.0"
	fixtureProto      = "1.1.0"
	fixtureRawGen     = "generation-alpha"
)

func fixtureNow() time.Time {
	return time.Date(2026, time.August, 19, 4, 12, 0, 0, time.UTC)
}

func seedDigest(seed byte) string {
	return "sha256:" + fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
		seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed,
		seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed)
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("operation succeeded, want %s", code)
	}
	refusal, ok := err.(*terminalbackend.Error)
	if !ok {
		t.Fatalf("error type = %T (%v), want *terminalbackend.Error with %s", err, err, code)
	}
	if refusal.Code != code {
		t.Fatalf("error code = %q, want %q", refusal.Code, code)
	}
}

// requireDetail pins which gate refused when two gates share one code: a
// mutant that reroutes past this package's gate to a same-code landed gate
// must still fail the killer.
func requireDetail(t *testing.T, err error, detail string) {
	t.Helper()
	if err == nil {
		t.Fatalf("operation succeeded, want detail %q", detail)
	}
	if !strings.Contains(err.Error(), detail) {
		t.Fatalf("error = %v, want detail %q", err, detail)
	}
}

// forbiddenIdentities is the eight-form corpus from Sections 4.B and 7.A:
// PID, handle, socket, path, named pipe, URL, token, and mutable endpoint.
// Each entry names the form and one concrete value of that form.
var forbiddenIdentities = []struct {
	form  string
	value string
}{
	{"pid", "12345"},
	{"handle", "0x00000000000004d2"},
	{"socket", "/tmp/tmux-1000/default"},
	{"path", "/var/run/ax/pane-1"},
	{"pipe", `\\.\pipe\ax-pane-1`},
	{"url", "https://mesh.local:8443/session/0198f4c8"},
	{"token", "ax-token-7f3a9c2e4b1d8f6a"},
	{"endpoint", "10.0.0.9:2222"},
}

// --- session chain fixtures ---

// sessionRecordJSON is the direct-session record the chain fixtures
// persist, transcribed from the axpane story fixture.
const sessionRecordJSON = `{
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
  "execution_profile": "standard",
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

func identifyObject(t *testing.T, value map[string]any, field string) []byte {
	t.Helper()
	value[field] = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	digest, _, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		t.Fatal(err)
	}
	value[field] = digest.String()
	raw, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func chainFixture(t *testing.T) (*sessrepo.Repository, string) {
	t.Helper()
	repository, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal([]byte(sessionRecordJSON), &record); err != nil {
		t.Fatal(err)
	}
	reference, err := repository.CreateSession(identifyObject(t, record, "record_id"))
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, err := repository.CreateLease(fixtureSession, sessrepo.CreateLeaseInput{
		LeaseID:        fixtureLeaseA,
		HolderHostID:   fixtureLocalHost,
		IssuedByHostID: fixtureLocalHost,
		CreatedAt:      fixtureCreatedAt,
	}); err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	appendEvent(t, repository, "1.0.0", "session.created", 1, fixtureLeaseA, 1, reference.RecordID, map[string]any{
		"session_record_id": reference.RecordID, "bootstrap_operation_id": fixtureBootstrap, "first_checkpoint_operation_id": fixtureOtherOp,
	})
	return repository, reference.RecordID
}

func appendEvent(t *testing.T, repository *sessrepo.Repository, schemaVersion, eventType string, epoch uint64, leaseID string, sequence int, predecessor string, payload map[string]any) string {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": schemaVersion, "event_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"subject_id": fixtureSession, "session_id": fixtureSession, "event_type": eventType, "created_by_host_id": fixtureLocalHost,
		"lease_epoch": epoch, "lease_id": leaseID, "lease_sequence": sequence, "predecessors": []string{predecessor},
		"created_at": fixtureCreatedAt, "payload": payload, "extensions": map[string]any{},
	}
	raw := identifyObject(t, value, "event_id")
	reference, err := repository.AppendEvent(fixtureSession, raw)
	if err != nil {
		t.Fatalf("AppendEvent(%s) error = %v", eventType, err)
	}
	return reference.EventID
}

func chainTail(t *testing.T, repository *sessrepo.Repository) sessrepo.EventSummary {
	t.Helper()
	events, err := repository.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("chain holds no events")
	}
	return events[len(events)-1]
}

func fixtureObservation(now time.Time) fencing.Observation {
	grant := sessrepo.FencingGrant{
		SessionID:   fixtureSession,
		RecordID:    seedDigest(0xA1),
		Token:       sessrepo.FencingToken{Epoch: 1, LeaseID: fixtureLeaseA, HolderHostID: fixtureLocalHost},
		ValidatedAt: now.Add(-time.Minute),
	}
	policy := sessrepo.FencingPolicy{RefreshInterval: time.Hour}
	return fencing.Observation{
		SessionID:   fixtureSession,
		Winner:      sessrepo.LeaseSummary{SessionID: fixtureSession, LeaseID: fixtureLeaseA, Epoch: 1, HolderHostID: fixtureLocalHost, Reason: "create"},
		HasWinner:   true,
		LocalHostID: fixtureLocalHost,
		Verified:    true,
		Grant:       grant,
		HasGrant:    true,
		Policy:      policy,
		Now:         now,
	}
}

func fixturePresented() fencing.PresentedToken {
	return fencing.PresentedToken{SessionID: fixtureSession, Epoch: 1, LeaseID: fixtureLeaseA}
}

// foldNewest derives the newest-checkpoint window bit from a real chain
// fold through the landed sessstate reducer: no newest means the bootstrap
// window is open.
func foldNewest(t *testing.T, repository *sessrepo.Repository) (bool, string) {
	t.Helper()
	recordJSON, err := repository.GetRecord(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	record, err := sessstate.DecodeRecord(recordJSON)
	if err != nil {
		t.Fatal(err)
	}
	summaries, err := repository.ListEvents(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	events := make([]sessstate.Event, 0, len(summaries))
	for _, summary := range summaries {
		raw, err := repository.GetEvent(fixtureSession, summary.EventID)
		if err != nil {
			t.Fatal(err)
		}
		event, err := sessstate.DecodeEvent(raw)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	projection, err := sessstate.Reduce(sessstate.Input{Record: record, Events: events})
	if err != nil {
		t.Fatal(err)
	}
	if !projection.HasCheckpoint {
		return false, ""
	}
	return true, projection.Newest.ID
}

// --- backend universe fixtures ---

// universe is one coherent admission world for one backend: a manifest
// with one static true claim, its probe echoing it, and the one signed
// evidence object proving the true claim.
type universe struct {
	backendID  string
	manifest   map[string]any
	manifestID string
	probe      map[string]any
	probeID    string
	evidence   map[string]any
	evidenceID string
	// ids is the sorted unique event evidence set: manifest, probe, and
	// capability evidence IDs.
	ids           []string
	rawGeneration string
	registry      *terminalbackend.Registry
	verify        terminalbackend.SignatureVerifier
	now           time.Time
}

func testRegistry(t *testing.T) *terminalbackend.Registry {
	t.Helper()
	registry, err := terminalbackend.New(fixtureImpl, []string{"1.0.0", "1.1.0"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return registry
}

func buildUniverse(t *testing.T, backendID, platform string, platforms []any, rawGeneration string) *universe {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate attestation key: %v", err)
	}
	now := time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC)
	world := &universe{
		backendID:     backendID,
		rawGeneration: rawGeneration,
		registry:      testRegistry(t),
		verify: func(issuerID string, message, signature []byte) error {
			digest := sha256.Sum256(message)
			return rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature)
		},
		now: now,
	}
	generationDigest, err := terminalbackend.GenerationDigest(rawGeneration)
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	fixtureID := seedDigest(0xF1)
	issuerID := seedDigest(0x1D)
	world.manifest = map[string]any{
		"schema":                 terminalbackend.SchemaManifest,
		"schema_version":         "1.0.0",
		"manifest_id":            "",
		"terminal_backend_id":    backendID,
		"implementation_version": fixtureImpl,
		"protocol_versions":      []any{"1.0.0", "1.1.0"},
		"platforms":              platforms,
		"implementation_kind":    "builtin_go",
		"executable_digest":      nil,
		"static_capability_claims": []any{
			map[string]any{
				"capability":            "durable_disconnect",
				"origin":                "static",
				"value":                 true,
				"generation_variable":   false,
				"dependent_operations":  []any{"create", "status"},
				"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
			},
		},
		"conformance_fixture_id": fixtureID,
		"extensions":             map[string]any{},
	}
	world.manifest["manifest_id"] = omitSelfIdentity(t, world.manifest, "manifest_id")
	world.manifestID = world.manifest["manifest_id"].(string)
	world.evidence = map[string]any{
		"schema":                     terminalbackend.SchemaCapabilityEvidence,
		"schema_version":             "1.0.0",
		"evidence_id":                "",
		"terminal_backend_id":        backendID,
		"implementation_version":     fixtureImpl,
		"protocol_version":           fixtureProto,
		"backend_generation_digest":  generationDigest,
		"capability":                 "durable_disconnect",
		"value":                      true,
		"platform":                   platform,
		"os_version":                 "14.5",
		"conformance_fixture_id":     fixtureID,
		"observed_at":                "2025-06-01T00:00:00.000Z",
		"expires_at":                 "2027-06-01T00:00:00.000Z",
		"issuer":                     terminalbackend.IssuerLocalProbe,
		"issuer_id":                  issuerID,
		"attestation_signature":      "",
		"facts":                      []any{"fixture_passed", "runtime_probe_passed"},
		"terminal_binding_id":        nil,
		"provider_id":                nil,
		"provider_build":             nil,
		"sentinel_result":            nil,
		"provider_auth_smoke_result": nil,
		"extensions":                 map[string]any{},
	}
	message := evidenceMessage(t, world.evidence)
	digest := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign evidence: %v", err)
	}
	world.evidence["attestation_signature"] = "rsa-sha256:" + base64.StdEncoding.EncodeToString(signature)
	world.evidence["evidence_id"] = omitSelfIdentity(t, world.evidence, "evidence_id")
	world.evidenceID = world.evidence["evidence_id"].(string)
	world.probe = map[string]any{
		"schema":                    terminalbackend.SchemaProbe,
		"schema_version":            "1.0.0",
		"probe_id":                  "",
		"terminal_backend_id":       backendID,
		"implementation_version":    fixtureImpl,
		"protocol_version":          fixtureProto,
		"implementation_kind":       "builtin_go",
		"executable_digest":         nil,
		"platform":                  platform,
		"os_version":                "14.5",
		"availability":              "available",
		"backend_generation_digest": generationDigest,
		"capability_claims": []any{
			map[string]any{
				"capability":            "durable_disconnect",
				"origin":                "static",
				"value":                 true,
				"generation_variable":   false,
				"dependent_operations":  []any{"create", "status"},
				"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
			},
		},
		"evidence_ids": []any{world.evidenceID},
		"probed_at":    "2026-01-15T12:00:00.000Z",
		"extensions":   map[string]any{},
	}
	world.probe["probe_id"] = omitSelfIdentity(t, world.probe, "probe_id")
	world.probeID = world.probe["probe_id"].(string)
	world.ids = []string{world.manifestID, world.probeID, world.evidenceID}
	sort.Strings(world.ids)
	// The universe self-checks through the landed admission before any
	// test uses it: a fixture that cannot admit is a broken test, not a
	// passing one.
	if _, err := world.registry.AdmitProbe(mustJSON(t, world.manifest), mustJSON(t, world.probe), [][]byte{mustJSON(t, world.evidence)}, rawGeneration, now, world.verify); err != nil {
		t.Fatalf("universe self-admission error = %v", err)
	}
	return world
}

func buildTmuxUniverse(t *testing.T) *universe {
	t.Helper()
	return buildUniverse(t, terminalbackend.BuiltinTmux, "linux", []any{"linux", "macos", "wsl2"}, fixtureRawGen)
}

func buildConptyUniverse(t *testing.T) *universe {
	t.Helper()
	return buildUniverse(t, terminalbackend.BuiltinConpty, "windows", []any{"windows"}, "generation-beta")
}

func omitSelfIdentity(t *testing.T, object map[string]any, selfField string) string {
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

func evidenceMessage(t *testing.T, object map[string]any) []byte {
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

func mustJSON(t *testing.T, object map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// mapUniverse is an EvidenceUniverse backed by an ID-keyed map.
type mapUniverse map[string][]byte

func (universe mapUniverse) Lookup(id string) ([]byte, bool) {
	raw, found := universe[id]
	return raw, found
}

func universeMap(t *testing.T, world *universe) mapUniverse {
	t.Helper()
	return mapUniverse{
		world.manifestID: mustJSON(t, world.manifest),
		world.probeID:    mustJSON(t, world.probe),
		world.evidenceID: mustJSON(t, world.evidence),
	}
}

// --- status engine fixtures ---

type mockBackend struct {
	observe func(body terminstance.StatusBody) (terminstance.StatusObservation, error)
}

func (backend *mockBackend) PerformEffect(_ context.Context, _ terminalbackend.SideEffect) (string, error) {
	return "", errors.New("mock backend performs no effects")
}

func (backend *mockBackend) ObserveStatus(_ context.Context, body terminstance.StatusBody) (terminstance.StatusObservation, error) {
	return backend.observe(body)
}

func testEngine(backend terminstance.Backend) *terminstance.Engine {
	return &terminstance.Engine{Backend: backend, Now: fixtureNow}
}

// --- attach fixtures ---

func attachAuth(t *testing.T, transport string, inputAuthorized bool, issuedAt, expiresAt string) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"policy_evidence_id":  seedDigest(0xA0),
		"authorizing_host_id": fixtureLocalHost,
		"transport":           transport,
		"input_authorized":    inputAuthorized,
		"issued_at":           issuedAt,
		"expires_at":          expiresAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func validAttachAuth(t *testing.T, transport string, inputAuthorized bool) []byte {
	t.Helper()
	return attachAuth(t, transport, inputAuthorized, "2026-08-19T04:00:00.000Z", "2026-08-19T06:00:00.000Z")
}

// --- binding fixtures ---

func bindingDoc(t *testing.T, mutate func(object map[string]any)) []byte {
	t.Helper()
	object := map[string]any{
		"schema":                 BindingSchema,
		"schema_version":         BindingSchemaVersion,
		"binding_id":             "",
		"session_id":             fixtureSession,
		"host_id":                fixtureLocalHost,
		"host_incarnation_id":    "0198f4c8-4a10-7b22-8b3c-1234567890ad",
		"terminal_instance_id":   fixtureInstance,
		"terminal_backend_id":    terminalbackend.BuiltinTmux,
		"implementation_version": fixtureImpl,
		"protocol_version":       fixtureProto,
		"backend_generation":     fixtureRawGen,
		"native_reference":       "tmux-session-payments-0",
		"created_at":             fixtureCreatedAt,
		"supersedes_binding_id":  nil,
		"extensions":             map[string]any{},
	}
	if mutate != nil {
		mutate(object)
	}
	if _, present := object["binding_id"]; present {
		if id, ok := object["binding_id"].(string); ok && id == "" {
			object["binding_id"] = omitSelfIdentity(t, object, "binding_id")
		}
	}
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func digestHex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
