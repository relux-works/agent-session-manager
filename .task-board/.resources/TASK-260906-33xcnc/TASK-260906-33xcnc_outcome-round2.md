# TASK-260906-33xcnc round 2: review F1/F2/F3 closed — outcome

**Ready for review.** Candidate tree `ca0bf6b68b174192432c4c46296cc4610609cdb9`
(detached-index `write-tree` over checkpoint `cd8591d` plus exactly the 25
paths listed below; the real index was untouched; computed after the last
worktree write, the logbook entry, so the entry is inside the candidate).

This round is **test-only**: no production file changed. The frozen files
(`provhost/protocol.go`, `provhost/identity.go`, `provhost/probe.go`,
`canonicaljson/closed_shapes.go`) are byte-identical to `cd8591d`; the only
production-tree delta in this round is none. All three blocking findings are
answered by stronger gates and batteries, each proven by a plant that failed
before the fix and kills after it.

## F1 — the delegation gate is load-bearing now

The reviewer's plant (`_, _ = environ.CheckDigest(raw)` decoy plus a regrown
`ParseDigest(strings.ToLower(value))` copy in `sessadapter/decode.go`) stayed
green because `pinsCheckDelegation` required only an `environ` reference and
the tuple battery had no vector where owner and drifted copy diverge.

- Structural (`internal/environ/shape_census_test.go`): helpers must now be
  exactly `return environ.Twin(args)` as the body's only statement, with the
  twin name matching the wrapper name (`checkDigest` → `environ.CheckDigest`;
  a wrong-twin delegation preserves the token while changing behavior and
  fails). Wrappers must attribute the owner verdict to reused non-blank names
  (`members, fault := environ.DecodeStrictObject(data)` with both reused,
  exactly one call in the body) and carry no duplicate rule of their own.
  The checks are pure functions (`checkHelperDelegation`,
  `checkWrapperDelegation`) proven by 9 synthetic controls in
  `TestDelegationPinsAreLoadBearing` — every reporting control preserves the
  `environ` token, including the reviewer's decoy verbatim.
- Behavioral (`tuple_agreement_test.go`): new `uppercase fingerprint refused`
  row — `sha256:` plus uppercased hex is refused by `environ.DecodeTuple`
  and must be refused by `sessadapter.DecodeTuple`. The plant now fails
  `TestCheckHelpersDelegateToEnviron/sessadapter` AND
  `TestTupleAgreementAcrossFacades/uppercase_fingerprint_refused` in one run.
- Battery: `D1_decoycheckdigest` (narrowing, the plant verbatim) and
  `D2_decoywrapperfault` (arm-deletion, fault-ignoring wrapper) both report
  `suites=behavioral,census` — the harness executes the behavioral suite, not
  only the static checker, as required.

## F2 — the duplicate class is witnessed per member, not per witness

The frame battery had one duplicate row (key `v`); narrowing
`duplicate && key != K` survived 6/8 response keys and 7/10 owner keys, with a
reachable last-wins exploit through `provhost.DecodeResponse` (duplicate body
took the second body).

- Owner + facades (`frame_agreement_test.go`):
  `TestFrameAgreementRefusesDuplicateOfEveryDerivedMember` sweeps 30 keys
  derived from production — `tupleRequired`, `observationRequired`,
  provhost `responseMembers`, dirnode `scanRequestRequired`, plus the
  battery's generic `v` — through `environ.DecodeStrictObject` (exact
  detail+member), `sessadapter.DecodeTuple`, `dirnode.CheckScanRequest`, and
  `provhost.DecodeManifest`. New members in any derived set are automatically
  swept. Battery adds 8 owner narrowing mutants (`E_dupnarrow_*`), one per
  survived key plus `body`.
- Retained copy (`provhost/protocol_test.go`):
  `TestDecodeResponseRefusesDuplicateOfEveryMember` drives `DecodeResponse`
  with a doubled member for all 6 `responseMembers` (derived at runtime; a
  seventh member fails closed for lack of a literal) plus the swept foreign
  keys `v`, `capabilities`, `provider_id`, `cursor`. Battery adds 6 retained
  narrowing mutants (`P_dupnarrow_*`), one per survived key.
