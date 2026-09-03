#!/bin/zsh
# Mutation harness for internal/axerror.
#
# Each mutant must be VERIFIED APPLIED before its measurement is trusted. A
# deletion is verified by the absence of the exact text it deleted, and an
# edit by the presence of the exact text it wrote. Revision 1 of this harness
# checked an unrelated present-marker after several deletions, which could not
# distinguish "mutation applied" from "perl no-op"; that is what mode=absent
# below fixes.
set -u
ROOT="$(pwd)"
BACKUP="$ROOT/.temp/TASK-260830-34elja/backup"
rm -rf "$BACKUP"; mkdir -p "$BACKUP"
BASE="$(go test ./internal/axerror/ 2>&1)"
if [ $? -ne 0 ]; then echo "BASELINE RED - refusing to mutate:"; echo "$BASE"; exit 1; fi
cp internal/axerror/*.go "$BACKUP"/
restore() { cp "$BACKUP"/*.go internal/axerror/; }
LOG="$ROOT/.temp/TASK-260830-34elja/mutation-01.log"
: > "$LOG"

# run_mutant <name> <file> <present|absent> <marker>
run_mutant() {
  local name="$1"; local file="$2"; local mode="$3"; local marker="$4"
  if [ "$mode" = "present" ]; then
    if ! grep -qF -- "$marker" "internal/axerror/$file"; then
      echo "MUTANT-ABSENT $name (expected text was not written)" | tee -a "$LOG"
      restore; return
    fi
  else
    if grep -qF -- "$marker" "internal/axerror/$file"; then
      echo "MUTANT-ABSENT $name (text that had to be deleted is still there)" | tee -a "$LOG"
      restore; return
    fi
  fi
  local out
  out="$(go test ./internal/axerror/ 2>&1)"
  local rc=$?
  if [ $rc -eq 0 ]; then
    echo "SURVIVED  $name -- suite still green" | tee -a "$LOG"
  else
    echo "KILLED    $name -- $(echo "$out" | grep -E '^\s+--- FAIL' | head -3 | tr '\n' ';')" | tee -a "$LOG"
  fi
  restore
  if ! go test ./internal/axerror/ >/dev/null 2>&1; then
    echo "RESTORE FAILED after $name - aborting" | tee -a "$LOG"; exit 1
  fi
}

# 1. Narrow the retryability policy: remove one forbidden class or code at a time.
perl -0pi -e 's/\t7:   `exit 7 is[^\n]*\n//' internal/axerror/registry.go
run_mutant "retryability: exit-7 class removed" registry.go absent '7:   `exit 7 is'
perl -0pi -e 's/\t130: `exit 130 is[^\n]*\n//' internal/axerror/registry.go
run_mutant "retryability: exit-130 class removed" registry.go absent '130: `exit 130 is'
perl -0pi -e 's/\t"operation_uncertain":[^\n]*\n//' internal/axerror/registry.go
run_mutant "retryability: operation_uncertain removed" registry.go absent '"operation_uncertain":'
perl -0pi -e 's/\t"terminal_backend_stale_generation":[^\n]*\n//' internal/axerror/registry.go
run_mutant "retryability: stale_generation removed" registry.go absent '"terminal_backend_stale_generation":'

# 2. Narrow version admission: accept a code registered by any version.
perl -0pi -e 's/\tif _, admitted := entry\.versions\[version\]; !admitted \{/\tif _, admitted := entry.versions[version]; false \&\& !admitted {/' internal/axerror/registry.go
run_mutant "registry: per-version code admission dropped" registry.go present 'false && !admitted'

# 3. Admit the success exit status as a failure class.
perl -0pi -e 's/\tif status == successExit \{\n\t\treturn false\n\t\}\n//' internal/axerror/registry.go
run_mutant "registry: success status admitted as a failure class" registry.go absent 'if status == successExit {'

# 4. Drop the nested excluded-key walk in details.
perl -0pi -e 's/\t\t\tif err := refuseExcludedKey\(nested\); err != nil \{\n\t\t\t\treturn err\n\t\t\t\}\n//' internal/axerror/details.go
run_mutant "details: nested excluded-key walk dropped" details.go absent 'refuseExcludedKey(nested)'

# 4a. Narrow the scanner to the credential class only.
perl -0pi -e 's/\t"(raw_transcript|scrollback|terminal_scrollback|transcript|dotenv|env_secret|environment_secret|secret|secrets|bundle_bytes|bundle_content|opaque_bundle)":[^\n]*\n//g' internal/axerror/details.go
run_mutant "details: scanner narrowed to the credential class" details.go absent '"raw_transcript":'

# 4b. Drop one credential key from the scanner.
perl -0pi -e 's/\t"password":[^\n]*\n//' internal/axerror/details.go
run_mutant "details: one credential key removed from the scanner" details.go absent '"password":'

# 4c. Widen the scanner back to substring matching, the removed false-positive defect.
perl -0pi -e 's/\tif class, excluded := excludedDetailKeys\[key\]; excluded \{/\tfor candidate, class := range excludedDetailKeys { if excluded := strings.Contains(key, candidate); excluded {/' internal/axerror/details.go
perl -0pi -e 's/\t\treturn fmt\.Errorf\("%w: key %q names %s, which no detail may contain", ErrInvalidDetails, key, class\)\n\t\}/\t\treturn fmt.Errorf("%w: key %q names %s, which no detail may contain", ErrInvalidDetails, key, class) } }/' internal/axerror/details.go
run_mutant "details: scanner widened back to substring matching" details.go present 'strings.Contains(key, candidate)'

# 5. Narrow the causal leak gate to the outermost cause only.
perl -0pi -e 's/\t\trendered = append\(rendered, current\.Error\(\)\)/\t\trendered = append(rendered, current.Error());  current = nil; continue/' internal/axerror/details.go
run_mutant "redaction: only the outermost cause link checked" details.go present 'current = nil; continue'

# 6. Drop the detail-value walk in the leak gate (message only).
perl -0pi -e 's/\t\t\tif valueContains\(details\[key\], rendered\) \{/\t\t\tif false \&\& valueContains(details[key], rendered) {/' internal/axerror/details.go
run_mutant "redaction: detail values no longer checked for the cause" details.go present 'false && valueContains'

# 7. Accept any major in the reader.
perl -0pi -e 's/\tif err != nil \|\| major != 1 \{/\tif err != nil \&\& major != 1 {/' internal/axerror/decode.go
run_mutant "reader: unsupported major admitted" decode.go present 'err != nil && major != 1'

# 8. Drop the registered-code exit agreement check.
perl -0pi -e 's/\t\tif exitCode != expectedExit \{/\t\tif false \&\& exitCode != expectedExit {/' internal/axerror/decode.go
run_mutant "reader: exit status may contradict a registered code" decode.go present 'false && exitCode != expectedExit'

# 8a. Narrow the exit_code JSON-type check: strip quotes before parsing, which
#     is exactly what reading the member through a json.Number field would do.
perl -0pi -e 's/\ttext := string\(bytes\.TrimSpace\(raw\)\)/\ttext := string(bytes.Trim(bytes.TrimSpace(raw), "\\""))/' internal/axerror/decode.go
run_mutant "reader: quoted exit_code admitted as a number" decode.go present 'bytes.Trim(bytes.TrimSpace(raw)'

# 9. Drift one bootstrap mapping to a plausible neighbour.
perl -0pi -e 's/otherwise: "task_board_bridge_unavailable"/otherwise: "task_board_bundle_invalid"/' internal/axerror/local.go
run_mutant "bootstrap: bridge fallback code drifted" local.go present 'task_board_bundle_invalid'

# 10. Let the local constructor take the version from a bound contract default.
perl -0pi -e 's/SurfaceTerminalBackend: \{version: Version130/SurfaceTerminalBackend: {version: Version120/' internal/axerror/local.go
run_mutant "bootstrap: terminal surface bound to the wrong error version" local.go present 'SurfaceTerminalBackend: {version: Version120'

# 11. Drop the typed-detail requirement for target_auth_missing in New.
perl -0pi -e 's/\tif spec\.Code == "target_auth_missing" \{/\tif false \&\& spec.Code == "target_auth_missing" {/' internal/axerror/axerror.go
run_mutant "typed details: target_auth_missing requirement dropped in the writer" axerror.go present 'false && spec.Code == "target_auth_missing"'

# 12. Let MarshalJSON emit a null identifier instead of omitting it.
perl -0pi -e 's/OperationID   \*string `json:"operation_id,omitempty"`/OperationID   *string `json:"operation_id"`/' internal/axerror/axerror.go
run_mutant "encoder: unknown operation identifier emitted as null" axerror.go present 'json:"operation_id"`'

# 13. Widen the message bound to bytes rather than characters.
perl -0pi -e 's/\tcount := utf8\.RuneCountInString\(message\)/\tcount := len(message)/' internal/axerror/axerror.go
run_mutant "message bound measured in bytes" axerror.go present 'count := len(message)'

# 14. Drop the closed-object rule in the reader.
perl -0pi -e 's/\tdecoder\.DisallowUnknownFields\(\)\n//' internal/axerror/decode.go
run_mutant "reader: top-level object no longer closed" decode.go absent 'decoder.DisallowUnknownFields()'

# 15. Detail ownership, the revision-1 defect. Four arms, each a narrowing of
#     the deep copy rather than a deletion of it, plus the two deletions.
perl -0pi -e 's/\t\tclone\[key\] = cloneDetailValue\(value\)/\t\tclone[key] = value/' internal/axerror/axerror.go
run_mutant "details: construction copy shallow again (revision-1 defect)" axerror.go present 'clone[key] = value'
perl -0pi -e 's/\t\t\tnested\[key\] = cloneDetailValue\(member\)/\t\t\tnested[key] = member/' internal/axerror/axerror.go
run_mutant "details: deep copy narrowed to one level (nested maps shared)" axerror.go present 'nested[key] = member'
perl -0pi -e 's/\t\t\tmembers\[index\] = cloneDetailValue\(member\)/\t\t\tmembers[index] = member/' internal/axerror/axerror.go
run_mutant "details: deep copy narrowed to maps only (arrays shared)" axerror.go present 'members[index] = member'
perl -0pi -e 's/\treturn cloneDetailValue\(value\), true/\treturn value, true/' internal/axerror/axerror.go
run_mutant "details: accessor hands out the live container again" axerror.go present 'return value, true'
perl -0pi -e 's/\tcase \[\]any:\n\t\tmembers := make\(\[\]any, len\(typed\)\)\n\t\tfor index, member := range typed \{\n\t\t\tmembers\[index\] = cloneDetailValue\(member\)\n\t\t\}\n\t\treturn members\n//' internal/axerror/axerror.go
run_mutant "details: array arm of the deep copy deleted" axerror.go absent 'members := make([]any, len(typed))'

restore
echo "--- final restore verified ---" | tee -a "$LOG"
go test ./internal/axerror/ -count=1 2>&1 | tail -2 | tee -a "$LOG"
