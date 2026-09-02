# BUG-260902-beqfwr: close-ssh-host-authentication-admission

## Description
sshHostAuthenticationBypass at internal/config/validation.go:846-871, called from :465, decides admission by matching three option names. SPEC.md:2499-2501 requires refusing StrictHostKeyChecking=no, an empty UserKnownHostsFile, or an equivalent host-authentication bypass. A name blacklist cannot satisfy equivalent by enumeration.

Bypasses confirmed by probe through the production loadConfigDocument, all loading with err=nil:

Grouped short flags. Only tokens equal to -o or prefixed -o are inspected, so -vo StrictHostKeyChecking=no and -4o UserKnownHostsFile=/dev/null are never seen. Verified against OpenSSH 10.2p1 on this host: ssh -G -vo StrictHostKeyChecking=no prints stricthostkeychecking false.

Live alias. StrictHostKeyChecking=false is an OpenSSH alias for no and is absent from the no|off match. Verified: ssh -G -o StrictHostKeyChecking=false prints stricthostkeychecking false.

Options the filter does not know at all: -F pointing at an attacker-writable config, ProxyCommand, KnownHostsCommand, Include, and PermitLocalCommand with LocalCommand. ProxyCommand is arbitrary command execution, strictly worse than a host-key bypass.

TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling at refusal_test.go:15-53 pins fourteen spellings of the two options the filter already knows. Its name claims completeness the filter does not have, and it is the reachability-not-correctness shape this board has rejected repeatedly.

Found by adversarial audit of landed main, each bypass reproduced against a live ssh binary rather than asserted.

## Scope
Normative scope: §6.3. The gate must derive the permitted option set and refuse the complement.

## Acceptance Criteria
Admission derives what is permitted and refuses everything else, so an option the parser has never heard of is refused rather than admitted. Grouped short flags are parsed the way OpenSSH parses them. Every cited bypass has a negative case at the production entry that reddens when only its own clause is weakened. The completeness claim in the existing test name is either earned or the name is corrected.
