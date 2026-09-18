# TASK-260830-2g5be6 Results — implement-native-capture-and-source-race-check (rev3)

Status: ready for review. Candidate left UNCOMMITTED in the Story
worktree (`task-board/story/STORY-260830-1cyj0q`) on the predecessor
checkpoint `2f844bb` (Story branch over trunk `a12d1bd`); no commit
on the Story branch. `origin/main` is `c3aae73` at handoff, unchanged
since spawn, so no `refresh-candidate` ran.

Revision 3 answers review RUN-260918-fdce92 (CHANGES REQUESTED on
rev2): production was correct on all 36 probes; the block was the
evidence contract — the published 17-of-17 member-level ratio
measured 15 of 17 (row 8 sealed unstable pre/post digests unpinned,
row 10 seal-before-gate unpinned at `CaptureAndProject`), plus four
P3 sub-class/attribution items. This rework is two strengthened
tests, three new tests, nine new harness rows, and TRACEABILITY
attribution fixes. No production logic changed: the only production
edit is the `project.go` ORDER-PIN comment, whose "and on any
emitted receipt blob" clause was inert (that plant never emits a
receipt) and now states the seal-error kill plus the new seal row.

## Delivered (rev1 scope, unchanged)

`internal/clonesnap` implements the capture operation over a real
on-disk provider store (second leaf of STORY-260830-1cyj0q):
contained walk through the `secprim` Guard, nine-class
classification with always-excluded bytes never opened, Capture
Boundary from measured evidence, the source-race gate before any
seal or projection output, workspace checkpoint + source head from
observed state, Section 10.2 descriptors with `capability_unavailable`
past 128 GiB, and target admission through the landed G2 and
maximal-safe gates. Every artifact seals through the landed
`clonebundle` builders and re-decodes before publish or return; no
second model exists. No `ax` CLI, provider process, or network; no
`internal/traceability` edit (FINAL leaf carries bindings + re-pin).

## Finding-by-finding rework table (rev2 verdict → rev3 change)

| Finding | Change | Test that fails without it | Evidence path |
|---|---|---|---|
| P2-α Row 8 (sealed unstable pre/post digests unpinned; `R2-unstable-seals-pre-as-post` SURVIVED) | `TestCaptureArchiveSealsUnstable` now asserts the SEALED boundary's `pre_capture_digest == result.PreDigest`, `post_capture_digest == result.PostDigest`, and sealed `source_generation == request.Generation` (decoded `CaptureBoundary` fields) | Same test FAILs under `N-unstable-seals-pre-as-post` (`race_test.go:168: sealed post_capture_digest = sha256:7113…, measured post = "sha256:d7e2…"`) | `mutants/run1/N-unstable-seals-pre-as-post.log`, `mutants/run2/N-unstable-seals-pre-as-post.log` (`# exit=1`, `--- FAIL: TestCaptureArchiveSealsUnstable`) |
| P2-α Row 10 ("before any seal" pinned at `Capture` but not at `CaptureAndProject`; `R2-project-seal-before-gate` SURVIVED) | `TestCaptureRaceEmitsNoReceipt` now scans the CAPTURE store for any decodable raw or capture manifest, the way `TestCaptureRaceSealsNoManifest` does | Same test FAILs under `N-race-order-project-seal` (`race_test.go:258: mutated capture sealed a raw manifest before the race gate`); `N-race-order-project` stays KILLED by the seal error message (`misses "source mutated during capture"`) | `mutants/run1/N-race-order-project-seal.log`, `mutants/run2/N-race-order-project-seal.log` (`# exit=1`); `N-race-order-project.log` shows the seal-error kill |
| P3-a (unplanned SPECIAL member unmeasured) | New `TestCaptureRefusesUnplannedSpecialMember`: an unplanned symlink `stray-link` to an outside secret refuses `not a plan candidate` naming the member, with no outside byte installed and zero blobs | Same test FAILs under `N-unplanned-special` (plant treats exactly `stray-link` as an intermediate; capture seals stable) | `mutants/run1/N-unplanned-special.log`, `mutants/run2/N-unplanned-special.log` (`# exit=1`) |
| P3-b (symlinked STORE root unmeasured) | New `TestCaptureRefusesSymlinkedStoreRoot`: refusal names `link-root` and `symlink`; zero blobs installed | Same test FAILs under `N-store-root-symlink` (narrowing keyed on the link name: follows the trailing link for exactly `link-root`, every other symlinked root still refuses) | `mutants/run1/N-store-root-symlink.log`, `mutants/run2/N-store-root-symlink.log` (`# exit=1`) |
| P3-c (excluded-member size claim untested) | Shipped the row (not the reword): new `TestCaptureRaceExcludedSizeChange` — the excluded `store/token-cache` size change refuses naming the member | Same test FAILs under `N-excluded-size` (exactly `store/token-cache` contributes a constant size; capture seals stable) | `mutants/run1/N-excluded-size.log`, `mutants/run2/N-excluded-size.log` (`# exit=1`) |
| P3-d (branch bound credited to the local check) | TRACEABILITY 1.6 row now states the measured gate is the landed `DecodeWorkspaceBinding` and the local `workspaceNullableText` is defence in depth | Behaviour pinned at the entry by `TestCheckpointWorkspaceMultibyteBranch`; a first-hand `maximum+1` narrowing SURVIVES with exit 0 because the landed decoder refuses first | `attribution/branch-bound-landed-decoder.log` (`# exit=0`, `--- PASS`) |
| P3-e (four pinned gates with no shipped row) | Shipped all four rows (not the pinned-without-row list): `N-plan-grammar-dotdot`, `N-required-blocked-ancestor`, `N-fingerprints-129`, `N-first-chunk-oversize` | Killers `TestCaptureRefusesParentPlanKey`, `TestCaptureRefusesSymlinkedIntermediate`, `TestCheckpointWorkspaceRefusesTooManyFingerprints` (local `maximum is 128` literal), `TestCaptureMultiChunkMember` each FAIL under their plant | `mutants/run1/N-plan-grammar-dotdot.log`, `N-required-blocked-ancestor.log`, `N-fingerprints-129.log`, `N-first-chunk-oversize.log` (+ `run2/` copies, `# exit=1`) |

