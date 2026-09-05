# TASK-260830-32jeti — base-refresh merge report (rev 7 → republish)

Run: RUN-260905-ea9a88 ([implementer] developer (muse)).
Brief: `integration_base_moved` refused rev 7 — trunk advanced to `2512f20`
(sibling STORY-260830-3m2mw8 landed) with changes to `LOGBOOK.md` and
`README.md`, which this Change Request also changes. Refresh onto trunk,
resolve the overlap in those two files, republish. Change no Go file.

## Pre-merge identity (read-only verification)

- Story branch tip `8fbc052`, fork base `57afcc6`, trunk `2512f20`
  (`git merge-base HEAD 2512f20` = `57afcc6`).
- Rev-7 patch file set (60 paths incl. 2 untracked) is identical to the
  worktree diff-vs-base file set: the tree IS the accepted rev-7 content.
- Trunk gained since fork: LOGBOOK +86, README +29/−6. This story adds:
  LOGBOOK +137, README +138/−4. Only these two files overlap. No other
  path is touched by both sides.

## LOGBOOK.md — merged, both sides kept whole (894 lines = 675 + 137 + 86 − 4)

The 4-line deficit from the naive sum is the two `## 2026-09-05` /
`## 2026-09-04` date headers plus their blanks, which are byte-identical
on both sides and counted once. Zero lines removed from either side
(`diff` both directions: additions only).

- `## 2026-09-05`: our 0935/0926/0830/0510/0435/0429/0327, then sibling 0022
  (00:22 is older than all seven; newest-first preserved).
- `## 2026-09-04`: sibling 2340/2210/2147 (21:xx–23:xx), then our
  0230/0210/0115 (00:xx–02:xx); newest-first preserved.
- Tail after base `### 2113` (EOF appendix, arrival order): sibling 5
  TASK-entries (landed first), then our 2321/1244. Sibling relative order
  untouched; no sibling entry dropped, rewritten, or reordered.
- Byte-identity asserted in-script: sibling 0022 block (10 lines), 09-04
  block (26 lines), tail block (43 lines) equal trunk bytes; our blocks
  equal pre-merge worktree bytes. Script: `/tmp/splice_logbook.py`.

## README.md — merged onto trunk text, contradictions resolved (not concatenated)

1. Our `## Provider Plugin Discovery and Trust` + `## Provider Plugin JSONL
   Protocol` sections inserted before `## Canonical JSON …` (region trunk
   does not touch). Sibling conpty / 15.3#3 / single-clause paragraphs kept.
2. Refusal-arm inventories (the brief's explicit check): the document now
   describes THREE per-package inventories — provider ("from its own
   source"), provhost ("from its own source"), terminalbackend
   ("AST-derived …"). Added one plain sentence to the provhost section:
   the three are independent implementations sharing no inventory code or
   table. Verified: separate derivation functions per package
   (`deriveRefusalArms` vs `auditRefusalInventory`/`deriveRefusalInventory`
   vs the terminalbackend file); the only cross-package touch is
   `provider.Builtins()` IDs in a profile-mapping test, not inventory
   machinery.
3. Ownership figures reconciled against the pin test, NOT concatenated:
   `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` measures the
   CURRENT tree (78 cases / 53 bindings in `ownership.v0.5.0.json`), so the
   merged paragraph keeps 78 / 53. Writing 77 (trunk's number) or 81 would
   redden the suite with no Go change allowed to fix it. Coverage block and
   "Fifty-three … 17/428" kept (sibling untouched; pinned by
   `tracecheck/main_test.go`).
4. POST-INTEGRATION NOTE (for the integration run, not this CR): trunk JSON
   holds 74 + 3 sibling cases (`terminal-manifest-probe-admission`,
   `terminal-provider-descriptor-7a`, `terminal-lifecycle-conformance`);
   this tree holds 74 + 4 ours (`provider-discovery-and-trust`,
   `provhost-quiesce-proof`, `provhost-profile-mapping`,
   `provhost-capability-gate`). Names are disjoint, so the union is 81
   cases / 53 bindings — whoever resolves the `ownership.v0.5.0.json`
   overlap at integrate time MUST also lift the README figures 78 → 81
   (and re-pin), or the pin test goes red on trunk. Section bindings stay
   53 (sibling added none).

## Verification (this tree, after merge)

- `go build ./...` → exit 0
- `go test ./... -count=1` → exit 0, all 15 packages ok
- `go vet ./...` → exit 0; gofmt gate clean
- `TestREADMEOwnershipFiguresAreDerivedFromTheMeasuredReport` → PASS
  (would FAIL on 77 or 81 — the concatenation defect, avoided)
- `traceability/...`, `cliresult` GuardInventory, canonicaljson session
  grammar README test → ok
- Full-tree sha256 before/after: EXACTLY two paths differ — `LOGBOOK.md`,
  `README.md`. Zero Go files (or any other file) changed.

## Publication

- Files differing from accepted rev 7: `LOGBOOK.md`, `README.md`, nothing
  else. Proven by full-tree sha256 before/after (no files added/removed,
  only those two modified) plus rev-7-patch/worktree file-set identity.
- Resulting revision: not yet published at handoff time. There is no
  producer-side publish command by design — `Publish` runs in the
  completion-time finalizer (`spawnruntime/changerequest.go`, gated on
  `producerCompletionHandedOff`, which this handoff to `to-review`
  satisfies). When this run finalizes, the publisher snapshots the tree
  above into the next revision (expected: rev 8, `repository_delta`
  present, same 60-path scope — file set unchanged, only the two files'
  bytes moved) and runs the §6.2.1 validation suite against it with a
  tree-drift check. Read the number back with
  `task-board worktree status` (change-req line for TASK-260830-32jeti).
- The combination still needs its review pass: the gate's complaint stands
  until a reviewer looks at the two sides together (round-10 review of the
  republished revision, then `accept_cr`, then integrate).
