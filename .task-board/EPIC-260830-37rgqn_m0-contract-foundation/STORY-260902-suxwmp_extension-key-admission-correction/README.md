# STORY-260902-suxwmp: extension-key-admission-correction

## Description
Remove an invented forbidden-word list that refuses legitimate reverse-DNS extension keys. Split out of STORY-260902-1hrtzp because that Story workspace recorded an empty checkpoint OID and can no longer admit a second leaf.

## Scope
Normative scope: §6 extension keys. Where the pinned spec is silent, be permissive.

## Acceptance Criteria
Extension keys are admitted by the reverse-DNS grammar the pinned spec declares, nothing more. Legitimate keys carrying env, auth, environment or endpoint labels load through the production entry, and the tests that pinned the invented rule are removed with it.
