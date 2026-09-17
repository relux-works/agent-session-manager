# TASK-260830-2x16gz conformance matrix — hostile-network fail-closed + AC-HOST-001

Authority: `relux-works/agent-session-manager-spec@v0.6.0` (`0cbdf100`),
sections 6.6, 11.10–11.10.5, 16.1. Every row is driven through a production
entry point by a named committed test, or is declared a stated bound below.
`n of m` ratios are counted over the rows in this matrix, not prose.

Production entries under test: `hostchannel.Dial` (`client.go`), `hostchannel.Serve`
(`server.go`), `(*hostchannel.Client).Call` (`client.go`), `hosttrust` enrollment /
rotation / revocation / generation (`enroll.go`, `rotate.go`, `authorize.go`,
`transact.go`), Config-4 selection (`config` loader + `hostchannel.LaunchForConfig`),
and the SSH transport boundary (real OpenSSH loopback carrier in
`TestHostileRealCarrierOpenSSH`).

## A. Seven hostile-network cases — 7 of 7 driven

| # | Case | Production call site | Named test(s) | Refusal class pinned |
|---|---|---|---|---|
| 1 | Unknown peers | `Serve` + `Dial` | `TestHostileUnknownPeerBothRoles/responder_refuses`, `.../initiator_refuses`; prior `TestRefusesUnenrolledRoot`, `TestVerifyPeerRefusals/unknown-leaf` | `authentication_failed`/7 both roles |
| 2 | Key changes | `Serve` + `Dial`; `Store.Rotate`, `Store.Enroll` | `TestHostileKeyChangeSameUUID/responder_refuses_reissued`, `.../initiator_refuses_reissued`, `.../rotation_admits_new_key_only_after_enrollment`, `.../no_second_rotation_before_revocation` | `authentication_failed`/7 pre-enrollment; `invalid_config`/3 for retired local activation; second rotation `ErrRotationRefused` with generation pinned |
| 3 | Spoofed host IDs | `Serve` + `Dial` via `Store.AuthorizeDispatch` | `TestHostileSpoofedHostID/hello_claims_destination`, `.../valid_but_unexpected_peer`; prior `TestRefusesHelloUUIDMismatch`, `TestRefusesServerHelloUUIDMismatch`, `TestRefusesDestinationMismatch` | `host_identity_mismatch`/7 (hello vs cert), `authentication_failed`/7 (valid-but-unexpected peer) |
| 4 | Disconnects at every phase boundary | `Serve` + `Dial` + `Call` | `TestHostileDisconnectPhases/mid_handshake_close`, `.../after_handshake_before_hello`, `.../hello_half_frame_close`, `.../mid_request_close`, `.../mid_response_close`, `.../server_aborts_hello_reply`, `.../server_aborts_response`; prior `TestSilentPeerHandshakeDeadline` | `transport_failure`/8 or `unframeable→transport_failure`/8 per phase; handler never runs on truncated phases; census clean |
| 5 | Replay | `Serve` + `Dial` + `Call`; `rpcwire.NewNonce` | `TestHostileReplay/tls_flight_replay_refused` (TLS 1.3 refuses), `.../encrypted_hello_replay_refused`, `.../rehello_is_inert_redispatch` (binding pinned), `TestHostileReplayNonceFreshness` | `authentication_failed`/7 or `transport_failure`/8 for cross-session replay; re-hello is inert re-dispatch (`incompatible_protocol`/6 from handler, binding unchanged, channel continues) |
| 6 | Oversized frames | `Serve` framing + `rpcwire` decode | `TestHostileOversizedFrames/hello_line_8mib_plus_one` (well-formed hello over cap), `.../dispatch_line_8mib_plus_one` (well-formed call at cap+1), `.../hello_below_advertised_floor/{object,line}`, `.../handshake_intake_bound_documented`; prior `TestHandshakeLimiterBounds`, `TestHandshakeIntakeIsBounded` | `unframeable→transport_failure`/8 for over-cap lines; `incompatible_protocol`/6 for below-floor offers; intake < 64 KiB documented (Go record framing fires first, 1 MiB limiter is the backstop) |
| 7 | Disclosure mismatches | `Serve` + `Dial` via `rpcwire` hello decode + `axerror.DecodeBound` | `TestHostileDisclosureMismatch/contracts_rpc_superset`, `.../contracts_rpc_missing`, `.../responder_contracts_drift`, `.../error_version_drift` (1.2.0 label on a 1.3.0-shaped body refused before the code is honored); prior `TestRefusesInvalidHello/*` | `incompatible_protocol`/6 all four vectors |

