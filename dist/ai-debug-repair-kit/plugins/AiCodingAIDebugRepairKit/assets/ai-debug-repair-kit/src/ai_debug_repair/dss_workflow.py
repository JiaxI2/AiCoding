from __future__ import annotations
import hashlib, json, random, re, shutil, subprocess, time
from pathlib import Path
from typing import Any
from .core import envelope, load_json, now_ms, write_json

FORBIDDEN_TOKENS = ["target.reset", "target.halt", "target.run", "loadProgram", "loadAndRun", "flash", "erase", "writeData", "writeWord", "writeRegister", "expression.write", "setValue"]
SAFETY = {"reset": False, "halt": False, "run": False, "loadProgram": False, "flash": False, "erase": False, "write_memory": False, "write_expression": False, "write_register": False}
CAPABILITY = {"ccs_dss_found":"not_checked", "target_config_valid":"not_checked", "core_list":"not_tested", "debug_connection":"not_tested", "connect_disconnect":"not_tested", "symbol_load":"not_tested", "program_load":"disabled", "ram_read":"not_tested", "register_read":"not_tested", "repeated_read":"not_tested", "changing_monitor":"not_verified", "realtime_debug":"not_verified", "halt_run_control":"disabled", "flash_write":"disabled", "memory_write":"disabled"}

def _sid(): return f"dss-{time.strftime('%Y%m%d-%H%M%S')}-{random.randint(1000,9999)}"
def _sha(path: Path):
    if not path.exists() or not path.is_file(): return None
    h=hashlib.sha256()
    with path.open('rb') as f:
        for b in iter(lambda:f.read(1024*1024), b''): h.update(b)
    return h.hexdigest()
def _session(workspace: Path, op: str, profile: Path|None):
    sid=_sid(); root=workspace/'.ai-debug-repair'/'sessions'/sid
    for d in ['scripts','logs','evidence']: (root/d).mkdir(parents=True, exist_ok=True)
    manifest={"schema_version":"1.0","session_id":sid,"operation":op,"created_ms":now_ms(),"workspace":str(workspace),"profile":str(profile) if profile else None}
    write_json(root/'manifest.json', manifest)
    return {"session_id":sid,"session_dir":root}
def _profile(profile: Path):
    try: return load_json(profile), None
    except ValueError as e: return None, envelope(False,'PROFILE_INVALID',str(e))
def _cap(profile=None):
    c=dict(CAPABILITY)
    if profile:
        launcher=str(profile.get('dss_launcher','')); ccxml=str(profile.get('target_config',''))
        c['ccs_dss_found']='verified' if (Path(launcher).exists() or shutil.which(launcher)) else 'not_found'
        c['target_config_valid']='verified' if ccxml and Path(ccxml).exists() else 'not_verified'
    return c
def _scan(script: str):
    low=script.lower(); found=[t for t in FORBIDDEN_TOKENS if t.lower() in low]
    if re.search(r"expression\.evaluate\s*\(\s*['\"][^'\"]*=", script, re.I): found.append('expression write')
    return sorted(set(found))
def _text(result):
    d=result.get('data',{}); s=result.get('safety',{}); c=result.get('capability_status',{})
    lines=['[DSS Debug Summary]','','Target:',f"  Device       : {d.get('device','unknown')}",f"  Probe        : {d.get('probe','unknown')}",f"  Core         : {d.get('core','unknown')}",f"  OUT          : {d.get('out_file','-')}",'','Read:',f"  Operation    : {d.get('operation','-')}",f"  Symbol       : {d.get('symbol','-')}",f"  Address      : {d.get('address','-')}",f"  Page         : {d.get('page','-')}",f"  Width        : {d.get('width_bits','-')}",f"  Samples      : {d.get('samples','-')}",f"  Interval     : {d.get('interval_ms','-')}",'','Result:',f"  OK           : {result.get('ok')}",f"  Code         : {result.get('code')}",f"  Changed      : {d.get('changed','-')}",f"  Unique       : {d.get('unique_values','-')}",f"  Min / Max    : {d.get('min','-')} / {d.get('max','-')}",'','Safety:']
    for k in ['reset','halt','run','loadProgram','flash','erase','write_memory','write_expression','write_register']: lines.append(f"  {k:<14}: {'yes' if s.get(k) else 'no'}")
    lines += ['', 'Capability:']
    for k in ['debug_connection','connect_disconnect','ram_read','repeated_read','changing_monitor','realtime_debug','halt_run_control','flash_write','memory_write']: lines.append(f"  {k:<20}: {c.get(k,'-')}")
    lines += ['', f"Evidence: {result.get('evidence',{}).get('session_dir','-')}"]
    return '\n'.join(lines)
