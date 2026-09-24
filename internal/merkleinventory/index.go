package merkleinventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	RPC2Version      = "2.0.0"
	objectSchema     = "schema"
	blobDescriptorV1 = "urn:ax:schema:blob"
)

type disposition uint8

const (
	included disposition = iota
	excluded
)

// Membership describes the validated identity class of one immutable JSON
// object. Excluded objects have no ID and cannot affect a namespace root.
type Membership struct {
	Namespace string
	ID        scalar.Digest
	Schema    string
	Version   string
	Excluded  bool
}

type storedObject struct {
	data []byte
	json bool
}

// Index is one in-memory immutable-object inventory snapshot for Mesh RPC 2.
// Calls are concurrency-safe. DurableIndex supplies persistence and in-process
// object union while this type remains the shared trie/admission engine.
type Index struct {
	version    string
	mu         sync.RWMutex
	objects    map[string]map[string]storedObject
	locations  map[string]string
	quarantine map[string][][]byte
}

// New accepts only the historical RPC 2 inventory. Later majors have
// additional namespace membership and authority rules and fail closed here.
func New(version string) (*Index, error) {
	namespaces, err := rpcwire.Namespaces(version)
	if err != nil {
		return nil, err
	}
	if version != RPC2Version || !slices.Equal(namespaces, rpc2Namespaces) {
		return nil, ErrUnsupportedVersion
	}
	objects := make(map[string]map[string]storedObject, len(namespaces))
	for _, namespace := range namespaces {
		objects[namespace] = map[string]storedObject{}
	}
	return &Index{
		version:    version,
		objects:    objects,
		locations:  map[string]string{},
		quarantine: map[string][][]byte{},
	}, nil
}

// ClassifyJSON validates an immutable object's closed shape and omit-self
// digest through canonicaljson before assigning its single RPC 2 namespace.
// Known local/transient schemas and objects without an immutable schema are
// explicitly excluded. Unknown schema identities fail closed.
func ClassifyJSON(data []byte) (Membership, error) {
	object, err := strictObject(data)
	if err != nil {
		return Membership{}, fmt.Errorf("%w: %v", ErrInvalidObject, err)
	}
	schemaRaw, hasSchema := object[objectSchema]
	if !hasSchema {
		return Membership{Excluded: true}, nil
	}
	schema, schemaOK := strictJSONString(schemaRaw)
	if !schemaOK || schema == "" {
		return Membership{}, refusal("integrity_failure", fmt.Errorf("%w: schema must be a non-empty string", ErrInvalidObject))
	}
	if isExcludedSchema(schema) {
		return Membership{Schema: schema, Excluded: true}, nil
	}
	version, versionOK := requiredJSONString(object, "schema_version")
	if !versionOK || version == "" {
		return Membership{}, refusal("integrity_failure", fmt.Errorf("%w: schema_version must be a non-empty string", ErrInvalidObject))
	}
	namespace, supported := rpc2SchemaNamespaces[schemaVersion{schema, version}]
	if !supported {
		return Membership{}, refusal("integrity_failure", fmt.Errorf("%w: unsupported schema class %s@%s", ErrInvalidObject, schema, version))
	}
	digest, selfField, err := canonicaljson.VerifyObjectIdentity(data)
	if err != nil {
		return Membership{}, refusal("integrity_failure", fmt.Errorf("%w: %v", ErrInvalidObject, err))
	}
	wantField, known := schemaSelfFields[schema]
	if !known || string(selfField) != wantField {
		return Membership{}, refusal("integrity_failure", fmt.Errorf("%w: schema %s has unexpected self field %q", ErrInvalidObject, schema, selfField))
	}
	return Membership{Namespace: namespace, ID: digest, Schema: schema, Version: version}, nil
}

// AddJSON admits one schema-valid immutable JSON object. Repeating identical
// bytes is idempotent. A same-ID byte conflict removes the ID from inventory,
// retains both byte strings in quarantine, and returns integrity_failure.
func (index *Index) AddJSON(data []byte) error {
	return index.addJSON(data, true)
}

// AddUnionJSON admits one validated object into an exchange set. Unlike
// AddJSON, it defers Tombstone/Acknowledgement link closure until the complete
// union is present; callers close that set with ValidateUnionClosure.
func (index *Index) AddUnionJSON(data []byte) error {
	return index.addJSON(data, false)
}

