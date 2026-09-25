# TASK-260830-2bnr39 review logbook handoff

Candidate: CR-TASK-260830-2bnr39-1 revision 1, tree 2a7575a9b8b5983308fe8769c88a4006858580cf, base 7aa151a9c31071bfab190fd8ae8259f502f2ebbc.

- Capture run from a repository subdirectory returns only that subtree's index entries and strips their prefix; stats then use the wrong root-relative names. CWD comes from the reviewing process rather than the captured directory.
- Re-reading the HEAD OID and ls-files debug rendering does not prove HEAD/ref and index consistency. A same-commit symbolic-ref change and a v2-to-v4 index rewrite both evade the gate.
- An inherited GIT_OPTIONAL_LOCKS=1 supersedes the runner's prepended zero. A production Capture in a disposable repository rewrote .git/index, then refused its own mutation. The no-durable-state claim is false under this environment.
- Fatal upstream/config/filter read exit statuses are accepted as missing/false/empty state. Missing core.symlinks is captured false even with Git's symlink behavior; a required filter configured with Git's boolean yes is omitted.
- A live 129-character remote name passes Capture although the pinned GitRemote domain is 1..128 characters.
- Producer evidence only supplies narrowing coverage for 13 of 16 declared gates. Its OID-swap neutral row invokes no tests (None exit); the review independently executes that control.

Evidence: TASK-260830-2bnr39_review-verdict-rev1.md and TASK-260830-2bnr39_review-evidence-rev1.tar.gz. All adversarial files and fixtures are disposable. No repository code or LOGBOOK.md was edited by the reviewer; this task-scoped logbook entry is attached for the producer to incorporate when correcting the existing candidate claims.
