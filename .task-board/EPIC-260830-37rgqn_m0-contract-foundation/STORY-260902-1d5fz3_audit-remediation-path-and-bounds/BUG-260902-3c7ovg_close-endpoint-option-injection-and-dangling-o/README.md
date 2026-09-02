# BUG-260902-3c7ovg: close-endpoint-option-injection-and-dangling-o

## Description
Two argv-admission gaps the second audit found that the SSH work did not cover, because they are outside the ssh_args field.

ENDPOINT OPTION INJECTION. internal/config/validation.go:448-450 and :465: the endpoint field is never inspected by the bypass gate and may begin with a hyphen. All of these load with err=nil today:
  endpoint = "-oStrictHostKeyChecking=no"
  endpoint = "-F/attacker/config"
  endpoint = "peer.example -oStrictHostKeyChecking=no"
SPEC.md:2494 passes endpoint as an atomic argv value. Unless the launcher inserts a -- separator, this is option injection through a field that passed validation. Probe TestAuditProbeEndpointOptionInjection, all err=nil.

DANGLING -o. A trailing -o with no operand is admitted rather than refused. Cheap to close and it removes an ambiguity from the parser the SSH admission work now depends on.

Also recorded here from the SSH review: the short-flag tables lack the both-directions key-set pin that permittedSSHOptionSamples gives the -o registry, so mutants permitting -E, -L or -A pass the suite green. Those are capability widenings outside SS6.3 and the two letters that reach the normative class, -F and -J, do carry named assertions - but the asymmetry is real and was disclosed rather than fixed.

## Scope
Normative scope: §6.3 host authentication and §6 argv admission, SPEC.md:2494 and :2499-2501.

## Acceptance Criteria
An endpoint beginning with a hyphen, or containing whitespace, is refused - or the field is required to match a [user@]host[:port] grammar. A dangling -o is refused. The short-flag tables gain the same both-directions key-set pin the -o registry has, so a mutant permitting an extra letter reddens. Each case drives the production Load entry and reddens when only its own clause is weakened.
