# TASK-260830-2890sd round-3 rework notes (uncommitted working tree)

## Blocking

F11 refusal-inventory floor: `auditRefusalInventory` now fails closed on an
empty scan (`ScannedFiles==0`), an empty domain (`Sites==0`), and a short
derivation (reverse check: every exercised site must resolve to a derived
site; bound derived from the test run, not hand-listed). `exportedSymbols`
floor moved to the caller `TestSection71AdvertisesNoPublicSDK` so an
early-return mutant cannot bypass it. Mutants B1/B2/H4/B5/H1 all RED.

F12 OSSystem seam negatives (`os_test.go`, production `OSSystem` + `Trust`/
`Verify` call sites named): ReadDir-failure-as-absence (G33), Inspect stat
failure (G35), Inspect attestation failure via documented `ownerAttester`
seam (G36), `fileOwnerUID` without metadata incl. real-stat forward control
(G38), `CurrentOperatorPolicy` no-administrator pin (G40), PathDirs empty
entries (G37). All RED. G40b (`Geteuid`->`Getuid`) is environment-equivalent
here (`euid=502 uid=502`, probed): recorded bound, not a closure. Windows
refusal covered by new `os_windows_test.go` (build-tagged) compiled via
`GOOS=windows go vet` + `GOOS=windows go build`, both exit 0.

## Non-blocking

F13 detail pins on every precondition/integrity subtest (`cannot list` /
`resolve` / `inspect` / `digest`, `re-resolved` / `re-inspected` / `re-read`)
plus `wantDetail` per trust-gate matrix cell: G9del/G10del/G19del RED.
F14 `vendor-ax-provider-evil` + `xax-provider-foo` anchor fixtures: G42 RED.
F15 `got==nil` no-partial-set assertions on all Discover refusal tests: G54 RED.
F16 `TestVerifyRefusesEmptyOwnerReceipt`: G55 RED. F17
`TestBuiltinsReturnsACopy`: G32 RED.

## Ratios

19-mutant harness over every rev4 survivor shape: 18 RED, 1
environment-equivalent (G40b). `go test ./...` exit 0, provider cover 97.0%,
`go vet` / `GOOS=windows go vet` / `GOOS=windows go build` exit 0.
LOGBOOK.md entry 0327 landed. Work left UNCOMMITTED per the CR-shape
instruction; no `repository_delta=empty` handoff.
