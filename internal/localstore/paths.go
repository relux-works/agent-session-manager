// Package localstore implements the AX owner-local path layout and immutable
// SHA-256 blob sink. It does not advertise runtime platform support.
package localstore

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/specpin"
)

var (
	ErrPathRegistry           = errors.New("AX path registry is invalid")
	ErrUnknownPathFlag        = errors.New("unknown AX path flag")
	ErrInvalidPath            = errors.New("invalid AX local path")
	ErrPathDefaultUnavailable = errors.New("AX platform path default is unavailable")
	ErrUnsafeOwnership        = errors.New("AX local path is not owner-only")
	ErrUnsupportedPlatform    = errors.New("AX local filesystem platform is unsupported")
)

const PathRegistrySHA256 = "1f7100bf212429bb519bf56d04e8c03b6bd2d2543e7307463a43fbe58e1eb9f0"

type PathClass string

const (
	PathConfig  PathClass = "config"
	PathData    PathClass = "data"
	PathState   PathClass = "state"
	PathCache   PathClass = "cache"
	PathRuntime PathClass = "runtime"
)

type PathValueKind string

const (
	PathValueConfigurationFile PathValueKind = "configuration_file"
	PathValueDirectory         PathValueKind = "directory"
)

type PathSource string

const (
	PathSourceFlag        PathSource = "flag"
	PathSourceEnvironment PathSource = "environment"
	PathSourceDefault     PathSource = "default"
)

type PathDefinition struct {
	Class       PathClass
	Flag        string
	Environment string
	ValueKind   PathValueKind
}

type ResolveRequest struct {
	Platform     scalar.Platform
	Flags        map[string]string
	Environment  map[string]string
	HomeDir      string
	TemporaryDir string
}

type ResolvedPath struct {
	Class  PathClass
	Value  scalar.AbsolutePath
	Source PathSource
}

type ResolvedPaths struct {
	platform scalar.Platform
	paths    map[PathClass]ResolvedPath
}

func (resolved ResolvedPaths) Platform() scalar.Platform { return resolved.platform }

func (resolved ResolvedPaths) Path(class PathClass) (ResolvedPath, bool) {
	value, ok := resolved.paths[class]
	return value, ok
}

func (resolved ResolvedPaths) Paths() []ResolvedPath {
	definitions, err := PathDefinitions()
	if err != nil {
		return nil
	}
	values := make([]ResolvedPath, 0, len(definitions))
	for _, definition := range definitions {
		if value, ok := resolved.paths[definition.Class]; ok {
			values = append(values, value)
		}
	}
	return values
}

type pathRegistry struct {
	Format          string                   `json:"format"`
	FormatVersion   int                      `json:"format_version"`
	RegistryVersion string                   `json:"registry_version"`
	Source          pathRegistrySource       `json:"source"`
	Paths           []pathDefinitionDocument `json:"paths"`
}

type pathRegistrySource struct {
	Repository     string `json:"repository"`
	Release        string `json:"release"`
	Commit         string `json:"commit"`
	DocumentPath   string `json:"document_path"`
	DocumentSHA256 string `json:"document_sha256"`
	Section        string `json:"section"`
}

type pathDefinitionDocument struct {
	Class       PathClass     `json:"class"`
	Flag        string        `json:"flag"`
	Environment string        `json:"environment"`
	ValueKind   PathValueKind `json:"value_kind"`
}

//go:embed path_registry.v0.5.0.json
var pathRegistryBytes []byte

var (
	pathRegistryOnce        sync.Once
	pathRegistryDefinitions []PathDefinition
	pathRegistryError       error
)

func PathDefinitions() ([]PathDefinition, error) {
	pathRegistryOnce.Do(loadPathDefinitions)
	if pathRegistryError != nil {
		return nil, pathRegistryError
	}
	return slices.Clone(pathRegistryDefinitions), nil
}

func loadPathDefinitions() {
	pathRegistryDefinitions, pathRegistryError = verifyPathRegistry(pathRegistryBytes)
}

// verifyPathRegistry accepts only the reviewed embedded registry bytes before
// interpreting their rows. A structurally valid replacement must not be able
// to mint its own path contract.
func verifyPathRegistry(candidate []byte) ([]PathDefinition, error) {
	if digest := fmt.Sprintf("%x", sha256.Sum256(candidate)); digest != PathRegistrySHA256 {
		return nil, fmt.Errorf("%w: byte digest is %s, want %s", ErrPathRegistry, digest, PathRegistrySHA256)
	}
	return decodePathRegistry(candidate)
}

