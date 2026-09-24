package merkleinventory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

type syncPropertyObject struct {
	namespace string
	id        string
	data      []byte
	lease     sessstate.LeaseHead
}

type generatedSyncCase struct {
	index          int
	scenario       int
	size           int
	skew           int
	orderIndex     int
	order          []int
	duplicate      bool
	gapTarget      int
	peerMask       int
	conflict       bool
	conflictTarget int
	conflictRound  int
}

var syncReferenceNamespaces = []string{"blob", "event", "manifest", "record", "tombstone", "tombstone_ack"}

func TestDurableSyncGeneratedPerturbationProduct(t *testing.T) {
	runFlag := flag.Lookup("test.run")
	maxSize := 3
	if runFlag != nil && strings.Contains(runFlag.Value.String(), "/") {
		// Explicit nested selectors opt into the full finite product. The
		// configured suite always runs the bounded N<=3 shard below.
		maxSize = 6
	}

	repoRoot := t.TempDir()
	repo, err := sessrepo.Open(filepath.Join(repoRoot, "authoritative"))
	if err != nil {
		t.Fatal(err)
	}
	record := validSessionRecordJSON(t)
	if _, err := repo.CreateSession(record); err != nil {
		t.Fatal(err)
	}
	cases := make([]generatedSyncCase, 0)
	validCaseCount, conflictCaseCount := 0, 0
	conflictTargets := make(map[int]map[int]bool)
	conflictRounds := make(map[int]map[int]bool)
	conflictTargetRounds := make(map[int]map[int]map[int]bool)
	for size := 1; size <= maxSize; size++ {
		for skew := 0; skew < 2; skew++ {
			orders := syncPropertyOrders(size)
			for orderIndex, order := range orders {
				for duplicate := 0; duplicate < 2; duplicate++ {
					for gapIndex := 0; gapIndex < 2; gapIndex++ {
						gapTargets := []int{-1}
						if gapIndex == 1 {
							gapTargets = []int{orderIndex % size}
						}
						for _, gapTarget := range gapTargets {
							for _, peerMask := range syncPropertyPeerMasks(size, gapTarget) {
								scenario := validCaseCount
								cases = append(cases, generatedSyncCase{
									index: len(cases), scenario: scenario, size: size, skew: skew, orderIndex: orderIndex,
									order: append([]int(nil), order...), duplicate: duplicate == 1, gapTarget: gapTarget,
									peerMask: peerMask,
								})
								validCaseCount++
								if size > 1 {
									target := syncPropertyConflictTarget(size, gapTarget, scenario)
									round := (scenario/size)%3 + 1
									if syncPropertyConflictFeasible(size, target, gapTarget, round) {
										cases = append(cases, generatedSyncCase{
											index: len(cases), scenario: scenario, size: size, skew: skew, orderIndex: orderIndex,
											order: append([]int(nil), order...), duplicate: duplicate == 1, gapTarget: gapTarget,
											peerMask: peerMask, conflict: true, conflictTarget: target, conflictRound: round,
										})
										conflictCaseCount++
										if conflictTargets[size] == nil {
											conflictTargets[size] = make(map[int]bool)
										}
										if conflictRounds[size] == nil {
											conflictRounds[size] = make(map[int]bool)
										}
										if conflictTargetRounds[size] == nil {
											conflictTargetRounds[size] = make(map[int]map[int]bool)
										}
										if conflictTargetRounds[size][target] == nil {
											conflictTargetRounds[size][target] = make(map[int]bool)
										}
										conflictTargets[size][target] = true
										conflictRounds[size][round] = true
										conflictTargetRounds[size][target][round] = true
									}
								}
							}
						}
					}
				}
			}
		}
	}
	for size := 2; size <= maxSize; size++ {
		for target := 0; target < size; target++ {
			if !conflictTargets[size][target] {
				t.Fatalf("generated conflict class omits identity position %d for N=%d", target, size)
			}
		}
		for round := 1; round <= 3; round++ {
			if !conflictRounds[size][round] {
				t.Fatalf("generated conflict class omits repeated-sync round %d for N=%d", round, size)
			}
		}
		for target := 0; target < size; target++ {
			for round := 1; round <= 3; round++ {
				if !conflictTargetRounds[size][target][round] {
					t.Fatalf("generated conflict class omits identity position %d at repeated-sync round %d for N=%d", target, round, size)
				}
			}
		}
	}
	if maxSize == 6 {
		t.Logf("generated production SyncFrom product: valid=%d conflict=%d total=%d; N=1..6, every permutation for N<=5, 34 deterministic N=6 permutations, duplicate=[none, replay all], gap=[none, delayed object selected cyclically], skew=[both directions], every proper object peer subset, valid passes=[1..3], every feasible conflict target × conflict round [1..3]", validCaseCount, conflictCaseCount, len(cases))
	} else {
		t.Logf("always-on generated production SyncFrom shard: valid=%d conflict=%d total=%d; N=1..3, every permutation, duplicate=[none, replay all], gap=[none, delayed object selected cyclically], skew=[both directions], every proper object peer subset, valid passes=[1..3]", validCaseCount, conflictCaseCount, len(cases))
	}
	for _, tc := range cases {
		tc := tc
		name := fmt.Sprintf("N%d/skew%d/order%03d/duplicate%t/gap%d/peer%02x", tc.size, tc.skew, tc.orderIndex, tc.duplicate, tc.gapTarget, tc.peerMask)
		if tc.conflict {
			name += fmt.Sprintf("/conflict-target%02d-round%d", tc.conflictTarget, tc.conflictRound)
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if tc.conflict {
				runDurableSyncGeneratedConflictCase(t, tc)
				return
			}
			runDurableSyncGeneratedCase(t, repo, tc.size, tc.skew, tc.orderIndex, tc.order, tc.duplicate, tc.gapTarget, tc.peerMask, tc.index)
		})
	}
}

