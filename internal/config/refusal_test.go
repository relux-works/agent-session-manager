package config

import (
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const testPeerID2 = "0198f4c8-9d40-7e55-8e6f-1234567890ab"

func TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args string
	}{
		{name: "separate equals", args: `["-o", "StrictHostKeyChecking=no"]`},
		{name: "strict off alias", args: `["-o", "StrictHostKeyChecking=off"]`},
		{name: "strict off alias combined", args: `["-oStrictHostKeyChecking=off"]`},
		{name: "separate whitespace", args: `["-o", "StrictHostKeyChecking no"]`},
		{name: "separator run", args: `["-o", "StrictHostKeyChecking = no"]`},
		{name: "combined whitespace", args: `["-oStrictHostKeyChecking no"]`},
		{name: "quoted value", args: `["-o", "StrictHostKeyChecking=\"no\""]`},
		{name: "known hosts null device", args: `["-o", "UserKnownHostsFile /dev/null"]`},
		{name: "known hosts none", args: `["-o", "UserKnownHostsFile=none"]`},
		{name: "known hosts tab separator", args: `["-o", "UserKnownHostsFile\t/dev/null"]`},
		{name: "combined known hosts", args: `["-oUserKnownHostsFile=/dev/null"]`},
		{name: "empty global known hosts", args: `["-o", "GlobalKnownHostsFile=\"\""]`},
		{name: "global known hosts none", args: `["-o", "GlobalKnownHostsFile=none"]`},
		{name: "global known hosts null device", args: `["-o", "GlobalKnownHostsFile=/dev/null"]`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			document := append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[mesh.peers]]
host_id = "`+testPeerID+`"
name = "peer"
endpoint = "peer.example"
platform = "linux"
ssh_args = `+test.args+`
workspace_roots = []
`)...)
			_, err := loadConfigDocument(document, scalar.PlatformMacOS, nil)
			requireConfigClause(t, err, "mesh.peers[0].ssh_args host authentication bypass")
		})
	}
}

func TestLoadClosesLegacyTerminalBackendVocabularyBeforeTranslation(t *testing.T) {
	t.Parallel()

	for _, version := range []string{Version1, Version2} {
		version := version
		t.Run(version, func(t *testing.T) {
			document := append(minimalValidConfigVersion(scalar.PlatformMacOS, version), []byte("\n[terminal]\nbackend = \"ax.tmux\"\n")...)
			_, err := loadConfigDocument(document, scalar.PlatformMacOS, nil)
			requireConfigClause(t, err, "terminal.backend")
		})
	}
}

func TestDecodeEnvelopeRefusalClausesAreIndividuallyPinned(t *testing.T) {
	t.Parallel()

	valid := minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion)
	_, err := Decode(valid, DecodeContext{RuntimePlatform: scalar.Platform("darwin")})
	requireConfigClause(t, err, "platform")

	_, err = Decode([]byte("schema ="), DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	requireDocumentClause(t, err, ErrConfigDecode, "TOML")

	wrongSchema := replaceRootString(valid, "schema", "urn:ax:schema:not-config")
	_, err = Decode(wrongSchema, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	requireConfigClause(t, err, "schema")
}

func TestTerminalCapabilityClosureIsDerivedFromPinnedCatalog(t *testing.T) {
	t.Parallel()

	var expected []string
	for _, capability := range catalog.Current().Capabilities {
		if capability.Family == catalog.Family("terminal_backend") {
			expected = append(expected, string(capability.Name))
		}
	}
	sort.Strings(expected)
	if len(expected) == 0 {
		t.Fatal("pinned catalog has no terminal_backend capabilities")
	}
	if len(terminalCapabilitySet) != len(expected) {
		t.Fatalf("production terminal capability set has %d members, pinned catalog requires %d", len(terminalCapabilitySet), len(expected))
	}
	for _, name := range expected {
		if _, ok := terminalCapabilitySet[name]; !ok {
			t.Errorf("pinned terminal capability %q is not enforced by production validation", name)
		}
	}

	configuration := validCurrentConfiguration()
	configuration.Terminal.RequiredCapabilities = expected
	configuration.Terminal.RequiredCapabilitiesExplicit = true
	if _, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS}); err != nil {
		t.Fatalf("EncodeCurrent(all pinned terminal capabilities) error = %v", err)
	}
}

func TestLoadRawRequiredMemberRefusalClauses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		document []byte
		clause   string
	}{
		{
			name:     "root identity",
			document: []byte("schema = \"urn:ax:schema:config\"\nschema_version = \"3.0.0\"\nhost_id = \"" + testHostID + "\"\nplatform = \"macos\"\n"),
			clause:   "required root identity",
		},
		{
			name:     "platform vocabulary",
			document: replaceRootString(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), "platform", "darwin"),
			clause:   "platform",
		},
		{
			name: "peer required member",
			document: append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[mesh.peers]]
host_id = "`+testPeerID+`"
name = "peer"
platform = "linux"
`)...),
			clause: "mesh.peers[0] required member",
		},
		{
			name: "peer platform vocabulary",
			document: append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[mesh.peers]]
