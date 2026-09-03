# TASK-260830-34elja — reviewer verdict, round 2 (CR-TASK-260830-34elja-2 rev2)

Run `RUN-260903-3b8a74`. Verdict: **ACCEPTED**.

## Which commit was reviewed, and why `repository_delta=empty`

HEAD `ebc4e3197ce0f3334450fe0344a21e4542d4a457`, branch
`task-board/story/STORY-260830-2jylym`, `git verify-commit` → `Good "git"
signature for oparin@me.com`, worktree clean, one commit past checkpoint
`d1d3ece`.

The CR reports `repository_delta=empty` because its `base_oid` **is** the leaf
commit and its `candidate_tree_oid` `49e6cb0d` equals both that commit's tree
and HEAD's tree. Verified directly:

```
git rev-parse HEAD^{tree}                = 49e6cb0d375dcb885832d64d9f2723853c7a38d4
git rev-parse ebc4e319^{tree}            = 49e6cb0d375dcb885832d64d9f2723853c7a38d4
candidate_tree_oid (CR rev2)             = 49e6cb0d375dcb885832d64d9f2723853c7a38d4
```

So the reviewable window is a zero-width window over already-committed work —
the same snapshot-artifact class the rev1 reviewer diagnosed, one revision
later and for a different reason (rev1: the checkpoint advanced onto the leaf;
rev2: `handoff` exited 0 after an amend without building a new CR revision).
**Accepting an empty delta here is not accepting an empty delivery.** The
delivery is the commit: `git diff d1d3ece ebc4e319` is 20 files, +3950/-37,
and the rework alone (`git diff 4c9efc0 ebc4e319`) is 10 files, +317/-33. Both
were reviewed as the real change. No repository change would have been the
wrong outcome for this leaf, and none was made — the code is there, in a
commit that the CR machinery failed to snapshot.

The tooling condition is real and recurring. It is an orchestration defect, not
a producer defect, and the producer disclosed it rather than leaving it to be
discovered. Flagged to the orchestrator below.

## Round 1's two blocking findings

