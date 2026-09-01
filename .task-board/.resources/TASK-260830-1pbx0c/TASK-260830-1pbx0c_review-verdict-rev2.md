# TASK-260830-1pbx0c review verdict — CR revision 2

## Verdict

Changes requested. Route `TASK-260830-1pbx0c` to `to-dev` for a new Change
Request revision.

Reviewed candidate: `CR-TASK-260830-1pbx0c-2` revision 2, base
`3679514cdc5a73adf8b76bd504e47e2623a00c9b`, candidate tree
`63d68267466bca3afd54a9c4588ea1263c39fccb`. The attached patch SHA-256 was
independently verified as
`905e886d09c8933ae7b8eaf9f6f49823555f849702c9bbb657489f3ce6d81b4e`, and a
temporary-index snapshot of the current managed worktree produced the exact
candidate tree OID.

Revision 2 correctly fixes both revision 1 findings: lowercase/unbounded RFC
3339 fractional syntax now reaches `ParseTimestamp`, and
`json.Marshal(BoundedInteger{})` refuses absent construction context while an
explicit `[0,0]` interval remains valid. The producer's expected-red log
reproduces both old defects. The following independent findings remain.

## Findings

### 1. High — the timestamp gate rejects a real RFC 3339 leap-second instant

AX v0.5.0 Section 1.6 requires real UTC RFC 3339 calendar instants with at
least millisecond precision. RFC 3339 Sections 5.6–5.8 permit second `60` at a
real inserted leap second and publish `1990-12-31T23:59:60Z` as an example.
Adding the AX-required fractional part yields the valid value
`1990-12-31T23:59:60.000Z`.

`internal/scalar/time_digest.go:61-69` delegates calendar validation to Go's
`time.Parse(time.RFC3339Nano)`, which rejects every second-60 value. This is a
narrowed-gate defect: the parser correctly refuses a fabricated leap second in
an ordinary minute, but also refuses the real published instant.

The reviewer probe drove the production `scalar.ParseTimestamp` entry point:

```text
=== RUN   TestRFC3339LeapSecondBoundary/published_RFC_example_is_a_real_UTC_instant
ParseTimestamp("1990-12-31T23:59:60.000Z") error = timestamp: must identify a real calendar instant: invalid AX scalar, want acceptance
=== RUN   TestRFC3339LeapSecondBoundary/ordinary_minute_cannot_mint_a_leap_second
--- PASS: TestRFC3339LeapSecondBoundary/ordinary_minute_cannot_mint_a_leap_second
```

Required rework:

- validate real RFC 3339 leap seconds without accepting arbitrary `:60`
  timestamps;
- add positive production-boundary coverage for at least the published RFC
  example with AX millisecond precision and negative coverage for a fabricated
  leap second;
- drive `ParseTimestamp`, JSON decode, text decode, and any exported conversion
  surface while preserving revision 2's arbitrary fractional precision,
  lowercase `t`/`z`, impossible-date, sub-millisecond, and non-UTC behavior.

Normative sources:

- AX v0.5.0 Section 1.6:
  <https://github.com/relux-works/agent-session-manager-spec/blob/28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c/SPEC.md#L208-L244>
- RFC 3339 Sections 5.6–5.8:
  <https://www.rfc-editor.org/rfc/rfc3339.html#section-5.6>

### 2. Medium — the traceability gate does not bind the task's Section 10 scope

The task scope names Sections 10.1–10.4, and the implementation evidence says
those sections are represented. The reviewed traceability inputs do not encode
that claim:

- `internal/specpin/v0.5.0.lock.json:14-20` limits `normative_scope` to Sections
  `1`, `17`, `20`, Appendix A, and Appendix D;
- `internal/traceability/ownership.v0.5.0.json:449-476` attaches the new scalar
  cases only to the existing `source:1`/`source:17` group and has no
  `source:10` owner;
- the production `tracecheck` therefore reports the same 17 normative sections
  and can remain green if every Section 10 scalar binding is absent.

This is the standard shape **check present but scope untested**: the gate proves
that registered ownership rows are internally consistent, not that the exact
Sections 10.1–10.4 scope assigned to this task has an implementation owner and
executable acceptance cases.

Required rework:

- bind Sections 10.1–10.4 through the repository's reviewed source-section or
  contract ownership model;
- attach the relevant scalar acceptance cases to that exact scope; and
- add a narrowed negative traceability test that fails when the Section 10
  binding is removed while the unrelated Section 1/17 bindings remain.

## Independent validation

| Command | Result |
| --- | --- |
| Candidate patch SHA-256 and temporary-index tree snapshot | pass; exact advertised digest and candidate tree |
| Revision 2 production-boundary probe | pass for both revision 1 fixes; expected failure for real leap-second acceptance |
| `go test ./internal/scalar -count=1 -v` | pass |
| `go test ./internal/scalar -cover -count=1` | pass; 88.4% statements |
| `go test ./... -count=1 -v` | pass |
| `go test ./... -cover -count=1` | pass; all packages green, scalar 88.4% |
| `go vet ./...` | pass |
| `go build ./...` | pass |
| `go run ./internal/traceability/cmd/tracecheck` | pass; 60 contracts, 17 normative sections, 25 cases, 30 fixtures, 55 compatibility contracts |
| Exact candidate `git diff --check` | pass |

The first probe invocation is not counted: it hit a reviewer-harness package
name collision with the prior revision's retained probe. The rerun used the
existing package name and produced the evidence above.

No durable-state crash/idempotency finding applies to this read-only scalar
package. README does not advertise a doctor result or runtime capability, but
its claim of real RFC 3339 timestamp validation remains unsupported until
finding 1 is fixed. The installed `task-board` exposes no separate logbook
command; these findings are persisted in this task outcome and task notes.
