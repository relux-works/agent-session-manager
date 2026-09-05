// Package terminalbackend implements the TerminalBackend registry: canonical
// backend identities, implementation/protocol version tuples, external
// executable trust admission, and duplicate or drift refusal.
//
// Normative scope is relux-works/agent-session-manager-spec@v0.5.0 §4.B
// (registry, identity, discovery, trust), §4.1 (backend interface versions),
// §6.5 (Configuration 3.0.0 TerminalBackend extension), and §7.A (Provider
// Protocol 3.0.0 Terminal Instance binding). Manifest/Probe schemas,
// capability evidence, and generation-bound admission are defined in
// manifest.go; lifecycle operations and legacy v0.4.3 translation belong to
// the sibling story tasks and are deliberately not defined here.
package terminalbackend

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Wire error codes are the exact Section 15.3 codes the registry may report.
// They are a subset of the pinned catalog error vocabulary; no other code is
// emitted by this package.
const (
	// CodeNotFound reports an unknown or malformed backend identity. A failed
	// read is never absence: callers must not treat this as "no backend".
	CodeNotFound = "terminal_backend_not_found"
	// CodeAmbiguous reports a duplicate or ambiguous backend ID. Re-registering
	// an identical record is still refused: identity registration is not
	// idempotent.
	CodeAmbiguous = "terminal_backend_ambiguous"
	// CodeUntrusted reports a disabled, unresolvable, or digest-mismatched
	// external executable. Trust is established before any probe.
	CodeUntrusted = "terminal_backend_untrusted"
	// CodeDrift reports a re-registration whose version tuple, kind,
	// platforms, or executable digest differs from the admitted record.
	CodeDrift = "terminal_backend_implementation_drift"
	// CodeRestoreMismatch reports an attempt to substitute the configured
	// default for the exact prior binding on restore or resume.
	CodeRestoreMismatch = "terminal_backend_restore_mismatch"
	// CodeStaleGeneration reports a generation that is out of bound or does
	// not match the validated binding.
	CodeStaleGeneration = "terminal_backend_stale_generation"
)

// Canonical built-in backend identities. The ax. namespace is reserved (§4.B);
// no third-party registration may use it.
const (
	BuiltinTmux   = "ax.tmux"
	BuiltinConpty = "ax.conpty"
)

// reservedNamespace is the only namespace this implementation may mint.
const reservedNamespace = "ax."

// maxIDBytes is the Section 4.B bound: 1-128 ASCII bytes.
const maxIDBytes = 128

// maxGenerationRunes is the Section 4.B/7.A bound: string[1..256] counts
// UTF-8 characters (SPEC.md:321), not bytes; terminal_backend_id is the
// only member of this package still bounded in ASCII bytes.
const maxGenerationRunes = 256

// maxProtocolVersions is the Section 4.B bound: sorted unique semver[1..32].
const maxProtocolVersions = 32

var (
	idPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	// semverPattern implements Semantic Versioning 2.0.0 in full, matching the
	// grammar the canonical and config packages enforce for the same members.
	semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
)

// Error is a registry refusal. Code is always one of the Code* constants;
// BackendID names the identity at issue when there is one. Detail is a static
// clause: it never echoes paths, digests, generations, or other local data.
type Error struct {
	Code      string
	BackendID string
	Detail    string
}

func (err *Error) Error() string {
	if err.BackendID == "" {
		return "terminal backend refused: " + err.Code + " at " + err.Detail
	}
	return "terminal backend refused: " + err.Code + " for " + err.BackendID + " at " + err.Detail
}

// errorCode reports whether err carries the given wire code.
func errorCode(err error, code string) bool {
	var refusal *Error
	if !errors.As(err, &refusal) {
		return false
	}
	return refusal.Code == code
}

// IsNotFound reports CodeNotFound refusals.
func IsNotFound(err error) bool { return errorCode(err, CodeNotFound) }

// IsAmbiguous reports CodeAmbiguous refusals.
func IsAmbiguous(err error) bool { return errorCode(err, CodeAmbiguous) }

