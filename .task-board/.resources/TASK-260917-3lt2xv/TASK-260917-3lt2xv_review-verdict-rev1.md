# Review verdict — TASK-260917-3lt2xv rev1 (CR-TASK-260917-3lt2xv-1)

## Verdict: ACCEPT

Revision 1 of `CR-TASK-260917-3lt2xv-1` (candidate tree `83e6eb6f`, base `62d44630`, 7 paths) is accepted. All 8 AC rows were driven independently through the production entry points; every refusal and the byte-identity claim were attacked with narrowing mutants and killed. One P3 coverage observation (no behavior defect) is recorded below; it does not block acceptance.

## Method and non-mutation guarantee

All destructive probes ran in filesystem-isolated copies (`/tmp/iso-cand`, temp roots under `/tmp/rev_*`, `/tmp/fwd_fake`); the live Story worktree was never edited, committed, or re-based by this run. Final live-tree state: `HEAD` still `62d446304391af187c417a88a2e14012b467956e`, `git status --short` shows exactly the 7 candidate paths modified, no untracked artifacts (`__pycache__`/`.pyc` absent).

## 1. Contract — verified with own instruments (all on live tree unless noted)

- `catalog.Adopted` is the single declaration (`internal/catalog/catalog.go:23`, `const Adopted Release = ReleaseV060`); `Current()` is defined through it (`catalog.go:195`, `ForRelease(Adopted)`). Only other `Current()` callers (`internal/config/validation.go`, `internal/canonicaljson/*`, `internal/cliresult/types.go`, `internal/axerror/registry.go`) are read-only consumers; no call-site changes needed.
- Divergence plant (isolated copy): `Current()` → `ForRelease(ReleaseV050)` while `Adopted` stays v0.6.0 ⇒ `TestCurrentIsPinnedToAdoptedRelease` RED, exit 1 (`Current().Release = "v0.5.0", want Adopted "v0.6.0"`). Pin test is live. Raw log: `divergence.log`.
- `-adopted -output internal/catalog/catalog_gen.go -check` exit 0; trunk explicit `‑metadata …v0.6.0.json ‑contracts …v0.6.0.lock.json … ‑check` exit 0 on the candidate tree.
- Adopted generation vs explicit v0.6.0 generation vs committed `catalog_gen.go`: `cmp` byte-identical all three ways.
- Refusals, each exit 1 with the distinct claimed message and no output file written: mixed `-metadata` / mixed `-contracts` (`-adopted cannot be combined with -metadata or -contracts`); missing `-output` (`-output is required with -adopted`); missing derived metadata (`read adopted metadata ".../catalog.v0.6.0.json"`); missing derived lock (`read adopted contract lock ".../v0.6.0.lock.json"`); v0.5.0 lock filed under the adopted name (`adopted contract lock release "v0.5.0" differs from adopted release "v0.6.0"`, no `generate catalog` wrap).
- Stale same-length output under `-adopted -check`: refused with `check catalog: generated catalog is stale...`, file not rewritten; explicit form behaves identically.

## 2. Forward proof (isolated copy)

- With `Adopted` planted to `Release("v9.9.9")` and the v0.6.0 pair copied under `catalog.v9.9.9.json` / `v9.9.9.lock.json`, `-adopted` derived the `v9.9.9` paths (found the files) and refused with `adopted contract lock release "v0.6.0" differs from adopted release "v9.9.9"`, writing no output. Derivation and lock gate both follow the declaration. Raw log: `forward-fake.log`.
- With `Adopted` planted to `ReleaseV050`, `-adopted` fails while explicit v0.6.0 stays green — adopted follows the constant, explicit path unaffected.
- Consequence for the consumer (TASK-260916-n9r71p): set `Adopted = ReleaseV070`, ship matching `catalog.v0.7.0.json` / `v0.7.0.lock.json`, keep command 21 untouched; construction then passes because the gate derives v0.7.0 inputs.

## 3. Config

JSON walk base-vs-candidate `task-board.config.json`: exactly 1 differing string, `spawn.worktree_isolation.validation.commands[20]` (command 21), explicit v0.6.0 form → `-adopted` form. No model/signing/gate/trigger field touched. Trunk suite and candidate suite differ only in that command. `task-board validate` (command 25) exit 0 on the tree.

## 4. No content change

`git diff --exit-code 62d44630 HEAD -- internal/catalog/catalog_gen.go internal/catalog/catalog.v0.6.0.json internal/catalog/catalog.v0.5.0.json internal/specpin/` exit 0 (empty).

## 5. Mutation

- Producer harness rerun in an isolated copy: 8/8 narrowing plants KILLED, harmless control C0 SURVIVED, harness exit 0 (`producer-rerun.log`, per-plant logs under `producer-logs/`).
- Each of the 8 KILLED rerun twice more with targeted `-run` masks: 16/16 exit 1, deterministic (`rerun-twice.log`, `rerun-*` logs). Denominator: 8 plants × 2 = 16 kills.
- Own narrowing mutants (isolated copy, full-package behavioral suite per plant):
  - R-A lock derivation pinned to `v0.5.0.lock.json` instead of `Adopted` ⇒ KILLED by `TestRunAdoptedCheckIsGreen` (exit 1).
  - R-B mismatch refusal swallowed for non-empty locks ⇒ KILLED by `TestRunAdoptedRefusesLockReleaseMismatch` (exit 1).
  - R-C mixed-flag guard weakened to `&& !check` ⇒ SURVIVED the committed suite (no committed test covers mixed+`-check`), but the custom behavioral probe proves the gate: pristine mixed+`-check` refused exit 1 with the mixing message; mutant admits it exit 0. See P3 below.
  - C-R harmless comment control ⇒ SURVIVED, exit 0.
