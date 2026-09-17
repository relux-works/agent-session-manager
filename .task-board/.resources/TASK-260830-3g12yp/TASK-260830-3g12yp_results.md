# TASK-260830-3g12yp results — implement provider and pane fencing gates

Status: ready for review. Candidate left UNCOMMITTED in the managed
Story worktree for handoff snapshot; no commit was made on the Story
branch.

## Outcome

`internal/fencing` gates every activation-class action on the winning
lease and the exact fencing epoch: provider activation/launch,
provider input, owner-authored mutation, checkpoint capture, and
terminal restore/wrapper first start, plus terminate-stale under
explicit force recovery. 18 of 18 AC rows are driven through
production entries by named committed tests (0 of 18 as a public CLI,
by stated ownership bound — no `ax` session command exists in this
tree).

AC coverage (row → production call site → named test):

1. activation → `AuthorizeActivation` → `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestAuthorizeEndToEndOverRepository`, `TestLifecycleCallersPassTheGate/fork_epoch_one_activation`
2. input → `AuthorizeInput` → `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestOldOwnerReconnectRejectedAfterForceTakeover`, `TestConcurrentForceTakeoversDeterministicWinner`
3. mutation → `AuthorizeMutation` → `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestLifecycleCallersPassTheGate/graceful_takeover_commit`
4. checkpoint → `AuthorizeCheckpoint` → `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestLifecycleCallersPassTheGate/stop_checkpoint`
5. restore → `AuthorizeRestore` → `TestAuthorizeAdmitsWinnerExactEpochAllOperations`, `TestLifecycleCallersPassTheGate/owner_resume_and_replica_park`
6. lower epoch → `Authorize` core → `TestAuthorizeRefusesLowerEpoch` (`stale_owner` / park `stale_owner`)
7. same-epoch loser → `Authorize` core → `TestAuthorizeRefusesSameEpochLoser` (`lease_conflict` / park `stale_owner`)
8. foreign session → `Authorize` core → `TestAuthorizeRefusesForeignSession` (`lease_conflict`)
9. expired grant → `Authorize` core → `sessrepo.CheckFencingExpiry` → `TestAuthorizeRefusesExpiredGrant` (`lease_conflict`)
10. absent lease → `Authorize` core + `Observe` → `TestAuthorizeParksAndRefusesAbsentWinner`, `TestObserveLoadsWinningLease` (park `restore_policy` / `lease_conflict`)
11. ambiguity → `Authorize` core → `TestAuthorizeParksAndRefusesAmbiguous` (park `restore_policy` / `lease_conflict`)
12. remote ownership → `Authorize` core → `TestAuthorizeParksAndRefusesRemote` (park `remote_owner` / `not_owner`)
13. unverified sync → `Authorize` core → `TestAuthorizeParksAndRefusesUnverified` (park `restore_policy` / `lease_conflict`)
14. old-owner reconnect → `AuthorizeInput`/`AuthorizeMutation`/`AuthorizeRestore`/`AuthorizeActivation` + `Repository.AppendEvent` + `Projector.Project` → `TestOldOwnerReconnectRejectedAfterForceTakeover`
15. concurrent takeovers → `CompareLeaseTuple`/`Compare` + `Authorize*` → `TestConcurrentForceTakeoversDeterministicWinner`
16. forgery → `LeaseToken.Bind` + census → `TestBindRefusesForgedToken`, `TestLeaseTokenConstructorCensus`, `TestLeaseTokenCensusPlants`, `TestLeaseTokenUnforgeableOutsidePackage`
17. terminate-stale → `AuthorizeTerminateStale` → `TestTerminateStale*` (8 tests)
18. LeaseToken projection → `LeaseToken.Bind` → `TestBindProjectsMintedToken`, `TestBindRefusesUnknownOperation`, `TestMintedTokenIsStableAcrossEntries`

## Negative evidence and mutants

- Every gate arm ships a narrowing mutant: 30 N-mutants (epoch `==`
  to `>=`/`<=`, lease/session/holder prefix confusion, expiry
  widening, grant/clock carve-outs, park/refuse swaps) plus 1
  token-preserving census-alias mutant executed against the full
  behavioral suite (263 passing assertions alongside the census
  failure). Battery result: 31/31 KILLED; the applied harmless
  control is SURVIVED; the not-applied and compile-failure controls
  classify separately. Harness:
  `python3 internal/fencing/testdata/mutate.py <evidence-dir>`.
  Per-plant raw logs and `mutants.json` are under
  `.temp/TASK-260830-3g12yp/mutants/` and inside the attached tarball.
