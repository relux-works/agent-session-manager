# TASK-260922-vcx6yo review verdict — CR rev2 — ACCEPTED

Reviewer: claude-opus-5-5 (low). Candidate tree 3e5f04afa6d5e5cb1bad2d305147a50cecf6a8f1 over base d8decbb; the live worktree tree (scratch GIT_INDEX_FILE) reproduced this OID exactly.

## Rev1 findings
| Finding | Status | Evidence |
|---|---|---|
| P1 lock only witnessed for input clients | FIXED | ops.go:157 AcquireAdmission is unconditional; TestAttachReadOnlyOverlapAdmissionAtomicAcrossLifecycleInstances added. My plant "lock only when !InputAuthorized" (the opposite axis from M2) KILLED (1 fail). |
| P2 input gate checked at peers[0] | FIXED | ops.go:335 loops over all peers; TestAttachOverlapInputGateChecksEveryPeer. Plants peers[:1] KILLED (3 fails), last-peer-only KILLED (1), "skip input gate when len(peers)<3" KILLED (2). |
| P3 census honesty | FIXED | TRACEABILITY admissionlock row now names both axes, and replay/concurrency gaps are stated as BOUNDs with owners. |

## Own instruments (rerun, not accepted from the log)
- `go build ./...`, `go vet` (termbind, tmuxserver), `GOOS=windows GOARCH=amd64 go vet ./...` all exit 0.
- `go test -count=1 ./internal/termbind ./internal/tmuxserver` passes with PATH=/usr/bin:/bin:<go dir> (no tmux reachable). Determinism holds.
- Mutants run on a copy with `-run Attach`: M1 peers[0] KILLED; M2 last-peer KILLED; M3 lock-readonly-only KILLED; M4 "multi_attach waived for an input client over a single read-only peer" KILLED; M5 KILLED; the M6 no-op control SURVIVED, so the harness can report a survivor.
- Census extended, not forked: Peers delegates the filename binding to Lookup; the lock file has no `.json` suffix and is skipped.

## Residual (accepted as stated bounds)
- B44 liveness: a receipt stays a possible peer because there is no retirement signal. That is over-strict, not over-permissive. Owner: the future protocol owner.
- The case where a caller holds a read-only receipt and replays with InputAuthorized=true skips the overlap gate. It relies on the landed store's idempotency_mismatch refusal, which I did not re-plant this round.
- I did not rerun the full importer grid or the full suite; those were accepted from the attached CR validation log.
