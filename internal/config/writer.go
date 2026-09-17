package config

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// EncodeCurrent validates and emits Configuration 3.0.0 TOML. It never writes
// a file or rewrites a legacy document; durable migration is a separate owner.
// A host-channel binding refuses here: emitting v4 content as v3 would
// silently drop the credential binding, and Config-4 readers must not rewrite
// historical files.
func EncodeCurrent(configuration Configuration, context DecodeContext) ([]byte, error) {
	configuration.Schema = SchemaID
	configuration.SchemaVersion = CurrentVersion
	if configuration.Mesh.HostChannel != nil {
		return nil, configError("mesh.host_channel", errors.Join(ErrConfigEncode, ErrConfigValidation))
	}
	if err := validateConfiguration(&configuration, context); err != nil {
		return nil, errors.Join(ErrConfigEncode, err)
	}
	raw := currentWire(configuration)
	var output bytes.Buffer
	encoder := toml.NewEncoder(&output)
	if err := encoder.Encode(raw); err != nil {
		return nil, configError("TOML", errors.Join(ErrConfigEncode, err))
	}
	return output.Bytes(), nil
}

func encodeVersion2(configuration Configuration, context DecodeContext) ([]byte, error) {
	current := currentWire(configuration)
	backend := ""
	switch configuration.Terminal.BackendID {
	case "ax.tmux":
		backend = "tmux"
	case "ax.conpty":
		backend = "conpty"
	default:
		return nil, configError("terminal.backend", errors.Join(ErrConfigEncode, fmt.Errorf("cannot represent backend in Configuration %s", Version2))) // config-refusal-subsumed: Migrate accepts Configuration 1.0.0 as the only v2 source and its closed reader admits only tmux or conpty
	}
	raw := rawV2{
		Schema: SchemaID, SchemaVersion: Version2,
		HostID: current.HostID, HostName: current.HostName, Platform: current.Platform,
		Mesh: current.Mesh, WorkspaceRoots: current.WorkspaceRoots, Providers: current.Providers, Sync: current.Sync,
		Terminal: rawLegacyTerminal{
			Backend: pointer(backend), SafeBoundaryTimeoutSeconds: pointer(configuration.Terminal.SafeBoundaryTimeoutSeconds),
			GracefulStopTimeoutSeconds: pointer(configuration.Terminal.GracefulStopTimeoutSeconds),
		},
		Service: current.Service, Restore: current.Restore, Profiles: current.Profiles, Directory: current.Directory,
		DirectoryInstallations:      current.DirectoryInstallations,
		DirectoryEnrichmentProfiles: current.DirectoryEnrichmentProfiles,
		DirectoryPeerDisclosure:     current.DirectoryPeerDisclosure,
	}
	var output bytes.Buffer
	if err := toml.NewEncoder(&output).Encode(raw); err != nil {
		return nil, configError("TOML", errors.Join(ErrConfigEncode, err)) // config-refusal-subsumed: v2 wire TOML - the v1 production reader supplies only closed scalar, slice, and map-free values to this private v2 wire encoder
	}
	if _, err := Decode(output.Bytes(), context); err != nil {
		return nil, errors.Join(ErrConfigEncode, err) // config-refusal-subsumed: v2 re-read - defence in depth only; the sole production caller is Migrate with a Configuration 1.0.0 source, whose closed reader has already validated every member this wire shape carries, so no valid v1 source can produce a v2 document this re-read refuses
	}
	return output.Bytes(), nil
}

