# TASK-260830-2056mm results — implement-terminal-binding-events-and-recovery

Final leaf of STORY-260830-ptxkqe (ax-pane-and-terminal-instance-binding, EPIC M1).
Candidate: UNCOMMITTED working tree on branch `task-board/story/STORY-260830-ptxkqe`
atop checkpoint ca1c1d9 (predecessors 1geqhj + kkh1an signed checkpoints).
No commit, rebase, or branch operation performed by this run.

Trunk: origin/main == c3aae73 at handoff (verified `git fetch origin main`;
merge-base == origin/main), so no `refresh-candidate` was needed.
`task-board.config.json` is byte-identical to HEAD (verified
`git diff HEAD --quiet -- task-board.config.json`). No validation command added.

Normative source: SPEC.v0.7.0 (v0.7.0 adoption on trunk at spawn).
Predecessor outcomes and the whole branch delta were read before starting;
the corrected 1geqhj rev2 authoring path (no `session.parked` under an
unheld lease, `fencing.AuthorizeMutation` composed) is built on, not forked.

## Deliverable

New `internal/termbind` composing the landed owners and the story's own
`axpane`/`terminstance` (never forked):

- (1) v4 payloads: `EmitTerminalCreated` / `EmitResumed` author the exact
  `terminal.created` and `session.resumed` payloads through the landed
  `sessrepo` append path under a locally held lease; v1-v3 registry untouched.
- (2) Evidence resolution: `ResolveEvidence` resolves each event's
  `evidence_ids` to exactly one Manifest, one Probe, and the Capability
  Evidence objects bound to the event tuple through landed
  `Registry.AdmitProbe`. Eleven foreign targets refuse (native reference,
  generation string, socket, pipe, endpoint, token, credential, terminal
  output, PID/handle, live-process fact, plus capability-echo); the binding
  digest is opaque — no Binding object, native_reference, or generation
  string is reachable through the event.
- (3) Identity refusal: `CheckInstanceIdentity` + `ParseTerminalBinding` +
  operation-body pre-scans refuse all eight forbidden forms
  (PID, handle, socket, path, pipe, URL, token, endpoint) with the pinned
  `terminal_backend_protocol_error` on the Binding, descriptor (adopted
  landed entry), operation bodies, bootstrap binding, and attach receipts;
  CLI Result 4 is a tripwire-pinned stated bound (unimplemented surface).
- (4) Lost-result recovery: `RecoverCreate` is read-only — one status read
  plus the durable (session_id, bootstrap_operation_id) binding proves
  absence or identifies the ONE child; unprovable shapes yield `unavailable`
  with `status_first`, never a second child, never a false absence claim.
  Window derives from the fold newest (`sessstate.Reduce`), never from the
  vacuous epoch-1 lease checkpoint field.
- (5) Skew and authority: takeover selects an admitted backend with a NEW
  digest and event (append-only, no fork); attach emits NEITHER event and
  changes no Owner/Replica, lease, or fencing state; v1 readers retain v4
  as inert immutable history and derive nothing.

## Coverage ratio (measured, not planned)

**60 of 60 AC rows driven** through production entries by named committed
tests — matrix in `internal/termbind/TRACEABILITY.md`, reproduced with
spec-clause mapping in `TASK-260830-2056mm_conformance-matrix.md`.
Killer-presence check: 21 distinct killer tests referenced by the harness,
all present (`go test -list`), none missing.

## Mutants (shipped harness, per-plant raw logs)

`PYTHONDONTWRITEBYTECODE=1 python3 internal/termbind/mutant_harness.py` —
30 rows: 28 narrowing + 1 supplementary arm-delete (D-emit-noresolve,
labeled) + 1 harmless SURVIVED control. Pass 1 and pass 2: **30/30 each**,
plus two full plain-list runs, exit 0. Per-row verbose logs (raw go output
+ subprocess exit) and verdict lists are in the evidence tarball;
production blobs sha256-identical before/after every run (diff clean).
No token-preserving source-text mutant applies: no gate inspects source
text (stated bound, harness executes the behavioral suite).
Extended batteries re-run in this session: axpane 56/56 twice (3 new
story-close rows verbose ×2), terminstance 122/122 twice (6 new rows
verbose ×2), blobs restored.

