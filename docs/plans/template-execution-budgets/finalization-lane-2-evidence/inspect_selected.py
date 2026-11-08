import json,subprocess
from pathlib import Path
root=Path.cwd()/'.git/finalization';art=root/'selected-measure';receipts=[]
for build in ['baseline','selected']:
 for mode in ['legacy_stream','metered_live_stream','restricted_owned']:
  prefix=art/(build+'-'+mode)
  for view,profile,flags in [('cpu-flat','cpu',['-top']),('cpu-cumulative','cpu',['-top','-cum']),('alloc','heap',['-top','-alloc_space']),('heap','heap',['-top','-inuse_space']),('mutex','mutex',['-top']),('block','block',['-top']),('source','cpu',['-list','checkedWriter|executionWriter|checkpoint|dataPrinter.*Write'])]:
   cmd=['go','tool','pprof','-nodecount=22','-source_path='+str(root/'profile-source'/build),'-trim_path='+str(Path.cwd())+':github.com/CloudyKit/jet/v6']+flags+[str(root/(build+'.test')),str(prefix)+'-'+profile+'.pprof']
   result=subprocess.run(cmd,capture_output=True,text=True,timeout=25)
   Path(str(prefix)+'-'+view+'.txt').write_text(result.stdout+result.stderr)
   receipts.append(dict(command=cmd,exit_code=result.returncode,execution_deadline_seconds=25))
   if result.returncode:raise SystemExit(result.returncode)
(art/'profile-inspection.json').write_text(json.dumps(receipts,indent=2)+'\n')
print('Matched default-sampling CPU/allocation/live/contention/source views complete')
