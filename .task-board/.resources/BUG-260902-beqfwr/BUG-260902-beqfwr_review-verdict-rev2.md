# BUG-260902-beqfwr — Review verdict rev 2: ACCEPTED

Reviewer run `RUN-260902-80664c`. Change Request `CR-BUG-260902-beqfwr-2` revision 2.
Worktree `.temp/STORY-260902-2230n7/worktree`, branch `task-board/story/STORY-260902-2230n7`.

    base OID        6f7771954ffb4484ff51395c290a82971d2f961e
    candidate tree  a251859a4abb4370be8c1e060de7a78494214ed8
    HEAD^{tree}     a251859a4abb4370be8c1e060de7a78494214ed8   (equal — I reviewed the exact candidate)
    leaf commit     bc3ed70, one commit past the checkpoint

## 1. The reapply is faithful, verified rather than trusted

Revision 2 is a reprovisioning of work already accepted at revision 1
(`BUG-260902-beqfwr_review-verdict.md`, run `RUN-260902-23ff4c`). The
instruction resource says "reapply, do not reimplement", so the first thing to
check is drift.

I applied `BUG-260902-beqfwr_accepted-work.patch` onto a clean extraction of the
checkpoint `6f77719` **outside any git work tree** (`patch -p1` in `/tmp`, so no
repository state could contaminate the comparison) and diffed the result against
the candidate:

| Path | vs accepted patch |
| --- | --- |
| `internal/config/sshargs.go` | identical |
| `internal/config/validation.go` | identical |
| `internal/config/refusal_test.go` | identical |
| `internal/config/schema_test.go` | identical |
| `internal/config/ssh_admission_test.go` | identical |
| `README.md` | identical |
| `LOGBOOK.md` | differs — deliberately |

The one `LOGBOOK.md` divergence is the correction the revision-1 review asked
for as a non-blocking finding: the accepted patch's STATUS line named commit
`7eac813` on the retired branch `STORY-260902-1hrtzp`, an object unreachable
from this leaf. The rev-2 entry names the checkpoint it sits on and no shipping
hash, and adds a REAPPLY line stating that gates and mutants were rerun rather
than inherited. That is the right fix, and it is the only content change.

The accepted patch also applies cleanly onto `6f77719` with no fuzz, so nothing
in the delta was silently reconciled against the advanced base.

## 2. Every gate rerun by me in this workspace

Real exit codes, this worktree, nothing inherited.

