# STORY-260830-2cxybz: reboot-restore-and-parked-recovery

## Description
Implement recover durable sessions after process, service, logout, and host restart without premature activation according to relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c).

## Scope
Normative scope: §13.11-13.13. Preserve all stronger global AX invariants and historical contract versions.

## Acceptance Criteria
Recover durable sessions after process, service, logout, and host restart without premature activation is implemented through production entry points with positive, negative, compatibility, idempotency, and recovery evidence appropriate to the scope.
