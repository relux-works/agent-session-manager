package tmuxserver

import (
	"os"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// RuntimeDirName is the fixed leaf name of the AX tmux runtime
// directory under the Runtime IPC root: pinned SPEC 806-808 selects
// the dedicated server by `tmux -S <runtime>/tmux/ax.sock`, so the
// leaf is `tmux`. It is AX-controlled vocabulary, never an ambient
// or conventional tmux name.
const RuntimeDirName = "tmux"

// Hooks arms the crash boundaries of EnsureRuntimeDir. AfterMkdir
// runs after the mkdir attempt resolves — the leaf created or
// already present — and before verification and the parent fsync; a
// crash there must leave a retry that converges to the one verified
// directory. Hooks is nil in production.
type Hooks struct {
	AfterMkdir func(dir string)
}

// EnsureRuntimeDir creates the owner-only runtime directory
// root/name when absent and verifies it when present, returning the
// joined path. Creation and verification enforce three independent
// gates — containment, mode, ownership — and each refuses alone:
//
//   - containment: the root must be an absolute path on the platform
//     grammar, the name must be one relative segment, and the commit
//     opens the parent through a pinned directory handle and creates
//     the leaf relative to that handle, so a symlinked intermediate
//     directory cannot redirect the commit outside the root;
//   - mode: the leaf must carry exactly 0700, enforced explicitly
//     after creation because mkdir honors umask;
//   - ownership: the leaf must belong to the effective UID.
//
// Lexical validation delegates to its landed owners (absolute root
// and relative member to scalar, the join to secprim.Guard); the
// descriptor-relative commit below is what this package owns, because
// no landed helper commits a directory relative to a handle.
//
// On failure the returned path distinguishes the refusal phase: a
// lexical refusal returns "", while a commit-phase refusal returns
// the joined path alongside the error. Callers must check the error
// first and never use the path when it is set; the split lets the
// acquisition wiring narrow exactly the commit-phase class.
func EnsureRuntimeDir(root, name string, platform scalar.Platform, hooks *Hooks) (string, error) {
	joined, err := lexicalRuntimePath(root, name, platform)
	if err != nil {
		return "", err
	}
	return joined, commitRuntimeDir(root, name, joined, hooks)
}

// VerifyRuntimeDir enforces the three custody gates on an existing
// runtime directory without creating it. Background acquisition calls
// this instead of Ensure: the directory must already exist with exact
// custody, and a missing directory refuses rather than being silently
// created on the authorizing path. Foreground acquisition calls
// Ensure, which provisions the directory it then verifies.
func VerifyRuntimeDir(root, name string, platform scalar.Platform) error {
	if _, err := lexicalRuntimePath(root, name, platform); err != nil {
		return err
	}
	return verifyRuntimeDir(root, name)
}

// lexicalRuntimePath validates the root grammar, the name grammar,
// and the lexical join. It answers the lexical question only; the
// commit answers the filesystem question through a handle.
func lexicalRuntimePath(root, name string, platform scalar.Platform) (string, error) {
	if _, err := scalar.ParsePlatform(string(platform)); err != nil {
		return "", &Error{Code: CodeInvalidArguments, Detail: "runtime platform"}
	}
	// Native Windows MUST NOT claim tmux or tmux resurrection
	// (pinned SPEC 1458), so the Windows platform refuses here on
	// every host, before any host-specific commit runs. Weakening
	// this arm reroutes a Windows request to the root refusal
	// instead of admitting it.
	if platform == scalar.PlatformWindows {
		return "", &Error{Code: CodeInvalidArguments, Detail: "runtime platform"}
	}
	if _, err := scalar.ParseAbsolutePath(platform, root); err != nil {
		return "", &Error{Code: CodeInvalidArguments, Detail: "runtime root"}
	}
	if err := checkRuntimeName(name); err != nil {
		return "", err
	}
	guard, err := secprim.NewGuard(root, platform, []string{name})
	if err != nil {
		return "", &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime containment"}
	}
	joined, err := guard.Resolve(name)
	if err != nil {
		return "", &Error{Code: CodeUnsafeRuntimeDir, Detail: "runtime containment"}
	}
	return joined, nil
}

// checkRuntimeName admits exactly one relative path segment. The
// shared grammar (absolute, parent, dot, empty, NUL, backslash,
// encoded separators) is decided by scalar; this gate adds only the
// single-segment rule scalar does not own, because the commit opens
// the leaf relative to the pinned parent handle and a second segment
// would name a path the handle walk never descends.
func checkRuntimeName(name string) error {
	if _, err := scalar.ParseRelativePath(name); err != nil {
		return &Error{Code: CodeInvalidArguments, Detail: "runtime name"}
	}
	if strings.Contains(name, "/") {
		return &Error{Code: CodeInvalidArguments, Detail: "runtime name"}
	}
	return nil
}

// sameDirectory compares two stat results for directory identity. It
// is a variable only so seam-failure tests can force the
// handle-mismatch path through the production entries: a handle that
// names a different directory than the path cannot be staged without
// a filesystem race, and without the seam the mismatch branch is
// untestable. Production code never reassigns it.
var sameDirectory = os.SameFile

// handleBoundToPath reports whether the open directory handle refers
// to the same directory as rootPath. It mirrors the private
// secprim.guardRootMatches check, which no caller outside secprim
// can invoke: a stat failure on either side is not a match, and an
// unreadable root fails closed into the mismatch refusal rather
// than admitting an unverifiable handle.
func handleBoundToPath(handle *os.File, rootPath string) bool {
	handleInfo, err := handle.Stat()
	if err != nil || handleInfo == nil {
		return false
	}
	pathInfo, err := os.Stat(rootPath)
	if err != nil || pathInfo == nil {
		return false
	}
	return sameDirectory(handleInfo, pathInfo)
}
