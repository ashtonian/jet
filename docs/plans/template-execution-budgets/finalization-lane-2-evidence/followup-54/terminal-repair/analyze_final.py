import hashlib,json,os,re,signal,statistics,subprocess,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';matched=art/'terminal-repair/matched';r=json.loads((matched/'results.json').read_text());assert r.get('inputs_unchanged') and len(r['steps'])==142 and all(s['exit_code']==0 for s in r['steps']);out=art/'terminal-repair/final-analysis';out.mkdir(exist_ok=True);data={};raw={};headers=[];active=None
for step in r['steps']:
 if step['profiling']:continue
 text=(matched/(step['name']+'.log')).read_text()
 if not headers:headers=[s for s in text.splitlines() if s.startswith(('goos:','goarch:','pkg:','cpu:'))]
 for line in text.splitlines():
  m=re.match(r'^BenchmarkExecutionModes/([^/]+)/([^/]+)/([^\s]+)-2\s+(\d+)\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op',line)
  if m:
   mode,fixture,fmt,n,ns,byte,alloc=m.groups();key=(step['variant'],mode,fixture,fmt);data.setdefault(key,[]).append([float(ns),int(byte),int(alloc)]);raw.setdefault(key,[]).append((n,ns,byte,alloc))
old=json.loads((art/'matched/results.json').read_text())
for step in old['steps']:
 if step['profiling'] or step['variant']!='upstream':continue
 for line in (art/'matched'/(step['name']+'.log')).read_text().splitlines():
  m=re.match(r'^BenchmarkExecutionModes/([^/]+)/([^/]+)/([^\s]+)-2\s+(\d+)\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op',line)
  if m:
   mode,fixture,fmt,n,ns,byte,alloc=m.groups();key=('upstream',mode,fixture,fmt);data.setdefault(key,[]).append([float(ns),int(byte),int(alloc)]);raw.setdefault(key,[]).append((n,ns,byte,alloc))
assert len(data)==53 and all(len(v)==6 for v in data.values())
summary=[]
for (variant,mode,fixture,fmt),values in data.items():
 if variant!='selected':continue
 base=data[('baseline',mode,fixture,fmt)];up=data[('upstream','legacy_owned' if mode=='restricted_owned' else 'legacy_stream',fixture,fmt)];med=lambda x:[statistics.median(v[i] for v in x) for i in range(3)]
 summary.append(dict(mode=mode,fixture=fixture,format=fmt,baseline=dict(samples=base,median=med(base)),selected=dict(samples=values,median=med(values)),upstream=dict(mode='legacy_owned' if mode=='restricted_owned' else 'legacy_stream',samples=up,median=med(up)),selected_vs_baseline=med(values)[0]/med(base)[0],selected_vs_upstream=med(values)[0]/med(up)[0]))
(out/'timings.json').write_text(json.dumps(summary,indent=2)+'\n')
for variant in ['baseline','selected']:
 lines=headers[:]
 for (v,mode,fixture,fmt),values in raw.items():
  if v==variant:
   for n,ns,byte,alloc in values:lines.append('BenchmarkRuntimeFollowup/'+mode+'/'+fixture+'/'+fmt+'-2 '+n+' '+ns+' ns/op '+byte+' B/op '+alloc+' allocs/op')
 (out/(variant+'.bench.txt')).write_text('\n'.join(lines)+'\n')
benchstat='<user>/go/bin/benchstat';bs=[]
cmd=[benchstat,str(out/'baseline.bench.txt'),str(out/'selected.bench.txt')];b=subprocess.run(cmd,capture_output=True,text=True,timeout=30);(out/'paired-benchstat.txt').write_text(b.stdout+b.stderr);assert b.returncode==0;bs.append(dict(command=cmd,exit_code=0,deadline=30))
for mode in ['legacy_stream','metered_background_stream','metered_live_stream','restricted_owned']:
 files=[]
 for variant,source_mode in [('upstream','legacy_owned' if mode=='restricted_owned' else 'legacy_stream'),('selected',mode)]:
  lines=headers[:]
  for (v,m,fixture,fmt),values in raw.items():
   if v==variant and m==source_mode:
    for n,ns,byte,alloc in values:lines.append('BenchmarkRuntimeFollowup/'+fixture+'/'+fmt+'-2 '+n+' '+ns+' ns/op '+byte+' B/op '+alloc+' allocs/op')
  f=out/(mode+'-'+variant+'.bench.txt');f.write_text('\n'.join(lines)+'\n');files.append(str(f))
 cmd=[benchstat]+files;b=subprocess.run(cmd,capture_output=True,text=True,timeout=30);(out/(mode+'-upstream-benchstat.txt')).write_text(b.stdout+b.stderr);assert b.returncode==0;bs.append(dict(command=cmd,exit_code=0,deadline=30))
