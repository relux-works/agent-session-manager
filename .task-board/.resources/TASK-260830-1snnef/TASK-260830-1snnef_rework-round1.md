# TASK-260830-1snnef round-1 rework evidence

Round-1 verdict (changes requested, 4 blocking findings) closed. No production change; no declared-table change; README sentence stands, now covered by the derivation.

## F1 — derivation blind spots closed (preferred fix, not a caveat)
- `deriveRefusalArms` now scans every FuncDecl body plus every package-level var initializer (arms attributed to the variable name).
- New `rejectConstructorAliases` fatals on any `mismatchf`/`integrityFailure` reference outside direct-call position.
- Replay A1 (package-level `var plantedA1 = func() *Error` with new detail, called from CheckReplicable): suite RED — bijection fails (194 derived vs 193 rows). Tree restored byte-identical (shasum).
- Replay A2 (`refuse := mismatchf`): suite RED — `constructor "mismatchf" referenced outside direct-call position`. Tree restored byte-identical.
- Inventory still derives 193 arms; both bijection tests green.

## F2 — CheckErrorAllowed refusal set derived as complement
- New full-row mirror `allowedErrorCodesByOperation` (all ten rows exact) + `conformanceOperations`.
- Admit test now exhaustive per row (proves mirror ⊆ production); refuse test iterates `catalog.Current().Errors` minus the row (proves production ∩ catalog ⊆ mirror); mirror-inside-catalog pin + empty-domain fail-closed guards.
- Replay P20 (admit `terminal_backend_unavailable` for all ten operations): `TestCheckErrorAllowedRefusesEveryUnlistedCode` RED, exit 1. Tree restored byte-identical.
- Non-catalog strings are an unbounded domain: three probes pin the shape with the sampling bound stated in-test.

## F3 — triple-equality third leg
- Added `CheckAttachResult(false, false, auth{InputAuthorized:true})` (only the auth conjunct catches it).
- Replay P8 (drop `resultInputAuthorized != auth.InputAuthorized`): test RED, exit 1. Restored byte-identical.

## F4 — attach expiry boundary instant
- Added admit at `expires - 1s` and refuse at exactly `expires`.
- Replay P16 (`!now.Before(expires)` → `now.After(expires)`): test RED, exit 1. Restored byte-identical.

## conformance.go permissive-mutant ratio
- 13/13 narrowing mutants killed (P1/P4/P5/P7/P8/P10/P14/P15/P16/P17/P18/P19/P20). P8/P16/P20 killed by this rework (replayed above, exit 1 each); the other 10 die by tests untouched by this rework over byte-identical production.
- Declined as stated bounds: P11 (legacy escape name; sampling over an unbounded string domain), P13 (empty-ID ProjectToLegacy still refuses, different wire code).

## Bounds carried forward (in tree, not dropped)
- Inventory kill-inflation: arm-deleting mutants read as killed by the inventory alone; post-leaf-3 scores NOT comparable to leaf-2 91/95. Stated in `refusal_arm_inventory_test.go` Stated bounds + LOGBOOK.
- Attribution drift via entry+code fallback, live example CheckTransition dead `"operation vocabulary"` arm (verified: transitionTable covers all ten ops; resolves through ParseOperation test). Stated in-file.
- Coverage: reviewer measured 89.9% on rev-1 tree; LOGBOOK corrected (was 90.0% rounding error). This rework tree measures 90.0% (test-only additions).

## Gates (all run directly, real exit codes)
- `go test ./... -count=1`: exit 0, 14/14 packages ok
- `go test ./internal/terminalbackend/ -race -count=1`: exit 0
- `go vet ./...`: exit 0
- `gofmt -l .`: clean (exit 0)
- `go run ./internal/traceability/cmd/tracecheck`: exit 0, acceptance_cases=76, section coverage unchanged (bindings=49, clauses_discharged=17/403)
- `go test ./internal/terminalbackend/ -cover`: 90.0% of statements

## Delta
- Modified: internal/terminalbackend/refusal_arm_inventory_test.go, internal/terminalbackend/conformance_test.go, LOGBOOK.md. No commit per brief (board checkpoints); repository_delta is non-empty.
