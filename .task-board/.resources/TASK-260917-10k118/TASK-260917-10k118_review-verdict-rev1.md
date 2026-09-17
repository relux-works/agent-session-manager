# TASK-260917-10k118 — independent review verdict, CR-TASK-260917-10k118-1 revision 1

Reviewer run: RUN-260917-fa6096 (claude-opus-5 max, reviewer/reviewer).
Change Request: `CR-TASK-260917-10k118-1` rev 1, kind `story_final`, integration
scope STORY-260917-110onn, base `7efe3854a5a14545f9fa2b7697accb1883249f1f`,
candidate tree `8903e2d85230cf488a91a7409ec427401a6da4c6`, 26 changed paths,
patch sha256 `07dacbe188fd0220882fb9509373966ae34e43487c9b932655b33ff082d6f280`.

## Verdict: ACCEPT

The carried v0.7.0 adoption delta was verified as a first review on this base with
my own instruments (fresh upstream clone, own §1.5 table extractor, own section
inventory extractor, own clause-line re-measurer with three control plants, own
canonical-digest mirror for the ownership registry, own binding-by-binding carry
audit against `git show 7efe385:internal/traceability/ownership.v0.6.0.json`,
own mutation battery on an isolated copy). The two new verification tests do what
the Story owes. No P1 or P2 finding. Three P3 observations are recorded below for
follow-up tasks; none changes the reviewed bytes of this candidate, whose AC is
explicitly "the union of the two accepted revisions plus the two verification
tests".

Method: isolated immutable probes only. The live Story worktree, index, branch and
HEAD were never mutated (one accidental `git add -N` on the live index during the
first tree-OID computation was reverted immediately with `git reset -- <paths>`;
`git diff --cached` is empty and `git status` is unchanged). Every measurement
below was taken on `/tmp/rev-10k118/cand`, a `git archive` export of the exact
candidate tree (its `git write-tree` = `8903e2d8…`), or on throwaway copies of it.
Scratch copies are deleted at the end of this run; durable evidence lives in
`.temp/TASK-260917-10k118/review/` and in the attached
`TASK-260917-10k118_review-evidence-rev1.tar.gz`.

## 1. Story diff, base, checkpoint, hygiene

| Check | Result |
| --- | --- |
| Working tree == candidate | temp-index `git write-tree` of the live worktree = `8903e2d85230cf488a91a7409ec427401a6da4c6` (CR candidate). Index tree = `7cb803a7…` = HEAD tree (nothing staged). |
| Patch resource | sha256 matches the record; `git apply --binary` onto a fresh export of `7efe385` reproduces tree `8903e2d8…` exactly. |
| Base / checkpoint | Story branch tip = checkpoint_oid = base = `7efe385` (no checkpoint commit in this Story; the candidate IS the whole Story delta). `git verify-commit 7efe385`: Good ECDSA signature, oparin@me.com. |
| Changed paths | exactly the 26 recorded paths; `task-board.config.json` NOT among them (blob `c764a06…` == base); no `__pycache__`, `.pyc`, probes or plants; `.task-board/**` untouched. |
| Historical byte-identity vs `7efe385` | IDENTICAL: ownership.v0.5.0.json (`039a0301`), ownership.v0.6.0.json (`b4f24807`, sha256 `212ae321…`), catalog.v0.5.0.json, catalog.v0.6.0.json, v0.5.0.lock.json, v0.6.0.lock.json, SPEC.md (v0.5.0), SPEC.v0.6.0.md, adoption-v0.6.0.md, task-board.config.json, .github/workflows/ci.yml. |
| Lost trunk code | per-file top-level function census base→candidate: 0 functions lost in any of the 12 modified Go files except the intended renames (`TestMainRejectsSyntacticallyValidNonexistentV060Section`→`…V070Section`, `expandV060SectionRange`→`expandV070SectionRange`, `TestCurrentMatchesReviewedV060Catalog`→`…V070Catalog` + new `TestV060ProjectionMatchesHistoricalLock`). |
| LOGBOOK | purely additive (0 `-` lines): TASK-260917-10k118 entry at top, TASK-260916-n9r71p entry below it, TASK-260916-2yzf5d entry at its replay position inside a 2026-09-17 section. |
| README | only adoption rewordings (every `-` line has its `+` rewording in the same hunk); explicit non-claim sentence for launch-plan / `SpawnPlan` stdin / `caller_launch_plan` / `stdin_resume_replay` / `environment_drift` / curator-run naming present at lines 96–99 and 3236–3238. |