// IsUntrusted reports CodeUntrusted refusals.
func IsUntrusted(err error) bool { return errorCode(err, CodeUntrusted) }

// IsDrift reports CodeDrift refusals.
func IsDrift(err error) bool { return errorCode(err, CodeDrift) }

// IsRestoreMismatch reports CodeRestoreMismatch refusals.
func IsRestoreMismatch(err error) bool { return errorCode(err, CodeRestoreMismatch) }

// IsStaleGeneration reports CodeStaleGeneration refusals.
func IsStaleGeneration(err error) bool { return errorCode(err, CodeStaleGeneration) }

// ParseID validates a canonical terminal backend identity: 1-128 ASCII bytes
// matching [a-z][a-z0-9]*(?:[.-][a-z0-9]+)*. It admits the reserved ax.
// namespace only for the two canonical built-ins; every other ax.-prefixed ID
// is refused so a third party cannot mint a trusted-looking identity.
func ParseID(value string) (string, error) {
	if !utf8.ValidString(value) || len(value) < 1 || len(value) > maxIDBytes {
		return "", &Error{Code: CodeNotFound, Detail: "terminal_backend_id bound"}
	}
	if !idPattern.MatchString(value) {
		return "", &Error{Code: CodeNotFound, BackendID: value, Detail: "terminal_backend_id grammar"}
	}
	if strings.HasPrefix(value, reservedNamespace) && value != BuiltinTmux && value != BuiltinConpty {
		return "", &Error{Code: CodeNotFound, BackendID: value, Detail: "terminal_backend_id reserved namespace"}
	}
	return value, nil
}

// mustParseID is the internal admission used where the caller already holds a
// validated record. It duplicates the ParseID rule so a Registry can never
// hold an identity its own grammar refuses.
func mustParseID(value string) error {
	_, err := ParseID(value)
	return err
}

// Kind is the Section 4.B implementation_kind vocabulary.
type Kind string

// Implementation kinds. Built-ins run inside the ax binary; externals arrive
// as adapter executables over the Section 4.B argv/framing wire.
const (
	KindBuiltinGo         Kind = "builtin_go"
	KindLocalProgram      Kind = "local_program"
	KindTrustedExecutable Kind = "trusted_executable"
	KindNativeRuntime     Kind = "native_runtime"
)

// parseKind admits exactly the closed Section 4.B vocabulary.
func parseKind(value string) (Kind, error) {
	switch Kind(value) {
	case KindBuiltinGo, KindLocalProgram, KindTrustedExecutable, KindNativeRuntime:
		return Kind(value), nil
	default:
		return "", &Error{Code: CodeNotFound, Detail: "implementation_kind vocabulary"}
	}
}

// needsDigest reports whether the kind carries an executable digest. Section
// 4.B: digest for local_program or trusted_executable; null otherwise.
func needsDigest(kind Kind) bool {
	return kind == KindLocalProgram || kind == KindTrustedExecutable
}

// Registration is one admitted backend identity with its exact version tuple,
// platform set, and trust binding. It is the registry half of the Section 4.B
// Manifest identity: the Manifest/Probe objects themselves belong to the
// sibling manifest task.
type Registration struct {
	// ID is the canonical terminal_backend_id.
	ID string
	// Kind is the closed implementation_kind.
	Kind Kind
	// ImplementationVersion is the exact admitted semver.
	ImplementationVersion string
	// ProtocolVersions are the sorted unique semver members in Terminal
	// Backend Protocol major 1.
	ProtocolVersions []string
	// Platforms is the sorted unique non-empty platform subset.
	Platforms []scalar.Platform
	// ExecutableDigest is the sha256: digest for external kinds and empty
	// (null) for built-in kinds.
	ExecutableDigest string
}

