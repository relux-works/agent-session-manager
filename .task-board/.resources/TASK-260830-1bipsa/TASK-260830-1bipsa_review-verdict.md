# TASK-260830-1bipsa — review verdict: ACCEPTED

Leaf 2 of 3, `implement-cli-result-envelopes`. Reviewed by RUN-260903-ac1cb0 against
`CR-TASK-260830-1bipsa-1` revision 1.

## 1. Why `repository_delta=empty` is not this producer's failure

The Change Request records base OID `f34d91d` and candidate tree `4eed7417`, which are
the same tree, so the patch has zero paths. **`f34d91d` is itself the producer's leaf
commit** — the CR base was pinned at the leaf instead of at its parent checkpoint
`ebc4e31`. Measured here:

```
git rev-parse HEAD^{tree}    = 4eed74176c37ffcd4e006d45142c09597fed5e5c
git rev-parse f34d91d^{tree} = 4eed74176c37ffcd4e006d45142c09597fed5e5c   (base == candidate)
git diff --stat ebc4e31 f34d91d = 31 files changed, 7241 insertions(+), 36 deletions(-)
```

This is the CR-snapshot construction bug the orchestrator already filed as #113, not a
producer that changed nothing. **The reviewable delta is `ebc4e31..f34d91d` and that is
what I reviewed.** The worktree is clean and `HEAD^{tree}` equals the candidate tree, so
the artefact under review is exactly the one the CR names. Accepting is therefore not
"accepting an empty change": it is accepting 31 files the snapshot failed to frame.

## 2. What I attacked, and with what

I did not re-read the producer's logs and agree with them. I wrote an independent
out-of-tree attack suite (`probe` package under `.temp/`, importing `internal/cliresult`,
`internal/axerror`, `internal/specdoc` and driving the real exported entry points),
mutated production constants by hand, and re-ran the producer's own harness. The probe is
attached as `TASK-260830-1bipsa_review-probe.md`. It was never committed; the reviewed
tree is byte-identical before and after.

### 2.1 Leaf 1's aliasing defect does NOT repeat (attack 1, the priority item)

Leaf 1 shipped a shallow `cloneDetails` that handed out the live nested container. I
probed the new package in both directions and on both construction paths:

| Attack | Result |
| --- | --- |
| Write through every container the caller passed to `New`, at every depth, after `New` returned | encoded output unchanged; no poison on the wire |
| Write through `Body()`, at every depth, then re-encode and re-read `Body()` | unchanged; fresh `Body()` clean |
| Write through `Extension()` | unchanged |
| Same, on a **decoded** result (the other construction path) | unchanged |
| `IDs.WithOperation` mutating its receiver | receiver unchanged; derived value correct |

`New` rebuilds the body from canonical bytes (`adoptObject`) and both accessors
deep-copy (`cloneObject`/`cloneValue`), so the defect class is removed by construction,
not by a test. Confirmed empirically, not read.

### 2.2 Gates attacked, not read

Every one of these is a negative arm I wrote and ran; all held.

- **`--yes` cannot bypass an expectation flag.** For both governed operations, dropping
  each expectation flag in turn while `--yes` is present is refused with exit 2; the full
  set without `--yes` non-interactively is refused with exit 16; interactive with `--yes`
  but no expectation flags is refused; unrelated flags (`--force`, `--yes`, `--whatever`)
  are not accepted as expectation flags; a nil invocation is refused rather than read as
  "interactive, so prompt". Interactive with the full set correctly demands a prompt.
- **`--yes` admission.** Every surface without a documented confirmation refuses `--yes`
  with exit 2; exactly three accept it (`materialize`, `stop`, `takeover`).
- **`rpc serve` refuses `--json`** in every spelling tried (`--json`, `--json=true`,
  after `--stdio`, after `--non-interactive`), and its `Mode()` can never be JSON.
- **Stream discipline.** A second document on stdout is refused; human text in JSON mode
  is refused; text mode with no rendering is refused; an outcome with both or neither is
  refused; a prompt under `--non-interactive` is refused *and writes nothing*; progress on
  a non-TTY is dropped and reports that it was dropped; a log never reaches stdout in
  either mode; a text-mode failure goes to stderr and never to stdout.
- **`Decode` refusals**, ~50 wire mutations: `ok=false` and every non-boolean `ok`; wrong
  schema; 8 bad/unregistered/wrong-major `schema_version` forms; an unknown top-level
  member; each of the 8 declared members omitted; a forbidden identifier made non-null;
  a non-UUIDv7 identifier; `body`/`extensions` as null/array/string/number; a clone tag
  inside a 1.0.0 envelope; an unknown tag; a list body under the `doctor` tag; an unknown
  body member; trailing content; a duplicate member; empty and malformed input. All
  refused.
