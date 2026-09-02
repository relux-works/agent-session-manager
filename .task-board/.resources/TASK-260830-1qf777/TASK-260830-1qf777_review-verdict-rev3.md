# TASK-260830-1qf777 review verdict — CR revision 3

**Verdict: changes requested (`to-dev`).**

Reviewed candidate tree `9700e40f4bc9787d495864cd97654b2a8864121b` against base
`48db30b59e5e1bbc5e0cf73ec2e0e0eec3d215d1`. The worktree was verified to hash to
exactly that candidate tree (`git write-tree` on a temporary index).

## What was verified green

| Check | Result |
| --- | --- |
| `go build ./...`, `go vet ./...` | clean |
| `go test ./... -count=1` | all 9 packages ok |
| `go test ./internal/config -cover -count=1` | ok, 92.3% of statements |
| `gofmt -l internal/` | clean |
| `tracecheck -section 3.2 6.1 6.2 6.3 6.4 6.5 17.1 17.2 17.4` | `traceability ok: contracts=60 normative_sections=36 acceptance_cases=35 fixtures=30 compatibility_contracts=55 assigned_scopes=9` |
| Ownership pin `reviewedOwnershipCanonicalSHA256` | re-read the ownership delta by hand; every new acceptance case names a production declaration that exists and tests that exist. The self-minted hash update is accepted on that basis. |

The design is genuinely good and most of it is attacked rather than read:

- Ordering matches SPEC §6.4 — validate source through the real `Load`, publish an
  owner-only versioned backup, fsync the directory, then write/rename/fsync the
  replacement. Failure at each durable step leaves the selected file byte-identical
  and a clean retry completes.
- The read-seam / mutating-seam symlink asymmetry is pinned in both directions at
  the real `LoadOS` / `MigrateOS` entries, and the refusal does not echo a
  machine-local path.
- The derived refusal inventory (`refusal_inventory_test.go`) is a real
  self-proving gate: it enumerates the package's error types from source, requires
  exactly one instrumented constructor per type, and reports stray literals.
- The producer's own 29-mutant sweep (`mutant-sweep-r3.log`) is all RED, and I
  re-ran an independent 20-mutant sweep of my own. 12 of mine also died,
  including `.bak.<version>` naming, backup-mode widening, the idempotent no-op,
  the schema-id envelope check, the mutating-seam symlink revert, the
  staging-file cleanup on both failure paths, and widening the migrated file's
  mode to world-readable.

Reviewer sweep log: `.temp/TASK-260830-1qf777/review-r3/review-mutants-r3.log`,
harness `.temp/TASK-260830-1qf777/review-r3/mutants.py`.

## Findings — surviving mutants on this task's own acceptance criteria

### F1 (blocking) — the read-only downgrade bound is not proven for the one-major step

SPEC §6.5 states literally: *"A v1/v2 binary opening v3 is read-only diagnostic."*
§6.4 states the same for v1. The suite only assesses `(document v3, reader 1.0.0)`
and `(document v1, reader 3.0.0)`. Narrowing the gate in `migration.go`

```go
-       if compareSemver(source, reader) > 0 {
+       if source.major > reader.major+1 {
```

leaves `go test ./internal/config -count=1` **green**. Under that mutant
`AssessCompatibility(v3document, Version2)` returns `Mode = "compatible"`, i.e.
the v2 reader is told it may write — the exact case the specification names.
Confirmed with a probe: the assertion passes on the clean tree and fails on the
mutant with `Mode = "compatible", want read-only-diagnostic`.

This is a narrowing mutant surviving on the AC's own words ("read-only downgrade
behavior"). Add the `(v3, reader 2.0.0)` and `(v2, reader 1.0.0)` cases, and a
paired positive control at equality so the gate stays a comparison rather than a
blanket.

### F2 (blocking) — migration never carries a non-default member, so silent value loss is invisible

Every migration fixture is `minimalValidConfigVersion(...)` — `schema`,
`schema_version`, `host_id`, `host_name`, `platform` — plus at most
`[terminal] backend`. No member with a non-default value ever crosses a
migration in any test. Three delete-only mutants therefore survive:

| Mutant | Effect | Suite |
| --- | --- | --- |
| `encodeVersion2` drops `SafeBoundaryTimeoutSeconds` | v1→v2 loses the operator's value | SURVIVED |
| `encodeVersion2` drops `GracefulStopTimeoutSeconds` | v1→v2 loses the operator's value | SURVIVED |
| `currentWire` drops `SafeBoundaryTimeoutSeconds` | v1/v2→v3 loses the operator's value | SURVIVED |

Confirmed concretely: a v1 document with `safe_boundary_timeout_seconds = 42`
migrated to 2.0.0 reads back as `42` on the clean tree and as `300` (the schema
default) under the first mutant, with every shipped test still passing.

SPEC §6.4 — *"Configuration 2.0.0 retains all Configuration 1 members"* — and
§6.5 — *"does not alter old bytes"* — are the contract being claimed here, and
nothing reddens when a durable, operator-visible mutation silently discards a
member. This is the highest-consequence gap in the change: the operation writes
to durable state.

Add at least one migration case whose source sets every v1/v2 member this package
decodes to a value distinct from its default, and assert the loaded post-migration
`Configuration` equals the loaded pre-migration `Configuration` except for
`SourceVersion`, the mapped `Terminal.BackendID`, and the recorded
`GeneratedSummaryUpgradeChoice`. A whole-struct comparison (rather than a
field list) is what makes a future added member fail closed.

### F3 (blocking, small) — the spec-mandated temp-file fsync has no test

SPEC §6.4: *"writes a complete v2 file to a same-directory temporary file,
**fsyncs it and the directory**, and atomically replaces the original."* README
repeats the claim ("writes and fsyncs a same-directory temporary file"). Deleting

```go
-       if err := file.Sync(); err != nil {
-               return clean(err)
-       }
```

from `writeTempFile` leaves the suite green — both the backup and the replacement
are then renamed without their contents ever being made durable. The *directory*
fsync is well covered (injected failure, recovery, and retry all assert), so this
is an asymmetry rather than an untestable property: `faultMigrationFileSystem.CreateTemp`
returns the raw `*os.File`, so a counting/failing `migrationFile` wrapper is a
small addition that closes it.

## Non-blocking observations (fix if convenient, not gating)

1. `encodeVersion2`'s round-trip `Decode(output.Bytes(), context)` re-validation can
   be deleted with the suite still green. It is defense in depth, but it is also
   the one refusal in this change that returns a bare `errors.Join(ErrConfigEncode, err)`
   rather than an instrumented constructor, so the derived inventory cannot see
   it either and the `encode target` subsumption comment does not name it.
2. `TestRefusalSubsumptionInventoryIsPinned` compares only `len(got) != len(want)`.
   The `want` map's keys and justifications are never read, so the doc comment
   ("it must be justified here") promises more than the assertion enforces.
   Additions still redden via the count and the derived inventory covers the
   substantive risk, so this is a robustness nit — but keying the pin on the
   `file: operation` string would make it mean what it says.
3. `writeAll`'s `written == 0 → io.ErrShortWrite` guard can be deleted with the
   suite green (defensive; low value).
4. Tightening the replacement file's mode to `0o600` survives, but the
   security-relevant direction — widening it with `|0o044` — reddens at
   `TestMigrateOSPerformsOneDurableMigrationAtTheRealProcessEntry`. Not an issue.
5. Admitting a non-canonical semver core (`01.0.0`) survives, but is behaviorally
   equivalent because `knownConfigVersion` gates the same values. Not an issue.
6. `MigrationResult.BackupPath` is populated before `replaceDurably` runs, so a
   failed migration returns a path that may not exist. Harmless while callers
   check `err` first; worth clearing on the error paths.

## Routing

`to-dev` with F1–F3. No production-code defect was found: the shipped behavior is
correct on every probe. What is missing is the negative and preservation evidence
that would keep it correct — three mutants that a reviewer could write in an
afternoon change durable operator state and no test notices.
