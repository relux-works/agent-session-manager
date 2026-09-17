# Review verdict — TASK-260830-3g12yp rev1: implement provider and pane fencing gates

Verdict: **ACCEPT** (revision 1). Route: `accept_cr(TASK-260830-3g12yp, revision=1,
evidence=TASK-260830-3g12yp_review-verdict-rev1.md)`.

Reviewer ran headless in the managed Story worktree at base `0ebb7fa` with the
candidate applied uncommitted. All probes were isolated immutable copies; the
live worktree, index, branch, and HEAD were never mutated. Scratch harnesses
live outside the repo under `/tmp/rev-3g12yp/`; per-plant raw logs and
subprocess exits are in the attached
`TASK-260830-3g12yp_review-evidence-rev1.tar.gz`.

## 1. Spec fidelity — confirmed

Pinned authority `internal/specdoc/SPEC.v0.6.0.md`, quoted independently:

- 2.2#1–#4 (lines 408–413): exactly one winning lease; replica MUST NOT
  launch/resume/accept input; every owner event carries winning epoch+lease;
  peers reject lower/losing events into a divergent branch. Gated by
  `Authorize*` (`internal/fencing/gate.go`) plus the predecessor's
  `AppendEvent`/projector preservation, driven by
  `TestAuthorizeAdmitsWinnerExactEpochAllOperations`,
  `TestAuthorizeParksAndRefusesRemote`,
  `TestOldOwnerReconnectRejectedAfterForceTakeover`.
- 5.3 tuple rule (lines 1954–1968): greatest `(epoch, lease_id)`, bytewise
  UUID order; revalidation before input/turn/checkpoint/push/resume
  (line 1970). Exact-equality enforcement in `Authorize` lines 356–375;
  expiry reused from `sessrepo.CheckFencingExpiry`, never restated.
- Section 4 wrapper rule (lines 1366–1372, 1418+): compare local fencing
  token before launch; park when remote/ambiguous/unverified; terminate only
  for explicit force recovery after preserving diagnostics.
  `AuthorizeRestore`/`AuthorizeActivation` park; `AuthorizeTerminateStale`
  (`internal/fencing/terminate.go`) authorizes only fenced targets under
  force+diagnostics and refuses the live owner.
- 7.5 `LeaseToken` (line 2952): `{session_id, lease_epoch, lease_id}` minted
  only from a passed gate (`mintLeaseToken`, sole `&leaseSealToken{}` site in
  `token.go`), bound to quiesce/capture/materialize via `Bind`.
- Park vocabulary (line 1799): `remote_owner|stale_owner|restore_policy|
  failed_handoff` + winning lease ID — exact `ParkReason` constants and
  `ParkDetails` extraction.
- Section 15.2 exits (lines ~14849/15.2 table): 10 =
  `not_owner,stale_owner,lease_conflict`; 2 = `invalid_arguments`; 3 =
  `local_precondition_failed`. The producer's 10/10/10/2/3 claim reproduced:
  `TestGateRefusalsUseRegisteredCodes` green in my run, and every park
  unwraps to its registered cause (`TestParkDecisionCarriesSection15Cause`).
  `ErrParked` is a decision marker, never a new class: `park()` always wraps
  a registered cause. No new class minted anywhere (only the six sentinels).

## 2. Refusal census — 28 of 28 arms driven through production entries

`Authorize` core, 18 arms, each reached through all five entries by a named
test: unknown operation, presented session/epoch/lease grammar (3),
foreign session, absent winner, winner grammar, unverified, ambiguous,
failed handoff, missing grant, missing clock, expired grant, unusable policy,
remote direction, lower epoch, ahead epoch, lease mismatch.
`AuthorizeTerminateStale`, 8 arms with 8 named tests (malformed target,
garbage winner, no winner, no force, no diagnostics, foreign session,
live owner at epochs 1 and 2, fenced-target authorization).
`LeaseToken.Bind`, 2 arms (forged/zero seal, unknown operation).
Ratio: **28 of 28 refusal/park arms driven; 18 of 18 AC rows driven, 0 of 18
as a public CLI by stated ownership bound** (no `ax` session command exists;
takeover/fork/stop/resume are modeled callers).

Taxonomy attacks (isolated copies, raw logs attached):

- Swap: exchanged the adjacent Ambiguous/HandoffFailed conditions. KILLED by
  the existing suites on both sides
  (`TestAuthorizeParksAndRefusesAmbiguous/activation/verified`,
  `TestAuthorizeParksAndRefusesFailedHandoff/activation/verified`, …).
  My both-flags probe passes on the clean tree, pinning precedence
  (ambiguity first) — log `own/A-swapped-*.log`.
