import json
import os
import subprocess
import sys
from pathlib import Path


def run_cli(*args, cwd=None):
    root = Path(__file__).resolve().parents[1]
    env = os.environ.copy()
    env["PYTHONPATH"] = str(root / "src") + os.pathsep + env.get("PYTHONPATH", "")
    return subprocess.run([sys.executable, "-m", "ai_debug_repair.cli", *args], cwd=cwd, text=True, capture_output=True, env=env)


def test_version_json():
    result = run_cli("version", "--output", "json")
    assert result.returncode == 0
    payload = json.loads(result.stdout)
    assert payload["ok"] is True
    assert payload["data"]["version"] == "0.3.2"


def test_init_and_profiles(tmp_path):
    result = run_cli("init", "--workspace", str(tmp_path), "--output", "json")
    assert result.returncode == 0
    profile = tmp_path / ".ai-debug-repair" / "profiles" / "loop.safe.json"
    assert profile.exists()
    result2 = run_cli("profile", "validate", "--profile", str(profile), "--output", "json")
    assert result2.returncode == 0


def test_build_and_test_profiles(tmp_path):
    run_cli("init", "--workspace", str(tmp_path), "--output", "json")
    build = tmp_path / ".ai-debug-repair" / "profiles" / "build.json"
    test = tmp_path / ".ai-debug-repair" / "profiles" / "test.json"
    assert run_cli("build", "run", "--profile", str(build), "--workspace", str(tmp_path), "--output", "json").returncode == 0
    assert run_cli("test", "run", "--profile", str(test), "--workspace", str(tmp_path), "--output", "json").returncode == 0
