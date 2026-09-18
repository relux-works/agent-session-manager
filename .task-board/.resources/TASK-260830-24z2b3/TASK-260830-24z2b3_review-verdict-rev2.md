# TASK-260830-24z2b3 — Review verdict, Change Request revision 2

Reviewer run: RUN-260917-c05bc8 (claude-opus-5, reviewer/reviewer).
Change Request: `CR-TASK-260830-24z2b3-2` revision 2, base
`888ae3dff6a55c29bfaf311957748d405ed1ab06`, candidate tree
`bee4c80724cee011a1bdde7a59b0479637046769` (21 paths; patch sha256
`c11360377daab6b6b2cfa34c1570cf452e1df9b411f55e55586505e8b7608f67`).
Authority: `internal/specdoc/SPEC.v0.6.0.md` §13.14.1, §7.8, §10.2 and the
common logical data model (§1.6, lines 286–337).

## Verdict: CHANGES REQUESTED → `to-dev`

Two P1 correctness holes in gates the leaf claims to own, four P2 fidelity /
evidence gaps, and P3 hygiene. The member-level spec fidelity, the row-21
exclusion gate, the provhost bound, the blob install path and the race gate
are all clean — the rework is narrow (see "Rework scope"). Nothing here is a
Stop-The-Line.

Method: isolated immutable copy of the exact candidate tree
(`git archive bee4c807…` → `.temp/TASK-260830-24z2b3/candidate`, `.task-board`
artifact dropped); the live Story worktree, index, branch and HEAD were never
touched (`git status` unchanged: ` M LOGBOOK.md`, ` M README.md`,
`?? internal/clonebundle/`; the temp-index tree of the live worktree equals
`bee4c807…`). Every plant and probe below ran through the production entry
points; raw logs are in `TASK-260830-24z2b3_review-evidence-rev2.tar.gz`
(`logs/`, `logs/mutants/*.log` one raw `go test` log per plant per run with
the subprocess exit code, `reviewer_mutants.py`, `refusal_census.py`,
`zz_review_probe_test.go.txt`, `zz_review_probe2_test.go.txt`).

## Findings

### P1-a — Descriptor identity is never linked to the entry (`rawmanifest.go`, `blobref.go`)

`VerifyDescriptorAgreement(descriptor, wantBlobID, wantSize)` checks the
descriptor's own claim against its omit-self digest and its `blob_id`/`size`
against the entry, but nothing ever compares the entry's
`blob_descriptor_id` with the identity of the descriptor bytes that were
verified. Consequences, all witnessed through production entries
(`logs/probes-01.log`):

- `BuildRawObjectManifest` with `inputs[0].DescriptorID =
  sha256("some-other-descriptor")` and the real descriptor bytes: **ADMITTED**
  — a sealed manifest whose entry references a descriptor that does not
  exist (probe P1).
- `DecodeRawObjectManifest` of that manifest: ADMITTED (P1b — expected,
  structure only).
- `VerifyRawManifestDescriptors` with a fetch that returns the real
  descriptor bytes for the wrong ID: **ADMITTED** (P1c) — the "evidence"
  entry accepts any descriptor whose blob ID and size agree, regardless of
  whether it is the descriptor the entry names.
- Mutant `R-descriptor-id-unlinked2` (after agreement passes, the sealed
  `blob_descriptor_id` is replaced by `sha256(input.DescriptorID)`, an
  unrelated digest): **SURVIVED** the whole suite 3/3.

§13.14.1: "Each entry has … blob ID, Blob Descriptor ID … The descriptor
agrees with blob ID and byte count" — the descriptor that must agree is the
one the entry identifies. Fix: `VerifyDescriptorAgreement` (or the two
callers) must also require `calculated == entry.BlobDescriptorID`; add the
negative (`DescriptorID` ≠ descriptor identity) at build, at
`VerifyRawManifestDescriptors`, and a narrowing mutant for the new arm.

### P1-b — The forked omit-self identity path lacks the AX number model (`decode.go` `omitSelfDigest`/`canonicalizeObject`)

