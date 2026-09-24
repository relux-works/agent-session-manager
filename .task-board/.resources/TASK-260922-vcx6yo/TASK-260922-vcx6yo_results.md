# TASK-260922-vcx6yo Results

**Status:** ready for review.  
**Run:** `RUN-260923-2efad9`.  
**Branch/checkpoint:** `task-board/story/STORY-260830-2t4g7i`,
`d8decbb03db419c8ad2707cadc2eafd3027a9c4d`.  
**Candidate:** uncommitted in the Story worktree; no commit, stage, rebase,
branch switch, or reset was performed.

## Acceptance coverage

**5 of 5 AC rows driven.** The named tests pass through `Lifecycle.Execute`
for attach admission, or are explicit bounds where the protocol lacks live
evidence.

| AC | Production call site | Evidence |
|---|---|---|
| 1. Gate overlapping and concurrent-input clients. | `Lifecycle.Execute(attach)` → `executeAttach` → `checkAttachOverlap`; AX authorization through `terminalbackend.CheckAttachRequest`. | `TestAttachOverlapRequiresMultiAttach` and `TestAttachOverlapRequiresMultiAttachForInputClient` cover read-only and input-authorized requesters. `TestAttachOverlapInputRequiresMultipleInputClients` and `TestAttachOverlapInputGateChecksEveryPeer` cover one, mixed, and multiple input-authorized peers. They assert literal `terminal_backend_capability_unproven`, no candidate receipt, and no vector. `TestAttachConcurrentInputRequiresAXPolicy` asserts literal `terminal_backend_unauthorized`, no receipt, and no vector. |
| 2. Make the overlap decision atomic and distinguish replay from a new client. | `executeAttach`; `AttachStore.AcquireAdmission`; `AttachStore.Lookup`. | Writable `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances` and read-only `TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances` pause after receipt staging and race independent Lifecycle/store values; each permits one receipt/vector and refuses the contender. `TestAttachAdmissionLockSharedAcrossStores` and `TestAttachAdmissionLockReleasedAfterProcessExit` cover independent stores, cancellation, process exit, and lock reuse. `TestAttachSameClientRetryWithPeerPresent` covers input, read-only, and conflicting retries beside a peer. |
| 3. State and apply the liveness rule while preserving the landed fail-closed census. | `executeAttach` → `checkAttachOverlap` → `AttachStore.Peers` / `Lookup`. | `TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer` proves UNKNOWN liveness is not absence for read-only and input-authorized receipts whose returned vectors have not been executed. Both refuse without a new receipt or vector. `TestAttachPeerDirectoryFailsClosed`, `TestAttachPeerFilenameMismatchFailsClosed`, `TestAttachOverlapPeerReadFailureFailsClosed`, and direct `AttachStore.Peers` refusal tests pin unreadable, corrupt, directory-shaped, misfiled, and foreign-keyed entries. The census is extended by its caller and not forked. |
| 4. Provide the complete gate × entry census and row-count symmetry. | The six scoped gates in `executeAttach`; the other `Lifecycle.Execute` handlers are listed as unreachable. | The matrix has six gates and all eight lifecycle operation columns: 6 attach cells are measured; 42 non-attach cells are unreachable by dispatch. `attachaxpolicy` symmetry: 2 shared helper calls / 1 gate / 1 lifecycle entry. Admission lock symmetry: 2 OS implementations / 1 shared API / 1 lifecycle call site. |
| 5. Prove composition over the complete importer set and close or restate B44. | Outcome-grid drivers keyed by `(package, entry, input)` over touched packages `termbind` and `tmuxserver`. | Base and candidate each have 216 rows with 216 shared keys, 0 base-only, 0 candidate-only, and 1 moved input class. The complete 36-package importer set ran: 33 outcome-driven packages plus 3 explicit inherited package-surface bounds (`cloneproject`, `crashgate`, `sshtransport`). The moved outcome is recorded below. B44 now states UNKNOWN-as-possible-peer for safe admission and assigns exact liveness/reclamation to a future authoritative protocol/receipt owner. |

## Rev1 reviewer findings and rework

