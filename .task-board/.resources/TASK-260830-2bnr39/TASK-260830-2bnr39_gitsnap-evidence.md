# TASK-260830-2bnr39 evidence: git repository and index snapshot

Scope: `relux-works/agent-session-manager-spec@v0.5.0` §10.2–10.4, §12.1–12.3,
leaf boundary: repository identity, HEAD/ref, worktree metadata, index
stages/flags, staged and unstaged deltas, file modes.

Deliverable: new package `internal/gitsnap` (production entry `Capture`)
plus README section. No existing file modified except `README.md` (new
section only; ownership paragraph untouched, tracecheck counts unchanged).

## AC coverage: 8 of 8 rows driven through the production entry

| AC row | Production call site | Named committed test |
| --- | --- | --- |
| Repository identity | `Capture` → `readRemotes` → `deriveIdentity` | `TestCaptureLiveBranchRepository`, `TestRemoteFetchPushMapping` |
| HEAD/ref (branch/detached/unborn) | `Capture` → `readHead` | `TestCaptureLiveBranchRepository`, `TestCaptureLiveHeadModes` |
| Worktree metadata | `Capture` → `readRepository` | `TestCaptureLiveBranchRepository` |
| Index stages (0–3) | `Capture` → `parseIndexEntries` | `TestCaptureLiveConflictStages`, `TestCompatConflictStages` |
| Index flags (ita/skip/assume/fsmonitor) | `Capture` → `parseIndexRecord` + `debugFlags` | `TestCaptureLiveIndexFlags`, `TestCompatFSMonitorBit` |
| Staged deltas | `Capture` → `parseDeltas` (cached) | `TestCaptureLiveBranchRepository` |
| Unstaged deltas | `Capture` → `parseDeltas` (worktree) | `TestCaptureLiveBranchRepository`, `TestCaptureLiveMissingTrackedPath` |
| File modes | `Capture` → `statIndexPaths`/`statDeltaPaths` | `TestCaptureLiveExecBitAndSymlinkKind` |

Exact contract fixtures: `WS-GIT-ROUNDTRIP-1` parent (4 lines) and child
(1 line) index corpus quoted verbatim, parsed through `Capture` into the
normative wire entries (`TestFixtureParentIndexLinesMatchWireEntries`,
`TestFixtureChildIndexLineMatchesWireEntry`). Negative fixtures:
`TM-GIT-N2` stage-4 (`TestRefuseIndexStageFour`), `TM-GIT-N3` class
(`TestRefuseBranchWithLiteralHeadRef`); `TM-GIT-N2` count arm is owned by
`internal/canonicaljson` (`TestTransferManifestNestedValueConstraintsReachBothIdentityEntries`,
"index entry count" case) with the construction invariant guarded here
(`TestEdgeEntryCountIsConstructionInvariant`).

## Gate census

One census row = one literal `refuse(Gate...)` call site in non-test
production files (comments/strings stripped). Property varies over
(gate, enclosing reader). Result: **59 call sites over 16 registry
gates** (`TestGateCensusCoversEveryRefusalSite`), 3 of 3 codes live
(`TestGateCodesAreClosedAndLive`). Controls: comment/string plants stay
uncounted, unregistered-gate plant reddens, alias-bound and
fresh-spelling plants stay invisible (stated blind spot; direct
invocation upheld by construction), method-literal plant counted.

## Mutant battery (harness `/tmp/gitsnap-mutants/run.py`, logs `/tmp/gitsnap-mutants/logs/`)

