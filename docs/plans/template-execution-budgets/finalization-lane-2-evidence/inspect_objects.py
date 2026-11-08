import hashlib,json,subprocess,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization';receipts=[];hashes={};groups=[]
for build in ['baseline','selected']:
 for mode in ['legacy_stream','metered_live_stream','restricted_owned']:
  prefix=art/'selected-measure'/(build+'-'+mode)
  groups.append((str(prefix),art/(build+'.test'),Path(str(prefix)+'-heap.pprof'),art/'profile-source'/build))
for variant in ['0','256','1024','4096','reuse4096']:
 stage=art/('reserve-'+variant)
 for profile in ['heap','large-small']:
  groups.append((str(stage/('objects' if profile=='heap' else 'large-small-objects')),art/('reserve-'+variant+'.test'),stage/(profile+'.pprof'),None))
for build in ['baseline','fixed']:
 for mode in ['legacy_stream','metered_live_stream','restricted_owned']:
  stage=art/'profiles'/(build+'-'+mode)
  groups.append((str(stage/'objects'),art/(build+'.test'),stage/'heap.pprof',art/'profile-source'/build))
for prefix,binary,profile,snapshot in groups:
 for p in [binary,profile]:
  assert p.is_file(),p
  hashes[str(p.relative_to(root))]=hashlib.sha256(p.read_bytes()).hexdigest()
 for view,flags in [('alloc-objects',['-top','-alloc_objects']),('alloc-objects-cumulative',['-top','-cum','-alloc_objects']),('inuse-objects',['-top','-inuse_objects']),('inuse-objects-cumulative',['-top','-cum','-inuse_objects'])]:
  command=['go','tool','pprof','-nodecount=25']+flags+[str(binary),str(profile)]
  started=time.time();result=subprocess.run(command,capture_output=True,text=True,timeout=25)
  output=Path(prefix+'-'+view+'.txt');output.write_text(result.stdout+result.stderr)
  receipts.append(dict(command=command,exit_code=result.returncode,output=str(output.relative_to(root)),started=started,finished=time.time(),execution_deadline_seconds=25))
  (art/'object-profile-inspection.json').write_text(json.dumps({'raw_inputs_sha256':hashes,'commands':receipts},indent=2)+'\n')
  assert result.returncode==0,(profile,view,result.stderr)
assert all(hashlib.sha256((root/p).read_bytes()).hexdigest()==h for p,h in hashes.items())
(art/'object-profile-source-proof.json').write_text(json.dumps({'raw_inputs_unchanged':True,'captures_reused':True,'new_benchmark_runs':0,'object_views':len(receipts),'raw_inputs_sha256':hashes},indent=2)+'\n')
print('Generated',len(receipts),'object-count flat/cumulative views from',len(groups),'existing heaps; capture/binary hashes unchanged; no benchmarks replayed')
