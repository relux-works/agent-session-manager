package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	testHostID = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	testPeerID = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	testDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
)

func TestLoadSupportsEveryPinnedConfigurationVersionAndTranslatesLegacyAtProductionEntry(t *testing.T) {
	t.Parallel()

	current := catalog.Current()
	var expected []string
	for _, contract := range current.Contracts {
		if contract.ID == catalog.ContractID(SchemaID) {
			expected = append([]string(nil), contract.Versions...)
			break
		}
	}
	if len(expected) == 0 {
		t.Fatal("pinned catalog has no Configuration contract")
	}
	fixtures := map[string][]byte{
		Version1:       minimalValidConfigVersion(scalar.PlatformMacOS, Version1),
		Version2:       minimalValidConfigVersion(scalar.PlatformMacOS, Version2),
		CurrentVersion: minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion),
	}
	if len(fixtures) != len(expected) {
		t.Fatalf("reader fixture registry has %d versions, pinned catalog requires %v", len(fixtures), expected)
	}
	for _, version := range expected {
		version := version
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			document, ok := fixtures[version]
			if !ok {
				t.Fatalf("pinned Configuration version %s has no production reader fixture", version)
			}
			snapshot, err := loadConfigDocument(document, scalar.PlatformMacOS, nil)
			if err != nil {
				t.Fatalf("Load(%s) error = %v", version, err)
			}
			loaded, ok := snapshot.Configuration()
			if !ok {
				t.Fatalf("Load(%s).Configuration() missing", version)
			}
			if loaded.SourceVersion != version || loaded.Value.SchemaVersion != CurrentVersion {
				t.Fatalf("Load(%s) versions = source %q/current %q", version, loaded.SourceVersion, loaded.Value.SchemaVersion)
			}
			if got := loaded.Value.Terminal.BackendID; got != "ax.tmux" {
				t.Fatalf("Load(%s) legacy/current terminal backend = %q, want ax.tmux", version, got)
			}
			if loaded.Value.Directory.Mode != "on_demand" || loaded.Value.Directory.DefaultMetadataPolicy != "local_only" {
				t.Fatalf("Load(%s) did not apply safe current directory defaults: %#v", version, loaded.Value.Directory)
			}
		})
	}
}

func TestLoadAcceptsExactV2DirectoryAndV3TerminalShapes(t *testing.T) {
	t.Parallel()

	v2 := string(minimalValidConfigVersion(scalar.PlatformMacOS, Version2)) + `
[directory]
enabled = true
mode = "service"
scan_interval_seconds = 5
scan_debounce_seconds = 0
scan_concurrency = 32
fresh_current_seconds = 1
fresh_aging_seconds = 2
fresh_stale_seconds = 3
plan_expiry_seconds = 30
default_metadata_policy = "reference_only"
generated_summary_upgrade_choice = "local_only"
default_enrichment_profile_id = "` + testDigest + `"
query_page_default = 1
query_page_max = 1
query_batch_max = 1
grep_result_max = 1
transcript_grep_enabled = true
embedding_index = "local_only"
observation_retention_days = 30
job_retention_days = 30
operation_retention_days = 90
provenance_compaction = true

[[directory_installations]]
installation_id = "` + testDigest + `"
environment_id = "local-installation"
provider_id = "codex"
adapter_id = "codex-local"
scan_root_authority_ids = ["` + testDigest + `"]
enabled = true
extensions = { "works.relux.fixture" = { count = 1 } }

[[directory_enrichment_profiles]]
profile_id = "` + testDigest + `"
enabled = true
max_concurrency = 1
metadata_policy = "local_only"
extensions = {}

[[directory_peer_disclosure]]
host_id = "` + testPeerID + `"
environment_observations = "mesh_sanitized"
native_observations = "reference_only"
manual_metadata = "mesh_sanitized"
generated_metadata = "local_only"
job_operation_status = "reference_only"
extensions = {}
`
	snapshot, err := loadConfigDocument([]byte(v2), scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatalf("Load(Configuration 2.0.0 exact extension) error = %v", err)
	}
	loaded, _ := snapshot.Configuration()
	if !loaded.Value.Directory.Enabled || len(loaded.Value.DirectoryInstallations) != 1 || len(loaded.Value.DirectoryPeerDisclosure) != 1 {
		t.Fatalf("v2 directory translation = %#v", loaded.Value)
	}

	v3 := string(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion)) + `
[terminal]
backend_id = "com.example.term"
safe_boundary_timeout_seconds = 3600
graceful_stop_timeout_seconds = 600
required_capabilities = ["durable_disconnect", "local_attach"]
multiple_input_policy = "explicit_allow"
transport_policy = ["local_only", "trusted_private_mesh"]

[[terminal.external_trust]]
backend_id = "com.example.term"
executable_path = "/opt/example/terminal"
executable_digest = "` + testDigest + `"
enabled = true

[[terminal.backend_config]]
backend_id = "com.example.term"
config_version = "1.0.0"
settings = { color = "blue" }
`
	snapshot, err = loadConfigDocument([]byte(v3), scalar.PlatformMacOS, exactBackendSettings{})
	if err != nil {
		t.Fatalf("Load(Configuration 3.0.0 exact terminal extension) error = %v", err)
	}
	loaded, _ = snapshot.Configuration()
	if loaded.Value.Terminal.BackendID != "com.example.term" || len(loaded.Value.Terminal.BackendConfig) != 1 {
		t.Fatalf("v3 terminal decode = %#v", loaded.Value.Terminal)
	}
}

