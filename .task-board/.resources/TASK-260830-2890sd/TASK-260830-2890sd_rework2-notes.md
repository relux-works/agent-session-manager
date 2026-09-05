# Round-3 rework notes — F6-F10 closed (commit 44a4699, signed, verified)`
`HEAD 44a4699, one commit past d8fc669, worktree clean. No production change; all findings were evidence defects.`
`## Fixes`
`- F6 uid 0: new owner_root_test.go — Approves unit (operator-only refuses 0; root-operator admits 0; root-admin admits 0), Discover end to end (uid:0 named in detail), Verify unchanged-tree accept under root-operator + integrity_failure under operator-only. Mutant if uid==0 return true reddens at all three levels (confirmed present at provider.go:147 before measuring).`
`- F7 plugin_dirs index domain: TestDiscoverRefusesRelativePluginDir swept over indices 0-2 with invalid_config + Detail pin. Mutant narrowed to index==0 reddens (index-1 case reports local_precondition_failed).`
`- F9 derived source inventory: new TestTrustGateSourceInventoryIsDerivedFromDiscover parses Discover with go/ast — exactly 2 collectDirectory sites with classes [path plugin_dirs], zero direct ReadDir/externalID in Discover, zero inline KindExternal (one builtin). Bijection with trustGateSources() rows incl. index-0 and index>0 coverage; ratio 2/2 derived classes covered. Reviewer fourth-source mutant (raw ReadDir, source system) reddens; collectDirectory-shaped fourth source also reddens.`
`- F10 digest sweep: new TestVerifyRefusesDigestChangeAtEveryByteIndex, 32/32 byte positions refused with 31-shared-bytes assertion each. Mutant sum[1:] reddens at byte 0 while the late-byte test passes — the mirrored shape demonstrated and closed.`
`- F8: rev1 entry 0210 and rev2 entry 0230 landed verbatim in LOGBOOK.md above 0115, newest first.`
`## Evidence`
`- go test ./internal/provider/ -count=1 exit 0, 43 top-level PASS, cover 94.1pct; go test ./... -count=1 exit 0, all 14 packages; go vet ./... exit 0; GOOS=windows build ok; tracecheck exit 0 unchanged at 17/403 bindings=49.`
`- provider.go verified byte-identical after every mutation batch (diff against pre-batch backup); worktree clean; commit signature verified with git verify-commit.`
`## Disclosures (not rework)`
`- Windows owner attestation still unverified at runtime (cross-compile only); §7.1 ownership claim still deferred to orchestrator (tracecheck 17/403 unchanged).