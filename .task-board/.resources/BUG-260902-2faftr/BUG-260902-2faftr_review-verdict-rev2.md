# BUG-260902-2faftr — review verdict: ACCEPTED (revision 2)

Reviewer run `RUN-260902-*` on `task-board/story/STORY-260902-2oiugz`. Change
Request `CR-BUG-260902-2faftr-2` revision 2.

Reviewed delta: base `67aed0bf45275c024c6d3ae683be69bab778f83d` -> candidate tree
`56edf2ec6b5efc04576ff2baa3cb23fdb65850ac`, verified equal to
`git rev-parse HEAD^{tree}` in the workspace (HEAD `29d8cee`). Working tree
clean before and after review; nothing in the reviewed worktree was modified —
every mutant ran on a `git archive` copy under `/tmp/mutant`.
`git rev-list --count HEAD..origin/main` = 0, so this measurement is at trunk,
not behind it (`origin/main` = `67aed0b`).

5 files, +438 / -21: `LOGBOOK.md`, `README.md`,
`internal/config/extension_key_admission_test.go`,
`internal/config/refusal_test.go`, `internal/config/validation.go`.

## 1. The removed rule is invented — re-verified against the pinned bytes

I did not take the previous review's spec verification on trust. I fetched
`SPEC.md` at tag `v0.5.0` from `relux-works/agent-session-manager-spec` and
hashed it:

    562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a

which is exactly the digest pinned in `internal/specpin/pin.go:30`,
`internal/catalog/catalog_gen.go:12`, `internal/traceability/ownership.v0.5.0.json`
and `.spec/README.md`. Every citation resolves verbatim at the stated line:

| Cited | Verified at | Text |
| --- | --- | --- |
| SPEC.md:345-347 | 345 | "A reverse-DNS key is 3–253 lowercase ASCII characters, contains at least one dot, and has dot-separated labels matching `[a-z][a-z0-9-]{0,62}`" |
| SPEC.md:347-349 | 347 | "ExtensionValue is JSON null, boolean, a common-model integer, string, array, or string-keyed object with maximum nesting depth 4" |
| SPEC.md:2344-2345 | 2344 | "Secret values MUST NOT be accepted in config fields; a provider MAY name a machine-local environment variable or credential profile." |
| SPEC.md:2562-2563 | 2562 | "No v2 table accepts a secret, endpoint credential, model token, auth root, or arbitrary environment passthrough." |
| SPEC.md:2596-2597 | 2596 | "An arbitrary blob, raw command/argv, secret, token, endpoint credential, unrestricted environment, or environment passthrough is forbidden." |

`grep -n -i "reverse-DNS"` over the pinned document returns 30 hits; every one
that constrains an `extensions` key states the grammar and nothing else
("Reverse-DNS extension keys only", "Reverse-DNS keys only"). No line anywhere
in the pinned document reserves or forbids a label. The nearest clause that
could have justified a name rule — SPEC.md:350-351, "An extension MUST NOT
shadow, weaken, or be required to interpret a core ownership, fencing,
path-safety, secret-exclusion, or transaction fact" — is a semantic rule about
what an extension may *mean*; a key blacklist neither enforces it nor is
implied by it. SPEC.md:2344 is decisive in the other direction: the
specification expressly permits a config field to *name* a credential profile,
which is what the removed rule refused.

## 2. Both admission points, and no widened gate elsewhere

`grep -rn hasForbiddenConfigName --include=*.go` at the candidate returns zero
hits: the helper and both call sites are gone, not narrowed or renamed. The
helper had no third caller, so nothing unrelated lost a gate.

The schema declares exactly three `extensions` maps (`schema.go:401,409,419`)
and `validateExtensions` is called from exactly three sites
(`validation.go:603,623,646`), so the rule is shared, not per-surface — there is
no context profile that carries the clause while another omits it.

`validateExtensions` still enforces `len(extensions) > 64`, `len(key) > 253`,
`reverseDNSPattern`, the depth-4 bound on both the array and map arms, the
float / oversize-integer value rules, and the 65,536-byte canonical bound. The
delta removes name checks and nothing else.

The two removed `refusal_test.go` cases are precisely the two that pinned the
invented rule; they were deleted, not inverted and not left asserting it.

## 3. Gates attacked, not read — eight single-clause mutants, re-run by me

Each mutant was applied to a `git archive HEAD` copy, run with
`go test ./internal/config -count=1`, and reverted; the copy was confirmed green
between runs. Counts are `--- FAIL` subtest lines.