- Token-preservation bound: no adopted-mode gate inspects source text — `grep Contains|HasPrefix|Index(` on `main.go` exit 1 (no hits). Lock gate JSON-decodes structurally, mixing reads parsed flags, staleness compares bytes. R-B is the behavioral equivalent of the token-preserving attack (well-formed `source.release` claim preserved, only the comparison weakened) and is killed.

## 6. Docs and hygiene

- README regenerate block + tool row document `-adopted` and the explicit equivalent; LOGBOOK entry is newest-first, factual, with no capability/availability claims (catalog comment explicitly disclaims availability).
- `gofmt` clean, `go vet` (full `./...`, command 3) exit 0, `go test ./internal/catalog/... ./internal/cataloggen/...` exit 0 incl. `-race`.
- Full 26-command suite on the exact tree, rerun by this review: commands 1–4, 6–26 exit 0 (command 4: 26 ok; command 6: 26 ok; all 13 fuzz gates exit 0; command 21 adopted exit 0). Command 5 (full `-race`): accepted from the attached producer `validation-race.log` (26 ok) plus this review's scoped `-race` rerun green on all touched packages — stated explicitly per headless-run rules.

## AC coverage: 8 of 8 rows driven through the production entry

Production sites: `run()` → `runAdopted()` → `checkAdoptedLockRelease()` → `cataloggen.Generate` (`internal/catalog/cmd/cataloggen/main.go`); `catalog.Adopted` / `catalog.Current()` (`internal/catalog/catalog.go`). All tests call `run()` except the pin test.

| # | AC row | Named committed test | Call site | Mutant |
|---|---|---|---|---|
| 1 | `-adopted -check` exits 0 | `TestRunAdoptedCheckIsGreen`, `TestConfiguredCRCatalogGateConsumesAdoptedAuthority` | `run()` → `runAdopted()` → `Generate` → `checkUnchanged` | N6 kills both; own R-A kills the former |
| 2 | byte-identical to explicit v0.6.0 | `TestRunAdoptedIsByteIdenticalToExplicit` (adopted vs explicit vs committed) | `run()` both forms → same `Generate` | N6 |
| 3 | explicit flags still work | `TestRunGeneratesCommittedCatalogAndSupportsIdenticalRetry` (pre-existing) | `run()` explicit path | unchanged path (forward proof: explicit green under planted Adopted) |
| 4 | mixed flags refused | `.../mixed_metadata`, `.../mixed_contracts` | `runAdopted()` mixing guard | N1, N2 |
| 5 | missing metadata/lock refused, distinct path-naming messages | `.../missing_metadata`, `.../missing_lock` | `runAdopted()` adopted reads | N3, N4 |
| 6 | lock release ≠ adopted refused pre-generation | `TestRunAdoptedRefusesLockReleaseMismatch` (asserts no `generate catalog` wrap, no output) | `checkAdoptedLockRelease()` before `Generate` | N5; own R-B |
| 7 | command 21 is the `-adopted` form, rest unchanged | `TestConfiguredCRCatalogGateConsumesAdoptedAuthority` (7-field shape pin + green run + stale-authority refusal) | configured argv → `run()` | N5 kills green run; shape pin fails any explicit revert |
| 8 | stale output under `-adopted -check` refused without rewrite | `TestRunAdoptedCheckRefusesStaleOutput` (same-length stale) | `runAdopted()` → `checkUnchanged` | N7 |

Supplemental: `TestCurrentIsPinnedToAdoptedRelease` (divergence RED proven above); `.../missing_output` covers `-output is required with -adopted` (N8).

## Integrate-time forecast: `validation_suite_changed` will not refuse this CR

The guard is `selectReviewedValidationSuite` in `tools/board-cli/internal/integration/reviewed_suite.go` (error `ErrorValidationSuiteChanged`, `internal/integration/errors.go:90`; enforced around `internal/integration/validation.go:70`). When evidence suite SHA differs from the configured SHA it does not refuse outright: for a reviewed change of the config source itself it admits the new suite iff the CR is accepted with exit-0 validation, the config path is inside the control root and in `ChangedPaths`, the evidence-tree config blob equals the candidate-tree config blob, base differs from candidate, the base-blob suite SHA equals the control side's, and the evidence SHA equals the candidate-blob suite SHA with the same profile — then it adopts the new commands/SHA. This CR satisfies that shape (config path in the 7 changed paths; evidence produced by the candidate suite, exit 0, rerun green above). Blocker condition to preserve: no further edits to `task-board.config.json` after the evidence run, or the evidence-blob ≠ candidate-blob check refuses.

## P3 observation (non-blocking, no behavior defect)

Committed tests cover mixed flags without `-check` only; the R-C probe shows mixed+`-check` is refused in production (correct) but untested. Suggested follow-up (not this CR): extend `TestRunAdoptedRefusesMixedAndMissingInputs` with a mixed+`-check` case asserting the mixing refusal. Production behavior already meets the AC.

## Rerun-vs-accepted statement

Reran myself on the exact candidate tree: adopted/explicit `-check`, byte-identity `cmp` ×2, all 6 refusal classes via CLI (each exit 1, no output written), stale-pair comparison, divergence RED, forward/fake-release probes, config JSON walk, no-content diff, producer battery (8 KILLED + control), 16 targeted re-kills, own R-A/R-B KILLED + R-C probe pair + control, token-search grep, gofmt/vet/scoped tests ± race, validation commands 1–4 and 6–26. Accepted from attached evidence without rerun: full-suite `-race` (command 5; producer `validation-race.log` shows 26 ok) — scoped `-race` on touched packages rerun green here.
