package terminstance

// RetryDisposition is the closed §4.C retry disposition enum. Every
// result carries exactly one.
type RetryDisposition string

// Retry dispositions.
const (
	DispositionReplaySame             RetryDisposition = "replay_same"
	DispositionStatusFirst            RetryDisposition = "status_first"
	DispositionNewAuthorization       RetryDisposition = "new_authorization"
	DispositionRequiredOperatorAction RetryDisposition = "required_operator_action"
)

// ParseRetryDisposition admits exactly the four §4.C dispositions.
func ParseRetryDisposition(value string) (RetryDisposition, error) {
	switch RetryDisposition(value) {
	case DispositionReplaySame, DispositionStatusFirst,
		DispositionNewAuthorization, DispositionRequiredOperatorAction:
		return RetryDisposition(value), nil
	default:
		return "", refuse(CodeProtocolError, "retry disposition vocabulary")
	}
}

// DispositionFor maps a terminal failure code to its retry disposition.
// committed reports whether any side effect committed before the failure:
// an error before the first effect restores the source state and may be
// retried or refused by class, while an error after a committed effect
// whose result cannot be proven funnels to status_first, because only a
// successful status read proves what happened. The spec pins the
// disposition vocabulary but not the per-code mapping, so this mapping is
// the leaf's contract, pinned literally per arm:
//
//   - terminal_backend_unauthorized names its own remedy in both cases:
//     the lease moved or lapsed and status cannot fix auth.
//   - terminal_backend_timeout and quiesce_timeout are clean before any
//     commit (retry the identical wait) and uncertain after one.
//   - stop_timeout, idempotency_mismatch, terminal_backend_stale_generation,
//     local_precondition_failed and terminal_backend_unavailable always
//     route through status: a stop wait, a window conflict, a drifted
//     generation, a refused precondition and an unserved backend all
//     require a fresh observation before any retry.
//   - terminal_backend_capability_unproven,
//     terminal_backend_integrity_failure, terminal_backend_process_failed,
//     terminal_backend_protocol_error and any unknown code stop automation
//     before any commit (an operator must prove the capability, heal the
//     backend, or fix the caller) and route through status after one,
//     because the committed effect's result is then unproven.
//
// A code the row forbids never reaches here: the engine routes backend
// reports through CheckErrorAllowed first, so the default arm below fires
// only for an uncoded failure the engine classified itself.
func DispositionFor(code string, committed bool) RetryDisposition {
	switch code {
	case CodeUnauthorized:
		return DispositionNewAuthorization
	case CodeTimeout, CodeQuiesceTimeout:
		if committed {
			return DispositionStatusFirst
		}
		return DispositionReplaySame
	case CodeStopTimeout, CodeIdempotencyMismatch, CodeStaleGeneration,
		CodePreconditionFailed, CodeUnavailable:
		return DispositionStatusFirst
	default:
		if committed {
			return DispositionStatusFirst
		}
		return DispositionRequiredOperatorAction
	}
}
