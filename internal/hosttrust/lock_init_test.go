package hosttrust

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// firstOpenBarrierFS pauses one child process immediately after it observes
// the lock file absent, so the parent can create and hold the lock first.
// Only the absence observation is synchronized; creation, verification,
// opening and locking are production code against OS files.
type firstOpenBarrierFS struct {
	osFileSystem
	dir string
}

func (filesystem firstOpenBarrierFS) Lstat(path string) (fs.FileInfo, error) {
	info, err := filesystem.osFileSystem.Lstat(path)
	if filepath.Base(path) == lockFileName && os.IsNotExist(err) {
		if err := os.WriteFile(filepath.Join(filesystem.dir, "observed"), []byte("ready"), 0o600); err != nil {
			panic(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			if _, err := os.Stat(filepath.Join(filesystem.dir, "continue")); err == nil {
				break
			}
			if time.Now().After(deadline) {
				panic("first-open barrier timed out")
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	return info, err
}

// Concurrent first-open must never replace a held authorization lock: the
// delayed child loses the atomic creation race, opens the existing inode,
// and blocks on the held lock instead of returning early with a split
// lock. Port of the independent reviewer's cross-process lock probe.
func TestFirstOpenPreservesHeldLock(t *testing.T) {
	if dir := os.Getenv("AX_FIRST_OPEN_BARRIER"); dir != "" {
		if _, err := openStore(resolvedPathsForTest(t, filepath.Join(dir, "config.toml"), dir), firstOpenBarrierFS{dir: dir}); err != nil {
			t.Fatal(err)
		}
		return
	}
	dir := t.TempDir()
	child := exec.Command(os.Args[0], "-test.run=^TestFirstOpenPreservesHeldLock$", "-test.timeout=10s")
	child.Env = append(os.Environ(), "AX_FIRST_OPEN_BARRIER="+dir)
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill() }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "observed")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child did not reach the absent-lock barrier")
		}
		time.Sleep(5 * time.Millisecond)
	}
	store, err := Open(resolvedPathsForTest(t, filepath.Join(dir, "config.toml"), dir))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Initialize(); err != nil {
		t.Fatal(err)
	}
	release, err := store.lock.exclusive()
	if err != nil {
		t.Fatal(err)
	}
	held, err := os.Stat(store.lock.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "continue"), []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	var early bool
	select {
	case err = <-done:
		early = true
	case <-time.After(300 * time.Millisecond):
	}
	release()
	if !early {
		err = <-done
	}
	if err != nil {
		t.Fatalf("child Open error = %v", err)
	}
	current, err := os.Stat(store.lock.path)
	if err != nil {
		t.Fatal(err)
	}
	if early || !os.SameFile(held, current) {
		t.Fatalf("concurrent first-open split the authorization lock: returnedWhileHeld=%v sameInode=%v", early, os.SameFile(held, current))
	}
}

// Concurrent opens of an existing store attach to the same lock inode, and
// a new open blocks while the authorization lock is held elsewhere.
func TestConcurrentOpenAttachesToExistingLock(t *testing.T) {
	store := setupStoreForTest(t)
	state := filepath.Join(store.root, "..")
	const openers = 8
	type result struct {
		store *Store
		err   error
	}
	results := make(chan result, openers)
	for range openers {
		go func() {
			opened, err := Open(resolvedPathsForTest(t, filepath.Join(state, "config.toml"), state))
			results <- result{store: opened, err: err}
		}()
	}
	wanted, err := os.Stat(store.lock.path)
	if err != nil {
		t.Fatal(err)
	}
	for range openers {
		outcome := <-results
		if outcome.err != nil {
			t.Fatalf("concurrent Open error = %v", outcome.err)
		}
		got, err := os.Stat(outcome.store.lock.path)
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(wanted, got) {
			t.Fatal("concurrent Open attached to a different lock inode")
		}
	}
	// A new open must wait for a held exclusive lock: recovery runs under
	// it, so opening beside a live commit would read mid-flight state.
	release, err := store.lock.exclusive()
	if err != nil {
		t.Fatal(err)
	}
	opened := make(chan error, 1)
	go func() {
		_, err := Open(resolvedPathsForTest(t, filepath.Join(state, "config.toml"), state))
		opened <- err
	}()
	select {
	case err := <-opened:
		t.Fatalf("Open completed while the authorization lock was held: %v", err)
	case <-time.After(250 * time.Millisecond):
	}
	release()
	if err := <-opened; err != nil {
		t.Fatalf("Open after release error = %v", err)
	}
}
