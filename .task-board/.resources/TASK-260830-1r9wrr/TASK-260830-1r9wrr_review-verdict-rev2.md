# TASK-260830-1r9wrr — CR rev2 review verdict

**Verdict: changes_requested → to-dev.**
**repeat-of: CR-TASK-260830-1r9wrr-1:F3** (R2-F1 below); **CR-TASK-260830-1r9wrr-1:F7** (R2-F2, partial closure).
Reviewer: `RUN-260907-cf1098`, Codex `gpt-6-astra`, high.

The event census still admits an uncensused unknown-v1 handler. Two compiling neighboring plants survive every shipped package gate while a new production-entry probe demonstrates the wrong lifecycle transition. This is the second consecutive F3-class finding: route through the missing-gate workflow, rather than treating it as another unrelated vector. No external or human-only blocker exists.

## Candidate and scope

Board `worktree status` confirms CR `CR-TASK-260830-1r9wrr-2`, revision 2, base `0d9d0ad5acee26fed7bff78b605a07ff5230242c`, candidate tree `6bdd953fb044b5b5fad568f94fcff86477e9d7fb`, repository delta present. The attached patch SHA-256 is `e0c2d0bdfae624af1fb1fd0aed313586681a6e1eeee8f7e11c18605b32637223`, verified from the materialized resource. Detached-index write-tree matches before and after this review. HEAD remains the checkpoint; `HEAD..main` is zero commits in this workspace's local refs, not a claim about current remote main.

The cancelled review left no candidate mismatch. This reviewer changed no managed production, test, documentation, or real-index file. All mutations and temporary probes ran in `git archive` copies of the candidate under `.temp/TASK-260830-1r9wrr/review-rev2-astra/`, with fresh copy-local backups and byte restoration. `integrity-final.json` records the final comparisons. No commit, branch switch, integration, or upstream #176/#177 work occurred.

Exact CR paths (16):

```
LOGBOOK.md
README.md
internal/sessstate/arms_test.go
internal/sessstate/census_plants_test.go
internal/sessstate/census_state_test.go
internal/sessstate/census_test.go
internal/sessstate/decode.go
internal/sessstate/doc.go
internal/sessstate/fixtures_test.go
internal/sessstate/negative_test.go
internal/sessstate/project.go
internal/sessstate/project_test.go
internal/sessstate/purity_test.go
internal/sessstate/reduce_test.go
internal/sessstate/sessstate.go
internal/sessstate/winner_test.go
```

**AC-row coverage: 10 of 10 rows driven.** The previous verdict's ten named production-call/test mappings remain valid and unchanged, as reproduced in the current producer outcome. The new full suite passes. These findings concern holes inside the mapped scope, not missing AC rows. `internal/sessrepo` and `internal/provhost` remain outside the delta.

## R2-F1 — High: event-type dataflow escapes the census

**repeat-of: CR-TASK-260830-1r9wrr-1:F3**
**Shape: bypass path around the check; unclassifiable site treated as nothing to inventory.**

`internal/sessstate/census_state_test.go:383-430`, `deriveHandledEventTypes`, recognizes a switch only when its tag is syntactically a `.Type` selector; its equality scan has the same restriction. It does not fail closed when `event.Type` is assigned to a variable or passed to a helper. The new plants in `census_plants_test.go:171-219` exercise direct-selector shapes only.

One line in `effect`'s default arm is changed from `return nil` to `return fold.reviewExtra(event, name)`. A new production file contains:

```go
func (fold *chainFold) reviewExtra(event Event, name string) error {
    kind := event.Type
    switch kind {
    case "session.reaped":
        return fold.step(StateIdle, name)
    }
    return nil
}
```

This is PB. PC goes one step further: `effect` passes `event.Type` to `reviewExtra(kind string, name string)`, whose switch is over `kind`. Both keep all existing searched-for tokens, 24 registered switch cases, refusal sites, and state constants intact. Both are wired into `Reduce → apply → effect → reviewExtra → step`, not dead helpers.

| Live production plant | Build | Behavior tests only | Producer `-run '.*'` gate | Unfiltered package gate | Independent unknown-v1 probe |
| --- | --- | --- | --- | --- | --- |
| PA: original direct `switch event.Type` | PASS | PASS | FAIL, census | FAIL, census | FAIL: idle |
| PB: `kind := event.Type; switch kind` | PASS | PASS | PASS | PASS | FAIL: idle |
| PC: pass `event.Type` into string helper | PASS | PASS | PASS | PASS | FAIL: idle |

**Shipped-gate kill ratio: 1 of 3 reviewer event-dispatch plants, entirely census-only; behavioral kills: 0 of 3. PB and PC survive.** No NOT_APPLIED or COMPILE_FAIL is included. `TestReviewerUnknownV1Neighbor` is added only after measuring the shipped gates, to avoid counting reviewer-added coverage as delivered coverage. It passes on the untouched candidate, then fails on each plant through canonical fixture construction and the real `Reduce` entry:

```
unknown-v1 session.reaped: state="idle"
unknown v1 changed state: got "idle" want running
```

Pinned normative commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, SPEC §5.2 lines 1829–1830, requires unknown major-v1 types to leave state unchanged. The existing `TestReduceTreatsUnknownV1TypeAsInert` drives `session.future_probe`; it cannot catch this other spelling. This finding is a proven completeness-gate failure, not a claim that the unmutated candidate already dispatches `session.reaped` incorrectly.

