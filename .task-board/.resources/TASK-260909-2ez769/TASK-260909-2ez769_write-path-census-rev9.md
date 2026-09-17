# TASK-260909-2ez769 rev9 typed write-path census evidence

## Gate and production boundary

The gate is `internal/config`'s `writepath_census_types_test.go`. It loads the current Go build context with `golang.org/x/tools/go/packages`, resolves `go/types` objects by package path, receiver, and method name, and audits Config/Host Trust Store write paths. The production entry tests are `TestWritePathCensusGate`, `TestWritePathCensusFailsClosedOnUnresolvedCallee`, `TestWritePathGateFlagsUnlockedCaller`, and `TestWritePathGateFlagsExecutableRogues`.

The hold-entry exception is exact: `isHoldEntryCall` accepts only the censused `hosttrust.Store` receiver identities (plus the test control receiver). A same-named method on an impostor receiver is rejected. This keeps the source gate tied to the real production authorization API.

## Rev9 controls

The census resolves direct calls and function values instead of trusting spelling alone. It rejects writer-shaped indirect paths through:

1. a dot-imported `os.WriteFile` function value;
2. a promoted method of an embedded backend;
3. a package-level slice of writer functions;
4. a function-typed struct field;
5. a function-typed parameter; and
6. a dot-imported writer stored in a composite literal.

Each shape has a source witness and an executable behavioral witness. Existing controls cover direct OS mutation, helper aliases, import aliases, map values, closures/IIFEs, method values, interface implementations, generic helpers, token-gated shapes, and unresolved callees. The behavioral witnesses actually mutate temporary files, so a static-only pass is not sufficient.

The gate fails closed for unresolved writer-shaped callees and for indirect writer-shaped calls not bound to a censused definition. The documented scope is the active Go build context; other platform/build-tag variants require their own census invocation.

## Mutation evidence

- `python3 internal/config/mutations_v4.py --output .temp/TASK-260909-2ez769/mutations-config-rev9 --only writepath-method-value-skip`
  - result: `writepath-method-value-skip: killed, exit=1`
  - failing production test: `TestWritePathGateFlagsExecutableRogues/backend_method_value` (and the enclosing suite)
  - the mutation preserves the searched token and changes behavior, satisfying the token-preserving source-gate attack requirement.

- `python3 internal/hosttrust/mutations.py --output .temp/TASK-260909-2ez769/mutations-hosttrust-rev9 --only hold-resource-skip`
  - result: `hold-resource-skip: killed, exit=1`
  - failing production test: `TestWithExclusiveHoldForConfigRefusesUnconfiguredStorePath`
  - the mutation retains the configured-path mismatch checks but admits a generic Store, so this is a narrowing resource-binding attack rather than delete-only evidence.

## Related runtime negative evidence

`TestWriteTempReplaceRefusesForeignStoreHold` and `TestWriteTempReplaceRefusesConfiguredHoldForDifferentConfigPath` prove that the Config writer refuses a foreign/unpaired hold before staging and leaves target bytes/staging unchanged. `TestWithExclusiveHoldForConfigRefusesUnconfiguredStorePath` and `TestWithExclusiveHoldForConfigRefusesDifferentPathOnConfiguredStore` exercise the resource binding at the hosttrust production API.

The final targeted run is recorded in `.temp/TASK-260909-2ez769/go-test-targeted-rev9.log`; mutation logs and raw harness output are in the two `mutations-*-rev9` directories.
