# TASK-260830-236x9n Review Logbook — CR Revision 2

## 2026-08-31 — Complete omit-self inventory gap

- Exact source: `relux-works/agent-session-manager-spec` commit
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- Finding: the candidate treats the 18 names summarized in Section 1.6 as the
  total identity surface, while the same pinned contract gives 14 additional
  schema-specific omit-self fields. The exported `CalculateObjectIdentity`
  entry falsely refuses representative Canonical Session, Terminal Backend
  Probe, Clone Raw Object Manifest, and Supported Environment Tuple Registry
  objects.
- Evidence: `missing-schema-probe-01.log` is expected-red with exit 1 against
  candidate source SHA-256
  `459fea91a440022a1673b3a01ab488cd38ea591c2e654d331347b0d0c7598e8f`.
- Decision: route to `to-dev`; this is ordinary implementation rework, not an
  external Stop-The-Line boundary.
- Additional anomaly: CR revision 2 includes an unrelated root `.DS_Store`
  binary, which must be removed from the next candidate.
- Preserved evidence: RFC 8785/full/race/trace/build gates pass; the revised
  Session Annotation wrong-reference narrowing mutant fails as intended.