// validate enforces every closed Section 4.B/6.5 rule on one record.
func (record Registration) validate() error {
	if err := mustParseID(record.ID); err != nil {
		return err
	}
	if _, err := parseKind(string(record.Kind)); err != nil {
		return err
	}
	if !semverPattern.MatchString(record.ImplementationVersion) {
		return &Error{Code: CodeDrift, BackendID: record.ID, Detail: "implementation_version semver"}
	}
	if err := validateProtocolVersions(record.ID, record.ProtocolVersions); err != nil {
		return err
	}
	if err := validatePlatforms(record.Platforms); err != nil {
		return &Error{Code: CodeNotFound, BackendID: record.ID, Detail: err.Error()}
	}
	if needsDigest(record.Kind) {
		if _, err := scalar.ParseDigest(record.ExecutableDigest); err != nil {
			return &Error{Code: CodeUntrusted, BackendID: record.ID, Detail: "executable_digest"}
		}
	} else if record.ExecutableDigest != "" {
		return &Error{Code: CodeDrift, BackendID: record.ID, Detail: "executable_digest must be null"}
	}
	return nil
}

// validateProtocolVersions enforces sorted unique semver[1..32], each in
// Terminal Backend Protocol major 1.
func validateProtocolVersions(backendID string, versions []string) error {
	if len(versions) < 1 || len(versions) > maxProtocolVersions {
		return &Error{Code: CodeDrift, BackendID: backendID, Detail: "protocol_versions bound"}
	}
	for index, version := range versions {
		if !semverPattern.MatchString(version) || semverMajor(version) != 1 {
			return &Error{Code: CodeDrift, BackendID: backendID, Detail: "protocol_versions major 1"}
		}
		if index > 0 && versions[index-1] >= version {
			return &Error{Code: CodeDrift, BackendID: backendID, Detail: "protocol_versions sorted unique"}
		}
	}
	return nil
}

// semverMajor returns the major number of a validated semver string.
func semverMajor(version string) int {
	major := 0
	for i := 0; i < len(version); i++ {
		if version[i] == '.' {
			break
		}
		major = major*10 + int(version[i]-'0')
	}
	return major
}

// validatePlatforms enforces a sorted unique non-empty subset of the closed
// macos|linux|wsl2|windows vocabulary.
func validatePlatforms(platforms []scalar.Platform) error {
	if len(platforms) == 0 || len(platforms) > 4 {
		return errors.New("platforms bound")
	}
	previous := ""
	for _, platform := range platforms {
		if _, err := scalar.ParsePlatform(platform.String()); err != nil {
			return errors.New("platforms vocabulary")
		}
		if previous >= platform.String() {
			return errors.New("platforms sorted unique")
		}
		previous = platform.String()
	}
	return nil
}

// cloneStrings deep-copies a version list so the registry never shares a
// backing array with its caller. Registration carries slices; a struct copy
// duplicates only the header, so every ingress and egress point must clone.
func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

// clonePlatforms deep-copies a platform set for the same reason as
// cloneStrings: without it a caller holding a resolved record keeps a live
// handle on admitted registry state and can convert drift into a duplicate.
func clonePlatforms(values []scalar.Platform) []scalar.Platform {
	if values == nil {
		return nil
	}
	cloned := make([]scalar.Platform, len(values))
	copy(cloned, values)
	return cloned
}

// cloneRecord deep-copies every slice member of a record.
func cloneRecord(record Registration) Registration {
	record.ProtocolVersions = cloneStrings(record.ProtocolVersions)
	record.Platforms = clonePlatforms(record.Platforms)
	return record
}

// equalRecord reports whether two validated records are member-for-member
// identical. Any difference is drift, never a compatible update.
func equalRecord(left, right Registration) bool {
	if left.ID != right.ID || left.Kind != right.Kind ||
		left.ImplementationVersion != right.ImplementationVersion ||
		left.ExecutableDigest != right.ExecutableDigest {
		return false
	}
	if len(left.ProtocolVersions) != len(right.ProtocolVersions) {
		return false
	}
	for i := range left.ProtocolVersions {
		if left.ProtocolVersions[i] != right.ProtocolVersions[i] {
			return false
		}
	}
	if len(left.Platforms) != len(right.Platforms) {
		return false
	}
	for i := range left.Platforms {
		if left.Platforms[i] != right.Platforms[i] {
			return false
		}
	}
	return true
}

