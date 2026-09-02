# BUG-260902-beqfwr — reapply re-verification evidence

Workspace `WS-3a91ab1688d9`, branch `task-board/story/STORY-260902-2230n7`,
checkpoint `6f7771954ffb4484ff51395c290a82971d2f961e`, leaf exactly one signed
single-parent commit past it.

The accepted work was reapplied from `BUG-260902-beqfwr_accepted-work.patch`
(`git am --3way`, applied clean), not reimplemented. Every gate and every mutant
below was rerun in this workspace; nothing here is inherited from the earlier
run.

One correction was made on top of the accepted patch: its `LOGBOOK.md` STATUS
line named commit `7eac813` on the retired branch `STORY-260902-1hrtzp`, an
object unreachable from this leaf. The line now records the checkpoint it sits
on and names no shipping hash.

## Cited OpenSSH behaviours, reproduced against the live binary

Host binary: `OpenSSH_10.2p1, LibreSSL 3.3.6`.

| argv | ssh -G output | meaning |
| --- | --- | --- |
| `ssh -G -vo StrictHostKeyChecking=no host` | `stricthostkeychecking false` | grouped short flag reaches the option; the old whole-token `-o` scan never saw it |
| `ssh -G -o StrictHostKeyChecking=false host` | `stricthostkeychecking false` | `false` is a live alias absent from the old `no\|off` match |
| `ssh -G -4o UserKnownHostsFile=/dev/null host` | `userknownhostsfile /dev/null` | second grouped-flag bypass |
| `ssh -G -i -oProxyCommand=id host` | no `proxycommand` line | `-i` consumes the next argv as its filename; admitting it is faithful, not a hole |
| `ssh -G -qp2222 host` | `port 2222` | a value-taking letter ends its group and takes the rest as its value |

## Gates

Each command was run as a standalone process with its output redirected to a
file; the exit code reported is the command's own.

| Gate | Command | Exit |
| --- | --- | ---: |
| Build | `go build ./...` | 0 |
| Vet | `go vet ./...` | 0 |
| Format | `gofmt -l .` (0 files listed) | 0 |
| Tests | `go test ./... -count=1 -cover` | 0 |
| Traceability | `go run ./internal/traceability/cmd/tracecheck` | 0 |
| Traceability, assigned scope | `go run ./internal/traceability/cmd/tracecheck -section 6.3` (`assigned_scopes=1`) | 0 |
| Catalog | `go run ./internal/catalog/cmd/cataloggen ... -check` | 0 |

## Coverage

| Package | Checkpoint `6f77719` | This leaf | Delta |
| --- | ---: | ---: | ---: |
| `internal/config` | 93.7% | 93.9% | +0.2pp |

The checkpoint figure was measured in a detached worktree at `6f77719`, not
quoted from the earlier run. No other package changed.

## Single-clause mutants

Each mutant weakens exactly one clause of `internal/config/sshargs.go`, then
`go test ./internal/config -count=1` runs and the tree is restored. All ten
reddened; none survived.

| Mutant | Exit | Top-level tests that failed |
| --- | ---: | --- |
| M1 permit `-F` | 1 | GroupedShortFlags, EverySSHShortOptionOutsideThePermittedTables, ShortOptionTablesAreDerived |
| M2 declare `ProxyCommand` permitted | 1 | AdmitsExactlyTheDeclaredPermitted, GroupedShortFlags, UndeclaredSSHOptionNames, RefusesThroughTheMeshPeerClause |
| M3 widen `StrictHostKeyChecking` to `no/off/false/accept-new` | 1 | GroupedShortFlags, OptionSpellingGrammar, SecurityBypassesDuplicatesAndUnknownBackendSettings |
| M4 stop walking a group past its first letter | 1 | six top-level tests, 61 failing nodes |
| M5 declare `Include` permitted | 1 | AdmitsExactlyTheDeclaredPermitted, UndeclaredSSHOptionNames |
| M6 declare `KnownHostsCommand` permitted | 1 | AdmitsExactlyTheDeclaredPermitted, UndeclaredSSHOptionNames |
| M7 declare `PermitLocalCommand`+`LocalCommand` permitted | 1 | AdmitsExactlyTheDeclaredPermitted, UndeclaredSSHOptionNames |
| M8 admit bare operands | 1 | RefusesSSHArgumentsThatAreNotOptions |
| M9 admit undeclared option names | 1 | GroupedShortFlags, UndeclaredSSHOptionNames, RefusesThroughTheMeshPeerClause |
| M10 let `UserKnownHostsFile` permit every value | 1 | GroupedShortFlags, EveryHostAuthenticationOptionDeclaredInTheRegistry, OptionSpellingGrammar |

M1 reddens on the explicitly named `-F` assertions, not on the derived
per-letter loop: permitting a letter deletes its own derived subtest rather than
failing it. That is why the cited bypasses carry named assertions alongside the
derivation.

Production call site under attack throughout:
`internal/config/validation.go:465`, `validateMesh` →
`admitSSHArguments(peer.SSHArgs)`, reached from `loadConfigDocument`. Every case
above drives that entry through a real TOML document, not the helper directly.

Mutant runner and raw logs: `.temp/BUG-260902-beqfwr/` in the Story worktree
(`mutants.py`, `mutants-01.log`, `go-test-final.log`, `tracecheck-final.log`,
`baseline-cover.log`).
