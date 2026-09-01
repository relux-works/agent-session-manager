# TASK-260830-8x76g1 logbook

## 2026-08-31 — native fuzz evidence was rejected by traceability

After the fuzz functions were registered as acceptance tests, the assigned-scope tracecheck exited 1 because the production source checker admitted only `Test*` declarations. This meant a real Go-native `Fuzz*` target could run green but could not be attached to spec ownership.

Decision: admit only executable Go naming families `Test*` and `Fuzz*`, preserving the Go rule that the first suffix rune must not be lowercase. Do not admit `Benchmark*`, `Example*`, or arbitrary helpers as conformance evidence. A regression test proves `FuzzBoundary` is admitted while `Fuzzhelper` and `BenchmarkBoundary` are refused.

## 2026-08-31 — board validation legacy anomaly

`task-board validate` exited 0 and printed 262 inherited `MISSING_ACTIVITY` diagnostics for legacy board elements. The current task was not listed. The raw output is retained in the validation archive. This is recorded as an anomaly rather than described as a diagnostic-free validation result.

## Scope boundary

The packages and fuzz targets are read-only. They do not publish immutable objects, advance references, implement migration rollback, or mutate durable state, so no crash/recovery claim was made. README wording remains explicit about those unavailable surfaces.

## 2026-08-31 — reviewer defeated the closed-shape boundary

Review of Change Request revision 1 found that the production identity entries validate generic JSON and the omit-self digest contract but do not validate the selected schema's closed shape. A throwaway test against the exact candidate tree proved that `CalculateObjectIdentity` and `VerifyObjectIdentity` both admit a Blob Descriptor with an unknown top-level member, a nested BlobChunk with an unknown member, and a nested chunk with invalid bounds. This violates the task's recursive-closed-shapes acceptance item and the exact Section 10.2 contract. The verdict was routed to implementation rework; detailed evidence is in `TASK-260830-8x76g1_review-verdict.md` and `TASK-260830-8x76g1_review-closed-shape-probe.log`.
