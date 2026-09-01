# TASK-260830-8x76g1 — CR rev 14 review verdict

- Reviewer run: `RUN-260901-5bce41` (claude / opus), independent of the codex producer.
- Change Request: `CR-TASK-260830-8x76g1-14`, revision 14, `repository_delta=present`.
- Base OID: `ad7275181ca82fc3fa29544e3893923a92d7b9d5`
- Candidate tree OID: `0ca65322a7bf171b8e51bef353250a35d4f094f5`
- **Verdict: CHANGES REQUESTED -> `to-dev`.** The rev13 finding is genuinely
  closed, every configured gate is green, and the whole numeric-bound sweep
  reproduces. An independent clause-level attack — a mutation class the
  producer's harness structurally cannot generate — found **12 spec-derived
  refusal clauses on the composed production path that no test pins**, each of
  them claimed as enforced in `testdata/constraint-enumeration.md`. Production
  code is correct on all 12; the rework is test-only plus artifact honesty.

## Binding and integrity

The worktree was hashed into a private index (`GIT_INDEX_FILE`, real index and
`HEAD` untouched) before review, after the mutation sweeps, and after every
probe file was deleted. All three times it produced
`0ca65322a7bf171b8e51bef353250a35d4f094f5`, byte-identical to the candidate
tree. `closed_shapes.go` hashes to its declared pre-sweep SHA-256
`3d34cb52cb483e513a8ba398ed351993a02c46c01d5f4086af083f038304b307` after every
mutant.

The rev13 -> rev14 delta is exactly what the producer reports: two files,
+49/-4, `boundary_constraints_test.go` and `testdata/constraint-enumeration.md`.
No production file, configuration, README claim, pinned specification, or
accepted leaf moved.

Accepted leaves verified against the attached checkpoint tarball (SHA-256
`9ae5e624…eac9c` confirmed): `internal/catalog`, `internal/cataloggen` and
`internal/specpin` are `diff -rq`-identical; all seven `internal/scalar` files
carried in the tarball are `cmp`-identical. The two `internal/canonicaljson`
files that differ from the tarball (`canonical.go`, `canonical_test.go`) are
this leaf's own scope: the delta is the extraction of `prepareObjectIdentity`
so both public entries share the closed-shape gate.

## Gates re-run by the reviewer at the candidate tree

Every configured `spawn.worktree_isolation.validation` command was re-run in the
foreground, not accepted from attached evidence.

| Gate | Result |
| --- | --- |
| `gofmt -l` over tracked+untracked Go files | empty, exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | 8/8 packages ok |
| `go test ./... -race -count=1` | 8/8 packages ok |
| `go test ./... -cover -count=1` | canonicaljson 83.6%, scalar 90.1%, traceability 85.0% |
| `FuzzScalarProductionEntries -fuzztime=100x` | PASS |
| `FuzzCanonicalizeRoundTrip -fuzztime=100x` | PASS |
| `FuzzObjectIdentityRepresentationInvariant -fuzztime=100x` | PASS |
| `FuzzClosedIdentityShapeRefusal -fuzztime=100x` | PASS |
| `tracecheck` | ok, `assigned_scopes=0` |
| `tracecheck -section 17.3` | ok, `assigned_scopes=1` |
| `cataloggen -check` | exit 0 |
| `GOOS=linux` / `GOOS=windows` builds | exit 0 |
| tracked JSON parse gate | exit 0 |
| `git diff --check` | exit 0 |

All four `Fuzz*` targets the candidate defines are wired into the configured
validation list with a fixed `-fuzztime=100x`, and
`TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget` derives that
requirement from the repository AST, so a new unwired target fails the suite.
No gate gap.

## Correctly delivered this cycle

**Finding 1 of rev13 is closed, verified independently.** Under the single
vet-clean mutant `if size == 0 {` -> `if false && size == 0 {`, the full
`internal/canonicaljson` suite now fails, and the sole failing test is
`TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries`:

```
--- FAIL: TestTrailingBlobChunkSizeMinimumReachesBothIdentityEntries (0.00s)
```

The new case reaches `closed_shapes.go:614` rather than being short-circuited
by the empty-descriptor rule, pairs an accepted trailing chunk of size 1 with a
refused trailing chunk of size 0, drives both `CalculateObjectIdentity` and
`VerifyObjectIdentity`, and asserts the exact reason
`BlobChunk[1] size must lie in [1, 4194304]`. The misleading
`"blob chunk size non-zero"` table case was renamed to the cross-field rule it
actually exercises. `constraint-enumeration.md:36` now names the test.

