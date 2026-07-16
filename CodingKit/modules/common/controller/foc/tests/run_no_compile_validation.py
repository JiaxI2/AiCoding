import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPORTS = ROOT / "reports"
REPORTS.mkdir(exist_ok=True)


def write_report(path, content):
    path.write_text(content.rstrip("\n") + "\n", encoding="utf-8", newline="\n")


def run(script):
    command = [sys.executable, script]
    proc = subprocess.run(command, cwd=ROOT, text=True, capture_output=True)
    return {
        "cmd": ["python", script],
        "returncode": proc.returncode,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
    }


def workload_mirror():
    sys.path.insert(0, str(ROOT / "tests"))
    from foc_model import foc_vf

    count = 100000
    checksum = 0.0
    for i in range(count):
        out = foc_vf(24.0, 0.001 * i, 0.0, 3.0, 12.0)
        checksum += out["duty"][0]
    return {
        "measurement_policy": "deterministic-output-only",
        "iterations": count,
        "checksum": round(checksum, 9),
    }


def main():
    steps = [
        run("tools/check_c_gbk.py"),
        run("tests/test_foc_behavior.py"),
        run("tests/check_static_boundary.py"),
    ]
    performance = workload_mirror()
    passed = sum(1 for s in steps if s["returncode"] == 0)
    failed = len(steps) - passed
    result = {"passed": passed, "failed": failed, "steps": steps, "performance": performance}
    write_report(REPORTS / "validation_results.json", json.dumps(result, indent=2, ensure_ascii=False))
    write_report(REPORTS / "performance_metrics.json", json.dumps(performance, indent=2))
    lines = [
        "# Validation Report",
        "",
        f"Passed: {passed}",
        f"Failed: {failed}",
        "",
        "## Deterministic Workload Mirror",
        "",
        f"Measurement policy: {performance['measurement_policy']}",
        f"Iterations: {performance['iterations']}",
        f"Checksum: {performance['checksum']}",
        "Wall-clock timing is intentionally not written to versioned reports.",
    ]
    for step in steps:
        lines.append("")
        lines.append("## " + " ".join(step["cmd"]))
        lines.append(f"Return code: {step['returncode']}")
        lines.append("```text")
        lines.append((step["stdout"] + step["stderr"]).rstrip("\n"))
        lines.append("```")
    write_report(REPORTS / "validation_report.md", "\n".join(lines))
    print(f"Passed: {passed}")
    print(f"Failed: {failed}")
    raise SystemExit(1 if failed else 0)


if __name__ == "__main__":
    main()
