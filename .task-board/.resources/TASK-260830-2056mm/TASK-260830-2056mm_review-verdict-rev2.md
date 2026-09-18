# TASK-260830-2056mm — independent review verdict, CR-TASK-260830-2056mm-2 revision 2

**Verdict: CHANGES REQUESTED (routed `to-dev`).** One P2 class, five P3.
No P1. The story-final Change Request is otherwise sound on my own
instruments: the two v4 payloads are member-for-member the pinned text and
are validated at the append boundary by the landed canonical owner; evidence
resolution, identity refusal on every surface, lost-create recovery (with a
real SIGKILL seam), takeover/attach authority, the registry re-pin, the
re-derived digest, tracecheck and the README figures all hold. The P2 is
the one closed-object parser this Story owns — the Section 4.B Terminal
Instance Binding 1.0.0 — which admits two members the pinned table refuses
and, through one of them, reaches the forked omit-self identity rule with
JSON numbers and nested duplicate members that the landed identity rule
refuses. That is exactly the "never fork canonicaljson / the AX number
model" class the producer brief listed under (c), and the parser's
member-grammar arms are executed 7 of 30 by the committed suite.

Reviewer: RUN-260918-acd5a8 (claude-opus-5 max). Reviewed bytes: base
`c3aae73df906df1b5aa962c7bc3c16437dd920fa` (= `origin/main`, fetched),
candidate tree `f8adb1616ce3d5e702e50e47ad5795859f4f12d5` (recomputed from
the live Story worktree through a temporary untracked-aware index: equal),
patch sha256 `1e4b15dde201f1c7f3a3169eb985d92e867877828fffe1d383babd451c4b7684`
(equal to the CR record), 89 changed paths, 0 outside the Story's scope
(`logs/00-identity.log`). Every probe ran in isolated `git archive` copies of
the exact tree under `.temp/TASK-260830-2056mm/review-rev2/` (`cand/`
tree-exact checks, `cand-h/` shipped harnesses, `cand-m/` reviewer mutants,
`cand-p/` probes); `cand-h` is byte-identical to the candidate after all six
harness passes and `cand-m` after both mutant passes. The live Story
worktree, index, branch and HEAD were never touched (`git status` is the
same 18-path list at the end as at the start; tree OID unchanged).
`PYTHONDONTWRITEBYTECODE=1` everywhere, 0 `__pycache__`/`.pyc` in any copy.
Every instrument, raw log and subprocess exit is in
`TASK-260830-2056mm_review-evidence-rev2.tar.gz` (paths below are relative
to that archive).

## 0. Inherited advisories (final leaf)

