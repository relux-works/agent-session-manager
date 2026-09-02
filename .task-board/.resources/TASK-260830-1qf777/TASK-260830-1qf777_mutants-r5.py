import subprocess, sys, os, re

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.join(HERE, "tree")
TASK = os.path.dirname(HERE)
GOCACHE = os.path.join(TASK, "r5", "gocache")


def borrow(path, name):
    """Load an earlier round's mutant list without executing its harness.

    Prior rounds are re-run here against the WHOLE package rather than a
    per-mutant -run mask, so a mutant that was only red under one named test
    still has to die against the full suite.
    """
    text = open(path).read()
    start = text.index(name + " = [")
    end = text.index("\ndef run(", start)
    namespace = {}
    exec(text[start:end], namespace)
    return [entry[:4] + (entry[-1],) for entry in namespace[name]]


PRIOR = (borrow(os.path.join(TASK, "mutants.py"), "MUTANTS")
         + borrow(os.path.join(TASK, "review-r3", "mutants.py"), "M"))

# R14's anchor moved: the v2 re-read now carries a config-refusal-subsumed
# marker naming why it is defence in depth. Re-point the mutant at the current
# text so the reviewer's mutant is still exercised rather than silently skipped.
R14_OLD = ("\tif _, err := Decode(output.Bytes(), context); err != nil {\n"
           "\t\treturn nil, errors.Join(ErrConfigEncode, err) // config-refusal-subsumed: v2 re-read"
           " - defence in depth only; the sole production caller is Migrate with a Configuration 1.0.0"
           " source, whose closed reader has already validated every member this wire shape carries,"
           " so no valid v1 source can produce a v2 document this re-read refuses\n\t}\n")
PRIOR = [(name, path,
          R14_OLD if name.startswith("R14") else old,
          "" if name.startswith("R14") else new,
          why)
         for name, path, old, new, why in PRIOR]

