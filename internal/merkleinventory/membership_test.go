package merkleinventory

import (
	"encoding/base64"
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

const (
	testSessionID = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	testHostID    = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	testGroupID   = "0198f4c8-5b20-7c33-8c4d-1234567890ab"
	testWorkspace = "0198f4c8-6c30-7d44-8d5e-1234567890ab"
	zeroDigest    = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
)

func TestClassifyAndAddJSONUsesValidatedSchemaMembership(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	record := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(record)
	if err != nil {
		t.Fatal(err)
	}
	if membership.Namespace != "record" || membership.Excluded || membership.ID.String() == "" {
		t.Fatalf("session record membership = %#v", membership)
	}
	if err := index.AddJSON(record); err != nil {
		t.Fatal(err)
	}
	if err := index.AddJSON(record); err != nil {
		t.Fatalf("identical immutable object should be idempotent: %v", err)
	}
	root, err := index.Root("record")
	if err != nil || root.Count.Uint64() != 1 || root.RootID.String() == "" {
		t.Fatalf("record root = %#v, %v", root, err)
	}
	if namespace, ok := index.MembershipOf(membership.ID); !ok || namespace != "record" {
		t.Fatalf("stored membership = %q, %v; want record", namespace, ok)
	}
}

func TestClassifyRefusesIdentityDigestMismatch(t *testing.T) {
	record := validSessionRecordJSON(t)
	mutated := strings.Replace(string(record), `"name":"payments-api"`, `"name":"payments-worker"`, 1)
	if mutated == string(record) {
		t.Fatal("identity mutation anchor was not found in the fixture")
	}
	if _, err := ClassifyJSON([]byte(mutated)); !hasCode(err, "integrity_failure") {
		t.Fatalf("ClassifyJSON(object with stale record_id) = %v, want literal integrity_failure", err)
	}
}

func TestAddJSONQuarantinesSameIDDifferentStoredBytes(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	record := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(record)
	if err != nil {
		t.Fatal(err)
	}
	prior := []byte("tampered prior payload")
	index.mu.Lock()
	index.objects["record"][membership.ID.String()] = storedObject{data: prior, json: true}
	index.locations[membership.ID.String()] = "record"
	index.mu.Unlock()

	err = index.AddJSON(record)
	if !hasCode(err, "integrity_failure") {
		t.Fatalf("AddJSON(same digest, different bytes) = %v, want literal integrity_failure", err)
	}
	if ids := index.QuarantinedIDs(); len(ids) != 1 || ids[0] != membership.ID.String() {
		t.Fatalf("quarantine IDs = %#v, want the conflicting digest", ids)
	}
	root, err := index.Root("record")
	if err != nil || root.Count.Uint64() != 0 {
		t.Fatalf("quarantined object remains in inventory: %#v, %v", root, err)
	}
	if len(index.quarantine[membership.ID.String()]) != 2 {
		t.Fatalf("quarantined byte variants = %d, want prior plus incoming", len(index.quarantine[membership.ID.String()]))
	}
}

func TestRPC2SchemaMembershipTableIsTotalAndDisjoint(t *testing.T) {
	wantNamespaces := map[schemaVersion]string{
		{"urn:ax:schema:session-record", "1.0.0"}:       "record",
		{"urn:ax:schema:session-record", "2.0.0"}:       "record",
		{"urn:ax:schema:session-record", "3.0.0"}:       "record",
		{"urn:ax:schema:session-record", "3.1.0"}:       "record",
		{"urn:ax:schema:session-event", "1.0.0"}:        "event",
		{"urn:ax:schema:session-event", "2.0.0"}:        "event",
		{"urn:ax:schema:session-event", "3.0.0"}:        "event",
		{"urn:ax:schema:session-event", "4.0.0"}:        "event",
		{"urn:ax:schema:lease", "1.0.0"}:                "record",
		{"urn:ax:schema:checkpoint", "1.0.0"}:           "record",
		{"urn:ax:schema:workspace-group", "1.0.0"}:      "record",
		{"urn:ax:schema:provider-identity", "1.0.0"}:    "record",
		{"urn:ax:schema:transfer-manifest", "1.0.0"}:    "manifest",
		{blobDescriptorV1, "1.0.0"}:                     "manifest",
		{"urn:ax:schema:materialization-plan", "1.0.0"}: "manifest",
		{"urn:ax:schema:materialization-plan", "2.0.0"}: "manifest",
		{"urn:ax:schema:task-board-bundle", "1.0.0"}:    "manifest",
		{"urn:ax:schema:tombstone", "1.0.0"}:            "tombstone",
		{"urn:ax:schema:tombstone-ack", "1.0.0"}:        "tombstone_ack",
	}
	if len(rpc2SchemaNamespaces) != len(wantNamespaces) {
		t.Fatalf("schema namespace table has %d rows, want the contract's %d", len(rpc2SchemaNamespaces), len(wantNamespaces))
	}
	for key, want := range wantNamespaces {
		if got, ok := rpc2SchemaNamespaces[key]; !ok || got != want {
			t.Errorf("schema %s@%s maps to %q, %v; want %q", key.schema, key.version, got, ok, want)
		}
		if _, excluded := excludedSchemas[key.schema]; excluded {
			t.Errorf("schema %s is both included and excluded", key.schema)
		}
	}

	wantExcluded := []string{
		"urn:ax:schema:config",
		"urn:ax:schema:provider-manifest",
		"urn:ax:schema:provider-probe",
		"urn:ax:schema:terminal-backend-manifest",
		"urn:ax:schema:terminal-backend-probe",
		"urn:ax:schema:chunk",
		"urn:ax:schema:canonical-event",
		"urn:ax:schema:materialization-journal",
		"urn:ax:schema:terminal-instance-binding",
		"urn:ax:schema:session-adapter-manifest",
		"urn:ax:schema:session-adapter-probe",
		"urn:ax:schema:session-directory-node-manifest",
		"urn:ax:schema:session-directory-node-request",
		"urn:ax:schema:session-directory-node-response",
		"urn:ax:schema:host-trust-store",
		"urn:ax:schema:launch-plan-request",
		"urn:ax:schema:error",
		"urn:ax:schema:observation",
		"urn:ax:schema:cli-result",
		"urn:ax:schema:session-clone-bundle",
		"urn:ax:schema:clone-raw-object-manifest",
		"urn:ax:schema:clone-capture-manifest",
		"urn:ax:schema:canonical-session",
		"urn:ax:schema:migration-checkpoint",
		"urn:ax:schema:fidelity-report",
		"urn:ax:schema:projection-plan",
		"urn:ax:schema:clone-projected-object-manifest",
		"urn:ax:schema:clone-read-back-evidence-manifest",
		"urn:ax:schema:clone-validation-report",
		"urn:ax:schema:clone-lineage-receipt",
		"urn:ax:schema:supported-environment-tuples",
		"urn:ax:schema:environment-observation",
		"urn:ax:schema:native-session-observation",
		"urn:ax:schema:session-inventory-batch",
		"urn:ax:schema:conversation-lineage-link",
		"urn:ax:schema:session-annotation",
		"urn:ax:schema:session-enrichment-profile",
		"urn:ax:schema:session-enrichment-job-request",
		"urn:ax:schema:session-enrichment-job-receipt",
		"urn:ax:schema:session-continuation-plan",
		"urn:ax:schema:session-directory-operation-receipt",
		"urn:ax:schema:session-directory-query",
	}
	if len(excludedSchemas) != len(wantExcluded) {
		t.Fatalf("excluded schema table has %d rows, want the contract's %d", len(excludedSchemas), len(wantExcluded))
	}
	seenExcluded := make(map[string]struct{}, len(wantExcluded))
	for _, schema := range wantExcluded {
		if _, duplicate := seenExcluded[schema]; duplicate {
			t.Fatalf("test contract repeats excluded schema %s", schema)
		}
		seenExcluded[schema] = struct{}{}
		if _, ok := excludedSchemas[schema]; !ok {
			t.Errorf("contract exclusion %s is missing", schema)
		}
		for key := range rpc2SchemaNamespaces {
			if key.schema == schema {
				t.Errorf("schema %s is both included and excluded", schema)
			}
		}
		data := []byte(`{"schema":` + `"` + schema + `"` + `,"schema_version":"1.0.0"}`)
		membership, err := ClassifyJSON(data)
		if err != nil || !membership.Excluded || membership.Namespace != "" {
			t.Errorf("ClassifyJSON(%s) = %#v, %v; want an excluded class", schema, membership, err)
		}
	}

	for _, transient := range []string{`{}`, `{"protocol":"urn:ax:protocol:provider","protocol_version":"2.0.0"}`, `{"operation":"inventory.children"}`} {
		membership, err := ClassifyJSON([]byte(transient))
		if err != nil || !membership.Excluded || membership.Namespace != "" {
			t.Errorf("transient/local object %s = %#v, %v; want excluded", transient, membership, err)
		}
	}
	unknown := []byte(`{"schema":"urn:ax:schema:unrecognized","schema_version":"1.0.0"}`)
	if _, err := ClassifyJSON(unknown); !hasCode(err, "integrity_failure") {
		t.Errorf("unknown schema classification = %v; want fail-closed integrity_failure", err)
	}
}

func TestBlobDescriptorOwnsManifestMembershipAndCompleteRawBlob(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("complete raw blob")
	descriptor := validBlobDescriptorJSON(t, raw)
	membership, err := ClassifyJSON(descriptor)
	if err != nil || membership.Namespace != "manifest" {
		t.Fatalf("descriptor membership = %#v, %v; want manifest", membership, err)
	}
	if err := index.AddBlob(descriptor, raw); err != nil {
		t.Fatal(err)
	}
	manifestRoot, err := index.Root("manifest")
	if err != nil || manifestRoot.Count.Uint64() != 1 {
		t.Fatalf("manifest root = %#v, %v", manifestRoot, err)
	}
	blobRoot, err := index.Root("blob")
	if err != nil || blobRoot.Count.Uint64() != 1 || blobRoot.RootID.String() == "" {
		t.Fatalf("blob root = %#v, %v", blobRoot, err)
	}
	chunk := json.RawMessage(`{"schema":"urn:ax:schema:chunk","schema_version":"1.0.0","index":0,"offset":0,"size":1,"chunk_id":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}`)
	chunkMembership, err := ClassifyJSON(chunk)
	if err != nil || !chunkMembership.Excluded || chunkMembership.Namespace != "" {
		t.Fatalf("chunk membership = %#v, %v; want excluded", chunkMembership, err)
	}
	if err := index.AddJSON(chunk); !errors.Is(err, ErrExcluded) {
		t.Fatalf("AddJSON(chunk) = %v, want excluded", err)
	}
	unchangedBlobRoot, err := index.Root("blob")
	if err != nil || unchangedBlobRoot != blobRoot {
		t.Fatalf("excluded chunk changed raw blob root: before %#v after %#v err=%v", blobRoot, unchangedBlobRoot, err)
	}
}

func TestBlobDescriptorByteReferencesAreVerifiedBeforeAdmission(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("complete raw blob")
	wrongChunkID := scalar.SHA256Digest([]byte("different chunk")).String()
	descriptor := blobDescriptorWithChunkID(t, raw, wrongChunkID)
	if _, err := ClassifyJSON(descriptor); err != nil {
		t.Fatalf("descriptor itself should be schema-valid before raw bytes are attached: %v", err)
	}
	if err := index.AddBlob(descriptor, raw); !hasCode(err, "integrity_failure") {
		t.Fatalf("AddBlob() = %v, want literal integrity_failure", err)
	}
	for _, namespace := range []string{"manifest", "blob"} {
		root, err := index.Root(namespace)
		if err != nil || root.Count.Uint64() != 0 {
			t.Errorf("failed blob admission changed %s root: %#v, %v", namespace, root, err)
		}
	}
}

func TestMixedNSN1RejectsDescriptorRelabelingChunksAndLocalMarker(t *testing.T) {
	// First reproduce the complete good MIXED-NS-1 roots through Index.Root.
	// Then model the N1 class errors using the same synthetic bytewise IDs.
	good, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	z := func(n byte) string { return "sha256:" + strings.Repeat("0", 63) + string(n) }
	two := func(n byte) string { return "sha256:" + strings.Repeat("2", 63) + string(n) }
	rootFixture := func(namespace string, count uint64, id string) Root {
		value, rootErr := scalar.NewUint53(count)
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		return Root{Namespace: namespace, Count: value, RootID: mustDigest(t, id)}
	}
	want := map[string]Root{
		"record":        rootFixture("record", 5, "sha256:44c71ab5fdb7403c57a8a929d9fc90b2db3e8829b2615901c0841216d6750580"),
		"event":         rootFixture("event", 1, "sha256:533934b217b3d2f3999a2ec8cbb5ccfc0a7aff29c4eee0f7d35a0adca8cef15e"),
		"manifest":      rootFixture("manifest", 4, "sha256:cb60d01f92e1dae564a49f2ae782b9cf479a9e3849d1c84a8c907b68dec85202"),
		"tombstone":     rootFixture("tombstone", 1, "sha256:b890fa63b85a48c6e6a5e42cece3fb50e545062da103bb1f5bcfaa80d922f0d9"),
		"tombstone_ack": rootFixture("tombstone_ack", 1, "sha256:4f5c58b2cee87816f09520570056ebe64c408dc484ff192aaf8583a308eda8e5"),
		"blob":          rootFixture("blob", 1, "sha256:eaff5e30efa41b95ad419948886433af94ab3c95de2a574a1c1cfcf6d45a7cd2"),
	}
	goodMembers := map[string][]string{
		"record":        {z('1'), z('2'), z('3'), z('4'), z('5')},
		"event":         {"sha256:" + strings.Repeat("1", 64)},
		"manifest":      {two('1'), two('2'), two('3'), two('4')},
		"tombstone":     {"sha256:" + strings.Repeat("3", 64)},
		"tombstone_ack": {"sha256:" + strings.Repeat("4", 64)},
		"blob":          {"sha256:" + strings.Repeat("5", 64)},
	}
	for namespace, ids := range goodMembers {
		for _, id := range ids {
			seedNormativeSyntheticID(good, namespace, mustDigest(t, id))
		}
	}
	for namespace, expected := range want {
		got, rootErr := good.Root(namespace)
		if rootErr != nil || got != expected {
			t.Fatalf("MIXED-NS-N1 valid baseline %s root = %#v, %v; want %#v", namespace, got, rootErr, expected)
		}
	}

	bad, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	badMembers := map[string][]string{
		"record":        {z('1'), z('2'), z('3'), z('4'), z('5'), two('2'), "sha256:" + strings.Repeat("7", 64)},
		"event":         goodMembers["event"],
		"manifest":      {two('1'), two('3'), two('4')},
		"tombstone":     goodMembers["tombstone"],
		"tombstone_ack": goodMembers["tombstone_ack"],
		"blob":          {goodMembers["blob"][0], "sha256:" + strings.Repeat("6", 64)},
	}
	for namespace, ids := range badMembers {
		for _, id := range ids {
			seedNormativeSyntheticID(bad, namespace, mustDigest(t, id))
		}
	}
	for namespace, expected := range want {
		got, rootErr := bad.Root(namespace)
		if rootErr != nil {
			t.Fatalf("MIXED-NS-N1 invalid %s root: %v", namespace, rootErr)
		}
		switch namespace {
		case "record", "manifest", "blob":
			if got == expected {
				t.Errorf("MIXED-NS-N1 invalid %s assignment matched normative root %#v", namespace, expected)
			}
		default:
			if got != expected {
				t.Errorf("MIXED-NS-N1 changed unrelated %s root: got %#v, want %#v", namespace, got, expected)
			}
		}
	}

	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("synthetic complete content")
	descriptor := validBlobDescriptorJSON(t, raw)
	membership, err := ClassifyJSON(descriptor)
	if err != nil || membership.Namespace != "manifest" {
		t.Fatalf("descriptor membership = %#v, %v", membership, err)
	}
	if err := index.AddJSON(descriptor); err != nil {
		t.Fatal(err)
	}
	recordRoot, err := index.Root("record")
	if err != nil {
		t.Fatal(err)
	}
	manifestRoot, err := index.Root("manifest")
	if err != nil {
		t.Fatal(err)
	}
	if recordRoot.Count.Uint64() != 0 || recordRoot == want["record"] {
		t.Fatalf("MIXED-NS-N1 descriptor incorrectly entered record root: %#v", recordRoot)
	}
	if manifestRoot.Count.Uint64() != 1 {
		t.Fatalf("descriptor not present in its schema namespace: %#v", manifestRoot)
	}
	local := json.RawMessage(`{"schema":"urn:ax:schema:terminal-instance-binding","schema_version":"1.0.0","binding_id":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}`)
	localMembership, err := ClassifyJSON(local)
	if err != nil || !localMembership.Excluded {
		t.Fatalf("local marker membership = %#v, %v; want excluded", localMembership, err)
	}
	if err := index.AddJSON(local); !errors.Is(err, ErrExcluded) {
		t.Fatalf("AddJSON(local marker) = %v, want excluded", err)
	}
	chunk := json.RawMessage(`{"schema":"urn:ax:schema:chunk","schema_version":"1.0.0","index":0,"offset":0,"size":1,"chunk_id":"sha256:6666666666666666666666666666666666666666666666666666666666666666"}`)
	chunkMembership, err := ClassifyJSON(chunk)
	if err != nil || !chunkMembership.Excluded {
		t.Fatalf("independently enumerated Chunk membership = %#v, %v; want excluded", chunkMembership, err)
	}
	if err := index.AddJSON(chunk); !errors.Is(err, ErrExcluded) {
		t.Fatalf("AddJSON(independent Chunk) = %v, want excluded", err)
	}
	wrongNamespaceLine := objectsGetRequest(t, membership.ID.String())
	_, err = index.ObjectsGet(wrongNamespaceLine, "record", rpcwire.MaxLineBytes)
	if !hasCode(err, "integrity_failure") {
		t.Fatalf("MIXED-NS-N1 objects.get descriptor refusal = %v, want literal integrity_failure", err)
	}
}

func TestObjectsGetRevalidatesStoredSchemaMembership(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("complete raw blob")
	descriptor := validBlobDescriptorJSON(t, raw)
	membership, err := ClassifyJSON(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	// Model an internally misrouted stored object so the production schema
	// membership check is reached after the location-index fast path.
	index.mu.Lock()
	index.objects["record"][membership.ID.String()] = storedObject{data: descriptor, json: true}
	index.locations[membership.ID.String()] = "record"
	index.mu.Unlock()
	_, err = index.ObjectsGet(objectsGetRequest(t, membership.ID.String()), "record", rpcwire.MaxLineBytes)
	if !hasCode(err, "integrity_failure") {
		t.Fatalf("ObjectsGet(stored Descriptor as record) = %v, want literal integrity_failure", err)
	}
}

func TestObjectsGetValidatesRequestedNamespaceAndNegotiatedLineLimit(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	record := validSessionRecordJSON(t)
	membership, err := ClassifyJSON(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := index.AddJSON(record); err != nil {
		t.Fatal(err)
	}
	requestLine := objectsGetRequest(t, membership.ID.String())
	responseLine, err := index.ObjectsGet(requestLine, "record", rpcwire.MaxLineBytes)
	if err != nil {
		t.Fatal(err)
	}
	request, err := rpcwire.DecodeRequest(requestLine)
	if err != nil {
		t.Fatal(err)
	}
	response, err := rpcwire.DecodeResponse(responseLine, request)
	if err != nil || !response.OK() {
		t.Fatalf("objects.get response decode = %#v, %v", response, err)
	}
	if _, err := index.ObjectsGet(requestLine, "record", uint64(len(responseLine))); err != nil {
		t.Fatalf("objects.get exact negotiated response length = %v, want success", err)
	}
	var body struct {
		Objects []WireObject `json:"objects"`
	}
	if err := json.Unmarshal(response.Body(), &body); err != nil || len(body.Objects) != 1 {
		t.Fatalf("objects.get body = %s, %v", response.Body(), err)
	}
	wireObject := body.Objects[0]
	if wireObject.ObjectID != membership.ID.String() || wireObject.MediaType != "application/json" || wireObject.Encoding != "json" {
		t.Fatalf("WireObject = %#v", wireObject)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(wireObject.Data)
	if err != nil || string(decoded) != string(record) {
		t.Fatalf("WireObject.data decode mismatch: %v", err)
	}
	_, err = index.ObjectsGet(requestLine, "event", rpcwire.MaxLineBytes)
	if !hasCode(err, "integrity_failure") {
		t.Fatalf("cross-namespace objects.get = %v, want literal integrity_failure", err)
	}
	_, err = index.ObjectsGet(requestLine, "record", uint64(len(responseLine)-1))
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("small-line objects.get = %v, want response-size refusal", err)
	}
	if !hasCode(err, "incompatible_protocol") {
		t.Fatalf("small-line error = %v, want literal incompatible_protocol", err)
	}
}

func TestObjectsGetRefusesUnsortedDuplicateAndUnavailableEncodingRequests(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	id0 := "sha256:" + strings.Repeat("0", 64)
	id1 := "sha256:" + strings.Repeat("1", 64)
	cases := []struct {
		name string
		body string
	}{
		{"duplicate IDs", fmt.Sprintf(`{"object_ids":[%q,%q],"encodings":["json"]}`, id0, id0)},
		{"unsorted IDs", fmt.Sprintf(`{"object_ids":[%q,%q],"encodings":["json"]}`, id1, id0)},
		{"empty IDs", `{"object_ids":[],"encodings":["json"]}`},
		{"empty encodings", fmt.Sprintf(`{"object_ids":[%q],"encodings":[]}`, id0)},
		{"cbor only", fmt.Sprintf(`{"object_ids":[%q],"encodings":["cbor"]}`, id0)},
		{"duplicate encodings", fmt.Sprintf(`{"object_ids":[%q],"encodings":["json","json"]}`, id0)},
		{"unsorted encodings", fmt.Sprintf(`{"object_ids":[%q],"encodings":["json","cbor"]}`, id0)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			line, err := rpcwire.EncodeRequest(RPC2Version, testRequestID, "objects.get", json.RawMessage(test.body))
			if err != nil {
				t.Fatal(err)
			}
			_, err = index.ObjectsGet(line, "record", rpcwire.MaxLineBytes)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("ObjectsGet() = %v, want invalid request", err)
			}
		})
	}
}

func TestObjectsGetRefusesRequestIDCountOutsideBound(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 4097)
	for value := range ids {
		ids[value] = fmt.Sprintf("sha256:%064x", value)
	}
	maxIDs := ids[:4096]
	_, err = index.ObjectsGet(objectsGetRequestMany(t, maxIDs), "record", rpcwire.MaxLineBytes)
	if !hasCode(err, "not_found") {
		t.Fatalf("ObjectsGet(4096 sorted IDs) = %v, want literal not_found after accepting the request bound", err)
	}
	line := objectsGetRequestMany(t, ids)
	_, err = index.ObjectsGet(line, "record", rpcwire.MaxLineBytes)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ObjectsGet(4097 IDs) = %v, want invalid request", err)
	}
	if _, err := FetchObjects(index, "record", ids, rpcwire.MaxLineBytes, func() (string, error) { return testRequestID, nil }); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("FetchObjects(4097 IDs) = %v, want invalid request", err)
	}
}

