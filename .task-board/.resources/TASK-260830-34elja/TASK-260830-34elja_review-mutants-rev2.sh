#!/bin/zsh
# Reviewer round-2 mutation harness for TASK-260830-34elja.
# Every mutant is verified PRESENT (or, for a deletion, the removed text ABSENT)
# in the file before the suite is believed. Sources are restored from a byte
# backup, never from git.
set -u
ROOT=$(pwd)
BK="$ROOT/.temp/TASK-260830-34elja/backup"
KILLED=0; SURVIVED=0; ABSENT=0

restore() {
  cp "$BK"/axerror.go "$BK"/binding.go "$BK"/decode.go "$BK"/details.go \
     "$BK"/local.go "$BK"/registry.go "$ROOT/internal/axerror/"
  cp "$BK"/traceability.go "$ROOT/internal/traceability/traceability.go"
  cp "$BK"/ownership.v0.5.0.json "$ROOT/internal/traceability/ownership.v0.5.0.json"
}

# run_mutant <name> <file> <perl-expr> <marker-mode:present|absent> <marker> [target]
run_mutant() {
  local name=$1 file=$2 expr=$3 mode=$4 marker=$5 target=${6:-./internal/axerror}
  restore
  perl -0pi -e "$expr" "$file"
  if [[ "$mode" == "present" ]]; then
    if ! grep -qF -- "$marker" "$file"; then
      printf 'MUTANT-ABSENT  %-52s (mutated text not in %s)\n' "$name" "$file"; ABSENT=$((ABSENT+1)); restore; return
    fi
  else
    if grep -qF -- "$marker" "$file"; then
      printf 'MUTANT-ABSENT  %-52s (text still present in %s)\n' "$name" "$file"; ABSENT=$((ABSENT+1)); restore; return
    fi
  fi
  local out
  out=$(go test "$target" -count=1 2>&1)
  if [[ $? -ne 0 ]]; then
    printf 'KILLED    %-52s %s\n' "$name" "$(echo "$out" | grep -m1 -E '^\s+.*_test.go:|FAIL|cannot|undefined' | head -1 | cut -c1-110)"
    KILLED=$((KILLED+1))
  else
    printf 'SURVIVED  %-52s <-- gate not covered\n' "$name"
    SURVIVED=$((SURVIVED+1))
  fi
  restore
}

A=internal/axerror/axerror.go
D=internal/axerror/details.go
C=internal/axerror/decode.go
R=internal/axerror/registry.go
L=internal/axerror/local.go

echo "--- F1: deep-copy narrowings (the round-1 fix) ---"
run_mutant "M01 clone map arm narrowed to one level" $A \
  's/nested\[key\] = cloneDetailValue\(member\)/nested[key] = member/' present 'nested[key] = member'
run_mutant "M02 clone slice arm narrowed to one level" $A \
  's/members\[index\] = cloneDetailValue\(member\)/members[index] = member/' present 'members[index] = member'
run_mutant "M03 Detail returns the live value" $A \
  's/return cloneDetailValue\(value\), true/return value, true/' present 'return value, true'
run_mutant "M04 New stores the caller map directly" $A \
  's/details:        cloneDetails\(spec\.Details\),/details:        spec.Details,/' present 'details:        spec.Details,'
run_mutant "M05 cloneDetails shallow (top level only)" $A \
  's/clone\[key\] = cloneDetailValue\(value\)/clone[key] = value/' present 'clone[key] = value'
run_mutant "M06 clone map arm dropped entirely" $A \
  's/\tcase map\[string\]any:\n\t\tnested := make\(map\[string\]any, len\(typed\)\)\n\t\tfor key, member := range typed \{\n\t\t\tnested\[key\] = cloneDetailValue\(member\)\n\t\t\}\n\t\treturn nested\n/\tcase map[string]any:\n\t\treturn typed\n/' present 'case map[string]any:
		return typed'

