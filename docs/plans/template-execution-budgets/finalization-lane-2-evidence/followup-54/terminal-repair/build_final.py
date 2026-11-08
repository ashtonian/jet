import hashlib,json,os,subprocess,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54/terminal-repair';paths=set(subprocess.check_output(['git','ls-files'],text=True).splitlines());paths.update(str(p) for p in Path('.').glob('*.go'));inputs={p:hashlib.sha256((root/p).read_bytes()).hexdigest() for p in sorted(paths) if p.endswith('.go') or p in ('go.mod','go.sum') or p.startswith('testData/')};binary=art/'render.test';cmd=['gate-lock','run-focused','jet-lane2-terminal-repair-build','--','go','test','-p=1','-parallel=2','-c','-o',str(binary),'.'];start=time.time()
with (art/'build.log').open('w') as log:rc=subprocess.call(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5'))
assert rc==0
for name in ['eval.go','exec.go','execution.go','execution_print.go','execution_output.go','restricted.go','set.go']:
 f=art/'source'/name;f.parent.mkdir(exist_ok=True);f.write_bytes((root/name).read_bytes())
r=dict(head=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),common_inputs=inputs,command=cmd,exit_code=rc,deadline=120,kill_after=5,elapsed=time.time()-start,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),go_version=subprocess.check_output(['go','version'],text=True).strip(),go_env=json.loads(subprocess.check_output(['go','env','-json','GOOS','GOARCH','GOROOT','GOFLAGS','CGO_ENABLED','GOTOOLCHAIN'],text=True)))
assert all(hashlib.sha256((root/p).read_bytes()).hexdigest()==h for p,h in inputs.items());r['inputs_unchanged']=True;(art/'build.json').write_text(json.dumps(r,indent=2)+'\n');print('Repaired source built and frozen:',r['head'])
