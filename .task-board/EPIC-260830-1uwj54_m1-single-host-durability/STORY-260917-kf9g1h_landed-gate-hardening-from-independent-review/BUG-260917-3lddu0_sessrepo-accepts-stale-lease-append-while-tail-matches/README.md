# BUG-260917-3lddu0: sessrepo-accepts-stale-lease-append-while-tail-matches

## Description
Found by RUN-260917-3934e0 (claude-opus-5 max) while reviewing TASK-260830-1geqhj CR1, probes 9 and 15. internal/sessrepo checkAppend admits an append under a superseded lease as long as the chain tail still sits on that lease (ErrStaleLease fires only after the tail moves). Probe 15 shows the blast radius: a profile.changed event appended under losing lease A after successor A2 won is accepted and then drives the effective profile (yolo, --dangerously-bypass-approvals-and-sandbox), which Section 2.4 forbids: losing-lease or ambiguous events MUST NOT change the effective profile.

## Scope
internal/sessrepo append admission; the writers that compose it must keep working unchanged.

## Acceptance Criteria
An append under a superseded lease is refused even when the chain tail still sits on that lease; a losing-lease profile.changed can never become the effective profile source; both are pinned by tests and by a narrowing mutant.
