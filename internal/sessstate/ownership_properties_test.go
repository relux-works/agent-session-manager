package sessstate

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Ownership reducer properties (TASK-260830-2atgj4).
//
// The four story-pinned v0.6.0 invariants over the landed production
// reducers (Reduce/resolveWinner/Compare; sessrepo.WinningLease;
// fencing.Authorize*) under Sections 2.2, 5.3 and 13.6-13.10:
//
//  1. union-order independence — every permutation of a union multiset
//     yields a byte-identical projection;
//  2. loser preservation — a losing lease is never dropped, rewritten
//     or promoted; its history stays visible as conflict evidence;
//  3. clock non-authority — created_at and wall-clock members never
//     influence the winner, the fencing decision or the parked/refused
//     outcome (store and gate halves live in sessrepo and fencing);
//  4. zero duplicate authorized owners — at most one authoritative
//     lease per session and epoch (store and gate halves live in
//     sessrepo and fencing).
//
// This file proves invariants 1 and 2 over Reduce. The generator is a
// closed small alphabet — epochs 1..4, lease IDs R/A/B/C (bytewise
// R < A < B < C) — with exhaustive enumeration where the space is
// small (all multisets of size 0..3, all distinct permutations, all
// ordered 2-partitions, three chain shapes) and bounded seeded-random
// exploration otherwise (seed and iteration counts recorded below).
// No new module dependency: the generators are hand-written loops.

// propLeaseIDs is the closed lease alphabet in bytewise order.
var propLeaseIDs = []string{testLeaseIDR, testLeaseID, testLeaseIDB, testLeaseIDC}

// propEpochs is the closed epoch alphabet.
var propEpochs = []uint64{1, 2, 3, 4}

// propAtoms is every (epoch, lease) pair of the closed alphabet.
func propAtoms() []LeaseHead {
	var atoms []LeaseHead
	for _, epoch := range propEpochs {
		for _, leaseID := range propLeaseIDs {
			atoms = append(atoms, LeaseHead{Epoch: epoch, LeaseID: leaseID})
		}
	}
	return atoms
}

// propMax is the independent winner oracle: the greatest (epoch,
// lease_id) tuple using only builtin operators, never production
// Compare. Directional agreement between propMax and Reduce pins the
// bytewise-greater-wins tie-break; a reducer that inverts the
// tie-break still permutes identically but disagrees with propMax.
func propMax(heads []LeaseHead) LeaseHead {
	var best LeaseHead
	for _, head := range heads {
		if head.Epoch > best.Epoch ||
			(head.Epoch == best.Epoch && head.LeaseID > best.LeaseID) {
			best = head
		}
	}
	return best
}

// propChain is one chain shape with its distinct envelope leases.
type propChain struct {
	name   string
	events []Event
	leases []LeaseHead
}

// propChains builds the three chain shapes: empty, one-epoch
// bootstrap under A@1, and a force-takeover succession A@1 -> B@2.
func propChains(t *testing.T, record Record) []propChain {
	t.Helper()
	raw := buildRecord(t, nil)
	_, bootstrap, _ := buildChain(t, raw, baseBootstrap(record))
	_, succession, _ := buildChain(t, raw, append(baseBootstrap(record),
		eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)},
	))
	return []propChain{
		{name: "empty", events: nil, leases: nil},
		{name: "bootstrap", events: bootstrap, leases: []LeaseHead{{Epoch: 1, LeaseID: testLeaseID}}},
		{name: "succession", events: succession, leases: []LeaseHead{{Epoch: 1, LeaseID: testLeaseID}, {Epoch: 2, LeaseID: testLeaseIDB}}},
	}
}

// propMultisets enumerates every multiset of atoms of size 0..maxSize
// as index combinations with replacement.
func propMultisets(atoms []LeaseHead, maxSize int) [][]LeaseHead {
	var out [][]LeaseHead
	out = append(out, nil)
	indices := make([]int, 0, maxSize)
	var rec func(start, depth, size int)
	for size := 1; size <= maxSize; size++ {
		indices = indices[:0]
		rec = func(start, depth, _ int) {
			if depth == size {
				multiset := make([]LeaseHead, size)
				for i, index := range indices {
					multiset[i] = atoms[index]
				}
				out = append(out, multiset)
				return
			}
			for i := start; i < len(atoms); i++ {
				indices = append(indices, i)
				rec(i, depth+1, size)
				indices = indices[:len(indices)-1]
			}
		}
		rec(0, 0, size)
	}
	return out
}

