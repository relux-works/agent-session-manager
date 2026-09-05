"""Round-3 finding replay for TASK-260830-1snnef.

Replays the reviewer's round-2 survivor mutants against the reworked tree
and requires each to go RED. Unlike the round-1/round-2 runners, restore()
snapshots and hash-verifies EVERY file a mutant touches (any extension,
including internal/traceability/ownership.v0.5.0.json), and asserts the
`git status --porcelain` fingerprint is identical before the first probe
and after the last one, so a JSON-touching mutant (T01 shape) cannot
contaminate the tree.
"""
import hashlib
import json
import os
import subprocess
import sys

PKG = "internal/terminalbackend"
CONF = os.path.join(PKG, "conformance.go")
TB = os.path.join(PKG, "terminalbackend.go")
OWN = "internal/traceability/ownership.v0.5.0.json"

WATCHED = [CONF, TB, OWN]


def sha(path):
    with open(path, "rb") as fh:
        return hashlib.sha256(fh.read()).hexdigest()


def porcelain():
    r = subprocess.run(["git", "status", "--porcelain"],
                       capture_output=True, text=True)
    assert r.returncode == 0, r.stderr
    return r.stdout


BASE_PORCELAIN = porcelain()
BASE_HASH = {f: sha(f) for f in WATCHED}

SNAP = {}


def snapshot(files):
    for f in files:
        if f not in SNAP:
            with open(f, "rb") as fh:
                SNAP[f] = fh.read()


def restore():
    for f, content in SNAP.items():
        with open(f, "wb") as fh:
            fh.write(content)
    for f, content in SNAP.items():
        assert sha(f) == hashlib.sha256(content).hexdigest(), f


def run_tests(pkgs):
    cmd = ["go", "test"] + pkgs + ["-count=1"]
    r = subprocess.run(cmd, capture_output=True, text=True)
    return r.returncode, r.stdout + r.stderr


def probe(pid, desc, edits, newfiles=(), pkgs=None, expect="RED"):
    snapshot([f for f, _, _ in edits])
    restore()
    for f, old, new in edits:
        with open(f) as fh:
            s = fh.read()
        if old not in s:
            restore()
            print("%s\tANCHOR-MISSING\t%s" % (pid, desc))
            return {"id": pid, "desc": desc, "result": "ANCHOR-MISSING",
                    "expect": expect}
        with open(f, "w") as fh:
            fh.write(s.replace(old, new, 1))
    created = []
    for name, body in newfiles:
        path = os.path.join(PKG, name)
        assert not os.path.exists(path), path
        with open(path, "w") as fh:
            fh.write(body)
        created.append(path)
    code, out = run_tests(pkgs or ["./internal/terminalbackend/"])
    fails = sorted({l.split()[2] for l in out.splitlines()
                    if l.strip().startswith("--- FAIL:")
                    and len(l.split()) > 2})
    # A t.Fatal in a derivation helper prints "FAIL:" with the test name;
    # build failures print no --- FAIL lines. Both are RED.
    verdict = "RED" if code != 0 else "GREEN"
    for path in created:
        os.remove(path)
    restore()
    status = "OK" if verdict == expect else "*** UNEXPECTED ***"
    print("%s\t%s\t(expect %s) %s\t%s" % (pid, verdict, expect, status,
                                             desc))
    if fails:
        print("      failing: %s" % ", ".join(fails[:8]))
    return {"id": pid, "desc": desc, "result": verdict, "expect": expect,
            "failing": fails}


ATTACHABLE = ("\tif result.Attachable && "
              "!((result.State == StateParked || result.State == StateActive)"
              " && result.AttachEvidenced) {")


def widen_state(state):
    return (ATTACHABLE,
            ATTACHABLE.replace("|| result.State == StateActive)",
                               "|| result.State == StateActive || "
                               "result.State == %s)" % state))


