# TASK-260830-21gygk — partial implementation and Stop-The-Line evidence

Run: RUN-260908-00dda9. Goal: GOAL-260908-72201b revision 1.
Resolved scope: TASK-260830-1r9wrr, TASK-260830-21gygk, TASK-260830-wbpf1v.
The exact board objective is preserved in goal-checkpoint.txt. Its terminal
alternative permits an evidenced Stop-The-Line boundary; this is that path,
not accepted implementation, a CR, or a role handoff. Full task scope and every
unresolved AC remain unchanged. No qualification answer or directive arrived.

## Preserved implementation

New `sessquery.Reader` drives real persisted `sessrepo` indexes and the accepted
`sessstate.Projector`: local exact NAME, allowlisted learned peer exact NAME,
UUID, not-found precedence; ASCII-fold ambiguity; deduplication of a replicated
identity; local list/status read summaries; deterministic bytewise ordering;
parked recovery inspection and missing-versus-failed-read distinction. The
retained `Repository.Resolve` now refuses unique non-exact case variants as §2.3
requires. A former positive test encoded the wrong broader behavior and was
corrected, without changing any accepted history.

No manual commit, branch switch, rebase, reset, clean, checkpoint, push, merge,
parent-goal update, sibling spawn or other-Story edit occurred. HEAD remains
`7208cc7427e1ebe95106d34f98214e2bbea4ae1c`; signature verified, exit 0.
Ten changed/new paths are enumerated with hashes in the source manifest.
The source archive and apply-checked patch preserve all work independently of
this headless session. They are partial source artifacts, not a forged CR.

## Acceptance audit

**2 of 5 AC rows driven in full at the library boundary; 3 of 5 partially driven.
0 of 5 rows accepted for public CLI delivery.** The unchanged denominator is
selector resolution, ambiguity, list summaries, status summaries, deterministic
sorting. `internal/sessquery/TRACEABILITY.md` names every entry, test and bound.
Tests are deliberately uncommitted in this managed Story; no committed-test or
accepted-handoff gate is asserted.

| AC row | Production entry and named test | Result |
| --- | --- | --- |
| NAME/UUID/qualified resolution | Reader.Resolve/Status; TestResolvePinnedPrecedence; Repository.Resolve/TestResolveLocalNameAndUUID | Partial: NAME/UUID tiers driven; qualified syntax/target/precedence pending. |
| Ambiguity | Reader.Resolve/Status; TestResolveExactNamesAndASCIICollisions, TestResolvePeerOrderAndReplicatedIdentity | Library row driven for local/cross-peer folded collisions; inputs trust upstream identity/configuration owners. |
| List summaries | Reader.List → Repository.ListSessions → Projector.Project; TestListStatusDerivedFactsAndStableOrder, TestListStatusCheckpointAndProjectionFailure | Partial: actual stored/derived facts, no supplemental observation or public-rendering claims. |
| Status summaries | Reader.Status/InspectLocal; same tests plus TestParkedReadRecoveryAndMissingVsMalformed, TestCreatingSummaryCannotClaimClosedCLIResult | Partial: healthy selectors and explicit local recovery inspection; closed CLI ownership gap remains real. |
| Sorting | Reader.List/Resolve; TestListStatusDerivedFactsAndStableOrder, TestResolvePeerOrderAndReplicatedIdentity | Library row driven: session ID order, deterministic peer-source tie order, duplicate allowlist/permutation stability. |

Read calls do not write durable state. The recovery driver interrupts a real
CreateSession after record installation, reads the parked diagnostics, invokes
the byte-identical retry through the repository owner and observes the healed
summary. Repeated list reads prove idempotent read output. It is not a new
mutation/recovery protocol. Projector/record admission remain single-owned.

Trust bounds: learned repositories and the configuration allowlist are upstream
inputs, not independently authenticated by Reader. No mesh fetch, authentication,
freshness, peer completeness, cross-peer lease union, atomic multi-session
snapshot, safe action planner or public ax invocation is proved. Logical index
eligibility is not process liveness. Source tie order is not ownership election.
No doctor, provider, platform, transport or terminal capability is advertised.

## Stop-The-Line: exact unresolved dependency and technical bound

The existing product question remains unanswered: supply the approved
qualified-selector contract, including syntax, whether it targets a source peer
or winning owner, and its precedence/ambiguity rules. v0.5.0 §2.3 and §14.1 do
not define it. The original TASK-260830-21gygk_stop-line.md was reread and retained
on the board; that question is already pending and is not asked again here.

