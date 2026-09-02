// Package canonicaljson implements the RFC 8785 JSON Canonicalization Scheme
// and the AX omit-self-field identity boundary for immutable logical objects.
package canonicaljson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gowebpki/jcs"
	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const maxSafeInteger int64 = 9007199254740991

// maxNestingDepth bounds how many JSON containers (objects or arrays) may be
// open at once in any document that reaches decodeValue, which is the shared
// strict decode path behind Canonicalize, CalculateObjectIdentity, and
// VerifyObjectIdentity. Without it, one recursion frame per nesting level lets
// an untrusted peer object exhaust the goroutine stack, and Go reports that as
// a fatal runtime error that recover cannot intercept.
//
// Rationale for 256: the deepest normative closed shape is the Transfer
// Manifest submodule tree at its 16-level maximum, which nests roughly 40
// containers including the identity envelope, and open extensions are bounded
// at 4 levels, so 256 leaves more than sixfold headroom over any accepted AX
// object. It is also far below encoding/json's 10,000-level Unmarshal cap and
// the unbounded recursion inside the JCS re-parse, so this typed refusal is
// always the first gate a deep document meets, and the decode stack stays at
// a few hundred small frames regardless of input size.
const maxNestingDepth = 256

var (
	// ErrInvalidJSON reports input outside the RFC 8785 I-JSON surface.
	ErrInvalidJSON = errors.New("invalid canonical JSON input")
	// ErrInvalidIdentity reports input outside the AX omit-self identity contract.
	ErrInvalidIdentity = errors.New("invalid canonical object identity")
)

// SelfField is one of the pinned AX immutable-object self-identity namespaces.
// Chunk IDs are intentionally absent: Section 10.3 defines chunk_id as a
// digest of raw bytes, not of a JSON object.
type SelfField string

const (
	SelfRecordID                   SelfField = "record_id"
	SelfEventID                    SelfField = "event_id"
	SelfCheckpointID               SelfField = "checkpoint_id"
	SelfDescriptorID               SelfField = "descriptor_id"
	SelfManifestID                 SelfField = "manifest_id"
	SelfPlanID                     SelfField = "plan_id"
	SelfTombstoneID                SelfField = "tombstone_id"
	SelfAckID                      SelfField = "ack_id"
	SelfBundleID                   SelfField = "bundle_id"
	SelfMarkerID                   SelfField = "marker_id"
	SelfObservationID              SelfField = "observation_id"
	SelfBatchID                    SelfField = "batch_id"
	SelfLineageLinkID              SelfField = "lineage_link_id"
	SelfAnnotationID               SelfField = "annotation_id"
	SelfProfileID                  SelfField = "profile_id"
	SelfJobRequestID               SelfField = "job_request_id"
	SelfJobReceiptID               SelfField = "job_receipt_id"
	SelfDirectoryReceiptID         SelfField = "directory_receipt_id"
	SelfProbeID                    SelfField = "probe_id"
	SelfBindingID                  SelfField = "binding_id"
	SelfEvidenceID                 SelfField = "evidence_id"
	SelfRawObjectManifestID        SelfField = "raw_object_manifest_id"
	SelfCaptureManifestID          SelfField = "capture_manifest_id"
	SelfCanonicalSessionID         SelfField = "canonical_session_id"
	SelfFidelityReportID           SelfField = "fidelity_report_id"
	SelfProjectionPlanID           SelfField = "projection_plan_id"
	SelfProjectedObjectManifestID  SelfField = "projected_object_manifest_id"
	SelfReadBackEvidenceManifestID SelfField = "read_back_evidence_manifest_id"
	SelfValidationReportID         SelfField = "validation_report_id"
	SelfMigrationCheckpointID      SelfField = "migration_checkpoint_id"
	SelfLineageReceiptID           SelfField = "lineage_receipt_id"
	SelfRegistryDigest             SelfField = "registry_digest"
)

type schemaIdentityKey struct {
	schema  string
	version string
}

type schemaIdentityContract struct {
	selfField          SelfField
	discriminatorName  string
	discriminatorValue string
}

// schemaIdentityContracts is derived from the generated catalog whose source
// is bound to the pinned v0.5.0 document. A schema may legitimately carry a
// differently named ID as a reference, so field-name counting is not an
// identity contract.
var schemaIdentityContracts = mustBuildSchemaIdentityContracts(catalog.Current().SelfIdentities)