## AC coverage: 17 of 17 rows driven and measured

Measured after the fixes, one row per line — row, driving test(s)
through the production entry, narrowing(s) that redden the row's
killer(s) in the shipped battery (every row executed KILLED 2/2
below; per-plant raw logs carry subprocess exits):

| # | Driving test(s) | Narrowing(s) reddening the killer(s) |
|---|---|---|
| 1 | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureIsByteIdenticalAcrossRuns` | `N-plan-sanitize` (sanitized-key member), `N-descriptor-empty`/`N-descriptor-offset`/`N-first-chunk-oversize` (descriptor member), `N-payload-install-blob-b` (blob presence); sealed members asserted literally |
| 2 | `TestCaptureRefusesSymlinkedIntermediate/OptionalSymlinkedIntermediate/TrailingSymlink/FIFO/OptionalFIFO/DirectoryMember/ParentPlanKey/SymlinkedStoreRoot` | `N-contain-trailing`, `N-contain-intermediate`, `N-fifo`, `N-optional-blocked-ancestor`, `N-required-blocked-ancestor`, `N-plan-grammar-dotdot`, `N-store-root-symlink` |
| 3 | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureOptionalAbsentSealsExcluded`, `TestCaptureRefusesMysteryPlanClass/UnplannedMember/PrefixSibling/RequiredMissing/DuplicatePlanKey/UnplannedSpecialMember` | `N-plan-class`, `N-unplanned`, `N-plan-prefix` (token-preserving), `N-required`, `N-optional`, `N-plan-order`, `N-unplanned-special` |
| 4 | `TestCaptureExcludesSecrets` | `N-exclude-credential`, `N-exclude-machine-auth`, `N-exclude-runtime-state`, `N-exclude-transient-lock`, `N-excluded-orphan-blob` (add-write) |
| 5 | `TestCaptureUnknownMakesRawIncomplete`, `TestCaptureOptionalAbsentUnknownMakesRawIncomplete`, `TestProjectMaximalSafeRequiresComplete/MaximalSafeAdmitsComplete/StrictExactAdmitsUnknown` | `N-maximal`, `N-clonebundle-unknown-excluded` (cross-package) |
| 6 | `TestCaptureSealsBothManifestsFromStoreBytes`, `TestCaptureImmutableSnapshotProof`, `TestCaptureRefusesUnknownProofKind/IdleCouplingViolation/SnapshotIdentityCoupling/EmptyGeneration` | `N-explicit` (generation member), sealed proof digests asserted literally (`proof.PreCaptureDigest == result.PreDigest`); on the stable path the gate proved pre == post before the seal, so sealing either measurement is byte-identical (equivalence shape, not a gate) |
| 7 | `TestCaptureRaceChangedByte` | `N-race-narrow` |
| 8 | `TestCaptureArchiveSealsUnstable` (NEW sealed pre/post/generation assertions), `TestCaptureArchiveRequiresExplicit`, `TestProjectRefusesUnstable` (refusal + target sink empty), `TestProjectRefusesUnstableDirect` | `N-unstable-seals-pre-as-post` (NEW), `N-explicit`, `N-g2`, `N-archive-emits-receipt` (add-write) |
| 9 | `TestCaptureRaceChangedByte/ChangedSize/ExcludedSizeChange/AppendedRecord/ReplacedFile/RemovedFile/UnreadableAtPost`, `TestCaptureArchiveSealsUnstable` | `N-race-narrow`, `N-excluded-size` (NEW), `N-post-fallback-captured` |
| 10 | `TestCaptureRaceSealsNoManifest`, `TestCaptureRaceEmitsNoReceipt` (NEW capture-store scan) | `N-race-order-capture` (Capture entry, manifest scan), `N-race-order-project-seal` (NEW, projection entry, manifest scan), `N-race-order-project` (projection entry, seal-error kill) |
| 11 | `TestCheckpointWorkspaceSealsObservedState/RelativeCwd/RootCwd/UnsortedFingerprints/DuplicateFingerprint/TooManyFingerprints/BadDigest/BadWorkspaceID`, `TestCaptureSealsBothManifestsFromStoreBytes` | `N-workspace-fingerprints`, `N-fingerprints-129` (NEW, local `maximum is 128` literal) |
| 12 | `TestCheckpointWorkspaceRefusesEscape/DotSegments/MissingCwd/FileCwd/SymlinkRoot` | `N-workspace-cwd` (defence-in-depth, message-keyed) |
| 13 | `TestCaptureExternalNativeBasis`, `TestCaptureRefusesMixedSourceBasis/UnknownBasisKind/UnsanitizedExternalRef/InvalidSourceIdentity` | entry re-validation pinned by `TestCaptureRefusesInvalidSourceIdentity` (five undecodable identities, no manifest published); basis grammar refusals asserted literally at the entry |
| 14 | `TestCaptureInstallsEveryPayloadBlob`, crash replay assertion in `TestCaptureCrashChildSelfTerminates`, `TestProjectIsIdempotent`, `TestCaptureHookFires` | `N-payload-install-blob-b`, `N-hook-crash` |
| 15 | `TestCaptureEmptyMemberSealsEmptyBlob/MultiChunkMember/DescriptorAgreementEndToEnd/RefusesOversizedMember/RefusesSparseOversizedMember`, `TestDefaultMaxSingleBytesIs128GiB` | `N-descriptor-empty`, `N-descriptor-offset`, `N-first-chunk-oversize` (NEW), `N-oversize` |
| 16 | `TestCaptureLogRecordsDigestsOnly`, `TestCaptureExcludesSecrets` (log + error scan) | `N-log-leak`, `N-excluded-orphan-blob` (add-write) |
| 17 | `TestProjectAdmitsStableEveryProfile`, `TestProjectRefusesUnknownProfile` | `N-fidelity` |

