# TASK-260830-1qf777 — review verdict, CR revision 5

**Verdict: ACCEPTED.**
Reviewer run `RUN-260901-fba8aa` (claude-opus-5). Change Request
`CR-TASK-260830-1qf777-5`, element `TASK-260830-1qf777`, integration scope
`STORY-260830-jeaivu`.

## Candidate identity, verified rather than trusted

The worktree tree was re-derived independently
(`git read-tree HEAD` into a scratch index, `git add -A`, `git write-tree`)
and equals the declared candidate tree
`613bb21abae3caa28a800b2222a1d53388cbf379` on a base of
`48db30b59e5e1bbc5e0cf73ec2e0e0eec3d215d1`. It was re-derived again after every
mutation batch and after the traceability probe; it is unchanged.

## What actually moved in revision 5

The producer's round-6 note claims no production code changed. That claim is
verified, not accepted: splitting the rev4 and rev5 patch resources per file and
comparing the hunks byte-for-byte gives

- identical: `internal/config/loader.go`, `migration.go`, `schema.go`,
  `validation.go`, `writer.go`, `go.mod`, `go.sum`, and every test file except
  the three below
- changed: `README.md` (one added paragraph), `internal/config/loader_test.go`
  (a comment block only), `internal/config/migration_os_test.go` (one added
  test), `internal/traceability/ownership.v0.5.0.json` (three added owner
  entries), `internal/traceability/traceability.go` (the reviewed digest
  constant)
- new: `internal/config/loader_home_test.go`

So revision 5 is an evidence-only revision on top of a production tree the rev4
review already found free of defects. The review therefore concentrates on
whether B1 is genuinely closed, and independently re-attacks the acceptance
gates rather than inheriting the prior verdict.

## B1 — self-minted home-capture evidence: CLOSED

The rev4 finding was that the only test touching the deferred `homeDirError`
hand-set that unexported field on a fixture, so it constructed the state it
claimed to observe. Both rev4 survivors are now dead, and neither dies by
accident:

| rev4 survivor | now | killed by |
| --- | --- | --- |
| `homeDirError = nil` after the capture | RED | `TestLoadOSCarriesTheRealUserHomeFailureAtEveryHomeDerivedClass` |
| `home = ""` after the capture | RED | `TestLoadOSDerivesPlatformDefaultsFromTheRealCapturedUserHome` |

Two further attacks on the same seam also die: substituting a constant home
(A03) and substituting a self-minted cause for the real `os.UserHomeDir` error
(A06). The state is reached the way production reaches it — the cases set or
clear the exact variable `os.UserHomeDir` consults and let the real `OSInputs`
capture run; nothing in `loader_home_test.go` assigns `Inputs.HomeDir` or
`Inputs.homeDirError`. The one test that still hand-sets the field,
`TestResolvePathsDefersButPreservesHomeLookupFailure`, is now labelled as the
injected half only and names the two real-entry cases that pin the other half —
which is the honest disposition the finding asked for, not a widened fixture.

`MigrateOS` is covered as the second consumer of the same capture:
`TestMigrateOSMigratesTheHomeDerivedConfigurationAtTheRealProcessEntry` places a
v1 document at the Section 3.2 home-derived default and lets the captured home
be the only thing selecting the file the durable mutation rewrites. Function
coverage confirms `MigrateOS` and `(*MigrationError).Error` are both at 100%.

## Independent attack on the acceptance criteria

36 reviewer-written mutants, one edit at a time, whole-package
`go test ./internal/config -count=1` with no `-run` mask, tree restored and
re-verified green between every mutant. **31 RED, 4 survivors, 1 positive
control green.** Full log: `TASK-260830-1qf777_reviewer-mutation-sweep-rev5.log`;
harness: `TASK-260830-1qf777_reviewer-mutants-rev5.py`.

Every named acceptance gate dies, and dies under a *narrowing or widening*
mutant rather than delete-only:

- **backup** — the existing-backup integrity gate dies clause by clause: the
  content clause (B08) and the owner-only-permission clause (B07) each die
  separately on `TestMigrateRefusesExistingBackupThatIsNotTheExactOwnerOnlySource`.
  Backing up the replacement instead of the original (E02) dies at the real
  `MigrateOS` entry.
