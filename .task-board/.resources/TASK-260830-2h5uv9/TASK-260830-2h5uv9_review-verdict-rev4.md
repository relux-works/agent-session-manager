# TASK-260830-2h5uv9 — review verdict, CR-TASK-260830-2h5uv9-4 revision 4

Verdict: **changes_requested** → `to-dev`
Reviewer run: RUN-260924-968ee6 (claude-opus-5-5). Candidate tree `5fb0df8e5066fa3380c3e4ae1f76aca07fc50871` (scratch-index `write-tree` of the live worktree equals it). Base `6d3bff9` = origin/main (trunk did not move). Prior verdict: rev3 (`sync-conflict-not-crossed-with-partial-peer`).

## Rev3 finding — status

Fixed for the vector and the four briefed narrowings. The conflict class is now generated inside the product (`runDurableSyncGeneratedConflictCase`: peer-only identity, reorder, duplicate, gap-fill round, conflict round 1..3, every target position), and `TestDurableSyncConflictWithPartialOverlapPeer` is always-run. My plants, each run ALONE on `-run TestDurableSyncConflictWithPartialOverlapPeer`:

| Plant (durable.go SyncFrom, common-ID build) | Kind | Named test | AST test |
| --- | --- | --- | --- |
| F control `_ = len(common)` | neutral | exit 0 SURVIVED | exit 0 |
| A `exists && len(peerIDs) == len(localIDs)` | narrowing | exit 1 KILLED | exit 0 (passes) |
| B `exists && len(peerIDs) <= len(localIDs)` (audit only when peer holds nothing extra) | narrowing | exit 1 KILLED | exit 0 |
| C `exists && namespace == durableJSONNamespaces[0]` | narrowing | exit 1 KILLED | exit 0 |
| D `exists && reviewSyncRound == 0` (first round only) | narrowing | exit 1 KILLED | exit 0 |
| **E `common = common[:min(1,len(common))]` (audit only the first common ID per namespace)** | narrowing | **exit 0** | exit 0 |

E under the full configured package suite `go test ./internal/merkleinventory -count=1`: **exit 0 (SURVIVED)**. E under `-run 'TestDurableSyncGeneratedPerturbationProduct/N3/'`: exit 1, 32 FAIL lines — only the selector-gated product sees it.

## Mandatory reruns (this reviewer, exact tree)

| Check | Result |
| --- | --- |
| Digest | tracecheck green, recomputes pinned `cf559de0…4c80`; plant (swap one test name in the 2h5uv9 case) → tracecheck exit 1 |
| Full `go test ./... -count=1` | archive copy: 45 ok, 1 FAIL = `internal/specpin` (needs `.task-board`, excluded per scratch rule); live worktree `go test ./internal/specpin`: ok |
| `GOOS=windows GOARCH=amd64 go vet ./...` | exit 0; gofmt -l clean |
| README plant | `88/610→89/610` → `TestREADMEMeasuredCoverageMatchesTracecheckReport` FAIL |
| tracecheck | `clauses_discharged=88/610`, acceptance_cases=163 |
| Importer grid | 28 packages (importers of canonicaljson/merkleinventory/provhost/sessquery/traceability, specpin excluded), `go test -json` base vs candidate: 0 fail in candidate; moved: merkleinventory pkg fail→pass (absent at base); removed: canonicaljson `TestUnsupportedSection10RecordSchemasValidateCommonEnvelopeBeforeRefusal/{tombstone,tombstone-ack}` (the named moved class: tombstone/ack unsupported → supported, landed in 147hsj rev2, reviewed); tmuxserver 2 subtests renamed (byte-length names follow scratch path length; +4/−2, not a behaviour move) |
| Determinism | 8 named sync tests `-count=3` ok (79s); product N2 shard `-count=3` ok |
| Registry decode | 163 cases; 0 base cases removed/changed; 0 base ownership rows lost; the 3 Story cases (nxqqaw→serve.go Dispatch, 147hsj and 2h5uv9→durable.go SyncFrom) are each referenced by 6/9/9 clause lists; every named test declaration exists in its file |
| task-board.config.json | byte-identical to base |

## Surface table

| Row | Result |
| --- | --- |
| (1) property — rev3 conflict × partial peer class | held for A–D (killed alone) |
| (1) property — common-ID audit, every identity | **broken** — F1 (E survives configured suite) |
| (1) other plants (tie-break, dup drop, regression, idempotence) | held at rev3 (P1,P3,P4,P5 killed); SyncFrom unchanged since rev3, not re-planted |
| (2) unbounded axes | held (unchanged since rev3) |
| (3) registry, digest, README | held |
| (4) composition | held |
| (5) shipped harness (mutations.py) rerun | not-attacked — budget; replaced by 5 own plants + control |

## findings

```json
[
  {
    "id": "sync-conflict-not-crossed-with-partial-peer",
    "row": "(1) property: every common identity byte-audited on every sync",
    "invariant": "Rework brief rev4 / SPEC v0.7.0 §11.4: every identity common to local and peer is byte-audited on every sync; a mismatch yields literal integrity_failure + quarantine + abort — killed under the configured suite",
    "mechanism": "internal/merkleinventory/sync_property_test.go L51-57: TestDurableSyncGeneratedPerturbationProduct t.Skip()s without a nested -run selector, so the configured `go test ./...` never runs the generated conflict class; the always-run TestDurableSyncConflictWithPartialOverlapPeer places at most one common identity per namespace, at the first position. A narrowing of durable.go L186-192 that audits only the first common ID per namespace (common = common[:1]) lets a conflicting second common identity diverge silently, and the configured suite stays green.",
    "reproductions": [
      {
        "test": "plants/plant.py E applied to durable.go (see evidence tar)",
        "command": "go test ./internal/merkleinventory -count=1",
        "expected": "should FAIL; observed exit 0 (plants/E.suite.log). Control: `-run 'TestDurableSyncGeneratedPerturbationProduct/N3/'` exit 1, 32 FAIL (plants/E.productN3.log) — the property exists but is not in the configured suite",
        "logs": ["plants/E.suite.log", "plants/E.named.log", "plants/E.productN3.log"]
      }
    ],
    "severity": "bypass",
    "repeat-of": "rev3 sync-conflict-not-crossed-with-partial-peer (same class at an adjacent site; rev3 note N1 predicted it)"
  }
]
```

Second consecutive same-class finding: the next revision must carry a named regression test and a narrowing mutant for the class (below).

## notes (non-blocking)

- N1: `TestDurableSyncCommonIDAuditIsUnconditionalInAST` is identifier-matched: plants A–D all pass it (they narrow the `common` construction, not the fetch call). It is not a structural argument for the class; the behavioural test carries the kill. State it as a bound or check the `common` construction too.
- N2: `internal/merkleinventory/__pycache__/mutations.cpython-314.pyc` is in the candidate tree — a build artifact; remove it and gitignore `__pycache__`.
- N3: tmuxserver byte-length subtest names depend on scratch path length (grid noise, not this Story).

## Rework scope (for the producer)

1. Make the conflict class measured by the configured suite: either an always-on bounded shard of the generated conflict class (e.g. N≤3, all targets × rounds) that runs without a selector, or a named always-run test with ≥2 common identities in ONE namespace where the conflicting identity is not first, crossed with a peer-only identity. Minimum: plant E must die when `go test ./internal/merkleinventory` runs.
2. Ship E (`common = common[:min(1,len(common))]`) and "audit only the last common ID" as harness rows, killed alone under the configured suite, plus the applied neutral control.
3. Remove the `.pyc` artifact. Rerun tracecheck/README pin/digest if registry test lists change.
