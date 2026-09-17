// Package hostchannel_test hostile conformance suite (TASK-260830-2x16gz).
//
// This file re-derives the seven original hostile-network cases of the peer
// authentication story through the real authenticated channel — unknown
// peers, key changes, spoofed host IDs, disconnects, replay, oversized
// frames, disclosure mismatches — and measures the AC-HOST-001 product-gate
// rows (real OpenSSH carrier, timeout/race, custody, revocation/generation,
// migration/downgrade, both role failures, stale reads, recovery bypasses)
// through the landed production entries hostchannel.Dial/Serve/Call,
// hosttrust enrollment/rotation/revocation/generation, Config-4 selection and
// the SSH transport boundary. No new product model lives here: every vector
// drives a production entry point and pins the exact Section 15 refusal class
// plus the absence of undeclared durable effects.
//
// Stated bounds (recorded, never passed): native Tailscale SSH (HC-PARITY by
// construction — no SSH-implementation branch exists in the package),
// platform lanes other than this host, and the Section 11.10.3 one-second
// idle self-close (measured in TestHostileRevocationLivePeer/idle_observation:
// the product fences the next dispatch and exposes the WatchGeneration
// cancellation primitive, but serveLoop carries no background watchdog, so an
// idle stream with zero traffic does not self-close; see the conformance
// matrix for the exact observation).
package hostchannel_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/hostchannel"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// assertLocalFailure pins the initiator-local Section 15 class and exit for
// a channel error through the production mapping, and cross-checks the exit
// against the Structured Error registry.
func assertLocalFailure(t *testing.T, err error, code string, exit int) {
	t.Helper()
	mapped, mappedExit, ok := hostchannel.LocalFailure(err)
	if !ok || string(mapped) != code || mappedExit != exit {
		t.Fatalf("LocalFailure(%v) = %q/%d/%v, want %q/%d", err, mapped, mappedExit, ok, code, exit)
	}
	registryExit, registryErr := axerror.ExitCodeFor(axerror.Version130, mapped)
	if registryErr != nil || registryExit != exit {
		t.Fatalf("registry exit for %q = %d, %v", mapped, registryExit, registryErr)
	}
}

// hostileCensus snapshots both fixture state trees for before/after
// comparison. A channel run performs no file writes; revocation tests use
// the trust-transition-aware assertion instead.
type hostileCensus struct {
	a map[string][]byte
	b map[string][]byte
}

func censusHostile(t *testing.T, f *fixture) hostileCensus {
	t.Helper()
	return hostileCensus{a: snapshotTree(t, f.dirA), b: snapshotTree(t, f.dirB)}
}

func (c hostileCensus) assertUnchanged(t *testing.T, f *fixture) {
	t.Helper()
	assertTreeEqual(t, f.dirA, c.a)
	assertTreeEqual(t, f.dirB, c.b)
}

func assertTreeEqual(t *testing.T, root string, before map[string][]byte) {
	t.Helper()
	after := snapshotTree(t, root)
	if len(after) != len(before) {
		t.Fatalf("state file set changed under %s: %d -> %d files", root, len(before), len(after))
	}
	for name, want := range before {
		got, ok := after[name]
		if !ok || !bytes.Equal(got, want) {
			t.Fatalf("state file %s changed", name)
		}
	}
}

// hostileTimeouts bounds every phase of a hostile exchange so a hanging peer
// fails the vector instead of the suite. Production defaults (10 s / 300 s)
// stay covered by TestLocalCredentialRefusals/default_timeouts.
func hostileTimeouts(client hostchannel.ClientConfig, server hostchannel.ServerConfig) (hostchannel.ClientConfig, hostchannel.ServerConfig) {
	client.HandshakeTimeout = 5 * time.Second
	client.HelloTimeout = 5 * time.Second
	client.RequestTimeout = 5 * time.Second
	server.HandshakeTimeout = 5 * time.Second
	server.HelloTimeout = 5 * time.Second
	server.RequestTimeout = 5 * time.Second
	return client, server
}

// countingHandler answers post-hello calls, records how many ran, and keeps
// the last binding for unchanged-binding assertions.
type countingHandler struct {
	calls   atomic.Int64
	last    atomic.Value // hostchannel.Binding
	answer  json.RawMessage
	refuse  map[string]bool
	onCall  func(binding hostchannel.Binding)
	entered chan<- struct{}
	release <-chan struct{}
}

func (h *countingHandler) handler() hostchannel.Handler {
	return func(_ context.Context, request rpcwire.Request, binding hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		h.calls.Add(1)
		h.last.Store(binding)
		if h.entered != nil {
			select {
			case h.entered <- struct{}{}:
			default:
			}
		}
		if h.release != nil {
			<-h.release
		}
		if h.onCall != nil {
			h.onCall(binding)
		}
		if h.refuse[request.Operation()] {
			return nil, errors.New("hostile handler refused " + request.Operation())
		}
		if len(h.answer) > 0 {
			return h.answer, nil
		}
		return json.RawMessage(`{}`), nil
	}
}

func (h *countingHandler) count() int { return int(h.calls.Load()) }

func (h *countingHandler) binding() (hostchannel.Binding, bool) {
	raw := h.last.Load()
	if raw == nil {
		return hostchannel.Binding{}, false
	}
	binding, ok := raw.(hostchannel.Binding)
	return binding, ok
}

// reissue mints a fresh credential for an already-enrolled host UUID through
// the production issuer: same host, new keys, new root. It is enrolled
// nowhere until an explicit Enroll commits it.
func reissue(t *testing.T, hostID string) hosttrust.IssuedCredential {
	t.Helper()
	issued, err := hosttrust.IssueCredential(hostID, time.Now().UTC(), nil)
	if err != nil {
		t.Fatalf("IssueCredential(%s): %v", hostID, err)
	}
	return issued
}

func credentialFrom(t *testing.T, issued hosttrust.IssuedCredential) tls.Certificate {
	t.Helper()
	cert, err := hostchannel.CredentialFromIssued(issued)
	if err != nil {
		t.Fatalf("CredentialFromIssued: %v", err)
	}
	return cert
}

// ---------------------------------------------------------------------------
// Case 1: unknown peers fail closed in both roles.
// Production call sites: hostchannel.Serve (server.go), hostchannel.Dial
// (client.go); refusal class via hostchannel.LocalFailure (channel.go).
// ---------------------------------------------------------------------------

func TestHostileUnknownPeerBothRoles(t *testing.T) {
	t.Run("responder_refuses", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		attacker := credentialFrom(t, reissue(t, hostC))
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		clientConfig, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		_ = clientConfig
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, attacker, "", nil))
		_ = rogue.Handshake()
		expectRogueHandshakeFails(t, rogue)
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "authentication_failed", 7)
		census.assertUnchanged(t, f)
	})
	t.Run("initiator_refuses", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		attacker := credentialFrom(t, reissue(t, hostC))
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, attacker, nil))
		rogueErr := make(chan error, 1)
		go func() { rogueErr <- rogue.Handshake() }()
		clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		_, dialErr := hostchannel.Dial(clientConn, clientConfig)
		if !errors.Is(dialErr, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Dial = %v", dialErr)
		}
		assertLocalFailure(t, dialErr, "authentication_failed", 7)
		<-rogueErr
		census.assertUnchanged(t, f)
	})
}

// ---------------------------------------------------------------------------
// Case 2: key changes fail closed. A re-issued leaf for an enrolled UUID is a
// different key under a different root; Section 11.10.2 never enrolls it
// implicitly, so both roles must refuse it until explicit enrollment.
// Production call sites: hostchannel.Serve/Dial; enrollment via
// hosttrust.Store.Enroll; rotation via hosttrust.Store.Rotate.
// ---------------------------------------------------------------------------

