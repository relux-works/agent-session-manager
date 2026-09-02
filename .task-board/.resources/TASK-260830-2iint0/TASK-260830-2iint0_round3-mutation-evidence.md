# TASK-260830-2iint0 round-3 mutation evidence

All mutants were applied individually, run uncached against the named
production-path test, and removed before the restored green gates.

| Mutant | Production boundary | Result |
| --- | --- | --- |
| Interpret `AX_SESSION_SECRET` only after flag and documented-env misses | `selectCandidate` default branch via `ResolvePaths` | exit 1; platform-default lookup set reported 11 names instead of the derived 10 (`TASK-260830-2iint0_mutant-unknown-env-round3.log`) |
| Replace `Load(inputs, overrides)` with `Load(inputs, nil)` | `LoadOS` | exit 1; explicit flag roots no longer beat populated environment roots (`TASK-260830-2iint0_mutant-loados-overrides-round3.log`) |
| Remove `inputs.LookupEnv == nil` refusal | `ResolvePaths` via `Load` | exit 1 with the expected nil-function panic, rather than a false green (`TASK-260830-2iint0_mutant-nil-lookup-round3.log`) |
| Render wrapped `%v` details in `Error.Error` | common error rendering boundary | exit 1; a machine-local credential path appears and is detected (`TASK-260830-2iint0_mutant-error-renderer-round3.log`) |

The reviewer’s two site-local no-echo mutants (root inspection and config read)
are now provably subsumed by `Error.Error`: even if either site wraps a selected
path, the common renderer cannot expose `Err` details. The attacked clause is
therefore the single boundary that controls all current 19 `&Error{}` sites and
future sites. `TestErrorFormattingNeverEchoesWrappedDetails` names this
subsuming check and also verifies that error identity remains available through
`Unwrap`.

The anti-overclaim gate was also re-run: `tracecheck -section 6.1` exits 1 with
`has no scoped implementation owner`. The failure is expected because this
task owns Section 3.2 path resolution, while unknown-key and secret-field
schema refusals remain outside this slice.

