# TASK-260830-2it6xy Implementation and Validation

## Outcome

Implemented a read-only normative source pin in `internal/specpin`.

- Upstream repository: `relux-works/agent-session-manager-spec`
- Release/tag: `v0.5.0`
- Annotated tag object: `d3da6614a6c7bf119a88c9596a86c0853c22cfb9`
- Peeled commit: `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`
- `SPEC.md` SHA-256: `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`
- Pin manifest SHA-256: `edd67e84cb173fb66efb9d719b28ae02cf7cd5c7c1462551f5c7667b35cf78b6`
- Ordered active contract rows: 60
- Historical v0.4.3 contract rows after the exact compatibility projection: 55
- Shipped upstream fixture identities pinned: 3

Production entry points `specpin.Current`, `specpin.Verify`,
`Manifest.ContractsForRelease`, and `Manifest.Fixture` fail closed on malformed,
partial, unknown, substituted, contract-drifted, fixture-drifted, or byte-different
pin data. Repeated reads return isolated values.

This scope creates no durable-state mutation, so crash recovery evidence is not
applicable. It adds no `ax` command, doctor result, conformance-target declaration,
or provider/platform/backend capability claim.

## Source Evidence

The source was checked out from the immutable annotated tag. Both the tag and
peeled commit verified with Ivan Oparin's configured SSH signing identity.
An independent comparator parsed the Section 1.5 table from the tagged `SPEC.md`
and matched all ordered names, URNs, and version lists against the lock. It also
matched each shipped JSON fixture's declared `fixture` ID and file SHA-256.

The pin-level positive and negative/refusal cases pass. This task does not claim
that future product conformance cases described inside the upstream fixtures have
executed; those belong to their owning implementation and acceptance tasks.

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| Expected-red `go test ./internal/specpin -run TestCurrentPinsPublishedV050Source -count=1` before setting the lock digest | 1 | Expected refusal: `lock digest is not TODO` |
| `go test ./... -v -count=1` | 0 | Positive, narrowed identity/version/fixture/capability refusals, partial-read refusal, compatibility, and idempotency pass |
| `go test ./... -cover -count=1` | 0 | 83.0% statement coverage |
| `go test -race ./... -count=1` | 0 | Race detector passes |
| `go build ./...` | 0 | All Go packages compile |
| `go vet ./...` | 0 | No findings |
| `gofmt -l internal/specpin/pin.go internal/specpin/pin_test.go` | 0 | Empty output |
| `jq empty internal/specpin/v0.5.0.lock.json` | 0 | Valid JSON |
| `git diff --check` | 0 | No whitespace errors |
| `curator status --check` | 0 | Managed `go-testing-tools` is up to date |
| `task-board validate` | 0 | Board valid |
| Upstream tag/commit/signature/digest/comparator checks | 0 each | See `source-pin-validation-01.log` |

## Evidence Files

- `go-test-01.log`
- `go-cover-01.log`
- `go-build-01.log`
- `go-race-01.log`
- `go-vet-01.log`
- `source-pin-validation-01.log`
- `repo-validation-01.log`

## Anomalies

- The first staged `git diff --cached --check` exited 2 because seven attached
  raw logs had a newly added blank line at EOF. Their task-scoped `.temp`
  sources were corrected, the existing board resources were updated through
  `task-board resource update`, and the staged diff gate was rerun.
- One attempted batch `check_item` mutation exited 1 because a single activity
  event cannot represent multiple checklist changes. It made no checklist
  changes; all six items were then checked through separate mutations, each
  exiting 0.
- The installed `project-management` skill references
  `references/negative-evidence.md`, but that file is absent. The assignment
  embedded the applicable Evidence Honesty Contract, which was followed.
- Neither the installed `task-board` CLI nor the repository provides a
  `logbook` command/artifact. Important findings and decisions are recorded in
  the task Notes and this outcome instead; no parallel board file was created.
