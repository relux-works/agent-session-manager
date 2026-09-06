# Mutation battery log — TASK-260830-3bkz0c

Harness: `.temp/TASK-260830-3bkz0c/mut.sh` (scoped to
`internal/environ`; restores from `/tmp/env-baseline` after every
run). Each mutant applied by its named python script with a
count==1 exact-match assertion (NOT-APPLIED if no textual
change). Gate: `go test ./internal/environ/ -count=1`.
Baseline tree sha: `.temp/TASK-260830-3bkz0c/pre-battery.sha`;
post-battery sha identical (`diff` empty → TREE_IDENTICAL).

## Narrowing mutants (gate stays present, admits exactly one member of the rejected class)

| Mutant | Narrowing | Named failing test | Result |
|---|---|---|---|
| N1 `n1_lowsurrogate.py`: drop the low-surrogate arm of the gate (admits lone lows, keeps refusing highs) | lone-low escapes | TestFrameAgreementAcrossFacades/bare_low_escape | KILLED |
| N2 `n2_bytelength.py`: StringLength counts bytes, not runes | 128-char/256-byte values | TestStringMeasureCountsRunes/environ_helper | KILLED |
| N3 `n3_tuplextensions.py`: tuple admits `extensions` member | one unknown member | TestTupleAgreementAcrossFacades/extensions_refused | KILLED |
| N4 `n4_caplength.py`: capabilities length check `<` instead of `!=` (admits 9-key map) | ninth capability | TestDecodeEnvironmentObservation/ninth_capability_refused | KILLED |
| N6 `n6_envunderscore.py`: env-id grammar admits `_` | underscore ids | TestSharedGrammarsAreOneLanguage | KILLED |
| T1 `t1_rawscan.py`: gate keeps its name and refusal texts but matches `\u` byte pairs without consuming `\\` pairs (raw-scan semantics) | escaped-backslash text | TestFrameAgreementAcrossFacades/escaped_backslash_{high,low,pair}_run2, _in_member_name, _nested (5 rows); TestSharedImplementationsAreCensused still PASSES under T1 | KILLED |

T1 is the token-preserving attack on the census source-text
gate: the symbol `HasLoneSurrogateEscape` stays defined, so a
grep-level gate would pass, while the behavioral suite fails on
exactly the escaped-backslash class. Census + behavior are
complementary by construction, not by assertion.

## Existence-only (not accepted as narrowing evidence)

| Mutant | What it proves | Result |
|---|---|---|
| D1 `d1_deletegate.py`: remove the surrogate-gate call (gate absent) | gate exists | KILLED (bare_high/low, high-followed-by-non-low, run3 rows) — existence only, excluded from the narrowing claim |

## Totals

Applied 7, killed 7. Narrowing kills 6 of 6 (N1, N2, N3, N4,
N6, T1). Census-only kills 0 (no census-only mutant run).
Over-applied (mutant fails to apply): 0. Survivors: none.
Every surviving-mutant bound is therefore vacuous; no bound
needs stating.
