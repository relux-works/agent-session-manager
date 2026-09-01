# TASK-260830-8x76g1 — CR rev 15 review verdict

- Reviewer run: `RUN-260901-209a89` (claude / opus), independent of the codex producer.
- Change Request: `CR-TASK-260830-8x76g1-15`, revision 15, `repository_delta=present`.
- Base OID: `ad7275181ca82fc3fa29544e3893923a92d7b9d5`
- Candidate tree OID: `1617e6afc56110589f6f0f524ceb94de192b43b5`
- **Verdict: CHANGES REQUESTED -> `to-dev`.** The twelve rev14 clause findings are
  genuinely closed — I reproduced the full 110-site clause sweep independently and
  all twelve now die. The remaining-survivor triage is mechanized and correct for
  24 of 25 rows. One row is a universally-quantified claim that is false, and the
  clause it excuses is the sole enforcer of a declared SPEC bound: with
  `closed_shapes.go:1748` disabled, both public identity entries **attest** a
  Transfer Manifest carrying a symlink entry with `target: ""`, which
  `constraint-enumeration.md:106` declares as `string[1..4096]`. Production code is
  correct; the rework is one negative case plus artifact honesty.

## Binding and integrity

The worktree was hashed into a private index (`GIT_INDEX_FILE`; the real index and
`HEAD` were never touched) before review, after the clause sweep, after each probe,
and after every gate. Every time it produced
`1617e6afc56110589f6f0f524ceb94de192b43b5`, byte-identical to the declared candidate
tree. `internal/canonicaljson/closed_shapes.go` hashes to its declared
`3d34cb52cb483e513a8ba398ed351993a02c46c01d5f4086af083f038304b307` after every
mutant and at the end of the review.

The rev14 -> rev15 delta is exactly what the producer reports: two files, +471/-9,
`internal/canonicaljson/clause_refusal_test.go` (new) and
`internal/canonicaljson/testdata/constraint-enumeration.md`. No production file,
configuration, README claim, pinned specification, or accepted leaf moved.

## Gates re-run by the reviewer at the candidate tree

Every configured `spawn.worktree_isolation.validation` command was re-run in the
foreground rather than accepted from attached evidence.

| Gate | Result |
| --- | --- |
| `gofmt -l` over tracked + untracked Go files | empty, exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | 8/8 packages ok |
| `go test ./... -race -count=1` | 8/8 packages ok |
| `go test ./... -cover -count=1` | canonicaljson 87.1%, scalar 90.1%, traceability 85.0% |
| `FuzzScalarProductionEntries -fuzztime=100x` | PASS, baseline 38 |
| `FuzzCanonicalizeRoundTrip -fuzztime=100x` | PASS, baseline 29 |
| `FuzzObjectIdentityRepresentationInvariant -fuzztime=100x` | PASS, baseline 31 |
| `FuzzClosedIdentityShapeRefusal -fuzztime=100x` | PASS, baseline 75 |
| `tracecheck` | ok, `assigned_scopes=0` |
| `tracecheck -section 17.3` | ok, `assigned_scopes=1` |
| `cataloggen -check` | exit 0 |
| `GOOS=linux` / `GOOS=windows` builds | exit 0 |
| tracked JSON parse gate | exit 0 |
| `git diff --check` | exit 0 |

## Correctly delivered this cycle

**All twelve rev14 sites are genuinely pinned.** I re-ran the complete rev14 clause
sweep — the same 110 `refusal-clauses.json` corpus, confirmed byte-identical to the
copy the producer used, each site disabled individually as `if false && (<cond>) {`
against an uncached `go test ./internal/canonicaljson/ -count=1`. Result:
**85 KILLED / 25 SURVIVED**. Every one of the twelve previously-surviving sites now
dies:

`:300` `:408` `:416` `:795` `:807` `:861` `:1082` `:1089` `:1097` `:1231` `:1295`
`:1307` `:1328` — all KILLED.

`:1231` dies under deletion here, and the producer additionally proves it by
**narrowing** (`strings.EqualFold(pathValue, previous)` -> `pathValue == previous`),
which fails
`TestSpecDerivedRefusalClausesReachBothIdentityEntries/submodule_sibling_path_case_collision`.
That is the shape the review contract asks for, not a delete-only mutant.

**My sweep reproduces the producer's exactly.** Comparing my 110 rows against
`clause-sweep/final-results.tsv`, the only disagreement is site 94
(`closed_shapes.go:1722`, `requireObjectValue` type guard): SURVIVED in their
mid-change run, KILLED in mine and in their own `final-survivor-rerun.tsv`. That is
precisely the drift the producer disclosed in the triage, so the reported
authoritative 25-row survivor list is honest and matches mine row for row.

