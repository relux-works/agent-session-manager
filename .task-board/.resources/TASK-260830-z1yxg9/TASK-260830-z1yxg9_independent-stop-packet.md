# TASK-260830-z1yxg9 — staged RPC implementation; unchanged security stop

Requested board status: blocked. This is partial working-tree progress toward
the complete original deliverable, not acceptance or a ready-for-review handoff.

## Preserved state and implementation

Base: signed c1eff016dce2e55c4e2c1828d5c20f3da84118bc, accepted tree
f1dbf8f2dcd80ec8fdee9c0e7e1822cd921d41ac. Identity checkpoint 43c0e2b9 and
transport checkpoint c1eff016 both verify, exit 0. HEAD..main was 0 at initial
inspection; no current remote-main equality is claimed. All task changes remain
uncommitted in the managed STORY-260830-1kiyj6 worktree. No prerequisite source
was changed. The task-scoped patch and candidate SHA-256 manifest preserve the
reviewable delta outside that worktree.

Added internal/rpcwire: exact request/success/failure envelopes, UUIDv7
correlation and containing-version checks, structural hello identity/nonce/maps,
limit floors/minima, v2/v3/v4 profile fixtures, inventory.roots 6/7/8 vocabulary
and request/result cardinality, historical Structured Error binding, and pure
bootstrap-rejection framing. Its public codec is reusable production code;
there is no parallel test-only implementation. Received values stay untrusted.
README, UNRESOLVED_QUESTIONS.md, package traceability, ownership gap prose/digest and the existing bounded
fuzz validation list are updated. Registry coverage remains 17/428 clauses.

## AC accounting

**0 of 7 full AC rows driven through an authenticated RPC runtime consumer.**
Five rows have new public-component evidence; that does not discharge their
end-to-end requirements. These tests are in the preserved uncommitted delta,
not committed tests or an accepted checkpoint.

| Original AC row | Production calls actually driven / named tests | Remaining bound |
| --- | --- | --- |
| Bilateral host admission | No new authentication consumer; existing peeridentity full-suite tests rerun | Independent inbound association remains unresolved; no admission result |
| Request/response correlation | rpcwire.DecodeRequest/DecodeResponse/EncodeRequest/EncodeSuccess; TestPinnedHelloProfiles, TestRequestRefusals, TestResponseRefusals | One expected request only; no live pending queue/replay state |
| Hello order, maps, nonce | rpcwire.DecodeRequest/DecodeResponse/Request.Hello; TestHelloRefusals, TestPinnedHelloProfiles, TestLimitsAndUntrustedIsolation | Maps/echo/shape proven; first-frame state, bilateral success and received freshness unproven |
| Namespace/cardinality | rpcwire.DecodeRequest/EncodeSuccess/DecodeResponse; TestInventoryNamespacesAndCardinality | inventory.roots only; no object-schema membership, Merkle verification or dispatch |
| Limits and partial I/O | rpcwire.DecodeRequest/EncodeRequest/OfferedLimits; TestFrameRefusals, TestWriterAndAccessorBounds, TestLimitsAndUntrustedIsolation; existing SSH Session.Send/Receive suite and narrowing rerun | Offered minima only; no authenticated negotiated object exchange |
| Deadlines and recovery | Existing sshtransport.Client.Open/Session.supervise; TestCancellationAndCleanup/deadline and selected rpc-timeout/cancel mutants rerun | Existing configured timeout/context ownership; no new hello lifecycle or durable mutation recovery |
| Structured errors/historical framing | rpcwire.EncodeFailure/DecodeResponse/EncodeRejection -> axerror.DecodeBound; TestHistoricalFailureBindings, TestBootstrapRejectionFraming | Bytes only; responder choosing rejection and sending one frame then closing remains unimplemented |

No ax executable or doctor/capability availability is added. Other operation
bodies are retained as opaque JSON, without operation validation or permission.
No durable state is mutated by this package. Major/minor selection remains
TASK-260830-219okr; all hostile-peer cases remain TASK-260830-2x16gz obligations.

## Exact unchanged decision / Stop-The-Line

Section 11.1 requires both sides to compare protocol host_id to the configured
allowlist entry after external SSH authentication. The outbound transport owns
an independently selected Target. The fixed inbound ax rpc serve --stdio,
current config schema and transport expose no production association selecting
one expected inbound peer independently of the received host_id.

The failed assumption remains Resolve(hello.host_id).CheckProtocolHost(hello.host_id):
it proves only allowlist membership. Nonces, UUID correlation, structural maps,
limits and codec success do not repair that association. The prior membership-
only interpretation remains WITHDRAWN. No substitute supplier, verified boolean,
implicit trust, configurable bypass or new SSH mapping was implemented.

Primary-owned alternatives remain:
1. Recommended for the unchanged cross-listed-host spoofing requirement: approve
   and route an external SSH identity/invocation-to-configured-host association
   contract, including authority, provisioning, bypass resistance and supported
   transports. This retains external key authentication but requires a contract
   beyond the currently specified invocation/config inputs.
