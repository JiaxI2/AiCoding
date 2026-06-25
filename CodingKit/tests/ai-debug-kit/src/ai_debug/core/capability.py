from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Capabilities:
    artifact_load: bool = False
    memory_read: bool = False
    memory_write: bool = False
    variable_read: bool = False
    telemetry_capture: bool = False
    fault_snapshot: bool = False
    flash: bool = False
