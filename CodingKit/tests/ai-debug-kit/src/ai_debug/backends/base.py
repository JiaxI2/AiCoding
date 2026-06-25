from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol

from ai_debug.core.address import TargetAddress
from ai_debug.core.capability import Capabilities
from ai_debug.core.result import OperationResult


@dataclass(frozen=True)
class MemoryBlock:
    address: TargetAddress
    data: bytes

    @property
    def octet_length(self) -> int:
        return len(self.data)


class DebugBackend(Protocol):
    def discover(self) -> OperationResult:
        ...

    def validate(self) -> OperationResult:
        ...

    def capabilities(self) -> Capabilities:
        ...

    def read_memory(self, address: TargetAddress, octet_length: int) -> MemoryBlock:
        ...

    def try_read_memory(self, address: TargetAddress, octet_length: int) -> OperationResult:
        ...

    def try_read_register(self, name: str) -> OperationResult:
        ...