Coverage: **7 of 7** hostile rows driven through production entries. Every
vector additionally asserts no handler effect on refused paths and a
before/after state census (or the exact revocation-transition census).

## B. AC-HOST-001 clause rows — 14 of 15 fully driven, 1 partial with measured bound

Gate text: "Drive the real Config-4 launch and RPC-5 dispatch on supported
SSH/platform lanes: full mutual TLS, exact enrolled certificate/UUID/hello, no
preauthentication effects, migration/downgrade refusals, fresh-key rotation,
revocation of idle/live streams within one second, expiry and concurrent
mutation-generation fencing. Exercise valid and invalid actual certificates,
both role failures, stale/failed reads and recovery bypasses."

| # | Clause | Production call site | Named test(s) | Result |
|---|---|---|---|---|
| B1 | Real Config-4 launch | `config.Load` + `LaunchForConfig`; `ServeArgv` | `TestHostileRealCarrierOpenSSH` (exact `ServeArgv` on the wire, observed server-side), `TestLaunchForConfig`, `TestServeArgvPinsChannelVersion` | Driven (OpenSSH lane; `ax` binary itself does not exist — forced command execs `Serve` directly) |
| B2 | RPC-5 dispatch on supported SSH/platform lanes | `Dial`/`Serve`/`Call` over `Stream` and over ssh stdio | `TestHostileRealCarrierOpenSSH`, `TestFullHandshakeHelloAndOperation`, `TestHostileRoleSwap` | Driven on OpenSSH loopback + pipe lanes; Tailscale + non-host lanes are stated bounds |
| B3 | Full mutual TLS | `ClientTLS`/`ServerTLS` + `VerifyPeer` supplement | `TestFullHandshakeHelloAndOperation` (1.3 + ALPN + no-resume asserted), `TestRefusesTLS12Offer/*`, `TestRefusesWrongALPN/*`, `TestRefusesMissingALPN`, `TestRefusesResumption`, `TestVerifyPeerRefusals/alpn-same-family` (new) | Driven |
| B4 | Exact enrolled certificate/UUID/hello | `VerifyPeer` + `AuthorizeDispatch` | Case 1–3 vectors above; `TestVerifyPeerAdmitsEnrolled`, `TestVerifyPeerRefusals/*` | Driven |
| B5 | No preauthentication effects | `Serve`/`Dial` sequencing | `TestRefusesPlaintextPreface`, `TestRefusesNonHelloBeforeHello`, `TestHostileDisconnectPhases/*` (handler-count 0), censuses | Driven |
| B6 | Migration/downgrade refusals | `config.Load` (required closed `mesh.host_channel`), `LaunchForConfig`, `Serve` major gate | `TestHostileV4WithoutHostChannelBinding` (loader refusal), `TestLaunchForConfigRefusesLegacy`, `TestForeignMajorClosesSilently/2.0.0+4.0.0`, `TestRefusesPlaintextPreface` | Driven |
| B7 | Fresh-key rotation | `Store.Rotate` + `Store.Enroll` + channel admission | `TestHostileKeyChangeSameUUID/rotation_admits_new_key_only_after_enrollment`, `.../no_second_rotation_before_revocation`, `TestRetiringCredentialAdmitsWithinBound` | Driven |
| B8 | Revocation of idle/live streams within one second | `Store.Revoke` + dispatch fence + `WatchGeneration` | `TestHostileRevocationLivePeer/next_dispatch_refused_fast` (revoke-to-fence < 1 s, logged), `.../watch_reports_commit_fast` (< 1 s, logged), `.../idle_observation` | **Partial**: dispatched/live work fences < 1 s; idle-with-zero-traffic self-close measured NOT satisfied (stated bound, log evidence) |
| B9 | Expiry fencing | `AuthorizeDispatch` time checks | `TestDispatchExpiryClosesWithoutFrame`, `TestRefusesExpiredLeaf`, `TestRefusesRetiredPastRetireAt` | Driven |
| B10 | Concurrent mutation-generation fencing | `WithMutationAuthorization` | `TestStaleGenerationAtMutation`, `TestStaleGenerationAtDispatch/*`, `TestHostileRecoveryBypass/stale_binding_never_rebinds`, `TestWatchGeneration` | Driven |
| B11 | Valid actual certificates | issuance + admission | `TestFullHandshakeHelloAndOperation`, `TestHostileRoleSwap`, `TestHostileConcurrentPeerConnections`, rotation positive half, copied-key pre-revocation half | Driven |
| B12 | Invalid actual certificates | `VerifyPeer` + pool | Cases 1–2 vectors; `TestRefusesRevokedLeaf`, `TestRefusesExpiredLeaf`, `TestCredentialFromIssuedRefusals`, `TestLocalCredentialRefusals/*` | Driven |
| B13 | Both role failures | `Dial` and `Serve` | Initiator: `TestHostileUnknownPeerBothRoles/initiator_refuses`, `TestHostileKeyChangeSameUUID/initiator_refuses_reissued`, `TestRefusesWrongServerName`, `TestRefusesServerHelloUUIDMismatch`, `TestHostileSpoofedHostID/valid_but_unexpected_peer`, `TestHostileDisclosureMismatch/responder_contracts_drift+error_version_drift`. Responder: case 1–3 responder halves + full refusal table | Driven |
| B14 | Stale/failed reads | `Store.ReadSnapshot` via `Dial`/`Serve` activation | `TestHostileStaleReadsAreIntegrityFailures/corrupt_store+unreadable_store+initiator_unreadable_store`, `TestInvalidActivation/missing_store` | Driven (`invalid_config`/3, never "absent") |
| B15 | Recovery bypasses | `Call` + `DecodeResponse` correlation | `TestHostileRecoveryBypass/call_after_close_refused`, `.../stale_binding_never_rebinds`, `.../response_id_mismatch_refused`, `TestHelloResponseCorrelation` | Driven |

