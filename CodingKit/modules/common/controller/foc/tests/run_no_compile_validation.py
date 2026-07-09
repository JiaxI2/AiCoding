import json
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REPORTS = ROOT / "reports"
REPORTS.mkdir(exist_ok=True)


def run(cmd):
    start = time.perf_counter()
    proc = subprocess.run(cmd, cwd=ROOT, text=True, capture_output=True)
    elapsed = time.perf_counter() - start
    return {
        "cmd": cmd,
        "returncode": proc.returncode,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
        "elapsed_s": elapsed,
    }


def perf():
    sys.path.insert(0, str(ROOT / "tests"))
    from foc_model import foc_vf
    count = 100000
    start = time.perf_counter()
    checksum = 0.0
    for i in range(count):
        out = foc_vf(24.0, 0.001 * i, 0.0, 3.0, 12.0)
        checksum += out["duty"][0]
    elapsed = time.perf_counter() - start
    return {
        "iterations": count,
        "elapsed_s": elapsed,
        "avg_us_per_iter_python_model": elapsed / count * 1_000_000.0,
        "checksum": checksum,
    }


def main():
    steps = [
        run([sys.executable, "tools/check_c_gbk.py"]),
        run([sys.executable, "tests/test_foc_behavior.py"]),
        run([sys.executable, "tests/check_static_boundary.py"]),
    ]
    performance = perf()
    passed = sum(1 for s in steps if s["returncode"] == 0)
    failed = len(steps) - passed
    result = {"passed": passed, "failed": failed, "steps": steps, "performance": performance}
    (REPORTS / "validation_results.json").write_text(json.dumps(result, indent=2, ensure_ascii=False), encoding="utf-8")
    (REPORTS / "performance_metrics.json").write_text(json.dumps(performance, indent=2), encoding="utf-8")
    lines = [
        "# Validation Report",
        "",
        f"Passed: {passed}",
        f"Failed: {failed}",
        "",
        "## Performance Mirror",
        "",
        f"Iterations: {performance['iterations']}",
        f"Average Python model time: {performance['avg_us_per_iter_python_model']:.3f} us/iter",
    ]
    for step in steps:
        lines.append("")
        lines.append("## " + " ".join(step["cmd"]))
        lines.append(f"Return code: {step['returncode']}")
        lines.append("```text")
        lines.append(step["stdout"] + step["stderr"])
        lines.append("```")
    (REPORTS / "validation_report.md").write_text("\n".join(lines), encoding="utf-8")
    print(f"Passed: {passed}")
    print(f"Failed: {failed}")
    raise SystemExit(1 if failed else 0)


if __name__ == "__main__":
    main()
