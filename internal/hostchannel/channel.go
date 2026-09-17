// Package hostchannel implements AX Section 11.10 Host Channel 1.0.0 over
// Mesh RPC 5.0.0: mutual TLS 1.3 host authentication over an ordered binary
// duplex stream, followed by an RPC-5 hello that is authorized against the
// current hosttrust generation before any dispatch.
//
// The package owns the TLS profile, the hello exchange, per-dispatch and
// per-mutation authorization boundaries, and the specified refusal framing.
// It owns no SSH process, no CLI, no operation semantics beyond admission,
// and no durable state: a channel run performs no file writes. Operation
// bodies stay opaque; the caller-supplied Handler answers post-hello calls.
package hostchannel

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	// ProtocolVersion is the only RPC major this channel frames.
	ProtocolVersion = "5.0.0"
	// ALPN is the exact Section 11.10.1 application protocol identifier.
	ALPN = "ax-host/1"
	// DefaultHandshakeTimeout and DefaultHelloTimeout are the specified
	// fixed 10-second wall-clock bounds. Configuration structs select them
	// with a zero duration; tests use small overrides.
	DefaultHandshakeTimeout = 10 * time.Second
	DefaultHelloTimeout     = 10 * time.Second
	// DefaultRequestTimeout is the Section 6.2 default RPC timeout used
	// when the caller supplies no configured value.
	DefaultRequestTimeout = 300 * time.Second
	// MaxHandshakeBytes caps incoming handshake bytes per side.
	MaxHandshakeBytes = 1024 * 1024
	// MaxAuthorityBytes is the TLS CertificateRequest authority-list bound
	// the encoded CA distinguished-name list must fit.
	MaxAuthorityBytes = 65535
	// HostSuffix is the only permitted DNS SAN parent. It is asserted
	// against issued leaf certificates by test, not trusted by spelling.
	HostSuffix = ".host.ax.invalid"
	// DefaultWatchPoll is the generation-watch sampling interval.
	DefaultWatchPoll = 50 * time.Millisecond
)

var (
	// ErrInvalidConfig reports a local configuration, credential-store or
	// activation failure before launch. The initiator maps it to the
	// invalid_config Error (exit 3).
	ErrInvalidConfig = errors.New("host channel configuration is invalid")
	// ErrAuthenticationFailed reports a known peer certificate or
	// handshake authentication failure. The initiator maps it to the
	// authentication_failed Error (exit 7).
	ErrAuthenticationFailed = errors.New("host channel peer authentication failed")
	// ErrTransportFailure reports transport I/O, a deadline, a byte cap,
	// or an unclassified EOF. The initiator maps it to the
	// transport_failure Error (exit 8).
	ErrTransportFailure = errors.New("host channel transport failed")
	// ErrProtocol reports a contract mismatch the peer actually framed:
	// a structurally invalid hello, a pre-hello non-hello request, or a
	// correlated failure the peer chose. It maps to the
	// incompatible_protocol Error (exit 6) and is never inferred from EOF.
	ErrProtocol = errors.New("host channel protocol mismatch")
	// ErrUnframeableInput reports input that must close without a frame:
	// unparseable bytes, an oversize line, or a foreign RPC major.
	ErrUnframeableInput = errors.New("host channel input is unframeable")
	// ErrHelloRequired reports a valid pre-hello non-hello request. It maps
	// to the incompatible_protocol Error (exit 6) like every framed
	// pre-hello contract refusal, but keeps its own identity so tests can
	// tell the sequencing gate from a malformed hello.
	ErrHelloRequired = errors.New("host channel requires hello first")
	// ErrALPNMismatch reports a negotiated ALPN other than the host
	// channel, including a missing negotiation. The tls.Config selects the
	// advertisement; this check enforces presence, which configuration
	// alone cannot.
	ErrALPNMismatch = errors.New("host channel ALPN mismatch")
	// ErrUnexpectedResume reports a resumed TLS connection, which the
	// channel forbids even when the peer negotiates one.
	ErrUnexpectedResume = errors.New("host channel resumed connection refused")
	// ErrUnverifiedChain reports a handshake that reached the supplement
	// without a standard-verified chain. The supplement never replaces
	// standard chain/name/time/EKU verification.
	ErrUnverifiedChain = errors.New("host channel chain verification missing")
	// ErrNoEnrolledMatch reports a peer certificate with zero or multiple
	// exact enrolled leaf/root/key matches. SAN or CA membership alone
	// never suffices.
	ErrNoEnrolledMatch = errors.New("host channel peer has no unique enrolled match")
	// ErrPeerTrustState reports a uniquely matched entry that is revoked
	// or past its retire_at bound.
	ErrPeerTrustState = errors.New("host channel peer trust state refuses admission")
	// ErrPeerProfile reports a matched entry whose certificates fail the
	// Host Credential Profile 1 checks at handshake time.
	ErrPeerProfile = errors.New("host channel peer credential profile refused")
)

