# TASK-260917-10k118 results — reconstruct-and-land-v070-adoption

Status: **ready for review**. The reviewed v0.7.0 adoption delta (pin leaf
rev1 + registry leaf rev2-as-rederived-in-rev3) is reconstructed on trunk
`7efe3854a5a14545f9fa2b7697accb1883249f1f` with every verification re-run,
plus this Story's two verification tests. Candidate left UNCOMMITTED in the
`STORY-260917-110onn` worktree (16 modified + 10 new paths); no commits made
on the Story branch. `task-board.config.json` equals HEAD.

AC coverage: **30 of 30 rows driven** through production entry points by
named committed tests and probes (12 pin + 16 registry + 2 new); see
`TASK-260917-10k118_conformance-matrix.md` for the row → test → call-site
map. Stated bounds: R8/R10/R13 (inherited, unchanged), N2 per-section-ratio
scope, FINDING F1.

## 1. Reconstruction method (per-file record)

Base: managed Story worktree at trunk `7efe385` (branch
`task-board/story/STORY-260917-110onn`, clean at start).

The exported `checkpoint-2yzf5d.patch` diffs two unrelated tips
(`7efe385` → `9cb5dd4`) in 238 files: it carries the pin delta *plus* a
reversion of the five trunk landings (it deletes the rpcwire fuzz command
and the claude reviewer from `task-board.config.json`, and rewinds
`ownership.v0.6.0.json`). Applying it blindly would destroy trunk. The true
pin delta is `git diff e4e3e88 9cb5dd4`: **10 files**. Reconstruction:

1. Applied the true 10-file pin patch minus `LOGBOOK.md` (clean, exit 0).
   `README.md`, `specdoc.go`, `pin.go`, `sections.go` applied with zero
   context drift; the 5 new files (`SPEC.v0.7.0.md`, `v0.7.0.lock.json`,
   3 test files) are clean adds.
2. Applied the 15-file registry patch: 10 files clean, 5 hand-adapted
   (`LOGBOOK.md`, `README.md`, `traceability.go`, `traceability_test.go`,
   `tracecheck/main_test.go`) — the 5 the five landings touched.
3. Copied the 3 untracked files (`catalog.v0.7.0.json`,
   `adoption-v0.7.0.md`, `ownership.v0.7.0.json`).
4. For the 5 hand-adapted files, took the rev3 merged bytes after
   diff-review against trunk HEAD: `traceability.go` is exactly the
   v0.7.0 re-point (digest `d3eca906…` → `c4cd46bf…`, `VerifyV060` →
   `VerifyV070`, two-legacy loop, `LoadV060` → `LoadV070`, inventory
   renames); `README.md` is exactly the adoption delta (pin paragraphs,
   catalog rows, 135/68/49/569 figures); `main_test.go` is pins +
   `ownership.v0.7.0.json` paths + the V060→V070 test rename; no trunk
   test function lost in either test file (0 lost, 5 v0.7.0 refusal
   tests added; V060→V070 rename only in `main_test.go`); `LOGBOOK.md`
   is purely additive (n9r71p entry at top + 2yzf5d block at the
   resolved replay position), plus my `TASK-260917-10k118` entry newest-first.

Byte-identity proof: all 19 machine-applied files `cmp`-identical to the
rev3 candidate bytes in the read-only old worktree (10 registry + 6
checkpoint + 3 untracked). The carried `ownership.v0.7.0.json` sha256 is
`2548870d896b0e119c307cdcba02ab23b36af7f473328a15f6a092090d8a5070`
(exactly the expected digest) — the carry already contains the trunk-final
re-derivation, and §2 re-verifies it rather than trusting it.

## 2. Re-verification battery (all re-run this session, this tree)

