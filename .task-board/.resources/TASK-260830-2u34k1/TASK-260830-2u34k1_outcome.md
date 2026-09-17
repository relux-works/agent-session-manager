# TASK-260830-2u34k1 — developer review handoff

Base: `7654d7cadb2c226bfa3db5f23ac355a730285eea`; managed Story worktree
`STORY-260830-1kiyj6`, unchanged branch/checkpoint. HEAD..main was 0 at inspection.
No manual commit, branch operation, remote connection, private-key inspection,
credential export, or remote publication was performed.

## Scope and production entry coverage

**7 of 7 scoped AC rows driven.** These are the seven deliverables explicitly
named by this leaf, not a claim that all normative clauses in §§6/11.1/16.1
or the sibling authenticated transport are implemented. Tests are in the
review candidate and will be committed through managed checkpoint/integration.

| AC row | Production call site | Named candidate tests | Bound |
| --- | --- | --- | --- |
| Host IDs | peeridentity.Load -> config.Load/Decode -> scalar UUIDv7 owner | TestLoadPeerIdentityVersions, TestLoadIdentityRefusals, TestLoadPinnedPeerConfigurationExample | Existing configured identity only; no host generation or persistence |
| Peer aliases | Directory.Resolve | TestResolveAllowlistAndAliasAmbiguity, TestSnapshotIsolationAndDisclosureBinding | Exact printable UTF-8 names, no case folding; ID/name collision refuses ambiguity |
| SSH targets | Directory.Resolve -> Target.RPCArgv | TestSSHTargetAtomicArgv, TestSSHTotalArgvByteBound, TestLoadPinnedPeerConfigurationExample, TestLoadIdentityRefusals | Atomic fixed RPC command arguments, canonical endpoint/options admission, port/IPv6 translation and total composed argument byte bound; no process launched |
| Key provenance | Target.KeyProvenance; Load selected-file reader | TestLoadPeerIdentityVersions, TestLoadAbsenceReadFailureAndRecovery, TestSSHTargetAtomicArgv | §11.1 delegates key/user authentication to external SSH; reports authority only. No actual key/fingerprint/verified-authentication assertion, invented config field or key reader |
| Allowlists | Directory.Resolve; Target.CheckProtocolHost | TestResolveAllowlistAndAliasAmbiguity, TestProtocolIdentityRefusal, TestLoadAbsenceReadFailureAndRecovery | Explicit configured stable IDs only; discovery/endpoint/another allowed ID cannot substitute. Protocol comparison is conditional on transport-owned SSH authentication |
| Disclosure policy | Directory.DisclosurePolicy | TestDisclosurePolicyRefusals, TestDisclosureClassPolicyBinding, TestSnapshotIsolationAndDisclosureBinding | Five metadata classes, config default/override and unset-summary refusal only; publisher still owns object validation/sanitization and authenticated export |
| Duplicate identity refusal | config.validateMesh through peeridentity.Load | TestLoadIdentityRefusals, config.TestLoadRefusesLocalHostAsPeer | Local-ID reuse and peer-peer duplicate IDs refuse separately from duplicate alias and runtime ID/alias ambiguity |

No durable operation was introduced. Crash/write-idempotency injection is N/A.
The real loader is exercised with missing, empty, malformed, failed-stat and
partial-with-error reads, followed by recovery. A failed read yields no usable
peer directory; it is never silently converted to an empty allowlist success.
Targets/snapshots/argv copies isolate caller mutation. Formatting does not
expose endpoints or key-file selectors. Test inputs are synthetic and hermetic.

README, .spec implementation map, ownership registry and its digest/count fixtures
are updated together. Seven executable acceptance links were added (98 -> 105);
measured normative section coverage is unchanged (17/428 keyword clauses), with
explicit partial-scope gaps. No CLI, doctor, transport, provider authentication,
at-rest encryption or replicated-data erasure capability is advertised.

## Validation

All gates were run by this developer, not accepted from another worker. Commands
ran as standalone processes with log redirection, no tee. Gate JSON contains
exact command, log and exit status; the tracked-JSON pipeline was additionally
rerun with zsh pipefail, exit 0. Initial red iterations remain red in this table.

