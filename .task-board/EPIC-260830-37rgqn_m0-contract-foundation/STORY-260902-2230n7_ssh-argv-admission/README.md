# STORY-260902-2230n7: ssh-argv-admission

## Description
Replace the SSH host-authentication name blacklist with a derived admission decision. Re-provisioned from STORY-260902-1hrtzp, whose workspace recorded an empty checkpoint OID and could no longer produce a story-final Change Request. The work is already produced and reviewed; it is attached as a precondition patch.

## Scope
Normative scope: §6.3 host authentication and the §6 argv admission surface.

## Acceptance Criteria
Admission derives the permitted option set and refuses the complement, so an option the parser has never heard of is refused rather than admitted. Every cited bypass is refused at the production entry with a case that reddens when only its own clause is weakened.
