import hashlib,json,os,subprocess,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';paths=set(subprocess.check_output(['git','ls-files'],text=True).splitlines());paths.update(str(p) for p in Path('.').glob('*.go'));inputs={p:hashlib.sha256((root/p).read_bytes()).hexdigest() for p in sorted(paths) if p.endswith('.go') or p in ('go.mod','go.sum') or p.startswith('testData/')};records={}
for name in ['baseline','selected']:
 d=art/'final'/name;d.mkdir(parents=True,exist_ok=True);binary=d/'render.test';cmd=['gate-lock','run-focused','jet-lane2-followup-final-build-'+name,'--','go','test','-p=1','-parallel=2']
 if name=='baseline':cmd+=['-overlay='+str(art/'baseline-overlay.json')]
 cmd+=['-c','-o',str(binary),'.'];start=time.time()
 with (d/'build.log').open('w') as log:rc=subprocess.call(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5'))
 records[name]=dict(command=cmd,exit_code=rc,deadline=120,kill_after=5,elapsed=time.time()-start,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest() if rc==0 else None);print(name,rc,flush=True);assert rc==0
for p in ['eval.go','exec.go','execution.go','execution_print.go','execution_output.go','restricted.go','set.go']:
 f=art/'selected-source'/p;f.parent.mkdir(exist_ok=True);f.write_bytes((root/p).read_bytes())
proof=dict(head=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),common_inputs=inputs,variants=records,baseline_overlay=json.loads((art/'baseline-overlay.json').read_text()),overlay_sha256=hashlib.sha256((art/'baseline-overlay.json').read_bytes()).hexdigest(),toolchain=subprocess.check_output(['go','version'],text=True).strip(),go_env=json.loads(subprocess.check_output(['go','env','-json','GOOS','GOARCH','GOROOT','GOFLAGS','CGO_ENABLED','GOTOOLCHAIN'],text=True)))
assert all(hashlib.sha256((root/p).read_bytes()).hexdigest()==h for p,h in inputs.items());proof['inputs_unchanged']=True;(art/'final-builds.json').write_text(json.dumps(proof,indent=2)+'\n')
