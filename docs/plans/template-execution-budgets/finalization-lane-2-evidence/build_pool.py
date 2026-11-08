import hashlib,json,os,subprocess
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization';names=['eval.go','execution_output.go'];original={p:Path(p).read_text() for p in names};stage=art/'reserve-reuse4096';stage.mkdir(exist_ok=True)
try:
 s=original['eval.go'].replace('outputOrigin *NodeBase','outputOrigin *NodeBase\n outputScratch []byte\n outputScratchUsed bool')
 s=s.replace('st.printer = dataPrinter{}','''st.printer = dataPrinter{}
 if st.outputScratchUsed {
  for i := range st.outputScratch { st.outputScratch[i] = 0 }
  st.outputScratchUsed = false
 }''',1);Path('eval.go').write_text(s)
 s=original['execution_output.go'];marker='st.reserveUnchecked(&budget.values, uint64(budget.options.MaxValueBytes), uint64(len(b)), "value", ErrValueLimit)';assert s.count(marker)==1
 s=s.replace(marker,marker+'''
 if len(b)>0 && len(b)<4096 && w.retained.Cap()==0 && w.retained==st.rootSink.retained {
  if st.outputScratch==nil { st.outputScratch=make([]byte,4096) }
  *w.retained=*bytes.NewBuffer(st.outputScratch[:0])
  st.outputScratchUsed=true
 }''');Path('execution_output.go').write_text(s)
 for name in names:(stage/name).write_text(Path(name).read_text())
 inputs={str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in list(Path('.').glob('*.go'))+[Path('go.mod'),Path('go.sum'),Path('testData/execution_workloads.json')]}
 binary=art/'reserve-reuse4096.test';command=['gate-lock','run-focused','jet-finalization-reuse4096-build','--','go','test','-p=1','-parallel=2','-c','-o',str(binary),'.']
 with (stage/'build.log').open('w') as log:
  rc=subprocess.call(command,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5'))
 stable=all(hashlib.sha256(Path(p).read_bytes()).hexdigest()==h for p,h in inputs.items())
 record=dict(command=command,source_inputs=inputs,exit_code=rc,inputs_stable=stable,execution_deadline=120,kill_after=5,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest() if rc==0 else None)
 (stage/'build.json').write_text(json.dumps(record,indent=2)+'\n');assert rc==0 and stable
finally:
 for name,text in original.items():Path(name).write_text(text)
 (stage/'restoration.json').write_text(json.dumps({'restored':all(Path(p).read_text()==v for p,v in original.items())})+'\n')
print('bounded 4096-byte reusable output variant built; sources restored')
