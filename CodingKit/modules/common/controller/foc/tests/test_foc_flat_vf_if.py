import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read_text(path):
    return path.read_text(encoding="utf-8")


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def forbidden_tokens():
    return [
        "foc_set_" + "leg" + "acy_control_mode",
        "Foc" + "ControlMode",
        "FOC_CONTROL" + "_MODE_",
        "Legacy " + "helper",
        "旧入口 `fo" + "c()`",
    ]


def main():
    foc_h = read_text(ROOT / "src" / "foc.h")
    foc_c = read_text(ROOT / "src" / "foc.c")
    readme = read_text(ROOT / "README.md")
    example = read_text(ROOT / "examples" / "flat_vf_if_example.c")

    for token in ["FOC_MODE_VF", "FOC_MODE_IF", "FOC_ANGLE_SENSOR", "FOC_ANGLE_OPEN_LOOP"]:
        require(token in foc_h, f"missing {token} in foc.h")

    struct_match = re.search(r"typedef\s+struct\s*\{(?P<body>.*?)\}\s*Foc\s*;", foc_h, re.S)
    require(struct_match is not None, "missing flat Foc struct")
    struct_body = struct_match.group("body")
    for field in ["cmd_iq", "theta_e", "duty_a"]:
        require(field in struct_body, f"missing {field} in Foc")

    require('#include "pid.h"' in foc_c, "foc.c must include pid.h")
    require("static float foc_pid_error" in foc_c, "foc_pid_error wrapper missing")
    require("pid(controller)" in foc_c, "foc_pid_error must call pid()")
    require("bool foc_loop(Foc *controller)" in foc_c, "foc_loop missing")
    require(re.search(r"\nbool\s+foc\s*\(", foc_c) is None, "removed entry point must not be defined")

    for token in forbidden_tokens():
        require(token not in foc_h, f"forbidden token remains in foc.h: {token}")
        require(token not in foc_c, f"forbidden token remains in foc.c: {token}")
        require(token not in readme, f"forbidden token remains in README.md: {token}")

    require("VF / IF" in readme and "两种核心模式" in readme, "README must describe VF / IF dual modes")
    require("IF + SENSOR" in readme and "三环闭环" in readme, "README must describe IF + SENSOR")
    require("本目录不保留历史兼容层" in readme, "README must state the history boundary")

    for token in ["foc.cmd_iq", "foc.theta_e", "foc.duty_a"]:
        require(token in example, f"example must use flat field {token}")
    for forbidden in ["foc.input.", "foc.config.", "foc.state."]:
        require(forbidden not in example, f"example must not use nested field {forbidden}")

    print("flat VF/IF FOC static checks passed")


if __name__ == "__main__":
    main()
