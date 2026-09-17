package hostchannel

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Handler answers one authorized post-hello call. The binding carries the
// authenticated peer and hello identities the dispatch was admitted under.
// The handler returns the success body, or an error that the server frames
// as an incompatible_protocol failure and continues from:
// operation-specific failures belong to the operation owners, and this
// channel mints no new Error enum. A boundary reports a side effect
// through mutate, which rechecks the current authorization generation
// under the exclusive lock before running it; a stale generation closes
// the stream without a frame instead.
type Handler func(ctx context.Context, request rpcwire.Request, binding Binding, mutate func(boundary func() error) error) (json.RawMessage, error)

// ServerConfig activates one responder stream. LocalHostID and
// LocalCredentialID come from Configuration 4 (host ID and
// mesh.host_channel credential); Allowlisted is the configured mesh peer
// set. Zero HandshakeTimeout, HelloTimeout, and RequestTimeout select the
// specified defaults; nonzero values override them for tests. A nil Now
// selects the UTC clock.
type ServerConfig struct {
	Store             *hosttrust.Store
	Credential        tls.Certificate
	LocalHostID       string
	LocalCredentialID string
	Allowlisted       []string
	Platform          string
	AXVersion         string
	Handler           Handler
	HandshakeTimeout  time.Duration
	HelloTimeout      time.Duration
	RequestTimeout    time.Duration
	Now               func() time.Time
}

func (config ServerConfig) clock() func() time.Time {
	if config.Now != nil {
		return config.Now
	}
	return func() time.Time { return time.Now().UTC() }
}

func (config ServerConfig) handshakeTimeout() time.Duration {
	if config.HandshakeTimeout > 0 {
		return config.HandshakeTimeout
	}
	return DefaultHandshakeTimeout
}

func (config ServerConfig) helloTimeout() time.Duration {
	if config.HelloTimeout > 0 {
		return config.HelloTimeout
	}
	return DefaultHelloTimeout
}

func (config ServerConfig) requestTimeout() time.Duration {
	if config.RequestTimeout > 0 {
		return config.RequestTimeout
	}
	return DefaultRequestTimeout
}

// checkLocalCredential validates the activation identity: the leaf parses,
// its SAN is the local host, its digest is the configured credential, and
// the snapshot carries it as an active profile-valid entry.
func checkLocalCredential(snapshot hosttrust.Snapshot, certificate tls.Certificate, hostID, credentialID string, now time.Time) error {
	if len(certificate.Certificate) == 0 {
		return fmt.Errorf("%w: local credential carries no leaf", ErrInvalidConfig)
	}
	leaf, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	parsed, err := scalar.ParseUUIDv7(hostID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != parsed.String()+HostSuffix {
		return fmt.Errorf("%w: local leaf SAN differs from the configured host", ErrInvalidConfig)
	}
	if scalar.SHA256Digest(certificate.Certificate[0]).String() != credentialID {
		return fmt.Errorf("%w: local leaf digest differs from the configured credential", ErrInvalidConfig)
	}
	for _, entry := range snapshot.Trust.Entries {
		if entry.CredentialID.String() != credentialID {
			continue
		}
		if entry.HostID.String() != parsed.String() || entry.State != hosttrust.EntryActive {
			return fmt.Errorf("%w: local credential entry is not active", ErrInvalidConfig)
		}
		if err := hosttrust.VerifyProfile(entry.LeafDER, entry.RootDER, parsed.String(), now); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
		}
		return nil
	}
	return fmt.Errorf("%w: local credential is not enrolled", ErrInvalidConfig)
}

// helloBody builds one local hello map: the configured host ID, fresh
// nonce material supplied by the caller, and the exact RPC-5 contract
// profile at the specified limit floors. No authentication, trust,
// configuration, error, or transport key is added.
func helloBody(hostID, platform, axVersion, nonce, nonceEcho string) (json.RawMessage, error) {
	contracts, err := rpcwire.ContractProfile(ProtocolVersion)
	if err != nil {
		return nil, err
	}
	lineBytes, err := scalar.NewUint53(rpcwire.MaxLineBytes)
	if err != nil {
		return nil, err
	}
	objectBytes, err := scalar.NewUint53(rpcwire.MinObjectBytes)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(rpcwire.Hello{
		HostID:         hostID,
		Platform:       platform,
		AXVersion:      axVersion,
		Nonce:          nonce,
		NonceEcho:      nonceEcho,
		Contracts:      contracts,
		MaxLineBytes:   lineBytes,
		MaxObjectBytes: objectBytes,
	})
	if err != nil {
		return nil, err
	}
	return body, nil
}

// validateHelloTemplate proves the configured platform and version shape a
// structurally valid RPC-5 hello before the stream opens, through the same
// encoder the handshake uses.
func validateHelloTemplate(hostID, platform, axVersion string) error {
	body, err := helloBody(hostID, platform, axVersion, rpcwire.NewNonce(), "")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	if _, err := rpcwire.EncodeRequest(ProtocolVersion, hostID, "hello", body); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return nil
}