**The assertion helpers are honest.** `assertIdentityEntriesRefuseWithReason` drives
`CalculateObjectIdentity` and then `VerifyObjectIdentity` on a body whose self-field
carries the *correct* omit-self digest (`withCorrectIdentityClaimForTest`), so
`VerifyObjectIdentity` cannot pass the test by refusing on a claim mismatch — the
shape gate is what has to refuse. Both public entries share `prepareObjectIdentity`,
so the gate is on the composed path.

**The recursive-deletion proof is real and non-vacuous.**
`TestEveryFixtureClosedShapeMemberIsRequiredAtBothIdentityEntries` runs **278**
subtests across eight fixture families, and I verified from the subtest names that
those families do reach all 24 registered closed shapes — including both
`WorkspaceSnapshotMember` arms, all four `ManifestEntry` arms, `BlobChunk`,
`GitIndexEntry`, `GitSubmodule`, `GitFeatures`, `GitRemote`, `GitObjectPack`,
Board Identity, Board Goal, and `MigrationProvenance`. So the header claim added to
`constraint-enumeration.md` this cycle is backed. Combined with the sweep, this is a
genuine mechanized subsumption proof for every "member missing" survivor: with one
`!ok` presence guard disabled, all 278 deletions are still refused, so
`requireExactMembers` (whose own guard at `:1705` is KILLED) is what catches them.

**Triage rows I independently verified as proof rather than plausibility:**

- `:620` uint53 coverage overflow is arithmetically unreachable: `maxBlobChunks` is
  32,768 and `maxChunkSize` is 4,194,304, so `covered` never exceeds 2^37 while
  `MaxUint53 - size` never drops below 2^53-1-2^22.
- `:1506` / `:1525` invalid-UTF-8 guards are unreachable through public JSON:
  `decodeStrict` rejects invalid UTF-8 bytes and unpaired surrogate escapes before
  any value reaches `validateExtensionValue`.
- `:1628` `nullableString` has exactly three call sites (GitHead `ref`,
  `requireNullableGitOID`, `requireNullableGitRef`) and every one applies a Git
  OID/ref grammar downstream, so the enumeration behind that row is complete.
- `:1794` / `:1798` / `:1845` / `:1593` / `:421` all delegate within the same
  function to `strconv.ParseUint`, the digest parser, `ParseRelativePath`, or the URL
  grammar, so subsumption is total rather than input-specific.
- The numeric corpus binds to this exact file: re-running the rev12 `genmutants.py`
  over the candidate's `closed_shapes.go` regenerates 71 mutants byte-identical to
  the rev14 reviewer corpus, and the attached `final-derived-results.tsv` tallies
  56 KILLED / 15 SURVIVED over the same set that rev13 and rev14 classified.

## Finding — `ManifestEntry.symlink.target` lower bound is enforced by one clause that no test pins

`constraint-enumeration.md:106` declares:

```
| `ManifestEntry.symlink` | `target` | Enforced exactly as declared before identity
calculation or verification. | `validateManifestEntries` | "string[1..4096];
lexically remains within materialization root" |
```

The upper bound is pinned — numeric mutant 14 (`closed_shapes.go:797`,
`> 4096 -> > 4097`) is KILLED. The **lower bound of 1** is enforced only by
`requireString`'s empty guard at `closed_shapes.go:1748`, which is clause-sweep
survivor 98. The producer's triage excuses it with a universal claim:

> `98 / line 1748` — required string empty — *Every caller applies exact grammar, a
> scalar validator, or a minimum length; the public-entry media-type case pins this
> path.*

That claim is true for eight of the nine `requireString` call sites and **false for
the ninth**. `validateManifestEntries` at `closed_shapes.go:794` reads
`target`, then checks only `utf8.RuneCountInString(target) > 4096` and
`symlinkTargetEscapes(pathValue, target)`. For `target == ""`,
`symlinkTargetEscapes` resolves `path.Clean(path.Join(path.Dir("link"), ""))` to
`"."`, which is neither `".."` nor `"../"`-prefixed, so it returns `false`. Nothing
else refuses the empty string.

I drove all nine empty-string members through both public entries with the single
mutant `closed_shapes.go:1748` -> `if false && (text == "") {`:

```
PROBE symlink target           ATTESTED calc=<nil> verify=<nil>
PROBE hardlink target_path     REFUSED  ... path: must be non-empty valid UTF-8
PROBE board logical_id         REFUSED  ... must contain 1..128 Unicode characters
PROBE task_element_id          REFUSED  ... must contain 1..128 UTF-8 bytes
PROBE board_goal goal_id       REFUSED  ... must contain 1..128 Unicode characters
PROBE managed tree_identity    REFUSED  ... must contain 1..256 Unicode characters
PROBE git repository_identity  REFUSED  ... must contain 1..256 Unicode characters
PROBE git remote name          REFUSED  ... must contain 1..128 Unicode characters
PROBE media_type               REFUSED  ... lowercase ASCII type/subtype
```