func TestHostileKeyChangeSameUUID(t *testing.T) {
	t.Run("responder_refuses_reissued", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		// Same UUID as the enrolled host A, fresh keys, enrolled nowhere.
		reissued := credentialFrom(t, reissue(t, hostA))
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, reissued, "", nil))
		_ = rogue.Handshake()
		expectRogueHandshakeFails(t, rogue)
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "authentication_failed", 7)
		census.assertUnchanged(t, f)
	})
	t.Run("initiator_refuses_reissued", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		reissued := credentialFrom(t, reissue(t, hostB))
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, reissued, nil))
		rogueErr := make(chan error, 1)
		go func() { rogueErr <- rogue.Handshake() }()
		clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		_, dialErr := hostchannel.Dial(clientConn, clientConfig)
		if !errors.Is(dialErr, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Dial = %v", dialErr)
		}
		assertLocalFailure(t, dialErr, "authentication_failed", 7)
		<-rogueErr
		census.assertUnchanged(t, f)
	})
	t.Run("rotation_admits_new_key_only_after_enrollment", func(t *testing.T) {
		f := newFixture(t)
		now := time.Now().UTC()
		newCredentialID, retireAt, err := f.storeA.Rotate(hostA, now)
		if err != nil {
			t.Fatalf("Rotate: %v", err)
		}
		if !retireAt.After(now) || retireAt.After(now.Add(hosttrust.RotationBound+time.Minute)) {
			t.Fatalf("retire_at = %v", retireAt)
		}
		rotated := rotatedCredential(t, f.storeA, newCredentialID)
		// Before enrollment on the verifier, the fresh key is refused even
		// though the UUID matches an enrolled host.
		dialWith := func(cert tls.Certificate) error {
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
			served := serveAsync(serverConn, serverConfig)
			rogue := tls.Client(clientConn, rogueClientConfig(t, f, cert, "", nil))
			_ = rogue.Handshake()
			expectRogueHandshakeFails(t, rogue)
			return awaitErr(t, served)
		}
		if err := dialWith(rotated.cert); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("unenrolled rotated key Serve = %v", err)
		}
		// The pre-rotation local credential cannot activate after the local
		// transition: activation requires an active local entry, so only the
		// fresh key initiates. (Peer-side overlap admission within retire_at
		// stays covered by TestRetiringCredentialAdmitsWithinBound.)
		{
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
			if _, err := hostchannel.Dial(clientConn, clientConfig); !errors.Is(err, hostchannel.ErrInvalidConfig) {
				t.Fatalf("Dial with retired local key = %v", err)
			}
		}
		// Explicit enrollment of the new tuple admits it.
		enroll(t, f.storeB, hosttrust.IssuedCredential{LeafDER: rotated.leafDER, RootDER: rotated.rootDER}, hostA, now)
		{
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
			rogue := tls.Client(clientConn, rogueClientConfig(t, f, rotated.cert, "", nil))
			if err := rogue.Handshake(); err != nil {
				t.Fatalf("handshake with enrolled rotated key: %v", err)
			}
			request, line, _ := rogueHello(t, hostA, nil)
			writeRaw(t, rogue, line)
			helloLine, err := hostchannel.ReadLine(rogue)
			if err != nil {
				t.Fatalf("ReadLine(hello): %v", err)
			}
			response, err := rpcwire.DecodeResponse(helloLine, request)
			if err != nil || !response.OK() {
				t.Fatalf("hello with enrolled rotated key = %+v, %v", response, err)
			}
			clientConn.Close()
			serverConn.Close()
			_ = awaitErr(t, served)
		}
	})
	t.Run("no_second_rotation_before_revocation", func(t *testing.T) {
		f := newFixture(t)
		before, err := f.storeA.ReadSnapshot()
		if err != nil {
			t.Fatalf("ReadSnapshot: %v", err)
		}
		if _, _, err := f.storeA.Rotate(hostA, time.Now().UTC()); err != nil {
			t.Fatalf("first Rotate: %v", err)
		}
		mid, err := f.storeA.ReadSnapshot()
		if err != nil {
			t.Fatalf("ReadSnapshot: %v", err)
		}
		if mid.Generation != before.Generation+1 {
			t.Fatalf("generation %d -> %d, want exactly one commit", before.Generation, mid.Generation)
		}
		if _, _, err := f.storeA.Rotate(hostA, time.Now().UTC()); err == nil {
			t.Fatal("second rotation admitted before revocation")
		} else if !errors.Is(err, hosttrust.ErrRotationRefused) {
			t.Fatalf("second Rotate = %v", err)
		}
		after, err := f.storeA.ReadSnapshot()
		if err != nil {
			t.Fatalf("ReadSnapshot: %v", err)
		}
		if after.Generation != mid.Generation {
			t.Fatalf("refused rotation moved the generation to %d", after.Generation)
		}
	})
}

// rotatedCredential loads a rotated leaf credential from the production
// custody layout the rotation committed: STATE_DIR/host-channel/
// credentials/HEX/{certificate,private-key,root}.pem.
type rotatedCredentialMaterial struct {
	cert    tls.Certificate
	leafDER []byte
	rootDER []byte
}

func rotatedCredential(t *testing.T, store *hosttrust.Store, credentialID string) rotatedCredentialMaterial {
	t.Helper()
	hexID, ok := strings.CutPrefix(credentialID, "sha256:")
	if !ok || len(hexID) != 64 {
		t.Fatalf("credential ID = %q", credentialID)
	}
	dir, err := store.CredentialDir(hexID)
	if err != nil {
		t.Fatalf("CredentialDir: %v", err)
	}
	readPEM := func(name, wantType string) []byte {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		block, _ := pem.Decode(raw)
		if block == nil || block.Type != wantType {
			t.Fatalf("%s is not %s PEM", name, wantType)
		}
		return block.Bytes
	}
	leafDER := readPEM("certificate.pem", "CERTIFICATE")
	keyDER := readPEM("private-key.pem", "PRIVATE KEY")
	rootDER := readPEM("root.pem", "CERTIFICATE")
	key, err := x509.ParsePKCS8PrivateKey(keyDER)
	if err != nil {
		t.Fatalf("ParsePKCS8PrivateKey: %v", err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parse rotated leaf: %v", err)
	}
	return rotatedCredentialMaterial{
		cert:    tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: key, Leaf: leaf},
		leafDER: leafDER,
		rootDER: rootDER,
	}
}

// ---------------------------------------------------------------------------
// Case 3: spoofed host IDs fail closed. A valid enrolled certificate proves
// only its own UUID: claiming the destination UUID in hello, or presenting a
// valid enrolled certificate for a different UUID than the selected
// destination, refuses on both roles.
// Production call sites: hostchannel.Serve/Dial; authorization via
// hosttrust.Store.AuthorizeDispatch.
// ---------------------------------------------------------------------------

func TestHostileSpoofedHostID(t *testing.T) {
	t.Run("hello_claims_destination", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		// Authenticated as A, claiming to be the destination B.
		request, line, _ := rogueHello(t, hostB, nil)
		writeRaw(t, rogue, line)
		assertFailure(t, readFailure(t, rogue, request, ""), "host_identity_mismatch", 7)
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hosttrust.ErrHostIdentityMismatch) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "authentication_failed", 7)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times for a spoofed hello", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("valid_but_unexpected_peer", func(t *testing.T) {
		f := newFixture(t)
		// C is valid and enrolled on the initiator side, but the selected
		// configured destination is B: a C certificate must not satisfy it.
		issuedC := reissue(t, hostC)
		enroll(t, f.storeA, issuedC, hostC, time.Now().UTC())
		certC := credentialFrom(t, issuedC)
		census := censusHostile(t, f)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, certC, nil))
		rogueErr := make(chan error, 1)
		go func() { rogueErr <- rogue.Handshake() }()
		clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		if clientConfig.ExpectedRemoteHostID != hostB {
			t.Fatalf("destination = %q", clientConfig.ExpectedRemoteHostID)
		}
		_, dialErr := hostchannel.Dial(clientConn, clientConfig)
		if !errors.Is(dialErr, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Dial = %v", dialErr)
		}
		assertLocalFailure(t, dialErr, "authentication_failed", 7)
		<-rogueErr
		census.assertUnchanged(t, f)
	})
}

// ---------------------------------------------------------------------------
// Case 4: disconnects at every phase boundary fail closed with no partial
// effect: mid-handshake, after handshake before hello, mid-frame, mid-request,
// mid-response, and responder aborts during the hello reply and during a
// response. Each vector pins the Section 15 class, proves the handler never
// observed the truncated exchange, and proves no durable write.
// Production call sites: hostchannel.Serve/Dial/Call.
// ---------------------------------------------------------------------------

