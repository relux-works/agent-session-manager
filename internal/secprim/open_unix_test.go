//go:build !windows

package secprim

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// stageTree builds the B1 escape fixture: a staging root holding one
// honest member file, plus a sibling "outside" tree whose loot file is
// reachable only through the symlinked intermediate component
// stage/sub -> outside. Symlink creation needs privilege on Windows,
// so this file is unix-only; Windows keeps its stated check-then-open
// bound.
func stageTree(t *testing.T) (stage, outside string) {
	t.Helper()
	base := t.TempDir()
	stage = filepath.Join(base, "stage")
	outside = filepath.Join(base, "outside")
	if err := os.MkdirAll(filepath.Join(stage, "ok"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "ok", "file"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "loot"), []byte("ESCAPED"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(stage, "sub")); err != nil {
		t.Skipf("no symlink privilege on %s", runtime.GOOS)
	}
	return stage, outside
}

// TestGuardOpenRefusesSymlinkedParent is the B1 vector: Guard.Resolve is
// lexical and still maps "sub/loot" inside the root, while the commit
// walk refuses it, because O_NOFOLLOW constrains only the final path
// component and a symlinked intermediate would otherwise redirect the
// open outside the root. The test pins both halves: Resolve admits
// (documenting that it answers the lexical question only) and Open
// refuses through the production commit entry.
func TestGuardOpenRefusesSymlinkedParent(t *testing.T) {
	t.Parallel()
	stage, _ := stageTree(t)
	guard, err := NewGuard(stage, scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := guard.Resolve("sub/loot")
	if err != nil {
		t.Fatalf("Resolve stopped being lexical: %v", err)
	}
	if resolved != filepath.Join(stage, "sub", "loot") {
		t.Fatalf("Resolve(sub/loot) = %q, want the lexical join", resolved)
	}
	root, err := OpenNoFollowDir(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	start := time.Now()
	_, err = guard.Open(root, "sub/loot")
	requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member symlink escape: sub/loot")
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("commit refusal took %v; the walk must fail fast, not stall", elapsed)
	}
}

// TestGuardOpenRefusesDeepSymlinkedParent is the C2 vector: a symlinked
// intermediate at depth 2 or 3 (three- and four-segment members) must
// refuse as "member symlink escape", the same rule as a first-level
// hop. Before the classify-before-close fix these refused as
// "open failed" instead, because classifyCommitErr fstat'd a descriptor
// closed one line earlier (from the second iteration onward
// previous == current). The witnesses pin the full refusal string, not
// the fact that some error occurred: a misclassified rule reddens
// here even though every branch still refuses.
func TestGuardOpenRefusesDeepSymlinkedParent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	stage := filepath.Join(base, "stage")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{
		filepath.Join(stage, "l1"),
		filepath.Join(stage, "a", "b"),
		outside,
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "loot"), []byte("ESCAPED"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The symlink stands on a non-first intermediate in both members:
	// l1/hop/loot carries it at depth 2, a/b/hop/loot at depth 3.
	for _, link := range []string{
		filepath.Join(stage, "l1", "hop"),
		filepath.Join(stage, "a", "b", "hop"),
	} {
		if err := os.Symlink(outside, link); err != nil {
			t.Skipf("no symlink privilege on %s", runtime.GOOS)
		}
	}
	guard, err := NewGuard(stage, scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := OpenNoFollowDir(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	for member := range map[string]bool{"l1/hop/loot": true, "a/b/hop/loot": true} {
		_, err := guard.Open(root, member)
		requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member symlink escape: "+member)
	}
}

// TestGuardOpenAdmitsHonestMembers proves the walk reaches through
// honest intermediate directories to the same bytes a direct open
// reads: the escape refusal above is narrow to the symlink shape, not
// a blanket commit failure.
func TestGuardOpenAdmitsHonestMembers(t *testing.T) {
	t.Parallel()
	stage, _ := stageTree(t)
	guard, err := NewGuard(stage, scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	root, err := OpenNoFollowDir(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	file, err := guard.Open(root, "ok/file")
	if err != nil {
		t.Fatalf("honest member refused: %v", err)
	}
	defer func() { _ = file.Close() }()
	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "inside" {
		t.Fatalf("commit read %q, want the staged bytes", contents)
	}
}

// TestGuardOpenRefusesCommitShapes drives every other commit refusal
// through the production entry: a trailing symlink (final-component
// escape), a missing member, an unmanaged member, a nil root handle,
// and a non-directory root handle.
func TestGuardOpenRefusesCommitShapes(t *testing.T) {
	t.Parallel()
	stage, _ := stageTree(t)
	guard, err := NewGuard(stage, scalar.PlatformLinux, nil)
	if err != nil {
		t.Fatal(err)
	}
	managed, err := NewGuard(stage, scalar.PlatformLinux, []string{"ok/file"})
	if err != nil {
		t.Fatal(err)
	}
	root, err := OpenNoFollowDir(stage)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	plain, err := os.Open(filepath.Join(stage, "ok", "file"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = plain.Close() }()
	final := filepath.Join(stage, "final-link")
	if err := os.Symlink(filepath.Join(stage, "ok", "file"), final); err != nil {
		t.Skipf("no symlink privilege on %s", runtime.GOOS)
	}
	cases := []struct {
		name   string
		guard  Guard
		root   *os.File
		member string
		check  func(t *testing.T, err error)
	}{
		{
			name: "final symlink", guard: guard, root: root, member: "final-link",
			check: func(t *testing.T, err error) {
				t.Helper()
				requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member symlink escape: final-link")
			},
		},
		{
			name: "missing member", guard: guard, root: root, member: "ok/absent",
			check: func(t *testing.T, err error) {
				t.Helper()
				requireRefusalPrefix(t, err, ErrUnsafePath, "secprim unsafe path: open failed: ok/absent")
			},
		},
		{
			name: "missing intermediate", guard: guard, root: root, member: "nodir/file",
			check: func(t *testing.T, err error) {
				t.Helper()
				requireRefusalPrefix(t, err, ErrUnsafePath, "secprim unsafe path: open failed: nodir/file")
			},
		},
		{
			name: "unmanaged member", guard: managed, root: root, member: "ok/other",
			check: func(t *testing.T, err error) {
				t.Helper()
				requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member unmanaged: ok/other")
			},
		},
		{
			name: "member grammar", guard: guard, root: root, member: "../escape",
			check: func(t *testing.T, err error) {
				t.Helper()
				requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: member grammar: ../escape")
			},
		},
		{
			name: "nil root handle", guard: guard, root: nil, member: "ok/file",
			check: func(t *testing.T, err error) {
				t.Helper()
				requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: guard root handle: ok/file")
			},
		},
		{
			name: "file root handle", guard: guard, root: plain, member: "ok/file",
			check: func(t *testing.T, err error) {
				t.Helper()
				requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: guard root handle: ok/file")
			},
		},
	}
	// The subtests share the root handle and run inline: a parallel
	// subtest would outlive the parent's deferred Close and stat a
	// closed descriptor.
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			_, err := kase.guard.Open(kase.root, kase.member)
			kase.check(t, err)
		})
	}
}

// TestOpenNoFollowFileRefusesFifoWithoutBlocking is the B2 vector: a
// read-only open(2) of a FIFO with no writer blocks in the kernel, and
// the shape check that refuses it runs only after the open returns, so
// without O_NONBLOCK the refusal is unreachable and the opener hangs.
// The goroutine plus timeout turns a regression into a failure instead
// of a stalled suite: with the flag the refusal arrives immediately.
func TestOpenNoFollowFileRefusesFifoWithoutBlocking(t *testing.T) {
	t.Parallel()
	fifo := filepath.Join(t.TempDir(), "pipe")
	makeFifo(t, fifo)
	done := make(chan error, 1)
	go func() {
		file, err := OpenNoFollowFile(fifo)
		if err == nil {
			_ = file.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		requireRefusal(t, err, ErrUnsafePath, "secprim unsafe path: target special file: pipe")
	case <-time.After(10 * time.Second):
		t.Fatal("OpenNoFollowFile blocked on a FIFO for 10s instead of refusing")
	}
}
