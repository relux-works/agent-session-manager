# TASK-260830-1jmmqn review verdict, Change Request revision 1

**Verdict: changes_requested → to-dev.** Reviewed base `547ea289e411bea803f4cd47b6314e4bb558ceb0`, candidate tree `c4fbfc7922287a2abfaadedbb37eba93308b1cdc`, patch SHA-256 `52bf2613d8bef85f7ea0dbff1b68da1ccf5268f8fa6d4e4247982a2a8bef2b45`. An alternate-index snapshot of the live worktree equaled the candidate tree. This is a `task_delta`; only `internal/cloneplan/*` changed, with no registry edit or planning-semantics implementation. No earlier reviewer verdict exists, so every `repeat-of` is `none`.

The brief supplied no catalog surface table or wall-clock review/free-hunt budget. I swept the gate groups in the attached conformance matrix. `held` means only the named attack did not reproduce on this tree. The exact candidate rejects the inputs in the three narrowing demonstrations below; the finding is that its committed evidence fails to kill a narrowed gate.

## Surface sweep

| Surface | Result | Attack/result |
| --- | --- | --- |
| Closed member sets and strict frame | **broken** F1 | Titlecase member normalization plant survived the existing package suite, then a public `DecodeProjectionPlan` test failed. Strict duplicate framing plants were killed. |
| Operation ordering and DAG | **broken** F2 | Four-operation equal-sequence dependency plant survived the existing package suite, then a public `BuildProjectionPlan` test failed. Independent two-operation equal-edge plant was killed. |
| Resource and entry nullability | **broken** F3 | A blob with two missing blob-only members bypassed the branch check and panicked in `BuildProjectedObjectManifest`; the existing package suite stayed green, while the new public-entry test failed. Resource blob-null plant was killed. |
| Sorted arrays, scalar bounds, plan constants, policies, vocabulary | **held** | Shipped harness rerun twice; own key-bound-plus-one and second-value constant plants killed. The one-million-mapping upper boundary was not attacked at a public entry because a suitable fixture was missing; the submitted edge test reaches only the factored helper. |
| Omit-self JCS identities | **held** | Independent Python JCS + SHA-256 recomputation matched both emitted fixture IDs. |
| Importer outcomes and composition | **held** | Base and candidate tests passed for six reused owner packages; all 171 files in those packages are byte-identical, so no class moved in their entries. The producer's table is package-level rather than the requested `(package, entry, input)` grid. |
| Exact-tree hygiene | **broken** F4 | `git diff --check BASE CANDIDATE` exits 2 on a new blank line at EOF in the mutant harness. The producer's plain `git diff --check` did not inspect its untracked files. |

## Findings

