REWORK for TASK-260830-nxqqaw, CR rev2 → rev3. Read `TASK-260830-nxqqaw_review-verdict-rev2.md` first; its findings JSON outranks this brief. The base is trunk `0ca3e4c`. If trunk moves before your handoff, run `task-board worktree refresh-candidate TASK-260830-nxqqaw` first.

SECOND ROUND OF ONE CLASS: lenient decoding standing in for a closed wire shape. In rev1 the request path turned a non-string `prefix` into `""`. In rev2 the client-side `WireObject` decode relies on `encoding/json`, which matches member names case-insensitively, so `{"DATA","OBJECT_ID",...}` is admitted. The reviewer's vector shows the invariant is open. It is not the work item. Fixing only `decodeOneWireObject` would leave the next decode site for the next round.

THE INVARIANT: every closed wire shape this leaf reads is decoded strictly, BY CONSTRUCTION. That covers requests and responses, on the server and the client, for `inventory.roots`, `inventory.children` and `objects.get`, plus the `WireObject` items and the outer envelopes.
- Member names are matched exactly, byte for byte and case-sensitively.
- The key set is exact: no missing, extra or duplicated members.
- Each value is typed exactly (JSON type and bounds).
- Nothing is ever defaulted from a zero value.

DELIVER:
1. **Decoder census first.** In the conformance matrix, list EVERY site in the package's production code that decodes wire bytes or a canonical map into a value. Grep `json.Unmarshal`, `json.NewDecoder`, `Decode(`, map accessors and `raw*` helpers. For each site give file:line, the shape it decodes, and which strict path it uses. A site without a row is the next finding.
2. **One strict decoder.** Every site in the census uses a single strict decoding path, either the existing request-path helpers (`requiredJSONString` and friends, generalised) or `internal/canonicaljson` if it already offers exact-member decoding. Reuse before writing. No wire path may `json.Unmarshal` into a struct.
3. **A guard test.** Add a committed AST or source test over the package's production files that fails if any wire-decode path calls `json.Unmarshal`/`json.NewDecoder` into a struct type, or reintroduces a lenient accessor. Control-plant it: show the guard reddens when you plant such a call. Watch for aliases: an import alias or a `var` binding of the decoder must not bypass it.
4. **Exhaustive shape tests per shape.** For EVERY shape in the census, cover every member with each of: miscased name, missing, extra member, duplicated member, and wrong JSON type (6 types). Drive the client shapes through `FetchObjects` with a raw-body hostile `ObjectSource`, and the server shapes through `Dispatch`. Assert the literal refusal code. Ship a narrowing that re-admits one case-folded name at a client site and one at a server site, and show each killed ALONE.
5. **Rev2 notes.** Pin the client encoding check (`fetchcbor`: an object labelled `cbor` when only JSON was requested is refused). Then either delete the unreachable `ObjectsGet` quarantine arm with a stated reason, or pin its code with a test that reaches it. Do not leave dead defence unpinned.
6. Rerun the importer grid and name the moved classes (tombstone/tombstone-ack admission under §10.7 stays named). Then run `GOOS=windows GOARCH=amd64 go vet ./...` and the full configured suite. Update the census, matrix, TRACEABILITY, results and evidence tar (under 1 MiB). Keep the checklist current, then run `task-board handoff TASK-260830-nxqqaw --role developer`.

SCRATCH RULE: no scratch under /tmp. LIVE-INDEX RULE applies. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max. Reviewer: claude-opus-5-5 low.
