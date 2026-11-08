import os,subprocess,json,time,signal,re,statistics,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/admission-followup';receipts=json.loads((out/'analysis-receipts.json').read_text()) if (out/'analysis-receipts.json').exists() else []
def stage(name,cmd,deadline=40):

 if any(r["name"]==name and r["command"]==cmd and r["status"] in ["passed","no_sampled_attribution"] for r in receipts):return
 start=time.time()
 with (out/(name+'.log')).open('w') as log:
  p=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
  try:code=p.wait(timeout=deadline)
  except subprocess.TimeoutExpired:
   os.killpg(p.pid,signal.SIGTERM)
   try:p.wait(timeout=5)
   except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL);p.wait(timeout=5)
   code=124
 status='passed' if code==0 else 'no_sampled_attribution' if code in [1,2] and 'no matches found' in (out/(name+'.log')).read_text() else 'failed'
 receipts.append({'name':name,'command':cmd,'deadline':deadline,'cleanup':5,'start':start,'end':time.time(),'exit_code':code,'status':status});(out/'analysis-receipts.json').write_text(json.dumps(receipts,indent=2)+'\n')
 if status=='failed':raise RuntimeError(name)
for phase in ['base','defaults']:
 binary=out/(phase+'.test')
 for shape in ['nil','scalar','map','large','concurrent']:
  stem='selected-'+phase+'-'+shape;print(stem,flush=True)
  for metric,profile in [('cpu','cpu'),('alloc_space','heap'),('alloc_objects','heap'),('inuse_space','heap'),('inuse_objects','heap'),('delay','mutex'),('delay','block')]:
   prefix=stem+'-'+profile+'-'+metric;pprof=out/(stem+'-'+profile+'.pprof');args=['go','tool','pprof','-sample_index='+metric]
   stage(prefix,args+['-top','-nodecount=40',str(binary),str(pprof)])
   if metric=='cpu':stage(prefix+'-cum',args+['-top','-cum','-nodecount=40',str(binary),str(pprof)])
   if metric in ['cpu','alloc_space','alloc_objects']:
    source=['-source_path='+str(root),'-trim_path=github.com/CloudyKit/jet/v6/']
    pattern='admitInputScopesInto|inputAdmission.*(clone|step|reserve|enterContainer)|inputHasMethods|borrowInputVars|Template.*execute'
    stage(prefix+'-source',args+source+['-list='+pattern,str(binary),str(pprof)])
    stage(prefix+'-call',args+['-peek='+pattern,str(binary),str(pprof)])
   if profile in ['mutex','block']:stage(prefix+'-call',args+['-peek=sync.*|inputAdmission|RenderRestrictedContext',str(binary),str(pprof)])
raw={}
for category in ['core','shape']:
 raw[category]={}
 for p in out.glob('paired-'+category+'-*.log'):
  phase=p.name.split('-')[-2]
  for line in p.read_text().splitlines():
   m=re.match(r'(Benchmark\S+)-2\s+\d+\s+([\d.]+) ns/op\s+([\d.]+) B/op\s+([\d.]+) allocs/op',line)
   if m:raw[category].setdefault(m[1],{}).setdefault(phase,[]).append(tuple(map(float,m.groups()[1:])))
assert len(raw["core"])==5, list(raw["core"])
assert len(raw["shape"])==6, list(raw["shape"])
summary={}
for category,fixtures in raw.items():
 summary[category]={}
 for name,phases in fixtures.items():
  summary[category][name]={k:{'samples':len(v),'ns':statistics.median(x[0] for x in v),'B':statistics.median(x[1] for x in v),'allocs':statistics.median(x[2] for x in v)} for k,v in phases.items()}
  a=summary[category][name];a['ratio']=a['defaults']['ns']/a['base']['ns'];assert a['base']['samples']==a['defaults']['samples']==6
(out/'paired-analysis.json').write_text(json.dumps(summary,indent=2)+'\n')
proof={}
inputs=json.loads((out/'inputs-with-defaults.json').read_text())
for phase in ['base','defaults']:
 mapping=json.loads(Path(inputs['variants'][phase]['overlay']).read_text())['Replace']
 for name in ['input.go','exec.go']:
  frozen=Path(mapping[str(root/name)]);proof[phase+'/'+name]={'frozen':str(frozen),'sha256':hashlib.sha256(frozen.read_bytes()).hexdigest(),'expected':inputs['variants'][phase]['hashes'][str(root/name)]}
  assert proof[phase+'/'+name]['sha256']==proof[phase+'/'+name]['expected']
(out/'source-proof.json').write_text(json.dumps(proof,indent=2)+'\n')
print('ANALYSIS COMPLETE',flush=True)
