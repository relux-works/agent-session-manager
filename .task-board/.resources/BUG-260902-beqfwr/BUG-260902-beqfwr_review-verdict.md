# BUG-260902-beqfwr — Review verdict: ACCEPTED

Reviewer run: RUN-260902-23ff4c. Change Request `CR-BUG-260902-beqfwr-1` revision 1.
Worktree `.temp/STORY-260902-1hrtzp/worktree`, branch `task-board/story/STORY-260902-1hrtzp`.

## 1. Why `repository_delta=empty` is not a producer no-op here

The CR snapshot reports an empty delta because its base OID and its candidate
tree are the same object:

    base OID          150040fef692e832c28f557cad5904e219811b9b   (= branch HEAD)
    candidate tree    7fce177f18f8aa3975002b466b2434f4f98737b6
    HEAD^{tree}       7fce177f18f8aa3975002b466b2434f4f98737b6
    150040f^{tree}    7fce177f18f8aa3975002b466b2434f4f98737b6

The producer committed its work to the story branch *before* the snapshot was
taken, so the base was computed at the post-commit head rather than at the
checkpoint. The delivered change is real and fully present:

    git rev-list --count main..HEAD  -> 1
    git rev-list --count HEAD..main  -> 0

The single commit past `main` is `150040f "BUG-260902-beqfwr: derive ssh_args
admission instead of blacklisting names"`, signed and verified
(`Good "git" signature for oparin@me.com`, ECDSA
SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM), touching 7 files
(+609/-55). That commit is what this review attacked. Accepting the empty
revision therefore accepts commit `150040f` as the reviewable delta, and the
orchestrator's checkpoint should treat `main..HEAD` as this leaf's scope.
No repository change was made *after* the commit, which is the right outcome
for this leaf: the leaf's work is the commit.

## 2. The change

`sshHostAuthenticationBypass` (three-name blacklist, `validation.go:846-871`)
and `sshKnownHostsDisabled` are deleted. `internal/config/sshargs.go` declares:

- `sshShortOptionsWithoutValue` / `sshShortOptionsWithValue` — the ssh(1)
  short-option arity transcribed from the OpenSSH usage text;
- `sshPermittedFlags` = `{4 6 C T a q v}`, `sshPermittedValueFlags` =
  `{i l p o}` with per-value rules;
- `sshOptionRegistry` — the closed `-o` name set with a value rule each.

`admitSSHArguments` walks argv with getopt semantics and returns a refusal
reason for the complement. Single production call site remains
`validation.go:465`.

## 3. Attack, not reading

### 3.1 Production reachability of the gate (grepped, not assumed)

`admitSSHArguments` has exactly one caller: `validateMesh` (`validation.go:465`).
`validateMesh` is called only from `validateConfiguration` (`:391`), which is
reached from every path by which `ssh_args` can enter or leave the process:
`validation.go:53`, `:77`, `:101` (the versioned decode paths) and
`writer.go:16` (the write path). There is no production path that materialises
a `Peer.SSHArgs` without crossing the gate. No `ssh` exec site exists in the
repository yet, so the config gate is the whole current surface.

### 3.2 Differential attack against the live binary (OpenSSH_10.2p1, LibreSSL 3.3.6)

Arity of every admitted letter was re-derived from the binary, not from the
comment. The 7 valueless letters each leave the destination intact
(`ssh -G -<f> -o BatchMode=yes h` resolves `host h`); each of `i l p o`
consumes the following token (`ssh -G -i h` warns on identity file `h` then
fails for a missing destination; `-p h` -> `Bad port 'h'`; `-o h` ->
`no argument after keyword "h"`; `-l h` -> usage). The tables match the binary.

63 adversarial argv were driven through the production `loadConfigDocument`.
Every dangerous spelling is REFUSED, including ones the producer never listed:
`ProxyJump`, `-J`, `CertificateFile`, `PKCS11Provider`, `SetEnv`, `SendEnv`,
`RemoteCommand`, `ControlMaster`/`ControlPath`, `PubkeyAuthentication=no`,
`PasswordAuthentication`, `HostName`, `Host=*`, `CanonicalizeHostname`,
`ForwardAgent`, `-A`, `Tunnel`, `FingerprintHash=md5`, `RequiredRSASize=1024`,
`PubkeyAcceptedAlgorithms=+ssh-rsa`, `CASignatureAlgorithms=+ssh-rsa`,
`RevokedHostKeys=/dev/null`, `SecurityKeyProvider`, `ObscureKeystrokeTiming`,
plus all eight bypasses named in the report.

