package termbind

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const admissionLockName = ".admission.lock"

// AcquireAdmission holds the per-instance attach-admission lock until
// release is called. Lifecycle callers hold it across the peer census,
// capability decision, and durable receipt commit. The OS lock makes this
// boundary shared by independently opened stores and processes. The lock
// file is persistent: removing it after unlock could split contenders
// across two inodes.
func (store *AttachStore) AcquireAdmission(ctx context.Context, instanceID string) (release func(), err error) {
	if ctx == nil {
		return nil, errors.New("attach admission context is nil")
	}
	if err := CheckInstanceIdentity(instanceID); err != nil {
		return nil, err
	}
	if _, err := scalar.ParseUUIDv7(instanceID); err != nil {
		return nil, fmt.Errorf("attach admission instance %q is not a UUIDv7: %w", instanceID, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dir := store.instanceDir(instanceID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create attach admission directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(dir, admissionLockName), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open attach admission lock: %w", err)
	}
	if store.hooks != nil && store.hooks.BeforeAdmissionLock != nil {
		store.hooks.BeforeAdmissionLock(instanceID)
	}
	for {
		if err := ctx.Err(); err != nil {
			_ = file.Close()
			return nil, err
		}
		locked, unlock, lockErr := tryLockAdmissionFile(file)
		if lockErr != nil {
			_ = file.Close()
			return nil, fmt.Errorf("lock attach admission: %w", lockErr)
		}
		if locked {
			return func() {
				_ = unlock()
				_ = file.Close()
			}, nil
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			_ = file.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
