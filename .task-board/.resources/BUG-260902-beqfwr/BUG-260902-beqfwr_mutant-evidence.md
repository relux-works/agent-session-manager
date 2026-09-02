# BUG-260902-beqfwr — single-clause mutant evidence

Each mutant weakens exactly one clause of `internal/config/sshargs.go` (nothing
is deleted outright), then `go test ./internal/config -count=1 -v` runs and the
failing test names are recorded. The harness restores the file afterwards; the
script is `.temp/BUG-260902-beqfwr/mutants.py`.

Control run (unmutated): exit 0, no failing test.

| Mutant | Clause weakened | Exit | Cases that redden |
| --- | --- | ---: | --- |
| M1 | add `'F'` to `sshPermittedValueFlags` | 1 | `…OutsideThePermittedTables` (explicit `-F` and `-F/path` assertions), `…GroupedShortFlags/an_unpermitted_letter_inside_a_group_is_refused` |
| M2 | add `proxycommand` to the registry permitting any value | 1 | `…UndeclaredSSHOptionNames/{ProxyCommand, ProxyCommand_combined, option_name_only}`, `…GroupedShortFlags/grouped_valueless_flag_still_lets_-o_take_the_next_argument`, `…AdmitsExactlyTheDeclaredPermittedSSHOptions`, `…RefusesThroughTheMeshPeerClause` |
| M3 | widen `StrictHostKeyChecking` from `yes` to `yes\|no` | 1 | `…OptionSpellingGrammar/{separate_equals, separate_whitespace, separator_run, combined_whitespace, quoted_value, strict_no_case_folded, strict_grouped_short_flag, strict_grouped_attached_value}`, `…GroupedShortFlags/{next_argument, attached_value}`, `…RefusesSecurityBypasses…/host-key_bypass` |
| M4 | stop walking a short-flag group after its first letter | 1 | every grouped case: `…OptionSpellingGrammar/{strict_grouped_short_flag, known_hosts_grouped_short_flag, strict_grouped_attached_value}`, all four `…GroupedShortFlags` subtests, and the 37 derived `…OutsideThePermittedTables/<letter>` grouped assertions |
| M5 | admit `-o` names the registry does not declare | 1 | all eleven `…UndeclaredSSHOptionNames` subtests, `…GroupedShortFlags/next_argument`, `…RefusesThroughTheMeshPeerClause` |
| M6 | permit any `UserKnownHostsFile` value | 1 | `…RegistryDeclared…/userknownhostsfile`, `…OptionSpellingGrammar/{known_hosts_null_device, known_hosts_none, known_hosts_tab_separator, combined_known_hosts, known_hosts_grouped_short_flag}`, `…GroupedShortFlags/{next_argument, attached_value}` |
| M7 | skip arguments that are not options | 1 | `…RefusesSSHArgumentsThatAreNotOptions` |
| M8 | permit `Include` | 1 | `…UndeclaredSSHOptionNames/Include`, `…AdmitsExactlyTheDeclaredPermittedSSHOptions` |
| M9 | permit `PermitLocalCommand` and `LocalCommand` | 1 | `…UndeclaredSSHOptionNames/{PermitLocalCommand_and_LocalCommand, LocalCommand_alone}`, `…AdmitsExactlyTheDeclaredPermittedSSHOptions` |
| M10 | permit `-p` values outside the port range | 1 | `…GroupedShortFlags/a_value-taking_letter_ends_its_group` |

Notes:

- M1 does not show a `…OutsideThePermittedTables/F` subtest, because permitting
  `F` removes that derived subtest from the loop. That is exactly why the `-F`
  bypass is also asserted explicitly by name in the parent test body, and those
  explicit assertions are what redden. A derived loop alone would have gone
  quiet under this mutant.
- M8 and M9 also redden `…AdmitsExactlyTheDeclaredPermittedSSHOptions` because
  widening the permitted registry without declaring an admitted/refused sample
  for the new name breaks the key-set equality that closes that test.
- M3 is a narrowing weakening (one extra permitted value), not a deletion, and
  it reddens only the `no`/`NO` spellings; `off`, `false`, and `accept-new`
  stay refused because they are outside the permitted set rather than on a
  blacklist.
- M4's blast radius is the point: group walking is a single clause that every
  grouped case depends on.

Raw run output: `.temp/BUG-260902-beqfwr/mutant-evidence.txt`.
