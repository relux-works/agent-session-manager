# TASK-260830-2ya5le Republish Evidence (RUN-260923-103c3f)

Republish continuation after the 2026-09-23 16:45Z CR-construction
failure (host disk full during snapshot; not a candidate defect).
Implementation NOT redone. Disk now has 34 GB free.

## Candidate identity

- Story branch: `task-board/story/STORY-260830-21bxa3`
- Tip: `0ca3e4c26e2b275212796657f785b9b450f6174e` == `origin/main`
  (`0ca3e4c`); tip descends from current trunk, so NO
  `refresh-candidate` was run.
- Candidate tree OID (via scratch `GIT_INDEX_FILE`, live index
  untouched): `cb244c15c2911d582c02564680c212fb00a9e27b`
- Scratch-index diff vs HEAD: 4 modified landed files
  (`internal/clonebundle/decode.go`, `event.go`, `rawmanifest.go`;
  `internal/environ/decode.go`; +67/-1) plus 18 new files under
  `internal/clonefidelity/` (6 production, TRACEABILITY.md, 10 test
  files, `testdata/mutant_harness.py`). Matches
  `TASK-260830-2ya5le_results.md` ("4 modified landed files + new
  internal/clonefidelity/", +67/-1) and the conformance matrix
  base (`0ca3e4c`). `git status` shows nothing else; nothing staged.

## Fast checks rerun on this tree (this run)

| # | Command | Exit |
|---|---------|------|
| 1 | `go build ./...` | 0 |
| 2 | `go vet ./...` | 0 |
| 3 | `GOOS=windows GOARCH=amd64 go vet ./...` | 0 |
| 4 | `go test -count=1 ./internal/clonefidelity/ ./internal/clonebundle/ ./internal/environ/` | 0 (`ok` x3: clonefidelity 2.856s, clonebundle 1.725s, environ 0.386s) |

Full configured suite, mutation battery, and race/fuzz gates were
already run green on this exact tree by the prior producer run (see
`TASK-260830-2ya5le_producer-evidence.tar.gz`, `MANIFEST.sha256`);
construction reruns the full suite itself. No source, test, config,
or environment identity changed since that evidence was recorded.

## Checklist

All 18 items verified true by the prior run and still true (tree
unchanged, fast checks green); left checked.
