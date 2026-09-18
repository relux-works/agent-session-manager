# TASK-260830-24z2b3 — Review verdict, Change Request revision 6

Reviewer run: RUN-260918-3180f5 (claude-opus-5, reviewer/reviewer).
Change Request: `CR-TASK-260830-24z2b3-6` revision 6, base
`a12d1bd5102014790e4c1a36ccc2fcb2422e8f25` (current origin/main, carries the
BUG-260918-354b03 resumesmoke fix), candidate tree
`7a3784136c4f24bea753ea1bd860ccde35fddd87` (24 paths; patch sha256
`40a273688d125702f7f580ff75d13a0711d7be1e3a9296ebae82a8408abf0319`, verified
against the attached patch and against `git diff a12d1bd 7a37841`).
Authority: `internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (lines 10331–10546), §7.8
(3738–3860), §10.2 (4851–4900) and the common logical data model §1.6
(244–337); the pinned sections are byte-identical to v0.6.0 as the producer
states. Revision 5 was never reviewed (its construction failed on the trunk
time bomb); this review covers the whole revision 6, with the rev4 rework
graded first.

## Verdict: ACCEPTED → `accept_cr(TASK-260830-24z2b3, revision=6)`

The P2-δ admit-hole that blocked revision 4 is closed at every Build-side
string admission, not only at the two witnessed entries, and the four P3 items
are fixed. My own attack of the whole revision found no admit-hole, no
spec-fidelity gap, and no unreproduced claim: every probe I drove through a
production entry refused what the pinned text forbids, and the sealed bytes
never carried a substituted U+FFFD. What remains is three P3 measurement gaps
(production correct, one member of a pinned class unmeasured) and a handful of
informational notes, recorded below for the sibling leaves; none of them is a
production defect or a reason to hold the leaf.

Method: isolated immutable copies of the exact candidate tree (`git archive
7a37841…` → `.temp/TASK-260830-24z2b3/review-rev6/{candidate,pristine,probe}`,
`.task-board` artifact dropped; the live worktree's temp-index tree OID was
confirmed equal to `7a37841…` before anything ran and `git status` was
` M LOGBOOK.md`, ` M README.md`, `?? internal/clonebundle/` before and after).
The live Story worktree, index, branch and HEAD were never touched. Every probe
and plant ran through the production entry points; `PYTHONDONTWRITEBYTECODE=1`
for every harness, no `__pycache__`/`.pyc` written; the candidate copy diffed
clean against the pristine copy after every battery. Raw evidence is in
`TASK-260830-24z2b3_review-evidence-rev6.tar.gz` (`logs/`, `logs/mutants/`
one raw `go test` log per plant per run with `# exit=`,
`logs/mutants-producer/run{1,2}/` the shipped harness's per-plant logs,
`reviewer_mutants_rev6.py`, `zz_rev6_probe*_test.go.txt`, the rev3/rev4 probe
files re-executed on this tree).

## 0. Rework verification — every rev4 finding graded

