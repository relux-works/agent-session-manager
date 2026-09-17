# Research logbook — TASK-260908-snfd1u

2026-09-08. Role: researcher. Proposal only; implementation owners remain blocked.

- FINDING: Pinned Section 2.3 supplies four unqualified resolution tiers, not a qualified-selector grammar. Session names are ASCII `[A-Za-z0-9][A-Za-z0-9._-]{0,63}` (Section 2.1); peer aliases have a broader string contract (Section 6). Typed source/ID syntax and canonical alias encoding avoid both delimiter and UUID/alias collisions.
- RECOMMENDATION, NOT APPROVAL: explicit index source, durable session UUID and lease revalidation; source never denotes the execution destination or winning owner.
- FINDING: Existing inbound `Resolve(claim).CheckProtocolHost(claim)` only checks membership. The owner's expected-red composition probe is recorded as exit 1; it was not rerun and is not a live SSH exploit.
- PLATFORM EVIDENCE: OpenSSH forced commands/key restrictions are deployable authentication building blocks, not an inherent AX identity association. General-purpose account access, startup code, writable mappings and client-supplied environment can bypass a wrapper. Windows lacks several POSIX OpenSSH auth-info/hooks. Tailscale SSH uses a separate node-policy authentication path; its docs do not establish the proposed protected responder context.
- RECOMMENDATION, NOT APPROVAL: dedicated restricted ingress identities with administrator-owned mapping and an OS-authenticated IPC hop to the AX service. A caller cannot select the expected peer via argv/env. Administrative provisioning cost and native-Tailscale limitation are explicit product decisions; mutual TLS with separate AX credentials is the standard-protocol alternative.
- SUPPORTING BOUND: Git capture needs a held boundary and a coherent external-writer strategy; repeated hashes are not proof against change/revert. Public `creating` requires an initial lease; a record-only interrupted create cannot yield invented public owner/lease fields.
- PRESERVATION: No existing implementation files, task statuses, AC, runtime ownership, spec pin, source-tool PR or deferred issue 176/177 changed. Findings are stored in this task's root scratch and attached through resource CRUD. Root/worktree LOGBOOK.md was deliberately not edited because this assignment permits only task-scoped scratch and own-task writes.
- EXECUTION: No Go suites, code gates, live SSH authentication, platform deployment, credential inspection or transport mutation were performed. Platform claims are documentation-backed; integrated bypass resistance is a future acceptance requirement.
- TOOL ANOMALIES: Initial `task-board q 'task(...)'` was rejected for unknown operation; corrected to documented `get(...)` (exit 0). Its containing read/inventory shell returned 2 because missing local skill paths also produced rg errors. A later inventory shell returned 1 after searching peeridentity/sshtransport in the wrong Story and finding no standalone logbook executable. Located both packages in STORY-260830-1kiyj6 and used the authorized task-scoped Markdown logbook instead. No workflow depended on the failed probes. The attempted obsolete Tailscale whoami URL returned an internal web error and supplies no evidence; all cited Tailscale claims use successfully retrieved official current docs. These were read-only discovery failures, not expected-red gates.

## Evidence and real exits

- Required initial `task-board m 'set_status(TASK-260908-snfd1u, status=analysis)'`: exit 0.
- `task-board --help`, `git --version`, `rg --version`, `python3 --version`: readiness logs in the same scratch directory; containing calls exit 0.
- Direct Python pin/snapshot check: exit 0. Verified exact embedded SPEC SHA-256 and recorded two source manifests, heads and local-main ancestry.
- Document word count: direct Python process, exit 0. Proposal is within the requested 1500–2200-word bound.
- Remaining artifact/reference/preservation audit: see attached evidence ledger and direct validation log; results entered only after the process returns.

Historical implementation suite/mutation numbers are neither revalidated nor counted as this research task's acceptance evidence.

- AUDIT RESULT: standalone `validate-proposal.py` exited 0; proposal is 1944 words, four exact source-resource links resolve, both pins match, and 466/466 plus 461/461 inventoried source bytes remain unchanged. This is document/scope validation only; actual deployment conformance remains untested.
