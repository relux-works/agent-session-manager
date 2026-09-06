package dirnode

import (
	"encoding/base64"
	"encoding/json"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the Section 7.9 scan operation framing and
// owns the (operation, operation_id) idempotency the scan and
// enrichment-run mutations use: repeating the same canonical body
// returns the prior durable result, and a changed body under a
// recorded key is an idempotency_mismatch without new records.
//
// Scan reads only declared non-auth roots, publishes observations
// and its Inventory Batch atomically, and never claims that an
// active prefix is a cloning-safe boundary. The InventoryBatch
// content crosses as a present object: its content owner is the
// shared-environment leaf of this Story, so CheckScanResponse
// requires the object shape and never inspects further.

// scanRequestMembers is the exact scan request-body member set.
var scanRequestMembers = map[string]bool{
	"operation_id":     true,
	"installation_ids": true,
	"prior_batch_id":   true,
	"cursor":           true,
	"max_instances":    true,
	"extensions":       true,
}

// scanRequestRequired lists scanRequestMembers in a fixed order.
var scanRequestRequired = []string{
	"operation_id",
	"installation_ids",
	"prior_batch_id",
	"cursor",
	"max_instances",
	"extensions",
}

// ScanRequest is one validated scan request body.
type ScanRequest struct {
	OperationID   string
	Installations []string
	HasPriorBatch bool
	PriorBatchID  string
	HasCursor     bool
	Cursor        string
	MaxInstances  uint64
}

// CheckScanRequest validates one scan-operation request body: the
// exact closed members, a UUIDv7 operation identifier, sorted
// unique installation digests in the count bound, a digest|null
// prior batch, a string[1..4096]|null cursor, and max_instances in
// uint53[1..65536]. Null is the only accepted absence for the
// nullable members: a missing-typed value is not an absence.
func CheckScanRequest(body []byte) (ScanRequest, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(body))
	if fault != nil {
		failure, err := failViolation("scan request "+fault.detail, fault.member)
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	if name, unknown := unknownMember(members, scanRequestMembers); unknown {
		failure, err := failViolation("scan request carries unknown member", name)
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	if name, missing := missingMember(members, scanRequestRequired); missing {
		failure, err := failViolation("scan request misses a required member", name)
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	operationID, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		failure, err := failViolation("scan request operation identifier is not a UUIDv7", "operation_id")
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	installations, ok := checkSortedUniqueDigests(members["installation_ids"], 1, 256)
	if !ok {
		failure, err := failViolation("scan request installations are not sorted unique digest[1..256]", "installation_ids")
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	prior, present, wasNull := checkOptionalDigest(members["prior_batch_id"])
	if !present {
		failure, err := failViolation("scan request prior batch is not a digest|null", "prior_batch_id")
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	cursor, cursorPresent, cursorNull := checkOptionalString(members["cursor"], 1, 4096)
	if !cursorPresent {
		failure, err := failViolation("scan request cursor is not a string[1..4096]|null", "cursor")
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	maximum, ok := checkUint53Bounds(members["max_instances"], 1, 65536)
	if !ok {
		failure, err := failViolation("scan request max_instances is not uint53[1..65536]", "max_instances")
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failViolation("scan request extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return ScanRequest{}, err
		}
		return ScanRequest{}, failure
	}
	return ScanRequest{
		OperationID:   operationID.String(),
		Installations: digestStrings(installations),
		HasPriorBatch: !wasNull,
		PriorBatchID:  prior.String(),
		HasCursor:     !cursorNull,
		Cursor:        cursor,
		MaxInstances:  maximum,
	}, nil
}

// scanResponseMembers is the exact scan success-body member set.
var scanResponseMembers = map[string]bool{
	"batch":                       true,
	"environment_observation_ids": true,
	"native_observation_ids":      true,
	"next_cursor":                 true,
	"extensions":                  true,
}

// scanResponseRequired lists scanResponseMembers in a fixed order.
var scanResponseRequired = []string{
	"batch",
	"environment_observation_ids",
	"native_observation_ids",
	"next_cursor",
	"extensions",
}

// ScanResponse is one validated scan success body. The batch
// crosses as presence: the InventoryBatch content owner is the
// shared-environment leaf, so only the object shape is checked.
type ScanResponse struct {
	HasBatch       bool
	EnvironmentIDs []string
	NativeIDs      []string
	HasNextCursor  bool
	NextCursor     string
}

// CheckScanResponse validates one scan-operation success body: the
// exact closed members, a present batch object, sorted unique
// environment observation digests in [1..256], sorted unique
// native observation digests in [0..65536], and a
// string[1..4096]|null next cursor.
func CheckScanResponse(body []byte) (ScanResponse, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(body))
	if fault != nil {
		failure, err := failViolation("scan response "+fault.detail, fault.member)
		if err != nil {
			return ScanResponse{}, err
		}
		return ScanResponse{}, failure
	}
	if name, unknown := unknownMember(members, scanResponseMembers); unknown {
		failure, err := failViolation("scan response carries unknown member", name)
		if err != nil {
			return ScanResponse{}, err
		}
		return ScanResponse{}, failure
	}
	if name, missing := missingMember(members, scanResponseRequired); missing {
		failure, err := failViolation("scan response misses a required member", name)
		if err != nil {
			return ScanResponse{}, err
		}
		return ScanResponse{}, failure
	}
	if _, fault := decodeStrictObject(bytesTrimSpace(members["batch"])); fault != nil {
		failure, err := failViolation("scan response batch is not an object", "batch")
		if err != nil {
			return ScanResponse{}, err
		}
		return ScanResponse{}, failure
	}
	environments, ok := checkSortedUniqueDigests(members["environment_observation_ids"], 1, 256)
	if !ok {
		failure, err := failViolation("scan response environment observations are not sorted unique digest[1..256]", "environment_observation_ids")
		if err != nil {
			return ScanResponse{}, err
		}
		return ScanResponse{}, failure
	}
	natives, ok := checkSortedUniqueDigests(members["native_observation_ids"], 0, 65536)
	if !ok {
		failure, err := failViolation("scan response native observations are not sorted unique digest[0..65536]", "native_observation_ids")
		if err != nil {
			return ScanResponse{}, err
		}
		return ScanResponse{}, failure
	}
	cursor, present, wasNull := checkOptionalString(members["next_cursor"], 1, 4096)
	if !present {
		failure, err := failViolation("scan response next cursor is not a string[1..4096]|null", "next_cursor")
		if err != nil {
			return ScanResponse{}, err
		}
		return ScanResponse{}, failure
	}
	if !checkExtensions(members["extensions"]) {
		failure, err := failViolation("scan response extensions are not reverse-DNS keyed", "extensions")
		if err != nil {
			return ScanResponse{}, err
		}
		return ScanResponse{}, failure
	}
	return ScanResponse{
		HasBatch:       true,
		EnvironmentIDs: digestStrings(environments),
		NativeIDs:      digestStrings(natives),
		HasNextCursor:  !wasNull,
		NextCursor:     cursor,
	}, nil
}

// journalRecord is one durable (operation, operation_id) entry: the
// digest of the canonical request body that produced it and the
// canonical result bytes. The result is stored, never recomputed:
// a retry after a lost response replays the recorded bytes.
type journalRecord struct {
	BodyDigest string `json:"body_digest"`
	Result     []byte `json:"result"`
}

// Journal is the host-side (operation, operation_id) idempotency
// record for the mutating directory operations (scan and
// enrichment-run). Records are keyed by operation and operation
// identifier and bind the canonical body digest: a repeat of the
// identical body replays the prior result with no new records, and
// a changed body under a recorded key is refused. The journal
// exports and imports its bytes so a restarted host recovers the
// same record without trusting a peer claim.
type Journal struct {
	records map[string]journalRecord
}

// NewJournal returns an empty idempotency journal.
func NewJournal() *Journal {
	return &Journal{records: map[string]journalRecord{}}
}

// journalKey binds the idempotency scope: one operation's
// operation_id never authorizes another operation's retry.
func journalKey(operation Operation, operationID string) string {
	return string(operation) + "\x00" + operationID
}

// CheckAndRecord applies the idempotency rule for one mutating
// call. canonicalBody must be the JCS canonical request body the
// host sent; result is the canonical result bytes to durably
// record. The first call for a key records and reports stored;
// a repeat with the identical body digest replays the recorded
// result; a changed body under a recorded key is an
// idempotency_mismatch and records nothing. An operation outside
// the closed registry or an identifier outside UUIDv7 is a caller
// error, never a key.
func (journal *Journal) CheckAndRecord(operation Operation, operationID string, canonicalBody []byte, result []byte) (stored bool, prior []byte, err error) {
	if !validOperation(string(operation)) {
		failure, faultErr := failUnknownOperation("idempotency key names an operation outside the closed registry", string(operation))
		if faultErr != nil {
			return false, nil, faultErr
		}
		return false, nil, failure
	}
	identifier, parseErr := scalar.ParseUUIDv7(operationID)
	if parseErr != nil {
		failure, faultErr := failInvalid("idempotency operation identifier is not a UUIDv7", "operation_id")
		if faultErr != nil {
			return false, nil, faultErr
		}
		return false, nil, failure
	}
	canonical, canonErr := canonicaljson.Canonicalize(bytesTrimSpace(canonicalBody))
	if canonErr != nil {
		failure, faultErr := failViolation("idempotency body is not canonical JSON", "body")
		if faultErr != nil {
			return false, nil, faultErr
		}
		return false, nil, failure
	}
	digest := scalar.SHA256Digest(canonical).String()
	key := journalKey(operation, identifier.String())
	if record, recorded := journal.records[key]; recorded {
		if record.BodyDigest != digest {
			failure, faultErr := failIdempotency("idempotent body changed under a recorded operation identifier", string(operation))
			if faultErr != nil {
				return false, nil, faultErr
			}
			return false, nil, failure
		}
		return false, append([]byte(nil), record.Result...), nil
	}
	storedResult := append([]byte(nil), result...)
	journal.records[key] = journalRecord{BodyDigest: digest, Result: storedResult}
	return true, nil, nil
}

// journalDocument is the exported journal shape: versioned, with
// records sorted by key so the bytes are deterministic.
type journalDocument struct {
	Schema  string                  `json:"schema"`
	Version int                     `json:"version"`
	Records []journalDocumentRecord `json:"records"`
}

// journalDocumentRecord is one exported entry.
type journalDocumentRecord struct {
	Key        string `json:"key"`
	BodyDigest string `json:"body_digest"`
	Result     []byte `json:"result"`
}

// journalExportSchema pins the export envelope identity.
const journalExportSchema = "urn:ax:internal:dirnode-journal"

// Export serializes the journal for durable placement. Records
// emit sorted by key so identical journals export identical bytes.
func (journal *Journal) Export() ([]byte, error) {
	keys := make([]string, 0, len(journal.records))
	for key := range journal.records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	document := journalDocument{Schema: journalExportSchema, Version: 1}
	for _, key := range keys {
		record := journal.records[key]
		document.Records = append(document.Records, journalDocumentRecord{
			Key:        key,
			BodyDigest: record.BodyDigest,
			Result:     append([]byte(nil), record.Result...),
		})
	}
	encoded, err := canonicaljson.Canonicalize(mustMarshalJSON(document))
	if err != nil {
		failure, faultErr := failIntegrity("idempotency journal is not exportable", "journal")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	return encoded, nil
}

// mustMarshalJSON encodes the export document. The document is
// built from validated in-memory records only, so encoding cannot
// fail; a failure is a host defect, not a refusal.
func mustMarshalJSON(document journalDocument) []byte {
	encoded, err := json.Marshal(document)
	if err != nil {
		panic("dirnode: journal document is not encodable: " + err.Error())
	}
	return encoded
}

// Import replaces the journal contents from exported bytes: a
// restarted host recovers the same record without trusting a peer
// claim. The bytes must be one complete object with the exact
// export schema and version, sorted unique keys, digest-shaped
// body bindings, and result values that are strings. A failed or
// partial read is an integrity failure, never an empty journal
// that would authorize fresh records under recorded keys.
func (journal *Journal) Import(data []byte) error {
	members, fault := decodeStrictObject(bytesTrimSpace(data))
	if fault != nil {
		failure, err := failIntegrity("idempotency journal "+fault.detail, "journal")
		if err != nil {
			return err
		}
		return failure
	}
	schema, ok := rawString(members["schema"])
	if !ok || schema != journalExportSchema {
		failure, err := failIntegrity("idempotency journal schema is not the journal export", "schema")
		if err != nil {
			return err
		}
		return failure
	}
	version, ok := rawUint53(members["version"])
	if !ok || version != 1 {
		failure, err := failIntegrity("idempotency journal version is not 1", "version")
		if err != nil {
			return err
		}
		return failure
	}
	elements, ok := decodeArray(members["records"])
	if !ok {
		failure, err := failIntegrity("idempotency journal records are not an array", "records")
		if err != nil {
			return err
		}
		return failure
	}
	records := make(map[string]journalRecord, len(elements))
	var previous string
	first := true
	for _, element := range elements {
		record, ok := decodeJournalRecord(element)
		if !ok {
			failure, err := failIntegrity("idempotency journal record is not a keyed digest binding", "records")
			if err != nil {
				return err
			}
			return failure
		}
		if !first && previous >= record.Key {
			failure, err := failIntegrity("idempotency journal records are not sorted unique", "records")
			if err != nil {
				return err
			}
			return failure
		}
		first = false
		previous = record.Key
		records[record.Key] = journalRecord{BodyDigest: record.BodyDigest, Result: record.Result}
	}
	journal.records = records
	return nil
}

// decodedJournalRecord is one validated export entry.
type decodedJournalRecord struct {
	Key        string
	BodyDigest string
	Result     []byte
}

// decodeJournalRecord validates one export entry: a closed keyed
// digest binding with a string result.
func decodeJournalRecord(raw json.RawMessage) (decodedJournalRecord, bool) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		return decodedJournalRecord{}, false
	}
	allowed := map[string]bool{"key": true, "body_digest": true, "result": true}
	if name, unknown := unknownMember(members, allowed); unknown {
		_ = name
		return decodedJournalRecord{}, false
	}
	required := []string{"key", "body_digest", "result"}
	if name, missing := missingMember(members, required); missing {
		_ = name
		return decodedJournalRecord{}, false
	}
	key, ok := checkStringBounds(members["key"], 1, 512)
	if !ok {
		return decodedJournalRecord{}, false
	}
	digest, ok := rawString(members["body_digest"])
	if !ok {
		return decodedJournalRecord{}, false
	}
	if _, ok := checkDigestString(digest); !ok {
		return decodedJournalRecord{}, false
	}
	encoded, ok := rawString(members["result"])
	if !ok {
		return decodedJournalRecord{}, false
	}
	result, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return decodedJournalRecord{}, false
	}
	return decodedJournalRecord{Key: key, BodyDigest: digest, Result: result}, true
}
