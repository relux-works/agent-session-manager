# BUG-260902-1qsdag review verdict — ACCEPTED

- Change Request: `CR-BUG-260902-1qsdag-1` revision 1
- Base OID: `013fa3bbc74f9f57b4ff4a5b8d4fa7fdb73718c4`
- Candidate tree OID: `0844e2fccd44c5aad3abb2e0a11aa674165063e8` (verified equal to worktree `HEAD^{tree}` at `15e5386`; `git status --short` empty, tree untouched by review)
- Reviewer run: `RUN-260902-1580cc` (not goal-bound)
- Verdict: **accepted**, with one non-blocking finding recorded below.

## What the change does

`PutBlob` no longer routes every non-`ErrUnsafeOwnership` inspection error to
`quarantineExisting`. `verifyBlobContent` (`internal/localstore/blob_inspection.go`)
becomes the single classifier both the object store and the projection source
scan consult, returning a `blobVerdict`. Only `blobMismatch` — a completed read
that disagrees with the declared size or digest — satisfies
`quarantineWarranted()`. `blobUnreadable` returns `ErrDurability` from
`resolveExistingEntry` and moves nothing. `storeOperations.openExisting` and
`projectionHooks.openBlob` are the injected read seams; both default to
`openBlobFile` in production (`defaultStoreOperations()`, and the nil fallback
at `projection.go:742` reached from `OpenProjection` → `openProjection(ctx, paths, projectionHooks{})`).

## How the gate was attacked, not read

I did not accept the producer's mutant log as the evidence. I built an
independent scratch copy of the candidate tree and ran five mutants the producer
did not run, chosen to attack the specific ways this gate could be defeated
while staying green. Evidence: `BUG-260902-1qsdag_review-mutant-evidence.log`.

