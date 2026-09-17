# TASK-260909-2ez769 — typed write-path census rev10

## Gate and production boundary

The authoritative gate is `internal/config/writepath_census_types_test.go`, exercised by `TestWritePathCensusGate`. It loads the active Go build context with `golang.org/x/tools/go/packages`, resolves package/method/alias/function-value identity with `go/types`, and audits Config plus Host Trust Store mutation paths. Unknown or unresolved calls, non-`*types.Func` targets, and unlisted object identities fail closed. The gate does not infer safety from spelling alone.

The accepted indirect edges are explicit object-identity policies for the production pair writers, their typed seams, the exact hold-entry callbacks, selected package initializers, and named function fields. The gate compares exact package path, receiver, method, parameter/local, or interface-method identity. Same-named functions, impostor receivers, unlisted method values, and unlisted callbacks do not inherit an exemption.

The filesystem rule keeps direct standard-library mutations inside the exact `osFileSystem` and `osMigrationFileSystem` backend method sets. `JointCommit`, `WithExclusiveHold`, and the pair writers are audited as hold-bound entry paths; runtime `HeldExclusive` validation independently rejects zero, released, foreign-resource, foreign-config, and foreign-state-root capabilities before mutation.

Scope bound: this census evaluates the active Go build context. Build-tagged or platform-specific sources require a separate invocation under that context. Windows custody has separate native tests and cross-compilation evidence, but no Windows runtime census execution is claimed here.

## Controls added or retained

The static gate and executable rogue suite cover both direct mutation and value/indirect escape shapes:

1. dot-imported `os.WriteFile` function value;
2. promoted method of an embedded backend;
3. package-level slice of writer functions;
4. function-typed struct field;
5. function-typed parameter;
6. dot-imported writer stored in a composite literal;
7. type-parameter method value;
8. channel callback;
9. immediate function literal;
10. local helper aliases, import aliases, map values, closures/IIFEs, method values, interface implementations, generic helpers, token-gated shapes, and unresolved callees.

Each executable control has a behavioral witness that performs the relevant temporary-file mutation. `TestWritePathCensusFailsClosedOnUnresolvedCallee` proves an unresolved writer-shaped call is not silently accepted. `TestWritePathGateFlagsExecutableRogues` proves the static findings correspond to executable mutation behavior, not just source tokens.

The token-preserving `writepath-method-value-skip` mutant changes behavior while preserving the searched-for source token and runs the behavioral suite; it is killed by the production rogue tests.

## Exact mutation results

### Config4 census and migration battery

Final exact-source run: `python3 internal/config/mutations_v4.py --output .temp/TASK-260909-2ez769/mutations-config-final` with an absolute task-scoped `GOCACHE`.

- neutral control: passed, exit 0;
- 11 narrowing/precision mutants: killed, exit 1;
- survivors: 0.

Killed mutants: `transport-ssh`, `preview-length`, `confirm-drops`, `apply-gen-direction`, `v4-explicit-label`, `compensate-source-swap`, `compensate-rollback-source-swap`, `replace-require-skip`, `legacy-revalidate-skip`, `replace-restore-skip`, and `writepath-method-value-skip`.

Evidence: `.temp/TASK-260909-2ez769/mutations-config-final/results.json` and `table.md`.

### Host Trust Store battery

Final exact-source runs were serialized with task-scoped absolute caches and restored the candidate source after every plant.

- neutral control: passed, exit 0;
- 34 narrowing/precision mutants: killed, exit 1;
- survivors: 0.

The unique killed set covers: `authorize-converge-ignore`, `compensate-marker-skip`, `compensate-restore-ignore`, `compensate-restore-verify-skip`, `compensate-verify-before-restore`, `custody-0644`, `custody-binding`, `dance-converge-ignore`, `digest-root-swap`, `dup-credid`, `exhaustion`, `expiry-grace`, `gen-ceiling`, `hold-no-lock`, `hold-require-skip`, `hold-state-root-skip`, `initialize-converge-ignore`, `joint-converge-skip`, `joint-marker-stale`, `lock-replacing-init`, `marker-unreadable`, `per-host-bound`, `recover-diverged`, `recover-forward-no-bump`, `resolve-source-skip`, `retire-bound`, `retire-expiry`, `revoke-state`, `rotation-expiry`, `stale-direction`, `stale-mutation-refresh`, `transact-converge-ignore`, `unique-root-pair`, and `unknown-field`.

Evidence: `.temp/TASK-260909-2ez769/mutations-hosttrust-rev10-01/`, `-02/`, `-03/`, `-04/`, `-05/`, `-06/`, `-comp-restore/`, `-order/`, and `-root/` (`results.json` in each directory). Duplicate root reruns were deduplicated by mutant name; no row was counted twice.

## Runtime gates paired with the census

- `hosttrust.HeldExclusive` is forged only by the exclusive-hold implementation, checked live before each pair write, and invalidated on release.
- `EnsureConfigBinding` is the explicit enrollment/binding step; pair writers and `JointCommit` cannot mint a binding from an unbound store.
- Config pair writes revalidate the selected source and state root under the same exclusive hold before staging, replacement, compensation, and generation commit.
- Compensation re-reads and verifies the marker and restored bytes under the same hold; if safe restore cannot be proven, the marker is kept for deterministic recovery rather than silently overwriting a converged source.
- Credential staging writes inert files before the admitting trust commit; failed issuance/rotation cleanup removes the uncommitted credential directory and its staged files.

## Evidence locations

- Static/census control logs: `.temp/TASK-260909-2ez769/census-strict-04.log`, `census-controls-02.log`.
- Full package results: `.temp/TASK-260909-2ez769/config-full-06.log`, `hosttrust-full-01.log`, `secprim-full-02.log`.
- Complete test and coverage runs: `go-test-all-02.log`, `go-cover-all-02.log`.
- Preservation audit: `preservation-final-01.log`.

The census is a negative gate: all reported survivors are zero, and the neutral controls establish that the harness itself still admits the unchanged candidate. Delete-only plants are not used as evidence.
