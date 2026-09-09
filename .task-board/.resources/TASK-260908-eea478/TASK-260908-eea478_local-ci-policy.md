# Local-only CI policy

User instruction on2026-09-08: run CI only locally and disable automatic hosted triggers. Effective repository configuration now disables AX CI346519002, spec validation340251033, task-board CI236218136 and release236218137. GitHub read-back confirms disabled_manually. The existing experimental workflow352147790 is also disabled. Dependency Graph is an unrelated security/dependency service and was not changed.

Existing YAML definitions remain as a reproducible gate inventory; no CI workflow is enabled, dispatched, or re-enabled. This is a repository-configuration task with no source code delta. Review-none reflects that bounded reversible operational scope, not a downgrade of implementation review. Primary goal revision13 preserves the full AX scope and establishes local-only CI. All new agent briefs inherit it.

For delivery, run the existing complete applicable checks locally from an exact clean PR head and record SHA, platform, toolchain, environment, commands, exit codes and logs. Local failure blocks delivery. Keep independent code review, configured human author and signatures, exact-head equality and ordinary fast-forward landing. Do not forge hosted checks or report cancelled workflows as success. Remote protection remains authoritative; any rejecting rule requires an explicitly recorded owner configuration change.

The pending source PR182 now runs the12 original workflow steps locally on signed bb7fb659ffa185825f0272c5a8408c7ce02a910b. Its results are at source .temp/goal-delivery-260908/local-ci-182-bb7/results.json. PR188 will receive local validation at its final signed head after182 lands. This policy change is complete independently of those ongoing implementation deliveries.

Verification:
- agent-session-manager: .github/workflows/ci.yml => disabled_manually (workflow346519002).
- agent-session-manager-spec: .github/workflows/validate.yml => disabled_manually (workflow340251033).
- skill-project-management: .github/workflows/ci.yml => disabled_manually (workflow236218136).
- skill-project-management: .github/workflows/exp-tmpdir-matrix.yml => disabled_manually (workflow352147790).
- skill-project-management: .github/workflows/release.yml => disabled_manually (workflow236218137).
- skill-project-management: dynamic/dependabot/update-graph => active (workflow234126588).
