# TASK-260924-3n78rv — Change Request revision 5 review

**Verdict: accepted.** Candidate tree `a5583e589d9ac3b6eaa2ed5504a25959c38970f7` over base `618d78de05450c40ec39df7c97073a6feecc8f81` is reproduced by a scratch index. The attached CR patch and the immutable Git diff both hash to `04573f2049906cfb73cc04be57941aff83bf8864ed78d128eef183bafe5c5354`. No registry or config file changed. This is a `task_delta`; acceptance routes to `integrating`, and this review makes no commit or landing claim.

## Surface results

| Row | Result | Attack |
| --- | --- | --- |
| Selection and branch routing | held | `TestSelectStrategyOracle`, `TestSelectProfileOracle`, and `TestSelectionAgreement` passed in the three-run package rerun; selection entries reached through their exported calls. |
| Canonical item classification | held | All-kinds classification corpus and closed-fact structural test passed in the three-run package rerun; the closed event-kind list is pinned to SPEC §13.14.1. |
| Strategy × profile × item planning | held | The 975-cell production-entry oracle passed three times; my continuation-one-kind narrowing failed `TestPlanItemWholeDomainOracle` alone. |
| Visible authority and escaping | held | Independent seven-text corpus over all 26 event kinds and three field positions passed `TestReviewerOwnAuthorityCorpus`; one-class authority escalation failed it alone; malformed UTF-8 refusal tests passed three times. |
| Historical tools and source accounting | held | Target-effects owner rule table and zero fold passed three times; independent semantic-empty and exact-reason probes passed; two single-class owner bypasses failed `TestReviewerOwnRecordCoupling` alone. |
| Importer outcome composition | held | My base checkout reran nine reused owner package suites green; candidate three-run planner package tests cover its importer entry inputs. Blob comparison found only the additive `clonefidelity/record.go` seam among 232 files in these nine packages. No existing `(package, entry, input)` class moved; the seam and eight new planner entries are candidate-only. |
| Candidate provenance and trunk composition | held | Scratch index produced CR tree `a5583e589d9ac3b6eaa2ed5504a25959c38970f7`; 362 trunk-delta paths equal candidate blobs; config hash unchanged; diff checks passed. |

## Findings

```json
{"findings": []}
```

## Notes

- The new `clonefidelity.ValidateDispositionRow` exports the existing row builder's validation. The planner passes each caller-supplied mapping to that owner through `PlanTargetEffects`; the revision 4 `semantic`/empty and `exact`/reason bypasses now refuse with the owner's literal rule text. Reviewer-authored single-class bypasses were each killed by the named production-entry test alone.
- The mapping type carries no canonical object or captured source evidence. Its derived digest is validator scaffolding for the target-effects fold, and the candidate states this bound. This leaf does not build a FidelityDispositionRecord from that digest.
- The unsharded `go test ./...` was interrupted after ten minutes under concurrent host suites; only its 28 completed packages count. Bounded shards covered the remaining packages. The `tmuxserver` shard initially failed because a long review `TMPDIR` exceeded the Unix socket length; a focused short-path test passed. A complete short-path package rerun exposed a deterministic pre-existing failure: `TestUnixDialerMissingDirectoryIsUnknown` expects `unknown` while unchanged `UnixDialer.Dial` maps `ENOENT` to `stale`. The focused test fails on both exact base and candidate with byte-identical production and test files. The remaining `tmuxserver` tests passed in a short-path rerun with that one baseline-failing test excluded. This is outside this Change Request delta and is excluded from the 19/19 leaf AC coverage claim; the full repository suite is not green on this host.
- My first sparse base copy omitted `README.md` and `task-board.config.json`, so `canonicaljson` failed to read two test fixtures. I added the exact base blobs and reran that owner package successfully; the initial failed read was never counted as an absence or a pass.

## Validation and coverage

The 19 TRACEABILITY rule rows have named production-entry tests (19 of 19): `SelectStrategy`/`SelectProfile` for selection; `ClassifyItem` for typed facts; `PlanItem`/`PlanSession` for 975 strategy × profile × item cells and count folds; `ProjectVisibleText`/`EscapeVisibleText` for visible authority, escaping and UTF-8; `PlanTargetEffects` for owner record rules and inert target effects. The owner-row census and static call graph supplement the behavioral rule table; neither is counted as a substitute for its production-entry probes. Independent authority corpus: 7 text classes × 26 event kinds × 3 field positions = 546 cells. The normative JCS fixture and SHA-256 hashes were recomputed independently. Windows vet and exact-tree hygiene passed. The unrelated pre-existing `tmuxserver` failure prevents a green full-repository test claim; all leaf and owner tests run for this review passed. Two complete harness passes each killed all 57 narrowings; the applied neutral control survived in both passes. The token-preserving visible-text mutant ran the behavioral escaping suite. Four reviewer-authored narrowing plants were killed; a reviewer-authored neutral control survived. Full validation details and command transcripts are in the review evidence archive.
