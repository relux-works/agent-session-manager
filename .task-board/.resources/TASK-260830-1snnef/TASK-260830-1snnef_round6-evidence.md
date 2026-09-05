# Round-6 evidence (review round 5 findings F1-F3)

Scope kept: test + doc evidence only. No production file edited this session
(only `muse.edit_file` targets were `internal/terminalbackend/terminalbackend_test.go`
and `README.md`; `terminalbackend.go` md5 `54fb328e1834a8b9b005b98c4e088d3c`
before and after the temporary mutant replays, which were restored byte-identical).

## F1 — checkGeneration utf8.ValidString arm now witnessed
- Added `{"invalid utf-8", "gen\xff"}` to the `refused` table of
  `TestCheckProviderDescriptorGenerationBounds` (expects
  `terminal_backend_stale_generation` at `backend_generation bound`).
- S11 replay (delete the `!utf8.ValidString(generation)` conjunct):
  `TestCheckProviderDescriptorGenerationBounds/invalid_utf-8` FAILS
  (`error = <nil>, want terminal_backend_stale_generation`), every other
  subtest passes. Mutant killed behaviorally, 3/3 shape as reviewer ran it.

## F2 — digest shape arm (terminalbackend.go:587) now behaviorally pinned
- Added `malformed digest on both sides` subtest to
  `TestCheckProviderDescriptorBindingDigestMismatch`: `"not-a-digest"` on
  descriptor AND binding, expects `terminal_backend_not_found` at
  `descriptor binding digest`. Equality arm cannot fire on equal values.
- C1/C6 replay (delete the `scalar.ParseDigest` arm): old descriptor-side
  cases `malformed_digest`, `empty_digest`, `foreign_binding_digest` all
  still PASS (arm #2 catches them — confirms the reported kill inflation),
  only `malformed_digest_on_both_sides` FAILS. Arm #1 now dies to a
  behavioral test, not only to the AST inventory.

## F3 — README sentence corrected
- `README.md` refusal-arm-inventory sentence now states what the inventory
  guarantees (every arm DECLARED with a resolving named test in both
  directions; behavioral proof stays with each row named test, resolved
  textually) instead of claiming an unwitnessed refusal fails the suite.

## Gates (exit codes observed directly, no pipes)
- `go vet ./internal/terminalbackend/` → exit 0
- `gofmt -l internal/terminalbackend/` → empty, exit 0
- Targeted runs of both touched tests on clean production → PASS, exit 0
- `go build ./...` → exit 0
- `go test ./... -count=1` → all 14 packages ok, exit 0