| Item | Where it landed | Grade |
| --- | --- | --- |
| 1geqhj rev4 P3-1 journal-only restore closure | `TestRunJournalOnlyRestoreClosureFromNewest` (axpane `leafclose_test.go`), row `N-journal-closure` KILLED ×2 in my reruns | Closed |
| 1geqhj rev4 P3-2 pure `Decide` on a missing closure | axpane TRACEABILITY story-close section: owner-stated bound (unreachable from every `Run` path) | Explicitly deferred |
| 1geqhj rev4 P3-3 remote owner, newest absent locally | axpane TRACEABILITY story-close section: fail-closed bound, deferred load is future work | Explicitly deferred |
| 1geqhj rev4 P3-4 `EmitParked` winner binding | production fix `events.go` (`decision.WinningLeaseID == params.LeaseID`), `TestEmitParkedWinningLeaseMismatchRefuses`, row `N-parked-winner` KILLED ×2 | Closed |
| 1geqhj rev4 P3-5 corrupt checkpoint blob is not absence | `TestRunCorruptCheckpointBlobIsNotAbsence`, row `N-ckpt-corrupt` KILLED ×2 | Closed |
| 1geqhj rev4 P3-6 reattach carries two instance IDs | closed by kkh1an (documented authority on reattach); nothing left for this leaf | Closed (predecessor) |
| 1geqhj rev4 P3-7 archive hygiene | this leaf's two archives are task-scoped, gzip, 0 pycache, no foreign-task artefacts (`logs/80`) | Closed |
| kkh1an rev4 P3-1 expiry recheck at the third position | `TestRV4W_ExpiryRecheckedBeforeThirdEffect`, row `N-engine-recheck-auth-expiry-third` KILLED ×2 | Closed |
| kkh1an rev4 P3-2 replay-site generation binding / corrupt image | `TestRV4W_ReplayedImageGenerationBindingAtEngine`, `TestRV4_CorruptCompletionImageIsIntegrityFailure`, rows `N-resume-checkresult-stale`, `N-resume-checkresult-disposition`, `N-resume-corrupt-image` KILLED ×2 | Closed |
| kkh1an rev4 P3-3 in-loop deadline instant | `TestRV4W_LoopDeadlineInstantCancelsWaiting`, row `N-engine-loop-deadline-instant` KILLED ×2 | Closed |
| kkh1an rev4 P3-4 mixed-case enum siblings | `TestRV4W_AuthorizationKindCaseSiblingRefused` + three corpora, row `N-auth-kind-case` KILLED ×2 | Closed |
| kkh1an rev4 note: `HandoffFailed` member if the final leaf wires a reporter | this leaf wires no `HandoffFailed` reporter (grep: only tests set it), so not applicable — but the results do not say so | Not applicable, unrecorded (P3-5) |
| kkh1an rev3 P3-5 / kkh1an doc: restore and terminate-stale execution "belongs to the story's final leaf" | termbind TRACEABILITY: "remains refused by the terminstance scope gates (owner: terminstance)" — the two leaves point at each other and neither brief assigned it | Deferred with a circular owner (P3-5) |

All eight closures executed on the exact tree (`logs/60-leafclose-witnesses.log`: 8 PASS).

## 1. What holds (verified myself on the exact tree)

