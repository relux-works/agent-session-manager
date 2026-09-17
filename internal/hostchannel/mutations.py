#!/usr/bin/env python3
"""Host-channel gate overlays. Every kill requires a named behavioral failure.

Each probe applies one or more exact-once edits to a pristine source file,
then runs the FULL behavioral suite (never a static checker alone) under a
go overlay; the sources on disk are never modified.
"""
import argparse
import json
from pathlib import Path
import subprocess
import sys
ROOT = Path(__file__).resolve().parents[2]
TESTS = "Test"
C = "internal/hostchannel/channel.go"
S = "internal/hostchannel/server.go"
CL = "internal/hostchannel/client.go"
RW_HELLO = "internal/rpcwire/hello.go"
RW_ENVELOPE = "internal/rpcwire/envelope.go"
AX_DECODE = "internal/axerror/decode.go"
HT_TRANSACT = "internal/hosttrust/transact.go"
PROBES = []
BOUNDS = {
    'client-cache': 'One-sided weakening is harmless: a caching client cannot resume against a server that issues no tickets, so the resumption suite still passes.',
}
def probe(name, path, edits, narrows, expected_test):
    PROBES.append((name, path, edits, narrows, expected_test))
def one(old, new):
    return [(old, new)]

ZSPKI = "sha256:" + "0" * 64
probe('neutral', C, one('total := 2', 'total := 1 + 1'), 'Equivalent authority size', None)
probe('known-bad', C, one('return parsed.String() + HostSuffix', 'return "x-" + parsed.String() + HostSuffix'), 'Wrong server name', 'TestServerNameMatchesIssuedLeaf')
probe('alpn-supplement', C, one('if state.NegotiatedProtocol != ALPN {', 'if state.NegotiatedProtocol != ALPN && state.NegotiatedProtocol != "rogue-alpn" {'), 'Admits exactly rogue-alpn', 'TestVerifyPeerRefusals/alpn')
probe('didresume', C, one('if state.DidResume {', 'if state.DidResume && len(state.VerifiedChains) == 0 {'), 'Admits resumed connections with chains', 'TestVerifyPeerRefusals/resumed')
probe('chains', C, one('if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 {', 'if len(state.VerifiedChains) == 0 && len(state.PeerCertificates) == 0 {'), 'Admits empty chains with a presented peer certificate', 'TestVerifyPeerRefusals/no-chains')
probe('match-leaf', C, one('if !bytes.Equal(entry.LeafDER, leaf.Raw) {', 'if !bytes.HasPrefix(leaf.Raw, entry.LeafDER[:len(entry.LeafDER)-1]) {'), 'Admits a leaf differing only in its last byte', 'TestVerifyPeerRefusals/leaf-flipped')
probe('match-root', C, one('if !bytes.Equal(entry.RootDER, root.Raw) {', 'if !bytes.HasPrefix(root.Raw, entry.RootDER[:len(entry.RootDER)-1]) {'), 'Admits a root differing only in its last byte', 'TestVerifyPeerRefusals/root-flipped')
probe('match-spki', C, one('if entry.SPKIID.String() != spki {', 'if entry.SPKIID.String() != spki && entry.SPKIID.String() != "%s" {' % ZSPKI), 'Admits exactly the zero SPKI', 'TestVerifyPeerRefusals/spki-zero')
probe('match-count', C, one('if len(matched) != 1 {', 'if len(matched) != 1 && len(matched) != 2 {'), 'Admits exactly a double match', 'TestVerifyPeerRefusals/double-match')
probe('state-revoked', C, one('\t\tdefault:\n\t\t\treturn ErrPeerTrustState', '\t\tcase hosttrust.EntryRevoked:\n\t\tdefault:\n\t\t\treturn ErrPeerTrustState'), 'Admits revoked entries while refusing other states', 'TestVerifyPeerRefusals/revoked')
probe('state-retiring', C, one('if err != nil || !now.Before(retireAt) {', 'if err != nil || !now.Before(retireAt.Add(time.Hour)) {'), 'Admits entries up to one hour past retire_at', 'TestVerifyPeerRefusals/retiring-past')
probe('profile', C, one('if err := hosttrust.VerifyProfile(entry.LeafDER, entry.RootDER, entry.HostID.String(), now); err != nil {\n\t\t\treturn fmt.Errorf("%w: %w", ErrPeerProfile, err)', 'if err := hosttrust.VerifyProfile(entry.LeafDER, entry.RootDER, entry.HostID.String(), now); err != nil && entry.State != hosttrust.EntryActive {\n\t\t\treturn fmt.Errorf("%w: %w", ErrPeerProfile, err)'), 'Admits profile-invalid active entries', 'TestVerifyPeerRefusals/expired')
probe('hello-id-server', S, one('\t\tHelloHostID:        hello.HostID,\n\t\tremoteLeafDER:      peerLeaf,', '\t\tHelloHostID:        peerHost,\n\t\tremoteLeafDER:      peerLeaf,'), 'Substitutes the verified UUID for the received hello ID', 'TestRefusesHelloUUIDMismatch')
probe('hello-id-client', CL, [
    ('\t\tHelloHostID:        hello.HostID,\n\t\tremoteLeafDER:      verified.leafDER,', '\t\tHelloHostID:        config.ExpectedRemoteHostID,\n\t\tremoteLeafDER:      verified.leafDER,'),
    ('\thello, err := response.Hello()\n\tif err != nil {', '\thello, err := response.Hello()\n\t_ = hello\n\tif err != nil {'),
], 'Substitutes the expected destination for the received hello ID', 'TestRefusesServerHelloUUIDMismatch')
probe('non-hello', S, one('if request.Operation() != "hello" {', 'if request.Operation() != "hello" && request.Operation() != "health.get" {'), 'Admits health.get before hello', 'TestRefusesNonHelloBeforeHello')
probe('stale-dispatch', S, one('if fresh.Generation != binding.Generation {', 'if fresh.Generation != binding.Generation && fresh.Generation != binding.Generation+1 {'), 'Admits exactly one generation past the binding', 'TestStaleGenerationAtDispatch/server_closes')
probe('stale-mutation', S, one('return config.Store.WithMutationAuthorization(binding.authRequest(config.Allowlisted, binding.RemoteHostID, now()), boundary)', 'fresh, freshErr := config.Store.ReadSnapshot()\n\t\t\tif freshErr != nil {\n\t\t\t\treturn freshErr\n\t\t\t}\n\t\t\trebound := binding\n\t\t\trebound.Generation = fresh.Generation\n\t\t\treturn config.Store.WithMutationAuthorization(rebound.authRequest(config.Allowlisted, binding.RemoteHostID, now()), boundary)'), 'Refreshes the stale binding to the current generation before the boundary', 'TestStaleGenerationAtMutation')
probe('handshake-cap', C, one('if limiter.count > limiter.limit {', 'if limiter.count > limiter.limit+1 {'), 'Admits exactly cap+1 handshake bytes', 'TestHandshakeLimiterBounds')
probe('ca-bound', C, one('return total > MaxAuthorityBytes', 'return total > MaxAuthorityBytes+1'), 'Admits exactly a 65536-byte authority list', 'TestAuthorityListOverflow')
probe('tls-min', C, one('MinVersion:         tls.VersionTLS13,', 'MinVersion:         tls.VersionTLS12,'), 'Admits TLS 1.2 in both roles', 'TestRefusesTLS12Offer/server')
probe('tls-min-client', C, one('MinVersion:         tls.VersionTLS13,', 'MinVersion:         tls.VersionTLS12,'), 'Admits TLS 1.2 in both roles', 'TestRefusesTLS12Offer/client')
probe('tickets', C, one('config.SessionTicketsDisabled = true', 'config.SessionTicketsDisabled = false'), 'Issues session tickets for resumption', 'TestRefusesResumption')
probe('client-cache', C, one('config.ClientSessionCache = nil', 'config.ClientSessionCache = tls.NewLRUClientSessionCache(1)'), 'Caches client sessions while the server issues no tickets', None)
probe('alpn-same-family', C, one('if state.NegotiatedProtocol != ALPN {', 'if state.NegotiatedProtocol != ALPN && state.NegotiatedProtocol != "ax-host/2" {'), 'Admits exactly the same-family ax-host/2 ALPN', 'TestVerifyPeerRefusals/alpn-same-family')
probe('oversize-dispatch', C, [
    (C, 'if len(line)+len(chunk) > rpcwire.MaxLineBytes+1 {', 'if len(line)+len(chunk) > rpcwire.MaxLineBytes+11 {'),
    (RW_ENVELOPE, 'if len(data) == 0 || len(data) > MaxLineBytes || bytes.ContainsAny(data, "\\n") {', 'if len(data) == 0 || len(data) > MaxLineBytes+11 || bytes.ContainsAny(data, "\\n") {'),
], 'Admits exactly an 8 MiB+10 line at framing and decode', 'TestHostileOversizedFrames/dispatch_line_8mib_plus_one')
probe('stale-client-rebind', CL, one('if fresh.Generation != client.binding.Generation {', 'if fresh.Generation != client.binding.Generation && fresh.Generation != client.binding.Generation+1 {'), 'Admits exactly one generation past the client binding', 'TestHostileRecoveryBypass/stale_binding_never_rebinds')
probe('rpc-contracts', RW_HELLO, one('if version != "2.0.0" && !slices.Equal(versions, expected) {', 'if version != "2.0.0" && !slices.Equal(versions, expected) && key != "rpc" {'), 'Admits any contracts.rpc disclosure', 'TestHostileDisclosureMismatch/responder_contracts_drift')
probe('error-version-drift', AX_DECODE, one('if candidate != expected {', 'if candidate != expected && candidate != "1.2.0" {'), 'Honors exactly a 1.2.0 error as 1.3.0', 'TestHostileDisclosureMismatch/error_version_drift')
probe('truncated-dispatch-ignored', S, one('\t\trequest, err := rpcwire.DecodeRequest(line)\n\t\tif err != nil {\n\t\t\t_ = stream.Close()\n\t\t\treturn fmt.Errorf("%w: %w", ErrUnframeableInput, err)\n\t\t}', '\t\trequest, err := rpcwire.DecodeRequest(line)\n\t\tif err != nil {\n\t\t\tcontinue\n\t\t}'), 'Ignores one truncated dispatch frame instead of closing unframeable', 'TestHostileDisconnectPhases/mid_request_close')
probe('snapshot-integrity', HT_TRANSACT, one('\tstore, err := DecodeTrust(document)\n\tif err != nil {\n\t\treturn Snapshot{}, err\n\t}', '\tstore, err := DecodeTrust(document)\n\tif err != nil {\n\t\treturn Snapshot{Generation: 1}, nil\n\t}'), 'Reads a corrupt store as an empty store', 'TestHostileStaleReadsAreIntegrityFailures/corrupt_store')
probe('correlation-health', RW_ENVELOPE, one('if id != request.id {', 'if id != request.id && request.operation != "health.get" {'), 'Accepts a miscorrelated health.get answer', 'TestHostileRecoveryBypass/response_id_mismatch_refused')
probe('nonce-static', RW_HELLO, one('\t_, _ = rand.Read(nonce[:]) // crypto/rand.Read is infallible in Go 1.25.', '\t_, _ = rand.Read(nonce[:]) // crypto/rand.Read is infallible in Go 1.25.\n\tclear(nonce[:])'), 'Mints one constant hello nonce', 'TestHostileReplayNonceFreshness')
probe('census-evasion', S, [
    ('\t"errors"\n\t"fmt"\n\t"time"', '\t"errors"\n\t"fmt"\n\t"os"\n\t"syscall"\n\t"time"'),
    ('\treturn serveLoop(frames, stream, conn, config, binding)',
     '\tfd, _ := syscall.Open(os.TempDir()+"/hc-evict", syscall.O_CREAT|syscall.O_WRONLY, 0600)\n\tsyscall.Write(fd, []byte("x"))\n\tsyscall.Close(fd)\n\treturn serveLoop(frames, stream, conn, config, binding)'),
], 'Writes a temp file through tokens the static census does not list', 'TestRuntimeWriteCensus')

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
    for name, path, edits, narrows, expected_test in PROBES:
        if opts.only and name not in opts.only.split(","):
            continue
        # Edits are (old, new) pairs against path, or (file, old, new)
        # triples for the rare gate enforced in more than one file. Every
        # plant must still match exactly once in its own file.
        grouped = {}
        for edit in edits:
            if len(edit) == 3:
                file, old, new = edit
            else:
                file, old, new = path, edit[0], edit[1]
            grouped.setdefault(file, []).append((old, new))
        folder = out / name
        folder.mkdir(exist_ok=True)
        originals = {}
        replace = {}
        for file, file_edits in grouped.items():
            source = ROOT / file
            original = source.read_bytes()
            originals[str(source)] = (source, original)
            mutated = original.decode()
            for old, new in file_edits:
                if mutated.count(old) != 1:
                    raise RuntimeError(f"{name}: plant must match exactly once in {file}, got {mutated.count(old)}")
                mutated = mutated.replace(old, new, 1)
            replacement = folder / (source.name if len(grouped) == 1 else file.replace("/", "_"))
            replacement.write_text(mutated)
            replace[str(source)] = str(replacement)
        overlay = folder / "overlay.json"
        overlay.write_text(json.dumps({"Replace": replace}))
        command = ["go", "test", "-overlay", str(overlay), "./internal/hostchannel", "-count=1", "-json", "-timeout=300s", "-run", TESTS]
        proc = subprocess.run(command, cwd=ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=420)
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
        for key in sorted(originals):
            source, original = originals[key]
            if source.read_bytes() != original:
                raise RuntimeError(f"{name}: source {source} changed during overlay run")
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