func currentWire(configuration Configuration) rawV3 {
	raw := rawV3{
		Schema: SchemaID, SchemaVersion: CurrentVersion,
		HostID: pointer(configuration.HostID), HostName: pointer(configuration.HostName),
		Platform: pointer(configuration.Platform.String()),
		Mesh: rawMesh{
			Transport: pointer(configuration.Mesh.Transport), SyncIntervalSeconds: pointer(configuration.Mesh.SyncIntervalSeconds),
			ConnectTimeoutSeconds: pointer(configuration.Mesh.ConnectTimeoutSeconds), RPCTimeoutSeconds: pointer(configuration.Mesh.RPCTimeoutSeconds),
			WorkspaceReplication: pointer(configuration.Mesh.WorkspaceReplication), PayloadEncryption: pointer(configuration.Mesh.PayloadEncryption),
		},
		WorkspaceRoots: wireWorkspaceRoots(configuration.WorkspaceRoots),
		Providers: rawProviders{
			PluginDirs: cloneStrings(configuration.Providers.PluginDirs), AllowPathPlugins: pointer(configuration.Providers.AllowPathPlugins),
			RequireExplicitTrust: pointer(configuration.Providers.RequireExplicitTrust),
		},
		Sync: rawSync{
			ChunkBytes: pointer(configuration.Sync.ChunkBytes), MaxParallelChunks: pointer(configuration.Sync.MaxParallelChunks),
			StagingRetentionHours: pointer(configuration.Sync.StagingRetentionHours), TombstoneMinRetentionDays: pointer(configuration.Sync.TombstoneMinRetentionDays),
		},
		Terminal: rawTerminal{
			BackendID: pointer(configuration.Terminal.BackendID), SafeBoundaryTimeoutSeconds: pointer(configuration.Terminal.SafeBoundaryTimeoutSeconds),
			GracefulStopTimeoutSeconds: pointer(configuration.Terminal.GracefulStopTimeoutSeconds),
			MultipleInputPolicy:        pointer(configuration.Terminal.MultipleInputPolicy),
		},
		Service:  rawService{Enabled: pointer(configuration.Service.Enabled), HealthIntervalSeconds: pointer(configuration.Service.HealthIntervalSeconds)},
		Restore:  rawRestore{AutoResume: pointer(configuration.Restore.AutoResume)},
		Profiles: rawProfiles{Yolo: rawYoloProfile{RequireFirstUseConfirmation: pointer(configuration.Profiles.Yolo.RequireFirstUseConfirmation)}},
		Directory: rawDirectory{
			Enabled: pointer(configuration.Directory.Enabled), Mode: pointer(configuration.Directory.Mode),
			ScanIntervalSeconds: pointer(configuration.Directory.ScanIntervalSeconds), ScanDebounceSeconds: pointer(configuration.Directory.ScanDebounceSeconds),
			ScanConcurrency: pointer(configuration.Directory.ScanConcurrency), FreshCurrentSeconds: pointer(configuration.Directory.FreshCurrentSeconds),
			FreshAgingSeconds: pointer(configuration.Directory.FreshAgingSeconds), FreshStaleSeconds: pointer(configuration.Directory.FreshStaleSeconds),
			PlanExpirySeconds: pointer(configuration.Directory.PlanExpirySeconds), DefaultMetadataPolicy: pointer(configuration.Directory.DefaultMetadataPolicy),
			GeneratedSummaryUpgradeChoice: pointer(configuration.Directory.GeneratedSummaryUpgradeChoice), DefaultEnrichmentProfileID: pointer(configuration.Directory.DefaultEnrichmentProfileID),
			QueryPageDefault: pointer(configuration.Directory.QueryPageDefault), QueryPageMax: pointer(configuration.Directory.QueryPageMax),
			QueryBatchMax: pointer(configuration.Directory.QueryBatchMax), GrepResultMax: pointer(configuration.Directory.GrepResultMax),
			TranscriptGrepEnabled: pointer(configuration.Directory.TranscriptGrepEnabled), EmbeddingIndex: pointer(configuration.Directory.EmbeddingIndex),
			ObservationRetentionDays: pointer(configuration.Directory.ObservationRetentionDays), JobRetentionDays: pointer(configuration.Directory.JobRetentionDays),
			OperationRetentionDays: pointer(configuration.Directory.OperationRetentionDays), ProvenanceCompaction: pointer(configuration.Directory.ProvenanceCompaction),
		},
	}
	if configuration.Terminal.RequiredCapabilitiesExplicit {
		values := cloneStrings(configuration.Terminal.RequiredCapabilities)
		raw.Terminal.RequiredCapabilities = &values
	}
	if configuration.Terminal.TransportPolicyExplicit {
		values := cloneStrings(configuration.Terminal.TransportPolicy)
		raw.Terminal.TransportPolicy = &values
	}
	raw.Mesh.Peers = make([]rawPeer, len(configuration.Mesh.Peers))
	for index, peer := range configuration.Mesh.Peers {
		raw.Mesh.Peers[index] = rawPeer{
			HostID: pointer(peer.HostID), Name: pointer(peer.Name), Endpoint: pointer(peer.Endpoint), Platform: pointer(peer.Platform.String()),
			SSHArgs: cloneStrings(peer.SSHArgs), WorkspaceRoots: wireWorkspaceRoots(peer.WorkspaceRoots),
		}
	}
	raw.Terminal.ExternalTrust = make([]rawExternalExecutableTrust, len(configuration.Terminal.ExternalTrust))
	for index, entry := range configuration.Terminal.ExternalTrust {
		raw.Terminal.ExternalTrust[index] = rawExternalExecutableTrust{
			BackendID: pointer(entry.BackendID), ExecutablePath: pointer(entry.ExecutablePath), ExecutableDigest: pointer(entry.ExecutableDigest), Enabled: pointer(entry.Enabled),
		}
	}
	raw.Terminal.BackendConfig = make([]rawBackendConfig, len(configuration.Terminal.BackendConfig))
	for index, entry := range configuration.Terminal.BackendConfig {
		raw.Terminal.BackendConfig[index] = rawBackendConfig{BackendID: pointer(entry.BackendID), ConfigVersion: pointer(entry.ConfigVersion), Settings: cloneAnyMap(entry.Settings)}
	}
	raw.DirectoryInstallations = make([]rawDirectoryInstallation, len(configuration.DirectoryInstallations))
	for index, entry := range configuration.DirectoryInstallations {
		raw.DirectoryInstallations[index] = rawDirectoryInstallation{
			InstallationID: pointer(entry.InstallationID), EnvironmentID: pointer(entry.EnvironmentID), ProviderID: pointer(entry.ProviderID), AdapterID: pointer(entry.AdapterID),
			ScanRootAuthorityIDs: cloneStrings(entry.ScanRootAuthorityIDs), Enabled: pointer(entry.Enabled), Extensions: cloneAnyMap(entry.Extensions),
		}
	}
	raw.DirectoryEnrichmentProfiles = make([]rawDirectoryEnrichmentProfile, len(configuration.DirectoryEnrichmentProfiles))
	for index, entry := range configuration.DirectoryEnrichmentProfiles {
		raw.DirectoryEnrichmentProfiles[index] = rawDirectoryEnrichmentProfile{
			ProfileID: pointer(entry.ProfileID), Enabled: pointer(entry.Enabled), MaxConcurrency: pointer(entry.MaxConcurrency), MetadataPolicy: pointer(entry.MetadataPolicy), Extensions: cloneAnyMap(entry.Extensions),
		}
	}
	raw.DirectoryPeerDisclosure = make([]rawDirectoryPeerDisclosure, len(configuration.DirectoryPeerDisclosure))
	for index, entry := range configuration.DirectoryPeerDisclosure {
		raw.DirectoryPeerDisclosure[index] = rawDirectoryPeerDisclosure{
			HostID: pointer(entry.HostID), EnvironmentObservations: pointer(entry.EnvironmentObservations), NativeObservations: pointer(entry.NativeObservations),
			ManualMetadata: pointer(entry.ManualMetadata), GeneratedMetadata: pointer(entry.GeneratedMetadata), JobOperationStatus: pointer(entry.JobOperationStatus), Extensions: cloneAnyMap(entry.Extensions),
		}
	}
	return raw
}

