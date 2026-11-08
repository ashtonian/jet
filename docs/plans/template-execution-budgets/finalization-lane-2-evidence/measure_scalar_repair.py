import hashlib,json,os,signal,subprocess,time
from pathlib import Path
root=Path.cwd()/'.git/finalization/scalar-repair';p=None;records=[]
def stop(signum,frame):
 if p is not None and p.poll() is None:
  os.killpg(p.pid,signal.SIGTERM)
  try:p.wait(timeout=5)
  except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop)
def run(name,build,flags,timeout=60):
 global p
 binary=root/(build+'.test');cmd=['/usr/bin/time','-l',str(binary)]+flags;start=time.time()
 with (root/(name+'.log')).open('w') as log:
  p=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,env=dict(os.environ,GOMAXPROCS='2'))
  try:rc=p.wait(timeout=timeout)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   rc=124
 records.append(dict(name=name,build=build,command=cmd,GOMAXPROCS=2,deadline=timeout,kill_after=5,exit_code=rc,started=start,finished=time.time(),binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest()))
 (root/'measurements.json').write_text(json.dumps(records,indent=2)+'\n')
 assert rc==0,(name,rc)
for sample in range(6):
 for build in (['unsafe','lease'] if sample%2==0 else ['lease','unsafe']):
  run('sequential-'+build+'-'+str(sample),build,['-test.run=^$','-test.bench=^BenchmarkScalarSequential$','-test.benchtime=1s','-test.count=1'])
 for build in (['local','lease'] if sample%2==0 else ['lease','local']):
  run('reentry-'+build+'-'+str(sample),build,['-test.run=^$','-test.bench=^BenchmarkScalarReentry$','-test.benchtime=1s','-test.count=1'])
for name,build,selection in [('sequential-unsafe','unsafe','^BenchmarkScalarSequential$/int$/live$'),('sequential-lease','lease','^BenchmarkScalarSequential$/int$/live$'),('reentry-local','local','^BenchmarkScalarReentry$/live$'),('reentry-lease','lease','^BenchmarkScalarReentry$/live$')]:
 prefix=root/name
 flags=['-test.run=^$','-test.bench='+selection,'-test.benchtime=2s','-test.count=1']
 for flag,suffix in [('cpuprofile','cpu'),('memprofile','heap'),('mutexprofile','mutex'),('blockprofile','block')]:flags.append('-test.'+flag+'='+str(prefix)+'-'+suffix+'.pprof')
 run(name+'-profile',build,flags)
print('Six interleaved valid-output scalar comparisons and affected profiles completed; unsafe reentry was never benchmarked')
