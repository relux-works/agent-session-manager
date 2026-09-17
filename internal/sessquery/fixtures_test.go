package sessquery

import (
	"encoding/json"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"testing"
)

const idA = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
const idB = "0198f4c8-3e70-7a11-8a2b-1234567890ac"
const hostA = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
const hostB = "0198f4c8-4a10-7b22-8b3c-1234567890ac"
const lease = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
const zeroDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// Verbatim Section 5.1 fixture, re-identified after test-specific changes.
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

func identity(t *testing.T, value map[string]any, field string) []byte {
	t.Helper()
	value[field] = zeroDigest
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
func record(t *testing.T, id, name string) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(specRecordExample), &value); err != nil {
		t.Fatal(err)
	}
	value["session_id"], value["subject_id"], value["name"] = id, id, name
	return identity(t, value, "record_id")
}
func repository(t *testing.T) (*sessrepo.Repository, string) {
	t.Helper()
	root := t.TempDir()
	repo, err := sessrepo.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return repo, root
}
func create(t *testing.T, repo *sessrepo.Repository, id, name string) sessrepo.SessionRef {
	t.Helper()
	ref, err := repo.CreateSession(record(t, id, name))
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
func appendEvent(t *testing.T, repo *sessrepo.Repository, id, predecessor, typ string, sequence int, payload map[string]any) string {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": zeroDigest,
		"subject_id": id, "session_id": id, "event_type": typ, "created_by_host_id": hostA,
		"lease_epoch": 1, "lease_id": lease, "lease_sequence": sequence, "predecessors": []string{predecessor},
		"created_at": "2026-08-19T04:00:00.000Z", "payload": payload, "extensions": map[string]any{},
	}
	ref, err := repo.AppendEvent(id, identity(t, value, "event_id"))
	if err != nil {
		t.Fatal(err)
	}
	return ref.EventID
}

// leaseRecordBytes builds one validated Lease Record for tests: an
// epoch-1 create lease uses a null predecessor and null checkpoint,
// while a successor names its predecessor and a checkpoint. The holder
// is the winning owner; issued_by and created_by equal the holder.
func leaseRecordBytes(t *testing.T, sessionID string, epoch uint64, leaseID, holder, predecessor string, hasPredecessor bool) []byte {
	t.Helper()
	var predecessorValue any
	if hasPredecessor {
		predecessorValue = predecessor
	}
	var checkpointValue any
	var reason string
	if epoch == 1 && !hasPredecessor {
		reason = "create"
	} else {
		reason = "graceful_takeover"
		checkpointValue = zeroDigest
	}
	value := map[string]any{
		"schema": "urn:ax:schema:lease", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "lease_id": leaseID, "epoch": json.Number(uint64String(epoch)),
		"holder_host_id": holder, "predecessor_lease_id": predecessorValue, "reason": reason,
		"checkpoint_id": checkpointValue, "issued_by_host_id": holder, "created_by_host_id": holder,
		"created_at": "2026-08-19T04:09:00.000Z", "extensions": map[string]any{},
	}
	return identity(t, value, "record_id")
}

func uint64String(value uint64) string {
	return json.Number(fmtSprintUint64(value)).String()
}

func fmtSprintUint64(value uint64) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

// leaseCreate returns an epoch-1 create Lease Record for the standard
// test lease and holder.
func leaseCreate(t *testing.T, sessionID, holder string) []byte {
	t.Helper()
	return leaseRecordBytes(t, sessionID, 1, lease, holder, "", false)
}

// checkpointRecordBytes builds one validated Checkpoint Record bound
// to (sessionID, epoch, leaseID): the checkpoint a successor lease
// references for its session and predecessor lease. The record is
// identified through the canonicaljson owner before return, so a
// fixture failure surfaces here, never as a production admission.
func checkpointRecordBytes(t *testing.T, sessionID string, epoch uint64, leaseID, creator string) []byte {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:checkpoint", "schema_version": "1.0.0", "checkpoint_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID,
		"lease_epoch": json.Number(uint64String(epoch)), "lease_id": leaseID,
		"safe_boundary": map[string]any{
			"provider_id": "codex", "provider_version": "0.147.0", "evidence": "accepted_test",
			"input_blocked": true, "foreground_idle": true, "background_idle": true,
			"open_processes": json.Number("0"), "open_database_handles": json.Number("0"),
		},
		"event_heads":           []string{zeroDigest},
		"workspace_manifest_id": zeroDigest,
		"provider_manifest_id":  zeroDigest,
		"task_board_bundle_id":  nil,
		"created_by_host_id":    creator,
		"created_at":            "2026-08-19T04:09:30.000Z",
		"status":                "validated",
		"extensions":            map[string]any{},
	}
	return identity(t, value, "checkpoint_id")
}

