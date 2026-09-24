package merkleinventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestNormativeRootAndChildFixtures(t *testing.T) {
	zero := "sha256:" + strings.Repeat("0", 64)
	one := "sha256:" + strings.Repeat("f", 64)
	fixtures := []struct {
		name string
		ids  []string
		want string
	}{
		{"empty", nil, "sha256:6ed4e56353243641ade20bd0f9e9ae426dcc2d02c962c0a2e7208baae2eef1e8"},
		{"singleton", []string{zero}, "sha256:ab4c73cb6d0612884e1a3fec5f4e93d3a7e47f4f96899681216524d3abfaab07"},
		{"branch", []string{zero, one}, "sha256:34bdaac6a61aa6d54cfd1315fa6325489e34b83206b891e765e02850a5f14c26"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			index, err := New(RPC2Version)
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range fixture.ids {
				seedNormativeSyntheticID(index, "record", mustDigest(t, id))
			}
			node, err := index.Child("record", "")
			if err != nil {
				t.Fatal(err)
			}
			if got := node.NodeHash.String(); got != fixture.want {
				t.Fatalf("node hash = %q, want literal %q", got, fixture.want)
			}
			body, err := json.Marshal(node)
			if err != nil {
				t.Fatal(err)
			}
			canonical, err := canonicaljson.Canonicalize(body)
			if err != nil {
				t.Fatal(err)
			}
			wantBody := map[string]string{
				"empty":     `{"children":[],"count":0,"ids":[],"namespace":"record","node_hash":"sha256:6ed4e56353243641ade20bd0f9e9ae426dcc2d02c962c0a2e7208baae2eef1e8","prefix":""}`,
				"singleton": `{"children":[],"count":1,"ids":["sha256:` + strings.Repeat("0", 64) + `"],"namespace":"record","node_hash":"sha256:ab4c73cb6d0612884e1a3fec5f4e93d3a7e47f4f96899681216524d3abfaab07","prefix":""}`,
				"branch":    `{"children":[{"count":1,"hash":"sha256:f047338baa8ba0e4050630c4bae2b18ae1e79366ed5d5b4b140952eebf74e6fc","label":"0"},{"count":1,"hash":"sha256:feb44d42c24a06a3ae665f317840b771cbd6d575dd0b8110f0c083a7fc0c504d","label":"f"}],"count":2,"ids":[],"namespace":"record","node_hash":"sha256:34bdaac6a61aa6d54cfd1315fa6325489e34b83206b891e765e02850a5f14c26","prefix":""}`,
			}[fixture.name]
			if string(canonical) != wantBody {
				t.Fatalf("canonical body = %s, want literal %s", canonical, wantBody)
			}
		})
	}
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	seedNormativeSyntheticID(index, "record", mustDigest(t, zero))
	seedNormativeSyntheticID(index, "record", mustDigest(t, one))
	branch, err := index.Child("record", "")
	if err != nil {
		t.Fatal(err)
	}
	wantChildHashes := []string{
		"sha256:f047338baa8ba0e4050630c4bae2b18ae1e79366ed5d5b4b140952eebf74e6fc",
		"sha256:feb44d42c24a06a3ae665f317840b771cbd6d575dd0b8110f0c083a7fc0c504d",
	}
	for index, child := range branch.Children {
		if child.Hash.String() != wantChildHashes[index] {
			t.Fatalf("child %q hash = %q, want literal %q", child.Label, child.Hash, wantChildHashes[index])
		}
	}
}