// ValidateUnionClosure verifies deferred Tombstone/Acknowledgement links for
// a process-local union before projection or handoff.
func (index *Index) ValidateUnionClosure() error {
	if index == nil {
		return ErrInvalidRequest
	}
	return index.validateTombstoneAckClosure()
}

// addJSON admits schema- and digest-valid JSON into an inventory snapshot.
// The durable union defers Tombstone/Acknowledgement cross-object closure
// until its complete received set is available; ordinary local ingestion
// keeps the historical immediate-reference requirement.
func (index *Index) addJSON(data []byte, requireTombstone bool) error {
	membership, err := ClassifyJSON(data)
	if err != nil {
		return err
	}
	if membership.Excluded {
		return ErrExcluded
	}
	index.mu.Lock()
	defer index.mu.Unlock()
	if requireTombstone {
		if err := index.validateTombstoneAckLinkLocked(membership, data); err != nil {
			return err
		}
	}
	return index.insertLocked(membership.Namespace, membership.ID.String(), data, true)
}

func (index *Index) validateTombstoneAckLinkLocked(membership Membership, data []byte) error {
	if membership.Schema != "urn:ax:schema:tombstone-ack" {
		return nil
	}
	acknowledgement, err := strictObject(data)
	if err != nil {
		return refusal("integrity_failure", err)
	}
	subjectID, subjectOK := requiredJSONString(acknowledgement, "subject_id")
	tombstoneID, tombstoneOK := requiredJSONString(acknowledgement, "tombstone_id")
	if !subjectOK || !tombstoneOK {
		return refusal("integrity_failure", fmt.Errorf("%w: Tombstone Acknowledgement link fields are invalid", ErrInvalidObject))
	}
	tombstone, exists := index.objects["tombstone"][tombstoneID]
	if !exists {
		return refusal("integrity_failure", fmt.Errorf("%w: Tombstone Acknowledgement references a Tombstone not admitted to this inventory", ErrInvalidObject))
	}
	referenced, err := strictObject(tombstone.data)
	if err != nil {
		return refusal("integrity_failure", fmt.Errorf("%w: stored Tombstone cannot be decoded", ErrIntegrity))
	}
	referencedSubjectID, referencedOK := requiredJSONString(referenced, "subject_id")
	if !referencedOK {
		return refusal("integrity_failure", fmt.Errorf("%w: stored Tombstone subject_id cannot be decoded", ErrIntegrity))
	}
	if referencedSubjectID != subjectID {
		return refusal("integrity_failure", fmt.Errorf("%w: Tombstone Acknowledgement subject_id does not match its Tombstone", ErrInvalidObject))
	}
	return nil
}

// AddBlob admits complete raw content only with its validated Blob Descriptor.
// Chunks are staged by transfer operations and have no API into this index.
func (index *Index) AddBlob(descriptorJSON, raw []byte) error {
	membership, err := ClassifyJSON(descriptorJSON)
	if err != nil {
		return err
	}
	if membership.Namespace != "manifest" || membership.Schema != blobDescriptorV1 {
		return refusal("integrity_failure", fmt.Errorf("%w: raw blobs require a Blob Descriptor", ErrInvalidObject))
	}
	descriptor, err := strictObject(descriptorJSON)
	if err != nil {
		return err
	}
	blobID, ok := requiredJSONString(descriptor, "blob_id")
	if !ok {
		return refusal("integrity_failure", fmt.Errorf("%w: Blob Descriptor blob_id must be a string", ErrInvalidObject))
	}
	parsed, err := scalar.ParseDigest(blobID)
	if err != nil || scalar.SHA256Digest(raw).String() != parsed.String() {
		return refusal("integrity_failure", fmt.Errorf("%w: Blob Descriptor blob_id does not match raw bytes", ErrInvalidObject))
	}
	if err := verifyDescriptorBytes(descriptor, raw); err != nil {
		return refusal("integrity_failure", err)
	}
	index.mu.Lock()
	defer index.mu.Unlock()
	if membership.ID.String() == parsed.String() {
		index.quarantine[membership.ID.String()] = append(index.quarantine[membership.ID.String()], bytes.Clone(descriptorJSON), bytes.Clone(raw))
		return refusal("integrity_failure", fmt.Errorf("%w: descriptor and raw blob share one identity", ErrIntegrity))
	}
	if err := index.preflightLocked(membership.Namespace, membership.ID.String(), descriptorJSON); err != nil {
		return err
	}
	if err := index.preflightLocked("blob", parsed.String(), raw); err != nil {
		return err
	}
	index.putLocked(membership.Namespace, membership.ID.String(), descriptorJSON, true)
	index.putLocked("blob", parsed.String(), raw, false)
	return nil
}

