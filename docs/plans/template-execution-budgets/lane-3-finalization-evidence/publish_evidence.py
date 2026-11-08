from pathlib import Path
import json,hashlib,difflib,subprocess,re
root=Path.cwd();raw=root/'.git/finalization';dest=root/'docs/plans/template-execution-budgets/lane-3-finalization-evidence';dest.mkdir(exist_ok=True)
def sanitize(text):
 lines=[line for line in text.splitlines() if not line.startswith('gate-lock: waiting for')]
 return ('\n'.join(lines)+'\n').replace(str(root),'$CHECKOUT').replace('$HOME/','$HOME/').replace(' on $HOST,',' on $HOST,')
hashes={}
# Complete tables, source views, samples and receipts; raw profiles/binaries stay local.
for p in raw.iterdir():
 if not p.is_file() or p.suffix not in ['.log','.json','.py','.txt']:continue
 if p.name.startswith(('implement_','prepare_pool','prepare_input','input-prepared','seeded_input_notes','active8-input','walk-compile','nopool-compile','lazy0-compile','inline8-compile','inline16-compile')):continue
 if p.name.endswith('-overlay.json') or p.name in ['latest-variants.json','followup-base.json']:continue
 if p.name in ['measure.py','refresh_overlays.py']:continue
 if p.name.endswith('.go.txt') and p.name not in ['benchmark.go.txt','candidate-modes.go.txt','strategy_benchmark.go.txt']:continue
 text=sanitize(p.read_text());target=dest/p.name;target.write_text(text);hashes[p.name]={'raw_sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'published_sha256':hashlib.sha256(target.read_bytes()).hexdigest()}
# Save independently reconstructable production deltas for every measured variant.
variants={}
for manifest in ['measurement-inputs.json','followup-inputs.json']:
 d=json.loads((raw/manifest).read_text())
 for phase,row in d['variants'].items():
  overlay=raw/(phase+'-overlay.json') if manifest.startswith('measurement') else raw/'followup-frozen'/(phase+'-overlay.json')
  replace=json.loads(overlay.read_text())['Replace'];patch=[]
  label=phase if manifest.startswith('measurement') else 'follow-'+phase
  sourcehashes={}
  for name,path in sorted(replace.items()):
   filename=Path(name).name
   if not name.startswith(str(root)+'/') or '/' in name[len(str(root))+1:] or not filename.endswith('.go') or filename.endswith('_test.go'):continue
   before=subprocess.run(['git','show','0e277b3:'+filename],capture_output=True,check=False)
   old=before.stdout.decode() if before.returncode==0 else '';new=Path(path).read_text()
   sourcehashes[filename]=hashlib.sha256(new.encode()).hexdigest()
   if old!=new:patch.extend(difflib.unified_diff(old.splitlines(True),new.splitlines(True),fromfile='a/'+filename,tofile='b/'+filename))
  target=dest/(label+'.patch');target.write_text(''.join(patch));variants[label]={'patch':target.name,'base':'0e277b3017b4afae1f984663c4e9244b4de0b290','production_sha256':sourcehashes,'manifest':manifest}
(dest/'variants.json').write_text(json.dumps(variants,indent=2)+'\n')
(dest/'artifact-hashes.json').write_text(json.dumps(hashes,indent=2)+'\n');print('portable artifacts',len(hashes),'variant patches',len(variants))
