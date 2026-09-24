package merkleinventory

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

func TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace(t *testing.T) {
	_, fixtures := inventoryObjectsByNamespace(t)
	for _, namespace := range durableJSONNamespaces {
		t.Run(namespace, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "union")
			data := fixtures[namespace]
			if code := crashDurableAdd(t, root, data, "object-install"); code != 73 {
				t.Fatalf("crash child exit = %d, want simulated process exit 73", code)
			}
			store, err := OpenDurable(root)
			if err != nil {
				t.Fatalf("OpenDurable after object-install crash: %v", err)
			}
			membership, err := ClassifyJSON(data)
			if err != nil {
				t.Fatal(err)
			}
			got, err := store.Object(namespace, membership.ID.String())
			if err != nil || !bytes.Equal(got, data) {
				t.Fatalf("Object after crash = %q, %v; want exact admitted bytes", got, err)
			}
			if err := store.AddJSON(data); err != nil {
				t.Fatalf("identical retry after restart = %v", err)
			}
			reopened, err := OpenDurable(root)
			if err != nil {
				t.Fatalf("OpenDurable after idempotent retry: %v", err)
			}
			rootAfter, err := reopened.Root(namespace)
			if err != nil || rootAfter.Count.Uint64() != 1 {
				t.Fatalf("root after identical crash/restart retry = %#v, %v; want count 1", rootAfter, err)
			}
		})
	}
}

func TestDurableConflictCrashRestoresQuarantineAndAbortsSync(t *testing.T) {
	original := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(original)
	if err != nil {
		t.Fatal(err)
	}
	variant := append([]byte("\n"), original...)
	for _, mode := range []string{"quarantine-candidate", "quarantine-pair", "quarantine-active-removed"} {
		t.Run(mode, func(t *testing.T) {
			localRoot := filepath.Join(t.TempDir(), "local")
			local, err := OpenDurable(localRoot)
			if err != nil {
				t.Fatal(err)
			}
			if err := local.AddJSON(original); err != nil {
				t.Fatal(err)
			}
			if code := crashDurableAdd(t, localRoot, variant, mode); code != 73 {
				t.Fatalf("quarantine crash child exit = %d, want simulated process exit 73", code)
			}
			reopened, err := OpenDurable(localRoot)
			if err != nil {
				t.Fatalf("OpenDurable after %s crash: %v", mode, err)
			}
			if !slices.Contains(reopened.index.QuarantinedIDs(), membership.ID.String()) {
				t.Fatalf("QuarantinedIDs after crash = %q, want %s", reopened.index.QuarantinedIDs(), membership.ID.String())
			}
			if _, err := reopened.Object("record", membership.ID.String()); !hasCode(err, "integrity_failure") {
				t.Fatalf("Object(quarantined identity) = %v, want literal integrity_failure", err)
			}
			activeRoot, err := reopened.Root("record")
			if err != nil || activeRoot.Count.Uint64() != 0 {
				t.Fatalf("record root after quarantine recovery = %#v, %v; want zero active records", activeRoot, err)
			}

			peer, err := OpenDurable(filepath.Join(t.TempDir(), "peer"))
			if err != nil {
				t.Fatal(err)
			}
			if err := peer.AddJSON(original); err != nil {
				t.Fatal(err)
			}
			if err := reopened.SyncFrom(peer, testRequestIDs()); !hasCode(err, "integrity_failure") || !strings.Contains(err.Error(), "integrity_failure") {
				t.Fatalf("SyncFrom(peer with quarantined local identity) = %v, want literal integrity_failure", err)
			}
		})
	}
}

func TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes(t *testing.T) {
	original := validSessionRecordJSON(t)
	variant := append([]byte("\n"), original...)
	local, err := OpenDurable(filepath.Join(t.TempDir(), "local"))
	if err != nil {
		t.Fatal(err)
	}
	peer, err := OpenDurable(filepath.Join(t.TempDir(), "peer"))
	if err != nil {
		t.Fatal(err)
	}
	if err := local.AddJSON(original); err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(variant); err != nil {
		t.Fatal(err)
	}
	localRoot, err := local.Root("record")
	peerRoot, peerErr := peer.Root("record")
	if err != nil || peerErr != nil || localRoot != peerRoot {
		t.Fatalf("precondition: semantic roots should match for same ID bytes variants; local=%#v/%v peer=%#v/%v", localRoot, err, peerRoot, peerErr)
	}
	err = local.SyncFrom(peer, testRequestIDs())
	if !hasCode(err, "integrity_failure") || !strings.Contains(err.Error(), "integrity_failure") {
		t.Fatalf("SyncFrom(common same-digest/different-byte identity) = %v, want literal integrity_failure", err)
	}
	membership, err := ClassifyJSON(original)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenDurable(local.root)
	if err != nil {
		t.Fatalf("OpenDurable after sync conflict: %v", err)
	}
	if !slices.Contains(reopened.index.QuarantinedIDs(), membership.ID.String()) {
		t.Fatalf("quarantine after common-ID audit = %q, want %s", reopened.index.QuarantinedIDs(), membership.ID.String())
	}
}

func TestDurableSyncFindsMissingJSONObjectsAndRetainsTombstoneRecords(t *testing.T) {
	_, fixtures := inventoryObjectsByNamespace(t)
	peer, err := OpenDurable(filepath.Join(t.TempDir(), "peer"))
	if err != nil {
		t.Fatal(err)
	}
	// Acknowledgement arrives first. The union persists both immutable
	// records, then validates their reference after the full object set arrives.
	for _, namespace := range []string{"tombstone_ack", "event", "record", "manifest", "tombstone"} {
		if err := peer.AddJSON(fixtures[namespace]); err != nil {
			t.Fatalf("peer AddJSON(%s): %v", namespace, err)
		}
	}
	local, err := OpenDurable(filepath.Join(t.TempDir(), "local"))
	if err != nil {
		t.Fatal(err)
	}
	if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
		t.Fatalf("SyncFrom(missing objects) = %v", err)
	}
	for _, namespace := range durableJSONNamespaces {
		want, err := peer.Root(namespace)
		got, gotErr := local.Root(namespace)
		if err != nil || gotErr != nil || got != want {
			t.Errorf("post-sync %s root = %#v/%v, want peer %#v/%v", namespace, got, gotErr, want, err)
		}
		id := objectID(t, fixtures[namespace])
		if gotBytes, readErr := local.Object(namespace, id); readErr != nil || !bytes.Equal(gotBytes, fixtures[namespace]) {
			t.Errorf("post-sync %s object = %q, %v", namespace, gotBytes, readErr)
		}
	}

	repo, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateSession(fixtures["record"]); err != nil {
		t.Fatal(err)
	}
	rowsBefore, err := repo.ListSessions()
	if err != nil || len(rowsBefore) != 1 {
		t.Fatalf("ListSessions before Tombstone union = %d/%v, want one", len(rowsBefore), err)
	}
	if err := local.ValidateUnionClosure(); err != nil {
		t.Fatalf("ValidateUnionClosure = %v", err)
	}
	rowsAfter, err := repo.ListSessions()
	if err != nil || len(rowsAfter) != 1 || rowsAfter[0].SessionID != testSessionID {
		t.Fatalf("ListSessions after Tombstone/Acknowledgement union = %#v/%v, want retained session %s", rowsAfter, err, testSessionID)
	}
}

