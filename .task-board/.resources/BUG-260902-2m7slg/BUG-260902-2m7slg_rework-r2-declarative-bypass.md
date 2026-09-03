# BUG-260902-2m7slg — rework round 2: closing the keyword-absence bypass

Branch `task-board/story/STORY-260903-b48t03`, one leaf commit past checkpoint
`fa06e6d`. Amends the revision-1 leaf; no second commit added.

## The blocking finding, reproduced and closed

Revision 1's obligation scanner matched uppercase RFC 2119 keywords only. A
section stating its obligations as a table measured **0 clauses**, was bucketed
`declarative`, and `declarative` was **admitted** as an assigned scope with no
clause evidence. That reproduced this Bug's own consequence paragraph verbatim,
reached through the scanner instead of through the label.

Reproduction on the revision-1 tree (reviewer's, re-confirmed):

```
go run ./internal/traceability/cmd/tracecheck -section 15.2   -> exit 0, assigned_scopes=1
go run ./internal/traceability/cmd/tracecheck -section 7.3    -> exit 0, assigned_scopes=1
```

On this tree:

```
$ go run ./internal/traceability/cmd/tracecheck -section 15.2 ; echo exit=$?
spec-to-code traceability check failed: assigned section "15.2" binding
"section:15.2" discharges 0/0 normative clauses, which is unmeasured coverage;
assigned-scope admission requires full: ForRelease only returns the reviewed
typed catalog rows that reference 15.2; the RFC 2119 scanner measures no clause
line under 15.2, because its 19-row exit-code registry is a normative table
stated without uppercase keywords, so the gate cannot measure this section - and
no exit-code mapping is implemented, the only os.Exit calls in the tree being
the exit(1) failure paths of cataloggen and tracecheck.
exit=1
```

`-section 7.3` and `-section 13.14.5` refuse identically. `-section 6.2` stays
green at exit 0 — the positive arm is preserved.

## What changed, and the decision behind it

The reviewer allowed either "gate admission on an explicit justification" or
"drop the bucket from the admitted set". **Both were taken, and the second is
the load-bearing one.**

1. `coverageDeclarative` → **`coverageUnmeasured`** (`"unmeasured"` in the
   registry). The old name asserted a fact about the specification; the new one
   states a fact about the gate. Keyword absence is a *failure to measure*, not
   a measured absence of obligation.
2. `unmeasured` now owes the **same mandatory gap** as every level below `full`.
   The `Gap != ""` refusal on that bucket is lifted, so the bindings with the
   least evidence behind them are no longer the only ones structurally forbidden
   from disclosing it.
3. **Assigned-scope admission requires `full` and nothing else.** Admitting on a
   reviewed justification sentence was considered and rejected: the gate cannot
   verify such a sentence, so unlocking admission with one is self-minted
   evidence for exactly the class this gate exists to refuse. A fallback defined
   for absence must not fire on a failure to read.

Production call site: `verifyOwnership` in `internal/traceability/traceability.go`,
reached from both `VerifyRepository` and `VerifyAssignedSections`, and from
`main -> run -> traceability.VerifyAssignedSections` in `cmd/tracecheck`.

## The blind spot, measured rather than asserted

`TestUnmeasuredCoverageIsAScannerBlindSpotNotAnAbsenceOfObligation` re-derives
the class from the pinned document. It is **19 of 157** pinned headings:

| Heading | body lines | Heading | body lines |
| --- | ---: | --- | ---: |
| 7.3 | 48 | 15.2 | 21 |
| 10.8.1 | 65 | 16.6 | 8 |
| 13.5 | 21 | 16.7 | 40 |
| 13.12 | 22 | 18.2 | 54 |
| 13.14.1 | 192 | 19.4 | 97 |
| 13.14.2 | 272 | appendix-a | 163 |
| 13.14.3 | 52 | appendix-b | 22 |
| 13.14.4 | 227 | appendix-c | 30 |
| 13.14.5 | 99 | 14.3 | 106 |
| 14.6 | 53 | | |

Every one has a substantive body. Not one is a heading with nothing to
discharge, so revision 1's "carries no obligation of its own" was false for all
19, not just the two that were bound. The test fails if any of them ever scores
below 8 body lines, which would mean a genuinely empty heading had appeared and
the reasoning needed re-checking.

## The three bindings, re-declared not re-bound

`unowned_sections` was **not available**: `expectedCatalogSectionBindings`
requires an implementation owner for `section:7.3`, `section:15.2` and
`section:13.14.5` (probed directly; 24 catalog-required bindings in total, all
three present). So each keeps its binding and carries an honest gap, and each is
routed out of the admitted set by the `full`-only rule. No sliver was re-bound
to a friendlier symbol.

| Key | Production | Coverage | Gap says |
| --- | --- | --- | --- |
| `section:7.3` | `catalog.go:ForRelease` | `unmeasured` 0/0 | scanner sees no clause; the closed Provider Manifest is implemented nowhere, only a catalog row naming its URN |
| `section:15.2` | `catalog.go:ForRelease` | `unmeasured` 0/0 | scanner sees no clause; the 19-row exit-code registry is implemented nowhere, the only `os.Exit` calls being `exit(1)` in cataloggen and tracecheck |
| `section:13.14.5` | `core_records.go:validateSessionEventV2` | `unmeasured` 0/0 | topically the right symbol, but the scanner cannot measure how much of the section it discharges |

§13.14.5 survives with a justification rather than a re-binding, as the brief
allowed, and its gap says exactly why.

## Gap quality, tightened

Was `len(gap) >= 32 && strings.Contains(gap, display)`. "Section 9.2 is not
fully covered here yet." satisfied it, and `Contains(gap, "6.5")` also matched
"6.55". Now `verifyGapDiscloses` requires:

- length ≥ 32;
- the section named as a **whole identifier** (`mentionsSection`, boundary
  regexp) — so 6.55 is not a mention of 6.5, and 13.15 is not a mention of 13.1;
- the **production declaration the binding is registered to**, named in the gap.

One shipped gap failed the new rule and was corrected: `section:1.6` is bound to
`ErrInvalidScalar` but its gap named only the package. Nothing else changed.

**Residual, stated not implied:** a sentence that names both its section and its
declaration and still says nothing useful is admitted. The gate cannot decide
otherwise. Recorded in README.md and in the `verifyGapDiscloses` doc comment.

## specdoc guard

`internal/specdoc` reaching non-test code made "the embedded document never
reaches the ax command" a claim with nothing behind it.
`TestEmbeddedDocumentNeverReachesAProductBinary` reads the module import graph
from source with `go/parser` (no toolchain, no network, build constraints
ignored so reachability is over-approximated), refuses any `main` outside
`tracecheck` that can reach the package, and **proves the detector** by planting
a `cmd/ax` importing `internal/traceability`. `cataloggen` was in the first
draft's allow list and was removed: it touches specdoc from its test binary
only. `TestModuleHasNoProductCommandYet` fails when `ax` lands, forcing that
decision to be deliberate.

## Anti-vacuity: the gate narrowed, not deleted

Four mutants, each confirmed **present in the file** by grep before measuring,
each restored from a `cp` backup afterwards and the restoration confirmed.

| # | Narrowing | Effect | Caught by |
| --- | --- | --- | --- |
| 1 | admission re-admits `coverageUnmeasured` | `-section 15.2` exits **0** — the bug, reproduced | 4 arms of `TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers` + `...BindsGranularScope...`; and at the command entry point `TestRunRefusesEveryAssignedSectionThatOnlySlivers` (7.3, 13.14.5, 15.2) + `TestRunReportsExactCoverageAndFailsClosed` |
| 2 | `verifyGapDiscloses` drops the declaration requirement | padded gap admitted | `TestUnmeasuredBindingWithAnHonestGapIsStillAccepted/the_gap_is_padded...`, `TestPlantedSliverRedens.../an_unmeasured_binding_pads_its_gap...` |
| 3 | `mentionsSection` reverts to `strings.Contains` | 6.55 passes for 6.5 | `TestMentionsSectionRequiresAWholeIdentifier` (5 rows), plus 2 gap mutants |
| 4 | specdoc allow list admits `cmd/ax` | planted product main not reported | `TestEmbeddedDocumentNeverReachesAProductBinary` |

Mutant 1 is the load-bearing one: it restores revision 1's behavior exactly and
both production entry points go red.

Registry-level mutants added to `TestPlantedSliverRedensTheProductionEntryPoints`
(each drives `VerifyRepository` **and** `VerifyAssignedSections`): unmeasured
binding drops its gap; pads its gap; relabels itself `unevidenced`; relabels
itself `full`; a gap names a neighbouring identifier; an unowned entry pads its
gap with a neighbouring identifier.

Positive arm preserved and extended: the synthetic `full` 11/11 Section 6.3
binding is still admitted, `-section 6.2` is still green, and
`TestUnmeasuredBindingWithAnHonestGapIsStillAccepted` proves the tightened
`unmeasured` bucket is still *declarable* — the gate is not vacuously strict, it
refuses only at admission.

## No assertion weakened

Three assertions changed direction and none was deleted:

- `TestVerifyAssignedSectionsBindsGranularScopeToOwnersAndExecutableCases` kept
  §6.2 and moved §13.14.5 into the refusal table, plus a new assertion that the
  *pair* `{6.2, 13.14.5}` is refused as a whole.
- `TestRunAdmitsOnlyAssignedSectionsWhoseBindingDischargesTheWholeSection` same.
- `TestRunReportsExactCoverageAndFailsClosed` kept its green assigned-scope run
  on §6.2 and gained a refusal assertion for the old `{6.2, 13.14.5}` pair,
  including "emits no success output".
- `replaceOwnershipProductionDeclaration` now rewrites the gap alongside the
  declaration, which *narrows* that mutant to one thing instead of letting the
  coverage check fire before the production-owner check it exists to reach.

Everything the reviewer accepted is untouched: the measured denominator, the
clause-enumeration attacks, `unowned_sections` as disclosure-not-exemption, the
digest pin over every new field.

## Measured state

```
traceability ok: contracts=60 normative_sections=36 acceptance_cases=43 fixtures=30 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=48 full=1 partial=0 sliver=1 unevidenced=43 unmeasured=3 unowned=2 clauses_discharged=2/394
```

Ratio unchanged at **2/394**; the three bindings moved from an admitted bucket
to a refused one. Assigned-scope admission now succeeds for `-section 6.2` and
nothing else — disclosed in README.md as a thin positive arm rather than hidden.

## Validation, real exit codes

| Command | Exit |
| --- | ---: |
| `gofmt -l .` | 0, no output |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1` | 0 (11/11 packages ok) |
| `go test ./... -count=1 -cover` | 0 |
| `go generate ./internal/catalog` | 0 |
| `cataloggen ... -check` | 0 |
| `tracecheck` | 0 |
| `tracecheck -section 6.2` | 0 |
| `tracecheck -section 7.3` | **1**, expected: unmeasured is refused |
| `tracecheck -section 13.14.5` | **1**, expected: unmeasured is refused |
| `tracecheck -section 15.2` | **1**, expected: unmeasured is refused |

Coverage after the change: specdoc 100.0%, traceability 86.4%, tracecheck 88.5%.

## Not fixed here, reported

- **§6.5 production defect.** SPEC.md:2585 requires `required_capabilities` to
  default to the platform lane minimum; `internal/config/validation.go` accepts
  only an empty default. Named in §6.5's gap and in README. It is a Configuration
  defect, not a traceability one, and belongs on its own board item.
- **Allowed-but-not-required sections** are neither forced to be owned nor
  disclosed unowned. Pre-existing; unchanged by this rework.
- **The named acceptance case's adequacy.** Unchanged residual: a complete
  enumeration pointed at one weak test is admitted. Stated in README.
