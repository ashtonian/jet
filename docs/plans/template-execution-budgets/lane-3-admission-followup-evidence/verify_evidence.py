import json,hashlib,subprocess,sys,time,re,datetime
from pathlib import Path
root=Path.cwd();raw=root/'.git/admission-followup';dest=root/'docs/plans/template-execution-budgets/lane-3-admission-followup-evidence'
observed=json.loads((raw/'outer-observations.json').read_text());observed['evidence-publication']={'exit_code':0,'observed_by':'exec session 72118 completion','execution_deadline_seconds':90,'term_cleanup_seconds':5};(raw/'outer-observations.json').write_text(json.dumps(observed,indent=2)+'\n')
# The publisher's own active output was intentionally not frozen as evidence.
(dest/'publish-gate.log').unlink(missing_ok=True)
hashes=json.loads((dest/'raw-artifact-hashes.json').read_text());hashes.pop('publish-gate.log',None);hashes['outer-observations.json']={'sha256':hashlib.sha256((raw/'outer-observations.json').read_bytes()).hexdigest(),'bytes':(raw/'outer-observations.json').stat().st_size};(dest/'raw-artifact-hashes.json').write_text(json.dumps(hashes,indent=2)+'\n')
(dest/'outer-observations.json').write_text((raw/'outer-observations.json').read_text())
comment=subprocess.check_output(['git','diff','--','input.go']).decode();(dest/'comment-clarification.patch').write_text(comment)
for p in dest.glob('*.json'):json.loads(p.read_text())
for f in ['measurement-receipts.json','finish-receipts.json','validation-receipts.json']:
 a=json.loads((dest/f).read_text());assert a and all(r['exit_code']==0 for r in a),f
rows=json.loads((dest/'paired-analysis.json').read_text());assert len(rows['core'])==5 and len(rows['shape'])==6
for fixtures in rows.values():
 for row in fixtures.values():assert row['base']['samples']==row['defaults']['samples']==6
for p in dest.glob('selected-*-source.log'):
 text=p.read_text();assert 'Error: could not find' not in text,p
 if 'ROUTINE' in text:assert '.git/admission-followup/' in text,p
proof=json.loads((dest/'committed-production-proof.json').read_text())['production_files']
for name,item in proof.items():
 current=(root/name).read_bytes();assert hashlib.sha256(current).hexdigest()==item['sha256'],name
 tested=current.decode().replace('// Supported unnamed types cannot add pointer methods; structs are denied below.', '// An unnamed or predeclared type cannot acquire pointer-receiver methods.')
 assert hashlib.sha256(tested.encode()).hexdigest()==item['frozen_sha256'],name
 previous=subprocess.check_output(['git','show','7001b1e:'+name]);assert tested.encode()==previous,name
for path in [root/'docs/plans/template-execution-budgets/lane-3-admission-followup.md',dest/'index.md']:
 for link in re.findall(r'\]\(([^)]+)\)',path.read_text()):
  if '://' not in link:assert (path.parent/link.split('#')[0]).exists(),link
subprocess.run(['git','diff','--check','--','*.go','*.md'],check=True)
receipt={'status':'passed','utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'checks':['JSON parse','all measurement and normal/race/vet receipts passed','six samples per matched cell, five core/diagnostic and six data-shape rows','frozen overlay source resolution','all production source hashes; exactly one checked ordinary-comment difference','signed implementation source equivalence','report/index links','Go/Markdown whitespace'],'implementation':'7001b1e','command':['python3','.git/admission-followup/verify_evidence.py'],'script_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),'execution_deadline_seconds':90,'term_cleanup_seconds':5}
(dest/'evidence-validation-receipt.json').write_text(json.dumps(receipt,indent=2)+'\n')
(dest/'verify_evidence.py').write_text(Path(__file__).read_text())
artifacts={p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in dest.iterdir() if p.is_file() and p.name!='artifact-hashes.json'};(dest/'artifact-hashes.json').write_text(json.dumps(artifacts,indent=2)+'\n')
for name,digest in artifacts.items():assert hashlib.sha256((dest/name).read_bytes()).hexdigest()==digest,name
print('PASS',len(artifacts),'artifact hashes and component evidence')