# Round-5 mutants: the three gaps the rev3 review left open, their widening and
# narrowing counterparts, the member-preservation class the property must now
# cover, and self-checks that redden the new derived gates themselves.
NEW = [
 ("N01-currentwire-drops-safe-boundary-timeout", "internal/config/writer.go",
  "BackendID: pointer(configuration.Terminal.BackendID), SafeBoundaryTimeoutSeconds: pointer(configuration.Terminal.SafeBoundaryTimeoutSeconds),",
  "BackendID: pointer(configuration.Terminal.BackendID),",
  "delete: v1/v2 -> v3 silently drops the operator's safe-boundary timeout (rev3 F2 survivor)"),
 ("N02-v2-encoder-drops-graceful-stop-timeout", "internal/config/writer.go",
  "GracefulStopTimeoutSeconds: pointer(configuration.Terminal.GracefulStopTimeoutSeconds),\n\t\t},\n\t\tService: current.Service",
  "},\n\t\tService: current.Service",
  "delete: v1 -> v2 silently drops the operator's graceful-stop timeout (rev3 F2 survivor)"),
 ("N03-v2-encoder-drops-safe-boundary-timeout", "internal/config/writer.go",
  "Backend: pointer(backend), SafeBoundaryTimeoutSeconds: pointer(configuration.Terminal.SafeBoundaryTimeoutSeconds),",
  "Backend: pointer(backend),",
  "delete: v1 -> v2 silently drops the operator's safe-boundary timeout (rev3 F2 survivor)"),
 ("N04-temp-file-fsync-deleted", "internal/config/migration.go",
  "\tif err := file.Sync(); err != nil {\n\t\treturn clean(err)\n\t}\n", "",
  "delete: a staged file is renamed into place without its contents fsynced (rev3 F3 survivor)"),
 ("N05-readonly-only-for-two-major-gap", "internal/config/migration.go",
  "if compareSemver(source, reader) > 0 {", "if source.major > reader.major+1 {",
  "narrow: a one-major-newer document is reported fully compatible (rev3 F1 survivor)"),
 ("N06-readonly-widened-to-equal-versions", "internal/config/migration.go",
  "if compareSemver(source, reader) > 0 {", "if compareSemver(source, reader) >= 0 {",
  "widen: a reader is refused write access to its own version"),
 ("N07-compatibility-mode-gate-deleted", "internal/config/migration.go",
  "\tmode := CompatibilityCompatible\n\tif compareSemver(source, reader) > 0 {\n\t\tmode = CompatibilityReadOnly\n\t} else if",
  "\tmode := CompatibilityCompatible\n\tif false {\n\t\tmode = CompatibilityReadOnly\n\t} else if",
  "delete: every document is reported writable regardless of reader version"),
 ("N08-currentwire-drops-mesh-sync-interval", "internal/config/writer.go",
  "Transport: pointer(configuration.Mesh.Transport), SyncIntervalSeconds: pointer(configuration.Mesh.SyncIntervalSeconds),",
  "Transport: pointer(configuration.Mesh.Transport),",
  "delete: migration loses the operator's mesh sync interval"),
 ("N09-currentwire-drops-service-health-interval", "internal/config/writer.go",
  "Service:  rawService{Enabled: pointer(configuration.Service.Enabled), HealthIntervalSeconds: pointer(configuration.Service.HealthIntervalSeconds)},",
  "Service:  rawService{Enabled: pointer(configuration.Service.Enabled)},",
  "delete: migration loses the operator's service health interval"),
 ("N10-currentwire-drops-allow-path-plugins", "internal/config/writer.go",
  "PluginDirs: cloneStrings(configuration.Providers.PluginDirs), AllowPathPlugins: pointer(configuration.Providers.AllowPathPlugins),",
  "PluginDirs: cloneStrings(configuration.Providers.PluginDirs),",
  "delete: migration loses the operator's path-plugin policy"),
 ("N11-currentwire-drops-restore-auto-resume", "internal/config/writer.go",
  "Restore:  rawRestore{AutoResume: pointer(configuration.Restore.AutoResume)},",
  "Restore:  rawRestore{},",
  "delete: migration loses the operator's auto-resume choice"),
 ("N12-currentwire-drops-yolo-confirmation", "internal/config/writer.go",
  "Profiles: rawProfiles{Yolo: rawYoloProfile{RequireFirstUseConfirmation: pointer(configuration.Profiles.Yolo.RequireFirstUseConfirmation)}},",
  "Profiles: rawProfiles{Yolo: rawYoloProfile{}},",
  "delete: migration loses the operator's yolo first-use confirmation choice"),
 ("N13-currentwire-drops-workspace-roots", "internal/config/writer.go",
  "WorkspaceRoots: wireWorkspaceRoots(configuration.WorkspaceRoots),\n\t\tProviders: rawProviders{",
  "Providers: rawProviders{",
  "delete: migration loses the operator's workspace roots"),
 ("N14-currentwire-drops-mesh-peers", "internal/config/writer.go",
  "\traw.Mesh.Peers = make([]rawPeer, len(configuration.Mesh.Peers))\n\tfor index, peer := range configuration.Mesh.Peers {",
  "\traw.Mesh.Peers = make([]rawPeer, 0)\n\tfor index, peer := range []Peer(nil) {",
  "delete: migration loses the operator's mesh peers"),
 ("N15-currentwire-drops-peer-ssh-args", "internal/config/writer.go",
  "SSHArgs: cloneStrings(peer.SSHArgs), WorkspaceRoots: wireWorkspaceRoots(peer.WorkspaceRoots),",
  "WorkspaceRoots: wireWorkspaceRoots(peer.WorkspaceRoots),",
  "delete: migration loses a peer's SSH arguments"),
 ("N16-v2-encoder-drops-workspace-roots", "internal/config/writer.go",
  "Mesh: current.Mesh, WorkspaceRoots: current.WorkspaceRoots, Providers: current.Providers, Sync: current.Sync,",
  "Mesh: current.Mesh, Providers: current.Providers, Sync: current.Sync,",
  "delete: v1 -> v2 loses the operator's workspace roots"),
 ("N17-temp-file-fsync-narrowed-to-replacement", "internal/config/migration.go",
  "\tif err := file.Sync(); err != nil {\n\t\treturn clean(err)\n\t}\n",
  "\tif strings.Contains(pattern, \"migrate\") {\n\t\tif err := file.Sync(); err != nil {\n\t\t\treturn clean(err)\n\t\t}\n\t}\n",
  "narrow: only the replacement is fsynced, the published backup never is"),
 ("N18-mesh-transport-vocabulary-widened", "internal/config/validation.go",
  'if mesh.Transport != "ssh" {', 'if !oneOf(mesh.Transport, "ssh", "quic") {',
  "widen: the mesh.transport saturation exemption stops being true"),
 ("N19-payload-encryption-vocabulary-widened", "internal/config/validation.go",
  'if mesh.PayloadEncryption != "none" {', 'if !oneOf(mesh.PayloadEncryption, "none", "aead") {',
  "widen: the mesh.payload_encryption saturation exemption stops being true"),
 ("N20-chunk-bytes-bound-widened", "internal/config/validation.go",
  "if value.ChunkBytes != 4_194_304 {", "if !between(value.ChunkBytes, 4_194_304, 8_388_608) {",
  "widen: the sync.chunk_bytes saturation exemption stops being true"),
 ("N21-explicit-trust-requirement-deleted", "internal/config/validation.go",
  '\tif !configuration.Providers.RequireExplicitTrust {\n\t\treturn configError("providers.require_explicit_trust", ErrConfigValidation)\n\t}\n',
  "",
  "delete: the providers.require_explicit_trust saturation exemption stops being true"),
 ("N22-legacy-backend-platform-coupling-deleted", "internal/config/validation.go",
  '\tif terminal.BackendID == "ax.conpty" && configuration.Platform != scalar.PlatformWindows {\n\t\treturn configError("terminal.backend_id unsupported platform", ErrConfigValidation)\n\t}\n',
  "",
  "delete: the terminal.backend saturation exemption stops being true"),
 ("N23-platform-probe-match-deleted", "internal/config/validation.go",
  '\tif configuration.Platform != context.RuntimePlatform {\n\t\treturn configError("platform must match runtime probe", ErrConfigValidation)\n\t}\n',
  "",
  "delete: the platform saturation exemption stops being true"),
 ("N24-rawv1-gains-an-uncovered-member", "internal/config/schema.go",
  'type rawV1 struct {\n\tSchema         string             `toml:"schema"`',
  'type rawV1 struct {\n\tSyntheticProbe *string            `toml:"synthetic_probe"`\n\tSchema         string             `toml:"schema"`',
  "self-check: a Configuration 1.0.0 member the saturated fixture does not carry must redden the derived completeness gate"),
 ("N25-rawv1-loses-a-member", "internal/config/schema.go",
  'type rawV1 struct {\n\tSchema         string             `toml:"schema"`\n\tSchemaVersion  string             `toml:"schema_version"`\n\tHostID         *string            `toml:"host_id"`',
  'type rawV1 struct {\n\tSchema         string             `toml:"schema"`\n\tSchemaVersion  string             `toml:"schema_version"`\n\tHostID         *string            `toml:"host_id_renamed"`',
  "self-check: a Configuration 1.0.0 member the derivation no longer declares must redden the derived completeness gate"),
 ("N26-positive-control-comment-only", "internal/config/migration.go",
  "// MigrationResult reports durable migration without claiming CLI or doctor",
  "// MigrationResult reports durable migration without claiming CLI or doctor tooling",
  "positive control: a comment-only edit must stay GREEN"),
]