2. Explicitly accept membership-only inbound admission, with the bound that an
   authenticated SSH user can claim any listed host. This weakens identity
   assurance and must be reconciled with the original hostile-peer requirement;
   no answer or approval is inferred here.

Exact input still needed: how the real responder independently selects its
configured expected peer, or an explicit human security decision accepting the
membership-only bound. Primary owns that decision and continuation routing.
The codec work can compose with either eventual approved association; remaining
consumer code must wait rather than fabricate the trust boundary.

## Commands and actual exits

| Command | Exit | Evidence / result |
| --- | ---: | --- |
| `go test ./internal/rpcwire -count=1 -v` | 0 | `rpc-tests-01.log`: Initial component suite |
| `go test ./internal/rpcwire -count=1 -v` | 0 | `rpc-tests-02.log`: Added bootstrap and writer bounds |
| `python3 internal/rpcwire/mutations.py --output .temp/TASK-260830-z1yxg9/mutations` | 1 | `mutations-01.log`: One survivor; recorded below; not passing |
| `go test ./... -v -count=1` | 1 | `full-tests-01.log`: New fuzz target missing required validation command; fixed |
| `go build ./...` | 0 | `build-01.log`: Before final refinement |
| `go vet ./...` | 0 | `vet-01.log`: Before final refinement |
| `go test ./internal/rpcwire -race -count=1 -cover` | 0 | `rpc-race-01.log`: 97.7% before final refinement |
| `go run ./internal/traceability/cmd/tracecheck` | 1 | `tracecheck-repin-01.log`: Expected red: changed gap text invalidated old ownership digest; no green claim |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `tracecheck-01.log`: Gap-text digest updated; 17/428 clauses unchanged |
| `python3 internal/rpcwire/mutations.py --output .temp/TASK-260830-z1yxg9/mutations-v2` | 0 | `mutations-02.log`: 32/32 narrowing witnesses killed; neutral and known-bad controls valid |
| `go test ./... -v -count=1` | 0 | `full-tests-02.log`: Resulting partial delta, full repository |
| `go test ./... -cover -count=1` | 0 | `full-coverage-01.log`: Resulting partial delta, full repository |
| `go test ./internal/rpcwire -run=^$ -fuzz=^FuzzUntrustedEnvelopes$ -fuzztime=100x -parallel=1` | 0 | `fuzz-01.log`: 100 executions |
| `go vet ./...` | 0 | `vet-02.log`: Resulting partial delta |
| `go build ./...` | 0 | `build-02.log`: Resulting partial delta |
| `GOOS=windows go vet ./...` | 0 | `windows-vet-01.log`: Type-check/vet only; no Windows runtime claim |
| `go test ./internal/rpcwire -race -count=1 -cover` | 0 | `rpc-race-02.log`: 97.8% component coverage |
| `python3 internal/sshtransport/mutations.py --only neutral,known-bad,send-plus-one,read-plus-one,read-partial,rpc-timeout,cancel-as-success --output .temp/TASK-260830-z1yxg9/transport-mutants` | 0 | `transport-mutants-01.log`: Five predecessor narrowing probes killed; both controls valid |
| `git diff --check` | 0 | `diff-check-01.log`: No whitespace errors |
| `gofmt -l internal/rpcwire internal/traceability/traceability.go` | 0 | `gofmt-01.log`: Output empty; separate Python assertion exit 0 |
| `git verify-commit 43c0e2b9` | 0 | `signature-identity-01.log`: Accepted prerequisite signature |
| `git verify-commit c1eff016` | 0 | `signature-transport-01.log`: Accepted prerequisite signature |

## Narrowing and instrument controls

Current RPC run: **32 of 32 narrowing witnesses killed**, plus a passing neutral
control and killed known-bad control. Predecessor subset: **5 of 5 narrowing
witnesses killed**, plus both controls. All kills have named behavioral failures
and compiled overlays. Original source bytes were verified unchanged by each
probe. The common-data-model probe preserves the Canonicalize token/call and
weakens only its refusal; it runs the behavioral suite. No source-only gate was
introduced. These ratios describe the selected mutants, not all possible faults.

Earlier run survivor (preserved, not relabeled a kill):

| Mutant | Gate weakened to admit | Named failing test | Exit | Surviving bound |
| --- | --- | --- | ---: | --- |
| correlation-version, run 1 | RPC3 response to an RPC2 expectation | none | 0 | Secondary revalidation of the request under RPC3 still rejected the frame; no independent mismatch-gate kill in run 1. Corrected dependency and killed by TestResponseRefusals/version in run 2. |

Current RPC mutant table:

