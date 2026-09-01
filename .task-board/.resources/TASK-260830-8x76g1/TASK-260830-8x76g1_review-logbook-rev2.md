# TASK-260830-8x76g1 review logbook — CR revision 2

- Review bound to candidate tree `2885010caafe4fc54651ea11232398825deada93`.
- Patch resource SHA-256 matched the handed Change Request digest.
- Original CR revision 1 unknown-member/chunk-invariant bypass is fixed on the
  composed `prepareObjectIdentity` call path.
- New attack: a non–reverse-DNS Transfer Manifest extension key was accepted by
  calculation and verification with a correct claim.
- New compatibility check: a 3,999-character multibyte symlink target was
  rejected because production counts bytes instead of normative UTF-8
  characters.
- Gate audit: `FuzzClosedIdentityShapeRefusal` is documented and manually
  runnable but absent from the configured worktree validation command list.
- Reviewer reran focused/full tests, all four fixed-count fuzz targets, scoped
  tracecheck, vet, build, and coverage; all passed.
- Verdict is ordinary implementation rework (`to-dev`), not an external
  Stop-The-Line blocker.
