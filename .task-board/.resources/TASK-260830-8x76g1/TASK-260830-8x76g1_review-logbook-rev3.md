# TASK-260830-8x76g1 review logbook — CR revision 3

- Reviewed immutable candidate tree `93e6d212a49ab7320e061254d9507a567ccd3852`; independently matched the published patch SHA-256.
- Read the pinned specification at commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, especially Sections 1.6, 10.1-10.4, and 17.3.
- Confirmed the revision-2 fixes: complete reverse-DNS extension-key grammar, extension count/depth/canonical-size checks, Unicode counting for the symlink/media-type checks that exist, and fixed-budget wiring for all four fuzz targets.
- Found a broader same-boundary bypass: recursive validators check nested member sets but omit their declared value constraints. Both production identity entries attested over-bound nested strings, malformed nested digests, forbidden Git index stage 4, and an impossible submodule state.
- The candidate's own full tests, build/vet, race, coverage, four fixed-budget fuzz runs, and scoped tracecheck all passed. The successful closed-shape fuzz run does not cover the failing mutation class.
- No external blocker or human-only decision is involved. Rework should remain in the same implementation scope and return as Change Request revision 4.
