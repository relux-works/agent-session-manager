package canonicaljson

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCheckManifestEntriesUsesManifestOwner(t *testing.T) {
	for _, raw := range []string{`[]`, `[{"path":"src","type":"directory","mode":493}]`, `[{"path":"current","type":"symlink","mode":511,"target":"releases/v1"}]`} {
		if err := CheckManifestEntries([]byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`null`, `{}`, `[`, `[{"path":"src","type":"directory","mode":493,"target":"leak"}]`, `[{"path":"current","type":"symlink","mode":511,"target":"../escape"}]`, `[{"path":"A","type":"directory","mode":493},{"path":"a","type":"directory","mode":493}]`} {
		if err := CheckManifestEntries([]byte(raw)); err == nil {
			t.Fatalf("admitted %s", raw)
		}
	}
}

func TestCheckManifestEntriesCountBounds(t *testing.T) {
	entries := make([]any, 65537)
	for i := range entries {
		entries[i] = map[string]any{"path": fmt.Sprintf("dir-%05d", i), "type": "directory", "mode": 493}
	}
	at, _ := json.Marshal(entries[:65536])
	over, _ := json.Marshal(entries)
	if err := CheckManifestEntries(at); err != nil {
		t.Fatal(err)
	}
	if err := CheckManifestEntries(over); err == nil {
		t.Fatal("65537 entries admitted")
	}
}