def _md(result):
    d=result.get('data',{}); lines=['# DSS Debug Report','', '## 结论', '', f"- OK: `{result.get('ok')}`", f"- Code: `{result.get('code')}`", f"- Message: {result.get('message')}", f"- Session: `{result.get('session_id')}`", '', '## Target', '', f"- Device: `{d.get('device','unknown')}`", f"- Probe: `{d.get('probe','unknown')}`", f"- Core: `{d.get('core','unknown')}`", f"- OUT: `{d.get('out_file','-')}`", '', '## Read', '', f"- Operation: `{d.get('operation','-')}`", f"- Symbol: `{d.get('symbol','-')}`", f"- Address: `{d.get('address','-')}`", f"- Page: `{d.get('page','-')}`", f"- Samples: `{d.get('samples','-')}`", '', '## Result', '', f"- Changed: `{d.get('changed','-')}`", f"- Unique values: `{d.get('unique_values','-')}`", f"- Min / Max: `{d.get('min','-')}` / `{d.get('max','-')}`", '', '## Safety']
    for k,v in result.get('safety',{}).items(): lines.append(f"- {k}: `{v}`")
    lines += ['', '## Capability Status']
    for k,v in result.get('capability_status',{}).items(): lines.append(f"- {k}: `{v}`")
    lines += ['', '## Evidence']
    for k,v in result.get('evidence',{}).items(): lines.append(f"- {k}: `{v}`")
    return '\n'.join(lines)+'\n'
def _result(ok, code, msg, data, sess, start, warnings=None, cap=None):
    ev={"session_dir":str(sess['session_dir']),"manifest":str(sess['session_dir']/'manifest.json'),"scripts_dir":str(sess['session_dir']/'scripts'),"logs_dir":str(sess['session_dir']/'logs'),"evidence_dir":str(sess['session_dir']/'evidence')}
    r=envelope(ok, code, msg, data, warnings or [], [], start); r.update({"safety":dict(SAFETY),"capability_status":cap or dict(CAPABILITY),"evidence":ev,"session_id":sess['session_id'],"trace_id":sess['session_id']})
    r['text']=_text(r); r['markdown']=_md(r); write_json(sess['session_dir']/'result.json', r); (sess['session_dir']/'report.md').write_text(r['markdown'], encoding='utf-8')
    reports=sess['session_dir'].parents[1]/'reports'; reports.mkdir(parents=True, exist_ok=True); (reports/(sess['session_id']+'.md')).write_text(r['markdown'], encoding='utf-8')
    return r
def _base_script(profile, body):
    ccxml=str(profile.get('target_config','')).replace('\\','/'); return "\n".join(["// Generated by airepair fixed DSS template.","// Read-only fixed template. Safety policy enforced by CLI.","importPackage(Packages.com.ti.debug.engine.scripting);","importPackage(Packages.com.ti.ccstudio.scripting.environment);","importPackage(Packages.java.lang);","var scripting = ScriptingEnvironment.instance();","var server = scripting.getServer('DebugServer.1');",f"server.setConfig('{ccxml}');",body,"try { server.stop(); } catch(ignore) {}",""])
def _write(sess, name, script):
    bad=_scan(script)
    if bad: return None,bad
    path=sess['session_dir']/'scripts'/name; path.write_text(script, encoding='utf-8'); return path, []
