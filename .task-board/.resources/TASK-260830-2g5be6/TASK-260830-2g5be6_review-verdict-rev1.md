# TASK-260830-2g5be6 — Review verdict, Change Request revision 1

Reviewer run: RUN-260918-cee501 (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260830-2g5be6-1` revision 1, base
`2f844bb49702a860c1199a6a2a0ca6a5cc878197` (the predecessor's checkpoint on the
Story branch), candidate tree `31c0f23c3424e3fce3b23f736022cf91d1783122` (22
paths; patch sha256 `d52c146e9c715d048d6cc86595166b8d065dc1baff6e83683d2a6b6a1745397f`,
verified equal to `git diff 2f844bb 31c0f23`). The live Story worktree's
temp-index tree OID equalled `31c0f23c…` before and after this review and
`git status` stayed ` M LOGBOOK.md`, ` M README.md`, `?? internal/clonesnap/`;
the worktree, index, branch and HEAD were never touched.
Authority: `internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (lines 10331–10546),
§10.2 (4851–4900), §7.8 (3738–3925), re-extracted and diffed member-for-member
against the candidate. (The reviewer brief's "Section 4.C / terminalbackend"
paragraph is a copy from another leaf; this review covers the capture scope the
task and producer brief define.)

## Verdict: CHANGES REQUESTED → `to-dev`

Production is correct on every probe I drove: I found no admit-hole, no
spec-fidelity gap, no fork of a landed gate that changes behaviour, and no
containment escape. What blocks acceptance is the evidence contract, in exactly
the shape the producer brief warned about: five of the seventeen AC rows claim
closure through named tests that do not reach the named member (a narrowing
survives the whole committed suite 2/2 while the row's tests stay green), one
of them a §10.2 MUST whose three cited tests never look at a payload blob; and
the three inherited P3 advisories the brief required the producer to close or
explicitly defer are neither closed nor mentioned — the in-scope one (P3-ε)
opens the maximal-safe hole at this leaf's own projection entry under the
predecessor's surviving mutant. The rework is small and precise (section
"Rework scope").

Method: isolated immutable copies of the exact candidate tree (`git archive
31c0f23c…` → `.temp/TASK-260830-2g5be6/review-rev1/{candidate,pristine,probe,
probe2,mut}`, `.task-board` artifact dropped, `diff -r` clean against
`pristine` before use and after every battery). Every probe and plant drove a
production entry (`Capture`, `CaptureAndProject`, `AdmitForTarget`,
`CheckpointWorkspace`, `BuildBlobDescriptor`); `PYTHONDONTWRITEBYTECODE=1` for
every harness, zero `__pycache__`/`.pyc` in any copy. Raw evidence:
`TASK-260830-2g5be6_review-evidence-rev1.tar.gz` (`logs/`, one raw `go test`
log per plant per run under `logs/mutants/run{1,2}/` and
`logs/mutants-producer/run{1,2}/`, `reviewer_mutants_rev1.py`,
`zz_rev1_probe_test.go.txt`, `probes-rev1.log` with 117 `PROBE` lines (plus `probe-c12-pristine.log`), the
`probe-under-mutant-*.log` executions of each survivor).

## 0. Inherited advisories (predecessor verdict rev6, RUN-260918-3180f5)

The producer brief: "Read them … close the ones your scope touches, and say in
your results which you closed and which you deliberately left to the final
leaf." `TASK-260830-2g5be6_results.md`, the conformance matrix, TRACEABILITY.md
and the LOGBOOK entry contain no mention of P3-ε, P3-ζ or P3-η (grep for
`P3`, `advisor`, `inherit`, `lone`, `surrogate`, `EncodeNativeIdentity`,
`excluded unknown`: zero hits).

| Item | In this leaf's scope? | Grade |
|---|---|---|
| P3-ε `hasUnknownClass` pinned only for the INCLUDED unknown member | **Yes** — AC row 5 "unknown drives raw_complete=false and blocks maximal_safe through the projection entry"; an optional absent `unknown` plan member seals an EXCLUDED unknown item here (`plan_optional_absent`) | **Silently dropped, still open.** `R1-clonebundle-unknown-ignores-excluded` (the predecessor's exact narrowing at `capturebuild.go:418`) **SURVIVED the whole clonesnap suite 2/2**. Executed under it (`logs/probe-under-mutant-epsilon.log`): `Capture` seals `raw_complete=true` beside an excluded unknown item and **`AdmitForTarget(…, "maximal_safe")` returns nil** — the hole opens at this leaf's projection entry — while `TestCaptureUnknownMakesRawIncomplete`, `TestProjectMaximalSafeRequiresComplete`, `TestProjectMaximalSafeAdmitsComplete`, `TestProjectStrictExactAdmitsUnknown` all PASS. Pristine is correct (probe B2: excluded unknown → `raw_complete=false`, `excluded_classes=[credential unknown]`, maximal_safe refused "blocked by an unknown-class item", strict_exact admits). |
| P3-ζ lone surrogate unpinned at `validText` | No (titles/sessions belong to the normalization leaf) | Silently dropped — must be stated as deferred. |
| P3-η `EncodeNativeIdentity` standalone admits what decoding refuses | Partially — this leaf is a caller of `IdentityDigest`/`BuildCaptureManifest`, which re-decode | Silently dropped. Production here is safe: probe F1 drove a 513-character native ID, kind `bogus`, a zero workspace ID, `native\xff` and `/abs/native` through `Capture`; every one refused (`native identity digest seals no undecodable identity: …`). The `CaptureRequest.SourceIdentity` "is the sanitized NativeIdentity" doc line should state that the entry re-validates it (it does), and the results must say η's TRACEABILITY line is the final leaf's or clonebundle's. |

## Findings

No P1.

### P2 — evidence contract (claimed evidence does not reach the claim)

- **P2-α Inflated AC ratio: "17 of 17 rows driven" at the member level is
  12 of 17.** Five rows name tests that stay green while a narrowing admits
  the row's own member; each plant SURVIVED the whole committed suite in two
  independent runs and its admitted behaviour was executed under the plant:
  - **Row 14 (§10.2 MUST "before publishing a record that references a blob …
    atomically install it")** — `R1-skip-payload-install-blob-b` (exactly
    `store/blob-b` sealed into the raw manifest, its blob never installed)
    SURVIVED 2/2. Under it (`logs/probe-under-mutant-payload-skip.log`) the
    three tests the matrix cites for the row — `TestCaptureSealsBothManifestsFromStoreBytes`,
    `TestCaptureCrashChildSelfTerminates`, `TestProjectIsIdempotent` — all
    PASS while the store holds 3 blobs (blob-a + two manifests) and the
    published raw manifest references a blob-b that is not there. None of the
    cited tests inspects a payload blob; only the manifest `Installed` flags
    and blob counts are asserted. This is the whole payload half of the MUST,
    not one member of it.
  - **Row 8 "unstable_archive … never enters a target branch (projection
    entry)"** — `R1-archive-emits-receipt` (the archive branch of
    `CaptureAndProject` writes the archive capture manifest into the TARGET
    sink before the G2 refusal) SURVIVED 2/2. Under it `TestProjectRefusesUnstable`,
    `TestProjectRefusesUnstableDirect` and `TestCaptureRaceEmitsNoReceipt`
    PASS while `target_blobs=1` (`logs/probe-under-mutant-archive-emits.log`).
    The G2 refusal is pinned at the error only; the target sink is inspected
    only under `RaceRefuse`. Pristine: probe C11 `target_blobs=0`.
  - **Row 2 containment "symlinked intermediate … refuse"** —
    `R1-blockedAncestor-optional` (drop the `blockedAncestor` guard from the
    optional-absent branch) SURVIVED 2/2: an OPTIONAL plan member whose
    intermediate is a symlink to an outside directory seals as
    `plan_optional_absent` and the capture succeeds (`err=<nil> result=true`,
    `logs/probe-under-mutant-blockedAncestor.log`) while
    `TestCaptureRefusesSymlinkedIntermediate/TrailingSymlink/FIFO` and
    `TestCaptureOptionalAbsentSealsExcluded` PASS. Every committed
    symlink/FIFO row is `Required: true`. Pristine refuses (probes A3, A4:
    `member symlink escape: link-escape/file`, `target special file`).
  - **Row 9 race "replaced file"** — `R1-post-falls-back-to-captured-hash`
    (a member the post measurement cannot re-read falls back to the captured
    hash) SURVIVED 2/2 with all seven `TestCaptureRace*` rows green. The
    admitted member is concrete: a member that grows PAST `MaxSingleBytes`
    between the walk and the post measurement, and a member whose permission
    is lost, both **seal `stable`** under the plant (`logs/probe-c12-under-mutant-post-fallback.log`:
    `err=<nil> stable=true` twice) while pristine refuses both
    (`logs/probe-c12-pristine.log`). The committed "replaced file" row
    replaces with a readable regular file; retype-to-symlink is caught by the
    enumeration shape, so the unreadable-regular-file class has no row.
  - **Row 5 unknown** — the excluded-unknown member (P3-ε above).
