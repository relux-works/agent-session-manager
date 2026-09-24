package merkleinventory

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

func tombstoneJSON(t *testing.T, createdAt string) []byte {
	t.Helper()
	object := map[string]any{
		"schema": "urn:ax:schema:tombstone", "schema_version": "1.0.0", "tombstone_id": zeroDigest,
		"scope": "session", "subject_id": testSessionID, "authorizing_session_id": testSessionID,
		"basis_event_id": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"lease_epoch":    json.Number("1"), "lease_id": "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		"target": map[string]any{
			"kind": "session", "session_id": testSessionID,
			"session_record_id":         "sha256:2222222222222222222222222222222222222222222222222222222222222222",
			"predecessor_checkpoint_id": "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		},
		"created_by_host_id": testHostID, "created_at": createdAt, "extensions": map[string]any{},
	}
	return identityJSON(t, object)
}

func tombstoneAckJSON(t *testing.T, tombstoneID, disposition, conflictCheckpointID, observedAt string) []byte {
	t.Helper()
	var conflict any
	if conflictCheckpointID != "" {
		conflict = conflictCheckpointID
	}
	object := map[string]any{
		"schema": "urn:ax:schema:tombstone-ack", "schema_version": "1.0.0", "ack_id": zeroDigest,
		"subject_id": testSessionID, "tombstone_id": tombstoneID, "acknowledging_host_id": testHostID,
		"disposition": disposition, "conflict_checkpoint_id": conflict,
		"observed_at": observedAt, "created_by_host_id": testHostID,
		"created_at": observedAt, "extensions": map[string]any{},
	}
	return identityJSON(t, object)
}

func identityJSON(t *testing.T, object map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	id, selfField, err := canonicaljson.CalculateObjectIdentity(raw)
	if err != nil {
		t.Fatalf("CalculateObjectIdentity(fixture) error = %v", err)
	}
	object[string(selfField)] = id.String()
	raw, err = json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicaljson.Canonicalize(raw)
	if err != nil {
		t.Fatalf("Canonicalize(fixture) error = %v", err)
	}
	if verified, _, err := canonicaljson.VerifyObjectIdentity(canonical); err != nil || verified != id {
		t.Fatalf("VerifyObjectIdentity(fixture) = %s, %v; want %s", verified, err, id)
	}
	return canonical
}

func TestAddTombstoneAcknowledgementRequiresAStoredMatchingTombstone(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	tombstone := tombstoneJSON(t, "2026-08-19T04:20:00.000Z")
	membership, err := ClassifyJSON(tombstone)
	if err != nil {
		t.Fatal(err)
	}
	acknowledgement := tombstoneAckJSON(t, membership.ID.String(), "applied", "", "2026-08-19T04:21:00.000Z")
	if err := index.AddJSON(acknowledgement); !hasCode(err, "integrity_failure") {
		t.Fatalf("AddJSON(Acknowledgement before Tombstone) = %v, want literal integrity_failure", err)
	}
	if err := index.AddJSON(tombstone); err != nil {
		t.Fatal(err)
	}
	if err := index.AddJSON(tombstoneAckJSON(t, membership.ID.String(), "applied", "", "2026-08-19T04:21:00.000Z")); err != nil {
		t.Fatalf("AddJSON(Acknowledgement with stored Tombstone) = %v", err)
	}

	wrongSubject := tombstoneAckJSON(t, membership.ID.String(), "not_target", "", "2026-08-19T04:22:00.000Z")
	var object map[string]any
	if err := json.Unmarshal(wrongSubject, &object); err != nil {
		t.Fatal(err)
	}
	object["subject_id"] = testGroupID
	wrongSubject = identityJSON(t, object)
	if err := index.AddJSON(wrongSubject); !hasCode(err, "integrity_failure") {
		t.Fatalf("AddJSON(Acknowledgement with mismatched subject) = %v, want literal integrity_failure", err)
	}
}