| Command | Exit | Log | Result / limitation |
| --- | ---: | --- | --- |
| `go test ./internal/peeridentity ./internal/config` | 1 | `focused-01.log` | Initial two incorrect expected error labels; fixed. |
| `go test ./internal/peeridentity ./internal/config -count=1` | 0 | `focused-02.log` | Focused green. |
| `go test ./... -v` | 1 | `tests-all-01.log` | Old compiled ownership digest overlapped new registry; not accepted evidence. |
| `go test ./... -v` | 1 | `tests-all-02.log` | Missed tracecheck output fixture 98 versus 105; fixed. |
| `go test ./... -v` | 0 | `tests-all-03.log` | 24 packages pass; Go cache used where applicable. New class-binding test added subsequently and run explicitly below. |
| `go test ./... -cover` | 1 | `coverage-01.log` | Same tracecheck output-count fixture failure. |
| `go test ./... -cover` | 0 | `coverage-02.log` | 24 packages pass. Peeridentity 97.5%; config 94.7%. |
| `go test ./... -race -count=1` | 1 | `race-01.log` | 23 packages pass; same tracecheck output-count fixture fails. No race detector report. |
| `go test ./internal/peeridentity ./internal/config ./internal/traceability/... -race -count=1` | 0 | `race-changed-02.log` | All changed packages pass after output-count fix. |
| `go test ./internal/peeridentity -count=1 -race -cover` | 0 | `identity-04.log` | Includes final per-class disclosure binding test; 97.5% coverage. |
| `python3 internal/peeridentity/mutations.py --output .temp/TASK-260830-2u34k1/mutations` | 1 | `mutations-01.log` | Unproven raw-class mutant survived; test strengthened. |
| `python3 internal/peeridentity/mutations.py --output .temp/TASK-260830-2u34k1/mutations-02` | 0 | `mutations-02.log` | Raw-class plant now killed; one documented subsumed survivor. |
| `python3 internal/peeridentity/mutations.py --output .temp/TASK-260830-2u34k1/mutations-03` | 0 | `mutations-03.log` | Added composed-argv bound plant. |
| `python3 internal/peeridentity/mutations.py --output .temp/TASK-260830-2u34k1/mutations-04` | 0 | `mutations-04.log` | Final corpus includes class-policy binding plant. |
| `go test ./... -race -count=1` | 0 | `race-02.log` | Stable production tree, full uncached race rerun. |

The 24 additional configured gates all exited 0: formatting, native build/vet,
13 fixed 100x fuzz smokes, tracecheck, catalog check, Linux/Windows amd64 builds,
tracked JSON parse, board validation, diff check and Windows vet. A fuzz-smoke
exit is not a claim every target reached mutation (baseline corpus may consume
the 100x budget). Cross-platform builds do not claim native Windows/Linux runtime
execution. No iOS target exists; native host is macOS arm64, Go 1.25.5.

**Board bound:** `task-board validate` returned exit 0 while reporting 225
shared activity/mirror issues. No issue named this task or Story. This is not an
issue-free-board claim. Shared board repair was not attempted.

## Narrowing evidence

Final behavioral corpus: 21 applied compiling probes = 18 narrowing kills,
1 known-bad control kill, 1 green neutral control, 1 surviving subsumed grammar
probe. Thus **18 of 19 narrowing probes killed**. Every kill has a named failing
test and actual go-test exit 1; parent harness exit 0 means its expected outcomes
matched. A compile failure, absent test, failed plant or invalid control is never
a kill. Originals are byte-checked unchanged while Go overlays supply mutants.
The harness runs production-entry behavioral tests, not the existing config
source census. It adds no new source-text-only enforcement gate. Guard tokens
and refusal strings remain present in narrowing plants.

The surviving protocol grammar plant bypasses grammar rejection for the literal
alias `workstation`; exact equality to the already validated configured UUID
still refuses it. Grammar is subsumed, not independently evidenced as a gate.
The earlier raw-class survivor was a test hole: local_only masked the missing
class refusal. A permissive-default case now kills that same plant.

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

## Evidence and review state

The attached ZIP contains all logs, exact commands/results, applied overlay
bodies and mutation tables, plus the final file hash manifest. LOGBOOK.md records
the authority boundary, discovered test gap, red verification iterations and
shared board anomaly. Work is left uncommitted for `task-board handoff --role
developer`; the orchestrator owns review, signed checkpoint and integration.
Ready for review; transport/lifecycle siblings remain outside this leaf.
