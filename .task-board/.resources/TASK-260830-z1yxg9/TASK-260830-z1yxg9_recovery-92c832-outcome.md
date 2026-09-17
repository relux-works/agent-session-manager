# TASK-260830-z1yxg9 — recovery RUN-260908-92c832

Disposition: blocked on the unchanged inbound trust-model decision. No implementation delta or review-ready Change Request.

## Current authority and preserved scope

Live notes identify this as automatic recovery attempt 1/3 from RUN-260908-282869, not a new product or architecture decision. No operator directives were present at the start or after validation. The full assignment, its ordering-decision precondition, both independent prerequisite CR1 verdicts, prior z1yxg9 blocked outcome and 2x16gz seven-case map were read. The ordering repair remains respected: current prerequisites are treeox, 33sfxc and 1tvg8e. Identity and SSH transport remain accepted, checkpointed prerequisites; their board status is integrating.

Worktree HEAD is c1eff016dce2e55c4e2c1828d5c20f3da84118bc, tree f1dbf8f2dcd80ec8fdee9c0e7e1822cd921d41ac. Both checkpoint signatures (43c0e2b9 and c1eff016) verify. HEAD..local-main is 0; no fresh remote-main claim. Pinned SPEC.md hash was independently checked against 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a, AX v0.5.0 commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c. Repository target is Go on macOS/Linux/WSL2/Windows; this run uses macOS arm64 / Go 1.25.5, not iOS.

No product code, tests, README, ownership registry, doctor/capability claims, branch, index or signing settings changed. No workers spawned, upstream issues176/177 changed, or remote publication attempted. Subsequent major negotiation219okr, closed-operation conformance19bjfj and all seven hostile-peer cases2x16gz retain their owners and scope.

## Constraint, failed assumption and current evidence

Section11.1 (SPEC.md lines7040–7056) fixes the SSH remote command and requires both sides to match protocol host_id to a configured allowlist entry after external SSH authentication. Section6.3 (2420–2427) requires stable allowlist identity and externally authenticated endpoint. These clauses do not define an inbound SSH-principal-to-AX-host association.

Client.Open selects its outbound Target before consuming untrusted bytes. The responder has no corresponding independent selection in config.Peer (internal/config/schema.go), Directory/Target (internal/peeridentity/identity.go), Session (internal/sshtransport/transport.go), or fixed `ax rpc serve --stdio`. Caller census again finds CheckProtocolHost only in tests and a transport comment. The accepted transport expressly exposes raw untrusted frames, not admitted identity. Reading the nonce and contracts does not supply the missing external identity association.

The tempting implementation `Resolve(hello.host_id).CheckProtocolHost(hello.host_id)` establishes membership only: an SSH caller can assert another listed ID. A new arbitrary expected-peer argument without a real production supplier or a caller-minted verified flag would hide the decision. Neither was added. This is a trust-model ambiguity under the assigned stronger spoofing expectation, not proof that OpenSSH is incapable of supporting a suitable deployment contract or that the pinned text mandates a new key registry.

Fresh disposable overlay tests run against production Directory/Target methods:

- TestResponderBindingIndependentSelectionControl: PASS, exit0. An independently selected A accepts A, refuses listed B and unknown lookup fails.
- TestResponderClaimSelectedTargetCannotAttestInboundIdentity: EXPECTED FAIL, exit1. Choosing expected B from claimed B admits it; no independent association is established.

These tests are architectural probes inherited from the earlier run and rerun here. They are not committed RPC tests, an implemented responder, real SSH spoofing, or cryptographic evidence. The expected-red test demonstrates the unsupported stronger interpretation; it is not green validation and does not mean the existing membership/comparator API violates its documented contract.

## Exact primary decision and routes

Recommended for cross-listed-host impersonation resistance: primary routes a real external-SSH-to-AX inbound identity association contract and its owner before this consumer. A deployment-owned restricted invocation can select an expected configured identity independently of stdin, but its authority, provisioning, supported transports and bypass resistance need an explicit decision. This adds an integration contract, so a leaf worker cannot silently select it.

Alternative: primary explicitly accepts membership-only responder admission after externally authenticated SSH and reconciles the unchanged spoofing AC with its bound: it does not distinguish listed hosts sharing authorized SSH access. This is a smaller implementation contract with a weaker attribution guarantee.

Exact missing input: how does the real responder independently select the expected configured peer, or is membership-only admission the intentionally accepted trust boundary? No tool repair, retry, endpoint comparison or nonce flag resolves that choice. Another unchanged automatic recovery run is not new authority. Primary owns routing and any product decision; the full deliverable stays open.