**The derived sweep reproduces exactly.** Re-running the rev12 `genmutants.py`
over the current `closed_shapes.go` produces 71 mutants byte-identical to the
producer's `rev14-mutants-derived.json` (`only mine: set()`, `only theirs:
set()`). My own independent sequential runner reproduces the same
56 KILLED / 15 SURVIVED split on those 71, plus 4/4 KILLED on the symlink clause
mutants — 60 killed, 15 raw survivors of 75, matching the producer's report,
including the previously actionable `line614`.

**Central closedness is well pinned.** Disabling the unknown-member refusal
inside `requireExactMembers` itself (`closed_shapes.go:1700`) fails 9 seed cases
at the public entry (`record-unknown-top-level`, `blob-unknown-top-level`,
`manifest-unknown-top-level`, and six nested variants).

**Validator totality holds.** `mustBuildImmutableObjectShapeValidators` binds the
table to `catalog.Current().SelfIdentities` and panics at package init on any
missing, extra, or nil row, so a newly registered schema cannot fall through to
extension-only attestation. Both public entries share `prepareObjectIdentity`,
so the gate is on the composed path, not a helper.

## Finding — 12 refusal clauses that production enforces and the suite does not pin

The producer's harness mutates numeric bounds, regex quantifiers, helper-call
limits and three named zero/empty guards. It cannot generate a mutant for a
tagged-union arm, a cross-field equality, a grammar predicate, or a
case-collision predicate. I ran a complementary sweep over **every** refusal
clause in `closed_shapes.go` — 110 `if <cond> { ... invalidIdentity(...) }`
sites, each disabled individually as `if false && (<cond>) {`, each against an
uncached `go test ./internal/canonicaljson/ -count=1`. Result: 65 KILLED,
45 SURVIVED.

For twelve of those survivors I confirmed at the public entry that unmutated
production really does refuse — so the clause is live, reachable, and its
disablement is a genuine widening, not dead code. Each probe drove
`CalculateObjectIdentity` on a shape that is valid in every other respect:

| Site | Refusal that unmutated production emits | Suite when disabled |
| --- | --- | --- |
| `:300` | `Session Record Launch Plan env_literals key "9BAD-NAME" has invalid environment-name grammar` | green |
| `:408` | `Session Record Board Identity logical_id has invalid grammar` (probed `-leading-hyphen` and `has space`) | green |
| `:416` | `local Session Record Board Identity remote_url must be null` | green |
| `:861` | `WorkspaceSnapshot workspace_group_id must equal manifest subject_id` | green |
| `:1082` | `member ref: git-ref: does not satisfy git check-ref-format grammar` | green |
| `:1089` | `branch GitHead requires non-null oid and ref` | green |
| `:1097` | `unborn GitHead requires null oid and refs/heads/ ref` | green |
| `:1231` | `GitSubmodule paths must not duplicate or case-collide` | green |
| `:1295` | `initialized GitSubmodule member repo_relative_cwd must be non-null` | green |
| `:1307` | `initialized GitSubmodule head must be branch or detached` | green |
| `:1328` | `GitSubmodule pack and features object formats must match` | green |
| `:795`/`:807` | symlink/hardlink `target` type refusal is reached only through `requireString`; no case drives the non-string arm | green |

The strongest of these is `:1231`, because it is a **narrowing** mutant, not a
delete-only one — exactly the shape the review contract asks for. Replacing
`strings.EqualFold(pathValue, previous)` with `pathValue == previous` keeps
exact-duplicate refusal intact and drops only the case-collision half:

```
### unmutated
PROBE submodule sibling case collision  REFUSED :: GitSubmodule paths must not duplicate or case-collide
### MUTANT closed_shapes.go:1231  strings.EqualFold(...) -> ==
PROBE calculate: field="manifest_id" err=<nil>
PROBE verify:    field="manifest_id" err=<nil>
### SAME MUTANT, WHOLE internal/canonicaljson SUITE
ok  github.com/relux-works/agent-session-manager/internal/canonicaljson  5.048s
```

Both public entries **attest** a Transfer Manifest whose git workspace member
carries sibling submodules `vendor/lib` and `vendor/LIB`, and nothing in the
repository notices. The sibling site `:888` (WorkspaceSnapshot group paths) is
correctly pinned — the same mutant there is KILLED — so this is a per-site gap,
not a missing mechanism. `simpleFoldKey -> identity` is also KILLED, so the
manifest-entry collision path is covered.

### Why this is a finding and not a nitpick

`testdata/constraint-enumeration.md` asserts every one of these as enforced:

- `:130` — `logical_id` … “Enforced exactly as declared before identity calculation or verification.”
- `:202` — “`env_literals` keys use the same grammar and are disjoint. | Enforced by `validateSessionLaunchPlan`.”
- `:204` — “Board Identity | Local requires null URL … | Enforced by `validateSessionBoardIdentity`.”
- `:213` — “WorkspaceSnapshot | Group ID equals manifest subject … | Enforced by `validateWorkspaceSnapshot`.”
- `:215` — “GitHead | Branch has OID and ref, detached has only OID, unborn has only a `refs/heads/` ref. | Enforced by `validateGitHead`.”
- `:217` — “Git object formats | Pack, head, index, feature, and submodule OIDs agree … | Enforced by … `validateGitSubmodule` …”
- `:219` — “GitSubmodule | … initialized state is complete and not unborn …”
- `:220` — “GitSubmodule | … and sibling paths do not case-collide. | Enforced by `validateGitSubmodules` and `validateGitSubmodule`.”

The standing Definition of Done requires that
`constraint-enumeration.md` “claims only what the suite proves … or the row
states plainly that production enforces it while no test pins it”, and that
gating behavior ships “negative tests that fail when the gate admits what it
must reject”. Twelve enforcement claims currently fail both rows.

### Required rework — test-only, plus artifact honesty

Do **not** edit `closed_shapes.go`, any other production file, any accepted
leaf, the validation configuration, README, or the pinned specification.
Production is correct on all twelve.

1. Add a public-entry negative case for each of the twelve sites above, driving
   both `CalculateObjectIdentity` and `VerifyObjectIdentity` and asserting the
   exact refusal reason, on an input that is valid in every other respect.
   Each must fail when its clause is disabled. The GitHead arms need the
   `branch` and `unborn` cases specifically — the `detached` arm (`:1093`) is
   already pinned. For `:1231` use a **narrowing** proof
   (`strings.EqualFold` -> `==`), not a delete-only one, so the case-collision
   half is what is pinned.
2. Re-run the full clause sweep and triage every remaining survivor with the
   same standard rev13 applied to the numeric survivors: name the *mechanism*
   that subsumes it and prove it at the public entries, or add a case. The 33
   survivors I did not classify are mostly `if !ok` presence/type guards that
   `requireExactMembers` and the per-member `requireX` calls plausibly subsume,
   and a handful (`:579`, `:582`, `:620`, `:624`, `:1705`, `:1748`, `:1798`)
   that look mutually redundant with a neighbouring invariant — but "plausibly"
   is not evidence, and I did not verify them. Report unknown where you do not
   prove it.
3. Re-run the rev12 `genmutants.py` derived sweep as before and keep it at zero
   actionable survivors.
4. Attach both sweep outputs. My harness and full results are attached as
   `TASK-260830-8x76g1_review-evidence-rev14.tar.gz`
   (`clausesweep.py`, `refusal-clauses.json`, `clause-results.tsv`,
   `openness.py`, `openness-results*.tsv`, `sweep.py`, `sweep-results.tsv`,
   `my-mutants.json`); reproduce rather than re-derive.

## Not findings

- **`requireExactMembers` per-site closedness.** Renaming the call at one site to
  an open variant survives 6 of 24 times once
  `TestConstraintEnumerationMatchesRequireExactMembers` is skipped
  (`:340`, `:437`, `:780`, `:792`, `:804`, `:933`). In the real suite all 24 are
  KILLED, and the AST test kills them for the right reason: it derives the call
  set and member lists from the production source and requires an exact
  one-to-one artifact mapping, so a removed, renamed, or widened member list
  fails. The mechanism itself is separately pinned by 9 fuzz seeds. No action.
- **README.** Makes no per-bound or completed-audit claim, correctly reports
  migration publication, atomic-reference advancement, `ax migrate` and
  `ax doctor` as unavailable. Its linearity claim reproduces through
  `TestTransferManifestMaximumEntryGateIsLinear` (65,536 entries under the
  5,242,880-byte cap, `< 2s`, `!race` build tag).
- **`GitIndex.entries` unreachability disclosure** is intact and honest;
  `TestGitIndexEntryCountBoundaryIsBoundBelowThePublicObjectSizeGate` logs the
  measured encoded size, requires it above the 5,242,880-byte cap, pins the
  validator boundary at 65,536/65,537, and proves both public entries refuse the
  oversized representation.
- **The 15 numeric survivors** are the same set rev13 classified. I reproduced
  the split rather than re-deriving each mechanism, and the two I spot-checked
  (`requireBoundedString` -> `requireString` empty subsumption at `:1748`;
  `line596` narrowing inside the `1<<32-1` literal) hold.

## Scope instruction for the next producer

Test-only, plus any artifact row you cannot back with an executable case. Close
the twelve sites, triage the remaining survivors with proof rather than
plausibility, re-run both sweeps, and attach the outputs.