func runDurableSyncGeneratedCase(t *testing.T, repo *sessrepo.Repository, size, skew, orderIndex int, order []int, duplicate bool, gapTarget, peerMask, caseIndex int) {
	t.Helper()
	objects := generatedSyncPropertyObjects(t, size, skew)
	allMask := (1 << size) - 1
	gapBit := 0
	if gapTarget >= 0 {
		gapBit = 1 << gapTarget
	}
	localMask := allMask ^ peerMask
	if gapTarget >= 0 {
		localMask &^= gapBit
	}
	caseRoot := t.TempDir()
	local, err := OpenDurable(filepath.Join(caseRoot, "local"))
	if err != nil {
		t.Fatalf("case %d OpenDurable(local): %v", caseIndex, err)
	}
	peer, err := OpenDurable(filepath.Join(caseRoot, "peer"))
	if err != nil {
		t.Fatalf("case %d OpenDurable(peer): %v", caseIndex, err)
	}
	seedSyncPropertyMask(t, local, objects, localMask, reverseSyncPropertyOrder(order), duplicate, caseIndex, "local")
	seedSyncPropertyMask(t, peer, objects, peerMask, order, duplicate, caseIndex, "peer")

	knownLocalMask := localMask
	var stableStore map[string][]byte
	for pass := 1; pass <= 3; pass++ {
		if gapTarget >= 0 && pass == 2 {
			addSyncPropertyObject(t, peer, objects[gapTarget], duplicate, caseIndex, "delayed gap")
		}
		beforeIDs := syncStoreIDs(local)
		syncErr := local.SyncFrom(peer, testRequestIDs())
		expectedClosureConflict := gapTarget == 4 && size == 6 && pass == 1
		if expectedClosureConflict {
			assertSyncLiteralConflict(t, syncErr, "SyncFrom(gap Tombstone with unioned Acknowledgement)")
		} else if syncErr != nil {
			t.Fatalf("case %d sync pass %d (N=%d order=%d duplicate=%t gapTarget=%d skew=%d peer=%06b): %v",
				caseIndex, pass, size, orderIndex, duplicate, gapTarget, skew, peerMask, syncErr)
		}
		if gapTarget >= 0 && pass >= 2 {
			knownLocalMask |= gapBit
		}
		knownLocalMask |= peerMask
		assertSyncIDsMonotone(t, beforeIDs, syncStoreIDs(local), caseIndex, pass)
		assertSyncRootsMatchReference(t, local, objectsForMask(objects, knownLocalMask), caseIndex, pass)
		if expectedClosureConflict {
			continue
		}
		if knownLocalMask == allMask {
			assertSyncObjectsMatchReference(t, local, objects, caseIndex)
			if stableStore == nil {
				stableStore = snapshotDurableStore(t, local.root)
			} else if !reflect.DeepEqual(snapshotDurableStore(t, local.root), stableStore) {
				t.Fatalf("case %d pass %d changed byte-identical store after convergence", caseIndex, pass)
			}
		}
	}
	if stableStore == nil {
		t.Fatalf("case %d ended without reaching its generated reference set", caseIndex)
	}
	projection, err := local.RebuildProjection(repo, testSessionID)
	if err != nil {
		t.Fatalf("case %d RebuildProjection: %v", caseIndex, err)
	}
	assertSyncProjectionReference(t, projection, objects, caseIndex)
	if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
		t.Fatalf("case %d post-convergence SyncFrom: %v", caseIndex, err)
	}
	if after := snapshotDurableStore(t, local.root); !reflect.DeepEqual(after, stableStore) {
		t.Fatalf("case %d post-convergence sync changed durable store bytes", caseIndex)
	}
}

func runDurableSyncGeneratedConflictCase(t *testing.T, tc generatedSyncCase) {
	t.Helper()
	objects := generatedSyncPropertyObjects(t, tc.size, tc.skew)
	allMask := (1 << tc.size) - 1
	targetBit := 1 << tc.conflictTarget
	gapBit := 0
	if tc.gapTarget >= 0 {
		gapBit = 1 << tc.gapTarget
	}
	peerOnly := -1
	for identity := 0; identity < tc.size; identity++ {
		if identity != tc.conflictTarget && identity != tc.gapTarget {
			peerOnly = identity
			break
		}
	}
	if peerOnly < 0 {
		peerOnly = tc.gapTarget
	}
	peerOnlyBit := 1 << peerOnly
	basePeerMask := tc.peerMask
	localMask := (allMask ^ basePeerMask) | targetBit
	localMask &^= peerOnlyBit | gapBit
	if tc.size == 6 {
		if tc.gapTarget == 4 {
			localMask &^= 1 << 5 // Never leave a local acknowledgement without its delayed Tombstone.
			basePeerMask &^= 1 << 5
		} else {
			if localMask&(1<<5) != 0 || tc.conflictTarget == 5 {
				localMask |= 1 << 4
			}
			if basePeerMask&(1<<5) != 0 || tc.conflictTarget == 5 {
				basePeerMask |= 1 << 4
			}
		}
		if tc.conflictTarget == 5 && tc.gapTarget == 4 {
			t.Fatalf("generated conflict case %d has no local Tombstone for its conflicting Acknowledgement", tc.index)
		}
	}
	caseRoot := t.TempDir()
	local, err := OpenDurable(filepath.Join(caseRoot, "local"))
	if err != nil {
		t.Fatalf("case %d OpenDurable(local): %v", tc.index, err)
	}
	seedSyncPropertyMask(t, local, objects, localMask, reverseSyncPropertyOrder(tc.order), tc.duplicate, tc.index, "conflict local")

	variant := append(bytes.Clone(objects[tc.conflictTarget].data), '\n')
	gapFillRound := 2
	if tc.conflictRound == 3 {
		gapFillRound = 3
	}
	for pass := 1; pass <= 3; pass++ {
		peerMask := basePeerMask | targetBit
		if pass >= tc.conflictRound {
			peerMask |= peerOnlyBit
		} else {
			peerMask &^= peerOnlyBit
		}
		if gapBit != 0 && pass >= gapFillRound {
			peerMask |= gapBit
		} else {
			peerMask &^= gapBit
		}
		if tc.size == 6 && peerMask&(1<<5) != 0 && tc.gapTarget != 4 {
			peerMask |= 1 << 4
		}
		if tc.size == 6 && tc.gapTarget == 5 && pass >= gapFillRound {
			peerMask |= 1 << 4
		}
		peerMask &= allMask
		peer, err := OpenDurable(filepath.Join(caseRoot, fmt.Sprintf("peer-%d", pass)))
		if err != nil {
			t.Fatalf("case %d OpenDurable(peer %d): %v", tc.index, pass, err)
		}
		if pass < tc.conflictRound {
			seedSyncPropertyMask(t, peer, objects, peerMask, tc.order, tc.duplicate, tc.index, "pre-conflict peer")
			beforeIDs := syncStoreIDs(local)
			if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
				t.Fatalf("case %d pass %d before generated conflict: %v", tc.index, pass, err)
			}
			assertSyncIDsMonotone(t, beforeIDs, syncStoreIDs(local), tc.index, pass)
			continue
		}
		seedSyncPropertyConflictMask(t, peer, objects, peerMask, tc.order, tc.duplicate, tc.conflictTarget, variant, tc.index)
		peerBytesBefore := snapshotDurableStore(t, peer.root)
		err = local.SyncFrom(peer, testRequestIDs())
		assertSyncLiteralConflict(t, err, fmt.Sprintf("SyncFrom(generated partial-overlap conflict pass %d)", pass))
		assertSyncQuarantineBytes(t, local, peer, objects[tc.conflictTarget], variant)
		if after := snapshotDurableStore(t, peer.root); !reflect.DeepEqual(after, peerBytesBefore) {
			t.Fatalf("case %d pass %d mutated the peer while refusing conflict", tc.index, pass)
		}
	}
}

