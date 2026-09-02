package localstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestPathRegistryDerivesEveryResolverEntryFromPinnedCatalog(t *testing.T) {
	t.Parallel()

	if _, err := verifyPathRegistry(pathRegistryBytes); err != nil {
		t.Fatalf("verifyPathRegistry(reviewed bytes) error = %v", err)
	}
	semanticallyValidDrift := bytes.Replace(
		pathRegistryBytes,
		[]byte(`"environment": "AX_DATA_DIR"`),
		[]byte(`"environment": "AX_DATA_ROOT"`),
		1,
	)
	if bytes.Equal(semanticallyValidDrift, pathRegistryBytes) {
		t.Fatal("path-registry drift fixture did not change the reviewed bytes")
	}
	for _, test := range []struct {
		name      string
		candidate []byte
	}{
		{name: "absent read", candidate: nil},
		{name: "partial read", candidate: pathRegistryBytes[:len(pathRegistryBytes)/2]},
		{name: "byte-different whitespace", candidate: append(bytes.Clone(pathRegistryBytes), '\n')},
		{name: "semantically valid unreviewed row rename", candidate: semanticallyValidDrift},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := verifyPathRegistry(test.candidate); !errors.Is(err, ErrPathRegistry) {
				t.Fatalf("verifyPathRegistry() error = %v, want %v", err, ErrPathRegistry)
			}
		})
	}

	definitions, err := PathDefinitions()
	if err != nil {
		t.Fatalf("PathDefinitions() error = %v", err)
	}
	if len(definitions) != 5 {
		t.Fatalf("PathDefinitions() count = %d, want exact v1.0.0 registry count 5", len(definitions))
	}

	seenClasses := make(map[PathClass]struct{}, len(definitions))
	seenFlags := make(map[string]struct{}, len(definitions))
	seenEnvironment := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		if _, duplicate := seenClasses[definition.Class]; duplicate {
			t.Fatalf("duplicate path class %q", definition.Class)
		}
		if _, duplicate := seenFlags[definition.Flag]; duplicate {
			t.Fatalf("duplicate path flag %q", definition.Flag)
		}
		if _, duplicate := seenEnvironment[definition.Environment]; duplicate {
			t.Fatalf("duplicate path environment %q", definition.Environment)
		}
		seenClasses[definition.Class] = struct{}{}
		seenFlags[definition.Flag] = struct{}{}
		seenEnvironment[definition.Environment] = struct{}{}

		flagPath := "/flags/" + string(definition.Class)
		environmentPath := "/environment/" + string(definition.Class)
		resolved, resolveErr := ResolvePaths(ResolveRequest{
			Platform:    scalar.PlatformLinux,
			Flags:       map[string]string{definition.Flag: flagPath},
			Environment: linuxEnvironment(map[string]string{definition.Environment: environmentPath}),
			HomeDir:     "/home/fixture",
		})
		if resolveErr != nil {
			t.Fatalf("ResolvePaths(flag %s) error = %v", definition.Flag, resolveErr)
		}
		got, ok := resolved.Path(definition.Class)
		if !ok || got.Value.String() != flagPath || got.Source != PathSourceFlag {
			t.Fatalf("ResolvePaths(flag %s) = %+v, %t; want flag path", definition.Flag, got, ok)
		}

		resolved, resolveErr = ResolvePaths(ResolveRequest{
			Platform:    scalar.PlatformLinux,
			Environment: linuxEnvironment(map[string]string{definition.Environment: environmentPath}),
			HomeDir:     "/home/fixture",
		})
		if resolveErr != nil {
			t.Fatalf("ResolvePaths(environment %s) error = %v", definition.Environment, resolveErr)
		}
		got, ok = resolved.Path(definition.Class)
		if !ok || got.Value.String() != environmentPath || got.Source != PathSourceEnvironment {
			t.Fatalf("ResolvePaths(environment %s) = %+v, %t; want environment path", definition.Environment, got, ok)
		}
	}
}