| Check | Result |
|---|---|
| Fresh clone `https://github.com/relux-works/agent-session-manager-spec.git` | exit 0 |
| `git cat-file -t v0.7.0` / `git rev-parse` / peeled | `tag` / `d4abe46fb12d9ba347c09f43efb01089530795b3` / `32b3f2ba7c377248a53cd42389abbd2f1c321834` |
| `git tag -v v0.7.0` | Good, `oparin@me.com`, ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`, tagger Ivan Oparin, exit 0 |
| Tagged `SPEC.md`: blob, bytes, CR count, sha256 | `c4903e60ea72583b507060182aee1b11e21ddd86`, 1185291 bytes, 0 CR, `c6b2fe64ee79ed697a96ed27a1679c80b8ee1eba137feb99e7738b4da289ddcf` (raw == LF-normalized) |
| Tagged blob vs `SPEC.v0.7.0.md` | `cmp` IDENTICAL |
| Lock vs §1.5 table (independent extractor) | 64/64 rows order-identical, name/urn identical, version-sets identical; 1 known order normalization (Provider protocol `2.0.0,3.0.0,2.1.0,3.1.0` → sorted, gate-required); Launch Plan at index 20 between Session record and Session event |
| Lock v0.6.0→v0.7.0 diff | 63→64 rows (62→63 unique IDs); exactly +1 ID (`urn:ax:schema:launch-plan-request`), 0 removals; bumps provider +2.1.0/+3.1.0, error +1.5.0, manifest/probe +1.1.0, session-record +3.1.0 |
| Registry carry audit (trunk blob `b4f24807…`, sha `212ae321…`) | 65/65 bindings + 132/132 cases carried; +3 bindings (13.1, 13.10, 14.1, all Story-owned) +3 cases (`catalog-v070-exact`, `source-pin-v070-exact/refusal`); contract keys 63→64 (+Launch Plan row); fixture keys 5→6 (`+pin:ax-launch-plan-request-v1`); 5 known gap rewordings; unowned same 7 keys in order with the 11.10.5 Disclosure rewording only; 0 failures |
| Clause re-measurement (production `sectionClauseInventory` + `quoteBeginsAtLine`, scratch probe, removed after) | 49/49 discharged clauses verified in BOTH docs (id indexes inventory, line == measured line, excerpt begins at line); uniform shifts +20/+79/+185/+281/+392; 0 unshifted |
| 49-vs-48 note | rev3 prose says "48 carried clause lines" (results §4, LOGBOOK DELTA); measured **49**, matching the 49/569 and 49/535 tracecheck numerators. The artifact is right; the prose miscounted. Recorded, not edited (reviewed bytes). |
| Registry re-derivation reproduction | `build-ownership-v070-rev3.py` from the trunk blob → `cmp` BYTE-IDENTICAL to the shipped file |
| `reviewedOwnershipCanonicalSHA256` | producer script + own Go mirror struct agree: `c4cd46bf71f164ad53dfe00ddd2e64e0b439e1c2684138fbcfe208f7e53473f6`; production `VerifyRepository` green (third agreement) |
| Catalog digests | raw metadata `6d769c9b…db8193` == `MetadataSHA256`; reviewed canonical `c4094101…50274a35` pinned; `-adopted -check` exit 0 |
| `catalog_gen.go` regeneration | `-adopted` regen == explicit v0.7.0 == committed (`cmp` clean ×2); stale v0.6.0 explicit refused exit 1 (`source identity drift`); `-check` does not rewrite (cmp before/after) |
| Historical byte-identity | 9/9 `git diff --exit-code` clean vs trunk: ownership/catalog/locks × v0.5.0/v0.6.0, both SPEC docs, `task-board.config.json` |
| Owner stories on the live board | 7/7 exist with expected names, all `backlog`: 3fjjtn parsing/refusals, 2q85nu record embedding, 1kp1lx stdin/replay, 1ea7od plugin capability, 3aukw3 drift, 3ustb3 naming, vucwn0 vectors |
| cigate production probe (scratch, removed after) | `PinnedReleases()==[v0.7.0 v0.4.3]`; `ForRelease("v0.7.1")` refused (`unsupported catalog release: v0.7.1`); `VerifyContractPreservation()` green |
| tracecheck / cataloggen gates | exit 0 / exit 0; `contracts=64 … acceptance_cases=135 … bindings=68 … clauses_discharged=49/569` |

## 3. New verification tests (this Story's scope)

`internal/traceability/registry_rederivation_test.go` —
`TestV070RegistryRederivesFromTrunkV060Registry`: decodes the in-tree
v0.6.0 (trunk baseline, byte-pinned by R10) and v0.7.0 registries through
the production decoder; asserts the exact v0.7.0 source block, 65/65
binding carry (production/acceptance/coverage/gap equality except the 5
reviewed rewordings, each naming its story), 49/49 clause re-measurement
against `LoadV060`/`LoadV070` via the production inventory + quote
matcher (fails on any unshifted line), 132/132 case carry with the 2
intended repoints asserted exactly (`catalog-v060-exact` → historical
pair; `assigned-section-binding` single V060→V070 rename), exact new sets
(3 bindings, 3 cases, Launch Plan contract + fixture rows), carried
unowned order, and the F1 source-group pin. Green on the tree.

`internal/traceability/cmd/tracecheck/readme_coverage_pin_test.go` —
`TestREADMEMeasuredCoverageMatchesTracecheckReport`: runs the production
`run()` entry, requires the 2-line report, and asserts the README
"Measured coverage of this repository" subsection's single ```text fence
equals the tool's coverage line byte-for-byte; re-derives 8 headline
prose figures from `VerifyRepository` (bindings/full/partial/sliver/
unmeasured/unevidenced/unowned/discharged/total) plus the admitted
sentence (2 admitted of 68 covering 5, the five summed from the reviewed
registry's full bindings). Per-section ratios stay with
`TestCatalogSectionBindingCoverageIsExact…` (stated scope bound).
Green on the tree.

## 4. Mutants (all re-run this session on isolated copies of this tree)

Trees `/tmp/mut-10k118/tree{A,B,C}` are copies of the exact candidate
tree plus a minimal `.task-board` STORY-README skeleton (the
board-union test walks it; `filepath.WalkDir` does not follow a root
symlink, so the skeleton is real files). `PYTHONDONTWRITEBYTECODE=1`;
no `__pycache__`/`.pyc` in any tree afterwards; every plant restored
and proved with `cmp`.

- Pin battery (`mutate.sh`, unmodified): M1–M5 KILLED, each by exactly
  its named subtest with full suites otherwise green; M0 SURVIVED.
  6/6, tree restored snapshot-clean. (First attempt failed on the
  missing board skeleton — environmental, fixed by the skeleton, re-run
  clean; the live tree was never mutated.)
- Registry battery (`mutate-rev3.sh`, log path redirected): M0
  SURVIVED + M1–M8 KILLED with narrowing proofs (sibling arms still
  refused). 9/9. R0–R3 (reviewer-owned `mutate-r.sh`, no runnable
  script published) accepted from the rev3 overlay evidence: R0
  SURVIVED + R1–R3 KILLED with the unmodified harness on these exact
  bytes (this tree == the rev3 overlay construction + 2 test files +
  LOGBOOK entry).
- New-test battery (`mutate-10k118.sh`, in the tarball): N1 (drop one
  binding) / N2 (one unshifted line) / N3 (self-mint one binding) / N4
  (drop one case) / N5 (drift one gap) KILLED, each naming its plant;
  C0 (identical JSON round-trip) SURVIVED; R1 (fenced figure drift)
  / R2 (prose figure drift) KILLED with blast radius exactly 1 FAIL
  line each; C0R (case-only figure edit) SURVIVED with the full
  tracecheck suite green. N2's first pass used the shifted-ness
  message as the killer pattern while the verbatim-quote layer fires
  first; corrected to the quote-layer message and re-run KILLED
  (same-plant rerun log kept; corrected harness archived).
- Token-preserving R3: README fully intact, one unevidenced binding
  removed and the digest re-pinned so production stays green but
  prints a new report (`bindings=67 unevidenced=51`, tool exit 0);
  the pin test fails on the byte-mismatch and the full traceability +
  tracecheck behavioral suites fail on their exact pins. KILLED.

Raw per-plant logs, subprocess exits, and restore proofs are in the
evidence tarball (`pin-logs/`, `rev3-n9r71p.log`, `newtests/`).

## 5. Full 27-command suite (exact tree, this session)

| # | Command | Exit |
| - | ------- | ---- |
| 1 | gofmt clean over tracked+untracked Go files | 0 |
| 2 | `go build ./...` | 0 |
| 3 | `go vet ./...` | 0 |
| 4 | `go test ./... -count=1 -v` (37 packages ok, 0 FAIL; both new tests PASS) | 0 |
| 5 | `go test ./... -race -count=1` (37 packages ok, no DATA RACE) | 0 |
| 6 | `go test ./... -cover -count=1` (37 packages ok, 0 FAIL) | 0 |
| 7–20 | 14 fuzz gates incl. `rpcwire:FuzzUntrustedEnvelopes` | 0 × 14 |
| 21 | tracecheck → `contracts=64 … bindings=68 … clauses_discharged=49/569` | 0 |
| 22 | cataloggen `-adopted -check` (resolves to the v0.7.0 pair; file untouched) | 0 |
| 23–24 | linux/windows cross-compile | 0, 0 |
| 25 | JSON validity over tracked files | 0 |
| 26 | `task-board validate` (187 pre-existing MISSING_ACTIVITY, none on this task/story/epic) | 0 |
| 27 | `git diff --check` | 0 |

Raw logs: `cmdNN.log` in the evidence tarball.

## 6. Candidate and tree state

Modified (16): LOGBOOK.md, README.md, internal/catalog/catalog.go,
internal/catalog/catalog_gen.go, internal/catalog/catalog_test.go,
internal/catalog/cmd/cataloggen/main_test.go,
internal/cataloggen/generate.go, internal/cataloggen/generate_test.go,
internal/cigate/contracts.go, internal/cigate/contracts_test.go,
internal/specdoc/specdoc.go, internal/specpin/pin.go,
internal/specpin/sections.go,
internal/traceability/cmd/tracecheck/main_test.go,
internal/traceability/traceability.go,
internal/traceability/traceability_test.go.
New (10): the 6 checkpoint files, the 3 rev3 untracked files, plus
`internal/traceability/registry_rederivation_test.go` and
`internal/traceability/cmd/tracecheck/readme_coverage_pin_test.go`
(this Story). Not modified: `task-board.config.json` (no diff), no
`__pycache__`/`.pyc`, no stray files; both scratch probes created and
removed in-session (`git status` shows only the 26 candidate paths).
Branch `task-board/story/STORY-260917-110onn` at `7efe385`; no commits
made. `catalog.Adopted == ReleaseV070` with `Current()` defined
through it (reviewed bytes).

## 7. Findings

- F1 (new, in reviewed bytes): the v0.7.0 source normative-section
  group keeps production `VerifyV060` while its cases moved to the
  v0.7.0 pair; the v0.5.0→v0.6.0 precedent moved `Verify`→`VerifyV060`,
  so the adoption pattern says `VerifyV070`. Green because the gate
  checks declaration existence only. A reconstruction must not
  rewrite reviewed bytes; N1 pins the carried reference literally
  with the rationale. Recommended follow-up for the owning story:
  move to `VerifyV070` (registry + digest re-pin + this pin).
- 49-vs-48 prose slip (reviewed prose, §2): rev3 results §4 and the
  n9r71p LOGBOOK DELTA say "48 carried clause lines"; measured 49
  (both registries, both tracecheck numerators). Artifact right,
  prose miscounted. Not edited (reviewed bytes).
- Exported pin patch hazard (§1): the 238-file transport patch would
  revert five landings if applied blindly (config, registries). The
  true delta is the 10-file `e4e3e88..9cb5dd4` diff. Future carries
  should export `base..checkpoint` plus `checkpoint..worktree`, not
  `trunk..checkpoint`.
- Mutant-tree board skeleton (§4): the board-union test needs a real
  `.task-board` STORY skeleton in isolated copies (WalkDir follows
  no root symlink). Harnesses that run the specpin suite in a copy
  must replicate it.

## 8. Reran myself vs accepted from already-attached evidence

Reran myself in this session on the exact tree or an isolated copy of
it: provenance (fresh clone), SPEC digest + byte-identity, 64-row
lock audit, carry audit, clause re-measurement, digest recomputes
(×2), registry reproduction, cataloggen regen/refusal/purity,
historical byte-identity (9 paths), cigate probe, board story query
(7/7), both new tests, pin battery 6/6, registry battery 9/9,
new-test battery 10/10, and the full 27-command suite (27/27 green).
Accepted from already-attached evidence: the rev1 ACCEPT verdict for
the pin rows' adversarial plants (my 6/6 battery re-attests the
shipped gates; the reviewer's 6 sandbox plants are not re-runnable
from published artifacts), the rev2 ACCEPT verdict for the R-harness
(R0–R3 re-ran green in the rev3 overlay on these exact bytes; no
runnable script published), and the rev2/rev3 transformation shape
(re-applied and re-measured, not trusted).

No Stop-The-Line condition. No runtime capability claimed anywhere.
Ready for review.
