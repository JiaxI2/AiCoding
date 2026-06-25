from __future__ import annotations

from ai_debug.backends.base import DebugBackend
from ai_debug.backends.jlink import JLinkBackend
from ai_debug.backends.simulator import SimulatorBackend


def backend_names() -> list[str]:
    return ["jlink", "simulator"]


def create_backend(name: str) -> DebugBackend:
    normalized = name.lower()
    if normalized == "simulator":
        return SimulatorBackend()
    if normalized == "jlink":
        return JLinkBackend.from_optional_dependency()
    raise ValueError(f"unknown backend: {name}")