func TestResolvePathsUsesExactPrecedenceDefaultsAndEmptyEnvironmentRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     ResolveRequest
		want        map[PathClass]string
		wantSources map[PathClass]PathSource
	}{
		{
			name: "macOS defaults with XDG config input",
			request: ResolveRequest{
				Platform: scalar.PlatformMacOS,
				Environment: map[string]string{
					"XDG_CONFIG_HOME": "/xdg/config",
				},
				HomeDir:      "/Users/fixture",
				TemporaryDir: "/private/var/folders/fixture/T",
			},
			want: map[PathClass]string{
				PathConfig:  "/xdg/config/ax/config.toml",
				PathData:    "/Users/fixture/Library/Application Support/ax",
				PathState:   "/Users/fixture/Library/Application Support/ax/state",
				PathCache:   "/Users/fixture/Library/Caches/ax",
				PathRuntime: "/private/var/folders/fixture/T/ax",
			},
		},
		{
			name: "Linux defaults",
			request: ResolveRequest{
				Platform: scalar.PlatformLinux,
				Environment: linuxEnvironment(map[string]string{
					"XDG_CONFIG_HOME": "/xdg/config",
					"XDG_DATA_HOME":   "/xdg/data",
					"XDG_STATE_HOME":  "/xdg/state",
					"XDG_CACHE_HOME":  "/xdg/cache",
				}),
				HomeDir: "/home/fixture",
			},
			want: map[PathClass]string{
				PathConfig:  "/xdg/config/ax/config.toml",
				PathData:    "/xdg/data/ax",
				PathState:   "/xdg/state/ax",
				PathCache:   "/xdg/cache/ax",
				PathRuntime: "/run/user/1000/ax",
			},
		},
		{
			name: "WSL2 fallback defaults and empty AX override",
			request: ResolveRequest{
				Platform: scalar.PlatformWSL2,
				Environment: linuxEnvironment(map[string]string{
					"AX_DATA_DIR": "",
				}),
				HomeDir: "/home/fixture",
			},
			want: map[PathClass]string{
				PathConfig:  "/home/fixture/.config/ax/config.toml",
				PathData:    "/home/fixture/.local/share/ax",
				PathState:   "/home/fixture/.local/state/ax",
				PathCache:   "/home/fixture/.cache/ax",
				PathRuntime: "/run/user/1000/ax",
			},
		},
		{
			name: "native Windows defaults",
			request: ResolveRequest{
				Platform: scalar.PlatformWindows,
				Environment: map[string]string{
					"APPDATA":      `C:\Users\fixture\AppData\Roaming`,
					"LOCALAPPDATA": `C:\Users\fixture\AppData\Local`,
				},
				HomeDir: `C:\Users\fixture`,
			},
			want: map[PathClass]string{
				PathConfig:  `C:\Users\fixture\AppData\Roaming\ax\config.toml`,
				PathData:    `C:\Users\fixture\AppData\Local\ax\data`,
				PathState:   `C:\Users\fixture\AppData\Local\ax\state`,
				PathCache:   `C:\Users\fixture\AppData\Local\ax\cache`,
				PathRuntime: `C:\Users\fixture\AppData\Local\ax\runtime`,
			},
		},
		{
			name: "flag over environment",
			request: ResolveRequest{
				Platform: scalar.PlatformLinux,
				Flags: map[string]string{
					"--data-dir": "/flag/data",
				},
				Environment: linuxEnvironment(map[string]string{
					"AX_DATA_DIR": "/environment/data",
				}),
				HomeDir: "/home/fixture",
			},
			want: map[PathClass]string{
				PathConfig:  "/home/fixture/.config/ax/config.toml",
				PathData:    "/flag/data",
				PathState:   "/home/fixture/.local/state/ax",
				PathCache:   "/home/fixture/.cache/ax",
				PathRuntime: "/run/user/1000/ax",
			},
			wantSources: map[PathClass]PathSource{PathData: PathSourceFlag},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			resolved, err := ResolvePaths(test.request)
			if err != nil {
				t.Fatalf("ResolvePaths() error = %v", err)
			}
			definitions, err := PathDefinitions()
			if err != nil {
				t.Fatalf("PathDefinitions() error = %v", err)
			}
			for _, definition := range definitions {
				got, ok := resolved.Path(definition.Class)
				if !ok {
					t.Fatalf("resolved path class %q is absent", definition.Class)
				}
				if got.Value.String() != test.want[definition.Class] {
					t.Errorf("path %s = %q, want %q", definition.Class, got.Value.String(), test.want[definition.Class])
				}
				wantSource := test.wantSources[definition.Class]
				if wantSource == "" {
					wantSource = PathSourceDefault
				}
				if got.Source != wantSource {
					t.Errorf("path %s source = %q, want %q", definition.Class, got.Source, wantSource)
				}
			}
		})
	}
}

