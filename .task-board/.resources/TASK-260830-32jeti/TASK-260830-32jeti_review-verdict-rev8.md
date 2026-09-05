# Round-10 reviewer verdict — CR-TASK-260830-32jeti rev 8

**Verdict: CHANGES REQUESTED** → `to-dev`

**repeat-of: none.** Round 9 accepted rev 7. This finding could not exist on
that branch and could not exist on trunk; it exists only in the combination,
which is exactly what §6.2 refused integration to have someone look at.

Reviewed at candidate tree `39853bf3acf148b62307a382e29677b9ed3c6c23`, base
`2512f2087bcea43481f8541ee780f11daeececd4`. Live worktree content was rebuilt
into a scratch index and hashes to `39853bf…` exactly, so what follows was
measured on the snapshotted candidate, not on a neighbouring tree.

---

## F1 (blocking) — this Story's own new CI step fails on the merged tree

`.github/workflows/ci.yml` adds, in this Story's delta:

```yaml
- name: Vet Windows build-tagged sources
  run: GOOS=windows go vet ./...
```

On the merged tree that step exits **1**:

```
$ GOOS=windows go vet ./...
# github.com/relux-works/agent-session-manager/internal/terminalbackend_test
vet: internal/terminalbackend/terminalbackend_test.go:972:20: undefined: syscall.Mkfifo
REAL EXIT=1
```

Per-package sweep over all 16 packages: exactly one failure, named above. The
sweep is complete, not sampled — `for p in $(go list ./...)` with each exit
code checked.

Why neither side could see it:

| tree | has `internal/terminalbackend` | has the `GOOS=windows` vet step | `GOOS=windows go vet ./...` |
| --- | --- | --- | ---: |
| rev 7 (this Story on `57afcc6`) | no — package set verified by `git ls-tree`, 13 packages, no `terminalbackend` | yes | exit 0 (round-9 gate table) |
| trunk `2512f20` (sibling landed) | yes | no — `grep GOOS=windows` on trunk's ci.yml returns nothing | never run |
| **candidate `39853bf`** | **yes** | **yes** | **exit 1** |

So the Story's branch was green because the offending package did not exist
there, and the sibling never had to compile that test for Windows because the
gate did not exist on trunk. Landing this makes `main` CI red on the next push.

The repo already contains the correct pattern for exactly this hazard:
`internal/localstore/projection_unix_test.go` uses `syscall.Mkfifo` four times
and is excluded on Windows by its `_unix` filename constraint.
`terminalbackend_test.go` carries no such constraint. Producer's call which
side of the seam to fix; the reviewer does not prescribe the edit.

This trips the DoD row *"Relevant build/validation commands run after changes
and build not broken"* — the Story ships the validation command and the tree it
ships into does not pass it.

---

## G-A — the Go work is byte-identical to the tree accepted at rev 7: CONFIRMED

The board's rev-7 CR patch (`TASK-260830-32jeti_change-request_rev7.patch`) was
materialized from the board and is byte-identical to the restoration source
(`sha256 c675711de70a…` on both). Applying it to a scratch index seeded at
`57afcc6` reconstructs rev-7 tree `3817cef48c266f4158370d84db95936fa58dd4ab`.

```
$ git diff --stat 3817cef… 39853bf… -- internal/provider internal/provhost .github
(empty)
```

`internal/provider`, `internal/provhost` and `.github/workflows/ci.yml` are
**byte-identical** to the tree whose round-9 verdict recorded 177 mutant rows,
zero resurrections, and two declared bounds surviving name for name. That
verdict transfers. Not re-measured, per the brief.

AC-row carry re-verified against the merged tree rather than assumed: all 21
tests named in the round-9 AC table still resolve in `internal/provhost` /
`internal/provider`. **9 of 9 AC rows driven**, production call sites unchanged
(`DecodeManifest`, `DecodeProbe`, `RequireCapability`, `ProfileMapping`,
`DecodeQuiesceProof`, `CheckIdentity`, `DecodeSpawnPlan`, `IdempotencyKeyFor`,
`DecodeStatusOutcome`/`decodeStrictObject`).

`internal/traceability` **is not** identical — four files differ — so it was
re-measured from scratch. See below.

## G-B — the hand-merged files

### `LOGBOOK.md`: exact, both directions

Parsed to `(date, "HHMM — title")` blocks and compared byte-for-byte:

| check | result |
| --- | ---: |
| entries: base 45 + sibling 9 + this Story 12 | 66, matches candidate exactly |
| sibling's 9 new blocks byte-identical to trunk | 9/9, 0 changed |
| base's 45 blocks as trunk carries them | 45/45, 0 changed, 0 missing |
| this Story's 12 new blocks byte-identical to rev 7 | 12/12, 0 changed |
| blocks in candidate from no source | 0 |
| candidate restricted to trunk entries == trunk order | true |
| candidate restricted to rev-7 entries == rev-7 order | true |
| duplicate `## DATE` headers | none — the shared `2026-09-05` header folded correctly |

Nothing dropped, reordered or reworded. The one apparent ordering inversion is
a pre-existing entry whose title lacks an `HHMM` prefix, not a merge artifact.

### `README.md`: both deliberate resolutions verified as TRUE of the merged tree

**Claim 2 — ownership figures measured, not written.** `tracecheck` run against
the merged tree emits, byte-for-byte, the figures the README publishes:

```
traceability ok: contracts=60 normative_sections=36 acceptance_cases=81 fixtures=30 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=53 full=1 partial=3 sliver=1 unevidenced=45 unmeasured=3 unowned=2 clauses_discharged=17/428
```

