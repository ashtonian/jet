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


def source_inputs(focused=False):
    names = subprocess.check_output(['git', 'ls-files', '-co', '--exclude-standard', '-z'], cwd=REPO).split(b'\0')
    return {os.fsdecode(name): hashlib.sha256((REPO / os.fsdecode(name)).read_bytes()).hexdigest()
            for name in names if name and (REPO / os.fsdecode(name)).is_file()
            and (not focused or Path(os.fsdecode(name)).suffix in ('.go', '.json')
                 or Path(os.fsdecode(name)).name in ('go.mod', 'go.sum')
                 or os.fsdecode(name).startswith('testData/'))}


def go_test(command):
    return len(command) > 1 and Path(command[0]).name == 'go' and command[1] == 'test'


def admitted(directory, command):
    receipt = {'command': command, 'cwd': str(REPO), 'start': time.time(),
               'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=REPO, text=True).strip(),
               'inputs': source_inputs(focused=go_test(command)),
               'toolchain': subprocess.check_output(['go', 'version'], text=True).strip(),
               'environment': {key: os.environ.get(key) for key in
                               ('GOMAXPROCS', 'GOFLAGS', 'GOTOOLCHAIN', 'GATE_LOCK_COMMAND_TIMEOUT',
                                'GATE_LOCK_COMMAND_KILL_AFTER')}}
    write(directory / 'admitted.json', receipt)
    result = subprocess.run(command, cwd=REPO)
    receipt.update(exit=result.returncode, end=time.time())
    after = source_inputs(focused=go_test(command))
    receipt['changed_inputs'] = sorted(name for name in set(after) | set(receipt['inputs'])
                                      if after.get(name) != receipt['inputs'].get(name))
    if result.returncode == 0 and receipt['changed_inputs']:
        receipt.update(exit=65, reason='source changed during admitted execution')
    write(directory / 'admitted.json', receipt)
    return receipt['exit']


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
    wrapped = [sys.executable, str(Path(__file__).resolve()), '--admitted', str(directory)] + command
    if lane == 'run-focused':
        wrapped = command
    args = ['gate-lock', lane, 'jet-l4-' + label, '--'] + wrapped
    receipt = {'command': args, 'queued': time.time(), 'deadline_seconds': int(seconds),
               'cleanup_seconds': 10}
    if lane == 'run-focused':
        receipt['frozen_inputs_before_queue'] = source_inputs(focused=True)
        receipt['toolchain'] = subprocess.check_output([command[0], 'version'], text=True).strip()
        receipt['environment'] = json.loads(subprocess.check_output(
            [command[0], 'env', '-json', 'GOOS', 'GOARCH', 'GOFLAGS', 'CGO_ENABLED'], text=True))
        receipt['runtime_environment'] = {key: os.environ.get(key) for key in ('GOGC', 'GOMEMLIMIT', 'GODEBUG')}
    write(directory / 'receipt.json', receipt)
    with (directory / 'run.log').open('w') as log:
        result = subprocess.run(args, cwd=REPO, env=environment, stdout=log, stderr=subprocess.STDOUT)
    receipt.update(exit=result.returncode, finished=time.time())
    if lane == 'run-focused':
        receipt['frozen_inputs_after_run'] = source_inputs(focused=True)
        receipt['inputs_unchanged'] = receipt['frozen_inputs_before_queue'] == receipt['frozen_inputs_after_run']
        if result.returncode == 0 and not receipt['inputs_unchanged']:
            receipt.update(exit=65, reason='focused source changed while queued or running')
    write(directory / 'receipt.json', receipt)
    print((directory / 'run.log').read_text()[-6000:])
    print('RECEIPT', directory / 'receipt.json')
    sys.exit(receipt['exit'])
