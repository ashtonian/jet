"""Run under installed gate-lock run-exclusive; prepare only archives source."""

import copy
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tarfile
import time


REPO = Path(__file__).resolve().parents[4]
UPSTREAM = '202464510d5b57f0afa9c5563f75babc1616682c'
CORRECTNESS = '822cf702300c6226fa56a3fcf2f8466dc6b9b712'
COMMON = ['execution_workloads_test.go', 'execution_measurements_test.go',
          'testData/execution_workloads.json']
RUNTIME_ENV = ('GOGC', 'GOMEMLIMIT', 'GODEBUG', 'GOFLAGS', 'GOTOOLCHAIN', 'CGO_ENABLED')
GO_ENV = ('GOOS', 'GOARCH', 'GOROOT', 'GOVERSION', 'GOFLAGS', 'GO111MODULE', 'GOMOD', 'GOWORK',
          'GOENV', 'GOTOOLDIR', 'CGO_ENABLED', 'GOEXPERIMENT',
          'GOAMD64', 'GOARM64', 'GOTOOLCHAIN', 'CC', 'CXX', 'CGO_CFLAGS', 'CGO_CPPFLAGS',
          'CGO_CXXFLAGS', 'CGO_LDFLAGS')
EXTENDED = ['data_json_256', 'data_unused_256', 'named_map', 'named_slice', 'array',
            'typed_nil', 'builtin_alias', 'catch_diagnostic']
RETENTION_PHASES = ['post_two_gc', 'warm_small', 'after_large', 'large_then_small', 'large_then_small_after_gc']


def interrupted(signum, _frame):
    raise SystemExit(128 + signum)


def stop_process(process):
    if process.poll() is not None:
        return
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        process.wait(timeout=2)


def effective_environment(directory):
    return {'runtime': {key: os.environ.get(key) for key in RUNTIME_ENV},
            'go': json.loads(subprocess.check_output(['go', 'env', '-json'] + list(GO_ENV),
                                                     cwd=directory, text=True, timeout=15))}


def valid_artifacts(receipt):
    return bool(receipt.get('artifacts')) and all(
        Path(path).exists() and hashlib.sha256(Path(path).read_bytes()).hexdigest() == digest
        for path, digest in receipt['artifacts'].items())


def prerequisites_valid(build, conformance, expected, binary):
    return (build.get('exit') == 0 and conformance.get('exit') == 0 and binary.exists()
            and all(receipt.get('specification', {}).get(key) == value
                    for receipt in (build, conformance) for key, value in expected.items())
            and valid_artifacts(build) and valid_artifacts(conformance)
            and conformance.get('specification', {}).get('binary_sha256') == hashlib.sha256(binary.read_bytes()).hexdigest())


def execute_stage(label, output, cwd, command, env, specification, receipts, seconds, artifacts=()):
    previous = receipts.get(label, {})
    old_specification = dict(previous.get('specification', {}))
    reuse_specification = dict(specification)
    if label.startswith('build-'):
        old_specification.pop('driver_sha256', None)
        reuse_specification.pop('driver_sha256', None)
    if old_specification == reuse_specification and previous.get('exit') == 0 and valid_artifacts(previous):
        print('REUSE ' + label, flush=True)
        return True
    print('START ' + label, flush=True)
    receipt = {'specification': copy.deepcopy(specification), 'start': time.time(), 'deadline_seconds': seconds,
               'cleanup_seconds': 7}
    receipts[label] = receipt
    (output / 'stages.json').write_text(json.dumps(receipts, indent=2))
    interruption = None
    with (output / (label + '.log')).open('w') as log:
        process = subprocess.Popen(command, cwd=cwd, env=env, stdout=log,
                                   stderr=subprocess.STDOUT, start_new_session=True)
        try:
            code = process.wait(timeout=seconds)
        except subprocess.TimeoutExpired:
            stop_process(process)
            code = 124
        except (KeyboardInterrupt, SystemExit) as error:
            stop_process(process)
            code = error.code if isinstance(error, SystemExit) else 130
            interruption = error
    receipt.update(exit=code, end=time.time())
    paths = [output / (label + '.log')] + list(artifacts)
    receipt['artifacts'] = {str(path): hashlib.sha256(path.read_bytes()).hexdigest()
                            for path in paths if path.exists()}
    if code == 0 and any(not path.exists() for path in artifacts):
        receipt.update(exit=65, reason='declared artifact missing')
    (output / 'stages.json').write_text(json.dumps(receipts, indent=2))
    print('END ' + label + ' exit=' + str(receipt['exit']), flush=True)
    if interruption is not None:
        raise interruption
    return receipt['exit'] == 0