Name-parsing divergence was attacked specifically, since the only remaining
way through a closed registry is to make AX read a permitted name where ssh
reads a dangerous one:

| Spelling | AX | live ssh | Verdict |
| --- | --- | --- | --- |
| `-o '"StrictHostKeyChecking"=no'` | refuse (undeclared name) | `unsupported option "=no"` | closed both sides |
| `-o $'BatchMode=yes\nProxyCommand=id'` | refuse (unpermitted value) | error, no `proxycommand` set | closed both sides |
| `-o 'batchmode=yes proxycommand=id'` | refuse | — | closed |
| `-o 'Port=+22'`, `-o 'Port=2_2'` | refuse | — | closed |
| `-oi /tmp/x` | refuse (name `i` undeclared) | error | closed |

Injection through the *value* of a permitted option cannot widen the name set:
AX truncates the name at the first `=` or Unicode space and requires the result
to be in the registry, and OpenSSH refuses a second directive in one `-o` with
"garbage at end of line".

### 3.3 The admitted set, checked against ssh rather than assumed

The one class that could diverge is a value-taking letter swallowing a token
that *looks* like an option. AX admits `-i -oProxyCommand=id`,
`-l -oStrictHostKeyChecking=no`, `-i -F/tmp/attacker_config`. Verified against
the binary that OpenSSH consumes them identically as optargs:

    ssh -G -F/tmp/ax_probe_cfg h        -> proxycommand id      (config WAS read)
    ssh -G -i -F/tmp/ax_probe_cfg h     -> no proxycommand line (consumed by -i)
    ssh -G -i -oProxyCommand=id h       -> no proxycommand line
    ssh -G -l -oStrictHostKeyChecking=no h -> "remote username contains invalid characters"

So AX's getopt model is faithful, and a token scanner that "found"
`-oProxyCommand=` inside `-i`'s optarg would have been the wrong answer. The
producer's logbook entry states exactly this and it reproduces.

### 3.4 Mutants I ran myself (five single-clause narrowings, no deletions)

Applied to `internal/config/sshargs.go`, one clause each, reverted between runs.

| # | Weakening | Reddens |
| --- | --- | --- |
| M1 | add `'F': sshFlagValueRule(sshWordValue)` to `sshPermittedValueFlags` | `TestLoadRefusesEverySSHShortOptionOutsideThePermittedTables`, `…GroupedShortFlags/an_unpermitted_letter_inside_a_group_is_refused` |
| M2 | `stricthostkeychecking` permits `"yes","no"` | `…SpellingGrammar` 8 subtests, `…GroupedShortFlags` 2 subtests, `TestLoadRefusesSecurityBypassesDuplicatesAndUnknownBackendSettings/host-key_bypass` |
| M3 | declare `"proxycommand": {permits: sshWordValue}` | `TestLoadRefusesUndeclaredSSHOptionNames…/ProxyCommand{,_combined}`, `/option_name_only`, `TestLoadAdmitsExactlyTheDeclaredPermittedSSHOptions` (key-set equality), `TestSSHArgumentAdmissionRefusesThroughTheMeshPeerClause` |
| M4 | `if position != 0 { continue }` in the group walk — reintroduces the exact reported `-vo` bypass | all 4 `…GroupedShortFlags` subtests + the whole derived per-letter walk |
| M5 | `continue` instead of `return sshRefusalUnpermittedArgument` for bare operands | `TestLoadRefusesSSHArgumentsThatAreNotOptions` |

Control run after reverting: `ok internal/config 0.885s`. Working tree clean
before and after (`git status --short` empty). Every mutant is a narrowing of a
single clause, and each reddens cases that name its own clause — no
delete-only mutant, no clause proved solely by absence.

## 4. Test-suite quality

Every case drives the real entry `loadConfigDocument` through
`peerDocumentWithSSHArgs`, and `TestSSHArgumentAdmissionRefusesThroughTheMeshPeerClause`
pins the refusal to the `mesh.peers[0].ssh_args ` clause so a case cannot pass
on an unrelated bound. `TestLoadAdmitsExactlyTheDeclaredPermittedSSHOptions`
asserts key-set equality between the sample map and the permitted registry, so
widening the registry without proving both directions reddens (M3 confirms).
`TestLoadAdmitsTheSpecifiedPeerArgumentExample` keeps the SPEC §6.3 example
`["-o", "BatchMode=yes"]` loadable, so the gate is not vacuously strict.