func TestNodeRuleShapesAndDepth64IntegrityFailure(t *testing.T) {
	zero := "sha256:" + strings.Repeat("0", 64)
	one := "sha256:" + strings.Repeat("0", 63) + "1"
	branch, err := ComputeNode("record", "", []string{zero, one, zero})
	if err != nil {
		t.Fatal(err)
	}
	if branch.Count.Uint64() != 2 || len(branch.IDs) != 0 || len(branch.Children) != 1 || branch.Children[0].Label != "0" {
		t.Fatalf("rule 1/4 did not produce a unique single-child branch: %#v", branch)
	}
	singleton, err := ComputeNode("record", "", []string{zero})
	if err != nil || singleton.Count.Uint64() != 1 || len(singleton.Children) != 0 || len(singleton.IDs) != 1 {
		t.Fatalf("rule 3 singleton shape = %#v, %v", singleton, err)
	}
	empty, err := ComputeNode("record", "", nil)
	if err != nil || empty.Count.Uint64() != 0 || len(empty.Children) != 0 || len(empty.IDs) != 0 {
		t.Fatalf("rule 2 empty-root shape = %#v, %v", empty, err)
	}
	_, err = ComputeNode("record", strings.Repeat("0", 64), []string{zero, zero})
	if !hasCode(err, "integrity_failure") {
		t.Fatalf("rule 5 depth-64 duplicate error = %v, want literal integrity_failure", err)
	}
}

func TestRuleFourRetainsSingleChildAtEverySharedPrefix(t *testing.T) {
	ids := []string{
		"sha256:" + strings.Repeat("0", 64),
		"sha256:" + strings.Repeat("0", 63) + "1",
	}
	for length := 0; length < 63; length++ {
		node, err := ComputeNode("record", strings.Repeat("0", length), ids)
		if err != nil {
			t.Fatalf("prefix length %d: %v", length, err)
		}
		if node.Count.Uint64() != 2 || len(node.IDs) != 0 || len(node.Children) != 1 || node.Children[0].Label != "0" {
			t.Fatalf("rule 4 at prefix length %d = %#v, want one retained child", length, node)
		}
	}
	branch, err := ComputeNode("record", strings.Repeat("0", 63), ids)
	if err != nil {
		t.Fatal(err)
	}
	if branch.Count.Uint64() != 2 || len(branch.IDs) != 0 || len(branch.Children) != 2 || branch.Children[0].Label != "0" || branch.Children[1].Label != "1" {
		t.Fatalf("rule 4 terminal branch = %#v, want sorted [0, 1] children", branch)
	}
}

func TestMixedNS1RootsFromNormativeSyntheticIDs(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	z := func(n byte) scalar.Digest {
		d, _ := scalar.ParseDigest("sha256:" + strings.Repeat("0", 63) + string(n))
		return d
	}
	two := func(n byte) scalar.Digest {
		d, _ := scalar.ParseDigest("sha256:" + strings.Repeat("2", 63) + string(n))
		return d
	}
	fixtures := []struct {
		namespace string
		ids       []scalar.Digest
		count     uint64
		root      string
	}{
		{"record", []scalar.Digest{z('1'), z('2'), z('3'), z('4'), z('5')}, 5, "sha256:44c71ab5fdb7403c57a8a929d9fc90b2db3e8829b2615901c0841216d6750580"},
		{"event", []scalar.Digest{digestOfRepeat('1')}, 1, "sha256:533934b217b3d2f3999a2ec8cbb5ccfc0a7aff29c4eee0f7d35a0adca8cef15e"},
		{"manifest", []scalar.Digest{two('1'), two('2'), two('3'), two('4')}, 4, "sha256:cb60d01f92e1dae564a49f2ae782b9cf479a9e3849d1c84a8c907b68dec85202"},
		{"tombstone", []scalar.Digest{digestOfRepeat('3')}, 1, "sha256:b890fa63b85a48c6e6a5e42cece3fb50e545062da103bb1f5bcfaa80d922f0d9"},
		{"tombstone_ack", []scalar.Digest{digestOfRepeat('4')}, 1, "sha256:4f5c58b2cee87816f09520570056ebe64c408dc484ff192aaf8583a308eda8e5"},
		{"blob", []scalar.Digest{digestOfRepeat('5')}, 1, "sha256:eaff5e30efa41b95ad419948886433af94ab3c95de2a574a1c1cfcf6d45a7cd2"},
	}
	for _, fixture := range fixtures {
		for _, id := range fixture.ids {
			seedNormativeSyntheticID(index, fixture.namespace, id)
		}
		root, err := index.Root(fixture.namespace)
		if err != nil {
			t.Fatal(err)
		}
		if root.Count.Uint64() != fixture.count || root.RootID.String() != fixture.root {
			t.Errorf("MIXED-NS-1 %s count/root = %d/%q, want literal %d/%q", fixture.namespace, root.Count.Uint64(), root.RootID, fixture.count, fixture.root)
		}
	}
}

