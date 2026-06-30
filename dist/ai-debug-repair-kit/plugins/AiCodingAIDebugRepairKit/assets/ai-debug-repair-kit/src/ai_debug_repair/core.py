from __future__ import annotations

import json
import platform
import shutil
import subprocess
import sys
import time
from pathlib import Path
from typing import Any


def now_ms() -> int:
    return int(time.time() * 1000)


def code_to_exit(code: str) -> int:
    return {
        "OK": 0,
        "INVALID_ARGUMENT": 2,
        "DEPENDENCY_MISSING": 3,
        "PROFILE_INVALID": 13,
        "POLICY_DENIED": 9,
        "VALIDATION_FAILED": 10,
        "COMMAND_FAILED": 10,
        "TIMEOUT": 11,
        "INTERNAL_ERROR": 20,
    }.get(code, 20)


def envelope(ok: bool, code: str, message: str, data: dict[str, Any] | None = None,
             warnings: list[str] | None = None, side_effects: list[str] | None = None,
             started_ms: int | None = None) -> dict[str, Any]:
    return {
        "schema_version": "1.0",
        "ok": ok,
        "code": code,
        "message": message,
        "data": data or {},
        "warnings": warnings or [],
        "side_effects": side_effects or [],
        "duration_ms": (now_ms() - started_ms) if started_ms else 0,
        "trace_id": "",
        "session_id": "",
    }


def print_result(result: dict[str, Any], output: str = "text") -> int:
    if output == "json":
        print(json.dumps(result, ensure_ascii=False, indent=2))
    elif output in {"md", "markdown"}:
        print(result.get("markdown") or json.dumps(result.get("data") or {}, ensure_ascii=False, indent=2))
    else:
        if result.get("text"):
            print(result["text"])
        else:
            status = "OK" if result.get("ok") else "FAIL"
            print(f"[{status}] {result.get('code')}: {result.get('message')}")
            data = result.get("data") or {}
            if data:
                print(json.dumps(data, ensure_ascii=False, indent=2))
            for warning in result.get("warnings", []):
                print(f"WARNING: {warning}", file=sys.stderr)
    return code_to_exit(str(result.get("code", "INTERNAL_ERROR")))


def load_json(path: Path) -> dict[str, Any]:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError as exc:
        raise ValueError(f"Missing file: {path}") from exc
    except json.JSONDecodeError as exc:
        raise ValueError(f"Invalid JSON {path}: {exc}") from exc


