from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass(frozen=True)
class Envelope:
    schema_version: str = "1.0"
    ok: bool = True
    code: str = "OK"
    message: str = "Operation completed"
    data: dict[str, Any] = field(default_factory=dict)
    warnings: list[str] = field(default_factory=list)
    side_effects: list[str] = field(default_factory=list)
    duration_ms: int = 0
    trace_id: str = ""
    session_id: str = ""


@dataclass(frozen=True)
class OperationResult:
    ok: bool
    code: str
    message: str
    data: dict[str, Any] = field(default_factory=dict)
    warnings: list[str] = field(default_factory=list)
    side_effects: list[str] = field(default_factory=list)

    @classmethod
    def ok_result(
        cls,
        data: dict[str, Any] | None = None,
        *,
        side_effects: list[str] | None = None,
    ) -> "OperationResult":
        return cls(
            ok=True,
            code="OK",
            message="Operation completed",
            data=data or {},
            side_effects=side_effects or [],
        )

    @classmethod
    def fail(cls, code: str, message: str) -> "OperationResult":
        return cls(ok=False, code=code, message=message)