func TestHostileDisconnectPhases(t *testing.T) {
	t.Run("mid_handshake_close", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		// Five handshake bytes, then the peer vanishes mid-flight.
		if _, err := clientConn.Write([]byte{0x16, 0x03, 0x01, 0x00, 0x01}); err != nil {
			t.Fatalf("rogue write: %v", err)
		}
		_ = clientConn.Close()
		start := time.Now()
		serveErr := awaitErr(t, served)
		if elapsed := time.Since(start); elapsed > 4*time.Second {
			t.Fatalf("mid-handshake close took %v", elapsed)
		}
		if !errors.Is(serveErr, hostchannel.ErrTransportFailure) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "transport_failure", 8)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("after_handshake_before_hello", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		serverConfig.HelloTimeout = 300 * time.Millisecond
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		// The authenticated peer goes silent instead of sending hello.
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrTransportFailure) {
			t.Fatalf("Serve = %v", serveErr)
		} else if !strings.Contains(serveErr.Error(), "deadline") {
			t.Fatalf("Serve = %v, want the hello deadline refusal", serveErr)
		}
		assertLocalFailure(t, serveErr, "transport_failure", 8)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("hello_half_frame_close", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		_, line, _ := rogueHello(t, hostA, nil)
		half := append(bytes.Clone(line[:len(line)/2]), '\n')
		_ = rogue.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := rogue.Write(half); err != nil {
			t.Fatalf("rogue write: %v", err)
		}
		_ = rogue.Close()
		_ = clientConn.Close()
		serveErr := awaitErr(t, served)
		// A truncated pre-hello frame is unframeable: close without a frame.
		if !errors.Is(serveErr, hostchannel.ErrUnframeableInput) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "transport_failure", 8)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("mid_request_close", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		request, line, _ := rogueHello(t, hostA, nil)
		writeRaw(t, rogue, line)
		helloLine, err := hostchannel.ReadLine(rogue)
		if err != nil {
			t.Fatalf("ReadLine(hello): %v", err)
		}
		if response, err := rpcwire.DecodeResponse(helloLine, request); err != nil || !response.OK() {
			t.Fatalf("hello = %+v, %v", response, err)
		}
		id, err := hostchannel.NewRequestID()
		if err != nil {
			t.Fatalf("NewRequestID: %v", err)
		}
		callLine, err := rpcwire.EncodeRequest("5.0.0", id, "health.get", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("EncodeRequest: %v", err)
		}
		// Half a post-hello request frame, terminated, then the peer
		// vanishes: the complete-but-truncated line reaches dispatch decode
		// and must close unframeable without running the handler.
		truncated := append(bytes.Clone(callLine[:len(callLine)/2]), '\n')
		_ = rogue.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := rogue.Write(truncated); err != nil {
			t.Fatalf("rogue write: %v", err)
		}
		_ = rogue.Close()
		_ = clientConn.Close()
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrUnframeableInput) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "transport_failure", 8)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times for a truncated request", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("mid_response_close", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		release := make(chan struct{})
		handler := &countingHandler{release: release}
		clientConn, serverConn := net.Pipe()
		defer serverConn.Close()
		clientConfig, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		called := make(chan error, 1)
		go func() {
			_, err := client.Call("health.get", json.RawMessage(`{}`))
			called <- err
		}()
		// Wait until the dispatch is admitted, then drop the peer mid-response.
		deadline := time.Now().Add(10 * time.Second)
		for handler.count() == 0 && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		if handler.count() == 0 {
			t.Fatal("dispatch never reached the handler")
		}
		_ = client.Close()
		close(release)
		select {
		case err := <-called:
			if !errors.Is(err, hostchannel.ErrTransportFailure) {
				t.Fatalf("Call = %v", err)
			}
			assertLocalFailure(t, err, "transport_failure", 8)
		case <-time.After(30 * time.Second):
			t.Fatal("Call did not return")
		}
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrTransportFailure) {
			t.Fatalf("Serve = %v", serveErr)
		}
		census.assertUnchanged(t, f)
	})
	t.Run("server_aborts_hello_reply", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
		aborted := make(chan struct{})
		go func() {
			defer close(aborted)
			if err := rogue.Handshake(); err != nil {
				return
			}
			// Read the hello, then vanish without the success reply.
			_, _ = hostchannel.ReadLine(rogue)
			_ = rogue.Close()
			_ = serverConn.Close()
		}()
		clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		_, dialErr := hostchannel.Dial(clientConn, clientConfig)
		if !errors.Is(dialErr, hostchannel.ErrTransportFailure) {
			t.Fatalf("Dial = %v", dialErr)
		}
		assertLocalFailure(t, dialErr, "transport_failure", 8)
		<-aborted
		census.assertUnchanged(t, f)
	})
	t.Run("server_aborts_response", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
		aborted := make(chan struct{})
		go func() {
			defer close(aborted)
			if err := rogue.Handshake(); err != nil {
				return
			}
			line, err := hostchannel.ReadLine(rogue)
			if err != nil {
				return
			}
			request, err := rpcwire.DecodeRequest(line)
			if err != nil {
				return
			}
			writeRaw(t, rogue, rogueHelloSuccess(t, request))
			// Read the post-hello call, then vanish without the response.
			_, _ = hostchannel.ReadLine(rogue)
			_ = rogue.Close()
			_ = serverConn.Close()
		}()
		clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer client.Close()
		_, callErr := client.Call("health.get", json.RawMessage(`{}`))
		if !errors.Is(callErr, hostchannel.ErrTransportFailure) {
			t.Fatalf("Call = %v", callErr)
		}
		assertLocalFailure(t, callErr, "transport_failure", 8)
		<-aborted
		census.assertUnchanged(t, f)
	})
}

// rogueHelloSuccess builds a well-formed correlated hello success for the
// responder UUID through public entries only.
func rogueHelloSuccess(t *testing.T, request rpcwire.Request) []byte {
	t.Helper()
	sent, err := request.Hello()
	if err != nil {
		t.Fatalf("Hello: %v", err)
	}
	contracts, err := rpcwire.ContractProfile("5.0.0")
	if err != nil {
		t.Fatalf("ContractProfile: %v", err)
	}
	lineBytes, err := scalar.NewUint53(rpcwire.MaxLineBytes)
	if err != nil {
		t.Fatalf("NewUint53: %v", err)
	}
	objectBytes, err := scalar.NewUint53(rpcwire.MinObjectBytes)
	if err != nil {
		t.Fatalf("NewUint53: %v", err)
	}
	body, err := json.Marshal(rpcwire.Hello{
		HostID: hostB, Platform: "linux", AXVersion: "0.6.0",
		Nonce: rpcwire.NewNonce(), NonceEcho: sent.Nonce, Contracts: contracts,
		MaxLineBytes: lineBytes, MaxObjectBytes: objectBytes,
	})
	if err != nil {
		t.Fatalf("marshal hello: %v", err)
	}
	success, err := rpcwire.EncodeSuccess(request, body)
	if err != nil {
		t.Fatalf("EncodeSuccess: %v", err)
	}
	return success
}

// recordWriter taps one direction of a stream for replay capture.
type recordWriter struct {
	w   io.Writer
	buf bytes.Buffer
	mu  sync.Mutex
}

func (r *recordWriter) Write(data []byte) (int, error) {
	n, err := r.w.Write(data)
	r.mu.Lock()
	r.buf.Write(data[:n])
	r.mu.Unlock()
	return n, err
}

func (r *recordWriter) bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return bytes.Clone(r.buf.Bytes())
}

// tapConn records initiator-to-responder bytes while proxying a net.Conn.
type tapConn struct {
	net.Conn
	tap *recordWriter
}

func (c *tapConn) Write(data []byte) (int, error) { return c.tap.Write(data) }

