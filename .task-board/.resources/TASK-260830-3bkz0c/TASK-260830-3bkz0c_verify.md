# TASK-260830-3bkz0c — recovery-run verification (RUN-260906-22beab)

Successor run after CR construction refused with
`change_request_base_authority_mismatch`: the committed candidate tree
`9b3f568b` (HEAD `afdba1d`) disagreed with the independently snapshotted
tree `292bc4242`. Diagnosis, derived from `git diff-tree` in both
directions: the snapshot already contained the full `internal/environ`
package AND the 9-line LOGBOOK `1347` entry, while the hand commit
`afdba1d` carried only the 11 `internal/environ` files and left the
LOGBOOK entry uncommitted. One-file delta, no code difference.

## Repair (no production change)

- `git add LOGBOOK.md && git commit --amend --no-edit -S`
- New HEAD `1296eccb0d19af3312b4a56ecb5dfc6f00c7d937`, single parent
  `d5ad5f6`, signed (ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`,
  `git verify-commit` good).
- `git rev-parse HEAD^{tree}` = `292bc4242de751821d59ee1e3e68c9f1670b4634`,
  byte-identical to the snapshotted tree. Worktree clean.
- Frozen packages untouched: `git diff d5ad5f6..HEAD --
  internal/sessadapter internal/dirnode internal/provider internal/provhost`
  is empty; the delta is 11 new `internal/environ` files + LOGBOOK only.
- Story branch synced with `--force-with-lease`; PR #35 head is now
  `1296ecc`, OPEN.

## Gates re-run directly by this run (no pipes, real exit codes)

| Gate | Exit |
|---|---|
| `go build ./...` | 0 |
| `gofmt -l internal/` (empty) | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `go test ./internal/environ/ -count=1` | 0 |
| `go test ./... -count=1` (19 packages ok) | 0 |
| `go test ./internal/environ/ -race -count=1` | 0 |
| `go test ./internal/environ/ -cover -count=1` (76.1%) | 0 |
| `go run ./internal/traceability/cmd/tracecheck` (contracts=60 sections=36 acceptance_cases=94) | 0 |

## Standing evidence (prior run, unchanged by this run)

- `TASK-260830-3bkz0c_boundary.md`: 7/7 AC rows driven through named
  production call sites; 4 live divergences ledgered (escaped-backslash
  surrogate split, arm splits, caller-rule equality).
- `TASK-260830-3bkz0c_mutation-log.md`: narrowing 6/6 killed
  (N1/N2/N3/N4/N6 + token-preserving T1); D1 existence-only; 0 survivors.
- Full-repo `-race` not re-run (exceeds one bounded call; prior run raced
  the touched package, re-confirmed above).
