#!/bin/bash
# Reviewer mutation harness. Same two-gate shape as the producer's, plus a
# real negative control: BEH = go test -run '.*', FULL = go test.
set -u
PKG=./internal/sessstate
BK=.temp/TASK-260830-1r9wrr/review/bk
run_one() {
  local ID="$1" FILE="$2" OLD="$3" NEW="$4"
  local F="internal/sessstate/$FILE"
  cp "$BK/$FILE" "$F"
  local COUNT
  COUNT=$(python3 -c "import sys; s=open('$F').read(); sys.stdout.write(str(s.count(sys.argv[1])))" "$OLD")
  if [ "$COUNT" != "1" ]; then echo "$ID: NOT_APPLIED (pattern count=$COUNT)"; cp "$BK/$FILE" "$F"; return; fi
  python3 -c "import sys; p='$F'; s=open(p).read(); s=s.replace(sys.argv[1], sys.argv[2], 1); open(p,'w').write(s)" "$OLD" "$NEW"
  if ! go build "$PKG" >/dev/null 2>&1; then echo "$ID: COMPILE_FAIL"; cp "$BK/$FILE" "$F"; return; fi
  local BEH FULL
  if go test "$PKG" -count=1 -run '.*' >"/tmp/rev_beh_$ID.log" 2>&1; then BEH=PASS; else BEH=FAIL; fi
  if go test "$PKG" -count=1 >"/tmp/rev_full_$ID.log" 2>&1; then FULL=PASS; else FULL=FAIL; fi
  local K
  if [ "$BEH" = "FAIL" ]; then K="KILLED(behavioral)"
  elif [ "$FULL" = "FAIL" ]; then K="KILLED(audit/census-only)"
  else K="SURVIVED"; fi
  echo "$ID: $K  behavioral=$BEH full=$FULL"
  [ "$BEH" = "FAIL" ] && grep -E "^\s*--- FAIL" "/tmp/rev_beh_$ID.log" | head -4
  [ "$BEH" = "PASS" ] && [ "$FULL" = "FAIL" ] && grep -E "sessstate |outside direct-call|outside the refuse funnel" "/tmp/rev_full_$ID.log" | head -4
  cp "$BK/$FILE" "$F"
}
echo "== baseline =="
go test "$PKG" -count=1 >/dev/null 2>&1 && echo "baseline green" || { echo "BASELINE RED"; exit 1; }

echo "== R0: NEGATIVE CONTROL (behaviour-neutral, line-count preserving) — must SURVIVE =="
run_one R0 sessstate.go 'func (fold *chainFold) tailEventID() string { return fold.tailID }' 'func (fold *chainFold) tailEventID() string { return "" + fold.tailID }'

echo "== R0b: POSITIVE CONTROL (known-bad, compiles) — must be KILLED behaviorally =="
run_one R0b project.go '	if found.Parked {' '	if found == nil || found.Parked {'

echo "== R1: candidate FIX for the local-lease comparison (does the suite notice?) =="
run_one R1 sessstate.go '	if Compare(local, fold.projection.Winner) == 0 {
		return
	}' '	if Compare(local, fold.projection.Winner) >= 0 {
		return
	}'

echo "== R2: candidate FIX for the duplicate-union case-0 branch (does the suite notice?) =="
run_one R2 sessstate.go '		case 0:
			if _, ok := onChain[lease.LeaseID]; !ok {
				offChainWinner = false
			}' '		case 0:
			_ = onChain'

echo "== R3: narrowing — staleSources admits stopped =="
run_one R3 sessstate.go '	StateRunning: {}, StateIdle: {}, StateQuiescing: {},
	StateFailed: {}, StateParked: {},' '	StateRunning: {}, StateIdle: {}, StateQuiescing: {},
	StateFailed: {}, StateParked: {}, StateStopped: {},'

echo "== R4: narrowing — transition table admits running->stopped =="
run_one R4 sessstate.go '	StateRunning: {
		StateIdle: {}, StateQuiescing: {}, StateFailed: {}, StateStale: {},
	},' '	StateRunning: {
		StateIdle: {}, StateQuiescing: {}, StateFailed: {}, StateStale: {}, StateStopped: {},
	},'

echo "== R5: narrowing — newest checkpoint keeps the OLDEST instead =="
run_one R5 sessstate.go '	fold.seenCheckpoint = true
	fold.projection.HasCheckpoint = true
	fold.projection.Newest = Checkpoint{ID: id, Kind: kind, EventID: eventID}' '	if fold.seenCheckpoint {
		return
	}
	fold.seenCheckpoint = true
	fold.projection.HasCheckpoint = true
	fold.projection.Newest = Checkpoint{ID: id, Kind: kind, EventID: eventID}'

echo "== R6: narrowing — provider version recorded even on record mismatch =="
run_one R6 sessstate.go '	} else {
		fold.projection.Provider.Version = version
	}
	return fold.step(StateRunning, what)' '	}
	fold.projection.Provider.Version = version
	return fold.step(StateRunning, what)'

echo "== R7: narrowing — local role inverted =="
run_one R7 sessstate.go '		if localHostID == fold.projection.OwnerHostID {
			fold.projection.LocalRole = "owner"
		} else {
			fold.projection.LocalRole = "replica"
		}' '		if localHostID != fold.projection.OwnerHostID {
			fold.projection.LocalRole = "owner"
		} else {
			fold.projection.LocalRole = "replica"
		}'

