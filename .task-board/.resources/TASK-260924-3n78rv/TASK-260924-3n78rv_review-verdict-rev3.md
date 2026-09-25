# TASK-260924-3n78rv — Change Request revision 3 review

Verdict: **changes_requested → to-dev**. Reviewed immutable candidate tree `2f250f4b111b45d49c5bf05f2cb9465166ea4ffc` over checkpoint `618d78de05450c40ec39df7c97073a6feecc8f81`; the attached patch SHA-256 matches `8721f4046eb6998675b3d635b570510683d914c7de4f55eded853d57235ced84`. The scratch-index snapshot including all 17 untracked paths equals the candidate tree. No registry or config file changed.

The prompt supplied no catalog surface table or minute budgets. I used the seven concrete rows in the revision 1 verdict, which cover this leaf. I attacked the rev1-to-rev3 rework first (all-kinds classification and trunk refresh), then swept the rows and made a free-hunt pass over public visible-text entries.

## Surface results

| Row | Result | Attack and observed result |
| --- | --- | --- |
| Selection and branch routing | held | 16-cell and profile oracles, plus harness selection narrowings; no reproduced deviation. |
| Canonical item classification | held | Independently injected six adversarial text classes into four positions at all 26 kinds through `ClassifyItem`, `PlanItem`, `PlanTargetEffects`, and `ProjectVisibleText` (624 cells). My `subagent_started` authority-text narrowing failed both the independent test and the committed all-kinds test when each ran alone. AST closed-fact guard and vocabularies were checked. This closes `classification-text-axis-untested` from rev1. |
| Strategy × profile × item planning | held | 975-cell committed whole-domain oracle; independent continuation, importer and aborted-call narrowings each failed a named production-entry test alone. Session tests and the full package suite passed. |
| Visible authority and escaping | **broken** | Independent all-kind corpus retained `user_context` and zero actions; authority-raising plant failed behavioral and AST tests alone. Direct `EscapeVisibleText` accepts invalid UTF-8 and changes its value; finding below. |
| Historical tools and source accounting | held | Independent semantic target-accounting plant failed `TestPlanTargetEffectsZeroSweep`; planned effects remained zero in the independent corpus. |
| Importer outcome composition | held | `(package, entry, input)` grid rerun: 232 files across nine reused owner packages byte-identical base/candidate; all nine package suites green on each. No existing input class moved. New `cloneplanning` entries have no base counterpart. |
| Candidate provenance and trunk composition | held | Trunk `6d3bff9` is ancestor of checkpoint; all 81 trunk-only non-board paths are blob-equal in candidate; scratch tree equals CR tree; `task-board.config.json` is byte-identical. This closes `stale-trunk-candidate` from rev1. |

No row is unreported or marked `not-attacked`. `held` describes only the attacks named here.

## Findings

```json
{
  "findings": [
    {
      "id": "escape-invalid-utf8-value-change",
      "row": "Visible authority and escaping",
      "invariant": "Text must be valid UTF-8 (pinned SPEC v0.7.0 line 256); an escaped typed field must decode to exactly its input or refuse. Every public entry must preserve that rule.",
      "mechanism": "internal/cloneplanning/visible.go:58-65: EscapeVisibleText passes its string directly to encoding/json.Marshal. Go JSON replaces invalid UTF-8 with U+FFFD and returns success. ProjectVisibleText validates UTF-8 first, but direct callers of the exported EscapeVisibleText entry bypass that validation. The existing EscapeVisibleText oracle supplies only valid strings.",
      "reproductions": [
        {
          "test_file": "reviewer_invalid_escape_test.go",
          "command": "cd <evidence>/own && go test -count=1 -run '^TestReviewerEscapeVisibleTextRejectsInvalidUTF8$' ./internal/cloneplanning/",
          "expected_failure": "exit 1: invalid UTF-8 byte FF is admitted and escaped as a JSON string containing U+FFFD instead of refusing",
          "logs": ["invalid-escape-repro.log"],
          "candidate_tree": "2f250f4b111b45d49c5bf05f2cb9465166ea4ffc"
        }
      ],
      "severity": "bypass",
      "repeat-of": "none"
    }
  ]
}
```

## Notes

- `TestReviewerVisibleProjectionRejectsInvalidUTF8Control` passes: the composed `ProjectVisibleText` entry rejects the same invalid byte. The failing direct helper is a separate exported production entry named in the conformance matrix. The defect is a value change, not merely a missing assertion.
- The configured CR validation transcript is bounded and has an explicit `... validation log truncated; 6570955 bytes omitted ...` marker. It prints 22 of 30 command/exit pairs and a system summary `required=30 green=30`. I treat the eight omitted command transcripts as **unverified from that log**, not as eight independently observed passes. My own full Go suite, Windows vet, package determinism, owner grid and mutation runs are separate evidence.
- Independent fixture recomputation: SHA-256 of `cloneplanning-store-fingerprint` is `96608f44b08c643d894cbd4d5e32f96c947d5693742cb35008cd583ff84c9586`; JCS bytes of the semantic environment tuple hash to `9f5854c1a4960bd0f52f8e36f6ae1b04c8527aba0ae04161aa1b973d1ace5eb5`. The source fixture's field order is not JCS and does not claim to be.
- No other production package imports `internal/cloneplanning` yet; assembly into the clone pipeline is a declared bound of this leaf.

## Validation and measured coverage

**17 of 18 behavioral AC rows fully driven.** The escaping row is partial: `ProjectVisibleText` rejects malformed UTF-8, but `EscapeVisibleText` admits it. The other 17 rows have named production-entry tests in the conformance matrix and were attacked as shown above.

- Exact-tree `go test -p 2 -count=1 ./...`: green, all packages.
- Exact `go test ./...`: exit 0, 48/48 packages green; complete raw output in `full-exact.log`.
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0.
- New all-kinds tests `-count=3`: exit 0.
- Base and candidate nine-package importer suites: exit 0 after adding the baseline README/config fixtures needed by canonicaljson tests. The initial sparse baseline run failed only because those two root files were omitted from the scratch copy; it was not treated as a product failure.
- Producer harness rerun twice on immutable-tree sparse copies: each run 50 KILLED, one neutral SURVIVED, zero mismatch/error, with raw per-plant logs.
- Five reviewer-authored narrowing plants: each failed its named test alone; one neutral comment control passed. The text-preserving authority plant also failed the AST guard alone.
- Scratch index `git diff --cached --check`: exit 0; resulting tree equals the immutable candidate.

## Rework scope (for the producer)

Make `EscapeVisibleText` refuse invalid UTF-8 before JSON marshaling (or use a validated typed text input shared with `ProjectVisibleText`). Add a named regression test that calls **both exported entries** with malformed UTF-8, requiring each to refuse. Ship a narrowing mutant at the direct escaping entry that skips validation for one invalid-byte class and is killed by the named test alone. Preserve the rest of the held rows; attach complete command evidence for any configured-suite commands relied on in the next review that the bounded CR log omits.