| Finding | Resolution | Entry evidence | Isolated narrowing evidence |
|---|---|---|---|
| P1 / reviewer M2: atomicity had only an input-authorized witness. | Lock acquisition remains unconditional for both client classes. Added a paused, cross-Lifecycle interleaving with two read-only clients and no `multi_attach`; after the first receipt commits, the second refuses with no receipt and no vector. | `TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances` drives `Lifecycle.Execute(attach)` and observes the actual receipt census and vector. | `N-M2-attach-admission-lock-input-only` was selected alone; harness exit 0 / mutant test exit 1. Raw log shows error nil, second receipt present, two-peer census, and a second read-only vector. |
| P2 / reviewer M5: `multiple_input_clients` was observed only at `peers[0]`. | Added one later input peer and multiple later input peers, always with read-only A sorting first. Each rejected new input attach leaves the receipt census unchanged and returns no vector. | `TestAttachOverlapInputGateChecksEveryPeer` runs both sorted-position subtests through `Lifecycle.Execute(attach)`. | `N-M5-attach-overlap-input-first-peer-only` was selected alone; harness exit 0 / mutant test exit 1. Raw log shows both subtests returned a receipt and writable vector. |
| P3: lock census was measured by one writable-axis row. | Recounted 6 scoped gates × 8 lifecycle entries = 48 cells: 6 attach cells measured and 42 non-attach cells unreachable by dispatch. Recounted 6 gates × 5 axes = 30 cells: 15 test-and-narrowing witnessed, 15 explicit bounds with owners. | The gate × entry table is mirrored in `internal/tmuxserver/TRACEABILITY.md`; the following axis table lists all five axes per gate. | `N-M2` and `N-M5` are named, shipped harness rows and were attributed by selecting each one alone. Other gates retain their named narrowing rows in the same harness. |

## Gate axis enumeration

| Gate | Client class | Peer count | Peer position in client-ID order | Same-client replay vs new client | Concurrent vs sequential |
|---|---|---|---|---|---|
| `admissionlock` | `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances` / `N-attach-admission-lock` for input clients; `TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances` / `N-M2-attach-admission-lock-input-only` for read-only clients. | Both paused tests start at an empty census and re-census one committed peer; each has an isolated lock narrowing. | BOUND: instance ID is the lock key and lock acquisition precedes peer sorting. Owner: `AttachStore.AcquireAdmission`. | BOUND: no simultaneous duplicate-replay test; replay/new requests use the same unconditional lock. Owner: `executeAttach`. | Both paused tests race independent Lifecycle/store values; `N-attach-admission-lock` and M2 run alone. |
| `overlapmulti` | `TestAttachOverlapRequiresMultiAttach` / `N-attach-overlap-multi` (read-only); `TestAttachOverlapRequiresMultiAttachForInputClient` / `N-attach-overlap-multi-input-client` (input-authorized). | One-peer refusal and a later two-peer refusal are driven by `TestAttachOverlapRequiresMultiAttach`; its narrowing admits the one-peer case. | BOUND: gate checks only `len(peers)>0`, not order or peer attributes. Owner: `checkAttachOverlap`. | `TestAttachSameClientRetryWithPeerPresent` covers read-only/input replay; `N-attach-overlap-replay-unproven` admits a receiptless new client. | BOUND: no separate multi_attach-only concurrency mutant; the stale empty-census race is assigned to `admissionlock`. Owner: `executeAttach` / `checkAttachOverlap`. |
| `overlapinput` | `TestAttachOverlapInputRequiresMultipleInputClients` covers allowed read-only and refused input requests; `N-attach-overlap-input` weakens the gate. | One, two, and three peer censuses are exercised by the existing test and `TestAttachOverlapInputGateChecksEveryPeer`; `N-M5-attach-overlap-input-first-peer-only` admits the mixed-peer cases. | `TestAttachOverlapInputGateChecksEveryPeer` sorts read-only A before input B and C; M5 kills both subtests when run alone. | `TestAttachSameClientRetryWithPeerPresent` covers valid replay; replay and M5 narrowings distinguish validated replay from a new input client. | `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances` races writable clients while the second lacks `multiple_input_clients`; `N-attach-admission-lock` admits from the stale empty census. Sequential gate decisions are killed by the input/M5 narrowings. |
| `attachaxpolicy` | `TestAttachConcurrentInputRequiresAXPolicy` asserts literal `terminal_backend_unauthorized`; `N-attach-ax-input-policy` admits the mismatched input class. | BOUND: `CheckAttachRequest` consumes no peer count. Owner: `terminalbackend.CheckAttachRequest`. | BOUND: neither shared-helper call site receives peer position. Owner: `terminalbackend.CheckAttachRequest`. | `TestAttachSameClientRetryWithPeerPresent` covers valid policy on replay. BOUND: no invalid-policy replay narrowing row; receipt commit shares the binding check. Owner: `AttachStore.Attach`. | BOUND: no simultaneous AX-policy mutant; authorization is checked at entry and again during receipt commit after the wait. Owner: `CheckAttachRequest` / `AttachStore.Attach`. |
| `overlapreplay` | `TestAttachSameClientRetryWithPeerPresent` covers read-only and input replay; `N-attach-overlap-replay-unproven` admits a receiptless caller. | BOUND: one peer is present; more than one other peer is not separately driven. Owner: `AttachStore.Lookup`. | BOUND: replay is proved by Lookup of the requester's receipt, not peer order. Owner: `checkAttachOverlap`. | Same-client replay is contrasted with new-client refusal; the isolated replay narrowing kills the bare-ID exemption. | BOUND: no concurrent duplicate replay; all requests acquire the same lock before Lookup. Owner: `executeAttach`. |
| `livenessunknown` | `TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer` covers read-only and input-authorized receipts whose returned vectors were not executed; `N-attach-liveness-unknown` retires the fixture receipt. | BOUND: one peer is the direct liveness witness; no status/retirement signal exists, so all valid `Peers` receipts remain possible. Owner: future authoritative receipt/protocol owner (B44). | BOUND: liveness reads no position/live-state field; `Peers` sorts without dropping valid receipts. Owner: `AttachStore.Peers`. | BOUND: replay exclusion is Lookup validation, not peer liveness. Owner: `checkAttachOverlap` / `AttachStore.Peers`. | BOUND: no mutable liveness signal to race; the census is serialized by `admissionlock`. Owner: `AttachStore.Peers`. |

