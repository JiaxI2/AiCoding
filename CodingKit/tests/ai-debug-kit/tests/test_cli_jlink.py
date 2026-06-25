import json
import os
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def run_ai_debug(*args: str) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env["PYTHONPATH"] = str(ROOT / "src")
    env["AI_DEBUG_JLINK_FAKE"] = "1"
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


def test_backend_list_outputs_simulator_and_jlink() -> None:
    result = run_ai_debug("backend", "list", "--output", "json")

    assert result.returncode == 0, result.stderr
    envelope = load_envelope(result.stdout)
    assert envelope["ok"] is True
    assert envelope["data"]["backends"] == ["jlink", "simulator"]


def test_jlink_discover_outputs_stable_json_envelope() -> None:
    result = run_ai_debug("backend", "discover", "--backend", "jlink", "--output", "json")

    assert result.returncode == 0, result.stderr
    envelope = load_envelope(result.stdout)
    assert envelope["ok"] is True
    assert envelope["code"] == "OK"
    assert envelope["data"]["devices"][0]["serial_number"] == 12345678


def test_jlink_memory_read_outputs_requested_length() -> None:
    result = run_ai_debug(
        "memory",
        "read",
        "0x20000000",
        "4",
        "--backend",
        "jlink",
        "--output",
        "json",
    )

    assert result.returncode == 0, result.stderr
    envelope = load_envelope(result.stdout)
    assert envelope["ok"] is True
    assert envelope["data"]["octet_length"] == 4
    assert envelope["data"]["raw_octets"] == "00010203"


def test_jlink_register_read_outputs_value() -> None:
    result = run_ai_debug("register", "read", "R0", "--backend", "jlink", "--output", "json")

    assert result.returncode == 0, result.stderr
    envelope = load_envelope(result.stdout)
    assert envelope["ok"] is True
    assert envelope["data"]["name"] == "R0"
    assert envelope["data"]["value"] == "0x12345678"
