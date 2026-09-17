package hostchannel_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/config"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/hostchannel"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// rogueClientConfig builds a raw TLS client the tests drive by hand. The
// pool comes from the production enrollment entry; mutate introduces the
// single hostile deviation under test.
func rogueClientConfig(t *testing.T, f *fixture, cert tls.Certificate, serverName string, mutate func(*tls.Config)) *tls.Config {
	t.Helper()
	snapshot, err := f.storeA.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	pool, err := hostchannel.EnrollmentPool(snapshot, time.Now().UTC())
	if err != nil {
		t.Fatalf("EnrollmentPool: %v", err)
	}
	name := serverName
	if name == "" {
		name, err = hostchannel.ServerName(hostB)
		if err != nil {
			t.Fatalf("ServerName: %v", err)
		}
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   name,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		NextProtos:   []string{"ax-host/1"},
	}
	if mutate != nil {
		mutate(config)
	}
	return config
}

func rogueServerConfig(t *testing.T, f *fixture, cert tls.Certificate, mutate func(*tls.Config)) *tls.Config {
	t.Helper()
	snapshot, err := f.storeB.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	pool, err := hostchannel.EnrollmentPool(snapshot, time.Now().UTC())
	if err != nil {
		t.Fatalf("EnrollmentPool: %v", err)
	}
	config := &tls.Config{
		Certificates:           []tls.Certificate{cert},
		ClientAuth:             tls.RequireAndVerifyClientCert,
		ClientCAs:              pool,
		MinVersion:             tls.VersionTLS13,
		MaxVersion:             tls.VersionTLS13,
		NextProtos:             []string{"ax-host/1"},
		SessionTicketsDisabled: true,
	}
	if mutate != nil {
		mutate(config)
	}
	return config
}

// rogueHello builds a structurally valid RPC-5 hello line from public
// entries, optionally reshaped by edit into an invalid frame.
func rogueHello(t *testing.T, hostID string, edit func(map[string]any)) (rpcwire.Request, []byte, string) {
	t.Helper()
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
	raw, err := json.Marshal(rpcwire.Hello{
		HostID: hostID, Platform: "linux", AXVersion: "0.6.0",
		Nonce: rpcwire.NewNonce(), Contracts: contracts,
		MaxLineBytes: lineBytes, MaxObjectBytes: objectBytes,
	})
	if err != nil {
		t.Fatalf("marshal hello: %v", err)
	}
	id, err := hostchannel.NewRequestID()
	if err != nil {
		t.Fatalf("NewRequestID: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("reshape hello: %v", err)
	}
	if edit != nil {
		edit(body)
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal edited hello: %v", err)
		}
	}
	envelope := map[string]any{
		"protocol": "urn:ax:protocol:rpc", "protocol_version": "5.0.0",
		"request_id": id, "operation": "hello", "body": json.RawMessage(raw),
	}
	line, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	request, err := rpcwire.DecodeRequest(line)
	if edit == nil && err != nil {
		t.Fatalf("DecodeRequest(rogue hello): %v", err)
	}
	return request, line, id
}

func writeRaw(t *testing.T, conn net.Conn, line []byte) {
	t.Helper()
	if _, err := conn.Write(append(line, '\n')); err != nil {
		t.Fatalf("rogue write: %v", err)
	}
}

