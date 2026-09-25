# TASK-260924-2zboyz — CR rev4 validation retry note (RUN-260924-a04a43)

## Why this retry exists

CR rev4 construction failed validation at command 4/30
(`go test ./... -count=1 -v`, exit 1): package
`internal/traceability/cmd/tracecheck` hit the 10-minute `go test`
timeout (`panic: test timed out after 10m0s`) under 49-way
full-suite contention. Gofmt/build/vet (commands 1-3) were green;
the run stopped before the remaining 26 commands.

## Root cause: load-dependent timing flake, no code defect

- The leaf (`internal/clonereadback`) has zero import edge from
  the failing package: `go list -deps` over
  `tracecheck`/`resumesmoke`/`tmuxserver` shows no
  `clonereadback` entry (rerun by this run).
- The same contention shape already appeared at rev1 (tracecheck
  600s under load, green alone) and rev3's CR validation passed
  on the same command without any leaf timing change — the
  outcome is machine-load dependent, not candidate dependent.
- This run changed NO source bytes: the worktree candidate is
  byte-identical to the immutable CR rev4 bytes (verified by
  applying `TASK-260924-2zboyz_change-request_rev4.patch` to a
  scratch tree and running `diff -r`: no differences).

## What this run reran itself (exit codes observed)

| Gate | Command | Exit | Detail |
|---|---|---|---|
| leaf suite | `go test -count=1 ./internal/clonereadback/` | 0 | ok 5.347s |
| leaf cover | `go test -count=1 -cover ./internal/clonereadback/` | 0 | 95.6% |
| importer set | owner suites `environ scalar canonicaljson clonebundle sessadapter` | 0 | 5/5 ok |
| vet | `go vet ./...` | 0 | clean |
| windows vet | `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | clean |
| gofmt | `test -z "$(gofmt -l ...)"` over tracked+untracked Go files | 0 | clean |
| determinism | `go test -count=3 ./internal/clonereadback/` | 0 | ok 15.310s |
| shard A0 | 15 fast pkgs | 0 | 15 ok |
| shard A1 | 15 fast pkgs | 0 | 15 ok |
| shard A2 | 15 fast pkgs | 0 | 15 ok |
| resumesmoke | alone | 0 | ok 85.012s |
| tmuxserver | alone | 0 | ok 67.846s |
| traceability | `traceability` + `tracecheck` | 0 | 2 ok, tracecheck 22.734s |

Full-suite total rerun by this run: 49/49 packages green, every
command exit 0, no contention timeout in any shard. The failed
gate (command 4) is therefore green on the exact candidate when
not starved by 49-way contention; the CR validation suite reruns
automatically on the rebuilt CR.

## Mutant spot-check (this run) + reuse statement

- Spot-reran via the committed harness
  (`python3 -B testdata/mutant_harness.py run`, one raw log per
  row): `N-seal-build` KILLED, `N-seal-decode` KILLED,
  `N-control-neutral` SURVIVED, `failures=0`. Kill mechanisms
  audited from the raw logs: genuine `--- FAIL` in the named
  killer with subprocess exit 1, control exit 0 — no
  compile-error kills. Tree verified restored afterwards.
- The full 53-row battery ×2 (43 narrowings killed, control
  survived, 9 plants red, failures=0) is REUSED from the rev4
  evidence tar: the candidate is byte-identical to the tree that
  evidence was recorded on (verified above), so the binding
  holds per the unchanged-identity rule.

## Hygiene and scope (this run)

- `git status`: only `?? internal/clonereadback/`; zero tracked
  diff; no `.pyc`/`__pycache__` anywhere in the worktree.
- Trunk unmoved at `6d3bff9` (`origin/main` identical) — no
  `refresh-candidate` run (never on a clean-room check; the
  candidate already descends from current trunk).
- No registry edit; `task_delta` scope unchanged (21 new files
  under `internal/clonereadback/`, matching the CR file list).
- Implementation spot-check vs the rework brief: report entries
  take ONLY `ValidatedReadBack` siblings,
  `checkSiblingsSealed` runs first at both entries, read-back
  entries mint the seal, regression + AST-guard + no-bytecode
  tests present with the names the results doc claims.