func seedSyncPropertyConflictMask(t *testing.T, store *DurableIndex, objects []syncPropertyObject, mask int, order []int, replay bool, conflictTarget int, variant []byte, caseIndex int) {
	t.Helper()
	seeded := make([][]byte, 0, len(objects))
	for _, index := range order {
		if mask&(1<<index) == 0 {
			continue
		}
		data := objects[index].data
		if index == conflictTarget {
			data = variant
		}
		if err := store.AddJSON(data); err != nil {
			t.Fatalf("case %d seed generated peer %s/%s: %v", caseIndex, objects[index].namespace, objects[index].id, err)
		}
		seeded = append(seeded, data)
	}
	if replay {
		for _, data := range seeded {
			if err := store.AddJSON(data); err != nil {
				t.Fatalf("case %d duplicate generated peer delivery: %v", caseIndex, err)
			}
		}
	}
}

func assertSyncQuarantineBytes(t *testing.T, local, peer *DurableIndex, object syncPropertyObject, variant []byte) {
	t.Helper()
	peerBytes, err := peer.Object(object.namespace, object.id)
	if err != nil || !bytes.Equal(peerBytes, variant) {
		t.Fatalf("peer Object(%s/%s) = %q/%v, want conflicting bytes %q", object.namespace, object.id, peerBytes, err, variant)
	}
	local.index.mu.RLock()
	candidates := make([][]byte, len(local.index.quarantine[object.id]))
	for index, candidate := range local.index.quarantine[object.id] {
		candidates[index] = bytes.Clone(candidate)
	}
	local.index.mu.RUnlock()
	hasOriginal, hasVariant := false, false
	for _, candidate := range candidates {
		hasOriginal = hasOriginal || bytes.Equal(candidate, object.data)
		hasVariant = hasVariant || bytes.Equal(candidate, variant)
	}
	if !hasOriginal || !hasVariant {
		t.Fatalf("local quarantine for %s has original=%t variant=%t (%d candidates)", object.id, hasOriginal, hasVariant, len(candidates))
	}
	if _, err := local.Object(object.namespace, object.id); !hasLiteralSyncConflict(err, "integrity_failure") {
		t.Fatalf("local Object(quarantined %s) = %v, want literal integrity_failure", object.id, err)
	}
}

func TestDurableSyncArrivalOrderConverges(t *testing.T) {
	repo, _ := openSyncProjectionFixture(t)
	objects := generatedSyncPropertyObjects(t, 5, 0)
	for orderIndex, order := range allPermutations(len(objects)) {
		root := filepath.Join(t.TempDir(), fmt.Sprintf("arrival-%03d", orderIndex))
		local, err := OpenDurable(filepath.Join(root, "local"))
		if err != nil {
			t.Fatal(err)
		}
		peer, err := OpenDurable(filepath.Join(root, "peer"))
		if err != nil {
			t.Fatal(err)
		}
		seedSyncPropertyMask(t, peer, objects, (1<<len(objects))-1, order, false, orderIndex, "arrival peer")
		if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
			t.Fatalf("order %d SyncFrom: %v", orderIndex, err)
		}
		assertSyncRootsMatchReference(t, local, objects, orderIndex, 1)
		projection, err := local.RebuildProjection(repo, testSessionID)
		if err != nil {
			t.Fatalf("order %d RebuildProjection: %v", orderIndex, err)
		}
		assertSyncProjectionReference(t, projection, objects, orderIndex)
	}
}

func TestDurableSyncGapFillsOnLaterPass(t *testing.T) {
	repo, objects := openSyncProjectionFixture(t)
	root := filepath.Join(t.TempDir(), "gap")
	local, err := OpenDurable(filepath.Join(root, "local"))
	if err != nil {
		t.Fatal(err)
	}
	peer, err := OpenDurable(filepath.Join(root, "peer"))
	if err != nil {
		t.Fatal(err)
	}
	order := []int{5, 4, 3, 2, 1, 0}
	withoutRecord := (1 << len(objects)) - 2
	seedSyncPropertyMask(t, local, objects, withoutRecord, order, false, 0, "gap local")
	if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
		t.Fatalf("first SyncFrom with missing record: %v", err)
	}
	assertSyncRootsMatchReference(t, local, objectsForMask(objects, withoutRecord), 0, 1)
	addSyncPropertyObject(t, peer, objects[0], false, 0, "gap source fill")
	if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
		t.Fatalf("second SyncFrom after gap fill: %v", err)
	}
	assertSyncRootsMatchReference(t, local, objects, 0, 2)
	projection, err := local.RebuildProjection(repo, testSessionID)
	if err != nil {
		t.Fatalf("RebuildProjection after gap fill: %v", err)
	}
	assertSyncProjectionReference(t, projection, objects, 0)
}

