//go:build windows

package secconftest

import "errors"

// makeFifo is unavailable on Windows: named pipes there are not
// filesystem FIFOs and this instrument does not emulate them. The
// fifo probe reports unavailable (and skips, since Windows is not
// strict) instead of pretending.
func makeFifo(dir, name string) (string, error) {
	return fifoBuild(dir, name)
}

// fifoBuild is the seam the probe-honesty test faults. It always
// fails on Windows, matching makeFifo; the test only asserts the
// probe reports the failure honestly.
var fifoBuild = func(dir, name string) (string, error) {
	return "", errors.New("secconftest: named-pipe fixtures are unavailable on Windows")
}