Better than "present as text": the figures are machine-bound. Mutant M1 below
reddened `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport`
(`readme_figures_pin_test.go:144`) with *"README ownership paragraph at line
1986 publishes 81 executable acceptance cases; VerifyRepository measures 80"*.
The claim cannot drift silently.

Arithmetic of the merge is also consistent: acceptance cases 74 (base) → 77
(sibling, +3) / 78 (this Story, +4) → 81 (merged). Set-level three-way
comparison of `ownership.v0.5.0.json` across all four refs: expected union 81
acceptance cases / 58 ownership rows / 2 unowned / 5 sources — candidate matches
each count, **0 missing, 0 extra, 0 entries matching no source version, 0 trunk
modifications lost, 0 rev-7 modifications lost.**

**Claim 1 — "independent per-package implementations sharing no mechanism".**
Verified structurally, not read:

- three files, three packages: `internal/provider/refusal_inventory_test.go`
  (`package provider`, 279 ln), `internal/provhost/refusal_arm_inventory_test.go`
  (`package provhost`, 837 ln), `internal/terminalbackend/refusal_arm_inventory_test.go`
  (`package terminalbackend`, 1216 ln). Unexported test helpers cannot cross a
  Go package boundary, so "shares no code" is compiler-enforced, not asserted.
- each derives from **its own** source: provider and provhost both scan
  `os.Getwd()`; terminalbackend parses relative filenames in its own dir.
- distinct mechanisms, not one copied three times: `deriveRefusalInventory` →
  `refusalInventory{ScannedFiles,…}` with a four-constructor map, vs
  `refusalArmsIn` → arm-string set with a `refusalArmCensusFloor = 166`, vs
  `deriveRefusalArms(t) []refusalArm`.
- no shared inventory import. The three arm-inventory files import stdlib only,
  except provhost's two `internal/scalar` uses, which are `var zero
  scalar.UUIDv7` / `scalar.Timestamp` — zero values, not a table.

Precision worth recording, since it is adjacent and easy to misread as a
contradiction: `internal/provhost/profile_test.go` **does** import
`internal/provider`, for the §7.7 six-provider profile mapping. That is a
package dependency, not shared inventory machinery, and the README sentence is
scoped to inventories. The claim is true today; note that tracked advisory **A6**
proposes a shared AST-inventory test core, and the day A6 lands this sentence
becomes false and must move with it.

## G-C — the two packages in one repository

Refusal vocabularies compose rather than collide: `terminalbackend` mirrors
provhost's `provider_*` family as `terminal_backend_*`
(`terminal_backend_protocol_error`, `_process_failed`, `_manifest_probe_mismatch`,
`_restore_mismatch`), and the only shared codes are the common `axerror`
taxonomy (`invalid_config`, `integrity_failure`, `local_precondition_failed`,
`idempotency_mismatch`). No code carries two meanings. No new structural
advisory beyond the four already tracked as `STORY-260905-3t31e9`.

## Gates I ran myself on the merged tree

| gate | result |
| --- | ---: |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=linux go vet ./...` | exit 0 |
| **`GOOS=windows go vet ./...`** | **exit 1 — F1** |
| `gofmt -l internal/` | clean |
| `go test ./...` (16 packages) | all `ok` |
| `tracecheck -root .` | exit 0, figures above |

## Attack on what is new: the traceability gate, re-measured

The Go work is carried; the ownership registry and its pin are new, so they were
attacked rather than read. The pin (`reviewedOwnershipCanonicalSHA256`) alone
only stops a silent edit, so every mutant **also re-pinned the digest** — the
self-minter's move — to test whether the coverage gate holds independently.
Baseline calibrated GREEN; ownership file restored and sha-verified after the run.

| id | mutant | pin | result |
| --- | --- | --- | ---: |
| M0 | baseline, unmutated | — | GREEN (calibration) |
| M1 | drop one sibling acceptance case | re-pinned | **KILLED** — README figures pin + traceability |
| M2 | sibling row, test declaration → nonexistent symbol (**narrowing**, row kept) | re-pinned | **KILLED** — `VerifyAssignedSections`, tracecheck exit 1 |
| M3 | sibling row, production declaration → nonexistent symbol (**narrowing**) | re-pinned | **KILLED** — `VerifyAssignedSections`, tracecheck exit 1 |
| M4 | sibling row cross-claims this Story's production site (`terminal-lifecycle-conformance` → `provhost/quiesce.go DecodeQuiesceProof`) | re-pinned | SURVIVED |
| M5 | sibling row narrowed 12 named tests → 1 (row kept) | re-pinned | SURVIVED |
| M6 | reformat only, no semantic change, pin **not** updated | left stale | SURVIVED — correct, this is the documented tolerance |

M1–M3 kill through a re-pinned digest, so the gate resolves declarations against
real source rather than trusting the registry. That is the load-bearing result.

**M4 and M5 are advisory, not blocking, and not this leaf's to fix.** Neither is
a regression: the gate never claimed to bind a row to the package it describes,
nor to enforce a per-row test-count floor, and both properties predate this
Story. M4 is newly *reachable* only because two provider-side packages now
coexist — recording it here so it is not rediscovered. Neither matches A1/A3/A6/A7;
they belong with the `STORY-260905-3t31e9` consolidation, not with a rev-9 rework.

---

## What rework needs

Only F1. Everything else in this round passed. Specifically **do not** re-run the
177-row battery — `internal/provider` and `internal/provhost` are byte-identical
to the accepted tree and re-measuring them buys nothing.

Rework must show `GOOS=windows go vet ./...` at exit 0 on the merged tree, with
`go test ./...` still green (the FIFO assertion must still execute on unix — a
fix that deletes the regular-file guard's only load-bearing witness trades one
defect for a worse one).
