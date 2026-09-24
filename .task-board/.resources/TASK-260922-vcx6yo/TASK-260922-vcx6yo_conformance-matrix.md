# TASK-260922-vcx6yo Conformance Matrix

Status: **ready for review**. Run: `RUN-260923-2efad9`. The Story worktree
candidate is based on checkpoint `d8decbb03db419c8ad2707cadc2eafd3027a9c4d`
and remains uncommitted.

## Normative authority

Pinned file: `internal/specdoc/SPEC.v0.7.0.md`, §4.C attach operation row,
line 1205:

> `multi_attach` for overlap; `multiple_input_clients` plus AX policy for
> concurrent input

The quoted capabilities are separate requirements. Transport authorization
does not supply either capability, and `multiple_input_clients` does not grant
AX permission to send input.

## Acceptance criteria coverage

**5 of 5 AC rows driven** through the named production entry. The tests use
in-memory/scripted adapters and do not launch tmux; literal refusal tokens are
asserted from the spec vocabulary.

| AC | Production call site | Named test evidence and observed result |
|---|---|---|
| 1. A distinct overlapping client needs `multi_attach`; a second input client also needs `multiple_input_clients` and AX input policy. Both rev7 reviewer scenarios refuse with no second receipt and no vector. | `Lifecycle.Execute(attach)` → `executeAttach` → `checkAttachOverlap`; shared `terminalbackend.CheckAttachRequest` is called at request preflight (`ops.go:87`) and again at receipt commit (`termbind/attach.go:143`). | `TestAttachOverlapRequiresMultiAttach` covers a read-only requester; `TestAttachOverlapRequiresMultiAttachForInputClient` covers an input-authorized requester while `multiple_input_clients` is present. `TestAttachOverlapInputRequiresMultipleInputClients` covers one input peer and a read-only requester; `TestAttachOverlapInputGateChecksEveryPeer` covers a read-only peer sorting before one and several input-authorized peers. All refusals assert literal `terminal_backend_capability_unproven`, no candidate receipt, and no vector. `TestAttachConcurrentInputRequiresAXPolicy` asserts literal `terminal_backend_unauthorized`, no receipt, and no vector. Narrowings: `N-attach-overlap-multi`, `N-attach-overlap-multi-input-client`, `N-attach-overlap-input`, `N-M5-attach-overlap-input-first-peer-only`, `N-attach-ax-input-policy`; each is isolated in the harness. |
| 2. Overlap decision is atomic; validated same-client replay is distinct from a new client even when another peer exists. | `Lifecycle.Execute(attach)` → `executeAttach`; `AttachStore.AcquireAdmission` holds the per-instance lock across `checkAttachOverlap` and receipt install. | `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances` pauses an input-authorized receipt after staging; `TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances` repeats the paused interleaving with two read-only clients and no `multi_attach`. Both use independent Lifecycle/store values, assert one receipt/vector, and refuse the contender after the first commit. `N-attach-admission-lock` and reviewer M2 `N-M2-attach-admission-lock-input-only` are run alone. `TestAttachAdmissionLockSharedAcrossStores` and `TestAttachAdmissionLockReleasedAfterProcessExit` prove cross-store contention, cancellation, kernel release, and persistent-file reuse. `TestAttachSameClientRetryWithPeerPresent` pins input and read-only replay plus an input-conflict refusal; replay is proven by the durable receipt, not the client ID. |
| 3. Receipt liveness with no positive live-client evidence has an explicit rule; corrupt, unreadable, directory-shaped, misfiled, and foreign-keyed census entries fail closed using the landed census. | `Lifecycle.Execute(attach)` → `executeAttach` → `checkAttachOverlap` → `AttachStore.Peers` / `Lookup`. | `TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer` covers both a read-only peer without `multi_attach` and an input-authorized peer without `multiple_input_clients`; each prior returned vector has not been executed, and each new receipt/vector is absent. `TestAttachPeerDirectoryFailsClosed`, `TestAttachPeerFilenameMismatchFailsClosed`, `TestAttachOverlapPeerReadFailureFailsClosed`, and direct `AttachStore.Peers` corruption tests exercise the existing census. Narrowing `N-attach-liveness-unknown` retires the fixture receipt and is killed through `Lifecycle.Execute`. The census implementation is not forked. |
| 4. Every task-added gate is crossed with every relevant production entry; mirrored traceability has row-count symmetry. | The task-added semantic gates are reached only through `Lifecycle.Execute(attach)` / `executeAttach`; dispatches for the other lifecycle operations are enumerated below. | The gate × entry table below names one M cell and an admitting narrowing row for each task-added gate. `internal/tmuxserver/TRACEABILITY.md` mirrors the rows, cell counts, symmetry statement, and aggregate coverage ratio; `internal/termbind/TRACEABILITY.md` adds the admission lock and UNKNOWN peer rule. `go run ./internal/traceability/cmd/tracecheck` passed. |
| 5. Composition is compared by OUTCOME over the complete importer set; every moved input class is named; B44 is closed or restated. | Touched packages `internal/termbind` and `internal/tmuxserver`; closure driver calls package entries keyed by `(package, entry, input)`. | Base and candidate each produced 216 rows over 33 outcome-driven packages. All 216 keys match; no base-only or candidate-only keys. One outcome moved, listed under composition below. The complete 36-package importer set ran; 3 inherited package-surface bounds remain explicit. B44 is restated under liveness below. |