**Axis census: 15 of 30 cells are test-and-narrowing witnessed; 15 of 30 are
explicit bounds.** The same table is mirrored in the package traceability map.

## Implementation

- Added `AttachStore.AcquireAdmission`, using a persistent per-instance lock
  file. Unix locks with `flock`; Windows locks with `LockFileEx`. The file is
  intentionally not removed, because unlinking it could split contenders
  across different inodes. Context cancellation bounds the wait.
- `executeAttach` now holds the shared admission lock from before the receipt
  census through capability admission, durable receipt install, and vector
  construction. After waiting for the local barrier and OS lock it rechecks
  quiescence, deadline, and generation; the receipt commit rechecks live AX
  authorization and expiry without a wait before its no-replace install.
- Distinct possible peers require `multi_attach`. New concurrent input also
  requires `multiple_input_clients` and matching AX input policy. Validated
  same-client retries are exempt from overlap admission; conflicting retries
  still refuse with `idempotency_mismatch`.
- A valid receipt is positive evidence of an admitted client claim, not proof
  that a tmux client is connected. The returned vector is executed by the
  caller and the protocol currently has no positive client-identity or detach
  signal. If liveness cannot be shown, it is **UNKNOWN** and remains a
  possible peer. This can conservatively block a new attach after detach or
  before the prior vector was executed; no proxy signal is treated as proof of
  death.
- The existing receipt namespace implementation stays authoritative. Its
  unreadable, corrupt, directory-shaped, foreign, misfiled, and wrong-key
  entries continue to fail closed. The only cross-process ordering claim is
  attach-versus-attach. Attach-versus-quiesce remains process-local under
  stated bound B42.
- Updated `README.md`, both package `TRACEABILITY.md` files, the tmuxserver
  mutant count, and newest-first `LOGBOOK.md`.

## Outcome comparison

| Outcome key | Base at checkpoint | Candidate | Result |
|---|---|---|---|
| `tmuxserver | ExecuteAttachOverlapRace | two-writable-clients/no-multiple-input-clients` | First and second writable clients both succeed; both receipts persist; second stages before first commit; second vector is returned. | First remains writable; the second refuses with `terminal_backend_capability_unproven at operation capability conditional`; only first receipt persists; second is not staged before first commit; no second vector is returned. | The sole moved input class; closes the rev7 overlap defect. |

The complete closure result is 216 shared keys, 0 base-only, 0 candidate-only,
and 1 moved. The outputs and provenance are in the producer evidence archive.

## Mutation evidence

- The complete tmuxserver harness contains **304 unique rows**. Six bounded
  selector batches covered rows 1–60, 61–120, 121–180, 181–240, 241–300, and
  301–304; every batch exited 0. All 304 outcomes matched expectation: 297
  narrowing mutants and 5 supplementary mutants were KILLED, the shadowed
  supplementary row SURVIVED, and harmless `C-control` SURVIVED.
