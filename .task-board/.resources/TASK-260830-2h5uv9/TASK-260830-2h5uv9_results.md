# TASK-260830-2h5uv9 producer results — rev5

Candidate: Story checkpoint `86a6188a0e35fd04a24eba62a01b71a54d6faa5d` plus the uncommitted rev5 worktree. Fork base: `0ca3e4c26e2b275212796657f785b9b450f6174e`. Current `origin/main`: `6d3bff999f75ed8585a7ef32074ab108f364ec51` (unchanged; no candidate refresh needed). Authority: pinned `internal/specdoc/SPEC.v0.7.0.md`, §§11.4, 5.3, and 10. Candidate remains uncommitted.

## Measured acceptance coverage

**1 of 2 task acceptance criteria is fully driven by named production-entry tests.** Criterion 1 is driven through `(*DurableIndex).SyncFrom` and `(*DurableIndex).RebuildProjection`, using `TestDurableSyncGeneratedPerturbationProduct` and the always-on `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord`. Criterion 2's exact contract, negative/refusal, crash recovery, and idempotency parts have named passing tests. Its no-unsupported-capability clause is a stated boundary: the internal library adds no public command, doctor, provider, network, or Host Channel advertisement entry point; README evidence makes no such claim. The producer brief contains no formal surface table; `coverage-map.md` records this as a brief gap rather than a waiver.

## Rev5 property and generated ranges

The configured package suite executes the default N≤3 generated shard with no selector and no `t.Skip` branch. The large product is available under nested selectors and was run in bounded shards:

| N | Valid-union cases | Arrival permutations | Run result |
| ---: | ---: | --- | --- |
| 1 | 8 | all | green in default shard |
| 2 | 40 | all | green in default shard |
| 3 | 264 | all | green in default shard |
| 4 | 2,208 | all 24 | 3 accepted order-range processes |
| 5 | 22,560 | all 120 | 24 accepted order-range processes |
| 6 | 12,920 | 34 deterministic unique samples | 17 accepted order-range processes |
| **Total** | **38,000** | N≤5 exhaustive; N=6 sampled | 44 accepted N4–N6 processes plus default N1–N3 execution |

Each case crosses duplicate delivery, both timestamp directions, an absent or later-filled gap, compatible partial-peer subsets, and 1–3 sync rounds. The object universe includes competing leases, a losing-lease branch, a Tombstone/Acknowledgement pair, and same-identity/different-byte variants. Expected roots and lease projection use test-only oracle algorithms, independent of production sync, root, classifier, and lease comparator. Every run either matches the reference or returns the literal typed `integrity_failure`; no silent divergence is accepted. N=6 order coverage is sampled, and set sizes beyond six are not claimed. Structural reasoning for later rounds over a stable peer is recorded in `axis-inventory.md`.

The new always-on position shard adds **432 generated conflict cases**: three common identities in one namespace × all six arrival permutations × duplicate on/off × both skew assignments × conflict rounds 1–3 × two peer-only lease identities. Each target position (first, middle, last) is audited through production `SyncFrom`, even when another identity is absent from the peer. It asserts exact quarantine bytes, peer immutability, partial-peer non-regression, and typed conflict on retry.

## Narrowing mutants

| Mutant | Narrowing | Named test that fails | Plant exit |
| --- | --- | --- | ---: |
| `sync-common-audit-first-id-only` | Audit only the first common identity | `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` | 1 |
| `sync-common-audit-last-id-only` | Audit only the last common identity | `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` | 1 |
| `sync-common-audit-even-ids-only` | Skip odd-indexed common identities | `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` | 1 |
| `sync-common-audit-first-namespace-only` | Audit common IDs only in the first namespace | `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord` | 1 |
| `sync-common-audit-partial-overlap-peer` | Audit only when peer has no peer-only object | `TestDurableSyncConflictWithPartialOverlapPeer` | 1 |
| `skip-census-token-preserving-behavior` | Keep the `Skip` token but make census ignore the call | `TestSkipCensusRejectsAnUnjustifiedPlantedSkip` | 1 |
| `control-neutral-comment` | Applied comment-only edit with no behavior change | `TestDispatchServesNormativeChildrenBody` | 0, survives by design |

The four review-requested audit plants were run alone and appear in the shipped harness. The applied control is the sole intentional survivor; its bound is only that an equivalent comment edit changes no gate or behavior. Full harness: 86 rows, 85 killed, 1 applied neutral control survived, 0 invalid/not-applied rows. Raw per-plant logs and full table are in the evidence archive.

The AST skip census scans package `_test.go` files. No skip sites are currently allowlisted; a new unjustified skip fails. Both the ordinary and token-preserving planted skips are rejected. The `.pyc` hygiene test now walks the repository root; its planted `.pyc` and `__pycache__` control is rejected. A task-owned scratch cache discovered during the run was removed, and the final repository scan found none.

## Registry, README, and composition

