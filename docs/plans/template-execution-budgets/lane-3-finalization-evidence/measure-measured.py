import os,sys,subprocess,signal,time,json,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/finalization';env=dict(os.environ,GOMAXPROCS='2');receipts=[]
def stage(name,args,timeout=120,expect=0):
 print(name,flush=True);start=time.time();log=out/(name+'.log')
 with log.open('w') as stream:
  p=subprocess.Popen(args,cwd=root,env=env,stdout=stream,stderr=subprocess.STDOUT,start_new_session=True)
  try:code=p.wait(timeout=timeout)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   code=124
 receipt={'name':name,'command':args,'start':start,'end':time.time(),'execution_deadline_seconds':timeout,'cleanup_seconds':5,'exit_code':code,'expected_exit':expect,'GOMAXPROCS':'2','log':str(log.relative_to(root))}
 receipts.append(receipt);(out/'measurement-stages.json').write_text(json.dumps(receipts,indent=2)+'\n')
 if code!=expect:raise RuntimeError(name+' failed; '+str(log))
 return log
phases=['baseline','graph4','input4','lazy0','inline8','inline16','walk','nopool']
inputs={'head':subprocess.check_output(['git','rev-parse','HEAD']).decode().strip(),'go':subprocess.check_output(['go','version']).decode().strip(),'go_env':subprocess.check_output(['go','env','GOOS','GOARCH','GOFLAGS']).decode().splitlines(),'GOMAXPROCS':'2','harness_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),'variants':{}}
for phase in phases:
 overlay=out/(phase+'-overlay.json');replacements=json.loads(overlay.read_text())['Replace'];inputs['variants'][phase]={'overlay_sha256':hashlib.sha256(overlay.read_bytes()).hexdigest(),'replacements':{name:hashlib.sha256(Path(target).read_bytes()).hexdigest() for name,target in replacements.items()}}
inputs['common']={str(p.relative_to(root)):hashlib.sha256(p.read_bytes()).hexdigest() for p in root.rglob('*.go') if '.git' not in p.parts and p.parent!=root}
inputs['dependencies']={p:hashlib.sha256((root/p).read_bytes()).hexdigest() for p in ['go.mod','go.sum','testData/execution_workloads.json']}
(out/'measurement-inputs.json').write_text(json.dumps(inputs,indent=2)+'\n')
for phase in phases:
 stage('build-'+phase,['go','test','-p=1','-c','-overlay='+str(out/(phase+'-overlay.json')),'-o',str(out/(phase+'.test')),'.'])
 stage('conformance-'+phase,[str(out/(phase+'.test')),'-test.run=^TestFinalReviewConformance$','-test.timeout=90s'])
 stage('layout-'+phase,['/usr/bin/time','-l',str(out/(phase+'.test')),'-test.run=^TestAdmissionLayoutAndRetained$','-test.v','-test.timeout=90s'])
stage('dag-control-baseline',[str(out/'baseline.test'),'-test.run=^TestCompilationSharedDiamondWork$','-test.timeout=90s'],expect=1)
stage('semantic-controls-fixed',[str(out/'input4.test'),'-test.run=^(TestCompilation|TestInputAdmission|TestExecutionCompilation|TestTemplateBundle|TestRestricted)','-test.timeout=90s'])
# Screen layout capacities on K-1/K/K+1 and larger graphs; these short samples
# select the structure, not the full-release performance verdict.
screens=['baseline','lazy0','input4','inline8','inline16','nopool']
for sample in range(3):
 order=screens[sample:]+screens[:sample]
 for phase in order:
  stage('screen-'+phase+'-'+str(sample),[str(out/(phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkAdmission(Lookup|Input|Cold)$','-test.benchtime=150ms','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=90s'])
for phase in ['graph4','walk']:
 stage('diamond-'+phase,[str(out/(phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkAdmissionDiamond$','-test.benchtime=1s','-test.count=6','-test.cpu=2','-test.benchmem','-test.timeout=90s'])
for sample in range(6):
 phases3=['baseline','graph4','input4'];order=phases3[sample%3:]+phases3[:sample%3]
 for phase in order:
  stage('matched-'+phase+'-'+str(sample),[str(out/(phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkFinalReview$/^(legacy_stream|legacy_owned|metered_background_stream|metered_live_stream|restricted_owned)$/^(header_interpolation|include)$/^(text|html)$','-test.benchtime=1s','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=90s'])
for phase in ['baseline','graph4','input4']:
 args=['/usr/bin/time','-l',str(out/(phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkFinalReview$/^(metered_live_stream|restricted_owned)$/^(header_interpolation|include)$/^html$','-test.benchtime=500ms','-test.count=1','-test.cpu=2','-test.timeout=90s']
 for flag,suffix in [('cpuprofile','cpu'),('memprofile','heap'),('mutexprofile','mutex'),('blockprofile','block')]:args.append('-test.'+flag+'='+str(out/(phase+'-'+suffix+'.pprof')))
 stage('profile-'+phase,args)
print('MEASUREMENT COMPLETE',flush=True)