Extension values are the only member class whose numbers are not
uint53-validated by the shape decoders, and they flow into the identity
through the package's own `omitSelfDigest`, which uses lenient
`encoding/json` decoding plus `canonicaljson.Canonicalize` — the exact fork
the owner warns about ("a caller that needs the AX number model must reach
it through one of the entry points named above rather than by canonicalizing
first", `canonical.go:214-221`). Witnessed (`logs/probes-01.log`):

- `BuildRawObjectManifest(... extensions={"com.example.f":1.5,
  "com.example.big": 1<<60})`: **ADMITTED**; the sealed bytes carry
  `"com.example.big":1152921504606847000` — 2^60 = 1152921504606846976 was
  **rounded and the build continued** (P2c/P2e), and `DecodeRawObjectManifest`
  admits the result (P2d).
- `omitSelfDigest` over `1.5`, `9007199254740993`, `1e2`, `-0`: ADMITTED
  (P2); only `1E400` fails, and only because `strconv.ParseFloat` overflows.

Common logical data model (SPEC.v0.6.0.md:305-310): "forbids floating-point
values … A decoder MUST reject a numeric literal at or beyond 2^53 …
Implementations MUST NOT round a value and continue." The same fork also
collapses nested duplicate members inside extension values (lenient decode;
`canonicaljson.decodeValue` refuses them at every depth): a manifest carrying
`"extensions":{"com.example.x":{"a":1,"a":2}}`, resealed over the collapsed
form, is **ADMITTED** by `DecodeRawObjectManifest` (probes D1/D2,
`logs/probes-03-nested-duplicate.log`). Fix: validate extension values recursively at Build and
Decode (integer literals only, |n| ≤ 2^53−1, no fraction/exponent, nested
objects duplicate-free), or route the omit-self through an owner entry that
carries the guarantee; add negatives and a narrowing mutant (admit exactly
`2^53`).

### P2-a — String bounds measured in bytes, forking the environ measure and contradicting the spec (`decode.go:111-127`, every `len(...)` bound in the Build paths)

SPEC.v0.6.0.md:337: "`string[n..m]` bounds UTF-8 characters".
`environ.StringLength` counts runes ("Section 1.6 string measure in
characters, not bytes"; sessadapter and dirnode delegate to it so "the two can
never drift"). `clonebundle.stringLength` returns `len(value)` and its
comment asserts the opposite convention. Witness (`logs/probes-02-string-measure.log`):
a 300-character / 600-byte native key is refused by `BuildRawObjectManifest`
("not a string[1..512]") and a 3000-character title by both
`BuildCanonicalSession` and `DecodeCanonicalSession`, while
`environ.CheckStringBounds(300 chars, 1..512)` admits. Over-strict only (bytes
≥ characters), so no admit hole, but a conformant peer's manifest is refused
here and the leaf forks the one repository measure. Fix: delegate to
`environ.CheckStringBounds`/`StringLength` (and rune-count the Build-side
`len()` checks), fix the comment, add one multibyte boundary row.

### P2-b — Decode-side gates are unmeasured: 10 narrowing plants (plus one informational) survive the whole suite (3/3 each)

The refusal census (`logs/refusal-census.log`, coverage-driven over every
`invalid(` site) counts 359 refusal sites, 156 covered, 203 uncovered. Most
rules are pinned once on the Build side and duplicated as separate code on
the Decode side, where no named test reaches them. Plants against the exact
candidate tree, `-run Test` (whole suite), deterministic over three runs
(`logs/mutants-reviewer-run1.log`, `logs/mutants-repeats-run2-3.log`):

| Plant | Site | Weakening | Verdict |
|---|---|---|---|
| R-capture-items-dup-decode | `capturebuild.go:345` | `<=`→`<` (equal-key duplicate item) | SURVIVED 3/3 |
| R-sorted-digests-dup-decode | `decode.go:259` | `<=`→`<` (duplicate digest; heads/parents/fingerprints) | SURVIVED 3/3 |
| R-sorted-strings-dup-decode | `decode.go:233` | `<=`→`<` (duplicate string; excluded_classes/reason_codes) | SURVIVED 3/3 |
| R-rawrefs-dup-decode | `event.go:196` | `<= 0`→`< 0` (duplicate raw_ref) | SURVIVED 3/3 |
| R-decode-item-class | `captureitem.go:109` | admit exactly class `mystery` | SURVIVED 3/3 |
| R-decode-actor-main-parent | `session.go:71` | admit exactly actor[0] main-with-parent | SURVIVED 3/3 |
| R-decode-evidence-status | `event.go:279` | admit exactly `capture_status=maybe` | SURVIVED 3/3 |
| R-decode-event-visibility | `event.go:547` | admit exactly `visibility=secret` | SURVIVED 3/3 |
| R-verify-descriptors-skip-0 | `rawmanifest.go:419` | skip disagreement at entry 0 | SURVIVED 3/3 |
| R-install-site-claim | `blobref.go:72` | install-site claim comparison disabled | SURVIVED 3/3 |
| R-sanitizer-admits-0x01 | `identity.go:292` | admit exactly U+0001 | SURVIVED 3/3 (informational; the class is pinned at \x00, \x1f, \x7f — R-sanitizer-admits-0x1f KILLED) |

Every one of these arms does refuse when reached (probes P10–P12 all
REFUSED), so the code is right and the evidence is missing. The DoD row
"every gate ships at least one NARROWING mutant" is not met for the Decode
entries the conformance matrix names as call sites (rows 6, 7, 12, 13, 16,
17, 18). Decode-side rules that ARE pinned (my plants KILLED 2/2): raw entry
class and total_bytes and order at decode, session head-subset at decode,
capture-manifest `raw_complete`+unknown at decode, both exclusion arms by
true narrowing (`logs/mutants-extra.log`).

### P2-c — Positive-only evidence on two verifying entries; one mutant row over-claims

- `VerifyRawManifestDescriptors` has no negative test (matrix row 18 cites
  only `TestBuildRawManifestRoundTrip`); its three refusal arms
  (`rawmanifest.go:414/417/420`) are uncovered and `R-verify-descriptors-skip-0`
  survives. Probes P15/P15b/P15c show the arms work.
- `InstallRawBlob`'s own claim comparison (`blobref.go:72`) and its
  `blob_id`/`size` arms (77/81) are uncovered; `R-install-site-claim`
  (`&& false` at the install site only) survives. The shipped
  `N-descriptor-claim` row patches both sites (`count=2`) and its note says
  "at both agreement and install sites", but the kill comes only from the
  `VerifyDescriptorAgreement` site (`TestBlobAgreementRefusals` resealed
  descriptor). One measured site is being reported as two.

### P2-d — `deriveRawComplete` admits duplicate raw keys hiding a missing included object (`capturebuild.go:420-427`)

`deriveRawComplete(items{a,b included}, planKeys=[a,b], rawKeys=[a,a])`
returns **true** (probe P4; `false` wanted): the "all and only included
objects" check compares `len(included)` with `len(rawKeys)` and then only
tests membership of each raw key, so a duplicate defeats the "all" half.
`planKeys` duplicates are refused (P4b), so the two halves are asymmetric.
`VerifyCaptureReconciliation` is a public verification entry taking
`[]string`; it must refuse duplicate raw keys (or take the decoded
`RawObjectManifest`). Add the negative and a narrowing mutant.

### P2-e — `IdentityDigest`/`EncodeNativeIdentity` seal identities that `DecodeNativeIdentity` refuses (`identity.go:127`, `rawmanifest.go:428`)

`IdentityDigest(NativeIdentity{IdentityKind:"bogus",…}, nil)` and
`IdentityDigest(valid, {"no-dots":1})` both return a digest (probes P9/P9b).
The doc comment claims "encoding never admits what decoding refused"; the
matrix (rows 4 and 15) names `IdentityDigest` as the `source_identity_digest`
production entry, and §13.14.1 defines that member as the digest of
*canonical sanitized* `NativeIdentity` bytes. `buildCaptureManifest` already
re-decodes the encoded bytes; `IdentityDigest` must do the same before
hashing. Add the negatives (bad kind, bad extension key, unsanitized opaque
identity) and a narrowing mutant.

### P2-f — `decode.go` re-implements the landed environ gate set instead of adopting it (architecture)

`rawUint53`, `checkDigest`, `checkUUIDv7`, `checkTimestamp`,
`checkSortedUniqueStrings`, `checkSortedUniqueDigests`, `checkStringBounds`
duplicate `environ.CheckUint53Bounds`, `CheckDigest`, `CheckUUIDv7`,
`CheckTimestamp`, `CheckSortedUniqueStrings`, `CheckSortedUniqueDigests`,
`CheckStringBounds`, which sessadapter and dirnode already delegate to. The
COMMON CONTRACT says "never fork a second model; extend or adopt". The fork
is where P2-a lives and where the unmeasured uniqueness arms of P2-b live.
Delegate; keep only the package-specific rules (sanitizer, exclusion,
coupling, reconciliation).

### P3 — hygiene, documentation, harness

- P3-a Sanitizer scope is undefined by the pinned text and the implemented
  scope is narrower than the comments imply: U+0085 (C1 control), U+2028,
  U+200B, `URN:AX:…` (case), schemeless `user:secret@host/path` are all
  ADMITTED (probe P3). State the exact sanitizer bound in TRACEABILITY.md or
  harden (`unicode.IsControl`, case-fold the AX marker).
- P3-b Content-block members (`type/content/blob_descriptor_id/media_type/
  extensions`) are a producer design not in the pinned text; a block with
  neither content nor descriptor, and one with both, are admitted (P7/P7b).
  State it as a bound or enforce "typed inline content XOR Blob Descriptor
  reference".
- P3-c `BuildCaptureManifest` seals a manifest whose items ≠ plan candidates
  with `raw_complete=false` (P5) instead of refusing; §13.14.1 says items are
  "one per plan candidate" and §7.8 capture requires "exact plan-key
  reconciliation". Refuse at Build, or state why an incomplete-reconciliation
  manifest is a valid archive object.
- P3-d `N-exclusion-config` is an arm-delete (`&& false`), not a narrowing;
  the arm is genuinely pinned (my `R-exclusion-config-narrow`, admitting
  exactly `settings.bak.3`, is KILLED) — replace the row so the harness says
  what it measures.
- P3-e The shipped harness discards per-plant output; `mutants.log` carries
  verdict lines only, while the COMMON CONTRACT asks for per-plant raw logs
  and subprocess exits. Write one log per plant.
- P3-f LOGBOOK OWNERSHIP line still says "reuses
  `Canonicalize`/`VerifyObjectIdentity` (blobs)" and in the same sentence
  "never the attesting `VerifyObjectIdentity` entry"; results.md claims the
  stale mention was corrected. Production calls neither
  `VerifyObjectIdentity` (only the test fixture does) — fix the sentence.
- P3-g `BoundaryInput.Core` is a caller-asserted boolean; "only core may
  construct" is therefore nominal at this entry and the real protection is
  `RefuseUnstableForTarget`. State that bound explicitly.
- P3-h Dead code: `_ = extensionBytes`, `_ = manifest`, `_ = workspace`,
  `kindOf`; the `decode.go:111` comment states the wrong unit (see P2-a).

## What is clean (verified, not read)

1. **Spec fidelity, member level** — all 16 closed shapes' member sets and
   all 6 vocabularies extracted from the code equal the pinned text
   member-for-member (raw manifest 11, entry 5, capture manifest 17, item 7,
   basis 6/3, boundary 3/8, proof 9, session 14, actor 7, event 14, evidence
   9, raw reference 4, NativeIdentity 6, WorkspaceBinding 8; 26 kinds, 9
   classes, 5 raw classes, 8 block types, 4 proof kinds, 3 identity kinds;
   visibility/status/actor-kind/disposition/basis/boundary/reason_code
   inline vocabularies). Every array/string bound matches the text (unit
   aside, P2-a). No member the text forbids is admitted; no vocabulary entry
   is missing.
2. **Row-21 exclusion** — `refuseExcludedMember` is called on every native
   item key at all four construction/decode paths (`buildRawEntries`,
   `decodeRawEntries`, `buildCaptureItem`, `decodeCaptureItem`); no other
   entry admits a member key. Every handoff class plus `..` escapes has a
   constructor negative in both manifests; absolute paths are refused
   earlier by the sanitizer (tested). True narrowings of both arms
   (`R-exclusion-config-narrow`, `R-exclusion-replication-narrow`) KILLED 2/2.
   The landed matchers admit `host-channel\trust.json` and
   `HOST-CHANNEL/trust.json` on POSIX (probe P3x) — hosttrust's contract,
   outside this leaf.
3. **provhost bound** — `TestNoProductionPathAttestsProviderIdentityBinding`
   PASS on the candidate tree (225 production files, 5 session-leaf sites,
   0 elsewhere; `logs/provhost-census.log`); module-wide grep: non-test
   `VerifyObjectIdentity` callers only in `internal/sessrepo` (5 sites);
   clonebundle production calls `CalculateObjectIdentity` only. An
   any-claim plant at the agreement site (`R-agreement-site-any-claim`) is
   KILLED 2/2 by the resealed-descriptor negative; `N-descriptor-claim`
   reproduces KILLED 3/3 (see P2-c for the install site).
4. **Blob install** — `InstallRawBlob` delegates to `localstore.PutBlob`
   (stage → fsync → size/digest verify → no-replace install → dir fsync);
   short and long byte streams are refused (P14/P14b), tampered bytes refused
   (`TestInstallRawBlobIsIdempotent`), double install agrees. The durability
   bound (no write path in this package; crash/idempotency owned by the
   landed localstore fault suites) is stated and accurate.
5. **Producer mutants** — 17 KILLED + control SURVIVED reproduce in the
   isolated copy, 3/3 each, no `__pycache__`/`.pyc` written
   (`PYTHONDONTWRITEBYTECODE=1`; candidate tree diff-clean against the live
   worktree after every run). My four brief-named classes: raw_complete
   ignoring one unknown item KILLED; always-excluded set missing
   `machine_auth` KILLED; raw subset admitting `machine_auth` KILLED;
   capture-item equal-key duplicate at Build KILLED (at Decode SURVIVED —
   P2-b); sanitizer admitting U+001F KILLED.
6. **Race gate** — see next section.
7. **Hygiene** — `gofmt -l` empty, `go vet` clean, `go build ./...` OK,
   `go test ./internal/clonebundle ./internal/provhost ./internal/localstore
   ./internal/canonicaljson -count=1` all ok, `tracecheck` exit 0,
   `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` exit
   0 on the exact tree; `internal/traceability` untouched; changed paths are
   exactly the 21 candidate paths; README section has test commands and no
   capability/CLI claim; LOGBOOK entry newest-first; sibling bounds stated
   in doc.go, TRACEABILITY.md, results.md.

## Race gate (item 7)

The package is untouched by this change: no file outside
`internal/clonebundle`, `README.md`, `LOGBOOK.md` changed; nothing imports
`clonebundle`; `go list -deps`/`-test -deps ./internal/sessquery` do not
include it, so the sessquery race binary is byte-for-byte the trunk binary.
The rev2 Change Request validation ran the current 27-command suite (trunk
`888ae3d`, `go test ./... -race -count=1 -timeout 25m`) and reports
`required=27 green=27 failed=0 missing=0` — the gate is green under the
suite the candidate is measured by; the producer's red figure was the old
600 s default on the pre-PR51 suite, and disclosing it as a host artifact
rather than claiming a pass was the right disposition.

My own measurement (`logs/sessquery-race.log`): `go test ./internal/sessquery
-race -count=1 -timeout 25m` on the candidate copy → `ok … 390.559s`, wall
394 s, exit 0, zero `DATA RACE` lines; host load averages 14.45 → 9.83 over
the run, with another session's `sessquery.test -test.timeout=25m0s` at
100% CPU at my start and a second sessquery binary appearing mid-run (this
is a shared host; "while nothing else runs" was not available). That is
under the old 600 s default even under that contention; the producer's
630 s was measured at load 19–40. Disposition: host-capacity artifact,
not a candidate defect; the 25 m timeout landed on trunk (`888ae3d`,
TASK-260917-3f8mkd) is the right orchestration fix and the rev2 validation
already ran under it. I did not locate the 96 s idle figure on the board
within the review budget and do not claim it.

## Rework scope (for the producer)

1. Link descriptor identity to the entry (P1-a): compare the calculated
   descriptor identity with `blob_descriptor_id` at `buildRawEntries` and
   `VerifyRawManifestDescriptors`; negatives + narrowing mutant.
2. Enforce the AX number model (and nested duplicate refusal) on extension
   values at Build and Decode (P1-b); negatives (`1.5`, `2^53`, `2^60`) +
   narrowing mutant.
3. Adopt the environ gates (P2-f) — which fixes the string measure (P2-a);
   one multibyte boundary row.
4. Add Decode-side negatives so every row in the P2-b table is KILLED, plus
   `VerifyRawManifestDescriptors` and `InstallRawBlob` negatives (P2-c);
   extend the harness with those rows and per-plant raw logs (P3-e); replace
   `N-exclusion-config` with a narrowing (P3-d); fix the `N-descriptor-claim`
   note or split it per site.
5. Refuse duplicate raw keys in reconciliation (P2-d); re-decode in
   `IdentityDigest` (P2-e).
6. State or enforce the P3-a/b/c/g bounds; fix the LOGBOOK sentence (P3-f)
   and the dead code (P3-h).

Checklist state left by this review: the four reviewer rows (Implementation
matches AC; Solution fits project architecture; Tests green; Gate … attacked,
not read) are left unchecked; "If review does not accept the work — verdict
evidence added and status routed by the explicit verdict branches" is
checked. Status routed to `to-dev`.
