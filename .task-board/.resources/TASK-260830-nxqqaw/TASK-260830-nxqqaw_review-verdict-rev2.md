# TASK-260830-nxqqaw — review verdict, CR rev2

Verdict: **changes_requested** → `to-dev`
Candidate tree b15b65d18864a5329ac3ee497ae6190b58a294ab (live worktree verified via a scratch GIT_INDEX_FILE), base 0ca3e4c. Reviewer run RUN-260923-53c19e (claude-opus-5-5 low).

## Mandatory reruns (mine, on a `git archive` copy of the exact tree)
- `go test ./...`: all green except `internal/specpin` (the archive has no `.task-board`, an environmental failure). specpin rerun in the live worktree: ok. Log: fullsuite.log, specpin.log.
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0. `gofmt -l internal`: empty.
- `-count=3` on merkleinventory, canonicaljson and provhost: ok.
- `task-board.config.json` and `internal/traceability`: no diff against base. No registry edit.
- Independent fixtures (`fixtures.py`, my own JCS+SHA-256): the empty, singleton and branch roots, both branch child hashes, and all six MIXED-NS-1 roots and counts equal the spec literals. The tests assert the same literals.
- Importer outcome grid (mine, `go test -json`, keyed package::test, over the 14 canonicaljson importers, base vs candidate): base 17633 outcomes, candidate 18000, 0 failures on either side, 0 pass→fail. Two landed subtests were removed: `TestUnsupportedSection10RecordSchemasValidateCommonEnvelopeBeforeRefusal/{tombstone,tombstone-ack}@1.0.0`. 369 were added. The moved class is tombstone/tombstone-ack fail-closed → admitted under §10.7 shapes. It is named in IMPORTER-OUTCOMES.md and in the closed_shapes.go comment. The provhost allowlist rewrite (5→6 sites) is a scoped bound update, not a behaviour acceptance.
- The shipped harness ran twice: every row KILLED both times, the neutral control SURVIVED, and the tree was restored (diff -r clean).

## Rev1 findings, re-checked
- `children-prefix-non-string-admitted`: fixed. `requiredJSONString` refuses non-strings, and the exhaustive request-shape tests exist.
- The rev1 notes `emptychild2`, `crossns` and `fetchns` are now covered: the prefix axis test, the 5×5 hostile-source test, and the server namespace-pair test.

## Findings
```json
[
  {
    "id": "wireobject-response-member-names-case-folded",
    "row": "objects.get client response decoding (FetchObjects / decodeOneWireObject, §11.3 WireObject)",
    "invariant": "A WireObject is the closed shape {object_id, media_type, encoding, data} with exact member names. A response whose members are named otherwise is refused.",
    "mechanism": "serve.go decodeOneWireObject checks only len(inner)==4, then json.Unmarshal(rawObjects[0], &object). encoding/json matches keys case-insensitively, so {\"DATA\",\"Encoding\",\"Media_Type\",\"OBJECT_ID\"} fills every field and is admitted. This is the same class as rev1 (lenient encoding/json decode standing in for the closed wire shape) at the adjacent client-side site.",
    "reproductions": [
      {"test": "review_probe_test.go :: TestReviewFetchObjectsRefusesMiscasedWireObjectKeys (copy into internal/merkleinventory)",
       "command": "go test -count=1 -run TestReviewFetchObjectsRefusesMiscasedWireObjectKeys -v ./internal/merkleinventory",
       "expected_failure": "miscase: objects=1 err=<nil> — a hostile ObjectSource's miscased WireObject is admitted",
       "log": "repro-F1.log"}
    ],
    "severity": "bypass",
    "repeat-of": "rev1 children-prefix-non-string-admitted (same class: lenient decode of a closed wire shape; adjacent site)"
  }
]
```

## Notes (non-blocking)
- `fetchcbor` SURVIVED. The narrowing: in decodeOneWireObject, `object.Encoding != "json"` → `(!= "json" && != "cbor")`. No test pins that the client refuses an object labelled `cbor` when it asked for JSON only. The candidate itself refuses it correctly.
- `quarevent`/`quarall` SURVIVED. The narrowing, even a full arm-delete, removes the ObjectsGet quarantine refusal. That arm is unreachable in practice: quarantineLocked deletes the object, so the request falls through to `not_found`. It is dead defence; either delete it or pin its code.
- My plants upperA, len65, notfound63, dupid and rule5gt2 were all KILLED by the full package, and each named killer ALONE killed its plant 2 of 2 times. The applied comment control SURVIVED.
- My narrowings: 5 of 8 killed. The 3 survivors are the notes above, and they are test gaps rather than behaviour defects.

## Surface rows
| Row | Result |
|---|---|
| Trie construction / fixtures / rule 5 / label order | held |
| inventory.children / roots request shape | held (rev1 fix verified) |
| Prefix axis 0..65 / not_found | held |
| objects.get server namespace pairs, sorting, dups | held |
| objects.get client response shape | **broken** (wireobject-response-member-names-case-folded) |
| Membership table / MIXED-NS-N1 / fail-closed schemas | held (shipped harness kills 2/2) |
| Union exchange rules (timestamp, same-digest) | held per shipped harness rerun 2/2. I did not add an independent timestamp plant (budget). |
| Importer outcome grid | held (my own grid diff, above) |

## Rework scope (for the producer)
1. Decode WireObject (and the outer `objects` response) with exact member names. Check the exact key set on the canonical map, or decode per key from that map as the request path already does. Add an exhaustive test over the four member names: miscased, missing, and an extra member. Drive it through FetchObjects with a raw-body hostile source, and ship a narrowing that admits one case-folded name and is killed.
2. Pin the client's encoding check (`fetchcbor`), and either pin or remove the unreachable ObjectsGet quarantine arm.
