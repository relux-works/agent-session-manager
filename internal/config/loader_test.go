package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestLoadAppliesPathPrecedenceAtProductionEntry(t *testing.T) {
	t.Parallel()

	environment := map[string]string{
		"AX_CONFIG":         "/env/config.toml",
		"AX_DATA_DIR":       "/env/data",
		"AX_STATE_DIR":      "/env/state",
		"AX_CACHE_DIR":      "",
		"AX_RUNTIME_DIR":    "/env/runtime",
		"AX_OPENAI_API_KEY": "must-not-be-interpreted",
		"OPENAI_API_KEY":    "must-not-be-read",
		"XDG_CONFIG_HOME":   "/xdg/config",
	}
	files := newFakeFileSystem()
	wantDocument := minimalValidConfig(scalar.PlatformMacOS)
	files.regular["/flag/config.toml"] = wantDocument
	files.regular["/env/config.toml"] = []byte("wrong = true\n")

	inputs := fixtureInputs(scalar.PlatformMacOS, environment, files)
	snapshot, err := Load(inputs, Overrides{
		ConfigFile: "/flag/config.toml",
		DataRoot:   "/flag/data",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	assertResolvedPath(t, snapshot.Paths(), ConfigFile, "/flag/config.toml", SourceFlag)
	assertResolvedPath(t, snapshot.Paths(), DataRoot, "/flag/data", SourceFlag)
	assertResolvedPath(t, snapshot.Paths(), StateRoot, "/env/state", SourceEnvironment)
	assertResolvedPath(t, snapshot.Paths(), CacheRoot, "/Users/test/Library/Caches/ax", SourcePlatformDefault)
	assertResolvedPath(t, snapshot.Paths(), RuntimeRoot, "/env/runtime", SourceEnvironment)

	if got, want := string(snapshot.Document()), string(wantDocument); got != want {
		t.Fatalf("Document() = %q, want %q", got, want)
	}
	if !snapshot.ConfigPresent() {
		t.Fatal("ConfigPresent() = false, want true for existing selected file")
	}

	environment["AX_STATE_DIR"] = "/changed/state"
	document := snapshot.Document()
	document[0] = 'X'
	assertResolvedPath(t, snapshot.Paths(), StateRoot, "/env/state", SourceEnvironment)
	if strings.HasPrefix(string(snapshot.Document()), "X") {
		t.Fatal("Document() exposed mutable loader state")
	}
}

func TestResolvePathsUsesNormativePlatformDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		platform    scalar.Platform
		environment map[string]string
		home        string
		temp        string
		working     string
		want        map[PathClass]string
	}{
		{
			name:     "macOS fallback",
			platform: scalar.PlatformMacOS,
			home:     "/Users/test",
			temp:     "/private/var/folders/test/T",
			working:  "/work",
			want: map[PathClass]string{
				ConfigFile:  "/Users/test/.config/ax/config.toml",
				DataRoot:    "/Users/test/Library/Application Support/ax",
				StateRoot:   "/Users/test/Library/Application Support/ax/state",
				CacheRoot:   "/Users/test/Library/Caches/ax",
				RuntimeRoot: "/private/var/folders/test/T/ax",
			},
		},
		{
			name:     "macOS XDG config input",
			platform: scalar.PlatformMacOS,
			environment: map[string]string{
				"XDG_CONFIG_HOME": "/xdg/config",
			},
			home:    "/Users/test",
			temp:    "/private/tmp/user",
			working: "/work",
			want: map[PathClass]string{
				ConfigFile:  "/xdg/config/ax/config.toml",
				DataRoot:    "/Users/test/Library/Application Support/ax",
				StateRoot:   "/Users/test/Library/Application Support/ax/state",
				CacheRoot:   "/Users/test/Library/Caches/ax",
				RuntimeRoot: "/private/tmp/user/ax",
			},
		},
		{
			name:     "Linux XDG inputs and fallbacks",
			platform: scalar.PlatformLinux,
			environment: map[string]string{
				"XDG_CONFIG_HOME": "/xdg/config",
				"XDG_STATE_HOME":  "/xdg/state",
				"XDG_RUNTIME_DIR": "/run/user/1000",
			},
			home:    "/home/test",
			temp:    "/tmp",
			working: "/work",
			want: map[PathClass]string{
				ConfigFile:  "/xdg/config/ax/config.toml",
				DataRoot:    "/home/test/.local/share/ax",
				StateRoot:   "/xdg/state/ax",
				CacheRoot:   "/home/test/.cache/ax",
				RuntimeRoot: "/run/user/1000/ax",
			},
		},
		{
			name:     "WSL2 uses Linux layout",
			platform: scalar.PlatformWSL2,
			environment: map[string]string{
				"XDG_RUNTIME_DIR": "/run/user/1000",
			},
			home:    "/home/test",
			temp:    "/tmp",
			working: "/work",
			want: map[PathClass]string{
				ConfigFile:  "/home/test/.config/ax/config.toml",
				DataRoot:    "/home/test/.local/share/ax",
				StateRoot:   "/home/test/.local/state/ax",
				CacheRoot:   "/home/test/.cache/ax",
				RuntimeRoot: "/run/user/1000/ax",
			},
		},
		{
			name:     "native Windows",
			platform: scalar.PlatformWindows,
			environment: map[string]string{
				"APPDATA":      "C:\\Users\\test\\AppData\\Roaming",
				"LOCALAPPDATA": "C:\\Users\\test\\AppData\\Local",
			},
			home:    "C:\\Users\\test",
			temp:    "C:\\Users\\test\\AppData\\Local\\Temp",
			working: "C:\\work",
			want: map[PathClass]string{
				ConfigFile:  "C:\\Users\\test\\AppData\\Roaming\\ax\\config.toml",
				DataRoot:    "C:\\Users\\test\\AppData\\Local\\ax\\data",
				StateRoot:   "C:\\Users\\test\\AppData\\Local\\ax\\state",
				CacheRoot:   "C:\\Users\\test\\AppData\\Local\\ax\\cache",
				RuntimeRoot: "C:\\Users\\test\\AppData\\Local\\Temp\\ax",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			inputs := Inputs{
				Platform:   test.platform,
				HomeDir:    test.home,
				TempDir:    test.temp,
				WorkingDir: test.working,
				LookupEnv:  mapLookup(test.environment),
			}
			paths, err := ResolvePaths(inputs, nil)
			if err != nil {
				t.Fatalf("ResolvePaths() error = %v", err)
			}
			if got, want := len(paths.All()), len(OverrideRegistry()); got != want {
				t.Fatalf("resolved path count = %d, want registry count %d", got, want)
			}
			for class, want := range test.want {
				assertResolvedPath(t, paths, class, want, SourcePlatformDefault)
			}
		})
	}
}

