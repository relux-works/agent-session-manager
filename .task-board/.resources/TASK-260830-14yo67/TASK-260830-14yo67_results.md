# TASK-260830-14yo67 — implement-workspace-checkpoint-records: results

Status: **ready for review** (developer → reviewer handoff).
Worktree: `.temp/STORY-260830-2rqigd/worktree`, candidate left
UNCOMMITTED. No CLI surface added; no commit on the Story branch.

Authority: `relux-works/agent-session-manager-spec@v0.6.0`
(commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`);
Sections 5.4, 10.5-10.6, 13.12-13.13 (headings retained from the
v0.5.0 scope citations).

## What was built

New package `internal/sessckpt` (production):

- `Store.Capture(chain, inputs)` — builds the typed checkpoint
  closure (provider identity, workspace manifest, task-board
  bundle, terminal evidence, source head) into a Checkpoint Record
  1.0.0, attests it through `sessrepo.AttestCheckpointRecord`,
  and installs it durably: content-addressed blob no-replace
  (`O_EXCL` + fsync + directory sync) first, operation receipt
  second. Same operation + identical inputs replays the recorded
  checkpoint without writing; same operation + moved inputs
  refuses `ErrCheckpointConflict` (`idempotency_mismatch`) and
  writes nothing.
- `Store.Admit(chain, raw, operationID, sessionKind)` — externally
  built records (e.g. received through sync) through the same
  attestation, variant, head-binding, and durability path.
- `Store.Get(checkpointID)` — byte-identical reads re-verified
  through the canonical owner on every read.
- Crash hooks `BeforeWrite` / `AfterBlob` / `AfterCommit` (nil in
  production), modeled on the `sessrepo` / TASK-260830-3qrfjp
  durable-write discipline.

Refusal taxonomy: `ErrInvalidCheckpoint` wraps
`canonicaljson.ErrInvalidIdentity` (`errors.Is` proves
`incompatible_schema` for CP-N1..CP-N4 on both layers); unknown
session/head propagate the `sessrepo` owner errors
(`ErrUnknownSession` / `ErrUnknownEvent`); moved-input retries
refuse `ErrCheckpointConflict`. Creator-holder binding is decided
by the landed `sessquery.admitCheckpoint` consumer, which the
tests drive on captured bytes (epoch-1 and successor positives
through `BuildPlan`/`Revalidate`, wrong-creator refusal with
`selector_observation_unavailable`).

## AC coverage: 8 of 8 rows driven through the production entries

| # | AC row | Call site | Tests |
| --- | --- | --- | --- |
| 1 | provider identity | `Capture` → `checkBoundary` + manifest leg | `TestCaptureDirectInstallsAttestedRecord`, `TestCaptureTaskBoardVariantInstalls`, `bad_provider_id`, `empty_provider_version` |
| 2 | workspace manifests | `Capture` → `checkDigestMember` | positives, `bad_workspace_manifest`, `TestCaptureMovedInputsChangeIdentity` |
| 3 | task-board bundle | `Capture`/`Admit` → `checkPersistenceVariant` | task-board positive, 5-arm variant table, `Admit(bad kind)` |
| 4 | terminal evidence | `Capture` → `checkBoundary` | 5-arm quiescence table incl. CP-N1, `bad_evidence` |
| 5 | source head | `Capture`/`Admit` → head binding vs `sessrepo` chain | 2 consumer positives, unknown/later-lease/malformed-head tests |
| 6 | exact fixtures + negatives | `Capture`, `Admit`, `Get`, `AttestCheckpointRecord` | spec example attests (digest pinned), CP-N1..CP-N4, malformed-frame/torn tests |
| 7 | crash/idempotency | `Store.install` + hooks | 3 hook tests, real-SIGKILL test, conflict/disagreement/replay tests |
| 8 | no unsupported capability | (no new surface) | stated bound: library only; README/doctor untouched, no CLI advertised |

Full clause map: `internal/sessckpt/TRACEABILITY.md` (also attached
as `TASK-260830-14yo67_conformance-matrix.md`).

## Validation (all on the final source unless noted)

- `go build ./...`, `go vet ./...`: clean.
- `go test ./... -count=1`: 27 packages ok (`full-test.log`).
- `go test ./internal/sessckpt/ -race -count=1`: ok
  (`sessckpt-race.log`); `-cover`: 81.8% (`sessckpt-cover.log`);
  verbose package log (`sessckpt-verbose.log`).
- Race for every other package: `race-rest.log` (all lighter
  packages), `race-sessrepo.log`, `race-canonicaljson.log`,
  `race-sessquery.log` — all ok.
- Full cover: `cover-full.log`.
- Narrowing mutants, shipped harness
  `internal/sessckpt/mutant_harness.py`: **7 of 7 killed**,
  1 harmless SURVIVED control, 0 NOT_APPLIED, 0 harness failures
  (`mutants/` per-plant raw logs + `summary.json`).
- `go run ./internal/traceability/cmd/tracecheck`: ok.
- `cataloggen -check`: ok.
- `task-board validate`: (see handoff transcript).
- `gofmt -l`, `git diff --check`: clean.

## Notable findings

1. Own negative test caught a real bug: `Admit` first skipped the
   session-kind check and admitted any kind. Fixed by routing
   `Admit` through `checkSessionKind` plus a `default` refusal in
   `checkPersistenceVariant` (`TestAdmitRefusesMalformedFrames/Admit(bad kind)`).
2. Race flake in the new SIGKILL test: after `syscall.Kill(self,
   SIGKILL)` the child fell through to `os.Exit(99)` before the
   kernel acted (once, under `-race`). Fixed by blocking forever
   after the kill; the parent carries a 60 s context timeout so a
   failed kill is a timeout failure, never a false pass. Stable
   across repeated race runs since.
3. The `section:5.4` ownership-registry gap ("creation ... not
   implemented") is now stale, but the registry projection digest
   is review-pinned against self-minted claims, so the registry
   text is deliberately untouched; the interim clause record is
   `internal/sessckpt/TRACEABILITY.md` bound 5. Registry refresh
   is left for review-gated maintenance.
4. Storage-level faults (ENOSPC, torn volume) are a stated bound:
   the install path follows the TASK-260830-3qrfjp model
   (no-replace + fsync + torn-read refusal, all driven), with no
   dedicated ENOSPC fault test in this package.

## Files

- `internal/sessckpt/doc.go`, `store.go`, `capture.go`
- `internal/sessckpt/capture_test.go`, `refusal_test.go`,
  `crash_test.go`, `crash_unix_test.go` (unix-only SIGKILL),
  `TRACEABILITY.md`, `mutant_harness.py`
- `LOGBOOK.md` (newest entry)
- Evidence pack: `TASK-260830-14yo67_producer-evidence.tar.gz`
  (test logs, mutant raw logs + summary, tracecheck/catalog logs)