func TestDispatchServesNormativeChildrenBody(t *testing.T) {
	zero := "sha256:" + strings.Repeat("0", 64)
	f := "sha256:" + strings.Repeat("f", 64)
	fixtures := []struct {
		name string
		ids  []string
		want string
	}{
		{"empty", nil, `{"children":[],"count":0,"ids":[],"namespace":"record","node_hash":"sha256:6ed4e56353243641ade20bd0f9e9ae426dcc2d02c962c0a2e7208baae2eef1e8","prefix":""}`},
		{"singleton", []string{zero}, `{"children":[],"count":1,"ids":["` + zero + `"],"namespace":"record","node_hash":"sha256:ab4c73cb6d0612884e1a3fec5f4e93d3a7e47f4f96899681216524d3abfaab07","prefix":""}`},
		{"branch", []string{zero, f}, `{"children":[{"count":1,"hash":"sha256:f047338baa8ba0e4050630c4bae2b18ae1e79366ed5d5b4b140952eebf74e6fc","label":"0"},{"count":1,"hash":"sha256:feb44d42c24a06a3ae665f317840b771cbd6d575dd0b8110f0c083a7fc0c504d","label":"f"}],"count":2,"ids":[],"namespace":"record","node_hash":"sha256:34bdaac6a61aa6d54cfd1315fa6325489e34b83206b891e765e02850a5f14c26","prefix":""}`},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			index, err := New(RPC2Version)
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range fixture.ids {
				seedNormativeSyntheticID(index, "record", mustDigest(t, id))
			}
			requestBody := json.RawMessage(`{"namespace":"record","prefix":""}`)
			line, err := rpcwire.EncodeRequest(RPC2Version, testRequestID, "inventory.children", requestBody)
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
				t.Fatalf("response decode = %#v, %v", response, err)
			}
			canonical, err := canonicaljson.Canonicalize(response.Body())
			if err != nil {
				t.Fatal(err)
			}
			if string(canonical) != fixture.want {
				t.Fatalf("inventory.children body = %s, want exact literal %s", canonical, fixture.want)
			}
		})
	}
}

