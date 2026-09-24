package termbind

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestAttachAdmissionLockSharedAcrossStores proves that separately opened
// stores arbitrate on one persistent per-instance OS lock and that a
// canceled waiter does not acquire it after the owner releases.
func TestAttachAdmissionLockSharedAcrossStores(t *testing.T) {
	root := t.TempDir()
	first, err := OpenAttachStore(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := OpenAttachStore(root)
	if err != nil {
		t.Fatal(err)
	}
	releaseFirst, err := first.AcquireAdmission(context.Background(), fixtureInstance)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if release, err := second.AcquireAdmission(ctx, fixtureInstance); err == nil {
		release()
		t.Fatal("second store acquired an instance lock held by the first store")
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second acquisition error = %v, want context deadline", err)
	}
	releaseFirst()
	releaseSecond, err := second.AcquireAdmission(context.Background(), fixtureInstance)
	if err != nil {
		t.Fatalf("second store did not acquire after release: %v", err)
	}
	defer releaseSecond()
}

// TestAttachAdmissionLockReleasedAfterProcessExit is both the parent and
// subprocess fixture. The helper holds the kernel lock until the parent
// kills it; the persistent lock file remains reusable after process death.
func TestAttachAdmissionLockReleasedAfterProcessExit(t *testing.T) {
	if root := os.Getenv("AX_TEST_ATTACH_LOCK_HELPER_ROOT"); root != "" {
		store, err := OpenAttachStore(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		release, err := store.AcquireAdmission(context.Background(), fixtureInstance)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stdout, "locked")
		for {
			runtime.KeepAlive(release)
			time.Sleep(time.Hour)
		}
	}

	root := t.TempDir()
	command := exec.Command(os.Args[0], "-test.run=^TestAttachAdmissionLockReleasedAfterProcessExit$")
	command.Env = append(os.Environ(), "AX_TEST_ATTACH_LOCK_HELPER_ROOT="+root)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(stdout).ReadString('\n')
		if err != nil {
			finished <- fmt.Errorf("read child lock-ready marker: %w (%s)", err, stderr.String())
			return
		}
		if line != "locked\n" {
			finished <- fmt.Errorf("child marker = %q, want locked", line)
			return
		}
		finished <- nil
	}()
	select {
	case err := <-finished:
		if err != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatal("child did not acquire the attach admission lock")
	}

	store, err := OpenAttachStore(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if release, err := store.AcquireAdmission(ctx, fixtureInstance); err == nil {
		release()
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatal("parent acquired the lock while the child process held it")
	} else if !errors.Is(err, context.DeadlineExceeded) {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("parent acquisition error = %v, want context deadline", err)
	}
	if err := command.Process.Kill(); err != nil {
		_ = command.Wait()
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("killed child exited successfully")
	}
	release, err := store.AcquireAdmission(context.Background(), fixtureInstance)
	if err != nil {
		t.Fatalf("lock remained held after child process exit: %v", err)
	}
	release()
}
