# BUG-260902-4ynj93 — reapply evidence

Accepted revision 3 of `BUG-260902-4ajzyz` reapplied onto trunk checkpoint
`139a691` on `task-board/story/STORY-260902-9s0i85`, one signed commit `7c9604d`.

## Byte-identity against the accepted tree

The accepted patch applied at exact context (`git apply --exclude=LOGBOOK.md`,
exit 0 — no `--3way`, no fuzz). Post-image blob hashes compared against the
patch's own `index a..b` lines:

| File | Accepted post-image | This tree | Verdict |
| --- | --- | --- | --- |
| `internal/config/endpoint.go` | `d5e59d9` | `d5e59d9` | byte-identical |
| `internal/config/endpoint_admission_test.go` | `56c9368` | `56c9368` | byte-identical |
| `internal/config/schema_test.go` | `e37b9cb` | `e37b9cb` | byte-identical |
| `internal/config/ssh_admission_test.go` | `504894f` | `504894f` | byte-identical |
| `internal/config/validation.go` | `e7505f9` | `e7505f9` | byte-identical |
| `LOGBOOK.md` | `59f5aff` | differs | permitted overlap resolution |
| `README.md` | `4c86919` | differs | **reported deviation, see below** |

### LOGBOOK.md — the permitted overlap

The accepted patch prepends its entry directly under `## 2026-09-02`, where
`### 1353` was then the newest entry. Trunk's day block has since gained five
entries (1810, 2240, 2235, 2110, 1730). The accepted `### 1620` entry is placed
in descending-timestamp order between `1730` and `1400`, verbatim; no accepted
sentence is rewritten. Two lines were appended to it — `- EVIDENCE (reapply):`
and `- REAPPLY:` — recording this reapply and the mutant re-proof below, in the
same shape the `1353` entry already uses on trunk.

### README.md — the one reported deviation

Not absorbed, reported. The trunk base blob is `3a0d0f7`; the accepted patch was
cut against `014f8f5`. The post-image blob therefore cannot match by
construction. The delta this leaf introduces is exactly the accepted +25 lines
and nothing else (`git diff --stat README.md` → `25 +++`, `1 file changed, 25
insertions(+)`, zero deletions); the whole-file difference is pre-existing trunk
content elsewhere in the file.

## Mutants re-proved at THIS tree

The acceptance criteria single out two clauses. Both were re-run here rather than
carried on the previous cycle's word.

| Mutant | Result | Reddens |
| --- | --- | --- |
| `admitEndpointPort`: `len(port) > 5` → `> 6` | KILLED (exit 1) | exactly `…BoundIsPinnedAtItsEdge/port_length_upper` and `…DeclaredMeshEndpointGrammar/port_at_length_bound` |
| `sshPermittedFlags` gains `'N'`, a letter no named assertion covers | KILLED (exit 1) | `TestLoadAdmitsExactlyTheDeclaredPermittedSSHShortOptions` alone — `ssh_admission_test.go:364: permitted flag -N has no declared sample` |

M1 confirms the port bound pins to its limit, not to a range: the seven-byte
`port over length bound` case survives the narrowing, and only the adjacent
six-byte `peer.example:000022` case catches it. M2 confirms the key-set pin
reddens for a letter outside the named capability-widening assertions (`-A -E -L
-R -D -W -w -S -O`), which is the derived-walk asymmetry it exists to close — the
derived test skips whatever production permits, so it stays green under M2.

Both mutants were reverted and the sources verified byte-identical by blob hash
afterwards (`sshargs.go` `a0e7b63` = `HEAD:internal/config/sshargs.go`;
`endpoint.go` `d5e59d9` = accepted). Logs: `mutant-m1-portbound.log`,
`mutant-m2-flagN.log`.

## Gates — real exit codes, each run as a standalone process

| Command | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` | 0 (empty listing) |
| `go test ./... -count=1` | 0 |
| `go test ./... -cover` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 |
| `go run ./internal/traceability/cmd/tracecheck -section 6.1 … -section 6.5` | 0 |
| `cataloggen -metadata … -contracts … -output … -check` | 0 |

`tracecheck` global: `contracts=60 normative_sections=36 acceptance_cases=38
fixtures=30 compatibility_contracts=55 assigned_scopes=0`; scoped 6.1-6.5 reports
`assigned_scopes=5`. No generated-file drift — `git status --short` after the
gates listed only the seven files of the reviewed scope.

One earlier `cataloggen -check` invocation exited 1 with `-metadata, -contracts,
and -output are required`: that was my wrong invocation, not a gate failure. The
canonical form from `README.md:869` then exited 0.

## Coverage

| Package | Trunk `139a691` | This tree | Delta |
| --- | ---: | ---: | ---: |
| `internal/config` | 94.4% | 94.7% | +0.3pp |

Base measured from a `git archive HEAD` copy under
`.temp/BUG-260902-4ynj93/base`, exit 0. No package regressed.