## Task-added gate × production-entry census

The complete `Lifecycle.Execute` operation dispatch is the entry set for this
leaf. Each other-operation cell is unreachable because dispatch selects that
operation's handler and does not call `executeAttach`, `checkAttachOverlap`, or
`AttachStore.AcquireAdmission`.

| Gate | attach | create | status | quiesce | wait-safe-boundary | request-stop | terminate-stale | restore |
|---|---|---|---|---|---|---|---|---|
| `overlapmulti` | **M** `TestAttachOverlapRequiresMultiAttach`, `TestAttachOverlapRequiresMultiAttachForInputClient` / `N-attach-overlap-multi`, `N-attach-overlap-multi-input-client` | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass |
| `overlapinput` | **M** `TestAttachOverlapInputRequiresMultipleInputClients`, `TestAttachOverlapInputGateChecksEveryPeer` / `N-attach-overlap-input`, `N-M5-attach-overlap-input-first-peer-only` | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass |
| `overlapreplay` | **M** `TestAttachSameClientRetryWithPeerPresent` / `N-attach-overlap-replay-unproven` | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass |
| `admissionlock` | **M** `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances`, `TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances` / `N-attach-admission-lock`, `N-M2-attach-admission-lock-input-only` | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass |
| `livenessunknown` | **M** `TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer` / `N-attach-liveness-unknown` | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass |
| `attachaxpolicy` | **M** `TestAttachConcurrentInputRequiresAXPolicy` / `N-attach-ax-input-policy` | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass | U1 dispatch bypass |

`M` means the named entry test drives the gate and the named narrowing mutant
was killed. `U1` means unreachable because that operation's dispatch handler
does not call the attach admission path. The scoped gate set has one reachable
attach cell per row: **6 gates × 1 reachable lifecycle entry = 6 M rows**;
the full matrix is 6 × 8 = 48 cells, with 42 U1 dispatch-bypass cells.
The AX policy rule calls the same shared `CheckAttachRequest` implementation
at request preflight and receipt commit within that one attach entry. Its
symmetry line is 2 helper call sites / 1 shared semantic gate / 1 reachable
lifecycle entry / 1 common-helper narrowing row. No task-added gate reaches a
second lifecycle-operation entry. `overlapmulti` and `overlapinput` were
introduced by the preceding attach leaf; this task reworked their axis evidence
and added the M2/M5 regressions. `peershape` remains the landed census contract
and is not duplicated here.

The admission gate has two OS implementations of the same semantic entry:
Unix `flock` and Windows `LockFileEx`. The shared `AcquireAdmission` contract
has one logical row. Unix process behavior is tested locally, including a
killed subprocess; Windows was cross-built and vetted but not run on a Windows
host. No source-text inspection gate was added, so a token-preserving
source-text mutant is not applicable.

## Reviewer rev1 axis enumeration

The five axes are client class, peer count, sorted peer position, replay versus
new client, and concurrent versus sequential calls. A cell is counted as
measured only when it names an entry test and an admitting narrowing; all other
cells state the bound and owner.

