from __future__ import annotations

import argparse
import csv
import io
import json
from pathlib import Path
import subprocess
import time

import pythoncom
import win32com.client


def powerpoint_processes() -> set[str]:
    result = subprocess.run(
        ["tasklist", "/FI", "IMAGENAME eq POWERPNT.EXE", "/FO", "CSV", "/NH"],
        check=False,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
    )
    processes = set()
    for row in csv.reader(io.StringIO(result.stdout)):
        if len(row) >= 2 and row[0].upper() == "POWERPNT.EXE":
            processes.add(row[1])
    return processes


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--visible", action="store_true")
    args = parser.parse_args()
    if not args.visible:
        raise RuntimeError("Release smoke requires explicit --visible")

    baseline = powerpoint_processes()
    if baseline:
        raise RuntimeError(
            "Close existing PowerPoint sessions before Release smoke so the test "
            "cannot attach to or close a user-owned session"
        )

    output_root = Path.cwd() / "test-results" / "release"
    output_root.mkdir(parents=True, exist_ok=True)
    output = (output_root / "ppt-mcp-smoke.pptx").resolve()
    app = None
    presentation = None
    pythoncom.CoInitialize()
    try:
        app = win32com.client.Dispatch("PowerPoint.Application")
        app.Visible = True
        presentation = app.Presentations.Add()
        slide = presentation.Slides.Add(1, 12)
        shape = slide.Shapes.AddTextbox(1, 72, 72, 520, 72)
        shape.TextFrame.TextRange.Text = "ppt-mcp Release smoke"
        presentation.SaveAs(str(output), 24)
        observed_text = shape.TextFrame.TextRange.Text
        slide_count = presentation.Slides.Count
        if slide_count != 1 or observed_text != "ppt-mcp Release smoke":
            raise RuntimeError("PowerPoint COM round-trip did not preserve content")
        if not output.is_file() or output.stat().st_size == 0:
            raise RuntimeError(f"PowerPoint smoke output is missing: {output}")
        print(
            json.dumps(
                {
                    "ok": True,
                    "visible": True,
                    "slides": slide_count,
                    "output": str(output),
                },
                ensure_ascii=False,
            )
        )
    finally:
        try:
            if presentation is not None:
                presentation.Saved = True
                presentation.Close()
        finally:
            try:
                if app is not None:
                    app.Quit()
            finally:
                pythoncom.CoUninitialize()

    orphaned = powerpoint_processes() - baseline
    deadline = time.monotonic() + 15
    while orphaned and time.monotonic() < deadline:
        time.sleep(1)
        orphaned = powerpoint_processes() - baseline
    if orphaned:
        raise RuntimeError(f"Orphaned POWERPNT.EXE processes: {sorted(orphaned)}")


if __name__ == "__main__":
    main()
