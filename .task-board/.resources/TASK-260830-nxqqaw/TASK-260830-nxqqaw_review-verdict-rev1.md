# TASK-260830-nxqqaw — review verdict, CR rev1

Verdict: **changes_requested** → `to-dev`
Candidate tree 89227e7c60f0390c181c8ebb901ccb66ebc0bb26, base 360c8bd. Reviewer run: claude-opus-5-5 low.

## Reruns (mine, on a `git archive` copy of the exact tree)
- `go test ./...`: green except `internal/specpin` (archive has no `.task-board`, so the failure is environmental). specpin rerun in the live worktree: ok.
- `GOOS=windows GOARCH=amd64 go vet ./...`: exit 0. `gofmt -l ./internal`: empty.
- `-count=3` on merkleinventory, canonicaljson, provhost: ok.
- `task-board.config.json`: byte-identical to base. No registry edit.
- Independent fixtures (own Python JCS+SHA-256, `fixtures.py`): empty, singleton and branch roots, both branch child hashes, and all six MIXED-NS-1 roots/counts equal the spec's literals exactly.
- Shipped harness `mutations.py --output`: every row KILLED, neutral control SURVIVED, tree restored. I did NOT rerun each KILLED twice (budget).

## Findings
```json
[
  {
    "id": "children-prefix-non-string-admitted",
    "row": "inventory.children request shape (§11.3 prefix 0..64 lowercase hex)",
    "invariant": "prefix must be a JSON string of 0..64 lowercase hex characters. Anything else is refused.",
    "mechanism": "serve.go childrenBody uses rawString(object[\"prefix\"]) (index.go:251), which returns \"\" for any non-string JSON value. validPrefix(\"\") passes, so a non-string prefix serves the ROOT node. Same pattern for namespace, but there \"\" fails validNamespace.",
    "reproductions": [
      {"test": "review_repro_test.go.txt :: TestReviewChildrenPrefixTypeConfusion (copy into internal/merkleinventory as *_test.go)",
       "command": "go test ./internal/merkleinventory -run TestReviewChildrenPrefixTypeConfusion -v",
       "expected_failure": "Dispatch admits prefix 0 / null / false / [] / {} and returns the root body (count 1, node_hash sha256:ab4c73cb…). rpcwire.EncodeRequest does not refuse them.",
       "log": "repro-F1.log"}
    ],
    "severity": "bypass",
    "repeat-of": "none"
  }
]
```

## Notes: surviving narrowings. Production behavior holds, but no test pins it. Required in the rework per the DoD rule that every gate has a killed narrowing.
- `emptychild2`: `serve.go` not_found arm narrowed to `len(prefix) != 2` SURVIVED. not_found is pinned only at prefix length 1 ("f"). My test `TestReviewNotFoundAtEveryPrefixLength` (lengths 1, 2, 63, 64) passes on the candidate.
- `crossns`: `ObjectsGet` cross-namespace arm narrowed with `&& expectedNamespace != "manifest"` SURVIVED. It is pinned only for requested namespace `event`. Under the plant a record requested as manifest falls through to `not_found`: it is still refused, but under a different literal code.
- `fetchns`: `decodeOneWireObject` dropping `membership.Namespace != expectedNamespace` SURVIVED. The client-side §11.4 rule ("objects.get MUST … reject an ID whose schema maps to another requested namespace") is unpinned: every test source is an honest Index. My hostile-source test `TestReviewFetchObjectsRejectsForeignNamespaceFromHostilePeer` passes on the candidate.
- rpcwire has no children/objects.get validators, so the local validators are not a fork of rpcwire. Owner note only.
- My own plants rule5 (`>2`), labelorder, upperhex and len65 were all KILLED. Control SURVIVED.

## Surface rows
| Row | Result |
|---|---|
| Trie construction / fixtures / rule 5 / label order | held |
| inventory.children request shape | **broken** (children-prefix-non-string-admitted) |
| inventory.roots shape (rpcwire) | held (delegated to rpcwire.DecodeRequest) |
| not_found non-materialized prefix | held (test gap: note emptychild2) |
| objects.get namespace membership (server + client) | held (test gaps: notes crossns, fetchns) |
| Membership table / MIXED-NS-N1 / unsupported schemas fail-closed | held (shipped harness kills) |
| Union exchange rules | held per shipped harness; no independent timestamp plant (budget) |
| Importer outcome grid | not-attacked: budget. Full suite is green on candidate; no grid diff was made. |

Measured: 4/4 of my narrowings killed plus 3 survivors, i.e. 4 of 7 of my plants killed.

## Rework scope (for the producer)
1. Decode the children body strictly: namespace and prefix must be JSON strings, refused otherwise. Add a committed test driving `Dispatch` with non-string prefix values, plus a narrowing (for example, accept `null`) that it kills.
2. Pin not_found at prefix lengths 2, 63 and 64. Pin cross-namespace refusal for every requested namespace (the namespace axis). Pin FetchObjects' foreign-namespace refusal with a hostile ObjectSource. Add the three survivor plants above to mutations.py.
