Review TASK-260830-21gygk against its actual published immutable CR and current pinned AX v0.6.0 specification (0cbdf100dbf84df50c64f792b1f940e3a67859a6). This is the final leaf of STORY-260830-3tq4ns. Use the standard independent reviewer lifecycle and local checks only. Do not treat this brief or producer counts as acceptance evidence.

Read task scope and original AC. Scope includes 14.7/14.7.1 and shared SelectionPlan construction/revalidation 14.7.2, refining 2.3 and authoritative summaries 5.7/14.7.3. Public CLI invocation belongs to caller tasks, but all assigned shared-library behavior must actually exist. Distinguish legitimate caller integration from missing shared semantics.

Inspect the producer outcome critically, especially these declared boundaries:
- lease_record_id is described as unbound, with an envelope tuple bound instead. Compare the actual normative plan/fact requirements and justify equivalence or report a gap.
- The outcome places 14.7.3 list refusal and 14.7.4 recovery elsewhere. Task scope explicitly includes authoritative summaries in 14.7.3; do not permit scope to be narrowed by an outcome note.
- Configuration and learned indexes are caller inputs; prove source allowlisting, complete reads, explicit-source no-fallback, errors vs absence, collisions, tombstones and current-fact revalidation at the shared API boundary.
- Check plan immutability/local attestation, exact source mapping, first-at literal grammar and stable deterministic sorting through real entry points.
- Re-derive the claimed 8/8 AC result; do not count future CLI behavior as already implemented.

The candidate predates the v0.6.0 adoption landing, and its producer combined incoming main with the preserved Story candidate. Verify README/LOGBOOK merge content, the pinned spec/catalog/configuration and ownership references, and preservation of accepted predecessor sessrepo/sessstate behavior. Initial CR publication required a task-board refresh fix; tool recovery is not proof of product correctness. Review the candidate that was actually published after refresh.

Producer mutation claims include 29 killed semantic mutants and controls, with earlier logs in /tmp/sel-mutants-final and /tmp/sel-mutants-01. Require durable scoped evidence or rerun the relevant instrument. A compile failure, unapplied plant, empty selection, or truncated log is not a behavioral kill. Recheck that controls demonstrate SURVIVED through the same delivered gates.

Preserve the earlier settled predecessor review conclusions unless this delta changes them: canonicaljson closed enums; multiple-predecessor 5.2 conformance; the four crash points inside creation; pure reducer concurrent calls; accepted predecessor AC. Focus on new or changed behavior rather than repeating closed reviews.

Attach a task-scoped verdict with concrete file/line and rerunnable evidence for findings, complete the live reviewer checklist and use the normal accept_cr or changes-requested handoff. Do not modify product code or manually land/checkpoint the Story. Review must bind the actual current CR revision, never a guessed revision or metadata-only substitute.
