# TASK-260830-g0pcnt review verdict — CR rev3: CHANGES REQUESTED

Reviewer: claude-opus-5-5 (low), RUN-260923-7f1001. Candidate tree cde6355d. I checked that the live worktree matches it through a scratch GIT_INDEX_FILE. Base 360c8bd. All work ran on `git archive` copies under /tmp/g0rv3.

## Mandatory results (rerun by me on the exact tree)
- `go test ./... -count=1`: 45/45 ok, no FAIL.
- `go build ./...`, `go vet ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` all exit 0. `gofmt -l internal/` is clean.
- Digest: the registry is byte-unchanged since rev2 (`git diff 83a43136 cde6355d -- internal/traceability` is empty). tracecheck exits 0 against the pin. Its output, `contracts=64 … acceptance_cases=160 …` / `bindings=70 full=4 partial=10 sliver=11 unevidenced=41 unmeasured=4 unowned=7 clauses_discharged=81/585`, equals README:3596.
- README plant: changing 81/585 to 82/585 made TestREADMEMeasuredCoverageMatchesTracecheckReport FAIL (KILLED). With the original restored it is ok.
- Registry, decoded: all six Story cases are referenced from ownership (tmux-attach-overlap/attach-semantics/lifecycle-operations/private-server-management/socket-custody-v070). There are 36 unreferenced cases, the same pre-existing set as rev2. `story-260922-derivation-side-profile-source` is still referenced.
- Trunk preservation: every path in `c9233ce..360c8bd` equals 360c8bd in the candidate, except the 7 known merged overlaps. `task-board.config.json` is byte-identical to the base. 360c8bd is an ancestor of the tip. 05d89f5, 602510f and 5443b3b all pass verify-commit.
- Importer outcome grid, rerun by me: the only importers are termbind and tmuxserver. For termbind/... + traceability/..., base 360c8bd has 413 (pkg,test,outcome) rows and the candidate has 429. All 413 base rows are present and unchanged: 0 moved, 16 new.
- Determinism: oracle, census and attach-property tests at `-count=3` → ok (121s).

## rev2 finding custody-mask-pinned-single-bit-axis: FIXED along the mode axis
TestCustodyModeOracleAtProductionEntries enumerates all 4096 low-12-bit modes × 4 component classes × Probe/Spawn/Execute. It checks each mode against an oracle in the test that is independent of the production predicates. Each mutant below was run ALONE with `-run '^TestCustodyModeOracleAtProductionEntries$'`:

| Plant (mine) | Kind | Package run | Killer alone |
|---|---|---|---|
| C0 comment whitespace in bind.go (applied, neutral) | control | SURVIVED (expected) | — |
| P2 ancestor admits group-write when setgid | narrowing | KILLED | oracle: "want … socket ancestor, got nil" |
| P3 runtime dir ignores sticky | narrowing | KILLED | oracle: "… runtime mode, got nil" |
| P4 root admits exactly 0711 | narrowing | KILLED | oracle: "… socket root, got nil" |
| P5 socket leaf ignores setuid | narrowing | KILLED | oracle: "… socket permissions, got nil" |
| **P1 ancestor walk checks only the first ancestor above root** | narrowing (position axis) | **SURVIVED** (tmuxserver+termbind exit 0) | none |

N1: disposed by TestExecuteSerializesSameClientAttachBeforeConcurrentWinnerArm (present, passes ×3). I did not re-attack it.

## Findings
```json
[
 {"id":"custody-ancestor-walk-pinned-at-first-position",
  "row":"socket substitution / custody (§3.2 806-813, component-by-component)",
  "invariant":"Every ancestor above the runtime root, at every depth up to the filesystem root, is refused if it is group/other writable and not sticky",
  "mechanism":"internal/tmuxserver/bind.go checkCustodyAncestors loop. The oracle test projects a mode only onto filepath.Dir(root), and the census/sticky tests chmod only filepath.Dir(root). No committed test puts an unsafe mode at depth ≥2, so a walk that stops after the first ancestor stays green.",
  "reproductions":[
   "plant P1: insert `if cleaned != filepath.Clean(root) { return nil }` before `dir, err := secprim.OpenNoFollowDir(parent)` in bind.go; `go test ./internal/tmuxserver/ ./internal/termbind/ -count=1` -> exit 0 (expected: failure). log: m-P1-ancestor-first-only.pkg.log",
   "probe probe_gp_unix_test.go (grandparent 0777, parent 0700): pristine -> `tmux_unsafe_socket_path at socket ancestor`; under P1 -> <nil> (ADMITTED)"],
  "severity":"bypass (evidence gap: production refuses correctly, but the suite stays green under a gate that admits a writable non-sticky grandparent that §3.2 refuses)",
  "repeat-of":"rev2 custody-mask-pinned-single-bit-axis (same class: custody rule pinned along one axis. The axis is now component position, not mode bits)"}
]
```

## Notes (non-blocking)
- `custodyModeProjection` is a package-level test seam that production reads on every custody check. It is nil in production, and no production code assigns it (grep). Acceptable, but consider noting it in TRACEABILITY with the other seams.
- Ancestor ownership (N2) is carried as a stated bound per the producer. I did not re-attack it.
- I did not rerun the shipped mutant harness this round. The kills above come from my own plants.

## Surface rows
- lost-response idempotency: held (×3 rerun; the rev2 E1 kill still applies, attach code unchanged since rev2)
- reconnect/multi-attach: held (N1 serialization test present). I did not attack it further.
- socket substitution/custody: **broken** (custody-ancestor-walk-pinned-at-first-position). The mode axis held: P2-P5 were KILLED.
- ownership-neutral attach: held (×3). I did not attack it further.
- registry/traceability: held
- trunk preservation/composition: held

Measured by me: 4 of 5 custody narrowings KILLED, plus 1 control SURVIVED.

## Rework scope (for the producer)
1. Pin the ancestor rule along the POSITION axis. Build a fixture with at least 3 ancestor levels under a private temp base. For each depth d (1..3), make only the ancestor at depth d writable and non-sticky (e.g. 0777 and 0770). Assert `tmux_unsafe_socket_path at socket ancestor` through Probe, Spawn and Execute with zero effects, and admit the 01777 sticky equivalent at each depth. Alternatively, extend the oracle projection to every ancestor on the walk, one at a time.
2. Ship P1 (stop after the first ancestor) and "skip the first ancestor" as harness mutants. Each must be KILLED by the new test run alone. Keep the neutral control.
3. This is the third round of one class. Before handoff, enumerate every axis of each custody gate (mode bits, component class, component position/depth, entry) and state for each axis which test varies it.