// readFailure decodes the one encrypted refusal and proves the stream ends
// there: a second frame must not follow.
func readFailure(t *testing.T, conn net.Conn, request rpcwire.Request, sentID string) *axerror.Error {
	t.Helper()
	// A refused peer may already have closed; the deadline then fails and
	// the read below terminates on the closure instead.
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	line, err := hostchannel.ReadLine(conn)
	if err != nil {
		t.Fatalf("ReadLine(refusal): %v", err)
	}
	if request.Operation() == "" {
		var shape struct {
			Version string          `json:"protocol_version"`
			ID      string          `json:"request_id"`
			OK      bool            `json:"ok"`
			Error   json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(line, &shape); err != nil {
			t.Fatalf("decode refusal: %v", err)
		}
		if shape.Version != "5.0.0" || shape.ID != sentID || shape.OK {
			t.Fatalf("refusal correlation = %+v", shape)
		}
		failure, err := axerror.DecodeBound(axerror.ContainingContract{ID: "urn:ax:protocol:rpc", Major: 5}, shape.Error)
		if err != nil {
			t.Fatalf("DecodeBound: %v", err)
		}
		expectClose(t, conn)
		return failure
	}
	response, err := rpcwire.DecodeResponse(line, request)
	if err != nil {
		t.Fatalf("DecodeResponse(refusal): %v", err)
	}
	if response.OK() {
		t.Fatal("refusal decoded as success")
	}
	expectClose(t, conn)
	return response.Failure()
}

func expectClose(t *testing.T, conn net.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if _, err := hostchannel.ReadLine(conn); !errors.Is(err, io.EOF) {
		t.Fatalf("stream did not close: %v", err)
	}
}

// expectRogueHandshakeFails proves a rogue cannot complete a handshake
// against a refusing server. A TLS 1.3 client may report handshake success
// before the server validates its flight, so the rogue reads next: the
// read drains the server alert (which also releases the rendezvous pipe)
// and must fail. Nothing is written: the server never reaches hello.
func expectRogueHandshakeFails(t *testing.T, rogue *tls.Conn) {
	t.Helper()
	_ = rogue.SetReadDeadline(time.Now().Add(10 * time.Second))
	if _, err := rogue.Read(make([]byte, 4096)); err == nil {
		t.Fatal("rogue read succeeded against a refusing server")
	}
}

func assertFailure(t *testing.T, failure *axerror.Error, code string, exit int) {
	t.Helper()
	if failure.Version() != axerror.Version130 {
		t.Fatalf("failure version = %s", failure.Version())
	}
	if string(failure.Code()) != code || failure.ExitCode() != exit {
		t.Fatalf("failure = %q/%d", failure.Code(), failure.ExitCode())
	}
	registryExit, err := axerror.ExitCodeFor(axerror.Version130, failure.Code())
	if err != nil || registryExit != exit {
		t.Fatalf("registry exit for %q = %d, %v", failure.Code(), registryExit, err)
	}
}

func TestRefusesTLS12Offer(t *testing.T) {
	t.Run("server", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", func(config *tls.Config) {
			config.MinVersion = tls.VersionTLS12
			config.MaxVersion = tls.VersionTLS12
		}))
		if err := rogue.Handshake(); err == nil {
			t.Fatal("TLS 1.2 handshake succeeded")
		}
		if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Serve = %v", err)
		}
	})
	t.Run("client", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, func(config *tls.Config) {
			config.MinVersion = tls.VersionTLS12
			config.MaxVersion = tls.VersionTLS12
		}))
		rogueErr := make(chan error, 1)
		go func() { rogueErr <- rogue.Handshake() }()
		if _, err := hostchannel.Dial(clientConn, f.clientConfig()); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Dial = %v", err)
		}
		<-rogueErr
	})
}

func TestRefusesWrongALPN(t *testing.T) {
	t.Run("server", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", func(config *tls.Config) {
			config.NextProtos = []string{"rogue-alpn"}
		}))
		if err := rogue.Handshake(); err == nil {
			t.Fatal("wrong-ALPN handshake succeeded")
		}
		if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Serve = %v", err)
		}
	})
	t.Run("client", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, func(config *tls.Config) {
			config.NextProtos = []string{"rogue-alpn"}
		}))
		rogueErr := make(chan error, 1)
		go func() { rogueErr <- rogue.Handshake() }()
		if _, err := hostchannel.Dial(clientConn, f.clientConfig()); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Dial = %v", err)
		}
		<-rogueErr
	})
}

func TestRefusesMissingALPN(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", func(config *tls.Config) {
		config.NextProtos = nil
	}))
	_ = rogue.Handshake()
	expectRogueHandshakeFails(t, rogue)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesResumption(t *testing.T) {
	f := newFixture(t)
	cache := tls.NewLRUClientSessionCache(16)
	for round := 0; round < 2; round++ {
		clientConn, serverConn := net.Pipe()
		served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", func(config *tls.Config) {
			config.ClientSessionCache = cache
		}))
		if err := rogue.Handshake(); err != nil {
			clientConn.Close()
			serverConn.Close()
			t.Fatalf("round %d handshake: %v", round, err)
		}
		if rogue.ConnectionState().DidResume {
			clientConn.Close()
			serverConn.Close()
			t.Fatalf("round %d resumed", round)
		}
		request, line, _ := rogueHello(t, hostA, nil)
		writeRaw(t, rogue, line)
		helloLine, err := hostchannel.ReadLine(rogue)
		if err != nil {
			clientConn.Close()
			serverConn.Close()
			t.Fatalf("round %d hello: %v", round, err)
		}
		if response, err := rpcwire.DecodeResponse(helloLine, request); err != nil || !response.OK() {
			clientConn.Close()
			serverConn.Close()
			t.Fatalf("round %d hello response: %+v, %v", round, response, err)
		}
		clientConn.Close()
		serverConn.Close()
		_ = awaitErr(t, served)
	}
}