func mustBuildSchemaIdentityContracts(definitions []catalog.SelfIdentityContract) map[schemaIdentityKey]schemaIdentityContract {
	contracts, err := buildSchemaIdentityContracts(definitions)
	if err != nil {
		panic(fmt.Sprintf("generated self-identity catalog is invalid: %v", err))
	}
	return contracts
}

func buildSchemaIdentityContracts(definitions []catalog.SelfIdentityContract) (map[schemaIdentityKey]schemaIdentityContract, error) {
	contracts := make(map[schemaIdentityKey]schemaIdentityContract)
	for _, definition := range definitions {
		contract := schemaIdentityContract{
			selfField:          SelfField(definition.SelfField),
			discriminatorName:  definition.DiscriminatorName,
			discriminatorValue: definition.DiscriminatorValue,
		}
		for _, version := range definition.ContractVersions {
			key := schemaIdentityKey{schema: string(definition.ContractID), version: version}
			if _, duplicate := contracts[key]; duplicate {
				return nil, fmt.Errorf("duplicate self-identity contract for %s@%s", key.schema, key.version)
			}
			contracts[key] = contract
		}
	}
	if err := validateSchemaIdentityContracts(contracts, definitions); err != nil {
		return nil, err
	}
	return contracts, nil
}

func validateSchemaIdentityContracts(contracts map[schemaIdentityKey]schemaIdentityContract, definitions []catalog.SelfIdentityContract) error {
	expected := 0
	for _, definition := range definitions {
		for _, version := range definition.ContractVersions {
			expected++
			key := schemaIdentityKey{schema: string(definition.ContractID), version: version}
			contract, ok := contracts[key]
			if !ok {
				return fmt.Errorf("missing self-identity contract for %s@%s", key.schema, key.version)
			}
			if contract.selfField != SelfField(definition.SelfField) ||
				contract.discriminatorName != definition.DiscriminatorName ||
				contract.discriminatorValue != definition.DiscriminatorValue {
				return fmt.Errorf("self-identity contract for %s@%s differs from generated catalog", key.schema, key.version)
			}
		}
	}
	if len(contracts) != expected {
		return fmt.Errorf("self-identity contract table has %d rows, generated catalog has %d", len(contracts), expected)
	}
	return nil
}

// Canonicalize is the production RFC 8785 entry point. It rejects malformed
// UTF-8, invalid surrogate escapes, duplicate object names, and non-I-JSON
// input before producing the canonical UTF-8 byte sequence.
func Canonicalize(input []byte) ([]byte, error) {
	value, err := decodeStrict(input)
	if err != nil {
		return nil, err
	}
	// Re-encode the parsed logical value before the JCS transform. This removes
	// leading/trailing whitespace around primitive top-level values without
	// changing json.Number tokens or string data.
	normalized, err := json.Marshal(value)
	if err != nil {
		return nil, invalidJSON("serialize logical JSON value: %v", err)
	}
	canonical, err := jcs.Transform(normalized)
	if err != nil {
		return nil, invalidJSON("canonicalize RFC 8785 value: %v", err)
	}
	return canonical, nil
}

// CalculateObjectIdentity validates the AX common logical number model,
// resolves the one self field from the object's trusted schema/version
// contract, omits that field, applies RFC 8785 JCS, and returns the SHA-256
// identity. Callers cannot select an arbitrary field to omit, which prevents
// self inclusion and digest cycles.
func CalculateObjectIdentity(input []byte) (scalar.Digest, SelfField, error) {
	object, selfField, _, err := prepareObjectIdentity(input)
	if err != nil {
		return scalar.Digest{}, "", err
	}
	return calculatePreparedObjectIdentity(object, selfField)
}

