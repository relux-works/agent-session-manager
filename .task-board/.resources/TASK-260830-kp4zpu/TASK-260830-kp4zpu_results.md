# TASK-260830-kp4zpu results — implement-provider-identity-validation

Status: ready for review. Candidate left UNCOMMITTED in the Story
worktree for the handoff snapshot; no commit on the Story branch.

## Deliverable

Host-side Provider Identity Record 1.0.0 creation and validation in
`internal/provhost` (extended, not duplicated):

- `CreateIdentity` (`identity_create.go`) builds Section 5.5 records
  from host params with the true omit-self `record_id` computed
  through the owner's `CalculateObjectIdentity`; byte-identical for
  identical params.
- `DecodeNativeDiscovery` decodes the Section 7.5
  `NativeDiscoveryProof` on the tuple platform.
- `StoreRootFor` resolves the six Section 8.2 documented store
  roots (Muse XDG-aware, Antigravity backend-only, Qwen/unknown
  refused).
- `CheckResumeTuple` enforces the Section 8.4 native-resume
  direction incl. the Muse macOS-arm64 exact-0.1.0 pin and the
  Appendix B gates; passing never advertises usable.
- `VerifyIdentityBuild` / `VerifyIdentityDiscovery` bind records to
  the exact probed tuple, store root, backend realm, and discovery
  proof, refusing drift, mismatch, absence, and unresolved realms.

## Coverage ratio

9 of 10 AC rows driven through the named production entries; the
tenth (crash evidence) is a stated bound: the entries are pure and
mutate no durable state, so no crash/restart probe applies. Per-row
call sites, tests, and bounds: `internal/provhost/TRACEABILITY.md`.
Clause-level mapping: `TASK-260830-kp4zpu_conformance-matrix.md`.

## Tests

- New behavior tests: `identity_create_test.go`,
  `identity_bind_test.go` (positives, refusal tables, derivation
  tests, `Host.Call` identify-session round trip of a created
  record).
- 50 new refusal arms, each witnessed in
  `declaredOperationWitnessesIdentityCreate`
  (`refusal_arm_operations_d_test.go`); 217/217 derived arms
  witnessed, floor raised 166 -> 217; every constructor site
  carries an exercised negative path (runtime audit green).
- Two new vocabulary derivations: `resumeProviders` from the
  Section 8.2 table, `discoveryMembers` from the Section 7.5 type
  row; census 24/24.
- One cross-package registration: the shared `reverseDNSPattern`
  grammar copy ledgered in `internal/environ/census_test.go`
  (required by `TestSharedGrammarsAreOneLanguage`; first full run
  caught the fresh-name copy, fixed by rename + ledger row).
- Mutants: shipped `internal/provhost/testdata/mutate_identity.py`,
  executed on the exact final source: 24/24 applied N plants
  KILLED (incl. two token-preserving: opaque `/` suffix-swap, muse
  `0.1.0` admission), harmless control SURVIVED, NOT_APPLIED and
  COMPILE_OR_HARNESS_FAILURE outside the numerator. Per-plant raw
  logs + `mutants.json` in the evidence tarball.

## Validation (this session, real exits)

- `gofmt` clean, `go build ./...`, `go vet ./...`: pass.
- `go test ./... -count=1`: EXIT=0, 26 packages ok (after the
  grammar-rename fix; the pre-fix run failed only
  `TestSharedGrammarsAreOneLanguage`, fixed, re-run green).
- `go test ./... -race -count=1`: EXIT=0, 26 packages ok (log
  `gotest-race.log`).
- `go test ./... -cover -count=1`: EXIT=0, 26 packages ok (log
  `gotest-cover.log`).
- 13/13 fuzz seed gates pass (scalar x1, canonicaljson x4,
  secconftest x8, 100x each).
- `tracecheck`: ok (contracts=63, sections=36, cases=101);
  `cataloggen -check`: pass.
- `GOOS=linux` / `GOOS=windows` builds: pass; JSON validation:
  pass; `task-board validate`: exit 0 (no issues on this
  task/story); `git diff --check`: pass.
- provhost coverage: 85.6% of statements.

## Bounds and non-claims

- Section 2.4 profile authority is sibling-owned
  (TASK-260830-3uzfyn); version ranges stay opaque; only Muse
  carries a cell-level version pin; discovery binding is
  at-or-under with byte-exact comparison; no `VerifyObjectIdentity`
  call site added outside `internal/sessrepo` (retained bound
  holds); no CLI/doctor/capability surface added or changed.

## Files changed (uncommitted candidate)

- `internal/provhost/identity_create.go` (new),
  `internal/provhost/identity_bind.go` (new),
  `internal/provhost/identity_create_test.go` (new),
  `internal/provhost/identity_bind_test.go` (new),
  `internal/provhost/refusal_arm_operations_d_test.go` (new),
  `internal/provhost/testdata/mutate_identity.py` (new),
  `internal/provhost/TRACEABILITY.md` (new),
  `internal/provhost/refusal_arm_inventory_test.go` (witness wiring
  + floor 217), `internal/provhost/closed_vocabulary_census_test.go`
  (2 derivations + 2 bogus rows), `internal/provhost/doc.go`
  (Section 8.2 accuracy), `internal/environ/census_test.go`
  (grammar ledger row), `README.md` (operation-layer sentence),
  `LOGBOOK.md` (entry).
