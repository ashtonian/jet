"""Analyze the fixed corpus; extended coverage never changes its weighting."""

import json
from pathlib import Path
import re
import statistics
import subprocess
import sys


def expected_cells(group, plan):
    benchmark = 'BenchmarkExecutionModeLatency' if group == 'latency' else 'BenchmarkConcurrentExecutionModes'
    cells = set()
    for tree, mode in plan['streams']:
        fixtures = plan['fixtures'] if group == 'latency' else plan['concurrent_fixtures']
        for fixture in fixtures:
            for format in plan['formats']:
                name = tree + '/' + benchmark + '/' + mode + '/' + fixture + '/' + format
                if group == 'latency':
                    cells.add(name)
                else:
                    for worker in plan['workers']:
                        for shared in plan['shared']:
                            cells.add(name + '/workers_' + str(worker) + '/shared_' + str(shared).lower())
    return cells


def coverage_report(results, expected, samples):
    missing = sorted(expected - set(results))
    extra = sorted(set(results) - expected)
    wrong = {key: len(value) for key, value in results.items() if len(value) != samples}
    return {'complete': not missing and not extra and not wrong, 'expected_cells': len(expected),
            'samples_per_cell': samples, 'missing': missing, 'extra': extra, 'wrong_sample_counts': wrong}


def main():
    work = Path(sys.argv[1]).resolve()
    output = work / 'results'
    stages = json.loads((output / 'stages.json').read_text())
    fixtures = json.loads((work / 'candidate/testData/execution_workloads.json').read_text())
    names = {f['name'] for f in fixtures}
    rows = {}
    lines = {}
    for path in sorted(output.glob('core-*.log')):
        tree = path.name.split('-')[1]
        if stages[path.stem]['exit'] != 0:
            raise RuntimeError('failed core stage: ' + path.stem)
        for line in path.read_text().splitlines():
            fields = line.split()
            if not fields or not fields[0].startswith('BenchmarkExecutionModes/'):
                continue
            match = re.fullmatch(r'BenchmarkExecutionModes/([^/]+)/([^/]+)/(text|html)-2', fields[0])
            if not match or match[2] not in names:
                raise RuntimeError('unexpected core benchmark: ' + fields[0])
            key = (tree, match[1], match[2] + '/' + match[3])
            metrics = {fields[i + 1]: float(fields[i]) for i in range(2, len(fields) - 1, 2)}
            rows.setdefault(key, []).append(metrics)
            lines.setdefault((tree, match[1]), []).append(line.replace('/' + match[1] + '/', '/'))

    medians = {}
    if len(rows) != 270:
        raise RuntimeError('core matrix incomplete: expected 9 source/mode pairs x 30 workloads')
    for key, samples in rows.items():
        if len(samples) != 6:
            raise RuntimeError('need six independent samples: ' + str(key))
        medians[key] = {metric: statistics.median(sample[metric] for sample in samples)
                        for metric in ('ns/op', 'B/op', 'allocs/op')}
    comparisons = [
        ('legacy-vs-upstream', 'upstream', 'legacy_stream', 'candidate', 'legacy_stream'),
        ('legacy-vs-correctness', 'correctness', 'legacy_stream', 'candidate', 'legacy_stream'),
        ('background-vs-upstream', 'upstream', 'legacy_stream', 'candidate', 'metered_background_stream'),
        ('live-vs-upstream', 'upstream', 'legacy_stream', 'candidate', 'metered_live_stream'),
        ('restricted-vs-owned-upstream', 'upstream', 'legacy_owned', 'candidate', 'restricted_owned'),
        ('restricted-vs-owned-candidate', 'candidate', 'legacy_owned', 'candidate', 'restricted_owned'),
    ]
    summary = {}
    for title, base, base_mode, candidate, candidate_mode in comparisons:
        ratios = {}
        for fixture in sorted(names):
            for format in ('text', 'html'):
                name = fixture + '/' + format
                old = medians[(base, base_mode, name)]
                new = medians[(candidate, candidate_mode, name)]
                ratios[name] = {'baseline': old, 'candidate': new,
                                'ratio': new['ns/op'] / old['ns/op']}
        if len(ratios) != 30:
            raise RuntimeError('fixed corpus/weight changed')
        files = []
        for label, tree, mode in [('base', base, base_mode), ('new', candidate, candidate_mode)]:
            path = output / (title + '-' + label + '.txt')
            path.write_text('\n'.join(lines[(tree, mode)]) + '\n')
            files.append(str(path))
        result = subprocess.run(['benchstat'] + files, capture_output=True, text=True, timeout=60)
        (output / (title + '-benchstat.txt')).write_text(result.stdout + result.stderr)
        result.check_returncode()
        median = statistics.median(value['ratio'] for value in ratios.values())
        summary[title] = {'median_ratio': median, 'at_most_20_percent': median <= 1.2,
                          'per_fixture': ratios}
    (output / 'core-analysis.json').write_text(json.dumps(summary, indent=2))
    print(json.dumps({key: {'median_ratio': value['median_ratio'],
                           'at_most_20_percent': value['at_most_20_percent']}
                      for key, value in summary.items()}, indent=2))

    # Missing groups remain explicit, including during an early core-only report.
    plan_path = output / 'coverage-plan.json'
    plans = json.loads(plan_path.read_text()) if plan_path.exists() else {}
    coverage = {}
    for group, benchmark in [('latency', 'BenchmarkExecutionModeLatency/'),
                             ('concurrent', 'BenchmarkConcurrentExecutionModes/')]:
        results = {}
        failed = [name for name, receipt in stages.items() if name.startswith(group + '-') and receipt.get('exit') != 0]
        for path in sorted(output.glob(group + '-*.log')):
            if stages.get(path.stem, {}).get('exit') != 0:
                continue
            tree = path.name.split('-')[1]
            for line in path.read_text().splitlines():
                fields = line.split()
                if fields and fields[0].startswith(benchmark):
                    name = re.sub(r'-[0-9]+$', '', fields[0])
                    results.setdefault(tree + '/' + name, []).append(
                        {fields[i + 1]: float(fields[i]) for i in range(2, len(fields) - 1, 2)})
        if group not in plans:
            coverage[group] = {'complete': False, 'reason': 'group not run'}
        else:
            plan = plans[group]
            coverage[group] = coverage_report(results, expected_cells(group, plan), plan['samples'])
            coverage[group]['failed_or_skipped_stages'] = failed
            coverage[group]['complete'] &= not failed
            if group == 'latency':
                invalid = [name for name, samples in results.items()
                           if any('p50_ns' not in sample or 'p99_ns' not in sample for sample in samples)]
                coverage[group]['missing_percentiles'] = invalid
                coverage[group]['complete'] &= not invalid
            else:
                controls = ['harness-control-' + str(i) for i in range(plan['samples'])]
                missing_controls = [name for name in controls if stages.get(name, {}).get('exit') != 0]
                coverage[group]['missing_harness_controls'] = missing_controls
                coverage[group]['complete'] &= not missing_controls
        (output / (group + '-samples.json')).write_text(json.dumps(results, indent=2))
    (output / 'coverage.json').write_text(json.dumps(coverage, indent=2))
    if not all(value['complete'] for value in coverage.values()) and '--partial' not in sys.argv[2:]:
        raise RuntimeError('final latency/concurrency coverage incomplete; see coverage.json; --partial is diagnostic only')


if __name__ == "__main__":
    main()
