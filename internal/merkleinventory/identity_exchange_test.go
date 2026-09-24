package merkleinventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestMixedNSExchangeIdentityLevelSyntheticIDs(t *testing.T) {
	// Section 11.4 says these synthetic IDs stand for schema-valid objects/bytes
	// and their contents are not used to compute the inventory trie. This row
	// proves identity-level roots and walking only; validated byte union is
	// exercised separately by TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects.
	namespaces := []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"}
	z := func(n byte) string { return "sha256:" + strings.Repeat("0", 63) + string(n) }
	two := func(n byte) string { return "sha256:" + strings.Repeat("2", 63) + string(n) }
	all := map[string][]string{
		"record":        {z('1'), z('2'), z('3'), z('4'), z('5')},
		"event":         {"sha256:" + strings.Repeat("1", 64)},
		"manifest":      {two('1'), two('2'), two('3'), two('4')},
		"tombstone":     {"sha256:" + strings.Repeat("3", 64)},
		"tombstone_ack": {"sha256:" + strings.Repeat("4", 64)},
		"blob":          {"sha256:" + strings.Repeat("5", 64)},
	}
	wantRoots := map[string]struct {
		count uint64
		root  string
	}{
		"record":        {5, "sha256:44c71ab5fdb7403c57a8a929d9fc90b2db3e8829b2615901c0841216d6750580"},
		"event":         {1, "sha256:533934b217b3d2f3999a2ec8cbb5ccfc0a7aff29c4eee0f7d35a0adca8cef15e"},
		"manifest":      {4, "sha256:cb60d01f92e1dae564a49f2ae782b9cf479a9e3849d1c84a8c907b68dec85202"},
		"tombstone":     {1, "sha256:b890fa63b85a48c6e6a5e42cece3fb50e545062da103bb1f5bcfaa80d922f0d9"},
		"tombstone_ack": {1, "sha256:4f5c58b2cee87816f09520570056ebe64c408dc484ff192aaf8583a308eda8e5"},
		"blob":          {1, "sha256:eaff5e30efa41b95ad419948886433af94ab3c95de2a574a1c1cfcf6d45a7cd2"},
	}
	missing := map[string][]string{
		"record":        {z('3')},
		"event":         nil,
		"manifest":      {two('2')},
		"tombstone":     nil,
		"tombstone_ack": nil,
		"blob":          {"sha256:" + strings.Repeat("5", 64)},
	}

	peerA, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	peerB, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, namespace := range namespaces {
		for _, rawID := range all[namespace] {
			id := mustDigest(t, rawID)
			seedNormativeSyntheticID(peerA, namespace, id)
			if !slices.Contains(missing[namespace], rawID) {
				seedNormativeSyntheticID(peerB, namespace, id)
			}
		}
	}

	rootsA := dispatchInventoryRoots(t, peerA)
	rootsB := dispatchInventoryRoots(t, peerB)
	changed := make([]string, 0, 3)
	for i, namespace := range namespaces {
		if rootsA[i].Namespace != namespace || rootsB[i].Namespace != namespace {
			t.Fatalf("inventory.roots order at %d = %q/%q, want literal %q", i, rootsA[i].Namespace, rootsB[i].Namespace, namespace)
		}
		want := wantRoots[namespace]
		if rootsA[i].Count.Uint64() != want.count || rootsA[i].RootID.String() != want.root {
			t.Errorf("MIXED-NS-1 inventory.roots %s = %d/%s, want literal %d/%s", namespace, rootsA[i].Count.Uint64(), rootsA[i].RootID, want.count, want.root)
		}
		if rootsA[i].RootID != rootsB[i].RootID {
			changed = append(changed, namespace)
		}
	}
	if !slices.Equal(changed, []string{"blob", "manifest", "record"}) {
		t.Fatalf("MIXED-NS-EXCHANGE changed roots = %#v, want literal [blob manifest record]", changed)
	}

	gotMissing := make(map[string][]string, len(namespaces))
	for _, namespace := range namespaces {
		ids, walkErr := MissingNamespaceIDs(peerB, peerA, namespace)
		if walkErr != nil {
			t.Fatalf("MissingNamespaceIDs(%s) = %v", namespace, walkErr)
		}
		gotMissing[namespace] = ids
		if !slices.Equal(ids, missing[namespace]) {
			t.Errorf("recursive %s inventory.children walk = %#v, want exact fixture identities %#v", namespace, ids, missing[namespace])
		}
		for _, id := range missing[namespace] {
			if !dispatchChildPathContainsID(t, peerA, namespace, id) {
				t.Errorf("peer A inventory.children walk did not reach %s in %s", id, namespace)
			}
			if dispatchChildPathContainsID(t, peerB, namespace, id) {
				t.Errorf("peer B inventory.children walk unexpectedly found %s in %s", id, namespace)
			}
		}
	}
	wantMissing := map[string][]string{
		"blob":     {"sha256:" + strings.Repeat("5", 64)},
		"manifest": {two('2')},
		"record":   {z('3')},
	}
	gotMissingSet := make(map[string][]string)
	for _, namespace := range []string{"blob", "manifest", "record"} {
		gotMissingSet[namespace] = gotMissing[namespace]
	}
	if len(gotMissingSet) != 3 || !slices.Equal(gotMissingSet["blob"], wantMissing["blob"]) ||
		!slices.Equal(gotMissingSet["manifest"], wantMissing["manifest"]) ||
		!slices.Equal(gotMissingSet["record"], wantMissing["record"]) {
		t.Fatalf("MIXED-NS-EXCHANGE recursive missing set = %#v, want exactly %#v", gotMissingSet, wantMissing)
	}
	if _, err := MissingObjectIDs(peerB, peerA, "blob"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("MissingObjectIDs(blob) = %v, want refusal at the Section 11.5 byte-transfer boundary", err)
	}

	for _, namespace := range namespaces {
		for _, rawID := range missing[namespace] {
			seedNormativeSyntheticID(peerB, namespace, mustDigest(t, rawID))
		}
	}
	rootsB = dispatchInventoryRoots(t, peerB)
	for i, namespace := range namespaces {
		want := wantRoots[namespace]
		if rootsB[i].Namespace != namespace || rootsB[i] != rootsA[i] || rootsB[i].Count.Uint64() != want.count || rootsB[i].RootID.String() != want.root {
			t.Errorf("post-exchange %s inventory.roots = %#v, want peer A and MIXED-NS-1 literal %d/%s", namespace, rootsB[i], want.count, want.root)
		}
	}
}

