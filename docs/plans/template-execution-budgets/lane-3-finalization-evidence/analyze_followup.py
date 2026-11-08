import subprocess,json,re,statistics,time,hashlib
from pathlib import Path
out=Path('.git/finalization');records=[]
def stage(name,args):
 start=time.time()
 with (out/(name+'.log')).open('w') as log:r=subprocess.run(args,stdout=log,stderr=subprocess.STDOUT,timeout=30)
 records.append({'name':name,'command':args,'start':start,'end':time.time(),'exit_code':r.returncode})
 (out/'follow-analysis-stages.json').write_text(json.dumps(records,indent=2)+'\n')
 text=(out/(name+'.log')).read_text()
 if r.returncode and 'no matches found' not in text:raise RuntimeError(name)
for phase in ['base4','pooled4','active8']:
 for kind,flags in [('cpu-flat',['-top']),('cpu-cumulative',['-top','-cum']),('alloc',['-top','-sample_index=alloc_space']),('heap',['-top','-sample_index=inuse_space']),('mutex',['-top']),('block',['-top'])]:
  suffix='cpu' if kind.startswith('cpu') else 'heap' if kind in ['alloc','heap'] else kind
  stage('follow-'+phase+'-'+kind,['go','tool','pprof',*flags,'-nodecount=30',str(out/('follow-'+phase+'.test')),str(out/('follow-'+phase+'-'+suffix+'.pprof'))])
 for name in ['admitInputScopes','borrowInputVars','clone']:
  stage('follow-'+phase+'-lines-'+name,['go','tool','pprof','-source_path='+str(Path.cwd()),'-trim_path=github.com/CloudyKit/jet/v6/','-list='+name,'-sample_index=alloc_space',str(out/('follow-'+phase+'.test')),str(out/('follow-'+phase+'-heap.pprof'))])
pattern=re.compile(r'^(Benchmark\S+)\s+\d+\s+([\d.]+) ns/op\s+([\d.]+) B/op\s+(\d+) allocs/op',re.M)
result={'capacity':{},'reuse':{}}
for kind,phases,prefix in [('capacity',['base4','cap8','cap16'],'follow-capacity-'),('reuse',['base4','pooled4','active8'],'follow-pair-')]:
 for phase in phases:
  rows={};parts=[]
  for p in sorted(out.glob(prefix+phase+'-*.log')):
   parts.append(p.read_text())
   for name,ns,bytes_,allocs in pattern.findall(parts[-1]):rows.setdefault(name,[]).append([float(ns),float(bytes_),int(allocs)])
  result[kind][phase]={name:{'n':len(values),'ns':statistics.median(v[0] for v in values),'bytes':statistics.median(v[1] for v in values),'allocs':statistics.median(v[2] for v in values)} for name,values in rows.items()}
  (out/(prefix+phase+'.txt')).write_text(''.join(parts))
(out/'follow-results.json').write_text(json.dumps(result,indent=2)+'\n')
stage('follow-pair-benchstat',['benchstat',str(out/'follow-pair-base4.txt'),str(out/'follow-pair-pooled4.txt'),str(out/'follow-pair-active8.txt')])
stage('follow-capacity-benchstat',['benchstat',str(out/'follow-capacity-base4.txt'),str(out/'follow-capacity-cap8.txt'),str(out/'follow-capacity-cap16.txt')])
print(json.dumps(result,indent=2))
