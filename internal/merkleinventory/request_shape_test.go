package merkleinventory

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

const invalidRequestLiteral = "invalid inventory request"

func TestDispatchRequestShapeStrictTypes(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	requestFields := []struct {
		operation string
		field     string
		valid     string
	}{
		{operation: "inventory.roots", field: "namespaces", valid: `["record"]`},
		{operation: "inventory.children", field: "namespace", valid: `"record"`},
		{operation: "inventory.children", field: "prefix", valid: `""`},
		{operation: "objects.get", field: "object_ids", valid: `["` + zeroDigest + `"]`},
		{operation: "objects.get", field: "encodings", valid: `["json"]`},
	}
	jsonKinds := []struct {
		name  string
		value string
	}{
		{name: "string", value: `"wrong-kind"`},
		{name: "number", value: `1`},
		{name: "boolean", value: `false`},
		{name: "null", value: `null`},
		{name: "array", value: `[]`},
		{name: "object", value: `{}`},
	}
	for _, field := range requestFields {
		operationSlug := strings.ReplaceAll(field.operation, ".", "_")
		for _, kind := range jsonKinds {
			value := kind.value
			if kind.name == "array" && (field.field == "namespaces" || field.field == "object_ids" || field.field == "encodings") {
				// The outer JSON kind is correct; a null member is not a string.
				value = `[null]`
			}
			if kind.name == "string" {
				switch field.field {
				case "namespace":
					value = `"unknown"`
				case "prefix":
					value = `"g"`
				case "encodings":
					value = `"json"`
				case "object_ids":
					value = `"` + zeroDigest + `"`
				}
			}
			t.Run(operationSlug+"_"+field.field+"_"+kind.name, func(t *testing.T) {
				assertRequestRejectedWithoutBody(t, index, field.operation, requestBody(field.operation, field.field, value))
			})
		}
		if field.field == "namespaces" || field.field == "object_ids" || field.field == "encodings" {
			members := []struct {
				name  string
				value string
			}{
				{name: "string", value: `"invalid-value"`},
				{name: "number", value: `1`},
				{name: "boolean", value: `false`},
				{name: "null", value: `null`},
				{name: "array", value: `[]`},
				{name: "object", value: `{}`},
			}
			for _, member := range members {
				t.Run(operationSlug+"_"+field.field+"_array_member_"+member.name, func(t *testing.T) {
					assertRequestRejectedWithoutBody(t, index, field.operation, requestBody(field.operation, field.field, `[`+member.value+`]`))
				})
			}
		}
		t.Run(operationSlug+"_"+field.field+"_missing", func(t *testing.T) {
			assertRequestRejectedWithoutBody(t, index, field.operation, missingRequestBody(field.operation, field.field))
		})
		t.Run(operationSlug+"_"+field.field+"_duplicate", func(t *testing.T) {
			assertRequestRejectedWithoutBody(t, index, field.operation, duplicateRequestBody(field.operation, field.field, field.valid))
		})
	}
}

func assertRequestRejectedWithoutBody(t *testing.T, index *Index, operation, body string) {
	t.Helper()
	line := rawRPCRequest(t, operation, body)
	context := []DispatchContext(nil)
	if operation == "objects.get" {
		context = append(context, DispatchContext{Namespace: "record", MaxLineBytes: rpcwire.MaxLineBytes})
	}
	response, err := index.Dispatch(line, context...)
	if response != nil {
		t.Fatalf("malformed %s request returned a body: %s", operation, response)
	}
	if err == nil || !errors.Is(err, ErrInvalidRequest) || err.Error() != invalidRequestLiteral {
		t.Fatalf("malformed %s request error = %v, want literal %q", operation, err, invalidRequestLiteral)
	}
}