| Gate | Client class | Peer count | Peer position in client-ID order | Same-client replay vs new client | Concurrent vs sequential |
|---|---|---|---|---|---|
| `admissionlock` | `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances` / `N-attach-admission-lock` for input clients; `TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances` / `N-M2-attach-admission-lock-input-only` for read-only clients. | Both paused tests start at an empty census and re-census one committed peer; both named lock narrowings are isolated. | BOUND: lock key is instance ID and acquisition precedes peer sorting. Owner: `AttachStore.AcquireAdmission`. | BOUND: no simultaneous duplicate-replay test; replay/new requests use the same unconditional lock. Owner: `executeAttach`. | Both paused tests race independent Lifecycle/store values. `N-attach-admission-lock` and M2 run alone. |
| `overlapmulti` | `TestAttachOverlapRequiresMultiAttach` / `N-attach-overlap-multi` for read-only; `TestAttachOverlapRequiresMultiAttachForInputClient` / `N-attach-overlap-multi-input-client` for input-authorized. | `TestAttachOverlapRequiresMultiAttach` refuses one peer and a later third client; `N-attach-overlap-multi` admits the one-peer case. | BOUND: decision reads only whether `len(peers)>0`, not order or peer fields. Owner: `checkAttachOverlap`. | `TestAttachSameClientRetryWithPeerPresent` covers both validated replay classes; `N-attach-overlap-replay-unproven` admits a receiptless caller. | BOUND: no multi_attach-only concurrency plant; the empty-census race is assigned to `admissionlock`. Owner: `executeAttach` / `checkAttachOverlap`. |
| `overlapinput` | `TestAttachOverlapInputRequiresMultipleInputClients` tests allowed read-only and refused input requests; `N-attach-overlap-input` weakens the gate. | One, two, and three peers appear across the existing row and `TestAttachOverlapInputGateChecksEveryPeer`; `N-M5-attach-overlap-input-first-peer-only` admits the mixed-peer cases. | The M5 test sorts read-only A before input B and C; M5 is run alone and kills both subtests. | Peer-present valid read-only/input replay is in `TestAttachSameClientRetryWithPeerPresent`; the replay mutant and M5 distinguish replay from a new input client. | `TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances` races writable clients while the second lacks `multiple_input_clients`; `N-attach-admission-lock` admits from the stale empty census. Sequential gate decisions are killed by the input and M5 rows. |
| `attachaxpolicy` | `TestAttachConcurrentInputRequiresAXPolicy` refuses mismatched input authorization with literal `terminal_backend_unauthorized`; `N-attach-ax-input-policy` admits it. | BOUND: `CheckAttachRequest` takes no peer count. Owner: `terminalbackend.CheckAttachRequest`. | BOUND: neither shared-helper call site receives the peer list. Owner: `terminalbackend.CheckAttachRequest`. | `TestAttachSameClientRetryWithPeerPresent` tests valid policy on replay. BOUND: no invalid-policy replay narrowing row; receipt commit shares the binding check. Owner: `AttachStore.Attach`. | BOUND: no simultaneous AX-policy mutant; authorization is checked at entry and again at receipt commit after the wait. Owner: `CheckAttachRequest` / `AttachStore.Attach`. |
| `overlapreplay` | `TestAttachSameClientRetryWithPeerPresent` covers read-only and input replay; `N-attach-overlap-replay-unproven` admits a receiptless caller. | BOUND: one other peer is present; replay with more peers is not separately driven. Owner: `AttachStore.Lookup`. | BOUND: replay is proved by Lookup of the requester's receipt, not peer order. Owner: `checkAttachOverlap`. | Same-client replay is contrasted with new-client refusal in the named tests; isolated replay mutant kills bare-ID exemption. | BOUND: no concurrent duplicate replay; all requests acquire the same lock before Lookup. Owner: `executeAttach`. |
| `livenessunknown` | `TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer` tests read-only and input-authorized receipts whose vectors have not been executed; `N-attach-liveness-unknown` retires the receipt. | BOUND: one peer is the direct liveness witness; no status/retirement signal exists, so every valid receipt returned by `Peers` remains possible. Owner: future authoritative receipt/protocol owner (B44). | BOUND: liveness reads no order/live-state field; `Peers` sorts without dropping receipts. Owner: `AttachStore.Peers`. | BOUND: replay exclusion is Lookup validation, not peer liveness. Owner: `checkAttachOverlap` / `AttachStore.Peers`. | BOUND: no mutable liveness signal to race; census is serialized by `admissionlock`. Owner: `AttachStore.Peers`. |

Axis census: **15 of 30 cells measured by test plus narrowing; 15 of 30 are
explicit bounds**. Symmetry: `attachaxpolicy` has 2 shared-helper call sites / 1
semantic gate / 1 lifecycle entry / 1 narrowing; `admissionlock` has 2 OS
implementations (`flock`, `LockFileEx`) / 1 shared API / 1 lifecycle call site
/ 1 Unix runtime interleaving test. Windows is cross-built and vetted, not
runtime-tested on a Windows host.

## Atomicity and facts revalidated at the effect boundary

`executeAttach` checks immutable request facts, quiescence, deadline,
authorization, matching transport, and server attestation before potentially
blocking admission/barrier waits. After those waits it acquires the local
ordering mutex and the persistent per-instance OS lock, then rechecks
quiescence, deadline, and backend generation. While the OS lock is held it
reads the shared peer census, decides capability admission, and holds the lock
through receipt install and vector construction. The receipt commit rechecks
the live AX authorization and expiry without a wait between check and its
no-replace install; the result binding is checked before returning the vector.

