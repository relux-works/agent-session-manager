package config

import (
	"bytes"
	"errors"
	"reflect"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	SchemaID       = "urn:ax:schema:config"
	Version1       = "1.0.0"
	Version2       = "2.0.0"
	CurrentVersion = "3.0.0"
)

var (
	ErrConfigDecode             = errors.New("configuration document decode failed")
	ErrConfigValidation         = errors.New("configuration document validation failed")
	ErrUnsupportedConfigVersion = errors.New("unsupported configuration version")
	ErrConfigEncode             = errors.New("configuration document encode failed")
)

// BackendSettingsValidator closes a backend_config.settings object for one
// exact backend implementation/config version. A nil validator means that no
// backend-specific settings schemas are registered and therefore every
// backend_config entry is refused.
type BackendSettingsValidator interface {
	ValidateBackendSettings(backendID, configVersion string, settings map[string]any) error
}

// DecodeContext carries facts that are external to the TOML document but are
// required by its exact validation rules.
type DecodeContext struct {
	RuntimePlatform scalar.Platform
	BackendSettings BackendSettingsValidator
}

// LoadedConfiguration is an exact source-version read translated into the
// current in-memory model. SourceVersion remains diagnostic provenance; Value
// always has SchemaVersion CurrentVersion.
type LoadedConfiguration struct {
	SourceVersion string
	Value         Configuration
}

// DocumentError preserves machine-actionable error identity while rendering
// only a static schema clause. Parser/validator details remain unwrap-able and
// cannot echo TOML values, paths, endpoints, or other machine-local data.
type DocumentError struct {
	Clause string
	Err    error
}

func (err *DocumentError) Error() string { return "configuration document rejected at " + err.Clause }
func (err *DocumentError) Unwrap() error { return err.Err }

// Configuration is the current Configuration 3.0.0 logical model.
type Configuration struct {
	Schema                      string
	SchemaVersion               string
	HostID                      string
	HostName                    string
	Platform                    scalar.Platform
	Mesh                        Mesh
	WorkspaceRoots              []WorkspaceRoot
	Providers                   Providers
	Sync                        Sync
	Terminal                    Terminal
	Service                     Service
	Restore                     Restore
	Profiles                    Profiles
	Directory                   Directory
	DirectoryInstallations      []DirectoryInstallation
	DirectoryEnrichmentProfiles []DirectoryEnrichmentProfile
	DirectoryPeerDisclosure     []DirectoryPeerDisclosure
}

type Mesh struct {
	Transport             string
	SyncIntervalSeconds   uint64
	ConnectTimeoutSeconds uint64
	RPCTimeoutSeconds     uint64
	WorkspaceReplication  bool
	PayloadEncryption     string
	Peers                 []Peer
}

type Peer struct {
	HostID         string
	Name           string
	Endpoint       string
	Platform       scalar.Platform
	SSHArgs        []string
	WorkspaceRoots []WorkspaceRoot
}

type WorkspaceRoot struct {
	LogicalRoot string
	Path        string
}

type Providers struct {
	PluginDirs           []string
	AllowPathPlugins     bool
	RequireExplicitTrust bool
}

type Sync struct {
	ChunkBytes                uint64
	MaxParallelChunks         uint64
	StagingRetentionHours     uint64
	TombstoneMinRetentionDays uint64
}

type Service struct {
	Enabled               bool
	HealthIntervalSeconds uint64
}

type Restore struct{ AutoResume bool }

type Profiles struct{ Yolo YoloProfile }

type YoloProfile struct{ RequireFirstUseConfirmation bool }

type Directory struct {
	Enabled                       bool
	Mode                          string
	ScanIntervalSeconds           uint64
	ScanDebounceSeconds           uint64
	ScanConcurrency               uint64
	FreshCurrentSeconds           uint64
	FreshAgingSeconds             uint64
	FreshStaleSeconds             uint64
	PlanExpirySeconds             uint64
	DefaultMetadataPolicy         string
	GeneratedSummaryUpgradeChoice string
	DefaultEnrichmentProfileID    string
	QueryPageDefault              uint64
	QueryPageMax                  uint64
	QueryBatchMax                 uint64
	GrepResultMax                 uint64
	TranscriptGrepEnabled         bool
	EmbeddingIndex                string
	ObservationRetentionDays      uint64
	JobRetentionDays              uint64
	OperationRetentionDays        uint64
	ProvenanceCompaction          bool
}

