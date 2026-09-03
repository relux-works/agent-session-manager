# BUG-260903-3nt3ah — Reviewer verdict: ACCEPTED

Change Request `CR-BUG-260903-3nt3ah-3` revision 3. Base `6c690da`, candidate tree
`2b3cefec7a31952105703d0be04934bad0c03e0b`. Verified `git rev-parse HEAD^{tree}` equals the
candidate tree, so the review below was run against the exact reviewable delta, not a later edit.

Changed paths (4): `LOGBOOK.md`, `README.md`, `internal/canonicaljson/canonical.go` (comments
only), `internal/canonicaljson/canonical_number_contract_test.go` (new, 774 lines).

## The decision under review

The AC allowed (a) make `Canonicalize` refuse the literals, or (b) document the RFC 8785
Appendix B rounding as intended with the reason. The producer chose (b). That is the correct
branch and the reason holds: RFC 8785 Section 3.2.2.3 serializes through the ECMAScript
double-to-string algorithm, and refusing here would contradict Appendix B — whose own published
samples `9007199254740992` and `295147905179352830000` are canonical *outputs* of exactly this
rounding, and both are present in the vendored vector table at `canonical_test.go:47-70`. Making
`Canonicalize` refuse would have required deleting or weakening that test. It was not touched.

## Verification performed (not read — attacked)

`canonical_test.go` is byte-for-byte absent from the diff. Confirmed independently: it does not
appear in `git diff --stat 6c690da 2b3cefe`.

**19 mutants, each confirmed present in the source before measuring** (the harness aborts with
`NOOP-MUTANT` if the edit did not apply — one mutant tripped that guard and was rewritten against
the real anchor rather than being scored as killed). 18 killed by the new suite; 1 survivor killed
by the pre-existing package suite.

Doc-only mutants — no code change, must redden anyway (all KILLED):

| Mutant | Killed by |
| --- | --- |
| `1.0 -> 1` retargeted to `1.0 -> 1.0` | DocumentedRoundingIsWhatCanonicalizeActuallyDoes |
| rounding row `18446744073709551615` deleted | same (audited literal set) |
| entry point `ValidateObservationStream` removed | DocumentsExactlyTheEntryPoints |
| fake entry point `CanonicalizeStrict` added | same |
| citation re-pointed NUM-UNSAFE-ROUND → NUM-UNSAFE-NUMBER | SpecificationCitationsQuoteThePinnedSpecification |
| citation weakened to non-unique fragment `"MUST"` | same (uniqueness clause) |
| shallow `400 sibling arrays` row deleted | ContainerLimitIsADepthNotACount (coverage requirement) |
| `256 containers open at once` → `256 containers` | same (per-occurrence prose scan) |
| `non-I-JSON` claim reinstated | DocumentsExactlyTheEntryPoints |
| cited test renamed | DocCommentNamesOnlyTestsThatExist |
| unparsable indented line added | parse gate (documented-but-unchecked cannot accumulate) |

Code mutants — narrowing, not only deleting:

| Mutant | Result |
| --- | --- |
| `maxSafeInteger` widened to 2^54-1 (gate kept, bound moved) | KILLED |
| float clause dropped, integer clause kept (partial gate) | KILLED |
| recursion into `map[string]any` removed (extensions carrier escapes) | KILLED |
| `validateAXNumbers` call deleted from `prepareObjectIdentity` | KILLED |
| `Canonicalize` made to call `validateAXNumbers` | KILLED |
| `maxNestingDepth` 256 → 257 | KILLED |
| `validateAXNumbers` registered in a function-value dispatch table | KILLED — builds clean, behaviour unchanged, and `TestValidateAXNumbersHasNoCallSiteThisGraphCannotSee` fires with `validateAXNumbers appears 2 times outside plain calls`. This is the guard that makes the derived entry-point set exact rather than merely sound, and it holds. |
| bound off-by-one on both sides (admits exactly 2^53) | **SURVIVED the new suite**; killed by pre-existing `TestCalculateObjectIdentityRefusesANonAXNumberNestedInAContainer` and `TestObservationEntryRefusalsBeforeShapeValidationReachProductionEntries`. See residual below. |