| # | Mutant | Intent | Result |
| --- | --- | --- | --- |
| R1 | Quarantine the staged candidate *before* classifying `blobUnreadable` (existing object still stays put) | A partial move that the error-type assertion alone would miss | **Killed** — `assertQuarantineNamespaceEmpty` reddens all 4 subtests |
| R2 | A failed `Lstat` returns `blobAbsent` instead of `blobUnreadable` | The absence-vs-failure conflation shape | **Killed** — `TestInspectExistingClassifiesEveryFaultItCanReach/lstat_fails_for_a_reason_other_than_absence` |
| R6 | Widen `quarantineWarranted()` to admit **only** `blobUnreadable` (targeted, not the producer's blanket widening) | The exact defect class, minimally expressed | **Killed** — verdict table plus 3 `inspectExisting` subtests |
| R7 | Delete the `blobUnreadable` branch **and** widen `quarantineWarranted()` — i.e. restore the exact pre-fix behavior | Would the suite have caught the shipped bug? | **Killed** — `TestPutBlobReportsIncompleteExistingReadAsDurabilityAndMovesNothing` (all 4, including the raced branch), plus the agreement test |
| R8 | The projection re-labels an incomplete read as `projectionRefusal(ErrProjectionSourceIntegrity, "blob %s could not be read")` instead of `sourceFailure` | Attacks the substring proxy in `classifyThroughProjection` | **SURVIVED** — see finding below |

R7 is the decisive one: the negative case reddens on the exact behavior this bug
reported, with the existing object actually moved into quarantine. The gate is
bounded from both directions — R6 proves narrowing is caught, and the producer's
MUTANT D (mismatch comparison narrowed to size only) proves the same-size digest
disagreement is a real bound rather than a delete-only artifact.

The seam is production-wired, not a dead field: `inspectExisting` reads
`store.operations.openExisting`, `defaultStoreOperations()` is the only
`storeOperations` literal in the package, and the negative cases drive `PutBlob`
— the production entry point — rather than the helper. The raced post-rename
branch is covered by its own subtest, which matters because that branch
previously carried a hand-copied second classification block.

## Independent verification of the producer's claims

| Claim | Verified | Result |
| --- | --- | --- |
| `go build ./...`, `go vet ./...`, `gofmt -l .` | rerun | exit 0, no output |
| `go test ./... -count=1` | rerun | exit 0, 10/10 packages ok |
| `go test ./... -cover` | rerun | exit 0 |
| `internal/localstore` 83.5% → 85.6% | **measured at the base commit myself** via `git archive 013fa3b` into a throwaway tree | base 83.5%, candidate 85.6% — no regression, claim honest |
| global tracecheck | rerun | exit 0 — `contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 assigned_scopes=0` |
| 30-section scoped tracecheck | rerun | exit 0 — `assigned_scopes=30` |
| `cataloggen -check` | rerun | exit 0 |
| `go test ./internal/localstore -race` | added by review | exit 0 |

The `reviewedOwnershipCanonicalSHA256` re-pin is the explicit review this
binding requires, and it is warranted: the five added declarations in
`ownership.v0.5.0.json` all exist in `blob_inspection_test.go` and all exercise
§3.2 object-store behavior. No self-minted ownership claim.

## Acceptance criteria

| AC clause | Verdict |
| --- | --- |
| A read failure during inspection is a durability error with nothing moved | **Met** — `resolveExistingEntry` returns `ErrDurability`; R1 and R7 both redden if anything moves |
| Quarantine only on a proven digest mismatch or representation disagreement | **Met** — `quarantineWarranted()` admits `blobMismatch` only; R6 and producer MUTANT B/C bound it in both directions |
| `storeOperations` gains an injectable reader | **Met** — `openExisting blobOpener`, defaulted in production, nil-guarded in `PutBlob` |
| A negative case proves a transient read leaves the object in place, and reddens when the classification is collapsed back | **Met** — R7 restores the pre-fix behavior and reddens 4 subtests plus the agreement test |
| Projection and store agree, asserted rather than assumed | **Met structurally**, with a caveat — see finding |

## Non-blocking finding: the projection half of the agreement is a substring proxy

`classifyThroughProjection` decides whether the projection called the condition a
proven mismatch with `strings.Contains(openErr.Error(), "contains")`
(`blob_inspection_test.go:275`). Mutant R8 shows this does not hold: changing the
projection's incomplete-read outcome from `sourceFailure` to a
`projectionRefusal(ErrProjectionSourceIntegrity, ...)` about the bytes leaves the
whole package green, and the derived refusal inventory does not catch it either.
The proxy is also brittle in the other direction — rewording the proven-mismatch
message to anything without the word "contains" produces a false red.

Why this is not grounds for changes requested:

- The projection has **no typed discriminator to assert on**. `sourceFailure` and
  every `projectionRefusal` on that path share the single
  `ErrProjectionSourceIntegrity` sentinel (`projection.go:32, 871`). That
  coarseness is pre-existing, unchanged by this work, and outside this bug's
  scope — the bug report explicitly states the projection already gets the
  behavior right.
- R8 does not create a classification disagreement. Both paths still route
  through `verifyBlobContent` and both still reach `blobUnreadable`; R8 changes
  only the label the projection puts on a refusal it was already raising under
  that same sentinel. The structural agreement — one classifier, not two
  hand-copied blocks — is stronger than any test could assert and is what the AC
  is actually protecting.
- The **load-bearing** assertions in that test are direct, not proxies:
  `projectionMoved` and `storeMoved` are `os.Lstat` results cross-checked against
  `result.ExistingQuarantinePath`. The safety property — the projection never
  moves a durable object, and the store moves one only on a proven mismatch —
  is asserted against the filesystem.

Recommended follow-up, not required for this leaf: if the projection ever gains a
distinct sentinel for "source read did not complete" versus "source bytes
disagree", replace the substring with `errors.Is` and drop the proxy. Worth a
Bug on the projection error taxonomy rather than rework here.

## Handoff

Reviewer-archetype run — no `commit_ack` supplied. `accept_cr` parks
`BUG-260902-1qsdag` at `to-review`; the orchestrator checkpoints or integrates
revision 1 and makes the `done` transition with `commit_ack=scope_committed`.
Carry the R8 finding into `LOGBOOK.md` on the next commit that touches it; the
review did not modify the reviewed tree.

## Ruling on the producer's self-flagged deviation

The producer was directed to reapply `BUG-260902-1qsdag_quarantine-classification.patch`
and not reimplement. It reimplemented, disclosed this unprompted, and offered to
drop its commit. **The deviation is accepted; the reimplementation stands.**

I verified the producer's justification rather than taking it at face value:

- `git apply --numstat` on the patch: it touches exactly
  `internal/localstore/object_store.go` and a new
  `internal/localstore/object_store_quarantine_proof_test.go`. It **never touches
  `projection.go`**. Under that patch the projection keeps its own independent
  open/copy/hash block, so the two paths would remain two classifiers that
  happen to agree today rather than one classifier they both consult.
- The patch's `TestObjectStoreAndProjectionAgreeThatAFailedReadIsNotAMismatch`
  drives its projection arm through the pre-existing `afterBlobStat` hook, which
  fires after the stat — not a read that did not complete. That asserts neither
  path calls the condition a mismatch; it does not assert they share a
  classification.
- Consequently the patch would leave AC clause 5 — "The projection and store
  paths agree, and the agreement is asserted rather than assumed" — unmet, and
  would give the projection no injectable read seam.
- The producer's claim that reapplying was a live option it declined is honest:
  I reproduced `git apply --check` against a `git archive 013fa3b` export,
  exit 0.

Honoring the directive literally would have shipped work that misses a named
acceptance criterion. The delivered commit is a strict superset of the patch —
every element is preserved under different names — so there is nothing to
recover by dropping it.

## Leaf shape

- `git rev-list --count 013fa3b..HEAD` = **1** — one commit past the checkpoint,
  as directed.
- `git verify-commit 15e5386` → Good signature, `oparin@me.com`, ECDSA
  `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`.
- Author `Ivan Oparin <oparin@me.com>` preserved.
- Branch not pushed or landed, per the Story workspace contract.
