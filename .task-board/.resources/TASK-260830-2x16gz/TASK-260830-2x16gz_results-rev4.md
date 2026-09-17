# TASK-260830-2x16gz results rev4 — CR2 rework (README figures + LOGBOOK annotation)

Role: developer (rework run, no product/test-code change).
Context: CR2 (`1ecd6439bc07fb5dfb8a3562788c339d28569396` on trunk
`fc67abdfc3888d6683b1eb89bc248e089ca6aca3`, Story tip `9636508`) was
reviewed by RUN-260917-948768 (verdict
`TASK-260830-2x16gz_review-verdict-rev2.md`): CHANGES REQUESTED with one
P2 (F1: README "Measured coverage of this repository" still states trunk
figures 60/44/11/49-of-511 while `tracecheck` on the tree prints
65/49/7/49-of-535) and two P3 notes (F2: LOGBOOK leaf entry rev1-time
figures; F3: `section:11.10.5` wording — explicitly NOT done here, handed
to the v0.7.0 registry re-derivation leaf per the rework brief).
Rev1/rev2/rev3 outcomes remain the implementation and refresh evidence;
this rev4 record covers only the F1 fix, the F2 annotation, the
tree-equality proof, and the full configured-suite rerun.

## 1. Edits (README.md: 4 lines; LOGBOOK.md: 1 inserted line)

Tool output on this tree (exact, `go run
./internal/traceability/cmd/tracecheck`):

```text
section coverage: bindings=65 full=2 partial=6 sliver=4 unevidenced=49 unmeasured=4 unowned=7 clauses_discharged=49/535
```

F1 fix — before -> after (line numbers in the edited file):

