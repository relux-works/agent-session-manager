# TASK-260830-8x76g1 — review verdict, CR revision 7

**Verdict: changes requested → `to-dev`.**

Reviewer run `RUN-260831-a35299` (claude/opus). Candidate tree
`2101a33011a27c98903679163545a7a778027b07` over base
`ad7275181ca82fc3fa29544e3893923a92d7b9d5`; the review worktree was verified
byte-identical to the candidate tree before and after every probe
(`git write-tree` from a scratch index returned the candidate OID both times).

Normative authority read directly: local clone of
`relux-works/agent-session-manager-spec` at commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, `SPEC.md`
sha256 `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`
— exactly the pin recorded in `.spec/README.md`. Sections 1.6, 2.1, 2.3, 5.1,
10.1–10.4 and 17.3 were read in full.

---

## F1 — BLOCKING. Session Record `name` is attested with no constraint at all

`internal/canonicaljson/closed_shapes.go:191` validates `name` with a bare
`requireString`, which only requires a non-empty UTF-8 string.

Section 5.1 field table declares:

| `name` | string | **Section 2.3 grammar** |

Section 2.3 defers the grammar to the Section 2.1 term (SPEC.md:363):

> | Session name | A mesh-unique human alias of 1–64 characters matching
> `[A-Za-z0-9][A-Za-z0-9._-]{0,63}`. |

Every other declared member of that table is enforced — digest, UUIDv7
subject/session equality, timestamp, `provider_id` through
`scalar.ParseProviderID`, `kind`/`execution_profile` closed enums, and the four
closed nested objects. `name` is the single field whose declared constraint is
silently dropped.

**Failure scenario, reproduced through both public entries** (log:
`probe-name-grammar.log`). Ten out-of-grammar names are calculated *and*
verified:

| Input `name` | Result |
| --- | --- |
| `-payments` (leading hyphen) | attested `sha256:da4723d4…` |
| `payments api` (space) | attested `sha256:4af1f9db…` |
| `../../etc/passwd` | attested `sha256:5cf8615e…` |
| `payments\napi` (LF control) | attested `sha256:97c806d0…` |
| `a\tb` (TAB control) | attested `sha256:891bb814…` |
| `платежи` (non-ASCII; §2.3 states the alphabet is ASCII) | attested `sha256:124660aa…` |
| 1024 `a` characters | attested `sha256:bf77f54b…` |
| 65 `a` characters (one over the bound) | attested `sha256:09474a06…` |
| `$(rm -rf /)` | attested `sha256:1a96681c…` |

`VerifyObjectIdentity` accepts the same objects with a correct omit-self claim
(PROBE-A2: `name` = 1024 chars + `/../escape` verified as
`sha256:1518d0ab…`). This is the same attestation-integrity class as CR rev1,
rev2, rev4 and rev6: identity attests an object the closed contract requires
the receiver to reject.

This is not a stray field. The DoD requires the constraints to be **derived by
systematic enumeration of every declared member**, and
`TASK-260830-8x76g1_constraint-enumeration.md` is the artifact that was
supposed to prevent a hand-picked subset. Its Section 10.1 paragraph lists
"the exact top-level member set, digest, UUIDv7 subject/session equality,
timestamp, host/workspace identifiers, provider/profile/kind enums, and the
closed Launch Plan, Task-board Reference, Board Identity, Board Goal, and Fork
Provenance nested objects" — `name` is absent from the enumeration exactly as
it is absent from the code. The enumeration is not exhaustive over the one
schema whose complete shape this candidate claims to own.

## F2 — README makes a completeness claim that does not reproduce

`README.md`: "The public calculation and verification entries currently accept
only Session Record `1.0.0`, Blob Descriptor `1.0.0`, and Transfer Manifest
`1.0.0`, **whose complete shapes are validated here**."

Given F1, that claim does not reproduce through either public entry. This is a
recurrence of the CR rev3 F2 finding (README claiming an audit that had not
happened). Either the claim is narrowed to what a reader can reproduce, or F1
is fixed so the claim becomes true.

## F3 — O(n²) case-collision scan stalls the identity gate on a valid input

`validateManifestEntries` (`closed_shapes.go:940-947`) compares each entry path
against every earlier path with `strings.EqualFold`, giving ~2.1×10⁹
comparisons at the declared `ManifestEntry[0..65536]` maximum.

Measured (PROBE-B/B2, `probe-perf.log`): a well-formed `workspace_tree`
Transfer Manifest with 65,536 directory entries encodes to **3,211,786 bytes**
— comfortably inside the 5,242,880-byte identity cap enforced in
`prepareObjectIdentity` — and `CalculateObjectIdentity` spends **6.61 s** on
it. Scaling is quadratic: n=2,000 → 17 ms; n=8,000 → 162 ms; n=20,000 → 875 ms.

Transfer Manifests arrive from peers over Mesh RPC, so this is CPU
amplification on an untrusted input at a validation gate. Entries are already
required to be strictly bytewise-sorted and unique, so a
`map[string]struct{}` keyed on the lowercased path makes the check O(n) with
no loss of coverage. `validateWorkspaceSnapshot` group paths and
`validateGitSubmodules` share the pattern but are bounded to 256 and are not a
practical concern.

---

## What is genuinely solid — do not rework these

Recorded so the next producer does not churn the parts that already hold.

**The totality gate is real, not a claim.** Deleting one row
(`register("urn:ax:schema:migration-checkpoint", …)`) makes the package panic
at init: `immutable-object shape validator registry is invalid: missing
immutable-object shape validator for urn:ax:schema:migration-checkpoint@1.0.0`.
A newly registered schema cannot fall through to extension-only attestation.
Confirmed by mutation.

