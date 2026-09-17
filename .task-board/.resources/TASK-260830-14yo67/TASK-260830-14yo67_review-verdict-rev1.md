# Review verdict — TASK-260830-14yo67 rev1 (CR-TASK-260830-14yo67-1): ACCEPT

Role: reviewer (RUN-260917-3e0204, muse-spark xhigh). Method: isolated immutable
probes only — full tree copied to scratch, live Story worktree never mutated;
all product evidence below was reproduced by the reviewer, never accepted from
producer logs alone. Live tree at verdict time: `M LOGBOOK.md` + `?? internal/sessckpt/`
only; HEAD still `62d446304391af187c417a88a2e14012b467956e`.

## Verdict: ACCEPT revision 1 → integrating

The candidate implements the typed checkpoint closure (provider identity,
workspace manifests, task-board bundle, terminal evidence, source head) over the
landed owners without duplicating them, drives consumer admission in tests,
ships crash/idempotency evidence with real SIGKILL, and proves every gate by
narrowing mutation. Two non-blocking follow-ups are recorded below (P2 registry
re-pin, P3 README section); both are Story-close items by precedent and neither
can be fixed inside this CR without self-minting the review-pinned digest.

## 1. Spec fidelity (pinned v0.6.0 §§5.4, 10.5–10.6, 13.12–13.13) — PASS

- §5.4 member set extracted from `SPEC.v0.6.0.md:1988-2005` (16 closed members)
  matches `buildCandidate` (`capture.go:341-358`) exactly; `subject_id ==
  session_id`, `status = validated`, omit-self `checkpoint_id` via
  `canonicaljson.CalculateObjectIdentity` + drift check in `identifyCandidate`.
- `TestSpecExampleAttestsThroughConsumerOwner` rerun by reviewer: PASS — verbatim
  §5.4 example attests through `sessrepo.AttestCheckpointRecord` with digest
  `sha256:e05199…819656` equal to the pinned value.
- CP-N1..CP-N4 each refused with `incompatible_schema` on both layers
  (`errors.Is` `ErrInvalidCheckpoint` + `canonicaljson.ErrInvalidIdentity`):
  CP-N1 `cp_n1_background_idle_false`, CP-N2 `cp_n2_direct_null_provider`, CP-N3
  `cp_n3_both_present`, CP-N4 `TestAdmitRefusesUnknownSafeBoundaryMember` (with
  benign re-identified control that admits).
- §§10.5–10.6: no plan/journal bytes written here — stated bound accepted; closure
  digests bound as plan inputs, variant gate refuses unselectable closures.
- §§13.12–13.13 rows mapped in `TRACEABILITY.md` and each driven except the
  ENOSPC dedicated fault (stated bound; torn-read path driven instead).
- AC coverage: **8 of 8 rows driven** through `Store.Capture` / `Store.Admit` /
  `Store.Get` (concur with producer matrix; rows 5–7 additionally driven by
  reviewer probes below).

## 2. Consumer admission — PASS (reviewer probes, iso copy)

| Vector | Entry | Result |
|---|---|---|
| heads strictly after owning lease | `Capture` | refused `ErrInvalidCheckpoint` (own-gate epoch text) |
| creator ≠ holder | `sessquery` consumer | refused `selector_observation_unavailable` |
| bundle-leg record as direct kind | `Admit` | refused `ErrInvalidCheckpoint` |
| moved task-board digest (re-identified) | `Admit` | refused (identity mismatch) |
| evidence member right name, wrong type | `Admit` | refused (owner type gate) |
| unknown extra top-level member | `Admit` | refused (closed-shape gate) |
| unknown session | `Admit` | propagates owner `ErrUnknownSession` |
| later-epoch head (re-identified) | `Admit` | refused `ErrInvalidCheckpoint` (mirror arm fires) |

File: `review-evidence/zz_review_probe_test.go`, log `review-probes.log`: 6/6 PASS.

## 3. Durability / idempotency — PASS

- Reviewer reran in live tree: `go test ./internal/sessckpt -run
  'TestCrashBeforeDurableWriteIsSafeRetry|TestCrashBetweenBlobAndReceiptResumes|
  TestCrashAfterReceiptReplaysRecordedResult|TestCaptureCrashChildSelfTerminates'
  -count=2` → all 4 PASS twice (exit 0); `-race -count=1` → ok (exit 0).
