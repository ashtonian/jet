import hashlib,json,os,re,signal,subprocess,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';out=art/'matched';out.mkdir(exist_ok=True);builds=json.loads((art/'final-builds.json').read_text());peer=Path('<integration-artifacts>/release-54b5614');binaries={k:art/'final'/k/'render.test' for k in ['baseline','selected']};binaries['upstream']=peer/'results/upstream.test'
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
assert subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip()==builds['head'];assert all(sha(root/p)==h for p,h in builds['common_inputs'].items());hashes={k:sha(p) for k,p in binaries.items()};records=[];active=None
receipt=out/'results.json';records=json.loads(receipt.read_text())['steps'] if receipt.exists() else []
metadata=dict(head=builds['head'],binary_sha256=hashes,go_version=builds['toolchain'],GOMAXPROCS=2,GOGC=os.environ.get('GOGC','default 100'),GOMEMLIMIT=os.environ.get('GOMEMLIMIT','default off'),timing='6 interleaved 1s normal samples; profile processes separate',steps=records)
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
 try:p.wait(timeout=5)
 except subprocess.TimeoutExpired:raise RuntimeError('process did not terminate after group KILL')
def stop(sig,frame):
 if active is not None:stop_group(active)
 raise SystemExit(143)
signal.signal(signal.SIGTERM,stop)
signal.signal(signal.SIGINT,stop)
def run(name,variant,bench,expected,profile=False):
 global active
 flags=['-test.run=^$','-test.bench='+bench,'-test.benchtime=1s','-test.count=1','-test.timeout=25s']
 if profile:
  for kind,flag in [('cpu','cpuprofile'),('heap','memprofile'),('mutex','mutexprofile'),('block','blockprofile')]:flags.append('-test.'+flag+'='+str(out/(name+'-'+kind+'.pprof')))
  flags.append('-test.memprofilerate=524288')
 command=['/usr/bin/time','-l',str(binaries[variant])]+flags
 prior=next((r for r in reversed(records) if r['name']==name),None)
 if prior is not None and prior['exit_code']==0 and prior['command']==command and prior['binary_sha256']==hashes[variant] and sha(out/(name+'.log'))==prior['log_sha256']:
  print('reuse',name,flush=True);return
 start=time.time();metadata['active']={'name':name,'command':command,'started':start};receipt.write_text(json.dumps(metadata,indent=2)+'\n')
 with (out/(name+'.log')).open('w') as log:
  active=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,start_new_session=True,env=dict(os.environ,GOMAXPROCS='2'))
  try:rc=active.wait(timeout=30)
  except subprocess.TimeoutExpired:stop_group(active);rc=124
 active=None
 text=(out/(name+'.log')).read_text();lines=[l for l in text.splitlines() if re.match(r'^Benchmark\S+\s+\d+\s+[\d.]+ ns/op',l)];row=dict(name=name,variant=variant,command=command,exit_code=rc,started=start,finished=time.time(),deadline=30,kill_after=5,binary_sha256=hashes[variant],log_sha256=sha(out/(name+'.log')),expected_cells=expected,actual_cells=len(lines),profiling=profile)
 records.append(row);metadata.pop('active',None);receipt.write_text(json.dumps(metadata,indent=2)+'\n');print(name,rc,len(lines),flush=True);assert rc==0 and len(lines)==expected,(name,rc,len(lines))
patterns=[('text','(header_literal|inheritance|nested_writing|header_interpolation)','text',4),('escape','escape_expansion','html',1)]
modes=['legacy_stream','metered_background_stream','metered_live_stream','restricted_owned']
for sample in range(6):
 variants=['upstream','baseline','selected'] if sample%2==0 else ['selected','baseline','upstream']
 for variant in variants:
  for mode in (['legacy_stream','legacy_owned'] if variant=='upstream' else modes):
   for label,fixtures,fmt,count in patterns:
    run('time-'+variant+'-'+mode+'-'+label+'-'+str(sample),variant,'^BenchmarkExecutionModes$/^'+mode+'$/^'+fixtures+'$/^'+fmt+'$',count)
for variant in ['baseline','selected']:
 serial_modes=['restricted_owned'] if variant=='baseline' else modes
 for mode in serial_modes:
  for fixture,fmt in [('header_literal','text'),('inheritance','text'),('nested_writing','text'),('header_interpolation','text'),('escape_expansion','html')]:
   run('serial-'+variant+'-'+mode+'-'+fixture,variant,'^BenchmarkExecutionModes$/^'+mode+'$/^'+fixture+'$/^'+fmt+'$',1,True)
 for mode in modes:
  for fixture,fmt in [('header_interpolation','text'),('escape_expansion','html')]:
   run('concurrent-'+variant+'-'+mode+'-'+fixture,variant,'^BenchmarkConcurrentExecutionModes$/^'+mode+'$/^'+fixture+'$/^'+fmt+'$/^workers_2$/^shared_true$',1,True)
metadata['inputs_unchanged']=all(sha(root/p)==h for p,h in builds['common_inputs'].items());metadata['finished']=time.time();receipt.write_text(json.dumps(metadata,indent=2)+'\n');assert metadata['inputs_unchanged'];print('Complete: 300 timing cells, 41 paired/selected profile processes; initial15 serial baseline captures reused',flush=True)