func TestInProcessTombstoneUnionRetainsBothTimesAndAcknowledgements(t *testing.T) {
	local, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	localTombstone := tombstoneJSON(t, "2026-08-19T04:20:00.000Z")
	peerTombstone := tombstoneJSON(t, "2026-08-20T04:20:00.000Z")
	localTombstoneMembership, err := ClassifyJSON(localTombstone)
	if err != nil {
		t.Fatal(err)
	}
	peerTombstoneMembership, err := ClassifyJSON(peerTombstone)
	if err != nil {
		t.Fatal(err)
	}
	localAck := tombstoneAckJSON(t, localTombstoneMembership.ID.String(), "applied", "", "2026-08-19T04:21:00.000Z")
	peerAck := tombstoneAckJSON(t, peerTombstoneMembership.ID.String(), "retained_conflict",
		"sha256:4444444444444444444444444444444444444444444444444444444444444444", "2026-08-20T04:21:00.000Z")
	for _, object := range [][]byte{localTombstone, localAck} {
		if err := local.AddJSON(object); err != nil {
			t.Fatal(err)
		}
	}
	for _, object := range [][]byte{peerTombstone, peerAck} {
		if err := peer.AddJSON(object); err != nil {
			t.Fatal(err)
		}
	}

	missingTombstones, err := MissingObjectIDs(local, peer, "tombstone")
	if err != nil || !slices.Equal(missingTombstones, []string{peerTombstoneMembership.ID.String()}) {
		t.Fatalf("Tombstone recursive walk = %#v, %v", missingTombstones, err)
	}
	requestIDs := []string{testRequestID, "0198f4c8-7a10-7b22-8b3c-2234567890ac", "0198f4c8-7a10-7b22-8b3c-2234567890ad", "0198f4c8-7a10-7b22-8b3c-2234567890ae"}
	next := 0
	newID := func() (string, error) {
		id := requestIDs[next]
		next++
		return id, nil
	}
	objects, err := FetchObjects(peer, "tombstone", missingTombstones, rpcwire.MaxLineBytes, newID)
	if err != nil || len(objects) != 1 {
		t.Fatalf("fetch missing Tombstone = %#v, %v", objects, err)
	}
	if err := local.AddJSON(decodeWireData(t, objects[0])); err != nil {
		t.Fatalf("union fetched Tombstone: %v", err)
	}
	missingAcks, err := MissingObjectIDs(local, peer, "tombstone_ack")
	peerAckMembership, errClassify := ClassifyJSON(peerAck)
	if err != nil || errClassify != nil || !slices.Equal(missingAcks, []string{peerAckMembership.ID.String()}) {
		t.Fatalf("Tombstone Acknowledgement recursive walk = %#v, %v; classify = %#v, %v", missingAcks, err, peerAckMembership, errClassify)
	}
	objects, err = FetchObjects(peer, "tombstone_ack", missingAcks, rpcwire.MaxLineBytes, newID)
	if err != nil || len(objects) != 1 {
		t.Fatalf("fetch missing Acknowledgement = %#v, %v", objects, err)
	}
	if err := local.AddJSON(decodeWireData(t, objects[0])); err != nil {
		t.Fatalf("union fetched Acknowledgement: %v", err)
	}
	missingTombstones, err = MissingObjectIDs(peer, local, "tombstone")
	if err != nil || !slices.Equal(missingTombstones, []string{localTombstoneMembership.ID.String()}) {
		t.Fatalf("reverse Tombstone recursive walk = %#v, %v", missingTombstones, err)
	}
	objects, err = FetchObjects(local, "tombstone", missingTombstones, rpcwire.MaxLineBytes, newID)
	if err != nil || len(objects) != 1 {
		t.Fatalf("fetch reverse Tombstone = %#v, %v", objects, err)
	}
	if err := peer.AddJSON(decodeWireData(t, objects[0])); err != nil {
		t.Fatalf("reverse union fetched Tombstone: %v", err)
	}
	missingAcks, err = MissingObjectIDs(peer, local, "tombstone_ack")
	if err != nil {
		t.Fatalf("reverse Tombstone Acknowledgement recursive walk: %v", err)
	}
	objects, err = FetchObjects(local, "tombstone_ack", missingAcks, rpcwire.MaxLineBytes, newID)
	if err != nil || len(objects) != 1 {
		t.Fatalf("fetch reverse Acknowledgement = %#v, %v", objects, err)
	}
	if err := peer.AddJSON(decodeWireData(t, objects[0])); err != nil {
		t.Fatalf("reverse union fetched Acknowledgement: %v", err)
	}
	for _, namespace := range []string{"tombstone", "tombstone_ack"} {
		localRoot, err := local.Root(namespace)
		peerRoot, peerErr := peer.Root(namespace)
		if err != nil || peerErr != nil || localRoot != peerRoot || localRoot.Count.Uint64() != 2 {
			t.Fatalf("%s union root local=%#v peer=%#v errors=%v/%v; want both immutable records retained", namespace, localRoot, peerRoot, err, peerErr)
		}
	}
	for _, object := range [][]byte{localTombstone, peerTombstone, localAck, peerAck} {
		membership, err := ClassifyJSON(object)
		if err != nil {
			t.Fatal(err)
		}
		namespace, ok := local.MembershipOf(membership.ID)
		if !ok || namespace != membership.Namespace {
			t.Errorf("unioned immutable ID %s has membership %q, %v; want %q", membership.ID, namespace, ok, membership.Namespace)
		}
	}
}