type DirectoryInstallation struct {
	InstallationID       string
	EnvironmentID        string
	ProviderID           string
	AdapterID            string
	ScanRootAuthorityIDs []string
	Enabled              bool
	Extensions           map[string]any
}

type DirectoryEnrichmentProfile struct {
	ProfileID      string
	Enabled        bool
	MaxConcurrency uint64
	MetadataPolicy string
	Extensions     map[string]any
}

type DirectoryPeerDisclosure struct {
	HostID                  string
	EnvironmentObservations string
	NativeObservations      string
	ManualMetadata          string
	GeneratedMetadata       string
	JobOperationStatus      string
	Extensions              map[string]any
}

type Terminal struct {
	BackendID                    string
	SafeBoundaryTimeoutSeconds   uint64
	GracefulStopTimeoutSeconds   uint64
	RequiredCapabilities         []string
	RequiredCapabilitiesExplicit bool
	MultipleInputPolicy          string
	TransportPolicy              []string
	TransportPolicyExplicit      bool
	ExternalTrust                []ExternalExecutableTrust
	BackendConfig                []BackendConfig
}

type ExternalExecutableTrust struct {
	BackendID        string
	ExecutablePath   string
	ExecutableDigest string
	Enabled          bool
}

type BackendConfig struct {
	BackendID     string
	ConfigVersion string
	Settings      map[string]any
}

type rawEnvelope struct {
	Schema        string `toml:"schema"`
	SchemaVersion string `toml:"schema_version"`
}

type rawCommon struct {
	Schema         string             `toml:"schema"`
	SchemaVersion  string             `toml:"schema_version"`
	HostID         *string            `toml:"host_id"`
	HostName       *string            `toml:"host_name"`
	Platform       *string            `toml:"platform"`
	Mesh           rawMesh            `toml:"mesh"`
	WorkspaceRoots []rawWorkspaceRoot `toml:"workspace_roots"`
	Providers      rawProviders       `toml:"providers"`
	Sync           rawSync            `toml:"sync"`
	Service        rawService         `toml:"service"`
	Restore        rawRestore         `toml:"restore"`
	Profiles       rawProfiles        `toml:"profiles"`
}

type rawV1 struct {
	Schema         string             `toml:"schema"`
	SchemaVersion  string             `toml:"schema_version"`
	HostID         *string            `toml:"host_id"`
	HostName       *string            `toml:"host_name"`
	Platform       *string            `toml:"platform"`
	Mesh           rawMesh            `toml:"mesh"`
	WorkspaceRoots []rawWorkspaceRoot `toml:"workspace_roots"`
	Providers      rawProviders       `toml:"providers"`
	Sync           rawSync            `toml:"sync"`
	Terminal       rawLegacyTerminal  `toml:"terminal"`
	Service        rawService         `toml:"service"`
	Restore        rawRestore         `toml:"restore"`
	Profiles       rawProfiles        `toml:"profiles"`
}

type rawV2 struct {
	Schema                      string                          `toml:"schema"`
	SchemaVersion               string                          `toml:"schema_version"`
	HostID                      *string                         `toml:"host_id"`
	HostName                    *string                         `toml:"host_name"`
	Platform                    *string                         `toml:"platform"`
	Mesh                        rawMesh                         `toml:"mesh"`
	WorkspaceRoots              []rawWorkspaceRoot              `toml:"workspace_roots"`
	Providers                   rawProviders                    `toml:"providers"`
	Sync                        rawSync                         `toml:"sync"`
	Terminal                    rawLegacyTerminal               `toml:"terminal"`
	Service                     rawService                      `toml:"service"`
	Restore                     rawRestore                      `toml:"restore"`
	Profiles                    rawProfiles                     `toml:"profiles"`
	Directory                   rawDirectory                    `toml:"directory"`
	DirectoryInstallations      []rawDirectoryInstallation      `toml:"directory_installations"`
	DirectoryEnrichmentProfiles []rawDirectoryEnrichmentProfile `toml:"directory_enrichment_profiles"`
	DirectoryPeerDisclosure     []rawDirectoryPeerDisclosure    `toml:"directory_peer_disclosure"`
}

