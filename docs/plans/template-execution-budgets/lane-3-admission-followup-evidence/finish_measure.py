import os,sys,subprocess,json,time,signal,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/admission-followup';manifest=json.loads((out/'inputs-with-defaults.json').read_text());receipts=[]
(out/'finish-script-sha256.txt').write_text(hashlib.sha256(Path(__file__).read_bytes()).hexdigest()+'\n')
def stage(name,cmd,deadline=90):
 print(name,flush=True);start=time.time()
 with (out/(name+'.log')).open('w') as log:
  p=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GOMAXPROCS='2'),start_new_session=True)
  try:code=p.wait(timeout=deadline)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   code=124
 receipts.append({'name':name,'command':cmd,'deadline':deadline,'cleanup':5,'start':start,'end':time.time(),'exit_code':code});(out/'finish-receipts.json').write_text(json.dumps(receipts,indent=2)+'\n')
 if code:raise RuntimeError(name)
for name in manifest['variants']:
 for k,v in json.loads(Path(manifest['variants'][name]['overlay']).read_text())['Replace'].items():assert hashlib.sha256(Path(v).read_bytes()).hexdigest()==manifest['variants'][name]['hashes'][k]
stage('build-defaults',['go','test','-p=1','-c','-overlay='+manifest['variants']['defaults']['overlay'],'-o',str(out/'defaults.test'),'.'])
stage('conformance-defaults',[str(out/'defaults.test'),'-test.run=^(TestInputAdmission|TestRestricted|TestCompilation|TestExecutionCompilation|TestTemplateBundle|TestExecutionModeConformance)','-test.timeout=60s'])
core='^BenchmarkExecutionModes$/^restricted_owned$/^(header_literal|header_interpolation|include|inheritance|named_map)$/^html$'
shape='^BenchmarkAdmissionInput$'
for sample in range(3):
 for name in (['direct','defaults'] if sample%2==0 else ['defaults','direct']):
  stage('default-screen-'+name+'-'+str(sample),[str(out/(name+'.test')),'-test.run=^$','-test.bench='+core,'-test.benchtime=200ms','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=60s'])
for sample in range(6):
 for name in (['base','defaults'] if sample%2==0 else ['defaults','base']):
  for kind,selector in [('core',core),('shape',shape)]:
   stage('paired-'+kind+'-'+name+'-'+str(sample),[str(out/(name+'.test')),'-test.run=^$','-test.bench='+selector,'-test.benchtime=1s','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=70s'])
for name in ['base','defaults']:
 for kind,selector in [('nil','^BenchmarkExecutionModes$/^restricted_owned$/^header_literal$/^html$'),('scalar','^BenchmarkExecutionModes$/^restricted_owned$/^header_interpolation$/^html$'),('map','^BenchmarkAdmissionInput$/^small_map$'),('large','^BenchmarkAdmissionInput$/^(nested|shared|large)$'),('concurrent','^BenchmarkConcurrentExecutionModes$/^restricted_owned$/^(header_interpolation|data_unused_256)$/^html$/^workers_12$/')]:
  stem='selected-'+name+'-'+kind;cmd=['/usr/bin/time','-l',str(out/(name+'.test')),'-test.run=^$','-test.bench='+selector,'-test.benchtime='+('3s' if kind in ['nil','scalar','map'] else '1s'),'-test.cpu='+('12' if kind=='concurrent' else '2'),'-test.count=1','-test.benchmem','-test.timeout=60s']
  for flag,suffix in [('cpuprofile','cpu'),('memprofile','heap'),('mutexprofile','mutex'),('blockprofile','block')]:cmd.append('-test.'+flag+'='+str(out/(stem+'-'+suffix+'.pprof')))
  stage(stem,cmd)
print('PAIRED AND PROFILES COMPLETE',flush=True)
