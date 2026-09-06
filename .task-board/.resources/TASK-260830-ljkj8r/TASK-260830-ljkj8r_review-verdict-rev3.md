# TASK-260830-ljkj8r — reviewer verdict, CR rev3 (RUN-260906-704600)

**ACCEPTED.** All four round-2 findings (C1–C4) are closed with
production-derived gates, and both derived gates survive the attacks the
brief asked for. Reproduced independently: candidate tree
`9f5ec76cf972cc73fbebfd8faa211f01078e4737` verified byte-identical before
and after **288 mutant/plant applications**.

## Ratios

| Battery | denominator | killed | survived | unmeasured |
| --- | ---: | ---: | ---: | ---: |
| Arm reachability (deletion, `if false && (cond)`) | 142 AST-derived if-sites | 140 | 2 (both declared `defensiveArms`) | 0 |
| Query obligations (`return false`→`true`) | 112 AST-derived sites | 111 | 1 (declared `defensiveObligations`) | 0 |
| C3 sorted-unique narrowing (`>=`→`>`) | 8 sites | 8 | 0 | 0 |
| C4 deadline widening + controls | 4 | 4 | 0 | 0 |
| Bound widening sample (`max`→`max+1`) | 14 sites | 14 | 0 | 0 |
| **Production-derived total** | **280** | **277** | **3 (all declared)** | **0** |
| Gate-integrity plants (A1/A2/A3/A3b/B1/B3a/B3b/traceability) | 8 | 8 caught | 0 | 0 |

- 0 `COMPILE_FAIL`, 0 `NOT_APPLIED`, 0 census-only kills counted.
- **AC rows: 9 of 9** driven through named production entry points by named
  committed tests (`round2.md` table; AC unchanged, table re-verified).
- Coverage 83.3% → 87.6% (`-cover`); function-level profile 83.2 → 87.5.

## G-A — is the reachability matcher as tight as its doc claims?

**Yes, in both slide directions.** My denominator is 142 if-sites derived by
my own `go/ast` selector; the producer's 144 is a strict superset (the two
extra, `bootstrap.go:151` and `scan.go:314`, are wrapper `if`s around nested
arms my selector attributed to the inner arm). No inflation.

| Plant | Result |
| --- | --- |
| A1 — second arm at the same site whose detail is a strict **prefix** of `request envelope carries unknown member`, registered in `defensiveArms` so the roster could not mask the matcher | **RED.** `row "…carries unknown member": message "…carries unknown" reaches "ctor\|failViolation\|request envelope carries unknown", not the named arm` |
| A2 — same, detail a strict **superstring** (`…unknown member of a closed set`) | **RED.** two literals matched → ambiguous → row failed, naming both |
| A3 — third `checkResponseIdentity` instantiation (`"probe"`) | **RED.** `contexts = map[failure:true probe:true success:true], want exactly success and failure` |
| A3b — context passed as a variable instead of a literal | **RED.** `context … is not a literal; the matcher cannot resolve it` |

Note A3 matters: the reachability rows themselves stayed **green** under the
third instantiation. `TestReachabilityContextsAreClosed` is the only thing
standing between the matcher and a silently unresolvable context, and it
fires.

**Kills are behavioral, not roster noise.** Every one of the 140 arm kills
named a `TestEveryDerivedArmFiresItsVector/…` subtest, and the deletion shape
preserves the `failX(` token the census greps for — so the census cannot be
the killer. Cross-checked against `TASK-260830-ljkj8r_arms_table.md`:
**140 of 140 agree with the producer-named killer row, 0 disagreements.**

**The diagonal is exact.** 139 of 140 sites are killed by exactly one
reachability row. The 140th (`protocol.go:414`) kills its conduit plus its 5
frame faults — the documented `also` carrier shape. **No reachability row
dies under more than one site deletion.** 145 of 146 rows serve as a killer;
the one that does not (`failUnknownOperation|dispatch names an operation
outside the closed registry`) sits in `RefuseUnknownOperation`, an
unconditional exported refusal with no `if` to mutate, driven by
`TestRefuseUnknownOperation` and mirroring the identical accepted API in
`internal/sessadapter`.

**The defensive escape hatch is 2 arms, not 609.** The brief's "609 defensive"
misreads the producer's note: 609 is the **line number** of the single
defensive obligation (`query.go:609`, `checkQueryParameters` obligation-1).
`defensiveArms` holds exactly 2 entries and `defensiveObligations` exactly 1.
Both arm rationales are true, not decoration:

- `scan.go:365` journal-not-exportable — attacked directly through the public
  `Import`→`Export` path with 5 hostile key vectors. Lone-surrogate escapes
  are **refused by `Import`** (`idempotency journal lone surrogate escape`);
  NUL, BOM, U+FFFD and control sequences all export cleanly. The arm is
  unreachable, as stated.
- `protocol.go:377` request-not-encodable — `decodeStrictObject` validates the
  raw body before `json.Marshal`, and every other member is a validated
  string/uint53. Unreachable, as stated.

## G-B — the 112-site census and the one-arm problem

**Denominator is production-derived and fails closed.** I re-derived it with
my own scanner: **112 sites, exact match.** Planting a new
`if len(name) > 4096 { return false }` in `checkQueryParameters` reddens
`TestQueryObligationRosterIsComplete` with
`derived obligations with no witness vector: query.go|checkQueryParameters|obligation-6`.
Not enumerated-and-frozen.

**Battery run against the behavioral test alone** (`-run
TestEveryQueryObligationIsDriven`), so the roster-renumbering artifact the
producer excluded is excluded from my numbers too: **111/112 killed**, sole
survivor `checkQueryParameters obligation-1`. **111 of 112 sites redden their
own named subtest** — the diagonal is complete.

