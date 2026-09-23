# `internal/sessckpt` traceability (TASK-260830-14yo67)

Authority: `relux-works/agent-session-manager-spec@v0.6.0`
(`internal/specpin/v0.6.0.lock.json`, embedded
`internal/specdoc/SPEC.v0.6.0.md`). The task scope cites the same
section numbers of v0.5.0; the headings are retained in v0.6.0, so
every row below names the pinned v0.6.0 heading. Historical v0.5.0
behavior is preserved: the closed Checkpoint Record 1.0.0 member
set is unchanged between the two revisions.

Production entry points: `Store.Capture`, `Store.Admit`,
`Store.Get`, and `Store.EventHeads` (`internal/sessckpt/capture.go`);
durable install (`internal/sessckpt/store.go`). No CLI surface is
offered; the `sessckpt` package is a library only.

## AC coverage (9 of 9 rows driven through the production entries)

| # | AC row | Production call site | Named tests |
| --- | --- | --- | --- |
| 1 | Closure over provider identity | `Capture` → `checkBoundary` (provider_id/version) + `provider_manifest_id` leg | `TestCaptureDirectInstallsAttestedRecord`, `TestCaptureTaskBoardVariantInstalls`, `TestCaptureRefusesMalformedClosureMembers/bad_provider_id`, `/empty_provider_version` |
| 2 | Closure over workspace manifests | `Capture` → `checkDigestMember(workspace_manifest_id)` | positives above, `TestCaptureRefusesMalformedClosureMembers/bad_workspace_manifest`, `TestCaptureMovedInputsChangeIdentity` |
| 3 | Closure over task-board bundle | `Capture`/`Admit` → `checkPersistenceVariant` | `TestCaptureTaskBoardVariantInstalls`, `TestCaptureRefusesBadPersistenceVariant` (all five arms), `TestAdmitRefusesMalformedFrames/Admit(bad kind)` |
| 4 | Closure over terminal evidence | `Capture` → `checkBoundary` (evidence enum + quiescence publication rule) | `TestCaptureRefusesNonQuiescentBoundary` (all five arms incl. CP-N1), `TestCaptureRefusesMalformedClosureMembers/bad_evidence` |
| 5 | Closure over source head | `Capture`/`Admit` → `checkHeadBinding`/`checkRawHeadBinding` against `sessrepo.Repository.ListEvents` | `TestCaptureAdmitsThroughWinningLeaseConsumer`, `TestCaptureSuccessorAdmitsThroughConsumer`, `TestCaptureRefusesUnknownSessionAndHead`, `TestCaptureRefusesLaterLeaseHead`, `TestCaptureRefusesMalformedHeads` |
| 6 | Exact contract fixtures and negative/refusal cases | `Capture`, `Admit`, `Get` + `sessrepo.AttestCheckpointRecord` | `TestSpecExampleAttestsThroughConsumerOwner`, `TestCaptureDirectInstallsAttestedRecord` (closed member set equals the normative example), CP-N1 (`cp_n1_background_idle_false`), CP-N2 (`cp_n2_direct_null_provider`), CP-N3 (`cp_n3_both_present`), CP-N4 (`TestAdmitRefusesUnknownSafeBoundaryMember` with benign re-identified control), `TestAdmitRefusesMalformedFrames`, `TestGetRefusesUnknownAndTorn` |
| 7 | Crash/idempotency evidence for durable capture | `Store.install` (blob-first, receipt-second, no-replace) + `BeforeWrite`/`AfterBlob`/`AfterCommit` hooks | `TestCrashBeforeDurableWriteIsSafeRetry`, `TestCrashBetweenBlobAndReceiptResumes`, `TestCrashAfterReceiptReplaysRecordedResult`, `TestCaptureCrashChildSelfTerminates` (real SIGKILL), `TestCaptureWithMovedInputsRefuses`, `TestCaptureRefusesDisagreeingDigestPath`, `TestCaptureReplayIsIdempotent`, `TestCaptureIdenticalClosuresShareIdentity` |
| 8 | No unsupported capability advertised | no `cmd/` surface, no provider/task-board I/O in the package | Stated bound (see below); `gofmt`/`go vet`/`go build` clean, README/doctor untouched by this change |
| 9 | Re-attest the event heads used by profile derivation | `Store.EventHeads` → `Get` + `extractAdmitted` + `checkRawHeadBinding` against `sessrepo.Repository` | `TestEventHeadsReattestsCheckpointClosureForSession` |

