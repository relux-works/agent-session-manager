# TASK-260908-eea478: adopt-local-only-ci-and-disable-hosted-triggers

## Description
Apply the user-approved local-only CI policy: disable GitHub automatic workflows, preserve executable local gates and exact-head validation evidence, and document signed PR delivery without hosted CI.

## Scope
Repository workflow configuration and durable operator policy only; no source code delta

## Acceptance Criteria
AX automatic workflow execution is disabled and verified by GitHub state; the local-only exact-head validation rule is persisted in the primary goal and task evidence.
