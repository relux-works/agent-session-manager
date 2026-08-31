# STORY-260831-2xdhn9: normative-pin-scope-completion

## Description
Complete the normative source pin so its declared scope covers every section the implementation board actually claims. The landed pin scoped only sections 1, 17, 20, Appendix A and Appendix D, while 64 of the 66 board Stories that declare a normative scope claim sections outside it.

## Scope
internal/specpin normative scope and lock file, internal/catalog section-scoped coverage, and the internal/traceability section-binding model and tracecheck. The pinned v0.5.0 source identity is fixed and must not change.

## Acceptance Criteria
Every section the board claims has pinned scope and an enforceable traceability binding; removing a single section binding fails the gate while unrelated bindings stay green; source identity digests are unchanged.
