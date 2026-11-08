import hashlib,json,os,subprocess,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';files=['exec.go','eval.go','execution.go'];current={p:(root/p).read_bytes() for p in files};baseline={p:(art/'baseline-source'/p).read_bytes() for p in files};builds={}
inputs={p:hashlib.sha256((root/p).read_bytes()).hexdigest() for p in subprocess.check_output(['git','ls-files'],text=True).splitlines() if p.endswith('.go') or p in ('go.mod','go.sum','testData/execution_workloads.json')}
for name,changed in [('baseline',[]),('setup',['exec.go']),('scope',['eval.go']),('dispatch',['execution.go']),('combined',files)]:
 d=art/'variants'/name;d.mkdir(parents=True,exist_ok=True);replace={}
 for p in files:
  f=d/p;f.write_bytes(current[p] if p in changed else baseline[p]);replace[str(root/p)]=str(f)
 overlay=d/'overlay.json';overlay.write_text(json.dumps({'Replace':replace})+'\n');binary=d/'render.test';cmd=['gate-lock','run-focused','jet-lane2-followup-build-'+name,'--','go','test','-p=1','-parallel=2','-overlay='+str(overlay),'-c','-o',str(binary),'.'];start=time.time()
 with (d/'build.log').open('w') as log:rc=subprocess.call(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5'))
 builds[name]=dict(command=cmd,exit_code=rc,deadline=120,kill_after=5,elapsed=time.time()-start,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest() if rc==0 else None,sources={p:hashlib.sha256((d/p).read_bytes()).hexdigest() for p in files},overlay_sha256=hashlib.sha256(overlay.read_bytes()).hexdigest());(art/'builds.json').write_text(json.dumps(dict(common_inputs=inputs,variants=builds),indent=2)+'\n');print(name,rc,flush=True);assert rc==0
assert all(hashlib.sha256((root/p).read_bytes()).hexdigest()==h for p,h in inputs.items())
