# TASK-260830-3bkz0c — RUN-260906-298e7a verification (no production change)

Successor run on the committed candidate `1296eccb0d19af3312b4a56ecb5dfc6f00c7d937`
(tree `292bc4242de751821d59ee1e3e68c9f1670b4634`). No production change by this
run — verification only. No new commit made, deliberately: the tree stays
byte-identical to the snapshotted candidate. Worktree clean at end of run.

Frozen packages untouched (byte-frozen leaves honored):
`git diff d5ad5f6..HEAD -- internal/sessadapter internal/dirnode internal/provider
internal/provhost --stat` is empty (exit 0). `git verify-commit HEAD` good (prior
run evidence; not re-run to avoid redundant signing I/O — HEAD unchanged).
PR #35 head == `1296ecc`, state OPEN (Change Request published; review/landing
is not this role's step).

## Gates run directly by this run (no pipes, real exit codes)

| Gate | Exit |
|---|---|
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `GOOS=windows go vet ./...` | 0 |
| `gofmt -l internal/` (empty output) | 0 |
| `go run ./internal/traceability/cmd/tracecheck` (contracts=60 normative_sections=36 acceptance_cases=94) | 0 |
| `go test ./internal/environ/ -count=1` | 0 |
| `go test ./internal/environ/ -race -count=1` | 0 |
| `go test ./internal/environ/ -cover -count=1` (76.1% of statements) | 0 |
| `go test ./... -count=1 -p 2` (19 packages ok) | 0 |

Bounds (stated, not hidden):
- Full-repo `-race` beyond `internal/environ` not run (exceeds one bounded
  shell call; same bound as the two prior recovery runs).
- Full-repo `-cover` not run; coverage gate is the touched package
  (`internal/environ`, 76.1%).
- Default-parallelism `go test ./... -count=1` not re-run by this run: the CR
  validator's command-4/18 failures on rev1/rev2 were root-caused by the prior
  run to the pre-existing canonicaljson wall-clock gate
  (`TestTransferManifestMaximumEntryGateIsLinear`, 2s hard budget, 3.5s under
  parallel load, 1.14s in isolation; no leaf causation possible — no file
  outside `internal/environ` imports environ, canonicaljson untouched by the
  diff). This run's `-p 2` full-suite pass on the identical tree is the
  deterministic re-proof. Recommendation stands: re-run the CR validation
  suite; no leaf code change is indicated.

## AC coverage: 7 of 7 rows driven (unchanged production, re-verified battery)

Standing claim from `TASK-260830-3bkz0c_boundary.md` (prior runs, re-confirmed
green by this run's `internal/environ` pass):

| # | AC row | Production call site | Named test |
|---|---|---|---|
| 1 | One environment library; census forbids new copies | `environ.DecodeStrictObject` (internal/environ/decode.go) | TestSharedImplementationsAreCensused |
| 2 | Shared fixtures drive every facade | `sessadapter.DecodeTuple`, `dirnode.CheckScanRequest`, `provhost.DecodeManifest/CheckIdentity/DecodeSpawnPlan`, `canonicaljson.Canonicalize/CalculateObjectIdentity` | all battery files via corpus_test.go builders |
| 3 | Shared identity | `provhost.CheckIdentity` vs `canonicaljson.CalculateObjectIdentity` | TestIdentityAgreementAcrossValidators, TestIdentityProviderEqualityIsCallerRule |
| 4 | Exact contract fixtures pass | `environ.DecodeTuple`, `environ.DecodeEnvironmentObservation` | valid rows in tuple/observation/identity/frame batteries |
| 5 | Negative/refusal cases with full rendering | same entries as 2–4 | every refusal row asserts the rule phrase in the refusal text |
| 6 | Crash/idempotency where durable state mutates | BOUND: pure validators, no store/cache (dirnode scan journal stays frozen leaf-2 scope) | TestEntriesAreDeterministic |
| 7 | No unsupported capability advertised | `environ.DecodeEnvironmentObservation` — exact-eight map, unknown/ninth-key refusal | ninth_capability_refused, unknown_capability_key, TestDirectoryCapabilityTableIsShared |

## Mutant battery re-executed by this run (all 7, harness `.temp/TASK-260830-3bkz0c/mut.sh`)

Baseline `/tmp/env-baseline` copied from the tree before the battery; tree
verified restored-identical after (pre/post sha diff empty → TREE_IDENTICAL,
`ENV_RESTORED_IDENTICAL`). Gate per mutant: `go test ./internal/environ/
-count=1`. Each mutant applied by its named python script (count==1 textual
change asserted by the harness; NOT-APPLIED would report, none did).

| Mutant | What it narrows the gate to | Named test that fails | Result |
|---|---|---|---|
| N1 `n1_lowsurrogate.py`: drop low-surrogate arm (admits lone lows, keeps refusing highs) | lone-low escapes | TestFrameAgreementAcrossFacades/bare_low_escape | KILLED |
| N2 `n2_bytelength.py`: StringLength counts bytes, not runes | 128-char/256-byte values | TestStringMeasureCountsRunes/environ_helper | KILLED |
| N3 `n3_tuplextensions.py`: tuple admits `extensions` member | one unknown member | TestTupleAgreementAcrossFacades/extensions_refused | KILLED |
| N4 `n4_caplength.py`: capabilities length check `!=` → `<` (admits 9-key map) | ninth capability | TestDecodeEnvironmentObservation/ninth_capability_refused | KILLED |
| N6 `n6_envunderscore.py`: env-id grammar admits `_` | underscore ids | TestSharedGrammarsAreOneLanguage | KILLED |
| T1 `t1_rawscan.py`: gate keeps name/texts, matches `\u` byte pairs without consuming `\\` pairs (raw-scan semantics) | escaped-backslash text | TestFrameAgreementAcrossFacades/escaped_backslash_high_run2, _low_run2, _pair_run2, _in_member_name, _nested (5 rows) | KILLED |
| D1 `d1_deletegate.py`: remove surrogate-gate call (gate absent) | existence only (excluded from narrowing claim) | TestFrameAgreementAcrossFacades/bare_high_escape, bare_low_escape, high_followed_by_non-low, escaped_backslash_high_run3 | KILLED |

Token-preserving attack (this run, additionally verified): under T1,
`go test ./internal/environ/ -count=1 -run
'TestSharedImplementationsAreCensused|TestSharedGrammarsAreOneLanguage'` exits
0 — the census still PASSES while the 5 escaped-backslash behavioral rows fail.
Census + behavior are complementary by construction, not by assertion.

Totals: applied 7, killed 7. Narrowing kills 6 of 6 (N1, N2, N3, N4, N6, T1).
Census-only kills 0. Over-applied (mutant fails to apply) 0. Survivors: none —
every survival bound is vacuous.

## Candidate tree OID

- `git rev-parse HEAD` = `1296eccb0d19af3312b4a56ecb5dfc6f00c7d937`
- `git rev-parse HEAD^{tree}` = `292bc4242de751821d59ee1e3e68c9f1670b4634`,
  byte-identical before and after this run's battery (`diff` empty).
- `git status --porcelain` empty at end of run.