func TestLoadRefusesUnknownClosedMembersUnsupportedVersionsAndMalformedReads(t *testing.T) {
	t.Parallel()

	for _, version := range []string{Version1, Version2, CurrentVersion} {
		version := version
		t.Run(version+" unknown root", func(t *testing.T) {
			document := append(minimalValidConfigVersion(scalar.PlatformMacOS, version), []byte("unknown = true\n")...)
			_, err := loadConfigDocument(document, scalar.PlatformMacOS, nil)
			if !errors.Is(err, ErrConfigDecode) {
				t.Fatalf("Load(%s unknown root) error = %v, want ErrConfigDecode", version, err)
			}
		})
	}
	for name, document := range map[string][]byte{
		"unsupported major": minimalValidConfigVersion(scalar.PlatformMacOS, "4.0.0"),
		"unsupported minor": minimalValidConfigVersion(scalar.PlatformMacOS, "3.1.0"),
		"wrong schema":      []byte("schema = \"urn:ax:schema:not-config\"\nschema_version = \"1.0.0\"\n"),
		"duplicate member":  append(minimalValidConfigVersion(scalar.PlatformMacOS, Version1), []byte("host_name = \"second\"\n")...),
		"partial read":      []byte("schema = \"urn:ax:schema:config\"\nschema_version = \"1.0.0\"\nhost_id ="),
	} {
		name, document := name, document
		t.Run(name, func(t *testing.T) {
			_, err := loadConfigDocument(document, scalar.PlatformMacOS, nil)
			if err == nil {
				t.Fatalf("Load(%s) accepted invalid document", name)
			}
		})
	}
	t.Run("platform differs from runtime probe", func(t *testing.T) {
		_, err := loadConfigDocument(minimalValidConfigVersion(scalar.PlatformLinux, CurrentVersion), scalar.PlatformMacOS, nil)
		if !errors.Is(err, ErrConfigValidation) {
			t.Fatalf("Load(platform mismatch) error = %v, want ErrConfigValidation", err)
		}
	})
	t.Run("built-in backend unsupported on platform", func(t *testing.T) {
		document := append(minimalValidConfigVersion(scalar.PlatformWindows, CurrentVersion), []byte("\n[terminal]\nbackend_id = \"ax.tmux\"\n")...)
		_, err := Decode(document, DecodeContext{RuntimePlatform: scalar.PlatformWindows})
		if !errors.Is(err, ErrConfigValidation) {
			t.Fatalf("Load(ax.tmux on Windows) error = %v, want ErrConfigValidation", err)
		}
	})
}

