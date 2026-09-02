package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

type closedMapMemberCase struct {
	configure func(*Configuration) BackendSettingsValidator
	omit      func(*rawV3)
	clause    string
}

func closedMapMemberCases() map[string]closedMapMemberCase {
	return map[string]closedMapMemberCase{
		"directory_installations[].extensions": {
			configure: func(configuration *Configuration) BackendSettingsValidator {
				configuration.DirectoryInstallations[0].Extensions = map[string]any{}
				return nil
			},
			omit:   func(raw *rawV3) { raw.DirectoryInstallations[0].Extensions = nil },
			clause: "directory_installations[0] required member",
		},
		"directory_enrichment_profiles[].extensions": {
			configure: func(configuration *Configuration) BackendSettingsValidator {
				configuration.DirectoryEnrichmentProfiles = []DirectoryEnrichmentProfile{{
					ProfileID: testDigest, Enabled: true, MaxConcurrency: 1,
					MetadataPolicy: "local_only", Extensions: map[string]any{},
				}}
				return nil
			},
			omit:   func(raw *rawV3) { raw.DirectoryEnrichmentProfiles[0].Extensions = nil },
			clause: "directory_enrichment_profiles[0] required member",
		},
		"directory_peer_disclosure[].extensions": {
			configure: func(configuration *Configuration) BackendSettingsValidator {
				configuration.DirectoryPeerDisclosure = []DirectoryPeerDisclosure{{
					HostID: testPeerID, EnvironmentObservations: "local_only", NativeObservations: "local_only",
					ManualMetadata: "local_only", GeneratedMetadata: "local_only", JobOperationStatus: "local_only",
					Extensions: map[string]any{},
				}}
				return nil
			},
			omit:   func(raw *rawV3) { raw.DirectoryPeerDisclosure[0].Extensions = nil },
			clause: "directory_peer_disclosure[0] required member",
		},
		"terminal.backend_config[].settings": {
			configure: func(configuration *Configuration) BackendSettingsValidator {
				configuration.Terminal.BackendConfig = []BackendConfig{{
					BackendID: "ax.tmux", ConfigVersion: "1.0.0", Settings: map[string]any{},
				}}
				return acceptBackendSettings{}
			},
			omit:   func(raw *rawV3) { raw.Terminal.BackendConfig[0].Settings = nil },
			clause: "terminal.backend_config[0] required member",
		},
	}
}

func TestEveryClosedMapMemberRoundTripsAnExplicitEmptyObjectAtProductionEntry(t *testing.T) {
	t.Parallel()

	cases := closedMapMemberCases()

	derived := closedMapMemberPaths(reflect.TypeFor[rawV3]())
	requireClosedMapMemberCaseCompleteness(t, cases, derived)

	for _, path := range derived {
		path := path
		t.Run(path, func(t *testing.T) {
			configuration := validCurrentConfiguration()
			settings := cases[path].configure(&configuration)
			context := DecodeContext{RuntimePlatform: scalar.PlatformMacOS, BackendSettings: settings}
			encoded, err := EncodeCurrent(configuration, context)
			if err != nil {
				t.Fatalf("EncodeCurrent(%s empty object) error = %v", path, err)
			}
			snapshot, err := loadConfigDocument(encoded, scalar.PlatformMacOS, settings)
			if err != nil {
				t.Fatalf("Load(EncodeCurrent(%s empty object)) error = %v\n%s", path, err, encoded)
			}
			loaded, ok := snapshot.Configuration()
			if !ok || loaded.SourceVersion != CurrentVersion {
				t.Fatalf("Load(EncodeCurrent(%s)).Configuration() = %#v, present=%v", path, loaded, ok)
			}
		})
	}
}

func TestLoadRefusesEveryAbsentRequiredClosedMapMemberAtProductionEntry(t *testing.T) {
	t.Parallel()

	cases := closedMapMemberCases()
	derived := closedMapMemberPaths(reflect.TypeFor[rawV3]())
	requireClosedMapMemberCaseCompleteness(t, cases, derived)

	for _, path := range derived {
		path := path
		t.Run(path, func(t *testing.T) {
			configuration := validCurrentConfiguration()
			testCase := cases[path]
			settings := testCase.configure(&configuration)
			raw := currentWire(configuration)

			complete, err := toml.Marshal(raw)
			if err != nil {
				t.Fatalf("marshal complete %s fixture: %v", path, err)
			}
			if _, err := loadConfigDocument(complete, scalar.PlatformMacOS, settings); err != nil {
				t.Fatalf("complete %s fixture is not valid: %v", path, err)
			}

			testCase.omit(&raw)
			omitted, err := toml.Marshal(raw)
			if err != nil {
				t.Fatalf("marshal %s omission fixture: %v", path, err)
			}
			_, err = loadConfigDocument(omitted, scalar.PlatformMacOS, settings)
			requireConfigClause(t, err, testCase.clause)
		})
	}
}