// LocalFailure maps a channel error to the initiator-local Structured Error
// code and exit status of Section 11.10.1. The second result is false when
// the error carries no initiator mapping.
func LocalFailure(err error) (axerror.Code, int, bool) {
	switch {
	case errors.Is(err, ErrInvalidConfig):
		return axerror.Code("invalid_config"), 3, true
	case errors.Is(err, ErrAuthenticationFailed),
		errors.Is(err, ErrALPNMismatch),
		errors.Is(err, hosttrust.ErrHostIdentityMismatch),
		errors.Is(err, hosttrust.ErrNotAllowlisted),
		errors.Is(err, hosttrust.ErrStaleGeneration),
		errors.Is(err, hosttrust.ErrAuthorizationRefused):
		return axerror.Code("authentication_failed"), 7, true
	case errors.Is(err, ErrProtocol),
		errors.Is(err, ErrHelloRequired):
		return axerror.Code("incompatible_protocol"), 6, true
	case errors.Is(err, ErrTransportFailure),
		errors.Is(err, ErrUnframeableInput):
		return axerror.Code("transport_failure"), 8, true
	default:
		return "", 0, false
	}
}

// Stream is the ordered binary stdin/stdout carrier the TLS session runs
// over. It is deliberately narrower than net.Conn: no listener, no PTY, no
// deadlines. Timeouts are enforced by closing the stream.
type Stream interface {
	io.Reader
	io.Writer
	Close() error
}

// ServerName derives the expected UUID-derived DNS SAN of a configured
// destination. The SSH endpoint is never a name here.
func ServerName(hostID string) (string, error) {
	parsed, err := scalar.ParseUUIDv7(hostID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return parsed.String() + HostSuffix, nil
}

// NewRequestID mints one fresh UUIDv7 request correlator.
func NewRequestID() (string, error) {
	var id [16]byte
	binary.BigEndian.PutUint64(id[:8], uint64(time.Now().UnixMilli())<<16)
	if _, err := rand.Read(id[8:]); err != nil {
		return "", fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	id[6] = id[6]&0x0f | 0x70
	id[8] = id[8]&0x3f | 0x80
	var text strings.Builder
	const digits = "0123456789abcdef"
	for i, b := range id {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			text.WriteByte('-')
		}
		text.WriteByte(digits[b>>4])
		text.WriteByte(digits[b&0x0f])
	}
	parsed, err := scalar.ParseUUIDv7(text.String())
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	return parsed.String(), nil
}

// AuthorityListOverflow reports whether the encoded CA distinguished-name
// list of the given subjects exceeds the TLS authority-list bound. The
// encoding is the 2-byte vector length plus one 2-byte length prefix per
// name. Activation refuses an overflowing pool instead of truncating it.
func AuthorityListOverflow(subjects [][]byte) bool {
	total := 2
	for _, subject := range subjects {
		total += 2 + len(subject)
	}
	return total > MaxAuthorityBytes
}

// EnrollmentPool builds the TLS trust pool from exactly the enrolled,
// non-revoked, currently profile-valid roots of the snapshot. The system
// pool is never consulted. Revoked or expired entries are excluded;
// activation refuses when the remaining authority list overflows.
func EnrollmentPool(snapshot hosttrust.Snapshot, now time.Time) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	var subjects [][]byte
	for _, entry := range snapshot.Trust.Entries {
		if entry.State == hosttrust.EntryRevoked {
			continue
		}
		if err := hosttrust.VerifyProfile(entry.LeafDER, entry.RootDER, entry.HostID.String(), now); err != nil {
			continue
		}
		root, err := x509.ParseCertificate(entry.RootDER)
		if err != nil {
			continue
		}
		pool.AddCert(root)
		subjects = append(subjects, root.RawSubject)
	}
	if AuthorityListOverflow(subjects) {
		return nil, fmt.Errorf("%w: enrolled authority list exceeds the TLS bound", ErrInvalidConfig)
	}
	return pool, nil
}

