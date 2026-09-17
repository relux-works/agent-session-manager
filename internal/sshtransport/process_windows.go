package sshtransport

import "os/exec"

// Windows uses os/exec's native CreateProcess argument encoding. Descendant
// termination is not claimed here; local SSH is the directly owned process.
func configureProcess(cmd *exec.Cmd) {}
func killProcess(cmd *exec.Cmd)      { _ = cmd.Process.Kill() }
