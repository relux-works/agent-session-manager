# TASK-260830-g0pcnt review verdict — CR rev2: CHANGES REQUESTED

Reviewer: claude-opus-5-5 (low), RUN-260923-b0055a. Candidate tree 83a43136. The live worktree matches it (checked with a scratch GIT_INDEX_FILE). Base 360c8bd. All work ran on `git archive` copies.

## Mandatory results (rerun by me on the exact tree)
- `go test ./... -count=1`: 45/45 ok. `specpin` needs `.task-board`, so I ran it in the live worktree, where it is ok.
- `go vet ./...` exit 0. `GOOS=windows GOARCH=amd64 go vet ./...` exit 0. `gofmt -l`: clean.
- Digest: tracecheck exit 0, meaning the pin `3664ab2f…` equals the recomputed projection. Its output: `contracts=64 normative_sections=36 acceptance_cases=160 … clauses_discharged=81/585`, which equals README line 3596.
- README plant: changing 81/585 to 82/585 made `TestREADMEMeasuredCoverageMatchesTracecheckReport` fail (KILLED). With the original restored it is ok.
- Registry, decoded: all six Story cases appear in their clause lists (4.2#4-7, 4.2#9, 4.C#3-7, 3.2#8). The 36 unreferenced cases are all pre-existing landed ids. No base case was removed or changed. `story-260922-derivation-side-profile-source` is still in `2.4#2`.
- Trunk preservation: all 57 trunk-only paths of `c9233ce..360c8bd` are blob-equal to 360c8bd, with 0 bad. `task-board.config.json` is byte-identical to the base. The checkpoints 05d89f5, 602510f and 5443b3b pass verify-commit, and 360c8bd is an ancestor of the tip.
- Importer outcome comparison, rerun by me: the importers of termbind/tmuxserver are only those two packages, and tmuxserver is new. For termbind and traceability/..., base has 413 (pkg,test,outcome) rows and the candidate has 429. All 413 base rows are present and unchanged, and there are 16 new passes. 0 classes moved. Landed test edits are limited to registry figure updates (155→160 cases, etc.).
- Determinism: 16 new or named tests at `-count=3` → 48/48 PASS.

## Mutation (my plants, not in the shipped harness; the package run is tmuxserver+termbind; killers were then run ALONE with -run)
| Plant | Kind | Result | Killer (alone) |
|---|---|---|---|
| C-control (comment whitespace in bind.go, applied) | neutral | SURVIVED (expected) | — |
| M1 socket leaf admits exactly other-rw (0606) | narrowing, combination | KILLED | TestCustodyPermissionSocketLeafExactModes |
| M2 ancestor treats setgid as sticky (`ModeSticky\|ModeSetgid`) | narrowing, special bit | **SURVIVED** | none |
| M3 Execute-only: ignore "socket permissions" refusal | narrowing, entry-scoped | KILLED | TestExecuteEveryOperationRefusesForeignOrPermissiveSocket |
| M4 Spawn-only: ignore "socket ancestor" refusal | narrowing, entry-scoped | KILLED | TestSocketCustodySymlinkRefusesAtProbeAndSpawn |
| M5 socket root admits exactly group r-x (0750) | narrowing, combination | **SURVIVED** | none |
| M6 Probe-only: ignore "socket ownership" refusal | narrowing, entry-scoped | KILLED | TestServerProberRefusesForeignSocketBeforeConnect |
| E1 attach Lookup arm admits read-only→writable | narrowing | KILLED | TestExecuteAttachReadOnlyReceiptCannotEscalateToWritable |
| E2 attach concurrent-winner arm admits read-only→writable | narrowing | SURVIVED | see note N1 |

Measured: 5 of 7 custody narrowings KILLED, 1 of 2 escalation narrowings KILLED. Probe on the pristine code: a 02777 (setgid, not sticky) ancestor is refused `socket ancestor`, and a 0750 root is refused `socket root`. Under M2 and M5 respectively, both return nil. The committed suite does not notice either change.

## Findings
```json
[
 {"id":"custody-mask-pinned-single-bit-axis","row":"socket substitution / custody (§3.2 806-813)",
  "invariant":"Custody refuses any root/runtime/socket mode with a non-owner bit and any group/other-writable ancestor that is not sticky, for every mode in the class, not only single-bit modes",
  "mechanism":"internal/tmuxserver/bind.go:108 (root mask) and bind.go:176-181 (writableByOthers). TestCustodyPermissionBitCensus enumerates each non-owner bit singly, plus 01777 for the sticky admit. No committed row drives a multi-bit combination at root or ancestor, and none drives a non-sticky special bit (setgid/setuid) on a writable ancestor. A narrowing that admits exactly one such member survives.",
  "reproductions":[
   "plant M2 (bind.go: `info.Mode()&os.ModeSticky == 0` -> `info.Mode()&(os.ModeSticky|os.ModeSetgid) == 0`); `go test -count=1 ./internal/tmuxserver/ ./internal/termbind/` -> exit 0 (expected: a failure). Under M2, CheckSocketCustody with a 02777 ancestor returns nil; pristine returns tmux_unsafe_socket_path/socket ancestor. log: plants/M2-ancestor-setgid-as-sticky.log",
   "plant M5 (bind.go:108: `permissions&0o077 != 0` -> `(permissions&0o077 != 0 && permissions&0o077 != 0o050)`); same command -> exit 0. Under M5 a 0750 socket root returns nil; pristine returns socket root. log: plants/M5-root-admit-group-rx-only.log"],
  "severity":"bypass (evidence gap: the committed suite stays green under a gate that admits a mode §3.2 refuses; production itself refuses correctly)",
  "repeat-of":"rev1 P2 (rule pinned along one axis: socket permission gate 0600/0644 only)"}
]
```
This is the second consecutive same-class finding. Rev3 must carry a named regression test and narrowing mutants for the class, as described below.

## Notes (non-blocking)
- N1: E2, the concurrent-winner arm at `termbind/attach.go:249`, is landed base code. It survived the tmuxserver+termbind suites. The matrix row at conformance-matrix:233 declares concurrent same-client arbitration a bound owned by `admissionlock`. If the admission lock serializes same-client attaches through Execute, the arm is unreachable from Execute. I did not prove that, so either pin E2 with a hook-driven concurrent test (the `AfterStage` winner) or state the unreachability with its reason.
- N2: the ancestor walk checks writability and kind but not ownership. A foreign non-root owner of an ancestor could rename. I could not reproduce this without root, so it is a note only.
- Inherited items: (a) escalation, KILLED by E1, see above. (b) The P3-A fresh-decoy narrowing: acquire.go is blob-unchanged since rev1, where it was KILLED by TestAcquireForegroundRefusesEveryCatalogDecoyRunning. I did not replant it this round. (c) Foreground integration: TestForegroundAcquireComposesProductionProbeWithAttach passes ×3. (d) B44 is carried as a stated bound with an owner (matrix:46, 244).
- I did not rerun the shipped mutant harness this round. My own plants are listed above.

## Surface rows
- lost-response idempotency: held (determinism rerun; E1 kill on the replay path)
- reconnect/multi-attach: held, from package green and census rows; I did not independently attack it this round
- socket substitution/custody: **broken** (custody-mask-pinned-single-bit-axis). Entry-scoped plants M3, M4 and M6 held.
- ownership-neutral attach: held (TestExecuteAttachDoesNotReadOrChangeSessionLease ×3); I did not attack it further
- registry/traceability: held
- trunk preservation/composition: held

## Rework scope (for the producer)
1. Replace the single-bit census with an EXHAUSTIVE mode enumeration per component class. For socket leaf, runtime `tmux` dir and root, cover all 64 values of the non-owner 6 bits with the owner triplet held at the admitted baseline, and add setuid/setgid/sticky. For ancestors, cover every writable combination with and without each special bit, with sticky as the only admit. Drive each row through Probe, Spawn and Execute as named subtests, and assert the literal `tmux_unsafe_socket_path` plus the detail, with zero Dial, spawn and dispatch.
2. Ship M2 and M5 above as harness mutants, plus a combination-admit narrowing for the socket leaf and for `ownerOnlyRuntimeDirMode`. Each must be KILLED by a test run alone.
3. Disposition N1 by pinning it or stating its unreachability with the reason.
4. The digest and README pin need re-derivation only if cited test names change. Keep `task-board.config.json` byte-identical to the base.
