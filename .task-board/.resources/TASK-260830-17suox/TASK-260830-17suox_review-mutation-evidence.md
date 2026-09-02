# TASK-260830-17suox review — gate attack evidence

Reviewer run: RUN-260901-f3815d
Candidate: CR-TASK-260830-17suox-2 rev 2
Base OID: 020d0b6c68c587b6463add58330050ceff71b87f
Candidate tree OID: 1234f3c9ff9caee7015d2fcc4db4f5cf6777ca12
Worktree tree OID recomputed by reviewer (before and after probes): 1234f3c9ff9caee7015d2fcc4db4f5cf6777ca12
Spec authority: local clone of relux-works/agent-session-manager-spec at
28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c, SPEC.md sha256
562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a (matches .spec/README.md pin).

## Baseline

    go build ./...        exit 0 (no output)
    go vet ./...          exit 0 (no output)
    go test ./... -count=1  exit 0, all 9 packages ok
      (log: .temp/TASK-260830-17suox/review-go-test.log)

## Finding 1 — SSH host-authentication bypass gate admits documented OpenSSH aliases

Production call site: internal/config/validation.go `sshHostAuthenticationBypass`,
reached from `validateMesh` <- `validateConfiguration` <- `translateV1/V2/V3` <-
`Decode` <- `config.Load`.

Probe driven through the production entry `config.Load` (temporary test file,
removed after the run):

    strict off            ssh_args = ["-o", "StrictHostKeyChecking=off"]        -> err=<nil>   ACCEPTED
    strict off combined   ssh_args = ["-oStrictHostKeyChecking=off"]            -> err=<nil>   ACCEPTED
    known hosts none      ssh_args = ["-o", "UserKnownHostsFile=none"]          -> err=<nil>   ACCEPTED
    global known none     ssh_args = ["-o", "GlobalKnownHostsFile=none"]        -> err=<nil>   ACCEPTED
    strict no (control)   ssh_args = ["-o", "StrictHostKeyChecking=no"]         -> refused

Platform authority, ssh_config(5) on this host:

    StrictHostKeyChecking ... "If this flag is set to no or off, ssh will
    automatically add new host keys to the user's known_hosts file, and allow
    connections to hosts with changed hostkeys to proceed."

    UserKnownHostsFile ... "A value of none causes ssh(1) to ignore any
    user-specific known hosts files."

SPEC.md §6.3: "SSH arguments that set StrictHostKeyChecking=no, an empty
UserKnownHostsFile, or an equivalent host-authentication bypass MUST fail
configuration validation."

`off` is the documented alias of the exact value the spec names. `none` is the
documented spelling of "no user known hosts file", the class the code already
handles for "" and /dev/null. The test
`TestLoadRefusesEveryOpenSSHHostAuthenticationBypassSpelling` enumerates nine
spellings and its name claims completeness, but the vocabulary is not derived
from the OpenSSH value grammar, so the missing aliases are invisible to it.

## Finding 2 — refusal clauses that survive individual disablement

Method: each clause disabled in isolation in internal/config/validation.go or
internal/config/schema.go, then `go test ./internal/config -count=1`. Source
restored after every mutant; final tree OID re-verified as 1234f3c9.

killed (suite reddened):
    delete ssh host-auth bypass gate
    delete require_explicit_trust gate
    delete runtime platform probe match
    delete missing backend-settings-schema refusal
    delete ssh argv total byte bound
    delete ssh arg count bound
    delete extension count bound
    delete extension map depth bound
    delete extension byte bound
    delete legacy terminal vocabulary gate
    delete registered-backend selection gate
    delete backend_config registration gate
    delete chunk_bytes constant
    delete v3 unknown-field strictness (DisallowUnknownFields)
    delete schema identifier gate
    delete freshness aging ordering
    delete query page ordering