- Swallow (one-token class change, stale→conflict on lower epoch): KILLED by
  `TestAuthorizeRefusesLowerEpoch` — log `own/B.log`.
- Delete park branch (remote launch park→hard refusal): KILLED by
  `TestAuthorizeParksAndRefusesRemote` — log `own/C.log`.

## 3. Behavioral probes — admitted/refused/parked with exact class

Own-plant matrix through the production entries (isolated copies):

- Lower epoch by one and by many: `stale_owner` (non-launch) / park
  `stale_owner` (launch) — existing vectors plus precedence probe.
- Same-epoch loser, lease sorting before and after the winner
  (`fenceLeaseC`, `fenceLeaseC2`): `lease_conflict` / park `stale_owner`.
- Foreign session with winner's epoch (prefix-sharing B, fully foreign C):
  `lease_conflict` on all entries.
- Expired grant at boundary second: boundary-fresh (exactly interval) mints;
  +1s refuses `lease_conflict`; zero policy refuses `invalid_arguments`.
- Absent lease with verified/unverified sync: park `restore_policy` (empty
  winner) on launch, `lease_conflict` elsewhere; `Observe` reports absence
  as data, unknown session propagates `ErrUnknownSession`.
- Ambiguous, remote (prefix-sharing and fully remote holders), unverified:
  park with exact reason + `not_owner`/`lease_conflict` cause, or hard
  refusal off-launch.
- Old-owner reconnect after force takeover: old token refuses `stale_owner`
  (input/mutation/checkpoint), parks `remote_owner` on the old wrapper;
  losing events preserved without application; union reports
  losing-branch-preserved with authoritative state standing.
- Concurrent force takeovers: bytewise-greater lease wins in either union
  order; loser stops accepting input with `lease_conflict`.
- Minted token replayed across operations: stable triple on
  quiesce/capture/materialize from every entry; unknown operation
  (`bogus,resume,stop`) refuses `invalid_arguments`.
- Token bound for epoch N after CAS to N+1: old token stale/remote-refused,
  rival token authorizes (`TestAuthorizeEndToEndOverRepository`).

## 4. Forgery bound — confirmed, plus one extra census-scope probe

- `go build ./internal/fencing/testdata/use` exits 0; `go build
  ./internal/fencing/testdata/forge` fails with
  `cannot refer to unexported field seal` (rerun by reviewer).
- `TestLeaseTokenConstructorCensus`, `TestLeaseTokenCensusPlants` (5
  shapes), `TestLeaseTokenUnforgeableOutsidePackage` all green in my run.
- Own probe M4: build-tagged (`//go:build linux`) alias-backdoor plant —
  census REJECTS it (KILLED, log `own/M4.log`). No build-tag hole in
  `invcore.MustScanProduction`.

## 5. Mutation — producer battery sampled, own plants added

Producer `mutants.json` (in evidence tarball): 31 KILLED (30 N + 1 T),
1 SURVIVED harmless control, NOT_APPLIED/COMPILE controls separate.
Reviewer reran in isolated copies (logs attached):

- `epoch` filter: N-presented-epoch, N-epoch-gte, N-epoch-lte — KILLED.
- `census` filter: T-census-alias — KILLED with 264 `=== RUN` lines
  (full behavioral suite executed alongside the census failure).
- Own narrowings: M2 holder-malformed winner KILLED; M3 terminate live-owner
  epoch-2 KILLED; B/C taxonomy KILLED (above); M4 census KILLED (above);
  C2 harmless-comment SURVIVED control (applied).
- M1/M1f SURVIVED with root-cause analysis (not a gate hole): admitting
  epoch-0 for lease `zzz` is masked by the later lease-grammar arm (same
  `invalid_arguments` class — defense in depth, log `own/M1.log`); admitting
  epoch-0 for a valid but undriven lease (`leaseC`) survives only because no
  test names the `{epoch 0, valid third lease}` vector. The epoch arm itself
  is narrowed-and-killed by N-presented-epoch. Filed as P3 below.

## 6. Agreement — 144 pairs, divergences killed both sides

