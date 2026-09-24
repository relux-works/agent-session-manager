package merkleinventory

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Dispatch validates a complete request through rpcwire and serves inventory
// operations. objects.get requires its namespace and response-line limit as
// negotiated out-of-band context.
func (index *Index) Dispatch(line []byte, contexts ...DispatchContext) ([]byte, error) {
	if len(contexts) > 1 {
		return nil, ErrInvalidRequest
	}
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	if request.Version() != index.version {
		return nil, ErrUnsupportedVersion
	}
	var body json.RawMessage
	switch request.Operation() {
	case "inventory.roots":
		if len(contexts) != 0 {
			return nil, ErrInvalidRequest
		}
		body, err = index.rootsBody(request.Body())
	case "inventory.children":
		if len(contexts) != 0 {
			return nil, ErrInvalidRequest
		}
		body, err = index.childrenBody(request.Body())
	case "objects.get":
		if len(contexts) != 1 {
			return nil, ErrInvalidRequest
		}
		return index.objectsGetRequest(request, contexts[0])
	default:
		return nil, ErrInvalidRequest
	}
	if err != nil {
		return nil, err
	}
	return rpcwire.EncodeSuccess(request, body)
}

func (index *Index) rootsBody(requestBody []byte) (json.RawMessage, error) {
	request, err := strictObject(requestBody)
	if err != nil || !exactMembers(request, "namespaces") {
		return nil, ErrInvalidRequest
	}
	// rpcwire.DecodeRequest already enforces exact keys, supported vocabulary,
	// sorted uniqueness and the [1..6] RPC 2 cardinality.
	namespaces, ok := requiredStringArray(request, "namespaces")
	if !ok {
		return nil, ErrInvalidRequest
	}
	roots := make([]Root, 0, len(namespaces))
	for _, namespace := range namespaces {
		node, err := index.nodeFor(namespace, "")
		if err != nil {
			return nil, err
		}
		roots = append(roots, Root{Namespace: namespace, Count: node.Count, RootID: node.NodeHash})
	}
	encoded, err := json.Marshal(struct {
		Roots []Root `json:"roots"`
	}{Roots: roots})
	return encoded, err
}

func (index *Index) childrenBody(requestBody []byte) (json.RawMessage, error) {
	object, err := strictObject(requestBody)
	if err != nil || !exactMembers(object, "namespace", "prefix") {
		return nil, ErrInvalidRequest
	}
	namespace, ok := requiredJSONString(object, "namespace")
	if !ok || !validNamespace(namespace) {
		return nil, ErrInvalidRequest
	}
	prefix, ok := requiredJSONString(object, "prefix")
	if !ok || !validPrefix(prefix) {
		return nil, ErrInvalidRequest
	}
	node, err := index.nodeFor(namespace, prefix)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(node)
	return encoded, err
}

func (index *Index) nodeFor(namespace, prefix string) (Node, error) {
	if !validNamespace(namespace) || !validPrefix(prefix) {
		return Node{}, ErrInvalidRequest
	}
	index.mu.RLock()
	ids := make([]string, 0, len(index.objects[namespace]))
	for id := range index.objects[namespace] {
		ids = append(ids, id)
	}
	index.mu.RUnlock()
	node, err := ComputeNode(namespace, prefix, ids)
	if err != nil {
		return Node{}, err
	}
	if prefix != "" && node.Count.Uint64() == 0 {
		return Node{}, refusal("not_found", ErrNotFound)
	}
	return node, nil
}

// Root is the production root calculation for a namespace snapshot.
func (index *Index) Root(namespace string) (Root, error) {
	node, err := index.nodeFor(namespace, "")
	if err != nil {
		return Root{}, err
	}
	return Root{Namespace: namespace, Count: node.Count, RootID: node.NodeHash}, nil
}

// Child returns one materialized node. A non-root prefix without members is
// not an empty node; it is the literal Section 11 refusal not_found.
func (index *Index) Child(namespace, prefix string) (Node, error) {
	return index.nodeFor(namespace, prefix)
}