func TestDecodeRenderedErrorsNeverEchoRejectedMachineLocalValues(t *testing.T) {
	t.Parallel()

	const privateValue = "/machine/private/secret-config-root"
	document := append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte("\n[providers]\nplugin_dirs = [1, \""+privateValue+"\"]\n")...)
	_, err := Decode(document, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	if !errors.Is(err, ErrConfigDecode) {
		t.Fatalf("Decode(malformed private value) error = %v, want ErrConfigDecode", err)
	}
	if strings.Contains(err.Error(), privateValue) {
		t.Fatalf("Decode() rendered rejected machine-local value: %v", err)
	}
	var documentError *DocumentError
	if !errors.As(err, &documentError) || documentError.Clause == "" {
		t.Fatalf("Decode() error = %T %v, want static DocumentError clause", err, err)
	}
}

func TestLoadRefusesSecurityBypassesDuplicatesAndUnknownBackendSettings(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"host-key bypass": `
[[mesh.peers]]
host_id = "` + testPeerID + `"
name = "peer"
endpoint = "peer.example"
platform = "linux"
ssh_args = ["-o", "StrictHostKeyChecking=no"]
workspace_roots = []
`,
		"duplicate peer identity": `
[[mesh.peers]]
host_id = "` + testPeerID + `"
name = "peer-a"
endpoint = "a.example"
platform = "linux"
[[mesh.peers]]
host_id = "` + testPeerID + `"
name = "peer-b"
endpoint = "b.example"
platform = "linux"
`,
		"unsorted capabilities": `
[terminal]
required_capabilities = ["local_attach", "durable_disconnect"]
`,
		"duplicate backend trust": `
[terminal]
[[terminal.external_trust]]
backend_id = "com.example.term"
executable_path = "/opt/a"
executable_digest = "` + testDigest + `"
enabled = true
[[terminal.external_trust]]
backend_id = "com.example.term"
executable_path = "/opt/b"
executable_digest = "` + testDigest + `"
enabled = true
`,
		"unregistered backend settings": `
[terminal]
[[terminal.backend_config]]
backend_id = "ax.tmux"
config_version = "1.0.0"
settings = {}
`,
	}
	for name, suffix := range tests {
		name, suffix := name, suffix
		t.Run(name, func(t *testing.T) {
			document := append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(suffix)...)
			_, err := loadConfigDocument(document, scalar.PlatformMacOS, nil)
			if !errors.Is(err, ErrConfigValidation) {
				t.Fatalf("Load(%s) error = %v, want ErrConfigValidation", name, err)
			}
		})
	}
}