type rawV3 struct {
	Schema                      string                          `toml:"schema"`
	SchemaVersion               string                          `toml:"schema_version"`
	HostID                      *string                         `toml:"host_id"`
	HostName                    *string                         `toml:"host_name"`
	Platform                    *string                         `toml:"platform"`
	Mesh                        rawMesh                         `toml:"mesh"`
	WorkspaceRoots              []rawWorkspaceRoot              `toml:"workspace_roots"`
	Providers                   rawProviders                    `toml:"providers"`
	Sync                        rawSync                         `toml:"sync"`
	Terminal                    rawTerminal                     `toml:"terminal"`
	Service                     rawService                      `toml:"service"`
	Restore                     rawRestore                      `toml:"restore"`
	Profiles                    rawProfiles                     `toml:"profiles"`
	Directory                   rawDirectory                    `toml:"directory"`
	DirectoryInstallations      []rawDirectoryInstallation      `toml:"directory_installations"`
	DirectoryEnrichmentProfiles []rawDirectoryEnrichmentProfile `toml:"directory_enrichment_profiles"`
	DirectoryPeerDisclosure     []rawDirectoryPeerDisclosure    `toml:"directory_peer_disclosure"`
}

type rawMesh struct {
	Transport             *string   `toml:"transport"`
	SyncIntervalSeconds   *uint64   `toml:"sync_interval_seconds"`
	ConnectTimeoutSeconds *uint64   `toml:"connect_timeout_seconds"`
	RPCTimeoutSeconds     *uint64   `toml:"rpc_timeout_seconds"`
	WorkspaceReplication  *bool     `toml:"workspace_replication"`
	PayloadEncryption     *string   `toml:"payload_encryption"`
	Peers                 []rawPeer `toml:"peers"`
}

type rawPeer struct {
	HostID         *string            `toml:"host_id"`
	Name           *string            `toml:"name"`
	Endpoint       *string            `toml:"endpoint"`
	Platform       *string            `toml:"platform"`
	SSHArgs        []string           `toml:"ssh_args"`
	WorkspaceRoots []rawWorkspaceRoot `toml:"workspace_roots"`
}

type rawWorkspaceRoot struct {
	LogicalRoot *string `toml:"logical_root"`
	Path        *string `toml:"path"`
}

type rawProviders struct {
	PluginDirs           []string `toml:"plugin_dirs"`
	AllowPathPlugins     *bool    `toml:"allow_path_plugins"`
	RequireExplicitTrust *bool    `toml:"require_explicit_trust"`
}

type rawSync struct {
	ChunkBytes                *uint64 `toml:"chunk_bytes"`
	MaxParallelChunks         *uint64 `toml:"max_parallel_chunks"`
	StagingRetentionHours     *uint64 `toml:"staging_retention_hours"`
	TombstoneMinRetentionDays *uint64 `toml:"tombstone_min_retention_days"`
}

type rawLegacyTerminal struct {
	Backend                    *string `toml:"backend"`
	SafeBoundaryTimeoutSeconds *uint64 `toml:"safe_boundary_timeout_seconds"`
	GracefulStopTimeoutSeconds *uint64 `toml:"graceful_stop_timeout_seconds"`
}

type rawTerminal struct {
	BackendID                  *string                      `toml:"backend_id"`
	SafeBoundaryTimeoutSeconds *uint64                      `toml:"safe_boundary_timeout_seconds"`
	GracefulStopTimeoutSeconds *uint64                      `toml:"graceful_stop_timeout_seconds"`
	RequiredCapabilities       *[]string                    `toml:"required_capabilities"`
	MultipleInputPolicy        *string                      `toml:"multiple_input_policy"`
	TransportPolicy            *[]string                    `toml:"transport_policy"`
	ExternalTrust              []rawExternalExecutableTrust `toml:"external_trust"`
	BackendConfig              []rawBackendConfig           `toml:"backend_config"`
}

