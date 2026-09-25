# TASK-260924-2zboyz — Change Request revision 2 review verdict

Verdict: **changes_requested → to-dev**. Reviewed the immutable candidate tree `711151d55db8db29eeba1b9ed25842f07c459911` against base `5e40826725a0880897d6bd8321c8fa8e062620d3` in `RUN-260924-405b2b`. The CR patch SHA-256 is `ee0427d08b229e2360be6150dc428497b913d2a713d1d2a94530be4788a23931`; all 17 changed scratch-copy blobs matched the pinned candidate tree. Revision 1 did not have a review verdict, so every `repeat-of` is `none`.

## Findings array

```json
[
  {
    "id": "mode-authority-unbound",
    "row": "mode provenance / no relabel",
    "invariant": "A read-back manifest's staged or live mode is fixed by ReadAuthority and cannot be relabeled (pinned SPEC v0.7.0 lines 3843, 10575–10581, 10753–10754).",
    "mechanism": "internal/clonereadback/readback.go:113–129 and :260–276 accept a caller-controlled mode with no ReadAuthority or expected-mode input; :379–454 verifies only the recomputable self digest. The committed TestModeBindsIdentity even admits the resealed opposite mode.",
    "reproductions": [
      "Copy evidence/repros/review_relabel_test.go into internal/clonereadback/ of an archive copy of candidate 711151d and run go test -count=1 ./internal/clonereadback/ -run '^TestReviewResealedStagedRelabelRefuses$'; expected exit 1 with 'resealed staged evidence admitted as live'. Raw log: review-relabel.log."
    ],
    "severity": "bypass",
    "priority": "P1",
    "repeat-of": "none"
  },
  {
    "id": "report-identical-read-refs",
    "row": "report staged/live distinctness",
    "invariant": "Staged and live read-back manifests are distinct, and the validation report aggregates both (pinned SPEC v0.7.0 lines 10575–10582, 10753–10763).",
    "mechanism": "internal/clonereadback/report.go:215–223 and :361–369 parse the two digest references independently but never refuse equality; decideValid at :137–169 ignores their relation. The same digest can fill both slots with valid=true.",
    "reproductions": [
      "Copy evidence/repros/review_equal_refs_test.go into internal/clonereadback/ of an archive copy of candidate 711151d and run go test -count=1 ./internal/clonereadback/ -run '^TestReviewValidationReportNeedsDistinctReadRefs$'; expected exit 1 with 'same digest admitted as both staged and live with valid=true'. Raw log: review-equal-refs.log."
    ],
    "severity": "bypass",
    "priority": "P1",
    "repeat-of": "none"
  },
  {
    "id": "envelope-census-gap",
    "row": "closed member census / gate proof",
    "invariant": "Every member, including schema, schema_version, and self-ID, has missing, extra, duplicate, miscase, and wrong-type attacks through the public decoder, and every gate has a narrowing killed alone.",
    "mechanism": "internal/clonereadback/members_test.go:296–416 derives only input-struct members for the full attack grid. The envelope delta is tested for presence and deletion, and schema/version value neighbors, but no miscase/wrong-type/duplicate sweep with a literal member verdict. An alias mutant that normalizes 'Schema' before closure admits a forbidden member while the entire committed clonereadback suite stays green. The conformance matrix also lists owner-nesting, extensions, scalar, UTF-8 and character bounds without shipped narrowing rows.",
    "reproductions": [
      "Apply the single-site Schema-alias narrowing in evidence/repros/envelope_alias_plant.py to internal/clonereadback/decode.go on candidate 711151d, then run go test -count=1 ./internal/clonereadback/; expected exit 0 (survivor). The companion temporary TestReviewPlantSchemaAliasAdmitted confirms the plant admits the forbidden spelling; raw log: envelope-alias-admission.log."
    ],
    "severity": "bypass",
    "priority": "P2",
    "repeat-of": "none"
  }
]
```

## Surface sweep

The supplied brief had no separate surface table; I used the candidate's gate × entry conformance table as the floor, then added report reference distinctness, importer outcomes, and hygiene. Each row has one result.

