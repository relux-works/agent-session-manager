//go:build !darwin && !linux

package gitsnap

import "errors"

// makeFifo reports unavailability where the test platform has no fifo.
func makeFifo(_ string) error {
	return errors.New("fifo unavailable on this platform")
}
