#!/usr/bin/env python3
"""Mesh major-negotiation gate overlays. Every kill requires a named behavioral failure.

Run with PYTHONDONTWRITEBYTECODE=1 so the harness leaves no __pycache__ behind.
Each probe applies one narrowing plant to a pristine overlay of
internal/meshneg/negotiate.go and executes the full package behavioral suite
against it. A narrowing plant keeps the gate present and weakens it to admit
exactly one member of the class the gate must reject.
"""
import argparse
import json
from pathlib import Path
import subprocess
import sys
ROOT = Path(__file__).resolve().parents[2]
TESTS = "Test"
N = "internal/meshneg/negotiate.go"
PROBES = []
BOUNDS = {
    "detail-text": "Human Detail text only; no test branches on it (harmless applied control, must survive).",
}
def probe(name,path,old,new,bound,test):
    PROBES.append((name,path,old,new,bound,test))
probe('neutral',N,'return []int{2}, nil','return []int{int(2)}, nil','Equivalent major value',None)
probe('config1-admits-3',N,'return []int{2}, nil','return []int{2, 3}, nil','Admits major 3 for Config-1 only','TestNegotiateMatrix/1.0.0-x-3.0.0')
probe('config2-admits-4',N,'return []int{2, 3}, nil','return []int{2, 3, 4}, nil','Admits major 4 for Config-2 only','TestNegotiateMatrix/2.0.0-x-4.0.0')
probe('config3-admits-5',N,'return []int{2, 3, 4}, nil','return []int{2, 3, 4, 5}, nil','Admits major 5 for Config-3 only','TestNegotiateMatrix/3.0.0-x-5.0.0')
probe('config4-admits-4',N,'return []int{5}, nil','return []int{4, 5}, nil','Admits major 4 for Config-4 only','TestConfig4NeverDowngrades')
probe('unknown-config-admits-9',N,'case "4.0.0":','case "4.0.0", "9.9.9":','Admits generation 9.9.9 as Config-4 only','TestLocalMajors/unknown-9.9.9')
probe('frame-admits-2.1',N,'"2.0.0": 2,','"2.0.0": 2,\n\t"2.1.0": 2,','Admits frame 2.1.0 as major 2 only','TestPeerOfferRefusals/frame-versions')
probe('shape-admits-extra-key',N,'if len(hello.Contracts) != len(profile) {','if len(hello.Contracts) != len(profile) && len(hello.Contracts) != len(profile)+1 {','Admits exactly one extra contracts key','TestPeerOfferRefusals/shape/extra-error-key')
probe('shape-admits-missing-lease',N,'if _, ok := hello.Contracts[key]; !ok {','if _, ok := hello.Contracts[key]; !ok && key != "lease" {','Admits a missing lease key only','TestPeerOfferRefusals/shape/substituted-key')
probe('rpc-admits-17',N,'len(offered) > 16','len(offered) > 17','Admits exactly 17 rpc versions','TestPeerOfferRefusals/shape/seventeen-rpc')
probe('rpc-admits-nil-empty',N,'len(offered) < 1','(len(offered) < 1 && offered != nil)','Admits a nil rpc array while still refusing an empty non-nil one','TestPeerOfferRefusals/shape/empty-rpc')
probe('mixed-admits-v3-in-v2',N,'if major != framing {','if major != framing && !(major == 3 && framing == 2) {','Admits major 3 inside a major-2 frame only','TestPeerOfferRefusals/mixed/two-majors-in-v2')
probe('mixed-admits-unsorted-pair',N,'offered[index-1] >= version','(offered[index-1] >= version && !(offered[index-1] == "2.1.0" && version == "2.0.0"))','Admits the 2.1.0/2.0.0 descending pair only','TestPeerOfferRefusals/mixed/unsorted')
probe('mixed-admits-v-prefix-value',N,'major, err := majorOf(version)\n\t\tif err != nil {\n\t\t\treturn 0, mixedRefusal(fmt.Sprintf("rpc version %q is not strict semver", version))\n\t\t}','major, err := majorOf(version)\n\t\tif err != nil && version != "v2.0.0" {\n\t\t\treturn 0, mixedRefusal(fmt.Sprintf("rpc version %q is not strict semver", version))\n\t\t}\n\t\tif version == "v2.0.0" {\n\t\t\tmajor, err = framing, nil\n\t\t}','Admits the value v2.0.0 as the framing major only','TestPeerOfferRefusals/mixed/not-semver')
probe('v3-exact-admits-3.0.1',N,'if frameVersion != "2.0.0" && !slices.Equal(offered, profile["rpc"]) {','if frameVersion != "2.0.0" && !slices.Equal(offered, profile["rpc"]) && !(frameVersion == "3.0.0" && slices.Equal(offered, []string{"3.0.1"})) {','Admits rpc [3.0.1] in a v3 frame only','TestPeerOfferRefusals/shape/v3-inexact-rpc')
probe('select-admits-local-6',N,'for _, major := range local {\n\t\tif major < 2 || major > 5 {','for _, major := range local {\n\t\tif (major < 2 || major > 5) && major != 6 {','Admits local major 6 only','TestSelectHighestCommon/refuse-local-names-6')
probe('select-admits-peer-1',N,'for _, major := range peer {\n\t\tif major < 2 || major > 5 {','for _, major := range peer {\n\t\tif (major < 2 || major > 5) && major != 1 {','Admits peer major 1 only','TestSelectHighestCommon/refuse-peer-mixes-1-with-common')
probe('select-picks-lowest',N,'candidate > best','(candidate < best || best == 0)','Selects the lowest common major instead of the highest','TestSelectHighestCommon/admit-highest-wins')
probe('select-admits-disjoint-2v3',N,'if best == 0 {','if best == 0 && !(len(local) == 1 && local[0] == 2 && len(peer) == 1 && peer[0] == 3) {','Admits the disjoint pair 2-against-3 as success','TestSelectHighestCommon/refuse-disjoint')
probe('exposure-directory-at-2',N,'major >= directoryFeatureMajor','major >= directoryFeatureMajor-1','Reports directory negotiated at major 2','TestPeerExposure/2.0.0')
probe('exposure-backend-at-3',N,'major >= backendEvidenceFeatureMajor','major >= backendEvidenceFeatureMajor-1','Reports backend evidence negotiated at major 3','TestPeerExposure/3.0.0')
probe('error-binding-provider-contract',N,'"urn:ax:protocol:rpc"','"urn:ax:protocol:provider"','Resolves the error binding under the provider contract','TestDecisionShapes/3.0.0')
probe('refusal-code-transport-failure',N,'Code:     axerror.Code("incompatible_protocol"),','Code:     axerror.Code("transport_failure"),','Peer refusals carry transport_failure instead of incompatible_protocol','TestNegotiateMatrix/3.0.0-x-5.0.0')
probe('directory-version-1.0.0',N,'DirectoryUnsupportedCode, axerror.Version120','DirectoryUnsupportedCode, axerror.Version100','Resolves directory exposure under Error 1.0.0','TestPeerExposure/2.0.0')
probe('backend-version-1.2.0',N,'BackendEvidenceUnsupportedCode, axerror.Version130','BackendEvidenceUnsupportedCode, axerror.Version120','Resolves backend exposure under Error 1.2.0','TestPeerExposure/2.0.0')
probe('major-extract-plus-one',N,'major, err := strconv.Atoi(match[1])','major, err := strconv.Atoi(match[1])\n\tmajor++','Version grammar unchanged; extracted major shifted by one','TestNegotiateMatrix')
probe('semver-admits-v-prefix',N,'^(0|[1-9][0-9]*)\\.','^v?(0|[1-9][0-9]*)\\.','Admits v-prefixed versions as their major','TestPeerOfferRefusals/mixed/not-semver')
probe('reason-shape-reports-mixed',N,'refuse(ReasonContractShape, ErrContractShape,','refuse(ReasonMixedMajor, ErrContractShape,','Shape refusals misreport as mixed_major','TestPeerOfferRefusals/shape/extra-error-key')
probe('exposure-directory-code-swap',N,'DirectoryUnsupportedCode = axerror.Code("directory_mesh_unsupported")','DirectoryUnsupportedCode = axerror.Code("continuation_route_unavailable")','Directory exposure carries continuation_route_unavailable (same 1.2.0/exit 6) instead of the Section 11.8 token','TestPeerExposure/2.0.0')
probe('exposure-backend-code-swap',N,'BackendEvidenceUnsupportedCode = axerror.Code("terminal_backend_unavailable")','BackendEvidenceUnsupportedCode = axerror.Code("terminal_backend_capability_unproven")','Backend exposure carries terminal_backend_capability_unproven (same 1.3.0/exit 6) instead of the bound RPC-4 token','TestPeerExposure/2.0.0')
probe('negotiate-unknown-falls-back',N,'\tlocal, err := LocalMajors(configVersion)\n\tif err != nil {\n\t\treturn Decision{}, err\n\t}','\tlocal, err := LocalMajors(configVersion)\n\tif err != nil {\n\t\tlocal = []int{2, 3, 4}\n\t}','Unknown configuration falls back to legacy [2 3 4] at the Negotiate site instead of refusing','TestNegotiateUnknownGeneration')
probe('frame-admits-5.1',N,'"5.0.0": 5,','"5.0.0": 5,\n\t"5.1.0": 5,','Admits frame 5.1.0 as major 5 only','TestPeerOfferRefusals/frame-versions')
probe('shape-admits-missing-checkpoint',N,'if _, ok := hello.Contracts[key]; !ok {','if _, ok := hello.Contracts[key]; !ok && key != "checkpoint" {','Admits a missing checkpoint key only','TestPeerOfferKeyMembershipPerKey/substituted-2.0.0-checkpoint')
probe('shape-admits-missing-task-board-bundle',N,'if _, ok := hello.Contracts[key]; !ok {','if _, ok := hello.Contracts[key]; !ok && key != "task_board_bundle" {','Admits a missing task_board_bundle key only','TestPeerOfferKeyMembershipPerKey/substituted-2.0.0-task_board_bundle')
probe('shape-admits-missing-directory-receipt',N,'if _, ok := hello.Contracts[key]; !ok {','if _, ok := hello.Contracts[key]; !ok && key != "session_directory_operation_receipt" {','Admits a missing session_directory_operation_receipt key only','TestPeerOfferKeyMembershipPerKey/substituted-3.0.0-session_directory_operation_receipt')
probe('shape-admits-missing-backend-evidence',N,'if _, ok := hello.Contracts[key]; !ok {','if _, ok := hello.Contracts[key]; !ok && key != "terminal_backend_evidence" {','Admits a missing terminal_backend_evidence key only','TestPeerOfferKeyMembershipPerKey/substituted-4.0.0-terminal_backend_evidence')
probe('negotiate-drops-local-fact',N,'refusal.Local = append([]int(nil), local...)','refusal.Local = nil','Drops the local offer fact on an offer-originated refusal','TestNegotiateOfferRefusalCarriesLocal/higher-major-in-v2')
probe('select-admits-local-1',N,'for _, major := range local {\n\t\tif major < 2 || major > 5 {','for _, major := range local {\n\t\tif (major < 2 || major > 5) && major != 1 {','Admits local major 1 only','TestSelectHighestCommon/refuse-local-names-1')
probe('detail-text',N,'"local and peer offers share no major ("+joinMajors(local)+" against "+joinMajors(peer)+")"','"no overlap ("+joinMajors(local)+" vs "+joinMajors(peer)+")"','Human Detail text only',None)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True)
    parser.add_argument("--only", help="Comma-separated probe names for a bounded follow-up")
    opts = parser.parse_args()
    if opts.only and not set(opts.only.split(",")).issubset({p[0] for p in PROBES}):
        parser.error("--only contains an unknown probe")
    out = Path(opts.output).resolve()
    out.mkdir(parents=True, exist_ok=True)
    results = []
    unexpected = False
    for name, path, old, new, narrows, expected_test in PROBES:
        if opts.only and name not in opts.only.split(","):
            continue
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
        command = ["go", "test", "-overlay", str(overlay), "./internal/meshneg", "-count=1", "-json", "-timeout=60s", "-run", TESTS]
        proc = subprocess.run(command, cwd=ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=180)
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
            "| --- | --- | --- | --- | --- |"]
    for r in results:
        tests = r["expected_test"] if r["expected_test"] in r["named_failing_tests"] else ", ".join(r["named_failing_tests"])
        rows.append(f'| {r["mutant"]} | {r["narrowing"]} | {tests or "none"} | {r["exit_code"]} | {r["result"]}: {r["bound"]} |')
    (out / "table.md").write_text("\n".join(rows)+"\n")
    return int(unexpected)


if __name__ == "__main__":
    sys.exit(main())
