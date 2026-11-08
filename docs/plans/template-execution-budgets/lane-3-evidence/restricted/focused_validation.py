import hashlib,json,os,subprocess,sys,time,signal
from pathlib import Path
out=Path('.git/lane3-evidence');label=sys.argv[1];cmd=sys.argv[2:]
files=subprocess.check_output(['git','ls-files','-z','--cached','--others','--exclude-standard']).decode().split('\0')
manifest={p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in sorted(set(files)) if p and Path(p).is_file() and (p.endswith('.go') or p in ('go.mod','go.sum') or p.startswith(('testData/','.github/')))}
inputs={'head':subprocess.check_output(['git','rev-parse','HEAD']).decode().strip(),'files':manifest,'toolchain':subprocess.check_output(['go','version']).decode().strip(),'GOMAXPROCS':os.getenv('GOMAXPROCS'),'GOFLAGS':os.getenv('GOFLAGS'),'harness_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
for flag in cmd:
 if flag.startswith('-overlay='):
  overlay=Path(flag.split('=',1)[1]);inputs['overlay']={'file':str(overlay),'sha256':hashlib.sha256(overlay.read_bytes()).hexdigest(),'replacements':{p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in json.loads(overlay.read_text())['Replace'].values() if p}}
(out/(label+'-inputs.json')).write_text(json.dumps(inputs,indent=2)+'\n')
env=dict(os.environ, GATE_LOCK_WAIT='1',GATE_LOCK_COMMAND_TIMEOUT='180',GATE_LOCK_COMMAND_KILL_AFTER='5')
gate=['gate-lock','run-focused','jet-lane3-'+label,'--']
start=time.time();code=124
with (out/(label+'.log')).open('w') as log:
 p=subprocess.Popen(gate+cmd,stdout=log,stderr=subprocess.STDOUT,env=env)
 code=p.wait()
(out/(label+'-receipt.json')).write_text(json.dumps({'command':cmd,'execution_deadline_seconds':180,'queue_wait_in_stage_elapsed':True,'supervisor':'installed gate-lock','gate_deadline_seconds':180,'term_cleanup_seconds':5,'start':start,'end':time.time(),'exit_code':code,'manifest':label+'-inputs.json'},indent=2)+'\n')
print(label,'exit',code,flush=True)
raise SystemExit(code)