func TestOverrideRegistryIsExactAndDrivesEveryEnvironmentResolution(t *testing.T) {
	t.Parallel()

	wantRegistry := []OverrideSpec{
		{Class: ConfigFile, Flag: "--config", Environment: "AX_CONFIG"},
		{Class: DataRoot, Flag: "--data-dir", Environment: "AX_DATA_DIR"},
		{Class: StateRoot, Flag: "--state-dir", Environment: "AX_STATE_DIR"},
		{Class: CacheRoot, Flag: "--cache-dir", Environment: "AX_CACHE_DIR"},
		{Class: RuntimeRoot, Flag: "--runtime-dir", Environment: "AX_RUNTIME_DIR"},
	}
	if got := OverrideRegistry(); !reflect.DeepEqual(got, wantRegistry) {
		t.Fatalf("OverrideRegistry() = %#v, want %#v", got, wantRegistry)
	}

	for _, specification := range OverrideRegistry() {
		specification := specification
		t.Run(string(specification.Class), func(t *testing.T) {
			t.Parallel()
			value := "/override/" + string(specification.Class)
			environment := map[string]string{
				specification.Environment: value,
				"AX_UNKNOWN":              "/must/not/be/used",
				"AX_PROVIDER_TOKEN":       "secret",
				"XDG_RUNTIME_DIR":         "/run/user/1000",
			}
			paths, err := ResolvePaths(fixtureInputs(scalar.PlatformLinux, environment, nil), nil)
			if err != nil {
				t.Fatalf("ResolvePaths() error = %v", err)
			}
			assertResolvedPath(t, paths, specification.Class, value, SourceEnvironment)
			for _, other := range OverrideRegistry() {
				path, _ := paths.Path(other.Class)
				if path.Value.String() == "/must/not/be/used" || path.Value.String() == "secret" {
					t.Fatalf("unknown/secret environment leaked into %s", other.Class)
				}
			}
		})
	}
}

func TestResolvePathsTreatsEmptyEnvironmentOverridesAsUnset(t *testing.T) {
	t.Parallel()

	emptyFlags := make(Overrides, len(OverrideRegistry()))
	for _, specification := range OverrideRegistry() {
		emptyFlags[specification.Class] = ""
	}
	paths, err := ResolvePaths(fixtureInputs(scalar.PlatformMacOS, map[string]string{
		"AX_CONFIG":      "",
		"AX_DATA_DIR":    "",
		"AX_STATE_DIR":   "",
		"AX_CACHE_DIR":   "",
		"AX_RUNTIME_DIR": "",
	}, nil), emptyFlags)
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	for _, specification := range OverrideRegistry() {
		path, ok := paths.Path(specification.Class)
		if !ok {
			t.Fatalf("Path(%s) is missing", specification.Class)
		}
		if path.Source != SourcePlatformDefault {
			t.Errorf("Path(%s).Source = %q, want %q", specification.Class, path.Source, SourcePlatformDefault)
		}
	}
}

func TestLoadOSCapturesExplicitEnvironmentAtProductionEntry(t *testing.T) {
	platform := hostPlatform(t)
	root := t.TempDir()
	configDirectory := filepath.Join(root, "config")
	if err := os.Mkdir(configDirectory, 0o700); err != nil {
		t.Fatalf("Mkdir(config directory): %v", err)
	}
	configFile := filepath.Join(configDirectory, "config.toml")
	wantDocument := minimalValidConfig(platform)
	if err := os.WriteFile(configFile, wantDocument, 0o600); err != nil {
		t.Fatalf("WriteFile(config): %v", err)
	}

	paths := map[PathClass]string{ConfigFile: configFile}
	for _, specification := range OverrideRegistry() {
		if specification.Class != ConfigFile {
			paths[specification.Class] = filepath.Join(root, string(specification.Class))
			if err := os.Mkdir(paths[specification.Class], 0o700); err != nil {
				t.Fatalf("Mkdir(%s): %v", specification.Class, err)
			}
		}
		t.Setenv(specification.Environment, paths[specification.Class])
	}

	snapshot, err := LoadOS(platform, nil)
	if err != nil {
		t.Fatalf("LoadOS() error = %v", err)
	}
	if !snapshot.ConfigPresent() {
		t.Fatal("LoadOS().ConfigPresent() = false, want true")
	}
	if !reflect.DeepEqual(snapshot.Document(), wantDocument) {
		t.Fatalf("LoadOS().Document() = %q, want %q", snapshot.Document(), wantDocument)
	}
	for _, specification := range OverrideRegistry() {
		assertResolvedPath(t, snapshot.Paths(), specification.Class, filepath.Clean(paths[specification.Class]), SourceEnvironment)
	}
}

