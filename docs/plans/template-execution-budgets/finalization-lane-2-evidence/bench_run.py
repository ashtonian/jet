import hashlib,json,os,signal,subprocess,sys,time
from pathlib import Path
art=Path.cwd()/'.git/finalization';name=sys.argv[1];binary=art/(name+'.test');out=art/(name+'-bench');out.mkdir(exist_ok=True)
receipt=dict(binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),binary=str(binary),harness={p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in ['zz_finalization_benchmark_test.go','zz_finalization_modes_test.go','testData/execution_workloads.json']},commands=[],environment={'GOMAXPROCS':'2'})
p=None
def stop(signum,frame):
 if p is not None and p.poll() is None:
  os.killpg(p.pid,signal.SIGTERM)
  try:p.wait(timeout=5)
  except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop)
for sample in range(6):
 command=[str(binary),'-test.run=^$','-test.bench=^BenchmarkFinalReview$/(legacy_stream|metered_live_stream|restricted_owned)$/(header_interpolation|escape_expansion)$/(html|text)$','-test.benchtime=1s','-test.count=1']
 with (out/(str(sample)+'.log')).open('w') as log:
  p=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,env=dict(os.environ,GOMAXPROCS='2'))
  try: rc=p.wait(timeout=100)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   rc=124
 receipt['commands'].append(dict(command=command,deadline=100,kill_after=5,exit_code=rc))
 (out/'result.json').write_text(json.dumps(receipt,indent=2)+'\n')
 if rc:sys.exit(rc)
print('six diagnostic samples complete',out)