Ratio: **17 of 17 AC rows driven and measured at the member
level** — every row's named members are reached by a committed test
through the production call site the matrix names, and every row's
gate carries at least one narrowing the battery reddens 2/2. (Rows
8 and 10 are the two this round closed; rows 2, 3, 9, 11, 15 each
gained a sub-class narrowing.)

## Inherited advisories (unchanged from rev2)

- P3-ε: CLOSED here (excluded unknown test + cross-package row).
- P3-ζ: DEFERRED to TASK-260830-32ypr2 (no text admission site).
- P3-η: PARTIAL — entry re-validation pinned here; standalone
  `EncodeNativeIdentity` contract deferred to the clonebundle owner
  or the Final leaf.

## Validation reran (this run, final tree)

- Commands 1-3 (gofmt check, build, vet): exit 0 (`cmd01.log`,
  `cmd02.log`, `cmd03.log`)
- Command 4 (`go test ./... -count=1`): exit 0, 39 packages ok,
  zero FAIL (`cmd04.log`)
- Command 5 (race gate): exit 0 in eight bounded package groups
  a–h (6+4+4+5+4+4+11+1 packages ok, zero `DATA RACE`, zero FAIL)
  instead of one 25m call, per the headless time bound
  (`cmd05a.log` … `cmd05h.log`; `sessquery` alone in group H)
- Command 6 (cover gate): exit 0 in two bounded groups (20+19
  packages ok, zero FAIL) (`cmd06a.log`, `cmd06b.log`;
  `internal/clonesnap` 87.5% statements)
- Commands 7-20 (fuzz smokes, 14 targets): exit 0 (`cmd07.log` …
  `cmd20.log`)
- Command 21 (`tracecheck`): exit 0, bindings=68, clauses 49/569 —
  unchanged, `internal/traceability` untouched (`cmd21.log`)
- Command 22 (`cataloggen -adopted -check`): exit 0, empty output
  (`cmd22.log`)
- Commands 23-24 (`GOOS=linux/windows` builds): exit 0
  (`cmd23.log`, `cmd24.log`)
