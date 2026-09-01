# TASK-260830-1pbx0c review verdict — CR revision 5

## Verdict

**Changes requested.** Route the task to `to-dev` for implementation rework.
This is ordinary, recoverable validation work and is not a Stop-The-Line
boundary.

Reviewed candidate:

- Change Request: `CR-TASK-260830-1pbx0c-5`, revision 5
- Base: `c9e5290b1506275f5417b26070fad0391a09c50a`
- Candidate tree: `286e728864d78c5decee38ced2c53893ad94ed4c`
- Repository delta: present, 12 paths
- Every changed worktree file matched the candidate-tree blob byte-for-byte.
- The supplied carry-forward archive matched its declared SHA-256 and all seven
  `internal/scalar` files matched the candidate byte-for-byte.

## Required finding

### F1 — Windows `absolute-path` accepts wildcard and reserved device components

Severity: high for a foundational routing scalar.

`ParseAbsolutePath` delegates native Windows values to
`validateWindowsAbsolutePath`, but `invalidWindowsSegment` at
`internal/scalar/path.go:160` rejects only empty/dot/parent components, colons,
and trailing dot/space. The production gate admits Win32-invalid wildcard
characters and reserved device aliases:

```text
windows absolute-path "C:\\unsafe\\*.json" accepted=true error=<nil>
windows absolute-path "C:\\unsafe\\CON" accepted=true error=<nil>
windows absolute-path "C:\\unsafe\\NUL.txt" accepted=true error=<nil>
windows absolute-path "\\\\server\\share\\COM1" accepted=true error=<nil>
```

This defeats the intended per-segment device-syntax refusal: `CON`, `NUL`, and
`COM1` resolve through the Win32 device namespace even when used below an
ordinary directory, while `*` is a wildcard rather than a file-name component.
The pinned SPEC at commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`
(verified document SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`)
defines `absolute-path` as absolute and lexically normalized for its containing
platform. Microsoft’s platform authority lists `< > : " / \\ | ? *`, control
characters 1–31, and the `CON`/`PRN`/`AUX`/`NUL`/`COM1..9`/`LPT1..9` families
(including extensions) as forbidden Win32 file or directory names:
https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file

Negative-evidence shape: **bypass path around the check**. A guard exists and
is directly exercised, but alternate Win32 device/wildcard components reach the
accepted state.

Required rework:

1. Reject every Win32-reserved component character, ASCII control character
   1–31, and reserved device name (case-insensitive, including names followed by
   an extension) in drive-qualified and UNC paths.
2. Preserve the already-correct drive/UNC, dot/parent, alternate-stream,
   trailing-dot/space, length, NUL, and platform-context refusals.
3. Add table-driven negative tests through the production entry points:
   `ParseAbsolutePath`, `DecodeAbsolutePathJSON`, and the publication boundary
   (`json.Marshal` on a validated value). Include drive and UNC forms and prove
   a narrowed mutant of the component gate fails the named test.

## Evidence that passed

- `go test ./internal/scalar -count=1 -v`: pass
- `go test ./internal/scalar -cover -count=1`: pass, 89.0%
- assigned `tracecheck` for Sections 1.6 and 10.1–10.4: pass,
  `assigned_scopes=5`
- `go test ./... -v -count=1`: pass; it includes all five owner-declaration
  rename mutants and the existing narrowed-section/acceptance-owner attacks
- `go test ./... -cover -count=1`: pass; scalar 89.0%
- `go test -race ./internal/scalar -count=1`: pass
- `go vet ./...`: pass
- `go build ./...`: pass
- `gofmt -d` on changed Go sources: empty
- exact CR diff check: pass

The published leap-second implementation is table-backed and correctly accepts
the real `1990-12-31T23:59:60.000Z` through parse, JSON, and text boundaries
while rejecting ordinary-minute and unpublished-date `:60`. The section-owner
rename attacks also pass. No unsupported doctor/capability claim was found, and
durable crash/idempotency evidence is not applicable to this read-only package.

Reviewer validation logs and the executable Windows-path probe are attached as
task-scoped outcome resources.
