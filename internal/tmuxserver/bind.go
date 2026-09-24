package tmuxserver

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// Unix socket path limits: sun_path byte size including the trailing
// NUL, per platform. macOS (darwin) allows 104 bytes; Linux and WSL2
// allow 108. A derived socket at or beyond the limit can never bind or
// connect, so the adapter refuses it before exec rather than surfacing
// a spawn failure.
const (
	sunPathLimitDarwin = 104
	sunPathLimitLinux  = 108
)

// CheckSocketLength refuses a derived socket path that cannot fit the
// platform sun_path (bound B18, decided here): the byte length must
// stay strictly below the platform limit, leaving room for the NUL.
// The check runs on every bind and every connect, before any custody
// or exec work, so a deep runtime root reports the length refusal, not
// a spawn failure. Native Windows refuses upstream: no tmux socket
// exists there.
func CheckSocketLength(socket string, platform scalar.Platform) error {
	limit := sunPathLimitLinux
	if platform == scalar.PlatformMacOS {
		limit = sunPathLimitDarwin
	}
	if len(socket) >= limit {
		return &Error{Code: CodeSocketPathTooLong, Detail: "socket path length"}
	}
	return nil
}

// CheckSocketCustody enforces the SPEC 810-812 before-bind/connect
// rejection (bounds B19/B20, decided here): before bind, connect,
// rename, or unlink, AX rejects a socket path, parent, or ancestor
// whose file kind, ownership, or permissions are unsafe. It answers
// for one derived socket under one Runtime IPC root:
//
//   - the socket must name the runtime leaf exactly (<root>/tmux/ax.sock):
//     a socket elsewhere is a substitution and refuses;
//   - the runtime leaf must pass the landed leaf custody (mode,
//     ownership, kind, containment via VerifyRuntimeDir);
//   - the root itself must be an euid-owned, mode-0700 directory,
//     never a symlink;
//   - every ancestor above the root up to / must be a directory,
//     never a symlink, never writable by group or other unless sticky
//     (read/execute bits are allowed for system prefixes such as 0755
//     /var; ownership is not required above the root);
//   - the socket path itself, when present, must be an euid-owned
//     socket with owner-only permissions: owner read/write bits are
//     required and no group/other bits are allowed. tmux may toggle the
//     owner's execute bit while attached sessions exist. Its identity is checked
//     again after those attributes so a path swap during validation
//     refuses before the caller binds or connects. Absence is bindable.
//
// The root and every ancestor open through the landed no-follow
// owner and verify through the open descriptor; a symlink at any
// level refuses, and the root handle is bound back to its path. A
// residual TOCTOU after the final socket lstat and before the
// operating-system bind/connect call remains a stated bound: Unix
// connect has no portable path-relative identity pin. The final
// lstat detects an identity change during custody validation; a later
// substitution is still constrained by generation-bound attestation.
func CheckSocketCustody(socket, root string, platform scalar.Platform) error {
	if socket != filepath.Join(root, RuntimeDirName, SocketName) {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket placement"}
	}
	if err := VerifyRuntimeDir(root, RuntimeDirName, platform); err != nil {
		return err
	}
	if err := checkCustodyRoot(root); err != nil {
		return err
	}
	if err := checkCustodyAncestors(root); err != nil {
		return err
	}
	return checkCustodySocket(socket)
}

// checkCustodyRoot gates the Runtime IPC root itself: an euid-owned
// mode-0700 directory, never a symlink. Unlike system ancestors, the
// AX-owned Runtime IPC root may not expose any permission bit to group
// or other users. The directory is opened through the landed no-follow owner and verified through the
// open descriptor, then bound to the path: the descriptor and a fresh
// lstat must name the same directory, so a swap between open and use
// refuses instead of verifying one directory and using another.
func checkCustodyRoot(root string) error {
	dir, err := secprim.OpenNoFollowDir(root)
	if err != nil {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket root"}
	}
	defer func() { _ = dir.Close() }()
	info, err := dir.Stat()
	if err != nil {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket root"}
	}
	mode := custodyModeForPath(root, info.Mode())
	if !mode.IsDir() {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket root"}
	}
	if !custodyOwnershipMatches(root, info) {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket root"}
	}
	if !ownerOnlyCustodyRootMode(mode) {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket root"}
	}
	pathed, err := os.Lstat(root)
	if err != nil || !sameCustodyIdentity(info, pathed) {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket root"}
	}
	return nil
}

// ownerOnlyCustodyRootMode implements the SPEC §3.2 mode-0700 custody
// rule for the runtime root. Special permission bits are not part of
// mode 0700 and therefore refuse as well.
func ownerOnlyCustodyRootMode(mode os.FileMode) bool {
	permissions := mode.Perm()
	return permissions&0o077 == 0 && permissions&0o700 == 0o700 &&
		mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0
}

// custodyModeProjection is a deterministic test seam for exhaustively
// exercising Unix permission modes through the production entries. The
// projection changes only the mode value observed by custody predicates;
// production leaves it nil and always uses the stat result.
var custodyModeProjection func(path string, mode os.FileMode) os.FileMode

