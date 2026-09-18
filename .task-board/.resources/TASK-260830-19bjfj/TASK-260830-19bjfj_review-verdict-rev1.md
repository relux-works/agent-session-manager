# TASK-260830-19bjfj — independent review verdict, Change Request rev1

- Element: TASK-260830-19bjfj (fuzz-rpc-operations-and-version-skew), FINAL leaf of STORY-260830-4qojoz (mesh-rpc-framing-and-negotiation), EPIC M2
- Change Request: `CR-TASK-260830-19bjfj-1` revision 1, kind `story_final`, state `ready`
- Base OID `888ae3dff6a55c29bfaf311957748d405ed1ab06`; candidate tree OID `9435c374808fc64c6281e5c59cc45cc0690e7ea8`; branch tip `e390d5b` (checkpoint of TASK-260830-219okr rev4); patch sha256 `bc4c8dde…0676` (recomputed from `git diff base..tree`, byte-identical to the resource)
- Producer run RUN-260917-cea647 (muse-spark max); reviewer run RUN-260917-81d7ae (claude-opus-5 max)
- Authority: `internal/specdoc/SPEC.v0.7.0.md` §11.2, §11.3, §17.1–17.4 (trunk 2fc6d50). §11.2, §11.3, §17.2–17.4 are byte-identical to the v0.6.0 text the candidate cites; §17.1 differs only in its release-description paragraph (normative rules identical). Verified by section-scoped sha256 (`logs/` in the evidence tar).
- Method: every instrument ran against an immutable extract of the CR tree (`git archive 9435c37`) under `.temp/TASK-260830-19bjfj/review/cand`; the live Story worktree, index, branch and HEAD were never mutated; `PYTHONDONTWRITEBYTECODE=1`, no `__pycache__`/`.pyc` anywhere (scanned).

## Verdict: CHANGES REQUESTED → `to-dev`