| rev4 finding | Grade | Executed evidence (this run, new tree) |
|---|---|---|
| P2-δ Build entries rewrite invalid UTF-8 to U+FFFD and seal it; `BuildCaptureManifest`/`IdentityDigest` launder a native key past the sanitizer; `native\xff` vs `native\xfe` seal byte-identical | **fixed** | rev4 section K re-executed (`logs/probes-rev4-k-on-rev6.log`): K1–K13 all REFUSED `native key is not valid UTF-8`, K4 both sides refuse, K14 `IdentityDigest(op\xff) == IdentityDigest(op\ufffd)` is now false. rev4 section C re-executed (`logs/probes-rev4-on-rev6.log`): C1–C14, C16, C17 REFUSED (C15 — NUL/BEL in a title — is admitted; §1.6 permits control characters in text, only native keys refuse them). My own class attack (`logs/probes-rev6-text.log`, section T): **43 Build-side string admissions × 2 shapes (`\xff` and the WTF-8 lone surrogate `\xed\xa0\x80`) = 86 probes, 86 REFUSED, `sealed_has_FFFD=false` on every one** — session title, actor name/model/source_native_id, session source_native_session_id, session extension value/key; evidence native_type, reason code [0] and [1], native_session_id, native_event_id, event extension nested slice→map; capture identity native/opaque/kind, item exclusion_reason/key/extension, identity extension, manifest extension, stable and unstable boundary generation, external_source_ref; raw source_native_session_id, entry key, raw extension value/key/array/pointer/interface/struct-field/map[string]string/map[string]int-key/json.RawMessage; `IdentityDigest` native/opaque/kind; `EncodeNativeIdentity` native/opaque/kind/extension; `ParseGeneration`. Section T2: raw-byte Build inputs are refused at the environ frame gate — payload with raw `\xff`, `\ud800`, `\udc00`, the `"\\ud800\udc00"` decoy, WTF-8 bytes, a non-message payload, evidence tuple, workspace (branch and extensions), capture tuple, descriptor media_type; `json.RawMessage` and custom-`MarshalJSON` extension values emitting `\xff` or `\ud800` are refused too (T2-14/15/16), so the stated "custom Marshaler trusted" bound is stricter in practice than stated. Section T3: a legitimate U+FFFD in a title is admitted and sealed (the input carried it), overlong `\xc0\x80` and a code point beyond U+10FFFF refuse, `\xff`/`\xfe` titles both refuse (T3-6). Section T4: Decode refuses raw `\xff`, `\ud800` and the decoy at the frame. Mutation: **13 one-step-away narrowings KILLED 2/2** — per-site exemption of exactly the killer value at title, actor name, actor model, native_type, reason code, exclusion_reason, generation and identity kind (`R6-text-site-*`), walker arms (`R6-text-walk-skip-keys`, `-skip-struct`, `-skip-first-element`) and the encode-time sanitize for both native keys (`R6-encode-skip-opaque-sanitize`, `R6-encode-skip-native-sanitize` — the original hole, one member); producer `N-text-utf8` KILLED 2/2. |
| P3-a″ evidence archive contaminated | **fixed** | `TASK-260830-24z2b3_producer-evidence.tar.gz` (real gzip): 168 files, all matching the expected pattern (27 `cmd*.log`, `pkg-*.log`, 2×67 per-plant logs, two battery logs, `p3b-probe.log`, `MANIFEST.sha256`); `shasum -c MANIFEST.sha256` all OK (167 rows); zero `__pycache__`/`.pyc`; no foreign artifacts. |
| P3-b″ TRACEABILITY over-claims the `-0`/`+`/leading-zero bound | **fixed** | Sentence now states `-0` admits (sealed as `0`), `+1`/`01` refuse at the frame; rev4 probes B3/B8 re-executed on this tree agree (`logs/probes-rev4-on-rev6.log`). |
| P3-c″ build-side reason-code duplicate unmeasured | **fixed** | `TestCanonicalEventRefusals/evidence/reasons_duplicate` exists; `N-reason-codes-dup-build` KILLED 2/2 in the shipped harness; my R10q (duplicate raw_refs) and the reason rows refuse. |
| P3-d″ producer ticked reviewer-owned checklist rows | **fixed as disposed** | The producer executed the literal instruction first and recorded the tool's refusal (`cannot hand off …: unchecked checklist items [15 16 17 18] … handoff evidence missing`), then verified each verdict row and checked it so the Change Request could construct. The handoff gate requires every item; this is the tool's contract, not a producer preference, so the disposition is correct and documented. |

## Findings

No P1. No P2.

### P3 — measurement gaps (production correct; one member of a pinned class unmeasured)

- **P3-ε `hasUnknownClass` is pinned only for the INCLUDED unknown member.** The
  narrowing `if item.Class == "unknown" && item.Included()`
  (`capturebuild.go:418`, `R6-unknown-ignores-excluded`) **SURVIVED the whole
  package suite 2/2**. Under it, driven through the production entries
  (`logs/probe-under-mutant-unknown-excluded.log`): `BuildCaptureManifest`
  seals `raw_complete=true` for a manifest carrying an excluded unknown-class
  item (spec: "Unknown makes raw completeness false"), `DecodeCaptureManifest`
  admits `raw_complete=true` beside that item, and
  `RefuseMaximalSafeUnlessComplete` no longer blocks it — while the committed
  suite (`ok`) stays green, because `TestBuildCaptureManifestUnknownMakesIncomplete`
  uses an included unknown item and the shipped `N-unknown-complete` narrows
  by item count, not by disposition. Production is right (my R1 probe: an
  excluded unknown item seals `raw_complete=false` on the pristine tree). Fix
  for the sibling: one excluded-unknown row driving build derivation, decode
  and `RefuseMaximalSafeUnlessComplete`, plus a harness row.
