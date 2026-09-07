# STORY-260905-3t31e9: cross-story-identity-ownership-and-inventory-consolidation

## Description


## Scope
Resolve the structural seams a consolidated cross-Story review (Fable, 2026-09-05) found between internal/terminalbackend, internal/provider and internal/provhost. These are decisions, not defects: each has two defensible answers and the wrong one is expensive to reverse after cmd/ax exists. Report: .temp/fable-consolidated-review.md

## Acceptance Criteria
A1 identity ownership is resolved to a single owner: either canonicaljson implements the three terminal closed shapes and terminalbackend parses through VerifyObjectIdentity, or ownership is formally transferred with the placeholder registrations removed or redirected and the ownership registry updated. Today the two owners give opposite verdicts on every valid terminal manifest. A3 the owner of the Provider Protocol 3.0.0 envelope is named and recorded: the Structured Error 1.3.0 binding, descriptor carriage in launch and resume bodies, and the placement of CheckProviderDescriptor. A6 a shared test-support core for AST inventories exists before an eighth hand-rolled walker appears: production-file glob, fail-closed parse, and a both-direction harness, with terminalbackend ported to executed witnesses rather than textual row resolution. A7 the trust asymmetry between terminalbackend.DigestFile and provider.trustCandidate is either justified by citing the 6.5 versus 7.1 difference or made deliberate.
