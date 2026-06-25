from __future__ import annotations

from ai_debug.core.session import DebugSession


def render_smoke_report(*, session: DebugSession, active_profile: dict, readback_pass: bool) -> str:
    status = "PASS" if readback_pass else "FAIL"
    return "\n".join(
        [
            "# AI Debug Kit Smoke Test Report",
            "",
            f"- session: `{session.session_id}`",
            f"- backend: `{active_profile['backend']}`",
            f"- platform: `{active_profile['platform']}`",
            f"- readback: `{status}`",
            "",
            "This report verifies deterministic simulator deployment, memory read, write, readback,",
            "session recording, and active-profile generation. It does not make business conclusions.",
            "claims.",
            "",
        ]
    )