| Surface row | Result | Attack and evidence |
|---|---|---|
| Closed members, including nested EvidenceObject | broken | Envelope-alias narrowing survives full package suite; F3. Derived-field attacks held. |
| Schema/version literals | held | `TestEnvelopeLiterals`; shipped N-schema and N-version both killed twice. |
| Mode vocabulary | held | `TestModeVocabularyOracle`; N-mode killed twice. |
| Evidence kind vocabulary | held | `TestEvidenceKindVocabularyOracle`; N-kind killed twice. |
| Native-ID equality | held | `TestNativeIdentityEqualityOracle`; length-equality narrowing killed twice. |
| Valid iff applicable checks | held | 1536 outcome combinations through Build and Decode; own one-false narrowing killed. Seven report booleans are all required and non-null, so applicability is fixed true at this schema entry. |
| Character string bounds | held | `TestNativeSessionStringEdges`, `TestMediaTypeStringEdges`; own 128→129 media narrowing killed. |
| Parsed-head order/count | held | `TestParsedHeadIDsSweep`; duplicate narrowing killed twice. |
| Evidence order | held | `TestEvidenceOrderSweep`; duplicate narrowing killed twice. |
| Evidence count | held | `TestEvidenceCountEdges`; 65537 narrowing killed twice. |
| Findings count | held | `TestFindingsCountEdges`; Build and Decode 4097 narrowings killed twice. |
| uint53/number model | held | `TestScalarGates`; 2^53 Build narrowing killed twice. |
| Scalar UUID/digest | held | `TestScalarGates` through Build and Decode. Shipped narrowing proof absent, included in F3. |
| Owner nesting | held | `TestOwnerDelegation`; own ignored-input owner call killed. Shipped call removal plant failed at compile, included in F3. |
| Extensions | held | `TestExtensionsClosed` through both entries; shipped narrowing proof absent, included in F3. |
| JCS identities | held | Independent Python SHA-256/JCS recomputation of both fixtures; relabel-old-ID narrowing killed twice. |
| Mode authority/no relabel | broken | Resealed staged evidence decodes as live; F1. |
| Build UTF-8 | held | `TestBuildUTF8Reflection` and heads/extensions probes; shipped narrowing proof absent, included in F3. |
| Owner-row census / call reachability | held | AST inventory and P-census/P-owner plants rerun twice; zero fidelity/plan row-carrying entries, digest references only. |
| Fork guard / literal member-list guard | held | Structural tests and three control plants rerun twice. |
| Report staged/live distinctness | broken | Equal digest references accepted with valid=true; F2. |
| Importer outcome grid | held | All five owner package Git trees, go.mod and go.sum are byte-identical base↔candidate; full candidate owner tests rerun. The new package has no base importer. See `importer-identities.csv`. |
| Config, formatting, Windows vet | held | `task-board.config.json` SHA-256 identical base↔candidate; gofmt clean; Windows vet exit 0. |

## Measured coverage and validation

The producer claimed 6/6 AC rows driven. I measured **4/6 fully driven** (identity, mode, valid rule, owner reachability); **3/6 satisfy** the current candidate. The closed-member row omits envelope miscase/wrong-type/duplicate attacks, the mode row fails, and the all-gates-narrowing row is incomplete. The 1536-combination valid oracle and both independent IDs did pass. All 24 shipped narrowings were killed in each of my two reruns, the applied neutral control survived both, and five structural plants went red twice. My own case-fold, failed-applicable-check, ignored-owner-input, and media-bound plants all failed a named test; the mode-output-swap survived its named mode test but fails another package test, so I record it as a note rather than a standalone finding.

Exact-tree validation: `go test -count=3 ./internal/clonereadback/` passed; `GOOS=windows GOARCH=amd64 go vet ./...` passed. The full `go test -count=1 -p 2 -timeout 8m ./...` attempt produced 45 passing package results, then exposed the expected `specpin` fixture dependency on `.task-board`, which the archive copy intentionally excluded. I stopped that invocation at 9 minutes under the shell bound before the last three packages. `go test -count=1 ./internal/specpin/` passed in the live Story worktree; `go test -count=1 -timeout 8m ./internal/traceability ./internal/traceability/cmd/tracecheck` passed in the exact candidate archive. `go test -count=1 -timeout 8m ./internal/tmuxserver/` also passed in the candidate archive. In total, every one of the 49 packages passed across the bounded calls; no single full-command green result is claimed because the archive excludes the board fixture required by `specpin` and the long command was stopped at its shell bound. The importer grid has no moved old-package class because the complete owner-package tree OIDs and module files are equal; this is a source-identity inference supported by the candidate test rerun, not a claim from the producer log.

## Notes

- The candidate also changes `internal/traceability/cmd/tracecheck/main_test.go` to copy `go.sum` into fixture roots. This is outside the new package despite the results document saying no landed file was touched. The change appears aimed at a suite timeout and does not alter the traceability registry; keep or remove it with an accurate scope explanation.
- Cross-input tuple, plan, projected-manifest, fidelity and object-store reconciliation belong to the final leaf as the producer states. I did not count those as failures in this leaf.
- The mode-output-swap plant (`own-plants/mode_result_swap.log`) survived `TestModeBindsIdentity` alone; `TestReadBackRoundTrip` would catch the returned-mode error in the full package. This is a narrow test-name gap, separate from F1.

## Rework scope (for the producer)

1. Bind each read-back mode to trusted `ReadAuthority` purpose (`target_staged` or `target_live`) at every exported production entry, or use an equally authoritative expected-mode input. Parsing caller-supplied authority JSON alone is self-minted evidence and does not close the gate. Recomputed JCS IDs alone do not carry the authority's stage. Add a named staged→live and live→staged regression test and an admitting narrowing per entry.
2. Refuse identical staged/live report digest references at Build and Decode. Add a named negative test and separate narrowings for the two entry paths.
3. Extend the reflection-derived member census so envelope members receive the complete attack grid and literal-code assertion. Add a single-site alias mutant that preserves the gate and is killed by the behavioral suite. Supply a killed narrowing for each still-unproved gate, including owner-nested, extensions, scalar, UTF-8 and character-bound entries, or state a justified bound for a gate delegated to a landed owner without inflating the measured ratio.
4. Keep this task's `task_delta` free of registry edits. Explain the tracecheck fixture edit or move it to its owning task. Re-run the configured suite and importer comparison on the revised exact tree.

## Task logbook entry

Revision 2 independently reproduced two mode/reference bypasses and one conformance-proof gap on the exact candidate. Source-identical owner imports, JCS identities, and the valid-bit oracle held under the attacks listed above. Route to the producer for focused rework; do not integrate this revision.