| Mutant | What the gate is narrowed to admit | Named failing test | Exit | Result / surviving bound |
| --- | --- | --- | ---: | --- |
| neutral | Equivalent major value | none | 0 | neutral passed:  |
| known-bad | Wrong request accessor | TestPinnedHelloProfiles/2.0.0 | 1 | killed:  |
| line-plus-one | Admits exactly 8388609 bytes | TestFrameRefusals/oversize | 1 | killed:  |
| lf | Admits final LF in a line payload | TestFrameRefusals/newline | 1 | killed:  |
| protocol | Admits urn:other only | TestRequestRefusals/protocol | 1 | killed:  |
| request-uuid | Admits one UUIDv4 request ID | TestRequestRefusals/uuid | 1 | killed:  |
| closed-extra | Admits one extra=true member | TestRequestRefusals/extra | 1 | killed:  |
| correlation-id | Admits one mismatched request ID | TestResponseRefusals/id | 1 | killed:  |
| correlation-version | Admits mismatched response version 3 only | TestResponseRefusals/version | 1 | killed:  |
| nonce-echo | Admits the wrong echo fixture only | TestResponseRefusals/nonce-echo | 1 | killed:  |
| failure-binding | Admits Error 1.0 in RPC3 while retaining other bindings | TestHistoricalFailureBindings/3.0.0 | 1 | killed:  |
| rejection-shape | Admits request-shaped object without operation | TestBootstrapRejectionFraming/response | 1 | killed:  |
| hello-host | Admits one malformed host ID | TestHelloRefusals/host | 1 | killed:  |
| hello-platform | Admits ios only | TestHelloRefusals/platform | 1 | killed:  |
| hello-version | Admits one leading-zero version | TestHelloRefusals/ax-version | 1 | killed:  |
| nonce-short | Admits exactly 120 nonce bits | TestHelloRefusals/nonce-short | 1 | killed:  |
| line-floor | Admits line floor minus one | TestHelloRefusals/line-floor | 1 | killed:  |
| object-floor | Admits object floor minus one | TestHelloRefusals/object-floor | 1 | killed:  |
| contract-extra | Admits one forbidden error map key | TestHelloRefusals/error-key | 1 | killed:  |
| contract-seventeen | Admits exactly 17 versions | TestHelloRefusals/seventeen-versions | 1 | killed:  |
| contract-empty | Admits present empty version array | TestHelloRefusals/empty-versions | 1 | killed:  |
| contract-duplicate | Admits duplicate adjacent versions | TestHelloRefusals/duplicate-versions | 1 | killed:  |
| contract-unsorted | Admits one descending adjacent pair | TestHelloRefusals/unsorted-versions | 1 | killed:  |
| contract-semver | Admits v1.0.0 version string only | TestHelloRefusals/invalid-version | 1 | killed:  |
| v3-exact | Admits nonexact v3 lease array | TestHelloRefusals/exact-map-3.0.0 | 1 | killed:  |
| v4-exact | Admits nonexact v4 lease array | TestHelloRefusals/exact-map-4.0.0 | 1 | killed:  |
| offered-pair | Admits one cross-request limit pair | TestLimitsAndUntrustedIsolation | 1 | killed:  |
| namespace-empty | Admits nonnil empty namespace array | TestInventoryNamespacesAndCardinality/2.0.0 | 1 | killed:  |
| namespace-member | Admits credential namespace only | TestInventoryNamespacesAndCardinality/2.0.0 | 1 | killed:  |
| namespace-duplicate | Admits duplicate namespaces | TestInventoryNamespacesAndCardinality/2.0.0 | 1 | killed:  |
| namespace-sort | Admits event/blob descending pair | TestInventoryNamespacesAndCardinality/2.0.0 | 1 | killed:  |
| roots-missing | Admits one missing requested root | TestInventoryNamespacesAndCardinality/2.0.0/missing | 1 | killed:  |
| root-association | Admits event root in blob slot | TestInventoryNamespacesAndCardinality/2.0.0 | 1 | killed:  |
| common-data-model | Admits the duplicate protocol fixture while retaining the canonicalization call | TestFrameRefusals/duplicate | 1 | killed:  |

Selected predecessor transport table:

| Mutant | What the gate is narrowed to admit | Named failing test | Exit | Result / surviving bound |
| --- | --- | --- | ---: | --- |
| neutral | Equivalent size accumulator | none | 0 | neutral passed:  |
| known-bad | Adds a remote command token | TestOpenStructuredCommand | 1 | killed:  |
| send-plus-one | Admits exactly one excess outbound byte | TestDuplexAndSendBoundaries/oversize | 1 | killed:  |
| read-plus-one | Admits exactly one excess inbound byte | TestReceiveFailureAndLimits/oversize | 1 | killed:  |
| read-partial | Treats one unterminated success-shaped frame as EOF | TestReceiveFailureAndLimits/partial | 1 | killed:  |
| cancel-as-success | Suppresses explicit cancellation but keeps other failures | TestCancellationAndCleanup/stall | 1 | killed:  |
| rpc-timeout | Allows one extra second of process lifetime | TestCancellationAndCleanup/deadline | 1 | killed:  |

## Boundaries and retained evidence

Full local tests, full coverage, relevant build/lint, component race and bounded
fuzz all ran in this run. Earlier full-suite failure and expected-red ownership
pin check retain exit 1. Full repository race, all older fuzz budgets, hosted CI,
real SSH key/replay tests and other OS runtime lanes were not run. Windows vet
is not runtime evidence. No Change Request, developer handoff, checkpoint,
integration, landing or completed-task status was invoked. Code, tests and logs
are preserved; the complete original task and its whole-deliverable checklist
remain open pending the exact decision above.
