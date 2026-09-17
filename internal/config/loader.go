// Package config resolves the process-lifetime AX path snapshot, strictly
// decodes Configuration 1.0.0, 2.0.0, or 3.0.0, and translates legacy values
// into the current in-memory model. It never mutates configuration or roots;
// explicit durable migration remains a separate concern.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var (
	ErrInvalidContext             = errors.New("invalid configuration loader context")
	ErrUnknownPathClass           = errors.New("unknown AX path class")
	ErrPlatformDefaultUnavailable = errors.New("platform path default unavailable")
	ErrConfigParentNotFound       = errors.New("selected configuration parent directory not found")
	ErrConfigParentNotDirectory   = errors.New("selected configuration parent path is not a directory")
	ErrConfigNotRegular           = errors.New("selected configuration path is not a regular file")
	ErrConfigRead                 = errors.New("configuration file read failed")
	ErrRootNotDirectory           = errors.New("selected root path is not a directory")
	ErrRootInspect                = errors.New("selected root path inspection failed")
)

// PathClass names one member of the complete AX 1.0.0 path override registry.
type PathClass string

const (
	ConfigFile  PathClass = "config-file"
	DataRoot    PathClass = "data-root"
	StateRoot   PathClass = "state-root"
	CacheRoot   PathClass = "cache-root"
	RuntimeRoot PathClass = "runtime-root"
)

// OverrideSpec binds one path class to its command flag and sole AX_*
// environment override. The registry is vocabulary, not a capability claim.
type OverrideSpec struct {
	Class       PathClass
	Flag        string
	Environment string
}

var overrideRegistry = [...]OverrideSpec{
	{Class: ConfigFile, Flag: "--config", Environment: "AX_CONFIG"},
	{Class: DataRoot, Flag: "--data-dir", Environment: "AX_DATA_DIR"},
	{Class: StateRoot, Flag: "--state-dir", Environment: "AX_STATE_DIR"},
	{Class: CacheRoot, Flag: "--cache-dir", Environment: "AX_CACHE_DIR"},
	{Class: RuntimeRoot, Flag: "--runtime-dir", Environment: "AX_RUNTIME_DIR"},
}

// OverrideRegistry returns an isolated ordered copy of the complete registry.
func OverrideRegistry() []OverrideSpec {
	registry := make([]OverrideSpec, len(overrideRegistry))
	copy(registry, overrideRegistry[:])
	return registry
}

// Overrides carries values supplied by command flags. Empty entries are unset.
type Overrides map[PathClass]string

// Source identifies the selecting precedence layer without carrying its value.
type Source string

const (
	SourceFlag            Source = "flag"
	SourceEnvironment     Source = "environment"
	SourcePlatformDefault Source = "platform-default"
)

// ResolvedPath is an absolute platform-bound path plus safe provenance.
type ResolvedPath struct {
	Value  scalar.AbsolutePath
	Source Source
}

// ResolvedPaths is an isolated process-lifetime path snapshot.
type ResolvedPaths struct {
	values map[PathClass]ResolvedPath
}

func (paths ResolvedPaths) Path(class PathClass) (ResolvedPath, bool) {
	value, ok := paths.values[class]
	return value, ok
}

func (paths ResolvedPaths) All() map[PathClass]ResolvedPath {
	values := make(map[PathClass]ResolvedPath, len(paths.values))
	for class, value := range paths.values {
		values[class] = value
	}
	return values
}

// Inputs captures all process inputs read during one resolution.
type Inputs struct {
	Platform        scalar.Platform
	HomeDir         string
	TempDir         string
	WorkingDir      string
	BackendSettings BackendSettingsValidator
	LookupEnv       func(string) (string, bool)
	Stat            func(string) (fs.FileInfo, error)
	ReadFile        func(string) ([]byte, error)
	homeDirError    error
}

// Snapshot contains one immutable path decision and selected TOML document.
type Snapshot struct {
	paths         ResolvedPaths
	localPaths    localstore.ResolvedPaths
	document      []byte
	configuration *LoadedConfiguration
	configPresent bool
}

func (snapshot Snapshot) Paths() ResolvedPaths {
	return ResolvedPaths{values: snapshot.paths.All()}
}

