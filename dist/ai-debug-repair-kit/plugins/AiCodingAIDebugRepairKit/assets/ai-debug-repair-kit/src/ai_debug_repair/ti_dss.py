from __future__ import annotations

import json
import os
import shutil
import subprocess
from pathlib import Path
from typing import Any

from .core import envelope, load_json, now_ms, write_json


FORBIDDEN_TRUE_FLAGS = [
    "allow_reset",
    "allow_halt",
    "allow_run",
    "allow_flash",
    "allow_write_memory",
    "allow_expression_write",
    "allow_register_write",
]


def default_profile() -> dict[str, Any]:
    return {
        "type": "ti_dss_backend",
        "backend": "ti_dss",
        "mode": "readonly",
        "ccs_root": "C:/ti/ccs1281/ccs",
        "dss_launcher": "C:/ti/ccs1281/ccs/ccs_base/scripting/bin/dss.bat",
        "target_config": "",
        "probe": "XDS110",
        "device": "TMS320F28388D",
        "core": "CPU1",
        "architecture": "c28x",
        "address_unit_bits": 16,
        "endianness": "little",
        "connect_timeout_seconds": 20,
        "expression_timeout_seconds": 5,
        "allowed_expressions": [
            "CpuTimer0Regs.TIM.all",
            "CpuTimer1Regs.TIM.all"
        ],
        "allowed_registers": [
            "PC"
        ],
        "forbidden_expression_patterns": [
            "=",
            "GEL_",
            "reset",
            "Reset",
            "run",
            "Run",
            "halt",
            "Halt",
            "flash",
            "Flash",
            "write",
            "Write"
        ],
        "allow_reset": False,
        "allow_halt": False,
        "allow_run": False,
        "allow_flash": False,
        "allow_write_memory": False,
        "allow_expression_write": False,
        "allow_register_write": False
    }


def dss_profile_template(path: Path) -> dict[str, Any]:
    started = now_ms()
    write_json(path, default_profile())
    return envelope(True, "OK", "TI DSS read-only profile template written", {"profile": str(path)}, started_ms=started)


def dss_capabilities(profile: dict[str, Any] | None = None) -> dict[str, Any]:
    started = now_ms()
    data = {
        "backend": "ti_dss",
        "transport": "TI XDS via CCS DSS DebugServer",
        "default_mode": "readonly",
        "non_invasive_default": True,
        "expression_read": True,
        "register_read": True,
        "memory_read": False,
        "telemetry_capture": "planned",
        "reset": False,
        "halt": False,
        "run": False,
        "flash": False,
        "memory_write": False,
        "expression_write": False,
        "register_write": False,
        "requires_target_config": True,
        "requires_ccs_dss": True,
        "profile_enforced_flags": FORBIDDEN_TRUE_FLAGS,
    }
    if profile:
        data["device"] = profile.get("device")
        data["core"] = profile.get("core")
        data["architecture"] = profile.get("architecture")
        data["address_unit_bits"] = profile.get("address_unit_bits")
    return envelope(True, "OK", "TI DSS capabilities", data, started_ms=started)


