import os,sys,subprocess,signal,time,json,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/finalization';env=dict(os.environ,GOMAXPROCS='2');folder=out/'followup-frozen';folder.mkdir(exist_ok=True)
base=json.loads((out/'followup-base.json').read_text())['Replace'];phases={'base4':base}
for phase,cap in [('cap8',8),('cap16',16)]:
 replace=dict(base);source=Path(base[str(root/'compile.go')]).read_text().replace('[4]parseSource',f'[{cap}]parseSource').replace('[4]*Template',f'[{cap}]*Template');target=folder/(phase+'-compile.go.txt');target.write_text(source);replace[str(root/'compile.go')]=str(target);phases[phase]=replace
current=dict(base)
for p in root.glob('*.go'):
 target=folder/p.name;target.write_bytes(p.read_bytes());current[str(p)]=str(target)
phases['pooled4']=current
active8=dict(current);target=folder/'active8-input.go.txt';target.write_bytes((out/'active8-input.go.txt').read_bytes());active8[str(root/'input.go')]=str(target);phases['active8']=active8
inputs={'go':subprocess.check_output(['go','version']).decode().strip(),'GOMAXPROCS':'2','variants':{}}
for phase,replace in phases.items():
 overlay=folder/(phase+'-overlay.json');overlay.write_text(json.dumps({'Replace':replace},indent=2)+'\n');inputs['variants'][phase]={name:hashlib.sha256(Path(target).read_bytes()).hexdigest() for name,target in replace.items()}
(out/'followup-inputs.json').write_text(json.dumps(inputs,indent=2)+'\n')
receipts=[]
def stage(name,args,deadline=120):
 print(name,flush=True);start=time.time()
 with (out/(name+'.log')).open('w') as log:
  p=subprocess.Popen(args,stdout=log,stderr=subprocess.STDOUT,env=env,start_new_session=True)
  try:code=p.wait(timeout=deadline)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   code=124
 receipts.append({'name':name,'command':args,'start':start,'end':time.time(),'deadline':deadline,'cleanup_seconds':5,'exit_code':code})
 (out/'followup-stages.json').write_text(json.dumps(receipts,indent=2)+'\n')
 if code:raise RuntimeError(name)
for phase in phases:
 stage('follow-build-'+phase,['go','test','-p=1','-c','-overlay='+str(folder/(phase+'-overlay.json')),'-o',str(out/('follow-'+phase+'.test')),'.'])
 stage('follow-conformance-'+phase,[str(out/('follow-'+phase+'.test')),'-test.run=^TestFinalReviewConformance$','-test.timeout=90s'])
# Current production and new controls are validated before timing.
stage('follow-active8-semantic',[str(out/'follow-active8.test'),'-test.run=^(TestCompilation|TestExecutionCompilation|TestRestricted|TestInputAdmission|TestTemplateBundle)','-test.timeout=90s'])
stage('follow-semantic',[str(out/'follow-pooled4.test'),'-test.run=^(TestCompilation|TestExecutionCompilation|TestRestricted|TestInputAdmission|TestTemplateBundle)','-test.timeout=90s'])
for sample in range(3):
 order=['base4','cap8','cap16'];order=order[sample%3:]+order[:sample%3]
 for phase in order:stage('follow-capacity-'+phase+'-'+str(sample),[str(out/('follow-'+phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkFinalReview$/^(legacy_stream|metered_live_stream|restricted_owned)$/^(header_interpolation|include)$/^(text|html)$','-test.benchtime=200ms','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=90s'])
for sample in range(6):
 order=['base4','pooled4','active8'];order=order[sample%3:]+order[:sample%3]
 for phase in order:stage('follow-pair-'+phase+'-'+str(sample),[str(out/('follow-'+phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkFinalReview$/^restricted_owned$/^(header_interpolation|include)$/^(text|html)$','-test.benchtime=1s','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=90s'])
for phase in ['base4','pooled4','active8']:
 args=['/usr/bin/time','-l',str(out/('follow-'+phase+'.test')),'-test.run=^$','-test.bench=^BenchmarkFinalReview$/^restricted_owned$/^(header_interpolation|include)$/^html$','-test.benchtime=500ms','-test.count=1','-test.cpu=2','-test.timeout=90s']
 for flag,suffix in [('cpuprofile','cpu'),('memprofile','heap'),('mutexprofile','mutex'),('blockprofile','block')]:args.append('-test.'+flag+'='+str(out/('follow-'+phase+'-'+suffix+'.pprof')))
 stage('follow-profile-'+phase,args)
print('FOLLOW-UP COMPLETE',flush=True)
