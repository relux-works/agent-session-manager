# TASK-260924-3n78rv — Change Request revision 1 review

Verdict: **changes requested → to-dev** for candidate tree `2530f4d80c9bf7bee95fa21ad89d2456db31816e` over base `5bfbc78f68c7251b75c9ac25ab7342cda8652ebc`. The candidate leaves one authority-axis narrowing undetected by its committed suite and also lacks current trunk content required by the producer and reviewer briefs. The Change Request patch SHA-256 and a scratch-index snapshot reproduce its tree exactly; this is a candidate-base defect, not a moving-worktree observation.

The brief supplied no catalog surface table or review/free-hunt minute budgets. I swept the concrete surfaces named in its scope and conformance matrix. No previous revision exists, so there is no rework diff.

## Surface results

| Row | Result | Attack through production entry or candidate boundary |
| --- | --- | --- |
| Selection and branch routing | held | `SelectStrategy` 16-cell oracle and `SelectProfile` profile/refusal oracle; both harness runs killed selection narrowings. |
| Canonical item classification | **broken** | The committed named test covers five carrying kinds. An exact-blob narrowing makes `subagent_started` classify as `approval_response` only for authorization-like payload text; the named test and full candidate package suite pass, while the independent all-kind public-entry test fails. |
| Strategy × profile × item planning | held | `PlanItem` 975-cell literal oracle; independent profile, event-kind, and aborted-result narrowings failed named tests run alone. `PlanSession` counts and bounds exercised by named tests. |
| Visible authority and escaping | held | Independent corpus at three text positions across all 26 kinds through `ProjectVisibleText`, plus JSON round trip; authority-raising `sudo` narrowing failed the behavioral test alone and the AST guard alone. Token-preserving escape mutant failed twice in the producer harness. |
| Historical tools and source accounting | held | `PlanTargetEffects` zero fold checked for every independently swept event mapping; source-accounting narrowing failed the named test alone. |
| Importer outcome composition | held | Outcome grid `(package, entry, input)` rerun against base and candidate for nine owner packages; all 232 owner files are byte-identical, both public-entry package suites passed, and no owner input class moved. New planning entries have no base counterpart. |
| Candidate provenance and trunk composition | **broken** | Exact CR base is not descended from `origin/main=6d3bff9`; 81/81 non-board paths introduced by trunk since the Story fork have different blobs in the candidate tree. `repro-stale-trunk.sh` fails against the pinned OIDs. |

`held` means the named attacks did not reproduce; it is not proof of absence. No table row was left unattacked. The free hunt examined the exact tree, caller census, CR bytes, config identity, tool/fact registry composition, and independent fixture arithmetic; it produced no additional blocking finding.

## Findings

```json
{
  "findings": [
    {
      "id": "classification-text-axis-untested",
      "row": "Canonical item classification",
      "invariant": "Adversarial text at every Canonical Event kind and position cannot change the classified item or create an authorization-like event; a named committed test must kill a narrowing of this class at ClassifyItem.",
      "mechanism": "TestClassifyItemIgnoresPayloadText in internal/cloneplanning/classification_test.go sweeps only tool_call, user_message, instruction_snapshot, usage and opaque_reasoning carriers. It omits subagent_started and other event kinds. A one-member narrowing in ClassifyItem therefore survives the named test and the complete candidate package suite.",
      "reproductions": [
        {
          "test_file": "reviewer_probe_test.go",
          "command": "python3 repro-classification-gap.py <exact-candidate-archive-copy>",
          "expected_failure": "candidate named test exit 0, candidate package suite exit 0, independent TestReviewerIndependentAuthoritySweep exit 1 with subagent_started classified as approval_response",
          "logs": ["classification-gap-named.log", "classification-gap-package.log", "classification-gap-independent.log", "classification-gap-repro.log"],
          "candidate_tree": "2530f4d80c9bf7bee95fa21ad89d2456db31816e"
        }
      ],
      "severity": "bypass",
      "repeat-of": "none"
    },
    {
      "id": "stale-trunk-candidate",
      "row": "Candidate provenance and trunk composition",
      "invariant": "The candidate checkpoint must descend from current trunk, and every trunk-only path must be blob-equal to trunk before handoff and review.",
      "mechanism": "The producer's refresh-candidate attempts failed on a stale recovery intent, but CR-TASK-260924-3n78rv-1 was published from checkpoint 5bfbc78, whose merge base with origin/main 6d3bff9 is 0ca3e4c. Its candidate tree omits all 81 trunk-only non-board path updates.",
      "reproductions": [
        {
          "test_file": "repro-stale-trunk.sh",
          "command": "./repro-stale-trunk.sh",
          "expected_failure": "exit 1: trunk_ancestor_of_base_exit=1; 81/81 trunk-only non-board paths differ; AssertionError candidate lacks current trunk blobs",
          "log": "stale-trunk.log",
          "candidate_tree": "2530f4d80c9bf7bee95fa21ad89d2456db31816e",
          "base_commit": "5bfbc78f68c7251b75c9ac25ab7342cda8652ebc",
          "trunk_commit": "6d3bff999f75ed8585a7ef32074ab108f364ec51"
        }
      ],
      "severity": "bypass",
      "repeat-of": "none"
    }
  ]
}
```