host_id = "`+testPeerID+`"
name = "peer"
endpoint = "peer.example"
platform = "darwin"
`)...),
			clause: "mesh.peers[0].platform",
		},
		{
			name: "directory installation required member",
			document: append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[directory_installations]]
installation_id = "`+testDigest+`"
`)...),
			clause: "directory_installations[0] required member",
		},
		{
			name: "directory profile required member",
			document: append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[directory_enrichment_profiles]]
profile_id = "`+testDigest+`"
`)...),
			clause: "directory_enrichment_profiles[0] required member",
		},
		{
			name: "directory disclosure required member",
			document: append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[directory_peer_disclosure]]
host_id = "`+testPeerID+`"
`)...),
			clause: "directory_peer_disclosure[0] required member",
		},
		{
			name: "external trust required member",
			document: append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[terminal.external_trust]]
backend_id = "com.example.term"
`)...),
			clause: "terminal.external_trust[0] required member",
		},
		{
			name: "backend config required member",
			document: append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[terminal.backend_config]]
backend_id = "ax.tmux"
`)...),
			clause: "terminal.backend_config[0] required member",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			_, err := loadConfigDocument(test.document, scalar.PlatformMacOS, nil)
			requireConfigClause(t, err, test.clause)
		})
	}
}

func TestEncodeCurrentRefusalClausesAreIndividuallyPinned(t *testing.T) {
	t.Parallel()

	type refusalCase struct {
		name    string
		clause  string
		mutate  func(*Configuration)
		context DecodeContext
	}
	validProfile := func() DirectoryEnrichmentProfile {
		return DirectoryEnrichmentProfile{ProfileID: testDigest, Enabled: true, MaxConcurrency: 1, MetadataPolicy: "local_only", Extensions: map[string]any{}}
	}
	validDisclosure := func() DirectoryPeerDisclosure {
		return DirectoryPeerDisclosure{
			HostID: testPeerID, EnvironmentObservations: "local_only", NativeObservations: "local_only",
			ManualMetadata: "local_only", GeneratedMetadata: "local_only", JobOperationStatus: "local_only", Extensions: map[string]any{},
		}
	}
	tests := []refusalCase{
		{name: "host UUIDv7", clause: "host_id", mutate: func(value *Configuration) { value.HostID = "not-a-uuid" }},
		{name: "mesh transport", clause: "mesh.transport", mutate: func(value *Configuration) { value.Mesh.Transport = "quic" }},
		{name: "mesh payload encryption", clause: "mesh.payload_encryption", mutate: func(value *Configuration) { value.Mesh.PayloadEncryption = "aes256" }},
		{name: "peer UUIDv7", clause: "mesh.peers[0].host_id", mutate: func(value *Configuration) { value.Mesh.Peers[0].HostID = "not-a-uuid" }},
		{name: "duplicate peer name", clause: "mesh.peers duplicate name", mutate: func(value *Configuration) {
			peer := value.Mesh.Peers[0]
			peer.HostID = testPeerID2
			value.Mesh.Peers = append(value.Mesh.Peers, peer)
		}},
		{name: "duplicate logical root", clause: "workspace_roots duplicate logical_root", mutate: func(value *Configuration) {
			value.WorkspaceRoots = append(value.WorkspaceRoots, value.WorkspaceRoots[0])
		}},
		{name: "plugin directory absolute path", clause: "providers.plugin_dirs", mutate: func(value *Configuration) { value.Providers.PluginDirs = []string{"relative/plugins"} }},
		{name: "explicit provider trust", clause: "providers.require_explicit_trust", mutate: func(value *Configuration) { value.Providers.RequireExplicitTrust = false }},
		{name: "directory mode", clause: "directory.mode", mutate: func(value *Configuration) { value.Directory.Mode = "always" }},
		{name: "directory fresh current maximum", clause: "directory.fresh_current_seconds", mutate: func(value *Configuration) { value.Directory.FreshCurrentSeconds = 86_401 }},
		{name: "directory metadata policy", clause: "directory.default_metadata_policy", mutate: func(value *Configuration) { value.Directory.DefaultMetadataPolicy = "everything" }},
		{name: "directory summary choice", clause: "directory.generated_summary_upgrade_choice", mutate: func(value *Configuration) { value.Directory.GeneratedSummaryUpgradeChoice = "everything" }},
		{name: "directory enrichment digest", clause: "directory.default_enrichment_profile_id", mutate: func(value *Configuration) { value.Directory.DefaultEnrichmentProfileID = "not-a-digest" }},
		{name: "directory embedding index", clause: "directory.embedding_index", mutate: func(value *Configuration) { value.Directory.EmbeddingIndex = "remote" }},
		{name: "installation digest", clause: "directory_installations[0].installation_id", mutate: func(value *Configuration) { value.DirectoryInstallations[0].InstallationID = "not-a-digest" }},
		{name: "duplicate installation", clause: "directory_installations duplicate installation_id", mutate: func(value *Configuration) {
			value.DirectoryInstallations = append(value.DirectoryInstallations, value.DirectoryInstallations[0])
		}},
		{name: "installation provider ID", clause: "directory_installations[0].provider_id", mutate: func(value *Configuration) { value.DirectoryInstallations[0].ProviderID = "INVALID" }},
		{name: "profile digest", clause: "directory_enrichment_profiles[0].profile_id", mutate: func(value *Configuration) {
			profile := validProfile()
			profile.ProfileID = "not-a-digest"
			value.DirectoryEnrichmentProfiles = []DirectoryEnrichmentProfile{profile}
		}},
		{name: "duplicate profile", clause: "directory_enrichment_profiles duplicate profile_id", mutate: func(value *Configuration) {
			profile := validProfile()
			value.DirectoryEnrichmentProfiles = []DirectoryEnrichmentProfile{profile, profile}
		}},
		{name: "profile concurrency", clause: "directory_enrichment_profiles[0].max_concurrency", mutate: func(value *Configuration) {
			profile := validProfile()
			profile.MaxConcurrency = 0
			value.DirectoryEnrichmentProfiles = []DirectoryEnrichmentProfile{profile}
		}},
		{name: "profile metadata policy", clause: "directory_enrichment_profiles[0].metadata_policy", mutate: func(value *Configuration) {
			profile := validProfile()
			profile.MetadataPolicy = "everything"
			value.DirectoryEnrichmentProfiles = []DirectoryEnrichmentProfile{profile}
		}},
		{name: "profile extensions", clause: "directory_enrichment_profiles[0].extensions", mutate: func(value *Configuration) {
			profile := validProfile()
			profile.Extensions = nil
			value.DirectoryEnrichmentProfiles = []DirectoryEnrichmentProfile{profile}
		}},
		{name: "disclosure UUIDv7", clause: "directory_peer_disclosure[0].host_id", mutate: func(value *Configuration) {
			disclosure := validDisclosure()
			disclosure.HostID = "not-a-uuid"
			value.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{disclosure}
		}},
		{name: "duplicate disclosure", clause: "directory_peer_disclosure duplicate host_id", mutate: func(value *Configuration) {
			disclosure := validDisclosure()
			value.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{disclosure, disclosure}
		}},
		{name: "disclosure environment policy", clause: "directory_peer_disclosure[0].environment_observations", mutate: func(value *Configuration) {
			disclosure := validDisclosure()
			disclosure.EnvironmentObservations = "everything"
			value.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{disclosure}
		}},
		{name: "disclosure native policy", clause: "directory_peer_disclosure[0].native_observations", mutate: func(value *Configuration) {
			disclosure := validDisclosure()
			disclosure.NativeObservations = "everything"
			value.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{disclosure}
		}},
		{name: "disclosure manual policy", clause: "directory_peer_disclosure[0].manual_metadata", mutate: func(value *Configuration) {
			disclosure := validDisclosure()
			disclosure.ManualMetadata = "everything"
			value.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{disclosure}
		}},
		{name: "disclosure generated policy", clause: "directory_peer_disclosure[0].generated_metadata", mutate: func(value *Configuration) {
			disclosure := validDisclosure()
			disclosure.GeneratedMetadata = "everything"
			value.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{disclosure}
		}},
		{name: "disclosure job policy", clause: "directory_peer_disclosure[0].job_operation_status", mutate: func(value *Configuration) {
			disclosure := validDisclosure()
			disclosure.JobOperationStatus = "everything"
			value.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{disclosure}
		}},
		{name: "disclosure extensions", clause: "directory_peer_disclosure[0].extensions", mutate: func(value *Configuration) {
			disclosure := validDisclosure()
			disclosure.Extensions = nil
			value.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{disclosure}
		}},
		{name: "terminal backend ID", clause: "terminal.backend_id", mutate: func(value *Configuration) { value.Terminal.BackendID = "INVALID_ID" }},
		{name: "capability default provenance", clause: "terminal.required_capabilities default provenance", mutate: func(value *Configuration) {
			value.Terminal.RequiredCapabilities = []string{"local_attach"}
			value.Terminal.RequiredCapabilitiesExplicit = false
		}},
		{name: "multiple input policy", clause: "terminal.multiple_input_policy", mutate: func(value *Configuration) { value.Terminal.MultipleInputPolicy = "allow_all" }},
		{name: "terminal transport policy", clause: "terminal.transport_policy", mutate: func(value *Configuration) {
			value.Terminal.TransportPolicy = []string{"relay"}
			value.Terminal.TransportPolicyExplicit = true
		}},
		{name: "transport default provenance", clause: "terminal.transport_policy default provenance", mutate: func(value *Configuration) {
			value.Terminal.TransportPolicy = []string{"local_only"}
			value.Terminal.TransportPolicyExplicit = false
		}},
		{name: "conpty platform", clause: "terminal.backend_id unsupported platform", mutate: func(value *Configuration) { value.Terminal.BackendID = "ax.conpty" }},
		{name: "external backend ID", clause: "terminal.external_trust[0].backend_id", mutate: func(value *Configuration) {
			value.Terminal.ExternalTrust = []ExternalExecutableTrust{{BackendID: "INVALID_ID", ExecutablePath: "/opt/backend", ExecutableDigest: testDigest, Enabled: true}}
		}},
		{name: "external executable absolute path", clause: "terminal.external_trust[0].executable_path", mutate: func(value *Configuration) {
			value.Terminal.ExternalTrust = []ExternalExecutableTrust{{BackendID: "com.example.term", ExecutablePath: "relative/backend", ExecutableDigest: testDigest, Enabled: true}}
		}},
		{name: "external executable digest", clause: "terminal.external_trust[0].executable_digest", mutate: func(value *Configuration) {
			value.Terminal.ExternalTrust = []ExternalExecutableTrust{{BackendID: "com.example.term", ExecutablePath: "/opt/backend", ExecutableDigest: "not-a-digest", Enabled: true}}
		}},
		{name: "unregistered selected backend", clause: "terminal.backend_id is not registered", mutate: func(value *Configuration) { value.Terminal.BackendID = "com.example.term" }},
		{name: "unregistered backend config", clause: "terminal.backend_config[0].backend_id is not registered", mutate: func(value *Configuration) {
			value.Terminal.BackendConfig = []BackendConfig{{BackendID: "com.example.term", ConfigVersion: "1.0.0", Settings: map[string]any{}}}
		}},
		{name: "duplicate backend config", clause: "terminal.backend_config duplicate backend_id", context: DecodeContext{RuntimePlatform: scalar.PlatformMacOS, BackendSettings: acceptBackendSettings{}}, mutate: func(value *Configuration) {
			entry := BackendConfig{BackendID: "ax.tmux", ConfigVersion: "1.0.0", Settings: map[string]any{}}
			value.Terminal.BackendConfig = []BackendConfig{entry, entry}
		}},
		{name: "backend config semver", clause: "terminal.backend_config[0].config_version", mutate: func(value *Configuration) {
			value.Terminal.BackendConfig = []BackendConfig{{BackendID: "ax.tmux", ConfigVersion: "v1", Settings: map[string]any{}}}
		}},
		{name: "backend config settings required", clause: "terminal.backend_config[0].settings", mutate: func(value *Configuration) {
			value.Terminal.BackendConfig = []BackendConfig{{BackendID: "ax.tmux", ConfigVersion: "1.0.0", Settings: nil}}
		}},
		{name: "backend settings schema refusal", clause: "terminal.backend_config[0].settings", context: DecodeContext{RuntimePlatform: scalar.PlatformMacOS, BackendSettings: rejectBackendSettings{}}, mutate: func(value *Configuration) {
			value.Terminal.BackendConfig = []BackendConfig{{BackendID: "ax.tmux", ConfigVersion: "1.0.0", Settings: map[string]any{}}}
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			configuration := validCurrentConfiguration()
			test.mutate(&configuration)
			context := test.context
			if context.RuntimePlatform == "" {
				context.RuntimePlatform = scalar.PlatformMacOS
			}
			_, err := EncodeCurrent(configuration, context)
			requireConfigClause(t, err, test.clause)
		})
	}
}

