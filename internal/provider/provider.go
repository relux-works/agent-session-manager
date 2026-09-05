package provider

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Stable Structured Error codes this package reports. Discovery-shape
// refusals (duplicates, malformed names, non-regular targets, unapproved
// owners, relative directories) report invalid_config; a filesystem read
// that cannot establish the discovery or verification preconditions reports
// local_precondition_failed; a trust receipt that no longer matches freshly
// read facts reports integrity_failure. The duplicate mapping is mandated by
// Section 7.1; the other two are derived mappings documented on the
// constructors, because Section 7.1 names no code for those cases.
const (
	codeInvalidConfig     = "invalid_config"
	codeLocalPrecondition = "local_precondition_failed"
	codeIntegrityFailure  = "integrity_failure"
)

// Error is the only error this package produces. Code carries the stable
// registry code above; Detail names the offending provider ID, path, or
// trust dimension in human text automation must never branch on; Cause
// preserves the underlying filesystem failure for errors.Is and errors.As.
type Error struct {
	code   string
	detail string
	cause  error
}

// Code reports the stable registry code for this refusal.
func (err Error) Code() string { return err.code }

// Detail reports the human-text detail for this refusal.
func (err Error) Detail() string { return err.detail }

// Unwrap exposes the underlying filesystem cause, if one was recorded.
func (err Error) Unwrap() error { return err.cause }

func (err Error) Error() string {
	if err.cause == nil {
		return "provider " + err.code + ": " + err.detail
	}
	return "provider " + err.code + ": " + err.detail + ": " + err.cause.Error()
}

// The four refusal constructors below are declared as variables so the
// refusal-inventory gate can observe every exercised refusal site. Each is
// the single construction site for its failure class; production code must
// not build Error values or wrap filesystem errors any other way.
//
// failDuplicate reports the Section 7.1 duplicate-ID refusal with the
// section-mandated invalid_config code.
//
// failInvalid reports a discovery-shape refusal: a malformed plugin name, a
// non-regular target, an unapproved owner, a relative directory, or a Trust
// call on a builtin adapter. These are operator-configuration or caller
// errors, so they share the invalid_config code.
//
// failPrecondition reports a filesystem read failure during discovery: an
// unreadable plugin directory, an unresolvable symlink, an unreadable
// target, or unattainable ownership metadata. A failed read is never a
// legitimate absence, so discovery aborts with local_precondition_failed
// instead of yielding a partial candidate set.
//
// failIntegrity reports a trust receipt that no longer matches freshly read
// facts, including a receipt whose facts cannot be re-read at all: a
// changed path target or digest requires renewed trust, and an unreadable
// file cannot prove it is unchanged.
var failDuplicate = func(providerID, firstSource, secondSource string) Error {
	return Error{
		code:   codeInvalidConfig,
		detail: fmt.Sprintf("duplicate provider %q from %s and %s", providerID, firstSource, secondSource),
	}
}

var failInvalid = func(detail string) Error {
	return Error{code: codeInvalidConfig, detail: detail}
}

var failPrecondition = func(detail string, cause error) Error {
	return Error{code: codeLocalPrecondition, detail: detail, cause: cause}
}

var failIntegrity = func(detail string, cause error) Error {
	return Error{code: codeIntegrityFailure, detail: detail, cause: cause}
}

// Kind distinguishes a built-in adapter from an external executable.
type Kind string

const (
	// KindBuiltin is an adapter compiled into the host. It has no
	// executable path, digest, or owner to trust.
	KindBuiltin Kind = "builtin"
	// KindExternal is an ax-provider-<id> executable discovered on the
	// filesystem and subject to trust recording and verification.
	KindExternal Kind = "external"
)

// ExecutablePrefix is the Section 7.1 external executable name prefix. The
// suffix is a provider ID in the scalar provider-id grammar.
const ExecutablePrefix = "ax-provider-"

// builtinOrder is the Section 7.1 built-in set in the section's listed
// order. The order is a discovery enumeration order only; Section 7.1
// states the order does not establish precedence.
var builtinOrder = []string{"codex", "claude", "gemini", "muse", "antigravity", "pi"}

// Builtins returns the Section 7.1 built-in provider IDs in discovery
// order. The result is a copy; the registry cannot be mutated through it.
func Builtins() []string {
	return append([]string(nil), builtinOrder...)
}

// Config carries the operator inputs Discover consumes. PluginDirs maps to
// providers.plugin_dirs in listed order and AllowPathPlugins to
// providers.allow_path_plugins; both arrive already parsed by
// internal/config, which additionally enforces absolute directories and
// require_explicit_trust. Platform is the host AX platform the paths are
// evaluated on.
type Config struct {
	Platform         scalar.Platform
	PluginDirs       []string
	AllowPathPlugins bool
}

