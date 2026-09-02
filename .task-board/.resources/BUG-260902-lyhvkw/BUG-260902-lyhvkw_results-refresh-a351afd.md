# BUG-260902-lyhvkw — base refresh onto trunk a351afd (RUN-260902-4add4d)

## Why this run existed

Change Request revision 1 (branch tip ef0c1a5, forked from 48db30b) was accepted by review, then trunk advanced to a351afd when STORY-260830-jeaivu (configuration loading) landed. Trunk changed three paths that revision 1 also changes — `README.md`, `internal/traceability/ownership.v0.5.0.json`, `internal/traceability/traceability.go` — so per the tracked-background-spawn contract the acceptance went stale (`integration_base_moved`) and the candidate had to be rebuilt on current trunk and republished as revision 2.

## What changed in this run

- `git rebase -S main` of the story branch: one conflict, on the ownership-registry digest pin `internal/traceability/traceability.go:42` (trunk repinned to `7badbfe8…` for the config tests; revision 1 repinned to `e37f8208…` for the nesting-depth tests). `README.md` and `ownership.v0.5.0.json` three-way merged cleanly and the merged registry carries both the config rows and the nesting-depth rows.
- Resolved by taking trunk's file, running tracecheck (fails closed, exit 1, reports the merged projection digest), and repinning `reviewedOwnershipCanonicalSHA256 = 9f7737cb07f853012fcc9e2359981e20e9b65df622f9da7fa4935a2180cd04b0`.
- `LOGBOOK.md`: one new entry recording the refresh and the digest-pin conflict finding.
- No change to the fix itself: `maxNestingDepth = 256` in `internal/canonicaljson/canonical.go:37`, gate in `decodeValue` at `canonical.go:362`, tests in `nesting_depth_test.go` — byte-identical to revision 1.

Signed head: `3e48785e15b4e818ca298efee5d6acddaa251a16` (verify-commit good, `main` a351afd is its parent, working tree clean). Pre-rebase tip ef0c1a5 preserved in reflog and as `refresh/fix-vs-48db30b-pre-rebase.patch`.

## Gates rerun by this run on the merged tree (real exit codes)

| Gate | Command | Exit |
| --- | --- | ---: |
| gofmt | `gofmt -l $(git ls-files '*.go')` | 0, no files |
| vet | `go vet ./...` | 0 |
| build | `go build ./...` | 0 |
| build linux/amd64 | `GOOS=linux GOARCH=amd64 go build ./...` | 0 |
| build windows/amd64 | `GOOS=windows GOARCH=amd64 go build ./...` | 0 |
| catalog | `cataloggen ... -check` | 0 |
| tracecheck (trunk digest, before repin) | `go run ./internal/traceability/cmd/tracecheck` | 1 (expected: digest mismatch, fails closed) |
| tracecheck full (after repin) | same | 0 |
| tracecheck sections 1.6/10.1-10.4/17.3 | `... -section ...` | 0 |
| tests verbose | `go test ./... -v -count=1` | 0 (218 PASS, 0 FAIL) |
| coverage | `go test ./... -cover -count=1` | 0 |
| race | `go test ./internal/canonicaljson -race -count=1` | 0 |
| fuzz x3, 100x | `FuzzCanonicalizeRoundTrip`, `FuzzObjectIdentityRepresentationInvariant`, `FuzzClosedIdentityShapeRefusal` | 0 / 0 / 0 |
| final on exact head 3e48785 | `go test ./... -count=1`; `go vet`; `go build`; tracecheck | 0 / 0 / 0 / 0 |

## Coverage vs trunk a351afd (baseline measured on `git archive main`)

| Package | Trunk a351afd | Branch 3e48785 |
| --- | ---: | ---: |
| internal/canonicaljson | 87.1% | 87.2% |
| internal/catalog | 97.6% | 97.6% |
| internal/catalog/cmd/cataloggen | 79.3% | 79.3% |
| internal/cataloggen | 83.9% | 83.9% |
| internal/config | 93.7% | 93.7% |
| internal/scalar | 90.1% | 90.1% |
| internal/specpin | 85.1% | 85.1% |
| internal/traceability | 85.0% | 85.0% |
| internal/traceability/cmd/tracecheck | 87.5% | 87.5% |

No regression.

## Mutants of the bound (all expected red, `go test ./internal/canonicaljson -count=1`)

| Mutant | Exit | Red tests | Notes |
| --- | ---: | ---: | --- |
| widen 256 -> 512 | 1 | 4 | pinned literal + refuse-at-257 at all three entries |
| widen 256 -> 257 | 1 | 4 | minimal widening reddens |
| narrow 256 -> 128 | 1 | 6 | accept-at-256 reddens |
| narrow 256 -> 255 | 1 | 6 | minimal narrowing reddens |
| delete gate (`if false`) | 1 | 2 | reproduces `fatal error: stack overflow` through the 2 MB regression test |

Source restored after each mutant; `git status` clean.

## Evidence

`.temp/BUG-260902-lyhvkw/refresh/` logs 00-18 in this workspace; attached: this file, `_mutants-refresh-a351afd.log`, `_tracecheck-refresh-a351afd.log`, `_go-test-cover-refresh-a351afd.log`.