1. `README.md:3152` (fenced sample under "`tracecheck` prints the ratio
   it measured rather than a sentence about it:"):
   - before: `section coverage: bindings=60 full=2 partial=6 sliver=4 unevidenced=44 unmeasured=4 unowned=11 clauses_discharged=49/511`
   - after: `section coverage: bindings=65 full=2 partial=6 sliver=4 unevidenced=49 unmeasured=4 unowned=7 clauses_discharged=49/535`
   - verified byte-for-byte against the tool line with `cmp` (BYTE-IDENTICAL).
2. `README.md:3155`:
   - before: `Sixty section bindings discharge 49 of the 511 normative clauses their`
   - after: `Sixty-five section bindings discharge 49 of the 535 normative clauses their`
3. `README.md:3217`:
   - before: `and forty-four are \`unevidenced\`. Eleven sections are recorded unowned.`
   - after: `and forty-nine are \`unevidenced\`. Seven sections are recorded unowned.`
4. `README.md:3229`:
   - before: `Two admitted bindings out of sixty cover five clauses, and that is`
   - after: `Two admitted bindings out of sixty-five cover five clauses, and that is`

Adjacent-figure audit of the same subsection (checked, left unchanged —
each already matches the tool output): `full` 2 (`README.md:3156`),
`partial` 6 (`:3164-3165`), `sliver` 4 (`:3203-3204`), `unmeasured` 4
(`:3215`), admitted bindings 2 covering five clauses (6.2 at 1/1 + 2.4 at
4/4), and the 132/65/7 registry line (`:3050-3052`, refreshed in rev3).
No other figure in the subsection contradicts the tool output.

F2 annotation — one dated line appended inside the leaf's own LOGBOOK
entry (after the rev1 NOTE line, rev1 text untouched):

```text
- 2026-09-17 rev3 on `fc67abd`: pin `d3eca906…`, report 132 cases / 65 bindings / 7 unowned / 49 of 535 clauses; harness denominator 24 prior + 9 new; refresh-candidate completed.
```

F3 (registry wording for `section:11.10.5`) NOT done here per the rework
brief — it changes the registry digest and belongs to the v0.7.0
registry re-derivation leaf.

## 2. Tree-equality proof

- Baseline (before edits): working tree reproduced through a temporary
  index (`GIT_INDEX_FILE`, live index untouched) hashes to
  `1ecd6439bc07fb5dfb8a3562788c339d28569396` — exactly the CR2 candidate
  tree. `git diff 1ecd6439 --stat` over tracked paths empty.
- After edits: working tree `65dc05ea7b0be83d64a331d5225e02a4f0a11e01`;
  `git diff-tree --stat 1ecd6439 65dc05ea` is exactly:
  `LOGBOOK.md | 1 +` and `README.md | 8 ++++----` (5 insertions, 4
  deletions). Every other candidate path byte-identical (only two
  `diff-tree -r` rows, both `M` for README.md/LOGBOOK.md with the new
  blobs `1f8fc60…` / `43c6d21…`).
- `git diff HEAD -- README.md LOGBOOK.md` shows only the candidate delta
  plus these edits (LOGBOOK: leaf block + annotation = pure insertion;
  README: rev3 figures + these four lines).
- `task-board.config.json` unchanged (`git status`/`git diff HEAD` empty
  for it).
- Final `git status`: exactly the 13 candidate paths (11 modified + 2
  new `hostile_*_test.go`); no `__pycache__`, no `.pyc`, no staged
  changes. Candidate left UNCOMMITTED on
  `task-board/story/STORY-260830-1kiyj6` at `9636508`; no commit, no
  push, no board edit beyond status/evidence.

## 3. Full validation rerun on `9636508` + rev4 edits (all observed this run)

Configured suite = 27 commands from `task-board.config.json`
(`spawn.worktree_isolation.validation.commands`):

| # | Command | Result |
|---|---|---|
| 1 | `gofmt` clean check | pass |
| 2 | `go build ./...` | pass |
| 3 | `go vet ./...` | pass |
| 4 | `go test ./... -count=1 -v` | exit 0, 37 packages ok, 0 FAIL, 21523 `--- PASS` (identical count to rev3), 14 SKIP, zero top-level `^--- FAIL`, zero `^FAIL` |
| 5 | `go test ./... -race -count=1` | exit 0 in both bounded batches (5 Story pkgs + remaining 32, exact `./...` set reconciled 0 missing / 0 extra), 37 ok, 0 FAIL, 0 data races |
| 6 | `go test ./... -cover -count=1` | exit 0 (hostchannel 84.1, hosttrust 76.3, rpcwire 97.8, traceability 86.2, tracecheck 88.5 — all identical to rev3) |
| 7–20 | 14 fuzz smokes (rpcwire 1, scalar 1, canonicaljson 4, secconftest 8) | 14/14 pass |
| 21 | `tracecheck` | exit 0; `acceptance_cases=132 ... bindings=65 ... unowned=7 clauses_discharged=49/535` |
| 22 | `cataloggen -adopted ... -check` | exit 0, no generated drift |
| 23–24 | `GOOS=linux/windows go build ./...` | exit 0 both |
| 25 | JSON parse over tracked `*.json` | pass |
| 26 | `task-board validate` | exit 0; 191 pre-existing board-hygiene issues (identical count to rev3), 0 referencing this task/story |
| 27 | `git diff --check` | clean |

Own suites (this run):

- `go test ./internal/hostchannel -run 'TestHostile' -count=1 -v`: exit 0,
  54 `--- PASS`, 0 FAIL. `TestHostileRealCarrierOpenSSH` PASS (0.25 s —
  OpenSSH loopback lane executed, not skipped). The single SKIP is
  `TestHostileSSHHelperResponder`, the helper-process guard (by design).
- `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport`: PASS
  (0.10 s) — the F1 fix satisfies the derivation gate.
- `go test ./internal/cigate/... ./internal/traceability/... -count=1`:
  exit 0 (3 ok) — README claim gates green.
- Mutation battery: NOT rerun — no product/test-code byte changed
  (tree-equality proof in §2 covers all `.go`/`.py` candidate paths);
  rev2 reviewer reran 33/33 probes on the identical tree (31 killed,
  neutral passed, `client-cache` documented SURVIVED) plus 9 own plants.

## 4. Coverage restatement (unchanged from rev1/rev2/rev3)

- Hostile rows: 7 of 7 driven (`Dial`/`Serve`/`Call`).
- AC-HOST-001 rows: 14 of 15 fully driven; B8 partial (measured bound).
- HC-* families: 10 of 10. P3 notes: 2 of 2 closed.
- Stated bounds unchanged (native Tailscale SSH, non-host lanes, idle
  self-close measured, 11.10.5 policy reader by design, no `ax` CLI).
- Registry/report figures: 132 cases / 65 bindings / 7 unowned /
  49 of 535 clauses; pin `d3eca906…` (unchanged — no registry edit).
- No unsupported capability advertised. F3 wording stays as reviewed.

## 5. Handoff evidence (rev4)

- `TASK-260830-2x16gz_results-rev4.md` (this file)
- `TASK-260830-2x16gz_conformance-matrix-rev4.md` (rev3 matrix + rev4 addendum)
- `TASK-260830-2x16gz_producer-evidence-rev4.tar.gz` (real gzip: rev4 suite/
  race/cover/fuzz/tracecheck/cataloggen/cross-build/validate logs,
  hostile-focused log, tree-equality proof logs, before/after figure
  excerpts, rev4 matrix)
