# TASK-260830-17suox reviewer gate logs (candidate tree 60ce1269)

## go build ./...
exit 0, no output.

## go vet ./...
exit 0, no output.

## go test ./... -count=1
```
ok  	github.com/relux-works/agent-session-manager/internal/canonicaljson	7.458s
ok  	github.com/relux-works/agent-session-manager/internal/catalog	0.368s
ok  	github.com/relux-works/agent-session-manager/internal/catalog/cmd/cataloggen	1.489s
ok  	github.com/relux-works/agent-session-manager/internal/cataloggen	2.019s
ok  	github.com/relux-works/agent-session-manager/internal/config	2.273s
ok  	github.com/relux-works/agent-session-manager/internal/scalar	2.995s
ok  	github.com/relux-works/agent-session-manager/internal/specpin	3.366s
ok  	github.com/relux-works/agent-session-manager/internal/traceability	1.282s
ok  	github.com/relux-works/agent-session-manager/internal/traceability/cmd/tracecheck	22.657s
exit: 0
```

## go test ./internal/config -cover -count=1
```
ok  	github.com/relux-works/agent-session-manager/internal/config	0.507s	coverage: 93.2% of statements
```

## go run ./internal/traceability/cmd/tracecheck
```
traceability ok: contracts=60 normative_sections=36 acceptance_cases=32 fixtures=30 compatibility_contracts=55 assigned_scopes=0
exit=0
```

## go run ./internal/traceability/cmd/tracecheck -section 3.2 -section 6.1 -section 6.2 -section 6.3 -section 6.4 -section 6.5 -section 17.1 -section 17.2
```
traceability ok: contracts=60 normative_sections=36 acceptance_cases=32 fixtures=30 compatibility_contracts=55 assigned_scopes=8
exit=0
```

## Specification authority check
```
$ git -C ~/Developer/ReluxWorks/agent-session-manager-spec rev-parse HEAD
28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c
$ shasum -a 256 SPEC.md
562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a  SPEC.md
```
Equal to `specpin.CommitV050` and `specpin.DocumentSHA256`.

## Candidate integrity check (before and after review)
```
$ GIT_INDEX_FILE=<scratch> git read-tree HEAD && git add -A && git write-tree
60ce126907dc3bd83b2741b10c1deb834410c4d0
```
Equal to the Change Request candidate tree OID. The reviewer modified no
repository file; all probes ran against copies under /tmp.

## Confirmed defect reproduction (empty extensions / settings round trip)
```
BUG: installation empty extensions round-trip refused: configuration document rejected at directory_installations[0] required member
BUG: profile empty extensions round-trip refused:      configuration document rejected at directory_enrichment_profiles[0] required member
BUG: EncodeCurrent output refused by Decode:           configuration document rejected at directory_peer_disclosure[0] required member
BUG: empty backend_config settings round-trip refused: configuration document rejected at terminal.backend_config[0] required member
```

Isolated go-toml v2.4.3 behavior (standalone program, no repository code):
```
marshal   map[string]any{}          -> "[entries.extensions]"  (bare table header)
unmarshal "[entries.extensions]"    -> nil map
unmarshal "extensions = {}"         -> non-nil empty map
unmarshal (member absent)           -> nil map
```

## Refusal-clause instrumentation
`configError` instrumented to record every clause string; package suite run.
98 of 101 distinct clauses reached. Never reached:
```
mesh.peers[N].workspace_roots[N].logical_root
mesh.peers[N].workspace_roots duplicate logical_root
mesh.peers[N].workspace_roots[N].path
```

## Mutation results (20 applied, 9 survived)
SURVIVED: M-A peer platform binding; M-B peer workspace-root member checks;
M-E scan_root_authority_ids minimum; M-3 legacy conpty mapping; M-4 legacy
Windows backend default; M-5 v3 Windows backend default; M-6 logical_root
grammar; M-7 extension key minimum length; M-8 extension int64 safe-integer bound.

KILLED: M-9 ax.conpty-on-non-Windows refusal; M-10 graceful stop upper bound;
M-11 ssh_args count bound; M-12 environment_id character bound; M-13 disabled
external-trust registration; plus delete-only mutants covered by the exact-clause
assertions in refusal_test.go.

## Platform-lane behavior probes against unmodified candidate (via Decode)
```
OK v1 windows default backend: accepted backend="ax.conpty"
OK v2 windows default backend: accepted backend="ax.conpty"
OK v1 windows explicit conpty: accepted backend="ax.conpty"
OK v1 windows explicit tmux refused: rejected at terminal.backend_id unsupported platform
OK v3 windows default backend: accepted backend="ax.conpty"
OK v3 windows native workspace path: accepted backend="ax.conpty"
OK v3 windows posix workspace path refused: rejected at workspace_roots[0].path
OK v3 wsl2 posix path: accepted backend="ax.tmux"
OK v1 wsl2 default backend: accepted backend="ax.tmux"
OK v1 linux default backend: accepted backend="ax.tmux"
OK windows peer POSIX workspace path refused
OK windows peer native workspace path accepted
OK peer duplicate logical_root refused
OK empty scan_root_authority_ids refused
OK uppercase logical_root refused
OK leading-digit logical_root refused
```
Production behavior is correct on all lanes; the lanes are simply unpinned.

## Closed-shape probes (all refused)
v3 terminal unknown nested key; v3 terminal legacy `backend` key; v1 terminal
`backend_id`; v1 `[directory]` table; v2 terminal `backend_id`; mesh unknown key;
peer unknown key; workspace-root unknown key; `profiles.yolo` unknown key;
unknown profile besides `yolo`; directory unknown key; external_trust unknown key;
directory_installations unknown key.
