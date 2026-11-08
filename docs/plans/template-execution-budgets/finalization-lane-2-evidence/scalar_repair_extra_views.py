import json,subprocess
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/scalar-repair';extra=[];prior={tuple(r['command']):r['exit_code'] for r in json.loads((art/'extra-inspection.json').read_text())} if (art/'extra-inspection.json').exists() else {}
for label,build in [('sequential-unsafe','unsafe'),('sequential-lease','lease'),('reentry-local','local'),('reentry-lease','lease')]:
 for view,flags in [('inuse-objects-cumulative',['-top','-cum','-inuse_objects']),('alloc-source',['-alloc_objects','-list','printValue'])]:
  cmd=['go','tool','pprof','-nodecount=22','-source_path='+str(art/build),'-trim_path='+str(root)+':github.com/CloudyKit/jet/v6']+flags+[str(art/(build+'.test')),str(art/(label+'-heap.pprof'))]
  f=art/(label+'-'+view+'.txt');reused=f.exists()
  if reused:
   text=f.read_text();rc=prior.get(tuple(cmd),1 if text.strip()=='no matches found for regexp: printValue' else 0)
  else:
   r=subprocess.run(cmd,capture_output=True,text=True,timeout=25);text=r.stdout+r.stderr;rc=r.returncode;f.write_text(text)
  empty_source=label.startswith('sequential-') and view=='alloc-source' and rc in (1,2) and text.strip()=='no matches found for regexp: printValue'
  extra.append(dict(command=cmd,deadline=25,exit_code=rc,reused_completed_output=reused,no_sampled_printValue_allocations=empty_source));(art/'extra-inspection.json').write_text(json.dumps(extra,indent=2)+'\n');assert rc==0 or empty_source
print('Eight supplementary frozen-source views completed without new captures; sequential printValue has no sampled allocation source')
