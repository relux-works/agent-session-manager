# BUG-260902-2m7slg rev2 — reviewer mutation and attack log

Candidate tree `163d498df0669dc85c9cccaf258980dadadf0c35`, HEAD `2d0962c`
(`git rev-parse HEAD^{tree}` equals the Change Request `candidate_tree_oid`).

Every mutant was applied to an isolated `git archive HEAD` copy under
`.temp/BUG-260902-2m7slg/mutant/`, never to the worktree. Each substitution was
confirmed PRESENT with `grep -n` before measuring, and the file was restored
from a backup between mutants. The unmutated copy is green
(`go test ./internal/traceability/... -count=1` ok, `tracecheck` exit 0).

## Narrowing mutants — 7 of 7 killed

Every mutant NARROWS a live guard; none deletes one.

| # | Narrowing | Present-check | Killed by |
| --- | --- | --- | --- |
| A | admission `measured.Level != coverageFull` → `== coverageSliver` (admits partial/unevidenced/unmeasured) | `traceability.go:402` | `TestVerifyAssignedSectionsBindsGranularScopeToOwnersAndExecutableCases`, `TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers`, `TestRunReportsExactCoverageAndFailsClosed`, `TestRunRefusesEveryAssignedSectionThatOnlySlivers` |
| B | `coverageBucket` `discharged >= total` → `discharged*2 >= total` (partial reported as full) | `traceability.go:1087` | `TestCoverageBucketNamesTheMeasuredRatio`, `TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted` |
| C | per-clause "acceptance case is owned by this binding" check made unreachable | `traceability.go:1185` | `TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted` |
| D | `quoteBeginsAtLine` short-circuited to `true` (excerpt fidelity unverified) | `traceability.go:1237` | `TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted` |
| E | `verifyGapDiscloses` drops the "gap names the production declaration" clause | `traceability.go:997` | `TestUnmeasuredBindingWithAnHonestGapIsStillAccepted`, `TestPlantedSliverRedensTheProductionEntryPoints` |
| F | `verifyUnownedSections` drops the catalog-required-owner refusal | `traceability.go:1266` | `TestPlantedSliverRedensTheProductionEntryPoints` |
| G | `normativeKeywordPattern` narrowed to `\b(MUST NOT)\b` (denominator shrinks) | `traceability.go:970` | 12 tests across both packages, incl. `TestSectionClauseInventoryIsMeasuredFromThePinnedDocument`, `TestUnmeasuredCoverageIsAScannerBlindSpotNotAnAbsenceOfObligation`, `TestVerifyRepositoryAcceptsExactOwnership` |

A is the load-bearing one: with it applied `tracecheck -section 9.2` exits 0
against a 0/35 binding — revision 1's behaviour restored — and four tests at
both production entry points go red.

## Positive arm verified on shipped data, not only synthetically

`tracecheck -section 6.2` exits 0 against the unmodified shipped registry, and
`TestPlantedSliverIsReportedAndAnAdequateBindingIsStillAdmitted` builds an
11-of-11 Section 6.3 binding from the pinned document and requires admission.
The gate is not vacuously strict.

## Residual probed directly, and it is exactly the size README states

Reviewer attack: enumerate ALL three Section 6.5 clauses from the pinned
document, point every one at the single existing acceptance case
`config-versioned-readers`, declare `coverage: "full"`, drop the gap, recompute
and re-pin `reviewedOwnershipCanonicalSHA256`.

```
tracecheck            -> exit 0, full=2 ... clauses_discharged=5/394
tracecheck -section 6.5 -> exit 0, assigned_scopes=1
```

This is admitted. It is exactly, and only, the class README.md declares out of
scope ("A binding could enumerate every clause of a section and point all of
them at one weak test, and the gate would admit it"). It is not a silent
bypass: it requires editing the reviewed registry AND re-pinning the reviewed
projection digest, both of which are review-gated. No wider hole was found.

## Dodge paths probed and closed

- move a slivered section into `unowned_sections`: refused when the generated
  catalog requires an owner (mutant F proves the guard is live), and an unowned
  section is refused at assigned-scope admission with its gap.
- drop a section binding entirely: refused as "no scoped implementation owner"
  and by the catalog-required-binding check.
- claim `coverage` on a contract/fixture owner: refused by
  `verifySectionCoverage`.
- self-mint any new field: every one is inside `json.Marshal(registry)` and the
  reviewed digest; confirmed by the digest refusal firing before I re-pinned it.

## Repository validation re-run by the reviewer on the candidate tree

| Command | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l .` | exit 0, no output |
| `go test ./... -count=1` | exit 0, 11/11 packages ok |
| `go test ./... -cover -count=1` | exit 0; specdoc 100.0%, traceability 86.4%, tracecheck 88.5% — matches the producer's claim |
| `go generate ./internal/catalog` + `git status --short` | exit 0, no drift |
| `tracecheck` | exit 0, `bindings=48 full=1 partial=0 sliver=1 unevidenced=43 unmeasured=3 unowned=2 clauses_discharged=2/394` |
| `tracecheck -section 6.2` | exit 0 |
| `tracecheck -section 2.2 / 18.4` | exit 1, recorded unowned |
| `tracecheck -section 6.5` | exit 1, unevidenced 0/3 |
| `tracecheck -section 7.3 / 13.14.5 / 15.2` | exit 1, unmeasured 0/0 |