**Free prose is the disclosed unchecked area, so I drove it by hand.** Every refusal the new doc
comment claims, through the real exported `Canonicalize`, all seven refused with a matching typed
error: malformed UTF-8, lone high surrogate escape, unpaired low surrogate escape, unescaped
control character in a string, duplicate object names, trailing data after the top-level value,
257 containers open at once. The paired acceptance is real too — a 401-container shallow document
canonicalizes without error. The prose list that replaced the false "non-I-JSON" claim is
accurate; the fix did not swap one unverified claim for another.

**Citations checked by hand against `internal/specdoc/SPEC.md`**, independently of the test that
checks them: line 292 "A decoder MUST reject a numeric literal at or beyond `<code>2^53</code>`
even if its host language can represent it", 295 "Implementations MUST NOT round a value and
continue.", 301 `NUM-UNSAFE-NUMBER`, 302 `NUM-UNSAFE-ROUND`. All four verbatim, all at the
declared lines, and both fixture attributions land on the row that declares them. `specdoc.Load`
verifies SHA-256 against `specpin.DocumentSHA256`, so these are pinned, not transcribed.

**Division of guarantees is where a caller will see it**: the `Canonicalize` doc (DIVISION OF
GUARANTEES), plus `CalculateObjectIdentity`, `VerifyObjectIdentity`, `validateAXNumbers`, and
README. The claim "this package exports no standalone AX number validator" is true — the exported
production surface is exactly `Canonicalize`, `CalculateObjectIdentity`, `VerifyObjectIdentity`,
`CanonicalByteLength`, `ValidateObservationEvent`, `ValidateObservationStream`, and no exported
number validator exists. The four named entry points are exactly those from which
`validateAXNumbers` is reachable; `CanonicalByteLength` is correctly excluded.

**Negative-test carrier is sound.** The refusals are carried on an `extensions` member because
`validateExtensionValue` accepts any `json.Number` unexamined, so only `validateAXNumbers` can
refuse them — confirmed by the recursion-removal and call-deletion mutants, which reddened rather
than being absorbed by another guard. A green baseline with a safe extension number runs first,
so the refusals are not vacuous.

## Validation

- `go test ./... -count=1` exit 0, 11 packages ok, no FAIL lines.
- `go vet ./...` clean; `gofmt -l internal/ cmd/` empty.
- `internal/canonicaljson` coverage 97.2%.
- `curator status --check` exits 0 here with `worktree: go-testing-tools not-installed`. The
  producer reported this as a pre-existing Story-worktree provisioning anomaly rather than
  explaining it away, which is the right treatment; the diff touches no Curator-managed path.

## Residual (named, not closed silently)

1. The doc cites `NUM-UNSAFE-NUMBER`, whose normative literal is `9007199254740992` = 2^53
   exactly, but no *documented row* drives that literal — `auditedRoundedLiterals` holds 2^53+1,
   the u64 max, and `1.0`. A bound moved by exactly one on each side therefore survives this file
   and is caught only by the older package tests. The guarantee is held and tested; it is the new
   doc-derived pin that does not reach its own cited boundary. Cheap future hardening: add the
   fixture literal to the audited set.
2. Claims about RFC 8785 itself remain unmeasured (not vendored), disclosed accurately in the test
   file and the LOGBOOK. What is checked is one step downstream, against the vendored Appendix B
   transcription.
3. Free prose outside the enumerated rows, names and citations is still unchecked. I drove it by
   hand this cycle; that is a review act, not a standing pin.

## Verdict

ACCEPTED. The AC is met: the contract is documented as what the code does, with the reason quoted
from both RFC 8785 and the pinned SPEC; the documented behaviour is pinned by tests that take the
comment as *input* rather than restating it, so doc-without-code and code-without-doc both redden;
and the division of guarantees is stated at the call sites a caller reads. The
LOGBOOK entry is honest about the class recurring three times in one comment and about why the
previous pin blessed a false clause — the treatment applied (move the claim out of prose into a
driven, re-measured row) is the correct one and I confirmed it works by mutation.

No `commit_ack` supplied — this is a reviewer-archetype run. Acceptance handed to the
commit-owning orchestrator.
