# TASK-260830-147hsj — Review verdict, CR revision 2

- Reviewer run: RUN-260924-19d34f (claude-opus-5-5)
- Candidate tree: af575645ae5704570dd9dbbde87e2fd47c36af8e over base b81258e (live worktree tree re-derived through a scratch GIT_INDEX_FILE and it matched)
- Verdict: **accepted** (`accept_cr`, revision 2)
- Prior verdicts: none (rev1 failed construction and has no verdict)

## Mandatory reruns (all run by me on the exact tree)

| Check | Result |
|---|---|
| `go test ./... -count=1` (archive copy) | all packages ok except `specpin`, which failed only because the archive excluded `.task-board` (lstat ../../.task-board). Rerun in the live worktree: `ok` |
| `GOOS=windows GOARCH=amd64 go vet ./...` | exit 0 |
| `go vet` + `gofmt -l` (merkleinventory, sessquery) | clean |
| Determinism, `-count=3` on the new tests (merkleinventory+sessquery, Durable/Projection/LeaseHeads/carry-over masks) | ok / ok |
| Independent JCS+SHA-256 (`jcs_check.py`, Python, self field omitted per canonicaljson) over 9 fixtures (5 namespace fixtures, session record, two competing leases, tombstone) | 9/9 canonical bytes, 9/9 self-digest match, 0 mismatches |
| `task-board.config.json` and `internal/traceability` vs base | byte-identical (no registry edit; task_delta scope held) |
| Importer outcome grid `(package, test)` base vs candidate over merkleinventory, sessckpt, sessquery, termbind | base 1430 pass, candidate 1494 pass; **0 existing rows moved**; 64 new rows (merkleinventory +28, sessquery +36), all pass |
| Shipped harness, the 23 leaf rows + control, run twice | 23/23 KILLED both runs, and the named test ran alone each time; `control-neutral-comment` SURVIVED both runs |

## Own plants (applied by hand to a scratch copy, named test run alone)

| Plant | Class | Result |
|---|---|---|
| R1: projection keeps only the lease with the greatest `created_at` (timestamp winner) | narrowing | KILLED (TestProjectionRebuildExhaustsUnionArrivalsWithoutTimestampAuthority) |
| R2: projection uses only the last-arrived lease (insertion-order dependence) | narrowing | KILLED (same test, conflict-kinds oracle) |
| R3: a losing-branch `session.idle` event in the union sets the projected state (leak into authoritative state) | narrowing | KILLED (same test, state oracle `creating`) |
| R4: on reopen, quarantine is forgotten when the conflict pair was persisted (restart idempotency / same-digest) | narrowing | KILLED (TestDurableConflictCrashRestoresQuarantineAndAbortsSync) |
| R5: reopen drops the `tombstone_ack` namespace (lost across restart) | narrowing | KILLED (TestDurableJSONAddIsIdempotentAcrossCrashRestartByNamespace) |
| R6: same-digest different bytes accepted when the variant is exactly one byte longer | narrowing | KILLED (TestDurableSyncAuditsCommonIDsAndQuarantinesDifferentBytes) |
| R0: neutral comment | applied control | SURVIVED |

## Surface table

| Row | Result | Attacks |
|---|---|---|
| Carry-overs (fetch-dup-ids, b64-roundtrip, walk-quarantine, walk-crossns) | held | Replanted all four through the harness twice; each is KILLED alone. walk-crossns is pinned by seeding internal state; the test states that AddJSON cannot produce it (IDs are global in `locations`). I accept this as a defense pin with a stated unreachability reason. |
| Durable union: validation before persist, idempotency, crash/restart | held | R4, R5, plus the shipped durable-* rows. Crash children exit with an explicit os.Exit(73) at install, first/second quarantine write, and active removal. |
| Same digest with different bytes: quarantine and abort | held | R6, plus sync-common-record-audit and durable-same-id-different-bytes |
| Tombstones and acks unioned, not executed | held | The ack-first order is admitted; unclosed and mismatched acks are refused with integrity_failure at both SyncFrom and RebuildProjection. The repo session is retained. |
| Arrival order / timestamps / losing lease | held | R1–R3. The permutation oracle is independent: its expectations are literal spec tuples and literal conflict tokens, not production constants. It covers 720 orders × 2 opposite timestamp profiles. |
| Projection rebuild purity | held | R2; RebuildProjection iterates sorted IDs only and reads no clock (inspected) |
| Axes / entries | held | Durable.AddJSON, SyncFrom, RebuildProjection (Index and DurableIndex) and LeaseHeadsForSession are each driven. LeaseHeads has a generated cardinality range. |
| Composition (importer grid) | held | I reran the grid myself; 0 rows moved |

Coverage ratio I measured: **6 of 6** derived AC rows driven through named production entries. This matches the producer's claim.

## findings

```json
[]
```

## notes (non-blocking)

- The permutation test holds the repository's authoritative chain fixed and permutes only the union arrival order. That is the correct model: the authoritative chain is not union-derived. Divergence *inside* the repo chain is out of this leaf.
- Crash evidence covers process-exit points and does not cover power loss. This is stated as a bound in CONFORMANCE-MATRIX.
- The full-suite `specpin` failure in archive copies is environmental (it needs `.task-board`) and is not a candidate defect.

Free hunt: I also checked SyncFrom lock order (store.mu only, peer read through index RWMutex; no cross-lock deadlock) and the load() quarantine migration re-entrancy (persist is idempotent by content hash). Nothing reproduced.