- **P2-β Inherited advisories neither closed nor deferred** (section 0). The
  in-scope one is part of P2-α; the other two must be named as deferred in the
  results so the final leaf inherits them on the record, not by accident.

### P3 — prose that outruns the tests; rows that measure the wrong thing

- **P3-γ A false justification stated in four places** (doc comment on
  `sealAndPublish`, TRACEABILITY.md "Stated bounds", LOGBOOK "ORDERING",
  results "Bounds"): "sealing the raw post measurement instead would fail
  closed under a reorder and leave the ordering unpinned". Executed
  (`logs/claim-post-sealed-under-reorder.log`): with `PostCaptureDigest:
  state.post.String()` AND the producer's own `N-race-order-capture` reorder
  applied together, `TestCaptureRaceSealsNoManifest` still FAILS ("mutated
  capture sealed a raw manifest before the race gate") — the raw manifest is
  published before the boundary is built, so the ordering stays pinned either
  way. The equivalence control `R1-stable-seals-post-as-pre` SURVIVED 2/2 as
  it must (under the gate pre == post; probe C9 sealed post == measured post).
  Sealing the pre value for `post_capture_digest` is not a defect, but the
  reason given for it is false; seal the measured post digest (the member the
  spec names) or reword the bound to what is true.
- **P3-δ `N-workspace-cwd` measures the message, not admission.** Under the
  row `/outside-workspace` is still refused by `scalar.ParseRelativePath`
  ("must not contain empty, dot, or parent segments"); the killer reddens on
  the missing literal "escapes the workspace root"
  (`logs/mutants-producer/run1/N-workspace-cwd.log`). The line-120 escape check
  is subsumed by the landed relative-path grammar on POSIX; state it as defence
  in depth or make the row assert admission.
- **P3-ε′ `N-exclude-*` rows are killed by the landed builder, not by the leak
  scan** (`raw object entry[0] class "credential" is outside the raw subset`);
  under those plants the excluded bytes WERE opened and installed before the
  seal refused. The scan is load-bearing — my `R1-excluded-bytes-into-blob`
  (orphan blob, no manifest entry, no log line) is KILLED 2/2 by
  `TestCaptureExcludesSecrets` — but the harness should carry that row so the
  claim "bytes never reach a blob" is measured by the scan, not inferred from a
  builder refusal.
- **P3-ζ′ Second spelling of the four-profile fidelity vocabulary.**
  `internal/sessadapter/operations.go:535` already holds the §7.8
  `fidelityProfiles` table (unexported); `clonesnap.validFidelityProfile` is a
  second copy. Not a behavioural fork, but the final leaf will need the same
  vocabulary — export one owner.

### Informational (no action required unless cheap)

- `CheckpointWorkspace` admits a relative `WorkspaceRoot` (probe E5,
  resolves against the process cwd; `secprim.NewGuard` requires absolute for
  the store root) and records `cwd_relative="link"` for a symlink inside the
  root that points outside (probe E1) — lexically contained, no authority
  granted; worth a stated line.
- Seal-time refusals (identity, basis, proof coupling, generation) run after
  the walk, so payload blobs are already installed as content-addressed
  orphans when they refuse (probe F1: `blobs=2`). No manifest is published;
  fail-fast validation of request scalars before the walk would be cheaper.
- `NativeSessionID` and `SourceIdentity.NativeSessionID` may disagree (probe
  F6) — spec-silent.
- The admission receipt `urn:ax:internal:clonesnap-admission-receipt` is a
  package-internal artifact carried only to pin ordering; it is documented as
  such. The final leaf should replace it with the real projection output or
  keep the bound explicit.
- Enumeration walks path strings (`os.ReadDir`/`os.Lstat`) while every open is
  openat-relative through the Guard; TRACEABILITY states this bound. A
  directory→symlink swap between `Lstat` and `ReadDir` could list outside
  NAMES into an unplanned-member refusal; no byte can follow (Guard root
  binding refuses, probe A8/A1).

## What is clean (verified, not read)

1. **Spec fidelity** (`logs/probes-rev1.log`, 117 probe lines): Raw Object
   Manifest entries from real bytes, sorted, sanitized, blob/descriptor IDs
   recomputed (A9: 11 plan-key shapes refuse — `./`, `//`, trailing `/`, `.`,
   `..`, `store/..`, backslash, NUL, `\n`, `%2e%2e`); `excluded_classes`
   exactly the excluded-row classes including a `plan_optional_absent`
   durable class (B1); one item per plan candidate; always-excluded classes
   never opened — 3 blobs total for a five-member store (B4), an excluded
   member that is a symlink to an outside secret is never followed (A6);
   every stable-proof kind with its coupling rule, 13 combinations (F2);
   generation and external-ref bounds 512/513 in characters (F3: 512
   two-byte characters admit; F4); timestamp/UUIDv7 stamps refuse (F5); the
   §10.2 example reproduces byte-exactly — `ax-example\n` seals
   `descriptor_id sha256:390c8f21…` and `blob_id sha256:9c21bad6…` (D3);
   chunking at 4194303/4194304/4194305/8388608/8388609 → 1/1/2/2/3 chunks
   (D2); per-object bound at 19/18 bytes both directions (D4); manifests
   addressed by the SHA-256 of their sealed bytes (D5).
2. **Containment** at depth 3 with the symlink in the MIDDLE component
   (A1: `member symlink escape: a/link/file`, no outside byte in the store),
   FIFO at depth 2 named as `target special file` never "open failed" and
   never stalling (A2), optional FIFO (A3), inside-store symlink (A5),
   unplanned symlink (A7), symlinked store root (A8), dangling intermediate
   and trailing symlinks (A10/A11). Producer rows `N-contain-*`, `N-fifo`
   KILLED 2/2.
3. **Exclusion**: four secret sequences reach no blob, manifest, log line or
   error string on pristine; `R1-excluded-bytes-into-blob` KILLED 2/2 proves
   the blob scan is load-bearing; refusal strings on the containment path
   carry no outside bytes (B6).
4. **Source race**: true `O_APPEND` append (C1), equal-size symlink swap
   (C2), excluded member size change (C3), new directory (C5) all refuse
   naming the member; equal-size content change of an EXCLUDED member and an
   mtime-only touch seal stable (C4/C6 — the stated bound, confirmed);
   archive policy seals `unstable_archive` with the two MEASURED digests and
   the raw manifest carrying the pre-mutation bytes (C7); archive policy on a
   quiet store seals stable (C8); a race through `CaptureAndProject` leaves
   both sinks without manifests or receipts (C10); the archive form through
   the projection entry publishes to the capture store, target sink empty,
   G2 refusal (C11). `R1-size-equality-is-proof` (size-only comparison)
   KILLED 2/2 by seven tests; `R1-archive-seals-stable`, `R1-unstable-not-core`
   KILLED 2/2; `R1-hook-after-post-measure` (seam control) KILLED 2/2.
5. **Ordering**: producer rows `N-race-order-capture` / `N-race-order-project`
   KILLED 2/2; the reorder is killed with either digest sealed (P3-γ).
6. **Crash/idempotency**: the producer's real-SIGKILL child reproduced 2/2
   (`TestCaptureCrashChildSelfTerminates` in both harness runs and the
   suite); my own SIGKILL at `AfterWalk` (`TestRev1ProbeCrashAfterWalk`,
   H1): 2 payload blobs, 0 manifests, 0 staged temp files left; the replay
   publishes both manifests (`raw_installed=true capture_installed=true`),
   changes no payload blob, and is byte-identical to a clean capture.
   `R1-publish-under-manifest-id` KILLED 2/2 (the store verifies the digest of
   what it installs).
7. **Blob discipline** on pristine: both payload blobs installed under their
   content digests (D1); `R1-descriptor-media-uppercase` KILLED 2/2 (landed
   closed-shape validator in the path); `R1-oversize-boundary-off-by-one`
   KILLED 2/2.
8. **Admission**: raw-as-capture, trailing data, empty, truncated refuse at
   the decoder through `AdmitForTarget` (G1); a forged `raw_complete:true`
   beside an unknown item refuses (G2); the receipt carries identities only
   (G3); nil target sink refuses before any capture (G4). `N-g2`, `N-maximal`,
   `N-fidelity` KILLED 2/2.
9. **Workspace**: 128 fingerprints admit, 129 refuse; branch `""` refuses, 1
   char admits, 1024/1025 characters pinned by the producer; `sha1:` and
   uppercase digests refuse (E3/E4); `./work`, `work/`, `work//trees`,
   `work/trees/alpha/.`, `..`, NUL and `/` refuse, `.` admits (E2); a bad
   workspace refuses before any install (E6).
10. **Producer harness reproduced**: 28 KILLED + `C-doc-comment` SURVIVED,
    twice, in the isolated copy; tree `diff -r` clean after each run; 29 raw
    per-plant logs with `# exit=` per run.
11. **Hygiene**: `gofmt -l` empty; `go vet` clean; `go build ./...`;
    `GOOS=linux`/`GOOS=windows` builds exit 0; `tracecheck` exit 0
    (bindings=68, clauses 49/569 — `internal/traceability` untouched);
    `cataloggen -adopted … -check` exit 0; `git diff --check` clean; package
    `-race` ok (0 DATA RACE), cover 86.9%; neighbour suites (clonebundle,
    localstore, secprim, canonicaljson, scalar, environ) ok. Changed paths are
    exactly the 22 candidate paths; README additive with no capability/CLI
    claim; LOGBOOK entry newest-first; no `__pycache__`/`.pyc`. The CR
    validation log reports `required=30 green=30` (the candidate config
    carries 27 commands; the control root 30); the producer evidence archive
    is a real gzip with 93 `MANIFEST.sha256` rows all OK, zero foreign
    artifacts, cmd04 39 packages ok, race groups 0 DATA RACE
    (`sessquery 379.6s`). I did not rerun the 25-minute whole-repository race
    gate; I accept it from the CR validation log and ran `-race` on the
    package.

## Reviewer mutation battery (`reviewer_mutants_rev1.py`, whole committed suite as killer, 2 runs, identical verdicts)

| Row | Kind | Verdict | Killers |
|---|---|---|---|
| R1-blockedAncestor-optional | narrowing | **SURVIVED 2/2** | — (P2-α row 2) |
| R1-excluded-bytes-into-blob | add-write | KILLED 2/2 | TestCaptureExcludesSecrets |
| R1-size-equality-is-proof | widening | KILLED 2/2 | 7 tests |
| R1-archive-seals-stable | narrowing | KILLED 2/2 | 3 tests |
| R1-archive-emits-receipt | add-write | **SURVIVED 2/2** | — (P2-α row 8) |
| R1-skip-payload-install-blob-b | narrowing | **SURVIVED 2/2** | — (P2-α row 14) |
| R1-post-falls-back-to-captured-hash | narrowing | **SURVIVED 2/2** | — (P2-α row 9) |
| R1-hook-after-post-measure | seam control | KILLED 2/2 | 11 tests |
| R1-descriptor-media-uppercase | narrowing | KILLED 2/2 | 40 tests |
| R1-unstable-not-core | narrowing | KILLED 2/2 | 3 tests |
| R1-publish-under-manifest-id | narrowing | KILLED 2/2 | 29 tests |
| R1-oversize-boundary-off-by-one | narrowing | KILLED 2/2 | TestCaptureRefusesOversizedMember |
| R1-stable-seals-post-as-pre | equivalence control | SURVIVED 2/2 (expected) | — |
| R1-clonebundle-unknown-ignores-excluded | narrowing (inherited ε) | **SURVIVED 2/2** | — (P2-α row 5) |
| C-rev1-comment | harmless control | SURVIVED 2/2 (expected) | — |

Measured ratio: 17 of 17 AC rows reached by a named test through a production
entry; **12 of 17 closed at the member level** (rows 2, 5, 8, 9, 14 each carry
a surviving narrowing whose admitted behaviour was executed above).

## Rework scope (for the producer)

1. Row 14: a committed test that asserts every raw-manifest entry's blob is
   present in the store after `Capture` (and after the crash replay), plus a
   harness row of the `R1-skip-payload-install-blob-b` shape.
2. Row 8: assert the TARGET sink is empty on the archive path of
   `CaptureAndProject` (`TestProjectRefusesUnstable`), plus a harness row of
   the `R1-archive-emits-receipt` shape.
3. Row 2: one `Required: false` member behind a symlinked intermediate (and
   one optional FIFO) through `Capture`, plus a harness row narrowing the
   optional-absent branch.
4. Row 9: one race row where a member becomes unreadable at post time without
   changing shape (grown past `MaxSingleBytes`, or `chmod 000`), plus a
   harness row of the `R1-post-falls-back-to-captured-hash` shape.
5. Row 5 / P3-ε: one optional absent `unknown` plan member driving
   `raw_complete=false` and `AdmitForTarget(maximal_safe)` refusal, plus a
   harness row of the `R1-clonebundle-unknown-ignores-excluded` shape (the
   plant lives in clonebundle; the killer lives here — say so).
6. Results: a section naming P3-ε closed and P3-ζ / P3-η deferred to the final
   leaf (or closed), and the measured ratio.
7. P3-γ: seal `state.post` for `post_capture_digest` or reword the bound in
   the four places; P3-δ: reword the `N-workspace-cwd` note (defence in
   depth, message-keyed) or assert admission; P3-ε′: add the orphan-blob leak
   row to the shipped harness; P3-ζ′: note the duplicated fidelity vocabulary
   as a bound for the final leaf (or export one owner).

Checklist state left by this review: the four reviewer rows (Implementation
matches AC; Solution fits project architecture; Tests green; Gate … attacked,
not read) are unchecked; "If review does not accept the work — verdict evidence
added and status routed by the explicit verdict branches" is checked. Status
routed to `to-dev`.
