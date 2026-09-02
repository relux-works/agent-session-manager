package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const maxConfigExtensionBytes = 65_536

var (
	logicalRootPattern       = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	terminalBackendIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	semverPattern            = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	reverseDNSPattern        = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}(?:\.[a-z][a-z0-9-]{0,62})+$`)
)

var terminalCapabilitySet = func() map[string]struct{} {
	values := make(map[string]struct{})
	for _, capability := range catalog.Current().Capabilities {
		if capability.Family == catalog.Family("terminal_backend") {
			values[string(capability.Name)] = struct{}{}
		}
	}
	if len(values) == 0 {
		panic("pinned catalog has no terminal_backend capabilities")
	}
	return values
}()

func translateV1(raw rawV1, context DecodeContext) (Configuration, error) {
	base, err := translateCommon(rawCommon{
		Schema: raw.Schema, SchemaVersion: raw.SchemaVersion, HostID: raw.HostID,
		HostName: raw.HostName, Platform: raw.Platform, Mesh: raw.Mesh,
		WorkspaceRoots: raw.WorkspaceRoots, Providers: raw.Providers, Sync: raw.Sync,
		Service: raw.Service, Restore: raw.Restore, Profiles: raw.Profiles,
	}, context)
	if err != nil {
		return Configuration{}, err
	}
	if err := validateLegacyTerminalBackend(raw.Terminal.Backend); err != nil {
		return Configuration{}, err
	}
	base.Terminal = translateLegacyTerminal(raw.Terminal, base.Platform)
	base.Directory = defaultDirectory()
	if err := validateConfiguration(&base, context); err != nil {
		return Configuration{}, err
	}
	return base, nil
}

func translateV2(raw rawV2, context DecodeContext) (Configuration, error) {
	if err := validateRawDirectoryPresence(raw.DirectoryInstallations, raw.DirectoryEnrichmentProfiles, raw.DirectoryPeerDisclosure); err != nil {
		return Configuration{}, err
	}
	base, err := translateCommon(rawCommon{
		Schema: raw.Schema, SchemaVersion: raw.SchemaVersion, HostID: raw.HostID,
		HostName: raw.HostName, Platform: raw.Platform, Mesh: raw.Mesh,
		WorkspaceRoots: raw.WorkspaceRoots, Providers: raw.Providers, Sync: raw.Sync,
		Service: raw.Service, Restore: raw.Restore, Profiles: raw.Profiles,
	}, context)
	if err != nil {
		return Configuration{}, err
	}
	if err := validateLegacyTerminalBackend(raw.Terminal.Backend); err != nil {
		return Configuration{}, err
	}
	base.Terminal = translateLegacyTerminal(raw.Terminal, base.Platform)
	translateDirectory(&base, raw.Directory, raw.DirectoryInstallations, raw.DirectoryEnrichmentProfiles, raw.DirectoryPeerDisclosure)
	if err := validateConfiguration(&base, context); err != nil {
		return Configuration{}, err
	}
	return base, nil
}

func translateV3(raw rawV3, context DecodeContext) (Configuration, error) {
	if err := validateRawDirectoryPresence(raw.DirectoryInstallations, raw.DirectoryEnrichmentProfiles, raw.DirectoryPeerDisclosure); err != nil {
		return Configuration{}, err
	}
	if err := validateRawTerminalPresence(raw.Terminal); err != nil {
		return Configuration{}, err
	}
	base, err := translateCommon(rawCommon{
		Schema: raw.Schema, SchemaVersion: raw.SchemaVersion, HostID: raw.HostID,
		HostName: raw.HostName, Platform: raw.Platform, Mesh: raw.Mesh,
		WorkspaceRoots: raw.WorkspaceRoots, Providers: raw.Providers, Sync: raw.Sync,
		Service: raw.Service, Restore: raw.Restore, Profiles: raw.Profiles,
	}, context)
	if err != nil {
		return Configuration{}, err
	}
	base.Terminal = translateTerminal(raw.Terminal, base.Platform)
	translateDirectory(&base, raw.Directory, raw.DirectoryInstallations, raw.DirectoryEnrichmentProfiles, raw.DirectoryPeerDisclosure)
	if err := validateConfiguration(&base, context); err != nil {
		return Configuration{}, err
	}
	return base, nil
}

func validateRawDirectoryPresence(installations []rawDirectoryInstallation, profiles []rawDirectoryEnrichmentProfile, disclosures []rawDirectoryPeerDisclosure) error {
	for index, entry := range installations {
		if entry.InstallationID == nil || entry.EnvironmentID == nil || entry.ProviderID == nil || entry.AdapterID == nil || entry.ScanRootAuthorityIDs == nil || entry.Enabled == nil || entry.Extensions == nil {
			return configError(fmt.Sprintf("directory_installations[%d] required member", index), ErrConfigValidation)
		}
	}
	for index, entry := range profiles {
		if entry.ProfileID == nil || entry.Enabled == nil || entry.MaxConcurrency == nil || entry.MetadataPolicy == nil || entry.Extensions == nil {
			return configError(fmt.Sprintf("directory_enrichment_profiles[%d] required member", index), ErrConfigValidation)
		}
	}
	for index, entry := range disclosures {
		if entry.HostID == nil || entry.EnvironmentObservations == nil || entry.NativeObservations == nil || entry.ManualMetadata == nil || entry.GeneratedMetadata == nil || entry.JobOperationStatus == nil || entry.Extensions == nil {
			return configError(fmt.Sprintf("directory_peer_disclosure[%d] required member", index), ErrConfigValidation)
		}
	}
	return nil
}

func validateRawTerminalPresence(terminal rawTerminal) error {
	for index, entry := range terminal.ExternalTrust {
		if entry.BackendID == nil || entry.ExecutablePath == nil || entry.ExecutableDigest == nil || entry.Enabled == nil {
			return configError(fmt.Sprintf("terminal.external_trust[%d] required member", index), ErrConfigValidation)
		}
	}
	for index, entry := range terminal.BackendConfig {
		if entry.BackendID == nil || entry.ConfigVersion == nil || entry.Settings == nil {
			return configError(fmt.Sprintf("terminal.backend_config[%d] required member", index), ErrConfigValidation)
		}
	}
	return nil
}

func translateCommon(raw rawCommon, context DecodeContext) (Configuration, error) {
	if raw.HostID == nil || raw.HostName == nil || raw.Platform == nil {
		return Configuration{}, configError("required root identity", ErrConfigValidation)
	}
	platform, err := scalar.ParsePlatform(*raw.Platform)
	if err != nil {
		return Configuration{}, configError("platform", errors.Join(ErrConfigValidation, err))
	}
	configuration := Configuration{
		Schema: SchemaID, SchemaVersion: CurrentVersion,
		HostID: *raw.HostID, HostName: *raw.HostName, Platform: platform,
		Mesh: Mesh{
			Transport:             valueOr(raw.Mesh.Transport, "ssh"),
			SyncIntervalSeconds:   valueOr(raw.Mesh.SyncIntervalSeconds, uint64(60)),
			ConnectTimeoutSeconds: valueOr(raw.Mesh.ConnectTimeoutSeconds, uint64(10)),
			RPCTimeoutSeconds:     valueOr(raw.Mesh.RPCTimeoutSeconds, uint64(300)),
			WorkspaceReplication:  valueOr(raw.Mesh.WorkspaceReplication, true),
			PayloadEncryption:     valueOr(raw.Mesh.PayloadEncryption, "none"),
		},
		Providers: Providers{
			AllowPathPlugins:     valueOr(raw.Providers.AllowPathPlugins, true),
			RequireExplicitTrust: valueOr(raw.Providers.RequireExplicitTrust, true),
		},
		Sync: Sync{
			ChunkBytes:                valueOr(raw.Sync.ChunkBytes, uint64(4_194_304)),
			MaxParallelChunks:         valueOr(raw.Sync.MaxParallelChunks, uint64(4)),
			StagingRetentionHours:     valueOr(raw.Sync.StagingRetentionHours, uint64(72)),
			TombstoneMinRetentionDays: valueOr(raw.Sync.TombstoneMinRetentionDays, uint64(90)),
		},
		Service: Service{
			Enabled:               valueOr(raw.Service.Enabled, true),
			HealthIntervalSeconds: valueOr(raw.Service.HealthIntervalSeconds, uint64(30)),
		},
		Restore:  Restore{AutoResume: valueOr(raw.Restore.AutoResume, false)},
		Profiles: Profiles{Yolo: YoloProfile{RequireFirstUseConfirmation: valueOr(raw.Profiles.Yolo.RequireFirstUseConfirmation, true)}},
	}
	configuration.Mesh.Peers = make([]Peer, len(raw.Mesh.Peers))
	for index, peer := range raw.Mesh.Peers {
		if peer.HostID == nil || peer.Name == nil || peer.Endpoint == nil || peer.Platform == nil {
			return Configuration{}, configError(fmt.Sprintf("mesh.peers[%d] required member", index), ErrConfigValidation)
		}
		peerPlatform, parseErr := scalar.ParsePlatform(*peer.Platform)
		if parseErr != nil {
			return Configuration{}, configError(fmt.Sprintf("mesh.peers[%d].platform", index), errors.Join(ErrConfigValidation, parseErr))
		}
		configuration.Mesh.Peers[index] = Peer{
			HostID: *peer.HostID, Name: *peer.Name, Endpoint: *peer.Endpoint,
			Platform: peerPlatform, SSHArgs: cloneStrings(peer.SSHArgs),
			WorkspaceRoots: translateWorkspaceRoots(peer.WorkspaceRoots),
		}
	}
	configuration.WorkspaceRoots = translateWorkspaceRoots(raw.WorkspaceRoots)
	configuration.Providers.PluginDirs = cloneStrings(raw.Providers.PluginDirs)
	return configuration, nil
}

func translateWorkspaceRoots(raw []rawWorkspaceRoot) []WorkspaceRoot {
	values := make([]WorkspaceRoot, len(raw))
	for index, root := range raw {
		if root.LogicalRoot != nil {
			values[index].LogicalRoot = *root.LogicalRoot
		}
		if root.Path != nil {
			values[index].Path = *root.Path
		}
	}
	return values
}

func translateLegacyTerminal(raw rawLegacyTerminal, platform scalar.Platform) Terminal {
	backend := valueOr(raw.Backend, legacyBackendDefault(platform))
	if backend == "tmux" {
		backend = "ax.tmux"
	} else if backend == "conpty" {
		backend = "ax.conpty"
	}
	return Terminal{
		BackendID:                  backend,
		SafeBoundaryTimeoutSeconds: valueOr(raw.SafeBoundaryTimeoutSeconds, uint64(300)),
		GracefulStopTimeoutSeconds: valueOr(raw.GracefulStopTimeoutSeconds, uint64(60)),
		MultipleInputPolicy:        "deny",
		TransportPolicy:            []string{"local_only", "trusted_private_mesh"},
	}
}

func validateLegacyTerminalBackend(backend *string) error {
	if backend != nil && !oneOf(*backend, "tmux", "conpty") {
		return configError("terminal.backend", ErrConfigValidation)
	}
	return nil
}

func translateTerminal(raw rawTerminal, platform scalar.Platform) Terminal {
	requiredCapabilities := []string(nil)
	if raw.RequiredCapabilities != nil {
		requiredCapabilities = cloneStrings(*raw.RequiredCapabilities)
	}
	transportPolicy := []string{"local_only", "trusted_private_mesh"}
	if raw.TransportPolicy != nil {
		transportPolicy = cloneStrings(*raw.TransportPolicy)
	}
	return Terminal{
		BackendID:                    valueOr(raw.BackendID, currentBackendDefault(platform)),
		SafeBoundaryTimeoutSeconds:   valueOr(raw.SafeBoundaryTimeoutSeconds, uint64(300)),
		GracefulStopTimeoutSeconds:   valueOr(raw.GracefulStopTimeoutSeconds, uint64(60)),
		RequiredCapabilities:         requiredCapabilities,
		RequiredCapabilitiesExplicit: raw.RequiredCapabilities != nil,
		MultipleInputPolicy:          valueOr(raw.MultipleInputPolicy, "deny"),
		TransportPolicy:              transportPolicy,
		TransportPolicyExplicit:      raw.TransportPolicy != nil,
		ExternalTrust:                translateExternalTrust(raw.ExternalTrust),
		BackendConfig:                translateBackendConfig(raw.BackendConfig),
	}
}

func translateExternalTrust(raw []rawExternalExecutableTrust) []ExternalExecutableTrust {
	values := make([]ExternalExecutableTrust, len(raw))
	for index, entry := range raw {
		if entry.BackendID != nil {
			values[index].BackendID = *entry.BackendID
		}
		if entry.ExecutablePath != nil {
			values[index].ExecutablePath = *entry.ExecutablePath
		}
		if entry.ExecutableDigest != nil {
			values[index].ExecutableDigest = *entry.ExecutableDigest
		}
		if entry.Enabled != nil {
			values[index].Enabled = *entry.Enabled
		}
	}
	return values
}

func translateBackendConfig(raw []rawBackendConfig) []BackendConfig {
	values := make([]BackendConfig, len(raw))
	for index, entry := range raw {
		if entry.BackendID != nil {
			values[index].BackendID = *entry.BackendID
		}
		if entry.ConfigVersion != nil {
			values[index].ConfigVersion = *entry.ConfigVersion
		}
		values[index].Settings = cloneAnyMap(entry.Settings)
	}
	return values
}

func translateDirectory(configuration *Configuration, raw rawDirectory, installations []rawDirectoryInstallation, profiles []rawDirectoryEnrichmentProfile, disclosures []rawDirectoryPeerDisclosure) {
	configuration.Directory = Directory{
		Enabled: valueOr(raw.Enabled, false), Mode: valueOr(raw.Mode, "on_demand"),
		ScanIntervalSeconds:           valueOr(raw.ScanIntervalSeconds, uint64(300)),
		ScanDebounceSeconds:           valueOr(raw.ScanDebounceSeconds, uint64(5)),
		ScanConcurrency:               valueOr(raw.ScanConcurrency, uint64(2)),
		FreshCurrentSeconds:           valueOr(raw.FreshCurrentSeconds, uint64(120)),
		FreshAgingSeconds:             valueOr(raw.FreshAgingSeconds, uint64(600)),
		FreshStaleSeconds:             valueOr(raw.FreshStaleSeconds, uint64(3600)),
		PlanExpirySeconds:             valueOr(raw.PlanExpirySeconds, uint64(300)),
		DefaultMetadataPolicy:         valueOr(raw.DefaultMetadataPolicy, "local_only"),
		GeneratedSummaryUpgradeChoice: valueOr(raw.GeneratedSummaryUpgradeChoice, "unset"),
		DefaultEnrichmentProfileID:    valueOr(raw.DefaultEnrichmentProfileID, ""),
		QueryPageDefault:              valueOr(raw.QueryPageDefault, uint64(100)),
		QueryPageMax:                  valueOr(raw.QueryPageMax, uint64(1000)),
		QueryBatchMax:                 valueOr(raw.QueryBatchMax, uint64(64)),
		GrepResultMax:                 valueOr(raw.GrepResultMax, uint64(1000)),
		TranscriptGrepEnabled:         valueOr(raw.TranscriptGrepEnabled, false),
		EmbeddingIndex:                valueOr(raw.EmbeddingIndex, "disabled"),
		ObservationRetentionDays:      valueOr(raw.ObservationRetentionDays, uint64(365)),
		JobRetentionDays:              valueOr(raw.JobRetentionDays, uint64(180)),
		OperationRetentionDays:        valueOr(raw.OperationRetentionDays, uint64(365)),
		ProvenanceCompaction:          valueOr(raw.ProvenanceCompaction, false),
	}
	configuration.DirectoryInstallations = make([]DirectoryInstallation, len(installations))
	for index, entry := range installations {
		value := &configuration.DirectoryInstallations[index]
		if entry.InstallationID != nil {
			value.InstallationID = *entry.InstallationID
		}
		if entry.EnvironmentID != nil {
			value.EnvironmentID = *entry.EnvironmentID
		}
		if entry.ProviderID != nil {
			value.ProviderID = *entry.ProviderID
		}
		if entry.AdapterID != nil {
			value.AdapterID = *entry.AdapterID
		}
		value.ScanRootAuthorityIDs = cloneStrings(entry.ScanRootAuthorityIDs)
		if entry.Enabled != nil {
			value.Enabled = *entry.Enabled
		}
		value.Extensions = cloneAnyMap(entry.Extensions)
	}
	configuration.DirectoryEnrichmentProfiles = make([]DirectoryEnrichmentProfile, len(profiles))
	for index, entry := range profiles {
		value := &configuration.DirectoryEnrichmentProfiles[index]
		if entry.ProfileID != nil {
			value.ProfileID = *entry.ProfileID
		}
		if entry.Enabled != nil {
			value.Enabled = *entry.Enabled
		}
		if entry.MaxConcurrency != nil {
			value.MaxConcurrency = *entry.MaxConcurrency
		}
		if entry.MetadataPolicy != nil {
			value.MetadataPolicy = *entry.MetadataPolicy
		}
		value.Extensions = cloneAnyMap(entry.Extensions)
	}
	configuration.DirectoryPeerDisclosure = make([]DirectoryPeerDisclosure, len(disclosures))
	for index, entry := range disclosures {
		value := &configuration.DirectoryPeerDisclosure[index]
		if entry.HostID != nil {
			value.HostID = *entry.HostID
		}
		if entry.EnvironmentObservations != nil {
			value.EnvironmentObservations = *entry.EnvironmentObservations
		}
		if entry.NativeObservations != nil {
			value.NativeObservations = *entry.NativeObservations
		}
		if entry.ManualMetadata != nil {
			value.ManualMetadata = *entry.ManualMetadata
		}
		if entry.GeneratedMetadata != nil {
			value.GeneratedMetadata = *entry.GeneratedMetadata
		}
		if entry.JobOperationStatus != nil {
			value.JobOperationStatus = *entry.JobOperationStatus
		}
		value.Extensions = cloneAnyMap(entry.Extensions)
	}
}

func defaultDirectory() Directory {
	configuration := Configuration{}
	translateDirectory(&configuration, rawDirectory{}, nil, nil, nil)
	return configuration.Directory
}

func validateConfiguration(configuration *Configuration, context DecodeContext) error {
	if _, err := scalar.ParseUUIDv7(configuration.HostID); err != nil {
		return configError("host_id", errors.Join(ErrConfigValidation, err))
	}
	if err := validatePrintableCharacters(configuration.HostName, 1, 64); err != nil {
		return configError("host_name", err)
	}
	if configuration.Platform != context.RuntimePlatform {
		return configError("platform must match runtime probe", ErrConfigValidation)
	}
	if err := validateMesh(configuration); err != nil {
		return err
	}
	if err := validateWorkspaceRoots("workspace_roots", configuration.WorkspaceRoots, configuration.Platform, -1); err != nil {
		return err
	}
	if err := validateProviders(configuration); err != nil {
		return err
	}
	if err := validateSync(configuration.Sync); err != nil {
		return err
	}
	if err := validateDirectory(configuration); err != nil {
		return err
	}
	if err := validateTerminal(configuration, context); err != nil {
		return err
	}
	if !between(configuration.Service.HealthIntervalSeconds, 5, 3600) {
		return configError("service.health_interval_seconds", ErrConfigValidation)
	}
	return nil
}

func validateMesh(configuration *Configuration) error {
	mesh := configuration.Mesh
	if mesh.Transport != "ssh" {
		return configError("mesh.transport", ErrConfigValidation)
	}
	if !between(mesh.SyncIntervalSeconds, 5, 86_400) {
		return configError("mesh.sync_interval_seconds", ErrConfigValidation)
	}
	if !between(mesh.ConnectTimeoutSeconds, 1, 300) {
		return configError("mesh.connect_timeout_seconds", ErrConfigValidation)
	}
	if !between(mesh.RPCTimeoutSeconds, 10, 3_600) {
		return configError("mesh.rpc_timeout_seconds", ErrConfigValidation)
	}
	if mesh.PayloadEncryption != "none" {
		return configError("mesh.payload_encryption", ErrConfigValidation)
	}
	hostIDs, names := map[string]struct{}{}, map[string]struct{}{}
	for index, peer := range mesh.Peers {
		prefix := fmt.Sprintf("mesh.peers[%d]", index)
		if _, err := scalar.ParseUUIDv7(peer.HostID); err != nil {
			return configError(prefix+".host_id", errors.Join(ErrConfigValidation, err))
		}
		if _, exists := hostIDs[peer.HostID]; exists {
			return configError("mesh.peers duplicate host_id", ErrConfigValidation)
		}
		hostIDs[peer.HostID] = struct{}{}
		if err := validatePrintableCharacters(peer.Name, 1, 64); err != nil {
			return configError(prefix+".name", err)
		}
		if _, exists := names[peer.Name]; exists {
			return configError("mesh.peers duplicate name", ErrConfigValidation)
		}
		names[peer.Name] = struct{}{}
		if err := validatePrintableCharacters(peer.Endpoint, 1, 1024); err != nil {
			return configError(prefix+".endpoint", err)
		}
		if reason := admitMeshEndpoint(peer.Endpoint); reason != endpointAdmitted {
			return configError(prefix+".endpoint "+reason, ErrConfigValidation)
		}
		if len(peer.SSHArgs) > 64 {
			return configError(prefix+".ssh_args", ErrConfigValidation)
		}
		totalBytes := len(peer.Endpoint)
		for _, argument := range peer.SSHArgs {
			if !utf8.ValidString(argument) || len(argument) < 1 || len(argument) > 4096 || strings.IndexByte(argument, 0) >= 0 {
				return configError(prefix+".ssh_args", ErrConfigValidation)
			}
			totalBytes += len(argument)
		}
		if totalBytes > 65_536 {
			return configError(prefix+" SSH argv bytes", ErrConfigValidation)
		}
		if reason := admitSSHArguments(peer.SSHArgs); reason != sshArgumentAdmitted {
			return configError(prefix+".ssh_args "+reason, ErrConfigValidation)
		}
		if err := validateWorkspaceRoots(prefix+".workspace_roots", peer.WorkspaceRoots, peer.Platform, 64); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkspaceRoots(prefix string, roots []WorkspaceRoot, platform scalar.Platform, max int) error {
	if max >= 0 && len(roots) > max {
		return configError(prefix, ErrConfigValidation)
	}
	seen := map[string]struct{}{}
	for index, root := range roots {
		if !logicalRootPattern.MatchString(root.LogicalRoot) {
			return configError(fmt.Sprintf("%s[%d].logical_root", prefix, index), ErrConfigValidation)
		}
		if _, exists := seen[root.LogicalRoot]; exists {
			return configError(prefix+" duplicate logical_root", ErrConfigValidation)
		}
		seen[root.LogicalRoot] = struct{}{}
		if _, err := scalar.ParseAbsolutePath(platform, root.Path); err != nil {
			return configError(fmt.Sprintf("%s[%d].path", prefix, index), errors.Join(ErrConfigValidation, err))
		}
	}
	return nil
}

func validateProviders(configuration *Configuration) error {
	for _, directory := range configuration.Providers.PluginDirs {
		if _, err := scalar.ParseAbsolutePath(configuration.Platform, directory); err != nil {
			return configError("providers.plugin_dirs", errors.Join(ErrConfigValidation, err))
		}
	}
	if !configuration.Providers.RequireExplicitTrust {
		return configError("providers.require_explicit_trust", ErrConfigValidation)
	}
	return nil
}

func validateSync(value Sync) error {
	if value.ChunkBytes != 4_194_304 {
		return configError("sync.chunk_bytes", ErrConfigValidation)
	}
	if !between(value.MaxParallelChunks, 1, 32) {
		return configError("sync.max_parallel_chunks", ErrConfigValidation)
	}
	if !between(value.StagingRetentionHours, 1, 720) {
		return configError("sync.staging_retention_hours", ErrConfigValidation)
	}
	if !between(value.TombstoneMinRetentionDays, 90, 3_650) {
		return configError("sync.tombstone_min_retention_days", ErrConfigValidation)
	}
	return nil
}

func validateDirectory(configuration *Configuration) error {
	directory := configuration.Directory
	if !oneOf(directory.Mode, "on_demand", "service") {
		return configError("directory.mode", ErrConfigValidation)
	}
	if !between(directory.ScanIntervalSeconds, 5, 86_400) {
		return configError("directory.scan_interval_seconds", ErrConfigValidation)
	}
	if !between(directory.ScanDebounceSeconds, 0, 3_600) {
		return configError("directory.scan_debounce_seconds", ErrConfigValidation)
	}
	if !between(directory.ScanConcurrency, 1, 32) {
		return configError("directory.scan_concurrency", ErrConfigValidation)
	}
	if !between(directory.FreshCurrentSeconds, 1, 86_400) {
		return configError("directory.fresh_current_seconds", ErrConfigValidation)
	}
	if directory.FreshAgingSeconds <= directory.FreshCurrentSeconds || directory.FreshAgingSeconds > 604_800 {
		return configError("directory.fresh_aging_seconds", ErrConfigValidation)
	}
	if directory.FreshStaleSeconds <= directory.FreshAgingSeconds || directory.FreshStaleSeconds > 31_536_000 {
		return configError("directory.fresh_stale_seconds", ErrConfigValidation)
	}
	if !between(directory.PlanExpirySeconds, 30, 3_600) {
		return configError("directory.plan_expiry_seconds", ErrConfigValidation)
	}
	if !metadataPolicy(directory.DefaultMetadataPolicy) {
		return configError("directory.default_metadata_policy", ErrConfigValidation)
	}
	if !oneOf(directory.GeneratedSummaryUpgradeChoice, "unset", "local_only", "mesh_sanitized", "reference_only") {
		return configError("directory.generated_summary_upgrade_choice", ErrConfigValidation)
	}
	if directory.DefaultEnrichmentProfileID != "" {
		if _, err := scalar.ParseDigest(directory.DefaultEnrichmentProfileID); err != nil {
			return configError("directory.default_enrichment_profile_id", errors.Join(ErrConfigValidation, err))
		}
	}
	if !between(directory.QueryPageDefault, 1, scalar.MaxUint53) || !between(directory.QueryPageMax, 1, scalar.MaxUint53) || directory.QueryPageDefault > directory.QueryPageMax {
		return configError("directory query page bounds", ErrConfigValidation)
	}
	if !between(directory.QueryBatchMax, 1, 64) {
		return configError("directory.query_batch_max", ErrConfigValidation)
	}
	if !between(directory.GrepResultMax, 1, 10_000) {
		return configError("directory.grep_result_max", ErrConfigValidation)
	}
	if !oneOf(directory.EmbeddingIndex, "disabled", "local_only") {
		return configError("directory.embedding_index", ErrConfigValidation)
	}
	if !between(directory.ObservationRetentionDays, 30, 3_650) {
		return configError("directory.observation_retention_days", ErrConfigValidation)
	}
	if !between(directory.JobRetentionDays, 30, 3_650) {
		return configError("directory.job_retention_days", ErrConfigValidation)
	}
	if !between(directory.OperationRetentionDays, 90, 3_650) {
		return configError("directory.operation_retention_days", ErrConfigValidation)
	}
	installationIDs := map[string]struct{}{}
	for index, entry := range configuration.DirectoryInstallations {
		prefix := fmt.Sprintf("directory_installations[%d]", index)
		if _, err := scalar.ParseDigest(entry.InstallationID); err != nil {
			return configError(prefix+".installation_id", errors.Join(ErrConfigValidation, err))
		}
		if _, exists := installationIDs[entry.InstallationID]; exists {
			return configError("directory_installations duplicate installation_id", ErrConfigValidation)
		}
		installationIDs[entry.InstallationID] = struct{}{}
		if err := validatePrintableCharacters(entry.EnvironmentID, 1, 64); err != nil {
			return configError(prefix+".environment_id", err)
		}
		if _, err := scalar.ParseProviderID(entry.ProviderID); err != nil {
			return configError(prefix+".provider_id", errors.Join(ErrConfigValidation, err))
		}
		if err := validatePrintableCharacters(entry.AdapterID, 1, 64); err != nil {
			return configError(prefix+".adapter_id", err)
		}
		if err := validateSortedUniqueDigests(entry.ScanRootAuthorityIDs, 1, 64); err != nil {
			return configError(prefix+".scan_root_authority_ids", err)
		}
		if err := validateExtensions(entry.Extensions); err != nil {
			return configError(prefix+".extensions", err)
		}
	}
	profileIDs := map[string]struct{}{}
	for index, entry := range configuration.DirectoryEnrichmentProfiles {
		prefix := fmt.Sprintf("directory_enrichment_profiles[%d]", index)
		if _, err := scalar.ParseDigest(entry.ProfileID); err != nil {
			return configError(prefix+".profile_id", errors.Join(ErrConfigValidation, err))
		}
		if _, exists := profileIDs[entry.ProfileID]; exists {
			return configError("directory_enrichment_profiles duplicate profile_id", ErrConfigValidation)
		}
		profileIDs[entry.ProfileID] = struct{}{}
		if !between(entry.MaxConcurrency, 1, 32) {
			return configError(prefix+".max_concurrency", ErrConfigValidation)
		}
		if !metadataPolicy(entry.MetadataPolicy) {
			return configError(prefix+".metadata_policy", ErrConfigValidation)
		}
		if err := validateExtensions(entry.Extensions); err != nil {
			return configError(prefix+".extensions", err)
		}
	}
	disclosureIDs := map[string]struct{}{}
	for index, entry := range configuration.DirectoryPeerDisclosure {
		prefix := fmt.Sprintf("directory_peer_disclosure[%d]", index)
		if _, err := scalar.ParseUUIDv7(entry.HostID); err != nil {
			return configError(prefix+".host_id", errors.Join(ErrConfigValidation, err))
		}
		if _, exists := disclosureIDs[entry.HostID]; exists {
			return configError("directory_peer_disclosure duplicate host_id", ErrConfigValidation)
		}
		disclosureIDs[entry.HostID] = struct{}{}
		for field, policy := range map[string]string{
			"environment_observations": entry.EnvironmentObservations, "native_observations": entry.NativeObservations,
			"manual_metadata": entry.ManualMetadata, "generated_metadata": entry.GeneratedMetadata,
			"job_operation_status": entry.JobOperationStatus,
		} {
			if !metadataPolicy(policy) {
				return configError(prefix+"."+field, ErrConfigValidation)
			}
		}
		if err := validateExtensions(entry.Extensions); err != nil {
			return configError(prefix+".extensions", err)
		}
	}
	return nil
}

func validateTerminal(configuration *Configuration, context DecodeContext) error {
	terminal := configuration.Terminal
	if !terminalBackendIDPattern.MatchString(terminal.BackendID) || len(terminal.BackendID) > 128 {
		return configError("terminal.backend_id", ErrConfigValidation)
	}
	if !between(terminal.SafeBoundaryTimeoutSeconds, 1, 3_600) {
		return configError("terminal.safe_boundary_timeout_seconds", ErrConfigValidation)
	}
	if !between(terminal.GracefulStopTimeoutSeconds, 1, 600) {
		return configError("terminal.graceful_stop_timeout_seconds", ErrConfigValidation)
	}
	if err := validateSortedUniqueClosed(terminal.RequiredCapabilities, terminalCapabilitySet); err != nil {
		return configError("terminal.required_capabilities", err)
	}
	if !terminal.RequiredCapabilitiesExplicit && len(terminal.RequiredCapabilities) != 0 {
		return configError("terminal.required_capabilities default provenance", ErrConfigValidation)
	}
	if !oneOf(terminal.MultipleInputPolicy, "deny", "explicit_allow") {
		return configError("terminal.multiple_input_policy", ErrConfigValidation)
	}
	if err := validateSortedUniqueClosed(terminal.TransportPolicy, map[string]struct{}{"local_only": {}, "trusted_private_mesh": {}}); err != nil {
		return configError("terminal.transport_policy", err)
	}
	if !terminal.TransportPolicyExplicit && !equalStrings(terminal.TransportPolicy, []string{"local_only", "trusted_private_mesh"}) {
		return configError("terminal.transport_policy default provenance", ErrConfigValidation)
	}
	if terminal.BackendID == "ax.tmux" && configuration.Platform == scalar.PlatformWindows {
		return configError("terminal.backend_id unsupported platform", ErrConfigValidation)
	}
	if terminal.BackendID == "ax.conpty" && configuration.Platform != scalar.PlatformWindows {
		return configError("terminal.backend_id unsupported platform", ErrConfigValidation)
	}
	registered := map[string]struct{}{"ax.tmux": {}, "ax.conpty": {}}
	trustIDs := map[string]struct{}{}
	for index, entry := range terminal.ExternalTrust {
		prefix := fmt.Sprintf("terminal.external_trust[%d]", index)
		if !terminalBackendIDPattern.MatchString(entry.BackendID) || len(entry.BackendID) > 128 {
			return configError(prefix+".backend_id", ErrConfigValidation)
		}
		if _, exists := trustIDs[entry.BackendID]; exists {
			return configError("terminal.external_trust duplicate backend_id", ErrConfigValidation)
		}
		trustIDs[entry.BackendID] = struct{}{}
		if entry.Enabled {
			registered[entry.BackendID] = struct{}{}
		}
		if _, err := scalar.ParseAbsolutePath(configuration.Platform, entry.ExecutablePath); err != nil {
			return configError(prefix+".executable_path", errors.Join(ErrConfigValidation, err))
		}
		if _, err := scalar.ParseDigest(entry.ExecutableDigest); err != nil {
			return configError(prefix+".executable_digest", errors.Join(ErrConfigValidation, err))
		}
	}
	if _, ok := registered[terminal.BackendID]; !ok {
		return configError("terminal.backend_id is not registered", ErrConfigValidation)
	}
	configIDs := map[string]struct{}{}
	for index, entry := range terminal.BackendConfig {
		prefix := fmt.Sprintf("terminal.backend_config[%d]", index)
		if _, ok := registered[entry.BackendID]; !ok {
			return configError(prefix+".backend_id is not registered", ErrConfigValidation)
		}
		if _, exists := configIDs[entry.BackendID]; exists {
			return configError("terminal.backend_config duplicate backend_id", ErrConfigValidation)
		}
		configIDs[entry.BackendID] = struct{}{}
		if !semverPattern.MatchString(entry.ConfigVersion) {
			return configError(prefix+".config_version", ErrConfigValidation)
		}
		if entry.Settings == nil {
			return configError(prefix+".settings", ErrConfigValidation)
		}
		if context.BackendSettings == nil {
			return configError(prefix+" has no registered settings schema", ErrConfigValidation)
		}
		if err := context.BackendSettings.ValidateBackendSettings(entry.BackendID, entry.ConfigVersion, cloneAnyMap(entry.Settings)); err != nil {
			return configError(prefix+".settings", errors.Join(ErrConfigValidation, err))
		}
	}
	return nil
}

func validateExtensions(extensions map[string]any) error {
	if extensions == nil {
		return ErrConfigValidation
	}
	if len(extensions) > 64 {
		return ErrConfigValidation
	}
	for key, value := range extensions {
		// SPEC.md:345-347 defines an extension key as exactly this grammar:
		// "A reverse-DNS key is 3-253 lowercase ASCII characters, contains at
		// least one dot, and has dot-separated labels matching
		// [a-z][a-z0-9-]{0,62}". It declares no reserved or forbidden label,
		// so no name is refused beyond that grammar and the byte bound.
		// reverseDNSPattern itself makes a.b the shortest accepted namespace,
		// so the three-byte minimum is structurally subsumed by that grammar.
		if len(key) > 253 || !reverseDNSPattern.MatchString(key) {
			return ErrConfigValidation
		}
		if err := validateExtensionValue(value, 0); err != nil {
			return err
		}
	}
	// The bound is on canonical bytes, so it is measured through the one shared
	// canonical measurement rather than through Go's HTML-escaped encoding. See
	// canonicaljson.CanonicalByteLength.
	canonicalBytes, err := canonicaljson.CanonicalByteLength(extensions)
	if err != nil || canonicalBytes > maxConfigExtensionBytes {
		return ErrConfigValidation
	}
	return nil
}

func validateExtensionValue(value any, depth int) error {
	switch typed := value.(type) {
	case nil, bool, string:
		return nil
	case int64:
		if typed < -scalar.MaxSafeInteger || typed > scalar.MaxSafeInteger {
			return ErrConfigValidation
		}
		return nil
	case uint64:
		if typed > scalar.MaxUint53 {
			return ErrConfigValidation
		}
		return nil
	case float64, float32:
		return ErrConfigValidation
	case []any:
		if depth >= 4 {
			return ErrConfigValidation
		}
		for _, item := range typed {
			if err := validateExtensionValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		if depth >= 4 {
			return ErrConfigValidation
		}
		// SPEC.md:347-349 constrains an ExtensionValue object only as
		// "string-keyed object with maximum nesting depth 4". The pinned spec
		// imposes no naming rule inside an extension value, so nested keys are
		// admitted as data and only the depth bound is enforced here.
		for _, item := range typed {
			if err := validateExtensionValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return ErrConfigValidation
	}
}

func validateSortedUniqueDigests(values []string, min, max int) error {
	if len(values) < min || len(values) > max {
		return ErrConfigValidation
	}
	for index, value := range values {
		if _, err := scalar.ParseDigest(value); err != nil {
			return errors.Join(ErrConfigValidation, err)
		}
		if index > 0 && values[index-1] >= value {
			return ErrConfigValidation
		}
	}
	return nil
}

func validateSortedUniqueClosed(values []string, allowed map[string]struct{}) error {
	for index, value := range values {
		if _, ok := allowed[value]; !ok {
			return ErrConfigValidation
		}
		if index > 0 && values[index-1] >= value {
			return ErrConfigValidation
		}
	}
	return nil
}

func validatePrintableCharacters(value string, min, max int) error {
	if !utf8.ValidString(value) {
		return ErrConfigValidation
	}
	count := utf8.RuneCountInString(value)
	if count < min || count > max {
		return ErrConfigValidation
	}
	for _, character := range value {
		if unicode.IsControl(character) || !unicode.IsPrint(character) {
			return ErrConfigValidation
		}
	}
	return nil
}

func metadataPolicy(value string) bool {
	return oneOf(value, "local_only", "mesh_sanitized", "reference_only")
}
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
func between(value, min, max uint64) bool { return value >= min && value <= max }
func legacyBackendDefault(platform scalar.Platform) string {
	if platform == scalar.PlatformWindows {
		return "conpty"
	}
	return "tmux"
}
func currentBackendDefault(platform scalar.Platform) string {
	if platform == scalar.PlatformWindows {
		return "ax.conpty"
	}
	return "ax.tmux"
}
func valueOr[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}
	return *value
}
func cloneStrings(values []string) []string { return append([]string(nil), values...) }

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func cloneAnyMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = cloneAny(item)
	}
	return cloned
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		cloned := make([]any, len(typed))
		for index, item := range typed {
			cloned[index] = cloneAny(item)
		}
		return cloned
	case []string:
		return cloneStrings(typed)
	default:
		return typed
	}
}

func cloneConfiguration(value Configuration) Configuration {
	cloned := value
	cloned.WorkspaceRoots = append([]WorkspaceRoot(nil), value.WorkspaceRoots...)
	cloned.Providers.PluginDirs = cloneStrings(value.Providers.PluginDirs)
	cloned.Mesh.Peers = make([]Peer, len(value.Mesh.Peers))
	for index, peer := range value.Mesh.Peers {
		cloned.Mesh.Peers[index] = peer
		cloned.Mesh.Peers[index].SSHArgs = cloneStrings(peer.SSHArgs)
		cloned.Mesh.Peers[index].WorkspaceRoots = append([]WorkspaceRoot(nil), peer.WorkspaceRoots...)
	}
	cloned.Terminal.RequiredCapabilities = cloneStrings(value.Terminal.RequiredCapabilities)
	cloned.Terminal.TransportPolicy = cloneStrings(value.Terminal.TransportPolicy)
	cloned.Terminal.ExternalTrust = append([]ExternalExecutableTrust(nil), value.Terminal.ExternalTrust...)
	cloned.Terminal.BackendConfig = make([]BackendConfig, len(value.Terminal.BackendConfig))
	for index, entry := range value.Terminal.BackendConfig {
		cloned.Terminal.BackendConfig[index] = entry
		cloned.Terminal.BackendConfig[index].Settings = cloneAnyMap(entry.Settings)
	}
	cloned.DirectoryInstallations = make([]DirectoryInstallation, len(value.DirectoryInstallations))
	for index, entry := range value.DirectoryInstallations {
		cloned.DirectoryInstallations[index] = entry
		cloned.DirectoryInstallations[index].ScanRootAuthorityIDs = cloneStrings(entry.ScanRootAuthorityIDs)
		cloned.DirectoryInstallations[index].Extensions = cloneAnyMap(entry.Extensions)
	}
	cloned.DirectoryEnrichmentProfiles = make([]DirectoryEnrichmentProfile, len(value.DirectoryEnrichmentProfiles))
	for index, entry := range value.DirectoryEnrichmentProfiles {
		cloned.DirectoryEnrichmentProfiles[index] = entry
		cloned.DirectoryEnrichmentProfiles[index].Extensions = cloneAnyMap(entry.Extensions)
	}
	cloned.DirectoryPeerDisclosure = make([]DirectoryPeerDisclosure, len(value.DirectoryPeerDisclosure))
	for index, entry := range value.DirectoryPeerDisclosure {
		cloned.DirectoryPeerDisclosure[index] = entry
		cloned.DirectoryPeerDisclosure[index].Extensions = cloneAnyMap(entry.Extensions)
	}
	return cloned
}