// TrustEntry is one Configuration 3.0.0 external_trust entry (§6.5): exactly a
// backend ID, an absolute executable path, an executable digest, and enabled.
type TrustEntry struct {
	BackendID        string
	ExecutablePath   string
	ExecutableDigest string
	Enabled          bool
}

// validateTrust enforces the Section 6.5 trust admission: grammar plus the
// reserved-namespace bar (third parties use a vendor-owned namespace),
// absolute path only (PATH-only discovery fails), and a well-formed digest.
// A disabled entry is not an error here; registration treats it as untrusted.
func (entry TrustEntry) validate(platform scalar.Platform) error {
	if err := mustParseID(entry.BackendID); err != nil {
		return err
	}
	if strings.HasPrefix(entry.BackendID, reservedNamespace) {
		return &Error{Code: CodeAmbiguous, BackendID: entry.BackendID, Detail: "external_trust reserved namespace"}
	}
	if _, err := scalar.ParseAbsolutePath(platform, entry.ExecutablePath); err != nil {
		return &Error{Code: CodeUntrusted, BackendID: entry.BackendID, Detail: "external_trust executable_path"}
	}
	if _, err := scalar.ParseDigest(entry.ExecutableDigest); err != nil {
		return &Error{Code: CodeUntrusted, BackendID: entry.BackendID, Detail: "external_trust executable_digest"}
	}
	return nil
}

// Registry is the process-local TerminalBackend registry. It opens with the
// two canonical built-ins admitted and accepts external adapters only through
// RegisterExternal with a validated trust entry. The zero value is unusable;
// construct with New.
type Registry struct {
	mutex   sync.RWMutex
	records map[string]Registration
}

// New admits the canonical built-ins ax.tmux (macos, linux, wsl2) and
// ax.conpty (windows) at the given implementation version with the given
// Terminal Backend Protocol versions. Both are builtin_go without an
// executable digest. Unknown members or a non-major-1 protocol list fail
// before either built-in is admitted.
func New(implementationVersion string, protocolVersions []string) (*Registry, error) {
	if !semverPattern.MatchString(implementationVersion) {
		return nil, &Error{Code: CodeDrift, Detail: "implementation_version semver"}
	}
	protocols := append([]string(nil), protocolVersions...)
	sort.Strings(protocols)
	if err := validateProtocolVersions(BuiltinTmux, protocols); err != nil {
		return nil, err
	}
	registry := &Registry{records: make(map[string]Registration, 4)}
	builtins := []Registration{
		{
			ID:                    BuiltinTmux,
			Kind:                  KindBuiltinGo,
			ImplementationVersion: implementationVersion,
			// Each built-in gets its own copy so the two admitted records
			// never alias each other. This is defence-in-depth, not a
			// bypass fix: callers only ever see egress clones (Resolve
			// clones, RegisterExternal clones on ingress, IDs returns
			// strings), so no exported path can reach these arrays.
			ProtocolVersions: cloneStrings(protocols),
			Platforms:        []scalar.Platform{scalar.PlatformLinux, scalar.PlatformMacOS, scalar.PlatformWSL2},
			// Platforms literal above is already sorted: linux < macos < wsl2.
		},
		{
			ID:                    BuiltinConpty,
			Kind:                  KindBuiltinGo,
			ImplementationVersion: implementationVersion,
			ProtocolVersions:      cloneStrings(protocols),
			Platforms:             []scalar.Platform{scalar.PlatformWindows},
		},
	}
	for _, record := range builtins {
		if err := record.validate(); err != nil {
			return nil, err
		}
		registry.records[record.ID] = record
	}
	return registry, nil
}

