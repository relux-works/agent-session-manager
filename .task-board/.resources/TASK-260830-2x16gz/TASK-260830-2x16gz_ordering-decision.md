# Peer authentication prerequisite ordering

Primary execution decision, 2026-09-08. This repairs implementation-board order under the existing full-project goal; it does not introduce a product rule or remove a conformance case.

## Evidence

TASK-260830-2x16gz requires seven hostile-peer/failure cases. Its accepted prerequisites are signed checkpoints43c0e2b9 (peeridentity) andc1eff016 (sshtransport), with independently accepted treesc775903a andf1dbf8f2. They deliberately do not claim runtime bilateral host admission. Pinned AX section11.1 requires both sides to verify protocol host_id after external SSH authentication; section11.2 defines the actual hello envelope. Existing TASK-260830-z1yxg9 owns that implementation but was blocked by2x16gz. Its other prerequisites treeox and33sfxc are both done. The producer's blocked-outcome and recovery evidence establish this ordering cycle without changing product files.

## Supported repair

Move the entire existing TASK-260830-z1yxg9 from STORY-260830-4qojoz into STORY-260830-1kiyj6. Its normative scope and acceptance criteria remain the same: envelope/hello, correlation, maps, namespace/cardinality, limits, deadlines and structured errors. Its code/rule ownership remains attached to the same task; no second handshake owner or substitute authorizer is created.

Replace its dependency on2x16gz with the accepted SSH transport task1tvg8e, preserving its other prerequisites. Make2x16gz depend onz1yxg9 as well as1tvg8e. Keep2x16gz as the final peer-auth conformance leaf, with all seven cases open. Add2x16gz as a prerequisite of the subsequent negotiation task219okr so another Story cannot start from an undelivered foundation. Remaining negotiation/closed-operation tasks stay in4qojoz.

The enlarged peer-auth Story will therefore deliver identity -> SSH transport -> actual RPC envelope/hello admission -> hostile-peer conformance in one managed worktree and one final Story delivery. No accepted checkpoint needs to be rewritten merely to accommodate a missing final CR. The original RPC Story retains negotiation and closed-operation conformance using the delivered prerequisite.

## Bounds and gates

Only apply after the stopped conformance recovery chain releases its workspace normally; preserve every accepted commit and any uncommitted work. Use task-board mutations, including its journaled set_parent, never board-file edits. Verify resulting graph has no dependency cycle, the new producer is ready, and conformance is still blocked on its actual prerequisite. All role spawns remain Codex Astra high/medium with fresh preflight.

The seven-case test description remains intact. Native changed-key and SSH ciphertext-replay evidence still require real hermetic external-SSH fixtures; config parsing or exit255 simulation is insufficient. Disclosure proof must be tied to the exact section6.4 policy enforcement and any actual runtime consumer needed for the claimed consequence; no network-publisher proof may be inferred from a policy string. If further implementation prerequisites are demonstrated, route those while preserving the requirement; do not mark a named row N/A simply to land.

Qualified-selector product syntax for TASK-260830-21gygk is a separate unanswered user decision. This ordering repair does not answer it. Source CI debt remains pending and never counts as passed.
