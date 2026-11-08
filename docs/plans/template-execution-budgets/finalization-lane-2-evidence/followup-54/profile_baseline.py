import hashlib,json,os,signal,subprocess,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';peer=Path('<integration-artifacts>/release-54b5614');binary=peer/'results/candidate.test';stages=[];active=None
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def stop(sig,frame):
 if active is not None and active.poll() is None:
  os.killpg(active.pid,signal.SIGTERM)
  try:active.wait(timeout=5)
  except subprocess.TimeoutExpired:os.killpg(active.pid,signal.SIGKILL);active.wait(timeout=5)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop)
inputs={p:sha(root/p) for p in subprocess.check_output(['git','ls-files'],text=True).splitlines() if p.endswith('.go') or p in ('go.mod','go.sum','testData/execution_workloads.json')}
peerinputs=json.loads((peer/'sources.json').read_text())['candidate']['inputs']
assert all(inputs.get(p)==h for p,h in peerinputs.items() if p.endswith('.go') or p in ('go.mod','go.sum','testData/execution_workloads.json'))
for name in ['eval.go','exec.go','execution.go','execution_output.go','execution_print.go','restricted.go']:(art/'baseline-source'/name).write_bytes((root/name).read_bytes())
record={'revision':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),'binary':str(binary),'binary_sha256':sha(binary),'inputs':inputs,'peer_manifest_sha256':sha(peer/'sources.json'),'toolchain':subprocess.check_output(['go','version'],text=True).strip(),'environment':dict(GOMAXPROCS=2),'steps':stages}
(art/'baseline.json').write_text(json.dumps(record,indent=2)+'\n')
for mode in ['legacy_stream','metered_background_stream','metered_live_stream']:
 for fixture,fmt in [('header_literal','text'),('inheritance','text'),('nested_writing','text'),('header_interpolation','text'),('escape_expansion','html')]:
  name=mode+'-'+fixture;prefix=art/name;cmd=['/usr/bin/time','-l',str(binary),'-test.run=^$','-test.bench=^BenchmarkExecutionModes$/'+mode+'$/'+fixture+'$/'+fmt+'$','-test.benchtime=1s','-test.count=1']
  for kind,flag in [('cpu','cpuprofile'),('heap','memprofile'),('mutex','mutexprofile'),('block','blockprofile')]:cmd.append('-test.'+flag+'='+str(prefix)+'-'+kind+'.pprof')
  start=time.time()
  with (art/(name+'.log')).open('w') as log:
   active=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,env=dict(os.environ,GOMAXPROCS='2'))
   try:rc=active.wait(timeout=20)
   except subprocess.TimeoutExpired:
    os.killpg(active.pid,signal.SIGTERM)
    try:active.wait(timeout=5)
    except subprocess.TimeoutExpired:os.killpg(active.pid,signal.SIGKILL);active.wait(timeout=5)
    rc=124
  stages.append(dict(name=name,command=cmd,exit_code=rc,deadline=20,kill_after=5,elapsed=time.time()-start));(art/'baseline.json').write_text(json.dumps(record,indent=2)+'\n');print(name,rc,flush=True);assert rc==0
record['inputs_unchanged']=all(sha(root/p)==h for p,h in inputs.items());assert record['inputs_unchanged'];(art/'baseline.json').write_text(json.dumps(record,indent=2)+'\n')