// custodyKindProjection and custodyOwnerProjection let package tests
// project metadata for one path component that cannot safely be changed on
// the host filesystem (notably ancestors and foreign owners). Production
// leaves both nil and evaluates the opened descriptor's metadata.
var custodyKindProjection func(path string, mode os.FileMode) os.FileMode
var custodyOwnerProjection func(path string, info os.FileInfo) os.FileInfo

func custodyModeForPath(path string, mode os.FileMode) os.FileMode {
	if custodyKindProjection != nil {
		mode = custodyKindProjection(path, mode)
	}
	if custodyModeProjection != nil {
		return custodyModeProjection(path, mode)
	}
	return mode
}

func custodyOwnershipMatches(path string, info os.FileInfo) bool {
	if custodyOwnerProjection != nil {
		info = custodyOwnerProjection(path, info)
	}
	return ownershipMatches(info)
}

// sameCustodyIdentity reports whether two stat results name the same
// filesystem object. It is a variable only so seam-failure tests can
// force the identity-mismatch path through the production entry: a
// swap between open and stat cannot be staged deterministically, and
// without the seam the mismatch branch is untestable. Production code
// never reassigns it.
var sameCustodyIdentity = os.SameFile

// checkCustodyAncestors walks every ancestor above the root up to the
// filesystem root: each must be a directory, never a symlink, and
// never writable by group or other unless sticky. Ownership is not
// decided above the root: system prefixes are root-owned by design.
//
// The walk climbs the LEXICAL path and opens every ancestor through
// the landed no-follow owner: a symlink at any level — including an
// intermediate alias above an otherwise valid root — refuses, because
// writes through an alias land outside the verified tree. The walk
// never resolves: resolving first would verify one path and use
// another. Callers pass a canonical root (the runtime-root owner
// canonicalizes at creation); system-conventional symlinks such as
// macOS /tmp are resolved by the caller, never excused here.
func checkCustodyAncestors(root string) error {
	cleaned := filepath.Clean(root)
	for {
		parent := filepath.Dir(cleaned)
		if parent == cleaned {
			return nil
		}
		dir, err := secprim.OpenNoFollowDir(parent)
		if err != nil {
			return &Error{Code: CodeUnsafeSocketPath, Detail: "socket ancestor"}
		}
		info, statErr := dir.Stat()
		_ = dir.Close()
		if statErr != nil {
			return &Error{Code: CodeUnsafeSocketPath, Detail: "socket ancestor"}
		}
		mode := custodyModeForPath(parent, info.Mode())
		if !mode.IsDir() || writableByOthers(mode) {
			return &Error{Code: CodeUnsafeSocketPath, Detail: "socket ancestor"}
		}
		if isFilesystemRoot(parent) {
			return nil
		}
		cleaned = parent
	}
}

// isFilesystemRoot reports the end of the ancestor walk: the cleaned
// path is the volume root (/ on unix; a drive root on Windows, which
// refuses upstream before this walk ever runs).
func isFilesystemRoot(path string) bool {
	return path == string(filepath.Separator) || strings.HasSuffix(path, ":\\")
}

// writableByOthers reports group/world writability without the sticky
// bit: system ancestors may expose read/execute bits for path traversal,
// and sticky directories (01777 /tmp) admit shared writes by design,
// while a plain 0777 or 0770 admits an unsafe rename.
func writableByOthers(mode os.FileMode) bool {
	perm := mode.Perm()
	if perm&0o022 == 0 {
		return false
	}
	return mode&os.ModeSticky == 0
}

// lstatCustodySocket is a test seam for a socket path replacement
// between the attribute check and the identity recheck. Production
// always uses os.Lstat; no path resolution or symlink following is
// permitted.
var lstatCustodySocket = os.Lstat

// checkCustodySocket gates the socket path itself: absence is bindable;
// a present path must be a current-user-owned socket with owner
// read/write access and no group/other permissions. tmux may add the
// owner's execute bit while a session is attached, which does not widen
// access to another user; that mode remains safe. The socket identity
// must remain stable across validation. This refuses
// a symlink, foreign socket, permissive socket, or a replacement staged
// while custody is being checked before any bind/connect call.
func checkCustodySocket(socket string) error {
	info, err := lstatCustodySocket(socket)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket kind"}
	}
	mode := custodyModeForPath(socket, info.Mode())
	if mode.Type() != os.ModeSocket {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket kind"}
	}
	if !custodyOwnershipMatches(socket, info) {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket ownership"}
	}
	if !ownerOnlySocketMode(mode) {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket permissions"}
	}
	pathed, err := lstatCustodySocket(socket)
	if err != nil || !sameCustodyIdentity(info, pathed) {
		return &Error{Code: CodeUnsafeSocketPath, Detail: "socket identity"}
	}
	return nil
}

// ownerOnlySocketMode permits tmux's owner execute-bit toggle while
// attached, but refuses every group/other and special permission bit.
// Owner read and write remain required for the Unix socket.
func ownerOnlySocketMode(mode os.FileMode) bool {
	permissions := mode.Perm()
	return permissions&0o077 == 0 && permissions&0o600 == 0o600 &&
		mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0
}
