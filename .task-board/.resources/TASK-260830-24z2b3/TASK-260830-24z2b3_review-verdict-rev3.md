# TASK-260830-24z2b3 — Review verdict, Change Request revision 3

Reviewer run: RUN-260917-62d051 (claude-opus-5, reviewer/reviewer).
Change Request: `CR-TASK-260830-24z2b3-3` revision 3, base
`888ae3dff6a55c29bfaf311957748d405ed1ab06`, candidate tree
`d5b5b0eedad46636fb0b18a8652f99849658e98e` (21 paths; patch sha256
`8f244fe68f1ce8acb4b8e6418b35b5e516464c0fa77a82eb7a03ee84306b2461`).
Authority: `internal/specdoc/SPEC.v0.6.0.md` §13.14.1 (lines 10050–10265),
§7.8, §10.2 (4666–4718) and the common logical data model §1.6 (224–337).

## Verdict: CHANGES REQUESTED → `to-dev`

The rev2 rework is real and complete on its own terms: every P1/P2/P3 finding of
RUN-260917-c05bc8 is fixed (graded below with my own executed evidence, each
reported vector plus a vector one step away, every rev2 survivor watched die
twice). What blocks acceptance is one correctness hole of exactly the P1-b
class at a member the rev2 review did not probe, plus three recurrences of the
"gate wired, evidence missing" shape the rev2 review already asked to close:

- **P1-c** — non-message Canonical Event payloads escape the AX number model
  and the nested-duplicate rule at Build AND Decode: `BuildCanonicalEvent`
  ROUNDS `9007199254740993` to `9007199254740992` and `2^60` to
  `1152921504606847000`, collapses `{"a":1,"a":2}` to `{"a":2}`, seals `1.5`
  as-is, and continues; `DecodeCanonicalEvent` admits all of them resealed.
- **P2-α** — the new extension-value gate is measured at 2 of its 17
  admission sites (per-site narrowing plants survive the whole suite at 15
  sites, 2/2).
- **P2-β** — `decodeRawEntries` admits an equal-key duplicate entry at Decode
  under a `<=`→`<` narrowing; the whole suite stays green (2/2).
- **P2-γ** — `checkPlanItemMatch` (the P3-c fix) is defeated by duplicate plan
  keys: `plan=[a,a,token]` with items `[a,b,token]` seals a manifest carrying
  an item that is not a plan candidate.

Nothing here is a Stop-The-Line. The rework scope is narrow (see the end).

Method: isolated immutable copies of the exact candidate tree (`git archive
d5b5b0ee…` → `.temp/TASK-260830-24z2b3/rev3-review/{candidate,pristine}`,
`.task-board` artifact dropped; the live worktree's own temp-index tree OID was
confirmed equal to `d5b5b0ee…` before anything ran). The live Story worktree,
index, branch and HEAD were never touched (`git status` before and after:
` M LOGBOOK.md`, ` M README.md`, `?? internal/clonebundle/`). Every probe and
plant ran through the production entry points; `PYTHONDONTWRITEBYTECODE=1`
for every harness, no `__pycache__`/`.pyc` written; both copies diff-clean
against each other after every battery. Raw evidence is in
`TASK-260830-24z2b3_review-evidence-rev3.tar.gz` (`logs/`, `logs/mutants/`
one raw `go test` log per plant per run with the subprocess exit code — 223
files, `reviewer_mutants_rev3.py`, `site_plants.py`, `per_site_runs.py`,
`zz_rev3_probe_test.go.txt`, `logs/probes-rev3.log`).

## 0. Rework verification — every rev2 finding graded

