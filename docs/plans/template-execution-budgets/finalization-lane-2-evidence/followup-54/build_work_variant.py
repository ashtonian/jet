import hashlib,json,os,subprocess
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';s=(root/'execution.go').read_text();start=s.index('func (st *Runtime) reserveWork(');end=s.index('\n// reserve checks',start)
s=s[:start]+'''func (st *Runtime) reserveWork(steps uint64) {
	b := &st.execution
	if b.terminal != nil || b.checkContext || (b.options.MaxSteps != 0 && steps > b.remaining) {
		st.checkWork(steps)
	}
	if b.ctx != nil && b.options.MaxSteps != 0 {
		b.remaining -= steps
	}
}

func (st *Runtime) checkWork(steps uint64) {
	st.rethrowTerminal()
	b := &st.execution
	if b.ctx == nil {
		return
	}
	if b.checkContext && b.ctx.Err() != nil {
		st.terminate(executionContextError(b.ctx))
	}
	if b.options.MaxSteps != 0 && steps > b.remaining {
		st.terminate(&LimitError{Kind: "steps", Limit: b.options.MaxSteps, cause: ErrStepLimit})
	}
}
'''+s[end:]
d=art/'variants/work-fast';d.mkdir(exist_ok=True);f=d/'execution.go';f.write_text(s);subprocess.run(['gofmt','-w',str(f)],check=True);o=d/'overlay.json';o.write_text(json.dumps({'Replace':{str(root/'execution.go'):str(f)}})+'\n');binary=d/'render.test';cmd=['gate-lock','run-focused','jet-lane2-followup-build-work-fast','--','go','test','-p=1','-parallel=2','-overlay='+str(o),'-c','-o',str(binary),'.']
with (d/'build.log').open('w') as log:rc=subprocess.call(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5'))
r=json.loads((art/'output-builds.json').read_text());r['variants']['work-fast']={'command':cmd,'exit_code':rc,'deadline':120,'kill_after':5,'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest() if rc==0 else None,'source_sha256':hashlib.sha256(f.read_bytes()).hexdigest(),'overlay_sha256':hashlib.sha256(o.read_bytes()).hexdigest()};(art/'output-builds.json').write_text(json.dumps(r,indent=2)+'\n');assert rc==0;print('work-fast: canonical charge with separated validation built')