// checkpointDigestOf returns the canonical digest of one validated
// Checkpoint Record without attaching it: successor leases name this
// digest in checkpoint_id.
func checkpointDigestOf(t *testing.T, checkpoint []byte) string {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(checkpoint, &value); err != nil {
		t.Fatal(err)
	}
	digest, ok := value["checkpoint_id"].(string)
	if !ok || digest == "" || digest == zeroDigest {
		t.Fatalf("checkpoint fixture carries no identified digest: %v", value["checkpoint_id"])
	}
	return digest
}

// leaseSuccessorBytes builds one validated epoch>1 Lease Record
// naming its predecessor lease and the digest of the validated
// Checkpoint Record taken under that predecessor.
func leaseSuccessorBytes(t *testing.T, sessionID string, epoch uint64, leaseID, holder, predecessor, checkpoint string) []byte {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:lease", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": sessionID, "session_id": sessionID, "lease_id": leaseID, "epoch": json.Number(uint64String(epoch)),
		"holder_host_id": holder, "predecessor_lease_id": predecessor, "reason": "graceful_takeover",
		"checkpoint_id": checkpoint, "issued_by_host_id": holder, "created_by_host_id": holder,
		"created_at": "2026-08-19T04:09:00.000Z", "extensions": map[string]any{},
	}
	return identity(t, value, "record_id")
}

// withSuccessor attaches a complete validated succession for one
// session: the epoch-1 create record, the epoch-2 successor naming
// it, and the Checkpoint Record taken under the predecessor that the
// successor references. The checkpoint names the latest real epoch-1
// head from the reader chain so the head-authority gate admits it.
// It returns the successor Lease Record digest for plan assertions.
func withSuccessor(t *testing.T, reader *Reader, sessionID, holder, successorLeaseID string) string {
	t.Helper()
	checkpoint := checkpointWithHeads(t, checkpointRecordBytes(t, sessionID, 1, lease, holder), headAtOrBefore(t, reader, sessionID, 1))
	digest := checkpointDigestOf(t, checkpoint)
	withCheckpoints(reader, checkpoint)
	withLeases(reader,
		leaseCreate(t, sessionID, holder),
		leaseSuccessorBytes(t, sessionID, 2, successorLeaseID, holder, lease, digest),
	)
	return digest
}

// withCheckpoints attaches validated Checkpoint Records to a reader.
func withCheckpoints(reader *Reader, checkpoints ...[]byte) *Reader {
	reader.CheckpointRecords = append(append([][]byte(nil), reader.CheckpointRecords...), checkpoints...)
	return reader
}

// withLeases attaches validated Lease Records to a reader.
func withLeases(reader *Reader, leases ...[]byte) *Reader {
	reader.LeaseRecords = append(append([][]byte(nil), reader.LeaseRecords...), leases...)
	return reader
}

// checkpointWithHeads rebinds one validated Checkpoint Record to the
// given event heads and re-identifies it through the canonicaljson
// owner. Section 5.4 requires every required checkpoint to resolve
// each head to a chained event at or before its bound lease, so
// valid fixtures must name a real head from their session chain,
// never the zero placeholder.
func checkpointWithHeads(t *testing.T, base []byte, heads ...string) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(base, &value); err != nil {
		t.Fatal(err)
	}
	value["event_heads"] = heads
	return identity(t, value, "checkpoint_id")
}

// headAtOrBefore returns the latest chained event for one session at
// or before the given lease epoch: the valid historical head a
// checkpoint bound to that epoch may name. It prefers the newest
// such event, so an epoch-1 checkpoint names the latest epoch-1
// event (never a later-lease tail), while a successor-epoch
// checkpoint may name its own-lease transfer event.
func headAtOrBefore(t *testing.T, reader *Reader, sessionID string, epoch uint64) string {
	t.Helper()
	events, err := reader.Local.ListEvents(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	best := ""
	for _, summary := range events {
		if summary.LeaseEpoch <= epoch {
			best = summary.EventID
		}
	}
	if best == "" {
		t.Fatalf("no chained event at or before epoch %d for session %s", epoch, sessionID)
	}
	return best
}
