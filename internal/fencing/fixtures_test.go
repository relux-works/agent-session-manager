package fencing

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// Fixed fencing-test identities. Every UUIDv7 keeps the version nibble
// 7 and an RFC 4122 variant; every UUIDv4 keeps nibble 4. The B/C
// variants share the first segment with the A identity (the
// prefix-confusion vector the prefix mutants admit) or differ in it
// (the fully foreign vector that still refuses under those mutants).
const (
	fenceSessionA = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	fenceSessionB = "0198f4c8-3e70-7a11-8a2b-1234567890ac"
	fenceSessionC = "0199f4c8-3e70-7a11-8a2b-1234567890ab"
	fenceHostA    = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	fenceHostB    = "0198f4c8-4a10-7b22-8b3c-1234567890ac"
	fenceHostC    = "0199f4c8-4a10-7b22-8b3c-1234567890ab"
	fenceLeaseA   = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	fenceLeaseOld = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	fenceLeaseC   = "cccccccc-dddd-4eee-8fff-111111111111"
	fenceLeaseC2  = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeef"
	fenceCreated  = "2026-08-19T04:00:00.000Z"
	fenceLeaseAt  = "2026-08-19T04:09:00.000Z"
)

const fenceZeroDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

var (
	fenceValidatedAt = time.Date(2026, 8, 19, 4, 9, 0, 0, time.UTC)
	fencePolicy      = sessrepo.FencingPolicy{RefreshInterval: time.Minute}
)

