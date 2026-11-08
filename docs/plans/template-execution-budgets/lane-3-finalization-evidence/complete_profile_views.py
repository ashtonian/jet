import subprocess,json,time,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/finalization';records=[]
for prefix in ['baseline','graph4','input4','follow-base4','follow-pooled4','follow-active8','shape-pooled4','shape-active8']:
 binary=prefix if not prefix.startswith('shape-') else 'follow-'+prefix[6:]
 profile=out/(prefix+'-heap.pprof')
 if not profile.exists():continue
 for sample in ['alloc_objects','inuse_objects']:
  args=['go','tool','pprof','-top','-sample_index='+sample,'-nodecount=30',str(out/(binary+'.test')),str(profile)];start=time.time()
  with (out/(prefix+'-'+sample+'.log')).open('w') as log:r=subprocess.run(args,stdout=log,stderr=subprocess.STDOUT,timeout=30)
  records.append({'command':args,'exit_code':r.returncode,'start':start,'end':time.time()})
 for kind,flags in [('cpu-flat',['-top']),('cpu-cumulative',['-top','-cum']),('alloc',['-top','-sample_index=alloc_space']),('heap',['-top','-sample_index=inuse_space']),('mutex',['-top']),('block',['-top'])]:
  target=out/(prefix+'-'+kind+'.log')
  if target.exists():continue
  suffix='cpu' if kind.startswith('cpu') else 'heap' if kind in ['alloc','heap'] else kind
  args=['go','tool','pprof',*flags,'-nodecount=30',str(out/(binary+'.test')),str(out/(prefix+'-'+suffix+'.pprof'))];start=time.time()
  with target.open('w') as log:r=subprocess.run(args,stdout=log,stderr=subprocess.STDOUT,timeout=30)
  records.append({'command':args,'exit_code':r.returncode,'start':start,'end':time.time()})
 if prefix.startswith('shape-'):
  args=['go','tool','pprof','-source_path='+str(root),'-trim_path=github.com/CloudyKit/jet/v6/','-list=clone|enterContainer|admitInputScopesInto','-sample_index=alloc_space',str(out/(binary+'.test')),str(profile)]
  with (out/(prefix+'-source.log')).open('w') as log:r=subprocess.run(args,stdout=log,stderr=subprocess.STDOUT,timeout=30)
  records.append({'command':args,'exit_code':r.returncode,'no_sampled_match':'no matches found' in (out/(prefix+'-source.log')).read_text()})
(out/'complete-profile-view-stages.json').write_text(json.dumps(records,indent=2)+'\n')
# The source path points into these immutable overlay snapshots. Verify hashes
# against the manifest captured before the corresponding binaries were built.
proof=[]
for manifest,variants in [('measurement-inputs.json',None),('followup-inputs.json',None)]:
 d=json.loads((out/manifest).read_text())
 for phase,row in d['variants'].items():
  replacements=row.get('replacements',row)
  overlay=out/(phase+'-overlay.json') if manifest=='measurement-inputs.json' else out/'followup-frozen'/(phase+'-overlay.json')
  paths=json.loads(overlay.read_text())['Replace']
  for name,expected in replacements.items():
   if not name.endswith(('compile.go','input.go','execution.go','exec.go')):continue
   path=Path(paths[name]);actual=hashlib.sha256(path.read_bytes()).hexdigest();assert actual==expected,(phase,path)
   proof.append({'variant':phase,'debug_source_file':str(path.relative_to(root)),'sha256':actual,'matches_build_manifest':True})
(out/'profile-source-resolution.json').write_text(json.dumps(proof,indent=2)+'\n')
print('complete object/space/profile views and frozen-source proof')