Coverage: **14 of 15** AC-HOST-001 rows fully driven; B8 partial with a
measured stated bound (idle-without-traffic self-close). No row is N/A.

## C. HC-* gate families — 10 of 10 with positive + negative vectors

Executable registry in `internal/hostchannel/gates_test.go`
(`TestHCGateRegistryIsComplete`), extended by this task:

| Gate | New vectors (this task) | Prior vectors retained |
|---|---|---|
| HC-TLS | `TestHostileRealCarrierOpenSSH`, `TestHostileRoleSwap` (+), `TestVerifyPeerRefusals/alpn-same-family`, `TestHostileReplay/tls_flight_replay_refused+encrypted_hello_replay_refused`, `TestHostileReplayNonceFreshness`, `TestHostileDisconnectPhases/mid_handshake_close`, `TestHostileOversizedFrames/handshake_intake_bound_documented` (−) | TLS1.2/ALPN/missing-ALPN/resumption/preface/deadline/limiter families |
| HC-CERT | `TestHostileCopiedKey` (± halves), `TestHostileUnknownPeerBothRoles/*`, `TestHostileKeyChangeSameUUID/responder+initiator_refuses_reissued` (−) | unenrolled/revoked/retired/expired/pool families |
| HC-MAP | `TestHostileSpoofedHostID/valid_but_unexpected_peer` (−) | double-match/leaf-flipped/root-flipped/spki-zero/wrong-server-name |
| HC-HELLO | `TestHostileSpoofedHostID/hello_claims_destination`, `TestHostileDisclosureMismatch/*` (4), `TestHostileOversizedFrames/hello_below_advertised_floor`, `TestHostileDisconnectPhases/after_handshake_before_hello+hello_half_frame_close+server_aborts_hello_reply` (−) | hello-mismatch/destination/allowlist/contract/nonce/correlation/deadline families |
| HC-DISPATCH | `TestHostileConcurrentPeerConnections` (+), oversize hello+dispatch lines, mid-request/mid-response/abort rows, `TestHostileStallingPeerRPCTimeout`, `TestHostileReplay/rehello_is_inert_redispatch`, `TestHostileRecoveryBypass/call_after_close_refused+response_id_mismatch_refused` (−) | non-hello/foreign-major/unframeable/timeout/expiry/body families |
| HC-MIGRATE | `TestHostileV4WithoutHostChannelBinding` (−) | legacy/empty-snapshot/foreign-major/activation/credential families |
| HC-LIFECYCLE | rotation enrollment positive (+), `no_second_rotation_before_revocation` + `next_dispatch_refused_fast` (−) | retiring-admits/mutation-fresh positives; revoked/retired/expired negatives |
| HC-GENERATION | `stale_binding_never_rebinds`, `watch_reports_commit_fast`, stale-reads trio (−) | dispatch/mutation staleness + watch families |
| HC-EXCLUDE | every `TestHostile*` vector carries a state census or the revocation-transition census | static + runtime censuses, binding redaction, census-evasion mutant |
| HC-PARITY | `TestHostileRealCarrierOpenSSH` (+) executed | stated bound: native Tailscale SSH not executed |

