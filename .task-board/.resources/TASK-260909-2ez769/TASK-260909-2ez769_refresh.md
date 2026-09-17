# TASK-260909-2ez769 refresh outcome (RUN-260910-a4ac2c)

Producer recovery run: story base moved from c1eff01 to a89328c
(trunk 8cf4aaa with the v0.6.0 adoption), candidate preserved and
integrated, full validation green. Worktree left UNCOMMITTED on
task-board/story/STORY-260830-1kiyj6 for handoff snapshot. No producer
commit, no integration, no main push.

## 1. Resume preconditions (verified, not assumed)

- origin/main contains the signed adoption STORY-260908-18woqo
  (5d98b08 + d727513) under trunk 8cf4aaa; results.md precondition holds.
- Canonical CLI: /Users/iv/.curator/global/bin/task-board (wrapper to
  curator cache build go-v1/9fd68ba9...); `which task-board` resolves to it.
- Exclusive story lease held by this run (RUN-260910-a4ac2c) throughout.

## 2. Preservation (re-verified this run, before any change)

- RPC control backup
  .temp/TASK-260830-z1yxg9/preserved-before-credentials-260910:
  15/15 file bytes match manifest SHA256.
- Parked delta .temp/TASK-260909-2ez769/parked-rpc-delta-260910:
  11/11 untracked files match manifest.
- 4 tracked RPC paths clean vs HEAD; no internal/rpcwire or
  UNRESOLVED_QUESTIONS.md remnants in worktree.
- Candidate fingerprinted before refresh (40 files,
  .temp/TASK-260909-2ez769/candidate-before-refresh.sha256);
  post-refresh all 40 bytes ALL-MATCH.
- No RPC byte absorbed: parked delta intact, RPC task untouched.

## 3. Managed base refresh (exclusive lease, signed replay)

- `task-board worktree refresh-candidate TASK-260909-2ez769` first
  replayed 43c0e2b onto 8cf4aaa, then stopped for a resolution packet;
  second round replayed c1eff01. Outcome: refresh_advanced, branch
  c1eff01 -> a89328c. Both replayed commits signed oparin@me.com
  (41106b6, a89328c; `git verify-commit` Good).
- Resolution packet schema (learned from tool errors, no docs exist):
  entries {checkpoint_oid, path, sha256, content-base64} in
  `resolutions[]`. Packet builder: /tmp/build-resolutions.py;
  packets /tmp/rr1.json (round 1) + /tmp/rr12.json (combined).
- Round-1 resolutions (REBASE_HEAD 43c0e2b): LOGBOOK.md merged
  newest-first (trunk 2026-09-09 block + checkpoint 2026-09-08 block +
  base); README.md trunk version + checkpoint peer-identity tail append;
  traceability.go / traceability_test.go / tracecheck main_test.go =
  trunk bytes (v0.6.0 registry pin a1ab2913 authoritative; checkpoint
  v0.5.0 digest superseded because the gate reads ownership.v0.6.0.json).
- Round-2 resolutions (REBASE_HEAD c1eff01): 3 traceability files reuse
  round-1 bytes (c1eff01 identical to 43c0e2b there); README.md adds the
  SSH `## SSH stdio transport` section and peer-section wording at
  mapped offsets (context-asserted). LOGBOOK.md merged cleanly by git.
- HEAD verification: 4/4 non-LOGBOOK resolutions byte-identical in
  a89328c; replay diffs vs originals carry the full trunk delta as
  expected for a rebase.

## 4. Combine (tool preserves candidate bytes; producer merges source)

- Restored 30 enumerated incoming paths from HEAD (8 deleted adoption
  files incl. SPEC.v0.6.0.md, v0.6.0.lock.json, ownership.v0.6.0.json,
  adoption-v0.6.0.md, task-board.config.json with the v0.6.0 cataloggen
  command; 22 modified non-candidate paths). No repo-wide reset; every
  path enumerated; candidate files untouched.
- 6 tracked candidate files (config migration/schema/validation/writer,
  refusal pin, census) verified ADOPTION-UNTOUCHED (HEAD == c1eff01):
  kept as-is. LOGBOOK.md re-merged (HEAD + 2 prior credential entries,
  newest-first verified). No v4/hosttrust path collisions with trunk.
- .task-board worktree paths left to tooling (checkout artifact).

