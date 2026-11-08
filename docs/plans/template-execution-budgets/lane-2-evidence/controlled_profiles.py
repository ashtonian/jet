import difflib
import hashlib
import json
from pathlib import Path
import signal
import subprocess
import sys

root = Path.cwd()
art = root / '.git/lane-2-validation'

def interrupt(signum, frame):
    raise KeyboardInterrupt()

signal.signal(signal.SIGTERM, interrupt)

controls = [
    ('value-control', 'execution_values.go',
     'st.reserve(&st.execution.values, uint64(st.execution.options.MaxValueBytes), uint64(size), "value", ErrValueLimit)',
     'st.reserve(&st.execution.values, 0, uint64(size), "value", ErrValueLimit)',
     '^TestExecutionBudgetProfile$/(doubling|repeat|replace|split)$'),
    ('output-control', 'execution_output.go',
     'st.reserve(&st.execution.output, uint64(st.execution.options.MaxOutputBytes), uint64(len(b)), "output", ErrOutputLimit)',
     'st.reserve(&st.execution.output, 0, uint64(len(b)), "output", ErrOutputLimit)',
     '^TestExecutionBudgetProfile$/(nestedTry|rollback|discard)$'),
]

for name, filename, before, after, selection in controls:
    if len(sys.argv) > 1:
        name += "-" + sys.argv[1]
    source = root / filename
    original = source.read_text()
    assert original.count(before) == 1
    variant = original.replace(before, after, 1)
    out = art / name
    out.mkdir(exist_ok=True)
    (out / 'enabled-source.go').write_text(original)
    (out / 'control-source.go').write_text(variant)
    (out / 'control.patch').write_text(''.join(difflib.unified_diff(original.splitlines(True), variant.splitlines(True), fromfile=filename, tofile=filename)))
    command = ['python3', str(art / 'run.py'), name, 'run-focused', '120',
               'go', 'test', '-p=1', '-parallel=2', '-timeout=60s', '-count=1', '-v',
               '-run', selection, '-memprofilerate=1',
               '-memprofile=' + str(out / 'allocs.pprof'),
               '-o=' + str(out / 'jet.test'), '-outputdir=' + str(out),
               '-exec=/usr/bin/time -l', '.']
    source.write_text(variant)
    try:
        rc = subprocess.call(command)
    finally:
        if source.read_text() != variant:
            raise SystemExit('controlled source changed externally; original is preserved in ' + str(out))
        source.write_text(original)
    proof = dict(expected_exit=1, actual_exit=rc,
                 enabled_sha256=hashlib.sha256(original.encode()).hexdigest(),
                 control_sha256=hashlib.sha256(variant.encode()).hexdigest(),
                 restored=source.read_text() == original,
                 fixtures='Fixed maximum produced string: 131072 bytes; no OOM probe')
    (out / 'control-result.json').write_text(json.dumps(proof, indent=2) + '\n')
    if rc != 1:
        raise SystemExit('unexpected controlled-baseline result: ' + str(rc))
