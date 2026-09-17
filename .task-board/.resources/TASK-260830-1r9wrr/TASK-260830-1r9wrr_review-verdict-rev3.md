# TASK-260830-1r9wrr — revision 3 reviewer verdict

Verdict: **accepted**, for `CR-TASK-260830-1r9wrr-3`, revision 3.
Reviewer: `RUN-260907-364977`, Codex gpt-6-astra high.
repeat-of: none. Prior R2-F1 and R2-F2 are closed as detailed below.
Record with `accept_cr(TASK-260830-1r9wrr, revision=3, evidence=TASK-260830-1r9wrr_review-verdict-rev3.md)`; this routes to integrating, not done. No reviewer checkpoint, integration, commit_ack, product edit or manual commit.

## Authoritative goal and prerequisite

`task-board spawn goal RUN-260907-364977` was queried initially, at directive checkpoints, and before this verdict. The active goal is **GOAL-260907-410a58 revision 1**, kind/success predicate **reviewer_verdict/reviewer_verdict**. Its objective is to record exactly one evidence-backed review branch for the assigned scope. Requested item: TASK-260830-1r9wrr; resolved scope: TASK-260830-1r9wrr and TASK-260830-wbpf1v. Parent binding supplied by the assignment is GOAL-260907-d09cee revision 1; no parent/primary goal was modified.

The acknowledged directive `RUN-260907-364977:nudge:33d055` clarifies that wbpf1v is prerequisite acceptance evidence, not a new review assignment, and that accepted CRs remain integrating for the producer-side transaction. I inspected its `review-verdict-rev4.md` (accepted by RUN-260907-fd2b95), `checkpoint-rev4.md`, and current worktree record: CR-TASK-260830-wbpf1v-4 is checkpointed, tree `7ab337c849fb1aa532927327d07329ea8f97825b`, checkpoint `0d9d0ad5acee26fed7bff78b605a07ff5230242c`. That commit's tree and signature were independently verified. Task status is integrating; this is not a claim of Story landing.

The shell lacked TASK_BOARD_DIR. The mandated initial bare status attempt hit the stale checkout board and was refused; no status changed there. Subsequent board access uses the explicit authoritative `/Users/iv/Developer/ReluxWorks/agent-session-manager/.task-board` path. Authoritative reviewing transition succeeded.

## Immutable candidate

Current board CR is ready revision 3, repository_delta=present, base `0d9d0ad5acee26fed7bff78b605a07ff5230242c`, candidate `2303fc0dd5ecb51f59ab456256ff1ca3b6c2bbf6`. Materialized CR patch SHA-256 is `c5f78f31f2fd95d0b62e3b920133de23942a382fa7837735adc6276d8893302c`, matching the record.

Detached-index read-tree HEAD, add of tracked/untracked nonignored candidate files excluding the checkout board, and write-tree reproduced the candidate before and after probes. The real index SHA-256 stayed `cec9f4d7b0f0c48bdcdcb9fca7be77e1e55c8ca92074554da59438d5ac1471fb`; HEAD stayed at the checkpoint. Independent probes ran only in a git-archive copy under task scratch and restored copy bytes. Rev2-to-rev3 has seven changed files; runtime implementation bytes are unchanged. No sessrepo/provhost changes were introduced.

Exact CR paths (16), independently compared with the board and outcome:

```
LOGBOOK.md
README.md
internal/sessstate/arms_test.go
internal/sessstate/census_plants_test.go
internal/sessstate/census_state_test.go
internal/sessstate/census_test.go
internal/sessstate/decode.go
internal/sessstate/doc.go
internal/sessstate/fixtures_test.go
internal/sessstate/negative_test.go
internal/sessstate/project.go
internal/sessstate/project_test.go
internal/sessstate/purity_test.go
internal/sessstate/reduce_test.go
internal/sessstate/sessstate.go
internal/sessstate/winner_test.go
```

## Gate-first prerequisite: R2-F1 closed

Shape previously failing: **bypass path around the check; unclassifiable site treated as nothing to inventory**. `deriveEventTypeUses` now walks every production AST from invcore, selects every direct selector named Type, records its owning function/receiver and exact statement context, and requires both-directions equality with the ownership ledger. Extra files, copied scalar values, helper arguments, address-taking, closures, package bindings, writes, duplicates, orphan rows and unclassifiable contexts cannot disappear silently. `TestEventTypeUsesAreOwned` is a normal test included in both repository-wide reruns. Existing enum/event/refusal checks remain composed with it.

