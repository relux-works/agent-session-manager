package secprim

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// requireRefusal asserts the full refusal rendering: kind, rule, and
// detail. A code or suffix several gates share proves nothing about which
// gate fired, so witnesses pin the whole string.
func requireRefusal(t *testing.T, err error, kind error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("admitted; want refusal %q", want)
	}
	if !errors.Is(err, kind) {
		t.Fatalf("error %v does not match kind %v; want %q", err, kind, want)
	}
	if err.Error() != want {
		t.Fatalf("refusal = %q, want %q", err.Error(), want)
	}
}

// requireRefusalPrefix pins the stable rule prefix of a cause-carrying
// refusal. The appended OS cause text differs by platform, so only the
// prefix is pinned here; the cause itself is proven with errors.Is at the
// call site.
func requireRefusalPrefix(t *testing.T, err error, kind error, wantPrefix string) {
	t.Helper()
	if err == nil {
		t.Fatalf("admitted; want refusal with prefix %q", wantPrefix)
	}
	if !errors.Is(err, kind) {
		t.Fatalf("error %v does not match kind %v; want prefix %q", err, kind, wantPrefix)
	}
	if !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Fatalf("refusal = %q, want prefix %q", err.Error(), wantPrefix)
	}
}

func TestCheckMemberPathAdmits(t *testing.T) {
	t.Parallel()
	members := []string{
		"a", "a/b/c", "dir.with.dots/file", "100%.txt", "a%20b",
		"a b/c d", "ünïcodé/名前", ".hidden", "a/.hidden/b",
		// A colon is legal on POSIX past the scalar drive-prefix rule
		// ("ab:c" is not drive-shaped); the ADS rule is Windows-only.
		"ab:c",
	}
	for _, platform := range []scalar.Platform{scalar.PlatformMacOS, scalar.PlatformLinux, scalar.PlatformWSL2} {
		for _, member := range members {
			if err := CheckMemberPath(platform, member); err != nil {
				t.Errorf("CheckMemberPath(%s, %q) = %v, want admission", platform, member, err)
			}
		}
	}
	windowsMembers := []string{"a", "a/b/c", "100%.txt", "a%20b", ".hidden"}
	for _, member := range windowsMembers {
		if err := CheckMemberPath(scalar.PlatformWindows, member); err != nil {
			t.Errorf("CheckMemberPath(windows, %q) = %v, want admission", member, err)
		}
	}
}

