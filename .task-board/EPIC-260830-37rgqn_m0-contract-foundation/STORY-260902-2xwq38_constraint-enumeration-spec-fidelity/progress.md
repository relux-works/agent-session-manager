## Status
to-dev

## Review
required

## Task Class
code

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
ROOT CAUSE, NOT ANOTHER INSTANCE. Six invented constraints were found and fixed during this audit remediation. Every one of them passed the constraint-enumeration gate. This bug is the mechanism that let them through, and closing it is worth more than any single one of those fixes.

THE DEFECT
internal/canonicaljson/constraint_inventory_test.go:52-121 derives inventory rows from requireExactMembers literals and compares them against call sites. That is a self-consistency proof: it shows the artifact agrees with the code. It is dressed as a fidelity proof, which would show the artifact agrees with the SPEC. It never opens SPEC.md.
The specExcerpt column is required only to be NON-EMPTY, at :98-100. Its text is never compared to the pinned document.

WHAT THAT ALLOWED, CONCRETELY
- The quoted word 'digest' reached the 'Pinned SPEC declaration' column for store_schema_fingerprint although no such text exists in the pinned document.
- 'overlapping' was narrowed to the child-partition case while the row still read as enforced.
Both audits reached this finding independently.

WHAT TO BUILD
Compare every specExcerpt against the pinned SPEC.md text. The comparison is already available and simply is not made: the pinned SPEC is fetchable and its SHA-256 is pinned in internal/specpin.
Required properties:
1. A row whose specExcerpt does not occur in the pinned SPEC.md reddens, naming the row and the absent text.
2. The comparison runs against the PINNED document, verified by SHA-256 against internal/specpin before use. Never against a live fetch of the spec repository default branch, and never against a copy that is not hash-checked.
3. Preserve the existing artifact-vs-code derivation. It resists drift between artifact and code, which is real value. This work ADDS the missing axis; it does not replace the existing one.
4. Quoting must be robust to benign formatting differences in a way you state explicitly (whitespace/line-wrap normalization is acceptable if declared). Do not make the match so loose that a substring of an unrelated sentence satisfies it.

ANTI-VACUITY, MANDATORY
The obvious failure mode here is a gate that passes because it compares two things that are both derived from the code. Prove the new comparison actually reads the spec:
- Introduce a row with an excerpt that is absent from the pinned SPEC and show it reddens.
- Perturb the pinned SPEC content and show the hash check refuses rather than silently comparing against the wrong document.
- Show the gate still passes on the unmodified tree.
Report the decisive mutant for each property, not a count.

DISCLOSURE DUTY
If any EXISTING row fails the new comparison, do not fix the row silently and do not weaken the comparison to accommodate it. Report every failing row with its text and the absent-from-spec evidence. A failing row is a seventh invented constraint and is a finding in its own right, which is the entire point of building this gate.

STANDING CRITERIA FOR THIS BOARD
- Drive real production entry points; no test-only helper as the subject.
- Narrow an equivalent mutant rather than deleting it; state equivalence explicitly.
- Never weaken or delete an existing assertion to make new work pass. If an existing test blocks you, say so and stop.
- Quote the pinned spec verbatim for any normative claim, with file and line.
- Disclose anything you found but did not fix.


## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-09-02T19:57:58Z

## Last Update
2026-09-02T22:49:19Z