// VerifiedPeer is the uniquely verified enrolled identity of a TLS peer.
// DER bytes stay unexported so status and error paths cannot log them.
type VerifiedPeer struct {
	HostID       string
	CredentialID string
	leafDER      []byte
	rootDER      []byte
}

// VerifyFunc supplements a successful standard TLS verification with the
// enrolled-identity checks. It never replaces standard verification.
type VerifyFunc func(tls.ConnectionState) error

// VerifyPeer returns the supplement plus the record it fills on success.
// The TLS version is enforced once, by the tls.Config both builders share
// below. The supplement requires the exact negotiated ALPN (configuration
// alone cannot refuse a missing negotiation), no resumption, a
// standard-verified chain, exactly one exact enrolled leaf/root/key match,
// current profile validity, and an admitting trust state. UUID/SAN binding
// comes from the profile check against the matched entry.
func VerifyPeer(snapshot hosttrust.Snapshot, now time.Time) (VerifyFunc, *VerifiedPeer) {
	verified := &VerifiedPeer{}
	verify := func(state tls.ConnectionState) error {
		if state.NegotiatedProtocol != ALPN {
			return ErrALPNMismatch
		}
		if state.DidResume {
			return ErrUnexpectedResume
		}
		if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 {
			return ErrUnverifiedChain
		}
		leaf := state.PeerCertificates[0]
		chain := state.VerifiedChains[0]
		root := chain[len(chain)-1]
		spki := scalar.SHA256Digest(leaf.RawSubjectPublicKeyInfo).String()
		var matched []hosttrust.CredentialEntry
		for _, entry := range snapshot.Trust.Entries {
			if !bytes.Equal(entry.LeafDER, leaf.Raw) {
				continue
			}
			if !bytes.Equal(entry.RootDER, root.Raw) {
				continue
			}
			if entry.SPKIID.String() != spki {
				continue
			}
			matched = append(matched, entry)
		}
		if len(matched) != 1 {
			return ErrNoEnrolledMatch
		}
		entry := matched[0]
		if err := hosttrust.VerifyProfile(entry.LeafDER, entry.RootDER, entry.HostID.String(), now); err != nil {
			return fmt.Errorf("%w: %w", ErrPeerProfile, err)
		}
		switch entry.State {
		case hosttrust.EntryActive:
		case hosttrust.EntryRetiring:
			if entry.RetireAt == nil {
				return ErrPeerTrustState
			}
			retireAt, err := entry.RetireAt.Time()
			if err != nil || !now.Before(retireAt) {
				return ErrPeerTrustState
			}
		default:
			return ErrPeerTrustState
		}
		verified.HostID = entry.HostID.String()
		verified.CredentialID = entry.CredentialID.String()
		verified.leafDER = bytes.Clone(leaf.Raw)
		verified.rootDER = bytes.Clone(root.Raw)
		return nil
	}
	return verify, verified
}

