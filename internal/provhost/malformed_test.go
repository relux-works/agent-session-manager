package provhost

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestOperationDecodersRejectMalformedFrames drives every
// operation-layer decoder with a malformed frame so the
// decodeStrictObject fault conduit at each entry point is exercised:
// the site audit requires every constructor call site to sit on an
// exercised negative path, including conduits that re-emit the frame
// fault rather than minting a new arm.
func TestOperationDecodersRejectMalformedFrames(t *testing.T) {
	rows := []struct {
		name string
		err  error
	}{
		{"manifest truncated", DecodeManifest([]byte(`{"schema":`))},
		{"manifest array", DecodeManifest([]byte(`[1,2]`))},
		{"probe truncated", DecodeProbe([]byte(`{"schema":`))},
		{"probe capabilities array", DecodeProbe([]byte(`{"schema":"urn:ax:schema:provider-probe","schema_version":"1.0.0","provider_id":"pi","provider_version":"0.73.1","platform":"macos","architecture":"arm64","capabilities":[],"warnings":[]}`))},
		{"probe capability scalar", DecodeProbe(probeVariant(t, `"native_resume": {
      "status": "available",
      "enabled": true,
      "evidence": "probed",
      "detail": "--session, --continue, and --resume are present"
    }`, `"native_resume": 7`))},
		{"quiesce truncated", quiesceErr([]byte(`{"provider_id":`))},
		{"spawn truncated", DecodeSpawnPlan([]byte(`{"argv":`), "codex", ProfileYOLO, scalar.PlatformLinux)},
		{"spawn literals array", DecodeSpawnPlan(spawnVariant(t, `"env_literals": {}`, `"env_literals": []`), "codex", ProfileYOLO, scalar.PlatformLinux)},
		{"identity truncated", CheckIdentity([]byte(`{"schema":`), "antigravity")},
		{"identity opaque array", CheckIdentity(identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": []`), "antigravity")},
		{"identify truncated", DecodeIdentifyResult([]byte(`{"identity":`), "antigravity")},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			if failureCode(t, row.err) != "provider_protocol_error" {
				t.Fatalf("code = %v, want provider_protocol_error", row.err)
			}
			if failureObject(t, row.err).Retryable() {
				t.Fatalf("malformed frame surfaces retryable: %v", row.err)
			}
		})
	}
}
