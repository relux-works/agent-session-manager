# TASK-260830-2iint0 rework logbook

- Decision: a missing selected `config.toml` is valid only when its resolved
  parent is an existing directory; callers distinguish it from an existing
  empty file through `Snapshot.ConfigPresent`.
- Decision: this path-selection task owns Section 3.2 and AC-PATH-001, not the
  whole of Section 6.1. The sibling versioned-schema loader must own Section
  6.1 because unknown-key and secret-field refusals are outside this slice.
- Evidence correction: one mutation campaign was invalidated after a baseline
  compile error was discovered. The baseline was fixed and rerun green before
  the counted 15-mutant campaign; all 15 narrowed mutants then failed for their
  intended assertions.
- Board anomaly: `task-board validate` exited 0 while reporting 256 legacy
  `MISSING_ACTIVITY` issues outside this task. No direct board-file repair was
  attempted.
