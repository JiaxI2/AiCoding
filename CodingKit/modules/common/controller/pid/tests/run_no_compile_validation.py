from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import traceback

ROOT = Path(__file__).resolve().parents[1]
TEST_DIR = ROOT / "tests"
REPORT_DIR = ROOT / "reports"
REPORT_DIR.mkdir(exist_ok=True)


def load_module(path: Path):
    spec = importlib.util.spec_from_file_location(path.stem, path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


def main() -> int:
    import sys
    sys.path.insert(0, str(TEST_DIR))
    results = []
    for file in sorted(TEST_DIR.glob("test_*.py")):
        module = load_module(file)
        for name in sorted(dir(module)):
            if not name.startswith("test_"):
                continue
            func = getattr(module, name)
            try:
                func()
                results.append({"name": f"{file.name}::{name}", "status": "PASS"})
            except Exception as exc:
                results.append({
                    "name": f"{file.name}::{name}",
                    "status": "FAIL",
                    "error": str(exc),
                    "traceback": traceback.format_exc(),
                })

    passed = sum(1 for item in results if item["status"] == "PASS")
    failed = len(results) - passed
    (REPORT_DIR / "validation_results.json").write_text(
        json.dumps({"passed": passed, "failed": failed, "tests": results}, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )

    # 性能报告：与 test_pid_performance 使用同一组参数，记录最终值。
    from pid_model import Config, Limit, PidModel, pid
    model = PidModel(Config(
        kp=2.0,
        ki=5.0,
        kd=0.0,
        control_freq=1000.0,
        output_limit=Limit(True, -1.0, 1.0),
        integral_limit=Limit(True, -1.0, 1.0),
        anti_windup_gain=1.0,
    ))
    plant = 0.0
    max_output = 0.0
    settled_at = None
    overshoot = 0.0
    for i in range(5000):
        model.input.setpoint = 1.0
        model.input.feedback = plant
        u = pid(model)
        plant += 0.001 * ((2.0 * u) - plant) / 0.15
        max_output = max(max_output, abs(u))
        overshoot = max(overshoot, plant - 1.0)
        if settled_at is None and i > 10 and abs(plant - 1.0) < 0.02:
            settled_at = i * 0.001
    metrics = {
        "control_freq_hz": 1000.0,
        "period_s": 0.001,
        "sim_time_s": 5.0,
        "final_value": plant,
        "steady_error": abs(1.0 - plant),
        "overshoot": max(0.0, overshoot),
        "settled_at_s": settled_at,
        "max_abs_output": max_output,
        "final_integral": model.state.integral,
    }
    (REPORT_DIR / "performance_metrics.json").write_text(
        json.dumps(metrics, indent=2, ensure_ascii=False), encoding="utf-8"
    )
    lines = [
        "# v1.6 no-compile 验证报告",
        "",
        f"- Passed: {passed}",
        f"- Failed: {failed}",
        "",
        "## 性能镜像仿真",
        "",
    ]
    for key, value in metrics.items():
        lines.append(f"- {key}: {value}")
    (REPORT_DIR / "validation_report.md").write_text("\n".join(lines) + "\n", encoding="utf-8")

    print(f"Passed: {passed}")
    print(f"Failed: {failed}")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
