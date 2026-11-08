"""Write matched profile views; report generation is not a cost attribution."""

import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys

sys.dont_write_bytecode = True
import run_matrix as matrix


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main(work):
    output = work / 'results'
    destination = output / 'profile-analysis'
    destination.mkdir(exist_ok=True)
    stages = json.loads((output / 'stages.json').read_text())
    proof = json.loads((work / 'sources.json').read_text())
    reports_path = destination / 'stages.json'
    reports = json.loads(reports_path.read_text()) if reports_path.exists() else {}
    failures = []
    plan_path = output / 'profile-plan.json'
    plan = json.loads(plan_path.read_text()) if plan_path.exists() else {}
    if not plan or not any(label.startswith('profile-concurrent-') for label in plan) or not any(label.startswith('retention-memory-') for label in plan):
        failures.append('full concurrent and live-retention capture plan missing')
    for label, paths in plan.items():
        receipt = stages.get(label, {})
        if receipt.get('exit') != 0 or any(path not in receipt.get('artifacts', {}) for path in paths):
            failures.append(label + ': planned capture incomplete')
    profiles = 0
    toolchain = subprocess.check_output(['go', 'version'], text=True, timeout=10).strip()
    for label, receipt in stages.items():
        if not (label.startswith('profile-') or label.startswith('retention-memory-')):
            continue
        if receipt.get('exit') != 0:
            failures.append(label + ': incomplete capture')
            continue
        specification = receipt['specification']
        tree = specification['tree']
        binary = matrix.binary_file(work, proof, tree)
        if digest(binary) != specification['binary_sha256']:
            failures.append(label + ': binary identity changed')
            continue
        source = matrix.source_directory(work, proof, tree)
        if matrix.manifest(source) != specification['inputs']:
            failures.append(label + ': frozen source changed')
            continue
        module = (source / 'go.mod').read_text().splitlines()[0].split()[1]
        source_path = str(source) + ':' + str(Path(specification['environment']['go']['GOROOT']) / 'src')
        captured = receipt.get('artifacts', {})
        for name, expected in captured.items():
            profile = Path(name)
            if profile.suffix != '.pprof':
                continue
            if not profile.exists() or digest(profile) != expected:
                failures.append(name + ': profile identity changed')
                continue
            profiles += 1
            if profile.name.endswith('-cpu.pprof'):
                views = [('cpu-flat', ['-top']), ('cpu-cumulative', ['-top', '-cum']),
                         ('cpu-call', ['-tree']), ('cpu-callees', ['-peek', 'github.com/(CloudyKit|ashtonian)/jet']),
                         ('cpu-source', ['-list', 'printValue|dataPrinter|checkedWriter|admitCached|admitInput|startCompilation|checkpoint|reserveWork|executeList'])]
            elif profile.name.endswith('-mutex.pprof'):
                views = [('mutex', ['-top']), ('mutex-call', ['-tree'])]
            elif profile.name.endswith('-block.pprof'):
                views = [('block', ['-top']), ('block-call', ['-tree'])]
            else:
                views = [(kind, ['-top', '-' + kind]) for kind in
                         ('alloc_space', 'alloc_objects', 'inuse_space', 'inuse_objects')]
                views += [('alloc-call', ['-tree', '-alloc_space']),
                          ('alloc-source', ['-alloc_space', '-list',
                            'admitInput|clone|admitCached|newParseAdmission|printValue|outputWriter|checkedOutput|RenderRestricted|recover'])]
            for view, flags in views:
                key = label + '-' + profile.stem + '-' + view
                target = destination / (key + '.log')
                command = ['go', 'tool', 'pprof', '-nodecount=40', '-source_path=' + source_path,
                           '-trim_path=' + module + '/', *flags, str(binary), str(profile)]
                inputs = {'command': command, 'binary_sha256': digest(binary), 'profile_sha256': expected,
                          'source_inputs': specification['inputs'], 'environment': specification['environment'],
                          'toolchain': toolchain}
                previous = reports.get(key, {})
                if (previous.get('specification') == inputs and previous.get('exit') == 1
                        and target.exists() and matrix.valid_artifacts(previous)
                        and 'no matches found for regexp:' in target.read_text()):
                    passed = False
                else:
                    passed = matrix.execute_stage(key, destination, source, command, dict(os.environ),
                                                  inputs, reports, 30)
                text = target.read_text()
                no_match = 'no matches found for regexp:' in text
                source_missing = view.endswith('source') and bool(re.search(r'could not find file|could not open|no source information', text, re.I))
                reports[key].update(source_unavailable=source_missing,
                                    no_sampled_function_match=no_match,
                                    sparse_or_empty='0% of 0 total' in text or 'No samples were found' in text)
                reports_path.write_text(json.dumps(reports, indent=2))
                if (not passed and not no_match) or source_missing:
                    failures.append(key)
    summary = {'profiles': profiles, 'reports': len(reports), 'failures': failures,
               'interpretation': 'Read the generated call/source and bytes/objects views; sparse heap is not retention proof. Capture logs contain native time/RSS observations. Instrumented timings are excluded from release comparisons.'}
    (destination / 'summary.json').write_text(json.dumps(summary, indent=2))
    print(json.dumps(summary, indent=2))
    if failures or profiles == 0:
        raise RuntimeError('profile coverage incomplete; inspect summary and receipts')


if __name__ == '__main__':
    signal.signal(signal.SIGTERM, matrix.interrupted)
    signal.signal(signal.SIGINT, matrix.interrupted)
    main(Path(sys.argv[1]).resolve())