func TestUnsupportedSessionRecord31RemainsFailClosed(t *testing.T) {
	assertUnsupportedSchemaDoesNotEnterInventory(t, "urn:ax:schema:session-record", "3.1.0", "record_id")
}

func TestUnsupportedMaterializationPlan10RemainsFailClosed(t *testing.T) {
	assertUnsupportedSchemaDoesNotEnterInventory(t, "urn:ax:schema:materialization-plan", "1.0.0", "plan_id")
}

func TestUnsupportedMaterializationPlan20RemainsFailClosed(t *testing.T) {
	assertUnsupportedSchemaDoesNotEnterInventory(t, "urn:ax:schema:materialization-plan", "2.0.0", "plan_id")
}

func TestUnsupportedTaskBoardBundle10RemainsFailClosed(t *testing.T) {
	assertUnsupportedSchemaDoesNotEnterInventory(t, "urn:ax:schema:task-board-bundle", "1.0.0", "bundle_id")
}

func assertUnsupportedSchemaDoesNotEnterInventory(t *testing.T, schema, version, selfField string) {
	t.Helper()
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	before := make([]Root, 0, 6)
	for _, namespace := range []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"} {
		root, rootErr := index.Root(namespace)
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		before = append(before, root)
	}
	withoutSelf, err := canonicaljson.Canonicalize([]byte(fmt.Sprintf(`{"schema":%q,"schema_version":%q}`, schema, version)))
	if err != nil {
		t.Fatal(err)
	}
	claimedID := scalar.SHA256Digest(withoutSelf).String()
	data := []byte(fmt.Sprintf(`{"schema":%q,"schema_version":%q,%q:%q}`, schema, version, selfField, claimedID))
	err = index.AddJSON(data)
	var refusalErr *Refusal
	if !errors.As(err, &refusalErr) || refusalErr.Code != "integrity_failure" {
		t.Fatalf("AddJSON(%s@%s) = %v, want typed literal integrity_failure", schema, version, err)
	}
	after := make([]Root, 0, 6)
	for _, namespace := range []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"} {
		root, rootErr := index.Root(namespace)
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		after = append(after, root)
	}
	for i := range before {
		if before[i] != after[i] || after[i].Count.Uint64() != 0 {
			t.Errorf("refused %s@%s moved %s root/count: before=%#v after=%#v", schema, version, before[i].Namespace, before[i], after[i])
		}
	}
	if ids := index.QuarantinedIDs(); len(ids) != 0 {
		t.Fatalf("refused %s@%s was quarantined as a byte conflict: %q", schema, version, ids)
	}
}

