#!/bin/zsh
# usage: mutate.sh <id> <file> <exact-old> <new>
set -e
BK=.temp/TASK-260830-wbpf1v-review3/backup
ID=$1; F=internal/sessrepo/$2; OLD=$3; NEW=$4
cp $BK/$2 $F
COUNT=$(python3 -c "
import sys
s=open('$F').read()
sys.stdout.write(str(s.count(sys.argv[1])))
" "$OLD")
if [ "$COUNT" != "1" ]; then echo "$ID: NOT_APPLIED (pattern count=$COUNT)"; cp $BK/$2 $F; exit 0; fi
python3 -c "
import sys
p='$F'
s=open(p).read()
s=s.replace(sys.argv[1], sys.argv[2], 1)
open(p,'w').write(s)
" "$OLD" "$NEW"
if ! go build ./internal/sessrepo >/dev/null 2>&1; then echo "$ID: COMPILE_FAIL"; cp $BK/$2 $F; exit 0; fi
BEH=$(go test ./internal/sessrepo -count=1 -run '.*' >/tmp/beh_$ID.log 2>&1 && echo PASS || echo FAIL)
FULL=$(go test ./internal/sessrepo -count=1 >/tmp/full_$ID.log 2>&1 && echo PASS || echo FAIL)
if [ "$BEH" = "FAIL" ]; then K="KILLED(behavioral)"; elif [ "$FULL" = "FAIL" ]; then K="KILLED(census-only)"; else K="SURVIVED"; fi
echo "$ID: $K  behavioral=$BEH full=$FULL"
if [ "$BEH" = "FAIL" ]; then grep -E "^\s+--- FAIL|^--- FAIL" /tmp/beh_$ID.log | head -6; fi
if [ "$BEH" = "PASS" ] && [ "$FULL" = "FAIL" ]; then grep -E "sessrepo (refusal|boundary|exercised)|--- FAIL" /tmp/full_$ID.log | head -4; fi
cp $BK/$2 $F
