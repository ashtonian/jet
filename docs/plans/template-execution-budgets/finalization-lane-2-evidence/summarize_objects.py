import json,re
from pathlib import Path
root=Path.cwd()/'.git/finalization';inspection=json.loads((root/'object-profile-inspection.json').read_text());rows=[]
for command in inspection['commands']:
 path=Path(command['output']);text=path.read_text();kind=re.search(r'^Type: (.+)$',text,re.M).group(1)
 nodes=[]
 for line in text.splitlines():
  m=re.match(r'^\s*([\d.]+[kMG]?)\s+([\d.]+)%\s+[\d.]+%\s+([\d.]+[kMG]?)\s+([\d.]+)%\s+(.+)$',line)
  if m:nodes.append(dict(flat=m[1],flat_percent=float(m[2]),cumulative=m[3],cumulative_percent=float(m[4]),symbol=m[5]))
 rows.append(dict(output=str(path),kind=kind,zero_total='0, 0% of 0 total' in text,top=nodes[:8],all_nodes=nodes))
(root/'object-profile-summary.json').write_text(json.dumps(rows,indent=2)+'\n')
print('Reviewed',len(rows),'views; zero inuse reports',sum(r['zero_total'] for r in rows if r['kind']=='inuse_objects'))
for r in rows:
 if r['output'].endswith('cumulative.txt'):continue
 print(r['output'], 'ZERO' if r['zero_total'] else '; '.join(x['flat']+' '+x['symbol'] for x in r['top'][:3]))