func TestRefusesPlaintextPreface(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	if _, err := clientConn.Write([]byte("GET / HTTP/1.0\r\n\r\n")); err != nil {
		t.Fatalf("rogue write: %v", err)
	}
	_ = clientConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	first := make([]byte, 1)
	_, err := clientConn.Read(first)
	switch {
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe):
		// Closed without sending: permitted, only JSON is forbidden.
	case err != nil:
		t.Fatalf("rogue read: %v", err)
	case first[0] == '{':
		t.Fatal("JSON sent before TLS completed")
	case first[0] != 0x15:
		t.Fatalf("preface reply = %x, want a TLS alert or close", first)
	}
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesUnenrolledRoot(t *testing.T) {
	f := newFixture(t)
	attacker, err := hosttrust.IssueCredential(hostC, time.Now().UTC(), nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	attackerCert, err := hostchannel.CredentialFromIssued(attacker)
	if err != nil {
		t.Fatalf("CredentialFromIssued: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, attackerCert, "", nil))
	_ = rogue.Handshake()
	expectRogueHandshakeFails(t, rogue)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesRevokedLeaf(t *testing.T) {
	f := newFixture(t)
	issuedC, err := hosttrust.IssueCredential(hostC, time.Now().UTC(), nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	enroll(t, f.storeB, issuedC, hostC, time.Now().UTC())
	certC, err := hostchannel.CredentialFromIssued(issuedC)
	if err != nil {
		t.Fatalf("CredentialFromIssued: %v", err)
	}
	if err := f.storeB.Revoke(scalar.SHA256Digest(issuedC.LeafDER).String()); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, certC, "", nil))
	_ = rogue.Handshake()
	expectRogueHandshakeFails(t, rogue)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesRetiredPastRetireAt(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	credentialID := scalar.SHA256Digest(f.issuedA.LeafDER).String()
	if err := f.storeB.MarkRetiring(credentialID, now.Add(time.Hour), now); err != nil {
		t.Fatalf("MarkRetiring: %v", err)
	}
	serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
	serverConfig.Now = func() time.Time { return now.Add(2 * time.Hour) }
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, serverConfig)
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
	_ = rogue.Handshake()
	expectRogueHandshakeFails(t, rogue)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesExpiredLeaf(t *testing.T) {
	f := newFixture(t)
	issuedAt := time.Now().UTC().Add(-100 * 24 * time.Hour)
	aged, err := hosttrust.IssueCredential(hostC, issuedAt, nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	enroll(t, f.storeB, aged, hostC, issuedAt)
	certC, err := hostchannel.CredentialFromIssued(aged)
	if err != nil {
		t.Fatalf("CredentialFromIssued: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, certC, "", nil))
	_ = rogue.Handshake()
	expectRogueHandshakeFails(t, rogue)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesWrongServerName(t *testing.T) {
	f := newFixture(t)
	issuedC, err := hosttrust.IssueCredential(hostC, time.Now().UTC(), nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	certC, err := hostchannel.CredentialFromIssued(issuedC)
	if err != nil {
		t.Fatalf("CredentialFromIssued: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	rogue := tls.Server(serverConn, rogueServerConfig(t, f, certC, nil))
	rogueErr := make(chan error, 1)
	go func() { rogueErr <- rogue.Handshake() }()
	if _, err := hostchannel.Dial(clientConn, f.clientConfig()); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Dial = %v", err)
	}
	<-rogueErr
}

func TestRefusesHelloUUIDMismatch(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
	if err := rogue.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	request, line, _ := rogueHello(t, hostC, nil)
	writeRaw(t, rogue, line)
	assertFailure(t, readFailure(t, rogue, request, ""), "host_identity_mismatch", 7)
	if err := awaitErr(t, served); !errors.Is(err, hosttrust.ErrHostIdentityMismatch) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesServerHelloUUIDMismatch(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
	dialed := make(chan error, 1)
	go func() {
		client, err := hostchannel.Dial(clientConn, f.clientConfig())
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
	lineBytes, _ := scalar.NewUint53(rpcwire.MaxLineBytes)
	objectBytes, _ := scalar.NewUint53(rpcwire.MinObjectBytes)
	body, err := json.Marshal(rpcwire.Hello{
		HostID: hostC, Platform: "linux", AXVersion: "0.6.0",
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
	writeRaw(t, rogue, success)
	select {
	case err := <-dialed:
		if !errors.Is(err, hosttrust.ErrHostIdentityMismatch) {
			t.Fatalf("Dial = %v", err)
		}
		if !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
			t.Fatalf("Dial is not an authentication failure: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Dial did not return")
	}
}

func TestRefusesDestinationMismatch(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	clientConfig := f.clientConfig()
	clientConfig.ExpectedRemoteHostID = hostC
	if _, err := hostchannel.Dial(clientConn, clientConfig); !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Dial = %v", err)
	}
	_ = awaitErr(t, served)
}

func TestRefusesNotAllowlisted(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
	serverConfig.Allowlisted = []string{hostC}
	served := serveAsync(serverConn, serverConfig)
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
	if err := rogue.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	request, line, _ := rogueHello(t, hostA, nil)
	writeRaw(t, rogue, line)
	assertFailure(t, readFailure(t, rogue, request, ""), "peer_not_allowlisted", 7)
	if err := awaitErr(t, served); !errors.Is(err, hosttrust.ErrNotAllowlisted) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesNonHelloBeforeHello(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
	if err := rogue.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	id, err := hostchannel.NewRequestID()
	if err != nil {
		t.Fatalf("NewRequestID: %v", err)
	}
	line, err := rpcwire.EncodeRequest("5.0.0", id, "health.get", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("EncodeRequest: %v", err)
	}
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		t.Fatalf("DecodeRequest: %v", err)
	}
	writeRaw(t, rogue, line)
	assertFailure(t, readFailure(t, rogue, request, ""), "incompatible_protocol", 6)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrHelloRequired) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestRefusesInvalidHello(t *testing.T) {
	cases := map[string]func(map[string]any){
		"contract": func(body map[string]any) {
			body["contracts"].(map[string]any)["rpc"] = []string{"4.0.0"}
		},
		"missing-key": func(body map[string]any) {
			delete(body["contracts"].(map[string]any), "lease")
		},
		"nonce": func(body map[string]any) { body["nonce"] = "short" },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
			rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
			if err := rogue.Handshake(); err != nil {
				t.Fatalf("handshake: %v", err)
			}
			_, line, sentID := rogueHello(t, hostA, edit)
			writeRaw(t, rogue, line)
			assertFailure(t, readFailure(t, rogue, rpcwire.Request{}, sentID), "incompatible_protocol", 6)
			if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrProtocol) {
				t.Fatalf("Serve = %v", err)
			}
		})
	}
}

func TestForeignMajorClosesSilently(t *testing.T) {
	for _, version := range []string{"2.0.0", "4.0.0"} {
		t.Run(version, func(t *testing.T) {
			f := newFixture(t)
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
			rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
			if err := rogue.Handshake(); err != nil {
				t.Fatalf("handshake: %v", err)
			}
			contracts, err := rpcwire.ContractProfile(version)
			if err != nil {
				t.Fatalf("ContractProfile: %v", err)
			}
			lineBytes, _ := scalar.NewUint53(rpcwire.MaxLineBytes)
			objectBytes, _ := scalar.NewUint53(rpcwire.MinObjectBytes)
			body, err := json.Marshal(rpcwire.Hello{
				HostID: hostA, Platform: "linux", AXVersion: "0.6.0",
				Nonce: rpcwire.NewNonce(), Contracts: contracts,
				MaxLineBytes: lineBytes, MaxObjectBytes: objectBytes,
			})
			if err != nil {
				t.Fatalf("marshal hello: %v", err)
			}
			id, err := hostchannel.NewRequestID()
			if err != nil {
				t.Fatalf("NewRequestID: %v", err)
			}
			line, err := rpcwire.EncodeRequest(version, id, "hello", body)
			if err != nil {
				t.Fatalf("EncodeRequest: %v", err)
			}
			writeRaw(t, rogue, line)
			expectClose(t, rogue)
			if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrUnframeableInput) {
				t.Fatalf("Serve = %v", err)
			}
		})
	}
}

func TestUnframeableHelloClosesSilently(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
	if err := rogue.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	writeRaw(t, rogue, []byte(`{"protocol":"urn:ax:protocol:rpc",`))
	expectClose(t, rogue)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrUnframeableInput) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestHandshakeLimiterBounds(t *testing.T) {
	exact := bytes.NewReader(bytes.Repeat([]byte{0xAA}, hostchannel.MaxHandshakeBytes))
	limiter := hostchannel.NewHandshakeLimiter(exact)
	if _, err := io.Copy(io.Discard, limiter); err != nil {
		t.Fatalf("exact cap: %v", err)
	}
	if limiter.Exceeded() {
		t.Fatal("exact cap latched exceeded")
	}
	over := bytes.NewReader(bytes.Repeat([]byte{0xAA}, hostchannel.MaxHandshakeBytes+1))
	limiter = hostchannel.NewHandshakeLimiter(over)
	_, err := io.Copy(io.Discard, limiter)
	if !errors.Is(err, hostchannel.ErrTransportFailure) {
		t.Fatalf("over cap: %v", err)
	}
	if !strings.Contains(err.Error(), "byte cap") {
		t.Fatalf("over cap = %v, want the byte-cap refusal", err)
	}
	if !limiter.Exceeded() {
		t.Fatal("exceeded not latched")
	}
	// Disarming after local handshake success ends accounting: later
	// application bytes never count against the handshake budget.
	large := bytes.NewReader(bytes.Repeat([]byte{0xAA}, hostchannel.MaxHandshakeBytes+100))
	limiter = hostchannel.NewHandshakeLimiter(large)
	if _, err := io.CopyN(io.Discard, limiter, 100); err != nil {
		t.Fatalf("prefix: %v", err)
	}
	limiter.Disarm()
	if _, err := io.Copy(io.Discard, limiter); err != nil {
		t.Fatalf("disarmed: %v", err)
	}
	if limiter.Exceeded() {
		t.Fatal("disarmed limiter latched exceeded")
	}
}

// countingStream measures hostile-path intake.
type countingStream struct {
	net.Conn
	reads atomic.Int64
}

func (s *countingStream) Read(data []byte) (int, error) {
	n, err := s.Conn.Read(data)
	s.reads.Add(int64(n))
	return n, err
}

func TestHandshakeIntakeIsBounded(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	counted := &countingStream{Conn: serverConn}
	served := serveAsync(counted, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	wrote := make(chan error, 1)
	go func() {
		_, err := clientConn.Write(bytes.Repeat([]byte("A"), 4*1024*1024))
		wrote <- err
	}()
	start := time.Now()
	err := awaitErr(t, served)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("hostile handshake took %v", elapsed)
	}
	if !errors.Is(err, hostchannel.ErrAuthenticationFailed) {
		t.Fatalf("Serve = %v", err)
	}
	if got := counted.reads.Load(); got >= 64*1024 {
		t.Fatalf("hostile intake = %d bytes", got)
	}
	<-wrote
}

func TestSilentPeerHandshakeDeadline(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
	serverConfig.HandshakeTimeout = 200 * time.Millisecond
	served := serveAsync(serverConn, serverConfig)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrTransportFailure) {
		t.Fatalf("Serve = %v", err)
	} else if !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("Serve = %v, want the deadline refusal", err)
	}
}

func TestHelloDeadline(t *testing.T) {
	t.Run("server", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
		serverConfig.HelloTimeout = 200 * time.Millisecond
		served := serveAsync(serverConn, serverConfig)
		rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrTransportFailure) {
			t.Fatalf("Serve = %v", err)
		} else if !strings.Contains(err.Error(), "deadline") {
			t.Fatalf("Serve = %v, want the deadline refusal", err)
		}
	})
	t.Run("client", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
		dialed := make(chan error, 1)
		go func() {
			clientConfig := f.clientConfig()
			clientConfig.HelloTimeout = 200 * time.Millisecond
			client, err := hostchannel.Dial(clientConn, clientConfig)
			if err == nil {
				_ = client.Close()
			}
			dialed <- err
		}()
		if err := rogue.Handshake(); err != nil {
			t.Fatalf("handshake: %v", err)
		}
		// The silent server still drains the hello; it just never answers.
		if _, err := hostchannel.ReadLine(rogue); err != nil {
			t.Fatalf("ReadLine(hello): %v", err)
		}
		select {
		case err := <-dialed:
			if !errors.Is(err, hostchannel.ErrTransportFailure) {
				t.Fatalf("Dial = %v", err)
			}
			if !strings.Contains(err.Error(), "deadline") {
				t.Fatalf("Dial = %v, want the deadline refusal", err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("Dial did not return")
		}
	})
}

func TestOversizeHelloLine(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	rogue := tls.Client(clientConn, rogueClientConfig(t, f, f.certA, "", nil))
	if err := rogue.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	junk := bytes.Repeat([]byte("B"), rpcwire.MaxLineBytes+10)
	wrote := make(chan error, 1)
	go func() {
		_, err := rogue.Write(append(junk, '\n'))
		wrote <- err
	}()
	expectClose(t, rogue)
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrUnframeableInput) {
		t.Fatalf("Serve = %v", err)
	}
	<-wrote
}

func TestStaleGenerationAtDispatch(t *testing.T) {
	t.Run("server closes", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
		client, err := hostchannel.Dial(clientConn, f.clientConfig())
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer client.Close()
		issuedC, err := hosttrust.IssueCredential(hostC, time.Now().UTC(), nil)
		if err != nil {
			t.Fatalf("IssueCredential: %v", err)
		}
		enroll(t, f.storeB, issuedC, hostC, time.Now().UTC())
		if _, err := client.Call("health.get", json.RawMessage(`{}`)); !errors.Is(err, hostchannel.ErrTransportFailure) {
			t.Fatalf("Call = %v", err)
		}
		if err := awaitErr(t, served); !errors.Is(err, hosttrust.ErrStaleGeneration) {
			t.Fatalf("Serve = %v", err)
		} else if !strings.Contains(err.Error(), "moved") {
			t.Fatalf("Serve = %v, want the channel currency gate", err)
		}
	})
	t.Run("client refuses to send", func(t *testing.T) {
		f := newFixture(t)
		var calls int
		handler := func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
			calls++
			return json.RawMessage(`{}`), nil
		}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		served := serveAsync(serverConn, f.serverConfig(handler))
		client, err := hostchannel.Dial(clientConn, f.clientConfig())
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer client.Close()
		if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
			t.Fatalf("Call = %+v, %v", response, err)
		}
		issuedC, err := hosttrust.IssueCredential(hostC, time.Now().UTC(), nil)
		if err != nil {
			t.Fatalf("IssueCredential: %v", err)
		}
		enroll(t, f.storeA, issuedC, hostC, time.Now().UTC())
		if _, err := client.Call("health.get", json.RawMessage(`{}`)); !errors.Is(err, hosttrust.ErrStaleGeneration) {
			t.Fatalf("second Call = %v", err)
		}
		if calls != 1 {
			t.Fatalf("handler calls = %d, stale Call transmitted", calls)
		}
		if err := client.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		_ = awaitErr(t, served)
	})
}

func TestStaleGenerationAtMutation(t *testing.T) {
	f := newFixture(t)
	admitted := make(chan struct{})
	committed := make(chan struct{})
	var ran int
	handler := func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, mutate func(func() error) error) (json.RawMessage, error) {
		close(admitted)
		<-committed
		if err := mutate(func() error { ran++; return nil }); err != nil {
			return nil, err
		}
		return json.RawMessage(`{}`), nil
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(handler))
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	called := make(chan error, 1)
	go func() {
		_, err := client.Call("health.get", json.RawMessage(`{}`))
		called <- err
	}()
	select {
	case <-admitted:
	case <-time.After(30 * time.Second):
		t.Fatal("dispatch was not admitted")
	}
	issuedC, err := hosttrust.IssueCredential(hostC, time.Now().UTC(), nil)
	if err != nil {
		t.Fatalf("IssueCredential: %v", err)
	}
	enroll(t, f.storeB, issuedC, hostC, time.Now().UTC())
	close(committed)
	select {
	case err := <-called:
		if !errors.Is(err, hostchannel.ErrTransportFailure) {
			t.Fatalf("Call = %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Call did not return")
	}
	if ran != 0 {
		t.Fatal("stale boundary ran")
	}
	if err := awaitErr(t, served); !errors.Is(err, hosttrust.ErrStaleGeneration) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestHandlerErrorFramedAndContinues(t *testing.T) {
	f := newFixture(t)
	handler := func(_ context.Context, request rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		if request.Operation() == "boom" {
			return nil, errors.New("handler refused")
		}
		return json.RawMessage(`{}`), nil
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(handler))
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	response, err := client.Call("boom", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Call(boom) = %v", err)
	}
	if response.OK() {
		t.Fatal("boom call succeeded")
	}
	assertFailure(t, response.Failure(), "incompatible_protocol", 6)
	response, err = client.Call("health.get", json.RawMessage(`{}`))
	if err != nil || !response.OK() {
		t.Fatalf("Call after failure = %+v, %v", response, err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served)
}

func TestRehelloIsPlainDispatch(t *testing.T) {
	f := newFixture(t)
	var sawHello bool
	handler := func(_ context.Context, request rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		if request.Operation() == "hello" {
			sawHello = true
			return nil, errors.New("no re-hello")
		}
		return json.RawMessage(`{}`), nil
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(handler))
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	_, line, _ := rogueHello(t, hostA, nil)
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		t.Fatalf("DecodeRequest: %v", err)
	}
	response, err := client.Call("hello", request.Body())
	if err != nil {
		t.Fatalf("Call(hello) = %v", err)
	}
	if response.OK() {
		t.Fatal("re-hello succeeded")
	}
	assertFailure(t, response.Failure(), "incompatible_protocol", 6)
	if !sawHello {
		t.Fatal("re-hello never reached the handler")
	}
	response, err = client.Call("health.get", json.RawMessage(`{}`))
	if err != nil || !response.OK() {
		t.Fatalf("Call after re-hello = %+v, %v", response, err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served)
}

func TestHelloResponseCorrelation(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
	dialed := make(chan error, 1)
	go func() {
		client, err := hostchannel.Dial(clientConn, f.clientConfig())
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
	otherID, err := hostchannel.NewRequestID()
	if err != nil {
		t.Fatalf("NewRequestID: %v", err)
	}
	reply, err := json.Marshal(map[string]any{
		"protocol": "urn:ax:protocol:rpc", "protocol_version": "5.0.0",
		"request_id": otherID, "ok": true, "body": json.RawMessage(body),
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
	case <-time.After(30 * time.Second):
		t.Fatal("Dial did not return")
	}
}

func TestDispatchTimeout(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
	serverConfig.RequestTimeout = 200 * time.Millisecond
	served := serveAsync(serverConn, serverConfig)
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrTransportFailure) {
		t.Fatalf("Serve = %v", err)
	} else if !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("Serve = %v, want the deadline refusal", err)
	}
}

func TestHelloFailureMapping(t *testing.T) {
	cases := []struct {
		code string
		want error
	}{
		{"host_identity_mismatch", hosttrust.ErrHostIdentityMismatch},
		{"peer_not_allowlisted", hosttrust.ErrNotAllowlisted},
		{"incompatible_protocol", hostchannel.ErrProtocol},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			f := newFixture(t)
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()
			rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
			dialed := make(chan error, 1)
			go func() {
				client, err := hostchannel.Dial(clientConn, f.clientConfig())
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
			failure, err := axerror.New(axerror.Spec{
				Version: axerror.Version130, Code: axerror.Code(tc.code),
				Message: "rogue refusal", IDs: axerror.NoIDs(), Details: axerror.Details{},
			})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			frame, err := rpcwire.EncodeFailure(request, failure)
			if err != nil {
				t.Fatalf("EncodeFailure: %v", err)
			}
			writeRaw(t, rogue, frame)
			select {
			case err := <-dialed:
				if !errors.Is(err, tc.want) {
					t.Fatalf("Dial = %v", err)
				}
			case <-time.After(30 * time.Second):
				t.Fatal("Dial did not return")
			}
		})
	}
}

func TestDispatchExpiryClosesWithoutFrame(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	credentialID := scalar.SHA256Digest(f.issuedA.LeafDER).String()
	if err := f.storeB.MarkRetiring(credentialID, now.Add(time.Hour), now); err != nil {
		t.Fatalf("MarkRetiring: %v", err)
	}
	var nowNanos atomic.Int64
	nowNanos.Store(now.UnixNano())
	serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
	serverConfig.Now = func() time.Time { return time.Unix(0, nowNanos.Load()).UTC() }
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, serverConfig)
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
		t.Fatalf("Call = %+v, %v", response, err)
	}
	nowNanos.Store(now.Add(2 * time.Hour).UnixNano())
	if _, err := client.Call("health.get", json.RawMessage(`{}`)); !errors.Is(err, hostchannel.ErrTransportFailure) {
		t.Fatalf("expired Call = %v", err)
	}
	if err := awaitErr(t, served); !errors.Is(err, hosttrust.ErrAuthorizationRefused) {
		t.Fatalf("Serve = %v", err)
	} else if errors.Is(err, hosttrust.ErrStaleGeneration) {
		t.Fatalf("Serve is stale, want expiry: %v", err)
	}
}

func TestHelloWriteDeadline(t *testing.T) {
	f := newFixture(t)
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	rogue := tls.Server(serverConn, rogueServerConfig(t, f, f.certB, nil))
	dialed := make(chan error, 1)
	go func() {
		clientConfig := f.clientConfig()
		clientConfig.HelloTimeout = 200 * time.Millisecond
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err == nil {
			_ = client.Close()
		}
		dialed <- err
	}()
	if err := rogue.Handshake(); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	// The rogue never reads the hello; the bounded write must fail, not hang.
	select {
	case err := <-dialed:
		if !errors.Is(err, hostchannel.ErrTransportFailure) {
			t.Fatalf("Dial = %v", err)
		}
		if !strings.Contains(err.Error(), "deadline") {
			t.Fatalf("Dial = %v, want the deadline refusal", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Dial did not return")
	}
}

func TestCallRefusesInvalidBody(t *testing.T) {
	f := newFixture(t)
	var calls int
	handler := func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		calls++
		return json.RawMessage(`{}`), nil
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(handler))
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	if _, err := client.Call("health.get", json.RawMessage(`{`)); err == nil {
		t.Fatal("Call accepted a malformed body")
	}
	if calls != 0 {
		t.Fatalf("handler calls = %d, invalid Call transmitted", calls)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served)
}

func TestHandlerGarbageBodyCloses(t *testing.T) {
	f := newFixture(t)
	handler := func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		return json.RawMessage(`{"roots":`), nil
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(handler))
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()
	if _, err := client.Call("health.get", json.RawMessage(`{}`)); !errors.Is(err, hostchannel.ErrTransportFailure) {
		t.Fatalf("Call = %v", err)
	}
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrTransportFailure) {
		t.Fatalf("Serve = %v", err)
	}
}

