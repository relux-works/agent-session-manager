package hostchannel

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
)

// ClientConfig activates one initiator stream. ExpectedRemoteHostID is the
// selected configured destination; the TLS server name and the hello
// identity must both equal it. LocalHostID, LocalCredentialID, and
// Allowlisted come from Configuration 4 like the server side. Zero
// HandshakeTimeout, HelloTimeout, and RequestTimeout select the specified
// defaults; nonzero values override them for tests. A nil Now selects the
// UTC clock.
type ClientConfig struct {
	Store                *hosttrust.Store
	Credential           tls.Certificate
	LocalHostID          string
	LocalCredentialID    string
	ExpectedRemoteHostID string
	Allowlisted          []string
	Platform             string
	AXVersion            string
	HandshakeTimeout     time.Duration
	HelloTimeout         time.Duration
	RequestTimeout       time.Duration
	Now                  func() time.Time
}

func (config ClientConfig) clock() func() time.Time {
	if config.Now != nil {
		return config.Now
	}
	return func() time.Time { return time.Now().UTC() }
}

func (config ClientConfig) handshakeTimeout() time.Duration {
	if config.HandshakeTimeout > 0 {
		return config.HandshakeTimeout
	}
	return DefaultHandshakeTimeout
}

func (config ClientConfig) helloTimeout() time.Duration {
	if config.HelloTimeout > 0 {
		return config.HelloTimeout
	}
	return DefaultHelloTimeout
}

func (config ClientConfig) requestTimeout() time.Duration {
	if config.RequestTimeout > 0 {
		return config.RequestTimeout
	}
	return DefaultRequestTimeout
}

// Client is one mutually authenticated initiator stream past hello
// success. Calls recheck the current authorization generation before
// sending; a moved generation refuses without transmitting.
type Client struct {
	conn     *tls.Conn
	stream   Stream
	frames   *FrameReader
	store    *hosttrust.Store
	binding  Binding
	allow    []string
	expected string
	timeout  time.Duration
	now      func() time.Time
}

