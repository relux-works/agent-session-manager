# TASK-260830-236x9n Rework Logbook

## 2026-08-31 — Schema-directed self identity

- Finding: counting every top-level name in the global 18-name ID registry
  falsely rejected four valid directory contracts because registered names can
  be ordinary references.
- Decision: use the exact pinned `schema` / `schema_version` contract as the
  sole self-field selector. Do not accept a caller-chosen omit field and do not
  infer identity from digest-looking values.
- Decision: require `document_kind=managed_replica_marker` for the shared
  Materialization Journal 2.0.0 schema/version pair. Mutable journal variants
  are not immutable marker identities.
- Evidence: a narrowed source mutant mapping Session Annotation to referenced
  `profile_id` failed the exported `CalculateObjectIdentity` production-entry
  test with exit code 1; the restored mapping passed with exit code 0.
- Anomaly corrected: README advertised 26 traceability acceptance cases while
  the production gate reported 29. README now matches the gate.
- Scope boundary: no durable mutation or capability/doctor/migration surface
  was introduced, so no crash-recovery claim is made.
