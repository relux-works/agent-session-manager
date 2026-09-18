# TASK-260830-2g5be6 — Review verdict, Change Request revision 3

Reviewer run: RUN-260918-4b4c38 (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260830-2g5be6-3` revision 3, base
`2f844bb49702a860c1199a6a2a0ca6a5cc878197` (the predecessor's checkpoint on the
Story branch), candidate tree `9114b46637f08df16d92727e869bfc846a029069` (22
paths; patch sha256 `6ce030b3b0f6be1329e194968760e6cbc774ae6f137a68199fbb9d03b2cf7c1e`,
verified byte-equal to `git diff 2f844bb 9114b466`). The live Story worktree's
temp-index tree OID (`git add -A` into a scratch index copy) equalled
`9114b466…` before this review and `git status` stayed ` M LOGBOOK.md`,
` M README.md`, `?? internal/clonesnap/` throughout; the worktree, index, branch
and HEAD were never touched. `origin/main` is still `c3aae73` (fetched; no
refresh was due). Authority: `internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (lines
10331–10546), §10.2 (4851–4903), §7.8 (3738–3924, `ResourceLimits` /
`CapturePlanItem` / capture-plan and capture bodies), re-extracted and
diffed member-for-member against the candidate. (As in rev1 and rev2, the
reviewer brief's "Section 4.C / terminalbackend" paragraph is a copy from
another leaf; this review covers the capture scope the task and producer
brief define.)

## Verdict: ACCEPTED → `accept_cr(TASK-260830-2g5be6, revision=3)`

No P1, no P2. Both revision-2 P2 rows are closed and MEASURED closed on this
tree: every rev2 survivor re-planted here is KILLED 2/2 by the whole committed
suite (section 0), and the ratio the producer publishes — 17 of 17 AC rows
driven and measured at the member level — is backed by my own battery under
the definition the rev2 verdict used (every member a row NAMES is reached by
a committed test through the named production entry, and a narrowing of that
member reddens it). Production is correct on every probe I drove (25 new
probe lines on the pristine tree, section "What is clean"): no admit-hole, no
containment escape, no leak, no fork of a landed gate that changes behaviour.

What remains is seven P3 evidence gaps and three informational items
(section "Findings"): sub-members, axes and sibling refusal arms that no
committed row reaches, each proven by a plant that SURVIVED the whole suite
2/2 with its admitted behaviour executed under the plant. None of them is a
member an AC row names, none changes production, and each is one test plus
one harness row. By the precedent this Story set (the predecessor's rev6 was
accepted with three P3 measurement gaps recorded for the sibling leaves) they
are recorded here as inherited advisories for the Story's FINAL leaf
TASK-260830-32ypr2 or a follow-up on `internal/clonesnap`, not as grounds for
a fourth revision. The orchestrator routes them; this verdict names each with
its exact test shape.

Method: isolated immutable copies of the exact candidate tree (`git archive
9114b466…` → `.temp/TASK-260830-2g5be6/review-rev3/{pristine,candidate}` in
the Story worktree's `.temp`, with the tracked `.task-board/.resources` and
`.task-board/.activity` artifacts dropped from both copies to keep scratch
small — the `specpin` board survey needs only `.task-board/**/STORY-*/README.md`,
which stayed); `diff -rq` clean against `pristine` before use and after every
battery, probe and hygiene run. Every probe and plant drove a production entry
(`Capture`, `CaptureAndProject`, `AdmitForTarget`, `BuildAdmissionReceipt`,
`CheckpointWorkspace`); `PYTHONDONTWRITEBYTECODE=1` for every harness, zero
`__pycache__`/`.pyc` in any copy or in the candidate tree. Raw evidence:
`TASK-260830-2g5be6_review-evidence-rev3.tar.gz` (`logs/`, one raw `go test`
log per plant per run under `logs/mutants/run{1,2}/` (31 each) and
`logs/mutants-producer/run{1,2}/` (44 each), `logs/under-mutant/` executions
of each survivor, `reviewer_mutants_rev3.py`, `run_under_mutant.py`, the two
probe files as `.txt`, `logs/probes-pristine.log` (25 `PROBE` lines),
`logs/crash-rev3-run{1,2}.log`, `logs/refusal-site-census.txt`,
`logs/hygiene.log`, `MANIFEST.sha256`).

## 0. Rework verification (rev2 findings against the NEW tree)

| rev2 finding | rev3 change | re-plant (whole suite, 2 runs) | Verdict |
|---|---|---|---|
| P2-α row 8 (sealed unstable pre/post digests unpinned) | `TestCaptureArchiveSealsUnstable` asserts decoded `CaptureBoundary.Pre/PostCaptureDigest == result.Pre/PostDigest` and `SourceGeneration == request.Generation` (`race_test.go:161-175`) | `R2-unstable-seals-pre-as-post` **KILLED 2/2** (killer `TestCaptureArchiveSealsUnstable`, `race_test.go:168: sealed post_capture_digest = sha256:7113…, measured post = "sha256:d7e2…"`) | closed (see P3-a for the labelling sub-member) |
| P2-α row 10 ("before any seal" unpinned at `CaptureAndProject`) | `TestCaptureRaceEmitsNoReceipt` scans the CAPTURE store (`storeSink`, the `request.Store` sink — verified it is the capture store, not the target) for any decodable raw or capture manifest (`race_test.go:256-263`) | `R2-project-seal-before-gate` **KILLED 2/2** (killer `TestCaptureRaceEmitsNoReceipt`, `race_test.go:258: mutated capture sealed a raw manifest before the race gate`) | closed |
| P3-a unplanned SPECIAL member | `TestCaptureRefusesUnplannedSpecialMember` (unplanned symlink `stray-link` → outside secret; refusal names it, no outside byte, zero blobs) | `R2-unplanned-special-admitted` **KILLED 2/2** | closed |
| P3-b symlinked STORE root | `TestCaptureRefusesSymlinkedStoreRoot` (refusal names `link-root` + `symlink`, zero blobs) | `R2-store-root-follows-symlink` (the rev2 widening, `os.Open` at the root) **KILLED 2/2** | closed |
| P3-c excluded member size change | `TestCaptureRaceExcludedSizeChange` (shipped the row, not the reword) | `R2-excluded-size-ignored` **KILLED 2/2** | closed |
| P3-d branch bound attribution | TRACEABILITY 1.6 row names the landed `DecodeWorkspaceBinding` as the measured gate, local check = defence in depth, with a first-hand survival probe (`attribution/branch-bound-landed-decoder.log`, exit 0) | `R2-branch-1025` **SURVIVED 2/2** as attributed (the landed decoder is the backstop). Note: on the pristine tree the LOCAL check fires first (`workspace.go:179` executed, the decoder refusal at `workspace.go:100` is never reached by the suite); the decoder "refuses first" only under the narrowing — wording, not substance | closed |
| P3-e four pinned gates with no row | `N-plan-grammar-dotdot`, `N-required-blocked-ancestor`, `N-fingerprints-129`, `N-first-chunk-oversize` shipped | my `R2-plan-grammar-admits-dotdot`, `R2-required-blockedAncestor-dropped`, `R2-fingerprints-129`, `R2-first-chunk-oversize` all **KILLED 2/2** | closed |
| Ratio restated | results/matrix/LOGBOOK publish 17 of 17 with the killer of every row named, and the finding-by-finding table | measured, see "Measured ratio" | closed |

The four new harness rows measure what their names claim: `N-unstable-seals-pre-as-post`
is a narrowing (the `state.pre` digest sealed for `post_capture_digest`, the
form otherwise intact); `N-race-order-project-seal` seals+publishes first,
gates second and PRESERVES the race error (its kill line is the capture-store
scan, `race_test.go:258`, while `N-race-order-project`'s kill line is the seal
error message, `race_test.go:252` — exactly as the producer states in the row
note, the TRACEABILITY bound and the results); `N-unplanned-special` and
`N-store-root-symlink` are name-keyed narrowings (`stray-link`, `link-root`);
`N-excluded-size` keys on `store/token-cache`. Also re-run from rev1/rev2:
`R1-*` five survivors + the widening + the seam control, `R2-capture-seal-before-gate`,
all KILLED 2/2.

## Findings

No P1. No P2.

### P3 — sub-members, axes and sibling arms with no committed row (production correct in every case; each plant SURVIVED the whole suite 2/2 and its admitted behaviour was executed under the plant, `logs/under-mutant/*.log`)

- **P3-a Row 8 digest LABELLING is pinned against the pipeline's own claim.**
  `R3-archive-swaps-pre-post` (the unstable form seals the post-capture
  measurement as `pre_capture_digest` and vice versa, and `CaptureResult`
  reports the same swap) **SURVIVED 2/2**. The rev3 assertion compares the
  sealed digests to `result.PreDigest`/`PostDigest` — the pipeline's own
  values (shape (a) of the producer brief: an expectation derived from
  production). Under the plant (probe PA): `sealed_pre == quiet_d0 = false`,
  `sealed_post == quiet_d0 = true`, `sealed_pre == result_pre = true` — the
  committed test passes while the archive manifest labels the digests
  backwards. Pristine: `sealed_pre == quiet_d0 = true`. The independent
  oracle is cheap and already in the suite's vocabulary: a quiet `Capture`
  over an identical unmutated fixture seals `pre == post == D0`; assert the
  archive's `pre_capture_digest == D0` and `post_capture_digest != D0`. Ship
  a harness row of the `R3-archive-swaps-pre-post` shape.
- **P3-b RaceArchive over a QUIET source is unmeasured (rule pinned along one
  axis).** `R3-archive-policy-always-unstable` (under `RaceArchive` +
  `OperatorExplicit` every capture seals `unstable_archive` before the gate
  runs) **SURVIVED 2/2**. Every committed archive-policy row injects a
  mutation; none drives the policy over an unmutated store. Under the plant
  (probe PB) a quiet store seals `kind=unstable_archive` carrying
  `source_not_quiescent` with `pre == post`, and target admission refuses it;
  pristine seals `stable` and admits. One row: `OnRace=RaceArchive`,
  `OperatorExplicit=true`, no hook → `BoundaryKind == "stable"`,
  `AdmitForTarget` admits.
- **P3-c A failure to re-list the store at POST time is unpinned (absence vs
  failure to read).** `R3-post-enumerate-failure-as-quiet` (`measurePost`
  treats an enumeration error as "no change") **SURVIVED 2/2**. Under it
  (probe PD, `store/` chmod 000 in `AfterWalk`) the capture **seals `stable`,
  `raw_complete=true`, and publishes both manifests**; pristine refuses
  `capture cannot enumerate store member "store": … permission denied` with
  nothing published. The committed "unreadable at post" row
  (`TestCaptureRaceUnreadableAtPost`) reaches the MEMBER-unreadable arm
  (`openMember` fails → unknown content → digests differ), not the
  enumeration arm (`walk.go:85` via `measurePost`, never executed by the
  suite — `logs/refusal-site-census.txt`). One row (skip when euid 0):
  chmod 000 the `store/` directory in `AfterWalk`, restore in `t.Cleanup`,
  assert the refusal names `"store"` and the capture store holds no manifest.
- **P3-d A subdirectory that cannot be listed at WALK time is unpinned.**
  `R3-enumerate-unreadable-as-empty` (an unreadable subdirectory enumerates
  as empty) **SURVIVED 2/2**. Under it (probe PE) a planned optional member
  inside a chmod-000 directory seals as `plan_optional_absent` and the
  manifest carries `raw_complete=true` — a completeness claim laundered from
  a read failure; pristine refuses `capture cannot enumerate store member
  "locked"` with zero blobs. The `walk.go` doc comment "A failed observation
  is a refusal, never an absence" is prose that outruns the tests (shape (f)).
  One row, or restate as a bound.
- **P3-e The race window is measurable only through the `AfterWalk` seam —
  state the bound.** `R3-post-measure-only-with-hook` (`measurePost` runs only
  when `Hooks.AfterWalk != nil`; every hook-less capture reports `post ==
  pre` without re-reading) **SURVIVED 2/2**. No deterministic seam-free
  injection exists (a concurrent mutator cannot be ordered against the
  pipeline without a hook), so this is a bound to STATE in TRACEABILITY and
  the results ("the mutation window is exercised only through the AfterWalk
  seam; `measurePost` is unconditional by inspection"), not a row to add. It
  is not stated today.
- **P3-f Exported `BuildAdmissionReceipt` standalone profile gate unpinned
  (the P3-η shape at this leaf).** `R3-receipt-profile-unchecked` (the
  standalone entry admits exactly profile `ultra`) **SURVIVED 2/2**; under it
  (probe PF) `BuildAdmissionReceipt(…, "ultra")` seals a receipt carrying
  `"fidelity_profile":"ultra"`; pristine refuses. The composed path is pinned
  (`AdmitForTarget` refuses first, `N-fidelity` KILLED), but the exported
  entry advertises its own check that no test drives (`project.go:181` never
  executed). One direct call, or state the bound.
- **P3-g Two smaller unrowed refusal sub-classes.** (i) `R3-empty-cwd-as-root`
  (an EMPTY workspace cwd seals as `"."`) **SURVIVED 2/2** — probe PH: the
  binding seals `cwd_relative:"."` for an unobserved cwd; pristine refuses
  `cwd_relative is empty` (`workspace.go:113`, never executed). (ii)
  `R3-drop-early-containment-pass` (ARM-DELETE, labelled) **SURVIVED 2/2** —
  the early containment pass's doc claim "this pass fails fast with nothing
  installed" is unpinned: probe PG plants a planned FIFO that sorts AFTER an
  included member; pristine installs 0 blobs before refusing, the plant
  installs 1. Either assert zero blobs in a row with that ordering or reword
  the comment (orphan payload blobs on seal-time refusals are already a
  recorded informational).

### Informational (no action required unless cheap)

- `R3-early-pass-opens-excluded-special` (ARM-DELETE, OVER-STRICT direction)
  SURVIVED 2/2: an excluded-class special member (symlink credential) is the
  only shape where "excluded members are never opened" is measured by nothing
  but the rev2 probe P13a and my probe PI (pristine: `kind=stable`, no leak,
  never opened; under the plant the Guard refuses it — safe direction).
- `R3-nil-target-sink-unchecked` (ARM-DELETE) SURVIVED 2/2: a nil target sink
  refuses before any capture on pristine (probe PJ, capture store empty);
  under the plant the capture publishes both manifests and then fails at the
  receipt with the landed "object store is not initialized" (no panic). rev2
  informational, unchanged.
- The four `N-exclude-*` narrowings are killed by the landed raw-manifest
  builder ("raw object entry[0] class "credential" is outside the raw
  subset"), i.e. at seal time after the excluded bytes were already opened
  and installed as an orphan payload blob; the leak class itself is pinned
  by the add-write `N-excluded-orphan-blob` (blob scan). Attribution only.
  `clonesnap.excludedClass` spells the four-class table that the landed
  `clonebundle.alwaysExcludedClass` (unexported) also spells — the same
  "spelled twice" shape the producer already records for the fidelity
  vocabulary (P3-ζ′); worth listing beside it.
- An unknown fidelity profile is refused only AFTER the capture has sealed
  and published both manifests (`CaptureAndProject` → `sealAndPublish` →
  `admitAndEmit` → `AdmitForTarget`); the manifests are valid and
  content-addressed, so nothing is wrong, but an entry-time
  `validFidelityProfile` check would avoid the durable writes.
- Refusal-site census (`logs/refusal-site-census.txt`, coverprofile over the
  committed suite): 25 of 60 `invalid(` call sites are executed; of the 35
  unexecuted, 26 are defensive or unreachable by construction (serialize /
  canonicalize / re-decode of builder-sealed bytes, `PutBlob` faults on a
  healthy store, negative sizes, read-vs-stat TOCTOU windows, the in-memory
  128 GiB descriptor bound behind the `openMember` limit, the
  "archive unexpectedly admitted" arm behind `AdmitForTarget`) and 9 are
  reachable classes with no committed row: `walk.go:85` (P3-c/d),
  `walk.go:90` (backslash-named member; the plan gate refuses it anyway),
  `capture.go:208` (relative store root: `OpenNoFollowDir` admits a relative
  path that exists, `NewGuard` refuses it — rev2 probe P5), `capture.go:460`
  (`AfterRawPublish` hook returning an error; `TestCaptureRefusesHookError`
  drives `AfterWalk` only), `project.go:87` (nil sink), `project.go:181`
  (P3-f), `workspace.go:113` (P3-g), `workspace.go:118` (`filepath.Rel`
  failure; effectively defensive once the root is absolute), `workspace.go:161`
  (a non-digest remote fingerprint; the landed decoder is the backstop as for
  the branch bound).

## What is clean (verified, not read)

1. **Spec fidelity** (`logs/probes-pristine.log`, re-extracted §13.14.1 /
   §10.2 / §7.8): the nine-class Capture Item vocabulary comes from the
   landed `ValidCaptureClass`; the always-excluded four seal `excluded` with
   null content and a stable reason (`<class>_excluded`), `excluded_classes`
   exactly `[credential machine_auth runtime_state transient_lock]` (S3a);
   `unknown` (included or excluded) drives `raw_complete=false` and
   `maximal_safe` refuses through `CaptureAndProject` with an empty target
   sink (S3c); the stable boundary carries the closed proof kind, generation,
   equal pre/post digests and the three idle facts through the landed
   `buildBoundary`/`checkProofCoupling`; the unstable form carries generation,
   both measured digests (PA pristine: `sealed_pre == quiet_d0`),
   `source_not_quiescent`, `operator_explicit=true`,
   `target_projection_forbidden=true`, is core-created (`R3-archive-not-core`
   KILLED 2/2 by three archive rows) and never enters a target branch (S4 ×4
   shapes through `CaptureAndProject`: `unstable_archive cannot enter a target
   branch`, target sink empty); **size equality is never proof**: equal-size
   differing bytes refuse at both `blob-a` and `blob-b` (S1,
   `TestCaptureRaceChangedByte`), and the rev1 widening `R1-size-equality-is-proof`
   is KILLED 2/2 by 7 tests. WorkspaceBinding: the seven members and bounds
   (`cwd_relative` 1..4096 and `"."` for the root per the landed
   `checkCwdRelative`, fingerprints sorted unique digest[0..128], branch
   string[1..1024] in characters, three nullable digests) validate through the
   landed `DecodeWorkspaceBinding` re-decode. Blob Descriptor: 4 MiB chunks,
   index from zero by one, contiguous from offset zero covering `size`, empty
   blob → `[]`, `application/octet-stream`, identity via
   `CalculateObjectIdentity` + landed `VerifyDescriptorAgreement`
   (`TestCaptureMultiChunkMember`, `TestCaptureEmptyMemberSealsEmptyBlob`,
   `TestCaptureDescriptorAgreementEndToEnd`; `N-descriptor-empty/offset`,
   `N-first-chunk-oversize` KILLED).
2. **Containment**: every payload open is `secprim.Guard.Open` from the
   `OpenNoFollowDir` root handle — the landed openat-relative walk
   (`open_member_unix.go`: `O_NOFOLLOW|O_DIRECTORY` per intermediate,
   `O_NOFOLLOW|O_NONBLOCK` on the final component, ELOOP → "member symlink
   escape"). Symlinked intermediate at depth 2 and 3 refuses `capture member
   "a/b/file": secprim unsafe path: member symlink escape: a/b/file` with
   `leaked=false`, 0 blobs (S2a/S2c); a FIFO at depth 2 refuses `target
   special file: pipe` without a writer, 0 blobs (S2b); the refusal names the
   member, never "open failed". Symlinked store root, unplanned special,
   optional/required members behind a blocked ancestor, trailing symlink,
   directory member: committed rows, all narrowings KILLED.
3. **Exclusion**: one member per always-excluded class with a unique secret;
   0 of 4 secrets in any blob, manifest, workspace binding or log line (S3a);
   a race refusal naming the excluded member carries no secret byte (S3b);
   `N-excluded-orphan-blob` (add-write) KILLED 2/2 proves the blob scan is
   load-bearing; an excluded symlink credential is never followed (PI).
4. **Source race**: changed byte (equal size), changed size, appended record,
   replaced file, removed file, excluded size change, grown past the bound
   — each refuses naming the member through `Capture`, and each seals
   `unstable_archive` refused at the target through `CaptureAndProject`
   under the archive policy (S4); `firstChangedMember` names the member in
   every refusal.
5. **Ordering**: at `Capture` (`N-race-order-capture`, `R2-capture-seal-before-gate`
   KILLED by the store scan) and now at `CaptureAndProject` for both halves
   (`N-race-order-project` by the seal error, `N-race-order-project-seal` and
   `R2-project-seal-before-gate` by the capture-store scan).
6. **Crash/idempotency** (`logs/crash-rev3-run{1,2}.log`,
   `logs/crash-producer-count2.log`): the producer's real-SIGKILL child at
   `AfterRawPublish` through `Capture` PASSES 2/2 and under `-count=2`; my
   SIGKILL at `AfterWalk` (H1, 2/2): 2 payload blobs, 0 manifests, 0 staged
   temp files, 0 quarantine entries, replay installs both manifests with
   every raw entry's blob present and bytes identical to a clean capture; my
   SIGKILL at `AfterRawPublish` through the PROJECTION entry (H2, new seam,
   2/2): 1 raw manifest, 0 capture manifests, 0 staged, target sink empty;
   the replay through `CaptureAndProject` verifies-and-reuses the raw
   manifest, installs the capture manifest and the receipt (4 capture blobs,
   1 target blob), all three byte-identical to a clean projection.
   `TestProjectIsIdempotent` triple replay reuses every blob.
7. **Blob discipline**: a clean capture leaves exactly 4 content-addressed
   blobs (2 payload + 2 manifests) and no non-object file under the data root
   (S5); both sealed manifests re-decode; every raw entry's blob is present
   after `Capture` and after the crash replays (`N-payload-install-blob-b`,
   `R1-skip-payload-install-blob-b` KILLED); durable writes go only through
   the landed `InstallRawBlob`/`PutBlob`.
8. **Admission**: G2, maximal-safe, unknown-profile rows KILLED
   (`N-g2`, `N-maximal`, `N-fidelity`); the excluded-unknown cross-package
   plant `R1-clonebundle-unknown-ignores-excluded` KILLED by this leaf's
   killer.
9. **Workspace**: escapes, dot segments, unobserved/file cwd, unsorted /
   duplicate / 129 fingerprints, bad digest, bad UUID, symlinked root all
   refuse; multibyte branch bound in characters (1024 admits, 1025 refuses).
10. **Producer harness reproduced**: 43 KILLED + `C-doc-comment` SURVIVED,
    twice, in the isolated copy (`logs/mutants-producer/run{1,2}/`, 44 raw
    logs each with `# exit=`; verdict lines identical across runs; tree
    `diff -rq` clean after each run).
11. **Hygiene** (`logs/hygiene.log`, `logs/vet.log`, `logs/gofmt-l.log`,
    `logs/repo-test.log`, `logs/pkg-race.log`): `gofmt -l` empty; `go vet
    ./...` exit 0 (also `GOOS=linux`/`GOOS=windows` vet of the package); `go
    build ./...` exit 0; `GOOS=linux`/`GOOS=windows` builds exit 0;
    `tracecheck` exit 0 (bindings=68, clauses 49/569 — `internal/traceability`
    untouched, 0 diff lines); `cataloggen -adopted … -check` exit 0; every
    tracked JSON parses; `git diff --check` clean both on the worktree and
    tree-to-tree (new files included); package suite 80/80, 0 skips, 87.5%
    statements; package + clonebundle `-race` ok, 0 DATA RACE; whole-repository
    `go test ./... -count=1` 39/39 ok (3m31s). Changed paths are exactly the
    22 candidate paths, nothing outside `internal/clonesnap/`, `README.md`,
    `LOGBOOK.md`; README and LOGBOOK purely additive (0 removed lines); README
    states "no ax command, no doctor result, no runtime capability claim";
    LOGBOOK entry newest-first; no `__pycache__`/`.pyc` in the tree; candidate
    `task-board.config.json` equals HEAD (27 commands). The producer evidence
    archive is a real gzip with 129 `MANIFEST.sha256` rows all OK, 44 per-plant
    logs per run with identical verdicts, the branch-bound attribution probe,
    and the 27-command logs (race gate in 8 bounded groups: 39 `ok`, 0 DATA
    RACE). The CR construction log reports `required=30 green=30 failed=0
    missing=0` against the control root (truncated at 64 KiB by the board);
    accepted as attached evidence for the 25-minute repository race gate,
    which I did not rerun whole — I ran `-race` on the two packages this leaf
    touches.

## Reviewer mutation battery (`reviewer_mutants_rev3.py`, whole committed suite as killer, 2 runs, identical verdicts)

| Row | Kind | Verdict | Killers |
|---|---|---|---|
| R2-unstable-seals-pre-as-post | narrowing | KILLED 2/2 | TestCaptureArchiveSealsUnstable |
| R2-project-seal-before-gate | narrowing (ordering) | KILLED 2/2 | TestCaptureRaceEmitsNoReceipt |
| R2-unplanned-special-admitted | narrowing | KILLED 2/2 | TestCaptureRefusesUnplannedSpecialMember |
| R2-store-root-follows-symlink | widening (root only) | KILLED 2/2 | TestCaptureRefusesSymlinkedStoreRoot |
| R2-excluded-size-ignored | narrowing | KILLED 2/2 | TestCaptureRaceExcludedSizeChange |
| R2-branch-1025 | narrowing | SURVIVED 2/2 (attributed: landed decoder backstop) | — |
| R2-capture-seal-before-gate | narrowing (ordering) | KILLED 2/2 | TestCaptureRaceSealsNoManifest |
| R2-required-blockedAncestor-dropped | narrowing | KILLED 2/2 | TestCaptureRefusesSymlinkedIntermediate |
| R2-plan-grammar-admits-dotdot | narrowing | KILLED 2/2 | TestCaptureRefusesParentPlanKey |
| R2-fingerprints-129 | narrowing | KILLED 2/2 | TestCheckpointWorkspaceRefusesTooManyFingerprints |
| R2-first-chunk-oversize | narrowing | KILLED 2/2 | TestCaptureMultiChunkMember |
| R1-blockedAncestor-optional | narrowing | KILLED 2/2 | TestCaptureRefusesOptionalSymlinkedIntermediate |
| R1-archive-emits-receipt | add-write | KILLED 2/2 | TestProjectRefusesUnstable |
| R1-skip-payload-install-blob-b | narrowing | KILLED 2/2 | TestCaptureInstallsEveryPayloadBlob, TestCaptureCrashChildSelfTerminates |
| R1-post-falls-back-to-captured-hash | narrowing | KILLED 2/2 | TestCaptureRaceUnreadableAtPost |
| R1-clonebundle-unknown-ignores-excluded | narrowing (cross-package) | KILLED 2/2 | TestCaptureOptionalAbsentUnknownMakesRawIncomplete |
| R1-size-equality-is-proof | widening | KILLED 2/2 | 7 tests |
| R1-hook-after-post-measure | seam control | KILLED 2/2 | 13 tests |
| R3-archive-swaps-pre-post | narrowing | **SURVIVED 2/2** | — (P3-a) |
| R3-archive-policy-always-unstable | narrowing | **SURVIVED 2/2** | — (P3-b) |
| R3-post-measure-only-with-hook | narrowing | **SURVIVED 2/2** | — (P3-e, bound) |
| R3-post-enumerate-failure-as-quiet | narrowing | **SURVIVED 2/2** | — (P3-c) |
| R3-enumerate-unreadable-as-empty | narrowing | **SURVIVED 2/2** | — (P3-d) |
| R3-receipt-profile-unchecked | narrowing | **SURVIVED 2/2** | — (P3-f) |
| R3-drop-early-containment-pass | arm-delete (labelled) | **SURVIVED 2/2** | — (P3-g ii) |
| R3-empty-cwd-as-root | narrowing | **SURVIVED 2/2** | — (P3-g i) |
| R3-early-pass-opens-excluded-special | arm-delete (over-strict) | SURVIVED 2/2 | — (informational) |
| R3-nil-target-sink-unchecked | arm-delete | SURVIVED 2/2 | — (informational) |
| R3-archive-not-core | narrowing (landed rule reached from this entry) | KILLED 2/2 | TestCaptureArchiveSealsUnstable, TestProjectRefusesUnstable, TestProjectRefusesUnstableDirect |
| C-stable-seals-pre-as-post | equivalence control | SURVIVED 2/2 (expected) | — |
| C-rev3-comment | harmless control | SURVIVED 2/2 (expected) | — |

18 KILLED, 13 SURVIVED (3 expected, 10 findings-bearing), 0 ERROR; every
row's raw log carries `# exit=` and the plant text; the candidate copy was
`diff -rq`-clean against pristine after both runs.

## Measured ratio

**17 of 17 AC rows driven through the named production entry by a named
committed test, and 17 of 17 measured at the member level** — every member a
row names is reached and a narrowing of it reddens the row's killer (rows 8
and 10, the rev2 gaps, are now KILLED by `TestCaptureArchiveSealsUnstable`
and `TestCaptureRaceEmitsNoReceipt`). Beyond the named members, 8 plants on
sub-members, axes and sibling refusal arms survive (P3-a..g above); rows 8
(labelling; archive-policy × quiet axis), 9/10 (enumeration-failure arm;
seam-only bound), 12 (empty cwd) and 17 (standalone receipt entry) carry
them. Refusal-arm census: 25 of 60 refusal sites executed by the committed
suite, 9 reachable arms without a row (all P3 or informational above).

## Advisories carried forward (for the orchestrator to route — Story FINAL leaf TASK-260830-32ypr2 or a follow-up on `internal/clonesnap`)

1. P3-a: independent oracle for the archive digests (`pre_capture_digest ==
   quiet capture digest`), harness row `R3-archive-swaps-pre-post`.
2. P3-b: one quiet-store row under `RaceArchive` + `OperatorExplicit`
   (`BoundaryKind == "stable"`, admits), harness row
   `R3-archive-policy-always-unstable`.
3. P3-c/d: two chmod-000 rows (store subdirectory unreadable at post; planned
   optional member in an unlistable directory at walk), skip for euid 0,
   harness rows `R3-post-enumerate-failure-as-quiet` /
   `R3-enumerate-unreadable-as-empty`; or restate the `walk.go` "never an
   absence" comment as a bound.
4. P3-e: state the AfterWalk-seam bound in TRACEABILITY and the results.
5. P3-f: one direct `BuildAdmissionReceipt(…, "ultra")` refusal row, harness
   row `R3-receipt-profile-unchecked`; or state the standalone-entry bound
   beside P3-η.
6. P3-g: empty-cwd refusal row; early-pass "nothing installed" row (planned
   special sorted after an included member, zero blobs) or reword.
7. Inherited from rev2/predecessor and still open by design: P3-ζ
   (lone-surrogate at `validText`), P3-η remainder (`EncodeNativeIdentity`
   standalone), P3-ζ′ (fidelity vocabulary spelled twice — now also the
   four-class exclusion table), all deferred to the final leaf as the
   producer states.

Checklist state left by this review: reviewer rows "Implementation matches
AC", "Solution fits project architecture", "Tests green" and "Gate, refusal,
validation, authorization, and attestation behavior attacked, not read" are
checked (each measured true: every named gate carries a killed narrowing;
positive-path-only evidence remains only on the P3 sub-members recorded
above); "If review does not accept the work — verdict evidence added and
status routed" is checked (not applicable on acceptance; the verdict
evidence is attached). Status routed by `accept_cr` to `integrating`.
