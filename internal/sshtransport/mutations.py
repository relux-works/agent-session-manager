#!/usr/bin/env python3
"""SSH transport behavioral overlays; no source-only or compile-error kills."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
TRANSPORT = "internal/sshtransport/transport.go"
UNIX = "internal/sshtransport/process_unix.go"
TESTS = "Test"
PROBES = [
 ("neutral", TRANSPORT, "size := 0", "size := int(0)", "Equivalent size accumulator", None),
 ("known-bad", TRANSPORT, "exec.Command(c.executable, argv...)", 'exec.Command(c.executable, append(argv, "unexpected")...)', "Adds a remote command token", "TestOpenStructuredCommand"),
 ("argv-plus-one", TRANSPORT, "size > 65_536", "size > 65_537", "Admits exactly 65537 composed bytes", "TestOpenComposedArgvBound/65537"),
 ("send-plus-one", TRANSPORT, "len(line) > MaxLineBytes", "len(line) > MaxLineBytes+1", "Admits exactly one excess outbound byte", "TestDuplexAndSendBoundaries/oversize"),
 ("read-plus-one", TRANSPORT, "len(line)+len(part) > MaxLineBytes", "len(line)+len(part) > MaxLineBytes+1", "Admits exactly one excess inbound byte", "TestReceiveFailureAndLimits/oversize"),
 ("send-empty", TRANSPORT, "len(line) == 0 ||", "(len(line) == 0 && line == nil) ||", "Admits a non-nil empty outbound line", "TestDuplexAndSendBoundaries/empty"),
 ("send-two-lines", TRANSPORT, "bytes.IndexByte(line, '\\n') >= 0", '(bytes.IndexByte(line, \'\\n\') >= 0 && string(line) != "{}\\n{}")', "Admits the two-frame outbound fixture", "TestDuplexAndSendBoundaries/newline"),
 ("read-empty", TRANSPORT, "if len(line) == 0 {", "if len(line) == 0 && line != nil {", "Admits the initial empty inbound frame", "TestReceiveFailureAndLimits/blank"),
 ("read-partial", TRANSPORT, "if len(line) != 0 {", 'if len(line) != 0 && string(line) != `{"ok":true}` {', "Treats one unterminated success-shaped frame as EOF", "TestReceiveFailureAndLimits/partial"),
 ("stderr-plus-one", TRANSPORT, "n > StderrLimit", "n > StderrLimit+1", "Admits exactly one excess stderr byte", "TestReceiveFailureAndLimits/stderr-over"),
 ("exit255", TRANSPORT, "if err != nil {\n\t\t\t\tcause = ErrExit", "if err != nil && exitCode != 255 {\n\t\t\t\tcause = ErrExit", "Treats SSH exit 255 alone as successful EOF", "TestReceiveFailureAndLimits/exit255"),
 ("cancel-as-success", TRANSPORT, "if cause != nil {\n\t\ts.result =", "if cause != nil && !errors.Is(cause, context.Canceled) {\n\t\ts.result =", "Suppresses explicit cancellation but keeps other failures", "TestCancellationAndCleanup/stall"),
 ("drain-as-success", TRANSPORT, "if cause != nil {\n\t\ts.result =", "if cause != nil && cause != ErrDrain {\n\t\ts.result =", "Suppresses inherited-pipe failure only", "TestInheritedPipeCleanup/descendant"),
 ("read-as-absence", TRANSPORT, "if cause != nil {\n\t\ts.result =", "if cause != nil && cause != ErrStream {\n\t\ts.result =", "Suppresses OS stream errors only", "TestReadFailureAndRecovery"),
 ("direct-child-only", UNIX, "_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)", "_ = syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)", "Kills direct child but admits inherited pipe descendants", "TestInheritedPipeCleanup/descendant"),
]

PROBES += [
 ("connect-timeout", TRANSPORT, "strconv.FormatUint(c.connectSeconds, 10)", "strconv.FormatUint(c.connectSeconds+1, 10)", "Allows one extra second before connection timeout", "TestOpenNativeSSHPolicy"),
 ("rpc-timeout", TRANSPORT, "context.WithTimeout(ctx, c.rpcTimeout)", "context.WithTimeout(ctx, c.rpcTimeout+time.Second)", "Allows one extra second of process lifetime", "TestCancellationAndCleanup/deadline"),
]

# Effective-policy plants preserve all other options and the host-key gate.
# Native OpenSSH -G evaluates behavior against a conflicting hermetic config.
POLICY = {
 "StrictHostKeyChecking=yes": "StrictHostKeyChecking=accept-new",
 "NoHostAuthenticationForLocalhost=no": "NoHostAuthenticationForLocalhost=yes",
 "VerifyHostKeyDNS=no": "VerifyHostKeyDNS=yes",
 "UpdateHostKeys=no": "UpdateHostKeys=yes",
 "BatchMode=yes": "BatchMode=no",
 "NumberOfPasswordPrompts=0": "NumberOfPasswordPrompts=1",
 "ControlMaster=no": "ControlMaster=auto",
 "ControlPath=none": "ControlPath=/fixture/control",
 "ControlPersist=no": "ControlPersist=1",
 "ClearAllForwardings=yes": "ClearAllForwardings=no",
 "ForwardAgent=no": "ForwardAgent=yes",
 "ForwardX11=no": "ForwardX11=yes",
 "Tunnel=no": "Tunnel=point-to-point",
 "PermitLocalCommand=no": "PermitLocalCommand=yes",
 "RemoteCommand=none": "RemoteCommand=false",
 "RequestTTY=no": "RequestTTY=force",
 "SessionType=default": "SessionType=none",
 "ForkAfterAuthentication=no": "ForkAfterAuthentication=yes",
 "StdinNull=no": "StdinNull=yes",
 "ConnectionAttempts=1": "ConnectionAttempts=2",
}
for old, new in POLICY.items():
 PROBES.append(("policy-"+old.split("=")[0], TRANSPORT, '"'+old+'"', '"'+new+'"', "Admits "+new+" while preserving all other SSH policy", "TestOpenNativeSSHPolicy"))
BOUNDS = {
 "policy-Tunnel": "Native ClearAllForwardings=yes suppresses tunnels too; the ClearAllForwardings mutant is killed. No independent Tunnel-policy kill claimed.",
 "policy-RequestTTY": "The fixed trailing -T from peeridentity overrides RequestTTY; no independent RequestTTY-policy kill claimed.",
}

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
        command = ["go", "test", "-overlay", str(overlay), "./internal/sshtransport", "-count=1", "-json", "-timeout=20s", "-run", TESTS]
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
