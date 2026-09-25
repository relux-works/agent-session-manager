package clonefidelity

import "testing"

// TestCheckRowCountEdges pins the exact dispositions cardinality
// bound at the edge and edge+1: 999999 and 1000000 admit, 0 and
// 1000001 refuse. This helper test pairs with the entry-level
// reachability rows in TestRowCountEdges (0 and 1000001 refuse
// through Build and Decode, 1 admits): the entry tests prove the
// entries reach this gate, this test proves the gate's exact bound
// without sealing a million-row report in the suite. The bound
// narrowing N-rowcount-max kills this test alone.
func TestCheckRowCountEdges(t *testing.T) {
	for _, count := range []int{1, 2, 999999, 1000000} {
		if err := checkRowCount(count); err != nil {
			t.Errorf("checkRowCount(%d) error = %v, want admit", count, err)
		}
	}
	for _, count := range []int{0, -1, 1000001, 2000000} {
		if err := checkRowCount(count); err == nil {
			t.Errorf("checkRowCount(%d) = nil, want refusal", count)
		}
	}
}
