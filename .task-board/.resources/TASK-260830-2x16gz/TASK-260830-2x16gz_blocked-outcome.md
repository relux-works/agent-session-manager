# TASK-260830-2x16gz — hostile-network conformance prerequisite

Disposition: blocked before product-code changes. This is an evidence-backed ownership/dependency decision for the primary owner, not a developer review handoff. The full seven-case acceptance criterion remains open. No CR was published and no task scope was silently removed.

## Exact accepted base

Both predecessor CR1 verdicts and outcomes were read, together with their checkpoint reports. The live managed workspace reports both CR1s checkpointed and this run (RUN-260907-2761f9) as its lease owner. Independently verified:

| Predecessor | Signed checkpoint | Accepted tree | Parent |
| --- | --- | --- | --- |
| TASK-260830-2u34k1 — implement-host-and-peer-identity | 43c0e2b9a33f0b08ee9a6a6f8740cc0a68dd8792 | c775903a1aa76cb3d433aeee89ebfd24e73159dd | 7654d7cadb2c226bfa3db5f23ac355a730285eea |
| TASK-260830-1tvg8e — implement-ssh-transport-and-command-boundary | c1eff016dce2e55c4e2c1828d5c20f3da84118bc | f1dbf8f2dcd80ec8fdee9c0e7e1822cd921d41ac | 43c0e2b9a33f0b08ee9a6a6f8740cc0a68dd8792 |

Both git verify-commit commands exited 0. Commit authors remain Ivan Oparin <oparin@me.com>. All measurements are at c1eff016, not an inferred current remote main. HEAD..local-main is 0; no remote-ref freshness claim. Final tracked diff, index diff and porcelain status are empty. No branch, index, signing, commit, refresh, integration or publication mutation was performed.

The embedded normative AX v0.5.0 document matches the pinned SHA-256 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a at upstream commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c. The document fidelity tests also passed. Relevant clauses: §6.3 (lines 2420–2501), §6.4 (2503–2573), §11.1 (7040–7056), §11.2 (7058–7171), §16.1 (11216–11230).

## Constraint and failed assumption

The failed assumption is that accepted configuration and SSH-process leaves provide the runtime enforcing every seven-case network consequence. They deliberately do not. They are real production API boundaries; this finding does not reject library APIs or require an ax executable merely for ceremony.

Section 11.1 requires both sides to verify the protocol host_id after SSH authentication. Target.CheckProtocolHost is a real, tested comparator, but its result has no production handshake/operation consumer. Source caller census finds the declaration, tests, and a transport comment only. Session.Receive returns raw bounded lines; it does not decode host IDs. Its documented earlier-frame semantics are provisional. A test that calls the comparator itself after reading a frame proves that API, not that production rejects an authenticated spoof before an operation.

The assignment explicitly reserves runtime hello, correlation and operation framing to TASK-260830-z1yxg9 — implement-rpc-envelope-and-hello, under STORY-260830-4qojoz — mesh-rpc-framing-and-negotiation. The live task is to-dev and its hard blockedBy set includes TASK-260830-2x16gz (along with TASK-260830-treeox and TASK-260830-33sfxc). Completing this leaf's full runtime spoof proof therefore requires its explicitly excluded downstream owner, while that owner waits on this leaf. This is a concrete ownership/dependency cycle; no missing tool or slow test caused it.

DisclosurePolicy is likewise a real configuration-policy boundary, not a payload authorization or authenticated publication operation. §6.4 configuration mismatch can be tested here; authenticated network disclosure requires its publisher. Existing README and .spec/README accurately state both distinctions. They were inspected, not rewritten to claim full conformance. internal/dirnode/query.go contains a disclosure_policy_digest field/parser, not a caller of peeridentity.DisclosurePolicy.

## Seven-case enforcement and evidence map

**3 of 7 AC rows driven at existing configuration/process/stream enforcement points in this run** (unknown selector, stream disconnect/error, oversized frame). **0 of 7 newly implemented conformance rows.** Two further rows have conditional comparator/configuration evidence only; two require actual SSH cryptographic fixtures not run here. This is not 7/7 hostile-network conformance and does not discharge the acceptance criterion.