func TestDurableSyncRefusesUnclosedAcknowledgementThenRecovers(t *testing.T) {
	_, fixtures := inventoryObjectsByNamespace(t)
	peer, err := OpenDurable(filepath.Join(t.TempDir(), "peer"))
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(fixtures["tombstone_ack"]); err != nil {
		t.Fatal(err)
	}
	local, err := OpenDurable(filepath.Join(t.TempDir(), "local"))
	if err != nil {
		t.Fatal(err)
	}
	err = local.SyncFrom(peer, testRequestIDs())
	if err == nil || !strings.Contains(err.Error(), "integrity_failure") {
		t.Fatalf("SyncFrom(unclosed acknowledgement) = %v, want literal integrity_failure", err)
	}
	if root, rootErr := local.Root("tombstone_ack"); rootErr != nil || root.Count.Uint64() != 1 {
		t.Fatalf("acknowledgement after closure refusal = %#v/%v, want retained record for later recovery", root, rootErr)
	}
	if root, rootErr := local.Root("tombstone"); rootErr != nil || root.Count.Uint64() != 0 {
		t.Fatalf("Tombstone root before recovery = %#v/%v, want empty", root, rootErr)
	}
	if err := local.AddJSON(fixtures["tombstone"]); err != nil {
		t.Fatalf("AddJSON(Tombstone recovery) = %v", err)
	}
	if err := local.ValidateUnionClosure(); err != nil {
		t.Fatalf("ValidateUnionClosure after the matching Tombstone arrives = %v", err)
	}
}

func TestDurableSyncRefusesAcknowledgementWithMismatchedTombstoneSubject(t *testing.T) {
	_, fixtures := inventoryObjectsByNamespace(t)
	var acknowledgement map[string]any
	if err := json.Unmarshal(fixtures["tombstone_ack"], &acknowledgement); err != nil {
		t.Fatal(err)
	}
	acknowledgement["subject_id"] = "0198f4c8-3e70-7a11-8a2b-1234567890ac"
	invalidAcknowledgement := identityJSON(t, acknowledgement)
	peer, err := OpenDurable(filepath.Join(t.TempDir(), "peer"))
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range [][]byte{fixtures["tombstone"], invalidAcknowledgement} {
		if err := peer.AddJSON(object); err != nil {
			t.Fatalf("peer AddJSON(validated object with mismatched link): %v", err)
		}
	}
	local, err := OpenDurable(filepath.Join(t.TempDir(), "local"))
	if err != nil {
		t.Fatal(err)
	}
	if err := local.SyncFrom(peer, testRequestIDs()); err == nil || !strings.Contains(err.Error(), "integrity_failure") {
		t.Fatalf("SyncFrom(Tombstone/Acknowledgement subject mismatch) = %v, want literal integrity_failure", err)
	}
	for namespace, object := range map[string][]byte{"tombstone": fixtures["tombstone"], "tombstone_ack": invalidAcknowledgement} {
		id := objectID(t, object)
		if got, err := local.Object(namespace, id); err != nil || !bytes.Equal(got, object) {
			t.Errorf("Object(%s, %s) after closure refusal = %q/%v, want retained immutable bytes", namespace, id, got, err)
		}
	}
}

func TestProjectionRebuildRefusesUnclosedAcknowledgement(t *testing.T) {
	_, fixtures := inventoryObjectsByNamespace(t)
	repo, err := sessrepo.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	record := validSessionRecordJSON(t)
	recordMembership, err := ClassifyJSON(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateSession(record); err != nil {
		t.Fatal(err)
	}
	event := unionEventJSON(t, recordMembership.ID.String(), "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "session.created", "2026-08-19T04:08:00.000Z", map[string]any{
		"session_record_id": recordMembership.ID.String(), "bootstrap_operation_id": testRequestID,
		"first_checkpoint_operation_id": "0198f4c8-6c30-7d44-8d5e-1234567890ac",
	})
	store, err := OpenDurable(filepath.Join(t.TempDir(), "union"))
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range [][]byte{record, event, unionLeaseJSON(t, "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", testHostID, "2026-08-19T04:09:00.000Z"), fixtures["tombstone_ack"]} {
		if err := store.AddJSON(object); err != nil {
			t.Fatalf("DurableIndex.AddJSON: %v", err)
		}
	}
	if _, err := repo.AppendEvent(testSessionID, event); err != nil {
		t.Fatalf("Repository.AppendEvent: %v", err)
	}
	if _, err := store.RebuildProjection(repo, testSessionID); err == nil || !strings.Contains(err.Error(), "integrity_failure") {
		t.Fatalf("RebuildProjection(unclosed acknowledgement) = %v, want literal integrity_failure", err)
	}
}

func TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority(t *testing.T) {
	timestamps := []struct {
		leaseA, leaseB, eventA, eventB, tombstone, acknowledgement string
	}{
		{"1900-01-01T00:00:00.000Z", "2099-12-31T23:59:59.999Z", "1901-01-01T00:00:00.000Z", "2098-01-01T00:00:00.000Z", "1902-01-01T00:00:00.000Z", "2097-01-01T00:00:00.000Z"},
		{"2099-12-31T23:59:59.999Z", "1900-01-01T00:00:00.000Z", "2098-01-01T00:00:00.000Z", "1901-01-01T00:00:00.000Z", "2097-01-01T00:00:00.000Z", "1902-01-01T00:00:00.000Z"},
	}
	for profileIndex, profile := range timestamps {
		record := validSessionRecordJSON(t)
		recordMembership, err := ClassifyJSON(record)
		if err != nil {
			t.Fatal(err)
		}
		repo, err := sessrepo.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.CreateSession(record); err != nil {
			t.Fatal(err)
		}
		leaseA := unionLeaseJSON(t, "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", testHostID, profile.leaseA)
		leaseBBytes := unionLeaseJSON(t, "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff", "0198f4c8-4a10-7b22-8b3c-2234567890ab", profile.leaseB)
		eventA := unionEventJSON(t, recordMembership.ID.String(), "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "session.created", profile.eventA, map[string]any{
			"session_record_id": recordMembership.ID.String(), "bootstrap_operation_id": testRequestID,
			"first_checkpoint_operation_id": "0198f4c8-6c30-7d44-8d5e-1234567890ac",
		})
		if _, err := repo.AppendEvent(testSessionID, eventA); err != nil {
			t.Fatalf("AppendEvent(authoritative branch): %v", err)
		}
		eventB := unionEventJSON(t, recordMembership.ID.String(), "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "session.idle", profile.eventB, map[string]any{
			"boundary_ref": "losing-branch", "foreground_idle": true, "background_idle": true,
		})
		tombstone := tombstoneJSON(t, profile.tombstone)
		tombstoneMembership, err := ClassifyJSON(tombstone)
		if err != nil {
			t.Fatal(err)
		}
		acknowledgement := tombstoneAckJSON(t, tombstoneMembership.ID.String(), "applied", "", profile.acknowledgement)
		variableSet := [][]byte{leaseA, leaseBBytes, eventA, eventB, tombstone, acknowledgement}
		// Exercise the durable public entry once for each opposite timestamp
		// profile. The exhaustive permutations below use the same production
		// union Index/RebuildProjection path without repeating filesystem fsync;
		// per-namespace crash/restart durability is covered above.
		durable, err := OpenDurable(filepath.Join(t.TempDir(), "durable-union"))
		if err != nil {
			t.Fatal(err)
		}
		if err := durable.AddJSON(record); err != nil {
			t.Fatal(err)
		}
		for _, object := range variableSet {
			if err := durable.AddJSON(object); err != nil {
				t.Fatalf("durable AddJSON: %v", err)
			}
		}
		durableProjection, err := durable.RebuildProjection(repo, testSessionID)
		if err != nil {
			t.Fatalf("durable RebuildProjection(profile %d) = %v", profileIndex, err)
		}
		assertIndependentUnionProjectionOracle(t, durableProjection)

		oracleObjects := make(map[string]map[string][]byte)
		for _, object := range append([][]byte{record}, variableSet...) {
			membership, err := ClassifyJSON(object)
			if err != nil || membership.Excluded {
				t.Fatalf("oracle fixture ClassifyJSON = %#v, %v", membership, err)
			}
			if oracleObjects[membership.Namespace] == nil {
				oracleObjects[membership.Namespace] = map[string][]byte{}
			}
			if _, exists := oracleObjects[membership.Namespace][membership.ID.String()]; exists {
				t.Fatalf("fixture set repeats identity %s", membership.ID)
			}
			oracleObjects[membership.Namespace][membership.ID.String()] = bytes.Clone(object)
		}
		orders := allPermutations(len(variableSet))
		var referenceProjection sessstate.Projection
		var referenceRootIDs map[string]string
		for orderIndex, order := range orders {
			index, err := New(RPC2Version)
			if err != nil {
				t.Fatal(err)
			}
			if err := index.AddUnionJSON(record); err != nil {
				t.Fatal(err)
			}
			for _, position := range order {
				if err := index.AddUnionJSON(variableSet[position]); err != nil {
					t.Fatalf("profile %d permutation %d AddJSON(%d) = %v", profileIndex, orderIndex, position, err)
				}
			}
			projection, err := index.RebuildProjection(repo, testSessionID)
			if err != nil {
				t.Fatalf("profile %d permutation %d RebuildProjection = %v", profileIndex, orderIndex, err)
			}
			assertIndependentUnionProjectionOracle(t, projection)
			if orderIndex == 0 {
				referenceProjection = projection
			} else if !reflect.DeepEqual(projection, referenceProjection) {
				t.Fatalf("profile %d permutation %d changed projection:\n got %#v\nwant %#v", profileIndex, orderIndex, projection, referenceProjection)
			}

			for namespace, expected := range oracleObjects {
				ids := index.objectIDs(namespace)
				wantIDs := make([]string, 0, len(expected))
				for id := range expected {
					wantIDs = append(wantIDs, id)
				}
				slices.Sort(wantIDs)
				if !slices.Equal(ids, wantIDs) {
					t.Fatalf("profile %d permutation %d union IDs for %s = %q, want oracle set %q", profileIndex, orderIndex, namespace, ids, wantIDs)
				}
				for id, wantBytes := range expected {
					index.mu.RLock()
					stored, exists := index.objects[namespace][id]
					var gotBytes []byte
					if exists {
						gotBytes = bytes.Clone(stored.data)
					}
					index.mu.RUnlock()
					if !exists || !bytes.Equal(gotBytes, wantBytes) {
						t.Fatalf("profile %d permutation %d union object %s/%s = %q, want oracle bytes", profileIndex, orderIndex, namespace, id, gotBytes)
					}
				}
			}
			rootIDs := map[string]string{}
			for _, namespace := range durableJSONNamespaces {
				root, err := index.Root(namespace)
				if err != nil || root.Count.Uint64() != uint64(len(oracleObjects[namespace])) {
					t.Fatalf("profile %d permutation %d root %s = %#v/%v; oracle count %d", profileIndex, orderIndex, namespace, root, err, len(oracleObjects[namespace]))
				}
				rootIDs[namespace] = root.RootID.String()
			}
			if orderIndex == 0 {
				referenceRootIDs = rootIDs
			} else if !reflect.DeepEqual(rootIDs, referenceRootIDs) {
				t.Fatalf("profile %d permutation %d changed Merkle roots: got %v want %v", profileIndex, orderIndex, rootIDs, referenceRootIDs)
			}
		}
		authoritativeEvent, err := ClassifyJSON(eventA)
		if err != nil {
			t.Fatal(err)
		}
		losingEvent, err := ClassifyJSON(eventB)
		if err != nil {
			t.Fatal(err)
		}
		authoritativeChain, err := repo.ListEvents(testSessionID)
		if err != nil || len(authoritativeChain) != 1 || authoritativeChain[0].EventID != authoritativeEvent.ID.String() || authoritativeChain[0].EventID == losingEvent.ID.String() {
			t.Fatalf("authoritative chain after union permutations = %#v/%v, want only winner-prefix event %s", authoritativeChain, err, authoritativeEvent.ID)
		}
	}
}

