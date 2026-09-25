# TASK-260830-2ya5le review verdict — Change Request revision 2

Candidate tree `7bade702b4bf260219b199c5c1f6d2355c9e0ea0`; base `0ca3e4c26e2b275212796657f785b9b450f6174e`; reviewer `RUN-260923-3e6bbd`.

## Revision 1 finding and rev2 delta

The prior `unbound-byte-aggregate` finding is answered by the orchestrator's explicit stated-bound decision in `TASK-260830-2ya5le_rework-rev2.md`. Revision 2 changes only the bound-witness test, two harness test names, and TRACEABILITY wording; production blobs are unchanged. The test now deliberately witnesses admission of different `byte_counts` over identical rows; the matrix and results claim **39 of 40 AC rows driven + 1 stated bound**, with ownership assigned to spec clarification and the §13.14.1 capture-manifest leaf. No row-reconciliation claim remains for bytes. The bound is not treated as a new defect. My site census measures **38 of 40 fully driven, 1 partially driven, 1 stated bound**: row 33 lacks three Build UTF-8 refusal sites.

## Surface sweep

The brief supplied no catalog surface table or wall-clock budget marker. I used the public-entry gate census and the same grouped rows as the rev1 verdict. Each row below has one result; held means only the named attacks did not reproduce.

| Surface | Result | Attack / evidence |
| --- | --- | --- |
| Closed disposition, profile, strategy, and reason vocabularies | held | Compared the tests' independently typed 7/5/5/19 literals with pinned §13.14.2. Ran the entry-level oracles three times. Reviewer casefold `Exact` and extension-as-core narrowings were killed alone. |
| Record rule grid and bounds | held | `TestRecordGridOracle` covers 140 axis cells through Build and Decode, including absent evidence and synthesized canonical objects. Reviewer synthesized-canonical and source-class +1 narrowings were killed alone. |
| Report identity and branch rules | held | Independently emitted target/archive reports and recomputed each omitted-self JCS SHA-256 in Python; both matched byte-exactly. Branch oracle ran three times. |
| Aggregate reconciliation | held | Counts, reason, event-kind, and content-block perturbation oracles ran three times. Reviewer one-cell content-block drift narrowing was killed alone. `byte_counts` value reconciliation is the documented row-24 stated bound; its exact-key/uint53 shape tests ran. |
| JSON framing, closed members, scalar bounds, sorted arrays, and nullability | held | Full package suite and three-run targeted suite passed. The producer's tests exercise both report entries, unknown/missing members, duplicate members, number model, string edges, order, and branch flips. |
| Build UTF-8 refusal sites | broken | `TestBuildUTF8` covers 4 of 7 Build sites. Three one-case narrowings at `source_class`, `target_locator`, and `forbid_reasons` survive that named test; the `source_class` narrowing also survives the full package suite. Reviewer entry probes fail because malformed bytes are admitted. Finding `unmeasured-build-utf8-sites`. |
| Stable refusal surface | held | The unmutated candidate returns literal `ErrInvalid` at all three reviewer probes. Five other reviewer narrowings yielded test failures rather than unstructured failures. |
| Importer compatibility | held | Independently ran base and candidate `go test -json` over clonebundle, environ, sessadapter, scalar, canonicaljson; 14,706 shared `(package, entry, input)` cases, zero moved outcomes, all five packages passed on each tree. |
| Scope and hygiene | held | Scratch-index tree equals the CR candidate OID and patch SHA-256 matches the CR record. `task-board.config.json` and ownership registry equal base; no sibling contracts or capability advertisement changed. Full `go test ./... -count=1`, Windows vet, gofmt passed. |
| Mutation evidence | broken | Producer archive manifest has 69 matching hashes and 63 raw plant logs with exits. Two reviewer reruns each killed all 62 shipped narrowings and preserved the control; five additional reviewer narrowings were killed. The three unshipped Build UTF-8 site narrowings survive the shipped tests, so the census is incomplete (same finding). |

## Entry and oracle audit

Production entries are `ValidDisposition`, `ValidProfile`, `ValidStrategy`, `IsCoreReasonCode`, `ValidReasonCode`, `BuildFidelityReport`, and `DecodeFidelityReport`. The first five are the registry entries; the two report entries reach their applicable gates through `buildFidelityReport`/`decodeDispositionRecord` and their shared validators. Strategy has no report field in this leaf. The test oracles retype closed lists from the pinned spec, enumerate the four record rules independently of production constants, and assert literal refusal text. Tests for aggregates perturb named cells through Decode; Build has explicit invalid-breakdown cases. The full suite and reviewer plants establish **38 of 40 AC rows fully driven**, row 33 partially driven (4 of 7 UTF-8 sites), and row 24 as the disclosed byte-value bound.

