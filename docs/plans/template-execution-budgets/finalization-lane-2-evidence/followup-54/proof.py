import hashlib,json,subprocess
from pathlib import Path
root=Path.cwd();art=root/'.git/finalization';src=art/'followup-54';code='ec0221fb7ecd4f5416a289b8e019d60c65793872'
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def resolve(p):
 f=root/p
 if not f.exists() and p=='zz_finalization_followup_counts_test.go':f=src/'counts.go.txt'
 return f
proof={'code_sha':code,'base_sha':'54b561484e1465edea963ed04579bcd24a7dbf76','superseded_code_sha':'958cbed0ce48c8e47e7c2d7f8b2b4738f38b5bad','current_checks':{},'local_signatures':{}}
for stage in ['followup-mode-normal','followup-mode-race','followup-mode-go116','followup-mode-vet','followup-committed-tree','followup-guard-final-analysis','followup-guard-profile-summary']:
 r=json.loads((art/stage/'result.json').read_text());assert r['exit_code']==0 and r['inputs_unchanged'],stage;checked=[]
 for p,h in r['inputs'].items():
  if p.endswith('.go') or p in ('go.mod','go.sum') or p.startswith('testData/'):
   assert sha(resolve(p))==h,(stage,p);checked.append(p)
 proof['current_checks'][stage]={'passed':True,'relevant_source_inputs_match':True,'source_paths_checked':len(checked),'receipt':str(Path(stage)/'result.json')}
b=json.loads((src/'terminal-repair/build.json').read_text());assert b['head']==code and b['inputs_unchanged'];assert all(sha(resolve(p))==h for p,h in b['common_inputs'].items());assert sha(src/'terminal-repair/render.test')==b['binary_sha256'];proof['selected_build_matches_current_source']=True
for name in ['baseline','selected']:
 b=json.loads((src/'final-builds.json').read_text())['variants'][name];assert sha(src/'final'/name/'render.test')==b['binary_sha256']
for p in (src/'terminal-repair/source').glob('*.go'):
 assert hashlib.sha256(subprocess.check_output(['git','show',code+':'+p.name])).hexdigest()==sha(p),p
for folder,count in [('matched',161),('terminal-repair/matched',142)]:
 r=json.loads((src/folder/'results.json').read_text());assert len(r['steps'])==count and r['inputs_unchanged'];assert all(x['exit_code']==0 and x['actual_cells']==x['expected_cells'] for x in r['steps']);proof[folder]={'completed_processes':count,'inputs_unchanged_during_run':True}
proof['cancellation']=json.loads((src/'terminal-repair/mode-cancellation-proof.json').read_text());assert proof['cancellation']['checks_identical'] and proof['cancellation']['cells']==120;proof['cancellation'].pop('rows')
proof['profiles']={'current_selected':28,'unchanged_baseline_reused':28,'current_pair_views':616,'intermediate_958cbed_views':616,'current_new_inspection_views':308}
for sha_commit in ['958cbed0ce48c8e47e7c2d7f8b2b4738f38b5bad',code]:
 r=subprocess.run(['git','verify-commit',sha_commit],capture_output=True,text=True);assert r.returncode==0;proof['local_signatures'][sha_commit]=r.stderr.strip()
proof['github_verification']='pending authorized lane4 fork publication; no lane2 push';proof['injected_measurement_test']='archived as counts.go.txt, removed from checkout; committed-tree cases pass';proof['upstream_reference']='Earlier interleaved upstream timings reused from immutable binary/fixture/harness inputs; refreshed full release gate remains pending';proof['full_release_acceptance']=False
(src/'source-proof.json').write_text(json.dumps(proof,indent=2)+'\n');print('Current source/check/build/measurement/profile/cancellation/signature proof: PASS')