// MissingObjectIDs recursively walks only peer nodes whose node hashes differ
// and returns peer JSON-object IDs absent from the local namespace. Raw blob
// IDs are a Section 11.5 transfer boundary and are refused here.
func MissingObjectIDs(local, peer *Index, namespace string) ([]string, error) {
	if namespace == "blob" {
		return nil, ErrInvalidRequest
	}
	return MissingNamespaceIDs(local, peer, namespace)
}

// MissingNamespaceIDs recursively walks peer inventory nodes whose hashes
// differ and returns the peer identity IDs absent from the local namespace.
// For the blob namespace this returns identities only; it never retrieves or
// transfers raw bytes. Section 11.5 owns blob transfer.
func MissingNamespaceIDs(local, peer *Index, namespace string) ([]string, error) {
	if local == nil || peer == nil || local.version != peer.version || !validNamespace(namespace) {
		return nil, ErrInvalidRequest
	}
	peerRoot, err := peer.Child(namespace, "")
	if err != nil {
		return nil, err
	}
	localRoot, err := local.Child(namespace, "")
	if err != nil {
		return nil, err
	}
	if peerRoot.NodeHash == localRoot.NodeHash {
		return nil, nil
	}
	missing := make(digestSet)
	if err := walkMissing(local, peer, namespace, "", missing); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(missing))
	for id := range missing {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}

type digestSet map[string]struct{}

func walkMissing(local, peer *Index, namespace, prefix string, missing digestSet) error {
	peerNode, err := peer.Child(namespace, prefix)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	localNode, localErr := local.Child(namespace, prefix)
	localExists := localErr == nil
	if localErr != nil && !errors.Is(localErr, ErrNotFound) {
		return localErr
	}
	if localExists && localNode.NodeHash == peerNode.NodeHash {
		return nil
	}
	if len(peerNode.IDs) == 1 {
		id := peerNode.IDs[0]
		local.mu.RLock()
		namespaceOfID, exists := local.locations[id]
		_, quarantined := local.quarantine[id]
		local.mu.RUnlock()
		if quarantined {
			return refusal("integrity_failure", fmt.Errorf("%w: local identity %s is quarantined", ErrIntegrity, id))
		}
		if exists && namespaceOfID != namespace {
			return refusal("integrity_failure", fmt.Errorf("%w: peer identity %s belongs to local namespace %s", ErrIntegrity, id, namespaceOfID))
		}
		if !exists {
			missing[id] = struct{}{}
		}
		return nil
	}
	for _, child := range peerNode.Children {
		if err := walkMissing(local, peer, namespace, prefix+child.Label, missing); err != nil {
			return err
		}
	}
	return nil
}

// Root is the exact inventory.roots success member owned by this package.
type Root struct {
	Namespace string        `json:"namespace"`
	Count     scalar.Uint53 `json:"count"`
	RootID    scalar.Digest `json:"root_id"`
}

// WireObject is the JSON form of Section 11.3's WireObject.
type WireObject struct {
	ObjectID  string `json:"object_id"`
	MediaType string `json:"media_type"`
	Encoding  string `json:"encoding"`
	Data      string `json:"data"`
}

// ObjectSource serves one encoded objects.get request. Index is the in-process
// implementation; a later transport adapter can provide the same boundary.
type ObjectSource interface {
	ObjectsGet(line []byte, expectedNamespace string, maxLineBytes uint64) ([]byte, error)
}

// RequestIDFactory keeps time/randomness with the caller. This helper does not
// create clocks or network traffic.
type RequestIDFactory func() (string, error)

