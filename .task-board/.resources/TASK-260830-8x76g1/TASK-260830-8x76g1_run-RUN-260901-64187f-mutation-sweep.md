# TASK-260830-8x76g1 rev11 mutation sweep replay

Candidate: final developer test tree for RUN-260901-64187f.

Criterion: every effective widening/removal of a declared bound must make the
uncached production-entry suite fail. A single-site edit that leaves an equal
or stricter independent production check is classified `SUBSUMED`, then the
multi-site effective widening is run and must be `KILLED`.

## Attached harness replay

| Batch | Raw outcome on final tree | Follow-up |
| --- | --- | --- |
| batch1 | 14 KILLED; Board logical-ID length and BlobChunk index single sites SUBSUMED | logical-ID combined mutant KILLED in batch5; index ceiling is unreachable behind position/size gates as rev11 recorded |
| batch2 | 15 KILLED; top-level Git submodules-array cap SUBSUMED | array + total combined mutant KILLED in batch5 |
| batch3 | 7 KILLED; nested submodules-array cap and extension-key minimum SUBSUMED; 7 stale-line SETUP-FAIL | both combined mutants KILLED in batch5; stale rows rerun below |
| batch4 | 3 KILLED; 5 stale-line SETUP-FAIL | corrected rows rerun below |
| batch5 | 9 KILLED | includes logical-ID, both submodule-cap, media-type, and extension-key combined mutants |
| batch6 (`./...`) | 7 KILLED | all seven rev11 upper-bound findings killed by the full repository suite |
| batch7 | 6 KILLED; 5 shared-string single sites SUBSUMED; 2 stale-line SETUP-FAIL | five combined lower-bound mutants and two corrected rows rerun below |

The exact batch files are the reviewer-supplied resource
`TASK-260830-8x76g1_rev11-mutation-sweep.tar.gz`. Each harness invocation
restored every edited file and asserted its pre-mutation SHA-256 before moving
to the next mutant.

## Corrected stale metadata replay

The production file did not move during this run; the attached JSON carried
line offsets from an earlier layout. Correcting only those bindings produced:

| Effective mutation | Result |
| --- | --- |
| env-name pattern 128 -> 129 | KILLED |
| session-name pattern 64 -> 65 | KILLED |
| semver leading zeros | KILLED |
| reverse-DNS dot removal | KILLED |
| reverse-DNS first-label 63 -> 64 | KILLED |
| media type uppercase type | KILLED |
| media type uppercase subtype | KILLED |
| extension object depth 4 -> 5 | KILLED |
| Board logical-ID regex-only maximum | SUBSUMED by the independent 128-character bound; combined variant KILLED |
| BlobChunk zero-size check only | SUBSUMED by empty-descriptor/coverage invariants; combined variant KILLED |

## Rev11 findings

| Finding family | Effective mutant result |
| --- | --- |
| BlobChunk.size upper bound | KILLED, package and full-suite runs |
| GitIndex.version lower edge | KILLED, package and full-suite runs |
| GitIndexEntry.mode uint32 | KILLED, package and full-suite runs |
| Board logical-ID 128 | KILLED with length + regex combined widening, package and full-suite runs |
| managed-tree project paths 256 | KILLED, package and full-suite runs |
| GitSubmodule repository identity 256 | KILLED, package and full-suite runs |
| GitSubmodule project paths 256 | KILLED, package and full-suite runs |
| task element ID minimum | KILLED directly |
| argv element minimum | KILLED directly |
| WorkspaceSnapshot members minimum | KILLED directly |
| board goal ID minimum | KILLED after removing shared non-empty gate |
| managed-tree identity minimum | KILLED after removing shared non-empty gate |
| Git workspace repository identity minimum | KILLED after removing shared non-empty gate |
| Git remote name minimum | KILLED after removing shared non-empty gate |
| GitSubmodule repository identity minimum | KILLED after removing shared non-empty gate |

Effective result: **KILLED 15/15 rev11 findings; actionable SURVIVED 0;
effective SETUP-FAIL 0.**

Raw anomalies retained for honesty:

- An accidental `mutate2.py batch3.json` invocation exited 1 (`KeyError:
  'edits'`); the correct `mutate.py batch3.json` invocation exited 0.
- Original stale-line rows print `SETUP-FAIL` even though the harness process
  exits 0; none is counted as executed until its corrected replay above.
- Raw one-site redundant checks print `SURVIVED`; none is presented as killed.
  Their effective multi-site widenings are the evidence used for the zero
  actionable-survivor result.
