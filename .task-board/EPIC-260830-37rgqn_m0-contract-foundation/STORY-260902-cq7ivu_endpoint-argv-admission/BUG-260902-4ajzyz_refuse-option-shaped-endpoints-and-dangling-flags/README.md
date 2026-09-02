# BUG-260902-4ajzyz: refuse-option-shaped-endpoints-and-dangling-flags

## Description
The endpoint field is never inspected by the argv admission gate and may begin with a hyphen, so a validated field carries option injection into the ssh command line. Confirmed by probe through production loadConfigDocument, all loading with err=nil before the fix:
  endpoint = -oStrictHostKeyChecking=no
  endpoint = -F/attacker/config
  endpoint = peer.example -oStrictHostKeyChecking=no
SPEC.md:2494 passes endpoint as an atomic argv value; unless the launcher inserts a -- separator this reaches ssh as options. Gate site internal/config/validation.go:448-450, call site :465.

A trailing value flag with no operand is likewise admitted rather than refused, leaving an ambiguity in the parser the ssh admission work now depends on.

Carried forward from the ssh-argv-admission review and since CONFIRMED REAL: the short-flag tables lacked the both-directions key-set pin the option registry has, so a mutant permitting -A survived the suite green.

Work was produced and reviewed to ACCEPTANCE on a previous element; the accepted patch is attached as a precondition. That element could not be integrated because my own reparenting left a cycle in its move chain, which is not repairable. Nothing is wrong with the work.

What the accepted review established, to preserve through the reapply: 17 mutants with 16 killed and one proven equivalent; an independent 34-shape hostile-endpoint probe all refused, including -ivan@peer.example which the grammar alone would admit without the leading-hyphen clause; 12 dangling value-flag spellings refused; the -A asymmetry reproduced as real and killed by the new pin; internal/config coverage 94.4 to 94.6 percent measured from an archive copy of the base.

## Scope
Normative scope: §6.3 host authentication and §6 argv admission, SPEC.md:2494 and :2499-2501.

## Acceptance Criteria
An endpoint that could be read as an option is refused at the production Load entry, dangling value flags are refused, and the short-flag tables carry the same both-directions key-set pin as the option registry. Each case reddens when only its own clause is weakened.
