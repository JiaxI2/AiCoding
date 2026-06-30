import json
from pathlib import Path

from ai_debug_repair.ti_dss import dss_profile_template, validate_dss_profile, generate_dss_script
from ai_debug_repair.jlink_guard import jlink_profile_template, jlink_invasive_operation


def test_ti_dss_template_validate_and_capabilities(tmp_path):
    profile = tmp_path / "ti-dss.json"
    assert dss_profile_template(profile)["ok"] is True
    result = validate_dss_profile(profile)
    assert result["ok"] is True
    caps = result["data"]["capabilities"]
    assert caps["reset"] is False
    assert caps["halt"] is False
    assert caps["flash"] is False
    assert caps["memory_write"] is False


def test_ti_dss_rejects_invasive_flags(tmp_path):
    profile = tmp_path / "ti-dss.json"
    dss_profile_template(profile)
    data = json.loads(profile.read_text())
    data["allow_reset"] = True
    profile.write_text(json.dumps(data))
    result = validate_dss_profile(profile)
    assert result["ok"] is False
    assert result["code"] == "PROFILE_INVALID"


def test_ti_dss_read_expression_generates_non_invasive_script(tmp_path):
    profile = tmp_path / "ti-dss.json"
    script = tmp_path / "read.js"
    dss_profile_template(profile)
    result = generate_dss_script(profile, "CpuTimer0Regs.TIM.all", None, script)
    assert result["ok"] is True
    text = script.read_text()
    assert "target.reset(" not in text
    assert "target.halt(" not in text
    assert ".run(" not in text


def test_jlink_invasive_interfaces_exist_but_default_denied(tmp_path):
    profile = tmp_path / "jlink.json"
    assert jlink_profile_template(profile)["ok"] is True
    result = jlink_invasive_operation(profile, "reset", approve=False)
    assert result["ok"] is False
    assert result["code"] == "POLICY_DENIED"


def test_jlink_enabled_operation_still_stubbed(tmp_path):
    profile = tmp_path / "jlink.json"
    jlink_profile_template(profile)
    data = json.loads(profile.read_text())
    data["mode"] = "maintenance"
    data["allow_reset"] = True
    profile.write_text(json.dumps(data))
    result = jlink_invasive_operation(profile, "reset", approve=True)
    assert result["ok"] is False
    assert result["code"] == "CAPABILITY_DISABLED"
