# TASK-260830-2ya5le checkpoint preconditions — CR rev3

Observed for integration run `RUN-260923-2126f3`.

## Accepted candidate and tree identity

- `task-board worktree status STORY-260830-21bxa3 --json` reported `CR-TASK-260830-2ya5le-3` revision 3 as `accepted`.
- The accepted candidate tree OID is `9870fb42e454a1163b4fc6abd778210260cf84d9`; the current uncommitted worktree was independently materialized through `.temp/TASK-260830-2ya5le/republish/candidate.index` using `GIT_INDEX_FILE`, and `git write-tree` returned the same OID.
- The scratch-index diff contains 22 paths and matches the accepted CR `changed_paths` list exactly. The real index was not written.
- The Story branch tip and checkpoint are both `0ca3e4c26e2b275212796657f785b9b450f6174e`; the candidate remains dirty and uncommitted, as required.

Changed paths (22):

```text
internal/clonebundle/decode.go
internal/clonebundle/event.go
internal/clonebundle/rawmanifest.go
internal/clonefidelity/TRACEABILITY.md
internal/clonefidelity/decode.go
internal/clonefidelity/doc.go
internal/clonefidelity/errors.go
internal/clonefidelity/fixtures_test.go
internal/clonefidelity/oracle_crosscheck_test.go
internal/clonefidelity/record.go
internal/clonefidelity/record_grid_test.go
internal/clonefidelity/report.go
internal/clonefidelity/report_aggregates_test.go
internal/clonefidelity/report_branches_test.go
internal/clonefidelity/report_gates_test.go
internal/clonefidelity/report_members_test.go
internal/clonefidelity/rowcount_internal_test.go
internal/clonefidelity/testdata/mutant_harness.py
internal/clonefidelity/utf8_reflection_test.go
internal/clonefidelity/vocab.go
internal/clonefidelity/vocab_oracle_test.go
internal/environ/decode.go
```

## Trunk observation

- `git fetch origin main`: exit 0.
- `origin/main` is `0ca3e4c26e2b275212796657f785b9b450f6174e`, equal to the recorded base/checkpoint. The branch tip descends from it, so no candidate refresh was needed.

## Fresh fast checks

| Command | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows GOARCH=amd64 go vet ./...` | exit 0 |
| `go test ./internal/clonebundle ./internal/clonefidelity ./internal/environ` | exit 0; all three packages passed |

The full configured suite was not rerun in this integration run; Change Request construction reruns it. No tracked candidate files were edited for this precondition check.
