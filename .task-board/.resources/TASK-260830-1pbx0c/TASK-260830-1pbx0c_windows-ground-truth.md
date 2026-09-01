# Win32 ground truth for ParseAbsolutePath(PlatformWindows, ...)

Measured on a real Windows 10 Pro host (SSH alias `win`, PowerShell 5.1,
go1.25.5 windows/amd64) by attempting `[IO.File]::WriteAllText` under `%TEMP%`.
This is platform behaviour, not folklore.

| Component | Real Win32 | AX validator at CR revision 5 |
| --- | --- | --- |
| `ok.json` | created | accepted — correct |
| `CON` | refused | **accepted** — defect |
| `con.txt` | refused | **accepted** — defect |
| `NUL.txt` | refused | **accepted** — defect |
| `COM1` | refused | **accepted** — defect |
| `LPT1` | refused | **accepted** — defect |
| `star*.json` | refused | **accepted** — defect |
| `q?b` | refused | **accepted** — defect |

Reviewer-reported shapes reproduced locally, plus two the review did not list:
`C:\unsafe\a?b` and `C:\unsafe\LPT1`.

## What the fix must honour

1. A reserved DOS device name is reserved **regardless of extension**: `con.txt`
   is refused exactly like `CON`. Matching only the exact bare name is
   insufficient.
2. The reserved set is `CON`, `PRN`, `AUX`, `NUL`, `COM1`-`COM9`, `LPT1`-`LPT9`,
   and matching is case-insensitive.
3. Win32 wildcard and reserved punctuation `* ? < > : " | ` are invalid in a
   path component; the drive-letter colon is the sole legitimate colon.
4. The rule applies to UNC components too: `\\server\share\COM1` must be refused.
5. Do not weaken any POSIX-platform behaviour or the refusals earlier revisions
   already established.

## Reproduction

Local, against the production entry point:

```go
scalar.ParseAbsolutePath(scalar.PlatformWindows, `C:\unsafe\CON`)
```

Remote, on the Windows worker: `[IO.File]::WriteAllText` under `%TEMP%` for each
component name, delivered through `powershell -EncodedCommand` because plain SSH
argument quoting is mangled by the cmd layer.
