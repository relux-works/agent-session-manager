package clonesnap

import (
	"fmt"
)

// This file owns the capture log. Section 10.2 requires logs to
// record digest, size, and media type only, never blob contents: the
// only record entry takes the member key, class, digest, size,
// media type, and reason — there is no parameter blob contents could
// cross. The leak suite scans every emitted line for fixture secrets.

// CaptureLog is one append-only capture log.
type CaptureLog struct {
	lines []string
}

// Linef appends one formatted log line. Callers pass digests, sizes,
// media types, member keys, and stable reasons only — never source
// bytes.
func (log *CaptureLog) Linef(format string, arguments ...any) {
	if log == nil {
		return
	}
	log.lines = append(log.lines, fmt.Sprintf(format, arguments...))
}

// Lines returns a copy of the recorded lines.
func (log *CaptureLog) Lines() []string {
	if log == nil {
		return nil
	}
	return append([]string(nil), log.lines...)
}
