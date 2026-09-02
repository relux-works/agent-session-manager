# Flight Logbook

> Institutional memory. Concise, factual, high-signal.
> Newest entries first. One block per insight.

## 2026-09-01

### 2355 — A deferred error field is untestable through a struct literal

- FINDING: When a capture-time refusal is replaced by a deferred error field
  (`OSInputs` stopped refusing a failed `os.UserHomeDir` and stored
  `Inputs.homeDirError` instead), the field itself cannot be evidenced by a
  fixture that assigns it. The case then only proves it is self-consistent.
- FIX: Reach the state through the exact process input production reads. Clear
  or set the variable `os.UserHomeDir` consults (`HOME` on unix, `USERPROFILE`
  on Windows, `home` on plan9) and let the real capture run.
- FINDING: The expected cause cannot be a literal or an identity comparison.
  `os.UserHomeDir` constructs a fresh `errors.New` per call, so `errors.Is` can
  never match; the only stable evidence is the message a real call returns in
  the same process environment.
- SCOPE: `internal/config/loader_home_test.go`,
  `internal/config/migration_os_test.go:TestMigrateOSMigratesTheHomeDerivedConfigurationAtTheRealProcessEntry`
- STATUS: Resolved — CR rev4 finding B1 closed, 16/16 mutants RED.

### 2355 — A gate proven only at the capture says nothing about its propagation sites

- FINDING: Killing the capture mutant (`homeDirError = nil`) leaves eight
  per-site `errors.Join(ErrPlatformDefaultUnavailable, inputs.homeDirError)`
  call sites unproven. The first home-derived path class to resolve masks every
  later one, so dropping the cause at any single later site stays green.
- FIX: Isolate one class at a time — supply real overrides for every registry
  class **except** the one under test, so exactly one falls through to the
  platform-default layer. Loop over the derived home-derived class set, so a
  class added later is covered without editing the case.
- FINDING: All eight sites (4 macOS + 4 Linux/WSL2) then die individually; a
  delete-only capture mutant reached none of them.
- SCOPE: `internal/config/loader.go:macOSDefault`, `linuxDefault`
- STATUS: Resolved.

### 2355 — "Host-gated" coverage was an unexamined assumption, not a real limit

- ANOMALY: Four Linux/WSL2 cause-drop mutants survived a darwin-host sweep
  while every macOS mutant died. The first read was "linuxDefault is
  unreachable from this host; declare it host-gated and report honestly".
- FINDING: Wrong. `LoadOS(platform, overrides)` takes the runtime-probed AX
  platform as a *parameter* — it never infers it from `GOOS`. `os.UserHomeDir`
  reads `$HOME` on both unix hosts and the rendered separator matches, so the
  macOS, Linux and WSL2 lanes all drive through the real entry from one host.
- FIX: `homeDrivenPlatforms` returns all three lanes on darwin/linux. The four
  survivors turned RED. Windows stays skipped for a real reason: production
  derives no path default from the user home there.
- DECISION: Before declaring a mutant host-gated or equivalent, check whether
  the production entry actually takes the varying dimension as an input. An
  honest "survivor" report is still worse than the coverage you could have had.
- STATUS: Resolved.

### 2340 — Reviewer rev4: a removed refusal can hide behind self-minted evidence

- FINDING: `internal/config/loader.go` deleted the `OSInputs` "capture user home"
  refusal and replaced it with a deferred `Inputs.homeDirError` field joined into
  `ErrPlatformDefaultUnavailable` by the nine `HomeDir == ""` branches. The design
  is right, but the only test (`TestResolvePathsDefersButPreservesHomeLookupFailure`)
  hand-sets `inputs.HomeDir = ""` and `inputs.homeDirError = homeFailure` on a
  fixture. It proves propagation of a synthetic value, not that the production
  capture ever supplies one.
- FINDING: two mutants survive the full suite as a result — `homeErr = nil` and
  `home = ""` immediately after `os.UserHomeDir()` in `OSInputs`. Every `LoadOS`
  and `MigrateOS` test overrides all five Section 3.2 path classes, so no test in
  the package ever reaches a home-derived platform default at a real entry.