// specRecordExample is the verbatim SPEC.md Section 5.1 normative
// example, the session-creation fixture.
const specRecordExample = `{
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
  "execution_profile": "yolo",
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

// fenceObservation returns the valid observation every positive test
// authorizes against: session A, winning lease A at epoch 2 held by
// the local host A, verified and unambiguous sync, a fresh grant, and
// the caller's clock reading.
func fenceObservation() Observation {
	return Observation{
		SessionID:     fenceSessionA,
		Winner:        sessrepo.LeaseSummary{SessionID: fenceSessionA, LeaseID: fenceLeaseA, Epoch: 2, HolderHostID: fenceHostA},
		HasWinner:     true,
		LocalHostID:   fenceHostA,
		Verified:      true,
		Ambiguous:     false,
		HandoffFailed: false,
		Grant:         sessrepo.FencingGrant{SessionID: fenceSessionA, ValidatedAt: fenceValidatedAt},
		HasGrant:      true,
		Policy:        fencePolicy,
		Now:           fenceValidatedAt,
	}
}

// fencePresented returns the exact winning token for fenceObservation.
func fencePresented() PresentedToken {
	return PresentedToken{SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseA}
}

// fenceEntries rosters every production gate entry with its launch
// class, so each refusal table drives all five through the same
// vectors.
func fenceEntries() map[string]struct {
	authorize func(PresentedToken, Observation) (LeaseToken, error)
	launch    bool
} {
	return map[string]struct {
		authorize func(PresentedToken, Observation) (LeaseToken, error)
		launch    bool
	}{
		"activation": {AuthorizeActivation, true},
		"input":      {AuthorizeInput, false},
		"mutation":   {AuthorizeMutation, false},
		"checkpoint": {AuthorizeCheckpoint, false},
		"restore":    {AuthorizeRestore, true},
	}
}

// mustMint fails unless the gate authorizes and the minted token binds
// to every provider operation with the exact winning triple.
func mustMint(t *testing.T, token LeaseToken, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s error = %v, want a minted token", what, err)
	}
	for _, operation := range []ProviderOperation{ProviderQuiesce, ProviderCapture, ProviderMaterialize} {
		bound, err := token.Bind(operation)
		if err != nil {
			t.Fatalf("%s Bind(%s) error = %v", what, operation, err)
		}
		if bound.SessionID != fenceSessionA || bound.Epoch != 2 || bound.LeaseID != fenceLeaseA || bound.Operation != operation {
			t.Fatalf("%s Bind(%s) = %+v, want the winning triple", what, operation, bound)
		}
	}
}

// mustRefuse fails unless err carries the sentinel and is not a park
// decision. A gate that parks where it must refuse, or mints, fails.
func mustRefuse(t *testing.T, err error, sentinel error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s succeeded, want refusal %v", what, sentinel)
	}
	if IsParked(err) {
		t.Fatalf("%s parked, want hard refusal %v: %v", what, sentinel, err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("%s error = %v, want errors.Is %v", what, err, sentinel)
	}
}

// mustPark fails unless err is a park decision with the exact reason,
// winning lease ID, and underlying Section 15 cause. A gate that
// refuses hard or mints where it must park fails here.
func mustPark(t *testing.T, err error, reason ParkReason, winningLeaseID string, cause error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s succeeded, want park %s", what, reason)
	}
	if !IsParked(err) {
		t.Fatalf("%s error = %v, want park %s", what, err, reason)
	}
	gotReason, gotLease, ok := ParkDetails(err)
	if !ok {
		t.Fatalf("%s parks without vocabulary: %v", what, err)
	}
	if gotReason != reason || gotLease != winningLeaseID {
		t.Fatalf("%s park = (%s, %s), want (%s, %s)", what, gotReason, gotLease, reason, winningLeaseID)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("%s park cause = %v, want errors.Is %v", what, err, cause)
	}
}

// openFenceRepository opens a repository under a test-temporary root
// and returns the data root alongside it, so tests can address the
// sessions layout without guessing.
func openFenceRepository(t *testing.T) (*sessrepo.Repository, string) {
	t.Helper()
	root := t.TempDir()
	repository, err := sessrepo.Open(root)
	if err != nil {
		t.Fatalf("Open(test root) error = %v", err)
	}
	return repository, root
}

// createFenceSession persists the SPEC record example.
func createFenceSession(t *testing.T, repository *sessrepo.Repository) sessrepo.SessionRef {
	t.Helper()
	reference, err := repository.CreateSession([]byte(specRecordExample))
	if err != nil {
		t.Fatalf("CreateSession(SPEC record) error = %v", err)
	}
	return reference
}

// identifyFenceObject recomputes the canonical digest of a built
// record or event through the production identity entry.
func identifyFenceObject(t *testing.T, object map[string]any, field string) []byte {
	t.Helper()
	object[field] = fenceZeroDigest
	staged, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal staged object: %v", err)
	}
	digest, _, err := canonicaljson.CalculateObjectIdentity(staged)
	if err != nil {
		t.Fatalf("CalculateObjectIdentity error = %v", err)
	}
	object[field] = digest.String()
	out, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("marshal object: %v", err)
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(out); err != nil {
		t.Fatalf("VerifyObjectIdentity error = %v", err)
	}
	return out
}

// buildFenceEvent derives a valid Session Event for the given chain
// position and authoring host.
func buildFenceEvent(t *testing.T, sessionID string, predecessors []string, epoch uint64, leaseID string, sequence uint64, hostID, eventType string, payload map[string]any) []byte {
	t.Helper()
	return identifyFenceObject(t, map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0",
		"event_id": fenceZeroDigest, "subject_id": sessionID, "session_id": sessionID,
		"event_type": eventType, "created_by_host_id": hostID,
		"lease_epoch": epoch, "lease_id": leaseID, "lease_sequence": sequence,
		"predecessors": predecessors, "created_at": fenceCreated,
		"payload": payload, "extensions": map[string]any{},
	}, "event_id")
}

// mustCreateFenceLease creates the epoch-1 lease or fails the test.
func mustCreateFenceLease(t *testing.T, repository *sessrepo.Repository, sessionID string, input sessrepo.CreateLeaseInput) sessrepo.LeaseRef {
	t.Helper()
	reference, err := repository.CreateLease(sessionID, input)
	if err != nil {
		t.Fatalf("CreateLease error = %v", err)
	}
	return reference
}

// mustSwapFenceLease advances the lease head or fails the test.
func mustSwapFenceLease(t *testing.T, repository *sessrepo.Repository, sessionID string, expected sessrepo.LeaseExpectation, input sessrepo.SuccessorLeaseInput) sessrepo.LeaseRef {
	t.Helper()
	reference, err := repository.CompareAndSwapLease(sessionID, expected, input)
	if err != nil {
		t.Fatalf("CompareAndSwapLease error = %v", err)
	}
	return reference
}
