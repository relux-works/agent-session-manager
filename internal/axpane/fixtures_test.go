package axpane

import (
	"bytes"
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
	"testing"
	"time"

	"github.com/gowebpki/jcs"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/fencing"
	"github.com/relux-works/agent-session-manager/internal/matjournal"
	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/sessckpt"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Fixture identities. UUIDv7 values keep the version and variant
// nibbles of the landed fixtures and vary only trailing hex.
const (
	fixtureSession    = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	fixtureForeign    = "0198f4c8-3e70-7a11-8a2b-1234567890ff"
	fixtureLocalHost  = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	fixtureRemoteHost = "0198f4c8-4a10-7b22-8b3c-1234567890ac"
	fixtureLeaseA     = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	fixtureLeaseB     = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	fixtureBootstrap  = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	fixtureOtherOp    = "0198f4c8-7d40-7e55-8e6f-1234567890ac"
	fixtureMat        = "0198f4c8-9a10-7b22-8b3c-1234567890ab"
	fixturePrepareOp  = "0198f4c8-9a10-7b22-8b3c-1234567890ac"
	fixtureInstance   = "0198f4c9-1111-7aaa-8aaa-1234567890ab"
	fixtureCreatedAt  = "2026-08-19T04:09:30.000Z"
)

func fixtureNow() time.Time {
	return time.Date(2026, time.August, 19, 4, 12, 0, 0, time.UTC)
}

func seedDigest(seed byte) string {
	return "sha256:" + fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
		seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed,
		seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed, seed)
}

// --- fencing fixtures ---

func fixtureGrant(now time.Time) (sessrepo.FencingGrant, sessrepo.FencingPolicy) {
	return sessrepo.FencingGrant{
			SessionID:   fixtureSession,
			RecordID:    seedDigest(0xA1),
			Token:       sessrepo.FencingToken{Epoch: 1, LeaseID: fixtureLeaseA, HolderHostID: fixtureLocalHost},
			ValidatedAt: now.Add(-time.Minute),
		}, sessrepo.FencingPolicy{
			RefreshInterval: time.Hour,
		}
}

func fixtureObservation(now time.Time) fencing.Observation {
	grant, policy := fixtureGrant(now)
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

// --- backend identity fixtures ---

// claimRows transcribes the §4.D registry rows this package needs,
// copied from the specification table rather than from production,
// so row drift fails.
func claimRows() map[string]map[string]any {
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
		"credential_capable_execution_realm": {
			"generation_variable":  true,
			"dependent_operations": []any{"create", "restore"},
			"evidence_requirements": []any{
				"conformance_fixture", "credential_sentinel", "provider_auth_smoke", "runtime_probe",
			},
		},
	}
}

