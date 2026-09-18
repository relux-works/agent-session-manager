# TASK-260830-2056mm results rev3 — Binding-parser rework (P2-1 + five P3)

Rework run (RUN-260918-03341f) after `TASK-260830-2056mm_review-verdict-rev2.md`
(RUN-260918-acd5a8, claude-opus-5 max): CHANGES REQUESTED with no P1, one P2
class (Binding parser closed-shape arms + identity fork) and five P3. The
reviewer reproduced everything else on its own instruments (v4 payloads,
evidence/identity/recovery/authority behavior, registry re-pin, harness
30/30 twice) — all kept.

Candidate: UNCOMMITTED working tree on branch
`task-board/story/STORY-260830-ptxkqe` atop checkpoint ca1c1d9.
No commit, rebase, or branch operation performed by this run.

Trunk: origin/main == c3aae73 at handoff (verified `git fetch origin main`;
merge-base == origin/main), so no `refresh-candidate` was needed.
`task-board.config.json` is byte-identical to HEAD (verified
`git diff HEAD --quiet -- task-board.config.json`). No validation command added.

Normative source: SPEC.v0.7.0 (unchanged from rev1/rev2).

## Inherited state

The worktree carried the rev2 candidate verbatim at start (`git status`:
15 modified paths + `internal/termbind/`, two leafclose tests untracked;
HEAD == ca1c1d9). Verified against the rework brief before changing
anything; the rev2 review evidence (`review-evidence-rev2.tar.gz` probes
and mutants) was used as the characterization oracle for the fix.

## Finding-by-finding table (finding → change → test that fails without it → evidence)

| Finding | Change | Test that fails without it | Evidence path |
|---|---|---|---|
| P2-1 protocol major | `ParseTerminalBinding` enforces major 1 (`isProtocolMajorOne` over landed `CheckSemver`) — `identity.go` | `TestParseTerminalBindingMemberArms/protocol_version_major_2` (parse succeeds without the arm) | `binding_arms_test.go`; `termbind-pass1/N-binding-protocol-major.log` (KILLED, exit 1) |
| P2-1 extensions | `extensions == {}` via landed strict frame (`isEmptyExtensionsObject`) — `identity.go` | `TestParseTerminalBindingMemberArms/extensions_non-empty` | `binding_arms_test.go`; `termbind-pass1/N-binding-extensions.log` (KILLED, exit 1) |
| P2-1 numbers | `bindingIdentity` refuses JSON numbers at any depth before JCS (`decodeIdentityDocument` + `refuseIdentityNumbers`) — `identity.go` | `TestBindingIdentityRefusesHostileBytes/float` (+5 number subtests) | `binding_arms_test.go`; `termbind-pass1/N-binding-identity-number.log` (KILLED, exit 1) |
| P2-1 nested dups | identity decode refuses duplicate members at every depth — `identity.go` | `TestBindingIdentityRefusesHostileBytes/nested_duplicate` | `binding_arms_test.go`; `termbind-pass1/N-binding-nested-dup.log` (KILLED, exit 1) |
| P2-1 arm battery | 37-subtest member-arm battery, every arm code+detail pinned — `binding_arms_test.go` | each subtest fails if its arm admits (e.g. `/supersedes_non-digest`) | `binding_arms_test.go`; `termbind-pass1/N-binding-supersedes.log` (KILLED, exit 1) |
| P2-1 agreement | verdict-agreement batteries vs landed rules | `TestBindingIdentityVerdictAgreesWithLandedAdmission` (numeric extension value), `TestBindingProtocolMajorAgreesWithLandedTuple` (14 versions) — fail on any divergence | `binding_arms_test.go`; `termbind-verbose.log` |
| P3-1 redundancy | pre-scan harness notes + wrapper doc comments corrected; README bootstrap sentence corrected; LOGBOOK F2 correction appended | `N-prescan-context`, `N-status-prescan` still KILLED — kill measures the Go error type | `termbind-pass1/N-prescan-context.log`, `N-status-prescan.log` |
| P3-2 resumed bound | `EmitResumed` checkpoint/pair binding stated explicitly as caller bound — `emit.go`, TRACEABILITY bounds | `TestEmitResumedCheckpointBindingIsCallerBound` pins the boundary (fabricated appends, fold newest) | `emit_test.go`; `termbind-verbose.log` |
| P3-3 generation | `TestRecoverGenerationBoundBeforeStatusRead` committed with row `N-recover-generation` | fails under the plant (read runs, no refusal) | `recover_test.go`; `termbind-pass1/N-recover-generation.log` (KILLED, exit 1) |
| P3-4a row57/5.2#18 | row 57 + 5.2 gap + README restated as bound | `TestV1ReaderSeesV4AsInert` kept as the payload census | TRACEABILITY row 57; registry 5.2 gap; `tracecheck-sections.log` |
| P3-4b 7.A citation | `TestAdmitDescriptorRefusesForbiddenIdentity` added to `terminal-instance-identity-exact` (+ mirror) | `TestV070RegistryRederivesFromTrunkV060Registry` (literal DeepEqual) | registry + `registry_rederivation_test.go`; `tracecheck.log` |
| P3-4c resolve doc | "every failure" scoped to resolution failures; pre-check arms named defence-in-depth — `resolve.go` | prose (behavior unchanged) | `resolve.go` doc comment |
| P3-4d README | bootstrap sentence states the plain refusal; v1 sentence states the bound | prose | README termbind section |
| P3-5 deferral | restore/terminate-stale follow-up owner + `HandoffFailed` non-applicability recorded here, in LOGBOOK, and in TRACEABILITY bounds | — (record, no behavior) | this file; LOGBOOK rev3 entry; TRACEABILITY bounds |