- No new error class is minted: `TestGateRefusalsUseRegisteredCodes`
  constructs every sentinel through `internal/axerror` with its exact
  exit (10/10/10/2/3), and every park unwraps to its Section 15 cause.
- Crash/idempotency: the gates are pure (no durable writes), proven
  by `TestGatesPerformNoDurableWrites` (byte-identical tree plus
  identical verdicts across two runs). Stated bound, not a seam drill.

## Traceability and agreement (story-close items)

- `section:5.3` moved from the `canonicaljson` shape validator to the
  real `CompareAndSwapLease` owner at 7/8 partial (only the
  caller-side union-maximum clause 5.3#5 stays open, disclosed in the
  gap); `section:2.2` moved from unowned to a 4/22 sliver bound to
  `Authorize`; 9 lease acceptance cases registered
  (`lease-record-lifecycle`, `lease-fencing-revalidation`,
  `lease-checkpoint-admission`, `lease-divergent-preservation`,
  `lease-union-resolution`, `lease-fencing-gates`,
  `lease-fencing-park`, `lease-terminate-stale`,
  `lease-token-capability`).
- Pin re-derived: 110 acceptance cases, 57 bindings (full 1, partial
  4, sliver 2, unevidenced 47, unmeasured 3), 11 unowned, 28/485
  clauses; `reviewedOwnershipCanonicalSHA256` updated and all
  shipped-state tables (`tracecheck` pins, `Report` literal,
  sliver-refusal rows, planted-sliver rows, README figures and
  measured-coverage prose) moved deliberately.
- `TestLeaseTupleOrderAgreesWithSessstateCompare` extended from 6
  example pairs to an exhaustive 144-pair enumeration (epochs
  {1,2,3} × lease IDs {x,y} × holders {a,b} per side, remaining
  summary members varied); `created_at` takes no part in either
  comparator (diagnostic-only per Section 5.3), disclosed in the
  test comment.
- Residual nit (out of scope, not changed): the `section:17.2` gap
  rationale ("no ownership implementation exists") predates this
  story; its measured coverage (0/1 unevidenced) is unchanged.

## Validation (all run in this session, this worktree)

- `go test ./internal/fencing/ -count=1` → ok (40 tests)
- `go test ./internal/fencing/ -cover -count=1` → ok, 97.5%
- `go test ./... -count=1` → exit 0, 27 packages ok, 0 FAIL
- `go test -race` (fencing, sessrepo, sessquery, sessstate, traceability/...) → exit 0
- `go vet ./...` and `GOOS=windows go vet ./...` → clean
- `go build ./...` and `GOOS=windows go build ./...` → clean
- `gofmt -l internal` → empty (428 files scanned)
- `go run ./internal/traceability/cmd/tracecheck` → ok (110/57/28/485)
- `python3 internal/fencing/testdata/mutate.py` → exit 0, 31/31 KILLED
- Full logs: `.temp/TASK-260830-3g12yp/full-test.log`,
  `.temp/TASK-260830-3g12yp/race-touched.log`,
  `.temp/TASK-260830-3g12yp/mutants/`

## Files changed (uncommitted candidate)

- New: `internal/fencing/{doc,gate,token,terminate}.go`,
  `internal/fencing/{fixtures,gate,token,terminate,codes,census,scenarios}_test.go`,
  `internal/fencing/testdata/{mutate.py,forge/forge.go,use/use.go}`,
  `internal/fencing/TRACEABILITY.md`
- Edited: `internal/sessquery/lease_store_adopt_test.go` (144-pair
  enumeration), `internal/traceability/ownership.v0.6.0.json`,
  `internal/traceability/traceability.go` (pin),
  `internal/traceability/traceability_test.go`,
  `internal/traceability/cmd/tracecheck/main_test.go`, `README.md`,
  `LOGBOOK.md`

## Bounds restated

No `ax` command, doctor surface, or runtime capability is added or
advertised; takeover/fork/stop/resume orchestration, terminal
backends, and provider processes are modeled callers; no `sessckpt`
package exists so `AuthorizeCheckpoint` is the capture admission;
`resume`/`stop`/`materialize-commit`/`native-store-plan` bindings and
clause 5.3#5 stay open as stated above. Full clause matrix:
`TASK-260830-3g12yp_conformance-matrix.md`.