## Pinned-clause map

### Section 5.4 Checkpoint Record

| Clause | Owner / call site | Evidence |
| --- | --- | --- |
| Closed 16-member shape, `schema`/`schema_version` exact, `status` = `validated`, `subject_id` = `session_id` | `canonicaljson.validateCheckpointRecord` via `sessrepo.AttestCheckpointRecord`; `buildCandidate` assembles exactly the closed set | member-set equality vs the normative example in `TestCaptureDirectInstallsAttestedRecord`; `TestSpecExampleAttestsThroughConsumerOwner` |
| `checkpoint_id` canonical omit-self digest | `canonicaljson.CalculateObjectIdentity` + `AttestCheckpointRecord`, drift-checked in `identifyCandidate` | every positive test attests; `TestAdmitRefusesMalformedFrames/Admit(tampered)` |
| Safe Boundary Evidence closed shape + publication rule (all idle true, counters zero) | `checkBoundary` first, owner backstop second | quiescence table; narrowing mutant `N-bg-idle` killed by `cp_n1_background_idle_false` |
| Persistence variant: exactly one leg; direct↔provider, task_board↔bundle | `checkPersistenceVariant` (+ `checkSessionKind`) first, owner presence backstop second | variant table; mutants `N-both-present`, `N-swapped-direct` killed |
| `event_heads` 1..64 sorted unique, resolving at or before the owning lease | `checkHeads` + `checkHeadBinding`/`checkRawHeadBinding` first, owner shape backstop second | heads table + unknown/later-lease tests; mutant `N-duplicate-heads`, `N-later-epoch-head` killed |
| CP-N1..CP-N4 rejected with `incompatible_schema` before publication | `Capture`/`Admit` refuse `ErrInvalidCheckpoint` wrapping `canonicaljson.ErrInvalidIdentity` (`errors.Is` proves both layers) | rows above; mutant `N-evidence-enum` (token-preserving) killed by `bad_evidence` |
| Normative example | attests through the same owner | `TestSpecExampleAttestsThroughConsumerOwner` (digest pinned) |
| Owning lease tuple, creator-holder, variant, head closure at admission | `sessquery.admitCheckpoint` (landed consumer, not re-implemented) | epoch-1 + successor `BuildPlan`/`Revalidate` positives; wrong-creator refusal (`selector_observation_unavailable`) |
| Force-takeover newest-validated-checkpoint selection | consumer behavior, out of capture scope | Stated bound: capture mints and installs; selection stays with the takeover owner |

### Sections 10.5-10.6 Materialization Plan / Journal

The checkpoint contributes its closure digests (`workspace_manifest_id`,
`provider_manifest_id`/`task_board_bundle_id`, checkpoint digest,
lease tuple) as plan source inputs; the plan/journal state machines
are owned by sibling TASK-260830-3k3e6m. Capture binds the exact
digest strings the plan rows consume and refuses to publish a
closure no plan row could select (variant gate). No plan or journal
bytes are written here: stated bound.

### Sections 13.12-13.13 Failure matrix / crash-outcome gate

