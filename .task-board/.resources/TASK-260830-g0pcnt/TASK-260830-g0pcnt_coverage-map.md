# TASK-260830-g0pcnt Supplemental Coverage Map

Pinned authority: internal/specdoc/SPEC.v0.7.0.md §3.2, lines 806–813.
The producer brief did not supply a catalog-derived surface table; this is a
brief gap. The rows below are a supplemental position × production-entry map,
not a substitute claim that the missing table was supplied.

## Path-position mode cells

Each row maps the exhaustive mode oracle's named position/entry subtest to a
narrowing mutant that was applied alone and killed by the oracle.

| Component position | Production entry | Named oracle subtest | Narrowing killed |
|---|---|---|---|
| depth 00 / socket leaf | Probe | TestCustodyModeOracleAtProductionEntries/socket_leaf/Probe/depth_00_socket_leaf | N-socket-mask-drop-other-triplet |
| depth 00 / socket leaf | Spawn | TestCustodyModeOracleAtProductionEntries/socket_leaf/Spawn/depth_00_socket_leaf | N-socket-mask-drop-other-triplet |
| depth 00 / socket leaf | Execute | TestCustodyModeOracleAtProductionEntries/socket_leaf/Execute/depth_00_socket_leaf | N-socket-mask-drop-other-triplet |
| depth 01 / runtime tmux directory | Probe | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Probe/depth_01_runtime_tmux_dir | N-runtime-dir-combination-admit |
| depth 01 / runtime tmux directory | Spawn | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Spawn/depth_01_runtime_tmux_dir | N-runtime-dir-combination-admit |
| depth 01 / runtime tmux directory | Execute | TestCustodyModeOracleAtProductionEntries/runtime_tmux_dir/Execute/depth_01_runtime_tmux_dir | N-runtime-dir-combination-admit |
| depth 02 / runtime root | Probe | TestCustodyModeOracleAtProductionEntries/root/Probe/depth_02_runtime_root | N-root-admit-0750 |
| depth 02 / runtime root | Spawn | TestCustodyModeOracleAtProductionEntries/root/Spawn/depth_02_runtime_root | N-root-admit-0750 |
| depth 02 / runtime root | Execute | TestCustodyModeOracleAtProductionEntries/root/Execute/depth_02_runtime_root | N-root-admit-0750 |
| depth 03 / ancestor | Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_03_ancestor | N-ancestor-walk-skips-first |
| depth 03 / ancestor | Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_03_ancestor | N-ancestor-walk-skips-first |
| depth 03 / ancestor | Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_03_ancestor | N-ancestor-walk-skips-first |
| depth 04 / ancestor | Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_04_ancestor | N-ancestor-walk-stops-after-first |
| depth 04 / ancestor | Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_04_ancestor | N-ancestor-walk-stops-after-first |
| depth 04 / ancestor | Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_04_ancestor | N-ancestor-walk-stops-after-first |
| depth 05 / ancestor | Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_05_ancestor | N-ancestor-walk-skips-last-below-root |
| depth 05 / ancestor | Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_05_ancestor | N-ancestor-walk-skips-last-below-root |
| depth 05 / ancestor | Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_05_ancestor | N-ancestor-walk-skips-last-below-root |
| depth 06 / filesystem root ancestor | Probe | TestCustodyModeOracleAtProductionEntries/ancestor/Probe/depth_06_ancestor | N-ancestor-walk-stops-after-first |
| depth 06 / filesystem root ancestor | Spawn | TestCustodyModeOracleAtProductionEntries/ancestor/Spawn/depth_06_ancestor | N-ancestor-walk-stops-after-first |
| depth 06 / filesystem root ancestor | Execute | TestCustodyModeOracleAtProductionEntries/ancestor/Execute/depth_06_ancestor | N-ancestor-walk-stops-after-first |

Measured: 21 of 21 observed path-position × entry cells. The oracle compares
86,016 of 86,016 mode-position-entry classifications. It also enumerates
105 of 105 kind-position-entry cases and 27 of 27 owner-position-entry cases
at owner-gated positions. Ancestor ownership is the N2 bound and is not counted
as measured.

## Metadata axes

| Axis | Test | Narrowing evidence / bound |
|---|---|---|
| Kind at every position and entry | TestCustodyPathKindOwnerOracleAtProductionEntries | Socket leaf is attacked by N-bind-socket-kind and N-socket-symlink. Runtime kind is pinned by directory-open and verification refusal tests. |
| Owner at socket, runtime directory, and runtime root through every entry | TestCustodyPathKindOwnerOracleAtProductionEntries | N-socket-ownership narrows the socket owner gate; N-ownership-uid narrows the shared owner predicate at all three owner-gated positions and entries. |
| Ancestor owner | Not measured | N2 bound: a foreign non-root owner could rename a protected ancestor; §3.2 lines 810–812 does not define accepted system-ancestor ownership. Owner: future tmux path-custody owner at checkCustodyAncestors. |
