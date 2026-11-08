import hashlib,json,os,signal,subprocess,sys,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization';source=root/'execution_output.go';original=source.read_text();p=None
marker='st.reserveUnchecked(&budget.values, uint64(budget.options.MaxValueBytes), uint64(len(b)), "value", ErrValueLimit)'
assert original.count(marker)==1

def stop(signum,frame):
 if p is not None and p.poll() is None:
  p.terminate()
 raise KeyboardInterrupt()
signal.signal(signal.SIGTERM,stop)
records=[]
try:
 for capacity in [0,256,1024,4096]:
  variant=original if capacity==0 else original.replace(marker,marker+'\n if len(b) > 0 && len(b) < '+str(capacity)+' && w.retained.Cap() == 0 { w.retained.Grow('+str(capacity)+') }',1)
  source.write_text(variant)
  stage=art/('reserve-'+str(capacity));stage.mkdir(exist_ok=True)
  (stage/'execution_output.go').write_text(variant)
  paths=list(root.glob('*.go'))+[root/'go.mod',root/'go.sum',root/'testData/execution_workloads.json']
  manifest={str(x.relative_to(root)):hashlib.sha256(x.read_bytes()).hexdigest() for x in paths}
  binary=art/('reserve-'+str(capacity)+'.test')
  command=['gate-lock','run-focused','jet-finalization-reserve-build-'+str(capacity),'--','go','test','-p=1','-parallel=2','-c','-o',str(binary),'.']
  env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5')
  with (stage/'build.log').open('w') as log:
   p=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT,env=env)
   rc=p.wait()
  stable=all(hashlib.sha256((root/x).read_bytes()).hexdigest()==h for x,h in manifest.items())
  record=dict(capacity=capacity,command=command,exit_code=rc,source_inputs=manifest,inputs_stable=stable,execution_deadline=120,kill_after=5,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest() if rc==0 else None)
  (stage/'build.json').write_text(json.dumps(record,indent=2)+'\n');records.append(record)
  assert stable and rc==0,(capacity,rc,stable)
finally:
 source.write_text(original)
 (art/'reserve-build-results.json').write_text(json.dumps({'restored':source.read_text()==original,'builds':records},indent=2)+'\n')
print('all four variants built; original source restored')
