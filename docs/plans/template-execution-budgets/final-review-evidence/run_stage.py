import sys,pathlib,subprocess,os,json,time,hashlib
root=pathlib.Path(__file__).parent
lane,label,tree,*cmd=sys.argv[1:]
cwd=root/tree if tree!="repo" else root.parents[2]
log=root/(label+".log")
env=dict(os.environ,GATE_LOCK_WAIT="1",GATE_LOCK_COMMAND_TIMEOUT=os.environ.get("REVIEW_TIMEOUT","180"),GATE_LOCK_COMMAND_KILL_AFTER="10")
args=["gate-lock",lane,"jet-final-"+label,"--",*cmd]
manifest={str(p.relative_to(cwd)):hashlib.sha256(p.read_bytes()).hexdigest() for p in cwd.rglob("*") if p.is_file() and (p.suffix in (".go",".json") or p.name in ("go.mod","go.sum")) and ".git" not in p.relative_to(cwd).parts}
receipt={"command":args,"cwd":str(cwd),"inputs":manifest,"start":time.time(),"toolchain":subprocess.check_output(["go","version"],text=True).strip(),"environment":{k:env.get(k) for k in ("GATE_LOCK_COMMAND_TIMEOUT","GATE_LOCK_COMMAND_KILL_AFTER","GOMAXPROCS","GOFLAGS")}}
with log.open("w") as f:p=subprocess.run(args,cwd=cwd,env=env,stdout=f,stderr=subprocess.STDOUT)
receipt.update(exit=p.returncode,end=time.time())
(root/(label+".json")).write_text(json.dumps(receipt,indent=2))
print(log.read_text()[-20000:])
print("RECEIPT",root/(label+".json"))
sys.exit(p.returncode)
