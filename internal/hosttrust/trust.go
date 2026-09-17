package hosttrust

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	// TrustSchemaID is the exact Host Trust Store schema identity.
	TrustSchemaID = "urn:ax:schema:host-trust-store"
	// TrustSchemaVersion is the only accepted trust store version.
	TrustSchemaVersion = "1.0.0"
	// MaxGeneration is the uint53 ceiling. A committed store at this
	// generation refuses every further mutation.
	MaxGeneration = uint64(1<<53 - 1)
	// MaxEntries bounds CredentialEntry[0..4096].
	MaxEntries = 4096
	// MaxDERBytes bounds each decoded leaf_der/root_der payload.
	MaxDERBytes = 16384
)

const (
	// EntryActive admits the credential for new streams.
	EntryActive = "active"
	// EntryRetiring admits only pre-rotation streams until retire_at.
	EntryRetiring = "retiring"
	// EntryRevoked forbids all future admissions. Public bytes are retained
	// permanently as a reuse tombstone.
	EntryRevoked = "revoked"
)

// CredentialEntry is one enrolled credential. retire_at is non-null exactly
// for retiring entries and later than enrolled_at.
type CredentialEntry struct {
	HostID       scalar.UUIDv7
	CredentialID scalar.Digest
	SPKIID       scalar.Digest
	RootID       scalar.Digest
	LeafDER      []byte
	RootDER      []byte
	State        string
	EnrolledAt   scalar.Timestamp
	RetireAt     *scalar.Timestamp
}

// TrustStore is the decoded Host Trust Store 1.0.0 document. Generation is
// greater than zero and increments by one per committed trust/config
// authorization change.
type TrustStore struct {
	Generation uint64
	Entries    []CredentialEntry
}

// Formatting a store or entry must never expose digests, UUIDs, DER bytes or
// timestamps: custody material never enters logs, snapshots or diagnostics.
func (TrustStore) String() string                    { return "host trust store" }
func (TrustStore) GoString() string                  { return "host trust store" }
func (TrustStore) Format(state fmt.State, verb rune) { _, _ = state.Write([]byte("host trust store")) }

func (CredentialEntry) String() string   { return "host trust credential entry" }
func (CredentialEntry) GoString() string { return "host trust credential entry" }
func (CredentialEntry) Format(state fmt.State, verb rune) {
	_, _ = state.Write([]byte("host trust credential entry"))
}

// DecodeTrust is the exact closed Host Trust Store 1.0.0 reader. Unknown,
// missing, duplicate-key, malformed, partial or downgraded state is refused,
// never read as an empty store and never given legacy meaning.
func DecodeTrust(document []byte) (TrustStore, error) {
	if err := rejectDuplicateKeys(document); err != nil {
		return TrustStore{}, err
	}
	var shape map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(document))
	if err := decoder.Decode(&shape); err != nil {
		return TrustStore{}, trustError(TrustError{Operation: "decode document", Err: errors.Join(ErrTrustDecode, err)})
	}
	if decoder.More() {
		return TrustStore{}, trustError(TrustError{Operation: "decode document", Err: ErrTrustDecode})
	}
	for key := range shape {
		switch key {
		case "schema", "schema_version", "generation", "entries":
		default:
			return TrustStore{}, trustError(TrustError{Operation: "decode document", Err: errors.Join(ErrTrustValidation, fmt.Errorf("unknown field"))})
		}
	}
	var schema, version string
	if err := decodeExactString(shape, "schema", &schema); err != nil {
		return TrustStore{}, err
	}
	if schema != TrustSchemaID {
		return TrustStore{}, trustError(TrustError{Operation: "decode schema", Err: ErrTrustValidation})
	}
	if err := decodeExactString(shape, "schema_version", &version); err != nil {
		return TrustStore{}, err
	}
	if version != TrustSchemaVersion {
		return TrustStore{}, trustError(TrustError{Operation: "decode schema version", Err: ErrTrustValidation})
	}
	generation, err := decodeGeneration(shape)
	if err != nil {
		return TrustStore{}, err
	}
	entries, err := decodeEntries(shape)
	if err != nil {
		return TrustStore{}, err
	}
	store := TrustStore{Generation: generation, Entries: entries}
	if err := checkSortedUnique(store.Entries); err != nil {
		return TrustStore{}, err
	}
	if err := checkMappingUnique(store.Entries); err != nil {
		return TrustStore{}, err
	}
	return store, nil
}

// checkMappingUnique enforces the cross-entry rules at read time, across all
// entries including revoked tombstones: a leaf, leaf public key or root must
// never map twice, even for the same host UUID. Identical leaf bytes imply an
// identical credential_id, which the ordering check already refuses, so the
// leaf member needs no separate comparison here.
func checkMappingUnique(entries []CredentialEntry) error {
	keys := make(map[string]struct{}, len(entries))
	roots := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if _, duplicate := keys[entry.SPKIID.String()]; duplicate {
			return trustError(TrustError{Operation: "decode entries", Err: ErrTrustValidation})
		}
		keys[entry.SPKIID.String()] = struct{}{}
		if _, duplicate := roots[entry.RootID.String()]; duplicate {
			return trustError(TrustError{Operation: "decode entries", Err: ErrTrustValidation})
		}
		roots[entry.RootID.String()] = struct{}{}
	}
	return nil
}