def manifest(directory):
    return {str(p.relative_to(directory)): hashlib.sha256(p.read_bytes()).hexdigest()
            for p in sorted(directory.rglob('*')) if p.is_file() and
            (p.suffix in ('.go', '.json') or p.name in ('go.mod', 'go.sum'))}


def prepare(work, candidate):
    work.mkdir(parents=True, exist_ok=True)
    if any(work.iterdir()):
        raise RuntimeError('prepare requires an empty artifact directory')
    sources = {'upstream': UPSTREAM, 'correctness': CORRECTNESS, 'candidate': candidate}
    proof = {}
    for name, revision in sources.items():
        destination = work / name
        destination.mkdir()
        archive = work / (name + '.tar')
        with archive.open('wb') as output:
            subprocess.run(['git', 'archive', revision], cwd=REPO, stdout=output,
                           check=True, timeout=60)
        with tarfile.open(archive) as source:
            source.extractall(destination, filter='data')
        archive.unlink()
        for filename in COMMON:
            (destination / filename).write_bytes((REPO / filename).read_bytes())
        proof[name] = {'revision': revision, 'inputs': manifest(destination)}
    if not (work / 'candidate/execution_measurement_modes_test.go').exists():
        raise RuntimeError('candidate revision must contain the candidate measurement modes')
    proof['harness'] = {filename: hashlib.sha256((REPO / filename).read_bytes()).hexdigest()
                        for filename in COMMON}
    proof['toolchain'] = subprocess.check_output(['go', 'version'], text=True).strip()
    proof['environment'] = effective_environment(REPO)
    proof['driver_sha256'] = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    (work / 'sources.json').write_text(json.dumps(proof, indent=2))