The technical body of the leaf is strong — the three inherited advisories are genuinely closed, every producer mutant reproduces, my own 38 narrowing plants die twice, all three fuzz targets fail on their controls, and the AC coverage the producer reports (23 of 29 rows driven) is the ratio I measured. What blocks acceptance is the story-close half of a story_final Change Request: the candidate was built on the stale base 888ae3d and binds the **v0.6.0** ownership registry although trunk (2fc6d50, PR #52 / 32ee605) adopted **v0.7.0** before the producer was spawned — exactly the case the producer brief told this leaf to refresh for first. As it stands, the whole-Story patch does not apply on trunk and its registry bindings, digest re-pin, README coverage figures and LOGBOOK ratios all describe a registry the adopted tree no longer reads. Nothing here can integrate.

## Findings

### P1-1 — story_final candidate built on the stale base; registry work bound to the no-longer-adopted v0.6.0 registry

- Board facts: workspace `WS-5bf73e12e171` reports `current_base_oid=888ae3d`, `selected_base_oid=upstream_oid=2fc6d50`. `git merge-base --is-ancestor` confirms 888ae3d and 2fc6d50 are both on origin/main, with the v0.7.0 adoption (`internal/specdoc/SPEC.v0.7.0.md`, `internal/specpin/v0.7.0.lock.json`, `internal/catalog/catalog.v0.7.0.json`, `internal/traceability/ownership.v0.7.0.json`, `traceability.go` re-pointed to v0.7.0 with `reviewedOwnershipCanonicalSHA256 = c4cd46bf…`, and the new `internal/traceability/cmd/tracecheck/readme_coverage_pin_test.go`) landed in between.
- The producer brief's TRUNK STATE paragraph named 2fc6d50 and instructed `task-board worktree refresh-candidate TASK-260830-19bjfj` EARLY, binding `ownership.v0.7.0.json`. The candidate never refreshed; `TASK-260830-19bjfj_results.md` states "v0.7.0 has not landed (`internal/specdoc` holds v0.6.0 only)" — a stale-worktree observation presented as a fact about trunk.
- Measured consequences (`logs/trunk-apply-check-01.log`): `git apply --check` of the CR patch on an extract of 2fc6d50 fails on 5 paths — `LOGBOOK.md`, `README.md`, `internal/traceability/cmd/tracecheck/main_test.go`, `internal/traceability/traceability.go`, `internal/traceability/traceability_test.go` (3-way is impossible: `repository lacks the necessary blob`). These are exactly the 5 paths where `git diff --name-only 888ae3d 2fc6d50` overlaps the candidate.
- The registry deliverable is invisible on trunk: the six new acceptance cases and the four upgraded bindings live in `ownership.v0.6.0.json`; trunk's `tracecheck` reads `ownership.v0.7.0.json`. The clause `line` coordinates written into the bindings (11.2#2 → 7179, 11.2#3 → 7223, 11.2#4 → 7235, 11.3#18 → 7549, 17.1#1/#4/#5 → 15146/15168/15181, 17.4#4 → 15240) are v0.6.0 line numbers; in v0.7.0 the same sentences sit at 7364/…/7734/…/15589 (measured by grep). The re-derived pin `1500e259…` is the correct projection of the candidate's v0.6.0 registry (I re-derived it through an overlay with a zeroed pin: `logs/digest-derivation-01.log`) — but it pins a registry trunk does not adopt.
- README: the "Measured coverage" subsection now reads `bindings=65 … clauses_discharged=57/535` and "138 executable acceptance cases … [ownership.v0.6.0.json]"; trunk's subsection reads `bindings=68 … 49/569`, 135 cases, `ownership.v0.7.0.json`, and trunk's `readme_coverage_pin_test.go` pins that subsection to the tree's own tracecheck output through a closed number-word map (`two, four, five, six, seven, fifty-two, sixty-eight`) — the candidate's "eight are partial / six are sliver / forty-five are unevidenced" prose cannot pass it without a deliberate update of that test's literals.
- Consequence for the DoD: story-close items (a) registry bindings, (b) digest re-pin, (c) tracecheck ratios, (d) README coverage subsection and (e) LOGBOOK ratios are not delivered against the adopted registry; "README/doctor/capability evidence and specification traceability are updated without unsupported claims" is unmet for the tree that would actually integrate.

### P2-1 — registry binding `11.3#18` is discharged by tests that do not drive the clause

- Clause 11.3#18 (SPEC line 7549 v0.6.0 / 7734 v0.7.0): "Arrays with digest identities MUST be sorted bytewise and contain no duplicate." The candidate binds it to acceptance case `rpcwire-inventory-arrays` (`TestInventoryNamespacesAndCardinality`, `TestNamespaceVocabularyPerMember`, `TestNamespaceMismatchMatrix`) with the gap text "sorted-unique digest arrays, inventory.roots lanes".
- No production code in `internal/rpcwire` parses an array of digest identities: `decodeNamespaces` validates a sorted-unique array of vocabulary strings, `validateRoots` reads one `root_id` digest per root and matches roots to the request's namespace order (`internal/rpcwire/inventory.go`). None of the three tests feeds a digest array. The clause is about `object_ids`, `known_lease_record_ids`, `installed_object_ids`, `manifest_ids`, `VerifiedState.object_ids` — all inside bodies the producer correctly lists as opaque stated bounds.
- This is an invented clause binding; the producer brief says "never invent a binding for a clause no test drives". Honest disposition: 11.3 stays `unevidenced 0/27` (the namespace sorted-unique rule is a §11.3 table-row constraint, not clause #18), which also moves the headline `clauses_discharged` figure. Filed P2 because it is a traceability claim that inflates the measured ratio the README and LOGBOOK restate.

### P3-1 — opaque-body object gate unpinned (reviewer plant R9 SURVIVED ×2)

- `DecodeRequest` refuses a body that is not an I-JSON object (`object(m["body"])`: non-object JSON, duplicate members, non-integral numbers) for every operation. For opaque operations no test witnesses that refusal: the plant `err != nil && op != "health.get"` (admits a `[]`/`null`/`"x"`/duplicate-member body for `health.get` only) survives the committed suite twice. `TestRequestRefusals/array-body` drives it on a hello (where `decodeHello` refuses anyway) and `TestOpaqueOperationBoundary` only sends invalid JSON (`{`).
- Both new fuzzers normalise bodies with `validObjectOrEmpty` (a non-object body becomes `{}`), so neither target can reach this arm either. Add a negative over an opaque operation (`[]`, `null`, `"x"`, `42`, `{"a":1,"a":2}` → `ErrFrame`) and let `FuzzClosedOperationBodies` feed non-object bodies.

### P3-2 — changed-body retry after a committed effect is unpinned (reviewer plant J1 SURVIVED ×2)

- `matjournal.replayCreateLocked` refuses a moved body at two sites (journal digest, receipt digest). A two-site narrowing that exempts only `journal.Phase == PhaseCommitted` survives twice: `TestCreateReplayAndConflict` pins `idempotency_mismatch` at staging, and the new `TestRPCBlindRetryAfterCommitNeverDoubles` retries the identical body only. Matrix rows 19/20 claim "changed body is idempotency_mismatch" and "blind retry after a committed effect never doubles it"; the combination (changed body after commit — the spec's "A changed canonical body … is idempotency_mismatch" has no phase qualifier) is not driven. Add `Create(moved)` after `setupPhase(t, store, PhaseCommitted)` → conflict naming the literal `idempotency_mismatch`, single journal directory retained.
- The reference arm-delete J2 (both sites unconditional) is KILLED ×2 by `TestCreateReplayAndConflict`, so the gate exists; only the committed-phase member of the class is unmeasured.

### P3-3 — the body fuzzer's oracle is member-name-only; the unknown-field fuzzer has no hello-body seed

- `FuzzClosedOperationBodies` re-uses the (possibly planted) `r.Hello()` and checks only the exact key set. A `len(b) >= 15` nonce plant passes the target at `-fuzztime=100x` (`reviewer-fuzz/reviewer-bodies-nonce-15`, exit 0) while the unit suite kills the same plant (`nonce-short`). The README/LOGBOOK sentence "reach the `hello`/`inventory.roots` body parsers rather than only the envelope" is true for reach and silent about the oracle's bound; state the bound (bounds and vocabularies are owned by the unit tests) or add an independent oracle for the hello bounds.
- `FuzzUnknownFields` carries no seed with an unknown member inside a hello body; a hello-body-extra plant passes at 100x (`reviewer-fuzz/reviewer-unknownfields-hello-body-extra`, exit 0) although the oracle would catch it if reached. Add the seed. (An inventory-body-extra plant and a class-level hello-extra plant on the bodies target FAIL as expected; the producer's three controls reproduce.)

### P3-4 — `TestFramingBounds/encode-refuses-8388609` is not an edge witness

- The encode lane pads a body with `MaxLineBytes` characters, so the encoded line is ~8.4 MB plus the envelope — far past 8388609. The decode lanes are exact (8388608 admitted, 8388609 refused, measured). Matrix row 21 says "both directions, literal byte counts". Either construct an exact 8388609-byte encode or state the encode direction as subsumed (`EncodeRequest` calls `DecodeRequest`).

### P3-5 — `TestErrorSchemaNotNegotiated` refusal lanes pin no class

- The three refusal lanes assert only `err == nil → Fatal`; any error (including a malformed-document `ErrFrame`) would pass. `bound-versions-admit` proves the builder is well-formed, so the refusal is real today, but pin the class returned through `axerror.DecodeBound` so a shape error cannot stand in for the binding refusal.

### P3-6 — production-derived expectations in the meshneg adversarial suite

- `TestPeerOfferKeyMembershipPerKey` and `TestNegotiateOfferRefusalCarriesLocal` compute `wantLocal` from `meshneg.LocalMajors(...)` and compare `Refusal.Reason` against `meshneg.Reason*` constants. The literal spec class (`incompatible_protocol`/6) is asserted, and `TestLocalMajors` pins the offer table literally elsewhere, so this is the mild form; still, assert the literal offer sets (`[2 3 4]`, `[5]`) in the N2 rows rather than the production function.

### P3-7 — `nonce-admits-standard-alphabet` measures the fixture, not the class

- The plant admits exactly the string `"+/v7+/v7+/v7+/v7+/v7+/v7"`; it proves the test pushes that string through `validNonce`, not that the standard alphabet is refused as a class. My R1 (`nonce-admits-padded`, strips every trailing `=` before the strict decode) is a class narrowing and is KILLED ×2 by `nonce-padded-refused`; do the same for the alphabet (translate `+`/`/` to `-`/`_` before decoding).

### P3-8 — evidence hygiene

- The validation table claims exit 0 for gofmt, build, vet, cataloggen `-check`, both cross-builds and the JSON census with no raw log attached for any of them (I re-ran all: exit 0, `logs/validation-misc-01.log`). The constructed `_rev1-validation.log` resource is truncated at exactly 65536 bytes.
- Matrix row 22 cites "truncated pre-hello frame" for `TestHostileDisconnectPhases/mid_request_close`, which is a post-hello half request; the pre-hello lane is `hello_half_frame_close`. Fix the citation.

## What I verified and what holds

### (0) Inherited advisories — all three CLOSED

| Item | Reviewer instrument | Result |
| --- | --- | --- |
| N1 key-membership gate per key | 25 plants `!ok && key != "<K>"` over every key of every profile, committed suite, two runs | 24 of 24 non-rpc keys KILLED ×2 by `TestPeerOfferKeyMembershipPerKey/substituted-<frame>-<key>` (the 88-key × {missing, substituted} sweep derived from `rpcwire.ContractProfile(frame)` through `Negotiate`); `rpc` SURVIVED ×2 as an equivalent mutant (subsumed by the `len(offered) < 1` arm, as the predecessor recorded) |
| N2 `Refusal.Local` on offer refusals | S12 `refusal.Local = nil` | KILLED ×2 (183 failing subtests: every N1 row plus `TestNegotiateOfferRefusalCarriesLocal`) |
| N3 local-side vocabulary member 1 | S14 `major < 1` on the local arm | KILLED ×2 by `TestSelectHighestCommon/refuse-local-names-1`; reachability bound (`LocalMajors` never yields 1) stated in TRACEABILITY.md |

### (1) Spec fidelity

- §11.3 operation table extracted: 24 operations. Only `hello` (§11.2) and `inventory.roots` bodies are parsed anywhere in non-test Go (`grep` census over `internal/` for every operation literal hits only `internal/catalog/catalog_gen.go`, a registry; `initiator_host_id`/`expected_lease` parsers: none; `hostchannel.serveLoop` hands every post-hello request to a consumer `Handler` with no replay cache). The 22-operation stated bound, the embedded-type bound and the initiator/lease revalidation bound are honest.
- hello member set: 7 missing + 7 wrong-type + 1 extra, all `ErrHello` through `DecodeRequest` (`TestClosedBodyMemberSweep`); bounds at both edges and one outside: nonce 16/15 bytes, contracts arrays 16/17 (and 1/0 via trunk `empty-versions`), `max_line_bytes` 8388608/8388607, `max_object_bytes` 5242880/5242879, each floor refusing while the other is generous (`TestClosedBodyBoundsWitnessed`); `platform`/`host_id`/`ax_version` vocabularies via trunk `TestHelloRefusals` (mutants `hello-platform`, `hello-host`, `hello-version` killed). inventory.roots: 1 admitted, 6/7/8 per major admitted, 0 refused, 7-in-v2 unreachable (my R8 survives as documented equivalent: sorted-unique members of a 6-vocabulary cannot number 7).
- Namespace vocabulary per major matches §11.3/§11.8/§11.9 (`{1..6}`, `[1..7]`, `[1..8]`); near-miss spellings and future-major members refused; 30-cell mismatch matrix through `EncodeSuccess`.

### (2) Fuzz targets — instruments attacked, not read

- The three `task-board.config.json` commands executed verbatim by me on the candidate copy: exit 0 each (`reviewer-fuzz/config-*.log`). Config diff is exactly the 3 added rows in the existing shape; byte-identical elsewhere.
- Producer controls reproduced through `-overlay` with their own `planted.go`: bodies FAIL (`admitted hello body is not the exact member set`), nsvocab FAIL (`invalid 4.0.0 namespaces admitted: credential`), unknownfields FAIL (`admitted envelope carries 6 members`).
- Reviewer controls: hello-extra (class) FAIL, nsvocab duplicate FAIL, nsvocab future-major-in-v2 FAIL, inventory-body-extra FAIL; nonce-15 PASS and hello-body-extra-seed PASS (P3-3).
- Seeds are in-code (`f.Add`), targets deterministic, no network; no `testdata/fuzz` corpus committed (the removed oracle-bug corpus file is indeed absent from the tree).

### (3) Section 17 — driven

- Both extension directions (`TestExtensionsDirections`: envelope top level `ErrFrame`, closed bodies `ErrHello`/`ErrInventory`, opaque body byte-identical); no-coercion shapes through `Negotiate` with the literal `incompatible_protocol`/6 (only-higher, only-lower, empty, duplicate, unsorted, 17-count); a 2.0.0 peer rejects major 1 at every rpcwire entry and both meshneg lanes; the error schema is not negotiated (RPC 2 refuses an Error 1.3.0 document, RPC 4/5 refuse 1.0.0, bound pairs admit; an `error` key in the hello map is refused by trunk `error-key` and the meshneg shape gate). My coercion plants: `major != framing → major > framing` (admits a lower major in the frame) KILLED ×2 by `TestMajor1RejectedThroughNegotiation/rpc-1.0.0-in-v2-frame` and `lower-major-in-v3`; unsorted-first-pair KILLED ×2; `select-highest-skips-5` KILLED ×2.

### (4) Replay and lost response

- Mismatched echo (ID → `ErrCorrelation`, version → `ErrVersion`, nonce echo → `ErrCorrelation`, zero expectation → `ErrCorrelation`); my zero-expectation narrowing KILLED ×2. Envelope `request_id` is correlation-only (no responder cache exists — honest bound). Operation-key replay adopted from `matjournal` (`TestCreateReplayAndConflict`: byte-identical replay, literal `idempotency_mismatch`); `materialize.status` recovery adopted as `Store.Get` returning the true committed phase with a single journal and a byte-identical blind-retry replay (`TestRPCBlindRetryAfterCommitNeverDoubles`, in-package). Gap: P3-2.

### (5) Framing

- 8388608 admitted / 8388609 refused on decode (exact byte counts asserted), embedded LF refused (my end-LF narrowing KILLED ×2), valid non-object JSON lines refused (`null`, `[]`, string, number, bool), truncated post-hello frame adopted from hostchannel, unpadded base64url admitted / padded and standard-alphabet refused at the nonce gate (my padded-class narrowing KILLED ×2), `max_object_bytes` enforced independently of `max_line_bytes`. Chunk-data base64url is an honest bound (no chunk validator exists). Gaps: P3-4 (encode edge), P3-7.

### (6) Story-close items

- `go run ./internal/traceability/cmd/tracecheck` on the candidate copy prints exactly `traceability ok: contracts=63 normative_sections=36 acceptance_cases=138 fixtures=32 compatibility_contracts=55 assigned_scopes=0` / `section coverage: bindings=65 full=2 partial=8 sliver=6 unevidenced=45 unmeasured=4 unowned=7 clauses_discharged=57/535`; section runs refuse with `11.2 3/5 partial`, `11.3 1/27 sliver`, `17.1 3/6 partial`, `17.4 1/4 sliver` — verbatim what the results quote. Digest `1500e259…` independently re-derived. README figures equal this output; LOGBOOK entry is newest-first and factually true for the candidate tree. All of it is against the wrong registry (P1-1) and one binding is invented (P2-1).

### (7) Mutation

- Producer harnesses re-executed twice each in the isolated copy: rpcwire 42 rows (40 KILLED by their named tests, `neutral` passed, `error-text` control SURVIVED, exit 0) and meshneg 39 rows (37 KILLED, `neutral` passed, `detail-text` SURVIVED, exit 0) — results identical to the producer's `results.json` in both runs.
- Reviewer harness `review_mutants_rev1.py` (45 rows, two full runs, per-plant raw `test.log`, overlays and planted sources under `reviewer-mutants/<name>/run{1,2}/`): 38 KILLED ×2 (25 inherited-advisory plants incl. the 24-key sweep, R1–R7, R10, M1–M3, J2), 4 SURVIVED ×2 (`N1-rpc` and `R8` documented equivalents; `R9` and `J1` findings), 2 harmless controls SURVIVED ×2 (`C1b` comment, `C2` comment), 1 invalid control (`C1`, typed constant → COMPILE_FAIL, superseded by `C1b` and reported as such, not as a survivor). Of the brief's four suggested shapes: member check dropped (R2 root item, R3 inventory body — both KILLED), bound widened by one (R8 — equivalent, documented; R1 padded class KILLED), unsorted array admitted (R4, R5, M2 — KILLED), pre-mutation revalidation skipped — no such entry exists in the landed code (stated bound); its nearest analogue in the adopted replay owner is J1/J2 (P3-2).

### (8) Bounds and hygiene

- gofmt clean, `go build ./...`, `go vet ./...`, `go test ./... -count=1` (38 ok, exit 0), `-race -cover` for rpcwire/meshneg/matjournal/hostchannel (ok, no DATA RACE; coverage 99.1 % / 87.8 % / 77.0 % / 84.1 %), cataloggen `-adopted … -check` exit 0, GOOS=linux/windows builds exit 0, JSON census over the tree exit 0 — all run by me on the candidate copy. The full `-race` run over all 38 packages was accepted from the producer's five chunked logs (38 distinct packages `ok`, zero `DATA RACE`); the CR construction log records the control root's 27 commands green.
- No deletions, renames or mode changes in the CR; no `__pycache__`/`.pyc`; the incident-restored files (`internal/matjournal/journal_test.go`, canonicaljson/scalar corpora) are byte-identical to the base. `task-board.config.json` is trunk + 3 rows. README/LOGBOOK edits are additive on the CANDIDATE's base — but that base is not trunk (P1-1).

## Coverage ratio measured

23 of 29 acceptance rows driven through a production entry by a named committed test, 6 stated bounds — the producer's count holds. Two driven rows carry an unwitnessed sub-case (row 19/20: changed body after commit, P3-2; row 21: encode edge, P3-4), and one registry clause claim (11.3#18) is not driven (P2-1).

## Evidence index (`TASK-260830-19bjfj_review-evidence-rev1.tar.gz`)

- `review_mutants_rev1.py`, `reviewer-mutants/{table.md,results-run1.json,results-run2.json,<plant>/run{1,2}/{test.log,overlay.json,<planted source>}}`
- `review_fuzz_rev1.py`, `reviewer-fuzz/{results.json,config-*.log,<control>/{fuzz.log,overlay.json,<planted source>}}`
- `producer-harness-rerun/{rpcwire,meshneg}-run{1,2}/{results.json,table.md,<probe>/test.log}`
- `logs/`: `pkg-tests-01.log`, `test-all-01.log`, `race-scoped-01.log`, `validation-misc-01.log`, `tracecheck-cand-01.log`, `tracecheck-cand-sections-01.log`, `digest-derivation-01.log`, `trunk-apply-check-01.log`, `reviewer-mutants-01.log`, `reviewer-fuzz-01.log`, `producer-harness-*.log`, `spec-section-digests.txt`
- `digest-derive/` (zeroed-pin overlay used for the derivation)

## Rework scope (for the producer)

1. Run `task-board worktree refresh-candidate TASK-260830-19bjfj` (checkpoint-bound `--replay-resolutions` path for LOGBOOK.md; the original checkpoint OID is `e390d5b263745e3af1f5a380940cebec8b1154b3`), then audit every path in `git diff --name-only 888ae3d 2fc6d50` that also appears in `git status` and restore trunk content by hand: `README.md` and `LOGBOOK.md` purely additive on the NEW trunk content, `internal/traceability/traceability.go` = trunk except the re-derived pin, `internal/traceability/traceability_test.go` and `cmd/tracecheck/main_test.go` = trunk plus the deliberate figure updates, `task-board.config.json` = trunk + the 3 fuzz rows. Revert every edit to `internal/traceability/ownership.v0.6.0.json`.
2. Re-create the bindings in `internal/traceability/ownership.v0.7.0.json` (six acceptance cases; 11.2, 17.1, 17.4 upgrades) with v0.7.0 clause coordinates; drop the 11.3#18 discharge (P2-1) and leave 11.3 at its honest unevidenced disposition unless a clause is actually driven. Re-derive `reviewedOwnershipCanonicalSHA256` from the v0.7.0 registry and show the derivation. Re-run `tracecheck` and the four section-scoped runs; quote the new ratios verbatim.
3. Update the README "Measured coverage of this repository" subsection to the new printed numbers and keep trunk's `readme_coverage_pin_test.go` green (its closed number-word map and literal regexes need a deliberate update); one newest-first LOGBOOK entry on top of the trunk entries; cite `SPEC.v0.7.0.md`.
4. Close the P3 test gaps: opaque-body non-object/duplicate-member negative and non-object bodies in the body fuzzer (P3-1); `Create(moved)` after commit → literal `idempotency_mismatch` (P3-2); hello-body unknown-member seed for `FuzzUnknownFields` and a stated oracle bound or independent bound oracle for `FuzzClosedOperationBodies` (P3-3); exact 8388609-byte encode witness or a stated subsumption (P3-4); pin the refusal class in `TestErrorSchemaNotNegotiated` (P3-5); literal offer sets in the N2 rows (P3-6); class-level alphabet narrowing (P3-7); attach raw logs for every validation row and fix the row-22 citation (P3-8).
5. Re-run the full configured suite (now 30 commands) on the refreshed tree and attach the logs; leave the candidate UNCOMMITTED for the handoff snapshot.