## AC accounting

**0 of 7 AC rows driven through a new RPC production consumer.** No row is discharged by this recovery.

| AC row | Required production call site | Stated bound |
| --- | --- | --- |
| Bilateral host admission | Initiator/responder hello consumer -> Target.CheckProtocolHost | Conditional comparator probes only; independent inbound association undecided. |
| Request/response correlation | RPC request/response consumer | No consumer added. |
| Hello order/maps/nonces | First-frame typed hello consumer | No consumer added. |
| Namespace/cardinality contracts | Typed RPC admission | No consumer added; existing owners preserved. |
| Limits/partial I/O | RPC limits over Session.Send/Receive | Prerequisite framing suite rerun, no negotiated-limit claim. |
| Deadlines/recovery | RPC session lifecycle | Prerequisite cancellation/recovery suite rerun, no RPC lifecycle claim. |
| Structured errors/historical versions | RPC response-or-close -> axerror binding | No consumer added; bindings preserved. |

No durable mutation was introduced or driven, so no crash/idempotency proof is claimed. No capability is advertised. No new source-text guard or behavioral gate is introduced. Checklists remain unchecked because the implementation AC has not been met.

## Direct command results

All listed validations were standalone subprocesses, without tee or pipeline masking. Exact argv and real exits are in commands.json; logs have matching names. This run reran focused tests/build/signatures and both architectural probes itself. Prior full-suite and mutation results are inherited evidence only.

| Command | Real exit | Log |
| --- | ---: | --- |
| `python3 --version` | 0 | readiness.log |
| `git verify-commit 43c0e2b9` | 0 | identity-signature.log |
| `git verify-commit c1eff016` | 0 | transport-signature.log |
| `git show -s --format=%H %T %P HEAD` | 0 | tree.log |
| `go test ./internal/peeridentity ./internal/sshtransport ./internal/specdoc -count=1 -v` | 0 | focused.log |
| `go build ./...` | 0 | build.log |
| `go test -overlay /Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/TASK-260830-z1yxg9/RUN-260908-92c832/binding-overlay.json ./internal/peeridentity -count=1 -v -run ^TestResponderBindingIndependentSelectionControl$` | 0 | binding-control.log |
| `go test -overlay /Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/TASK-260830-z1yxg9/RUN-260908-92c832/binding-overlay.json ./internal/peeridentity -count=1 -v -run ^TestResponderClaimSelectedTargetCannotAttestInboundIdentity$` | 1 | binding-expected-red.log |
| `git status --porcelain=v1` | 0 | status.log |
| `git diff --check` | 0 | diff-check.log |
| `git diff --cached --exit-code HEAD` | 0 | index.log |
| `git diff --exit-code HEAD` | 0 | diff.log |
| `task-board spawn directives RUN-260908-92c832` | 0 | directives.log |
| `task-board q --format compact get(TASK-260830-z1yxg9) { status notes blockedBy checklist outcomeResources }` | 0 | live-task.log |
| `git rev-list --count HEAD..main` | 0 | main-distance.log |

The expected-red command exits1 because the unsupported claim-selected construction admits another configured identity. It is deliberately reported failing. Full repository tests, full coverage, race, vet/lint, fuzz, cross-platform builds and configured handoff gates were not rerun: the same unresolved trust-model decision stopped product changes. No command-specific completion box is checked from inherited evidence.

## Inherited narrowing accounting — not rerun here

The preceding run's attached evidence.zip and outcome report **1 of 2 narrowing mutants killed**, one survivor, plus valid neutral/known-bad controls. They were read, not relabeled as new results. No narrowing mutant was executed in this recovery; no new gate ships.

| Mutant | What it narrows the gate to | Named failing test | Prior exit | Result / surviving bound |
| --- | --- | --- | ---: | --- |
| protocol-id | Admits one additional allowlisted UUID while retaining mismatch check | TestProtocolIdentityRefusal | 1 | Comparator kill only; no inbound association proof. |
| protocol-grammar | Admits workstation through UUID grammar, retains equality | none | 0 | Survivor: equality against the validated target still refuses it; no independent grammar kill. |
| neutral | Equivalent external_ssh expression | none | 0 | Neutral control only. |
| known-bad | Fixed command becomes wrong-command | TestLoadPeerIdentityVersions | 1 | Instrument control only. |

Fresh recovery outcome, logbook and evidence archive are attached before blocked disposition. Handoff is not invoked: its developer precondition is review readiness, which this run does not claim.