func decodePathRegistry(candidate []byte) ([]PathDefinition, error) {
	decoder := json.NewDecoder(bytes.NewReader(candidate))
	decoder.DisallowUnknownFields()
	var registry pathRegistry
	if err := decoder.Decode(&registry); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrPathRegistry, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: trailing JSON", ErrPathRegistry)
	}
	manifest, err := specpin.Current()
	if err != nil {
		return nil, fmt.Errorf("%w: verify source pin: %v", ErrPathRegistry, err)
	}
	source := registry.Source
	if registry.Format != "ax-platform-path-registry" || registry.FormatVersion != 1 || registry.RegistryVersion != "1.0.0" ||
		source.Repository != manifest.Source.Repository || source.Release != manifest.Source.Release ||
		source.Commit != manifest.Source.Commit || source.DocumentPath != manifest.Source.Document.Path ||
		source.DocumentSHA256 != manifest.Source.Document.SHA256 || source.Section != "3.2" {
		return nil, fmt.Errorf("%w: identity differs from pinned Section 3.2", ErrPathRegistry)
	}
	if len(registry.Paths) != 5 {
		return nil, fmt.Errorf("%w: v1.0.0 path count is %d, want 5", ErrPathRegistry, len(registry.Paths))
	}
	seenClasses := make(map[PathClass]struct{}, len(registry.Paths))
	seenFlags := make(map[string]struct{}, len(registry.Paths))
	seenEnvironment := make(map[string]struct{}, len(registry.Paths))
	definitions := make([]PathDefinition, 0, len(registry.Paths))
	for index, document := range registry.Paths {
		if !validPathClass(document.Class) || !validPathValueKind(document.Class, document.ValueKind) ||
			!strings.HasPrefix(document.Flag, "--") || !strings.HasPrefix(document.Environment, "AX_") {
			return nil, fmt.Errorf("%w: malformed row %d", ErrPathRegistry, index)
		}
		if _, duplicate := seenClasses[document.Class]; duplicate {
			return nil, fmt.Errorf("%w: duplicate class %q", ErrPathRegistry, document.Class)
		}
		if _, duplicate := seenFlags[document.Flag]; duplicate {
			return nil, fmt.Errorf("%w: duplicate flag %q", ErrPathRegistry, document.Flag)
		}
		if _, duplicate := seenEnvironment[document.Environment]; duplicate {
			return nil, fmt.Errorf("%w: duplicate environment %q", ErrPathRegistry, document.Environment)
		}
		seenClasses[document.Class] = struct{}{}
		seenFlags[document.Flag] = struct{}{}
		seenEnvironment[document.Environment] = struct{}{}
		definitions = append(definitions, PathDefinition(document))
	}
	return definitions, nil
}

func validPathClass(class PathClass) bool {
	switch class {
	case PathConfig, PathData, PathState, PathCache, PathRuntime:
		return true
	default:
		return false
	}
}

func validPathValueKind(class PathClass, kind PathValueKind) bool {
	if class == PathConfig {
		return kind == PathValueConfigurationFile
	}
	return kind == PathValueDirectory
}

// ResolvePaths is the production deterministic path-resolution entry point.
// Callers capture flags, environment, home, and temporary inputs once, then
// keep the returned roots for the process lifetime.
func ResolvePaths(request ResolveRequest) (ResolvedPaths, error) {
	// This early refusal keeps the public error anchored to the platform input.
	// It is independently subsumed by scalar.ParseAbsolutePath below, which
	// calls scalar.ParsePlatform for every row in the pinned non-empty registry.
	if _, err := scalar.ParsePlatform(request.Platform.String()); err != nil {
		return ResolvedPaths{}, fmt.Errorf("%w: platform: %v", ErrInvalidPath, err)
	}
	definitions, err := PathDefinitions()
	if err != nil {
		return ResolvedPaths{}, err
	}
	knownFlags := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		knownFlags[definition.Flag] = struct{}{}
	}
	for flag := range request.Flags {
		if _, ok := knownFlags[flag]; !ok {
			return ResolvedPaths{}, fmt.Errorf("%w: %s", ErrUnknownPathFlag, flag)
		}
	}
	resolved := ResolvedPaths{platform: request.Platform, paths: make(map[PathClass]ResolvedPath, len(definitions))}
	for _, definition := range definitions {
		candidate := ""
		source := PathSource("")
		if value, present := request.Flags[definition.Flag]; present {
			if value == "" {
				return ResolvedPaths{}, fmt.Errorf("%w: %s must not be empty", ErrInvalidPath, definition.Flag)
			}
			candidate = value
			source = PathSourceFlag
		} else if value, present := environmentValue(request.Platform, request.Environment, definition.Environment); present && value != "" {
			candidate = value
			source = PathSourceEnvironment
		} else {
			candidate, err = platformDefault(request, definition.Class)
			if err != nil {
				return ResolvedPaths{}, err
			}
			source = PathSourceDefault
		}
		absolute, parseErr := scalar.ParseAbsolutePath(request.Platform, candidate)
		if parseErr != nil {
			return ResolvedPaths{}, fmt.Errorf("%w: %s: %v", ErrInvalidPath, definition.Class, parseErr)
		}
		resolved.paths[definition.Class] = ResolvedPath{Class: definition.Class, Value: absolute, Source: source}
	}
	return resolved, nil
}