Options remain: (1) supply an existing approved contract (recommended first),
(2) have the product/spec owner define the missing rule, or (3) explicitly defer
qualification through an authorized scope revision. Option 3 is a scope
reduction and was not selected. No guessed grammar, host-filter default or
qualified-string fixture was introduced. Tests containing @ or / merely probe
ordinary not-found matching, never an approved qualification refusal contract.

A separate concrete limit prevents claiming full public summaries from current
owners: an event-free valid Session Record projects empty owner, lease and local
role; CLI Result's closed SessionSummary requires a UUID owner, nonempty host
name, positive lease epoch, UUID lease and owner/replica role. The real
Reader.List + cliresult.New driver TestCreatingSummaryCannotClaimClosedCLIResult
reproduces the failure. Inventing values or adding a private wire variant would
contradict the contract. Host display names, checkpoint timestamp/age, workspace
state, provider/platform observations, process liveness and per-peer sync times
also require their authoritative observation integrations. Reader deliberately
preserves only facts current owners can establish. This is an explicit partial
summary bound, not evidence that implementing a renderer is inherently impossible.
The next implementation must obtain those owners' facts and an authoritative
representation/failure path for the unknown-owner case, then drive actual public
list/status output. This run does not expand into runtime-quiescence, upstream
176/177 or other Story ownership to fake those facts.

No unsuccessful implementation experiment forced a model workaround. The one
initial red test was an invalid event fixture, corrected from the canonical
payload. Full checks then passed; routine rework is not the stop reason.

## Scoped predecessor evidence

Authoritative current task reads show both prerequisite tasks `integrating` and
`integrationCheckpointed=true`; statuses were not regressed to manufacture the
literal to-review predicate. The attached/read reducer acceptance JSON identifies
CR-TASK-260830-1r9wrr-3 accepted by RUN-260907-364977 and prerequisite
CR-TASK-260830-wbpf1v-4 checkpointed. The repository rev4 verdict explicitly
records ACCEPTED by RUN-260907-fd2b95. These are accepted prerequisite history,
not a new review of this partial work. Their current scoped board projection and
stored acceptance documents are included in this packet.

## Validation and real exits

All validation processes ran directly, without tee or a pipe-chain status mask.
No process remains for deferred evidence collection. The full-repository tests
and coverage are this run's own executions, not inherited review claims. Later
test/documentation/comment changes were separately verified as listed; the
production behavior did not change after full-suite execution.

| Command | Real exit | Log and scope |
| --- | --- | --- |
| `go test ./internal/sessquery ./internal/sessrepo -count=1` | 1 | `focused-first.log` — Initial fixture incorrectly used extensions in session.created payload; corrected to canonical required members. |
| `go test ./internal/sessquery ./internal/sessrepo -count=1 -v` | 0 | `focused.log` — Initial green behavioral suite. |
| `go test ./internal/sessquery -count=1 -v` | 0 | `query-tests.log` — Added checkpoint and projection-failure entry driver. |
| `python3 internal/sessquery/testdata/mutate.py <evidence>/mutants` | 0 | `mutants/mutants.json` — All 11 valid behavioral mutants killed; raw mutant/control exits below. |
| `go test ./... -v` | 0 | `full-tests.log` — Entire repository; before the later CLI-boundary test and final documentation/comment edits, with the same product behavior. |
| `go test ./... -cover` | 0 | `full-coverage.log` — Entire repository, same scope/timing as full tests. sessquery 98.9%, sessrepo 87.6%. |
| `go build ./...` | 0 | `build.log` — Repository build. |
| `go vet ./...` | 0 | `vet.log` — Repository vet. |
| `go test ./internal/sessquery -count=1 -v` | 0 | `query-with-bound.log` — Added real CLI unknown-owner boundary test. |
| `go test ./internal/sessquery ./internal/sessrepo ./internal/sessstate -race -count=1` | 0 | `race.log` — All three changed/dependency packages; observed process exit 0. |
| `go test ./internal/cigate ./internal/traceability ./internal/specpin ./internal/specdoc -count=1` | 0 | `docs-and-pin.log` — Documentation/capability non-claims and normative pin after README additions. |
| `go test ./internal/sessquery -cover -count=1` | 0 | `query-coverage.log` — Current new test set, 98.9% statement coverage. |
| `go vet ./...` | 0 | `vet-after.log` — Re-run after new tests/documentation. |
| `go build ./...` | 0 | `build-after.log` — Re-run after new tests/documentation. |
| `git diff --check` | 0 | `diff-check.log` — Whitespace validation. |
| `gofmt -l <all changed Go files>; assert empty output` | 0 | `gofmt.log` — Formatter process exit 0 and empty output; assertion exit 0. |
| `git verify-commit HEAD` | 0 | `checkpoint-signature.log` — Preserved signed checkpoint; no new commit created. |
| `go test ./internal/sessquery ./internal/sessrepo -count=1 -v` | 0 | `focused-current.log` — Current source after last eligibility-comment clarification. |
| `GIT_CEILING_DIRECTORIES=<evidence> git apply --check <partial-source.patch>` | 0 | `patch-check.log` — Against an isolated git archive of preserved HEAD; no managed-tree mutation. |

