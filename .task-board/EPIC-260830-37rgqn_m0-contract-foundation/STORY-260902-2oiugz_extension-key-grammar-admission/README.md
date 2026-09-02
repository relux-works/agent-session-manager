# STORY-260902-2oiugz: extension-key-grammar-admission

## Description
Remove an invented forbidden-word list that refuses legitimate reverse-DNS extension keys, found by adversarial audit of landed configuration code.

## Scope
Normative scope: §6 extension keys. Where the pinned spec is silent, be permissive.

## Acceptance Criteria
Extension keys are admitted by the reverse-DNS grammar the pinned spec declares and nothing more. Legitimate keys carrying env, auth, environment or endpoint labels load through the production entry, and the tests that pinned the invented rule are removed with it.
