import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

name, lane, deadline, *command = sys.argv[1:]
root = Path.cwd()
art = root / '.git/finalization' / name
art.mkdir(parents=True, exist_ok=True)
paths = sorted(set(subprocess.check_output(
    ['git', 'ls-files', '-co', '--exclude-standard'], text=True).splitlines()))
paths = sorted(set(paths + [p.name for p in root.glob('zz_finalization_*_test.go')]))
manifest = {p: hashlib.sha256((root / p).read_bytes()).hexdigest()
            for p in paths if (root / p).is_file()
            and (p.endswith(('.go', '.md')) or p in ('go.mod', 'go.sum')
                 or p.startswith('testData/'))}
argv = ['gate-lock', lane, 'jet-finalization-lane-2-' + name, '--'] + command
env = dict(os.environ, GATE_LOCK_WAIT='1', GATE_LOCK_COMMAND_TIMEOUT=deadline,
           GATE_LOCK_COMMAND_KILL_AFTER='5')
receipt = {
    'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip(),
    'inputs': manifest, 'command': argv, 'execution_timeout_seconds': int(deadline),
    'kill_after_seconds': 5, 'queued_at': time.time(),
    'harness_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
    'go_version': subprocess.check_output(['go', 'version'], text=True).strip(),
    'go_env': json.loads(subprocess.check_output(
        ['go', 'env', '-json', 'GOOS', 'GOARCH', 'GOMOD', 'GOWORK', 'GOTOOLCHAIN',
         'GOFLAGS'], text=True)),
    'environment': {k: v for k, v in env.items() if k.startswith('GATE_LOCK_')},
}
(art / 'inputs.json').write_text(json.dumps(receipt, indent=2) + '\n')
with (art / 'run.log').open('w') as log:
    rc = subprocess.call(argv, stdout=log, stderr=subprocess.STDOUT, env=env)
receipt.update(exit_code=rc, finished_at=time.time(), inputs_unchanged=all(
    hashlib.sha256((root / p).read_bytes()).hexdigest() == h
    for p, h in manifest.items()))
(art / 'result.json').write_text(json.dumps(receipt, indent=2) + '\n')
print((art / 'run.log').read_text())
print('receipt:', art / 'result.json')
sys.exit(rc)