## 5. Real semantic integration (adoption "v4 unimplemented" now stale)

The v0.6.0 adoption pinned Config-4 Load/Migrate refusal contingent on
no implementation ("No config production code changes remain"). This
leaf IS the implementation, so 4 reviewed adoption test files were
updated minimally to the new true behavior (production code untouched):

- compatibility_regression_test.go: removed `const Version4` colliding
  with production schema.go (identical value); v4 lanes now POSITIVE
  per-lane loads of complete v4 docs (windows/ax.conpty/D-path/native
  plugdir, wsl2/ax.tmux//srv path; SourceVersion 4.0.0 + host_channel
  asserted); conpty-on-v4 now closed-shape refusal (ErrConfigDecode,
  explicitly not ErrUnsupportedConfigVersion).
- schema_test.go: v4 branch now bare-doc required-member refusal
  (ErrConfigValidation, still fail-closed, no legacy selection) plus
  complete-doc positive (source/value 4.0.0, ssh_tls13, host_channel);
  "unsupported major" renamed to "v4 bare census document".
- migration_test.go: assessor cross-product now 6 read-only / 10
  compatible / 0 refused (was 6/6/4); v4-reader special case removed.
- migration_refusal_test.go: v4 row now wants ErrMigrationV4Explicit
  ("v4 requires explicit preview and confirmation"); harness converted
  to per-row want (was hardcoded ErrMigrationTarget).

New expectations were measured first with throwaway probes (since
deleted): both lanes encode+load v4 positively; bare-v4 refuses with
ErrConfigValidation; conpty-on-v4 refuses with ErrConfigDecode.

## 6. Validation (exact provenance: branch a89328c, worktree as left)

- gofmt clean; `go vet ./...` clean; GOOS=windows build of
  hosttrust+config ok; `go build ./...` exit 0.
- `go test ./... -count=1`: 26/26 packages ok.
- `go test ./... -cover`: config 93.2%, hosttrust 77.0%,
  peeridentity 97.5%, secprim 94.4%.
- tracecheck exit 0 (contracts=63 sections=36 cases=101 fixtures=32,
  bindings=56, clauses 17/463); catalog v0.6.0 `-check` exit 0 (the
  former gate-21 input now present and passing).
- Mutation batteries rerun on the new base:
  .temp/TASK-260909-2ez769/mutations-hosttrust-refresh: 15 narrowing
  KILLED by named tests + 1 neutral passed (manifest.json bound to
  a89328c with per-source digests).
  .temp/TASK-260909-2ez769/mutations-config-refresh: 4 narrowing KILLED
  + 1 neutral passed; the v4-explicit mutant now also kills via the
  updated adoption row
  (TestMigrateRefusesEveryTargetOutsideTheUpgradeVocabulary/v4_requires_explicit_preview_and_confirmation).
- No gate searches source text for a token; preserve-token mutant
  clause remains vacuous (stated, per retry3).
- Negative/AC accounting from results.md (21/21 rows driven) and retry3
  (4 hardened gates + 5 refusal rows + genuinely-unsorted fixture)
  carries over unchanged: all touched source files byte-identical;
  this run EXTENDS the migration/reader rows through the updated
  adoption tests above (named passing subtests observed:
  4.0.0/windows, 4.0.0/wsl2, document_4.0.0_reader_4.0.0,
  v4_requires_explicit_preview_and_confirmation,
  v4_bare_census_document).

## 7. Known bounds (unchanged from prior runs)

Live TLS 1.3 launch/handshake/dispatch lifecycle stays with the RPC
transport task; Windows DACL beyond owner-SID equality has no runner
here (code builds for windows, unix enforced); peer-side distribution
of enrollment material is operator duty per spec. The active v0.6.0
registry does not yet name hosttrust/Config4 owners (adoptions
13-owner census predates this leaf); the gate passes because owners
must resolve to declarations, not the reverse. Registry ownership for
the new code is integrator/reviewer business, flagged here explicitly.

## 8. Handoff state

Branch task-board/story/STORY-260830-1kiyj6 at a89328c, candidate
UNCOMMITTED (11 tracked: 7 credential + 4 adoption-test integration;
untracked: internal/hosttrust/*, internal/config migration_v4* +
schema_v4_test + mutations_v4.py, peeridentity v4 interop test).
Ready for review on the current adoption base; CR validation command 21
inputs are present in-tree.
