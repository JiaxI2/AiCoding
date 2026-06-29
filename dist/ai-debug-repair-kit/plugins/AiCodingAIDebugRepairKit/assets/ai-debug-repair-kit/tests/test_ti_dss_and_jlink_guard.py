import json
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def run_cli(*args):
    return subprocess.run([sys.executable, "-m", "ai_debug_repair.cli", *args], cwd=ROOT, text=True, capture_output=True, timeout=20)


def test_ti_dss_template_validate_and_capabilities(tmp_path, monkeypatch):
    monkeypatch.setenv("PYTHONPATH", str(ROOT / "src"))
    profile = tmp_path / "ti-dss.json"
    assert run_cli("dss", "profile-template", "--profile", str(profile), "--output", "json").returncode == 0
    r = run_cli("dss", "validate-profile", "--profile", str(profile), "--output", "json")
    assert r.returncode == 0, r.stderr
    payload = json.loads(r.stdout)
    caps = payload["data"]["capabilities"]
    assert caps["reset"] is False
    assert caps["halt"] is False
    assert caps["flash"] is False
    assert caps["memory_write"] is False


def test_ti_dss_rejects_invasive_flags(tmp_path, monkeypatch):
    monkeypatch.setenv("PYTHONPATH", str(ROOT / "src"))
    profile = tmp_path / "ti-dss.json"
    run_cli("dss", "profile-template", "--profile", str(profile), "--output", "json")
    data = json.loads(profile.read_text())
    data["allow_reset"] = True
    profile.write_text(json.dumps(data))
    r = run_cli("dss", "validate-profile", "--profile", str(profile), "--output", "json")
    assert r.returncode != 0
    assert json.loads(r.stdout)["code"] == "PROFILE_INVALID"


def test_ti_dss_read_expression_generates_non_invasive_script(tmp_path, monkeypatch):
    monkeypatch.setenv("PYTHONPATH", str(ROOT / "src"))
    profile = tmp_path / "ti-dss.json"
    script = tmp_path / "read.js"
    run_cli("dss", "profile-template", "--profile", str(profile), "--output", "json")
    r = run_cli("dss", "read-expression", "--profile", str(profile), "--expression", "CpuTimer0Regs.TIM.all", "--script-out", str(script), "--output", "json")
    assert r.returncode == 0, r.stderr
    text = script.read_text()
    assert "target.reset(" not in text
    assert "target.halt(" not in text
    assert ".run(" not in text


def test_jlink_invasive_interfaces_exist_but_default_denied(tmp_path, monkeypatch):
    monkeypatch.setenv("PYTHONPATH", str(ROOT / "src"))
    profile = tmp_path / "jlink.json"
    assert run_cli("jlink", "profile-template", "--profile", str(profile), "--output", "json").returncode == 0
    r = run_cli("jlink", "reset", "--profile", str(profile), "--output", "json")
    assert r.returncode != 0
    payload = json.loads(r.stdout)
    assert payload["code"] == "POLICY_DENIED"


def test_jlink_enabled_operation_still_stubbed(tmp_path, monkeypatch):
    monkeypatch.setenv("PYTHONPATH", str(ROOT / "src"))
    profile = tmp_path / "jlink.json"
    run_cli("jlink", "profile-template", "--profile", str(profile), "--output", "json")
    data = json.loads(profile.read_text())
    data["mode"] = "maintenance"
    data["allow_reset"] = True
    profile.write_text(json.dumps(data))
    r = run_cli("jlink", "reset", "--profile", str(profile), "--approve", "--output", "json")
    assert r.returncode != 0
    payload = json.loads(r.stdout)
    assert payload["code"] == "CAPABILITY_DISABLED"
