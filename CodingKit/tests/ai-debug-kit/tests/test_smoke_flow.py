import json
import os
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def test_smoke_test_creates_profile_session_and_report(tmp_path: Path) -> None:
    env = os.environ.copy()
    env["PYTHONPATH"] = str(ROOT / "src")

    result = subprocess.run(
        [
            sys.executable,
            "-m",
            "ai_debug",
            "smoke-test",
            "--workspace",
            str(tmp_path),
            "--output",
            "json",
        ],
        cwd=ROOT,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )

    assert result.returncode == 0, result.stderr
    envelope = json.loads(result.stdout)
    assert envelope["ok"] is True
    assert envelope["code"] == "OK"

    profile = Path(envelope["data"]["active_profile"])
    session = Path(envelope["data"]["session_bundle"])
    report = Path(envelope["data"]["report"])

    assert profile.is_file()
    assert session.is_dir()
    assert report.is_file()

    profile_data = json.loads(profile.read_text(encoding="utf-8"))
    assert profile_data["installation_status"] == "ready"
    assert profile_data["backend"] == "simulator"
    assert profile_data["capabilities"]["memory_read"] is True

    actions = session / "actions.jsonl"
    assert actions.is_file()
    assert "memory.read" in actions.read_text(encoding="utf-8")
    assert "readback" in report.read_text(encoding="utf-8")