// baseTLS carries the single shared enforcement of the TLS version floor
// and ceiling, the exact ALPN, and mandatory verification. Both roles
// build on it so no role can drift from the profile.
func baseTLS() *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		NextProtos:         alpnProtocols(),
		InsecureSkipVerify: false,
	}
}

func alpnProtocols() []string { return []string{ALPN} }

// ClientTLS builds the initiator TLS 1.3 client configuration: enrolled
// roots only, no session cache, the exact ALPN, and the destination SAN as
// server name. The supplement runs inside VerifyConnection.
func ClientTLS(certificate tls.Certificate, pool *x509.CertPool, serverName string, verify VerifyFunc) *tls.Config {
	config := baseTLS()
	config.Certificates = []tls.Certificate{certificate}
	config.RootCAs = pool
	config.ServerName = serverName
	config.ClientSessionCache = nil
	config.VerifyConnection = verify
	return config
}

// ServerTLS builds the responder TLS 1.3 server configuration: enrolled
// roots only, mandatory verified client certificates, no session tickets,
// the exact ALPN, and the supplement inside VerifyConnection.
func ServerTLS(certificate tls.Certificate, pool *x509.CertPool, verify VerifyFunc) *tls.Config {
	config := baseTLS()
	config.Certificates = []tls.Certificate{certificate}
	config.ClientAuth = tls.RequireAndVerifyClientCert
	config.ClientCAs = pool
	config.SessionTicketsDisabled = true
	config.VerifyConnection = verify
	return config
}

// HandshakeLimiter caps incoming handshake bytes while armed. The channel
// disarms it immediately after local handshake success, so application
// records never count against the handshake budget.
type HandshakeLimiter struct {
	reader io.Reader
	count  int64
	armed  atomic.Bool
	exceed atomic.Bool
	limit  int64
}

// NewHandshakeLimiter arms a 1 MiB incoming cap over the handshake stream.
func NewHandshakeLimiter(reader io.Reader) *HandshakeLimiter {
	limiter := &HandshakeLimiter{reader: reader, limit: MaxHandshakeBytes}
	limiter.armed.Store(true)
	return limiter
}

// Disarm ends handshake byte accounting after local handshake success.
func (limiter *HandshakeLimiter) Disarm() { limiter.armed.Store(false) }

// Exceeded reports whether the cap fired.
func (limiter *HandshakeLimiter) Exceeded() bool { return limiter.exceed.Load() }

func (limiter *HandshakeLimiter) Read(data []byte) (int, error) {
	n, err := limiter.reader.Read(data)
	if !limiter.armed.Load() {
		return n, err
	}
	limiter.count += int64(n)
	if limiter.count > limiter.limit {
		limiter.exceed.Store(true)
		return n, fmt.Errorf("%w: handshake byte cap exceeded", ErrTransportFailure)
	}
	return n, err
}

// streamConn adapts a Stream to the net.Conn shape crypto/tls requires.
// Addresses are static pipe labels; deadlines are unsupported because the
// channel enforces its bounds by closing the stream.
type streamConn struct {
	Stream
}

type pipeAddr string

func (addr pipeAddr) Network() string { return "pipe" }
func (addr pipeAddr) String() string  { return string(addr) }

func (conn streamConn) LocalAddr() net.Addr  { return pipeAddr("local") }
func (conn streamConn) RemoteAddr() net.Addr { return pipeAddr("remote") }
func (conn streamConn) SetDeadline(_ time.Time) error {
	return errors.New("host channel deadlines close the stream")
}
func (conn streamConn) SetReadDeadline(_ time.Time) error {
	return errors.New("host channel deadlines close the stream")
}
func (conn streamConn) SetWriteDeadline(_ time.Time) error {
	return errors.New("host channel deadlines close the stream")
}

