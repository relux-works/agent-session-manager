# TASK-260830-treeox developer evidence

## Outcome

The repository now has a fail-closed specification-to-code ownership gate.
traceability.VerifyRepository is the production library entry point and
go run ./internal/traceability/cmd/tracecheck is the headless CI entry point.
.github/workflows/ci.yml invokes it before generation, tests, vet, and build.

The reviewed ownership registry covers:

- 60 v0.5.0 contract rows;
- 17 pinned or catalog-referenced normative section keys;
- 15 executable acceptance cases;
- 30 exact fixture identities or catalog fixture anchors;
- all 55 v0.4.3 compatibility contracts as an owned subset.

Every owner resolves through Go AST inspection to a concrete production
declaration and executable test declaration. The gate also re-verifies the
exact source lock, reviewed catalog metadata, and committed generated catalog.
It refuses narrowed, duplicate, absent, malformed, unreadable, stale, forged,
or self-minted evidence. Registry semantics are bound by canonical SHA-256.

The gate is read-only. Its idempotency/isolation test verifies two identical
checks return identical reports without changing any repository bytes. It
does not expose an ax command, doctor result, or availability/support field.
Existing durable catalog operations retain their exact idempotency scopes and
crash/lost-result recovery evidence, exercised by the targeted suite.

## Task delta

- .github/workflows/ci.yml
- internal/traceability/traceability.go
- internal/traceability/traceability_test.go
- internal/traceability/ownership.v0.5.0.json
- internal/traceability/cmd/tracecheck/main.go
- internal/traceability/cmd/tracecheck/main_test.go
- README.md ownership-gate and tool documentation

The Story worktree already contained orchestrator-managed staged index state
for the accepted catalog task. It was preserved. Relative to Story HEAD
6d602ba, this task changes only the paths above; go generate left the accepted
catalog output byte-identical.

## Validation and real exit codes

| Command | Exit | Evidence |
| --- | ---: | --- |
| Production tracecheck | 0 | tracecheck-final-02.log; exact counts 60/17/15/30/55 |
| Relevant source/catalog/traceability package tests | 0 | targeted-final.log; positive, refusal, compatibility, durable recovery, isolation |
| go generate ./internal/catalog | 0 | go-generate-final.log; empty output, no generated drift |
| go test ./... -count=1 -v | 0 | go-test-all-final.log |
| go test ./... -count=1 -cover | 0 | go-test-cover-final.log; traceability 80.2%, tracecheck 80.0% |
| go test ./... -count=1 -race | 0 | go-test-race-final.log |
| go vet ./... | 0 | go-vet-final.log; empty output |
| go build ./... | 0 | go-build-final.log; empty output |
| Ownership JSON validation | 0 | json-validate-final.log; empty output |
| git diff --check | 0 | git-diff-check-final.log; empty output |
| Traceability gofmt listing | 0 | gofmt-final.log; zero bytes, no unformatted files |

## Meaningful expected-red proof

The production CLI was run against a task-local registry mutant that removed
only the final contract key while leaving the owner group non-empty. It exited
1 and identified Session Directory Query
[urn:ax:schema:session-directory-query] as having no implementation owner.
This is a narrowing mutant, not a delete-the-entire-gate mutant. See
meaningful-red-01.log.

## Environment anomaly

The assignment referenced
project-management/references/negative-evidence.md, but that file is absent
from all installed project-management skill trees. The assignment's embedded
Evidence That Counts contract was available and followed; the failed probe and
tool versions are recorded in tool-readiness.md. This did not block product
implementation or validation.
