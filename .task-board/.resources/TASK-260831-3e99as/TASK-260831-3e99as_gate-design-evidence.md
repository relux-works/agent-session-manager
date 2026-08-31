# AX task-board landing-gate configuration — design evidence

Date: 2026-08-31. Probed against accepted candidate tree
`892b4dc4c8453138a790ce85c97ebc9e0ec44f3e` (CR-TASK-260830-treeox-3), extracted
into a clean throwaway checkout so the probe sees exactly what a managed Story
worktree sees — i.e. **without** gitignored paths.

## Why this configuration is needed at all

AX has no `task-board.config.json`. Two consequences, both measured:

1. `worktree integrate` refuses `validation_not_configured`. The landing gate
   defaults to refusing; an absent suite is not an empty suite.
2. Every AX spawn ran `gpt-5.6-sol` at reasoning effort **medium** — the CLI
   default — because there are no `spawn.ceilings` and no
   `spawn.workload_classes`. Confirmed in
   `TASK-260829-1m2pv0_spawn-log_-implementer--developer--codex-_RUN-260828-280971.log`:
   `model: gpt-5.6-sol` / `reasoning effort: medium`.
   The source repository pins `high` for all eleven workload classes.

Accepted `CR-TASK-260830-treeox-3` additionally carries **no** `validation`
evidence at all, because none was configured when it was accepted. Landing this
config advances AX `main`, which forces the accepted CR to reparent and
revalidate — and that revalidation is what produces the missing evidence.
The ordering config -> reparent -> integrate is therefore mandatory.

## Commands: probe results

All commands were run on the candidate tree. Positive result, and for the
gating ones a mutant that the command must kill.

| # | Command | Positive | Negative (mutant killed) |
| ---: | --- | :-: | --- |
| 1 | `test -z "$(gofmt -l .)"` | pass | unformatted file -> fails |
| 2 | `go build ./...` | pass | — |
| 3 | `go vet ./...` | pass | — |
| 4 | `go test ./... -count=1 -v` | pass | — |
| 5 | `go test ./... -race -count=1` | pass | — |
| 6 | `go test ./... -cover -count=1` | pass | — |
| 7 | `go run ./internal/traceability/cmd/tracecheck` | pass | see below, two mutants |
| 8 | `go generate ./internal/catalog && git diff --exit-code -- internal/catalog/catalog_gen.go` | pass | stale generated file -> fails |
| 9 | `GOOS=linux GOARCH=amd64 go build ./...` | pass | — |
| 10 | `GOOS=windows GOARCH=amd64 go build ./...` | pass | — |
| 11 | JSON validity over all non-board `*.json` | pass | broken JSON -> fails |
| 12 | `task-board validate` | pass | — |
| 13 | `git diff --check` | pass | — |

`tracecheck` mutants, both killed with precise diagnostics:

- Renamed the owned production declaration `Current` in `internal/specpin/pin.go`:
  `acceptance case "source-pin-exact" production owner: declaration "Current" is
  absent from "internal/specpin/pin.go"`.
- Dropped one acceptance case from `ownership.v0.5.0.json`:
  `ownership registry projection digest 6b71fb95… differs from reviewed 66c846ef…`.

The ownership registry is itself digest-pinned, so the gate cannot be satisfied
by editing the registry to match the code.

## `curator status --check` is deliberately EXCLUDED

It cannot pass in the tree the gate runs against, and including it would break
every integration:

```
$ curator status --check      # clean candidate tree, no gitignored paths
ax: go-testing-tools not-installed
rc=1
```

AX `.gitignore` ignores `.agents/`, `.claude/skills/`, and `.codex/skills/` —
exactly the paths Curator owns. A managed Story worktree therefore never
contains them. Curator parity belongs to the developer workflow and to the
`Skillfile.json` review, not to a tree-bound landing gate. `Skillfile.json`
itself is still covered, by command 11.

## Cross-platform note

`GOOS=windows` cross-builds are a compile gate only. A real Windows worker is
available over SSH (alias `win`: Windows 10 Pro, PowerShell 5.1,
go1.25.5 windows/amd64, git 2.50.1) with a Go toolchain identical in version to
the local darwin/arm64 one, so genuine Windows execution is available for the
platform-adapter Stories in M1+ rather than only cross-compilation.

## Delivery shape

This config must land through a normal tracked, signed, reviewed PR rather than
through `worktree integrate`. Integrating it via a Story worktree would require
the landing gate it is itself introducing — a bootstrap paradox. It is a single
configuration file with no production-code delta, so proportional process is a
tracked task plus orchestrator-owned inline implementation and self-review,
with the deterministic probe evidence above as its outcome artifact.
