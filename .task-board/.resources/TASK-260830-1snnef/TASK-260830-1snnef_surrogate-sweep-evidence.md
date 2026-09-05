# TASK-260830-1snnef: surrogate gate corpus + WTF-8 evidence

## Finding (reproduced on clean tree before fix)
- `TestSurrogateGateAgreesWithCanonicalJSON` (10 hand vectors): PASS.
- Raw WTF-8 doc `"<ED A0 80>"`: `hasLoneSurrogateEscape=false` while `canonicaljson.Canonicalize` refuses (`input is not valid UTF-8`). Gate blind, agreement test green while implementations disagree. Same corpus defect as the sibling story.

## Production change (`internal/terminalbackend/manifest.go`, `canonicaljson` untouched)
- `hasLoneSurrogateEscape` now also reports raw WTF-8 (CESU-8) surrogate encodings `ED A0..BF 80..BF` (= U+D800..U+DFFF) inside string literals, via `isWTF8SurrogateAt`. `ED 80..9F` (U+D000..U+D7FF, valid UTF-8) does not match. The escaped byte after a backslash is tested explicitly so the backslash skip cannot hide a WTF-8 head.
- `decodeStrictObject` ordering unchanged: encoding arm fires first on WTF-8 (`document encoding`), surrogate arm second, decoder last. The rule still fires before any canonicalization.

## Tests
- `TestSurrogateGateAgreesWithCanonicalJSON` (`internal_pin_test.go`): 10-vector table replaced by derived sweep (~4100 vectors): every U+D800..U+DFFF code point as lone escape (reject); fixed high + every surrogate-range second (admit iff low); opposite-corner pair; outside-range representatives {0000,0041,00E9,D7FF,E000,FFFF} alone (admit) and after high (reject); uppercase hex; escaped backslash; plain/multibyte; WTF-8 U+D800/U+DC00/U+DFFF/mid (reject both), U+D7FF valid (admit both), WTF-8 after backslash escapes (reject both).
- `TestDocumentWTF8SurrogateRefused` (`manifest_test.go`): raw `ED A0 80` injected at ParseProbe/ParseManifest/ParseEvidence entries refuses with `document encoding` before canonicalization.

## Mutant kills (measured, production restored identical after)
- Low-bound narrowing `unit <= 0xdfff` -> `<= 0xdc00`: sweep FAILS with 1023 agreement failures (0xDC01..0xDFFF), matching the reported 1023 admissions.
- High-bound narrowing `unit <= 0xdbff` -> `<= 0xd800`: sweep FAILS with 1024 agreement failures (0xD801..0xDBFF).
- Restored file verified byte-identical (`diff` clean) before final runs.

## Gates (exit codes)
- `gofmt -l internal/terminalbackend/`: clean (no output).
- `go vet ./internal/terminalbackend/`: exit 0.
- `go test ./internal/terminalbackend/`: exit 0.
- `go test ./...`: exit 0 (all packages ok).
- `go test ./... -cover`: exit 0 (terminalbackend 90.3%).