## 2. Trunk drift since the base and the validation-suite question (brief §precedent)

`origin/main` is now `888ae3dff6a55c29bfaf311957748d405ed1ab06` (one commit past
the base, pushed 2026-09-17T17:56Z, 19 minutes after the worktree was
provisioned): `task-board.config.json` only, command 5 becomes
`go test ./... -race -count=1 -timeout 25m`. The candidate does not touch that
path (its config blob equals the base blob), so the path sets are disjoint.

Will `validation_suite_changed` recur? I recomputed the suite identity the way
`remoteconfig.WorktreeValidationConfig.SuiteSHA256` does (sha256 of
`json.Marshal(commands)`):

| Config | Suite sha256 | Command 5 |
| --- | --- | --- |
| base `7efe385` | `689a4b23…e4d7eb` | `go test ./... -race -count=1` |
| trunk `888ae3d` | `e068770a…d8a536` | `go test ./... -race -count=1 -timeout 25m` |
| **CR record `validation.suite_sha256`** | **`e068770a8270c57fd3960af45970b8d3c5fc511fa28c42a62cc25ca7e8d8a536`** | (27 commands, exit 0, tree `8903e2d8…`, completed 2026-09-17T19:10:53Z) |

The CR was constructed with the control root's CURRENT suite (trunk `888ae3d`), so
`selectReviewedValidationSuite` sees `record.Validation.SuiteSHA256 ==
s.req.ValidationSuiteSHA` and returns nil: the refusal will NOT recur as long as
trunk's suite is still `e068770a…` at integrate time. The base drift itself goes
through the ordinary disjoint-path reparent + revalidation. The reviewed candidate
config string for command 22 is
`go run ./internal/catalog/cmd/cataloggen -adopted -output internal/catalog/catalog_gen.go -check`
(trunk-owned, unchanged by the candidate; command 21 is `tracecheck`).

## 3. Specification pin (carried 2yzf5d rev1, verified as a first review)