func TestCheckMemberPathRefuses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		platform scalar.Platform
		member   string
		want     string
	}{
		{"empty", scalar.PlatformLinux, "", "secprim unsafe path: member grammar: "},
		{"dot", scalar.PlatformLinux, ".", "secprim unsafe path: member grammar: ."},
		{"parent", scalar.PlatformLinux, "..", "secprim unsafe path: member grammar: .."},
		{"nested parent", scalar.PlatformLinux, "a/../b", "secprim unsafe path: member grammar: a/../b"},
		{"nested dot", scalar.PlatformLinux, "a/./b", "secprim unsafe path: member grammar: a/./b"},
		{"empty segment", scalar.PlatformLinux, "a//b", "secprim unsafe path: member grammar: a//b"},
		{"absolute", scalar.PlatformLinux, "/abs", "secprim unsafe path: member grammar: /abs"},
		{"trailing slash", scalar.PlatformLinux, "a/", "secprim unsafe path: member grammar: a/"},
		{"backslash", scalar.PlatformLinux, `a\b`, `secprim unsafe path: member grammar: a\b`},
		{"nul", scalar.PlatformLinux, "a\x00b", "secprim unsafe path: member grammar: a\x00b"},
		{"drive", scalar.PlatformLinux, "c:x", "secprim unsafe path: member grammar: c:x"},
		{"encoded slash", scalar.PlatformLinux, "a%2fb", "secprim unsafe path: member grammar: a%2fb"},
		{"encoded slash upper", scalar.PlatformLinux, "a%2Fb", "secprim unsafe path: member grammar: a%2Fb"},
		{"encoded backslash", scalar.PlatformLinux, "a%5cb", "secprim unsafe path: member grammar: a%5cb"},
		{"double-encoded slash", scalar.PlatformLinux, "a%252fb", "secprim unsafe path: member grammar: a%252fb"},
		{"encoded dot", scalar.PlatformLinux, "a%2e%2e/b", "secprim unsafe path: member encoded dot: a%2e%2e/b"},
		{"encoded dot single", scalar.PlatformLinux, "%2e", "secprim unsafe path: member encoded dot: %2e"},
		{"encoded dot upper", scalar.PlatformLinux, "a%2E/b", "secprim unsafe path: member encoded dot: a%2E/b"},
		{"double-encoded dot", scalar.PlatformLinux, "a%252e/b", "secprim unsafe path: member encoded dot: a%252e/b"},
		{"overlong separator", scalar.PlatformLinux, "a%c0%afb", "secprim unsafe path: member overlong separator: a%c0%afb"},
		{"overlong separator upper", scalar.PlatformLinux, "a%C0%AFb", "secprim unsafe path: member overlong separator: a%C0%AFb"},
		{"double-encoded overlong", scalar.PlatformLinux, "a%25c0%afb", "secprim unsafe path: member overlong separator: a%25c0%afb"},
		{"windows ads", scalar.PlatformWindows, "ab:c", "secprim unsafe path: member alternate stream: ab:c"},
		{"windows ads nested", scalar.PlatformWindows, "a/b:c/d", "secprim unsafe path: member alternate stream: a/b:c/d"},
		{"windows drive", scalar.PlatformWindows, "a:b", "secprim unsafe path: member grammar: a:b"},
		{"posix drive", scalar.PlatformLinux, "a:b", "secprim unsafe path: member grammar: a:b"},
		{"windows con", scalar.PlatformWindows, "con", "secprim unsafe path: member reserved device name: con"},
		{"windows con upper", scalar.PlatformWindows, "CON", "secprim unsafe path: member reserved device name: CON"},
		{"windows con ext", scalar.PlatformWindows, "con.txt", "secprim unsafe path: member reserved device name: con.txt"},
		{"windows lpt", scalar.PlatformWindows, "dir/lpt1", "secprim unsafe path: member reserved device name: dir/lpt1"},
		{"windows com", scalar.PlatformWindows, "com9.log", "secprim unsafe path: member reserved device name: com9.log"},
		{"windows trailing dot", scalar.PlatformWindows, "a./b", "secprim unsafe path: member trailing dot or space: a./b"},
		{"windows trailing space", scalar.PlatformWindows, "a /b", "secprim unsafe path: member trailing dot or space: a /b"},
		{"windows backslash", scalar.PlatformWindows, `a\b`, `secprim unsafe path: member grammar: a\b`},
		{"windows absolute", scalar.PlatformWindows, `C:\x`, `secprim unsafe path: member grammar: C:\x`},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			t.Parallel()
			err := CheckMemberPath(kase.platform, kase.member)
			requireRefusal(t, err, ErrUnsafePath, kase.want)
		})
	}
}

func TestCheckMemberPathTruncatesLongMembers(t *testing.T) {
	t.Parallel()
	member := strings.Repeat("a", 40) + "/../" + strings.Repeat("b", 40)
	err := CheckMemberPath(scalar.PlatformLinux, member)
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member grammar: "+member[:64]+"...")
}