// LocalPaths returns the immutable canonical pair minted by the owner-local
// path resolver. Config readers retain Paths for legacy diagnostics, but all
// trust/config pairing and durable mutation must use this value: its path map
// is private to internal/localstore and cannot be forged by callers.
func (snapshot Snapshot) LocalPaths() localstore.ResolvedPaths { return snapshot.localPaths }

func (snapshot Snapshot) Document() []byte {
	return append([]byte(nil), snapshot.document...)
}

// ConfigPresent distinguishes an existing empty configuration document from
// an admissible not-yet-created configuration path.
func (snapshot Snapshot) ConfigPresent() bool {
	return snapshot.configPresent
}

// Configuration returns an isolated current-model translation when a selected
// configuration file existed and passed exact versioned validation.
func (snapshot Snapshot) Configuration() (LoadedConfiguration, bool) {
	if snapshot.configuration == nil {
		return LoadedConfiguration{}, false
	}
	return LoadedConfiguration{
		SourceVersion: snapshot.configuration.SourceVersion,
		Value:         cloneConfiguration(snapshot.configuration.Value),
	}, true
}

// Error reports a failed class/layer without echoing machine-local values.
// Wrapped details remain available to errors.Is/errors.As through Unwrap, but
// are deliberately excluded from the rendered message because OS and
// filesystem errors may contain selected paths.
type Error struct {
	Operation string
	Class     PathClass
	Source    Source
	Err       error
}

func (err *Error) Error() string {
	if err.Class == "" {
		return fmt.Sprintf("configuration %s failed", err.Operation)
	}
	if err.Source == "" {
		return fmt.Sprintf("configuration %s for %s failed", err.Operation, err.Class)
	}
	return fmt.Sprintf("configuration %s for %s from %s failed", err.Operation, err.Class, err.Source)
}

func (err *Error) Unwrap() error { return err.Err }

// OSInputs captures real process inputs once. The caller supplies the exact
// runtime-probed AX platform so WSL2 is never guessed from GOOS.
func OSInputs(platform scalar.Platform) (Inputs, error) {
	if _, err := scalar.ParsePlatform(platform.String()); err != nil {
		return Inputs{}, loaderError(Error{Operation: "capture OS inputs", Err: ErrInvalidContext})
	}
	home, homeErr := os.UserHomeDir()
	working, err := os.Getwd()
	if err != nil {
		return Inputs{}, loaderError(Error{Operation: "capture working directory", Err: errors.Join(ErrInvalidContext, err)}) // config-refusal-subsumed: os.Getwd has no injectable seam and does not fail on any supported host, including a cwd unlinked underneath the process, so this guard stays fail-closed and is pinned by the refusal-subsumption inventory
	}
	return Inputs{
		Platform:   platform,
		HomeDir:    cleanCapturedPath(home),
		TempDir:    filepath.Clean(os.TempDir()),
		WorkingDir: filepath.Clean(working),
		LookupEnv:  os.LookupEnv,
		// Read-side path selection resolves symlinks and then enforces the
		// Section 3.2 value kinds on the resolved target: a configuration file
		// symlinked onto a regular file loads, a root symlinked onto a
		// directory loads, and either one pointed at the wrong kind still
		// fails closed. Section 3.2 and Section 6.1 state no no-follow
		// requirement for these five classes; where the specification wants a
		// path verified without following a symlink it says so explicitly, as
		// it does for the Section 3.2 tmux socket parent. Refusing a symlinked
		// XDG or Application Support root would invent a constraint and break
		// an ordinary dotfiles layout, so this seam stays symlink-following.
		// The durable migration path is deliberately stricter; see
		// osMigrationFileSystem.Stat.
		Stat:         os.Stat,
		ReadFile:     os.ReadFile,
		homeDirError: homeErr,
	}, nil
}

func cleanCapturedPath(value string) string {
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}

func LoadOS(platform scalar.Platform, overrides Overrides) (Snapshot, error) {
	inputs, err := OSInputs(platform)
	if err != nil {
		return Snapshot{}, err
	}
	return Load(inputs, overrides)
}

