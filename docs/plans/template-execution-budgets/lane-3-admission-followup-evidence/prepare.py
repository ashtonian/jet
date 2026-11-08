import json,subprocess,hashlib
from pathlib import Path
root=Path.cwd();out=root/'.git/admission-followup'
base=json.loads((out/'base-overlay.json').read_text())['Replace']
harness=(root/'.git/finalization/strategy_benchmark.go.txt').read_text().replace('{"nested",nested}', '{"small_map",map[string]interface{}{"value":"content"}},{"nested",nested}')
(out/'strategy_benchmark.go.txt').write_text(harness)
variants={}
for name in ['base','split','direct']:
 folder=out/(name+'-frozen');folder.mkdir(exist_ok=True);replace={}
 for p in root.glob('*.go'):
  content=p.read_bytes()
  if p.name=='input.go':
   if name=='base':content=Path(base[str(p)]).read_bytes()
   if name=='split':content=(out/'split-input.go').read_bytes()
  target=folder/(p.name+'.txt');target.write_bytes(content);replace[str(p)]=str(target)
 replace[str(root/'admission_followup_benchmark_test.go')]=str(out/'strategy_benchmark.go.txt')
 overlay=out/(name+'-overlay.json');overlay.write_text(json.dumps({'Replace':replace},indent=2)+'\n');variants[name]={'overlay':str(overlay),'hashes':{k:hashlib.sha256(Path(v).read_bytes()).hexdigest() for k,v in replace.items()}}
manifest={'base':subprocess.check_output(['git','rev-parse','HEAD']).decode().strip(),'go':subprocess.check_output(['go','version']).decode().strip(),'goenv':json.loads(subprocess.check_output(['go','env','-json'])),'variants':variants,'data':{str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in [root/'go.mod',root/'go.sum',root/'testData/execution_workloads.json']}}
(out/'inputs.json').write_text(json.dumps(manifest,indent=2)+'\n')
