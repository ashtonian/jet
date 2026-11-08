import hashlib,json,os,signal,subprocess,time
from pathlib import Path
art=Path.cwd()/'.git/finalization/followup-54';records=[];active=None
builds=json.loads((art/'output-builds.json').read_text());builds['variants']['combined']=json.loads((art/'builds.json').read_text())['variants']['combined']
def stop(sig,frame):
 if active is not None and active.poll() is None:
  os.killpg(active.pid,signal.SIGTERM)
  try:active.wait(timeout=5)
  except subprocess.TimeoutExpired:os.killpg(active.pid,signal.SIGKILL);active.wait(timeout=5)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop)
for sample in range(3):
 order=['combined','work-fast','legacy-view'];order=order if sample%2==0 else order[::-1]
 for name in order:
  binary=art/'variants'/name/'render.test';assert hashlib.sha256(binary.read_bytes()).hexdigest()==builds['variants'][name]['binary_sha256'];cmd=['/usr/bin/time','-l',str(binary),'-test.run=^$','-test.bench=^BenchmarkExecutionModes$/(legacy_stream|metered_background_stream|metered_live_stream)$/(header_literal|header_interpolation|nested_writing|escape_expansion)$/(text|html)$','-test.benchtime=200ms','-test.count=1'];start=time.time()
  with (art/('work-prototype-'+name+'-'+str(sample)+'.log')).open('w') as log:
   active=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,env=dict(os.environ,GOMAXPROCS='2'))
   try:rc=active.wait(timeout=30)
   except subprocess.TimeoutExpired:
    os.killpg(active.pid,signal.SIGTERM)
    try:active.wait(timeout=5)
    except subprocess.TimeoutExpired:os.killpg(active.pid,signal.SIGKILL);active.wait(timeout=5)
    rc=124
  records.append(dict(variant=name,sample=sample,command=cmd,exit_code=rc,deadline=30,GOMAXPROCS=2,kill_after=5,elapsed=time.time()-start));(art/'work-prototype.json').write_text(json.dumps(records,indent=2)+'\n');print(sample,name,rc,flush=True);assert rc==0