// ---------------------------------------------------------------------------
// Case 5: replay fails closed. A recorded TLS flight replayed on a fresh
// connection is refused by TLS 1.3 itself (fresh server random/keyshare make
// the replayed Finished unverifiable); a recorded encrypted hello replayed
// outside its session is handshake garbage; a hello replayed inside its
// session is plain re-dispatch under the unchanged binding, never
// re-authentication. Fresh 256-bit nonce material per hello is pinned
// separately (killed by the nonce-static mutant).
// Production call sites: hostchannel.Serve/Dial/Call; nonce via
// rpcwire.NewNonce.
// ---------------------------------------------------------------------------

// captureHandshakeAndHello runs one well-formed rogue session and returns the
// exact initiator byte spans: the full handshake flight and the encrypted
// hello flight that followed it.
func captureHandshakeAndHello(t *testing.T, f *fixture) (handshake, hello []byte) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	tap := &recordWriter{w: clientConn}
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(&tapConn{Conn: clientConn, tap: tap}, rogueClientConfig(t, f, f.certA, "", nil))
	if err := rogue.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	handshake = tap.bytes()
	request, line, _ := rogueHello(t, hostA, nil)
	writeRaw(t, rogue, line)
	helloLine, err := hostchannel.ReadLine(rogue)
	if err != nil {
		t.Fatalf("ReadLine(hello): %v", err)
	}
	if response, err := rpcwire.DecodeResponse(helloLine, request); err != nil || !response.OK() {
		t.Fatalf("hello = %+v, %v", response, err)
	}
	full := tap.bytes()
	hello = bytes.Clone(full[len(handshake):])
	if len(handshake) == 0 || len(hello) == 0 {
		t.Fatalf("capture spans are empty: %d + %d bytes", len(handshake), len(hello))
	}
	clientConn.Close()
	serverConn.Close()
	_ = awaitErr(t, served)
	return handshake, hello
}

// replayToFreshServer feeds recorded bytes to a fresh responder and drains
// its replies. The responder must refuse: a replay is never a session.
func replayToFreshServer(t *testing.T, f *fixture, flight []byte, handler hostchannel.Handler) error {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler))
	served := serveAsync(serverConn, serverConfig)
	_ = clientConn.SetDeadline(time.Now().Add(15 * time.Second))
	_ = serverConn.SetDeadline(time.Now().Add(15 * time.Second))
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		_, _ = io.Copy(io.Discard, clientConn)
	}()
	_, _ = clientConn.Write(flight)
	serveErr := awaitErr(t, served)
	_ = clientConn.Close()
	_ = serverConn.Close()
	<-drained
	return serveErr
}

func TestHostileReplay(t *testing.T) {
	t.Run("tls_flight_replay_refused", func(t *testing.T) {
		f := newFixture(t)
		handshake, hello := captureHandshakeAndHello(t, f)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		// The full recorded initiator flight, replayed verbatim on a fresh
		// connection: TLS 1.3 cannot verify the replayed Finished against
		// fresh server randomness, so the handshake must fail.
		flight := append(bytes.Clone(handshake), hello...)
		serveErr := replayToFreshServer(t, f, flight, handler.handler())
		if serveErr == nil {
			t.Fatal("replayed TLS flight admitted")
		}
		if !errors.Is(serveErr, hostchannel.ErrAuthenticationFailed) && !errors.Is(serveErr, hostchannel.ErrTransportFailure) {
			t.Fatalf("Serve = %v", serveErr)
		}
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times for a replay", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("encrypted_hello_replay_refused", func(t *testing.T) {
		f := newFixture(t)
		_, hello := captureHandshakeAndHello(t, f)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		// The recorded encrypted hello alone, offered as a fresh handshake:
		// application records before any handshake are refused.
		serveErr := replayToFreshServer(t, f, bytes.Clone(hello), handler.handler())
		if serveErr == nil {
			t.Fatal("replayed encrypted hello admitted")
		}
		if !errors.Is(serveErr, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "authentication_failed", 7)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times for a replay", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("rehello_is_inert_redispatch", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		var bindings []hostchannel.Binding
		handler := &countingHandler{refuse: map[string]bool{"hello": true}}
		wrapped := handler.handler()
		seen := func(_ context.Context, request rpcwire.Request, binding hostchannel.Binding, mutate func(func() error) error) (json.RawMessage, error) {
			body, err := wrapped(context.Background(), request, binding, mutate)
			bindings = append(bindings, binding)
			return body, err
		}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		clientConfig, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(seen))
		served := serveAsync(serverConn, serverConfig)
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer client.Close()
		_, line, _ := rogueHello(t, hostA, nil)
		request, err := rpcwire.DecodeRequest(line)
		if err != nil {
			t.Fatalf("DecodeRequest: %v", err)
		}
		// The exact hello bytes, replayed inside the session: dispatched as
		// an ordinary call (here refused by the handler), never
		// re-authenticated, and the binding must not move.
		response, err := client.Call("hello", request.Body())
		if err != nil {
			t.Fatalf("Call(hello) = %v", err)
		}
		if response.OK() {
			t.Fatal("replayed hello succeeded")
		}
		assertFailure(t, response.Failure(), "incompatible_protocol", 6)
		if handler.count() != 1 {
			t.Fatalf("handler calls = %d", handler.count())
		}
		if len(bindings) != 1 {
			t.Fatalf("bindings observed = %d", len(bindings))
		}
		replayed := bindings[0]
		if replayed.RemoteHostID != hostA || replayed.HelloHostID != hostA || replayed.LocalHostID != hostB {
			t.Fatal("replayed hello carried a foreign binding")
		}
		// The channel continues undisturbed after the inert replay, under
		// the same binding: the replay neither re-authenticated nor moved
		// the authorization facts.
		if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
			t.Fatalf("Call after replay = %+v, %v", response, err)
		}
		if len(bindings) != 2 {
			t.Fatalf("bindings observed = %d", len(bindings))
		}
		next := bindings[1]
		if next.Generation != replayed.Generation || next.RemoteHostID != replayed.RemoteHostID ||
			next.RemoteCredentialID != replayed.RemoteCredentialID || next.HelloHostID != replayed.HelloHostID {
			t.Fatal("replayed hello moved the authorization binding")
		}
		if err := client.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		_ = awaitErr(t, served)
		census.assertUnchanged(t, f)
	})
}

func TestHostileReplayNonceFreshness(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 256; i++ {
		nonce := rpcwire.NewNonce()
		if len(nonce) < 22 {
			t.Fatalf("nonce %d too short: %q", i, nonce)
		}
		if _, duplicate := seen[nonce]; duplicate {
			t.Fatalf("duplicate nonce at draw %d", i)
		}
		seen[nonce] = struct{}{}
	}
}

// ---------------------------------------------------------------------------
// Case 6: oversized frames fail closed. The 8 MiB line cap fires at hello and
// at dispatch with an unframeable close; the 5 MiB / 8 MiB hello floors are
// disclosure checks (an offer below the floor is a contract mismatch); the
// 1 MiB handshake cap is the backstop behind Go record framing.
// Production call sites: hostchannel.Serve/Dial/Call framing; limits from
// rpcwire.MaxLineBytes / rpcwire.MinObjectBytes.
// ---------------------------------------------------------------------------

