# TASK-260830-236x9n developer rework revision 3

## Outcome

- Moved all v0.5.0 schema/version/self-field identity contracts into the reviewed generated catalog and derived the production `CalculateObjectIdentity` lookup from `catalog.Current().SelfIdentities`.
- Added the 14 omitted Terminal Backend, clone, migration, and supported-environment registry identity contracts identified by CR revision 2. The generated catalog now contains 40 identity definitions covering 46 exact schema/version rows.
- Preserved the schema-directed omission rule: registered ID-shaped reference members remain ordinary data. Callers still cannot select an omit field.
- Fixed `catalog.ForRelease` version merging for duplicate pinned Contract IDs. Without the fix, materialization-journal 3.0.0 replaced 2.0.0 and silently removed the managed-replica-marker identity.
- Updated README and traceability ownership without adding doctor, runtime capability, migration publication, or durable-state claims.
- Removed the unrelated root `.DS_Store` from the candidate. A recoverable copy is task-local under `.temp/` and is not part of the repository delta.

## Source evidence

- Normative source: local `agent-session-manager-spec` checkout at exact commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, tagged `v0.5.0`.
- The reviewed catalog metadata remains bound to SPEC.md SHA-256 `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a` through the existing spec pin.
- Generated metadata semantic digest: `7bbc5172fbd77216ef4888126787a91a2aabea63b8aa308a9a8ac2ccbc1e5bab`.
- Traceability ownership semantic digest: `503670f924aaa855f690d1fd2a9a0aee8078f1645fe1caad6a0986fe46359a76`.

## Validation with real exit codes

Green gates:

- `go generate ./internal/catalog` — exit 0.
- `go test ./internal/canonicaljson ./internal/catalog ./internal/cataloggen -count=1` — exit 0.
- `go test ./internal/canonicaljson -cover -count=1` — exit 0, 82.5% statements.
- `go test ./... -v -count=1` — exit 0.
- `go test ./... -cover -count=1` — exit 0; canonicaljson 82.5%, catalog 97.6%, cataloggen 83.9%, scalar 89.6%, specpin 85.1%, traceability 84.6%.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- `go mod verify` — exit 0.
- `gofmt -l internal` plus empty-output assertion — exit 0.
- `git diff --check` — exit 0.
- default `tracecheck` — exit 0, 60 contracts / 36 sections / 29 acceptance cases / 30 fixtures / 55 compatibility contracts.
- assigned `tracecheck -section 1.6 -section 10.1 -section 10.2 -section 10.3 -section 10.4` — exit 0 with 5 assigned scopes.

Expected-red evidence, reported as failures:

- Initial catalog generation after metadata expansion — exit 1 because the reviewed metadata digest had not yet been updated; this produced the new digest above.
- First focused run — exit 1 because `ForRelease` dropped materialization-journal 2.0.0; the subsequent merge fix and regression test made the rerun green.
- Missing-row mutant deleting `canonical-session@1.0.0` from the production table — exit 1; `CalculateObjectIdentity` refused the schema and the completeness test reported 45 implementation rows versus 46 generated rows.
- Wrong-mapping mutant changing Session Annotation from `annotation_id` to referenced `profile_id` — exit 1; the production-entry test observed the wrong field and digest.
- Traceability after renaming the completeness test — exit 1 until the reviewed ownership digest was explicitly updated.
- `tracecheck -section 10.999` — exit 1 with the expected nonexistent-section refusal.

## Handoff

The developer candidate is ready for review. Full logs, including every expected-red result above, are attached separately in the task-scoped validation archive.