func TestDurableSyncClockSkewDoesNotSelectLeaseWinner(t *testing.T) {
	repo, _ := openSyncProjectionFixture(t)
	for skew := 0; skew < 2; skew++ {
		objects := generatedSyncPropertyObjects(t, 3, skew)
		root := filepath.Join(t.TempDir(), fmt.Sprintf("skew-%d", skew))
		local, err := OpenDurable(filepath.Join(root, "local"))
		if err != nil {
			t.Fatal(err)
		}
		peer, err := OpenDurable(filepath.Join(root, "peer"))
		if err != nil {
			t.Fatal(err)
		}
		order := []int{2, 0, 1}
		seedSyncPropertyMask(t, peer, objects, (1<<len(objects))-1, order, false, skew, "clock peer")
		if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
			t.Fatalf("skew %d SyncFrom: %v", skew, err)
		}
		projection, err := local.RebuildProjection(repo, testSessionID)
		if err != nil {
			t.Fatalf("skew %d RebuildProjection: %v", skew, err)
		}
		assertSyncProjectionReference(t, projection, objects, skew)
		if projection.Winner != (sessstate.LeaseHead{Epoch: 1, LeaseID: "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"}) {
			t.Fatalf("skew %d winner = %+v, want literal §5.3 tuple winner", skew, projection.Winner)
		}
	}
}

func TestDurableSyncPartialPeerNeverRegressesLocalRoots(t *testing.T) {
	_, objects := openSyncProjectionFixture(t)
	root := filepath.Join(t.TempDir(), "partial")
	local, err := OpenDurable(filepath.Join(root, "local"))
	if err != nil {
		t.Fatal(err)
	}
	peer, err := OpenDurable(filepath.Join(root, "peer"))
	if err != nil {
		t.Fatal(err)
	}
	order := []int{0, 1, 2, 3, 4, 5}
	seedSyncPropertyMask(t, local, objects, (1<<len(objects))-1, order, false, 0, "partial local")
	seedSyncPropertyMask(t, peer, objects, 0b000101, order, false, 0, "partial peer")
	before := snapshotDurableStore(t, local.root)
	if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
		t.Fatalf("SyncFrom(partial peer) = %v, want valid union", err)
	}
	assertSyncRootsMatchReference(t, local, objects, 0, 1)
	if after := snapshotDurableStore(t, local.root); !reflect.DeepEqual(after, before) {
		t.Fatalf("partial peer changed local durable bytes when it supplied no new identities")
	}
}

func TestDurableSyncDuplicateDeliveryIsByteIdentical(t *testing.T) {
	_, objects := openSyncProjectionFixture(t)
	root := filepath.Join(t.TempDir(), "duplicates")
	local, err := OpenDurable(filepath.Join(root, "local"))
	if err != nil {
		t.Fatal(err)
	}
	peer, err := OpenDurable(filepath.Join(root, "peer"))
	if err != nil {
		t.Fatal(err)
	}
	order := []int{5, 2, 0, 4, 1, 3}
	fullMask := (1 << len(objects)) - 1
	seedSyncPropertyMask(t, local, objects, fullMask, order, true, 0, "duplicate local")
	seedSyncPropertyMask(t, peer, objects, fullMask, order, true, 0, "duplicate peer")
	if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
		t.Fatalf("initial SyncFrom: %v", err)
	}
	stable := snapshotDurableStore(t, local.root)
	if err := local.SyncFrom(peer, testRequestIDs()); err != nil {
		t.Fatalf("duplicate SyncFrom: %v", err)
	}
	if after := snapshotDurableStore(t, local.root); !reflect.DeepEqual(after, stable) {
		t.Fatalf("identical redelivery changed durable bytes")
	}
	for _, object := range objects {
		if err := local.AddJSON(object.data); err != nil {
			t.Fatalf("AddJSON(replay %s/%s): %v", object.namespace, object.id, err)
		}
	}
	if after := snapshotDurableStore(t, local.root); !reflect.DeepEqual(after, stable) {
		t.Fatalf("identical AddJSON replay changed durable bytes")
	}
}

func TestDurableSyncConflictWithPartialOverlapPeer(t *testing.T) {
	t.Run("review_reproduction", func(t *testing.T) {
		objects := generatedSyncPropertyObjects(t, 2, 0)
		original := objects[0]
		variant := append(bytes.Clone(original.data), '\n')
		local, err := OpenDurable(filepath.Join(t.TempDir(), "local"))
		if err != nil {
			t.Fatal(err)
		}
		peer, err := OpenDurable(filepath.Join(t.TempDir(), "peer"))
		if err != nil {
			t.Fatal(err)
		}
		if err := local.AddJSON(original.data); err != nil {
			t.Fatal(err)
		}
		for _, data := range [][]byte{variant, objects[1].data, variant, objects[1].data} {
			if err := peer.AddJSON(data); err != nil {
				t.Fatalf("seed partial-overlap peer: %v", err)
			}
		}
		peerBefore := snapshotDurableStore(t, peer.root)
		err = local.SyncFrom(peer, testRequestIDs())
		assertSyncLiteralConflict(t, err, "SyncFrom(partial-overlap peer with same identity and peer-only record)")
		assertSyncQuarantineBytes(t, local, peer, original, variant)
		if after := snapshotDurableStore(t, peer.root); !reflect.DeepEqual(after, peerBefore) {
			t.Fatal("conflict refusal mutated the peer's durable bytes")
		}
		if _, err := local.Object(objects[1].namespace, objects[1].id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("local Object(peer-only identity) = %v, want not_found after abort", err)
		}
		if got, err := peer.Object(objects[1].namespace, objects[1].id); err != nil || !bytes.Equal(got, objects[1].data) {
			t.Fatalf("peer Object(peer-only identity) = %q/%v, want exact peer-only bytes", got, err)
		}
	})

	t.Run("after_successful_sync", func(t *testing.T) {
		for skew := 0; skew < 2; skew++ {
			objects := generatedSyncPropertyObjects(t, 3, skew)
			original := objects[0]
			variant := append(bytes.Clone(original.data), '\n')
			local, err := OpenDurable(filepath.Join(t.TempDir(), fmt.Sprintf("local-%d", skew)))
			if err != nil {
				t.Fatal(err)
			}
			if err := local.AddJSON(original.data); err != nil {
				t.Fatal(err)
			}
			firstPeer, err := OpenDurable(filepath.Join(t.TempDir(), fmt.Sprintf("first-peer-%d", skew)))
			if err != nil {
				t.Fatal(err)
			}
			for _, data := range [][]byte{original.data, objects[1].data, original.data, objects[1].data} {
				if err := firstPeer.AddJSON(data); err != nil {
					t.Fatalf("seed first peer: %v", err)
				}
			}
			if err := local.SyncFrom(firstPeer, testRequestIDs()); err != nil {
				t.Fatalf("first SyncFrom before conflict: %v", err)
			}
			secondPeer, err := OpenDurable(filepath.Join(t.TempDir(), fmt.Sprintf("second-peer-%d", skew)))
			if err != nil {
				t.Fatal(err)
			}
			for _, data := range [][]byte{variant, objects[1].data, objects[2].data, variant, objects[1].data, objects[2].data} {
				if err := secondPeer.AddJSON(data); err != nil {
					t.Fatalf("seed second peer: %v", err)
				}
			}
			peerBefore := snapshotDurableStore(t, secondPeer.root)
			assertSyncLiteralConflict(t, local.SyncFrom(secondPeer, testRequestIDs()), "SyncFrom(partial-overlap peer on later sync round)")
			assertSyncQuarantineBytes(t, local, secondPeer, original, variant)
			if _, err := local.Object(objects[2].namespace, objects[2].id); !errors.Is(err, ErrNotFound) {
				t.Fatalf("local Object(late peer-only identity) = %v, want not_found after conflict abort", err)
			}
			assertSyncLiteralConflict(t, local.SyncFrom(secondPeer, testRequestIDs()), "SyncFrom(repeated quarantined conflict)")
			assertSyncQuarantineBytes(t, local, secondPeer, original, variant)
			if after := snapshotDurableStore(t, secondPeer.root); !reflect.DeepEqual(after, peerBefore) {
				t.Fatal("repeated conflict refusal mutated the peer's durable bytes")
			}
		}
	})
}

