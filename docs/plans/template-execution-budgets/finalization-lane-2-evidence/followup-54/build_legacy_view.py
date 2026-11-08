import hashlib,json,os,subprocess
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';sources={p:(root/p).read_text() for p in ['exec.go','eval.go','execution_output.go']}
s=sources['execution_output.go'];marker='func (w *checkedWriter) Write(b []byte) (int, error) {';assert marker in s
s=s.replace(marker,'''// legacyBufferWriter shares the checked destination and latches, with no active
// meter. Only runtime setup selects this view for a concrete bytes.Buffer.
type legacyBufferWriter checkedWriter

func (w *legacyBufferWriter) Write(b []byte) (int, error) {
	if w.runtime.execution.terminal == nil {
		return w.buffer.Write(b)
	}
	return (*checkedWriter)(w).Write(b)
}

func (st *Runtime) checkedDestination(w io.Writer) *checkedWriter {
	var sink *checkedWriter
	switch w := w.(type) {
	case *checkedWriter:
		sink = w
	case *legacyBufferWriter:
		sink = (*checkedWriter)(w)
	}
	if sink != nil && sink.runtime == st {
		return sink
	}
	return nil
}

'''+marker)
s=s.replace('if parent, ok := tx.parent.(*checkedWriter); ok && parent.runtime == st {','if parent := st.checkedDestination(tx.parent); parent != nil {');sources['execution_output.go']=s
s=sources['exec.go'];s=s.replace('\tst.Writer = &st.rootSink','\tif ctx == nil && st.rootSink.buffer != nil {\n\t\tst.Writer = (*legacyBufferWriter)(&st.rootSink)\n\t} else {\n\t\tst.Writer = &st.rootSink\n\t}');sources['exec.go']=s
s=sources['eval.go'];s=s.replace('if sink, ok := w.Writer.(*checkedWriter); !ok || sink.runtime != w.rootSink.runtime {','if w.rootSink.runtime.checkedDestination(w.Writer) == nil {');sources['eval.go']=s
d=art/'variants/legacy-view';d.mkdir(exist_ok=True);replace={}
for name,s in sources.items():
 f=d/name;f.write_text(s);subprocess.run(['gofmt','-w',str(f)],check=True);replace[str(root/name)]=str(f)
o=d/'overlay.json';o.write_text(json.dumps({'Replace':replace})+'\n');binary=d/'render.test';cmd=['gate-lock','run-focused','jet-lane2-followup-build-legacy-view','--','go','test','-p=1','-parallel=2','-overlay='+str(o),'-c','-o',str(binary),'.']
with (d/'build.log').open('w') as log:rc=subprocess.call(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5'))
r=json.loads((art/'output-builds.json').read_text());r['variants']['legacy-view']={'command':cmd,'exit_code':rc,'deadline':120,'kill_after':5,'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest() if rc==0 else None,'source_sha256':{p:hashlib.sha256(Path(f).read_bytes()).hexdigest() for p,f in replace.items()},'overlay_sha256':hashlib.sha256(o.read_bytes()).hexdigest()};(art/'output-builds.json').write_text(json.dumps(r,indent=2)+'\n');assert rc==0;print('legacy-view built with shared checked destination and latches; active path unchanged')