| Mutant | Narrows the gate to | Named test that fails | Result |
| --- | --- | --- | --- |
| M1 stage admits 4 | stage 4 only | `TestRefuseIndexStageFour` (exit 1); `TestEdgeStageThreeAccepts` stays green (exit 0) | KILLED |
| M2 version admits 5 | version 5 only | `TestRefuseIndexVersionFive` (1); `TestCompatIndexVersions` green (0) | KILLED |
| M3 remotes admits 17 | 17 only | `TestEdgeRemotesSeventeenRefuses` (1); 16-accept green (0) | KILLED |
| M4 entries admits 65537 | 65537 only | `TestEdgeIndexMaxPlusOneRefuses` (1); 65536-accept green (0) | KILLED |
| M5 identity admits 257 chars | 257 only | `TestEdgeIdentity257CharsRefuses` (1); 256-accept green (0) | KILLED |
| M6 unborn admits tags | `refs/` prefix | `TestRefuseUnbornNonBranchRef` (1) | KILLED |
| M7 delta admits X | status X only | `TestRefuseUnknownDeltaStatus` (1) | KILLED |
| M8b branch admits 40-hex HEAD in sha256 repo | 40-hex only | `TestRefuseCrossFormatOID` (1); `TestCompatSHA256ObjectFormat` green (0) | KILLED |
| M9 sort admits duplicate (path,stage) | duplicates only | `TestRefuseDuplicateIndexEntry` (1); unsorted still refused (0) | KILLED |
| M10 fetch/push swapped, tokens preserved | swapped mapping | full suite exit 1 driven by `TestRemoteFetchPushMapping`; `TestAcceptBaselineSnapshot` green (0) | KILLED |
| M11 assume-bit reads skip bit, tokens preserved | swapped flags | full suite exit 1 driven by `TestCaptureLiveBranchRepository` | KILLED |
| M17 push-URL validation skipped | credentialed push only | `TestRefuseCredentialPushURL` (1); `TestRemoteFetchPushMapping` green (0) | KILLED |
| M18 index admits 8-hex OID | 8-hex OIDs only | `TestRefuseIndexBadOID` (1); `TestRefuseIndexBadMode` green (0) | KILLED |
| M19 worktree admits FIFO as other | FIFOs only | `TestRefuseSpecialWorktreeFile` (1); `TestCaptureLiveBranchRepository` green (0) | KILLED |
| M12 remotes-min disabled (`<1`→`<0`) | empty remotes | `TestRefuseNoRemotes` (1) | KILLED |
| M13 branch-ref clause disabled | literal HEAD ref | `TestRefuseBranchWithLiteralHeadRef` (1) | KILLED |
| M14 consistency index clause disabled | index moves unnoticed | `TestConsistencyRefusesIndexMutationArmedOnce` (1); head-mutation still refused (0) | KILLED |
| M15 corrupt-head clause disabled | empty HEAD | `TestRefuseCorruptHead` (1) | KILLED |
| M16 upstream clause disabled | bad upstream | `TestRefuseBadUpstream` (1) | KILLED |
| C-bad inverted sort comparator | — (must redden) | `TestAcceptBaselineSnapshot` (1) | KILLED |
| C-neutral oid-swap | behavior-neutral on reachable inputs (prefixes are constructed, not parsed) | — (exit 0, survival required) | SURVIVED-NEUTRAL-OK |
| C-neutral comment-only | no behavior change | full suite (exit 0) | SURVIVED-NEUTRAL-OK |

Totals: killed 20, neutral-ok 2, holes 0, unexpected 0, unmeasured 0 (22 rows).
Every run used `-count=1`; files restored from backup with identity
verified after the run (`snapshot.go` identical to pre-run bytes).

Per-gate narrowing account: 13 of 16 gates ship a narrowing mutant that
admits exactly one rejected member (M1–M7, M8b, M9, M14, M17–M19, plus
M6 for the tag arm of HeadCorrupt). `GateNotRepository` is an
environment precondition over an open class (any non-repo state) and
ships existence evidence only: any weakening admits all non-repos, which
is the delete class. `GateHeadRef` and `GateUpstreamRef` delegate their
grammars to `internal/scalar` (clause-disable mutants M13/M16 prove the
delegation call sites are live; grammar narrowing belongs to the scalar
battery). `GateRemoteURL` fetch-arm grammar likewise delegates; its push
arm narrows in this battery (M17).

Notable find during the battery: the first M8 (ParseGitOIDForObjectFormat
→ ParseGitOID) survived because OID prefixes are constructed from the
probed format, so a cross-format string is unreachable through `Capture`
and the swap is neutral on reachable inputs. The witness was replaced
with a true narrowing (M8b) and the swap kept as a labeled neutral
control instead of being silently dropped.

## Crash/idempotency (Capture is read-only; no durable mutation)

- `TestIdempotentRepeatCapture`: two live captures encode byte-identically.
- `TestConsistencyRefusesIndexMutationArmedOnce` / `TestConsistencyRefusesHeadMutation`:
  one-shot scripted faults; the tests assert the armed queue is empty
  afterwards (fault provably fired inside the window) and the runner log
  shows the closing re-read.
- `TestConsistencyRetryHeals`: stable replay captures.

## Validation commands (real exit codes, standalone processes)

- `go build ./...` → exit 0
- `go vet ./internal/gitsnap/` → exit 0
- `go test ./internal/gitsnap/ -count=1` → exit 0 (61 tests)
- `go test ./internal/gitsnap/ -cover -count=1` → exit 0, 83.5% statements
- `go test ./... -count=1` → exit 0 (all packages)
- `go test ./... -cover -count=1` → exit 0
- `go run ./internal/traceability/cmd/tracecheck` → exit 0, counts unchanged
  (bindings=53, acceptance_cases=98)

## Stated bounds (not implemented here, by story design)

Object packs, raw index/pack blobs, untracked/ignored bytes, symlink
targets, submodule recursion, working-tree manifests (sibling leaves);
`entry_count` wire-mismatch refusal (`internal/canonicaljson`); live
fsmonitor-valid observation (environment-dependent; bit parsed through
the entry); `axerror` envelope mapping (CLI owner).