func TestDurableSyncCommonIDAuditIsUnconditionalInAST(t *testing.T) {
	fset := token.NewFileSet()
	sourcePath := os.Getenv("MERKLE_SYNC_AUDIT_SOURCE")
	if sourcePath == "" {
		sourcePath = "durable.go"
	}
	file, err := parser.ParseFile(fset, sourcePath, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", sourcePath, err)
	}
	var syncBody *ast.BlockStmt
	ast.Inspect(file, func(node ast.Node) bool {
		decl, ok := node.(*ast.FuncDecl)
		if ok && decl.Name.Name == "SyncFrom" && decl.Recv != nil && len(decl.Recv.List) == 1 {
			syncBody = decl.Body
		}
		return true
	})
	if syncBody == nil {
		t.Fatal("DurableIndex.SyncFrom production entry not found")
	}
	var commonCalls []*ast.CallExpr
	ast.Inspect(syncBody, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && isCommonIDFetch(call) {
			commonCalls = append(commonCalls, call)
		}
		return true
	})
	if len(commonCalls) != 1 {
		t.Fatalf("DurableIndex.SyncFrom has %d direct common-ID fetch calls, want exactly one unconditional call", len(commonCalls))
	}
	commonCall := commonCalls[0]
	var guards []string
	var namespaceRange bool
	ast.Inspect(syncBody, func(node ast.Node) bool {
		switch statement := node.(type) {
		case *ast.IfStmt:
			if astNodeContains(statement.Body, commonCall.Pos()) || astNodeContains(statement.Else, commonCall.Pos()) {
				guards = append(guards, fmt.Sprintf("conditional at %d", fset.Position(statement.If).Line))
			}
			if statement.Pos() < commonCall.Pos() && statement.End() <= commonCall.Pos() && containsControlExit(statement) && containsSyncAuditGateIdentifier(statement.Cond) {
				guards = append(guards, fmt.Sprintf("conditional early exit at %d", fset.Position(statement.If).Line))
			}
		case *ast.RangeStmt:
			key, keyOK := statement.Key.(*ast.Ident)
			value, valueOK := statement.Value.(*ast.Ident)
			rangeSource, sourceOK := statement.X.(*ast.Ident)
			if astNodeContains(statement.Body, commonCall.Pos()) && keyOK && key.Name == "_" && valueOK && value.Name == "namespace" && sourceOK && rangeSource.Name == "durableJSONNamespaces" {
				namespaceRange = true
			}
		}
		return true
	})
	if len(guards) != 0 {
		t.Fatalf("common-ID byte audit is gated by %v; every namespace must audit common IDs independent of peer size, missing set, or namespace position", guards)
	}
	if !namespaceRange {
		t.Fatal("common-ID byte audit is not inside the full durableJSONNamespaces iteration")
	}
}

func isCommonIDFetch(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "fetchAndAdd" || len(call.Args) != 4 {
		return false
	}
	common, ok := call.Args[2].(*ast.Ident)
	return ok && common.Name == "common"
}

func astNodeContains(node ast.Node, pos token.Pos) bool {
	return node != nil && node.Pos() <= pos && pos <= node.End()
}

func containsControlExit(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		switch exit := child.(type) {
		case *ast.ReturnStmt:
			found = true
		case *ast.BranchStmt:
			if exit.Tok == token.CONTINUE || exit.Tok == token.BREAK {
				found = true
			}
		}
		if found {
			return false
		}
		return true
	})
	return found
}

func containsSyncAuditGateIdentifier(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		identifier, ok := child.(*ast.Ident)
		if !ok {
			return !found
		}
		if identifier.Name == "missing" || identifier.Name == "peerIDs" || identifier.Name == "index" || strings.HasPrefix(identifier.Name, "namespace") {
			found = true
			return false
		}
		return true
	})
	return found
}

