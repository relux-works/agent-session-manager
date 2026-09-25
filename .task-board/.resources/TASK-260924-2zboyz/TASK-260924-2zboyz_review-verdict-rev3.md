# TASK-260924-2zboyz — Change Request revision 3 review verdict

Verdict: **changes_requested → to-dev**. Reviewed CR-TASK-260924-2zboyz-3 revision 3, base `5e40826725a0880897d6bd8321c8fa8e062620d3`, exact candidate tree `f4478515c9182f3c3961337f7ea71782d033fa8e` in `RUN-260924-b025e5`. A scratch index over the live uncommitted candidate produced that exact tree OID. The board patch SHA-256 matched `a0bd9ed973fb051f9160bdd1af784b619da1c35c81dc267a2a24d34f28cb1976`. The diff since revision 2 was inspected first; it removes the tracecheck fixture edit and adds authority, report-reference, and envelope-census work.

## Findings array

```verdict-findings
{
  "findings": [
    {
      "id": "report-sibling-provenance-unbound",
      "row": "Report validated sibling provenance",
      "invariant": "The validation report aggregates two distinct validated staged and live read-back manifests, not caller-created stand-ins that merely repeat their IDs and mode labels (pinned SPEC v0.7.0 §13.14.2, lines 10753–10763).",
      "mechanism": "internal/clonereadback/report.go:192–208 checkReadRefs verifies only ManifestID and Mode on caller-supplied exported ReadBackManifest structs. BuildValidationReport at :217 and DecodeValidationReport at :359 accept those structs without establishing their decoded or sealed provenance; every other field can be zero.",
      "reproductions": [
        {
          "test_file": "evidence/review_forged_test.go",
          "command": "go test -count=1 ./internal/clonereadback/ -run '^TestReviewForgedReadSiblingsRefused$'",
          "expected_failure": "Exit 1: Build accepted caller-minted read IDs with no read-back manifest bytes; Decode accepted caller-minted read IDs with no read-back manifest bytes (both independent entry paths).",
          "pinned_blobs": [
            "sha256:34378334a937aeee5415307f770b69aeb1553b25b4a56a8b118aad8529d06796",
            "sha256:1fbe2f29170404b22a13589d1e3baf1a7c6a1033adbf7934ed2ebf38941bd084",
            "sha256:fdaa3dc31e639fd38d14a96635a7214d64b1a6577dc3148942006b3bc691982f"
          ]
        }
      ],
      "severity": "bypass",
      "priority": "P1",
      "repeat-of": "none"
    }
  ],
  "notes": [
    "Generated mutant_harness.cpython-314.pyc blob should be removed during rework.",
    "Read-back authority freshness and caller-chain provenance are stated bounds of this leaf."
  ],
  "surface_results": [
    {
      "row": "Closed members, including envelope and EvidenceObject",
      "result": "held"
    },
    {
      "row": "Schema/version literals",
      "result": "held"
    },
    {
      "row": "Mode vocabulary",
      "result": "held"
    },
    {
      "row": "Evidence kind vocabulary",
      "result": "held"
    },
    {
      "row": "Native-ID equality",
      "result": "held"
    },
    {
      "row": "Valid iff applicable checks",
      "result": "held"
    },
    {
      "row": "Character string bounds",
      "result": "held"
    },
    {
      "row": "Parsed-head order/count",
      "result": "held"
    },
    {
      "row": "Evidence order",
      "result": "held"
    },
    {
      "row": "Evidence count",
      "result": "held"
    },
    {
      "row": "Findings count",
      "result": "held"
    },
    {
      "row": "uint53/number model",
      "result": "held"
    },
    {
      "row": "Scalar UUID/digest",
      "result": "held"
    },
    {
      "row": "Owner nesting",
      "result": "held"
    },
    {
      "row": "Extensions",
      "result": "held"
    },
    {
      "row": "JCS identities",
      "result": "held"
    },
    {
      "row": "Mode authority/no relabel",
      "result": "held"
    },
    {
      "row": "Build UTF-8",
      "result": "held"
    },
    {
      "row": "Owner-row census / call reachability",
      "result": "held"
    },
    {
      "row": "Fork guard / literal member-list guard",
      "result": "held"
    },
    {
      "row": "Report staged/live reference relation",
      "result": "held"
    },
    {
      "row": "Importer outcome grid",
      "result": "held"
    },
    {
      "row": "Config, formatting, Windows vet",
      "result": "held"
    },
    {
      "row": "Report validated sibling provenance",
      "result": "broken"
    }
  ],
  "free_hunt": []
}
```

## Surface sweep

The brief has no separate surface table, so I used the candidate’s gate and composition rows, importer/hygiene rows, and the sibling provenance row above as the floor. `held` means only the named attacks did not reproduce.

