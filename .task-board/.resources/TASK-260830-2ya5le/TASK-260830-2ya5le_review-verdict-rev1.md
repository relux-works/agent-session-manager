# TASK-260830-2ya5le review verdict — Change Request revision 1

Candidate tree: `cb244c15c2911d582c02564680c212fb00a9e27b`; base `0ca3e4c26e2b275212796657f785b9b450f6174e`. Reviewer run `RUN-260923-2c3826`.

## Surface sweep

The reviewer brief did not supply a catalog surface table. I use the candidate's gate census grouped by public entry, with each group below recorded as it is attacked.

| Surface | Result | Attack / evidence |
| --- | --- | --- |
| Closed disposition, profile, strategy, and reason vocabularies | held | The committed oracle tests were run at `-count=3` through `Valid*`, `BuildFidelityReport`, and `DecodeFidelityReport`; case, whitespace, near-miss, non-string, and extension inputs did not reproduce. `determinism.log`. |
| Record rule grid | held | `TestRecordGridOracle -count=3` drove the seven dispositions, five reason cardinalities, and both presence bits through Build and Decode; no unexpected admission reproduced. `determinism.log`. |
| Report identity and branch rules | held | `TestReportIdentityIndependent -count=3` passed through both entries. Independent recomputation and branch attack recorded below. |
| Aggregate reconciliation | broken | `TestByteCounts` explicitly admits a one-cell byte drift over identical rows at `report_aggregates_test.go:337-339`; see finding `unbound-byte-aggregate`. |
| JSON framing, closed members, scalar bounds, sorted arrays, and branch nullability | held | `go test ./internal/clonefidelity -count=3` passed all new tests. Independent Python JCS/SHA-256 recomputation of a target and an archive report matched both claimed IDs (`independent-jcs.log`); this assertion uses Python `json.dumps(sort_keys=True,separators=(',',':'),ensure_ascii=False)` and `hashlib`, not candidate digest code. |
| Stable refusal surface | held | The full package rerun includes member, vocabulary, numeric, strict-frame, and branch refusal cases. No unstructured failure reproduced through either public report entry. |
| Importer compatibility | held | I independently ran base and candidate `go test -json` across clonebundle, environ, sessadapter, scalar, and canonicaljson. Parsing `(package, top-level test entry, subtest input)` yields 14,706 shared cases, zero moved outcomes, zero additions/removals (`importer-grid.log`). The new clonefidelity package has no base entry. |
| Scope and hygiene | held | The exact diff contains 21 paths; `task-board.config.json` and `internal/traceability/ownership.v0.7.0.json` are byte-identical to base. No sibling report/plan contract or command capability was added. Scratch-index tree equals the CR candidate OID. Windows vet and full `go test ./... -count=1` passed. |
| Mutation evidence for implemented gates | held | Shipped harness rerun twice: 62/62 narrowings KILLED in each run; one applied neutral control SURVIVED in each run. Every plant has a raw log with process exit (`mutant-summary.log`). Four reviewer plants also KILLED alone: casefold disposition, extension shadowing a core reason, source-class bound +1, and event-kind one-cell drift; a line-count-preserving comment control SURVIVED (`own-mutants/`). |

## Evidence identity and scope

The live uncommitted worktree stages through a scratch `GIT_INDEX_FILE` to the exact CR tree OID. The producer archive's 170 manifested entries matched SHA-256, and all 31 command logs end with `exit=0`; the CR validation log ends after command 30. I reran `go test ./... -count=1`, `GOOS=windows GOARCH=amd64 go vet ./...`, all new package tests at `-count=3`, and the base/candidate importer grid myself. The two independent Python JCS recomputations matched. No prior review revision exists.

No catalog surface row was supplied in the reviewer brief, and no review/free-hunt budget marker was supplied. The grouped rows above cover the candidate's public-entry gate census. The aggregate row remains broken; every other swept row is held only for its named attacks.

## Verdict

**changes_requested → analysis** (P1 contract bypass; no earlier revision). **39 of 40 acceptance behaviors verified.** The producer's `40 of 40` claim counts the byte-aggregate row even though its named test asserts admission of the opposite behavior.

```json
{
  "findings": [
    {
      "id": "unbound-byte-aggregate",
      "row": "aggregate reconciliation",
      "invariant": "The task requires byte_counts to derive from disposition rows and a one-cell disagreement to be refused through both report entries.",
      "mechanism": "internal/clonefidelity/report.go:394-409 builds caller-supplied ByteCounts after shape validation; report.go:804-809 decodes it after shape validation. Neither entry has a row byte measure or lower evidence input. Identical rows and evidence IDs therefore admit byte_counts.exact=0 and byte_counts.exact=1. The committed TestByteCounts at report_aggregates_test.go:337-339 deliberately asserts this admission, while the matrix still counts the AC row as covered.",
      "reproductions": [
        {
          "test_file": ".temp/TASK-260830-2ya5le/review_byte_drift.go (included in review evidence archive)",
          "command": "go run .temp/TASK-260830-2ya5le/review_byte_drift.go",
          "expected_failure": "BuildFidelityReport must refuse one-cell byte drift over identical rows; instead the probe panics on admission (exit 1).",
          "log": "byte-drift.log"
        },
        {
          "test_file": ".temp/TASK-260830-2ya5le/review_decode_byte_drift.go (included in review evidence archive)",
          "command": "go run .temp/TASK-260830-2ya5le/review_decode_byte_drift.go",
          "expected_failure": "DecodeFidelityReport must refuse the sealed one-cell drift; instead the probe panics on admission (exit 1).",
          "log": "decode-byte-drift.log"
        }
      ],
      "severity": "bypass",
      "repeat-of": "none"
    }
  ],
  "notes": [
    "SPEC v0.7.0 section 13.14.2 lists byte_counts as an aggregate, but FidelityDispositionRecord has no byte measure. The producer disclosed that limitation. The required row-derived byte invariant cannot be established by the current Build/Decode signatures; ownership of the missing measure or a narrowed acceptance clause needs resolution before implementation rework."
  ]
}
```

## Rework scope (for the producer)

Resolve the report's byte-count authority with the specification owner: either supply an independently verifiable byte measure/linked evidence to both production validation paths, or revise the task acceptance clause explicitly so the report treats byte counts as unverified claims. Do not add a check that guesses bytes from digest IDs or row count. After the contract is settled, add an entry-level negative test for one-cell drift and a narrowing mutant for each implemented path; correct the AC ratio and the matrix/TRACEABILITY claims. The vocabulary, record, identity, branch, importer, and existing non-byte aggregate work need no unrelated rewrite.