**The one-arm problem is resolved by mutation, not by message.** All 112 still
report `query_invalid` through `checkQueryOperation`, and
`TestEveryQueryObligationIsDriven` asserts code only. That is sufficient here,
and the reason is measurable rather than asserted: **93 of 112 mutants kill
exactly one subtest.** Every overlap is one of the two documented structural
shapes and nothing else:

- framing gates `checkQueryOperation` ob-7…ob-11 kill their own row plus the
  validator rows beneath them (ob-7 → 68 rows, the parameter union; ob-8 → the
  projection rows; ob-9 → pagination; ob-10 → sort; ob-11 → flags);
- layered delegates `checkEnumSubset`, `checkUUIDv7DigestSubset`,
  `validQueryField`, `validQueryPreset` pair with the `checkFilters` /
  `checkQueryProjection` rows that traverse them.

A witness satisfiable by a sibling obligation would not redden when its own
site alone is flipped. Each of the 111 does. That is per-reason resolution.

**Defensive accounting — 1 site, tripwire verified in both directions.**
Removing `"hosts"` from `queryParameterMembers` reddens
`TestQueryParameterMembersMatchRegistry` (`table keys = 16, want the 17 union
members`), and under that divergence flipping obligation-1 additionally
reddens `TestSortedUniqueUUIDv7Edges/hosts_host_ids` and
`TestDecodeQueryOperationValidatorsRefuse` — i.e. the site is behaviourally
load-bearing exactly when the tripwire says it is reachable. A true stated
bound.

## G-C — C4 and the residue

Both `EncodeRequest` entries are driven independently: the row returns "admitted"
if **either** entry admits an out-of-range value, so an entry that admits while
the other refuses fails the row. The comment now says exactly that. 4/4 killed
(`ceiling+1`, dropped-ceiling disjunct, and both floor controls), each by
`TestUint53BoundEdges/request_deadline_ms` **and** the reachability row, with
`TestEncodeRequestRefusals` now carrying the direct `3600001` refusal.
The merged `failUnknownOperation` key is resolved per site: `protocol.go:338`
kills the `|via EncodeRequest` row, `protocol.go:496` kills the decode row.

No arm remains resolvable only by code: every reachability row asserts the full
rendering (wire code + derived wrapper + derived detail), and the wrapper is
AST-derived from the eight constructor bodies with any non
`Code:<lit>, Message:<lit>+detail` shape a violation.

## G-D — ratio, resurrections, tree

**Resurrections: none, and structurally none possible.** Production is
byte-identical to rev2 across all eight files (hashes reproduce the producer's:
`bootstrap 1f81b4bc`, `decode 99ac4a1b`, `doc 7f79e567`, `manifest c7eeac01`,
`probe 96141492`, `protocol a438b041`, `query c0aad9f7`, `scan 93f378a0`).
Test inventory 92 → 98 with **zero removals**; the only deleted assertion lines
are the chained deadline drive C4 replaced. All four modified test files
strengthen (three unknown-member vectors retargeted, one refusal added, two
census prose corrections). Spot re-verification: 14/14 bound widenings still
die, C3 8/8 still die 1:1.

**Coverage delta 83.3% → 87.6% accounted at function granularity**: 20 functions
moved, 11 `query.go` validators to 100% (`checkPlanContinueParameters` +27.3,
`checkQuerySort` +24.1, `checkQueryOperation` +22.6, `checkSubject` +21.4,
`checkExecutePlanParameters` +18.8, `checkCaller` +18.4, …) plus the envelope
and manifest paths the reachability rows now drive. Exactly the C1/C2 surface.
Zero functions at 0.0% in either revision.

**Suite, from the restored tree:** 18 packages ok, `go vet ./...` clean,
`go build ./...` clean, `gofmt -l internal/` clean, `tracecheck` ok
(`acceptance_cases=94`). Ownership re-pin is computed, not asserted: renaming
one acceptance case id fails closed with the exact mismatch
(`371d39f6… differs from reviewed fbdbb5b6…`).

README is correctly hedged — the §7.9 clause binding stays `unevidenced`, the
framed subset is stated as exactly manifest/probe/scan plus Directory Query and
pinned by `TestFramedSubsetIsExact`, and the eight unframed operations are
disclosed rather than silently absent. No unsupported capability claim.

## Non-blocking notes

- **N1 (repeat of rev1-N2, story-level):** `internal/dirnode` is not imported by
  any command yet, identical to the accepted `sessadapter` and `provhost`
  leaves. Its "production entry point" is the package's exported API. This is
  the Story's integration step, not a leaf finding.
- **N2:** `TestUint53BoundEdges/request deadline_ms` keys its out-of-range
  branch on the hardcoded literals `0` and `3600001`. If the row's `min`/`max`
  ever move without those literals, the row silently degrades to the round-2
  shape (decode refuses, encode never checked). The coupling is stated in the
  comment and the two live three lines apart; noted, not blocking.
- **N3:** `TASK-260830-ljkj8r_round3.md` carries a stray non-English token
  ("the two제도 comment-only corrections"). Board artifact only; cosmetic.

## Evidence produced by this run

`.temp/TASK-260830-ljkj8r/rev3/` — `arm_results.tsv` (142 rows),
`oblig_results.tsv` (112 rows), `c3_results.tsv`, `arm_sites.txt`,
`obligation_sites.txt`, `final-suite.log`. Batteries and helper scripts:
`arm_battery.py`, `oblig_battery.py`, `c3_battery.py`, `tree.sh`.
