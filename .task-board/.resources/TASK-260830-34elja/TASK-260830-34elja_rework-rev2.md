# TASK-260830-34elja — Rework of CR-TASK-260830-34elja-1 rev1

Reviewer verdict `changes_requested` (`TASK-260830-34elja_review-verdict.md`),
two blocking findings, both reproduced by the reviewer and both reproduced again
here before being fixed. Commit `4c9efc0` amended in place to
`ebc4e3197ce0f3334450fe0344a21e4542d4a457`, so the branch stays exactly one
commit past checkpoint `d1d3eceb2c42e7a094206465d128272bc81008f5`.

## F1 (blocking) — validated details were aliased

`cloneDetails` was a shallow copy and `Detail` returned the live nested
container, so `ValidateDetails` checked a graph the constructed `*Error` did not
own. Every Section 15.1 detail bound could be violated after construction
through the package's own exported accessor.

**Fix.** `cloneDetails` now calls a new `cloneDetailValue` that deep-copies the
two container kinds a detail value may open (`map[string]any`, `[]any`) and
returns every other admitted value unchanged — safe rather than partial, because
`validateDetailValue` admits only nil, bool, string, `json.Number` and those two
containers, and the four scalar forms are immutable Go values. `Detail` returns
`cloneDetailValue(value)` rather than the stored value.

`decodeBody` no longer clones: its map is allocated by the decoder from the
document bytes and no caller holds a reference, so a clone there would have been
an unkillable line. That ownership is stated at the assignment instead.

**Reproduction, before and after.** The reviewer's probe
(`.temp/TASK-260830-34elja/probe/main.go`, extended with a `password` check)
run against the fixed package — `probe-after-fix-01.log`:

```
marshal err: <nil>
encoded bytes: 198
carries password -> false
carries count:7 -> false
carries depth-7 nest -> false
Decode of our own output: <nil>
```

Reviewer's measurement of the same probe on rev1: 33,036 bytes, `"count":7`
present, depth 7, and `Decode` of the package's own output refused with
`details["context"] exceeds maximum nesting depth 4`.

**Tests.** `internal/axerror/ownership_test.go`, three cases:

| Case | Attacks |
| --- | --- |
| `TestConstructionDoesNotAliasTheCallerDetailGraph` | four containers the *caller* retains, at depth 1, depth 2, inside an array and inside a nested array; plus the top-level map and an array-member replacement |
| `TestDetailAccessorDoesNotHandOutTheLiveContainer` | the same graph reached through `Detail`, at two depths and through both array levels, plus a second read to prove isolation is per call |
| `TestDecodedObjectsOwnTheirDetailsToo` | the accessor on an object built by `Decode` from wire bytes |

Each mutation writes all four violations — a forbidden class key, 32 KiB past
the 16 KiB canonical bound, a value 7 containers deep, and a Go `int` —
and then asserts the object's encoded bytes did not move **and** that `Decode`
still accepts them.

The containers are taken at three depths and on both sides of an array
deliberately: that is what makes a deep copy **narrowed to one level** fail
these tests rather than only a copy deleted outright.

## F2 (blocking) — clause 15.1#6 was claimed against the wrong actor

SPEC.md:11047's only RFC 2119 sentence is "The plugin MUST NOT masquerade a
different-major object as v2", verified verbatim against the embedded pinned
document. It binds the provider plugin; this repository's only binaries are
`cataloggen` and `tracecheck`. The host half of the same table row, which
`LocalFromUntrusted` implements, carries no RFC 2119 keyword and is therefore
not what the clause scanner measured.

**Fix.** Clause `15.1#6` dropped from
`internal/traceability/ownership.v0.5.0.json`; the §15.1 `gap` now names both
undischarged clauses with the reason for each; `reviewedOwnershipCanonicalSHA256`
recomputed `5a4f769a…` → `b6254945…`; the three pinned expectation strings and
the README ratios corrected.

| Claim | rev1 | rev2 |
| --- | ---: | ---: |
| section 15.1 | 6/7 `partial` | 5/7 `partial` |
| repository `clauses_discharged` | 10/394 | 9/394 |
| `tracecheck` exit | 0 | 0 |