MUTANTS = [
    # G1: attachability widenings, one per excluded state (C13/C26-C29),
    # plus the stopped control (C30). All must be RED.
    ("G1-quiescing", "attachability admits quiescing",
     [dict(f=CONF, old=widen_state("StateQuiescing")[0],
           new=widen_state("StateQuiescing")[1])]),
    ("G1-creating", "attachability admits creating",
     [dict(f=CONF, old=widen_state("StateCreating")[0],
           new=widen_state("StateCreating")[1])]),
    ("G1-absent", "attachability admits absent",
     [dict(f=CONF, old=widen_state("StateAbsent")[0],
           new=widen_state("StateAbsent")[1])]),
    ("G1-unavailable", "attachability admits unavailable",
     [dict(f=CONF, old=widen_state("StateUnavailable")[0],
           new=widen_state("StateUnavailable")[1])]),
    ("G1-stale_fenced", "attachability admits stale_fenced",
     [dict(f=CONF, old=widen_state("StateStaleFenced")[0],
           new=widen_state("StateStaleFenced")[1])]),
    ("G1-stopped-ctrl", "CONTROL: attachability admits stopped",
     [dict(f=CONF, old=widen_state("StateStopped")[0],
           new=widen_state("StateStopped")[1])]),
    # G2: plain-error smuggling through package-level vars (W5b/W9b/W16),
    # the FuncDecl control (W15), a new-file control (W12), and the unused
    # var shape (W18). All must be RED after the twin-walk fix.
    ("G2-var-errors-New", "package var errors.New returned from production",
     [dict(f=TB,
           old=("func validateProtocolVersions(backendID string, "
                "versions []string) error {"),
           new=("var errPlanted = errors.New(\"planted plain refusal\")\n\n"
                "func validateProtocolVersions(backendID string, "
                "versions []string) error {\n"
                "\tif len(versions) == 7777 {\n"
                "\t\treturn errPlanted\n"
                "\t}"))]),
    ("G2-var-fmt-Errorf", "package var fmt.Errorf returned from production",
     [dict(f=TB,
           old=("func validateProtocolVersions(backendID string, "
                "versions []string) error {"),
           new=("var errPlanted2 = fmt.Errorf(\"planted fmt refusal\")\n\n"
                "func validateProtocolVersions(backendID string, "
                "versions []string) error {\n"
                "\tif len(versions) == 7777 {\n"
                "\t\treturn errPlanted2\n"
                "\t}"))]),
    ("G2-var-func-literal", "plain error in package var func literal",
     [dict(f=TB,
           old=("func validateProtocolVersions(backendID string, "
                "versions []string) error {"),
           new=("var plantedFn = func() error { "
                "return errors.New(\"planted plain refusal in a var "
                "func literal\") }\n\n"
                "func validateProtocolVersions(backendID string, "
                "versions []string) error {\n"
                "\tif len(versions) == 7777 {\n"
                "\t\treturn plantedFn()\n"
                "\t}"))]),
    ("G2-funcdecl-ctrl", "CONTROL: identical errors.New in FuncDecl body",
     [dict(f=TB,
           old=("func validateProtocolVersions(backendID string, "
                "versions []string) error {"),
           new=("func validateProtocolVersions(backendID string, "
                "versions []string) error {\n"
                "\tif len(versions) == 7777 {\n"
                "\t\treturn errors.New(\"planted plain refusal\")\n"
                "\t}"))]),
    ("G2-unused-var", "package var errors.New never returned",
     [dict(f=TB,
           old=("func validateProtocolVersions(backendID string, "
                "versions []string) error {"),
           new=("var errUnusedPlant = errors.New(\"planted unused plain "
                "refusal\")\n\nvar _ = errUnusedPlant\n\n"
                "func validateProtocolVersions(backendID string, "
                "versions []string) error {"))]),
    # G3: replication table widenings. C40c shape (add), init injection,
    # and the two reclassification controls (all RED).
    ("G3-add-member", "replicationMembers gains ax_pane_pid as safe",
     [dict(f=CONF,
           old=("\"conformance_fixture_id\", \"capability_claims\", "
                "\"evidence_ids\","),
           new=("\"conformance_fixture_id\", \"capability_claims\", "
                "\"evidence_ids\", \"ax_pane_pid\","))]),
    ("G3-init-injection", "init() injects ax_pane_pid as safe evidence",
     [dict(f=CONF, old="func CheckReplicable(members []string) error {",
           new=("func init() { replicationMembers[\"ax_pane_pid\"] = "
                "ReplicationSafeEvidence }\n\n"
                "func CheckReplicable(members []string) error {"))]),
    ("G3-reclassify-ctrl", "CONTROL: native_pid forbidden -> safe",
     [dict(f=CONF,
           old=("\"backend_generation_digest\", \"capability\", "
                "\"dependent_operations\","),
           new=("\"backend_generation_digest\", \"capability\", "
                "\"dependent_operations\", \"native_pid\",")),
      dict(f=CONF,
           old="\"native_pid\", \"process_handle\", \"endpoint\", \"token\",",
           new="\"process_handle\", \"endpoint\", \"token\",")]),
    ("G3-promote-ctrl", "CONTROL: attach_credential sensitive -> safe",
     [dict(f=CONF,
           old=("\"backend_generation_digest\", \"capability\", "
                "\"dependent_operations\","),
           new=("\"backend_generation_digest\", \"capability\", "
                "\"dependent_operations\", \"attach_credential\",")),
      dict(f=CONF,
           old="\"named_pipe\", \"attach_credential\", \"relay_credential\",",
           new="\"named_pipe\", \"relay_credential\",")]),
    # G4: vocabulary widenings as conversions (fail the derivation) plus
    # the running control. All must be RED.
    ("G4-detached", "ParseInstanceState admits detached",
     [dict(f=CONF,
           old=("\t\tStateQuiescing, StateStopped, StateStaleFenced, "
                "StateUnavailable:\n"
                "\t\treturn InstanceState(value), nil"),
           new=("\t\tStateQuiescing, StateStopped, StateStaleFenced, "
                "StateUnavailable, InstanceState(\"detached\"):\n"
                "\t\treturn InstanceState(value), nil"))]),
    ("G4-running-ctrl", "CONTROL: ParseInstanceState admits running",
     [dict(f=CONF,
           old=("\t\tStateQuiescing, StateStopped, StateStaleFenced, "
                "StateUnavailable:\n"
                "\t\treturn InstanceState(value), nil"),
           new=("\t\tStateQuiescing, StateStopped, StateStaleFenced, "
                "StateUnavailable, InstanceState(\"running\"):\n"
                "\t\treturn InstanceState(value), nil"))]),
    ("G4-detach", "ParseOperation admits detach",
     [dict(f=CONF,
           old=("\t\tOperationRequestStop, OperationTerminateStale, "
                "OperationRestore:\n"
                "\t\treturn Operation(value), nil"),
           new=("\t\tOperationRequestStop, OperationTerminateStale, "
                "OperationRestore, Operation(\"detach\"):\n"
                "\t\treturn Operation(value), nil"))]),
    ("G4-input_reopened", "ParseSideEffect admits input_reopened",
     [dict(f=CONF,
           old=("\tcase EffectBindingPersisted, EffectWrapperStarted, "
                "EffectAttachClientCreated,"),
           new=("\tcase SideEffect(\"input_reopened\"), "
                "EffectBindingPersisted, EffectWrapperStarted, "
                "EffectAttachClientCreated,"))]),
    ("G4-ssh_tunnel", "parseTransport admits ssh_tunnel",
     [dict(f=CONF,
           old=("\tcase TransportLocalOnly, TransportTrustedPrivateMesh, "
                "TransportThirdPartyRelay:"),
           new=("\tcase TransportLocalOnly, TransportTrustedPrivateMesh, "
                "TransportThirdPartyRelay, "
                "PresentationTransport(\"ssh_tunnel\"):"))]),
    # F2 guard: error-mapping widening must stay RED (no production change).
    ("F2-guard", "CheckErrorAllowed admits unavailable for every operation",
     [dict(f=CONF, old="\tfor _, allowed := range allowedOperationErrors"
                       "[parsedOperation] {",
           new=("\tif code == \"terminal_backend_unavailable\" {\n"
                "\t\treturn nil\n"
                "\t}\n"
                "\tfor _, allowed := range allowedOperationErrors"
                "[parsedOperation] {"))]),
]


