# TASK-260830-8x76g1 rework logbook

- Root cause: identity calculation selected a trusted self field and validated
  generic JSON/common numbers, but it never invoked the selected schema's
  closed member contract. Invalid Blob Descriptor objects could therefore be
  digested and then verified with a correctly computed claim.
- Decision: one composed gate in `prepareObjectIdentity` owns shape admission
  for both calculation and verification. There is no second bypassable verifier
  path.
- Scope decision: Section 10.1 is an envelope plus schema-specific fields, not
  a closed universal record shape. Exact validation is implemented only where
  Sections 10.2 and 10.4 publish complete closed objects.
- Section 17.3 evidence is limited to the immutable migrated-from extension
  contribution. Publication, reference advancement, retention, configuration
  migration, and runtime capabilities remain explicitly unavailable.
- The first validator draft extended into unrelated Git semantic policy and
  reduced focused coverage to 67.0%. It was narrowed to the task's exact closed
  shape and BlobChunk invariants; final focused coverage is 80.7%.
- Final validation is green. Board validation retains 262 inherited
  `MISSING_ACTIVITY` diagnostics with exit 0; this task is not listed.
