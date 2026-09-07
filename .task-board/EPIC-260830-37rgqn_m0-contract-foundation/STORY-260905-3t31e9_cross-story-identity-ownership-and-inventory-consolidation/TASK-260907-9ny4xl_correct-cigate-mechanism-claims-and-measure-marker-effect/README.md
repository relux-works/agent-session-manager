# TASK-260907-9ny4xl: correct-cigate-mechanism-claims-and-measure-marker-effect

## Description
Close R1-R6 from TASK-260830-2uowwk review-verdict-rev2 and the residue named in the secprim and secconftest verdicts. R1: ci.yml:298 and the LOGBOOK 2249 entry state that ProbeState.Available is dead, but claims.go:235 reads it; what remains discarded is the live probe outcome, which doc.go states correctly and those two do not. R2: ci.yml:240-241 names the wrong guard as fail-closed. R3: the marker census measures occurrence rather than effect - markers classify 0 of 2. R4: a narrowing mutant of the sentence splitter survives the whole suite. R5/R6: two stated bounds to record, including that Linux execution was unverified at review time (now answered by the PR run, so restate it as answered rather than open). Also carry the pre-existing defect the verdict deliberately did not charge to that Change Request: the catalog-freshness step is defeated by rewriting //go:generate as // go:generate, which preserves the token; that exists at HEAD and predates the leaf.

## Scope
relux-works/agent-session-manager-spec@v0.5.0 (commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c). Residue from the accepted STORY-260830-1i3qu7 review verdicts; no gate admits what its contract forbids, these are claim accuracy and one unmeasured region.

## Acceptance Criteria
Production behavior demonstrates: every mechanism claim in ci.yml, doc.go, README.md and the LOGBOOK describes what the code does, proven by a test that fails when the claim and the behaviour diverge. The marker census measures effect rather than occurrence and fails closed on a marker that classifies nothing. The sentence-splitter narrowing mutant dies. The catalog-freshness step rejects a token-preserving //go:generate rewrite. Each stated bound names what escapes it and would survive being attacked directly.