func TestGuardResolve(t *testing.T) {
	t.Parallel()
	guard, err := NewGuard("/stage/root", scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatalf("NewGuard = %v", err)
	}
	for member, want := range map[string]string{
		"a":     "/stage/root/a",
		"a/b/c": "/stage/root/a/b/c",
	} {
		got, err := guard.Resolve(member)
		if err != nil {
			t.Errorf("Resolve(%q) = %v", member, err)
			continue
		}
		if got != want {
			t.Errorf("Resolve(%q) = %q, want %q", member, got, want)
		}
	}
	if _, err := NewGuard("relative/root", scalar.PlatformLinux, nil); err == nil {
		t.Error("NewGuard(relative) admitted; the root must be absolute")
	} else {
		requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: guard root: relative/root")
	}
	windows, err := NewGuard(`C:\stage`, scalar.PlatformWindows, nil)
	if err != nil {
		t.Fatalf("NewGuard windows = %v", err)
	}
	got, err := windows.Resolve("a/b")
	if err != nil {
		t.Fatalf("Resolve windows = %v", err)
	}
	if got != `C:\stage\a\b` {
		t.Fatalf("Resolve windows = %q, want native separators", got)
	}
	unc, err := NewGuard(`\\server\share`, scalar.PlatformWindows, nil)
	if err != nil {
		t.Fatalf("NewGuard UNC = %v", err)
	}
	got, err = unc.Resolve("a/b")
	if err != nil {
		t.Fatalf("Resolve UNC = %v", err)
	}
	if got != `\\server\share\a\b` {
		t.Fatalf("Resolve UNC = %q, want the share preserved", got)
	}
}