func TestDurableSyncSameDigestDifferentBytesIsTypedConflict(t *testing.T) {
	original := validSessionRecordJSON(t)
	variant := append(bytes.Clone(original), '\n')
	if referenceJSONField(t, original, "record_id") != referenceJSONField(t, variant, "record_id") || bytes.Equal(original, variant) {
		t.Fatal("conflict fixture must retain one identity while changing raw bytes")
	}
	root := filepath.Join(t.TempDir(), "same-digest")
	local, err := OpenDurable(filepath.Join(root, "local"))
	if err != nil {
		t.Fatal(err)
	}
	peer, err := OpenDurable(filepath.Join(root, "peer"))
	if err != nil {
		t.Fatal(err)
	}
	if err := local.AddJSON(original); err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(variant); err != nil {
		t.Fatal(err)
	}
	err = local.SyncFrom(peer, testRequestIDs())
	assertSyncLiteralConflict(t, err, "SyncFrom(same digest, different bytes)")
	id := referenceJSONField(t, original, "record_id")
	if !slices.Contains(local.index.QuarantinedIDs(), id) {
		t.Fatalf("quarantined IDs = %q, want %s", local.index.QuarantinedIDs(), id)
	}
	if _, err := local.Object("record", id); !hasLiteralSyncConflict(err, "integrity_failure") {
		t.Fatalf("Object(quarantined identity) = %v, want literal integrity_failure", err)
	}
	reopened, err := OpenDurable(filepath.Join(root, "local"))
	if err != nil {
		t.Fatalf("OpenDurable after conflict: %v", err)
	}
	if !slices.Contains(reopened.index.QuarantinedIDs(), id) {
		t.Fatalf("reopened quarantine = %q, want %s", reopened.index.QuarantinedIDs(), id)
	}
	rootAfter, err := reopened.Root("record")
	if err != nil || rootAfter.Count.Uint64() != 0 {
		t.Fatalf("record root after conflict = %#v/%v, want empty active root", rootAfter, err)
	}
}

func TestDurableSyncRefusesPeerNamespaceMismatch(t *testing.T) {
	_, fixtures := inventoryObjectsByNamespace(t)
	peer, err := OpenDurable(filepath.Join(t.TempDir(), "peer"))
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(fixtures["event"]); err != nil {
		t.Fatal(err)
	}
	if err := peer.AddJSON(fixtures["manifest"]); err != nil {
		t.Fatal(err)
	}
	eventID := objectID(t, fixtures["event"])
	peer.index.mu.Lock()
	corrupt := peer.index.objects["event"][eventID]
	corrupt.data = bytes.Clone(fixtures["manifest"])
	peer.index.objects["event"][eventID] = corrupt
	peer.index.mu.Unlock()

	local, err := OpenDurable(filepath.Join(t.TempDir(), "local"))
	if err != nil {
		t.Fatal(err)
	}
	err = local.SyncFrom(peer, testRequestIDs())
	assertSyncLiteralConflict(t, err, "SyncFrom(peer record identity returned as a manifest)")
	for _, namespace := range durableJSONNamespaces {
		root, rootErr := local.Root(namespace)
		if rootErr != nil || root.Count.Uint64() != 0 {
			t.Errorf("local %s root after peer namespace conflict = %#v/%v, want empty", namespace, root, rootErr)
		}
	}
	if got := local.index.QuarantinedIDs(); len(got) != 0 {
		t.Fatalf("local quarantine after rejected peer namespace = %q, want empty", got)
	}
}

func TestDurableSyncUnclosedAcknowledgementAborts(t *testing.T) {
	_, objects := openSyncProjectionFixture(t)
	root := filepath.Join(t.TempDir(), "unclosed-ack")
	local, err := OpenDurable(filepath.Join(root, "local"))
	if err != nil {
		t.Fatal(err)
	}
	peer, err := OpenDurable(filepath.Join(root, "peer"))
	if err != nil {
		t.Fatal(err)
	}
	// The sixth generated identity is the acknowledgement. Withhold its
	// referenced Tombstone to drive the closure refusal through SyncFrom.
	if err := local.AddJSON(objects[0].data); err != nil {
		t.Fatal(err)
	}
	if err := local.AddJSON(objects[5].data); err != nil {
		t.Fatal(err)
	}
	assertSyncLiteralConflict(t, local.SyncFrom(peer, testRequestIDs()), "SyncFrom(unclosed Tombstone Acknowledgement)")
}