All named tests below already belong to the signed predecessor checkpoints. This run adds no committed test and does not represent disposable or uncommitted probes as committed evidence.

| Required case | Normative layer and actual production call site | Named committed test rerun / positive neighbor | Refusal, recovery and exact bound |
| --- | --- | --- | --- |
| Unknown peers | §§6.3,16.1: sshtransport.New → Client.Open → Directory.Resolve before exec.Start | TestOpenRefusesBeforeStart; TestOpenStructuredCommand; TestResolveAllowlistAndAliasAmbiguity | Unknown discovery/endpoint selectors return ErrNotAllowlisted with no Session; valid configured targets start. Fresh sessions and read-error recovery are separate controls. Counts the local allowlist row; unknown SSH host-key behavior is not inferred. |
| Key changes | §§6.3,11.1,16.1: Client.Open → strict argv → external OpenSSH host-key verification | TestOpenNativeSSHPolicy; TestReceiveFailureAndLimits/exit255, with empty/max controls | Native -G proves policy resolution, and a fixture exit 255 proves ErrExit propagation. Neither proves changed-key rejection. Not counted. Real native SSH with synthetic old/new keys, accepted-key neighbor, rejection-before-remote-command and fresh-connection recovery remains required. |
| Spoofed host IDs | §11.1 after authenticated SSH: Target.CheckProtocolHost; missing bilateral runtime caller | TestProtocolIdentityRefusal; TestLoadPeerIdentityVersions | Comparator rejects another allowlisted ID, alias, malformed/zero target and accepts the exact configured ID. Conditional API proof only; no runtime operation is gated by it. Not counted as the requested network refusal. Explicit downstream ownership cycle blocks that proof. |
| Disconnects | §§11.1,16.1 process/stream consequence: Session.Send/Receive/Wait → supervise/readLines | TestReceiveFailureAndLimits/partial, /exit7, /exit255; TestCancellationAndCleanup/closed-input; TestReadFailureAndRecovery; empty and normal duplex controls | Partial LF frame, closed input/read error and unsuccessful process exit remain failures, never normal EOF. ReadFailureAndRecovery opens a fresh successful session. Counts local stream-disconnect propagation only; no claim about rollback of already-applied RPC operations or an actual dropped encrypted SSH connection. |
| Replay | §16.1 SSH transport integrity delegated by §11.1 | No committed actual cryptographic replay driver in the available predecessor APIs/tests | Not counted. A duplicated raw application line through Session.Receive is not SSH ciphertext replay. Do not invent a blanket duplicate-operation refusal: §11.2 nonce/echo/correlation and §11.3 operation-specific idempotency belong downstream, and valid retries may return recorded results. Required SSH replay driver must capture/reinject encrypted traffic in a disposable fixture and prove native rejection with normal-traffic/reconnect controls. |
| Oversized frames | §11.2 bound already enforced by accepted transport: Session.Send/readLines via Receive | TestDuplexAndSendBoundaries/oversize; TestReceiveFailureAndLimits/oversize; exact-MaxLineBytes controls in the same tests | 8 MiB accepted, one extra byte refused as ErrLine and session terminated. A sink prevents receive rejection from masking a missing send guard. Fresh independent sessions succeed. Counts the stream bound only; negotiated limits/JSON decoding remain downstream. |
| Disclosure mismatches | §6.4 configuration: peeridentity.Load → Directory.DisclosurePolicy; authenticated publisher absent | TestDisclosureClassPolicyBinding; TestDisclosurePolicyRefusals; TestSnapshotIsolationAndDisclosureBinding; allowed mesh_sanitized/reference_only neighbors | Five metadata classes retain per-peer policy, unset/local-only and unknown-class refusals. Snapshot mutation does not alter decisions. Not counted as network disclosure enforcement: policy strings do not authorize arbitrary bytes, and a test-only publisher cannot fill the gap. |

No durable AX write was introduced or exercised. Crash-write/idempotency evidence is therefore not claimed. Process/read recovery is distinct from durable operation recovery.