- **migration** — dropping any wire member dies on the derived preservation
  property (B11, B12); the target vocabulary (B16), the disclosure vocabulary
  at a supplied value (B21), the pre-durable source-kind refusal (B18), the
  per-file fsync (B13) and the rollback rename (E01) all die.
- **unknown-field refusal** — removing `DisallowUnknownFields` dies (B04).
- **bounds** — proven in both directions at the production entry: widening
  `max_parallel_chunks` past 32 (B01), narrowing its floor to 2 (B02), and
  widening the terminal safe-boundary ceiling (B03) all die.
- **duplicate backend rejection** — narrowing both duplicate `backend_id`
  rejections dies (B05, B06).
- **read-only downgrade** — the rev3 survivor (narrowing to a two-major gap)
  is RED (B09), as is narrowing the migration downgrade refusal (B10) and
  widening the known-reader gate at a supplied value (C04).
- **symlink seam split** — both directions hold: widening the mutating seam
  (B14) and narrowing the read seam back to the original F1 defect (B15) are
  each RED against their own test.

I also attacked the two derived gates themselves rather than reading them.
A bypass composite literal (C01), a second byte-identical constructor (C02) and
a brand-new never-exercised refusal site (C03) all redden. The subsumption
escape hatch holds in both directions: deleting a declared marker reddens (D01),
and — the attack that matters — attaching a self-declared subsumption marker to
a synthetic never-exercised site does **not** silence it (D02). A producer
cannot buy its way out of the inventory with a comment.

The traceability gate fails closed on a fabricated owner: renaming one ownership
declaration makes `tracecheck` fail on sections 6.3/6.4/6.5/17.1/17.2 naming the
absent declaration. The three ownership entries revision 5 adds are exactly the
three tests the producer claims, and all three exist.

## Survivors, reported as survivors

Four mutants survived. None is a finding, and each is reported with its reason
rather than folded into the RED count.

1. **B19** — `!ConfigPresent() || !decoded` narrowed to `&&`. Equivalent on every
   reachable state: a successful `Load` returns either (`configuration` non-nil
   **and** `configPresent` true) or (nil **and** false), so the two operands are
   never independently true. Defence in depth over one reachable condition.
2. **B17 / B20, with control B22** — adding a brand-new member to a closed
   vocabulary that no case supplies is not observable by any finite sampling
   suite. Every mutant that admits a value the suite *does* supply dies (B21,
   C04, B23). Control **B22** shows the identical arbitrary widening also
   survives on `directory.mode`, a sibling vocabulary outside this delta, so this
   is the existing package-wide standard rather than something this change
   introduced. Killing the class needs the vocabulary derived from a pinned SPEC
   literal and asserted for set equality. Recorded as an observation for a future
   leaf; not blocking here.

## Gates run by this reviewer

| Gate | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` | 0 (empty) |
| `go test ./... -cover -count=1` | 0 (internal/config 93.7%) |
| `go test ./... -count=1` (post-sweep) | 0 |
| `go test ./internal/config -race -count=1` | 0 |
| `go mod verify` | 0 |
| `git diff --check` | 0 |
| `tracecheck` global | 0 |
| `tracecheck -section 3.2 6.1–6.5 17.1 17.2 17.4` | 0, assigned_scopes=9 |
| `go test ./internal/config -count=1` under `env -i` with `HOME=` | 0 |

The hermetic run matters specifically for this revision: the new cases
manipulate `HOME` and must not depend on the developer's real home. They do not.

## Definition of Done

- [x] Production entry points implement the scoped deliverable — backup,
      migration, unknown-field refusal, bounds, duplicate backend rejection and
      read-only downgrade all attacked at `Migrate`/`MigrateOS`/`Load`/`LoadOS`/
      `AssessCompatibility`, not at helpers
- [x] Positive, negative, compatibility and recovery tests pass; logs attached
- [x] README, traceability and capability claims verified against the code; the
      one added README paragraph is accurate and advertises no CLI, doctor or
      backend capability
- [x] Gating behaviour covered by negative tests that fail when the gate admits
      what it must reject, with the production call site named
- [x] Lint clean, build not broken
- [x] Implementation matches AC; solution fits the package architecture
- [x] Gates attacked rather than read — 36 independent mutants

No production defect was found on any probe. No stop-the-line condition: no
external blocker and no human-only decision. Accepted.