| Check | Result | Evidence |
| --- | --- | --- |
| gofmt (empty list) / `go build ./...` / `go vet ./...`; `GOOS=linux` and `GOOS=windows` builds; `GOOS=windows go vet` termbind; JSON validity; `git diff --check base..candidate`; `cataloggen -adopted -check` | all exit 0 | `logs/02`, `logs/80` |
| `go test ./... -count=1` on the exact tree | 41/41 packages `ok` (40 in the first run; `internal/specpin` failed only because I had stripped `.task-board` from the copy and passed once restored) | `logs/04`, `logs/04b` |
| `go test ./... -race -count=1` (Story packages + the other 34 packages, two bounded calls) | 41/41 `ok`, 0 `DATA RACE` — the rev1 race is gone | `logs/70`, `logs/71` |
| `go test ./internal/termbind -count=1 -v` | 187 PASS lines, 0 FAIL, 0 SKIP; `-cover` 76.9% | `logs/83`, `logs/termbind.coverprofile` |
| CR construction suite | runtime-recorded `required=30 green=30 failed=0 missing=0` on the exact tree (accepted from the attached bounded `_rev2-validation.log`; the 17 fuzz gates and `task-board validate` were not rerun by me) | resource |
| Payloads vs the pinned text (`SPEC.v0.7.0.md` 1956–1985): `terminal.created` = terminal_binding_id, terminal_backend_id, implementation_version, protocol_version, evidence_ids[1..256]; `session.resumed` = checkpoint_id, execution_profile, profile_source_event_id\|null + the same five — identical to `axpane.TerminalCreatedPayload`/`ResumedPayload` and to the landed `canonicaljson` 4.0.0 registry (`core_records.go:574-575`, `validateTerminalV4Payload`), which is composed at the append boundary by `axpane.Emit → sessrepo.AppendEvent → VerifyObjectIdentity` | holds | source read |
| Bounds at the append boundary: 256 and 1 admit, 257 and 0 refuse (`member evidence_ids exceeds maximum length 256` / `requires at least 1 entries`); unsorted and duplicated sets refuse at `ResolveEvidence` and at the emission entries | holds | `TestRV2_AppendBoundaryEvidenceBounds` `logs/15`; producer rows 17–20, 29 |
| Evidence resolution: eleven foreign kinds, unresolvable ID, substitution under another ID, foreign backend / protocol drift, landed admission faults (claim, forgery, expiry, drift) all refuse the mismatch class; nil registry fails closed (`terminal_backend_not_found at registry unavailable`), never panics; the event bytes carry the opaque digest and no native reference, generation, or instance ID | holds | producer rows 16–26, 34; `TestRV2_ResolveEvidenceNilRegistry` `logs/14` |
| Identity: eight forbidden forms refuse at `CheckInstanceIdentity`, `ParseTerminalBinding`, the landed `AdmitProviderDescriptor`, the operation-body wrappers, `axpane.Store.Bind` (plain error, nothing persisted) and `AttachStore.Attach`; the story gate (`scalar.ParseUUIDv7`) agrees with the landed body authority (`environ.CheckUUIDv7`) on a 15-spelling corpus (upper-case, v4, wrong variants, `urn:uuid:`, braces, NUL, 35/37 chars): 0 disagreements | holds | `TestRV2_IdentityGateAgreesWithLandedUUIDv7` `logs/12` |
| Recovery: real SIGKILL between the durable bind + effect and the result → the ONE child; duplicate retry same verdict; changed op in the fold-derived open window → `idempotency_mismatch`, outside it proceeds; five unprovable shapes → `unavailable`/`status_first`, never absence; status read failure is an error. My narrowings on the two arms the producer did not mutate — absence claimed without the identity match (`RV2-M7`), backend-`unavailable` reported as the child (`RV2-M8`) — KILLED ×2 by `TestRecoverCreateNeverClaimsAbsent` | holds | `logs/50`, `logs/51` |
| Authority: emission under a losing epoch, under a REMOTE winner (presented = winner, local host not the holder → `not_owner`) and under unverified ownership refuses with zero appends (composition of `fencing.AuthorizeMutation` through `axpane.Emit`); takeover selects the conpty backend with a new digest and a new v4 event, the first event's bytes unchanged, one Logical Session, no fork; attach emits nothing and takes no repository | holds | `TestRV2_EmitUnderRemoteWinnerRefuses` `logs/14`; producer rows 32, 51, 54 |
| Story close: the ADOPTED `ownership.v0.7.0.json` is edited (trunk rows merged, four bindings upgraded, `section:4.1` added with the mirrored `storyNewV070Bindings` literal list); `reviewedOwnershipCanonicalSHA256` re-derived by me from the registry as it stands: `89944b8c6ec75703c0e07432890182ebbb9bb405f28502b05a6632662cebee9b` (canonical bytes 166083) = the pin; `tracecheck` green with exactly `contracts=64 normative_sections=36 acceptance_cases=147 fixtures=33 compatibility_contracts=55 assigned_scopes=0` / `bindings=69 full=2 partial=9 sliver=9 unevidenced=45 unmeasured=4 unowned=7 clauses_discharged=63/574`; section runs 4.1 → 1/5 sliver, 4.B → 1/12 sliver, 4.D → 1/3 sliver, 5.2 → 3/18 sliver, 7.A → 1/2 partial (each exit 1 by design); every clause line/excerpt re-read verbatim in the pinned document; `TestREADMEMeasuredCoverageMatchesTracecheckReport` executed and PASS; README figures equal the printed report; README and LOGBOOK purely additive on the c3aae73 content (README removes only the superseded coverage-figure lines) | holds | `logs/24`, `25`, `26`, `27` |
| Bounds and hygiene: `task-board.config.json` byte-identical to trunk; the only trunk-moved paths (2fc6d50..c3aae73) in the candidate delta are README/LOGBOOK (additive) and the traceability files (edited on top of c3aae73); 0 `__pycache__`/`.pyc`; both producer archives are real gzip, task-scoped, carry `cmd01..cmd30` (cmd05 in 8 chunks, 0 `DATA RACE`), 30+30 raw per-plant termbind logs, blob OIDs before/after, no foreign-task artefacts | holds | `logs/80` |
| Shipped harnesses, isolated pristine copy, two full passes each with raw per-plant output and subprocess exits | termbind 30/30 ×2 (28 narrowing + 1 labelled arm-delete KILLED, `C-control` SURVIVED); axpane 56/56 ×2; terminstance 122/122 ×2; verdict lists identical across passes; copy byte-identical afterwards | `logs/harness/*-pass{1,2}-verdicts.log`, `*-verbose.log` |

