# BUG-260918-354b03: resumesmoke-deadline-frozen-against-the-real-clock

## Description
internal/resumesmoke/support_test.go hard-codes smokeDeadline = 2026-09-18T00:00:00.000Z. checkParams validates that deadline against the fixture FAKE clock (smokeClock = 2026-09-17T00:00:00Z), so it looked like a future instant forever, but provhost/runner.go:351 wraps the provider call in context.WithDeadline on the REAL clock. From 2026-09-18T00:00Z the fake-provider child is cut off before it answers and TestSmokeThroughRealProviderProcess fails with provider_timeout, which turns validation commands 4, 5 and 6 (go test ./... in three forms) RED on every tree including trunk. Found by the independent reviewer RUN-260917-e5ea66 while reviewing TASK-260830-19bjfj story_final rev2; reproduced by the orchestrator at trunk 2fc6d50. The production code is right: a deadline that governs a real child process must be a real instant. The fixture is wrong to freeze it.

## Scope
internal/resumesmoke/support_test.go only; no production change.

## Acceptance Criteria
The smoke deadline is derived from the real clock so it cannot expire again; TestSmokeThroughRealProviderProcess passes at any wall-clock date; the frozen smokeClock still governs recorded facts so replay stays byte-identical; go test ./... is green on trunk.
