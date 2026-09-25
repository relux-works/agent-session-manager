# TASK-260924-2zboyz: implement-readback-manifest-and-validation-report-schemas

## Description
Split from TASK-260830-1esv6u on 2026-09-24 (one attack surface per leaf). Clone Read-Back Evidence Manifest 1.0.0 (staged|live, bound to plan/projected manifest, equal expected/observed native IDs, target tuple, parsed count/heads, Workspace Binding, structural digest, sorted evidence blobs; modes cannot be relabeled) and Clone Validation Report 1.0.0 (aggregates both reads, identity/workspace/marker/resume checks, Fidelity Report, findings, valid result; every applicable check must pass) as closed schemas with JCS identities, pinned SPEC v0.7.0 §13.14.2. Reuse internal/clonefidelity and internal/cloneplan as the only validators of their rules. This is an implementation deliverable, not a specification rewrite.

## Scope
(define task scope)

## Acceptance Criteria
Both schemas are closed (reflection-derived member census, every member missing/extra/duplicated/miscased/wrong-type refused), ids recomputed independently, staged/live modes cannot be relabeled, every applicable check must pass for valid=true (whole-domain oracle over check outcomes), every entry calls the owner validators (AST census), each gate has a narrowing killed alone.
