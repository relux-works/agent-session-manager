# Review verdict — CR-TASK-260830-2x16gz-2 revision 2 (CHANGES REQUESTED, one P2)

Reviewer run: RUN-260917-948768 (reviewer, claude-opus-5 max).
Candidate tree `1ecd6439bc07fb5dfb8a3562788c339d28569396` over trunk base
`fc67abdfc3888d6683b1eb89bc248e089ca6aca3`, Story branch tip `9636508`
(z1yxg9 replay) — 119 changed paths (whole-Story delta: four replayed
checkpoints plus this test-only leaf). Prior revision: CR1 on base `e4e3e88`,
candidate tree `1d03fbc7…`, accepted by RUN-260917-f36c51, demoted stale by
`integration_base_moved`. This review is SCOPED to the reconciliation as
briefed; the rev1 deep review was not redone because the product tree did not
change (proof in §1).

Method: isolated immutable probes only. The candidate tree OID was reproduced
from the working tree through a temporary index file (`GIT_INDEX_FILE`), so the
live index, branch and HEAD were never touched (`git write-tree` on the live
index stayed `ddacb0cb`, `git status` stayed at the 13 candidate paths after
every step). Mutants ran under `go test -overlay` or in a `/tmp` `git archive`
copy of the candidate tree (deleted at the end). `PYTHONDONTWRITEBYTECODE=1`
throughout. Evidence: `TASK-260830-2x16gz_review-evidence-rev2.tar.gz` (real
gzip; `run-notes.md` inside indexes every file named below).

## Verdict: CHANGES REQUESTED — P2 (README figures), two P3 notes

The reconciliation itself is exact and every executable gate is green (§§1–6).
What blocks is the DoD row "README/doctor/capability evidence and specification
traceability are updated without unsupported claims": the candidate README's
"Measured coverage of this repository" subsection states four figures that the
tree's own `tracecheck` contradicts (§7, F1). The republish brief for rev3
explicitly asked to "refresh the tracecheck/report pins and README figures";
one of five README figure sites was refreshed. The fix is four lines; the next
review can be scoped to `README.md` (and `LOGBOOK.md` if the P3 annotation is
taken) by path-set equality against `1ecd6439` for everything else.

## 1. Product tree unchanged since rev1 — proven by path sets

- `git diff --name-only 1d03fbc7 1ecd6439` (rev1 candidate vs rev2 candidate)
  is exactly the 321-path trunk delta `git diff --name-only e4e3e88 fc67abd`
  (both directions of `comm` empty). Every path outside the trunk delta —
  all of `internal/hostchannel`, `hosttrust`, `rpcwire`, `sshtransport`,
  `config`, `peeridentity`, `axerror`, `go.mod`, `go.sum`, `.spec/README.md` —
  is byte-identical to the accepted rev1 candidate. The only trunk-delta path
  under a Story product package is the test file
  `internal/secprim/census_test.go` (§2, resolution 1).
- Of the 321 trunk-delta paths, 313 are byte-identical to trunk `fc67abd` in
  the candidate (stale carried copies correctly restored). The remaining 8 are
  the declared reconciliation paths: `internal/secprim/census_test.go`,
  `internal/traceability/{ownership.v0.6.0.json, traceability.go,
  traceability_test.go, cmd/tracecheck/main_test.go}`, `LOGBOOK.md`,
  `README.md`, `task-board.config.json`.
- `task-board.config.json`: 3-way (`git merge-file`) reconstructs the candidate
  byte-for-byte with zero conflicts; candidate vs HEAD is empty; candidate vs
  trunk is exactly the one z1yxg9 line
  `go test ./internal/rpcwire -run=^$ -fuzz=^FuzzUntrustedEnvelopes$ -fuzztime=100x -parallel=1`.
- `git diff --name-only HEAD 1ecd6439` = the 13 candidate paths (11 modified +
  `hostile_test.go`, `hostile_carrier_test.go`); no production file among them.

## 2. Refresh replays — signed and faithful (4 of 4)

| Original | Replayed | verify-commit | parent | paths | exact carry | clean 3-way | conflicts |
|---|---|---|---|---:|---:|---:|---:|
| `14d1176` (2u34k1) | `3337a0b` | Good, oparin@me.com | `fc67abd` | 9 | 7 | 2 | 0 |
| `4677a87` (1tvg8e) | `6ebe774` | Good | `3337a0b` | 10 | 9 | 1 | 0 |
| `a14f718` (2ez769) | `c3fec77` | Good | `6ebe774` | 74 | 71 | 2 | 1 |
| `c3de159` (z1yxg9) | `9636508` | Good | `c3fec77` | 30 | 25 | 4 | 1 |