func TestConfigurationNumericBoundsAcceptLimitsAndRefusePastLimitsAtProductionEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, table, key string
		min, max         uint64
	}{
		{"mesh sync interval", "mesh", "sync_interval_seconds", 5, 86_400},
		{"mesh connect timeout", "mesh", "connect_timeout_seconds", 1, 300},
		{"mesh RPC timeout", "mesh", "rpc_timeout_seconds", 10, 3_600},
		{"parallel chunks", "sync", "max_parallel_chunks", 1, 32},
		{"staging retention", "sync", "staging_retention_hours", 1, 720},
		{"tombstone retention", "sync", "tombstone_min_retention_days", 90, 3_650},
		{"safe boundary timeout", "terminal", "safe_boundary_timeout_seconds", 1, 3_600},
		{"graceful stop timeout", "terminal", "graceful_stop_timeout_seconds", 1, 600},
		{"service health interval", "service", "health_interval_seconds", 5, 3_600},
		{"directory scan interval", "directory", "scan_interval_seconds", 5, 86_400},
		{"directory scan debounce", "directory", "scan_debounce_seconds", 0, 3_600},
		{"directory scan concurrency", "directory", "scan_concurrency", 1, 32},
		{"directory plan expiry", "directory", "plan_expiry_seconds", 30, 3_600},
		{"directory query batch", "directory", "query_batch_max", 1, 64},
		{"directory grep result", "directory", "grep_result_max", 1, 10_000},
		{"directory observation retention", "directory", "observation_retention_days", 30, 3_650},
		{"directory job retention", "directory", "job_retention_days", 30, 3_650},
		{"directory operation retention", "directory", "operation_retention_days", 90, 3_650},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			for label, value := range map[string]uint64{"min": test.min, "max": test.max} {
				document := configWithScalar(test.table, test.key, fmt.Sprint(value))
				if _, err := loadConfigDocument(document, scalar.PlatformMacOS, nil); err != nil {
					t.Errorf("%s at %s=%d refused: %v", test.name, label, value, err)
				}
			}
			if test.min > 0 {
				document := configWithScalar(test.table, test.key, fmt.Sprint(test.min-1))
				if _, err := loadConfigDocument(document, scalar.PlatformMacOS, nil); !errors.Is(err, ErrConfigValidation) {
					t.Errorf("%s below min error = %v", test.name, err)
				}
			} else {
				document := configWithScalar(test.table, test.key, "-1")
				if _, err := loadConfigDocument(document, scalar.PlatformMacOS, nil); err == nil {
					t.Errorf("%s accepted negative value", test.name)
				}
			}
			document := configWithScalar(test.table, test.key, fmt.Sprint(test.max+1))
			if _, err := loadConfigDocument(document, scalar.PlatformMacOS, nil); !errors.Is(err, ErrConfigValidation) {
				t.Errorf("%s above max error = %v", test.name, err)
			}
		})
	}

	for _, value := range []uint64{4_194_303, 4_194_305} {
		if _, err := loadConfigDocument(configWithScalar("sync", "chunk_bytes", fmt.Sprint(value)), scalar.PlatformMacOS, nil); !errors.Is(err, ErrConfigValidation) {
			t.Errorf("sync.chunk_bytes=%d error = %v", value, err)
		}
	}
	if _, err := loadConfigDocument(configWithScalar("sync", "chunk_bytes", "4194304"), scalar.PlatformMacOS, nil); err != nil {
		t.Errorf("sync.chunk_bytes exact constant refused: %v", err)
	}
}

func TestConfigurationCharacterAndArrayBoundsCountCharactersAndBytesExactly(t *testing.T) {
	t.Parallel()

	for label, name := range map[string]string{
		"64 Unicode characters": strings.Repeat("界", 64),
		"64 ASCII characters":   strings.Repeat("a", 64),
	} {
		document := replaceRootString(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), "host_name", name)
		if _, err := loadConfigDocument(document, scalar.PlatformMacOS, nil); err != nil {
			t.Errorf("host_name %s refused: %v", label, err)
		}
	}
	document := replaceRootString(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), "host_name", strings.Repeat("界", 65))
	if _, err := loadConfigDocument(document, scalar.PlatformMacOS, nil); !errors.Is(err, ErrConfigValidation) {
		t.Errorf("65-character host_name error = %v", err)
	}
	document = replaceRootString(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), "host_name", "")
	if _, err := loadConfigDocument(document, scalar.PlatformMacOS, nil); !errors.Is(err, ErrConfigValidation) {
		t.Errorf("empty host_name error = %v", err)
	}

	peerPrefix := string(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion)) + `
[[mesh.peers]]
host_id = "` + testPeerID + `"
name = "peer"
endpoint = "x"
platform = "linux"
`
	atArgument := peerPrefix + "ssh_args = [\"-i" + strings.Repeat("x", 4094) + "\"]\n"
	if _, err := loadConfigDocument([]byte(atArgument), scalar.PlatformMacOS, nil); err != nil {
		t.Fatalf("4096-byte SSH argument refused: %v", err)
	}
	overArgument := peerPrefix + "ssh_args = [\"-i" + strings.Repeat("x", 4095) + "\"]\n"
	if _, err := loadConfigDocument([]byte(overArgument), scalar.PlatformMacOS, nil); !errors.Is(err, ErrConfigValidation) {
		t.Fatalf("4097-byte SSH argument error = %v", err)
	}

	var capabilities []string
	for _, capability := range catalog.Current().Capabilities {
		if capability.Family == catalog.Family("terminal_backend") {
			capabilities = append(capabilities, string(capability.Name))
		}
	}
	if len(capabilities) == 0 {
		t.Fatal("pinned catalog has no terminal_backend capabilities")
	}
	sort.Strings(capabilities)
	configuration := validCurrentConfiguration()
	configuration.Terminal.RequiredCapabilities = capabilities
	configuration.Terminal.RequiredCapabilitiesExplicit = true
	encoded, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	if err != nil {
		t.Fatalf("EncodeCurrent(all 16 closed capabilities) error = %v", err)
	}
	if _, err := loadConfigDocument(encoded, scalar.PlatformMacOS, nil); err != nil {
		t.Fatalf("Load(all 16 closed capabilities) error = %v", err)
	}
}

