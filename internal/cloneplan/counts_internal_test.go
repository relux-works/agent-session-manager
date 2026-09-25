package cloneplan

import (
	"errors"
	"testing"
)

// TestFactoredCountGates pins the exact cardinality edges at the
// factored gates: admission at min/max, refusal at min-1/max+1.
// Entry reachability pins at both production entries in the
// external suite, including the million-row upper edge
// (TestMappingCountMillionEntryLevel seals and decodes it).
func TestFactoredCountGates(t *testing.T) {
	cases := []struct {
		label   string
		gate    func(int) error
		minimum int
		maximum int
		token   string
	}{
		{"mappings", checkMappingCount, 1, 1000000, "item_mappings"},
		{"operations", checkOperationCount, 1, 65536, "target_operations"},
		{"resources", checkExpectedResourceCount, 0, 65536, "expected_resources"},
		{"events", checkSynthesizedEventCount, 0, 65536, "synthesized_events"},
		{"contracts", checkContractCount, 1, 64, "required_contracts"},
		{"entries", checkProjectedEntryCount, 0, 65536, "entries"},
	}
	for _, row := range cases {
		t.Run(row.label, func(t *testing.T) {
			if err := row.gate(row.minimum); err != nil {
				t.Errorf("min %d refused: %v", row.minimum, err)
			}
			if err := row.gate(row.maximum); err != nil {
				t.Errorf("max %d refused: %v", row.maximum, err)
			}
			if err := row.gate(row.minimum - 1); !errors.Is(err, ErrInvalid) {
				t.Errorf("min-1 %d admits, want ErrInvalid", row.minimum-1)
			}
			if err := row.gate(row.maximum + 1); !errors.Is(err, ErrInvalid) {
				t.Errorf("max+1 %d admits, want ErrInvalid", row.maximum+1)
			}
		})
	}
}