Unmutated, all nine are refused by `:1748` itself with
`member target must be a non-empty UTF-8 string`. Under the mutant the whole
`internal/canonicaljson` suite stays green (`exit 0`), so nothing in the repository
notices that both `CalculateObjectIdentity` and `VerifyObjectIdentity` now attest a
Transfer Manifest whose symlink entry has an empty target — a structurally invalid
object under the pinned `string[1..4096]` declaration.

### Why this is a finding and not a nitpick

Three standing Definition-of-Done rows land on it directly:

- "every remaining survivor is triaged with proof that it is dead code or genuinely
  subsumed, **never with plausibility**" — this row generalizes from one caller
  (`media_type`) to all callers, and the generalization is false.
- "`constraint-enumeration.md` claims only what the suite proves: every row asserting
  a bound is enforced exactly as declared is backed by an executable boundary case,
  or the row states plainly that production enforces it while no test pins it" —
  row 106 asserts `string[1..4096]` is "enforced exactly as declared"; the `1` has no
  executable case.
- "each closed bound has accept-at-bound and refuse-past-bound cases through both
  identity entries" — `target` has accept-at-bound (a one-character target is
  accepted in the workspace-tree fixtures) but no refuse-past-bound at the lower end.

The failing shape has a name: **empty symlink target attested under single-clause
disablement of `closed_shapes.go:1748`**.

### Required rework — test-only, plus one artifact row

Do **not** edit `closed_shapes.go`, any other production file, any accepted leaf, the
validation configuration, README, or the pinned specification. Production refuses
`target: ""` correctly today.

1. Add a public-entry negative case for a `workspace_tree` Transfer Manifest whose
   symlink entry carries `target: ""`, driving both `CalculateObjectIdentity` and
   `VerifyObjectIdentity` and asserting the exact reason
   `member target must be a non-empty UTF-8 string`. It must fail under the mutant
   `closed_shapes.go:1748` -> `if false && (text == "") {`. Pair it with the
   accept-at-bound case (a one-character target) so the `[1..` end of the declared
   bound has both directions, the same standard the `..4096]` end already meets.
2. Correct the triage row for survivor 98. Either state the surviving fact —
   `requireString`'s empty guard is the sole enforcer for the symlink `target`
   caller, and is now pinned by the case above — or drop the universal
   "every caller" phrasing. My probe source and both logs are attached; the eight
   genuinely-subsumed members are enumerated there, so the corrected row can be
   exact rather than hedged.
3. Re-run the clause sweep and confirm survivor 98 (`line1748`) flips to KILLED and
   nothing else regresses; re-run the rev12 `genmutants.py` derived sweep and keep it
   at zero actionable survivors. Attach both outputs.

This is bounded, not another ladder rung: it is the same clause sweep and the same
survivor list rev14 already defined, with one row of that triage executed on a false
premise. Reproduce with my attached harness rather than re-deriving.

## Not findings

- **The other 24 triage rows.** Verified above; each is backed by the sweep plus an
  executable case that targets that clause's input class, or by an arithmetic /
  decoder unreachability argument I checked myself.
- **`TestPublicIdentityRefusesNilShapeValidatorRegistryEntry`** mutates
  `immutableObjectShapeValidators` and restores it in `t.Cleanup`; it is not
  `t.Parallel()`, and the `-race` gate is green.
- **Fuzz wiring.** All four `Fuzz*` targets are in the configured validation list at a
  fixed `-fuzztime=100x`, and
  `TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget` derives that
  requirement from the repository AST, so a new unwired target fails the suite.
- **Coverage moved 83.6% -> 87.1%** in `canonicaljson`, consistent with a test-only
  revision.

## Reviewer honesty note

I re-executed the full 110-site clause sweep myself (85/25, matching the producer)
and every configured gate above. I did **not** re-execute all 71 numeric mutants; I
verified that the corpus regenerates byte-identically from `genmutants.py` against
this exact `closed_shapes.go` and that the attached results tally 56/15 over the same
survivor set rev13 and rev14 independently reproduced. That is a binding check, not a
re-run, and it is reported as such.

Separately: the machine's root volume reached 100% (148 MiB free, 926 GiB disk)
partway through this review, with 54 GB of it in the Go build cache accumulated
across these mutation sweeps. I ran `go clean -cache` to continue. That is a
regenerable cache and no repository or board state was touched, but it is recorded
here because it happened during the run.

## Scope instruction for the next producer

Test-only, plus the one triage row. One negative case and one accept-at-bound case
for `ManifestEntry.symlink.target`, an honest row for clause-sweep survivor 98, both
sweeps re-run and attached. Nothing else.
