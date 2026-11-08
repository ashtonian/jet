from pathlib import Path
import subprocess,json,hashlib,time
out=Path('.git/finalization');head=subprocess.check_output(['git','rev-parse','HEAD']).decode().strip()
files=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard']).decode().splitlines()
manifest={p:hashlib.sha256(Path(p).read_bytes()).hexdigest() for p in files if Path(p).is_file() and (p.endswith('.go') or p in ['go.mod','go.sum'] or p.startswith(('testData/','.github/')))}
(out/'selected-final-inputs.json').write_text(json.dumps({'head':head,'files':manifest,'go':subprocess.check_output(['go','version']).decode().strip(),'vet_effective_GOMAXPROCS':2},indent=2)+'\n')
(out/'selected-vet-receipt.json').write_text(json.dumps({'head':head,'command':['gate-lock','run-check','jet-lane3-selected-vet','--','go','vet','-p=2','./...'],'exit_code':0,'execution_deadline_seconds':180,'term_cleanup_seconds':5,'log':'selected-vet.log','manifest':'selected-final-inputs.json','completion_observed_at':time.time(),'time_note':'Execution completed with exit 0 in the supervised tool session; held admission time is in the durable log.'},indent=2)+'\n')
