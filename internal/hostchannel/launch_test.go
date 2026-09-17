package hostchannel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/hostchannel"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/peeridentity"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const v4Document = `schema = 'urn:ax:schema:config'
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

func loadSnapshot(t *testing.T, document string) config.Snapshot {
	t.Helper()
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(filename, []byte(document), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	snapshot, err := config.Load(config.Inputs{
		Platform: scalar.PlatformMacOS, HomeDir: directory, TempDir: directory, WorkingDir: directory,
		LookupEnv: func(name string) (string, bool) {
			if name == "AX_CONFIG" {
				return filename, true
			}
			return "", false
		},
		Stat: os.Stat, ReadFile: os.ReadFile,
	}, nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return snapshot
}

func TestServeArgvPinsChannelVersion(t *testing.T) {
	if len(hostchannel.ServeArgv) != 6 {
		t.Fatalf("ServeArgv = %q", hostchannel.ServeArgv)
	}
	if !slices.Equal(hostchannel.ServeArgv[:4], []string{"ax", "rpc", "serve", "--stdio"}) {
		t.Fatalf("ServeArgv = %q", hostchannel.ServeArgv)
	}
	if hostchannel.ServeArgv[4] != "--host-channel" || hostchannel.ServeArgv[5] != config.HostChannelVersion {
		t.Fatalf("ServeArgv = %q", hostchannel.ServeArgv)
	}
}

func TestLaunchForConfig(t *testing.T) {
	snapshot := loadSnapshot(t, v4Document)
	argv, err := hostchannel.LaunchForConfig(snapshot, hostB)
	if err != nil {
		t.Fatalf("LaunchForConfig: %v", err)
	}
	want := []string{"-T", "-o", "BatchMode=yes", "peer.example", "ax", "rpc", "serve", "--stdio", "--host-channel", "1.0.0"}
	if !slices.Equal(argv, want) {
		t.Fatalf("argv = %q", argv)
	}
	directory, err := peeridentity.FromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("FromSnapshot: %v", err)
	}
	target, err := directory.Resolve(hostB)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	legacy, err := target.RPCArgv()
	if err != nil {
		t.Fatalf("RPCArgv: %v", err)
	}
	if !slices.Equal(argv, append(slices.Clone(legacy), "--host-channel", "1.0.0")) {
		t.Fatalf("argv = %q, legacy = %q", argv, legacy)
	}
	if _, err := hostchannel.LaunchForConfig(snapshot, hostC); !errors.Is(err, peeridentity.ErrNotAllowlisted) {
		t.Fatalf("unknown selector = %v", err)
	}
}

func TestLaunchForConfigRefusesLegacy(t *testing.T) {
	v3 := `schema = "urn:ax:schema:config"
schema_version = "3.0.0"
host_id = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
host_name = "local"
platform = "macos"

[[mesh.peers]]
host_id = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
name = "peer"
endpoint = "peer.example"
platform = "linux"
`
	snapshot := loadSnapshot(t, v3)
	loaded, ok := snapshot.Configuration()
	if !ok || loaded.SourceVersion != config.CurrentVersion {
		t.Fatalf("source = %q ok=%v", loaded.SourceVersion, ok)
	}
	if _, err := hostchannel.LaunchForConfig(snapshot, hostB); !errors.Is(err, hostchannel.ErrInvalidConfig) {
		t.Fatalf("legacy launch = %v", err)
	}
}

func TestWatchGeneration(t *testing.T) {
	f := newFixture(t)
	snapshot, err := f.storeA.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := hostchannel.WatchGeneration(ctx, f.storeA, snapshot.Generation, 10*time.Millisecond); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled watch = %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	watched := make(chan error, 1)
	go func() {
		watched <- hostchannel.WatchGeneration(ctx, f.storeA, snapshot.Generation, 10*time.Millisecond)
	}()
	issuedC, err := hosttrust.IssueCredential(hostC, time.Now().UTC(), nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	enroll(t, f.storeA, issuedC, hostC, time.Now().UTC())
	select {
	case err := <-watched:
		if !errors.Is(err, hosttrust.ErrStaleGeneration) {
			t.Fatalf("watch = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("watch did not report the commit")
	}
	if err := hostchannel.WatchGeneration(context.Background(), nil, 0, 0); !errors.Is(err, hostchannel.ErrInvalidConfig) {
		t.Fatalf("nil store watch = %v", err)
	}
}