The `schema_test.go` rewrite is honest: `"x"*4096` was a bare operand, now
correctly refused, and was replaced with `-i` + 4094 x (identical 4096 bytes)
and `-i` + 4095 x (4097 bytes). The length and count bounds run before
admission, so both bounds are still proven at and past the limit.

## 5. AC ledger

| AC | Status | Evidence |
| --- | --- | --- |
| Admission derives the permitted set and refuses the complement; unknown option refused | met | `admitSSHArguments`; 63-case probe, `ThisOptionDoesNotExistYet=1` refused; M3/M4 |
| Grouped short flags parse the OpenSSH way, case per spelling | met | `…GroupedShortFlags` 4 subtests; arity re-derived from the live binary; M4 |
| `StrictHostKeyChecking=false` refused, alias cited | met | `…SpellingGrammar/strict_false_alias`; alias reconfirmed via `ssh -G` |
| `-F`, `ProxyCommand`, `KnownHostsCommand`, `Include`, `PermitLocalCommand`+`LocalCommand` each refused at the production entry | met | `…UndeclaredSSHOptionNamesAtTheProductionEntry`, `…OutsideThePermittedTables`; independently re-probed |
| Each negative case reddens under its own single-clause weakening | met | five reviewer-run mutants, section 3.4 |
| Completeness claim earned or name corrected | met | renamed to `TestLoadRefusesHostAuthenticationBypassAcrossTheOptionSpellingGrammar`; doc comment names where name-completeness is earned (`…DeclaredInTheRegistry` + closed admission) |
| Gates exit 0, no coverage regression | met | section 6 |

## 6. Gates re-run by the reviewer (real exit codes, this worktree)

| Gate | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` | 0 (no output) |
| `go test ./... -count=1` | 0 (9/9 packages ok) |
| `go test ./... -cover -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `go run ./internal/traceability/cmd/tracecheck -section 6.3` | 0 |
| `cataloggen … -check` | 0 |

`internal/config` coverage 93.9%; no package below its prior level. Working
tree clean after all reviewer probes were removed.

## 7. Non-blocking findings for the orchestrator

1. **Stale commit SHA in `LOGBOOK.md`.** The new entry's STATUS line names
   commit `7eac813`. That object exists but is *not* reachable from HEAD and
   carries a different tree (`964d44e…` vs the delivered `7fce177…`): the
   producer committed once, then re-committed as `150040f`. The logbook
   therefore points at an abandoned commit. Correct it to `150040f` during
   checkpoint/integration. Not grounds for rework — the SHA cannot be known
   inside the commit that carries it, and nothing else in the entry is wrong.

2. **Over-admission that fails closed at ssh, not a bypass.**
   `parseSSHConfigOption` splits the name on any `unicode.IsSpace` rune and
   lowercases with `strings.ToLower`, while OpenSSH splits on ASCII
   `" \t\r\n"`/quote/`=` and compares with `strcasecmp`. So
   `-o $'BatchMode yes'` is ADMITTED by AX while ssh answers
   `Bad configuration option`. The divergence is one-directional: AX truncates
   the name at the separator and then requires registry membership, so no
   dangerous keyword can reach the permitted set this way — the failure mode is
   an ssh startup error, not a relaxed host check. Confirmed against the binary.
   Worth a follow-up only if AX later wants to reject at config time what ssh
   will reject at exec time.

3. The registry refuses `UserKnownHostsFile` and `GlobalKnownHostsFile` for
   *every* value, which is stricter than SPEC §6.3's "an empty
   `UserKnownHostsFile`". SPEC constrains only what MUST be refused and the
   §6.3 example argv still loads, so this is spec-conformant. Flagged so a
   future legitimate-custom-known-hosts request is a deliberate registry
   widening with both-direction samples, not a surprise.

## Verdict

**ACCEPTED.** The gate now derives admission from declared sources and refuses
the complement; every bypass in the report reproduces as a refusal at the
production entry; the negative suite is driven from `loadConfigDocument` and
reddens under single-clause narrowings I applied myself; and my independent
attack — including 23 option names the producer never enumerated and a
differential check against the live OpenSSH 10.2p1 binary — found no admitted
argv that resolves to a host-authentication bypass or command execution.

Reviewer-archetype run: no `commit_ack` supplied. The commit-owning mover
should treat `main..HEAD` (commit `150040f`) as this leaf's scope.
