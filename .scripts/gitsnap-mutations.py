#!/usr/bin/env python3
"""Replay gitsnap-specific mutations in a disposable copy; never edit the source tree."""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--out', required=True, type=Path)
parser.add_argument('--vectors', type=Path, help='Use a task-specific vector file')
parser.add_argument('--only', help='Regex selecting vector names (controls still run)')
args = parser.parse_args()
root = Path(__file__).resolve().parent.parent
out = args.out.resolve()
out.mkdir(parents=True, exist_ok=True)
work = out / 'candidate'
if work.exists():
    parser.error('output candidate already exists; choose a fresh --out directory')
# Snapshot only the module and leaf dependencies; no .git, board or secrets.
for relative in ['go.mod', 'go.sum', 'internal']:
    source, target = root / relative, work / relative
    target.parent.mkdir(parents=True, exist_ok=True)
    if source.is_dir():
        shutil.copytree(source, target)
    else:
        shutil.copy2(source, target)
rows = json.loads((args.vectors or root / 'internal/gitsnap/testdata/mutations.json').read_text())
rows.sort(key=lambda row: 0 if row['name'].startswith('C-') else 1)
if args.only:
    rows = [r for r in rows if r['name'].startswith('C-') or re.search(args.only, r['name'])]
env = dict(os.environ, GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_SYSTEM=os.devnull)
results = []

def run(name, command):
    path = out / (name + '.log')
    with path.open('w') as log:
        log.write('$ ' + ' '.join(command) + '\n')
        log.flush()
        try:
            process = subprocess.run(command, cwd=work, env=env, stdout=log,
                                     stderr=subprocess.STDOUT, timeout=240)
            code = process.returncode
        except subprocess.TimeoutExpired:
            log.write('\nTIMEOUT: process killed; no exit status inferred\n')
            return None, [], []
        log.write('\nEXIT=' + str(code) + '\n')
    body = path.read_text()
    return (code, re.findall(r'^--- FAIL: (\S+)', body, re.M),
            re.findall(r'^=== RUN\s+(\S+)', body, re.M))

for row in rows:
    name = row['name']
    package = row.get('package', './internal/gitsnap')
    target = work / 'internal/gitsnap' / row['file']
    original = target.read_bytes()
    text = original.decode()
    result = {'name': name, 'bound': row.get('bound', ''), 'status': 'NOT_APPLIED'}
    try:
        old, limit = row['old'], row.get('limit', 0)
        hits = text.count(old) if old else 1
        if old and ((not limit and hits != 1) or (limit and hits < limit)):
            result['matches'] = hits
            continue
        changed = text.replace(old, row['new'], limit or hits) if old else text + row['new']
        target.write_text(changed)
        result['applied'] = True
        compile_code, _, _ = run(name + '.compile', ['go', 'test', package, '-run', '^$', '-count=1'])
        result['compile_exit'] = compile_code
        if compile_code != 0:
            result['status'] = 'COMPILE_FAILED' if compile_code is not None else 'TIMEOUT'
            continue
        full = row.get('full', False) or row['expect'] == 'survive'
        command = ['go', 'test', package, '-v', '-count=1']
        if not full:
            if not row['tests']:
                result['status'] = 'NO_BEHAVIOR_SELECTED'
                continue
            command += ['-run', '^(' + '|'.join(row['tests']) + ')$']
        code, failing, executed = run(name + '.behavior', command)
        result.update(behavior_exit=code, failing=failing, executed=executed)
        positive = None
        if row['positive']:
            positive, _, ran = run(name + '.positive', ['go', 'test', package, '-v', '-count=1', '-run', '^(' + '|'.join(row['positive']) + ')$'])
            if not set(row['positive']).issubset(ran):
                positive = 'NOT_EXECUTED'
        result['positive_exit'] = positive
        if code is None:
            result['status'] = 'TIMEOUT'
        elif not executed or not set(row['tests']).issubset(executed):
            result['status'] = 'NO_BEHAVIOR_EXECUTED'
        elif row['expect'] == 'survive':
            result['status'] = 'NEUTRAL_PASS' if code == 0 else 'NEUTRAL_FAILED'
        elif code and all(t in failing for t in row['tests']) and positive in (None, 0):
            result['status'] = 'BEHAVIOR_KILL'
        elif code:
            result['status'] = 'WRONG_FAILURE'
        else:
            result['status'] = 'SURVIVOR'
            result['bound'] = result['bound'] or 'Selected behavioral drivers did not detect this weakening; no gate-coverage claim is established by this vector.'
    finally:
        target.write_bytes(original)
        assert target.read_bytes() == original
        results.append(result)
        (out / 'results.json').write_text(json.dumps(results, indent=2) + '\n')
        print(name + ': ' + result['status'], flush=True)
    if name.startswith('C-') and result['status'] not in ('BEHAVIOR_KILL', 'NEUTRAL_PASS'):
        break  # Do not interpret the battery when its instrument control fails.
sys.exit(0 if all(r['status'] in ('BEHAVIOR_KILL', 'NEUTRAL_PASS') for r in results) else 1)
