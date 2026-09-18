# TASK-260830-2g5be6 — Review verdict, Change Request revision 2

Reviewer run: RUN-260918-fdce92 (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260830-2g5be6-2` revision 2, base
`2f844bb49702a860c1199a6a2a0ca6a5cc878197` (the predecessor's checkpoint on the
Story branch), candidate tree `e7c9a12d1cd487e7697ef3b33f71430ef65f136a` (22
paths; patch sha256 `c0a19165154358956450a216e3702ad7dda76f227ff3afb030e47ba1dec5de37`,
verified equal to `git diff 2f844bb e7c9a12d`). The live Story worktree's
temp-index tree OID (`git read-tree HEAD && git add -A` into a scratch index)
equalled `e7c9a12d…` before this review and `git status` stayed
` M LOGBOOK.md`, ` M README.md`, `?? internal/clonesnap/`; the worktree, index,
branch and HEAD were never touched. `origin/main` is still `c3aae73`
(unchanged since the rework spawn; no refresh was due).
Authority: `internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (lines 10331–10546),
§10.2 (4851–4900), §7.8 (3738–3925), re-extracted and diffed member-for-member
against the candidate. (As in rev1, the reviewer brief's "Section 4.C /
terminalbackend" paragraph is a copy from another leaf; this review covers the
capture scope the task and producer brief define.)

## Verdict: CHANGES REQUESTED → `to-dev`

Revision 2 closes everything revision 1 asked for: all five rev1 survivors are
KILLED 2/2 on the new tree (section 0), the three inherited advisories carry an
explicit disposition, the P3-γ/δ/ε′/ζ′ prose-and-rows items are fixed, and
production is still correct on every probe I drove (36 new probe lines, no
admit-hole, no containment escape, no fork of a landed gate that changes
behaviour). What blocks acceptance is, once more and more narrowly, the
evidence contract: the results, matrix and LOGBOOK restate **"17 of 17 AC rows
measured at the member level — every row's narrowing reddens its killer"**,
and I measure **15 of 17**. Two AC rows carry a named member whose committed
test does not reach it — a narrowing SURVIVES the whole committed suite 2/2
while the row's tests stay green, and the admitted behaviour was executed
under the plant:

- **Row 8** — the `unstable_archive` form's `pre/post digests`: the sealed
  archive manifest can carry the PRE digest for `post_capture_digest` (equal
  digests while flagged `source_not_quiescent`) and `TestCaptureArchiveSealsUnstable`
  stays green; it asserts the RESULT's measured digests, never the sealed ones.
- **Row 10** — "Mutation check runs before any seal or projection output" at
  `CaptureAndProject`: the projection entry can seal and PUBLISH the raw
  manifest before the race gate and still return the race refusal;
  `TestCaptureRaceEmitsNoReceipt` stays green because it inspects only the
  target sink and the error text. The shipped `N-race-order-project` row is
  killed by the *seal error message* (the landed builder refusing unequal
  digests), not by a sink assertion; the `RacePolicy` doc contract "RaceRefuse
  aborts capture with no sealed manifest" is unpinned at that entry.

Both are one assertion plus one harness row each. Three more surviving plants
(P3) name sub-classes with no committed row at all (an unplanned SPECIAL
member, a symlinked store root, an EXCLUDED member's size change — the last
one a TRACEABILITY claim stated as fact). The rework is precise and small
(section "Rework scope").

Method: isolated immutable copies of the exact candidate tree (`git archive
e7c9a12d…` → `.temp/TASK-260830-2g5be6/rev2-review/{pristine,candidate}`,
`.task-board` artifact dropped; `diff -rq` clean against `pristine` before use
and after every battery, probe and hygiene run). Every probe and plant drove a
production entry (`Capture`, `CaptureAndProject`, `AdmitForTarget`,
`CheckpointWorkspace`); `PYTHONDONTWRITEBYTECODE=1` for every harness, zero
`__pycache__`/`.pyc` in any copy or in the candidate tree. Raw evidence:
`TASK-260830-2g5be6_review-evidence-rev2.tar.gz` (`logs/`, one raw `go test`
log per plant per run under `logs/mutants/run{1,2}/` (20 each) and
`logs/mutants-producer/run{1,2}/` (35 each), `logs/under-mutant/` executions
of each survivor, `reviewer_mutants_rev2.py`, `run_under_mutant.py`, the three
probe files as `.txt`, `logs/probes-rev2.log` with 36 `PROBE` lines,
`logs/crash-rev2.log`, `MANIFEST.sha256`).

## 0. Rework verification (rev1 findings against the NEW tree)

| rev1 finding | rev2 change | re-plant (whole suite, 2 runs) | Verdict |
|---|---|---|---|
| P2-α row 14 (§10.2 MUST, no test looked at a payload blob) | `TestCaptureInstallsEveryPayloadBlob` + `assertEveryRawBlobInstalled` in the crash replay | `R1-skip-payload-install-blob-b` **KILLED 2/2** (killers: `TestCaptureInstallsEveryPayloadBlob`, `TestCaptureCrashChildSelfTerminates`) | closed |
| P2-α row 8 (target sink never inspected on the archive path) | `TestProjectRefusesUnstable` keeps its sink and asserts zero blobs | `R1-archive-emits-receipt` **KILLED 2/2** | closed |
| P2-α row 2 (every symlink/FIFO row `Required: true`) | `TestCaptureRefusesOptionalSymlinkedIntermediate`, `TestCaptureRefusesOptionalFIFO` | `R1-blockedAncestor-optional` **KILLED 2/2** | closed |
| P2-α row 9 (unreadable-at-post class had no row) | `TestCaptureRaceUnreadableAtPost` (grown past `MaxSingleBytes`) | `R1-post-falls-back-to-captured-hash` **KILLED 2/2**; my chmod-000 member of the class also refuses on pristine (probe P8) | closed |
| P2-α row 5 / P3-ε (excluded unknown unpinned) | `TestCaptureOptionalAbsentUnknownMakesRawIncomplete` | `R1-clonebundle-unknown-ignores-excluded` **KILLED 2/2**; through `CaptureAndProject(maximal_safe)` proper the excluded unknown refuses with an empty target sink (probe P11a) | closed |
| P2-β inherited advisories silent | results §"Inherited advisories": ε CLOSED (test + cross-package row), ζ DEFERRED to TASK-260830-32ypr2 with reason, η PARTIAL (entry re-validation pinned by `TestCaptureRefusesInvalidSourceIdentity`; standalone contract deferred) | dispositions verified against the tree; the `CaptureRequest.SourceIdentity` doc now states the re-validation | closed |
| P3-γ false "fail closed under a reorder" justification | seals `state.post` for `post_capture_digest`; bound reworded in all four places; `TestCaptureRaceEmitsNoReceipt` asserts the race literal | sealed post == measured post on both paths (probe P10a/P10b); `C-stable-seals-pre-as-post` SURVIVED as an equivalence control must | closed (but see P2-α row 10 for what the new killer actually measures) |
| P3-δ `N-workspace-cwd` message-keyed | row note reworded as defence-in-depth | reproduced KILLED 2/2, note honest | closed |
| P3-ε′ leak rows killed by the builder, not the scan | `N-excluded-orphan-blob` add-write shipped | reproduced KILLED 2/2 by `TestCaptureExcludesSecrets`'s blob scan | closed |
| P3-ζ′ fidelity vocabulary spelled twice | recorded as a bound in `project.go`, TRACEABILITY, matrix, results | present | closed (bound) |

Also re-run from rev1: `R1-size-equality-is-proof` (widening) KILLED 2/2 by 7
tests; `R1-hook-after-post-measure` (seam control) KILLED 2/2 by 12 tests.

## Findings

No P1.

### P2 — evidence contract (claimed member-level ratio is inflated)

- **P2-α "17 of 17 measured at the member level" is 15 of 17.** Two rows name
  a member their tests do not reach; each plant SURVIVED the whole committed
  suite in two independent runs (`logs/mutants/run{1,2}/`) and its admitted
  behaviour was executed under the plant (`logs/under-mutant/`):
  - **Row 8 "unstable_archive carries generation, pre/post digests, …"** —
    `R2-unstable-seals-pre-as-post` (the unstable `BoundaryInput` seals
    `state.pre` for `PostCaptureDigest`, `capture.go:482`) **SURVIVED 2/2**.
    Under it (`logs/under-mutant/R2-unstable-seals-pre-as-post.log`, probe U5):
    `kind=unstable_archive`, measured `pre != post` true, **sealed pre == sealed
    post true, sealed post == measured post false** — the archive manifest
    misrepresents the post-capture state while `TestCaptureArchiveSealsUnstable`,
    `TestCaptureArchiveRequiresExplicit`, `TestProjectRefusesUnstable/Direct`
    all PASS. The test asserts the literal `kind`/`reason_code`/
    `operator_explicit`/`target_projection_forbidden` members and
    `result.PreDigest != result.PostDigest` (the pipeline's measurement, not
    the seal); the generation is pinned only indirectly through `N-g2`. The
    §7.8 `capture` response and §13.14.1 name these two digests as the
    evidence the form exists to carry. Pristine seals the two measured digests
    (probe P10b).
  - **Row 10 "Mutation check runs before any seal or projection output —
    `Capture`, `CaptureAndProject` (ORDER-PIN)"** — `R2-project-seal-before-gate`
    (`CaptureAndProject` calls `sealAndPublish(false)` first, then the race
    gate, and returns the race refusal; the stable capture-manifest seal fails
    closed on unequal digests so no receipt can be emitted) **SURVIVED 2/2**.
    Under it (`logs/under-mutant/R2-project-seal-before-gate.log`, probe U1): a
    race under `RaceRefuse` through the projection entry returns the race
    refusal with **`raw_manifests=1`** published in the capture store
    (pristine: 0) — `TestCaptureRaceEmitsNoReceipt` PASSES (race literal
    present, target sink empty). The same shape at the `Capture` entry,
    `R2-capture-seal-before-gate`, is KILLED 2/2 by `TestCaptureRaceSealsNoManifest`'s
    store scan, so the rule is pinned on one of the two named call sites only
    (the rev1 brief's shape (b)). The shipped `N-race-order-project` row is
    killed by `race_test.go:220: refusal "… stable snapshot proof: … capture
    digests differ …" misses "source mutated during capture"` — the seal
    error message, not a sink; its note's "and on any emitted receipt blob" is
    inert because under that plant no receipt is ever emitted. The `RacePolicy`
    doc ("RaceRefuse aborts capture with no sealed manifest") and the
    `CaptureAndProject` doc ("no projection output is produced before the race
    is decided") are therefore pinned only for the receipt, not for the seal,
    at the projection entry.

### P3 — sub-classes with no committed row; prose that outruns the tests

- **P3-a Unplanned SPECIAL members are unmeasured.** `R2-unplanned-special-admitted`
  (`intermediatePrefix` treats exactly one unplanned symlink `stray-link` as
  an intermediate, `walk.go:220`) **SURVIVED 2/2**. Under it (probe U2) an
  unplanned symlink to an outside secret is silently dropped and the capture
  seals **`stable`, `raw_complete=true`** (no outside byte leaks — the special
  is never opened). Pristine refuses `capture store member "stray-link" is not
  a plan candidate` (probe P1; unplanned FIFO likewise, P2). Every committed
  unplanned row (`zz-extra`, `blob-a-evil`) is a regular file, so the matrix
  row 3 / TRACEABILITY "Unplanned members refuse" holds for regular files only.
- **P3-b A symlinked STORE root has no committed test.** `R2-store-root-follows-symlink`
  (`OpenNoFollowDir` → `os.Open` at `capture.go:201`; widening, labelled) **SURVIVED
  2/2**; under it a symlinked root captures successfully (probe U3). Pristine
  refuses `secprim unsafe path: target symlink: link-root` (probe P4).
  `TestCheckpointWorkspaceRefusesSymlinkRoot` pins the workspace root; nothing
  pins the store root that row 2 "capture never escapes the store root" rests
  on.
- **P3-c TRACEABILITY "presence, shape, and size changes still race" (for
  excluded members) is a claim without a test.** `R2-excluded-size-ignored`
  (exactly `store/token-cache` contributes a constant size to the source
  record, `capture.go:347`) **SURVIVED 2/2**; under it the credential member's
  size change seals `stable` silently (probe U4). Pristine refuses naming
  `store/token-cache` (probe P9b). Every committed race row mutates an
  INCLUDED member or adds/removes a file. Either ship the row or state the
  bound as "presence and shape".
- **P3-d `workspaceNullableText` is defence in depth, credited as the gate.**
  `R2-branch-1025` (`length > maximum+1`) **SURVIVED 2/2**: the 1025-character
  branch is refused by the landed `DecodeWorkspaceBinding` (`identity.go:238`,
  same literal `string[1..1024]`), so `TestCheckpointWorkspaceMultibyteBranch`
  stays green (probe U6: `sealed workspace binding refused: … branch is not a
  string[1..1024]`). Production correct; the TRACEABILITY row "1.6 string
  measure in characters | `workspaceNullableText` via `environ.StringLength`"
  attributes the kill to the local check. State it (the fingerprint-count
  bound is different: `R2-fingerprints-129` is KILLED by the local literal
  "maximum is 128").
- **P3-e Harness census.** Beyond the rows above, the shipped harness carries
  no row for the plan member-grammar gate (`R2-plan-grammar-admits-dotdot`
  KILLED 2/2 by `TestCaptureRefusesParentPlanKey` — `SanitizeNativeKey` admits
  `..`, so `CheckMemberPath` at `walk.go:141` is the only gate), the required
  branch's `blockedAncestor` (`R2-required-blockedAncestor-dropped` KILLED
  2/2, message-keyed on "symlink"), the fingerprint count and the chunk size
  bound (`R2-first-chunk-oversize` KILLED 2/2). All pinned; add them or state
  them.

### Informational (no action required unless cheap)

- `AdmitForTarget` admits a sealed capture manifest with trailing JSON
  whitespace (`…}\n`, probe P6b `err=<nil>`); non-whitespace trailing data,
  raw-as-capture, empty, truncated and a forged `raw_complete:true` beside an
  unknown item all refuse (P6a/c/d/e). This is the landed strict decoder's
  frame rule ("trailing data" = non-whitespace) — clonebundle owner's call,
  not this leaf's.
- Nil `TargetSink` refuses before any capture (probe P3, store empty) — no
  committed test, trivial.
- A store subdirectory made unreadable at post time refuses with the
  enumeration error, not the race literal, and never archives (probe P9d,
  archive policy) — acceptable under "refusal or unstable_archive"; worth a
  line.
- Seal-time refusals still leave payload blobs as content-addressed orphans
  (rev1 informational; unchanged).

## What is clean (verified, not read)

1. **Spec fidelity** (`logs/probes-rev2.log`): the nine-class vocabulary and
   always-excluded four; included/excluded content rule; `plan_optional_absent`;
   excluded unknown → `raw_complete=false` and `maximal_safe` refused through
   `CaptureAndProject` (P11a) while `strict_exact` admits (P11b); stable proof
   digests sealed equal to the measurement on both paths (P10a/b); the archive
   raw manifest carries the pre-mutation bytes (P10c); plan-key grammar
   refuses `..`, `./`, `//`, trailing `/`, `.`, `store/./x`, backslash and
   `%2e%2e` for OPTIONAL keys too, before any install (P7); empty plan over an
   empty store seals 0 entries, `raw_complete=true`, spec-consistent (P16);
   required-ness of an excluded class is inert (P17, matches the stated
   bound).
2. **Containment**: symlinked intermediate (required and optional), trailing
   symlink, FIFO (required and optional), directory member, excluded symlink
   to an outside secret never followed (P13a `leaked=false`), symlinked store
   root refused, relative store root refused (P5), backslash-named member
   refused (P12). Producer rows `N-contain-*`, `N-fifo`, `N-optional-blocked-ancestor`
   KILLED 2/2.
3. **Exclusion**: `TestCaptureExcludesSecrets` scan is load-bearing
   (`N-excluded-orphan-blob` KILLED 2/2); no secret in blob, manifest, log or
   error.
4. **Source race**: changed byte at equal size, changed size, appended
   record, replaced file, removed file, grown-past-bound, chmod 000 (P8),
   retype to a same-content symlink (P9), excluded member size change (P9b)
   all refuse naming the member; excluded equal-size content change seals
   stable (P9c — the stated bound, confirmed); `R1-size-equality-is-proof`
   KILLED 2/2.
5. **Ordering** at the `Capture` entry: `N-race-order-capture` and
   `R2-capture-seal-before-gate` KILLED 2/2 by the store scan. At the
   projection entry: pinned for the receipt (`N-race-order-project`, killed
   by the seal error) — see P2-α row 10 for the seal half.
6. **Crash/idempotency**: the producer's real-SIGKILL child at
   `AfterRawPublish` reproduced 2/2 in the suite and 2/2 under `-count=2`;
   my own SIGKILL at `AfterWalk` (`TestRev2CrashKillAfterWalk`, run twice,
   `logs/crash-rev2.log`): 2 payload blobs, 0 manifests, **0 staged temp
   files, 0 quarantine entries**; the replay installs both manifests
   (`raw_installed=true capture_installed=true`, 4 blobs), every raw entry's
   blob is present, and the replayed bytes are identical to a clean capture.
   `TestProjectIsIdempotent` replay verifies-and-reuses every blob and the
   receipt.
7. **Blob discipline**: `N-payload-install-blob-b` KILLED 2/2 (every raw
   entry's blob present after `Capture` and after the crash replay);
   `N-descriptor-empty/offset`, `R2-first-chunk-oversize` KILLED 2/2; the
   §10.2 example vector reproduced by the predecessor's rows stands.
8. **Admission**: G2, maximal-safe, unknown-profile rows KILLED 2/2; forged
   inputs refuse at the landed decoder (P6a/c/d/e).
9. **Workspace**: 129 fingerprints refuse (`R2-fingerprints-129` KILLED),
   duplicate/unsorted, escapes, sibling-prefix absolute cwd (P18c), empty cwd
   (P18b) refuse; a symlink inside the root pointing outside seals
   `cwd_relative="link"` (P18a, lexically contained, rev1 informational).
10. **Producer harness reproduced**: 34 KILLED + `C-doc-comment` SURVIVED,
    twice, in the isolated copy (`logs/mutants-producer/run{1,2}/`, 35 raw
    logs with `# exit=` each); tree `diff -rq` clean after each run.
11. **Hygiene** (`logs/hygiene*.log`): `gofmt -l` empty over the whole tree;
    `go vet ./...` exit 0; `go build ./...` exit 0; `GOOS=linux`/`GOOS=windows`
    builds exit 0; `tracecheck` exit 0 (bindings=68, clauses 49/569 —
    `internal/traceability` untouched); `cataloggen -adopted … -check` exit 0;
    `git diff --check` clean on the worktree; package `-race` ok (0 DATA
    RACE), cover 87.3%; neighbour suites (clonebundle, localstore, secprim,
    canonicaljson, scalar, environ) ok; whole-repository `go test ./...
    -count=1` 39/39 ok (`logs/repo-test.log` + `repo-test-specpin.log`; the
    specpin package needs the tracked `.task-board` artifact I had dropped
    from the copy and passes with it restored). Changed paths are exactly the
    22 candidate paths; README and LOGBOOK purely additive (0 removed lines),
    README makes no capability/CLI claim; LOGBOOK entry newest-first; the
    producer evidence archive is a real gzip with 106 `MANIFEST.sha256` rows
    all OK, no `__pycache__`, 35 per-plant logs per run with identical run1/
    run2 verdicts; candidate `task-board.config.json` equals HEAD (27
    commands; the CR validation log reports `required=30 green=30` against
    the control root — accepted for the 25-minute race gate, which I did not
    rerun; I ran `-race` on the package).

## Reviewer mutation battery (`reviewer_mutants_rev2.py`, whole committed suite as killer, 2 runs, identical verdicts)

| Row | Kind | Verdict | Killers |
|---|---|---|---|
| R1-blockedAncestor-optional | narrowing | KILLED 2/2 | TestCaptureRefusesOptionalSymlinkedIntermediate |
| R1-archive-emits-receipt | add-write | KILLED 2/2 | TestProjectRefusesUnstable |
| R1-skip-payload-install-blob-b | narrowing | KILLED 2/2 | TestCaptureInstallsEveryPayloadBlob, TestCaptureCrashChildSelfTerminates |
| R1-post-falls-back-to-captured-hash | narrowing | KILLED 2/2 | TestCaptureRaceUnreadableAtPost |
| R1-clonebundle-unknown-ignores-excluded | narrowing (inherited ε) | KILLED 2/2 | TestCaptureOptionalAbsentUnknownMakesRawIncomplete |
| R1-size-equality-is-proof | widening | KILLED 2/2 | 7 tests |
| R1-hook-after-post-measure | seam control | KILLED 2/2 | 12 tests |
| R2-project-seal-before-gate | narrowing (ordering) | **SURVIVED 2/2** | — (P2-α row 10) |
| R2-capture-seal-before-gate | narrowing (ordering, control) | KILLED 2/2 | TestCaptureRaceSealsNoManifest |
| R2-unplanned-special-admitted | narrowing | **SURVIVED 2/2** | — (P3-a) |
| R2-store-root-follows-symlink | widening (root only) | **SURVIVED 2/2** | — (P3-b) |
| R2-excluded-size-ignored | narrowing | **SURVIVED 2/2** | — (P3-c) |
| R2-unstable-seals-pre-as-post | narrowing | **SURVIVED 2/2** | — (P2-α row 8) |
| R2-required-blockedAncestor-dropped | narrowing | KILLED 2/2 | TestCaptureRefusesSymlinkedIntermediate |
| R2-plan-grammar-admits-dotdot | narrowing | KILLED 2/2 | TestCaptureRefusesParentPlanKey |
| R2-fingerprints-129 | narrowing | KILLED 2/2 | TestCheckpointWorkspaceRefusesTooManyFingerprints |
| R2-branch-1025 | narrowing | **SURVIVED 2/2** | — (P3-d, landed decoder is the gate) |
| R2-first-chunk-oversize | narrowing | KILLED 2/2 | TestCaptureMultiChunkMember |
| C-stable-seals-pre-as-post | equivalence control | SURVIVED 2/2 (expected) | — |
| C-rev2-comment | harmless control | SURVIVED 2/2 (expected) | — |

Measured ratio: 17 of 17 AC rows reached by a named test through a production
entry; **15 of 17 closed at the member level** (rows 8 and 10 each carry a
surviving narrowing on a member the row names, executed above); rows 2 and 3
each carry an unmeasured sub-class (symlinked store root; unplanned special
member) graded P3 because the row's stated members are pinned.

## Rework scope (for the producer)

1. Row 8: in `TestCaptureArchiveSealsUnstable` assert the SEALED boundary's
   `pre_capture_digest == result.PreDigest` and `post_capture_digest ==
   result.PostDigest` (decoded `CaptureBoundary.Pre/PostCaptureDigest` or the
   literal strings in the sealed bytes) and that the sealed generation equals
   the request generation; ship a harness row of the
   `R2-unstable-seals-pre-as-post` shape.
2. Row 10: in `TestCaptureRaceEmitsNoReceipt` scan the CAPTURE store for any
   decodable raw or capture manifest (as `TestCaptureRaceSealsNoManifest`
   does) so "before any seal" is pinned at `CaptureAndProject` too; ship a
   harness row of the `R2-project-seal-before-gate` shape (seal first, gate
   second, race error preserved) and note that `N-race-order-project` is
   killed by the seal error message.
3. P3-a: one committed row with an UNPLANNED special member (a symlink to an
   outside file; optionally a FIFO) refusing "not a plan candidate" naming the
   member with no outside bytes installed; harness row of the
   `R2-unplanned-special-admitted` shape.
4. P3-b: `TestCaptureRefusesSymlinkedStoreRoot` (refusal names the link,
   nothing installed); harness row of the `R2-store-root-follows-symlink`
   shape (a narrowing keyed on the link name is fine).
5. P3-c: one race row where the EXCLUDED member's size changes at post
   (refuses naming `store/token-cache`) with a harness row of the
   `R2-excluded-size-ignored` shape — or reword the TRACEABILITY/results
   bound to "presence and shape changes still race".
6. P3-d/e: state in TRACEABILITY that the branch bound's measured gate is the
   landed `DecodeWorkspaceBinding` (local check = defence in depth), and
   either add rows for the plan member-grammar gate, the required-branch
   `blockedAncestor`, the fingerprint count and the chunk size bound, or list
   them as pinned-by-test-without-row.
7. Results, matrix, LOGBOOK: replace "17 of 17 measured at the member level /
   every row's narrowing reddens its killer" with the ratio measured after
   the fixes; keep the finding-by-finding table style of rev2.

Checklist state left by this review: reviewer rows "Implementation matches
AC", "Solution fits project architecture" and "Tests green" are checked (each
measured true); "Gate … attacked, not read — positive-path-only evidence is
not accepted" is unchecked (two AC-row members are pinned by positive-path
evidence only); "If review does not accept the work — verdict evidence added
and status routed by the explicit verdict branches" is checked. Status routed
to `to-dev`.
