import pathlib,subprocess,json,re,statistics
root=pathlib.Path(__file__).parent
b=root/'bench'
if not (root/'benchmarks.json').exists() or json.loads((root/'benchmarks.json').read_text())['exit']!=0:raise RuntimeError('benchmark prerequisite incomplete')
for cpu in b.glob('profile-*-cpu.pprof'):
 name=cpu.name[:-len('-cpu.pprof')];tree=name.split('-')[1];binary=b/(tree+'.test')
 for label,flags,suffix in [('cpu-flat',[], 'cpu'),('cpu-cumulative',['-cum'],'cpu'),('alloc',['-alloc_space'],'heap'),('heap',['-inuse_space'],'heap'),('mutex',[],'mutex'),('block',[],'block')]:
  args=['go','tool','pprof','-top','-nodecount=18',*flags,str(binary),str(b/(name+'-'+suffix+'.pprof'))]
  p=subprocess.run(args,capture_output=True,text=True,timeout=30);(b/(name+'-'+label+'.txt')).write_text(p.stdout+p.stderr)
  if p.returncode:raise RuntimeError('pprof failed '+name+' '+label)
rows={}
for f in b.glob('render-*.txt'):
 tree=f.name.split('-')[1]
 for line in f.read_text().splitlines():
  m=re.match(r'BenchmarkFinalReview/([^/]+)/([^/]+)/(html|text)-2\s+\d+\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op',line)
  if m:rows.setdefault(tree+'/'+ '/'.join(m.group(1,2,3)),[]).append([float(m[4]),int(m[5]),int(m[6])])
medians={name:[statistics.median(v[i] for v in values) for i in range(3)] for name,values in rows.items()}
(b/'medians.json').write_text(json.dumps(medians,indent=2))
comparisons=[('legacy-vs-upstream','upstream','legacy_stream','candidate','legacy_stream'),('legacy-vs-correctness','correctness','legacy_stream','candidate','legacy_stream'),('background-vs-upstream','upstream','legacy_stream','candidate','metered_background_stream'),('live-vs-upstream','upstream','legacy_stream','candidate','metered_live_stream'),('restricted-vs-owned-upstream','upstream','legacy_owned','candidate','restricted_owned'),('restricted-vs-owned-candidate','candidate','legacy_owned','candidate','restricted_owned')]
summary={}
for title,base,bmode,new,nmode in comparisons:
 files=[]
 for tag,tree,mode in [('base',base,bmode),('new',new,nmode)]:
  lines=['goos: darwin','goarch: arm64','pkg: github.com/CloudyKit/jet/v6']
  for f in sorted(b.glob('render-'+tree+'-*.txt')):
   lines += [line.replace('/'+mode+'/','/') for line in f.read_text().splitlines() if line.startswith('BenchmarkFinalReview/'+mode+'/')]
  f=b/(title+'-'+tag+'.txt');f.write_text('\n'.join(lines)+'\n');files.append(str(f))
 p=subprocess.run(['benchstat',*files],capture_output=True,text=True,timeout=60)
 (b/(title+'-benchstat.txt')).write_text(p.stdout+p.stderr)
 if p.returncode:raise RuntimeError('benchstat failed')
 ratios={name.split('/',2)[2]:medians[new+'/'+nmode+'/'+name.split('/',2)[2]][0]/values[0] for name,values in medians.items() if name.startswith(base+'/'+bmode+'/')}
 summary[title]={'median_ratio':statistics.median(ratios.values()),'per_fixture':ratios}
(b/'ratios.json').write_text(json.dumps(summary,indent=2));print(json.dumps({k:v['median_ratio'] for k,v in summary.items()},indent=2))
print('INSPECTED ALL PROFILE KINDS; read generated reports before attributing costs')
