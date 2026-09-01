//go:build !race

package canonicaljson

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// The race detector deliberately instruments every map and string operation,
// so wall-clock regression thresholds are measured only in an ordinary test
// binary. Functional and race coverage of this gate remain in canonical_test.
func TestTransferManifestMaximumEntryGateIsLinear(t *testing.T) {
	object := validTransferManifestObject("workspace_tree")
	entries := make([]any, 65_536)
	for index := range entries {
		entries[index] = map[string]any{
			"path": fmt.Sprintf("p%05x", index),
			"type": "directory",
			"mode": json.Number("493"),
		}
	}
	object["entries"] = entries
	input := mustJSON(t, object)
	if len(input) > 5_242_880 {
		t.Fatalf("maximum-entry Transfer Manifest encodes to %d bytes, want at most 5242880", len(input))
	}

	started := time.Now()
	if _, field, err := CalculateObjectIdentity(input); err != nil || field != SelfManifestID {
		t.Fatalf("CalculateObjectIdentity(maximum-entry Transfer Manifest) field = %q, error = %v", field, err)
	}
	if elapsed := time.Since(started); elapsed >= 2*time.Second {
		t.Fatalf("65,536-entry Transfer Manifest calculation gate took %s, want less than 2s", elapsed)
	}
}
