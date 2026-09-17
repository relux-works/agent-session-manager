# TASK-260917-3lt2xv conformance matrix — AC row to evidence

Production entries: `run()` / `runAdopted()` /
`checkAdoptedLockRelease()` in `internal/catalog/cmd/cataloggen/main.go`;
`catalog.Adopted` / `catalog.Current()` in `internal/catalog/catalog.go`.
All tests below call `run()` (the production CLI entry behind `main`)
except the catalog pin test, which calls the production `Current()` /
`ForRelease()` entries.

| # | AC row | Test (committed) | Production call site | Mutant |
|---|---|---|---|---|
| 1 | `-adopted -check` exits 0 on trunk | `TestRunAdoptedCheckIsGreen`; `TestConfiguredCRCatalogGateConsumesAdoptedAuthority` (configured command 21, rooted) | `run()` → `runAdopted()` → `Generate` → `checkUnchanged` | N6 kills both (derivation weakened) |
| 2 | byte-identical to explicit v0.6.0 invocation | `TestRunAdoptedIsByteIdenticalToExplicit` (adopted vs explicit vs committed) | `run()` both forms → same `Generate` call | N6 (derives v0.5.0 metadata) |
| 3 | explicit `-metadata`/`-contracts` still work | `TestRunGeneratesCommittedCatalogAndSupportsIdenticalRetry` (pre-existing) | `run()` explicit path | unchanged code path |
| 4 | mixing `-adopted` with explicit flags refused | `TestRunAdoptedRefusesMixedAndMissingInputs/mixed_metadata`, `.../mixed_contracts` | `runAdopted()` mixing guard | N1, N2 (each admits one side) |
| 5 | missing metadata/lock for current release refused, distinct messages | `.../missing_metadata`, `.../missing_lock` (messages name derived paths) | `runAdopted()` adopted reads | N3, N4 (admit exactly IsNotExist) |
| 6 | lock whose release != adopted refused, distinct message, pre-generation | `TestRunAdoptedRefusesLockReleaseMismatch` (v0.5.0 lock under adopted name; asserts no `generate catalog` wrap, no output written) | `checkAdoptedLockRelease()` before `Generate` | N5 (admits exactly v0.5.0) |
| 7 | command 21 is the `-adopted` form; every other command unchanged | `TestConfiguredCRCatalogGateConsumesAdoptedAuthority` (7-field shape pin + green run + stale-authority refusal); config diff is the single line | configured `go run` argv → `run()` | N5 kills the green run; shape pin fails on any explicit-form revert |
| 8 | stale output under `-adopted -check` refused without rewrite | `TestRunAdoptedCheckRefusesStaleOutput` (same-length stale fixture) | `runAdopted()` → `checkUnchanged` | N7 (length-only comparison; pre-existing different-length stale test stays green under it) |

Supplemental (not an AC row): `TestCurrentIsPinnedToAdoptedRelease`
pins `Current().Release == Adopted` through the production catalog entry;
`-output is required with -adopted` refusal covered by
`.../missing_output`, mutant N8.

Coverage: **8 of 8 AC rows driven** through the production entry.
Mutants: 8/8 narrowing KILLED + harmless control SURVIVED
(`mutants/summary.tsv`, raw logs in `mutants/logs/`).
