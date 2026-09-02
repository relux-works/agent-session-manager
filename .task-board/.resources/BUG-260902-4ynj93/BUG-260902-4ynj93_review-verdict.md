# BUG-260902-4ynj93 — review verdict: ACCEPTED (CR rev 1)

Candidate `6f33458` on `task-board/story/STORY-260902-9s0i85`, one signed commit
`7c9604d` past base `139a691`. Reviewed independently of the producer's evidence:
the accepted patch was re-fetched from the board, the accepted tree reconstructed
from it, and every claim re-measured at this tree.

## The acceptance criterion is byte-identity, and it holds

The accepted tree was rebuilt in a scratch directory outside the repository:
`git archive 139a691 | tar -x` then `patch -p1 < BUG-260902-4ynj93_accepted-rev3.patch`,
which applied with no rejects. `cmp` against the candidate:

| File | cmp vs reconstructed accepted tree | Blob at HEAD | Accepted post-image |
| --- | --- | --- | --- |
| `internal/config/endpoint.go` | identical | `d5e59d9` | `d5e59d9` |
| `internal/config/endpoint_admission_test.go` | identical | `56c9368` | `56c9368` |
| `internal/config/schema_test.go` | identical | `e37b9cb` | `e37b9cb` |
| `internal/config/ssh_admission_test.go` | identical | `504894f` | `504894f` |
| `internal/config/validation.go` | identical | `e7505f9` | `e7505f9` |
| `README.md` | identical | `426fd55` | `4c86919` (base differs) |
| `LOGBOOK.md` | differs | — | permitted overlap |

`README.md` is the deviation the producer reported, and the report is accurate
rather than a hedge. Its trunk base blob is `3a0d0f7` against the patch's
`014f8f5`, so the post-image hash cannot match by construction — but the delta
this leaf introduces is exactly the accepted hunk. Verified by diffing the
candidate's added lines (`git diff 139a691 HEAD -- README.md`) against the
accepted patch's own `+` lines: identical, 25 insertions, **zero deletions**. No
pre-existing trunk content was absorbed or displaced.

`LOGBOOK.md`: the accepted `### 1620` entry is placed between `### 1730` and
`### 1400`, verbatim, with two appended lines (`EVIDENCE (reapply)`, `REAPPLY`).
Heading order of the day block is otherwise unchanged from `139a691`; no existing
entry was rewritten, moved or dropped.

## The two AC-named clauses were attacked, not read

Mutants applied to the real sources, run through `go test ./internal/config/...`,
reverted, and the sources re-hashed to the accepted blobs afterwards.

| Mutant | Result | Reddens |
| --- | --- | --- |
| `admitEndpointPort` `len(port) > 5` → `> 6` | KILLED | exactly `…BoundIsPinnedAtItsEdge/port_length_upper` and `…DeclaredMeshEndpointGrammar/port_at_length_bound` |
| `sshPermittedFlags` gains `'N'` — a letter no named assertion covers | KILLED | `TestLoadAdmitsExactlyTheDeclaredPermittedSSHShortOptions` **alone** |

The first confirms the bound pins to its limit, not a range: the seven-byte
`port over length bound` case survives the narrowing and only the adjacent
six-byte `peer.example:000022` catches it. The second confirms the key-set pin is
what closes the derived-walk asymmetry — `-N` is caught by nothing else in the
package, because the derived walk skips whatever production permits.

## Eleven further mutants, all killed

