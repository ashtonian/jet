"""Bound an installed host-gate stage and record inputs after admission."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time


REPO = Path(__file__).resolve().parents[4]


def write(path, value):
    path.write_text(json.dumps(value, indent=2))


def source_inputs():
    names = subprocess.check_output(['git', 'ls-files', '-co', '--exclude-standard', '-z'], cwd=REPO).split(b'\0')
    return {os.fsdecode(name): hashlib.sha256((REPO / os.fsdecode(name)).read_bytes()).hexdigest()
            for name in names if name and (REPO / os.fsdecode(name)).is_file()}


def admitted(directory, command):
    receipt = {'command': command, 'cwd': str(REPO), 'start': time.time(),
               'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=REPO, text=True).strip(),
               'inputs': source_inputs(),
               'toolchain': subprocess.check_output(['go', 'version'], text=True).strip(),
               'environment': {key: os.environ.get(key) for key in
                               ('GOMAXPROCS', 'GOFLAGS', 'GOTOOLCHAIN', 'GATE_LOCK_COMMAND_TIMEOUT',
                                'GATE_LOCK_COMMAND_KILL_AFTER')}}
    write(directory / 'admitted.json', receipt)
    result = subprocess.run(command, cwd=REPO)
    receipt.update(exit=result.returncode, end=time.time())
    after = source_inputs()
    receipt['changed_inputs'] = sorted(name for name in set(after) | set(receipt['inputs'])
                                      if after.get(name) != receipt['inputs'].get(name))
    write(directory / 'admitted.json', receipt)
    return result.returncode


if __name__ == '__main__':
    if sys.argv[1] == '--admitted':
        sys.exit(admitted(Path(sys.argv[2]), sys.argv[3:]))
    output, label, lane, seconds, *command = sys.argv[1:]
    if not command or lane not in ('run', 'run-check', 'run-focused', 'run-exclusive'):
        raise SystemExit('usage: run_stage.py OUTPUT LABEL POOL SECONDS COMMAND...')
    directory = Path(output).resolve() / label
    directory.mkdir(parents=True, exist_ok=False)
    environment = dict(os.environ, GATE_LOCK_WAIT='1', GATE_LOCK_COMMAND_TIMEOUT=seconds,
                       GATE_LOCK_COMMAND_KILL_AFTER='10')
    args = ['gate-lock', lane, 'jet-l4-' + label, '--', sys.executable,
            str(Path(__file__).resolve()), '--admitted', str(directory)] + command
    receipt = {'command': args, 'queued': time.time(), 'deadline_seconds': int(seconds),
               'cleanup_seconds': 10}
    write(directory / 'receipt.json', receipt)
    with (directory / 'run.log').open('w') as log:
        result = subprocess.run(args, cwd=REPO, env=environment, stdout=log, stderr=subprocess.STDOUT)
    receipt.update(exit=result.returncode, finished=time.time())
    write(directory / 'receipt.json', receipt)
    print((directory / 'run.log').read_text()[-6000:])
    print('RECEIPT', directory / 'receipt.json')
    sys.exit(result.returncode)