- **P3-ζ the lone-surrogate member is unpinned at the shared text gate.**
  `R6-text-helper-lone-title` (`validText` admits exactly one WTF-8
  lone-surrogate title) **SURVIVED 2/2**: the only committed lone-surrogate row
  (`capture_source_identity_native_lone_surrogate`) reaches
  `SanitizeNativeKey`'s own `utf8.ValidString`, never `validText`. §1.6 names
  lone surrogates explicitly and the rev4 fix scope named them; production
  refuses them at every site (section T, `lone` rows). Fix: one lone-surrogate
  row through a `validText` site (e.g. the title) plus a harness row.
- **P3-η `EncodeNativeIdentity` standalone admits identity fields decoding
  refuses.** Section T5: a 513-character native session ID or opaque identity,
  kind `bogus`, and a zero `LogicalWorkspaceID` (sealed as `""`) are all
  encoded. Both production callers (`IdentityDigest`, `buildCaptureManifest`)
  re-decode and refuse (T5-2, T5-3), and the doc comment says "renders one
  validated identity", so no production path seals them; but the exported
  entry now advertises "encoding never admits what decoding refused" for
  extensions and native keys while still trusting the caller for bounds,
  vocabulary and the workspace ID. State the pre-validated-identity bound
  explicitly in TRACEABILITY.md (or re-decode inside the entry).

### Informational (no action required unless cheap)

- Per-member narrowings that survive because the class is pinned by another
  member: `R6-sanitizer-c0-tab` (TAB; C0 pinned by U+0000/U+001F/U+007F, and
  `R6-sanitizer-c0-u001f` KILLED), `R6-evidence-partial-operation` (rev4
  informational re-plant; rows use `exact`). Both SURVIVED 2/2.
- The sanitizer admits a single-backslash rooted key (`\rooted`, R13) while
  `\\unc` and `C:\` refuse; `C:x` (drive-relative), `./x`, `%2f` and `x//y`
  admit. The pinned text does not define the sanitizer's scope and the
  package states its bound; Go's own `filepath.IsAbs` treats `\rooted` as
  non-absolute. Worth one line in the stated bound.
- The "custom `json.Marshaler` trusted" bound is over-conservative: marshaled
  extension bytes pass `checkExtensionsClosed` → `environ.DecodeStrictObject`,
  whose frame gate refuses invalid UTF-8 and lone-surrogate escapes (T2-14/15/16).
- Spec-silent behaviours re-observed and unchanged from rev4: control
  characters in a title admit (C15); `-0` seals as `0`; plan keys match as a
  set (D4); an actor whose parent is itself or names no actor admits
  (I23/I24); `raw_refs` order is bytewise over the JCS encoding, so offsets
  `10` then `9` admit (R10s) exactly as §7.8 defines "sorted unique";
  `CheckOrdinalContiguity` checks the set, not the slice order (I16).

## What is clean (verified, not read)

1. **Spec fidelity, member level** — re-extracted every closed shape from the
   pinned text and compared member-for-member with the code: Raw Object
   Manifest (11 members) and `RawObjectEntry` (5; the prose "and extensions"
   is overridden by the closed shape), Capture Manifest (17), Source Basis
   `ax_session` (6) / `external_native` (3), `CaptureItem` (7, nine classes),
   Capture Boundary `stable` (3) / `unstable_archive` (8),
   `StableSnapshotProof` (9, four kinds), `NativeIdentity` (6, three kinds),
   `WorkspaceBinding` (8), Canonical Session (14), `Actor` (7, three kinds),
   Canonical Event (14, 26 kinds, four visibilities), `SourceEvidence` (9,
   four statuses), `RawReference` (4), eight content-block types; bounds
   [1..512]/[1..4096]/[1..1024]/[0..64]/[0..128]/[0..65536]/[1..1000000]/
   [1..1024]/[0..9]/[1..128][0..128]/64 KiB (bytes — R10e/f/g) all match.
   The rev4→rev6 production delta (`git diff 33b52bcb 7a37841`) touches only
   the UTF-8 gate sites (`decode.go` walker, `identity.go` encode sanitize,
   `generation.go`, `session.go`, `event.go`, `captureitem.go`,
   `rawmanifest.go` one guard) — nothing else moved, so the rev3/rev4
   verification of descriptor linking, exclusion and blob install stands and
   was re-executed here.