func claimMap(capability, origin string, value bool) map[string]any {
	row := claimRows()[capability]
	return map[string]any{
		"capability":            capability,
		"origin":                origin,
		"value":                 value,
		"generation_variable":   row["generation_variable"],
		"dependent_operations":  row["dependent_operations"],
		"evidence_requirements": row["evidence_requirements"],
	}
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

// backendUniverse is one coherent admission world: a builtin ax.tmux
// manifest with static true claims, its probe, and the evidence set
// proving the true claims. attachAdmitted selects whether the probe
// carries local_attach true (remote attach offered) or false
// (takeover offered instead).
type backendUniverse struct {
	rawGeneration string
	manifest      map[string]any
	probe         map[string]any
	evidence      []map[string]any
	verify        terminalbackend.SignatureVerifier
	now           time.Time
	key           *rsa.PrivateKey
	fixtureID     string
	issuerID      string
}

func buildBackendUniverse(t *testing.T, attachAdmitted bool) *backendUniverse {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate attestation key: %v", err)
	}
	universe := &backendUniverse{
		rawGeneration: "generation-alpha",
		now:           time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC),
		verify: func(issuerID string, message, signature []byte) error {
			digest := sha256.Sum256(message)
			return rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature)
		},
		key:       key,
		fixtureID: seedDigest(0xF1),
		issuerID:  seedDigest(0x1D),
	}
	generationDigest, err := terminalbackend.GenerationDigest(universe.rawGeneration)
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	fixtureID := seedDigest(0xF1)
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
			claimMap("durable_disconnect", "static", true),
			claimMap("headless_creation", "static", true),
			claimMap("local_attach", "static", true),
			claimMap("reboot_restoration", "static", true),
		},
		"conformance_fixture_id": fixtureID,
		"extensions":             map[string]any{},
	}
	universe.manifest["manifest_id"] = omitSelfIdentity(t, universe.manifest, "manifest_id")

	probeClaims := []any{
		claimMap("durable_disconnect", "static", true),
		claimMap("headless_creation", "probed", true),
		claimMap("local_attach", "probed", attachAdmitted),
		claimMap("reboot_restoration", "probed", true),
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
	issuerID := seedDigest(0x1D)
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
		message := evidenceMessage(t, object)
		digest := sha256.Sum256(message)
		signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		if err != nil {
			t.Fatalf("sign evidence: %v", err)
		}
		object["attestation_signature"] = "rsa-sha256:" + base64.StdEncoding.EncodeToString(signature)
		object["evidence_id"] = omitSelfIdentity(t, object, "evidence_id")
		universe.evidence = append(universe.evidence, object)
		ids = append(ids, object["evidence_id"].(string))
	}
	sort.Strings(ids)
	asValues := make([]any, 0, len(ids))
	for _, id := range ids {
		asValues = append(asValues, id)
	}
	universe.probe["evidence_ids"] = asValues
	universe.probe["probe_id"] = omitSelfIdentity(t, universe.probe, "probe_id")
	return universe
}

// addRealmEvidence extends the universe with the admitted
// credential_capable_execution_realm row bound to the given host
// binding and provider build: the static and probed claims plus the
// signed evidence carrying the terminal binding, provider id/build,
// sentinel and smoke results, generation digest, OS version, and
// liveness. It re-computes the evidence set and both omit-self
// identities.
func addRealmEvidence(t *testing.T, universe *backendUniverse, bindingID, providerID, providerBuild string) {
	t.Helper()
	generationDigest, err := terminalbackend.GenerationDigest(universe.rawGeneration)
	if err != nil {
		t.Fatalf("GenerationDigest() error = %v", err)
	}
	manifestClaims := append(
		universe.manifest["static_capability_claims"].([]any),
		claimMap("credential_capable_execution_realm", "static", true),
	)
	sort.Slice(manifestClaims, func(i, j int) bool {
		return manifestClaims[i].(map[string]any)["capability"].(string) < manifestClaims[j].(map[string]any)["capability"].(string)
	})
	universe.manifest["static_capability_claims"] = manifestClaims
	probeClaims := append(
		universe.probe["capability_claims"].([]any),
		claimMap("credential_capable_execution_realm", "probed", true),
	)
	sort.Slice(probeClaims, func(i, j int) bool {
		return probeClaims[i].(map[string]any)["capability"].(string) < probeClaims[j].(map[string]any)["capability"].(string)
	})
	universe.probe["capability_claims"] = probeClaims
	object := map[string]any{
		"schema":                     terminalbackend.SchemaCapabilityEvidence,
		"schema_version":             "1.0.0",
		"evidence_id":                "",
		"terminal_backend_id":        terminalbackend.BuiltinTmux,
		"implementation_version":     "2.1.0",
		"protocol_version":           "1.1.0",
		"backend_generation_digest":  generationDigest,
		"capability":                 "credential_capable_execution_realm",
		"value":                      true,
		"platform":                   "linux",
		"os_version":                 "14.5",
		"conformance_fixture_id":     universe.fixtureID,
		"observed_at":                "2025-06-01T00:00:00.000Z",
		"expires_at":                 "2027-06-01T00:00:00.000Z",
		"issuer":                     terminalbackend.IssuerLocalProbe,
		"issuer_id":                  universe.issuerID,
		"attestation_signature":      "",
		"facts":                      []any{"fixture_passed", "provider_auth_passed", "runtime_probe_passed", "sentinel_passed"},
		"terminal_binding_id":        bindingID,
		"provider_id":                providerID,
		"provider_build":             providerBuild,
		"sentinel_result":            "passed",
		"provider_auth_smoke_result": "passed",
		"extensions":                 map[string]any{},
	}
	message := evidenceMessage(t, object)
	digest := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, universe.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign realm evidence: %v", err)
	}
	object["attestation_signature"] = "rsa-sha256:" + base64.StdEncoding.EncodeToString(signature)
	object["evidence_id"] = omitSelfIdentity(t, object, "evidence_id")
	universe.evidence = append(universe.evidence, object)
	var ids []any
	for _, existing := range universe.evidence {
		ids = append(ids, existing["evidence_id"])
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].(string) < ids[j].(string) })
	universe.probe["evidence_ids"] = ids
	universe.manifest["manifest_id"] = omitSelfIdentity(t, universe.manifest, "manifest_id")
	universe.probe["probe_id"] = omitSelfIdentity(t, universe.probe, "probe_id")
}