func TestResolvePathsRefusesInvalidInputsAndIgnoresUnknownAXEnvironment(t *testing.T) {
	t.Parallel()

	base := ResolveRequest{
		Platform:    scalar.PlatformLinux,
		Environment: linuxEnvironment(map[string]string{"AX_UNKNOWN": "/must/not/be/read"}),
		HomeDir:     "/home/fixture",
	}
	resolved, err := ResolvePaths(base)
	if err != nil {
		t.Fatalf("ResolvePaths(unknown AX variable) error = %v", err)
	}
	for _, path := range resolved.Paths() {
		if path.Value.String() == "/must/not/be/read" {
			t.Fatalf("unknown AX variable selected for class %q", path.Class)
		}
	}

	tests := []struct {
		name    string
		mutate  func(*ResolveRequest)
		wantErr error
	}{
		{"unknown flag", func(request *ResolveRequest) { request.Flags = map[string]string{"--unknown": "/tmp/value"} }, ErrUnknownPathFlag},
		{"empty flag", func(request *ResolveRequest) { request.Flags = map[string]string{"--data-dir": ""} }, ErrInvalidPath},
		{"relative flag", func(request *ResolveRequest) { request.Flags = map[string]string{"--data-dir": "relative"} }, ErrInvalidPath},
		{"relative AX override", func(request *ResolveRequest) { request.Environment["AX_DATA_DIR"] = "relative" }, ErrInvalidPath},
		{"relative XDG input", func(request *ResolveRequest) { request.Environment["XDG_DATA_HOME"] = "relative" }, ErrInvalidPath},
		{"missing Linux runtime input", func(request *ResolveRequest) { delete(request.Environment, "XDG_RUNTIME_DIR") }, ErrPathDefaultUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := base
			request.Flags = nil
			request.Environment = cloneStrings(base.Environment)
			test.mutate(&request)
			_, resolveErr := ResolvePaths(request)
			if !errors.Is(resolveErr, test.wantErr) {
				t.Fatalf("ResolvePaths() error = %v, want %v", resolveErr, test.wantErr)
			}
		})
	}
}

func TestResolvePathsDoesNotReadUnavailableDefaultsBehindOverrides(t *testing.T) {
	t.Parallel()

	definitions, err := PathDefinitions()
	if err != nil {
		t.Fatal(err)
	}
	flags := make(map[string]string, len(definitions))
	for _, definition := range definitions {
		flags[definition.Flag] = "/explicit/" + string(definition.Class)
	}
	resolved, err := ResolvePaths(ResolveRequest{
		Platform:    scalar.PlatformLinux,
		Flags:       flags,
		Environment: map[string]string{},
	})
	if err != nil {
		t.Fatalf("ResolvePaths(all flags without home/XDG defaults) error = %v", err)
	}
	for _, definition := range definitions {
		got, ok := resolved.Path(definition.Class)
		if !ok || got.Source != PathSourceFlag || got.Value.String() != flags[definition.Flag] {
			t.Errorf("resolved %s = %+v, %t; want explicit flag", definition.Class, got, ok)
		}
	}

	resolved, err = ResolvePaths(ResolveRequest{
		Platform: scalar.PlatformLinux,
		Flags: map[string]string{
			"--config":    "/explicit/config.toml",
			"--data-dir":  "/explicit/data",
			"--state-dir": "/explicit/state",
			"--cache-dir": "/explicit/cache",
		},
		Environment: map[string]string{"AX_RUNTIME_DIR": "/explicit/runtime"},
	})
	if err != nil {
		t.Fatalf("ResolvePaths(AX runtime without XDG runtime) error = %v", err)
	}
	got, _ := resolved.Path(PathRuntime)
	if got.Source != PathSourceEnvironment || got.Value.String() != "/explicit/runtime" {
		t.Fatalf("runtime = %+v, want AX_RUNTIME_DIR", got)
	}
}

func TestResolvePathsRefusesUnknownPlatformEvenWhenDefaultsAreOverridden(t *testing.T) {
	t.Parallel()

	definitions, err := PathDefinitions()
	if err != nil {
		t.Fatal(err)
	}
	flags := make(map[string]string, len(definitions))
	for _, definition := range definitions {
		flags[definition.Flag] = "/explicit/" + string(definition.Class)
	}
	_, err = ResolvePaths(ResolveRequest{
		Platform: scalar.Platform("unknown-platform"),
		Flags:    flags,
	})
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("ResolvePaths(unknown platform with all flags) error = %v, want %v", err, ErrInvalidPath)
	}
}