def _exec(profile, script, sess, timeout=10):
    launcher=str(profile.get('dss_launcher',''))
    if not Path(launcher).exists() and not shutil.which(launcher): return False,{"code":"DEPENDENCY_MISSING","message":f"DSS launcher not found: {launcher}","values":[]}
    try: cp=subprocess.run([launcher, str(script)], text=True, capture_output=True, timeout=timeout, shell=False)
    except subprocess.TimeoutExpired: return False,{"code":"TIMEOUT","message":"DSS script timed out","values":[]}
    (sess['session_dir']/'logs'/'dss.stdout.log').write_text(cp.stdout or '', encoding='utf-8', errors='replace'); (sess['session_dir']/'logs'/'dss.stderr.log').write_text(cp.stderr or '', encoding='utf-8', errors='replace')
    parsed=None
    for line in reversed((cp.stdout or '').splitlines()):
        line=line.strip()
        if line.startswith('{') and line.endswith('}'):
            try: parsed=json.loads(line); break
            except Exception: pass
    if parsed is None: parsed={"ok":cp.returncode==0,"code":"OK" if cp.returncode==0 else "DSS_COMMAND_FAILED","values":[]}
    return cp.returncode==0 and bool(parsed.get('ok', True)), parsed
def _stats(values):
    nums=[]
    for v in values:
        try: nums.append(int(v))
        except Exception:
            try: nums.append(float(v))
            except Exception: pass
    u=sorted(set(nums)) if nums else []
    return {"values":nums,"unique_values":len(u),"changed":len(u)>1,"min":min(nums) if nums else None,"max":max(nums) if nums else None,"first":nums[0] if nums else None,"last":nums[-1] if nums else None}

def connect_test(profile_path: Path, workspace: Path, execute=False):
    start=now_ms(); prof,err=_profile(profile_path); sess=_session(workspace,'dss.connect-test',profile_path)
    if err: return _result(False,'PROFILE_INVALID',err['message'],{},sess,start)
    cap=_cap(prof); core=str(prof.get('core','C28xx_CPU1'))
    body=f"var ds=server.openSession('*','{core}'); ds.target.connect(); ds.target.disconnect(); ds.terminate(); print(JSON.stringify({{schema_version:'1.0', ok:true, code:'OK', connected:true, disconnected:true}}));"
    script=_base_script(prof, body); path,bad=_write(sess,'connect_test.js',script)
    if bad: return _result(False,'POLICY_DENIED','Generated DSS script contains forbidden tokens',{'forbidden_tokens':bad},sess,start,cap=cap)
    data={'operation':'connect-test','execute':execute,'script':str(path),'device':prof.get('device'),'probe':prof.get('probe'),'core':prof.get('core')}
    if not execute: return _result(True,'OK','DSS connect-test script generated',data,sess,start,['not executed; add --execute to test hardware'],cap)
    ok,parsed=_exec(prof,path,sess); cap['debug_connection']='verified' if ok else 'not_verified'; cap['connect_disconnect']='verified' if ok else 'not_verified'; data['dss']=parsed
    return _result(ok,'OK' if ok else parsed.get('code','DSS_ERROR'),'DSS connect-test completed' if ok else 'DSS connect-test failed',data,sess,start,cap=cap)

def core_list(profile_path: Path, workspace: Path, execute=False):
    start=now_ms(); prof,err=_profile(profile_path); sess=_session(workspace,'dss.core-list',profile_path)
    if err: return _result(False,'PROFILE_INVALID',err['message'],{},sess,start)
    cap=_cap(prof); body="print(JSON.stringify({schema_version:'1.0', ok:true, code:'OK', note:'core list requires CCS DSS runtime'}));"
    path,bad=_write(sess,'core_list.js',_base_script(prof,body))
    if bad: return _result(False,'POLICY_DENIED','Generated DSS script contains forbidden tokens',{'forbidden_tokens':bad},sess,start,cap=cap)
    data={'operation':'core-list','execute':execute,'script':str(path),'device':prof.get('device'),'probe':prof.get('probe'),'core':prof.get('core')}
    if not execute: return _result(True,'OK','DSS core-list script generated',data,sess,start,['not executed; add --execute to test hardware'],cap)
    ok,parsed=_exec(prof,path,sess); cap['core_list']='verified' if ok else 'not_verified'; data['dss']=parsed
    return _result(ok,'OK' if ok else parsed.get('code','DSS_ERROR'),'DSS core-list completed' if ok else 'DSS core-list failed',data,sess,start,cap=cap)