func requestBody(operation, field, value string) string {
	switch operation + "/" + field {
	case "inventory.roots/namespaces":
		return `{"namespaces":` + value + `}`
	case "inventory.children/namespace":
		return `{"namespace":` + value + `,"prefix":""}`
	case "inventory.children/prefix":
		return `{"namespace":"record","prefix":` + value + `}`
	case "objects.get/object_ids":
		return `{"object_ids":` + value + `,"encodings":["json"]}`
	case "objects.get/encodings":
		return `{"object_ids":["` + zeroDigest + `"],"encodings":` + value + `}`
	default:
		panic("unhandled request field " + operation + "/" + field)
	}
}

func missingRequestBody(operation, field string) string {
	switch operation + "/" + field {
	case "inventory.roots/namespaces":
		return `{}`
	case "inventory.children/namespace":
		return `{"prefix":""}`
	case "inventory.children/prefix":
		return `{"namespace":"record"}`
	case "objects.get/object_ids":
		return `{"encodings":["json"]}`
	case "objects.get/encodings":
		return `{"object_ids":["` + zeroDigest + `"]}`
	default:
		panic("unhandled request field " + operation + "/" + field)
	}
}

func duplicateRequestBody(operation, field, valid string) string {
	switch operation + "/" + field {
	case "inventory.roots/namespaces":
		return `{"namespaces":` + valid + `,"namespaces":["record"]}`
	case "inventory.children/namespace":
		return `{"namespace":"record","namespace":"manifest","prefix":""}`
	case "inventory.children/prefix":
		return `{"namespace":"record","prefix":"","prefix":"0"}`
	case "objects.get/object_ids":
		return `{"object_ids":["` + zeroDigest + `"],"object_ids":["` + zeroDigest + `"],"encodings":["json"]}`
	case "objects.get/encodings":
		return `{"object_ids":["` + zeroDigest + `"],"encodings":["json"],"encodings":["json"]}`
	default:
		panic("unhandled request field " + operation + "/" + field)
	}
}

func rawRPCRequest(t *testing.T, operation, body string) []byte {
	t.Helper()
	envelope := struct {
		Protocol        string          `json:"protocol"`
		ProtocolVersion string          `json:"protocol_version"`
		RequestID       string          `json:"request_id"`
		Operation       string          `json:"operation"`
		Body            json.RawMessage `json:"body"`
	}{
		Protocol: rpcwire.Protocol, ProtocolVersion: RPC2Version, RequestID: testRequestID,
		Operation: operation, Body: json.RawMessage(body),
	}
	line, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return line
}

func TestDispatchPrefixAxisCoversEveryLengthAndAlphabet(t *testing.T) {
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	seedNormativeSyntheticID(index, "record", mustDigest(t, zeroDigest))
	for length := 0; length <= 65; length++ {
		length := length
		if length == 0 {
			t.Run("lower_hex_len_00_root", func(t *testing.T) {
				assertChildrenBody(t, index, "")
			})
			continue
		}
		if length <= 64 {
			t.Run(fmt.Sprintf("lower_hex_len_%02d_materialized", length), func(t *testing.T) {
				assertChildrenBody(t, index, strings.Repeat("0", length))
			})
			t.Run(fmt.Sprintf("lower_hex_len_%02d_absent", length), func(t *testing.T) {
				_, err := dispatchChildren(index, strings.Repeat("f", length))
				if !hasCode(err, "not_found") {
					t.Fatalf("valid absent prefix length %d error = %v, want literal not_found", length, err)
				}
			})
		} else {
			t.Run("lower_hex_len_65_refused", func(t *testing.T) {
				assertRequestRejectedWithoutBody(t, index, "inventory.children", requestBody("inventory.children", "prefix", `"`+strings.Repeat("0", length)+`"`))
			})
		}
		for _, axis := range []struct {
			name   string
			prefix string
		}{
			{name: "uppercase", prefix: "A" + strings.Repeat("0", length-1)},
			{name: "non_hex", prefix: "g" + strings.Repeat("0", length-1)},
		} {
			axis := axis
			t.Run(fmt.Sprintf("%s_len_%02d_refused", axis.name, length), func(t *testing.T) {
				assertRequestRejectedWithoutBody(t, index, "inventory.children", requestBody("inventory.children", "prefix", `"`+axis.prefix+`"`))
			})
		}
	}
}

