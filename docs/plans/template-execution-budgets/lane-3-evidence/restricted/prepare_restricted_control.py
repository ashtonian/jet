"""Prepare a Git-source Go overlay without creating another checkout."""
import argparse,json,subprocess
from pathlib import Path
parser=argparse.ArgumentParser();parser.add_argument('mode',choices=['baseline','compiler']);args=parser.parse_args()
root=Path(subprocess.check_output(['git','rev-parse','--show-toplevel']).decode().strip())
artifacts=root/'.git/lane3-evidence';artifacts.mkdir(exist_ok=True)
rev='bda0fa8fe7e3f92b1580f4902ea38623cbe038e8' if args.mode=='baseline' else '484d70de3c8bfb6f9da06483637e14e276eff439'
out=artifacts/(args.mode+'-source');out.mkdir(exist_ok=True)
current=subprocess.check_output(['git','ls-files','--cached','--others','--exclude-standard','*.go']).decode().splitlines()
original=subprocess.check_output(['git','ls-tree','-r','--name-only',rev]).decode().splitlines()
replace={}
for name in sorted(set(current+original)):
 if '/' in name or not name.endswith('.go'):continue
 old=subprocess.run(['git','show',rev+':'+name],capture_output=True,timeout=15)
 if old.returncode==0:
  target=out/name;target.write_bytes(old.stdout);replace[str(root/name)]=str(target)
 else:replace[str(root/name)]=''
if args.mode=='baseline':
 fixture=root/'docs/plans/template-execution-budgets/lane-3-evidence/restricted/restricted-baseline-test.txt'
 replace[str(root/'restricted_test.go')]=str(fixture)
path=artifacts/(args.mode+'-overlay.json');path.write_text(json.dumps({'Replace':replace},indent=2)+'\n');print(path.relative_to(root))
