// Package provider implements the M0 trusted provider plugin host boundary
// of pinned specification Section 7.1: deterministic executable discovery,
// trust recording, and substitution detection.
//
// The actor every Section 7.1 RFC 2119 keyword governs here is the ax host
// (this repository). The provider plugin is a child process this package
// never builds, probes, or executes: Discover takes no probe or execution
// callback, so the Section 7.1 duplicate refusal structurally happens before
// any probe or execution. Later leaves own the wire protocol (Section 7.2),
// manifests and capabilities (Sections 7.3-7.7, 8), and process lifecycle.
//
// This package is an internal M0 host boundary, not a public stable plugin
// SDK: Section 7.1 states that M0 MUST NOT advertise a public stable plugin
// SDK, so no exported symbol offers plugin authorship, version negotiation,
// or compatibility promises. The package lives under internal/ and declares
// no stability contract.
//
// Discovery consumes only operator configuration (providers.plugin_dirs in
// listed order, built-in adapters, then PATH when allow_path_plugins is
// true) and the host filesystem. Everything crossing that boundary is
// treated as attacker-influenced: a failed or partial read is reported as a
// failure, never as an absence that a fallback could treat as satisfied.
//
// The package mutates no durable state of its own. Trust returns a
// TrustRecord value the host persists; Verify rechecks that receipt against
// freshly read filesystem facts. There is therefore no crash or idempotency
// surface inside this package: repeated Discover calls over an unchanged
// tree return identical candidates, and Verify is a pure recheck.
//
// A discovered candidate carries no availability, status, or capability
// claim. Discovery answers which executable serves a provider ID; whether
// any capability is available is decided by the probe plane (a later leaf),
// never advertised here.
//
// Wire composition contract: Error Detail deliberately names provider
// IDs, paths, and trust dimensions in human text, and Error() appends
// the filesystem cause verbatim for errors.Is and errors.As. That
// rendering must never become a wire message as-is:
// axerror.refuseCausalLeak refuses any message reproducing a local
// cause, so the future cmd/ax lift of a discovery failure into a
// Structured Error must rebuild the message (dynamic facts into the
// Details map, cause stripped) rather than quoting Error(). No such
// call site exists yet; this paragraph names the rule that lift must
// follow.
package provider
