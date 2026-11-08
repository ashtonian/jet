import subprocess,json,time
from pathlib import Path
out=Path('.git/finalization');records=[]
for phase in ['baseline','graph4','input4']:
 for name in ['admitInputScopes','admitCached','startCompilation','prepareSource']:
  label=phase+'-resolved-'+name;args=['go','tool','pprof','-source_path='+str(Path.cwd()),'-trim_path=github.com/CloudyKit/jet/v6/','-list='+name,'-sample_index=alloc_space',str(out/(phase+'.test')),str(out/(phase+'-heap.pprof'))];start=time.time()
  with (out/(label+'.log')).open('w') as log:r=subprocess.run(args,stdout=log,stderr=subprocess.STDOUT,timeout=30)
  text=(out/(label+'.log')).read_text();records.append({'command':args,'start':start,'end':time.time(),'exit':r.returncode,'no_sampled_match':'no matches found' in text,'resolved':'could not find file' not in text})
  if 'could not find file' in text:raise RuntimeError(label+' unresolved source')
(out/'resolved-source-stages.json').write_text(json.dumps(records,indent=2)+'\n')
print('source report resolution complete')