// TestLoadOSResolvesSymlinksAndStillEnforcesKindsAtProductionEntry pins the
// declared read-side symlink stance at the real LoadOS entry, in both
// directions and one clause per case. Section 3.2 gives the configuration file
// the value kind "Existing regular file, or a not-yet-created regular-file path
// whose parent exists" and every other class the value kind "Directory";
// neither Section 3.2 nor Section 6.1 states a no-follow requirement for these
// five classes, so the resolved target decides the kind. Each negative case
// violates exactly one kind clause with every other input valid.
func TestLoadOSResolvesSymlinksAndStillEnforcesKindsAtProductionEntry(t *testing.T) {
	platform := hostPlatform(t)

	tests := []struct {
		name string
		// setup returns the ConfigFile override and any non-ConfigFile
		// overrides it wants to replace in the default all-real-directories set.
		setup   func(t *testing.T, root string, overrides Overrides) string
		want    error
		present bool
	}{
		{
			name: "configuration file symlinked onto a regular file loads",
			setup: func(t *testing.T, root string, _ Overrides) string {
				target := filepath.Join(root, "target.toml")
				writeFixtureFile(t, target, minimalValidConfig(platform))
				return symlinkFixture(t, target, filepath.Join(root, "config.toml"))
			},
			present: true,
		},
		{
			name: "configuration file symlinked onto a directory is refused",
			setup: func(t *testing.T, root string, _ Overrides) string {
				target := filepath.Join(root, "target-directory")
				if err := os.Mkdir(target, 0o700); err != nil {
					t.Fatal(err)
				}
				return symlinkFixture(t, target, filepath.Join(root, "config.toml"))
			},
			want: ErrConfigNotRegular,
		},
		{
			name: "durable data root symlinked onto a directory loads",
			setup: func(t *testing.T, root string, overrides Overrides) string {
				target := filepath.Join(root, "target-data")
				if err := os.Mkdir(target, 0o700); err != nil {
					t.Fatal(err)
				}
				overrides[DataRoot] = symlinkFixture(t, target, filepath.Join(root, "data-link"))
				return writeFixtureFile(t, filepath.Join(root, "config.toml"), minimalValidConfig(platform))
			},
			present: true,
		},
		{
			name: "durable data root symlinked onto a regular file is refused",
			setup: func(t *testing.T, root string, overrides Overrides) string {
				target := writeFixtureFile(t, filepath.Join(root, "target-file"), []byte("not a directory\n"))
				overrides[DataRoot] = symlinkFixture(t, target, filepath.Join(root, "data-link"))
				return writeFixtureFile(t, filepath.Join(root, "config.toml"), minimalValidConfig(platform))
			},
			want: ErrRootNotDirectory,
		},
		{
			name: "not-yet-created configuration file under a symlinked parent is admitted",
			setup: func(t *testing.T, root string, _ Overrides) string {
				target := filepath.Join(root, "target-config-directory")
				if err := os.Mkdir(target, 0o700); err != nil {
					t.Fatal(err)
				}
				link := symlinkFixture(t, target, filepath.Join(root, "config-directory"))
				return filepath.Join(link, "config.toml")
			},
			present: false,
		},
		{
			// Declared consequence of resolving before the kind check: a
			// configuration symlink whose target does not exist resolves to an
			// absent regular-file path whose parent directory exists, which is
			// exactly the second Section 3.2 value kind. Load never writes, so
			// this admission creates nothing.
			name: "dangling configuration symlink is admitted as not-yet-created",
			setup: func(t *testing.T, root string, _ Overrides) string {
				return symlinkFixture(t, filepath.Join(root, "never-created.toml"), filepath.Join(root, "config.toml"))
			},
			present: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			overrides := make(Overrides, len(OverrideRegistry()))
			for _, specification := range OverrideRegistry() {
				if specification.Class == ConfigFile {
					continue
				}
				directory := filepath.Join(root, string(specification.Class))
				if err := os.Mkdir(directory, 0o700); err != nil {
					t.Fatal(err)
				}
				overrides[specification.Class] = directory
			}
			overrides[ConfigFile] = test.setup(t, root, overrides)

			snapshot, err := LoadOS(platform, overrides)
			if test.want != nil {
				if !errors.Is(err, test.want) {
					t.Fatalf("LoadOS() error = %v, want %v", err, test.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadOS() error = %v, want a loaded snapshot", err)
			}
			if got := snapshot.ConfigPresent(); got != test.present {
				t.Fatalf("LoadOS().ConfigPresent() = %v, want %v", got, test.present)
			}
			if !test.present {
				return
			}
			if !bytes.Equal(snapshot.Document(), minimalValidConfig(platform)) {
				t.Fatalf("LoadOS().Document() = %q, want the resolved target bytes", snapshot.Document())
			}
			if _, ok := snapshot.Configuration(); !ok {
				t.Fatal("LoadOS() decoded no configuration from the resolved target")
			}
		})
	}
}

func symlinkFixture(t *testing.T, target, link string) string {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("host cannot create test symlink: %v", err)
	}
	return link
}

func writeFixtureFile(t *testing.T, name string, document []byte) string {
	t.Helper()
	if err := os.WriteFile(name, document, 0o600); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestLoadOSAppliesExplicitOverridesAtProductionEntry(t *testing.T) {
	platform := hostPlatform(t)
	root := t.TempDir()
	overrides := make(Overrides, len(OverrideRegistry()))

	for _, specification := range OverrideRegistry() {
		environmentValue := filepath.Join(root, "environment", string(specification.Class))
		overrideValue := filepath.Join(root, "flag", string(specification.Class))
		if specification.Class == ConfigFile {
			environmentValue += ".toml"
			overrideValue += ".toml"
			if err := os.MkdirAll(filepath.Dir(overrideValue), 0o700); err != nil {
				t.Fatalf("MkdirAll(flag config parent): %v", err)
			}
			if err := os.WriteFile(overrideValue, minimalValidConfig(platform), 0o600); err != nil {
				t.Fatalf("WriteFile(flag config): %v", err)
			}
		} else if err := os.MkdirAll(overrideValue, 0o700); err != nil {
			t.Fatalf("MkdirAll(%s): %v", specification.Class, err)
		}
		t.Setenv(specification.Environment, environmentValue)
		overrides[specification.Class] = overrideValue
	}

	snapshot, err := LoadOS(platform, overrides)
	if err != nil {
		t.Fatalf("LoadOS(explicit overrides) error = %v", err)
	}
	for _, specification := range OverrideRegistry() {
		assertResolvedPath(t, snapshot.Paths(), specification.Class, overrides[specification.Class], SourceFlag)
	}
}

func TestLoadOSRefusesInvalidRuntimePlatformBeforeReadingOS(t *testing.T) {
	inputs, err := OSInputs(scalar.Platform("solaris"))
	if !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("OSInputs() error = %v, want ErrInvalidContext", err)
	}
	if !reflect.DeepEqual(inputs, Inputs{}) {
		t.Fatalf("OSInputs() = %#v, want zero inputs on invalid platform", inputs)
	}

	_, err = LoadOS(scalar.Platform("solaris"), nil)
	if !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("LoadOS() error = %v, want ErrInvalidContext", err)
	}
}

