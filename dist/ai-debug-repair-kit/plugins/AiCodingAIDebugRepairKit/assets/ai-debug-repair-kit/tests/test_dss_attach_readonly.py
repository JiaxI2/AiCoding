import json
import os
import subprocess
import sys
from pathlib import Path


def run_cli(*args, cwd=None):
    root = Path(__file__).resolve().parents[1]
    env = os.environ.copy()
    env["PYTHONPATH"] = str(root / "src") + os.pathsep + env.get("PYTHONPATH", "")
    return subprocess.run([sys.executable, "-m", "ai_debug_repair.cli", *args], cwd=cwd or root, text=True, capture_output=True, env=env, timeout=20)


def write_profile(tmp_path):
    ccxml = tmp_path / "target.ccxml"
    ccxml.write_text("<configurations><configuration></configuration></configurations>", encoding="utf-8")
    profile = tmp_path / "attach.json"
    profile.write_text(json.dumps({
        "schema_version": "1.0",
        "target_config": str(ccxml),
        "dss_launcher": "dss.bat",
        "device": "TMS320F28388D",
        "probe": "XDS100v3",
        "core": "CPU1",
        "allowed_expressions": ["g_test", "CpuTimer0Regs.TIM.all"],
    }), encoding="utf-8")
    return profile


def script_from(payload):
    return Path(payload["data"]["script"]).read_text(encoding="utf-8")


def uncommented(text):
    return "\n".join(line for line in text.splitlines() if not line.strip().startswith("//"))


def test_attach_readonly_core_list_dry_run_generates_fixed_template(tmp_path):
    profile = write_profile(tmp_path)
    result = run_cli("dss", "attach-readonly", "core-list", "--profile", str(profile), "--workspace", str(tmp_path), "--output", "json")
    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    assert payload["ok"] is True
    text = uncommented(script_from(payload))
    assert "server.getListOfCPUs()" in text
    assert "target.connect" not in text
    for forbidden in ["target.reset", "target.halt", "target.run", "loadProgram", "flash", "erase", "writeData", "writeRegister"]:
        assert forbidden.lower() not in text.lower()


def test_attach_readonly_connect_test_uses_resolved_core_and_no_invasive_calls(tmp_path):
    profile = write_profile(tmp_path)
    result = run_cli("dss", "attach-readonly", "connect-test", "--profile", str(profile), "--workspace", str(tmp_path), "--output", "json")
    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    text = uncommented(script_from(payload))
    assert "server.getListOfCPUs()" in text
    assert "server.openSession(corePath)" in text
    assert "ds.target.connect()" in text
    assert "ds.target.disconnect()" in text
    for forbidden in ["target.reset", "target.halt", "target.run", "loadProgram", "flash", "erase", "writeData", "writeRegister"]:
        assert forbidden.lower() not in text.lower()


def test_attach_readonly_read_expression_requires_allowlist(tmp_path):
    profile = write_profile(tmp_path)
    result = run_cli("dss", "attach-readonly", "read-expression", "--profile", str(profile), "--workspace", str(tmp_path), "--expression", "not_allowed", "--output", "json")
    assert result.returncode != 0
    payload = json.loads(result.stdout)
    assert payload["code"] == "POLICY_DENIED"


def test_attach_readonly_monitor_symbol_dry_run_uses_expression_read_only(tmp_path):
    profile = write_profile(tmp_path)
    out = tmp_path / "app.out"
    out.write_text("fake", encoding="utf-8")
    result = run_cli("dss", "attach-readonly", "monitor-symbol", "--profile", str(profile), "--workspace", str(tmp_path), "--out", str(out), "--symbol", "g_test", "--samples", "3", "--interval-ms", "10", "--output", "json")
    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    text = uncommented(script_from(payload))
    assert "ds.symbol.load" in text
    assert "ds.expression.evaluateToString('g_test')" in text
    assert "Thread.sleep(10)" in text
    for forbidden in ["target.reset", "target.halt", "target.run", "loadProgram", "flash", "erase", "writeData", "writeRegister"]:
        assert forbidden.lower() not in text.lower()


def test_attach_readonly_derive_ccxml_dry_run_clears_initialization_script(tmp_path):
    profile = write_profile(tmp_path)
    out = tmp_path / "nogel.ccxml"
    result = run_cli("dss", "attach-readonly", "derive-ccxml", "--profile", str(profile), "--workspace", str(tmp_path), "--ccxml-out", str(out), "--output", "json")
    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    text = script_from(payload)
    assert "setOption('initialization script', '')" in text

def test_attach_readonly_allows_flash_word_inside_symbol_path(tmp_path):
    profile = write_profile(tmp_path)
    flash_dir = tmp_path / "FLASH"
    flash_dir.mkdir()
    out = flash_dir / "app.out"
    out.write_text("fake", encoding="utf-8")
    result = run_cli("dss", "attach-readonly", "monitor-symbol", "--profile", str(profile), "--workspace", str(tmp_path), "--out", str(out), "--symbol", "g_test", "--samples", "1", "--output", "json")
    assert result.returncode == 0, result.stdout
    payload = json.loads(result.stdout)
    assert payload["ok"] is True
    assert payload["data"]["script"] is not None
    assert "FLASH" in payload["data"]["out_file"]


def test_attach_readonly_detects_dss_stderr_exception():
    from ai_debug_repair.dss_attach import _stderr_has_script_error
    assert _stderr_has_script_error("org.mozilla.javascript.WrappedException: Wrapped java.lang.NullPointerException") is True
    assert _stderr_has_script_error("SLF4J: Defaulting to no-operation (NOP) logger implementation") is False

def test_attach_readonly_monitor_address_dry_run_uses_memory_read(tmp_path):
    profile = write_profile(tmp_path)
    result = run_cli("dss", "attach-readonly", "monitor-address", "--profile", str(profile), "--workspace", str(tmp_path), "--address", "0xB4C0", "--samples", "2", "--output", "json")
    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    text = uncommented(script_from(payload))
    assert "ds.memory.readData(Memory.Page.DATA,46272,16,false)" in text
    for forbidden in ["target.reset", "target.halt", "target.run", "loadProgram", "flash", "erase", "writeData", "writeRegister"]:
        assert forbidden.lower() not in text.lower()