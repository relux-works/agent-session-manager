# TASK-260830-nxqqaw — review verdict, CR rev3

Verdict: **accepted** (`accept_cr`, revision 3)
Candidate tree de8741fcc0f1cda0bef3ab870dfad1c795e4d43b. The live worktree matches it (checked through a scratch GIT_INDEX_FILE). Base 0ca3e4c. Reviewer run RUN-260923-78f0db (claude-opus-5-5 low).

## Mandatory reruns (mine, on `git archive` copies under review-scratch)
- `go test ./...` on the exact tree: every package ok except `internal/specpin`, which fails in the archive copy only because the copy has no `.task-board`. Rerun in the live worktree: specpin ok (specpin.log). full.log.
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0. `gofmt -l internal`: empty.
- `-count=3` on merkleinventory, canonicaljson and provhost: ok (count3.log).
- `task-board.config.json` and `internal/traceability` show no diff against base, so there is no registry edit.
- Independent fixtures (fixtures.py, my own JCS+SHA-256): I recomputed the empty, singleton and branch roots, both branch child hashes, and all six MIXED-NS-1 roots. All 11 digests occur literally in the committed merkleinventory tests.
- Importer outcome grid (`go test -json`, keyed (package, test), over 25 packages: every importer of canonicaljson, provhost and merkleinventory, plus those packages themselves). Base has 21274 outcomes and the candidate 22336. There are 0 failures on either side and 0 moved outcomes. Two base subtests are removed: `canonicaljson TestUnsupportedSection10RecordSchemasValidateCommonEnvelopeBeforeRefusal/{tombstone,tombstone-ack}@1.0.0`. That is the named moved class: tombstone and tombstone-ack were fail-closed and are now admitted under the §10.7 shapes, as recorded in IMPORTER-OUTCOMES.md. 1064 outcomes are added (merkleinventory 695, canonicaljson 369). (grid-diff.txt)
- The shipped harness (`mutations.py`) ran twice. Both runs: 47 KILLED, 1 KILLED by behaviour with the source census passing, and the neutral control SURVIVED. The tree was restored (`diff -r` against a pristine archive is clean).

## Rev2 finding re-checked
`wireobject-response-member-names-case-folded` is fixed. The rev2 repro probe now refuses the miscased WireObject (repro-F1-rev3.log: `miscase: objects=0 err=invalid inventory object`). The mechanism is closed by construction:
- `strictJSON` accepts only map, RawMessage-array, string and Uint53 destinations. It runs `canonicaljson.Canonicalize` first, which refuses duplicates and malformed input.
- Every shape uses `exactMembers` on the map, and map keys are exact.
- The AST guard with alias control-plants exists (`TestWireDecoderASTGuardRejectsAliasedUnmarshalBindings`, `TestStrictJSONRejectsStructDestinations`).
- My plant `extra-members` (`!=` → `<` in exactMembers) is KILLED at both the server and the client entry.

## Findings
```json
[]
```

## Notes (non-blocking)
Each note below is a survived narrowing. In every case the candidate's own behaviour is correct, so these are test gaps, not defects. The Story final leaf or a follow-up should pin them.
- `fetch-dup-ids`: in FetchObjects, `ids[index-1] >= id` → `>` SURVIVED. The client-side duplicate-ID refusal is unpinned. CONFORMANCE-MATRIX row "objects.get request shape" lists `FetchObjects` next to a duplicate axis, but the killers only drive the server side. That row overstates coverage at the client entry.
- `b64-roundtrip`: in decodeOneWireObject, dropping `base64URL(dataBytes) != encodedData` SURVIVED. Go's base64 decoder skips `\r\n` even in Strict mode, so the round-trip check is the only thing refusing a newline-bearing `data`. That check is live and unpinned.
- `walk-quarantine`: `if quarantined {` → `&& namespace != "record"` SURVIVED the shipped suite. The arm is reachable: my probe `TestReviewWalkRefusesQuarantinedLocalIdentity` quarantines a local record and walks a peer that holds it. The candidate returns literal `integrity_failure`, and the plant turns that into the ID being reported as missing (walk-probe.log). This is the twin of the rev2 ObjectsGet quarantine note, and it is still unpinned in the committed suite.
- `walk-crossns`: `exists && namespaceOfID != namespace` narrowed to exempt manifest SURVIVED. This arm is reachable only through identity-level seeding or internal state.
- `mediatype` (drop the media_type check) was KILLED by `TestFetchObjectsRejectsEveryNonExactSuccessWireShape`. The applied comment control SURVIVED.
- My own plants: 2 of 6 killed. The survivors are listed above.

## Surface rows
| Row | Result |
|---|---|
| Trie construction / fixtures / rules 1-5 / label order | held (independent fixtures; harness rules killed 2/2) |
| inventory.roots / inventory.children request shape | held (extra-members killed) |
| Prefix axis 0..65 / not_found | held (harness prefix rows killed 2/2) |
| objects.get server namespace pairs, order, dups | held |
| objects.get client response shape (rev2 finding) | held (rev2 repro refused; extra-members, mediatype killed) |
| Membership table / MIXED-NS-N1 / four fail-closed schemas | held (harness 2/2) |
| Union exchange (timestamp winner, same-digest quarantine) | held (tombstone-timestamp-winner, same-id-quarantine killed 2/2) |
| Recursive walk integrity arms | held behaviourally (probe); unpinned in suite, see notes |
| Importer outcome grid | held (my own grid, above) |

Measured: 48 of 48 shipped narrowings killed ×2; my plants 2 of 6 killed (4 survivors = notes above).
