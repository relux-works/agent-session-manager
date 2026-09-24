package merkleinventory

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

func TestFetchObjectsRefusesDuplicateIDs(t *testing.T) {
	peer, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	data := validSessionRecordJSON(t)
	if err := peer.AddJSON(data); err != nil {
		t.Fatal(err)
	}
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	source := &countingObjectSource{source: peer}
	objects, err := FetchObjects(source, "record", []string{membership.ID.String(), membership.ID.String()}, rpcwire.MaxLineBytes, func() (string, error) {
		return testRequestID, nil
	})
	if len(objects) != 0 || !errors.Is(err, ErrInvalidRequest) || err.Error() != ErrInvalidRequest.Error() {
		t.Fatalf("FetchObjects(duplicate IDs) = (%d objects, %v), want literal %q", len(objects), err, ErrInvalidRequest.Error())
	}
	if source.calls != 0 {
		t.Fatalf("duplicate IDs reached ObjectsGet %d times; client must refuse before the request", source.calls)
	}
}

func TestFetchObjectsRefusesNewlineBearingBase64urlData(t *testing.T) {
	data := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	cut := len(encoded) / 2
	encoded = encoded[:cut] + "\r\n" + encoded[cut:]
	source := &hostileObjectSource{object: WireObject{
		ObjectID: membership.ID.String(), MediaType: "application/json", Encoding: "json", Data: encoded,
	}}
	objects, err := FetchObjects(source, "record", []string{membership.ID.String()}, rpcwire.MaxLineBytes, func() (string, error) {
		return testRequestID, nil
	})
	if len(objects) != 0 || !errors.Is(err, ErrInvalidObject) || err.Error() != ErrInvalidObject.Error() {
		t.Fatalf("FetchObjects(newline-bearing base64url) = (%d objects, %v), want literal %q", len(objects), err, ErrInvalidObject.Error())
	}
}

func TestMissingObjectIDsRefusesLocalQuarantinedIdentity(t *testing.T) {
	data := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	local, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := local.AddJSON(data); err != nil {
		t.Fatal(err)
	}
	// Whitespace is outside the canonical identity, so it produces a distinct
	// byte representation with the same validated immutable ID.
	variant := []byte("\n" + string(data))
	if err := local.AddJSON(variant); !hasCode(err, "integrity_failure") {
		t.Fatalf("AddJSON(same identity, different bytes) = %v, want literal integrity_failure", err)
	}
	peer, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(data); err != nil {
		t.Fatal(err)
	}
	missing, err := MissingObjectIDs(local, peer, "record")
	if len(missing) != 0 || !hasCode(err, "integrity_failure") || !strings.Contains(err.Error(), "integrity_failure") {
		t.Fatalf("MissingObjectIDs(quarantined local identity %s) = (%v, %v), want literal integrity_failure", membership.ID, missing, err)
	}
}

func TestMissingObjectIDsRefusesCrossNamespaceIdentity(t *testing.T) {
	data := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	local, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(data); err != nil {
		t.Fatal(err)
	}
	// This state cannot be produced by AddJSON: ClassifyJSON assigns this
	// record only to `record`, and locations is global to all namespaces.
	// Seed the malformed internal state to pin the recursive walker defense.
	local.mu.Lock()
	local.objects["manifest"][membership.ID.String()] = storedObject{data: append([]byte(nil), data...), json: true}
	local.locations[membership.ID.String()] = "manifest"
	local.mu.Unlock()

	missing, err := MissingObjectIDs(local, peer, "record")
	if len(missing) != 0 || !hasCode(err, "integrity_failure") || !strings.Contains(err.Error(), "integrity_failure") {
		t.Fatalf("MissingObjectIDs(cross-namespace %s) = (%v, %v), want literal integrity_failure", membership.ID, missing, err)
	}
}

type countingObjectSource struct {
	source ObjectSource
	calls  int
}

func (source *countingObjectSource) ObjectsGet(line []byte, namespace string, maxLineBytes uint64) ([]byte, error) {
	source.calls++
	return source.source.ObjectsGet(line, namespace, maxLineBytes)
}
