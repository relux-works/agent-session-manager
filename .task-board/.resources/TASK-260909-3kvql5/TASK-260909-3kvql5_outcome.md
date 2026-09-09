# TASK-260909-3kvql5 outcome: admit Muse producers + medium Astra reviewers

## Provenance (explicit-config bootstrap)

This run was spawned through the supported explicit `TASK_BOARD_CONFIG` source
(`.../.temp/TASK-260909-3kvql5/approved-runtime-config.json`, immutable
operational input, NOT modified) because the canonical config was still
exclusive-Codex. The approved file served as policy reference only. Root
runtime, control repository and board authority remain this AX repository.
Canonical preflights below were run with `TASK_BOARD_CONFIG` overridden to the
worktree candidate file (read-only queries); post-landing preflights on the
default source must re-prove the real policy.

## Candidate identity (uncommitted, Story worktree)

- Branch: `task-board/story/STORY-260909-2uobs6` (no producer commit made)
- Base HEAD: `7654d7cadb2c226bfa3db5f23ac355a730285eea`
- `task-board.config.json` sha256: `ac85fa77b01f0f0d07a9831965504731b627660abb6ea5983c25809c53134e6b`
- Changed paths (2, policy scope only): `task-board.config.json`, `README.md`
- Candidate `task-board.config.json` is byte-identical to the approved runtime
  config (`diff` clean); `git diff` vs HEAD is confined to the authorized delta.

## Change (authorized delta only)

1. `spawn.preferred_agentic_system`: `{"exclusive":"codex"}` ->
   `{"mixed":["muse","codex"]}`.
2. `spawn.ceilings.codex.roles.reviewer` added: Astra medium-only entries with
   `default_model gpt-6-astra` / `default_reasoning_effort medium`.
3. `spawn.workload_classes`: `muse/muse-spark/xhigh` inserted rank 1 in every
   producer class (unified, implementation, mechanical, debugging, testing,
   research, documentation, migration, operations); `review` reduced to
   Codex Astra medium only; `architecture` unchanged (Codex high, medium).
4. README model-routing paragraph aligned (Muse-first producers, ceiling
   restricts Codex reviewer effort to Astra medium with Codex reviewer medium
   default, required operator routing selects Codex Astra medium for all new
   reviewers, Muse reviewer admission remains possible at provider ceiling
   level with no recommendations, empty recommendations advisory not refusal,
   architecture high-then-medium, mechanical/documentation/operations Muse
   xhigh > Codex medium > Codex high, mixed allow-set limited to Muse+Codex,
   retained Claude ceiling inactive, existing runs preserved); preflight
   examples updated to `developer/muse/mechanical` and
   `reviewer/codex/review`. Rev2 wording; no provider-role exclusion is
   claimed beyond the Codex reviewer ceiling.
5. Preserved byte-identical: `mode`, `local`, `version_control`, `features`,
   `spawn.enabled`, `max_parallel`, all provider ceiling entries/models,
   `worktree_isolation.validation` commands, `.github` (no CI/delivery change).

## Command exits (all run directly, exit codes real)

| Command | Exit | Result |
| --- | --- | --- |
| preflight battery (8 checks, production entry point `task-board q 'project_config(view=spawn-preflight,…)'`) on candidate | 0 | 8/8 PASS |
| mutant harness (4 mutants x full battery + static checks) | 0 (harness); nonzero batteries as expected | M1/M3/M4 killed; M2 position-only, not a refusal kill (see table) |
| `git ls-files -z '*.json' \| xargs … json.load` | 0 | all JSON parse |
| `git diff --check` | 0 | clean |
| gofmt gate (`gofmt -l` over tracked+untracked go files) | 0 | clean |
| `go build ./...`, `go vet ./...` | 0 | pass |
| `GOOS=linux/windows GOARCH=amd64 go build ./...` | 0 | pass |
| `go test ./internal/canonicaljson -run TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget` | 0 | PASS (config-coupled: validation commands preserved) |
| `go test ./... -count=1` | 0 | all 23 packages ok |
| `go test ./... -cover -count=1` | 0 | ok (e.g. secprim 94.4%, secconftest 92.6%) |
| `task-board validate` | 0 | 233 pre-existing board ledger-mirror lints, unchanged by this edit (233 before and after; config adds none) |

## AC coverage: 8 of 8 rows driven; plus 2 stated bounds

Evidence kinds: rows 1–5 and 7 are direct preflight observations through the
production entry point `task-board q project_config(view=spawn-preflight,…)`;
rows 6 and 8 are static comparisons (JSON parse, structural diff, file
inventory), not behavioral refusals; bounds A–B are future/operational
lifecycle bounds not drivable in this producer run.
1. Developer Muse xhigh preflight succeeds — P1 (`developer/muse/mechanical`
   rank 1 `muse-spark/xhigh`) and P4 (default routing Muse-first). Direct preflight. Driven.
