# TASK-260830-2ya5le re-verify (rev3 successor run RUN-260923-6effa1)

Successor run after RUN-260830-2ya5le/RUN-260923-dc35bb finished the full
rev3 rework but its handoff failed at Change Request construction
("validation suite semaphore state lock timed out" — infrastructure,
not a candidate defect). This run verified the tree is intact and
re-ran the gates below firsthand on the exact tree, then handed off.
No source, test, or doc byte was changed by this run.

## Tree identity

- Scratch-`GIT_INDEX_FILE` tree OID:
  `9870fb42e454a1163b4fc6abd778210260cf84d9` — byte-equal to the
  rev3 candidate OID recorded in the prior run's notes and results.
- Base `0ca3e4c26e2b275212796657f785b9b450f6174e` == trunk; no
  refresh needed. Branch `task-board/story/STORY-260830-21bxa3`.
- Candidate left UNCOMMITTED; live index untouched (all comparisons
  through scratch indexes, since removed).
- Changed paths vs base: 4 modified landed files
  (`internal/clonebundle/decode.go`, `event.go`, `rawmanifest.go`,
  `internal/environ/decode.go`) + new `internal/clonefidelity/`
  (18 files incl. `utf8_reflection_test.go`).
- Attached rev3 artifacts read back and cross-checked: results,
  conformance matrix (§A–§E complete, §E ends at the control row,
  row 33 names all 8 narrowings, ratio 39 of 40 + 1 stated bound),
  evidence tar (gzip integrity exit 0, 440,811 bytes, 186/186
  `MANIFEST.sha256` hashes verified, 0 bad).

## Gates rerun by THIS run (real exit codes, no pipes)

| Check | Command | Exit |
| --- | --- | --- |
| build | `go build ./...` | 0 |
| vet | `go vet ./...` | 0 |
| windows vet | `GOOS=windows GOARCH=amd64 go vet ./...` | 0 |
| gofmt | `gofmt -l` on clonefidelity/clonebundle/environ (zero files) | 0 |
| whitespace | `git diff --check` | 0 |
| package tests | `go test ./internal/clonefidelity/ -count=1` | 0 |
| touched pkgs | `go test ./internal/clonebundle/ ./internal/environ/ -count=1` | 0 |
| new UTF-8 tests ×3 | `-run 'TestUTF8RefusalReflectionGrid\|TestRegistryPredicatesRefuseInvalidUTF8\|TestBuildUTF8ExtensionsNestedDepth' -count=3` | 0 |
| reflection grid ×1 | `-run TestUTF8RefusalReflectionGrid -v`: 33 derived members incl. the 3 reviewer sites, 363 subtests pass, 0 FAIL | 0 |
| mutant battery | `mutant_harness.py` full 71 rows, 71/71 `ok:`, 71 raw logs | 0 |
| full suite | `go test ./... -count=1`: 45 `ok`, 0 FAIL | 0 |
| tree stability | scratch tree OID after harness == `9870fb42…` (files restored); no `__pycache__`/`.pyc` | — |

## Mutant evidence rerun (rev3 rows + control)

Killer for every narrowing row: `TestUTF8RefusalReflectionGrid` run
alone (harness `-run=` pins it; raw log shows exit 1 with
`--- FAIL` at the exact member subtest — spot-checked
`N-build-utf8-source-class`). Zero survivors among narrowings.

| Mutant | What it narrows the gate to | Named test that fails | Verdict |
| --- | --- | --- | --- |
| N-build-utf8-source-class | admits exactly `ff fe` source_class past the pre-marshal gate (reviewer site 1/3) | TestUTF8RefusalReflectionGrid | KILLED |
| N-build-utf8-target-locator | admits exactly `ff fe` target_locator (reviewer site 2/3) | TestUTF8RefusalReflectionGrid | KILLED |
| N-build-utf8-forbid-reasons | admits exactly `ff fe` forbid_reasons item (reviewer site 3/3) | TestUTF8RefusalReflectionGrid | KILLED |
| N-build-utf8-source-item-key | admits exactly `ff fe` source_item_key (row shape) | TestUTF8RefusalReflectionGrid | KILLED |
| N-build-utf8-explanation | admits exactly `ff fe` explanation (row shape) | TestUTF8RefusalReflectionGrid | KILLED |
| N-build-utf8-reason-codes | admits exactly `ff fe` reason past the UTF-8 arm; kill via downstream vocabulary literal | TestUTF8RefusalReflectionGrid | KILLED |
| N-build-utf8-row-extensions | admits exactly `ff fe` row extension probe value (row shape) | TestUTF8RefusalReflectionGrid | KILLED |
| N-build-utf8-report-extensions | admits exactly `ff fe` report extension probe value (report shape) | TestUTF8RefusalReflectionGrid | KILLED |
| C-doc-comment (control) | comment-only edit; proves the harness observes outcomes | — (must pass) | SURVIVED |

Full battery: 70 KILLED + 1 control SURVIVED, harness exit 0.

## Reused evidence (unchanged identity)

Tree OID is unchanged since the prior run's green validation, so per
the reuse rule the following are accepted from the attached,
manifest-verified rev3 evidence tar rather than replayed: race gate
(45/45, zero data races), 17 fuzz gates, cover numbers, tracecheck,
cataloggen, importer outcome grid (zero moved), linux/windows builds,
`task-board validate`. CR construction revalidates the configured
suite itself.

## Handoff state

Ready for review. Checklist: all items verified true — checked.
Attached (this run): `TASK-260830-2ya5le_reverify-rev3.md` (this
file). Rev3 results, conformance matrix, and evidence tar remain the
attached producer artifacts.