func platformDefault(request ResolveRequest, class PathClass) (string, error) {
	join := func(elements ...string) string { return joinPlatform(request.Platform, elements...) }
	switch request.Platform {
	case scalar.PlatformMacOS:
		if class == PathRuntime {
			temporary, err := requireAbsoluteInput(request.Platform, "temporary directory", request.TemporaryDir)
			if err != nil {
				return "", err
			}
			return join(temporary, "ax"), nil
		}
		home, err := requireAbsoluteInput(request.Platform, "home directory", request.HomeDir)
		if err != nil {
			return "", err
		}
		switch class {
		case PathConfig:
			configBase, err := optionalAbsoluteEnvironment(request, "XDG_CONFIG_HOME", join(home, ".config"))
			if err != nil {
				return "", err
			}
			return join(configBase, "ax", "config.toml"), nil
		case PathData:
			return join(home, "Library", "Application Support", "ax"), nil
		case PathState:
			return join(home, "Library", "Application Support", "ax", "state"), nil
		case PathCache:
			return join(home, "Library", "Caches", "ax"), nil
		}
	case scalar.PlatformLinux, scalar.PlatformWSL2:
		if class == PathRuntime {
			runtimeBase, present := environmentValue(request.Platform, request.Environment, "XDG_RUNTIME_DIR")
			if !present || runtimeBase == "" {
				return "", fmt.Errorf("%w: XDG_RUNTIME_DIR", ErrPathDefaultUnavailable)
			}
			if _, err := requireAbsoluteInput(request.Platform, "XDG_RUNTIME_DIR", runtimeBase); err != nil {
				return "", err
			}
			return join(runtimeBase, "ax"), nil
		}
		home, err := requireAbsoluteInput(request.Platform, "home directory", request.HomeDir)
		if err != nil {
			return "", err
		}
		switch class {
		case PathConfig:
			base, err := optionalAbsoluteEnvironment(request, "XDG_CONFIG_HOME", join(home, ".config"))
			if err != nil {
				return "", err
			}
			return join(base, "ax", "config.toml"), nil
		case PathData:
			base, err := optionalAbsoluteEnvironment(request, "XDG_DATA_HOME", join(home, ".local", "share"))
			if err != nil {
				return "", err
			}
			return join(base, "ax"), nil
		case PathState:
			base, err := optionalAbsoluteEnvironment(request, "XDG_STATE_HOME", join(home, ".local", "state"))
			if err != nil {
				return "", err
			}
			return join(base, "ax"), nil
		case PathCache:
			base, err := optionalAbsoluteEnvironment(request, "XDG_CACHE_HOME", join(home, ".cache"))
			if err != nil {
				return "", err
			}
			return join(base, "ax"), nil
		}
	case scalar.PlatformWindows:
		appData, appDataPresent := environmentValue(request.Platform, request.Environment, "APPDATA")
		localAppData, localAppDataPresent := environmentValue(request.Platform, request.Environment, "LOCALAPPDATA")
		if class == PathConfig {
			if !appDataPresent || appData == "" {
				return "", fmt.Errorf("%w: APPDATA", ErrPathDefaultUnavailable)
			}
			if _, err := requireAbsoluteInput(request.Platform, "APPDATA", appData); err != nil {
				return "", err
			}
			return join(appData, "ax", "config.toml"), nil
		}
		if !localAppDataPresent || localAppData == "" {
			return "", fmt.Errorf("%w: LOCALAPPDATA", ErrPathDefaultUnavailable)
		}
		if _, err := requireAbsoluteInput(request.Platform, "LOCALAPPDATA", localAppData); err != nil {
			return "", err
		}
		return join(localAppData, "ax", string(class)), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedPlatform, request.Platform)
	}
	return "", fmt.Errorf("%w: unknown path class %q", ErrPathRegistry, class)
}

