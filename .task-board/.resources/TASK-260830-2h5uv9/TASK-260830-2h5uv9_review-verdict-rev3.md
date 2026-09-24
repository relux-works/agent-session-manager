# TASK-260830-2h5uv9 — review verdict, CR-TASK-260830-2h5uv9-3 revision 3

Verdict: **changes_requested** → `to-dev`
Reviewer run: RUN-260924-d9ce65 (claude-opus-5-5). Candidate tree `c7549f1d141a1f372722822efcca6234c3bcac6d` (live worktree index-free `write-tree` equals it), base `6d3bff9` (= origin/main at review time; trunk did not move).
Prior verdicts: none attached (rev1/rev2 were construction failures, never reviewed), so every `repeat-of` is `none`.

## Mandatory reruns (performed by this reviewer, exact tree)

| Check | Result |
| --- | --- |
| Digest re-derivation | tracecheck recomputes `sha256(json.Marshal(registry))` = `22bc6b00…fd2e0` = pinned. Plant: removing `story-260830-2h5uv9-…` from clause 11.4#7 → tracecheck exit 1, digest `07b1d670…` differs. |
| Full `go test ./... -count=1` | 45/46 ok on an archive copy; the one FAIL is `internal/specpin` needing `.task-board` (excluded from the copy per scratch rule); `go test ./internal/specpin` in the live worktree: ok. |
| `GOOS=windows GOARCH=amd64 go vet ./...` | exit 0. gofmt clean. |
| README coverage plants | `88/610→89/610` in the fenced line → `TestREADMEMeasuredCoverageMatchesTracecheckReport` FAIL; prose "88 of the 610→89" → tracecheck package FAIL. |
| Importer outcome comparison | 24 importer packages of {canonicaljson, merkleinventory, provhost, sessquery, traceability}, `go test -json` base 6d3bff9 vs candidate: 0 moved (package,test) rows, 0 removed; +732 merkleinventory, +36 sessquery rows. Base merkleinventory package-level fail = package absent at base. Keyed by test name, so same-name rewrites in canonicaljson/provhost (tombstone/ack "shape unavailable" → supported; landed in 147hsj, already reviewed) are the named moved class, not visible in the grid. |
| Determinism | `-run TestDurableSync -count=3` ok (67s); product shard N2 ok; product shard N1–4 ok (162s). |
| `task-board.config.json` | byte-identical to base. |
| tracecheck | green: `clauses_discharged=88/610`, acceptance_cases=163. |

## Registry edge (decoded)

163 cases, 0 base cases removed or changed, 0 base ownership rows lost. New cases: nxqqaw → 11.4#1–#5; 147hsj and 2h5uv9 → 5.3#6, 5.3#7, 10.7#13, 11.4#1, #2, #7. Every new case appears in at least one clause list of its row; production owners are `serve.go Dispatch` / `durable.go SyncFrom`. Held.

## Own plants (raw logs in evidence tar; each run alone = configured merkleinventory+sessquery suite, then a product shard)

| Plant | Kind | Configured suite | Product shard | Result |
| --- | --- | --- | --- | --- |
| C0 neutral `_ = len(...)` in SyncFrom | control | exit 0 | N3 exit 0 | SURVIVED (harness can report survivors) |
| P1 same-epoch timestamp tie-break in `LeaseHeadsForSession` | narrowing | exit 1 (6 tests) | N3 exit 1 | KILLED |
| P3 partial peer with no acks drops local acks | narrowing | exit 1 | N6 shard exit 1 | KILLED |
| P4 SyncFrom appends a byte to a store file each call | narrowing | exit 1 | N3 exit 1 | KILLED |
| P5 same-identity accept when bytes differ only in whitespace | narrowing | exit 1 (3 tests) | N3 exit 1 | KILLED |
| **P2 skip common-ID byte audit unless every peer ID is common** | narrowing | **exit 0** | **N3/skew0 exit 0** | **SURVIVED** |

## Surface table

No surface table was supplied by the brief (producer also recorded this). Rows swept from the brief's items:

