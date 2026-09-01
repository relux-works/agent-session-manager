# TASK-260830-1pbx0c — rework cycle 3 results

## Delivered behavior

- `ParseTimestamp` accepts the published UTC leap second `1990-12-31T23:59:60.000Z`.
- The same value is accepted through JSON and text decoding; the exact wire text is retained.
- `Timestamp.Time()` maps the leap second to `1991-01-01T00:00:00Z`, because Go `time.Time` has no leap-second representation.
- Fabricated `:60` values are refused at the production entry and both decoders: an ordinary minute on the real date, an unpublished neighboring date, and 2026-12-31 (IERS Bulletin C 72 announces no leap second).
- The immutable table contains all 27 published positive UTC leap-second dates from 1972-06-30 through 2016-12-31.

## Assigned-scope ownership

- Registered ten scalar acceptance cases in `internal/traceability/ownership.v0.5.0.json`.
- Registered `internal/scalar` production declarations as scoped owners for Sections 1.6, 10.1, 10.2, 10.3, and 10.4.
- `tracecheck` reports 60 contracts, 36 normative-section inventory keys, 26 executable acceptance cases, 30 fixtures, 55 compatibility contracts, and five assigned scopes.
- `TestMainRejectsRenamedScalarSectionOwnerDeclarations` mutates the real owning Go declaration for each of the five bindings in an isolated repository and proves the production `main -> run -> VerifyAssignedSections` path refuses every mutant.
- The prior pinned-but-unowned negative probe was moved from newly owned Section 10.1 to still-unowned Section 10.5.

## Current authority and scope

The managed Story workspace remains based at `3679514c` because CR revision 2 left it dirty. The landed strict traceability authority is `c9e5290b`. Its catalog/specpin/tracecheck foundation was applied to this candidate without Git switch, rebase, merge, commit, or board-file changes. A direct content comparison against `c9e5290b` for unchanged authority files exited 0; task-specific deltas are README, scalar sources/tests, ownership bindings/digest, and traceability expectations/mutants.

Section 17.3 concerns durable immutable-data migration. `internal/scalar` is read-only and neither mutates durable state nor claims Section 17.3 migration ownership. No crash/idempotency evidence is applicable to the scalar operation itself. README adds no `ax doctor`, provider/platform availability, or runtime capability claim.

## Validation evidence

| Command | Exit | Result |
| --- | ---: | --- |
| Focused leap-second test before implementation | 1 | Expected red: real `:60` rejected by Go `time.Parse` |
| Focused ownership tests before digest repin | 1 | Expected red: canonical registry digest changed |
| `go test ./internal/scalar -run '^TestTimestampAcceptsPublishedLeapSecondsAndRefusesFabricatedOnes$' -count=1 -v` | 0 | Production/JSON/text positive and negative cases pass |
| `go test ./internal/scalar -count=1 -v` | 0 | All 15 top-level scalar tests pass |
| `go test ./internal/scalar -cover -count=1` | 0 | 89.0% statement coverage |
| `go run ./internal/traceability/cmd/tracecheck -section 1.6 -section 10.1 -section 10.2 -section 10.3 -section 10.4` | 0 | `assigned_scopes=5` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | Global ownership gate passes |
| First `go test ./... -count=1 -v` | 1 | Real rework failure: two old tests still expected Section 10.1 to be unowned |
| Rerun `go test ./... -count=1 -v` | 0 | Repository-wide suite passes after moving that negative probe to Section 10.5 |
| `go test ./... -cover -count=1` | 0 | Scalar 89.0%; all packages pass |
| `go vet ./...` | 0 | Clean |
| `go build ./...` | 0 | Build passes |
| `go generate ./internal/catalog` | 0 | Generated catalog remains current |
| `git diff --check` | 0 | No whitespace errors |
| Authority content comparison against `c9e5290b` | 0 | Unchanged imported authority files are byte-identical |

All commands ran directly as standalone processes without `tee`. Full logs and the two expected-red/rework-failure logs are attached separately.

## Sources

- Pinned AX v0.5.0 source commit: `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, Section 1.6 and Sections 10.1-10.4.
- IERS Bulletin C metadata: https://datacenter.iers.org/productMetadata.php?id=16
- IERS Bulletin C 72: https://datacenter.iers.org/data/html/bulletinc-072.html