// Load resolves all roots, then reads exactly the selected regular file or
// admits a not-yet-created file whose parent is an existing directory.
func Load(inputs Inputs, overrides Overrides) (Snapshot, error) {
	paths, localPaths, err := resolvePathSnapshot(inputs, overrides)
	if err != nil {
		return Snapshot{}, err
	}
	if inputs.Stat == nil || inputs.ReadFile == nil {
		return Snapshot{}, loaderError(Error{Operation: "read selected file", Err: ErrInvalidContext})
	}
	if err := validateRootKinds(paths, inputs.Stat); err != nil {
		return Snapshot{}, err
	}

	selected, ok := paths.Path(ConfigFile)
	if !ok {
		return Snapshot{}, loaderError(Error{Operation: "read selected file", Class: ConfigFile, Err: ErrInvalidContext}) // config-refusal-subsumed: ResolvePaths populates every overrideRegistry class, pinned by TestResolvePathsPopulatesEveryOverrideRegistryClass
	}
	filename := selected.Value.String()
	info, err := inputs.Stat(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return loadAbsentConfig(inputs, paths, localPaths, selected)
		}
		return Snapshot{}, loaderError(Error{
			Operation: "stat selected file",
			Class:     ConfigFile,
			Source:    selected.Source,
			Err:       ErrConfigRead,
		})
	}
	if !info.Mode().IsRegular() {
		return Snapshot{}, loaderError(Error{
			Operation: "stat selected file",
			Class:     ConfigFile,
			Source:    selected.Source,
			Err:       ErrConfigNotRegular,
		})
	}

	document, err := inputs.ReadFile(filename)
	if err != nil {
		return Snapshot{}, loaderError(Error{
			Operation: "read selected file",
			Class:     ConfigFile,
			Source:    selected.Source,
			Err:       ErrConfigRead,
		})
	}
	configuration, err := Decode(document, DecodeContext{
		RuntimePlatform: inputs.Platform,
		BackendSettings: inputs.BackendSettings,
	})
	if err != nil {
		return Snapshot{}, loaderError(Error{
			Operation: "decode selected file",
			Class:     ConfigFile,
			Source:    selected.Source,
			Err:       err,
		})
	}
	return Snapshot{
		paths:         ResolvedPaths{values: paths.All()},
		localPaths:    localPaths,
		document:      append([]byte(nil), document...),
		configuration: &configuration,
		configPresent: true,
	}, nil
}

func loadAbsentConfig(inputs Inputs, paths ResolvedPaths, localPaths localstore.ResolvedPaths, selected ResolvedPath) (Snapshot, error) {
	parent, err := configParent(inputs.Platform, selected.Value.String())
	if err != nil {
		return Snapshot{}, loaderError(Error{
			Operation: "resolve selected file parent",
			Class:     ConfigFile,
			Source:    selected.Source,
			Err:       ErrInvalidContext,
		})
	}
	info, err := inputs.Stat(parent)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Snapshot{}, loaderError(Error{
				Operation: "stat selected file parent",
				Class:     ConfigFile,
				Source:    selected.Source,
				Err:       ErrConfigParentNotFound,
			})
		}
		return Snapshot{}, loaderError(Error{
			Operation: "stat selected file parent",
			Class:     ConfigFile,
			Source:    selected.Source,
			Err:       ErrConfigRead,
		})
	}
	if !info.IsDir() {
		return Snapshot{}, loaderError(Error{
			Operation: "stat selected file parent",
			Class:     ConfigFile,
			Source:    selected.Source,
			Err:       ErrConfigParentNotDirectory,
		})
	}
	return Snapshot{
		paths:         ResolvedPaths{values: paths.All()},
		localPaths:    localPaths,
		configPresent: false,
	}, nil
}

