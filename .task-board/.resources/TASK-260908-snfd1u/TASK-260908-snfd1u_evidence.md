# Research evidence ledger — TASK-260908-snfd1u

This ledger supports the bounded proposal, not acceptance of either blocked implementation. Research date: 2026-09-08. No Go suite, mutation suite, live SSH session, enrollment or platform deployment was run.

## Local authority and inspection

- Selector Story worktree: `.temp/STORY-260830-3tq4ns/worktree`, HEAD `7208cc7427e1ebe95106d34f98214e2bbea4ae1c`; existing uncommitted implementation retained.
- RPC Story worktree: `.temp/STORY-260830-1kiyj6/worktree`, HEAD `c1eff016dce2e55c4e2c1828d5c20f3da84118bc`; existing uncommitted implementation retained.
- Both `HEAD..main` counts were 0 against local main during snapshot collection. No fresh remote-main equality is asserted; no fetch, branch switch, rebase or commit was performed.
- `internal/specdoc/SPEC.md`: both copies match SHA-256 `562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`, pinned upstream `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.
- Read normative Sections 2.1/2.3, 5.7, 6.2/6.3, 7.6, 11.1/11.2, 12.3, 13.1, 14.2, 15.1/15.3, 16.1, 17.1/17.2. The task's required Sections 2.3/6/11.1/11.2/16 are covered; supporting sections bound summaries, capture, errors and migration.
- Read current `internal/sessquery/query.go` (Reader, source provenance, resolution, summary bounds), `internal/sessstate/project.go` (authoritative projection and empty owner), and session repository context.
- Read RPC Story `internal/peeridentity/identity.go` (Directory.Resolve, Target.RPCArgv, Target.CheckProtocolHost) and `internal/sshtransport/transport.go` (Client.Open, Session.Target, process-start-only result).
- Exact stop packet and partial-routing resource links are embedded in the proposal. The prior identity counterexample is an architectural composition probe, exit 1 expected-red, not a native SSH attack or this run's test.

## Primary-source fact checking

All URLs below were successfully retrieved through the web tool on the research date. Claims in the brief use citations at the relevant paragraph. These sources establish documented building blocks, not an integrated AX deployment proof.

| Source | Claim checked |
| --- | --- |
| https://man.openbsd.org/sshd#AUTHORIZED_KEYS_FILE_FORMAT | Forced key commands, restrictions, certificate principals, client-controlled original command |
| https://man.openbsd.org/sshd_config#ForceCommand | ForceCommand, independent forwarding restriction, ExposeAuthInfo, root-owned principal hooks |
| https://man.openbsd.org/ssh-keygen#-Y | SSHSIG domain namespace, signing/verification, allowed signers; raw signing is not an AX authentication protocol |
| https://man.openbsd.org/ssh-keygen#KEY_REVOCATION_LISTS | KRL key/certificate revocation support |
| https://tailscale.com/docs/features/tailscale-ssh | Separate SSH server, WireGuard node authentication, acceptEnv, source/user rules, Linux/open-source-macOS server restriction |
| https://tailscale.com/docs/reference/ssh-over-tailscale | Ordinary SSH over the Tailscale network is distinct from Tailscale SSH |
| https://learn.microsoft.com/en-us/windows-server/administration/openssh/openssh-server-configuration | Windows OpenSSH differences: no ExposeAuthInfo/AuthorizedKeysCommand/AuthorizedPrincipalsCommand; SFTP-only chroot |
| https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man3/getpeereid.3.html | Archived Apple documentation for Unix-socket peer effective credentials; no current deployment test claimed |
| https://learn.microsoft.com/en-us/windows/win32/ipc/impersonating-a-named-pipe-client | Named-pipe client token impersonation building block; exact AX SID boundary remains to be tested |
| https://www.rfc-editor.org/rfc/rfc8446.html | TLS 1.3 client/server authentication, CertificateVerify/Finished, protected records and 0-RTT replay limitations |
| https://git-scm.com/docs/git-bundle | Bundle refs/reachable objects do not constitute index/worktree capture; copying live repositories can race writers |

Native Tailscale SSH's protected node→AX invocation binding was not established by the inspected documentation. This is an explicit evidence limit, not a claim that no such integration can be engineered. The rejected obsolete whoami URL was not used as evidence.

## Checks run directly and actual exits

| Command/check | Actual exit | Evidence |
| --- | ---: | --- |
| Direct Python source snapshot plus exact SPEC digest assertion | 0 | `source-snapshot-01.json`, `pin-check-01.log` |
| Direct Python document word count after revision | 0 | 1944 whitespace-delimited words |
| `python3 /Users/iv/Developer/ReluxWorks/agent-session-manager/.temp/TASK-260908-snfd1u/validate-proposal.py` (stdout redirected, no pipeline) | 0 | `proposal-validation-01.log` |

The document audit checks length, balanced code fences, required topic markers, four existing stop/routing references and exact source preservation. It does not mechanically prove recommendation correctness. Manual self-review checked the proposal against the task brief: two selector meanings, alias/name collision handling, deterministic errors, source/destination separation, identity lifecycle, writable-account bypass, platform limits, closed protocol migration and both supporting notes.

Preservation check: 466/466 inventoried selector-Story file byte hashes and 461/461 RPC-Story hashes unchanged between source snapshots, with identical HEAD and porcelain status. This excludes ignored files and `.task-board/`; it is a bounded comparison, not a claim of global filesystem quiescence.

## Handoff scope

Proposal, task-scoped logbook and this evidence ledger are outcomes for this research task only. Required future behavioral/refusal cases are enumerated in the proposal; none are reported as passing. Product/spec approval and continuation of original owners remain with the coordinator. No blocked owner is marked implemented or accepted.
