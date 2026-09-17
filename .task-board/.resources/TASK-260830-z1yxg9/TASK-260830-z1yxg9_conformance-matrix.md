# TASK-260830-z1yxg9 conformance matrix (Section 11.10.4 HC-* gates)

Authority: agent-session-manager-spec@v0.6.0, Section 11.10. Every vector
runs through the production entries (`hostchannel.Serve`/`Dial`/`Call`,
`VerifyPeer`, `EnrollmentPool`, `RunHandshake`, `NewHandshakeLimiter`,
`AuthorityListOverflow`, `LaunchForConfig`, `WatchGeneration`, plus the
`rpcwire` codec). Registry executable in `internal/hostchannel/gates_test.go`;
mutants in `internal/hostchannel/mutations.py` (full suite per probe).

## HC-TLS — TLS profile

| Vector | Test | Mutant |
| --- | --- | --- |
| Full TLS 1.3 + alpn + no-resume handshake, both roles | TestFullHandshakeHelloAndOperation (+) | — |
| TLS 1.2 offer refused, both directions | TestRefusesTLS12Offer/server, /client (−) | tls-min, tls-min-client (both killed) |
| Wrong ALPN refused, both directions | TestRefusesWrongALPN/server, /client (−) | — (protocol-level; supplement unit below) |
| Missing ALPN refused (config cannot) | TestRefusesMissingALPN (−), TestVerifyPeerRefusals/missing-alpn (−) | alpn-supplement (killed) |
| Wrong ALPN at supplement | TestVerifyPeerRefusals/alpn (−) | alpn-supplement (killed) |
| Resumption refused across two sessions | TestRefusesResumption (−) | tickets (killed) |
| Resumed state refused by supplement | TestVerifyPeerRefusals/resumed (−) | didresume (killed) |
| Plaintext preface: alert-or-close, never JSON | TestRefusesPlaintextPreface (−) | — |
| Silent peer hits the 10 s bound | TestSilentPeerHandshakeDeadline (−) | — |
| Handshake byte cap exact boundary + disarm | TestHandshakeLimiterBounds (−/+) | handshake-cap (killed) |
| Hostile intake bounded end to end | TestHandshakeIntakeIsBounded (−) | — |
| Client cache without server tickets: harmless | — (bound) | client-cache (SURVIVED, one-sided weakening cannot resume) |

## HC-CERT — enrolled credentials

| Vector | Test | Mutant |
| --- | --- | --- |
| Enrolled peer admitted | TestVerifyPeerAdmitsEnrolled (+), TestFullHandshakeHelloAndOperation (+) | — |
| Unenrolled root refused | TestRefusesUnenrolledRoot (−) | — |
| Revoked leaf refused | TestRefusesRevokedLeaf (−), TestVerifyPeerRefusals/revoked (−) | state-revoked (killed) |
| Retired past retire_at refused | TestRefusesRetiredPastRetireAt (−), TestVerifyPeerRefusals/retiring-past (−) | state-retiring (killed) |
| Retiring without bound refused | TestVerifyPeerRefusals/retiring-nil (−) | — |
| Expired leaf refused | TestRefusesExpiredLeaf (−), TestVerifyPeerRefusals/expired (−) | profile (killed) |
| Empty chains / empty certs refused | TestVerifyPeerRefusals/no-chains, /no-certificates (−) | chains (killed) |
| Unknown leaf refused | TestVerifyPeerRefusals/unknown-leaf (−) | — |
| Pool excludes revoked; overflow refuses activation | TestEnrollmentPoolExcludesRevoked (+), TestEnrollmentPoolRefusesOverflow (−) | ca-bound (killed) |
| Leaf/key credential construction + refusals | TestCredentialFromIssued (+), TestCredentialFromIssuedRefusals (−) | — |

## HC-MAP — exact match and naming

| Vector | Test | Mutant |
| --- | --- | --- |
| ServerName equals the issued leaf SAN | TestServerNameMatchesIssuedLeaf (+) | known-bad (killed) |
| Double match refused | TestVerifyPeerRefusals/double-match (−) | match-count (killed) |
| Leaf differing in one byte refused | TestVerifyPeerRefusals/leaf-flipped (−) | match-leaf (killed) |
| Root differing in one byte refused | TestVerifyPeerRefusals/root-flipped (−) | match-root (killed) |
| Wrong SPKI refused | TestVerifyPeerRefusals/spki-zero (−) | match-spki (killed) |
| Wrong destination SAN refused at TLS | TestRefusesWrongServerName (−), TestRefusesDestinationMismatch (−) | — |

## HC-HELLO — authorized hello