M = PRIOR + NEW
EXPECT_GREEN = {"N26-positive-control-comment-only"}

# R13 cannot die and must not be reported as if it could. encodeVersion2's
# directory-collection passthrough is reachable only from Migrate, which reaches
# that encoder only with a Configuration 1.0.0 source, and rawV1 declares no
# directory collection. The named subsuming check pins that precondition and
# reddens if Configuration 1.0.0 ever gains one.
EXPECT_SUBSUMED = {
    "R13-v2-encoder-drops-directory-tables": "TestVersion1SourcesNeverCarryDirectoryCollections",
}


def run(name, path, old, new, why):
    full = os.path.join(ROOT, path)
    src = open(full).read()
    if src.count(old) != 1:
        return (name, "INVALID", "anchor count=%d" % src.count(old), why)
    open(full, "w").write(src.replace(old, new, 1))
    try:
        environment = dict(os.environ, GOCACHE=GOCACHE)
        p = subprocess.run(["go", "test", "./internal/config/", "-count=1"], cwd=ROOT,
                           capture_output=True, text=True, timeout=900, env=environment)
        out = p.stdout + p.stderr
        alive = p.returncode == 0
        if name in EXPECT_GREEN:
            status = "CONTROL-GREEN" if alive else "CONTROL-BROKEN"
        elif name in EXPECT_SUBSUMED:
            status = "SUBSUMED" if alive else "RED-UNEXPECTED"
        else:
            status = "SURVIVED" if alive else "RED"
        first = "\n".join([l for l in out.splitlines()
                           if "FAIL:" in l or "cannot use" in l or "undefined" in l
                           or "declared and not used" in l or "unknown field" in l][:4])
        return (name, status, first, why)
    finally:
        open(full, "w").write(src)


BEGIN = int(sys.argv[1]) if len(sys.argv) > 1 else 0
END = int(sys.argv[2]) if len(sys.argv) > 2 else len(M)
SELECTED = M[BEGIN:END]
print("running mutants [%d:%d) of %d" % (BEGIN, END, len(M)), flush=True)

results = []
for m in SELECTED:
    r = run(*m)
    results.append(r)
    print("%-46s %-14s %s" % (r[0], r[1], r[3]), flush=True)
    if r[2]:
        for l in r[2].splitlines():
            print("    | " + l, flush=True)

print()
print("total:", len(results))
print("RED:", sum(1 for r in results if r[1] == "RED"))
print("SURVIVED:", [r[0] for r in results if r[1] == "SURVIVED"])
print("INVALID:", [(r[0], r[2]) for r in results if r[1] == "INVALID"])
print("CONTROL:", [(r[0], r[1]) for r in results if r[1].startswith("CONTROL")])
print("SUBSUMED:", [(r[0], EXPECT_SUBSUMED[r[0]]) for r in results if r[1] == "SUBSUMED"])
print("SUBSUMPTION CLAIM NO LONGER NEEDED:", [r[0] for r in results if r[1] == "RED-UNEXPECTED"])

# The restored tree must be green again, or every verdict above is meaningless.
restored = subprocess.run(["go", "test", "./internal/config/", "-count=1"], cwd=ROOT,
                          capture_output=True, text=True, env=dict(os.environ, GOCACHE=GOCACHE))
print("restored tree exit:", restored.returncode)

bad = [r for r in results if r[1] in ("SURVIVED", "INVALID", "CONTROL-BROKEN")]
sys.exit(1 if bad or restored.returncode != 0 else 0)