P3-5 records: executing restore/terminate-stale is deferred to a Story
follow-up task (no board task exists yet — no brief in this Story assigned
it; neither the kkh1an "final leaf" note nor this leaf's old "owner:
terminstance" line named a real owner). The kkh1an `HandoffFailed` note is
NOT APPLICABLE because this leaf wires no `HandoffFailed` reporter (only
tests set the member).

One test edit beyond additions: `TestParseTerminalBindingClosedShape/extra_member`
now uses a string extra (was numeric) with code+detail pins — the new
number walk refuses a numeric extra before the member-set arm can be
measured, so the numeric value masked the `N-binding-members` narrowing
(SURVIVED once, then KILLED after the fix; both states are on record in
this run's pass-1 log, which was re-run for that row).

## Coverage ratio (measured, not planned)

**65 of 66 AC rows driven** through production entries by named committed
tests — matrix in `internal/termbind/TRACEABILITY.md` (rows 1–66 contiguous,
verified by script), reproduced with spec-clause mapping in
`TASK-260830-2056mm_conformance-matrix-rev3.md`. Row 57 is the one stated
bound (payload census — no v1-v3-only reader exists on trunk). All 60
package top-level tests are cited and all PASS (`termbind-verbose.log`:
60 `--- PASS`, 0 FAIL, 0 SKIP). Coverage 83.7% of statements (was 76.9%).

## Mutants (shipped harness, per-plant raw logs)

`PYTHONDONTWRITEBYTECODE=1 python3 internal/termbind/mutant_harness.py` —
36 rows: 34 narrowing + 1 supplementary arm-delete (D-emit-noresolve,
labeled) + 1 harmless SURVIVED control. Pass 1 and pass 2: **36/36
each**, verdict lists byte-identical (`pass1-verdicts.txt`,
`pass2-verdicts.txt`). Per-row verbose logs (raw go output + subprocess
exit) in `termbind-pass1/` and `termbind-pass2/` (72 logs); production
blobs sha256-identical before/after every run
(`termbind-blobs-before.sha`, diff clean). No token-preserving
source-text mutant applies: no gate inspects source text (stated bound,
harness executes the behavioral suite).
Extended batteries re-run in this session: axpane 56/56 once,
terminstance 122/122 once (both untouched by this rework; per-plant logs
carried from rev2 evidence), blobs restored.

## Crash / idempotency

No new durable write in this rework (parser strictness, identity decode,
doc/bound/test-only changes) — crash/idempotency evidence is carried
from rev1/rev2 (real SIGKILL on the attach and lost-create seams,
append-hook abort/replay, byte-identical retry replay). The `N-binding-members`
killer change is test-only and re-verified (row KILLED, suite green).

## Story-close items

- Registry (`ownership.v0.7.0.json`, adopted): 5.2 gap reworded to the
  row-57 bound; eight test citations added across four cases
  (binding-exact +5 arm/agreement tests, identity-exact +descriptor test,
  resumed-exact +caller-bound pin, recovery-exact +generation witness);
  mirrors in `registry_rederivation_test.go` updated identically.
- Digest re-derived from the registry as it stands (transient in-package
  probe calling `decodeOwnershipRegistry` + `json.Marshal` + sha256,
  removed after):
  `DERIVED-DIGEST: b09b5b9f881a68969dc67dff54e1a31c3a82b7320e25fdc28ef9fce96b65bed5 (canonical bytes: 167094)`
  — equals the pin in `traceability.go`.
  Derivation command: `go test ./internal/traceability/ -run TestZZDerivePin -count=1 -v`
- `go run ./internal/traceability/cmd/tracecheck` GREEN:
  `traceability ok: contracts=64 normative_sections=36 acceptance_cases=147
  fixtures=33 compatibility_contracts=55 assigned_scopes=0`
  `section coverage: bindings=69 full=2 partial=9 sliver=9 unevidenced=45
  unmeasured=4 unowned=7 clauses_discharged=63/574`
- Section-scoped measured ratios (quoted verbatim; exit 1 by design):
  4.1 → 1/5 sliver; 4.B → 1/12 sliver; 4.D → 1/3 sliver; 5.2 → 3/18
  sliver; 7.A → 1/2 partial. (`tracecheck-sections.log`)
- README: termbind section corrected (bootstrap plain refusal, v1 bound,
  34-narrowing count); "Measured coverage of this repository" unchanged
  (figures identical; `TestREADMEMeasuredCoverageMatchesTracecheckReport`
  executed and PASS in the traceability suite run).
- LOGBOOK: one newest-first rev3 entry with the F2 correction paragraph
  (appended, history preserved).

## Validation (this run)

Reran myself (logs in `TASK-260830-2056mm_producer-evidence-rev3.tar.gz`):

- `go test ./internal/termbind/ -count=1 -v` → 60 PASS / 0 FAIL / 0 SKIP
- `go test ./internal/termbind/ -cover` → 83.7%
- `go test ./... -count=1` → 41/41 ok
- `go test -race -count=1` in 4 bounded chunks (story + rest1–3) →
  41/41 ok, 0 DATA RACE
- termbind harness 36/36 twice (verbose per-plant logs both passes);
  axpane 56/56, terminstance 122/122 once each
- `go run ./internal/traceability/cmd/tracecheck` + all five
  section-scoped runs; `go test ./internal/traceability/...` green
- `gofmt -l` clean; `go vet ./...`, `go build ./...`, GOOS=linux/windows
  builds, GOOS=windows vet, `git diff --check` clean
- `cataloggen -adopted -output internal/catalog/catalog_gen.go -check` → exit 0
- `task-board validate` → exit 0 (182 pre-existing issues on other
  elements, 0 mentioning this task)
- killer-presence + row-contiguity script checks (harness refs 27/27
  present, matrix 60/60 present, rows 1–66 contiguous)
- `PYTHONDONTWRITEBYTECODE=1` throughout; 0 `__pycache__`/`.pyc`

Accepted from already-attached rev1/rev2 evidence (bytes unchanged):
crash-seam logs, append-hook logs, rev2 per-plant logs for the untouched
axpane/terminstance rows.

## Rejection-pattern self-check

(a) Literals from spec text asserted at entries (pinned codes, `1.0.0`
major selection, `{}` extensions, 1..256/512 bounds, digest grammar) —
the battery pins code+detail per arm, never a production constant.
(b) Every rule driven through the entry the matrix names; the
Build/identity sides each carry narrowings (member arms + direct-identity
walk rows). (c) Landed owners composed (semver/frame/surrogate to
environ, UUIDv7/digest/timestamp to scalar, backend/generation/admission
to terminalbackend, append/fold/query to sessrepo/sessstate/sessquery);
the two mirrors (identity decode+walk, major selection) carry
verdict-agreement batteries against the landed rules. (d) Inputs driven,
effects asserted; row 57 is now labeled a census, not a drive.
(e) 34 narrowing rows + labeled arm-delete + applied SURVIVED control,
one raw log per plant per run; the two pre-scan rows honestly labeled as
type-measuring. (f) Every "composes/never" sentence backed by a
failing-without test or stated as an explicit bound (pre-scan
redundancy, resumed caller bound, v1 bound, restore/terminate-stale
deferral). Reported ratio (65/66) is measured on this tree, not planned.