- Own fault-injection states: (a) blob-without-receipt resumes identical retry
  under one identity; (b) receipt-without-blob → retry errors (conflict), `Get`
  operational `os.IsNotExist` (never invalid-class, never admitted); (c) torn
  blob → `Get` refuses `incompatible_schema`.
- Replay writes nothing (receipt mtime unchanged); moved inputs → conflict, no
  write; different-op identical inputs share checkpoint identity (pure digest);
  byte-identical reads across two handles.

## 4. Refusal census — 44 of 47 arms directly driven

`checkBoundary` 8/8, `checkHeads` 3 code arms/5 cases, `checkSessionKind` 1/1,
`checkPersistenceVariant` 3/4, digest legs 3/3, scalar grammar 8/8,
`checkHeadBinding` 5/5, `checkRawHeadBinding` 4/5 (unknown-head mirror covered by
shape attestation + Capture-side arm; same-epoch foreign-lease Admit mirror
untested — P3, add a forged-Admit negative in a later leaf), Admit frame 5/5,
`Get` 3/3, install replay/conflict/disagree 3/4 (receipt-race reread is
concurrency-only). Three arms unreachable-by-construction with backstop proof:
variant `default` (kind pre-validated; R6 SURVIVED), `extractAdmitted` drift
guards (cannot fire on owner-attested bytes; fail closed), epoch-0 grammar
(head binding refuses first; R2b SURVIVED).

## 5. Mutation — PASS

- Producer harness rerun in isolation **3×: 7/7 narrowing KILLED + 1/1 harmless
  SURVIVED control, 0 NOT_APPLIED, 0 harness failures, exit 0** each run
  (`iso-mutants*/summary.json` + per-plant raw logs with return codes).
- Reviewer-owned narrowing mutants on producer-untouched arms (iso,
  per-plant raw logs in `iso-review-mutants/`):
  KILLED — R1 same-epoch foreign-lease neutered, R3 blob compare length-only
  (own same-length killer), R4 quiescence arm swallowed, R5 variant messages
  swapped, R2c `open_processes` bound `!=0 → >1`. Informative survivals, each
  proving a layered backstop still refuses end-to-end: R2 kind allowlist
  widened (variant gate backstop), R2b epoch-0 bound removed (head-binding
  backstop), R2d version-length removed (owner shape backstop), R6 `default`
  deleted (unreachable defense-in-depth). Control C2 comment-only SURVIVED applied.

## 6. Traceability / docs / hygiene

- P2 (non-blocking, Story-close): `section:5.4` registry gap text ("creation …
  not implemented") is stale while `doc.go`/`TRACEABILITY.md`/LOGBOOK describe
  capture as implemented. The projection digest is review-pinned
  (`traceability.go:43`), so the producer correctly did not self-mint; per §7.1
  precedent (LOGBOOK:831) the claim lands before Story close. **Requirement: a
  later leaf of STORY-260830-2rqigd updates the `section:5.4` binding and
  re-pins the digest before `story_final`.**
- P3 (non-blocking, Story-close): README has no `internal/sessckpt` section
  though it documents sibling packages with test commands. **Requirement: add
  the package section (with test commands) before `story_final`.**
- Hygiene: changed paths exactly the 10 candidate paths; `gofmt -l` clean,
  `git diff --check` clean, no `__pycache__`/`.pyc`, `task-board.config.json`
  unmodified; `tracecheck` exit 0; `cataloggen -check` exit 0; `go vet`
  (sessckpt/sessrepo/sessquery) exit 0; `go test
  ./internal/sessckpt ./internal/sessrepo ./internal/sessquery -count=1` all ok
  (sessquery 215s) — rerun by reviewer, exit 0.
- No CLI or capability advertised: concur with stated bound (library only).

## Evidence attached

`TASK-260830-14yo67_review-evidence-rev1.tar.gz`: reviewer probe test + runner,
probe log (6/6 PASS), package full log, vet log, producer-harness 3-run
summaries + per-plant logs, reviewer-mutant per-plant logs + summary.
Live merged checklist left for the accept transaction; scratch iso copies
deleted after packing.
