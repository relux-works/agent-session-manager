# TASK-260830-34elja — Review Verdict: CHANGES REQUESTED

Reviewer run `RUN-260903-737126`. Change Request `CR-TASK-260830-34elja-1` revision 1.
Reviewed commit `4c9efc09bded14e5dcdf96ae0191fb2c15e57cf7` (worktree HEAD, branch
`task-board/story/STORY-260830-2jylym`), 19 files, +3666/-37.

## On `repository_delta=empty`

The Change Request snapshot reports zero changed paths because its base OID **is**
the producer's own commit: `git rev-parse 4c9efc0^{tree}` and `git rev-parse
HEAD^{tree}` are both `42868f86e9823b46b5b390efe1a5e60c0cca2f2f`. The checkpoint
advanced onto the leaf commit before the candidate was snapshotted, so the
reviewable window closed over already-committed work rather than over an empty
producer run. This is a snapshot artifact, not an empty delivery: the leaf
delivered a 1449-line production package and a 1810-line test suite. I reviewed
the commit contents directly, as the orchestrator's reviewer brief directed.

Emptiness is therefore not the reason for this verdict. The reasons are below.

## Verdict

`changes_requested` → `to-dev`. Two confirmed defects, both reproduced.

---

## F1 (blocking) — Validated details are aliased: the writer emits an object its own reader refuses

`cloneDetails` (`internal/axerror/axerror.go:321`) is a **shallow** copy, and
`Detail` (`internal/axerror/axerror.go:249`) hands the caller the live nested
container. `ValidateDetails` therefore checks a graph the constructed `*Error`
does not own, and every Section 15.1 detail bound can be violated *after*
construction through the package's own exported surface.

Reproduced (probe: `.temp/TASK-260830-34elja/probe/main.go`):

```go
failure, _ := axerror.New(axerror.Spec{
    Version: axerror.Version100, Code: "provider_protocol_error",
    Message: "unusable first frame", IDs: axerror.NoIDs(),
    Details: axerror.Details{"context": map[string]any{"stream": "stdout"}},
})
v, _ := failure.Detail("context")          // the package's own accessor
m := v.(map[string]any)
m["password"] = "hunter2"                  // a class Section 15.1 forbids outright
```

`json.Marshal(failure)` then emits:

```json
{"schema":"urn:ax:schema:error","schema_version":"1.0.0","code":"provider_protocol_error",
 "message":"unusable first frame","exit_code":13,"retryable":false,
 "details":{"context":{"password":"hunter2","stream":"stdout"}}}
```

The same accessor also defeats every other declared bound. In one probe run
`MarshalJSON` emitted **33,036 bytes** (declared bound 16 KiB), carried a
value nested **7** containers deep (declared bound 4), and carried
`"count":7` — a Go `int`, a type `validateDetailValue` refuses outright.
`axerror.Decode` then rejected the package's own output:

```
Decode of our own output: invalid structured error details: details["context"] exceeds maximum nesting depth 4
```

A caller that retains the nested map it passed to `New` gets the same result
without touching the accessor at all. Only the **top-level** map is protected;
`details["secret"] = ...` on the caller's own map correctly did not appear.

Why this is blocking, not a style note:

- It is a **bypass path around the check** — one of the named negative shapes.
  `ValidateDetails` is the production entry point the ownership registry names
  for acceptance case `structured-error-detail-redaction`, which discharges
  clause **15.1#2** ("Details MUST be redacted and schema-valid"); the same
  aliasing defeats the 16 KiB / depth-4 / key-grammar bounds that clause
  **15.1#3** claims. Both clauses are discharged by tests that only ever call
  `ValidateDetails` on a map nobody mutates afterwards. **No test covers
  post-construction mutation**, so nothing fails when the refusal stops
  happening.
- It contradicts an exported doc comment, which this board treats as a
  contract. `Error`'s own comment says "Every field is unexported and the only
  encoder is `MarshalJSON`, so the closed Section 15.1 object is the only shape
  this type can produce"; `RedactionBound` enumerates the gate's limits and says
  nothing about a validated object being mutable. The limitation is real and
  undisclosed.
- It does not match the project's own convention. `internal/config` already
  returns "one immutable resolved-root snapshot" for exactly this reason.
- This is the foundation leaf four m0 Stories bind to. Leaf 2 will marshal
  these objects into CLI Result envelopes; a validated-then-mutable object is
  inherited by everything downstream.

Fix shape: deep-copy on the way in and on the way out (or expose details as an
immutable projection). Both arms need a test — construct, mutate through the
retained alias and through `Detail`, and assert the encoded bytes are unchanged
and still decode.

---

## F2 (blocking) — Clause 15.1#6 is claimed against an obligation that binds the provider plugin, not this repository

`internal/traceability/ownership.v0.5.0.json` claims clause `15.1#6` at
SPEC.md:11047, discharged by acceptance case `structured-error-local-untrusted`
(`internal/axerror/local.go:LocalFromUntrusted`).