func TestHostileOversizedFrames(t *testing.T) {
	t.Run("hello_line_8mib_plus_one", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		// A structurally valid hello whose version build tag overflows the
		// line cap: the cap fires before any field is trusted.
		_, line, _ := rogueHello(t, hostA, func(body map[string]any) {
			body["ax_version"] = "0.6.0+" + strings.Repeat("D", rpcwire.MaxLineBytes+1024)
		})
		if len(line) <= rpcwire.MaxLineBytes {
			t.Fatalf("hello line = %d bytes, want over the cap", len(line))
		}
		wrote := make(chan error, 1)
		go func() {
			_, err := rogue.Write(append(bytes.Clone(line), '\n'))
			wrote <- err
		}()
		expectClose(t, rogue)
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrUnframeableInput) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "transport_failure", 8)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times", handler.count())
		}
		<-wrote
		census.assertUnchanged(t, f)
	})
	t.Run("dispatch_line_8mib_plus_one", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		request, line, _ := rogueHello(t, hostA, nil)
		writeRaw(t, rogue, line)
		helloLine, err := hostchannel.ReadLine(rogue)
		if err != nil {
			t.Fatalf("ReadLine(hello): %v", err)
		}
		if response, err := rpcwire.DecodeResponse(helloLine, request); err != nil || !response.OK() {
			t.Fatalf("hello = %+v, %v", response, err)
		}
		// A structurally valid call sized exactly one byte over the cap:
		// the cap fires before decode, so the handler never observes it.
		id, err := hostchannel.NewRequestID()
		if err != nil {
			t.Fatalf("NewRequestID: %v", err)
		}
		pad := func(n int) []byte {
			line, err := json.Marshal(map[string]any{
				"protocol": "urn:ax:protocol:rpc", "protocol_version": "5.0.0",
				"request_id": id, "operation": "health.get",
				"body": map[string]any{"pad": strings.Repeat("D", n)},
			})
			if err != nil {
				t.Fatalf("marshal padded call: %v", err)
			}
			return line
		}
		callLine := pad(rpcwire.MaxLineBytes + 1 - len(pad(0)))
		if len(callLine) != rpcwire.MaxLineBytes+1 {
			t.Fatalf("call line = %d bytes", len(callLine))
		}
		wrote := make(chan error, 1)
		go func() {
			_, err := rogue.Write(append(bytes.Clone(callLine), '\n'))
			wrote <- err
		}()
		expectClose(t, rogue)
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrUnframeableInput) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "transport_failure", 8)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times for an oversize dispatch", handler.count())
		}
		<-wrote
		census.assertUnchanged(t, f)
	})
	t.Run("hello_below_advertised_floor", func(t *testing.T) {
		floors := map[string]func(map[string]any){
			"object": func(body map[string]any) { body["max_object_bytes"] = float64(rpcwire.MinObjectBytes - 1) },
			"line":   func(body map[string]any) { body["max_line_bytes"] = float64(rpcwire.MaxLineBytes - 1) },
		}
		for name, edit := range floors {
			t.Run(name, func(t *testing.T) {
				f := newFixture(t)
				census := censusHostile(t, f)
				handler := &countingHandler{}
				clientConn, serverConn := net.Pipe()
				defer clientConn.Close()
				defer serverConn.Close()
				_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
				served := serveAsync(serverConn, serverConfig)
				rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
				if err := rogue.Handshake(); err != nil {
					t.Fatalf("handshake: %v", err)
				}
				_, line, sentID := rogueHello(t, hostA, edit)
				writeRaw(t, rogue, line)
				assertFailure(t, readFailure(t, rogue, rpcwire.Request{}, sentID), "incompatible_protocol", 6)
				serveErr := awaitErr(t, served)
				if !errors.Is(serveErr, hostchannel.ErrProtocol) {
					t.Fatalf("Serve = %v", serveErr)
				}
				assertLocalFailure(t, serveErr, "incompatible_protocol", 6)
				if handler.count() != 0 {
					t.Fatalf("handler ran %d times", handler.count())
				}
				census.assertUnchanged(t, f)
			})
		}
	})
	t.Run("handshake_intake_bound_documented", func(t *testing.T) {
		// Mechanism note: a garbage flood is refused by Go TLS record
		// framing (the first record never parses), so intake stays far below
		// the 1 MiB channel limiter; the limiter is the backstop for
		// well-formed-but-huge handshakes and is pinned at its exact edge by
		// TestHandshakeLimiterBounds plus the handshake-cap mutant.
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		counted := &countingStream{Conn: serverConn}
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(counted, serverConfig)
		wrote := make(chan error, 1)
		go func() {
			_, err := clientConn.Write(bytes.Repeat([]byte("A"), 4*1024*1024))
			wrote <- err
		}()
		start := time.Now()
		serveErr := awaitErr(t, served)
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("hostile handshake took %v", elapsed)
		}
		if !errors.Is(serveErr, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Serve = %v", serveErr)
		}
		if got := counted.reads.Load(); got >= 64*1024 {
			t.Fatalf("hostile intake = %d bytes", got)
		}
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times", handler.count())
		}
		<-wrote
		census.assertUnchanged(t, f)
	})
}

// ---------------------------------------------------------------------------
// Case 7: disclosure mismatches fail closed. contracts.rpc other than exactly
// ["5.0.0"] refuses on receipt in both roles; a Structured Error whose
// version drifts from the static 1.3.0 binding is refused, never honored.
// Production call sites: hostchannel.Serve/Dial via rpcwire hello decode and
// axerror.DecodeBound.
// ---------------------------------------------------------------------------

func TestHostileDisclosureMismatch(t *testing.T) {
	t.Run("contracts_rpc_superset", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		_, line, sentID := rogueHello(t, hostA, func(body map[string]any) {
			body["contracts"].(map[string]any)["rpc"] = []string{"4.0.0", "5.0.0"}
		})
		writeRaw(t, rogue, line)
		assertFailure(t, readFailure(t, rogue, rpcwire.Request{}, sentID), "incompatible_protocol", 6)
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrProtocol) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "incompatible_protocol", 6)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("contracts_rpc_missing", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		_, line, sentID := rogueHello(t, hostA, func(body map[string]any) {
			delete(body["contracts"].(map[string]any), "rpc")
		})
		writeRaw(t, rogue, line)
		assertFailure(t, readFailure(t, rogue, rpcwire.Request{}, sentID), "incompatible_protocol", 6)
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hostchannel.ErrProtocol) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "incompatible_protocol", 6)
		if handler.count() != 0 {
			t.Fatalf("handler ran %d times", handler.count())
		}
		census.assertUnchanged(t, f)
	})
	t.Run("responder_contracts_drift", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
		dialed := make(chan error, 1)
		go func() {
			clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
			client, err := hostchannel.Dial(clientConn, clientConfig)
			if err == nil {
				_ = client.Close()
			}
			dialed <- err
		}()
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		line, err := hostchannel.ReadLine(rogue)
		if err != nil {
			t.Fatalf("ReadLine(hello): %v", err)
		}
		request, err := rpcwire.DecodeRequest(line)
		if err != nil {
			t.Fatalf("DecodeRequest: %v", err)
		}
		sent, err := request.Hello()
		if err != nil {
			t.Fatalf("Hello: %v", err)
		}
		contracts, err := rpcwire.ContractProfile("5.0.0")
		if err != nil {
			t.Fatalf("ContractProfile: %v", err)
		}
		contracts["rpc"] = []string{"4.0.0"}
		lineBytes, _ := scalar.NewUint53(rpcwire.MaxLineBytes)
		objectBytes, _ := scalar.NewUint53(rpcwire.MinObjectBytes)
		body, err := json.Marshal(rpcwire.Hello{
			HostID: hostB, Platform: "linux", AXVersion: "0.6.0",
			Nonce: rpcwire.NewNonce(), NonceEcho: sent.Nonce, Contracts: contracts,
			MaxLineBytes: lineBytes, MaxObjectBytes: objectBytes,
		})
		if err != nil {
			t.Fatalf("marshal hello: %v", err)
		}
		// EncodeSuccess would validate; the rogue frames the drifted reply
		// by hand so the initiator's decoder is the gate under test.
		reply, err := json.Marshal(map[string]any{
			"protocol": "urn:ax:protocol:rpc", "protocol_version": "5.0.0",
			"request_id": request.ID(), "ok": true, "body": json.RawMessage(body),
		})
		if err != nil {
			t.Fatalf("marshal reply: %v", err)
		}
		writeRaw(t, rogue, reply)
		select {
		case err := <-dialed:
			if !errors.Is(err, hostchannel.ErrProtocol) {
				t.Fatalf("Dial = %v", err)
			}
			assertLocalFailure(t, err, "incompatible_protocol", 6)
		case <-time.After(30 * time.Second):
			t.Fatal("Dial did not return")
		}
		census.assertUnchanged(t, f)
	})
	t.Run("error_version_drift", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
		dialed := make(chan error, 1)
		go func() {
			clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
			client, err := hostchannel.Dial(clientConn, clientConfig)
			if err == nil {
				_ = client.Close()
			}
			dialed <- err
		}()
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		line, err := hostchannel.ReadLine(rogue)
		if err != nil {
			t.Fatalf("ReadLine(hello): %v", err)
		}
		request, err := rpcwire.DecodeRequest(line)
		if err != nil {
			t.Fatalf("DecodeRequest: %v", err)
		}
		// A hello failure whose body is fully 1.3.0-shaped except the
		// schema_version label, drifted to 1.2.0: the static 1.3.0 binding
		// refuses the drifted version before the code is honored, so even a
		// mapped code surfaces as a protocol mismatch, never as its mapped
		// class.
		reply, err := json.Marshal(map[string]any{
			"protocol": "urn:ax:protocol:rpc", "protocol_version": "5.0.0",
			"request_id": request.ID(), "ok": false,
			"error": map[string]any{
				"schema": "urn:ax:schema:error", "schema_version": "1.2.0",
				"code": "peer_not_allowlisted", "message": "rogue drifted refusal",
				"exit_code": 7, "retryable": false, "details": map[string]any{},
			},
		})
		if err != nil {
			t.Fatalf("marshal reply: %v", err)
		}
		writeRaw(t, rogue, reply)
		select {
		case err := <-dialed:
			if !errors.Is(err, hostchannel.ErrProtocol) {
				t.Fatalf("Dial = %v", err)
			}
			if errors.Is(err, hosttrust.ErrNotAllowlisted) {
				t.Fatalf("Dial honored the drifted code: %v", err)
			}
			if !strings.Contains(err.Error(), "version") {
				t.Fatalf("Dial = %v, want the version-drift cause", err)
			}
			assertLocalFailure(t, err, "incompatible_protocol", 6)
		case <-time.After(30 * time.Second):
			t.Fatal("Dial did not return")
		}
		census.assertUnchanged(t, f)
	})
}

