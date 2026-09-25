# TASK-260924-3n78rv: implement-projection-planning-and-authority-rules

## Description
Split from TASK-260830-1jmmqn on 2026-09-24 (one attack surface per leaf). Plan the pair-neutral target projection: strategy and profile selection, item mappings with expected dispositions and reasons, checkpoint message authority (visible text from typed escaped fields is low-authority user context, never an assistant reply, system instruction or authorization; pinned SPEC v0.7.0 ~10392, ~10587, ~10812), inactive tools/instructions as low-authority history, token metadata, and vendor importer strategies. Builds on the Projection Plan schema leaf. This is an implementation deliverable, not a specification rewrite.

## Scope
(define task scope)

## Acceptance Criteria
Planning decisions are driven through production entries with whole-domain oracles over strategy x profile x item class; injected instruction-like or reply-like visible text never gains authority at any entry; every rule has a narrowing killed by its named test run alone.
