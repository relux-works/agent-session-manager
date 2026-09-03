# BUG-260902-2m7slg — Reviewer verdict for CR-BUG-260902-2m7slg-3 (rev 3)

**Verdict: ACCEPTED.**

Candidate tree `163d498df0669dc85c9cccaf258980dadadf0c35`, base
`fa06e6d86926252233578b755d0466b3df713fbb`, leaf `2d0962c`, branch
`task-board/story/STORY-260903-b48t03`. Reviewed in the Story worktree; the tree
was unmodified by this review (`git write-tree` = `HEAD^{tree}` = `163d498`
before and after). Every mutation below was run in a throwaway `git archive`
copy under `/tmp`, never in the worktree.

## What changed between rev2 and rev3

Nothing in the repository. `rev2.patch` and `rev3.patch` are byte-identical
(sha256 `484fc11374c6b5d33f691c8a4cfa5294a2539e7d20bd4b619973a72ff270d357`). The
rev2 rejection was solely that `BUG-260902-2m7slg_coverage-gate.md` still
described the rev1 gate model and was false about admission, the
`declarative`/`unmeasured` level, and the printed report line. Rev3 rewrote that
artifact in place. So this review had two jobs: re-attack the shipped gate, and
verify the artifact against the shipped binary line by line.

## The gate was attacked, not read

14 independent **narrowing** mutants applied to `internal/traceability/traceability.go`
in the sandbox — each keeps the check present and weakens its class, none is a
delete-only mutant. All 14 were killed by `go test ./internal/traceability`:

| # | Narrowing mutation | Result |
| --- | --- | --- |
| M1 | admission `!= coverageFull` → `== coverageSliver` (admits unevidenced/unmeasured/partial) | killed |
| M2 | `coverageBucket` returns `full` at the partial threshold | killed |
| M3 | clause inventory stops descending into subheadings (shrinks the denominator) | killed |
| M4 | unowned entry may cover a catalog-required section | killed |
| M5 | excerpt may match anywhere in the document instead of beginning at its line | killed |
| M6 | a section may be both owned and unowned | killed |
| M7 | assigned admission ignores the unowned disclosure | killed |
| M8 | gap need not name the production declaration | killed |
| M9 | gap section mention by bare substring (re-opens 6.55-for-6.5) | killed |
| M10 | a clause may discharge through a case the binding does not own | killed |
| M11 | declared coverage label is trusted instead of recomputed | killed |
| M12 | contract/fixture owners may claim section coverage | killed |
| M13 | clause line attribution not checked | killed |
| M14 | unowned key need not be a real v0.5.0 section | killed |

The producer's own load-bearing mutant was reproduced independently.
`if measured.Level != coverageFull` → `... && measured.Level != coverageUnmeasured`:
`tracecheck -section 15.2`, `-section 7.3` and `-section 13.14.5` all flip to
**exit 0** — this Bug's consequence paragraph verbatim — and
`TestVerifyAssignedSectionsBindsGranularScopeToOwnersAndExecutableCases`,
`TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers`,
`TestRunReportsExactCoverageAndFailsClosed` and
`TestRunRefusesEveryAssignedSectionThatOnlySlivers` redden. Both production entry
points (`VerifyAssignedSections`, and `main` → `run` → `VerifyAssignedSections`)
are covered.

## Anti-vacuity, both arms, on the shipped data

Driven through the real command, not a helper:

* `go run ./internal/traceability/cmd/tracecheck -section 6.2` → **exit 0**, the
  genuinely adequate 1/1 binding still admitted. Its clause (`SPEC.md:2417`,
  "terminal backend MUST be `conpty`" on native Windows) is really asserted by
  `TestEveryPinnedReaderHasPositiveNativeWindowsAndWSL2Lanes`, which is really
  registered under the `config-versioned-readers` acceptance case, driving the
  production `Load` across every pinned reader version. The positive arm is real,
  not synthetic.
* `-section 6.5` → exit 1, `0/3 unevidenced`; `-section 2.2` and `-section 18.4`
  → exit 1, recorded unowned; `-section 10.3` → exit 1, `1/3 sliver`;
  `-section 7.3`, `-section 15.2`, `-section 13.14.5` → exit 1, `0/0 unmeasured`.
  All three named slivers redden at the production entry point.

## The measured ratio was recomputed independently

A throwaway dump built in the sandbox from the shipped registry plus a fresh
`sectionClauseInventory` over the pinned document reproduces **every row** of the
artifact's 48-binding table — key, level, numerator, denominator, production
declaration — and totals `2/394`, matching the printed
`clauses_discharged=2/394`. Spot-verified denominators: 4.B 12, 1.6 31, 7.5 53,
9.2 35, 11.3 27, appendix-d 16, 6.2 1, 10.3 3.

## Acceptance criteria

* **A binding declares its coverage; assigned-scope admission gates on full.**
  Yes, and the declaration is recomputed rather than believed (M11).
* **The three known slivers re-declared honestly, not quietly upgraded.**
  `section:2.2` and `section:18.4` moved to `unowned_sections` with gap and
  evidence; `section:6.5` kept its binding at `unevidenced` 0/3 with a gap naming
  the real `required_capabilities` defect. Old registry had 50 section bindings;
  new has 48 + 2 disclosed unowned. Nothing was re-bound to a friendlier symbol
  and nothing vanished.
* **A binding claiming full while implementing one clause reddens the gate,
  proven by planting one.** Proven 13 ways in
  `TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted` and 15 ways in
  `TestPlantedSliverRedensTheProductionEntryPoints` (subtest counts verified: 13,
  15, 4), plus M1–M14 above.
