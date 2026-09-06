// Package secprim implements the AX shared safe process and filesystem
// primitives of pinned specification Sections 16.3, 16.4, and 16.7: no-follow
// path handling, staging containment, structured argv, environment
// allowlists, terminal control-string escaping, and secret redaction for
// free-text streams.
//
// The package is a library behind production launch and rendering paths,
// not a second authority beside them:
//
//   - path grammars stay in internal/scalar: CheckMemberPath admits exactly
//     the scalar relative-path grammar plus the Windows alternate-stream,
//     trailing-dot/space, and reserved-device rules Section 16.3 names, and
//     delegates every shared rule to scalar rather than retyping it;
//   - byte-counted exec rules (Provider SpawnPlan argv and env literals,
//     Section 5.1) stay in internal/provhost: that package validates the
//     wire plan, while this package builds the launch environment and
//     validates the argv vector at process start;
//   - environment-name grammar is owned here and internal/provhost delegates
//     to it, so the two packages cannot drift on what a name is;
//   - Structured Error detail redaction stays in internal/axerror: that
//     package refuses exact credential keys and causal leaks on the error
//     wire, while this package scrubs free-text streams (stderr, logs,
//     terminal rendering) where no key gate can see;
//   - SSH argv admission stays in internal/config and the provider entry
//     point rule (`ax pane SESSION_ID`) stays in internal/terminalbackend:
//     both are closed vocabularies over their own domain, not instances of
//     the generic argv rule here;
//   - the five-variable AX_* path registry stays in internal/config and
//     internal/localstore, which already implement AC-PATH-001.
//
// Trust for external executables is deliberately not implemented here.
// internal/provider.trustCandidate and internal/terminalbackend.DigestFile
// already establish trust with different fact sets because their
// specifications differ: Section 7.1 requires the approving owner identity
// beside the canonical path and digest for provider plugins, while the
// Section 4.B trust tuple carries platform, adapter identity, manifest and
// probe digests with no owner fact. A third path here would be another
// copy, not convergence; the difference is contractual, not accidental.
//
// Every refusal is a plain *Error naming the kind, rule, and member:
// facades keep their own refusal dialects, so this package names rules
// instead of minting codes, following internal/environ.
//
// Stated bounds. This package mutates no durable state and keeps no cache:
// Guard resolution is lexical, OpenNoFollowFile opens read-only, and every
// other entry point is a pure function of its input. Crash and idempotency
// handling therefore have no surface here: there is no journal to recover
// and no second actor to collide with. Tombstone deletion scope (Section
// 16.3 "broad deletion from tombstones") has no implementation in this
// repository yet and is not modeled. The operator default for which
// environment names a provider plugin inherits is product policy with no
// in-repo provider implementation to derive needs from, so BuildEnv takes
// the allowlist explicitly and ExecRunner inherits the parent environment
// until the operator supplies one. Grapheme-width policing stops at the
// enumerated invisible controls: full East-Asian/emoji width measurement
// needs terminal width tables this package does not carry. Content-level
// secret scrubbing is best-effort by specification (Section 16.2 claims no
// reliable scrubbing), and Redact states exactly the three shapes it
// decides.
package secprim
