import hashlib,json,re,statistics,subprocess
from pathlib import Path
root=Path.cwd()/'.git/finalization/scalar-repair';records=json.loads((root/'measurements.json').read_text());assert len(records)==28 and all(x['exit_code']==0 for x in records)
for family,builds in [('sequential',['unsafe','lease']),('reentry',['local','lease'])]:
 for build in builds:(root/(family+'-'+build+'.txt')).write_text(''.join((root/(family+'-'+build+'-'+str(i)+'.log')).read_text() for i in range(6)))
 cmd=['<user>/go/bin/benchstat']+[str(root/(family+'-'+build+'.txt')) for build in builds]
 r=subprocess.run(cmd,capture_output=True,text=True,timeout=60);(root/(family+'-benchstat.txt')).write_text(r.stdout+r.stderr);assert r.returncode==0
summary=[]
for family,builds in [('sequential',['unsafe','lease']),('reentry',['local','lease'])]:
 names={}
 for build in builds:
  for sample in range(6):
   for line in (root/(family+'-'+build+'-'+str(sample)+'.log')).read_text().splitlines():
    m=re.match(r'^(BenchmarkScalar\S+)\s+\d+\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op',line)
    if m:names.setdefault(m[1],{}).setdefault(build,[]).append(tuple(map(float,m.groups()[1:])))
 for name,data in names.items():
  assert all(len(v)==6 for v in data.values())
  summary.append(dict(family=family,fixture=name,data={b:dict(samples=v,ns=statistics.median(x[0] for x in v),bytes=statistics.median(x[1] for x in v),allocs=statistics.median(x[2] for x in v)) for b,v in data.items()}))
(root/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
views=[]
for label,build in [('sequential-unsafe','unsafe'),('sequential-lease','lease'),('reentry-local','local'),('reentry-lease','lease')]:
 for view,kind,flags in [('cpu-flat','cpu',['-top']),('cpu-cumulative','cpu',['-top','-cum']),('source','cpu',['-list','printValue|dataPrinter.*Write|dataPrinter.*value']),('alloc-bytes','heap',['-top','-alloc_space']),('alloc-objects','heap',['-top','-alloc_objects']),('alloc-objects-cumulative','heap',['-top','-cum','-alloc_objects']),('inuse-bytes','heap',['-top','-inuse_space']),('inuse-objects','heap',['-top','-inuse_objects']),('mutex','mutex',['-top']),('block','block',['-top'])]:
  cmd=['go','tool','pprof','-nodecount=22','-source_path='+str(root/build),'-trim_path='+str(Path.cwd())+':github.com/CloudyKit/jet/v6']+flags+[str(root/(build+'.test')),str(root/(label+'-'+kind+'.pprof'))]
  r=subprocess.run(cmd,capture_output=True,text=True,timeout=25);(root/(label+'-'+view+'.txt')).write_text(r.stdout+r.stderr);views.append(dict(command=cmd,deadline=25,exit_code=r.returncode));assert r.returncode==0
(root/'inspection.json').write_text(json.dumps(views,indent=2)+'\n')
print('All affected scalar profiles and interleaved timing samples inspected; no invalid-output timing compared')
