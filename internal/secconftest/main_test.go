package secconftest

import (
	"fmt"
	"os"
	"testing"
)

// TestMain runs the suite and then prints the process-wide capability
// counters, so a real verdict recorded by a parallel Require is
// observable in the output. Sequential tests (including
// TestSkipReportReadsRecordedVerdicts and the fake-Require wiring
// tests) print before the first parallel CONT, so without this hook
// no printed report can carry a parallel verdict under any host
// condition. The per-test --- SKIP / --- FAIL lines under go test -v
// remain the primary skip datum; this line witnesses the counters
// after every test, including the parallel ones.
func TestMain(m *testing.M) {
	code := m.Run()
	fmt.Printf("%s\n", Report())
	os.Exit(code)
}
