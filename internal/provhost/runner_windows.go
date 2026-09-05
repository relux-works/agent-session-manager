//go:build windows

package provhost

import (
	"context"
	"os/exec"
)

// newCommandContext is the Windows process start: the default
// context kill applies to the direct child only. A child tree holding
// the pipes past the kill is a stated bound on this platform; the wait
// still ends via WaitDelay and the call still reports provider_timeout.
func newCommandContext(ctx context.Context, executable string) *exec.Cmd {
	return exec.CommandContext(ctx, executable)
}