// ---------------------------------------------------------------------------
// AC-HOST-001 rows: timeout/race, custody, revocation/generation, role swap.
// ---------------------------------------------------------------------------

func TestHostileConcurrentPeerConnections(t *testing.T) {
	f := newFixture(t)
	census := censusHostile(t, f)
	var total atomic.Int64
	handler := func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		total.Add(1)
		return json.RawMessage(`{}`), nil
	}
	open := func() (hostchannel.Stream, hostchannel.Stream, chan error) {
		clientConn, serverConn := net.Pipe()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler))
		return clientConn, serverConn, serveAsync(serverConn, serverConfig)
	}
	clientConn1, serverConn1, served1 := open()
	defer clientConn1.Close()
	defer serverConn1.Close()
	clientConn2, serverConn2, served2 := open()
	defer clientConn2.Close()
	defer serverConn2.Close()
	dial := func(conn hostchannel.Stream) *hostchannel.Client {
		clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(handler))
		client, err := hostchannel.Dial(conn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		return client
	}
	// Two concurrent connections from one peer: both admit with the same
	// current generation, calls interleave, and closing one leaves the other
	// undisturbed.
	first := make(chan *hostchannel.Client, 1)
	go func() { first <- dial(clientConn1) }()
	second := dial(clientConn2)
	primary := <-first
	if primary.Binding().Generation != second.Binding().Generation {
		t.Fatal("concurrent connections bound different generations")
	}
	call := func(client *hostchannel.Client) {
		response, err := client.Call("health.get", json.RawMessage(`{}`))
		if err != nil || !response.OK() {
			t.Fatalf("Call = %+v, %v", response, err)
		}
	}
	call(primary)
	call(second)
	if err := primary.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served1)
	call(second)
	if total.Load() != 3 {
		t.Fatalf("handler calls = %d", total.Load())
	}
	if err := second.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served2)
	census.assertUnchanged(t, f)
}

func TestHostileStallingPeerRPCTimeout(t *testing.T) {
	f := newFixture(t)
	census := censusHostile(t, f)
	release := make(chan struct{})
	handler := &countingHandler{release: release}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	clientConfig, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
	clientConfig.RequestTimeout = 300 * time.Millisecond
	serverConfig.RequestTimeout = 30 * time.Second
	served := serveAsync(serverConn, serverConfig)
	client, err := hostchannel.Dial(clientConn, clientConfig)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	// The peer stalls mid-request against the configured RPC timeout: the
	// caller fails closed on its bound instead of hanging on the peer.
	called := make(chan error, 1)
	go func() {
		_, err := client.Call("health.get", json.RawMessage(`{}`))
		called <- err
	}()
	deadline := time.Now().Add(10 * time.Second)
	for handler.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if handler.count() == 0 {
		t.Fatal("dispatch never reached the handler")
	}
	select {
	case err := <-called:
		if !errors.Is(err, hostchannel.ErrTransportFailure) {
			t.Fatalf("Call = %v", err)
		} else if !strings.Contains(err.Error(), "deadline") {
			t.Fatalf("Call = %v, want the RPC timeout refusal", err)
		}
		assertLocalFailure(t, err, "transport_failure", 8)
	case <-time.After(30 * time.Second):
		t.Fatal("Call did not return")
	}
	close(release)
	serveErr := awaitErr(t, served)
	if !errors.Is(serveErr, hostchannel.ErrTransportFailure) {
		t.Fatalf("Serve = %v", serveErr)
	}
	census.assertUnchanged(t, f)
}

