package secprim

import (
	"os"
	"path"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// CheckMemberPath admits one staging or archive member name on the given
// platform under Section 16.3. The member is a slash-separated relative
// path: the shared relative grammar (absolute, drive-qualified, backslash,
// NUL, empty, dot, parent segments, and repeatedly encoded separators) is
// decided by scalar.ParseRelativePath, which owns it. This gate adds only
// the platform rules scalar does not own because they apply to member
// names rather than to absolute paths:
//
//   - encoded dots and overlong separators (%2e, %c0%af, any case and %25
//     nesting): an extension past scalar, because extraction stacks
//     commonly percent-decode after validation and an admitted "%2e%2e"
//     becomes a parent segment below this gate;
//   - Windows alternate-stream and NTFS syntax: any colon outside a refused
//     drive prefix (Section 16.3 "alternate streams");
//   - Windows trailing dots and spaces per segment, which the filesystem
//     strips and which therefore alias distinct members;
//   - Windows reserved device names, decided by the same table scalar uses
//     for absolute paths.
func CheckMemberPath(platform scalar.Platform, member string) error {
	if _, err := scalar.ParseRelativePath(member); err != nil {
		return failPath("member grammar", memberErrorTarget(member))
	}
	if containsEncodedDot(member) {
		return failPath("member encoded dot", memberErrorTarget(member))
	}
	if containsOverlongSeparator(member) {
		return failPath("member overlong separator", memberErrorTarget(member))
	}
	if platform == scalar.PlatformWindows {
		for _, segment := range strings.Split(member, "/") {
			if strings.Contains(segment, ":") {
				return failPath("member alternate stream", memberErrorTarget(member))
			}
			if strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
				return failPath("member trailing dot or space", memberErrorTarget(member))
			}
			if scalar.IsReservedWindowsDeviceName(segment) {
				return failPath("member reserved device name", memberErrorTarget(member))
			}
		}
	}
	return nil
}

// unwrapPercentEncoding folds %25 layers the way scalar's separator check
// does, so a twice-encoded dot or separator cannot hide behind its outer
// encoding.
func unwrapPercentEncoding(value string) string {
	candidate := strings.ToLower(value)
	for {
		next := strings.ReplaceAll(candidate, "%25", "%")
		if next == candidate {
			return candidate
		}
		candidate = next
	}
}

// containsEncodedDot reports a percent-encoded dot (%2e, any case, any
// %25 nesting). scalar owns the encoded-separator rule (%2f/%5c) but not
// encoded dots; this gate extends it because extraction stacks commonly
// percent-decode after validation, turning an admitted "%2e%2e" into a
// parent segment below this gate. A literal percent followed by other
// hex (for example "100%.txt" or "a%20b") keeps passing through.
func containsEncodedDot(member string) bool {
	return strings.Contains(unwrapPercentEncoding(member), "%2e")
}

// containsOverlongSeparator reports the overlong UTF-8 encoding of slash
// (%c0%af, any case, any %25 nesting), which some decoders accept as a
// separator. Same ownership rationale as containsEncodedDot.
func containsOverlongSeparator(member string) bool {
	return strings.Contains(unwrapPercentEncoding(member), "%c0%af")
}

// memberErrorTarget names the refused member without echoing unbounded
// foreign bytes: the first 64 characters identify the site for an
// operator while a hostile multi-megabyte member cannot flood a log.
func memberErrorTarget(member string) string {
	if len(member) > 64 {
		return member[:64] + "..."
	}
	return member
}

// Guard is a containment boundary for staging and extraction: a validated
// absolute root on one platform plus the optional exact set of managed
// member names that may be replaced. A nil managed set disables the
// managed check; a non-nil set (even an empty one) restricts Resolve to
// its members, so denying by default is representable and distinct from
// not checking. The zero value is unusable; build one with NewGuard.
type Guard struct {
	root         string
	platform     scalar.Platform
	managed      map[string]bool
	checkManaged bool
}

// NewGuard validates the staging root and records the managed set.
// The root must be an absolute path on the platform: a relative root
// fails here instead of resolving against the process working directory
// at commit time.
func NewGuard(root string, platform scalar.Platform, managed []string) (Guard, error) {
	if _, err := scalar.ParseAbsolutePath(platform, root); err != nil {
		return Guard{}, failPath("guard root", memberErrorTarget(root))
	}
	guard := Guard{root: root, platform: platform}
	if managed != nil {
		guard.checkManaged = true
		guard.managed = make(map[string]bool, len(managed))
		for _, member := range managed {
			guard.managed[member] = true
		}
	}
	return guard, nil
}