```verdict-findings
{
  "findings": [
    {
      "id": "member-case-variant-uncounted",
      "row": "Closed member sets and strict frame",
      "invariant": "The committed member census must derive all members from Go types and kill a narrowing that accepts a miscased required member through DecodeProjectionPlan.",
      "mechanism": "internal/cloneplan/members_test.go:87 hand-lists the member census; TestMemberCensusMiscased at line 280 tries only strings.ToUpper. Reflection in utf8_reflection_test.go covers only string-valued UTF-8 fields. A Titlecase Strategy normalization at decodeStrictObject survives the existing suite.",
      "reproductions": [
        {
          "test_file": "review_casefold_test.go",
          "command": "go test -count=1 -run '^TestReviewTitlecaseRefusal$' ./internal/cloneplan on exact candidate: exit 0",
          "expected_failure": "on casefold-plant.patch applied to candidate: expected test failure, exit 1, Titlecase member admitted; logs casefold-candidate.log and casefold-casefold.log",
          "pinned_blobs": [
            "sha256:c17ab7089cc1f2cc22c9290fb39845fe5d357d26e8044551a9e120ed32984ca2",
            "sha256:d08ed7f25035f5f6c705416e353568db838ac17c7fd97b843afc8cb3258bdc93"
          ]
        },
        {
          "test_file": "go test -count=1 ./internal/cloneplan on the casefold mutant without the review test: exit 0",
          "command": "log casefold-existing-suite.log",
          "expected_failure": "go test -count=1 ./internal/cloneplan on the casefold mutant without the review test: exit 0; log casefold-existing-suite.log",
          "pinned_blobs": [
            "sha256:c17ab7089cc1f2cc22c9290fb39845fe5d357d26e8044551a9e120ed32984ca2",
            "sha256:d08ed7f25035f5f6c705416e353568db838ac17c7fd97b843afc8cb3258bdc93"
          ]
        },
        {
          "test_file": "coverage_audit.py <exact candidate copy>: expected exit 1 for non-reflection-derived member census",
          "command": "log coverage-audit.log",
          "expected_failure": "coverage_audit.py <exact candidate copy>: expected exit 1 for non-reflection-derived member census; log coverage-audit.log",
          "pinned_blobs": [
            "sha256:c17ab7089cc1f2cc22c9290fb39845fe5d357d26e8044551a9e120ed32984ca2",
            "sha256:d08ed7f25035f5f6c705416e353568db838ac17c7fd97b843afc8cb3258bdc93"
          ]
        }
      ],
      "severity": "bypass",
      "repeat-of": "none"
    },
    {
      "id": "dag-large-invalid-uncounted",
      "row": "Operation ordering and DAG",
      "invariant": "Every dependency graph through five operations is classified against the independent oracle, including invalid four- and five-operation graphs.",
      "mechanism": "internal/cloneplan/dag_test.go:108 enumerates all graphs only through three operations; lines 144+ enumerate only valid four- and five-operation graphs. The four-operation self-edge exception in dagfour-plant.patch survives the existing suite.",
      "reproductions": [
        {
          "test_file": "review_dagfour_test.go",
          "command": "go test -count=1 -run '^TestReviewFourNodeSelfEdgeRefusal$' ./internal/cloneplan on exact candidate: exit 0",
          "expected_failure": "on dagfour-plant.patch: expected test failure, exit 1, four-node self edge admitted; logs dagfour-candidate.log and dagfour-dagfour.log",
          "pinned_blobs": [
            "sha256:8f11ee28b8a2f685a9f6b36bb8c7c6bab6a0f4f13a4a9592c46f3dcdecca43f3",
            "sha256:d6a6f394a7cf7020c755ed6a424b92b1b5d4dd31fc54703cf03fc9b71553dcd0"
          ]
        },
        {
          "test_file": "go test -count=1 ./internal/cloneplan on the dagfour mutant without the review test: exit 0",
          "command": "log dagfour-existing-suite.log",
          "expected_failure": "go test -count=1 ./internal/cloneplan on the dagfour mutant without the review test: exit 0; log dagfour-existing-suite.log",
          "pinned_blobs": [
            "sha256:8f11ee28b8a2f685a9f6b36bb8c7c6bab6a0f4f13a4a9592c46f3dcdecca43f3",
            "sha256:d6a6f394a7cf7020c755ed6a424b92b1b5d4dd31fc54703cf03fc9b71553dcd0"
          ]
        },
        {
          "test_file": "coverage_audit.py <exact candidate copy>: expected exit 1 for no all-graph sweep through five operations",
          "command": "log coverage-audit.log",
          "expected_failure": "coverage_audit.py <exact candidate copy>: expected exit 1 for no all-graph sweep through five operations; log coverage-audit.log",
          "pinned_blobs": [
            "sha256:8f11ee28b8a2f685a9f6b36bb8c7c6bab6a0f4f13a4a9592c46f3dcdecca43f3",
            "sha256:d6a6f394a7cf7020c755ed6a424b92b1b5d4dd31fc54703cf03fc9b71553dcd0"
          ]
        }
      ],
      "severity": "bypass",
      "repeat-of": "none"
    },
    {
      "id": "entry-presence-combinations-uncounted",
      "row": "Resource and entry nullability",
      "invariant": "The blob/directory kind × each blob-only field-presence combination must be checked through both manifest entries and refuse structurally.",
      "mechanism": "internal/cloneplan/nullability_test.go:120 toggles byte_count, blob_id, and blob_descriptor_id together. TestEntryBranchLiterals covers each missing alone, leaving two-missing combinations unmeasured. The branchcombo-plant.patch bypasses the guard for one such combination; BuildProjectedObjectManifest then dereferences nil.",
      "reproductions": [
        {
          "test_file": "review_branchcombo_test.go",
          "command": "go test -count=1 -run '^TestReviewBlobDoubleMissingRefusal$' ./internal/cloneplan on exact candidate: exit 0",
          "expected_failure": "on branchcombo-plant.patch: expected test failure through a panic, exit 1; logs branchcombo-candidate.log and branchcombo-branchcombo.log",
          "pinned_blobs": [
            "sha256:041984a72c5e9cd8b59dfb533ffe2e56733370e3f15b0299f00e2de634dddfa3",
            "sha256:64ee2403895a0b12fcbd22b377ba9a82a8c46748f35b53fcf0701f351f5a1082"
          ]
        },
        {
          "test_file": "go test -count=1 ./internal/cloneplan on the branchcombo mutant without the review test: exit 0",
          "command": "log branchcombo-existing-suite.log",
          "expected_failure": "go test -count=1 ./internal/cloneplan on the branchcombo mutant without the review test: exit 0; log branchcombo-existing-suite.log",
          "pinned_blobs": [
            "sha256:041984a72c5e9cd8b59dfb533ffe2e56733370e3f15b0299f00e2de634dddfa3",
            "sha256:64ee2403895a0b12fcbd22b377ba9a82a8c46748f35b53fcf0701f351f5a1082"
          ]
        },
        {
          "test_file": "coverage_audit.py <exact candidate copy>: expected exit 1 for the absent independent three-field grid",
          "command": "log coverage-audit.log",
          "expected_failure": "coverage_audit.py <exact candidate copy>: expected exit 1 for the absent independent three-field grid; log coverage-audit.log",
          "pinned_blobs": [
            "sha256:041984a72c5e9cd8b59dfb533ffe2e56733370e3f15b0299f00e2de634dddfa3",
            "sha256:64ee2403895a0b12fcbd22b377ba9a82a8c46748f35b53fcf0701f351f5a1082"
          ]
        }
      ],
      "severity": "robustness",
      "repeat-of": "none"
    },
    {
      "id": "exact-tree-diff-check-missed",
      "row": "Exact-tree hygiene",
      "invariant": "The candidate passes the required lint/diff hygiene check on the reviewable tree, and the reported validation covers that same tree.",
      "mechanism": "internal/cloneplan/testdata/mutant_harness.py:1325 adds a blank line at EOF. The CR validation log ran plain git diff --check against an untracked-only package, which produced exit 0 without checking those files.",
      "reproductions": [
        {
          "test_file": "diff-check",
          "command": "git diff --check 547ea289e411bea803f4cd47b6314e4bb558ceb0 c4fbfc7922287a2abfaadedbb37eba93308b1cdc on exact candidate blobs: expected failure exit 2, new blank line at EOF",
          "expected_failure": "compare CR validation log's plain git diff --check exit 0",
          "pinned_blobs": [
            "sha256:7cb3be7525b463b25cf427048aaeef3a4d79f01f36b6adec64c3cd88b988dc30"
          ]
        }
      ],
      "severity": "regression",
      "repeat-of": "none"
    }
  ],
  "notes": [
    "The case-only unknownMember plant survived because a separate missing-member gate still refused; it is not a finding.",
    "The importer table lacks explicit (package, entry, input) keys; six owner packages are byte-identical and their base/candidate tests passed, so no moved class was found.",
    "The 1,000,000 mapping upper edge is checked at a helper, not a public entry; it remains an unchecked edge of AC row 16."
  ],
  "surface_results": [
    {
      "row": "Closed member sets and strict frame",
      "result": "broken"
    },
    {
      "row": "Operation ordering and DAG",
      "result": "broken"
    },
    {
      "row": "Resource and entry nullability",
      "result": "broken"
    },
    {
      "row": "Sorted arrays, scalar bounds, plan constants, policies, vocabulary",
      "result": "held"
    },
    {
      "row": "Omit-self JCS identities",
      "result": "held"
    },
    {
      "row": "Importer outcomes and composition",
      "result": "held"
    },
    {
      "row": "Exact-tree hygiene",
      "result": "broken"
    }
  ],
  "free_hunt": []
}
```