// resolvePathSnapshot is the only configuration production path-resolution
// entry. localstore.ResolvePaths mints one immutable five-class result; the
// config ResolvedPaths value is a compatibility projection of those exact
// bytes, while pair mutations retain the opaque localstore result itself.
func resolvePathSnapshot(inputs Inputs, overrides Overrides) (ResolvedPaths, localstore.ResolvedPaths, error) {
	if _, err := scalar.ParsePlatform(inputs.Platform.String()); err != nil || inputs.LookupEnv == nil {
		return ResolvedPaths{}, localstore.ResolvedPaths{}, loaderError(Error{Operation: "validate inputs", Err: ErrInvalidContext})
	}
	if err := validateOverrideClasses(overrides); err != nil {
		return ResolvedPaths{}, localstore.ResolvedPaths{}, err
	}
	resolved, err := localstore.ResolvePaths(localstore.ResolveRequest{
		Platform:     inputs.Platform,
		Flags:        localstoreFlags(overrides),
		LookupEnv:    inputs.LookupEnv,
		HomeDir:      inputs.HomeDir,
		TemporaryDir: inputs.TempDir,
		WorkingDir:   inputs.WorkingDir,
	})
	if err != nil {
		return ResolvedPaths{}, localstore.ResolvedPaths{}, translateLocalPathError(inputs, err)
	}
	paths, err := projectLocalPaths(resolved)
	if err != nil {
		return ResolvedPaths{}, localstore.ResolvedPaths{}, err
	}
	return paths, resolved, nil
}

func resolveLocalPaths(inputs Inputs, overrides Overrides) (localstore.ResolvedPaths, error) {
	_, paths, err := resolvePathSnapshot(inputs, overrides)
	return paths, err
}

// ResolvePaths preserves the historical config-facing value/provenance API,
// but delegates resolution to the same localstore resolver used by Load and
// every Host Trust Store pair consumer.
func ResolvePaths(inputs Inputs, overrides Overrides) (ResolvedPaths, error) {
	paths, _, err := resolvePathSnapshot(inputs, overrides)
	return paths, err
}

func localstoreFlags(overrides Overrides) map[string]string {
	flags := make(map[string]string, len(overrides))
	for _, specification := range overrideRegistry {
		if value := overrides[specification.Class]; value != "" {
			flags[specification.Flag] = value
		}
	}
	return flags
}

func projectLocalPaths(localPaths localstore.ResolvedPaths) (ResolvedPaths, error) {
	values := make(map[PathClass]ResolvedPath, len(overrideRegistry))
	for _, specification := range overrideRegistry {
		localClass := localPathClass(specification.Class)
		resolved, ok := localPaths.Path(localClass)
		if !ok {
			return ResolvedPaths{}, loaderError(Error{Operation: "project resolved paths", Class: specification.Class, Err: ErrInvalidContext}) // config-refusal-subsumed: the pinned localstore registry and resolver always return all five classes; a missing projection member is impossible at this production boundary
		}
		values[specification.Class] = ResolvedPath{Value: resolved.Value, Source: configPathSource(resolved.Source)}
	}
	return ResolvedPaths{values: values}, nil
}

func localPathClass(class PathClass) localstore.PathClass {
	switch class {
	case ConfigFile:
		return localstore.PathConfig
	case DataRoot:
		return localstore.PathData
	case StateRoot:
		return localstore.PathState
	case CacheRoot:
		return localstore.PathCache
	case RuntimeRoot:
		return localstore.PathRuntime
	default:
		return localstore.PathClass(class)
	}
}

func configPathSource(source localstore.PathSource) Source {
	switch source {
	case localstore.PathSourceFlag:
		return SourceFlag
	case localstore.PathSourceEnvironment:
		return SourceEnvironment
	case localstore.PathSourceDefault:
		return SourcePlatformDefault
	default:
		return Source("")
	}
}

func configPathClass(class localstore.PathClass) PathClass {
	switch class {
	case localstore.PathConfig:
		return ConfigFile
	case localstore.PathData:
		return DataRoot
	case localstore.PathState:
		return StateRoot
	case localstore.PathCache:
		return CacheRoot
	case localstore.PathRuntime:
		return RuntimeRoot
	default:
		return PathClass(class)
	}
}

func translateLocalPathError(inputs Inputs, err error) error {
	var pathErr *localstore.PathResolutionError
	if !errors.As(err, &pathErr) {
		cause := error(ErrInvalidContext)
		if errors.Is(err, localstore.ErrInvalidPath) || errors.Is(err, localstore.ErrUnsupportedPlatform) {
			cause = ErrInvalidContext
		} else {
			cause = errors.Join(ErrInvalidContext, err)
		}
		return loaderError(Error{Operation: "resolve paths", Err: cause}) // config-refusal-subsumed: every current localstore path refusal is a typed class/source error; this defensive branch only covers a corrupt registry or future resolver error outside that contract
	}
	configClass := configPathClass(pathErr.Class)
	source := configPathSource(pathErr.Source)
	cause := error(ErrInvalidContext)
	operation := "resolve absolute path"
	if errors.Is(pathErr.Err, localstore.ErrPathDefaultUnavailable) {
		cause = ErrPlatformDefaultUnavailable
		operation = "resolve platform default"
		if pathErr.Source == localstore.PathSourceDefault && inputs.HomeDir == "" && inputs.homeDirError != nil && homeDerivedPathClass(inputs.Platform, configClass) {
			cause = errors.Join(ErrPlatformDefaultUnavailable, inputs.homeDirError)
		}
	}
	return loaderError(Error{Operation: operation, Class: configClass, Source: source, Err: cause})
}

