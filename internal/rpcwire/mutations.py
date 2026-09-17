#!/usr/bin/env python3
"""RPC structural gate overlays. Every kill requires a named behavioral failure."""
import argparse
import json
from pathlib import Path
import subprocess
import sys
ROOT = Path(__file__).resolve().parents[2]
TESTS = "Test"
E = "internal/rpcwire/envelope.go"
H = "internal/rpcwire/hello.go"
I = "internal/rpcwire/inventory.go"
PROBES = []
BOUNDS = {}
def probe(name,path,old,new,bound,test):
    PROBES.append((name,path,old,new,bound,test))
probe('neutral',E,'return 2, nil','return int(2), nil','Equivalent major value',None)
probe('known-bad',E,'return r.id','return "wrong"','Wrong request accessor','TestPinnedHelloProfiles/2.0.0')
probe('line-plus-one',E,'len(data) > MaxLineBytes','len(data) > MaxLineBytes+1','Admits exactly 8388609 bytes','TestFrameRefusals/oversize')
probe('lf',E,'bytes.ContainsAny(data, "\\n")','(bytes.ContainsAny(data, "\\n") && data[len(data)-1] != \'\\n\')','Admits final LF in a line payload','TestFrameRefusals/newline')
probe('protocol',E,'stringValue(m["protocol"]) != Protocol','stringValue(m["protocol"]) != Protocol && stringValue(m["protocol"]) != "urn:other"','Admits urn:other only','TestRequestRefusals/protocol')
probe('request-uuid',E,'err != nil {\n\t\treturn nil, "", "", ErrFrame\n\t}\n\treturn m, version, id','err != nil && id != "0198f4c8-a070-4188-9172-1234567890ab" {\n\t\treturn nil, "", "", ErrFrame\n\t}\n\treturn m, version, id','Admits one UUIDv4 request ID','TestRequestRefusals/uuid')
probe('closed-extra',E,'len(m) != len(keys)','len(m) != len(keys) && !(len(m) == len(keys)+1 && string(m["extra"]) == "true")','Admits one extra=true member','TestRequestRefusals/extra')
probe('correlation-id',E,'id != request.id','(id != request.id && id != "0198f4c8-a070-7188-9172-2234567890ab")','Admits one mismatched request ID','TestResponseRefusals/id')
probe('correlation-version',E,'version != request.version','(version != request.version && version != "3.0.0")','Admits mismatched response version 3 only','TestResponseRefusals/version')
probe('nonce-echo',E,'h.NonceEcho != sent.Nonce','h.NonceEcho != sent.Nonce && h.NonceEcho != "cXJzdHV2d3h5ejAxMjM0NQ"','Admits the wrong echo fixture only','TestResponseRefusals/nonce-echo')
probe('failure-binding',E,'n, _ := major(version)','n, _ := major(version)\n if n == 3 { n = 2 }','Admits Error 1.0 in RPC3 while retaining other bindings','TestHistoricalFailureBindings/3.0.0')
probe('rejection-shape',E,'if _, ok := m["operation"]; !ok {','if _, ok := m["operation"]; !ok && m["body"] == nil {','Admits request-shaped object without operation','TestBootstrapRejectionFraming/response')
probe('hello-host',H,'err != nil {\n\t\treturn Hello{}, ErrHello\n\t}\n\tif _, err = scalar.ParsePlatform','err != nil && h.HostID != "0198f4c8" {\n\t\treturn Hello{}, ErrHello\n\t}\n\tif _, err = scalar.ParsePlatform','Admits one malformed host ID','TestHelloRefusals/host')
probe('hello-platform',H,'scalar.ParsePlatform(h.Platform); err != nil','scalar.ParsePlatform(h.Platform); err != nil && h.Platform != "ios"','Admits ios only','TestHelloRefusals/platform')
probe('hello-version',H,'!semver.MatchString(h.AXVersion)','(!semver.MatchString(h.AXVersion) && h.AXVersion != "01.0.0")','Admits one leading-zero version','TestHelloRefusals/ax-version')
probe('nonce-short',H,'len(b) >= 16','len(b) >= 15','Admits exactly 120 nonce bits','TestHelloRefusals/nonce-short')
probe('line-floor',H,'h.MaxLineBytes.Uint64() < MaxLineBytes','h.MaxLineBytes.Uint64() < MaxLineBytes-1','Admits line floor minus one','TestHelloRefusals/line-floor')
probe('object-floor',H,'h.MaxObjectBytes.Uint64() < MinObjectBytes','h.MaxObjectBytes.Uint64() < MinObjectBytes-1','Admits object floor minus one','TestHelloRefusals/object-floor')
probe('contract-extra',H,'len(h.Contracts) != len(profile)','len(h.Contracts) != len(profile) && !(len(h.Contracts)==len(profile)+1 && h.Contracts["error"] != nil)','Admits one forbidden error map key','TestHelloRefusals/error-key')
probe('contract-seventeen',H,'len(versions) > 16','len(versions) > 17','Admits exactly 17 versions','TestHelloRefusals/seventeen-versions')
probe('contract-empty',H,'len(versions) < 1','len(versions) < 1 && versions == nil','Admits present empty version array','TestHelloRefusals/empty-versions')
probe('contract-duplicate',H,'versions[i-1] >= v','versions[i-1] > v','Admits duplicate adjacent versions','TestHelloRefusals/duplicate-versions')
probe('contract-unsorted',H,'versions[i-1] >= v','(versions[i-1] >= v && !(versions[i-1] == "2.0.0" && v == "1.0.0"))','Admits one descending adjacent pair','TestHelloRefusals/unsorted-versions')
probe('contract-semver',H,'!semver.MatchString(v)','(!semver.MatchString(v) && v != "v1.0.0")','Admits v1.0.0 version string only','TestHelloRefusals/invalid-version')
probe('v3-exact',H,'version != "2.0.0" && !slices.Equal(versions, expected)','version != "2.0.0" && !slices.Equal(versions, expected) && !(version == "3.0.0" && key == "lease")','Admits nonexact v3 lease array','TestHelloRefusals/exact-map-3.0.0')
probe('v4-exact',H,'version != "2.0.0" && !slices.Equal(versions, expected)','version != "2.0.0" && !slices.Equal(versions, expected) && !(version == "4.0.0" && key == "lease")','Admits nonexact v4 lease array','TestHelloRefusals/exact-map-4.0.0')
probe('v5-exact',H,'version != "2.0.0" && !slices.Equal(versions, expected)','version != "2.0.0" && !slices.Equal(versions, expected) && !(version == "5.0.0" && key == "lease")','Admits nonexact v5 lease array','TestHelloRefusals/exact-map-5.0.0')
probe('offered-pair',H,'response.request.id != request.id','(response.request.id != request.id && request.id != "0198f4c8-a070-7188-9172-2234567890ab")','Admits one cross-request limit pair','TestLimitsAndUntrustedIsolation')
probe('namespace-empty',I,'len(names) < 1','len(names) < 1 && names == nil','Admits nonnil empty namespace array','TestInventoryNamespacesAndCardinality/2.0.0')
probe('namespace-member',I,'!slices.Contains(allowed, name)','(!slices.Contains(allowed, name) && name != "credential")','Admits credential namespace only','TestInventoryNamespacesAndCardinality/2.0.0')
probe('namespace-duplicate',I,'names[i-1] >= name','names[i-1] > name','Admits duplicate namespaces','TestInventoryNamespacesAndCardinality/2.0.0')
probe('namespace-sort',I,'names[i-1] >= name','(names[i-1] >= name && !(names[i-1] == "event" && name == "blob"))','Admits event/blob descending pair','TestInventoryNamespacesAndCardinality/2.0.0')
probe('roots-missing',I,'len(roots) != len(names)','len(roots) != len(names) && len(roots) != len(names)-1','Admits one missing requested root','TestInventoryNamespacesAndCardinality/2.0.0/missing')
probe('root-association',I,'root.Namespace != names[i]','root.Namespace != names[i] && !(i==0 && root.Namespace=="event")','Admits event root in blob slot','TestInventoryNamespacesAndCardinality/2.0.0')

probe('common-data-model',E,'canonicaljson.Canonicalize(data); err != nil','canonicaljson.Canonicalize(data); err != nil && !bytes.HasPrefix(data, []byte(`{"protocol":"urn:other",`))','Admits the duplicate protocol fixture while retaining the canonicalization call','TestFrameRefusals/duplicate')

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
        command = ["go", "test", "-overlay", str(overlay), "./internal/rpcwire", "-count=1", "-json", "-timeout=20s", "-run", TESTS]
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
