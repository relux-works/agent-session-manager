package provhost

// This file owns the single Provider protocol 2.0.0 idempotency key
// (operation, operation_id). Every provider mutation request that
// carries operation_id uses this key and no other: an identical
// canonical mutation request MUST return a byte-identical result
// across a fresh plugin process, and any changed member is
// idempotency_mismatch with no new mutation. There is no second
// tuple- or path-based key in provider protocol 2.0.0, so this file
// defines exactly one key type and exactly one keyed-operation set.

// keyedOperations lists the operations whose request bodies carry
// operation_id in the Section 7.5 registry: the four materialization
// mutations plus fork, whose operation ID is the Session Record's
// fork_provenance.operation_id. Fork is side-effect-free, but an
// identical retry still returns the same plan and a changed body is
// still idempotency_mismatch, so it carries the key. Status reads
// omit operation_id by construction and are evolving reads, never
// receipts.
var keyedOperations = []Operation{
	OpCapture,
	OpMaterialize,
	OpMaterializeCommit,
	OpMaterializeRollback,
	OpFork,
}

// MutationOperations returns the operations keyed by
// (operation, operation_id), in registry order. The result is a copy;
// the set cannot be mutated through it.
func MutationOperations() []Operation {
	return append([]Operation(nil), keyedOperations...)
}

// isKeyedOperation reports whether the operation carries an
// operation_id mutation key.
func isKeyedOperation(operation Operation) bool {
	for _, keyed := range keyedOperations {
		if operation == keyed {
			return true
		}
	}
	return false
}

// IdempotencyKey is the sole provider mutation deduplication key:
// the operation tag plus the caller-supplied operation ID. One
// operation ID used by the materialize family identifies exactly one
// materialization transaction, and reusing the mutation value across
// distinct operation tags does not alias their recorded results,
// because the tag is part of the key.
type IdempotencyKey struct {
	operation   Operation
	operationID string
}

// Operation reports the mutation tag the key was issued for.
func (key IdempotencyKey) Operation() Operation {
	return key.operation
}

// OperationID reports the caller-supplied operation ID the key was
// issued for.
func (key IdempotencyKey) OperationID() string {
	return key.operationID
}

// String renders the key in the Section 7.5 (operation,
// operation_id) form.
func (key IdempotencyKey) String() string {
	return "(" + string(key.operation) + ", " + key.operationID + ")"
}

// IdempotencyKeyFor issues the mutation key for one operation tag and
// one caller-supplied operation ID. An unknown operation, an
// operation that carries no operation_id, and a non-UUIDv7 operation
// ID are invalid_config caller errors: a retry under a forged,
// mistagged, or unkeyed identity is not a retry at all.
func IdempotencyKeyFor(operation Operation, operationID string) (IdempotencyKey, error) {
	if !validOperation(string(operation)) {
		failure, err := failInvalid("idempotency key names an unknown operation")
		if err != nil {
			return IdempotencyKey{}, err
		}
		return IdempotencyKey{}, failure
	}
	if !isKeyedOperation(operation) {
		failure, err := failInvalid("idempotency key names an operation without operation_id")
		if err != nil {
			return IdempotencyKey{}, err
		}
		return IdempotencyKey{}, failure
	}
	if !isUUIDv7(operationID) {
		failure, err := failInvalid("idempotency operation_id is not a UUIDv7")
		if err != nil {
			return IdempotencyKey{}, err
		}
		return IdempotencyKey{}, failure
	}
	return IdempotencyKey{operation: operation, operationID: operationID}, nil
}
