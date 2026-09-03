#!/bin/zsh
set -u
BK=".temp/TASK-260830-34elja/review-backup"
restore() { cp "$BK"/*.go internal/axerror/; }
run() { # name file marker
  if ! grep -qF -- "$3" "internal/axerror/$2"; then echo "MUTANT-ABSENT $1"; restore; return; fi
  out="$(go test ./internal/axerror/ -count=1 2>&1)"; rc=$?
  if [ $rc -eq 0 ]; then echo "SURVIVED  $1"; else echo "KILLED    $1 -- $(echo "$out"|grep -E '^\s+--- FAIL'|head -2|tr '\n' ';')"; fi
  restore
}
# M1: reader-side typed-detail requirement dropped
perl -0pi -e 's/\tif code == "target_auth_missing" \{/\tif false \&\& code == "target_auth_missing" {/' internal/axerror/decode.go
run "reader: target_auth_missing typed details not required" decode.go 'if false && code == "target_auth_missing"'
# M2: exit_code quoted-string refusal removed
perl -0pi -e 's/if text == "" \|\| text\[0\] == .".. \{/if text == "" {/' internal/axerror/decode.go
run "reader: quoted exit_code admitted" decode.go 'if text == "" {'
# M3: bound-version mismatch check removed
perl -0pi -e 's/\tif candidate != expected \{/\tif false \&\& candidate != expected {/' internal/axerror/decode.go
run "reader: document may pick its own bound version" decode.go 'if false && candidate != expected'
# M4: trailing content admitted
perl -0pi -e 's/if err := decoder\.Decode\(&trailing\); !errors\.Is\(err, io\.EOF\) \{/if err := decoder.Decode(\&trailing); false \&\& !errors.Is(err, io.EOF) {/' internal/axerror/decode.go
run "reader: trailing content after the object admitted" decode.go 'false && !errors.Is(err, io.EOF)'
# M5: unknown code reported as registered
perl -0pi -e 's/\t\tregistered = false/\t\tregistered = true/' internal/axerror/decode.go
run "reader: unknown code reported as registered" decode.go 'registered = true'
# M6: causal-leak gate disabled by raising the minimum
perl -0pi -e 's/\tminRedactableCause = 8/\tminRedactableCause = 100000/' internal/axerror/details.go
run "redaction: causal gate never fires" details.go 'minRedactableCause = 100000'
# M7: 16 KiB canonical bound widened
perl -0pi -e 's/\tmaxDetailCanonical = 16 \* 1024/\tmaxDetailCanonical = 16 * 1024 * 1024/' internal/axerror/details.go
run "details: canonical size bound widened" details.go 'maxDetailCanonical = 16 * 1024 * 1024'
# M8: 64-key bound widened
perl -0pi -e 's/\tmaxDetailKeys      = 64/\tmaxDetailKeys      = 640/' internal/axerror/details.go
run "details: key-count bound widened" details.go 'maxDetailKeys      = 640'
# M9: depth bound widened
perl -0pi -e 's/\tmaxDetailDepth     = 4/\tmaxDetailDepth     = 40/' internal/axerror/details.go
run "details: nesting-depth bound widened" details.go 'maxDetailDepth     = 40'
# M10: LocalFromUntrusted admits an unpinned outcome by defaulting
perl -0pi -e 's/\tdefault:\n\t\treturn nil, fmt\.Errorf\("%w: outcome %q is not a pinned classification", ErrInvalidStructuredError, outcome\)/\tdefault:\n\t\tcode = mapping.otherwise/' internal/axerror/local.go
run "bootstrap: unpinned outcome falls back to the otherwise code" local.go 'code = mapping.otherwise'
# M11: code grammar not enforced on the reader
perl -0pi -e 's/\tif err := validateCodeGrammar\(code\); err != nil \{\n\t\treturn nil, err\n\t\}\n\tif document\.Message == nil/\tif document.Message == nil/' internal/axerror/decode.go
run "reader: code grammar not enforced" decode.go 'code := Code(*document.Code)'
# M12: BindingFor returns a default instead of refusing
perl -0pi -e 's/\tif !bound \{\n\t\treturn "", fmt\.Errorf\("%w: %s major %d", ErrUnboundContract, contract\.ID, contract\.Major\)\n\t\}/\tif !bound {\n\t\treturn Version100, nil\n\t}/' internal/axerror/binding.go
run "binding: unbound contract falls back to 1.0.0" binding.go 'return Version100, nil'
restore
go test ./internal/axerror -count=1 >/dev/null 2>&1 && echo "--- final restore verified green ---" || echo "--- RESTORE FAILED ---"
