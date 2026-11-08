import hashlib,json,os,subprocess,time
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';writer=(root/'execution_output.go').read_text();printing=(root/'execution_print.go').read_text();old='func (w *checkedWriter) Write(b []byte) (int, error) {';assert writer.count(old)==1
fast=writer.replace(old,'''func (w *checkedWriter) Write(b []byte) (int, error) {
	if w.runtime.execution.ctx == nil && w.runtime.execution.terminal == nil && w.buffer != nil {
		return w.buffer.Write(b)
	}
	return w.writeChecked(b)
}

func (w *checkedWriter) writeChecked(b []byte) (int, error) {''')
old='if v.Type().Implements(formatterType) || v.Type().Implements(stringerType) || v.Type().Implements(errorType) {';assert printing.count(old)==1
methodless=printing.replace(old,'if typ := v.Type(); typ.NumMethod() != 0 && (typ.Implements(formatterType) || typ.Implements(stringerType) || typ.Implements(errorType)) {')
inputs={p:hashlib.sha256((root/p).read_bytes()).hexdigest() for p in subprocess.check_output(['git','ls-files'],text=True).splitlines() if p.endswith('.go') or p in ('go.mod','go.sum','testData/execution_workloads.json')};records={}
for name,use_fast,use_methods in [('buffer-fast',True,False),('methodless',False,True),('both-output',True,True)]:
 d=art/'variants'/name;d.mkdir(exist_ok=True);replace={}
 for filename,s in [('execution_output.go',fast if use_fast else writer),('execution_print.go',methodless if use_methods else printing)]:
  f=d/filename;f.write_text(s);subprocess.run(['gofmt','-w',str(f)],check=True);replace[str(root/filename)]=str(f)
 overlay=d/'overlay.json';overlay.write_text(json.dumps({'Replace':replace})+'\n');binary=d/'render.test';cmd=['gate-lock','run-focused','jet-lane2-followup-build-'+name,'--','go','test','-p=1','-parallel=2','-overlay='+str(overlay),'-c','-o',str(binary),'.'];start=time.time()
 with (d/'build.log').open('w') as log:rc=subprocess.call(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5'))
 records[name]=dict(command=cmd,exit_code=rc,deadline=120,kill_after=5,elapsed=time.time()-start,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest() if rc==0 else None,source_sha256={p:hashlib.sha256(Path(f).read_bytes()).hexdigest() for p,f in replace.items()},overlay_sha256=hashlib.sha256(overlay.read_bytes()).hexdigest());(art/'output-builds.json').write_text(json.dumps(dict(common_inputs=inputs,variants=records),indent=2)+'\n');print(name,rc,flush=True);assert rc==0
assert all(hashlib.sha256((root/p).read_bytes()).hexdigest()==h for p,h in inputs.items())
