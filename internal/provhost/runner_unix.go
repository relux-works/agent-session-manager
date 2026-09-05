//go:build !windows

package provhost

import (
	"context"
	"os/exec"
	"syscall"
)

// newCommandContext starts the plugin in its own process group so the
// deadline kill reaches the whole tree: provider CLIs routinely spawn
// children, and killing only the parent would leave grandchildren holding
// the pipes open, hanging the wait past the deadline. A grandchild that
// deliberately escapes with setsid is beyond this bound; see doc.go.
func newCommandContext(ctx context.Context, executable string) *exec.Cmd {
	command := exec.CommandContext(ctx, executable)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	return command
}