func mustJSON(t *testing.T, object map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

func (universe *backendUniverse) facts(t *testing.T) BackendFacts {
	t.Helper()
	evidence := make([][]byte, 0, len(universe.evidence))
	for _, object := range universe.evidence {
		evidence = append(evidence, mustJSON(t, object))
	}
	return BackendFacts{
		Manifest:      mustJSON(t, universe.manifest),
		Probe:         mustJSON(t, universe.probe),
		Evidence:      evidence,
		RawGeneration: universe.rawGeneration,
		Verify:        universe.verify,
		Now:           universe.now,
	}
}

func (universe *backendUniverse) hostBinding() terminalbackend.InstanceBinding {
	return terminalbackend.InstanceBinding{
		BackendID:             terminalbackend.BuiltinTmux,
		ImplementationVersion: "2.1.0",
		ProtocolVersion:       "1.1.0",
		Generation:            universe.rawGeneration,
		TerminalBindingID:     seedDigest(0xB1),
	}
}

// --- provider fixtures ---

func fixtureBuild() provhost.BuildTuple {
	return provhost.BuildTuple{
		ProviderID:      "codex",
		ProviderVersion: "0.147.0",
		Platform:        "linux",
		Architecture:    "amd64",
	}
}

func fixtureIdentity(t *testing.T) []byte {
	t.Helper()
	record, err := provhost.CreateIdentity(provhost.IdentityParams{
		SessionID:            fixtureSession,
		ProviderID:           "codex",
		ProviderVersion:      "0.147.0",
		ProviderVersionRange: "opaque-range",
		NativeSessionID:      "native-session-alpha",
		IdentityKind:         "session_uuid",
		LogicalWorkspaceID:   fixtureLocalHost,
		CreatedByHostID:      fixtureLocalHost,
		CreatedAt:            fixtureCreatedAt,
	})
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
	return record
}

// --- smoke fixtures ---

// smokeRecord builds one valid smoke record document through the
// omit-self recipe: the closed 16-member shape with a consistent
// verdict over its checks. It mirrors the resumesmoke encoding
// recipe (placeholder frame substitution) without importing its
// unexported encoder.
func smokeRecord(t *testing.T, verdict string, cell string, checks []any, tuple map[string]any) []byte {
	t.Helper()
	object := map[string]any{
		"schema":                 "urn:ax:schema:native-resume-smoke",
		"schema_version":         "1.0.0",
		"record_id":              "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"tuple":                  tuple,
		"resume_cell":            cell,
		"gate_ref":               "gate:8.4/codex",
		"verdict":                verdict,
		"probe_digest":           seedDigest(0xC1),
		"store_root":             "/home/ivan/.local/state/ax/provider-stores/codex",
		"discovery_proof_digest": seedDigest(0xC2),
		"identity_record_id":     seedDigest(0xC3),
		"spawn_plan_digest":      seedDigest(0xC4),
		"quiescence_digest":      seedDigest(0xC5),
		"checks":                 checks,
		"started_at":             "2026-08-19T04:09:30.000Z",
		"completed_at":           "2026-08-19T04:10:30.000Z",
	}
	staged, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal staged smoke record: %v", err)
	}
	sum := sha256.Sum256(staged)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	framed := []byte(`"record_id":"sha256:0000000000000000000000000000000000000000000000000000000000000000"`)
	claimed := []byte(`"record_id":"` + digest + `"`)
	if !bytes.Contains(staged, framed) {
		t.Fatalf("smoke record placeholder frame is absent")
	}
	return bytes.Replace(staged, framed, claimed, 1)
}

