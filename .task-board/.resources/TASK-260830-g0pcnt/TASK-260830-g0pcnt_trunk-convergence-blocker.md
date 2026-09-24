# Unresolved Questions

## TASK-260830-g0pcnt — current trunk registry convergence

The tmux attach implementation is complete at Story checkpoint cf2ad952176d470caad454c0c161ae8253260d54, but final Story registry acceptance is pending because origin/main moved after the workspace's c9233ce2b7d98b70707cb3aec28f919d53a4c020 merge base.

Evidence:

- origin/main is 14d636e40fffbe8a044dcb80bb5ee28c0372f31b, two commits ahead of the Story merge base.
- The trunk delta changes internal/traceability/ownership.v0.7.0.json for landed STORY-260922-cpkajd, adding profile-source acceptance case/test edges and updating the Section 2.4 production/test references.
- Applying only that registry patch causes TestV070RegistryRederivesFromTrunkV060Registry to fail because the pinned Story tree has the older Section 2.4 projection, and causes TestREADMEMeasuredCoverageMatchesTracecheckReport to fail because trunk's renamed test declaration TestAppendGateRefusesLosingLeaseProfileEvent is absent from this tree. The exact patch and failing log are in .temp/TASK-260830-g0pcnt/origin-main-registry.patch and registry-trunk-merge-focused-01.log.
- Reversing that one-file delta restores the checkpoint-consistent registry; registry rederivation, acceptance-case clause edges, README measured-coverage pin, and default tracecheck pass again.
- The managed task-board worktree converge command is documented as orchestrator-only and refuses tracked producer runs. This run is explicitly barred from branch/base operations, and the task scope excludes the other Story's implementation files.

Decision/input needed: the owning orchestrator/operator must converge this managed Story workspace onto fresh trunk, carry and revalidate the uncommitted candidate, merge the trunk registry rows, rederive the canonical digest, and rerun traceability. The current task candidate should then be handed to review. No production-tmux code decision is open.

Recommended route: run the authorized managed worktree convergence from an operator/orchestrator session, then route this task back for registry rederivation and final validation. Do not substitute a GitHub merge action or widen this leaf with unrelated profile-source code.
