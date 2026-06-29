from __future__ import annotations

from pathlib import Path
from typing import Any

from .core import envelope, load_json, now_ms, write_json


INVASIVE_FLAGS = {
    "reset": "allow_reset",
    "halt": "allow_halt",
    "flash": "allow_flash",
    "write_memory": "allow_write_memory",
}


def default_jlink_profile() -> dict[str, Any]:
    return {
        "type": "jlink_backend",
        "backend": "jlink",
        "mode": "readonly",
        "device": "",
        "serial": "",
        "interface": "swd",
        "speed_khz": 4000,
        "architecture": "generic",
        "core": "default",
        "address_unit_bits": 8,
        "endianness": "little",
        "allowed_memory_ranges": [
            {"space": "data", "start": "0x20000000", "length": 65536}
        ],
        "allow_reset": False,
        "allow_halt": False,
        "allow_flash": False,
        "allow_write_memory": False,
        "notes": "J-Link profile with invasive operation interfaces present but disabled by default."
    }


def jlink_profile_template(path: Path) -> dict[str, Any]:
    started = now_ms()
    write_json(path, default_jlink_profile())
    return envelope(True, "OK", "J-Link profile template written", {"profile": str(path)}, started_ms=started)


def jlink_capabilities(profile: dict[str, Any] | None = None) -> dict[str, Any]:
    started = now_ms()
    p = profile or {}
    data = {
        "backend": "jlink",
        "transport": "SEGGER J-Link",
        "non_invasive_default": True,
        "memory_read": True,
        "register_read": True,
        "reset": bool(p.get("allow_reset", False)),
        "halt": bool(p.get("allow_halt", False)),
        "flash": bool(p.get("allow_flash", False)),
        "memory_write": bool(p.get("allow_write_memory", False)),
        "reset_interface_present": True,
        "halt_interface_present": True,
        "flash_interface_present": True,
        "write_memory_interface_present": True,
        "implementation_status": {
            "reset": "policy-gated stub",
            "halt": "policy-gated stub",
            "flash": "policy-gated stub",
            "write_memory": "policy-gated stub"
        }
    }
    if profile:
        data["device"] = profile.get("device")
        data["architecture"] = profile.get("architecture")
        data["address_unit_bits"] = profile.get("address_unit_bits")
    return envelope(True, "OK", "J-Link capabilities", data, started_ms=started)


def validate_jlink_profile(profile_path: Path) -> dict[str, Any]:
    started = now_ms()
    try:
        profile = load_json(profile_path)
    except ValueError as exc:
        return envelope(False, "PROFILE_INVALID", str(exc), started_ms=started)

    errors: list[str] = []
    warnings: list[str] = []
    if profile.get("type") != "jlink_backend":
        errors.append("type must be jlink_backend")
    if profile.get("backend") != "jlink":
        errors.append("backend must be jlink")
    if profile.get("mode") not in ("readonly", "maintenance"):
        errors.append("mode must be readonly or maintenance")
    if profile.get("mode") == "readonly":
        for op, flag in INVASIVE_FLAGS.items():
            if bool(profile.get(flag, False)):
                errors.append(f"{flag}=true is forbidden while mode=readonly")
    if not profile.get("device"):
        warnings.append("device is empty; real J-Link hardware access usually requires an exact SEGGER device name")
    if int(profile.get("address_unit_bits", 0)) not in (8, 16, 32):
        errors.append("address_unit_bits must be one of 8, 16, 32")
    if profile.get("endianness") not in ("little", "big"):
        errors.append("endianness must be little or big")
    if not isinstance(profile.get("allowed_memory_ranges", []), list):
        errors.append("allowed_memory_ranges must be a list")

    if errors:
        return envelope(False, "PROFILE_INVALID", "J-Link profile validation failed", {"errors": errors, "profile": str(profile_path)}, warnings, started_ms=started)
    return envelope(True, "OK", "J-Link profile is valid", {"profile": str(profile_path), "capabilities": jlink_capabilities(profile)["data"]}, warnings, started_ms=started)


def jlink_invasive_operation(profile_path: Path, operation: str, approve: bool = False) -> dict[str, Any]:
    started = now_ms()
    try:
        profile = load_json(profile_path)
    except ValueError as exc:
        return envelope(False, "PROFILE_INVALID", str(exc), started_ms=started)

    validation = validate_jlink_profile(profile_path)
    if not validation["ok"]:
        return validation

    flag = INVASIVE_FLAGS.get(operation)
    if not flag:
        return envelope(False, "INVALID_ARGUMENT", f"unsupported J-Link operation: {operation}", started_ms=started)

    if not bool(profile.get(flag, False)):
        return envelope(False, "POLICY_DENIED", f"J-Link {operation} interface exists but {flag}=false", {
            "operation": operation,
            "flag": flag,
            "implemented": "stub",
            "side_effects": [],
        }, started_ms=started)

    if not approve:
        return envelope(False, "POLICY_DENIED", f"J-Link {operation} requires explicit approval flag", {
            "operation": operation,
            "flag": flag,
            "required_approval": True,
            "implemented": "stub",
            "side_effects": [],
        }, started_ms=started)

    return envelope(False, "CAPABILITY_DISABLED", f"J-Link {operation} is policy-enabled but hardware implementation is not included in this kit package", {
        "operation": operation,
        "flag": flag,
        "implemented": "stub",
        "next_step": "Implement in the main ai-debug-kit JLinkBackend using pylink, not in the repair-loop package.",
        "side_effects": [],
    }, started_ms=started)