- NOTE: the front-loaded DoD item 8 ("suite must pass under `env -i` with an empty
  HOME") and this gap are the same coin. The suite is green under `env -i` partly
  *because* nothing exercises the home path; determinism evidence does not imply
  wiring evidence. Both were re-verified separately on revision 4.
- FIX: a ~30-line pair of cases at the real `LoadOS` entry closes it —
  `t.Setenv("HOME", "")` with all `AX_*`/`XDG_*` cleared kills the dropped-cause
  mutant, and `t.Setenv("HOME", t.TempDir())` asserting the macOS
  `Library/Application Support/ax` default kills the dropped-home mutant. Written
  and verified; attached as `TASK-260830-1qf777_reviewer-home-probe-rev4_test.go`.
  Both pass under `env -i` with an empty ambient HOME.
- NOTE: general shape for later review rounds — when a change *removes* a refusal,
  the replacement path needs evidence at the same entry the refusal used to guard.
  A test that injects the value the production capture was supposed to produce is
  the self-minted-evidence shape wearing a refactor's clothes.
- SCOPE: `internal/config/loader.go` (`OSInputs`, `macOSDefault`, `linuxDefault`),
  `internal/config/loader_test.go`.
- STATUS: routed `to-dev` as the single blocking finding of CR revision 4. All 44
  other reviewer mutants were killed or shown equivalent; the three revision-3
  blocking findings are fixed and each was re-killed with its original mutant.


### 2340 — Derive the property, then make the derivation itself falsifiable
- DECISION: Closing rev3 F2 by asserting three named members would have re-created the defect. The shape that works has three coupled parts: derive the member set from the versioned wire type (`wireLeafTOMLPaths` over `reflect.TypeOf(rawV1{})`), require a fixture to cover every derived path at a value distinguishable from the one a dropped member decays to, and compare the WHOLE loaded `Configuration` before and after migration. Any two of the three leave a hole: derivation without saturation passes vacuously, saturation without whole-struct comparison misses the next member, whole-struct comparison over a defaults-only fixture compares defaults to defaults.
- FINDING: The distinguishability check needs a defaults oracle, and the honest one is production output — migrate a minimal v1 through the real `Migrate` and flatten the result. `currentWire` emits every member explicitly, so that document IS the decay target. No hand-written default table to drift.
- FINDING: Ten Configuration 1.0.0 members are pinned to a single valid value by the validators, so they cannot be saturated. An exemption list is where this kind of property quietly rots. Fix: each exemption declares the alternate value it claims is refused, and the test drives that alternate through the real `Load`. Mutants N18-N23 widen each validator in turn and every one reddens the exemption self-check — the list cannot outlive the constraint that justified it.
- FINDING: rev3 observation 5 said the non-canonical semver mutant (`01.0.0`) was subsumed because `knownConfigVersion` gates the same values. It is not: the read-only branch returns before that check ever runs, so `AssessCompatibility("04.0.0", reader 3.0.0)` succeeds under the mutant. A subsumption claim is a behavioral claim and needs the same proof as any other.
- DECISION: R13 (encodeVersion2 dropping directory collections) genuinely cannot die — Migrate reaches that encoder only with a v1 source and rawV1 declares no directory collection. Reported as a SURVIVOR with its subsuming check named (`TestVersion1SourcesNeverCarryDirectoryCollections`), not folded into the RED count. The harness also prints `SUBSUMPTION CLAIM NO LONGER NEEDED` if it ever starts dying. A sweep that reports 73/73 by quietly reclassifying its one honest survivor is worth less than one that reports 71 and explains the other two.
- SCOPE: internal/config/{migration_preservation_test.go,migration_test.go,migration_refusal_test.go,refusal_structure_test.go,writer.go,migration.go}, README.md, internal/traceability/{ownership.v0.5.0.json,traceability.go}
- STATUS: 73 mutants — 71 RED, 1 declared-subsumed, 1 positive control GREEN, 0 unexplained survivors. internal/config coverage 92.3% -> 93.1%.

### 2335 — A single-path `git checkout --` is as destructive as a repo-wide one in an uncommitted candidate
- ROOT CAUSE: I rewrote `internal/traceability/ownership.v0.5.0.json` with `json.dumps(indent=2)`, which churned the whole file's formatting, then ran `git checkout -- <that one path>` to undo it. That path carried ~100 lines of UNCOMMITTED prior-round work, and the checkout discarded it along with my formatting churn. The story-final candidate is uncommitted by contract, so HEAD is not a safe restore point for any file in it.
- FIX: Recovered byte-identically from the pre-mutation tree copy at `.temp/TASK-260830-1qf777/r5/tree/`, verified equal to `d583fe8:internal/traceability/ownership.v0.5.0.json` (the round-3 hand-commit that survives in the reflog). Then made the same edit surgically as a text replacement, preserving the file's existing formatting and producing a 20-line diff instead of a 1000-line one.
- DECISION: In an uncommitted candidate, never use `git checkout --` to undo an edit. Edit surgically so there is nothing to undo, and keep a pre-edit copy when a scripted rewrite is unavoidable. The rsync tree copy taken for the mutation harness turned out to be the recovery point; that is luck, not process.
- FINDING: `reviewedOwnershipCanonicalSHA256` is a review gate, not a formatting checksum — adding acceptance-case tests requires moving it deliberately (`1abb92b1…` -> `8885305e…`). That is the gate working as designed and it belongs in the diff, visible.
- SCOPE: internal/traceability/ownership.v0.5.0.json, internal/traceability/traceability.go


### 2310 — A minimal fixture makes every "preserves all members" claim unfalsifiable
- FINDING: Reviewer sweep on CR rev3. Every migration fixture is `minimalValidConfigVersion` (schema, schema_version, host_id, host_name, platform) plus at most `[terminal] backend`. No member with a non-default value ever crosses a migration, so deleting `SafeBoundaryTimeoutSeconds` from `encodeVersion2` (internal/config/writer.go:44), deleting `GracefulStopTimeoutSeconds` from the same literal, or deleting `SafeBoundaryTimeoutSeconds` from `currentWire` (internal/config/writer.go:82) all leave `go test ./internal/config` green.
- FINDING: Confirmed by probe, not inferred: a v1 `safe_boundary_timeout_seconds = 42` reads back as 42 on the clean tree and as 300 (the schema default) under the mutant, with the whole shipped suite passing. SPEC 6.4 "Configuration 2.0.0 retains all Configuration 1 members" is the contract being silently broken.
- DECISION: A default-only fixture is a positive-path-only test wearing a round-trip costume. Preservation must be asserted as a whole-struct equality against a source that sets every decoded member away from its default, so a future added member fails closed rather than defaulting quietly.
- FINDING: Same class, different gate — narrowing `AssessCompatibility` from `compareSemver(source, reader) > 0` to `source.major > reader.major+1` also survives, because the suite only assesses (v3, reader 1.0.0) and (v1, reader 3.0.0). The one-major step is the case SPEC 6.5 names literally ("A v1/v2 binary opening v3 is read-only diagnostic") and it is the only one not tested.
- FINDING: Third survivor — deleting `file.Sync()` in `writeTempFile` (internal/config/migration.go) survives. The *directory* fsync is covered by injected failure, recovery and retry; the *file* fsync half of SPEC 6.4 "fsyncs it and the directory" has nothing. An asymmetry, not an untestable property: `faultMigrationFileSystem.CreateTemp` already returns the raw `*os.File`.
- NOTE: The producer's own 29-mutant sweep was all RED and re-confirmed. All three survivors came from mutants the producer's sweep did not contain. A clean sweep is evidence about the mutants chosen, not about the gate.
- SCOPE: internal/config/{migration,writer}.go, migration_test.go, migration_refusal_test.go
- STATUS: Routed to to-dev. Reviewer sweep log `.temp/TASK-260830-1qf777/review-r3/review-mutants-r3.log`; verdict `TASK-260830-1qf777_review-verdict-rev3.md`.

### 2255 — A sentinel-only refusal assertion can be silently subsumed downstream
- FINDING: `TestMigrateOSRefusesAnUnknownPlatformBeforeTouchingTheFilesystem` asserted only `errors.Is(err, ErrInvalidContext)`. `ResolvePaths` (internal/config/loader.go:359) refuses the same platform later with the same sentinel, so mutant M27 — `MigrateOS` discarding its own `OSInputs` error and continuing with an empty `Inputs` — SURVIVED while the test reported green.
- DECISION: When two guards on the same input share a sentinel, the negative case must name the refusing site, not the sentinel. Test now asserts `(*Error).Operation == "capture OS inputs"` versus the downstream `"validate inputs"`. M27 then died.
- FINDING: This is the "gate present but subsumed downstream" shape, not the usual "gate missing" shape — the gate existed, was reachable, and still proved nothing.
- SCOPE: internal/config/migration_os_test.go:203

### 2250 — Read seam and mutating seam need opposite symlink stances
- ROOT CAUSE: A prior round changed the loader's `os.Stat` to `os.Lstat`, silently refusing a symlinked XDG/Application Support root. The pinned SPEC (28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c) resolves symlinks before checking (SPEC.md:258, :845, :5270, and the :2648 idiom) and never refuses because-symlink — an invented constraint that also breaks a mainstream dotfiles layout.
- DECISION: Split the seam by direction, not by path class. Read side follows symlinks and applies the Section 3.2 value kind to the resolved target (internal/config/loader.go:199). Mutating side does not follow them (internal/config/migration.go `osMigrationFileSystem.Stat`), because an atomic rename replaces the operator's link rather than the document it points at; SPEC is silent about that mutation, so it fails closed.
- FIX: Both stances declared in README.md and pinned in both directions at the real `LoadOS`/`MigrateOS` entries; mutants M23 (widen the mutating seam) and M24 (narrow the read seam — the original defect) both redden.
- FIX: `replaceDurably` now re-inspects the selected file's kind/mode BEFORE any durable write. Previously it refused a symlinked source only after publishing `<config>.bak.<version>`, so a refused migration still mutated the directory. Mutant M26 restores the old ordering and dies on the leaked staging file.
- SCOPE: internal/config/{loader,migration}.go, migration_os_test.go, loader_test.go, migration_refusal_test.go, README.md, internal/traceability/ownership.v0.5.0.json
- STATUS: Resolved. 29/29 mutants dead, zero survivors. internal/config coverage 91.8% -> 92.3%; `MigrateOS` and `(*MigrationError).Error` 0.0% -> 100.0%.

### 2108 — Derived refusal-inventory gates must prove their own coverage
- FINDING: The `internal/config` refusal inventory walked `configError` call sites only. Nineteen `&Error{}` and eighteen `&MigrationError{}` literal sites were invisible to it, so the README refusal guarantee was over-broad while the suite reported green.
- FINDING: Same meta-defect hit all three m0 story leaves in the same review round — a derived gate that scans one form of a construct and is blind to an equivalent duplicate.
- DECISION: A derived gate must assert that the construct set it scans is COMPLETE for the package, not merely derive an inventory. Enumerating one more form by name is not enough.
- FIX: Every refusal now goes through exactly one instrumented package-level constructor per error type — `configError`/`DocumentError`, `loaderError`/`Error` (internal/config/loader.go:661), `migrationError`/`MigrationError` (internal/config/migration.go:385). `deriveRefusalInventory` in internal/config/refusal_inventory_test.go derives error types from `Error() string` declarations, requires exactly one constructor each, and reports any composite literal built outside its constructor. A raw literal, a duplicated constructor, or a fourth error type all redden the suite.
- FINDING: `os.Getwd` does not fail on darwin when the process cwd is unlinked underneath it — getcwd still resolves the old path. The `capture working directory` refusal in internal/config/loader.go has no reachable test on a supported host and is marked subsumed rather than faked.
- DECISION: `config-refusal-subsumed:` exclusions are now ratcheted by `TestRefusalSubsumptionInventoryIsPinned`, so an exclusion cannot be added silently to make a gate green.
- SCOPE: internal/config/{loader,migration}.go, refusal_inventory_test.go, refusal_structure_test.go, migration_refusal_test.go, README.md, internal/traceability/ownership.v0.5.0.json
- STATUS: Resolved. 22/22 mutants died, including four self-coverage mutants (duplicate constructor, bypass literal, never-exercised site in each refusal form). internal/config coverage 83.9% -> 91.8%.

## Round 4 — the final leaf of a Story must NOT be hand-committed

**Finding.** Round 3 exited 0 with every gate green, and Change Request
construction still failed afterwards with
`change_request_base_authority_mismatch`. The defect was that round 3 committed
its work onto the Story branch (`d583fe8`).

**Why it matters beyond this task.** Once a leaf becomes the *final unresolved*
leaf of its Story, its Change Request silently changes kind from `task_delta` to
`story_final`, and only then does `convergeStoryFinalBaseAuthority` apply. That
gate wants the candidate **uncommitted**, with branch tip and HEAD still on the
Story checkpoint, so the CR can be built as the whole trunk-to-worktree Story
delta. Committing is the board's job (`task-board worktree checkpoint` /
`integrate`), not the producer's.

The trap is that the same commit habit is harmless on every earlier leaf and on
a Story whose branch is one commit past trunk (the BUG-260831-3lb2jt shape,
which requires `checkpoint == selected`). It only bites on the last leaf of a
Story that already has checkpoints — i.e. exactly at integration time, after all
the review rounds are spent.

**Detection.** `task-board worktree status --json` — compare
`checkpoint_oid` against `branch_tip_oid`. If they differ on a final leaf, the
CR cannot be constructed.

**Repair.** `git reset --mixed <checkpoint_oid>` in the managed worktree. It
preserves every byte; verify by recomputing the worktree tree with `add -A` into
an alternate index and comparing it to the un-committed commit's tree. Here both
were `9700e40f4bc9787d495864cd97654b2a8864121b`, and `d583fe8` stayed reachable
in the reflog.

**Anomaly worth naming.** The round-3 agent's own gate battery was green and
comprehensive; nothing in `go test`/`vet`/mutation coverage could have caught
this, because the defect lived in Git provenance rather than in the program.
A producer finishing a Story's last leaf should check the workspace record, not
only the test suite.
