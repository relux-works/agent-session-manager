# TASK-260830-2890sd — round-3 mutation log (raw sweep output)

Harness: `.temp/TASK-260830-2890sd-r3/sweep.py` — restores every file from a pristine backup before and after each mutant; `git status --short` verified empty at the end.

```
--- batch1 ---
--- batch2 ---
--- batch3 ---
--- batch4 ---
--- batch5 ---
--- batch6 ---
killed    A1 deriveDiscoverSourceSites returns empty domain
killed    A2 derivation misses one real collectDirectory site
killed    A3 production bypass source (ReadDir + externalID + inline KindExternal)
killed    A4 inline KindExternal candidate only
killed    A5 direct externalID name parse in Discover only
SURVIVED  B1 deriveRefusalInventory returns empty domain
SURVIVED  B2 refusal inventory derives a short domain (1 of N sites)
killed    B3 new unexercised production refusal call site
killed    B4 stray Error literal outside the instrumented constructors
SURVIVED  B5 stray Error literal + empty-but-successful directory read (wrong cwd)
killed    G1  plugin_dirs abs-path gate narrowed to index 0
killed    G1b plugin_dirs abs-path gate deleted
killed    G2  PATH abs-path gate narrowed to index 0
killed    G3  allow_path_plugins gate removed
killed    G4  duplicate refusal narrowed to same-source duplicates
killed    G5  collectDirectory ReadDir failure treated as empty directory
killed    G6b collectDirectory bytewise sort removed
killed    G8  malformed provider name skipped instead of refused
SURVIVED  G9  trustCandidate Canonicalize failure deletion form (canon = path)
killed    G9n trustCandidate Canonicalize refusal narrowed to /opt paths
SURVIVED  G10 trustCandidate Inspect failure deletion form (fabricated FileInfo)
killed    G10n2 trustCandidate Inspect refusal narrowed to /zzz paths
killed    G11 trustCandidate regular-file gate deleted
killed    G12 trustCandidate owner-approval gate deleted
killed    G12b owner gate narrowed to non-path sources
killed    G13 trustCandidate digest read failure treated as empty content
killed    G14 Approves admits uid 0 (superuser exception)
killed    G15 Approves admits any uid when an administrator set is configured
killed    G15b Approves administrator branch deleted
killed    G16 Trust builtin refusal deleted
killed    G17 Verify re-resolve failure treated as unchanged
killed    G17n Verify re-resolve refusal narrowed to /zzz paths
killed    G18 Verify canonical-target retarget gate deleted
killed    G18b OwnerPolicy operator-UID branch deleted
SURVIVED  G19 Verify re-inspect failure deletion form (fabricated FileInfo)
killed    G19n Verify re-inspect refusal narrowed to /zzz paths
killed    G19b OwnerIdentity loses the uid discriminator
killed    G20 Verify regular-file gate deleted
killed    G21 Verify owner-approval half of the owner gate dropped
killed    G22 Verify recorded-owner half of the owner gate dropped
killed    G23 Verify re-read failure treated as unchanged
killed    G23n Verify re-read refusal narrowed to /zzz paths
killed    G24n2 Verify digest gate compares only the first byte
killed    G24n3 Verify digest gate compares only the last byte
SURVIVED  G25 digestBytes length gate removed            [equivalent]
SURVIVED  G26 digestBytes non-hex gate removed           [equivalent]
SURVIVED  G27 unhex accepts uppercase hex                [equivalent]
killed    G28 SourcePath reports presence for builtins
killed    G29 CanonicalPath reports presence for builtins
killed    G30 Digest reports presence for builtins
killed    G31 Owner reports presence for builtins
SURVIVED  G32 Builtins returns the live registry slice
SURVIVED  G33 OSSystem.ReadDir error swallowed as empty directory
killed    G34 OSSystem.Canonicalize skips symlink resolution
SURVIVED  G34b OSSystem.Canonicalize Abs error swallowed [equivalent]
SURVIVED  G35 OSSystem.Inspect stat error swallowed
SURVIVED  G36 OSSystem.Inspect owner-attestation failure treated as uid 0
SURVIVED  G36b OSSystem.Inspect reports lstat shape      [equivalent]
SURVIVED  G37 OSSystem.PathDirs keeps empty PATH entries
killed    G37b OSSystem.PathDirs returns nil for a non-empty PATH
SURVIVED  G38 fileOwnerUID unix returns uid 0 when ownership metadata unavailable
SURVIVED  G39 fileOwnerUID windows attests uid 0 instead of refusing [build-tag bound]
SURVIVED  G40 CurrentOperatorPolicy seeds an administrator wildcard {0}
SURVIVED  G40b CurrentOperatorPolicy uses real uid instead of effective uid
killed    G41 externalID accepts a bare prefix with an empty provider id
SURVIVED  G42 externalID matches any name containing the prefix (Cut vs CutPrefix)
killed    G43 raw fmt.Errorf minted in provider.go outside the constructors
killed    G44 panic minted in provider.go
killed    G45 a fourth refusal code leaks into the closed code set
killed    G46 builtin registry loses a Section 7.1 adapter
killed    G47 builtin registry order permuted
killed    G48 discovery order: builtins enumerated before plugin_dirs
killed    G49 ExecutablePrefix loosened to ax-
killed    G50 Trust drops the recorded owner identity
killed    G51 Trust drops the undisguised source path Verify re-resolves
killed    G52 Trust drops the trust instant
killed    G53 Discover accepts a duplicate when both observations are byte-identical
SURVIVED  G54 Discover returns the partial candidate set alongside a refusal
SURVIVED  G55 Verify accepts when the receipt owner is empty
SURVIVED  H1 exportedSymbols derives an empty symbol domain (no floor)
killed    H2 structMembers derives an empty member domain (floor control)
killed    H3 production-source scan finds no files (floor control)
SURVIVED  H4 refusal inventory skips provider.go entirely (short domain)
```

**59 killed / 84 run. 25 survived; 5 marked `[equivalent]` → 20 substantive survivors across 9 defects (F11-F17 plus the two deletion-form precondition pairs).**

Excluded as BROKEN (mutant did not compile, re-run in a corrected form): G6 unused `sort` import, G24/G24n unused `subtle`, first G38 unused `fmt`, G10n (bad mutant: the /opt narrowing still covered the real fixture path).
