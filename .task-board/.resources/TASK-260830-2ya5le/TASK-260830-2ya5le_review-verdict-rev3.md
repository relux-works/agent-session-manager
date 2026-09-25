# TASK-260830-2ya5le review verdict — Change Request revision 3

Candidate tree: `9870fb42e454a1163b4fc6abd778210260cf84d9`; base: `0ca3e4c26e2b275212796657f785b9b450f6174e`; reviewer run: `RUN-260923-61c37e`. The scratch-index tree equals the candidate. The CR patch SHA-256 is `9fe52f03a04fb1381ac6b1b2472f62c2c46397d24efddd233899031450f44935`. The revision 2 to 3 diff changes only `utf8_reflection_test.go`, the mutant harness, and TRACEABILITY; production blobs are unchanged.

## Verdict

**accepted** for revision 3. The revision 2 finding `unmeasured-build-utf8-sites` is closed: the reflection-derived input census has 33 nonempty string-carrying members, and the test drives 33 × 5 malformed-UTF-8 classes × Build/Decode = 330 report-entry cells. The three previously missed Build fields (`source_class`, `target_locator`, `forbid_reasons`) have literal ErrInvalid assertions and isolated admitting narrowings killed by that test alone. The approved `byte_counts` value-reconciliation stated bound remains explicit, with its spec/capture-manifest owner; it is not counted as driven. Measured AC coverage: **39 of 40 rows driven + 1 stated bound**.

```json
{
  "findings": [],
  "notes": [
    "The brief supplied no catalog surface table or review-budget marker. This verdict uses the revision 2 public-entry gate sweep as the floor. Held states only that the named attacks did not reproduce.",
    "The reflection grid starts at Build input types. Derived output-only reason_counts is separately exercised through DecodeFidelityReport by strict-frame and reason-count reconciliation tests; it is not a Build input. No malformed UTF-8 admission reproduced there."
  ]
}
```

## Surface sweep

| Surface | Result | Attack and evidence |
| --- | --- | --- |
| Closed disposition, profile, strategy, and reason vocabularies | held | Matched the 7/5/5/19 test oracles against pinned SPEC v0.7.0 §13.14.2; reran entry oracles three times. Own casefold-disposition narrowing was killed by `TestDispositionVocabularyOracle` alone. |
| Record rule grid and bounds | held | Reran `TestRecordGridOracle` three times; its 140 axis cells reach Build and Decode. Own source-class bound +1 narrowing was killed by `TestRecordFieldBounds` alone. Revision 2's synthesized-canonical attack remains applicable because production is byte-identical. |
| Report identity and branch rules | held | Independently emitted target and archive reports, removed only `fidelity_report_id`, encoded sorted compact JSON with Python, and recomputed SHA-256; both matched exactly. Branch tests ran in the full suite. |
| Aggregate reconciliation | held | Counts, reasons, kind and block tests ran three times. Own single-cell content-block drift narrowing was killed by `TestContentBlockBreakdown` alone. `byte_counts` value reconciliation is the approved stated bound; shape and uint53 remain tested. |
| JSON framing, members, scalar bounds, sorted arrays and nullability | held | Full `go test ./... -count=1` passed (45 packages). Existing named negative tests cover both entries; revision 2 already attacked these unchanged production sites. |
| Build UTF-8 refusal sites | held | `TestUTF8RefusalReflectionGrid` passed at all 330 cells. Shipped eight new Build narrowings were each killed alone in two independent harness runs. Own overlong `source_class` narrowing was killed by the grid alone. |
| Stable refusal surface | held | The grid asserts `errors.Is(ErrInvalid)` and literal member-specific Build details; Decode asserts the strict-frame literal. The five registry predicates reject all five malformed classes. |
| Importer compatibility | held | Independently reran base and candidate `go test -json` over clonebundle, environ, sessadapter, scalar, canonicaljson: 14,711 keyed outcomes on each side and zero moved outcomes. Initial partial archive lacked README/config and failed; after adding those exact base/candidate files, both complete reruns passed and matched. |
| Scope and hygiene | held | Ownership registry, README and `task-board.config.json` equal base. No sibling-leaf contract was implemented. `git diff --check`, gofmt, Windows vet and full suite passed. |
| Mutation evidence | held | Shipped harness: 70 KILLED + applied neutral SURVIVED in each of two reruns, with 71 raw subprocess logs per run. Four independent narrowings killed; own applied neutral control survived. |

## Validation identity

- Full `go test ./... -count=1`: exit 0, 45 package result lines; reviewer log complete.
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0.
- New and relevant oracle tests `-count=3`: exit 0.
- Importer grid: base 14,711 / candidate 14,711 `(package, entry, input)` verdicts, zero moved classes; both runs exit 0.
- Independent JCS/SHA-256: target `sha256:21ae31dd7ca60257fa7ff14a1a6e442d3ad0faa2debf6321557fc624d920d89e`; archive `sha256:881a040afe53927d748805b4a4c3ffed3a0cacb01de97b3297093abeaf60a0a2`; both equal the sealed IDs.
- Shipped harness rerun twice: 71/71 expected results each, with raw logs and exit codes. Independent plants: four KILLED, one SURVIVED control.
- Producer evidence manifest rechecked after full extraction; all listed hashes matched. CR construction log ends with `required=30 green=30 failed=0 missing=0`. The candidate patch digest matches the immutable CR record.

Free hunt produced no blocking finding. The tests establish these named cases only, not absence of all defects.