**F1 — validated details were aliased.** Fixed and independently reproduced.
`cloneDetails` now recurses through `cloneDetailValue`; `Detail` deep-copies on
the way out. I wrote my own probe from scratch (not the producer's) against a
graph with map-in-map, map-in-slice, slice-in-slice and slice-in-map at depths
1–3 plus every admitted scalar kind, and attacked **both** directions the round-2
brief asked about:

| Attack | Result |
| --- | --- |
| inbound: mutate the caller's retained top-level map | bytes unchanged |
| inbound: retained nested map at depth 3 | bytes unchanged |
| inbound: retained slice element at depth 2 | bytes unchanged |
| inbound: map inside a slice, depth 3 | bytes unchanged |
| inbound: map at slice index 0 | bytes unchanged |
| inbound: slice-inside-slice element | bytes unchanged |
| inbound: deepest map inside a nested slice | bytes unchanged |
| outbound: mutate every container `Detail` returns, at four positions | bytes unchanged |
| outbound: replace a slice member returned by `Detail` | bytes unchanged |
| outbound: same on a `Decode`d object | bytes unchanged |
| `DetailKeys` mutated by the caller | fresh slice each call |
| `Decode` of the package's own output | byte-identical round trip |

**The copy is complete for the admitted value set.** `validateDetailValue`
admits only `nil`, `bool`, `string`, `json.Number`, `[]any`, `map[string]any`;
the four scalar forms are immutable Go values and the two container forms are
both recursed. I attacked the one gap a Go type switch leaves — a **named** map
type whose underlying type is `map[string]any`, which would miss `case
map[string]any` and fall through `cloneDetailValue`'s `default` arm as a live
alias. `axerror.Details` itself is such a type, and it is refused upstream:
`details["context"] uses unsupported JSON type axerror.Details`. No mutable
value can reach the default arm.

**F2 — clause 15.1#6 claimed against the provider plugin.** Fixed. Clause
dropped, the §15.1 gap now names #5 and #6 side by side on the same reasoning,
digest recomputed to `b6254945`. §15.1 is 5/7, `clauses_discharged` 9/394, and
the ratio is consistent in all four places it is published: `main_test.go:26`,
`main_test.go:59`, `traceability_test.go:626`/`:628`, and `README.md:1467`/
`:1474`. I re-read SPEC.md:11047 myself: its only RFC 2119 sentence is *"The
plugin MUST NOT masquerade a different-major object as v2"*, the host half of
that table row carries no keyword, and the drop is correct.

## The round-2 brief's four attack items

**1. Is the copy complete, in both directions?** Yes — table above.

**2. Is the copy paid for once or per access?** **Per access.** Two successive
`Detail("context")` calls return distinct allocations; a caller reading a large
nested detail in a loop pays a full deep copy each time. This is a deliberate
correctness/cost trade and it is the right one here (details are bounded at
16 KiB and depth 4), but it is a surprise worth stating, and `Detail`'s doc
comment says "deep copy" without saying "every call". Non-blocking.

**3. A third wrong-actor clause?** **No.** I read every claimed line and named
the actor its RFC 2119 keyword governs:

| Clause | SPEC line | Actor the keyword binds | Implemented here |
| --- | --- | --- | --- |
| 15.1#1 | 11002 | emitter ("IDs … MUST be present when known") | yes — `IDs`/`New` |
| 15.1#2 | 11003 | emitter ("Details MUST be redacted and schema-valid") | yes — `ValidateDetails` |
| 15.1#3 | 11010 | readers ("MUST never infer success, authority, or a remediation action") | yes — `Decode`/`Detail` |
| 15.1#4 | 11019 | emitter ("MUST use that exact object") | yes — `BindingFor` |
| 15.1#7 | 11053 | receivers ("MUST NOT parse a different major's payload") | yes — `Decode` |
| 15.3#1 | 11121 | reader ("MUST NOT be interpreted as success") | yes — `Decode` |
| 15.3#2 | 11133 | implementations ("MUST NOT mint a realm-specific code") | yes — closed registry |

F2's shape was: the line's *only* keyword binds an actor this repository does
not build. None of the seven survivors has it. 15.1#1 and 15.1#3 carry a second
keyword aimed at "automation"/"readers" as consumers, but each line also carries
an obligation on an actor this repository does implement, and that is what the
acceptance cases discharge.

**4. N1/N2/N3.** All three addressed, each verified rather than accepted:
- **N1** — the redundant leading-quote guard is gone; the wrong-type refusal
  table went from 4 rows to 9 (`9.5 "9" "" 1e1 4294967296 true [] {} [9]`), and
  a quote-stripping mutant (`bytes.Trim(..., "\"")`) compiles and is **killed**
  by `TestReaderAndAccessorEdges`. The commit no longer presents the guard as
  the fix.
- **N2** — `run_mutant` now takes `present|absent` and verifies a deletion by
  the absence of the deleted text.
- **N3** — the `target_auth_missing` / `capability_unavailable` asymmetry is
  explained in `New`'s doc comment, correctly, as "requiring them everywhere
  would invent a constraint §15.3 does not state".

**5. Nothing from round 1 regressed.** The rework touches 10 files; `binding.go`,
`registry.go`, `local.go`, `details.go` and every §15.2/§15.3 table are
untouched. `decode_test.go`'s only change **widens** a refusal table. The one
defense the rework *removed* — `cloneDetails` in `decodeBody` — I attacked: the
map is allocated by `encoding/json` from the document bytes inside
`decodeClosedDocument`, no caller can hold a reference to it, `json.RawMessage`
copies rather than aliasing the input buffer, and the outbound arm is still
covered by `TestDecodedObjectsOwnTheirDetailsToo`. Removing an unkillable line
was right.

## Attacked, not read

**My own mutation suite: 41 mutations applied and compiled, 39 killed, 2
survivors.** Written independently of the producer's; every mutant verified
present in the file (or, for a deletion, verified absent) before its measurement
was believed, and a mutant that fails to compile is reported as no-build rather
than as a kill. The raw arithmetic, so it can be checked against the logs: batch
one ran 34, of which 5 failed to apply (bad regex on my side) and 1 (`M29`)
broke compilation and was **discarded rather than scored as a kill** — the
producer's harness would have scored it KILLED. Batch two repaired those six and
added 7 more; 2 further mutants were run by hand. The 2 survivors are the same
`minRedactableCause` bound at two widths (R1 below). Logs:
`TASK-260830-34elja_review-mutants-rev2-01.log`, `-02.log`.

Preferring narrowings over deletions, as the standing bar requires:
- deep copy narrowed to one level, to maps only, to arrays only; accessor
  returns the live value; `New` stores the caller's map — **all killed**
- depth 4→5, canonical 16 KiB→64 KiB, keys 64→128, key grammar `{0,63}`→`{0,127}`,
  message 4096→8192 and 1→0 — **all killed**
- one credential key removed from the scanner; nested excluded-key walk removed
  — **killed**
- causal scan restricted to the outermost link; restricted to skip detail
  values; `valueContains` blinded to maps — **all killed**
- major gate widened to admit major 2; version match reduced to majors only;
  `DisallowUnknownFields` removed; trailing content admitted; decoded exit/code
  contradiction admitted; success status 0 admitted as a failure class —
  **all killed**
- retryability: exit-7 class, exit-16 class and `operation_uncertain` each
  removed separately; refusal skipped in `New`; skipped in `Decode` —
  **all killed**
- per-version code admission removed; unregistered version admitted; typed-detail
  non-empty check narrowed to presence-only; bootstrap version pin drifted;
  unknown outcome silently mapped to the fallback code — **all killed**

**The producer's own 26-mutant harness, re-run by me from the attached resource
on a clean tree: 26 KILLED, 0 SURVIVED, 0 MUTANT-ABSENT**, final restore
verified. I spot-checked two of its mutants by hand and confirmed they compile,
so those kills are test failures and not build failures.

**Causal redaction.** I could not get an unredacted cause onto the wire. `cause`
is unexported, `MarshalJSON` builds a closed `wireError` that has no field for
it, `causeChainText` walks both `Unwrap() error` and `Unwrap() []error` arms,
and both the message and every nested detail value are scanned.

**The digest pin is recomputed, not declared.** A semantically inert edit to a
gap string (one added full stop) reddens the gate: `digest be328132… differs
from reviewed b6254945…`. Coverage cannot be self-minted.

**The gate checks the claim, not only the pin.** I pointed clause 15.1#1 at line
11004 instead of 11002 and blanked the digest: the gate refused with `clause
"15.1#1" declares line 11004, but the pinned clause is at line 11002` — the
line/excerpt binding is enforced independently of the digest, so a claim cannot
be relocated to a friendlier sentence even by someone who recomputes the pin.

**§15.2 table verbatim.** I re-derived the table from SPEC.md:11077–11094 with a
parser and compared every row against `exitMeanings`: **all rows match exactly**.

**§17.2 was not re-bound** to a friendlier symbol. Its binding still names
`internal/config/writer.go:EncodeCurrent` with a gap stating plainly that it
discharges nothing. That was the easy way to look green and it was refused
again. **§14.2 is correctly left unclaimed**: its operative sentence for this
leaf is "The process exit status MUST equal that error's `exit_code`", which is
leaf 2's scope, and no catalog row requires a §14.2 binding, so the gate is not
being dodged.

## Evidence reproduced on the restored tree

| Check | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l .` | no files |
| `go test ./... -count=1` | exit 0, 12 packages |
| `go test ./internal/axerror -count=1 -cover` | exit 0, **99.7%** of statements |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0, `clauses_discharged=9/394` |

The 99.7% figure is **measured by me**, not accepted from the producer's log.
Log: `TASK-260830-34elja_review-final-validation-rev2-01.log`.

Working tree left byte-identical to `HEAD` (`git status --short` empty). All
mutations were applied against a private byte backup and restored from it, never
with `git checkout`.

## Non-blocking findings

**R1 — the causal-redaction sensitivity bound is not defended by any test.**
`minRedactableCause = 8` (details.go:39) decides which causes get scanned at all.
Raising it to **16** leaves the suite green; raising it to **64** leaves the suite
green; only 4096 reddens. Every cause in the test corpus is ≥64 characters, so
the constant's actual value is unpinned: an edit widening it to 63 silently stops
scanning a whole class of short causes — exactly the "a bound proved only by
deleting the gate" shape this board refuses. It is a *sensitivity* parameter of a
redaction gate, and the gate's own `RedactionBound` string does not state it.
Suggested: one case with a cause of exactly `minRedactableCause` runes that must
be refused, and one just under it that must be admitted.

**R2 — a new prose count that does not match the measured table.** `README.md:1248`
("the closed **nineteen-row** Section 15.2 table") and `registry.go:64`
("nineteen rows, closed") are both introduced by this leaf. The pinned table at
SPEC.md:11077–11094 has **18** rows (0, 2–17, 130). The implementation and its
reviewed literal in `registry_test.go` are both correct at 18 and are compared by
length and by content, so nothing is broken — only the sentence is wrong. It
repeats a miscount already present board-wide (`README.md:1426`,
`traceability.go:155`, and the BUG-260902-2m7slg artifacts), which is why it is
worth correcting at source rather than only here. This is the board's own rule
applied to itself: a count stated as prose drifts from the count that was
measured.

**R3 — bound, not a finding.** `registry_test.go`'s `pinned` exit table is a
hand-transcribed literal; nothing re-derives it from SPEC.md, so a document repin
would not redden it. `specpin` pins the document digest, so the document cannot
change silently, and I verified all 18 rows verbatim against SPEC.md this round.

**R4 — the producer's harness cannot distinguish a non-compiling mutant from a
killed one.** `run_mutant` runs `go test` directly with no `go build` gate, so a
mutation that breaks compilation reports KILLED. No false kill occurred — I
re-ran all 26 and hand-checked the ambiguous ones — but my harness added the
build gate and it caught one of my own mutants that would otherwise have scored
a false kill.

## Stated bounds of this review

- `internal/axerror` still has **no production caller anywhere in the
  repository**: no `.go` file outside the package imports it. Every behavior
  here is proven at the package boundary and none end-to-end through a command.
  Expected for this leaf — leaf 2 owns the CLI envelope and exit-code wiring —
  and disclosed rather than inferred away.
- The clause scanner counts RFC 2119 lines and cannot decide **which actor** a
  MUST binds. That is how 15.1#6 passed on rev1, nothing in this rework closes
  the class, and my actor table above is a human read, not a machine check.
- No fuzz targets exist in `internal/axerror`; none were run.
- I did not re-verify the 15-row containing-contract binding table row by row
  this round. `binding.go` is untouched by the rework and round 1 verified it
  verbatim; that is an accepted round-1 result, not a round-2 measurement.

## For the orchestrator

Accepted work is parked at `to-review` by `accept_cr`. Two things to carry:

1. **The CR snapshot window has now failed twice on this leaf, for two different
   reasons.** rev1: the checkpoint advanced onto the leaf commit before the
   candidate was snapshotted. rev2: `handoff` exited 0 after a producer amend
   and built no new revision, so `CR-…-2` points at a tree identical to its own
   base. Both times the reviewable delta had to be reconstructed by hand from
   the commit. This is a tooling defect worth a board item of its own.
2. R1 and R2 are cheap and belong to this package. R2 in particular should be
   fixed at its source, since the "nineteen-row" miscount is repeated in
   pre-existing repository prose.
