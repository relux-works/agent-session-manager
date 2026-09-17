package fencing

// leaseSealToken is deliberately unexported and is minted only by the
// free mintLeaseToken constructor below. The type-level census pins
// that construction boundary across every production source file; the
// token then makes a zero or field-assembled LeaseToken unusable at
// runtime. There is no exported constructor and no exported field: a
// caller outside this package cannot spell a live token.
type leaseSealToken struct{}

// leaseSeal holds the fenced authority a passed gate established: the
// session with the exact winning (epoch, lease_id) tuple. Its fields
// are private so callers cannot treat the minted projection as a
// public record-shaped DTO.
type leaseSeal struct {
	token   *leaseSealToken
	session string
	epoch   uint64
	leaseID string
}

// LeaseToken is the sealed fencing capability Authorize mints for a
// passed gate: the Section 7.5 LeaseToken projection
// {session_id, lease_epoch, lease_id} for provider operations. A zero
// value or a value assembled without the constructor-only seal token
// is not authority, even when a caller fills the outer struct.
type LeaseToken struct {
	seal *leaseSeal
}

// mintLeaseToken is the single constructor for a live LeaseToken. It
// runs only at the end of Authorize, after every ownership, grant,
// and tuple arm has passed.
func mintLeaseToken(session string, epoch uint64, leaseID string) LeaseToken {
	return LeaseToken{seal: &leaseSeal{token: &leaseSealToken{}, session: session, epoch: epoch, leaseID: leaseID}}
}

// authority opens the seal for the capability consumers. A missing
// seal or seal token refuses lease_conflict before any other check,
// so forgery can never degrade into a different class.
func (token LeaseToken) authority() (*leaseSeal, error) {
	if token.seal == nil || token.seal.token == nil {
		return nil, refuse(ErrLeaseConflict, "fencing capability seal is missing")
	}
	return token.seal, nil
}

// ProviderOperation is one Section 7.5 provider operation that carries
// the LeaseToken. Resume, stop, materialize-commit, and
// native-store-plan carry the same triple on the wire; their callers
// have no gate entry in this package yet, so binding them here would
// be an untestable claim and stays a stated bound.
type ProviderOperation string

const (
	// ProviderQuiesce binds the token to a quiesce operation.
	ProviderQuiesce ProviderOperation = "quiesce"
	// ProviderCapture binds the token to a capture operation.
	ProviderCapture ProviderOperation = "capture"
	// ProviderMaterialize binds the token to a materialize operation.
	ProviderMaterialize ProviderOperation = "materialize"
)

// validProviderOperation reports whether op is a bound provider
// operation. Anything else is a malformed bind call.
func validProviderOperation(op ProviderOperation) bool {
	switch op {
	case ProviderQuiesce, ProviderCapture, ProviderMaterialize:
		return true
	}
	return false
}

// ProviderLease is the wire projection of a bound capability: the
// Section 7.5 LeaseToken triple with the operation it authorizes. It
// carries no seal because it is evidence of one past authorization,
// not authority for another; only Bind mints it, and only from a live
// sealed token.
type ProviderLease struct {
	SessionID string
	Epoch     uint64
	LeaseID   string
	Operation ProviderOperation
}

// Bind projects a live LeaseToken onto one provider operation. A zero
// or forged token refuses lease_conflict; an unknown operation refuses
// invalid_arguments. Only a token minted by a passed Authorize gate
// reaches the projection.
func (token LeaseToken) Bind(operation ProviderOperation) (ProviderLease, error) {
	seal, err := token.authority()
	if err != nil {
		return ProviderLease{}, err
	}
	if !validProviderOperation(operation) {
		return ProviderLease{}, refuse(ErrInvalidArguments, "unknown provider operation %q", string(operation))
	}
	return ProviderLease{SessionID: seal.session, Epoch: seal.epoch, LeaseID: seal.leaseID, Operation: operation}, nil
}