def validate_dss_profile(profile_path: Path) -> dict[str, Any]:
    started = now_ms()
    try:
        profile = load_json(profile_path)
    except ValueError as exc:
        return envelope(False, "PROFILE_INVALID", str(exc), started_ms=started)

    errors: list[str] = []
    warnings: list[str] = []
    if profile.get("type") != "ti_dss_backend":
        errors.append("type must be ti_dss_backend")
    if profile.get("backend") != "ti_dss":
        errors.append("backend must be ti_dss")
    if profile.get("mode") != "readonly":
        errors.append("mode must be readonly")
    for flag in FORBIDDEN_TRUE_FLAGS:
        if bool(profile.get(flag, False)):
            errors.append(f"{flag}=true is forbidden for the non-invasive default backend")
    if not profile.get("dss_launcher"):
        errors.append("dss_launcher is required")
    if not profile.get("target_config"):
        warnings.append("target_config is empty; profile can be validated but cannot execute against hardware until filled")
    if int(profile.get("address_unit_bits", 0)) not in (8, 16, 32):
        errors.append("address_unit_bits must be one of 8, 16, 32")
    if profile.get("endianness") not in ("little", "big"):
        errors.append("endianness must be little or big")
    if not isinstance(profile.get("allowed_expressions", []), list):
        errors.append("allowed_expressions must be a list")
    if not isinstance(profile.get("allowed_registers", []), list):
        errors.append("allowed_registers must be a list")

    if errors:
        return envelope(False, "PROFILE_INVALID", "TI DSS profile validation failed", {"errors": errors, "profile": str(profile_path)}, warnings, started_ms=started)

    launcher = str(profile.get("dss_launcher", ""))
    data = {
        "profile": str(profile_path),
        "mode": "readonly",
        "dss_launcher": launcher,
        "dss_launcher_exists": Path(launcher).exists(),
        "dss_on_path": shutil.which("dss") or shutil.which("dss.bat") or shutil.which("dss.sh"),
        "capabilities": dss_capabilities(profile)["data"],
    }
    return envelope(True, "OK", "TI DSS profile is valid", data, warnings, started_ms=started)


def dss_doctor(profile_path: Path | None = None) -> dict[str, Any]:
    started = now_ms()
    data: dict[str, Any] = {
        "java": shutil.which("java"),
        "dss_on_path": shutil.which("dss") or shutil.which("dss.bat") or shutil.which("dss.sh"),
        "env_CCS_ROOT": os.environ.get("CCS_ROOT"),
        "env_TI_CCS_ROOT": os.environ.get("TI_CCS_ROOT"),
    }
    warnings: list[str] = []
    if profile_path:
        try:
            profile = load_json(profile_path)
            data["profile"] = str(profile_path)
            data["dss_launcher"] = profile.get("dss_launcher")
            data["dss_launcher_exists"] = Path(str(profile.get("dss_launcher", ""))).exists()
            data["target_config"] = profile.get("target_config")
            data["target_config_exists"] = Path(str(profile.get("target_config", ""))).exists() if profile.get("target_config") else False
            data["capabilities"] = dss_capabilities(profile)["data"]
        except ValueError as exc:
            return envelope(False, "PROFILE_INVALID", str(exc), data, started_ms=started)
    if not data.get("java"):
        warnings.append("java not found on PATH; CCS DSS usually requires Java from CCS or system Java")
    if not data.get("dss_on_path") and not data.get("dss_launcher_exists"):
        warnings.append("DSS launcher not found. Set dss_launcher in profile to ccs_base/scripting/bin/dss.bat or dss.sh")
    return envelope(True, "OK", "TI DSS doctor completed", data, warnings, started_ms=started)


def _is_expression_allowed(profile: dict[str, Any], target: str, kind: str) -> tuple[bool, str]:
    if kind == "register":
        allowed = profile.get("allowed_registers", [])
        if target not in allowed:
            return False, f"register is not listed in allowed_registers: {target}"
        return True, "OK"

    allowed = profile.get("allowed_expressions", [])
    if target not in allowed:
        return False, f"expression is not listed in allowed_expressions: {target}"
    for pat in profile.get("forbidden_expression_patterns", []):
        if pat and pat in target:
            return False, f"expression contains forbidden pattern: {pat}"
    return True, "OK"