## Notes

- The source spec gives exact vocabularies and authority/inertness rules; some profile mapping priorities are explicitly leaf-pinned choices in `TRACEABILITY.md`. The oracle pins those choices with literal expectations and the review attacks them at exported entries.
- No other production package imports `internal/cloneplanning` yet. The leaf defines the planning entries; assembly into the clone pipeline is a stated bound, consistent with the split Story.
- The producer's configured validation log reports `coverage_unit=exact_command_shard`, 30/30 green and `test_case_coverage=unknown`; I did not treat that line as complete test-case coverage. My exact-tree suite and target tests supply separate evidence.
- No fixed normative output digest appears in this leaf's fixtures. I independently recomputed its SHA-256 seed and JCS tuple bytes. The tuple fixture is semantic JSON input, not a claimed JCS encoding.

## Validation and measured coverage

The candidate's **17 of 18 behavioral AC rows are fully driven** by named production tests; the text-class × event-kind × position row is only partially covered at `ClassifyItem`. The explicit trunk-refresh delivery requirement also fails. The independent scratch sweep exercised all 26 Canonical Event kinds in `ClassifyItem` and `ProjectVisibleText`, five adversarial text classes including mixed script and a long payload, three field positions, and all plan item classes via the 975-cell oracle. The 232-file importer identity includes `clonefidelity`, `cloneplan`, `clonebundle`, `cloneproject`, `clonesnap`, `environ`, `scalar`, `sessadapter`, and `canonicaljson`.

- Exact-tree `go test -p 2 -count=1 ./...`: 47/47 packages passed.
- Exact-tree default `go test ./...`: interrupted after 9 minutes (exit 143) to honor the headless single-call bound under concurrent test load; 28/47 package results were green, 19 were unreported. This is **partial**, not a pass. All 47 packages passed separately in the complete `-p 2 -count=1` invocation above.
- `GOOS=windows GOARCH=amd64 go vet ./...`: passed.
- `go test -count=3 ./internal/cloneplanning/`: passed.
- Importer owner tests: nine packages passed on base and candidate.
- Producer harness rerun twice on an archive copy of the exact candidate: each run 48 `KILLED`, one harmless control `SURVIVED`, zero errors/mismatches, with raw per-plant logs.
- Five independently constructed narrowing plants: each killed by its named single test; harmless comment control survived.
- Scratch-index `git diff --cached --check`: passed; scratch-index tree equals CR candidate tree; live index untouched. `task-board.config.json` is blob-equal to trunk `6d3bff9`.

Raw commands, exits and logs are attached in `TASK-260924-3n78rv_review-evidence-rev1.tar.gz`.

## Rework scope (for the producer)

Add a committed all-kind adversarial text sweep at `ClassifyItem` and a narrowing mutant at an omitted kind such as `subagent_started` that the named test kills alone. Then resolve the stale `refresh-candidate` recovery intent through the managed task-board worktree flow, and republish a candidate whose checkpoint descends from current `origin/main`. Verify every trunk-only path is blob-equal to that base and rerun the tree-bound validation after refresh. Keep the existing passing gate tests; do not manually rewrite or commit the managed Story branch. A new reviewer cycle must examine the refreshed exact tree.
