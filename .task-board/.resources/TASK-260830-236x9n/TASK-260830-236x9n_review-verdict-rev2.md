# TASK-260830-236x9n Review Verdict — CR Revision 2

Verdict: **changes requested**. Route: `to-dev`.

Reviewed Change Request `CR-TASK-260830-236x9n-2` revision 2, base
`006ba4ebe1d59525f3ea266497a09848cf781c2c`, candidate tree
`5ac8966b178e722ecb205cc02392b9738ef57722`, patch SHA-256
`12f45af43b1dea6f1524161d183fea860898f7032a821823909b13dd3a0e5b20`.

## F1 — Blocking: the production identity registry is not total

`internal/canonicaljson/canonical.go:28-73` declares the 18 names copied from
the Section 1.6 summary to be the total, closed self-identity namespace, and
`schemaIdentityContracts` at lines 90-123 accepts only schemas using those
names. That is narrower than the exact pinned v0.5.0 contract. The same pinned
specification defines additional immutable schemas with an explicit JCS
omit-self identity, including:

- `terminal-backend-probe/probe_id`,
  `terminal-instance-binding/binding_id`, and
  `terminal-capability-evidence/evidence_id`;
- `clone-raw-object-manifest/raw_object_manifest_id`,
  `clone-capture-manifest/capture_manifest_id`, and
  `canonical-session/canonical_session_id`;
- `fidelity-report/fidelity_report_id`,
  `projection-plan/projection_plan_id`,
  `clone-projected-object-manifest/projected_object_manifest_id`,
  `clone-read-back-evidence-manifest/read_back_evidence_manifest_id`, and
  `clone-validation-report/validation_report_id`;
- `migration-checkpoint/migration_checkpoint_id`,
  `clone-lineage-receipt/lineage_receipt_id`, and
  `supported-environment-tuples/registry_digest`.

This is not a theoretical catalog discrepancy. A task-scoped probe drove the
exported production entry `canonicaljson.CalculateObjectIdentity` against the
exact candidate source. It falsely refused all four representative normative
objects below with `has no supported immutable self-identity contract`:

1. Canonical Session 1.0.0;
2. Terminal Backend Probe 1.0.0;
3. Clone Raw Object Manifest 1.0.0;
4. Supported Environment Tuple Registry 1.0.0.

The current green test named
`TestCalculateObjectIdentityOmitsEveryNormativeSelfField` is circular: it
constructs its expected set from the same narrowed 18-name interpretation and
asserts that production has exactly 18 entries. It therefore cannot detect a
missing schema or a self field outside that list. This is the negative-evidence
shape **a narrowed bound around the production gate**: the selected subset is
well tested, while required members outside the subset are rejected.

Required rework:

1. Build the schema/version-to-self-field contract from the complete pinned
   v0.5.0 omit-self surface, not only the Section 1.6 summary list. Preserve
   schema discriminators such as the managed-replica marker variant.
2. Add production-entry calculate and verify coverage for every pinned
   schema/version that declares an omit-self identity, including self fields
   not present in the current `SelfField` constants.
3. Add a narrowing mutant that removes or misroutes one member of this broader
   schema set and require the production-entry test to fail. Keep the four
   reference-collision tests from revision 2; their wrong-reference mutant was
   independently rerun and failed as expected.
4. Correct README wording that presents the package as the AX immutable-object
   identity calculation until the full pinned surface reproduces.

## F2 — Repository hygiene

The candidate adds a 6,148-byte Apple `.DS_Store` binary at repository root.
It has no product, test, traceability, or documentation role and is outside the
task scope. Remove it from the next candidate (and ignore this artifact if the
repository policy intends to prevent recurrence).

## Independent validation

Validation ran against a clean archive of the exact candidate tree, not the
managed worktree's mixed index state.

| Check | Result |
| --- | --- |
| Patch resource SHA-256 and candidate tree identity | Match the handed-off CR |
| `go test ./internal/canonicaljson -count=1 -v` | Pass |
| `go test ./... -count=1 -v` | Pass |
| `go test ./... -cover -count=1` | Pass; `canonicaljson` 82.6% |
| `go test -race ./... -count=1` | Pass |
| Assigned tracecheck for 1.6 and 10.1-10.4 | Pass; 5 scopes, 29 cases |
| Default tracecheck | Pass |
| `tracecheck -section 10.999` | Expected refusal, exit 1 |
| Wrong Session Annotation mapping mutant | Expected-red, exit 1 |
| Missing-schema production-entry probe | Expected-red, exit 1; four false refusals reproduced |
| `go vet ./...`, `go build ./...`, `go mod verify`, `gofmt -l` | Pass/clean |
| `git diff --check` | Pass |

RFC 8785 vectors, strict JSON refusals, representation stability, the four
revision-1 reference collisions, and the wrong-reference narrowing mutant are
all sound in this candidate. The new finding is that those tests prove only a
subset of the promised total identity surface.

The package remains deterministic and read-only, so durable crash/recovery
evidence is not applicable. The revision-1 dependency assessment is unchanged:
`github.com/gowebpki/jcs v1.0.1` is load-bearing for ECMAScript number
serialization, string serialization, and UTF-16 property ordering; this verdict
does not reopen that separate owner decision.