Required rework: make event-type use single-owned or fail closed on aliasing/escaping/unclassifiable dispatch; keep PB and PC as live production control plants. The composed behavioral suite must run with the census. Do not close the finding by adding only `session.reaped` to a hand-picked behavioral list. Correct README's exhaustive dispatch claim and outcome/LOGBOOK's PA classification: PA was killed by `TestEventHandlingIsCensused`, not by behavior. A narrowed test selector is not proof of a behavioral kill when that selector still runs census tests.

Evidence: `reviewer-plants.py`, `reviewer-probes.go.txt`, `reviewer-plants-results.json`, `PA_direct_switch-*.log`, `PB_var_binding-*.log`, `PC_helper_argument-*.log` in the attached evidence archive. `reviewer-baseline-probes.log` is the control.

## R2-F2 — Low, required: empty-chain Local remains an unstated, unpinned bound

**repeat-of: CR-TASK-260830-1r9wrr-1:F7 (Local half).**

Union validation is repaired, but `applyLocalLease` still returns on `!fold.haveLease` at `sessstate.go:996`, even when a well-formed union reports a winner. `doc.go:81-87`, README, and `TestReduceEmptyChainValidatesUnion` cover Union only. No candidate test supplies Local on that empty-chain path.

`TestReviewerEmptyChainLocalBound` drove ten combinations through `Reduce`: empty/nonempty valid Union against absent, smaller, larger, malformed, and partial Local values. Every combination returns `creating`, unknown local role, and only the union-selected winner; a malformed Local is ignored without error. This may be the correct bound because no authoritative process/lease exists. The requested remedy need not add validation: explicitly state that empty-chain Local is ignored (including its validation/role consequences) and commit a test for that chosen bound, including with a reported union winner. Do not claim F7 wholly closed on Union-only evidence.

Evidence: `reviewer-baseline-probes.log`, `reviewer-probes.go.txt`. These reviewer probes describe current behavior and pass; this is missing stated-bound/test evidence, not an assertion that creating must become stale.

## Closures and independent validation

- **F1 closed.** Both new duplicate-off-chain tests pass; N13 reintroduces the old bug and fails those tests. Independent probes also preserve exactly one named union-supersedes conflict, winner C, running state and divergent-history warning across four interleavings containing repeated and distinct off-chain leases.
- **F2 closed.** N14 fails the new greater-epoch/greater-ID tests. Additional Local-vs-Union probes cover lower epoch, same-epoch lower ID, equality and higher epoch. The pinned §5.7 definition is a losing fencing token; §5.3 supplies greatest tuple ordering, supporting `< 0` as the stale condition. Local does not rewrite the reported winner.
- **F4 closed.** Re-derived 69 refusal sites: 39 sessstate + 28 decode + 2 project; baseline invcore-derived census agrees in both directions. README and current outcome agree. Current coverage re-derived as 92.2% for sessstate.
- **F5 closed.** Full producer battery rerun: 28 applied, 28 killed in its selected roster, with 22 behavioral, 3 census-only (C1/T1/T2), 3 audit-only (A1/A2/A3). Controls 2 of 2 behave as intended: XN compiles and passes both gates; XK compiles and fails `TestProjectLeavesStoppedPastTakeoverUnstaled`. Thus 28/28 is reproducible for that roster, but does not establish that every production gate/dispatch is covered. The raw harness still labels C1/T1/T2 behavioral because it includes census tests; this review corrects the classification. Neither that label nor PA's red gate counts as a behavioral kill.
- **F6 closed.** No current `scalar.ParseUUIDv7` delegation or audit row remains; `environ.CheckUUIDv7` owns that grammar. F7 Union half is closed; Local half remains R2-F2.

| Rerun in this reviewer session | Result | Full log |
| --- | --- | --- |
| `go test ./internal/sessstate -count=1 -v` | exit 0 | baseline-01.log |
| `go test ./... -count=1` | exit 0, 25 package result rows | full-suite-01.log |
| `go test ./... -cover -count=1` | exit 0; sessstate 92.2% | coverage-01.log |
| `go test -race ./internal/sessstate -count=1` | exit 0 | race-01.log |
| `go build ./...` | exit 0 | build-01.log |
| `go vet ./internal/sessstate` | exit 0 | vet-01.log |
| `gofmt -l internal/sessstate`, `git diff --check` | clean | format-01.log, diff-check-01.log |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0 | tracecheck-01.log |
| Relocated producer battery, all rows | exit 0, controls verified | producer-battery-rerun-02.log + battery-detail/ |
| Reviewer neighbor/Local/unknown baseline probes | exit 0 | reviewer-baseline-probes.log |
| Reviewer live dispatch plant harness | exit 0; 2 gate survivors found and witnessed | reviewer-plants-01.log + per-shape logs |

Accepted from prior attached evidence without re-litigating: closed canonical enums, multi-predecessor conformance, four create-window crash-point firing checks, 64-call purity and cross-call slice isolation, and the AC 10/10 mapping. The new full suite reruns their committed coverage, but the prior 64-call reviewer probe was not rerun. Windows and broader-package race/cigate selections were not rerun. Logs retained here are full local outputs, not the truncated upstream #178 validation resource.

Run is not goal-bound (`task-board spawn goal` returned none); operator directives were checked and none were present. Reviewer findings are also persisted in `TASK-260830-1r9wrr_review-logbook-rev2.md`. Acceptance was not recorded; no `accept_cr` or `commit_ack` was used.
