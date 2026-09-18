# TASK-260830-219okr clause coverage (package-local)

Normative authority: `internal/specdoc/SPEC.v0.6.0.md` at commit
`0cbdf100dbf84df50c64f792b1f940e3a67859a6` (relux-works/agent-session-manager-spec@v0.6.0).
Historical scope retained: SPEC v0.5.0 (commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`),
Sections 11.2-11.3 and 17. This leaf does not edit `internal/traceability`;
the story's final leaf carries the registry bindings and re-pin.

## Covered clauses

| Clause | Rule | Call site | Test |
| --- | --- | --- | --- |
| 6.4 | Config-2 directory extension pairs with RPC 3 | LocalMajors ("2.0.0" offers 2,3) | TestLocalMajors, TestLocalMajorsTracksConfigVocabulary, TestNegotiateMatrix |
| 6.5 | Config-3 TerminalBackend extension pairs with RPC 4 | LocalMajors ("3.0.0" offers 2,3,4) | TestLocalMajors, TestLocalMajorsTracksConfigVocabulary, TestNegotiateMatrix |
| 6.6 | Config-4 requires Host Channel 1 and RPC 5 for every peer | LocalMajors ("4.0.0" offers exactly 5) | TestConfig4NeverDowngrades, TestLocalMajorsTracksConfigVocabulary |
| 6.6 | Legacy dual-stack obligations never require Config-4 to accept legacy | Negotiate (Config-4 x legacy refuses) | TestConfig4NeverDowngrades |
| 6.6 | Unknown configuration is a refusal, never legacy selection | Negotiate via LocalMajors default | TestNegotiateUnknownGeneration (unknown generations x every frame at the production entry), TestLocalMajors unknown-* |
| 6.6 | No sniffing, extension, argv, env, or failed handshake selects a fallback | Negotiate signature and body | TestNoFallbackSurface |
| 11.2 | 14-key hello contracts; unknown members fail; absent contract is incompatible_protocol | PeerOffer shape checks | TestPeerOfferRefusals/shape, TestSemverAgreementWithRPCWire |
| 11.2 | Unsupported major closes without a peer error frame; initiator emits locally | Refusal (frames nothing) | TestRefusalShape, TestRefusalClassesResolveFromTheRegistry |
| 11.3 | Closed operation registry and envelope discipline | (precondition: caller decodes via rpcwire first) | decodedHello fixtures in every matrix test |
| 11.8 | v3/v2 dual-stack; v2 negotiation is core-only; peer exposed as directory_mesh_unsupported, never zero inventory | decisionFor, PeerView | TestPeerExposure (literal token pins), TestExposureCarriesNoInventory |
| 11.8 | Exact 24-key map and 7 namespaces; v2 bounds invalid in v3 | PeerOffer, decisionFor via rpcwire | TestPeerOfferRefusals, TestDecisionShapes |
| 11.9 | v4 serves v3 dual-stack, preserves v2 interop; backend evidence unsupported rather than empty; both unsupported on v4/v2 | decisionFor, PeerView | TestPeerExposure (literal token pins) |
| 11.9 | Exact 25-key map and 8 namespaces; v2/v3 bounds invalid in v4 | PeerOffer, decisionFor via rpcwire | TestPeerOfferRefusals, TestDecisionShapes |
| 11.10.1 | RPC 5 has exact RPC-4 shapes with protocol_version and contracts.rpc 5.0.0; error stays bound to 1.3.0 | decisionFor via rpcwire, axerror | TestDecisionShapes/5.0.0 |
| 11.10.1 | Other RPC majors close without a frame; incompatible_protocol (exit 6) only when actually known | Refusal, Negotiate inputs | TestConfig4NeverDowngrades, TestLegacyNeverSelectsRPC5 |
| 11.10.1 | Hello identity equality, allowlist, and generation are admission, not negotiation | Negotiate (never consulted) | TestNegotiationIsNotAdmission |
| 15.1 | Mesh RPC statically binds per-major error versions; error never a hello key | decisionFor via axerror.BindingFor | TestDecisionShapes |
| 15.1/15.3 | incompatible_protocol exit 6; invalid_config exit 3; directory_mesh_unsupported and terminal_backend_unavailable exit 6 | refuse, exposure via axerror.ExitCodeFor | TestRefusalClassesResolveFromTheRegistry |
| 17.1 | No major selected by coercion | PeerOffer mixing refusal, Select | TestPeerOfferRefusals/mixed, TestSelectHighestCommon |
| 17.2 | Reader rejects unsupported major; accepts same/lower minor | pinnedFrameVersions exact match | TestPeerOfferRefusals/frame-versions |
| 17.4 | Report activation unavailable rather than omit the session | BackendEvidenceUnsupportedCode binding | TestPeerExposure |

## Stated bounds (not covered by this leaf)

| Bound | Reason |
| --- | --- |
| Within-major minor selection for v2 1-16 ranges | Major negotiation only; any 2.x range selects major 2 (pinned by admitted-ranges) |
| Handshake sequencing (hello-first, request_id echo, one framed refusal then close) | Owned by rpcwire decode plus the connection responder; Negotiate takes decoded hellos |
| Hostchannel admission (identity equality, allowlist, trust generation) | Owned by TASK-260830-z1yxg9; this package proves separation by ignoring those members |
| Fuzzing of rpc operations and version skew | Sibling TASK-260830-19bjfj |
| Registry bindings and digest re-pin in internal/traceability | Story final leaf |
| Crash/idempotency evidence | No durable write exists in this package. Direct imports are errors, fmt, regexp, slices, strconv, strings plus the rpcwire and axerror owners (no os, io, net, time, rand, or exec); the owners are consulted through pure functions only (ContractProfile, Namespaces, ExitCodeFor, BindingFor). Determinism pinned by TestNegotiationIsDeterministic |
| Non-rpc hello array values | Validated exactly by rpcwire decode (v3+ per-key arrays; v2 1-16 sorted-unique semver per key) and trusted by PeerOffer, which re-validates the key set and the rpc array only. Direct callers must pass decode-produced hellos. Bound pinned by TestPeerOfferTrustsDecodedNonRPCArrays |
| Unparseable frame-version class | The gate answers incompatible_protocol/exit 6 while Section 15.1 assigns transport_failure to an unrecognizable version; unreachable through rpcwire decode, which only the four pinned strings pass |

## Final-leaf closure (TASK-260830-19bjfj, test-only; production untouched)

| Item | Disposition |
| --- | --- |
| N1 key-membership gate pinned per key | `TestPeerOfferKeyMembershipPerKey` derives missing-key and substituted-key cases from `rpcwire.ContractProfile(frame)` for every frame and every key through `Negotiate`; four per-key narrowing probes killed |
| N2 `Refusal.Local` on offer refusals | Asserted against the literal offer sets (`[2 3 4]`, `[5]`, never derived from production) in every key-membership case plus `TestNegotiateOfferRefusalCarriesLocal` mixed-major cases; `negotiate-drops-local-fact` probe killed |
| N3 local-side vocabulary member 1 | `refuse-local-names-1` row beside a common major; `select-admits-local-1` probe killed. Reachability bound: `LocalMajors` never yields 1, so only a direct `Select` caller observes the arm |
| No-coercion shapes at the composed entry | `TestNoMajorSelectedByCoercion` (only-higher, only-lower, empty, duplicate, unsorted, out-of-range-count) and `TestMajor1RejectedThroughNegotiation` through `Negotiate` |
| Registry bindings and digest re-pin | Done by this leaf against the adopted `ownership.v0.7.0.json`: 11.2 partial 3/5, 17.1 partial 3/6, 17.4 sliver 1/4; 11.3 stays unevidenced (no test drives a digest-identity array); digest `b4ae9091…`; 17.2/17.3 keep their existing unevidenced owners |