I ran the permanent live-control wrapper through the unfiltered full suite and coverage suite. All four plants compile and reach Reduce → apply → effect → reviewExtra → step. Their independently generated probes show the wrong running-to-idle transition, so they are live controls. The probe is installed after the shipped gates; it is not counted as delivered behavioral coverage.

| Control | Build | Behavior / census / audit exits | Result |
| --- | --- | --- | --- |
| PA direct selector | 0 | 0 / 1 / 0 | census kill: event handling and Type ownership |
| PB copied Type | 0 | 0 / 1 / 0 | census kill: Type ownership |
| PC helper argument | 0 | 0 / 1 / 0 | census kill: Type ownership |
| PD pointer neighbor | 0 | 0 / 1 / 0 | census kill: Type ownership |
| XN identity transform | 0 | 0 / 0 / 0 | intended neutral survivor |
| XK changed default with searched tokens retained | 0 | 1 / 0 / 0 | TestReduceTreatsUnknownV1TypeAsInert |
| Q1 independent alias + closure + copied return | 0 | 0 / 1 / 0 | TestEventTypeUsesAreOwned |
| Q2 independent diagnostic-string dispatch | 0 | 0 / 0 / 0 | declared residual survivor, below |

Within direct-access escape scope: **5 killed / 5 applied, all census-only; 0 / 5 behavioral kills** (four shipped controls plus independent Q1). Neutral, known-bad and residual controls are separate, not denominator padding. Neither NOT_APPLIED nor COMPILE_FAIL is counted as a kill.

I also reran the delivered G-N gate-instrument narrowing in a fresh copy: it permits only keys ending in `|kind := event.Type` while keeping all other ownership checks. It compiles; the composed behavioral/census/audit wrapper fails specifically at `TestCensusLiveEventOwnershipPlants/PB`, because the weakened census admits PB. The external Python process exits 0 for successfully observing the expected Go exit 1. Full logs preserve both statuses.

**Residual bound is real and accepted, not hidden.** README and the ledger explicitly limit this to direct-field-access ownership, excluding reflection/unsafe/serialization and interpretation of allowed diagnostic strings. Q2 dispatches on the existing diagnostic `name` prefix for a fresh unknown event; it adds no Type access, passes all three shipped layers, and my subsequent `TestReviewerUnknownNeighbor` shows idle instead of running. The baseline probe is green. This reproduces precisely the declared diagnostic residual. It does not refute the narrowed claim or show a defect in the unmutated candidate. Future changes routing authority through diagnostic strings need separate review; no whole-program arbitrary-Go proof is asserted.

The attached missing-gate prerequisite predates the Local-bound work and reports 52 behavioral tests at that checkpoint; the final candidate has 53 after adding the Local test. These are different measured checkpoints, not inconsistent simultaneous claims.

## Empty-chain Local: R2-F2 closed

`TestReduceEmptyChainLocalIsIgnored` drives **14 / 14 combinations** via Reduce: absent, lesser, equal, greater, malformed ID, epoch-only and ID-only Local, each with absent Union and a valid multi-entry selected Union. It checks the entire Projection against the no-Local baseline and requires the union's named unchanged-state conflict. All pass in my unfiltered full rerun.

The bound matches `applyLocalLease`: without an authoritative event/process lease, Local is ignored and not validated. Even with a union winner, state remains creating; owner/local role remain unknown. README and doc.go explicitly state this. Historical F7 closure correctly distinguishes its Union and Local halves.

## Acceptance coverage and preserved findings

**10 of 10 AC rows driven.** Checked before source review; all named tests remain present and ran in the full suite. This preserves the previously reviewed scope rather than treating seven rework files as a smaller task.