func TestCredentialFromIssuedRefusals(t *testing.T) {
	f := newFixture(t)
	badKey := f.issuedA
	badKey.LeafKeyDER = []byte("not-a-key")
	if _, err := hostchannel.CredentialFromIssued(badKey); !errors.Is(err, hostchannel.ErrInvalidConfig) {
		t.Fatalf("bad key = %v", err)
	}
	badLeaf := f.issuedA
	badLeaf.LeafDER = []byte("not-a-cert")
	if _, err := hostchannel.CredentialFromIssued(badLeaf); !errors.Is(err, hostchannel.ErrInvalidConfig) {
		t.Fatalf("bad leaf = %v", err)
	}
}

func TestLocalCredentialRefusals(t *testing.T) {
	t.Run("wrong credential id", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
		serverConfig.LocalCredentialID = f.credAID
		if err := hostchannel.Serve(serverConn, serverConfig); !errors.Is(err, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Serve = %v", err)
		}
	})
	t.Run("revoked self", func(t *testing.T) {
		f := newFixture(t)
		if err := f.storeB.Revoke(f.credBID); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		if err := hostchannel.Serve(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`)))); !errors.Is(err, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Serve = %v", err)
		}
	})
	t.Run("nil stream", func(t *testing.T) {
		f := newFixture(t)
		if _, err := hostchannel.Dial(nil, f.clientConfig()); !errors.Is(err, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Dial = %v", err)
		}
	})
	t.Run("default timeouts", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
		serverConfig.RequestTimeout = 0
		serverConfig.HelloTimeout = 0
		serverConfig.HandshakeTimeout = 0
		served := serveAsync(serverConn, serverConfig)
		clientConfig := f.clientConfig()
		clientConfig.RequestTimeout = 0
		clientConfig.HelloTimeout = 0
		clientConfig.HandshakeTimeout = 0
		client, err := hostchannel.Dial(clientConn, clientConfig)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		defer client.Close()
		if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
			t.Fatalf("Call = %+v, %v", response, err)
		}
		if err := client.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		_ = awaitErr(t, served)
	})
}

func TestLaunchForConfigEmptySnapshot(t *testing.T) {
	if _, err := hostchannel.LaunchForConfig(config.Snapshot{}, hostB); !errors.Is(err, hostchannel.ErrInvalidConfig) {
		t.Fatalf("empty snapshot = %v", err)
	}
}

func TestInvalidActivation(t *testing.T) {
	t.Run("bad destination", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		clientConfig := f.clientConfig()
		clientConfig.ExpectedRemoteHostID = "not-a-uuid"
		// Shortened-deadline hook: an invalid destination refuses at
		// selection, before any I/O, and must never reach the 10 s
		// handshake/hello bounds.
		clientConfig.HandshakeTimeout = 500 * time.Millisecond
		clientConfig.HelloTimeout = 500 * time.Millisecond
		clientConfig.RequestTimeout = 500 * time.Millisecond
		start := time.Now()
		if _, err := hostchannel.Dial(clientConn, clientConfig); !errors.Is(err, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Dial = %v", err)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("bad destination took %v", elapsed)
		}
	})
	t.Run("nil store", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
		serverConfig.Store = nil
		if err := hostchannel.Serve(serverConn, serverConfig); !errors.Is(err, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Serve = %v", err)
		}
	})
	t.Run("credential host mismatch", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
		serverConfig.LocalHostID = hostA
		if err := hostchannel.Serve(serverConn, serverConfig); !errors.Is(err, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Serve = %v", err)
		}
	})
	t.Run("bad platform", func(t *testing.T) {
		f := newFixture(t)
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
		serverConfig.Platform = "ios"
		if err := hostchannel.Serve(serverConn, serverConfig); !errors.Is(err, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Serve = %v", err)
		}
	})
	t.Run("missing store", func(t *testing.T) {
		f := newFixture(t)
		empty := t.TempDir()
		store, err := hosttrust.Open(testPaths(t, empty+"/config.toml", empty))
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()
		serverConfig := f.serverConfig(echoHandler(json.RawMessage(`{}`)))
		serverConfig.Store = store
		if err := hostchannel.Serve(serverConn, serverConfig); !errors.Is(err, hostchannel.ErrInvalidConfig) {
			t.Fatalf("Serve = %v", err)
		}
	})
}