// FetchObjects requests one object per RPC call. With Section 11.3's 5 MiB
// object floor and 8 MiB line floor, singleton batches leave room for base64url
// expansion and framing. Every returned object is decoded and schema-checked
// again before it leaves this helper.
func FetchObjects(source ObjectSource, namespace string, ids []string, maxLineBytes uint64, nextRequestID RequestIDFactory) ([]WireObject, error) {
	if source == nil || !validNamespace(namespace) || namespace == "blob" || len(ids) < 1 || len(ids) > 4096 || maxLineBytes == 0 || maxLineBytes > rpcwire.MaxLineBytes || nextRequestID == nil {
		return nil, ErrInvalidRequest
	}
	for index, id := range ids {
		if _, err := scalar.ParseDigest(id); err != nil || (index > 0 && ids[index-1] >= id) {
			return nil, ErrInvalidRequest
		}
	}
	seenRequestIDs := map[string]struct{}{}
	objects := make([]WireObject, 0, len(ids))
	for _, id := range ids {
		requestID, err := nextRequestID()
		if err != nil {
			return nil, err
		}
		if _, duplicate := seenRequestIDs[requestID]; duplicate {
			return nil, refusal("integrity_failure", fmt.Errorf("%w: request ID factory repeated an ID", ErrIntegrity))
		}
		seenRequestIDs[requestID] = struct{}{}
		body, err := json.Marshal(struct {
			ObjectIDs []string `json:"object_ids"`
			Encodings []string `json:"encodings"`
		}{ObjectIDs: []string{id}, Encodings: []string{"json"}})
		if err != nil {
			return nil, err
		}
		line, err := rpcwire.EncodeRequest(RPC2Version, requestID, "objects.get", body)
		if err != nil {
			return nil, err
		}
		responseLine, err := source.ObjectsGet(line, namespace, maxLineBytes)
		if err != nil {
			return nil, err
		}
		request, err := rpcwire.DecodeRequest(line)
		if err != nil {
			return nil, err
		}
		response, err := rpcwire.DecodeResponse(responseLine, request)
		if err != nil || !response.OK() {
			return nil, ErrInvalidObject
		}
		wireObject, err := decodeOneWireObject(response.Body(), id, namespace)
		if err != nil {
			return nil, err
		}
		objects = append(objects, wireObject)
	}
	return objects, nil
}

func decodeOneWireObject(data []byte, expectedID, expectedNamespace string) (WireObject, error) {
	outer, err := strictObject(data)
	if err != nil || !exactMembers(outer, "objects") {
		return WireObject{}, ErrInvalidObject
	}
	rawObjects, err := strictArray(outer["objects"])
	if err != nil || len(rawObjects) != 1 {
		return WireObject{}, ErrInvalidObject
	}
	inner, err := strictObject(rawObjects[0])
	if err != nil || !exactMembers(inner, "object_id", "media_type", "encoding", "data") {
		return WireObject{}, ErrInvalidObject
	}
	objectID, objectIDOK := requiredJSONString(inner, "object_id")
	mediaType, mediaTypeOK := requiredJSONString(inner, "media_type")
	encoding, encodingOK := requiredJSONString(inner, "encoding")
	encodedData, dataOK := requiredJSONString(inner, "data")
	if !objectIDOK || !mediaTypeOK || !encodingOK || !dataOK || objectID != expectedID || mediaType != "application/json" || encoding != "json" {
		return WireObject{}, ErrInvalidObject
	}
	dataBytes, err := base64.RawURLEncoding.Strict().DecodeString(encodedData)
	if err != nil || base64URL(dataBytes) != encodedData {
		return WireObject{}, ErrInvalidObject
	}
	membership, err := ClassifyJSON(dataBytes)
	if err != nil || membership.Excluded || membership.Namespace != expectedNamespace || membership.ID.String() != expectedID {
		return WireObject{}, refusal("integrity_failure", fmt.Errorf("%w: returned object does not belong to requested namespace", ErrIntegrity))
	}
	return WireObject{ObjectID: objectID, MediaType: mediaType, Encoding: encoding, Data: encodedData}, nil
}

// DispatchContext supplies values that §11.3 deliberately keeps outside the
// objects.get operation body.
type DispatchContext struct {
	Namespace    string
	MaxLineBytes uint64
}

