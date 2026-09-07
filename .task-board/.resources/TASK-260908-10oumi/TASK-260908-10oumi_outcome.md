# TASK-260908-10oumi producer evidence

Scope: README.md and task-board.config.json only, uncommitted managed Story worktree at checkpoint 7aa151a9c31071bfab190fd8ae8259f502f2ebbc. No commits, branch changes, integration, runtime-record edits, or new spawns performed by this producer. Orchestrator owns reviewer routing and signed exact-head PR delivery.

Applied the candidate routing delta: exclusive Codex provider admission; exact gpt-6-astra medium/high pairs; all 11 workload classes recommend only those pairs. Mechanical/documentation/operations rank medium first; review and remaining classes rank high first. Retained unrelated configuration, including dormant Claude/Muse ceiling definitions, fast mode, version control, features, isolation and every validation command. README documents effective admission, recommendation order, retained work, and preflight commands.

## AC coverage and bounds

**3 of 5 AC rows driven through the production preflight CLI; 0 of 5 driven by newly committed tests.** No test source is committed because the explicit scope permits only config and matching README edits. The attached task-scoped verify.py contains named executable checks. It is evidence, not a shipped test suite or proof of an actual agent launch. All preflights explicitly set TASK_BOARD_CONFIG to the modified worktree file rather than relying on the launcher's temporary config.

| AC row | Evidence / production call site | Bound |
| --- | --- | --- |
| Codex-only admission | reject_claude, reject_muse, reject_fable; task-board q project_config(view=spawn-preflight, role=reviewer, agent=PROVIDER, workload_class=review) | Each command actually exits 1 with agent_not_allowed_by_preferred_agentic_system; expected refusals, not green commands. No real spawn attempted. |
| Exact Astra medium/high pairs | preflight_review and every workload; task-board q project_config(view=spawn-preflight, role=ROLE, agent=codex, workload_class=CLASS) | Exact admitted_pairs projection checked. Launch path itself and per-role overrides beyond developer/reviewer not exercised. |
| All recommendations Astra | preflight_CLASS for all 11 configured classes, same production CLI | Checks pair identities and order independently of config values. |
| README agreement | Manual diff review against candidate and effective preflights | Documentation review, not production behavior. |
| Focused validation and review | verify.py, configured repository gates, git diff review | Independent reviewer verdict and PR delivery belong to orchestrator after handoff. |

Existing work preservation is bounded by an initially clean managed worktree, a two-file resulting diff, unchanged unrelated parsed config, and no run-control mutations. This does not claim a behavioral test of an already-running Muse process.

## Negative / mutant evidence

The external task-board implementation is unchanged; policy config is tested through its actual production query surface. No source-text gate is introduced, so the source-token-preserving mutant requirement is inapplicable. The following temporary mutants keep the policy gate present and are outside the managed worktree.

| Mutant | What it narrows the gate to | Named test that fails | Survivor bound |
| --- | --- | --- | --- |
| admit_astra_low | Adds exactly one forbidden model/effort pair, gpt-6-astra:low | preflight_review: exact_pairs | Killed; preflight admission projection only, no launch claim. |
| admit_claude | Adds exactly one forbidden provider, Claude, while keeping provider filtering | preflight_review: provider_admission | Killed; provider allow-set projection only, no launch claim. |

No survivors among these two mutants. verify.py exits 0 only after both expected assertion failures occur. Negative provider commands retain their actual exit 1; positive preflights exit 0. focused-02.log and preflights.json retain evidence.

## Validation

See gate-results.jsonl for exact configured commands and real exit codes, gate-NN.log for individual output, readiness-01.log for tool versions, and inspection-notes.md for exploratory read failures. Each gate runs as its own subprocess with pipefail, without tee; the harness records its actual status. Native target: macOS arm64, Go 1.25.5; repository-required cross builds: Linux amd64 and Windows amd64. No iOS target exists.

No Go product behavior changed and no unrelated Go tests were added. The existing full suite, race suite, coverage suite, build/vet/gofmt, fuzz runs, traceability/catalog checks, JSON parsing, board validation, and diff check are the repository-configured validation contract. Handoff may independently repeat these gates in its managed candidate checkout.

Checklist interpretation: implementation means the two-file config/documentation change; test and negative-evidence items refer to the attached scoped CLI checks with the committed-test and launch bounds above. Source-text-gate item is not applicable. No significant new product anomaly required a logbook entry. Review is pending.

All 26 configured validation commands were run by this producer and exited 0. The focused verification harness exited 0 on both executions; the revised second execution uses independent expected recommendation orders. No validation result was accepted from another run.
