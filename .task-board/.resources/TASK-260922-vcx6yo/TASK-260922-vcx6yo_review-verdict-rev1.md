# TASK-260922-vcx6yo review verdict — CR rev1: CHANGES REQUESTED

Reviewer: claude-opus-5-5 low. Candidate tree 2a23b9f (base d8decbb), reviewed on a `git archive` copy at /tmp/vcx; the live index was not touched.

## Reproduced
- `go test ./internal/termbind/ ./internal/tmuxserver/` on the candidate: both packages ok, exit 0. The full repo suite, windows vet, the determinism check with tmux unresolvable, and the importer outcome grid were NOT rerun by me. Those were accepted from the attached validation log only, and I did not independently verify them.
- Mutants I planted in ops.go (run with the full tmuxserver package and -count=1):

| ID | Narrowing | Result |
|---|---|---|
| M1 | AcquireAdmission replaced by a no-op | KILLED by TestAttachOverlapAdmissionAtomicAcrossLifecycleInstances |
| M2 | admission lock taken ONLY when body.InputAuthorized | **SURVIVED** |
| M3 | multi_attach skipped when requester and first peer are read-only | KILLED (TestAttachOverlapRequiresMultiAttach, TestAttachReceiptWithoutPositiveLivenessRemainsPossiblePeer) |
| M4 | replay exemption disabled | KILLED by TestAttachSameClientRetryWithPeerPresent |
| M5 | multiple_input_clients checked only against peers[0] | **SURVIVED** |
| M6 | admission lock released before the receipt commit | KILLED by the atomicity test |
| C0 | harmless defer wrapper (control) | ok, applied |

My narrowings: 4 of 6 killed. That is 1 of 2 on the atomicity gate and 3 of 4 on the overlap capability gates.

## Findings
**P1-A (atomicity only witnessed on the writable axis).** M2 survives, so no test drives the read-only-no-multi scenario concurrently. Under M2, two concurrent read-only attaches on an instance without multi_attach would both read an empty census and both commit receipts and vectors. This is exactly the rev7 scenario, reached through the race instead of sequentially. The DoD asks for atomicity "where overlap is decided", and overlap is decided for read-only clients too. Rule pinned along one axis.

**P2-A (concurrent-input gate pinned at one peer position).** M5 survives. Every multiple_input_clients row has exactly one input-authorized peer, and that peer sorts first. Setup that exposes it: peer A is read-only and sorts first, peer B is input-authorized, a new writable client C arrives with multi_attach and without multiple_input_clients. Under M5, C is admitted with a second writable vector.

**P3 (census honesty).** The gate × entry census and the conformance matrix mark the lock gate Measured by a single writable-axis row. Re-derive the census once the P1/P2 rows exist.

Liveness rule (unknown → possible peer), Peers integrity (the .admission.lock file lacks the .json suffix and is skipped, so the census is not forked), and the Windows LockFileEx arm: read, no finding.

## Rework scope (for the producer)
1. Add a paused-adapter concurrent test with two READ-ONLY clients and no multi_attach. Exactly one receipt and one vector may result, and the M2 narrowing must fail it.
2. Add a concurrent-input row with multiple peers where the input-authorized peer is NOT first in client-ID order. It must refuse without multiple_input_clients and commit no receipt, and the M5 narrowing must fail it.
3. Refresh the census and matrix rows, with kill attributions for M2 and M5.