func TestLoadTreatsInlineAndMultilineEmptyClosedMapsAsPresent(t *testing.T) {
	t.Parallel()

	prefix := string(minimalValidConfigVersion(scalar.PlatformMacOS, Version2)) + `
[[directory_installations]]
installation_id = "` + testDigest + `"
environment_id = "local"
provider_id = "codex"
adapter_id = "codex-local"
scan_root_authority_ids = ["` + testDigest + `"]
enabled = true
`
	for name, suffix := range map[string]string{
		"inline object":         "extensions = {}\n",
		"multiline empty table": "[directory_installations.extensions]\n",
	} {
		name, suffix := name, suffix
		t.Run(name, func(t *testing.T) {
			if _, err := loadConfigDocument([]byte(prefix+suffix), scalar.PlatformMacOS, nil); err != nil {
				t.Fatalf("Load(%s empty object) error = %v", name, err)
			}
		})
	}
}

func TestEveryPinnedReaderHasPositiveNativeWindowsAndWSL2Lanes(t *testing.T) {
	t.Parallel()

	for _, version := range pinnedConfigurationVersions(t) {
		version := version
		for _, platformCase := range []struct {
			platform scalar.Platform
			path     string
			backend  string
		}{
			{platform: scalar.PlatformWindows, path: `D:\Developer\ReluxWorks`, backend: "ax.conpty"},
			{platform: scalar.PlatformWSL2, path: "/srv/relux", backend: "ax.tmux"},
		} {
			platformCase := platformCase
			t.Run(version+"/"+platformCase.platform.String(), func(t *testing.T) {
				document := append(minimalValidConfigVersion(platformCase.platform, version), []byte(fmt.Sprintf(`
[[workspace_roots]]
logical_root = "relux"
path = %q
`, platformCase.path))...)
				snapshot, err := loadConfigDocument(document, platformCase.platform, nil)
				if err != nil {
					t.Fatalf("Load(%s %s native path) error = %v", version, platformCase.platform, err)
				}
				loaded, ok := snapshot.Configuration()
				if !ok || loaded.Value.Terminal.BackendID != platformCase.backend {
					t.Fatalf("Load(%s %s) backend = %q, present=%v, want %q", version, platformCase.platform, loaded.Value.Terminal.BackendID, ok, platformCase.backend)
				}
			})
		}
	}
}

func TestLegacyConPTYTranslationIsPinnedForEveryLegacyReader(t *testing.T) {
	t.Parallel()

	for _, version := range pinnedConfigurationVersions(t) {
		if version == CurrentVersion {
			continue
		}
		document := append(minimalValidConfigVersion(scalar.PlatformWindows, version), []byte("\n[terminal]\nbackend = \"conpty\"\n")...)
		snapshot, err := loadConfigDocument(document, scalar.PlatformWindows, nil)
		if err != nil {
			t.Fatalf("Load(%s explicit conpty) error = %v", version, err)
		}
		loaded, _ := snapshot.Configuration()
		if loaded.Value.Terminal.BackendID != "ax.conpty" {
			t.Fatalf("Load(%s explicit conpty) backend = %q", version, loaded.Value.Terminal.BackendID)
		}
	}
}

func TestPeerWorkspaceRootValidationUsesPeerPlatformAndPinsEveryNestedClause(t *testing.T) {
	t.Parallel()

	validWindowsPeer := func() Configuration {
		configuration := validCurrentConfiguration()
		configuration.Mesh.Peers[0].Platform = scalar.PlatformWindows
		configuration.Mesh.Peers[0].WorkspaceRoots = []WorkspaceRoot{{LogicalRoot: "relux", Path: `D:\Developer\ReluxWorks`}}
		return configuration
	}
	if _, err := EncodeCurrent(validWindowsPeer(), DecodeContext{RuntimePlatform: scalar.PlatformMacOS}); err != nil {
		t.Fatalf("EncodeCurrent(native Windows peer root) error = %v", err)
	}

	tests := []struct {
		name, clause string
		mutate       func(*Configuration)
	}{
		{name: "peer path grammar", clause: "mesh.peers[0].workspace_roots[0].path", mutate: func(value *Configuration) {
			value.Mesh.Peers[0].WorkspaceRoots[0].Path = "/srv/relux"
		}},
		{name: "peer logical-root grammar", clause: "mesh.peers[0].workspace_roots[0].logical_root", mutate: func(value *Configuration) {
			value.Mesh.Peers[0].WorkspaceRoots[0].LogicalRoot = "Relux"
		}},
		{name: "peer duplicate logical root", clause: "mesh.peers[0].workspace_roots duplicate logical_root", mutate: func(value *Configuration) {
			value.Mesh.Peers[0].WorkspaceRoots = append(value.Mesh.Peers[0].WorkspaceRoots, value.Mesh.Peers[0].WorkspaceRoots[0])
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			configuration := validWindowsPeer()
			test.mutate(&configuration)
			_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
			requireConfigClause(t, err, test.clause)
		})
	}
}