* **Sections whose semantics are unimplemented are recorded as unowned.** Yes,
  and `unowned_sections` is a disclosure not an exemption: it cannot cover a
  catalog-required section (M4 killed), cannot double as owned (M6 killed),
  cannot be self-minted for a nonexistent section (M14 killed), and is inside the
  reviewed projection digest.

## Reported facts were checked, not taken on trust

* The artifact's residual claim — that an enabled `external_trust` backend is
  admitted as the selected backend on native Windows while `SPEC.md:2417`
  requires `conpty` — is **real**. `internal/config/validation.go:682-687`
  refuses only the two built-in platform mismatches; an enabled external entry
  joins `registered` and is selectable. Correctly reported as out of scope and
  owed its own board item.
* `SPEC.md:2585` really says `required_capabilities` defaults to the platform
  lane minimum; `validateTerminal` really accepts only an empty default
  (`!RequiredCapabilitiesExplicit && len(...) != 0` → refusal). The 6.5 gap is
  accurate.
* `SPEC.md:11073+` really is the exit-code table with no RFC 2119 keyword; the
  `unmeasured` blind-spot story holds.
* The scanner blind spot is measured, not asserted:
  `TestUnmeasuredCoverageIsAScannerBlindSpotNotAnAbsenceOfObligation` re-derives
  19 of 157 headings from the pinned document and requires each to have a
  substantive body. Keyword absence is reported as a failure to measure, and the
  bucket is refused for admission rather than admitted on an unverifiable
  sentence — the correct handling of "an absence and a failure to read are
  different facts".

## Scope stated truthfully

The residual — *the gate cannot decide that the named acceptance case exercises
the clause's meaning* — is stated in `README.md`, in the `verifySectionBindingCoverage`
doc comment, and in the artifact, in each case with the concrete instance
(`section:6.2`) rather than as an abstraction. The gap-quality bound ("a sentence
naming both and still saying nothing is admitted") and the thin single-row
positive arm are both disclosed rather than hidden. That is the honest scope
statement the AC asked for.

## No assertion weakened or deleted

Deleted positive-admission assertions were **inverted**, not dropped:
`TestRunAssigned*` became `TestRunRefusesEveryAssignedSectionThatOnlySlivers` and
`TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers`, which pin each
refusal by exact measured ratio, so a section that becomes covered must leave the
table deliberately. The "an unrelated binding stays green" isolation assertions
in `TestMainRejectsOneNarrowedAssignedSectionBinding`,
`TestMainRejectsDetachedScopeSpecificAcceptanceCase` and
`TestMainRejectsMissingScopeSpecificProductionDeclaration` were migrated 7.9 →
6.2, not removed. Two rename-mutant rows left the table only because
`section:2.2` and `section:18.4` no longer have owners to rename; the same
declarations stay attacked via the `section:2.1`/`section:2.3`/`section:2.4` and
`section:3.3` rows.

New guard `TestEmbeddedDocumentNeverReachesAProductBinary` is itself
anti-vacuity-proven (it plants a `cmd/ax` and requires exactly one violation
naming it) and is the right response to `internal/specdoc` entering non-test
code.

## Validation re-run by this review on the candidate tree

| Command | Exit | Result |
| --- | ---: | --- |
| `go build ./...` | 0 | |
| `go vet ./...` | 0 | |
| `gofmt -l .` | 0 | no output |
| `go test ./... ` | 0 | 11/11 packages ok |
| `go test ./internal/specdoc ./internal/traceability ./internal/traceability/cmd/tracecheck -count=1 -cover` | 0 | specdoc 100.0%, traceability 86.4%, tracecheck 88.5% — exactly as the artifact claims |
| `go generate ./internal/catalog` + `git diff --exit-code -- internal/catalog/catalog_gen.go` | 0 | no generated-catalog drift |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `bindings=48 full=1 partial=0 sliver=1 unevidenced=43 unmeasured=3 unowned=2 clauses_discharged=2/394` |
| 8 × `tracecheck -section <known sliver>` | 1 each | refusal text matches the artifact verbatim |

`.github/workflows/ci.yml` really invokes the repository-wide gate, and
`TestCIWorkflowInvokesTraceabilityGate` still pins that.

## One defect found, not blocking

`BUG-260902-2m7slg_coverage-gate.md`, section "The load-bearing mutant,
re-measured by this run", says **"Five test functions redden"** and then names
and enumerates **four**. Measured on this tree with that exact mutant: four test
functions fail — `TestVerifyAssignedSectionsBindsGranularScopeToOwnersAndExecutableCases`,
`TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers`,
`TestRunReportsExactCoverageAndFailsClosed`,
`TestRunRefusesEveryAssignedSectionThatOnlySlivers` — which is exactly the list
the artifact prints. The word "Five" should be "Four".

Not grounds for rework: it is a one-word arithmetic slip that is self-refuting
against its own adjacent enumeration, it misstates no gate behaviour, exit code,
coverage level or ratio, and every other number in the artifact that this review
independently recomputed (13/15/4 mutant subtests, 2/394, all 48 table rows, the
three coverage percentages, all ten reproduced exit codes) is exact. Correct it
in place at the next touch of this artifact.

## Handoff

Reviewer-archetype run; no `commit_ack` supplied. Acceptance recorded with
`accept_cr(BUG-260902-2m7slg, revision=3, evidence=BUG-260902-2m7slg_review-verdict.md)`,
which parks the element at `to-review`. The orchestrator checkpoints or
integrates the accepted revision and makes the `done` transition with
`commit_ack=scope_committed`.

Two follow-ups the change correctly reports rather than fixes, each owed its own
board item: Section 6.2's `conpty` clause is not enforced against enabled
external terminal backends, and Section 6.5's `required_capabilities` default is
empty where the platform lane minimum is required.
