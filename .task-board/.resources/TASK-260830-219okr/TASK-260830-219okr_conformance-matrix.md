# TASK-260830-219okr conformance matrix — dual-stack major negotiation

Authority: `internal/specdoc/SPEC.v0.6.0.md` (commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`),
Sections 6.4/6.5/6.6, 11.2-11.3, 11.8, 11.9, 11.10.1, 15.1-15.3, 17.1-17.4.
Historical scope: SPEC v0.5.0, Sections 11.2-11.3 and 17 (unchanged behavior retained).

Coverage: **11 of 11 AC rows driven** through production entries by named committed tests.
Production package: `internal/meshneg` (`negotiate.go`).

Matrix rev3 (rework rev4): rows 3/4 now pin the literal exposure tokens at the
`Negotiate` entry (no expectation derives from the production constants); row 5
credits the unknown-generation refusal to `TestNegotiateUnknownGeneration`
through `Negotiate`, with `TestLocalMajors unknown-*` kept as the helper-level
pin; four narrowing mutants added (two same-exit code swaps, one
Negotiate-site fallback, one 5.x frame edge).

| # | AC row | Pinned clauses | Production call site | Named test(s) | Verdict |
| --- | --- | --- | --- | --- | --- |
| 1 | Negotiate supported majors: highest common major for legacy generations; Config-4 selects RPC 5 | 6.4, 6.5, 6.6, 11.8, 11.9, 11.10.1, 17.1 | `meshneg.Negotiate` via `LocalMajors`, `PeerOffer`, `Select` | `TestNegotiateMatrix` (16 generation-x-major pairs), `TestSelectHighestCommon` admit-*, `TestLocalMajors`, `TestLocalMajorsTracksConfigVocabulary` | PASS |
| 2 | Preserve core-only peers: v2 negotiation performs core sync only, no major coercion | 11.8, 11.9, 17.1 | `meshneg.Negotiate`, `meshneg.PeerOffer` | `TestNegotiateMatrix` (`*-x-2.0.0`), `TestSemverAgreementWithRPCWire` other-major-* (decode admits syntax, negotiation refuses coercion) | PASS |
| 3 | Expose unsupported directory activation as `directory_mesh_unsupported`, never zero inventory | 11.8, 15.3 (1.2.0 exit-6 row) | `meshneg.Negotiate` → `Decision.Peer.Directory` | `TestPeerExposure` (asserts the literal `directory_mesh_unsupported` token at the `Negotiate` entry), `TestExposureCarriesNoInventory`, `TestRefusalClassesResolveFromTheRegistry` (literal codes) | PASS |
| 4 | Expose unsupported backend activation with the RPC-4 equivalent (`terminal_backend_unavailable`), never empty | 11.9, 15.3 (1.3.0 exit-6 row), 17.4 | `meshneg.Negotiate` → `Decision.Peer.BackendEvidence` | `TestPeerExposure` (asserts the literal `terminal_backend_unavailable` token at the `Negotiate` entry), `TestExposureCarriesNoInventory`, `TestRefusalClassesResolveFromTheRegistry` (literal codes) | PASS |
| 5 | Refuse no-common-major with `incompatible_protocol`/exit 6, only when incompatibility is actually known; unknown local config refuses `invalid_config`/exit 3 | 11.2, 11.10.1, 15.1, 15.3, 6.6 | `meshneg.Negotiate`/`Select`/`LocalMajors` → `*meshneg.Refusal` | `TestNegotiateMatrix` (refusal pairs), `TestNegotiateUnknownGeneration` (8 unknown generations x 4 frames through `Negotiate`, asserting code/exit/reason/sentinel and zero `Decision`), `TestSelectHighestCommon` refuse-*, `TestLocalMajors` unknown-*, `TestRefusalShape`, `TestRefusalClassesResolveFromTheRegistry` | PASS |
| 6 | Config-4 endpoint never downgrades below RPC 5 and refuses every legacy frame | 6.6, 11.10.1 | `meshneg.LocalMajors`, `meshneg.Negotiate` | `TestConfig4NeverDowngrades`, `TestLocalMajorsTracksConfigVocabulary` | PASS |
| 7 | Legacy installation offered RPC-5-only refuses instead of coercing | 11.8, 11.9, 17.1 | `meshneg.Negotiate` | `TestLegacyNeverSelectsRPC5` | PASS |
| 8 | Refuse cross-major mixing: mixed rpc arrays, frame/offer disagreement, v2 bound inside v3+ frame | 11.2, 11.8, 11.9, 17.1, 17.2 | `meshneg.PeerOffer` | `TestPeerOfferRefusals` (frame-versions incl. 4.1.0/5.0.1/5.1.0 edges, shape, mixed), `TestSemverAgreementWithRPCWire` | PASS |
| 9 | No byte sniffing, extension field, argv value, environment value, or failed handshake selects a fallback | 6.6 | `meshneg.Negotiate` (no such input exists) | `TestNoFallbackSurface` (environment, argv, stateless, extension-member) | PASS |
| 10 | Exact per-major shapes: 14/24/25/25 contract keys, 6/7/8/8 namespaces, bound Structured Error 1.0.0/1.2.0/1.3.0/1.3.0; RPC 5 carries exact RPC-4 shapes | 11.2, 11.8, 11.9, 11.10.1, 15.1, 15.3 | `meshneg.Negotiate` → `Decision` via `rpcwire.ContractProfile`, `rpcwire.Namespaces`, `axerror.BindingFor` | `TestDecisionShapes` | PASS |
| 11 | Negotiation never substitutes admission and advertises no capability: identity/allowlist/generation ignored; outcomes pure data | 11.10.1, 6.6 | `meshneg.Negotiate` (ignores identity-adjacent members) | `TestNegotiationIsNotAdmission`, `TestNegotiationIsDeterministic`, `TestExposureCarriesNoInventory` | PASS |

## Negative-test inventory (every refusal gate)

| Gate (production site) | Rejected class | Named negative test | Narrowing mutant (killed) |
| --- | --- | --- | --- |
| `LocalMajors` Config-1 row | majors 3, 4, 5 for Config-1 | `TestNegotiateMatrix/1.0.0-x-3.0.0` | `config1-admits-3` |
| `LocalMajors` Config-2 row | majors 4, 5 for Config-2 | `TestNegotiateMatrix/2.0.0-x-4.0.0` | `config2-admits-4` |
| `LocalMajors` Config-3 row | major 5 for Config-3 | `TestNegotiateMatrix/3.0.0-x-5.0.0` | `config3-admits-5` |
| `LocalMajors` Config-4 row | majors 2, 3, 4 for Config-4 | `TestConfig4NeverDowngrades` | `config4-admits-4` |
| `LocalMajors` default | any generation outside 1.0.0-4.0.0 | `TestLocalMajors/unknown-9.9.9` | `unknown-config-admits-9` |
| `Negotiate` unknown-config propagation | legacy fallback at the Negotiate site | `TestNegotiateUnknownGeneration` | `negotiate-unknown-falls-back` |
| `PeerOffer` frame vocabulary | versions outside pinned 2.0.0-5.0.0 | `TestPeerOfferRefusals/frame-versions` | `frame-admits-2.1`, `frame-admits-5.1` |
| `PeerOffer` contracts key count | extra keys (incl. extensions) | `TestPeerOfferRefusals/shape/extra-error-key`, `TestNoFallbackSurface/extension-member` | `shape-admits-extra-key` |
| `PeerOffer` contracts key membership | substituted keys at equal count | `TestPeerOfferRefusals/shape/substituted-key` | `shape-admits-missing-lease` |
| `PeerOffer` rpc array length | empty and 17-entry arrays | `TestPeerOfferRefusals/shape/empty-rpc`, `.../seventeen-rpc` | `rpc-admits-nil-empty`, `rpc-admits-17` |
| `PeerOffer` rpc major agreement | mixed majors, cross-major frames | `TestPeerOfferRefusals/mixed/*` | `mixed-admits-v3-in-v2`, `mixed-admits-v-prefix-value` |
| `PeerOffer` rpc version grammar | non-semver versions | `TestPeerOfferRefusals/mixed/not-semver`, `TestSemverAgreementWithRPCWire` | `semver-admits-v-prefix`, `major-extract-plus-one` (token-preserving) |
| `PeerOffer` rpc sort order | duplicate/unsorted arrays | `TestPeerOfferRefusals/mixed/unsorted` | `mixed-admits-unsorted-pair` |
| `PeerOffer` v3+ exactness | inexact v3/v4/v5 rpc arrays | `TestPeerOfferRefusals/shape/v3-inexact-rpc` | `v3-exact-admits-3.0.1` |
| `Select` input vocabulary | majors outside 2/3/4/5 on either side | `TestSelectHighestCommon/refuse-local-names-6`, `refuse-peer-mixes-1-with-common` | `select-admits-local-6`, `select-admits-peer-1` |
| `Select` highest-common rule | non-highest selection | `TestSelectHighestCommon/admit-highest-wins` | `select-picks-lowest` |
| `Select` disjoint refusal | disjoint offer pairs | `TestSelectHighestCommon/refuse-disjoint` | `select-admits-disjoint-2v3` |
| `decisionFor` directory threshold | directory negotiated below major 3 | `TestPeerExposure/2.0.0` | `exposure-directory-at-2` |
| `decisionFor` backend threshold | backend negotiated below major 4 | `TestPeerExposure/3.0.0` | `exposure-backend-at-3` |
| `decisionFor` exposure codes | same-exit code swaps (wrong registered token) | `TestPeerExposure/2.0.0` (both swaps), `TestPeerExposure/3.0.0` (backend swap) | `exposure-directory-code-swap`, `exposure-backend-code-swap` |
| `decisionFor` error binding | wrong containing contract | `TestDecisionShapes/3.0.0` | `error-binding-provider-contract` |
| `refuse` refusal class | wrong Section 15 code | `TestNegotiateMatrix/3.0.0-x-5.0.0` | `refusal-code-transport-failure` |
| `exposure` version pins | wrong introducing Error version | `TestPeerExposure/2.0.0` | `directory-version-1.0.0`, `backend-version-1.2.0` |
| `shapeRefusal` reason | misreported reason | `TestPeerOfferRefusals/shape/extra-error-key` | `reason-shape-reports-mixed` |

Harness: `internal/meshneg/mutations.py` (31 narrowing probes, all killed; neutral passed;
`detail-text` harmless control survived as designed). Per-plant raw logs, subprocess exits,
and mutant table under `.temp/TASK-260830-219okr/mutations-meshneg/`, archived in the
producer evidence tar. (Rework rev4 adds 4 probes to the rev3 27: two same-exit exposure
code swaps, one Negotiate-site unknown-config fallback, one 5.x frame edge. Tallied from
`results.json`, not the prose.)

## Stated bounds

- Within-major minor selection for v2 1-16 rpc ranges is not performed; any 2.x range
  selects major 2 (`TestPeerOfferRefusals/admitted-ranges`).
- Non-`rpc` hello array values are validated by rpcwire decode (exact per-key arrays
  for v3+, 1-16 sorted-unique semver per key for v2) and trusted by `PeerOffer`,
  which re-validates the key set and the `rpc` array only; direct callers must pass
  decode-produced hellos (`TestPeerOfferTrustsDecodedNonRPCArrays`).
- Unparseable frame versions answer `incompatible_protocol`/exit 6 here while Section
  15.1 assigns `transport_failure` to an unrecognizable version; unreachable through
  rpcwire decode, which only the four pinned strings pass.
- Handshake sequencing (hello-first, request_id echo, one framed refusal then close) stays
  with rpcwire decode plus the connection responder; `Negotiate` takes decoded hellos.
- Admission (identity equality, allowlist, trust generation) stays with TASK-260830-z1yxg9;
  separation is pinned by `TestNegotiationIsNotAdmission`.
- No durable state is written; crash/idempotency surface is absent by construction
  (direct imports exclude os/io/net/time/rand/exec; owners consulted through pure
  functions only; `TestNegotiationIsDeterministic` pins statelessness).
- Traceability registry bindings and digest re-pin belong to the story final leaf; clause
  coverage lives in `internal/meshneg/TRACEABILITY.md`.
- No `ax` command, doctor result, or runtime capability is claimed anywhere in this leaf.
