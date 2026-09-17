# TASK-260917-3lt2xv results — release-agnostic cataloggen check

## Outcome

`cataloggen` gained an `-adopted` mode that derives its metadata and lock
inputs from the code-level adopted release (`catalog.Adopted`), and
validation command 21 in `task-board.config.json` is now the release-agnostic
form. The explicit `-metadata`/`-contracts` form keeps working; mixing the
two is refused, as are a missing derived input and a lock whose source
release differs from adopted. Adopted output is byte-identical to the
explicit v0.6.0 invocation and to the committed generator output. No catalog
content, metadata, lock, or generated-output file changed.

## Candidate paths (uncommitted, 7 files)

- `internal/catalog/catalog.go` — `const Adopted Release = ReleaseV060`;
  `Current()` defined through `Adopted`.
- `internal/catalog/catalog_test.go` —
  `TestCurrentIsPinnedToAdoptedRelease`.
- `internal/catalog/cmd/cataloggen/main.go` — `-adopted`/`-root` flags,
  `runAdopted`, `checkAdoptedLockRelease`.
- `internal/catalog/cmd/cataloggen/main_test.go` — adopted positives,
  byte-identity, refusals, rewritten configured-gate test.
- `task-board.config.json` — command 21 is the `-adopted` form.
- `README.md` — regenerate block + tool row document both forms.
- `LOGBOOK.md` — newest-first entry, no capability claims.

## Production entry points

- Adopted mode: `run()` → `runAdopted()` in
  `internal/catalog/cmd/cataloggen/main.go`; lock-release gate
  `checkAdoptedLockRelease()` runs before `cataloggen.Generate`.
- Explicit mode: unchanged `run()` path, same `Generate` call.
- Adopted release: `catalog.Adopted`, consumed by `catalog.Current()`.

## AC coverage: 8 of 8 rows driven through the production entry

See `TASK-260917-3lt2xv_conformance-matrix.md` for the row-by-row map with
named tests, call sites, and mutants.

## Refusals (all through `run()`, all with narrowing mutants)

| Refusal | Message | Killing test | Mutant |
|---|---|---|---|
| `-adopted` + `-metadata` | `-adopted cannot be combined with -metadata or -contracts` | `TestRunAdoptedRefusesMixedAndMissingInputs/mixed_metadata` | N1 (admits metadata mixing only) |
| `-adopted` + `-contracts` | same | `.../mixed_contracts` | N2 (admits contracts mixing only) |
| missing `-output` | `-output is required with -adopted` | `.../missing_output` | N8 (admits no-check case only) |
| missing derived metadata | `read adopted metadata "<path>": ...` | `.../missing_metadata` | N3 (admits exactly IsNotExist) |
| missing derived lock | `read adopted contract lock "<path>": ...` | `.../missing_lock` | N4 (admits exactly IsNotExist) |
| lock release != adopted | `adopted contract lock release "v0.5.0" differs from adopted release "v0.6.0"` | `TestRunAdoptedRefusesLockReleaseMismatch` | N5 (admits exactly v0.5.0) |
| stale output under `-adopted -check` | `check catalog: generated catalog is stale...` | `TestRunAdoptedCheckRefusesStaleOutput` | N7 (length-only comparison) |
| byte-identity derivation | n/a (positive claim) | `TestRunAdoptedIsByteIdenticalToExplicit` | N6 (derives v0.5.0 metadata) |

Battery: 8/8 narrowing plants KILLED, each by exactly its named test (N5
additionally fails every adopted positive, as a release-gate weakening
must); harmless comment control C0 SURVIVED with 12/12 package tests green.
Harness + per-plant raw logs + subprocess exits in the evidence tarball
(`mutants/run-mutants.sh`, `mutants/logs/`, `mutants/summary.tsv`).

Token-preserving-mutant bound: no adopted-mode gate inspects source text
for a token — the lock gate JSON-decodes the release claim structurally,
the mixing gate reads parsed flag values, the staleness gate compares
bytes — so the token-preservation attack has no target here. N5 is its
behavioral equivalent: the v0.5.0 lock bytes still carry a well-formed
`source.release` claim and only the comparison is weakened.

## Validation (full configured suite, run locally)

- Commands 1–6: gofmt clean, `go build`, `go vet`, `go test ./...`
  (26/26 ok), `-race`, `-cover` — all exit 0.
- Commands 7–19: all thirteen 100x fuzz gates — exit 0.
- Command 18: `tracecheck` — exit 0.
- Command 21 (candidate): `-adopted` form — exit 0.
- Command 21 (trunk explicit v0.6.0 form): exit 0 on this tree.
- Commands 22–26: linux/windows cross-builds, JSON parse sweep,
  `task-board validate`, `git diff --check` — exit 0.
- `git diff --exit-code internal/catalog/catalog_gen.go
  internal/catalog/catalog.v0.6.0.json internal/specpin/` — empty.

## Handoff state

Candidate left UNCOMMITTED in the Story worktree for snapshot. No
`__pycache__`/`.pyc` artifacts. Evidence tarball
`TASK-260917-3lt2xv_producer-evidence.tar.gz` attached alongside this note
and the conformance matrix.
