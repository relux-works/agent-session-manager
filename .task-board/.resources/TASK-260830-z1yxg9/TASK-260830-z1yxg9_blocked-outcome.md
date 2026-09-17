# TASK-260830-z1yxg9 — inbound identity decision required

Status requested: blocked. Role: developer. Run: RUN-260908-282869.
No repository implementation delta; no Change Request or review readiness claim.
The entire original task and all seven subsequent hostile-peer cases remain open.

## Verified starting point

The managed Story worktree is at signed checkpoint
`c1eff016dce2e55c4e2c1828d5c20f3da84118bc`, tree
`f1dbf8f2dcd80ec8fdee9c0e7e1822cd921d41ac`. Identity checkpoint
`43c0e2b9` and transport checkpoint `c1eff016` signatures both verify (exit 0).
HEAD..main is 0 at inspection; this is not a claim of current remote-main equality.
Pinned AX v0.5.0 commit is `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`;
the embedded SPEC.md SHA-256 equals its source pin.

Read the ordering-decision precondition, both independent CR1 verdicts, and
TASK-260830-2x16gz_blocked-outcome.md. Live prerequisite tasks are integrating;
the current task depends on treeox, 33sfxc and accepted 1tvg8e. The ordering
repair is respected. This report does not revive the obsolete Story boundary.
Major negotiation remains 219okr and closed-operation conformance remains
19bjfj. Neither was implemented or rerouted.

## Precise unresolved boundary

The full assignment requires real bilateral protocol host admission after the
external SSH boundary. An outbound initiator can select its configured Target
before `Client.Open`, and compare the received hello host ID to that independent
target. The responder has no corresponding independent selection in the current
fixed `ax rpc serve --stdio` command, configuration, or transport API.

Evidence:

- SPEC.md 7040–7056 (§11.1) delegates SSH host-key/user authentication externally,
  fixes the remote command, and requires both sides' protocol host-ID check.
- SPEC.md 2420–2427 (§6.3) requires allowlisted stable ID plus the endpoint's
  externally authenticated host. It does not specify inbound user/key/principal
  to AX host-ID association.
- `internal/config/schema.go`, `Peer`, holds HostID, Name, Endpoint, Platform,
  SSHArgs and WorkspaceRoots. No inbound principal/host association exists.
- `internal/peeridentity/identity.go`, `Directory.Resolve` and
  `Target.CheckProtocolHost`, provide config selection and equality, explicitly
  not authentication evidence. The caller must already have the expected target.
- `internal/sshtransport/transport.go`, `Client.Open` and `Session.Target`,
  retain the outbound target. The package launches an SSH client; it provides no
  responder invocation context. Its raw frames are explicitly untrusted.
- The only current `CheckProtocolHost` callers are tests. The only binaries are
  cataloggen and tracecheck; an ax responder entry point has not already solved
  this association elsewhere.
- OpenSSH documents ExposeAuthInfo as optional (default no), exposing public
  user-authentication information through SSH_USER_AUTH. ForceCommand is an
  externally configured command restriction. Neither supplies an AX UUID
  association by itself. Official reference checked 2026-09-08:
  https://man.openbsd.org/sshd_config#ExposeAuthInfo and
  https://man.openbsd.org/sshd_config#ForceCommand . No host SSH settings,
  credentials, auth files, or actual peers were inspected or modified.

This is an unresolved trust-model choice, not a claim that OpenSSH cannot carry
such an association or that the pinned text explicitly requires a new key
registry. If membership-only inbound admission is the intended contract, that
needs a precise accepted bound: an authenticated SSH user may assert any listed
host ID. This cannot also be described as rejecting a listed host impersonating
another listed host. Fresh nonces, echoed nonces and request IDs bind messages;
they do not establish that absent external association.

## Failed assumption and concrete counterexample

Tempting composition: `d.Resolve(hello.host_id)` followed by
`target.CheckProtocolHost(hello.host_id)`. It checks membership and compares the
claim to an expected value chosen by the claim. It does not independently select
the inbound principal's host ID.

A disposable Go overlay drives the existing production Directory/Target methods:

- `TestResponderBindingIndependentSelectionControl` passes (exit 0): independently
  selected peer A accepts A, rejects other allowlisted B, and unknown resolution
  fails.
- `TestResponderClaimSelectedTargetCannotAttestInboundIdentity` is **expected red**
  (exit 1): selecting B from claimed B then checking B accepts it. The test demands
  the stronger independent-association property and demonstrates that this
  proposed composition cannot establish it.

These are architectural probes, not committed tests, not an implemented RPC
consumer, and not a real SSH spoofing/cryptographic attack. No observed native
SSH compromise is claimed. No synthetic external process exit is used as proof
of authentication. The expected-red test is not reported as green validation.

## Options and exact primary decision

1. Recommended if the assignment includes cross-listed-host impersonation:
   route an explicit external-SSH-to-AX inbound identity association contract and
   owner before this consumer. For example, a deployment-owned restricted
   invocation could select one expected configured identity independently of
   stdin. Primary must decide its authority, provisioning, supported transports,
   bypass resistance and delivery scope. This keeps key authentication outside
   AX but introduces an integration contract absent from the current pin/API.
   An optional OpenSSH auth-info mapping is another candidate, with portability
   and public-credential mapping costs; it is not selected here.
