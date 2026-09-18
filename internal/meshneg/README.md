# Mesh RPC major negotiation

Owner: TASK-260830-219okr in STORY-260830-4qojoz. Authority: embedded AX
v0.6.0 SPEC.md, commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`,
Sections 11.2-11.3, 11.8, 11.9, 11.10.1, 15.1-15.3, 17.1-17.4, and 6.6.

`Negotiate` selects exactly one common Mesh RPC major from the local
installation's configuration generation and the peer's decoded hello frame.
Config-4 installations offer exactly RPC 5; explicitly legacy installations
offer the historical dual-stack sets (Config-1: 2; Config-2: 2 and 3;
Config-3: 2, 3, and 4). The peer offer must be single-major and agree with
its framing major; any mixing is refused rather than coerced. A disjoint
pair is refused as no-common-major with the pinned `incompatible_protocol`
class (exit 6), and only from actually known inputs: unframed bytes never
reach this package.

A selection carries the exact pinned shapes for its major (contracts map,
namespace vocabulary, statically bound Structured Error version), rebuilt
from the landed owners rather than copied from peer bytes, plus the peer
exposure: above-core activation the selection does not carry is reported as
explicitly unsupported (`directory_mesh_unsupported`, or the RPC-4
equivalent `terminal_backend_unavailable`), never as zero inventory. Every
outcome is pure data: no wire bytes are framed, no connection is opened, no
capability is advertised, and no admission decision is made. Identity
equality, allowlist membership, and trust generation remain the
`internal/hostchannel` boundary owned by TASK-260830-z1yxg9.

## Implemented surface and traceability

| Pinned rule | Production call site | Named test | Bound |
| --- | --- | --- | --- |
| 6.4/6.5/6.6 generation to offered majors; 11.8/11.9 dual-stack; 11.10.1 RPC 5 only | LocalMajors | TestLocalMajors, TestLocalMajorsTracksConfigVocabulary, TestNegotiateMatrix | Generation input is config.LoadedConfiguration.SourceVersion (never the normalized SchemaVersion); unknown generation refuses invalid_config/3 at the Negotiate entry (TestNegotiateUnknownGeneration), never legacy selection |
| 11.2/11.8/11.9/11.10.1 exact per-major contracts; 17.1 no coercion | PeerOffer | TestPeerOfferRefusals, TestSemverAgreementWithRPCWire, TestPeerOfferTrustsDecodedNonRPCArrays | Key set and rpc array re-validated here; non-rpc array values validated by rpcwire decode and trusted (direct callers must pass decode-produced hellos). v2 1-16 minor ranges negotiate the major; within-major minor selection stays future work |
| Highest common major; 17.2 reader rules | Select | TestSelectHighestCommon | Inputs outside 2/3/4/5 refuse even beside a common pinned major; unparseable frame versions answer incompatible_protocol/6 (unreachable via decode, which only the pinned strings pass) |
| No-common-major refusal, known only; 15.1-15.3 class | Negotiate -> Refusal | TestNegotiateMatrix, TestRefusalShape, TestRefusalClassesResolveFromTheRegistry | Frames nothing; the initiator maps the class to its local error |
| 11.8/11.9 core-only preservation; unsupported activation, never zero inventory | Negotiate -> Decision.Peer | TestPeerExposure, TestExposureCarriesNoInventory | Exposure derives from the negotiated major alone |
| 6.6 no downgrade of Config-4; legacy never coerces RPC 5 | LocalMajors, Negotiate | TestConfig4NeverDowngrades, TestLegacyNeverSelectsRPC5 | Config-4 offers exactly [5] |
| 11.10.1 admission separation (identity, allowlist, generation) | Negotiate (ignores them) | TestNegotiationIsNotAdmission | Admission still required before dispatch |
| 6.6 no sniffing/extension/argv/env/handshake fallback | Negotiate (no such input) | TestNoFallbackSurface | Pure function; statelessness and determinism pinned by test |
| 11.2/11.8/11.9/11.10.1 shapes; 15.1/15.3/11.10.1 error bindings | decisionFor via rpcwire, axerror | TestDecisionShapes | Shapes rebuilt from profiles, never copied from the peer |

**11 of 11 AC rows are driven through the production entries by named
tests** (matrix in `TASK-260830-219okr_conformance-matrix.md`): common-major
selection, core-only preservation, directory exposure, backend exposure,
no-common-major refusal, Config-4 no-downgrade, admission separation,
cross-major refusal, no-fallback surface, exact per-major shapes with bound
error versions, and pure-data outcomes.

The package performs no I/O and keeps no state. Its direct imports are
`errors`, `fmt`, `regexp`, `slices`, `strconv`, `strings` plus the `rpcwire`
and `axerror` owners (no `os`, `io`, `net`, `time`, `rand`, or `exec`), and
the owners are consulted through pure functions only (`ContractProfile`,
`Namespaces`, `ExitCodeFor`, `BindingFor`), so there is no crash or
idempotency surface. Determinism over the full input space is pinned by
`TestNegotiationIsDeterministic`.

## Checks

```bash
go test ./internal/meshneg -count=1 -v
go test ./internal/meshneg -race -count=1 -cover
PYTHONDONTWRITEBYTECODE=1 python3 internal/meshneg/mutations.py --output .temp/TASK-260830-219okr/mutations-meshneg
```

The mutant harness applies 31 narrowing plants plus a neutral control and
one harmless applied SURVIVED control; every narrowing plant is killed by
its named test with per-plant raw logs, and source originals remain
untouched (overlay run).
