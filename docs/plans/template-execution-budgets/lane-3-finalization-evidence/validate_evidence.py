import os,json,subprocess,hashlib,re,time
from pathlib import Path
root=Path.cwd();dest=root/'docs/plans/template-execution-budgets/lane-3-finalization-evidence';out=root/'.git/finalization';start=time.time();env=dict(os.environ,GIT_INDEX_FILE=str(out/'validation-index'))
variants=json.loads((dest/'variants.json').read_text());verified=[]
for name,row in variants.items():
 subprocess.run(['git','read-tree',row['base']],env=env,check=True,timeout=15)
 patch=(dest/row['patch']).read_bytes()
 if patch:subprocess.run(['git','apply','--cached','--whitespace=error'],env=env,input=patch,check=True,timeout=15)
 for filename,want in row['production_sha256'].items():
  actual=hashlib.sha256(subprocess.check_output(['git','show',':'+filename],env=env,timeout=15)).hexdigest();assert actual==want,(name,filename)
 verified.append(name)
paths=[root/'docs/plans/template-execution-budgets/lane-3-finalization.md',dest/'index.md']
for p in paths:
 for href in re.findall(r'\[[^\]]*\]\(([^)]+)\)',p.read_text()):
  if href.startswith(('http:','https:','#')):continue
  assert (p.parent/href.split('#')[0]).exists(),(p,href)
subprocess.run(['git','diff','--check'],check=True,timeout=15)
result={'command':'python3 .git/finalization/validate_evidence.py','exit_code':0,'start':start,'end':time.time(),'execution_deadline_seconds':180,'cleanup_seconds':5,'variant_patches_verified':verified,'docs_sha256':{str(p.relative_to(root)):hashlib.sha256(p.read_bytes()).hexdigest() for p in paths}}
(out/'evidence-validation-receipt.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result,indent=2))