func TestCurrentEntriesProveRelationalCollectionAndNestedBoundsInBothDirections(t *testing.T) {
	t.Parallel()

	assertCurrentAccepted := func(t *testing.T, configuration Configuration) {
		t.Helper()
		encoded, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
		if err != nil {
			t.Fatalf("EncodeCurrent(at limit) error = %v", err)
		}
		if _, err := loadConfigDocument(encoded, scalar.PlatformMacOS, nil); err != nil {
			t.Fatalf("Load(EncodeCurrent(at limit)) error = %v", err)
		}
	}
	assertCurrentRefused := func(t *testing.T, configuration Configuration) {
		t.Helper()
		if _, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS}); !errors.Is(err, ErrConfigValidation) {
			t.Fatalf("EncodeCurrent(past limit) error = %v, want ErrConfigValidation", err)
		}
	}

	t.Run("directory freshness ordering and maxima", func(t *testing.T) {
		at := validCurrentConfiguration()
		at.Directory.FreshCurrentSeconds = 86_400
		at.Directory.FreshAgingSeconds = 604_800
		at.Directory.FreshStaleSeconds = 31_536_000
		assertCurrentAccepted(t, at)
		for name, mutate := range map[string]func(*Configuration){
			"current max":    func(value *Configuration) { value.Directory.FreshCurrentSeconds = 86_401 },
			"aging ordering": func(value *Configuration) { value.Directory.FreshAgingSeconds = value.Directory.FreshCurrentSeconds },
			"aging max": func(value *Configuration) {
				value.Directory.FreshAgingSeconds = 604_801
				value.Directory.FreshStaleSeconds = 604_802
			},
			"stale ordering": func(value *Configuration) { value.Directory.FreshStaleSeconds = value.Directory.FreshAgingSeconds },
			"stale max":      func(value *Configuration) { value.Directory.FreshStaleSeconds = 31_536_001 },
		} {
			value := validCurrentConfiguration()
			mutate(&value)
			t.Run(name, func(t *testing.T) { assertCurrentRefused(t, value) })
		}
	})

	t.Run("query page uint53 and ordering", func(t *testing.T) {
		at := validCurrentConfiguration()
		at.Directory.QueryPageDefault = scalar.MaxUint53
		at.Directory.QueryPageMax = scalar.MaxUint53
		assertCurrentAccepted(t, at)
		over := validCurrentConfiguration()
		over.Directory.QueryPageDefault = scalar.MaxUint53 + 1
		over.Directory.QueryPageMax = scalar.MaxUint53 + 1
		assertCurrentRefused(t, over)
		ordered := validCurrentConfiguration()
		ordered.Directory.QueryPageDefault = 2
		ordered.Directory.QueryPageMax = 1
		assertCurrentRefused(t, ordered)
	})

	t.Run("peer SSH argument count and total bytes", func(t *testing.T) {
		at := validCurrentConfiguration()
		at.Mesh.Peers[0].Endpoint = "x"
		at.Mesh.Peers[0].SSHArgs = make([]string, 16)
		for index := range at.Mesh.Peers[0].SSHArgs {
			at.Mesh.Peers[0].SSHArgs[index] = "-i" + strings.Repeat("x", 4094)
		}
		at.Mesh.Peers[0].SSHArgs[0] = "-i" + strings.Repeat("x", 4093)
		assertCurrentAccepted(t, at)
		overBytes := cloneConfiguration(at)
		overBytes.Mesh.Peers[0].SSHArgs[0] += "x"
		assertCurrentRefused(t, overBytes)
		overCount := validCurrentConfiguration()
		overCount.Mesh.Peers[0].SSHArgs = make([]string, 65)
		for index := range overCount.Mesh.Peers[0].SSHArgs {
			overCount.Mesh.Peers[0].SSHArgs[index] = "-q"
		}
		assertCurrentRefused(t, overCount)
	})

	t.Run("peer workspace roots and scan authorities", func(t *testing.T) {
		at := validCurrentConfiguration()
		at.Mesh.Peers[0].WorkspaceRoots = make([]WorkspaceRoot, 64)
		for index := range at.Mesh.Peers[0].WorkspaceRoots {
			at.Mesh.Peers[0].WorkspaceRoots[index] = WorkspaceRoot{LogicalRoot: fmt.Sprintf("r%d", index), Path: fmt.Sprintf("/srv/r%d", index)}
		}
		at.DirectoryInstallations[0].ScanRootAuthorityIDs = makeDigests(64)
		assertCurrentAccepted(t, at)
		overRoots := cloneConfiguration(at)
		overRoots.Mesh.Peers[0].WorkspaceRoots = append(overRoots.Mesh.Peers[0].WorkspaceRoots, WorkspaceRoot{LogicalRoot: "overflow", Path: "/srv/overflow"})
		assertCurrentRefused(t, overRoots)
		overAuthorities := cloneConfiguration(at)
		overAuthorities.DirectoryInstallations[0].ScanRootAuthorityIDs = makeDigests(65)
		assertCurrentRefused(t, overAuthorities)
	})

	t.Run("character bounds on endpoint logical root path and directory IDs", func(t *testing.T) {
		at := validCurrentConfiguration()
		at.Mesh.Peers[0].Name = strings.Repeat("界", 64)
		at.Mesh.Peers[0].Endpoint = strings.Repeat("界", 1024)
		at.WorkspaceRoots[0].LogicalRoot = "a" + strings.Repeat("b", 63)
		at.WorkspaceRoots[0].Path = "/" + strings.Repeat("p", 32_766)
		at.DirectoryInstallations[0].EnvironmentID = strings.Repeat("界", 64)
		at.DirectoryInstallations[0].AdapterID = strings.Repeat("界", 64)
		assertCurrentAccepted(t, at)
		for name, mutate := range map[string]func(*Configuration){
			"peer name":      func(value *Configuration) { value.Mesh.Peers[0].Name = strings.Repeat("界", 65) },
			"endpoint":       func(value *Configuration) { value.Mesh.Peers[0].Endpoint = strings.Repeat("界", 1025) },
			"logical root":   func(value *Configuration) { value.WorkspaceRoots[0].LogicalRoot = "a" + strings.Repeat("b", 64) },
			"absolute path":  func(value *Configuration) { value.WorkspaceRoots[0].Path = "/" + strings.Repeat("p", 32_767) },
			"environment id": func(value *Configuration) { value.DirectoryInstallations[0].EnvironmentID = strings.Repeat("界", 65) },
			"adapter id":     func(value *Configuration) { value.DirectoryInstallations[0].AdapterID = strings.Repeat("界", 65) },
		} {
			value := validCurrentConfiguration()
			mutate(&value)
			t.Run(name, func(t *testing.T) { assertCurrentRefused(t, value) })
		}
	})

	t.Run("terminal backend ID byte bound", func(t *testing.T) {
		at := validCurrentConfiguration()
		backendID := "a." + strings.Repeat("b", 126)
		at.Terminal.BackendID = backendID
		at.Terminal.ExternalTrust = []ExternalExecutableTrust{{BackendID: backendID, ExecutablePath: "/opt/backend", ExecutableDigest: testDigest, Enabled: true}}
		assertCurrentAccepted(t, at)
		over := cloneConfiguration(at)
		over.Terminal.BackendID += "b"
		over.Terminal.ExternalTrust[0].BackendID = over.Terminal.BackendID
		assertCurrentRefused(t, over)
	})

	t.Run("extension entry depth and canonical byte bounds", func(t *testing.T) {
		// relux-works/agent-session-manager-spec@v0.5.0 Section 6 fixes this
		// bound independently of the production implementation constant.
		const specificationExtensionObjectMaxBytes = 65_536

		at := validCurrentConfiguration()
		at.DirectoryInstallations[0].Extensions = make(map[string]any, 64)
		for index := 0; index < 64; index++ {
			at.DirectoryInstallations[0].Extensions[fmt.Sprintf("works.relux.k%d", index)] = int64(index)
		}
		assertCurrentAccepted(t, at)
		overCount := cloneConfiguration(at)
		overCount.DirectoryInstallations[0].Extensions["works.relux.overflow"] = true
		assertCurrentRefused(t, overCount)

		depth := validCurrentConfiguration()
		depth.DirectoryInstallations[0].Extensions = map[string]any{"works.relux.depth": map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": "leaf"}}}}}
		assertCurrentAccepted(t, depth)
		overDepth := cloneConfiguration(depth)
		overDepth.DirectoryInstallations[0].Extensions = map[string]any{"works.relux.depth": map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": map[string]any{"e": "leaf"}}}}}}
		assertCurrentRefused(t, overDepth)

		bytesAt := validCurrentConfiguration()
		key := "works.relux.bytes"
		baseSize, _ := json.Marshal(map[string]any{key: ""})
		payload := strings.Repeat("x", specificationExtensionObjectMaxBytes-len(baseSize))
		bytesAt.DirectoryInstallations[0].Extensions = map[string]any{key: payload}
		encodedJSON, _ := json.Marshal(bytesAt.DirectoryInstallations[0].Extensions)
		if len(encodedJSON) != specificationExtensionObjectMaxBytes {
			t.Fatalf("extension fixture JSON bytes = %d, want SPEC limit %d", len(encodedJSON), specificationExtensionObjectMaxBytes)
		}
		assertCurrentAccepted(t, bytesAt)
		overSize := cloneConfiguration(bytesAt)
		overSize.DirectoryInstallations[0].Extensions[key] = payload + "x"
		assertCurrentRefused(t, overSize)
	})

	t.Run("extension key byte bound", func(t *testing.T) {
		at := validCurrentConfiguration()
		at.DirectoryInstallations[0].Extensions = map[string]any{extensionKey(61): true}
		assertCurrentAccepted(t, at)
		over := validCurrentConfiguration()
		over.DirectoryInstallations[0].Extensions = map[string]any{extensionKey(62): true}
		assertCurrentRefused(t, over)
	})
}

func TestEncodeCurrentRoundTripsCurrentVersionAndReturnsIsolatedSnapshots(t *testing.T) {
	t.Parallel()

	configuration := validCurrentConfiguration()
	encoded, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	if err != nil {
		t.Fatalf("EncodeCurrent() error = %v", err)
	}
	if strings.Contains(string(encoded), "schema_version = \"1.0.0\"") || strings.Contains(string(encoded), "backend = \"tmux\"") {
		t.Fatalf("EncodeCurrent() emitted legacy syntax:\n%s", encoded)
	}
	snapshot, err := loadConfigDocument(encoded, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatalf("Load(EncodeCurrent()) error = %v", err)
	}
	loaded, ok := snapshot.Configuration()
	if !ok || loaded.SourceVersion != CurrentVersion {
		t.Fatalf("round-trip loaded = %#v, %v", loaded, ok)
	}
	first, _ := snapshot.Configuration()
	first.Value.Mesh.Peers[0].SSHArgs[0] = "mutated"
	first.Value.DirectoryInstallations[0].Extensions["works.relux.fixture"] = "mutated"
	second, _ := snapshot.Configuration()
	if second.Value.Mesh.Peers[0].SSHArgs[0] == "mutated" || reflect.DeepEqual(first.Value.DirectoryInstallations[0].Extensions, second.Value.DirectoryInstallations[0].Extensions) {
		t.Fatal("Snapshot.Configuration() exposed mutable internal slices/maps")
	}
}

type exactBackendSettings struct{}

func (exactBackendSettings) ValidateBackendSettings(backendID, version string, settings map[string]any) error {
	if backendID != "com.example.term" || version != "1.0.0" || len(settings) != 1 || settings["color"] != "blue" {
		return errors.New("settings do not match registered com.example.term@1.0.0 shape")
	}
	return nil
}

func loadConfigDocument(document []byte, platform scalar.Platform, settings BackendSettingsValidator) (Snapshot, error) {
	files := newFakeFileSystem()
	configPath := "/config/config.toml"
	environment := map[string]string{"AX_CONFIG": configPath}
	switch platform {
	case scalar.PlatformLinux, scalar.PlatformWSL2:
		environment["XDG_RUNTIME_DIR"] = "/run/user/1000"
	case scalar.PlatformWindows:
		configPath = `C:\config\config.toml`
		environment["AX_CONFIG"] = configPath
		environment["APPDATA"] = `C:\Users\test\AppData\Roaming`
		environment["LOCALAPPDATA"] = `C:\Users\test\AppData\Local`
	}
	files.regular[configPath] = append([]byte(nil), document...)
	inputs := fixtureInputs(platform, environment, files)
	inputs.BackendSettings = settings
	return Load(inputs, nil)
}

func minimalValidConfigVersion(platform scalar.Platform, version string) []byte {
	return []byte(fmt.Sprintf(
		"schema = %q\nschema_version = %q\nhost_id = %q\nhost_name = %q\nplatform = %q\n",
		SchemaID, version, testHostID, "fixture-host", platform.String(),
	))
}

func configWithScalar(table, key, value string) []byte {
	return []byte(string(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion)) + "\n[" + table + "]\n" + key + " = " + value + "\n")
}

