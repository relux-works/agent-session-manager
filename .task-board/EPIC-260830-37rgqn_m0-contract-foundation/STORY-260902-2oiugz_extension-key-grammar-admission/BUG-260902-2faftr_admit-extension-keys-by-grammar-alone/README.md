# BUG-260902-2faftr: admit-extension-keys-by-grammar-alone

## Description
hasForbiddenConfigName in internal/config/validation.go refuses any name containing secret, token, password, credential, auth, environment, env or endpoint, applied to reverse-DNS extension keys and to nested map keys at any depth.

The pinned spec reserves no extension-key label and imposes no naming rule inside an extension value object. It goes further: SPEC.md:2344-2345 explicitly permits a configuration field to NAME a credential profile, which is exactly what the rule refused. So this is not merely undeclared; it contradicts the contract.

Confirmed by probe: works.relux.env-tools, works.relux.environment, com.example.auth-manager and io.example.endpoint-list are all refused, while the rule inspects zero values, so a key holding an actual credential passes. It was pinned by tests, meaning an earlier review round hardened the invented rule instead of catching it.

Work for this was already produced and reviewed to acceptance on a previous element; the accepted patch is attached as a precondition. That element was discarded because its parent Story had been deleted while a move chain still referenced it, which no correction surface can repair.

## Scope
Normative scope: §6 extension keys.

## Acceptance Criteria
The invented rule and its call sites are removed, both admission points decide by grammar and the depth bound alone, and the tests that pinned the rule are removed rather than left asserting it. Legitimate reverse-DNS keys carrying every previously blacklisted label are admitted and round-trip through the production Load entry. Re-adding the blacklist as a narrowing mutant reddens the suite.
