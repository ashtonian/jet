import pathlib, shutil, subprocess, sys, tarfile

assets = pathlib.Path(__file__).resolve().parent
if len(sys.argv) != 5:
    raise SystemExit('usage: prepare.py NEW_WORK_DIR BASELINE_REPO CORRECTNESS_REPO CANDIDATE_REPO')
work = pathlib.Path(sys.argv[1]).resolve()
if work.exists() and any(work.iterdir()):
    raise SystemExit('work directory must be empty')
work.mkdir(parents=True, exist_ok=True)
sources = {
    'upstream': (sys.argv[2], '202464510d5b57f0afa9c5563f75babc1616682c'),
    'correctness': (sys.argv[3], '822cf702300c6226fa56a3fcf2f8466dc6b9b712'),
    'candidate': (sys.argv[4], '0e277b3017b4afae1f984663c4e9244b4de0b290'),
}
for name, (repo, sha) in sources.items():
    target = work / name
    target.mkdir()
    archive = work / (name + '.tar')
    with archive.open('wb') as output:
        subprocess.run(['git', '-C', repo, 'archive', '--format=tar', sha], stdout=output, check=True, timeout=60)
    with tarfile.open(archive) as source:
        source.extractall(target, filter='data')
    archive.unlink()
shutil.copytree(work / 'candidate', work / 'probes')
for tree in ['upstream', 'correctness', 'candidate']:
    target = work / tree
    shutil.copyfile(assets / 'benchmark.go.txt', target / 'final_review_bench_test.go')
    if tree != 'candidate':
        shutil.copyfile(work / 'candidate' / 'execution_workloads_test.go', target / 'execution_workloads_test.go')
        shutil.copyfile(work / 'candidate' / 'testData/execution_workloads.json', target / 'testData/execution_workloads.json')
shutil.copyfile(assets / 'candidate-modes.go.txt', work / 'candidate/final_review_modes_test.go')
for tree in ['upstream', 'probes']:
    shutil.copyfile(assets / 'legacy-probes.go.txt', work / tree / 'review_legacy_probe_test.go')
for source, target in [('restricted-probes.go.txt', 'review_probe_test.go'), ('compatibility-probes.go.txt', 'review_extra_test.go')]:
    shutil.copyfile(assets / source, work / 'probes' / target)
for script in ['run_stage.py', 'run_bench.py', 'analyze_bench.py']:
    shutil.copyfile(assets / script, work / script)
print(work)
