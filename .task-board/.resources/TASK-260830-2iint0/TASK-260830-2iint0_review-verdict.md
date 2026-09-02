# TASK-260830-2iint0 — Reviewer Verdict

- Change Request: `CR-TASK-260830-2iint0-3` revision `3`
- Base OID: `48db30b59e5e1bbc5e0cf73ec2e0e0eec3d215d1`
- Candidate tree OID: `add84fb96f149b6ef2d312b335e361b71d8bdcd8`
- Reviewer run: `RUN-260901-5f8771` (not goal-bound)
- Verdict: **ACCEPTED**

Working-tree identity was proven, not assumed: `git write-tree` against an
isolated temporary index returned `add84fb96f149b6ef2d312b335e361b71d8bdcd8`,
exactly the candidate tree. All mutation probes below were reverted and the
identity re-verified afterwards; no review residue remains.

## Scope and boundary

Spec authority checked against the pinned source: local
`agent-session-manager-spec` HEAD is `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`,
the exact commit named in the task scope. The delta implements §3.2 platform
path selection plus the §6.1 configuration-file *selection* rule.

The boundary claim is honest and I verified it does not overclaim. §6.1's
unknown-top-level-key refusal and secret-field refusal require versioned TOML
decoding, which is owned by sibling tasks `TASK-260830-17suox`
(versioned readers/writers) and `TASK-260830-1qf777` (unknown-field refusal,
bounds, downgrade), both still `to-dev`. The traceability registry binds only
`section:3.2`, and `TestRunDoesNotOverclaimWholeConfigLoadingSection` pins the
refusal for `-section 6.1`. That is the correct shape: an unclaimed section
fails the gate rather than being silently absorbed.

Normative conformance spot-checked row by row against §3.2: the five-row
override registry, empty-value-is-unset, absolute-before-use, the macOS /
Linux+WSL2 / native-Windows default tables, XDG variables as *inputs* to
defaults rather than additional `AX_*` overrides, and the "unknown `AX_*` is
ordinary process environment and MUST NOT be interpreted" rule. All match.

## Gates attacked, not read

35 mutants applied to `internal/config/loader.go`; 1 was a compile failure
re-run in corrected form, 1 was my own equivalent (no-op) mutant. Of the 33
real mutants, **31 were killed** by the suite. Full log:
`TASK-260830-2iint0_review-mutation-report.txt`.

Killed classes included both delete-only and narrowing shapes:

| Attack shape | Example mutants | Result |
| --- | --- | --- |
| Precedence inversion | flag/env order swapped | killed by 4 tests |
| Bound narrowing | empty env/flag value treated as set | killed |
| Gate deletion | root-kind check uncalled; regular-file check removed; unknown-class validation removed; nil `Stat`/`ReadFile` guard removed; platform validation removed | killed |
| Gate narrowing | root-kind check restricted to one class; kind check admits symlinks; UNC/traversal clamp instead of refusal | killed |
| Absence/failure conflation | any stat error treated as absence; parent read failure collapsed into parent-not-found; read error swallowed into an empty document | killed |
| Fail-open defaults | Linux `XDG_RUNTIME_DIR` falling back to temp; Windows `APPDATA` falling back to home | killed |
| Wrong normative default | macOS state loses `state` leaf; Linux state falls back to `share`; Windows leaves collapsed; Windows join uses POSIX separator | killed |
| Provenance leak | rendered error echoes the wrapped OS detail (selected path) | killed |
| Aliasing | `Document()`/`All()` return internal state | killed |

Two survivors, both proven to still fail closed — reported as observations,
not defects:

- Dropping the drive-relative Windows refusal (`C:foo`, `Z:`) leaves the suite
  green, but the class is **provably subsumed**: the joined form always yields
  a path segment containing `:`, which `scalar.invalidWindowsSegment` rejects.
  Probed directly — every drive-relative candidate is refused either way. The
  subsuming check is named here rather than left implicit.