def run(work, group):
    output = work / 'results'
    output.mkdir(exist_ok=True)
    proof = json.loads((work / 'sources.json').read_text())
    version = subprocess.check_output(['go', 'version'], text=True).strip()
    if version != proof['toolchain']:
        raise RuntimeError('toolchain changed since source preparation')
    for tree in ('upstream', 'correctness', 'candidate'):
        if manifest(work / tree) != proof[tree]['inputs']:
            raise RuntimeError('frozen source inputs changed: ' + tree)
    fixtures = json.loads((work / 'candidate/testData/execution_workloads.json').read_text())
    core = '(' + '|'.join(f['name'] for f in fixtures) + ')'
    extended = '(' + '|'.join(EXTENDED) + ')'
    stages = output / 'stages.json'
    receipts = json.loads(stages.read_text()) if stages.exists() else {}
    failures = []
    environments = {tree: effective_environment(work / tree) for tree in ('upstream', 'correctness', 'candidate')}

    def stage(label, tree, arguments, seconds=600, cores=2):
        env = dict(os.environ, GOMAXPROCS=str(cores))
        command = arguments
        specification = {'command': command, 'tree': tree, 'GOMAXPROCS': str(cores),
                         'inputs': proof[tree]['inputs'], 'toolchain': proof['toolchain'],
                         'driver_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                         'environment': environments[tree]}
        artifacts = []
        if label.startswith('build-'):
            artifacts.append(Path(binary(tree)))
        else:
            specification['binary_sha256'] = hashlib.sha256(Path(binary(tree)).read_bytes()).hexdigest()
        for argument in command:
            if any(argument.startswith('-test.' + name + 'profile=') for name in ('cpu', 'mem', 'mutex', 'block')):
                artifacts.append(Path(argument.split('=', 1)[1]))
            if argument.startswith('JET_RETENTION_PROFILE_DIR='):
                directory = Path(argument.split('=', 1)[1])
                artifacts.extend(directory / (phase + '.pprof') for phase in RETENTION_PHASES)
        passed = execute_stage(label, output, work / tree, command, env, specification, receipts, seconds, artifacts)
        if not passed:
            failures.append(label)
        return passed

    def finish():
        if failures:
            raise RuntimeError('failed or skipped stages: ' + ', '.join(failures))

    def ready(tree, label):
        build = receipts.get('build-' + tree, {})
        conformance = receipts.get('conformance-' + tree, {})
        path = Path(binary(tree))
        expected = {'inputs': proof[tree]['inputs'], 'toolchain': proof['toolchain'], 'environment': environments[tree]}
        if prerequisites_valid(build, conformance, expected, path):
            return True
        receipts[label] = {'skipped': 'build/conformance prerequisite incomplete', 'tree': tree}
        stages.write_text(json.dumps(receipts, indent=2))
        failures.append(label)
        print('SKIP ' + label, flush=True)
        return False

    def binary(tree):
        return str(output / (tree + '.test'))

    def benchmark(tree, pattern, duration='1s', cores=2):
        command = [binary(tree), '-test.run=^$', '-test.bench=' + pattern,
                   '-test.benchtime=' + duration, '-test.count=1', '-test.cpu=' + str(cores),
                   '-test.benchmem', '-test.timeout=9m']
        if sys.platform == 'darwin':
            command = ['/usr/bin/time', '-l'] + command
        return command

    if group == 'build':
        for tree in ('upstream', 'correctness', 'candidate'):
            if not stage('build-' + tree, tree, ['go', 'test', '-p=1', '-c', '-o', binary(tree), '.'], 180):
                receipts['conformance-' + tree] = {'skipped': 'build failed', 'tree': tree}
                stages.write_text(json.dumps(receipts, indent=2))
                continue
            pattern = '^TestExecutionModeConformance$'
            if tree != 'candidate':
                names = [f['name'] for f in fixtures] + [name for name in EXTENDED if name != 'named_map']
                pattern += '/[^/]+/^(' + '|'.join(names) + ')$/'
            stage('conformance-' + tree, tree, [binary(tree), '-test.run=' + pattern, '-test.timeout=90s'], 120)
        finish()
        return
    if group in ('core', 'extended', 'phases', 'latency', 'concurrent'):
        streams = [('upstream', 'legacy_stream'), ('candidate', 'legacy_stream'),
                   ('correctness', 'legacy_stream'), ('candidate', 'metered_background_stream'),
                   ('upstream', 'legacy_owned'), ('candidate', 'restricted_owned'),
                   ('correctness', 'legacy_owned'), ('candidate', 'metered_live_stream'),
                   ('candidate', 'legacy_owned')]
        plan_path = output / 'coverage-plan.json'
        plan = json.loads(plan_path.read_text()) if plan_path.exists() else {}
        plan[group] = {'streams': streams, 'samples': 6, 'formats': ['text', 'html'],
                       'fixtures': [f['name'] for f in fixtures],
                       'workers': sorted({1, 2, 8, int(os.environ['GOMAXPROCS'])}),
                       'shared': [True, False], 'concurrent_fixtures': [
                           'header_interpolation', 'body_8192', 'nested_silent', 'escape_expansion', 'data_json_256']}
        if group == 'latency':
            plan[group]['fixtures_by_tree'] = {
                tree: [f['name'] for f in fixtures] + [name for name in EXTENDED if tree == 'candidate' or name != 'named_map']
                for tree in ('upstream', 'correctness', 'candidate')}
        plan_path.write_text(json.dumps(plan, indent=2))
        for sample in range(6):
            order = streams[sample % len(streams):] + streams[:sample % len(streams)]
            for tree, mode in order:
                label = group + '-' + tree + '-' + mode + '-' + str(sample)
                if not ready(tree, label):
                    continue
                workloads = core if group == 'core' else extended
                # The inherited named-map panic is retained as a failing control;
                # its upstream timing is undefined, while all candidate cases run.
                if group == 'extended' and tree != 'candidate':
                    workloads = extended.replace('|named_map', '')
                pattern = '^BenchmarkExecutionModes$/^' + mode + '$/^' + workloads + '$/'
                if group == 'phases':
                    pattern = '^BenchmarkExecutionPhases$/^' + mode + '$/'
                elif group == 'latency':
                    pattern = '^BenchmarkExecutionModeLatency$/^' + mode + '$/'
                    if tree != 'candidate':
                        names = plan[group]['fixtures_by_tree'][tree]
                        pattern += '^(' + '|'.join(names) + ')$/'
                elif group == 'concurrent':
                    pattern = '^BenchmarkConcurrentExecutionModes$/^' + mode + '$/'
                cores = int(os.environ['GOMAXPROCS']) if group == 'concurrent' else 2
                stage(label, tree, benchmark(tree, pattern, cores=cores), 900, cores)
            if group == 'concurrent':
                if ready('candidate', 'harness-control-' + str(sample)):
                    stage('harness-control-' + str(sample), 'candidate',
                          benchmark('candidate', '^BenchmarkExecutionHarnessControl$', cores=cores), 90, cores)
        finish()
        return
    if group == 'profiles':
        modes = [('upstream', 'legacy_stream'), ('upstream', 'legacy_owned'),
                 ('correctness', 'legacy_stream'), ('correctness', 'legacy_owned')]
        modes += [('candidate', name) for name in ('legacy_stream', 'legacy_owned',
                  'metered_background_stream', 'metered_live_stream', 'restricted_owned')]
        plan_path = output / 'profile-plan.json'
        plan = json.loads(plan_path.read_text()) if plan_path.exists() else {}
        for tree, mode in modes:
            for prefix in ('profile-', 'profile-concurrent-'):
                label = prefix + tree + '-' + mode
                plan[label] = [str(output / (label + '-' + kind + '.pprof')) for kind in ('cpu', 'heap', 'mutex', 'block')]
        plan_path.write_text(json.dumps(plan, indent=2))
        for tree, mode in modes:
            label = 'profile-' + tree + '-' + mode
            if not ready(tree, label):
                continue
            command = benchmark(tree, '^BenchmarkExecutionModes$/^' + mode + '$/^(header_interpolation|nested_silent|escape_expansion|include|block_content)$/^html$', '1s')
            for flag, suffix in [('cpuprofile', 'cpu'), ('memprofile', 'heap'),
                                 ('mutexprofile', 'mutex'), ('blockprofile', 'block')]:
                command.append('-test.' + flag + '=' + str(output / (label + '-' + suffix + '.pprof')))
            stage(label, tree, command, 120)
        cores = int(os.environ['GOMAXPROCS'])
        concurrent = [('upstream', 'legacy_stream'), ('upstream', 'legacy_owned'),
                      ('correctness', 'legacy_stream'), ('correctness', 'legacy_owned')]
        concurrent += [('candidate', mode) for mode in ('legacy_stream', 'legacy_owned',
                       'metered_background_stream', 'metered_live_stream', 'restricted_owned')]
        for tree, mode in concurrent:
            label = 'profile-concurrent-' + tree + '-' + mode
            if not ready(tree, label):
                continue
            command = benchmark(tree, '^BenchmarkConcurrentExecutionModes$/^' + mode + '$/^(header_interpolation|escape_expansion)$/^html$/^workers_' + str(cores) + '$/', '1s', cores)
            for flag, suffix in [('cpuprofile', 'cpu'), ('memprofile', 'heap'),
                                 ('mutexprofile', 'mutex'), ('blockprofile', 'block')]:
                command.append('-test.' + flag + '=' + str(output / (label + '-' + suffix + '.pprof')))
            stage(label, tree, command, 180, cores)
        finish()
        return
    if group == 'diagnostics':
        plan_path = output / 'profile-plan.json'
        plan = json.loads(plan_path.read_text()) if plan_path.exists() else {}
        for tree in ('upstream', 'correctness', 'candidate'):
            modes = ['legacy_stream', 'legacy_owned']
            if tree == 'candidate':
                modes += ['metered_background_stream', 'metered_live_stream', 'restricted_owned']
            for mode in modes:
                label = 'retention-' + tree + '-' + mode
                if not ready(tree, label):
                    continue
                command = ['env', 'JET_RETENTION_MEASUREMENTS=1', binary(tree),
                           '-test.run=^TestExecutionRetentionMeasurements$/^' + mode + '$',
                           '-test.v', '-test.timeout=60s']
                if sys.platform == 'darwin':
                    command = ['/usr/bin/time', '-l'] + command
                stage(label, tree, command, 90)
                label = 'retention-memory-' + tree + '-' + mode
                directory = output / label
                directory.mkdir(exist_ok=True)
                plan[label] = [str(directory / (phase + '.pprof')) for phase in RETENTION_PHASES]
                plan_path.write_text(json.dumps(plan, indent=2))
                command = ['env', 'JET_RETENTION_MEASUREMENTS=1', 'JET_RETENTION_PROFILE_DIR=' + str(directory),
                           binary(tree), '-test.run=^TestExecutionRetentionMeasurements$/^' + mode + '$',
                           '-test.memprofilerate=1', '-test.v', '-test.timeout=60s']
                if sys.platform == 'darwin':
                    command = ['/usr/bin/time', '-l'] + command
                stage(label, tree, command, 90)
        command = ['env', 'JET_CANCELLATION_SAMPLES=128', binary('candidate'),
                   '-test.run=^TestExecution(Cancellation|Callback|MeasurementBounds)',
                   '-test.v', '-test.timeout=120s']
        if ready('candidate', 'cancellation-candidate'):
            stage('cancellation-candidate', 'candidate', command, 150)
        finish()
        return
    raise RuntimeError('unknown group: ' + group)


if __name__ == '__main__':
    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    work = Path(sys.argv[1]).resolve()
    if sys.argv[2] == 'prepare':
        prepare(work, sys.argv[3])
    else:
        run(work, sys.argv[2])