// TestResolvePathsDefersButPreservesHomeLookupFailure pins only the injected
// half: an arbitrary captured cause reaches the caller through the
// platform-default refusal. It hand-sets the unexported field, so it cannot
// witness whether production ever reaches that state. The real capture is
// pinned at the production entry by
// TestLoadOSCarriesTheRealUserHomeFailureAtEveryHomeDerivedClass and
// TestLoadOSDerivesPlatformDefaultsFromTheRealCapturedUserHome.
func TestResolvePathsDefersButPreservesHomeLookupFailure(t *testing.T) {
	homeFailure := errors.New("synthetic user-home lookup failure")
	inputs := fixtureInputs(scalar.PlatformMacOS, map[string]string{}, nil)
	inputs.HomeDir = ""
	inputs.homeDirError = homeFailure
	_, err := ResolvePaths(inputs, nil)
	if !errors.Is(err, ErrPlatformDefaultUnavailable) || !errors.Is(err, homeFailure) {
		t.Fatalf("ResolvePaths(deferred home failure) error = %v", err)
	}
}

func TestResolvePathsNeverEnumeratesOrPassesThroughAmbientSecrets(t *testing.T) {
	t.Parallel()

	allowed := map[string]string{
		"AX_CONFIG":      "/config/config.toml",
		"AX_DATA_DIR":    "/data",
		"AX_STATE_DIR":   "/state",
		"AX_CACHE_DIR":   "/cache",
		"AX_RUNTIME_DIR": "/runtime",
	}
	lookedUp := make(map[string]int)
	inputs := fixtureInputs(scalar.PlatformMacOS, nil, nil)
	inputs.LookupEnv = func(name string) (string, bool) {
		lookedUp[name]++
		value, ok := allowed[name]
		return value, ok
	}

	paths, err := ResolvePaths(inputs, nil)
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	if got, want := len(lookedUp), len(OverrideRegistry()); got != want {
		t.Fatalf("environment lookup count = %d, want exact override registry count %d: %#v", got, want, lookedUp)
	}
	for _, specification := range OverrideRegistry() {
		if lookedUp[specification.Environment] != 1 {
			t.Errorf("lookup count for %s = %d, want 1", specification.Environment, lookedUp[specification.Environment])
		}
		assertResolvedPath(t, paths, specification.Class, allowed[specification.Environment], SourceEnvironment)
	}
	for _, forbidden := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "AX_PROVIDER_TOKEN", "HOME"} {
		if lookedUp[forbidden] != 0 {
			t.Errorf("secret/ambient variable %s was read", forbidden)
		}
	}
}

func TestResolvePathsLimitsEnvironmentLookupsAtEveryPrecedenceLayer(t *testing.T) {
	t.Parallel()

	registry := OverrideRegistry()
	flagOverrides := make(Overrides, len(registry))
	environmentOverrides := make(map[string]string, len(registry))
	registryEnvironment := make(map[string]struct{}, len(registry))
	for _, specification := range registry {
		flagOverrides[specification.Class] = "/flag/" + string(specification.Class)
		environmentOverrides[specification.Environment] = "/environment/" + string(specification.Class)
		registryEnvironment[specification.Environment] = struct{}{}
	}

	linuxDefaultEnvironment := map[PathClass]string{
		ConfigFile:  "XDG_CONFIG_HOME",
		DataRoot:    "XDG_DATA_HOME",
		StateRoot:   "XDG_STATE_HOME",
		CacheRoot:   "XDG_CACHE_HOME",
		RuntimeRoot: "XDG_RUNTIME_DIR",
	}
	defaultValues := map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}
	defaultEnvironment := make(map[string]struct{}, len(registryEnvironment)+len(linuxDefaultEnvironment))
	for name := range registryEnvironment {
		defaultEnvironment[name] = struct{}{}
	}
	for _, name := range linuxDefaultEnvironment {
		defaultEnvironment[name] = struct{}{}
	}

	tests := []struct {
		name        string
		values      map[string]string
		overrides   Overrides
		wantLookups map[string]struct{}
	}{
		{name: "flag", overrides: flagOverrides, wantLookups: map[string]struct{}{}},
		{name: "documented environment", values: environmentOverrides, wantLookups: registryEnvironment},
		{name: "platform default", values: defaultValues, wantLookups: defaultEnvironment},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			lookedUp := make(map[string]int)
			inputs := fixtureInputs(scalar.PlatformLinux, nil, nil)
			inputs.LookupEnv = func(name string) (string, bool) {
				lookedUp[name]++
				value, ok := test.values[name]
				return value, ok
			}
			if _, err := ResolvePaths(inputs, test.overrides); err != nil {
				t.Fatalf("ResolvePaths(%s) error = %v", test.name, err)
			}
			if got, want := len(lookedUp), len(test.wantLookups); got != want {
				t.Fatalf("ResolvePaths(%s) looked up %d names, want %d: %#v", test.name, got, want, lookedUp)
			}
			for name := range test.wantLookups {
				if lookedUp[name] != 1 {
					t.Errorf("ResolvePaths(%s) lookup count for %s = %d, want 1", test.name, name, lookedUp[name])
				}
			}
			for name := range lookedUp {
				if _, ok := test.wantLookups[name]; !ok {
					t.Errorf("ResolvePaths(%s) interpreted undocumented environment variable %s", test.name, name)
				}
			}
		})
	}
}