func TestEncodeCurrentPinsPreviouslyCompositeRefusalClauses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		clause string
		mutate func(*Configuration)
	}{
		{name: "host name invalid UTF-8", clause: "host_name", mutate: func(value *Configuration) {
			value.HostName = string([]byte{0xff})
		}},
		{name: "host name control character", clause: "host_name", mutate: func(value *Configuration) {
			value.HostName = "host\nname"
		}},
		{name: "SSH argument invalid UTF-8", clause: "mesh.peers[0].ssh_args", mutate: func(value *Configuration) {
			value.Mesh.Peers[0].SSHArgs = []string{string([]byte{0xff})}
		}},
		{name: "SSH argument NUL", clause: "mesh.peers[0].ssh_args", mutate: func(value *Configuration) {
			value.Mesh.Peers[0].SSHArgs = []string{"arg\x00value"}
		}},
		{name: "extension reverse-DNS namespace", clause: "directory_installations[0].extensions", mutate: func(value *Configuration) {
			value.DirectoryInstallations[0].Extensions = map[string]any{"singlelabel": true}
		}},
		{name: "extension forbidden root name", clause: "directory_installations[0].extensions", mutate: func(value *Configuration) {
			value.DirectoryInstallations[0].Extensions = map[string]any{"works.relux.token": "fixture"}
		}},
		{name: "extension forbidden nested name", clause: "directory_installations[0].extensions", mutate: func(value *Configuration) {
			value.DirectoryInstallations[0].Extensions = map[string]any{"works.relux.fixture": map[string]any{"endpoint": "fixture"}}
		}},
		{name: "extension float", clause: "directory_installations[0].extensions", mutate: func(value *Configuration) {
			value.DirectoryInstallations[0].Extensions = map[string]any{"works.relux.fixture": 1.5}
		}},
		{name: "extension array depth", clause: "directory_installations[0].extensions", mutate: func(value *Configuration) {
			value.DirectoryInstallations[0].Extensions = map[string]any{"works.relux.fixture": []any{[]any{[]any{[]any{[]any{"leaf"}}}}}}
		}},
		{name: "duplicate required capability", clause: "terminal.required_capabilities", mutate: func(value *Configuration) {
			value.Terminal.RequiredCapabilities = []string{"local_attach", "local_attach"}
			value.Terminal.RequiredCapabilitiesExplicit = true
		}},
		{name: "duplicate transport policy", clause: "terminal.transport_policy", mutate: func(value *Configuration) {
			value.Terminal.TransportPolicy = []string{"local_only", "local_only"}
			value.Terminal.TransportPolicyExplicit = true
		}},
		{name: "duplicate scan authority digest", clause: "directory_installations[0].scan_root_authority_ids", mutate: func(value *Configuration) {
			value.DirectoryInstallations[0].ScanRootAuthorityIDs = []string{testDigest, testDigest}
		}},
		{name: "selected backend ID byte bound", clause: "terminal.backend_id", mutate: func(value *Configuration) {
			value.Terminal.BackendID = "a." + strings.Repeat("b", 127)
		}},
		{name: "external trust backend ID byte bound", clause: "terminal.external_trust[0].backend_id", mutate: func(value *Configuration) {
			value.Terminal.ExternalTrust = []ExternalExecutableTrust{{BackendID: "a." + strings.Repeat("b", 127), ExecutablePath: "/opt/backend", ExecutableDigest: testDigest, Enabled: true}}
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			configuration := validCurrentConfiguration()
			test.mutate(&configuration)
			_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
			requireConfigClause(t, err, test.clause)
		})
	}
}

