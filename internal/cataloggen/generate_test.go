package cataloggen_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/cataloggen"
)

func TestGenerateMatchesCommittedTypedCatalog(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	got, err := cataloggen.Generate(metadata, lock)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	want, err := os.ReadFile(filepath.Join("..", "catalog", "catalog_gen.go"))
	if err != nil {
		t.Fatalf("read committed generated catalog: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("generated catalog is stale; run go generate ./internal/catalog")
	}
}

func TestGenerateIsDeterministicAndSideEffectFree(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	first, err := cataloggen.Generate(metadata, lock)
	if err != nil {
		t.Fatalf("first Generate() error = %v", err)
	}
	second, err := cataloggen.Generate(metadata, lock)
	if err != nil {
		t.Fatalf("second Generate() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("Generate returned byte-different output for identical inputs")
	}
}

func TestGenerateRejectsMalformedPartialAndSubstitutedMetadata(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	tests := []struct {
		name     string
		metadata []byte
	}{
		{name: "absent", metadata: nil},
		{name: "partial", metadata: metadata[:len(metadata)/2]},
		{name: "trailing value", metadata: append(append([]byte(nil), metadata...), []byte("{}")...)},
		{name: "source substitution", metadata: mutateMetadata(t, metadata, func(document map[string]any) {
			document["source"].(map[string]any)["commit"] = "0000000000000000000000000000000000000000"
		})},
		{name: "unknown member", metadata: mutateMetadata(t, metadata, func(document map[string]any) {
			document["unexpected"] = true
		})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := cataloggen.Generate(test.metadata, lock); !errors.Is(err, cataloggen.ErrInvalidMetadata) {
				t.Fatalf("Generate() error = %v, want ErrInvalidMetadata", err)
			}
		})
	}
}

func TestGenerateRejectsChangedNormativeLock(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	changed := append([]byte(nil), lock...)
	changed[len(changed)-2] = ' '
	if _, err := cataloggen.Generate(metadata, changed); !errors.Is(err, cataloggen.ErrInvalidMetadata) {
		t.Fatalf("Generate() error = %v, want ErrInvalidMetadata", err)
	}
}

func TestGenerateRejectsDuplicateAndUnboundOperations(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "duplicate operation",
			mutate: func(document map[string]any) {
				family := document["operation_families"].([]any)[0].(map[string]any)
				operations := family["operations"].([]any)
				family["operations"] = append(operations, operations[0])
			},
		},
		{
			name: "unknown contract",
			mutate: func(document map[string]any) {
				document["operation_families"].([]any)[0].(map[string]any)["contract_id"] = "urn:ax:protocol:forged"
			},
		},
		{
			name: "unsupported contract version",
			mutate: func(document map[string]any) {
				document["operation_families"].([]any)[0].(map[string]any)["contract_versions"] = []any{"9.9.9"}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := mutateMetadata(t, metadata, test.mutate)
			if _, err := cataloggen.Generate(candidate, lock); !errors.Is(err, cataloggen.ErrInvalidMetadata) {
				t.Fatalf("Generate() error = %v, want ErrInvalidMetadata", err)
			}
		})
	}
}

func TestGenerateRejectsInvalidOrNarrowedSelfIdentityContracts(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "deleted reviewed contract",
			mutate: func(document map[string]any) {
				identities := document["self_identity_contracts"].([]any)
				document["self_identity_contracts"] = identities[1:]
			},
		},
		{
			name: "unknown contract",
			mutate: func(document map[string]any) {
				document["self_identity_contracts"].([]any)[0].(map[string]any)["contract_id"] = "urn:ax:schema:forged"
			},
		},
		{
			name: "unsupported version",
			mutate: func(document map[string]any) {
				document["self_identity_contracts"].([]any)[0].(map[string]any)["contract_versions"] = []any{"9.9.9"}
			},
		},
		{
			name: "invalid self field",
			mutate: func(document map[string]any) {
				document["self_identity_contracts"].([]any)[0].(map[string]any)["self_field"] = "RecordID"
			},
		},
		{
			name: "incomplete discriminator",
			mutate: func(document map[string]any) {
				document["self_identity_contracts"].([]any)[0].(map[string]any)["discriminator_name"] = "document_kind"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := mutateMetadata(t, metadata, test.mutate)
			if _, err := cataloggen.Generate(candidate, lock); !errors.Is(err, cataloggen.ErrInvalidMetadata) {
				t.Fatalf("Generate() error = %v, want ErrInvalidMetadata", err)
			}
		})
	}
}