func TestResolvePathsConvertsRelativeFlagsAgainstCapturedWorkingDirectory(t *testing.T) {
	t.Parallel()

	paths, err := ResolvePaths(fixtureInputs(scalar.PlatformLinux, map[string]string{
		"XDG_RUNTIME_DIR": "/run/user/1000",
	}, nil), Overrides{
		ConfigFile:  "config/ax.toml",
		DataRoot:    "data",
		StateRoot:   "state",
		CacheRoot:   "cache",
		RuntimeRoot: "runtime",
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	for class, suffix := range map[PathClass]string{
		ConfigFile:  "config/ax.toml",
		DataRoot:    "data",
		StateRoot:   "state",
		CacheRoot:   "cache",
		RuntimeRoot: "runtime",
	} {
		assertResolvedPath(t, paths, class, "/work/"+suffix, SourceFlag)
	}
}

func TestResolvePathsRefusesUnknownClassesAndInvalidContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		inputs Inputs
		flags  Overrides
		want   error
	}{
		{
			name:   "unknown override class",
			inputs: fixtureInputs(scalar.PlatformMacOS, nil, nil),
			flags:  Overrides{PathClass("credential"): "/secret"},
			want:   ErrUnknownPathClass,
		},
		{
			name: "unknown platform",
			inputs: Inputs{
				Platform:   scalar.Platform("solaris"),
				HomeDir:    "/home/test",
				TempDir:    "/tmp",
				WorkingDir: "/work",
				LookupEnv:  mapLookup(nil),
			},
			want: ErrInvalidContext,
		},
		{
			name: "Linux runtime default unavailable",
			inputs: Inputs{
				Platform:   scalar.PlatformLinux,
				HomeDir:    "/home/test",
				TempDir:    "/tmp",
				WorkingDir: "/work",
				LookupEnv:  mapLookup(nil),
			},
			want: ErrPlatformDefaultUnavailable,
		},
		{
			name: "Windows defaults unavailable",
			inputs: Inputs{
				Platform:   scalar.PlatformWindows,
				HomeDir:    "C:\\Users\\test",
				TempDir:    "C:\\Temp",
				WorkingDir: "C:\\work",
				LookupEnv:  mapLookup(nil),
			},
			flags: Overrides{
				DataRoot:    "C:\\data",
				StateRoot:   "C:\\state",
				CacheRoot:   "C:\\cache",
				RuntimeRoot: "C:\\runtime",
			},
			want: ErrPlatformDefaultUnavailable,
		},
		{
			name: "Windows local app data unavailable",
			inputs: Inputs{
				Platform:   scalar.PlatformWindows,
				HomeDir:    "C:\\Users\\test",
				TempDir:    "C:\\Temp",
				WorkingDir: "C:\\work",
				LookupEnv:  mapLookup(nil),
			},
			flags: Overrides{
				ConfigFile:  "C:\\config.toml",
				RuntimeRoot: "C:\\runtime",
			},
			want: ErrPlatformDefaultUnavailable,
		},
		{
			name: "macOS home default unavailable",
			inputs: Inputs{
				Platform:   scalar.PlatformMacOS,
				TempDir:    "/tmp",
				WorkingDir: "/work",
				LookupEnv:  mapLookup(nil),
			},
			want: ErrPlatformDefaultUnavailable,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ResolvePaths(test.inputs, test.flags)
			if !errors.Is(err, test.want) {
				t.Fatalf("ResolvePaths() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestResolvePathsRejectsInvalidPlatformBeforePathResolution(t *testing.T) {
	t.Parallel()

	_, err := ResolvePaths(Inputs{
		Platform:   scalar.Platform("solaris"),
		WorkingDir: "/work",
		LookupEnv:  mapLookup(nil),
	}, Overrides{
		ConfigFile:  "/config.toml",
		DataRoot:    "/data",
		StateRoot:   "/state",
		CacheRoot:   "/cache",
		RuntimeRoot: "/runtime",
	})
	if !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("ResolvePaths() error = %v, want ErrInvalidContext", err)
	}
}

func TestLoadErrorsDoNotEchoSelectedPathValues(t *testing.T) {
	t.Parallel()

	const privateRoot = "/machine/private"
	tests := []struct {
		name  string
		setup func(*fakeFileSystem, map[string]string)
		want  error
	}{
		{
			name: "config stat failure",
			setup: func(files *fakeFileSystem, _ map[string]string) {
				files.statErrors[privateRoot+"/config.toml"] = &fs.PathError{Op: "stat", Path: privateRoot + "/config.toml", Err: fs.ErrPermission}
			},
			want: ErrConfigRead,
		},
		{
			name: "config read failure",
			setup: func(files *fakeFileSystem, _ map[string]string) {
				files.regular[privateRoot+"/config.toml"] = []byte("schema = \"urn:ax:schema:config\"\n")
				files.readErrors[privateRoot+"/config.toml"] = &fs.PathError{Op: "read", Path: privateRoot + "/config.toml", Err: fs.ErrPermission}
			},
			want: ErrConfigRead,
		},
		{
			name: "config parent inspection failure",
			setup: func(files *fakeFileSystem, _ map[string]string) {
				files.statErrors[privateRoot] = &fs.PathError{Op: "stat", Path: privateRoot, Err: fs.ErrPermission}
			},
			want: ErrConfigRead,
		},
		{
			name: "root inspection failure",
			setup: func(files *fakeFileSystem, environment map[string]string) {
				files.regular[privateRoot+"/config.toml"] = []byte("schema = \"urn:ax:schema:config\"\n")
				environment["AX_DATA_DIR"] = privateRoot + "/data"
				files.statErrors[privateRoot+"/data"] = &fs.PathError{Op: "stat", Path: privateRoot + "/data", Err: fs.ErrPermission}
			},
			want: ErrRootInspect,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := newFakeFileSystem()
			environment := map[string]string{"AX_CONFIG": privateRoot + "/config.toml"}
			test.setup(files, environment)
			_, err := Load(fixtureInputs(scalar.PlatformMacOS, environment, files), nil)
			if !errors.Is(err, test.want) {
				t.Fatalf("Load() error = %v, want %v", err, test.want)
			}
			if strings.Contains(err.Error(), privateRoot) {
				t.Fatalf("Load() error echoed selected path: %v", err)
			}
		})
	}
}

func TestErrorFormattingNeverEchoesWrappedDetails(t *testing.T) {
	t.Parallel()

	const secret = "/machine/private/credential-path"
	wrapped := &fs.PathError{Op: "open", Path: secret, Err: fs.ErrPermission}
	err := &Error{
		Operation: "read selected file",
		Class:     ConfigFile,
		Source:    SourceEnvironment,
		Err:       errors.Join(ErrConfigRead, wrapped),
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), wrapped.Error()) {
		t.Fatalf("Error.Error() exposed wrapped details: %v", err)
	}
	if !errors.Is(err, ErrConfigRead) || !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Error.Unwrap() lost error identity: %v", err)
	}

	// *MigrationError carries the same claim as *Error and the durable path
	// joins raw OS errors that quote absolute paths, so the rendered migration
	// message is held to the same contract: neither a machine-local path nor
	// document content may survive formatting, while errors.Is identity must.
	const document = "host_name = \"private-workstation\""
	migrationWrapped := &fs.PathError{Op: "rename", Path: secret, Err: fs.ErrPermission}
	migration := &MigrationError{
		Operation: "replace source",
		Err:       errors.Join(ErrMigrationReplace, migrationWrapped, errors.New(document)),
	}
	rendered := migration.Error()
	if strings.Contains(rendered, secret) || strings.Contains(rendered, migrationWrapped.Error()) || strings.Contains(rendered, document) {
		t.Fatalf("MigrationError.Error() exposed wrapped details: %v", migration)
	}
	if rendered != "configuration migration replace source failed" {
		t.Fatalf("MigrationError.Error() = %q", rendered)
	}
	if !errors.Is(migration, ErrMigrationReplace) || !errors.Is(migration, fs.ErrPermission) {
		t.Fatalf("MigrationError.Unwrap() lost error identity: %v", migration)
	}
}

