import json
import os
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def run_ai_debug(*args: str) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env["PYTHONPATH"] = str(ROOT / "src")
    return subprocess.run(
        [sys.executable, "-m", "ai_debug", *args],
        cwd=ROOT,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def load_envelope(stdout: str) -> dict:
    return json.loads(stdout)


def test_version_outputs_json_envelope() -> None:
    result = run_ai_debug("version", "--output", "json")

    assert result.returncode == 0, result.stderr
    envelope = load_envelope(result.stdout)
    assert envelope["schema_version"] == "1.0"
    assert envelope["ok"] is True
    assert envelope["code"] == "OK"
    assert envelope["data"]["name"] == "ai-debug-kit"
    assert envelope["data"]["version"]


def test_doctor_outputs_capability_summary() -> None:
    result = run_ai_debug("doctor", "--output", "json")

    assert result.returncode == 0, result.stderr
    envelope = load_envelope(result.stdout)
    assert envelope["ok"] is True
    assert envelope["code"] == "OK"
    assert envelope["data"]["backend"] == "simulator"
    assert envelope["data"]["validated"]["cli"] is True
    assert envelope["data"]["validated"]["simulator"] is True