2. If the intended §11.1 server boundary is membership-only after externally
   authenticated SSH, explicitly confirm that interpretation and the threat-model
   bound. Implement exact UUID lookup plus typed hello validation without claiming
   that it distinguishes two listed hosts sharing authorized SSH access. Primary
   must reconcile that bound with the unchanged spoofed-host conformance case.
3. A caller-supplied `verified` boolean, arbitrary expected-peer argument with no
   production supplier, claim-selected target, endpoint/address comparison, or
   test-only host selector is not a substitute for the decision. None was added.

Exact requested decision: how does the real responder select the configured
allowlist entry independently of the received hello, or is membership-only
admission intentionally the accepted inbound trust boundary? Primary owns that
architecture/security interpretation and any graph repair. Do not retry this
producer unchanged to accumulate another version of the same gap.

## AC accounting

**0 of 7 AC rows driven through a new RPC production consumer.** No original
implementation acceptance criterion is discharged by this run. The seven rows
below expand the assigned deliverable, including its bilateral prerequisite.

| AC row | Required production call site | Current evidence / stated bound |
| --- | --- | --- |
| Bilateral host admission | Actual responder and initiator hello consumers -> Target.CheckProtocolHost | No consumer added; conditional Directory/Target probes above only. Independent responder target is unresolved. |
| Request/response correlation | RPC exchange request ID -> response echo validation | Not implemented or tested by this run. |
| Hello order, maps and nonce binding | RPC first-frame typed request/response consumer | Pinned examples inspected; no runtime driver. |
| Namespace/cardinality contracts | Typed hello/operation contract admission | Existing canonicaljson, CLI map vocabulary and ownership read; no RPC body admitted. |
| Limits and partial I/O | RPC negotiated limits over SSH Session.Send/Receive | Existing transport tests rerun; no negotiated RPC limit proof. |
| Deadlines and recovery | RPC session lifecycle over external SSH | Existing transport cancellation/recovery tests rerun; no hello deadline or operation recovery claim. |
| Structured errors and historical framing | RPC bootstrap response-or-close -> axerror exact binding | Pinned §§11.2, 11.3, 15.1, 17 inspected; no RPC error consumer/producer added. |

No durable AX mutation exists in this delta. No crash/idempotency claim is made;
operation-specific retry semantics remain unchanged. No README, doctor,
capability, ownership registry or specification claim was changed. No source-text
checker or new product gate was introduced.

## Direct verification

Local platform: macOS arm64; repository is Go with macOS/Linux/WSL2/Windows
support, no iOS target. See readiness logs for exact tools.

| Standalone command | Real exit | Evidence |
| --- | ---: | --- |
| go test ./internal/peeridentity ./internal/sshtransport ./internal/specdoc -count=1 -v | 0 | focused-01.log |
| go build ./... | 0 | build-01.log |
| git verify-commit 43c0e2b9 | 0 | signature-identity-01.log |
| git verify-commit c1eff016 | 0 | signature-transport-01.log |
| go test with binding-overlay.json, -run ^TestResponderBindingIndependentSelectionControl$ | 0 | binding-control-01.log |
| go test with binding-overlay.json, -run ^TestResponderClaimSelectedTargetCannotAttestInboundIdentity$ | 1 | binding-expected-red-01.log; expected failure of the proposed claim-selected composition |
| Selected predecessor mutation harness (four probes below) | 0 | selected-mutations-01.log; child exits and full commands in identity-mutants/results.json |

No tee or pipeline masked a gate exit. Focused tests/build were rerun here;
predecessor verdicts were read for acceptance provenance, not relabeled as new
validation. Full repository tests, full coverage, full race, vet/lint, fuzz,
cross-builds and configured final-tree gates were **not run**, because product
implementation stopped at this architecture decision before any repository delta.
Their checklist items remain unchecked. Nothing was committed, integrated,
remotely published, rebased or staged. No workers were spawned.

## Narrowing evidence

Predecessor gates only: **1 of 2 narrowing mutants killed**, one survivor, plus
a passing neutral and killed known-bad control. Every kill compiled and has a
named failing behavioral test. All applied overlays and full logs are attached.
The equality kill proves the comparator when independently selected; it does not
repair or prove an inbound selector.

| Mutant | What it admits / changes | Named failing test | Exit | Result / surviving bound |
| --- | --- | --- | ---: | --- |
| neutral | Equivalent external_ssh return expression | none | 0 | Neutral control, no kill claimed. |
| known-bad | Changes fixed ax command to wrong-command | TestLoadPeerIdentityVersions | 1 | Known-bad control killed. |
| protocol-grammar | Allows workstation through UUID grammar while retaining equality | none | 0 | Survivor: equality to the validated target still rejects malformed ID; no independent grammar kill. |
| protocol-id | Admits exactly one other allowlisted host ID while retaining mismatch gate | TestProtocolIdentityRefusal | 1 | Killed; comparator proof only. |

Outcome, logbook and evidence archive are attached with task-ID prefixes. The
worktree remains the accepted prerequisite tree; all evidence lives in ignored
scratch paths and board resources. Handoff is not invoked because role work is
blocked, not ready for review.