The importer grid has no moved classes. The report and registry are library surfaces; the diff adds no CLI/doctor/capability claim. The operation writes no durable state, so the crash/idempotency AC is satisfied by pure deterministic Build and Decode tests. Sibling-leaf contracts remain identity or digest fields only.

## Validation and evidence identity

- Exact candidate: scratch `GIT_INDEX_FILE` tree OID equals the Change Request tree; patch SHA-256 `76df01ffe0d660ce199e53c336ff1940fcd89b7411b477a10357fa3c64cbb850`.
- `go test ./... -count=1`: exit 0, 45 packages listed, log complete.
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0 on the clean rerun. An earlier vet attempt overlapped deletion of a temporary fixture emitter and is classified as unrun.
- New gate tests `-count=3`: exit 0.
- Independent JCS: target digest `sha256:3b0c26079bc6d8524fb9bf18d98840a5f9211110dd68dc97c8959388dfccdba5`; archive digest `sha256:73cd56e2a69e1d896f69537e550ff22b58ea21706dcf6e00d277880cfdb86421`; both recomputed exactly.
- Importer outcome grid: base 14,706 / candidate 14,706 cases, zero moved.
- Shipped mutant harness: two independent exact-candidate archive runs, each exit 0 with 62 KILLED and one applied neutral control SURVIVED; 63 raw logs with subprocess exits per run.
- `gofmt -l` on touched Go packages: zero listed files.
- CR construction validation log: 30 of 30 configured command shards green; unrelated historical board activity warnings did not fail the validator.

## Verdict

**changes_requested → to-dev** (P2, one mechanism). The byte-value bound is correctly documented, and every other swept row held for its named attacks. The Build UTF-8 refusal evidence is incomplete: an admitting narrowing survives the committed suite. This is a verification bypass found with a mutant; the unmodified candidate currently refuses the three malformed inputs.

```json
{
  "findings": [
    {
      "id": "unmeasured-build-utf8-sites",
      "row": "Build UTF-8 refusal sites",
      "invariant": "Every BuildFidelityReport string gate that refuses invalid UTF-8 must be driven by a named committed negative test and killed by an admitting narrowing run alone.",
      "mechanism": "internal/clonefidelity/record.go:92 and :115 and decode.go:262 check source_class, target_locator, and forbid_reasons before JSON sealing. TestBuildUTF8 exercises source_item_key, explanation, reason_codes, and extensions, but none of those three sites. The shipped harness has no narrowing at them. A one-case patch for byte sequence ff fe makes BuildFidelityReport admit rewritten strings; TestBuildUTF8 still passes, and the source_class patch leaves the whole clonefidelity package suite green.",
      "reproductions": [
        {
          "test_file": "utf8-gap/review_utf8_probe_test.go and utf8-gap/replay.py",
          "command": "python3 replay.py in a clean archive of tree 7bade702b4bf260219b199c5c1f6d2355c9e0ea0, with both files copied to its root",
          "expected_failure": "For each of source_class, target_locator, and forbid_reasons, the baseline reviewer probe passes; after the applied one-case narrowing, committed TestBuildUTF8 still passes but the reviewer probe fails because BuildFidelityReport returns nil error. The source_class narrowing also leaves the full shipped package suite green.",
          "logs": "utf8-gap/replay-summary.log and utf8-gap/replay-logs/*.log"
        }
      ],
      "severity": "bypass",
      "priority": "P2",
      "repeat-of": "none"
    }
  ],
  "notes": [
    "Rev1 unbound-byte-aggregate is resolved as an explicit stated specification bound by the orchestrator decision; it is not repeated here.",
    "The new mutant_harness.py ends with an extra blank line (`git diff --check` against the candidate tree warns); gofmt and vet are clean. This is nonblocking formatting only."
  ]
}
```

## Rework scope (for the producer)

Add committed BuildFidelityReport tests for invalid UTF-8 in `source_class`, `target_locator`, and `forbid_reasons`, asserting `ErrInvalid` and literal member-specific details. Add one admitting narrowing per site to the shipped harness and show each new named test kills its narrowing alone with raw logs. Then update the gate × entry census, TRACEABILITY, and measured AC ratio. Keep the orchestrator-approved `byte_counts` stated bound and the unchanged production behavior unless a new test exposes a real defect. This is one `to-dev` rework cycle for the single mechanism, not a separate research or harness task.

Free hunt found no other blocking mechanism. No surface row was omitted.
