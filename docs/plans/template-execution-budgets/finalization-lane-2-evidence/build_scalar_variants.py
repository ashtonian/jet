import hashlib,json,os,subprocess
from pathlib import Path
root=Path.cwd();out=root/'.git/finalization/scalar-repair';out.mkdir(exist_ok=True)
source=root/'execution_print.go';fixed=source.read_text();unsafe=subprocess.check_output(['git','show','56d25c8:execution_print.go'],text=True)
start=fixed.index('\tp := &st.printer');end=fixed.index('\n\terr := p.value',start)
local=fixed[:start]+'\tp := &dataPrinter{runtime: st, writer: w}\n\tdefer func() { *p = dataPrinter{} }()'+fixed[end:]
versions={'unsafe':unsafe,'lease':fixed,'local':local}
inputs={str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in list(Path('.').glob('*.go'))+[Path('go.mod'),Path('go.sum')]}
for name,text in versions.items():
 stage=out/name;stage.mkdir(exist_ok=True);snapshot=stage/'execution_print.go';snapshot.write_text(text);overlay=stage/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(source):str(snapshot)}})+'\n')
 binary=out/(name+'.test');cmd=['gate-lock','run-focused','jet-scalar-repair-build-'+name,'--','go','test','-p=1','-parallel=2','-overlay='+str(overlay),'-c','-o',str(binary),'.']
 with (stage/'build.log').open('w') as log:rc=subprocess.call(cmd,stdout=log,stderr=subprocess.STDOUT,env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='120',GATE_LOCK_COMMAND_KILL_AFTER='5'))
 record=dict(command=cmd,exit_code=rc,deadline=120,kill_after=5,common_inputs=inputs,overlay_sha256=hashlib.sha256(overlay.read_bytes()).hexdigest(),source_sha256=hashlib.sha256(text.encode()).hexdigest(),binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest() if rc==0 else None)
 (stage/'build.json').write_text(json.dumps(record,indent=2)+'\n');assert rc==0
assert all(hashlib.sha256(Path(p).read_bytes()).hexdigest()==h for p,h in inputs.items())
print('unsafe sequential control, safe invocation-local control and repaired workspace lease binaries built; source unchanged')
