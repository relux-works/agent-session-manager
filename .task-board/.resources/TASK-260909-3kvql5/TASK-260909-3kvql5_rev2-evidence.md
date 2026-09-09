# TASK-260909-3kvql5 rev2 evidence: R1/R2 documentation corrections

Tool identity: `/Users/iv/.curator/global/bin/task-board` (`task-board version dev`).
All board/policy commands below ran through that absolute binary, not the bare
`~/.local/bin` build.

## Candidate identity (uncommitted, Story worktree)

- Branch: `task-board/story/STORY-260909-2uobs6` (no producer commit made)
- Base HEAD: `7654d7cadb2c226bfa3db5f23ac355a730285eea`
- `task-board.config.json` sha256:
  `ac85fa77b01f0f0d07a9831965504731b627660abb6ea5983c25809c53134e6b`
  (unchanged from rev1; byte-identical to the approved runtime reference)
- Changed paths (2, policy scope only): `task-board.config.json`, `README.md`
- Rev2 source delta vs rev1: README reviewer wording only. Config untouched.

## R1 — README rescoped to the Codex reviewer ceiling

Old sentence claimed reviewers are restricted to Codex Astra medium "by the
reviewer role ceiling and its medium default". The ceiling at
`spawn.ceilings.codex.roles.reviewer` restricts Codex reviewer effort only:
`reviewer/muse/review` preflight exits 0, admitted through
`spawn.ceilings.muse` with empty recommendations.

New README wording (exact):

> Producers recommend Muse Spark xhigh first; the ceiling restricts Codex
> reviewer effort to Astra medium, with a Codex reviewer medium default, and
> the review workload recommends medium only. Required operator routing
> selects Codex Astra medium for all new reviewers; Muse reviewer admission
> remains possible at provider ceiling level with no recommendations. Empty
> recommendations are advisory, not refusal.

No provider-role exclusion beyond the Codex reviewer ceiling is claimed.

## R2 — outcome corrections

- Coverage header: `6 of 8 rows driven` corrected to `8 of 8 rows driven;
  plus 2 stated bounds`, matching the 8 driven rows + bounds A–B inventory.
- Evidence kinds now explicit: rows 1–5 and 7 are direct preflight
  observations through `task-board q project_config(view=spawn-preflight,…)`;
  rows 6 and 8 are static comparisons (JSON parse, structural diff, file
  inventory); bounds A–B are future/operational lifecycle bounds.
- M2 reclassified: the review-workload-high-first mutant produces a
  recommendation-position assertion (recs index shift after ceiling
  filtering), not an admitted-high behavioral kill; the high pair never
  reached an admission surface, so it does not prove high-effort launch
  refusal. Stated as a limit in the mutant table.
- Blanket `No surviving mutants` claim removed; replaced with: M1/M3/M4
  killed by named behavioral checks, M2 position-only with stated limit, no
  blanket gate-kill claim. Delete-only mutants still not used as evidence.

## Rev2 narrow checks (all run directly, real exits)

| Check | Exit | Result |
| --- | --- | --- |
| `git diff --stat` | 0 | 2 files: `README.md`, `task-board.config.json` |
| `git diff --check` | 0 | clean |
| `python3 json.load(task-board.config.json)` | 0 | candidate JSON parses |
| `cmp` candidate vs approved runtime config | 0 | byte-identical |
| non-policy-field preservation (base vs candidate, `spawn.preferred_agentic_system`, `spawn.ceilings.codex.roles`, `spawn.workload_classes` removed) | 0 | `non-policy-fields-equal=True` |
| preflight `developer/muse/implementation` | 0 | recs `[(muse, muse-spark, xhigh, rank 1)]` |
| preflight `reviewer/codex/review` | 0 | default `gpt-6-astra/medium`, recs `[(gpt-6-astra, medium)]`, ceiling `[{gpt-6-astra:[medium]}]` |
| preflight `reviewer/muse/review` | 0 | ceiling `[{muse-spark:[xhigh]}]`, recs `[]` (R1 proof) |
| preflight `developer/claude/implementation` | 1 | `agent_not_allowed_by_preferred_agentic_system` (allowed: muse, codex) |
| preflight `developer/gemini/implementation` | 1 | `agent_not_allowed_by_preferred_agentic_system` (allowed: muse, codex) |
| preflight `developer/codex/architecture` | 0 | recs `[(high, 1), (medium, 2)]` |

All preflights used `TASK_BOARD_CONFIG` pointing at the candidate canonical
file (read-only queries), not the bootstrap override.

## Limitations (unchanged bounds)

- Historical rev1 validation exits (`gofmt`, `go build`, `go vet`,
  `go test ./...`, `task-board validate`) are preserved as reported in the
  outcome, not rerun; no Go behavior changed and the rev2 delta is
  README wording plus outcome text.
- No broad Go/adversarial suites rerun beyond the checks above; no
  committed Go tests (config + README + evidence scope).
- Actual launches and existing active-run continuity not exercised;
  post-landing default-source preflights must re-prove the real policy.
- No integration, PR publication, landing, tag, hosted CI, or new worker
  from this producer. Corrected revision handed off for Astra medium review.
