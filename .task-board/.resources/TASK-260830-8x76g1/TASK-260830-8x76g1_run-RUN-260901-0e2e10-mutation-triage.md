# TASK-260830-8x76g1 mutation triage correction — RUN-260901-0e2e10

This artifact supersedes only site 98 in
`TASK-260830-8x76g1_run-RUN-260901-885c85-mutation-triage.md`. All other rows
remain unchanged.

| Site | Clause | Corrected classification and proof |
| ---: | --- | --- |
| 98 / line 1748 | required string empty | Live refusal clause and sole lower-bound enforcer for `ManifestEntry.symlink.target`. `TestSymlinkTargetLowerBoundReachesBothIdentityEntries` drives `target: ""` through both `CalculateObjectIdentity` and `VerifyObjectIdentity`, requires the exact `member target must be a non-empty UTF-8 string` reason, and pairs it with accepted `target: "a"`. Disabling `if text == ""` now fails the package suite, so the 110-site clause sweep classifies this row KILLED. |

The full clause corpus changed exactly one status versus reviewer rev15:

```diff
-98 line1748 SURVIVED if text == "" {
+98 line1748 KILLED   if text == "" {
```

Final clause tally: 86 KILLED / 24 raw survivors. The remaining 24 are exactly
the rows the rev15 reviewer independently accepted as dead-code or genuinely
subsumed proofs; no new survivor appeared.

The 71-mutant numeric/bound sweep remains byte-identical to its prior reviewed
result: 56 KILLED / 15 raw survivors, all with the existing executable or
arithmetic subsumption proofs. The implicit `string[1..` lower bound is not a
numeric literal and therefore belongs to the clause sweep; that previous
harness blind spot is now explicitly pinned rather than generalized away.