// Dial opens the initiator side over one stream: mutual TLS 1.3 against
// the destination SAN, one hello, and current-generation authorization.
// The hello success is validated before Dial returns; no other operation
// moves first.
func Dial(stream Stream, config ClientConfig) (*Client, error) {
	if stream == nil || config.Store == nil {
		_ = closeQuietly(stream)
		return nil, ErrInvalidConfig
	}
	serverName, err := ServerName(config.ExpectedRemoteHostID)
	if err != nil {
		_ = closeQuietly(stream)
		return nil, err
	}
	now := config.clock()
	current := now()
	snapshot, err := config.Store.ReadSnapshot()
	if err != nil {
		_ = closeQuietly(stream)
		return nil, fmt.Errorf("%w: trust snapshot unreadable: %v", ErrInvalidConfig, err)
	}
	if err := checkLocalCredential(snapshot, config.Credential, config.LocalHostID, config.LocalCredentialID, current); err != nil {
		_ = closeQuietly(stream)
		return nil, err
	}
	pool, err := EnrollmentPool(snapshot, current)
	if err != nil {
		_ = closeQuietly(stream)
		return nil, err
	}
	if err := validateHelloTemplate(config.LocalHostID, config.Platform, config.AXVersion); err != nil {
		_ = closeQuietly(stream)
		return nil, err
	}
	verify, verified := VerifyPeer(snapshot, current)
	limiter := NewHandshakeLimiter(stream)
	conn := tls.Client(streamConn{Stream: cappedStream{Stream: stream, limiter: limiter}}, ClientTLS(config.Credential, pool, serverName, verify))
	if err := RunHandshake(conn, stream, limiter, config.handshakeTimeout()); err != nil {
		_ = stream.Close()
		return nil, err
	}
	limiter.Disarm()

	body, err := helloBody(config.LocalHostID, config.Platform, config.AXVersion, rpcwire.NewNonce(), "")
	if err != nil {
		_ = stream.Close()
		return nil, fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	requestID, err := NewRequestID()
	if err != nil {
		_ = stream.Close()
		return nil, err
	}
	line, err := rpcwire.EncodeRequest(ProtocolVersion, requestID, "hello", body)
	if err != nil {
		_ = stream.Close()
		return nil, fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		_ = stream.Close()
		return nil, fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	frames := NewFrameReader(conn)
	if err := writeLineBounded(conn, stream, line, config.helloTimeout()); err != nil {
		_ = stream.Close()
		return nil, err
	}
	responseLine, err := readLineBounded(frames, stream, config.helloTimeout())
	if err != nil {
		_ = stream.Close()
		return nil, err
	}
	response, err := rpcwire.DecodeResponse(responseLine, request)
	if err != nil {
		_ = stream.Close()
		if errors.Is(err, rpcwire.ErrVersion) || errors.Is(err, rpcwire.ErrFrame) {
			return nil, fmt.Errorf("%w: peer hello reply is unframeable: %v", ErrTransportFailure, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrProtocol, err)
	}
	if !response.OK() {
		_ = stream.Close()
		return nil, helloRefusal(response.Failure())
	}
	hello, err := response.Hello()
	if err != nil {
		_ = stream.Close()
		return nil, fmt.Errorf("%w: %w", ErrProtocol, err)
	}
	fresh, err := config.Store.ReadSnapshot()
	if err != nil {
		_ = stream.Close()
		return nil, fmt.Errorf("%w: generation read failed: %v", ErrTransportFailure, err)
	}
	if fresh.Generation != snapshot.Generation {
		_ = stream.Close()
		return nil, fmt.Errorf("authorization generation moved during hello: %w", hosttrust.ErrStaleGeneration)
	}
	binding := Binding{
		Generation:         fresh.Generation,
		LocalHostID:        config.LocalHostID,
		LocalCredentialID:  config.LocalCredentialID,
		RemoteHostID:       verified.HostID,
		RemoteCredentialID: verified.CredentialID,
		HelloHostID:        hello.HostID,
		remoteLeafDER:      verified.leafDER,
		remoteRootDER:      verified.rootDER,
	}
	if err := config.Store.AuthorizeDispatch(fresh, binding.authRequest(config.Allowlisted, config.ExpectedRemoteHostID, now())); err != nil {
		_ = stream.Close()
		return nil, fmt.Errorf("%w: %w", ErrAuthenticationFailed, err)
	}
	return &Client{
		conn:     conn,
		stream:   stream,
		frames:   frames,
		store:    config.Store,
		binding:  binding,
		allow:    append([]string(nil), config.Allowlisted...),
		expected: config.ExpectedRemoteHostID,
		timeout:  config.requestTimeout(),
		now:      now,
	}, nil
}

// helloRefusal maps a correlated hello failure frame to the initiator
// error. Identity and allowlist refusals are authentication failures; a
// contract refusal is a protocol mismatch.
func helloRefusal(failure *axerror.Error) error {
	switch failure.Code() {
	case axerror.Code("host_identity_mismatch"):
		return fmt.Errorf("%w: peer refused hello identity", hosttrust.ErrHostIdentityMismatch)
	case axerror.Code("peer_not_allowlisted"):
		return fmt.Errorf("%w: peer is not allowlisted", hosttrust.ErrNotAllowlisted)
	default:
		return fmt.Errorf("%w: peer refused hello", ErrProtocol)
	}
}

func closeQuietly(stream Stream) error {
	if stream == nil {
		return nil
	}
	return stream.Close()
}

// Binding reports the stream's authorization binding.
func (client *Client) Binding() Binding { return client.binding }

// PeerHostID reports the verified enrolled UUID of the responder.
func (client *Client) PeerHostID() string { return client.binding.RemoteHostID }

// ConnectionState exposes the negotiated TLS parameters for resumption
// and version assertions.
func (client *Client) ConnectionState() tls.ConnectionState { return client.conn.ConnectionState() }

// Close ends the stream.
func (client *Client) Close() error { return client.stream.Close() }

// Call sends one post-hello request and returns the correlated response.
// The current generation is rechecked before sending; a moved generation
// refuses without transmitting. Framed failure responses return with a nil
// error so the caller inspects the exact Error class and exit status. A
// failed Call closes the stream: the exchange cannot resume mid-frame.
func (client *Client) Call(operation string, body json.RawMessage) (rpcwire.Response, error) {
	fresh, err := client.store.ReadSnapshot()
	if err != nil {
		return rpcwire.Response{}, fmt.Errorf("%w: generation read failed: %v", ErrTransportFailure, err)
	}
	if fresh.Generation != client.binding.Generation {
		return rpcwire.Response{}, fmt.Errorf("authorization generation moved: %w", hosttrust.ErrStaleGeneration)
	}
	requestID, err := NewRequestID()
	if err != nil {
		return rpcwire.Response{}, err
	}
	line, err := rpcwire.EncodeRequest(ProtocolVersion, requestID, operation, body)
	if err != nil {
		return rpcwire.Response{}, err
	}
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		return rpcwire.Response{}, err
	}
	if err := writeLineBounded(client.conn, client.stream, line, client.timeout); err != nil {
		_ = client.stream.Close()
		return rpcwire.Response{}, err
	}
	responseLine, err := readLineBounded(client.frames, client.stream, client.timeout)
	if err != nil {
		_ = client.stream.Close()
		return rpcwire.Response{}, err
	}
	response, err := rpcwire.DecodeResponse(responseLine, request)
	if err != nil {
		if errors.Is(err, rpcwire.ErrVersion) || errors.Is(err, rpcwire.ErrFrame) {
			return rpcwire.Response{}, fmt.Errorf("%w: peer reply is unframeable: %v", ErrTransportFailure, err)
		}
		return rpcwire.Response{}, fmt.Errorf("%w: %w", ErrProtocol, err)
	}
	return response, nil
}