## 2. Coverage statement

Producer claim: 60 of 60 rows driven. Re-derived (`logs/82-ratio-check.log`):
60 numbered rows (1..60, contiguous, unique); 53 distinct cited top-level
tests = the package's whole `go test -list`, every one PASS in my verbose
run. **I measure 59 of 60 driven at the named entry.** Row 57 ("v4 carries
no v1 derivation member: a v1 reader stays inert") is a payload-member
census — no v1-v3-only reader exists on trunk, the landed `sessstate`
reader is v4-aware — so it is a stated bound wearing a driven row's clothes
(P3-4a). Rows 11 and 12 are driven, but at wrapper entries no production
path calls, and their refusal is the landed parsers' (P3-1). Rows 3–8 hide
the real gap: the Binding parser's member-grammar arms (P2-1).

## 3. Findings

### P2-1 — `ParseTerminalBinding` admits two members the pinned §4.B Binding table refuses, and through one of them the forked identity rule launders JSON numbers and nested duplicates (production; class: closed-shape rule unenforced + landed-gate fork, producer-brief patterns (c) and (f))

Pinned text (`SPEC.v0.7.0.md` 1058–1073): `protocol_version` = "semver in
Terminal Backend Protocol major 1, equal to the Probe"; `extensions` =
"exact empty object `{}`". Both are decidable from the document alone and
the landed owners enforce both on their own objects (`CheckVersionTuple`
refuses `protocol_version major 1`; `terminalbackend.checkExtensions`
admits exactly `{}`). The candidate's parser (`identity.go:183`,
`identity.go:222`) checks `environ.CheckSemver` only and
`environ.CheckExtensions`, which admits any reverse-DNS-keyed object with
arbitrary values.

Executed (`probes/zz_review2_probe_test.go`, `logs/12-probes-all-pass{1,2}.log`,
identical across passes):

- `TestRV2_BindingProtocolVersionMajorOne`: `protocol_version` `2.0.0`,
  `0.9.0`, `3.1.4` are **ADMITTED** (parsed binding returned). Control: the
  landed `CheckVersionTuple` refuses `2.0.0` with
  `terminal_backend_implementation_drift … protocol_version major 1`
  (`TestRV2_LandedTupleRefusesNonMajorOne`). Reviewer mutant
  `RV2-M3` (admit exactly `2.0.0` past the semver arm) SURVIVED ×2 because
  it is equivalent to the shipped code: no major-1 arm exists to narrow.
- `TestRV2_BindingExtensionsMustBeEmptyObject`:
  `"extensions":{"com.example.note":"y"}` is **ADMITTED**.