func TestDispatchServesAllSixInventoryRootsInRPCWireOrder(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
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
		t.Fatalf("inventory.roots decode = %#v, %v", response, err)
	}
	canonical, err := canonicaljson.Canonicalize(response.Body())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"roots":[{"count":0,"namespace":"blob","root_id":"sha256:dca44fdfc5688a1d1f14863103b9005060b8d133ed8979bd0859ffca66fbf1da"},{"count":0,"namespace":"event","root_id":"sha256:928ecd990c39d77bcd631915f9accc538336a3df722b4414c360a2321ad34421"},{"count":0,"namespace":"manifest","root_id":"sha256:267753fb6df4d845422f455a0ea3d875c36d27d502319889ae179e0a1c1a15ad"},{"count":0,"namespace":"record","root_id":"sha256:6ed4e56353243641ade20bd0f9e9ae426dcc2d02c962c0a2e7208baae2eef1e8"},{"count":0,"namespace":"tombstone","root_id":"sha256:9ae7db9e1c000fb88436c1aba73ede34550e61d00ea59fb5e6a611df6ea91566"},{"count":0,"namespace":"tombstone_ack","root_id":"sha256:6581a91cc788fa284d340a64b6aad85ebbee06e5dd02a106b8fd924e2573fffa"}]}`
	if string(canonical) != want {
		t.Fatalf("inventory.roots body = %s, want literal %s", canonical, want)
	}
}

func TestDispatchBuildsSixteenSortedNibbleChildren(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range "0123456789abcdef" {
		id := "sha256:" + string(label) + strings.Repeat("0", 63)
		seedNormativeSyntheticID(index, "record", mustDigest(t, id))
	}
	line, err := rpcwire.EncodeRequest(RPC2Version, testRequestID, "inventory.children", json.RawMessage(`{"namespace":"record","prefix":""}`))
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
		t.Fatalf("sixteen-child response = %#v, %v", response, err)
	}
	node, err := decodeInventoryNode(response.Body())
	if err != nil {
		t.Fatal(err)
	}
	if node.Count.Uint64() != 16 || len(node.Children) != 16 || len(node.IDs) != 0 {
		t.Fatalf("sixteen-child node count/children/ids = %d/%d/%d", node.Count.Uint64(), len(node.Children), len(node.IDs))
	}
	for index, label := range "0123456789abcdef" {
		if node.Children[index].Label != string(label) || node.Children[index].Count.Uint64() != 1 {
			t.Errorf("children[%d] = %#v, want sorted %q child of count 1", index, node.Children[index], label)
		}
	}
}

func decodeInventoryNode(data []byte) (Node, error) {
	var node Node
	err := json.Unmarshal(data, &node)
	return node, err
}

func TestDispatchPrefixAxesAndNotFound(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	zero := mustDigest(t, "sha256:"+strings.Repeat("0", 64))
	seedNormativeSyntheticID(index, "record", zero)
	tests := []struct {
		name   string
		prefix string
		code   string
	}{
		{"root-length-0", "", ""},
		{"materialized-length-1", "0", ""},
		{"materialized-length-63", strings.Repeat("0", 63), ""},
		{"materialized-length-64", strings.Repeat("0", 64), ""},
		{"valid-missing-prefix", "f", "not_found"},
		{"length-65", strings.Repeat("0", 65), "invalid"},
		{"nonhex", "g", "invalid"},
		{"uppercase", "A", "invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := json.RawMessage(fmt.Sprintf(`{"namespace":"record","prefix":%q}`, test.prefix))
			line, err := rpcwire.EncodeRequest(RPC2Version, testRequestID, "inventory.children", body)
			if err != nil {
				t.Fatal(err)
			}
			_, err = index.Dispatch(line)
			if test.code == "" && err != nil {
				t.Fatalf("Dispatch() = %v, want success", err)
			}
			if test.code == "not_found" && !hasCode(err, "not_found") {
				t.Fatalf("Dispatch() = %v, want literal not_found", err)
			}
			if test.code == "invalid" && !errorsIsInvalid(err) {
				t.Fatalf("Dispatch() = %v, want invalid request refusal", err)
			}
		})
	}
}

const testRequestID = "0198f4c8-7a10-7b22-8b3c-2234567890ab"

func digestOfRepeat(c byte) scalar.Digest {
	return mustDigest(nil, "sha256:"+strings.Repeat(string(c), 64))
}

func seedNormativeSyntheticID(index *Index, namespace string, id scalar.Digest) {
	index.mu.Lock()
	defer index.mu.Unlock()
	index.objects[namespace][id.String()] = storedObject{}
	index.locations[id.String()] = namespace
}

func mustDigest(t *testing.T, raw string) scalar.Digest {
	digest, err := scalar.ParseDigest(raw)
	if err != nil {
		if t != nil {
			t.Fatal(err)
		}
		panic(err)
	}
	return digest
}

func hasCode(err error, code string) bool {
	var refusal *Refusal
	return errors.As(err, &refusal) && refusal.Code == code
}

func errorsIsInvalid(err error) bool {
	return errors.Is(err, ErrInvalidRequest)
}