func TestFetchObjectsUsesBoundedSingletonBatches(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	first := validSessionRecordNamedJSON(t, "payments-api-a")
	second := validSessionRecordNamedJSON(t, "payments-api-b")
	firstMembership, err := ClassifyJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	secondMembership, err := ClassifyJSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if err := index.AddJSON(first); err != nil {
		t.Fatal(err)
	}
	if err := index.AddJSON(second); err != nil {
		t.Fatal(err)
	}
	ids := []string{firstMembership.ID.String(), secondMembership.ID.String()}
	slices.Sort(ids)
	lineLimit := uint64(0)
	for _, id := range ids {
		line, err := index.ObjectsGet(objectsGetRequest(t, id), "record", rpcwire.MaxLineBytes)
		if err != nil {
			t.Fatal(err)
		}
		lineLimit = max(lineLimit, uint64(len(line)))
	}
	combinedRequest := objectsGetRequestMany(t, ids)
	if _, err := index.ObjectsGet(combinedRequest, "record", lineLimit); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("combined objects.get = %v, want negotiated-line refusal", err)
	}
	source := &recordingSource{index: index}
	requestIDs := []string{
		"0198f4c8-7a10-7b22-8b3c-2234567890ab",
		"0198f4c8-7a10-7b22-8b3c-2234567890ac",
	}
	next := 0
	objects, err := FetchObjects(source, "record", ids, lineLimit, func() (string, error) {
		id := requestIDs[next]
		next++
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 2 || len(objects) != 2 || next != 2 {
		t.Fatalf("bounded fetch calls/objects/IDs = %d/%d/%d, want 2/2/2", source.calls, len(objects), next)
	}
	for i, object := range objects {
		if object.ObjectID != ids[i] {
			t.Errorf("objects[%d].object_id = %q, want %q", i, object.ObjectID, ids[i])
		}
	}
}

func TestInProcessExchangeWalksChangedNamespacesAndFetchesExactObjects(t *testing.T) {
	local, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	commonRecord := validSessionRecordJSON(t)
	commonTombstone := tombstoneJSON(t, "2026-08-19T04:20:00.000Z")
	commonTombstoneMembership, err := ClassifyJSON(commonTombstone)
	if err != nil {
		t.Fatal(err)
	}
	commonAck := tombstoneAckJSON(t, commonTombstoneMembership.ID.String(), "applied", "", "2026-08-19T04:21:00.000Z")
	checkpoint := validCheckpointJSON(t)
	rawBlob := []byte("checkpoint referenced content")
	descriptor := validBlobDescriptorJSON(t, rawBlob)
	checkpointMembership, err := ClassifyJSON(checkpoint)
	if err != nil || checkpointMembership.Namespace != "record" {
		t.Fatalf("Checkpoint membership = %#v, %v", checkpointMembership, err)
	}
	descriptorMembership, err := ClassifyJSON(descriptor)
	if err != nil || descriptorMembership.Namespace != "manifest" {
		t.Fatalf("Descriptor membership = %#v, %v", descriptorMembership, err)
	}
	if err := local.AddJSON(commonRecord); err != nil {
		t.Fatal(err)
	}
	if err := local.AddJSON(commonTombstone); err != nil {
		t.Fatal(err)
	}
	if err := local.AddJSON(commonAck); err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(commonRecord); err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(commonTombstone); err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(commonAck); err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := peer.AddBlob(descriptor, rawBlob); err != nil {
		t.Fatal(err)
	}
	changed := []string{}
	for _, namespace := range rpc2Namespaces {
		localRoot, err := local.Root(namespace)
		if err != nil {
			t.Fatal(err)
		}
		peerRoot, err := peer.Root(namespace)
		if err != nil {
			t.Fatal(err)
		}
		if localRoot.RootID != peerRoot.RootID {
			changed = append(changed, namespace)
		}
	}
	if !slices.Equal(changed, []string{"blob", "manifest", "record"}) {
		t.Fatalf("MIXED-NS-EXCHANGE changed namespaces = %#v, want [blob manifest record]", changed)
	}
	missingRecords, err := MissingObjectIDs(local, peer, "record")
	if err != nil || !slices.Equal(missingRecords, []string{checkpointMembership.ID.String()}) {
		t.Fatalf("record recursive walk = %#v, %v; want exactly missing Checkpoint", missingRecords, err)
	}
	missingManifests, err := MissingObjectIDs(local, peer, "manifest")
	if err != nil || !slices.Equal(missingManifests, []string{descriptorMembership.ID.String()}) {
		t.Fatalf("manifest recursive walk = %#v, %v; want exactly missing Descriptor", missingManifests, err)
	}
	if _, err := MissingObjectIDs(local, peer, "blob"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("blob object walk = %v, want explicit Section 11.5 transfer boundary", err)
	}
	requestIDs := []string{testRequestID, "0198f4c8-7a10-7b22-8b3c-2234567890ac"}
	next := 0
	newID := func() (string, error) {
		id := requestIDs[next]
		next++
		return id, nil
	}
	checkpointObjects, err := FetchObjects(peer, "record", missingRecords, rpcwire.MaxLineBytes, newID)
	if err != nil || len(checkpointObjects) != 1 || checkpointObjects[0].ObjectID != checkpointMembership.ID.String() {
		t.Fatalf("fetched Checkpoint = %#v, %v", checkpointObjects, err)
	}
	if err := local.AddJSON(decodeWireData(t, checkpointObjects[0])); err != nil {
		t.Fatal(err)
	}
	localRecordRoot, err := local.Root("record")
	peerRecordRoot, peerErr := peer.Root("record")
	if err != nil || peerErr != nil || localRecordRoot.RootID != peerRecordRoot.RootID {
		t.Fatalf("record roots differ before blob admission: %v / %v", localRecordRoot, peerRecordRoot)
	}
	descriptorObjects, err := FetchObjects(peer, "manifest", missingManifests, rpcwire.MaxLineBytes, newID)
	if err != nil || len(descriptorObjects) != 1 || descriptorObjects[0].ObjectID != descriptorMembership.ID.String() {
		t.Fatalf("fetched Descriptor = %#v, %v", descriptorObjects, err)
	}
	descriptorBytes := decodeWireData(t, descriptorObjects[0])
	if err := local.AddJSON(descriptorBytes); err != nil {
		t.Fatal(err)
	}
	// Raw bytes cross the admission boundary only after the record union. Actual
	// chunks.put transport and resumable staging are owned by Section 11.5.
	if err := local.AddBlob(descriptorBytes, rawBlob); err != nil {
		t.Fatal(err)
	}
	wantFinalRoots := map[string]struct {
		count uint64
		root  string
	}{
		"blob":          {1, "sha256:dc1027aa615302df482d5485f52c444e0c6b47555832082cd21a4098ef8fe36c"},
		"event":         {0, "sha256:928ecd990c39d77bcd631915f9accc538336a3df722b4414c360a2321ad34421"},
		"manifest":      {1, "sha256:ceaa07174b269be4c83e90db36d759445f8eee7721c44b2fca137eaf6afc343e"},
		"record":        {2, "sha256:0d3e6a4ccdd428363e86446144d1b38dc50bd5a919d4e19e592bc40cb758f5d8"},
		"tombstone":     {1, "sha256:fb9e23c8c6b3740c395c47cf105e6a7c17c7a21f5fd2737e67f1b0645b0cec4e"},
		"tombstone_ack": {1, "sha256:381ecbb6d2b03590add6234dff998a47c6608384df04c8277dc28b05e2f8c15f"},
	}
	for _, namespace := range rpc2Namespaces {
		localRoot, err := local.Root(namespace)
		if err != nil {
			t.Fatal(err)
		}
		peerRoot, err := peer.Root(namespace)
		if err != nil {
			t.Fatal(err)
		}
		if localRoot != peerRoot {
			t.Errorf("post-exchange %s roots differ: %#v / %#v", namespace, localRoot, peerRoot)
		}
		want := wantFinalRoots[namespace]
		if localRoot.Count.Uint64() != want.count || localRoot.RootID.String() != want.root {
			t.Errorf("post-exchange %s count/root = %d/%s, want literal fixture %d/%s", namespace, localRoot.Count.Uint64(), localRoot.RootID, want.count, want.root)
		}
	}

	// Keep the proof on real Descriptor bytes at the objects.get entry too: a
	// misrouted stored object must not be returned as a record during exchange.
	misrouted, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	misrouted.mu.Lock()
	misrouted.objects["record"][descriptorMembership.ID.String()] = storedObject{data: descriptor, json: true}
	misrouted.locations[descriptorMembership.ID.String()] = "record"
	misrouted.mu.Unlock()
	_, err = misrouted.ObjectsGet(objectsGetRequest(t, descriptorMembership.ID.String()), "record", rpcwire.MaxLineBytes)
	if !hasCode(err, "integrity_failure") {
		t.Fatalf("exchange objects.get returned a manifest Descriptor as record: %v, want literal integrity_failure", err)
	}
}

func TestNewRefusesHigherRPCNamespaceSets(t *testing.T) {
	for _, version := range []string{"3.0.0", "4.0.0", "5.0.0"} {
		if _, err := New(version); !errors.Is(err, ErrUnsupportedVersion) {
			t.Errorf("New(%q) = %v, want explicit unsupported-version refusal", version, err)
		}
	}
	if _, err := New("6.0.0"); !errors.Is(err, rpcwire.ErrVersion) {
		t.Errorf("New(6.0.0) = %v, want rpcwire version refusal", err)
	}
}

func validSessionRecordJSON(t *testing.T) []byte {
	t.Helper()
	object := map[string]any{
		"schema": "urn:ax:schema:session-record", "schema_version": "1.0.0", "record_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "name": "payments-api", "kind": "direct",
		"created_at": "2026-08-19T04:00:00.000Z", "created_by_host_id": testHostID, "provider_id": "codex",
		"workspace_group_id": testGroupID, "execution_profile": "yolo",
		"launch_plan": map[string]any{
			"argv": []string{"codex"}, "cwd_workspace_id": testWorkspace, "cwd_relative": "src",
			"env_names": []string{}, "env_literals": map[string]string{}, "contains_secrets": false, "extensions": map[string]any{},
		},
		"task_board": nil, "fork_provenance": nil, "extensions": map[string]any{},
	}
	return sealIdentity(t, object, "record_id")
}

func validSessionRecordNamedJSON(t *testing.T, name string) []byte {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(validSessionRecordJSON(t), &object); err != nil {
		t.Fatal(err)
	}
	object["record_id"] = zeroDigest
	object["name"] = name
	return sealIdentity(t, object, "record_id")
}

func validCheckpointJSON(t *testing.T) []byte {
	t.Helper()
	object := map[string]any{
		"schema": "urn:ax:schema:checkpoint", "schema_version": "1.0.0", "checkpoint_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "lease_epoch": 4, "lease_id": "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		"safe_boundary": map[string]any{
			"provider_id": "codex", "provider_version": "0.147.0", "evidence": "accepted_test",
			"input_blocked": true, "foreground_idle": true, "background_idle": true,
			"open_processes": 0, "open_database_handles": 0,
		},
		"event_heads": []string{zeroDigest}, "workspace_manifest_id": zeroDigest,
		"provider_manifest_id": zeroDigest, "task_board_bundle_id": nil,
		"created_by_host_id": testHostID, "created_at": "2026-08-19T04:09:30.000Z", "status": "validated", "extensions": map[string]any{},
	}
	return sealIdentity(t, object, "checkpoint_id")
}

func validBlobDescriptorJSON(t *testing.T, raw []byte) []byte {
	t.Helper()
	digest := scalar.SHA256Digest(raw).String()
	return blobDescriptorWithChunkID(t, raw, digest)
}

func blobDescriptorWithChunkID(t *testing.T, raw []byte, chunkID string) []byte {
	t.Helper()
	digest := scalar.SHA256Digest(raw).String()
	object := map[string]any{
		"schema": "urn:ax:schema:blob", "schema_version": "1.0.0", "descriptor_id": zeroDigest,
		"blob_id": digest, "size": len(raw), "media_type": "application/octet-stream",
		"chunks": []any{map[string]any{"index": 0, "offset": 0, "size": len(raw), "chunk_id": chunkID}},
	}
	return sealIdentity(t, object, "descriptor_id")
}

func sealIdentity(t *testing.T, object map[string]any, selfField string) []byte {
	t.Helper()
	initial, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	identity, _, err := canonicaljson.CalculateObjectIdentity(initial)
	if err != nil {
		t.Fatalf("CalculateObjectIdentity: %v", err)
	}
	object[selfField] = identity.String()
	sealed, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	verified, _, err := canonicaljson.VerifyObjectIdentity(sealed)
	if err != nil || verified.String() != identity.String() {
		t.Fatalf("VerifyObjectIdentity: %v; id=%q want %q", err, verified, identity)
	}
	return sealed
}

func objectsGetRequest(t *testing.T, id string) []byte {
	t.Helper()
	return objectsGetRequestMany(t, []string{id})
}

func objectsGetRequestMany(t *testing.T, ids []string) []byte {
	t.Helper()
	body, err := json.Marshal(struct {
		ObjectIDs []string `json:"object_ids"`
		Encodings []string `json:"encodings"`
	}{ObjectIDs: ids, Encodings: []string{"json"}})
	if err != nil {
		t.Fatal(err)
	}
	line, err := rpcwire.EncodeRequest(RPC2Version, testRequestID, "objects.get", body)
	if err != nil {
		t.Fatal(err)
	}
	return line
}

func decodeWireData(t *testing.T, object WireObject) []byte {
	t.Helper()
	data, err := base64.RawURLEncoding.DecodeString(object.Data)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type recordingSource struct {
	index *Index
	calls int
}

func (source *recordingSource) ObjectsGet(line []byte, namespace string, maxLineBytes uint64) ([]byte, error) {
	source.calls++
	return source.index.ObjectsGet(line, namespace, maxLineBytes)
}