| Surface row | Result | Attack / evidence |
|---|---|---|
| Closed members, including envelope and EvidenceObject | held | `TestMemberCensusUnknownMissing`, `Miscased`, `WrongTypes`, `Duplicated`; envelope alias narrowing killed. |
| Schema/version literals | held | `TestEnvelopeLiterals`, shipped schema/version narrowings. |
| Mode vocabulary | held | `TestModeVocabularyOracle`; archived neighbor refused. |
| Evidence kind vocabulary | held | `TestEvidenceKindVocabularyOracle`; native neighbor refused. |
| Native-ID equality | held | `TestNativeIdentityEqualityOracle`; same-length mismatch refused. |
| Valid iff applicable checks | held | `TestValidWholeDomainOracle`, 1536 combinations through both entries; failed marker narrowing killed. |
| Character string bounds | held | `TestMediaTypeStringEdges`, `TestNativeSessionStringEdges`, `TestParsedHeadIDsSweep`; edge+1 narrowings killed. |
| Parsed-head order/count | held | `TestParsedHeadIDsSweep`; duplicates, disorder, 1025 refused. |
| Evidence order | held | `TestEvidenceOrderSweep`; duplicate pair refused. |
| Evidence count | held | `TestEvidenceCountEdges`; 65537 refused. |
| Findings count | held | `TestFindingsCountEdges`; 4097 refused at both entries. |
| uint53/number model | held | `TestScalarGates`; 2^53 Build narrowing killed, Decode delegated to environ. |
| Scalar UUID/digest | held | `TestScalarGates`, `TestDecodeScalarStrings`; malformed values refused. |
| Owner nesting | held | `TestOwnerDelegation` through four entries; owner calls reached. |
| Extensions | held | `TestExtensionsClosed` through four entries; open keys refused. |
| JCS identities | held | Independent Python SHA-256/JCS recomputation over fixture documents. |
| Mode authority/no relabel | held | Purpose × claim × entry grid, resealed opposite mode refused. |
| Build UTF-8 | held | `TestBuildUTF8Reflection`; invalid bytes refused. |
| Owner-row census / call reachability | held | AST census and negative owner delegation; fidelity/plan row-carrying entries absent. |
| Fork guard / literal member-list guard | held | Structural tests and control plants. |
| Report staged/live reference relation | held | `TestReportReadRefRelationOracle`; equal, swapped and missing claims refused. |
| Importer outcome grid | held | Base/candidate source and dependency trees identical for five owner packages; candidate tests rerun. |
| Config, formatting, Windows vet | held | Config byte-identical; diff check and Windows vet passed. |
| Report validated sibling provenance | broken | Caller-minted ID/mode-only structs with invented digest IDs admitted at both public report entries; finding above. |

## Validation and coverage

Measured AC coverage: **6 of 6 named rows driven** through the exported production entries by the rerun tests; **5 of 6 held** under this review, because the report aggregation row admits caller-minted sibling structs. The source-derived member census covers 16 read-back, 5 evidence-row, and 23 report members, envelope included; its source has no hand-typed member list. The valid oracle runs 1536 combinations with all seven boolean checks applicable in this schema. The owner-entry AST census is empty for fidelity/plan row-shaped data, and the live package has no imports of either owner; tuple, binding, finding, scalar, extension, and JCS owners are invoked at the relevant entries. The split leaf makes cross-input tuple and plan/fidelity reconciliation stated bounds for the final Story leaf.

Independent reruns on the exact tree:

- `go test -count=1 ./...`: exit 0, 49 packages `ok`, 0 `FAIL` (raw `full-go-test.log`; the slow tracecheck package completed at 523.895s).
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0; `go test -count=3 ./internal/clonereadback/`: exit 0; gofmt on candidate Go files and `git diff --check`: clean.
- Five owner importer package suites: exit 0. The 498-row `(package, entry, named test input)` importer grid records identical base/candidate package tree OIDs (and marks entries without a directly named test) for environ, scalar, canonicaljson, clonebundle and sessadapter; `go.mod` and `go.sum` OIDs are equal too, so no old-package input class moved.
- Own single-site plants: five behavioral weakenings were killed alone (`alias-mode`, `relabel-live`, `valid-marker-decode`, `owner-object-bypass`, `head-bound-decode`), and an applied neutral comment survived. Raw commands, exits and failures are in `own-plants/*.log`.
- Own Python canonicalization plus SHA-256, with only the self-ID removed, matched all three staged, live and report fixture identities byte-for-byte (`independent_jcs.py`, `independent-jcs.log`).
- The report sibling provenance reproduction failed its own named test at both exported entries with explicit admission messages; all standard tests remained green.

Shipped harness rerun: all 47 rows were rerun twice against the exact candidate code. Run 1 was split into bounded sequential shards after 39 rows; its remaining eight rows ran after restoring the scratch package byte-for-byte from the live candidate. Run 2 ran as one bounded call. Both audits found 41/41 narrowing rows killed by named behavioral test failures, five structural plants red, the applied neutral control surviving, and zero missing, malformed, or mismatched row logs (`mutants-full-1-audit.log`, `mutants-full-2-audit.log`). These checks do not claim absence of other defects.

## Notes list

```json
[
  "The candidate includes a generated testdata/__pycache__/mutant_harness.cpython-314.pyc blob; remove it during rework.",
  "Read-back authority freshness and caller-chain provenance are stated bounds of this leaf; this review does not claim the exported authority struct itself is unforgeable."
]
```

## Rework scope (for the producer)

1. Bind report siblings to validated read-back documents at both Build and Decode. A public, caller-mutable `ReadBackManifest` carrying only ID and Mode cannot serve as evidence of a validated read. Preserve the distinct/swap checks and make the report refuse the forged-sibling reproduction.
2. Add a named regression test through each report entry and a narrowing per entry that admits the forged-sibling class when run alone.
3. Keep `task_delta` free of registry edits, remove the generated pyc, rerun the importer comparison and configured suite, and attach the evidence for another review.

## Task logbook entry

Revision 3 closes the previously reproduced mode relabel, equal reference, and envelope alias cases. The new report sibling binding compares fields on public structs without proving they came from a validated read-back manifest. Both public report entries accept invented digest IDs and mostly zero-valued siblings without any read-back manifest bytes. Route to development rework; do not integrate this revision.