2. **Refusal census** (`logs/probes-rev6-refusals.log`, 150 probes): unknown
   class derives `raw_complete=false` at build (included and excluded) and
   `raw_complete=true`+unknown refuses at decode; included+reason,
   excluded+content, excluded−reason refuse; all four always-excluded classes
   refuse included at build and decode; unsorted/duplicate items and entries
   refuse at build and decode; `total_bytes` disagreement, descriptor
   byte-count disagreement refuse; stable without proof / `proof:null`,
   pre≠post, closed_store with identity or `input_blocked=false`,
   immutable_snapshot without identity refuse; unstable with `Core=false`
   refuses, `Core=true` seals and `RefuseUnstableForTarget` refuses it; every
   unstable member missing or falsified refuses; `excluded_classes`
   extra/missing refuse; two mains / zero mains / head outside events /
   duplicate events / duplicate actors / subagent without parent / main with
   parent refuse at build and decode; message payload without
   `content_blocks`, block type outside the vocabulary, both/neither content
   arms, 65537-byte inline content refuse (65536 admits); kind/visibility
   outside the vocabularies, duplicate/65 parents, synthesized coupling in
   all three directions, duplicate and unsorted `raw_refs` refuse; ordinals
   `[0,2]`, `[0,0]`, `[1]` refuse; `Generation.Equal("g1","g10")` is false;
   `ParseGeneration` refuses empty and 513 characters and admits 512
   characters / 1024 bytes; the sanitizer refuses `/abs`, `C:\`, `c:/`,
   `\\unc`, `user:pw@host`, schemed credentials, `urn:ax:` in any case, C0/C1
   controls, U+200B/U+2028/U+2029/U+FEFF and a UUIDv7 form; every one of the
   four sealed shapes refuses schema swap, version `1.0.1`, unknown member,
   missing member, trailing data, duplicate member, bad extension key and a
   `1.5` extension value.
3. **Row-21 exclusion** (`logs/probes-rev6-exclusion.log`): derived from the
   source list `hosttrust.ExcludedFromReplication()` (5 paths, each also as
   `…/x` and `…/deep/private-key.pem`) plus every handoff class
   (`.trust-stage-*`, `.custody-stage-*`, `lock-stage-*`, `.bak.<v>`,
   `.pre-rollback.<v>`, `.ax-config-*`, `.ax-config-binding.`), `../escape`,
   `state/../../escape`, `host-channel/./trust.json`, `host-channel//trust.json`
   and an absolute form: **every one refused at all three construction paths**
   (raw entry, excluded capture item, included capture item) and at both
   decode paths. `TestRawManifestExcludesTrustMaterial` /
   `TestCaptureManifestExcludesTrustMaterial` carry a constructor negative per
   class; the package has exactly two member-key construction paths and both
   call `refuseExcludedMember`. The admitted neighbours (`HOST-CHANNEL/…`,
   `host-channel\trust.json` on POSIX, `x/host-channel/…`, bare `lock`,
   `…/credentials-export.json`, `trust.json.bak`) are the hosttrust matcher's
   anchored STATE_DIR-relative contract, outside this leaf, unchanged since
   rev3.
4. **provhost bound** — `TestNoProductionPathAttestsProviderIdentityBinding`
   PASS on the exact tree (`logs/provhost-census.log`: 225 production files, 5
   session-leaf call sites, 0 elsewhere); module grep: non-test
   `VerifyObjectIdentity` callers are only `internal/sessrepo` (5 sites);
   `clonebundle` production calls `CalculateObjectIdentity` ×1 and
   `Canonicalize` ×3; nothing imports `clonebundle`. Accept-any-claim plants
   `R6-verify-claim-any`, `R6-install-claim-any` and the entry-link plant
   `R6-verify-entry-link-any` KILLED 2/2; producer
   `N-descriptor-claim-agreement`, `N-install-site-claim`,
   `N-descriptor-id-link` KILLED 2/2.
5. **Blob install** (`logs/probes-rev6-blob-determinism.log`): exact
   agreement admits; size+1, other blob, other descriptor id, forged self
   claim, a different descriptor of equal size, and a chunk-coverage
   disagreement (owner: "BlobChunk coverage is 15 bytes, want exactly 16")
   refuse; install exact admits, re-install with different bytes refuses
   (no-replace, "declared and staged identity differ"), same bytes admits
   (idempotent), short/long bytes and a forged claim refuse, the empty blob
   installs. The package performs no durable write (no `os`/`WriteFile`
   call in production files); durability is `localstore.PutBlob` with its
   landed `TestPutBlobCrashBoundariesLeaveRecoverableState` and
   no-replace suites, green in the four-package run — the stated purity
   bound is accurate.
