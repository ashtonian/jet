import os,sys,subprocess,json,time,signal,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/admission-followup';manifest=json.loads((out/'inputs.json').read_text());receipts=[]
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
 receipts.append({'name':name,'command':cmd,'deadline':deadline,'cleanup':5,'start':start,'end':time.time(),'exit_code':code});(out/'measurement-receipts.json').write_text(json.dumps(receipts,indent=2)+'\n')
 if code:raise RuntimeError(name)
for name in manifest['variants']:
 for k,v in json.loads(Path(manifest['variants'][name]['overlay']).read_text())['Replace'].items():
  assert hashlib.sha256(Path(v).read_bytes()).hexdigest()==manifest['variants'][name]['hashes'][k]
 stage('build-'+name,['go','test','-p=1','-c','-overlay='+manifest['variants'][name]['overlay'],'-o',str(out/(name+'.test')),'.'])
 stage('conformance-'+name,[str(out/(name+'.test')),'-test.run=^(TestInputAdmission|TestRestricted|TestCompilation|TestExecutionCompilation|TestTemplateBundle|TestExecutionModeConformance)','-test.timeout=60s'])
for sample in range(3):
 order=['base','split','direct'];order=order[sample%3:]+order[:sample%3]
 for name in order:
  stage('screen-core-'+name+'-'+str(sample),[str(out/(name+'.test')),'-test.run=^$','-test.bench=^BenchmarkExecutionModes$/^restricted_owned$/^(header_literal|header_interpolation|include|inheritance|named_map)$/^html$','-test.benchtime=200ms','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=60s'])
  stage('screen-shape-'+name+'-'+str(sample),[str(out/(name+'.test')),'-test.run=^$','-test.bench=^BenchmarkAdmissionInput$','-test.benchtime=200ms','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=60s'])
print('SCREEN COMPLETE',flush=True)
