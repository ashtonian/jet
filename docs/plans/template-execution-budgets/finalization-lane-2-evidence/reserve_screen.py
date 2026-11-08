import hashlib,json,os,signal,subprocess,sys,time
from pathlib import Path
root=Path.cwd()/'.git/finalization';records=[];p=None

def stop(signum,frame):
 if p is not None and p.poll() is None:
  os.killpg(p.pid,signal.SIGTERM)
  try:p.wait(timeout=5)
  except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop)
for variant in (sys.argv[1:] or ['0','256','1024','4096']):
 binary=root/('reserve-'+variant+'.test');out=root/('reserve-'+variant)
 tasks=[('bench',['-test.run=^$','-test.bench=^BenchmarkOutputReserve$','-test.benchtime=200ms','-test.count=3']),('large-small',['-test.run=^TestOutputOwnedLargeThenSmall$','-test.count=30','-test.memprofile='+str(out/'large-small.pprof'),'-test.memprofilerate=1']),('profile',['-test.run=^$','-test.bench=^BenchmarkOutputReserve$/escape_expansion$/warm$','-test.benchtime=1s','-test.count=1','-test.cpuprofile='+str(out/'cpu.pprof'),'-test.memprofile='+str(out/'heap.pprof'),'-test.memprofilerate=1','-test.mutexprofile='+str(out/'mutex.pprof'),'-test.blockprofile='+str(out/'block.pprof')])]
 for name,flags in tasks:
  command=['/usr/bin/time','-l',str(binary)]+flags
  started=time.time()
  with (out/(name+'.log')).open('w') as log:
   p=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,env=dict(os.environ,GOMAXPROCS='2'))
   try:rc=p.wait(timeout=100)
   except subprocess.TimeoutExpired:
    os.killpg(p.pid,signal.SIGTERM)
    try:p.wait(timeout=5)
    except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
    rc=124
  record=dict(variant=variant,stage=name,command=command,environment={'GOMAXPROCS':'2'},deadline_seconds=100,cleanup_seconds=5,exit_code=rc,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started=started,finished=time.time());records.append(record)
  (out/(name+'.json')).write_text(json.dumps(record,indent=2)+'\n')
  if rc:sys.exit(rc)
(root/('reserve-screen-'+('-'.join(sys.argv[1:]) or 'capacities')+'.json')).write_text(json.dumps(records,indent=2)+'\n')
print('reserve screening complete')