// RunHandshake completes the TLS handshake under a wall-clock deadline,
// enforced by closing the stream even when the peer sends nothing. A fired
// deadline, an exceeded byte cap, or a transport read failure maps to
// ErrTransportFailure; every other handshake failure is peer
// authentication. No application bytes may move before this returns nil.
func RunHandshake(conn *tls.Conn, stream Stream, limiter *HandshakeLimiter, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- conn.Handshake() }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		if err == nil {
			return nil
		}
		if limiter != nil && limiter.Exceeded() {
			return fmt.Errorf("%w: %w", ErrTransportFailure, err)
		}
		return classifyHandshakeError(err)
	case <-timer.C:
		_ = stream.Close()
		err := <-done
		if limiter != nil && limiter.Exceeded() {
			return fmt.Errorf("%w: handshake deadline exceeded", ErrTransportFailure)
		}
		if err == nil {
			return fmt.Errorf("%w: handshake deadline exceeded", ErrTransportFailure)
		}
		return fmt.Errorf("%w: handshake deadline exceeded: %v", ErrTransportFailure, classifyHandshakeError(err))
	}
}

func classifyHandshakeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTransportFailure) {
		return err
	}
	// A received TLS alert unwraps to an OpError whose Op is "remote
	// error": that is peer authentication, never transport I/O. Every
	// other OpError, a timeout, a closed conn, or an EOF variant is the
	// transport failing underneath the handshake.
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op != "remote error" {
		return fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	return fmt.Errorf("%w: %w", ErrAuthenticationFailed, err)
}

// WriteLine sends one LF-terminated frame. The payload must already be a
// single LF-free line within the wire bound; the check here is defensive.
func WriteLine(writer io.Writer, line []byte) error {
	if len(line) == 0 || len(line) > rpcwire.MaxLineBytes || bytes.IndexByte(line, '\n') >= 0 {
		return ErrUnframeableInput
	}
	frame := make([]byte, 0, len(line)+1)
	frame = append(frame, line...)
	frame = append(frame, '\n')
	if _, err := writer.Write(frame); err != nil {
		return fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	return nil
}

// FrameReader decodes successive LF-terminated frames from one stream. It
// must be the only reader on its stream: buffering may hold bytes past
// the current frame. One FrameReader serves a whole connection.
type FrameReader struct {
	buffered *bufio.Reader
}

// NewFrameReader buffers one stream for successive frame reads.
func NewFrameReader(reader io.Reader) *FrameReader {
	return &FrameReader{buffered: bufio.NewReader(reader)}
}

// ReadLine reads one LF-terminated frame without the terminator. An
// over-bound line or mid-line truncation is unframeable and must close
// without a frame. A stream that ends with no frame bytes surfaces as
// io.EOF for the transport mapping, including an abrupt end: zero bytes
// is never a truncated frame.
func (frames *FrameReader) ReadLine() ([]byte, error) {
	var line []byte
	for {
		chunk, err := frames.buffered.ReadSlice('\n')
		if len(chunk) > 0 {
			if len(line)+len(chunk) > rpcwire.MaxLineBytes+1 {
				return nil, ErrUnframeableInput
			}
			line = append(line, chunk...)
		}
		switch {
		case err == nil:
			return line[:len(line)-1], nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case len(line) == 0 && (errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrClosedPipe)):
			return nil, io.EOF
		default:
			return nil, ErrUnframeableInput
		}
	}
}

// ReadLine reads a single frame from a stream no other reader shares. It
// is a one-shot convenience for tests and single-frame peers; a
// connection that reads more than one frame must use NewFrameReader.
func ReadLine(reader io.Reader) ([]byte, error) {
	return NewFrameReader(reader).ReadLine()
}

// writeLineBounded sends one frame under a wall-clock deadline enforced by
// closing the stream. A peer that never reads cannot stall the exchange
// past its bound. A fired deadline maps to ErrTransportFailure.
func writeLineBounded(writer io.Writer, stream Stream, line []byte, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- WriteLine(writer, line) }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		_ = stream.Close()
		<-done
		return fmt.Errorf("%w: frame deadline exceeded", ErrTransportFailure)
	}
}

