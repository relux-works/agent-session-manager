# TASK-260830-1snnef — republication note for CR revision 8

Run: RUN-260905-14f244 ([implementer] developer (muse)).
Purpose: republish the accepted round-6 candidate under the upgraded tooling's
complete producer binding. No code, test, README, or LOGBOOK byte was changed
by this run.

## Tree identity (read-only, alternate index — worktree untouched)

Recipe mirrors `changerequest.Take`: seed a disposable index from base tree
`20704201f91b3c33ece67af06e053176d0dd44c1`
(base OID `57afcc6dc5019672780baad393a0cef4873871b9`), `add -A`, restore the
`.task-board` prefix from base, `write-tree`.

- Before publish: `771f9c53fb708e5b401df371899c0d316bfd8e61` — equals
  `rev-000007.json` `candidate_tree_oid`. MATCH.
- Base tree recomputed `20704201…` — equals the record's `base_tree_oid`. MATCH.

Publication itself snapshots through the same alternate-index path, so it
cannot move this tree; the next reviewer run re-verifies with VerifyNoDrift.

Revision 7 carried `repository_delta=present` with 17 changed paths; the tree
is byte-identical, so revision 8 must report the same, not `empty`.

## Why revision 8 exists (tooling, not code)

Round-6 technical verdict is ACCEPT (277 mutant rows, ten batteries, zero
resurrections, gates green). `accept_cr` revision 7 was refused only because
revs 5–7 predate the binary built 2026-09-05T13:10:45Z, which requires
`ProducerRunID` + `ProducerRole` + `ProducerArchetype`. Rev 7 has
`producer_run_id=RUN-260905-f02929`, `producer_role=null`,
`producer_archetype=null`. This run's manifest carries `role=developer`,
`role_archetype=implementer`, so the completion-time publisher binds all three.

## AC coverage: 5 of 5 rows driven through production entry points

| AC row | Production call site | Named committed tests |
| --- | --- | --- |
| Lifecycle semantics | `CheckTransition` (`internal/terminalbackend/conformance.go:305`) | `TestCheckTransitionAdmitsEveryTableRow`, `TestCheckTransitionRefusesEveryIllegalSource`, `TestCheckTransitionNeverEntersFencingOrBootstrapStates`, `TestOnlyQuiesceInputEntersQuiescing` |
| Attach ownership neutrality | `CheckAttachResult` (`conformance.go:690`), `ParseAttachAuthorization` (`conformance.go:594`) | `TestCheckAttachResultRequiresTripleEquality`, `TestParseAttachAuthorizationRefusesNonNeutralPolicy`, `TestAttachAuthorizationCarriesNoLease` |
| ax pane enforcement | `CheckEntrypoint` (`conformance.go:703`) | `TestCheckEntrypointAdmitsOnlyAxPane`, `TestCheckEntrypointNamesItsArm` |
| Replication exclusions | `CheckReplicable` (`conformance.go:923`) | `TestReplicationClassificationIsClosed`, `TestReplicationMembersAreExactlyTheClosedTable`, `TestCheckReplicableRefusesEveryNonSafeClass` |
| Historical translation | `legacyForward` (`conformance.go:952`), `ProjectToLegacy` (`conformance.go:979`) | `TestHistoricalTranslationMapsOnlyTheImmutablePair`, `TestLegacyForwardIsExactlyTheImmutablePair`, `TestLegacyReverseProjectionExistsOnlyForThePair` |

## Narrowing mutants (item 17 — gate weakened, not deleted; named failing test)

- Surrogate escape bounds, `manifest.go:302/304/307`: five mutants, each
  narrowing one bound to admit exactly a slice of the rejected class —
  low upper `0xdfff→0xdc00`, high upper `0xdbff→0xd900`,
  high lower `0xd800→0xd900`, pair-continuation upper/lower. Named failing
  test `TestSurrogateGateAgreesWithCanonicalJSON`; refusal counts 1023 / 768 /
  1543 / 1024 / 257 match the exact width of each hole
  (`TASK-260830-1snnef_review6-verdict.md` G-A).
- Binding-digest shape arm, `terminalbackend.go:587`: arm neutralised via
  `err != nil && false` (arm count and literals intact, inventory uninflatabale);
  exactly `TestCheckProviderDescriptorBindingDigestMismatch/malformed_digest_on_both_sides`
  fails (G-B F2).
- Generation UTF-8 conjunct, `terminalbackend.go:606`: dropped conjunct admits
  `"gen\xff"`; only `TestCheckProviderDescriptorGenerationBounds/invalid_utf-8`
  fails (G-B F1).
- Replication table: adding one `safe_evidence` member survives the
  one-directional test; the bidirectional live-map pin
  (`TestReplicationMembersAreExactlyTheClosedTable`) kills it.
- Legacy widening L3 (third escape name → `vendor.screen`): GREEN before the
  pin, RED 5/5 through `TestLegacyForwardIsExactlyTheImmutablePair` after.

## Token-preserving attacks on source-inspecting gates (item 18 — token kept, behavior changed, behavioral suite executed)

- Plain-error attribution gate: `newPlain = errors.New` alias called from a
  `FuncDecl` (V5), import-aliased `errs.New`, package-level `var` func literal
  (A1/W5b/W16), constructor through a func value (A2), `fmt.Errorf` variant
  (W9b) — the searched-for token shape is preserved while the behavior the gate
  must catch changes; harnesses execute the full package suite plus vet/gofmt,
  and kills are attributed to behavioral tests, not the inventory alone.
- `isWTF8SurrogateAt` punch-out (`raw[index+1] != 0xa1`, admitting 64 of 2048
  encodings): survives the package — recorded as the single stated bound, with
  the production predicate separately swept over all 2048 encodings
  (missed = 0/2048) and the branch proven production-dead (disabling it moves
  only the white-box pin; leaf2 M81 killing the covering `utf8.Valid` arm).
- Surrogate agreement: disabling `isWTF8SurrogateAt` reddens
  `TestSurrogateGateAgreesWithCanonicalJSON` with 6 agreement failures.

Full per-mutant tables: `TASK-260830-1snnef_review6-mutation-results.json`
(277 rows). Survivor bounds: C45 (disclosed unbounded-domain), S10
(equivalent), B5 (intended-GREEN ordering control), J1 (receiver-ident
match on import-aliased errors).