func (index *Index) insertLocked(namespace, id string, data []byte, isJSON bool) error {
	if err := index.preflightLocked(namespace, id, data); err != nil {
		return err
	}
	index.putLocked(namespace, id, data, isJSON)
	return nil
}

func (index *Index) preflightLocked(namespace, id string, data []byte) error {
	if !validNamespace(namespace) {
		return ErrInvalidRequest
	}
	if _, quarantined := index.quarantine[id]; quarantined {
		index.quarantine[id] = append(index.quarantine[id], bytes.Clone(data))
		return refusal("integrity_failure", fmt.Errorf("%w: identity is already quarantined", ErrIntegrity))
	}
	if prior, exists := index.locations[id]; exists && prior != namespace {
		return index.quarantineLocked(id, index.objects[prior][id].data, data)
	}
	if previous, exists := index.objects[namespace][id]; exists {
		if bytes.Equal(previous.data, data) {
			return nil
		}
		return index.quarantineLocked(id, previous.data, data)
	}
	return nil
}

func (index *Index) putLocked(namespace, id string, data []byte, isJSON bool) {
	if _, exists := index.objects[namespace][id]; exists {
		return
	}
	index.objects[namespace][id] = storedObject{data: bytes.Clone(data), json: isJSON}
	index.locations[id] = namespace
}

func (index *Index) quarantineLocked(id string, previous, incoming []byte) error {
	if priorNamespace, exists := index.locations[id]; exists {
		delete(index.objects[priorNamespace], id)
		delete(index.locations, id)
	}
	index.quarantine[id] = append(index.quarantine[id], bytes.Clone(previous), bytes.Clone(incoming))
	return refusal("integrity_failure", fmt.Errorf("%w: same digest has different bytes", ErrIntegrity))
}

