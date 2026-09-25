# TASK-260830-1jmmqn review verdict — Change Request revision 3

**Verdict: changes_requested → to-dev (P2).** Reviewed base `547ea289e411bea803f4cd47b6314e4bb558ceb0`, exact candidate tree `e7646ecd9b3d00f9d69ae1feb63985100f8ffed9`, and patch SHA-256 `ecb0b7c26b52a4f11129d48def70c2486573412dbdf481ae59bd76d7332bfaf1`. The 24-path patch bytes match `git diff --binary` for those OIDs. This is a `task_delta`; the registry and board configuration are unchanged. Planning semantics remain in TASK-260924-3n78rv.

The candidate itself refuses the tested invalid `materialization_intent="copy"`. The finding is that its submitted tests and mutant battery do not detect a narrow change that admits it. The producer's `clone2` narrowing is killed, but each independent intent gate widened to admit `copy` passes the entire submitted `cloneplan` package suite. The build-side widening also passes `TestPlanConstants`. A new public-entry scratch test fails on that mutant. No product-code edit was made in this review.

## Surface sweep

| Surface | Result | Attack and result |
| --- | --- | --- |
| Closed member sets and strict frame | held | Reflection census present; a new `Fidelity_profile` case-normalization plant was killed by `TestMemberCensusMiscased` alone. |
| Operation ordering and DAG | held | All graphs through four operations and the five-operation edge classes run; an own four-operation self-edge narrowing was killed by `TestDAGWholeDomainCensus` alone. |
| Resource and entry nullability | held | Full 48-cell entry grid and no-panic path run; an own descriptor-only branch narrowing was killed by `TestEntryNullabilityGrid` alone. |
| Sorted arrays, scalar bounds, plan constants, policies, vocabulary | broken: F1 | An own mode-bound +1 plant was killed by `TestNumberModel`. A second-value `materialization_intent="copy"` plant survived `TestPlanConstants` and the full submitted package suite. |
| Omit-self JCS identities | held | Independent Python sorted-key canonical serialization and SHA-256 match both emitted fixture IDs; fixtures contain no floats or non-ASCII strings. |
| Importer outcomes and composition | held | Re-ran six reused-owner package suites on base and candidate; all pass. The 22 `(package, entry, input)` rows have byte-identical owner code, so no moved class was inferred. |
| Exact-tree hygiene | held | `git diff --check BASE CANDIDATE` exits 0; `gofmt -l` is empty; Windows vet exits 0. |

The rev1 `member-case-variant-uncounted`, `dag-large-invalid-uncounted`, `entry-presence-combinations-uncounted`, and `exact-tree-diff-check-missed` mechanisms did not reproduce against revision 3. The earlier one-million mapping and importer-grid notes are addressed by named tests and a triple-keyed grid. There were no supplied review or free-hunt minute markers; no surface was left `not-attacked`.

## Findings