func homeDerivedPathClass(platform scalar.Platform, class PathClass) bool {
	switch platform {
	case scalar.PlatformMacOS, scalar.PlatformLinux, scalar.PlatformWSL2:
		switch class {
		case ConfigFile, DataRoot, StateRoot, CacheRoot:
			return true
		}
	}
	return false
}

func configPathFromLocalPaths(paths localstore.ResolvedPaths) (string, error) {
	selected, ok := paths.Path(localstore.PathConfig)
	if !ok || selected.Value.String() == "" {
		return "", loaderError(Error{Operation: "resolve selected file", Class: ConfigFile, Err: ErrInvalidContext}) // config-refusal-subsumed: every pair consumer receives the opaque five-class result minted by localstore.ResolvePaths
	}
	return selected.Value.String(), nil
}

func validateRootKinds(paths ResolvedPaths, stat func(string) (fs.FileInfo, error)) error {
	for _, specification := range overrideRegistry {
		if specification.Class == ConfigFile {
			continue
		}
		resolved, ok := paths.Path(specification.Class)
		if !ok {
			return loaderError(Error{Operation: "inspect selected root", Class: specification.Class, Err: ErrInvalidContext}) // config-refusal-subsumed: ResolvePaths populates every overrideRegistry class, pinned by TestResolvePathsPopulatesEveryOverrideRegistryClass
		}
		info, err := stat(resolved.Value.String())
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return loaderError(Error{
				Operation: "inspect selected root",
				Class:     specification.Class,
				Source:    resolved.Source,
				Err:       ErrRootInspect,
			})
		}
		if !info.IsDir() {
			return loaderError(Error{
				Operation: "inspect selected root",
				Class:     specification.Class,
				Source:    resolved.Source,
				Err:       ErrRootNotDirectory,
			})
		}
	}
	return nil
}

func validateOverrideClasses(overrides Overrides) error {
	known := make(map[PathClass]struct{}, len(overrideRegistry))
	for _, specification := range overrideRegistry {
		known[specification.Class] = struct{}{}
	}
	for class := range overrides {
		if _, ok := known[class]; !ok {
			return loaderError(Error{Operation: "validate override", Class: class, Source: SourceFlag, Err: ErrUnknownPathClass})
		}
	}
	return nil
}

func configParent(platform scalar.Platform, filename string) (string, error) {
	switch platform {
	case scalar.PlatformWindows:
		if len(filename) >= 3 && isASCIILetter(filename[0]) && filename[1] == ':' && filename[2] == '\\' {
			separator := strings.LastIndex(filename, "\\")
			if separator < 2 || separator == len(filename)-1 {
				return "", ErrInvalidContext
			}
			if separator == 2 {
				return filename[:3], nil
			}
			return filename[:separator], nil
		}
		if strings.HasPrefix(filename, "\\\\") {
			segments := strings.Split(filename[2:], "\\")
			if len(segments) < 3 || segments[0] == "" || segments[1] == "" || segments[len(segments)-1] == "" {
				return "", ErrInvalidContext
			}
			return "\\\\" + strings.Join(segments[:len(segments)-1], "\\"), nil
		}
		return "", ErrInvalidContext
	default:
		if !strings.HasPrefix(filename, "/") || filename == "/" {
			return "", ErrInvalidContext
		}
		return path.Dir(filename), nil
	}
}

func isASCIILetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

// loaderError is the only construction site for *Error. Routing every path
// refusal through one indirection lets the derived refusal inventory observe
// each production site and lets that inventory assert its own completeness.
var loaderError = func(value Error) error {
	return &value
}
