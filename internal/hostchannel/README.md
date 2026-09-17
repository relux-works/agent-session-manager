# Host Channel (`internal/hostchannel`)

Mutual TLS 1.3 host authentication plus the authorized RPC-5 hello:
Section 11.10 Host Channel 1.0.0 over Mesh RPC 5.0.0. This package is the
authenticated runtime consumer of `internal/rpcwire`.

Authority: relux-works/agent-session-manager-spec@v0.6.0
(commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6`), sections 11.1
(bilateral host admission), 11.2-11.3, 11.9, 11.10-11.10.1, 11.10.4
(HC-* gates) and 17 (structured errors). Section 11.10.5 is an upstream
synthetic source validator, not an AX runtime JSON-policy reader, and is
not implemented here.

## What it does

- Runs one channel over an ordered binary duplex `Stream`
  (`io.Reader`/`io.Writer`/`Close`: no listener, no PTY, no SSH process).
- Negotiates mutual TLS 1.3 exactly: version floor and ceiling 1.3, ALPN
  `ax-host/1`, enrolled roots only (never the system pool), mandatory
  verified client certificates, no session tickets, no client cache, no
  resumption, no early data, no fallback.
- Supplements (never replaces) standard chain/name/time/EKU verification
  with the enrolled-identity checks: exact ALPN presence, no resumption,
  standard-verified chain, exactly one exact enrolled leaf/root/key match,
  current Host Credential Profile 1 validity, and an admitting trust
  state (active, or retiring before `retire_at`).
- Exchanges one hello per side with fresh nonces, the exact RPC-5
  25-key contract map, and the 8 MiB line / 5 MiB object floors. Both
  hello IDs are checked: the *received* ID must equal the verified
  enrolled UUID of the presenting peer under the *current* authorization
  generation, and the client TLS server name must equal the selected
  configured destination. The hello success is validated before any
  other operation moves.
- Dispatches authorized post-hello calls (`Handler`) and guards every
  mutation boundary (`WithMutationAuthorization`) under the current
  generation. A trust commit invalidates bound streams: dispatch,
  sending, and mutation recheck currency and refuse stale bindings.
- Frames refusals exactly once per the contract, then closes: identity
  `host_identity_mismatch`, allowlist `peer_not_allowlisted` (both exit
  7), contract `incompatible_protocol` (exit 6, only when actually
  known). Unframeable input and foreign RPC majors close without a
  frame. Before TLS completes, only standard TLS alerts move, never
  JSON. Initiator-local failures map to `authentication_failed` (7),
  `transport_failure` (8), and `invalid_config` (3).

## What it does not do

- No `ax` command, no SSH process launch, no listener: `LaunchArgv` and
  `LaunchForConfig` only *compute* the specified remote argv
  (`ax rpc serve --stdio --host-channel 1.0.0`) under Configuration 4.
  Real process/SSH wiring and CLI exit mapping are later leaves.
- No operation semantics: bodies stay opaque; unknown operations and
  handler errors frame `incompatible_protocol`. Merkle verification,
  object exchange, negotiation, and schema membership belong elsewhere.
- No durable state: a channel run performs no file writes (static and
  runtime censuses prove it). Trust commits are the caller's.
- No clock skew, no retries, no fallback majors on the channel.

## Entries

- `Serve(stream, config)` runs the responder core; `Dial(stream,
  config)` opens the initiator side; `(*Client).Call(op, body)` sends
  one authorized post-hello request; `WatchGeneration` reports a commit
  past the bound generation.
- `ClientTLS`/`ServerTLS` build the profile; `VerifyPeer` is the
  supplement; `EnrollmentPool` builds roots plus the CA-list bound;
  `RunHandshake` enforces the deadline and byte cap by closing the
  stream; `NewRequestID` mints UUIDv7 correlators; `ServerName` derives
  destination SANs; `LocalFailure` maps errors to initiator codes/exits.
- Timeouts: 10 s handshake, 10 s hello, configured RPC timeout (default
  300 s per 6.2). Zero selects the specified defaults; tests override.
  Incoming handshake bytes cap at 1 MiB; the encoded CA DN list must fit
  65,535 bytes or activation refuses.

## Verification

`go test ./internal/hostchannel -count=1` (race-clean): full
handshake/hello/operation positives in both roles over real TLS with
hosttrust-issued credentials, the ten HC-* gate families in
`gates_test.go`, refusal tests that assert the exact Error 1.3.0
class/exit plus close behavior, deadline/cap/generation/staleness
vectors, static and runtime write censuses, and launch/config model
checks. The hostile-network conformance suite (`hostile_test.go`,
`hostile_carrier_test.go`, acceptance case `AC-HOST-001`) re-derives
the seven story cases through `Dial`/`Serve`/`Call` with per-vector
refusal classes and state censuses, and executes the real OpenSSH
loopback carrier lane. `mutations.py` carries one narrowing mutant per
refusal gate (see `table.md` in its output), a neutral control, one
documented harmless SURVIVED control (client session cache without
server tickets), and a token-preserving census-evasion mutant killed by
the runtime census while the static census passes.

## Original acceptance rows, re-derived through this consumer

| Row | Driven through | Test |
| --- | --- | --- |
| Request/response correlation | `Dial` + `Call` + `Serve` | `TestFullHandshakeHelloAndOperation`, `TestHelloResponseCorrelation` |
| Hello maps and nonce echo | `Dial` + `Serve` hello exchange | `TestFullHandshakeHelloAndOperation`, `TestRefusesInvalidHello/*` |
| Inventory namespace/cardinality | `Call("inventory.roots")` dispatch | `TestInventoryRootsThroughChannel` |
| Offered limits (line/object floors) | hello validation both roles | `TestFullHandshakeHelloAndOperation`, `TestOversizeHelloLine` |
| Historical Structured Error framing | refusal frames via `Serve`, mapping via `Dial` | `TestRefusesNonHelloBeforeHello`, `TestHelloFailureMapping/*` |
| Bilateral admission before dispatch | `AuthorizeDispatch` at hello both roles | `TestRefusesHelloUUIDMismatch`, `TestRefusesServerHelloUUIDMismatch`, `TestRefusesNotAllowlisted`, `TestRefusesDestinationMismatch` |
| Deadlines and generation recovery | bounded reads/writes, stale gates, watch | `TestHelloDeadline/*`, `TestDispatchTimeout`, `TestStaleGenerationAtDispatch/*`, `TestStaleGenerationAtMutation`, `TestWatchGeneration` |

Stated bounds instead of tests: native Tailscale SSH and non-host
platform lanes (no SSH-implementation branch exists; the same profile
runs over any `Stream`, and the OpenSSH lane is executed by
`TestHostileRealCarrierOpenSSH`), the one-second self-close of an idle
stream with zero traffic (measured: the next dispatch fences and
`WatchGeneration` reports the commit; `serveLoop` carries no background
watchdog — see `TestHostileRevocationLivePeer/idle_observation`), and
TLS alert text (owned by crypto/tls). Post-hello re-hello is plain
dispatch, pinned by `TestRehelloIsPlainDispatch` and
`TestHostileReplay/rehello_is_inert_redispatch`.
