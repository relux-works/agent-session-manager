# TASK-260830-1bipsa — implement-cli-result-envelopes

Leaf 2 of 3 on `task-board/story/STORY-260830-2jylym`. Scope:
relux-works/agent-session-manager-spec@v0.5.0 (28bf96d7), §14.2, §15, §17.2.

## What shipped

`internal/cliresult` (3,438 production lines, 3,098 test lines), the Section
14.2 CLI Result contract:

| Element | Production entry point |
| --- | --- |
| Version registry and static per-tag selection | `VersionForCommand`, `RegisteredVersionForCommand` |
| Closed eight-member envelope, writer and reader | `New`, `Decode`, `Result.MarshalJSON` |
| Eighteen closed command bodies and six embedded types | `validateBody` and the per-tag validators |
| Identifier nullability and session-scope equality | `validateIdentifiers`, `validateNestedSessionScope` |
| §17.2 reader rules | `Decode` (major rejection first), `acceptsVersion` |
| §14.2 common flags and its two refusals | `ParseCommonFlags`, `AcceptsYes`, `AcceptsJSON` |
| Destructive-operation confirmation | `RequireConfirmation`, `ExpectationFlags` |
| stdout/stderr rendering boundary | `Emitter` (`Emit`, `Log`, `Progress`, `Prompt`) |
| Exact §15.2 process exit status | `ExitStatus`, `Emit` |

Design decision that removes a defect class rather than documenting it: the
writer and the reader share one validator over one value model. `New` marshals
the caller's body, runs it through the strict `canonicaljson` model, and
re-parses it, so the graph the validators checked is the graph the object
encodes and the writer structurally cannot emit an object its own `Decode`
refuses — the defect CR-TASK-260830-34elja-1 found in `axerror`.

## Two residuals from leaf 1's review, both closed

- **R2 — the "nineteen-row" miscount.** §15.2 has eighteen body rows
  (`[0, 2-17, 130]`). Four places said nineteen, one of them landed in `main`.
  All corrected, and the count is no longer written in prose:
  `TestExitStatusRegistryMatchesThePinnedTableRowForRow` measures the rows from
  `internal/specdoc` and compares the count and every status against
  `exitMeanings` and a reviewed literal set.
- **R1 — `minRedactableCause` unpinned.** Every cause in the corpus was ≥64
  characters, so 16 and 64 both left the suite green.
  `TestMinimumRedactableCauseIsPinnedAtItsBoundary` uses a cause of exactly 8
  characters that must be scanned and one of 7 that must be skipped, on the
  message and on a diagnostic value, plus a multi-byte fixture a byte-counted
  bound would fail. Verified: 7, 9 and 16 each redden.

## §15.2 and clause 14.2#6 — what is and is not claimed

§15.2 stays `unmeasured`: its table carries no RFC 2119 keyword, so the clause
scanner measures zero obligations and no clause can be enumerated. Its binding
stays on `internal/axerror/registry.go:ExitCodeFor`; the gap now records that
the mapping is implemented and measured against the pinned table, that
`internal/cliresult` maps a failure to that exact status, and that what is
missing is the process — no `ax` binary exists, and the only `os.Exit` calls in
the tree are the `exit(1)` paths of `cataloggen` and `tracecheck`.

For the same reason clause `14.2#6` — "the process exit status MUST equal that
error's exit_code" — is **not** claimed. `Emit` and `ExitStatus` return that
exact status, and the process-level half is disclosed as unowned.

## Measured evidence

| Metric | Before | After | Notes |
| --- | ---: | ---: | --- |
| Section bindings | 48 | 49 | `section:14.2` added |
| Clauses discharged | 9/394 | 17/403 | §14.2 at 8/9 |
| Acceptance cases | 53 | 65 | 12 new, all resolving to real declarations |
| `partial` bindings | 2 | 3 | §14.2, §15.1, §15.3 |
| `internal/cliresult` coverage | — | 95.3% | statements |
| `internal/axerror` coverage | 99.7% | 99.7% | unchanged |
| Registered command tags built | — | 18/44 | 8 clone, 14 directory, 4 terminal unbuilt |
| Registered versions built | — | 2/4 | 3.0.0 and 4.0.0 unbuilt |

Assigned-scope admission still refuses `-section 14.2` with its exact ratio,
and 14.2 is now in both refusal disclosure tables.

## Stated bounds (`cliresult.ContractBound`, asserted by its own test)

- CLI Result 3.0.0 and 4.0.0 are registered and not built; their tags are
  refused with `ErrUnimplementedVersion`, never emitted with an unchecked body.
- The eight `session.clone.*` tags select 2.0.0 — that selection is the §14.2
  rule implemented here — but their §14.1 bodies over §13.14 clone types are
  not built.
- The takeover adoption rule needs a session kind the body does not carry.
  `New` requires it; `Decode` cannot have it and exposes
  `VerifyTakeoverAdoption` rather than skipping a MUST silently.
- An absolute-path member is admitted when it is absolute on any supported
  platform, because a CLI Result names none. `VerifyDestinationPlatform`
  narrows it.
- The §18.1 total order of a `logs` event array is not checked.
- §17.2's same-or-lower-minor rule is implemented and unit-tested as
  `acceptsVersion`; the registry carries no two CLI Result versions sharing a
  major, so `Decode` reaches it only with equal versions today.

## Validation commands and real exit codes

| Command | Exit | Note |
| --- | ---: | --- |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `gofmt -l .` | 0 | no output |
| `go test ./... -count=1` | 0 | 13 packages |
| `go test ./internal/cliresult -count=1 -v` | 0 | 321 passing cases, `cliresult-test-01.log` |
| `go test ./... -cover -count=1` | 0 | `coverage.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `tracecheck-01.log` |
| `go run .../tracecheck -section 14.2` | 1 | **expected red**: 8/9 is `partial` and admission requires `full` |
| `python3 .temp/TASK-260830-1bipsa/mutants.py` | 0 | 50 mutants, 48/48 non-subsumed killed |

## Mutation harness

`mutants.py` edits exactly one gate per mutant, verifies the edit really
applied (removed text gone, written text present), verifies the mutant
compiles, runs the package suite, and requires a red. Two mutants are declared
`SUBSUMED` with the guard that refuses the same input earlier, each with a test
pinning the invariant that makes it subsumed:

- `capability-bound` — the capability vocabulary has exactly seven members, so
  a map of admitted names can never exceed the `[0..7]` count bound.
- `decode-required-nullable-present` — `decodeClosedDocument` already refuses a
  document missing any of the eight declared members.

One mutant looked killed and was not: `decode-tag-version-agreement`. A clone
tag inside a 1.0.0 envelope errors either way, because the unimplemented-body
refusal message also contains "selects CLI Result 2.0.0". The decisive case is
a 1.0.0 takeover body inside a 2.0.0 envelope, where nothing else in the reader
objects.
