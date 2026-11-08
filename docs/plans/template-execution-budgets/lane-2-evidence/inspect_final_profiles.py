import json
from pathlib import Path
import subprocess
import time

root = Path.cwd() / '.git/lane-2-validation'
receipts = []
for name in ['value-control-final', 'value-guarded-final', 'output-control-final', 'output-guarded-final']:
    stage = root / name
    for view, flags in [
        ('flat', ['-top', '-alloc_space', '-nodecount=16']),
        ('cumulative', ['-top', '-cum', '-alloc_space', '-nodecount=16']),
        ('source', ['-alloc_space', '-list', 'concatString|builtinRepeat|builtinSplit|executionWriter.*write|beginOutputTransaction'])
    ]:
        command = ['go', 'tool', 'pprof'] + flags + [str(stage / 'jet.test'), str(stage / 'allocs.pprof')]
        started = time.time()
        result = subprocess.run(command, capture_output=True, text=True, timeout=25)
        (stage / (view + '.txt')).write_text(result.stdout + result.stderr)
        receipts.append(dict(command=command, timeout_seconds=25, started=started, finished=time.time(), exit_code=result.returncode))
        if result.returncode:
            print(result.stderr)
            raise SystemExit(result.returncode)
    print(name + '\n' + (stage / 'flat.txt').read_text())
(root / 'profile-inspection-stages.json').write_text(json.dumps(receipts, indent=2) + '\n')