func prepareObjectIdentity(input []byte) (map[string]any, SelfField, scalar.Digest, error) {
	if len(input) > 5_242_880 {
		return nil, "", scalar.Digest{}, invalidIdentity("encoded identity object is %d bytes, maximum is 5242880", len(input))
	}
	value, err := decodeStrict(input)
	if err != nil {
		return nil, "", scalar.Digest{}, err
	}
	if err := validateAXNumbers(value); err != nil {
		return nil, "", scalar.Digest{}, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, "", scalar.Digest{}, invalidIdentity("identity input must be a JSON object")
	}

	selfField, claimed, err := resolveSelfField(object)
	if err != nil {
		return nil, "", scalar.Digest{}, err
	}
	// This is the composed production gate: closed contract validation happens
	// after trusted schema selection and before either calculation or attestation.
	if err := validateImmutableObjectShape(object); err != nil {
		return nil, "", scalar.Digest{}, err
	}
	return object, selfField, claimed, nil
}

func calculatePreparedObjectIdentity(object map[string]any, selfField SelfField) (scalar.Digest, SelfField, error) {
	omitted := make(map[string]any, len(object)-1)
	for name, member := range object {
		if name != string(selfField) {
			omitted[name] = member
		}
	}
	serialized, err := json.Marshal(omitted)
	if err != nil {
		return scalar.Digest{}, "", invalidIdentity("serialize omit-self object: %v", err)
	}
	canonical, err := jcs.Transform(serialized)
	if err != nil {
		return scalar.Digest{}, "", invalidIdentity("canonicalize omit-self object: %v", err)
	}
	return scalar.SHA256Digest(canonical), selfField, nil
}

// VerifyObjectIdentity drives the same production calculation and refuses a
// malformed, self-included, or otherwise mismatched claimed identity.
func VerifyObjectIdentity(input []byte) (scalar.Digest, SelfField, error) {
	object, selfField, claimed, err := prepareObjectIdentity(input)
	if err != nil {
		return scalar.Digest{}, "", err
	}
	calculated, calculatedField, err := calculatePreparedObjectIdentity(object, selfField)
	if err != nil {
		return scalar.Digest{}, "", err
	}
	if calculatedField != selfField || claimed != calculated {
		return scalar.Digest{}, "", invalidIdentity(
			"%s claim %q does not match omit-self digest %q",
			selfField,
			claimed,
			calculated,
		)
	}
	return calculated, selfField, nil
}

func resolveSelfField(object map[string]any) (SelfField, scalar.Digest, error) {
	schema, err := requiredStringMember(object, "schema")
	if err != nil {
		return "", scalar.Digest{}, err
	}
	version, err := requiredStringMember(object, "schema_version")
	if err != nil {
		return "", scalar.Digest{}, err
	}
	contract, ok := schemaIdentityContracts[schemaIdentityKey{schema: schema, version: version}]
	if !ok {
		return "", scalar.Digest{}, invalidIdentity(
			"schema %q version %q has no supported immutable self-identity contract",
			schema,
			version,
		)
	}
	if contract.discriminatorName != "" {
		discriminator, err := requiredStringMember(object, contract.discriminatorName)
		if err != nil {
			return "", scalar.Digest{}, err
		}
		if discriminator != contract.discriminatorValue {
			return "", scalar.Digest{}, invalidIdentity(
				"schema %q version %q variant %s=%q is not an immutable self-identity contract",
				schema,
				version,
				contract.discriminatorName,
				discriminator,
			)
		}
	}

	value, ok := object[string(contract.selfField)]
	if !ok {
		return "", scalar.Digest{}, invalidIdentity(
			"schema %q version %q requires self field %s",
			schema,
			version,
			contract.selfField,
		)
	}
	text, ok := value.(string)
	if !ok {
		return "", scalar.Digest{}, invalidIdentity("self field %s must contain a digest string", contract.selfField)
	}
	claim, err := scalar.ParseDigest(text)
	if err != nil {
		return "", scalar.Digest{}, invalidIdentity("self field %s: %v", contract.selfField, err)
	}
	return contract.selfField, claim, nil
}

func requiredStringMember(object map[string]any, name string) (string, error) {
	value, ok := object[name]
	if !ok {
		return "", invalidIdentity("identity input requires string member %s", name)
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return "", invalidIdentity("identity input member %s must be a non-empty string", name)
	}
	return text, nil
}

func decodeStrict(input []byte) (any, error) {
	if len(input) == 0 {
		return nil, invalidJSON("input is empty")
	}
	if !utf8.Valid(input) {
		return nil, invalidJSON("input is not valid UTF-8")
	}
	if err := validateSurrogateEscapes(input); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	value, err := decodeValue(decoder, 0)
	if err != nil {
		return nil, err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, invalidJSON("read trailing JSON data: %v", err)
		}
		return nil, invalidJSON("unexpected trailing JSON token %v", token)
	}
	return value, nil
}