echo "--- Section 15.1 detail bounds (narrowed, not deleted) ---"
run_mutant "M07 depth bound 4 -> 5" $D 's/maxDetailDepth     = 4/maxDetailDepth     = 5/' present 'maxDetailDepth     = 5'
run_mutant "M08 canonical size 16 KiB -> 64 KiB" $D 's/maxDetailCanonical = 16 \* 1024/maxDetailCanonical = 64 * 1024/' present 'maxDetailCanonical = 64 * 1024'
run_mutant "M09 key count 64 -> 128" $D 's/maxDetailKeys      = 64/maxDetailKeys      = 128/' present 'maxDetailKeys      = 128'
run_mutant "M10 key grammar widened to 127 chars" $D 's/\[a-z\]\[a-z0-9_\]\{0,63\}\$`\)/[a-z][a-z0-9_]{0,127}$`)/' present '{0,127}$`)'
run_mutant "M11 excluded-class scan only at top level" $D 's/\t\t\tif err := refuseExcludedKey\(nested\); err != nil \{\n\t\t\t\treturn err\n\t\t\t\}\n//' absent 'refuseExcludedKey(nested)'
run_mutant "M12 one credential key removed from the table" $D 's/\t"password":           "credential",\n//' absent '"password":           "credential",'
run_mutant "M13 message upper bound 4096 -> 8192" $D 's/maxMessageRunes    = 4096/maxMessageRunes    = 8192/' present 'maxMessageRunes    = 8192'
run_mutant "M14 message lower bound 1 -> 0" $D 's/minMessageRunes    = 1/minMessageRunes    = 0/' present 'minMessageRunes    = 0'

echo "--- causal redaction (narrowed) ---"
run_mutant "M15 causal scan skips detail values" $D 's/\t\tfor _, key := range sortedKeys\(details\) \{\n\t\t\tif valueContains\(details\[key\], rendered\) \{\n/\t\tfor _, key := range []string{} {\n\t\t\tif valueContains(details[key], rendered) {\n/' present 'for _, key := range []string{} {'
run_mutant "M16 causal scan only the outermost cause" $D 's/rendered = append\(rendered, current\.Error\(\)\)\n\t\tswitch unwrapped := current\.\(type\) \{/rendered = append(rendered, current.Error())\n\t\tif len(rendered) > 0 {\n\t\t\tbreak\n\t\t}\n\t\tswitch unwrapped := current.(type) {/' present 'if len(rendered) > 0 {'
run_mutant "M17 causal minimum length 8 -> 64" $D 's/minRedactableCause = 8/minRedactableCause = 64/' present 'minRedactableCause = 64'
run_mutant "M18 valueContains ignores maps" $D 's/\tcase map\[string\]any:\n\t\tfor _, member := range typed \{\n\t\t\tif valueContains\(member, needle\) \{\n\t\t\t\treturn true\n\t\t\t\}\n\t\t\}\n\treturn false/\treturn false/' absent 'case map[string]any:
		for _, member := range typed {
			if valueContains(member, needle) {'

echo "--- reader gates (narrowed) ---"
run_mutant "M19 major gate admits major 2 as well" $C 's/if err != nil \|\| major != 1 \{/if err != nil || (major != 1 \&\& major != 2) {/' present '(major != 1 && major != 2)'
run_mutant "M20 version match compares majors only" $C 's/if candidate != expected \{/if candidate[0] != expected[0] {/' present 'if candidate[0] != expected[0] {'
run_mutant "M21 unknown members tolerated" $C 's/\tdecoder\.DisallowUnknownFields\(\)\n//' absent 'decoder.DisallowUnknownFields()'
run_mutant "M22 trailing content tolerated" $C 's/if err := decoder\.Decode\(&trailing\); !errors\.Is\(err, io\.EOF\) \{/if err := decoder.Decode(\&trailing); false \&\& !errors.Is(err, io.EOF) {/' present 'false && !errors.Is(err, io.EOF)'
run_mutant "M23 decoded exit-code mismatch tolerated" $C 's/\t\tif exitCode != expectedExit \{/\t\tif false \&\& exitCode != expectedExit {/' present 'if false && exitCode != expectedExit {'
run_mutant "M24 decoder accepts success status 0" $R 's/func IsFailureExitStatus\(status int\) bool \{/func IsFailureExitStatus(status int) bool {\n\tif status == successExit {\n\t\treturn true\n\t}/' present 'if status == successExit {
		return true
	}'
run_mutant "M25 decode drops the retryability refusal" $C 's/\tif \*document\.Retryable \{\n\t\tif reason, forbidden := RetryabilityRefusal\(code, exitCode\); forbidden \{/\tif false \&\& *document.Retryable {\n\t\tif reason, forbidden := RetryabilityRefusal(code, exitCode); forbidden {/' present 'if false && *document.Retryable {'
run_mutant "M26 decode skips detail validation" $C 's/\tif err := ValidateDetails\(details\); err != nil \{\n\t\treturn nil, err\n\t\}\n\tif code == "target_auth_missing"/\tif code == "target_auth_missing"/' absent 'if err := ValidateDetails(details); err != nil {
		return nil, err
	}
	if code == "target_auth_missing"'