```verdict-findings
{
  "findings": [
    {
      "id": "plan-constant-second-value-uncounted",
      "row": "Sorted arrays, scalar bounds, plan constants, policies, vocabulary",
      "invariant": "Every value other than the pinned materialization_intent=clone constant must be refused at BuildProjectionPlan and DecodeProjectionPlan, and a narrowed gate admitting a second value must fail a named public-entry test.",
      "mechanism": "internal/cloneplan/bounds_test.go:447-452 probes prepare, clone2 and Clone only. internal/cloneplan/testdata/mutant_harness.py:617-634 widens the two independent plans.go intent gates only for clone2. Widening plans.go:160 to admit copy survives TestPlanConstants and the entire submitted package suite; the decode-side copy narrowing also survives the submitted package suite and is killed by the new reviewer test.",
      "reproductions": [
        {
          "test_file": "review_copy_test.go (attached scratch test)",
          "command": "go test -count=1 -run '^TestReviewCopyConstant$' ./internal/cloneplan on exact candidate source and tests plus the scratch test",
          "expected_failure": "candidate exits 0; with only plans.go build intent conditional widened to admit copy, exits 1 at TestReviewCopyConstant/build; with only decode conditional widened to admit copy, exits 1 at TestReviewCopyConstant/decode; see own-copy-candidate.log, own-copy-build.log, own-copy-decode.log",
          "pinned_blobs": ["candidate-tree:e7646ecd9b3d00f9d69ae1feb63985100f8ffed9", "plans.go:sha256:c2f7eb93506291635218beb49a589d8920bf32459d2906d73c9c3c74e311f02f"]
        },
        {
          "test_file": "internal/cloneplan/bounds_test.go",
          "command": "go test -count=1 -run '^TestPlanConstants$' ./internal/cloneplan, then go test -count=1 ./internal/cloneplan, with only the build intent conditional widened to admit copy",
          "expected_failure": "both SHOULD fail for a narrowing; both exit 0 instead. The decode-side widening also passes the entire submitted package suite. See own-second-intent.log, own-second-intent-whole-package.log, and own-second-intent-decode-whole-package.log. The exact candidate itself refuses copy through the new reviewer test.",
          "pinned_blobs": ["candidate-tree:e7646ecd9b3d00f9d69ae1feb63985100f8ffed9", "plans.go:sha256:c2f7eb93506291635218beb49a589d8920bf32459d2906d73c9c3c74e311f02f"]
        }
      ],
      "severity": "bypass",
      "repeat-of": "none"
    }
  ],
  "notes": [],
  "surface_results": [
    {"row":"Closed member sets and strict frame","result":"held"},
    {"row":"Operation ordering and DAG","result":"held"},
    {"row":"Resource and entry nullability","result":"held"},
    {"row":"Sorted arrays, scalar bounds, plan constants, policies, vocabulary","result":"broken","finding_ids":["plan-constant-second-value-uncounted"]},
    {"row":"Omit-self JCS identities","result":"held"},
    {"row":"Importer outcomes and composition","result":"held"},
    {"row":"Exact-tree hygiene","result":"held"}
  ],
  "free_hunt": []
}
```

## Validation and measured coverage

On the exact candidate tree: `go test ./...` passed; `GOOS=windows GOARCH=amd64 go vet ./...` passed; `go test -count=3 ./internal/cloneplan` passed; `git diff --check BASE CANDIDATE` passed; `gofmt -l internal/cloneplan` returned no paths. The independent digest recomputation matched plan ID `sha256:00404ea72f6d702baa25ba8f5c13e49404861cfd6f780a1ce3b0a4057e2bc297` and manifest ID `sha256:1df60fbc9fa0f356f1801eab1277ba4cb93ad07219dc13f2e53d9705115d2bca`. The shipped harness was rerun twice: 125/125 expected outcomes on each run: 124 KILLED and the neutral control SURVIVED. Four other own narrowings were killed; the own neutral comment control survived. Logs and scripts are in the attached evidence archive.

**Measured complete coverage: 17 of 18 leaf-schema AC rows.** The plan-constants row has a named test at the public Build/Decode entries, but the submitted suite does not kill the independent `copy` second-value narrowing. The other 17 rows have named entry tests and the attacks listed in the conformance matrix; this ratio states test coverage, not a proof that no other defect exists. Planning semantics are excluded by the Story split.

## Rework scope (for the producer)

Add a named committed public-entry regression for `materialization_intent="copy"` on both Build and Decode, using the literal refusal from the pinned spec. Add an isolated narrowing mutant for each independent intent gate that admits `copy` while keeping the gate present; each named killer must fail alone, with raw logs and subprocess exit. Broaden the constant evidence structurally or with an independent oracle so the submitted test does not depend on three hand-selected invalid strings. Update the conformance matrix and measured ratio, rerun the relevant suite and hygiene checks, then hand off the same task for another review. No separate research decision is needed.
