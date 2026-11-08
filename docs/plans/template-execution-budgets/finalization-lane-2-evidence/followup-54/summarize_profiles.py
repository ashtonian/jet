import hashlib,json,re
from pathlib import Path
art=Path.cwd()/'.git/finalization/followup-54';captures=[]
def rows(p):
 text=p.read_text();nodes=[]
 for line in text.splitlines():
  f=line.split(maxsplit=5)
  if len(f)==6 and f[1].endswith('%') and f[2].endswith('%') and f[4].endswith('%'):
   try:nodes.append(dict(flat=f[0],flat_percent=float(f[1][:-1]),cumulative=f[3],cumulative_percent=float(f[4][:-1]),name=f[5]))
   except ValueError:pass
 return text,nodes
base=json.loads((art/'baseline.json').read_text())
entries=[('serial-baseline-'+x['name'],art/x['name'],art/'analysis'/x['name']) for x in base['steps']]
matched=json.loads((art/'matched/results.json').read_text());entries +=[(x['name'],art/'matched'/x['name'],art/'final-analysis'/x['name']) for x in matched['steps'] if x['profiling']]
for name,prefix,views in entries:
 d=dict(name=name)
 for view in ['cpu-flat','cpu-cumulative','alloc-bytes','alloc-objects','alloc-objects-cumulative','inuse-bytes','inuse-objects','inuse-objects-cumulative','mutex','block']:
  text,ns=rows(Path(str(views)+'-'+view+'.txt'));d[view]={'showing':next((l for l in text.splitlines() if l.startswith('Showing nodes accounting')),''),'top':ns[:3],'jet_top':[n for n in ns if 'github.com/CloudyKit/jet/v6.(' in n['name']][:5]}
  if view=='cpu-flat':d[view]['work_and_output']=[n for n in ns if any(s in n['name'] for s in ['reserveWork','checkpoint','cancelCtx','checkedWriter','recover','enterActive','dataPrinter'])]
 text=Path(str(prefix)+'.log').read_text();d['peak_rss_bytes']=int(re.search(r'(\d+)\s+maximum resident set size',text).group(1));captures.append(d)
assert len(captures)==56
(art/'final-analysis/profile-summary.json').write_text(json.dumps(captures,indent=2)+'\n')
for p in (art/'final-analysis').glob('*-source.txt'):
 s=p.read_text();assert 'Error: could not' not in s,p
 if '-selected-' in p.name and 'func (t *Template) execute(' in s:assert 'Legacy execution uses the state already cleared' in s,p
 if '-baseline-' in p.name and 'func (t *Template) execute(' in s:assert 'st.execution = executionState{' in s,p
print('56 captures cross-checked: CPU flat/cumulative, allocation/live bytes and objects, contention, RSS and frozen source mapping')