Per pair: `git diff --name-only` of the original and replayed commits equal;
subjects and author equal; for every path the replayed blob equals either the
original blob (trunk did not touch it) or the clean `git merge-file`
reconstruction (original as ours, original parent as base, new parent as
theirs). Exactly the two declared conflicts remain:

- Resolution 1, `internal/secprim/census_test.go` @ `c3fec77` (sha256
  `7a99d2da…`, 28287 bytes — matches the rev3 record): same-anchor row
  insertion; trunk's `provhost|decl|unrestrictedTokens` row is kept first,
  then the checkpoint's six credential/custody rows; the shared `sites:` opener
  is closed correctly; `gofmt` clean; the package list line carries the
  checkpoint's `hosttrust` addition over trunk's unchanged list. Liveness of
  both sides proven by mutants M7/M8 (§6).
- Resolution 2, `internal/traceability/traceability.go` @ `9636508` (sha256
  `4497782a…`): the conflict is the single pin line (checkpoint `5102d3bc…` vs
  trunk `39087ef8…`); resolved to `eefba9e5…`, which my standalone recompute
  reproduces over the `9636508` registry (§4).
- `LOGBOOK.md` auto-merged at every replay: trunk→HEAD is 136 insertions, 0
  deletions (every 2u34k1/2ez769/z1yxg9 entry present; 1tvg8e never wrote an
  entry, matching its original patch).

## 3. Stale-copy audit and hand merges — exact

- `README.md`: `git merge-file` yields one conflict (the 132/65/7 figure line);
  outside it the candidate equals the merge output line for line. Trunk→
  candidate is 202 insertions / 8 deletions; the 8 deletions are the three
  figure lines and the five-line `15.3#3` sentence the Story (z1yxg9) rewrites
  because `internal/rpcwire` now builds hello frames — a Story change already
  present in the rev1 candidate, not a reconciliation loss. Trunk's `-adopted`
  regions and every trunk-new section are intact.
- `LOGBOOK.md`: one conflict (append-append at the top); resolved by inserting
  the leaf's 9-line entry under trunk's existing `## 2026-09-17` heading
  (duplicate heading dropped). Candidate vs HEAD = 9 insertions, 0 deletions;
  newest-first holds at the top; zero conflict markers; trunk's 3lt2xv,
  2atgj4, 17ootk, 2zvo8m, 3uzfyn, kp4zpu, 3k3e6m, 14yo67, 3g12yp, 2f5393
  entries all present.
- `ownership.v0.6.0.json`: JSON-level 3-way (base `e4e3e88`, ours `1d03fbc7`,
  theirs `fc67abd`): acceptance cases 101 unchanged + 30 trunk-only +
  1 story-only (`AC-HOST-001`) = 132, ordered as trunk's list then
  `AC-HOST-001`; ownership 46 unchanged + 11 trunk-only + 13 story-only = 70;
  unowned 6 unchanged + 1 trunk deletion (`2.2`) + 5 story-only (deletions of
  `11.10.1`–`11.10.4`, `11.10.5` text) = 7. No key modified by both sides;
  every story-only value equals the accepted rev1 value; `format`,
  `format_version`, `source` equal trunk.
- `traceability_test.go`, `cmd/tracecheck/main_test.go`: trunk's `2.2`/`2.4`/
  `18.4` expectation updates present; only the figure pins moved
  (132/65/49 unevidenced/7 unowned/49 of 535); both packages pass (§5).
- `traceability.go`: the one-line re-pin `eefba9e5…` → `d3eca906…`.

## 4. Digest re-pin — reproduced independently

Standalone Go program over verbatim copies of the projection structs
(`tools/digestcheck-main.go`; `json.Marshal` of the decoded registry, sha256):

| Registry | cases/ownership/unowned | digest | claimed |
|---|---|---|---|
| trunk `fc67abd` | 131/65/11 | `39087ef8…` | ✓ |
| original `c3de159` | 101/61/12 | `5102d3bc…` | ✓ |
| replayed `9636508` | 131/65/11 | `eefba9e5…` | ✓ |
| rev1 candidate `1d03fbc7` | 102/66/8 | `e327bb08…` | ✓ |
| rev2 candidate `1ecd6439` | 132/70/7 | `d3eca906…` | ✓ (the pinned constant) |

`tracecheck` on the candidate: exit 0,
`acceptance_cases=132 … bindings=65 full=2 partial=6 sliver=4 unevidenced=49
unmeasured=4 unowned=7 clauses_discharged=49/535`. The +24 clauses over trunk
are exactly the five new bindings' sections (11.10.1: 10, 11.10.2: 5,
11.10.3: 3, 11.10.4: 1, 16.1: 5), each refused for assigned-scope admission as
`0/n … unevidenced` (probed with `-section`).

## 5. Suites rerun on the candidate tree (all observed this run)

