# TASK-260902-2xbu6m — reviewer verdict: ACCEPTED

Reviewer run `RUN-260902-4aa7aa`. Change Request `CR-TASK-260902-2xbu6m-1` revision 1.
Base `10aaa161fafc87d7bcb0e3e64096223662c6e656`, candidate tree `89ab34312d324a392d95474f036f72ad4375423f`,
branch `task-board/story/STORY-260902-2zsqwk`, head `af1ad4e`.

This is a reapply task. The three leaves were already reviewed to acceptance, so
review targets the *faithfulness of the transplant* and the *nine conflict
resolutions* — not a fourth pass over accepted content. Everything below was
recomputed by this reviewer from the four trees; nothing was taken from the
producer's notes.

## 1. Shape of the delivery — verified

| Property | Expected | Observed |
| --- | --- | --- |
| Commits past trunk | 1 | 1 (`af1ad4e`) |
| Parent | `10aaa16` | `10aaa16` |
| Tree | `89ab3431` | `89ab3431` |
| Signature | signed, author Ivan Oparin | `git verify-commit` → Good "git" signature for oparin@me.com, ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`, `%G?`=G |
| Worktree | clean | clean (only gitignored `.temp/`) |
| Paths touched vs trunk | 39 | 39 (25 A, 14 M, **0 D, 0 R**) — every other file in the tree is byte-equal to trunk |

Leaf file set (`48db30b..fc94e4a`) is exactly the 39 CR paths. Nothing extra was
carried in, nothing was dropped.

## 2. Byte-identity to the accepted trees — asserted, not assumed

Leaf set ∩ trunk-changed set = 8 files. The remaining **31 files are byte-identical
to the accepted leaf tree `fc94e4a`**:

```
git diff --quiet fc94e4a 89ab3431 -- <31 non-overlap paths>   # exit 0
```

For each of the 8 overlap files I ran a three-way expectation
(`normalize(diff leaf cand) == normalize(diff base trunk)` and
`normalize(diff trunk cand) == normalize(diff base leaf)`):

| File | Result |
| --- | --- |
| `testdata/constraint-enumeration.md` | exact clean merge, both directions match |
| `task-board.config.json` | exact clean merge, both directions match |
| `LOGBOOK.md` | add/add; leaf's 127 lines and trunk's 100 lines are **both full subsequences** of the merged 229 lines, **0 non-blank lines lost from either side**, 8 new lines = the 1810 entry |
| `README.md` | conflict hunks only (§3) — every other hunk matches trunk's contribution byte for byte |
| `ownership.v0.5.0.json` | conflict hunks only (§3) |
| `traceability.go` | derived digest only (§4) |
| `traceability_test.go` | derived count only (§5) |
| `cmd/tracecheck/main_test.go` | derived count only (§5) |

Every deviation from the accepted tree is accounted for by a named conflict.
None is unexplained.

## 3. The four structural hunks — union verified mechanically

**README Go-toolchain row.** Parsed the row into cells and tokenised:

- non-scoped commands: `set(cand) == set(leaf) ∪ set(trunk)` → **True**, 0 dropped, 0 invented
- the two per-side scoped `tracecheck` invocations merged into **one** command
- sections: leaf 19, trunk 8, **disjoint**, candidate 27 = exact union, no dups, ascending
- outputs cell identical on all three sides; description cell carries both sides' phrases

I then ran that exact 27-section command: `assigned_scopes=27`, exit 0.

**`ownership.v0.5.0.json`, both arrays.** Structural three-way merge check, not a
line diff:

- `acceptance_cases` keyed by `id`: 38 = 35∪ — keysets equal `leaf ∪ trunk`, **0 anomalies** across all 38: every `production` agrees, every `tests` list is the exact set union of both sides, no duplicates, no member-set drift.
- `ownership` keyed by `(kind, keys)`: base 35, leaf 44, trunk 44, candidate **53 = 35 + 9 + 9**. Ran a real three-way resolution per key (unchanged / one-side-changed / both-identical) — **0 deviations, 0 both-sides-changed conflicts**, 0 duplicate keys.
- The four `section_binding` keys that a naive set-union flags (`5.2`, `13.14.5`, `13.15`, `18.1`) are **leaf rebindings** from `ForRelease@catalog.go` to the specific validator, untouched by trunk. Taking the leaf side is the correct three-way answer, not a loss.
- Referential integrity: **0 dangling `acceptance_cases` references** from `ownership`.

## 4. The derived digest — recomputed, proven by attack

`reviewedOwnershipCanonicalSHA256` = `f126a4a00c64744586cc9d2bfc07e03a3923a93ee910970fe9183c9869759716`,
distinct from base `30403d58…`, leaf `f405dde1…`, and trunk `9f7737cb…`.

I did not take the producer's word. I recomputed it from the `traceability.go:286`
refusal by perturbing the constant in a scratch copy of the candidate tree — the
refusal reports the digest of the *merged registry as it stands*:

| Mutant | tracecheck | `go test ./internal/traceability/...` | Refusal reported |
| --- | --- | --- | --- |
| M1 flip last hex digit `…716` → `…717` | exit 1 | exit 1 | computed `f126a4a0…9716` |
| M2 **copy trunk's digest** `9f7737cb…` | exit 1 | exit 1 | computed `f126a4a0…9716` |
| M3 **copy leaf's digest** `f405dde1…` | exit 1 | exit 1 | computed `f126a4a0…9716` |

M2 and M3 are the ones that matter for this AC: **both sides' values are actively
refused against the merged registry**, so the pinned value cannot have been copied
from either. The value the gate itself computes is exactly what is pinned.

## 5. The five derived counts — arithmetic checked against ground truth

`acceptance_cases`: base 29 → leaf 32 (+3), trunk 35 (+6), candidate **38 = 29+3+6**,
and `len(registry.acceptance_cases)` on the merged tree is **38**. All three pins
(`traceability_test.go:AcceptanceCases`, both `acceptance_cases=` strings in
`cmd/tracecheck/main_test.go`) carry 38, and global tracecheck prints
`acceptance_cases=38`. The counts are not asserted — they are the registry.

README count sentence 30 → 38: I confirmed the producer's anomaly report. Trunk's
README said 30 while trunk's own registry carried 35, a five-behind drift, and
**nothing gates that sentence** (`grep` finds no Go reader of it; the only README
gate is `session_record_versions_test.go:545`, which pins three grammar *claim
strings*). Correcting it to 38 restores `README == len(acceptance_cases)`. Recorded
in LOGBOOK as pre-existing and not introduced here — reported, not absorbed. The
sentence remaining ungated is a pre-existing repository property, out of scope for
a reapply, and correctly left as a finding rather than silently fixed with a new gate.

## 6. Gates — all rerun by this reviewer at `af1ad4e`, all exit 0

| Gate | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `GOOS=linux` / `GOOS=windows` cross-build | exit 0 / exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l .` | empty |
| `go test ./... -count=1` | exit 0, 9/9 packages ok |
| `tracecheck` (global) | `contracts=60 normative_sections=36 acceptance_cases=38 fixtures=30 compatibility_contracts=55 assigned_scopes=0`, exit 0 |
| `tracecheck` 27-section scoped (the README row verbatim) | `assigned_scopes=27`, exit 0 |
| `cataloggen -check` | exit 0 |
| 5 seeded fuzz targets @ `-fuzztime=100x` | all PASS (`FuzzScalarProductionEntries`, `FuzzCanonicalizeRoundTrip`, `FuzzObjectIdentityRepresentationInvariant`, `FuzzClosedIdentityShapeRefusal`, `FuzzObservationEventRefusal`) |