// Resolve maps one member name to its commit path inside the guard root.
// It enforces the two conjuncts of containment independently: the member
// grammar (CheckMemberPath, which refuses traversal lexically) and the
// joined-prefix check below (which refuses anything whose cleaned join
// escapes the root even if the grammar ever admits a new shape). Either
// conjunct refuses alone.
//
// The prefix check is currently unreachable through this entry: the
// member grammar refuses every parent segment, so no admitted member can
// produce a join outside the root. It stays as defense in depth against a
// future grammar relaxation, and TestGuardPrefixCheckIsLoadBearing pins
// the property it guards (every resolved join stays within the root) so
// weakening the grammar reddens there first.
func (guard Guard) Resolve(member string) (string, error) {
	if err := CheckMemberPath(guard.platform, member); err != nil {
		return "", err
	}
	if guard.checkManaged && !guard.managed[member] {
		return "", failPath("member unmanaged", memberErrorTarget(member))
	}
	joined := joinSlash(guard.slashRoot(), member)
	if !withinRoot(guard.slashRoot(), joined) {
		return "", failPath("member containment", memberErrorTarget(member))
	}
	if guard.platform == scalar.PlatformWindows {
		return strings.ReplaceAll(joined, "/", `\`), nil
	}
	return joined, nil
}

// slashRoot renders the validated root with forward slashes so the join
// and prefix check are platform-parameterized rather than host-native:
// windows cases are decided lexically on any host.
func (guard Guard) slashRoot() string {
	if guard.platform == scalar.PlatformWindows {
		return strings.ReplaceAll(guard.root, `\`, "/")
	}
	return guard.root
}

// joinSlash joins a slash-form root with a grammar-checked member and
// cleans the result, preserving a UNC leading double slash that path
// Clean would otherwise collapse.
func joinSlash(slashRoot, member string) string {
	full := strings.TrimSuffix(slashRoot, "/") + "/" + member
	cleaned := path.Clean(full)
	if strings.HasPrefix(slashRoot, "//") && !strings.HasPrefix(cleaned, "//") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

// withinRoot reports whether the cleaned join stays inside the cleaned
// root: equality (the root itself) or a slash-boundary prefix. A bare
// string prefix is not containment: "/root-evil" starts with "/root" but
// is not inside it.
func withinRoot(slashRoot, joined string) bool {
	base := strings.TrimSuffix(slashRoot, "/")
	if base == "" {
		base = "/"
	}
	if joined == base {
		return true
	}
	return strings.HasPrefix(joined, base+"/")
}

// DetectCaseCollision reports whether candidate collides with an existing
// member name under case folding for a case-insensitive destination
// (Section 16.3 "case-fold collisions"). It returns the colliding entry.
// Comparison is Unicode simple folding via EqualFold over the exact entry
// spellings the caller lists: the caller owns the directory read, this
// function owns the decision.
func DetectCaseCollision(seen []string, candidate string) (string, bool) {
	for _, existing := range seen {
		if existing != candidate && strings.EqualFold(existing, candidate) {
			return existing, true
		}
	}
	return "", false
}

// CheckRegularTarget refuses a stat result that is not a plain regular
// file: symlinks, directories, devices, named pipes, and sockets
// (Section 16.3 "symlink/reparse escape" and "device/FIFO/socket
// creation"). A nil result is refused rather than treated as absent: a
// failed stat is never a legitimate absence.
func CheckRegularTarget(info os.FileInfo) error {
	if info == nil {
		return failPath("target stat missing", "nil file info")
	}
	mode := info.Mode()
	switch {
	case mode&os.ModeSymlink != 0:
		return failPath("target symlink", info.Name())
	case info.IsDir():
		return failPath("target directory", info.Name())
	case mode&os.ModeDevice != 0,
		mode&os.ModeNamedPipe != 0,
		mode&os.ModeSocket != 0,
		mode&os.ModeCharDevice != 0,
		!mode.IsRegular():
		return failPath("target special file", info.Name())
	}
	return nil
}