| Gate | Result |
|---|---|
| `gofmt -l` (tracked+untracked Go) / `go build ./...` / `go vet` (Story pkgs + secprim) | empty / exit 0 / exit 0 |
| `go test ./internal/traceability/... ./internal/secprim -count=1 -v` | exit 0 (tracecheck pins and census union live) |
| `go test ./internal/hostchannel -run TestHostile -count=1 -v` | exit 0, 54 `--- PASS`, 0 FAIL, 1 SKIP (`TestHostileSSHHelperResponder`, the forced-command child guard, by design); `TestHostileRealCarrierOpenSSH` PASS 0.35 s — executed, not skipped; revoke-to-fence 22.6 ms, revoke-to-watch 35.9 ms, idle stream still open 1.2 s (observed, stated bound unchanged) |
| Story packages (`hostchannel rpcwire hosttrust traceability/... config peeridentity sshtransport secprim`) `-count=1` | 9 ok |
| `-race` (`hostchannel rpcwire hosttrust traceability/...`) | exit 0, 5 ok, 0 data races |
| remaining 28 packages `-count=1` | exit 0, 28 ok (37 of 37 packages green in total) |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0 (figures above) |
| `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` | exit 0 |
| `git diff --check` (worktree and `fc67abd..1ecd6439`) | clean |
| stray files (`__pycache__`, `.pyc`, sshd state, keys) in changed paths / worktree | none |

Carrier lane read-check (not a re-review): `hostile_carrier_test.go` starts an
unprivileged `sshd -D -e -f` on `127.0.0.1` with a temp host key, forced
`command="…",no-pty` authorized key, client `-T` first, asserts first stdout
bytes `0x16 0x03`, wire command == `ServeArgv`, no PTY, one served call,
bounded stderr; every skip path names its reason (`Skipf`). Native Tailscale
SSH stays a stated bound in the matrix, README and hostchannel README.