echo "--- writer gates (narrowed) ---"
run_mutant "M27 New drops the retryability refusal" $A 's/\tif spec\.Retryable \{\n\t\tif reason, forbidden := RetryabilityRefusal\(spec\.Code, exitCode\); forbidden \{/\tif false \&\& spec.Retryable {\n\t\tif reason, forbidden := RetryabilityRefusal(spec.Code, exitCode); forbidden {/' present 'if false && spec.Retryable {'
run_mutant "M28 retryability refusal loses the exit-16 class" $R 's/case 7, 16:/case 7:/' present 'case 7:'
run_mutant "M29 typed-detail presence admits an empty string" $D 's/\t\tif !ok \|\| text == "" \{/\t\tif !ok {/' present 'if !ok {'
run_mutant "M30 New skips the typed-detail requirement" $A 's/\tif spec\.Code == "target_auth_missing" \{/\tif false \&\& spec.Code == "target_auth_missing" {/' present 'if false && spec.Code == "target_auth_missing" {'
run_mutant "M31 New skips the causal-leak refusal" $A 's/\tif err := refuseCausalLeak\(spec\.Message, spec\.Details, spec\.Cause\); err != nil \{\n\t\treturn nil, err\n\t\}\n//' absent 'refuseCausalLeak(spec.Message, spec.Details, spec.Cause)'
run_mutant "M32 code admitted for every registered version" $R 's/if _, carried := entry\.versions\[version\]; !carried \{/if _, carried := entry.versions[version]; false \&\& !carried {/' present 'if _, carried := entry.versions[version]; false && !carried {'
run_mutant "M33 local surface table loses its version pin" $L 's/SurfaceMeshRPC: \{version: Version100/SurfaceMeshRPC: {version: Version130/' present 'SurfaceMeshRPC: {version: Version130'
run_mutant "M34 local constructor admits an unknown outcome" $L 's/\tdefault:\n\t\treturn nil, fmt\.Errorf\("%w: outcome %q is not a pinned classification", ErrInvalidStructuredError, outcome\)/\tdefault:\n\t\tcode = mapping.otherwise/' present 'code = mapping.otherwise'

echo "SUMMARY killed=$KILLED survived=$SURVIVED mutant-absent=$ABSENT"
restore
#!/bin/zsh
set -u
ROOT=$(pwd)
BK="$ROOT/.temp/TASK-260830-34elja/backup"
KILLED=0; SURVIVED=0; ABSENT=0
restore() {
  cp "$BK"/axerror.go "$BK"/binding.go "$BK"/decode.go "$BK"/details.go \
     "$BK"/local.go "$BK"/registry.go "$ROOT/internal/axerror/"
}
run_mutant() {
  local name=$1 file=$2 expr=$3 mode=$4 marker=$5 target=${6:-./internal/axerror}
  restore
  perl -0pi -e "$expr" "$file"
  if [[ "$mode" == "present" ]]; then
    grep -qF -- "$marker" "$file" || { printf 'MUTANT-ABSENT  %-52s\n' "$name"; ABSENT=$((ABSENT+1)); restore; return; }
  else
    grep -qF -- "$marker" "$file" && { printf 'MUTANT-ABSENT  %-52s\n' "$name"; ABSENT=$((ABSENT+1)); restore; return; }
  fi
  # a mutant that does not compile is not a kill
  if ! go build ./internal/axerror >/dev/null 2>&1; then
    printf 'MUTANT-NOBUILD %-52s (compile error, not a kill)\n' "$name"; ABSENT=$((ABSENT+1)); restore; return
  fi
  local out; out=$(go test "$target" -count=1 2>&1)
  if [[ $? -ne 0 ]]; then
    printf 'KILLED    %-52s %s\n' "$name" "$(echo "$out" | grep -m1 -E '^\s+.*_test.go:|--- FAIL' | cut -c1-100)"
    KILLED=$((KILLED+1))
  else
    printf 'SURVIVED  %-52s <-- gate not covered\n' "$name"; SURVIVED=$((SURVIVED+1))
  fi
  restore
}
D=internal/axerror/details.go
C=internal/axerror/decode.go
R=internal/axerror/registry.go

