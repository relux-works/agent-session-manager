// Package termbind writes versioned terminal binding events, recovers
// lost create results by bootstrap operation ID, and refuses PID/endpoint
// identity. It is the final leaf of the ax-pane-and-terminal-instance-binding
// story.
//
// Normative scope: AX v0.7.0 Section 4.B (Terminal Instance Binding 1.0.0
// closed object and the identity rule), Section 5.2 Session Event 4.0.0
// Terminal Instance events (terminal.created and session.resumed v4
// payloads), Section 4.D (evidence binding), Section 7.A (Provider
// Protocol 3 descriptor identity, cited landed), and Section 13.1
// (bootstrap idempotency and lost-result recovery).
//
// Division of authority (composed, never forked): v4 payload construction
// and lease-authorized emission come from internal/axpane
// (TerminalCreatedPayload, ResumedPayload, Emit); the durable bootstrap
// binding comes from internal/axpane (Store.Bind/Lookup/Status); status
// observation comes from internal/terminstance (Engine.ExecuteStatus);
// backend admission comes from internal/terminalbackend
// (Registry.AdmitProbe, ParseManifest, ParseProbe, ParseEvidence,
// AdmitProviderDescriptor, GenerationDigest); event durability comes from
// internal/sessrepo (AppendEvent); the newest-checkpoint window and the
// lifecycle fold come from internal/sessstate (Reduce); UUIDv7, digest,
// and timestamp grammars come from internal/scalar; strict object frames
// come from internal/environ (DecodeStrictObject); closed event shapes
// come from internal/canonicaljson at the append boundary. What this
// package owns is the composition the earlier leaves deliberately left
// out: the Terminal Instance Binding 1.0.0 parser (no parser exists on
// trunk), the coded terminal-instance identity gate and its operation-body
// pre-scans, evidence-ID resolution to admitted Manifest/Probe/Evidence
// bound to the event tuple, the v4 emission entries, lost-create recovery,
// and the durable attach-client receipt store.
//
// Bounds: no `ax` CLI surface, no tmux/ConPTY process control, no provider
// process, and no mesh transport live here. CLI Result 4.0.0 is registered
// but unimplemented in internal/cliresult, so the Result 4 identity
// surface is a stated bound pinned by a tripwire test, not a gate.
// Signature verification is caller-supplied through
// terminalbackend.SignatureVerifier. The bootstrap window bit is derived
// by the caller from the sessstate fold's newest checkpoint (the axpane
// precedent for caller-derived facts); recovery tests close the loop by
// folding a real chain.
package termbind
