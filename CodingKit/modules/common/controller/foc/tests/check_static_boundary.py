from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FORBIDDEN = [
    "ODrive", "Dummy", "peng-zhihui", "odriverobotics", "VESC", "vedderb", "bldc"
]
REQUIRED_C_FILES = [
    "src/foc_math.h", "src/foc_math.c",
    "src/foc_svpwm.h", "src/foc_svpwm.c",
    "src/foc_angle.h", "src/foc_angle.c",
    "src/foc_motion.h", "src/foc_motion.c",
    "src/foc.h", "src/foc.c",
]
REQUIRED_README_PHRASES = [
    "开环电压", "闭环电流", "FOC_ANGLE_MODE_OPEN_LOOP",
    "FOC_CONTROL_MODE_CLOSED_CURRENT", "foc_angle_update", "foc_svpwm",
    "foc_init", "零电流偏置", "foc_motion_update", "anti-cogging", "前馈"
]
REQUIRED_C_PHRASES = [
    "FOC_ANGLE_MODE_OPEN_LOOP",
    "FOC_CONTROL_MODE_OPEN_VOLTAGE",
    "FOC_CONTROL_MODE_CLOSED_CURRENT",
    "开环电压模式",
    "闭环电流模式",
    "foc_init",
    "foc_current_offset_accumulate",
    "零电流偏置",
    "FOC_MOTION_CONTROL_POSITION",
    "FOC_MOTION_CONTROL_VELOCITY",
    "FOC_MOTION_INPUT_POS_FILTER",
    "foc_motion_update",
]


def main():
    failures = []
    for rel in REQUIRED_C_FILES:
        path = ROOT / rel
        if not path.exists():
            failures.append(f"missing {rel}")
            continue
        text = path.read_text(encoding="gbk")
        for word in FORBIDDEN:
            if word in text:
                failures.append(f"{rel}: forbidden word {word}")
        if "@author HU JIAXUAN" not in text:
            failures.append(f"{rel}: missing author")
    combined_c = "\n".join((ROOT / rel).read_text(encoding="gbk") for rel in REQUIRED_C_FILES)
    for phrase in REQUIRED_C_PHRASES:
        if phrase not in combined_c:
            failures.append(f"C source missing phrase: {phrase}")
    readme = (ROOT / "README.md").read_text(encoding="utf-8")
    for word in FORBIDDEN:
        if word in readme:
            failures.append(f"README.md: forbidden word {word}")
    for phrase in REQUIRED_README_PHRASES:
        if phrase not in readme:
            failures.append(f"README missing phrase: {phrase}")
    if failures:
        print("Failed:")
        for item in failures:
            print("- " + item)
        raise SystemExit(1)
    print("Static boundary checks passed.")


if __name__ == "__main__":
    main()