// decodeValue decodes one logical JSON value. depth counts the containers
// already open around this value, so the top-level value is decoded at depth 0
// and a container at depth maxNestingDepth-1 is the last one allowed to open.
func decodeValue(decoder *json.Decoder, depth int) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, invalidJSON("decode JSON token: %v", err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	if depth >= maxNestingDepth {
		return nil, invalidJSON("nesting depth %d exceeds maximum %d", depth+1, maxNestingDepth)
	}

	switch delimiter {
	case '{':
		object := make(map[string]any)
		for decoder.More() {
			nameToken, err := decoder.Token()
			if err != nil {
				return nil, invalidJSON("decode object name: %v", err)
			}
			name, ok := nameToken.(string)
			if !ok {
				return nil, invalidJSON("object name is not a string")
			}
			if _, duplicate := object[name]; duplicate {
				return nil, invalidJSON("duplicate object member %q", name)
			}
			value, err := decodeValue(decoder, depth+1)
			if err != nil {
				return nil, err
			}
			object[name] = value
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return nil, invalidJSON("decode object terminator: %v", err)
		}
		return object, nil
	case '[':
		array := make([]any, 0)
		for decoder.More() {
			value, err := decodeValue(decoder, depth+1)
			if err != nil {
				return nil, err
			}
			array = append(array, value)
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return nil, invalidJSON("decode array terminator: %v", err)
		}
		return array, nil
	default:
		return nil, invalidJSON("unexpected delimiter %q", delimiter)
	}
}

func validateAXNumbers(value any) error {
	switch typed := value.(type) {
	case json.Number:
		literal := typed.String()
		if strings.ContainsAny(literal, ".eE") {
			return invalidIdentity("floating-point number %q is forbidden by the AX common model", literal)
		}
		integer, err := strconv.ParseInt(literal, 10, 64)
		if err != nil || integer < -maxSafeInteger || integer > maxSafeInteger {
			return invalidIdentity("integer %q is outside the AX safe-integer interval", literal)
		}
	case []any:
		for _, element := range typed {
			if err := validateAXNumbers(element); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, member := range typed {
			if err := validateAXNumbers(member); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSurrogateEscapes(input []byte) error {
	for index := 0; index < len(input); index++ {
		if input[index] != '"' {
			continue
		}
		for index++; index < len(input); index++ {
			switch input[index] {
			case '"':
				goto stringComplete
			case '\\':
				index++
				if index >= len(input) {
					return invalidJSON("unterminated string escape")
				}
				if input[index] != 'u' {
					continue
				}
				codeUnit, end, err := readUTF16Escape(input, index-1)
				if err != nil {
					return err
				}
				index = end - 1
				switch {
				case codeUnit >= 0xd800 && codeUnit <= 0xdbff:
					second, secondEnd, err := readUTF16Escape(input, end)
					if err != nil || second < 0xdc00 || second > 0xdfff {
						return invalidJSON("high surrogate escape must be followed by a low surrogate escape")
					}
					index = secondEnd - 1
				case codeUnit >= 0xdc00 && codeUnit <= 0xdfff:
					return invalidJSON("lone low surrogate escape is forbidden")
				}
			case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
				0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
				0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f:
				return invalidJSON("unescaped control character in string")
			}
		}
		return invalidJSON("unterminated JSON string")
	stringComplete:
	}
	return nil
}

func readUTF16Escape(input []byte, start int) (uint16, int, error) {
	if start < 0 || start+6 > len(input) || input[start] != '\\' || input[start+1] != 'u' {
		return 0, start, invalidJSON("expected UTF-16 escape")
	}
	value, err := strconv.ParseUint(string(input[start+2:start+6]), 16, 16)
	if err != nil {
		return 0, start, invalidJSON("malformed UTF-16 escape")
	}
	return uint16(value), start + 6, nil
}

func invalidJSON(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidJSON, fmt.Sprintf(format, arguments...))
}

func invalidIdentity(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidIdentity, fmt.Sprintf(format, arguments...))
}