`TestLeaseTupleOrderAgreesWithSessstateCompare` enumerates
3×2×2 per side = 144 pairs (lease IDs {x,y}, holders {a,b}, epochs
{1,2,3}; holder/record/predecessor/checkpoint/reason varied to prove they
never affect order). Both comparators read use only (epoch, lease_id);
`created_at` is diagnostic-only per §5.3 and a member of neither comparator
— the test comment discloses this substitution for the brief's
created_at-ordering ask. Reviewer flipped the lease tie-break on each side
in isolated copies: D1 (sessrepo) KILLED, D2 (sessstate) KILLED
(logs `diverge/logs/D1*`, `D2*`).

## 7. Traceability and docs — recomputed, not read

- `go run ./internal/traceability/cmd/tracecheck` rerun by reviewer: exit 0,
  `acceptance_cases=110 bindings=57 clauses_discharged=28/485` — exactly the
  claimed figures. The `reviewedOwnershipCanonicalSHA256` re-pin matches
  (tracecheck verifies the pin; green = match).
- All 28 clause line references resolve into `SPEC.v0.6.0.md`; sampled
  excerpts (5.3#1@1927, 5.3#2@1957, 2.2#1@408, 2.2#2@410, …) match the
  pinned text verbatim.
- 9 acceptance cases bind existing declarations (`Authorize`,
  `AuthorizeActivation`, `AuthorizeTerminateStale`, `Bind`,
  `CompareAndSwapLease`, `winningLeaseFor`, `AppendEvent`, `Reduce`,
  `VerifyFencingToken`) to tests that drive them — no self-minted claim.
- 5.3 at 7/8 partial (only caller-side union-maximum 5.3#5 open, disclosed);
  2.2 at 4/22 sliver (replication/secret/store families disclosed as gaps).
- README fencing section + figures (110/57/28/485) match my reproduced
  tracecheck output; test/mutant commands verified runnable; no CLI,
  doctor, or capability claim. LOGBOOK entry truthful.
- Residual `section:17.2` gap prose predates this story and is unchanged —
  correctly out of scope.

## 8. Hygiene and gates — all rerun by reviewer

- Changed paths = exactly the 22 CR paths (7 tracked modifications +
  `internal/fencing/` untracked); no `__pycache__`/`.pyc`; no
  `task-board.config.json`; no stray plants (worktree contains only the
  candidate; my harnesses live in `/tmp`).
- `go test ./internal/fencing/ -count=1`: ok, 40/40 PASS, 0 FAIL.
- `go test ./internal/sessrepo/ ./internal/sessquery/ ./internal/sessstate/
  -count=1`: all ok (sessquery 223s, sessstate 98s).
- `go test ./internal/traceability/... -count=1`: ok.
- `-race` all green, rerun by reviewer: fencing (5s), sessrepo (29s),
  sessstate (90s), traceability/... (49s + 268s), sessquery (485s).
- `go vet` (touched trees), `gofmt -l internal` (empty), `cataloggen
  -check` (exit 0), `tracecheck` (exit 0), `git diff --check` (exit 0),
  `go build ./...` (exit 0), `GOOS=windows go build` + `go vet` on
  fencing/sessrepo/sessquery/sessstate (exit 0): all clean.
- `TestGatesPerformNoDurableWrites` (byte-identical tree + identical
  verdicts across two runs): green — purity bound holds, no crash seam.

## Findings (advisory, non-blocking)

- P3: add one vector to `TestAuthorizeRefusesMalformedPresented` —
  `{session A, epoch 0, valid third lease}` expecting `invalid_arguments`
  — so the M1f narrowing shape is pinned alongside N-presented-epoch.
  Fits a sibling leaf or follow-up; not worth a respawn cycle alone.
- P3: the `section:17.2` gap rationale ("no ownership implementation
  exists") is now stale prose beside the new ownership code; consider
  rewording to its narrower reader-enum meaning when that section is next
  touched.

## What was rerun vs accepted

Reran myself: fencing/sessrepo/sessquery/sessstate/traceability suites
(normal and -race, sessquery race 485s exit 0), tracecheck, cataloggen
-check, vet, gofmt, `go build ./...`, GOOS=windows build/vet, `git
diff --check`, forge/use builds, epoch + census mutant subsets, all own
attacks (M1/M1f/M2/M3/A/B/C/C2/M4/D1/D2) with raw logs.
Accepted from attached evidence: full 31/31 producer battery per-plant
logs (sampled 4/31 by rerun) and the full `go test ./...` 27-package
sweep from the CR validation log (touched-package equivalents rerun
here).

No product edits were made. Candidate left UNCOMMITTED as required.
