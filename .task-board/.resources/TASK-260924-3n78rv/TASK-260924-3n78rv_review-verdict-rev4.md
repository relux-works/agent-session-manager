# TASK-260924-3n78rv — Change Request revision 4 review

**Verdict: changes_requested → to-dev.** I reviewed candidate tree `684f1c80208c29c6c7d2db63aedffe33bd576e9f` over checkpoint `618d78de05450c40ec39df7c97073a6feecc8f81`. A scratch index containing the 20 uncommitted paths produced exactly that tree; the CR patch SHA-256 is `df599294718baaccb45877cbcac52379032667b46ca17ae6aba3c32bc42b0e3b`. No registry or config path changed. The revision 3 `escape-invalid-utf8-value-change` bypass is fixed: the direct escape entry rejects the exact FF probe and all five malformed classes; the exported-entry census, structural guard, and `-count=3` rerun pass.

The brief supplied no explicit catalog surface table or minute budget. I used the seven rows persisted in the revision 3 review, swept all rows, and then tested the malformed mapping class in the target-effects free hunt. `held` means only the attacks named here did not reproduce.

## Surface results

| Row | Result | Attack and observed result |
| --- | --- | --- |
| Selection and branch routing | held | 16 signal cells and profile oracle; selection mutants killed. UTF-8 profile refusal passes. |
| Canonical item classification | held | Independent six-text corpus across 31 event items and four payload positions; 744 chained cells through `ClassifyItem`, `PlanItem`, `PlanTargetEffects`, and `ProjectVisibleText` passed. Committed all-kinds sweep and AST guard passed; subagent and usage narrowings killed. |
| Strategy × profile × item planning | held | 975-cell independent-spec oracle passed. Own aborted-tool and continuation-kind narrowings failed their named entry tests alone. |
| Visible authority and escaping | held | Own authorization-text authority escalation failed the public-entry test alone; direct malformed UTF-8 refusal, authority corpus, structural guard, and escaping mutants passed/killed as expected. This closes the revision 3 finding. |
| Historical tools and source accounting | **broken** | Own one-class target-accounting plant was killed, and valid plans fold to zero. A separate direct public-entry probe found `PlanTargetEffects` admits malformed disposition/reason pairs; finding below. |
| Importer outcome composition | held | Base and candidate source trees are blob-identical for 232 files in nine reused packages; base owner suites and candidate full suite passed. The `(package, entry, input)` grid in the conformance matrix has no moved owner input class; new `cloneplanning` entries have no base counterpart. Initial sparse base test lacked dependencies and failed setup; after adding the unchanged dependencies, the rerun passed. |
| Candidate provenance and trunk composition | held | `6d3bff9` is an ancestor of the checkpoint; all 362 trunk-delta paths were blob-equal in the candidate. Scratch index equals the CR tree; `task-board.config.json` blob is unchanged; exact-tree diff checks passed. |

## Findings

```json
{
  "findings": [
    {
      "id": "effects-reason-set-coupling-missing",
      "row": "Historical tools and source accounting",
      "invariant": "SPEC v0.7.0 §13.14.2 says every non-exact row names one or more reasons. An exact row carries no reasons in the landed clonefidelity record rule and in PlanItem's own output invariant. PlanTargetEffects says it validates each caller-supplied mapping before folding effects.",
      "mechanism": "internal/cloneplanning/effects.go:29-51 checks disposition and reason vocabulary and sorted order, but never couples reason cardinality to disposition. A caller can submit a semantic mapping with no reason, or an exact mapping with an operator_policy reason, and obtain successful target effects.",
      "reproductions": [
        {
          "test_file": "reviewer_probe_test.go",
          "command": "cd <candidate-tree-copy> && cp <evidence>/reviewer_probe_test.go internal/cloneplanning/ && go test -count=1 -run '^TestReviewerEffectsRejectMalformedMapping$' ./internal/cloneplanning",
          "expected_failure": "exit 1: both malformed mapping cells are admitted by PlanTargetEffects",
          "logs": ["own-probes.log"],
          "candidate_tree": "684f1c80208c29c6c7d2db63aedffe33bd576e9f"
        }
      ],
      "severity": "bypass",
      "repeat-of": "none"
    }
  ]
}
```

## Notes

- `PlanItem("archive_only", invalid-UTF8-profile, validItem)` refuses with the archive-branch message before reporting invalid UTF-8. This is a priority ambiguity when two invalid inputs coexist, so I recorded it as a note, not a blocking finding. The same malformed profile alone is refused with the literal UTF-8 marker at each tested text entry.
- The CR validation transcript contains an explicit truncation marker and omits raw output for some commands. I did not count omitted command transcripts as passes; the complete independent runs below are the verification basis.
- `internal/cloneplanning` still has no production caller outside this package. Assembly into the clone pipeline is a stated bound of this leaf.

## Validation and coverage

**18 of 18 named AC rows have production-entry tests; one additional public-entry validation class failed the independent negative probe.** The target-effects reason-set gate is the uncovered class, despite the positive 556-mapping zero sweep. The exact candidate's `go test ./...` passed (48 packages); Windows `GOOS=windows GOARCH=amd64 go vet ./...` passed; the 11 new test groups at `-count=3` passed. The nine base owner package suites passed after the sparse-copy dependency repair; all 232 owner files are byte-identical to candidate. A reviewer-owned JCS+SHA-256 script reproduced fixture hashes `96608f44…c9586` and `9f5854c1…a5eb5`. The producer mutation battery rerun yielded 52 KILLED, one harmless SURVIVED, no errors; four reviewer-authored narrowings were each killed by a named test run alone, and a reviewer-authored harmless control survived. Full raw logs and the independent probe file are in the attached review evidence archive. Scratch-index `git diff --cached --check` and immutable-tree `git diff --check` passed.

## Rework scope (for the producer)

Validate the disposition/reason cardinality coupling at the exported `PlanTargetEffects` entry. Add a committed negative test for `semantic` with no reason and `exact` with a reason, anchored to the pinned rule and owner behavior. Ship at least one narrowing mutant for each admitted class, killed by its named test alone. Keep the held rows and the rev3 UTF-8 regression intact; include the complete command evidence in the next handoff.