The deterministic paused-stage test proves the relevant overlap interval:
the second request cannot census “no peer” while the first request has staged
but not yet installed its receipt. After the first commit, the second census
observes the peer and refuses. Attach-to-attach admission is atomic across
independent Lifecycle values and processes. Attach-versus-quiesce ordering
continues to use the process-local mutex; its cross-process bound B42 remains.

## Same-client replay

`checkAttachOverlap` validates that a same-client receipt exists and matches
the replay before exempting it from overlap capabilities. A bare client ID is
not sufficient. `TestAttachSameClientRetryWithPeerPresent` exercises another
peer in both input-authorized and read-only shapes, confirms byte-identical
receipt replay, and confirms a changed input request refuses with literal
`idempotency_mismatch` while preserving the peer receipt.

## Liveness rule and B44

The durable receipt proves that AX admitted a client claim. It does not prove
that a tmux client remains connected. `Lifecycle.Execute(attach)` returns a
vector for the caller to execute, and the current protocol exposes no positive
client-identity join or detach/retirement signal. When a receipt's client can
no longer be shown live, liveness is **UNKNOWN**; the receipt remains a
possible peer. This may conservatively block another client after the first
client detached or before the returned vector was executed. Unknown is never
treated as absence.

This closes B44's admission-safety gap: every possible overlapping client is
subject to the capability gates, same-client retries require a validated
receipt, and census read/shape/key failures remain fail-closed through the
landed `AttachStore.Peers` path. Exact current-client liveness and stale
receipt reclamation remain a stated bound owned by a future authoritative
protocol/receipt owner; this task does not invent a proxy signal.

## OUTCOME-keyed importer comparison

The mechanically derived complete importer closure contains 36 packages:
33 outcome-driven package labels in the accepted 216-row corpus plus three
explicit package-surface bounds (`cloneproject`, `crashgate`, `sshtransport`)
retained from TASK-260830-1c28dz rev10. All 36 package tests passed. No new
internal package imports were added; the new external platform imports are
`golang.org/x/sys/unix` and `golang.org/x/sys/windows`.

| Key `(package, entry, input)` | Base outcome at checkpoint | Candidate outcome | Classification |
|---|---|---|---|
| `tmuxserver | ExecuteAttachOverlapRace | two-writable-clients/no-multiple-input-clients` | First and second writable; both receipts committed; second staged before first commit; second vector returned. | First writable; second refused with literal `terminal_backend_capability_unproven at operation capability conditional`; first receipt remains, no second receipt, second is not staged before first commit, and no second vector. | **Only moved input class**; closes the rev7 admission defect. |

Base and candidate each have 216 outcomes and 216 shared keys; 0 base-only,
0 candidate-only, 1 moved. Inputs and outputs are preserved in `closure-base.jsonl`,
`closure-candidate.jsonl`, and `closure-diff.txt` in the evidence archive.

## Durable-write evidence and limits

The `.admission.lock` file is an empty, persistent coordination inode. It is
created idempotently with `O_CREATE|O_RDWR`, never unlinked (to avoid splitting
waiters across inodes), and its kernel lock is released on explicit release or
process exit. `TestAttachAdmissionLockSharedAcrossStores` and
`TestAttachAdmissionLockReleasedAfterProcessExit` exercise reuse after cancel
and process death. The existing receipt commit remains a no-replace install;
`TestAttachCrashChildSelfTerminates` proves a receipt survives SIGKILL after
install and an identical retry replays it. `TestAttachSameClientRetryWithPeerPresent`
proves replay leaves the receipt bytes unchanged.

## Evidence artifacts

- `TASK-260922-vcx6yo_results.md` — ratio, implementation, validation, bounds,
  process anomaly, and handoff state.
- `TASK-260922-vcx6yo_conformance-matrix.md` — this matrix.
- `TASK-260922-vcx6yo_producer-evidence.tar.gz` — tracked patch, untracked
  sources, mutation and validation logs, importer audit, and closure outputs.

## Review-rework validation

- All 45 Go packages passed the configured regular, race, and coverage suites.
  Each sweep used three non-overlapping groups of 15 packages; all nine group
  logs are under `logs/go-test-*-group-*-rev2.log`.
- All 36 packages in the importer closure passed in six bounded groups
  (`logs/importer-01.log` through `logs/importer-06.log`). The 216-key
  OUTCOME comparison is accepted from the prior candidate evidence because
  this review rework changed no production code; importer tests were rerun.
- The 304-row tmuxserver mutation pass ran in six bounded batches. It produced
  302 KILLED and 2 expected SURVIVED outcomes, with no harness mismatches.
  The two reviewer narrowings M2 and M5 were each also selected alone; raw
  outcomes and batch summaries are included in the archive.
- All 30 configured validation entries were exercised; the validation table
  and exact log filenames are in `TASK-260922-vcx6yo_results.md` and `logs/`.