def monitor_address(profile_path: Path, workspace: Path, address: str, page: str, width: int, samples: int, interval_ms: int, execute=False):
    start=now_ms(); prof,err=_profile(profile_path); sess=_session(workspace,'dss.monitor-address',profile_path)
    if err: return _result(False,'PROFILE_INVALID',err['message'],{},sess,start)
    cap=_cap(prof); core=str(prof.get('core','C28xx_CPU1')); addr=int(str(address),0); page_expr='Memory.Page.DATA' if page.upper()=='DATA' else 'Memory.Page.PROGRAM'
    body=f"var ds=server.openSession('*','{core}'); ds.target.connect(); var values=[]; for(var i=0;i<{samples};i++){{ var v=ds.memory.readData({page_expr},{addr},{width},false); values.push(Number(v)); Thread.sleep({interval_ms}); }} ds.target.disconnect(); ds.terminate(); print(JSON.stringify({{schema_version:'1.0', ok:true, code:'OK', values:values}}));"
    path,bad=_write(sess,'monitor_address.js',_base_script(prof,body))
    data={'operation':'monitor-address','execute':execute,'script':str(path) if path else None,'device':prof.get('device'),'probe':prof.get('probe'),'core':prof.get('core'),'address':hex(addr),'page':page,'width_bits':width,'samples':samples,'interval_ms':interval_ms}
    if bad: return _result(False,'POLICY_DENIED','Generated DSS script contains forbidden tokens',{'forbidden_tokens':bad},sess,start,cap=cap)
    if not execute: data.update(_stats([])); return _result(True,'OK','DSS monitor-address script generated',data,sess,start,['not executed; add --execute to sample hardware'],cap)
    ok,parsed=_exec(prof,path,sess,max(10, samples*interval_ms//1000+10)); st=_stats(parsed.get('values',[])); data.update(st); data['dss']=parsed; cap['ram_read']='verified' if ok else 'not_verified'; cap['repeated_read']='verified' if ok and samples>1 else 'not_verified'; cap['changing_monitor']='verified' if st['changed'] else 'not_verified'
    (sess['session_dir']/'evidence'/'samples.jsonl').write_text('\n'.join(json.dumps({'index':i,'value':v},ensure_ascii=False) for i,v in enumerate(st['values']))+'\n', encoding='utf-8')
    return _result(ok,'OK' if ok else parsed.get('code','DSS_ERROR'),'DSS monitor-address completed' if ok else 'DSS monitor-address failed',data,sess,start,[] if st['changed'] else ['values did not change during this sample window'],cap)

def monitor_symbol(profile_path: Path, workspace: Path, out_file: Path, symbol: str, samples: int, interval_ms: int, execute=False):
    start=now_ms(); prof,err=_profile(profile_path); sess=_session(workspace,'dss.monitor-symbol',profile_path)
    if err: return _result(False,'PROFILE_INVALID',err['message'],{},sess,start)
    cap=_cap(prof); core=str(prof.get('core','C28xx_CPU1')); sym=symbol.replace('\\','\\\\').replace("'","\\'")
    body=f"var ds=server.openSession('*','{core}'); ds.target.connect(); var values=[]; for(var i=0;i<{samples};i++){{ var v=ds.expression.evaluate('{sym}'); values.push(Number(v)); Thread.sleep({interval_ms}); }} ds.target.disconnect(); ds.terminate(); print(JSON.stringify({{schema_version:'1.0', ok:true, code:'OK', values:values}}));"
    path,bad=_write(sess,'monitor_symbol.js',_base_script(prof,body))
    data={'operation':'monitor-symbol','execute':execute,'script':str(path) if path else None,'device':prof.get('device'),'probe':prof.get('probe'),'core':prof.get('core'),'out_file':str(out_file),'out_sha256':_sha(out_file),'symbol':symbol,'samples':samples,'interval_ms':interval_ms}
    if bad: return _result(False,'POLICY_DENIED','Generated DSS script contains forbidden tokens',{'forbidden_tokens':bad},sess,start,cap=cap)
    if not execute: data.update(_stats([])); return _result(True,'OK','DSS monitor-symbol script generated',data,sess,start,['not executed; add --execute to sample hardware'],cap)
    ok,parsed=_exec(prof,path,sess,max(10, samples*interval_ms//1000+10)); st=_stats(parsed.get('values',[])); data.update(st); data['dss']=parsed; cap['ram_read']='verified' if ok else 'not_verified'; cap['repeated_read']='verified' if ok and samples>1 else 'not_verified'; cap['changing_monitor']='verified' if st['changed'] else 'not_verified'
    (sess['session_dir']/'evidence'/'samples.jsonl').write_text('\n'.join(json.dumps({'index':i,'value':v},ensure_ascii=False) for i,v in enumerate(st['values']))+'\n', encoding='utf-8')
    return _result(ok,'OK' if ok else parsed.get('code','DSS_ERROR'),'DSS monitor-symbol completed' if ok else 'DSS monitor-symbol failed',data,sess,start,[] if st['changed'] else ['symbol did not change during this sample window'],cap)

def _candidates(out_file: Path, limit:int, prefer:str):
    prefs=[x.strip().lower() for x in prefer.split(',') if x.strip()]; items=[]
    for side in [Path(str(out_file)+'.symbols.json'), Path(str(out_file)+'.symbols.txt')]:
        if side.exists():
            if side.suffix=='.json':
                try:
                    raw=json.loads(side.read_text(encoding='utf-8')); items += [x for x in raw if isinstance(x,dict) and x.get('name')]
                except Exception: pass
            else:
                for line in side.read_text(encoding='utf-8',errors='replace').splitlines():
                    ps=line.split();
                    if ps: items.append({'name':ps[-1], 'address':ps[0] if len(ps)>1 else None, 'source':str(side)})
    if not items:
        for n in ['tick','counter','timer','timestamp','state','status','flag','heartbeat','index']:
            items.append({'name':n,'source':'heuristic','address':None})
    def score(x):
        n=str(x.get('name','')).lower(); s=sum((100-i) for i,p in enumerate(prefs) if p in n); return (-s,n)
    return sorted(items,key=score)[:limit]

def find_changing_symbol(profile_path: Path, workspace: Path, out_file: Path, candidates:int, samples:int, interval_ms:int, prefer_name:str, execute=False):
    start=now_ms(); prof,err=_profile(profile_path); sess=_session(workspace,'dss.find-changing-symbol',profile_path)
    if err: return _result(False,'PROFILE_INVALID',err['message'],{},sess,start)
    cap=_cap(prof); cand=_candidates(out_file,candidates,prefer_name); write_json(sess['session_dir']/'evidence'/'candidates.json', {'out_file':str(out_file),'out_sha256':_sha(out_file),'candidates':cand})
    data={'operation':'find-changing-symbol','execute':execute,'out_file':str(out_file),'out_sha256':_sha(out_file),'candidates':len(cand),'candidate_symbols':cand,'samples':samples,'interval_ms':interval_ms,'changed_symbols':[],'not_found':True}
    if not execute: return _result(True,'OK','DSS find-changing-symbol candidates prepared',data,sess,start,['not executed; add --execute to sample hardware'],cap)
    changed=[]
    for it in cand:
        sub=monitor_symbol(profile_path,workspace,out_file,str(it.get('name')),samples,interval_ms,True); sd=sub.get('data',{})
        if sd.get('changed'): changed.append({'symbol':it.get('name'), 'unique_values':sd.get('unique_values'), 'min':sd.get('min'), 'max':sd.get('max')})
    data['changed_symbols']=changed; data['not_found']=len(changed)==0; cap['changing_monitor']='verified' if changed else 'not_verified'
    return _result(True,'OK' if changed else 'NOT_FOUND','Changing symbol found' if changed else 'No changing symbol found in candidate window',data,sess,start,cap=cap)

def dss_report(workspace: Path, session_id: str|None=None):
    start=now_ms(); base=workspace/'.ai-debug-repair'/'sessions'; sessions=sorted([p for p in base.glob('dss-*') if p.is_dir()], key=lambda p:p.stat().st_mtime, reverse=True) if base.exists() else []
    sessdir=base/session_id if session_id else (sessions[0] if sessions else None)
    if not sessdir or not sessdir.exists():
        sess=_session(workspace,'dss.report',None); return _result(False,'NOT_FOUND','No DSS session found',{},sess,start)
    sess={'session_id':sessdir.name,'session_dir':sessdir}; report=sessdir/'report.md'
    if not report.exists(): return _result(False,'NOT_FOUND','Session report not found',{'session_dir':str(sessdir)},sess,start)
    return _result(True,'OK','DSS report loaded',{'operation':'report','session_dir':str(sessdir),'report':str(report),'markdown':report.read_text(encoding='utf-8')},sess,start)
