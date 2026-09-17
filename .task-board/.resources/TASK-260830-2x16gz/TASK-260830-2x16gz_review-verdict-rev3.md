# Review verdict — CR-TASK-260830-2x16gz-3 revision 3 (ACCEPTED)

Reviewer run: RUN-260917-f6f23d (reviewer, claude-opus-5 max).
Candidate tree `65dc05ea7b0be83d64a331d5225e02a4f0a11e01` over trunk base
`fc67abdfc3888d6683b1eb89bc248e089ca6aca3`, Story branch tip `9636508`
(z1yxg9 replay), 119 changed paths (whole-Story delta). Prior revision: CR2
(tree `1ecd6439…`) refused by RUN-260917-948768 for one P2 (README
"Measured coverage" figures were trunk's) and one P3 (LOGBOOK annotation).

This review is SCOPED, as briefed, to `README.md` and `LOGBOOK.md`: the CR2
deep review (reconciliation, replays, digest, hostile matrix, mutation
battery) was not redone because the product tree did not change — proven by
path-set equality in §1, not assumed.

Method: isolated immutable probes only. The candidate tree OID was
reproduced from the working tree through a temporary index file
(`GIT_INDEX_FILE`); the live index stayed at `ddacb0cb`, HEAD at `9636508`,
`git status` at the 13 candidate paths after every step. README plants ran in
a `/tmp` `git archive` copy of the candidate tree (deleted). No product edit,
commit, checkpoint or integration by this run. `PYTHONDONTWRITEBYTECODE=1`.
Evidence archive: `TASK-260830-2x16gz_review-evidence-rev3.tar.gz` (real
gzip; `run-notes.md` inside indexes every log named below).

## Verdict: ACCEPT — no P1, no P2; two P3 notes for follow-up leaves

## 1. Path-set equality against CR2 — exactly README.md + LOGBOOK.md

- Working tree → temp-index `git write-tree` = `65dc05ea…` (the CR record's
  candidate tree). `git diff-tree -r --name-status 1ecd6439 65dc05ea` is
  exactly `M LOGBOOK.md`, `M README.md`; `--stat`: `LOGBOOK.md | 1 +`,
  `README.md | 8 ++++----` (5 insertions, 4 deletions). Every other path —
  all of `internal/hostchannel`, `hosttrust`, `rpcwire`, `sshtransport`,
  `config`, `peeridentity`, `axerror`, `secprim`, `traceability` (registry,
  pin, tests), `task-board.config.json`, `go.mod`/`go.sum` — is byte-identical
  to the CR2 candidate the rev2 review verified.
- Immutable CR bytes: the attached patch hashes to
  `baac345629f2…78791f` (matches the CR record); applied with `git apply` to a
  fresh `git archive` of `fc67abd` (tree `da44e4d3…` reproduced), the result
  hashes to `65dc05ea…` — patch, base and candidate agree; 119 `diff --git`
  headers = 119 changed paths.
- Leaf delta vs HEAD (`git diff --name-only HEAD 65dc05ea`): the 13 candidate
  paths; the only non-test, non-doc, non-registry file is
  `internal/traceability/traceability.go`, whose delta is the one-line pin
  `eefba9e5…` → `d3eca906…` (unchanged since CR2).
- `task-board.config.json` = HEAD; vs trunk exactly the one z1yxg9 line
  `go test ./internal/rpcwire -run=^$ -fuzz=^FuzzUntrustedEnvelopes$ -fuzztime=100x -parallel=1`.
- `origin/main` re-fetched this run = `fc67abd` — the CR base is still the
  current trunk authority.

## 2. README F1 fix — the four lines equal the tool output

`go run ./internal/traceability/cmd/tracecheck` on this tree, exit 0:

```text
traceability ok: contracts=63 normative_sections=36 acceptance_cases=132 fixtures=32 compatibility_contracts=55 assigned_scopes=0
section coverage: bindings=65 full=2 partial=6 sliver=4 unevidenced=49 unmeasured=4 unowned=7 clauses_discharged=49/535
```

| README line | Content now | Tool figure | Check |
|---|---|---|---|
| 3152 (fenced sample) | `section coverage: bindings=65 … unevidenced=49 unmeasured=4 unowned=7 clauses_discharged=49/535` | same line | `cmp` BYTE-IDENTICAL |
| 3155 | "Sixty-five section bindings discharge 49 of the 535 normative clauses" | bindings=65, 49/535 | ✓ |
| 3217 | "forty-nine are `unevidenced`. Seven sections are recorded unowned." | unevidenced=49, unowned=7 | ✓ |
| 3229 | "Two admitted bindings out of sixty-five cover five clauses" | full=2, bindings=65, 1+4 clauses | ✓ |

Independent audit of every other figure in the subsection (lines 3147–3260),
each re-derived from the registry and from `tracecheck -section` probes
(`sections-01.log`), none contradicted:

- classes from the registry: full = {2.4, 6.2}; partial = {5.3, 7.7, 13.13,
  14.2, 15.1, 15.3}; sliver = {2.2, 5.5, 8, 10.3}; unmeasured = {7.3, 13.12,
  13.14.5, 15.2}; unevidenced 49; unowned = {18.4, 11.10.5, 14.7.1–14.7.5}
  — the README names exactly these sections in each class;
- per-section ratios: 13.13 9/11, 14.2 8/9, 5.3 7/8, 15.1 5/7, 15.3 2/3,
  7.7 3/4, 10.3 1/3, 5.5 1/3, 8 4/12, 2.2 4/22 (each refused exit 1 with that
  ratio); `-section 6.2` and `-section 2.4` admitted exit 0 — "and nothing
  else" is pinned by `TestVerifyAssignedSectionsRefusesEveryBindingThatOnlySlivers`
  / `TestRunRefusesEveryAssignedSectionThatOnlySlivers` (green, §4);
  1+4+34+10 = 49 discharged, consistent with the tool line;
- "All 13 sections added by v0.6.0 name pending task owners in the reviewed
  registry gaps": each of 6.6, 11.10, 11.10.1–11.10.5, 14.7, 14.7.1–14.7.5
  carries a gap (binding or unowned) naming a `TASK-…` ID — verified;
- the first paragraph under "## Specification-to-Code Ownership Gate"
  (63/36/132/65/7/32/55) equals the report and is measured by
  `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` (green).
- The `132/65/7` registry line at README 3050–3052 (refreshed in rev3) is
  unchanged and still exact.

## 3. LOGBOOK F2 annotation — additive, dated, inside the leaf's entry

`git diff --numstat HEAD -- LOGBOOK.md` = 10 insertions, 0 deletions (the
9-line leaf block + the new line); CR2→CR3 is the single inserted line
`- 2026-09-17 rev3 on `fc67abd`: pin `d3eca906…`, report 132 cases / 65 bindings / 7 unowned / 49 of 535 clauses; harness denominator 24 prior + 9 new; refresh-candidate completed.`
placed as the last bullet of the `### TASK-260830-2x16gz` entry under
`## 2026-09-17`, rev1 text untouched, newest-first preserved, no conflict
markers. Each figure in the line matches the tool output, the pin constant
in `traceability.go:43` and the rev2-verified harness denominator
(24 prior + 9 new = 33). F3 (`section:11.10.5` wording) was explicitly
deferred to the v0.7.0 registry re-derivation leaf by the rework brief; the
registry and pin are unchanged, consistent with that decision.

## 4. Gates rerun on the candidate tree (all observed this run)

| Gate | Result |
|---|---|
| `gofmt -l` (tracked+untracked Go) / `go build ./...` / `go vet ./...` | empty / exit 0 / exit 0 |
| `go test ./internal/hostchannel -run TestHostile -count=1 -v` | exit 0; 54 `--- PASS` (17 top-level), 0 FAIL, 1 SKIP (`TestHostileSSHHelperResponder`, helper-process guard by design); `TestHostileRealCarrierOpenSSH` PASS 0.34 s — OpenSSH loopback lane executed, not skipped; revoke-to-fence 11.5 ms, revoke-to-watch 10.7 ms, idle stream still open 1.2 s (stated bound unchanged) |
| Story packages `-count=1 -v` (hostchannel, rpcwire, hosttrust, traceability/..., config, peeridentity, sshtransport, secprim, cigate) | exit 0, 10 ok, 1749 `--- PASS`, 0 FAIL, 1 SKIP (same); `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` PASS; both only-slivers tables PASS |
| Remaining 27 packages `-count=1` (two batches) | exit 0, 27 ok — 37 of 37 packages green |
| `-race` (hostchannel, rpcwire, hosttrust, traceability/..., config, peeridentity, sshtransport, secprim) | exit 0, 9 ok, 0 `DATA RACE` |
| `-cover` (hostchannel, rpcwire, hosttrust, traceability/...) | 84.1 / 97.8 / 76.3 / 86.2 / 88.5 % — identical to the producer's rev4 figures |
| 14 configured fuzz smokes (`-fuzztime=100x`) | 14 of 14 exit 0 |
| `tracecheck` | exit 0 (figures in §2) |
| `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | exit 0 |
| `GOOS=linux` / `GOOS=windows` `go build ./...` | exit 0 / exit 0 |
| JSON parse over tracked `*.json` | exit 0 |
| `git diff --check` (worktree and `fc67abd..65dc05ea`) | clean |
| stray files (`__pycache__`, `.pyc`, sshd/host-key/authorized_keys leftovers) | none; `git status --untracked-files=all` = the 13 candidate paths |

Accepted from attached evidence, format and exits spot-checked: the
producer's rev4 `-race` over the remaining 32 packages (`batch5b-race-rest.log`,
32 ok, 0 races) and full-tree `-cover` (37 ok), and `task-board validate`
(exit 0; 191 issues in the producer log / 269 in the board's own CR
validation log — pre-existing board-hygiene counts against two different
board checkouts, none referencing this leaf). The board's CR3 validation
summary reads `required=26 green=26 failed=0 missing=0`.

## 5. Attacked, not read — README measurement plants (isolated copy)

| Plant | Where | Result |
|---|---|---|
| R1 measured paragraph: "65 exact section bindings" → "66" | README 3051 | KILLED exit 1: `README ownership paragraph at line 3046 publishes 66 exact section bindings; VerifyRepository measures 65` |
| R2 subsection: sample line `unowned=7` → `unowned=8` and prose "Seven" → "Eight" | README 3152, 3217 | SURVIVED exit 0 — no test reads the "Measured coverage" subsection (see F1 below) |
| R3 control: trailing newline appended | README | SURVIVED exit 0 (applied harmless control) |
| R0 baseline | unmutated copy | exit 0 |

Mutation battery on the product tree: NOT rerun this revision — every
`.go`/`.py` candidate path is byte-identical to CR2 (§1), on which the rev2
review reran the 33-probe harness (31 KILLED, `neutral` passed,
`client-cache` documented SURVIVED) plus nine own registry/census plants.

## 6. Findings

No P1. No P2.

**F1 — P3 (follow-up, not this leaf): the README "Measured coverage of this
repository" subsection is unmeasured.** Plant R2 shows a wrong `unowned=8`
in the fenced sample line and a wrong "Eight sections" in the prose leave
`internal/traceability` and `internal/cigate` green. That is the exact
mechanism by which the P2 in CR1/CR2 arose and it exists on trunk `fc67abd`
too; the leaf's fix is correct on this tree (§2, byte-compared) but
nothing keeps it correct. Recommend a small traceability leaf that compares
the fenced `section coverage:` line (and the four numbered prose figures)
against `Report`, in the style of `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport`.
Out of scope for a story_final README/LOGBOOK rework; does not block.

**F2 — P3 (evidence wording, no repo change): `results-rev4.md` §3 states
"`TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport`: PASS — the F1
fix satisfies the derivation gate".** That test measures only the first
paragraph under the ownership heading and was green before and after the
fix (R2); the F1 lines are proven by the `cmp` against the tool output, not
by that gate. Recorded so the next reader does not infer a gate that is not
there.

F3 from rev2 (`section:11.10.5` "Pending implementation owner" wording)
remains deferred to the v0.7.0 registry re-derivation leaf per the rework
brief; unchanged registry, unchanged digest.

B8 (revocation of idle streams within one second) is unchanged from the rev1
decision: product tree byte-identical, measured bound observed again this
run (idle stream open 1.2 s; fence/watch ≈ 11 ms), `11.10.3` registered
`unevidenced` 0/3 so no clause claim is made on it.

## 7. Reran-vs-accepted

Reran myself with observed exits: candidate tree reproduction; CR2→CR3
path-set diff; patch digest and base→candidate reconstruction; tracecheck
and 24 `-section` probes; registry class/owner derivation; README `cmp`;
LOGBOOK numstat; every gate in §4 not marked accepted; the four README
plants. Accepted from attached evidence: rev4 `-race` on the 32 non-Story
packages, full-tree `-cover`, `task-board validate`; the rev2 mutation
battery on the identical product tree.

## 8. Handoff state

Verdict: ACCEPT. `accept_cr(TASK-260830-2x16gz, revision=3,
evidence=TASK-260830-2x16gz_review-verdict-rev3.md)` — the CR becomes
`accepted`, the element routes to `integrating`; the bound producer run
(developer/implementer) owns checkpoint and integration; no `commit_ack`
from this run. Candidate left UNCOMMITTED in the Story worktree (13 paths,
unchanged by this run); live index/HEAD untouched; scratch copies removed;
review scratch kept under `.temp/TASK-260830-2x16gz/review-rev3/`
(gitignored).