- Reviewer finding P1 / M2 is closed by
  `N-M2-attach-admission-lock-input-only` against
  `TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances`. The raw
  mutant log shows the weakened lock admits the second read-only attach,
  persists its receipt, and returns its read-only vector. P2 / M5 is closed by
  `N-M5-attach-overlap-input-first-peer-only` against both sorted-peer cases in
  `TestAttachOverlapInputGateChecksEveryPeer`; its raw log shows both later
  input clients persist receipts and receive vectors. Each reviewer mutant was
  selected alone before the complete batched pass and appears again in it.
- The additional `N-attach-overlap-multi-input-client` narrowing verifies that
  `multi_attach` also gates an input-authorized requester while
  `multiple_input_clients` is present. The AX-policy, UNKNOWN-liveness, and
  shared-admission-lock narrowing rows also KILLED at their named
  `Lifecycle.Execute(attach)` tests. Raw logs are under
  `logs/mutants-full-rev2/`; batch summaries are `logs/mutants-batch-01.log`
  through `logs/mutants-batch-06.log`.
- An initial unbounded aggregate invocation was interrupted with exit 130 after
  32 rows so the harness could be rerun within the headless shell limit. That
  partial invocation is preserved as `logs/mutants-full-rev2.log` and is not
  included in the 304-row count; the six complete batches reran every row.
- The prior `internal/termbind/mutant_harness.py` full-pass evidence is accepted
  from the attached candidate evidence: this review rework changed no
  termbind production code or mutant harness. Its comment-only control
  SURVIVED. `TestAttachAdmissionLockReleasedAfterProcessExit` and
  `TestAttachCrashChildSelfTerminates` cover kernel-lock and receipt
  crash/retry behavior.
- The earlier attach-barrier mutant anchor and AX mutant placement corrections
  are retained from the prior candidate evidence. No source-text inspection
  gate was added, so a token-preserving source-text mutant is not applicable.

## Validation

All **30** configured validation entries were exercised in this review rework.
`go list ./...` returned 45 packages. The three full-repository test sweeps
used three non-overlapping 15-package partitions to stay within the headless
shell limit; every package appeared once in each sweep, with the configured
flags preserved. All grouped calls returned exit 0:

- `go test ... -count=1 -v`, `go test ... -race -count=1 -timeout 25m`, and
  `go test ... -cover -count=1`. Logs are `go-test-all-group-01-rev2.log`
  through `-03-rev2.log`, with matching `go-test-race-group-*` and
  `go-test-cover-group-*` files.
- Formatting, `go build ./...`, full `go vet ./...`, Linux amd64 and Windows
  amd64 builds, and `GOOS=windows GOARCH=amd64 go vet ./...` all passed.
- All 17 configured fuzz targets completed `100x -parallel=1`; each has an
  individual `fuzz-*-rev2.log` file.
- `tracecheck`, `cataloggen -check`, tracked JSON parsing, `task-board
  validate`, and `git diff --check` passed. `task-board validate` exited 0 and
  reported 254 issues on unrelated task IDs; no result named this task or its
  Story.
- All 36 packages in the accepted importer closure passed again in six
  sequential package groups (`importer-01.log` through `importer-06.log`). The
  OUTCOME-keyed base/candidate comparison (216 shared keys, 0 base-only, 0
  candidate-only, 1 moved input class) is accepted from the attached prior
  candidate evidence because production code did not change during this
  reviewer rework; its closure inputs, outputs, and importer audit are included
  in the refreshed archive.
- The final focused overlap/lock/census tests and both touched-package tests
  passed. The complete 45-package regular, race, and coverage sweeps above were
  rerun after the test/harness/documentation rework.

Two earlier validation failures were corrected and rerun in the prior
candidate: Windows vet caught `os.File.Fd()`'s `uintptr` type at `LockFileEx`,
and native vet caught a test fixture copying a `Lifecycle` mutex. The Windows
wrapper now casts to `windows.Handle`; the fixture initializes a second
Lifecycle explicitly. Current native/Windows builds and vet pass.

## Process anomaly

An earlier path typo briefly created
`/Users/iv/Developer/ReluxWorks/agent-session-manager/internal/termbind/admission_test.go`
outside the Story worktree at `2026-09-23T05:25:21+0400`. It was removed
immediately; the removal timestamp was not captured. A final presence check
confirmed the path is absent. No task-board control file was directly edited.

## Evidence paths

The evidence archive contains command logs under `logs/`, the importer audit,
base/candidate closure outputs and diff, the candidate patch, untracked source
files, this results file, and the conformance matrix. The worktree itself
remains the source of the uncommitted candidate for the Story snapshot.