for variant in ['previous','selected']:
 lines=headers[:]
 for (v,mode,fixture,fmt),values in raw.items():
  if v==variant and mode!='legacy_stream' and fixture=='escape_expansion':
   for n,ns,byte,alloc in values:lines.append('BenchmarkGuardRepair/'+mode+'-2 '+n+' '+ns+' ns/op '+byte+' B/op '+alloc+' allocs/op')
 (out/(variant+'-guard.bench.txt')).write_text('\n'.join(lines)+'\n')
cmd=[benchstat,str(out/'previous-guard.bench.txt'),str(out/'selected-guard.bench.txt')];b=subprocess.run(cmd,capture_output=True,text=True,timeout=30);(out/'guard-repair-benchstat.txt').write_text(b.stdout+b.stderr);assert b.returncode==0;bs.append(dict(command=cmd,exit_code=0,deadline=30))
repair=[]
for (v,mode,fixture,fmt),values in data.items():
 if v=='previous':repair.append(dict(mode=mode,previous=values,selected=data[('selected',mode,fixture,fmt)]))
(out/'guard-repair.json').write_text(json.dumps(repair,indent=2)+'\n')
(out/'benchstat-commands.json').write_text(json.dumps(bs,indent=2)+'\n')
def stop_group(p):
 try:os.killpg(p.pid,signal.SIGTERM)
 except ProcessLookupError:pass
 until=time.monotonic()+5
 while time.monotonic()<until:
  try:os.killpg(p.pid,0)
  except ProcessLookupError:break
  time.sleep(.05)
 try:os.killpg(p.pid,signal.SIGKILL)
 except ProcessLookupError:pass
 p.wait(timeout=5)
def stop(sig,frame):
 if active is not None:stop_group(active)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop);signal.signal(signal.SIGINT,stop)
inspection_file=out/'inspection.json';views=json.loads(inspection_file.read_text()) if inspection_file.exists() else []
for step in r['steps']:
 if not step['profiling']:continue
 name=step['name'];variant=step['variant'];binary=art/'terminal-repair/render.test';source=art/'terminal-repair/source';fallback='<integration-artifacts>/release-54b5614/candidate'
 for view,kind,flags in [('cpu-flat','cpu',['-top']),('cpu-cumulative','cpu',['-top','-cum']),('source','cpu',['-list','Template.*execute|Runtime.*recover|Runtime.*checkpoint|Runtime.*enter|checkedWriter.*Write|escapeeWriter.*Write|dataPrinter.*value']),('alloc-bytes','heap',['-top','-alloc_space']),('alloc-objects','heap',['-top','-alloc_objects']),('alloc-objects-cumulative','heap',['-top','-cum','-alloc_objects']),('inuse-bytes','heap',['-top','-inuse_space']),('inuse-objects','heap',['-top','-inuse_objects']),('inuse-objects-cumulative','heap',['-top','-cum','-inuse_objects']),('mutex','mutex',['-top']),('block','block',['-top'])]:
  profile=matched/(name+'-'+kind+'.pprof');f=out/(name+'-'+view+'.txt');cmd=['go','tool','pprof','-nodecount=30','-source_path='+str(source)+':'+fallback,'-trim_path='+str(root)+':github.com/CloudyKit/jet/v6']+flags+[str(binary),str(profile)];ph=hashlib.sha256(profile.read_bytes()).hexdigest();prior=next((v for v in reversed(views) if v['output']==str(f)),None)
  if prior and prior['exit_code']==0 and prior['command']==cmd and prior['profile_sha256']==ph and f.exists() and hashlib.sha256(f.read_bytes()).hexdigest()==prior['output_sha256']:continue
  active=subprocess.Popen(cmd,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,start_new_session=True)
  try:text,_=active.communicate(timeout=25);rc=active.returncode
  except subprocess.TimeoutExpired:stop_group(active);text,_=active.communicate(timeout=5);rc=124
  active=None;f.write_text(text);views.append(dict(command=cmd,exit_code=rc,deadline=25,kill_after=5,output=str(f),profile_sha256=ph,output_sha256=hashlib.sha256(f.read_bytes()).hexdigest()));inspection_file.write_text(json.dumps(views,indent=2)+'\n');assert rc==0,(name,view,rc)
 print('inspected',name,flush=True)
print('20 repaired-source subset comparisons; prior upstream reference retained; 308 current profile views plus reused baseline captures')