// QuarantinedIDs returns sorted IDs whose same-digest byte conflicts aborted
// admission. It exposes no quarantined bytes.
func (index *Index) QuarantinedIDs() []string {
	index.mu.RLock()
	defer index.mu.RUnlock()
	ids := make([]string, 0, len(index.quarantine))
	for id := range index.quarantine {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func verifyDescriptorBytes(descriptor map[string]json.RawMessage, raw []byte) error {
	totalSize, sizeOK := requiredUint53(descriptor, "size")
	if !sizeOK || totalSize.Uint64() != uint64(len(raw)) {
		return fmt.Errorf("%w: Blob Descriptor size does not match raw bytes", ErrInvalidObject)
	}
	chunksRaw, ok := descriptor["chunks"]
	if !ok {
		return fmt.Errorf("%w: Blob Descriptor chunks are invalid", ErrInvalidObject)
	}
	chunks, err := strictArray(chunksRaw)
	if err != nil {
		return fmt.Errorf("%w: Blob Descriptor chunks are invalid", ErrInvalidObject)
	}
	for _, rawChunk := range chunks {
		chunk, err := strictObject(rawChunk)
		if err != nil || !exactMembers(chunk, "chunk_id", "index", "offset", "size") {
			return fmt.Errorf("%w: BlobChunk is invalid", ErrInvalidObject)
		}
		chunkID, chunkIDOK := requiredJSONString(chunk, "chunk_id")
		_, indexOK := requiredUint53(chunk, "index")
		offset, offsetOK := requiredUint53(chunk, "offset")
		size, chunkSizeOK := requiredUint53(chunk, "size")
		if !chunkIDOK || !indexOK || !offsetOK || !chunkSizeOK {
			return fmt.Errorf("%w: BlobChunk is invalid", ErrInvalidObject)
		}
		end := offset.Uint64() + size.Uint64()
		if end > uint64(len(raw)) || offset.Uint64() > end {
			return fmt.Errorf("%w: BlobChunk lies outside raw bytes", ErrInvalidObject)
		}
		parsedChunkID, err := scalar.ParseDigest(chunkID)
		if err != nil || scalar.SHA256Digest(raw[offset.Uint64():end]).String() != parsedChunkID.String() {
			return fmt.Errorf("%w: BlobChunk chunk_id does not match raw bytes", ErrInvalidObject)
		}
	}
	return nil
}

type schemaVersion struct{ schema, version string }

var rpc2SchemaNamespaces = map[schemaVersion]string{
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

var schemaSelfFields = map[string]string{
	"urn:ax:schema:session-record":       "record_id",
	"urn:ax:schema:session-event":        "event_id",
	"urn:ax:schema:lease":                "record_id",
	"urn:ax:schema:checkpoint":           "checkpoint_id",
	"urn:ax:schema:workspace-group":      "record_id",
	"urn:ax:schema:provider-identity":    "record_id",
	"urn:ax:schema:transfer-manifest":    "manifest_id",
	blobDescriptorV1:                     "descriptor_id",
	"urn:ax:schema:materialization-plan": "plan_id",
	"urn:ax:schema:task-board-bundle":    "bundle_id",
	"urn:ax:schema:tombstone":            "tombstone_id",
	"urn:ax:schema:tombstone-ack":        "ack_id",
}

var excludedSchemas = map[string]struct{}{
	"urn:ax:schema:config":                              {},
	"urn:ax:schema:provider-manifest":                   {},
	"urn:ax:schema:provider-probe":                      {},
	"urn:ax:schema:terminal-backend-manifest":           {},
	"urn:ax:schema:terminal-backend-probe":              {},
	"urn:ax:schema:chunk":                               {},
	"urn:ax:schema:canonical-event":                     {},
	"urn:ax:schema:materialization-journal":             {},
	"urn:ax:schema:terminal-instance-binding":           {},
	"urn:ax:schema:session-adapter-manifest":            {},
	"urn:ax:schema:session-adapter-probe":               {},
	"urn:ax:schema:session-directory-node-manifest":     {},
	"urn:ax:schema:session-directory-node-request":      {},
	"urn:ax:schema:session-directory-node-response":     {},
	"urn:ax:schema:host-trust-store":                    {},
	"urn:ax:schema:launch-plan-request":                 {},
	"urn:ax:schema:error":                               {},
	"urn:ax:schema:observation":                         {},
	"urn:ax:schema:cli-result":                          {},
	"urn:ax:schema:session-clone-bundle":                {},
	"urn:ax:schema:clone-raw-object-manifest":           {},
	"urn:ax:schema:clone-capture-manifest":              {},
	"urn:ax:schema:canonical-session":                   {},
	"urn:ax:schema:migration-checkpoint":                {},
	"urn:ax:schema:fidelity-report":                     {},
	"urn:ax:schema:projection-plan":                     {},
	"urn:ax:schema:clone-projected-object-manifest":     {},
	"urn:ax:schema:clone-read-back-evidence-manifest":   {},
	"urn:ax:schema:clone-validation-report":             {},
	"urn:ax:schema:clone-lineage-receipt":               {},
	"urn:ax:schema:supported-environment-tuples":        {},
	"urn:ax:schema:environment-observation":             {},
	"urn:ax:schema:native-session-observation":          {},
	"urn:ax:schema:session-inventory-batch":             {},
	"urn:ax:schema:conversation-lineage-link":           {},
	"urn:ax:schema:session-annotation":                  {},
	"urn:ax:schema:session-enrichment-profile":          {},
	"urn:ax:schema:session-enrichment-job-request":      {},
	"urn:ax:schema:session-enrichment-job-receipt":      {},
	"urn:ax:schema:session-continuation-plan":           {},
	"urn:ax:schema:session-directory-operation-receipt": {},
	"urn:ax:schema:session-directory-query":             {},
}

func isExcludedSchema(schema string) bool {
	_, excluded := excludedSchemas[schema]
	return excluded
}

// MembershipOf returns the currently indexed namespace for a digest, if any.
func (index *Index) MembershipOf(id scalar.Digest) (string, bool) {
	index.mu.RLock()
	defer index.mu.RUnlock()
	namespace, ok := index.locations[id.String()]
	return namespace, ok
}
