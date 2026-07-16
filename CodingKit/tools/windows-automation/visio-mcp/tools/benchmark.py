from __future__ import annotations

import argparse
import json
from pathlib import Path
import statistics
import sys
import time

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from visio_mcp.model import load_json
from visio_mcp.service import VisioService


def percentile(values: list[float], ratio: float) -> float:
    ordered = sorted(values)
    return ordered[min(len(ordered) - 1, int((len(ordered) - 1) * ratio))]


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--renderer", default="mock")
    parser.add_argument("--iterations", type=int, default=30)
    parser.add_argument("--input", default="examples/visio-mcp-architecture.json")
    args = parser.parse_args()
    service = VisioService(renderer=args.renderer)
    data = load_json(args.input)
    timings = {"validate": [], "plan": [], "render": [], "snapshot": []}
    for index in range(args.iterations):
        started = time.perf_counter()
        service.validate(data)
        timings["validate"].append((time.perf_counter() - started) * 1000)
        started = time.perf_counter()
        service.plan(data)
        timings["plan"].append((time.perf_counter() - started) * 1000)
        output = f"test-results/bench/{index}.json"
        started = time.perf_counter()
        rendered = service.render(data, output, False, True)
        timings["render"].append((time.perf_counter() - started) * 1000)
        started = time.perf_counter()
        service.snapshot(rendered["sessionId"], f"test-results/bench/{index}.png")
        timings["snapshot"].append((time.perf_counter() - started) * 1000)
        service.close(rendered["sessionId"])
    report = {
        key: {
            "p50": round(statistics.median(values), 2),
            "p95": round(percentile(values, 0.95), 2),
            "max": round(max(values), 2),
        }
        for key, values in timings.items()
    }
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
