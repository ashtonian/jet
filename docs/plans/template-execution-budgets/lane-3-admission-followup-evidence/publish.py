import difflib,json,hashlib,re,subprocess
from pathlib import Path
root=Path.cwd();raw=root/'.git/admission-followup';dest=root/'docs/plans/template-execution-budgets/lane-3-admission-followup-evidence';dest.mkdir(exist_ok=True)
peer='<FROZEN54>'
def sanitize(s):
 return s.replace(peer,'<FROZEN54>').replace(str(root),'<CHECKOUT>').replace('<HOME>','<HOME>').replace(' on <HOST>,',' on <HOST>,')
rawhash={}
for f in raw.iterdir():
 if f.is_file() and f.suffix in ['.log','.json','.py','.txt','.pprof','.test']:
  rawhash[f.name]={'sha256':hashlib.sha256(f.read_bytes()).hexdigest(),'bytes':f.stat().st_size}
  if f.suffix in ['.log','.json','.py','.txt'] and f.name not in ['split-input.go'] and 'exec.go' not in f.name:
   (dest/f.name).write_text(sanitize(f.read_text()))
for label,old,new,path in [('scalar-container','base-frozen/input.go.txt','split-frozen/input.go.txt','input.go'),('direct-map','split-frozen/input.go.txt','direct-frozen/input.go.txt','input.go'),('default-policy','base-frozen/exec.go.txt','defaults-exec.go.txt','exec.go')]:
 delta=''.join(difflib.unified_diff((raw/old).read_text().splitlines(True),(raw/new).read_text().splitlines(True),fromfile='a/'+path,tofile='b/'+path));(dest/(label+'.patch')).write_text(delta)
patch=subprocess.check_output(['git','diff','54b561484e1465edea963ed04579bcd24a7dbf76','7001b1e','--','input.go','input_test.go','exec.go']).decode();(dest/'implementation.patch').write_text(patch)
inputs=json.loads((raw/'inputs-with-defaults.json').read_text());mapping=json.loads(Path(inputs['variants']['defaults']['overlay']).read_text())['Replace'];proof={}
for p in root.glob('*.go'):
 if p.name.endswith('_test.go'):continue
 key=str(p);expected=inputs['variants']['defaults']['hashes'][key];current=hashlib.sha256(p.read_bytes()).hexdigest();normalized=p.read_text().replace('// Supported unnamed types cannot add pointer methods; structs are denied below.', '// An unnamed or predeclared type cannot acquire pointer-receiver methods.');assert hashlib.sha256(normalized.encode()).hexdigest()==expected,p
 proof[p.name]={'sha256':current,'frozen_sha256':expected,'ordinary_comment_only_difference':current!=expected,'frozen':sanitize(mapping[key])}
(dest/'committed-production-proof.json').write_text(json.dumps({'implementation':'7001b1e','production_files':proof,'test_difference':'Only the later deterministic cancellation test was added after the benchmark sources were frozen; validation-inputs.json records its exact final source. Executable production source matches every frozen selected file; input.go has one ordinary-comment clarification explicitly checked by exact normalization.'},indent=2)+'\n')
(dest/'raw-artifact-hashes.json').write_text(json.dumps(rawhash,indent=2)+'\n')
index='''# Admission follow-up evidence

Base is exact combined `54b561484e1465edea963ed04579bcd24a7dbf76`; implementation is `7001b1e`. The [report](../lane-3-admission-followup.md) interprets these artifacts and the remaining gates.

- `inputs-with-defaults.json` records frozen baseline/split/direct/default-policy sources and toolchain/configuration. `committed-production-proof.json` proves executable selected production source matches the signed implementation, recording one ordinary-comment clarification separately. Three isolated patches and the complete implementation patch preserve replayable deltas; `strategy_benchmark.go.txt` is the same-package supplementary harness.
- `measurement-receipts.json` records three 200 ms diagnostic screens. `finish-receipts.json` records six interleaved one-second pairs and separate instrumented profile commands; `paired-analysis.json` contains all paired medians, counts and ratios. Tiny normal/core data and deep/shared/large shapes stay separate.
- `baseline-profile-receipts.json` and preliminary baseline views use the frozen combined binary. One source report with no sampled input allocation stopped the first analysis attempt; its nonzero result is preserved. Resume reuses successful measurements. No sampled attribution means no sample, never zero cost.
- `selected-*` views cover CPU flat/cumulative plus frozen source/call context, allocation/live bytes and objects, mutex/block and process RSS. Concurrent selected profiles contain the interpolation fixture with shared and independent Sets at 12 workers; the existing harness excludes `data_unused_256` from concurrency, so that alternative selector matches no additional case. Data-heavy concurrent performance remains a combined Lane 4 gate. Contexts are independently created per worker even when the Set is shared. No context-lock inference is made.
- `validation-inputs.json`, `validation-receipts.json` and validation logs record the final normal/race/vet batch, including deterministic cancellation controls. Queue elapsed time is distinct from bounded execution. Gate logs/observed outer receipts preserve admission/release outcomes.

Absolute source/host paths are sanitized to `<CHECKOUT>`, `<FROZEN54>`, `<HOME>` and `<HOST>`. Exact original artifacts, binaries and raw pprof files remain in this checkout's `.git/admission-followup`; `raw-artifact-hashes.json` identifies them without committing build binaries/profiles. Replay scripts require resolving those placeholders. Profile source paths refer to immutable overlay snapshots, not current files. Do not use these component measurements as the full fixed-15-fixture acceptance result.
'''
(dest/'index.md').write_text(index)
artifacts={f.name:hashlib.sha256(f.read_bytes()).hexdigest() for f in dest.iterdir() if f.is_file() and f.name!='artifact-hashes.json'};(dest/'artifact-hashes.json').write_text(json.dumps(artifacts,indent=2)+'\n');print('published',len(artifacts),'artifacts',sum(f.stat().st_size for f in dest.iterdir()),'bytes')