def generate_dss_script(profile_path: Path, expression: str | None, register: str | None, output_path: Path) -> dict[str, Any]:
    started = now_ms()
    try:
        profile = load_json(profile_path)
    except ValueError as exc:
        return envelope(False, "PROFILE_INVALID", str(exc), started_ms=started)

    validation = validate_dss_profile(profile_path)
    if not validation["ok"]:
        return validation

    if expression and register:
        return envelope(False, "INVALID_ARGUMENT", "Use either expression or register, not both", started_ms=started)
    if not expression and not register:
        return envelope(False, "INVALID_ARGUMENT", "expression or register is required", started_ms=started)

    target = expression or register or ""
    kind = "register" if register else "expression"
    allowed, reason = _is_expression_allowed(profile, target, kind)
    if not allowed:
        return envelope(False, "POLICY_DENIED", reason, {"target": target, "kind": kind}, started_ms=started)

    target_config = str(profile.get("target_config", "")).replace("\\", "/")
    core = str(profile.get("core", "CPU1"))
    escaped_target = target.replace("\\", "\\\\").replace('"', '\\"')

    script = "\n".join([
        "// Generated by ai-debug-repair-kit.",
        "// Non-invasive default TI XDS / CCS DSS read-only script.",
        "// Forbidden by policy: reset, halt, run, flash, memory write, expression write, register write.",
        "importPackage(Packages.com.ti.debug.engine.scripting);",
        "importPackage(Packages.com.ti.ccstudio.scripting.environment);",
        "var scripting = ScriptingEnvironment.instance();",
        f"scripting.traceBegin(\"{output_path.as_posix()}.trace.xml\", \"DefaultStylesheet.xsl\");",
        "var server = scripting.getServer(\"DebugServer.1\");",
        f"server.setConfig(\"{target_config}\");",
        "var debugSession = null;",
        "try {",
        f"  debugSession = server.openSession(\"*\", \"{core}\");",
        "  debugSession.target.connect();",
        "  var value = null;",
        f"  var op = \"{kind}\";",
        "  if (op == \"register\") {",
        f"    value = debugSession.memory.readRegister(\"{escaped_target}\");",
        "  } else {",
        f"    value = debugSession.expression.evaluate(\"{escaped_target}\");",
        "  }",
        "  print(JSON.stringify({schema_version:\"1.0\", ok:true, code:\"OK\", backend:\"ti_dss\", mode:\"readonly\", operation:op+\".read\", target:\"" + escaped_target + "\", value:String(value), side_effects:[]}));",
        "} catch (e) {",
        "  print(JSON.stringify({schema_version:\"1.0\", ok:false, code:\"DSS_ERROR\", backend:\"ti_dss\", mode:\"readonly\", message:String(e), side_effects:[]}));",
        "} finally {",
        "  try { if (debugSession != null) { debugSession.terminate(); } } catch (ignore) {}",
        "  try { server.stop(); } catch (ignore2) {}",
        "  scripting.traceEnd();",
        "}",
        ""
    ])

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(script, encoding="utf-8")
    return envelope(True, "OK", "TI DSS read-only script generated", {
        "script": str(output_path),
        "profile": str(profile_path),
        "kind": kind,
        "target": target,
        "mode": "readonly",
        "execute": False,
    }, started_ms=started)


def execute_dss_script(profile_path: Path, script_path: Path, timeout_seconds: int | None = None) -> dict[str, Any]:
    started = now_ms()
    try:
        profile = load_json(profile_path)
    except ValueError as exc:
        return envelope(False, "PROFILE_INVALID", str(exc), started_ms=started)
    validation = validate_dss_profile(profile_path)
    if not validation["ok"]:
        return validation
    launcher = str(profile.get("dss_launcher", ""))
    if not Path(launcher).exists() and not shutil.which(launcher):
        return envelope(False, "DEPENDENCY_MISSING", f"DSS launcher not found: {launcher}", {"dss_launcher": launcher}, started_ms=started)
    if not script_path.exists():
        return envelope(False, "INVALID_ARGUMENT", f"script not found: {script_path}", started_ms=started)
    timeout = int(timeout_seconds or profile.get("expression_timeout_seconds", 5))
    try:
        completed = subprocess.run([launcher, str(script_path)], text=True, capture_output=True, timeout=timeout, shell=False)
    except subprocess.TimeoutExpired:
        return envelope(False, "TIMEOUT", f"DSS script timed out after {timeout}s", {"script": str(script_path)}, started_ms=started)
    except FileNotFoundError:
        return envelope(False, "DEPENDENCY_MISSING", f"DSS launcher not found: {launcher}", started_ms=started)
    data = {"script": str(script_path), "returncode": completed.returncode, "stdout": completed.stdout, "stderr": completed.stderr, "side_effects": []}
    ok = completed.returncode == 0
    return envelope(ok, "OK" if ok else "DSS_COMMAND_FAILED", "DSS script executed" if ok else "DSS script failed", data, started_ms=started)
