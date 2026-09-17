package hostchannel_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/hostchannel"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	hostA = "0198f4c8-7d40-7e55-8e6f-1234567890aa"
	hostB = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	hostC = "0198f4c8-7d40-7e55-8e6f-1234567890ac"
)

type fixture struct {
	now     time.Time
	dirA    string
	dirB    string
	storeA  *hosttrust.Store
	storeB  *hosttrust.Store
	issuedA hosttrust.IssuedCredential
	issuedB hosttrust.IssuedCredential
	certA   tls.Certificate
	certB   tls.Certificate
	credAID string
	credBID string
}

func testPaths(t *testing.T, configPath, stateDir string) localstore.ResolvedPaths {
	t.Helper()
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{
		Platform: scalar.PlatformLinux,
		Flags: map[string]string{
			"--config":      configPath,
			"--data-dir":    filepath.Join(stateDir, "data"),
			"--state-dir":   stateDir,
			"--cache-dir":   filepath.Join(stateDir, "cache"),
			"--runtime-dir": filepath.Join(stateDir, "runtime"),
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	return paths
}

func openStore(t *testing.T, stateDir string) *hosttrust.Store {
	t.Helper()
	store, err := hosttrust.Open(testPaths(t, filepath.Join(stateDir, "config.toml"), stateDir))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := store.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return store
}

func enroll(t *testing.T, store *hosttrust.Store, issued hosttrust.IssuedCredential, hostID string, at time.Time) string {
	t.Helper()
	fingerprints, err := hosttrust.FingerprintsOf(issued.LeafDER, issued.RootDER)
	if err != nil {
		t.Fatalf("FingerprintsOf: %v", err)
	}
	credentialID, err := store.Enroll(hosttrust.EnrollInput{
		HostID:     hostID,
		LeafDER:    issued.LeafDER,
		RootDER:    issued.RootDER,
		Authorized: fingerprints,
		EnrolledAt: at,
	})
	if err != nil {
		t.Fatalf("Enroll(%s): %v", hostID, err)
	}
	return credentialID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	f := &fixture{now: now, dirA: t.TempDir(), dirB: t.TempDir()}
	f.storeA = openStore(t, f.dirA)
	f.storeB = openStore(t, f.dirB)
	var err error
	f.issuedA, err = hosttrust.IssueCredential(hostA, now, nil)
	if err != nil {
		t.Fatalf("IssueCredential A: %v", err)
	}
	f.issuedB, err = hosttrust.IssueCredential(hostB, now, nil)
	if err != nil {
		t.Fatalf("IssueCredential B: %v", err)
	}
	enroll(t, f.storeA, f.issuedA, hostA, now)
	enroll(t, f.storeA, f.issuedB, hostB, now)
	enroll(t, f.storeB, f.issuedA, hostA, now)
	enroll(t, f.storeB, f.issuedB, hostB, now)
	f.certA, err = hostchannel.CredentialFromIssued(f.issuedA)
	if err != nil {
		t.Fatalf("CredentialFromIssued A: %v", err)
	}
	f.certB, err = hostchannel.CredentialFromIssued(f.issuedB)
	if err != nil {
		t.Fatalf("CredentialFromIssued B: %v", err)
	}
	f.credAID = scalar.SHA256Digest(f.issuedA.LeafDER).String()
	f.credBID = scalar.SHA256Digest(f.issuedB.LeafDER).String()
	return f
}

func (f *fixture) serverConfig(handler hostchannel.Handler) hostchannel.ServerConfig {
	return hostchannel.ServerConfig{
		Store:             f.storeB,
		Credential:        f.certB,
		LocalHostID:       hostB,
		LocalCredentialID: f.credBID,
		Allowlisted:       []string{hostA},
		Platform:          "linux",
		AXVersion:         "0.6.0",
		Handler:           handler,
		RequestTimeout:    5 * time.Second,
	}
}

func (f *fixture) clientConfig() hostchannel.ClientConfig {
	return hostchannel.ClientConfig{
		Store:                f.storeA,
		Credential:           f.certA,
		LocalHostID:          hostA,
		LocalCredentialID:    f.credAID,
		ExpectedRemoteHostID: hostB,
		Allowlisted:          []string{hostB},
		Platform:             "linux",
		AXVersion:            "0.6.0",
		RequestTimeout:       5 * time.Second,
	}
}

func serveAsync(stream hostchannel.Stream, config hostchannel.ServerConfig) chan error {
	done := make(chan error, 1)
	go func() { done <- hostchannel.Serve(stream, config) }()
	return done
}

func awaitErr(t *testing.T, done chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the server")
		return nil
	}
}

func echoHandler(body json.RawMessage) hostchannel.Handler {
	return func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		return body, nil
	}
}