- Retention justification, now true: the provhost decoder and surrogate gate
  stay independent (frozen-file constraint, see below) and are pinned in both
  directions per member — every narrowing in either direction fails its own
  row while all others stay green. Convergence onto
  `environ.DecodeStrictObject` remains deferred, not abandoned; no
  leaf-owned production was touched, so the accepted-tree delta stays
  attributable per the brief's correction.
- Honest survivor in-session: the first `P_dupnarrow_provider_id` run
  SURVIVED green — the envelope sweeper lacked the foreign `provider_id`
  key, so no subtest existed to fail. The key was added, the mutant re-run,
  and it is KILLED. Reported as found-fixed-and-killed.

## F3 — denominator restated on the corrected basis

Re-derived from production, keeping the prior review's correction (the 11
`&Fault{}` exits are exits, not non-exits):

| component | count | basis |
| --- | ---: | --- |
| direct `refuse(` sites (`tuple.go` 10 + `observation.go` 21) | 31 | reproduced exactly |
| bool-false exits (`decode.go` 40 + `observation.go` 18 + `tuple.go` 1) | 59 | reproduced exactly |
| `&Fault{}` frame exits (`decode.go` 60, 63, 68, 72, 78, 82, 85, 89, 94, 96, 99) | 11 | previously excluded; now counted |
| **stated denominator** | **101** | **89 + 11 = 100 is the corrected pre-ladder-guard value; +1 is the new ladder empty-guard** |
| bool-`return <expr>` arms (`CheckEnvironmentID`, `CheckSemver`, `isNull`, `decode.go:239` forward) | 4 | named bound beyond the harness grammar |
| every exit kind | 105 | 101 + 4 |

The outcome's "two stories stale" wording is withdrawn: 100 was the corrected
value of the same inventory, and 90 was the uncorrected basis with the ladder
guard added. The tell holds and is now answered structurally: N1, M15, M15n,
M16, M17, M17n all weaken exits inside the previously excluded 11, and the
denominator now contains them. Coverage against the 101 basis: every one of
the 11 frame exits sits behind at least one battery mutant (surrogate: N1,
M18, T1; duplicate: M15, M15n, 8×E; trailing: M16; UTF-8: M17, M17n;
not-object positions: reached through every malformed-frame vector and the
D2 arm-slide), and all 31 refuse sites are exercised through production
entries with runtime site recording (refusal-site audit, derived == exercised
both ways, 39 expanded rules unique and exactly observed).

## AC coverage: 6 of 6 rows driven through production entries

