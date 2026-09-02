# STORY-260902-1d5fz3: audit-remediation-path-and-bounds

## Description
Remaining confirmed findings from the two adversarial audits of landed AX code, other than the three already landed (recursion DoS, SSH admission, invented extension-key blacklist). Each was reproduced or established by code reading against the pinned SPEC v0.5.0 at 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c.

## Scope
Normative scope: §1.6, §3.2, §6, §10.1, §13.14, §17.2, Appendix D.

## Acceptance Criteria
Each finding is either fixed with a negative case that reddens when only its own clause is weakened, or closed with a written decision citing the pinned line that makes it a non-defect. No finding is closed by assertion alone.
