# BUG-260902-beqfwr — close ssh host authentication admission

Commit `7eac813` on `task-board/story/STORY-260902-1hrtzp` (signed, verified).

## What was wrong

`sshHostAuthenticationBypass` (validation.go:846, called from :465) decided
admission by matching three option names. SPEC.md §6.3 requires refusing
`StrictHostKeyChecking=no`, an empty `UserKnownHostsFile`, "or an equivalent
host-authentication bypass" — an equivalence class with no enumeration.

## What changed

`internal/config/sshargs.go` (new) declares the permitted set and refuses its
complement:

| Declared source | Content |
| --- | --- |
| `sshShortOptionsWithoutValue` / `sshShortOptionsWithValue` | `ssh(1)` short-option arity transcribed from the OpenSSH 10.2p1 usage text |
| `sshPermittedFlags` | valueless short options AX admits: `4 6 C T a q v` |
| `sshPermittedValueFlags` | value-taking short options AX admits: `i l p o`, each with its value rule |
| `sshOptionRegistry` | `-o` option names AX admits, each with the values its rule permits; plus the host-authentication names declared with no permitted value so their refusal reports the §6.3 clause |

`admitSSHArguments` walks argv with getopt semantics — grouped short flags, a
value-taking letter ending its group and taking the rest of the argument or the
whole next argument — and returns a closed-vocabulary refusal reason for the
first argument outside the declaration. `validateMesh` appends that reason to
the existing `mesh.peers[i].ssh_args` clause. There is still exactly one
refusal call site, so the package's refusal-inventory audit is unchanged.

An option name the parser has never heard of is refused, not admitted:
`ProxyCommand`, `Include`, `LocalCommand`, `ProxyUseFdpass`, `Match`, and
`ThisOptionDoesNotExistYet` are all refused because the registry does not
declare them, not because they are listed anywhere.

`StrictHostKeyChecking` is declared with only its enforcing spelling `yes`, so
the live OpenSSH aliases `no`, `off`, `false`, `NO`, `FALSE`, and `accept-new`
fall outside the permitted set without any alias being listed.

## Live OpenSSH evidence (OpenSSH_10.2p1, LibreSSL 3.3.6)

| Probe | OpenSSH result |
| --- | --- |
| `ssh -G -F /dev/null -vo StrictHostKeyChecking=no probe.example` | `stricthostkeychecking false` |
| `ssh -G -F /dev/null -4o UserKnownHostsFile=/dev/null probe.example` | `addressfamily inet`, `userknownhostsfile /dev/null` |
| `ssh -G -F /dev/null -o StrictHostKeyChecking=false probe.example` | `stricthostkeychecking false` |
| `ssh -G -F /dev/null -o StrictHostKeyChecking=NO probe.example` | `stricthostkeychecking false` |
| `ssh -G -F /dev/null -qp2222 probe.example` | `port 2222` (group ends at `p`, digits are its value) |
| `ssh -G -F /dev/null -i -oProxyCommand=id probe.example` | no `proxycommand` line — `-i` consumed the next argv as a filename |
| `ssh -G -F /dev/null -o ProxyCommand=id probe.example` | `proxycommand id` (control for the line above) |
| `ssh -G -F /dev/null -o "$(printf 'BatchMode=yes\nProxyCommand=evil')" …` | `command-line line 0: unsupported option` — a newline does not smuggle a second option |

Every one of these argv shapes is now refused by `loadConfigDocument` except
the two that are legitimately harmless (`-qp2222`, `-i -oProxyCommand=id`),
which are admitted for the same reason OpenSSH reads them that way.

## Tests

New `internal/config/ssh_admission_test.go`, all driving the production
`loadConfigDocument` entry through a mesh peer:

- `TestSSHShortOptionTablesAreDerivedFromTheOpenSSHGrammar` — pins the arity of
  every admitted letter to the transcribed grammar; a letter admitted with the
  wrong arity would make the group walk diverge from OpenSSH.
- `TestLoadRefusesEveryHostAuthenticationOptionDeclaredInTheRegistry` — derives
  its cases from `sshOptionRegistry`; a newly declared host-authentication
  option is covered the moment it is added.
- `TestLoadAdmitsExactlyTheDeclaredPermittedSSHOptions` — closes admission in
  both directions; the sample key set must equal the permitted registry.
- `TestLoadRefusesEverySSHShortOptionOutsideThePermittedTables` — walks the
  whole declared grammar; `-F` is also asserted explicitly in both spellings.
- `TestLoadParsesGroupedShortFlagsTheWayOpenSSHParsesThem` — the `-vo`, `-4o`,
  `-voStrict…`, `-qp2222`, and `-i <next argv>` shapes above.
- `TestLoadRefusesUndeclaredSSHOptionNamesAtTheProductionEntry` —
  ProxyCommand, ProxyUseFdpass, Include, PermitLocalCommand+LocalCommand,
  KnownHostsCommand, Match exec, and an unknown name.
- `TestLoadRefusesSSHArgumentsThatAreNotOptions` — a bare destination or remote
  command, `-`, `--`, a GNU-style long option, and a flag without its value.
- `TestLoadAdmitsTheSpecifiedPeerArgumentExample` — the §6.3 example argv and a
  realistic peer argv still load, so admission is not vacuously strict.

`TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling` claimed a
completeness the filter did not have. It is renamed to
`TestLoadRefusesHostAuthenticationBypassAcrossTheOptionSpellingGrammar`, which
is what it actually covers (separator, quoting, and grouping spellings), and its
doc comment names where name-completeness is now earned: the registry-derived
test plus the closed admission. Six new cases were added to it: the `false` and
`FALSE` aliases, `NO`, `accept-new`, and the two grouped short-flag forms.

Two pre-existing bound tests in `schema_test.go` used `"xxxx…"` filler as
`ssh_args`, which is a bare operand and is now refused. They were rewritten to
use `-i<path>` argv of the same byte lengths, so they still prove the 4,096-byte
per-argument, 65,536-byte total, and 64-argument bounds at and past the limit.

## Gates

| Gate | Command | Exit |
| --- | --- | ---: |
| Build | `go build ./...` | 0 |
| Vet | `go vet ./...` | 0 |
| Format | `gofmt -l .` (no output) | 0 |
| Tests | `go test ./... -count=1` | 0 |
| Coverage | `go test ./... -cover -count=1` | 0 |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| Traceability (assigned scope) | `tracecheck -section 3.2 -section 6.1 … -section 17.4` | 0 |
| Catalog | `cataloggen … -check` | 0 |

`internal/config` coverage 93.7% → **93.9%**; no package regressed.

`golangci-lint run ./internal/config/...` exits 1 on one pre-existing
staticcheck QF1003 suggestion at `validation.go:211` (`could use tagged switch
on backend`), which is untouched by this change. The repository declares no
golangci-lint configuration or gate; its declared lint gates are `gofmt` and
`go vet`, both clean.
