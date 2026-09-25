# TASK-260830-1jmmqn recovery note (RUN-260924-9b0e9a): rev2 CR validation failure

## Failure

CR-TASK-260830-1jmmqn-2 revision 2 validation failed at command 4/30
(`go test ./... -count=1 -v`), exit 1, completed 2026-09-24T03:24:54Z.
The attached validation log is truncated (6,093,044 bytes omitted), so the
failing package is not visible in the resource.

## Diagnosis: environmental disk-full, not a code defect

1. **Tree identity.** The current worktree candidate tree, computed through a
   scratch `GIT_INDEX_FILE` (real index untouched), is
   `e7646ecd9b3d00f9d69ae1feb63985100f8ffed9` — byte-identical to the
   `tree_oid` the failed validation ran against. Same tree, different
   outcome: the code is not the variable.
2. **Same tree passed before.** rev1 (identical except cloneplan test files)
   passed the full 30-command validation at 2026-09-24T01:47:38Z, exit 0.
3. **Sibling failed in the same window, visibly disk-full.** TASK-260830-147hsj
   rev1 validation failed at 2026-09-24T03:08:04Z (~17 min earlier) with
   dozens of `TempDir: mkdir /var/folders/.../T/...: no space left on device`
   failures across its suite (visible in its validation-log resource because
   the truncation window lands later). Same host, same temp volume, same
   `go test ./...` gate.
4. **Known incident class.** The reviewer brief records that on 2026-09-23
   scratch copies filled this disk and broke three CR constructions.

Conclusion: the rev2 failure was a disk-full environmental failure during the
full-suite gate. No production or test change is required; the fix is to
re-run validation with free space (45 GiB available now, TMPDIR at 3 GB)
and retry the handoff.

## Re-verification by this run (same tree, exit codes real)

| Command | Exit | Evidence |
|---|---|---|
| `go test ./... -count=1` | 0, 46/46 `ok` (slowest: sessquery 311s, resumesmoke 234s, tracecheck 204s, cloneplan 52s) | `TASK-260830-1jmmqn_fullsuite-rerun.log` |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | this note |
| `test -z "$(gofmt -l ...)"` | 0, clean | this note |
| trunk freshness: `origin/main` = `0ca3e4c`, story base = merge-base | no refresh needed | this note |

Accepted from the unchanged rev2 evidence (tree-identical, per brief
standing order 10): mutant battery (124 KILLED + control SURVIVED x2 runs),
`-count=3` determinism, coverage 96.8%, importer grid, scratch-index
diff-check. Re-running the multi-hour mutant battery on a byte-identical
tree adds no signal.

## Change

None. No file in `internal/cloneplan/` or elsewhere was modified by this
run; the candidate remains the uncommitted `internal/cloneplan/` package
(24 paths, `task_delta` scope, no registry edit).