- `TestRV2_BindingIdentityLaundersNumbers` (4 subtests): with the
  `binding_id` computed the way the candidate's own `bindingIdentity`
  computes it, `{"com.example.n":1.0}`, `9007199254740993`, `1e2` and `-0`
  in `extensions` are all **ADMITTED** — `bindingIdentity` decodes to
  `map[string]any` (float64), re-marshals, then JCS-transforms, so the
  digest is taken over a rounded/reformatted document. The landed
  `terminalbackend.objectIdentity` walks the document with
  `refuseIdentityNumbers` before the transform (NUM-UNSAFE-NUMBER); the
  fork has no walk. The doc comment on `bindingIdentity` ("Every member is
  type-checked before this runs, so no JSON number can reach the digest
  bytes") is false as shipped: extension values are not type-checked.
- `TestRV2_BindingExtensionNestedShapes/nested_duplicate_member`:
  `{"com.example.o":{"a":"1","a":"2"}}` is **ADMITTED** (`json.Decoder`
  keeps the last duplicate; the strict frame decoder guards the top level
  only). Lone surrogates are refused by the frame decoder — that half is
  fine.
- The parser's refusal arms are executed 7 of 30 by the committed suite
  (`logs/82-ratio-check.log` and the coverprofile census in
  `logs/termbind.coverprofile`): schema_version, binding_id digest,
  session/host/incarnation UUIDv7, terminal_backend_id, implementation
  semver, protocol semver, created_at, the whole non-null
  `supersedes_binding_id` path and the extensions arm are never reached.
  Reviewer mutant `RV2-M5` (admit exactly `"not-a-digest"` as
  `supersedes_binding_id`) SURVIVED ×2 with a non-equivalent witness
  `TestRV2W_SupersedesMustBeDigest` (PASS pristine, FAIL ×2 under the
  plant; `logs/53`).

Why P2 and not P1: the Binding is host-local, AX-authored and never
replicated, so the over-admission produces a stored document with junk
extensions or a foreign protocol major, not an unsafe launch or
authorization. Why not P3: it is the one new closed-object parser this
Story owns, two rows of its pinned table are unenforced, the identity rule
is a fork of exactly the class the brief named, and the matrix's rows 3–8
present the parser as measured while 23 of its 30 arms are not.

Fix (for the producer): add the two arms delegating to the landed
authorities (`semverMajor(protocol) == 1` as `CheckVersionTuple` does;
extensions exactly `{}` as the landed manifest/probe/evidence parsers do),
make `bindingIdentity` refuse any JSON number before the transform (mirror
`refuseIdentityNumbers`, or export the landed rule from `terminalbackend`
and call it) and refuse nested duplicate members, then drive every member
arm of the table with a negative through `ParseTerminalBinding` and ship a
narrowing row per arm (major-1 admitted for one value, one non-empty
extensions object admitted, one number admitted, `supersedes` one
non-digest admitted). Re-run `TestBindingIdentityAgreesWithLandedAdmission`
with a numeric extension value so the agreement claim covers the number
model.

### P3-1 — The operation-body pre-scans add no refusal class; their mutant rows are killed by a Go error type, and the LOGBOOK/harness notes state the opposite (evidence + prose)

`terminstance.ParseMutationContext` and `ParseStatusBody` already refuse
every non-UUIDv7 `terminal_instance_id` with the pinned wire code
`terminal_backend_protocol_error` (`context.go:78`, `status.go:126`;
`TestRV2_LandedBodyParsersAlreadyRefusePinnedCode`: 16/16 forms, code
`terminal_backend_protocol_error`, type `*terminstance.Error`). The termbind
wrappers `AdmitMutationContext`/`AdmitStatusBody` re-run the same rule and
return the same code as a `*terminalbackend.Error`. Rows `N-prescan-context`
and `N-status-prescan` are KILLED only because `requireCode` asserts the
Go type `*terminalbackend.Error`; with the pre-scan gone, production still
refuses the PID with the pinned code. The harness notes ("the landed plain
refusal surfaces instead of the pinned code") and LOGBOOK F2 ("the
mutation-context/status/bootstrap surfaces needed one (landed plain
refusals)") are factually wrong for the two body surfaces, and no wrapper
exists for the bootstrap surface at all (row 14 drives the landed
`Store.Bind`, whose refusal is a plain uncoded error — the README sentence
"with the pinned protocol class on … the bootstrap binding" outruns it).
No production caller uses either wrapper (grep: tests only). Either drop
the wrappers and name the landed parsers as rows 11/12's call sites, or
keep them as a documented redundancy with the notes corrected; append a
LOGBOOK correction paragraph (the log is append-only).

### P3-2 — `EmitResumed` binds neither `checkpoint_id` to the chain nor the profile pair to the referenced checkpoint's closure (production characterization; same shape as 1geqhj P3-4)

`TestRV2_ResumedCheckpointNotBoundToChain` (`logs/14`): a never-published
digest with `execution_profile: yolo` and a null source appends through
`EmitResumed`, and the landed fold then reports that digest as the newest
checkpoint (`HasCheckpoint=true`, `Newest=sha256:fafa…`). The landed
`sessprofile.CheckResumedPair` ("binds one observed session.resumed pair to
the effective pair of its referenced checkpoint's own closure") has zero
production callers. Through the wrapper's decision both values would come
from the fold; a direct caller can author a lying newest. Compose the
landed check (derive the want over the referenced checkpoint's heads) and
require the checkpoint to be one the chain/store knows, or state the bound
in `emit.go` and TRACEABILITY in those words.

### P3-3 — Unmeasured arm found by narrowing: the recovery generation bound (evidence)

`RV2-M9` (`recover.go`: skip `GenerationDigest` exactly for a 257-character
generation before the status read) SURVIVED ×2; witness
`TestRV2W_RecoverGenerationBoundBeforeStatusRead` (PASS pristine:
`terminal_backend_stale_generation at backend_generation bound` with zero
status reads; FAIL ×2 under the plant: the read runs). Commit the witness
and a row. (`RV2-M6`, the manifest implementation-version arm of
`checkEventBinding`, also SURVIVED ×2 but is equivalent: the landed
`AdmitProbe` refuses the same member with
`terminal_backend_implementation_drift`, `TestRV2W_ManifestImplDriftCaughtByLandedAdmission`
PASS under the plant ×2 — note only.)

### P3-4 — Traceability and prose accuracy

- (a) Row 57 and the registry's 5.2#18 discharge rest on a payload-member
  census; no v1-v3-only reader exists on trunk, so "a v1 reader stays
  inert" cannot be driven here. Restate as a bound (the v4-aware landed
  reader retains v4 bytes verbatim and the query read writes nothing —
  that half is driven by row 56).
- (b) The 7.A#1 clause and the section 7.A binding cite
  `terminal-instance-identity-exact` for "the eight-form identity pin
  through the same entry", but that acceptance case lists no test that
  reaches `AdmitProviderDescriptor` (`TestAdmitDescriptorRefusesForbiddenIdentity`
  is cited nowhere in the registry). The clause is discharged by the landed
  `terminal-provider-descriptor-7a` alone; add the descriptor test to the
  case or drop the citation.
- (c) `resolve.go` says "Every failure refuses
  terminal_backend_manifest_probe_mismatch" while the three shape pre-check
  arms return uncoded `fmt.Errorf` values (`want 1..256`, `not sorted
  unique`, `not a digest`); the pre-check itself is a third copy of a rule
  the landed `canonicaljson` (and `axpane.checkEvidenceIDs`) already
  enforce — defence in depth, but say so instead of "every failure".
- (d) README: "with the pinned protocol class on … the bootstrap binding"
  (see P3-1).

### P3-5 — Deferral hygiene

Restore and terminate-stale execution is deferred by kkh1an to "the
story's final leaf" (`terminstance/doc.go`, rev3 P3-5 exit route) and by
this leaf back to "owner: terminstance"; neither brief assigned it, so the
Story closes with a normative operation pair that has no follow-up owner.
Record the real owner (a Story follow-up task) in the results and LOGBOOK,
and state in the results that the kkh1an `HandoffFailed` note is not
applicable because this leaf wires no reporter.

## 4. Reviewer instruments

`probes/zz_review2_probe_test.go` (15 top-level probes: 4 CHARACTERIZATION
failures = P2-1 (8 failing subtests), 3 witnesses `TestRV2W_*` and 8
composition/agreement/bound probes PASS; two passes identical,
`logs/12-probes-all-pass{1,2}.log`);
`probes/zz_review2_digest_probe_test.go` (`logs/25`);
`probes/review2_mutants.py` (9 narrowings + 2 controls against the whole
committed termbind suite, two passes with identical verdicts, raw per-plant
logs with exits and diffs in `logs/rv2-mutants-pass{1,2}/`: KILLED — M4
native_reference 513, M7, M8, M10 attach foreign-session lookup, M11
first-pair inversion, control K; SURVIVED — M3 equivalent-by-absence (P2-1),
M5 non-equivalent (P2-1), M6 equivalent (landed admission), M9
non-equivalent (P3-3), control C); `probes/run_witness_under_mutant.py` +
`probes/run_witnesses.sh` (`logs/53`, `logs/rv2-witness-pass{1,2}/`);
shipped harness reruns (`logs/harness/`); `logs/00`–`83`, coverprofile,
tree listings, `REVIEW-MANIFEST.txt` (sha256 of every file).

## 5. DoD walk (live merged checklist)

Production entry points implement the scoped deliverable — yes for events,
recovery and identity; the Binding parser under-enforces its table (P2-1).
Positive/negative/compatibility/recovery tests pass with logs — yes.
README/traceability without unsupported claims — mostly; P3-1/P3-4 name
the sentences that outrun the tests. Uncommitted in the Story worktree —
yes (tree OID equal, HEAD = checkpoint). Ratio — 59 of 60 measured against
the claimed 60 of 60. Negative tests that fail when the gate admits — yes
for the gates the rows name; two arms of the Binding parser have no gate
to test. Narrowing mutant per gate — shipped 30/56/122 KILLED ×2 with
working controls; my 9 narrowings: 5 KILLED ×2, 2 non-equivalent survivors
with witnesses, 1 equivalent, 1 equivalent-by-absence. No source-text gate
exists. Lint/build — clean. Outcome artefacts — attached, hygienic.
Logbook — newest-first, additive, one factual error (P3-1). Implementation
matches AC — not for the Binding table (P2-1). Fits the architecture — the
identity rule is forked from the landed owner (P2-1); everything else
composes. Tests green — yes. Gates attacked, not read — yes.

## Rework scope (for the producer)

1. `ParseTerminalBinding`: enforce `protocol_version` major 1 and
   `extensions == {}` per §4.B; make `bindingIdentity` refuse JSON numbers
   (at any depth) and nested duplicate members before the JCS transform,
   preferably by reusing the landed `terminalbackend` rule; drive every
   member arm of the Binding table negatively through the entry and ship
   narrowing rows for the new arms plus `supersedes_binding_id`; re-run the
   landed-agreement test with a numeric extension value.
2. Commit `TestRV2W_RecoverGenerationBoundBeforeStatusRead` (or an
   equivalent) with a harness row.
3. `EmitResumed`: compose `sessprofile.CheckResumedPair` and a
   known-checkpoint check, or state the bound explicitly.
4. Correct the prose: harness notes for the two pre-scan rows and LOGBOOK
   F2 (append a correction), README's "pinned protocol class … bootstrap
   binding", `resolve.go`'s "every failure", row 57 / 5.2#18 as a bound,
   the 7.A citation.
5. Record the restore/terminate-stale follow-up owner and the
   `HandoffFailed` non-applicability in the results and LOGBOOK.
6. Rerun your own package suite, the three harnesses and the configured
   suite before handoff; keep the candidate uncommitted on the Story
   branch.

Verdict recorded by RUN-260918-acd5a8 by routing `to-dev`; no product edits,
commits, checkpoints or integration were performed; the isolated copies are
deleted after attachment. `commit_ack` is not supplied.
