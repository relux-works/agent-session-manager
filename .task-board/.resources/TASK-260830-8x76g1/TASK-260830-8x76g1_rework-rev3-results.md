# TASK-260830-8x76g1 rework revision 3 results

## Scope and pinned authority

- Task: `TASK-260830-8x76g1` (`fuzz-common-types-and-canonicalization`)
- Spawn run: `RUN-260831-9484d4`
- Pinned specification: `relux-works/agent-session-manager-spec` v0.5.0, commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`
- Scoped sections: 1.6, 10.1-10.4, and the immutable identity contribution from 17.3
- This revision addresses CR revision 2 findings F1-F3 as an audit of the shared validators and configured fuzz suite.

## Production behavior

Both public identity entries, `CalculateObjectIdentity` and `VerifyObjectIdentity`, route through `prepareObjectIdentity` and invoke `validateImmutableObjectShape` before digest calculation or attestation.

The composed call chain now validates every `extensions` map reached by `closed_shapes.go` through `validateMigrationExtensionObject -> validateExtensionsObject` before the special `works.relux.ax.migrated-from` payload is interpreted. It enforces:

- 3-253 lowercase ASCII reverse-DNS keys;
- at least one dot and dot-separated `[a-z][a-z0-9-]{0,62}` labels;
- at most 64 extension members;
- ExtensionValue nesting depth at most 4;
- complete JCS extension-object size at most 65,536 bytes; and
- the existing closed migration-provenance payload shape when that registered key is present.

`string[n..m]` checks in the scoped validator use Unicode character counts. The Transfer Manifest symlink target accepts 4,096 multibyte characters and refuses 4,097. Blob media type remains lowercase ASCII but uses the same character-count rule before its grammar check.

`TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget` discovers every repository `Fuzz*` function and requires the worktree validation configuration to contain exactly one matching command with `-fuzztime=100x -parallel=1`. `FuzzClosedIdentityShapeRefusal` is now configured and its committed corpus includes invalid Transfer Manifest and generic record extension keys.

## Negative evidence

The focused tests were run before the production fix and failed with exit code 1:

```text
go test ./internal/canonicaljson -run 'Test(IdentityExtensionKeysUseCompleteReverseDNSGrammar|IdentityExtensionsEnforceCountDepthAndCanonicalSize|ManifestStringBoundsCountUnicodeCharactersAtProductionEntries|ConfiguredValidationRunsEveryFuzzTargetWithFixedBudget)$' -count=1
```

Observed failures proved the real bypasses: `CalculateObjectIdentity` accepted every malformed reverse-DNS case and the Transfer Manifest `not_namespaced` key, the 4,096-character multibyte target was rejected as more than 4,096 bytes, extension count/depth/size were not gated, and the configured suite contained the closed-shape fuzz command zero times. The same focused command passed with exit code 0 after the shared fix.

Positive and negative boundary tests drive both public identity entries. They accept minimum and maximum valid reverse-DNS keys, 64 members, depth 4, exactly 65,536 canonical extension bytes, and 4,096 multibyte target characters. They refuse invalid key syntax at the generic record and Transfer Manifest extension points, 65 members, depth 5, 65,537 canonical bytes, and 4,097 target characters.

## Validation results

Every command below ran as a standalone process. Logs are in the attached revision 3 validation archive.

| Gate | Exit code | Evidence |
| --- | ---: | --- |
| `test -z "$(gofmt -l .)"` | 0 | `01-format.log` |
| `go build ./...` | 0 | `02-build.log` |
| `go vet ./...` | 0 | `03-vet.log` |
| `go test ./... -count=1 -v` | 0 | `04-full-tests.log` |
| `go test ./... -race -count=1` | 0 | `05-race.log` |
| `go test ./... -cover -count=1` | 0 | `06-coverage.log` |
| scalar fuzz, fixed `100x`, one worker | 0 | `07-scalar-fuzz.log` |
| JCS round-trip fuzz, fixed `100x`, one worker | 0 | `08-canonicalize-fuzz.log` |
| identity invariant fuzz, fixed `100x`, one worker | 0 | `09-identity-fuzz.log` |
| closed-shape refusal fuzz, fixed `100x`, one worker | 0 | `10-closed-shape-fuzz.log` |
| global tracecheck | 0 | `11-trace-global.log` |
| tracecheck for 1.6, 10.1-10.4, 17.3 | 0 | `12-trace-scoped.log` (`assigned_scopes=6`) |
| generated catalog check | 0 | `13-catalog-check.log` |
| Linux amd64 cross-build | 0 | `14-linux-build.log` |
| Windows amd64 cross-build | 0 | `15-windows-build.log` |
| tracked JSON parse validation | 0 | `16-json.log` |
| `task-board validate` | 0 | `17-board-validate.log` |
| `git diff --check` | 0 | `18-diff-check.log` |
| focused CR revision 2 regression tests | 0 | `19-focused-review-findings.log` |

Coverage: `internal/canonicaljson` 81.2%, `internal/scalar` 90.5%, `internal/traceability` 85.0%, and `internal/traceability/cmd/tracecheck` 87.5%.

`task-board validate` exited 0 while reporting 262 inherited `MISSING_ACTIVITY` diagnostics. `TASK-260830-8x76g1` is not listed in those diagnostics.

## Capability boundary

This is read-only identity validation. It does not mutate durable product state, publish migrations, atomically advance references, add `ax migrate`, add `ax doctor` output, or advertise a runtime capability. Crash/idempotency recovery evidence is therefore not applicable to this revision.