| Mutant | Result | Reddens |
| --- | --- | --- |
| leading-hyphen clause carved out for `@` (the rev1 review's live bypass) | KILLED | all 7 `…OptionShapedEndpointsTheGrammarWouldAdmit` cases |
| leading-hyphen clause deleted | KILLED | both option-shaped suites, 15 leaves |
| whitespace clause deleted | KILLED | all 10 `…CarryingWhitespace` cases |
| host widened to anything `net.ParseIP` accepts | KILLED | `bare_IPv6_and_port`, `bare_IPv6_loopback` |
| bracketed host drops `To4() == nil` (admits `[192.0.2.10]`) | KILLED | `bracketed_IPv4` |
| user split at first `@` instead of last | KILLED | `second_at_sign` |
| port split at first `:` instead of last | KILLED | `bare_IPv6`, `bare_IPv6_and_port`, `bare_IPv6_loopback`, `colon_inside_host` |
| user upper bound 64 → 65 | KILLED | `user_length_upper`, `user_over_bound` |
| host label upper bound 63 → 64 | KILLED | `host_label_length_upper`, `label_over_bound` |
| port value upper 65535 → 65536 | KILLED | `port_value_upper`, `port_over_bound` |
| label edge-hyphen clause deleted | KILLED | `label_starts_with_dash`, `label_ends_with_dash` |
| `sshPermittedValueFlags` gains `-L` | KILLED | key-set pin + `…CapabilityWideningFlags/local_forward` |
| `sshPermittedFlags` gains `-A` | KILLED | key-set pin + `…CapabilityWideningFlags/agent_forwarding` |

Every kill reddens only the cases that own the clause. No mutant survived, and
none reddened the whole suite indiscriminately.

## Independent hostile probe, 92 shapes

A throwaway in-package probe drove 92 hostile endpoints through the production
`loadConfigDocument` entry: the reported option shapes, six Unicode hyphen
lookalikes (U+2010/2011/2012/2013/2014/2212/FF0D/2796/05CA/02D7), five invisible
prefixes ahead of a real hyphen (ZWSP, BOM, LRM, SHY, word-joiner, ZWJ), NUL/ESC/VT/FF,
five Unicode space separators, four non-ASCII or fullwidth digit ports, the port
and label bound neighbours, bare and bracketed IPv6, zone identifiers, shell
metacharacters, URL and path shapes.

**2 of 92 admitted, both correctly:** `peer.example:65535` (the declared upper
bound, included deliberately as a control) and `xn--bcher-kva.example` (punycode,
a legal LDH DNS name). Nothing option-shaped, whitespace-carrying or
homoglyph-prefixed got through. The six ordinary controls all still load, so the
gate is not vacuously strict.

## Production reachability, not a helper called from nowhere

`config.Load` / `LoadOS` → `Decode` → `translateV1` / `translateV2` / `translateV3`
→ `validateConfiguration` (`validation.go:53`, `:77`, `:101`) → `validateMesh`
(`:391`) → `admitMeshEndpoint` (`validation.go:452`). The gate sits on the common
validator, so all three schema readers reach it; `writer.go:16` re-validates on
the write path. `grep -rn '\.Endpoint'` shows a single production ingress at
`validation.go:186` and no path that populates `Mesh.Peers[].Endpoint` around the
validator.

## Gates, re-run at this tree

| Command | Exit |
| --- | ---: |
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` | 0 (empty) |
| `go test ./... -count=1` | 0 (all 9 packages ok) |
| `go test ./... -cover` | 0 |
| `go run ./internal/traceability/cmd/tracecheck` | 0 — `contracts=60 normative_sections=36 acceptance_cases=38 fixtures=30 compatibility_contracts=55 assigned_scopes=0` |
| `tracecheck -section 6.1 … 6.5` | 0 — `assigned_scopes=5` |
| `cataloggen … -check` | 0, no generated-file drift |

Coverage measured independently, base from a `git archive 139a691` copy:
`internal/config` **94.4% → 94.7%**, no package regressed. Working tree clean
after every mutant and probe; `git status --short` empty and all three mutated
sources re-hash to their accepted blobs.

`git verify-commit 7c9604d`: Good signature, `Ivan Oparin <oparin@me.com>`,
ECDSA `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`. Exactly one commit
past the checkpoint.

## Verdict

**ACCEPTED.** The reapply is byte-exact outside `LOGBOOK.md`, the one deviation
is reported and is provably a base difference rather than an absorbed change,
both clauses the acceptance criteria single out redden under their own narrowing
at this tree, and thirteen mutants plus a 92-shape probe found no surviving
weakening. No `commit_ack` supplied — the commit-owning mover makes the `done`
transition.