func failure13(code axerror.Code, message string, cause error) (*axerror.Error, error) {
	return axerror.New(axerror.Spec{
		Version: axerror.Version130,
		Code:    code,
		Message: message,
		IDs:     axerror.NoIDs(),
		Details: axerror.Details{},
		Cause:   cause,
	})
}

// Serve runs the responder core over one stream: mutual TLS 1.3, one
// authorized hello, then the dispatch loop. It returns when the stream
// closes. Pre-hello refusals send at most one encrypted Error 1.3.0
// failure and close; unframeable input and foreign majors close without a
// frame. Before TLS completes it sends nothing but standard TLS alerts.
func Serve(stream Stream, config ServerConfig) error {
	if stream == nil || config.Store == nil || config.Handler == nil {
		return ErrInvalidConfig
	}
	now := config.clock()
	current := now()
	snapshot, err := config.Store.ReadSnapshot()
	if err != nil {
		return fmt.Errorf("%w: trust snapshot unreadable: %v", ErrInvalidConfig, err)
	}
	if err := checkLocalCredential(snapshot, config.Credential, config.LocalHostID, config.LocalCredentialID, current); err != nil {
		return err
	}
	pool, err := EnrollmentPool(snapshot, current)
	if err != nil {
		return err
	}
	if err := validateHelloTemplate(config.LocalHostID, config.Platform, config.AXVersion); err != nil {
		return err
	}
	verify, verified := VerifyPeer(snapshot, current)
	limiter := NewHandshakeLimiter(stream)
	conn := tls.Server(streamConn{Stream: cappedStream{Stream: stream, limiter: limiter}}, ServerTLS(config.Credential, pool, verify))
	if err := RunHandshake(conn, stream, limiter, config.handshakeTimeout()); err != nil {
		_ = stream.Close()
		return err
	}
	limiter.Disarm()
	bound := snapshot.Generation
	peerLeaf, peerRoot := verified.leafDER, verified.rootDER
	peerHost, peerCredential := verified.HostID, verified.CredentialID
	frames := NewFrameReader(conn)

	line, err := readLineBounded(frames, stream, config.helloTimeout())
	if err != nil {
		_ = stream.Close()
		return err
	}
	request, err := rpcwire.DecodeRequest(line)
	if err != nil {
		return refuseHello(stream, conn, line, err, config.helloTimeout())
	}
	if request.Version() != ProtocolVersion {
		_ = stream.Close()
		return fmt.Errorf("%w: foreign RPC major on the channel", ErrUnframeableInput)
	}
	if request.Operation() != "hello" {
		return sendHelloFailure(stream, conn, request, line, axerror.Code("incompatible_protocol"), ErrHelloRequired, config.helloTimeout())
	}
	hello, err := request.Hello()
	if err != nil {
		return refuseHello(stream, conn, line, err, config.helloTimeout())
	}
	fresh, err := config.Store.ReadSnapshot()
	if err != nil {
		_ = stream.Close()
		return fmt.Errorf("%w: generation read failed: %v", ErrTransportFailure, err)
	}
	if fresh.Generation != bound {
		_ = stream.Close()
		return fmt.Errorf("authorization generation moved during hello: %w", hosttrust.ErrStaleGeneration)
	}
	binding := Binding{
		Generation:         fresh.Generation,
		LocalHostID:        config.LocalHostID,
		LocalCredentialID:  config.LocalCredentialID,
		RemoteHostID:       peerHost,
		RemoteCredentialID: peerCredential,
		HelloHostID:        hello.HostID,
		remoteLeafDER:      peerLeaf,
		remoteRootDER:      peerRoot,
	}
	if err := config.Store.AuthorizeDispatch(fresh, binding.authRequest(config.Allowlisted, peerHost, now())); err != nil {
		switch {
		case errors.Is(err, hosttrust.ErrHostIdentityMismatch):
			return sendHelloFailure(stream, conn, request, line, axerror.Code("host_identity_mismatch"), err, config.helloTimeout())
		case errors.Is(err, hosttrust.ErrNotAllowlisted):
			return sendHelloFailure(stream, conn, request, line, axerror.Code("peer_not_allowlisted"), err, config.helloTimeout())
		default:
			_ = stream.Close()
			return err
		}
	}
	responseBody, err := helloBody(config.LocalHostID, config.Platform, config.AXVersion, rpcwire.NewNonce(), hello.Nonce)
	if err != nil {
		_ = stream.Close()
		return fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	success, err := rpcwire.EncodeSuccess(request, responseBody)
	if err != nil {
		_ = stream.Close()
		return fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	if err := writeLineBounded(conn, stream, success, config.helloTimeout()); err != nil {
		_ = stream.Close()
		return err
	}
	return serveLoop(frames, stream, conn, config, binding)
}

// refuseHello frames one Error 1.3.0 failure for a structurally invalid
// RPC-5 hello and closes. Unframeable input closes without a frame.
func refuseHello(stream Stream, conn *tls.Conn, line []byte, decodeErr error, timeout time.Duration) error {
	defer func() { _ = stream.Close() }()
	if errors.Is(decodeErr, rpcwire.ErrVersion) || errors.Is(decodeErr, rpcwire.ErrFrame) {
		return fmt.Errorf("%w: %w", ErrUnframeableInput, decodeErr)
	}
	failure, err := failure13("incompatible_protocol", "host channel hello refused", decodeErr)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	frame, err := rpcwire.EncodeRejection(line, failure)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnframeableInput, err)
	}
	if err := writeLineBounded(conn, stream, frame, timeout); err != nil {
		return err
	}
	return fmt.Errorf("%w: %w", ErrProtocol, decodeErr)
}

