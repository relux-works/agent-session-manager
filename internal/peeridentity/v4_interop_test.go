package peeridentity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestConfiguration4ConsumesPeerIdentity proves a Configuration 4.0.0
// document flows through the accepted SSH/peer-identity surface unchanged:
// the allowlist resolution, endpoint plan and disclosure policy that this
// package owns keep working on a migrated mesh. The document below is the
// exact Configuration 4.0.0 wire shape with fixture identity.
func TestConfiguration4ConsumesPeerIdentity(t *testing.T) {
	document := `schema = 'urn:ax:schema:config'
schema_version = '4.0.0'
host_id = '0198f4c8-4a10-7b22-8b3c-1234567890ab'
host_name = 'fixture-host'
platform = 'macos'
directory_enrichment_profiles = []
directory_peer_disclosure = []

[mesh]
transport = 'ssh_tls13'
sync_interval_seconds = 60
connect_timeout_seconds = 10
rpc_timeout_seconds = 300
workspace_replication = true
payload_encryption = 'none'

[[mesh.peers]]
host_id = '0198f4c8-7d40-7e55-8e6f-1234567890ab'
name = 'peer'
endpoint = 'peer.example'
platform = 'linux'
ssh_args = ['-o', 'BatchMode=yes']
workspace_roots = []

[mesh.host_channel]
version = '1.0.0'
credential_id = 'sha256:0000000000000000000000000000000000000000000000000000000000000000'

[[workspace_roots]]
logical_root = 'relux'
path = '/Users/test/Developer'

[providers]
plugin_dirs = ['/Users/test/.local/libexec/ax/providers']
allow_path_plugins = true
require_explicit_trust = true

[sync]
chunk_bytes = 4194304
max_parallel_chunks = 4
staging_retention_hours = 72
tombstone_min_retention_days = 90

[terminal]
backend_id = 'ax.tmux'
safe_boundary_timeout_seconds = 300
graceful_stop_timeout_seconds = 60
multiple_input_policy = 'deny'
transport_policy = ['local_only', 'trusted_private_mesh']
external_trust = []
backend_config = []

[service]
enabled = true
health_interval_seconds = 30

[restore]
auto_resume = false

[profiles]
[profiles.yolo]
require_first_use_confirmation = true

[directory]
enabled = false
mode = 'on_demand'
scan_interval_seconds = 300
scan_debounce_seconds = 5
scan_concurrency = 2
fresh_current_seconds = 120
fresh_aging_seconds = 600
fresh_stale_seconds = 3600
plan_expiry_seconds = 300
default_metadata_policy = 'local_only'
generated_summary_upgrade_choice = 'unset'
default_enrichment_profile_id = ''
query_page_default = 100
query_page_max = 1000
query_batch_max = 64
grep_result_max = 1000
transcript_grep_enabled = false
embedding_index = 'disabled'
observation_retention_days = 365
job_retention_days = 180
operation_retention_days = 365
provenance_compaction = false

[[directory_installations]]
installation_id = 'sha256:0000000000000000000000000000000000000000000000000000000000000000'
environment_id = 'local'
provider_id = 'codex'
adapter_id = 'codex-local'
scan_root_authority_ids = ['sha256:0000000000000000000000000000000000000000000000000000000000000000']
enabled = true

[directory_installations.extensions]
[directory_installations.extensions.'works.relux.fixture']
count = 1
`
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(filename, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	inputs := config.Inputs{
		Platform: scalar.PlatformMacOS, HomeDir: directory, TempDir: directory, WorkingDir: directory,
		LookupEnv: func(name string) (string, bool) {
			if name == "AX_CONFIG" {
				return filename, true
			}
			return "", false
		},
		Stat: os.Stat, ReadFile: os.ReadFile,
	}
	snapshot, err := config.Load(inputs, nil)
	if err != nil {
		t.Fatalf("Load(v4) error = %v", err)
	}
	loaded, ok := snapshot.Configuration()
	if !ok || loaded.SourceVersion != config.Version4 {
		t.Fatalf("Load(v4) source = %q ok=%v", loaded.SourceVersion, ok)
	}
	peers, err := FromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("FromSnapshot(v4) error = %v", err)
	}
	if peers.Local().ID != "0198f4c8-4a10-7b22-8b3c-1234567890ab" {
		t.Fatalf("local host = %q", peers.Local().ID)
	}
	target, err := peers.Resolve("0198f4c8-7d40-7e55-8e6f-1234567890ab")
	if err != nil {
		t.Fatalf("Resolve(peer) error = %v", err)
	}
	if target.Host().ID != "0198f4c8-7d40-7e55-8e6f-1234567890ab" {
		t.Fatalf("resolved host = %q", target.Host().ID)
	}
	if !strings.Contains(target.KeyProvenance(), "external_ssh") {
		t.Fatalf("key provenance = %q", target.KeyProvenance())
	}
	if _, err := peers.Resolve("unknown-host"); err == nil {
		t.Fatal("Resolve(unknown) succeeded on a v4 mesh, want refusal")
	}
}