// readLineBounded reads one frame under a wall-clock deadline enforced by
// closing the stream. A fired deadline maps to ErrTransportFailure.
func readLineBounded(frames *FrameReader, stream Stream, timeout time.Duration) ([]byte, error) {
	type result struct {
		line []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		line, err := frames.ReadLine()
		done <- result{line: line, err: err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case outcome := <-done:
		if errors.Is(outcome.err, io.EOF) {
			return nil, fmt.Errorf("%w: peer closed the stream", ErrTransportFailure)
		}
		return outcome.line, outcome.err
	case <-timer.C:
		_ = stream.Close()
		<-done
		return nil, fmt.Errorf("%w: frame deadline exceeded", ErrTransportFailure)
	}
}

// Binding ties an admitted stream to its authorization facts: the snapshot
// generation it was bound under, the local credential, and the verified
// remote credential with its authenticated hello identity.
type Binding struct {
	Generation         uint64
	LocalHostID        string
	LocalCredentialID  string
	RemoteHostID       string
	RemoteCredentialID string
	HelloHostID        string
	remoteLeafDER      []byte
	remoteRootDER      []byte
}

// Formatting a binding never exposes identities, digests, or key material.
func (Binding) String() string   { return "host channel binding" }
func (Binding) GoString() string { return "host channel binding" }
func (Binding) Format(state fmt.State, verb rune) {
	_, _ = state.Write([]byte("host channel binding"))
}

func (binding Binding) authRequest(allowlisted []string, expected string, now time.Time) hosttrust.AuthRequest {
	return hosttrust.AuthRequest{
		SnapshotGeneration:   binding.Generation,
		LocalHostID:          binding.LocalHostID,
		LocalCredentialID:    binding.LocalCredentialID,
		RemoteLeafDER:        bytes.Clone(binding.remoteLeafDER),
		RemoteRootDER:        bytes.Clone(binding.remoteRootDER),
		ExpectedRemoteHostID: expected,
		Allowlisted:          append([]string(nil), allowlisted...),
		HelloHostID:          binding.HelloHostID,
		Now:                  now,
	}
}

// CredentialFromIssued builds the TLS leaf credential from issued material:
// the leaf DER plus its PKCS#8 private key. The root private key never
// exists here; issuance destroyed it.
func CredentialFromIssued(issued hosttrust.IssuedCredential) (tls.Certificate, error) {
	key, err := x509.ParsePKCS8PrivateKey(issued.LeafKeyDER)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	leaf, err := x509.ParseCertificate(issued.LeafDER)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	switch key := key.(type) {
	case *ecdsa.PrivateKey:
		if !key.PublicKey.Equal(leaf.PublicKey) {
			return tls.Certificate{}, fmt.Errorf("%w: leaf key mismatch", ErrInvalidConfig)
		}
	default:
		return tls.Certificate{}, fmt.Errorf("%w: unexpected leaf key type", ErrInvalidConfig)
	}
	return tls.Certificate{Certificate: [][]byte{bytes.Clone(issued.LeafDER)}, PrivateKey: key, Leaf: leaf}, nil
}

// WatchGeneration blocks until the committed generation moves past the
// bound one, the snapshot becomes unreadable, or the context ends. A moved
// generation returns hosttrust.ErrStaleGeneration: the caller must close
// the bound stream. Filesystem notifications alone never satisfy this
// watch; only committed reads do. A nil error is never returned: context
// expiry surfaces as the context error.
func WatchGeneration(ctx context.Context, store *hosttrust.Store, bound uint64, poll time.Duration) error {
	if ctx == nil || store == nil {
		return ErrInvalidConfig
	}
	if poll <= 0 {
		poll = DefaultWatchPoll
	}
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			snapshot, err := store.ReadSnapshot()
			if err != nil {
				return fmt.Errorf("%w: generation read failed: %v", ErrTransportFailure, err)
			}
			if snapshot.Generation != bound {
				return fmt.Errorf("host channel generation moved: %w", hosttrust.ErrStaleGeneration)
			}
		}
	}
}