func smokeTuple() map[string]any {
	return map[string]any{
		"provider_id":      "codex",
		"provider_version": "0.147.0",
		"platform":         "linux",
		"architecture":     "amd64",
	}
}

func passingSmokeChecks() []any {
	return []any{
		map[string]any{"name": "resume-cell", "outcome": "pass", "detail": "cell A"},
		map[string]any{"name": "tuple-gate", "outcome": "pass", "detail": "gate 8.4"},
		map[string]any{"name": "probe", "outcome": "pass", "detail": "probe digest"},
	}
}

// --- session/profile fixtures ---

const sessionRecordFixture = `{
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

// chainFixture persists a session with its bootstrap event and
// epoch-1 lease, returning the repository, the session/record IDs,
// and the created event digest.
func chainFixture(t *testing.T) (*sessrepo.Repository, string) {
	t.Helper()
	repository, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal([]byte(sessionRecordFixture), &record); err != nil {
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
	appendChainEvent(t, repository, "session.created", 1, fixtureLeaseA, 1, reference.RecordID, map[string]any{
		"session_record_id": reference.RecordID, "bootstrap_operation_id": fixtureBootstrap, "first_checkpoint_operation_id": fixtureOtherOp,
	})
	return repository, reference.RecordID
}

func appendChainEvent(t *testing.T, repository *sessrepo.Repository, eventType string, epoch uint64, leaseID string, sequence int, predecessor string, payload map[string]any) string {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
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

// --- journal fixtures ---

// journalFixture creates a journal over one workspace authority and
// walks it to committed: the after-restore step-3 valid
// materialization. No managed replica participates, so no marker
// evidence is required; no provider or bridge participates, so only
// the authority leg must commit.
func journalFixture(t *testing.T, store *matjournal.Store) matjournal.Journal {
	t.Helper()
	request := []byte(`{"materialization_id":"` + fixtureMat + `","operation_id":"` + fixturePrepareOp + `","plan_id":"` + seedDigest(0xD1) + `"}`)
	inputs := matjournal.CreateInputs{
		MaterializationID:  fixtureMat,
		PrepareOperationID: fixturePrepareOp,
		RequestBody:        request,
		PlanID:             seedDigest(0xD1),
		SourceCheckpointID: seedDigest(0xD2),
		Plan: []matjournal.PlanAuthority{
			{ID: "workspace_relux", Kind: matjournal.PlanKindWorkspace, Platform: "linux", RootPath: "/home/ivan/work/relux"},
		},
		HostPlatform: "linux",
		Extensions:   map[string]string{},
	}
	if _, _, err := store.Create(inputs); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	for _, phase := range []string{matjournal.PhaseValidating, matjournal.PhasePrepared, matjournal.PhaseCommitting} {
		if _, err := store.Transition(fixtureMat, phase, matjournal.TransitionOpts{}); err != nil {
			t.Fatalf("Transition(%s) error = %v", phase, err)
		}
	}
	rollback := "/home/ivan/work/relux.prior"
	if _, err := store.UpdateAuthority(fixtureMat, "workspace_relux", matjournal.AuthorityState{
		RootPath:           "/home/ivan/work/relux",
		CompletedSequences: []uint64{1},
		RollbackRoot:       &rollback,
		State:              matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatalf("UpdateAuthority(prepared) error = %v", err)
	}
	if _, err := store.UpdateAuthority(fixtureMat, "workspace_relux", matjournal.AuthorityState{
		RootPath:           "/home/ivan/work/relux",
		CompletedSequences: []uint64{1},
		State:              matjournal.AuthorityCommitted,
	}); err != nil {
		t.Fatalf("UpdateAuthority(committed) error = %v", err)
	}
	journal, err := store.Transition(fixtureMat, matjournal.PhaseCommitted, matjournal.TransitionOpts{})
	if err != nil {
		t.Fatalf("Transition(committed) error = %v", err)
	}
	return journal
}

// --- checkpoint fixtures ---

// checkpointFixture captures a direct checkpoint over the chain
// fixture heads and returns its digest and bytes.
func checkpointFixture(t *testing.T, repository *sessrepo.Repository, store *sessckpt.Store, heads []string) (string, []byte) {
	t.Helper()
	reference, raw, err := store.Capture(repository, sessckpt.Inputs{
		OperationID:         fixtureBootstrap,
		SessionID:           fixtureSession,
		SessionKind:         sessckpt.SessionKindDirect,
		LeaseEpoch:          1,
		LeaseID:             fixtureLeaseA,
		CreatorHostID:       fixtureLocalHost,
		WorkspaceManifestID: seedDigest(0xE1),
		ProviderManifestID:  seedDigest(0xE2),
		Boundary: sessckpt.SafeBoundary{
			ProviderID:          "codex",
			ProviderVersion:     "0.147.0",
			Evidence:            sessckpt.EvidenceAcceptedTest,
			InputBlocked:        true,
			ForegroundIdle:      true,
			BackgroundIdle:      true,
			OpenProcesses:       0,
			OpenDatabaseHandles: 0,
		},
		EventHeads: heads,
		CreatedAt:  fixtureCreatedAt,
		Extensions: map[string]string{},
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	return reference.CheckpointID, raw
}

var errFixtureConfig = errors.New("fixture configuration is invalid")

// --- fold-newest fixtures ---

// publishCheckpoint appends the ordinary same-host closure to the
// chain — session.idle then a checkpointed session.stopped naming
// the checkpoint — so the landed fold derives state=stopped with
// the checkpoint as newest. nextSeq is the idle event's lease
// sequence; it returns the stopped event's digest.
func publishCheckpoint(t *testing.T, repository *sessrepo.Repository, headID string, nextSeq int, ckptID string) string {
	t.Helper()
	idleID := appendChainEvent(t, repository, "session.idle", 1, fixtureLeaseA, nextSeq, headID, map[string]any{
		"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true,
	})
	return appendChainEvent(t, repository, "session.stopped", 1, fixtureLeaseA, nextSeq+1, idleID, map[string]any{
		"graceful": true, "checkpoint_id": ckptID, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true,
	})
}

// journalSourced creates a committed journal sourced from the given
// checkpoint digest, in its own store: the after-restore step-3
// valid materialization for exactly that checkpoint.
func journalSourced(t *testing.T, source string) *matjournal.Store {
	t.Helper()
	store, err := matjournal.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	request := []byte(`{"materialization_id":"` + fixtureMat + `","operation_id":"` + fixturePrepareOp + `","plan_id":"` + seedDigest(0xD1) + `"}`)
	inputs := matjournal.CreateInputs{
		MaterializationID:  fixtureMat,
		PrepareOperationID: fixturePrepareOp,
		RequestBody:        request,
		PlanID:             seedDigest(0xD1),
		SourceCheckpointID: source,
		Plan: []matjournal.PlanAuthority{
			{ID: "workspace_relux", Kind: matjournal.PlanKindWorkspace, Platform: "linux", RootPath: "/home/ivan/work/relux"},
		},
		HostPlatform: "linux",
		Extensions:   map[string]string{},
	}
	if _, _, err := store.Create(inputs); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	for _, phase := range []string{matjournal.PhaseValidating, matjournal.PhasePrepared, matjournal.PhaseCommitting} {
		if _, err := store.Transition(fixtureMat, phase, matjournal.TransitionOpts{}); err != nil {
			t.Fatalf("Transition(%s) error = %v", phase, err)
		}
	}
	rollback := "/home/ivan/work/relux.prior"
	if _, err := store.UpdateAuthority(fixtureMat, "workspace_relux", matjournal.AuthorityState{
		RootPath: "/home/ivan/work/relux", CompletedSequences: []uint64{1}, RollbackRoot: &rollback, State: matjournal.AuthorityPrepared,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateAuthority(fixtureMat, "workspace_relux", matjournal.AuthorityState{
		RootPath: "/home/ivan/work/relux", CompletedSequences: []uint64{1}, State: matjournal.AuthorityCommitted,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(fixtureMat, matjournal.PhaseCommitted, matjournal.TransitionOpts{}); err != nil {
		t.Fatal(err)
	}
	return store
}

// captureUnpublished captures a second self-consistent checkpoint
// of the session that the chain never publishes: newest stays
// whatever the chain names, so admission must refuse this digest.
func captureUnpublished(t *testing.T, repository *sessrepo.Repository, store *sessckpt.Store, operation string, heads []string) (string, []byte) {
	t.Helper()
	reference, raw, err := store.Capture(repository, sessckpt.Inputs{
		OperationID:         operation,
		SessionID:           fixtureSession,
		SessionKind:         sessckpt.SessionKindDirect,
		LeaseEpoch:          1,
		LeaseID:             fixtureLeaseA,
		CreatorHostID:       fixtureLocalHost,
		WorkspaceManifestID: seedDigest(0xE1),
		ProviderManifestID:  seedDigest(0xE2),
		Boundary: sessckpt.SafeBoundary{
			ProviderID: "codex", ProviderVersion: "0.147.0", Evidence: sessckpt.EvidenceAcceptedTest,
			InputBlocked: true, ForegroundIdle: true, BackgroundIdle: true, OpenProcesses: 0, OpenDatabaseHandles: 0,
		},
		EventHeads: heads,
		CreatedAt:  fixtureCreatedAt,
		Extensions: map[string]string{},
	})
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	return reference.CheckpointID, raw
}

// successorLease installs a successor lease carrying the given
// checkpoint through CompareAndSwapLease: the handoff base for
// the agreement arm.
func successorLease(t *testing.T, repository *sessrepo.Repository, leaseID, holderHostID, checkpointID string) {
	t.Helper()
	leases, err := repository.ListLeases(fixtureSession)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CompareAndSwapLease(fixtureSession, sessrepo.LeaseExpectation{RecordID: leases[len(leases)-1].RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{
			LeaseID:        leaseID,
			HolderHostID:   holderHostID,
			IssuedByHostID: holderHostID,
			CreatedAt:      fixtureCreatedAt,
		},
		Reason:       "graceful_takeover",
		CheckpointID: checkpointID,
	}); err != nil {
		t.Fatalf("CompareAndSwapLease() error = %v", err)
	}
}

// addSecondRealmEvidence clones the universe's latest realm
// evidence object with a new terminal binding and provider build,
// re-signs and re-identifies it: two admitted realm objects whose
// order the cross-bind must not depend on.
func addSecondRealmEvidence(t *testing.T, universe *backendUniverse, bindingID, providerID, providerBuild string) {
	t.Helper()
	if len(universe.evidence) == 0 {
		t.Fatal("universe carries no evidence to clone")
	}
	first := universe.evidence[len(universe.evidence)-1]
	object := map[string]any{}
	for key, value := range first {
		object[key] = value
	}
	object["terminal_binding_id"] = bindingID
	object["provider_id"] = providerID
	object["provider_build"] = providerBuild
	object["attestation_signature"] = ""
	object["evidence_id"] = ""
	message := evidenceMessage(t, object)
	digest := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, universe.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	object["attestation_signature"] = "rsa-sha256:" + base64.StdEncoding.EncodeToString(signature)
	object["evidence_id"] = omitSelfIdentity(t, object, "evidence_id")
	universe.evidence = append(universe.evidence, object)
	var ids []any
	for _, existing := range universe.evidence {
		ids = append(ids, existing["evidence_id"])
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].(string) < ids[j].(string) })
	universe.probe["evidence_ids"] = ids
	universe.probe["probe_id"] = omitSelfIdentity(t, universe.probe, "probe_id")
}
