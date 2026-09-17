package sessquery

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

func TestResolvePinnedPrecedence(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	// Both sessions carry their authoritative initial lease so the
	// status half of each precedence row stays representable.
	localRef := create(t, local, idA, "Local")
	appendEvent(t, local, idA, localRef.RecordID, "session.created", 1, map[string]any{"session_record_id": localRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	peerRef := create(t, peer, idB, idA)
	appendEvent(t, peer, idB, peerRef.RecordID, "session.created", 1, map[string]any{"session_record_id": peerRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	reader := &Reader{Local: local, AllowlistedPeerIDs: []string{hostA}, Peers: []PeerIndex{{hostA, peer}}}
	for _, row := range []struct{ selector, id, peer string }{
		{"Local", idA, ""}, {idA, idB, hostA}, {idB, idB, hostA},
	} {
		got, err := reader.Resolve(row.selector)
		if err != nil || got.SessionID != row.id || got.PeerHostID != row.peer {
			t.Fatalf("Resolve(%q) = %+v, %v", row.selector, got, err)
		}
		status, err := reader.Status(row.selector)
		if err != nil || status.Projection.SessionID != row.id {
			t.Fatalf("Status(%q) = %+v, %v", row.selector, status, err)
		}
	}
	// Local exact name wins even if a peer learned the same name for another ID.
	other, _ := repository(t)
	create(t, other, idB, "Local")
	reader.Peers = []PeerIndex{{hostA, other}}
	got, err := reader.Resolve("Local")
	if err != nil || got.SessionID != idA || got.PeerHostID != "" {
		t.Fatalf("local priority: %+v %v", got, err)
	}
	// A local UUID-shaped name also wins over the UUID identity.
	local2, _ := repository(t)
	create(t, local2, idA, "Original")
	create(t, local2, idB, idA)
	got, err = (&Reader{Local: local2}).Resolve(idA)
	if err != nil || got.SessionID != idB {
		t.Fatalf("local name before UUID: %+v %v", got, err)
	}
}

func TestResolveExactNamesAndASCIICollisions(t *testing.T) {
	for _, peers := range []bool{false, true} {
		t.Run(map[bool]string{false: "local", true: "peers"}[peers], func(t *testing.T) {
			local, _ := repository(t)
			source := local
			r := &Reader{Local: local}
			if peers {
				source, _ = repository(t)
				r.AllowlistedPeerIDs = []string{hostA}
				r.Peers = []PeerIndex{{hostA, source}}
			}
			create(t, source, idA, "Az-09._")
			for _, query := range []string{"az-09._", "AZ-09._", "Ａz-09._", "no-such-session", "host/Name"} {
				if _, err := r.Resolve(query); !errors.Is(err, ErrNotFound) {
					t.Fatalf("%q: %v", query, err)
				}
			}
			// v0.6.0 Section 14.7.1 decides the qualified-selector
			// grammar the partial leaf left open: "Name@host" is a
			// qualified selector with an unknown source, and the empty
			// argument carries no key, so both are invalid arguments,
			// never silent names. "host/Name" carries no "@" and stays
			// not found.
			for _, query := range []string{"Name@host", "", "Az-09._@unknown", "Az-09._@", "@local"} {
				if _, err := r.Resolve(query); !errors.Is(err, ErrInvalidArgument) {
					t.Fatalf("%q: %v", query, err)
				}
			}
			if _, err := r.Resolve("Az-09._"); err != nil {
				t.Fatal(err)
			}
			create(t, source, idB, "aZ-09._")
			for _, query := range []string{"Az-09._", "aZ-09._", "AZ-09._"} {
				if _, err := r.Resolve(query); !errors.Is(err, ErrAmbiguous) {
					t.Fatalf("%q: %v", query, err)
				}
				if _, err := r.Status(query); !errors.Is(err, ErrAmbiguous) {
					t.Fatalf("status %q: %v", query, err)
				}
			}
		})
	}
}

func TestResolveAllowlistAndReadFailures(t *testing.T) {
	local, root := repository(t)
	peer, peerRoot := repository(t)
	create(t, local, idA, "Local")
	create(t, peer, idB, idA)
	r := &Reader{Local: local, Peers: []PeerIndex{{hostA, peer}}}
	got, err := r.Resolve(idA)
	if err != nil || got.SessionID != idA {
		t.Fatalf("unallowlisted peer affected UUID: %+v %v", got, err)
	}
	// No learned index is absence; a known index whose read fails is not.
	r.AllowlistedPeerIDs = []string{hostB}
	got, err = r.Resolve(idA)
	if err != nil || got.SessionID != idA {
		t.Fatalf("absent learned index: %+v %v", got, err)
	}
	r.AllowlistedPeerIDs = []string{hostA}
	if err := os.Rename(filepath.Join(peerRoot, "sessions"), filepath.Join(peerRoot, "saved")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Resolve(idA); !errors.Is(err, sessrepo.ErrRepositoryPath) {
		t.Fatalf("failed peer read fell through: %v", err)
	}
	// Local exact name does not depend on lower-tier reads.
	if _, err := r.Resolve("Local"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peerRoot, "sessions"), []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Status(idA); !errors.Is(err, sessrepo.ErrRepositoryPath) {
		t.Fatalf("non-directory read: %v", err)
	}
	r.Peers = []PeerIndex{{hostA, nil}}
	if _, err := r.Resolve(idA); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil learned index: %v", err)
	}
	r.Peers = []PeerIndex{{hostA, nil}, {hostA, peer}}
	if _, err := r.Resolve(idA); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("duplicate learned index: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(root, "sessions")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.List(); !errors.Is(err, sessrepo.ErrRepositoryPath) {
		t.Fatalf("missing local root: %v", err)
	}
	if _, err := r.Resolve("Local"); !errors.Is(err, sessrepo.ErrRepositoryPath) {
		t.Fatalf("missing root became not found: %v", err)
	}
}

func TestResolvePeerOrderAndReplicatedIdentity(t *testing.T) {
	local, _ := repository(t)
	a, _ := repository(t)
	b, _ := repository(t)
	create(t, a, idA, "Shared")
	create(t, b, idA, "Shared")
	r := &Reader{Local: local, AllowlistedPeerIDs: []string{hostB, hostA, hostB}, Peers: []PeerIndex{{hostB, b}, {hostA, a}}}
	first, err := r.Resolve("Shared")
	if err != nil || first.PeerHostID != hostA {
		t.Fatalf("replicated identity: %+v %v", first, err)
	}
	r.Peers[0], r.Peers[1] = r.Peers[1], r.Peers[0]
	second, err := r.Resolve("Shared")
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("permutation changed selection: %+v %v", second, err)
	}
	create(t, b, idB, "sHARED")
	if _, err := r.Resolve("Shared"); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("cross-peer collision: %v", err)
	}
}

func TestListStatusDerivedFactsAndStableOrder(t *testing.T) {
	repo, _ := repository(t)
	// Both listed sessions carry their authoritative initial lease:
	// a record without any valid lease is an interrupted prefix the
	// list refuses (see the bootstrap refusal tests), never a row
	// with invented owner facts.
	zed := create(t, repo, idB, "Zed")
	appendEvent(t, repo, idB, zed.RecordID, "session.created", 1, map[string]any{"session_record_id": zed.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	ref := create(t, repo, idA, "Alpha")
	appendEvent(t, repo, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	r := &Reader{Local: repo, LocalHostID: hostA}
	list, err := r.List()
	if err != nil || len(list) != 2 {
		t.Fatalf("List: %+v %v", list, err)
	}
	if list[0].Projection.SessionID != idA || list[1].Projection.SessionID != idB {
		t.Fatalf("order: %+v", list)
	}
	p := list[0].Projection
	if p.Name != "Alpha" || p.Kind != "direct" || p.Provider.ID != "codex" || p.State != sessstate.StateCreating || p.OwnerHostID != hostA || p.LocalRole != "owner" || p.Winner.LeaseID != lease || p.Winner.Epoch != 1 {
		t.Fatalf("derived summary: %+v", p)
	}
	second := list[1].Projection
	if second.Name != "Zed" || second.OwnerHostID != hostA || second.Winner.LeaseID != lease || second.Winner.Epoch != 1 {
		t.Fatalf("second summary: %+v", second)
	}
	for _, selector := range []string{"Alpha", idA} {
		status, err := r.Status(selector)
		if err != nil || !reflect.DeepEqual(status, list[0]) {
			t.Fatalf("status disagrees: %+v %v", status, err)
		}
	}
	again, err := r.List()
	if err != nil || !reflect.DeepEqual(list, again) {
		t.Fatalf("read is not idempotent: %+v %v", again, err)
	}
	r.LocalHostID = hostB
	status, err := r.Status(idA)
	if err != nil || status.Projection.LocalRole != "replica" {
		t.Fatalf("replica: %+v %v", status, err)
	}
}

func TestParkedReadRecoveryAndMissingVsMalformed(t *testing.T) {
	repo, root := repository(t)
	raw := record(t, idA, "Parked")
	repo.AfterCreateStep = func(step sessrepo.CreateStep) error {
		if step == sessrepo.CreateStepRecord {
			return errors.New("crash")
		}
		return nil
	}
	if _, err := repo.CreateSession(raw); err == nil {
		t.Fatal("fault did not fire")
	}
	createHealthy := func() {
		repo.AfterCreateStep = nil
		healthy := create(t, repo, idB, "Healthy")
		appendEvent(t, repo, idB, healthy.RecordID, "session.created", 1, map[string]any{"session_record_id": healthy.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	}
	createHealthy()
	r := &Reader{Local: repo}
	list, err := r.List()
	if err != nil || len(list) != 2 || list[0].Projection.Parked == nil {
		t.Fatalf("parked omitted: %+v %v", list, err)
	}
	// A parked row carries no invented owner: the read failed, so no
	// winning lease was observed and none is claimed.
	if parked := list[0].Projection; parked.OwnerHostID != "" || parked.LocalRole != "" || !parked.Winner.IsZero() {
		t.Fatalf("invented parked owner: %+v", parked)
	}
	if _, err := r.Resolve("Parked"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("parked routed: %v", err)
	}
	if _, err := r.Resolve(idA); !errors.Is(err, ErrNotFound) {
		t.Fatalf("parked UUID routed: %v", err)
	}
	if _, err := r.Resolve("Healthy"); err != nil {
		t.Fatal(err)
	}
	parked, err := r.InspectLocal(idA)
	if err != nil || parked.Projection.Parked == nil || parked.Projection.Parked.RetryHint == "" {
		t.Fatalf("parked status: %+v %v", parked, err)
	}
	if _, err := repo.CreateSession(raw); err != nil {
		t.Fatal(err)
	}
	// The healed record alone is still an interrupted prefix: only
	// its authoritative initial lease makes it a status again. The
	// retried record is byte-identical, so the parked row's digest
	// still names it.
	if _, err := r.Status("Parked"); !errors.Is(err, ErrBootstrapIncomplete) {
		t.Fatalf("healed record-only status: %v", err)
	}
	recordID := list[0].Projection.RecordID
	appendEvent(t, repo, idA, recordID, "session.created", 1, map[string]any{"session_record_id": recordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	if _, err := r.Status("Parked"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sessions", idA, "record.json"), []byte("malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	damaged, err := r.InspectLocal(idA)
	if err != nil || damaged.Projection.State != sessstate.StateParked || damaged.Projection.Parked.BlockingReason == "" {
		t.Fatalf("malformed treated as absent: %+v %v", damaged, err)
	}
	if _, err := r.InspectLocal(hostA); !errors.Is(err, sessstate.ErrUnknownSession) {
		t.Fatalf("missing: %v", err)
	}
}

func TestReaderAbsentAndNil(t *testing.T) {
	for _, r := range []*Reader{nil, {}} {
		if _, err := r.List(); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
		if _, err := r.InspectLocal(idA); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
		if _, err := r.Status(idA); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	}
	repo, _ := repository(t)
	rows, err := (&Reader{Local: repo}).List()
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty: %+v %v", rows, err)
	}
	if _, err := (&Reader{Local: repo}).Status(idA); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestResolveExcludesTombstonedButListsIt(t *testing.T) {
	repo, _ := repository(t)
	ref := create(t, repo, idA, "Deleted")
	event := appendEvent(t, repo, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	event = appendEvent(t, repo, idA, event, "session.failed", 2, map[string]any{"error_code": "E_LAUNCH", "retryable": true, "operation_id": nil})
	appendEvent(t, repo, idA, event, "session.tombstoned", 3, map[string]any{"tombstone_id": zeroDigest})
	r := &Reader{Local: repo}
	rows, err := r.List()
	if err != nil || len(rows) != 1 || rows[0].Projection.State != sessstate.StateTombstoned {
		t.Fatalf("tombstone lost: %+v %v", rows, err)
	}
	for _, selector := range []string{"Deleted", idA} {
		if _, err := r.Resolve(selector); !errors.Is(err, ErrNotFound) {
			t.Fatalf("tombstone routed: %v", err)
		}
	}
}

func TestListStatusCheckpointAndProjectionFailure(t *testing.T) {
	repo, _ := repository(t)
	ref := create(t, repo, idA, "Checkpointed")
	previous := ref.RecordID
	rows := []struct {
		typ     string
		payload map[string]any
	}{
		{"session.created", map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB}},
		{"provider.launched", map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"}},
		{"session.idle", map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true}},
		{"checkpoint.created", map[string]any{"checkpoint_id": zeroDigest, "kind": "manual"}},
		{"checkpoint.created", map[string]any{"checkpoint_id": zeroDigest, "kind": "manual"}},
		{"session.stopped", map[string]any{"graceful": true, "checkpoint_id": zeroDigest, "resumable": true, "closure_kind": "checkpointed", "process_closed": true, "store_closed": true}},
	}
	for index, row := range rows {
		previous = appendEvent(t, repo, idA, previous, row.typ, index+1, row.payload)
	}
	r := &Reader{Local: repo, LocalHostID: hostA}
	status, err := r.Status("Checkpointed")
	if err != nil || status.Projection.State != sessstate.StateStopped || !status.Projection.HasCheckpoint || status.Projection.Newest.ID != zeroDigest || status.Projection.Provider.Version != "0.147.0" {
		t.Fatalf("checkpoint summary: %+v %v", status, err)
	}
	list, err := r.List()
	if err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], status) {
		t.Fatalf("checkpoint list/status disagreement: %+v %v", list, err)
	}
	// Repository continuity admits this event; the state owner's terminal-state
	// transition gate refuses it. That failure must reach the read entry unchanged.
	appendEvent(t, repo, idA, previous, "session.created", 7, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	if _, err := r.List(); err == nil {
		t.Fatal("invalid projection was returned as a summary")
	}
	if _, err := r.Status(idA); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid projection became absence: %v", err)
	}
}