## Viable routing, tradeoffs, recommendation

1. Recommended: primary authorizes a narrow dependency/ownership reordering that makes the accepted identity/transport prerequisites available to the existing RPC owner, permits that owner to implement the real bilateral hello admission, and returns this seven-case conformance task to execute against it. Preserve both signed checkpoints. The primary must select a supported managed-workspace/Story delivery route; this worker must not integrate an unfinished Story or rewrite board files to achieve it. Keep the native key-change/replay fixture work and all seven rows in the eventual conformance deliverable.
2. Primary may instead explicitly transfer the minimal runtime identity-admission ownership into this Story and revise the downstream owner consistently. This changes architecture/scope and the currently explicit exclusion; it cannot be inferred by this worker. It risks duplicating handshake ownership unless resolved centrally.
3. A scope reduction to comparator/configuration tests would permit narrower evidence, but would not fulfill the current seven-case requirement. It is not recommended and was not performed.

Exact primary decision needed: resolve TASK-260830-2x16gz ↔ TASK-260830-z1yxg9 ordering/ownership while retaining full seven-case production conformance, and clarify whether disclosure mismatch acceptance is the §6.4 policy boundary or authenticated publishing. No dependency mutation or N/A substitution was made here.

A listener-free native OpenSSH harness with ephemeral fixture keys and a stdio SSH server/proxy is a viable engineering approach for key changes and transport replay; it was not attempted after the ownership stop and is not claimed impossible. It must suppress ambient config, agent/key state and use no real peers. This report does not cite its unbuilt design as test evidence.

## Direct validation and limits

Platform inspected: Go module (go 1.25.0), macOS/Linux/WSL2/Windows support; local runtime macOS arm64, Go 1.25.5 and OpenSSH per attached readiness logs. No iOS target. No dependencies/tools installed.

| Command | Real exit | Evidence |
| --- | ---: | --- |
| git verify-commit for each checkpoint above | 0, 0 | signature-identity-01.log, signature-transport-01.log |
| go test ./internal/peeridentity ./internal/sshtransport ./internal/specdoc -count=1 -v | 0 | focused-01.log, all three packages pass |
| go build ./... | 0 | build-01.log |
| git diff --check | 0 | diff-check-01.log |
| python3 internal/peeridentity/mutations.py --output .temp/TASK-260830-2x16gz/identity-mutants | 0 | identity-mutants-01.log and full per-probe events/results |
| python3 internal/sshtransport/mutations.py --output .temp/TASK-260830-2x16gz/transport-mutants --only neutral,known-bad,read-plus-one,send-plus-one,exit255 | 0 | transport-mutants-01.log and full per-probe events/results |
| git status --porcelain=v1; git diff --cached --exit-code HEAD; git diff --exit-code HEAD (separate processes) | 0 each | final-inspection-01..03.log; all empty |
| go list import graph; pinned digest and checkpoint tree/parent assertions | 0 | final-inspection-05.log, verified-identities.json |

Every gate was a direct standalone process without tee/pipeline status masking. Mutation child exits are 1 for named behavioral failures and 0 for passing controls/survivor; the harness exit 0 means its expectations held, not that mutant tests passed. There were no compile failures or missing-test kills. Earlier missing skill-path read exited 1, then the main-checkout Curator-managed skill was read successfully. A discovery rg of absent rpc/disclosure filenames exited 1; source caller census and import graph, not that filename result, establish the ownership evidence. No initial failing product test was hidden.

Full repository tests, full coverage, race, lint/vet, fuzz, cross-builds and hosted gates were not rerun: this run stopped before any repository delta at a concrete ownership prerequisite. Predecessor reviewer resources contain their prior full runs; those are inherited evidence only, not this run's seven-case proof. No full-suite or lint checklist item is checked on that basis. No new source-text enforcement gate was added; source searches above are inspection evidence, not behavioral kills.

## Narrowing witnesses rerun

