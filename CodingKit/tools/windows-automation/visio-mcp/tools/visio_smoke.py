from __future__ import annotations

import argparse
import csv
import io
import json
from pathlib import Path
import subprocess
import time

from visio_mcp.model import load_json
from visio_mcp.service import VisioService


def visio_processes() -> set[str]:
    result = subprocess.run(
        ["tasklist", "/FI", "IMAGENAME eq VISIO.EXE", "/FO", "CSV", "/NH"],
        check=False,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
    )
    processes = set()
    for row in csv.reader(io.StringIO(result.stdout)):
        if len(row) >= 2 and row[0].upper() == "VISIO.EXE":
            processes.add(row[1])
    return processes


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--visible", action="store_true")
    parser.add_argument("--input", default="examples/visio-mcp-architecture.json")
    args = parser.parse_args()
    root = Path.cwd()
    output_root = root / "test-results" / "release"
    output_root.mkdir(parents=True, exist_ok=True)
    input_path = Path(args.input)
    case_name = input_path.stem
    baseline = visio_processes()
    service = VisioService(renderer="visio")
    session_id = ""
    try:
        doctor = service.doctor()
        if not doctor["renderer"].get("available"):
            raise RuntimeError(doctor["renderer"].get("error", "Visio COM is unavailable"))
        rendered = service.render(
            load_json(root / input_path),
            output_root / f"{case_name}.vsdx",
            visible=args.visible,
            auto_repair_enabled=True,
        )
        session_id = rendered["sessionId"]
        snapshot = service.snapshot(session_id, output_root / f"{case_name}.png")
        inspected = service.inspect(session_id)
        quality = service.quality_check(session_id)
        inspection_path = output_root / f"{case_name}.inspection.json"
        quality_path = output_root / f"{case_name}.quality.json"
        inspection_path.write_text(
            json.dumps(inspected, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        quality_path.write_text(
            json.dumps(quality, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        exported = service.export(session_id, ["png", "svg", "pdf", "vsdx"], output_root)
        blocking = [
            finding
            for section in ("structure", "connectors")
            for finding in quality[section]["findings"]
            if finding["severity"] == "error"
        ]
        if blocking:
            raise RuntimeError(f"Blocking diagram quality findings: {blocking}")
        print(
            json.dumps(
                {
                    "case": case_name,
                    "doctor": doctor,
                    "render": rendered,
                    "snapshot": snapshot,
                    "inspect": {
                        "nodes": len(inspected["nodes"]),
                        "edges": len(inspected["edges"]),
                        "output": str(inspection_path),
                    },
                    "quality": quality,
                    "qualityOutput": str(quality_path),
                    "export": exported,
                },
                ensure_ascii=False,
                indent=2,
            )
        )
    finally:
        if session_id:
            service.close(session_id)
    orphaned = visio_processes() - baseline
    deadline = time.monotonic() + 15
    while orphaned and time.monotonic() < deadline:
        time.sleep(1)
        orphaned = visio_processes() - baseline
    if orphaned:
        raise RuntimeError(f"Orphaned VISIO.EXE processes: {sorted(orphaned)}")


if __name__ == "__main__":
    main()