- Command 25 (JSON validation): exit 0 (`cmd25.log`)
- Command 26 (`task-board validate`): exit 0; the 184 reported
  issues are pre-existing `[MISSING_ACTIVITY]` warnings on
  unrelated tasks, none naming this task (`cmd26.log`)
- Command 27 (`git diff --check`): exit 0 (`cmd27.log`)
- Package suite: exit 0 (`pkg-test.log`), 87.5% statements
  (`pkg-cover.log`); package race exit 0, zero `DATA RACE`
  (`pkg-race.log`)
- Mutant harness: 44/44 ok in each of two runs (43 KILLED + 1
  SURVIVED control) with per-plant raw logs carrying subprocess
  exits (`mutants/run1/*.log`, `mutants/run2/*.log`,
  `mutants-run1.log`, `mutants-run2.log`),
  `PYTHONDONTWRITEBYTECODE=1`, no `__pycache__`, tree restored
  (`git status` shows only candidate paths after both runs)
- Branch-bound attribution probe: first-hand `maximum+1`
  narrowing SURVIVES with exit 0 (landed decoder refuses first),
  tree restored byte-identical
  (`attribution/branch-bound-landed-decoder.log`)

## Mutant rows (41 narrowing + 2 add-write + 1 control)

Narrowings (gate stays present, admits exactly one forbidden
member): `N-contain-trailing`, `N-contain-intermediate`, `N-fifo`,
`N-unplanned`, `N-plan-prefix` (token-preserving, killer drives
full `Capture`), `N-plan-order`, `N-plan-sanitize`, `N-plan-class`,
four `N-exclude-*`, `N-required`, `N-optional`, `N-race-narrow`,
`N-race-order-capture`, `N-race-order-project`,
`N-race-order-project-seal` (NEW), `N-payload-install-blob-b`,
`N-optional-blocked-ancestor`, `N-post-fallback-captured`,
`N-clonebundle-unknown-excluded` (cross-package), `N-explicit`,
`N-g2`, `N-maximal`, `N-fidelity`, `N-oversize`,
`N-descriptor-empty`, `N-descriptor-offset`,
`N-workspace-cwd` (defence-in-depth, message-keyed),
`N-workspace-fingerprints`, `N-log-leak`, `N-hook-crash`,
`N-unstable-seals-pre-as-post` (NEW), `N-unplanned-special` (NEW),
`N-store-root-symlink` (NEW, keyed on the link name),
`N-excluded-size` (NEW), `N-plan-grammar-dotdot` (NEW),
`N-required-blocked-ancestor` (NEW), `N-fingerprints-129` (NEW),
`N-first-chunk-oversize` (NEW).
Add-writes (gate stays present, exactly one forbidden write added):
`N-archive-emits-receipt`, `N-excluded-orphan-blob`. Control:
`C-doc-comment` SURVIVED.

## Bounds for siblings (not waived)

- No `ax` CLI, provider process, or network (fixture store);
  Section 7.8 operation wiring stays `sessadapter`.
- The sealed stable proof carries the two measured source digests
  (pre and post); the race gate proved them equal before the seal
  ran. The ordering mutants prove the gate precedes the seal:
  `N-race-order-capture` (manifest scan), `N-race-order-project`
  (seal error message), `N-race-order-project-seal` (manifest
  scan at the projection entry).
- Excluded-byte equal-size changes are below the detection floor
  (never opened by design); presence, shape, and size changes
  still race (size pinned by `TestCaptureRaceExcludedSizeChange`
  + `N-excluded-size`).
- Sub-walk transient swaps restored before post-measure are below
  the floor (no fs snapshot).
- Optional absent candidates seal as excluded
  (`plan_optional_absent`); `MaxSingleBytes` mirrors the 7.8
  `ResourceLimits` per-object bound (default 137438953472).
- The admission receipt proves admission and ordering only;
  projection planning past admission is the final leaf's.
- The branch `string[1..1024]` bound is measured at the landed
  `DecodeWorkspaceBinding`; the local `workspaceNullableText`
  check is defence in depth (first-hand survival probe above).
- Fidelity vocabulary spelled twice (P3-ζ′); `EncodeNativeIdentity`
  standalone contract (P3-η remainder); lone-surrogate text row
  (P3-ζ) — all deferred to the Final leaf as bounds above.
- `internal/traceability` bindings + re-pin (FINAL leaf).

## Attachments

- `TASK-260830-2g5be6_conformance-matrix.md`
- `TASK-260830-2g5be6_producer-evidence.tar.gz` (27-command suite
  logs, package suite, 2×43+1 mutant battery with per-plant logs,
  branch-bound attribution probe, file manifest)
