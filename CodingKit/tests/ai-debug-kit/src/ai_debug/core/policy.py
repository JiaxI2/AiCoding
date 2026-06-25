from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum

from ai_debug.core.result import OperationResult


class RiskLevel(str, Enum):
    L0 = "L0"
    L1 = "L1"
    L2 = "L2"
    L3 = "L3"
    L4 = "L4"
    L5 = "L5"


@dataclass(frozen=True)
class Approval:
    granted_levels: set[RiskLevel] = field(default_factory=set)

    @classmethod
    def none(cls) -> "Approval":
        return cls()

    def allows(self, level: RiskLevel) -> bool:
        return level in self.granted_levels


@dataclass(frozen=True)
class Policy:
    read_only: bool = True

    def check(self, level: RiskLevel, approval: Approval) -> OperationResult | None:
        if level in {RiskLevel.L0, RiskLevel.L1}:
            return None
        if self.read_only:
            return OperationResult.fail("POLICY_DENIED", "policy is read-only")
        if not approval.allows(level):
            return OperationResult.fail("POLICY_DENIED", f"approval missing for {level.value}")
        return None