| AC row | Verdict | Production call site / battery |
| --- | --- | --- |
| provhost decoder + surrogate gate | justified + per-member pinned | `provhost.DecodeResponse` via `TestDecodeResponseRefusesDuplicateOfEveryMember` (10 keys: 6 derived + 4 foreign) + 6 `P_dupnarrow_*`; `provhost.DecodeManifest` via frame battery + 30-key sweeper |
| canonicaljson frame gate | justified + pinned | `canonicaljson.Canonicalize` via frame battery `canonicalRefuse` verdicts (both directions reddened by the reviewer's own plants) |
| scalar third-spelling gate | justified + pinned | `scalar.DecodeClosedEnumJSON` vs `environ.DecodeStrictObject`, 15 vectors + C1/C2/C3 (C3 token-preserving) |
| sessadapter/dirnode per-helper copies | converged + load-bearing pinned | wrappers call `environ.*`; `TestCheckHelpersDelegateToEnviron` + `TestDelegatingWrappersCallEnviron` + `TestDelegationPinsAreLoadBearing` + D1/D2 (`suites=behavioral,census`); `sessadapter.DecodeTuple` uppercase vector |
| sibling rawUint53 trailing texture | hardened ×3 + ladder guard ×3 | `TestRawUint53RefusesTrailingData` (×3 pkgs) + `TestParseUint53LiteralRefusesNonDigits` + S1–S6 |
| F2 residue exits | attacked + re-stated on 101 | 16 new mutants (D1/D2 delegation decoys + 14 duplicate-class narrowings); denominator re-derived 31+59+11=101 (105 with bool-expr arms); callerless account unchanged from round 1 |

## Mutant battery: 70 killed / 71 applied on the 101-exit denominator

Harness `/tmp/mutbattery_leaf6/run.py` (attached), full log attached,
per-mutant table attached (`mutant-table.md`), results JSON attached.
Denominator derived from production at runtime (31 refuse + 59 bool-false +
11 frame-fault = 101). 49 narrowing (gate stays, admits exactly one rejected
member — including D1 plus 14 duplicate-class narrowings and the T1/C3
token-preserving raw-scan mutants), 17 arm-deletion (including D2), 3
census-only (behavioral green, census red), 1 audit-only (behavioral green,
alias audit red). `NOT_APPLIED` and `COMPILE_FAIL` are distinct rows, proven
by harness-validation mutants `H1`/`H2`. The one survivor, `R_digestnonstr`,
is predicted with a now property-pinned bound
(`TestDigestBridgeRefusesEmptyDownstream` closes the N2 rationale gap: if
`scalar.ParseDigest("")` ever admitted, that test reddens first; the bridge
is narrowed by `M7`/`M7n`). No unpredicted survivor, no `KILLED_OTHER`, no
billed `NOT_APPLIED`/`COMPILE_FAIL`.

## Gates (real exit codes, each run directly)

`go build ./...` 0; `go vet ./...` 0; `GOOS=windows go vet ./...` 0;
`GOOS=windows go build ./...` 0; `gofmt -l internal/` empty;
`go test ./... -count=1` 0 (all 23 packages ok);
`go test ./... -race -count=1` 0 (all 23 packages ok — full-repo race run
this round, no race bound); `-cover` environ 92.4 / sessadapter 82.0 /
dirnode 87.7 / provhost 86.0 / scalar 90.1 / invcore 70.2.

## Changed paths (exactly 25)

Modified (19): `LOGBOOK.md`, `internal/dirnode/bound_census_test.go`,
`internal/dirnode/decode.go`, `internal/dirnode/probe.go`,
`internal/dirnode/query.go`, `internal/environ/census_test.go`,
`internal/environ/decode.go`, `internal/environ/decode_unit_test.go`,
`internal/environ/frame_agreement_test.go`,
`internal/environ/shape_census_test.go`,
`internal/environ/tuple_agreement_test.go`,
`internal/provhost/opdecode.go`, `internal/provhost/protocol_test.go`,
`internal/sessadapter/bound_census_test.go`, `internal/sessadapter/decode.go`,
`internal/sessadapter/identity_census_test.go`, `internal/sessadapter/manifest.go`,
`internal/sessadapter/probe.go`, `internal/sessadapter/tuple.go`.
New (6): `internal/dirnode/rawuint53_test.go`,
`internal/environ/census_audit_test.go`,
`internal/environ/refusal_site_audit_test.go`,
`internal/environ/scalar_agreement_test.go`,
`internal/provhost/rawuint53_test.go`,
`internal/sessadapter/rawuint53_test.go`.
No untracked ungitignored scratch file in the worktree root (checked:
`git status` shows only the above) or the candidate tree (the 25 paths above
are exactly `git diff-tree --name-only -r cd8591d <tree>` in both
directions). Battery, tables, and backups live in `/tmp/mutbattery_leaf6/`.

## Bounds and notes

- Full-repo `-race` was run (23/23 ok); the round-1 race bound is withdrawn.
- N2 closed by pin, not rationale: `TestDigestBridgeRefusesEmptyDownstream`.
- N4 (`refuse` as a swappable var) is unchanged from round 1; still
  package-private, swapped only in `TestMain`, green under `-race`.
- This round changes no production code, so no resurrections are possible;
  the reviewer's held-green plants (scalar/provhost/canonicaljson both
  directions, refusal-site unexercised-site + alias, census var-binding +
  fresh-name + orphan) were not re-run here — the suites that caught them
  are green unchanged, and the battery re-proves the arms (C1–C3, A1, G1–G3
  all KILLED above).