func TestProjectionRebuildRefusesAuthorityBytesMissingFromUnion(t *testing.T) {
	t.Run("record", func(t *testing.T) {
		repo, err := sessrepo.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		record := validSessionRecordJSON(t)
		if _, err := repo.CreateSession(record); err != nil {
			t.Fatal(err)
		}
		index, err := New(RPC2Version)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := index.RebuildProjection(repo, testSessionID); err == nil || !strings.Contains(err.Error(), "integrity_failure") {
			t.Fatalf("RebuildProjection(missing union record) = %v, want literal integrity_failure", err)
		}
	})
	t.Run("event", func(t *testing.T) {
		repo, err := sessrepo.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		record := validSessionRecordJSON(t)
		recordMembership, err := ClassifyJSON(record)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.CreateSession(record); err != nil {
			t.Fatal(err)
		}
		event := unionEventJSON(t, recordMembership.ID.String(), "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "session.created", "2026-08-19T04:09:00.000Z", map[string]any{
			"session_record_id": recordMembership.ID.String(), "bootstrap_operation_id": testRequestID,
			"first_checkpoint_operation_id": "0198f4c8-6c30-7d44-8d5e-1234567890ac",
		})
		if _, err := repo.AppendEvent(testSessionID, event); err != nil {
			t.Fatal(err)
		}
		index, err := New(RPC2Version)
		if err != nil {
			t.Fatal(err)
		}
		if err := index.AddUnionJSON(record); err != nil {
			t.Fatal(err)
		}
		if _, err := index.RebuildProjection(repo, testSessionID); err == nil || !strings.Contains(err.Error(), "integrity_failure") {
			t.Fatalf("RebuildProjection(missing union event) = %v, want literal integrity_failure", err)
		}
	})
}