// sendHelloFailure sends at most one encrypted hello refusal and closes.
func sendHelloFailure(stream Stream, conn *tls.Conn, request rpcwire.Request, line []byte, code axerror.Code, cause error, timeout time.Duration) error {
	defer func() { _ = stream.Close() }()
	failure, err := failure13(code, "host channel hello refused", cause)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransportFailure, err)
	}
	frame, err := rpcwire.EncodeRejection(line, failure)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnframeableInput, err)
	}
	if err := writeLineBounded(conn, stream, frame, timeout); err != nil {
		return err
	}
	if code == axerror.Code("incompatible_protocol") && !errors.Is(cause, ErrHelloRequired) {
		return fmt.Errorf("%w: %w", ErrProtocol, cause)
	}
	return cause
}

// serveLoop dispatches authorized post-hello calls until the stream ends.
// Every dispatch rechecks the current generation under the authorization
// lock; a moved generation or failed check closes without a frame.
func serveLoop(frames *FrameReader, stream Stream, conn *tls.Conn, config ServerConfig, binding Binding) error {
	now := config.clock()
	for {
		line, err := readLineBounded(frames, stream, config.requestTimeout())
		if err != nil {
			_ = stream.Close()
			return err
		}
		request, err := rpcwire.DecodeRequest(line)
		if err != nil {
			_ = stream.Close()
			return fmt.Errorf("%w: %w", ErrUnframeableInput, err)
		}
		if request.Version() != ProtocolVersion {
			_ = stream.Close()
			return fmt.Errorf("%w: foreign RPC major on the channel", ErrUnframeableInput)
		}
		fresh, err := config.Store.ReadSnapshot()
		if err != nil {
			_ = stream.Close()
			return fmt.Errorf("%w: generation read failed: %v", ErrTransportFailure, err)
		}
		if fresh.Generation != binding.Generation {
			_ = stream.Close()
			return fmt.Errorf("authorization generation moved: %w", hosttrust.ErrStaleGeneration)
		}
		if err := config.Store.AuthorizeDispatch(fresh, binding.authRequest(config.Allowlisted, binding.RemoteHostID, now())); err != nil {
			_ = stream.Close()
			return err
		}
		mutate := func(boundary func() error) error {
			return config.Store.WithMutationAuthorization(binding.authRequest(config.Allowlisted, binding.RemoteHostID, now()), boundary)
		}
		body, err := config.Handler(context.Background(), request, binding, mutate)
		if err != nil {
			if errors.Is(err, hosttrust.ErrStaleGeneration) {
				_ = stream.Close()
				return err
			}
			failure, buildErr := failure13("incompatible_protocol", "host channel operation refused", err)
			if buildErr != nil {
				_ = stream.Close()
				return fmt.Errorf("%w: %w", ErrTransportFailure, buildErr)
			}
			frame, buildErr := rpcwire.EncodeFailure(request, failure)
			if buildErr != nil {
				_ = stream.Close()
				return fmt.Errorf("%w: %w", ErrTransportFailure, buildErr)
			}
			if writeErr := writeLineBounded(conn, stream, frame, config.requestTimeout()); writeErr != nil {
				_ = stream.Close()
				return writeErr
			}
			continue
		}
		success, err := rpcwire.EncodeSuccess(request, body)
		if err != nil {
			_ = stream.Close()
			return fmt.Errorf("%w: %w", ErrTransportFailure, err)
		}
		if err := writeLineBounded(conn, stream, success, config.requestTimeout()); err != nil {
			_ = stream.Close()
			return err
		}
	}
}

// cappedStream routes TLS reads through the handshake limiter.
type cappedStream struct {
	Stream
	limiter *HandshakeLimiter
}

func (stream cappedStream) Read(data []byte) (int, error) { return stream.limiter.Read(data) }