Full logs include nested expected-red census plants printed by passing parent
tests. The outer full-suite exit is 0; their printed nested failures must not be
mistaken for an outer failing run or hidden. Windows runtime, remote transport,
public CLI and public summary observations were not executed; no claim is made.

## Mutant evidence

11 of 11 valid applied behavioral mutants are killed by named tests (9 narrowing,
2 ordering). Before/after controls exit 0. Each mutant go test exits 1. The compile
control and not-applied control have no named failing behavioral test and provide
no behavioral kill: they are reported as survivor/control bounds below. Mutation
of source does not count as proof unless the behavioral suite fails by name.
No source-text-inspection gate was added, so no static-token claim is substituted
for this behavioral battery.

| Mutant | Narrowing/change | Named failing test | Real exit / survival bound |
| --- | --- | --- | --- |
| control-before | Unmutated source | — | 0; green control, not a mutant kill |
| N-collision-pair | Allow exactly a two-identity folded bucket; larger buckets still refuse. | `TestResolveExactNamesAndASCIICollisions`, `TestResolveExactNamesAndASCIICollisions/local`, `TestResolveExactNamesAndASCIICollisions/peers`, `TestResolvePeerOrderAndReplicatedIdentity` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| N-case-variant | Allow the sole non-exact query az-09._ within its folded bucket. | `TestResolveExactNamesAndASCIICollisions`, `TestResolveExactNamesAndASCIICollisions/local`, `TestResolveExactNamesAndASCIICollisions/peers` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| N-parked-id | Allow one parked session identity into live resolution. | `TestParkedReadRecoveryAndMissingVsMalformed` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| N-tombstoned-id | Allow one tombstoned session identity into live resolution. | `TestResolveExcludesTombstonedButListsIt` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| N-unlisted-peer | Read one peer outside the allowlist; all other peer identities still filtered. | `TestResolveAllowlistAndReadFailures` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| N-duplicate-peer | Permit duplicate sources for one peer identity, retain duplicate refusal for other identities. | `TestResolveAllowlistAndReadFailures` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| N-read-failure-fallback | Treat repository-path read failure as empty, while retaining other read refusals. | `TestResolveAllowlistAndReadFailures` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| N-projection-failure-fallback | Omit one identity on failed projection instead of propagating its failure. | `TestListStatusCheckpointAndProjectionFailure` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| N-repository-case-variant | Allow one non-exact query at the retained repository Resolve entry. | `TestResolveLocalNameAndUUID` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| B-peer-order | Reverse peer-source tie ordering; behavioral determinism fixture must fail. | `TestResolvePeerOrderAndReplicatedIdentity` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| B-session-order | Reverse session output order; behavioral list fixture must fail. | `TestListStatusDerivedFactsAndStableOrder`, `TestParkedReadRecoveryAndMissingVsMalformed` | 1; Killed within the stated fixture class; no broader mutation completeness claim. |
| C-not-applied | Application control: missing patch is not a killed mutant. | None | not run; Survivor/control: behavior unproved; NOT_APPLIED. |
| C-compile-failure | Compilation control: no named failed behavioral test is not a kill. | None | 1; Survivor/control: behavior unproved; COMPILE_OR_HARNESS_FAILURE. |
| control-after | Unmutated source | — | 0; green control, not a mutant kill |

The initializer/nil-repository programmer-error guards are tested but have no
narrowing mutant in this packet. Neither all-gate completeness nor the task's
whole-scope mutation checklist is claimed. The explicit guards/table and 98.9%
statement coverage are bounded evidence, not a proof of every acceptance clause.

## Goal-turn classification

This goal turn made progress: product source and tests changed, actual checks
ran, and new scoped source/evidence artifacts were persisted. The previous run's
stop packet also yielded contract evidence, so this is not a no-progress retry.
The board's Stop-The-Line terminal alternative is the applicable disposition;
no scope was shrunk, no accepted implementation claimed, and no provider-limit
or ordinary failed check was used as a success condition.
