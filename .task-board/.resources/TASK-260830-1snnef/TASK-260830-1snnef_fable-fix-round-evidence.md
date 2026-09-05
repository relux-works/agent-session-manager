# TASK-260830-1snnef — Fable fix round (B1–B4) evidence

Scope: `.temp/fable-consolidated-review.md` findings B1–B4 against
`internal/terminalbackend` (+ registry pins). Leaf-3 work untouched:
`conformance.go` / `legacyForward` pin not modified; `canonicaljson`
not modified (coordination constraint).

## Production changes

- B1 — `terminalbackend.go`: `InstanceBinding` gains `TerminalBindingID`
  (digest of the validated Terminal Instance Binding 1.0.0, SPEC.md:3461);
  `CheckProviderDescriptor` shape-checks it via `scalar.ParseDigest` and
  checks equality, each with its own `CodeNotFound` / `descriptor binding
  digest` arm. Doc comment now names all four §7.A dimensions.
  Ratio: 4/4 mandated dimensions implemented (was 3/4).
- B2 — `checkGeneration` (`terminalbackend.go`), `GenerationDigest`,
  `boundedStringMember`, and the inline `provider_build` check
  (`manifest.go:1962`, same class, unnamed in the report) count
  `utf8.RuneCountInString`; `utf8.ValidString` precondition kept;
  `maxGenerationBytes` renamed `maxGenerationRunes`. `ParseID` still
  counts ASCII bytes per spec.
- B3 — `manifest.go`: `hasLoneSurrogateEscape` + `readUTF16EscapeUnit`
  (local port of `canonicaljson.validateSurrogateEscapes`), called from
  `decodeStrictObject` before any decode/canonicalization; single
  `CodeMismatch` / `document surrogate escape` arm. Answers only the
  surrogate question; malformed escapes and control chars stay with the
  syntax arm (stated bound in code + pin test).
- B4 — `ownership.v0.5.0.json`: new `terminal-provider-descriptor-7a`
  group owning `CheckProviderDescriptor` with 4 test refs. Re-pinned
  `reviewedOwnershipCanonicalSHA256` to
  `8281d96b…2e8af5ea17`; updated derived figure 76→77 in
  `traceability_test.go`, `tracecheck/main_test.go` (x2), `README.md:1859`.

## Tests

- `TestCheckProviderDescriptorBindingDigestMismatch` (malformed / empty /
  foreign digest; asserts code + `descriptor binding digest` clause).
- Multibyte boundaries: 256×`é` admitted / 257×`é` refused in
  `TestCheckProviderDescriptorGenerationBounds`, `TestGenerationDigestBounds`,
  new `TestMultibyteStringBounds` (probe `os_version`, evidence
  `provider_build`, re-sealed so only the bound decides).
- `TestDocumentSurrogateEscapeRefused` (escape injected into marshaled
  bytes at ParseManifest/ParseProbe/ParseEvidence; asserts the surrogate
  arm, proving refusal precedes canonicalization).
- `TestSurrogateGateAgreesWithCanonicalJSON` (white-box, 10 shared
  vectors incl. valid pair, lone high/low, high+text, escaped backslash).
- Inventory rows: 2× `descriptor binding digest` (occurrences 1, 2),
  1× `document surrogate escape`.

## Mutant kills (each restored after; final suite re-run green)

- M1 narrowed the digest-mismatch arm to a tautology-adjacent condition:
  `foreign binding digest` admitted → test red. Killed.
- M2 reverted `checkGeneration` to `len()`: `256 two-byte runes`
  refused → test red. Killed.
- M3 deleted the gate call: surrogate docs slid to `document identity
  binding` (member checks passed post-U+FFFD replacement) → test red,
  empirically confirming the reported violation shape. Killed.
- M4 narrowed the gate to high surrogates only: `lone low surrogate`
  and `low then high` vectors red. Killed.

## Gate log (exit codes)

- `go build ./...` → 0
- `go vet ./...` → 0
- `go test ./internal/terminalbackend/ ./internal/traceability/ -count=3` → 0
- `go test ./...` (full suite) → 0
- coverage: terminalbackend 90.2%, traceability 86.6%, config 94.7%
- M1–M4 mutant runs → non-zero (expected-red), production restored each time

## Notes

- Existing `InstanceBinding` constructions (4 sites) gained digests; no
  other package constructs the struct (single `grep` sweep).
- `gofmt -l` clean on touched packages.
- Do-not-commit instruction honored: work left in the working tree.
