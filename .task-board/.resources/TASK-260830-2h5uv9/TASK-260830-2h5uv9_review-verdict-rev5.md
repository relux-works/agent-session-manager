# TASK-260830-2h5uv9 — review verdict, CR-TASK-260830-2h5uv9-5 revision 5

Verdict: **accepted** (`accept_cr`, revision 5)
Reviewer run: RUN-260924-a39b3d (claude-opus-5-5). Candidate tree `6d76b8d4c6dbdeb6f7716071ad578dda9ff1534d`: a scratch-index `write-tree` of the live worktree produces the same tree. Base `6d3bff9` (trunk has not moved). The prior verdict is rev4 (`sync-conflict-not-crossed-with-partial-peer`, plant E survived the configured suite).

## Rev4 finding: status, fixed
The rev5 delta against rev4 is test, harness, doc and registry changes only. `durable.go` and the other production files are unchanged. The `.pyc` file is gone.
- `TestDurableSyncGeneratedPerturbationProduct` no longer skips. With no selector it runs a bounded N<=3 shard, and a nested selector opts into N<=6. No `t.Skip` remains in the package outside the census test's planted fixtures.
- `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` is always-run and has 432 cases: 3 common IDs × every position × rounds 1..3 × peer-only record × skew × order × duplicate.

My own plants sit before the common-ID `fetchAndAdd` in `SyncFrom`. Each was run with `go test ./internal/merkleinventory -count=1`, which is the **configured suite with no selector**:

| Plant | Kind | Exit | FAIL lines |
| --- | --- | ---: | ---: |
| CTRL `_ = len(common)` | neutral control | 0 SURVIVED | 0 |
| E `common = common[:min(1,len(common))]` (rev4 reproduction) | narrowing | 1 KILLED | 321 |
| LAST, last common ID only | narrowing | 1 KILLED | 308 |
| EVEN, even positions only | narrowing | 1 KILLED | 177 |
| ODD, position 0 plus odd positions (drops the middle-even) | narrowing | 1 KILLED | 145 |
| MID, first plus last only | narrowing | 1 KILLED | 145 |
| NS0, first namespace only | narrowing | 1 KILLED | 731 |
| EXTRA, first ID only when the peer holds extras | narrowing | 1 KILLED | 321 |

I ran the shipped harness twice. It covered the sync-common*, skip-census* and control rows, 13 rows per run. Both runs exited 0 with every row KILLED, and the control reported `survived-control`. The harness restored the copy.

## Mandatory reruns (this reviewer, exact tree, archive copy)
| Check | Result |
| --- | --- |
| Full `go test ./... -count=1` | Every package ok except `internal/specpin`, which needs `.task-board` (excluded from the archive). Run live, `go test ./internal/specpin` is ok. |
| `GOOS=windows GOARCH=amd64 go vet ./...` | exit 0. `gofmt -l internal` is clean. |
| tracecheck | `traceability ok … acceptance_cases=163`, `clauses_discharged=88/610`. The digest recomputes to the pinned constant (tracecheck green). |
| Registry plant | Renaming the new test name in the ownership JSON makes tracecheck exit 1 (declaration absent). |
| README plant | `88/610` changed to `89/610` makes the tracecheck README pin FAIL. |
| Registry decode | 163 cases. The nxqqaw, 147hsj and 2h5uv9 case IDs are referenced in both `acceptance_cases` and `ownership` (7/10/10 string refs). The 2h5uv9 case names the new rev5 test and tracecheck resolves the declaration. |
| Importer grid | 27 importer packages of canonicaljson/provhost/sessquery/traceability (merkleinventory is absent at base and ran fully above). Running `go test -json` on base and candidate gives 0 failures on each side and **0 moved pass/fail**. Removed: canonicaljson unsupported-envelope subtests for tombstone and tombstone-ack. This is the named moved class, since those schemas are now supported (§10), same as rev4. Removed/added tmuxserver path-length subtests are noise from the scratch path (rev4 N3). Added: 369 canonicaljson, 36 sessquery, 1 traceability. |
| Determinism | New always-on tests plus the census and hygiene tests, run with `-count=3`: ok (124s). |
| task-board.config.json | Byte-identical to base. |

## Surface table
| Row | Result |
| --- | --- |
| (1) Property: every common identity audited at every position, in every namespace | held (7 plants killed under the configured suite, control survives) |
| (1) Generated product runs in the configured suite | held (no skip; shard runs by default; skip census has a planted control, killed) |
| (1) Other plants (tie-break, dup drop, regression, idempotence) | held (production unchanged since rev3/rev4, when these were killed; harness sync rows green) |
| (2) Unbounded axes | held (unchanged since rev4) |
| (3) Registry, digest, README | held |
| (4) Composition | held |
| (5) Harness rerun, hygiene, Windows vet | held |

Free hunt: the MID/ODD/EXTRA plants, beyond the briefed set, all killed. No findings.

## findings
```json
[]
```

## notes
- N1: `TestDurableSyncCommonIDAuditIsUnconditionalInAST` is still identifier-matched (rev4 N1). The behavioural shard now carries the kill, so this is a bound and not a defect.
- N2: tmuxserver subtest names depend on path length, which adds grid noise. This is not caused by this Story.
