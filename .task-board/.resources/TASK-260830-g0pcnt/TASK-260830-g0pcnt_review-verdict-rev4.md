# TASK-260830-g0pcnt review verdict — CR rev4 — CHANGES REQUESTED

Reviewer run RUN-260923-850a18 (claude-opus-5-5 low). Candidate tree 916dc1a (live worktree tree re-derived through a scratch index: equal). Base 0ca3e4c.

## Rev3 finding
custody-ancestor-walk-pinned-at-first-position: FIXED at the rev3 vector. The P1 plant ("stop after the first ancestor") is KILLED by TestCustodyModeOracleAtProductionEntries/ancestor and TestCustodyPathKindOwnerOracleAtProductionEntries.

## My mutants (bind.go, each run alone against tmuxserver+termbind, -count=1)
| Plant | Kind | Result | Killer |
|---|---|---|---|
| P1 stop after the first ancestor | narrowing (position) | KILLED | ModeOracle/ancestor, KindOwnerOracle |
| skip the ancestor at 3 slashes | narrowing (position) | KILLED | BitCensus/ancestor/group_write/* |
| skip the last ancestor below / | narrowing (position) | KILLED | ModeOracle/ancestor |
| skip relative depth 2 | narrowing (position) | KILLED | ModeOracle/ancestor |
| check only even depths | narrowing (position) | KILLED | ModeOracle/ancestor |
| ancestor mask 0o022→0o020 (other-write ignored) | narrowing (mask) | KILLED | BitCensus/ancestor/other_write/{Probe,Spawn,Execute} |
| ancestor kind check dropped | narrowing (kind) | KILLED | KindOwnerOracle/{Probe,Spawn,Execute} |
| root setgid admitted | narrowing (special bit) | KILLED | ModeOracle/root |
| **walk capped after 4 ancestors** | narrowing (position / walk length) | **SURVIVED** (exit 0) | none |
| control: Clean(Clean(root)) | applied neutral | SURVIVED (harness able to report a survivor) | — |

Measured: 8 of 9 narrowings KILLED; the control SURVIVED as it should.

## Findings
```json
[
 {"id":"custody-ancestor-walk-pinned-at-fixture-depth",
  "row":"socket substitution / custody (§3.2 806-813, component-by-component)",
  "invariant":"Every ancestor from the runtime root up to the filesystem root, at ANY depth, is refused if it is group/other writable and not sticky. A real runtime root (for example macOS /private/var/folders/xx/yyy/T/..., 5+ ancestors) is deeper than the fixture.",
  "mechanism":"internal/tmuxserver/bind.go checkCustodyAncestors loop. Every oracle, census and kind/owner fixture roots at lxProbeRoot or lxShortRoot, which is /private/tmp/axprobeNNN/r: exactly 4 ancestors. The sweep covers every position of THAT walk, but the walk length is fixed, so a walk that stops after the fixture's depth (a cap of 4) is indistinguishable from the real walk.",
  "reproductions":[
   "plant depthCap4: replace `cleaned := filepath.Clean(root)\\n\\tfor {` with `cleaned := filepath.Clean(root)\\n\\tn := 0\\n\\tfor {\\n\\t\\tn++\\n\\t\\tif n > 4 { return nil }`; then `go test ./internal/tmuxserver/ ./internal/termbind/ -count=1` -> exit 0 (expected: failure). Log: depthCap4.log",
   "probe zz_deep_probe_unix_test.go (root /private/tmp/axdeepN/a/b/c/d/e/f/g, where a is 0777; the probe calls checkCustodyAncestors, which all three entries share): pristine -> `tmux_unsafe_socket_path at socket ancestor`; under depthCap4 -> <nil> (ADMITTED). Logs: probe-deep-pristine.log, probe-deep-mutant.log"],
  "severity":"bypass (evidence gap: production refuses correctly, but the suite stays green under a gate that admits a world-writable ancestor at depth >= 5)",
  "repeat-of":"rev3 custody-ancestor-walk-pinned-at-first-position (same class, fourth consecutive round: the position axis is now pinned up to the fixture's depth, not over walk length)"}
]
```

## Notes (non-blocking)
- The registry has 36 acceptance cases that no ownership entry references. All 36 are trunk cases present at 0ca3e4c; this Story introduces none of them. The 5 new Story cases (tmux-*-v070) are all referenced, and no base case was lost. Trunk hygiene, outside this Story.
- N2 (ancestor ownership) is still a stated bound. I did not attack it.

## Mandatory results
- Digest: reviewedOwnershipCanonicalSHA256 = 3664ab2f…; the registry-rederivation tests pass (`go test ./internal/traceability/...` ok).
- tracecheck: `clauses_discharged=81/585`, matching README line 3599.
- README plant 81→82: TestREADMEMeasuredCoverageMatchesTracecheckReport FAILS (pin works).
- Full `go test ./... -count=1` on the exact tree: exit 0.
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0. `go vet ./...` exit 0, gofmt clean.
- task-board.config.json is byte-identical to 0ca3e4c.
- Importer outcome comparison (termbind, the only importer that exists on trunk; tmuxserver is new): 60 trunk outcomes, 0 moved, 13 added.
- Determinism: TestCustody*, TestExecuteSerializes*, *Attach* at -count=3: exit 0.
- cpkajd case story-260922-derivation-side-profile-source is present.

## Surface rows
- socket substitution / custody: **broken** (custody-ancestor-walk-pinned-at-fixture-depth). The mode, kind, mask and special-bit axes held (8 KILLED).
- lost-response idempotency: not-attacked (budget; the attach code is unchanged since rev2, where it held).
- reconnect / multi-attach: not-attacked (budget; held at rev3).
- ownership-neutral attach: not-attacked (budget; held at rev3).
- inherited items (escalation, P3-A decoy, foreground integration, B44): not-attacked this round (held at rev2/rev3; the code is unchanged).
- registry / traceability: held (see above).
- trunk preservation / composition: held (config identical, importer grid 0 moved, full suite green).

## Rework scope (for the producer)
1. Make walk LENGTH an axis. Build the oracle and kind/owner fixtures with a deep root, at least 8 ancestors (nested dirs under the temp base), and keep sweeping every position. Better: also add a test that varies the root depth (for example 1..10 extra levels) and asserts that a 0777 ancestor at the deepest non-root position is refused through Probe, Spawn and Execute.
2. Ship "walk capped after K ancestors" as a harness mutant for K = fixture depth and K = fixture depth + 2. Each must be KILLED by the oracle test run alone.
3. Add walk length/depth to the axis inventory. Name the maximum depth witnessed and state why the walk has no length-dependent branch.