// RegisterExternal admits one external adapter (§4.B discovery before
// activation, §6.5 trust). The trust entry must be enabled and name an
// absolute executable whose digest exactly equals the observed record digest;
// a mismatch fails before any probe. The observed record must use an external
// kind, carry the same ID, and pass full record validation. Registering an
// already-known ID fails closed: an identical record is CodeAmbiguous
// (registration is not idempotent) and any difference is CodeDrift.
func (registry *Registry) RegisterExternal(platform scalar.Platform, entry TrustEntry, observed Registration) error {
	if registry == nil {
		return &Error{Code: CodeNotFound, Detail: "registry unavailable"}
	}
	if err := entry.validate(platform); err != nil {
		return err
	}
	if !entry.Enabled {
		return &Error{Code: CodeUntrusted, BackendID: entry.BackendID, Detail: "external_trust disabled"}
	}
	if observed.ID != entry.BackendID {
		return &Error{Code: CodeAmbiguous, BackendID: entry.BackendID, Detail: "external_trust identity binding"}
	}
	if observed.Kind != KindLocalProgram && observed.Kind != KindTrustedExecutable {
		return &Error{Code: CodeUntrusted, BackendID: entry.BackendID, Detail: "external implementation_kind"}
	}
	if err := observed.validate(); err != nil {
		return err
	}
	if observed.ExecutableDigest != entry.ExecutableDigest {
		return &Error{Code: CodeUntrusted, BackendID: entry.BackendID, Detail: "executable substitution"}
	}
	registry.mutex.Lock()
	defer registry.mutex.Unlock()
	existing, known := registry.records[observed.ID]
	if !known {
		// Store a copy: the caller keeps its own record, and retaining
		// the slices verbatim would leave the admitted record mutable
		// without passing any gate.
		registry.records[observed.ID] = cloneRecord(observed)
		return nil
	}
	if equalRecord(existing, observed) {
		return &Error{Code: CodeAmbiguous, BackendID: observed.ID, Detail: "duplicate backend_id"}
	}
	return &Error{Code: CodeDrift, BackendID: observed.ID, Detail: "implementation drift"}
}

// Resolve returns the admitted record for a canonical ID. An unknown,
// malformed, or never-admitted ID is CodeNotFound: a registry read failure is
// never absence, and callers must not fall back to a default.
func (registry *Registry) Resolve(backendID string) (Registration, error) {
	if registry == nil {
		return Registration{}, &Error{Code: CodeNotFound, Detail: "registry unavailable"}
	}
	id, err := ParseID(backendID)
	if err != nil {
		return Registration{}, err
	}
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	record, known := registry.records[id]
	if !known {
		return Registration{}, &Error{Code: CodeNotFound, BackendID: id, Detail: "unregistered terminal_backend_id"}
	}
	// Hand back a copy: the struct header copy shares the backing arrays,
	// so returning the stored record would hand out live interior state.
	return cloneRecord(record), nil
}