// propPermutations enumerates every distinct permutation of a
// multiset. Sizes stay small (at most 3! = 6 exhaustive), so a
// Heap's enumeration with an encoding set is enough.
func propPermutations(multiset []LeaseHead) [][]LeaseHead {
	if len(multiset) == 0 {
		return [][]LeaseHead{nil}
	}
	encode := func(heads []LeaseHead) string {
		var sb strings.Builder
		for _, head := range heads {
			fmt.Fprintf(&sb, "%d/%s;", head.Epoch, head.LeaseID)
		}
		return sb.String()
	}
	var out [][]LeaseHead
	seen := map[string]struct{}{}
	work := append([]LeaseHead(nil), multiset...)
	var generate func(n int)
	generate = func(n int) {
		if n == 1 {
			if key := encode(work); key != "" {
				if _, dup := seen[key]; !dup {
					seen[key] = struct{}{}
					out = append(out, append([]LeaseHead(nil), work...))
				}
			}
			return
		}
		for i := 0; i < n; i++ {
			generate(n - 1)
			if n%2 == 1 {
				work[0], work[n-1] = work[n-1], work[0]
			} else {
				work[i], work[n-1] = work[n-1], work[i]
			}
		}
	}
	generate(len(work))
	return out
}

// propSortedUnion is the canonical representative of a union
// multiset: ascending tuple order by the independent oracle order.
func propSortedUnion(multiset []LeaseHead) []LeaseHead {
	sorted := append([]LeaseHead(nil), multiset...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Epoch != sorted[j].Epoch {
			return sorted[i].Epoch < sorted[j].Epoch
		}
		return sorted[i].LeaseID < sorted[j].LeaseID
	})
	return sorted
}

// propPartitions enumerates every ordered 2-partition concatenation of
// a multiset: the 2^n ways to split its positions into a first and a
// second successive observation, concatenated back into one union
// list. Every concatenation is a permutation of the multiset, so
// permutation-invariance entails partition-invariance; the partition
// test executes the enumeration distinctly as the reconnect
// narrative (observe U1, then U1++U2 after reconnect).
func propPartitions(multiset []LeaseHead) [][]LeaseHead {
	var out [][]LeaseHead
	for mask := 0; mask < (1 << len(multiset)); mask++ {
		var first, second []LeaseHead
		for i, head := range multiset {
			if mask&(1<<i) == 0 {
				first = append(first, head)
			} else {
				second = append(second, head)
			}
		}
		out = append(out, append(first, second...))
	}
	return out
}

func propReduce(t *testing.T, record Record, chain propChain, union []LeaseHead) Projection {
	t.Helper()
	projection, err := Reduce(Input{Record: record, Events: chain.events, Union: union})
	if err != nil {
		t.Fatalf("Reduce(chain %s union %+v) error = %v", chain.name, union, err)
	}
	return projection
}

// TestOwnershipUnionOrderIndependent proves invariant 1: for every
// union multiset over the closed alphabet, every distinct
// permutation reduces to a byte-identical projection (winner,
// conflicts, divergent-history warnings), and the winner equals the
// independent tuple maximum. Exhaustive over multisets of size 0..3
// (969 multisets over 16 atoms); bounded seeded-random over sizes
// 4..6 (seed 260830, 2000 multisets, at most 24 sampled permutations
// each).
func TestOwnershipUnionOrderIndependent(t *testing.T) {
	record := decodeTestRecord(t)
	chains := propChains(t, record)
	atoms := propAtoms()
	var reduceCalls int
	check := func(chain propChain, multiset, union []LeaseHead, canonical Projection) {
		t.Helper()
		got := propReduce(t, record, chain, union)
		reduceCalls++
		if !reflect.DeepEqual(got, canonical) {
			t.Fatalf("chain %s multiset %+v permutation %+v:\ngot  %+v\nwant %+v",
				chain.name, multiset, union, got, canonical)
		}
		if want := propMax(append(append([]LeaseHead(nil), chain.leases...), multiset...)); got.Winner != want {
			t.Fatalf("chain %s multiset %+v: winner = %+v, want independent maximum %+v",
				chain.name, multiset, got.Winner, want)
		}
	}
	multisets := propMultisets(atoms, 3)
	for _, chain := range chains {
		for _, multiset := range multisets {
			canonical := propReduce(t, record, chain, propSortedUnion(multiset))
			reduceCalls++
			for _, permutation := range propPermutations(multiset) {
				check(chain, multiset, permutation, canonical)
			}
		}
	}
	exhaustiveMultisets := len(multisets) * len(chains)
	rng := rand.New(rand.NewSource(260830))
	const randomMultisets = 2000
	for i := 0; i < randomMultisets; i++ {
		size := 4 + rng.Intn(3)
		multiset := make([]LeaseHead, size)
		for j := range multiset {
			multiset[j] = atoms[rng.Intn(len(atoms))]
		}
		permutations := propPermutations(multiset)
		if len(permutations) > 24 {
			sampled := make([][]LeaseHead, 0, 24)
			for _, index := range rng.Perm(len(permutations))[:24] {
				sampled = append(sampled, permutations[index])
			}
			permutations = sampled
		}
		chain := chains[rng.Intn(len(chains))]
		canonical := propReduce(t, record, chain, propSortedUnion(multiset))
		reduceCalls++
		for _, permutation := range permutations {
			check(chain, multiset, permutation, canonical)
		}
	}
	t.Logf("union-order: %d exhaustive (multiset, chain) pairs + %d seeded-random multisets (seed 260830), %d Reduce calls",
		exhaustiveMultisets, randomMultisets, reduceCalls)
}

