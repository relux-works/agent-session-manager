# TASK-260830-g0pcnt review verdict — CR rev5 — CHANGES REQUESTED

Reviewer run RUN-260923-3ee53c (claude-opus-5-5 low). Candidate tree ba6b178, re-derived from the live worktree through a scratch GIT_INDEX_FILE: equal. Base 0ca3e4c. The rev4→rev5 delta touches only tests, TRACEABILITY, the harness, README and LOGBOOK; production bind.go is unchanged.

## Rev4 finding (custody-ancestor-walk-pinned-at-fixture-depth)
FIXED at the rev4 vector. An in-loop "cap after 4 ancestors" mutant is KILLED (TestCustodyModeOracleAtProductionEntries/extra_nested_depth_08/ancestor). The AST guard TestCustodyAncestorWalkHasNoLengthDependentControlFlow also rejects any counter, literal or early nil inside checkCustodyAncestors.

## My mutants (bind.go, each run alone)
| Plant | Kind | Result |
|---|---|---|
| cap4: in-loop counter, return nil after 4 ancestors | narrowing (walk length) | KILLED (ModeOracle/extra_nested_depth_08/ancestor) |
| callee12: in custodyModeForPath, clear 0o022 on dirs when strings.Count(path,"/") > 12 | narrowing (walk length, in a callee) | KILLED (ModeOracle/…/runtime_tmux_dir) |
| callee6: same, > 6 | narrowing | KILLED |
| **callee24: same, > 24** | narrowing (walk length, in a callee) | **SURVIVED**: tmuxserver+termbind full packages, exit 0 |
| neutral: Clean(Clean(root)) | applied control | SURVIVED, as a control should |

Measured: 3 of 4 narrowings KILLED.

## Findings
```json
[
 {"id":"custody-ancestor-walk-pinned-at-fixture-depth",
  "row":"socket substitution / custody (§3.2 806-813, component-by-component)",
  "invariant":"Every ancestor at ANY depth up to the platform PATH_MAX is refused if it is group/other writable and not sticky. The unbounded walk-length axis is closed only if EVERY function on the per-component decision path is shown to be length-independent, not only the loop body.",
  "mechanism":"The structural argument (matrix row `writableByOthers | Walk length`, TestCustodyAncestorWalkHasNoLengthDependentControlFlow) inspects only the body of checkCustodyAncestors (bind.go). The per-component decision flows through custodyModeForPath(parent, mode) (bind.go:143), which receives the full path, and through isFilesystemRoot(parent) and secprim.OpenNoFollowDir(parent). A length-dependent branch placed in a callee is invisible to the AST guard. The behavioral generator stops at base+16 levels (~22 separators), so a cap above that survives.",
  "reproductions":["plant callee24 (review-scratch plant.py): in custodyModeForPath, after projection, `if mode.IsDir() && strings.Count(path, \"/\") > 24 { mode &^= 0o022 }`; `go test ./internal/tmuxserver/ ./internal/termbind/ -count=1` -> exit 0 (expected: failure). Log callee24-full.log. The same plant at >12 and at >6 is KILLED, which shows the survivor lies purely beyond the generator range."],
  "severity":"bypass (evidence gap: production is correct today, but the suite stays green under a gate that admits a world-writable ancestor at more than 24 path separators)",
  "repeat-of":"rev4 custody-ancestor-walk-pinned-at-fixture-depth (same mechanism, fifth round of the class: the structural census is scoped to one FuncDecl, and the generator range is finite)"}
]
```

## Notes
- The fixture base is /private/tmp (lxProbeRoot); the fixture scratch location is producer hygiene, not a finding.
- N2 (ancestor ownership) is still a stated bound. Not attacked.

## Mandatory results
- Digest / registry: ownership.v0.7.0.json is unchanged since rev4 (verified there). tracecheck: `clauses_discharged=81/585`, equal to README:3599.
- README plant 81→82: the tracecheck README pin test FAILS (pin works).
- Full `go test ./... -count=1` on the exact tree (archive copy): every package ok except specpin, which failed only because `.task-board` was excluded from the copy; specpin rerun in the live worktree: ok.
- `GOOS=windows GOARCH=amd64 go vet ./...` exit 0; `go vet` exit 0; gofmt clean.
- task-board.config.json is byte-identical to 0ca3e4c.
- Importer outcome grid (termbind, rerun by me): base 265 outcomes, all present and unchanged in the candidate; 0 moved, 13 added.
- Determinism: the new depth/name/symlink tests at -count=3: exit 0.

## Surface rows
- socket substitution / custody: **broken** (above). The mode, kind, position and in-loop cap axes held.
- lost-response idempotency, reconnect/multi-attach, ownership-neutral attach, inherited items: not-attacked (budget). The attach code is unchanged since rev2/rev3, where these rows held.
- registry/traceability: held. trunk preservation/composition: held.

## Rework scope (for the producer)
Close the walk-length axis by construction, not with a larger K:
1. Generate the depth up to the platform limit: nest 1-byte directory names until the socket path approaches PATH_MAX (darwin 1024, or the sun_path-independent ancestor walk only), and put a 0777 non-sticky ancestor at the deepest and middle positions. Any path-count cap below PATH_MAX then dies.
2. OR extend the structural guard to the transitive callees on the decision path (custodyModeForPath, writableByOthers, isFilesystemRoot, and the secprim open), and control-plant it with callee24.
3. Ship callee24 (and a cap in isFilesystemRoot) as harness mutants, each KILLED by a test run alone.