// IDs returns the sorted admitted identities. It reports membership only,
// never availability.
func (registry *Registry) IDs() []string {
	if registry == nil {
		return nil
	}
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()
	ids := make([]string, 0, len(registry.records))
	for id := range registry.records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// DefaultForPlatform returns the platform default for new activation (§6.5):
// ax.tmux on macos/linux/wsl2 and ax.conpty on windows. Defaults select only
// a new Terminal Instance; restore and resume must use RequireRestoreBinding.
func DefaultForPlatform(platform scalar.Platform) (string, error) {
	switch platform {
	case scalar.PlatformMacOS, scalar.PlatformLinux, scalar.PlatformWSL2:
		return BuiltinTmux, nil
	case scalar.PlatformWindows:
		return BuiltinConpty, nil
	default:
		return "", &Error{Code: CodeNotFound, Detail: "platform vocabulary"}
	}
}

// RequireRestoreBinding resolves the exact prior binding for restore or
// resume (§4.B, §6.5). The candidate must equal the bound backend ID; using
// the current configured default instead is CodeRestoreMismatch. The bound ID
// must be registered: unknown or unsupported IDs remain visible but cannot
// activate.
func (registry *Registry) RequireRestoreBinding(boundBackendID, candidateBackendID string) (Registration, error) {
	if registry == nil {
		return Registration{}, &Error{Code: CodeNotFound, Detail: "registry unavailable"}
	}
	bound, err := ParseID(boundBackendID)
	if err != nil {
		return Registration{}, err
	}
	candidate, err := ParseID(candidateBackendID)
	if err != nil {
		return Registration{}, err
	}
	if candidate != bound {
		return Registration{}, &Error{Code: CodeRestoreMismatch, BackendID: candidate, Detail: "restore requires the prior binding"}
	}
	return registry.Resolve(bound)
}

// CheckVersionTuple admits one (implementation_version, protocol_version)
// pair against the admitted protocol list: the implementation must be
// semver, the protocol must be semver in Terminal Backend Protocol major 1
// and exactly one member of the list. Array-to-scalar selection is
// membership, not aggregate equality (§4.B).
func CheckVersionTuple(backendID, implementationVersion, protocolVersion string, protocolVersions []string) error {
	if !semverPattern.MatchString(implementationVersion) {
		return &Error{Code: CodeDrift, BackendID: backendID, Detail: "implementation_version semver"}
	}
	if !semverPattern.MatchString(protocolVersion) || semverMajor(protocolVersion) != 1 {
		return &Error{Code: CodeDrift, BackendID: backendID, Detail: "protocol_version major 1"}
	}
	for _, member := range protocolVersions {
		if protocolVersion == member {
			return nil
		}
	}
	return &Error{Code: CodeDrift, BackendID: backendID, Detail: "protocol_version membership"}
}

// InstanceBinding is the validated host-local binding subset a §7.A provider
// descriptor must match: binding digest, backend ID, versions, and generation
// (SPEC.md §7.A). TerminalBindingID is the digest of the validated Terminal
// Instance Binding 1.0.0 carried as terminal_binding_id (SPEC.md:3461).
type InstanceBinding struct {
	BackendID             string
	ImplementationVersion string
	ProtocolVersion       string
	Generation            string
	TerminalBindingID     string
}

// CheckProviderDescriptor enforces the Section 7.A rule: the provider rejects
// a descriptor whose binding digest, backend ID, versions, or generation does
// not match the AX-validated host-local binding before launching or observing
// a provider process. Generation lengths 0 and over 256 UTF-8 characters are
// invalid; 256 is accepted.
func CheckProviderDescriptor(descriptor, binding InstanceBinding) error {
	if err := checkGeneration(descriptor.Generation); err != nil {
		return err
	}
	if err := checkGeneration(binding.Generation); err != nil {
		return err
	}
	if _, err := ParseID(descriptor.BackendID); err != nil {
		return err
	}
	if descriptor.BackendID != binding.BackendID {
		return &Error{Code: CodeNotFound, BackendID: descriptor.BackendID, Detail: "descriptor backend binding"}
	}
	if _, err := scalar.ParseDigest(descriptor.TerminalBindingID); err != nil {
		return &Error{Code: CodeNotFound, BackendID: descriptor.BackendID, Detail: "descriptor binding digest"}
	}
	if descriptor.TerminalBindingID != binding.TerminalBindingID {
		return &Error{Code: CodeNotFound, BackendID: descriptor.BackendID, Detail: "descriptor binding digest"}
	}
	if descriptor.ImplementationVersion != binding.ImplementationVersion ||
		descriptor.ProtocolVersion != binding.ProtocolVersion {
		return &Error{Code: CodeDrift, BackendID: descriptor.BackendID, Detail: "descriptor version binding"}
	}
	if descriptor.Generation != binding.Generation {
		return &Error{Code: CodeStaleGeneration, BackendID: descriptor.BackendID, Detail: "descriptor generation binding"}
	}
	return nil
}

// checkGeneration enforces string[1..256] measured in UTF-8 characters over
// valid UTF-8 (SPEC.md:321).
func checkGeneration(generation string) error {
	if !utf8.ValidString(generation) || utf8.RuneCountInString(generation) < 1 || utf8.RuneCountInString(generation) > maxGenerationRunes {
		return &Error{Code: CodeStaleGeneration, Detail: "backend_generation bound"}
	}
	return nil
}

// DigestFile returns the sha256: digest of the executable at path after
// resolving symlinks. The target must be a regular file; anything else (or a
// read failure) is an error, never a digest. It distinguishes a failed read
// from an absent trust entry: callers must not fall back to PATH discovery.
func DigestFile(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("terminal backend executable: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("terminal backend executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("terminal backend executable %s: not a regular file", resolved)
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("terminal backend executable: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
