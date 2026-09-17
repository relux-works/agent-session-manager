package sessquery

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

func sortStringsForTest(values []string) {
	sort.Strings(values)
}

// appendEventAs chains one event under an explicit envelope lease,
// unlike the epoch-1-only fixtures helper.
func appendEventAs(t *testing.T, repo *sessrepo.Repository, id, predecessor, typ string, sequence int, epoch uint64, leaseID string, payload map[string]any) string {
	t.Helper()
	value := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": zeroDigest,
		"subject_id": id, "session_id": id, "event_type": typ, "created_by_host_id": hostA,
		"lease_epoch": epoch, "lease_id": leaseID, "lease_sequence": sequence, "predecessors": []string{predecessor},
		"created_at": "2026-08-19T04:00:00.000Z", "payload": payload, "extensions": map[string]any{},
	}
	raw := identity(t, value, "event_id")
	ref, err := repo.AppendEvent(id, raw)
	if err != nil {
		t.Fatal(err)
	}
	return ref.EventID
}

// copySessionDir replicates one persisted session directory byte for
// byte, so two sources hold the same identity with agreeing digests.
func copySessionDir(t *testing.T, srcRoot, dstRoot, sessionID string) {
	t.Helper()
	src := filepath.Join(srcRoot, "sessions", sessionID)
	dst := filepath.Join(dstRoot, "sessions", sessionID)
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dst, "events"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			blobs, err := os.ReadDir(filepath.Join(src, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			for _, blob := range blobs {
				bytes, err := os.ReadFile(filepath.Join(src, entry.Name(), blob.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dst, entry.Name(), blob.Name()), bytes, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			continue
		}
		bytes, err := os.ReadFile(filepath.Join(src, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, entry.Name()), bytes, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// plannedSession builds a local session with its bootstrap event and
// validated initial Lease Record, returning the reader, roots, record
// digest, and created event ID. The reader carries the known local
// host identity every local plan requires for its source binding, plus
// the winning lease authority every plan requires.
func plannedSession(t *testing.T) (*Reader, string, sessrepo.SessionRef, string) {
	t.Helper()
	local, root := repository(t)
	ref := create(t, local, idA, "alpha")
	created := appendEvent(t, local, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	reader := &Reader{Local: local, LocalHostID: hostA}
	withLeases(reader, leaseCreate(t, idA, hostA))
	return reader, root, ref, created
}

func mustBuild(t *testing.T, reader *Reader, args PlanArgs) SelectionPlan {
	t.Helper()
	plan, err := reader.BuildPlan(args)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func mustStale(t *testing.T, err error, reason string) {
	t.Helper()
	if !errors.Is(err, ErrPlanStale) {
		t.Fatalf("error = %v, want selector_plan_stale", err)
	}
	if !strings.Contains(err.Error(), reason) {
		t.Fatalf("error = %v, want reason %q", err, reason)
	}
}

func TestBuildPlanBindsCurrentFacts(t *testing.T) {
	reader, _, ref, created := plannedSession(t)
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	if plan.SelectorVersion != SelectorVersion || plan.Selector != "alpha" || plan.SessionID != idA {
		t.Fatalf("identity: %+v", plan)
	}
	if plan.SessionRecordID != ref.RecordID {
		t.Fatalf("record digest = %q, want %q", plan.SessionRecordID, ref.RecordID)
	}
	if plan.SourceHostID != hostA || plan.SourceAlias != nil {
		t.Fatalf("local source: host %q alias %+v", plan.SourceHostID, plan.SourceAlias)
	}
	for _, digest := range []string{plan.SourceIndexDigest, plan.ConfigurationDigest, plan.ExpectationDigest} {
		if _, err := scalar.ParseDigest(digest); err != nil {
			t.Fatalf("digest %q: %v", digest, err)
		}
	}
	if plan.SourceIndexDigest == plan.ConfigurationDigest {
		t.Fatalf("index and configuration digests collide")
	}
	if plan.LeaseEpoch != 1 || plan.LeaseID != lease || plan.OwnerHostID != hostA {
		t.Fatalf("lease: %+v", plan)
	}
	// The plan binds the canonical digest of the admitted winning
	// Lease Record, never an envelope fingerprint.
	wantLease, err := reader.winningLeaseFor(idA, "direct", reader.Local)
	if err != nil {
		t.Fatal(err)
	}
	if plan.LeaseRecordID != wantLease.Digest {
		t.Fatalf("lease_record_id = %q, want %q", plan.LeaseRecordID, wantLease.Digest)
	}
	if _, err := scalar.ParseDigest(plan.LeaseRecordID); err != nil {
		t.Fatalf("lease_record_id %q: %v", plan.LeaseRecordID, err)
	}
	wantHeads := append([]string{wantLease.Digest, created}, []string{}...)
	sortStringsForTest(wantHeads)
	if !reflect.DeepEqual(plan.AuthorityHeads, wantHeads) {
		t.Fatalf("heads: %+v, want %+v", plan.AuthorityHeads, wantHeads)
	}
	if plan.Action != ActionStatus || plan.DestinationHostID != nil {
		t.Fatalf("action: %+v", plan)
	}
	if plan.ExpectationDigest != "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a" {
		t.Fatalf("empty expectations digest = %q", plan.ExpectationDigest)
	}
	again := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	if !reflect.DeepEqual(plan, again) {
		t.Fatalf("rebuild diverged:\n%+v\n%+v", plan, again)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("fresh plan is not current: %v", err)
	}
}

func TestBuildPlanPeerSourceFacts(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	create(t, local, idA, "alpha")
	ref := create(t, peer, idB, "beta")
	created := appendEvent(t, peer, idB, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	_ = created
	reader := &Reader{
		Local:              local,
		LocalHostID:        hostB,
		AllowlistedPeerIDs: []string{hostA},
		Peers:              []PeerIndex{{hostA, peer}},
		Aliases:            map[string]string{hostA: "workstation"},
	}
	withLeases(reader, leaseCreate(t, idB, hostA))
	plan := mustBuild(t, reader, PlanArgs{
		Selector:     "beta@peer:workstation",
		Action:       ActionAttach,
		Destination:  hostB,
		Expectations: []byte(`{"checkpoint_id":null}`),
	})
	if plan.SourceHostID != hostA || plan.SourceAlias == nil || *plan.SourceAlias != "workstation" {
		t.Fatalf("source: %+v", plan)
	}
	if plan.SessionID != idB || plan.SessionRecordID != ref.RecordID {
		t.Fatalf("identity: %+v", plan)
	}
	if plan.LeaseEpoch != 1 || plan.LeaseID != lease || plan.OwnerHostID != hostA {
		t.Fatalf("lease: %+v", plan)
	}
	leaseWinner, err := reader.winningLeaseFor(idB, "direct", reader.Local)
	if err != nil {
		t.Fatal(err)
	}
	if plan.LeaseRecordID != leaseWinner.Digest {
		t.Fatalf("lease_record_id = %q, want %q", plan.LeaseRecordID, leaseWinner.Digest)
	}
	if len(plan.AuthorityHeads) == 0 {
		t.Fatalf("heads are empty: %+v", plan.AuthorityHeads)
	}
	if plan.DestinationHostID == nil || *plan.DestinationHostID != hostB {
		t.Fatalf("destination: %+v", plan)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("fresh peer plan is not current: %v", err)
	}
}

func TestBuildPlanRecordOnlySessionRefusesBootstrap(t *testing.T) {
	local, _ := repository(t)
	create(t, local, idA, "alpha")
	reader := &Reader{Local: local, LocalHostID: hostA}
	// A complete read proving a record without any valid lease is an
	// interrupted persistence prefix for the bootstrap-recovery leaf:
	// no plan binds an empty lease triple or placeholder, and there is
	// nothing current to revalidate.
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrBootstrapIncomplete) {
		t.Fatalf("record-only build = %v, want selector_bootstrap_incomplete", err)
	}
	if _, err := reader.BuildPlan(PlanArgs{Selector: "id:" + idA, Action: ActionStatus}); !errors.Is(err, ErrBootstrapIncomplete) {
		t.Fatalf("record-only id build = %v, want selector_bootstrap_incomplete", err)
	}
}

func TestBuildPlanLocalSourceRequiresKnownHost(t *testing.T) {
	reader, _, _, _ := plannedSession(t)
	reader.LocalHostID = ""
	// The local source cannot be named without the local host
	// identity: the plan binds a UUIDv7 source host or refuses.
	// Peer selections still bind their own known host.
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha", Action: ActionStatus}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("unknown-host local build = %v, want invalid_config", err)
	}
	peer, _ := repository(t)
	ref := create(t, peer, idB, "beta")
	appendEvent(t, peer, idB, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	withLeases(reader, leaseCreate(t, idB, hostA))
	reader.AllowlistedPeerIDs = []string{hostA}
	reader.Peers = []PeerIndex{{hostA, peer}}
	reader.Aliases = map[string]string{hostA: "workstation"}
	plan := mustBuild(t, reader, PlanArgs{Selector: "beta@peer:workstation", Action: ActionStatus})
	if plan.SourceHostID != hostA {
		t.Fatalf("peer source without local host: %+v", plan)
	}
	if err := reader.Revalidate(plan); err != nil {
		t.Fatalf("peer plan without local host is not current: %v", err)
	}
}

func TestBuildPlanArgumentGates(t *testing.T) {
	reader, _, _, _ := plannedSession(t)
	for _, args := range []PlanArgs{
		{Selector: "alpha", Action: "launch"},
		{Selector: "alpha", Action: ""},
		{Selector: "alpha", Action: ActionStatus, Destination: "not-a-uuid"},
		{Selector: "alpha", Action: ActionStatus, Destination: lease},
		{Selector: "alpha", Action: ActionStatus, Expectations: []byte(`{bad`)},
		{Selector: "alpha", Action: ActionStatus, Expectations: []byte(`[1]`)},
		{Selector: "alpha", Action: ActionStatus, Expectations: []byte(`"str"`)},
		{Selector: "alpha@unknown", Action: ActionStatus},
		{Selector: "", Action: ActionStatus},
	} {
		if _, err := reader.BuildPlan(args); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("BuildPlan(%+v) = %v, want invalid_arguments", args, err)
		}
	}
	if _, err := reader.BuildPlan(PlanArgs{Selector: "alpha"}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("missing action: %v", err)
	}
}

func TestPlanDigestAndMarshalRoundTrip(t *testing.T) {
	reader, _, _, _ := plannedSession(t)
	plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionDiff})
	first, err := plan.Digest()
	if err != nil {
		t.Fatal(err)
	}
	second, err := plan.Digest()
	if err != nil || first != second {
		t.Fatalf("digest unstable: %q %q %v", first, second, err)
	}
	raw, err := plan.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParsePlan(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan, parsed) {
		t.Fatalf("round trip diverged:\n%+v\n%+v", plan, parsed)
	}
	// Tampering with any member breaks the locally verified digest.
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	wire["action"] = ActionAttach
	tampered, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePlan(tampered); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("tampered plan parsed: %v", err)
	}
	if _, err := ParsePlan([]byte(`{broken`)); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("malformed encoding: %v", err)
	}
	// A valid persisted encoding is never authority by itself: current
	// facts still decide through Revalidate.
	raw, err = plan.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err = ParsePlan(raw)
	if err != nil {
		t.Fatal(err)
	}
	tailForRoundTrip, err := chainTail(reader.Local, idA)
	if err != nil {
		t.Fatal(err)
	}
	appendEvent(t, reader.Local, idA, tailForRoundTrip, "session.idle", 2, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	if err := reader.Revalidate(parsed); !errors.Is(err, ErrPlanStale) {
		t.Fatalf("persisted digest overrode current facts: %v", err)
	}
}

func TestRevalidateDetectsEachFactChange(t *testing.T) {
	t.Run("same lease event changes heads", func(t *testing.T) {
		reader, _, _, _ := plannedSession(t)
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		tail, err := chainTail(reader.Local, idA)
		if err != nil {
			t.Fatal(err)
		}
		appendEvent(t, reader.Local, idA, tail, "session.idle", 2, map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
		mustStale(t, reader.Revalidate(plan), "authority heads changed")
	})
	t.Run("successor lease changes winner", func(t *testing.T) {
		reader, _, _, created := plannedSession(t)
		launched := appendEvent(t, reader.Local, idA, created, "provider.launched", 2, map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"})
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		appendEventAs(t, reader.Local, idA, launched, "lease.transferred", 1, 2, leaseB, map[string]any{
			"operation_id": hostB, "from_host_id": hostA, "to_host_id": hostA,
			"predecessor_lease_id": lease, "new_lease_id": leaseB,
		})
		mustStale(t, reader.Revalidate(plan), "winning lease changed")
	})
	t.Run("record replacement changes digest", func(t *testing.T) {
		reader, root, _, _ := plannedSession(t)
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		if err := os.WriteFile(filepath.Join(root, "sessions", idA, "record.json"), record(t, idA, "renamed"), 0o600); err != nil {
			t.Fatal(err)
		}
		mustStale(t, reader.Revalidate(plan), "session record changed")
	})
	t.Run("changed nonempty record refuses", func(t *testing.T) {
		reader, root, _, _ := plannedSession(t)
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		// Rewrite both the record and its chain index so the session
		// stays live with a different nonempty digest: a narrowing
		// that admits one changed nonempty digest must still fail.
		renamed := record(t, idA, "renamed-live")
		var renamedMembers map[string]json.RawMessage
		if err := json.Unmarshal(renamed, &renamedMembers); err != nil {
			t.Fatal(err)
		}
		var renamedID string
		if err := json.Unmarshal(renamedMembers["record_id"], &renamedID); err != nil {
			t.Fatal(err)
		}
		if renamedID == plan.SessionRecordID || renamedID == zeroDigest {
			t.Fatalf("renamed fixture did not change the digest: %q", renamedID)
		}
		if err := os.WriteFile(filepath.Join(root, "sessions", idA, "record.json"), renamed, 0o600); err != nil {
			t.Fatal(err)
		}
		chainPath := filepath.Join(root, "sessions", idA, "chain.json")
		chainBytes, err := os.ReadFile(chainPath)
		if err != nil {
			t.Fatal(err)
		}
		var chain map[string]any
		if err := json.Unmarshal(chainBytes, &chain); err != nil {
			t.Fatal(err)
		}
		chain["record_id"] = renamedID
		rewritten, err := json.Marshal(chain)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(chainPath, rewritten, 0o600); err != nil {
			t.Fatal(err)
		}
		mustStale(t, reader.Revalidate(plan), "session record changed")
	})
	t.Run("alias remap changes binding", func(t *testing.T) {
		local, _ := repository(t)
		repoA, rootA := repository(t)
		repoB, rootB := repository(t)
		ref := create(t, repoA, idA, "shared")
		appendEvent(t, repoA, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		copySessionDir(t, rootA, rootB, idA)
		reader := &Reader{
			Local:              local,
			AllowlistedPeerIDs: []string{hostA, hostB},
			Peers:              []PeerIndex{{hostA, repoA}, {hostB, repoB}},
			Aliases:            map[string]string{hostA: "ws"},
		}
		withLeases(reader, leaseCreate(t, idA, hostA))
		plan := mustBuild(t, reader, PlanArgs{Selector: "shared@peer:ws", Action: ActionStatus})
		if plan.SourceHostID != hostA {
			t.Fatalf("source: %+v", plan)
		}
		reader.Aliases = map[string]string{hostB: "ws"}
		mustStale(t, reader.Revalidate(plan), "source binding changed")
	})
	t.Run("unrelated mapping change changes configuration", func(t *testing.T) {
		reader, _, _, _ := plannedSession(t)
		peer, _ := repository(t)
		reader.Peers = []PeerIndex{{hostC, peer}}
		reader.AllowlistedPeerIDs = []string{hostC}
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		reader.Aliases = map[string]string{hostC: "ws"}
		mustStale(t, reader.Revalidate(plan), "configuration changed")
	})
	t.Run("revocation refuses allowlist", func(t *testing.T) {
		local, _ := repository(t)
		peer, _ := repository(t)
		localRef := create(t, local, idA, "alpha")
		appendEvent(t, local, idA, localRef.RecordID, "session.created", 1, map[string]any{"session_record_id": localRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		peerRef := create(t, peer, idB, "beta")
		appendEvent(t, peer, idB, peerRef.RecordID, "session.created", 1, map[string]any{"session_record_id": peerRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		reader := &Reader{
			Local:              local,
			AllowlistedPeerIDs: []string{hostA},
			Peers:              []PeerIndex{{hostA, peer}},
			Aliases:            map[string]string{hostA: "ws"},
		}
		withLeases(reader, leaseCreate(t, idA, hostA), leaseCreate(t, idB, hostA))
		plan := mustBuild(t, reader, PlanArgs{Selector: "beta@peer:ws", Action: ActionStatus})
		reader.AllowlistedPeerIDs = nil
		err := reader.Revalidate(plan)
		if !errors.Is(err, ErrPeerNotAllowlisted) {
			t.Fatalf("revocation = %v, want peer_not_allowlisted", err)
		}
	})
	t.Run("tombstone invalidates live plan", func(t *testing.T) {
		reader, _, ref, created := plannedSession(t)
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		failed := appendEvent(t, reader.Local, idA, created, "session.failed", 2, map[string]any{"error_code": "E_LAUNCH", "retryable": true, "operation_id": nil})
		_ = ref
		appendEvent(t, reader.Local, idA, failed, "session.tombstoned", 3, map[string]any{"tombstone_id": zeroDigest})
		mustStale(t, reader.Revalidate(plan), "authority heads changed")
	})
	t.Run("deletion empties selection", func(t *testing.T) {
		reader, root, _, _ := plannedSession(t)
		// A surviving sibling keeps the index readable, so only the
		// absence check can report the planned identity: substituting
		// the sibling must never validate.
		create(t, reader.Local, idB, "bravo")
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		if err := os.RemoveAll(filepath.Join(root, "sessions", idA)); err != nil {
			t.Fatal(err)
		}
		mustStale(t, reader.Revalidate(plan), "selection absent from source index")
	})
	t.Run("unrelated session change changes index", func(t *testing.T) {
		reader, _, _, _ := plannedSession(t)
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		ref := create(t, reader.Local, idB, "bravo")
		appendEvent(t, reader.Local, idB, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		mustStale(t, reader.Revalidate(plan), "source index changed")
	})
	t.Run("invalid projection propagates", func(t *testing.T) {
		reader, _, ref, created := plannedSession(t)
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		// A terminal close followed by a second bootstrap is
		// continuity-admitted by the repository and refused by the
		// state owner: the failure reaches revalidation unchanged,
		// never as absence or staleness.
		previous := created
		for index, row := range []struct {
			typ     string
			payload map[string]any
		}{
			{"provider.launched", map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"}},
			{"session.idle", map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true}},
			{"checkpoint.created", map[string]any{"checkpoint_id": zeroDigest, "kind": "manual"}},
			{"checkpoint.created", map[string]any{"checkpoint_id": zeroDigest, "kind": "manual"}},
			{"session.stopped", map[string]any{"graceful": true, "checkpoint_id": zeroDigest, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true}},
		} {
			previous = appendEvent(t, reader.Local, idA, previous, row.typ, index+2, row.payload)
		}
		appendEvent(t, reader.Local, idA, previous, "session.created", 7, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		err := reader.Revalidate(plan)
		if err == nil || errors.Is(err, ErrPlanStale) || errors.Is(err, ErrNotFound) {
			t.Fatalf("projection failure became %v", err)
		}
		if !errors.Is(err, sessstate.ErrInvalidTransition) {
			t.Fatalf("projection failure = %v", err)
		}
	})
}

func TestRevalidateReadFailures(t *testing.T) {
	t.Run("peer read failure keeps class", func(t *testing.T) {
		local, _ := repository(t)
		peer, peerRoot := repository(t)
		localRef := create(t, local, idA, "alpha")
		appendEvent(t, local, idA, localRef.RecordID, "session.created", 1, map[string]any{"session_record_id": localRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		peerRef := create(t, peer, idB, "beta")
		appendEvent(t, peer, idB, peerRef.RecordID, "session.created", 1, map[string]any{"session_record_id": peerRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		reader := &Reader{
			Local:              local,
			AllowlistedPeerIDs: []string{hostA},
			Peers:              []PeerIndex{{hostA, peer}},
			Aliases:            map[string]string{hostA: "ws"},
		}
		withLeases(reader, leaseCreate(t, idA, hostA), leaseCreate(t, idB, hostA))
		plan := mustBuild(t, reader, PlanArgs{Selector: "beta@peer:ws", Action: ActionStatus})
		if err := os.Rename(filepath.Join(peerRoot, "sessions"), filepath.Join(peerRoot, "saved")); err != nil {
			t.Fatal(err)
		}
		err := reader.Revalidate(plan)
		if !errors.Is(err, ErrSourceReadFailed) || !errors.Is(err, sessrepo.ErrRepositoryPath) {
			t.Fatalf("peer revalidation failure = %v", err)
		}
	})
	t.Run("local read failure keeps repository error", func(t *testing.T) {
		reader, root, _, _ := plannedSession(t)
		plan := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
		if err := os.Rename(filepath.Join(root, "sessions"), filepath.Join(root, "saved")); err != nil {
			t.Fatal(err)
		}
		err := reader.Revalidate(plan)
		if !errors.Is(err, sessrepo.ErrRepositoryPath) {
			t.Fatalf("local revalidation failure = %v", err)
		}
		if errors.Is(err, ErrSourceReadFailed) {
			t.Fatalf("local failure misclassified as remote: %v", err)
		}
	})
	t.Run("never falls back to another source", func(t *testing.T) {
		local, _ := repository(t)
		peer, peerRoot := repository(t)
		localRef := create(t, local, idA, "beta")
		appendEvent(t, local, idA, localRef.RecordID, "session.created", 1, map[string]any{"session_record_id": localRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		peerRef := create(t, peer, idB, "beta")
		appendEvent(t, peer, idB, peerRef.RecordID, "session.created", 1, map[string]any{"session_record_id": peerRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
		reader := &Reader{
			Local:              local,
			AllowlistedPeerIDs: []string{hostA},
			Peers:              []PeerIndex{{hostA, peer}},
			Aliases:            map[string]string{hostA: "ws"},
		}
		withLeases(reader, leaseCreate(t, idA, hostA), leaseCreate(t, idB, hostA))
		plan := mustBuild(t, reader, PlanArgs{Selector: "beta@peer:ws", Action: ActionStatus})
		if err := os.Rename(filepath.Join(peerRoot, "sessions"), filepath.Join(peerRoot, "saved")); err != nil {
			t.Fatal(err)
		}
		// The local index holds a live "beta" under another UUID, but
		// revalidation rereads the plan source only: no substitution.
		if err := reader.Revalidate(plan); !errors.Is(err, ErrSourceReadFailed) {
			t.Fatalf("revalidation fell back: %v", err)
		}
	})
}

func TestRevalidateMalformedPlans(t *testing.T) {
	reader, _, _, _ := plannedSession(t)
	valid := mustBuild(t, reader, PlanArgs{Selector: "alpha", Action: ActionStatus})
	cases := []struct {
		name   string
		mutate func(plan *SelectionPlan)
	}{
		{"version", func(plan *SelectionPlan) { plan.SelectorVersion = "2.0.0" }},
		{"selector", func(plan *SelectionPlan) { plan.Selector = "alpha@unknown" }},
		{"session", func(plan *SelectionPlan) { plan.SessionID = "nope" }},
		{"record", func(plan *SelectionPlan) { plan.SessionRecordID = "nope" }},
		{"source host", func(plan *SelectionPlan) { plan.SourceHostID = "nope" }},
		{"empty source host", func(plan *SelectionPlan) { plan.SourceHostID = "" }},
		{"lease record", func(plan *SelectionPlan) { plan.LeaseRecordID = "nope" }},
		{"empty lease record", func(plan *SelectionPlan) { plan.LeaseRecordID = "" }},
		{"empty lease triple", func(plan *SelectionPlan) {
			plan.LeaseEpoch, plan.LeaseID, plan.OwnerHostID = 0, "", ""
		}},
		{"alias control", func(plan *SelectionPlan) {
			bad := "a\tb"
			plan.SourceAlias = &bad
		}},
		{"index digest", func(plan *SelectionPlan) { plan.SourceIndexDigest = "nope" }},
		{"configuration digest", func(plan *SelectionPlan) { plan.ConfigurationDigest = "nope" }},
		{"expectation digest", func(plan *SelectionPlan) { plan.ExpectationDigest = "nope" }},
		{"partial lease", func(plan *SelectionPlan) { plan.LeaseID = "" }},
		{"epoch zero with lease", func(plan *SelectionPlan) { plan.LeaseEpoch = 0 }},
		{"epoch overflow", func(plan *SelectionPlan) { plan.LeaseEpoch = 1 << 53 }},
		{"lease shape", func(plan *SelectionPlan) { plan.LeaseID = "nope" }},
		{"owner shape", func(plan *SelectionPlan) { plan.OwnerHostID = "nope" }},
		{"head shape", func(plan *SelectionPlan) { plan.AuthorityHeads = []string{"nope"} }},
		{"heads unsorted", func(plan *SelectionPlan) {
			plan.AuthorityHeads = []string{"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", zeroDigest}
		}},
		{"heads duplicated", func(plan *SelectionPlan) {
			plan.AuthorityHeads = []string{zeroDigest, zeroDigest}
		}},
		{"action", func(plan *SelectionPlan) { plan.Action = "launch" }},
		{"destination", func(plan *SelectionPlan) {
			bad := "nope"
			plan.DestinationHostID = &bad
		}},
	}
	for _, row := range cases {
		t.Run(row.name, func(t *testing.T) {
			plan := valid
			row.mutate(&plan)
			if err := reader.Revalidate(plan); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("Revalidate = %v, want invalid_arguments", err)
			}
			if _, err := plan.Digest(); err != nil {
				t.Fatalf("Digest = %v", err)
			}
		})
	}
	if err := reader.Revalidate(SelectionPlan{}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("empty plan accepted")
	}
	if err := (&Reader{}).Revalidate(valid); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("reader without index accepted")
	}
}

func TestActionBoundaryTable(t *testing.T) {
	matrix := map[string][]string{
		ActionStatus:      {BoundaryProjection, BoundaryRetry},
		ActionAttach:      {BoundaryPreEffect, BoundaryFencing, BoundaryRetry, BoundaryRemoteDispatch, BoundaryRemoteAdmission, BoundaryTransportResume},
		ActionTakeover:    {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
		ActionFork:        {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
		ActionStop:        {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
		ActionResume:      {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
		ActionSync:        {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery, BoundaryTransportResume},
		ActionDiff:        {BoundaryProjection, BoundaryRetry},
		ActionMaterialize: {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery},
		ActionSetProfile:  {BoundaryPreEffect, BoundaryFencing, BoundaryCommit, BoundaryRetry, BoundaryRecovery},
		ActionLogs:        {BoundaryProjection, BoundaryRetry, BoundaryRemoteDispatch, BoundaryRemoteAdmission, BoundaryTransportResume},
		ActionCancel:      {BoundaryProjection},
	}
	if len(planBoundaries) != len(matrix) {
		t.Fatalf("boundary table covers %d actions, want %d", len(planBoundaries), len(matrix))
	}
	for action, want := range matrix {
		got, ok := BoundariesFor(action)
		if !ok || !reflect.DeepEqual(got, want) {
			t.Fatalf("BoundariesFor(%q) = %v, %v", action, got, ok)
		}
	}
	for _, action := range []string{"", "launch", "STATUS", "attach --local"} {
		if _, ok := BoundariesFor(action); ok {
			t.Fatalf("BoundariesFor(%q) admitted", action)
		}
	}
}