6. **Mutation** — shipped harness reproduced in the isolated copy: **66 KILLED
   + `C-doc-comment` SURVIVED, twice**, tree restored, no `__pycache__`
   (`logs/mutants-producer-run{1,2}.log`, 134 per-plant logs with exits).
   My battery `reviewer_mutants_rev6.py` (30 rows × 2 runs, whole-suite
   killer, identical verdicts both runs, `logs/mutants-reviewer-rev6-run{1,2}.log`):
   the brief's classes KILLED — `R6-excluded-set-missing-machine_auth`,
   `R6-raw-subset-admits-machine_auth`, `R6-items-dup-build`,
   `R6-items-dup-decode`, `R6-raw-entries-dup-decode`, `R6-sanitizer-c0-u001f`;
   plus `R6-reconcile-flag-one-way`, `R6-ordinal-dup-zero`,
   `R6-excluded-exact-extra`, the three claim plants and the thirteen text
   plants; `C6-comment-control` SURVIVED; the four survivors are P3-ε, P3-ζ and
   the two informational per-member rows above. Determinism: identical inputs
   seal byte-identical objects for all four shapes, extension insertion order
   and non-canonical/reordered workspace and payload bytes seal identically,
   Build output is JCS-idempotent and its self digest recomputes (D1–D9).
7. **Race gate** — see the next section.
8. **Hygiene** — `gofmt -l` empty; `go vet` clean; `go build ./...` OK;
   `go test ./internal/clonebundle ./internal/provhost ./internal/localstore
   ./internal/canonicaljson -count=1` all ok (`logs/four-packages.log`);
   `go run ./internal/traceability/cmd/tracecheck` exit 0 (bindings=68, clauses 49/569 — `internal/traceability` untouched, `logs/tracecheck.log`); `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` exit 0 on the exact tree (`logs/cataloggen.log`); `git diff --check a12d1bd 7a37841` clean; `GOOS=linux`/`GOOS=windows` builds exit 0 (`logs/cross-builds.log`); package cover 87.1% and package `-race` ok (`logs/pkg-cover.log`, `logs/pkg-race.log`);
   changed paths are exactly the 24 candidate paths (README/LOGBOOK purely
   additive; nothing outside `internal/clonebundle`; `internal/traceability`
   untouched; no `__pycache__`/`.pyc` in the CR); the CR validation log
   reports `required=30 green=30 failed=0 missing=0`. README section has the
   three test commands and no capability/CLI claim. LOGBOOK entry is at the
   top of the newest section. All 31 matrix-named tests exist (33 in the
   package); the 18-of-18 ratio and call sites check out. Sibling bounds
   stated in `doc.go`, TRACEABILITY.md and results.md.

## Race gate (item 7)

`internal/sessquery` is untouched by this change (the candidate touches nothing outside `internal/clonebundle`, README and LOGBOOK; nothing imports `clonebundle`). The rev6 Change Request validation ran the configured suite with the landed `-timeout 25m` and is green (`required=30 green=30`; the producer's cmd05 reports `ok internal/sessquery 390.347s`). My own measurement on the pristine candidate copy (`logs/sessquery-race.log`): `go test ./internal/sessquery -race -count=1 -timeout 25m` → `ok … 315.649s`, wall 317.9 s (user 264.5 s, sys 28.5 s), exit 0, zero `DATA RACE` lines, host load 8.78 → 7.92 with other sessions' `go` and python jobs live the whole time (an idle host was not available; nothing of mine ran concurrently). Four measurements across rounds — rev2 390.6 s, rev3 347.4 s, rev4 417.4 s, rev6 315.6 s — all sit under the old 600 s default and well under the 25 m gate; the producer's earlier 665 s was under the full concurrent suite. Disposition unchanged: host-capacity artifact, not a candidate defect; the 25 m timeout on trunk (`888ae3d`) is the orchestration fix, the gate is green in this revision, and the producer's disclosure was the right disposition.

## Checklist and routing

Live merged checklist: every row verified true by this review and left
checked. `accept_cr(TASK-260830-24z2b3, revision=6,
evidence=TASK-260830-24z2b3_review-verdict-rev6.md)` routes the element to
`integrating`; the three P3 rows above are recorded for the sibling leaves
(`TASK-260830-2g5be6`, `TASK-260830-32ypr2`) and do not hold this leaf.
