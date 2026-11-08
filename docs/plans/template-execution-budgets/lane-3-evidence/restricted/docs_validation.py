import ast,hashlib,json,re,subprocess,time
from pathlib import Path
out=Path('.git/lane3-evidence');start=time.time()
files=[Path('docs/restricted.md'),Path('README.md'),*map(Path,subprocess.check_output(['git','diff','--name-only','--','*.md']).decode().splitlines()),*Path('docs/plans/template-execution-budgets/lane-3-evidence/restricted').glob('*.md')]
manifest={str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
(out/'restricted-docs-inputs.json').write_text(json.dumps(manifest,indent=2)+'\n')
errors=[]
for p in files:
 for target in re.findall(r'\[[^\]]*\]\(([^)]+)\)',p.read_text()):
  if target.startswith(('https:','http:','#','mailto:')):continue
  target=target.split('#')[0]
  if target and not (p.parent/target).exists():errors.append(str(p)+': '+target)
for p in Path('docs/plans/template-execution-budgets/lane-3-evidence/restricted').glob('*.py'):ast.parse(p.read_text())
for head,base in [('contrib/bounded-compilation','contrib/pull-lexer'),('contrib/runtime-compilation','prereq/runtime-compilation')]:
 paths=subprocess.check_output(['git','diff','--name-only',base,head]).decode().splitlines()
 for p in paths:
  if p.startswith(('docs/plans/','docs/rfcs/')) or p in ['go.mod','go.sum','docs/index.md']:errors.append(head+': unexpected review path '+p)
 for p in subprocess.check_output(['git','ls-tree','-r','--name-only',head]).decode().splitlines():
  if p.endswith('.md'):
   source=subprocess.check_output(['git','show',head+':'+p]).decode()
   for target in re.findall(r'\[[^\]]*\]\(([^)]+)\)',source):
    if target.startswith(('http:','https:','#','mailto:')):continue
    target=target.split('#')[0]
    if not target or target.startswith('/') or '<' in target:continue
    resolved=Path(p).parent/target
    # Git accepts normalized forward-slash paths, not literal .. components.
    import posixpath
    normalized=posixpath.normpath(str(resolved))
    check=subprocess.run(['git','cat-file','-e',head+':'+normalized],capture_output=True)
    if check.returncode and ('rfcs/' in target or 'plans/' in target):errors.append(head+': removed internal target '+p+' -> '+target)
subprocess.run(['git','diff','--check'],check=True,timeout=15)
result={'command':'python3 .git/lane3-evidence/docs_validation.py','execution_deadline_seconds':180,'term_cleanup_seconds':5,'start':start,'end':time.time(),'exit_code':1 if errors else 0,'errors':errors,'manifest':'restricted-docs-inputs.json'}
(out/'restricted-docs-receipt.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result,indent=2));assert not errors
