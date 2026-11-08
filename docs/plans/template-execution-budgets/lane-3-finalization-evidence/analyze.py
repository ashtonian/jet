import subprocess,json,hashlib,re,statistics,time
from pathlib import Path
out=Path('.git/finalization');records=json.loads((out/'analysis-stages.json').read_text()) if (out/'analysis-stages.json').exists() else []
def command(label,args,timeout=120):
 if any(r['name']==label and r['exit_code']==0 for r in records):return
 start=time.time()
 with (out/(label+'.log')).open('w') as log: p=subprocess.run(args,stdout=log,stderr=subprocess.STDOUT,timeout=timeout)
 records.append({'name':label,'command':args,'exit_code':p.returncode,'start':start,'end':time.time(),'deadline':timeout})
 (out/'analysis-stages.json').write_text(json.dumps(records,indent=2)+'\n')
 if p.returncode and 'no matches found' not in (out/(label+'.log')).read_text():raise RuntimeError(label)
for phase in ['baseline','graph4','input4']:
 for suffix,flags in [('cpu-flat',['-top']),('cpu-cumulative',['-top','-cum']),('alloc',['-top','-sample_index=alloc_space']),('heap',['-top','-sample_index=inuse_space']),('mutex',['-top']),('block',['-top'])]:
  profile='heap' if suffix in ['alloc','heap'] else 'cpu' if suffix.startswith('cpu') else suffix
  command(phase+'-'+suffix,['go','tool','pprof',*flags,'-nodecount=30',str(out/(phase+'.test')),str(out/(phase+'-'+profile+'.pprof'))],30)
 for name in ['admitInputScopes','clone','admitCached','startCompilation','prepareSource']:
  command(phase+'-alloc-lines-'+name,['go','tool','pprof','-list='+name,'-sample_index=alloc_space',str(out/(phase+'.test')),str(out/(phase+'-heap.pprof'))],30)
pattern=re.compile(r'^(Benchmark\S+)\s+\d+\s+([\d.]+) ns/op\s+([\d.]+) B/op\s+(\d+) allocs/op',re.M)
result={'matched':{},'screen':{},'diamond':{}}
for phase in ['baseline','graph4','input4']:
 rows={}
 for p in out.glob('matched-'+phase+'-*.log'):
  for name,ns,bytes_,allocs in pattern.findall(p.read_text()):rows.setdefault(name,[]).append([float(ns),float(bytes_),int(allocs)])
 result['matched'][phase]={name:{'n':len(values),'ns':statistics.median(v[0] for v in values),'bytes':statistics.median(v[1] for v in values),'allocs':statistics.median(v[2] for v in values)} for name,values in rows.items()}
 combined=out/('matched-'+phase+'.txt');combined.write_text(''.join(p.read_text() for p in sorted(out.glob('matched-'+phase+'-*.log'))))
for phase in ['baseline','lazy0','input4','inline8','inline16','nopool']:
 rows={}
 for p in out.glob('screen-'+phase+'-*.log'):
  for name,ns,bytes_,allocs in pattern.findall(p.read_text()):rows.setdefault(name,[]).append([float(ns),float(bytes_),int(allocs)])
 result['screen'][phase]={name:{'n':len(values),'ns':statistics.median(v[0] for v in values),'bytes':statistics.median(v[1] for v in values),'allocs':statistics.median(v[2] for v in values)} for name,values in rows.items()}
for phase in ['graph4','walk']:
 values=pattern.findall((out/('diamond-'+phase+'.log')).read_text());result['diamond'][phase]={'ns':statistics.median(float(v[1]) for v in values),'bytes':statistics.median(float(v[2]) for v in values),'allocs':statistics.median(float(v[3]) for v in values),'n':len(values)}
(out/'results.json').write_text(json.dumps(result,indent=2)+'\n')
command('matched-benchstat',['benchstat',str(out/'matched-baseline.txt'),str(out/'matched-graph4.txt'),str(out/'matched-input4.txt')],30)
for phase in ['baseline','graph4','input4']:
 text=(out/(phase+'-alloc.log')).read_text();print(phase,'alloc summary');print('\n'.join(text.splitlines()[:16]))
print('analysis complete',flush=True)
