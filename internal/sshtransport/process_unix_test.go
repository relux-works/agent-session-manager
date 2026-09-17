//go:build !windows

package sshtransport

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func fixtureDescendant(mode string) {
	executable, _ := os.Executable()
	cmd := exec.Command(executable)
	cmd.Env = []string{"AX_SSH_FIXTURE=hold", "GOCOVERDIR=" + os.Getenv("GOCOVERDIR"), "GORACE=atexit_sleep_ms=0"}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if mode == "detached" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if err := cmd.Start(); err != nil {
		os.Exit(91)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]int{"pid": cmd.Process.Pid})
}

func TestInheritedPipeCleanup(t *testing.T) {
	for _, mode := range []string{"descendant", "detached"} {
		t.Run(mode, func(t *testing.T) {
			s := open(t, client(t, mode, ""))
			line, err := s.Receive()
			if err != nil {
				t.Fatal(err)
			}
			var identity struct {
				PID int `json:"pid"`
			}
			if json.Unmarshal(line, &identity) != nil || identity.PID <= 0 {
				t.Fatal("missing fixture identity")
			}
			t.Cleanup(func() { _ = syscall.Kill(identity.PID, syscall.SIGKILL) })
			if err = s.Wait(); !errors.Is(err, ErrDrain) {
				t.Fatalf("inherited pipe: %v", err)
			}
			if s.cmd.ProcessState == nil {
				t.Fatal("direct process not reaped")
			}
			if mode == "descendant" {
				deadline := time.Now().Add(2 * time.Second)
				for syscall.Kill(identity.PID, 0) == nil && time.Now().Before(deadline) {
					time.Sleep(5 * time.Millisecond)
				}
				if err := syscall.Kill(identity.PID, 0); err != syscall.ESRCH {
					t.Fatal("owned group descendant survived")
				}
			} else if syscall.Kill(identity.PID, 0) != nil {
				t.Fatal("detached descendant unexpectedly within ownership; bound fixture invalid")
			}
		})
	}
}

func TestReadFailureAndRecovery(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			c := client(t, "stall", "")
			s := open(t, c)
			if _, err := s.Receive(); err != nil {
				t.Fatal(err)
			}
			// Fault the real owned OS pipe, not a parser or a fake Runner.
			if stream == "stdout" {
				_ = s.stdout.Close()
			} else {
				_ = s.stderr.Close()
			}
			if err := s.Wait(); !errors.Is(err, ErrStream) {
				t.Fatal(err)
			}
			c.env[0] = "AX_SSH_FIXTURE=empty"
			next := open(t, c)
			if err := next.Wait(); err != nil {
				t.Fatalf("recovery: %v", err)
			}
		})
	}
}

func TestCancelExitRace(t *testing.T) {
	c := client(t, "empty", "")
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		s, err := c.Open(ctx, "peer")
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		err = s.Wait()
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if s.cmd.ProcessState == nil {
			t.Fatal("direct child not reaped")
		}
	}
}
