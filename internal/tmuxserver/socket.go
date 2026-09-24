package tmuxserver

import (
	"path/filepath"
	"strings"
)

// SocketName is the fixed private socket leaf inside the runtime
// directory: pinned SPEC 806-807 selects the dedicated server by
// `tmux -S <runtime>/tmux/ax.sock`, so the leaf is `ax.sock`. AX
// derives its server socket from this name under the owner-only
// directory and from nothing else.
const SocketName = "ax.sock"

// Ambient environment names the four discovery vectors AX must never
// use. TMUXEnv and TMUXTmpDir name the two environment variables
// whose values may carry an ambient socket; InheritedSocket is a
// socket path handed to the process out of band; DefaultPath is the
// operator default socket path; ConventionalName is the bare
// conventional socket name. No member of this struct ever selects
// the acquisition socket: ResolveSocket derives the socket from the
// runtime directory and refuses any explicit override.
type Ambient struct {
	TMUXEnv          string
	TMUXTmpDir       string
	InheritedSocket  string
	DefaultPath      string
	ConventionalName string
}

// Ambient environment variable names. These are the lookup keys the
// caller passes to its own environment function; ObserveAmbient
// records whatever the lookup returns for them without validating
// names or values, because the observation is evidence only and
// resolution never reads it.
const (
	// EnvTMUX is the tmux(1) session environment variable.
	EnvTMUX = "TMUX"
	// EnvTMUXTmpDir is the tmux(1) socket-directory override.
	EnvTMUXTmpDir = "TMUX_TMPDIR"
)

// ConventionalSocketName is the operator default socket leaf that AX
// must never resolve to.
const ConventionalSocketName = "default"

// ObserveAmbient records the ambient socket facts visible to the
// caller without using any of them. The lookup is caller-supplied so
// the production entry never reads the process environment itself:
// ambient discovery cannot hide in an os.Getenv call this package
// never makes. The observation is evidence only; resolution derives
// the socket from the runtime directory and never reads it.
func ObserveAmbient(lookup func(name string) (string, bool), inherited, defPath, conventional string) Ambient {
	observed := Ambient{
		InheritedSocket:  inherited,
		DefaultPath:      defPath,
		ConventionalName: conventional,
	}
	if lookup == nil {
		return observed
	}
	if value, ok := lookup(EnvTMUX); ok {
		observed.TMUXEnv = value
	}
	if value, ok := lookup(EnvTMUXTmpDir); ok {
		observed.TMUXTmpDir = value
	}
	return observed
}

// SocketPath derives the dedicated server socket from the runtime
// directory. It is the only socket constructor: acquisition never
// substitutes an ambient value for its result.
func SocketPath(runtimeDir string) string {
	return filepath.Join(runtimeDir, SocketName)
}

// ResolveSocket returns the acquisition socket for the runtime
// directory. An explicit override refuses with the ambient code no
// matter which vector supplied it: there is no override surface,
// only the derived socket. The ambient observation is carried for
// evidence and asserted disjoint from the result by the caller
// tests; resolution never reads it.
func ResolveSocket(runtimeDir string, override string, ambient Ambient) (string, error) {
	if override != "" {
		return "", &Error{Code: CodeAmbientReuse, Detail: "socket override"}
	}
	socket := SocketPath(runtimeDir)
	if err := checkAmbientDisjoint(socket, ambient); err != nil {
		return "", err
	}
	return socket, nil
}

// checkAmbientDisjoint refuses the degenerate case where an ambient
// value already names the derived socket: using the socket then
// would be indistinguishable from reusing the ambient server, so
// acquisition refuses instead of proceeding on an ambiguous fact.
// The comparison runs over cleaned spellings, so an ambient value
// that names the socket through a `/./` segment or a doubled
// separator refuses like the byte-identical spelling; the TMUX
// member compares its socket component, because tmux(1) encodes that
// variable as `<socket>,<pid>,<index>`. A symlink alias of the root
// that resolves to the same socket is not detected and stays a
// stated bound.
func checkAmbientDisjoint(socket string, ambient Ambient) error {
	for _, candidate := range []string{
		tmuxEnvSocket(ambient.TMUXEnv),
		ambient.TMUXTmpDir,
		ambient.InheritedSocket,
		ambient.DefaultPath,
		ambient.ConventionalName,
	} {
		if candidate != "" && filepath.Clean(candidate) == socket {
			return &Error{Code: CodeAmbientReuse, Detail: "ambient collision"}
		}
	}
	return nil
}

// tmuxEnvSocket returns the socket component of a TMUX environment
// value, which tmux(1) encodes as `<socket>,<pid>,<index>`: the
// substring before the first comma. A value without a comma compares
// whole, and an empty component compares as absent.
func tmuxEnvSocket(value string) string {
	if i := strings.IndexByte(value, ','); i >= 0 {
		return value[:i]
	}
	return value
}
