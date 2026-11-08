import hashlib,json,os,signal,subprocess,time
from pathlib import Path
root=Path.cwd()/'.git/finalization';out=root/'selected-measure';out.mkdir(exist_ok=True);p=None;receipts=[]
def stop(signum,frame):
 if p is not None and p.poll() is None:
  os.killpg(p.pid,signal.SIGTERM)
  try:p.wait(timeout=5)
  except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop)
def stage(name,binary,flags,deadline=90):
 global p
 command=['/usr/bin/time','-l',str(binary)]+flags
 started=time.time()
 with (out/(name+'.log')).open('w') as log:
  p=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,env=dict(os.environ,GOMAXPROCS='2'))
  try:rc=p.wait(timeout=deadline)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   rc=124
 receipts.append(dict(name=name,command=command,deadline=deadline,cleanup=5,started=started,finished=time.time(),exit_code=rc,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),environment={'GOMAXPROCS':'2'}));(out/'results.json').write_text(json.dumps(receipts,indent=2)+'\n')
 if rc:raise SystemExit(rc)
for sample in range(6):
 for build in (['baseline','selected'] if sample%2==0 else ['selected','baseline']):
  stage(build+'-'+str(sample),root/(build+'.test'),['-test.run=^$','-test.bench=^BenchmarkFinalReview$/(legacy_stream|metered_background_stream|metered_live_stream|restricted_owned)$/(header_interpolation|escape_expansion)$/(html|text)$','-test.benchtime=1s','-test.count=1'])
for build in ['baseline','selected']:
 for mode in ['legacy_stream','metered_live_stream','restricted_owned']:
  prefix=out/(build+'-'+mode)
  flags=['-test.run=^$','-test.bench=^BenchmarkFinalReview$/'+mode+'$/escape_expansion$/html$','-test.benchtime=2s','-test.count=1','-test.cpuprofile='+str(prefix)+'-cpu.pprof','-test.memprofile='+str(prefix)+'-heap.pprof','-test.mutexprofile='+str(prefix)+'-mutex.pprof','-test.blockprofile='+str(prefix)+'-block.pprof']
  stage(build+'-'+mode+'-profile',root/(build+'.test'),flags)
print('six interleaved output samples and matched CPU/default-sampling memory/contention profiles complete')