func optionalAbsoluteEnvironment(request ResolveRequest, name, fallback string) (string, error) {
	value, present := environmentValue(request.Platform, request.Environment, name)
	if !present || value == "" {
		return fallback, nil
	}
	return requireAbsoluteInput(request.Platform, name, value)
}

func requireAbsoluteInput(platform scalar.Platform, name, value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%w: %s is empty", ErrPathDefaultUnavailable, name)
	}
	if _, err := scalar.ParseAbsolutePath(platform, value); err != nil {
		return "", fmt.Errorf("%w: %s: %v", ErrInvalidPath, name, err)
	}
	return value, nil
}

func environmentValue(platform scalar.Platform, environment map[string]string, name string) (string, bool) {
	if platform != scalar.PlatformWindows {
		value, ok := environment[name]
		return value, ok
	}
	for key, value := range environment {
		if strings.EqualFold(key, name) {
			return value, true
		}
	}
	return "", false
}

func joinPlatform(platform scalar.Platform, elements ...string) string {
	if platform == scalar.PlatformWindows {
		result := strings.TrimRight(elements[0], `\`)
		for _, element := range elements[1:] {
			result += `\` + strings.Trim(element, `\`)
		}
		return result
	}
	return path.Join(elements...)
}

// InitializeLayout creates or verifies the owner-only roots selected for this
// native process. Windows refuses until its user-only DACL implementation lands.
func InitializeLayout(resolved ResolvedPaths) error {
	if !nativePlatformMatches(resolved.platform) {
		return fmt.Errorf("%w: cannot initialize %s paths on %s", ErrUnsupportedPlatform, resolved.platform, runtime.GOOS)
	}
	definitions, err := PathDefinitions()
	if err != nil {
		return err
	}
	for _, definition := range definitions {
		value, ok := resolved.paths[definition.Class]
		if !ok {
			return fmt.Errorf("%w: missing resolved %s path", ErrInvalidPath, definition.Class)
		}
		if definition.ValueKind == PathValueConfigurationFile {
			if err := initializeConfigPath(value); err != nil {
				return err
			}
			continue
		}
		if err := ensureOwnerDirectory(value.Value.String()); err != nil {
			return fmt.Errorf("initialize %s root: %w", definition.Class, err)
		}
	}
	return nil
}

func initializeConfigPath(value ResolvedPath) error {
	filename := value.Value.String()
	parent := filepath.Dir(filename)
	if value.Source == PathSourceDefault {
		if err := ensureOwnerDirectory(parent); err != nil {
			return fmt.Errorf("initialize config root: %w", err)
		}
	} else if err := verifyOwnerDirectory(parent); err != nil {
		return fmt.Errorf("verify config root: %w", err)
	}
	info, err := os.Lstat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lstat config file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: config path is not a regular file", ErrUnsafeOwnership)
	}
	if err := verifyOwnerFileInfo(info, 0o600); err != nil {
		return fmt.Errorf("verify config file: %w", err)
	}
	return nil
}

func nativePlatformMatches(platform scalar.Platform) bool {
	switch runtime.GOOS {
	case "darwin":
		return platform == scalar.PlatformMacOS
	case "linux":
		return platform == scalar.PlatformLinux || platform == scalar.PlatformWSL2
	case "windows":
		return platform == scalar.PlatformWindows
	default:
		return false
	}
}

func ensureOwnerDirectory(name string) error {
	info, err := os.Lstat(name)
	if err == nil {
		return verifyOwnerDirectoryInfo(info)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(name)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create parent: %w", err)
	}
	created := false
	if err := os.Mkdir(name, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create directory: %w", err)
		}
	} else {
		created = true
	}
	if created {
		if err := os.Chmod(name, 0o700); err != nil {
			return fmt.Errorf("set owner-only directory mode: %w", err)
		}
	}
	if err := verifyOwnerDirectory(name); err != nil {
		return err
	}
	if err := syncDirectory(parent); err != nil {
		return fmt.Errorf("sync parent directory: %w", err)
	}
	return nil
}

func verifyOwnerDirectory(name string) error {
	info, err := os.Lstat(name)
	if err != nil {
		return err
	}
	return verifyOwnerDirectoryInfo(info)
}

func verifyOwnerDirectoryInfo(info os.FileInfo) error {
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: expected a real directory", ErrUnsafeOwnership)
	}
	return verifyOwnerFileInfo(info, 0o700)
}

func syncDirectory(name string) error {
	directory, err := os.Open(name)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