2. Reviewer Astra medium preflight succeeds — P2 (admitted, recs medium-only). Direct preflight. Driven.
3. Review role default + recommendations prefer medium — P2 (`resolved_role_default`
   `gpt-6-astra/medium`) + N3 (recs exactly `[medium]`, len 1). Direct preflight. Driven.
4. Codex reviewer high not admitted — N1 (`resolved_role_ceiling` models exactly
   `[{gpt-6-astra:[medium]}]`; no `reasoning_effort` arg exists on preflight, so the
   ceiling object is the refusal surface). Direct preflight observation of the
   Codex reviewer ceiling, not an actual high-effort launch attempt. Driven.
5. Other providers excluded — N2a/N2b (`agent=claude`, `agent=gemini` refused with
   `agent_not_allowed_by_preferred_agentic_system`; claude refused despite its retained
   ceiling entry, proving ceilings do not imply admission). Direct preflight. Driven.
6. JSON + focused config checks pass — `json.load` over all `*.json` plus
   candidate-vs-approved structural comparison. Static comparison. Driven.
7. Orchestrator Astra high/medium usable — P3 (`developer/codex/architecture`
   recs high rank 1, medium rank 2). Direct preflight. Driven.
8. No unrelated runtime/hosted CI — `git diff --stat` (2 files), non-policy-field
   assertion script, `.github` untouched. Static comparison. Driven.
- Bound A (future lifecycle, not drivable here): independent Astra-medium review +
  signed exact-head PR landing — owned by Primary after handoff.
- Bound B (operational): existing active runs unchanged — no run was created,
  stopped, or edited; config-only change.

## Mutant table (behavioral battery is the gate; static = file-level assertions)

| Mutant | Narrowing | Named failing checks | Outcome / bound stated |
| --- | --- | --- | --- |
| M1 reviewer ceiling entries medium+high | admits exactly one extra effort | N1-reviewer-ceiling-medium-only | KILLED. Survival would mean high reviewers admittable. |
| M2 review workload high inserted first | recommendation ordering only (high pair filtered by Codex reviewer ceiling before recs) | P2 position assertion: recs index shift (`review[1]`) | NOT a behavioral refusal kill: the high pair never reached an admission surface, so this is a recommendation-position assertion with the stated limit that it does not prove high-effort launch refusal. |
| M3 mixed allow-set +claude | admits exactly one excluded provider | N2a-claude-excluded (claude admitted, refusal gone) | KILLED. Survival would mean a third provider routable. |
| M4 reviewer ceiling high-only, `medium` default token kept | preserves searched-for token, changes behavior | P2, N1, N3 fail while static check still reports default `gpt-6-astra/medium` | KILLED. Proves the static token check is insufficient and the behavioral (preflight) suite is required. |

M1, M3, and M4 are killed by the named behavioral checks above; M2 is a
recommendation-position assertion with the stated limit, not a refusal kill,
so no blanket gate-kill claim is made. Delete-only mutants were not used as
evidence.

## Policy observations

- `reviewer/muse/review` is not refused at ceiling level (muse ceiling has no role
  restriction) but returns empty recommendations (`recs: []`) because the review
  class holds only the Codex medium pair, filtered when targeting muse.
- `task-board validate` reports 233 pre-existing ledger-mirror issues with exit 0;
  count identical before/after this change.

## Limitations

- Candidate preflights used a read-only `TASK_BOARD_CONFIG` override to the
  candidate file because the live session still sources the approved runtime
  config; post-landing preflights on the default source must re-prove the policy.
- No committed Go tests: scope limits this task to config + README + evidence and
  no Go behavior changed; the 8-check preflight battery plus mutant harness are
  the tests. Rev1 probes persist under worktree `.temp/TASK-260909-3kvql5/`
  (gitignored); rev2 narrow checks persist under worktree
  `.temp/TASK-260909-3kvql5/rev2/` (gitignored) and are attached as
  `TASK-260909-3kvql5_rev2-evidence.md`.
- Full suite ran as `go test ./... -count=1` (exit 0) and `-cover` (exit 0); a
  separate full `-v` rerun was not repeated.
- Repo `LOGBOOK.md` intentionally untouched to keep the delivery diff policy-only;
  provenance recorded here instead.

## Rev2 corrections (RUN-260909-e61106 rework)

- R1: README reviewer sentence rescoped to the Codex reviewer ceiling with the
  required operator-routing / Muse-admission / advisory-recs wording; no
  provider-role exclusion is claimed beyond that ceiling.
- R2: coverage header corrected to 8 of 8 rows driven plus 2 stated bounds,
  with direct-preflight vs static-comparison vs lifecycle-bound classification;
  M2 reclassified as a recommendation-position assertion (not a refusal kill)
  and the blanket no-survivor gate claim removed.
- Config unchanged from rev1: candidate `task-board.config.json` remains
  byte-identical to the approved runtime reference; rev2 delta is README
  wording plus this outcome correction. Narrow rev2 checks (diff, JSON,
  policy preflights) are recorded in `TASK-260909-3kvql5_rev2-evidence.md`.
  Historical rev1 validation exits above are preserved as reported, not rerun.
