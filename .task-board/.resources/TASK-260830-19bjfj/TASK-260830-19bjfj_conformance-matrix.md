# TASK-260830-19bjfj conformance matrix — fuzz-rpc-operations-and-version-skew (rev2)

Authority: `internal/specdoc/SPEC.v0.7.0.md` (§11.2-11.3, §17.1-17.4).
Story boundary: mesh-rpc-framing-and-negotiation (STORY-260830-4qojoz).
Verdict key: DRIVEN (named committed test through the production entry) or
STATED BOUND (declared, with owner). Coverage: **23 of 29 rows driven**,
6 stated bounds. No row claims a test that does not exist; every named test
below is in the candidate tree or on trunk as cited.

## (1) Closed operation bodies

| # | Row | Verdict | Production call site | Named test |
| --- | --- | --- | --- | --- |
| 1 | `hello` body: missing member, extra member, wrong-type member each refuse with the pinned class | DRIVEN | `rpcwire.DecodeRequest` (`internal/rpcwire/envelope.go`) | `TestClosedBodyMemberSweep` (new, 15 subtests, `errors.Is ErrHello`) + `TestHelloRefusals` (trunk) |
| 2 | `inventory.roots` body: missing/extra/wrong-type member each refuse with the pinned class | DRIVEN | `rpcwire.DecodeRequest` | `TestClosedBodyMemberSweep` (7 subtests, `errors.Is ErrInventory`) + `TestInventoryNamespacesAndCardinality` (trunk) |
| 3 | Other 22 §11.3 operation bodies (`health.get`, `inventory.children`, `objects.get`, `transfer.begin`, `transfer.status`, `chunks.put`, `transfer.validate`, `transfer.commit`, `materialize.prepare`, `materialize.commit`, `materialize.status`, `materialize.finalize`, `materialize.rollback`, `lease.refresh`, `tombstone.ack`, `session.status`, `session.stop`, `handoff.prepare`, `handoff.quiesce`, `handoff.stop`, `handoff.commit`, `handoff.abort`) | STATED BOUND | none (opaque by design; `TestOpaqueOperationBoundary` pins passthrough; the I-JSON object gate over opaque bodies IS driven by `TestOpaqueBodyMustBeObject` through `rpcwire.DecodeRequest`) | — |
| 4 | Every bound of the parsed bodies at both edges and one step outside | DRIVEN | `rpcwire.DecodeRequest` | `TestClosedBodyBoundsWitnessed` (new: nonce 16B admit/15B refuse, contracts 16 admit/17 refuse, line/object floors + independence) + `TestHelloRefusals`, `TestLimitsAndUntrustedIsolation` (trunk) |
| 5 | `Namespace` admitted only by its exact per-major vocabulary | DRIVEN | `rpcwire.DecodeRequest` via `decodeNamespaces` | `TestNamespaceVocabularyPerMember` (new: every member admitted singly, near-miss/future members refused) + `TestInventoryNamespacesAndCardinality` (trunk) |
| 6 | `LeaseExpectation`, `LeaseHead`, `GroupMemberExpectation`, `WorkspaceGroupExpectation`, `MaterializationKind`, `MaterializationIntent` exact vocabularies | STATED BOUND | none (no RPC body parser consumes them; lease/fencing shapes belong to the fencing/sessrepo owners) | — |
| 7 | `initiator_host_id` on every mutation body, exact lease expectation on session mutations, both revalidated immediately before mutation | STATED BOUND | none (mutation bodies are opaque to this story's codec; authorization belongs to the hosttrust/fencing owners) | — |

## (2) Fuzzing

| # | Row | Verdict | Production call site | Named test |
| --- | --- | --- | --- | --- |
| 8 | Body fuzzer keyed over the operation vocabulary reaches the body parsers | DRIVEN | `rpcwire.DecodeRequest`/`EncodeRequest`/`Hello` | `FuzzClosedOperationBodies` (new, 38 in-code seeds incl. valid + must-reject + non-object bodies; oracle: identity + object gate + round-trip + member sets, bounds owned by the unit suite; control plants FAIL, logs `fuzz-controls/bodies/` + `p3-controls/p3-1-bodies-object-oracle.log`) |
| 9 | Namespace fuzzer with spec-transcribed oracle | DRIVEN | `rpcwire.DecodeRequest` | `FuzzNamespaceVocabulary` (new; control plant FAILS, log `fuzz-controls/nsvocab/`) |
| 10 | Unknown-field fuzzer over raw lines | DRIVEN | `rpcwire.DecodeRequest`/`EncodeRequest` | `FuzzUnknownFields` (new, 7 in-code seeds incl. the hello-body-extra seed; control plants FAIL, logs `fuzz-controls/unknownfields/` + `p3-controls/p3-3-unknownfields-hello-body-seed.log`) |
| 11 | New targets registered as bounded validation commands | DRIVEN | `task-board.config.json` (+3 rows, same shape as the existing envelope row) | Config diff + `fuzz-smokes/*.log` (all exit 0, run by the producer) |

## (3) Section 17 compatibility

| # | Row | Verdict | Production call site | Named test |
| --- | --- | --- | --- | --- |
| 12 | Namespaced `extensions` admitted in opaque bodies; the same data at envelope top level refuses (both directions) | DRIVEN | `rpcwire.DecodeRequest`/`EncodeRequest` | `TestExtensionsDirections` (new) + `TestUnknownFieldsRefusedEverywhere` (new) |
| 13 | No major selected by coercion: only-higher, only-lower, empty, duplicate, unsorted, out-of-range-count offers each refuse pinned | DRIVEN | `meshneg.Negotiate` (`internal/meshneg/negotiate.go`) | `TestNoMajorSelectedByCoercion` (new, literal `incompatible_protocol`/6) + `TestPeerOfferRefusals`, `TestSelectHighestCommon` (trunk) |
| 14 | Highest mutually supported minor within a common major | STATED BOUND | none (no minor-selection entry exists; `Negotiate` selects majors only — predecessor bound retained) | — |
| 15 | A 2.0.0 peer rejects major 1 rather than reinterpreting it | DRIVEN | `rpcwire.DecodeRequest`/`EncodeRequest`/`ContractProfile`/`Namespaces` and `meshneg.Negotiate` | `TestMajor1RejectedNotReinterpreted` (new, `ErrVersion` at every entry) + `TestMajor1RejectedThroughNegotiation` (new, frame + rpc-array lanes) |
| 16 | Error version fixed by the containing protocol, never negotiated (RPC 2 → 1.0.0, RPC 4/5 → 1.3.0); a peer-supplied foreign error schema refuses | DRIVEN | `rpcwire.DecodeResponse` via `axerror.DecodeBound` | `TestErrorSchemaNotNegotiated` (new: RPC-2/1.3.0, RPC-5/1.0.0, RPC-4/1.0.0 refuse with `ErrVersionMismatch`; bound pairs admit) + `TestHistoricalFailureBindings` (trunk) |

## (4) Replay and lost response

| # | Row | Verdict | Production call site | Named test |
| --- | --- | --- | --- | --- |
| 17 | Mismatched echo refuses (ID, version, nonce echo) | DRIVEN | `rpcwire.DecodeResponse` | `TestResponseCorrelationClasses` (new: `ErrCorrelation`/`ErrVersion` sentinels) + `TestResponseRefusals` (trunk) |
| 18 | Duplicate envelope `request_id`: identical body replays / changed body is `idempotency_mismatch` | STATED BOUND | none (envelope `request_id` is correlation-only per §11.3; no responder replay cache exists in the landed code) | — |
| 19 | Operation-key retry: identical body replays byte-identical result, changed body is `idempotency_mismatch` | DRIVEN (adopted owner + new pin) | `matjournal.Store.Create` (`internal/matjournal/store.go`) | `TestCreateReplayAndConflict` (trunk: byte-identical replay + literal `idempotency_mismatch` at staging) + `TestRPCChangedBodyAfterCommitRefusesMismatch` (new: committed-phase member, literal `idempotency_mismatch`, single retained journal) |
| 20 | Lost-response recovery through `materialize.status`: true durable phase learned, blind retry never doubles a committed effect | DRIVEN (adopted owner + new pin) | `matjournal.Store.Get`/`Create` | `TestRPCBlindRetryAfterCommitNeverDoubles` (new: committed phase learned, single journal, byte-identical replay) + `TestRecoverTerminalReplays`, `TestTransitionGuards` (trunk) |

## (5) Framing

| # | Row | Verdict | Production call site | Named test |
| --- | --- | --- | --- | --- |
| 21 | 8388608 accepted, 8388609 refused | DRIVEN | `rpcwire.DecodeRequest`/`EncodeRequest` | `TestFramingBounds` (new, both directions, literal byte counts incl. the exact encode edge) + `TestFrameRefusals` (trunk) |
| 22 | Embedded newline, truncated line, valid JSON non-object | DRIVEN | `rpcwire.DecodeRequest`; `hostchannel.FrameReader.ReadLine` (truncation, adopted) | `TestFramingBounds` (new: newline + `null`/`[]`/string/number/bool) + `TestHostileDisconnectPhases/hello_half_frame_close` (trunk: truncated pre-hello frame closes unframeable, handler never runs) + `TestFrameRefusals` (trunk) |
| 23 | Unpadded base64url accepted, padded/standard-alphabet refused | DRIVEN at the nonce gate | `rpcwire.decodeHello` via `DecodeRequest` | `TestClosedBodyBoundsWitnessed` (new: 16B unpadded admit, padded refuse, standard-alphabet refuse; class narrowing `nonce-admits-standard-alphabet` killed) |
| 24 | Chunk-data base64url strictness for `chunks.put` | STATED BOUND | none (`chunks.put` bodies are opaque; no chunk-data validator exists) | — |
| 25 | `max_object_bytes` enforced independently of `max_line_bytes` | DRIVEN | `rpcwire.decodeHello` via `DecodeRequest` | `TestClosedBodyBoundsWitnessed` (new: each floor refuses while the other is generous; exact floors admit) |

## Contract fixtures, negatives, crash/idempotency, capability discipline

| # | Row | Verdict | Production call site | Named test |
| --- | --- | --- | --- | --- |
| 26 | Exact contract fixtures pass | DRIVEN | `rpcwire.DecodeRequest`/`DecodeResponse`; `matjournal` (adopted) | `TestPinnedHelloProfiles` (v2/v3/v4/v5 hello fixtures + round-trips, trunk), `TestLimitsAndUntrustedIsolation` (16-version bound, trunk), `TestOpenedFixtureMatchesProduction` (`MAT-TASK-BOARD-OPENED-STATUS`, trunk). The §11.3 `lease_conflict` failure example is covered structurally (shape + class), not byte-exact — no test pins its literal bytes. |
| 27 | Negative/refusal cases pass | DRIVEN | entries above | Every DRIVEN row carries refusal lanes asserting the pinned sentinel/class; meshneg refusals assert literal `incompatible_protocol`/6 and `invalid_config`/3 (no production-constant comparisons); error-schema lanes assert `ErrVersionMismatch`; N2 rows assert literal offer sets |
| 28 | Crash/idempotency evidence for durable mutations | DRIVEN | (purity) `rpcwire`/`meshneg`; (durability) `matjournal` (adopted) | Purity bound: RPC/meshneg mutate nothing (`TestNegotiationIsDeterministic`, stateless codec; no store, clock, or I/O). Durability: `TestCreateReplayAndConflict`, `TestCrashBetweenJournalAndReceiptReplays`, `TestRecoverTerminalReplays`, `TestRPCBlindRetryAfterCommitNeverDoubles`, `TestRPCChangedBodyAfterCommitRefusesMismatch` |
| 29 | No unsupported capability advertised | VERIFIED BY CONSTRUCTION | candidate diff inventory | The candidate adds no production export and changes no production behavior file (only the traceability pin constant): `git status` shows tests, harnesses, registry, config, README, LOGBOOK only. README carries no `ax`/doctor/capability claim; `TestExposureCarriesNoInventory` pins the peer view member set. |

## Inherited advisories (predecessor review RUN-260917-b1ebcb)

| Item | Disposition | Killer |
| --- | --- | --- |
| N1 key-membership gate per key | FIXED (test gap, production correct) | `TestPeerOfferKeyMembershipPerKey` (176 subtests); probes `shape-admits-missing-{checkpoint,task-board-bundle,directory-receipt,backend-evidence}` killed |
| N2 `Refusal.Local` on offer refusals | FIXED | Local asserted against the literal `[2 3 4]`/`[5]` sets in every N1 subtest + `TestNegotiateOfferRefusalCarriesLocal`; probe `negotiate-drops-local-fact` killed |
| N3 local-side vocabulary member 1 | FIXED + reachability bound | `TestSelectHighestCommon/refuse-local-names-1` (beside a common major); probe `select-admits-local-1` killed |

## Clause map (§11.2/11.3/17.1-17.4 normative inventory, v0.7.0 lines)

| Clause | Disposition |
| --- | --- |
| 11.2#2, #3, #4 (lines 7364/7408/7420) | Discharged in registry (partial 3/5); 11.2#1/#5 in gap (hello-first sequencing, RPC-5 lane only) |
| 11.3#1-#27 | Unevidenced 0/27: no production code parses a digest-identity array, so no clause is claimed (P2-1) |
| 17.1#1, #4, #5 (lines 15567/15589/15602) | Discharged in registry (partial 3/6); 17.1#2/#3/#6 in gap with owners |
| 17.2#1, 17.3#1-#3 | Existing unevidenced owners retained (this story adds no measurable clause) |
| 17.4#4 (line 15661) | Discharged in registry (sliver 1/4); 17.4#1-#3 in gap (no upgrade/downgrade flow) |
