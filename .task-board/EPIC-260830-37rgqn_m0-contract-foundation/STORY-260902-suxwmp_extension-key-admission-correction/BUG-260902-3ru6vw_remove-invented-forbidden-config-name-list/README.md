# BUG-260902-3ru6vw: remove-invented-forbidden-config-name-list

## Description
hasForbiddenConfigName at internal/config/validation.go:893-900, applied at :745 to reverse-DNS extension keys and at :790 to nested map keys at any depth, refuses any name containing secret, token, password, credential, auth, environment, env or endpoint.

The pinned spec defines extension keys as reverse-DNS grammar only (SPEC.md:344-352). The secret rule at SPEC.md:2596-2597 is about VALUES, not key names. The nested-key arm is wholly invented; the spec imposes no naming inside extension values.

Consequences confirmed by probe: works.relux.env-tools, com.example.auth-manager, works.relux.environment and io.example.endpoint-list are all refused as forbidden, so legitimate reverse-DNS keys owned by this organisation cannot be used. As secret detection it inspects zero values, so com.example.deploy = AKIA... passes.

The rule is pinned by refusal_test.go:390-395, which means the review loop hardened an invented constraint rather than removing it. This is the invented-constraint class the board has rejected three times on other leaves.

## Scope
Normative scope: §6 extension keys. Where the pinned spec is silent, be permissive.

## Acceptance Criteria
The invented name blacklist is removed, or narrowed to exactly what the pinned spec declares with the declaring line quoted verbatim. Legitimate reverse-DNS keys containing env, auth or endpoint labels load. The tests that pinned the invented rule are removed with it. If any value-level secret rule is kept, it cites SPEC.md:2596-2597 and inspects values rather than key names.
