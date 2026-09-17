package hosttrust

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var entryFields = []string{"host_id", "credential_id", "spki_id", "root_id", "leaf_der", "root_der", "state", "enrolled_at", "retire_at"}

func decodeExactString(shape map[string]json.RawMessage, key string, target *string) error {
	raw, ok := shape[key]
	if !ok {
		return trustError(TrustError{Operation: "decode " + key, Err: ErrTrustValidation})
	}
	var value string
	decoder := strictJSONDecoder(raw)
	if err := decoder.Decode(&value); err != nil {
		return trustError(TrustError{Operation: "decode " + key, Err: errors.Join(ErrTrustDecode, err)})
	}
	*target = value
	return nil
}

func strictJSONDecoder(raw json.RawMessage) *json.Decoder {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	return decoder
}

func decodeGeneration(shape map[string]json.RawMessage) (uint64, error) {
	raw, ok := shape["generation"]
	if !ok {
		return 0, trustError(TrustError{Operation: "decode generation", Err: ErrTrustValidation})
	}
	var value any
	if err := strictJSONDecoder(raw).Decode(&value); err != nil {
		return 0, trustError(TrustError{Operation: "decode generation", Err: errors.Join(ErrTrustDecode, err)})
	}
	// A JSON string holding digits is not a uint53: generation must be a
	// bare JSON number.
	number, ok := value.(json.Number)
	if !ok {
		return 0, trustError(TrustError{Operation: "decode generation", Err: ErrTrustValidation})
	}
	text := number.String()
	if text == "" {
		return 0, trustError(TrustError{Operation: "decode generation", Err: ErrTrustValidation})
	}
	for _, digit := range text {
		if digit < '0' || digit > '9' {
			return 0, trustError(TrustError{Operation: "decode generation", Err: ErrTrustValidation})
		}
	}
	parsed, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, trustError(TrustError{Operation: "decode generation", Err: ErrTrustValidation})
	}
	if parsed == 0 || parsed > MaxGeneration {
		return 0, trustError(TrustError{Operation: "decode generation", Err: ErrTrustValidation})
	}
	return parsed, nil
}

