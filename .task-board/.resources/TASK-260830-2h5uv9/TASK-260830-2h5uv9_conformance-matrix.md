# TASK-260830-2h5uv9 conformance matrix — rev5

Authority is pinned `internal/specdoc/SPEC.v0.7.0.md`: §11.4 union rules 1–7 and “no last-writer-wins”; §5.3 lease tuple ordering and losing branch; §10 immutable object classes. The production boundary is `DurableIndex.SyncFrom`, with projection verified through `DurableIndex.RebuildProjection`.

## Acceptance coverage

| Acceptance criterion / behavior | Production entry | Executed evidence | Narrowing mutant evidence |
| --- | --- | --- | --- |
| Reorder × duplicate × gap-fill × both clock skews × partial peer × 1–3 rounds either converges to independent six-root/projection reference or returns explicit typed conflict | `DurableIndex.SyncFrom` → fetch/add; `RebuildProjection` | `TestDurableSyncGeneratedPerturbationProduct`; 38,000 valid-union plus 37,765 conflict cases. N≤5 orders exhaustive; N=6 uses 34 deterministic samples. | `sync-missing-record-fetch`; `union-lease-created-at-winner`; `sync-partial-peer-regresses-local-root`; `sync-repeat-record-quarantine-write` |
| Same-identity/different-byte conflict remains audited beside peer-only objects and after retries | `DurableIndex.SyncFrom` common-ID audit → quarantine/abort | `TestDurableSyncConflictWithPartialOverlapPeer`; generated conflict class; literal `integrity_failure` assertions and exact quarantine bytes | `sync-common-audit-partial-overlap-peer`; `sync-common-audit-only-without-missing`; `sync-common-audit-first-sync-only` |
| Every common ID is audited at first/middle/last positions in one namespace, crossed with order, duplicate, skew, conflict round and peer-only identities | `DurableIndex.SyncFrom` common-ID audit → quarantine/abort | `TestDurableSyncConflictAuditsEveryCommonIDPositionWithPeerOnlyRecord`; 432 generated cases; runs in default package suite | `sync-common-audit-first-id-only`; `sync-common-audit-last-id-only`; `sync-common-audit-even-ids-only`; all individually killed by this named test |
| Audit has no condition on missing-set size, peer-set size, or namespace index | `DurableIndex.SyncFrom` and in-module callees | `TestDurableSyncCommonIDAuditIsUnconditionalInAST`; behavioral tests above | `sync-common-audit-ast-control`; `sync-common-audit-early-namespace-return`; `sync-common-audit-first-namespace-only` |
| Partial peer cannot regress roots; repeated sync after convergence is a byte-identical no-op | `DurableIndex.SyncFrom` | `TestDurableSyncPartialPeerNeverRegressesLocalRoots`; `TestDurableSyncDuplicateDeliveryIsByteIdentical`; generated reference checks snapshot all store files | `sync-partial-peer-regresses-local-root`; `sync-repeat-record-quarantine-write` |
| Schema/namespace mismatch and unresolved acknowledgement refuse with literal typed errors, then recover when the missing Tombstone arrives | `DurableIndex.SyncFrom` → object validation/closure | `TestDurableSyncRefusesPeerNamespaceMismatch`; `TestDurableSyncUnclosedAcknowledgementThenRecovers`; exact `integrity_failure` checks | `sync-peer-namespace-mismatch`; `sync-unclosed-tombstone-ack` |
| Skip gates cannot be hidden; bytecode artifacts do not enter the repository tree | AST census; repository tree walk | `TestNoUnjustifiedTestSkips`; two planted-skip tests; `TestPackageTreeHasNoPythonBytecode`; planted bytecode negative test | `skip-census-unjustified-planted-skip`; `skip-census-token-preserving-behavior`; `TestPythonBytecodeHygieneRejectsPlantedArtifacts` |

**Measured task acceptance ratio: 1 of 2 criteria fully driven by named tests.** The second criterion's contract fixtures, negative/refusal, crash recovery and idempotency clauses are exercised by named tests; its “no unsupported capability is advertised” part is a stated boundary because this internal package introduces no public advertisement entry point. README/package documentation makes no new capability claim. The brief contains no formal surface table; this absence is recorded as a brief gap.

## Registry edge decode

| Story acceptance case | Clause bindings containing the case | Case production declaration / decoded clause owner |
| --- | --- | --- |
| `story-260830-nxqqaw-inventory-exchange` | §11.4 | Case declaration: `internal/merkleinventory/serve.go` `Dispatch`; binding owner: `internal/merkleinventory/durable.go` `SyncFrom` |
| `story-260830-147hsj-durable-union-projection` | §§5.3, 10.7, 11.4 | `internal/sessrepo/lease_store.go` `CompareAndSwapLease`; `internal/merkleinventory/durable.go` `SyncFrom` |
| `story-260830-2h5uv9-order-duplicate-gap-convergence` | §§5.3, 10.7, 11.4 | `internal/sessrepo/lease_store.go` `CompareAndSwapLease`; `internal/merkleinventory/durable.go` `SyncFrom` |

Decoded registry audit confirms every row appears in each listed clause's acceptance-case list and lists its test declarations in `registry-decode-rev5.md`. The rederived canonical digest is `2302a10a0174ada997614ea7d6112add30e9b2328f6b25ea796ea175f48568cd`. Tracecheck is green; README pin passes and the wrong-figure plant fails as expected. Section-scoped runs refuse partial bindings at §5.3 (7/8), §10.7 (1/18), and §11.4 (6/7); gaps are explicitly owned elsewhere.

## Out-of-contract rows and stated bounds

| Row | Acceptance/specification bound |
| --- | --- |
| Raw blobs, transfer staging/retry, destination materialization | §11.5–§11.6 owned by the transfer/materialization stories; this task implements immutable JSON anti-entropy only |
| Lease takeover issuance | §5.3#5 is outside this union task; no takeover flow is added |
| Tombstone lifecycle/deletion, acknowledgement authority/disposition/retention | §10.7 clauses beyond #13 are outside this immutable-union criterion |
| Arbitrary divergent event DAG reconstruction | Projection consumes the `sessrepo` authoritative chain after union byte checks |
| Physical power loss and native Windows crash behavior | Process-crash/restart is tested; physical power-loss/native Windows crash is not claimed |
| All UUID/digest values, N>6, all N=6 permutations | `canonicaljson`/`scalar` own identity gates; generated range is N≤6 and 34 deterministic N=6 orders |
| Coverage-map surface table | Producer brief has no formal surface table; retained as a stated brief gap, not a waiver |

The rev5 rework diff adds production-entry tests, mutation harness rows, traceability bindings/digest pins and docs; it does not change `DurableIndex` production behavior. `task-board.config.json` is byte-identical to the fork base.