// TestOwnershipUnionOrderRegression pins the defect witness that
// motivated the deterministic losing-branch report: the rival B@2
// below the off-chain winner C@3 must be reported identically
// whether it arrives before or after the winner.
func TestOwnershipUnionOrderRegression(t *testing.T) {
	record := decodeTestRecord(t)
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, baseBootstrap(record))
	first := propReduce(t, record, propChain{name: "bootstrap", events: events},
		[]LeaseHead{{Epoch: 3, LeaseID: testLeaseIDC}, {Epoch: 2, LeaseID: testLeaseIDB}})
	second := propReduce(t, record, propChain{name: "bootstrap", events: events},
		[]LeaseHead{{Epoch: 2, LeaseID: testLeaseIDB}, {Epoch: 3, LeaseID: testLeaseIDC}})
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("union order changed the projection:\nfirst  %+v\nsecond %+v", first, second)
	}
}

// TestOwnershipUnionPartitionStable proves the successive-union half
// of invariant 1: every ordered 2-partition of a union multiset,
// concatenated back into one union list, reduces to the canonical
// projection. Exhaustive over multisets of size 0..3 across the
// three chain shapes.
func TestOwnershipUnionPartitionStable(t *testing.T) {
	record := decodeTestRecord(t)
	chains := propChains(t, record)
	var reduceCalls, partitions int
	for _, chain := range chains {
		for _, multiset := range propMultisets(propAtoms(), 3) {
			canonical := propReduce(t, record, chain, propSortedUnion(multiset))
			reduceCalls++
			for _, concat := range propPartitions(multiset) {
				partitions++
				got := propReduce(t, record, chain, concat)
				reduceCalls++
				if !reflect.DeepEqual(got, canonical) {
					t.Fatalf("chain %s multiset %+v partition %+v:\ngot  %+v\nwant %+v",
						chain.name, multiset, concat, got, canonical)
				}
			}
		}
	}
	t.Logf("union-partition: %d partition concatenations, %d Reduce calls", partitions, reduceCalls)
}

