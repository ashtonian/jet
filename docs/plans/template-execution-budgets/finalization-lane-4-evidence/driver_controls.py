"""Controlled child-process and receipt tests; these are not Jet performance data."""

import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time

sys.dont_write_bytecode = True


def load(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


matrix = load('run_matrix')
analysis = load('analyze_matrix')


def termination_child(directory):
    signal.signal(signal.SIGTERM, matrix.interrupted)
    signal.signal(signal.SIGINT, matrix.interrupted)
    receipts = {}
    command = [sys.executable, str(directory.parent / 'blocking_child.py'), str(directory / 'state.json')]
    matrix.execute_stage('active', directory, directory, command, dict(os.environ),
                         {'command': command}, receipts, 60)
    later = [sys.executable, '-c', 'from pathlib import Path; Path("later").touch()']
    matrix.execute_stage('later', directory, directory, later, dict(os.environ),
                         {'command': later}, receipts, 10)


def main(directory):
    directory.mkdir(parents=True, exist_ok=False)
    receipts = {}
    count = directory / 'count'
    profile = directory / 'concurrent-mutex.pprof'
    command = [sys.executable, '-c',
               'from pathlib import Path; p=Path("count"); p.write_text(str(int(p.read_text())+1) if p.exists() else "1"); Path("concurrent-mutex.pprof").write_bytes(b"control artifact")']
    environment = matrix.effective_environment(matrix.REPO)
    specification = {'command': command, 'inputs': {'control.go': 'source-a'}, 'environment': environment}

    def execute(spec):
        return matrix.execute_stage('concurrent-contention', directory, directory, command,
                                    dict(os.environ), spec, receipts, 10, [profile])

    assert execute(specification)
    assert execute(specification)
    assert count.read_text() == '1', 'valid stage was executed twice'
    changed = copy.deepcopy(specification)
    changed['environment']['runtime']['GOGC'] = '50'
    assert execute(changed) and count.read_text() == '2', 'runtime setting reused stale result'
    changed['inputs']['control.go'] = 'source-b'
    assert execute(changed) and count.read_text() == '3', 'source change reused stale result'
    profile.unlink()
    assert execute(changed) and count.read_text() == '4', 'missing profile artifact reused'
    assert str(profile) in receipts['concurrent-contention']['artifacts']

    expected = {'inputs': {'control.go': 'source-b'}, 'environment': environment, 'toolchain': 'control'}
    build = {'exit': 0, 'specification': copy.deepcopy(expected),
             'artifacts': {str(profile): hashlib.sha256(profile.read_bytes()).hexdigest()}}
    conformance = copy.deepcopy(build)
    conformance['specification']['binary_sha256'] = hashlib.sha256(profile.read_bytes()).hexdigest()
    assert matrix.prerequisites_valid(build, conformance, expected, profile)
    conformance['specification']['environment']['runtime']['GOMEMLIMIT'] = '32MiB'
    assert not matrix.prerequisites_valid(build, conformance, expected, profile)
    conformance = copy.deepcopy(build)
    conformance['specification']['binary_sha256'] = 'wrong'
    assert not matrix.prerequisites_valid(build, conformance, expected, profile)

    (directory / 'blocking_child.py').write_text('''import json,os,signal,subprocess,sys,time
from pathlib import Path
child=subprocess.Popen([sys.executable,'-c','import time; time.sleep(120)'])
def stopped(number,frame):
    child.wait(timeout=5)
    raise SystemExit(128+number)
signal.signal(signal.SIGTERM,stopped)
signal.signal(signal.SIGINT,stopped)
p=Path(sys.argv[1]); temp=p.with_suffix('.tmp')
temp.write_text(json.dumps({'pid':os.getpid(),'child':child.pid})); temp.replace(p)
time.sleep(120)
''')
    for number in (signal.SIGTERM, signal.SIGINT):
        case = directory / signal.Signals(number).name
        case.mkdir()
        with (case / 'driver.log').open('w') as log:
            driver = subprocess.Popen([sys.executable, str(Path(__file__).resolve()), '--termination-child', str(case)],
                                      stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
            state = None
            try:
                deadline = time.monotonic() + 10
                while not (case / 'state.json').exists():
                    if driver.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError('controlled child did not become ready')
                    time.sleep(0.01)
                state = json.loads((case / 'state.json').read_text())
                os.kill(driver.pid, number)
                assert driver.wait(timeout=15) == 128 + number
                recorded = json.loads((case / 'stages.json').read_text())
                assert recorded['active']['exit'] == 128 + number
                assert 'later' not in recorded and not (case / 'later').exists()
                for pid in state.values():
                    try:
                        os.kill(pid, 0)
                    except ProcessLookupError:
                        continue
                    raise AssertionError('controlled descendant survived: ' + str(pid))
            finally:
                matrix.stop_process(driver)
                if state is not None:
                    try:
                        os.killpg(state['pid'], signal.SIGKILL)
                    except ProcessLookupError:
                        pass

    streams = [('upstream', 'legacy_stream'), ('upstream', 'legacy_owned'),
               ('correctness', 'legacy_stream'), ('correctness', 'legacy_owned')]
    streams += [('candidate', mode) for mode in ('legacy_stream', 'legacy_owned',
                'metered_background_stream', 'metered_live_stream', 'restricted_owned')]
    fixtures = json.loads((matrix.REPO / 'testData/execution_workloads.json').read_text())
    plan = {'streams': streams, 'fixtures': [f['name'] for f in fixtures], 'samples': 6,
            'formats': ['text', 'html'], 'workers': [1, 2, 8, 12], 'shared': [True, False],
            'concurrent_fixtures': ['header_interpolation', 'body_8192', 'nested_silent', 'escape_expansion', 'data_json_256']}
    for group, count in [('latency', 270), ('concurrent', 720)]:
        cells = analysis.expected_cells(group, plan)
        assert len(cells) == count
        results = {key: [{} for _ in range(6)] for key in cells}
        assert analysis.coverage_report(results, cells, 6)['complete']
        first = next(iter(results))
        results[first].pop()
        assert not analysis.coverage_report(results, cells, 6)['complete']
        del results[first]
        assert not analysis.coverage_report(results, cells, 6)['complete']
        assert not analysis.coverage_report({}, cells, 6)['complete']
    plan['fixtures_by_tree'] = {
        tree: plan['fixtures'] + [name for name in matrix.EXTENDED if tree == 'candidate' or name != 'named_map']
        for tree in ('upstream', 'correctness', 'candidate')}
    assert len(analysis.expected_cells('latency', plan)) == 406
    (directory / 'result.json').write_text(json.dumps({'scope': 'driver lifecycle controls only',
        'reuse': True, 'source_environment_binary_identity': True, 'profile_artifacts': True,
        'term_int_cleanup_and_no_later_stage': True, 'incomplete_coverage_denied': True}, indent=2))
    print('DRIVER CONTROLS PASS (not Jet benchmark data)')


if __name__ == '__main__':
    if sys.argv[1] == '--termination-child':
        termination_child(Path(sys.argv[2]))
    else:
        main(Path(sys.argv[1]).resolve())