// OwnerPolicy decides whether a filesystem UID may own a trusted external
// executable. OperatorUID is the operator's own identity;
// AdministratorUIDs are the additional administrator-approved identities,
// if the host configures any. An empty administrator set trusts the
// operator alone; no superuser exception is implied.
type OwnerPolicy struct {
	OperatorUID       uint32
	AdministratorUIDs []uint32
}

// Approves reports whether uid may own a trusted executable.
func (policy OwnerPolicy) Approves(uid uint32) bool {
	if uid == policy.OperatorUID {
		return true
	}
	for _, admin := range policy.AdministratorUIDs {
		if uid == admin {
			return true
		}
	}
	return false
}

// OwnerIdentity renders a UID in the stable form TrustRecord stores.
func (policy OwnerPolicy) OwnerIdentity(uid uint32) string {
	return fmt.Sprintf("uid:%d", uid)
}

// FileInfo is the ownership and shape fact System.Inspect reports for one
// path. UID is the owning identity of the symlink target.
type FileInfo struct {
	IsRegular bool
	UID       uint32
}

// System is the filesystem seam Discover and Verify read through.
// Production code supplies OSSystem; tests supply a scripted fake. Every
// method reports a failed or partial read as an error, never as an empty
// result the caller could mistake for absence.
type System interface {
	// ReadDir returns the entry names of dir, unsorted.
	ReadDir(dir string) ([]string, error)
	// Canonicalize resolves dir against an absolute base and every
	// symlink in the path, returning the canonical absolute target.
	Canonicalize(path string) (string, error)
	// Inspect stats the symlink target at an already-canonical path.
	Inspect(canonical string) (FileInfo, error)
	// ReadFile returns the full bytes of the symlink target at an
	// already-canonical path.
	ReadFile(canonical string) ([]byte, error)
	// PathDirs returns the PATH search directories in listed order.
	PathDirs() []string
}

// Candidate is one discovered provider. External candidates carry the
// trust-time facts Section 7.1 requires: the canonical absolute executable
// path, the SHA-256 digest of its bytes, and the approving owner identity.
// Builtin candidates carry none of those; the zero accessors report
// absence. A Candidate carries no availability, status, or capability
// claim: discovery never advertises what the probe plane has not proven.
type Candidate struct {
	id     string
	kind   Kind
	source string
	path   string
	canon  string
	digest scalar.Digest
	owner  string
}

// ID reports the provider ID this candidate declares.
func (candidate Candidate) ID() string { return candidate.id }

// Kind reports whether this candidate is built-in or external.
func (candidate Candidate) Kind() Kind { return candidate.kind }

// Source names where the candidate was found: plugin_dirs[i], builtin, or
// path. It is a diagnostic label, not a precedence rank.
func (candidate Candidate) Source() string { return candidate.source }

// SourcePath reports the undisguised discovery path of an external
// candidate: the plugin or PATH directory joined with the executable name.
// Verify re-resolves this path to detect symlink retargeting.
func (candidate Candidate) SourcePath() (string, bool) {
	if candidate.kind != KindExternal {
		return "", false
	}
	return candidate.path, true
}

// CanonicalPath reports the symlink-resolved absolute target recorded at
// trust time for an external candidate.
func (candidate Candidate) CanonicalPath() (string, bool) {
	if candidate.kind != KindExternal {
		return "", false
	}
	return candidate.canon, true
}

// Digest reports the SHA-256 digest of the target bytes recorded at trust
// time for an external candidate.
func (candidate Candidate) Digest() (scalar.Digest, bool) {
	if candidate.kind != KindExternal {
		return scalar.Digest{}, false
	}
	return candidate.digest, true
}

// Owner reports the approving owner identity recorded at trust time for an
// external candidate.
func (candidate Candidate) Owner() (string, bool) {
	if candidate.kind != KindExternal {
		return "", false
	}
	return candidate.owner, true
}

// TrustRecord is the trust-time receipt for one accepted external
// candidate: the canonical absolute executable path and SHA-256 digest
// Section 7.1 requires, plus the undisguised discovery path Verify
// re-resolves, the approving owner identity, and the trust instant. The
// host persists this receipt; Verify rechecks it.
type TrustRecord struct {
	providerID string
	sourcePath string
	canon      string
	digest     scalar.Digest
	owner      string
	trustedAt  scalar.Timestamp
}

// ProviderID reports the trusted provider ID.
func (record TrustRecord) ProviderID() string { return record.providerID }

// SourcePath reports the undisguised discovery path Verify re-resolves.
func (record TrustRecord) SourcePath() string { return record.sourcePath }

// CanonicalPath reports the recorded canonical absolute target.
func (record TrustRecord) CanonicalPath() string { return record.canon }