func replaceRootString(document []byte, key, value string) []byte {
	lines := strings.Split(string(document), "\n")
	for index, line := range lines {
		if strings.HasPrefix(line, key+" = ") {
			lines[index] = fmt.Sprintf("%s = %q", key, value)
			break
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func validCurrentConfiguration() Configuration {
	configuration := Configuration{
		Schema: SchemaID, SchemaVersion: CurrentVersion, HostID: testHostID,
		HostName: "fixture-host", Platform: scalar.PlatformMacOS,
		Mesh: Mesh{
			Transport: "ssh", SyncIntervalSeconds: 60, ConnectTimeoutSeconds: 10, RPCTimeoutSeconds: 300,
			WorkspaceReplication: true, PayloadEncryption: "none",
			Peers: []Peer{{HostID: testPeerID, Name: "peer", Endpoint: "peer.example", Platform: scalar.PlatformLinux, SSHArgs: []string{"-o", "BatchMode=yes"}}},
		},
		WorkspaceRoots: []WorkspaceRoot{{LogicalRoot: "relux", Path: "/Users/test/Developer"}},
		Providers:      Providers{PluginDirs: []string{"/Users/test/.local/libexec/ax/providers"}, AllowPathPlugins: true, RequireExplicitTrust: true},
		Sync:           Sync{ChunkBytes: 4_194_304, MaxParallelChunks: 4, StagingRetentionHours: 72, TombstoneMinRetentionDays: 90},
		Terminal:       Terminal{BackendID: "ax.tmux", SafeBoundaryTimeoutSeconds: 300, GracefulStopTimeoutSeconds: 60, MultipleInputPolicy: "deny", TransportPolicy: []string{"local_only", "trusted_private_mesh"}, TransportPolicyExplicit: true},
		Service:        Service{Enabled: true, HealthIntervalSeconds: 30}, Restore: Restore{AutoResume: false},
		Profiles: Profiles{Yolo: YoloProfile{RequireFirstUseConfirmation: true}}, Directory: defaultDirectory(),
		DirectoryInstallations: []DirectoryInstallation{{
			InstallationID: testDigest, EnvironmentID: "local", ProviderID: "codex", AdapterID: "codex-local",
			ScanRootAuthorityIDs: []string{testDigest}, Enabled: true, Extensions: map[string]any{"works.relux.fixture": map[string]any{"count": int64(1)}},
		}},
	}
	return configuration
}

func makeDigests(count int) []string {
	values := make([]string, count)
	for index := range values {
		values[index] = fmt.Sprintf("sha256:%064x", index)
	}
	return values
}

func extensionKey(lastLabelLength int) string {
	return strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", lastLabelLength)
}