21 of 22 narrowing witnesses killed: identity 18/19 and transport 3/3 selected; two known-bad controls killed and two neutral controls passed. All witnesses are predecessor-owned instrumentation, reproduced against the exact checkpoint. The protocol-ID mutant kill proves the comparator only and cannot substitute for its missing runtime caller. The exit255 kill proves error propagation only, not native changed-key or replay rejection.

| Mutant | What the gate is narrowed to admit | Named failing test | Exit | Result / surviving bound |
| --- | --- | --- | ---: | --- |
| disclosure-class-binding | Allows native observations to borrow manual metadata permission | TestDisclosureClassPolicyBinding/manual_metadata | 1 | killed:  |
| composed-argv-bound | Admits exactly 65537 composed SSH argv bytes | TestSSHTotalArgvByteBound | 1 | killed:  |
| neutral | Equivalent expression | none | 0 | neutral passed:  |
| known-bad | Fixed command changed, code still compiles | TestLoadPeerIdentityVersions | 1 | killed:  |
| local-duplicate | Peer-peer uniqueness retained; admits the local identity as a remote peer | TestLoadRefusesLocalHostAsPeer | 1 | killed:  |
| peer-duplicate | Admits duplicate identity with the different-alias fixture | TestLoadIdentityRefusals/duplicate_identity | 1 | killed:  |
| alias-duplicate | Admits duplicate alias for one peer ID | TestLoadIdentityRefusals/duplicate_alias | 1 | killed:  |
| local-id-grammar | Admits one malformed local host ID | TestLoadIdentityRefusals/malformed_local_UUID | 1 | killed:  |
| peer-id-grammar | Admits explicitly empty peer ID | TestLoadIdentityRefusals/empty_peer_ID | 1 | killed:  |
| alias-bound | Admits 65-character aliases | TestLoadIdentityRefusals/long_alias | 1 | killed:  |
| endpoint-grammar | Admits one option-shaped endpoint | TestLoadIdentityRefusals/option_endpoint | 1 | killed:  |
| ssh-auth-policy | Admits one grouped host-key bypass | TestLoadIdentityRefusals/grouped_host_key_bypass | 1 | killed:  |
| selector-allowlist | Admits one discovery candidate | TestResolveAllowlistAndAliasAmbiguity | 1 | killed:  |
| alias-ambiguity | Admits one ID/alias collision | TestResolveAllowlistAndAliasAmbiguity | 1 | killed:  |
| protocol-grammar | Allows one malformed protocol ID through grammar only; exact identity gate subsumes it | none | 0 | survived: Grammar alone is subsumed by exact target-ID equality; malformed IDs still fail equality. No independent grammar kill claimed. |
| protocol-id | Admits one other allowlisted peer ID | TestProtocolIdentityRefusal | 1 | killed:  |
| disclosure-class | Admits raw excerpts to the policy class registry | TestDisclosurePolicyRefusals | 1 | killed:  |
| disclosure-unset | Admits unset summary choice for one peer | TestDisclosurePolicyRefusals/unset_summary_choice | 1 | killed:  |
| disclosure-local | Admits local-only manual metadata | TestDisclosurePolicyRefusals/default_local_only | 1 | killed:  |
| disclosure-binding | Allows one peer override to affect another peer | TestSnapshotIsolationAndDisclosureBinding | 1 | killed:  |
| read-error | Admits full-looking bytes returned together with a failed read | TestLoadAbsenceReadFailureAndRecovery/partial_read | 1 | killed:  |

| Mutant | What the gate is narrowed to admit | Named failing test | Exit | Result / surviving bound |
| --- | --- | --- | ---: | --- |
| neutral | Equivalent size accumulator | none | 0 | neutral passed:  |
| known-bad | Adds a remote command token | TestOpenStructuredCommand | 1 | killed:  |
| send-plus-one | Admits exactly one excess outbound byte | TestDuplexAndSendBoundaries/oversize | 1 | killed:  |
| read-plus-one | Admits exactly one excess inbound byte | TestReceiveFailureAndLimits/oversize | 1 | killed:  |
| exit255 | Treats SSH exit 255 alone as successful EOF | TestReceiveFailureAndLimits/exit255 | 1 | killed:  |