func TestFullHandshakeHelloAndOperation(t *testing.T) {
	f := newFixture(t)
	var seen hostchannel.Binding
	var calls int
	handler := func(_ context.Context, request rpcwire.Request, binding hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		calls++
		seen = binding
		if request.Operation() != "health.get" {
			return nil, errors.New("unexpected operation")
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
	state := client.ConnectionState()
	if state.Version != tls.VersionTLS13 {
		t.Fatalf("negotiated TLS version = %x", state.Version)
	}
	if state.NegotiatedProtocol != "ax-host/1" {
		t.Fatalf("negotiated ALPN = %q", state.NegotiatedProtocol)
	}
	if state.DidResume {
		t.Fatal("connection resumed")
	}
	if client.PeerHostID() != hostB {
		t.Fatalf("peer = %q", client.PeerHostID())
	}
	if client.Binding().HelloHostID != hostB || client.Binding().RemoteHostID != hostB {
		t.Fatalf("client binding = %+v", client.Binding())
	}
	response, err := client.Call("health.get", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !response.OK() || string(response.Body()) != `{}` {
		t.Fatalf("response = %+v", response)
	}
	if calls != 1 {
		t.Fatalf("handler calls = %d", calls)
	}
	if seen.RemoteHostID != hostA || seen.HelloHostID != hostA || seen.LocalHostID != hostB {
		t.Fatalf("server binding = %+v", seen)
	}
	snapshot, err := f.storeB.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if seen.Generation != snapshot.Generation {
		t.Fatalf("server bound generation %d, committed %d", seen.Generation, snapshot.Generation)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := awaitErr(t, served); !errors.Is(err, hostchannel.ErrTransportFailure) {
		t.Fatalf("Serve after close = %v", err)
	}
}

func TestInventoryRootsThroughChannel(t *testing.T) {
	f := newFixture(t)
	handler := func(_ context.Context, request rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		var shape struct {
			Namespaces []string `json:"namespaces"`
		}
		if err := json.Unmarshal(request.Body(), &shape); err != nil {
			return nil, err
		}
		roots := make([]map[string]any, 0, len(shape.Namespaces))
		for _, name := range shape.Namespaces {
			roots = append(roots, map[string]any{
				"namespace": name,
				"count":     0,
				"root_id":   "sha256:" + strings.Repeat("0", 64),
			})
		}
		return json.Marshal(map[string]any{"roots": roots})
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
	names, err := rpcwire.Namespaces("5.0.0")
	if err != nil {
		t.Fatalf("Namespaces: %v", err)
	}
	if len(names) != 8 {
		t.Fatalf("RPC-5 namespaces = %d", len(names))
	}
	query, err := json.Marshal(map[string]any{"namespaces": names})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	response, err := client.Call("inventory.roots", query)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !response.OK() {
		t.Fatalf("response failure = %+v", response.Failure())
	}
	var shape struct {
		Roots []rpcwire.Root `json:"roots"`
	}
	if err := json.Unmarshal(response.Body(), &shape); err != nil {
		t.Fatalf("decode roots: %v", err)
	}
	if len(shape.Roots) != 8 || shape.Roots[0].Namespace != names[0] {
		t.Fatalf("roots = %+v", shape.Roots)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served)
}

func TestMutationBoundaryRunsFresh(t *testing.T) {
	f := newFixture(t)
	var ran int
	handler := func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, mutate func(func() error) error) (json.RawMessage, error) {
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
	response, err := client.Call("health.get", json.RawMessage(`{}`))
	if err != nil || !response.OK() {
		t.Fatalf("Call = %+v, %v", response, err)
	}
	if ran != 1 {
		t.Fatalf("boundary ran %d times", ran)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served)
}

func TestRetiringCredentialAdmitsWithinBound(t *testing.T) {
	f := newFixture(t)
	now := time.Now().UTC()
	credentialID := scalar.SHA256Digest(f.issuedA.LeafDER).String()
	if err := f.storeB.MarkRetiring(credentialID, now.Add(time.Hour), now); err != nil {
		t.Fatalf("MarkRetiring: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	served := serveAsync(serverConn, f.serverConfig(echoHandler(json.RawMessage(`{}`))))
	client, err := hostchannel.Dial(clientConn, f.clientConfig())
	if err != nil {
		t.Fatalf("Dial with retiring peer: %v", err)
	}
	defer client.Close()
	if response, err := client.Call("health.get", json.RawMessage(`{}`)); err != nil || !response.OK() {
		t.Fatalf("Call = %+v, %v", response, err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_ = awaitErr(t, served)
}

func TestServerNameMatchesIssuedLeaf(t *testing.T) {
	f := newFixture(t)
	leaf, err := x509.ParseCertificate(f.issuedB.LeafDER)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}
	name, err := hostchannel.ServerName(hostB)
	if err != nil {
		t.Fatalf("ServerName: %v", err)
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != name {
		t.Fatalf("SAN = %q, ServerName = %q", leaf.DNSNames, name)
	}
	if _, err := hostchannel.ServerName("not-a-uuid"); !errors.Is(err, hostchannel.ErrInvalidConfig) {
		t.Fatalf("bad host ServerName = %v", err)
	}
}

func TestNewRequestID(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 100; i++ {
		id, err := hostchannel.NewRequestID()
		if err != nil {
			t.Fatalf("NewRequestID: %v", err)
		}
		if _, err := scalar.ParseUUIDv7(id); err != nil {
			t.Fatalf("request ID %q is not UUIDv7: %v", id, err)
		}
		if id[14] != '7' {
			t.Fatalf("request ID %q has no v7 nibble", id)
		}
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("duplicate request ID %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestLocalFailureMapping(t *testing.T) {
	cases := []struct {
		err  error
		code string
		exit int
	}{
		{hostchannel.ErrInvalidConfig, "invalid_config", 3},
		{hostchannel.ErrAuthenticationFailed, "authentication_failed", 7},
		{hostchannel.ErrTransportFailure, "transport_failure", 8},
		{hostchannel.ErrProtocol, "incompatible_protocol", 6},
		{hostchannel.ErrHelloRequired, "incompatible_protocol", 6},
		{hostchannel.ErrUnframeableInput, "transport_failure", 8},
		{hostchannel.ErrALPNMismatch, "authentication_failed", 7},
		{hosttrust.ErrHostIdentityMismatch, "authentication_failed", 7},
		{hosttrust.ErrNotAllowlisted, "authentication_failed", 7},
		{hosttrust.ErrStaleGeneration, "authentication_failed", 7},
		{hosttrust.ErrAuthorizationRefused, "authentication_failed", 7},
	}
	for _, tc := range cases {
		code, exit, ok := hostchannel.LocalFailure(tc.err)
		if !ok || string(code) != tc.code || exit != tc.exit {
			t.Fatalf("LocalFailure(%v) = %q/%d/%v", tc.err, code, exit, ok)
		}
		registryExit, err := axerror.ExitCodeFor(axerror.Version130, code)
		if err != nil || registryExit != tc.exit {
			t.Fatalf("registry exit for %q = %d, %v", code, registryExit, err)
		}
	}
	if _, _, ok := hostchannel.LocalFailure(errors.New("boom")); ok {
		t.Fatal("unknown error mapped")
	}
}

func TestBindingRedaction(t *testing.T) {
	binding := hostchannel.Binding{LocalHostID: hostA, RemoteHostID: hostB, HelloHostID: hostA}
	for _, rendered := range []string{
		fmt.Sprintf("%v", binding),
		fmt.Sprintf("%#v", binding),
		fmt.Sprintf("%s", binding),
	} {
		if strings.Contains(rendered, hostA) || strings.Contains(rendered, hostB) {
			t.Fatalf("binding leaks identity: %q", rendered)
		}
	}
}

func TestAuthorityListOverflow(t *testing.T) {
	if hostchannel.AuthorityListOverflow(nil) {
		t.Fatal("empty list overflows")
	}
	fitting := [][]byte{make([]byte, 65531)}
	if hostchannel.AuthorityListOverflow(fitting) {
		t.Fatal("65535-byte list overflows")
	}
	exact := [][]byte{make([]byte, 65532)}
	if !hostchannel.AuthorityListOverflow(exact) {
		t.Fatal("65536-byte list admitted")
	}
	many := make([][]byte, 2000)
	for i := range many {
		many[i] = make([]byte, 64)
	}
	if !hostchannel.AuthorityListOverflow(many) {
		t.Fatal("large list admitted")
	}
}

func TestEnrollmentPoolExcludesRevoked(t *testing.T) {
	f := newFixture(t)
	if err := f.storeB.Revoke(scalar.SHA256Digest(f.issuedA.LeafDER).String()); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	snapshot, err := f.storeB.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	pool, err := hostchannel.EnrollmentPool(snapshot, time.Now().UTC())
	if err != nil {
		t.Fatalf("EnrollmentPool: %v", err)
	}
	if pool == nil || len(pool.Subjects()) != 1 {
		t.Fatalf("pool subjects = %d", len(pool.Subjects()))
	}
}

func TestEnrollmentPoolRefusesOverflow(t *testing.T) {
	f := newFixture(t)
	snapshot, err := f.storeB.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	var self hosttrust.CredentialEntry
	for _, entry := range snapshot.Trust.Entries {
		if entry.HostID.String() == hostB {
			self = entry
		}
	}
	bloated := hosttrust.Snapshot{Generation: snapshot.Generation}
	for i := 0; i < 2000; i++ {
		bloated.Trust.Entries = append(bloated.Trust.Entries, self)
	}
	if _, err := hostchannel.EnrollmentPool(bloated, time.Now().UTC()); !errors.Is(err, hostchannel.ErrInvalidConfig) {
		t.Fatalf("overflow pool = %v", err)
	}
}

func TestCredentialFromIssued(t *testing.T) {
	f := newFixture(t)
	cert, err := hostchannel.CredentialFromIssued(f.issuedA)
	if err != nil {
		t.Fatalf("CredentialFromIssued: %v", err)
	}
	if cert.Leaf == nil || len(cert.Certificate) != 1 {
		t.Fatal("credential shape")
	}
	swapped := f.issuedA
	swapped.LeafKeyDER = f.issuedB.LeafKeyDER
	if _, err := hostchannel.CredentialFromIssued(swapped); !errors.Is(err, hostchannel.ErrInvalidConfig) {
		t.Fatalf("mismatched key = %v", err)
	}
}
