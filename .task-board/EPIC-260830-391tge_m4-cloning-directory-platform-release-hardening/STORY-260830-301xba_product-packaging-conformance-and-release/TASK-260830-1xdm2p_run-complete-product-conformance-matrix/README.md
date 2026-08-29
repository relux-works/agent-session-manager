# TASK-260830-1xdm2p: run-complete-product-conformance-matrix

## Description
Run platform lanes, provider suites, terminal backends, cloning, Directory, lifecycle, fault, security, and migration acceptance cases. This is an implementation deliverable, not a specification rewrite.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c); §19-20, Appendices A-D. Work only inside the product-packaging-conformance-and-release story boundary.

## Acceptance Criteria
Production behavior demonstrates: Run platform lanes, provider suites, terminal backends, cloning, Directory, lifecycle, fault, security, and migration acceptance cases. Exact contract fixtures and negative/refusal cases pass; crash/idempotency evidence is included when the operation mutates durable state; no unsupported capability is advertised.