func assertIndependentUnionProjectionOracle(t *testing.T, projection sessstate.Projection) {
	t.Helper()
	if projection.SessionID != "0198f4c8-3e70-7a11-8a2b-1234567890ab" || projection.State != sessstate.State("creating") {
		t.Fatalf("projection session/state = %s/%s, want literal session and creating", projection.SessionID, projection.State)
	}
	if projection.Winner.Epoch != 1 || projection.Winner.LeaseID != "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff" {
		t.Fatalf("projection winner = %+v, want tuple (1, bbbbbbbb-cccc-4ddd-8eee-ffffffffffff) by §5.3 UUID byte order", projection.Winner)
	}
	wantKinds := []string{"losing_branch_preserved", "same_epoch_tie", "union_supersedes_chain"}
	gotKinds := make([]string, 0, len(projection.Conflicts))
	for _, conflict := range projection.Conflicts {
		gotKinds = append(gotKinds, conflict.Kind)
	}
	if !slices.Equal(gotKinds, wantKinds) {
		t.Fatalf("projection conflict kinds = %q, want literal oracle %q", gotKinds, wantKinds)
	}
}

func unionLeaseJSON(t *testing.T, leaseID, holder, createdAt string) []byte {
	t.Helper()
	return identityJSON(t, map[string]any{
		"schema": "urn:ax:schema:lease", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "lease_id": leaseID, "epoch": json.Number("1"),
		"holder_host_id": holder, "predecessor_lease_id": nil, "reason": "create", "checkpoint_id": nil,
		"issued_by_host_id": holder, "created_by_host_id": holder, "created_at": createdAt, "extensions": map[string]any{},
	})
}

func unionEventJSON(t *testing.T, recordID, leaseID, eventType, createdAt string, payload map[string]any) []byte {
	t.Helper()
	return identityJSON(t, map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "event_type": eventType,
		"created_by_host_id": testHostID, "lease_epoch": json.Number("1"), "lease_id": leaseID,
		"lease_sequence": json.Number("1"), "predecessors": []string{recordID}, "created_at": createdAt,
		"payload": payload, "extensions": map[string]any{},
	})
}

func allPermutations(size int) [][]int {
	values := make([]int, size)
	for index := range values {
		values[index] = index
	}
	var output [][]int
	var visit func(int)
	visit = func(position int) {
		if position == len(values) {
			output = append(output, append([]int(nil), values...))
			return
		}
		for index := position; index < len(values); index++ {
			values[position], values[index] = values[index], values[position]
			visit(position + 1)
			values[position], values[index] = values[index], values[position]
		}
	}
	visit(0)
	return output
}