func TestPreviouslyUnpinnedConfigurationBoundsAtProductionEntry(t *testing.T) {
	t.Parallel()

	t.Run("scan authority minimum", func(t *testing.T) {
		configuration := validCurrentConfiguration()
		configuration.DirectoryInstallations[0].ScanRootAuthorityIDs = []string{}
		_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
		requireConfigClause(t, err, "directory_installations[0].scan_root_authority_ids")
	})

	t.Run("logical-root grammar independent of length", func(t *testing.T) {
		for _, logicalRoot := range []string{"Relux", "1relux"} {
			configuration := validCurrentConfiguration()
			configuration.WorkspaceRoots[0].LogicalRoot = logicalRoot
			_, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
			requireConfigClause(t, err, "workspace_roots[0].logical_root")
		}
	})

	t.Run("extension namespace grammar subsumes the three-byte minimum", func(t *testing.T) {
		at := validCurrentConfiguration()
		at.DirectoryInstallations[0].Extensions = map[string]any{"a.b": true}
		if _, err := EncodeCurrent(at, DecodeContext{RuntimePlatform: scalar.PlatformMacOS}); err != nil {
			t.Fatalf("EncodeCurrent(shortest reverse-DNS extension key) error = %v", err)
		}
		past := validCurrentConfiguration()
		past.DirectoryInstallations[0].Extensions = map[string]any{"ab": true}
		_, err := EncodeCurrent(past, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
		requireConfigClause(t, err, "directory_installations[0].extensions")
	})

	t.Run("extension int64 safe-integer bound", func(t *testing.T) {
		at := validCurrentConfiguration()
		at.DirectoryInstallations[0].Extensions = map[string]any{"works.relux.number": int64(scalar.MaxSafeInteger)}
		if _, err := EncodeCurrent(at, DecodeContext{RuntimePlatform: scalar.PlatformMacOS}); err != nil {
			t.Fatalf("EncodeCurrent(extension int64 at safe limit) error = %v", err)
		}
		past := validCurrentConfiguration()
		past.DirectoryInstallations[0].Extensions = map[string]any{"works.relux.number": int64(scalar.MaxSafeInteger + 1)}
		_, err := EncodeCurrent(past, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
		requireConfigClause(t, err, "directory_installations[0].extensions")
	})
}

func closedMapMemberPaths(root reflect.Type) []string {
	closedMap := reflect.TypeFor[map[string]any]()
	var paths []string
	var visit func(reflect.Type, string)
	visit = func(value reflect.Type, prefix string) {
		if value.Kind() == reflect.Pointer {
			value = value.Elem()
		}
		if value.Kind() != reflect.Struct {
			return
		}
		for index := 0; index < value.NumField(); index++ {
			field := value.Field(index)
			name := strings.Split(field.Tag.Get("toml"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			path := name
			if prefix != "" {
				path = prefix + "." + name
			}
			fieldType := field.Type
			if fieldType == closedMap {
				paths = append(paths, path)
				continue
			}
			if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Struct {
				visit(fieldType.Elem(), path+"[]")
				continue
			}
			visit(fieldType, path)
		}
	}
	visit(root, "")
	sort.Strings(paths)
	return paths
}

func requireClosedMapMemberCaseCompleteness(t *testing.T, cases map[string]closedMapMemberCase, derived []string) {
	t.Helper()
	declared := make([]string, 0, len(cases))
	for path := range cases {
		declared = append(declared, path)
	}
	sort.Strings(declared)
	if !reflect.DeepEqual(declared, derived) {
		t.Fatalf("closed-map cases = %v, rawV3 closed map members = %v", declared, derived)
	}
}

func pinnedConfigurationVersions(t *testing.T) []string {
	t.Helper()
	for _, contract := range catalog.Current().Contracts {
		if contract.ID == catalog.ContractID(SchemaID) {
			return append([]string(nil), contract.Versions...)
		}
	}
	t.Fatal("pinned catalog has no Configuration contract")
	return nil
}
