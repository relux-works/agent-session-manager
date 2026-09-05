# Rework notes — F1-F5 (review RUN-260904-4f8f04) 

Head: d8fc669 (amended leaf 292fc6d; signed, verified; branch one commit past checkpoint, worktree clean). Production diff: Discover validates every PATH entry as absolute (invalid_config, PATH[i] detail) + doc comment. No other production change. 

F1 — PATH trust gates: new TestDiscoverEnforcesTrustGatesAcrossSources, 3 sources x 4 dims = 12/12 refused with expected codes. Mutant (force IsRegular+operator UID for source==path): suite red, 10/12 (name/read still refuse — correct, they fire before/after the bypass). 
F2 — relative PATH bypass: closed; fake-seam TestDiscoverRefusesRelativePATHDir (. and /abs+relative) + production-seam TestOSSystemRefusesRelativePATHDir (relative PATH via real OSSystem). Mutant (drop PATH absolute check): both tests red. 
F3 — shape-only fixture: replaced_with_directory now keeps bytes identical so only the IsRegular branch can refuse; Detail pins added per Verify subtest (target changed / digest changed / owner is now / no longer a regular file). Mutant (delete Verify IsRegular check): red. 
F4 — digest 1-of-32: new TestVerifyDetectsLateByteDigestChange (receipt differs only in final digest byte, 31-byte prefix asserted shared). Narrow-to-[:1] mutant: caught ONLY by the new test (neighbors stay green) — confirms the diagnosis and the fix. 
F5 — owner-identity half: new approved-administrator subtest (uid 1000 -> approved adminUID 7, bytes/shape/approval unchanged) asserting uid:7 detail. Mutant (drop identity != record.owner): red. 

Gates: provider 38 PASS, cover 94.1pct; go test ./... green (14 pkgs); go vet clean; tracecheck exit 0, still 17/403 — ownership JSON untouched, 7.3 manifest claim still deferred per the verdicts own judgement. Stated bound unchanged: native Windows owner attestation unimplemented (os_windows.go refuses; unverified at runtime on this host).