def main():
    results = []
    for pid, desc, edits in MUTANTS:
        results.append(probe(pid, desc,
                             [(e["f"], e["old"], e["new"]) for e in edits]))
    # Tooling-note self-test: touch the JSON file the old runner never
    # restored, run nothing, restore, and require the hash back.
    snapshot([OWN])
    restore()
    with open(OWN) as fh:
        s = fh.read()
    anchor = "terminal-lifecycle-conformance"
    assert anchor in s, "T01 anchor moved; self-test is stale"
    with open(OWN, "w") as fh:
        fh.write(s.replace(anchor, "terminal-lifecycle-CONTAMINATED", 1))
    assert sha(OWN) != BASE_HASH[OWN]
    restore()
    json_ok = sha(OWN) == BASE_HASH[OWN]
    print("TOOLING-RESTORE-JSON\t%s\t(non-Go restore self-test)" %
          ("OK" if json_ok else "*** CONTAMINATED ***"))
    # New-file cleanup check is inside probe(); final tree identity:
    restore()
    assert porcelain() == BASE_PORCELAIN, "tree fingerprint moved"
    for f, h in BASE_HASH.items():
        assert sha(f) == h, f
    print("TREE-IDENTITY\tOK\t(porcelain + watched hashes identical)")
    killed = [r["id"] for r in results if r["result"] == "RED"]
    survived = [r["id"] for r in results if r["result"] == "GREEN"]
    missing = [r["id"] for r in results
               if r["result"] not in ("RED", "GREEN")]
    print("\nROUND-3 REPLAY: total=%d killed=%d survived=%d "
          "anchor_missing=%d" % (len(results), len(killed), len(survived),
                                 len(missing)))
    print("SURVIVED: %s" % survived)
    print("MISSING: %s" % missing)
    out = {"results": results, "json_restore_ok": json_ok,
           "tree_identity_ok": True}
    with open(".temp/TASK-260830-1snnef-r3/replay-results.json", "w") as fh:
        json.dump(out, fh, indent=1)
    if survived or missing or not json_ok:
        sys.exit(1)


if __name__ == "__main__":
    main()
