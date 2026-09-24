package tmuxserver

import (
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// BuildArgv builds the dedicated server spawn vector: tmux addressed
// by the exact -S socket with the stable pane entrypoint the spec
// requires every managed pane to run (ax pane SESSION_ID). The
// socket must be the one derived from the runtime directory: an
// ambient socket passed here refuses instead of being addressed.
// The session identity delegates to the landed UUIDv7 grammar, and
// the assembled vector passes the landed argv shape gate before it
// is returned, so a malformed vector never reaches the spawner.
func BuildArgv(runtimeDir, socket, sessionID string) ([]string, error) {
	if socket != SocketPath(runtimeDir) {
		return nil, &Error{Code: CodeAmbientReuse, Detail: "spawn socket"}
	}
	session, err := scalar.ParseUUIDv7(sessionID)
	if err != nil {
		return nil, &Error{Code: CodeInvalidArguments, Detail: "session identity"}
	}
	argv := []string{
		"tmux",
		"-S", socket,
		"new-session",
		"-d",
		"-s", session.String(),
		"ax", "pane", session.String(),
	}
	if err := secprim.CheckArgv(argv); err != nil {
		return nil, &Error{Code: CodeInvalidArguments, Detail: "spawn argv"}
	}
	return argv, nil
}