func objectID(t *testing.T, data []byte) string {
	t.Helper()
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	return membership.ID.String()
}

func testRequestIDs() RequestIDFactory {
	var next uint64
	return func() (string, error) {
		next++
		return fmt.Sprintf("0198f4c8-7a10-7b22-8b3c-%012x", next), nil
	}
}

func crashDurableAdd(t *testing.T, root string, data []byte, mode string) int {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestDurableCrashHelperProcess$")
	command.Env = append(os.Environ(),
		"AX_DURABLE_CRASH_ROOT="+root,
		"AX_DURABLE_CRASH_DATA="+base64.StdEncoding.EncodeToString(data),
		"AX_DURABLE_CRASH_MODE="+mode,
	)
	output, err := command.CombinedOutput()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("crash child err = %v, output %s; want explicit process exit", err, output)
	}
	return exitError.ExitCode()
}

func TestDurableCrashHelperProcess(t *testing.T) {
	root := os.Getenv("AX_DURABLE_CRASH_ROOT")
	if root == "" {
		return
	}
	data, err := base64.StdEncoding.DecodeString(os.Getenv("AX_DURABLE_CRASH_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := OpenDurable(root)
	if err != nil {
		t.Fatal(err)
	}
	switch os.Getenv("AX_DURABLE_CRASH_MODE") {
	case "object-install":
		store.ops.afterObjectInstall = func(string) { os.Exit(73) }
	case "quarantine-candidate":
		store.ops.afterQuarantineCandidate = func(string) { os.Exit(73) }
	case "quarantine-pair":
		writes := 0
		store.ops.afterQuarantineCandidate = func(string) {
			writes++
			if writes == 2 {
				os.Exit(73)
			}
		}
	case "quarantine-active-removed":
		store.ops.afterActiveRemoval = func(string) { os.Exit(73) }
	default:
		t.Fatalf("unknown crash mode %q", os.Getenv("AX_DURABLE_CRASH_MODE"))
	}
	if err := store.AddJSON(data); err != nil {
		t.Fatal(err)
	}
	os.Exit(74)
}

func TestDurableAddRejectsBeforePersistingInvalidObjects(t *testing.T) {
	store, err := OpenDurable(filepath.Join(t.TempDir(), "union"))
	if err != nil {
		t.Fatal(err)
	}
	invalid := [][]byte{[]byte(`{"schema":"urn:ax:schema:unknown","schema_version":"1.0.0"}`), []byte(`not-json`)}
	for _, data := range invalid {
		if err := store.AddJSON(data); err == nil {
			t.Fatalf("AddJSON(%q) = nil, want validation refusal", data)
		}
	}
	reopened, err := OpenDurable(store.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, namespace := range durableJSONNamespaces {
		root, err := reopened.Root(namespace)
		if err != nil || root.Count.Uint64() != 0 {
			t.Errorf("%s root after invalid add = %#v/%v, want empty", namespace, root, err)
		}
	}
}

func TestDurableStoreRejectsCorruptActiveBytesOnOpen(t *testing.T) {
	root := filepath.Join(t.TempDir(), "union")
	store, err := OpenDurable(root)
	if err != nil {
		t.Fatal(err)
	}
	data := validSessionRecordJSON(t)
	if err := store.AddJSON(data); err != nil {
		t.Fatal(err)
	}
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	path, err := store.objectPath("record", membership.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"forged":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenDurable(root); !errors.Is(err, ErrDurableCorrupt) {
		t.Fatalf("OpenDurable(corrupt object) = %v, want ErrDurableCorrupt", err)
	}
}

func TestDurableIndexRejectsBlobNamespacePersistence(t *testing.T) {
	store, err := OpenDurable(filepath.Join(t.TempDir(), "union"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Root("blob"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Object("blob", scalar.SHA256Digest([]byte("raw")).String()); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Object(blob) = %v, want §11.5 transfer-boundary refusal", err)
	}
}

func ExampleDurableIndex() {
	fmt.Println("immutable JSON union; no blob staging or materialization")
	// Output: immutable JSON union; no blob staging or materialization
}