func TestEncodeCurrentRefusesSelectedExternalBackendWhoseOnlyTrustEntryIsDisabled(t *testing.T) {
	t.Parallel()

	configuration := validCurrentConfiguration()
	configuration.Terminal.BackendID = "com.example.term"
	configuration.Terminal.ExternalTrust = []ExternalExecutableTrust{{
		BackendID: "com.example.term", ExecutablePath: "/opt/example/terminal",
		ExecutableDigest: testDigest, Enabled: false,
	}}
	_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	requireConfigClause(t, err, "terminal.backend_id is not registered")
}

func TestEncodeCurrentPreservesExplicitDenyAllAndDisabledUnselectedTrust(t *testing.T) {
	t.Parallel()

	configuration := validCurrentConfiguration()
	configuration.Terminal.TransportPolicy = []string{}
	configuration.Terminal.TransportPolicyExplicit = true
	configuration.Terminal.ExternalTrust = []ExternalExecutableTrust{{
		BackendID: "com.example.term", ExecutablePath: "/opt/example/terminal",
		ExecutableDigest: testDigest, Enabled: false,
	}}
	encoded, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	if err != nil {
		t.Fatalf("EncodeCurrent(explicit empty transport policy) error = %v", err)
	}
	snapshot, err := loadConfigDocument(encoded, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatalf("Load(EncodeCurrent(explicit policy and disabled trust)) error = %v", err)
	}
	loaded, ok := snapshot.Configuration()
	if !ok || !loaded.Value.Terminal.TransportPolicyExplicit || len(loaded.Value.Terminal.TransportPolicy) != 0 {
		t.Fatalf("round-trip transport policy = %#v, present=%v", loaded.Value.Terminal.TransportPolicy, ok)
	}
	if len(loaded.Value.Terminal.ExternalTrust) != 1 || loaded.Value.Terminal.ExternalTrust[0].Enabled {
		t.Fatalf("round-trip external trust = %#v", loaded.Value.Terminal.ExternalTrust)
	}
}