func openSyncProjectionFixture(t *testing.T) (*sessrepo.Repository, []syncPropertyObject) {
	t.Helper()
	record := validSessionRecordJSON(t)
	repo, err := sessrepo.Open(filepath.Join(t.TempDir(), "authoritative"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateSession(record); err != nil {
		t.Fatal(err)
	}
	return repo, generatedSyncPropertyObjects(t, 6, 0)
}

func generatedSyncPropertyObjects(t *testing.T, size, skew int) []syncPropertyObject {
	t.Helper()
	if size < 1 || size > 6 || (skew != 0 && skew != 1) {
		t.Fatalf("generated sync fixture size/skew = %d/%d, outside test generator domain", size, skew)
	}
	early, late := "1900-01-01T00:00:00.000Z", "2099-12-31T23:59:59.999Z"
	leaseATime, leaseBTime := early, late
	if skew == 1 {
		leaseATime, leaseBTime = late, early
	}
	record := validSessionRecordJSON(t)
	recordID := referenceJSONField(t, record, "record_id")
	leaseAID, leaseBID := "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	leaseA := unionLeaseJSON(t, leaseAID, testHostID, leaseATime)
	leaseB := unionLeaseJSON(t, leaseBID, "0198f4c8-4a10-7b22-8b3c-2234567890ab", leaseBTime)
	branchEvent := unionEventJSON(t, recordID, leaseAID, "session.idle", leaseATime, map[string]any{
		"boundary_ref": "generated-losing-lease-branch", "foreground_idle": true, "background_idle": true,
	})
	tombstoneTime, ackTime := leaseBTime, leaseATime
	tombstone := tombstoneJSON(t, tombstoneTime)
	tombstoneID := referenceJSONField(t, tombstone, "tombstone_id")
	acknowledgement := tombstoneAckJSON(t, tombstoneID, "applied", "", ackTime)
	all := []syncPropertyObject{
		{namespace: "record", id: recordID, data: record},
		{namespace: "record", id: referenceJSONField(t, leaseA, "record_id"), data: leaseA, lease: sessstate.LeaseHead{Epoch: 1, LeaseID: leaseAID}},
		{namespace: "record", id: referenceJSONField(t, leaseB, "record_id"), data: leaseB, lease: sessstate.LeaseHead{Epoch: 1, LeaseID: leaseBID}},
		{namespace: "event", id: referenceJSONField(t, branchEvent, "event_id"), data: branchEvent},
		{namespace: "tombstone", id: tombstoneID, data: tombstone},
		{namespace: "tombstone_ack", id: referenceJSONField(t, acknowledgement, "ack_id"), data: acknowledgement},
	}
	return all[:size]
}

func syncPropertyOrders(size int) [][]int {
	if size <= 5 {
		return allPermutations(size)
	}
	rng := rand.New(rand.NewSource(2608302))
	seen := map[string]bool{}
	orders := make([][]int, 0, 34)
	appendOrder := func(order []int) {
		key := fmt.Sprint(order)
		if seen[key] {
			return
		}
		seen[key] = true
		orders = append(orders, append([]int(nil), order...))
	}
	appendOrder([]int{0, 1, 2, 3, 4, 5})
	appendOrder([]int{5, 4, 3, 2, 1, 0})
	for len(orders) < 34 {
		appendOrder(rng.Perm(size))
	}
	return orders
}

func syncPropertyConflictTarget(size, gapTarget, scenario int) int {
	target := scenario % size
	if target == gapTarget {
		target = (target + 1) % size
	}
	return target
}

func syncPropertyConflictFeasible(size, target, gapTarget, conflictRound int) bool {
	if size < 2 || target < 0 || target >= size || target == gapTarget || conflictRound < 1 || conflictRound > 3 {
		return false
	}
	if size == 6 && target == 5 && gapTarget == 4 {
		return false // The delayed Tombstone cannot close a locally present Ack before this conflict.
	}
	for identity := 0; identity < size; identity++ {
		if identity != target && identity != gapTarget {
			return true
		}
	}
	return gapTarget >= 0 && conflictRound == 2
}

func syncPropertyPeerMasks(size, gapTarget int) []int {
	limit := (1 << size) - 1 // The full object set is not a partial peer.
	if gapTarget >= 0 {
		// Enumerate every peer subset of the non-gap identities.
		masks := make([]int, 0, 1<<(size-1))
		for subset := 0; subset < 1<<(size-1); subset++ {
			mask, sourceBit := 0, 0
			for identity := 0; identity < size; identity++ {
				if identity == gapTarget {
					continue
				}
				if subset&(1<<sourceBit) != 0 {
					mask |= 1 << identity
				}
				sourceBit++
			}
			masks = append(masks, mask)
		}
		return masks
	}
	masks := make([]int, 0, limit)
	for mask := 0; mask < limit; mask++ {
		masks = append(masks, mask)
	}
	return masks
}

func reverseSyncPropertyOrder(order []int) []int {
	reversed := append([]int(nil), order...)
	slices.Reverse(reversed)
	return reversed
}

func seedSyncPropertyMask(t *testing.T, store *DurableIndex, objects []syncPropertyObject, mask int, order []int, replay bool, caseIndex int, label string) {
	t.Helper()
	for _, index := range order {
		if mask&(1<<index) == 0 {
			continue
		}
		addSyncPropertyObject(t, store, objects[index], replay, caseIndex, label)
	}
}

func addSyncPropertyObject(t *testing.T, store *DurableIndex, object syncPropertyObject, replay bool, caseIndex int, label string) {
	t.Helper()
	if err := store.AddJSON(object.data); err != nil {
		t.Fatalf("case %d %s AddJSON(%s/%s): %v", caseIndex, label, object.namespace, object.id, err)
	}
	if replay {
		if err := store.AddJSON(object.data); err != nil {
			t.Fatalf("case %d %s AddJSON(duplicate %s/%s): %v", caseIndex, label, object.namespace, object.id, err)
		}
	}
}

func objectsForMask(objects []syncPropertyObject, mask int) []syncPropertyObject {
	selected := make([]syncPropertyObject, 0, len(objects))
	for index, object := range objects {
		if mask&(1<<index) != 0 {
			selected = append(selected, object)
		}
	}
	return selected
}

func assertSyncRootsMatchReference(t *testing.T, store *DurableIndex, objects []syncPropertyObject, caseIndex, pass int) {
	t.Helper()
	for _, namespace := range syncReferenceNamespaces {
		ids := make([]string, 0)
		for _, object := range objects {
			if object.namespace == namespace {
				ids = append(ids, object.id)
			}
		}
		wantCount, wantRoot := referenceMerkleRoot(namespace, ids)
		got, err := store.Root(namespace)
		if err != nil || got.Count.Uint64() != uint64(wantCount) || got.RootID.String() != wantRoot {
			t.Fatalf("case %d pass %d root %s = %#v/%v, want independent count/root %d/%s", caseIndex, pass, namespace, got, err, wantCount, wantRoot)
		}
	}
}

func referenceMerkleRoot(namespace string, ids []string) (int, string) {
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)
	return len(ordered), referenceNodeHash(namespace, "", ordered)
}

func referenceNodeHash(namespace, prefix string, ids []string) string {
	ordered := append([]string(nil), ids...)
	sort.Strings(ordered)
	type child struct {
		label string
		count int
		hash  string
	}
	children := make([]child, 0)
	leafIDs := make([]string, 0, 1)
	if len(ordered) == 1 {
		leafIDs = ordered
	} else if len(ordered) > 1 {
		if len(prefix) >= 64 {
			panic("independent Merkle oracle received duplicate full-width digest identities")
		}
		groups := map[byte][]string{}
		for _, id := range ordered {
			hexID := strings.TrimPrefix(id, "sha256:")
			groups[hexID[len(prefix)]] = append(groups[hexID[len(prefix)]], id)
		}
		labels := make([]int, 0, len(groups))
		for label := range groups {
			labels = append(labels, int(label))
		}
		sort.Ints(labels)
		for _, rawLabel := range labels {
			label := byte(rawLabel)
			members := groups[label]
			children = append(children, child{
				label: string(label), count: len(members),
				hash: referenceNodeHash(namespace, prefix+string(label), members),
			})
		}
	}
	var body strings.Builder
	body.WriteString(`{"children":[`)
	for index, item := range children {
		if index > 0 {
			body.WriteByte(',')
		}
		body.WriteString(`{"count":`)
		fmt.Fprint(&body, item.count)
		body.WriteString(`,"hash":"`)
		body.WriteString(item.hash)
		body.WriteString(`","label":"`)
		body.WriteString(item.label)
		body.WriteString(`"}`)
	}
	body.WriteString(`],"count":`)
	fmt.Fprint(&body, len(ordered))
	body.WriteString(`,"domain":"urn:ax:merkle-node:1","ids":[`)
	for index, id := range leafIDs {
		if index > 0 {
			body.WriteByte(',')
		}
		body.WriteByte('"')
		body.WriteString(id)
		body.WriteByte('"')
	}
	body.WriteString(`],"namespace":"`)
	body.WriteString(namespace)
	body.WriteString(`","prefix":"`)
	body.WriteString(prefix)
	body.WriteString(`"}`)
	sum := sha256.Sum256([]byte(body.String()))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func assertSyncProjectionReference(t *testing.T, got sessstate.Projection, objects []syncPropertyObject, caseIndex int) {
	t.Helper()
	var expectedRecordID string
	leases := make([]sessstate.LeaseHead, 0, 2)
	for _, object := range objects {
		if object.namespace == "record" && object.lease.LeaseID == "" {
			expectedRecordID = object.id
		}
		if object.lease.LeaseID != "" {
			leases = append(leases, object.lease)
		}
	}
	if expectedRecordID == "" {
		t.Fatalf("case %d independent projection has no Session Record", caseIndex)
	}
	wantWinner := sessstate.LeaseHead{}
	for _, lease := range leases {
		if syncReferenceLeaseCompare(lease, wantWinner) > 0 {
			wantWinner = lease
		}
	}
	wantKinds, wantRules := make([]string, 0), make([]string, 0)
	hasWinner := wantWinner.LeaseID != ""
	if hasWinner {
		losers := append([]sessstate.LeaseHead(nil), leases...)
		sort.Slice(losers, func(i, j int) bool { return syncReferenceLeaseCompare(losers[i], losers[j]) < 0 })
		for _, lease := range losers {
			if syncReferenceLeaseCompare(lease, wantWinner) < 0 {
				wantKinds = append(wantKinds, "losing_branch_preserved")
				wantRules = append(wantRules, "preserved-never-applied")
			}
		}
		byEpoch := map[uint64]map[string]bool{}
		for _, lease := range leases {
			if byEpoch[lease.Epoch] == nil {
				byEpoch[lease.Epoch] = map[string]bool{}
			}
			byEpoch[lease.Epoch][lease.LeaseID] = true
		}
		epochs := make([]uint64, 0, len(byEpoch))
		for epoch := range byEpoch {
			epochs = append(epochs, epoch)
		}
		slices.Sort(epochs)
		for _, epoch := range epochs {
			if len(byEpoch[epoch]) > 1 {
				wantKinds = append(wantKinds, "same_epoch_tie")
				wantRules = append(wantRules, "greatest-lease-id-wins")
			}
		}
		wantKinds = append(wantKinds, "union_supersedes_chain")
		wantRules = append(wantRules, "authoritative-state-unchanged")
	}
	gotKinds, gotRules := make([]string, 0, len(got.Conflicts)), make([]string, 0, len(got.Conflicts))
	for _, conflict := range got.Conflicts {
		gotKinds = append(gotKinds, conflict.Kind)
		gotRules = append(gotRules, conflict.Rule)
	}
	wantWarnings := []string(nil)
	if hasWinner {
		wantWarnings = []string{"divergent_history"}
	}
	if got.SessionID != testSessionID || got.RecordID != expectedRecordID || got.Name != "payments-api" || got.Kind != "direct" ||
		got.State != sessstate.State("creating") || got.Winner != wantWinner || !slices.Equal(gotKinds, wantKinds) || !slices.Equal(gotRules, wantRules) ||
		got.Provider.ID != "codex" || got.HasCheckpoint || got.HasTerminal || got.OwnerHostID != "" || got.LocalRole != "" || got.Parked != nil ||
		!slices.Equal(got.Warnings, wantWarnings) {
		t.Fatalf("case %d projection = %+v; independent §5.3 / record-only oracle expects winner=%+v conflicts=%v rules=%v state=creating",
			caseIndex, got, wantWinner, wantKinds, wantRules)
	}
}

func syncReferenceLeaseCompare(left, right sessstate.LeaseHead) int {
	if left.Epoch < right.Epoch {
		return -1
	}
	if left.Epoch > right.Epoch {
		return 1
	}
	return strings.Compare(left.LeaseID, right.LeaseID)
}

func assertSyncObjectsMatchReference(t *testing.T, store *DurableIndex, objects []syncPropertyObject, caseIndex int) {
	t.Helper()
	for _, object := range objects {
		got, err := store.Object(object.namespace, object.id)
		if err != nil || !bytes.Equal(got, object.data) {
			t.Fatalf("case %d Object(%s/%s) = %q/%v; want exact fixture bytes", caseIndex, object.namespace, object.id, got, err)
		}
	}
}

func syncStoreIDs(store *DurableIndex) map[string][]string {
	ids := make(map[string][]string, len(syncReferenceNamespaces))
	for _, namespace := range syncReferenceNamespaces {
		ids[namespace] = store.index.objectIDs(namespace)
	}
	return ids
}

func assertSyncIDsMonotone(t *testing.T, before, after map[string][]string, caseIndex, pass int) {
	t.Helper()
	for namespace, oldIDs := range before {
		newIDs := after[namespace]
		for _, oldID := range oldIDs {
			if !slices.Contains(newIDs, oldID) {
				t.Fatalf("case %d pass %d partial peer removed existing %s identity %s", caseIndex, pass, namespace, oldID)
			}
		}
		beforeCount, _ := referenceMerkleRoot(namespace, oldIDs)
		afterCount, _ := referenceMerkleRoot(namespace, newIDs)
		if afterCount < beforeCount {
			t.Fatalf("case %d pass %d root count for %s regressed %d→%d", caseIndex, pass, namespace, beforeCount, afterCount)
		}
	}
}

func snapshotDurableStore(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[relative] = data
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot durable store %s: %v", root, err)
	}
	return files
}

func referenceJSONField(t *testing.T, data []byte, name string) string {
	t.Helper()
	var members map[string]json.RawMessage
	if err := json.Unmarshal(data, &members); err != nil {
		t.Fatalf("independent fixture read: %v", err)
	}
	raw, ok := members[name]
	if !ok {
		t.Fatalf("independent fixture field %q missing", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || value == "" {
		t.Fatalf("independent fixture field %q = %s/%v, want string", name, raw, err)
	}
	return value
}

func assertSyncLiteralConflict(t *testing.T, err error, entry string) {
	t.Helper()
	if !hasLiteralSyncConflict(err, "integrity_failure") {
		t.Fatalf("%s error = %v, want typed literal integrity_failure", entry, err)
	}
}

func hasLiteralSyncConflict(err error, code string) bool {
	var refusalErr *Refusal
	return errors.As(err, &refusalErr) && refusalErr.Code == code
}
