import hashlib,json,subprocess,sys
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization/followup-54';record=json.loads((art/'baseline.json').read_text());binary=Path(record['binary']);out=art/'analysis';out.mkdir(exist_ok=True);views=[]
source=str(art/'baseline-source')+':<integration-artifacts>/release-54b5614/candidate'
for stage in record['steps']:
 name=stage['name']
 for view,profile,flags in [('cpu-flat','cpu',['-top']),('cpu-cumulative','cpu',['-top','-cum']),('source','cpu',['-list','Template.*execute|Runtime.*recover|Runtime.*checkpoint|Runtime.*enter|checkedWriter.*Write']),('alloc-bytes','heap',['-top','-alloc_space']),('alloc-objects','heap',['-top','-alloc_objects']),('alloc-objects-cumulative','heap',['-top','-cum','-alloc_objects']),('inuse-bytes','heap',['-top','-inuse_space']),('inuse-objects','heap',['-top','-inuse_objects']),('inuse-objects-cumulative','heap',['-top','-cum','-inuse_objects']),('mutex','mutex',['-top']),('block','block',['-top'])]:
  profile=art/(name+'-'+profile+'.pprof');cmd=['go','tool','pprof','-nodecount=30','-source_path='+source,'-trim_path='+str(root)+':github.com/CloudyKit/jet/v6']+flags+[str(binary),str(profile)];r=subprocess.run(cmd,capture_output=True,text=True,timeout=20);f=out/(name+'-'+view+'.txt');f.write_text(r.stdout+r.stderr);views.append(dict(command=cmd,exit_code=r.returncode,deadline=20,output=str(f),profile_sha256=hashlib.sha256(profile.read_bytes()).hexdigest()));assert r.returncode==0,(name,view,r.stderr)
 (art/'inspection.json').write_text(json.dumps(views,indent=2)+'\n');print(name,flush=True)
print('15 isolated workflows, 165 frozen-source profile views')