// EncodeVersion4 validates and emits Configuration 4.0.0 TOML: the retained
// Configuration 3.0.0 wire plus the exact required mesh.transport and closed
// mesh.host_channel table. It never writes a file; durable replacement is an
// explicit ApplyV4 concern. The emitted bytes re-decode to the same value as
// defence in depth.
func EncodeVersion4(configuration Configuration, context DecodeContext) ([]byte, error) {
	configuration.Schema = SchemaID
	configuration.SchemaVersion = Version4
	if configuration.Mesh.HostChannel == nil {
		return nil, configError("mesh.host_channel", errors.Join(ErrConfigEncode, ErrConfigValidation))
	}
	if err := validateConfigurationV4(&configuration, context); err != nil {
		return nil, errors.Join(ErrConfigEncode, err)
	}
	v3 := currentWire(configuration)
	raw := rawV4{
		Schema: v3.Schema, SchemaVersion: Version4,
		HostID: v3.HostID, HostName: v3.HostName, Platform: v3.Platform,
		Mesh: rawMeshV4{
			Transport: v3.Mesh.Transport, SyncIntervalSeconds: v3.Mesh.SyncIntervalSeconds,
			ConnectTimeoutSeconds: v3.Mesh.ConnectTimeoutSeconds, RPCTimeoutSeconds: v3.Mesh.RPCTimeoutSeconds,
			WorkspaceReplication: v3.Mesh.WorkspaceReplication, PayloadEncryption: v3.Mesh.PayloadEncryption,
			Peers: v3.Mesh.Peers,
			HostChannel: &rawHostChannel{
				Version:      pointer(configuration.Mesh.HostChannel.Version),
				CredentialID: pointer(configuration.Mesh.HostChannel.CredentialID),
			},
		},
		WorkspaceRoots: v3.WorkspaceRoots, Providers: v3.Providers, Sync: v3.Sync,
		Terminal: v3.Terminal, Service: v3.Service, Restore: v3.Restore, Profiles: v3.Profiles,
		Directory: v3.Directory, DirectoryInstallations: v3.DirectoryInstallations,
		DirectoryEnrichmentProfiles: v3.DirectoryEnrichmentProfiles, DirectoryPeerDisclosure: v3.DirectoryPeerDisclosure,
	}
	var output bytes.Buffer
	encoder := toml.NewEncoder(&output)
	if err := encoder.Encode(raw); err != nil {
		return nil, configError("TOML", errors.Join(ErrConfigEncode, err)) // config-refusal-subsumed: v4 wire TOML - the preview and the v4 validator admit only closed scalar values to this private v4 wire encoder, mirroring the legacy wire defence
	}
	roundTrip, err := Decode(output.Bytes(), context)
	if err != nil {
		return nil, errors.Join(ErrConfigEncode, err)
	}
	if roundTrip.SourceVersion != Version4 || roundTrip.Value.Mesh.HostChannel == nil ||
		roundTrip.Value.Mesh.HostChannel.Version != configuration.Mesh.HostChannel.Version ||
		roundTrip.Value.Mesh.HostChannel.CredentialID != configuration.Mesh.HostChannel.CredentialID ||
		roundTrip.Value.Mesh.Transport != TransportSSHTLS13 {
		return nil, configError("mesh.host_channel", errors.Join(ErrConfigEncode, ErrConfigValidation)) // config-refusal-subsumed: v4 re-read - defence in depth only; the v4 wire carries only values the closed v4 reader accepted, so no valid Configuration 4.0.0 source can produce a v4 document this re-read refuses
	}
	return output.Bytes(), nil
}

func wireWorkspaceRoots(values []WorkspaceRoot) []rawWorkspaceRoot {
	raw := make([]rawWorkspaceRoot, len(values))
	for index, value := range values {
		raw[index] = rawWorkspaceRoot{LogicalRoot: pointer(value.LogicalRoot), Path: pointer(value.Path)}
	}
	return raw
}

func pointer[T any](value T) *T { return &value }