Accepted from attached evidence (format spot-checked): the producer's 14 fuzz
smokes, `GOOS=linux/windows go build`, JSON parse gate and `task-board
validate` (exit 0 — note the CR validation log reports 269 issues while the
rev3 record says 191; both are pre-existing board-hygiene issues against the
worktree's board checkout and neither count is a candidate defect).

## 6. Mutation — producer battery 33 of 33 rerun; 9 own plants

Producer harness `internal/hostchannel/mutations.py` rerun in three bounded
batches (`--only`), full behavioral suite per plant under overlay, harness
exit 0 each: 31 KILLED by their named vectors (all 9 new probes:
`alpn-same-family`, `oversize-dispatch`, `stale-client-rebind`,
`rpc-contracts`, `error-version-drift`, `truncated-dispatch-ignored`,
`snapshot-integrity`, `correlation-health`, `nonce-static`; the 22 prior
including token-preserving `census-evasion` killed by `TestRuntimeWriteCensus`),
`neutral` passed, `client-cache` SURVIVED as the documented harmless control.
Denominator: 33 registered probes (24 prior + 9 new; the rev1 text's "26
prior" counted pre-leaf probes across packages — rev3 record already corrects
this).

Reviewer-owned plants on the reconciled surfaces (raw logs in
`own-mutants/`):

| Plant | Where | Result |
|---|---|---|
| M1 self-minted acceptance case naming a nonexistent test, live pin | registry copy, live `tracecheck` binary `-root` | REFUSED exit 1: `declaration "TestHostileReviewerNonexistent" is absent` (structural check fires before the digest) |
| M2 same, digest RE-PINNED to the mutated registry (compiled from the copy) | registry + pin | REFUSED exit 1, same structural refusal — a re-pin does not launder a claim naming no real test |
| M3 structurally valid duplicate of AC-HOST-001 under a new ID, NOT re-pinned | registry | REFUSED exit 1: `projection digest 471d354d… differs from reviewed d3eca906…` — the pin binds the merged registry content |
| M3 re-pinned | registry + pin | admitted (exit 0, 133 cases) — by design: the pin IS the review binding |
| M4 leaf binding `11.10.4` claims `full` with zero clauses, re-pinned | registry + pin | REFUSED: `claims full coverage but discharges 0 of the 1 normative clauses` |
| M5b leaf binding `11.10.1` production declaration renamed to a nonexistent one, gap rewritten to match, re-pinned | registry + pin | REFUSED: `declaration "ServeReviewerNonexistent" is absent from "internal/hostchannel/server.go"` (M5 without the gap rewrite was refused earlier by the gap-name check) |
| M6 leaf binding `11.10.3` with no acceptance case, re-pinned | registry + pin | REFUSED: `has no scope-specific acceptance owner` |
| M7 delete trunk's `provhost|decl|unrestrictedTokens` row from the merged census | `secprim/census_test.go` overlay | KILLED exit 1: `TestSecretSiteRosterIsComplete` names exactly `provhost|decl|unrestrictedTokens` |
| M8 delete the checkpoint's `hosttrust|decl|cleanupCredentialDirectory` row | overlay | KILLED exit 1: same test names exactly that site |
| M9 comment-only control | overlay | SURVIVED exit 0 (applied harmless control) |

(A first M7 attempt was a compile failure from a bad cut and is recorded as
NOT measured; the re-plant above compiles, `gofmt` clean.)

## 7. Findings

**F1 — P2 (blocks): README "Measured coverage of this repository" figures are
trunk's, not the candidate's.** `tracecheck` on this tree prints
`bindings=65 … unevidenced=49 … unowned=7 clauses_discharged=49/535`; the
candidate `README.md` states:

- `README.md:3152` (fenced sample introduced by "`tracecheck` prints the ratio
  it measured rather than a sentence about it:") —
  `section coverage: bindings=60 full=2 partial=6 sliver=4 unevidenced=44 unmeasured=4 unowned=11 clauses_discharged=49/511`
  → must be `… bindings=65 … unevidenced=49 … unowned=7 clauses_discharged=49/535`
- `README.md:3155` — "Sixty section bindings discharge 49 of the 511 normative
  clauses their" → "Sixty-five … 49 of the 535 …"
- `README.md:3217` — "and forty-four are `unevidenced`. Eleven sections are
  recorded unowned." → "forty-nine … Seven …"
- `README.md:3229` — "Two admitted bindings out of sixty cover five clauses"
  → "out of sixty-five"

Trunk `fc67abd` keeps this subsection exact against its own gate (60/44/11/
49/511), and the landed precedent (2atgj4) updates it per Story. The rev1
candidate carried the same class (its subsection said 56/463/48/12 while its
gate printed 61/487/53/8 — missed by the rev1 review), and the rev3 republish
brief's "refresh … README figures" refreshed only the 132/65/7 line at
`README.md:3050–3052`. Four false statements about the traceability state, one
of them a quoted tool output, fail the DoD row "README … updated without
unsupported claims". Rework: edit exactly those four lines; no other path
needs to change. The next review can be scoped to `README.md` by verifying
path-set equality against `1ecd6439` elsewhere.

**F2 — P3 (non-blocking): the leaf's LOGBOOK entry carries rev1-time
statements** — "digest re-pinned `e327bb08...`; report 102 cases / 61
bindings / 8 unowned / 17/487 clauses", "pre-existing 26 probes", and the
trailing NOTE "refresh-candidate … fails … worktree untouched at `244a7dc`"
(already a rev1 P3). They were true on `2026-09-17` at rev1 and are a dated
journal record, but the landing tree's pin is `d3eca906…` and its report is
132/65/7/49 of 535. Recommend a one-line dated annotation in the same entry
(e.g. "rev3 on `fc67abd`: pin `d3eca906…`, report 132/65/7, 49/535; harness
denominator 24 prior + 9 new") while the README is being fixed.

**F3 — P3 (wording): `section:11.10.5` unowned text ends with "Pending
implementation owner: TASK-260830-2x16gz."** while the same gap and evidence
state that no policy reader is implemented by design and that the section is
consumed as source evidence only. "Pending implementation owner" for a
section with nothing to implement will read as stale once this leaf lands;
"Disclosure owner" or dropping the sentence would be accurate. Optional; it
changes the registry digest, so take it only together with a re-pin.

No P1. B8 (revocation of idle streams within one second) is unchanged from the
rev1 decision: the product tree is byte-identical, the measured bound is still
observed (idle stream open 1.2 s, fence/watch in tens of ms), and `11.10.3` is
registered `unevidenced` 0/3 so no clause claim is made on it.

## 8. Reran-vs-accepted

Reran myself with observed exits: candidate tree OID reproduction; path-set
proofs (§1); replay signatures ×4, parent chain, per-path 3-way fidelity (§2);
`git merge-file` reconstructions and the JSON 3-way of the registry (§3);
five digest recomputes (§4); every gate in §5 except the four noted as
accepted; the full 33-probe harness and nine own plants (§6). Accepted from
attached evidence: fuzz smokes, cross-builds, JSON parse, `task-board
validate` (as stated in §5).

## 9. Handoff state

Verdict: changes requested (P2 F1). Routing `TASK-260830-2x16gz` to `to-dev`
after attaching this verdict and `TASK-260830-2x16gz_review-evidence-rev2.tar.gz`.
CR revision 2 is not accepted; the candidate stays UNCOMMITTED in the Story
worktree (13 paths, unchanged by this run); no commit, checkpoint, integration
or product edit by this run. Scratch copy `/tmp/rev2-2x16gz/` removed; review
scratch kept under `.temp/TASK-260830-2x16gz/review-rev2/` (gitignored).
