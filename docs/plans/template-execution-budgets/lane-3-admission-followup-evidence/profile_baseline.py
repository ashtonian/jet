import os,sys,subprocess,json,time,signal,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/admission-followup';peer=Path('<FROZEN54>');binary=peer/'results/candidate.test';receipts=json.loads((out/'baseline-profile-receipts.json').read_text()) if (out/'baseline-profile-receipts.json').exists() else []
manifest={'base':'54b561484e1465edea963ed04579bcd24a7dbf76','binary':str(binary),'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'script_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),'build_receipt':str(peer/'results/stages.json'),'source':str(peer/'candidate'),'toolchain':subprocess.check_output(['go','version']).decode().strip(),'env':{k:os.getenv(k) for k in ['GOFLAGS','GOGC','GOMEMLIMIT','GODEBUG']}}
(out/'baseline-profile-inputs.json').write_text(json.dumps(manifest,indent=2)+'\n')
def stage(name,cmd,deadline=60):
 
 if any(r['name']==name and r['command']==cmd and r['exit_code']==0 for r in receipts):return
 print(name,flush=True);start=time.time()
 with (out/(name+'.log')).open('w') as log:
  p=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GOMAXPROCS='2'),start_new_session=True)
  try:code=p.wait(timeout=deadline)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   code=124
 receipts.append({'name':name,'command':cmd,'deadline':deadline,'cleanup':5,'start':start,'end':time.time(),'exit_code':code});(out/'baseline-profile-receipts.json').write_text(json.dumps(receipts,indent=2)+'\n')
 if code and 'no matches found' not in (out/(name+'.log')).read_text():raise RuntimeError(name)
for name,fixture in [('nil','header_literal'),('scalar','header_interpolation'),('map','named_map')]:
 stem='baseline-'+name
 cmd=['/usr/bin/time','-l',str(binary),'-test.run=^$','-test.bench=^BenchmarkExecutionModes$/^restricted_owned$/^'+fixture+'$/^html$','-test.benchtime=3s','-test.cpu=2','-test.count=1','-test.benchmem','-test.timeout=40s']
 for flag,suffix in [('cpuprofile','cpu'),('memprofile','heap'),('mutexprofile','mutex'),('blockprofile','block')]:cmd.append('-test.'+flag+'='+str(out/(stem+'-'+suffix+'.pprof')))
 stage(stem,cmd)
 for metric,profile in [('cpu','cpu'),('alloc_space','heap'),('alloc_objects','heap'),('inuse_space','heap'),('inuse_objects','heap'),('delay','mutex'),('delay','block')]:
  prefix=stem+'-'+profile+'-'+metric
  stage(prefix,['go','tool','pprof','-sample_index='+metric,'-top','-nodecount=35',str(binary),str(out/(stem+'-'+profile+'.pprof'))])
  if metric=='cpu':
   stage(prefix+'-cum',['go','tool','pprof','-sample_index=cpu','-top','-cum','-nodecount=35',str(binary),str(out/(stem+'-cpu.pprof'))])
  if profile=='cpu' or metric.startswith('alloc'):
   base=['go','tool','pprof','-source_path='+str(peer/'candidate'),'-trim_path=github.com/CloudyKit/jet/v6/','-sample_index='+metric]
   stage(prefix+'-source',base+['-list=admitInputScopesInto|inputAdmission.*(clone|step|reserve|enterContainer)|borrowInputVars',str(binary),str(out/(stem+'-'+profile+'.pprof'))])
   stage(prefix+'-call',base+['-peek=admitInputScopesInto|inputAdmission.*(clone|step|reserve|enterContainer)|borrowInputVars',str(binary),str(out/(stem+'-'+profile+'.pprof'))])
print('BASELINE PROFILES COMPLETE',flush=True)