func assertChildrenBody(t *testing.T, index *Index, prefix string) {
	t.Helper()
	body, err := dispatchChildren(index, prefix)
	if err != nil {
		t.Fatalf("materialized prefix %q error = %v", prefix, err)
	}
	var node Node
	if err := json.Unmarshal(body, &node); err != nil {
		t.Fatalf("children body for prefix %q: %v", prefix, err)
	}
	if node.Namespace != "record" || node.Prefix != prefix {
		t.Fatalf("children body identifies (%q,%q), want (record,%q)", node.Namespace, node.Prefix, prefix)
	}
}

func dispatchChildren(index *Index, prefix string) ([]byte, error) {
	body, err := json.Marshal(struct {
		Namespace string `json:"namespace"`
		Prefix    string `json:"prefix"`
	}{Namespace: "record", Prefix: prefix})
	if err != nil {
		return nil, err
	}
	line, err := rpcwire.EncodeRequest(RPC2Version, testRequestID, "inventory.children", body)
	if err != nil {
		return nil, err
	}
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		return nil, err
	}
	responseLine, err := index.Dispatch(line)
	if err != nil {
		return nil, err
	}
	response, err := rpcwire.DecodeResponse(responseLine, request)
	if err != nil {
		return nil, err
	}
	if !response.OK() {
		return nil, fmt.Errorf("Dispatch returned a failure response")
	}
	return response.Body(), nil
}

func TestObjectsGetRefusesEveryForeignNamespacePair(t *testing.T) {
	index, objects := inventoryObjectsByNamespace(t)
	requestedNamespaces := []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"}
	trueNamespaces := []string{"event", "manifest", "record", "tombstone", "tombstone_ack"}
	for _, requested := range requestedNamespaces {
		for _, trueNamespace := range trueNamespaces {
			requested, trueNamespace := requested, trueNamespace
			t.Run("requested_"+requested+"_object_"+trueNamespace, func(t *testing.T) {
				membership, err := ClassifyJSON(objects[trueNamespace])
				if err != nil {
					t.Fatal(err)
				}
				line := objectsGetRequest(t, membership.ID.String())
				response, err := index.ObjectsGet(line, requested, rpcwire.MaxLineBytes)
				if requested != trueNamespace {
					if response != nil || !hasCode(err, "integrity_failure") {
						t.Fatalf("ObjectsGet(%s, %s) returned body=%t, error=%v; want no body and literal integrity_failure", requested, trueNamespace, response != nil, err)
					}
					return
				}
				if err != nil || len(response) == 0 {
					t.Fatalf("ObjectsGet(%s, %s) response length=%d, error=%v; want body", requested, trueNamespace, len(response), err)
				}
			})
		}
	}
}

func TestFetchObjectsRefusesEveryHostileForeignNamespacePair(t *testing.T) {
	_, objects := inventoryObjectsByNamespace(t)
	jsonNamespaces := []string{"event", "manifest", "record", "tombstone", "tombstone_ack"}
	for _, requested := range jsonNamespaces {
		for _, trueNamespace := range jsonNamespaces {
			requested, trueNamespace := requested, trueNamespace
			t.Run("requested_"+requested+"_object_"+trueNamespace, func(t *testing.T) {
				data := objects[trueNamespace]
				membership, err := ClassifyJSON(data)
				if err != nil {
					t.Fatal(err)
				}
				source := hostileObjectSource{object: WireObject{
					ObjectID: membership.ID.String(), MediaType: "application/json", Encoding: "json",
					Data: base64.RawURLEncoding.EncodeToString(data),
				}}
				objects, err := FetchObjects(&source, requested, []string{membership.ID.String()}, rpcwire.MaxLineBytes, func() (string, error) {
					return testRequestID, nil
				})
				if requested != trueNamespace {
					if objects != nil || !hasCode(err, "integrity_failure") {
						t.Fatalf("FetchObjects(%s, %s) returned %d objects, error=%v; want no objects and literal integrity_failure", requested, trueNamespace, len(objects), err)
					}
					return
				}
				if err != nil || len(objects) != 1 || objects[0].ObjectID != membership.ID.String() {
					t.Fatalf("FetchObjects(%s, %s) returned %d objects, error=%v; want the validated object", requested, trueNamespace, len(objects), err)
				}
			})
		}
	}
}

