# TASK-260830-2xdt8t G1-G3 close-out verification (commit 9481a20)

## Scope
Close review-round-2 blockers G1-G3 on internal/terminalbackend. No new production guards; one comment correction plus three test pins.

## Changes (3 files, +92/-20, signed 9481a20, sig Good ECDSA SHA256:V6JiKG7J)
- G1: deleted unfailable black-box subtest (built-ins via cloning Resolve); added white-box TestBuiltinsHoldIndependentProtocolArrays observing stored backing arrays; corrected New comment to defence-in-depth.
- G2: added TestRegisterExternalAdmitsFullPlatformSet (last admitted value 4).
- G3: added TestRegisterExternalRefusesReservedNamespaceDistinguishingValue (ax.conpty, asserts ambiguous + external_trust reserved namespace clause).

## Mutant verification (each applied with occurrence-count==1, restored byte-identical via cmp)
- G1 revert both New clones to shared slice -> TestBuiltinsHoldIndependentProtocolArrays FAILS (share one protocol backing array). Killed.
- G2 narrow >4 to >3 -> admission test FAILS exit 1 (refused terminal_backend_not_found at platforms bound). Killed.
- G3 narrow ax. bar to == ax.tmux -> test FAILS exit 1 (falls through to terminal_backend_implementation_drift, want ambiguous). Killed, fall-through path matches reviewer prediction.

## Gates run directly (real exit codes)
- go build ./... exit 0
- go vet ./... exit 0
- gofmt -l internal/ empty
- go test ./... -count=1 exit 0 (14 packages)
- go run ./internal/traceability/cmd/tracecheck exit 0 (74 cases, 17/403, unmoved; no digest re-pin needed, no new declarations)
- go run ./internal/traceability/cmd/tracecheck -section 6.2 exit 0