func TestResolvePathsRefusesUnavailableWindowsDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		flags       map[string]string
		environment map[string]string
	}{
		{
			name:        "APPDATA absent",
			environment: map[string]string{"LOCALAPPDATA": `C:\Users\fixture\AppData\Local`},
		},
		{
			name:        "LOCALAPPDATA absent behind explicit config",
			flags:       map[string]string{"--config": `C:\explicit\config.toml`},
			environment: map[string]string{"APPDATA": `C:\Users\fixture\AppData\Roaming`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ResolvePaths(ResolveRequest{
				Platform:    scalar.PlatformWindows,
				Flags:       test.flags,
				Environment: test.environment,
			})
			if !errors.Is(err, ErrPathDefaultUnavailable) {
				t.Fatalf("ResolvePaths() error = %v, want %v", err, ErrPathDefaultUnavailable)
			}
		})
	}
}

func TestResolvePathsMatchesWindowsEnvironmentKeysCaseInsensitively(t *testing.T) {
	t.Parallel()

	resolved, err := ResolvePaths(ResolveRequest{
		Platform: scalar.PlatformWindows,
		Environment: map[string]string{
			"appdata":      `C:\Users\fixture\AppData\Roaming`,
			"LocalAppData": `C:\Users\fixture\AppData\Local`,
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths(mixed-case Windows environment) error = %v", err)
	}
	config, ok := resolved.Path(PathConfig)
	if !ok || config.Value.String() != `C:\Users\fixture\AppData\Roaming\ax\config.toml` {
		t.Fatalf("Windows config path = %+v, %t", config, ok)
	}
	runtimePath, ok := resolved.Path(PathRuntime)
	if !ok || runtimePath.Value.String() != `C:\Users\fixture\AppData\Local\ax\runtime` {
		t.Fatalf("Windows runtime path = %+v, %t", runtimePath, ok)
	}
}

func TestInitializeLayoutCreatesOnlyOwnerAccessibleNativeRoots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission assertions")
	}

	home := t.TempDir()
	resolved, err := ResolvePaths(ResolveRequest{
		Platform: nativeTestPlatform(),
		Environment: map[string]string{
			"XDG_CONFIG_HOME": filepath.Join(home, "config"),
			"XDG_DATA_HOME":   filepath.Join(home, "data"),
			"XDG_STATE_HOME":  filepath.Join(home, "state"),
			"XDG_CACHE_HOME":  filepath.Join(home, "cache"),
			"XDG_RUNTIME_DIR": filepath.Join(home, "runtime"),
		},
		HomeDir:      home,
		TemporaryDir: filepath.Join(home, "temporary"),
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	if err := InitializeLayout(resolved); err != nil {
		t.Fatalf("InitializeLayout() error = %v", err)
	}

	for _, resolvedPath := range resolved.Paths() {
		path := resolvedPath.Value.String()
		if resolvedPath.Class == PathConfig {
			path = filepath.Dir(path)
		}
		info, statErr := os.Lstat(path)
		if statErr != nil {
			t.Fatalf("lstat(%s) error = %v", resolvedPath.Class, statErr)
		}
		if !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Errorf("%s mode = %s, want owner-only directory 0700", resolvedPath.Class, info.Mode())
		}
	}
}

func TestInitializeLayoutRefusesUnsafeExistingOwnedRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission assertions")
	}

	for _, mode := range []os.FileMode{0o755, 0o750, 0o500} {
		mode := mode
		t.Run(mode.String(), func(t *testing.T) {
			home := t.TempDir()
			data := filepath.Join(home, "unsafe-data")
			if err := os.Mkdir(data, 0o700); err != nil {
				t.Fatalf("prepare unsafe data root: %v", err)
			}
			if err := os.Chmod(data, mode); err != nil {
				t.Fatalf("set unsafe data-root fixture mode: %v", err)
			}
			resolved, err := ResolvePaths(ResolveRequest{
				Platform: nativeTestPlatform(),
				Flags: map[string]string{
					"--config":      filepath.Join(home, "config", "config.toml"),
					"--data-dir":    data,
					"--state-dir":   filepath.Join(home, "state"),
					"--cache-dir":   filepath.Join(home, "cache"),
					"--runtime-dir": filepath.Join(home, "runtime"),
				},
				HomeDir:      home,
				TemporaryDir: filepath.Join(home, "temporary"),
			})
			if err != nil {
				t.Fatalf("ResolvePaths() error = %v", err)
			}
			if err := os.Mkdir(filepath.Join(home, "config"), 0o700); err != nil {
				t.Fatalf("prepare config parent: %v", err)
			}
			if err := InitializeLayout(resolved); !errors.Is(err, ErrUnsafeOwnership) {
				t.Fatalf("InitializeLayout(mode %04o) error = %v, want %v", mode, err, ErrUnsafeOwnership)
			}
		})
	}
}

