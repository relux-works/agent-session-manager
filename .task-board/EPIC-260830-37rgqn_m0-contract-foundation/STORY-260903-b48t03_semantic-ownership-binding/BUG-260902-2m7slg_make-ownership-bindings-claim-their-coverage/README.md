# BUG-260902-2m7slg: make-ownership-bindings-claim-their-coverage

## Description
The ownership gate proves that a Go symbol exists and an acceptance case is linked. It never proves the section MEANING is implemented, so a sliver implementation satisfies a whole-section binding and the gate reports it as owned.

internal/traceability/traceability.go:678-712 sourceChecker.verify parses the file and checks hasDeclaration only; verifyOwnershipGroups at :332-420 adds acceptance-case linkage. Neither compares implementation against the section it claims.

Three slivers found, in three independently produced branches:
  section:2.2 bound to validateSessionRecordCommon, which validates one enum and a name grammar. SPEC.md:388-430 SS2.2 is the lease and replica global invariants, and the linked acceptance cases are canonical-identity cases.
  section:18.4 bound to OpenProjection. SPEC.md:11767-11790 SS18.4 is audit retention; no retention code exists in localstore.
  section:6.5 bound to translateV3 while required_capabilities defaults to empty at internal/config/validation.go:664-668, not the platform lane minimum of SPEC.md:2585.

Consequence: any future Story assigned SS2.2 or SS18.4 finds the section already owned and can legitimately do nothing. The gate presents symbol reachability as semantic coverage, and three branches reached that state without anyone noticing.

This is the structural finding both audits independently reached. It is expensive and it is the one that decides whether the traceability story means anything.

## Scope
Normative scope: §17.2 traceability and the ownership registry.

## Acceptance Criteria
A binding declares its coverage - full, partial or sliver - and assigned-scope admission gates on full, or bindings resolve to acceptance cases whose names enumerate the section MUST clauses they discharge. Either way, the three known slivers are re-declared honestly rather than quietly upgraded. A binding that claims full coverage while implementing one clause reddens the gate, proven by planting one. Sections whose semantics are unimplemented are recorded as unowned instead of bound to an unrelated symbol.