| AC | Production call site | Named driving test/evidence |
| --- | --- | --- |
| 1 All 11 states | Reduce + Projector.Project | TestReduceDerivesEverySessionState |
| 2 Winning epochs, tie, gap | Reduce → resolveWinner, Compare | TestWinnerResolutionClearWin, Tie, TieOrdersBytewise, Gap; DuplicateOffChainUnionKeepsConflict; RepeatedOffChainUnionStaysResolved |
| 3 Individually valid conflicts | chainFold.conflict + named rules | TestWinnerResolutionTie, LosingBranchPreserved, UnionSupersedesChain, PastEpochTie; TestProviderMismatchKeepsRecordPinned |
| 4 Current checkpoints | noteCheckpoint via event effect | TestProjectDerivesFullLifecycle |
| 5 Provider identity | effectProviderLaunched/Identified | TestProjectDerivesFullLifecycle |
| 6 Terminal binding v1/v4 | effectTerminalCreated/Resumed | TestProjectDerivesFullLifecycle; TestProjectDerivesV4TerminalBinding |
| 7 Parked reason and retry | Projector.Project + Reduce | TestProjectDerivesParkedBareDirectory, RecordWithoutChain, TornStore |
| 8 Negative/refusal paths | step, trackLease, continuity, decode | TestReduceRefusesUnlistedTransitions; StoppedWithNullCheckpoint; BootstrapAbortStopWithoutAbort; TaskBoardRepeatMismatch; LeaseLinkageMismatch; TestReduceEmptyChainValidatesUnion; TestReduceLocalLeaseTupleRule |
| 9 Crash, recovery, purity | sessrepo four secconftest create boundaries → Project; Reduce | TestProjectDerivesParkedBareDirectory, RecordWithoutChain, EventsDirBoundary; TestProjectRecoversPastChainBoundary; TestProjectIsIdempotentAcrossReads; TestReduceIsPure; TestReduceRefusesChainForbiddenReordering |
| 10 Capability honesty | no new command/doctor/capability claim | tracecheck; TestNoProductionPathAttestsProviderIdentityBinding |

Authoritative spec is the local byte-pinned v0.5.0 document at commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`; pin/document checks pass. Sections 5.2 unknown-type inert behavior and 5.7 states were inspected. Full name-choice/rendering, profiles, capability/workspace/mesh derivations remain existing explicitly stated sibling bounds, not new claims.

Accepted from prior attached review evidence without re-litigating: closed canonical enums, multiple predecessors, four create-window fault locations, the independent 64-call purity/slice-isolation probe, and F1/F2/F4/F5/F6 closures. Their committed tests ran again. This projector is read-only and owns no durable write window; crash tests drive sessrepo production creation and read/recovery through Project.

## Measurements and validation

Independent in-package invcore derivation reproduces: **69 refusal sites / 69 rows** (39 sessstate, 28 decode, 2 project), **11 states / 11 rows**, **24 events / 24 rows**, **4 Type uses / 4 ownership rows**. Parse-only unregistered/orphan/unclassifiable and alias controls run in the suite and are not misreported as compiling behavioral mutants.

| Rerun directly by this reviewer | Result and full log |
| --- | --- |
| go test ./... -v -count=1 | exit 0, 25 package result rows; full-test.log (5,142,281 bytes) |
| go test ./... -cover -count=1 | exit 0, 25 packages, sessstate 92.2%; coverage.log |
| go build ./... | exit 0; build.log |
| go vet ./... | exit 0; vet.log |
| tracecheck | exit 0; tracecheck.log |
| gofmt -l internal/sessstate; git diff --check | clean; format.log, diff-check.log |
| Focused race: purity, forbidden reordering, repeated Project reads | exit 0; race.log |
| G-N narrowing, composed live wrapper | expected Go exit 1 at PB; gate/G-N-composed.log; harness exit 0 |
| Independent Q1/Q2 probes and census measurements | expected outcomes above; independent-results.json and per-command logs |

I audited the attached producer `battery-results.json` against all named failures in its full per-layer logs: **28 / 28 selected kills = 22 behavioral + 3 census-only + 3 audit-only**, categories 14 narrowing + 8 arm-deletion + 3 census-only + 3 audit-only. All builds exit 0; XN and XK are excluded controls. I did **not** independently rerun that entire legacy 28-mutant roster this round. Its runtime source replacements are unchanged, prior reviewer rerun evidence exists, and current source-derived counts and controls were independently rerun. This selected roster is not claimed as one mutant per refusal site or exhaustive arbitrary-Go coverage.

Not independently rerun: full-repository race, cross-platform compile, fuzz, and unrelated configured CI contract selections. No green result is inferred for those. All direct logs are retained whole; nested expected-red control logs are not actual baseline suite failures. No background verification process remains.

Evidence: TASK-260830-1r9wrr_review-evidence-rev3.tar.gz, this verdict, and TASK-260830-1r9wrr_review-logbook-rev3.md. The archive includes commands, full logs, probe source, gate harness, identities, goal/directive snapshots and prerequisite acceptance/checkpoint evidence.