func TestGenerateRejectsNonEmptyNarrowedDurableMutationEvidence(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "idempotency scope",
			mutate: func(document map[string]any) {
				family := document["operation_families"].([]any)[0].(map[string]any)
				group := family["mutation_groups"].([]any)[0].(map[string]any)
				group["idempotency_key"] = "bootstrap_operation_id"
			},
		},
		{
			name: "recovery evidence",
			mutate: func(document map[string]any) {
				family := document["operation_families"].([]any)[0].(map[string]any)
				group := family["mutation_groups"].([]any)[0].(map[string]any)
				evidence := group["recovery_evidence"].([]any)
				group["recovery_evidence"] = evidence[:len(evidence)-1]
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := mutateMetadata(t, metadata, test.mutate)
			if _, err := cataloggen.Generate(candidate, lock); !errors.Is(err, cataloggen.ErrInvalidMetadata) {
				t.Fatalf("Generate() error = %v, want ErrInvalidMetadata", err)
			}
		})
	}
}

func TestGenerateRejectsForgedNonEmptyTraceability(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "normative section",
			mutate: func(document map[string]any) {
				document["operation_families"].([]any)[0].(map[string]any)["normative_section"] = "section-does-not-exist"
			},
		},
		{
			name: "fixture anchor",
			mutate: func(document map[string]any) {
				document["operation_families"].([]any)[0].(map[string]any)["fixture_families"] = []any{"fixture-does-not-exist"}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := mutateMetadata(t, metadata, test.mutate)
			if _, err := cataloggen.Generate(candidate, lock); !errors.Is(err, cataloggen.ErrInvalidMetadata) {
				t.Fatalf("Generate() error = %v, want ErrInvalidMetadata", err)
			}
		})
	}
}

func TestGenerateAcceptsSemanticallyIdenticalJSONFormatting(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	var document any
	if err := json.Unmarshal(metadata, &document); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	reformatted, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("reformat metadata: %v", err)
	}
	if _, err := cataloggen.Generate(reformatted, lock); err != nil {
		t.Fatalf("Generate(reformatted metadata) error = %v", err)
	}
}

func TestGenerateRejectsCapabilityAvailabilityClaim(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	candidate := mutateMetadata(t, metadata, func(document map[string]any) {
		family := document["capability_families"].([]any)[0].(map[string]any)
		capabilities := family["capabilities"].([]any)
		capabilities[0] = map[string]any{"name": capabilities[0], "enabled": true}
	})
	if _, err := cataloggen.Generate(candidate, lock); !errors.Is(err, cataloggen.ErrInvalidMetadata) {
		t.Fatalf("Generate() error = %v, want ErrInvalidMetadata", err)
	}
}

func TestGenerateRejectsReleaseContractSubstitution(t *testing.T) {
	t.Parallel()

	metadata, lock := sourceInputs(t)
	candidate := mutateMetadata(t, metadata, func(document map[string]any) {
		terminal := document["operation_families"].([]any)[0].(map[string]any)
		terminal["releases"] = []any{"v0.4.3", "v0.5.0"}
	})
	if _, err := cataloggen.Generate(candidate, lock); !errors.Is(err, cataloggen.ErrInvalidMetadata) {
		t.Fatalf("Generate() error = %v, want ErrInvalidMetadata", err)
	}
}

func sourceInputs(t *testing.T) ([]byte, []byte) {
	t.Helper()
	metadata, err := os.ReadFile(filepath.Join("..", "catalog", "catalog.v0.5.0.json"))
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	lock, err := os.ReadFile(filepath.Join("..", "specpin", "v0.5.0.lock.json"))
	if err != nil {
		t.Fatalf("read source lock: %v", err)
	}
	return metadata, lock
}

func mutateMetadata(t *testing.T, source []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(source, &document); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	mutate(document)
	candidate, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode metadata mutation: %v", err)
	}
	return candidate
}