| rev2 finding | Grade | Executed evidence (this run, new tree) |
|---|---|---|
| P1-a descriptor identity unlinked | **fixed** | Probes P1 (reported vector) REFUSED "identity … disagrees with entry claim"; one step away: P1x/P1y (a real sibling descriptor for the same bytes — same `blob_id`/`size`, other `media_type`) REFUSED at build in both directions; P1z REFUSED at `VerifyRawManifestDescriptors` when the fetch returns the sibling; P1w REFUSED at entry 1 (not only entry 0). Rev2 witness `R-descriptor-id-unlinked2` (SURVIVED 3/3 then) KILLED 2/2; `R-verify-descriptors-skip-0` KILLED 2/2; producer `N-descriptor-id-link` KILLED 2/2; my `R3-link-any-claim` (accept any claim) and `R3-link-narrow-sibling` KILLED 2/2. |
| P1-b AX number model in the identity fork | **fixed for `extensions`, recurs in payloads (P1-c)** | Build: `1.5`, `2^53`, `2^53+1`, `-(2^53)`, `2^60` (int and float), `1e21`, `json.Number("1e2")`, nested depth-3 float/`2^53` all REFUSED "outside the AX safe-integer model"; edges `2^53-1`, `-(2^53-1)` ADMITTED. Decode: `1.5`, `2^53`, `2^53+1`, `-(2^53)`, `1e2`, `1E400`, `[1.5]`, `{"a":{"b":1e2}}`, `{"a":1,"a":2}`, `{"a":{"b":1,"b":2}}`, `[{"b":1,"b":2}]` REFUSED unsealed AND resealed over the collapsed form (probes P2/P2b/P2c); `-0` admits (stated bound, parity with `validateAXNumbers`). Wiring: `1.5` injected at every one of the 16 Decode admission sites REFUSED (P2w), and at every Build input (P2v). Producer `N-extension-number` (admit exactly `2^53`), `N-extension-nested-dup`, `N-extension-keys` KILLED 2/2. `omitSelfDigest` itself still decodes leniently (P2 rows: it ADMITS `1.5`, `2^53+1`, nested duplicates) — acceptable only because every admission site gates first; see P1-c for the site that does not. |
| P2-a bytes measure | **fixed** | S0: `environ.StringLength == stringLength == 300` for 300 chars/600 bytes; S1–S6: 300/512 chars admitted, 513 refused for native keys (2- and 4-byte characters), title 4096/4097, reason 128/129, generation 512/513, reason code 128/129; decode of a 512-char key admitted. |
| P2-f environ fork | **fixed** | `decode.go` scalar helpers are one-line delegations (`environ.CheckUint53Bounds/CheckStringBounds/CheckDigest/CheckUUIDv7/CheckTimestamp/CheckSortedUniqueStrings/CheckSortedUniqueDigests/CheckExtensions/StringLength/DecodeStrictObject`); grep for `rawUint53|parseUint53|text <= previous|value <= previous|regexp|ParseFloat|RuneCount` in production: none (only `hasDrivePrefix` keeps a byte `len`, correctly). Not merely renamed: the wrapper-class narrowings `R-sorted-digests-dup-decode`/`R-sorted-strings-dup-decode` (admit ANY equal pair, not the killer's literal) KILLED 2/2 whole-suite and KILLED at every site individually (heads, parents, fingerprints; excluded_classes, reason_codes — `logs/mutants-per-site.log`). |
| P2-b ten unmeasured decode arms | **fixed** (one same-class residue: P2-β) | All ten rev2 survivors re-planted on the new tree and KILLED 2/2 each: `R-capture-items-dup-decode`, `R-sorted-digests-dup-decode`, `R-sorted-strings-dup-decode`, `R-rawrefs-dup-decode`, `R-decode-item-class`, `R-decode-actor-main-parent`, `R-decode-evidence-status`, `R-decode-event-visibility`, `R-verify-descriptors-skip-0`, `R-install-site-claim`. Producer rows `N-capture-items-dup-decode`, `N-sorted-*-dup`, `N-rawrefs-dup-decode`, `N-decode-*`, `N-verify-descriptors-skip-0`, `N-install-site-claim` KILLED 2/2. Probes P10–P12 all REFUSED. |
| P2-c positive-only verifiers; over-claiming row | **fixed** | `TestVerifyRawManifestDescriptorsRefusals` drives fetch-error/absent/wrong/link (P15/P15b re-probed); `TestInstallRawBlobRefusals` drives the install-site claim + short/long; `N-descriptor-claim` split into `N-descriptor-claim-agreement` + `N-install-site-claim`, both KILLED 2/2. |
| P2-d duplicate raw keys | **fixed** | P4 `deriveRawComplete(raw=[a,a])=false`; P4d/P4e `VerifyCaptureReconciliation` REFUSED "raw keys carry duplicate"; plan-key duplicates REFUSED (P4f/P4g); `N-reconcile-rawkeys-dup` KILLED 2/2; my `R3-derive-raw-subset` (drop the all-half) KILLED 2/2. |
| P2-e IdentityDigest seals undecodable | **fixed** | P9–P9g: bad kind, bad extension key, extension `1.5`, opaque `urn:ax:`, empty opaque, 513-char opaque, 513-char native ID all REFUSED "seals no undecodable identity"; `N-identity-rededecode` KILLED 2/2. |
| P3-a sanitizer scope | **fixed** (bound stated; two corpus gaps, P3) | U+0085/U+009F (Cc), U+2028 (Zl), U+2029 (Zp), U+200B/U+FEFF/U+2060/U+00AD/U+061C/U+200D/U+180E/U+0600/U+FFF9/U+E0001 (Cf), tab/newline REFUSED; `URN:AX:`/`Urn:Ax:`/`urn:AX:` REFUSED; `user:secret@host/path`, `http://:pw@host/`, `:@` REFUSED; spaces, NBSP, ideographic space, `a@b:c`, `@:`, `~/home`, private-use U+E000 ADMITTED (as the stated bound says). `R3-sanitizer-ax-case`, `R3-sanitizer-schemeless-cred`, `R3-sanitizer-c1` (U+0085), `R3-sanitizer-zwsp`, `R3-sanitizer-abs`, producer `N-sanitizer-control`/`N-sanitize-uuid` all KILLED 2/2. |
| P3-b content-block members | **fixed** (one residue, P3) | P7/P7b neither/both REFUSED "not exclusive"; `N-block-xor` KILLED 2/2. |
| P3-c items ≠ plan sealed | **fixed** (one residue, P2-γ) | P5/P5b extra/missing plan key REFUSED "one per plan candidate"; `N-plan-item-match` and my `R3-plan-match-renamed` KILLED 2/2. |
| P3-d `N-exclusion-config` arm-delete | **fixed** | Now a true narrowing (admits exactly `settings.bak.3`), KILLED 2/2; my two further token-preserving narrowings (`R3-exclusion-replication-narrow` = pending-commit.json, `R3-exclusion-config-narrow-nested` = `state/.ax-config-stage-2`) KILLED 2/2. |
| P3-e verdict-only harness log | **fixed** | Shipped `--log-dir` writes one raw log per plant with `# exit=` and the go test output (producer archive: 37 logs per run × 2; my own smoke of the shipped harness: `logs/shipped-harness/`). |
| P3-f LOGBOOK sentence | **fixed** | LOGBOOK now says production calls `CalculateObjectIdentity` + local comparisons, never `VerifyObjectIdentity` (only the fixture) — verified by module grep (below). |
| P3-g `Core` nominal | **fixed** | Stated in the `BoundaryInput` comment and TRACEABILITY.md; `RefuseUnstableForTarget` named as the real protection. |
| P3-h dead code | **fixed** (new dead code found, P3) | `_ = extensionBytes/manifest/workspace`, `kindOf`, wrong-unit comment gone. |

## Findings

### P1-c — Non-message payloads escape the number model and the duplicate rule (`event.go` `checkPayload`, `BuildCanonicalEvent`, `DecodeCanonicalEvent`)

`checkPayload` runs `decodeStrictObject` (top-level frame only) and, for the 24
non-message-like kinds, validates only `content_blocks` when present; the
payload then enters the object through lenient `json.Unmarshal` (float64,
last-duplicate-wins) and is sealed by `canonicalizeObject`. Witnessed through
the production entries for kinds `usage`, `tool_call`, `opaque_event`, `error`
(`logs/probes-rev3.log`, rows N1/N1b/N2):

| Caller payload | `BuildCanonicalEvent` | Sealed payload bytes | `DecodeCanonicalEvent` (resealed) |
|---|---|---|---|
| `{"n":1.5}` | ADMITTED | `{"n":1.5}` | ADMITTED (`Payload["n"]=1.5`) |
| `{"n":9007199254740992}` | ADMITTED | `{"n":9007199254740992}` | ADMITTED |
| `{"n":9007199254740993}` | ADMITTED | `{"n":9007199254740992}` — **rounded and continued** | ADMITTED |
| `{"n":1152921504606846976}` | ADMITTED | `{"n":1152921504606847000}` — rounded | ADMITTED |
| `{"n":1e2}` | ADMITTED | `{"n":100}` | ADMITTED |
| `{"facts":{"a":1,"a":2}}` | ADMITTED | `{"facts":{"a":2}}` — duplicate collapsed | ADMITTED |
| `{"n":[{"a":1,"a":2}]}` | ADMITTED | `{"n":[{"a":2}]}` | ADMITTED |

§1.6: "floating-point numbers … and duplicate keys are forbidden"; "The
safe-integer restriction is part of every 1.0.0 JSON, CBOR, identity, and wire
contract. A decoder MUST reject a numeric literal at or beyond 2^53 … 
Implementations MUST NOT round a value and continue"; fixture NUM-UNSAFE-ROUND
names exactly the behaviour witnessed ("an implementation that first rounds it
to 9007199254740992 is nonconforming"). This is the P1-b mechanism at a
different member: the omit-self identity of a Canonical Event 1.0.0 is sealed
over rounded or collapsed bytes. TRACEABILITY.md states "per-kind fact values
(including their numbers) are the normalization sibling's registry"; that bound
covers WHICH facts a kind may carry ("Payloads contain only their registered
typed facts"), not the §1.6 value model, which governs every member of every
1.0.0 object and which this leaf's own `checkPayload` is the only gate for — a
sibling registry cannot un-round what `BuildCanonicalEvent` has already sealed,
and `opaque_event` payloads have no registry at all. Message-like payloads are
closed (P2w/P2v rows: block and payload `extensions` refuse `1.5`/`2^53`).

Fix: walk the payload object with the existing `checkExtensionValues` walker
(it already accepts any object and enforces integer literals, |n| ≤ 2^53−1, no
fraction/exponent, duplicate-free nested objects, depth bound) at Build and
Decode for every kind, or decode the payload with `UseNumber` and validate
before sealing; negatives for `1.5`, `2^53`, `2^53+1` (assert the sealed bytes
are never produced), a nested duplicate, at both entries for a non-message
kind; a narrowing mutant (admit exactly `2^53` in payloads).

### P2-α — The extension-value gate is measured at 2 of 17 sites

`checkExtensionsClosed` is called at 16 Decode admission sites plus the shared
Build helper `encodeExtensions`. A per-site narrowing that keeps the key check
and drops the value model at ONE site (`site_plants.py`, whole suite as killer,
2 runs each, `logs/mutants/S-ext-values-*`):

| Site | Verdict |
|---|---|
| `rawmanifest.go:319` (raw manifest decode), `rawmanifest.go:483` (`encodeExtensions`, all Build entries) | KILLED 2/2 |
| `boundary.go:90` ax basis, `:116` external basis, `:331` stable proof, `:428` stable boundary, `:465` unstable boundary; `capturebuild.go:308` capture manifest; `captureitem.go:143` capture item; `event.go:297` source evidence, `:558` event, `:603` message payload, `:675` content block; `identity.go:112` native identity, `:233` workspace binding; `session.go:94` actor, `:454` session | **SURVIVED 2/2** (15 sites) |

The arms all work (P2w rows all REFUSED), so this is missing evidence, not
broken code — the same shape as rev2 P2-b for the new gate. The conformance
matrix cites one test (`TestRawManifestRefusals/extension_values`) for the
whole rule and the refusal census confirms the 14 Decode `extensions %s` sites
are uncovered. Fix (test-only): one value-fault row per admission site — a
table over every Decode entry injecting `1.5` (or `2^53`) into that shape's
`extensions`, plus the two payload-level sites — and a harness row per site or
a per-site loop.

### P2-β — `decodeRawEntries` equal-key duplicate at Decode is unmeasured (`rawmanifest.go:389`)

`R3-raw-entries-dup-decode` (`key <= previous` → `key < previous` at the Decode
site only) SURVIVED the whole suite 2/2 (`logs/mutants/R3-raw-entries-dup-decode-run{1,2}.log`)
and SURVIVED `-run TestRawManifestRefusals/decode`. The decode table has
"unsorted entries" (a swap) but no equal-key duplicate row; the producer's
`N-raw-order` mutates only the Build site. Same class as rev2 P2-b (the capture
items got their decode duplicate row in rev3; the raw entries did not). Fix
(test-only): a decode duplicate-entry row + harness row.

### P2-γ — Duplicate plan keys defeat the one-per-plan-candidate rule at Build (`capturebuild.go:361-375`)

`checkPlanItemMatch` compares `len(items)` with `len(planKeys)` and then checks
plan→item membership only, so a repeated plan key hides an unplanned item.
Probe P5c: `PlanKeys=[store/blob-a, store/blob-a, store/token-cache]`,
`Items=[store/blob-a, store/blob-b, store/token-cache]` → `BuildCaptureManifest`
ADMITTED and sealed (`raw_complete=false`, 3 items) although `store/blob-b` is
not a plan candidate — exactly the object P3-c was fixed to refuse. This is the
P2-d asymmetry one level up (the "no extra" half is defeated by a duplicate the
way the "all" half of `deriveRawComplete` was). The capture-plan sink is
"sorted unique CapturePlanItem" (§7.8), so duplicate plan keys are malformed
input that Build must refuse, not seal. Fix: refuse duplicate `PlanKeys` in
`checkPlanItemMatch` (and/or check item→plan membership); negative + narrowing
mutant. `VerifyCaptureReconciliation` is not affected (P4f/P4g: duplicate plan
keys derive false and are refused).

### P3 — bounds, precision, hygiene

- P3-a′ **Content block `content: null` passes the XOR** (`event.go:650-663`):
  `hasContent` is member presence, and `rawString(null)` succeeds with `""`,
  so `{"type":"text","content":null}` is ADMITTED (probe P7d) while
  `blob_descriptor_id: null` is REFUSED (P7e). Make the XOR a non-null test.
- P3-b′ **Refusal detail misreports value faults** at `captureitem.go:144`,
  `session.go:95`, `event.go:676`: every extension fault (number, nested
  duplicate, depth) is reported as "are not reverse-DNS keyed" (P2w rows
  `items[0].extensions 1.5`, `actors[0].extensions 1.5`,
  `content_blocks[0].extensions 1.5`). Use `%s` with the fault like the other
  13 sites.
- P3-c′ **Unmeasured arms found by narrowing** (each SURVIVED 2/2, whole
  suite; the arms themselves work per probe): the `input_blocked` boolean of
  the closed_store/provider_quiescence coupling (`R3-proof-input-blocked`;
  probe P20c refuses; only foreground/background idle have rows); the negative
  edge `-(2^53)` of `checkAXNumberLiteral` (`R3-ax-number-negative-edge`;
  probe refuses); the Zp arm (U+2029) of the sanitizer (`R3-sanitizer-zp`;
  corpus pins Zl U+2028 and Cf only); the payload top-level duplicate refusal
  (`R3-payload-strict-object`; probe N3e refuses); the decode-side external
  actor with null parent (`R3-decode-external-null-parent`; decode rows use a
  subagent only). Informational, not defects: U+009F (C1 class pinned by
  U+0085), the fraction arm of `checkAXNumberLiteral` (subsumed by
  `ParseInt`), the 256-depth bound moved by one.
- P3-d′ **`EncodeNativeIdentity` (exported) rounds** unsafe extension values:
  `EncodeNativeIdentity(identity, {"com.example.n": 1<<60})` returns bytes
  carrying `1152921504606847000` (probe P2u) while its comment says "encoding
  never admits what decoding refused". The two production consumers
  (`IdentityDigest`, `buildCaptureManifest`) re-decode, so no sealed object is
  affected; either validate inside the exported entry or state the
  pre-validated-input contract instead of the false sentence.
- P3-e′ **`N-main-count` note over-claims**: `count=2` patches build AND decode
  but its killer (`build_actors/two_mains`) reaches the build site only. The
  decode site IS pinned (`decode/two_mains` kills my decode-only
  `R3-decode-main-count`), so split the row or fix the note.
- P3-f′ **Dead code**: `sourceBasisObject`, `digestOrEmpty`,
  `digestOrEmptyDigest` (`boundary.go:192-226`) have no caller.
- P3-g′ **Refusal census** (`logs/refusal-census.log`, coverage-driven over
  every `invalid(` site): 366 sites, 181 covered, 185 uncovered (rev2:
  359/156/203). Beyond P2-α, the substantive uncovered arms are: `version is
  not 1.0.0` at capture/session/event decode (only the raw manifest has a
  "bad version" row), `items`/`actors`/`entries`/`raw_refs` "not an array",
  every count maximum (65536 entries/items/raw_refs, 1024 actors, 128 reason
  codes), the Build-side uint53 overflow arms, `blob descriptor self field is
  %q` and `native key is not valid UTF-8` (probe `\xff` refuses). State them
  as defense-in-depth or add rows.

## What is clean (verified, not read)

1. **Spec fidelity, member level** — the rev2→rev3 delta touches no member
   table or vocabulary (`git diff bee4c807…d5b5b0ee`: only gate wiring,
   wrappers, comments), so the rev2 member-for-member verification of the 16
   closed shapes and 6 vocabularies stands; I re-read §13.14.1 and re-checked
   the rules the rework added against the text: "typed inline content or Blob
   Descriptor references" (XOR), "one per plan candidate", "inline content is
   at most 64 KiB" (bytes — N3c refuses 40,000 chars/80,000 bytes),
   "Closed-store/provider-quiescence requires all booleans true and null
   snapshot identity; immutable-snapshot/log-prefix requires non-null
   identity" (P20/P20b/P20c/P21), 0..64 parents (P11i refuses 65), 1..1024
   actors (P12i refuses 1025), `RawObjectEntry` has exactly five members (the
   prose "and extensions" is overridden by the closed shape, which the code
   follows).
2. **Row-21 exclusion** — `refuseExcludedMember` is called at all four
   construction/decode paths (`buildRawEntries`, `decodeRawEntries`,
   `buildCaptureItem`, `decodeCaptureItem`); no other entry admits a member
   key. Every handoff class (trust.json, config-binding.json,
   pending-commit.json, credentials/<hex>/private-key.pem, credentials/leaf.pem,
   lock, `.trust-stage-*`, `.custody-stage-*`, `lock-stage-*`, `.bak.N`,
   `.pre-rollback.N`, `.ax-config-*`) plus absolute paths and `..` escapes has
   a constructor negative in both manifests and a decode negative; true
   token-preserving narrowings of both arms KILLED 2/2. The landed matchers
   admit `host-channel\trust.json`, `HOST-CHANNEL/trust.json`,
   `x/host-channel/trust.json`, `host-channel/TRUST.json` (P3x) — hosttrust's
   anchored STATE_DIR-relative contract, outside this leaf, as in rev2.
3. **provhost bound** — `TestNoProductionPathAttestsProviderIdentityBinding`
   PASS on the exact tree (225 production files, 5 session-leaf call sites, 0
   elsewhere; `logs/provhost-census.log`); module-wide grep: non-test
   `VerifyObjectIdentity` callers are only `internal/sessrepo` (5 sites);
   `clonebundle` production calls `CalculateObjectIdentity` ×2 and
   `Canonicalize` ×3 only. The any-claim plant at the link arm
   (`R3-link-any-claim`) is KILLED 2/2; the wrong-claim negatives pin both
   the agreement and the install site (`N-descriptor-claim-agreement`,
   `N-install-site-claim`, `R-install-site-claim` KILLED 2/2).
4. **Blob install** — `InstallRawBlob` delegates to `localstore.PutBlob`:
   short, long, same-length-wrong bytes REFUSED "declared and staged identity
   differ" (P14/P14b/P14c); a re-install with different bytes after a correct
   install REFUSED, the same bytes ADMITTED (P14f/P14g — no-replace); a
   descriptor whose `size` disagrees with its chunks is refused by the owner
   before install (P14h); `blob_id != chunk digest` resealed is refused at the
   claim (P14d). Durability bound stated and accurate (no write path in this
   package; `internal/localstore` fault suites green in the four-package run).
   No crash plant of my own: there is no write between which and a receipt
   this package could crash.
5. **Mutation** — producer battery reproduces in the isolated copy: 36 KILLED
   + control SURVIVED, run twice (`logs/mutants-producer-runs1-2.log`, 74 raw
   logs); rev2 re-plants: 19 rows × 2 (`logs/mutants-rev2-replants-runs1-2.log`);
   rev3 plants: 35 rows × 2 + 1 re-plant (`logs/mutants-rev3-runs1-2.log`,
   `…-unstable-explicit-replant.log`): 25 KILLED 2/2, 9 SURVIVED 2/2 (P2-β,
   P3-c′, and the three informational), control SURVIVED 2/2; per-site
   extension plants 17 × 2; per-site wrapper runs 7. The brief's four named
   classes: raw_complete ignoring one unknown item KILLED; always-excluded set
   missing a class KILLED (rev2 machine_auth + producer credential); equal-key
   duplicate at Build KILLED (raw `N-raw-order`, capture
   `R-capture-items-dup-build`) and at Decode KILLED for capture items /
   SURVIVED for raw entries (P2-β); sanitizer admitting one control character
   KILLED for U+001F and U+0085. Every harness restored the tree (diff-clean
   between the two copies after each battery).
6. **Race gate** — see the next section.
7. **Hygiene** — `gofmt -l` empty; `go vet` clean; `go build ./...` OK;
   `go test ./internal/clonebundle ./internal/provhost ./internal/localstore
   ./internal/canonicaljson -count=1` all ok (`logs/four-packages.log`);
   `go run ./internal/traceability/cmd/tracecheck` exit 0 (bindings=65,
   clauses 49/535 — unchanged, `internal/traceability` untouched);
   `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` exit
   0 on the exact tree; `git diff --check` clean; changed paths are exactly
   the 21 candidate paths (nothing outside `internal/clonebundle`, README,
   LOGBOOK; no `__pycache__`/`.pyc`); nothing imports `clonebundle`. README
   section has the three test commands and states "no `ax` command, no
   `doctor` result, and no runtime capability claim". LOGBOOK entry is
   newest-first under the 2026-09-17 heading. Sibling bounds (per-kind payload
   registries, `maximal_safe` planning, G2 admission, §7.8 wiring, traceability
   re-pin) are stated in doc.go, TRACEABILITY.md, results.md. Package
   coverage 83.9% statements. The conformance matrix's 29 named tests and all
   named call sites exist (owner entries `CheckRequestBody`, `CheckSuccessBody`,
   `CheckFreshSink` in sessadapter; the four
   `rejectUnsupportedImmutableObjectShape` rows in canonicaljson).

## Race gate (item 7)

The package is untouched by this change: no file outside
`internal/clonebundle`, `README.md`, `LOGBOOK.md` changed; `go list -deps` and
`-test -deps ./internal/sessquery` do not include `clonebundle`, and nothing
imports it, so the sessquery race binary is the trunk binary. The rev3 Change
Request validation ran the configured 27-command suite with
`go test ./... -race -count=1 -timeout 25m` and reports
`required=27 green=27 failed=0 missing=0`; the producer's own cmd05 shows
`ok internal/sessquery 665.226s` under the 25 m timeout. Disposition unchanged
from rev2: a host-capacity artifact, not a candidate defect; the 25 m timeout
landed on trunk (`888ae3d`, TASK-260917-3f8mkd) is the right orchestration
fix, and the honest disclosure was the right call.

My own measurement (`logs/sessquery-race.log`): `go test ./internal/sessquery
-race -count=1 -timeout 25m` on the pristine candidate copy → `ok … 347.359s`,
wall 349.6 s, exit 0, zero `DATA RACE` lines; host load averages 14.06 → 15.80
over the run with two other `.test` binaries alive at the start (this is a
shared host running other sessions' suites and my own mutation batteries; an
idle host was not available). That is under the old 600 s default even under
that contention; the producer's 665 s was measured while the full 27-command
suite ran concurrently. Two measurements under different load (rev2: 390.6 s
at load 14→10; rev3: 347.4 s at load 14→16) put the idle figure well under the
limit; I did not locate the STORY-260830-1oqfec 96 s figure on the board and do
not claim it.

## Rework scope (for the producer)

1. P1-c: validate non-message payload values (number model + nested
   duplicates, depth) at Build and Decode with the existing walker; negatives
   at both entries for a non-message kind, assert the sealed bytes are never
   rounded/collapsed; narrowing mutant.
2. P2-α: one extension value-fault negative per admission site (16 Decode
   sites + the two payload-level sites), harness rows.
3. P2-β: decode duplicate-entry row for the raw manifest + harness row.
4. P2-γ: refuse duplicate plan keys in `checkPlanItemMatch`; negative +
   narrowing mutant.
5. P3: null-content XOR; the three misreporting messages; rows for
   `input_blocked`, `-(2^53)`, U+2029, payload top-level duplicate, decode
   external-null-parent; `EncodeNativeIdentity` contract; `N-main-count` note;
   dead `sourceBasisObject`; state or cover the census arms in P3-g′.

Checklist state left by this review: the four reviewer rows (Implementation
matches AC; Solution fits project architecture; Tests green; Gate … attacked,
not read) are unchecked; "If review does not accept the work — verdict evidence
added and status routed by the explicit verdict branches" is checked. Status
routed to `to-dev`.