func TestLoadAcceptsAbsentConfigOnlyWhenParentDirectoryExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		platform    scalar.Platform
		config      string
		parent      string
		environment map[string]string
	}{
		{
			name:     "Unix absolute path",
			platform: scalar.PlatformMacOS,
			config:   "/config/config.toml",
			parent:   "/config",
			environment: map[string]string{
				"AX_CONFIG": "/config/config.toml",
			},
		},
		{
			name:     "Windows drive path",
			platform: scalar.PlatformWindows,
			config:   "C:\\config\\config.toml",
			parent:   "C:\\config",
			environment: map[string]string{
				"AX_CONFIG":      "C:\\config\\config.toml",
				"AX_DATA_DIR":    "C:\\data",
				"AX_STATE_DIR":   "C:\\state",
				"AX_CACHE_DIR":   "C:\\cache",
				"AX_RUNTIME_DIR": "C:\\runtime",
			},
		},
		{
			name:     "Windows UNC path",
			platform: scalar.PlatformWindows,
			config:   "\\\\server\\share\\config.toml",
			parent:   "\\\\server\\share",
			environment: map[string]string{
				"AX_CONFIG":      "\\\\server\\share\\config.toml",
				"AX_DATA_DIR":    "C:\\data",
				"AX_STATE_DIR":   "C:\\state",
				"AX_CACHE_DIR":   "C:\\cache",
				"AX_RUNTIME_DIR": "C:\\runtime",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := newFakeFileSystem()
			files.directories[test.parent] = true
			snapshot, err := Load(fixtureInputs(test.platform, test.environment, files), nil)
			if err != nil {
				t.Fatalf("Load(absent config with existing parent) error = %v", err)
			}
			if snapshot.ConfigPresent() {
				t.Fatal("ConfigPresent() = true, want false for not-yet-created config")
			}
			if len(snapshot.Document()) != 0 {
				t.Fatalf("Document() = %q, want empty absent document", snapshot.Document())
			}
			assertResolvedPath(t, snapshot.Paths(), ConfigFile, test.config, SourceEnvironment)
		})
	}
}

func TestLoadDistinguishesAbsenceFromReadFailureAndWrongKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(*fakeFileSystem)
		want  error
	}{
		{name: "absent selected file parent", want: ErrConfigParentNotFound},
		{
			name: "selected file parent is not a directory",
			setup: func(files *fakeFileSystem) {
				files.regular["/config"] = []byte("not a directory")
			},
			want: ErrConfigParentNotDirectory,
		},
		{
			name: "selected file parent inspection fails",
			setup: func(files *fakeFileSystem) {
				files.statErrors["/config"] = fs.ErrPermission
			},
			want: ErrConfigRead,
		},
		{
			name: "stat failure is not absence",
			setup: func(files *fakeFileSystem) {
				files.statErrors["/config/config.toml"] = fs.ErrPermission
			},
			want: ErrConfigRead,
		},
		{
			name: "read races after successful stat",
			setup: func(files *fakeFileSystem) {
				files.regular["/config/config.toml"] = []byte("schema = \"urn:ax:schema:config\"\n")
				files.readErrors["/config/config.toml"] = fs.ErrNotExist
			},
			want: ErrConfigRead,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			files := newFakeFileSystem()
			if test.setup != nil {
				test.setup(files)
			}
			inputs := fixtureInputs(scalar.PlatformMacOS, map[string]string{
				"AX_CONFIG": "/config/config.toml",
			}, files)
			_, err := Load(inputs, nil)
			if !errors.Is(err, test.want) {
				t.Fatalf("Load() error = %v, want %v", err, test.want)
			}
			if test.want != ErrConfigParentNotFound && errors.Is(err, ErrConfigParentNotFound) {
				t.Fatalf("Load() collapsed %v into missing parent", test.want)
			}
		})
	}
}

