# TASK-260831-2susw4: extend-normative-pin-scope-to-board-claimed-sections

## Description
The landed normative source pin covers only normative_scope [1, 17, 20, appendix-a, appendix-d], but 64 of the 66 board Stories that declare a normative scope claim sections outside it. The board collectively claims sections 1-11, 13-20, Appendix A and Appendix D. The traceability gate therefore cannot bind most assigned scope, and every later Story hits the same wall, as CR-TASK-260830-1pbx0c-2 review finding 2 demonstrated for sections 10.1-10.4.

## Scope
internal/specpin normative_scope and its lock file, internal/catalog contract catalog coverage where section-scoped, and the internal/traceability ownership model and tracecheck section binding. Extend coverage to every section the board claims. Do not weaken existing digest pinning, contract identity, or compatibility evidence.

## Acceptance Criteria
The pinned normative scope covers every section the board Stories claim; tracecheck binds an assigned section scope to implementation owners and executable acceptance cases rather than only proving internal consistency of registered rows; a narrowed negative test fails when one section binding is removed while unrelated section bindings remain green; the existing v0.5.0 source digest, tag, commit, and SPEC.md SHA-256 are unchanged; the full configured validation suite passes.