// TestOwnershipLoserHistoryPreserved proves invariant 2 over the
// reducer: every union lease strictly below the winning tuple is
// named in a losing_branch_preserved conflict as "(epoch, lease-id)"
// with the final winner as the target; every epoch shared by two
// distinct leases is named in a same_epoch_tie conflict; the winner
// is the independent maximum (a loser is never promoted); and the
// authoritative derivation (state, checkpoint, provider, terminal,
// owner) equals the union-free projection (a loser never rewrites
// state). Exhaustive over multisets of size 0..3 across the three
// chain shapes.
func TestOwnershipLoserHistoryPreserved(t *testing.T) {
	record := decodeTestRecord(t)
	chains := propChains(t, record)
	var reduceCalls, multisets int
	for _, chain := range chains {
		baseline := propReduce(t, record, chain, nil)
		reduceCalls++
		for _, multiset := range propMultisets(propAtoms(), 3) {
			multisets++
			got := propReduce(t, record, chain, propSortedUnion(multiset))
			reduceCalls++
			winner := propMax(append(append([]LeaseHead(nil), chain.leases...), multiset...))
			if got.Winner != winner {
				t.Fatalf("chain %s multiset %+v: winner = %+v, want %+v",
					chain.name, multiset, got.Winner, winner)
			}
			seen := map[LeaseHead]struct{}{}
			for _, loser := range multiset {
				if _, dup := seen[loser]; dup {
					continue
				}
				seen[loser] = struct{}{}
				if loser.Epoch > winner.Epoch ||
					(loser.Epoch == winner.Epoch && loser.LeaseID >= winner.LeaseID) {
					continue
				}
				fragment := fmt.Sprintf("(%d, %s)", loser.Epoch, loser.LeaseID)
				found := false
				for _, conflict := range got.Conflicts {
					if conflict.Kind == ConflictLosingBranchPreserved &&
						strings.Contains(conflict.Detail, fragment) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("chain %s multiset %+v: losing lease %+v has no losing_branch_preserved evidence in %+v",
						chain.name, multiset, loser, got.Conflicts)
				}
			}
			byEpoch := map[uint64]map[string]struct{}{}
			for _, head := range append(append([]LeaseHead(nil), chain.leases...), multiset...) {
				if byEpoch[head.Epoch] == nil {
					byEpoch[head.Epoch] = map[string]struct{}{}
				}
				byEpoch[head.Epoch][head.LeaseID] = struct{}{}
			}
			for epoch, rivals := range byEpoch {
				if len(rivals) < 2 {
					continue
				}
				fragment := fmt.Sprintf("epoch %d is shared", epoch)
				found := false
				for _, conflict := range got.Conflicts {
					if conflict.Kind == ConflictSameEpochTie &&
						strings.Contains(conflict.Detail, fragment) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("chain %s multiset %+v: epoch %d tie has no same_epoch_tie evidence in %+v",
						chain.name, multiset, epoch, got.Conflicts)
				}
			}
			if got.State != baseline.State ||
				got.HasCheckpoint != baseline.HasCheckpoint || got.Newest != baseline.Newest ||
				got.Provider != baseline.Provider || got.HasTerminal != baseline.HasTerminal ||
				got.Terminal != baseline.Terminal || got.OwnerHostID != baseline.OwnerHostID {
				t.Fatalf("chain %s multiset %+v: union rewrote authoritative state:\ngot  %+v\nbase %+v",
					chain.name, multiset, got, baseline)
			}
		}
	}
	t.Logf("loser-preserved: %d (multiset, chain) pairs, %d Reduce calls", multisets, reduceCalls)
}

// TestOwnershipWinnerIsTupleMaximum pins the Compare direction the
// union properties rely on: Compare agrees in sign with the
// independent oracle over every pair of the closed alphabet plus the
// zero head, is reflexive, antisymmetric, and transitive over every
// triple. Exhaustive: 17 atoms, 289 pairs, 4913 triples.
func TestOwnershipWinnerIsTupleMaximum(t *testing.T) {
	atoms := append([]LeaseHead{{}}, propAtoms()...)
	sign := func(value int) int {
		switch {
		case value < 0:
			return -1
		case value > 0:
			return 1
		}
		return 0
	}
	oracle := func(a, b LeaseHead) int {
		return sign(func() int {
			best := propMax([]LeaseHead{a, b})
			switch {
			case best == a && best == b:
				return 0
			case best == a:
				return 1
			default:
				return -1
			}
		}())
	}
	for _, a := range atoms {
		if got := Compare(a, a); got != 0 {
			t.Fatalf("Compare(%+v, %+v) = %d, want 0", a, a, got)
		}
		for _, b := range atoms {
			if got, want := sign(Compare(a, b)), oracle(a, b); got != want {
				t.Fatalf("Compare(%+v, %+v) sign = %d, want oracle %d", a, b, got, want)
			}
			if Compare(a, b) != -Compare(b, a) {
				t.Fatalf("Compare(%+v, %+v) = %d but Compare(%+v, %+v) = %d, want antisymmetry",
					a, b, Compare(a, b), b, a, Compare(b, a))
			}
		}
	}
	for _, a := range atoms {
		for _, b := range atoms {
			for _, c := range atoms {
				if Compare(a, b) >= 0 && Compare(b, c) >= 0 && Compare(a, c) < 0 {
					t.Fatalf("Compare is not transitive at %+v, %+v, %+v", a, b, c)
				}
			}
		}
	}
	t.Logf("winner-maximum: %d atoms, %d pairs, %d triples", len(atoms), len(atoms)*len(atoms), len(atoms)*len(atoms)*len(atoms))
}