**The fuzz-wiring gate is real.** Adding an unwired
`FuzzReviewMutantUnwiredTarget` to `internal/scalar` fails
`TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget`:
`configured validation contains "go test ./internal/scalar -run=^$
-fuzz=^FuzzReviewMutantUnwiredTarget$ -fuzztime=100x -parallel=1" 0 times,
want exactly once`. The check is AST-derived from every `_test.go` in the repo,
so it is not circular. Confirmed by mutation.

**Four narrowing mutants are each killed by a named existing test** — the
negative evidence is not delete-only:

| Narrowing mutant | Killed by |
| --- | --- |
| `requireBoundedString` counts bytes instead of characters | `TestTransferManifestNestedUnicodeBoundsAcceptExactMultibyteLimits` |
| BlobChunk index equality relaxed to `index+1` | `TestClosedIdentityShapesRefuseUnknownMembersAndBlobChunkInvariants`, `FuzzClosedIdentityShapeRefusal` |
| extensions reverse-DNS grammar relaxed to "contains a dot" | `TestIdentityExtensionKeysUseCompleteReverseDNSGrammar` |
| symlink target bound widened 4096 → 8192 | `TestManifestStringBoundsCountUnicodeCharactersAtProductionEntries` |

**The byte-vs-character audit is correct in both directions.** I checked every
bounded string against the pinned text, not just the previously named ones:

- Declared in **bytes** and implemented in bytes: Launch Plan `argv` elements
  ("1–4,096 UTF-8 bytes"), `env_literals` values ("at most 4,096 UTF-8 bytes
  each"), `task_element_id` ("1–128 printable non-control UTF-8 bytes").
  PROBE-G confirms 4096 ASCII bytes accepted, 4097 refused, and 2048 multibyte
  characters (6144 bytes) refused — which is what the spec requires.
- Declared in **characters** and implemented with `utf8.RuneCountInString`:
  `media_type[1..255]`, symlink `target[1..4096]`, `logical_id[1..128]`,
  `goal_id[1..128]`, `repository_identity[1..256]`, `tree_identity[1..256]`,
  GitRemote `name[1..128]`. PROBE-D confirms 256 and 128 multibyte characters
  are **accepted** at the boundary; PROBE-E confirms 4096 accepted / 4097
  refused.
- Bare `string` correctly acquires no undeclared bound: §17.3 `schema_id` is
  `string`, and an empty `schema_id` is accepted (PROBE-I) — matching the
  enumeration's stated rule.
- extensions keys: 253 accepted, 254 refused (PROBE-C); the grammar
  `^[a-z][a-z0-9-]{0,62}(\.[a-z][a-z0-9-]{0,62})+$` matches §1.6 exactly,
  including the at-least-one-dot requirement.

**Sections 10.2–10.4 member sets check out line by line** against the pinned
tables: Blob Descriptor's exact seven members with no extension point, BlobChunk's
four, the Transfer Manifest's fifteen, the ManifestEntry tagged union's four
variants, WorkspaceSnapshot's two, both snapshot-member variants (15/8), and
GitRemote/GitHead/GitObjectPack/GitIndex/GitIndexEntry/GitSubmodule/GitFeatures
(3/3/7/6/8/14/10). Kind invariants, sparse-pair coupling, submodule
depth-16/count-256/acyclicity, gitlink↔stage-0-mode-160000 binding, and the
TM-GIT-N2 stage-4 and entry-count refusals all match.

**Accepted-leaf content is intact.** `internal/scalar/{scalar,names,path,
integer,time_digest,uuid,scalar_test}.go` are byte-identical to
`TASK-260830-8x76g1_accepted-leaves-tree.tar.gz`; this leaf only adds
`git.go`, `git_test.go`, `fuzz_test.go` and `testdata`. `canonical.go` changes
are the additive `prepareObjectIdentity` composition that installs the gate.

**Gates I ran myself, all exit 0**: `go build ./...`, `go vet ./...`,
`go test ./... -count=1` (8 packages), `tracecheck -section 17.3`
(`assigned_scopes=1`), and all four fuzz targets at the configured
`-fuzztime=100x -parallel=1`.

---

## Rework instruction

Do **not** just add a `name` regex and resubmit. Five rounds have now ended the
same way: the validator gains exactly the fields the previous review named, and
the next review finds the next unenumerated one. The structural fix that ended
this cycle for the registry (registry-derived completeness test) has not been
applied to the *field* level.

1. Fix F1 by validating `name` against the §2.1 term
   `[A-Za-z0-9][A-Za-z0-9._-]{0,63}` with a 1–64 character bound, proven in
   both directions through `CalculateObjectIdentity` **and**
   `VerifyObjectIdentity`: a 64-character valid name accepted, and 65
   characters / leading hyphen / space / control character / non-ASCII refused.
2. Make the enumeration per-member and total for every schema with a complete
   validator. `TASK-260830-8x76g1_constraint-enumeration.md` must carry one row
   per declared member of Session Record 1.0.0, Blob Descriptor 1.0.0 and
   Transfer Manifest 1.0.0 (including every nested closed type), each naming
   the enforcing call site. A member whose disposition is "no additional
   constraint beyond its type" must say so explicitly and quote the spec text
   that makes it a bare type — so a dropped constraint is visible as a missing
   row rather than invisible as a silent omission. Prefer deriving the member
   list from the closed-shape `requireExactMembers` argument lists so the
   artifact and the code cannot drift.
3. Fix F2: after F1, re-check that every README sentence about this package
   reproduces through the two public entries; remove or narrow anything that
   does not.
4. Fix F3: replace the quadratic `EqualFold` scan in `validateManifestEntries`
   with a lowercased-path set, and add a bounded regression asserting that a
   65,536-entry manifest validates well within the suite budget.

Re-review will re-run the same probes plus the mutation set above.
