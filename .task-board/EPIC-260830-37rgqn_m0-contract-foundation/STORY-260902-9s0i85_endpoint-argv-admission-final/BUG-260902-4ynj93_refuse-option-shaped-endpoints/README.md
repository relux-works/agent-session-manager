# BUG-260902-4ynj93: refuse-option-shaped-endpoints

## Description
The endpoint field was never inspected by the argv admission gate and could begin with a hyphen, carrying option injection into the ssh command line through a field that had passed validation. Reproduced before the fix at the production entry: -oStrictHostKeyChecking=no, -F/attacker/config and peer.example -oStrictHostKeyChecking=no all load with err=nil. SPEC.md:2494 passes endpoint as an atomic argv value. Gate site internal/config/validation.go:448-450, call site :465.

Dangling value flags with no operand were likewise admitted, leaving an ambiguity in the parser the ssh admission work depends on. And the short-flag tables lacked the both-directions key-set pin the option registry has, so a mutant permitting an extra letter survived green - carried forward from the ssh-argv-admission review and since confirmed real.

WORK IS ACCEPTED AND ATTACHED. Three revisions, the third accepted after two findings the reviews caught and I would not want lost:

The leading-hyphen clause was at first pinned by cases that CONTAINED NO @, so the downstream host grammar refused them anyway and the clause the test named was never deciding. Narrowing it kept the suite green while -ivan@peer.example loaded - and ssh(1) reads that as -i with identity file van@peer.example. Fixed by cases only that clause can refuse.

The port length bound was pinned by a SEVEN-character port, which fixes the bound to a range rather than to its limit; the narrowed form then survives. Fixed by the adjacent value, peer.example:000022.

Final review: 29 mutants, 27 killed and 2 proven equivalent; a 76-shape hostile probe including Unicode hyphen lookalikes, zero-width and BOM prefixes and non-ASCII digits, admitting only ordinary hosts; and before/after reachability - the same six hostile endpoints load 18 of 18 on the pre-fix base across all three readers and are refused 0 of 18 at the candidate.

## Scope
Normative scope: §6.3 and §6 argv admission.

## Acceptance Criteria
Reapply the attached accepted work onto current trunk, resolving only the LOGBOOK overlap. Everything else is byte-identical to the accepted tree, and any deviation outside LOGBOOK.md is reported rather than absorbed.