func dispatchInventoryRoots(t *testing.T, index *Index) []Root {
	t.Helper()
	line, err := rpcwire.EncodeRequest(RPC2Version, testRequestID, "inventory.roots", json.RawMessage(`{"namespaces":["blob","event","manifest","record","tombstone","tombstone_ack"]}`))
	if err != nil {
		t.Fatal(err)
	}
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		t.Fatal(err)
	}
	responseLine, err := index.Dispatch(line)
	if err != nil {
		t.Fatal(err)
	}
	response, err := rpcwire.DecodeResponse(responseLine, request)
	if err != nil || !response.OK() {
		t.Fatalf("inventory.roots response = %#v, %v", response, err)
	}
	var body struct {
		Roots []Root `json:"roots"`
	}
	if err := json.Unmarshal(response.Body(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Roots) != 6 {
		t.Fatalf("inventory.roots returned %d roots, want literal 6", len(body.Roots))
	}
	return body.Roots
}

func dispatchChildPathContainsID(t *testing.T, index *Index, namespace, id string) bool {
	t.Helper()
	hex := strings.TrimPrefix(id, "sha256:")
	prefix := ""
	for len(prefix) <= 64 {
		line, err := rpcwire.EncodeRequest(RPC2Version, testRequestID, "inventory.children", json.RawMessage(fmt.Sprintf(`{"namespace":%q,"prefix":%q}`, namespace, prefix)))
		if err != nil {
			t.Fatal(err)
		}
		responseLine, err := index.Dispatch(line)
		if err != nil {
			if hasCode(err, "not_found") {
				return false
			}
			t.Fatalf("inventory.children(%s,%s) = %v", namespace, prefix, err)
		}
		var node Node
		if err := json.Unmarshal(responseLineBody(t, responseLine, line), &node); err != nil {
			t.Fatal(err)
		}
		if len(node.IDs) > 0 {
			return len(node.IDs) == 1 && node.IDs[0] == id
		}
		if len(prefix) == 64 {
			return false
		}
		prefix += string(hex[len(prefix)])
	}
	return false
}

func responseLineBody(t *testing.T, responseLine, requestLine []byte) []byte {
	t.Helper()
	request, err := rpcwire.DecodeRequest(requestLine)
	if err != nil {
		t.Fatal(err)
	}
	response, err := rpcwire.DecodeResponse(responseLine, request)
	if err != nil || !response.OK() {
		t.Fatalf("inventory.children response = %#v, %v", response, err)
	}
	return response.Body()
}
