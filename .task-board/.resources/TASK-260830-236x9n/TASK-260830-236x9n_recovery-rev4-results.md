# TASK-260830-236x9n recovery revision 4

## Outcome

- Recovered CR revision 3 after validation command 8/13 falsely treated the candidate's intentional `internal/catalog/catalog_gen.go` delta as post-generation drift.
- Added the production `cataloggen -check` mode. It regenerates bytes from the reviewed metadata and pinned contract lock, compares them with the requested output, and refuses stale output without rewriting it.
- Replaced the worktree Change Request gate with the one-process `cataloggen -check` production entry. This proves generated-output freshness in an uncommitted candidate tree without comparing the candidate itself to baseline `HEAD`.
- Added a positive identical-output check and a negative stale-output refusal test. README tool documentation now includes the exact check command.
- Preserved the generated catalog containing all 40 reviewed self-identity definitions / 46 schema-version rows.
- Added `.DS_Store` to the root `.gitignore` because Finder immediately recreated the removed file. `git check-ignore -v .DS_Store` now exits 0 and proves the next Change Request candidate excludes it.
- No durable product mutation, doctor result, migration transaction, or runtime capability claim was added.

## Validation and real exit codes

Green commands:

- `go test ./internal/catalog/cmd/cataloggen ./internal/cataloggen ./internal/catalog -count=1 -v` — exit 0. Includes `TestRunCheckRefusesStaleOutputWithoutRewritingIt`.
- `go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check` — exit 0 before and after regeneration.
- `go generate ./internal/catalog` — exit 0.
- `go test ./... -count=1 -v` — exit 0.
- `go test ./... -race -count=1` — exit 0.
- `go test ./... -cover -count=1` — exit 0; canonicaljson 82.5%, catalog 97.6%, cataloggen CLI 79.3%, cataloggen 83.9%, scalar 89.6%, specpin 85.1%, traceability 84.6%, tracecheck 87.5%.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `GOOS=linux GOARCH=amd64 go build ./...` — exit 0.
- `GOOS=windows GOARCH=amd64 go build ./...` — exit 0.
- `go mod verify` — exit 0 (`all modules verified`).
- `test -z "$(gofmt -l .)"` — exit 0.
- tracked JSON validation under `zsh` with `pipefail` — exit 0.
- `git diff --check` — exit 0.
- default `go run ./internal/traceability/cmd/tracecheck` — exit 0: 60 contracts, 36 normative sections, 29 acceptance cases, 30 fixtures, 55 compatibility contracts.
- assigned `tracecheck` for Sections 1.6 and 10.1-10.4 — exit 0 with 5 assigned scopes.
- `task-board validate` — exit 0. It also printed 263 pre-existing `MISSING_ACTIVITY` warnings for unrelated legacy board elements; this is recorded as an anomaly rather than described as clean output.
- `git check-ignore -v .DS_Store` — exit 0, matched `.gitignore:10`.

Expected-red commands, reported as failures:

- Production `cataloggen -check` against `.temp/TASK-260830-236x9n/stale_catalog.go` — exit 1 with `generated catalog is stale`; the stale file was not rewritten.
- `go run ./internal/traceability/cmd/tracecheck -section 10.999` — exit 1 with the expected nonexistent-section refusal.

## Handoff

The recovered candidate is ready for review. The board-managed Change Request validation suite should now exercise the same generated-output check without the false baseline-diff failure.