// Digest reports the recorded SHA-256 digest of the target bytes.
func (record TrustRecord) Digest() scalar.Digest { return record.digest }

// Owner reports the recorded approving owner identity.
func (record TrustRecord) Owner() string { return record.owner }

// TrustedAt reports the trust instant.
func (record TrustRecord) TrustedAt() scalar.Timestamp { return record.trustedAt }

// externalID splits an ax-provider-<id> file name. It reports false for
// names outside the prefix so ordinary directory entries are skipped; a
// prefixed name whose suffix fails the provider-id grammar is a malformed
// candidate, reported as an error rather than silently skipped.
func externalID(name string) (string, bool, error) {
	suffix, found := strings.CutPrefix(name, ExecutablePrefix)
	if !found {
		return "", false, nil
	}
	if _, err := scalar.ParseProviderID(suffix); err != nil {
		return "", false, err
	}
	return suffix, true, nil
}

// Discover deterministically enumerates provider candidates in the Section
// 7.1 source order: configured plugin directories in listed order (entry
// names sorted bytewise within each directory, because filesystem order is
// not deterministic), then built-in adapters in registry order, then PATH
// directories in listed order only when AllowPathPlugins is true. Both
// configured plugin directories and PATH entries must be absolute paths:
// a relative entry fails with invalid_config instead of resolving against
// the process working directory.
//
// Discover never probes or executes a candidate, so the duplicate refusal
// below happens before either by construction: the function has no probe
// or execution dependency to invoke. If two candidates declare the same
// provider ID, Discover fails with invalid_config naming both sources; the
// operator must remove or rename one candidate. There is no
// duplicate-selection override, including for byte-identical observations
// of one file through two directories.
//
// Every accepted external candidate records its canonical absolute path,
// SHA-256 digest, and approving owner at trust time: symlinks are resolved
// before comparison, the target must be a regular file, and its owner must
// satisfy policy. A matching directory entry that cannot become an
// accepted candidate (malformed name, unresolvable link, non-regular
// target, unapproved owner) fails discovery instead of being skipped,
// because a skipped trust check would silently narrow the trust boundary.
// Filesystem read failures abort discovery instead of yielding a partial
// set. Discover holds no state and starts no process.
func Discover(cfg Config, owner OwnerPolicy, system System) ([]Candidate, error) {
	seen := make(map[string]string)
	var out []Candidate
	add := func(candidate Candidate) error {
		if first, duplicate := seen[candidate.id]; duplicate {
			return failDuplicate(candidate.id, first, candidate.source)
		}
		seen[candidate.id] = candidate.source
		out = append(out, candidate)
		return nil
	}
	for index, dir := range cfg.PluginDirs {
		if _, err := scalar.ParseAbsolutePath(cfg.Platform, dir); err != nil {
			return nil, failInvalid(fmt.Sprintf("providers.plugin_dirs[%d] %q is not an absolute path", index, dir))
		}
		source := fmt.Sprintf("plugin_dirs[%d]", index)
		if err := collectDirectory(owner, system, dir, source, add); err != nil {
			return nil, err
		}
	}
	for _, id := range builtinOrder {
		if err := add(Candidate{id: id, kind: KindBuiltin, source: "builtin"}); err != nil {
			return nil, err
		}
	}
	if cfg.AllowPathPlugins {
		for index, dir := range system.PathDirs() {
			if _, err := scalar.ParseAbsolutePath(cfg.Platform, dir); err != nil {
				return nil, failInvalid(fmt.Sprintf("PATH[%d] %q is not an absolute path", index, dir))
			}
			if err := collectDirectory(owner, system, dir, "path", add); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// collectDirectory folds every ax-provider-<id> entry of one directory into
// add in bytewise name order. PATH entries reuse this path, so both sources
// enforce the same trust checks; only the diagnostic source label differs.
func collectDirectory(owner OwnerPolicy, system System, dir, source string, add func(Candidate) error) error {
	names, err := system.ReadDir(dir)
	if err != nil {
		return failPrecondition(fmt.Sprintf("cannot list %s directory %q", source, dir), err)
	}
	sort.Strings(names)
	for _, name := range names {
		id, candidate, err := externalID(name)
		if err != nil {
			return failInvalid(fmt.Sprintf("%s entry %q is not a provider executable name: %v", source, filepath.Join(dir, name), err))
		}
		if !candidate {
			continue
		}
		found, err := trustCandidate(owner, system, id, filepath.Join(dir, name), source)
		if err != nil {
			return err
		}
		if err := add(found); err != nil {
			return err
		}
	}
	return nil
}

// trustCandidate establishes the trust-time facts for one external
// executable: symlink resolution before comparison, regular-file target,
// approved owner, and digest over the target bytes.
func trustCandidate(owner OwnerPolicy, system System, id, path, source string) (Candidate, error) {
	canon, err := system.Canonicalize(path)
	if err != nil {
		return Candidate{}, failPrecondition(fmt.Sprintf("cannot resolve provider %q at %q", id, path), err)
	}
	info, err := system.Inspect(canon)
	if err != nil {
		return Candidate{}, failPrecondition(fmt.Sprintf("cannot inspect provider %q target %q", id, canon), err)
	}
	if !info.IsRegular {
		return Candidate{}, failInvalid(fmt.Sprintf("provider %q target %q is not a regular file", id, canon))
	}
	if !owner.Approves(info.UID) {
		return Candidate{}, failInvalid(fmt.Sprintf("provider %q target %q is owned by %s, which the trust policy does not approve", id, canon, owner.OwnerIdentity(info.UID)))
	}
	content, err := system.ReadFile(canon)
	if err != nil {
		return Candidate{}, failPrecondition(fmt.Sprintf("cannot digest provider %q target %q", id, canon), err)
	}
	digest := scalar.SHA256Digest(content)
	return Candidate{
		id:     id,
		kind:   KindExternal,
		source: source,
		path:   path,
		canon:  canon,
		digest: digest,
		owner:  owner.OwnerIdentity(info.UID),
	}, nil
}

// Trust records the trust receipt for one accepted external candidate.
// Trusting a builtin adapter is refused: it has no executable bytes to
// record. Trust is pure: identical inputs return identical receipts.
func Trust(candidate Candidate, trustedAt scalar.Timestamp) (TrustRecord, error) {
	if candidate.kind != KindExternal {
		return TrustRecord{}, failInvalid(fmt.Sprintf("provider %q is a builtin adapter with no executable to trust", candidate.id))
	}
	return TrustRecord{
		providerID: candidate.id,
		sourcePath: candidate.path,
		canon:      candidate.canon,
		digest:     candidate.digest,
		owner:      candidate.owner,
		trustedAt:  trustedAt,
	}, nil
}

// Verify rechecks a trust receipt against freshly read filesystem facts. A
// changed path target or digest requires renewed trust: any mismatch in
// the re-resolved canonical path, the target shape, the approving owner,
// or the digest fails with integrity_failure, as does any read failure,
// because an unreadable file cannot prove it is unchanged. Verify holds no
// state and starts no process; repeated verification over an unchanged
// tree succeeds identically.
func Verify(record TrustRecord, owner OwnerPolicy, system System) error {
	canon, err := system.Canonicalize(record.sourcePath)
	if err != nil {
		return failIntegrity(fmt.Sprintf("provider %q trust receipt cannot be re-resolved and requires renewed trust", record.providerID), err)
	}
	if canon != record.canon {
		return failIntegrity(fmt.Sprintf("provider %q executable target changed from %q to %q and requires renewed trust", record.providerID, record.canon, canon), nil)
	}
	info, err := system.Inspect(canon)
	if err != nil {
		return failIntegrity(fmt.Sprintf("provider %q executable target %q cannot be re-inspected and requires renewed trust", record.providerID, canon), err)
	}
	if !info.IsRegular {
		return failIntegrity(fmt.Sprintf("provider %q executable target %q is no longer a regular file and requires renewed trust", record.providerID, canon), nil)
	}
	if identity := owner.OwnerIdentity(info.UID); !owner.Approves(info.UID) || identity != record.owner {
		return failIntegrity(fmt.Sprintf("provider %q executable owner is now %s and requires renewed trust", record.providerID, identity), nil)
	}
	content, err := system.ReadFile(canon)
	if err != nil {
		return failIntegrity(fmt.Sprintf("provider %q executable target %q cannot be re-read and requires renewed trust", record.providerID, canon), err)
	}
	sum := sha256.Sum256(content)
	if subtle.ConstantTimeCompare(sum[:], digestBytes(record.digest)) != 1 {
		return failIntegrity(fmt.Sprintf("provider %q executable digest changed and requires renewed trust", record.providerID), nil)
	}
	return nil
}

// digestBytes decodes the recorded digest for constant-time comparison. A
// TrustRecord always carries a digest produced by SHA256Digest, so the
// decode cannot fail; a receipt that does not decode compares unequal
// rather than panicking.
func digestBytes(digest scalar.Digest) []byte {
	raw := digest.Hex()
	if len(raw) != sha256.Size*2 {
		return nil
	}
	decoded := make([]byte, sha256.Size)
	for i := range decoded {
		hi := unhex(raw[2*i])
		lo := unhex(raw[2*i+1])
		if hi > 15 || lo > 15 {
			return nil
		}
		decoded[i] = hi<<4 | lo
	}
	return decoded
}

func unhex(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	default:
		return 255
	}
}