## 7. Coverage — measured against both baselines, not asserted

Both baselines re-measured by this reviewer from `git archive` extractions of the
exact trees (`fc94e4a` accepted, `10aaa16` trunk):

| Package | Accepted `fc94e4a` | Trunk `10aaa16` | Candidate | Delta |
| --- | ---: | ---: | ---: | --- |
| canonicaljson | 97.2% | 87.2% | **97.2%** | = accepted, +10.0pp vs trunk |
| config | *(absent)* | 94.4% | **94.4%** | = trunk |
| catalog | 97.6% | 97.6% | 97.6% | = |
| catalog/cmd/cataloggen | 79.3% | 79.3% | 79.3% | = |
| cataloggen | 83.9% | 83.9% | 83.9% | = |
| scalar | 90.1% | 90.1% | 90.1% | = |
| specpin | 85.1% | 85.1% | 85.1% | = |
| traceability | 85.0% | 85.0% | 85.0% | = |
| traceability/cmd/tracecheck | 87.5% | 87.5% | 87.5% | = |

No regression against either baseline on any package.

## 8. Gates attacked, not read

Beyond M1–M3 above, on scratch copies of the candidate tree:

| Mutant | Class | Result |
| --- | --- | --- |
| M4 drop one **leaf-contributed** test entry from a union-merged acceptance case | narrowing the merge | tracecheck exit 1 (`e9647268…` ≠ reviewed), traceability tests exit 1 |
| M5 drop one **trunk-contributed** test entry from the same case | narrowing the merge | tracecheck exit 1 (`6fb3a2bb…` ≠ reviewed), traceability tests exit 1 |
| M6 rename a union-merged owner to a nonexistent declaration | forged owner | refused **by name before the digest is compared**: `declaration "TestThisDeclarationDoesNotExistAnywhere" is absent from …session_record_versions_test.go` |
| P1 **widen** `len(opaque) > 32` → `> 64` (`core_records.go:262`, `validateProviderIdentityRecord`) | widening | 11 failures incl. `TestEveryLiteralBoundaryIsDrivenAtTheValuesWhereItFlips/core_records.go\|validateProviderIdentityRecord\|len(opaque)_>_32_at_33` |
| P2 **widen** `len(value) > 128` → `> 256` (`requireTerminalBackendID`) | widening | 15 failures incl. the named boundary case at 129 |
| P3 **widen** Observation `result` vocabulary by one member (`+aborted`) | vocabulary widening | `TestEveryClosedVocabularyAdmitsExactlyItsPinnedSet` fails, `TestEveryProductionRefusalGuardIsExecuted` fails |
| P4 **narrow** Observation `phase` bound `1..128` → `1..64` | narrowing | 11 failures incl. `TestEveryCoreRecordDeclaredBoundAcceptsAtItsLimitAndRefusesPastIt/validateObservationEvent\|nullableBoundedString\|phase\|1..128` |
| R1 alter the README grammar claim `Section 1.6 reverse-DNS` | merged-README gate | `TestSessionRecordDeclaredGrammarRowsReachIdentityProductionEntries` fails at `session_record_versions_test.go:555` |

Every mutant reverted and the tree re-verified green after each. M4/M5 matter
specifically for this task: **both directions of the union merge are gated** — you
cannot quietly lose either side's contribution. P1–P4 are not delete-only mutants;
they widen and narrow live bounds and vocabularies, proving the reapplied
validators are wired and executed **on the new base**, not merely compiled. The
production call site for the registry gate is `internal/traceability.Verify`
refusing at `traceability.go:286`, driven by `cmd/tracecheck`.

## 9. Producer's reported facts — independently confirmed

Every load-bearing claim in the producer notes reproduces: the digest, all five
counts, the 27-section scoped run, coverage on both baselines, and the trunk README
anomaly. The two files the producer flagged as "deviating" from the accepted tree
(`constraint-enumeration.md`, `task-board.config.json`) are exact clean three-way
merges — an honest report of a real deviation from byte-identity, not a defect.

## Verdict

**ACCEPTED.** The transplant is faithful, every deviation from the accepted trees
is a named and correctly resolved conflict, the derived digest is provably a
recomputation rather than a copy, every gate exits 0, and coverage does not regress
against either baseline. Handed to the orchestrator for checkpoint/integration —
this reviewer supplied no `commit_ack`.