| Applicable row | Capture behavior | Evidence |
| --- | --- | --- |
| Invalid digest/schema | refused before any durable byte (`ErrInvalidCheckpoint`) | all negative tests assert zero writes only where counted (`TestCrashBeforeDurableWriteIsSafeRetry` counts 0/0) |
| Operator interrupt / crash before first byte | nothing mutated; identical retry safe (`safe_retry`) | injector `PointPrepareEnter` test |
| Crash between blob and receipt | complete verifying blob, no receipt; identical retry completes (`safe_retry`) | hook test + real SIGKILL test (`TestCaptureCrashChildSelfTerminates`) |
| Crash after receipt | recorded result replayed, no second identity (`safe_retry` via receipt) | hook test |
| Retry with moved inputs | refuses `ErrCheckpointConflict` (`idempotency_mismatch`), writes nothing | `TestCaptureWithMovedInputsRefuses` + narrowing mutant `N-idempotency-session` killed |
| Disk-full / rename-blocked / permission faults | install errors propagate; no-replace keeps existing blobs; torn bytes refuse attestation on read | Design property on the TASK-260830-3qrfjp durable-write model; torn-read path driven by `TestGetRefusesUnknownAndTorn` and `TestCaptureRefusesDisagreeingDigestPath`. No dedicated ENOSPC fault test in this package: stated bound (storage faults are owned with their fault model by `internal/localstore` + TASK-260830-3qrfjp) |
| Two live owners for one session | out of capture scope (lease arbitration) | Stated bound |

## Negative-evidence proofs

Shipped harness: `internal/sessckpt/mutant_harness.py`. Run:

    PYTHONDONTWRITEBYTECODE=1 python3 internal/sessckpt/mutant_harness.py --log-dir .temp/TASK-260830-14yo67/mutants

Result on the delivered source: **7 of 7 narrowing mutants killed**,
**1 of 1 harmless SURVIVED controls applied**, 0 NOT_APPLIED, 0
compile/harness failures. Per-plant raw logs (command, return code,
full output) live under the log dir with `summary.json`; the same
logs ship in `TASK-260830-14yo67_producer-evidence.tar.gz`.

| Plant | Weakening (admits exactly the named member) | Killer |
| --- | --- | --- |
| `N-bg-idle` | quiescence gate passes `background_idle=false` when no process is open | `TestCaptureRefusesNonQuiescentBoundary/cp_n1_background_idle_false` (own-gate message attribution; the owner backstop still refuses with different text) |
| `N-both-present` | presence gate passes both-legs-present | `TestCaptureRefusesBadPersistenceVariant/cp_n3_both_present` (the kind gate then fires with different attribution) |
| `N-swapped-direct` | direct arm passes bundle-only | `TestCaptureRefusesBadPersistenceVariant/swapped_direct_takes_bundle` (clean admission kill: capture succeeds) |
| `N-duplicate-heads` | order gate passes duplicates (`<=` → `<`) | `TestCaptureRefusesMalformedHeads/duplicated` (own-gate `sorted unique` attribution; owner backstop text differs) |
| `N-later-epoch-head` | epoch arm admits heads one epoch past the owner (`>` → `>+1`) | `TestCaptureRefusesLaterLeaseHead` (clean admission kill) |
| `N-idempotency-session` | conflict gate passes moved inputs under the same session | `TestCaptureWithMovedInputsRefuses` (clean admission kill: retry succeeds) |
| `N-evidence-enum` | closed enum admits exactly `provider_gossip`; every searched-for token preserved | `TestCaptureRefusesMalformedClosureMembers/bad_evidence` (own-gate `safe_boundary` attribution) |
| `C-harmless-comment` | comment-only edit | `TestCaptureDirectInstallsAttestedRecord` still passes (SURVIVED) |

## Stated bounds (not driven, declared)

1. Raw `Admit` trusts the supplier for lease-tuple/creator truth the
   way any sync-installed record does; the consumer re-decides
   creator-holder at admission. Chain binding is still enforced
   (nil chain refuses closed).
2. Capture performs no provider, task-board, or terminal I/O: the
   quiescence facts are caller-declared and the publication rule is
   enforced on the declaration. Live observation belongs to the
   provider/terminal owners.
3. Manifest bytes are referenced by digest only; manifest resolution
   and Transfer Manifest validation belong to their owners.
4. No CLI, no RPC, no replication bytes are emitted or advertised.
5. The `section:5.4` registry binding in
   `internal/traceability/ownership.v0.6.0.json` still records
   "checkpoint creation ... is not implemented". That sentence is
   stale as of this change, but the registry projection digest is
   review-pinned (`reviewedOwnershipCanonicalSHA256`:
   "ownership claims cannot be self-minted without an explicit
   review of this binding"), so the registry text is left
   untouched for the review-gated registry maintenance. This
   matrix is the interim clause record.