func TestLoadRefusesEveryNonRegularConfigKind(t *testing.T) {
	t.Parallel()

	kinds := map[string]fs.FileMode{
		"directory":  fs.ModeDir | 0o700,
		"named-pipe": fs.ModeNamedPipe | 0o600,
		"socket":     fs.ModeSocket | 0o600,
		"device":     fs.ModeDevice | 0o600,
		"symlink":    fs.ModeSymlink | 0o600,
	}
	for name, mode := range kinds {
		name, mode := name, mode
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			files := newFakeFileSystem()
			files.modes["/config/config.toml"] = mode
			inputs := fixtureInputs(scalar.PlatformMacOS, map[string]string{
				"AX_CONFIG": "/config/config.toml",
			}, files)
			_, err := Load(inputs, nil)
			if !errors.Is(err, ErrConfigNotRegular) {
				t.Fatalf("Load(%s) error = %v, want ErrConfigNotRegular", name, err)
			}
		})
	}
}

func TestLoadRefusesWrongOrUnreadableRootKinds(t *testing.T) {
	t.Parallel()

	environment := map[string]string{}
	for _, specification := range OverrideRegistry() {
		environment[specification.Environment] = "/" + string(specification.Class)
	}

	for _, specification := range OverrideRegistry() {
		if specification.Class == ConfigFile {
			continue
		}
		specification := specification
		for _, failure := range []struct {
			name  string
			want  error
			setup func(*fakeFileSystem, string)
		}{
			{
				name: "wrong-kind",
				want: ErrRootNotDirectory,
				setup: func(files *fakeFileSystem, selected string) {
					files.regular[selected] = []byte("not a directory")
				},
			},
			{
				name: "inspection-failure",
				want: ErrRootInspect,
				setup: func(files *fakeFileSystem, selected string) {
					files.statErrors[selected] = fs.ErrPermission
				},
			},
		} {
			failure := failure
			t.Run(string(specification.Class)+"/"+failure.name, func(t *testing.T) {
				t.Parallel()
				files := newFakeFileSystem()
				files.regular["/config-file"] = []byte("schema = \"urn:ax:schema:config\"\n")
				failure.setup(files, environment[specification.Environment])
				_, err := Load(fixtureInputs(scalar.PlatformMacOS, environment, files), nil)
				if !errors.Is(err, failure.want) {
					t.Fatalf("Load(%s) error = %v, want %v", specification.Class, err, failure.want)
				}
			})
		}
	}
}

func TestResolvePathsNormalizesWindowsRelativeAndUNCRoots(t *testing.T) {
	t.Parallel()

	inputs := Inputs{
		Platform:   scalar.PlatformWindows,
		HomeDir:    "C:\\Users\\test",
		TempDir:    "C:\\Temp",
		WorkingDir: "C:\\work\\project",
		LookupEnv:  mapLookup(nil),
	}
	paths, err := ResolvePaths(inputs, Overrides{
		ConfigFile:  "config\\ax.toml",
		DataRoot:    "\\\\server\\share",
		StateRoot:   "state\\.\\current",
		CacheRoot:   "cache\\old\\..\\current",
		RuntimeRoot: "runtime",
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}
	assertResolvedPath(t, paths, ConfigFile, "C:\\work\\project\\config\\ax.toml", SourceFlag)
	assertResolvedPath(t, paths, DataRoot, "\\\\server\\share", SourceFlag)
	assertResolvedPath(t, paths, StateRoot, "C:\\work\\project\\state\\current", SourceFlag)
	assertResolvedPath(t, paths, CacheRoot, "C:\\work\\project\\cache\\current", SourceFlag)
	assertResolvedPath(t, paths, RuntimeRoot, "C:\\work\\project\\runtime", SourceFlag)
}

func TestResolvePathsRefusesWindowsTraversalAboveRoot(t *testing.T) {
	t.Parallel()

	_, err := ResolvePaths(Inputs{
		Platform:   scalar.PlatformWindows,
		WorkingDir: "C:\\work",
		LookupEnv:  mapLookup(nil),
	}, Overrides{
		ConfigFile:  "C:\\config.toml",
		DataRoot:    "..\\..\\escape",
		StateRoot:   "C:\\state",
		CacheRoot:   "C:\\cache",
		RuntimeRoot: "C:\\runtime",
	})
	if !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("ResolvePaths() error = %v, want ErrInvalidContext", err)
	}
}

func TestLoadRefusesMissingFilesystemFunctions(t *testing.T) {
	t.Parallel()

	files := newFakeFileSystem()
	files.regular["/config/config.toml"] = minimalValidConfig(scalar.PlatformMacOS)
	inputs := fixtureInputs(scalar.PlatformMacOS, map[string]string{
		"AX_CONFIG": "/config/config.toml",
	}, files)
	inputType := reflect.TypeOf(inputs)
	foundFunction := false
	for fieldIndex := 0; fieldIndex < inputType.NumField(); fieldIndex++ {
		field := inputType.Field(fieldIndex)
		if field.Type.Kind() != reflect.Func {
			continue
		}
		foundFunction = true
		functionFieldIndex := fieldIndex
		functionField := field
		t.Run(functionField.Name, func(t *testing.T) {
			t.Parallel()
			missing := inputs
			reflect.ValueOf(&missing).Elem().Field(functionFieldIndex).Set(reflect.Zero(functionField.Type))
			_, err := Load(missing, nil)
			if !errors.Is(err, ErrInvalidContext) {
				t.Fatalf("Load(missing %s) error = %v, want ErrInvalidContext", functionField.Name, err)
			}
		})
	}
	if !foundFunction {
		t.Fatal("Inputs exposes no injected function fields to validate")
	}
}

