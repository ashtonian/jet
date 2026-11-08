import os,pathlib,subprocess,time,json,hashlib,signal
root=pathlib.Path(__file__).parent
out=root/'bench';out.mkdir(exist_ok=True)
env=dict(os.environ,GOMAXPROCS='2')
receipts=[]
def stage(name,tree,args,timeout=120):
 print(name,flush=True);start=time.time();log=out/(name+'.txt')
 with log.open('w') as f:
  p=subprocess.Popen(args,cwd=root/tree,env=env,stdout=f,stderr=subprocess.STDOUT,start_new_session=True)
  try:code=p.wait(timeout=timeout)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:code=p.wait(timeout=10)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);code=p.wait()
   code=124
 receipts.append(dict(name=name,tree=tree,command=args,exit=code,start=start,end=time.time(),GOMAXPROCS='2'))
 (out/'stages.json').write_text(json.dumps(receipts,indent=2))
 if code:raise RuntimeError(name+' failed; '+str(log))
for tree in ['upstream','correctness','candidate']:
 manifest={str(p.relative_to(root/tree)):hashlib.sha256(p.read_bytes()).hexdigest() for p in (root/tree).rglob('*') if p.is_file() and (p.suffix in ['.go','.json'] or p.name in ['go.mod','go.sum'])}
 (out/(tree+'-inputs.json')).write_text(json.dumps(manifest,indent=2))
 stage('build-'+tree,tree,['go','test','-p=1','-c','-o',str(out/(tree+'.test')),'.'])
 stage('conformance-'+tree,tree,[str(out/(tree+'.test')),'-test.run=^TestFinalReviewConformance$','-test.timeout=60s'])
for sample in range(6):
 trees=['upstream','correctness','candidate'];trees=trees[sample%3:]+trees[:sample%3]
 for tree in trees:
  stage('render-'+tree+'-'+str(sample),tree,[str(out/(tree+'.test')),'-test.run=^$','-test.bench=^BenchmarkFinalReview$','-test.benchtime=100ms','-test.count=1','-test.cpu=2','-test.benchmem','-test.timeout=120s'])
for tree in ['upstream','correctness','candidate']:
 stage('compile-'+tree,tree,[str(out/(tree+'.test')),'-test.run=^$','-test.bench=^BenchmarkFinalCompile$','-test.benchtime=100ms','-test.count=6','-test.cpu=2','-test.benchmem','-test.timeout=120s'])
for tree,mode in [('upstream','legacy_stream'),('correctness','legacy_stream'),('candidate','legacy_stream'),('candidate','legacy_owned'),('candidate','metered_background_stream'),('candidate','metered_live_stream'),('candidate','restricted_owned')]:
 name='profile-'+tree+'-'+mode
 args=['/usr/bin/time','-l',str(out/(tree+'.test')),'-test.run=^$','-test.bench=^BenchmarkFinalReview$/^'+mode+'$/^(header_interpolation|nested_silent|escape_expansion|include|block_content)$/^html$','-test.benchtime=500ms','-test.count=1','-test.cpu=2','-test.timeout=60s']
 for flag,suffix in [('cpuprofile','cpu'),('memprofile','heap'),('mutexprofile','mutex'),('blockprofile','block')]:args.append('-test.'+flag+'='+str(out/(name+'-'+suffix+'.pprof')))
 stage(name,tree,args,90)
print('BENCHMARK BATCH COMPLETE',flush=True)
