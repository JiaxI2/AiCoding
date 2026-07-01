from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
HEADER = (ROOT / "src" / "pid.h").read_text(encoding="gbk")
SOURCE = (ROOT / "src" / "pid.c").read_text(encoding="gbk")
README = (ROOT / "README.md").read_text(encoding="utf-8")
DESIGN = (ROOT / "docs" / "design.md").read_text(encoding="utf-8")


def test_public_function_declarations():
    pid_decls = re.findall(r"^float\s+pid\s*\(\s*Pid\s*\*\s*controller\s*\)\s*;", HEADER, re.M)
    init_decls = re.findall(r"^bool\s+pid_init\s*\(\s*Pid\s*\*\s*controller\s*\)\s*;", HEADER, re.M)
    assert len(pid_decls) == 1
    assert len(init_decls) == 1
    assert "float pid(Pid *controller, float" not in HEADER
    assert "pid_reset" not in HEADER
    assert "pid_step" not in HEADER


def test_external_function_definitions_are_limited():
    external_defs = re.findall(r"^(?!static)(?:[A-Za-z_][\w\s\*]+)\s+([A-Za-z_]\w*)\s*\([^;]*\)\s*\{", SOURCE, re.M)
    assert external_defs == ["pid_init", "pid"]


def test_core_has_no_application_terms():
    forbidden = ["position", "velocity", "current", "axis", "motor", "FOC", "PWM", "ADC", "encoder"]
    core = (HEADER + SOURCE).lower()
    assert not any(term.lower() in core for term in forbidden)


def test_package_has_no_third_party_project_names():
    forbidden = ["OD" + "rive", "Dummy" + "-Robot", "peng" + "-zhihui", "odriver" + "obotics"]
    text = HEADER + SOURCE + README + DESIGN
    assert not any(term in text for term in forbidden)


def test_author_is_hu_jiaxuan():
    assert "HU JIAXUAN" in HEADER
    assert "HU JIAXUAN" in SOURCE
    assert "   Hu          " not in SOURCE
    assert ("OD" + "rive") not in SOURCE


def test_struct_fields_have_chinese_comments():
    for field in ["setpoint", "feedback", "feedforward", "kp", "ki", "kd", "controlFreq", "outputLimit", "state"]:
        assert field in HEADER
    assert "单位同被控量" in HEADER
    assert "标幺时为 pu" in HEADER
    assert "/* 控制频率，单位 Hz。 */" in HEADER
    assert "antiWindupGain" in HEADER
    assert "被控量单位/输出单位" in HEADER
    assert "/* 配置区。 */" in HEADER


def test_readme_documents_required_config():
    assert "最少必配参数" in README
    assert "controlFreq" in README
    assert "kp / ki / kd" in README
    assert "outputLimit" in README
    assert "config.dt" not in README
    assert "float dt" not in README


def test_c99_style_basics():
    assert "\t" not in HEADER
    assert "\t" not in SOURCE
    assert "float pid(Pid *controller)\n{" in SOURCE


def test_header_has_module_level_usage_notes():
    assert "@file pid.h" in HEADER
    assert "通用 PID 控制单元" in HEADER
    assert "使用流程" in HEADER
    assert "单周期控制链" in HEADER
    assert "设计边界" in HEADER
    assert "输入整形" in HEADER
    assert "微分滤波" in HEADER
    assert "bool pid_init(Pid *controller);" in HEADER
    assert "float pid(Pid *controller);" in HEADER
    assert len(HEADER.splitlines()) < 230


def test_c_sources_are_gbk():
    for path in list((ROOT / "src").glob("*.c")) + list((ROOT / "src").glob("*.h")) + list((ROOT / "examples").glob("*.c")):
        path.read_text(encoding="gbk")
    assert "GBK" in README


def test_units_and_pu_are_documented():
    assert "单位约定" in README
    assert "pu" in README
    assert "controlFreq" in HEADER
    assert "单位 Hz" in HEADER
    assert "pu/s" in HEADER


def test_readme_documents_controller_unit_boundary():
    assert "通用 PID 控制单元" in README
    assert "控制链说明" in README
    assert "内置控制工具边界" in README
    assert "setpointLimit" in README
    assert "derivativeFilterCoef" in README
    assert "deadband" in README
    assert "上层业务" in README
    assert "单周期控制链" in DESIGN
    assert "内置可选工具" in DESIGN