| Gate | Exit |
| --- | ---: |
| `gofmt -l .` | 0 (no output) |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `go test ./... -count=1` | 0 — 9/9 packages ok |
| `go test ./... -cover -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 (`contracts=60 normative_sections=36 acceptance_cases=35`) |
| `go run ./internal/traceability/cmd/tracecheck -section 6.3` | 0 (`assigned_scopes=1`) |
| `cataloggen … -check` | 0 |

Coverage, measured by me at both ends rather than quoted:

| Package | Checkpoint `6f77719` | Candidate | Delta |
| --- | ---: | ---: | ---: |
| `internal/config` | 93.7% | 93.9% | +0.2pp |

The checkpoint figure came from `git archive 6f77719` into a scratch directory
and `go test ./internal/config -cover -count=1` there. No package regressed.

## 3. Attack, not reading

### 3.1 The OpenSSH claims reproduce

Every behaviour the bug report and the producer cite was re-driven against the
live binary on this host (`OpenSSH_10.2p1, LibreSSL 3.3.6`):

| argv | `ssh -G` result |
| --- | --- |
| `-vo StrictHostKeyChecking=no` | `stricthostkeychecking false` |
| `-voStrictHostKeyChecking=no` | `stricthostkeychecking false` |
| `-o StrictHostKeyChecking=false` | `stricthostkeychecking false` |
| `-4o UserKnownHostsFile=/dev/null` | `userknownhostsfile /dev/null` |
| `-i -oProxyCommand=id` | no `proxycommand` line — `-i` swallowed the token |
| `-qp2222` | `port 2222` |

The transcribed usage grammar in `sshargs.go:24-31` is byte-for-byte the usage
text this binary prints. The arity of every admitted letter matches.

### 3.2 Differential attack on the admitted set (the part a name blacklist can't survive)

Reading the tables proves nothing about what they let through, so I enumerated
the admitted set instead of inspecting it. A reviewer-only probe drove **6,710
generated argv** — every ASCII letter and digit as a bare flag, as a flag with a
separate value, with an attached value, and inside a group; 80 ssh_config option
names crossed with 13 values, in six spellings each (`-o N=V`, `-oN=V`,
`-vo N=V`, `-o "N V"`, lowercased, quoted); plus optarg-swallowing and mixed
groups — through the production `loadConfigDocument` entry, not through the
helper. **260 were admitted.**

I then replayed all 260 against the live binary and diffed the fully resolved
`ssh -G` output against a `-F /dev/null` baseline. The complete set of resolved
settings that *any* admitted argv can move:

    addressfamily  batchmode  compression  connecttimeout  identitiesonly
    identityfile   loglevel   port         requesttty      serveralivecountmax
    serveraliveinterval       stricthostkeychecking        tcpkeepalive  user

That is exactly the declared surface. `stricthostkeychecking` only ever moves
`ask -> true`, i.e. strictly toward enforcement. Scanning the same 260 replays
for sixteen dangerous keys — `proxycommand`, `localcommand`,
`permitlocalcommand`, `knownhostscommand`, `userknownhostsfile`,
`globalknownhostsfile`, `checkhostip`, `verifyhostkeydns`, `updatehostkeys`,
`forwardagent`, `proxyjump`, `remotecommand`, `localforward`, `remoteforward`,
`dynamicforward`, `revokedhostkeys` — gives **0 divergences from baseline**.

No argv this gate admits resolves to a host-authentication bypass or to command
execution. That is a measured result, not an inference from the tables.

One admitted argv makes ssh complain: `-l -oStrictHostKeyChecking=no` warns
about the pseudo-terminal, having consumed the `-o` token as a username. That is
faithful getopt behaviour and the safe direction.

### 3.3 Reachability grepped, not assumed

`admitSSHArguments` has one caller, `validateMesh` at `internal/config/validation.go:465`.
`validateMesh` is called only from `validateConfiguration` (`:391`), reached from
the three versioned decode paths (`validation.go:53`, `:77`, `:101`) and the
write path (`writer.go:16`). No production path materialises a `Peer.SSHArgs`
without crossing the gate. The dead `sshHostAuthenticationBypass` and
`sshKnownHostsDisabled` helpers are deleted, not orphaned.

### 3.4 Twelve single-clause mutants I applied myself

Each mutant is a *narrowing of one clause* — no delete-only mutants. Every one
was applied to a fresh extraction of the candidate tree in a scratch sandbox, so
the reviewed worktree was never modified (`git status --short` empty throughout,
`HEAD^{tree}` still `a251859…` afterwards).

| # | Single-clause weakening | Result |
| --- | --- | --- |
| M1 | permit `-F` (config-file redirection) | RED — `…OutsideThePermittedTables`, `…GroupedShortFlags/an_unpermitted_letter_inside_a_group` |
| M2 | `stricthostkeychecking` permits `no/off/false/accept-new` | RED — 13 `…OptionSpellingGrammar` subtests |
| M3 | `break` instead of `continue` on a valueless letter (kills group walking) | RED — `…GroupedShortFlags` + the whole derived per-letter walk, reason mismatch |
| M4 | declare `proxycommand` permitted | RED — `…UndeclaredSSHOptionNames/ProxyCommand{,_combined}`, `…AdmitsExactlyTheDeclaredPermitted`, `…RefusesThroughTheMeshPeerClause` |
| M5 | `knownhostscommand` permits a word value | RED — `…UndeclaredSSHOptionNames/KnownHostsCommand`, `…AdmitsExactlyTheDeclaredPermitted` |
| M6 | declare `include` permitted | RED — `…UndeclaredSSHOptionNames/Include`, `…AdmitsExactlyTheDeclaredPermitted` |
| M7 | declare `localcommand` + `permitlocalcommand` permitted | RED — both `…UndeclaredSSHOptionNames` cases, `…AdmitsExactlyTheDeclaredPermitted` |
| M8 | widen `-p` bound to 65536 | RED — `…GroupedShortFlags/a_value-taking_letter_ends_its_group` |
| M9a | stop case-folding the option **name** | RED — 7 `…OptionSpellingGrammar` subtests |
| M9b | stop case-folding the option **value** | RED — `…AdmitsExactlyTheDeclaredPermitted/loglevel` |
| M10 | drop the host-authentication refusal reason | RED — 6 `…OptionSpellingGrammar` + 6 `…DeclaredInTheRegistry` subtests |
| M11 | admit bare operands instead of refusing | RED — `TestLoadRefusesSSHArgumentsThatAreNotOptions` |
| M12 | split the option name on `=` only, not whitespace | RED — 4 `…OptionSpellingGrammar` subtests |

Each reddens cases that name its own clause. M3 is the important one: it
reintroduces the exact reported `-vo` bypass and 61 nodes go red.

`TestLoadAdmitsExactlyTheDeclaredPermittedSSHOptions` asserts key-set equality
between the sample map and the permitted registry in both directions, which is
why M4–M7 cannot widen the `-o` registry quietly. That is the right shape.

### 3.5 Test-suite honesty

Every case drives `loadConfigDocument` through a real TOML document; none calls
`admitSSHArguments` directly. `TestSSHArgumentAdmissionRefusesThroughTheMeshPeerClause`
pins the refusal to the `mesh.peers[0].ssh_args ` prefix so a case cannot pass
on an unrelated bound. `TestLoadAdmitsTheSpecifiedPeerArgumentExample` keeps the
§6.3 example argv loadable, so the gate is not vacuously strict — and my
260-argv admitted set independently confirms it is not.

The `schema_test.go` rewrite is honest: `"x"*4096` was a bare operand and is now
correctly refused, replaced with `-i` + 4094 `x` (identical 4096 bytes) and
`-i` + 4095 `x` (4097 bytes). The length and count bounds run before admission,
so both bounds are still proven at and past the limit.

The old `TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling` name
claimed completeness the code did not have. It is renamed to
`…AcrossTheOptionSpellingGrammar` and its doc comment states where completeness
*is* earned — the registry-derived test plus the closed admission. That
satisfies the AC's "earned or the name is corrected" as *corrected*.

## 4. AC ledger

| AC | Status | Evidence |
| --- | --- | --- |
| Admission derives the permitted set and refuses the complement; an unknown option is refused | met | 6,710-argv probe: 6,450 refused, 260 admitted, and the 260 move only declared settings (§3.2); M4/M6 |
| Grouped short flags parse the OpenSSH way, a case per spelling | met | `…GroupedShortFlags` 4 subtests; arity matches the live usage text; M3 |
| `StrictHostKeyChecking=false` refused, alias cited | met | `…OptionSpellingGrammar/strict_false_alias`; alias reproduced live (§3.1); M2 |
| `-F`, `ProxyCommand`, `KnownHostsCommand`, `Include`, `PermitLocalCommand`+`LocalCommand` each refused at the production entry | met | `…UndeclaredSSHOptionNamesAtTheProductionEntry`, `…OutsideThePermittedTables`; M1, M4–M7 |
| Each negative case reddens under its own single-clause weakening | met for every cited bypass | twelve reviewer mutants, §3.4; see finding 1 for the uncited residue |
| Completeness claim earned or name corrected | met | renamed; doc comment names where completeness is earned |
| Gates exit 0, no coverage regression | met | §2, 93.7% → 93.9% |

## 5. Findings recorded, not blocking

1. **The short-flag tables have no both-directions pin, so widening them is
   silent.** `permittedSSHOptionSamples` pins the `-o` registry key set exactly —
   that is why M4–M7 all redden. `sshPermittedFlags` / `sshPermittedValueFlags`
   have no analogue. I confirmed the consequence with three more mutants that
   the **entire suite passes green**:

   | Mutant | Widening | Suite |
   | --- | --- | --- |
   | M13 | permit `-E` (ssh log file — an arbitrary-file-write primitive) | GREEN |
   | M14 | permit `-L` (local port forwarding) | GREEN |
   | M15 | permit `-A` (agent forwarding — the exact capability `-a` is permitted to *disable*) | GREEN |

   `TestLoadRefusesEverySSHShortOptionOutsideThePermittedTables` derives its
   cases per letter and *skips* permitted ones, so permitting a letter deletes
   its own subtest rather than failing it. The producer identified this shape in
   the logbook and mitigated it with explicitly named assertions — but only for
   the two letters that can reach this leaf's normative class: `-F` (config
   redirection, M1 reddens) and `-J` (jump host). Both are covered.

   This is why it is a finding and not rework: `-E`, `-L`, `-R`, `-D`, `-W`,
   `-S`, `-A` are capability widenings, not §6.3 host-authentication bypasses,
   and the production gate refuses all of them correctly today (verified in
   §3.2). The gap is in mutation coverage of a non-normative area. The fix is
   the existing pattern applied to a second pair of tables — a pinned expected
   flag set with an admitted/refused sample per letter — and it belongs in the
   next ssh_args leaf, not in a reapply revision.

2. **`parseSSHConfigOption` over-admits names that ssh will reject at exec.**
   It splits on any `unicode.IsSpace` rune and folds with `strings.ToLower`,
   while OpenSSH splits on ASCII `" \t\r\n"`/quote/`=` and compares with
   `strcasecmp`. So `-o $'BatchMode yes'` is admitted by AX while ssh answers
   `Bad configuration option`. The divergence is one-directional — AX truncates
   the name at the separator and then *requires registry membership*, so no
   dangerous keyword reaches the permitted set this way. Failure mode is an ssh
   startup error, never a relaxed host check. Carried forward from the rev-1
   review, still accurate.

3. **The registry refuses `UserKnownHostsFile` and `GlobalKnownHostsFile` for
   every value**, which is stricter than §6.3's "an empty `UserKnownHostsFile`".
   §6.3 constrains what MUST be refused and the example argv still loads, so this
   is conformant. Flagged so a future legitimate custom-known-hosts request is a
   deliberate registry widening with both-direction samples rather than a
   surprise.

## Verdict

**ACCEPTED.** The reapply is byte-identical to the accepted work for all code
and documentation, with the single deliberate `LOGBOOK.md` correction the
revision-1 review asked for. All eight gates exit 0 in this workspace and
`internal/config` coverage moves 93.7% → 93.9% against a checkpoint figure I
measured myself. Twelve single-clause narrowing mutants each redden cases that
name their own clause. And the attack that matters for an equivalence-class
requirement — enumerating the 260 argv the gate actually admits and replaying
every one against OpenSSH 10.2p1 — finds zero divergences on sixteen dangerous
resolved settings and no movement of `stricthostkeychecking` except toward
enforcement.

Reviewer-archetype run: **no `commit_ack` supplied**. The commit-owning mover
should treat the single signed commit `bc3ed70` past checkpoint `6f77719` as
this leaf's scope.