type rawExternalExecutableTrust struct {
	BackendID        *string `toml:"backend_id"`
	ExecutablePath   *string `toml:"executable_path"`
	ExecutableDigest *string `toml:"executable_digest"`
	Enabled          *bool   `toml:"enabled"`
}

type rawBackendConfig struct {
	BackendID     *string        `toml:"backend_id"`
	ConfigVersion *string        `toml:"config_version"`
	Settings      map[string]any `toml:"settings"`
}

type rawService struct {
	Enabled               *bool   `toml:"enabled"`
	HealthIntervalSeconds *uint64 `toml:"health_interval_seconds"`
}

type rawRestore struct {
	AutoResume *bool `toml:"auto_resume"`
}

type rawProfiles struct {
	Yolo rawYoloProfile `toml:"yolo"`
}

type rawYoloProfile struct {
	RequireFirstUseConfirmation *bool `toml:"require_first_use_confirmation"`
}

type rawDirectory struct {
	Enabled                       *bool   `toml:"enabled"`
	Mode                          *string `toml:"mode"`
	ScanIntervalSeconds           *uint64 `toml:"scan_interval_seconds"`
	ScanDebounceSeconds           *uint64 `toml:"scan_debounce_seconds"`
	ScanConcurrency               *uint64 `toml:"scan_concurrency"`
	FreshCurrentSeconds           *uint64 `toml:"fresh_current_seconds"`
	FreshAgingSeconds             *uint64 `toml:"fresh_aging_seconds"`
	FreshStaleSeconds             *uint64 `toml:"fresh_stale_seconds"`
	PlanExpirySeconds             *uint64 `toml:"plan_expiry_seconds"`
	DefaultMetadataPolicy         *string `toml:"default_metadata_policy"`
	GeneratedSummaryUpgradeChoice *string `toml:"generated_summary_upgrade_choice"`
	DefaultEnrichmentProfileID    *string `toml:"default_enrichment_profile_id"`
	QueryPageDefault              *uint64 `toml:"query_page_default"`
	QueryPageMax                  *uint64 `toml:"query_page_max"`
	QueryBatchMax                 *uint64 `toml:"query_batch_max"`
	GrepResultMax                 *uint64 `toml:"grep_result_max"`
	TranscriptGrepEnabled         *bool   `toml:"transcript_grep_enabled"`
	EmbeddingIndex                *string `toml:"embedding_index"`
	ObservationRetentionDays      *uint64 `toml:"observation_retention_days"`
	JobRetentionDays              *uint64 `toml:"job_retention_days"`
	OperationRetentionDays        *uint64 `toml:"operation_retention_days"`
	ProvenanceCompaction          *bool   `toml:"provenance_compaction"`
}

type rawDirectoryInstallation struct {
	InstallationID       *string        `toml:"installation_id"`
	EnvironmentID        *string        `toml:"environment_id"`
	ProviderID           *string        `toml:"provider_id"`
	AdapterID            *string        `toml:"adapter_id"`
	ScanRootAuthorityIDs []string       `toml:"scan_root_authority_ids"`
	Enabled              *bool          `toml:"enabled"`
	Extensions           map[string]any `toml:"extensions"`
}

type rawDirectoryEnrichmentProfile struct {
	ProfileID      *string        `toml:"profile_id"`
	Enabled        *bool          `toml:"enabled"`
	MaxConcurrency *uint64        `toml:"max_concurrency"`
	MetadataPolicy *string        `toml:"metadata_policy"`
	Extensions     map[string]any `toml:"extensions"`
}

type rawDirectoryPeerDisclosure struct {
	HostID                  *string        `toml:"host_id"`
	EnvironmentObservations *string        `toml:"environment_observations"`
	NativeObservations      *string        `toml:"native_observations"`
	ManualMetadata          *string        `toml:"manual_metadata"`
	GeneratedMetadata       *string        `toml:"generated_metadata"`
	JobOperationStatus      *string        `toml:"job_operation_status"`
	Extensions              map[string]any `toml:"extensions"`
}

