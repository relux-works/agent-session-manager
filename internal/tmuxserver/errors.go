package tmuxserver

// Local refusal codes. The spec names capability_unavailable for the
// background broker-or-refuse path (emitted through axerror with its
// typed details); the codes below name the package-local safety gates
// the spec states as MUST NOT rules without assigning a wire code.
// Tests assert the literal strings, never these constants.
const (
	// CodeAmbientReuse refuses an explicit socket override: the only
	// admissible server socket is the one derived from the runtime
	// directory, so any caller-supplied socket (ambient environment
	// value, inherited socket, default path, or conventional name)
	// refuses instead of redirecting acquisition.
	CodeAmbientReuse = "tmux_ambient_server_reuse"
	// CodeBackgroundCreationRefused refuses direct server creation
	// on the background path. A background caller never creates; it
	// contacts the broker or receives capability_unavailable.
	CodeBackgroundCreationRefused = "tmux_background_creation_refused"
	// CodeUnsafeRuntimeDir refuses a runtime directory that fails
	// any of the three custody gates: mode, ownership, containment.
	CodeUnsafeRuntimeDir = "tmux_unsafe_runtime_dir"
	// CodeReadinessNotAuthorizing refuses an authorization attempt
	// backed only by a diagnostic hint or a cached observation.
	CodeReadinessNotAuthorizing = "tmux_readiness_not_authorizing"
	// CodeInvalidArguments refuses a malformed acquisition request:
	// unknown caller, unknown platform, malformed generation, or
	// missing typed-detail members.
	CodeInvalidArguments = "tmux_invalid_arguments"
	// CodeSpawnFailed reports a dedicated-server spawn failure.
	// The spawner is caller-supplied; this code carries no fallback.
	CodeSpawnFailed = "tmux_server_spawn_failed"
	// CodeUnsafeSocketPath refuses a socket path, parent, or ancestor
	// whose file kind, ownership, or permissions are unsafe, before
	// bind or connect (SPEC 810-812). It names the lifecycle bind
	// step's rejection; the runtime leaf custody it delegates to keeps
	// its own code.
	CodeUnsafeSocketPath = "tmux_unsafe_socket_path"
	// CodeSocketPathTooLong refuses a derived socket path that cannot
	// fit the platform sun_path, before bind or connect, so a deep
	// runtime root reports the length refusal instead of surfacing as
	// a spawn failure in the adapter.
	CodeSocketPathTooLong = "tmux_socket_path_too_long"
)

// Error is a package-local refusal. Code is one of the Code*
// constants; Detail is a static clause naming the gate. It never
// echoes paths, socket names, generations, or environment values.
type Error struct {
	Code   string
	Detail string
}

func (err *Error) Error() string {
	return "tmux server refused: " + err.Code + " at " + err.Detail
}