The gate counts a clause as one line matching
`\b(MUST NOT|MUST|SHALL NOT|SHALL|REQUIRED)\b` (`traceability.go:970`). Line
11047's **only** RFC 2119 keyword is:

> The plugin MUST NOT masquerade a different-major object as v2.

That obligation binds the **provider plugin** — the child process. This
repository builds no provider plugin; its only binaries are `cataloggen` and
`tracecheck`. `LocalFromUntrusted` is the **host** emitter, and the host half of
that table row ("The host accepts no child error object, terminates/waits for
exit as applicable, and emits its own local Error 1.0.0…") carries no RFC 2119
keyword at all, so it is not what the gate measured.

This is the same ground on which clause `15.1#5` was — correctly and honestly —
declared undischarged: "This repository contains no RPC hello builder at all."
It contains no provider plugin either. The 15.1 `gap` text discloses only #5.

Effect on the published numbers:

| Claim | Published | Honest | Level |
| --- | ---: | ---: | --- |
| section 15.1 | 6/7 | 5/7 | `partial` either way |
| clauses_discharged | 10/394 | 9/394 | gate still exits 0 |

No gate outcome changes — 15.1 stays `partial` and assigned-scope admission
still refuses it. But the inflated ratio is now published in `README.md`, in the
`tracecheck` expectation strings in `main_test.go` and `traceability_test.go`,
and inside the reviewed ownership digest pin. This board's standing rule is that
a gate reports coverage as a **measured ratio**; one clause of the eight newly
claimed is measured against an actor this repository does not implement.

Fix shape: drop `15.1#6`, extend the 15.1 `gap` to name it alongside #5 with the
same reasoning, recompute `reviewedOwnershipCanonicalSHA256`, and update the
three expectation strings and the README ratios to 5/7 and 9/394.

---

## Non-blocking notes

- **N1 — a redundant guard with no independent coverage.** Deleting the
  `text[0] == '"'` refusal in `decodeExitStatus` (`decode.go:213`) leaves the
  suite green: `strconv.ParseInt` already refuses `"9"`. Behavior stays correct
  — the real fix for the reported defect was moving `exit_code` to
  `*json.RawMessage`, and `TestReaderAndAccessorEdges` does pin `"9"` as
  refused. Not a hole, but the commit message presents the quote check as the
  fix and no mutant distinguishes it.
- **N2 — several mutant-presence markers assert an unrelated string.** The
  harness checks e.g. `'16:  \`exit 16 is'` after deleting the *exit-7* row, and
  `'func IsFailureExitStatus'` after deleting a guard — neither can distinguish
  "mutation applied" from "perl no-op". It produced no false KILL here (an
  unapplied mutation reports SURVIVED, not KILLED), but it is weaker than the
  board's "confirm every mutant is PRESENT" rule intends.
- **N3 — the typed-detail asymmetry is right but unexplained.** `New` enforces
  `requireDetailKeys` for `target_auth_missing` and not for
  `capability_unavailable`. I judge this **correct**: `capability_unavailable`
  is a general exit-6 code and Section 15.3 names those five typed details only
  for the realm/broker-evidence case, so a blanket requirement would be an
  invented constraint; `target_auth_missing` has exactly one declared use. Worth
  a sentence in the comment so the asymmetry reads as deliberate rather than
  missed.

---

## What I attacked and what held

Independently reproduced, not accepted from the attached logs:

| Check | Result |
| --- | --- |
| `go build ./...`, `go vet ./...`, `gofmt -l internal/` | clean |
| `go test ./... -count=1` | exit 0, 12 packages ok (`review-go-test-all.log`) |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0, `clauses_discharged=10/394` |
| Producer's 20-mutant harness, re-run from the resource | **20/20 KILLED**, final restore green |
| My own 12 narrowing mutants on gates the harness missed | **11/12 killed**; the survivor is N1 |

My twelve mutants and their results — reader-side typed-detail requirement
dropped, quoted `exit_code` admitted (N1, survived), document picks its own bound
version, trailing content admitted, unknown code reported as registered, causal
gate never fires, 16 KiB bound widened, key-count bound widened, depth bound
widened, unpinned bootstrap outcome falling back to `otherwise`, reader code
grammar dropped, `BindingFor` defaulting to 1.0.0 instead of refusing — are in
`.temp/TASK-260830-34elja/review-mutants.sh`.

Verified against the pinned document, byte-exact via `internal/specdoc`:

- All 19 Section 15.2 exit rows reproduced verbatim in `exitMeanings`; no row
  invented, none dropped.
- All 15 binding-table rows trace to a quoted 15.1/15.3/17.1 sentence.
- All eight claimed clause excerpts occupy the exact lines declared
  (11002, 11003, 11010, 11019, 11047, 11053, 11121, 11133).
- **§17.2 was not rebound to a friendlier symbol.** Its single clause is
  "MAY retain an unknown *event* as inert history but MUST NOT derive state from
  it" — a reader obligation about events, which an unknown error *code* does not
  discharge. The producer left the wrong `EncodeCurrent` binding in place and
  disclosed it. That was the easy way to look green and it was correctly refused.
- **§14.2 correctly left unclaimed** — it is the CLI flag/output section and
  belongs to leaf 2.
- **The digest pin is recomputed, not declared.** A semantically inert edit to a
  `gap` string (no other check disturbed) reddens the gate with
  `ownership registry projection digest cb585032… differs from reviewed 5a4f769a…`.
  Coverage claims cannot be self-minted.
- No gate logic was weakened in `internal/traceability`: the only changes are the
  digest pin and expectation strings, confirmed by reading the full diff.
- Clause subjects for `15.1#1/#2/#3/#4/#7` and `15.3#1/#2` align with the actors
  the implementation actually plays (writer, reader, receiver). Only `15.1#6`
  does not.
- Causal redaction: the cause is an unexported field, `MarshalJSON` cannot reach
  it, the whole `Unwrap` chain including `errors.Join` arms is scanned, and I
  could not get an unredacted cause onto the wire through the exported surface.
  Its stated bounds (paraphrase undetected, causes under 8 runes skipped) are
  asserted by a test rather than left to be discovered.
- Retryability is symmetric writer/reader and is a refusal table, never a
  permission; the "admitted" arms prove it does not simply refuse everything.

## Stated bound of this review

`internal/axerror` has **no production caller anywhere in the repository** — no
`.go` file outside the package imports it. That is expected and disclosed for
this leaf (leaf 2 owns exit-code wiring and the CLI envelope), and the ownership
registry names the package's own exported functions as the production entry
points. It does mean every behavior here is proven at the package boundary and
none of it is yet proven end-to-end through a command. I report that as a bound,
not as a finding against this leaf.

I did not re-run the fuzz targets or the `-cover` measurement; I accepted the
99.7% statement-coverage figure from the attached log rather than reproducing it.

---

## Logbook

I did **not** write `LOGBOOK.md`. The reviewer role is read-only on the
repository and this branch must stay exactly one commit past the checkpoint with
a clean worktree; a reviewer commit would break both. Two entries are worth
recording during rework, and the rework producer should add them:

- **Validate-then-alias is a bypass path, not a gate.** A validator that checks a
  graph the constructed value does not *own* has run and then let the checked
  data change. `New` validated a `Details` map and kept it through a shallow
  `cloneDetails`; `Detail` handed the live nested container back, and every
  Section 15.1 bound fell after validation through the package's own exported
  accessor. `internal/config` already returns an isolated snapshot — that is the
  convention this package should have followed.
- **Claim a clause against the actor its MUST binds, not against a topically
  related section.** Clause `15.1#6`'s only RFC 2119 sentence binds the provider
  plugin; it was claimed by the host-side emitter, and the host half of the same
  table row carries no RFC 2119 keyword at all. The clause scanner counts lines,
  not intent, so it cannot catch this — a human reading which noun the MUST
  governs is the only check there is.
