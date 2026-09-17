package hosttrust

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
)

// EncodeTrust renders the canonical Host Trust Store 1.0.0 bytes: compact
// JSON, fixed field order, entries sorted by credential_id. Encoding the same
// store twice yields identical bytes. It never writes a file; durable commits
// belong to the store transaction.
func EncodeTrust(store TrustStore) ([]byte, error) {
	if store.Generation == 0 || store.Generation > MaxGeneration {
		return nil, trustError(TrustError{Operation: "encode generation", Err: ErrTrustValidation})
	}
	if len(store.Entries) > MaxEntries {
		return nil, trustError(TrustError{Operation: "encode entries", Err: ErrTrustValidation})
	}
	entries := append([]CredentialEntry(nil), store.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CredentialID.String() < entries[j].CredentialID.String()
	})
	for index := 1; index < len(entries); index++ {
		if entries[index-1].CredentialID.String() == entries[index].CredentialID.String() {
			return nil, trustError(TrustError{Operation: "encode entries", Err: ErrTrustValidation})
		}
	}
	var output bytes.Buffer
	output.WriteString(`{"schema":` + quoteJSON(TrustSchemaID))
	output.WriteString(`,"schema_version":` + quoteJSON(TrustSchemaVersion))
	output.WriteString(`,"generation":` + strconv.FormatUint(store.Generation, 10))
	output.WriteString(`,"entries":[`)
	for index, entry := range entries {
		if index > 0 {
			output.WriteByte(',')
		}
		encoded, err := encodeEntry(entry)
		if err != nil {
			return nil, err
		}
		output.Write(encoded)
	}
	output.WriteString(`]}`)
	document := output.Bytes()
	if _, err := DecodeTrust(document); err != nil {
		return nil, trustError(TrustError{Operation: "encode round trip", Err: errors.Join(ErrTrustValidation, err)})
	}
	return document, nil
}

func encodeEntry(entry CredentialEntry) ([]byte, error) {
	if entry.State != EntryActive && entry.State != EntryRetiring && entry.State != EntryRevoked {
		return nil, trustError(TrustError{Operation: "encode state", Err: ErrTrustValidation})
	}
	if entry.State == EntryRetiring && entry.RetireAt == nil {
		return nil, trustError(TrustError{Operation: "encode retire_at", Err: ErrTrustValidation})
	}
	if entry.State != EntryRetiring && entry.RetireAt != nil {
		return nil, trustError(TrustError{Operation: "encode retire_at", Err: ErrTrustValidation})
	}
	if len(entry.LeafDER) == 0 || len(entry.LeafDER) > MaxDERBytes || len(entry.RootDER) == 0 || len(entry.RootDER) > MaxDERBytes {
		return nil, trustError(TrustError{Operation: "encode certificate bytes", Err: ErrTrustValidation})
	}
	retireAt := "null"
	if entry.RetireAt != nil {
		retireAt = quoteJSON(entry.RetireAt.String())
	}
	var output bytes.Buffer
	output.WriteString(`{"host_id":` + quoteJSON(entry.HostID.String()))
	output.WriteString(`,"credential_id":` + quoteJSON(entry.CredentialID.String()))
	output.WriteString(`,"spki_id":` + quoteJSON(entry.SPKIID.String()))
	output.WriteString(`,"root_id":` + quoteJSON(entry.RootID.String()))
	output.WriteString(`,"leaf_der":` + quoteJSON(base64.RawURLEncoding.EncodeToString(entry.LeafDER)))
	output.WriteString(`,"root_der":` + quoteJSON(base64.RawURLEncoding.EncodeToString(entry.RootDER)))
	output.WriteString(`,"state":` + quoteJSON(entry.State))
	output.WriteString(`,"enrolled_at":` + quoteJSON(entry.EnrolledAt.String()))
	output.WriteString(`,"retire_at":` + retireAt + `}`)
	return output.Bytes(), nil
}

func quoteJSON(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}