func TestLoadIsIdempotentReadOnlyAndReturnsIsolatedSnapshots(t *testing.T) {
	t.Parallel()

	files := newFakeFileSystem()
	files.regular["/config/config.toml"] = minimalValidConfig(scalar.PlatformMacOS)
	inputs := fixtureInputs(scalar.PlatformMacOS, map[string]string{
		"AX_CONFIG": "/config/config.toml",
	}, files)

	first, err := Load(inputs, nil)
	if err != nil {
		t.Fatalf("first Load() error = %v", err)
	}
	second, err := Load(inputs, nil)
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if !reflect.DeepEqual(first.Paths().All(), second.Paths().All()) || !reflect.DeepEqual(first.Document(), second.Document()) {
		t.Fatal("identical Load() calls returned different snapshots")
	}
	if got, want := files.readCount["/config/config.toml"], 2; got != want {
		t.Fatalf("config read count = %d, want %d", got, want)
	}
	if files.writeCount != 0 {
		t.Fatalf("Load() performed %d writes", files.writeCount)
	}

	firstDocument := first.Document()
	firstDocument[0] = 'X'
	firstPaths := first.Paths().All()
	delete(firstPaths, ConfigFile)
	if reflect.DeepEqual(first.Document(), firstDocument) {
		t.Fatal("snapshot document aliases caller mutation")
	}
	if _, ok := first.Paths().Path(ConfigFile); !ok {
		t.Fatal("snapshot paths alias caller mutation")
	}
}

func assertResolvedPath(t *testing.T, paths ResolvedPaths, class PathClass, want string, source Source) {
	t.Helper()
	resolved, ok := paths.Path(class)
	if !ok {
		t.Fatalf("Path(%s) is missing", class)
	}
	if got := resolved.Value.String(); got != want {
		t.Errorf("Path(%s) = %q, want %q", class, got, want)
	}
	if resolved.Source != source {
		t.Errorf("Path(%s).Source = %q, want %q", class, resolved.Source, source)
	}
}

func fixtureInputs(platform scalar.Platform, environment map[string]string, files *fakeFileSystem) Inputs {
	inputs := Inputs{
		Platform:   platform,
		HomeDir:    "/Users/test",
		TempDir:    "/private/tmp/test",
		WorkingDir: "/work",
		LookupEnv:  mapLookup(environment),
	}
	if platform == scalar.PlatformLinux || platform == scalar.PlatformWSL2 {
		inputs.HomeDir = "/home/test"
		inputs.TempDir = "/tmp"
	}
	if platform == scalar.PlatformWindows {
		inputs.HomeDir = "C:\\Users\\test"
		inputs.TempDir = "C:\\Temp"
		inputs.WorkingDir = "C:\\work"
	}
	if files != nil {
		inputs.Stat = files.Stat
		inputs.ReadFile = files.ReadFile
	}
	return inputs
}

func hostPlatform(t *testing.T) scalar.Platform {
	t.Helper()
	switch runtime.GOOS {
	case "darwin":
		return scalar.PlatformMacOS
	case "linux":
		return scalar.PlatformLinux
	case "windows":
		return scalar.PlatformWindows
	default:
		t.Skipf("LoadOS fixture does not support GOOS=%s", runtime.GOOS)
		return ""
	}
}

func minimalValidConfig(platform scalar.Platform) []byte {
	return []byte(fmt.Sprintf(
		"schema = %q\nschema_version = %q\nhost_id = %q\nhost_name = %q\nplatform = %q\n",
		SchemaID,
		Version1,
		"0198f4c8-4a10-7b22-8b3c-1234567890ab",
		"fixture-host",
		platform.String(),
	))
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

type fakeFileSystem struct {
	regular     map[string][]byte
	directories map[string]bool
	modes       map[string]fs.FileMode
	statErrors  map[string]error
	readErrors  map[string]error
	readCount   map[string]int
	writeCount  int
}

func newFakeFileSystem() *fakeFileSystem {
	return &fakeFileSystem{
		regular:     make(map[string][]byte),
		directories: make(map[string]bool),
		modes:       make(map[string]fs.FileMode),
		statErrors:  make(map[string]error),
		readErrors:  make(map[string]error),
		readCount:   make(map[string]int),
	}
}

func (files *fakeFileSystem) Stat(name string) (fs.FileInfo, error) {
	if err := files.statErrors[name]; err != nil {
		return nil, err
	}
	if data, ok := files.regular[name]; ok {
		return fakeFileInfo{name: name, size: int64(len(data)), mode: 0o600}, nil
	}
	if files.directories[name] {
		return fakeFileInfo{name: name, mode: fs.ModeDir | 0o700}, nil
	}
	if mode, ok := files.modes[name]; ok {
		return fakeFileInfo{name: name, mode: mode}, nil
	}
	return nil, fs.ErrNotExist
}

func (files *fakeFileSystem) ReadFile(name string) ([]byte, error) {
	files.readCount[name]++
	if err := files.readErrors[name]; err != nil {
		return nil, err
	}
	data, ok := files.regular[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

type fakeFileInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (info fakeFileInfo) Name() string       { return info.name }
func (info fakeFileInfo) Size() int64        { return info.size }
func (info fakeFileInfo) Mode() fs.FileMode  { return info.mode }
func (info fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (info fakeFileInfo) IsDir() bool        { return info.mode.IsDir() }
func (info fakeFileInfo) Sys() any           { return nil }
