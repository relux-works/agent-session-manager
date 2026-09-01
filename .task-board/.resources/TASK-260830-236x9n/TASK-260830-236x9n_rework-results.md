# TASK-260830-236x9n Rework Results

## Outcome

The review-blocking global self-field count was replaced with a fail-closed
schema-directed identity contract. `CalculateObjectIdentity` and
`VerifyObjectIdentity` now resolve the omitted field from the exact validated
`schema` / `schema_version` pair. The Materialization Journal 2.0.0 mapping
also requires `document_kind=managed_replica_marker`, so mutable journal
variants cannot enter the immutable marker identity path.

Other registered ID names remain in the hashed object as references. Production
entry tests cover Session Annotation (`annotation_id` with `profile_id`),
Enrichment Job Request (`job_request_id` with `profile_id`), Enrichment Job
Receipt (`job_receipt_id` with `job_request_id` and `profile_id`), and Directory
Operation Receipt (`directory_receipt_id` with `plan_id`). The README now states
the schema-directed rule and no longer claims every object has exactly one
globally known top-level ID name.

The new collision test is registered under the existing
`canonical-object-identity` traceability acceptance case. The reviewed semantic
ownership projection digest was updated to
`481012e1629bd8d6f3a44f9b80526826d6bdedbc8540f0da5a9d4c1c9c156b03`.
The README inventory count was corrected from 26 to the current 29 acceptance
cases reported by the production tracecheck entry point.

## Normative source

- Source: `relux-works/agent-session-manager-spec` commit
  `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c` (`v0.5.0`).
- Materialized `SPEC.md` SHA-256:
  `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.
- Applied rule: Section 1.6 says each schema names one self field; schema/version
  selects the omit field. Section 10.8 provides the four reference-collision
  object shapes required by the review verdict.

## Negative evidence

The source-level narrowing mutant changed only the Session Annotation mapping
from `annotation_id` to the referenced `profile_id`, then ran:

```text
go test ./internal/canonicaljson \
  -run '^TestCalculateObjectIdentityUsesSchemaDefinedSelfFieldWithRegisteredReferences/Session_Annotation$' \
  -count=1
```

Real exit code: **1** (expected-red). The production-entry test reported that
`CalculateObjectIdentity` returned `profile_id` and a different digest instead
of the schema-defined `annotation_id`. The source file was restored byte-for-byte
from its pre-mutant copy, and the full four-shape production-entry test then
passed with exit code 0.

`go run ./internal/traceability/cmd/tracecheck -section 10.999` also refused the
nonexistent section with real exit code **1** (expected-red). The intentional
ownership-registry edit first produced a reviewed-digest mismatch with real
exit code **1** before the new semantic projection digest was pinned; both
default and assigned-scope tracecheck runs passed after that explicit update.

## Final validation

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/canonicaljson -count=1 -v` | 0 | All RFC 8785, schema mapping, reference collision, digest-cycle, and refusal tests passed |
| `go test ./internal/canonicaljson -cover -count=1` | 0 | 82.6% statement coverage |
| `go test ./... -v -count=1` | 0 | Repository-wide tests passed |
| `go test ./... -cover -count=1` | 0 | All package coverage runs passed |
| `go run ./internal/traceability/cmd/tracecheck -section 1.6 -section 10.1 -section 10.2 -section 10.3 -section 10.4` | 0 | 5 assigned scopes, 29 acceptance cases |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | Default gate passed, 29 acceptance cases |
| `go run ./internal/traceability/cmd/tracecheck -section 10.999` | 1 | Expected refusal: nonexistent v0.5.0 section |
| Wrong Session Annotation schema mapping mutant | 1 | Expected-red production-entry failure |
| Post-mutant four-shape production-entry test | 0 | Restored source passed |
| `go vet ./...` | 0 | Clean |
| `go build ./...` | 0 | Build passed |
| `go mod verify` | 0 | All modules verified |
| `gofmt -d internal/canonicaljson/canonical.go internal/canonicaljson/canonical_test.go` plus empty-output assertion | 0 / 0 | Clean |
| `git diff --check` | 0 | Clean |

The package is deterministic and read-only. It does not mutate durable state,
so crash/recovery or idempotent durable-write evidence is not applicable. No
doctor result, migration publication, runtime capability, or provider/platform
availability claim was added.