| Row | Result |
| --- | --- |
| (1) property over the product — reorder/duplicate/gap/skew/partial/rounds | held for convergence (P1,P3,P4 killed; oracle independent: hand-built Merkle node JSON + own lease compare, no production call) |
| (1) property — same-digest-different-bytes crossed with the product | **broken** — F1 |
| (2) unbounded axes | held (bounded ranges + structural paragraph in axis-inventory; set composition is a fixed prefix list, declared as a stated bound) |
| (3) registry edge, digest, README | held |
| (4) composition / importer grid | held |
| (5) shipped harness rerun (77 rows) | not-attacked — budget; replaced by 5 own plants + control above |

## findings

```json
[
  {
    "id": "sync-conflict-not-crossed-with-partial-peer",
    "row": "(1) property over the product: same-digest-different-bytes x partial peer",
    "invariant": "SPEC v0.7.0 §11.4 union rules / AC: same-digest-different-bytes surfaces as literal integrity_failure at every entry and stops the sync, in every generated run of the perturbation product (never a silent divergence)",
    "mechanism": "internal/merkleinventory/sync_property_test.go runDurableSyncGeneratedCase (~L150-160): the conflict branch is a fixed tail scenario — a fresh conflictPeer holding ONLY objects[0]+'\\n' — so every conflict run has peerIDs == common. No generated or named test delivers a conflicting identity from a peer that also holds identities the local lacks, or under reorder/gap. A narrowing of DurableIndex.SyncFrom (durable.go ~L186) that audits common IDs only when len(common)==len(peerIDs) leaves local and peer silently holding different bytes for one identity, and nothing reddens.",
    "reproductions": [
      {
        "test": "plants/review_repro_test.go (TestReviewSameIdentityDifferentBytesFromPartialOverlapPeer: local holds lease A; peer holds lease A+'\\n' and lease B; assert literal integrity_failure from SyncFrom)",
        "command": "go test ./internal/merkleinventory -count=1 -run TestReviewSame -v",
        "expected": "candidate: PASS (production correct today); with plant P2 applied: FAIL (SyncFrom returns nil). Logs: plants/repro-candidate.log, plants/repro-P2.log",
        "logs": ["plants/P2_skip_common_audit_on_partial_overlap.suite.log (exit 0)", "plants/P2_skip_common_audit_on_partial_overlap.product.log (exit 0)"]
      }
    ],
    "severity": "bypass",
    "repeat-of": "none"
  }
]
```

Severity note: production is correct today; the finding is that the AC's central rule (conflict at every entry, crossed with the product) is unmeasured in the partial-overlap class, so a narrowing that silently diverges replicas lands green. Graded bypass because the admitted state is exactly the forbidden silent divergence.

## notes (non-blocking)

- N1: `TestDurableSyncGeneratedPerturbationProduct` `t.Skip`s under the configured `go test ./...` (no nested selector), so the 38,000-case product is never run by CR validation; only the named sub-properties gate trunk. Every plant I killed also died in the named tests, so no reproduction — but a future regression caught only by the product will land green. Consider a bounded always-on shard (e.g. N≤3) in the configured suite.
- N2: `expectedClosureConflict := gapTarget == 4 && size == 6 && pass == 1` hard-codes the object position of the Tombstone; correct for the fixed prefix list, brittle if the universe is reordered.
- N3: shipped 77-row harness not rerun (budget).

## Rework scope (for the producer)

1. Cross the same-identity/different-bytes branch with the product: the conflicting variant must arrive from a generated peer (partial overlap: the peer also carries identities the local lacks; under reorder, duplicate, and gap-filled rounds), asserting literal `integrity_failure`, quarantine, and no silent divergence (both replicas' bytes compared). Keep an always-run named test for the partial-overlap case (the reviewer repro above is the minimum).
2. Ship the P2 narrowing (`if len(common) == len(peerIDs) { fetchAndAdd(common) }`) as a harness row; it must die ALONE under the configured suite. State the invariant in the row (common IDs are byte-audited regardless of what else the peer holds), not only this vector — plant one adjacent narrowing too (e.g. audit only when `len(missing)==0`, or audit only the first namespace).
3. Re-run tracecheck/README pin/digest after test additions (registry test lists may need the new test name).