- **Takeover adoption (clauses 14.2#8/#9).** All 8 kind×adopted×resumed combinations
  behave exactly as the pinned sentence requires: task-board `resumed=true, adopted=false`
  refused, direct `adopted=true` refused, the other six admitted. The rule cannot be
  skipped by omitting the kind (refused), forging it (`taskboard`, `TASK_BOARD`,
  `"direct "`, `0` all refused), or supplying one to a command that takes none. `Decode`
  admits a violating document — documented, because the document carries no kind — and
  `VerifyTakeoverAdoption` then catches it, including for a bogus kind.
- **Unbuilt versions are refused, never emitted.** All 8 `session.clone.*` tags select
  2.0.0 and `New` refuses them with `ErrUnimplementedVersion` for any body; the 3.0.0 and
  4.0.0 tags likewise; a tag that exists nowhere is `ErrUnknownCommand` — the two facts
  stay distinct.
- **Exit-code mapping.** For every registered code in the 1.0.0 registry, `ExitStatus`
  and the real `Emit` path both return exactly the registry's status, the emitted document
  is a Structured Error (never a CLI Result with `ok`), and its `exit_code` member equals
  the returned process status. 17 distinct failure statuses reached; none maps to 0;
  success is always 0.

### 2.3 R2 (the miscount) — closed at source, and the derivation is real

I measured Section 15.2 from the digest-verified pinned document myself through
`specdoc.SectionID`/`TableRowAt`: **18 body rows**. The repository's own
`pinnedExitStatusRows` derives the same way — it is not a cached literal. I mutated the
production registry four ways, each verified present before measuring:

| Mutant | Result |
| --- | --- |
| delete row 17 from `exitMeanings` | RED |
| add a forged row 99 | RED |
| **renumber 16 → 18 (count unchanged)** | RED |
| `successExit` 0 → 2 | RED |

The same-count renumbering reddening is the decisive one: the pin is not count-only.
Surviving "nineteen" strings in the tree are comments that *record* the error plus
superseded LOGBOOK history (entry 21 explicitly supersedes 109); board resources of other
tasks are historical evidence and correctly untouched.

### 2.4 R1 (`minRedactableCause`) — closed, with one accuracy finding

Mutating the constant, each mutant verified present:

| `minRedactableCause` | Suite |
| ---: | --- |
| 7 (narrowing) | RED |
| 9 (narrowing) | RED |
| 16 | RED |
| 63 | RED |

Previously 16 and 64 left the suite green. The residual is closed: a silent widening is
now impossible.

**Finding (accepted, not rework):** the pin works via the literal `minRedactableCause != 8`
guard, *not* via the two boundary fixtures. Both fixtures are computed from the constant
(`strings.Repeat("s", minRedactableCause)`), so they slide with it. I proved this by
removing the guard and widening the constant to 63: the suite went **GREEN**. The test's
own comment — "The two rows below are one character apart on either side of the constant,
so moving it in either direction reddens exactly one of them" — is therefore false about
its own mechanism, and the task note's "pinned one character on either side of 8"
overstates what the fixtures alone achieve.

This does not reopen the leaf. The residual's purpose was that a widening must not be
silent, and it is not: 63 reddens loudly. A literal equality on a magic constant plus
behaviour verified at that value is a legitimate pin. What is wrong is a *comment* about
why the test catches drift. Correct the comment (and, if wanted, hard-code the fixtures at
8 and 7 so the behavioural arms pin independently) in leaf 3 or a follow-up.

### 2.5 Section 15.2 was NOT re-bound, and the claimed clauses are real

`section:15.2` still binds `internal/axerror/registry.go:ExitCodeFor` — byte-identical to
what leaf 1 left (verified against `ebc4e31`). Coverage stays `unmeasured` with an honest
updated gap. The easy claim was available and was refused.

I verified all 8 newly claimed §14.2 clauses **against the pinned document itself**, not
against the ownership file's own assertions. Each excerpt appears verbatim at its declared
`SPEC.md` line, `SectionID` puts every one inside 14.2, and each carries an RFC 2119
keyword:

```
14.2#1 line 10473 "All user commands MUST support:"
14.2#2 line 10488 "--yes; commands without such a confirmation MUST reject it."
14.2#3 line 10491 "mode, stdout MUST contain exactly one JSON document; logs remain on stderr."
14.2#4 line 10494 "Destructive or split-brain-risk operations MUST prompt in interactive mode and"
14.2#5 line 10496 "non-interactive mode. --yes alone MUST NOT bypass an expected"
14.2#7 line 10554 "RPC protocol endpoint, not a CLI Result producer, and MUST reject"
14.2#8 line 10605 "For a task-board takeover, adopted MUST be true before"
14.2#9 line 10606 "resumed can be true; for a direct takeover it MUST be false. The"
```

**Wrong-actor check (leaf 1's finding):** every one of the eight governs the **ax CLI
host** — the flag set it must support, the flags it must reject, the stdout discipline it
must keep, the confirmation it must demand, and the adoption rule its own takeover result
must satisfy. None binds the provider plugin or any actor this repository does not build.
The shape does not repeat. Every acceptance case named by every clause is declared in the
file's `acceptance_cases` list.

The one refused clause is the right one. 14.2#6 requires the *process exit status* to
equal the failure's `exit_code`; no library can terminate a process, this repository builds
no `ax` binary (I confirmed `internal/cliresult` has **zero importers** outside its own
package, and the only `os.Exit` calls in the tree are the `exit(1)` paths of `cataloggen`
and `tracecheck`), so it stays undischarged. The line drawn between #6 and #3 is coherent:
#3 is a property of the writer and is fully enforceable over a caller-supplied stream,
while #6 needs process termination. §14.2 is declared `partial` at 8/9 and is consequently
still refused by assigned-scope admission.

The ownership file is hash-gated by `reviewedOwnershipCanonicalSHA256`, which the producer
updated in the same commit. That gate exists to force exactly the review I performed
above; the claims are now independently checked against the pinned text, so the updated
hash is earned rather than self-minted.

### 2.6 Producer evidence reproduced, not accepted on report

| Claim | Reproduced |
| --- | --- |
| `go build ./...`, `go vet ./...`, `gofmt -l .` | 0 / 0 / no output |
| `go test ./... -count=1` | 13 of 13 packages ok |
| tracecheck repository gate | exit 0, `bindings=49 ... clauses_discharged=17/403` — exact |
| tracecheck `-section 14.2` | exit 1 with its measured 8/9 partial ratio — expected red |
| mutation harness, 50 mutants | 48 killed, 2 subsumed, **0 broken**, 62s |
| `internal/cliresult` coverage | 95.3% |
| `internal/axerror` coverage | 99.7% |
| 44 tags / 18 built, 4 versions / 2 built, 31 surfaces / 29 user | all measured true |

The denominator moving 394→403 is honest: §14.2 had no scoped owner before, so binding it
brought its 9 clauses into the inventory; 8 were discharged (9→17).

The harness is itself sound: it requires a unique anchor, asserts the removed text is gone
**and the written text present**, requires the mutant to compile, and restores in a
`finally`; broken or uncompilable mutants fail the run. Both subsumption claims are pinned
by tests of the invariant that makes them subsumed —
`TestCapabilityCountBoundIsSubsumedByTheVocabulary` reddens if the vocabulary outgrows the
bound, and the decode-nullable guard's earlier refusal is one my own probe confirmed.
Several mutants **narrow** rather than delete (`array-lower-bound` drops only the lower
bound, `string-upper-bound` widens by one), so this is not delete-only evidence.

## 3. Finding recorded for leaf 3 (compatibility) — not a defect

`Decode` runs `decodeClosedDocument` (which refuses an unknown top-level member) *before*
`verifyEnvelopeIdentity` (which refuses an unsupported major). Measured:

```
future-major (2.0.0) + a new top-level member -> "unknown top-level member \"new_in_v2\""
                                                 errors.Is(ErrUnsupportedMajor) = false
future-major alone                            -> ErrUnsupportedMajor  (correct)
```

Both paths **refuse**, so there is no bypass and this is not rework. It matters for leaf 3
because §1.6's fail-closed unknown-member rule is scoped to "a major version 1 object" —
precisely the case where the major should already have been settled — so a future-major
document carrying a new member reports a structural fact instead of the version fact a
compatibility caller needs. §17.2 numbers "rejects an unsupported major" first but states
no explicit precedence, so the current order is defensible; leaf 3 owns the decision.

## 4. Why no repository change was required of *me*

Reviewer runs are read-only. Two environment actions were taken outside the repository and
are disclosed: the Go build cache had grown to **134 GB** against 150 MiB of free disk,
which made the suite fail to link (`no space left on device`) — `go clean -cache` freed
135 GB and is worth flagging as a host-hygiene issue unrelated to this change. All
mutations to production files were made with file backups (never `git checkout`, which in
a story worktree discards unstaged work), and every one was restored: `git status` is
clean and `HEAD^{tree}` is still `4eed7417`, byte-identical to the candidate.

## 5. Verdict

**Accepted.** The scoped deliverable is implemented at the production entry points the
task names, both named residuals are closed at source, the eight new clause claims are
verifiable against the pinned document with the correct actor, the one clause that could
not be honestly claimed was refused along with all of §15.2, every gate I attacked held,
and every stated number is a measured ratio that reproduces. The single accuracy finding
(§2.4) is a comment describing its own test's mechanism incorrectly while the gate itself
is correct and closed — recorded for follow-up, not grounds for rework.

Non-final leaf: to be checkpointed, not integrated. Branch is one signed commit past
`ebc4e31`; worktree clean.
