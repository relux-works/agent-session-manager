# TASK-260830-8x76g1 republish logbook

## 2026-09-01 — Finder drift recovery

- Verified both attached recovery archives against their declared SHA-256 digests and inspected their contents without overwriting the later revision 6 working tree.
- Confirmed the current non-Finder candidate matches CR revision 6 by a successful reverse-apply dry run with `.DS_Store` excluded.
- The unfiltered reverse-apply dry run failed with exit 1 specifically on `.DS_Store`; this is expected-red evidence that the recorded CR revision carried the Finder artifact.
- Deleted the root and temporary `.DS_Store` files. Finder recreated both during evidence inspection, proving deletion alone was not stable.
- Added `.DS_Store` to the existing repository `.gitignore`. The temporary copy is independently ignored by the pre-existing `.temp/` rule.
- Re-ran the full configured validation suite before the ignore change, then re-ran the affected formatting, full-test, build, and diff gates after it. Every green claim is backed by an exit code 0 log.
- No external blocker or forced-fit condition remains.
