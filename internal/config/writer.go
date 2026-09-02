package config

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// EncodeCurrent validates and emits Configuration 3.0.0 TOML. It never writes
// a file or rewrites a legacy document; durable migration is a separate owner.
func EncodeCurrent(configuration Configuration, context DecodeContext) ([]byte, error) {
	configuration.Schema = SchemaID
	configuration.SchemaVersion = CurrentVersion
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

func wireWorkspaceRoots(values []WorkspaceRoot) []rawWorkspaceRoot {
	raw := make([]rawWorkspaceRoot, len(values))
	for index, value := range values {
		raw[index] = rawWorkspaceRoot{LogicalRoot: pointer(value.LogicalRoot), Path: pointer(value.Path)}
	}
	return raw
}

func pointer[T any](value T) *T { return &value }