Coverage: **10 of 10** families with positive and negative vectors.

## D. Stated bounds (recorded, never passed)

| Bound | Owner / evidence |
|---|---|
| Native Tailscale SSH not executed | This suite; `TestHostileRealCarrierOpenSSH` covers OpenSSH only; package carries no SSH-implementation branch (HC-PARITY by construction) |
| Platform lanes other than this host (darwin/arm64 here) | This suite; linux/windows builds compile (`GOOS=... go build`), no runtime lane claim |
| Section 11.10.3 one-second self-close of an idle stream with zero traffic | Measured NOT satisfied: `TestHostileRevocationLivePeer/idle_observation` log; next-dispatch fence + `WatchGeneration` measured < 1 s |
| Section 11.10.5 upstream policy vectors as source evidence only | No policy reader implemented (by design); unowned registry entry retained with fresh gap |
| No `ax` CLI / SSH process launch in product | `LaunchArgv`/`LaunchForConfig` compute argv only; carrier test's forced command execs `Serve` via the test binary; `SSH_ORIGINAL_COMMAND` proves the exact `ServeArgv` words crossed the wire |
| `LaunchForConfig` nil-binding branch unreachable via public entries | Config loader requires closed `mesh.host_channel` in v4; branch is defense-in-depth behind the loader gate (no mutant — unreachable, documented) |
| TLS alert text, handshake-failure races on `Dial` side | Owned by crypto/tls + rendezvous timing; class pinned (`transport_failure`), message not pinned |

## E. Narrowing-mutant summary — 9 of 9 new probes killed

Harness: `internal/hostchannel/mutations.py` (extended with multi-file
`(file, old, new)` plants for the dual-enforced line cap). Full behavioral
suite per plant; sources untouched (overlay). Pre-existing 26 probes re-run:
neutral passed, 24 killed, 1 documented harmless SURVIVED (`client-cache`).

| Mutant | Narrowing (admits exactly) | Named killing test | Result |
|---|---|---|---|
| `alpn-same-family` | `ax-host/2` ALPN | `TestVerifyPeerRefusals/alpn-same-family` | killed |
| `oversize-dispatch` | 8 MiB+10 line at framing AND decode | `TestHostileOversizedFrames/dispatch_line_8mib_plus_one` | killed |
| `stale-client-rebind` | one generation past the client binding | `TestHostileRecoveryBypass/stale_binding_never_rebinds` | killed |
| `rpc-contracts` | any `contracts.rpc` disclosure | `TestHostileDisclosureMismatch/responder_contracts_drift` | killed |
| `error-version-drift` | a 1.2.0 error honored as 1.3.0 | `TestHostileDisclosureMismatch/error_version_drift` | killed |
| `truncated-dispatch-ignored` | one truncated dispatch frame skipped instead of unframeable close | `TestHostileDisconnectPhases/mid_request_close` | killed |
| `snapshot-integrity` | corrupt store read as empty store | `TestHostileStaleReadsAreIntegrityFailures/corrupt_store` | killed |
| `correlation-health` | miscorrelated `health.get` answer accepted | `TestHostileRecoveryBypass/response_id_mismatch_refused` | killed |
| `nonce-static` | one constant hello nonce | `TestHostileReplayNonceFreshness` | killed |

Gates without a new mutant name their covering probe: supplement exact-match
(`match-leaf/match-root/match-spki/match-count`), trust-state
(`state-revoked/state-retiring`), hello identity (`hello-id-server/client`),
resumption (`tickets`), limiter (`handshake-cap`), CA bound (`ca-bound`),
server generation (`stale-dispatch`), mutation rebind (`stale-mutation`).
Replay refusal is TLS-1.3-intrinsic (stdlib owns the gate; ours are
no-tickets/no-cache/fresh-nonce, all mutated). No new source-text-inspecting
gate was added (the static census + its token-preserving evasion mutant are
prior work, re-run green).

## F. P3 notes from z1yxg9 — 2 of 2 closed

| Note | Resolution |
|---|---|
| Same-family `ax-host/2` ALPN refusal vector in `TestVerifyPeerRefusals` | Added `alpn-same-family` row + `alpn-same-family` mutant, killed |
| Shortened-deadline hook for `TestInvalidActivation/bad_destination` | Explicit 500 ms overrides + < 2 s elapsed assertion |
