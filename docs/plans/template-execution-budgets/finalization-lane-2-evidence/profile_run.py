import hashlib,json,os,signal,subprocess,sys,time
from pathlib import Path
root=Path.cwd()/'.git/finalization';receipts=[];p=None

def stop(signum,frame):
 if p is not None and p.poll() is None:
  os.killpg(p.pid,signal.SIGTERM)
  try:p.wait(timeout=5)
  except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop)
for build in ['baseline','fixed']:
 binary=root/(build+'.test')
 for mode in ['legacy_stream','metered_live_stream','restricted_owned']:
  art=root/'profiles'/(build+'-'+mode);art.mkdir(parents=True,exist_ok=True)
  command=['/usr/bin/time','-l',str(binary),'-test.run=^$','-test.bench=^BenchmarkFinalReview$/'+mode+'$/escape_expansion$/html$','-test.benchtime=1s','-test.count=1','-test.cpuprofile='+str(art/'cpu.pprof'),'-test.memprofile='+str(art/'heap.pprof'),'-test.memprofilerate=1','-test.mutexprofile='+str(art/'mutex.pprof'),'-test.blockprofile='+str(art/'block.pprof')]
  started=time.time()
  with (art/'run.log').open('w') as log:
   p=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,env=dict(os.environ,GOMAXPROCS='2'))
   try:rc=p.wait(timeout=60)
   except subprocess.TimeoutExpired:
    os.killpg(p.pid,signal.SIGTERM)
    try:p.wait(timeout=5)
    except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
    rc=124
  receipts.append(dict(command=command,execution_timeout=60,term_kill_after=5,started=started,finished=time.time(),exit_code=rc,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),environment={'GOMAXPROCS':'2'}))
  (root/'profiles'/'results.json').write_text(json.dumps(receipts,indent=2)+'\n')
  if rc:sys.exit(rc)
print('all six profile processes completed')
