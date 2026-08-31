# TASK-260831-2susw4 rework outcome

## Candidate

- Base/candidate branch: `task-board/story/STORY-260831-2xdhn9` at base commit `3679514cdc5a` with a staged 15-file task-scoped candidate and no unstaged repository delta.
- The normative pin and generated catalog now carry the board-derived 24-entry union: Sections 1-20 and Appendices A-D.
- `TestPinnedNormativeScopeEqualsBoardStoryUnion` scans the tracked board Story `README.md` files, requires exactly 66 `Normative scope:` declarations, expands section and appendix ranges, and compares the derived union with the pin. This specifically prevents the earlier decimal-truncation and `Appendices A-D` survey omissions.
- The exact pinned v0.5.0 headings produce an immutable 157-identifier inventory with SHA-256 `a71f012f8c8a39baed38e70ee0175abd16bef712a1c3389ad0242f7b5e53ebeb`. The inventory digest is part of the source lock and is checked without changing the upstream tag, commit, or SPEC digest.
- `tracecheck -section` validates each exact section/range endpoint against that inventory before resolving its pinned top-level `source:*` ownership key. The owner must name a real production declaration and registered executable acceptance cases.
- Sections 10.1-10.4 and Section 12.1 now pass through the production `main -> run -> traceability.VerifyAssignedSections` path.

## Negative evidence

- Pre-rework `go run ./internal/traceability/cmd/tracecheck -section 12.1`: exit 1 because Section 12 was outside the incomplete pin.
- Pre-rework `go run ./internal/traceability/cmd/tracecheck -section 10.999`: exit 0, reproducing the forged-subsection acceptance defect.
- Red-first board-union regression test: exit 1 and named missing `[12 appendix-b appendix-c]` before the fix.
- Red-first forged-section production test: exit 1 because the nested production tracecheck incorrectly returned success before the fix.
- `TestMainRejectsOneNarrowedAssignedSectionBinding`: exit 0 after removing only `source:10`, re-pinning the isolated mutant, proving Section 10.1 refuses while unrelated Section 9.2 stays green.
- `TestMainRejectsSyntacticallyValidNonexistentV050Section`: exit 0; production tracecheck refuses `10.999` and accepts real `10.1`.
- Post-fix direct forged probe: tracecheck exit 1 with `assigned section "10.999" is not a real v0.5.0 section identifier` and no success output.

## Source identity

- Tag object type/OID: annotated tag `d3da6614a6c7bf119a88c9596a86c0853c22cfb9`.
- Peeled commit: `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- Exact `SPEC.md` SHA-256: `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.
- Extracted heading inventory: 157 identifiers, SHA-256 `a71f012f8c8a39baed38e70ee0175abd16bef712a1c3389ad0242f7b5e53ebeb`.

## Configured validation suite

All commands ran as foreground standalone processes; no command was piped through `tee`.

| # | Command | Exit |
| ---: | --- | ---: |
| 1 | `test -z "$(gofmt -l .)"` | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` | 0 |
| 5 | `go test ./... -race -count=1` | 0 |
| 6 | `go test ./... -cover -count=1` | 0 |
| 7 | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| 8 | `go generate ./internal/catalog && git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 |
| 9 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| 10 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| 11 | strict JSON parse over `git ls-files -z '*.json'` with `pipefail` | 0 |
| 12 | `task-board validate` | 0 |
| 13 | `git diff --check` | 0 |

Coverage was 85.1% for `internal/specpin`, 84.9% for `internal/traceability`, and 87.5% for the production tracecheck command package. `task-board validate` returned exit 0 while printing 264 inherited `MISSING_ACTIVITY` diagnostics; that anomaly is preserved in the attached gate log and was not represented as a clean diagnostic report.