// ObjectsGet serves one exact objects.get request in the contextual namespace
// whose differing trie node caused the exchange. Blob bytes use Section 11.5
// transfer operations and are never returned by this method.
func (index *Index) ObjectsGet(line []byte, expectedNamespace string, maxLineBytes uint64) ([]byte, error) {
	return index.Dispatch(line, DispatchContext{Namespace: expectedNamespace, MaxLineBytes: maxLineBytes})
}

func (index *Index) objectsGetRequest(request rpcwire.Request, context DispatchContext) ([]byte, error) {
	if request.Version() != index.version {
		return nil, ErrUnsupportedVersion
	}
	if !validNamespace(context.Namespace) {
		return nil, ErrInvalidRequest
	}
	if context.MaxLineBytes == 0 || context.MaxLineBytes > rpcwire.MaxLineBytes {
		return nil, ErrInvalidRequest
	}
	ids, encodings, err := decodeObjectsGetRequest(request.Body())
	if err != nil {
		return nil, err
	}
	if !slices.Contains(encodings, "json") {
		return nil, ErrInvalidRequest
	}
	objects := make([]WireObject, 0, len(ids))
	index.mu.RLock()
	for _, id := range ids {
		if namespace, found := index.locations[id]; found && namespace != context.Namespace {
			index.mu.RUnlock()
			return nil, refusal("integrity_failure", fmt.Errorf("%w: object %s belongs to %s, requested under %s", ErrIntegrity, id, namespace, context.Namespace))
		}
		stored, found := index.objects[context.Namespace][id]
		if !found || !stored.json || len(stored.data) == 0 {
			index.mu.RUnlock()
			return nil, refusal("not_found", ErrNotFound)
		}
		membership, verifyErr := ClassifyJSON(stored.data)
		if verifyErr != nil || membership.Excluded || membership.Namespace != context.Namespace || membership.ID.String() != id {
			index.mu.RUnlock()
			return nil, refusal("integrity_failure", fmt.Errorf("%w: stored object failed schema-to-namespace validation", ErrIntegrity))
		}
		objects = append(objects, WireObject{ObjectID: id, MediaType: "application/json", Encoding: "json", Data: base64URL(stored.data)})
	}
	index.mu.RUnlock()
	body, err := json.Marshal(struct {
		Objects []WireObject `json:"objects"`
	}{Objects: objects})
	if err != nil {
		return nil, err
	}
	lineOut, err := rpcwire.EncodeSuccess(request, body)
	if err != nil {
		if errors.Is(err, rpcwire.ErrFrame) {
			return nil, refusal("incompatible_protocol", ErrResponseTooLarge)
		}
		return nil, err
	}
	if uint64(len(lineOut)) > context.MaxLineBytes {
		return nil, refusal("incompatible_protocol", ErrResponseTooLarge)
	}
	return lineOut, nil
}

func base64URL(data []byte) string { return base64.RawURLEncoding.EncodeToString(data) }

func decodeObjectsGetRequest(data []byte) ([]string, []string, error) {
	object, err := strictObject(data)
	if err != nil || !exactMembers(object, "object_ids", "encodings") {
		return nil, nil, ErrInvalidRequest
	}
	objectIDs, ok := requiredStringArray(object, "object_ids")
	if !ok {
		return nil, nil, ErrInvalidRequest
	}
	encodings, ok := requiredStringArray(object, "encodings")
	if !ok {
		return nil, nil, ErrInvalidRequest
	}
	if len(objectIDs) < 1 || len(objectIDs) > 4096 || len(encodings) < 1 || len(encodings) > 2 {
		return nil, nil, ErrInvalidRequest
	}
	for i, raw := range objectIDs {
		if _, err := scalar.ParseDigest(raw); err != nil || (i > 0 && objectIDs[i-1] >= raw) {
			return nil, nil, ErrInvalidRequest
		}
	}
	for i, encoding := range encodings {
		if encoding != "cbor" && encoding != "json" || (i > 0 && encodings[i-1] >= encoding) {
			return nil, nil, ErrInvalidRequest
		}
	}
	return objectIDs, encodings, nil
}
