//go:build windows

package tmuxserver

// Native Windows MUST NOT claim tmux or tmux resurrection (pinned
// SPEC 1458), so the Windows runtime-directory arms refuse every
// request. The lexical path in runtime.go already refuses
// PlatformWindows on every host before these entries run; the stubs
// exist because Go requires the symbols to compile under
// GOOS=windows, and they fail closed so a weakened lexical gate
// still reaches no tmux directory on this host.
func commitRuntimeDir(root, name, joined string, hooks *Hooks) error {
	return &Error{Code: CodeInvalidArguments, Detail: "runtime platform host"}
}

// verifyRuntimeDir refuses on Windows for the same reason as the
// commit arm above: no tmux runtime directory exists on this host.
func verifyRuntimeDir(root, name string) error {
	return &Error{Code: CodeInvalidArguments, Detail: "runtime platform host"}
}
