// Package tmuxserver owns AX private tmux server management: owner-only
// runtime directories and dedicated tmux -S servers with no ambient or
// default-server discovery or reuse. It is the first leaf of the
// production-tmux-backend story.
//
// Normative scope: AX v0.7.0 Section 3.2 (platform paths: the
// dedicated socket `<runtime>/tmux/ax.sock` under the Runtime IPC
// root — the socket is runtime IPC, never durable identity) and
// Section 4.2 (tmux backend, dedicated -S server,
// credential-sensitive creation, background broker-or-refuse,
// launchctl managername as diagnostic hint only, cached observations
// never authorize resume), Section 4.C (operation vocabulary cited for
// the wrapper entrypoint shape only), Section 4.D (capability evidence
// cited landed), and Section 4.E (sensitive runtime state is owner-only
// and non-replicable).
//
// Division of authority (composed, never forked): the caller vocabulary
// (foreground/background) comes from internal/axpane, which already
// gates the wrapper decision over it; the capability_unavailable typed
// refusal comes from internal/axerror.NewRealmEvidenceUnavailable, which
// enforces the exact Section 15.3 detail set; the attestation verdict
// comes from the landed internal/terminalbackend admission
// (Reconcile/ResolveEvidence), consumed here as an Admitted set
// composed only with the generation equality the wrapper owner
// already enforces — sentinel/smoke sufficiency, signature
// verification, liveness, and expiry are the admission's verdict, not
// this leaf's; path grammars come from internal/scalar (absolute root,
// relative member, UUIDv7 session); the Runtime IPC root value comes
// from the landed internal/localstore layout (PathRuntime), resolved
// by the caller and supplied as Request.Root — this leaf resolves no
// root of its own; lexical containment comes from
// internal/secprim.Guard; argv shape validation comes from
// internal/secprim.CheckArgv. What this package owns is the composition
// no landed package provides: the descriptor-relative runtime-directory
// commit (mkdirat from a pinned root handle, not a path-string
// MkdirAll), the three independent custody gates (mode, ownership,
// containment), the five-member ambient non-use proof (TMUX and
// TMUX_TMPDIR environment values, inherited socket, default path,
// conventional name), the background broker-or-refuse acquisition path
// with its no-fallback gate over a typed same-user generation-bound
// broker principal, the foreground ensure-then-attach-or-spawn path,
// and the dedicated -S argv builder.
//
// Bounds: no ax CLI surface, no lifecycle operations (create, attach,
// status, quiesce, stop, terminate, restore belong to the sibling
// leaves), no ConPTY process control, no provider process, no mesh
// transport, and no production probe or spawner implementation —
// caller-supplied probes report the landed admission verdicts, and the
// exec/probe adapters are sibling scope. Native Windows refuses every
// request. No test in this package starts a tmux process: every
// broker, server, and spawner interaction runs through injected
// dependencies, and the suite passes with no tmux installed.
package tmuxserver
