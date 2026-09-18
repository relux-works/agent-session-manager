# TASK-260830-24z2b3 — Review verdict, Change Request revision 4

Reviewer run: RUN-260917-ab25c8 (claude-opus-5, reviewer/reviewer).
Change Request: `CR-TASK-260830-24z2b3-4` revision 4, base
`2fc6d5074058ab7970c87982deac50b3e3f5fd91`, candidate tree
`33b52bcbe16b7dc2b40f8e54168768d2cc73f768` (23 paths; patch sha256
`6bed41e87cfa6e9395bd725bdc840c1ad7ddac118aa783c5ef6065d3d2402b84`, verified).
Authority: `internal/specdoc/SPEC.v0.7.0.md` §13.14.1 (lines 10331–10546), §7.8,
§10.2 and the common logical data model §1.6 (lines 244–337); the pinned
sections are byte-identical to v0.6.0 as the producer states.

## Verdict: CHANGES REQUESTED → `to-dev`

The rev3 rework is real and complete on its own terms: P1-c, P2-α, P2-β, P2-γ
and every P3 item of RUN-260917-62d051 is fixed, each graded below with my own
executed evidence (the reported vector plus vectors one step away; every rev3
survivor re-planted on this tree and watched die twice). What blocks acceptance
is one new admit-hole of the same "rewrite and continue" class that P1-b and
P1-c closed for numbers, found one §1.6 rule away at text:

- **P2-δ** — every Build entry seals Go string inputs that are not valid UTF-8
  after `encoding/json` has silently rewritten the bytes to U+FFFD, and for two
  production entries (`BuildCaptureManifest`, `IdentityDigest`) this launders a
  NativeIdentity native key past the sanitizer's own `native key is not valid
  UTF-8` refusal — the arm the rev4 corpus added and TRACEABILITY.md states as
  the sanitizer's bound. Two distinct inputs seal byte-identical objects.

Nothing here is a Stop-The-Line. The rework scope is narrow (see the end).

Method: isolated immutable copies of the exact candidate tree (`git archive
33b52bcb…` → `.temp/TASK-260830-24z2b3/review-rev4/{candidate,pristine,probe}`,
`.task-board` artifact dropped; the live worktree's temp-index tree OID was
confirmed equal to `33b52bcb…` before anything ran). The live Story worktree,
index, branch and HEAD were never touched (`git status` before and after:
` M LOGBOOK.md`, ` M README.md`, `?? internal/clonebundle/`). Every probe and
plant ran through the production entry points; `PYTHONDONTWRITEBYTECODE=1` for
every harness, no `__pycache__`/`.pyc` written; the candidate copy diffed clean
against the pristine copy after every battery. Raw evidence is in
`TASK-260830-24z2b3_review-evidence-rev4.tar.gz` (`logs/`, `logs/mutants/` one
raw `go test` log per plant per run with the subprocess exit code,
`logs/mutants-producer/run{1,2}/` the shipped harness's per-plant logs,
`reviewer_mutants_rev4.py`, `site_plants.py`, `zz_rev4_probe*_test.go.txt`,
`logs/probes-rev4*.log`, `logs/probes-rev3-on-rev4.log`).

## 0. Rework verification — every rev3 finding graded

| rev3 finding | Grade | Executed evidence (this run, new tree) |
|---|---|---|
| P1-c non-message payloads escape the number model and the duplicate rule | **fixed** | `logs/probes-rev4.log` section A: all 24 non-message kinds × 8 vectors (`2^53+1`, `2^53`, `-(2^53)`, `2^60`, `1.5`, `1e2`, nested duplicate, array-nested duplicate) × 3 entries (Build, Decode unsealed, Decode resealed with the literal preserved textually) = 576 probes, **0 admitted**. Section B one step away: block-extension `2^53+1` under a non-message kind refused (B1); deep-array `2^53+1` refused (B2); `1.0`, `0.0`, `1E2`, `1e+2`, `1e-2`, `-1.5` refused as value faults; `1.`, `.5`, `01`, `+1`, `0x10`, `Infinity`, `NaN` refused at the frame (B4–B8); beyond-int64 refused (B7); safe edges seal exactly (`{"n":9007199254740991}` / `-9007199254740991`, B1b nested); `-0` seals as `0` (JCS form, stated bound). Sealed-bytes exactness asserted by the committed `TestCanonicalEventPayloadValueModel/build_sealed_exact` and `decode` rows. The rev3 probe battery (`zz_rev3_probe_test.go`) re-executed on this tree: all 28 N1 and 7 N2 vectors REFUSED (`logs/probes-rev3-on-rev4.log`). Producer `N-payload-number` KILLED 2/2. |
| P2-α extension-value gate measured at 2 of 17 sites | **fixed** | 17 `checkExtensionsClosed` admission sites counted by grep (boundary.go:90/116/295/392/429, captureitem.go:143, capturebuild.go:308, event.go:298/559/607/751, identity.go:112/237, session.go:94/454, rawmanifest.go:319/483); `TestExtensionValueModelAtEverySite` has 17 rows mapping one-to-one. Class narrowing (drop the value model at exactly one site, whole suite as killer, `site_plants.py`): **17/17 KILLED 2/2** (`logs/mutants-ext-sites-runs1-2.log`, rev3: 2/17). Producer's 17 literal-keyed `N-ext-values-*` rows KILLED 2/2. |
| P2-β `decodeRawEntries` equal-key duplicate at Decode unmeasured | **fixed** | `R3-raw-entries-dup-decode` (`key <= previous` → `<` at the Decode site, whole suite) KILLED 2/2 (rev3: SURVIVED 2/2); producer `N-raw-order-decode` KILLED 2/2; `TestRawManifestRefusals/decode/duplicate_entries` exists. |
| P2-γ duplicate plan keys defeat one-per-plan-candidate | **fixed** | P5c `plan=[a,a,token] items=[a,b,token]` REFUSED "plan keys carry duplicate" (D1); one step away: `[a,token,token]` (D2), `[a,b,b]` (D3) REFUSED; extra/missing candidate REFUSED (D5/D6); `VerifyCaptureReconciliation` with `[a,a,token]` REFUSED (D7); `deriveRawComplete([a,a,token])=false` (D13); `N-plan-keys-dup` KILLED 2/2. |
| P3-a′ content `null` passes the XOR | **fixed** | E1 `content:null`, E2 `descriptor:null`, E3 both null, E6 neither, E7 both REFUSED "not exclusive"; `N-block-content-null` KILLED 2/2. (E4 `content:null` beside a descriptor admits — the null is the absent arm; fine.) |
| P3-b′ three misreporting details | **fixed** | G1 capture item / G2 actor / G3 content block extension `1.5` at decode now report "carry a number outside the AX safe-integer model"; bad key still reports "reverse-DNS" (G4). |
| P3-c′ five unmeasured arms | **fixed** | Re-planted on this tree, all KILLED 2/2: `R3-proof-input-blocked`, `R3-ax-number-negative-edge`, `R3-sanitizer-zp`, `R3-payload-strict-object`, `R3-decode-external-null-parent` (rev3: SURVIVED 2/2 each). Producer rows `N-proof-input-blocked`, `N-ax-number-negative-edge`, `N-sanitizer-zp`, `N-payload-top-dup`, `N-decode-external-null-parent` KILLED 2/2. |
| P3-d′ `EncodeNativeIdentity` rounds | **fixed** | F1 `2^60`, F2 `1.5`, F3 bad key, F5 nested duplicate REFUSED at encode; F4 safe max admits; P2u re-probed REFUSED; `N-encode-identity-extensions` KILLED 2/2. (The pre-validated-identity contract still holds for the identity fields themselves — F6 — which is where P2-δ lives.) |
| P3-e′ `N-main-count` note | **fixed** | Split into `N-main-count-build` / `N-main-count-decode`, each KILLED 2/2; my `R3-decode-main-count` (decode site only, whole suite) KILLED 2/2. |
| P3-f′ dead code | **fixed** | `sourceBasisObject`, `digestOrEmpty`, `digestOrEmptyDigest` gone (grep: none). |
| P3-g′ refusal census | **fixed as disposed** | Version `1.0.1` at capture/session/event decode, `items`/`actors`/`raw_refs` not-an-array, schema of another type all REFUSED (H1–H8); coverprofile-driven census now 369 sites / 213 covered / 156 uncovered (rev3: 366/181/185, `logs/refusal-census.log`); the remainder is owner-grammar repeats ("is not a digest/UUIDv7/timestamp/uint53"), frame/unknown/missing-member arms per shape and infrastructure errors, stated as defense-in-depth in TRACEABILITY.md. |

## Findings

### P2-δ — Build entries rewrite invalid UTF-8 to U+FFFD and seal it; two entries launder a native key past the sanitizer (`identity.go` `EncodeNativeIdentity`, `capturebuild.go:118-124`, `rawmanifest.go:431-443`, every Build string admission)

§1.6: "text MUST be valid UTF-8"; "Decoders MUST reject lone surrogate code
points before canonicalization"; and the whole rev3/rev4 round rests on "MUST
NOT round a value and continue". A Go string carrying `\xff` (or the WTF-8
lone-surrogate bytes `\xed\xa0\x80`) is not valid text; every Build entry
passes it to `json.Marshal`, which substitutes U+FFFD and continues, and the
omit-self identity is sealed over the rewritten bytes. Witnessed through the
production entries (`logs/probes-rev4.log` section C, `logs/probes-rev4-k.log`):

| Entry / member | Input | Result |
|---|---|---|
| `BuildCaptureManifest` `source_identity.native_session_id` | `"native\xff"` | **ADMITTED**, sealed `"native�"`, `DecodeCaptureManifest` of the sealed bytes admits (K1/K1b) |
| `BuildCaptureManifest` `source_identity.opaque_identity` | `"op\xff"` | **ADMITTED**, sealed `"op�"` (K2) |
| `BuildCaptureManifest` `source_identity.native_session_id` | `"native\xed\xa0\x80"` (lone surrogate) | **ADMITTED**, sealed `"native���"` (K3) |
| `BuildCaptureManifest` | `"native\xff"` vs `"native\xfe"` | two distinct inputs seal **byte-identical** manifests (K4) |
| `IdentityDigest` `opaque_identity` | `"op\xff"` | **ADMITTED**; digest equals `IdentityDigest("op�")` (K13/K14) |
| `BuildCanonicalSession` title / actor name / actor model / extension string | `…\xff` | ADMITTED, sealed with U+FFFD (C1–C4); lone-surrogate title sealed as three U+FFFD (C14) |
| `BuildCanonicalEvent` evidence `native_type` / reason code | `…\xff` | ADMITTED, sealed with U+FFFD (C6, C7) |
| `BuildCaptureManifest` item `exclusion_reason` / boundary `source_generation` / item extension string / identity extension string | `…\xff` | ADMITTED, sealed with U+FFFD (C9–C12); `ParseGeneration("g\xff")` admits (C16) |

Every other native-key Build admission refuses the same byte (`native key is
not valid UTF-8`): raw manifest `source_native_session_id` and entry keys (K5,
K6), evidence `native_session_id`/`native_event_id` (K7, K8), actor
`source_native_id` (K9), `external_source_ref` (K10), capture item keys (K11),
`IdentityDigest` `native_session_id` (K12), and every Decode entry refuses the
bytes at the frame (`environ.DecodeStrictObject`, C8). The hole is exactly the
`EncodeNativeIdentity` path: `buildCaptureManifest` encodes the caller's
`NativeIdentity` first and sanitizes only the re-decoded (already rewritten)
bytes, and `IdentityDigest` sanitizes `NativeSessionID` but not
`OpaqueIdentity` before encoding — so the invalid-UTF-8 member of the
sanitizer's class, pinned by `TestIdentityRefusals/sanitizer` (`"a\xffb"`) and
stated in TRACEABILITY.md ("refuses … invalid UTF-8"), is admitted by two
production entries because `json.Marshal` rewrites it before the gate runs.
The sanitizer corpus drives the function, not these entries — the driven half
is right, the entry path is not.

This is the P1-b/P1-c mechanism at a different §1.6 rule: the fix for numbers
validated the literal before the host could round it; text has no such gate at
Build, and this repository already treats U+FFFD substitution as an admit hole
(`internal/environ` refuses lone-surrogate escapes at Decode for exactly this
reason). Graded P2 rather than P1: Decode refuses the bytes, the sealed output
is well-formed, and the affected inputs are core-produced Go strings; but a
production gate admits what it must reject and two different inputs collide on
one identity.

Fix: one shared text gate (`utf8.ValidString`, refusing invalid UTF-8) at every
Build-side string admission — title, actor name/model, native_type, reason
codes, exclusion_reason, generation, extension string values (walk the
marshalled bytes or the Go values before `encodeExtensions` accepts them) — and
sanitize `NativeSessionID`/`OpaqueIdentity` before `EncodeNativeIdentity`
marshals them (or make the exported entry validate them the way it now
validates extensions). Negatives at both laundering entries
(`BuildCaptureManifest` source identity, `IdentityDigest` opaque identity) plus
one per Build shape; assert the sealed bytes never carry U+FFFD for an input
that did not; one narrowing mutant (admit exactly one invalid byte value, e.g.
`\xff`, or exempt exactly one member).

### P3 — bounds, precision, hygiene

- P3-a″ **Producer evidence archive is contaminated.**
  `TASK-260830-24z2b3_producer-evidence.tar.gz` (rev4) carries, beside the
  27 command logs, package logs and the 2×65 per-plant logs it describes, 20+
  foreign artifacts dated 2026-09-16/17 from the TASK-260909-2ez769 review
  (`attacks.py`, `attacks.json`, `test_r4_cr_smuggle.py`, `test_r4_bypass_new.py`,
  `test_r3_*_rerun.py`, `capture_cr.py`, `replay_mutants.py`,
  `review-evidence-rev4.zip`, `review-logbook-rev4.md`, `mypy.log`, …) and a
  `__pycache__/` with three `.pyc` files — the contract says no `__pycache__`,
  and the MANIFEST.sha256 lists them as if they were this task's evidence.
  Rebuild the archive from a task-scoped directory (the rev4 logs themselves
  are fine: `mutants-run{1,2}.log` reproduce exactly in my runs).
- P3-b″ **TRACEABILITY.md over-claims a bound**: "`-0` and leading-`+`/leading-zero
  integer literals in extension values and payloads admit". Only `-0` admits
  (sealed as `0`, the JCS form); `+1` and `01` are refused at the frame gate
  (`decodeStrictObject`/`checkExtensionValues` cannot tokenize them — probes B8:
  "not a JSON object"). Fix the sentence.
- P3-c″ **Build-side reason-code duplicate unmeasured**: `R4-reason-codes-dup-build`
  (`reason <= previousReason` → `<` in `buildSourceEvidence`) SURVIVED the whole
  suite 2/2; the arm works (L1: `ReasonCodes: ["a","a"]` at Build REFUSED "not
  sorted unique") but the build rows have only `unsorted` (`b,a`) and `empty`;
  the duplicate row exists at Decode only. Same class as P2-β one site over:
  add the build duplicate row and a harness row.
- P3-d″ **Producer ticked reviewer-owned checklist rows.** The live checklist
  arrived with "Implementation matches AC", "Solution fits project
  architecture", "Tests green", "Gate … attacked, not read" and "If review does
  not accept …" all checked; those are verdict rows and are reset by this
  review. When "reset and re-complete the checklist" is the brief, re-complete
  the producer rows only.
- Informational (not defects, no action required unless cheap): per-member
  narrowings that survive because the class is pinned by one member —
  `R4-sanitizer-c1-u009f` (stated owner-informational), `R4-sanitizer-c0-u0001`
  (C0 pinned by U+0000/U+001F/U+007F), `R4-plan-dup-other-key`
  (`store/token-cache` duplicate; the row duplicates `store/blob-a`),
  `R4-cwd-dotdot` (bare `..`; row has `../escape`), `R4-unstable-reason-code`
  (`source_quiescent`; row has `tired`), `R4-evidence-partial-operation`
  (`partial`+operation; rows use `exact`) — every arm refuses when probed (L2–L6).
  Spec-silent behaviours worth a stated line: an actor whose
  `parent_actor_id` is itself or names no actor is admitted (I23/I24); plan
  keys are matched as a set, so an unsorted permutation of the plan admits
  (D4) while the §7.8 sink is sorted unique; content-block `content` is always
  a string, so a `json` block carries its JSON as text (E9).

## What is clean (verified, not read)

1. **Spec fidelity, member level** — re-extracted every closed shape from the
   pinned text and compared member-for-member: Raw Object Manifest (11
   members) and `RawObjectEntry` (5; the prose "and extensions" is overridden
   by the closed shape), Capture Manifest (17), Source Basis `ax_session` (6) /
   `external_native` (3), `CaptureItem` (7, nine classes), Capture Boundary
   `stable` (3) / `unstable_archive` (8), `StableSnapshotProof` (9, four
   kinds), `NativeIdentity` (6, three kinds), `WorkspaceBinding` (8),
   Canonical Session (14), `Actor` (7, three kinds), Canonical Event (14, 26
   kinds, four visibilities), `SourceEvidence` (9, four statuses),
   `RawReference` (4), eight content-block types; every bound
   ([1..512]/[1..4096]/[1..1024]/[0..64]/[0..128]/[0..65536]/[1..1000000]/
   [1..1024]/[0..9]/[1..128][0..128]/64 KiB) matches. The rev3→rev4 production
   delta (`git diff d5b5b0ee…33b52bcb`) touches only `event.go` (payload value
   model), `capturebuild.go` (plan-key duplicates), `captureitem.go`/`session.go`
   (detail text), `identity.go` (encode validation), `decode.go` (one error
   value), `boundary.go` (dead code removed); `blobref.go` and `rawmanifest.go`
   are byte-identical to rev3, so the rev3 verification of descriptor linking,
   exclusion and blob install stands and was re-executed here.
2. **Row-21 exclusion** — `refuseExcludedMember` at all four
   construction/decode paths; every handoff class plus `../escape` and
   `state/../../escape` has a constructor negative in both manifests and a
   decode negative; absolute paths are refused one gate earlier by the
   sanitizer. The rev3 P3x rows re-executed unchanged (`host-channel\trust.json`,
   `HOST-CHANNEL/…`, `x/host-channel/…`, bare `lock` admit — the hosttrust
   matcher's anchored STATE_DIR-relative contract, outside this leaf). Producer
   `N-exclusion-config` KILLED 2/2.
3. **provhost bound** — `TestNoProductionPathAttestsProviderIdentityBinding`
   PASS on the exact tree (`logs/provhost-census.log`); module grep: non-test
   `VerifyObjectIdentity` callers are only `internal/sessrepo` (5 sites);
   `clonebundle` production calls `CalculateObjectIdentity` ×1 and
   `Canonicalize` ×3 only; nothing imports `clonebundle`. Producer
   `N-descriptor-claim-agreement`, `N-install-site-claim`, `N-descriptor-id-link`
   KILLED 2/2; rev3 P1/P1x/P1y/P1z/P1w re-executed REFUSED.
4. **Blob install** — unchanged code; rev3 P14 series re-executed on this tree:
   short/long/same-length-wrong bytes REFUSED, `blob_id != chunk digest` REFUSED
   at the claim, correct install ADMITTED, re-install different bytes REFUSED
   (no-replace), same bytes ADMITTED, owner refuses size/chunk disagreement.
   Durability bound stated and accurate (no write path in this package;
   `internal/localstore` green in the four-package run).
5. **Mutation** — shipped harness reproduced in the isolated copy: 64 KILLED +
   `C-doc-comment` SURVIVED, twice, tree restored (`logs/mutants-producer-runs{1,2}.log`,
   130 per-plant logs); rev3 survivors 7/7 KILLED 2/2; per-site class plants
   17/17 KILLED 2/2; my rev4 plants (`reviewer_mutants_rev4.py`, 23 rows × 2):
   the brief's four classes KILLED (`R4-unknown-ignores-one`,
   `R4-excluded-set-missing-runtime_state`, `R4-raw-subset-admits-transient_lock`,
   sorted-unique duplicates via `R3-raw-entries-dup-decode`/`R4-rawrefs-dup-build`,
   sanitizer control via the producer's `N-sanitizer-control`), plus
   `R4-heads-skip-last-{build,decode}`, `R4-sanitizer-schemeless-cred`,
   `R4-drive-prefix-backslash`, `R4-inline-65537`, `R4-ordinal-edge`,
   `R4-actor-external-null-parent-build`, `R4-digests-equal-one-order`,
   `R4-unstable-operator-explicit` (re-planted after a compile-fail first
   shape), `R4-generation-empty`, `R4-immutable-identity-required` KILLED 2/2;
   `C4-comment-control` SURVIVED 2/2; the seven survivors are P3-c″ and the
   informational per-member rows above.
6. **Race gate** — see the next section.
7. **Hygiene** — `gofmt -l` empty; `go vet` clean; `go build ./...` OK; `go test
   ./internal/clonebundle ./internal/provhost ./internal/localstore
   ./internal/canonicaljson -count=1` all ok (`logs/four-packages.log`);
   `go run ./internal/traceability/cmd/tracecheck` exit 0 (bindings=68,
   clauses 49/569 — `internal/traceability` untouched); `cataloggen -adopted
   -output internal/catalog/catalog_gen.go -check` exit 0 on the exact tree;
   `git diff --check` clean; changed paths are exactly the 23 candidate paths
   (README/LOGBOOK purely additive, +27/+10; nothing outside
   `internal/clonebundle`; no `__pycache__`/`.pyc` in the CR; the CR validation
   log reports `required=27 green=27 failed=0 missing=0`). README section has
   the three test commands and no capability claim. LOGBOOK entry newest-first.
   Package coverage 86.8% statements. All 32 matrix-named tests exist; the
   conformance matrix's 18-of-18 ratio and call sites check out. Sibling bounds
   stated in doc.go, TRACEABILITY.md and results.md.

## Race gate (item 7)

`internal/sessquery` is untouched by this change (`go list -deps` and
`-test -deps` of sessquery: 0 clonebundle; nothing imports clonebundle). The
rev4 Change Request validation ran the 27-command suite with the landed
`-timeout 25m` and is green (`ok internal/sessquery 390.722s` in the producer's
cmd05). My own measurement on the pristine candidate copy
(`logs/sessquery-race.log`): `go test ./internal/sessquery -race -count=1
-timeout 25m` → `ok … 417.433s`, wall 419.6 s, exit 0, zero `DATA RACE` lines,
host load 10.95 → 17.28 with another session's full `go test ./...` running
concurrently (eight `.test` binaries alive mid-run); an idle host was not
available. Three measurements under different load (rev2 390.6 s, rev3 347.4 s,
rev4 417.4 s) all sit under the old 600 s default; the producer's earlier 665 s
was under the full concurrent suite. Disposition unchanged: host-capacity
artifact, not a candidate defect; the 25 m timeout on trunk (`888ae3d`) is the
orchestration fix and the gate is green in this revision.

## Rework scope (for the producer)

1. P2-δ: a shared valid-UTF-8 text gate at every Build-side string admission;
   sanitize `NativeSessionID` and `OpaqueIdentity` before `EncodeNativeIdentity`
   marshals them (or validate them inside the exported entry); negatives at
   `BuildCaptureManifest` (source identity native/opaque), `IdentityDigest`
   (opaque) and one per Build shape, asserting no U+FFFD in the sealed bytes;
   one narrowing mutant + harness row.
2. P3-a″: rebuild the producer evidence archive from a task-scoped directory
   (no foreign artifacts, no `__pycache__`).
3. P3-b″: fix the `-0`/`+`/leading-zero sentence in TRACEABILITY.md.
4. P3-c″: build-side reason-code duplicate row + harness row.
5. P3-d″: re-complete only the producer checklist rows.

Checklist state left by this review: the four reviewer rows (Implementation
matches AC; Solution fits project architecture; Tests green; Gate … attacked,
not read) are unchecked; "If review does not accept the work — verdict evidence
added and status routed by the explicit verdict branches" is checked. Status
routed to `to-dev`.
