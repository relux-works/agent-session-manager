// Package crashgate executes the Section 13.13 crash/restart outcome gate
// over the landed checkpoint and journal owners
// (STORY-260830-2rqigd, TASK-260830-17ootk).
//
// Authority: relux-works/agent-session-manager-spec@v0.6.0, Section
// 13.13 (crash/restart outcome gate) and Section 13.12 (failure
// matrix). The task scope cites the same section numbers of v0.5.0;
// the headings are retained in v0.6.0.
//
// The package owns the boundary registry: every crash boundary ID of
// the Section 13.13 registry table plus the CR-CLONE prose range,
// each classified per direct/task-board path as reachable through
// the landed owners or NOT APPLICABLE with the exact owning
// flow/package named. The enumeration is verified against the pinned
// specification by a derivation test (no hand-typed boundary list);
// the classification is explicit harness data, reviewed with the
// conformance matrix in TRACEABILITY.md.
//
// The package owns no recovery behavior and changes none. Every
// reachable row is driven through the landed production entries:
//
//   - internal/sessckpt Store.Capture for checkpoint-capture
//     boundaries (blob-then-receipt install, receipt replay);
//     recovery is the idempotent retry converging on the recorded
//     result (safe_retry by construction; capture is append-only
//     and has no rollback or parked semantics).
//   - internal/matjournal Store.Create and its journal updates for
//     journal boundaries, with Store.Recover as the recovery
//     evaluator classifying exactly one of safe_retry,
//     explicit_rollback, or recoverable_parked_state.
//
// Reused owners: internal/sessrepo (chain reads, checkpoint
// attestation), internal/secconftest (crash-injection vocabulary),
// internal/specdoc (pinned text; test binaries only), and the
// scalar/canonical/axerror grammar owners. Product code never reads
// the specification text.
//
// Stated bounds: boundaries whose durable write belongs to an
// unlanded owner (lease arbitration beyond sessrepo persistence,
// terminal creation, provider process start and native discovery,
// bridge launch/export/adopt execution, sync transfer, lifecycle
// orchestration, the clone flow) are recorded NOT APPLICABLE, never
// silently skipped. Status probes stay modeled inputs, as in
// matjournal. The package adds no ax command, no doctor result, and
// no runtime capability claim.
package crashgate
