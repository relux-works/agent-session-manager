import re, collections, json

SRC='internal/terminalbackend/manifest.go'
src=open(SRC).read().splitlines()
sites=[]
for i,ln in enumerate(src,1):
    if ln.lstrip().startswith('//'): continue
    for m in re.finditer(r'mismatchf\("([^"]*)"', ln): sites.append((i,m.group(1)))
    for m in re.finditer(r'integrityFailure\("([^"]*)"', ln): sites.append((i,m.group(1)))
    for m in re.finditer(r'Detail:\s*"([^"]*)"', ln): sites.append((i,m.group(1)))
# drop helper definitions (they use variables, already excluded by literal match)

blocks=[]
for line in open('.temp/TASK-260830-1zvmw7/round3/cov.out'):
    line=line.strip()
    if not line.startswith('github.com/relux-works/agent-session-manager/internal/terminalbackend/manifest.go:'): continue
    spec,stmts,count=line.rsplit(' ',2)
    rng=spec.split(':',1)[1]
    a,b=rng.split(',')
    sl,sc=map(int,a.split('.')); el,ec=map(int,b.split('.'))
    blocks.append((sl,el,int(count)))

def executed(line):
    cand=[b for b in blocks if b[0]<=line<=b[1]]
    if not cand: return None
    cand.sort(key=lambda b:(b[1]-b[0]))
    return cand[0][2]>0

tests=''
for f in ['manifest_test.go','manifest_pin_test.go','internal_pin_test.go','terminalbackend_test.go']:
    tests+=open('internal/terminalbackend/'+f).read()

by_detail=collections.Counter(d for _,d in sites)
ex=[s for s in sites if executed(s[0])]
nx=[s for s in sites if executed(s[0])==False]
unk=[s for s in sites if executed(s[0]) is None]

ex_asserted=[s for s in ex if tests.count('"%s"'%s[1])>0]
ex_unasserted=[s for s in ex if tests.count('"%s"'%s[1])==0]

print("total arm sites            :", len(sites))
print("executed by suite          :", len(ex))
print("not executed               :", len(nx), " (no-block:", len(unk), ")")
print("executed & identity asserted:", len(ex_asserted), "/", len(ex))
print("executed & UNasserted      :", [(l,d) for l,d in ex_unasserted])
print()
uniq=[s for s in ex if by_detail[s[1]]==1]
shared=[s for s in ex if by_detail[s[1]]>1]
print("executed sites with file-unique detail :", len(uniq))
print("executed sites sharing a detail        :", len(shared))
sh=collections.Counter(d for _,d in shared)
for d,n in sorted(sh.items()):
    print("   %-30s executed_sites=%d total_sites=%d" % (d,n,by_detail[d]))
distinct_ex=set(d for _,d in ex)
print()
print("distinct executed arm identities        :", len(distinct_ex))
print("distinct executed identities asserted   :", sum(1 for d in distinct_ex if tests.count('"%s"'%d)>0))
print()
print("NOT-EXECUTED site details (unique):")
for d,n in sorted(collections.Counter(d for _,d in nx).items()):
    print("   %-30s sites=%d asserted_elsewhere=%s" % (d,n, tests.count('"%s"'%d)>0))
