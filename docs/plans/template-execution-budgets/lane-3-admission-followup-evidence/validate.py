import subprocess,os,time,json,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/admission-followup'
files=subprocess.check_output(['git','ls-files','-z','--cached','--others','--exclude-standard']).decode().split('\0');manifest={p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in sorted(set(files)) if p and Path(p).is_file() and (p.endswith('.go') or p in ['go.mod','go.sum'] or p.startswith(('testData/','.github/')))}
(out/'validation-inputs.json').write_text(json.dumps({'head':subprocess.check_output(['git','rev-parse','HEAD']).decode().strip(),'files':manifest,'go':subprocess.check_output(['go','version']).decode().strip(),'goenv':json.loads(subprocess.check_output(['go','env','-json'])),'script_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest()},indent=2)+'\n')
receipts=[]
selector='^(TestInputAdmission|TestRestricted|TestCompilation|TestExecutionCompilation|TestTemplateBundle|TestExecutionModeConformance|TestExecutionMeasurementBounds)'
for name,mode,args in [('normal','run-focused',['go','test','-p=1','-parallel=2','-count=1','-timeout=90s','-run='+selector,'.']),('race','run-focused',['go','test','-race','-p=1','-parallel=2','-count=1','-timeout=90s','-run='+selector,'.']),('vet','run-check',['go','vet','-p=1','.'])]:
 cmd=['gate-lock',mode,'jet-lane3-admission-followup-'+name,'--']+args;print(name,flush=True);start=time.time()
 env=dict(os.environ,GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='180',GATE_LOCK_COMMAND_KILL_AFTER='5')
 with (out/('validation-'+name+'.log')).open('w') as log:code=subprocess.call(cmd,env=env,stdout=log,stderr=subprocess.STDOUT)
 receipts.append({'name':name,'command':cmd,'execution_deadline':180,'test_deadline':90 if name!='vet' else None,'term_cleanup':5,'queue_wait_separate':True,'elapsed_includes_queue':True,'start':start,'end':time.time(),'exit_code':code});(out/'validation-receipts.json').write_text(json.dumps(receipts,indent=2)+'\n')
print('VALIDATION COMPLETE',flush=True)
raise SystemExit(any(r['exit_code'] for r in receipts))
