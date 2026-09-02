package canonicaljson

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
)

// This file carries the named negative cases for the decoding, identity-contract
// and RFC 8785 refusals that a member-level sweep cannot reach: they refuse
// input before any record shape is selected, or they belong to the init-time
// cross-check that binds this package's tables to the generated catalog.

// TestCalculateObjectIdentityRefusesMalformedJSONInput drives the strict decoder
// through the real entry point. Each case violates exactly one decoding rule and
// asserts the refusal that rule emits, so a case cannot be satisfied by an
// earlier disjunct.
func TestCalculateObjectIdentityRefusesMalformedJSONInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty input", "", "input is empty"},
		{"truncated object", `{"schema":`, "decode JSON token"},
		{"unterminated object", `{"schema":"a"`, "decode object terminator"},
		{"unterminated array", `{"schema":["a"`, "decode array terminator"},
		{"top-level array", `[]`, "identity input must be a JSON object"},
		{"trailing token", `{"schema":"a"} 1`, "unexpected trailing JSON token"},
		{"trailing garbage", `{"schema":"a"} @`, "read trailing JSON data"},
		{"unterminated string", `{"schema":"a`, "unterminated JSON string"},
		{"unterminated escape", `{"schema":"a\`, "unterminated string escape"},
		{"unescaped control character", "{\"schema\":\"ab\"}", "unescaped control character in string"},
		{"malformed UTF-16 escape", `{"schema":"\uZZZZ"}`, "malformed UTF-16 escape"},
		{"lone low surrogate", `{"schema":"\udc00"}`, "lone low surrogate escape is forbidden"},
		{"unpaired high surrogate", `{"schema":"\ud800a"}`, "high surrogate escape must be followed by a low surrogate escape"},
		{"invalid UTF-8", "{\"schema\":\"\xff\"}", "input is not valid UTF-8"},
		{"duplicate member", `{"schema":"a","schema":"b"}`, "duplicate object member"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := CalculateObjectIdentity([]byte(test.input)); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("CalculateObjectIdentity(%q) error = %v, want a refusal containing %q", test.input, err, test.want)
			}
			if _, _, err := VerifyObjectIdentity([]byte(test.input)); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("VerifyObjectIdentity(%q) error = %v, want a refusal containing %q", test.input, err, test.want)
			}
		})
	}
}

// TestCalculateObjectIdentityRefusesANonAXNumberNestedInAContainer pins the
// recursive arm of the AX number model. A top-level member proves the scalar
// gate; only a nested one proves the walk reaches into arrays and objects, which
// is where an unsafe integer would otherwise be smuggled past the model.
func TestCalculateObjectIdentityRefusesANonAXNumberNestedInAContainer(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"inside an array", `{"schema":"a","values":[9007199254740992]}`, "outside the AX safe-integer interval"},
		{"inside a nested object", `{"schema":"a","nested":{"value":1.5}}`, "floating-point number"},
		{"inside an array inside an object", `{"schema":"a","nested":{"values":[1e3]}}`, "floating-point number"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := CalculateObjectIdentity([]byte(test.input)); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("CalculateObjectIdentity(%q) error = %v, want a refusal containing %q", test.input, err, test.want)
			}
		})
	}
}

// TestResolveSelfFieldRefusesAMalformedDiscriminatorMember pins the
// discriminator read. Self-field resolution runs before any closed-shape
// validation, so a discriminator that is not a non-empty string must be refused
// there rather than selecting a contract from a value that was never a string.
//
// The subject is derived from the generated catalog rather than named, and the
// candidate is built from that definition: the only discriminated contract is
// registered to an explicit refusal, so no valid fixture carries one and the
// guard would otherwise be unreachable from the fixture set.
func TestResolveSelfFieldRefusesAMalformedDiscriminatorMember(t *testing.T) {
	schema, version, discriminator, value := discriminatedSelfIdentityContract(t)

	t.Run("value not declared by the contract", func(t *testing.T) {
		object := map[string]any{"schema": schema, "schema_version": version, discriminator: value + "-other"}
		assertIdentityRefusalContains(t, object, "is not an immutable self-identity contract")
	})
	for _, test := range []struct {
		name  string
		value any
	}{
		{"number", json.Number("1")},
		{"null", nil},
		{"empty string", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			object := map[string]any{"schema": schema, "schema_version": version, discriminator: test.value}
			assertIdentityRefusalContains(t, object, "identity input member "+discriminator+" must be a non-empty string")
		})
	}
}

func assertIdentityRefusalContains(t *testing.T, object map[string]any, want string) {
	t.Helper()
	if _, _, err := CalculateObjectIdentity(mustJSON(t, object)); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("CalculateObjectIdentity error = %v, want a refusal containing %q", err, want)
	}
}

// discriminatedSelfIdentityContract derives one discriminated self-identity
// contract from the generated catalog, so a catalog change moves the test with
// it instead of leaving a stale schema name behind.
func discriminatedSelfIdentityContract(t *testing.T) (schema, version, discriminator, value string) {
	t.Helper()

	for _, definition := range catalog.Current().SelfIdentities {
		if definition.DiscriminatorName == "" || len(definition.ContractVersions) == 0 {
			continue
		}
		return string(definition.ContractID), definition.ContractVersions[0],
			definition.DiscriminatorName, definition.DiscriminatorValue
	}
	t.Fatal("the generated catalog declares no discriminated self-identity contract, so the discriminator guard has no subject")
	return "", "", "", ""
}

// TestCanonicalizeRefusesALogicalValueThatCannotBeSerialized pins the
// serialization arm of the exported RFC 8785 entry. It is reached with a
// json.Number carrying a token the encoder refuses, which is the one logical
// value decodeStrict can produce that json.Marshal will not accept.
func TestCanonicalizeRefusesALogicalValueThatCannotBeSerialized(t *testing.T) {
	if _, err := Canonicalize([]byte(`{"value":01}`)); err == nil {
		t.Fatal("Canonicalize accepted a JSON number with a leading zero, which is not I-JSON")
	}
}

// TestSchemaIdentityContractValidationRefusesADivergentTable drives the
// init-time cross-check that binds this package's self-identity table to the
// generated catalog. The production call site is buildSchemaIdentityContracts,
// whose caller panics on error; the check is exercised here as the pure function
// it is, so both of its refusals are proven rather than declared unreachable.
func TestSchemaIdentityContractValidationRefusesADivergentTable(t *testing.T) {
	t.Parallel()

	definitions := catalog.Current().SelfIdentities
	if len(definitions) == 0 {
		t.Fatal("the generated catalog declares no self-identity contracts")
	}
	complete, err := buildSchemaIdentityContracts(definitions)
	if err != nil {
		t.Fatalf("unmutated catalog is already refused: %v; every case below would be vacuous", err)
	}

	first := definitions[0]
	if len(first.ContractVersions) == 0 {
		t.Fatal("the first generated self-identity contract declares no versions")
	}
	firstKey := schemaIdentityKey{schema: string(first.ContractID), version: first.ContractVersions[0]}

	tests := []struct {
		name      string
		contracts func() map[schemaIdentityKey]schemaIdentityContract
		want      string
	}{
		{"a missing contract", func() map[schemaIdentityKey]schemaIdentityContract {
			contracts := copySchemaIdentityContracts(complete)
			delete(contracts, firstKey)
			return contracts
		}, "missing self-identity contract"},
		{"a contract whose self field differs from the catalog", func() map[schemaIdentityKey]schemaIdentityContract {
			contracts := copySchemaIdentityContracts(complete)
			contract := contracts[firstKey]
			contract.selfField = SelfField("forged_id")
			contracts[firstKey] = contract
			return contracts
		}, "differs from generated catalog"},
		{"a contract for a schema the catalog does not declare", func() map[schemaIdentityKey]schemaIdentityContract {
			contracts := copySchemaIdentityContracts(complete)
			contracts[schemaIdentityKey{schema: "urn:ax:schema:not-in-catalog", version: "1.0.0"}] = contracts[firstKey]
			return contracts
		}, "self-identity contract table has"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSchemaIdentityContracts(test.contracts(), definitions)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateSchemaIdentityContracts error = %v, want a refusal containing %q", err, test.want)
			}
		})
	}
}

// TestBuildSchemaIdentityContractsRefusesADuplicateContract pins the duplicate
// guard that runs before the cross-check above.
func TestBuildSchemaIdentityContractsRefusesADuplicateContract(t *testing.T) {
	t.Parallel()

	definitions := catalog.Current().SelfIdentities
	if len(definitions) == 0 || len(definitions[0].ContractVersions) == 0 {
		t.Fatal("the generated catalog declares no versioned self-identity contract")
	}
	duplicated := append(append([]catalog.SelfIdentityContract{}, definitions...), definitions[0])
	if _, err := buildSchemaIdentityContracts(duplicated); err == nil ||
		!strings.Contains(err.Error(), "duplicate self-identity contract") {
		t.Fatalf("buildSchemaIdentityContracts error = %v, want a duplicate-contract refusal", err)
	}
}

// TestImmutableObjectShapeValidatorRegistryRefusesADivergentTable drives the
// init-time cross-check that binds the shape-validator table to the generated
// registry, so a schema registered without a validator, or a validator
// registered for a schema the catalog does not declare, cannot fall through to
// extension-only attestation.
func TestImmutableObjectShapeValidatorRegistryRefusesADivergentTable(t *testing.T) {
	t.Parallel()

	definitions := catalog.Current().SelfIdentities
	if err := validateImmutableObjectShapeValidators(immutableObjectShapeValidators, definitions); err != nil {
		t.Fatalf("unmutated validator table is already refused: %v; every case below would be vacuous", err)
	}

	var firstKey schemaIdentityKey
	for key := range immutableObjectShapeValidators {
		firstKey = key
		break
	}

	tests := []struct {
		name       string
		validators func() map[schemaIdentityKey]immutableObjectShapeValidator
		want       string
	}{
		{"a missing validator", func() map[schemaIdentityKey]immutableObjectShapeValidator {
			validators := copyShapeValidators(immutableObjectShapeValidators)
			delete(validators, firstKey)
			return validators
		}, "missing immutable-object shape validator"},
		{"a validator for an unregistered schema", func() map[schemaIdentityKey]immutableObjectShapeValidator {
			validators := copyShapeValidators(immutableObjectShapeValidators)
			validators[schemaIdentityKey{schema: "urn:ax:schema:not-in-catalog", version: "1.0.0"}] = validators[firstKey]
			return validators
		}, "validator table has"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateImmutableObjectShapeValidators(test.validators(), definitions)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateImmutableObjectShapeValidators error = %v, want a refusal containing %q", err, test.want)
			}
		})
	}
}

// TestImmutableObjectShapeValidatorRegistryRefusesAnUnregisteredSchemaRow pins
// the third refusal of the same cross-check: a table with the right row count
// but a row the catalog never declared.
func TestImmutableObjectShapeValidatorRegistryRefusesAnUnregisteredSchemaRow(t *testing.T) {
	t.Parallel()

	definitions := catalog.Current().SelfIdentities
	validators := copyShapeValidators(immutableObjectShapeValidators)
	var firstKey schemaIdentityKey
	for key := range validators {
		firstKey = key
		break
	}
	validator := validators[firstKey]
	delete(validators, firstKey)
	validators[schemaIdentityKey{schema: "urn:ax:schema:not-in-catalog", version: "1.0.0"}] = validator

	err := validateImmutableObjectShapeValidators(validators, definitions)
	if err == nil || !strings.Contains(err.Error(), "missing immutable-object shape validator") {
		t.Fatalf("validateImmutableObjectShapeValidators error = %v, want a missing-validator refusal", err)
	}
}

func copySchemaIdentityContracts(source map[schemaIdentityKey]schemaIdentityContract) map[schemaIdentityKey]schemaIdentityContract {
	copied := make(map[schemaIdentityKey]schemaIdentityContract, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func copyShapeValidators(source map[schemaIdentityKey]immutableObjectShapeValidator) map[schemaIdentityKey]immutableObjectShapeValidator {
	copied := make(map[schemaIdentityKey]immutableObjectShapeValidator, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}
