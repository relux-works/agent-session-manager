//go:build !darwin && !linux

package localstore

type projectionLock struct{}

func acquireProjectionLock(string) (*projectionLock, error) {
	return nil, projectionRefusal(ErrUnsupportedPlatform, "SQLite projection lock")
}

func (lock *projectionLock) release() error { return nil }
