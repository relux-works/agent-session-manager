# TASK-260830-1pbx0c Windows path rework results

## Outcome

The CR revision 5 Win32 path bypass is fixed at the `ParseAbsolutePath`
production entry. Native Windows drive-qualified and UNC components now refuse:

- Win32-reserved punctuation: `< > : " | ? *`;
- ASCII control characters 1 through 31;
- `CON`, `PRN`, `AUX`, `NUL`, `COM1` through `COM9`, and `LPT1` through
  `LPT9`, case-insensitively and with any extension.

The same validation is reached by `DecodeAbsolutePathJSON` and by
`AbsolutePath.MarshalJSON`. Existing drive/UNC, dot/parent, alternate-stream,
trailing-dot/space, length, NUL, and platform-context refusals remain in place.
Positive compatibility tests preserve ordinary names such as `COM0`, `COM10`,
and `CONSOLE`, and prove that Windows-only punctuation rules do not change
POSIX path validation.

This cycle changed only the common component decision in
`internal/scalar/path.go`, its production-boundary tests in
`internal/scalar/scalar_test.go`, and the existing README scalar capability
description. The final carry-forward comparison exits 1 as expected and lists
exactly `path.go` and `scalar_test.go` under `internal/scalar`; the other five
carried-forward scalar files remain byte-identical.

## Authority and scope

- Carry-forward archive SHA-256:
  `462775940304492d41238279e3e0b3ec461523785749ad7a7659e7d60037c387`.
  Before rework, all seven extracted files matched the worktree (`diff` exit 0).
- Pinned specification commit:
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- Downloaded pinned `SPEC.md` SHA-256:
  `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.
  Section 1.6 defines `absolute-path` as absolute and lexically normalized for
  the containing platform.
- Windows ground truth: task precondition
  `TASK-260830-1pbx0c_windows-ground-truth.md`.
- Platform authority:
  <https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file>.

The scalar package is read-only validation. No durable state is mutated, so
crash/idempotency evidence is not applicable. No `ax doctor` result or runtime
capability is added or advertised.

## Negative evidence

- Baseline expected-red:
  `go test ./internal/scalar -run '^TestWindowsAbsolutePathRefusesWin32InvalidComponentsAtEveryBoundary$' -count=1 -v`
  exited 1 before the production fix because the real parse, JSON decode, and
  marshal boundaries admitted reserved components.
- Narrowed-mutant expected-red: after temporarily narrowing DOS-device matching
  to bare names only, the same named test exited 1 on extension aliases such as
  `con.txt` and `NUL.txt`. This proves the extension bound by narrowing the gate,
  not by deleting it. The production line was restored through a reverse patch,
  and the immediate post-mutant rerun exited 0.

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused Windows path tests | 0 | `windows-path-focused.log` |
| Post-mutant named regression test | 0 | `windows-path-post-mutant.log` |
| `go test ./internal/scalar -count=1 -v` | 0 | `scalar-tests.log` |
| `go test ./internal/scalar -cover -count=1` | 0 | `scalar-coverage.log` — 89.6% |
| `go test -race ./internal/scalar -count=1` | 0 | `scalar-race.log` |
| `go test ./... -v -count=1` | 0 | `go-test-all.log` |
| `go test ./... -cover -count=1` | 0 | `go-coverage-all.log` — scalar 89.6% |
| `go vet ./...` | 0 | `go-vet.log` |
| `go build ./...` | 0 | `go-build.log` |
| `go generate ./internal/catalog` | 0 | `go-generate.log`; generated diff empty |
| Assigned Sections 1.6 and 10.1-10.4 tracecheck | 0 | `tracecheck-assigned.log`; `assigned_scopes=5` |
| Global tracecheck | 0 | `tracecheck-global.log` |
| `curator status --check` | 0 | `curator-status.log` |
| `gofmt -d` on changed Go files | 0 | `gofmt-diff.log`; empty |
| `git diff --check` | 0 | `git-diff-check.log`; empty |
| Generated catalog diff | 0 | `generated-diff.log`; empty |

`task-board validate` itself exited 0 but printed 264 pre-existing
`MISSING_ACTIVITY` issues for unrelated legacy board elements. The current task
was not listed. This is recorded as an anomaly, not reported as a green board
validation gate, and no foreign board scope was modified.