func TestEncodeCurrentReportsTOMLFailureForValidatorAcceptedUnencodableSettings(t *testing.T) {
	t.Parallel()

	configuration := validCurrentConfiguration()
	configuration.Terminal.BackendConfig = []BackendConfig{{
		BackendID: "ax.tmux", ConfigVersion: "1.0.0", Settings: map[string]any{"unsupported": make(chan int)},
	}}
	_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS, BackendSettings: acceptBackendSettings{}})
	requireDocumentClause(t, err, ErrConfigEncode, "TOML")
}

type acceptBackendSettings struct{}

func (acceptBackendSettings) ValidateBackendSettings(string, string, map[string]any) error {
	return nil
}

type rejectBackendSettings struct{}

func (rejectBackendSettings) ValidateBackendSettings(string, string, map[string]any) error {
	return errors.New("fixture settings rejected")
}

func requireConfigClause(t *testing.T, err error, clause string) {
	t.Helper()
	requireDocumentClause(t, err, ErrConfigValidation, clause)
}

func requireDocumentClause(t *testing.T, err, identity error, clause string) {
	t.Helper()
	if !errors.Is(err, identity) {
		t.Fatalf("error = %v, want %v at %q", err, identity, clause)
	}
	var documentError *DocumentError
	if !errors.As(err, &documentError) {
		t.Fatalf("error = %T %v, want DocumentError at %q", err, err, clause)
	}
	if documentError.Clause != clause {
		t.Fatalf("error clause = %q, want %q", documentError.Clause, clause)
	}
}
