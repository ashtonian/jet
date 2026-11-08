"""Summarize bounded observations, retaining completed-before-request samples."""

import hashlib
import json
from pathlib import Path
import re
import statistics
import sys


def distribution(values):
    values = sorted(values)
    if not values:
        return None
    return {'samples': len(values), 'p50_ns': statistics.median(values),
            'p99_ns': values[(len(values) - 1) * 99 // 100], 'max_ns': values[-1]}


def main(work, callback):
    results = work / 'results'
    stages = json.loads((results / 'stages.json').read_text())
    native = results / 'cancellation-candidate.log'
    if stages['cancellation-candidate'].get('exit') != 0:
        raise RuntimeError('native cancellation stage did not pass')
    receipt = json.loads((callback.parent / 'receipt.json').read_text())
    if receipt.get('exit') != 0 or not receipt.get('inputs_unchanged'):
        raise RuntimeError('callback distribution stage did not pass on unchanged inputs')
    inputs = stages['cancellation-candidate']['specification']['inputs']
    for name, digest in inputs.items():
        if name.endswith('.go') or name in ('go.mod', 'go.sum') or name.startswith('testData/'):
            if receipt['frozen_inputs_after_run'].get(name) != digest:
                raise RuntimeError('callback/native source mismatch: ' + name)
    rows = {}
    active = None
    for line in native.read_text().splitlines():
        match = re.match(r'=== RUN\s+TestExecutionCancellationMeasurements/(\w+)$', line)
        if match:
            active = match[1]
        elif '"overshoot_ns":' in line:
            rows.setdefault(active, []).append(json.loads(line[line.index('{'):]))
    expected = {'interpreter_loop', 'prepaid_builtin_operation', 'compile_admission', 'input_admission'}
    if set(rows) != expected or any(len(samples) != 128 for samples in rows.values()):
        raise RuntimeError('native sample coverage is incomplete')
    observations = {}
    for name, samples in rows.items():
        observations[name] = {
            'all': distribution([sample['overshoot_ns'] for sample in samples]),
            'canceled_only': distribution([sample['overshoot_ns'] for sample in samples if sample['canceled']]),
            'completed_before_request': sum(sample['overshoot_ns'] < 0 for sample in samples),
            'uncanceled_samples': sum(not sample['canceled'] for sample in samples),
            'uncanceled_nonnegative_overshoot': sum(not sample['canceled'] and sample['overshoot_ns'] >= 0 for sample in samples),
            'post_return_checks': sum(sample['post_return_checks'] for sample in samples)}
    callbacks = {}
    for line in callback.read_text().splitlines():
        match = re.match(r'=== RUN\s+TestExecutionCallbackCancellationOwnership/cooperative_(true|false)$', line)
        if match:
            active = match[1]
        elif 'callback cancellation return_ns=' in line:
            callbacks.setdefault(active, []).append(int(line.split('return_ns=')[1]))
    if set(callbacks) != {'true', 'false'} or any(len(values) != 128 for values in callbacks.values()):
        raise RuntimeError('callback sample coverage is incomplete')
    report = {
        'quantiles': 'Median and sorted[(n-1)*99//100]; p99 is the observed order statistic, not a confidence bound.',
        'native': observations, 'callbacks': {key: distribution(value) for key, value in callbacks.items()},
        'limits': ['Native timings include the synchronized measurement context; deterministic controls are separate.',
                   'Negative overshoots mean completion preceded the cancellation request.',
                   'An uncanceled return alone does not establish that its return timestamp preceded the request.',
                   'The request timestamp precedes cancel(); completion can race between a final checkpoint and return. Nonnegative overshoot with nil error alone proves no missed observable cancellation.',
                   'Blocked callback observations explicitly release and join the callback; they do not establish preemption.'],
        'raw_sha256': {str(path): hashlib.sha256(path.read_bytes()).hexdigest() for path in (native, callback)}}
    (results / 'cancellation-distributions.json').write_text(json.dumps(report, indent=2))
    print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main(Path(sys.argv[1]).resolve(), Path(sys.argv[2]).resolve())
