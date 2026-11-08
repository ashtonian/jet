import subprocess,os,signal,json,time
from pathlib import Path
out=Path('.git/finalization');env=dict(os.environ,GOMAXPROCS='2');receipts=[]
def stage(name,args,deadline=90):
 print(name,flush=True);start=time.time()
 with (out/(name+'.log')).open('w') as log:
  p=subprocess.Popen(args,stdout=log,stderr=subprocess.STDOUT,env=env,start_new_session=True)
  try:code=p.wait(timeout=deadline)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   code=124
 receipts.append({'name':name,'command':args,'deadline':deadline,'cleanup_seconds':5,'start':start,'end':time.time(),'exit_code':code});(out/'input-shape-stages.json').write_text(json.dumps(receipts,indent=2)+'\n')
 if code:raise RuntimeError(name)
for phase in ['pooled4','active8']:
 stage('shape-layout-'+phase,['/usr/bin/time','-l',str(out/('follow-'+phase+'.test')),'-test.run=^TestAdmissionLayoutAndRetained$','-test.v','-test.timeout=60s'])
for sample in range(3):
 order=['pooled4','active8'];order=order[sample%2:]+order[:sample%2]
 for phase in order:stage('shape-'+phase+'-'+str(sample),[str(out/('follow-'+phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkAdmissionInput$','-test.benchtime=200ms','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=60s'])
for phase in ['pooled4','active8']:
 args=['/usr/bin/time','-l',str(out/('follow-'+phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkAdmissionInput$/^(nested|large)$','-test.benchtime=500ms','-test.count=1','-test.cpu=2','-test.timeout=60s']
 for flag,suffix in [('cpuprofile','cpu'),('memprofile','heap'),('mutexprofile','mutex'),('blockprofile','block')]:args.append('-test.'+flag+'='+str(out/('shape-'+phase+'-'+suffix+'.pprof')))
 stage('shape-profile-'+phase,args)
print('SHAPES COMPLETE')