func TestHostileCopiedKey(t *testing.T) {
	f := newFixture(t)
	// Exfiltrated bytes: the leaf and private key re-parsed from raw DER, as
	// an attacker holding a copy would present them. Section 11.10.4 states
	// a copied key authenticates until each verifier revokes it; both halves
	// are asserted here through the real channel.
	leaf, err := x509.ParseCertificate(bytes.Clone(f.issuedA.LeafDER))
	if err != nil {
		t.Fatalf("parse copied leaf: %v", err)
	}
	key, err := x509.ParsePKCS8PrivateKey(bytes.Clone(f.issuedA.LeafKeyDER))
	if err != nil {
		t.Fatalf("parse copied key: %v", err)
	}
	copied := tls.Certificate{Certificate: [][]byte{bytes.Clone(leaf.Raw)}, PrivateKey: key, Leaf: leaf}
	// dialWith runs production Dial against production Serve and reports
	// both sides. When the server refuses mid-handshake the initiator may
	// observe either the alert or the close (TLS 1.3 lets the client finish
	// first); both map to transport_failure on the Dial side while the
	// refusing Serve side deterministically reports authentication_failed.
	// The short hello bound resolves that rendezvous quickly.
	dialWith := func(cert tls.Certificate) (dialErr, serveErr error) {
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		served := serveAsync(serverConn, serverConfig)
		clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		clientConfig.Credential = cert
		clientConfig.HelloTimeout = 1500 * time.Millisecond
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			return err, awaitErr(t, served)
		}
		_ = client.Close()
		return nil, awaitErr(t, served)
	}
	if dialErr, _ := dialWith(copied); dialErr != nil {
		t.Fatalf("copied key Dial = %v", dialErr)
	}
	// Revocation on the verifier ends the copied key: the next connection
	// refuses at the handshake. Revocation is a legitimate durable mutation;
	// assert it is exactly the trust transition, nothing else.
	before := censusHostile(t, f)
	beforeGen, err := f.storeB.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if err := f.storeB.Revoke(f.credAID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	afterGen, err := f.storeB.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if afterGen.Generation != beforeGen.Generation+1 {
		t.Fatalf("revocation moved %d -> %d", beforeGen.Generation, afterGen.Generation)
	}
	after := censusHostile(t, f)
	assertRevocationCensus(t, f, before, after)
	dialErr, serveErr := dialWith(copied)
	if !errors.Is(dialErr, hostchannel.ErrTransportFailure) {
		t.Fatalf("revoked copied key Dial = %v", dialErr)
	}
	assertLocalFailure(t, dialErr, "transport_failure", 8)
	if !errors.Is(serveErr, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("revoked copied key Serve = %v", serveErr)
	}
	assertLocalFailure(t, serveErr, "authentication_failed", 7)
	// Revocation on the initiator side refuses locally before launch.
	if err := f.storeA.Revoke(f.credAID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if dialErr, _ := dialWith(copied); !errors.Is(dialErr, hostchannel.ErrInvalidConfig) {
		t.Fatalf("self-revoked Dial = %v", dialErr)
	} else {
		assertLocalFailure(t, dialErr, "invalid_config", 3)
	}
}

// assertRevocationCensus proves a revocation commit changed exactly the trust
// document: same file set everywhere, only trust.json bytes differ on the
// mutated side.
func assertRevocationCensus(t *testing.T, f *fixture, before, after hostileCensus) {
	t.Helper()
	if len(after.a) != len(before.a) || len(after.b) != len(before.b) {
		t.Fatal("revocation changed the state file set")
	}
	for name, want := range before.a {
		if got, ok := after.a[name]; !ok || !bytes.Equal(got, want) {
			t.Fatalf("revocation touched initiator file %s", name)
		}
	}
	changed := 0
	for name, want := range before.b {
		got, ok := after.b[name]
		if !ok {
			t.Fatalf("revocation removed responder file %s", name)
		}
		if !bytes.Equal(got, want) {
			changed++
			if !strings.HasSuffix(name, "trust.json") {
				t.Fatalf("revocation changed non-trust file %s", name)
			}
		}
	}
	if changed != 1 {
		t.Fatalf("revocation changed %d responder files", changed)
	}
}

func TestHostileRevocationLivePeer(t *testing.T) {
	t.Run("next_dispatch_refused_fast", func(t *testing.T) {
		f := newFixture(t)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		clientConfig, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer client.Close()
		if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
			t.Fatalf("Call = %+v, %v", response, err)
		}
		before := censusHostile(t, f)
		start := time.Now()
		if err := f.storeB.Revoke(f.credAID); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		// The live stream fences at the next dispatch: the server closes
		// without a frame and the caller fails closed, well inside the
		// one-second Section 11.10.3 bound for dispatched work.
		_, callErr := client.Call("health.get", json.RawMessage(`{}`))
		elapsed := time.Since(start)
		if !errors.Is(callErr, hostchannel.ErrTransportFailure) {
			t.Fatalf("Call = %v", callErr)
		}
		assertLocalFailure(t, callErr, "transport_failure", 8)
		serveErr := awaitErr(t, served)
		if !errors.Is(serveErr, hosttrust.ErrStaleGeneration) && !errors.Is(serveErr, hosttrust.ErrAuthorizationRefused) {
			t.Fatalf("Serve = %v", serveErr)
		}
		if elapsed > time.Second {
			t.Fatalf("revoked dispatch fenced in %v", elapsed)
		}
		t.Logf("revoke-to-fence latency: %v", elapsed)
		if handler.count() != 1 {
			t.Fatalf("handler calls = %d, revoked dispatch ran", handler.count())
		}
		assertRevocationCensus(t, f, before, censusHostile(t, f))
	})
	t.Run("idle_observation", func(t *testing.T) {
		f := newFixture(t)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		clientConfig, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		serverConfig.RequestTimeout = 30 * time.Second
		served := serveAsync(serverConn, serverConfig)
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer client.Close()
		if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
			t.Fatalf("Call = %+v, %v", response, err)
		}
		if err := f.storeB.Revoke(f.credAID); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		// Observe (not assert) whether the idle stream self-closes without
		// traffic: the product fences the next dispatch (asserted below)
		// and exposes WatchGeneration for cancellation, but serveLoop
		// carries no background watchdog. The observation is evidence for
		// the conformance matrix stated bound.
		select {
		case err := <-served:
			t.Logf("idle stream self-closed during the observation window: %v", err)
			served = nil
		case <-time.After(1200 * time.Millisecond):
			t.Log("idle stream still open 1.2 s after revocation without traffic")
		}
		// No later dispatch from the stale generation is permitted: the
		// fenced call fails closed and the handler sees nothing further.
		if served != nil {
			if _, err := client.Call("health.get", json.RawMessage(`{}`)); !errors.Is(err, hostchannel.ErrTransportFailure) {
				t.Fatalf("post-revocation Call = %v", err)
			}
			if err := awaitErr(t, served); !errors.Is(err, hosttrust.ErrStaleGeneration) && !errors.Is(err, hosttrust.ErrAuthorizationRefused) {
				t.Fatalf("Serve = %v", err)
			}
		}
		if handler.count() != 1 {
			t.Fatalf("handler calls = %d", handler.count())
		}
	})
	t.Run("watch_reports_commit_fast", func(t *testing.T) {
		f := newFixture(t)
		snapshot, err := f.storeB.ReadSnapshot()
		if err != nil {
			t.Fatalf("ReadSnapshot: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		watched := make(chan error, 1)
		go func() {
			watched <- hostchannel.WatchGeneration(ctx, f.storeB, snapshot.Generation, 10*time.Millisecond)
		}()
		start := time.Now()
		if err := f.storeB.Revoke(f.credAID); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		select {
		case err := <-watched:
			elapsed := time.Since(start)
			if !errors.Is(err, hosttrust.ErrStaleGeneration) {
				t.Fatalf("watch = %v", err)
			}
			if elapsed > time.Second {
				t.Fatalf("watch reported in %v", elapsed)
			}
			t.Logf("revoke-to-watch latency: %v", elapsed)
		case <-time.After(10 * time.Second):
			t.Fatal("watch did not report the revocation")
		}
	})
}

func TestHostileStaleReadsAreIntegrityFailures(t *testing.T) {
	t.Run("corrupt_store", func(t *testing.T) {
		f := newFixture(t)
		path := f.storeB.TrustPath()
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(trust): %v", err)
		}
		corrupt := bytes.Clone(original)
		if len(corrupt) < 64 {
			t.Fatalf("trust document too short: %d bytes", len(corrupt))
		}
		corrupt[len(corrupt)/2] ^= 0xFF
		if err := os.WriteFile(path, corrupt, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		defer func() { _ = os.WriteFile(path, original, 0o600) }()
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		// A malformed store is an integrity failure, never an empty store:
		// activation refuses before any byte moves.
		serveErr := hostchannel.Serve(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		if !errors.Is(serveErr, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Serve = %v", serveErr)
		} else if !strings.Contains(serveErr.Error(), "snapshot") {
			t.Fatalf("Serve = %v, want the snapshot integrity cause", serveErr)
		}
		assertLocalFailure(t, serveErr, "invalid_config", 3)
		// Restoring the bytes restores authority: nothing was silently
		// repaired, migrated, or reported as absent.
		if err := os.WriteFile(path, original, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		snapshot, err := f.storeB.ReadSnapshot()
		if err != nil {
			t.Fatalf("ReadSnapshot after restore: %v", err)
		}
		if len(snapshot.Trust.Entries) != 2 {
			t.Fatalf("entries after restore = %d", len(snapshot.Trust.Entries))
		}
	})
	t.Run("unreadable_store", func(t *testing.T) {
		f := newFixture(t)
		path := f.storeB.TrustPath()
		if err := os.Chmod(path, 0); err != nil {
			t.Fatalf("Chmod: %v", err)
		}
		defer func() { _ = os.Chmod(path, 0o600) }()
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		serveErr := hostchannel.Serve(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		if !errors.Is(serveErr, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Serve = %v", serveErr)
		}
		assertLocalFailure(t, serveErr, "invalid_config", 3)
		t.Logf("unreadable store refusal: %v", serveErr)
	})
	t.Run("initiator_unreadable_store", func(t *testing.T) {
		f := newFixture(t)
		path := f.storeA.TrustPath()
		if err := os.Chmod(path, 0); err != nil {
			t.Fatalf("Chmod: %v", err)
		}
		defer func() { _ = os.Chmod(path, 0o600) }()
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		_, dialErr := hostchannel.Dial(clientConn, f.clientConfig())
		if !errors.Is(dialErr, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Dial = %v", dialErr)
		}
		assertLocalFailure(t, dialErr, "invalid_config", 3)
	})
}

func TestHostileRecoveryBypass(t *testing.T) {
	t.Run("call_after_close_refused", func(t *testing.T) {
		f := newFixture(t)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		clientConfig, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
			t.Fatalf("Call = %+v, %v", response, err)
		}
		if err := client.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		_ = awaitErr(t, served)
		// No fresh-handle substitution: a closed client never opens a new
		// session behind the caller's back.
		if _, err := client.Call("health.get", json.RawMessage(`{}`)); err == nil {
			t.Fatal("Call after Close succeeded")
		} else if !errors.Is(err, hostchannel.ErrTransportFailure) {
			t.Fatalf("Call after Close = %v", err)
		}
		if handler.count() != 1 {
			t.Fatalf("handler calls = %d", handler.count())
		}
	})
	t.Run("stale_binding_never_rebinds", func(t *testing.T) {
		f := newFixture(t)
		handler := &countingHandler{}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		clientConfig, serverConfig := hostileTimeouts(f.clientConfig(), f.serverConfig(handler.handler()))
		served := serveAsync(serverConn, serverConfig)
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer client.Close()
		if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
			t.Fatalf("Call = %+v, %v", response, err)
		}
		bound := client.Binding().Generation
		// Two successive trust commits: every retry still refuses under the
		// original binding. Work prepared under an old generation cannot
		// gain new authority by refreshing a cached number.
		enroll(t, f.storeA, reissue(t, hostC), hostC, time.Now().UTC())
		if _, err := client.Call("health.get", json.RawMessage(`{}`)); !errors.Is(err, hosttrust.ErrStaleGeneration) {
			t.Fatalf("first stale Call = %v", err)
		}
		enroll(t, f.storeA, reissue(t, "0198f4c8-7d40-7e55-8e6f-1234567890ad"), "0198f4c8-7d40-7e55-8e6f-1234567890ad", time.Now().UTC())
		if _, err := client.Call("health.get", json.RawMessage(`{}`)); !errors.Is(err, hosttrust.ErrStaleGeneration) {
			t.Fatalf("second stale Call = %v", err)
		}
		if client.Binding().Generation != bound {
			t.Fatal("stale binding silently rebound")
		}
		if handler.count() != 1 {
			t.Fatalf("handler calls = %d, stale Call transmitted", handler.count())
		}
		if err := client.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		_ = awaitErr(t, served)
	})
	t.Run("response_id_mismatch_refused", func(t *testing.T) {
		f := newFixture(t)
		census := censusHostile(t, f)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
		called := make(chan error, 1)
		go func() {
			clientConfig, _ := hostileTimeouts(f.clientConfig(), f.serverConfig(echoHandler(json.RawMessage(`{}`))))
			client, err := hostchannel.Dial(clientConn, clientConfig)
			if err != nil {
				called <- err
				return
			}
			defer client.Close()
			_, err = client.Call("health.get", json.RawMessage(`{}`))
			called <- err
		}()
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		line, err := hostchannel.ReadLine(rogue)
		if err != nil {
			t.Fatalf("ReadLine(hello): %v", err)
		}
		request, err := rpcwire.DecodeRequest(line)
		if err != nil {
			t.Fatalf("DecodeRequest: %v", err)
		}
		writeRaw(t, rogue, rogueHelloSuccess(t, request))
		callLine, err := hostchannel.ReadLine(rogue)
		if err != nil {
			t.Fatalf("ReadLine(call): %v", err)
		}
		call, err := rpcwire.DecodeRequest(callLine)
		if err != nil {
			t.Fatalf("DecodeRequest(call): %v", err)
		}
		otherID, err := hostchannel.NewRequestID()
		if err != nil {
			t.Fatalf("NewRequestID: %v", err)
		}
		// A success frame correlated to a different request ID: refused as
		// a protocol mismatch, never accepted as the pending call's answer.
		reply, err := json.Marshal(map[string]any{
			"protocol": "urn:ax:protocol:rpc", "protocol_version": "5.0.0",
			"request_id": otherID, "ok": true, "body": json.RawMessage(`{}`),
		})
		if err != nil {
			t.Fatalf("marshal reply: %v", err)
		}
		_ = call
		writeRaw(t, rogue, reply)
		select {
		case err := <-called:
			if !errors.Is(err, hostchannel.ErrProtocol) {
				t.Fatalf("Call = %v", err)
			}
			assertLocalFailure(t, err, "incompatible_protocol", 6)
		case <-time.After(30 * time.Second):
			t.Fatal("Call did not return")
		}
		census.assertUnchanged(t, f)
	})
}

// tryLoadSnapshot mirrors loadSnapshot but reports loader failures instead of
// failing: the migration refusal under test happens inside config.Load.
func tryLoadSnapshot(t *testing.T, document string) (config.Snapshot, error) {
	t.Helper()
	directory := t.TempDir()
	filename := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(filename, []byte(document), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return config.Load(config.Inputs{
		Platform: scalar.PlatformMacOS, HomeDir: directory, TempDir: directory, WorkingDir: directory,
		LookupEnv: func(name string) (string, bool) {
			if name == "AX_CONFIG" {
				return filename, true
			}
			return "", false
		},
		Stat: os.Stat, ReadFile: os.ReadFile,
	}, nil)
}

func TestHostileV4WithoutHostChannelBinding(t *testing.T) {
	// A Configuration 4 document that selects no host-channel route refuses
	// at load: mesh.host_channel is required and closed in v4, so there is
	// no legacy fallback and no unbound v4 snapshot can reach the launch
	// path. (LaunchForConfig's nil-binding branch is unreachable
	// defense-in-depth behind this loader gate.)
	legacy := v4Document
	legacy = strings.Replace(legacy, "[mesh.host_channel]\n", "", 1)
	legacy = strings.Replace(legacy, "version = '1.0.0'\n", "", 1)
	legacy = strings.Replace(legacy, "credential_id = 'sha256:0000000000000000000000000000000000000000000000000000000000000000'\n", "", 1)
	_, loadErr := tryLoadSnapshot(t, legacy)
	if loadErr == nil {
		t.Fatal("v4 without host_channel loaded")
	}
	if !errors.Is(loadErr, config.ErrConfigValidation) {
		t.Fatalf("loader refusal = %v, want a validation failure", loadErr)
	}
	// The rendered message is redacted by design; the member cause travels
	// the Unwrap chain.
	seen := false
	for err := loadErr; err != nil; err = errors.Unwrap(err) {
		if strings.Contains(err.Error(), "host_channel") {
			seen = true
		}
		if joined, ok := err.(interface{ Unwrap() []error }); ok {
			for _, inner := range joined.Unwrap() {
				if strings.Contains(inner.Error(), "host_channel") {
					seen = true
				}
			}
		}
	}
	if !seen {
		t.Fatalf("loader refusal chain lacks the host_channel cause: %#v", loadErr)
	}
	t.Logf("loader refusal chain pins mesh.host_channel: %v", loadErr)
}

func TestHostileRoleSwap(t *testing.T) {
	// Positive control: the channel admits in the reversed role assignment,
	// proving the rig refuses hostile vectors rather than one direction.
	f := newFixture(t)
	census := censusHostile(t, f)
	handler := &countingHandler{}
	serverConfig := hostchannel.ServerConfig{
		Store: f.storeA, Credential: f.certA,
		LocalHostID: hostA, LocalCredentialID: f.credAID,
		Allowlisted: []string{hostB}, Platform: "linux", AXVersion: "0.6.0",
		Handler: handler.handler(), RequestTimeout: 5 * time.Second,
	}
	clientConfig := hostchannel.ClientConfig{
		Store: f.storeB, Credential: f.certB,
		LocalHostID: hostB, LocalCredentialID: f.credBID,
		ExpectedRemoteHostID: hostA, Allowlisted: []string{hostA},
		Platform: "linux", AXVersion: "0.6.0", RequestTimeout: 5 * time.Second,
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, serverConfig)
	client, err := hostchannel.Dial(clientConn, clientConfig)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	if client.PeerHostID() != hostA {
		t.Fatalf("peer = %q", client.PeerHostID())
	}
	if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
		t.Fatalf("Call = %+v, %v", response, err)
	}
	if handler.count() != 1 {
		t.Fatalf("handler calls = %d", handler.count())
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served)
	census.assertUnchanged(t, f)
}
