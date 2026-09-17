# Mesh RPC wire components

Owner: TASK-260830-z1yxg9 in STORY-260830-1kiyj6. Authority: embedded AX
v0.6.0 SPEC.md, commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`.
This package covers Mesh RPC 2/3/4/5 structural envelopes; RPC 5.0.0
reuses the exact RPC-4 shapes with `contracts.rpc` `["5.0.0"]` per
11.10.1, and Structured Error stays statically bound to 1.3.0 for RPC
majors 4 and 5. The authenticated consumer is `internal/hostchannel`
(Host Channel 1.0.0), which binds both hello IDs to verified enrolled
identities under the current hosttrust generation before dispatch.

`EncodeRequest` / `DecodeRequest` and `EncodeSuccess` / `EncodeFailure` /
`DecodeResponse` implement the reusable wire surface. `EncodeRejection` can
frame a supported-version bootstrap failure despite an invalid body; it returns
no bytes for an unframeable identity/version/JSON/size. All output is LF-free
for line framing; input is one LF-terminated line. The codec does not open
connections or invoke transport methods itself; `internal/hostchannel`
frames these lines over its TLS stream.

The package owns no responder, handshake state, authenticated identity,
capability advertisement, operation dispatch, or pending-request queue. Its
success means structural validity only. In particular, `Request.Hello().HostID`
is an untrusted claim, `Response.OK()` means only the envelope's `ok=true`, and
`OfferedLimits` computes minima without making them authenticated or installing
them on a connection. Repeating `DecodeResponse` is possible; this codec does
not claim replay resistance. Received randomness/freshness cannot be proven by
nonce shape validation. `NewNonce` obtains 256 random bits from crypto/rand.

## Implemented surface and traceability

| Pinned rule | Production call site | Named test | Bound |
| --- | --- | --- | --- |
| 11.2 exact envelopes, UUIDv7 ID echo; 17.1 containing version first | DecodeRequest, DecodeResponse, EncodeRequest, EncodeSuccess | TestPinnedHelloProfiles, TestRequestRefusals, TestResponseRefusals | Correlation of one supplied request; no connection sequencing |
| 1.6 closed objects and common JSON model | DecodeRequest/DecodeResponse through object/exact | TestFrameRefusals, TestRequestRefusals | Shared canonicaljson validates bytes; opaque operation semantics remain with operation owners |
| 11.2 fourteen-key hello map, nonce/echo and limit floors | DecodeRequest, DecodeResponse, Request.Hello, OfferedLimits | TestHelloRefusals, TestLimitsAndUntrustedIsolation | v2 arrays have 1–16 bytewise sorted unique semvers; selecting mutually supported versions remains TASK-260830-219okr |
| 11.8/11.9 exact historical 24/25-key maps | DecodeRequest/DecodeResponse | TestPinnedHelloProfiles, TestHelloRefusals | Exact pinned 2.0.0/3.0.0/4.0.0 syntax; no automatic major or minor negotiation |
| 11.3/11.8/11.9 inventory.roots 6/7/8 namespace vocabulary and bounds | DecodeRequest, DecodeResponse, EncodeSuccess | TestInventoryNamespacesAndCardinality | Explicit subsets remain valid; results must match requested namespaces. No Merkle digest verification, schema membership or object exchange |
| 11.10.1 RPC 5 shapes and static Error 1.3.0 binding | ContractProfile, Namespaces, EncodeFailure -> axerror.DecodeBound | TestPinnedHelloProfiles, TestInventoryNamespacesAndCardinality, TestHistoricalFailureBindings | Exact RPC-4 shapes with contracts.rpc [5.0.0]; 8 namespaces |
| 11.2/15.1/17 static historical errors and bootstrap framing | EncodeFailure, DecodeResponse, EncodeRejection -> axerror.DecodeBound | TestHistoricalFailureBindings, TestBootstrapRejectionFraming | hostchannel Serve chooses/sends the one framed refusal and closes |
| 11.2 8 MiB line bound | DecodeRequest, EncodeRequest, DecodeResponse | TestFrameRefusals, TestWriterAndAccessorBounds | Codec line limit; partial I/O is owned by sshtransport |

**7 of 7 full RPC AC rows are proven through the authenticated runtime
consumer** (`internal/hostchannel` Serve/Dial/Call): correlation, hello
maps/nonce, inventory namespace/cardinality, offered limits, historical
errors, bilateral admission before dispatch, and deadlines/generation
recovery. The per-row call sites and tests are accounted in
`internal/hostchannel/README.md`. Codec tests alone prove the structural
rows only.

Other operation bodies are retained as isolated raw JSON objects, without
schema or permission validation. No caller may dispatch from this library's
success. Object limits are advertised structural offers; object decoding,
negotiated transfer enforcement and durable mutation/recovery remain with their
operation owners. There is no durable state change in this package, hence no
new crash/idempotency behavior to attest.

## Existing owners and unchanged decision

`sshtransport` already owns the fixed 8 MiB LF framing, bounded partial I/O,
process teardown and earlier-of-caller/configured RPC timeout. Section 6.2
provides the 300-second default and 10–3600-second configuration range; sections
11.2–11.3 do not define an additional numeric hello timeout. This package adds
no competing timer or stream pump. `peeridentity` retains configured identity
selection/comparison; `axerror` retains error validation and static binding.

The v0.6.0 Section 11.10 binding resolves the association decision: identity
comes from verified enrolled host certificates (mutual TLS 1.3) before hello
and dispatch, both hello IDs are checked against the current authorization
generation, and no expected-peer injection, verified flag, claim-selected
target or fallback is introduced. The membership-only interpretation stays
withdrawn. TASK-260830-219okr and TASK-260830-2x16gz retain their complete
subsequent negotiation and hostile-peer obligations.

## Verification tools

| Purpose | Command | Output |
| --- | --- | --- |
| Component behavior | `go test ./internal/rpcwire -count=1 -v` | stdout; task logs in `.temp/TASK-260830-z1yxg9/` |
| Races and coverage | `go test ./internal/rpcwire -race -count=1 -cover` | stdout and task log |
| Behavioral narrowing and controls | `python3 internal/rpcwire/mutations.py --output .temp/TASK-260830-z1yxg9/mutations` | Per-mutant Go overlay, JSON events, real exit, named failures and Markdown table; source remains unchanged |
| Bounded fuzz smoke | `go test ./internal/rpcwire -run '^$' -fuzz '^FuzzUntrustedEnvelopes$' -fuzztime=100x -parallel=1` | stdout and task log |

The overlay harness accepts `--only NAME,NAME` for bounded reruns. A kill requires
an actual named failing behavioral test. Compile errors, empty test selections
and instrumentation failures are not kills. Survivors must retain explicit
bounds in the task outcome. No source-only check is added by this package.