func decodeEntries(shape map[string]json.RawMessage) ([]CredentialEntry, error) {
	raw, ok := shape["entries"]
	if !ok {
		return nil, trustError(TrustError{Operation: "decode entries", Err: ErrTrustValidation})
	}
	var items []map[string]json.RawMessage
	decoder := strictJSONDecoder(raw)
	if err := decoder.Decode(&items); err != nil {
		return nil, trustError(TrustError{Operation: "decode entries", Err: errors.Join(ErrTrustDecode, err)})
	}
	if items == nil {
		return nil, trustError(TrustError{Operation: "decode entries", Err: ErrTrustValidation})
	}
	if len(items) > MaxEntries {
		return nil, trustError(TrustError{Operation: "decode entries", Err: ErrTrustValidation})
	}
	entries := make([]CredentialEntry, 0, len(items))
	for index := range items {
		entry, err := decodeEntry(items[index])
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func decodeEntry(shape map[string]json.RawMessage) (CredentialEntry, error) {
	if len(shape) != len(entryFields) {
		return CredentialEntry{}, trustError(TrustError{Operation: "decode entry", Err: ErrTrustValidation})
	}
	for _, field := range entryFields {
		if _, ok := shape[field]; !ok {
			return CredentialEntry{}, trustError(TrustError{Operation: "decode entry", Err: ErrTrustValidation})
		}
	}
	var entry CredentialEntry
	if err := decodeEntryString(shape, "host_id", func(value string) error {
		parsed, err := scalar.ParseUUIDv7(value)
		if err != nil {
			return err
		}
		entry.HostID = parsed
		return nil
	}); err != nil {
		return CredentialEntry{}, err
	}
	for _, field := range []string{"credential_id", "spki_id", "root_id"} {
		if err := decodeEntryDigest(shape, field, &entry, field); err != nil {
			return CredentialEntry{}, err
		}
	}
	var leafDER, rootDER []byte
	for _, field := range []string{"leaf_der", "root_der"} {
		der, err := decodeEntryDER(shape, field)
		if err != nil {
			return CredentialEntry{}, err
		}
		if field == "leaf_der" {
			leafDER = der
		} else {
			rootDER = der
		}
	}
	entry.LeafDER = leafDER
	entry.RootDER = rootDER
	if err := decodeEntryString(shape, "state", func(value string) error {
		switch value {
		case EntryActive, EntryRetiring, EntryRevoked:
			entry.State = value
			return nil
		default:
			return ErrTrustValidation
		}
	}); err != nil {
		return CredentialEntry{}, err
	}
	if err := decodeEntryString(shape, "enrolled_at", func(value string) error {
		parsed, err := scalar.ParseTimestamp(value)
		if err != nil {
			return err
		}
		entry.EnrolledAt = parsed
		return nil
	}); err != nil {
		return CredentialEntry{}, err
	}
	retireRaw := shape["retire_at"]
	if string(retireRaw) == "null" {
		if entry.State == EntryRetiring {
			return CredentialEntry{}, trustError(TrustError{Operation: "decode retire_at", Err: ErrTrustValidation})
		}
	} else {
		var text string
		if err := strictJSONDecoder(retireRaw).Decode(&text); err != nil {
			return CredentialEntry{}, trustError(TrustError{Operation: "decode retire_at", Err: errors.Join(ErrTrustDecode, err)})
		}
		parsed, err := scalar.ParseTimestamp(text)
		if err != nil {
			return CredentialEntry{}, trustError(TrustError{Operation: "decode retire_at", Err: errors.Join(ErrTrustDecode, err)})
		}
		if entry.State != EntryRetiring {
			return CredentialEntry{}, trustError(TrustError{Operation: "decode retire_at", Err: ErrTrustValidation})
		}
		entry.RetireAt = &parsed
	}
	if entry.RetireAt != nil {
		retireAt, err := entry.RetireAt.Time()
		if err != nil {
			return CredentialEntry{}, trustError(TrustError{Operation: "decode retire_at", Err: errors.Join(ErrTrustDecode, err)})
		}
		enrolledAt, err := entry.EnrolledAt.Time()
		if err != nil {
			return CredentialEntry{}, trustError(TrustError{Operation: "decode enrolled_at", Err: errors.Join(ErrTrustDecode, err)})
		}
		if !retireAt.After(enrolledAt) {
			return CredentialEntry{}, trustError(TrustError{Operation: "decode retire_at", Err: ErrTrustValidation})
		}
	}
	leaf, err := x509.ParseCertificate(entry.LeafDER)
	if err != nil {
		return CredentialEntry{}, trustError(TrustError{Operation: "decode leaf_der", Err: errors.Join(ErrTrustDecode, err)})
	}
	if _, err := x509.ParseCertificate(entry.RootDER); err != nil {
		return CredentialEntry{}, trustError(TrustError{Operation: "decode root_der", Err: errors.Join(ErrTrustDecode, err)})
	}
	// The identifiers must reproduce the enclosed bytes: a store that names
	// another certificate's digest for these bytes is malformed, whether by
	// corruption or by hand, and never decodes.
	if scalar.SHA256Digest(entry.LeafDER).String() != entry.CredentialID.String() {
		return CredentialEntry{}, trustError(TrustError{Operation: "decode credential_id", Err: ErrTrustValidation})
	}
	if scalar.SHA256Digest(leaf.RawSubjectPublicKeyInfo).String() != entry.SPKIID.String() {
		return CredentialEntry{}, trustError(TrustError{Operation: "decode spki_id", Err: ErrTrustValidation})
	}
	if scalar.SHA256Digest(entry.RootDER).String() != entry.RootID.String() {
		return CredentialEntry{}, trustError(TrustError{Operation: "decode root_id", Err: ErrTrustValidation})
	}
	return entry, nil
}

func decodeEntryString(shape map[string]json.RawMessage, field string, accept func(string) error) error {
	var value string
	if err := strictJSONDecoder(shape[field]).Decode(&value); err != nil {
		return trustError(TrustError{Operation: "decode " + field, Err: errors.Join(ErrTrustDecode, err)})
	}
	if err := accept(value); err != nil {
		return trustError(TrustError{Operation: "decode " + field, Err: errors.Join(ErrTrustValidation, err)})
	}
	return nil
}

func decodeEntryDigest(shape map[string]json.RawMessage, field string, entry *CredentialEntry, target string) error {
	return decodeEntryString(shape, field, func(value string) error {
		parsed, err := scalar.ParseDigest(value)
		if err != nil {
			return err
		}
		switch target {
		case "credential_id":
			entry.CredentialID = parsed
		case "spki_id":
			entry.SPKIID = parsed
		case "root_id":
			entry.RootID = parsed
		}
		return nil
	})
}

func decodeEntryDER(shape map[string]json.RawMessage, field string) ([]byte, error) {
	var encoded string
	if err := strictJSONDecoder(shape[field]).Decode(&encoded); err != nil {
		return nil, trustError(TrustError{Operation: "decode " + field, Err: errors.Join(ErrTrustDecode, err)})
	}
	// base64url unpadded canonical: reject padded, standard-alphabet and
	// whitespace variants by re-encoding the decoded bytes.
	der, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, trustError(TrustError{Operation: "decode " + field, Err: errors.Join(ErrTrustDecode, err)})
	}
	if len(der) == 0 || len(der) > MaxDERBytes {
		return nil, trustError(TrustError{Operation: "decode " + field, Err: ErrTrustValidation})
	}
	if base64.RawURLEncoding.EncodeToString(der) != encoded {
		return nil, trustError(TrustError{Operation: "decode " + field, Err: ErrTrustValidation})
	}
	return der, nil
}

// rejectDuplicateKeys refuses JSON objects that repeat a key. encoding/json
// keeps only the last duplicate silently, which would let a partial or
// downgraded state hide behind an earlier key, so the token stream is
// audited before decoding.
func rejectDuplicateKeys(document []byte) error {
	type frame struct {
		object    bool
		expectKey bool
		keys      map[string]struct{}
	}
	var stack []frame
	decoder := json.NewDecoder(bytes.NewReader(document))
	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return trustError(TrustError{Operation: "decode document", Err: errors.Join(ErrTrustDecode, err)})
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				stack = append(stack, frame{object: true, expectKey: true, keys: map[string]struct{}{}})
			case '[':
				stack = append(stack, frame{})
			case '}', ']':
				if len(stack) == 0 {
					return trustError(TrustError{Operation: "decode document", Err: ErrTrustDecode})
				}
				wantObject := delim == '}'
				if stack[len(stack)-1].object != wantObject {
					return trustError(TrustError{Operation: "decode document", Err: ErrTrustDecode})
				}
				stack = stack[:len(stack)-1]
				// A closed nested value completes its parent's key: the
				// parent object expects the next key.
				if len(stack) > 0 && stack[len(stack)-1].object {
					stack[len(stack)-1].expectKey = true
				}
			}
			continue
		}
		if len(stack) == 0 {
			continue
		}
		top := &stack[len(stack)-1]
		text, isString := token.(string)
		if top.object && top.expectKey {
			if !isString {
				return trustError(TrustError{Operation: "decode document", Err: ErrTrustDecode})
			}
			if _, seen := top.keys[text]; seen {
				return trustError(TrustError{Operation: "decode document", Err: ErrTrustValidation})
			}
			top.keys[text] = struct{}{}
			top.expectKey = false
			continue
		}
		if top.object {
			top.expectKey = true
		}
	}
	if len(stack) != 0 {
		return trustError(TrustError{Operation: "decode document", Err: ErrTrustDecode})
	}
	return nil
}

func checkSortedUnique(entries []CredentialEntry) error {
	for index := 1; index < len(entries); index++ {
		previous := entries[index-1].CredentialID.String()
		current := entries[index].CredentialID.String()
		if previous >= current {
			return trustError(TrustError{Operation: "decode entries", Err: ErrTrustValidation})
		}
	}
	return nil
}