func Decode(document []byte, context DecodeContext) (LoadedConfiguration, error) {
	if _, err := scalar.ParsePlatform(context.RuntimePlatform.String()); err != nil {
		return LoadedConfiguration{}, configError("platform", ErrConfigValidation)
	}
	var envelope rawEnvelope
	if err := toml.NewDecoder(bytes.NewReader(document)).Decode(&envelope); err != nil {
		return LoadedConfiguration{}, configError("TOML", errors.Join(ErrConfigDecode, err))
	}
	if envelope.Schema != SchemaID {
		return LoadedConfiguration{}, configError("schema", ErrConfigValidation)
	}

	var value Configuration
	var err error
	switch envelope.SchemaVersion {
	case Version1:
		var raw rawV1
		if err = decodeStrict(document, &raw); err == nil {
			value, err = translateV1(raw, context)
		}
	case Version2:
		var raw rawV2
		if err = decodeStrict(document, &raw); err == nil {
			value, err = translateV2(raw, context)
		}
	case CurrentVersion:
		var raw rawV3
		if err = decodeStrict(document, &raw); err == nil {
			value, err = translateV3(raw, context)
		}
	default:
		return LoadedConfiguration{}, configError("schema_version", ErrUnsupportedConfigVersion)
	}
	if err != nil {
		return LoadedConfiguration{}, err
	}
	return LoadedConfiguration{SourceVersion: envelope.SchemaVersion, Value: value}, nil
}

func decodeStrict(document []byte, destination any) error {
	decoder := toml.NewDecoder(bytes.NewReader(document)).DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return configError("closed TOML shape", errors.Join(ErrConfigDecode, err))
	}
	var shape map[string]any
	if err := toml.Unmarshal(document, &shape); err != nil {
		return configError("closed TOML shape", errors.Join(ErrConfigDecode, err)) // config-refusal-subsumed: envelope TOML syntax parse
	}
	restorePresentEmptyMaps(reflect.ValueOf(destination), reflect.ValueOf(shape))
	return nil
}

// restorePresentEmptyMaps repairs a go-toml decoding ambiguity without
// weakening required-member validation. go-toml leaves a map field nil for a
// present but empty table such as [entry.extensions], which is otherwise
// indistinguishable from an absent required member in the typed value. The
// generic document shape retains table presence, so this schema-driven walk
// initializes only map members that were actually present in the source.
func restorePresentEmptyMaps(destination, source reflect.Value) {
	destination = indirectValue(destination)
	source = indirectValue(source)
	if !destination.IsValid() || !source.IsValid() {
		return
	}

	switch destination.Kind() {
	case reflect.Struct:
		if source.Kind() != reflect.Map || source.Type().Key().Kind() != reflect.String {
			return
		}
		for index := 0; index < destination.NumField(); index++ {
			fieldType := destination.Type().Field(index)
			name := strings.Split(fieldType.Tag.Get("toml"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			sourceField := source.MapIndex(reflect.ValueOf(name).Convert(source.Type().Key()))
			if !sourceField.IsValid() {
				continue
			}
			destinationField := destination.Field(index)
			unwrappedSource := indirectValue(sourceField)
			if destinationField.Kind() == reflect.Map {
				if destinationField.IsNil() && unwrappedSource.IsValid() && unwrappedSource.Kind() == reflect.Map {
					destinationField.Set(reflect.MakeMap(destinationField.Type()))
				}
				continue
			}
			restorePresentEmptyMaps(destinationField, sourceField)
		}
	case reflect.Slice:
		if source.Kind() != reflect.Slice {
			return
		}
		limit := destination.Len()
		if source.Len() < limit {
			limit = source.Len()
		}
		for index := 0; index < limit; index++ {
			restorePresentEmptyMaps(destination.Index(index), source.Index(index))
		}
	}
}

func indirectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

var configError = func(field string, err error) error {
	return &DocumentError{Clause: field, Err: err}
}
