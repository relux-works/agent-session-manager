# TASK-260830-g0pcnt review verdict — CR rev6 — ACCEPTED

Reviewer run RUN-260924-d2d2e0 (claude-opus-5-5 low). I re-derived the candidate tree from the live worktree through a scratch GIT_INDEX_FILE: `bc9bb130`, equal to the CR. Base `0ca3e4c`. The rev5→rev6 delta (`ba6b178..bc9bb130`) touches only `custody_walk_depth_unix_test.go`, `mutant_harness.py`, `TRACEABILITY.md` and `LOGBOOK.md`. Production, the registry and README are unchanged since rev5.

## Rev5 finding (custody-ancestor-walk-pinned-at-fixture-depth): FIXED
- `TestCustodyAncestorWalkGeneratedDepthToPathMax` generates one-byte nesting up to `unix.PathMax`. It asserts that the runtime root has PATH_MAX-1 bytes and the deepest ancestor has PATH_MAX-3 bytes, and it refuses a 0777 ancestor at the deepest and at a middle position.
- `TestCustodyAncestorWalkHasNoLengthDependentControlFlow` walks the production call graph transitively from `checkCustodyAncestors`.
- The PATH_MAX fixture uses `t.TempDir()` with a reverse-order cleanup. Nothing is left under /private/tmp.

## My plants (bind.go, each run alone on an archive copy; command `go test ./internal/tmuxserver/ ./internal/termbind/ -count=1`)
| Plant | Kind | Result / killer |
|---|---|---|
| callee24: `strings.Count(path,"/")>24` clears 0o022 in custodyModeForPath | narrowing, walk length in a callee | KILLED: GeneratedDepthToPathMax/deepest_non_root and /middle, plus the call-graph guard |
| funcvar: package `var ancestorSkip = func(p) bool {len(p)>600 && len(p)<1000}` called in the loop (a behaviourally unwitnessed band between middle and deepest) | narrowing, guard evasion through a func literal | KILLED: guard only (behaviour cannot see this band; the guard covers it) |
| mapcap: `deepSet[len(parent)/100]` lookup clears 0o022 for bytes 700-899 | narrowing, threshold hidden in map data | KILLED: guard only |
| escalate: attach.go:151 `InputAuthorized != x` → `(... && !x)` (admits read-only→writable replay only) | narrowing, inherited item 2a | KILLED: TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable |
| neutral: `Clean(Clean(root))` | applied control | SURVIVED (exit 0), as a control should |

Measured: 4 of 4 narrowings KILLED; the control survived. No plant was NOT_APPLIED (each was checked with cmp).

## Mandatory results
- Registry / digest: `ownership.v0.7.0.json` is byte-unchanged since rev5, where the digest was re-derived and verified. tracecheck on the live tree prints `clauses_discharged=81/585`, equal to README:3599. The README pin plant was run in rev5 and README is unchanged; I did not rerun it this round.
- Full suite on the exact tree:
  - tmuxserver and termbind: green under the neutral control, which is functionally the exact tree.
  - All other packages: `ok`.
  - specpin: `ok`, run in the live worktree.
- `GOOS=windows GOARCH=amd64 go vet ./...`, `go vet` and gofmt: all clean.
- `task-board.config.json` is byte-identical to `0ca3e4c`.
- Importer outcome grid: production is unchanged since rev5, where the reviewer reran it (0 moved). I carried that result over and did not rerun it this round.
- Determinism: `-run 'TestCustodyAncestorWalk|TestCustodyPathComponent|TestCustodySymlinkChain' -count=3` exit 0.

## Surface rows
- socket substitution / custody: **held** (the plants above).
- read-only→writable escalation (inherited 2a): **held** (the escalate plant).
- lost-response idempotency, reconnect/multi-attach, ownership-neutral attach, the fresh-decoy P3-A item and foreground composition: **held**. They were attacked in rev2-rev4, and the attach and lifecycle production is byte-unchanged since then. This round I attacked the shared attach replay gate (escalate).
- registry/traceability, trunk preservation, composition: **held**. The registry and README are unchanged, tracecheck equals the README, and the config is byte-equal.

## Findings
```json
[]
```

## Notes
- The call-graph guard is the sole killer for length bands that the behavioural fixture does not witness (between the middle and deepest positions). That is by design, since the transitive guard is the structural half, but it means the guard itself is load-bearing.
- N2 (ancestor ownership) and B44 liveness remain stated bounds, as in prior rounds.
