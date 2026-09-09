# TASK-260908-3kvnm2 results: pin-reviewed-approved-normative-source

## Provenance (verified this session, upstream read-only)

- Signed annotated tag v0.6.0, object 40c123eb8399efa8e05cbc009110940ed861a785; `git verify-tag` exit 0 (ECDSA SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM).
- Peeled commit 0cbdf100dbf84df50c64f792b1f940e3a67859a6; `git verify-commit` exit 0; `ls-remote` peeled ref equal.
- SPEC.md 1150005 bytes, SHA256 74504539fb43c28ae3450622bc1002e643f116cd3e14df4567a882231e90896b, measured from the tagged blob. Delivery is the signed git tag (no hosted GitHub Release); only immutable commit/blob URLs referenced.

## What changed (Story worktree, UNCOMMITTED)

- `internal/specpin/v0.6.0.lock.json` (new): 63 contracts, 5 fixtures, v0.4.3 baseline; lock SHA256 005cd4ffb5aba6792786727bf7d9fa59305191c81b210aae8d3d564d80661fb3. Registry cross-checked row-for-row in order against SPEC section 1.5.
- `internal/specpin/pin.go`: V060 constants + `BytesV060`/`CurrentV060`/`VerifyV060`, `validateV060`, v0.5.0/v0.4.3 historical derivation from a v0.6.0 manifest. `Current`/`Verify` still serve v0.5.0.
- `internal/specpin/sections.go`: 170-entry v0.6.0 inventory (SHA256 2ef110e0f87c5d174bceafb33a0db981f5dc74a384f8b599f12b87cb4af2eb94) + `SectionInventoryV060`/`IsSectionV060`.
- `internal/specdoc/SPEC.v0.6.0.md` (new, exact tag bytes) + `BytesV060`/`LoadV060`/`ParseV060`. `Load`/`Parse` still serve v0.5.0.
- Tests: `pin060_test.go`, `sections060_test.go`, `specdoc060_test.go` (15 new tests). README provenance paragraphs, LOGBOOK entry.
- Untouched: `v0.5.0.lock.json`, `SPEC.md`, all pre-existing tests (still green, unmodified).

## AC coverage: 4 of 4 rows driven through production entry points

1. Pin references verified signed landed source/release: `TestCurrentV060PinsSignedV060Source` via `CurrentV060` (tag object, commit, doc digest, scope, 63 rows, 5 fixtures).
2. Embedded SPEC and section inventories match measured bytes: `TestLoadV060AcceptsOnlyTheAdoptedDocumentDigest` via `LoadV060`/`BytesV060` (1150005 bytes + digest + 16454 lines); `TestSectionInventoryV060MatchesAdoptedHeadingDigest` via `SectionInventoryV060` (170 entries + digest).
3. Old pin provenance remains auditable: every pre-existing pin/scope/specdoc test passes unmodified; `TestSectionInventoryV060PreservesV050History` (v0.5.0 inventory is an ordered subsequence); `TestContractsForReleaseV060` proves the v0.6.0 projections of v0.5.0/v0.4.3 are row-equal to the real historical locks.
4. Real pin readers reject wrong source/hash/version evidence: `VerifyV060` refusal table (11 mutations incl. token-preserving stale commit/object/doc), partial/malformed reads, cross-version refusal both directions (`TestVerifyRefusesForeignReleaseBytes`, `TestParseRefusesAdoptedDocument`, `TestStalePinCannotAuthorizeNewRelease`), unknown-release refusals; `ParseV060` refusal table (8 cases incl. stale v0.5.0 document).

## Negative evidence: 4 narrowing mutants, 0 survivors

Each mutant keeps the gate and admits exactly one rejected member (exact probed digest); each kills exactly its named subtest while the full package suite (behavioral tests included) stays green; all restored cmp-clean. Detail: `.temp/TASK-260908-3kvnm2/mutant-results.md`.

- M1 digest+trailing-newline kills `.../byte_different_whitespace`.
- M2 identity+stale-commit (v0.6.0 tokens preserved) kills `.../stale_commit_keeps_v0.6.0_tokens`.
- M3 ParseV060+stale-v0.5.0-doc kills `.../stale_v0.5.0_document`.
- M4 token-preserving delta-row Versions drop kills `.../trimmed_configuration_delta`.

## Validation (explicit exits, full logs under `.temp/TASK-260908-3kvnm2/`)

- `go vet ./...` exit 0; gofmt clean.
- `go test ./... -count=1` exit 0: 23 packages ok, 0 FAIL (log `gotest-all.log`).
- `go test ./... -count=1 -cover` exit 0: 23 ok; specpin 87.2%, specdoc 100.0% (log `gotest-cover.log`).

## Stated bounds and handoff notes for the reviewer

- Selector fixture ID `ax-session-selector-conformance-v1` is pin-local (path-stem convention): upstream `session_selector_conformance.json` carries `specification_version` 0.6.0 + `contract` 1.0.0, no `fixture` discriminator. Bound by path + SHA-256 `2bda47f5...`.
- No new runtime support is advertised: `Current`/`Verify`/`Load` still serve v0.5.0; catalogue/traceability regeneration belongs to sibling TASK-260908-2tkufa after this leaf checkpoints (recorded cross-leaf dependency).
- SPEC prose describes the historical v0.5.0 registry loosely (`Every other row is unchanged`); the pin follows the disambiguating delta sentences (selector: Session selector 1.0.0 + CLI Result 5.0.0 + Structured Error 1.4.0; auth: Configuration 4.0.0 + Mesh RPC 5.0.0 + Host Channel/Trust Store 1.0.0) and proves the derivation against the real v0.5.0 lock bytes.
- Work left UNCOMMITTED in STORY-260908-18woqo worktree for the handoff snapshot. Ready for review.