echo "== R8: narrowing — stale_process warning dropped =="
run_one R8 sessstate.go '	if fold.state == StateStale {
		warnings["stale_process"] = struct{}{}
	}' '	if fold.state == StateStale && len(fold.leases) > 99 {
		warnings["stale_process"] = struct{}{}
	}'

echo "== R9: narrowing — epoch-gap conflict only past a jump of 3 =="
run_one R9 sessstate.go '		if lease.Epoch > fold.current.Epoch+1 {
			fold.conflict(ConflictEpochGap' '		if lease.Epoch > fold.current.Epoch+3 {
			fold.conflict(ConflictEpochGap'

echo "== R10: narrowing — losing union branch conflict dropped for epoch 1 rivals =="
run_one R10 sessstate.go '		case -1:
			fold.conflict(ConflictLosingBranchPreserved' '		case -1:
			if lease.Epoch == 1 { break }
			fold.conflict(ConflictLosingBranchPreserved'
#!/bin/bash
set -u
PKG=./internal/sessstate
BK=.temp/TASK-260830-1r9wrr/review/bk
PLANT=internal/sessstate/zz_plant.go
restore() { cp "$BK/sessstate.go" internal/sessstate/sessstate.go; rm -f "$PLANT"; }
run_plant() {
  local ID="$1" BODY="$2" WIRE="${3:-no}"
  restore
  printf '%s' "$BODY" > "$PLANT"
  if [ "$WIRE" = "yes" ]; then
    python3 - <<'PY'
import io
p='internal/sessstate/sessstate.go'
s=open(p).read()
old='''	case "task_board.launched":
		return fold.effectTaskBoardLaunched(event, name)
	default:
		return nil
	}'''
new='''	case "task_board.launched":
		return fold.effectTaskBoardLaunched(event, name)
	default:
		return fold.effectExtra(event, name)
	}'''
assert s.count(old)==1, s.count(old)
open(p,'w').write(s.replace(old,new,1))
PY
  fi
  if ! go build "$PKG" >/dev/null 2>&1; then echo "$ID: COMPILE_FAIL"; go build "$PKG" 2>&1|head -3; restore; return; fi
  local BEH FULL
  if go test "$PKG" -count=1 -run '.*' >"/tmp/plant_beh_$ID.log" 2>&1; then BEH=PASS; else BEH=FAIL; fi
  if go test "$PKG" -count=1 >"/tmp/plant_full_$ID.log" 2>&1; then FULL=PASS; else FULL=FAIL; fi
  if [ "$BEH" = "FAIL" ]; then echo "$ID: KILLED(behavioral) beh=$BEH full=$FULL"
  elif [ "$FULL" = "FAIL" ]; then echo "$ID: KILLED(audit/census-only) beh=$BEH full=$FULL"
  else echo "$ID: SURVIVED beh=$BEH full=$FULL"; fi
  [ "$BEH" = "FAIL" ] && grep -E "^\s*--- FAIL" "/tmp/plant_beh_$ID.log" | head -3
  [ "$BEH" = "PASS" ] && [ "$FULL" = "FAIL" ] && grep -E "sessstate |outside direct-call|zz_plant" "/tmp/plant_full_$ID.log" | head -3
  restore
}

echo "== PA: fresh-spelling handler in a new production file, wired into effect's default =="
echo "     (unknown v1 type 'session.reaped' silently moves running -> idle; Section 5.2 says inert)"
run_plant PA 'package sessstate

// effectExtra is a second event-handling site outside the censused
// effect switch.
func (fold *chainFold) effectExtra(event Event, what string) error {
	switch event.Type {
	case "session.reaped":
		return fold.step(StateIdle, what)
	}
	return nil
}
' yes

echo "== PB: same fresh-spelling handler, but raising a refusal through a var-bound funnel =="
run_plant PB 'package sessstate

var deny = refuse

func (fold *chainFold) effectExtra(event Event, what string) error {
	switch event.Type {
	case "session.reaped":
		return deny(ErrIntegrity, "%s reaped", what)
	}
	return nil
}
' yes

echo "== PC: alias-bound var over a watched owner delegation, in a new production file =="
run_plant PC 'package sessstate

import "github.com/relux-works/agent-session-manager/internal/scalar"

var parseDigest = scalar.ParseDigest

func (fold *chainFold) effectExtra(event Event, what string) error {
	if event.Type == "session.reaped" {
		if _, err := parseDigest(event.ID); err != nil {
			return nil
		}
		return fold.step(StateIdle, what)
	}
	return nil
}
' yes

echo "== PD: new State spelling declared in a new production file =="
run_plant PD 'package sessstate

// StateReaped is an unregistered spelling.
const StateReaped State = "reaped"

func (fold *chainFold) effectExtra(event Event, what string) error {
	if event.Type == "session.reaped" {
		fold.projection.State = StateReaped
	}
	return nil
}
' yes

echo "== PE: direct refuse() call at a new production site (unregistered census row) =="
run_plant PE 'package sessstate

func (fold *chainFold) effectExtra(event Event, what string) error {
	if event.Type == "session.reaped" {
		return refuse(ErrIntegrity, "%s is reaped", what)
	}
	return nil
}
' yes