SURVIVED (suite still green with the clause disabled):
    delete forbidden-name gate            hasForbiddenConfigName -> always false
    delete printable/control check        unicode.IsControl / !unicode.IsPrint
    delete printable invalid-UTF8 check   utf8.ValidString in validatePrintableCharacters
    delete invalid-UTF8 ssh arg           utf8.ValidString on ssh_args element
    delete NUL byte in ssh arg            strings.IndexByte(argument, 0) >= 0
    delete extension reverse-DNS namespace  reverseDNSPattern on extension keys
    delete extension key length bound     len(key) < 3 || len(key) > 253
    delete extension float refusal        case float64, float32 -> return nil
    delete extension list depth bound     []any depth >= 4
    delete extension nested forbidden-name  hasForbiddenConfigName inside maps
    delete backend_id 128-byte bound      terminal.backend_id length
    delete external trust backend_id byte bound  external_trust[].backend_id length

NARROWING mutants that survived (delete-only would have been killed; narrowing
proves the uniqueness half of "sorted unique" is unpinned):
    validateSortedUniqueClosed  values[index-1] >= value  ->  > value   SURVIVED
    validateSortedUniqueDigests values[index-1] >= value  ->  > value   SURVIVED

Production does reject those inputs today; no test would notice if it stopped.
`hasForbiddenConfigName` is the §6.4 "no secret, endpoint credential, model
token, auth root, or arbitrary environment passthrough" rule and has zero
negative coverage. The two backend_id byte bounds appear mutually subsuming for
the single fixture that exercises them; that subsumption is not documented and
the subsuming check is not named.

## Finding 3 — external_trust `enabled` is required but never consulted

Probe through `config.Load`:

    [terminal]
    backend_id = "com.example.term"
    [[terminal.external_trust]]
    backend_id = "com.example.term"
    executable_path = "/opt/example/terminal"
    executable_digest = "sha256:000...0"
    enabled = false

    -> err=<nil>   ACCEPTED

internal/config/validation.go `validateTerminal` writes
`registered[entry.BackendID] = struct{}{}` for every trust entry regardless of
`entry.Enabled`, so a backend whose only trust entry is disabled still satisfies
the "terminal.backend_id is not registered" gate. `Enabled` is decoded, presence-
checked, cloned and re-emitted, but no production path reads its value.

SPEC.md §6.5: "Duplicate IDs or config entries, unknown IDs/settings, ambiguous
discovery, manifest/probe/config mismatch, unsupported platform, unavailable
required capability, or untrusted executable fails configuration/activation."

## Observation A — empty explicit transport_policy accepted

    Terminal.TransportPolicy = []string{}, TransportPolicyExplicit = true
    -> EncodeCurrent err=<nil>   ACCEPTED

§6.5 gives transport_policy as a "sorted unique subset of
local_only|trusted_private_mesh; default both". An empty subset admits no
transport at all. Whether the empty set is legal is genuinely ambiguous in the
text; flagging it so the decision is explicit rather than incidental.

## Observation B — section:17.5 traceability binding

internal/traceability/ownership.v0.5.0.json binds `section:17.5` to
internal/config/validation.go `translateV2`. The only §17.5 sentence that names
Configuration behavior is "Config 2 has the explicit backup/atomic migration and
read-only downgrade behavior in Section 6.4" — and README.md correctly states
that durable `ax migrate config` and downgrade diagnostic behavior remain
separate implementation scope. The rest of §17.5 is Directory Node / Mesh RPC
release compatibility that this package does not touch. §6.4 carries a milder
version of the same shape.

## What was verified positively

- Reader/writer dispatch for 1.0.0 / 2.0.0 / 3.0.0 is derived from the pinned
  catalog contract, not from a hand-written list; the terminal capability
  closure is derived from the pinned catalog `terminal_backend` family. Both
  fail closed when the pinned source grows a member.
- Closed-shape strictness is real per version: a v1 document carrying
  `[directory]`, `terminal.backend_id`, or any unknown root key is refused by
  `DisallowUnknownFields` against the version-specific struct, and a v3 document
  carrying legacy `terminal.backend` is refused the same way.
- Legacy translation maps only tmux|conpty and refuses any other
  `terminal.backend` value before translation, per §6.5.
- Both writers and readers share one `validateConfiguration` gate, so a value
  that cannot be read cannot be written.
- Errors render a static clause and do not echo rejected machine-local values.
- The package writes no file and performs no migration, and README says so
  rather than claiming migration support.
