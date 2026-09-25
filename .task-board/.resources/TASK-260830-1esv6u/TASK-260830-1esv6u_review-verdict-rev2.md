# TASK-260830-1esv6u — review verdict, Change Request revision 2

**Verdict: changes_requested → to-dev.** Reviewed exact candidate tree `63ed292fcbfae0b40f614d79036bbd4cf1ccc94e` against base `5b7876be29cf578ff02583660d77fc10944e855c`; the live scratch-index snapshot independently reproduced that tree before and after validation. The rev1 and rev2 patch resources are byte-identical. No rev1 reviewer verdict resource was present, so every finding has `repeat-of: none`. CR rev2 patch SHA-256 is `410309180becdfce509cf82651f4a074be0959c727518aa8358e93c8e9234d6e`.

## Surface sweep

| Row | Result | Attack and evidence |
|---|---|---|
| Native evidence → candidate → canonical chain | **broken** | Public `Reconcile` admitted swapped and double-claimed raw evidence (`source-link-probe.log`). |
| Generated reconciliation product | **broken** | Source-bound coverage audit found only 10 unique capture vectors at each N=4,5,6, with two read outcomes per run (`property-coverage-audit.log`). |
| Owner call reachability | held | Removing `cloneplan.DecodeProjectionPlan` from `Reconcile` killed both `TestEntryOwnerReachability` and the public-entry acceptance test (`owner-bypass-plant.log`). |
| Refusal reachability and token-preserving behavior | held | Candidate harness rerun: 58 narrowings and four token-preserving plants killed; applied comment control survived (`harness.log`, raw harness logs). This row asserts only those plants. |
| Decoded registry clause edges | **broken** | All five Story acceptance cases have binding entries but zero decoded clause edges; tracecheck still exits 0 (`registry-clause-audit.log`, `tracecheck.log`). |
| README pin and merged trunk | held | README figure plant failed the pin test; canonical struct digest re-derived equal; 42 trunk-only changed paths blob-equal; tracecheck green (`readme-plant.log`, `digest-struct.log`, `trunk-merge.log`). |
| Importer outcome identity | held | Reran the keyed 11-row grid against checkpoint `f416d54` and candidate: every named production package is blob-equal, so these pure owner entries retain their outcomes; moved classes: none (`importer-grid.log`). |
| Other predecessor-package behavior beyond the checkpointed suites | not-attacked | Time budget; their production blobs are identical to the signed Story checkpoint and the exact-tree full suite passed. |

The prompt supplied no leaf surface table or numerical review/free-hunt budgets. The table above covers the scoped surfaces attacked; `held` does not claim absence of other defects.

## Required findings array

```json
[
  {
    "id": "source-evidence-chain-mismatch",
    "row": "Native evidence → candidate → canonical chain",
    "invariant": "A fidelity row for a captured candidate must claim that candidate's tier-1 raw evidence, with no raw item double-counted across rows.",
    "mechanism": "internal/clonereconcile/reconcile.go:447-451 checks each SourceEvidenceIDs digest for membership in the global rawSet, but never compares it to rawByCandidate[row.SourceItemKey] or checks row-level reuse.",
    "reproductions": [
      "source_link_probe_test.go (copied as internal/clonereconcile/reviewer_source_link_test.go into a git archive of tree 63ed292): go test ./internal/clonereconcile -run '^TestReviewerSourceEvidenceMustFollowCandidateChain$' -count=1 -v; expected FAIL because Reconcile admits the swap and double-count subtests; observed exit 1, both admitted (source-link-probe.log)."
    ],
    "severity": "bypass",
    "repeat-of": "none"
  },
  {
    "id": "reconciliation-product-not-enumerated",
    "row": "Generated reconciliation product",
    "invariant": "The generated property enumerates capture sets through N=6 and crosses each with all nine staged/live read-back outcomes.",
    "mechanism": "internal/clonereconcile/property_test.go:500-569 exhausts only N<=3. At N=4..6 it creates five uniform vectors plus 55 periodic mixes (only five distinct mixes) and executes two outcomes per run.",
    "reproductions": [
      "reviewer_property_coverage.py: python3 reviewer_property_coverage.py from the exact candidate root; expected exit 1 for incomplete enumeration; observed 10/625, 10/3125, and 10/15625 unique vectors at N=4,5,6 and two outcomes per run (property-coverage-audit.log)."
    ],
    "severity": "bypass",
    "repeat-of": "none"
  },
  {
    "id": "story-cases-have-no-clause-edges",
    "row": "Decoded registry clause edges",
    "invariant": "Each of the five Story acceptance cases appears in a decoded clause list, and tracecheck rejects an unlinked case.",
    "mechanism": "internal/traceability/registry_rederivation_test.go:219-259 explicitly expects zero clauses for section:13.14.2 and checks only binding-level acceptance_cases; internal/traceability/ownership.v0.7.0.json has no clause edge for any of the five cases. The production tracecheck still accepts this registry.",
    "reproductions": [
      "reviewer_registry_clauses.py: python3 reviewer_registry_clauses.py from the exact candidate root; expected exit 1 on missing decoded edges; observed five of five missing (registry-clause-audit.log).",
      "go run ./internal/traceability/cmd/tracecheck; expected nonzero for the same unlinked registry under the task contract, observed exit 0 (tracecheck.log)."
    ],
    "severity": "bypass",
    "repeat-of": "none"
  }
]
```

## Notes

- The story-final leaf claims 6 of 6 AC rows with named production-entry tests. I confirmed the six mappings, but only 4 of 6 are substantiated after the source-link and missing-negative-shape attacks; rows 2 and 4 need rework. This is a measured result, not a claim that the other four prove all possible inputs.
- `go test ./... -count=1` passed on the exact tree (51 packages); `GOOS=windows GOARCH=amd64 go vet ./...` passed; targeted new tests with `-count=3` passed. Baseline green did not detect finding 1.
- Registry canonical digest `ab7b66cc6d85df8ed7eac2aa5cd7b7fa635db20a3e5092824490b435f94ed3f1` matches its reviewed constant. The README coverage plant reddened. Trunk-only paths are blob-equal to trunk.
- The importer comparison is against the final leaf's checkpoint `f416d54`, not the Story's trunk base: clonefidelity, cloneplan, and clonereadback do not exist at the trunk base. The 11 pure owner production surfaces are byte-identical between checkpoint and candidate.
- The producer's configured validation log reports 30/30 green commands. My own exact-tree full Go test, Windows vet, harness, tracecheck, digest, importer, README plant, and determinism runs are in the attached evidence. I did not independently rerun every command of the 30-command configured suite.

## Rework scope (for the producer)

1. **P1:** Bind each non-synthesized fidelity row's source evidence to its own candidate/raw chain in `Reconcile`; reject a swapped, foreign, missing, or double-claimed raw digest with literal code and detail. Add a public-entry regression test and an admitting narrowing mutant that fails alone.
2. **P2:** Enumerate the full N<=6 capture-class product and all nine read-back outcomes per set with the independent oracle. Keep the generated coverage count/assertion executable so sampling or periodic duplicates cannot be reported as exhaustive.
3. **P2:** Make decoded clause-level traceability for all five Story cases satisfy the stated registry contract, with a check that fails when one edge is absent. If the present RFC-keyword clause model cannot represent the pinned declarative sentence, document the precise blocker and route a contract decision rather than claiming a zero-clause binding satisfies the requirement.

One verdict carries all three mechanisms. No code was changed in the live Story worktree.