| Check | Result |
| --- | --- |
| Fresh read-only clone `https://github.com/relux-works/agent-session-manager-spec.git` | `git cat-file -t v0.7.0` = `tag`; tag object `d4abe46fb12d9ba347c09f43efb01089530795b3`; peeled `32b3f2ba7c377248a53cd42389abbd2f1c321834`; `git tag -v v0.7.0` → Good "git" signature for oparin@me.com, ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`, tagger Ivan Oparin. |
| SPEC.md at the peeled commit | blob `c4903e60ea72583b507060182aee1b11e21ddd86`, 1185291 bytes, 0 CR bytes, sha256 `c6b2fe64ee79ed697a96ed27a1679c80b8ee1eba137feb99e7738b4da289ddcf` raw == LF-normalized; `cmp` IDENTICAL to `internal/specdoc/SPEC.v0.7.0.md`. |
| Six fixture digests | all six lock rows equal sha256 of the fixtures at the peeled commit (session_directory `a6351a83…`, terminal_backend `67de0d78…`, roadmap `6023ec0d…`, host_channel `0d6529a8…`, session_selector `f0f1cb2d…`, launch_plan_request `ec4310b8…`). |
| §1.5 contract table vs `v0.7.0.lock.json` (own extractor) | 64/64 rows, same order, name/URN identical, version sets identical; one order normalization (Provider protocol `2.0.0,3.0.0,2.1.0,3.1.0` → sorted). Delta over `v0.6.0.lock.json`: +1 row (`Launch Plan request`, index 20 between Session record and Session event), 0 removals, exactly 5 widened lists (provider +2.1.0/+3.1.0, manifest +1.1.0, probe +1.1.0, session-record +3.1.0, error +1.5.0) — matching `v060VersionCeilings` and `v070OnlyContractKeys` in pin.go. |
| Section inventory (own extractor over SPEC.v0.7.0.md) | 170 identifiers, element-equal to `sectionInventoryV070`; digest `2ef110e0f87c5d174bceafb33a0db981f5dc74a384f8b599f12b87cb4af2eb94` == `SectionInventorySHA256V070` == the v0.6.0 digest (v0.7.0 adds no numbered heading). |
| History projections | `ContractsForRelease` v0.7.0→v0.6.0/v0.5.0/v0.4.3 proven by the committed pin070 tests (green on the tree); cigate derives 64 / 55 rows from both authorities. |

## 4. Catalog

| Check | Result |
| --- | --- |
| `catalog.v0.7.0.json` vs `catalog.v0.6.0.json` (semantic diff) | source block → v0.7.0/`32b3f2ba…`/`c6b2fe64…`; every family gains `v0.7.0` in `releases`; ONLY the provider operation family and provider capability family gain `2.1.0`/`3.1.0`; error catalog stays 1.0.0–1.3.0; self-identity session-record stays 1.0.0–3.0.0; no launch-plan family; no `caller_launch_plan` capability (7 provider capability names). |
| Raw metadata digest | sha256(catalog.v0.7.0.json) = `6d769c9b…db8193` == `// Metadata SHA-256` header of catalog_gen.go. |
| Reviewed canonical digest | `reviewedMetadataCanonicalSHA256 = c4094101…50274a35` enforced by `Generate` (`-adopted -check` exit 0 proves the shipped metadata projects to it). |
| Command 22 (`cataloggen -adopted … -check`) | exit 0; file untouched (copy `git status` clean after). Explicit v0.7.0 pair `-check`: exit 0. Explicit stale v0.6.0 pair: exit 1 `normative source pin mismatch: source identity drift`. |
| `catalog.Adopted == ReleaseV070`, `Current()` via `ForRelease(Adopted)` | read in catalog.go; `TestCurrentIsPinnedToAdoptedRelease` + `TestCurrentMatchesReviewedV070Catalog` green. |
| canonicaljson init panic (census-only rationale) | REPRODUCED: adding `"3.1.0"` to the session-record self-identity row in a copy of catalog_gen.go → `panic: immutable-object shape validator registry is invalid: missing immutable-object shape validator for urn:ax:schema:session-record@3.1.0` at `canonicaljson.init` (closed_shapes.go:152). |
| Is census-only faithful? | Production consumers of `catalog.Current()` are config/validation.go (Capabilities), canonicaljson (SelfIdentities, Events), cliresult (Capabilities → 7 provider names), axerror (Errors). None reads `.Contracts`; the only `.Contracts` readers are cigate/traceability (census gates). So `ForRelease/Current` advertise 3.1.0/1.5.0/1.1.0/launch-plan only in the contract census, and no consumer treats a census row as validator support. Provider protocol 2.1.0/3.1.0 add members, not operations/capabilities (SPEC 7.5), so carrying the same 15 operations / 7 names under the new labels is consistent; `provhost` still declares manifest/probe 1.0.0. |

## 5. Traceability