func TestGuardManagedSet(t *testing.T) {
	t.Parallel()
	open, err := NewGuard("/stage", scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := open.Resolve("anything/goes"); err != nil {
		t.Fatalf("nil managed set refused a member: %v", err)
	}
	empty, err := NewGuard("/stage", scalar.PlatformLinux, []string{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = empty.Resolve("a")
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member unmanaged: a")
	exact, err := NewGuard("/stage", scalar.PlatformLinux, []string{"a/b"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exact.Resolve("a/b"); err != nil {
		t.Fatalf("managed member refused: %v", err)
	}
	_, err = exact.Resolve("a/c")
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member unmanaged: a/c")
	// The managed set is exact, never case-folded: a case variant of a
	// managed member is a different member and refuses as unmanaged.
	_, err = exact.Resolve("A/B")
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member unmanaged: A/B")
}

// TestGuardPrefixCheckIsLoadBearing is the structural tripwire for the
// defensive joined-prefix conjunct in Resolve: the member grammar refuses
// every parent segment, so no admitted member can escape the root, and
// this test proves that property over a generated hostile corpus rather
// than trusting the grammar. If the grammar ever weakens, an escaped join
// reddens here first.
func TestGuardPrefixCheckIsLoadBearing(t *testing.T) {
	t.Parallel()
	segments := []string{"a", ".", "..", "", "b c", "%2e%2e", "%2f", "x"}
	guard, err := NewGuard("/stage/root", scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	escaped := 0
	admitted := 0
	var build func(depth int, current string)
	build = func(depth int, current string) {
		if depth == 3 {
			got, err := guard.Resolve(current)
			if err != nil {
				return
			}
			admitted++
			if got != "/stage/root" && !strings.HasPrefix(got, "/stage/root/") {
				escaped++
				t.Errorf("Resolve(%q) escaped to %q", current, got)
			}
			return
		}
		for _, segment := range segments {
			next := segment
			if current != "" {
				next = current + "/" + segment
			}
			build(depth+1, next)
		}
	}
	build(0, "")
	if admitted == 0 {
		t.Fatal("no generated member was admitted; the corpus proves nothing")
	}
	if escaped != 0 {
		t.Fatalf("%d admitted members escaped the root", escaped)
	}
	t.Logf("prefix property holds over %d admitted of %d generated members", admitted, 8*8*8)
}

func TestDetectCaseCollision(t *testing.T) {
	t.Parallel()
	if _, found := DetectCaseCollision([]string{"README.md", "src"}, "other"); found {
		t.Fatal("distinct names reported as a collision")
	}
	if _, found := DetectCaseCollision([]string{"README.md"}, "README.md"); found {
		t.Fatal("an identical name is the same file, not a collision")
	}
	hit, found := DetectCaseCollision([]string{"README.md", "src"}, "readme.MD")
	if !found || hit != "README.md" {
		t.Fatalf("case-fold collision missed: %q %v", hit, found)
	}
}

func TestCheckRegularTarget(t *testing.T) {
	t.Parallel()
	requireRefusal(t, CheckRegularTarget(nil), ErrUnsafePath, "secprim unsafe path: target stat missing: nil file info")
	root := t.TempDir()
	regular := filepath.Join(root, "file")
	if err := os.WriteFile(regular, []byte("bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(regular)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckRegularTarget(info); err != nil {
		t.Fatalf("regular file refused: %v", err)
	}
	dirInfo, err := os.Lstat(root)
	if err != nil {
		t.Fatal(err)
	}
	requireRefusal(t, CheckRegularTarget(dirInfo), ErrUnsafePath, "secprim unsafe path: target directory: "+dirInfo.Name())
	// An empty directory refuses exactly like a populated one: size
	// never participates in the directory decision.
	empty := filepath.Join(root, "emptydir")
	if err := os.Mkdir(empty, 0o700); err != nil {
		t.Fatal(err)
	}
	emptyInfo, err := os.Lstat(empty)
	if err != nil {
		t.Fatal(err)
	}
	requireRefusal(t, CheckRegularTarget(emptyInfo), ErrUnsafePath, "secprim unsafe path: target directory: emptydir")
	link := filepath.Join(root, "link")
	if err := os.Symlink(regular, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("no symlink privilege on %s", runtime.GOOS)
		}
		t.Fatal(err)
	}
	linkInfo, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	requireRefusal(t, CheckRegularTarget(linkInfo), ErrUnsafePath, "secprim unsafe path: target symlink: link")
	if runtime.GOOS != "windows" {
		fifo := filepath.Join(root, "fifo")
		makeFifo(t, fifo)
		fifoInfo, err := os.Lstat(fifo)
		if err != nil {
			t.Fatal(err)
		}
		requireRefusal(t, CheckRegularTarget(fifoInfo), ErrUnsafePath, "secprim unsafe path: target special file: fifo")
		nullInfo, err := os.Lstat("/dev/null")
		if err != nil {
			t.Fatal(err)
		}
		requireRefusal(t, CheckRegularTarget(nullInfo), ErrUnsafePath, "secprim unsafe path: target special file: null")
	}
}

func TestOpenNoFollowFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	regular := filepath.Join(root, "file")
	if err := os.WriteFile(regular, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := OpenNoFollowFile(regular)
	if err != nil {
		t.Fatalf("regular file refused: %v", err)
	}
	contents := make([]byte, 7)
	if _, err := file.Read(contents); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if string(contents) != "payload" {
		t.Fatalf("read %q through the no-follow handle", contents)
	}
	subdir := filepath.Join(root, "dir")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = OpenNoFollowFile(subdir)
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: target directory: dir")
	// Cause-carrying refusals pin the stable prefix, not the OS text: the
	// errno rendering differs by platform while the rule and target do
	// not. errors.Is still proves the kind, and errors.Is(err,
	// fs.ErrNotExist) proves the cause survived.
	_, err = OpenNoFollowFile(filepath.Join(root, "missing"))
	target := filepath.Join(root, "missing")
	if len(target) > 64 {
		target = target[:64] + "..."
	}
	requireRefusalPrefix(t, err, ErrUnsafePath, "secprim unsafe path: open failed: "+target)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("open cause does not wrap ErrNotExist: %v", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(regular, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("no symlink privilege on %s", runtime.GOOS)
		}
		t.Fatal(err)
	}
	_, err = OpenNoFollowFile(link)
	if err == nil {
		t.Fatal("symlink traversed by the no-follow open")
	}
	if !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("symlink refusal = %v, want ErrUnsafePath", err)
	}
}

func TestOpenNoFollowDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir, err := OpenNoFollowDir(root)
	if err != nil {
		t.Fatalf("directory refused: %v", err)
	}
	_ = dir.Close()
	regular := filepath.Join(root, "file")
	if err := os.WriteFile(regular, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = OpenNoFollowDir(regular)
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: target directory: file")
	_, err = OpenNoFollowDir(filepath.Join(root, "missing"))
	if err == nil || !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("missing dir refusal = %v, want ErrUnsafePath", err)
	}
	link := filepath.Join(root, "linkdir")
	if err := os.Symlink(root, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("no symlink privilege on %s", runtime.GOOS)
		}
		t.Fatal(err)
	}
	_, err = OpenNoFollowDir(link)
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: target symlink: linkdir")
}

// TestOpenNoFollowDirRefusesUnreadableDir proves the post-check open arm:
// the directory passes the Lstat fast path but the open itself fails.
// Dropping all mode bits makes the open fail deterministically on unix;
// Windows ACLs do not express a mode-000 directory, so the vector is
// skipped there and the site carries a platform exemption in the audit.
func TestOpenNoFollowDirRefusesUnreadableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mode-000 directories are not expressible on Windows")
	}
	if privilegedUser() {
		t.Skip("root bypasses mode bits, so the open cannot fail")
	}
	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	if err := os.Mkdir(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(locked, 0o700) }()
	_, err := OpenNoFollowDir(locked)
	requireRefusalPrefix(t, err, ErrUnsafePath, "secprim unsafe path: open failed: "+namedTarget(locked))
}

// errSeamFailure is the forced stat failure the seam returns. It is a
// test-only value and never crosses a production refusal rendering.
var errSeamFailure = errors.New("secprim forced stat failure")

// namedTarget mirrors the production 64-character member naming bound so
// prefix witnesses pin the truncation contract rather than retyping it.
func namedTarget(path string) string {
	if len(path) > 64 {
		return path[:64] + "..."
	}
	return path
}

// TestOpenStatFailureSeam forces the post-open stat-failure arms through
// the production openers: fstat on a live descriptor does not fail on any
// supported platform, so the seam is the only way to prove these arms
// refuse instead of panicking. The seam is restored before returning.
func TestOpenStatFailureSeam(t *testing.T) {
	previous := statOpenedFile
	defer func() { statOpenedFile = previous }()
	statOpenedFile = func(file *os.File) (os.FileInfo, error) {
		return nil, errSeamFailure
	}
	root := t.TempDir()
	regular := filepath.Join(root, "file")
	if err := os.WriteFile(regular, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := OpenNoFollowFile(regular)
	requireRefusalPrefix(t, err, ErrUnsafePath, "secprim unsafe path: open stat failed: "+namedTarget(regular))
	_, err = OpenNoFollowDir(root)
	requireRefusalPrefix(t, err, ErrUnsafePath, "secprim unsafe path: open stat failed: "+namedTarget(root))
	// The guard commit walk stats the descriptor it opened, so its
	// stat-failure arm fires here too rather than dereferencing a nil
	// result.
	guard, err := NewGuard(root, scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The root handle must be opened with a live stat: briefly restore
	// the production stat around the handle open, then re-arm the seam
	// for the commit stat below.
	statOpenedFile = previous
	dir, err := OpenNoFollowDir(root)
	statOpenedFile = func(file *os.File) (os.FileInfo, error) {
		return nil, errSeamFailure
	}
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dir.Close() }()
	_, err = guard.Open(dir, "file")
	requireRefusalPrefix(t, err, ErrUnsafePath, "secprim unsafe path: open stat failed: file")
	// A nil stat result without an error is refused as missing, never
	// dereferenced.
	statOpenedFile = func(file *os.File) (os.FileInfo, error) {
		return nil, nil
	}
	_, err = OpenNoFollowFile(regular)
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: target stat missing: nil file info")
	_, err = OpenNoFollowDir(root)
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: target directory: "+root)
}