No gate outcome moves: §15.1 stays `partial` and assigned-scope admission still
refuses it. The digest was **recomputed from the edited registry, not declared**
— the gate reddened on the first `tracecheck` run after the edit and only went
green once the pin was updated, which is the log entry above the fix in
`tracecheck-01.log`'s predecessor run.

## Non-blocking notes

- **N1 — redundant `exit_code` guard.** The `text[0] == '"'` refusal was dead
  weight in front of `strconv.ParseInt`, which already refuses quoted bytes; the
  real fix for the reported defect was reading the member from
  `*json.RawMessage`. The guard is removed and the doc comment now says which
  parse does the work. The refusal table widened from four to nine wrong JSON
  types (`9.5`, `"9"`, `""`, `1e1`, `4294967296`, `true`, `[]`, `{}`, `[9]`),
  and a new mutant strips quotes before parsing — the exact narrowing a
  `json.Number` field would reintroduce — and is killed.
- **N2 — unverifiable mutant markers.** `run_mutant` now takes `present|absent`.
  A deletion is verified by the **absence** of the exact text it deleted, an
  edit by the presence of the exact text it wrote. Nine mutants moved to
  `absent` mode.
- **N3 — typed-detail asymmetry.** `New`'s doc comment now states why
  `target_auth_missing` carries a presence requirement and
  `capability_unavailable` does not.
- **Logbook.** The reviewer's two entries are written, plus the N1 correction of
  the 0742 entry's claim about the leading-quote guard, in `LOGBOOK.md` at 0753.

## Evidence — every command run in this rework, with its real exit code

| Command | Exit | Result | Log |
| --- | ---: | --- | --- |
| `go build ./...` | 0 | clean | `go-build-01.log` |
| `go vet ./...` | 0 | clean | `go-vet-01.log` |
| `gofmt -l .` | 0 | 0 files listed | `gofmt-01.log` |
| `go test ./internal/axerror/ -count=1 -v` | 0 | pass | `go-test-axerror-01.log` |
| `go test ./internal/axerror/ -count=1 -cover` | 0 | 99.7% of statements | `go-test-axerror-cover-01.log` |
| `go test ./... -count=1` | 0 | 12 packages ok | `go-test-all-01.log` |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | `clauses_discharged=9/394` | `tracecheck-01.log` |
| `zsh mutants.sh` | 0 | 26 mutants, **26 KILLED, 0 SURVIVED, 0 MUTANT-ABSENT** | `mutation-01.log` |
| `go run ./.temp/TASK-260830-34elja/probe` | 0 | F1 no longer reproduces | `probe-after-fix-01.log` |

The 99.7% figure is a measured `-cover` run in this rework, not carried over
from rev1. The one uncovered region is `MarshalJSON` at 92.3%: the encode-error
arm, unreachable for a validated object and reported rather than ignored so a
future member type cannot fail silently.

Five of the 26 mutants are new and all target F1:

```
KILLED    details: construction copy shallow again (revision-1 defect)
KILLED    details: deep copy narrowed to one level (nested maps shared)
KILLED    details: deep copy narrowed to maps only (arrays shared)
KILLED    details: accessor hands out the live container again
KILLED    details: array arm of the deep copy deleted
```

Three of those five are **narrowings**, not deletions, which is what proves the
bound rather than the existence of the copy.

## Stated bounds of this rework

- `internal/axerror` still has **no production caller anywhere in the
  repository**; no `.go` file outside the package imports it. Leaf 2 owns the
  exit-code wiring and the CLI envelope. Every behavior here is proven at the
  package boundary and none of it is yet proven end-to-end through a command.
  The ownership registry names the package's own exported functions as the
  production entry points, which is honest for this leaf and is the thing leaf 2
  has to change.
- The clause scanner counts lines carrying an RFC 2119 keyword. It cannot decide
  **which actor** a MUST binds — that is exactly how `15.1#6` passed the gate on
  rev1. No check added here closes that class; a human reading the subject of
  the sentence remains the only one.
- Redaction is unchanged and its bound is unchanged: exact key names and
  verbatim cause containment only. `RedactionBound` states it and a test asserts
  the constant still says it. A secret under an innocuous key is admitted.
- Fuzz targets were not re-run in this rework.
