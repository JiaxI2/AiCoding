from __future__ import annotations

from pathlib import Path
import xml.etree.ElementTree as ET
import zipfile

from pid_model import Config, Limit, PidModel, pid


ROOT = Path(__file__).resolve().parents[1]
MODEL = ROOT / "simulink" / "models" / "reference" / "antiwindup_complete_reference.slx"


def _load_system_root():
    with zipfile.ZipFile(MODEL, "r") as archive:
        return ET.fromstring(archive.read("simulink/systems/system_root.xml"))


def _params(node):
    return {p.attrib.get("Name"): (p.text or "") for p in node.findall("P")}


def _blocks(system):
    result = {}
    for block in system.findall("Block"):
        name = block.attrib.get("Name")
        result[name] = {
            "type": block.attrib.get("BlockType"),
            "sid": block.attrib.get("SID"),
            "params": _params(block),
        }
    return result


def _lines(system):
    result = []
    for line in system.findall("Line"):
        params = _params(line)
        src = params.get("Src")
        dst = params.get("Dst")
        if src and dst:
            result.append((src, dst))
        for branch in line.findall("Branch"):
            bparams = _params(branch)
            bdst = bparams.get("Dst")
            if src and bdst:
                result.append((src, bdst))
    return result


def test_reference_model_is_expected_back_calculation_topology():
    system = _load_system_root()
    blocks = _blocks(system)
    lines = set(_lines(system))

    assert blocks["Saturation"]["type"] == "Saturate"
    assert blocks["Saturation"]["params"]["UpperLimit"] == "2"
    assert blocks["Saturation"]["params"]["LowerLimit"] == "-2"

    assert blocks["Gain"]["type"] == "Gain"
    assert blocks["Gain"]["params"]["Gain"] == "1400"

    assert blocks["Gain2"]["type"] == "Gain"
    assert blocks["Gain2"]["params"]["Gain"] == "20"

    assert blocks["Sum2"]["params"]["Inputs"] == "|+-"
    assert blocks["Sum3"]["params"]["Inputs"] == "|+-"

    # uc -> Saturation -> u
    assert ("5#out:1", "27#in:1") in lines
    # uc and u enter Sum2 as uc - u
    assert ("5#out:1", "28#in:1") in lines
    assert ("27#out:1", "28#in:2") in lines
    # anti-windup correction goes back to the integrator input path
    assert ("28#out:1", "29#in:1") in lines
    assert ("29#out:1", "30#in:2") in lines
    # e and correction enter Sum3 as e - Kaw * (uc - u)
    assert ("12#out:1", "30#in:1") in lines
    assert ("30#out:1", "4#in:1") in lines
    assert ("4#out:1", "3#in:1") in lines


def test_pid_formula_matches_reference_topology_for_one_step():
    model = PidModel(Config(
        kp=1.0,
        ki=1400.0,
        kd=0.0,
        control_freq=1000.0,
        output_limit=Limit(True, -2.0, 2.0),
        anti_windup_gain=20.0,
    ))

    model.input.setpoint = 10.0
    model.input.feedback = 0.0
    out = pid(model)

    expected_raw = 10.0
    expected_sat = 2.0
    expected_integral = 1400.0 * (10.0 + 20.0 * (expected_sat - expected_raw)) * 0.001

    assert out == expected_sat
    assert abs(model.state.raw_output - expected_raw) < 1e-9
    assert abs(model.state.output - expected_sat) < 1e-9
    assert abs(model.state.integral - expected_integral) < 1e-9
