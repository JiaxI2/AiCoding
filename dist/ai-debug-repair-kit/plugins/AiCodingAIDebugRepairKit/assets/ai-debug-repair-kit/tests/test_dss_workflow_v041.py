import json, subprocess, sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def run(*args): return subprocess.run([sys.executable,'-m','ai_debug_repair.cli',*args], cwd=ROOT, text=True, capture_output=True, timeout=20)
def make_profile(tmp_path):
    p=tmp_path/'ti.json'; r=run('dss','profile-template','--profile',str(p),'--output','json'); assert r.returncode==0; return p
def test_connect_test_standard_envelope(tmp_path, monkeypatch):
    monkeypatch.setenv('PYTHONPATH', str(ROOT/'src')); p=make_profile(tmp_path); r=run('dss','connect-test','--profile',str(p),'--workspace',str(tmp_path),'--output','json'); assert r.returncode==0; j=json.loads(r.stdout);
    for k in ['schema_version','ok','code','message','data','safety','capability_status','evidence','warnings','side_effects','duration_ms','trace_id','session_id']: assert k in j
    assert Path(j['evidence']['session_dir']).exists()
def test_monitor_address_generates_session_and_summary(tmp_path, monkeypatch):
    monkeypatch.setenv('PYTHONPATH', str(ROOT/'src')); p=make_profile(tmp_path); r=run('dss','monitor-address','--profile',str(p),'--workspace',str(tmp_path),'--address','0xB4C0','--samples','10','--output','json'); assert r.returncode==0; j=json.loads(r.stdout); assert j['data']['samples']==10; assert 'changed' in j['data']; assert Path(j['evidence']['session_dir']).exists()
def test_monitor_symbol_generates_markdown(tmp_path, monkeypatch):
    monkeypatch.setenv('PYTHONPATH', str(ROOT/'src')); p=make_profile(tmp_path); out=tmp_path/'app.out'; out.write_text('fake'); r=run('dss','monitor-symbol','--profile',str(p),'--workspace',str(tmp_path),'--out',str(out),'--symbol','g_test','--samples','3','--output','md'); assert r.returncode==0; assert '# DSS Debug Report' in r.stdout
def test_find_changing_symbol_not_executed_is_verifiable(tmp_path, monkeypatch):
    monkeypatch.setenv('PYTHONPATH', str(ROOT/'src')); p=make_profile(tmp_path); out=tmp_path/'app.out'; out.write_text('fake'); r=run('dss','find-changing-symbol','--profile',str(p),'--workspace',str(tmp_path),'--out',str(out),'--candidates','5','--output','json'); assert r.returncode==0; j=json.loads(r.stdout); assert j['data']['not_found'] is True; assert Path(j['evidence']['session_dir']).exists()