| Mutant | Direction | Exit | Subtests reddened | Named cases |
| --- | --- | ---: | ---: | --- |
| Re-add key blacklist + helper | narrowing | 1 | 18 | 17 of 20 `…GrammarAlone/admits_*`, plus `…PreservedAsData/preserves_a_whole_extension_key…` |
| Re-add nested-key blacklist | narrowing | 1 | 21 | all 11 `…PreservedAsData/*` + `…AdmittedAsData/*` |
| Drop `reverseDNSPattern` gate | widening | 1 | 11 | `…RefusalStillEnforcesTheReverseDNSGrammar/*`, `…CompositeRefusalClauses/extension_reverse-DNS_namespace`, `…BoundsAtProductionEntry/extension_namespace_grammar…` |
| Widen 253 -> 254 byte bound | widening | 1 | 2 | `refuses_a_key_longer_than_253_bytes`, `extension_key_byte_bound` |
| Widen map depth 4 -> 5 | widening | 1 | 2 | `still_refuses_object_nesting_past_depth_4`, `extension_entry_depth_and_canonical_byte_bounds` |
| Silently `delete` nested `endpoint`/`token` from the loaded map | preservation | 1 | 5 | `…PreservedAsData/preserves_nested_endpoint`, `…nested_token`, `…at_every_admitted_depth`, `…non-string_nested_values`, `…a_whole_extension_key…` |
| Drop `DisallowUnknownFields` | widening | 1 | 5 | closed-table-shape cases incl. `SPEC.md:2562-2563_stays_enforced…` |
| Drop registered backend-settings schema call | widening | 1 | 2 | incl. `SPEC.md:2596-2597_stays_enforced…` |

The two narrowing mutants the acceptance criteria require both redden, and the
deletion mutant confirms the previous review's non-blocking finding is genuinely
closed: the round-trip assertions observe the live decoded map, not a clone, so
a key that is admitted and then dropped fails.

Every mutant count claimed in the shipped `LOGBOOK.md` STATUS bullet that I
sampled independently (key blacklist 18, nested-key blacklist 21,
`DisallowUnknownFields` 5, backend-settings schema 2) reproduced exactly. The
producer's reported numbers are honest.

## 4. Production entry, not a helper

`loadConfigDocument` (`schema_test.go:622`) is a fixture wrapper that builds a
fake filesystem and calls the exported `Load(inputs, nil)`
(`loader.go:215`), which stats, reads and `Decode`s the selected file. The
admission, refusal and round-trip cases therefore drive the real production
entry — verified by reading the wrapper, not inferred from its name. The
round-trip assertions read the value back out of `Snapshot.Configuration()` and
compare re-encoded JSON byte-for-byte, so admission and preservation are pinned
as separate properties.

## 5. Repository gates, run in this workspace

| Gate | Result |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l ./internal` | empty |
| `go test ./... -count=1` | 0, all 9 packages ok |
| `go test ./... -cover -count=1` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 — contracts=60 sections=36 acceptance_cases=35 fixtures=30 compatibility=55 |
| scoped `tracecheck -section 6.1..6.5 17.1 17.2 17.4` | 0 — assigned_scopes=8 |
| `cataloggen -check` | 0, working tree still clean afterwards |

Coverage: `internal/config` 94.4%, against a 93.9% baseline I measured myself
from a `git archive` copy of base `67aed0b`. No regression; the preservation
tests raise it.

## 6. Acceptance criteria

- Invented rule and both call sites removed; admission decides by grammar and
  the depth bound alone — verified by grep and by the widening mutants.
- Every previously blacklisted label admitted and round-tripping through the
  production `Load` entry — 20 admission cases plus 11 byte-identity cases.
- Pinning tests removed with the rule, not left asserting it — verified in the
  `refusal_test.go` delta.
- Re-added blacklist reddens the suite — both arms, 18 and 21 subtests.
- Gates, vet, build, tracecheck, catalog check green with no coverage
  regression.

## 7. Finding recorded, non-blocking

`LOGBOOK.md`, PROVENANCE bullet: "The code and test deltas are byte-identical to
the accepted patch." That was true at revision 1. It is false at revision 2:
`internal/config/extension_key_admission_test.go` differs from the attached
`BUG-260902-2faftr_accepted-work.patch` version by 82 diff lines — the
`requireExtensionsRoundTripTo` helper and the whole
`TestExtensionValueObjectKeysArePreservedAsData` test, added in this revision
and described correctly two paragraphs above the same bullet.

The stale sentence over-attributes the lineage of the preservation test — the
most load-bearing new evidence in the change — to an earlier independent
acceptance it was never part of. It is documentation-only: no code, gate,
evidence or measurement claim is affected, every measurable claim in the same
entry reproduced exactly, and the correct account sits adjacent in the same
entry. It does not meet the bar for rework, so it is recorded here rather than
routed. Recommended correction, one sentence:

    The code delta is byte-identical to the accepted patch; the test delta adds
    the preservation test this revision's review finding required.

The rest of the provenance bullet is accurate: `parseSSHConfigOption` is indeed
absent from base `67aed0b` (`git show 67aed0bf:internal/config/validation.go |
grep -c parseSSHConfigOption` = 0), so the reapply did require relocating the
helper-removal hunk by hand.

## Verdict

ACCEPTED. Recorded with `accept_cr(BUG-260902-2faftr, revision=2)`. The element
parks at `to-review` for the orchestrator to checkpoint or integrate; this
reviewer supplies no `commit_ack`.
