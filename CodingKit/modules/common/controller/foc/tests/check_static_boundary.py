import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REQUIRED_FILES = [
    "src/foc_math.h", "src/foc_math.c",
    "src/foc_svpwm.h", "src/foc_svpwm.c",
    "src/foc.h", "src/foc.c",
]
REMOVED_FILES = [
    f"src/foc_{'angle'}.h", f"src/foc_{'angle'}.c",
    f"src/foc_{'motion'}.h", f"src/foc_{'motion'}.c",
]
FORBIDDEN_TOKENS = [
    "foc_set_" + "leg" + "acy_control_mode",
    "Foc" + "ControlMode",
    "FOC_CONTROL" + "_MODE_",
    "Legacy " + "helper",
    "旧入口 `fo" + "c()`",
    "foc_" + "run",
    "修改" + "记录",
]
SCAN_DIRS = ["src", "examples", "tests", "docs"]


def read_text(path):
    data = path.read_bytes()
    for encoding in ("utf-8", "gbk"):
        try:
            return data.decode(encoding)
        except UnicodeDecodeError:
            pass
    return data.decode("utf-8", errors="ignore")


def iter_text_files():
    for dirname in SCAN_DIRS:
        base = ROOT / dirname
        if not base.exists():
            continue
        for path in base.rglob("*"):
            if path.is_file() and path.suffix.lower() in {".c", ".h", ".py", ".md", ".txt"}:
                yield path
    yield ROOT / "README.md"
    yield ROOT / "CMakeLists.txt"


def static_function_names(text, terminator):
    pattern = r"^static\s+[A-Za-z_][A-Za-z0-9_\s\*]*?\s+([A-Za-z_][A-Za-z0-9_]*)\s*\([^;{}]*\)\s*" + terminator
    return set(re.findall(pattern, text, re.M))


def check_static_forward_declarations(path, failures):
    text = read_text(path)
    definitions = static_function_names(text, r"\{")
    declarations = static_function_names(text, r";")
    for name in sorted(definitions - declarations):
        rel = path.relative_to(ROOT)
        failures.append(f"{rel}: static function missing forward declaration: {name}")


def main():
    failures = []
    for rel in REQUIRED_FILES:
        path = ROOT / rel
        if not path.exists():
            failures.append(f"missing {rel}")
            continue
        text = read_text(path)
        if "@author HU JIAXUAN" not in text:
            failures.append(f"{rel}: missing author")
    for rel in REMOVED_FILES:
        if (ROOT / rel).exists():
            failures.append(f"removed module still exists: {rel}")

    foc_h = read_text(ROOT / "src" / "foc.h")
    foc_c = read_text(ROOT / "src" / "foc.c")
    for token in ["FOC_MODE_VF", "FOC_MODE_IF", "FOC_ANGLE_SENSOR", "FOC_ANGLE_OPEN_LOOP"]:
        if token not in foc_h:
            failures.append(f"foc.h missing {token}")
    if "bool foc_loop(Foc *controller)" not in foc_h or "bool foc_loop(Foc *controller)" not in foc_c:
        failures.append("foc_loop entry point missing")

    for path in (ROOT / "src").glob("*.c"):
        check_static_forward_declarations(path, failures)

    for path in iter_text_files():
        text = read_text(path)
        rel = path.relative_to(ROOT)
        for token in FORBIDDEN_TOKENS:
            if token in text:
                failures.append(f"{rel}: forbidden token remains: {token}")
        if re.search(r"(?<![A-Za-z0-9_])foc\s*\(", text):
            failures.append(f"{rel}: removed foc entry point call remains")
        if "\ufffd" in text:
            failures.append(f"{rel}: replacement-character mojibake remains")

    if failures:
        print("Failed:")
        for item in failures:
            print("- " + item)
        raise SystemExit(1)
    print("Static boundary checks passed.")


if __name__ == "__main__":
    main()
