#!/bin/bash
# Round-2 fix verification: apply each mutant, confirm presence, run the
# FULL provhost package (the inventory audit only fires unmasked), restore
# from cp backups (never git checkout), and byte-verify the restore.
set -u
PKG=./internal/provhost
BAK=/tmp/rev2bak
LOG=/tmp/rev2logs
mkdir -p "$BAK" "$LOG"
cp "$PKG"/protocol.go "$PKG"/runner.go "$PKG"/status.go "$BAK"/
PRE=$(sha256sum "$PKG"/protocol.go "$PKG"/runner.go "$PKG"/status.go | sha256sum | cut -d' ' -f1)

apply_once() { # file old new
  python3 - "$1" "$2" "$3" <<'EOF'
import sys
path, old, new = sys.argv[1], sys.argv[2], sys.argv[3]
src = open(path).read()
n = src.count(old)
if n != 1:
    print("APPLY_ERROR: %d occurrences in %s" % (n, path)); sys.exit(1)
open(path, "w").write(src.replace(old, new))
print("applied")
EOF
}

run_mutant() { # id file old new
  ID="$1"; F="$2"; OLD="$3"; NEW="$4"
  if ! apply_once "$PKG/$F" "$OLD" "$NEW" > "$LOG/$ID.log" 2>&1; then
    echo "$ID APPLY_FAILED"; cp "$BAK/$F" "$PKG/$F"; return
  fi
  if ! grep -q -F "$NEW" "$PKG/$F"; then
    echo "$ID NOT_PRESENT_AFTER_APPLY"; cp "$BAK/$F" "$PKG/$F"; return
  fi
  go test "$PKG" -count=1 >> "$LOG/$ID.log" 2>&1
  RC=$?
  cp "$BAK/$F" "$PKG/$F"
  if [ $RC -eq 0 ]; then echo "$ID SURVIVED"; else
    echo "$ID KILLED  $(grep -oE '^\s*--- FAIL: [A-Za-z0-9_/]+' "$LOG/$ID.log" | head -4 | tr '\n' ' ')"
  fi
}

TAB=$'\t'
NL=$'\n'

# F21 parseMajor arms
run_mutant M19 protocol.go "${TAB}if len(parts) != 3 {${NL}${TAB}${TAB}return 0, false${NL}${TAB}}" ""
run_mutant M20 protocol.go "${TAB}if len(parts[0]) == 0 {${NL}${TAB}${TAB}return 0, false${NL}${TAB}}" ""
run_mutant M21 protocol.go "${TAB}${TAB}if len(rest) == 0 {${NL}${TAB}${TAB}${TAB}return 0, false${NL}${TAB}${TAB}}" ""
run_mutant M22 protocol.go "${TAB}${TAB}for i := 0; i < len(rest); i++ {${NL}${TAB}${TAB}${TAB}if rest[i] < '0' || rest[i] > '9' {${NL}${TAB}${TAB}${TAB}${TAB}return 0, false${NL}${TAB}${TAB}${TAB}}${NL}${TAB}${TAB}}" ""
# F25 required-member gates
run_mutant M09 protocol.go "${TAB}required := []string{\"protocol\", \"protocol_version\", \"request_id\", \"ok\"}${NL}${TAB}for _, name := range required {${NL}${TAB}${TAB}if _, present := members[name]; !present {${NL}${TAB}${TAB}${TAB}return &frameFault{detail: \"missing member\", member: name}${NL}${TAB}${TAB}}${NL}${TAB}}" ""
run_mutant M09a protocol.go 'required := []string{"protocol", "protocol_version", "request_id", "ok"}' 'required := []string{"ok"}'
run_mutant M09b protocol.go 'required := []string{"protocol", "protocol_version", "request_id", "ok"}' 'required := []string{"protocol", "protocol_version", "request_id"}'
run_mutant S02 status.go "${TAB}for _, name := range []string{\"materialization_id\", \"transaction_id\", \"transaction_authority_id\", \"plan_id\", \"state\", \"rollback_token\", \"native_discovery\"} {${NL}${TAB}${TAB}if _, present := members[name]; !present {${NL}${TAB}${TAB}${TAB}return integrity(\"status body misses a required member\", \"\")${NL}${TAB}${TAB}}${NL}${TAB}}" ""
run_mutant S02a status.go '"materialization_id", "transaction_id", "transaction_authority_id", "plan_id", "state", "rollback_token", "native_discovery"' '"materialization_id"'
run_mutant M27 protocol.go "${TAB}${TAB}if err := decoder.Decode(&raw); err != nil {${NL}${TAB}${TAB}${TAB}return nil, &frameFault{detail: \"not a JSON object\", member: key}${NL}${TAB}${TAB}}" ""
# F23 readCapped
run_mutant R09 runner.go 'data, err := io.ReadAll(io.LimitReader(stream, int64(cap)+1))' 'data, err := io.ReadAll(stream)'
run_mutant R07 runner.go "${TAB}if len(data) > cap {${NL}${TAB}${TAB}return data[:cap], nil${NL}${TAB}}" ""
# F24 request bound halved
run_mutant M06c protocol.go "${TAB}if len(frame) > MaxFrameBytes {${NL}${TAB}${TAB}failure, fault := failInvalid(\"request frame exceeds 8 MiB\")" "${TAB}if len(frame) > 4<<20 {${NL}${TAB}${TAB}failure, fault := failInvalid(\"request frame exceeds 8 MiB\")"
# F22 token floor weakened
run_mutant T08 status.go 'len(raw) < 32' 'len(raw) < 8'
run_mutant T16 status.go 'len(raw) < 32' 'len(raw) < 16'
run_mutant T48 status.go 'len(raw) < 32' 'len(raw) < 48'
# F26 exit code
run_mutant R17 runner.go "${TAB}if exit, ok := waitErr.(*exec.ExitError); ok {${NL}${TAB}${TAB}result.ExitCode = exit.ExitCode()${NL}" "${TAB}if exit, ok := waitErr.(*exec.ExitError); ok {${NL}${TAB}${TAB}_ = exit${NL}"
# F27 terminator
run_mutant R14 runner.go "_, writeErr = stdinPipe.Write([]byte{'\n'})" "_, writeErr = stdinPipe.Write([]byte{})"

POST=$(sha256sum "$PKG"/protocol.go "$PKG"/runner.go "$PKG"/status.go | sha256sum | cut -d' ' -f1)
if [ "$PRE" = "$POST" ]; then echo "RESTORE_OK tree identical to pre-sweep bytes"; else echo "RESTORE_FAIL tree differs"; fi