- Loosening the UNC `configParent` segment bound changes only the error
  *identity* for a share-root config path; probed, the load still refuses
  (`resolve selected file parent`). Unreachable in practice, because an
  existing share root is rejected earlier by the regular-file kind check.

## Registry completeness attacked directly

The DoD's derived-completeness rule was tested, not assumed. I applied a
**coordinated two-sided drop** of the `AX_CACHE_DIR` row from both the
production `overrideRegistry` and the test's expected list — the exact edit a
duplicate-hand-written-list pin would miss. **It reddened**: five independent
per-class expectation tables plus the
`len(paths.All()) == len(OverrideRegistry())` assertion caught it. Completeness
is cross-pinned across multiple independent sites rather than by a single
mirrored list.

It is not literally derived from a generated source, because no derivable
source exists yet: `catalog.v0.5.0.json` has no path-override family, and
`SPEC.md` is pinned by digest rather than vendored. Extending the reviewed
catalog metadata is the repo's established pattern (`internal/canonicaljson`
derives from `catalog.Current().SelfIdentities`) and is the right follow-up,
but it would require touching the catalog schema and its
`reviewedMetadataCanonicalSHA256` — outside this leaf. Recorded as follow-up,
not rework.

## Traceability evidence attacked

The ownership registry could have been self-minted (the producer changed both
`ownership.v0.5.0.json` and the pinned `reviewedOwnershipCanonicalSHA256` in
one delta), so I attacked the gate rather than trusting the pair:

| Forgery attempt | Result |
| --- | --- |
| Test declaration that does not exist in the named file | refused |
| Production declaration that does not exist | refused |
| `section:3.2` owner pointed at a bogus symbol | refused |
| Silently dropping a test from `AC-PATH-001` | refused on projection digest |

The gate verifies declarations actually exist in the named files and pins the
semantic projection. Ownership file restored byte-identical afterwards
(sha256 `bb0e49fd20c8a730120446700e41f387b773a91a805cfa04644321bb5cf9834a`).

## Commands run by this reviewer

| Command | Result |
| --- | --- |
| `go test ./... -count=1` | all 9 packages ok |
| `go test ./... -cover -count=1` | `internal/config` 86.6%; no package regressed |
| `go run ./internal/traceability/cmd/tracecheck -section 3.2` | `assigned_scopes=1`, ok |
| `go vet ./...` | clean |
| `gofmt -l .` | clean |

README commands were executed as written and reproduce the documented output.

## Observations for follow-up (non-blocking)

1. **Derived registry completeness.** Add a path-override family to the
   reviewed catalog metadata and derive `overrideRegistry` from
   `catalog.Current()`, matching the `internal/canonicaljson` precedent. Today
   completeness is cross-pinned but not derived.
2. **Unreachable FileInfo shape.** `TestLoadRefusesEveryNonRegularConfigKind/symlink`
   injects `fs.ModeSymlink`, which `os.Stat` can never return because it
   follows links. The subtest is harmless but proves nothing about production
   behavior for a symlinked config path. §3.2 does not forbid a symlink
   resolving to a regular file for the operator-supplied config value, so this
   is a test-shape note, not a missing refusal.
3. **README phrasing.** "the dependency-injected production entry used by
   deterministic tests and alternate launchers" is forward-looking; no
   alternate launcher exists yet. The Project Status section still disclaims
   operator command behavior, so no operator-visible capability is
   over-advertised, but the wording could be tightened when the CLI lands.

## Definition of Done

Met. Production entry points (`Load`, `LoadOS`, `ResolvePaths`) implement the
scoped deliverable; positive, negative, compatibility, and recovery behavior is
covered with attached logs; README and traceability evidence are updated
without unsupported claims; declared bounds fail closed in both directions;
gates were attacked individually and the suite reddened for every non-subsumed
clause; no test depends on ambient environment beyond `t.Setenv` and an
explicit runtime-probed platform.

No `commit_ack` supplied — this is a reviewer-archetype run. The orchestrator
owns the checkpoint/integration and the final `done` transition.