func inventoryObjectsByNamespace(t *testing.T) (*Index, map[string][]byte) {
	t.Helper()
	index, err := New(RPC2Version)
	if err != nil {
		t.Fatal(err)
	}
	objects := map[string][]byte{
		"record":    validSessionRecordJSON(t),
		"event":     validInventorySessionEventJSON(t),
		"manifest":  validBlobDescriptorJSON(t, []byte("inventory namespace fixture")),
		"tombstone": tombstoneJSON(t, "2026-08-19T04:20:00.000Z"),
	}
	tombstoneMembership, err := ClassifyJSON(objects["tombstone"])
	if err != nil {
		t.Fatal(err)
	}
	objects["tombstone_ack"] = tombstoneAckJSON(t, tombstoneMembership.ID.String(), "applied", "", "2026-08-19T04:21:00.000Z")
	for _, namespace := range []string{"event", "manifest", "record", "tombstone", "tombstone_ack"} {
		membership, err := ClassifyJSON(objects[namespace])
		if err != nil || membership.Excluded || membership.Namespace != namespace {
			t.Fatalf("fixture for %s membership = %#v, %v", namespace, membership, err)
		}
		if err := index.AddJSON(objects[namespace]); err != nil {
			t.Fatalf("AddJSON(%s fixture): %v", namespace, err)
		}
	}
	return index, objects
}

func validInventorySessionEventJSON(t *testing.T) []byte {
	t.Helper()
	object := map[string]any{
		"schema": "urn:ax:schema:session-event", "schema_version": "1.0.0", "event_id": zeroDigest,
		"subject_id": testSessionID, "session_id": testSessionID, "event_type": "session.created",
		"created_by_host_id": testHostID, "lease_epoch": 4, "lease_id": "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
		"lease_sequence": 1, "predecessors": []string{zeroDigest}, "created_at": "2026-08-19T04:08:00.000Z",
		"payload": map[string]any{
			"session_record_id": zeroDigest, "bootstrap_operation_id": testRequestID,
			"first_checkpoint_operation_id": "0198f4c8-6c30-7d44-8d5e-1234567890ac",
		},
		"extensions": map[string]any{},
	}
	return sealIdentity(t, object, "event_id")
}

type hostileObjectSource struct {
	object WireObject
}

func (source *hostileObjectSource) ObjectsGet(line []byte, _ string, _ uint64) ([]byte, error) {
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(struct {
		Objects []WireObject `json:"objects"`
	}{Objects: []WireObject{source.object}})
	if err != nil {
		return nil, err
	}
	return rpcwire.EncodeSuccess(request, body)
}

func TestRequestShapeFixtureIdentitiesAreDistinct(t *testing.T) {
	_, objects := inventoryObjectsByNamespace(t)
	seen := map[string]string{}
	for namespace, data := range objects {
		membership, err := ClassifyJSON(data)
		if err != nil {
			t.Fatal(err)
		}
		if prior, exists := seen[membership.ID.String()]; exists {
			t.Fatalf("%s and %s fixtures share object ID %s", namespace, prior, membership.ID)
		}
		seen[membership.ID.String()] = namespace
	}
	if len(seen) != 5 {
		t.Fatalf("fixture namespace count = %d, want 5", len(seen))
	}
}