run_mutant "M10b key grammar widened to 127 chars" $D \
  's/\{0,63\}\$`\)/{0,127}\$`)/' present '{0,127}$`)'
run_mutant "M18b valueContains ignores nested maps" $D \
  's/\tcase map\[string\]any:\n\t\tfor _, member := range typed \{\n\t\t\tif valueContains\(member, needle\) \{\n\t\t\t\treturn true\n\t\t\t\}\n\t\t\}\n\t\}\n\treturn false\n\}/\tcase map[string]any:\n\t\t_ = typed\n\t}\n\treturn false\n}/' present 'case map[string]any:
		_ = typed
	}
	return false
}'
run_mutant "M26b decode skips detail validation" $C \
  's/\tif err := ValidateDetails\(details\); err != nil \{\n\t\treturn nil, err\n\t\}\n\tif code == "target_auth_missing" \{/\tif code == "target_auth_missing" {/' present '	details := *document.Details
	if code == "target_auth_missing" {'
run_mutant "M28b retryability loses the exit-16 class" $R \
  's/\t16:  `exit 16 is "Explicit policy refusal, including missing destructive confirmation"; the identical request cannot succeed without new confirmation`,\n//' absent '16:  `exit 16 is'
run_mutant "M28c retryability loses the exit-7 class" $R \
  's/\t7:   `exit 7 is "Authentication\/authorization\/allowlist failure"; the identical request cannot succeed without new authority`,\n//' absent '7:   `exit 7 is'
run_mutant "M28d retryability loses operation_uncertain" $R \
  's/\t"operation_uncertain":               `Section 15.3: "operation_uncertain is not retry permission: status\/recovery inspection is mandatory"`,\n//' absent '"operation_uncertain":               `Section 15.3'
run_mutant "M32b per-version code admission removed" $R \
  's/\tif _, admitted := entry\.versions\[version\]; !admitted \{\n\t\treturn 0, fmt\.Errorf\(\n\t\t\t"%w: %q is not registered by Structured Error %s", ErrUnregisteredCode, code, version\)\n\t\}\n//' absent 'is not registered by Structured Error %s'
run_mutant "M35 ExitCodeFor admits an unregistered version" $R \
  's/func ExitCodeFor\(version Version, code Code\) \(int, error\) \{\n\tif !isRegisteredVersion\(version\) \{/func ExitCodeFor(version Version, code Code) (int, error) {\n\tif false \&\& !isRegisteredVersion(version) {/' present 'if false && !isRegisteredVersion(version) {'
run_mutant "M36 typed-detail non-empty check narrowed" $D \
  's/\t\ttext, ok := value\.\(string\)\n\t\tif !ok \|\| text == "" \{/\t\ttext, ok := value.(string)\n\t\t_ = text\n\t\tif !ok {/' present '_ = text
		if !ok {'
run_mutant "M37 causal minimum 8 -> 16 (one step narrower)" $D \
  's/minRedactableCause = 8/minRedactableCause = 16/' present 'minRedactableCause = 16'
run_mutant "M38 causal minimum 8 -> 64" $D \
  's/minRedactableCause = 8/minRedactableCause = 64/' present 'minRedactableCause = 64'
run_mutant "M39 causal minimum 8 -> 4096 (scan effectively off)" $D \
  's/minRedactableCause = 8/minRedactableCause = 4096/' present 'minRedactableCause = 4096'
run_mutant "M40 detail key must merely be non-empty" $D \
  's/if !detailKeyPattern\.MatchString\(key\) \{/if key == "" {/' present 'if key == "" {'
echo "SUMMARY killed=$KILLED survived=$SURVIVED absent-or-nobuild=$ABSENT"
restore