func TestInitializeLayoutRefusesOwnerModeRegularFileAtDirectoryRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission assertions")
	}

	home := t.TempDir()
	configRoot := filepath.Join(home, "config")
	if err := os.Mkdir(configRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(home, "data")
	if err := os.WriteFile(data, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(data, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolvePaths(ResolveRequest{
		Platform: nativeTestPlatform(),
		Flags: map[string]string{
			"--config":      filepath.Join(configRoot, "config.toml"),
			"--data-dir":    data,
			"--state-dir":   filepath.Join(home, "state"),
			"--cache-dir":   filepath.Join(home, "cache"),
			"--runtime-dir": filepath.Join(home, "runtime"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := InitializeLayout(resolved); !errors.Is(err, ErrUnsafeOwnership) {
		t.Fatalf("InitializeLayout(owner-mode file root) error = %v, want %v", err, ErrUnsafeOwnership)
	}
	info, statErr := os.Lstat(data)
	if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o700 {
		t.Fatalf("directory-root fixture changed: info/error = %v/%v", info, statErr)
	}
}

func linuxEnvironment(extra map[string]string) map[string]string {
	values := map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}
	for key, value := range extra {
		values[key] = value
	}
	return values
}

func cloneStrings(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func nativeTestPlatform() scalar.Platform {
	switch runtime.GOOS {
	case "darwin":
		return scalar.PlatformMacOS
	case "linux":
		return scalar.PlatformLinux
	default:
		return scalar.PlatformWindows
	}
}

func TestPathDefinitionsReturnIsolatedCopies(t *testing.T) {
	t.Parallel()

	first, err := PathDefinitions()
	if err != nil {
		t.Fatalf("PathDefinitions() error = %v", err)
	}
	second, err := PathDefinitions()
	if err != nil {
		t.Fatalf("PathDefinitions() second error = %v", err)
	}
	if len(first) == 0 || !slices.Equal(first, second) {
		t.Fatalf("PathDefinitions() copies differ: first=%v second=%v", first, second)
	}
	first[0].Flag = "mutated"
	third, err := PathDefinitions()
	if err != nil {
		t.Fatalf("PathDefinitions() third error = %v", err)
	}
	if third[0].Flag == "mutated" {
		t.Fatal("PathDefinitions() returned caller-mutable registry storage")
	}
}

func TestDecodePathRegistryRefusesNarrowedMalformedAndDuplicateCatalogs(t *testing.T) {
	t.Parallel()

	if _, err := decodePathRegistry([]byte("{")); !errors.Is(err, ErrPathRegistry) {
		t.Fatalf("decodePathRegistry(malformed) error = %v, want %v", err, ErrPathRegistry)
	}
	if _, err := decodePathRegistry(append(append([]byte(nil), pathRegistryBytes...), []byte("{}")...)); !errors.Is(err, ErrPathRegistry) {
		t.Fatalf("decodePathRegistry(trailing) error = %v, want %v", err, ErrPathRegistry)
	}

	tests := []struct {
		name   string
		mutate func(*pathRegistry)
	}{
		{"source identity", func(registry *pathRegistry) { registry.Source.Commit = "0000000000000000000000000000000000000000" }},
		{"narrowed row count", func(registry *pathRegistry) { registry.Paths = registry.Paths[:4] }},
		{"unknown class", func(registry *pathRegistry) { registry.Paths[0].Class = "unknown" }},
		{"wrong value kind", func(registry *pathRegistry) { registry.Paths[0].ValueKind = PathValueDirectory }},
		{"malformed flag", func(registry *pathRegistry) { registry.Paths[0].Flag = "config" }},
		{"malformed environment", func(registry *pathRegistry) { registry.Paths[0].Environment = "CONFIG" }},
		{"duplicate class", func(registry *pathRegistry) { registry.Paths[2].Class = registry.Paths[1].Class }},
		{"duplicate flag", func(registry *pathRegistry) { registry.Paths[1].Flag = registry.Paths[0].Flag }},
		{"duplicate environment", func(registry *pathRegistry) { registry.Paths[1].Environment = registry.Paths[0].Environment }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var registry pathRegistry
			if err := json.Unmarshal(pathRegistryBytes, &registry); err != nil {
				t.Fatal(err)
			}
			test.mutate(&registry)
			candidate, err := json.Marshal(registry)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodePathRegistry(candidate); !errors.Is(err, ErrPathRegistry) {
				t.Fatalf("decodePathRegistry() error = %v, want %v", err, ErrPathRegistry)
			}
		})
	}
}

func TestInitializeLayoutValidatesExplicitConfigurationPathKindsAndModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission assertions")
	}

	tests := []struct {
		name    string
		prepare func(*testing.T, string)
		wantErr error
	}{
		{"absent file with existing private parent", func(*testing.T, string) {}, nil},
		{"existing private regular file", func(t *testing.T, name string) {
			if err := os.WriteFile(name, []byte("config"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(name, 0o600); err != nil {
				t.Fatal(err)
			}
		}, nil},
		{"existing directory", func(t *testing.T, name string) {
			if err := os.Mkdir(name, 0o700); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafeOwnership},
		{"existing owner-mode directory", func(t *testing.T, name string) {
			if err := os.Mkdir(name, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(name, 0o600); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafeOwnership},
		{"existing broad regular file", func(t *testing.T, name string) {
			if err := os.WriteFile(name, []byte("config"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(name, 0o644); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafeOwnership},
		{"existing group-readable regular file", func(t *testing.T, name string) {
			if err := os.WriteFile(name, []byte("config"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(name, 0o640); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafeOwnership},
		{"existing owner-read-only regular file", func(t *testing.T, name string) {
			if err := os.WriteFile(name, []byte("config"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(name, 0o400); err != nil {
				t.Fatal(err)
			}
		}, ErrUnsafeOwnership},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			configRoot := filepath.Join(home, "config")
			if err := os.Mkdir(configRoot, 0o700); err != nil {
				t.Fatal(err)
			}
			configFile := filepath.Join(configRoot, "config.toml")
			test.prepare(t, configFile)
			resolved, err := ResolvePaths(ResolveRequest{
				Platform: nativeTestPlatform(),
				Flags: map[string]string{
					"--config": configFile, "--data-dir": filepath.Join(home, "data"),
					"--state-dir": filepath.Join(home, "state"), "--cache-dir": filepath.Join(home, "cache"),
					"--runtime-dir": filepath.Join(home, "runtime"),
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if resolved.Platform() != nativeTestPlatform() {
				t.Fatalf("ResolvedPaths.Platform() = %s", resolved.Platform())
			}
			err = InitializeLayout(resolved)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("InitializeLayout() error = %v, want %v", err, test.wantErr)
			}
		})
	}

	home := t.TempDir()
	resolved, err := ResolvePaths(ResolveRequest{
		Platform: nativeTestPlatform(),
		Flags: map[string]string{
			"--config": filepath.Join(home, "missing", "config.toml"), "--data-dir": filepath.Join(home, "data"),
			"--state-dir": filepath.Join(home, "state"), "--cache-dir": filepath.Join(home, "cache"),
			"--runtime-dir": filepath.Join(home, "runtime"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := InitializeLayout(resolved); err == nil {
		t.Fatal("InitializeLayout(explicit config with absent parent) error = nil, want refusal")
	}
	if err := InitializeLayout(ResolvedPaths{}); !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("InitializeLayout(zero paths) error = %v, want %v", err, ErrUnsupportedPlatform)
	}
	if err := InitializeLayout(ResolvedPaths{platform: nativeTestPlatform(), paths: map[PathClass]ResolvedPath{}}); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("InitializeLayout(missing catalog path) error = %v, want %v", err, ErrInvalidPath)
	}
}

func TestOwnerDirectoryHelpersRefuseTraversalAndMissingSyncTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix owner-directory implementation")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ensureOwnerChildTree(root, ".."); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("ensureOwnerChildTree(parent traversal) error = %v, want %v", err, ErrInvalidPath)
	}
	if err := syncDirectory(filepath.Join(root, "missing")); err == nil {
		t.Fatal("syncDirectory(missing) error = nil, want refusal")
	}
}
