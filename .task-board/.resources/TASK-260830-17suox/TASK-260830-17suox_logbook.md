# TASK-260830-17suox logbook

- Rework closed the reviewer-confirmed OpenSSH bypass by parsing separator/quote variants and documented `off`/`none` aliases; separate, combined, tab, quoted, null-device, and empty-known-hosts forms remain pinned at `Load`.
- Terminal capability closure derives from `catalog.Current()` family `terminal_backend`; no hand-copied/self-minted capability registry remains.
- Configuration 1.0.0/2.0.0 reject any explicit legacy backend outside `tmux|conpty` before translation. v1/v2 legacy mapping and v3 defaults are positively exercised on native Windows and WSL2.
- `external_trust[].enabled` controls backend registration. Disabled-only selection refuses; disabled unselected entries remain round-trippable.
- Explicit `transport_policy = []` is retained as the valid deny-all subset; omission alone receives the two-value default.
- `backend_config.settings` remains backend/version-owned and requires an exact caller-supplied validator.
- `DocumentError` preserves `errors.Is`/`errors.As` while rendered errors expose only static clauses.
- Section 17.5 remains unclaimed; durable backup/fsync/atomic replacement and downgrade diagnostics belong to sibling `TASK-260830-1qf777`.
- CR revision 4 fixed a go-toml v2.4.3 ambiguity: a present empty multiline table decoded into a nil typed map. The reader now derives map presence from the generic document shape and initializes only schema map fields that were actually present, preserving absence-vs-read-failure semantics.
- Empty-object compatibility coverage derives all current `map[string]any` members from `rawV3`; the four current paths round-trip through `EncodeCurrent` and `Load`.
- Refusal completeness is now executable during a full package run: Go AST derives every production `configError` call site and TestMain requires each site to be exercised. Focused `-run` executions deliberately do not claim the full inventory gate.
- The extension namespace three-byte minimum is provably subsumed by `reverseDNSPattern`: `a.b` is the shortest accepted member and `ab` refuses. The redundant explicit minimum was removed and the subsuming grammar is named in code and tested.
- Mutation sweep attempts were reported honestly: the first runner had compile-only invalid mutants; the second exposed disappearing artifacts in a shared Go cache. The corrected runner preserves arguments in disabled guards, uses a task-scoped `GOCACHE`, and produced 109/109 behavioral kills with zero invalids or survivors.
- A normal coverage run also hit the shared-cache failure (`fmt` artifact missing) and exited 1; rerunning with task-scoped `GOCACHE` passed. No global cache was cleaned or mutated.
- Ownership evidence was re-pinned only after the registry named the new derived/platform/refusal declarations; the expected pre-pin tracecheck failed exit 1 with digest `dde31415...`, then the reviewed constant was updated and global/scoped checks passed.
- CR revision 5 closes the rev4 "tests pass for the wrong reason" class without
  changing production behavior. Closed-map omission fixtures now remove exactly
  one member from an otherwise accepted current wire document and require the
  raw presence clause, so downstream fail-closed validation cannot satisfy the
  assertion accidentally.
- The Section 6 canonical extension-object bound is now fixture-owned by the
  pinned normative literal `65_536`, not by the implementation constant being
  tested. A widening mutant to `655_360` is therefore killed.
- Six expected-red attacks exited 1: restore-absent-map N-1, four individual
  presence-disjunct mutants P-1 through P-4, and bound-widening E-5. All mutated
  sources were restored and compared byte-identical before final green gates.