## Notes

- The case-only `unknownMember` plant in `own_plants.py` survived because the independent missing-member gate still refused the document. It was dropped as a non-finding; `case_member.log` records this.
- The producer's importer table is not keyed by `(package, entry, input)`. My byte-identity comparison and base/candidate package runs found no moved class, so this formatting gap did not become a separate blocking finding.
- The mapping upper count of 1,000,000 is tested at `checkMappingCount` in `counts_internal_test.go`, but not sealed/decoded at the production entries. This is an unchecked edge of AC row 16 and should be closed or reported as an explicit entry-level bound in the next revision.
- The producer's claimed 18/18 leaf rows counts rows 1, 5, 14, and 16 as fully driven. My measured complete production-entry coverage is **14 of 18 AC rows**. The rest have named tests but lack the requested derived census, whole graph/domain grid, or entry-level upper edge. Planning semantics belong to TASK-260924-3n78rv and are excluded from this denominator.
- No baseline production bypass was found in the attacked inputs. The three surviving plants show that the submitted tests permit distinct weakenings; they are not claims that the exact candidate currently admits those inputs.

## Validation and rework scope (for the producer)

The exact tree passed `go test -count=1 ./...`, `GOOS=windows GOARCH=amd64 go vet ./...`, `go test -count=3 ./internal/cloneplan/...`, and independent JCS recomputation for the plan and manifest fixtures. Base and candidate reused-owner package tests passed, with byte-identical code. The shipped 117 narrowing plants and one applied harmless control were rerun twice by this reviewer; see the two run summaries and per-plant logs in the evidence tar. `gofmt -l` returned no Go files. The exact-tree `git diff --check` failed as above.

Add a reflection-derived member census covering all schema types and multiple miscasing forms; commit the titlecase refusal test. Enumerate invalid as well as valid four/five-operation dependency graphs or provide an equivalent generated whole-domain proof that kills the four-node self-edge plant. Sweep all independent blob-only member-presence combinations through Build and Decode with structured-refusal assertions and kill the two-missing plant. Close or explicitly bound the million-mapping production-entry edge, expand the importer table to the requested triple key, remove the blank line, and run diff hygiene against the snapshotted candidate rather than the live untracked diff. Update the measured coverage ratio and conformance matrix from the resulting tests. Reuse this same task/CR rework path; no separate research or architecture decision is needed.