## Crash / idempotency

Real SIGKILL seams (unix): `TestAttachCrashChildSelfTerminates`,
`TestRecoverCreateAfterKillRecoversChild`; append-hook abort/replay
`TestEmitAppendHookSeams`; byte-identical retry replay
`TestEmitAppendsIdempotently`, `TestAttachIdenticalRetryReplays`.

## Story-close items

- Registry: `section:4.1` bound NEW (sliver 1/5) via `storyNewV070Bindings`
  with literal matching + clause remeasurement; 4.B/4.D/5.2 upgraded
  (slivers), 7.A upgraded (partial); seven new acceptance cases. NOTE: the
  producer's "§13.1" recovery rule lives at §4.1 lines 1395/1399 in the
  pinned v0.7.0 document; §13.1's eight measured clauses are
  launch-plan/boot-direct/argv rules owned by another story, so
  `section:13.1` is untouched.
- Digest re-pinned by derivation (transient in-package probe calling
  `decodeOwnershipRegistry` + `json.Marshal` + sha256, removed after):
  `DERIVED-DIGEST: 89944b8c6ec75703c0e07432890182ebbb9bb405f28502b05a6632662cebee9b
  (canonical bytes: 166083)` — equals the pin in `traceability.go`.
- `go run ./internal/traceability/cmd/tracecheck` GREEN:
  `traceability ok: contracts=64 normative_sections=36 acceptance_cases=147
  fixtures=33 compatibility_contracts=55 assigned_scopes=0`
  `section coverage: bindings=69 full=2 partial=9 sliver=9 unevidenced=45
  unmeasured=4 unowned=7 clauses_discharged=63/574`
- Section-scoped measured ratios (quoted verbatim; exit 1 by design —
  assigned-scope admission requires full, these bindings honestly carry
  sliver/partial gap text): 4.1 → 1/5 sliver; 4.B → 1/12 sliver;
  4.D → 1/3 sliver; 5.2 → 3/18 sliver; 7.A → 1/2 partial.
- README: `termbind` leaf section with exact test commands, no CLI/capability
  claims; "Measured coverage of this repository" updated to the printed
  numbers (pin test green).
- LOGBOOK: one newest-first story entry for TASK-260830-2056mm.
- Inherited residuals CLOSED: axpane P3-1/P3-4/P3-5, terminstance P3-1..P3-4
  (witnesses verbatim, mirror rows); P3-2/P3-3 stated as owner bounds.

## Validation suite (30 configured commands)

`cmd01-cmd30.log` (cmd05 split a–h) — all exit 0, in the evidence tarball.
Accepted from already-attached logs: cmd01–cmd30. Reran myself in this
session: full `tracecheck`, all five section-scoped runs, `go test
./internal/termbind/`, all three mutant harnesses twice each with per-row
verbose logs, the killer-presence ratio check, and the digest derivation.
`task-board validate` exits 0 (reports pre-existing missing-activity issues
on other elements, none on this task).

## Rejection-pattern self-check

(a) Literals asserted from spec text at entries (pinned codes, digests,
bounds), not production constants. (b) Every rule driven through the entry
the matrix names; dual Build/Decode sides each carry narrowing rows.
(c) Landed owners composed (scalar UUIDv7, canonicaljson closed shapes,
sessrepo/sessstate/sessquery, Registry.AdmitProbe, sessprofile) — only
package-specific rules are new. (d) Inputs driven, effects asserted; no
census stands in for behavior. (e) 28 narrowing rows + labeled arm-delete +
applied SURVIVED control, one raw log per plant per run. (f) Every
"composes/never" sentence is backed by a failing-without test or stated as
an explicit bound (attach no-emit absence-by-construction, CLI Result 4
tripwire, lease-authority adoption rows).