| Check | Result |
| --- | --- |
| Registry file digest | sha256(ownership.v0.7.0.json) = `2548870d896b0e119c307cdcba02ab23b36af7f473328a15f6a092090d8a5070` (the brief's expected digest). |
| Canonical projection digest (own Python mirror of the Go struct marshal) | `c4cd46bf71f164ad53dfe00ddd2e64e0b439e1c2684138fbcfe208f7e53473f6` == `reviewedOwnershipCanonicalSHA256`; the same mirror reproduces trunk's v0.6.0 pin `d3eca906…` from ownership.v0.6.0.json, so the mirror is calibrated. |
| Carry audit vs trunk v0.6.0 registry (own script) | acceptance cases 132→135: 0 dropped, +3 (`source-pin-v070-exact`, `source-pin-v070-refusal`, `catalog-v070-exact`); exactly 2 carried cases changed: `assigned-section-binding` (single V060→V070 test rename) and `catalog-v060-exact` (repointed to `ForRelease` / `TestV060ProjectionMatchesHistoricalLock`). Section bindings 65→68: 0 dropped, +`section:13.1`, +`section:13.10`, +`section:14.1` (all `unevidenced`, 0 clauses, gaps name STORY-260916-3fjjtn / 3aukw3 / 3fjjtn(+1ea7od split)); production/acceptance/coverage of all 65 carried bindings unchanged; exactly 5 gap rewordings (5.1→2q85nu, 7.3→1ea7od, 7.4→1ea7od, 7.5→1kp1lx, appendix-d→vucwn0 with the measured 16→18 clause count). Contract group +`Launch Plan request [urn:ax:schema:launch-plan-request]` (carried order preserved), cases +`catalog-v070-exact`; pin fixture group +`pin:ax-launch-plan-request-v1`, cases +v070 pair; source group cases → v070 pair; unowned: same 7 keys in order, only 11.10.5 reworded to "Disclosure owner: TASK-260830-2x16gz" (evidence unchanged). |
| Clause lines (own re-measurer: heading ownership via the same atx/numbered/appendix rules, RFC 2119 keyword scan, excerpt-begins-at-line with hard boundaries) | v0.7.0: 49/49 clauses OK; trunk v0.6.0 registry vs SPEC.v0.6.0.md: 49/49 OK; line shifts {+20:8, +79:9, +185:8, +281:9, +392:15}, **0 unshifted**. Instrument controls: a +1 line plant → 2 failures, an excerpt edit → 1 failure, all 49 lines left at v0.6.0 values → 98 failures. |
| Denominator | 535 → 569 = carried 65 bindings measured in the v0.7.0 document 542 (+7: 5.1 9→11, 7.5 53→56, Appendix D 16→18) + new bindings 13.1 (8) + 13.10 (3) + 14.1 (16) = 27; discharged stays 49. tracecheck prints exactly `bindings=68 full=2 partial=6 sliver=4 unevidenced=52 unmeasured=4 unowned=7 clauses_discharged=49/569`. |
| Owner Stories on the live board | 7/7 exist under EPIC-260916-18uelk, all `backlog`, titles match the adoption map (3fjjtn parsing/refusals, 2q85nu record embedding, 1kp1lx stdin/replay, 1ea7od plugin capability, 3aukw3 drift, 3ustb3 naming, vucwn0 vectors). |
| Runtime admission | `VerifyAssignedSections` refuses all six adopted sections with the "Pending implementation owner" gap (`TestAdoptedSectionsHavePendingOwnersAndRefuseRuntimeAdmission` 6/6 + 2 splits, green); mutant C2 below proves the refusal is live for 13.10 at the `tracecheck -section` entry. |
| Legacy projections | v0.6.0 and v0.5.0 are verified twice: inside `VerifyRepository` (lock-derived rows vs `catalog.ForRelease`, both compiled-in) and independently in `internal/catalog` (`TestV060ProjectionMatchesHistoricalLock` reads the real v0.6.0 lock bytes as an oracle and asserts the Launch Plan row is absent; `TestV050ProjectionMatchesHistoricalLock` likewise). |
| 11.10.5 wording | "Disclosure owner: TASK-260830-2x16gz" (2x16gz P3 applied). |

## 6. cigate

`PinnedReleases()` = `[v0.7.0 v0.4.3]`; `VerifyContractPreservation()` green;
pin/catalog derived sets 64 and 55 rows. Drift table (7 rows in
`TestCheckReleaseRootsRefusesDrift`) green. v0.7.1 plants: one-sided
`checkReleaseRoots("v0.7.1",…,"v0.7.0",…)` and `(…"v0.7.0"…,"v0.7.1"…)` refused as
root drift; `catalog.ForRelease("v0.7.1")` and `ContractsForRelease("v0.7.1")`
refused with `ErrUnsupportedRelease`; a both-sides `v0.7.1` re-point of
`PinnedReleases` passes the root-agreement check but is refused downstream by the
lock (`unsupported specification release: v0.7.1`), reddening
`TestVerifyContractPreservationLive` and `TestDerivedSetsAreComplete`.

## 7. Gates re-run by me on the exact tree (isolated export)

| Gate | Exit |
| --- | --- |
| gofmt over tracked Go files | 0 |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./internal/{specpin,specdoc,catalog/...,cataloggen,cigate,traceability/...} -count=1` | 0 (8 packages ok) |
| same six packages `-race -count=1` | 0 |
| `go test ./... -count=1` (whole repository) | 0 (37 packages ok, 0 FAIL) |
| `go run ./internal/traceability/cmd/tracecheck` | 0, `contracts=64 normative_sections=36 acceptance_cases=135 fixtures=33 compatibility_contracts=55 assigned_scopes=0` + the coverage line above |
| `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | 0 |
| explicit v0.6.0 `cataloggen … -check` | 1 (refused: source identity drift) |

Accepted from already-attached evidence: the CR construction record (27/27
commands exit 0 on tree `8903e2d8…` under suite `e068770a…`, runner
RUN-260917-48ff7b) for the 14 fuzz gates, cross-compiles, JSON validity,
`task-board validate` and `git diff --check`; the producer's `cmdNN.log` set.
Everything else above I re-ran myself.

## 8. Mutation battery (isolated copy `/tmp/rev-10k118/mut`, pristine `…/cand`, every plant restored and `cmp`-proved; `PYTHONDONTWRITEBYTECODE=1`, no `__pycache__`)

Legend: KILLED = the named test fails under the plant (and, where stated, the
production entry admits the narrowed member, proving the plant is live);
SURVIVED = no shipped test reddens. "(x2)" = the named killer was executed twice
with `-count=1`, both red. Raw per-plant logs (`<id>-named.log`,
`<id>-named-rerun.log`, `<id>-suite.log`, `<id>-tracecheck.log`) and the harness
`review-mutants.sh` are in the evidence archive.

Registry plants through the production entry (`tracecheck` = `VerifyRepository`),
each with `reviewedOwnershipCanonicalSHA256` re-pinned by my digest mirror so the
SEMANTIC gate is what gets measured, not the digest:

| Plant | Result |
| --- | --- |
| J1 section:6.2 production declaration → non-existent `NoSuchReviewerDeclaration` (gap absent on a `full` binding, nothing else changed) | REFUSED, exit 1: `section binding "section:6.2" production owner: declaration "NoSuchReviewerDeclaration" is absent from "internal/config/schema.go"` (run manually in the probe copy after the harness's first attempt tripped on the missing gap key and measured the pristine registry — that attempt is discarded). |
| J2 section:6.2 keeps its clause with acceptance links detached (evidenced claim, no acceptance owner) | REFUSED, exit 1: `section binding "section:6.2" clause "6.2#1" names no acceptance case that discharges it`. |
| J3 section:13.10 gap owner `STORY-260916-3aukw3` → `TASK-260916-3aukw3` | tool exit 0 (production bound: board identity is not a repository fact); `TestAdoptedSectionsHavePendingOwnersAndRefuseRuntimeAdmission` FAIL and `TestV070RegistryRederivesFromTrunkV060Registry` FAIL — both named tests kill it. |
| J4 clause 6.2#1 line 2512 → 2513 | REFUSED, exit 1: `section binding "section:6.2" clause "6.2#1" declares line 2513, but the pinned clause is at line 2512`. |
| J5 section:13.10 binding removed and disclosed unowned (dodge) | REFUSED, exit 1 (`unowned section "section:13.10" states no evidence for its gap`); both named tests FAIL as well. |

Production-code narrowing mutants:

| Mutant | Result |
| --- | --- |
| C1 `VerifyRepository` legacy loop drops the v0.5.0 projection (only v0.6.0 verified) | SURVIVED (traceability, tracecheck and catalog suites green; tool exit 0). Both operands are compiled-in artifacts, so no test fixture can make them disagree; the property is pinned independently by `TestV050ProjectionMatchesHistoricalLock` / `TestV060ProjectionMatchesHistoricalLock` against the real lock bytes. Stated bound, same shape as trunk's single-release loop. |
| C2 assigned-scope admission `measured.Level != coverageFull && key != "section:13.10"` (admits exactly the adopted 13.10) | KILLED: `tracecheck -section 13.10` on the mutant exits 0 and prints `assigned_scopes=1` (narrowing live); `TestAdoptedSectionsHavePendingOwnersAndRefuseRuntimeAdmission/13.10` fails: `VerifyAssignedSections(13.10) = <nil>; pending owner must not grant runtime admission`; traceability suite exit 1 (tracecheck cmd suite stays green — its sliver table does not list 13.10, see observations). |
| C3 cigate `checkReleaseRoots` historical root compared by `strings.HasPrefix` | SURVIVED the shipped cigate suite (drift table has no same-prefix row). My probe row `checkReleaseRoots("v0.7.0","v0.4.3","v0.7.0","v0.4.30")` fails on the mutant and passes on pristine → P3-2. |
| C4 specpin `validateV070` `len(Contracts) != 64` → `< 64` (admits a 65-row lock) | SURVIVED specpin + cataloggen suites: every mutated lock in the tests also fails the `ManifestSHA256V070` digest and the tests only assert `errors.Is(ErrPinMismatch)` → P3-3. |
| C5 specpin `ContractsForRelease(v0.7.0)` also accepts a v0.6.0 manifest | KILLED: `TestStalePinsCannotAuthorizeV070`: `v0.6.0 ContractsForRelease(v0.7.0) error = <nil>, want ErrUnsupportedRelease`. |
| C6 specdoc `ParseV070` also admits the v0.6.0 digest | KILLED: `TestParseV070RefusesEveryNonAdoptedDocument/stale_v0.6.0_document`: `ParseV070(stale v0.6.0 document) error = <nil>`. |
| C7 cataloggen `Generate` falls back to `VerifyV060` when `VerifyV070` refuses | KILLED: `TestGenerateRefusesStaleV060Authority` (`… unsupported specification release: v0.7.0, want the lock layer refusal` — the test pins the refusing layer, not just the refusal). |
| C8 `resolveAssignedSections` keeps `IsSectionV060` | SURVIVED as expected — equivalent mutant, the two inventories are identifier-equal (documented in sections.go and adoption-v0.7.0.md). |
| CTL harmless operand swap in cigate current-root comparison | SURVIVED (control). |

The two new verification tests (producer plants N1–N5/R1–R3/C0/C0R re-run by me,
plus my own N6, N7, R2b), each KILLED run twice:

| Plant | Result |
| --- | --- |
| N1 drop carried binding section:6.2 | KILLED x2 (`drops 1 carried bindings: ["section:6.2"]`). |
| N2 clause 6.2#1 left at its v0.6.0 line 2433 | KILLED x2 (`v0.7.0 binding "section:6.2" clause 6.2#1 excerpt does not begin at line 2433` — the verbatim-quote layer fires before the shift assert, as the producer corrected). |
| N3 self-minted section:9.9 binding | KILLED x2 (`section:9.9" is not carried and not a reviewed addition`). |
| N4 drop carried case `config-versioned-readers` | KILLED x2. |
| N5 drift the section:6.1 gap | KILLED x2 (`gap changed without a reviewed rewording`). |
| N6 (mine) section:6.2 re-attributed to the existing `catalog.go:ForRelease` | KILLED x2 (`binding "section:6.2" production moved from {internal/config/schema.go Decode} to {internal/catalog/catalog.go ForRelease}`). |
| N7 (mine) new section:13.10 claims `sliver` with a fabricated clause | KILLED x2 (`claims sliver coverage, want unevidenced for a story-owned stub`). |
| C0 identical JSON round-trip | SURVIVED (control). |
| R1 fenced `bindings=68` → `67`, prose intact | KILLED x2 (`README fenced coverage line = … tracecheck printed …`). |
| R2 prose `569` → `568`, fence intact | KILLED x2 (`publishes 568`). |
| R2b (mine) prose `Two bindings are full` → `Three …` | KILLED x2 (`matches "(Two) bindings are `full`" 0 times, want exactly 1`). |
| R3 token-preserving: README byte-intact, section:10.1 binding removed, digest re-pinned to `2fd59880…` | KILLED x2: production stays green (tool exit 0, prints `bindings=67 … unevidenced=51 … clauses_discharged=49/566`), the pin test fails on the byte mismatch, and the behavioral suites redden (traceability 5 FAIL lines, tracecheck 5 FAIL lines) — the harness executes the behavioral suite, not only the checker. |
| C0R case-only `SIXTY-EIGHT` | SURVIVED (control; the closed number-word map lowercases). |

Restore proof: `git status --short` of the mutation copy empty after the run; no
`__pycache__`/`.pyc`; the pristine export still hashes to tree `8903e2d8…`.
Score: 15 KILLED (C2, C5, C6, C7, N1–N7, R1, R2, R2b, R3) / 4 SURVIVED-by-design
(C1 bound, C8 equivalent, CTL, C0, C0R controls) / 2 SURVIVED findings (C3 → P3-2,
C4 → P3-3); registry plants J1, J2, J4, J5 refused by the production entry, J3
refused by the named tests (production bound).

## 9. Findings

- **P3-1 (carried reviewed bytes, registry attribution):** the v0.7.0 source
  normative-section group (`source:1 … source:appendix-d`, 24 keys) keeps
  production `internal/specpin/pin.go:VerifyV060` while its acceptance cases moved
  to `source-pin-v070-exact/refusal` (production `CurrentV070`/`VerifyV070`). The
  v0.5.0→v0.6.0 precedent moved `Verify`→`VerifyV060`. The gate only checks that
  the declaration exists, so it is green; the producer disclosed it as F1 and pinned
  the carried literal in the re-derivation test with the rationale. No gate or
  runtime impact; follow-up: re-point to `VerifyV070`, re-pin the registry digest
  and the F1 pin in one reviewed change.
- **P3-2 (carried tests, cigate):** `checkReleaseRoots` compares roots by string
  equality, but the drift table has no same-prefix row, so my prefix mutant (C3)
  survived the shipped suite; a one-row addition (`v0.4.3` vs `v0.4.30`) kills it.
  Production passes constants, so the exposure is a future partial re-point that
  shares a prefix.
- **P3-3 (inherited test shape, specpin):** the pin negative tests assert only
  `errors.Is(err, ErrPinMismatch)`, and every mutated lock also fails the
  `ManifestSHA256V070` digest, so a narrowing of `validateV070`'s structural checks
  (C4: `!= 64` → `< 64`) survives — the structural layer is shielded by the digest
  and is unmeasured by tests. Same shape as the trunk `validate`/`validateV060`
  gates (not a regression); follow-up: assert the structural message on a digest-
  preserving fixture or run the structural checks in their own test.
- Observations (no grade): the sliver-refusal tables (`TestVerifyAssignedSections
  RefusesEveryBindingThatOnlySlivers`, `TestRunRefusesEveryAssignedSectionThatOnly
  Slivers`) are hand-enumerated and never listed every below-full binding (40 rows
  vs 66 today); the three new bindings are refused at the production entry by
  `TestAdoptedSectionsHavePendingOwnersAndRefuseRuntimeAdmission` (library level)
  but not by the `run()`-level table. The adopted-section denominator is an
  explicit 6-row table because the inventories are identifier-equal; the seventh
  area (Appendix D / vucwn0) and 7.3 are pinned by the re-derivation test's
  `rewordedV070Gaps` instead. The exported 238-file pin patch hazard, the 49-vs-48
  prose slip in the rev3 results, the stale ci.yml job name, and the 193 pre-existing
  `MISSING_ACTIVITY` board issues are correctly out of this candidate's scope.
- Production bound (not a finding): `VerifyRepository` cannot know board
  identity, so a gap naming `TASK-…` instead of `STORY-…` as the pending owner is
  admitted by the tool once the digest is re-pinned (J3); the two named tests
  (`TestAdoptedSectionsHavePendingOwnersAndRefuseRuntimeAdmission`,
  `TestV070RegistryRederivesFromTrunkV060Registry`) and the digest layer are the
  real gates, as the producer's R8 states.

## 10. Checklist coverage (reviewer-owned items)

- Implementation matches AC: yes — union of the two accepted revisions (19 machine-
  applied files byte-equal to the rev3 bytes per the producer, re-verified here by
  content, not trusted) plus the two verification tests; task-board.config.json ==
  HEAD.
- Solution fits project architecture: yes — adopt-not-fork (V070 entry points beside
  V060/V050, historical projections derived, no consumer widened, census-only rows
  documented in generate.go and adoption-v0.7.0.md).
- Tests green: yes (section 7).
- Gates attacked, not read: section 8.
- Verdict recorded: this document; `accept_cr(TASK-260917-10k118, revision=1,
  evidence=TASK-260917-10k118_review-verdict-rev1.md)`.
