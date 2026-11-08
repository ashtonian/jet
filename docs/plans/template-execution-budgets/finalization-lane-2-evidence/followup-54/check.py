import hashlib,json,re,subprocess
from pathlib import Path
root=Path.cwd();base=root/'docs/plans/template-execution-budgets/finalization-lane-2-evidence';out=base/'followup-54'
for f in [root/'docs/plans/template-execution-budgets/finalization-lane-2.md',base/'index.md']:
 for link in re.findall(r'\]\(([^)]+)\)',f.read_text()):
  if '://' not in link and not link.startswith('#'):assert (f.parent/link.split('#')[0]).exists(),(f,link)
for f in out.rglob('*'):
 if not f.is_file():continue
 s=f.read_text();assert not any('/'+x in s for x in ['Users/','Volumes/','private/var/folders/']) and ' on '+'studio' not in s,f
 if f.suffix=='.json':
  r=json.loads(s)
  if isinstance(r,dict) and 'input_manifest' in r:assert (f.parent/r['input_manifest']).exists(),f
cache={}
for entry in json.loads((out/'source-patches.json').read_text()):
 name=entry['file']
 if name not in cache:cache[name]=subprocess.check_output(['git','show','54b5614:'+name],text=True).splitlines(keepends=True)
 original=cache[name];patch=(out/entry['patch']).read_text().splitlines(keepends=True);result=[];cursor=0;i=2
 while i<len(patch):
  m=re.match(r'^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@',patch[i]);assert m,(entry,patch[i]);start=int(m[1]);count=int(m[2]) if m[2] is not None else 1;pos=start if count==0 else start-1;assert pos>=cursor;result.extend(original[cursor:pos]);cursor=pos;i+=1
  while i<len(patch) and not patch[i].startswith('@@'):
   l=patch[i]
   if l.startswith('-'):assert original[cursor]==l[1:],(entry,l);cursor+=1
   elif l.startswith('+'):result.append(l[1:])
   else:assert False,(entry,l)
   i+=1
 result.extend(original[cursor:]);assert hashlib.sha256(''.join(result).encode()).hexdigest()==entry['sha256'],entry
manifest=json.loads((base/'raw-artifact-sha256.json').read_text());verified=0
for p,info in manifest.items():
 if not p.startswith('followup-54/'):continue
 f=root/'.git/finalization'/p;assert hashlib.sha256(f.read_bytes()).hexdigest()==info['sha256'],p;verified+=1
proof=json.loads((out/'source-proof.json').read_text());assert proof['code_sha']=='ec0221fb7ecd4f5416a289b8e019d60c65793872' and not proof['full_release_acceptance'];assert proof['cancellation']['cells']==120
subprocess.run(['git','diff','--exit-code',proof['code_sha'],'--','*.go','go.mod','go.sum','testData/'],check=True)
assert subprocess.check_output(['gofmt','-l','eval.go','exec.go','execution.go','execution_print.go','set.go','execution_test.go','restricted_test.go','execution_output_reentry_test.go'],text=True)==''
assert not (root/'zz_finalization_followup_counts_test.go').exists()
subprocess.run(['git','diff','--cached','--check'],check=True)
print('Evidence links/JSON/manifests/privacy, source-patch reconstruction, '+str(verified)+' raw hashes, formatting and exact-source reuse: PASS')
