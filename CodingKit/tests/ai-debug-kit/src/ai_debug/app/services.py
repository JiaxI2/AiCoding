from __future__ import annotations

import json
from dataclasses import asdict
from datetime import UTC, datetime
from pathlib import Path

from ai_debug import __version__
from ai_debug.backends.jlink import TargetProfile
from ai_debug.backends.registry import backend_names, create_backend
from ai_debug.backends.simulator import SimulatorBackend
from ai_debug.core.address import TargetAddress
from ai_debug.core.policy import Approval, Policy, RiskLevel
from ai_debug.core.session import DebugSession
from ai_debug.reports.markdown import render_smoke_report


def now_utc() -> str:
    return datetime.now(UTC).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def doctor_summary() -> dict:
    backend = SimulatorBackend()
    return {
        "kit_version": __version__,
        "backend": "simulator",
        "registered_backends": backend_names(),
        "validated": {
            "cli": True,
            "simulator": True,
            "skills": False,
            "jlink_dependency": create_backend("jlink").discover().ok,
        },
        "capabilities": asdict(backend.capabilities()),
    }


def write_json(path: Path, data: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def jlink_profile_json() -> dict:
    profile = TargetProfile.generic_default()
    return {
        "schema_version": "1.0",
        "backend": profile.backend,
        "device": profile.device,
        "interface": profile.interface,
        "speed_khz": profile.speed_khz,
        "address_unit_bits": profile.address_unit_bits,
        "endianness": profile.endianness,
        "architecture": profile.architecture,
        "core": profile.core,
        "allowed_memory_ranges": [
            {"space": item.space, "start": f"0x{item.start:08X}", "length": item.length}
            for item in profile.allowed_memory_ranges
        ],
        "profiles": {
            "generic": {
                "architecture": profile.architecture,
                "core": profile.core,
                "address_unit_bits": profile.address_unit_bits,
            },
            "c2000-c28x": {
                "architecture": "c28x",
                "core_examples": ["cpu1", "cpu2", "cm"],
                "address_unit_bits": 16,
                "transport_note": "C2000/C28x support is target-profile driven; TI DSS/XDS backend is a future transport path.",
            },
        },
    }


def backend_statuses() -> dict:
    statuses = {}
    for name in backend_names():
        backend = create_backend(name)
        discover = backend.discover()
        devices = discover.data.get("devices", []) if isinstance(discover.data, dict) else []
        statuses[name] = {
            "detected": discover.ok and len(devices) > 0,
            "code": discover.code,
            "message": discover.message,
            "capabilities": asdict(backend.capabilities()),
        }
    return statuses


def run_smoke_test(workspace: Path) -> dict:
    workspace = workspace.resolve()
    deployment_dir = workspace / ".ai-debug" / "deployment"
    targets_dir = workspace / ".ai-debug" / "targets"
    session = DebugSession.create(workspace)
    backend = SimulatorBackend(size=256)
    address = TargetAddress(space="data", value=16, address_unit_bits=8)
    policy = Policy(read_only=False)
    approval = Approval(granted_levels={RiskLevel.L3})

    read_before = backend.read_memory(address, 4)
    session.record_action(
        operation="memory.read",
        result_code="OK",
        risk_level="L1",
        approved=True,
        details={"address": address.value, "octet_length": read_before.octet_length},
    )
    session.record_observation(
        kind="memory",
        data={"address": address.value, "raw_octets": read_before.data.hex()},
    )

    write_result = backend.try_write_memory(address, b"\xAA\x55\x12\x34", policy=policy, approval=approval)
    session.record_action(
        operation="memory.write",
        result_code=write_result.code,
        risk_level="L3",
        approved=True,
        details={"address": address.value, "octet_length": 4},
    )

    readback = backend.read_memory(address, 4)
    readback_pass = readback.data == b"\xAA\x55\x12\x34"
    session.record_action(
        operation="verify.readback",
        result_code="OK" if readback_pass else "VALIDATION_FAILED",
        risk_level="L1",
        approved=True,
        details={"readback": readback.data.hex(), "passed": readback_pass},
    )
    session.record_observation(
        kind="readback",
        data={"address": address.value, "raw_octets": readback.data.hex(), "passed": readback_pass},
    )

    active_profile = {
        "schema_version": "1.0",
        "kit_version": __version__,
        "agent": "codex",
        "workspace": str(workspace),
        "installation_status": "ready",
        "validated_at": now_utc(),
        "backend": "simulator",
        "platform": "generic",
        "capabilities": asdict(backend.capabilities()),
        "evidence": {
            "smoke_test": str(deployment_dir / "smoke-test.json"),
            "backends": str(deployment_dir / "backends.json"),
            "jlink_target_profile": str(targets_dir / "jlink-generic.json"),
        },
    }
    smoke_result = {
        "schema_version": "1.0",
        "ok": readback_pass,
        "backend": "simulator",
        "session_id": session.session_id,
        "readback": readback.data.hex(),
    }

    active_profile_path = deployment_dir / "active-profile.json"
    smoke_path = deployment_dir / "smoke-test.json"
    backends_path = deployment_dir / "backends.json"
    jlink_profile_path = targets_dir / "jlink-generic.json"
    write_json(active_profile_path, active_profile)
    write_json(smoke_path, smoke_result)
    write_json(backends_path, {"schema_version": "1.0", "backends": backend_statuses()})
    write_json(jlink_profile_path, jlink_profile_json())

    session_dir = session.export()
    report_path = session_dir / "final-report.md"
    report_path.write_text(
        render_smoke_report(session=session, active_profile=active_profile, readback_pass=readback_pass),
        encoding="utf-8",
    )

    return {
        "active_profile": str(active_profile_path),
        "smoke_test": str(smoke_path),
        "backends": str(backends_path),
        "jlink_target_profile": str(jlink_profile_path),
        "session_bundle": str(session_dir),
        "report": str(report_path),
    }