def write_json(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def append_jsonl(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(data, ensure_ascii=False) + "\n")


def state_root(workspace: Path) -> Path:
    return workspace / ".ai-debug-repair"


def package_root() -> Path:
    return Path(__file__).resolve().parents[2]


def examples_root() -> Path:
    return package_root() / "examples"


def copy_examples(workspace: Path) -> dict[str, str]:
    root = state_root(workspace)
    profiles = root / "profiles"
    profiles.mkdir(parents=True, exist_ok=True)
    source = examples_root() / "profiles"
    copied: dict[str, str] = {}
    for name in ["build.json", "test.json", "loop.safe.json"]:
        dst = profiles / name
        if not dst.exists():
            shutil.copy2(source / name, dst)
            copied[name] = str(dst)
    return copied


def doctor() -> dict[str, Any]:
    return {
        "python": sys.version.split()[0],
        "python_executable": sys.executable,
        "platform": platform.platform(),
        "cwd": str(Path.cwd()),
        "git": shutil.which("git"),
        "pip": shutil.which("pip") or shutil.which("pip3"),
        "package": "ai-debug-repair-kit",
    }


def validate_loop_profile(profile: dict[str, Any]) -> list[str]:
    errors: list[str] = []
    max_iterations = int(profile.get("max_iterations", 0))
    if max_iterations <= 0:
        errors.append("max_iterations must be > 0")
    if max_iterations > 5 and not profile.get("allow_more_than_5", False):
        errors.append("max_iterations > 5 requires allow_more_than_5=true")
    if not profile.get("build_profile"):
        errors.append("build_profile is required")
    if not profile.get("test_profile"):
        errors.append("test_profile is required")
    if not profile.get("allowed_paths"):
        errors.append("allowed_paths must not be empty")
    forbidden = set(profile.get("forbidden_paths", []))
    allowed = set(profile.get("allowed_paths", []))
    if forbidden & allowed:
        errors.append("forbidden_paths must not overlap allowed_paths")
    if profile.get("allow_flash", False):
        errors.append("allow_flash is not supported in this kit version")
    if profile.get("auto_commit", False):
        errors.append("auto_commit is forbidden")
    return errors


def validate_profile_file(profile_path: Path) -> dict[str, Any]:
    started = now_ms()
    try:
        profile = load_json(profile_path)
    except ValueError as exc:
        return envelope(False, "PROFILE_INVALID", str(exc), started_ms=started)

    if profile_path.name.endswith("loop.safe.json") or profile.get("type") == "repair_loop":
        errors = validate_loop_profile(profile)
    elif profile.get("type") in {"build", "test"}:
        command = profile.get("command")
        errors = []
        if not isinstance(command, list) or not command:
            errors.append("command must be a non-empty list")
        if profile.get("allow_shell", False):
            errors.append("allow_shell=true is forbidden")
    else:
        errors = ["unknown profile type"]

    if errors:
        return envelope(False, "PROFILE_INVALID", "Profile validation failed", {"errors": errors, "profile": str(profile_path)}, started_ms=started)
    return envelope(True, "OK", "Profile is valid", {"profile": str(profile_path), "type": profile.get("type")}, started_ms=started)


def run_command_profile(profile_path: Path, workspace: Path, kind: str, timeout: int | None = None) -> dict[str, Any]:
    started = now_ms()
    try:
        profile = load_json(profile_path)
    except ValueError as exc:
        return envelope(False, "PROFILE_INVALID", str(exc), started_ms=started)

    command = profile.get("command")
    if not isinstance(command, list) or not command:
        return envelope(False, "PROFILE_INVALID", f"{kind} profile command must be a non-empty list", started_ms=started)
    if profile.get("allow_shell", False):
        return envelope(False, "POLICY_DENIED", "allow_shell=true is forbidden", started_ms=started)

    cwd = Path(profile.get("cwd", "."))
    if not cwd.is_absolute():
        cwd = workspace / cwd
    timeout_s = int(timeout or profile.get("timeout_seconds", 120))
    run_id = str(int(time.time() * 1000))
    logs_dir = state_root(workspace) / "runs" / run_id
    logs_dir.mkdir(parents=True, exist_ok=True)
    stdout_path = logs_dir / f"{kind}.stdout.log"
    stderr_path = logs_dir / f"{kind}.stderr.log"

    try:
        completed = subprocess.run(command, cwd=str(cwd), text=True, capture_output=True, timeout=timeout_s, shell=False)
    except FileNotFoundError:
        return envelope(False, "DEPENDENCY_MISSING", f"Command not found: {command[0]}", {"command": command}, started_ms=started)
    except subprocess.TimeoutExpired:
        return envelope(False, "TIMEOUT", f"{kind} command timed out after {timeout_s}s", {"command": command}, started_ms=started)

    stdout_path.write_text(completed.stdout or "", encoding="utf-8", errors="replace")
    stderr_path.write_text(completed.stderr or "", encoding="utf-8", errors="replace")

    expected_exit = int(profile.get("pass_exit_code", 0))
    ok = completed.returncode == expected_exit
    data: dict[str, Any] = {
        "kind": kind,
        "command": command,
        "cwd": str(cwd),
        "returncode": completed.returncode,
        "expected_exit": expected_exit,
        "stdout_path": str(stdout_path),
        "stderr_path": str(stderr_path),
        "run_id": run_id,
    }

    verdict_json = profile.get("verdict_json")
    if verdict_json:
        verdict_path = Path(verdict_json)
        if not verdict_path.is_absolute():
            verdict_path = cwd / verdict_path
        data["verdict_json"] = str(verdict_path)
        if verdict_path.exists():
            try:
                verdict = load_json(verdict_path)
                data["verdict"] = verdict
                pass_field = str(profile.get("verdict_pass_field", "ok"))
                if pass_field in verdict:
                    ok = bool(verdict.get(pass_field))
            except ValueError as exc:
                return envelope(False, "VALIDATION_FAILED", str(exc), data, started_ms=started)
        else:
            return envelope(False, "VALIDATION_FAILED", f"verdict_json not found: {verdict_path}", data, started_ms=started)

    result = envelope(ok, "OK" if ok else "COMMAND_FAILED", f"{kind} {'passed' if ok else 'failed'}", data, started_ms=started)
    append_jsonl(state_root(workspace) / "runs" / "runs.jsonl", result)
    return result


def git_status(workspace: Path) -> dict[str, Any]:
    if not shutil.which("git"):
        return {"available": False, "status": None}
    try:
        completed = subprocess.run(["git", "status", "--short"], cwd=str(workspace), text=True, capture_output=True, timeout=10, shell=False)
        return {"available": True, "returncode": completed.returncode, "status": completed.stdout.splitlines(), "stderr": completed.stderr}
    except Exception as exc:
        return {"available": True, "error": str(exc)}


def loop_status(profile_path: Path, workspace: Path) -> dict[str, Any]:
    started = now_ms()
    validation = validate_profile_file(profile_path)
    if not validation["ok"]:
        return validation
    profile = load_json(profile_path)
    attempts_path = state_root(workspace) / "attempts.jsonl"
    attempts = []
    if attempts_path.exists():
        attempts = [line for line in attempts_path.read_text(encoding="utf-8").splitlines() if line.strip()]
    data = {
        "profile": str(profile_path),
        "max_iterations": profile.get("max_iterations"),
        "attempts_used": len(attempts),
        "remaining": max(0, int(profile.get("max_iterations", 0)) - len(attempts)),
        "git": git_status(workspace),
        "attempts_path": str(attempts_path),
    }
    return envelope(True, "OK", "Loop status generated", data, started_ms=started)


def export_context(profile_path: Path, workspace: Path) -> dict[str, Any]:
    started = now_ms()
    validation = validate_profile_file(profile_path)
    if not validation["ok"]:
        return validation
    profile = load_json(profile_path)
    root = state_root(workspace)
    context_path = root / "repair-context.json"
    build_profile = Path(profile["build_profile"])
    test_profile = Path(profile["test_profile"])
    if not build_profile.is_absolute():
        build_profile = workspace / build_profile
    if not test_profile.is_absolute():
        test_profile = workspace / test_profile
    context = {
        "workspace": str(workspace),
        "profile": profile,
        "build_profile_path": str(build_profile),
        "test_profile_path": str(test_profile),
        "build_profile_exists": build_profile.exists(),
        "test_profile_exists": test_profile.exists(),
        "git": git_status(workspace),
        "allowed_paths": profile.get("allowed_paths", []),
        "forbidden_paths": profile.get("forbidden_paths", []),
        "generated_at_ms": now_ms(),
    }
    write_json(context_path, context)
    return envelope(True, "OK", "Repair context exported", {"context_path": str(context_path), "context": context}, started_ms=started)


def record_attempt(profile_path: Path, workspace: Path, result: str, notes: str) -> dict[str, Any]:
    started = now_ms()
    validation = validate_profile_file(profile_path)
    if not validation["ok"]:
        return validation
    profile = load_json(profile_path)
    attempts_path = state_root(workspace) / "attempts.jsonl"
    existing = []
    if attempts_path.exists():
        existing = [line for line in attempts_path.read_text(encoding="utf-8").splitlines() if line.strip()]
    attempt = {"attempt": len(existing) + 1, "result": result, "notes": notes, "timestamp_ms": now_ms(), "git": git_status(workspace)}
    append_jsonl(attempts_path, attempt)
    max_iterations = int(profile.get("max_iterations", 0))
    warnings = []
    if len(existing) + 1 >= max_iterations and result != "pass":
        warnings.append("max_iterations reached")
    return envelope(True, "OK", "Attempt recorded", {"attempt": attempt, "attempts_path": str(attempts_path)}, warnings=warnings, started_ms=started)


def generate_report(workspace: Path) -> dict[str, Any]:
    started = now_ms()
    root = state_root(workspace)
    report_path = root / "repair-report.md"
    attempts_path = root / "attempts.jsonl"
    attempts = []
    if attempts_path.exists():
        for line in attempts_path.read_text(encoding="utf-8").splitlines():
            if line.strip():
                attempts.append(json.loads(line))
    lines = ["# AI Debug Repair Report", "", f"- Workspace: `{workspace}`", f"- Attempts: {len(attempts)}", "", "## Attempts", ""]
    if attempts:
        for attempt in attempts:
            lines.append(f"- Attempt {attempt.get('attempt')}: `{attempt.get('result')}` — {attempt.get('notes')}")
    else:
        lines.append("- No attempts recorded.")
    lines.extend(["", "## Review Requirement", "", "This kit never commits automatically. Human review is required before commit, flash, or deployment.", ""])
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text("\n".join(lines), encoding="utf-8")
    return envelope(True, "OK", "Report generated", {"report_path": str(report_path), "attempts": len(attempts)}, started_ms=started)
