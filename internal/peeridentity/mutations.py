#!/usr/bin/env python3
"""Task-scoped behavioral narrowing probes using Go overlays, without editing sources."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
IDENTITY = "internal/peeridentity/identity.go"
CONFIG = "internal/config/validation.go"
TESTS = "TestLoadPeerIdentityVersions|TestLoadIdentityRefusals|TestResolveAllowlistAndAliasAmbiguity|TestProtocolIdentityRefusal|TestSSHTargetAtomicArgv|TestDisclosurePolicyRefusals|TestLoadAbsenceReadFailureAndRecovery|TestSnapshotIsolationAndDisclosureBinding|TestLoadRefusesLocalHostAsPeer|TestSSHTotalArgvByteBound|TestDisclosureClassPolicyBinding"

# Each plant retains the gate and admits one specific forbidden fixture value
# or route. Neutral and known-bad controls validate the overlay instrument.
PROBES = [
    ("disclosure-class-binding", IDENTITY, 'policy = entry.NativeObservations', 'policy = entry.ManualMetadata', "Allows native observations to borrow manual metadata permission", "TestDisclosureClassPolicyBinding/manual_metadata"),
    ("composed-argv-bound", IDENTITY, 'totalBytes > 65_536', 'totalBytes > 65_537', "Admits exactly 65537 composed SSH argv bytes", "TestSSHTotalArgvByteBound"),
    ("neutral", IDENTITY, 'return "external_ssh"', 'return ("external_ssh")', "Equivalent expression", None),
    ("known-bad", IDENTITY, '"ax", "rpc", "serve", "--stdio"', '"wrong-command", "rpc", "serve", "--stdio"', "Fixed command changed, code still compiles", "TestLoadPeerIdentityVersions"),
    ("local-duplicate", CONFIG, 'map[string]struct{}{configuration.HostID: {}}', 'map[string]struct{}{}', "Peer-peer uniqueness retained; admits the local identity as a remote peer", "TestLoadRefusesLocalHostAsPeer"),
    ("peer-duplicate", CONFIG, 'if _, exists := hostIDs[peer.HostID]; exists {', 'if _, exists := hostIDs[peer.HostID]; exists && peer.Name != "different alias" {', "Admits duplicate identity with the different-alias fixture", "TestLoadIdentityRefusals/duplicate_identity"),
    ("alias-duplicate", CONFIG, 'if _, exists := names[peer.Name]; exists {', 'if _, exists := names[peer.Name]; exists && peer.HostID != "0198f4c8-a070-7188-9172-1234567890ab" {', "Admits duplicate alias for one peer ID", "TestLoadIdentityRefusals/duplicate_alias"),
    ("local-id-grammar", CONFIG, 'scalar.ParseUUIDv7(configuration.HostID); err != nil {', 'scalar.ParseUUIDv7(configuration.HostID); err != nil && configuration.HostID != "bad-local-secret" {', "Admits one malformed local host ID", "TestLoadIdentityRefusals/malformed_local_UUID"),
    ("peer-id-grammar", CONFIG, 'scalar.ParseUUIDv7(peer.HostID); err != nil {', 'scalar.ParseUUIDv7(peer.HostID); err != nil && peer.HostID != "" {', "Admits explicitly empty peer ID", "TestLoadIdentityRefusals/empty_peer_ID"),
    ("alias-bound", CONFIG, 'validatePrintableCharacters(peer.Name, 1, 64)', 'validatePrintableCharacters(peer.Name, 1, 65)', "Admits 65-character aliases", "TestLoadIdentityRefusals/long_alias"),
    ("endpoint-grammar", CONFIG, 'reason != endpointAdmitted {', 'reason != endpointAdmitted && peer.Endpoint != "-oStrictHostKeyChecking=no" {', "Admits one option-shaped endpoint", "TestLoadIdentityRefusals/option_endpoint"),
    ("ssh-auth-policy", CONFIG, 'reason != sshArgumentAdmitted {', 'reason != sshArgumentAdmitted && !(len(peer.SSHArgs) == 1 && peer.SSHArgs[0] == "-voStrictHostKeyChecking=no") {', "Admits one grouped host-key bypass", "TestLoadIdentityRefusals/grouped_host_key_bypass"),
    ("selector-allowlist", IDENTITY, 'peer.HostID != selector && peer.Name != selector {', 'peer.HostID != selector && peer.Name != selector && selector != "discovered.example" {', "Admits one discovery candidate", "TestResolveAllowlistAndAliasAmbiguity"),
    ("alias-ambiguity", IDENTITY, 'if match >= 0 {', 'if match >= 0 && selector != "0198f4c8-7d40-7e55-8e6f-1234567890ab" {', "Admits one ID/alias collision", "TestResolveAllowlistAndAliasAmbiguity"),
    ("protocol-grammar", IDENTITY, 'if _, err := scalar.ParseUUIDv7(hostID); err != nil {', 'if _, err := scalar.ParseUUIDv7(hostID); err != nil && hostID != "workstation" {', "Allows one malformed protocol ID through grammar only; exact identity gate subsumes it", None),
    ("protocol-id", IDENTITY, 'if hostID != t.host.ID {', 'if hostID != t.host.ID && hostID != "0198f4c8-a070-7188-9172-1234567890ab" {', "Admits one other allowlisted peer ID", "TestProtocolIdentityRefusal"),
    ("disclosure-class", IDENTITY, 'case "environment_observations", "native_observations", "manual_metadata", "generated_metadata", "job_operation_status":', 'case "environment_observations", "native_observations", "manual_metadata", "generated_metadata", "job_operation_status", "raw_excerpts":', "Admits raw excerpts to the policy class registry", "TestDisclosurePolicyRefusals"),
    ("disclosure-unset", IDENTITY, 'class == "generated_metadata" && d.summaryChoice == "unset"', 'class == "generated_metadata" && d.summaryChoice == "unset" && selector != "0198f4c8-7d40-7e55-8e6f-1234567890ab"', "Admits unset summary choice for one peer", "TestDisclosurePolicyRefusals/unset_summary_choice"),
    ("disclosure-local", IDENTITY, 'policy != "mesh_sanitized" && policy != "reference_only"', 'policy != "mesh_sanitized" && policy != "reference_only" && !(policy == "local_only" && class == "manual_metadata")', "Admits local-only manual metadata", "TestDisclosurePolicyRefusals/default_local_only"),
    ("disclosure-binding", IDENTITY, 'entry.HostID != target.host.ID {', 'entry.HostID != target.host.ID && entry.HostID != "0198f4c8-7d40-7e55-8e6f-1234567890ab" {', "Allows one peer override to affect another peer", "TestSnapshotIsolationAndDisclosureBinding"),
    ("read-error", 'internal/config/loader.go', 'document, err := inputs.ReadFile(filename)\n\tif err != nil {', 'document, err := inputs.ReadFile(filename)\n\tif err != nil && len(document) == 0 {', "Admits full-looking bytes returned together with a failed read", "TestLoadAbsenceReadFailureAndRecovery/partial_read"),
]

# Survivors are explicitly bounded, never represented as behavioral kills.
BOUNDS = {
    "protocol-grammar": "Grammar alone is subsumed by exact target-ID equality; malformed IDs still fail equality. No independent grammar kill claimed.",
}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True)
    opts = parser.parse_args()
    out = Path(opts.output).resolve()
    out.mkdir(parents=True, exist_ok=True)
    results = []
    unexpected = False
    for name, path, old, new, narrows, expected_test in PROBES:
        source = ROOT / path
        original = source.read_bytes()
        text = original.decode()
        if text.count(old) != 1:
            raise RuntimeError(f"{name}: plant must match exactly once, got {text.count(old)}")
        folder = out / name
        folder.mkdir(exist_ok=True)
        replacement = folder / source.name
        replacement.write_text(text.replace(old, new, 1))
        overlay = folder / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(source): str(replacement)}}))
        command = ["go", "test", "-overlay", str(overlay), "./internal/peeridentity", "./internal/config", "-count=1", "-json", "-run", TESTS]
        proc = subprocess.run(command, cwd=ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=120)
        (folder / "test.log").write_bytes(proc.stdout)
        events = []
        for line in proc.stdout.decode(errors="replace").splitlines():
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                pass
        failed = [e["Test"] for e in events if e.get("Action") == "fail" and "Test" in e]
        passed = [e["Test"] for e in events if e.get("Action") == "pass" and "Test" in e]
        killed = proc.returncode != 0 and bool(failed) and (expected_test is None or expected_test in failed)
        if name == "neutral":
            valid = proc.returncode == 0 and bool(passed)
            state = "neutral passed" if valid else "invalid control"
        elif name in BOUNDS:
            valid = proc.returncode == 0 and bool(passed)
            state = "survived" if valid else "unexpected result"
        else:
            valid = killed
            state = "killed" if killed else "survived" if proc.returncode == 0 and passed else "invalid instrument/compile failure"
        if not valid:
            unexpected = True
        if source.read_bytes() != original:
            raise RuntimeError(f"{name}: source changed during overlay run")
        row = {"mutant": name, "narrowing": narrows, "exit_code": proc.returncode, "result": state,
               "named_failing_tests": failed, "expected_test": expected_test,
               "bound": BOUNDS.get(name, "" if killed or name == "neutral" else "No named failing test established; this gate member is unproven."),
               "command": command}
        results.append(row)
        print(f"{name}: {state}, exit={proc.returncode}, tests={','.join(failed)}", flush=True)
        (out / "results.json").write_text(json.dumps(results, indent=2) + "\n")
    rows = ["| Mutant | What the gate is narrowed to admit | Named failing test | Exit | Result / surviving bound |",
            "| --- | --- | --- | ---: | --- |"]
    for r in results:
        tests = r["expected_test"] if r["expected_test"] in r["named_failing_tests"] else ", ".join(r["named_failing_tests"])
        rows.append(f'| {r["mutant"]} | {r["narrowing"]} | {tests or "none"} | {r["exit_code"]} | {r["result"]}: {r["bound"]} |')
    (out / "table.md").write_text("\n".join(rows)+"\n")
    return int(unexpected)


if __name__ == "__main__":
    sys.exit(main())