| Vector | Test | Mutant |
| --- | --- | --- |
| Full hello + operation, both roles | TestFullHandshakeHelloAndOperation (+), TestInventoryRootsThroughChannel (+) | — |
| Fresh UUIDv7 correlators; local code/exit map | TestNewRequestID (+), TestLocalFailureMapping (+) | — |
| Wrong hello ID refused, both roles (exact frame/exit) | TestRefusesHelloUUIDMismatch (−), TestRefusesServerHelloUUIDMismatch (−) | hello-id-server, hello-id-client (both killed) |
| Destination mismatch refused | TestRefusesDestinationMismatch (−) | — |
| Not allowlisted refused (exact frame/exit) | TestRefusesNotAllowlisted (−) | — |
| Invalid hello refused (exact frame/exit) | TestRefusesInvalidHello/contract, /missing-key, /nonce (−) | — |
| Hello failure frames mapped by initiator | TestHelloFailureMapping/host_identity_mismatch, /peer_not_allowlisted, /incompatible_protocol (−) | — |
| Hello reply correlation enforced | TestHelloResponseCorrelation (−) | — |
| Hello read/write deadlines | TestHelloDeadline/server, /client, TestHelloWriteDeadline (−) | — |

## HC-DISPATCH — post-hello dispatch

| Vector | Test | Mutant |
| --- | --- | --- |
| Authorized call + framed handler error + continue | TestFullHandshakeHelloAndOperation (+), TestHandlerErrorFramedAndContinues (+) | — |
| Non-hello before hello: one frame + close | TestRefusesNonHelloBeforeHello (−) | non-hello (killed) |
| Foreign majors close silently | TestForeignMajorClosesSilently/2.0.0, /4.0.0 (−) | — |
| Unframeable + oversize close silently | TestUnframeableHelloClosesSilently (−), TestOversizeHelloLine (−) | — |
| Dispatch timeout + mid-stream expiry | TestDispatchTimeout (−), TestDispatchExpiryClosesWithoutFrame (−) | — |
| Invalid call bodies + handler garbage | TestCallRefusesInvalidBody (−), TestHandlerGarbageBodyCloses (−) | — |
| Re-hello is plain dispatch | TestRehelloIsPlainDispatch (+) | — |

## HC-MIGRATE — Config 4 selection

| Vector | Test | Mutant |
| --- | --- | --- |
| Exact remote argv under Configuration 4 | TestLaunchForConfig (+), TestServeArgvPinsChannelVersion (+) | — |
| Legacy/unbound/unknown refused | TestLaunchForConfigRefusesLegacy (−), TestLaunchForConfigEmptySnapshot (−), unknown-selector subcase (−) | — |
| Activation refusals | TestInvalidActivation (−), TestLocalCredentialRefusals (−) | — |

## HC-LIFECYCLE — credential lifecycle on the channel

| Vector | Test | Mutant |
| --- | --- | --- |
| Retiring within bound admitted; fresh boundary runs | TestRetiringCredentialAdmitsWithinBound (+), TestMutationBoundaryRunsFresh (+) | — |
| Revoked / retired-past / expired refused | TestRefusesRevokedLeaf, TestRefusesRetiredPastRetireAt, TestRefusesExpiredLeaf (−) | state-revoked, state-retiring, profile (killed) |

## HC-GENERATION — authorization currency

| Vector | Test | Mutant |
| --- | --- | --- |
| Bound stream runs at the committed generation | TestMutationBoundaryRunsFresh (+), TestFullHandshakeHelloAndOperation (+) | — |
| Authority list bound exact | TestAuthorityListOverflow (+) | ca-bound (killed) |
| Stale dispatch: server closes, client never sends | TestStaleGenerationAtDispatch/server_closes, /client_refuses_to_send (−) | stale-dispatch (killed) |
| Stale mutation boundary never runs | TestStaleGenerationAtMutation (−) | stale-mutation (killed) |
| Generation watch reports commits | TestWatchGeneration (−/+) | — |

## HC-EXCLUDE — no durable writes, no leakage

| Vector | Test | Mutant |
| --- | --- | --- |
| Static write/exec/listen/system-pool census | TestStaticWriteCensus (+) | census-evasion (passes static by design) |
| Runtime census: state trees + trust bytes + temp tripwire | TestRuntimeWriteCensus (+) | census-evasion (killed behaviorally) |
| Binding redaction | TestBindingRedaction (+) | — |

## HC-PARITY — carrier independence

Stated bound, not a test: the package takes an abstract ordered binary
`Stream` and contains no SSH-implementation branch; the same TLS profile,
supplement, and refusal table run over OpenSSH and native Tailscale SSH
carriers. Positive evidence is every handshake test running over the
transport-agnostic `Stream` interface.