Decoded `ownership.v0.7.0.json` confirms that all three Story acceptance cases appear in their decoded clause-binding lists. Production ownership is recorded for `SyncFrom` (and the §5.3 lease owner). The expected-red digest derivation printed `2302a10a0174ada997614ea7d6112add30e9b2328f6b25ea796ea175f48568cd`; after pinning, `go run ./internal/traceability/cmd/tracecheck` exits 0 and prints:

```text
traceability ok: contracts=64 normative_sections=36 acceptance_cases=163 fixtures=33 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=72 full=4 partial=11 sliver=12 unevidenced=41 unmeasured=4 unowned=7 clauses_discharged=88/610
```

The README measured-coverage pin passes (exit 0); the plant changing `bindings=72` to `bindings=73` fails (exit 1). The README coverage figure grep and the seven ownership-paragraph figure pins are included as evidence. Section-scoped diagnostics exit 1 by design: §5.3 discharges 7/8, §10.7 1/18, and §11.4 6/7; the missing clauses are named in the conformance matrix and belong to other owners.

Exact importer graph, base→candidate, was re-derived with Go 1.25.5 `go list -test -json ./...`, `GOWORK=off`, darwin/arm64:

| Measure | Fork base | Candidate | Delta |
| --- | ---: | ---: | ---: |
| Package paths | 44 | 46 | +2 |
| Import references | 1,468 | 1,592 | +124 |
| Internal-package references | 317 | 350 | +33 |
| Direct importers of canonicaljson / rpcwire / provhost | 22 / 3 / 5 | 23 / 4 / 6 | +1 each |
| Union of tracked target importers | 25 | 27 | +2 |

Added target edges are `merkleinventory` → canonicaljson/rpcwire and test-only `tmuxserver` → provhost; no target edge was removed. `termbind` trunk-only changes are separate. The immutable fork-base importer tests were previously green (25 packages, 8,967 pass events) and are reused; the exact candidate suite covers all 27 target importer packages. Full `(package, entry, input)` grid and moved classes are in `internal/merkleinventory/IMPORTER-OUTCOMES.md`.

## Validation record

| Command / evidence | Exit | Log |
| --- | ---: | --- |
| Focused default product, 432 position cases, skip census, hygiene plants | 0 | `.temp/TASK-260830-2h5uv9/focused-rev5-final.log` |
| New test determinism, `-count=3` | 0 | `determinism-rev5-final.log`; package result `merkleinventory 289.984s` |
| Full configured `go test ./... -count=1 -v` on the final source tree | 0 | 46/46 package summaries green; `full-configured-rev5-final2.log` |
| `GOOS=windows GOARCH=amd64 go vet ./...` | 0 | `.temp/TASK-260830-2h5uv9/windows-vet-rev5-final.log` |
| `go build ./...` | 0 | `.temp/TASK-260830-2h5uv9/build-current-01.log` |
| Tracecheck and registry re-derivation test | 0 | `.temp/TASK-260830-2h5uv9/tracecheck-rev5.log`, `traceability-retest-rev5.log` |
| README coverage pin / wrong-figure plant | 0 / 1 expected | `readme-coverage-pin-rev5.log`, `readme-coverage-plant-rev5.log` |
| Section-scoped diagnostics (§5.3 / §10.7 / §11.4) | 1 / 1 / 1 expected | `section-5.3-rev5.log`, `section-10.7-rev5.log`, `section-11.4-rev5.log` |
| `gofmt -l` changed Go files; `git diff --check`; scratch-index `git diff --cached --check` | 0 / 0 / 0 | `gofmt-rev5-final.log`, `diff-check-rev5-final.log`; scratch index under `.temp/TASK-260830-2h5uv9/` |
| Byte compare `task-board.config.json` against base `6d3bff9` | 0 | exact `cmp`, base copy under task temp |

An earlier full-suite attempt exited 1 because the traceability re-derivation test did not yet expect the newly registered acceptance test. Its expected-case list was updated; the exact-tree rerun is the row above. Do not count the earlier failure as green.

## Out-of-contract rows and bounds

- §11.5–§11.6 raw-blob transfer, chunk staging/retry, and destination materialization belong to their separate owners; this leaf handles immutable JSON union and makes no transfer claim.
- §5.3#5 lease takeover issuance is not implemented here; sync preserves validated competing lease tuples and losing branches.
- Of §10.7, only immutable Tombstone/Acknowledgement exchange clause #13 is claimed by this leaf; issuance authority, target-state effects, deletion, acknowledgement disposition/authorization, and retention remain separate behavior.
- Arbitrary divergent event-DAG reconstruction is not claimed; projection consumes the repository-owned authoritative event chain after byte-presence checks.
- Physical power-loss behavior and native Windows crash simulation are not claimed; process-crash/restart evidence is present.
- SHA-256/UUID domains, N>6, and all N=6 order permutations are not exhaustively claimed; canonicaljson/scalar own identity validation, N≤6 was generated, and N=6 uses 34 deterministic orders.
- The producer brief supplies no formal surface table; coverage-map reports the gap. No claim silently infers coverage beyond the named tests and bounds above.
